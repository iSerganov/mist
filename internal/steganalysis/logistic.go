package steganalysis

import (
	"encoding/json"
	"fmt"
	"math"
	"slices"
)

const (
	trainSteps = 300
	learnRate  = 0.5
	l2         = 1e-2
)

// Logistic is an L2-regularised logistic regression on standardised
// features.
type Logistic struct {
	w, mean, scale []float64
	b              float64
}

// TrainLogistic fits a Logistic to x labelled by y with full-batch gradient
// descent. It is deterministic, and x must not be empty.
func TrainLogistic(x [][]float64, y []bool) *Logistic {
	return TrainLogisticPenalty(x, y, l2)
}

// TrainLogisticPenalty is TrainLogistic with an explicit L2 penalty.
// Standardisation is fit on x alone.
func TrainLogisticPenalty(x [][]float64, y []bool, penalty float64) *Logistic {
	dim := len(x[0])
	m := &Logistic{w: make([]float64, dim), mean: make([]float64, dim), scale: make([]float64, dim)}
	m.standardise(x)
	z := make([][]float64, len(x))
	for i, xi := range x {
		z[i] = m.normalise(xi)
	}
	n := float64(len(x))
	grad := make([]float64, dim)
	for range trainSteps {
		clear(grad)
		var gb float64
		for i, zi := range z {
			err := sigmoid(m.b+dot(m.w, zi)) - label(y[i])
			for j, v := range zi {
				grad[j] += err * v
			}
			gb += err
		}
		for j := range m.w {
			m.w[j] -= learnRate * (grad[j]/n + penalty*m.w[j])
		}
		m.b -= learnRate * gb / n
	}
	return m
}

// Score returns the probability that x belongs to the positive class.
func (m *Logistic) Score(x []float64) float64 {
	return sigmoid(m.b + dot(m.w, m.normalise(x)))
}

type logisticJSON struct {
	W     []float64 `json:"w"`
	Mean  []float64 `json:"mean"`
	Scale []float64 `json:"scale"`
	B     float64   `json:"b"`
}

// MarshalJSON writes the fitted weights and standardisation, so a model
// trained once can score files in another process.
func (m *Logistic) MarshalJSON() ([]byte, error) {
	return json.Marshal(logisticJSON{W: m.w, Mean: m.mean, Scale: m.scale, B: m.b})
}

// UnmarshalJSON reads a model MarshalJSON wrote.
func (m *Logistic) UnmarshalJSON(b []byte) error {
	var v logisticJSON
	if err := json.Unmarshal(b, &v); err != nil {
		return err
	}
	if len(v.Mean) != len(v.W) || len(v.Scale) != len(v.W) {
		return fmt.Errorf("logistic: %d weights, %d means, %d scales", len(v.W), len(v.Mean), len(v.Scale))
	}
	m.w, m.mean, m.scale, m.b = v.W, v.Mean, v.Scale, v.B
	return nil
}

// Dim is the length of the feature vector the model scores.
func (m *Logistic) Dim() int { return len(m.w) }

// CrossValidate returns an out-of-fold score for every sample. Samples are
// split into folds by group, and each fold is scored by a model trained on
// the others, so no group is scored by a model that saw it. With fewer than
// two distinct groups the split falls back to individual samples.
func CrossValidate(x [][]float64, y []bool, groups []int, folds int) []float64 {
	fold, k := assignFolds(groups, folds)
	out := make([]float64, len(x))
	for f := range k {
		var tx [][]float64
		var ty []bool
		for i := range x {
			if fold[i] != f {
				tx = append(tx, x[i])
				ty = append(ty, y[i])
			}
		}
		var m *Logistic
		if len(tx) > 0 {
			m = TrainLogistic(tx, ty)
		}
		for i := range x {
			if fold[i] != f {
				continue
			}
			out[i] = 0.5
			if m != nil {
				out[i] = m.Score(x[i])
			}
		}
	}
	return out
}

func assignFolds(groups []int, folds int) ([]int, int) {
	ids := slices.Compact(slices.Sorted(slices.Values(groups)))
	out := make([]int, len(groups))
	if len(ids) < 2 {
		for i := range out {
			out[i] = i % folds
		}
		return out, folds
	}
	folds = min(folds, len(ids))
	for i, g := range groups {
		rank, _ := slices.BinarySearch(ids, g)
		out[i] = rank % folds
	}
	return out, folds
}

// nestedPenalties is the inner-fold grid. It is fixed so a report can name
// the search that produced its classifier scores.
var nestedPenalties = []float64{1e-3, 1e-2, 1e-1}

// NestedPenaltyGrid is the L2 penalties NestedCrossValidate tries on its
// inner folds, in search order.
func NestedPenaltyGrid() []float64 {
	return append([]float64(nil), nestedPenalties...)
}

// NestedCrossValidate chooses the L2 penalty on groups that are not in the
// outer test fold, optionally calibrates on a further held-out slice of
// those groups, and only then scores the outer fold. Fewer than four
// distinct groups cannot support that split, so it falls back to
// CrossValidate with the protocol penalty.
func NestedCrossValidate(x [][]float64, y []bool, groups []int, outer int) []float64 {
	ids := slices.Compact(slices.Sorted(slices.Values(groups)))
	if len(ids) < 4 {
		return CrossValidate(x, y, groups, outer)
	}
	fold, k := assignFolds(groups, outer)
	out := make([]float64, len(x))
	for f := range k {
		var train, test []int
		for i := range x {
			if fold[i] == f {
				test = append(test, i)
				continue
			}
			train = append(train, i)
		}
		penalty := selectPenalty(x, y, groups, train)
		fit, cal := splitCalibration(groups, train)
		if len(fit) == 0 || !bothLabels(y, fit) {
			for _, i := range test {
				out[i] = 0.5
			}
			continue
		}
		m := TrainLogisticPenalty(rows(x, fit), labels(y, fit), penalty)
		a, b := 0.0, 1.0
		if len(cal) >= 4 && bothLabels(y, cal) {
			a, b = fitPlatt(m, x, y, cal)
		}
		for _, i := range test {
			out[i] = platt(a, b, m.Score(x[i]))
		}
	}
	return out
}

func selectPenalty(x [][]float64, y []bool, groups []int, train []int) float64 {
	subX, subY, subG := rows(x, train), labels(y, train), rowGroups(groups, train)
	ids := slices.Compact(slices.Sorted(slices.Values(subG)))
	if len(ids) < 2 {
		return l2
	}
	best, bestLoss := l2, math.Inf(1)
	for _, penalty := range nestedPenalties {
		scores := crossValidatePenalty(subX, subY, subG, min(3, len(ids)), penalty)
		loss := logLoss(subY, scores)
		if loss < bestLoss {
			best, bestLoss = penalty, loss
		}
	}
	return best
}

func crossValidatePenalty(x [][]float64, y []bool, groups []int, folds int, penalty float64) []float64 {
	fold, k := assignFolds(groups, folds)
	out := make([]float64, len(x))
	for f := range k {
		var tx [][]float64
		var ty []bool
		for i := range x {
			if fold[i] != f {
				tx = append(tx, x[i])
				ty = append(ty, y[i])
			}
		}
		var m *Logistic
		if len(tx) > 0 && hasBoth(ty) {
			m = TrainLogisticPenalty(tx, ty, penalty)
		}
		for i := range x {
			if fold[i] != f {
				continue
			}
			out[i] = 0.5
			if m != nil {
				out[i] = m.Score(x[i])
			}
		}
	}
	return out
}

func hasBoth(y []bool) bool {
	var pos, neg bool
	for _, v := range y {
		pos = pos || v
		neg = neg || !v
	}
	return pos && neg
}

func splitCalibration(groups []int, train []int) (fit, cal []int) {
	sub := rowGroups(groups, train)
	ids := slices.Compact(slices.Sorted(slices.Values(sub)))
	if len(ids) < 3 {
		return train, nil
	}
	held := ids[len(ids)-1]
	for i, row := range train {
		if sub[i] == held {
			cal = append(cal, row)
			continue
		}
		fit = append(fit, row)
	}
	if len(fit) == 0 {
		return train, nil
	}
	return fit, cal
}

func fitPlatt(m *Logistic, x [][]float64, y []bool, idx []int) (a, b float64) {
	b = 1
	const steps = 40
	n := float64(len(idx))
	for range steps {
		var ga, gb float64
		for _, i := range idx {
			logit := logitScore(m.Score(x[i]))
			err := sigmoid(a+b*logit) - label(y[i])
			ga += err
			gb += err * logit
		}
		a -= 0.1 * ga / n
		b -= 0.1 * gb / n
		if b < 1e-3 {
			b = 1e-3
		}
	}
	return a, b
}

func platt(a, b, p float64) float64 { return sigmoid(a + b*logitScore(p)) }

func logitScore(p float64) float64 {
	if p < 1e-6 {
		p = 1e-6
	}
	if p > 1-1e-6 {
		p = 1 - 1e-6
	}
	return math.Log(p / (1 - p))
}

func logLoss(y []bool, scores []float64) float64 {
	var s float64
	for i, score := range scores {
		p := score
		if p < 1e-6 {
			p = 1e-6
		}
		if p > 1-1e-6 {
			p = 1 - 1e-6
		}
		t := label(y[i])
		s += -(t*math.Log(p) + (1-t)*math.Log(1-p))
	}
	return s / float64(len(y))
}

func bothLabels(y []bool, idx []int) bool {
	var pos, neg bool
	for _, i := range idx {
		pos = pos || y[i]
		neg = neg || !y[i]
	}
	return pos && neg
}

func rows(x [][]float64, idx []int) [][]float64 {
	out := make([][]float64, len(idx))
	for i, row := range idx {
		out[i] = x[row]
	}
	return out
}

func labels(y []bool, idx []int) []bool {
	out := make([]bool, len(idx))
	for i, row := range idx {
		out[i] = y[row]
	}
	return out
}

func rowGroups(groups []int, idx []int) []int {
	out := make([]int, len(idx))
	for i, row := range idx {
		out[i] = groups[row]
	}
	return out
}

func (m *Logistic) standardise(x [][]float64) {
	n := float64(len(x))
	for _, xi := range x {
		for j, v := range xi {
			m.mean[j] += v / n
		}
	}
	for _, xi := range x {
		for j, v := range xi {
			m.scale[j] += (v - m.mean[j]) * (v - m.mean[j]) / n
		}
	}
	for j, v := range m.scale {
		m.scale[j] = math.Sqrt(v)
		if m.scale[j] == 0 {
			m.scale[j] = 1
		}
	}
}

func (m *Logistic) normalise(x []float64) []float64 {
	z := make([]float64, len(x))
	for j, v := range x {
		z[j] = (v - m.mean[j]) / m.scale[j]
	}
	return z
}

func sigmoid(t float64) float64 { return 1 / (1 + math.Exp(-t)) }

func label(b bool) float64 {
	if b {
		return 1
	}
	return 0
}

func dot(a, b []float64) float64 {
	var s float64
	for i := range a {
		s += a[i] * b[i]
	}
	return s
}

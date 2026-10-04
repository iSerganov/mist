package steganalysis

import (
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
			m.w[j] -= learnRate * (grad[j]/n + l2*m.w[j])
		}
		m.b -= learnRate * gb / n
	}
	return m
}

// Score returns the probability that x belongs to the positive class.
func (m *Logistic) Score(x []float64) float64 {
	return sigmoid(m.b + dot(m.w, m.normalise(x)))
}

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

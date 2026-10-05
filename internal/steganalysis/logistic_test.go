package steganalysis

import (
	"testing"

	"github.com/stretchr/testify/suite"
)

type ClassifierSuite struct {
	suite.Suite
}

func TestClassifierSuite(t *testing.T) {
	suite.Run(t, &ClassifierSuite{})
}

func (s *ClassifierSuite) TestFeaturesStartWithDetectorScores() {
	v := peaked(newRand(1))
	got := Features(v)
	ds := Detectors()
	s.Require().Len(got, len(ds)+(2*spamT+1)*(2*spamT+2))
	for i, d := range ds {
		s.Equal(d.Score(v), got[i], d.Name)
	}
}

func (s *ClassifierSuite) TestTransitions() {
	const side = 2*spamT + 1
	at := func(a, b int) int { return (a+spamT)*side + b + spamT }
	tests := []struct {
		title string
		v     []int32
		want  map[int]float64
	}{
		{"constant stream stays at zero difference", []int32{4, 4, 4, 4, 4}, map[int]float64{at(0, 0): 1}},
		{"steady ramp repeats its step", []int32{0, 2, 4, 6, 8}, map[int]float64{at(2, 2): 1}},
		{"large jumps are truncated", []int32{0, 100, 200, 300}, map[int]float64{at(spamT, spamT): 1}},
		{"alternating steps split evenly", []int32{0, 1, 0, 1, 0, 1}, map[int]float64{at(1, -1): 1, at(-1, 1): 1}},
		{"too short for a transition", []int32{1, 2}, map[int]float64{}},
	}
	for _, tc := range tests {
		s.Run(tc.title, func() {
			got := transitions(diff(tc.v))
			for i, p := range got {
				s.InDelta(tc.want[i], p, 1e-12, "cell %d", i)
			}
		})
	}
}

func (s *ClassifierSuite) TestTrainLogisticSeparatesSeparableData() {
	x := [][]float64{{-3, 1}, {-2, 0}, {-1, 1}, {1, 0}, {2, 1}, {3, 0}}
	y := []bool{false, false, false, true, true, true}
	m := TrainLogistic(x, y)
	for i, xi := range x {
		if y[i] {
			s.Greater(m.Score(xi), 0.5)
		} else {
			s.Less(m.Score(xi), 0.5)
		}
	}
}

func (s *ClassifierSuite) TestAssignFolds() {
	tests := []struct {
		title     string
		groups    []int
		folds     int
		wantFolds int
	}{
		{"more groups than folds", []int{7, 7, 3, 3, 9, 9, 1, 1}, 3, 3},
		{"fewer groups than folds", []int{5, 5, 2, 2}, 5, 2},
		{"single group falls back to samples", []int{4, 4, 4, 4, 4}, 2, 2},
	}
	for _, tc := range tests {
		s.Run(tc.title, func() {
			fold, k := assignFolds(tc.groups, tc.folds)
			s.Equal(tc.wantFolds, k)
			seen := map[int]bool{}
			for _, f := range fold {
				s.GreaterOrEqual(f, 0)
				s.Less(f, k)
				seen[f] = true
			}
			s.Len(seen, k)
		})
	}
}

func (s *ClassifierSuite) TestAssignFoldsKeepsEachGroupInOneFold() {
	groups := []int{1, 2, 3, 1, 2, 3, 4, 4}
	fold, _ := assignFolds(groups, 2)
	home := map[int]int{}
	for i, g := range groups {
		if f, ok := home[g]; ok {
			s.Equal(f, fold[i], "group %d", g)
		}
		home[g] = fold[i]
	}
}

func (s *ClassifierSuite) TestCrossValidate() {
	tests := []struct {
		title   string
		embed   embedFunc
		rate    float64
		wantMin float64
		wantMax float64
	}{
		{"learns plus-minus-one embedding", matchLSB, 0.5, 0.9, 1},
		{"stays near chance without embedding", matchLSB, 0, 0.3, 0.7},
	}
	for _, tc := range tests {
		s.Run(tc.title, func() {
			x, y, g := classifierSet(tc.embed, tc.rate)
			oof := CrossValidate(x, y, g, 4)
			var pos, neg []float64
			for i, v := range oof {
				if y[i] {
					pos = append(pos, v)
				} else {
					neg = append(neg, v)
				}
			}
			auc := AUC(pos, neg)
			s.GreaterOrEqual(auc, tc.wantMin)
			s.LessOrEqual(auc, tc.wantMax)
		})
	}
}

func (s *ClassifierSuite) TestCrossValidateIsDeterministic() {
	x, y, g := classifierSet(matchLSB, 0.5)
	s.Equal(CrossValidate(x, y, g, 4), CrossValidate(x, y, g, 4))
}

func (s *ClassifierSuite) TestCrossValidateWithOneSample() {
	s.Equal([]float64{0.5}, CrossValidate([][]float64{{1}}, []bool{true}, []int{0}, 2))
}

func (s *ClassifierSuite) TestNestedFallsBackWithoutEnoughGroups() {
	x := [][]float64{{0}, {1}, {0}, {1}}
	y := []bool{false, true, false, true}
	g := []int{0, 0, 1, 1}
	s.Equal(CrossValidate(x, y, g, 2), NestedCrossValidate(x, y, g, 2))
}

func (s *ClassifierSuite) TestNestedCrossValidateSeparatesHeldOutGroups() {
	var x [][]float64
	var y []bool
	var g []int
	for i := range 8 {
		x = append(x, []float64{float64(i), -1}, []float64{float64(i), 1})
		y = append(y, false, true)
		g = append(g, i, i)
	}
	first := NestedCrossValidate(x, y, g, 4)
	s.Equal(first, NestedCrossValidate(x, y, g, 4))
	var pos, neg []float64
	for i, v := range first {
		if y[i] {
			pos = append(pos, v)
		} else {
			neg = append(neg, v)
		}
	}
	s.Greater(AUC(pos, neg), 0.8)
}

func classifierSet(embed embedFunc, rate float64) ([][]float64, []bool, []int) {
	const chunk = 10000
	var x [][]float64
	var y []bool
	var g []int
	for carrier := range 8 {
		c := peaked(newRand(uint64(carrier + 10)))
		e := embed(c, rate, newRand(uint64(carrier+100)))
		for off := 0; off+chunk <= len(c); off += chunk {
			x = append(x, Features(c[off:off+chunk]), Features(e[off:off+chunk]))
			y = append(y, false, true)
			g = append(g, carrier, carrier)
		}
	}
	return x, y, g
}

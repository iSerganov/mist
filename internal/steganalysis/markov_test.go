package steganalysis

import (
	"math"
	"math/rand/v2"
	"testing"

	"github.com/stretchr/testify/suite"
)

type MarkovSuite struct {
	suite.Suite
}

func TestMarkovSuite(t *testing.T) {
	suite.Run(t, &MarkovSuite{})
}

const side = 2*spamT + 1

func cell(a, b int) int { return (a+spamT)*side + b + spamT }

func (s *MarkovSuite) TestSecondDifferences() {
	tests := []struct {
		title string
		v     []int32
		want  map[int]float64
	}{
		{"steady ramp has no curvature", []int32{0, 3, 6, 9, 12}, map[int]float64{cell(0, 0): 1}},
		{"parabola keeps a constant curvature", []int32{0, 1, 4, 9, 16, 25}, map[int]float64{cell(2, 2): 1}},
		{"alternating values swing between ±2", []int32{0, 1, 0, 1, 0, 1}, map[int]float64{cell(-2, 2): 1, cell(2, -2): 1}},
		{"large curvature is truncated", []int32{0, 0, 100, 300, 600}, map[int]float64{cell(spamT, spamT): 1}},
		{"too short for a transition", []int32{1, 2, 3}, map[int]float64{}},
	}
	for _, tc := range tests {
		s.Run(tc.title, func() {
			got := Markov(tc.v)
			s.Require().Len(got, side*side)
			for i, p := range got {
				s.InDelta(tc.want[i], p, 1e-12, "cell %d", i)
			}
		})
	}
}

func (s *MarkovSuite) TestRowsAreDistributions() {
	tests := []struct {
		title   string
		carrier carrierFunc
	}{
		{"peaked carrier", peaked},
		{"smooth carrier", smooth},
		{"zero-mean noise", noise},
	}
	for _, tc := range tests {
		s.Run(tc.title, func() {
			got := Markov(tc.carrier(newRand(7)))
			s.Require().Len(got, side*side)
			for a := range side {
				var sum float64
				for b := range side {
					sum += got[a*side+b]
				}
				if sum != 0 {
					s.InDelta(1, sum, 1e-9, "row %d", a-spamT)
				}
			}
		})
	}
}

// Negating zero-mean, symmetric noise negates every second difference,
// so the matrix must look the same turned through 180°.
func (s *MarkovSuite) TestZeroMeanNoiseIsCentrosymmetric() {
	got := Markov(noise(newRand(3)))
	for a := -spamT; a <= spamT; a++ {
		for b := -spamT; b <= spamT; b++ {
			s.InDelta(got[cell(a, b)], got[cell(-a, -b)], 0.03, "(%d,%d)", a, b)
		}
	}
}

func (s *MarkovSuite) TestSeparatesReplacementOnASmoothSignal() {
	const chunks = 20
	var x [][]float64
	var y []bool
	var groups []int
	rng := newRand(11)
	for i := range chunks {
		clean := smoothQuiet(rng)
		x = append(x, Markov(clean), Markov(replaceLSB(clean, 0.5, rng)))
		y = append(y, false, true)
		groups = append(groups, i, i)
	}
	var pos, neg []float64
	for i, v := range CrossValidate(x, y, groups, 5) {
		if y[i] {
			pos = append(pos, v)
		} else {
			neg = append(neg, v)
		}
	}
	s.Greater(AUC(pos, neg), 0.9)
}

func noise(rng *rand.Rand) []int32 {
	v := make([]int32, fixtureLen)
	for i := range v {
		v[i] = int32(rng.IntN(5)) - 2
	}
	return v
}

// smoothQuiet is smooth with sub-step noise, so its curvature sits inside
// ±3 where the features can see it.
func smoothQuiet(rng *rand.Rand) []int32 {
	v := make([]int32, fixtureLen/4)
	phase := rng.Float64() * 100
	for i := range v {
		t := float64(i) + phase
		v[i] = int32(math.Round(800*math.Sin(t/37) + 300*math.Sin(t/11) + rng.NormFloat64()*0.4))
	}
	return v
}

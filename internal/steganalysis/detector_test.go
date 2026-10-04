package steganalysis

import (
	"math"
	"math/rand/v2"
	"testing"

	"github.com/stretchr/testify/suite"
)

type DetectorSuite struct {
	suite.Suite
}

func TestDetectorSuite(t *testing.T) {
	suite.Run(t, &DetectorSuite{})
}

func (s *DetectorSuite) TestEstimatorsTrackReplacementRate() {
	tests := []struct {
		title   string
		score   func([]int32) float64
		carrier carrierFunc
		rate    float64
	}{
		{"spa on a clean smooth carrier", SPA, smooth, 0},
		{"spa on a smooth carrier at 10%", SPA, smooth, 0.1},
		{"spa on a smooth carrier at 25%", SPA, smooth, 0.25},
		{"spa on a smooth carrier at 50%", SPA, smooth, 0.5},
		{"spa on a clean peaked carrier", SPA, peaked, 0},
		{"spa on a peaked carrier at 25%", SPA, peaked, 0.25},
		{"spa on a peaked carrier at 50%", SPA, peaked, 0.5},
		{"rs on a clean smooth carrier", RS, smooth, 0},
		{"rs on a smooth carrier at 10%", RS, smooth, 0.1},
		{"rs on a smooth carrier at 25%", RS, smooth, 0.25},
		{"rs on a smooth carrier at 50%", RS, smooth, 0.5},
		{"rs on a clean peaked carrier", RS, peaked, 0},
		{"rs on a peaked carrier at 50%", RS, peaked, 0.5},
	}
	for _, tc := range tests {
		s.Run(tc.title, func() {
			v := replaceLSB(tc.carrier(newRand(1)), tc.rate, newRand(2))
			s.InDelta(tc.rate, tc.score(v), 0.1)
		})
	}
}

func (s *DetectorSuite) TestDetectorsScoreEmbeddingAboveClean() {
	tests := []struct {
		title   string
		score   func([]int32) float64
		carrier carrierFunc
		embed   embedFunc
		rate    float64
	}{
		{"chi-square sees full replacement on a peaked carrier", ChiSquare, peaked, replaceLSB, 1},
		{"spa sees partial replacement on a smooth carrier", SPA, smooth, replaceLSB, 0.25},
		{"rs sees partial replacement on a smooth carrier", RS, smooth, replaceLSB, 0.25},
		{"hcf sees matching on a peaked carrier", HCF, peaked, matchLSB, 0.5},
	}
	for _, tc := range tests {
		s.Run(tc.title, func() {
			c := tc.carrier(newRand(1))
			s.Greater(tc.score(tc.embed(c, tc.rate, newRand(2))), tc.score(c)+0.01)
		})
	}
}

func (s *DetectorSuite) TestDetectorsReturnZeroWithoutEvidence() {
	inputs := []struct {
		title string
		v     []int32
	}{
		{"nil", nil},
		{"single value", []int32{7}},
		{"constant run", []int32{5, 5, 5, 5, 5, 5, 5, 5, 5, 5, 5, 5, 5, 5, 5, 5}},
	}
	for _, d := range Detectors() {
		for _, in := range inputs {
			s.Run(d.Name+" on "+in.title, func() {
				s.Zero(d.Score(in.v))
			})
		}
	}
}

func (s *DetectorSuite) TestScoresStayInUnitRange() {
	for _, d := range Detectors() {
		for _, rate := range []float64{0, 0.5, 1} {
			s.Run(d.Name, func() {
				got := d.Score(replaceLSB(smooth(newRand(1)), rate, newRand(2)))
				s.GreaterOrEqual(got, 0.0)
				s.LessOrEqual(got, 1.0)
			})
		}
	}
}

const fixtureLen = 40000

type carrierFunc func(rng *rand.Rand) []int32

type embedFunc func(v []int32, rate float64, rng *rand.Rand) []int32

func peaked(rng *rand.Rand) []int32 {
	v := make([]int32, fixtureLen)
	for i := range v {
		v[i] = int32(math.Round(rng.ExpFloat64() * 3 * float64(1-2*rng.IntN(2))))
	}
	return v
}

func smooth(rng *rand.Rand) []int32 {
	v := make([]int32, fixtureLen)
	for i := range v {
		t := float64(i)
		v[i] = int32(math.Round(800*math.Sin(t/37) + 300*math.Sin(t/11) + rng.NormFloat64()*4))
	}
	return v
}

func replaceLSB(v []int32, rate float64, rng *rand.Rand) []int32 {
	return perturb(v, rate, rng, func(x int32, bit int32) int32 { return x&^1 | bit })
}

func matchLSB(v []int32, rate float64, rng *rand.Rand) []int32 {
	return perturb(v, rate, rng, func(x int32, bit int32) int32 {
		if x&1 == bit {
			return x
		}
		return x + int32(1-2*rng.IntN(2))
	})
}

func perturb(v []int32, rate float64, rng *rand.Rand, set func(x, bit int32) int32) []int32 {
	out := make([]int32, len(v))
	for i, x := range v {
		out[i] = x
		if rng.Float64() < rate {
			out[i] = set(x, int32(rng.IntN(2)))
		}
	}
	return out
}

func newRand(seed uint64) *rand.Rand {
	return rand.New(rand.NewPCG(seed, seed))
}

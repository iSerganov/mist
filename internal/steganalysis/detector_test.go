package steganalysis

import (
	"fmt"
	"math"
	"math/cmplx"
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
		{"spa on a smooth carrier at 100%", SPA, smooth, 1},
		{"rs on a smooth carrier at 100%", RS, smooth, 1},
		{"rs on a clean peaked carrier", RS, peaked, 0},
		{"rs on a peaked carrier at 50%", RS, peaked, 0.5},
	}
	for _, tc := range tests {
		s.Run(tc.title, func() {
			v := replaceLSB(tc.carrier(newRand(1)), tc.rate, newRand(2))
			s.InDelta(tc.rate, tc.score(v), 0.08)
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

func (s *DetectorSuite) TestGammaP() {
	tests := []struct {
		title string
		a, x  float64
		want  float64
	}{
		{"zero x", 3, 0, 0},
		{"exponential cdf through the series branch", 1, 1, 1 - math.Exp(-1)},
		{"half shape equals erf", 0.5, 2, math.Erf(math.Sqrt2)},
		{"chi-square 95th percentile at ten degrees", 5, 9.1535, 0.95},
		{"erlang tail through the continued fraction", 2, 10, 1 - 11*math.Exp(-10)},
	}
	for _, tc := range tests {
		s.Run(tc.title, func() {
			s.InDelta(tc.want, gammaP(tc.a, tc.x), 1e-5)
		})
	}
}

func (s *DetectorSuite) TestFFTMatchesNaiveDFT() {
	rng := newRand(3)
	in := make([]complex128, 16)
	for i := range in {
		in[i] = complex(rng.NormFloat64(), 0)
	}
	got := append([]complex128(nil), in...)
	fft(got)
	for k := range in {
		var want complex128
		for n, x := range in {
			want += x * cmplx.Rect(1, -2*math.Pi*float64(k*n)/float64(len(in)))
		}
		s.InDelta(real(want), real(got[k]), 1e-9)
		s.InDelta(imag(want), imag(got[k]), 1e-9)
	}
}

func (s *DetectorSuite) TestRootsTakeRealPartWhenDiscriminantIsNegative() {
	tests := []struct {
		title   string
		a, b, c float64
		lo, hi  float64
		ok      bool
	}{
		{"two real roots", 1, -3, 2, 1, 2, true},
		{"complex pair collapses to its real part", 1, -2, 5, 1, 1, true},
		{"linear equation", 0, 2, -4, 2, 2, true},
		{"degenerate equation", 0, 0, 1, 0, 0, false},
	}
	for _, tc := range tests {
		s.Run(tc.title, func() {
			lo, hi, ok := roots(tc.a, tc.b, tc.c)
			s.Equal(tc.ok, ok)
			s.InDelta(tc.lo, lo, 1e-12)
			s.InDelta(tc.hi, hi, 1e-12)
		})
	}
}

func (s *DetectorSuite) TestHCFHasBoundedRange() {
	tests := []struct {
		title string
		v     []int32
		zero  bool
	}{
		{"24-bit span is not scored", []int32{-4000000, 4000000, 1, 2, 3}, true},
		{"full int32 span is not scored", []int32{math.MinInt32, math.MaxInt32}, true},
		{"16-bit span is scored", append(peaked(newRand(1)), -32768, 32767), false},
	}
	for _, tc := range tests {
		s.Run(tc.title, func() {
			got := HCF(tc.v)
			s.Equal(tc.zero, got == 0)
		})
	}
}

func (s *DetectorSuite) TestHCFBarelyMovesWhenRangeCrossesPowerOfTwo() {
	v := peaked(newRand(1))
	below := append(append([]int32(nil), v...), 40)
	above := append(append([]int32(nil), v...), 100)
	s.InDelta(HCF(below), HCF(above), 1e-2)
}

func (s *DetectorSuite) TestScoresStayInUnitRange() {
	for _, d := range Detectors() {
		for _, rate := range []float64{0, 0.5, 1} {
			s.Run(fmt.Sprintf("%s at rate %v", d.Name, rate), func() {
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

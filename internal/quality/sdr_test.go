package quality

import (
	"math"
	"math/rand/v2"
	"testing"

	"github.com/stretchr/testify/suite"
)

const testLen = 20000

type SDRSuite struct {
	suite.Suite
}

func TestSDRSuite(t *testing.T) {
	suite.Run(t, &SDRSuite{})
}

func noise(seed uint64, ch, n int) [][]float32 {
	rng := rand.New(rand.NewPCG(seed, seed))
	out := make([][]float32, ch)
	for c := range out {
		out[c] = make([]float32, n)
		for i := range out[c] {
			out[c][i] = float32(rng.NormFloat64() * 0.2)
		}
	}
	return out
}

func silence(ch, n int) [][]float32 {
	out := make([][]float32, ch)
	for c := range out {
		out[c] = make([]float32, n)
	}
	return out
}

func delay(sig [][]float32, d int) [][]float32 {
	out := make([][]float32, len(sig))
	for c, p := range sig {
		out[c] = make([]float32, len(p))
		for i := range p {
			if j := i - d; j >= 0 && j < len(p) {
				out[c][i] = p[j]
			}
		}
	}
	return out
}

func scale(sig [][]float32, k float32) [][]float32 {
	out := make([][]float32, len(sig))
	for c, p := range sig {
		out[c] = make([]float32, len(p))
		for i, v := range p {
			out[c][i] = v * k
		}
	}
	return out
}

func (s *SDRSuite) TestLag() {
	ref := noise(1, 2, testLen)
	tests := []struct {
		title  string
		test   [][]float32
		maxLag int
		want   int
	}{
		{"identical signals", ref, 100, 0},
		{"test delayed", delay(ref, 37), 100, 37},
		{"test advanced", delay(ref, -23), 100, -23},
		{"silence prefers no shift", silence(2, testLen), 100, 0},
	}
	for _, tc := range tests {
		s.Run(tc.title, func() {
			s.Equal(tc.want, Lag(ref, tc.test, tc.maxLag))
		})
	}
}

func (s *SDRSuite) TestLagOfReferenceShorterThanSearchRangeIsZero() {
	short := noise(1, 2, 50)
	s.Zero(Lag(short, short, 100))
}

func (s *SDRSuite) TestLagIgnoresDelayBeyondRange() {
	ref := noise(1, 2, testLen)
	s.NotEqual(80, Lag(ref, delay(ref, 80), 50))
}

func (s *SDRSuite) TestSDRIsInfiniteWhenAlignedSignalsMatch() {
	ref := noise(1, 2, testLen)
	tests := []struct {
		title string
		test  [][]float32
	}{
		{"identical signals", ref},
		{"delayed copy", delay(ref, 37)},
		{"advanced copy", delay(ref, -23)},
	}
	for _, tc := range tests {
		s.Run(tc.title, func() {
			s.True(math.IsInf(SDR(ref, tc.test, 100), 1))
		})
	}
}

func (s *SDRSuite) TestSDRMeasuresDistortion() {
	ref := noise(1, 2, testLen)
	tests := []struct {
		title string
		test  [][]float32
		want  float64
	}{
		{"ten percent gain error is 20 dB", scale(ref, 1.1), 20},
		{"half amplitude is 6 dB", scale(ref, 0.5), 6.0206},
		{"a delayed copy with gain error is aligned first", delay(scale(ref, 1.1), 37), 20},
	}
	for _, tc := range tests {
		s.Run(tc.title, func() {
			s.InDelta(tc.want, SDR(ref, tc.test, 100), 1e-3)
		})
	}
}

func (s *SDRSuite) TestSDRDropsWhenMisaligned() {
	ref := noise(1, 2, testLen)
	s.Less(SDR(ref, delay(ref, 37), 10), 1.0)
}

func (s *SDRSuite) TestSegSNR() {
	ref := noise(1, 2, testLen)
	tests := []struct {
		title string
		ref   [][]float32
		test  [][]float32
		want  float64
	}{
		{"identical signals hit the ceiling", ref, ref, segCeiling},
		{"ten percent gain error is 20 dB in every segment", ref, scale(ref, 1.1), 20},
		{"noise against a silent reference hits the floor", silence(2, testLen), ref, segFloor},
		{"a delayed copy is aligned before measuring", ref, delay(ref, 37), segCeiling},
	}
	for _, tc := range tests {
		s.Run(tc.title, func() {
			s.InDelta(tc.want, SegSNR(tc.ref, tc.test, 100, 1000), 1e-3)
		})
	}
}

func (s *SDRSuite) TestSegSNRWithoutOverlapIsNaN() {
	s.True(math.IsNaN(SegSNR(nil, noise(1, 1, 10), 0, 4)))
}

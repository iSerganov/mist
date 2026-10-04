package dsp

import (
	"math"
	"math/cmplx"
	"math/rand/v2"
	"testing"

	"github.com/stretchr/testify/suite"
)

type FFTSuite struct {
	suite.Suite
}

func TestFFTSuite(t *testing.T) {
	suite.Run(t, &FFTSuite{})
}

func naiveDFT(in []complex128) []complex128 {
	out := make([]complex128, len(in))
	for k := range out {
		for t, v := range in {
			out[k] += v * cmplx.Rect(1, -2*math.Pi*float64(k*t)/float64(len(in)))
		}
	}
	return out
}

func randomSignal(n int) []complex128 {
	rng := rand.New(rand.NewPCG(1, 1))
	out := make([]complex128, n)
	for i := range out {
		out[i] = complex(rng.NormFloat64(), rng.NormFloat64())
	}
	return out
}

func (s *FFTSuite) TestNextPow2() {
	tests := []struct {
		title string
		n     int
		want  int
	}{
		{"zero", 0, 1},
		{"one", 1, 1},
		{"two", 2, 2},
		{"just above a power", 5, 8},
		{"exact power", 1024, 1024},
		{"one past a power", 1025, 2048},
	}
	for _, tc := range tests {
		s.Run(tc.title, func() {
			s.Equal(tc.want, NextPow2(tc.n))
		})
	}
}

func (s *FFTSuite) TestFFTMatchesNaiveDFT() {
	tests := []struct {
		title string
		in    []complex128
	}{
		{"single sample", []complex128{3}},
		{"impulse", []complex128{1, 0, 0, 0, 0, 0, 0, 0}},
		{"constant", []complex128{2, 2, 2, 2}},
		{"random 64", randomSignal(64)},
		{"random 1024", randomSignal(1024)},
	}
	for _, tc := range tests {
		s.Run(tc.title, func() {
			want := naiveDFT(tc.in)
			got := append([]complex128(nil), tc.in...)
			FFT(got)
			for i := range want {
				s.InDelta(0, cmplx.Abs(want[i]-got[i]), 1e-9)
			}
		})
	}
}

func (s *FFTSuite) TestIFFTInvertsFFT() {
	tests := []struct {
		title string
		n     int
	}{
		{"single sample", 1},
		{"pair", 2},
		{"16 samples", 16},
		{"4096 samples", 4096},
	}
	for _, tc := range tests {
		s.Run(tc.title, func() {
			in := randomSignal(tc.n)
			got := append([]complex128(nil), in...)
			FFT(got)
			IFFT(got)
			for i := range in {
				s.InDelta(0, cmplx.Abs(in[i]-got[i]), 1e-9)
			}
		})
	}
}

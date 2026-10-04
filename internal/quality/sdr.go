// Package quality measures how far a processed signal has drifted from its
// reference: signal-to-distortion ratios on planar float PCM, and
// perceptual scores from external tools.
package quality

import (
	"math"
	"math/cmplx"

	"github.com/iSerganov/mist/internal/dsp"
)

const (
	alignWindow = 1 << 18
	segFloor    = -10
	segCeiling  = 35
)

// Lag returns the offset d within ±maxLag at which test[i+d] best matches
// ref[i], by FFT cross-correlation of a window of ref starting maxLag
// samples in against test. Ties, including all-silent signals, resolve to
// the smallest |d|.
func Lag(ref, test [][]float32, maxLag int) int {
	chans := min(len(ref), len(test))
	if chans == 0 {
		return 0
	}
	start := min(maxLag, len(ref[0]))
	w := min(alignWindow, len(ref[0])-start)
	if w <= 0 {
		return 0
	}
	n := dsp.NextPow2(w + 2*maxLag)
	acc := make([]complex128, n)
	r, x := make([]complex128, n), make([]complex128, n)
	for ch := range chans {
		clear(r)
		clear(x)
		for i := range w {
			r[i] = complex(float64(ref[ch][start+i]), 0)
		}
		for i := range w + 2*maxLag {
			if j := start - maxLag + i; j >= 0 && j < len(test[ch]) {
				x[i] = complex(float64(test[ch][j]), 0)
			}
		}
		dsp.FFT(r)
		dsp.FFT(x)
		for k := range acc {
			acc[k] += cmplx.Conj(r[k]) * x[k]
		}
	}
	dsp.IFFT(acc)
	best, bestCorr := 0, real(acc[maxLag])
	for d := 1; d <= maxLag; d++ {
		for _, c := range []int{d, -d} {
			if v := real(acc[c+maxLag]); v > bestCorr {
				best, bestCorr = c, v
			}
		}
	}
	return best
}

// SDR returns the signal-to-distortion ratio of test against ref in dB,
// after aligning test by Lag. It is +Inf when the aligned signals are
// identical.
func SDR(ref, test [][]float32, maxLag int) float64 {
	d := Lag(ref, test, maxLag)
	lo, hi := span(ref, test, d)
	return db(energy(ref, test, d, lo, hi))
}

// SegSNR returns the mean segmental SNR of test against ref in dB over
// segments of seg samples, after aligning test by Lag. Each segment is
// clamped to [-10, 35] dB so silence cannot dominate the mean. It is NaN
// when the aligned signals do not overlap.
func SegSNR(ref, test [][]float32, maxLag, seg int) float64 {
	d := Lag(ref, test, maxLag)
	lo, hi := span(ref, test, d)
	var sum float64
	var n int
	for s := lo; s < hi; s += seg {
		sum += math.Max(segFloor, math.Min(segCeiling, db(energy(ref, test, d, s, min(s+seg, hi)))))
		n++
	}
	return sum / float64(n)
}

func energy(ref, test [][]float32, d, lo, hi int) (sig, noise float64) {
	for ch := range min(len(ref), len(test)) {
		r, t := ref[ch], test[ch]
		for i := lo; i < hi; i++ {
			e := float64(r[i]) - float64(t[i+d])
			sig += float64(r[i]) * float64(r[i])
			noise += e * e
		}
	}
	return sig, noise
}

func span(ref, test [][]float32, d int) (lo, hi int) {
	if len(ref) == 0 || len(test) == 0 {
		return 0, 0
	}
	lo = max(0, -d)
	hi = min(len(ref[0]), len(test[0])-d)
	return lo, max(lo, hi)
}

func db(sig, noise float64) float64 {
	if noise == 0 {
		return math.Inf(1)
	}
	return 10 * math.Log10(sig/noise)
}

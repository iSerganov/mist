// Package quality measures how far a processed signal has drifted from its
// reference: signal-to-distortion ratios on planar float PCM, and
// perceptual scores from external tools.
package quality

import "math"

const (
	alignWindow = 1 << 18
	segFloor    = -10
	segCeiling  = 35
)

// Lag returns the offset d within ±maxLag at which test[i+d] best matches
// ref[i], by cross-correlation over the first part of their overlap. Ties,
// including all-silent signals, resolve to the smallest |d|.
func Lag(ref, test [][]float32, maxLag int) int {
	best, bestCorr := 0, correlate(ref, test, 0)
	for d := 1; d <= maxLag; d++ {
		for _, c := range []int{d, -d} {
			if v := correlate(ref, test, c); v > bestCorr {
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

func correlate(ref, test [][]float32, d int) float64 {
	lo, hi := span(ref, test, d)
	hi = min(hi, lo+alignWindow)
	var c float64
	for ch := range min(len(ref), len(test)) {
		r, t := ref[ch], test[ch]
		for i := lo; i < hi; i++ {
			c += float64(r[i]) * float64(t[i+d])
		}
	}
	return c
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

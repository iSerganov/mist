package steganalysis

import (
	"math/cmplx"
	"slices"

	"github.com/iSerganov/mist/internal/dsp"
)

// HCF is Harmsen-Pearlman histogram characteristic function analysis. ±1
// embedding low-pass filters the value histogram, which pulls the centre of
// mass of its Fourier magnitude towards zero; the score is one minus that
// centre of mass, normalised to [0, 1].
func HCF(v []int32) float64 {
	if len(v) == 0 {
		return 0
	}
	lo, hi := slices.Min(v), slices.Max(v)
	n := dsp.NextPow2(int(int64(hi) - int64(lo) + 1))
	if n < 2 {
		return 0
	}
	h := make([]complex128, n)
	for _, x := range v {
		h[int64(x)-int64(lo)]++
	}
	dsp.FFT(h)
	var num, den float64
	for k := 1; k <= n/2; k++ {
		a := cmplx.Abs(h[k])
		num += float64(k) * a
		den += a
	}
	if den == 0 {
		return 0
	}
	return clamp01(1 - num/den/float64(n/2))
}

package steganalysis

import (
	"math"
	"math/bits"
	"math/cmplx"
	"slices"
)

const maxHCFBins = 1 << 20

// HCF is Harmsen-Pearlman histogram characteristic function analysis. ±1
// embedding low-pass filters the value histogram, which pulls the centre of
// mass of its Fourier magnitude towards zero; the score is one minus that
// centre of mass, normalised to [0, 1].
func HCF(v []int32) float64 {
	if len(v) == 0 {
		return 0
	}
	lo, hi := slices.Min(v), slices.Max(v)
	span := uint64(int64(hi) - int64(lo))
	if span >= maxHCFBins {
		return 0
	}
	n := 1 << bits.Len64(span)
	if n < 2 {
		return 0
	}
	h := make([]complex128, n)
	for _, x := range v {
		h[int64(x)-int64(lo)]++
	}
	fft(h)
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

func fft(a []complex128) {
	n := len(a)
	for i, j := 1, 0; i < n; i++ {
		bit := n >> 1
		for ; j&bit != 0; bit >>= 1 {
			j ^= bit
		}
		j ^= bit
		if i < j {
			a[i], a[j] = a[j], a[i]
		}
	}
	for size := 2; size <= n; size <<= 1 {
		w := cmplx.Rect(1, -2*math.Pi/float64(size))
		for start := 0; start < n; start += size {
			wk := complex(1, 0)
			for k := range size / 2 {
				u, t := a[start+k], wk*a[start+k+size/2]
				a[start+k], a[start+k+size/2] = u+t, u-t
				wk *= w
			}
		}
	}
}

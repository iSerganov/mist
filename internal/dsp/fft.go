// Package dsp holds signal-processing primitives shared by the analysis
// packages.
package dsp

import (
	"math"
	"math/bits"
	"math/cmplx"
)

// NextPow2 returns the smallest power of two that is at least n, and 1 for
// n below 2.
func NextPow2(n int) int {
	if n < 2 {
		return 1
	}
	return 1 << bits.Len(uint(n-1))
}

// FFT replaces a with its discrete Fourier transform, in place. len(a) must
// be a power of two.
func FFT(a []complex128) {
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

// IFFT replaces a with its inverse discrete Fourier transform, in place.
// len(a) must be a power of two.
func IFFT(a []complex128) {
	for i := range a {
		a[i] = cmplx.Conj(a[i])
	}
	FFT(a)
	scale := complex(1/float64(len(a)), 0)
	for i := range a {
		a[i] = cmplx.Conj(a[i]) * scale
	}
}

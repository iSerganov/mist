package steganalysis

import "math"

const rsGroup = 4

var rsMask = [rsGroup]int32{0, 1, 1, 0}

// RS is Fridrich's regular/singular groups analysis. It estimates the
// fraction of values carrying LSB-replaced bits from how flipping LSBs
// changes the smoothness of small groups.
func RS(v []int32) float64 {
	if len(v) < rsGroup {
		return 0
	}
	inv := make([]int32, len(v))
	for i, x := range v {
		inv[i] = x ^ 1
	}
	d0 := rsDelta(v, 1)
	d1 := rsDelta(inv, 1)
	dn0 := rsDelta(v, -1)
	dn1 := rsDelta(inv, -1)
	r1, r2, ok := roots(2*(d1+d0), dn0-dn1-d1-3*d0, d0-dn0)
	if !ok {
		return 0
	}
	z := r1
	if math.Abs(r2) < math.Abs(r1) {
		z = r2
	}
	return clamp01(z / (z - 0.5))
}

func rsDelta(v []int32, sign int32) float64 {
	var regular, singular, groups float64
	var g [rsGroup]int32
	for off := 0; off+rsGroup <= len(v); off += rsGroup {
		for i := range g {
			g[i] = flip(v[off+i], rsMask[i]*sign)
		}
		before, after := roughness(v[off:off+rsGroup]), roughness(g[:])
		switch {
		case after > before:
			regular++
		case after < before:
			singular++
		}
		groups++
	}
	return (regular - singular) / groups
}

func flip(x, m int32) int32 {
	switch m {
	case 1:
		return x ^ 1
	case -1:
		return ((x + 1) ^ 1) - 1
	}
	return x
}

func roughness(g []int32) int64 {
	var s int64
	for i := 0; i+1 < len(g); i++ {
		d := int64(g[i+1]) - int64(g[i])
		if d < 0 {
			d = -d
		}
		s += d
	}
	return s
}

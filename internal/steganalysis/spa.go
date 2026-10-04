package steganalysis

// SPA is Dumitrescu-Wu-Wang sample pair analysis, summed over every
// difference class as Ker formulates it. It estimates the fraction of
// values carrying LSB-replaced bits from how odd-difference pairs split
// between an odd and an even smaller value.
func SPA(v []int32) float64 {
	var c0, d0, x, y float64
	for i := 0; i+1 < len(v); i++ {
		u, w := v[i], v[i+1]
		if u>>1 == w>>1 {
			c0++
		}
		switch {
		case u == w:
			d0++
		case (u^w)&1 == 1 && min(u, w)&1 == 1:
			x++
		case (u^w)&1 == 1:
			y++
		}
	}
	r1, r2, ok := roots(c0/2, -(d0 + y - x), y-x)
	if !ok {
		return 0
	}
	return clamp01(min(r1, r2))
}

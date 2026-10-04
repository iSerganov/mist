package steganalysis

import (
	"maps"
	"math"
	"slices"
)

const minExpected = 5

// ChiSquare is the Westfeld-Pfitzmann pairs-of-values test. LSB replacement
// pulls the counts of 2k and 2k+1 towards each other; the score is the
// chi-square p-value that they are already equal.
func ChiSquare(v []int32) float64 {
	pairs := map[int32][2]float64{}
	for _, x := range v {
		p := pairs[x>>1]
		p[x&1]++
		pairs[x>>1] = p
	}
	var chi float64
	df := -1
	for _, k := range slices.Sorted(maps.Keys(pairs)) {
		p := pairs[k]
		e := (p[0] + p[1]) / 2
		if e < minExpected {
			continue
		}
		chi += (p[0] - e) * (p[0] - e) / e
		df++
	}
	if df < 1 {
		return 0
	}
	return 1 - gammaP(float64(df)/2, chi/2)
}

func gammaP(a, x float64) float64 {
	if x <= 0 {
		return 0
	}
	lg, _ := math.Lgamma(a)
	front := math.Exp(a*math.Log(x) - x - lg)
	if x < a+1 {
		sum, del := 1/a, 1/a
		for ap := a + 1; math.Abs(del) > math.Abs(sum)*1e-15; ap++ {
			del *= x / ap
			sum += del
		}
		return clamp01(sum * front)
	}
	const tiny = 1e-300
	b := x + 1 - a
	c, d := 1/tiny, 1/b
	h := d
	for i := 1.0; i < 1e6; i++ {
		an := -i * (i - a)
		b += 2
		d = an*d + b
		if math.Abs(d) < tiny {
			d = tiny
		}
		c = b + an/c
		if math.Abs(c) < tiny {
			c = tiny
		}
		d = 1 / d
		del := d * c
		h *= del
		if math.Abs(del-1) < 1e-15 {
			break
		}
	}
	return clamp01(1 - front*h)
}

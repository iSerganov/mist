// Package steganalysis scores streams of quantized values for traces of LSB
// embedding and measures how well those scores separate two populations.
package steganalysis

import "math"

// Detector is a named steganalysis test. Score returns a value in [0, 1];
// higher means the stream is more likely to carry embedded data.
type Detector struct {
	Name  string
	Score func([]int32) float64
}

// Detectors returns every classical detector in this package.
func Detectors() []Detector {
	return []Detector{
		{Name: "chi-square", Score: ChiSquare},
		{Name: "spa", Score: SPA},
		{Name: "rs", Score: RS},
		{Name: "hcf-com", Score: HCF},
	}
}

func roots(a, b, c float64) (float64, float64, bool) {
	if a == 0 {
		if b == 0 {
			return 0, 0, false
		}
		r := -c / b
		return r, r, true
	}
	s := math.Sqrt(math.Max(0, b*b-4*a*c))
	return (-b - s) / (2 * a), (-b + s) / (2 * a), true
}

func clamp01(x float64) float64 {
	return math.Max(0, math.Min(1, x))
}

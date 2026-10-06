package steganalysis

import (
	"math"
	"slices"
)

// MaxLR caps one detector's likelihood ratio, so a score in the tail of a
// small reference population cannot decide a verdict alone.
const MaxLR = 20

// minBandwidth keeps the kernel from collapsing onto a population whose
// scores are all equal, as chi-square's are on audio it cannot read.
const minBandwidth = 1e-3

// densityFloor is added to both densities, so a score far from either
// population reads as no evidence rather than 0/0.
const densityFloor = 1e-9

// Reference is one detector's file scores on known clean and known stego
// files, the population a single file's score is read against.
type Reference struct {
	Clean []float64 `json:"clean"`
	Stego []float64 `json:"stego"`
}

// LikelihoodRatio is how much likelier score is under the stego population
// than under the clean one, from a Gaussian kernel density estimate of
// each, clamped to [1/MaxLR, MaxLR]. Above 1 favours stego.
func (r Reference) LikelihoodRatio(score float64) float64 {
	if len(r.Clean) == 0 || len(r.Stego) == 0 {
		return 1
	}
	lr := (density(r.Stego, score) + densityFloor) / (density(r.Clean, score) + densityFloor)
	return math.Max(1.0/MaxLR, math.Min(MaxLR, lr))
}

// AUC is how well the reference populations separate: 0.5 is chance.
func (r Reference) AUC() float64 {
	return AUC(r.Stego, r.Clean)
}

// Posterior combines likelihood ratios into a probability of stego at the
// given prior. It averages their logarithms rather than summing them: the
// detectors read overlapping features of one file, so their evidence is
// not independent and a sum would count it several times.
func Posterior(lrs []float64, prior float64) float64 {
	if len(lrs) == 0 {
		return prior
	}
	var sum float64
	for _, lr := range lrs {
		sum += math.Log(lr)
	}
	odds := prior / (1 - prior) * math.Exp(sum/float64(len(lrs)))
	return odds / (1 + odds)
}

func density(sample []float64, x float64) float64 {
	h := bandwidth(sample)
	var sum float64
	for _, v := range sample {
		z := (x - v) / h
		sum += math.Exp(-z * z / 2)
	}
	return sum / (float64(len(sample)) * h * math.Sqrt(2*math.Pi))
}

// bandwidth is Silverman's rule of thumb.
func bandwidth(sample []float64) float64 {
	n := float64(len(sample))
	var mean float64
	for _, v := range sample {
		mean += v
	}
	mean /= n
	var ss float64
	for _, v := range sample {
		ss += (v - mean) * (v - mean)
	}
	sd := math.Sqrt(ss / n)
	sorted := slices.Sorted(slices.Values(sample))
	iqr := quantile(sorted, 0.75) - quantile(sorted, 0.25)
	spread := sd
	if iqr > 0 {
		spread = math.Min(sd, iqr/1.34)
	}
	return math.Max(minBandwidth, 0.9*spread*math.Pow(n, -0.2))
}

func quantile(sorted []float64, q float64) float64 {
	pos := q * float64(len(sorted)-1)
	lo := int(pos)
	hi := min(lo+1, len(sorted)-1)
	return sorted[lo] + (pos-float64(lo))*(sorted[hi]-sorted[lo])
}

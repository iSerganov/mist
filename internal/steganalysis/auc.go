package steganalysis

import (
	"cmp"
	"math"
	"math/rand/v2"
	"slices"
)

// AUC is the area under the ROC curve for scores of a positive and a
// negative population: the probability that a random positive outscores a
// random negative, ties counting half. 0.5 is chance.
func AUC(pos, neg []float64) float64 {
	if len(pos) == 0 || len(neg) == 0 {
		return 0.5
	}
	type obs struct {
		v   float64
		pos bool
	}
	all := make([]obs, 0, len(pos)+len(neg))
	for _, v := range pos {
		all = append(all, obs{v, true})
	}
	for _, v := range neg {
		all = append(all, obs{v, false})
	}
	slices.SortFunc(all, func(a, b obs) int { return cmp.Compare(a.v, b.v) })
	var rankSum float64
	for i := 0; i < len(all); {
		j := i
		for j < len(all) && all[j].v == all[i].v {
			j++
		}
		rank := float64(i+j+1) / 2
		for _, o := range all[i:j] {
			if o.pos {
				rankSum += rank
			}
		}
		i = j
	}
	np, nn := float64(len(pos)), float64(len(neg))
	return (rankSum - np*(np+1)/2) / (np * nn)
}

// AUCInterval is a percentile bootstrap 95% confidence interval for AUC,
// resampling each population rounds times from a generator seeded by seed.
func AUCInterval(pos, neg []float64, rounds int, seed uint64) (lo, hi float64) {
	if rounds < 1 {
		auc := AUC(pos, neg)
		return auc, auc
	}
	rng := rand.New(rand.NewPCG(seed, seed))
	bp, bn := make([]float64, len(pos)), make([]float64, len(neg))
	aucs := make([]float64, rounds)
	for r := range aucs {
		resample(rng, pos, bp)
		resample(rng, neg, bn)
		aucs[r] = AUC(bp, bn)
	}
	slices.Sort(aucs)
	last := float64(rounds - 1)
	return aucs[int(0.025*last)], aucs[int(math.Ceil(0.975*last))]
}

func resample(rng *rand.Rand, src, dst []float64) {
	for i := range dst {
		dst[i] = src[rng.IntN(len(src))]
	}
}

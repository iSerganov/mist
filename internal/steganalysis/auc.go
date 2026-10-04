package steganalysis

import (
	"cmp"
	"maps"
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

// AUCInterval is a percentile bootstrap 95% confidence interval for AUC.
// Scores come in groups that are not independent, such as the chunks of
// one carrier scored stego and clean, so each of rounds draws picks whole
// groups with replacement and takes every positive and negative score of
// each. posGroup and negGroup give each score's group; seed fixes the draws.
func AUCInterval(pos, neg []float64, posGroup, negGroup []int, rounds int, seed uint64) (lo, hi float64) {
	if rounds < 1 {
		auc := AUC(pos, neg)
		return auc, auc
	}
	type cluster struct{ pos, neg []float64 }
	byID := map[int]*cluster{}
	member := func(id int) *cluster {
		if byID[id] == nil {
			byID[id] = &cluster{}
		}
		return byID[id]
	}
	for i, v := range pos {
		c := member(posGroup[i])
		c.pos = append(c.pos, v)
	}
	for i, v := range neg {
		c := member(negGroup[i])
		c.neg = append(c.neg, v)
	}
	ids := slices.Sorted(maps.Keys(byID))
	rng := rand.New(rand.NewPCG(seed, seed))
	aucs := make([]float64, rounds)
	var bp, bn []float64
	for r := range aucs {
		bp, bn = bp[:0], bn[:0]
		for range ids {
			c := byID[ids[rng.IntN(len(ids))]]
			bp, bn = append(bp, c.pos...), append(bn, c.neg...)
		}
		aucs[r] = AUC(bp, bn)
	}
	slices.Sort(aucs)
	last := float64(rounds - 1)
	return aucs[int(0.025*last)], aucs[int(math.Ceil(0.975*last))]
}

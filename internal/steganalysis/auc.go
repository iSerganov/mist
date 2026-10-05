package steganalysis

import (
	"cmp"
	"math"
	"math/rand/v2"
	"slices"
)

// Detectability folds a reversed detector back onto the same scale as a
// forward one. 0.5 is chance and 1 is a perfect separation either way.
func Detectability(auc float64) float64 {
	d := auc - 0.5
	if d < 0 {
		d = -d
	}
	return 0.5 + d
}

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
// groups with replacement. posGroup and negGroup give each score's group;
// seed fixes the draws.
//
// A round compares every drawn group's positives with the negatives of
// the same draw and of every other drawn group, never of a second draw
// of the same group: that pair would be a within-group comparison passing
// for a between-group one. Between-group pairs are scaled by n/(n−1) for
// n groups, so on average a round weights the two kinds as the data does.
// Without that, a group drawn twice counts its own paired comparison four
// times over, and the interval drifts off the estimate towards it.
func AUCInterval(pos, neg []float64, posGroup, negGroup []int, rounds int, seed uint64) (lo, hi float64) {
	if rounds < 1 {
		auc := AUC(pos, neg)
		return auc, auc
	}
	ids, wins, pairs := clusterPairs(pos, neg, posGroup, negGroup)
	n := len(ids)
	between := 1.0
	if n > 1 {
		between = float64(n) / float64(n-1)
	}
	rng := rand.New(rand.NewPCG(seed, seed))
	draws := make([]float64, n)
	aucs := make([]float64, rounds)
	for r := range aucs {
		clear(draws)
		for range n {
			draws[rng.IntN(n)]++
		}
		var w, p float64
		for c, kc := range draws {
			for d, kd := range draws {
				k := kc * kd * between
				if c == d {
					k = kc
				}
				w += k * wins[c][d]
				p += k * pairs[c][d]
			}
		}
		aucs[r] = 0.5
		if p > 0 {
			aucs[r] = w / p
		}
	}
	slices.Sort(aucs)
	last := float64(rounds - 1)
	return aucs[int(0.025*last)], aucs[int(math.Ceil(0.975*last))]
}

// HierarchicalInterval is a two-stage percentile bootstrap for AUC.
// Families are drawn with replacement, then the recordings inside each
// drawn family. posFamily and posRecording align with pos; the negative
// slices align with neg. A recording drawn once contributes its paired
// comparison once, the same way AUCInterval does, so a family of one
// recording reproduces that interval for the same seed.
func HierarchicalInterval(pos, neg []float64, posFamily, posRecording, negFamily, negRecording []int, rounds int, seed uint64) (lo, hi float64) {
	if rounds < 1 || len(pos) == 0 || len(neg) == 0 {
		auc := AUC(pos, neg)
		return auc, auc
	}
	ids, wins, pairs := clusterPairs(pos, neg, posRecording, negRecording)
	recFamily := map[int]int{}
	for i, rec := range posRecording {
		if i < len(posFamily) {
			recFamily[rec] = posFamily[i]
		}
	}
	for i, rec := range negRecording {
		if _, ok := recFamily[rec]; !ok && i < len(negFamily) {
			recFamily[rec] = negFamily[i]
		}
	}
	famOf := make([]int, len(ids))
	for i, id := range ids {
		famOf[i] = recFamily[id]
	}
	var members [][]int
	at := map[int]int{}
	for rec := range famOf {
		fam := famOf[rec]
		slot, ok := at[fam]
		if !ok {
			slot = len(members)
			at[fam] = slot
			members = append(members, nil)
		}
		members[slot] = append(members[slot], rec)
	}
	n := len(ids)
	between := 1.0
	if n > 1 {
		between = float64(n) / float64(n-1)
	}
	direct := len(members) == n
	for i, mem := range members {
		if len(mem) != 1 || mem[0] != i {
			direct = false
		}
	}
	rng := rand.New(rand.NewPCG(seed, seed))
	aucs := make([]float64, rounds)
	for r := range aucs {
		draws := make([]float64, n)
		if direct {
			for range n {
				draws[rng.IntN(n)]++
			}
		} else {
			for range len(members) {
				mem := members[rng.IntN(len(members))]
				if len(mem) == 1 {
					draws[mem[0]]++
					continue
				}
				for range len(mem) {
					draws[mem[rng.IntN(len(mem))]]++
				}
			}
		}
		var w, p float64
		for c, kc := range draws {
			for d, kd := range draws {
				k := kc * kd * between
				if c == d {
					k = kc
				}
				w += k * wins[c][d]
				p += k * pairs[c][d]
			}
		}
		aucs[r] = 0.5
		if p > 0 {
			aucs[r] = w / p
		}
	}
	slices.Sort(aucs)
	last := float64(rounds - 1)
	return aucs[int(0.025*last)], aucs[int(math.Ceil(0.975*last))]
}

// clusterPairs counts, for every pair of groups c and d, how many of c's
// positives outscore d's negatives (ties counting half) out of how many
// such pairs there are. Summed over all c and d they give AUC exactly.
func clusterPairs(pos, neg []float64, posGroup, negGroup []int) (ids []int, wins, pairs [][]float64) {
	at := map[int]int{}
	index := func(id int) int {
		if _, ok := at[id]; !ok {
			at[id] = len(ids)
			ids = append(ids, id)
		}
		return at[id]
	}
	for _, g := range posGroup {
		index(g)
	}
	for _, g := range negGroup {
		index(g)
	}
	byPos, byNeg := make([][]float64, len(ids)), make([][]float64, len(ids))
	for i, v := range pos {
		c := at[posGroup[i]]
		byPos[c] = append(byPos[c], v)
	}
	for i, v := range neg {
		d := at[negGroup[i]]
		byNeg[d] = append(byNeg[d], v)
	}
	for _, ns := range byNeg {
		slices.Sort(ns)
	}
	wins, pairs = make([][]float64, len(ids)), make([][]float64, len(ids))
	for c, ps := range byPos {
		wins[c], pairs[c] = make([]float64, len(ids)), make([]float64, len(ids))
		for d, ns := range byNeg {
			for _, v := range ps {
				below, _ := slices.BinarySearch(ns, v)
				upTo, _ := slices.BinarySearch(ns, math.Nextafter(v, math.Inf(1)))
				wins[c][d] += float64(below) + float64(upTo-below)/2
			}
			pairs[c][d] = float64(len(ps) * len(ns))
		}
	}
	return ids, wins, pairs
}

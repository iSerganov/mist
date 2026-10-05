package steganalysis

import "math/rand/v2"

// PairedPermutationP is the two-sided p-value for |AUC−½| when whole
// families swap their positive and negative labels together. pos[i] is
// paired with neg[i] and family[i]. The observed labelling counts as one
// permutation, so the p-value cannot be zero.
func PairedPermutationP(pos, neg []float64, family []int, perms int, seed uint64) float64 {
	if len(pos) == 0 || len(pos) != len(neg) || len(pos) != len(family) || perms < 1 {
		return 1
	}
	observed := Detectability(AUC(pos, neg)) - 0.5
	extreme := 1
	workPos, workNeg := append([]float64(nil), pos...), append([]float64(nil), neg...)
	ids := uniqueInts(family)
	rng := rand.New(rand.NewPCG(seed, seed^0x9e3779b97f4a7c15))
	for range perms {
		flip := map[int]bool{}
		for _, id := range ids {
			flip[id] = rng.IntN(2) == 1
		}
		copy(workPos, pos)
		copy(workNeg, neg)
		for i, id := range family {
			if flip[id] {
				workPos[i], workNeg[i] = workNeg[i], workPos[i]
			}
		}
		if Detectability(AUC(workPos, workNeg))-0.5 >= observed {
			extreme++
		}
	}
	return float64(extreme) / float64(perms+1)
}

// RefitPermutationP repeats NestedCrossValidate after the same family-wise
// label swap. It is the confirmatory null for a model whose scores depend
// on the labels, and it is deterministic for a seed.
func RefitPermutationP(x [][]float64, y []bool, groups []int, folds, perms int, seed uint64) float64 {
	if len(x) == 0 || len(x) != len(y) || len(x) != len(groups) || perms < 1 {
		return 1
	}
	observed := orientation(scoreAUC(NestedCrossValidate(x, y, groups, folds), y))
	extreme := 1
	ids := uniqueInts(groups)
	rng := rand.New(rand.NewPCG(seed, seed^0x6a09e667f3bcc909))
	swapped := make([]bool, len(y))
	for range perms {
		flip := map[int]bool{}
		for _, id := range ids {
			flip[id] = rng.IntN(2) == 1
		}
		for i, g := range groups {
			swapped[i] = y[i]
			if flip[g] {
				swapped[i] = !swapped[i]
			}
		}
		if orientation(scoreAUC(NestedCrossValidate(x, swapped, groups, folds), swapped)) >= observed {
			extreme++
		}
	}
	return float64(extreme) / float64(perms+1)
}

func scoreAUC(scores []float64, y []bool) float64 {
	var pos, neg []float64
	for i, score := range scores {
		if y[i] {
			pos = append(pos, score)
			continue
		}
		neg = append(neg, score)
	}
	return AUC(pos, neg)
}

func orientation(auc float64) float64 { return Detectability(auc) - 0.5 }

func uniqueInts(v []int) []int {
	seen := map[int]bool{}
	var out []int
	for _, id := range v {
		if !seen[id] {
			seen[id] = true
			out = append(out, id)
		}
	}
	return out
}

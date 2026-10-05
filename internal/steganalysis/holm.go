package steganalysis

import (
	"cmp"
	"slices"
)

// Holm adjusts p-values with Holm's step-down procedure so a family of
// confirmatory tests keeps its family-wise error rate. The result is in
// the input order. A non-finite value is treated as 1.
func Holm(p []float64) []float64 {
	n := len(p)
	idx := make([]int, n)
	for i := range idx {
		idx[i] = i
	}
	finite := func(v float64) float64 {
		if v != v || v < 0 {
			return 1
		}
		if v > 1 {
			return 1
		}
		return v
	}
	slices.SortFunc(idx, func(a, b int) int { return cmp.Compare(finite(p[a]), finite(p[b])) })
	adj := make([]float64, n)
	var running float64
	for rank, i := range idx {
		v := float64(n-rank) * finite(p[i])
		if v > 1 {
			v = 1
		}
		if v < running {
			v = running
		}
		running = v
		adj[i] = running
	}
	return adj
}

// BH adjusts p-values with Benjamini-Hochberg so an exploratory family can
// be read at a chosen false-discovery rate. The result is in the input
// order. A non-finite value is treated as 1.
func BH(p []float64) []float64 {
	n := len(p)
	idx := make([]int, n)
	for i := range idx {
		idx[i] = i
	}
	finite := func(v float64) float64 {
		if v != v || v < 0 || v > 1 {
			return 1
		}
		return v
	}
	slices.SortFunc(idx, func(a, b int) int { return cmp.Compare(finite(p[a]), finite(p[b])) })
	adj := make([]float64, n)
	running := 1.0
	for rank := n - 1; rank >= 0; rank-- {
		i := idx[rank]
		v := float64(n) / float64(rank+1) * finite(p[i])
		if v > 1 {
			v = 1
		}
		if v > running {
			v = running
		}
		running = v
		adj[i] = running
	}
	return adj
}

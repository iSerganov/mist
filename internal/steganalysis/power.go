package steganalysis

import "math/rand/v2"

const (
	powerCap       = 256
	powerSims      = 30
	powerBootstrap = 40
)

// FamiliesForPower estimates how many independent paired families are
// needed for a lineage-cluster 95% interval to exclude 0.5 at least `power`
// of the time, when the location shift is set so the pilot itself has AUC
// `target`. The pilot's own dispersion is kept. reached is false when the
// search stops at powerCap without crossing `power`, or when the pilot
// cannot be shifted to `target`.
func FamiliesForPower(pos, neg []float64, target, power float64, seed uint64) (families int, reached bool) {
	if len(pos) < 4 || len(pos) != len(neg) || power <= 0 || power >= 1 {
		return 0, false
	}
	shifted, ok := shiftToAUC(pos, neg, target)
	if !ok {
		return 0, false
	}
	for n := 8; n <= powerCap; n *= 2 {
		if powerAt(shifted, neg, n, powerSims, seed) >= power {
			return n, true
		}
	}
	return powerCap, false
}

func shiftToAUC(pos, neg []float64, target float64) ([]float64, bool) {
	base := append([]float64(nil), pos...)
	var mean float64
	for i := range base {
		mean += (base[i] - neg[i]) / float64(len(base))
	}
	for i := range base {
		base[i] -= mean
	}
	lo, hi := -2.0, 2.0
	var best []float64
	bestGap := 1.0
	for range 24 {
		mid := (lo + hi) / 2
		trial := append([]float64(nil), base...)
		for i := range trial {
			trial[i] += mid
		}
		gap := AUC(trial, neg) - target
		if abs(gap) < bestGap {
			bestGap = abs(gap)
			best = trial
		}
		if gap < 0 {
			lo = mid
			continue
		}
		hi = mid
	}
	// Eight paired scores move AUC in steps of 1/64, so the nearest
	// attainable value can sit a little over 0.02 away from the target.
	if best == nil || bestGap > 0.03 {
		return nil, false
	}
	return best, true
}

func powerAt(pos, neg []float64, n, sims int, seed uint64) float64 {
	rng := rand.New(rand.NewPCG(seed, uint64(n)))
	var hits int
	family := make([]int, n)
	for i := range family {
		family[i] = i
	}
	drawPos, drawNeg := make([]float64, n), make([]float64, n)
	for sim := range sims {
		for i := range n {
			j := rng.IntN(len(pos))
			drawPos[i], drawNeg[i] = pos[j], neg[j]
		}
		lo, hi := AUCInterval(drawPos, drawNeg, family, family, powerBootstrap, seed+uint64(sim))
		if lo > 0.5 || hi < 0.5 {
			hits++
		}
	}
	return float64(hits) / float64(sims)
}

func abs(v float64) float64 {
	if v < 0 {
		return -v
	}
	return v
}

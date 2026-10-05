package steganalysis

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/suite"
)

type AUCSuite struct {
	suite.Suite
}

func TestAUCSuite(t *testing.T) {
	suite.Run(t, &AUCSuite{})
}

func (s *AUCSuite) TestAUC() {
	tests := []struct {
		title string
		pos   []float64
		neg   []float64
		want  float64
	}{
		{"perfectly separated", []float64{3, 4, 5}, []float64{0, 1, 2}, 1},
		{"perfectly reversed", []float64{0, 1, 2}, []float64{3, 4, 5}, 0},
		{"identical populations", []float64{1, 2, 3}, []float64{1, 2, 3}, 0.5},
		{"all scores tied", []float64{7, 7}, []float64{7, 7, 7}, 0.5},
		{"partial overlap", []float64{2, 3}, []float64{1, 2.5}, 0.75},
		{"tie counts half", []float64{1, 2}, []float64{1}, 0.75},
		{"empty positives", nil, []float64{1}, 0.5},
		{"empty negatives", []float64{1}, nil, 0.5},
	}
	for _, tc := range tests {
		s.Run(tc.title, func() {
			s.InDelta(tc.want, AUC(tc.pos, tc.neg), 1e-12)
		})
	}
}

// ungrouped puts every score in a group of its own.
func ungrouped(n, from int) []int {
	g := make([]int, n)
	for i := range g {
		g[i] = from + i
	}
	return g
}

func (s *AUCSuite) TestAUCIntervalPinsUnambiguousPopulations() {
	tests := []struct {
		title string
		pos   []float64
		neg   []float64
		want  float64
	}{
		{"separated populations pin both bounds at one", []float64{3, 4, 5}, []float64{0, 1, 2}, 1},
		{"constant populations pin both bounds at chance", []float64{1, 1}, []float64{1, 1}, 0.5},
	}
	for _, tc := range tests {
		s.Run(tc.title, func() {
			lo, hi := AUCInterval(tc.pos, tc.neg, ungrouped(len(tc.pos), 0), ungrouped(len(tc.neg), 0), 500, 7)
			s.Equal(tc.want, lo)
			s.Equal(tc.want, hi)
		})
	}
}

func (s *AUCSuite) TestAUCIntervalBracketsEstimate() {
	pos := []float64{0.3, 0.6, 0.45, 0.95, 0.7, 0.5, 0.85, 0.75}
	neg := []float64{0.1, 0.4, 0.35, 0.8, 0.65, 0.2, 0.9, 0.55}
	lo, hi := AUCInterval(pos, neg, ungrouped(len(pos), 0), ungrouped(len(neg), 0), 500, 7)
	auc := AUC(pos, neg)
	s.LessOrEqual(lo, auc)
	s.GreaterOrEqual(hi, auc)
	s.Less(lo, hi)
}

func (s *AUCSuite) TestAUCIntervalWidensWhenScoresShareAGroup() {
	const copies = 50
	base := []struct{ pos, neg float64 }{{0.3, 0.1}, {0.6, 0.4}, {0.45, 0.35}, {0.95, 0.8}, {0.7, 0.65}, {0.5, 0.2}}
	var pos, neg []float64
	var groups []int
	for g, b := range base {
		for range copies {
			pos, neg, groups = append(pos, b.pos), append(neg, b.neg), append(groups, g)
		}
	}
	clo, chi := AUCInterval(pos, neg, groups, groups, 500, 7)
	ulo, uhi := AUCInterval(pos, neg, ungrouped(len(pos), 0), ungrouped(len(neg), len(pos)), 500, 7)
	s.Greater(chi-clo, 2*(uhi-ulo))
}

// A stego copy scored just above its own clean copy is what the harness
// sees for every carrier. Redrawing a carrier must not count that paired
// comparison more often than the data does, or the interval leaves the
// estimate behind; the old bootstrap missed it in 199 of 200 trials.
func (s *AUCSuite) TestAUCIntervalBracketsEstimateForPairedCarriers() {
	tests := []struct {
		title     string
		carriers  int
		chunks    int
		shift     float64
		maxMisses int
	}{
		{"one score per carrier, stego a hair above", 13, 1, 0.01, 10},
		{"one score per carrier, stego a hair below", 13, 1, -0.01, 10},
		{"many chunks per carrier", 13, 20, 0.01, 10},
	}
	for _, tc := range tests {
		s.Run(tc.title, func() {
			rng := newRand(5)
			misses := 0
			for trial := range 200 {
				var pos, neg []float64
				var groups []int
				for c := range tc.carriers {
					b := rng.NormFloat64()
					for range tc.chunks {
						v := b + 0.1*rng.NormFloat64()
						pos, neg, groups = append(pos, v+tc.shift), append(neg, v), append(groups, c)
					}
				}
				lo, hi := AUCInterval(pos, neg, groups, groups, 500, uint64(trial))
				if auc := AUC(pos, neg); auc < lo || auc > hi {
					misses++
				}
			}
			s.LessOrEqual(misses, tc.maxMisses)
		})
	}
}

func (s *AUCSuite) TestClusterPairsSumToAUC() {
	tests := []struct {
		title    string
		pos, neg []float64
		pg, ng   []int
	}{
		{"paired groups", []float64{0.3, 0.6, 0.45, 0.9}, []float64{0.1, 0.65, 0.45, 0.2}, []int{0, 0, 1, 2}, []int{0, 1, 1, 2}},
		{"ties across groups", []float64{1, 1, 2}, []float64{1, 2, 2}, []int{0, 1, 1}, []int{1, 0, 2}},
		{"groups on one side only", []float64{0.2, 0.8}, []float64{0.5}, []int{0, 1}, []int{2}},
	}
	for _, tc := range tests {
		s.Run(tc.title, func() {
			_, wins, pairs := clusterPairs(tc.pos, tc.neg, tc.pg, tc.ng)
			var w, p float64
			for c := range wins {
				for d := range wins[c] {
					w, p = w+wins[c][d], p+pairs[c][d]
				}
			}
			s.InDelta(AUC(tc.pos, tc.neg), w/p, 1e-12)
		})
	}
}

func (s *AUCSuite) TestAUCIntervalIsDeterministicPerSeed() {
	pos := []float64{0.3, 0.6, 0.45, 0.95, 0.7}
	neg := []float64{0.1, 0.4, 0.35, 0.8, 0.65}
	groups := []int{0, 0, 1, 2, 2}
	lo1, hi1 := AUCInterval(pos, neg, groups, groups, 200, 42)
	lo2, hi2 := AUCInterval(pos, neg, groups, groups, 200, 42)
	s.Equal(lo1, lo2)
	s.Equal(hi1, hi2)
}

func (s *AUCSuite) TestDetectabilityFoldsReversedDetectors() {
	s.Equal(0.5, Detectability(0.5))
	s.Equal(0.9, Detectability(0.9))
	s.Equal(0.9, Detectability(0.1))
}

func (s *AUCSuite) TestHierarchicalIntervalMatchesCarrierBootstrap() {
	pos := []float64{0.3, 0.6, 0.45, 0.95, 0.7}
	neg := []float64{0.1, 0.4, 0.35, 0.8, 0.65}
	groups := []int{0, 0, 1, 2, 2}
	wantLo, wantHi := AUCInterval(pos, neg, groups, groups, 200, 42)
	lo, hi := HierarchicalInterval(pos, neg, groups, groups, groups, groups, 200, 42)
	s.Equal(wantLo, lo)
	s.Equal(wantHi, hi)
}

func (s *AUCSuite) TestHierarchicalIntervalPinsSeparatedFamilies() {
	pos := []float64{3, 4, 5, 6}
	neg := []float64{0, 1, 0, 1}
	family := []int{0, 0, 1, 1}
	recording := []int{0, 1, 2, 3}
	lo, hi := HierarchicalInterval(pos, neg, family, recording, family, recording, 100, 3)
	s.Equal(1.0, lo)
	s.Equal(1.0, hi)
}

func (s *AUCSuite) TestAUCIntervalWithoutRoundsIsThePointEstimate() {
	pos := []float64{0.3, 0.6, 0.45}
	neg := []float64{0.1, 0.4, 0.5}
	for _, rounds := range []int{0, -3} {
		s.Run(fmt.Sprintf("%d rounds", rounds), func() {
			lo, hi := AUCInterval(pos, neg, ungrouped(3, 0), ungrouped(3, 0), rounds, 7)
			s.Equal(AUC(pos, neg), lo)
			s.Equal(AUC(pos, neg), hi)
		})
	}
}

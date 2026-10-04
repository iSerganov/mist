package steganalysis

import (
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
			lo, hi := AUCInterval(tc.pos, tc.neg, 500, 7)
			s.Equal(tc.want, lo)
			s.Equal(tc.want, hi)
		})
	}
}

func (s *AUCSuite) TestAUCIntervalBracketsEstimate() {
	pos := []float64{0.3, 0.6, 0.45, 0.95, 0.7, 0.5, 0.85, 0.75}
	neg := []float64{0.1, 0.4, 0.35, 0.8, 0.65, 0.2, 0.9, 0.55}
	lo, hi := AUCInterval(pos, neg, 500, 7)
	auc := AUC(pos, neg)
	s.LessOrEqual(lo, auc)
	s.GreaterOrEqual(hi, auc)
	s.Less(lo, hi)
}

func (s *AUCSuite) TestAUCIntervalIsDeterministicPerSeed() {
	pos := []float64{0.3, 0.6, 0.45, 0.95, 0.7}
	neg := []float64{0.1, 0.4, 0.35, 0.8, 0.65}
	lo1, hi1 := AUCInterval(pos, neg, 200, 42)
	lo2, hi2 := AUCInterval(pos, neg, 200, 42)
	s.Equal(lo1, lo2)
	s.Equal(hi1, hi2)
}

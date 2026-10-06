package steganalysis

import (
	"testing"

	"github.com/stretchr/testify/suite"
)

type ReferenceSuite struct {
	suite.Suite
}

func TestReferenceSuite(t *testing.T) {
	suite.Run(t, &ReferenceSuite{})
}

func (s *ReferenceSuite) TestLikelihoodRatio() {
	apart := Reference{Clean: []float64{0.1, 0.12, 0.09, 0.11, 0.1}, Stego: []float64{0.9, 0.88, 0.91, 0.9, 0.89}}
	same := Reference{Clean: []float64{0.4, 0.5, 0.6}, Stego: []float64{0.4, 0.5, 0.6}}
	constant := Reference{Clean: []float64{0, 0, 0}, Stego: []float64{0, 0, 0}}
	tests := []struct {
		title  string
		ref    Reference
		score  float64
		lo, hi float64
	}{
		{"a score among the stego files favours stego up to the cap", apart, 0.9, MaxLR, MaxLR},
		{"a score among the clean files favours clean down to the cap", apart, 0.1, 1.0 / MaxLR, 1.0 / MaxLR},
		{"identical populations carry no evidence", same, 0.5, 0.999, 1.001},
		{"constant identical populations carry no evidence", constant, 0, 0.999, 1.001},
		{"a score far from both populations carries no evidence", apart, 50, 0.999, 1.001},
		{"an empty reference carries no evidence", Reference{}, 0.3, 1, 1},
	}
	for _, tc := range tests {
		s.Run(tc.title, func() {
			lr := tc.ref.LikelihoodRatio(tc.score)
			s.GreaterOrEqual(lr, tc.lo)
			s.LessOrEqual(lr, tc.hi)
		})
	}
}

func (s *ReferenceSuite) TestPosterior() {
	tests := []struct {
		title  string
		lrs    []float64
		prior  float64
		lo, hi float64
	}{
		{"no evidence leaves the prior", nil, 0.5, 0.5, 0.5},
		{"neutral evidence leaves the prior", []float64{1, 1, 1}, 0.5, 0.4999, 0.5001},
		{"one strong ratio moves the posterior", []float64{MaxLR}, 0.5, 0.95, 0.96},
		{"repeating the same evidence does not compound it", []float64{MaxLR, MaxLR, MaxLR}, 0.5, 0.95, 0.96},
		{"opposite evidence cancels", []float64{4, 0.25}, 0.5, 0.4999, 0.5001},
		{"evidence against stego lowers the posterior", []float64{0.1, 0.2}, 0.5, 0.1, 0.15},
	}
	for _, tc := range tests {
		s.Run(tc.title, func() {
			p := Posterior(tc.lrs, tc.prior)
			s.GreaterOrEqual(p, tc.lo)
			s.LessOrEqual(p, tc.hi)
		})
	}
}

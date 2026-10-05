package steganalysis

import (
	"testing"

	"github.com/stretchr/testify/suite"
)

type PermutationSuite struct {
	suite.Suite
}

func TestPermutationSuite(t *testing.T) {
	suite.Run(t, &PermutationSuite{})
}

func (s *PermutationSuite) TestPairedPermutationIsOneWhenScoresMatch() {
	pos := []float64{0.2, 0.4, 0.6, 0.8}
	neg := []float64{0.2, 0.4, 0.6, 0.8}
	family := []int{0, 1, 2, 3}
	s.Equal(1.0, PairedPermutationP(pos, neg, family, 50, 1))
}

func (s *PermutationSuite) TestPairedPermutationRejectsAFixedSeparation() {
	pos := []float64{1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1}
	neg := []float64{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0}
	family := []int{0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11}
	s.Less(PairedPermutationP(pos, neg, family, 199, 2), 0.05)
}

func (s *PermutationSuite) TestRefitPermutationIsDeterministic() {
	var x [][]float64
	var y []bool
	var g []int
	for i := range 6 {
		x = append(x, []float64{float64(i), -1}, []float64{float64(i), 1})
		y = append(y, false, true)
		g = append(g, i, i)
	}
	s.Equal(RefitPermutationP(x, y, g, 3, 4, 9), RefitPermutationP(x, y, g, 3, 4, 9))
}

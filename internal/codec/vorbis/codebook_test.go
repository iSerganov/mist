package vorbis

import (
	"testing"

	"github.com/stretchr/testify/suite"
)

type CodebookSuite struct {
	suite.Suite
}

func TestCodebookSuite(t *testing.T) {
	suite.Run(t, &CodebookSuite{})
}

// flipBook is a one-dimensional codebook whose entry vectors are easy to
// reason about: entries 0–3 share a code length, entry 4 stands alone.
func flipBook() *codebook {
	cb := &codebook{
		dim:     1,
		entries: 5,
		lens:    []uint8{2, 2, 2, 2, 3},
		used:    []int{0, 1, 2, 3, 4},
		lookup:  1,
		vecs:    [][]float64{{0}, {1}, {5}, {2}, {0.5}},
	}
	cb.buildFlips()
	return cb
}

func (s *CodebookSuite) TestFlipCostIsTheDistanceToTheSubstitute() {
	cb := flipBook()
	tests := []struct {
		title     string
		entry     int
		wantCost  float64
		wantOK    bool
		wantEntry int
	}{
		{"nearest odd vector to 0 is entry 1", 0, 1, true, 1},
		{"entry 2 sits far from every odd vector", 2, 9, true, 3},
		{"nearest even vector to 3 is entry 0", 3, 4, true, 0},
		{"no other entry of its length", 4, 0, false, -1},
	}
	for _, tc := range tests {
		s.Run(tc.title, func() {
			cost, ok := cb.flipCost(tc.entry)
			s.Equal(tc.wantOK, ok)
			s.InDelta(tc.wantCost, cost, 1e-12)
			if ok {
				s.Equal(tc.wantEntry, substitute(cb, tc.entry, tc.entry^1))
			}
		})
	}
}

func (s *CodebookSuite) TestBookWithoutVectorsCostsOneUnitPerLegalFlip() {
	cb := &codebook{dim: 1, entries: 3, lens: []uint8{2, 2, 3}, used: []int{0, 1, 2}}
	cb.buildFlips()
	cost, ok := cb.flipCost(0)
	s.True(ok)
	s.Equal(1.0, cost)
	_, ok = cb.flipCost(2)
	s.False(ok)
}

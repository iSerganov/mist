package stego

import (
	"math/rand/v2"
	"testing"

	"github.com/stretchr/testify/suite"
)

type STCSuite struct {
	suite.Suite
}

func TestSTCSuite(t *testing.T) {
	suite.Run(t, &STCSuite{})
}

func randomBits(rng *rand.Rand, n int) []uint8 {
	out := make([]uint8, n)
	for i := range out {
		out[i] = uint8(rng.IntN(2))
	}
	return out
}

func uniform(n int) []float32 {
	out := make([]float32, n)
	for i := range out {
		out[i] = 1
	}
	return out
}

func applyFlips(cover []uint8, flips []int) []uint8 {
	y := append([]uint8(nil), cover...)
	for _, j := range flips {
		y[j] ^= 1
	}
	return y
}

func (s *STCSuite) TestSyndromeOfEmbeddedCoverIsTheMessage() {
	tests := []struct {
		title string
		m, w  int
	}{
		{"one cover element per bit", 40, 1},
		{"two per bit", 40, 2},
		{"fewer bits than the code height", 3, 50},
		{"a single bit", 1, 50},
		{"Mist's own rate", 500, 50},
	}
	rng := rand.New(rand.NewPCG(7, 7))
	for _, tc := range tests {
		s.Run(tc.title, func() {
			code := newSTC([]byte("position key"), tc.w)
			cover := randomBits(rng, tc.m*tc.w)
			msg := randomBits(rng, tc.m)
			flips, err := code.embed(cover, uniform(len(cover)), msg)
			s.Require().NoError(err)
			s.Equal(msg, code.syndrome(applyFlips(cover, flips), tc.m))
		})
	}
}

// The point of the code: LSB matching changes half the positions it
// writes, one per two bits. The trellis code should need well under a
// third of that at a 1/50 rate.
func (s *STCSuite) TestChangesFarFewerValuesThanLSBMatching() {
	rng := rand.New(rand.NewPCG(1, 2))
	const m, w = 4000, 50
	code := newSTC([]byte("position key"), w)
	cover := randomBits(rng, m*w)
	flips, err := code.embed(cover, uniform(len(cover)), randomBits(rng, m))
	s.Require().NoError(err)
	s.Less(float64(len(flips))/m, 0.16, "changes per message bit")
}

func (s *STCSuite) TestRoutesAroundExpensiveValues() {
	rng := rand.New(rand.NewPCG(3, 4))
	const m, w = 1000, 50
	code := newSTC([]byte("position key"), w)
	cover := randomBits(rng, m*w)
	cost := uniform(len(cover))
	for j := range cost {
		if j%4 == 0 {
			cost[j] = wetCost
		}
	}
	msg := randomBits(rng, m)
	flips, err := code.embed(cover, cost, msg)
	s.Require().NoError(err)
	for _, j := range flips {
		s.NotZero(j%4, "flipped a value marked wet at %d", j)
	}
	s.Equal(msg, code.syndrome(applyFlips(cover, flips), m))
}

// Row 0 of the syndrome is the parity of block 0 alone, so a message bit
// that disagrees with it forces a flip there. With block 0 all wet, every
// path carries wetCost; the cheap flips after it must still be chosen as
// carefully as when block 0 costs one unit.
func (s *STCSuite) TestAWetFlipDoesNotBlurTheCostsAfterIt() {
	rng := rand.New(rand.NewPCG(5, 6))
	const m, w = 400, 50
	code := newSTC([]byte("position key"), w)
	cover := randomBits(rng, m*w)
	msg := randomBits(rng, m)
	var parity uint8
	for _, b := range cover[:w] {
		parity ^= b
	}
	msg[0] = parity ^ 1
	cost := make([]float32, len(cover))
	for j := range cost {
		cost[j] = minCost + rng.Float32()*0.01
	}
	tail := func(block0 float32) float64 {
		c := append([]float32(nil), cost...)
		for j := range w {
			c[j] = block0
		}
		flips, err := code.embed(cover, c, msg)
		s.Require().NoError(err)
		s.Require().Equal(msg, code.syndrome(applyFlips(cover, flips), m))
		var sum float64
		for _, j := range flips {
			if j >= w {
				sum += float64(c[j])
			}
		}
		return sum
	}
	s.InDelta(tail(1), tail(wetCost), 1e-6)
}

func (s *STCSuite) TestMatrixDependsOnTheKey() {
	s.Equal(newSTC([]byte("a"), 50), newSTC([]byte("a"), 50))
	s.NotEqual(newSTC([]byte("a"), 50), newSTC([]byte("b"), 50))
}

func (s *STCSuite) TestRejectsMismatchedLengths() {
	code := newSTC([]byte("k"), 10)
	_, err := code.embed(make([]uint8, 19), uniform(19), make([]uint8, 2))
	s.ErrorIs(err, errNoSolution)
}

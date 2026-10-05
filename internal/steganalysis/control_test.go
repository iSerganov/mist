package steganalysis

import (
	"crypto/ecdh"
	"crypto/rand"
	"math"
	"testing"

	"github.com/stretchr/testify/suite"
)

type ControlSuite struct {
	suite.Suite
}

func TestControlSuite(t *testing.T) {
	suite.Run(t, &ControlSuite{})
}

func (s *ControlSuite) TestRichVectorIsFrozen() {
	v := shortSmooth(4096, 1)
	got := Rich(v, 1)
	s.Len(got, len(RichNames()))
	for i, x := range got {
		s.False(math.IsNaN(x), RichNames()[i])
		s.False(math.IsInf(x, 0), RichNames()[i])
	}
	s.Equal(got, RichPlanar(v, 1))
	planar := append(append([]int32(nil), v...), shortSmooth(4096, 2)...)
	side := RichPlanar(planar, 2)
	s.NotZero(side[len(side)-1])
}

func (s *ControlSuite) TestFrozenPrimariesDetectTheirControls() {
	s.Run("hcf sees high-rate matching", func() {
		c := peaked(newRand(1))
		s.Greater(HCF(matchLSB(c, 0.5, newRand(4))), HCF(c)+0.01)
	})
	s.Run("classifier sees lsb replacement", func() {
		x, y, g := embeddedRows(peakedShort, replaceLSB, 1, Features)
		s.Greater(splitAUC(CrossValidate(x, y, g, 4), y), 0.8)
	})
	s.Run("markov sees high-rate matching", func() {
		x, y, g := embeddedRows(lowAmplitude, matchLSB, 1, Markov)
		s.Greater(splitAUC(CrossValidate(x, y, g, 4), y), 0.8)
	})
	s.Run("rich sees high-rate matching", func() {
		x, y, g := embeddedRows(lowAmplitude, matchLSB, 1, func(v []int32) []float64 { return Rich(v, 1) })
		s.Greater(splitAUC(CrossValidate(x, y, g, 4), y), 0.8)
	})
	s.Run("fld stumps and subspace see high-rate matching", func() {
		x, y, _ := embeddedRows(lowAmplitude, matchLSB, 1, func(v []int32) []float64 { return Rich(v, 1) })
		s.Greater(heldOutAUC(x, y, func(tx [][]float64, ty []bool) scorer { return TrainFLD(tx, ty) }), 0.8)
		s.Greater(heldOutAUC(x, y, func(tx [][]float64, ty []bool) scorer { return TrainStumps(tx, ty, 20) }), 0.8)
		s.Greater(heldOutAUC(x, y, func(tx [][]float64, ty []bool) scorer {
			return TrainSubspace(tx, ty, 6, 8, 1)
		}), 0.8)
	})
	s.Run("raw x25519 public key", func() {
		k, err := ecdh.X25519().GenerateKey(rand.Reader)
		s.Require().NoError(err)
		s.True(HonestX25519(k.PublicKey().Bytes()))
		noise := make([]byte, 32)
		for i := range noise {
			noise[i] = 0xff
		}
		s.False(HonestX25519(noise))
	})
	s.Run("selection channel sees lsb replacement at known positions", func() {
		c := shortSmooth(4000, 5)
		at := make([]int, 0, len(c)/10)
		for i := 0; i < len(c); i += 10 {
			at = append(at, i)
		}
		s.Less(ParityGap(c, at), 0.08)
		forced := append([]int32(nil), c...)
		for _, i := range at {
			forced[i] |= 1
		}
		s.Greater(ParityGap(forced, at), 0.2)
	})
	s.Run("unmasked length", func() {
		s.Equal(1.0, PlausibleLength([]byte{0, 0, 1, 0}))
		s.Equal(0.0, PlausibleLength([]byte{0xff, 0xff, 0xff, 0xff}))
	})
	s.Run("no filler", func() {
		head := shortSmooth(64, 6)
		s.Equal(1.0, DeadTail(append(head, make([]int32, 64)...)))
		s.Equal(0.0, DeadTail(shortSmooth(256, 7)))
	})
	s.Run("random index neighbor", func() {
		c := peakedShort(8000, 8)
		s.Greater(HCF(matchLSB(c, 0.5, newRand(9))), HCF(c)+0.01)
	})
	s.Run("mismatched metadata", func() {
		var x [][]float64
		var y []bool
		var g []int
		for i := range 8 {
			x = append(x, []float64{1000}, []float64{2000 + float64(i)})
			y = append(y, false, true)
			g = append(g, i, i+8)
		}
		s.Greater(splitAUC(CrossValidate(x, y, g, 4), y), 0.9)
	})
}

func (s *ControlSuite) TestChangedFractionIsTheCoverOracle() {
	c := shortSmooth(1000, 10)
	s.Zero(ChangedFraction(c, c))
	s.InDelta(0.25, ChangedFraction(c, matchLSB(c, 0.5, newRand(11))), 0.08)
}

type scorer interface {
	Score([]float64) float64
}

func embeddedRows(carrier func(int, uint64) []int32, embed embedFunc, rate float64, feat func([]int32) []float64) ([][]float64, []bool, []int) {
	const n = 24
	x := make([][]float64, n)
	y := make([]bool, n)
	g := make([]int, n)
	for i := range n {
		c := carrier(8192, uint64(i+1))
		if i%2 == 1 {
			c = embed(c, rate, newRand(uint64(i+30)))
			y[i] = true
		}
		x[i] = feat(c)
		g[i] = i
	}
	return x, y, g
}

func heldOutAUC(x [][]float64, y []bool, fit func([][]float64, []bool) scorer) float64 {
	var pos, neg []float64
	for i := range x {
		var tx [][]float64
		var ty []bool
		for j := range x {
			if j == i {
				continue
			}
			tx = append(tx, x[j])
			ty = append(ty, y[j])
		}
		p := fit(tx, ty).Score(x[i])
		if y[i] {
			pos = append(pos, p)
			continue
		}
		neg = append(neg, p)
	}
	return AUC(pos, neg)
}

func splitAUC(scores []float64, y []bool) float64 {
	var pos, neg []float64
	for i, p := range scores {
		if y[i] {
			pos = append(pos, p)
			continue
		}
		neg = append(neg, p)
	}
	return AUC(pos, neg)
}

func lowAmplitude(n int, seed uint64) []int32 {
	rng := newRand(seed)
	v := make([]int32, n)
	for i := range v {
		t := float64(i)
		v[i] = int32(math.Round(8*math.Sin(t/17) + rng.NormFloat64()))
	}
	return v
}

func shortSmooth(n int, seed uint64) []int32 {
	rng := newRand(seed)
	v := make([]int32, n)
	for i := range v {
		t := float64(i)
		v[i] = int32(math.Round(800*math.Sin(t/37) + 300*math.Sin(t/11) + rng.NormFloat64()*4))
	}
	return v
}

func peakedShort(n int, seed uint64) []int32 {
	rng := newRand(seed)
	v := make([]int32, n)
	for i := range v {
		v[i] = int32(math.Round(rng.ExpFloat64() * 3 * float64(1-2*rng.IntN(2))))
	}
	return v
}

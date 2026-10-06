package stego

import (
	"math"
	"math/rand/v2"
)

// A ±1 in a smooth stretch is a larger fraction of the local residual than
// the same step in noise, which is what a warden's predictor sees. The
// floor and the ratio cap keep that preference from emptying the smooth
// part of the frame: a cost of zero would be a selection channel.
const (
	costFloor    = 0.25
	costRatio    = 8
	ditherAmp    = 0.05
	costWin      = 8
	residualTemp = 32
)

func (c *sampleCover) choose(p int, v int32, mayQuiet bool) int32 {
	up, down := c.allowed(v, v+1, mayQuiet), c.allowed(v, v-1, mayQuiet)
	switch {
	case up && !down:
		return v + 1
	case down && !up:
		return v - 1
	case !up && !down:
		return v
	}
	// Weight each legal step by the histogram flow the receiver's marginal
	// already implied, discounted by how much that step grows the residual.
	// Taking the minimum every time walks the frame toward its peak.
	wu := math.Sqrt(float64(c.hist[v+1])) / (1 + c.baseScore(p, v, v+1)/residualTemp)
	wd := math.Sqrt(float64(c.hist[v-1])) / (1 + c.baseScore(p, v, v-1)/residualTemp)
	if wu+wd == 0 || rand.Float64()*(wu+wd) < wu {
		return v + 1
	}
	return v - 1
}

func (c *sampleCover) positionCost(p int, v int32, may bool) float64 {
	best := math.Inf(1)
	if c.allowed(v, v+1, may) {
		best = math.Min(best, c.baseScore(p, v, v+1))
	}
	if c.allowed(v, v-1, may) {
		best = math.Min(best, c.baseScore(p, v, v-1))
	}
	if math.IsInf(best, 1) {
		best = 1
	}
	ch := len(c.s.Planes)
	t, k := p/ch, p%ch
	best += c.edgePenalty(t, k) + clipPenalty(v, c.s.lo(), c.s.hi())
	return best / (1 + c.mad(t, k))
}

// baseScore is how much a step to `to` grows the local residual, plus the
// stereo disturbance. Dividing by the neighbourhood's own residual, in
// positionCost, is the tonality term: a small residual is a tone.
func (c *sampleCover) baseScore(p int, v, to int32) float64 {
	ch := len(c.s.Planes)
	t, k := p/ch, p%ch
	score := math.Max(0, c.d1Grow(t, k, v, to)) + math.Max(0, c.d2Grow(t, k, v, to))
	return score + c.stereoPenalty(t, k, v, to)
}

func (c *sampleCover) d1Grow(t, k int, v, to int32) float64 {
	var g float64
	if prev, ok := c.atCh(t-1, k); ok {
		g += absGrow(v-prev, to-prev)
	}
	if next, ok := c.atCh(t+1, k); ok {
		g += absGrow(next-v, next-to)
	}
	return g
}

func (c *sampleCover) d2Grow(t, k int, v, to int32) float64 {
	prev, okP := c.atCh(t-1, k)
	next, okN := c.atCh(t+1, k)
	if !okP || !okN {
		return 0
	}
	return absGrow(prev-2*v+next, prev-2*to+next)
}

func (c *sampleCover) stereoPenalty(t, k int, v, to int32) float64 {
	if len(c.s.Planes) < 2 {
		return 0
	}
	other, ok := c.atCh(t, 1-k)
	if !ok {
		return 0
	}
	side, neu := abs32(v-other), abs32(to-other)
	g := 0.0
	if neu > side {
		g = float64(neu - side)
	}
	if side == 0 {
		g += 0.5
	}
	return 0.25 * g
}

func (c *sampleCover) mad(t, k int) float64 {
	var sum float64
	var n int
	for d := -costWin; d <= costWin; d++ {
		if d == 0 {
			continue
		}
		a, oka := c.atCh(t+d, k)
		b, okb := c.atCh(t+d-1, k)
		if !oka || !okb {
			continue
		}
		sum += math.Abs(float64(a - b))
		n++
	}
	if n == 0 {
		return 0
	}
	return sum / float64(n)
}

func (c *sampleCover) edgePenalty(t, k int) float64 {
	ch := len(c.s.Planes)
	best := silenceRun
	for d := 1; d < silenceRun; d++ {
		for _, u := range []int{t - d, t + d} {
			if u < 0 || u >= c.s.N {
				if d < best {
					best = d
				}
				continue
			}
			if c.s.quiet(u*ch+k) && d < best {
				best = d
			}
		}
	}
	if best >= silenceRun {
		return 0
	}
	return float64(silenceRun-best) / float64(silenceRun)
}

func clipPenalty(v, lo, hi int32) float64 {
	dist := v - lo
	if hi-v < dist {
		dist = hi - v
	}
	if dist >= 8 {
		return 0
	}
	return float64(8-dist) / 8
}

func (c *sampleCover) atCh(t, k int) (int32, bool) {
	if t < 0 || t >= c.s.N || k < 0 || k >= len(c.s.Planes) {
		return 0, false
	}
	return c.s.At(t*len(c.s.Planes) + k), true
}

func absGrow(old, neu int32) float64 {
	return math.Abs(float64(neu)) - math.Abs(float64(old))
}

func regularizeCost(raw float64, key []byte, pos int) float32 {
	if raw < costFloor {
		raw = costFloor
	}
	hi := costFloor * costRatio
	if raw > hi {
		raw = hi
	}
	if len(key) > 0 {
		raw *= 1 + ditherAmp*float64(ditherUnit(key, pos))
	}
	if raw < costFloor {
		raw = costFloor
	}
	if raw > hi {
		raw = hi
	}
	return float32(raw)
}

func ditherUnit(key []byte, pos int) float32 {
	h := uint32(2166136261)
	for _, b := range key {
		h ^= uint32(b)
		h *= 16777619
	}
	h ^= uint32(pos)
	h *= 16777619
	return float32(int(h%10001)-5000) / 5000
}

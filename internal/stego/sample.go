package stego

import (
	"math"
	"math/rand/v2"
)

// Samples is one stego frame of planar float PCM, seen as a flat run of
// integer samples. A lossless codec reproduces those integers exactly,
// so a bit written into a sample's LSB is still there after the file has
// been encoded and decoded again — no bitstream surgery required.
//
// Position k maps to channel k%len(Planes), sample Off+k/len(Planes):
// interleaved order. Embed and Listen both derive their positions from
// that one rule, which is what lets them agree without sharing a buffer.
//
// Scale is the encoder's integer grid (av.SampleScale). Values are read
// and written on that grid, so the float planes here always hold samples
// the encoder can represent exactly.
type Samples struct {
	Planes [][]float32
	Off    int
	N      int
	Scale  float32
}

// Len is every sample in the window, silent or not.
func (s Samples) Len() int {
	if s.N <= 0 || len(s.Planes) == 0 {
		return 0
	}
	return s.N * len(s.Planes)
}

// At returns the sample at position i on the encoder's integer grid.
func (s Samples) At(i int) int32 {
	p, k := s.locate(i)
	return s.clamp(int32(math.Round(float64(p[k] * s.Scale))))
}

// Set writes v back. A value outside the grid moves by two rather than
// one, because the encoder would clip it and a clip would take the bit
// just embedded with it.
func (s Samples) Set(i int, v int32) {
	p, k := s.locate(i)
	p[k] = float32(s.clamp(v)) / s.Scale
}

// Capacity is the payload bytes this frame holds at the constant density.
func (s Samples) Capacity() int { return slots(newSampleCover(s).Len()) / 8 }

func (s Samples) locate(i int) ([]float32, int) {
	ch := len(s.Planes)
	return s.Planes[i%ch], s.Off + i/ch
}

func (s Samples) clamp(v int32) int32 {
	hi, lo := s.hi(), s.lo()
	for v > hi {
		v -= 2
	}
	for v < lo {
		v += 2
	}
	return v
}

func (s Samples) hi() int32 { return int32(s.Scale) - 1 }
func (s Samples) lo() int32 { return -int32(s.Scale) }

// Silence detection. A plain encoder leaves digital silence — and the
// zero padding at the end of a file — exactly as it found it, so ±1
// changes there are the clearest tell a lossless output has. A run of at
// least silenceRun samples in one channel, none beyond ±silenceFloor, is
// silence and is left out of the carrier; so is a quiet run of any length
// at the edge of the window, which may be the tail of a longer one.
const (
	silenceRun   = 32
	silenceFloor = 2
)

// sampleCover is the carrier over one window's non-silent samples.
//
// The receiver must find the same runs silent, so no ±1 may create,
// lengthen or join one: a sample just above the floor moves towards zero
// only when the quiet run that would make stays short and clear of the
// window's edges. Silent samples themselves are never touched.
type sampleCover struct {
	s      Samples
	elig   []int
	posKey []byte
	// hist is the frame's histogram before embedding, which step steers by.
	hist map[int32]int
}

func newSampleCover(s Samples) *sampleCover {
	c := &sampleCover{s: s, hist: map[int32]int{}}
	ch := len(s.Planes)
	if s.Len() == 0 {
		return c
	}
	silent := make([]bool, s.Len())
	for k := range ch {
		for t := 0; t < s.N; {
			if !s.quiet(t*ch + k) {
				t++
				continue
			}
			end := t
			for end < s.N && s.quiet(end*ch+k) {
				end++
			}
			if end-t >= silenceRun || t == 0 || end == s.N {
				for u := t; u < end; u++ {
					silent[u*ch+k] = true
				}
			}
			t = end
		}
	}
	for i, quiet := range silent {
		if !quiet {
			c.elig = append(c.elig, i)
			c.hist[s.At(i)]++
		}
	}
	return c
}

func (s Samples) quiet(i int) bool { return abs32(s.At(i)) <= silenceFloor }

func (c *sampleCover) Len() int       { return len(c.elig) }
func (c *sampleCover) At(i int) int32 { return c.s.At(c.elig[i]) }

func (c *sampleCover) Cost(i int) float32 {
	p := c.elig[i]
	v := c.s.At(p)
	return regularizeCost(c.positionCost(p, v, c.mayStep(p, v)), c.posKey, p)
}

func (c *sampleCover) mayStep(p int, v int32) bool {
	return abs32(v) != silenceFloor+1 || c.mayQuiet(p)
}

func (c *sampleCover) Flip(i int) {
	p := c.elig[i]
	v := c.s.At(p)
	to := c.choose(p, v, c.mayStep(p, v))
	if h := c.hist[v]; h > 1 {
		c.hist[v] = h - 1
	} else {
		delete(c.hist, v)
	}
	c.hist[to]++
	c.s.Set(p, to)
}

// mayQuiet reports whether sample p could become quiet without making a
// silent run: the quiet run it would join, counting itself, must stay
// shorter than silenceRun and touch neither edge of the window.
func (c *sampleCover) mayQuiet(p int) bool {
	ch := len(c.s.Planes)
	t, k := p/ch, p%ch
	if t == 0 || t == c.s.N-1 {
		return false
	}
	run := 1
	for u := t - 1; c.s.quiet(u*ch + k); u-- {
		if u == 0 {
			return false
		}
		run++
	}
	for u := t + 1; c.s.quiet(u*ch + k); u++ {
		if u == c.s.N-1 {
			return false
		}
		run++
	}
	return run < silenceRun
}

// allowed reports whether v may move to to: on the grid, and not into
// quiet when mayQuiet says that would make silence.
func (c *sampleCover) allowed(v, to int32, mayQuiet bool) bool {
	if to > c.s.hi() || to < c.s.lo() {
		return false
	}
	return mayQuiet || abs32(v) <= silenceFloor || abs32(to) > silenceFloor
}

// step picks v+1 or v-1. Either carries the bit; the choice decides the
// histogram. Random ±1 moves samples off a peak faster than they come
// back, flattening it, which is what histogram detectors look for. Going
// up with probability √h(v+1) / (√h(v+1)+√h(v-1)) balances the flow
// across every boundary exactly where the histogram is locally geometric,
// as audio's is everywhere but its peak, so on average it stays put.
func (c *sampleCover) step(v int32, mayQuiet bool) int32 {
	up, down := c.allowed(v, v+1, mayQuiet), c.allowed(v, v-1, mayQuiet)
	if up && down {
		a, b := math.Sqrt(float64(c.hist[v+1])), math.Sqrt(float64(c.hist[v-1]))
		up = rand.Float64()*(a+b) < a || a+b == 0 && rand.IntN(2) == 0
	}
	if up {
		return v + 1
	}
	return v - 1
}

func abs32(n int32) int32 {
	if n < 0 {
		return -n
	}
	return n
}

// ApplySamples embeds bits into one frame of PCM. Nil bits fill the frame
// with filler at the same rate, the way an empty residue frame is.
//
// Every sample is first written back onto the grid. The receiver reads
// the LSB of every sample the code covers, not only the ones it changed,
// and an off-grid sample — anything decoded from a lossy carrier — would
// otherwise be rounded or clipped by the encoder in a way At cannot see.
func ApplySamples(s Samples, posKey []byte, bits Bits) error {
	for i := range s.Len() {
		s.Set(i, s.At(i))
	}
	cover := newSampleCover(s)
	cover.posKey = posKey
	return place(cover, posKey, bits)
}

// RecoverSamples reads the constant-density bit string from one frame.
func RecoverSamples(s Samples, posKey []byte) (Bits, error) {
	return lift(newSampleCover(s), posKey)
}

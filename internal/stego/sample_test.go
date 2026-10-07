package stego

import (
	"math"
	"math/rand/v2"
	"testing"

	"github.com/stretchr/testify/suite"
)

type SampleSuite struct {
	suite.Suite
}

func TestSampleSuite(t *testing.T) {
	suite.Run(t, &SampleSuite{})
}

const testScale = 1 << 15

var testKey = []byte("position key, 32 bytes long.....")

// planes builds a stereo window of n frames whose values come from f,
// already on the 16-bit grid.
func planes(n int, f func(i int) int32) [][]float32 {
	out := [][]float32{make([]float32, n), make([]float32, n)}
	for i := range n {
		out[0][i] = float32(f(i)) / testScale
		out[1][i] = float32(-f(i)) / testScale
	}
	return out
}

func noise(rng *rand.Rand, amp int) func(int) int32 {
	return func(int) int32 { return int32(rng.IntN(2*amp+1) - amp) }
}

func values(s Samples) []int32 {
	out := make([]int32, s.Len())
	for i := range out {
		out[i] = s.At(i)
	}
	return out
}

func (s *SampleSuite) TestRoundTrip() {
	rng := rand.New(rand.NewPCG(1, 1))
	tests := []struct {
		title string
		f     func(int) int32
	}{
		{"loud noise", noise(rng, 3000)},
		{"quiet noise just above the floor", noise(rng, silenceFloor+1)},
		{"silence broken by bursts", func(i int) int32 {
			if (i/2000)%2 == 0 {
				return 0
			}
			return int32(rng.IntN(4001) - 2000)
		}},
	}
	for _, tc := range tests {
		s.Run(tc.title, func() {
			sm := Samples{Planes: planes(44100, tc.f), N: 44100, Scale: testScale}
			payload := []byte("a message long enough to span several bytes")
			s.Require().NoError(ApplySamples(sm, testKey, payload))
			got, err := RecoverSamples(sm, testKey)
			s.Require().NoError(err)
			s.Equal(payload, []byte(got[:len(payload)]))
		})
	}
}

// A carrier decoded from a lossy file is off the grid, and the encoder
// rounds such a sample its own way. The receiver reads every covered
// sample, so the sender must have put each one where the encoder will.
func (s *SampleSuite) TestOffGridCarrierIsSnappedBeforeEmbedding() {
	rng := rand.New(rand.NewPCG(2, 2))
	n := 22050
	p := [][]float32{make([]float32, n), make([]float32, n)}
	for _, pl := range p {
		for i := range pl {
			pl[i] = (rng.Float32()*2 - 1) * 1.01 // a little past full scale too
		}
	}
	sm := Samples{Planes: p, N: n, Scale: testScale}
	s.Require().NoError(ApplySamples(sm, testKey, []byte("x")))
	for _, pl := range p {
		for i, v := range pl {
			g := v * testScale
			s.Require().Equal(float32(int32(g)), g, "sample %d left off the grid", i)
			s.Require().LessOrEqual(g, float32(testScale-1))
			s.Require().GreaterOrEqual(g, float32(-testScale))
		}
	}
}

// The receiver re-derives the silent runs from the stego samples, so
// embedding must leave them exactly where they were. The hard signals
// are the ones crowded with samples just above the floor and with quiet
// runs one short of silence, where a single step down would make one.
func (s *SampleSuite) TestSilenceIsUntouchedAndReadsTheSameAfterEmbedding() {
	tests := []struct {
		title string
		f     func(rng *rand.Rand) func(int) int32
	}{
		{"loud and silent stretches", func(rng *rand.Rand) func(int) int32 {
			return func(i int) int32 {
				if (i/1000)%2 == 0 {
					return int32(rng.IntN(2*silenceFloor+1) - silenceFloor)
				}
				return int32(rng.IntN(201) - 100)
			}
		}},
		{"hovering around the floor", func(rng *rand.Rand) func(int) int32 {
			return func(int) int32 { return int32(rng.IntN(2*silenceFloor+3) - silenceFloor - 1) }
		}},
		{"quiet runs one short of silence between single loud samples", func(*rand.Rand) func(int) int32 {
			return func(i int) int32 {
				if i%silenceRun == 0 {
					return silenceFloor + 1
				}
				return 0
			}
		}},
		{"quiet runs exactly silence long", func(*rand.Rand) func(int) int32 {
			return func(i int) int32 {
				if i%(silenceRun+1) == 0 {
					return -silenceFloor - 1
				}
				return 1
			}
		}},
	}
	for _, tc := range tests {
		s.Run(tc.title, func() {
			for seed := range 4 {
				rng := rand.New(rand.NewPCG(uint64(seed), 3))
				n := 44100
				sm := Samples{Planes: planes(n, tc.f(rng)), N: n, Scale: testScale}
				before := values(sm)
				cover := newSampleCover(sm).elig
				s.Require().NotEmpty(cover)

				s.Require().NoError(ApplySamples(sm, []byte{byte(seed)}, nil))
				inCover := make([]bool, sm.Len())
				for _, i := range cover {
					inCover[i] = true
				}
				changed := 0
				for i, v := range values(sm) {
					if v != before[i] {
						changed++
						s.Require().True(inCover[i], "silent sample %d changed", i)
					}
				}
				s.Positive(changed, "filler perturbed nothing")
				s.Require().Equal(cover, newSampleCover(sm).elig, "receiver would see a different cover")
			}
		})
	}
}

func (s *SampleSuite) TestStepsTowardsZeroOnlyWhereThatMakesNoSilence() {
	loud := int32(silenceFloor + 1)
	tests := []struct {
		title string
		run   []int32 // one channel
		at    int
		want  bool
	}{
		{"between loud neighbours", []int32{100, loud, 100, 100}, 1, true},
		{"joining a short quiet run", []int32{100, loud, 0, 0, 100, 100}, 1, true},
		{"at the start of the window", []int32{loud, 100, 100}, 0, false},
		{"joining a quiet run that reaches the end", []int32{100, loud, 0, 0}, 1, false},
		{"joining a run to silenceRun long", append([]int32{100, loud}, append(make([]int32, silenceRun-1), 100, 100)...), 1, false},
		{"joining a run to just short of it", append([]int32{100, loud}, append(make([]int32, silenceRun-2), 100, 100)...), 1, true},
	}
	for _, tc := range tests {
		s.Run(tc.title, func() {
			p := make([]float32, len(tc.run))
			for i, v := range tc.run {
				p[i] = float32(v) / testScale
			}
			c := newSampleCover(Samples{Planes: [][]float32{p}, N: len(p), Scale: testScale})
			s.Equal(tc.want, c.mayQuiet(tc.at))
		})
	}
}

func (s *SampleSuite) TestCostPrefersNoiseAndStaysBounded() {
	n := 4000
	f := func(i int) int32 {
		if i < n/2 {
			return 200 + int32(i%3)
		}
		return int32((i*1103515245+12345)%2001 - 1000)
	}
	sm := Samples{Planes: planes(n, f), N: n, Scale: testScale}
	cover := newSampleCover(sm)
	cover.posKey = testKey
	var smooth, noisy float64
	var ns, nn int
	ch := 2
	for i := range cover.Len() {
		p := cover.elig[i]
		cost := float64(cover.Cost(i))
		s.GreaterOrEqual(cost, costFloor*float64(1-ditherAmp)-1e-4)
		s.LessOrEqual(cost, costFloor*costRatio*float64(1+ditherAmp)+1e-3)
		if p/ch < n/2 {
			smooth += cost
			ns++
		} else {
			noisy += cost
			nn++
		}
	}
	s.Positive(ns)
	s.Positive(nn)
	s.Greater(smooth/float64(ns), noisy/float64(nn))

	other := newSampleCover(sm)
	other.posKey = []byte("another position key, 32 bytes.")
	s.NotEqual(cover.Cost(0), other.Cost(0))
	again := newSampleCover(sm)
	again.posKey = testKey
	s.Equal(cover.Cost(0), again.Cost(0))
}

func (s *SampleSuite) TestEligibleListSurvivesEmbedding() {
	sm := Samples{Planes: planes(2000, func(i int) int32 { return 100 + int32(i%50) }), N: 2000, Scale: testScale}
	before := newSampleCover(sm).elig
	s.Require().NoError(ApplySamples(sm, testKey, []byte("hi")))
	s.Equal(before, newSampleCover(sm).elig)
}

func (s *SampleSuite) TestFlipPrefersTheSmallerResidual() {
	// One quiet sample between loud neighbours: stepping up shrinks the
	// residual, stepping down grows it. The histogram is flat, so the
	// residual is the whole decision.
	vals := []int32{100, 0, 100}
	p := make([]float32, len(vals))
	for i, v := range vals {
		p[i] = float32(v) / testScale
	}
	c := newSampleCover(Samples{Planes: [][]float32{p}, N: len(p), Scale: testScale})
	s.Less(c.baseScore(1, 0, 1), c.baseScore(1, 0, -1))
}

func (s *SampleSuite) TestStepDirection() {
	tests := []struct {
		title    string
		v        int32
		mayQuiet bool
		want     []int32
	}{
		{"just above the floor, may not go quiet", silenceFloor + 1, false, []int32{silenceFloor + 2}},
		{"negative, may not go quiet", -silenceFloor - 1, false, []int32{-silenceFloor - 2}},
		{"just above the floor, may go quiet", silenceFloor + 1, true, []int32{silenceFloor, silenceFloor + 2}},
		{"top of the grid", testScale - 1, true, []int32{testScale - 2}},
		{"bottom of the grid", -testScale, true, []int32{-testScale + 1}},
	}
	for _, tc := range tests {
		s.Run(tc.title, func() {
			c := newSampleCover(Samples{Planes: [][]float32{{0}}, N: 1, Scale: testScale})
			c.hist = map[int32]int{tc.v - 1: 1, tc.v + 1: 1}
			seen := map[int32]bool{}
			for range 64 {
				seen[c.step(tc.v, tc.mayQuiet)] = true
			}
			s.Len(seen, len(tc.want))
			for _, w := range tc.want {
				s.True(seen[w], "never moved to %d", w)
			}
		})
	}
}

// Random ±1 moves samples off a peak faster than they come back, which
// widens the histogram: on average every change adds exactly 1 to Σv².
// Steering by the histogram should leave Σv² near where it was. The
// residual tilts a close call, so the mean can drift a little; it still
// has to stay below the +1 per change that an unsteered ±1 adds.
func (s *SampleSuite) TestKeepsTheHistogramFromWidening() {
	var widened, changes int
	for trial := range 8 {
		rng := rand.New(rand.NewPCG(uint64(trial), 4))
		laplace := func(int) int32 {
			x := -3 * math.Log(rng.Float64())
			if rng.IntN(2) == 0 {
				x = -x
			}
			return int32(math.Round(x))
		}
		const n = 44100 * 4
		sm := Samples{Planes: planes(n, laplace), N: n, Scale: testScale}
		before := values(sm)
		s.Require().NoError(ApplySamples(sm, []byte{byte(trial)}, nil))
		for i, v := range values(sm) {
			if v != before[i] {
				changes++
				widened += int(v*v - before[i]*before[i])
			}
		}
	}
	s.Greater(changes, 1000)
	s.Less(math.Abs(float64(widened))/float64(changes), 0.75, "Σv² grew by %d over %d changes", widened, changes)
}

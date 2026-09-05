// Package frame models fixed-duration stego frames and Listen's phase search.
//
// A live stream has no known total length and a receiver may join mid-stream.
// Mist therefore splits audio into consecutive self-contained frames of
// duration D (a protocol constant). The same payload is re-sealed with a
// fresh ephemeral key in every frame.
//
// Frame boundaries are not marked in the bitstream. Listen tries a small
// set of candidate phase offsets and uses AEAD tag verification as the
// correctness oracle — the offset that authenticates is the right one;
// every other offset fails identically to "no message".
package frame

import "time"

// Frame is one self-contained stego unit.
type Frame struct {
	Index    int64
	Offset   time.Duration
	Duration time.Duration
	PCMStart int // sample index in the current Listen session
	PCMEnd   int
}

// Phase is a candidate frame-boundary offset tried by Listen on join.
type Phase struct {
	Offset time.Duration
	Index  int
}

// Params are the audio properties needed to convert duration to samples.
type Params struct {
	SampleRate int
	Channels   int
	Duration   time.Duration
}

// Samples returns the number of interleaved PCM frames (sample-instants)
// in one stego frame for p.
func (p Params) Samples() int {
	if p.SampleRate <= 0 || p.Duration <= 0 {
		return 0
	}
	return int(p.Duration.Seconds() * float64(p.SampleRate))
}

// Split returns the sequence of frames covering nSamples at p.
func Split(nSamples int, p Params) []Frame {
	_ = nSamples
	_ = p
	return nil
}

// CandidatePhases returns the phase offsets Listen should try when the
// recording start is unknown. hop is the search step; it must divide
// Duration or be smaller than it.
func CandidatePhases(d, hop time.Duration) []Phase {
	_ = d
	_ = hop
	return nil
}

// Capacity is the payload bytes that fit in one frame after encryption
// overhead, given nEligible coefficients and a constant embedding density.
func Capacity(nEligible int, density float64, overhead int) int {
	_ = nEligible
	_ = density
	_ = overhead
	return 0
}

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
// A short tail is its own final frame so a clip shorter than Duration
// still carries one embed.
func Split(nSamples int, p Params) []Frame {
	win := p.Samples()
	if nSamples <= 0 || win <= 0 {
		return nil
	}
	var out []Frame
	for i, start := 0, 0; start < nSamples; i++ {
		end := start + win
		if end > nSamples {
			end = nSamples
		}
		out = append(out, Frame{
			Index:    int64(i),
			Offset:   time.Duration(start) * time.Second / time.Duration(p.SampleRate),
			Duration: time.Duration(end-start) * time.Second / time.Duration(p.SampleRate),
			PCMStart: start,
			PCMEnd:   end,
		})
		start = end
	}
	return out
}

// CandidatePhases returns the phase offsets Listen should try when the
// recording start is unknown. hop is the search step; it must divide
// Duration or be smaller than it.
func CandidatePhases(d, hop time.Duration) []Phase {
	if d <= 0 {
		return nil
	}
	if hop <= 0 || hop > d {
		hop = d
	}
	var out []Phase
	for i, off := 0, time.Duration(0); off < d; i++ {
		out = append(out, Phase{Offset: off, Index: i})
		off += hop
	}
	return out
}

// Capacity is the payload bytes that fit in one frame after encryption
// overhead, given nEligible coefficients and a constant embedding density.
func Capacity(nEligible int, density float64, overhead int) int {
	if nEligible <= 0 || density <= 0 {
		return 0
	}
	bytes := int(float64(nEligible)*density) / 8
	if bytes <= overhead {
		return 0
	}
	return bytes - overhead
}

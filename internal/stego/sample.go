package stego

import "math"

// Samples is one stego frame of planar float PCM, seen as a flat carrier
// of integer samples. A lossless codec reproduces those integers exactly,
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

// Len implements carrier: every sample in the window is eligible. Unlike
// residues there is no frequency band to respect — the LSB of a 16-bit
// sample is 96 dB down whatever it encodes.
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
func (s Samples) Capacity() int { return slots(s.Len()) / 8 }

func (s Samples) locate(i int) ([]float32, int) {
	ch := len(s.Planes)
	return s.Planes[i%ch], s.Off + i/ch
}

func (s Samples) clamp(v int32) int32 {
	hi, lo := int32(s.Scale)-1, -int32(s.Scale)
	for v > hi {
		v -= 2
	}
	for v < lo {
		v += 2
	}
	return v
}

// ApplySamples embeds bits into one frame of PCM. Nil bits fill the frame
// with filler at the same density, the way an empty residue frame is.
func ApplySamples(s Samples, posKey []byte, bits Bits) error {
	return place(s, posKey, bits)
}

// RecoverSamples reads the constant-density bit string from one frame.
func RecoverSamples(s Samples, posKey []byte) (Bits, error) {
	return lift(s, posKey)
}

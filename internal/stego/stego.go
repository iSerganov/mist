// Package stego embeds and extracts bit strings in quantized Vorbis residues.
//
// Design constraints this package must preserve:
//   - Keyed PRNG position selection — never touch coefficients sequentially.
//   - Constant embedding density — every encode perturbs the same fraction
//     of eligible coefficients; unused room is CSPRNG filler.
//   - LSB matching (±1), never LSB replacement.
//   - Extraction reads residues from the compressed bitstream and stops
//     before inverse MDCT. No PCM reanalysis.
package stego

import "github.com/iSerganov/mist/internal/codec"

// Density is the fraction of eligible coefficients perturbed on every
// encode, whether or not a real payload is present. It is a protocol
// constant: changing it is a breaking change for Listen.
const Density = 0.10

// Bits is a packed bit string to embed or a bit string just extracted.
type Bits []byte

// Embedder writes payload bits into one PCM window and returns compressed
// packets whose residues carry those bits.
//
// Two implementations are expected:
//   - BlackBox: drive unmodified libvorbisenc as an oracle, perturb PCM,
//     re-encode, inspect residue parity, retry until the target bits land.
//   - Patched: a vendored libvorbis that exposes quantized residues via
//     a callback so they can be rewritten before Huffman pack.
type Embedder interface {
	Embed(pcm codec.PCM, bits Bits) ([]codec.Packet, error)
	Close() error
}

// Extractor reads payload bits from compressed packets by parsing
// quantized residues and stopping before inverse MDCT.
type Extractor interface {
	Extract(packets []codec.Packet) (Bits, error)
}

// Filler returns CSPRNG bytes used to pad a frame out to constant density.
func Filler(n int) ([]byte, error) {
	_ = n
	return nil, errUnimplemented
}

// NewBlackBoxEmbedder returns the encoder-in-the-loop embedder.
func NewBlackBoxEmbedder(enc codec.Encoder) Embedder {
	_ = enc
	return unimplementedEmbedder{}
}

// NewPatchedEmbedder returns the embedder that talks to a patched libvorbis.
func NewPatchedEmbedder(enc codec.Encoder) Embedder {
	_ = enc
	return unimplementedEmbedder{}
}

// NewExtractor returns the bitstream-domain extractor.
func NewExtractor(c codec.Codec) Extractor {
	_ = c
	return unimplementedExtractor{}
}

type unimplementedEmbedder struct{}

func (unimplementedEmbedder) Embed(codec.PCM, Bits) ([]codec.Packet, error) {
	return nil, errUnimplemented
}

func (unimplementedEmbedder) Close() error { return nil }

type unimplementedExtractor struct{}

func (unimplementedExtractor) Extract([]codec.Packet) (Bits, error) {
	return nil, errUnimplemented
}

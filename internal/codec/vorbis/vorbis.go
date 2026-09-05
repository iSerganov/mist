// Package vorbis is the Phase 1 Codec: Ogg Vorbis via libav, with a
// residue parser that stops after Huffman/codebook decode.
//
// Standard PCM↔Vorbis conversion uses unmodified libavcodec/libavformat.
// Coefficient-domain access does not: libvorbisenc discards quantized
// residues inside the encoder. Embedding therefore goes through one of
// the two paths in package stego (black-box search or a patched libvorbis).
// Extraction only needs the public Vorbis bitstream spec.
package vorbis

import "github.com/iSerganov/mist/internal/codec"

// Name is the codec name reported by Codec.Name.
const Name = "vorbis"

// Codec implements codec.Codec for Vorbis.
type Codec struct{}

// New returns the Vorbis codec.
func New() *Codec { return &Codec{} }

// Name implements codec.Codec.
func (*Codec) Name() string { return Name }

// ID implements codec.Codec.
func (*Codec) ID() codec.ID { return codec.IDVorbis }

// NewEncoder implements codec.Codec.
func (*Codec) NewEncoder(p codec.Params) (codec.Encoder, error) {
	_ = p
	return nil, errUnimplemented
}

// NewDecoder implements codec.Codec.
func (*Codec) NewDecoder(p codec.Params) (codec.Decoder, error) {
	_ = p
	return nil, errUnimplemented
}

// Residues parses quantized MDCT residues from a Vorbis audio packet
// and stops before inverse MDCT.
func (*Codec) Residues(pkt codec.Packet) ([]codec.Residue, error) {
	_ = pkt
	return nil, errUnimplemented
}

// Setup holds the three Vorbis identification / comment / setup headers
// needed to parse subsequent audio packets.
type Setup struct {
	Ident     []byte
	Comment   []byte
	Codebooks []byte
}

// ParseSetup decodes the three header packets from a fresh Ogg stream.
func ParseSetup(ident, comment, setup []byte) (*Setup, error) {
	return nil, errUnimplemented
}

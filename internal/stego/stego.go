// Package stego embeds and extracts bit strings in quantized Vorbis residues.
//
// PCM encode still uses unmodified libav. After libav emits packets this
// package reads residues from the bitstream, writes the bits through a
// syndrome-trellis code over the eligible residues in keyed order, and
// rewrites the residue section. Extraction is the inverse and never looks
// at PCM. Constant density means unused room is CSPRNG filler so presence
// and absence have the same footprint.
package stego

import (
	"crypto/rand"
	"fmt"

	"github.com/iSerganov/mist/internal/codec"
)

// Density is the payload rate: bits carried per eligible coefficient, on
// every encode, whether or not a real payload is present. It is a protocol
// constant: changing it is a breaking change for Listen.
//
// Every changed value is damage, so this is kept just high enough to be
// useful: roughly 130 bytes per 8-second Vorbis frame. At this rate the
// trellis code changes about 0.3% of the eligible values, 0.14 changes
// per bit, where writing each bit into a value of its own changed 1%.
const Density = 0.02

// maxFlipCost bounds how far one flip may move a residue's spectral
// vector, as a squared distance in dequantized residue units. A flip past
// it costs wetCost, so the trellis code takes it only when no path avoids
// it. minCost keeps an exactly free substitute from looking costless,
// which would let the code pile flips onto it for nothing.
const (
	maxFlipCost = 9.0
	wetCost     = 1e6
	minCost     = 1e-3
)

// Bits is a packed bit string to embed or a bit string just extracted.
type Bits []byte

// Embedder writes payload bits into one PCM window and returns compressed
// packets whose residues carry those bits.
type Embedder interface {
	Embed(pcm codec.PCM, bits Bits) ([]codec.Packet, error)
	Close() error
}

// Extractor reads payload bits from compressed packets by parsing
// quantized residues and stopping before inverse MDCT.
type Extractor interface {
	Extract(packets []codec.Packet) (Bits, error)
}

type rewriter interface {
	Residues(codec.Packet) ([]codec.Residue, error)
	Rewrite(codec.Packet, []codec.Residue) (codec.Packet, error)
}

// Filler returns CSPRNG bytes used to pad a frame out to constant density.
func Filler(n int) ([]byte, error) {
	if n <= 0 {
		return []byte{}, nil
	}
	out := make([]byte, n)
	if _, err := rand.Read(out); err != nil {
		return nil, fmt.Errorf("stego: filler: %w", err)
	}
	return out, nil
}

// NewBlackBoxEmbedder encodes PCM with the stock libav encoder, then
// rewrites residue LSBs in the resulting packets. The encoder is an
// oracle for legal Vorbis; it never sees the payload bits.
func NewBlackBoxEmbedder(enc codec.Encoder, c rewriter, posKey []byte) Embedder {
	return &boxEmbedder{enc: enc, codec: c, posKey: append([]byte(nil), posKey...)}
}

// NewPatchedEmbedder is the same path until a vendored libvorbis exists.
func NewPatchedEmbedder(enc codec.Encoder, c rewriter, posKey []byte) Embedder {
	return NewBlackBoxEmbedder(enc, c, posKey)
}

// NewExtractor returns the bitstream-domain extractor.
func NewExtractor(c rewriter, posKey []byte) Extractor {
	return &extractor{codec: c, posKey: append([]byte(nil), posKey...)}
}

type boxEmbedder struct {
	enc    codec.Encoder
	codec  rewriter
	posKey []byte
}

func (e *boxEmbedder) Embed(pcm codec.PCM, bits Bits) ([]codec.Packet, error) {
	if e == nil || e.enc == nil || e.codec == nil {
		return nil, errUnimplemented
	}
	pkts, err := e.enc.Encode(pcm)
	if err != nil {
		return nil, err
	}
	return Apply(e.codec, e.posKey, pkts, bits)
}

// Apply writes bits into already-encoded packets using posKey.
func Apply(c rewriter, posKey []byte, pkts []codec.Packet, bits Bits) ([]codec.Packet, error) {
	return embedPackets(c, posKey, pkts, bits)
}

// Capacity returns the maximum bytes embeddable in pkts at the constant
// density, so a caller can decide whether a frame is even large enough
// for the protocol envelope before calling Apply. It returns ErrNoResidues
// if pkts has no eligible residues at all (e.g. a near-empty tail frame).
func Capacity(c rewriter, pkts []codec.Packet) (int, error) {
	res, err := collectResidues(c, pkts)
	if err != nil {
		return 0, err
	}
	return slots(res.Len()) / 8, nil
}

// Recover reads the constant-density bit string from packets.
func Recover(c rewriter, posKey []byte, pkts []codec.Packet) (Bits, error) {
	return (&extractor{codec: c, posKey: posKey}).Extract(pkts)
}

func (e *boxEmbedder) Close() error {
	if e == nil || e.enc == nil {
		return nil
	}
	return e.enc.Close()
}

type extractor struct {
	codec  rewriter
	posKey []byte
}

func (x *extractor) Extract(packets []codec.Packet) (Bits, error) {
	if x == nil || x.codec == nil {
		return nil, errUnimplemented
	}
	res, err := collectResidues(x.codec, packets)
	if err != nil {
		return nil, err
	}
	return lift(res, x.posKey)
}

func embedPackets(c rewriter, posKey []byte, pkts []codec.Packet, bits Bits) ([]codec.Packet, error) {
	res, err := collectResidues(c, pkts)
	if err != nil {
		return nil, err
	}
	if err := place(res, posKey, bits); err != nil {
		return nil, err
	}
	return rewriteAll(c, pkts, res.all)
}

// EligibleValues returns the quantized values of every residue in pkts that
// embedding may touch, in the order positions index them: what a warden
// who knows the bands and codebooks would inspect.
func EligibleValues(c rewriter, pkts []codec.Packet) ([]int32, error) {
	res, err := collectResidues(c, pkts)
	if err != nil {
		return nil, err
	}
	out := make([]int32, res.Len())
	for i := range out {
		out[i] = res.At(i)
	}
	return out, nil
}

// residues is the carrier over one frame's quantized Vorbis residues:
// the eligible subset of the symbols the packets decoded to.
type residues struct {
	all      []codec.Residue
	views    []ResidueView
	eligible []int
}

func (r *residues) Len() int       { return len(r.eligible) }
func (r *residues) At(i int) int32 { return r.views[r.eligible[i]].Value }

// Cost is how far the residue's spectral vector moves when it is
// substituted. Past maxFlipCost it jumps to wetCost, so the code routes
// around the flip whenever it can.
func (r *residues) Cost(i int) float32 {
	d := r.all[r.eligible[i]].FlipCost
	if d > maxFlipCost {
		return wetCost
	}
	return float32(d) + minCost
}

// Flip only needs the parity right: the codec swaps in the entry of that
// parity nearest the original when it rewrites the packet.
func (r *residues) Flip(i int) {
	v := r.views[r.eligible[i]].Value ^ 1
	r.views[r.eligible[i]].Value = v
	r.all[r.eligible[i]].Value = v
}

func collectResidues(c rewriter, pkts []codec.Packet) (*residues, error) {
	var all []codec.Residue
	for _, pkt := range pkts {
		if len(pkt.Data) == 0 || pkt.Data[0]&1 == 1 {
			continue
		}
		r, err := c.Residues(pkt)
		if err != nil {
			return nil, err
		}
		all = append(all, r...)
	}
	if len(all) == 0 {
		return nil, ErrNoResidues
	}
	views := make([]ResidueView, len(all))
	for i, r := range all {
		views[i] = ResidueView{Index: r.Index, Band: r.Band, Value: r.Value, Unflippable: r.Unflippable}
	}
	el := Eligible(views, DefaultBands)
	if len(el) == 0 {
		return nil, ErrNoResidues
	}
	return &residues{all: all, views: views, eligible: el}, nil
}

func rewriteAll(c rewriter, pkts []codec.Packet, res []codec.Residue) ([]codec.Packet, error) {
	out := make([]codec.Packet, 0, len(pkts))
	off := 0
	for _, pkt := range pkts {
		if len(pkt.Data) == 0 || pkt.Data[0]&1 == 1 {
			out = append(out, pkt)
			continue
		}
		n, err := c.Residues(pkt)
		if err != nil {
			return nil, err
		}
		end := off + len(n)
		if end > len(res) {
			return nil, ErrNoResidues
		}
		rew, err := c.Rewrite(pkt, res[off:end])
		if err != nil {
			return nil, err
		}
		out = append(out, rew)
		off = end
	}
	return out, nil
}

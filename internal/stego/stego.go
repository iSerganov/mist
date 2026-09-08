// Package stego embeds and extracts bit strings in quantized Vorbis residues.
//
// PCM encode still uses unmodified libav. After libav emits packets this
// package reads residues from the bitstream, applies LSB matching at keyed
// positions, and rewrites the residue section. Extraction is the inverse
// and never looks at PCM. Constant density means unused room is CSPRNG
// filler so presence and absence have the same footprint.
package stego

import (
	"crypto/rand"
	"fmt"

	"github.com/iSerganov/mist/internal/codec"
)

// Density is the fraction of eligible coefficients perturbed on every
// encode, whether or not a real payload is present. It is a protocol
// constant: changing it is a breaking change for Listen.
//
// Every perturbed residue is audible damage, so this is kept just high
// enough to be useful. Measured on real music, 2% inside DefaultBands
// costs about 0.2 dB of signal-to-distortion against a plain transcode
// while leaving roughly 130 bytes per 8-second frame. Raising it trades
// audio quality for capacity in direct proportion: 10% cost 4 dB.
const Density = 0.02

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
	_, views, err := collectResidues(c, pkts)
	if err != nil {
		return 0, err
	}
	el := Eligible(views, DefaultBands)
	if len(el) == 0 {
		return 0, ErrNoResidues
	}
	nbits := int(float64(len(el)) * Density)
	return nbits / 8, nil
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
	res, views, err := collectResidues(x.codec, packets)
	if err != nil {
		return nil, err
	}
	_ = res
	el := Eligible(views, DefaultBands)
	if len(el) == 0 {
		return nil, ErrNoResidues
	}
	nbits := int(float64(len(el)) * Density)
	if nbits < 1 {
		return nil, ErrNoResidues
	}
	sel := NewSelector(x.posKey, len(el))
	pos := sel.Pick(nbits)
	out := make([]byte, (nbits+7)/8)
	for i, p := range pos {
		bit := LSB(views[el[p]].Value)
		if bit == 1 {
			out[i/8] |= 1 << uint(i%8)
		}
	}
	return out, nil
}

func embedPackets(c rewriter, posKey []byte, pkts []codec.Packet, bits Bits) ([]codec.Packet, error) {
	res, views, err := collectResidues(c, pkts)
	if err != nil {
		return nil, err
	}
	el := Eligible(views, DefaultBands)
	if len(el) == 0 {
		return nil, ErrNoResidues
	}
	nbits := int(float64(len(el)) * Density)
	if nbits < 1 {
		return nil, ErrNoResidues
	}
	need := (nbits + 7) / 8
	if len(bits) > need {
		return nil, ErrCapacity
	}
	payload := make([]byte, need)
	copy(payload, bits)
	if len(bits) < need {
		pad, err := Filler(need - len(bits))
		if err != nil {
			return nil, err
		}
		copy(payload[len(bits):], pad)
	}
	sel := NewSelector(posKey, len(el))
	pos := sel.Pick(nbits)
	for i, p := range pos {
		bit := (payload[i/8] >> uint(i%8)) & 1
		ri := el[p]
		views[ri].Value = Match(views[ri].Value, bit)
		res[ri].Value = views[ri].Value
	}
	return rewriteAll(c, pkts, res)
}

func collectResidues(c rewriter, pkts []codec.Packet) ([]codec.Residue, []ResidueView, error) {
	var res []codec.Residue
	for _, pkt := range pkts {
		if len(pkt.Data) == 0 || pkt.Data[0]&1 == 1 {
			continue
		}
		r, err := c.Residues(pkt)
		if err != nil {
			return nil, nil, err
		}
		res = append(res, r...)
	}
	if len(res) == 0 {
		return nil, nil, ErrNoResidues
	}
	views := make([]ResidueView, len(res))
	for i, r := range res {
		views[i] = ResidueView{Index: r.Index, Band: r.Band, Value: r.Value, Unflippable: r.Unflippable}
	}
	return res, views, nil
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

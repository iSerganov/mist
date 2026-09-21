// Package vorbis is the Phase 1 codec: PCM ↔ Ogg Vorbis via package av,
// plus a bitstream residue parser used by stego.
//
// PCM encode and decode go through unmodified libavcodec only. Residue
// access is a Go read of the Vorbis packet (Huffman/codebook, stop before
// iMDCT). Rewrite puts modified residue VQ entries back into the same packet
// so a stock encoder never has to see the payload bits.
package vorbis

import (
	"fmt"

	"github.com/iSerganov/mist/internal/av"
	"github.com/iSerganov/mist/internal/codec"
)

// Name is the codec name reported by Codec.Name.
const Name = "vorbis"

const (
	headerIdent   = 1
	headerComment = 3
	headerSetup   = 5
	identSize     = 30
	magic         = "vorbis"
)

// Codec implements codec.Codec for Vorbis.
type Codec struct {
	setup *Setup
}

// New returns the Vorbis codec.
func New() *Codec { return &Codec{} }

// Load parses libavcodec Xiph extradata so Residues and Rewrite can
// walk audio packets. Call this with Encoder.Params().Extradata after
// a successful libav open.
func (c *Codec) Load(extra []byte) error {
	ident, comment, setup, err := SplitExtradata(extra)
	if err != nil {
		return err
	}
	s, err := ParseSetup(ident, comment, setup)
	if err != nil {
		return err
	}
	if s.tables == nil {
		return fmt.Errorf("%w: no codebooks", ErrBadSetup)
	}
	c.setup = s
	return nil
}

// Setup returns the parsed headers, or nil before Load.
func (c *Codec) Setup() *Setup { return c.setup }

// Name implements codec.Codec.
func (*Codec) Name() string { return Name }

// ID implements codec.Codec.
func (*Codec) ID() codec.ID { return codec.IDVorbis }

// NewEncoder opens a libav Vorbis encoder. After open, Params() on the
// concrete encoder holds Extradata the matching decoder needs.
func (*Codec) NewEncoder(p codec.Params) (codec.Encoder, error) {
	if err := av.Init(); err != nil {
		return nil, err
	}
	return av.NewEncoder(paramsToInfo(p))
}

// NewDecoder opens a libav Vorbis decoder. p.Extradata must be the three
// setup headers from a matching encoder or demuxer; without them libav
// cannot reconstruct codebooks and open fails.
func (*Codec) NewDecoder(p codec.Params) (codec.Decoder, error) {
	if len(p.Extradata) == 0 {
		return nil, fmt.Errorf("%w: missing extradata", ErrBadSetup)
	}
	if err := av.Init(); err != nil {
		return nil, err
	}
	return av.NewDecoder(paramsToInfo(p))
}

// Residues classifies a Vorbis packet. Header packets (odd first byte
// 1/3/5) are ErrBadPacket. Audio packets need Load first; without
// codebooks the return is ErrBadSetup, not an empty slice.
func (c *Codec) Residues(pkt codec.Packet) ([]codec.Residue, error) {
	if len(pkt.Data) == 0 {
		return nil, ErrBadPacket
	}
	if pkt.Data[0]&1 == 1 {
		return nil, ErrBadPacket
	}
	if c == nil || c.setup == nil || c.setup.tables == nil {
		return nil, fmt.Errorf("%w: codebooks not parsed", ErrBadSetup)
	}
	st, err := decodeAudio(c.setup, pkt.Data)
	if err != nil {
		return nil, err
	}
	return residuesFrom(st, c.setup.Rate, c.setup.tables.books), nil
}

// Rewrite writes res (VQ entry indexes from Residues) back into pkt.
// Prefix bits (mode, windows, floors) are copied unchanged. The result
// must still be a legal Vorbis packet so libav can decode it to PCM.
func (c *Codec) Rewrite(pkt codec.Packet, res []codec.Residue) (codec.Packet, error) {
	if c == nil || c.setup == nil || c.setup.tables == nil {
		return codec.Packet{}, fmt.Errorf("%w: codebooks not parsed", ErrBadSetup)
	}
	if len(pkt.Data) == 0 || pkt.Data[0]&1 == 1 {
		return codec.Packet{}, ErrBadPacket
	}
	st, err := decodeAudio(c.setup, pkt.Data)
	if err != nil {
		return codec.Packet{}, err
	}
	orig := entryValues(st.progs)
	applyResidues(st, res)
	sanitizePrograms(c.setup.tables.books, st.progs, orig)
	data, err := encodeAudio(c.setup, pkt.Data, st)
	if err != nil {
		return codec.Packet{}, err
	}
	out := pkt
	out.Data = data
	return out, nil
}

func paramsToInfo(p codec.Params) av.AudioInfo {
	rate, ch := p.SampleRate, p.Channels
	if rate <= 0 {
		rate = 44100
	}
	if ch <= 0 {
		ch = 2
	}
	fmt := p.Format
	if fmt == codec.SampleFmtNone {
		fmt = codec.SampleFmtFLTP
	}
	return av.AudioInfo{
		CodecID:    av.CodecIDVorbis,
		SampleRate: rate,
		Channels:   ch,
		SampleFmt:  fmt,
		Bitrate:    p.Bitrate,
		Extradata:  p.Extradata,
	}
}


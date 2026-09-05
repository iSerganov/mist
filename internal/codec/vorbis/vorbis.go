// Package vorbis is the Phase 1 codec: PCM ↔ Ogg Vorbis via package av,
// plus a bitstream header parser for later residue extraction.
//
// Standard encode and decode use unmodified libavcodec. Coefficient-domain
// access does not: libvorbisenc discards quantized residues inside the
// encoder. Residues therefore classifies header vs audio packets and
// parses identification fully, but returns ErrBadSetup until codebooks are
// decoded. Embedding stays in package stego (black-box search or a patch).
package vorbis

import (
	"encoding/binary"
	"errors"
	"fmt"
	"math"

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
type Codec struct{}

// New returns the Vorbis codec.
func New() *Codec { return &Codec{} }

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
	info := paramsToInfo(p)
	enc, err := av.NewEncoder(info)
	if err != nil {
		return nil, err
	}
	return &encoder{enc: enc}, nil
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
	dec, err := av.NewDecoder(paramsToInfo(p))
	if err != nil {
		return nil, err
	}
	return &decoder{dec: dec}, nil
}

// Residues classifies a Vorbis packet. Header packets (odd first byte
// 1/3/5) are ErrBadPacket. Audio packets need Huffman/codebook decode
// that is not implemented yet, so the return is ErrBadSetup rather than
// a silent empty slice — callers must not treat that as "no residues".
func (*Codec) Residues(pkt codec.Packet) ([]codec.Residue, error) {
	if len(pkt.Data) == 0 {
		return nil, ErrBadPacket
	}
	if pkt.Data[0]&1 == 1 {
		return nil, ErrBadPacket
	}
	return nil, fmt.Errorf("%w: codebooks not parsed", ErrBadSetup)
}

type encoder struct {
	enc *av.Encoder
}

// Params returns the stream after encoder open, including Vorbis
// extradata the decoder needs to reconstruct the three setup headers.
func (e *encoder) Params() codec.Params {
	if e == nil || e.enc == nil {
		return codec.Params{ID: codec.IDVorbis}
	}
	return e.enc.Info().Params()
}

func (e *encoder) Encode(pcm codec.PCM) ([]codec.Packet, error) {
	if e == nil || e.enc == nil {
		return nil, av.ErrClosed
	}
	fs := e.enc.Info().FrameSize
	if fs <= 0 {
		fs = pcm.NbSamples
	}
	var out []codec.Packet
	for _, chunk := range splitPCM(pcm, fs) {
		if err := e.enc.Send(frameFromPCM(chunk)); err != nil && !errors.Is(err, av.ErrAgain) {
			return out, err
		}
		pkts, err := drainEnc(e.enc)
		out = append(out, pkts...)
		if err != nil {
			return out, err
		}
	}
	return out, nil
}

func (e *encoder) Flush() ([]codec.Packet, error) {
	if e == nil || e.enc == nil {
		return nil, av.ErrClosed
	}
	if err := e.enc.Send(av.Frame{}); err != nil && !errors.Is(err, av.ErrAgain) && !errors.Is(err, av.ErrEOF) {
		return nil, err
	}
	return drainEnc(e.enc)
}

func (e *encoder) Close() error {
	if e == nil || e.enc == nil {
		return nil
	}
	err := e.enc.Close()
	e.enc = nil
	return err
}

type decoder struct {
	dec *av.Decoder
}

func (d *decoder) Decode(pkt codec.Packet) ([]codec.PCM, error) {
	if d == nil || d.dec == nil {
		return nil, av.ErrClosed
	}
	if err := d.dec.Send(av.FromCodecPacket(pkt)); err != nil && !errors.Is(err, av.ErrAgain) && !errors.Is(err, av.ErrEOF) {
		return nil, err
	}
	return drainDec(d.dec)
}

func (d *decoder) Close() error {
	if d == nil || d.dec == nil {
		return nil
	}
	err := d.dec.Close()
	d.dec = nil
	return err
}

func drainEnc(enc *av.Encoder) ([]codec.Packet, error) {
	var out []codec.Packet
	for {
		pkt, err := enc.Receive()
		if errors.Is(err, av.ErrAgain) || errors.Is(err, av.ErrEOF) {
			return out, nil
		}
		if err != nil {
			return nil, err
		}
		out = append(out, av.ToCodecPacket(pkt))
	}
}

func drainDec(dec *av.Decoder) ([]codec.PCM, error) {
	var out []codec.PCM
	for {
		fr, err := dec.Receive()
		if errors.Is(err, av.ErrAgain) || errors.Is(err, av.ErrEOF) {
			return out, nil
		}
		if err != nil {
			return nil, err
		}
		out = append(out, frameToPCM(fr))
	}
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

func pcmPlanes(p codec.PCM) [][]float32 {
	switch {
	case len(p.Planes) > 0:
		return p.Planes
	case len(p.Samples) > 0 && p.Channels > 0:
		return deinterleave(p.Samples, p.Channels, p.NbSamples)
	default:
		return nil
	}
}

// splitPCM cuts p into encoder-sized windows. libvorbis rejects any
// nb_samples other than frame_size, so a short tail is zero-padded.
func splitPCM(p codec.PCM, fs int) []codec.PCM {
	if fs <= 0 || p.NbSamples <= 0 {
		return nil
	}
	planes := pcmPlanes(p)
	var out []codec.PCM
	for off := 0; off < p.NbSamples; off += fs {
		chunk := make([][]float32, len(planes))
		for i, pl := range planes {
			sl := make([]float32, fs)
			end := off + fs
			if end > len(pl) {
				end = len(pl)
			}
			if off < len(pl) {
				copy(sl, pl[off:end])
			}
			chunk[i] = sl
		}
		out = append(out, codec.PCM{
			Planes:     chunk,
			NbSamples:  fs,
			Channels:   p.Channels,
			SampleRate: p.SampleRate,
			Format:     codec.SampleFmtFLTP,
			PTS:        p.PTS + int64(off),
		})
	}
	return out
}

func frameFromPCM(p codec.PCM) av.Frame {
	ch, n := p.Channels, p.NbSamples
	planes := pcmPlanes(p)
	data := make([][]byte, len(planes))
	for i, pl := range planes {
		data[i] = floatsToBytes(pl)
	}
	return av.Frame{
		Data:       data,
		NbSamples:  n,
		Channels:   ch,
		SampleRate: p.SampleRate,
		Format:     codec.SampleFmtFLTP,
		PTS:        p.PTS,
	}
}

func frameToPCM(f av.Frame) codec.PCM {
	planes := make([][]float32, len(f.Data))
	for i, b := range f.Data {
		planes[i] = bytesToFloats(b)
	}
	return codec.PCM{
		Planes:     planes,
		NbSamples:  f.NbSamples,
		Channels:   f.Channels,
		SampleRate: f.SampleRate,
		Format:     f.Format,
		PTS:        f.PTS,
	}
}

func floatsToBytes(in []float32) []byte {
	out := make([]byte, len(in)*4)
	for i, v := range in {
		binary.LittleEndian.PutUint32(out[i*4:], math.Float32bits(v))
	}
	return out
}

func bytesToFloats(b []byte) []float32 {
	n := len(b) / 4
	out := make([]float32, n)
	for i := 0; i < n; i++ {
		out[i] = math.Float32frombits(binary.LittleEndian.Uint32(b[i*4:]))
	}
	return out
}

func deinterleave(in []float32, ch, n int) [][]float32 {
	if ch <= 0 {
		return nil
	}
	out := make([][]float32, ch)
	for c := 0; c < ch; c++ {
		out[c] = make([]float32, n)
		for i := 0; i < n && i*ch+c < len(in); i++ {
			out[c][i] = in[i*ch+c]
		}
	}
	return out
}

// Package av is the cgo boundary to libavformat, libavcodec, and libavutil.
//
// This package knows nothing about steganography. It demuxes, muxes,
// decodes to PCM, and encodes from PCM. Coefficient-domain access lives
// in package vorbis and package stego.
//
// Default builds compile against the C stubs in cgo.c so the tree builds
// without a system FFmpeg. Replacing those stubs with real libav calls
// (and adding `#cgo pkg-config: libavformat libavcodec libavutil`) is the
// implementation step. Builds with CGO_ENABLED=0 use the pure-Go fallback
// in stub.go; the Go API is identical.
package av

import (
	"io"
	"unsafe"

	"github.com/iSerganov/mist/internal/codec"
)

// AudioInfo is the subset of AVCodecParameters we need for an audio stream.
type AudioInfo struct {
	CodecID    int
	SampleRate int
	Channels   int
	SampleFmt  codec.SampleFormat
	Bitrate    int64
	DurationUs int64
}

// Params converts AudioInfo to the codec-layer Params.
func (a AudioInfo) Params() codec.Params {
	id := codec.IDNone
	if a.CodecID == CodecIDVorbis {
		id = codec.IDVorbis
	}
	return codec.Params{
		ID:         id,
		SampleRate: a.SampleRate,
		Channels:   a.Channels,
		Bitrate:    a.Bitrate,
		Format:     a.SampleFmt,
	}
}

// Codec IDs used at this boundary. They are Mist-local, not AV_CODEC_ID_*.
// The C layer maps them to libav values.
const (
	CodecIDNone   = 0
	CodecIDVorbis = 1
	CodecIDPCM    = 2
	CodecIDMP3    = 3
	CodecIDAAC    = 4
)

// Packet is a compressed AVPacket owned by Go after a successful read.
type Packet struct {
	Data        []byte
	StreamIndex int
	Flags       int
	PTS         int64
	DTS         int64
	Duration    int64
}

// Frame is a decoded AVFrame of PCM samples.
type Frame struct {
	Data       [][]byte
	Linesize   []int
	NbSamples  int
	Channels   int
	SampleRate int
	Format     codec.SampleFormat
	PTS        int64
}

// PCM converts f into the codec-layer PCM view.
func (f Frame) PCM() codec.PCM {
	return codec.PCM{
		NbSamples:  f.NbSamples,
		Channels:   f.Channels,
		SampleRate: f.SampleRate,
		Format:     f.Format,
		PTS:        f.PTS,
	}
}

// Demuxer reads compressed packets from a file, URL, or io.Reader.
type Demuxer struct {
	handle unsafe.Pointer
	info   AudioInfo
	closer io.Closer
}

// Muxer writes compressed packets to a file or io.Writer as an Ogg stream.
type Muxer struct {
	handle unsafe.Pointer
	info   AudioInfo
	closer io.Closer
}

// Decoder decodes compressed packets to PCM.
type Decoder struct {
	handle unsafe.Pointer
	info   AudioInfo
}

// Encoder encodes PCM to compressed packets.
type Encoder struct {
	handle unsafe.Pointer
	info   AudioInfo
}

// Info returns the audio stream parameters discovered at open.
func (d *Demuxer) Info() AudioInfo { return d.info }

// Info returns the audio parameters the muxer was opened with.
func (m *Muxer) Info() AudioInfo { return m.info }

// Info returns the decoder parameters.
func (d *Decoder) Info() AudioInfo { return d.info }

// Info returns the encoder parameters.
func (e *Encoder) Info() AudioInfo { return e.info }

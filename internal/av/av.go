// Package av is the cgo boundary to libavformat, libavcodec, and libavutil.
//
// This package knows nothing about steganography. It demuxes Ogg, muxes
// Vorbis packets, decodes to PCM, and encodes from PCM. Residue access
// lives in package vorbis. CGO builds need a system FFmpeg with libvorbis;
// CGO_ENABLED=0 uses stub.go so the module still type-checks.
package av

import (
	"io"
	"unsafe"

	"github.com/iSerganov/mist/internal/codec"
)

// AudioInfo is the subset of AVCodecParameters we need for an audio stream.
// Extradata is the codec private blob (Vorbis identification/comment/setup
// in Xiph lacing). The decoder will not open without it. FrameSize is the
// encoder's required sample count per Send, or 0 if the codec accepts any.
type AudioInfo struct {
	CodecID    int
	SampleRate int
	Channels   int
	SampleFmt  codec.SampleFormat
	Bitrate    int64
	DurationUs int64
	Extradata  []byte
	FrameSize  int
}

// Params converts AudioInfo to the codec-layer Params.
func (a AudioInfo) Params() codec.Params {
	id := codec.IDNone
	switch a.CodecID {
	case CodecIDVorbis:
		id = codec.IDVorbis
	case CodecIDPCM:
		id = codec.IDPCM
	case CodecIDMP3:
		id = codec.IDMP3
	case CodecIDAAC:
		id = codec.IDAAC
	}
	return codec.Params{
		ID:         id,
		SampleRate: a.SampleRate,
		Channels:   a.Channels,
		Bitrate:    a.Bitrate,
		Format:     a.SampleFmt,
		Extradata:  a.Extradata,
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
// Data is a copy; the C buffer is freed before the function returns.
type Packet struct {
	Data        []byte
	StreamIndex int
	Flags       int
	PTS         int64
	DTS         int64
	Duration    int64
}

// Frame is a decoded AVFrame of PCM samples.
// Planar formats use one Data slice per channel; packed formats use one.
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

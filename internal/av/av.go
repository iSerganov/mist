// Package av is the cgo boundary to libavformat, libavcodec, libavutil and
// libswresample.
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
// NativeCodecID is libav's own AVCodecID for the stream, carried verbatim
// so any format the installed FFmpeg can decode is usable; CodecID stays
// Mist-local and is CodecIDNone for everything else. CodecName is libav's
// display name, for error messages. Bits is bits_per_raw_sample, 0 when
// the stream does not say. VBR asks an encoder for constant Quality on its
// own scale (libvorbis -1..10), as ffmpeg's -q:a does, instead of Bitrate.
type AudioInfo struct {
	CodecID       int
	NativeCodecID int
	CodecName     string
	Container     string
	SampleRate    int
	Channels      int
	SampleFmt     codec.SampleFormat
	Bits          int
	Bitrate       int64
	VBR           bool
	Quality       int
	DurationUs    int64
	Extradata     []byte
	FrameSize     int
}

// Tag is one container metadata entry, in the order libav stored it.
type Tag struct {
	Key   string
	Value string
}

// Metadata is the tags ffmpeg copies onto an encode: the file's own
// entries, then the audio stream's. Order is the order a probe reports.
type Metadata struct {
	Format []Tag
	Stream []Tag
}

// TagSet selects which dictionary a tag belongs to.
type TagSet int

const (
	// TagsFormat is the file-level dictionary.
	TagsFormat TagSet = iota
	// TagsStream is the audio stream's dictionary.
	TagsStream
)

// Format is one output target: an encoder plus the container libav wraps
// it in. Every field is libav's own answer — CodecName from
// avcodec_get_name, Lossless from AV_CODEC_PROP_LOSSLESS — so which
// codecs qualify is the installed FFmpeg's decision, not a list here.
type Format struct {
	Container string
	CodecName string
	Ext       string
	CodecID   int
	Lossless  bool
}

// Info returns the encoder parameters for writing f from src. The
// encoder takes src's sample format and depth as its starting point, the
// way the ffmpeg command line does, and settles on what it can write.
func (f Format) Info(rate, channels int, src AudioInfo) AudioInfo {
	return AudioInfo{
		NativeCodecID: f.CodecID,
		Container:     f.Container,
		SampleRate:    rate,
		Channels:      channels,
		SampleFmt:     src.SampleFmt,
		Bits:          src.Bits,
	}
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
	// SkipStart and SkipEnd are the samples a decoder drops from the start
	// and the end of this packet's output: MP3 encoder delay and padding,
	// Opus pre-skip, the part of an Ogg page past its granule position.
	SkipStart uint32
	SkipEnd   uint32
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

// Metadata returns the container tags in the order libav stored them.
func (d *Demuxer) Metadata() (Metadata, error) { return avDemuxMetadata(d) }

// Info returns the audio parameters the muxer was opened with.
func (m *Muxer) Info() AudioInfo { return m.info }

// Info returns exactly what NewDecoder was called with, never refreshed
// from the opened codec context: callers who need the stego grid a stream
// was written at want the demuxer's own probed AudioInfo, not this.
func (d *Decoder) Info() AudioInfo { return d.info }

// Info returns the encoder parameters. It re-reads them: a FLAC encoder
// writes the MD5 and the total sample count into its extradata only once
// it has been flushed, and the muxer copies that blob into STREAMINFO.
func (e *Encoder) Info() AudioInfo {
	if e == nil {
		return AudioInfo{}
	}
	if e.handle != nil {
		container := e.info.Container
		if err := avEncInfo(e); err == nil {
			e.info.Container = container
		}
	}
	return e.info
}

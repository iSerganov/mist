// Package trace reads what an audio file shows without looking at its
// samples: container identity, length, how it ends, and the page or
// STREAMINFO facts a metadata warden can read. It compares two such
// traces field by field, and makes the plain ffmpeg encode a trace is
// compared against.
package trace

import (
	"bytes"
	"context"
	"encoding/binary"
	"fmt"
	"math"
	"os"
	"os/exec"
	"path/filepath"
)

// nominalSlack is how far two nominal bitrates may differ and still read
// as the same encoder setting rather than a different rate rule.
const nominalSlack = 0.05

// Trace is one file's identity as a warden who does not decode it sees it.
type Trace struct {
	Samples       int     `json:"samples"`
	ZeroTail      int     `json:"zero_tail"`
	SampleFmt     string  `json:"sample_fmt"`
	NominalKbps   float64 `json:"nominal_kbps"`
	Kbps          float64 `json:"kbps"`
	Bytes         int     `json:"bytes,omitempty"`
	Codec         string  `json:"codec,omitempty"`
	Format        string  `json:"format,omitempty"`
	Channels      int     `json:"channels,omitempty"`
	SampleRate    int     `json:"sample_rate,omitempty"`
	Bits          int     `json:"bits,omitempty"`
	ChannelLayout string  `json:"channel_layout,omitempty"`
	TimeBase      string  `json:"time_base,omitempty"`
	Encoder       string  `json:"encoder,omitempty"`
	TagOrder      string  `json:"tag_order,omitempty"`
	CodecTag      string  `json:"codec_tag,omitempty"`
	Profile       string  `json:"profile,omitempty"`
	Streams       int     `json:"streams,omitempty"`
	Packets       int     `json:"packets,omitempty"`
	MeanPacket    int     `json:"mean_packet,omitempty"`
	SkipStart     int     `json:"skip_start,omitempty"`
	Probe         string  `json:"probe,omitempty"`
	Flac          bool    `json:"flac,omitempty"`
	FlacSamples   int64   `json:"flac_samples,omitempty"`
	FlacMD5       bool    `json:"flac_md5,omitempty"`
	FlacBlocks    int     `json:"flac_blocks,omitempty"`
	Padding       int     `json:"padding,omitempty"`
	UnknownMeta   int     `json:"unknown_meta,omitempty"`
	OggPages      int     `json:"ogg_pages,omitempty"`
	OggSerials    int     `json:"ogg_serials,omitempty"`
	OggMeanPage   int     `json:"ogg_mean_page,omitempty"`
	OggMeanLace   int     `json:"ogg_mean_lace,omitempty"`
	OggGranule    uint64  `json:"ogg_granule,omitempty"`
	OggFirst      uint64  `json:"ogg_first_granule,omitempty"`
}

// Decoded is what Of needs from a decode of the file: the planes, the
// rate they play at, and what libav reported about the stream.
type Decoded struct {
	Planes     [][]float32
	SampleRate int
	SampleFmt  string
	Bitrate    int64
}

// Of traces the file b decoded as d, asking the ffprobe binary for the
// fields only a demuxer reports. A probe failure is returned alongside a
// trace that holds everything else, with Probe set to "unavailable".
func Of(ctx context.Context, ffprobe string, b []byte, d Decoded) (Trace, error) {
	t := Trace{
		ZeroTail:    zeroTail(d.Planes),
		SampleFmt:   d.SampleFmt,
		NominalKbps: float64(d.Bitrate) / 1000,
		Bytes:       len(b),
	}
	if len(d.Planes) > 0 {
		t.Samples = len(d.Planes[0])
	}
	if t.Samples > 0 && d.SampleRate > 0 {
		t.Kbps = float64(len(b)) * 8 / 1000 / (float64(t.Samples) / float64(d.SampleRate))
	}
	summarizeOgg(b, &t)
	summarizeFlac(b, &t)
	if err := probe(ctx, ffprobe, b, &t); err != nil {
		t.Probe = "unavailable"
		return t, err
	}
	return t, nil
}

// zeroTail counts the trailing samples that are digital zero on every
// channel, which is what padding leaves behind.
func zeroTail(planes [][]float32) int {
	if len(planes) == 0 {
		return 0
	}
	n := 0
	for i := len(planes[0]) - 1; i >= 0; i-- {
		for _, p := range planes {
			if i < len(p) && p[i] != 0 {
				return n
			}
		}
		n++
	}
	return n
}

// Diff names identity fields that should match under one workflow.
// File size, packet size and page size stay out: VBR and embedding move
// them, and the metadata classifier is what scores those. Ogg serial
// values stay out because each encode draws a new one.
func Diff(a, b Trace) []string {
	var out []string
	if a.Samples != b.Samples {
		out = append(out, "length")
	}
	if a.ZeroTail != b.ZeroTail {
		out = append(out, "zero tail")
	}
	if a.SampleFmt != b.SampleFmt {
		out = append(out, "sample format")
	}
	if math.Abs(a.NominalKbps-b.NominalKbps) > nominalSlack*max(a.NominalKbps, b.NominalKbps) {
		out = append(out, "nominal bitrate")
	}
	differString("codec", a.Codec, b.Codec, &out)
	differString("container", a.Format, b.Format, &out)
	differInt("channels", a.Channels, b.Channels, &out)
	differInt("sample rate", a.SampleRate, b.SampleRate, &out)
	differInt("raw depth", a.Bits, b.Bits, &out)
	differString("channel layout", a.ChannelLayout, b.ChannelLayout, &out)
	differString("timebase", a.TimeBase, b.TimeBase, &out)
	differString("encoder tag", a.Encoder, b.Encoder, &out)
	differString("metadata tags", a.TagOrder, b.TagOrder, &out)
	differString("codec tag", a.CodecTag, b.CodecTag, &out)
	differString("profile", a.Profile, b.Profile, &out)
	differInt("stream count", a.Streams, b.Streams, &out)
	differInt("skip samples", a.SkipStart, b.SkipStart, &out)
	if a.Flac || b.Flac {
		if a.FlacMD5 != b.FlacMD5 {
			out = append(out, "flac md5")
		}
		if a.FlacSamples != b.FlacSamples {
			out = append(out, "flac sample count")
		}
	}
	if a.UnknownMeta != b.UnknownMeta && (a.UnknownMeta > 0 || b.UnknownMeta > 0) {
		out = append(out, "unknown metadata")
	}
	if a.OggPages > 0 || b.OggPages > 0 {
		if a.OggSerials != b.OggSerials {
			out = append(out, "ogg serials")
		}
		if a.OggGranule != b.OggGranule {
			out = append(out, "ogg granule")
		}
	}
	return out
}

func differString(name, a, b string, out *[]string) {
	if a == "" && b == "" {
		return
	}
	if a != b {
		*out = append(*out, name)
	}
}

func differInt(name string, a, b int, out *[]string) {
	if a == 0 && b == 0 {
		return
	}
	if a != b {
		*out = append(*out, name)
	}
}

func summarizeOgg(b []byte, t *Trace) {
	if !bytes.HasPrefix(b, []byte("OggS")) {
		return
	}
	serials := map[uint32]struct{}{}
	var bodies, laces int
	var first, last uint64
	saw := false
	off := 0
	for off+27 <= len(b) && string(b[off:off+4]) == "OggS" {
		granule := binary.LittleEndian.Uint64(b[off+6 : off+14])
		serial := binary.LittleEndian.Uint32(b[off+14 : off+18])
		nseg := int(b[off+26])
		if off+27+nseg > len(b) {
			break
		}
		body := 0
		for i := range nseg {
			body += int(b[off+27+i])
		}
		if off+27+nseg+body > len(b) {
			break
		}
		serials[serial] = struct{}{}
		t.OggPages++
		bodies += body
		laces += nseg
		if granule != ^uint64(0) {
			if !saw {
				first = granule
				saw = true
			}
			last = granule
		}
		off += 27 + nseg + body
	}
	t.OggSerials = len(serials)
	if t.OggPages > 0 {
		t.OggMeanPage = bodies / t.OggPages
		t.OggMeanLace = laces / t.OggPages
	}
	if saw {
		t.OggFirst = first
		t.OggGranule = last
	}
}

func summarizeFlac(b []byte, t *Trace) {
	if !bytes.HasPrefix(b, []byte("fLaC")) {
		return
	}
	t.Flac = true
	off := 4
	for off+4 <= len(b) {
		hdr := binary.BigEndian.Uint32(b[off : off+4])
		last := hdr&0x80000000 != 0
		typ := int((hdr >> 24) & 0x7f)
		size := int(hdr & 0xffffff)
		off += 4
		if off+size > len(b) {
			return
		}
		body := b[off : off+size]
		off += size
		switch typ {
		case 0:
			if len(body) >= 34 {
				t.FlacSamples = int64(bitsAt(body, 108, 36))
				t.FlacMD5 = nonzero(body[18:34])
			}
		case 1:
			t.Padding += size
		default:
			if typ > 6 {
				t.UnknownMeta++
			} else {
				t.FlacBlocks++
			}
		}
		if last {
			return
		}
	}
}

func bitsAt(b []byte, bit, n int) uint64 {
	var v uint64
	for i := range n {
		p := bit + i
		if p/8 >= len(b) {
			return v
		}
		v = (v << 1) | uint64((b[p/8]>>(7-uint(p%8)))&1)
	}
	return v
}

func nonzero(b []byte) bool {
	for _, c := range b {
		if c != 0 {
			return true
		}
	}
	return false
}

// Encode runs the ffmpeg CLI over data, a file with extension ext, keeping
// only the first audio stream so cover art does not become a video stream.
// codec may be empty for the container's default encoder; extra goes in
// before the output, which is written as container. This is the plain
// encode a warden without Mist compares a file against.
func Encode(ctx context.Context, ffmpeg string, data []byte, ext, container, codec string, extra ...string) ([]byte, error) {
	dir, err := os.MkdirTemp("", "mist-twin-*")
	if err != nil {
		return nil, err
	}
	defer func() { _ = os.RemoveAll(dir) }()
	in, out := filepath.Join(dir, "carrier"+ext), filepath.Join(dir, "twin")
	if err := os.WriteFile(in, data, 0o600); err != nil {
		return nil, err
	}
	args := []string{"-nostdin", "-loglevel", "error", "-i", in, "-map", "0:a:0"}
	if codec != "" {
		args = append(args, "-c:a", codec)
	}
	args = append(append(args, extra...), "-f", container, out)
	if msg, err := exec.CommandContext(ctx, ffmpeg, args...).CombinedOutput(); err != nil {
		return nil, fmt.Errorf("ffmpeg: %v: %s", err, bytes.TrimSpace(msg))
	}
	return os.ReadFile(out)
}

// Ext is the file extension the leading magic bytes of b imply, for tools
// that pick a demuxer by name.
func Ext(b []byte) string {
	switch {
	case bytes.HasPrefix(b, []byte("OggS")):
		return ".ogg"
	case bytes.HasPrefix(b, []byte("fLaC")):
		return ".flac"
	case bytes.HasPrefix(b, []byte("RIFF")):
		return ".wav"
	case bytes.HasPrefix(b, []byte("FORM")):
		return ".aiff"
	default:
		return ".bin"
	}
}

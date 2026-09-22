package mist

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/iSerganov/mist/internal/av"
	"github.com/iSerganov/mist/internal/codec"
	"github.com/iSerganov/mist/internal/codec/vorbis"
	"github.com/iSerganov/mist/internal/frame"
	"github.com/iSerganov/mist/internal/wire"
)

// envelopeFloor is the smallest room, in bytes, a frame needs before it can
// carry a whole, unspanned payload: the crypto envelope plus the wire
// payload header. It does not depend on the recipient key's value, only
// its size, so estimating capacity needs no key at all.
const envelopeFloor = EnvelopeOverhead + wire.PayloadHeaderSize

// Source describes the carrier Embed would decode, straight from what the
// installed libav reports about it — nothing here is Mist's own analysis.
type Source struct {
	Codec      string // libav's display name for the source codec
	Container  string // demuxer's name for the source container
	SampleRate int
	Channels   int
	Bitrate    int64 // bits per second as the source reports it, 0 if it does not
	Duration   time.Duration
	Lossless   bool // whether libav's codec descriptor marks the source lossless
}

// Capacity reports how much of a carrier Embed could use for a given
// target format, without producing any output.
type Capacity struct {
	Source Source
	Format Format
	// Frames is the number of stego frames the carrier splits into.
	Frames int
	// Usable is how many of those frames have room to contribute to
	// TotalCapacity — the protocol envelope plus at least one byte, paid
	// once by whichever usable frame Embed writes first.
	Usable int
	// FrameCapacity is the largest plaintext, in bytes, a single unspanned
	// frame could carry — what Embed uses when the payload fits in one
	// frame outright.
	FrameCapacity int
	// TotalCapacity is the largest plaintext, in bytes, Embed could carry
	// overall by spanning the payload across every usable frame in turn
	// when it does not fit in just one.
	TotalCapacity int
}

// EstimateCapacity decodes source the way Embed would for the target
// picked by formatName/codecName (LookupFormat's rules) and reports how
// large a text payload Embed could carry into it, without writing
// anything out. It costs what Embed costs: the carrier still has to be
// decoded and, for a Vorbis target, re-encoded, because capacity depends
// on what the target encoder actually produces, not on the source.
func EstimateCapacity(ctx context.Context, source, formatName, codecName string) (Capacity, error) {
	if err := ctx.Err(); err != nil {
		return Capacity{}, err
	}
	target, err := lookupFormat(formatName, codecName)
	if err != nil {
		return Capacity{}, err
	}
	pcm, info, err := decodeSource(source)
	if err != nil {
		return Capacity{}, err
	}
	return estimatePCM(ctx, target, pcm, info)
}

func decodeSource(source string) (codec.PCM, av.AudioInfo, error) {
	if strings.HasPrefix(source, "http://") || strings.HasPrefix(source, "https://") {
		return decodeCarrierURL(source)
	}
	rc, err := openSource(source)
	if err != nil {
		return codec.PCM{}, av.AudioInfo{}, err
	}
	defer func() { _ = rc.Close() }()
	return decodeCarrier(rc)
}

func estimatePCM(ctx context.Context, target av.Format, pcm codec.PCM, info av.AudioInfo) (Capacity, error) {
	enc, err := av.NewEncoder(target.Info(pcm.SampleRate, pcm.Channels, targetBitrate(target, info.Params())))
	if err != nil {
		return Capacity{}, fmt.Errorf("%w: encoder: %v", ErrCarrier, err)
	}
	defer func() { _ = enc.Close() }()

	var rooms []int
	if target.Lossless {
		pcm = padToWindow(pcm, enc.Window())
		rooms = sampleRooms(sampleFrames(pcm, av.SampleScale(enc.Info().SampleFmt)))
	} else if rooms, err = residueRooms(ctx, enc, pcm); err != nil {
		return Capacity{}, err
	}
	c := tabulateRooms(target, rooms)
	c.Source = sourceOf(info)
	return c, nil
}

func sourceOf(info av.AudioInfo) Source {
	return Source{
		Codec:      info.CodecName,
		Container:  info.Container,
		SampleRate: info.SampleRate,
		Channels:   info.Channels,
		Bitrate:    info.Bitrate,
		Duration:   time.Duration(info.DurationUs) * time.Microsecond,
		Lossless:   av.Lossless(info.NativeCodecID),
	}
}

// residueRooms is the Vorbis path: only the encoder's actual residues say
// how many are eligible, so the carrier is encoded for real.
func residueRooms(ctx context.Context, enc *av.Encoder, pcm codec.PCM) ([]int, error) {
	pkts, err := enc.Encode(pcm)
	if err != nil {
		return nil, err
	}
	flushed, err := enc.Flush()
	if err != nil {
		return nil, err
	}
	vc := vorbis.New()
	if err := vc.Load(enc.Info().Extradata); err != nil {
		return nil, fmt.Errorf("%w: setup", err)
	}
	groups := frame.GroupPackets(append(pkts, flushed...), frameParams(pcm))
	if len(groups) == 0 {
		return nil, ErrCarrier
	}
	return groupRooms(ctx, vc, groups)
}

// tabulateRooms reports both the single largest frame's capacity
// (FrameCapacity, what Embed uses when the payload fits in one frame) and
// the total capacity spanning every usable frame would offer
// (TotalCapacity) — the same accounting planSpan does, without needing an
// actual payload to run it against: the first usable frame pays the
// larger span-start header, every one after it the smaller continuation
// header.
func tabulateRooms(target av.Format, rooms []int) Capacity {
	c := Capacity{
		Format: Format{Container: target.Container, Codec: target.CodecName, Ext: target.Ext, Lossless: target.Lossless},
		Frames: len(rooms),
	}
	spanHeader := wire.SpanStartHeaderSize
	for _, room := range rooms {
		if single := room - envelopeFloor; single > c.FrameCapacity {
			c.FrameCapacity = single
		}
		avail := room - EnvelopeOverhead - spanHeader
		if avail <= 0 {
			continue
		}
		c.Usable++
		c.TotalCapacity += avail
		spanHeader = wire.SpanContinueHeaderSize
	}
	return c
}

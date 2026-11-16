package mist

import (
	"context"
	"time"

	"github.com/iSerganov/mist/internal/av"
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
	// FrameCapacity is the largest plaintext, in bytes, Embed writes into
	// one frame unspanned. That is the first frame with room: a payload too
	// big for it is spanned from there, even if a later frame is larger.
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
	in, err := openPath(ctx, source)
	if err != nil {
		return Capacity{}, err
	}
	info := in.d.Info()
	rooms, err := (&Emitter{target: target}).measure(ctx, in)
	if err != nil {
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

// tabulateRooms reports both what Embed writes unspanned (FrameCapacity)
// and the total capacity spanning every usable frame would offer
// (TotalCapacity) — the same accounting spanPlan does, without needing an
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
		if c.Usable == 0 {
			c.FrameCapacity = max(c.FrameCapacity, room-envelopeFloor)
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

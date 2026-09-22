package mist

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/iSerganov/mist/internal/av"
	"github.com/iSerganov/mist/internal/codec"
	"github.com/iSerganov/mist/internal/codec/vorbis"
	"github.com/iSerganov/mist/internal/frame"
	"github.com/iSerganov/mist/internal/stego"
	"github.com/iSerganov/mist/internal/wire"
)

// envelopeFloor is the smallest room, in bytes, a frame needs before it can
// carry even one byte of plaintext: the crypto envelope plus the wire
// payload header. It does not depend on the recipient key's value, only its
// size, so estimating capacity needs no key at all.
const envelopeFloor = EnvelopeOverhead + wire.PayloadHeaderSize

// Capacity reports how much of a carrier Embed could use for a given
// target format, without producing any output.
type Capacity struct {
	Format Format
	// Frames is the number of stego frames the carrier splits into.
	Frames int
	// Usable is how many of those frames have room for the protocol
	// envelope plus at least one byte of plaintext.
	Usable int
	// MaxPayload is the largest plaintext, in bytes, a single Embed call
	// could carry — the room in the frame with the most space, since the
	// payload rides in the first frame that fits it.
	MaxPayload int
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
	pcm, params, err := decodeSource(source)
	if err != nil {
		return Capacity{}, err
	}
	return estimatePCM(ctx, target, pcm, params)
}

func decodeSource(source string) (codec.PCM, codec.Params, error) {
	if strings.HasPrefix(source, "http://") || strings.HasPrefix(source, "https://") {
		return decodeCarrierURL(source)
	}
	rc, err := openSource(source)
	if err != nil {
		return codec.PCM{}, codec.Params{}, err
	}
	defer func() { _ = rc.Close() }()
	return decodeCarrier(rc)
}

func estimatePCM(ctx context.Context, target av.Format, pcm codec.PCM, params codec.Params) (Capacity, error) {
	enc, err := av.NewEncoder(target.Info(pcm.SampleRate, pcm.Channels, targetBitrate(target, params)))
	if err != nil {
		return Capacity{}, fmt.Errorf("%w: encoder: %v", ErrCarrier, err)
	}
	defer func() { _ = enc.Close() }()

	var rooms []int
	if target.Lossless {
		pcm = padToWindow(pcm, enc.Window())
		rooms = sampleRooms(pcm, av.SampleScale(enc.Info().SampleFmt))
	} else if rooms, err = residueRooms(ctx, enc, pcm); err != nil {
		return Capacity{}, err
	}
	return tabulateRooms(target, rooms), nil
}

// sampleRooms is the lossless path: capacity per window is arithmetic over
// the sample count, so no actual encode is needed to know it.
func sampleRooms(pcm codec.PCM, scale float32) []int {
	frames := sampleFrames(pcm, scale)
	rooms := make([]int, len(frames))
	for i, w := range frames {
		rooms[i] = w.Capacity()
	}
	return rooms
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
	rooms := make([]int, len(groups))
	for i, g := range groups {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		room, err := stego.Capacity(vc, g.Packets)
		if errors.Is(err, stego.ErrNoResidues) {
			continue
		}
		if err != nil {
			return nil, fmt.Errorf("%w: %v", ErrCarrier, err)
		}
		rooms[i] = room
	}
	return rooms, nil
}

func tabulateRooms(target av.Format, rooms []int) Capacity {
	c := Capacity{
		Format: Format{Container: target.Container, Codec: target.CodecName, Ext: target.Ext, Lossless: target.Lossless},
		Frames: len(rooms),
	}
	for _, room := range rooms {
		if room < envelopeFloor {
			continue
		}
		c.Usable++
		if payload := room - envelopeFloor; payload > c.MaxPayload {
			c.MaxPayload = payload
		}
	}
	return c
}

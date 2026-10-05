//go:build harness

package mist

import (
	"bytes"
	"errors"
	"fmt"

	"github.com/iSerganov/mist/internal/av"
	"github.com/iSerganov/mist/internal/codec"
	"github.com/iSerganov/mist/internal/codec/vorbis"
	"github.com/iSerganov/mist/internal/crypto"
	"github.com/iSerganov/mist/internal/frame"
	"github.com/iSerganov/mist/internal/steganalysis"
	"github.com/iSerganov/mist/internal/stego"
	"github.com/iSerganov/mist/internal/wire"
)

const keyAwareName = "key-aware"

// keyAwareScore is the warden who knows the recipient's public key but not
// the private one. Position seeds come from the public key alone, so it
// recovers the first frame's bits exactly as the catcher does and looks at
// the envelope's leading ephemeral key without opening anything. The score
// is 1 when those 32 bytes look like an honest X25519 public key and 0
// when they look like noise, or when no frame is big enough to hold an
// envelope, as in a clean file at a low rate. The payload goes in the first frame with room,
// so that is the frame to read.
func keyAwareScore(data, pub []byte) (float64, error) {
	env, err := firstEnvelope(data, pub)
	if errors.Is(err, errNoEnvelope) {
		return 0, nil
	}
	if err != nil {
		return 0, err
	}
	if steganalysis.HonestX25519(env.EphemeralPub[:]) {
		return 1, nil
	}
	return 0, nil
}

var errNoEnvelope = errors.New("no frame had room for an envelope")

func firstEnvelope(data, pub []byte) (wire.Envelope, error) {
	d, err := av.OpenDemuxerReader(bytes.NewReader(data))
	if err != nil {
		return wire.Envelope{}, err
	}
	defer func() { _ = d.Close() }()
	info := d.Info()
	p := info.Params()
	fp := frame.Params{SampleRate: p.SampleRate, Channels: p.Channels, Duration: FrameDuration}

	var read func(av.Packet, bool) (wire.Envelope, bool, error)
	switch {
	case info.CodecID == av.CodecIDVorbis:
		vc := vorbis.New()
		if err := vc.Load(info.Extradata); err != nil {
			return wire.Envelope{}, err
		}
		grouper := frame.NewGrouper(fp)
		read = func(pkt av.Packet, final bool) (wire.Envelope, bool, error) {
			g, ok := frame.Group{}, false
			if final {
				g, ok = grouper.Flush()
			} else {
				g, ok = grouper.Push(av.ToCodecPacket(pkt))
			}
			if !ok || g.Partial {
				return wire.Envelope{}, false, nil
			}
			seed, err := crypto.PositionSeed(pub, g.Index)
			if err != nil {
				return wire.Envelope{}, false, err
			}
			bits, err := stego.Recover(vc, seed, g.Packets)
			return envelopeOf(bits, err)
		}
	case av.Lossless(info.NativeCodecID):
		dec, err := av.NewDecoder(info)
		if err != nil {
			return wire.Envelope{}, err
		}
		defer func() { _ = dec.Close() }()
		win := newWindower(fp, av.SampleScale(info.SampleFmt))
		windows := func(frames []sampleFrame) (wire.Envelope, bool, error) {
			for _, w := range frames {
				if w.partial {
					continue
				}
				seed, err := crypto.PositionSeed(pub, w.index)
				if err != nil {
					return wire.Envelope{}, false, err
				}
				bits, err := stego.RecoverSamples(w.Samples, seed)
				if env, ok, err := envelopeOf(bits, err); ok || err != nil {
					return env, ok, err
				}
			}
			return wire.Envelope{}, false, nil
		}
		read = func(pkt av.Packet, final bool) (wire.Envelope, bool, error) {
			var pcm []codec.PCM
			if final {
				_ = dec.Send(av.Packet{})
				if pcm, err = av.DrainPCM(dec); err != nil {
					return wire.Envelope{}, false, err
				}
			} else if pcm, err = dec.Decode(av.ToCodecPacket(pkt)); err != nil {
				return wire.Envelope{}, false, err
			}
			for _, c := range pcm {
				if env, ok, err := windows(win.push(c)); ok || err != nil {
					return env, ok, err
				}
			}
			if final {
				return windows(win.flush())
			}
			return wire.Envelope{}, false, nil
		}
	default:
		return wire.Envelope{}, fmt.Errorf("%w: %s", ErrUnsupportedCodec, codecName(info))
	}

	for {
		pkt, err := d.NextPacket()
		final := errors.Is(err, av.ErrEOF)
		if err != nil && !final {
			return wire.Envelope{}, err
		}
		if env, ok, err := read(pkt, final); ok || err != nil {
			return env, err
		}
		if final {
			return wire.Envelope{}, errNoEnvelope
		}
	}
}

// envelopeOf skips a frame with no usable bits, which is the same silent
// skip the catcher makes, and reports only genuine failures.
func envelopeOf(bits stego.Bits, err error) (wire.Envelope, bool, error) {
	if err != nil {
		return wire.Envelope{}, false, nil
	}
	env, err := wire.UnmarshalEnvelope(bits)
	return env, err == nil, nil
}

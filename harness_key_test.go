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

// selectionScore is the warden who knows the recipient public key and reads
// the LSB gap at the positions that key implies. The first chunk stands in
// for frame 0: the harness value stream is not split on the same silence
// mask Embed uses, so this is the operational approximation, not the
// sender's private cost. A cover the warden does not hold is not an input.
func selectionScore(vals []int32, pub []byte) (float64, error) {
	if len(vals) == 0 {
		return 0, nil
	}
	window := vals
	if len(window) > harnessChunk {
		window = window[:harnessChunk]
	}
	seed, err := crypto.PositionSeed(pub, 0)
	if err != nil {
		return 0, err
	}
	at := stego.NewSelector(seed, len(window)).Pick(stego.Covered(len(window)))
	return steganalysis.ParityGap(window, at), nil
}

func (s *HarnessSuite) TestSelectionScoreSeesAPlantedBias() {
	pub, _, err := crypto.GenerateX25519()
	s.Require().NoError(err)
	vals := make([]int32, 4000)
	for i := range vals {
		vals[i] = int32(i % 7)
	}
	clean, err := selectionScore(vals, pub)
	s.Require().NoError(err)
	seed, err := crypto.PositionSeed(pub, 0)
	s.Require().NoError(err)
	for _, i := range stego.NewSelector(seed, len(vals)).Pick(stego.Covered(len(vals))) {
		vals[i] |= 1
	}
	forced, err := selectionScore(vals, pub)
	s.Require().NoError(err)
	s.Greater(forced, clean+0.15)
}

func (s *HarnessSuite) TestEnvelopeRepIsNotAnHonestKey() {
	pub, priv, err := crypto.GenerateX25519()
	s.Require().NoError(err)
	_, other, err := crypto.GenerateX25519()
	s.Require().NoError(err)
	var top, hits int
	seen := map[[32]byte]bool{}
	const seals = 64
	for range seals {
		env, err := crypto.Seal([]byte("mist"), pub)
		s.Require().NoError(err)
		if steganalysis.HonestX25519(env.EphemeralPub[:]) {
			hits++
		}
		if env.EphemeralPub[31]&0x80 != 0 {
			top++
		}
		s.False(seen[env.EphemeralPub])
		seen[env.EphemeralPub] = true
		_, err = crypto.Open(env, other)
		s.ErrorIs(err, crypto.ErrOpen)
		_, err = crypto.Open(env, priv)
		s.NoError(err)
	}
	// A random 32-byte string passes about one time in 32, so a handful of
	// hits is the floor of that test, not evidence the representative is a key.
	s.Less(hits, seals/8)
	s.NotZero(top)
	a, err := crypto.PositionSeed(pub, 0)
	s.Require().NoError(err)
	b, err := crypto.PositionSeed(pub, 1)
	s.Require().NoError(err)
	s.NotEqual(a, b)
	otherPub, _, err := crypto.GenerateX25519()
	s.Require().NoError(err)
	left, err := crypto.Seal([]byte("mist"), pub)
	s.Require().NoError(err)
	right, err := crypto.Seal([]byte("mist"), otherPub)
	s.Require().NoError(err)
	s.NotEqual(left.EphemeralPub, right.EphemeralPub)
	_, signPriv, err := crypto.GenerateEd25519()
	s.Require().NoError(err)
	sig, err := crypto.Sign(signPriv, []byte("mist"))
	s.Require().NoError(err)
	signed, err := wire.MarshalPayload(wire.Payload{Version: wire.CurrentVersion, Type: 0x01, Data: []byte("mist"), Signature: sig})
	s.Require().NoError(err)
	env, err := crypto.Seal(signed, pub)
	s.Require().NoError(err)
	s.False(steganalysis.HonestX25519(env.EphemeralPub[:]))
	start := wire.MarshalSpanStart(64, []byte("chunk"))
	cont := wire.MarshalSpanContinue([]byte("chunk"))
	s.NotEqual(wire.CurrentVersion, start[0])
	s.NotEqual(start[0], cont[0])
}

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

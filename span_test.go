package mist

import (
	"bytes"
	"context"
	"crypto/rand"
	"fmt"
	"testing"
	"time"

	"github.com/iSerganov/mist/internal/av"
	"github.com/iSerganov/mist/internal/crypto"
	"github.com/iSerganov/mist/internal/stego"
	"github.com/iSerganov/mist/internal/wire"
	"github.com/stretchr/testify/suite"
)

type SpanSuite struct {
	audioSuite
}

func TestSpanSuite(t *testing.T) { suite.Run(t, new(SpanSuite)) }

func (s *SpanSuite) requireFormat(name string) {
	if !av.Available() {
		s.T().Skip("libav not available")
	}
	if _, err := LookupFormat(name, ""); err != nil {
		s.T().Skipf("this FFmpeg cannot write %s: %v", name, err)
	}
}

// marshaledText is the wire-framed plaintext a text Payload turns into
// before sealing — the byte stream planChunks/opener actually work with.
func (s *SpanSuite) marshaledText(text string) []byte {
	full, err := marshalPlain(Text(text), nil)
	s.Require().NoError(err)
	return full
}

// planChunks always prefers a single, unmodified frame over spanning, so
// the common case pays no span overhead at all.
func (s *SpanSuite) TestPlanChunksPrefersASingleFrame() {
	full := bytes.Repeat([]byte{0xAB}, 10)
	room := []int{100, 5, 100}
	plan := planChunks(full, room)
	s.Require().NotNil(plan)
	s.Equal(full, plan[0])
	s.Nil(plan[1])
	s.Nil(plan[2])
}

// A single-fit search still looks for the first frame that alone has
// room, skipping smaller ones ahead of it.
func (s *SpanSuite) TestPlanChunksSkipsTooSmallFramesForASingleFit() {
	full := bytes.Repeat([]byte{0xCD}, 90)
	room := []int{100, 200} // needs EnvelopeOverhead(64)+90 = 154
	plan := planChunks(full, room)
	s.Require().NotNil(plan)
	s.Nil(plan[0])
	s.Equal(full, plan[1])
}

// When no single frame is large enough, planChunks spans the payload
// across as many consecutive frames as it takes, opening with
// MarshalSpanStart and continuing with MarshalSpanContinue.
func (s *SpanSuite) TestPlanChunksSpansWhenNoSingleFrameFits() {
	full := bytes.Repeat([]byte{0xEF}, 300)
	room := []int{200, 200, 200} // none alone clears EnvelopeOverhead+300=364
	plan := planChunks(full, room)
	s.Require().NotNil(plan)

	totalLen, chunk0, ok := wire.UnmarshalSpanStart(plan[0])
	s.Require().True(ok)
	s.Equal(uint32(len(full)), totalLen)

	chunk1, ok := wire.UnmarshalSpanContinue(plan[1])
	s.Require().True(ok)
	chunk2, ok := wire.UnmarshalSpanContinue(plan[2])
	s.Require().True(ok)

	reassembled := append(append(append([]byte{}, chunk0...), chunk1...), chunk2...)
	s.Equal(full, reassembled)
}

// A frame with no room at all is skipped over during spanning, the same
// as one with no eligible residues at all — it neither starts nor
// continues the span.
func (s *SpanSuite) TestPlanChunksSkipsAZeroRoomFrameMidSpan() {
	full := bytes.Repeat([]byte{0x11}, 90)
	room := []int{100, 0, 100, 100} // no single frame clears 64+90=154
	plan := planChunks(full, room)
	s.Require().NotNil(plan)
	s.Nil(plan[1], "the zero-room frame must carry nothing")

	totalLen, chunk0, ok := wire.UnmarshalSpanStart(plan[0])
	s.Require().True(ok)
	s.Equal(uint32(len(full)), totalLen)
	chunk2, ok := wire.UnmarshalSpanContinue(plan[2])
	s.Require().True(ok)
	chunk3, ok := wire.UnmarshalSpanContinue(plan[3])
	s.Require().True(ok)

	reassembled := append(append(append([]byte{}, chunk0...), chunk2...), chunk3...)
	s.Equal(full, reassembled)
}

// A payload that fits nowhere, even split across every frame offered, is
// reported as such — the caller turns that into ErrNoCapacity.
func (s *SpanSuite) TestPlanChunksFailsWhenNothingFits() {
	full := bytes.Repeat([]byte{0x22}, 1000)
	room := []int{50, 50}
	s.Nil(planChunks(full, room))
}

// opener reassembles a payload from consecutive start/continue chunks,
// tested directly against already-decrypted plaintext so it does not
// need real crypto or audio to exercise the state machine.
func (s *SpanSuite) TestOpenerReassemblesASpan() {
	full := s.marshaledText("a longer message than one chunk can hold")

	var o opener
	res, ok := o.start(0, wire.MarshalSpanStart(uint32(len(full)), full[:10]))
	s.False(ok, "the span is not complete yet")
	s.Equal(Result{}, res)

	_, ok = o.resume(1, wire.MarshalSpanContinue(full[10:25]))
	s.False(ok)

	res, ok = o.resume(2, wire.MarshalSpanContinue(full[25:]))
	s.Require().True(ok)
	s.Equal(int64(0), res.FrameIdx, "the result names the frame the span started in")
	s.Equal("a longer message than one chunk can hold", string(res.Payload.Data))
}

// A frame that authenticates but does not carry a valid continuation
// marker means the span was interrupted — the partial assembly must be
// discarded, never handed out, and must not leak into whatever comes
// next.
func (s *SpanSuite) TestOpenerDiscardsAnInterruptedSpan() {
	full := s.marshaledText("hello there, this needs more than one chunk")

	var o opener
	_, ok := o.start(0, wire.MarshalSpanStart(uint32(len(full)), full[:5]))
	s.False(ok)
	s.True(o.spanning)

	// Not a valid continuation chunk: the span is abandoned rather than
	// silently misinterpreted.
	res, ok := o.resume(1, []byte("not a continuation"))
	s.False(ok)
	s.Equal(Result{}, res)
	s.False(o.spanning, "assembly state must be cleared, not left dangling")

	// A fresh, complete payload right after must work normally.
	fresh := s.marshaledText("fresh message")
	res, ok = o.start(2, fresh)
	s.Require().True(ok)
	s.Equal("fresh message", string(res.Payload.Data))
}

func (s *SpanSuite) TestOpenerHandlesAnUnspannedPayloadUnchanged() {
	full := s.marshaledText("fits in one frame")

	var o opener
	res, ok := o.start(3, full)
	s.Require().True(ok)
	s.False(o.spanning)
	s.Equal(int64(3), res.FrameIdx)
	s.Equal("fits in one frame", string(res.Payload.Data))
}

// A message too large for any single Vorbis frame round-trips through a
// real Embed/Extract by spanning several.
func (s *SpanSuite) TestEmbedExtractSpansAcrossVorbisFrames() {
	s.requireLibav()
	pub, priv, err := GenerateKeyPair()
	s.Require().NoError(err)

	msg := string(bytes.Repeat([]byte("the eagle lands at dawn, "), 24)) // 600 bytes, three frames or more
	out := s.stego(pub, Text(msg), s.pcmCarrier(4*FrameDuration))
	res := s.extract(priv, out)

	s.Require().Len(res, 1, "one reassembled message, however many frames it took")
	s.Equal(msg, string(res[0].Payload.Data))
}

// The same, for a lossless target — cheaper to reach since capacity there
// is far larger relative to a single frame.
func (s *SpanSuite) TestEmbedExtractSpansAcrossLosslessFrames() {
	s.requireFormat("flac")
	pub, priv, err := GenerateKeyPair()
	s.Require().NoError(err)

	msg := string(bytes.Repeat([]byte("0123456789"), 200)) // 2000 bytes, three frames
	out := s.stego(pub, Text(msg), s.pcmCarrier(4*FrameDuration), WithFormat("flac"))
	res := s.extract(priv, out)

	s.Require().Len(res, 1)
	s.Equal(msg, string(res[0].Payload.Data))
}

// A message too large even split across every usable frame still fails
// loudly, exactly as a single-frame overflow always has.
func (s *SpanSuite) TestEmbedFailsWhenSpanningStillIsNotEnough() {
	s.requireLibav()
	pub, _, err := GenerateKeyPair()
	s.Require().NoError(err)

	em, err := NewEmitter(pub)
	s.Require().NoError(err)
	huge := string(bytes.Repeat([]byte{0x41}, 200_000))
	_, err = em.EmbedReader(context.Background(), bytes.NewReader(s.pcmCarrier(4*FrameDuration)), Text(huge))
	s.Require().ErrorIs(err, ErrNoCapacity)
}

// Every payload type comes back with its type and bytes intact, and a wrong
// key recovers nothing. Spanning is type-blind and covered above.
func (s *SpanSuite) TestTypedPayloadsRoundTrip() {
	random := func(n int) []byte {
		b := make([]byte, n)
		_, err := rand.Read(b)
		s.Require().NoError(err)
		return b
	}
	file := func(name string, data []byte) Payload {
		p, err := File(name, data)
		s.Require().NoError(err)
		return p
	}
	tests := []struct {
		title   string
		format  string
		payload Payload
		seconds int
	}{
		{"named file in flac", "flac", file("quarterly/report.pdf", random(200)), 9},
		{"image in wav", "wav", Image(random(200)), 9},
		{"audio in wav", "wav", Audio(random(200)), 9},
		{"image in vorbis", "", Image(random(150)), 32},
	}
	for _, tc := range tests {
		s.Run(tc.title, func() {
			if tc.format == "" {
				s.requireLibav()
			} else {
				s.requireFormat(tc.format)
			}
			pub, priv, err := GenerateKeyPair()
			s.Require().NoError(err)
			var opts []EmitterOption
			if tc.format != "" {
				opts = append(opts, WithFormat(tc.format))
			}
			out := s.stego(pub, tc.payload, s.pcmCarrier(time.Duration(tc.seconds)*time.Second), opts...)

			res := s.extract(priv, out)
			s.Require().Len(res, 1)
			s.Equal(tc.payload.Type, res[0].Payload.Type)
			s.Equal(tc.payload.Data, res[0].Payload.Data)

			_, wrongPriv, err := GenerateKeyPair()
			s.Require().NoError(err)
			s.Empty(s.extract(wrongPriv, out))
		})
	}
}

// A span stops at the first frame that should have carried a chunk and did
// not. Only WithPartial hands out what arrived before it, and only once the
// first chunk has authenticated.
func (s *SpanSuite) TestInterruptedSpan() {
	pub, priv, err := GenerateKeyPair()
	s.Require().NoError(err)
	_, other, err := GenerateKeyPair()
	s.Require().NoError(err)
	msg := bytes.Repeat([]byte("abcdefghij"), 33)
	full := s.marshaledText(string(msg))
	room := EnvelopeOverhead + wire.SpanStartHeaderSize + 100
	plan := planSpan(full, []int{room, room, room, room})
	s.Require().NotNil(plan[3])

	sealed := func(i int) stego.Bits {
		env, err := crypto.Seal(plan[i], pub)
		s.Require().NoError(err)
		return env.Marshal()
	}
	noise := stego.Bits(bytes.Repeat([]byte{0x5a}, room))
	prefix := func(chunks int) []byte {
		n := len(plan[0]) - wire.SpanStartHeaderSize
		for i := 1; i < chunks; i++ {
			n += len(plan[i]) - wire.SpanContinueHeaderSize
		}
		return full[wire.PayloadHeaderSize:n]
	}
	type frame struct {
		bits stego.Bits
		room int
	}
	tests := []struct {
		title       string
		key         []byte
		frames      []frame
		wantPartial []byte
		wantFull    bool
	}{
		{"chunk lost mid span", priv, []frame{{sealed(0), room}, {sealed(1), room}, {noise, room}, {sealed(3), room}}, prefix(2), false},
		{"first chunk lost", priv, []frame{{noise, room}, {sealed(1), room}, {sealed(2), room}, {sealed(3), room}}, nil, false},
		{"stream ends mid span", priv, []frame{{sealed(0), room}, {sealed(1), room}}, prefix(2), false},
		{"frame without room is not a gap", priv, []frame{{sealed(0), room}, {noise, 0}, {sealed(1), room}, {sealed(2), room}, {sealed(3), room}}, nil, true},
		{"chunk out of order breaks the span", priv, []frame{{sealed(0), room}, {sealed(0), room}}, prefix(1), false},
		{"wrong key", other, []frame{{sealed(0), room}, {sealed(1), room}, {noise, room}}, nil, false},
	}
	for _, tc := range tests {
		for _, keep := range []bool{false, true} {
			s.Run(fmt.Sprintf("%s keep=%v", tc.title, keep), func() {
				o := &opener{priv: tc.key, keep: keep}
				var got []Result
				for i, f := range tc.frames {
					if r, ok := o.open(int64(i), f.bits, nil, func() int { return f.room }); ok {
						got = append(got, r)
					}
				}
				if r, ok := o.interrupt(); ok {
					got = append(got, r)
				}

				switch {
				case tc.wantFull:
					s.Require().Len(got, 1)
					s.False(got[0].Partial)
					s.Equal(msg, got[0].Payload.Data)
				case keep && tc.wantPartial != nil:
					s.Require().Len(got, 1)
					s.True(got[0].Partial)
					s.Equal(len(msg), got[0].Total)
					s.Equal(PayloadText, got[0].Payload.Type)
					s.Equal(tc.wantPartial, got[0].Payload.Data)
				default:
					s.Empty(got)
				}
			})
		}
	}
}

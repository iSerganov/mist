package mist

import (
	"bytes"
	"context"
	"io"
	"testing"
	"time"

	"github.com/iSerganov/mist/internal/av"
	"github.com/iSerganov/mist/internal/codec"
	"github.com/iSerganov/mist/internal/stego"
	"github.com/stretchr/testify/suite"
)

type LosslessSuite struct {
	audioSuite
}

func TestLosslessSuite(t *testing.T) { suite.Run(t, new(LosslessSuite)) }

func (s *LosslessSuite) requireFormat(name string) {
	if !av.Available() {
		s.T().Skip("libav not available")
	}
	if _, err := LookupFormat(name, ""); err != nil {
		s.T().Skipf("this FFmpeg cannot write %s: %v", name, err)
	}
}

// Every lossless codec embeds in PCM samples, so one carrier has to come
// back out of all of them unchanged. Both carriers matter: the long one
// puts the message in a whole frame, the short one in the trailing
// partial frame, where the encoder's own padding is what would break the
// two sides' agreement about where a sample sits.
func (s *LosslessSuite) TestRoundTripPerFormat() {
	tests := []struct {
		title  string
		format string
		codec  string
	}{
		{"flac", "flac", ""},
		{"wav", "wav", ""},
		{"wavpack", "wv", ""},
		{"tta", "tta", ""},
		{"aiff", "aiff", ""},
		{"caf", "caf", ""},
		{"alac by codec", "caf", "alac"},
	}
	carriers := map[string][]byte{
		"whole frames":  s.pcmCarrier(20 * time.Second),
		"partial frame": s.pcmCarrier(2 * time.Second),
	}
	for _, tc := range tests {
		s.Run(tc.title, func() {
			s.requireFormat(tc.format)
			for shape, carrier := range carriers {
				s.Run(shape, func() {
					pub, priv, err := GenerateKeyPair()
					s.Require().NoError(err)

					out := s.stego(pub, Text("meet me at the pier"), carrier,
						WithFormat(tc.format), WithCodec(tc.codec))
					res := s.extract(priv, out)

					s.Require().NotEmpty(res)
					s.Equal("meet me at the pier", string(res[0].Payload.Data))
					s.Equal(PayloadText, res[0].Payload.Type)
				})
			}
		})
	}
}

// Channel count is part of the position mapping, and a carrier too small
// for the envelope has to fail loudly rather than write an empty file.
func (s *LosslessSuite) TestCarrierShapes() {
	tests := []struct {
		title    string
		channels int
		duration time.Duration
		wantErr  error
	}{
		{"mono", 1, 20 * time.Second, nil},
		{"stereo spanning frames", 2, 25 * time.Second, nil},
		{"far too short", 2, 100 * time.Millisecond, ErrNoCapacity},
	}
	for _, tc := range tests {
		s.Run(tc.title, func() {
			s.requireFormat("flac")
			pub, priv, err := GenerateKeyPair()
			s.Require().NoError(err)
			carrier := wav(testRate, tc.channels, int(tc.duration.Seconds()*testRate))

			em, err := NewEmitter(pub, WithFormat("flac"))
			s.Require().NoError(err)
			out, err := em.EmbedReader(context.Background(), bytes.NewReader(carrier), Text("edge"))
			if tc.wantErr != nil {
				s.Require().ErrorIs(err, tc.wantErr)
				return
			}
			s.Require().NoError(err)
			defer func() { _ = out.Close() }()
			raw, err := io.ReadAll(out)
			s.Require().NoError(err)

			res := s.extract(priv, raw)
			s.Require().NotEmpty(res)
			s.Equal("edge", string(res[0].Payload.Data))
		})
	}
}

// The wrong private key must recover nothing at all — not an error, not
// an empty result, nothing.
func (s *LosslessSuite) TestWrongKeyRecoversNothing() {
	s.requireFormat("flac")
	pub, _, err := GenerateKeyPair()
	s.Require().NoError(err)
	_, other, err := GenerateKeyPair()
	s.Require().NoError(err)

	out := s.stego(pub, Text("secret"), s.pcmCarrier(20*time.Second), WithFormat("flac"))
	s.Empty(s.extract(other, out))
}

// A two-byte message and a two-hundred-byte one must leave the same
// footprint: every frame is written at the same density whatever is in
// it, with filler taking up the slack. Only half the written positions
// actually move — LSB matching leaves a sample alone when its bit is
// already right — so the counts match statistically, not exactly. Without
// the filler the long payload would touch several times as many samples
// as the short one, which is what this pins down.
func (s *LosslessSuite) TestConstantDensity() {
	pcm := s.decodedCarrier(20 * time.Second)
	short := s.perturbed(pcm, Text("hi"))
	long := s.perturbed(pcm, Text(string(bytes.Repeat([]byte("x"), 200))))

	s.Positive(short)
	s.InEpsilon(short, long, 0.05, "payload length must not show in the footprint")
}

// A lossless embed is close to transparent: only the LSB of a fraction of
// the samples moves, and only by one step.
func (s *LosslessSuite) TestOnlyLeastSignificantBitsMove() {
	pcm := s.decodedCarrier(10 * time.Second)
	before := append([]float32(nil), pcm.Planes[0]...)

	em, err := NewEmitter(mustPub(s), WithFormat("flac"))
	s.Require().NoError(err)
	s.Require().NoError(em.embedSamples(context.Background(), pcm, 1<<15, []byte("payload")))

	for i, was := range before {
		delta := int(was*(1<<15)) - int(pcm.Planes[0][i]*(1<<15))
		s.LessOrEqual(abs(delta), 1, "sample %d moved by %d", i, delta)
	}
}

// Embed and Listen must cut frames at the same sample offsets; a window
// out by one scrambles every position in it.
func (s *LosslessSuite) TestWindowerMatchesEmbedFrames() {
	pcm := s.decodedCarrier(25 * time.Second)
	want := sampleFrames(pcm, 1<<15)

	w := newWindower(frameParams(pcm), 1<<15)
	got := w.push(pcm)
	got = append(got, w.flush()...)

	s.Require().Len(got, len(want))
	for i := range want {
		s.Equal(want[i].index, got[i].index, "frame %d index", i)
		s.Equal(want[i].N, got[i].N, "frame %d length", i)
	}
}

// A listener that joins mid-window cannot align to it, so that window is
// reported as partial and skipped rather than mis-read. The chunk size
// matters: a decoder hands over far less than a frame at a time, so the
// discarded tail spans many pushes and the count has to survive between
// them.
func (s *LosslessSuite) TestWindowerDropsPartialJoin() {
	tests := []struct {
		title string
		chunk int
	}{
		{"one chunk holds everything", 0},
		{"decoder-sized chunks", 4096},
	}
	for _, tc := range tests {
		s.Run(tc.title, func() {
			pcm := s.decodedCarrier(20 * time.Second)
			fp := frameParams(pcm)
			pcm.PTS = int64(fp.Samples() / 3)

			w := newWindower(fp, 1<<15)
			got := w.push(pcm)
			if tc.chunk > 0 {
				w = newWindower(fp, 1<<15)
				got = nil
				for _, part := range chunked(pcm, tc.chunk) {
					got = append(got, w.push(part)...)
				}
			}
			got = append(got, w.flush()...)

			s.Require().NotEmpty(got)
			s.True(got[0].partial, "the half-seen frame must be reported, not read")
			s.EqualValues(0, got[0].index)
			s.Require().Greater(len(got), 1)
			s.EqualValues(1, got[1].index, "reading resumes at the next whole frame")
			s.False(got[1].partial)
			s.Equal(fp.Samples(), got[1].N, "the first whole frame must be whole")

			// Alignment, not just bookkeeping: frame 1 has to begin at the
			// frame boundary, which in the carrier sits this far in.
			at := fp.Samples() - int(pcm.PTS)
			s.Equal(pcm.Planes[0][at], got[1].Planes[0][got[1].Off],
				"frame 1 starts at the wrong sample")
		})
	}
}

// chunked splits pcm the way a decoder hands it over, carrying the
// running timestamp so only the first chunk sets the phase.
func chunked(pcm codec.PCM, n int) []codec.PCM {
	var out []codec.PCM
	for off := 0; off < pcm.NbSamples; off += n {
		part := codec.PCM{
			NbSamples: min(n, pcm.NbSamples-off), Channels: pcm.Channels,
			SampleRate: pcm.SampleRate, Format: pcm.Format, PTS: pcm.PTS + int64(off),
		}
		for _, p := range pcm.Planes {
			part.Planes = append(part.Planes, p[off:min(off+n, len(p))])
		}
		out = append(out, part)
	}
	return out
}

// decodedCarrier is the carrier as Embed sees it: float planes ready to
// be written into.
func (s *LosslessSuite) decodedCarrier(d time.Duration) codec.PCM {
	s.T().Helper()
	if !av.Available() {
		s.T().Skip("libav not available")
	}
	pcm, _, err := decodeCarrier(bytes.NewReader(s.pcmCarrier(d)))
	s.Require().NoError(err)
	return pcm
}

// perturbed embeds into a copy of pcm and counts the samples that moved.
func (s *LosslessSuite) perturbed(pcm codec.PCM, payload Payload) int {
	s.T().Helper()
	before := make([][]float32, len(pcm.Planes))
	copyOf := codec.PCM{
		Planes: make([][]float32, len(pcm.Planes)), NbSamples: pcm.NbSamples,
		Channels: pcm.Channels, SampleRate: pcm.SampleRate, Format: pcm.Format,
	}
	for i, p := range pcm.Planes {
		before[i] = append([]float32(nil), p...)
		copyOf.Planes[i] = append([]float32(nil), p...)
	}
	plain, err := marshalPlain(payload, nil)
	s.Require().NoError(err)
	em, err := NewEmitter(mustPub(s), WithFormat("flac"))
	s.Require().NoError(err)
	s.Require().NoError(em.embedSamples(context.Background(), copyOf, 1<<15, plain))

	n := 0
	for i, p := range copyOf.Planes {
		for j, v := range p {
			if v != before[i][j] {
				n++
			}
		}
	}
	return n
}

func mustPub(s *LosslessSuite) []byte {
	s.T().Helper()
	pub, _, err := GenerateKeyPair()
	s.Require().NoError(err)
	return pub
}

func abs(n int) int {
	if n < 0 {
		return -n
	}
	return n
}

// Samples is the one place the two embedding domains differ, so its grid
// arithmetic is worth pinning down on its own.
func (s *LosslessSuite) TestSampleGridRoundTrip() {
	tests := []struct {
		title string
		value int32
		bit   uint8
	}{
		{"mid-scale keeps its bit", 1000, 0},
		{"mid-scale flips", 1001, 0},
		{"top of range cannot clip upward", 32767, 0},
		{"bottom of range cannot clip downward", -32768, 1},
		{"zero", 0, 1},
	}
	for _, tc := range tests {
		s.Run(tc.title, func() {
			planes := [][]float32{{float32(tc.value) / (1 << 15)}}
			sm := stego.Samples{Planes: planes, N: 1, Scale: 1 << 15}
			s.Require().Equal(tc.value, sm.At(0))

			sm.Set(0, stego.Match(sm.At(0), tc.bit))
			got := sm.At(0)
			s.EqualValues(tc.bit, got&1, "bit not embedded")
			s.LessOrEqual(abs(int(got-tc.value)), 2, "moved too far")
			s.GreaterOrEqual(got, int32(-32768))
			s.LessOrEqual(got, int32(32767))
		})
	}
}

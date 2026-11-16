package mist

import (
	"bytes"
	"context"
	"encoding/binary"
	"net/http"
	"net/http/httptest"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/suite"
)

type LiveSuite struct {
	audioSuite
}

func TestLiveSuite(t *testing.T) { suite.Run(t, new(LiveSuite)) }

func (s *LiveSuite) SetupTest() { s.requireLibav() }

func (s *LiveSuite) requireTarget(format, codec string) {
	if _, err := LookupFormat(format, codec); err != nil {
		s.T().Skipf("this FFmpeg cannot write %s %s: %v", format, codec, err)
	}
}

// Re-sealing puts the message in every frame that holds it, each under
// its own ephemeral key, so each is read without any other.
func (s *LiveSuite) TestResealCarriesThePayloadInEveryFrame() {
	tests := []struct {
		title  string
		format string
		codec  string
	}{
		{"flac", "flac", ""},
		{"ogg vorbis", "ogg", ""},
	}
	for _, tc := range tests {
		s.Run(tc.title, func() {
			s.requireTarget(tc.format, tc.codec)
			pub, priv, err := GenerateKeyPair()
			s.Require().NoError(err)

			out := s.stego(pub, Text("on air"), s.pcmCarrier(4*FrameDuration),
				WithFormat(tc.format), WithCodec(tc.codec), WithReseal())
			res := s.extract(priv, out)

			s.Require().Len(res, 4)
			for i, r := range res {
				s.Equal("on air", string(r.Payload.Data))
				s.Equal(int64(i), r.FrameIdx)
			}
			_, other, err := GenerateKeyPair()
			s.Require().NoError(err)
			s.Empty(s.extract(other, out), "a wrong key must read nothing from any frame")
		})
	}
}

// A re-sealed payload is never spanned: one that needs two frames is
// refused even though the carrier could hold it once.
func (s *LiveSuite) TestResealRefusesAPayloadThatNeedsASpan() {
	s.requireTarget("flac", "")
	pub, priv, err := GenerateKeyPair()
	s.Require().NoError(err)
	carrier := s.pcmCarrier(4 * FrameDuration)
	msg := Text(string(bytes.Repeat([]byte("0123456789"), 200)))

	spanned := s.extract(priv, s.stego(pub, msg, carrier, WithFormat("flac")))
	s.Require().Len(spanned, 1, "the carrier holds the payload when it may span")

	em, err := NewEmitter(pub, WithFormat("flac"), WithReseal())
	s.Require().NoError(err)
	_, err = em.EmbedReader(context.Background(), bytes.NewReader(carrier), msg)
	s.ErrorIs(err, ErrNoCapacity)
}

// A listener who tunes in after the stream began has none of its first
// frames. Ogg keeps absolute time in every page, so the Vorbis frames that
// follow still line up and the next whole one gives the message.
func (s *LiveSuite) TestLateJoinerReadsTheNextFrame() {
	pub, priv, err := GenerateKeyPair()
	s.Require().NoError(err)
	out := s.stego(pub, Text("on air"), s.pcmCarrier(5*FrameDuration), WithReseal())

	// Joins 12 s in: halfway through frame 1.
	late := dropOggAudioBefore(out, 12*testRate)
	s.Require().Less(len(late), len(out))
	res := s.extract(priv, late)

	s.Require().NotEmpty(res)
	s.Equal(int64(2), res[0].FrameIdx, "frame 1 was joined midway and cannot be read")
	for _, r := range res {
		s.Equal("on air", string(r.Payload.Data))
	}
}

// A live carrier has no end, so nothing can wait for one: output has to
// appear while the source is still sending, and cancelling the context is
// the normal way to stop, leaving a finished stream behind.
func (s *LiveSuite) TestLiveCarrierIsWrittenAsItArrives() {
	tests := []struct {
		title  string
		opts   []EmitterOption
		atMost int
	}{
		{"embedded once", nil, 1},
		{"re-sealed every frame", []EmitterOption{WithReseal()}, 3},
	}
	for _, tc := range tests {
		s.Run(tc.title, func() {
			s.requireTarget("flac", "")
			pub, priv, err := GenerateKeyPair()
			s.Require().NoError(err)
			carrier := s.carrier(4 * FrameDuration)
			release := make(chan struct{})
			var requests atomic.Int32
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				requests.Add(1)
				// No length and no ranges: all libav can do is read on.
				_, _ = w.Write(carrier[:len(carrier)*5/8])
				w.(http.Flusher).Flush()
				<-release
			}))
			defer srv.Close()
			defer close(release)

			var frames atomic.Int32
			opts := append(tc.opts, WithFormat("flac"), WithEmitProgress(func(Progress) { frames.Add(1) }))
			em, err := NewEmitter(pub, opts...)
			s.Require().NoError(err)
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			var out lockedBuffer
			done := make(chan error, 1)
			go func() { done <- em.EmbedTo(ctx, srv.URL+"/live.ogg", plainWriter{&out}, Text("on air")) }()

			// The server has sent 20 s and is holding the rest: two frames.
			s.Require().Eventually(func() bool { return frames.Load() >= 2 && out.Len() > 0 },
				60*time.Second, 20*time.Millisecond, "nothing was written while the source was still open")
			cancel()
			s.Require().NoError(<-done)

			s.Equal(int32(1), requests.Load())
			res := s.extract(priv, out.Bytes())
			s.Require().NotEmpty(res)
			s.LessOrEqual(len(res), tc.atMost)
			if tc.atMost > 1 {
				s.Greater(len(res), 1)
			}
			for _, r := range res {
				s.Equal("on air", string(r.Payload.Data))
			}
		})
	}
}

type lockedBuffer struct {
	mu  sync.Mutex
	buf bytes.Buffer
}

func (b *lockedBuffer) Write(p []byte) (int, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buf.Write(p)
}

func (b *lockedBuffer) Len() int {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buf.Len()
}

func (b *lockedBuffer) Bytes() []byte {
	b.mu.Lock()
	defer b.mu.Unlock()
	return append([]byte(nil), b.buf.Bytes()...)
}

// dropOggAudioBefore removes the audio pages of an Ogg stream that end
// before the given granule position, keeping the header pages: what a
// listener who connects late is sent.
func dropOggAudioBefore(b []byte, granule int64) []byte {
	var out []byte
	joined := false
	for len(b) >= 27 && string(b[:4]) == "OggS" {
		segs := int(b[26])
		size := 27 + segs
		for _, l := range b[27 : 27+segs] {
			size += int(l)
		}
		pos := int64(binary.LittleEndian.Uint64(b[6:14]))
		if pos >= granule {
			joined = true
		}
		if pos == 0 || joined {
			out = append(out, b[:size]...)
		}
		b = b[size:]
	}
	return out
}

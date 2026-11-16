package mist

import (
	"bytes"
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/iSerganov/mist/internal/codec/vorbis"
	"github.com/iSerganov/mist/internal/frame"
	"github.com/iSerganov/mist/internal/stego"
	"github.com/stretchr/testify/suite"
)

type EmbedToSuite struct {
	audioSuite
}

func TestEmbedToSuite(t *testing.T) { suite.Run(t, new(EmbedToSuite)) }

func (s *EmbedToSuite) SetupTest() { s.requireLibav() }

// plainWriter hides every method of a buffer but Write, so the muxer
// cannot seek back into what it has written.
type plainWriter struct{ w io.Writer }

func (p plainWriter) Write(b []byte) (int, error) { return p.w.Write(b) }

// A stream has no end to seek back from. The audio and the message must
// come out the same whether or not the destination can seek.
func (s *EmbedToSuite) TestRoundTripWhetherOrNotTheDestinationSeeks() {
	tests := []struct {
		title   string
		format  string
		carrier []byte
	}{
		{"flac", "flac", s.pcmCarrier(18 * time.Second)},
		{"wav", "wav", s.pcmCarrier(18 * time.Second)},
		{"flac with a partial last frame", "flac", s.pcmCarrier(2 * time.Second)},
		{"ogg vorbis", "ogg", s.pcmCarrier(18 * time.Second)},
	}
	for _, tc := range tests {
		s.Run(tc.title, func() {
			if _, err := LookupFormat(tc.format, ""); err != nil {
				s.T().Skipf("this FFmpeg cannot write %s: %v", tc.format, err)
			}
			pub, priv, err := GenerateKeyPair()
			s.Require().NoError(err)
			em, err := NewEmitter(pub, WithFormat(tc.format))
			s.Require().NoError(err)
			path := s.writeTemp(tc.carrier)
			want, _, _, err := decodeCarrier(bytes.NewReader(tc.carrier))
			s.Require().NoError(err)

			var seekable seekBuf
			var plain bytes.Buffer
			s.Require().NoError(em.EmbedTo(context.Background(), path, &seekable, Text("on air")))
			s.Require().NoError(em.EmbedTo(context.Background(), path, plainWriter{&plain}, Text("on air")))

			for name, out := range map[string][]byte{"seekable": seekable.data, "plain": plain.Bytes()} {
				got, _, _, err := decodeCarrier(bytes.NewReader(out))
				s.Require().NoError(err, name)
				s.Equal(want.NbSamples, got.NbSamples, name)
				res := s.extract(priv, out)
				s.Require().Len(res, 1, name)
				s.Equal("on air", string(res[0].Payload.Data), name)
			}
		})
	}
}

// Embed decides where the payload goes from the rooms it sees, so its frames
// must be the ones a reader of the finished file finds. For Vorbis that
// means the loopback, read without seeking, rebuilds the timestamps a file
// gives.
func (s *EmbedToSuite) TestMeasuredRoomsAreWhatAReaderSees() {
	// Bursts between silences make libvorbis switch block sizes.
	samples := make([]int16, 5*8*testRate*2)
	for i := range samples {
		if (i/(testRate/2))%3 != 0 {
			samples[i] = int16(i*7919%16000 - 8000)
		}
	}
	carrier := wavFile(testRate, 2, samples)
	pub, _, err := GenerateKeyPair()
	s.Require().NoError(err)

	s.Run("ogg vorbis", func() {
		em, err := NewEmitter(pub)
		s.Require().NoError(err)
		measured := s.measure(em, carrier)

		info, pkts, err := readPackets(bytes.NewReader(s.stego(pub, Text("x"), carrier)))
		s.Require().NoError(err)
		vc := vorbis.New()
		s.Require().NoError(vc.Load(info.Extradata))
		var seen []int
		for _, g := range frame.GroupPackets(pkts, frame.Params{SampleRate: testRate, Channels: 2, Duration: FrameDuration}) {
			room, err := stego.Capacity(vc, g.Packets)
			if !errors.Is(err, stego.ErrNoResidues) {
				s.Require().NoError(err)
			}
			seen = append(seen, room)
		}
		s.Equal(seen, measured)
	})

	s.Run("flac", func() {
		em, err := NewEmitter(pub, WithFormat("flac"))
		s.Require().NoError(err)
		measured := s.measure(em, carrier)

		pcm, _, _, err := decodeCarrier(bytes.NewReader(s.stego(pub, Text("x"), carrier, WithFormat("flac"))))
		s.Require().NoError(err)
		var seen []int
		for _, w := range sampleFrames(pcm, 1<<15) {
			seen = append(seen, w.Capacity())
		}
		s.Equal(seen, measured)
	})
}

func (s *EmbedToSuite) TestURLCarrierRoundTrips() {
	tests := []struct {
		title  string
		format string
	}{
		{"flac", "flac"},
		{"ogg vorbis", "ogg"},
	}
	carrier := s.pcmCarrier(18 * time.Second)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.ServeContent(w, r, "carrier.wav", time.Time{}, bytes.NewReader(carrier))
	}))
	defer srv.Close()
	for _, tc := range tests {
		s.Run(tc.title, func() {
			pub, priv, err := GenerateKeyPair()
			s.Require().NoError(err)
			em, err := NewEmitter(pub, WithFormat(tc.format))
			s.Require().NoError(err)

			var out seekBuf
			s.Require().NoError(em.EmbedTo(context.Background(), srv.URL+"/carrier.wav", &out, Text("on air")))

			res := s.extract(priv, out.data)
			s.Require().Len(res, 1)
			s.Equal("on air", string(res[0].Payload.Data))
		})
	}
}

// The carrier is read once, so a payload that fits nowhere is only known
// at its end. It is still loud, and Embed hands back no stream at all.
func (s *EmbedToSuite) TestNoCapacityIsReportedAfterTheCarrierEnds() {
	pub, _, err := GenerateKeyPair()
	s.Require().NoError(err)
	em, err := NewEmitter(pub, WithFormat("flac"))
	s.Require().NoError(err)
	path := s.writeTemp(s.pcmCarrier(100 * time.Millisecond))

	s.ErrorIs(em.EmbedTo(context.Background(), path, &bytes.Buffer{}, Text("too big")), ErrNoCapacity)
	rc, err := em.Embed(context.Background(), path, Text("too big"))
	s.ErrorIs(err, ErrNoCapacity)
	s.Nil(rc)
}

// A message that does not fit the first frame with room is spanned from
// there rather than held back for a later, larger frame: deciding that
// would need the rest of the carrier first.
func (s *EmbedToSuite) TestPayloadIsSpannedFromTheFirstFrameWithRoom() {
	pub, priv, err := GenerateKeyPair()
	s.Require().NoError(err)
	em, err := NewEmitter(pub, WithFormat("flac"))
	s.Require().NoError(err)
	// Four seconds of silence leave the first frame half the room of the rest.
	samples := make([]int16, 4*8*testRate*2)
	for i := 4 * testRate * 2; i < len(samples); i++ {
		samples[i] = int16(i*7919%16000 - 8000)
	}
	carrier := wavFile(testRate, 2, samples)
	rooms := s.measure(em, carrier)
	s.Require().Len(rooms, 4)
	msg := string(bytes.Repeat([]byte("x"), rooms[0]-envelopeFloor+1))
	s.Require().GreaterOrEqual(rooms[1]-envelopeFloor, len(msg), "a later frame must hold the message whole")

	res := s.extract(priv, s.stego(pub, Text(msg), carrier, WithFormat("flac")))

	s.Require().Len(res, 1)
	s.Equal(msg, string(res[0].Payload.Data))
	s.Equal(int64(0), res[0].FrameIdx, "the span starts in the first frame")
}

// Embed returns a finished stream, which a source with no end never gives.
func (s *EmbedToSuite) TestEmbedRefusesALiveSource() {
	carrier := s.carrier(2 * FrameDuration)
	release := make(chan struct{})
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write(carrier)
		w.(http.Flusher).Flush()
		<-release
	}))
	defer srv.Close()
	defer close(release)
	pub, _, err := GenerateKeyPair()
	s.Require().NoError(err)
	em, err := NewEmitter(pub)
	s.Require().NoError(err)

	rc, err := em.Embed(context.Background(), srv.URL+"/live.ogg", Text("x"))

	s.ErrorIs(err, ErrInvalidSource)
	s.Nil(rc)
}

// A server that accepts the connection and sends nothing must not hold
// EmbedTo past its context: libav's network IO is told to give up.
func (s *EmbedToSuite) TestCancelReachesAStalledSource() {
	release := make(chan struct{})
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.(http.Flusher).Flush()
		<-release
	}))
	defer srv.Close()
	defer close(release)
	pub, _, err := GenerateKeyPair()
	s.Require().NoError(err)
	em, err := NewEmitter(pub)
	s.Require().NoError(err)
	ctx, cancel := context.WithTimeout(context.Background(), 300*time.Millisecond)
	defer cancel()

	done := make(chan error, 1)
	go func() { done <- em.EmbedTo(ctx, srv.URL+"/stalled.ogg", &bytes.Buffer{}, Text("x")) }()

	select {
	case err := <-done:
		s.Error(err)
	case <-time.After(10 * time.Second):
		s.Fail("EmbedTo was still blocked 10 s after its context ended")
	}
}

// measure is every stego frame's room in carrier, as Embed sees it.
func (s *EmbedToSuite) measure(em *Emitter, carrier []byte) []int {
	s.T().Helper()
	in, err := openReader(bytes.NewReader(carrier))
	s.Require().NoError(err)
	rooms, err := em.measure(context.Background(), in)
	s.Require().NoError(err)
	return rooms
}

func (s *EmbedToSuite) TestRejectsBadArguments() {
	pub, _, err := GenerateKeyPair()
	s.Require().NoError(err)
	em, err := NewEmitter(pub, WithFormat("flac"))
	s.Require().NoError(err)
	path := s.writeTemp(s.pcmCarrier(time.Second))
	cancelled, cancel := context.WithCancel(context.Background())
	cancel()

	tests := []struct {
		title string
		ctx   context.Context
		src   string
		dst   io.Writer
		want  error
	}{
		{"nil destination", context.Background(), path, nil, ErrInvalidSource},
		{"missing carrier", context.Background(), path + ".absent", &bytes.Buffer{}, ErrCarrier},
		{"cancelled context", cancelled, path, &bytes.Buffer{}, context.Canceled},
	}
	for _, tc := range tests {
		s.Run(tc.title, func() {
			s.ErrorIs(em.EmbedTo(tc.ctx, tc.src, tc.dst, Text("x")), tc.want)
		})
	}
}

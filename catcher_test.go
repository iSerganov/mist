package mist

import (
	"bytes"
	"context"
	"log/slog"
	"testing"
	"time"

	"github.com/iSerganov/mist/internal/av"
	"github.com/stretchr/testify/suite"
)

type CatcherSuite struct {
	audioSuite
}

func TestCatcherSuite(t *testing.T) {
	suite.Run(t, &CatcherSuite{})
}

// collect drains ch, guarding against a Listen that never closes.
func (s *CatcherSuite) collect(ch <-chan Result) []Result {
	s.T().Helper()
	var out []Result
	done := make(chan struct{})
	go func() {
		defer close(done)
		for r := range ch {
			out = append(out, r)
		}
	}()
	select {
	case <-done:
	case <-time.After(30 * time.Second):
		s.FailNow("Listen did not close its channel")
	}
	return out
}

func (s *CatcherSuite) TestNewCatcherRejectsNilKey() {
	c, err := NewCatcher(nil)
	s.Nil(c)
	s.ErrorIs(err, ErrInvalidKey)
}

func (s *CatcherSuite) TestNewCatcherCopiesPrivateKey() {
	key := []byte{1, 2, 3}
	c, err := NewCatcher(key)
	s.Require().NoError(err)
	key[0] = 9
	s.Equal(byte(1), c.priv[0])
	s.Len(c.priv, 3)
}

func (s *CatcherSuite) TestRejectsBadArguments() {
	_, priv, err := GenerateKeyPair()
	s.Require().NoError(err)
	short, err := NewCatcher([]byte{1, 2, 3})
	s.Require().NoError(err)
	ok, err := NewCatcher(priv)
	s.Require().NoError(err)

	tests := []struct {
		title   string
		call    func() error
		wantErr error
	}{
		{"short key", func() error { _, err := short.Listen(context.Background(), "x.ogg"); return err }, ErrInvalidKey},
		{"empty source", func() error { _, err := ok.Listen(context.Background(), ""); return err }, ErrInvalidSource},
		{"nil reader", func() error { _, err := ok.ListenReader(context.Background(), nil); return err }, ErrInvalidSource},
		{"nil file", func() error { _, err := ok.Extract(context.Background(), nil); return err }, ErrInvalidSource},
		{"missing file", func() error { _, err := ok.Listen(context.Background(), "does-not-exist.ogg"); return err }, ErrCarrier},
		{"not audio", func() error {
			_, err := ok.ListenReader(context.Background(), bytes.NewReader([]byte("nope")))
			return err
		}, ErrCarrier},
	}
	for _, tc := range tests {
		s.Run(tc.title, func() {
			s.ErrorIs(tc.call(), tc.wantErr)
		})
	}
}

// newScanner is the only place that knows extraction is Vorbis-specific,
// so an unsupported carrier is rejected there rather than failing vaguely
// further down.
func (s *CatcherSuite) TestRejectsNonVorbisCarrier() {
	_, priv, err := GenerateKeyPair()
	s.Require().NoError(err)

	tests := []struct {
		title string
		id    int
	}{
		{"unmapped codec", av.CodecIDNone},
		{"mp3", av.CodecIDMP3},
		{"aac", av.CodecIDAAC},
	}
	for _, tc := range tests {
		s.Run(tc.title, func() {
			_, err := newScanner(priv, av.AudioInfo{CodecID: tc.id}, slog.New(slog.DiscardHandler))
			s.ErrorIs(err, ErrUnsupportedCodec)
		})
	}
}

func (s *CatcherSuite) TestListen() {
	s.requireLibav()
	pub, priv, err := GenerateKeyPair()
	s.Require().NoError(err)
	path := s.writeTemp(s.stego(pub, Text("over a path"), s.carrier(FrameDuration)))

	c, err := NewCatcher(priv)
	s.Require().NoError(err)
	ch, err := c.Listen(context.Background(), path)
	s.Require().NoError(err)

	got := s.collect(ch)
	s.Require().NotEmpty(got)
	s.Equal("over a path", string(got[0].Payload.Data))
	s.Equal(PayloadText, got[0].Payload.Type)
}

func (s *CatcherSuite) TestListenReader() {
	s.requireLibav()
	pub, priv, err := GenerateKeyPair()
	s.Require().NoError(err)
	raw := s.stego(pub, Text("over a reader"), s.carrier(FrameDuration))

	c, err := NewCatcher(priv)
	s.Require().NoError(err)
	ch, err := c.ListenReader(context.Background(), bytes.NewReader(raw))
	s.Require().NoError(err)

	got := s.collect(ch)
	s.Require().NotEmpty(got)
	s.Equal("over a reader", string(got[0].Payload.Data))
}

func (s *CatcherSuite) TestExtract() {
	s.requireLibav()
	pub, priv, err := GenerateKeyPair()
	s.Require().NoError(err)

	tests := []struct {
		title string
		dur   time.Duration
		text  string
	}{
		{"single frame", FrameDuration, "one frame"},
		{"payload repeats across frames", 2 * FrameDuration, "looped every frame"},
	}
	for _, tc := range tests {
		s.Run(tc.title, func() {
			f := s.openTemp(s.stego(pub, Text(tc.text), s.carrier(tc.dur)))
			c, err := NewCatcher(priv)
			s.Require().NoError(err)

			got, err := c.Extract(context.Background(), f)
			s.Require().NoError(err)
			s.Require().NotEmpty(got)
			for _, r := range got {
				s.Equal(tc.text, string(r.Payload.Data))
			}
		})
	}
}

// Every whole frame carries the payload afresh, so a listener that joins
// late still recovers it.
func (s *CatcherSuite) TestPayloadRepeatsInEveryFrame() {
	s.requireLibav()
	pub, priv, err := GenerateKeyPair()
	s.Require().NoError(err)
	f := s.openTemp(s.stego(pub, Text("repeated"), s.carrier(3*FrameDuration)))

	c, err := NewCatcher(priv)
	s.Require().NoError(err)
	got, err := c.Extract(context.Background(), f)
	s.Require().NoError(err)

	s.Require().GreaterOrEqual(len(got), 3, "one result per whole frame")
	for i, r := range got {
		s.Equal("repeated", string(r.Payload.Data))
		s.Equal(int64(i), r.FrameIdx)
	}
}

// A wrong key is indistinguishable from an unmarked carrier: no results,
// no error, nothing that says "something was here".
func (s *CatcherSuite) TestSilentOnUndecryptableInput() {
	s.requireLibav()
	pub, _, err := GenerateKeyPair()
	s.Require().NoError(err)
	_, otherPriv, err := GenerateKeyPair()
	s.Require().NoError(err)

	tests := []struct {
		title string
		raw   []byte
	}{
		{"wrong key", s.stego(pub, Text("not for you"), s.carrier(FrameDuration))},
		{"no payload embedded", s.carrier(FrameDuration)},
	}
	for _, tc := range tests {
		s.Run(tc.title, func() {
			c, err := NewCatcher(otherPriv)
			s.Require().NoError(err)
			got, err := c.Extract(context.Background(), s.openTemp(tc.raw))
			s.NoError(err)
			s.Empty(got)
		})
	}
}

func (s *CatcherSuite) TestListenCancel() {
	_, priv, err := GenerateKeyPair()
	s.Require().NoError(err)
	c, err := NewCatcher(priv)
	s.Require().NoError(err)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	ch, err := c.Listen(ctx, "https://radio.example.com/live.ogg")
	s.Require().NoError(err, "a dead context is not a setup failure")
	s.Empty(s.collect(ch))
	s.Error(ctx.Err())
}

func (s *CatcherSuite) TestListenStopsWhenContextEnds() {
	s.requireLibav()
	pub, priv, err := GenerateKeyPair()
	s.Require().NoError(err)
	raw := s.stego(pub, Text("cancel me"), s.carrier(2*FrameDuration))

	ctx, cancel := context.WithCancel(context.Background())
	c, err := NewCatcher(priv)
	s.Require().NoError(err)
	ch, err := c.ListenReader(ctx, bytes.NewReader(raw))
	s.Require().NoError(err)

	s.Require().NotEmpty((<-ch).Payload.Data, "first frame arrives before cancelling")
	cancel()
	s.collect(ch)
	s.Error(ctx.Err())
}

// A demuxer read already in flight cannot be interrupted, so relay is what
// keeps Listen's promise: cancel must close the channel even while the scan
// behind it is parked, or a caller ranging over a quiet live stream hangs
// until the server speaks again.
func (s *CatcherSuite) TestRelayClosesOnCancel() {
	tests := []struct {
		title  string
		source func() chan Result
	}{
		{"scan parked, nothing to send", func() chan Result { return make(chan Result) }},
		{"scan has a pending result", func() chan Result {
			ch := make(chan Result, 1)
			ch <- Result{Payload: Text("pending")}
			return ch
		}},
	}
	for _, tc := range tests {
		s.Run(tc.title, func() {
			ctx, cancel := context.WithCancel(context.Background())
			out := relay(ctx, tc.source())
			cancel()

			drained := make(chan struct{})
			go func() {
				defer close(drained)
				for range out {
				}
			}()
			select {
			case <-drained:
			case <-time.After(5 * time.Second):
				s.FailNow("relay kept the channel open after cancel")
			}
		})
	}
}

func (s *CatcherSuite) TestRelayForwardsUntilSourceCloses() {
	in := make(chan Result, 2)
	in <- Result{Payload: Text("one"), FrameIdx: 0}
	in <- Result{Payload: Text("two"), FrameIdx: 1}
	close(in)

	var got []string
	for r := range relay(context.Background(), in) {
		got = append(got, string(r.Payload.Data))
	}
	s.Equal([]string{"one", "two"}, got)
}

// A listener that joins mid-frame cannot align to the emitter's window, so
// it says so once and skips that frame rather than failing silently.
func (s *CatcherSuite) TestJoinMidStreamLogsAndSkips() {
	s.requireLibav()
	pub, priv, err := GenerateKeyPair()
	s.Require().NoError(err)
	raw := s.stego(pub, Text("mid stream"), s.carrier(FrameDuration))

	d, err := av.OpenDemuxerReader(bytes.NewReader(raw))
	s.Require().NoError(err)
	defer func() { _ = d.Close() }()

	var logs bytes.Buffer
	sc, err := newScanner(priv, d.Info(), slog.New(slog.NewTextHandler(&logs, nil)))
	s.Require().NoError(err)

	var got []Result
	for i := 0; ; i++ {
		pkt, err := d.NextPacket()
		if err != nil {
			break
		}
		if i < 20 {
			continue // drop the leading packets: we joined late
		}
		if r, ok := sc.push(av.ToCodecPacket(pkt)); ok {
			got = append(got, r)
		}
	}
	if r, ok := sc.flush(); ok {
		got = append(got, r)
	}

	s.Empty(got)
	s.Contains(logs.String(), "joined in the middle of transmission")
}

func (s *CatcherSuite) TestOptions() {
	c, err := NewCatcher(make([]byte, PrivateKeySize),
		WithMaxRetries(7),
		WithBackoff(3*time.Second),
		WithLogger(slog.New(slog.DiscardHandler)),
	)
	s.Require().NoError(err)
	s.Equal(7, c.maxRetries)
	s.Equal(3*time.Second, c.backoff)
	s.NotNil(c.log)
}

func (s *CatcherSuite) TestRetrier() {
	tests := []struct {
		title    string
		left     int
		backoff  time.Duration
		cancel   bool
		wantWait bool
	}{
		{"budget available", 2, 0, false, true},
		{"budget spent", 0, 0, false, false},
		{"cancelled while waiting", 2, time.Hour, true, false},
	}
	for _, tc := range tests {
		s.Run(tc.title, func() {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			if tc.cancel {
				cancel()
			}
			r := retrier{left: tc.left, backoff: tc.backoff}
			s.Equal(tc.wantWait, r.wait(ctx))
		})
	}
}

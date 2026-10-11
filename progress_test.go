package mist

import (
	"bytes"
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/suite"
)

type ProgressSuite struct {
	audioSuite
}

func TestProgressSuite(t *testing.T) { suite.Run(t, new(ProgressSuite)) }

func (s *ProgressSuite) requireFormat(name string) {
	s.requireLibav()
	if _, err := LookupFormat(name, ""); err != nil {
		s.T().Skipf("this FFmpeg cannot write %s: %v", name, err)
	}
}

func (s *ProgressSuite) record() (*[]Progress, ProgressFunc) {
	var got []Progress
	return &got, func(p Progress) { got = append(got, p) }
}

func (s *ProgressSuite) TestEmbedReportsEveryFrameInOrder() {
	tests := []struct {
		title  string
		format string
	}{
		{"vorbis", "ogg"},
		{"flac", "flac"},
	}
	for _, tc := range tests {
		s.Run(tc.title, func() {
			s.requireFormat(tc.format)
			pub, _, err := GenerateKeyPair()
			s.Require().NoError(err)
			got, f := s.record()

			s.stego(pub, Text("progress"), s.pcmCarrier(17*time.Second), WithFormat(tc.format), WithEmitProgress(f))

			s.Require().NotEmpty(*got)
			for i, p := range *got {
				s.Equal(ProgressEmbed, p.Stage)
				s.Equal(i+1, p.Done)
				s.Equal(len(*got), p.Total)
			}
		})
	}
}

func (s *ProgressSuite) TestEmbedReportsNothingBeforeNoCapacity() {
	s.requireFormat("flac")
	pub, _, err := GenerateKeyPair()
	s.Require().NoError(err)
	got, f := s.record()
	em, err := NewEmitter(pub, WithFormat("flac"), WithEmitProgress(f))
	s.Require().NoError(err)

	_, err = em.EmbedReader(context.Background(), bytes.NewReader(s.pcmCarrier(100*time.Millisecond)), Text("too big"))

	s.ErrorIs(err, ErrNoCapacity)
	s.Empty(*got)
}

func (s *ProgressSuite) TestCatchAssemblesASpanToItsLength() {
	s.requireFormat("flac")
	pub, priv, err := GenerateKeyPair()
	s.Require().NoError(err)
	msg := bytes.Repeat([]byte("0123456789"), 200)
	out := s.stego(pub, Text(string(msg)), s.pcmCarrier(4*FrameDuration), WithFormat("flac"))
	got, f := s.record()

	res := s.extractWith(out, priv, WithCatchProgress(f))

	s.Require().Len(res, 1)
	var scan, assemble []Progress
	for _, p := range *got {
		if p.Stage == ProgressAssemble {
			assemble = append(assemble, p)
		} else {
			scan = append(scan, p)
		}
	}
	s.Require().Greater(len(assemble), 1)
	last := assemble[len(assemble)-1]
	s.Equal(last.Total, last.Done)
	s.Equal(4, scan[len(scan)-1].Done)
	s.Equal(4, scan[len(scan)-1].Total)
}

func (s *ProgressSuite) TestWrongKeyReportsTheSameScanAsTheRightKey() {
	s.requireFormat("flac")
	pub, priv, err := GenerateKeyPair()
	s.Require().NoError(err)
	_, other, err := GenerateKeyPair()
	s.Require().NoError(err)
	out := s.stego(pub, Text(string(bytes.Repeat([]byte("x"), 2000))), s.pcmCarrier(4*FrameDuration), WithFormat("flac"))
	right, rf := s.record()
	wrong, wf := s.record()

	s.extractWith(out, priv, WithCatchProgress(rf))
	s.Empty(s.extractWith(out, other, WithCatchProgress(wf)))

	var scanned []Progress
	for _, p := range *right {
		if p.Stage == ProgressScan {
			scanned = append(scanned, p)
		}
	}
	s.Equal(scanned, *wrong)
}

func (s *ProgressSuite) extractWith(stream, priv []byte, opts ...CatcherOption) []Result {
	s.T().Helper()
	c, err := NewCatcher(priv, opts...)
	s.Require().NoError(err)
	res, err := c.Extract(context.Background(), s.openTemp(stream))
	s.Require().NoError(err)
	return res
}

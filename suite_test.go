package mist

import (
	"bytes"
	"context"
	"io"
	"os"
	"time"

	"github.com/iSerganov/mist/internal/av"
	"github.com/iSerganov/mist/internal/codec"
	"github.com/iSerganov/mist/internal/codec/vorbis"
	"github.com/stretchr/testify/suite"
)

const testRate = 44100

// audioSuite carries the fixtures both the Emitter and Catcher suites need:
// a plain carrier, a stego stream built from one, and a file on disk.
type audioSuite struct {
	suite.Suite
}

func (s *audioSuite) requireLibav() {
	if !av.Available() {
		s.T().Skip("libav Vorbis encoder not available")
	}
}

// carrier returns a plain Ogg Vorbis stream lasting d.
func (s *audioSuite) carrier(d time.Duration) []byte {
	s.T().Helper()
	c := vorbis.New()
	enc, err := c.NewEncoder(codec.DefaultVorbis)
	s.Require().NoError(err)
	defer func() { _ = enc.Close() }()
	info := enc.(interface{ Params() codec.Params }).Params()

	n := int(d.Seconds() * testRate)
	pkts, err := enc.Encode(testSine(testRate, 2, n, 440))
	s.Require().NoError(err)
	flushed, err := enc.Flush()
	s.Require().NoError(err)

	rc, err := muxPackets(info, append(pkts, flushed...))
	s.Require().NoError(err)
	defer func() { _ = rc.Close() }()
	raw, err := io.ReadAll(rc)
	s.Require().NoError(err)
	return raw
}

// stego returns an Ogg stream carrying payload for pub.
func (s *audioSuite) stego(pub []byte, payload Payload, carrier []byte) []byte {
	s.T().Helper()
	em, err := NewEmitter(pub)
	s.Require().NoError(err)
	out, err := em.EmbedReader(context.Background(), bytes.NewReader(carrier), payload)
	s.Require().NoError(err)
	defer func() { _ = out.Close() }()
	raw, err := io.ReadAll(out)
	s.Require().NoError(err)
	return raw
}

func (s *audioSuite) writeTemp(b []byte) string {
	s.T().Helper()
	f, err := os.CreateTemp(s.T().TempDir(), "mist-*.ogg")
	s.Require().NoError(err)
	_, err = f.Write(b)
	s.Require().NoError(err)
	s.Require().NoError(f.Close())
	return f.Name()
}

func (s *audioSuite) openTemp(b []byte) *os.File {
	s.T().Helper()
	f, err := os.Open(s.writeTemp(b))
	s.Require().NoError(err)
	s.T().Cleanup(func() { _ = f.Close() })
	return f
}

func testSine(rate, ch, n int, freq float64) codec.PCM {
	planes := make([][]float32, ch)
	for c := range planes {
		planes[c] = make([]float32, n)
		for i := range planes[c] {
			planes[c][i] = float32(sineAt(freq, float64(i), float64(rate)))
		}
	}
	return codec.PCM{
		Planes:     planes,
		NbSamples:  n,
		Channels:   ch,
		SampleRate: rate,
		Format:     codec.SampleFmtFLTP,
	}
}

func sineAt(freq, i, rate float64) float64 {
	return 0.2 * (2*(((i*freq/rate)+0.25)-float64(int((i*freq/rate)+0.25))) - 1)
}

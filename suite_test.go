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
	info := enc.(*av.Encoder).Info()

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

// pcmCarrier returns a plain 16-bit WAV of deterministic noise lasting d.
// Unlike carrier it needs no encoder at all, so the lossless paths can be
// exercised on a build without libvorbis.
func (s *audioSuite) pcmCarrier(d time.Duration) []byte {
	s.T().Helper()
	return wav(testRate, 2, int(d.Seconds()*testRate))
}

// stego returns a stream carrying payload for pub, in the named format
// ("" for the default Ogg Vorbis).
func (s *audioSuite) stego(pub []byte, payload Payload, carrier []byte, opts ...EmitterOption) []byte {
	s.T().Helper()
	em, err := NewEmitter(pub, opts...)
	s.Require().NoError(err)
	out, err := em.EmbedReader(context.Background(), bytes.NewReader(carrier), payload)
	s.Require().NoError(err)
	defer func() { _ = out.Close() }()
	raw, err := io.ReadAll(out)
	s.Require().NoError(err)
	return raw
}

// extract reads every payload back out of a stego stream.
func (s *audioSuite) extract(priv, stream []byte) []Result {
	s.T().Helper()
	c, err := NewCatcher(priv)
	s.Require().NoError(err)
	res, err := c.Extract(context.Background(), s.openTemp(stream))
	s.Require().NoError(err)
	return res
}

// wav builds a 16-bit PCM WAV of pseudo-random noise. Tonal audio yields
// too few usable residues for the Vorbis path, and a fixed seed keeps a
// failure reproducible.
func wav(rate, ch, n int) []byte {
	var b bytes.Buffer
	u16 := func(v uint16) { b.Write([]byte{byte(v), byte(v >> 8)}) }
	u32 := func(v uint32) { b.Write([]byte{byte(v), byte(v >> 8), byte(v >> 16), byte(v >> 24)}) }
	size := n * ch * 2
	b.WriteString("RIFF")
	u32(uint32(36 + size))
	b.WriteString("WAVEfmt ")
	u32(16)
	u16(1)
	u16(uint16(ch))
	u32(uint32(rate))
	u32(uint32(rate * ch * 2))
	u16(uint16(ch * 2))
	u16(16)
	b.WriteString("data")
	u32(uint32(size))
	x := uint32(12345)
	for i := 0; i < n*ch; i++ {
		x = x*1664525 + 1013904223
		u16(uint16(int16(int32(x>>16) % 8000)))
	}
	return b.Bytes()
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

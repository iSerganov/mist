package main

import (
	"bytes"
	"context"
	"encoding/binary"
	"math"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/iSerganov/mist"
	"github.com/iSerganov/mist/internal/av"
	"github.com/iSerganov/mist/internal/codec"
	"github.com/stretchr/testify/suite"
)

type CLISuite struct {
	suite.Suite
	stdout *bytes.Buffer
}

func TestCLISuite(t *testing.T) {
	suite.Run(t, &CLISuite{})
}

func (s *CLISuite) SetupTest() {
	s.stdout = &bytes.Buffer{}
	out = s.stdout
	colorOn = false
}

func (s *CLISuite) TearDownTest() { out = os.Stdout }

// run executes the CLI exactly as a shell would.
func (s *CLISuite) run(args ...string) error {
	s.T().Helper()
	cmd := newRootCmd()
	cmd.SetArgs(args)
	cmd.SetOut(s.stdout)
	cmd.SetErr(s.stdout)
	return cmd.ExecuteContext(context.Background())
}

func (s *CLISuite) requireLibav() {
	if !av.Available() {
		s.T().Skip("libav Vorbis encoder not available")
	}
}

// carrier writes a noisy Ogg Vorbis file: a pure tone yields too few
// residues to embed into, which is a property worth not depending on here.
func (s *CLISuite) carrier(dir string, d time.Duration) string {
	s.T().Helper()
	enc, err := av.NewEncoder(av.AudioInfo{
		CodecID:    av.CodecIDVorbis,
		SampleRate: 44100,
		Channels:   2,
		SampleFmt:  codec.SampleFmtFLTP,
		Bitrate:    128_000,
	})
	s.Require().NoError(err)
	defer func() { _ = enc.Close() }()

	path := filepath.Join(dir, "carrier.ogg")
	f, err := os.Create(path)
	s.Require().NoError(err)
	defer func() { _ = f.Close() }()

	m, err := av.NewMuxer(f, enc.Info())
	s.Require().NoError(err)
	s.Require().NoError(m.WriteHeader())

	total := int(d.Seconds() * 44100)
	size := enc.Info().FrameSize
	if size <= 0 {
		size = 1024
	}
	var seed uint32 = 0x12345678
	for off := 0; off < total; off += size {
		planes := make([][]byte, 2)
		for c := range planes {
			buf := make([]byte, size*4)
			for i := range size {
				seed = seed*1664525 + 1013904223
				v := (float32(seed>>8)/float32(1<<24) - 0.5) * 0.6
				binary.LittleEndian.PutUint32(buf[i*4:], math.Float32bits(v))
			}
			planes[c] = buf
		}
		s.Require().NoError(enc.Send(av.Frame{
			Data: planes, NbSamples: size, Channels: 2,
			SampleRate: 44100, Format: codec.SampleFmtFLTP, PTS: int64(off),
		}))
		s.drain(enc, m)
	}
	_ = enc.Send(av.Frame{})
	s.drain(enc, m)
	s.Require().NoError(m.WriteTrailer())
	s.Require().NoError(m.Close())
	return path
}

func (s *CLISuite) drain(enc *av.Encoder, m *av.Muxer) {
	s.T().Helper()
	for {
		pkt, err := enc.Receive()
		if err != nil {
			return
		}
		s.Require().NoError(m.WritePacket(pkt))
	}
}

func (s *CLISuite) TestKeyRoundTrip() {
	dir := s.T().TempDir()
	path := filepath.Join(dir, "k.key")
	key := bytes.Repeat([]byte{0xab}, mist.PrivateKeySize)

	s.Require().NoError(writeKey(path, key, privatePerm))
	got, err := readKey(path, mist.PrivateKeySize)
	s.Require().NoError(err)
	s.Equal(key, got)

	info, err := os.Stat(path)
	s.Require().NoError(err)
	s.Equal(privatePerm, info.Mode().Perm(), "private keys must not be world readable")
}

func (s *CLISuite) TestReadKeyRejects() {
	dir := s.T().TempDir()
	write := func(name, body string) string {
		p := filepath.Join(dir, name)
		s.Require().NoError(os.WriteFile(p, []byte(body), 0o600))
		return p
	}

	tests := []struct {
		title string
		path  string
	}{
		{"missing file", filepath.Join(dir, "absent.key")},
		{"not hex", write("bad.key", "zzzz")},
		{"wrong length", write("short.key", "abcd")},
		{"empty", write("empty.key", "")},
	}
	for _, tc := range tests {
		s.Run(tc.title, func() {
			_, err := readKey(tc.path, mist.PrivateKeySize)
			s.Error(err)
		})
	}
}

func (s *CLISuite) TestDerivedPaths() {
	tests := []struct {
		title string
		input string
		stego string
		pub   string
		priv  string
	}{
		{"plain name", "song.ogg", "song.stego.ogg", "song.stego.pub", "song.stego.key"},
		{"nested dir", "a/b/song.wav", "a/b/song.stego.ogg", "a/b/song.stego.pub", "a/b/song.stego.key"},
		{"no extension", "song", "song.stego.ogg", "song.stego.pub", "song.stego.key"},
	}
	for _, tc := range tests {
		s.Run(tc.title, func() {
			stego := stegoPath(tc.input)
			s.Equal(tc.stego, stego)
			pub, priv := keyPaths(stego)
			s.Equal(tc.pub, pub)
			s.Equal(tc.priv, priv)
		})
	}
}

func (s *CLISuite) TestRejectsMissingFlags() {
	tests := []struct {
		title string
		args  []string
	}{
		{"embed without input", []string{"embed", "--data", "hi"}},
		{"embed without data", []string{"embed", "--input", "x.ogg"}},
		{"catch without input", []string{"catch", "--key", "k.key"}},
		{"catch without key", []string{"catch", "--input", "x.ogg"}},
	}
	for _, tc := range tests {
		s.Run(tc.title, func() {
			s.Error(s.run(tc.args...))
		})
	}
}

func (s *CLISuite) TestEmbedThenCatch() {
	s.requireLibav()
	dir := s.T().TempDir()
	in := s.carrier(dir, 9*time.Second)
	stego := filepath.Join(dir, "out.ogg")

	s.Require().NoError(s.run("embed", "--input", in, "--data", "the eagle lands at dawn", "--output", stego))

	pub, priv := keyPaths(stego)
	s.FileExists(stego)
	s.FileExists(pub)
	s.FileExists(priv)
	s.Contains(s.stdout.String(), "embedded into")

	s.stdout.Reset()
	s.Require().NoError(s.run("catch", "--input", stego, "--key", priv))
	s.Contains(s.stdout.String(), "the eagle lands at dawn")
	s.Contains(s.stdout.String(), "recovered")
}

func (s *CLISuite) TestCatchWithSuppliedKey() {
	s.requireLibav()
	dir := s.T().TempDir()
	in := s.carrier(dir, 9*time.Second)
	stego := filepath.Join(dir, "out.ogg")

	pub, priv, err := mist.GenerateKeyPair()
	s.Require().NoError(err)
	pubPath := filepath.Join(dir, "peer.pub")
	privPath := filepath.Join(dir, "peer.key")
	s.Require().NoError(writeKey(pubPath, pub, publicPerm))
	s.Require().NoError(writeKey(privPath, priv, privatePerm))

	s.Require().NoError(s.run("embed", "-i", in, "-d", "supplied key", "-o", stego, "-k", pubPath))

	// No keypair is minted when one was supplied.
	derivedPub, _ := keyPaths(stego)
	s.NoFileExists(derivedPub)

	s.stdout.Reset()
	s.Require().NoError(s.run("catch", "-i", stego, "-k", privPath))
	s.Contains(s.stdout.String(), "supplied key")
}

func (s *CLISuite) TestCatchReportsNothingForWrongKey() {
	s.requireLibav()
	dir := s.T().TempDir()
	in := s.carrier(dir, 9*time.Second)
	stego := filepath.Join(dir, "out.ogg")
	s.Require().NoError(s.run("embed", "-i", in, "-d", "secret", "-o", stego))

	_, other, err := mist.GenerateKeyPair()
	s.Require().NoError(err)
	otherPath := filepath.Join(dir, "other.key")
	s.Require().NoError(writeKey(otherPath, other, privatePerm))

	s.stdout.Reset()
	err = s.run("catch", "-i", stego, "-k", otherPath)
	s.Error(err, "a run that recovers nothing exits non-zero")
	s.NotContains(s.stdout.String(), "secret")
}

func (s *CLISuite) TestCatchStopsOnTimeout() {
	s.requireLibav()
	dir := s.T().TempDir()
	in := s.carrier(dir, 9*time.Second)
	stego := filepath.Join(dir, "out.ogg")
	s.Require().NoError(s.run("embed", "-i", in, "-d", "timed", "-o", stego))
	_, priv := keyPaths(stego)

	s.stdout.Reset()
	_ = s.run("catch", "-i", stego, "-k", priv, "--timeout", "5s")
	s.Contains(s.stdout.String(), "5s")
}

func (s *CLISuite) TestDescribeTimeout() {
	tests := []struct {
		title string
		in    time.Duration
		want  string
	}{
		{"zero reads to eof", 0, "none, read to end of stream"},
		{"negative reads to eof", -time.Second, "none, read to end of stream"},
		{"positive", 90 * time.Second, "1m30s"},
	}
	for _, tc := range tests {
		s.Run(tc.title, func() {
			s.Equal(tc.want, describeTimeout(tc.in))
		})
	}
}

func (s *CLISuite) TestStopReason() {
	deadline, cancelDeadline := context.WithTimeout(context.Background(), time.Nanosecond)
	defer cancelDeadline()
	<-deadline.Done()

	cancelled, cancel := context.WithCancel(context.Background())
	cancel()

	tests := []struct {
		title string
		ctx   context.Context
		want  string
	}{
		{"eof", context.Background(), "end of stream"},
		{"timeout", deadline, "timeout reached"},
		{"interrupt", cancelled, "interrupted"},
	}
	for _, tc := range tests {
		s.Run(tc.title, func() {
			s.Equal(tc.want, stopReason(tc.ctx))
		})
	}
}

func (s *CLISuite) TestHumanBytes() {
	tests := []struct {
		title string
		in    int64
		want  string
	}{
		{"bytes", 512, "512 B"},
		{"kibibytes", 2048, "2.0 KiB"},
		{"mebibytes", 5 << 20, "5.0 MiB"},
	}
	for _, tc := range tests {
		s.Run(tc.title, func() {
			s.Equal(tc.want, humanBytes(tc.in))
		})
	}
}

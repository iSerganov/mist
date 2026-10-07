package mist

import (
	"bytes"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/suite"
)

type DecodeSuite struct {
	audioSuite
}

func TestDecodeSuite(t *testing.T) {
	suite.Run(t, &DecodeSuite{})
}

// TestLengthMatchesFFmpeg checks that a lossy carrier decodes to as many
// samples as the ffmpeg command line decodes it to. The encoder delay,
// padding and pre-skip a container records are samples ffmpeg drops, and
// an output longer than a plain encode of the same carrier is a difference
// from one.
func (s *DecodeSuite) TestLengthMatchesFFmpeg() {
	s.requireLibav()
	if _, err := exec.LookPath("ffmpeg"); err != nil {
		s.T().Skip("ffmpeg CLI not on PATH")
	}
	tests := []struct {
		title string
		ext   string
		args  []string
	}{
		{"mp3 with encoder delay and padding", ".mp3", []string{"-c:a", "libmp3lame"}},
		{"opus with pre-skip", ".opus", []string{"-c:a", "libopus"}},
		{"aac in an m4a", ".m4a", []string{"-c:a", "aac"}},
		{"vorbis in an ogg whose last page is cut by its granule position", ".ogg", []string{"-c:a", "libvorbis"}},
	}
	dir := s.T().TempDir()
	source := filepath.Join(dir, "source.wav")
	s.Require().NoError(os.WriteFile(source, s.pcmCarrier(3*time.Second), 0o600))
	for _, tc := range tests {
		s.Run(tc.title, func() {
			carrier := filepath.Join(dir, "carrier"+tc.ext)
			args := append([]string{"-nostdin", "-loglevel", "error", "-y", "-i", source}, tc.args...)
			if msg, err := exec.Command("ffmpeg", append(args, carrier)...).CombinedOutput(); err != nil {
				s.T().Skipf("this ffmpeg cannot write %s: %s", tc.ext, bytes.TrimSpace(msg))
			}
			raw, err := os.ReadFile(carrier)
			s.Require().NoError(err)
			pcm, _, _, err := decodeCarrier(bytes.NewReader(raw))
			s.Require().NoError(err)

			want, err := exec.Command("ffmpeg", "-nostdin", "-loglevel", "error", "-i", carrier, "-map", "0:a:0", "-f", "f32le", "-").Output()
			s.Require().NoError(err)
			s.Equal(len(want)/4/pcm.Channels, pcm.NbSamples)
		})
	}
}

// TestOutputMatchesFFmpeg checks the two pipeline differences a warden can
// see without looking at the payload: container tags, and a WAV tail of
// digital zeros. Both are properties of the encode, not of the embedding.
func (s *DecodeSuite) TestOutputMatchesFFmpeg() {
	s.requireLibav()
	if _, err := exec.LookPath("ffmpeg"); err != nil || exec.Command("ffprobe", "-version").Run() != nil {
		s.T().Skip("ffmpeg and ffprobe are required")
	}
	dir := s.T().TempDir()
	sources := []struct {
		name string
		args []string
	}{
		{"tagged.wav", []string{
			"-c:a", "pcm_s16le",
			"-metadata", "title=Hello",
			"-metadata", "artist=A",
			"-metadata", "album=B",
			"-metadata", "comment=C",
			"-metadata", "date=2020",
			"-metadata", "track=1",
			"-metadata", "TLEN=200",
			"-metadata", "creation_time=2020-01-01T00:00:00.000000Z",
			"-metadata", "company_name=Acme",
			"-metadata", "product_name=Widget",
			"-metadata", "product_version=1",
		}},
		{"tagged.opus", []string{
			"-c:a", "libopus",
			"-metadata", "ALBUM=Metal",
			"-metadata", "TITLE=Song",
			"-metadata", "GENRE=x",
			"-metadata", "ENCODER=opusenc",
			"-metadata", "ENCODER_OPTIONS=--bitrate 96",
			"-metadata", "track=1",
			"-metadata", "TRACKTOTAL=12",
		}},
	}
	for _, src := range sources {
		s.Run(src.name, func() {
			in := filepath.Join(dir, src.name)
			args := append([]string{
				"-nostdin", "-loglevel", "error", "-y",
				"-f", "lavfi", "-i", "sine=frequency=440:sample_rate=44100:duration=1",
			}, src.args...)
			if msg, err := exec.Command("ffmpeg", append(args, in)...).CombinedOutput(); err != nil {
				s.T().Skipf("this ffmpeg cannot write %s: %s", src.name, bytes.TrimSpace(msg))
			}
			raw, err := os.ReadFile(in)
			s.Require().NoError(err)
			targets := []struct {
				container string
				ext       string
				lossless  bool
			}{
				{"flac", ".flac", true},
				{"wav", ".wav", true},
				{"ogg", ".ogg", false},
			}
			for _, tg := range targets {
				s.Run(tg.container, func() {
					ff := filepath.Join(dir, src.name+tg.ext)
					ffArgs := []string{"-nostdin", "-loglevel", "error", "-y", "-i", in, "-map", "0:a:0"}
					if tg.container == "ogg" {
						ffArgs = append(ffArgs, "-c:a", "libvorbis")
					}
					ffArgs = append(ffArgs, "-f", tg.container, ff)
					msg, err := exec.Command("ffmpeg", ffArgs...).CombinedOutput()
					s.Require().NoError(err, "%s", msg)

					got := filepath.Join(dir, src.name+".mist"+tg.ext)
					s.Require().NoError(os.WriteFile(got, s.reencode(raw, tg.container), 0o600))
					s.Equal(probeTags(s.T(), ff), probeTags(s.T(), got))
					if tg.lossless {
						s.equalPCM(ff, got)
					}
				})
			}
		})
	}
}

func (s *DecodeSuite) reencode(raw []byte, container string) []byte {
	s.T().Helper()
	pcm, info, meta, err := decodeCarrier(bytes.NewReader(raw))
	s.Require().NoError(err)
	target, err := lookupFormat(container, "")
	s.Require().NoError(err)
	enc, err := openEncoder(target, pcm, info)
	s.Require().NoError(err)
	defer func() { _ = enc.Close() }()
	if target.Lossless {
		s.Require().NoError(enc.Snap(pcm.Planes, pcm.Frames))
	}
	rc, err := encodeAndMux(enc, pcm, meta, nil)
	s.Require().NoError(err)
	defer func() { _ = rc.Close() }()
	out, err := io.ReadAll(rc)
	s.Require().NoError(err)
	return out
}

func (s *DecodeSuite) equalPCM(ff, mist string) {
	s.T().Helper()
	left, _, _, err := decodeCarrier(bytes.NewReader(mustRead(s.T(), ff)))
	s.Require().NoError(err)
	right, _, _, err := decodeCarrier(bytes.NewReader(mustRead(s.T(), mist)))
	s.Require().NoError(err)
	s.Require().Equal(left.NbSamples, right.NbSamples)
	s.Equal(trailingZeros(left.Planes), trailingZeros(right.Planes))
	for c := range left.Planes {
		s.Equal(left.Planes[c], right.Planes[c])
	}
}

func trailingZeros(planes [][]float32) int {
	if len(planes) == 0 || len(planes[0]) == 0 {
		return 0
	}
	n := 0
	for i := len(planes[0]) - 1; i >= 0; i-- {
		for _, p := range planes {
			if i < len(p) && p[i] != 0 {
				return n
			}
		}
		n++
	}
	return n
}

func probeTags(t *testing.T, path string) string {
	t.Helper()
	out, err := exec.Command("ffprobe", "-v", "error", "-show_entries", "format_tags:stream_tags", "-of", "compact=p=0", path).Output()
	if err != nil {
		t.Fatalf("ffprobe %s: %v", path, err)
	}
	return string(out)
}

func mustRead(t *testing.T, path string) []byte {
	t.Helper()
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return b
}

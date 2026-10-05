package mist

import (
	"bytes"
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
			pcm, _, err := decodeCarrier(bytes.NewReader(raw))
			s.Require().NoError(err)

			want, err := exec.Command("ffmpeg", "-nostdin", "-loglevel", "error", "-i", carrier, "-map", "0:a:0", "-f", "f32le", "-").Output()
			s.Require().NoError(err)
			s.Equal(len(want)/4/pcm.Channels, pcm.NbSamples)
		})
	}
}

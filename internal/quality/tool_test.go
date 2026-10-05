package quality

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/suite"
)

const fakeFFmpeg = `for a; do last=$a; done
: > "$last"`

const requireWavs = `for a; do case $a in *.wav) [ -f "$a" ] || exit 3;; esac; done
`

type ToolSuite struct {
	suite.Suite
	bin string
	ref string
	deg string
}

func TestToolSuite(t *testing.T) {
	suite.Run(t, &ToolSuite{})
}

func (s *ToolSuite) SetupTest() { s.resetBins() }

func (s *ToolSuite) SetupSubTest() { s.resetBins() }

func (s *ToolSuite) resetBins() {
	s.bin = s.T().TempDir()
	s.T().Setenv("PATH", s.bin)
	s.T().Setenv("MIST_FFMPEG", "")
	s.T().Setenv("MIST_VISQOL", "")
	s.T().Setenv("MIST_PEAQ", "")
	data := s.T().TempDir()
	s.ref, s.deg = filepath.Join(data, "ref.flac"), filepath.Join(data, "deg.ogg")
	s.Require().NoError(os.WriteFile(s.ref, []byte("ref"), 0o600))
	s.Require().NoError(os.WriteFile(s.deg, []byte("deg"), 0o600))
}

func (s *ToolSuite) install(name, body string) {
	s.Require().NoError(os.WriteFile(filepath.Join(s.bin, name), []byte("#!/bin/sh\n"+body+"\n"), 0o700))
}

func (s *ToolSuite) TestScoreParsesToolOutput() {
	tests := []struct {
		title  string
		tool   Tool
		output string
		want   float64
	}{
		{"visqol mos-lqo", ViSQOL(), `echo "MOS-LQO:		4.25"`, 4.25},
		{"peaq objective difference grade", PEAQ(), `echo "Objective Difference Grade: -0.512"; echo "Distortion Index: 1.9"`, -0.512},
	}
	for _, tc := range tests {
		s.Run(tc.title, func() {
			s.install("ffmpeg", fakeFFmpeg)
			s.install(tc.tool.bin, requireWavs+tc.output)
			got, err := tc.tool.Score(context.Background(), s.ref, s.deg)
			s.Require().NoError(err)
			s.InDelta(tc.want, got, 1e-9)
		})
	}
}

func (s *ToolSuite) TestScoreFailures() {
	tests := []struct {
		title   string
		ffmpeg  string
		tool    string
		wantErr error
	}{
		{"tool missing", fakeFFmpeg, "", ErrUnavailable},
		{"ffmpeg missing", "", `echo "MOS-LQO: 4"`, ErrUnavailable},
		{"tool prints no score", fakeFFmpeg, `echo "warming up"`, ErrNoScore},
		{"tool exits non-zero", fakeFFmpeg, `echo boom; exit 1`, nil},
		{"ffmpeg exits non-zero", `exit 1`, `echo "MOS-LQO: 4"`, nil},
	}
	for _, tc := range tests {
		s.Run(tc.title, func() {
			if tc.ffmpeg != "" {
				s.install("ffmpeg", tc.ffmpeg)
			}
			if tc.tool != "" {
				s.install("visqol", tc.tool)
			}
			_, err := ViSQOL().Score(context.Background(), s.ref, s.deg)
			s.Require().Error(err)
			if tc.wantErr != nil {
				s.ErrorIs(err, tc.wantErr)
			}
		})
	}
}

func (s *ToolSuite) TestAvailable() {
	tests := []struct {
		title   string
		install []string
		want    bool
	}{
		{"nothing installed", nil, false},
		{"only the tool", []string{"peaq"}, false},
		{"only ffmpeg", []string{"ffmpeg"}, false},
		{"tool and ffmpeg", []string{"peaq", "ffmpeg"}, true},
	}
	for _, tc := range tests {
		s.Run(tc.title, func() {
			for _, name := range tc.install {
				s.install(name, "true")
			}
			s.Equal(tc.want, PEAQ().Available())
		})
	}
}

func (s *ToolSuite) TestConfiguredBinaryPaths() {
	ffmpeg := filepath.Join(s.bin, "custom-ffmpeg")
	visqol := filepath.Join(s.bin, "custom-visqol")
	s.Require().NoError(os.WriteFile(ffmpeg, []byte("#!/bin/sh\n"+fakeFFmpeg+"\n"), 0o700))
	s.Require().NoError(os.WriteFile(visqol, []byte("#!/bin/sh\n"+requireWavs+`echo "MOS-LQO: 4.5"`+"\n"), 0o700))
	s.T().Setenv("MIST_FFMPEG", ffmpeg)
	s.T().Setenv("MIST_VISQOL", visqol)

	tool := ViSQOL()
	s.Equal(ffmpeg, tool.ffmpeg)
	s.Equal(visqol, tool.bin)
	score, err := tool.Score(context.Background(), s.ref, s.deg)
	s.Require().NoError(err)
	s.Equal(4.5, score)
}

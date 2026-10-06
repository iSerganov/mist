package mist

import (
	"bytes"
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"github.com/iSerganov/mist/internal/trace"
	"github.com/stretchr/testify/suite"
)

type AnalyzeSuite struct {
	audioSuite
}

func TestAnalyzeSuite(t *testing.T) {
	suite.Run(t, &AnalyzeSuite{})
}

// file writes b to a temporary file with the given extension.
func (s *AnalyzeSuite) file(b []byte, ext string) string {
	s.T().Helper()
	path := filepath.Join(s.T().TempDir(), "track"+ext)
	s.Require().NoError(os.WriteFile(path, b, 0o600))
	return path
}

// clean is carrier through Mist's own encoder with nothing embedded.
func (s *AnalyzeSuite) clean(carrier []byte, format string) []byte {
	s.T().Helper()
	target, err := lookupFormat(format, "")
	s.Require().NoError(err)
	pcm, info, err := decodeCarrier(bytes.NewReader(carrier))
	s.Require().NoError(err)
	out, err := plainEncode(target, pcm, info)
	s.Require().NoError(err)
	return out
}

func (s *AnalyzeSuite) TestProbe() {
	s.requireLibav()
	tests := []struct {
		title    string
		data     []byte
		ext      string
		codec    string
		channels int
		wantErr  bool
	}{
		{"reads a wav without decoding it", s.pcmCarrier(time.Second), ".wav", "pcm_s16le", 2, false},
		{"rejects a text file", []byte("not audio at all\n"), ".txt", "", 0, true},
	}
	for _, tc := range tests {
		s.Run(tc.title, func() {
			src, err := Probe(s.file(tc.data, tc.ext))
			if tc.wantErr {
				s.Error(err)
				return
			}
			s.Require().NoError(err)
			s.Equal(tc.codec, src.Codec)
			s.Equal(tc.channels, src.Channels)
			s.True(src.Lossless)
		})
	}
}

func (s *AnalyzeSuite) TestBlindAnalysisRunsEveryStage() {
	s.requireLibav()
	pub, _, err := GenerateKeyPair()
	s.Require().NoError(err)
	carrier := s.pcmCarrier(10 * time.Second)
	tests := []struct {
		title string
		data  []byte
	}{
		{"a clean flac", s.clean(carrier, "flac")},
		{"a mist flac", s.stego(pub, Text("hello"), carrier, WithFormat("flac"))},
	}
	for _, tc := range tests {
		s.Run(tc.title, func() {
			var seen []string
			a, err := Analyze(context.Background(), s.file(tc.data, ".flac"), AnalyzeOptions{
				Progress: func(st Stage) { seen = append(seen, st.Name) },
			})
			s.Require().NoError(err)
			s.Equal(BasisBlind, a.Basis)
			s.GreaterOrEqual(a.Probability, 0.0)
			s.LessOrEqual(a.Probability, 1.0)
			s.NotEmpty(a.Calibration)
			names := map[string]Stage{}
			for _, st := range a.Stages {
				names[st.Name] = st
			}
			s.Len(seen, len(a.Stages))
			for _, ran := range []string{"format", "chi-square", "spa", "rs", "hcf-com", "classifier", "markov", "rich", "dead tail", "container", "silence"} {
				s.Equal(StageRan, names[ran].Status, ran)
			}
			for _, notRun := range []string{"known cover", "sdr", "selection", "key-aware", "invariance", "perceptual"} {
				s.Equal(StageNotRun, names[notRun].Status, notRun)
				s.NotEmpty(names[notRun].Note, notRun)
			}
			s.Contains(names, "fingerprint")
		})
	}
}

func (s *AnalyzeSuite) TestKnownCoverDecides() {
	s.requireLibav()
	pub, _, err := GenerateKeyPair()
	s.Require().NoError(err)
	carrier := s.pcmCarrier(10 * time.Second)
	original := s.file(carrier, ".wav")
	other := s.file(s.pcmCarrier(11*time.Second), ".wav")
	tests := []struct {
		title     string
		data      []byte
		ext       string
		reference string
		basis     Basis
		want      float64
	}{
		{"a mist flac against its original", s.stego(pub, Text("hello"), carrier, WithFormat("flac")), ".flac", original, BasisKnownCover, knownStego},
		{"a clean flac against its original", s.clean(carrier, "flac"), ".flac", original, BasisKnownCover, knownClean},
		{"a mist flac against another recording", s.stego(pub, Text("hello"), carrier, WithFormat("flac")), ".flac", other, BasisBlind, -1},
		{"a mist ogg against its original", s.stego(pub, Text("hello"), carrier), ".ogg", original, BasisKnownCover, knownStego},
		{"a clean ogg against its original", s.clean(carrier, "ogg"), ".ogg", original, BasisKnownCover, knownClean},
	}
	for _, tc := range tests {
		s.Run(tc.title, func() {
			a, err := Analyze(context.Background(), s.file(tc.data, tc.ext), AnalyzeOptions{Reference: tc.reference})
			s.Require().NoError(err)
			s.Equal(tc.basis, a.Basis)
			if tc.want >= 0 {
				s.Equal(tc.want, a.Probability)
			}
			for _, st := range a.Stages {
				if st.Name == "known cover" {
					s.NotEmpty(st.Note)
				}
			}
		})
	}
}

func (s *AnalyzeSuite) TestLossyFormatCannotCarryAMessage() {
	s.requireLibav()
	ffmpeg, err := exec.LookPath("ffmpeg")
	if err != nil {
		s.T().Skip("ffmpeg not on PATH")
	}
	mp3, err := trace.Encode(context.Background(), ffmpeg, s.pcmCarrier(2*time.Second), ".wav", "mp3", "")
	if err != nil {
		s.T().Skipf("this ffmpeg cannot write mp3: %v", err)
	}
	a, err := Analyze(context.Background(), s.file(mp3, ".mp3"), AnalyzeOptions{})
	s.Require().NoError(err)
	s.Equal(BasisFormat, a.Basis)
	s.Equal(0.0, a.Probability)
	s.Require().NotEmpty(a.Stages)
	s.Contains(a.Stages[0].Note, "cannot")
	names := map[string]Stage{}
	for _, st := range a.Stages {
		names[st.Name] = st
	}
	s.Equal(StageRan, names["container"].Status)
	s.Equal(ScopeAny, names["container"].Scope)
	s.Equal(StageRan, names["mp3 header bits"].Status)
	s.Equal(StageNotRun, names["silence"].Status)
	s.Equal(SuspicionNone, a.Suspicion)
}

func (s *AnalyzeSuite) TestOtherToolsAreFound() {
	s.requireLibav()
	silent := make([]int16, 2*testRate*2)
	for i := 1000; i < len(silent); i += 101 {
		silent[i] = int16(1 - 2*(i%2))
	}
	tests := []struct {
		title string
		data  []byte
		stage string
		want  Suspicion
	}{
		{"a clean wav", s.pcmCarrier(2 * time.Second), "container", SuspicionNone},
		{"a wav with a file appended", append(s.pcmCarrier(2*time.Second), bytes.Repeat([]byte("secret!"), 40)...), "container", SuspicionHigh},
		{"lsb replacement in silence", wavFile(testRate, 2, silent), "silence", SuspicionHigh},
	}
	for _, tc := range tests {
		s.Run(tc.title, func() {
			a, err := Analyze(context.Background(), s.file(tc.data, ".wav"), AnalyzeOptions{})
			s.Require().NoError(err)
			s.Equal(tc.want, a.Suspicion)
			for _, st := range a.Stages {
				if st.Name == tc.stage {
					s.Equal(tc.want, st.Suspicion)
					s.Equal(ScopeAny, st.Scope)
					if tc.want > SuspicionNone {
						s.NotEmpty(st.Findings)
					}
					return
				}
			}
			s.Fail("no stage " + tc.stage)
		})
	}
}

func (s *AnalyzeSuite) TestCoverDiff() {
	base := []int32{10, -4, 7, 0, 3, 100, -50, 8, 9, 11}
	edit := func(n int, d int32) []int32 {
		out := append([]int32(nil), base...)
		for i := range n {
			out[i] += d
		}
		return out
	}
	long := make([]int32, 1000)
	sparse := append([]int32(nil), long...)
	sparse[500] = 1
	tests := []struct {
		title    string
		clean    []int32
		suspect  []int32
		lossless bool
		want     float64
	}{
		{"identical values are clean", base, base, true, 0},
		{"one ±1 in a thousand is mist's footprint", long, sparse, true, 1},
		{"a change of two is not mist's", long, func() []int32 { v := append([]int32(nil), long...); v[3] = 2; return v }(), true, -1},
		{"a vorbis residue may move by more than one", long, func() []int32 { v := append([]int32(nil), long...); v[3] = 3; return v }(), false, 1},
		{"changes past mist's rate mean another recording", base, edit(5, 1), true, -1},
		{"different lengths are another recording", base, base[:9], true, -1},
	}
	for _, tc := range tests {
		s.Run(tc.title, func() {
			got := coverDiff(tc.clean, tc.suspect, tc.lossless)
			s.Equal(tc.want, got.Score)
			s.NotEmpty(got.Note)
		})
	}
}

func (s *AnalyzeSuite) TestCalibrationLookup() {
	cal := calibration{Formats: []calibratedFormat{
		{Format: "ogg/vorbis", Codec: "vorbis", Container: "ogg"},
		{Format: "flac", Codec: "flac", Container: "flac", Lossless: true},
		{Format: "wav/pcm_s16le", Codec: "pcm_s16le", Container: "wav", Lossless: true},
	}}
	tests := []struct {
		title     string
		codec     string
		container string
		lossless  bool
		want      string
		exact     bool
		found     bool
	}{
		{"the same codec matches exactly", "vorbis", "ogg", false, "ogg/vorbis", true, true},
		{"a deeper pcm in the same container borrows it", "pcm_s24le", "wav", true, "wav/pcm_s16le", false, true},
		{"another lossless codec borrows the first lossless format", "alac", "mov", true, "flac", false, true},
		{"a lossy codec finds nothing", "mp3", "mp3", false, "", true, false},
	}
	for _, tc := range tests {
		s.Run(tc.title, func() {
			f, note, ok := cal.lookup(tc.codec, tc.container, tc.lossless)
			s.Equal(tc.found, ok)
			s.Equal(tc.want, f.Format)
			s.Equal(tc.exact, note == "")
		})
	}
}

func (s *AnalyzeSuite) TestEmbeddedCalibrationLoads() {
	cal, err := loadCalibration(calibrationJSON)
	s.Require().NoError(err)
	s.NotEmpty(cal.Formats)
	for _, f := range cal.Formats {
		for _, w := range trainedWardens {
			st, ok := f.stage(w.name)
			s.True(ok, "%s %s", f.Format, w.name)
			s.NotNil(st.Model, "%s %s", f.Format, w.name)
		}
	}
}

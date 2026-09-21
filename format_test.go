package mist

import (
	"testing"

	"github.com/iSerganov/mist/internal/av"
	"github.com/stretchr/testify/suite"
)

type FormatSuite struct {
	suite.Suite
}

func TestFormatSuite(t *testing.T) { suite.Run(t, new(FormatSuite)) }

func (s *FormatSuite) SetupTest() {
	if !av.Available() {
		s.T().Skip("libav not available")
	}
}

// A name is resolved the way ffmpeg resolves one: as an output path, a
// container short name, or an encoder name, with the codec overriding the
// container's default.
func (s *FormatSuite) TestLookup() {
	tests := []struct {
		title     string
		name      string
		codec     string
		container string
		wantCodec string
		lossless  bool
	}{
		{"empty is the default", "", "", "ogg", "vorbis", false},
		{"container short name", "flac", "", "flac", "flac", true},
		{"output path extension", "song.flac", "", "flac", "flac", true},
		{"path in a directory", "/tmp/a.b/song.wav", "", "wav", "pcm_s16le", true},
		{"explicit codec overrides", "song.caf", "alac", "caf", "alac", true},
		{"codec alone picks a container", "", "flac", "flac", "flac", true},
	}
	for _, tc := range tests {
		s.Run(tc.title, func() {
			f, err := LookupFormat(tc.name, tc.codec)
			s.Require().NoError(err)
			s.Equal(tc.container, f.Container)
			s.Equal(tc.wantCodec, f.Codec)
			s.Equal(tc.lossless, f.Lossless)
			s.NotEmpty(f.Ext)
		})
	}
}

// Vorbis is the only lossy target: it is the only one whose bitstream
// Mist rewrites. Everything else lossy must be refused loudly, because
// the alternative is a file with nothing in it.
func (s *FormatSuite) TestRejects() {
	tests := []struct {
		title string
		name  string
		codec string
	}{
		{"lossy container", "song.mp3", ""},
		{"lossy codec", "song.mka", "aac"},
		{"unknown extension", "song.zzz", ""},
		{"unknown codec", "song.flac", "notacodec"},
	}
	for _, tc := range tests {
		s.Run(tc.title, func() {
			_, err := LookupFormat(tc.name, tc.codec)
			s.ErrorIs(err, ErrUnsupportedCodec)
		})
	}
}

// NewEmitter resolves the target up front, so a bad format costs nothing
// and does not surface after a carrier has been decoded.
func (s *FormatSuite) TestEmitterRejectsFormatEarly() {
	pub, _, err := GenerateKeyPair()
	s.Require().NoError(err)

	_, err = NewEmitter(pub, WithFormat("song.mp3"))
	s.ErrorIs(err, ErrUnsupportedCodec)
}

// The listing is what `mist formats` shows, so it must be non-empty, free
// of duplicates, and made only of targets LookupFormat accepts.
func (s *FormatSuite) TestFormatsAreUsable() {
	all := Formats()
	s.Require().NotEmpty(all)

	seen := map[Format]bool{}
	lossless := 0
	for _, f := range all {
		s.False(seen[f], "duplicate target %s", f)
		seen[f] = true
		s.NotEmpty(f.Ext)
		got, err := LookupFormat(f.Container, f.Codec)
		s.Require().NoError(err, "listed target %s does not resolve", f)
		s.Equal(f.Lossless, got.Lossless)
		if f.Lossless {
			lossless++
		}
	}
	s.Positive(lossless, "a build with libav encodes at least one lossless codec")
}

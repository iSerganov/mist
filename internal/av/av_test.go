package av

import (
	"testing"

	"github.com/iSerganov/mist/internal/codec"
	"github.com/stretchr/testify/suite"
)

type AVSuite struct {
	suite.Suite
}

func TestAVSuite(t *testing.T) {
	suite.Run(t, &AVSuite{})
}

func (s *AVSuite) TestAudioInfoParamsVorbis() {
	info := AudioInfo{
		CodecID:    CodecIDVorbis,
		SampleRate: 44100,
		Channels:   2,
		SampleFmt:  codec.SampleFmtFLTP,
		Bitrate:    128_000,
	}
	p := info.Params()
	s.Equal(codec.IDVorbis, p.ID)
	s.Equal(44100, p.SampleRate)
	s.Equal(2, p.Channels)
}

func (s *AVSuite) TestInit() {
	s.T().Skip("TODO: Init talks to libav and is idempotent")
}

func (s *AVSuite) TestOpenDemuxer() {
	s.T().Skip("TODO: demux an Ogg Vorbis file and report AudioInfo")
}

func (s *AVSuite) TestDemuxMuxRoundTrip() {
	s.T().Skip("TODO: packets survive demux → mux without coefficient changes")
}

func (s *AVSuite) TestEncodeDecodePCM() {
	s.T().Skip("TODO: PCM → Vorbis → PCM via libav (reference path, no stego)")
}

func (s *AVSuite) TestCustomIO() {
	s.T().Skip("TODO: OpenDemuxerReader / NewMuxer over io.Reader/Writer")
}

package codec

import (
	"testing"

	"github.com/stretchr/testify/suite"
)

type CodecSuite struct {
	suite.Suite
}

func TestCodecSuite(t *testing.T) {
	suite.Run(t, &CodecSuite{})
}

func (s *CodecSuite) TestDefaultVorbis() {
	p := DefaultVorbis
	s.Equal(IDVorbis, p.ID)
	s.Equal(44100, p.SampleRate)
	s.Equal(2, p.Channels)
	s.Equal(SampleFmtFLTP, p.Format)
}

func (s *CodecSuite) TestSampleFormatMatchesLibav() {
	s.Equal(SampleFormat(-1), SampleFmtNone)
	s.Equal(SampleFormat(8), SampleFmtFLTP)
}

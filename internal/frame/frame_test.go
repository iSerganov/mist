package frame

import (
	"testing"
	"time"

	"github.com/stretchr/testify/suite"
)

type FrameSuite struct {
	suite.Suite
}

func TestFrameSuite(t *testing.T) {
	suite.Run(t, &FrameSuite{})
}

func (s *FrameSuite) TestParamsSamples() {
	p := Params{SampleRate: 44100, Channels: 2, Duration: time.Second}
	s.Equal(44100, p.Samples())
}

func (s *FrameSuite) TestSplit() {
	s.T().Skip("TODO: split a PCM length into consecutive stego frames")
}

func (s *FrameSuite) TestCandidatePhases() {
	s.T().Skip("TODO: phase offsets covering one frame duration")
}

func (s *FrameSuite) TestCapacity() {
	s.T().Skip("TODO: eligible * density / 8 - overhead")
}

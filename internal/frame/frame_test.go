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
	p := Params{SampleRate: 100, Channels: 1, Duration: time.Second}
	got := Split(250, p)
	s.Require().Len(got, 3)
	s.Equal(0, got[0].PCMStart)
	s.Equal(100, got[0].PCMEnd)
	s.Equal(int64(2), got[2].Index)
	s.Equal(250, got[2].PCMEnd)
	s.Empty(Split(0, p))
	s.Empty(Split(10, Params{}))
}

func (s *FrameSuite) TestCandidatePhases() {
	got := CandidatePhases(8*time.Second, 2*time.Second)
	s.Require().Len(got, 4)
	s.Equal(time.Duration(0), got[0].Offset)
	s.Equal(6*time.Second, got[3].Offset)
	s.Len(CandidatePhases(time.Second, 0), 1)
}

func (s *FrameSuite) TestCapacity() {
	s.Equal(90, Capacity(8000, 0.10, 10))
	s.Equal(0, Capacity(10, 0.10, 10))
	s.Equal(0, Capacity(0, 0.10, 0))
}

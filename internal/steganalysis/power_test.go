package steganalysis

import (
	"testing"

	"github.com/stretchr/testify/suite"
)

type PowerSuite struct {
	suite.Suite
}

func TestPowerSuite(t *testing.T) {
	suite.Run(t, &PowerSuite{})
}

func (s *PowerSuite) TestStrongShiftIsReachable() {
	pos := []float64{1.0, 1.2, 0.9, 1.1, 1.3, 0.8, 1.05, 1.15}
	neg := []float64{0.0, 0.1, -0.1, 0.05, 0.2, -0.05, 0.0, 0.1}
	n, reached := FamiliesForPower(pos, neg, 0.9, 0.8, 1)
	s.T().Log(n, reached)
	s.True(reached)
	s.Positive(n)
}

func (s *PowerSuite) TestTiedPilotDoesNotClaimPower() {
	pos := []float64{0.5, 0.5, 0.5, 0.5, 0.5, 0.5}
	neg := []float64{0.5, 0.5, 0.5, 0.5, 0.5, 0.5}
	_, reached := FamiliesForPower(pos, neg, 0.55, 0.9, 1)
	s.False(reached)
}

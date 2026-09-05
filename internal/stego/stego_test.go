package stego

import (
	"testing"

	"github.com/stretchr/testify/suite"
)

type StegoSuite struct {
	suite.Suite
}

func TestStegoSuite(t *testing.T) {
	suite.Run(t, &StegoSuite{})
}

func (s *StegoSuite) TestLSB() {
	s.Equal(uint8(0), LSB(2))
	s.Equal(uint8(1), LSB(3))
}

func (s *StegoSuite) TestMatchLeavesMatchingBit() {
	s.T().Skip("TODO: Match is a no-op when the LSB already equals the target bit")
}

func (s *StegoSuite) TestMatchFlipsByOne() {
	s.T().Skip("TODO: Match changes coeff by exactly ±1 when the LSB differs")
}

func (s *StegoSuite) TestSelectorDeterministic() {
	s.T().Skip("TODO: same position key yields the same index sequence")
}

func (s *StegoSuite) TestFillerLength() {
	s.T().Skip("TODO: Filler returns n cryptographically random bytes")
}

func (s *StegoSuite) TestEmbedExtractRoundTrip() {
	s.T().Skip("TODO: bits survive a black-box or patched embed then extract")
}

func (s *StegoSuite) TestConstantDensity() {
	s.T().Skip("TODO: empty payload perturbs the same coefficient count as a full one")
}

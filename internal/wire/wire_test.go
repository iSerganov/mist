package wire

import (
	"testing"

	"github.com/stretchr/testify/suite"
)

type WireSuite struct {
	suite.Suite
}

func TestWireSuite(t *testing.T) {
	suite.Run(t, &WireSuite{})
}

func (s *WireSuite) TestEnvelopePrefixSize() {
	s.Equal(EphemeralPubSize+NonceSize, EnvelopePrefix)
	s.Equal(EnvelopePrefix+TagSize, EnvelopeOverhead)
	s.Equal(6, PayloadHeaderSize)
}

func (s *WireSuite) TestMarshalPayload() {
	s.T().Skip("TODO: version + type + big-endian length + data")
}

func (s *WireSuite) TestUnmarshalPayloadRejectsShort() {
	s.T().Skip("TODO: buffers shorter than the header return ErrShort")
}

func (s *WireSuite) TestEnvelopeMarshalRoundTrip() {
	s.T().Skip("TODO: Marshal then UnmarshalEnvelope")
}

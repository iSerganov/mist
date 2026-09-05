package crypto

import (
	"testing"

	"github.com/stretchr/testify/suite"
)

type CryptoSuite struct {
	suite.Suite
}

func TestCryptoSuite(t *testing.T) {
	suite.Run(t, &CryptoSuite{})
}

func (s *CryptoSuite) TestGenerateX25519() {
	s.T().Skip("TODO: generate a 32-byte X25519 keypair")
}

func (s *CryptoSuite) TestSealOpenRoundTrip() {
	s.T().Skip("TODO: Seal then Open recovers the plaintext")
}

func (s *CryptoSuite) TestOpenWrongKey() {
	s.T().Skip("TODO: Open with the wrong private key fails without a distinguished error")
}

func (s *CryptoSuite) TestFreshEphemeralPerSeal() {
	s.T().Skip("TODO: two Seals of the same plaintext produce different envelopes")
}

func (s *CryptoSuite) TestSignVerify() {
	s.T().Skip("TODO: Ed25519 sign/verify round trip")
}

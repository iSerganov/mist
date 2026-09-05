package mist

import (
	"testing"
	"time"

	"github.com/stretchr/testify/suite"
)

type PayloadSuite struct {
	suite.Suite
}

func TestPayloadSuite(t *testing.T) {
	suite.Run(t, &PayloadSuite{})
}

func (s *PayloadSuite) TestPayloadTypeValues() {
	tests := []struct {
		name string
		got  PayloadType
		want PayloadType
	}{
		{"text", PayloadText, 0x01},
		{"image", PayloadImage, 0x02},
		{"audio", PayloadAudio, 0x03},
		{"file", PayloadFile, 0x04},
	}
	for _, tt := range tests {
		s.Run(tt.name, func() {
			s.Equal(tt.want, tt.got)
		})
	}
}

func (s *PayloadSuite) TestProtocolConstants() {
	s.Equal(byte(1), Version)
	s.Equal(32, PublicKeySize)
	s.Equal(32, PrivateKeySize)
	s.Equal(PublicKeySize+NonceSize+TagSize, EnvelopeOverhead)
	s.Equal(8*time.Second, FrameDuration)
}

func (s *PayloadSuite) TestTextPayload() {
	p := Text("hello")
	s.Equal(PayloadText, p.Type)
	s.Equal("hello", string(p.Data))
}

func (s *PayloadSuite) TestGenerateKeyPair() {
	s.T().Skip("TODO: X25519 key generation")
	pub, priv, err := GenerateKeyPair()
	s.NoError(err)
	s.Len(pub, PublicKeySize)
	s.Len(priv, PrivateKeySize)
}

func (s *PayloadSuite) TestGenerateSigningKeyPair() {
	s.T().Skip("TODO: Ed25519 key generation")
	pub, priv, err := GenerateSigningKeyPair()
	s.NoError(err)
	s.Len(pub, SigningPublicKeySize)
	s.Len(priv, SigningPrivateKeySize)
}

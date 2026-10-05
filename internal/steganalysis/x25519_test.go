package steganalysis

import (
	"crypto/ecdh"
	"crypto/rand"
	"testing"

	"github.com/stretchr/testify/suite"
)

type X25519Suite struct {
	suite.Suite
}

func TestX25519Suite(t *testing.T) {
	suite.Run(t, &X25519Suite{})
}

func (s *X25519Suite) TestHonestKeysPass() {
	for range 50 {
		k, err := ecdh.X25519().GenerateKey(rand.Reader)
		s.Require().NoError(err)
		s.True(HonestX25519(k.PublicKey().Bytes()))
	}
}

func (s *X25519Suite) TestRandomBytesMostlyFail() {
	const n = 400
	passed := 0
	for range n {
		b := make([]byte, 32)
		_, err := rand.Read(b)
		s.Require().NoError(err)
		if HonestX25519(b) {
			passed++
		}
	}
	s.Less(passed, n/8, "about 1 in 32 random strings should pass")
}

func (s *X25519Suite) TestRejects() {
	honest, err := ecdh.X25519().GenerateKey(rand.Reader)
	s.Require().NoError(err)
	topBit := honest.PublicKey().Bytes()
	topBit[31] |= 0x80
	tests := []struct {
		title string
		key   []byte
	}{
		{"an honest key with its top bit set", topBit},
		{"a key that is too short", make([]byte, 31)},
		{"u=2, which is outside the prime-order subgroup", append([]byte{2}, make([]byte, 31)...)},
	}
	for _, tc := range tests {
		s.Run(tc.title, func() { s.False(HonestX25519(tc.key)) })
	}
}

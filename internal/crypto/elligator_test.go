package crypto

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/suite"
)

type ElligatorSuite struct {
	suite.Suite
}

func TestElligatorSuite(t *testing.T) {
	suite.Run(t, &ElligatorSuite{})
}

func (s *ElligatorSuite) TestRepresentativeMapsBackToThePublicKey() {
	represented := 0
	for range 300 {
		pub, _, err := GenerateX25519()
		s.Require().NoError(err)
		rep, ok, err := representative(pub)
		s.Require().NoError(err)
		if !ok {
			continue
		}
		represented++
		s.Equal(pub, publicFromRepresentative(rep))
	}
	s.InDelta(150, represented, 45, "about half of all keys have a representative")
}

func (s *ElligatorSuite) TestRepresentativeLooksUniform() {
	const n = 600
	var ones [256]int
	for range n {
		rep, _, err := generateEphemeral()
		s.Require().NoError(err)
		for bit := range 256 {
			ones[bit] += int(rep[bit/8] >> (bit % 8) & 1)
		}
	}
	for bit, c := range ones {
		s.InDelta(n/2, c, 5*0.5*24.5, "bit %d is biased: %d of %d", bit, c, n)
	}
}

func (s *ElligatorSuite) TestEveryEphemeralKeyOpens() {
	pub, priv, err := GenerateX25519()
	s.Require().NoError(err)
	for range 100 {
		env, err := Seal([]byte("hi"), pub)
		s.Require().NoError(err)
		plain, err := Open(env, priv)
		s.Require().NoError(err)
		s.True(bytes.Equal([]byte("hi"), plain))
	}
}

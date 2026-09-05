package vorbis

import (
	"testing"

	"github.com/iSerganov/mist/internal/codec"
	"github.com/stretchr/testify/suite"
)

type VorbisSuite struct {
	suite.Suite
}

func TestVorbisSuite(t *testing.T) {
	suite.Run(t, &VorbisSuite{})
}

func (s *VorbisSuite) TestCodecIdentity() {
	c := New()
	s.Equal(Name, c.Name())
	s.Equal(codec.IDVorbis, c.ID())
}

func (s *VorbisSuite) TestResidues() {
	s.T().Skip("TODO: parse quantized residues and stop before inverse MDCT")
}

func (s *VorbisSuite) TestParseSetup() {
	s.T().Skip("TODO: decode ident / comment / setup headers")
}

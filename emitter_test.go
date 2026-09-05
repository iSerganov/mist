package mist

import (
	"context"
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/suite"
)

type EmitterSuite struct {
	suite.Suite
}

func TestEmitterSuite(t *testing.T) {
	suite.Run(t, &EmitterSuite{})
}

func (s *EmitterSuite) TestNewEmitterRejectsNilKey() {
	e, err := NewEmitter(nil)
	s.Nil(e)
	s.ErrorIs(err, ErrInvalidKey)
}

func (s *EmitterSuite) TestNewEmitterCopiesPublicKey() {
	key := []byte{1, 2, 3}
	e, err := NewEmitter(key)
	s.Require().NoError(err)
	key[0] = 9
	s.Equal(byte(1), e.pub[0])
	s.Len(e.pub, 3)
}

func (s *EmitterSuite) TestWithSenderAuthCopiesKey() {
	sender := []byte{4, 5, 6}
	e, err := NewEmitter([]byte{1}, WithSenderAuth(sender))
	s.Require().NoError(err)
	sender[0] = 9
	s.Equal(byte(4), e.senderPriv[0])
}

func (s *EmitterSuite) TestEmbed() {
	s.T().Skip("TODO: encrypt and embed into a path or URL carrier")
	e, err := NewEmitter(make([]byte, PublicKeySize))
	s.Require().NoError(err)
	r, err := e.Embed(context.Background(), "testdata/empty.ogg", Text("hello"))
	s.Require().NoError(err)
	s.NoError(r.Close())
}

func (s *EmitterSuite) TestEmbedReader() {
	s.T().Skip("TODO: encrypt and embed into an io.Reader carrier")
	e, err := NewEmitter(make([]byte, PublicKeySize))
	s.Require().NoError(err)
	r, err := e.EmbedReader(context.Background(), strings.NewReader(""), Text("hello"))
	s.Require().NoError(err)
	s.NoError(r.Close())
}

func (s *EmitterSuite) TestEmbedFile() {
	s.T().Skip("TODO: encrypt and embed into an *os.File carrier")
	e, err := NewEmitter(make([]byte, PublicKeySize))
	s.Require().NoError(err)
	r, err := e.EmbedFile(context.Background(), os.Stdin, Text("hello"))
	s.Require().NoError(err)
	s.NoError(r.Close())
}

func (s *EmitterSuite) TestFrameCapacity() {
	s.T().Skip("TODO: capacity from eligible residues, density, and envelope overhead")
	s.Greater(FrameCapacity(), 0)
}

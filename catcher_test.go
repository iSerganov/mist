package mist

import (
	"context"
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/suite"
)

type CatcherSuite struct {
	suite.Suite
}

func TestCatcherSuite(t *testing.T) {
	suite.Run(t, &CatcherSuite{})
}

func (s *CatcherSuite) TestNewCatcherRejectsNilKey() {
	c, err := NewCatcher(nil)
	s.Nil(c)
	s.Error(err)
}

func (s *CatcherSuite) TestNewCatcherCopiesPrivateKey() {
	key := []byte{1, 2, 3}
	c, err := NewCatcher(key)
	s.Require().NoError(err)
	key[0] = 9
	s.Equal(byte(1), c.priv[0])
	s.Len(c.priv, 3)
}

func (s *CatcherSuite) TestListen() {
	s.T().Skip("TODO: scan a file or live URL and emit authenticated frames")
	c, err := NewCatcher(make([]byte, PrivateKeySize))
	s.Require().NoError(err)
	ch, err := c.Listen(context.Background(), "testdata/empty.ogg")
	s.Require().NoError(err)
	for range ch {
	}
}

func (s *CatcherSuite) TestListenReader() {
	s.T().Skip("TODO: scan an io.Reader bitstream and emit authenticated frames")
	c, err := NewCatcher(make([]byte, PrivateKeySize))
	s.Require().NoError(err)
	ch, err := c.ListenReader(context.Background(), strings.NewReader(""))
	s.Require().NoError(err)
	for range ch {
	}
}

func (s *CatcherSuite) TestExtract() {
	s.T().Skip("TODO: synchronously collect every authenticated frame from a file")
	c, err := NewCatcher(make([]byte, PrivateKeySize))
	s.Require().NoError(err)
	results, err := c.Extract(context.Background(), os.Stdin)
	s.Require().NoError(err)
	s.NotNil(results)
}

func (s *CatcherSuite) TestListenCancel() {
	s.T().Skip("TODO: cancelled context closes the result channel")
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	c, err := NewCatcher(make([]byte, PrivateKeySize))
	s.Require().NoError(err)
	ch, err := c.Listen(ctx, "https://radio.example.com/live.ogg")
	s.Require().NoError(err)
	for range ch {
	}
	s.Error(ctx.Err())
}

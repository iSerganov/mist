package mist

import (
	"strings"
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
	s.Equal(PublicKeySize+LengthSize+NonceSize+TagSize, EnvelopeOverhead)
	s.Equal(8*time.Second, FrameDuration)
}

func (s *PayloadSuite) TestTextPayload() {
	p := Text("hello")
	s.Equal(PayloadText, p.Type)
	s.Equal("hello", string(p.Data))
}

func (s *PayloadSuite) TestTypedConstructors() {
	tests := []struct {
		title    string
		payload  Payload
		wantType PayloadType
	}{
		{"image", Image([]byte("jpeg bytes")), PayloadImage},
		{"audio", Audio([]byte("wav bytes")), PayloadAudio},
	}
	for _, tc := range tests {
		s.Run(tc.title, func() {
			s.Equal(tc.wantType, tc.payload.Type)
			s.NotEmpty(tc.payload.Data)
		})
	}
}

func (s *PayloadSuite) TestFileNameRoundTrip() {
	tests := []struct {
		title string
		name  string
		data  []byte
	}{
		{"plain name", "report.pdf", []byte("contents")},
		{"empty name", "", []byte("contents")},
		{"empty content", "empty.txt", nil},
		{"unicode name", "отчёт.txt", []byte{0, 1, 2}},
		{"longest allowed name", strings.Repeat("a", 65535), []byte("x")},
	}
	for _, tc := range tests {
		s.Run(tc.title, func() {
			p, err := File(tc.name, tc.data)
			s.Require().NoError(err)
			s.Equal(PayloadFile, p.Type)

			name, data, err := p.FileName()
			s.Require().NoError(err)
			s.Equal(tc.name, name)
			s.Equal(string(tc.data), string(data))
		})
	}
}

func (s *PayloadSuite) TestFileRejectsOverlongName() {
	_, err := File(strings.Repeat("a", 65536), nil)
	s.ErrorIs(err, ErrInvalidPayload)
}

func (s *PayloadSuite) TestFileNameRejectsMalformedPayloads() {
	tests := []struct {
		title   string
		payload Payload
	}{
		{"text payload", Text("hello")},
		{"image payload", Image([]byte{0, 3, 'a', 'b', 'c'})},
		{"file without name length", Payload{Type: PayloadFile, Data: []byte{0}}},
		{"file with name cut short", Payload{Type: PayloadFile, Data: []byte{0, 9, 'a', 'b'}}},
	}
	for _, tc := range tests {
		s.Run(tc.title, func() {
			_, _, err := tc.payload.FileName()
			s.ErrorIs(err, ErrInvalidPayload)
		})
	}
}

func (s *PayloadSuite) TestGenerateKeyPair() {
	pub, priv, err := GenerateKeyPair()
	s.NoError(err)
	s.Len(pub, PublicKeySize)
	s.Len(priv, PrivateKeySize)
	pub2, priv2, err := GenerateKeyPair()
	s.NoError(err)
	s.NotEqual(pub, pub2)
	s.NotEqual(priv, priv2)
}

func (s *PayloadSuite) TestGenerateSigningKeyPair() {
	pub, priv, err := GenerateSigningKeyPair()
	s.NoError(err)
	s.Len(pub, SigningPublicKeySize)
	s.Len(priv, SigningPrivateKeySize)
}

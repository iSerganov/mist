package wire

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/suite"
)

type WireSuite struct {
	suite.Suite
}

func TestWireSuite(t *testing.T) {
	suite.Run(t, &WireSuite{})
}

func (s *WireSuite) TestSizes() {
	s.Equal(6, PayloadHeaderSize)
	s.Equal(EphemeralPubSize+NonceSize, EnvelopePrefix)
	s.Equal(EnvelopePrefix+TagSize, EnvelopeOverhead)
	s.Equal(64, SignatureSize)
}

func (s *WireSuite) TestMarshalUnmarshalPayload() {
	sig := bytes.Repeat([]byte{0xab}, SignatureSize)
	tests := []struct {
		title string
		in    Payload
	}{
		{"empty data", Payload{Version: CurrentVersion, Type: 0x01}},
		{"text", Payload{Version: CurrentVersion, Type: 0x01, Data: []byte("hello")}},
		{"binary", Payload{Version: CurrentVersion, Type: 0x04, Data: []byte{0x00, 0xff, 0x01}}},
		{"with signature", Payload{Version: CurrentVersion, Type: 0x01, Data: []byte("hi"), Signature: sig}},
		{"signature only", Payload{Version: CurrentVersion, Type: 0x01, Signature: sig}},
		{"default version", Payload{Type: 0x02, Data: []byte("x")}},
	}
	for _, tc := range tests {
		s.Run(tc.title, func() {
			raw, err := MarshalPayload(tc.in)
			s.Require().NoError(err)
			got, err := UnmarshalPayload(raw)
			s.Require().NoError(err)
			wantVer := tc.in.Version
			if wantVer == 0 {
				wantVer = CurrentVersion
			}
			s.Equal(wantVer, got.Version)
			s.Equal(tc.in.Type, got.Type)
			s.Equal(tc.in.Data, got.Data)
			if len(tc.in.Signature) == 0 {
				s.Empty(got.Signature)
			} else {
				s.Equal(tc.in.Signature, got.Signature)
			}
		})
	}
}

func (s *WireSuite) TestMarshalPayloadRejects() {
	tests := []struct {
		title   string
		in      Payload
		wantErr error
	}{
		{"bad version", Payload{Version: 2, Type: 0x01}, ErrVersion},
		{"short signature", Payload{Type: 0x01, Signature: []byte("nope")}, ErrLength},
		{"long signature", Payload{Type: 0x01, Signature: bytes.Repeat([]byte{1}, SignatureSize+1)}, ErrLength},
	}
	for _, tc := range tests {
		s.Run(tc.title, func() {
			_, err := MarshalPayload(tc.in)
			s.ErrorIs(err, tc.wantErr)
		})
	}
}

func (s *WireSuite) TestUnmarshalPayloadRejects() {
	ok, err := MarshalPayload(Payload{Type: 0x01, Data: []byte("hi")})
	s.Require().NoError(err)

	short := ok[:PayloadHeaderSize-1]
	badVer := append([]byte{}, ok...)
	badVer[0] = 9
	overLen := append([]byte{}, ok...)
	PutUint32(overLen[2:6], 100)
	oddTail := append(append([]byte{}, ok...), 0x01, 0x02)

	tests := []struct {
		title   string
		in      []byte
		wantErr error
	}{
		{"nil", nil, ErrShort},
		{"empty", []byte{}, ErrShort},
		{"short header", short, ErrShort},
		{"bad version", badVer, ErrVersion},
		{"length past end", overLen, ErrLength},
		{"odd leftover", oddTail, ErrLength},
	}
	for _, tc := range tests {
		s.Run(tc.title, func() {
			_, err := UnmarshalPayload(tc.in)
			s.ErrorIs(err, tc.wantErr)
		})
	}
}

func (s *WireSuite) TestUnmarshalPayloadDoesNotAliasInput() {
	raw, err := MarshalPayload(Payload{Type: 0x01, Data: []byte("abc")})
	s.Require().NoError(err)
	got, err := UnmarshalPayload(raw)
	s.Require().NoError(err)
	raw[PayloadHeaderSize] = 'Z'
	s.Equal([]byte("abc"), got.Data)
}

func (s *WireSuite) TestEnvelopeRoundTrip() {
	tests := []struct {
		title string
		ct    []byte
	}{
		{"tag only", bytes.Repeat([]byte{0x11}, TagSize)},
		{"short message", append(bytes.Repeat([]byte{0x22}, 8), bytes.Repeat([]byte{0x33}, TagSize)...)},
		{"longer", bytes.Repeat([]byte{0x44}, 64+TagSize)},
	}
	for _, tc := range tests {
		s.Run(tc.title, func() {
			var env Envelope
			copy(env.EphemeralPub[:], bytes.Repeat([]byte{0xaa}, EphemeralPubSize))
			copy(env.Nonce[:], bytes.Repeat([]byte{0xbb}, NonceSize))
			env.Ciphertext = tc.ct
			got, err := UnmarshalEnvelope(env.Marshal())
			s.Require().NoError(err)
			s.Equal(env.EphemeralPub, got.EphemeralPub)
			s.Equal(env.Nonce, got.Nonce)
			s.Equal(tc.ct, got.Ciphertext)
		})
	}
}

func (s *WireSuite) TestUnmarshalEnvelopeRejects() {
	tests := []struct {
		title string
		in    []byte
	}{
		{"nil", nil},
		{"empty", []byte{}},
		{"one short", make([]byte, EnvelopeOverhead-1)},
		{"prefix only", make([]byte, EnvelopePrefix)},
	}
	for _, tc := range tests {
		s.Run(tc.title, func() {
			_, err := UnmarshalEnvelope(tc.in)
			s.ErrorIs(err, ErrShort)
		})
	}
}

func (s *WireSuite) TestPutUint32() {
	b := make([]byte, 4)
	PutUint32(b, 0x01020304)
	s.Equal([]byte{1, 2, 3, 4}, b)
	s.Equal(uint32(0x01020304), Uint32(b))
}

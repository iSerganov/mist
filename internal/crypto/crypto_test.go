package crypto

import (
	"bytes"
	"testing"

	"github.com/iSerganov/mist/internal/wire"
	"github.com/stretchr/testify/suite"
)

type CryptoSuite struct {
	suite.Suite
}

func TestCryptoSuite(t *testing.T) {
	suite.Run(t, &CryptoSuite{})
}

func (s *CryptoSuite) TestGenerateX25519() {
	pub, priv, err := GenerateX25519()
	s.Require().NoError(err)
	s.Len(pub, PublicKeySize)
	s.Len(priv, PrivateKeySize)

	pub2, priv2, err := GenerateX25519()
	s.Require().NoError(err)
	s.NotEqual(pub, pub2)
	s.NotEqual(priv, priv2)
}

func (s *CryptoSuite) TestSharedSecret() {
	alicePub, alicePriv, err := GenerateX25519()
	s.Require().NoError(err)
	bobPub, bobPriv, err := GenerateX25519()
	s.Require().NoError(err)

	ab, err := SharedSecret(alicePriv, bobPub)
	s.Require().NoError(err)
	ba, err := SharedSecret(bobPriv, alicePub)
	s.Require().NoError(err)
	s.Equal(ab, ba)
	s.Len(ab, 32)
}

func (s *CryptoSuite) TestSharedSecretRejects() {
	_, priv, err := GenerateX25519()
	s.Require().NoError(err)
	pub, _, err := GenerateX25519()
	s.Require().NoError(err)

	tests := []struct {
		title    string
		priv, pk []byte
	}{
		{"nil priv", nil, pub},
		{"short priv", []byte{1}, pub},
		{"nil pub", priv, nil},
		{"short pub", priv, []byte{1, 2, 3}},
		{"zero pub", priv, make([]byte, PublicKeySize)},
	}
	for _, tc := range tests {
		s.Run(tc.title, func() {
			_, err := SharedSecret(tc.priv, tc.pk)
			s.ErrorIs(err, ErrInvalidKey)
		})
	}
}

func (s *CryptoSuite) TestDeriveKeys() {
	shared := bytes.Repeat([]byte{0x07}, 32)
	a, err := DeriveKeys(shared)
	s.Require().NoError(err)
	b, err := DeriveKeys(shared)
	s.Require().NoError(err)
	s.Equal(a.AEAD, b.AEAD)
	s.Equal(a.Position, b.Position)
	s.Len(a.AEAD, AEADKeySize)
	s.Len(a.Position, PositionKeySize)
	s.NotEqual(a.AEAD, a.Position)

	other := append([]byte{}, shared...)
	other[0] ^= 0xff
	c, err := DeriveKeys(other)
	s.Require().NoError(err)
	s.NotEqual(a.AEAD, c.AEAD)
	s.NotEqual(a.Position, c.Position)
}

func (s *CryptoSuite) TestDeriveKeysRejectsEmpty() {
	_, err := DeriveKeys(nil)
	s.ErrorIs(err, ErrInvalidKey)
}

func (s *CryptoSuite) TestSealOpenRoundTrip() {
	pub, priv, err := GenerateX25519()
	s.Require().NoError(err)

	tests := []struct {
		title string
		plain []byte
	}{
		{"empty", nil},
		{"empty slice", []byte{}},
		{"short", []byte("hi")},
		{"text", []byte("hello mist")},
		{"binary", bytes.Repeat([]byte{0x00, 0xff}, 64)},
	}
	for _, tc := range tests {
		s.Run(tc.title, func() {
			env, err := Seal(tc.plain, pub)
			s.Require().NoError(err)
			s.Require().NotNil(env)
			s.Len(env.Ciphertext, len(tc.plain)+wire.TagSize)
			got, err := Open(env, priv)
			s.Require().NoError(err)
			if len(tc.plain) == 0 {
				s.Empty(got)
			} else {
				s.Equal(tc.plain, got)
			}
		})
	}
}

func (s *CryptoSuite) TestOpenRejects() {
	pub, priv, err := GenerateX25519()
	s.Require().NoError(err)
	_, otherPriv, err := GenerateX25519()
	s.Require().NoError(err)
	env, err := Seal([]byte("secret"), pub)
	s.Require().NoError(err)

	tamperedCT := *env
	tamperedCT.Ciphertext = append([]byte{}, env.Ciphertext...)
	tamperedCT.Ciphertext[0] ^= 0x01

	tamperedNonce := *env
	tamperedNonce.Nonce[0] ^= 0x01

	tamperedPub := *env
	tamperedPub.EphemeralPub[0] ^= 0x01

	tests := []struct {
		title string
		env   *wire.Envelope
		priv  []byte
	}{
		{"nil envelope", nil, priv},
		{"wrong key", env, otherPriv},
		{"bad priv size", env, []byte{1}},
		{"tampered ciphertext", &tamperedCT, priv},
		{"tampered nonce", &tamperedNonce, priv},
		{"tampered eph pub", &tamperedPub, priv},
	}
	for _, tc := range tests {
		s.Run(tc.title, func() {
			_, err := Open(tc.env, tc.priv)
			s.ErrorIs(err, ErrOpen)
		})
	}
}

func (s *CryptoSuite) TestSealFreshEphemeral() {
	pub, _, err := GenerateX25519()
	s.Require().NoError(err)
	plain := []byte("same")
	a, err := Seal(plain, pub)
	s.Require().NoError(err)
	b, err := Seal(plain, pub)
	s.Require().NoError(err)
	s.NotEqual(a.EphemeralPub, b.EphemeralPub)
	s.NotEqual(a.Nonce, b.Nonce)
	s.NotEqual(a.Ciphertext, b.Ciphertext)
}

func (s *CryptoSuite) TestSealRejectsBadRecipient() {
	tests := []struct {
		title string
		pub   []byte
	}{
		{"nil", nil},
		{"short", []byte{1, 2}},
		{"zero", make([]byte, PublicKeySize)},
	}
	for _, tc := range tests {
		s.Run(tc.title, func() {
			_, err := Seal([]byte("x"), tc.pub)
			s.ErrorIs(err, ErrInvalidKey)
		})
	}
}

func (s *CryptoSuite) TestEnvelopeWireRoundTrip() {
	pub, priv, err := GenerateX25519()
	s.Require().NoError(err)
	env, err := Seal([]byte("frame"), pub)
	s.Require().NoError(err)
	parsed, err := wire.UnmarshalEnvelope(env.Marshal())
	s.Require().NoError(err)
	got, err := Open(&parsed, priv)
	s.Require().NoError(err)
	s.Equal([]byte("frame"), got)
}

func (s *CryptoSuite) TestSignVerify() {
	pub, priv, err := GenerateEd25519()
	s.Require().NoError(err)
	s.Len(pub, SigningPublicKeySize)
	s.Len(priv, SigningPrivateKeySize)

	tests := []struct {
		title string
		msg   []byte
	}{
		{"empty", nil},
		{"text", []byte("hello")},
		{"binary", bytes.Repeat([]byte{0x5a}, 100)},
	}
	for _, tc := range tests {
		s.Run(tc.title, func() {
			sig, err := Sign(priv, tc.msg)
			s.Require().NoError(err)
			s.Len(sig, SignatureSize)
			s.NoError(Verify(pub, tc.msg, sig))
		})
	}
}

func (s *CryptoSuite) TestVerifyRejects() {
	pub, priv, err := GenerateEd25519()
	s.Require().NoError(err)
	otherPub, _, err := GenerateEd25519()
	s.Require().NoError(err)
	sig, err := Sign(priv, []byte("msg"))
	s.Require().NoError(err)
	badSig := append([]byte{}, sig...)
	badSig[0] ^= 0x01

	tests := []struct {
		title         string
		pub, msg, sig []byte
	}{
		{"wrong message", pub, []byte("other"), sig},
		{"wrong key", otherPub, []byte("msg"), sig},
		{"tampered sig", pub, []byte("msg"), badSig},
		{"short pub", pub[:8], []byte("msg"), sig},
		{"short sig", pub, []byte("msg"), sig[:8]},
		{"nil sig", pub, []byte("msg"), nil},
	}
	for _, tc := range tests {
		s.Run(tc.title, func() {
			s.ErrorIs(Verify(tc.pub, tc.msg, tc.sig), ErrVerify)
		})
	}
}

func (s *CryptoSuite) TestSignRejectsBadKey() {
	_, err := Sign([]byte("short"), []byte("msg"))
	s.ErrorIs(err, ErrInvalidKey)
}

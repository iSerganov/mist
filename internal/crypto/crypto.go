// Package crypto is the hybrid box: X25519, HKDF, ChaCha20-Poly1305, Ed25519.
// Seal uses a fresh ephemeral key per call. Decrypt failures are always ErrOpen.
package crypto

import (
	"crypto/ecdh"
	"crypto/ed25519"
	"crypto/hkdf"
	"crypto/rand"
	"crypto/sha256"
	"fmt"
	"io"

	"github.com/iSerganov/mist/internal/wire"
	"golang.org/x/crypto/chacha20poly1305"
)

const (
	PublicKeySize         = 32
	PrivateKeySize        = 32
	SigningPublicKeySize  = ed25519.PublicKeySize
	SigningPrivateKeySize = ed25519.PrivateKeySize
	SignatureSize         = ed25519.SignatureSize
	NonceSize             = chacha20poly1305.NonceSize
	AEADKeySize           = chacha20poly1305.KeySize
	PositionKeySize       = 32
)

const (
	hkdfSalt = "mist-v1"
	infoAEAD = "mist-aead-v1"
	infoPos  = "mist-pos-v1"
)

// Derived is the HKDF split of one ECDH shared secret.
type Derived struct {
	AEAD     []byte
	Position []byte
}

// GenerateX25519 creates a recipient or ephemeral X25519 keypair.
func GenerateX25519() (pub, priv []byte, err error) {
	k, err := ecdh.X25519().GenerateKey(rand.Reader)
	if err != nil {
		return nil, nil, fmt.Errorf("crypto: generate x25519: %w", err)
	}
	return k.PublicKey().Bytes(), k.Bytes(), nil
}

// SharedSecret performs X25519 ECDH.
func SharedSecret(priv, peerPub []byte) ([]byte, error) {
	curve := ecdh.X25519()
	sk, err := curve.NewPrivateKey(priv)
	if err != nil {
		return nil, fmt.Errorf("%w: private", ErrInvalidKey)
	}
	pk, err := curve.NewPublicKey(peerPub)
	if err != nil {
		return nil, fmt.Errorf("%w: public", ErrInvalidKey)
	}
	secret, err := sk.ECDH(pk)
	if err != nil {
		return nil, fmt.Errorf("%w: ecdh", ErrInvalidKey)
	}
	return secret, nil
}

// DeriveKeys runs HKDF-SHA256 twice with distinct info strings.
func DeriveKeys(shared []byte) (Derived, error) {
	if len(shared) == 0 {
		return Derived{}, ErrInvalidKey
	}
	salt := []byte(hkdfSalt)
	aead, err := hkdf.Key(sha256.New, shared, salt, infoAEAD, AEADKeySize)
	if err != nil {
		return Derived{}, fmt.Errorf("crypto: hkdf aead: %w", err)
	}
	pos, err := hkdf.Key(sha256.New, shared, salt, infoPos, PositionKeySize)
	if err != nil {
		return Derived{}, fmt.Errorf("crypto: hkdf position: %w", err)
	}
	return Derived{AEAD: aead, Position: pos}, nil
}

// Seal encrypts plaintext for recipientPub and returns a wire envelope.
func Seal(plaintext, recipientPub []byte) (*wire.Envelope, error) {
	ephPub, ephPriv, err := GenerateX25519()
	if err != nil {
		return nil, err
	}
	shared, err := SharedSecret(ephPriv, recipientPub)
	if err != nil {
		return nil, err
	}
	keys, err := DeriveKeys(shared)
	if err != nil {
		return nil, err
	}
	aead, err := chacha20poly1305.New(keys.AEAD)
	if err != nil {
		return nil, fmt.Errorf("crypto: aead: %w", err)
	}
	var nonce [NonceSize]byte
	if _, err := io.ReadFull(rand.Reader, nonce[:]); err != nil {
		return nil, fmt.Errorf("crypto: nonce: %w", err)
	}
	ct := aead.Seal(nil, nonce[:], plaintext, ephPub)
	env := &wire.Envelope{Ciphertext: ct}
	copy(env.EphemeralPub[:], ephPub)
	env.Nonce = nonce
	return env, nil
}

// Open decrypts and authenticates env. Failures are ErrOpen, never a typed miss.
func Open(env *wire.Envelope, recipientPriv []byte) ([]byte, error) {
	if env == nil {
		return nil, ErrOpen
	}
	shared, err := SharedSecret(recipientPriv, env.EphemeralPub[:])
	if err != nil {
		return nil, ErrOpen
	}
	keys, err := DeriveKeys(shared)
	if err != nil {
		return nil, ErrOpen
	}
	aead, err := chacha20poly1305.New(keys.AEAD)
	if err != nil {
		return nil, ErrOpen
	}
	plain, err := aead.Open(nil, env.Nonce[:], env.Ciphertext, env.EphemeralPub[:])
	if err != nil {
		return nil, ErrOpen
	}
	return plain, nil
}

// GenerateEd25519 creates a signing keypair for optional sender authentication.
func GenerateEd25519() (pub, priv []byte, err error) {
	pub, priv, err = ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return nil, nil, fmt.Errorf("crypto: generate ed25519: %w", err)
	}
	return pub, priv, nil
}

// Sign returns an Ed25519 signature over msg.
func Sign(priv, msg []byte) ([]byte, error) {
	if len(priv) != SigningPrivateKeySize {
		return nil, ErrInvalidKey
	}
	return ed25519.Sign(ed25519.PrivateKey(priv), msg), nil
}

// Verify checks an Ed25519 signature.
func Verify(pub, msg, sig []byte) error {
	if len(pub) != SigningPublicKeySize || len(sig) != SignatureSize {
		return ErrVerify
	}
	if !ed25519.Verify(ed25519.PublicKey(pub), msg, sig) {
		return ErrVerify
	}
	return nil
}

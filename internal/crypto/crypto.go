// Package crypto implements Mist's hybrid box and optional sender signatures.
//
// Seal:
//  1. generate ephemeral X25519 keypair
//  2. ECDH(ephemeral_priv, recipient_pub) → shared secret
//  3. HKDF(shared) → AEAD key and independent position-selection key
//  4. ChaCha20-Poly1305 encrypt plaintext
//
// Open is the inverse using the recipient's static private key.
// A fresh ephemeral keypair is required per stego frame so looping a
// payload across a live stream does not produce correlatable ciphertext.
package crypto

import "github.com/iSerganov/mist/internal/wire"

// Key sizes, kept here so internal packages do not import the root module.
const (
	PublicKeySize         = 32
	PrivateKeySize        = 32
	SigningPublicKeySize  = 32
	SigningPrivateKeySize = 64
	SignatureSize         = 64
	NonceSize             = 12
	AEADKeySize           = 32
	PositionKeySize       = 32
)

// Derived holds the two HKDF outputs of a single ECDH shared secret.
// The AEAD key and the position-selection key must be cryptographically
// independent so a warden who somehow recovered one learns nothing of the other.
type Derived struct {
	AEAD     []byte
	Position []byte
}

// GenerateX25519 creates a recipient (or ephemeral) X25519 keypair.
func GenerateX25519() (pub, priv []byte, err error) {
	return nil, nil, errUnimplemented
}

// SharedSecret performs X25519 ECDH.
func SharedSecret(priv, peerPub []byte) ([]byte, error) {
	return nil, errUnimplemented
}

// DeriveKeys runs HKDF on a shared secret and splits the output.
func DeriveKeys(shared []byte) (Derived, error) {
	return Derived{}, errUnimplemented
}

// Seal encrypts plaintext for recipientPub and returns the wire envelope
// (ephemeral public key || nonce || ciphertext||tag).
func Seal(plaintext, recipientPub []byte) (*wire.Envelope, error) {
	return nil, errUnimplemented
}

// Open decrypts and authenticates env with the recipient's private key.
// A failed AEAD tag is indistinguishable from "no message" to callers
// above this package — it is a plain error, never a distinguished type.
func Open(env *wire.Envelope, recipientPriv []byte) ([]byte, error) {
	return nil, errUnimplemented
}

// GenerateEd25519 creates a signing keypair for optional sender authentication.
func GenerateEd25519() (pub, priv []byte, err error) {
	return nil, nil, errUnimplemented
}

// Sign returns an Ed25519 signature over msg.
func Sign(priv, msg []byte) ([]byte, error) {
	return nil, errUnimplemented
}

// Verify checks an Ed25519 signature.
func Verify(pub, msg, sig []byte) error {
	return errUnimplemented
}

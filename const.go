package mist

import "time"

// Version is the payload framing version written inside the ciphertext.
const Version byte = 1

// X25519 key sizes in bytes.
const (
	PublicKeySize  = 32
	PrivateKeySize = 32
)

// Ed25519 sizes for optional sender authentication.
const (
	SigningPublicKeySize  = 32
	SigningPrivateKeySize = 64
	SignatureSize         = 64
)

// ChaCha20-Poly1305 sizes in bytes.
const (
	NonceSize = 12
	TagSize   = 16
)

// EnvelopeOverhead is the number of bytes added by hybrid encryption
// before embedding: ephemeral public key + nonce + Poly1305 tag.
const EnvelopeOverhead = PublicKeySize + NonceSize + TagSize

// FrameDuration is the fixed stego-frame length. Embed loops the payload
// across consecutive frames of this duration; Listen uses the same value
// to recover frame phase. It is a protocol constant, not a per-call option.
const FrameDuration = 8 * time.Second

// Package wire defines the encrypted envelope and the inner payload framing.
//
// Inner payload (AEAD plaintext):
//
//	[ 1 byte  ] version
//	[ 1 byte  ] payload type
//	[ 4 bytes ] payload length (uint32, big-endian)
//	[ N bytes ] payload data
//	[ 64 bytes ] optional Ed25519 signature (only when sender auth is on)
//
// Outer envelope (what is embedded into coefficients):
//
//	[ 32 bytes ] ephemeral X25519 public key
//	[ 12 bytes ] ChaCha20-Poly1305 nonce
//	[ M bytes  ] ciphertext || 16-byte Poly1305 tag
package wire

import "encoding/binary"

const (
	VersionSize       = 1
	TypeSize          = 1
	LengthSize        = 4
	PayloadHeaderSize = VersionSize + TypeSize + LengthSize

	EphemeralPubSize = 32
	NonceSize        = 12
	TagSize          = 16
	EnvelopePrefix   = EphemeralPubSize + NonceSize
	EnvelopeOverhead = EnvelopePrefix + TagSize
)

// Payload is the inner framed plaintext, before AEAD.
type Payload struct {
	Version   byte
	Type      byte
	Data      []byte
	Signature []byte // nil when sender authentication is off
}

// Envelope is the hybrid-encryption box that gets embedded per stego frame.
type Envelope struct {
	EphemeralPub [EphemeralPubSize]byte
	Nonce        [NonceSize]byte
	Ciphertext   []byte // includes the 16-byte Poly1305 tag
}

// MarshalPayload encodes p in the inner framing layout.
func MarshalPayload(p Payload) ([]byte, error) {
	_ = p
	return nil, errUnimplemented
}

// UnmarshalPayload decodes the inner framing layout.
func UnmarshalPayload(b []byte) (Payload, error) {
	_ = b
	return Payload{}, errUnimplemented
}

// Marshal encodes the outer envelope as a single byte string for embedding.
func (e Envelope) Marshal() []byte {
	out := make([]byte, EnvelopePrefix+len(e.Ciphertext))
	copy(out[0:EphemeralPubSize], e.EphemeralPub[:])
	copy(out[EphemeralPubSize:EnvelopePrefix], e.Nonce[:])
	copy(out[EnvelopePrefix:], e.Ciphertext)
	return out
}

// UnmarshalEnvelope decodes an outer envelope. Ciphertext may be empty in
// the stub; the real implementation must reject inputs shorter than
// EnvelopeOverhead.
func UnmarshalEnvelope(b []byte) (Envelope, error) {
	_ = b
	return Envelope{}, errUnimplemented
}

// PutUint32 is the endianness used by the payload length field.
func PutUint32(b []byte, v uint32) {
	binary.BigEndian.PutUint32(b, v)
}

// Uint32 reads the payload length field.
func Uint32(b []byte) uint32 {
	return binary.BigEndian.Uint32(b)
}

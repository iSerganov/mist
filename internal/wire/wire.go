// Package wire is the inner payload framing and the outer hybrid-box envelope.
package wire

import (
	"encoding/binary"
	"fmt"
)

const (
	CurrentVersion    byte = 1
	VersionSize            = 1
	TypeSize               = 1
	LengthSize             = 4
	PayloadHeaderSize      = VersionSize + TypeSize + LengthSize
	SignatureSize          = 64

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
	Signature []byte
}

// Envelope is the hybrid-encryption box embedded in one stego frame.
type Envelope struct {
	EphemeralPub [EphemeralPubSize]byte
	Nonce        [NonceSize]byte
	Ciphertext   []byte
}

// MarshalPayload encodes version, type, big-endian length, data, and optional signature.
func MarshalPayload(p Payload) ([]byte, error) {
	if p.Version == 0 {
		p.Version = CurrentVersion
	}
	if p.Version != CurrentVersion {
		return nil, fmt.Errorf("%w: %d", ErrVersion, p.Version)
	}
	sigLen := len(p.Signature)
	if sigLen != 0 && sigLen != SignatureSize {
		return nil, fmt.Errorf("%w: signature", ErrLength)
	}
	out := make([]byte, PayloadHeaderSize+len(p.Data)+sigLen)
	out[0] = p.Version
	out[1] = p.Type
	binary.BigEndian.PutUint32(out[2:6], uint32(len(p.Data)))
	copy(out[PayloadHeaderSize:], p.Data)
	copy(out[PayloadHeaderSize+len(p.Data):], p.Signature)
	return out, nil
}

// UnmarshalPayload decodes the inner framing layout.
func UnmarshalPayload(b []byte) (Payload, error) {
	if len(b) < PayloadHeaderSize {
		return Payload{}, ErrShort
	}
	ver := b[0]
	if ver != CurrentVersion {
		return Payload{}, fmt.Errorf("%w: %d", ErrVersion, ver)
	}
	n := binary.BigEndian.Uint32(b[2:6])
	rest := b[PayloadHeaderSize:]
	if uint64(n) > uint64(len(rest)) {
		return Payload{}, ErrLength
	}
	data := rest[:n]
	sig := rest[n:]
	if len(sig) != 0 && len(sig) != SignatureSize {
		return Payload{}, ErrLength
	}
	p := Payload{
		Version: ver,
		Type:    b[1],
		Data:    append([]byte(nil), data...),
	}
	if len(sig) == SignatureSize {
		p.Signature = append([]byte(nil), sig...)
	}
	return p, nil
}

// Marshal encodes the outer envelope as a single byte string for embedding.
func (e Envelope) Marshal() []byte {
	out := make([]byte, EnvelopePrefix+len(e.Ciphertext))
	copy(out[0:EphemeralPubSize], e.EphemeralPub[:])
	copy(out[EphemeralPubSize:EnvelopePrefix], e.Nonce[:])
	copy(out[EnvelopePrefix:], e.Ciphertext)
	return out
}

// UnmarshalEnvelope decodes an outer envelope. b must be at least EnvelopeOverhead.
func UnmarshalEnvelope(b []byte) (Envelope, error) {
	if len(b) < EnvelopeOverhead {
		return Envelope{}, ErrShort
	}
	var env Envelope
	copy(env.EphemeralPub[:], b[:EphemeralPubSize])
	copy(env.Nonce[:], b[EphemeralPubSize:EnvelopePrefix])
	env.Ciphertext = append([]byte(nil), b[EnvelopePrefix:]...)
	return env, nil
}

// PutUint32 writes a big-endian uint32, the payload length encoding.
func PutUint32(b []byte, v uint32) {
	binary.BigEndian.PutUint32(b, v)
}

// Uint32 reads a big-endian uint32.
func Uint32(b []byte) uint32 {
	return binary.BigEndian.Uint32(b)
}

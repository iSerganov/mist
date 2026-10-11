package mist

import (
	"encoding/binary"
	"fmt"
	"math"
)

// PayloadType identifies the kind of data inside a Payload.
// The type byte is encrypted with the payload, so a warden who isolates
// the embedded bytes still cannot read it without the private key.
type PayloadType byte

// Payload types. The values are part of the wire format.
const (
	PayloadText  PayloadType = 0x01
	PayloadImage PayloadType = 0x02
	PayloadAudio PayloadType = 0x03
	PayloadFile  PayloadType = 0x04
)

// Payload is the unit that Emitter encrypts and Catcher emits.
// A PayloadFile carries its name in Data; build and read it with File and
// FileName. For every other type Data is the content itself.
type Payload struct {
	Type PayloadType
	Data []byte
}

// Text returns a text payload.
func Text(s string) Payload {
	return Payload{Type: PayloadText, Data: []byte(s)}
}

// Image returns an image payload. Data is the encoded image as it is on disk.
func Image(data []byte) Payload {
	return Payload{Type: PayloadImage, Data: data}
}

// Audio returns an audio payload. Data is the encoded audio as it is on disk.
func Audio(data []byte) Payload {
	return Payload{Type: PayloadAudio, Data: data}
}

// File returns a file payload that remembers name. The name travels inside
// Data as a big-endian u16 length and the name bytes, ahead of the content.
func File(name string, data []byte) (Payload, error) {
	if len(name) > math.MaxUint16 {
		return Payload{}, fmt.Errorf("%w: file name is %d bytes, limit %d",
			ErrInvalidPayload, len(name), math.MaxUint16)
	}
	buf := make([]byte, 2+len(name)+len(data))
	binary.BigEndian.PutUint16(buf, uint16(len(name)))
	copy(buf[2:], name)
	copy(buf[2+len(name):], data)
	return Payload{Type: PayloadFile, Data: buf}, nil
}

// FileName splits a PayloadFile into the name it was built with and its
// content. It returns ErrInvalidPayload for any other type or truncated Data.
func (p Payload) FileName() (name string, data []byte, err error) {
	if p.Type != PayloadFile {
		return "", nil, fmt.Errorf("%w: type %#02x is not a file", ErrInvalidPayload, byte(p.Type))
	}
	if len(p.Data) < 2 {
		return "", nil, fmt.Errorf("%w: file payload has no name length", ErrInvalidPayload)
	}
	n := int(binary.BigEndian.Uint16(p.Data))
	if len(p.Data) < 2+n {
		return "", nil, fmt.Errorf("%w: file name is cut short", ErrInvalidPayload)
	}
	return string(p.Data[2 : 2+n]), p.Data[2+n:], nil
}

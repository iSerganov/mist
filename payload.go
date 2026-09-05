package mist

// PayloadType identifies the kind of data inside a Payload.
// The type byte is encrypted with the payload, so a warden who isolates
// the embedded bytes still cannot read it without the private key.
type PayloadType byte

const (
	PayloadText  PayloadType = 0x01
	PayloadImage PayloadType = 0x02
	PayloadAudio PayloadType = 0x03
	PayloadFile  PayloadType = 0x04
)

// Payload is the unit that Emitter encrypts and Catcher emits.
// Phase 1 only uses PayloadText; the wire format is already type-agnostic.
type Payload struct {
	Type PayloadType
	Data []byte
}

// Text returns a text payload.
func Text(s string) Payload {
	return Payload{Type: PayloadText, Data: []byte(s)}
}

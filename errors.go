package mist

import "errors"

// Sentinel errors returned by the public API. Per-frame decrypt failures
// inside Listen are not surfaced — they are indistinguishable from frames
// that carry no payload and must stay silent.
var (
	ErrInvalidKey       = errors.New("mist: invalid key")
	ErrInvalidPayload   = errors.New("mist: invalid payload")
	ErrInvalidSource    = errors.New("mist: invalid source")
	ErrUnsupportedCodec = errors.New("mist: unsupported codec")
	ErrCarrier          = errors.New("mist: unreadable carrier")
	ErrClosed           = errors.New("mist: already closed")
	// ErrNoCapacity means no frame in the carrier had room for the
	// envelope, so nothing was embedded. Quiet or tonal audio yields too
	// few usable residues; a richer or longer carrier fixes it.
	ErrNoCapacity = errors.New("mist: carrier has no frame with room for the payload")
)

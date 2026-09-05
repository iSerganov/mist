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
)

// errUnimplemented is returned by stubs until the corresponding layer is filled in.
var errUnimplemented = errors.New("mist: unimplemented")

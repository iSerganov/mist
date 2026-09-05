package wire

import "errors"

var (
	ErrShort   = errors.New("wire: buffer too short")
	ErrLength  = errors.New("wire: length mismatch")
	ErrVersion = errors.New("wire: unsupported version")
)

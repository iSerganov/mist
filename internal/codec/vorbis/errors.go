package vorbis

import "errors"

var (
	ErrBadPacket     = errors.New("vorbis: bad packet")
	ErrBadSetup      = errors.New("vorbis: bad setup header")
	errUnimplemented = errors.New("vorbis: unimplemented")
)

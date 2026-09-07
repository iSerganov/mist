package vorbis

import "errors"

var (
	ErrBadPacket     = errors.New("vorbis: bad packet")
	ErrBadSetup      = errors.New("vorbis: bad setup header")
	errEOP           = errors.New("vorbis: end of packet")
	errUnimplemented = errors.New("vorbis: unimplemented")
)

package stego

import "errors"

var (
	ErrCapacity      = errors.New("stego: payload exceeds frame capacity")
	ErrNoResidues    = errors.New("stego: no embeddable residues")
	errUnimplemented = errors.New("stego: unimplemented")
)

package stego

// Selector picks which eligible coefficients are touched, scattered across
// the carrier by a keyed PRNG. The seed is the HKDF position subkey, never
// the AEAD key itself.
type Selector struct {
	key       []byte
	nEligible int
}

// NewSelector builds a position stream for nEligible coefficients.
func NewSelector(positionKey []byte, nEligible int) *Selector {
	return &Selector{key: positionKey, nEligible: nEligible}
}

// Pick returns n distinct eligible indexes in [0, nEligible).
// The real implementation must be deterministic for a given key so
// Embed and Listen walk the same sequence.
func (s *Selector) Pick(n int) []int {
	_ = n
	return nil
}

// Eligible returns the coefficient indexes that sit in embeddable
// high-frequency bands for the given residue layout.
func Eligible(residues []ResidueView, bands BandSet) []int {
	_ = residues
	_ = bands
	return nil
}

// ResidueView is the subset of a residue the selector needs.
type ResidueView struct {
	Index int
	Band  int
	Value int32
}

// BandSet is the set of frequency bands allowed for embedding.
type BandSet struct {
	FromHz int
	ToHz   int
}

// DefaultBands is a starting high-frequency window. It will be replaced
// once the perceptual-masking harness exists.
var DefaultBands = BandSet{FromHz: 8000, ToHz: 16000}

package stego

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/binary"
)

// Selector picks which eligible coefficients are touched, scattered across
// the carrier by a keyed PRNG. The seed is the HKDF position subkey, never
// the AEAD key itself. Embed and Listen must see the same sequence.
type Selector struct {
	key       []byte
	nEligible int
}

// NewSelector builds a position stream for nEligible coefficients.
func NewSelector(positionKey []byte, nEligible int) *Selector {
	return &Selector{
		key:       append([]byte(nil), positionKey...),
		nEligible: nEligible,
	}
}

// Pick returns n distinct eligible indexes in [0, nEligible).
// Deterministic Fisher–Yates using HMAC-SHA256(key, counter).
func (s *Selector) Pick(n int) []int {
	if s == nil || s.nEligible <= 0 || n <= 0 {
		return nil
	}
	if n > s.nEligible {
		n = s.nEligible
	}
	idx := make([]int, s.nEligible)
	for i := range idx {
		idx[i] = i
	}
	var ctr uint64
	for i := 0; i < n; i++ {
		j := i + s.bounded(ctr, s.nEligible-i)
		ctr++
		idx[i], idx[j] = idx[j], idx[i]
	}
	return idx[:n]
}

func (s *Selector) bounded(ctr uint64, n int) int {
	if n <= 1 {
		return 0
	}
	mac := hmac.New(sha256.New, s.key)
	var buf [8]byte
	binary.BigEndian.PutUint64(buf[:], ctr)
	_, _ = mac.Write(buf[:])
	sum := mac.Sum(nil)
	v := binary.BigEndian.Uint64(sum[:8])
	return int(v % uint64(n))
}

// Eligible returns the coefficient indexes that sit in embeddable
// high-frequency bands for the given residue layout. Residues the codec
// marked Unflippable are excluded — LSB matching them would need a
// differently-sized codeword and desync the bitstream.
func Eligible(residues []ResidueView, bands BandSet) []int {
	var out []int
	for i, r := range residues {
		if r.Unflippable {
			continue
		}
		if r.Band >= bands.FromHz && r.Band < bands.ToHz {
			out = append(out, i)
		}
	}
	return out
}

// ResidueView is the subset of a residue the selector needs.
type ResidueView struct {
	Index       int
	Band        int
	Value       int32
	Unflippable bool
}

// BandSet is the set of frequency bands allowed for embedding.
type BandSet struct {
	FromHz int
	ToHz   int
}

// DefaultBands keeps embedding above 6 kHz, where the ear is least
// sensitive to the error a flipped residue introduces. Perturbing the
// whole spectrum instead costs about 4 dB of signal-to-distortion on real
// music; confining it here makes the embedding all but free. Roughly half
// the flippable residues sit above this bound at normal bitrates, so the
// capacity given up is modest.
var DefaultBands = BandSet{FromHz: 6000, ToHz: 48000}

package stego

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/binary"

	"golang.org/x/crypto/chacha20"
)

// keyStream is a deterministic stream of uint64s keyed by the position
// subkey. label separates the streams drawn from one key, so the
// permutation and the trellis code's matrix never share output.
type keyStream struct {
	c   *chacha20.Cipher
	buf [1024]byte
	off int
}

func newKeyStream(key []byte, label string) *keyStream {
	mac := hmac.New(sha256.New, key)
	_, _ = mac.Write([]byte(label))
	// A 32-byte key and a 12-byte nonce are always valid, so this cannot fail.
	c, _ := chacha20.NewUnauthenticatedCipher(mac.Sum(nil), make([]byte, chacha20.NonceSize))
	k := &keyStream{c: c}
	k.off = len(k.buf)
	return k
}

func (k *keyStream) next() uint64 {
	if k.off == len(k.buf) {
		clear(k.buf[:])
		k.c.XORKeyStream(k.buf[:], k.buf[:])
		k.off = 0
	}
	v := binary.LittleEndian.Uint64(k.buf[k.off:])
	k.off += 8
	return v
}

// below returns a value in [0, n).
func (k *keyStream) below(n int) int {
	if n <= 1 {
		return 0
	}
	return int(k.next() % uint64(n))
}

// Selector orders the eligible coefficients by a keyed shuffle, so a
// position's place in the code is scattered across the carrier. The seed
// is the HKDF position subkey, never the AEAD key itself. Embed and Listen
// must see the same order.
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

// Pick returns the first n indexes of a keyed permutation of
// [0, nEligible): n distinct values.
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
	ks := newKeyStream(s.key, "mist-perm-v2")
	for i := 0; i < n; i++ {
		j := i + ks.below(s.nEligible-i)
		idx[i], idx[j] = idx[j], idx[i]
	}
	return idx[:n]
}

// Eligible returns the coefficient indexes that sit in embeddable
// high-frequency bands for the given residue layout. Residues the codec
// marked Unflippable are excluded — flipping them would need a
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

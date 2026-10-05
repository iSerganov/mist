package steganalysis

import (
	"encoding/binary"
	"math"
)

// SelectionName is the public-key selection-channel warden. It sees
// positions the recipient key implies and nothing the sender keeps private.
const SelectionName = "selection"

// FrozenPrimaries is the Holm family embedder changes are scored against.
// It is frozen before that tuning. Chi-square, SPA and RS stay exploratory.
func FrozenPrimaries() []string {
	return []string{"hcf-com", "classifier", "markov", "key-aware", RichName, SelectionName}
}

// ParityGap is how far the LSB rate at the listed positions sits from the
// LSB rate everywhere else. A warden who can derive the positions — from
// the recipient public key and the frame index — computes this without
// the plaintext. Sender-only costs are not an input.
func ParityGap(v []int32, at []int) float64 {
	if len(v) == 0 || len(at) == 0 {
		return 0
	}
	mark := make([]bool, len(v))
	for _, i := range at {
		if i >= 0 && i < len(v) {
			mark[i] = true
		}
	}
	var sel, other float64
	var ns, no int
	for i, x := range v {
		bit := float64(x & 1)
		if mark[i] {
			sel += bit
			ns++
			continue
		}
		other += bit
		no++
	}
	if ns == 0 || no == 0 {
		return 0
	}
	return math.Abs(sel/float64(ns) - other/float64(no))
}

// ChangedFraction is the share of positions that differ. It is an oracle:
// it needs the cover, which a warden who does not hold the carrier does
// not have. It is not an operational score.
func ChangedFraction(cover, stego []int32) float64 {
	n := min(len(cover), len(stego))
	if n == 0 {
		return 0
	}
	var c int
	for i := range n {
		if cover[i] != stego[i] {
			c++
		}
	}
	return float64(c) / float64(n)
}

// PlausibleLength scores an unmasked big-endian length that could be a
// payload. A masked length should not look like one. This is the positive
// control for leaving the length in the clear, not a detector Mist's own
// masked field is expected to fail.
func PlausibleLength(b []byte) float64 {
	if len(b) < 4 {
		return 0
	}
	n := binary.BigEndian.Uint32(b[:4])
	if n > 0 && n < 1<<20 {
		return 1
	}
	return 0
}

// DeadTail scores a stream whose tail is constant while its head is not,
// which is what embedding without filler leaves behind.
func DeadTail(v []int32) float64 {
	if len(v) < 8 {
		return 0
	}
	head, tail := v[:len(v)/4], v[len(v)*3/4:]
	if constant(tail) && !constant(head) {
		return 1
	}
	return 0
}

func constant(v []int32) bool {
	if len(v) == 0 {
		return true
	}
	for _, x := range v[1:] {
		if x != v[0] {
			return false
		}
	}
	return true
}

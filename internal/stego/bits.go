package stego

import "slices"

// carrier is indexed access to the values one stego frame can embed in:
// residue VQ entries for a lossy codec, PCM samples for a lossless one.
// Everything above this interface — the constant density, the keyed
// order, the trellis code, the filler — is shared by both, so the two
// paths cannot drift apart.
//
// Len and At must read the same before and after embedding: the receiver
// sees only the result. Cost and Flip are the sender's alone, so they may
// use anything the sender knows.
type carrier interface {
	Len() int
	At(i int) int32
	// Cost is the damage flipping value i's LSB would do, relative to the
	// other values in the frame.
	Cost(i int) float32
	// Flip moves value i by ±1, which flips its LSB.
	Flip(i int)
}

// slots is the bit budget for n eligible carriers at the constant density.
func slots(n int) int { return int(float64(n) * Density) }

// Covered is how many values of a frame of n eligible carriers the constant
// density touches. A warden who knows the recipient public key uses the
// same count the embedder does.
func Covered(n int) int { return slots(n) }

// layout is how a frame of n eligible values carries its slots(n) bits:
// which values form the cover, in keyed order, and the code over them.
func layout(n int, posKey []byte) (cover []int, code stcCode, m int) {
	m = slots(n)
	if m < 1 {
		return nil, stcCode{}, 0
	}
	w := n / m
	return NewSelector(posKey, n).Pick(m * w), newSTC(posKey, w), m
}

// place writes bits through the trellis code and pads the rest of the
// budget with CSPRNG filler, so a frame carrying a payload and a frame
// carrying nothing perturb the same kind of value at the same rate.
func place(c carrier, posKey []byte, bits Bits) error {
	cover, code, m := layout(c.Len(), posKey)
	if m < 1 {
		return ErrNoResidues
	}
	payload := make([]byte, (m+7)/8)
	if len(bits) > len(payload) {
		return ErrCapacity
	}
	copy(payload, bits)
	pad, err := Filler(len(payload) - len(bits))
	if err != nil {
		return err
	}
	copy(payload[len(bits):], pad)

	msg := make([]uint8, m)
	for i := range msg {
		msg[i] = payload[i/8] >> uint(i%8) & 1
	}
	x := make([]uint8, len(cover))
	cost := make([]float32, len(cover))
	for j, p := range cover {
		x[j], cost[j] = LSB(c.At(p)), c.Cost(p)
	}
	flips, err := code.embed(x, cost, msg)
	if err != nil {
		return err
	}
	// Flipped in cover order, which is keyed and so scattered across the
	// frame: a carrier that balances its flips sees them in no pattern.
	slices.Sort(flips)
	for _, j := range flips {
		c.Flip(cover[j])
	}
	return nil
}

// lift reads the constant-density bit string back out of c.
func lift(c carrier, posKey []byte) (Bits, error) {
	cover, code, m := layout(c.Len(), posKey)
	if m < 1 {
		return nil, ErrNoResidues
	}
	y := make([]uint8, len(cover))
	for j, p := range cover {
		y[j] = LSB(c.At(p))
	}
	out := make([]byte, (m+7)/8)
	for i, bit := range code.syndrome(y, m) {
		out[i/8] |= bit << uint(i%8)
	}
	return out, nil
}

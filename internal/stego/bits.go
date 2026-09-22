package stego

// carrier is indexed access to the values one stego frame can embed in:
// residue VQ entries for a lossy codec, PCM samples for a lossless one.
// Everything above this interface — the constant density, the keyed
// positions, LSB matching, the filler — is shared by both, so the two
// paths cannot drift apart.
type carrier interface {
	Len() int
	At(i int) int32
	Set(i int, v int32)
}

// slots is the bit budget for n eligible carriers at the constant density.
func slots(n int) int { return int(float64(n) * Density) }

// place writes bits at keyed positions and pads the rest of the budget
// with CSPRNG filler, so a frame carrying a payload and a frame carrying
// nothing perturb the same fraction of the same kind of value.
func place(c carrier, posKey []byte, bits Bits) error {
	nbits := slots(c.Len())
	if nbits < 1 {
		return ErrNoResidues
	}
	payload := make([]byte, (nbits+7)/8)
	if len(bits) > len(payload) {
		return ErrCapacity
	}
	copy(payload, bits)
	pad, err := Filler(len(payload) - len(bits))
	if err != nil {
		return err
	}
	copy(payload[len(bits):], pad)
	for i, p := range NewSelector(posKey, c.Len()).Pick(nbits) {
		c.Set(p, Match(c.At(p), (payload[i/8]>>uint(i%8))&1))
	}
	return nil
}

// lift reads the constant-density bit string back out of c.
func lift(c carrier, posKey []byte) (Bits, error) {
	nbits := slots(c.Len())
	if nbits < 1 {
		return nil, ErrNoResidues
	}
	out := make([]byte, (nbits+7)/8)
	for i, p := range NewSelector(posKey, c.Len()).Pick(nbits) {
		if LSB(c.At(p)) == 1 {
			out[i/8] |= 1 << uint(i%8)
		}
	}
	return out, nil
}

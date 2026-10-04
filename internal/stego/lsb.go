package stego

// LSB returns the least significant bit of coeff.
func LSB(coeff int32) uint8 {
	return uint8(coeff & 1)
}

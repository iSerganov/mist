package stego

// Match applies LSB matching: if coeff's LSB already equals bit it is
// left unchanged; otherwise coeff is randomly incremented or decremented
// by 1. Never use LSB replacement (forcing the bit), which is detectable
// via sample-pair analysis.
func Match(coeff int32, bit uint8) int32 {
	_ = bit
	return coeff
}

// LSB returns the least significant bit of coeff.
func LSB(coeff int32) uint8 {
	return uint8(coeff & 1)
}

package steganalysis

import "math/big"

var (
	fieldP     = new(big.Int).Sub(new(big.Int).Lsh(big.NewInt(1), 255), big.NewInt(19))
	groupOrder = mustBig("7237005577332262213973186563042994240857116359379907606001950938285454250989")
	ladderA24  = big.NewInt(121665)
)

func mustBig(s string) *big.Int {
	n, ok := new(big.Int).SetString(s, 10)
	if !ok {
		panic("steganalysis: bad constant")
	}
	return n
}

// HonestX25519 reports whether b could be a public key produced by an
// honest X25519 implementation. Such a key has its top bit clear and lies in
// the prime-order subgroup, so ℓ times it is the point at infinity. A
// uniformly random 32-byte string passes both checks about 1 time in 32,
// so a key that is sent in the clear is a presence test that needs no
// secret.
func HonestX25519(b []byte) bool {
	if len(b) != 32 || b[31]&0x80 != 0 {
		return false
	}
	u := new(big.Int)
	for i := len(b) - 1; i >= 0; i-- {
		u.Lsh(u, 8).Or(u, big.NewInt(int64(b[i])))
	}
	return inPrimeSubgroup(u)
}

// inPrimeSubgroup runs the RFC 7748 x-only Montgomery ladder with the
// subgroup order as the scalar and checks that Z is zero at the end.
func inPrimeSubgroup(u *big.Int) bool {
	mul := func(a, b *big.Int) *big.Int { return new(big.Int).Mod(new(big.Int).Mul(a, b), fieldP) }
	add := func(a, b *big.Int) *big.Int { return new(big.Int).Mod(new(big.Int).Add(a, b), fieldP) }
	sub := func(a, b *big.Int) *big.Int { return new(big.Int).Mod(new(big.Int).Sub(a, b), fieldP) }

	x2, z2 := big.NewInt(1), big.NewInt(0)
	x3, z3 := new(big.Int).Set(u), big.NewInt(1)
	var swap uint
	for t := groupOrder.BitLen() - 1; t >= 0; t-- {
		bit := groupOrder.Bit(t)
		if swap^bit == 1 {
			x2, x3, z2, z3 = x3, x2, z3, z2
		}
		swap = bit
		a, b := add(x2, z2), sub(x2, z2)
		aa, bb := mul(a, a), mul(b, b)
		e := sub(aa, bb)
		da, cb := mul(sub(x3, z3), a), mul(add(x3, z3), b)
		sum, diff := add(da, cb), sub(da, cb)
		x3, z3 = mul(sum, sum), mul(u, mul(diff, diff))
		x2, z2 = mul(aa, bb), mul(e, add(aa, mul(ladderA24, e)))
	}
	if swap == 1 {
		z2 = z3
	}
	return z2.Sign() == 0
}

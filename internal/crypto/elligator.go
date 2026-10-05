package crypto

import (
	"crypto/rand"
	"fmt"
	"math/big"
)

// An X25519 public key sent in the clear is not 32 random bytes: its top bit
// is always clear and it always lies in the prime-order subgroup, so anyone
// who can read the envelope can test for it. Elligator 2 (Bernstein,
// Hamburg, Krasnova, Lange, 2013) sends a representative of the key that is
// uniformly distributed instead. X25519 uses only the u-coordinate, so the
// receiver recovers the same key from the representative.

var (
	fieldP = new(big.Int).Sub(new(big.Int).Lsh(big.NewInt(1), 255), big.NewInt(19))
	curveA = big.NewInt(486662)
	bigOne = big.NewInt(1)
	// sqrtM1 is a square root of -1, 2^((p-1)/4).
	sqrtM1 = new(big.Int).Exp(big.NewInt(2), new(big.Int).Rsh(new(big.Int).Sub(fieldP, bigOne), 2), fieldP)
)

func fmod(x *big.Int) *big.Int { return x.Mod(x, fieldP) }

func fmul(a, b *big.Int) *big.Int { return fmod(new(big.Int).Mul(a, b)) }

func finv(a *big.Int) *big.Int { return new(big.Int).ModInverse(a, fieldP) }

// legendre is 1 for a non-zero square, -1 for a non-square and 0 for zero.
func legendre(a *big.Int) int {
	l := new(big.Int).Exp(a, new(big.Int).Rsh(new(big.Int).Sub(fieldP, bigOne), 1), fieldP)
	switch {
	case l.Sign() == 0:
		return 0
	case l.Cmp(bigOne) == 0:
		return 1
	}
	return -1
}

// fsqrt returns a square root of a, which must be a square. p is 5 mod 8.
func fsqrt(a *big.Int) *big.Int {
	r := new(big.Int).Exp(a, new(big.Int).Rsh(new(big.Int).Add(fieldP, big.NewInt(3)), 3), fieldP)
	if fmul(r, r).Cmp(a) != 0 {
		r = fmul(r, sqrtM1)
	}
	return r
}

func fromLE(b []byte) *big.Int {
	be := make([]byte, len(b))
	for i := range b {
		be[len(b)-1-i] = b[i]
	}
	return new(big.Int).SetBytes(be)
}

func toLE(n *big.Int, size int) []byte {
	out := make([]byte, size)
	be := n.Bytes()
	for i := range be {
		out[len(be)-1-i] = be[i]
	}
	return out
}

// representative returns the Elligator 2 representative of the X25519
// public key pub, or false if the key has none, which is about half of all
// keys. The two top bits of the 256-bit result are random padding, since a
// representative has 254 bits.
func representative(pub []byte) ([]byte, bool, error) {
	u := fromLE(pub)
	d := fmod(new(big.Int).Add(u, curveA))
	if u.Sign() == 0 || d.Sign() == 0 {
		return nil, false, nil
	}
	if legendre(fmod(new(big.Int).Neg(fmul(big.NewInt(2), fmul(u, d))))) != 1 {
		return nil, false, nil
	}
	// r² = -u / (2(u+A)); either root maps back to u.
	r2 := fmul(fmod(new(big.Int).Neg(u)), finv(fmul(big.NewInt(2), d)))
	r := fsqrt(r2)
	if half := new(big.Int).Rsh(fieldP, 1); r.Cmp(half) > 0 {
		r = new(big.Int).Sub(fieldP, r)
	}
	out := toLE(r, 32)
	var pad [1]byte
	if _, err := rand.Read(pad[:]); err != nil {
		return nil, false, fmt.Errorf("crypto: representative padding: %w", err)
	}
	out[31] |= pad[0] & 0xc0
	return out, true, nil
}

// publicFromRepresentative is the Elligator 2 direct map: it turns a
// representative back into the u-coordinate of the public key.
func publicFromRepresentative(rep []byte) []byte {
	masked := append([]byte(nil), rep...)
	masked[31] &= 0x3f
	r := fromLE(masked)
	w := fmul(fmod(new(big.Int).Neg(curveA)), finv(fmod(new(big.Int).Add(bigOne, fmul(big.NewInt(2), fmul(r, r))))))
	f := fmod(new(big.Int).Add(fmul(w, fmul(w, w)), fmul(curveA, fmul(w, w))))
	f = fmod(f.Add(f, w))
	u := w
	if legendre(f) == -1 {
		u = fmod(new(big.Int).Neg(new(big.Int).Add(w, curveA)))
	}
	return toLE(u, 32)
}

// generateEphemeral draws X25519 keypairs until one has a representative,
// which takes two tries on average.
func generateEphemeral() (rep, priv []byte, err error) {
	for {
		pub, priv, err := GenerateX25519()
		if err != nil {
			return nil, nil, err
		}
		rep, ok, err := representative(pub)
		if err != nil {
			return nil, nil, err
		}
		if ok {
			return rep, priv, nil
		}
	}
}

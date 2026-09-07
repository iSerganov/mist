package vorbis

import "math"

type codebook struct {
	dim, entries int
	lens         []uint8
	codes        []uint32
	used         []int
	lookup       int
	min, delta   float64
	seq          bool
	quantvals    []uint32
	quantN       int
	tree         *hNode
}

type hNode struct {
	entry    int
	child    [2]*hNode
	internal bool
}

func unpackCodebook(b *bits) (*codebook, error) {
	sync, err := b.read(24)
	if err != nil || sync != 0x564342 {
		return nil, ErrBadSetup
	}
	dim, err := b.read(16)
	if err != nil {
		return nil, err
	}
	entries, err := b.read(24)
	if err != nil {
		return nil, err
	}
	cb := &codebook{dim: int(dim), entries: int(entries), lens: make([]uint8, entries)}
	ordered, err := b.read1()
	if err != nil {
		return nil, err
	}
	if ordered == 1 {
		cur, err := b.read(5)
		if err != nil {
			return nil, err
		}
		cur++
		i := 0
		for i < cb.entries {
			nbits := ilog(cb.entries - i)
			cnt, err := b.read(nbits)
			if err != nil {
				return nil, err
			}
			for j := 0; j < int(cnt) && i < cb.entries; j++ {
				cb.lens[i] = uint8(cur)
				i++
			}
			cur++
		}
	} else {
		sparse, err := b.read1()
		if err != nil {
			return nil, err
		}
		for i := 0; i < cb.entries; i++ {
			if sparse == 1 {
				used, err := b.read1()
				if err != nil {
					return nil, err
				}
				if used == 0 {
					continue
				}
			}
			l, err := b.read(5)
			if err != nil {
				return nil, err
			}
			cb.lens[i] = uint8(l + 1)
		}
	}
	if err := cb.buildHuffman(); err != nil {
		return nil, err
	}
	lt, err := b.read(4)
	if err != nil {
		return nil, err
	}
	cb.lookup = int(lt)
	if cb.lookup == 0 {
		return cb, nil
	}
	if cb.lookup > 2 {
		return nil, ErrBadSetup
	}
	minb, err := b.read(32)
	if err != nil {
		return nil, err
	}
	delb, err := b.read(32)
	if err != nil {
		return nil, err
	}
	cb.min = float32Unpack(minb)
	cb.delta = float32Unpack(delb)
	vbits, err := b.read(4)
	if err != nil {
		return nil, err
	}
	vbits++
	seq, err := b.read1()
	if err != nil {
		return nil, err
	}
	cb.seq = seq == 1
	var nval int
	if cb.lookup == 1 {
		nval = lookup1Values(cb.entries, cb.dim)
	} else {
		nval = cb.entries * cb.dim
	}
	cb.quantN = nval
	cb.quantvals = make([]uint32, nval)
	for i := 0; i < nval; i++ {
		v, err := b.read(int(vbits))
		if err != nil {
			return nil, err
		}
		cb.quantvals[i] = v
	}
	return cb, nil
}

func (cb *codebook) buildHuffman() error {
	cb.codes = make([]uint32, cb.entries)
	cb.used = nil
	marker := make([]uint32, 33)
	raw := make([]uint32, cb.entries)
	for i := 0; i < cb.entries; i++ {
		length := int(cb.lens[i])
		if length == 0 {
			continue
		}
		if length > 32 {
			return ErrBadSetup
		}
		entry := marker[length]
		if length < 32 && entry>>uint(length) != 0 {
			return ErrBadSetup
		}
		raw[i] = entry
		for j := length; j > 0; j-- {
			if marker[j]&1 != 0 {
				marker[j] = marker[j-1] << 1
				break
			}
			marker[j]++
		}
		for j := length + 1; j < 33; j++ {
			if marker[j]>>1 == entry {
				entry = marker[j]
				marker[j] = marker[j-1] << 1
			} else {
				break
			}
		}
	}
	cb.tree = &hNode{entry: -1, internal: true}
	for i := 0; i < cb.entries; i++ {
		if cb.lens[i] == 0 {
			continue
		}
		cb.codes[i] = bitReverse(raw[i], int(cb.lens[i]))
		cb.used = append(cb.used, i)
		if err := cb.tree.insert(cb.codes[i], int(cb.lens[i]), i); err != nil {
			return err
		}
	}
	return nil
}

func (n *hNode) insert(code uint32, length, entry int) error {
	cur := n
	for i := 0; i < length; i++ {
		bit := (code >> uint(i)) & 1
		if cur.child[bit] == nil {
			cur.child[bit] = &hNode{entry: -1, internal: true}
		}
		cur = cur.child[bit]
	}
	if !cur.internal && cur.entry >= 0 {
		return ErrBadSetup
	}
	cur.internal = false
	cur.entry = entry
	return nil
}

func (cb *codebook) decode(b *bits) (int, error) {
	if cb.tree == nil {
		return 0, ErrBadSetup
	}
	n := cb.tree
	for n.internal {
		bit, err := b.read1()
		if err != nil {
			return 0, err
		}
		n = n.child[bit]
		if n == nil {
			return 0, ErrBadPacket
		}
	}
	return n.entry, nil
}

func (cb *codebook) encode(w *writer, entry int) error {
	if entry < 0 || entry >= cb.entries || cb.lens[entry] == 0 {
		return ErrBadPacket
	}
	w.write(int(cb.lens[entry]), cb.codes[entry])
	return nil
}

func (cb *codebook) vq(entry int) []float64 {
	out := make([]float64, cb.dim)
	if cb.lookup == 0 || entry < 0 || entry >= cb.entries {
		return out
	}
	last := 0.0
	if cb.lookup == 1 {
		index := entry
		q := cb.quantN
		if q <= 0 {
			return out
		}
		for i := 0; i < cb.dim; i++ {
			off := index % q
			index /= q
			v := float64(cb.quantvals[off])*cb.delta + cb.min + last
			if cb.seq {
				last = v
			}
			out[i] = v
		}
		return out
	}
	base := entry * cb.dim
	for i := 0; i < cb.dim && base+i < len(cb.quantvals); i++ {
		v := float64(cb.quantvals[base+i])*cb.delta + cb.min + last
		if cb.seq {
			last = v
		}
		out[i] = v
	}
	return out
}

func (cb *codebook) nearest(vec []float64) int {
	best, bestD := 0, math.Inf(1)
	for _, e := range cb.used {
		v := cb.vq(e)
		var d float64
		for i := 0; i < cb.dim && i < len(vec); i++ {
			x := vec[i] - v[i]
			d += x * x
		}
		if d < bestD {
			bestD, best = d, e
		}
	}
	return best
}

func bitReverse(v uint32, n int) uint32 {
	var r uint32
	for i := 0; i < n; i++ {
		r = (r << 1) | (v & 1)
		v >>= 1
	}
	return r
}

func float32Unpack(x uint32) float64 {
	mant := float64(x & 0x1fffff)
	if x&0x80000000 != 0 {
		mant = -mant
	}
	exp := int((x & 0x7fe00000) >> 21)
	return math.Ldexp(mant, exp-788)
}

func lookup1Values(entries, dim int) int {
	if dim < 1 || entries < 1 {
		return 0
	}
	vals := int(math.Floor(math.Pow(float64(entries), 1/float64(dim))))
	for {
		acc, acc1 := 1, 1
		for i := 0; i < dim; i++ {
			acc *= vals
			acc1 *= vals + 1
		}
		if acc <= entries && acc1 > entries {
			return vals
		}
		if acc > entries {
			vals--
		} else {
			vals++
		}
		if vals < 0 {
			return 0
		}
	}
}

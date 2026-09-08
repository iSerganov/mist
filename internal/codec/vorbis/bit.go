package vorbis

// bits is a Vorbis LSB-first bit packer. Bit 0 of each byte is read first.
type bits struct {
	d   []byte
	pos int
}

func newBits(d []byte) *bits { return &bits{d: d} }

func (b *bits) remaining() int { return len(b.d)*8 - b.pos }

func (b *bits) read(n int) (uint32, error) {
	if n <= 0 {
		return 0, nil
	}
	if b.remaining() < n {
		return 0, errEOP
	}
	var v uint32
	for i := 0; i < n; i++ {
		by := b.pos / 8
		bi := uint(b.pos % 8)
		v |= uint32((b.d[by]>>bi)&1) << uint(i)
		b.pos++
	}
	return v, nil
}

func (b *bits) read1() (uint32, error) { return b.read(1) }

type writer struct {
	d   []byte
	pos int
}

func newWriter() *writer { return &writer{} }

func (w *writer) write(n int, v uint32) {
	for i := 0; i < n; i++ {
		if w.pos/8 >= len(w.d) {
			w.d = append(w.d, 0)
		}
		if (v>>uint(i))&1 == 1 {
			w.d[w.pos/8] |= 1 << uint(w.pos%8)
		}
		w.pos++
	}
}

func (w *writer) writeBits(src []byte, nbits int) {
	w.writeBitsFrom(src, 0, nbits)
}

// writeBitsFrom copies nbits from src starting at bit offset off.
func (w *writer) writeBitsFrom(src []byte, off, nbits int) {
	for i := 0; i < nbits; i++ {
		by := (off + i) / 8
		bi := uint((off + i) % 8)
		bit := uint32(0)
		if by < len(src) {
			bit = uint32((src[by] >> bi) & 1)
		}
		w.write(1, bit)
	}
}

func (w *writer) bytes() []byte {
	n := (w.pos + 7) / 8
	out := make([]byte, n)
	copy(out, w.d)
	return out
}

func ilog(x int) int {
	n := 0
	for x > 0 {
		n++
		x >>= 1
	}
	return n
}

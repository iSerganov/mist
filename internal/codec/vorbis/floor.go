package vorbis

type floor struct {
	typ        int
	partitions int
	pclass     []int
	classDim   []int
	classSubs  []int
	classBook  []int
	subBook    [][]int
	mult       int
	xs         []int
}

func unpackFloor(b *bits, typ, nbooks int) (floor, error) {
	var f floor
	f.typ = typ
	if typ == 0 {
		return f, unpackFloor0(b)
	}
	if typ != 1 {
		return f, ErrBadSetup
	}
	np, err := b.read(5)
	if err != nil {
		return f, err
	}
	f.partitions = int(np)
	maxClass := -1
	f.pclass = make([]int, f.partitions)
	for i := 0; i < f.partitions; i++ {
		c, err := b.read(4)
		if err != nil {
			return f, err
		}
		f.pclass[i] = int(c)
		if int(c) > maxClass {
			maxClass = int(c)
		}
	}
	f.classDim = make([]int, maxClass+1)
	f.classSubs = make([]int, maxClass+1)
	f.classBook = make([]int, maxClass+1)
	f.subBook = make([][]int, maxClass+1)
	for i := 0; i <= maxClass; i++ {
		d, err := b.read(3)
		if err != nil {
			return f, err
		}
		f.classDim[i] = int(d) + 1
		s, err := b.read(2)
		if err != nil {
			return f, err
		}
		f.classSubs[i] = int(s)
		if s != 0 {
			bk, err := b.read(8)
			if err != nil {
				return f, err
			}
			if int(bk) >= nbooks {
				return f, ErrBadSetup
			}
			f.classBook[i] = int(bk)
		}
		nsub := 1 << uint(s)
		f.subBook[i] = make([]int, nsub)
		for j := 0; j < nsub; j++ {
			sb, err := b.read(8)
			if err != nil {
				return f, err
			}
			f.subBook[i][j] = int(sb) - 1
		}
	}
	mult, err := b.read(2)
	if err != nil {
		return f, err
	}
	f.mult = int(mult) + 1
	rbits, err := b.read(4)
	if err != nil {
		return f, err
	}
	f.xs = []int{0, 1 << rbits}
	for i := 0; i < f.partitions; i++ {
		c := f.pclass[i]
		for j := 0; j < f.classDim[c]; j++ {
			x, err := b.read(int(rbits))
			if err != nil {
				return f, err
			}
			f.xs = append(f.xs, int(x))
		}
	}
	if len(f.xs) > 65 {
		return f, ErrBadSetup
	}
	return f, nil
}

func unpackFloor0(b *bits) error {
	if _, err := b.read(8); err != nil { // order
		return err
	}
	if _, err := b.read(16); err != nil { // rate
		return err
	}
	if _, err := b.read(16); err != nil { // bark_map_size
		return err
	}
	if _, err := b.read(6); err != nil { // ampbits
		return err
	}
	if _, err := b.read(8); err != nil { // ampdB
		return err
	}
	n, err := b.read(4)
	if err != nil {
		return err
	}
	for i := 0; i < int(n)+1; i++ {
		if _, err := b.read(8); err != nil {
			return err
		}
	}
	return nil
}

// readFloor consumes one channel's floor. used is false when the
// "nonzero" bit is clear (no residue for this channel).
func readFloor(b *bits, f floor, books []*codebook) (used bool, err error) {
	nz, err := b.read1()
	if err != nil {
		return false, err
	}
	if nz == 0 {
		return false, nil
	}
	if f.typ == 0 {
		return false, ErrBadSetup
	}
	ranges := []int{256, 128, 86, 64}
	if f.mult < 1 || f.mult > 4 {
		return false, ErrBadSetup
	}
	rng := ranges[f.mult-1]
	bits := ilog(rng - 1)
	if _, err := b.read(bits); err != nil {
		return false, err
	}
	if _, err := b.read(bits); err != nil {
		return false, err
	}
	for i := 0; i < f.partitions; i++ {
		class := f.pclass[i]
		cdim := f.classDim[class]
		cbits := f.classSubs[class]
		csub := (1 << uint(cbits)) - 1
		cval := 0
		if cbits > 0 {
			e, err := books[f.classBook[class]].decode(b)
			if err != nil {
				return false, err
			}
			cval = e
		}
		for j := 0; j < cdim; j++ {
			book := f.subBook[class][cval&csub]
			cval >>= uint(cbits)
			if book >= 0 {
				if _, err := books[book].decode(b); err != nil {
					return false, err
				}
			}
		}
	}
	return true, nil
}

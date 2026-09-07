package vorbis

import "fmt"

type tables struct {
	books    []*codebook
	floors   []floor
	residues []residueCfg
	maps     []mapping
	modes    []mode
}

type mode struct {
	blockflag bool
	mapping   int
}

type mapping struct {
	submaps  int
	coupling [][2]int
	mux      []int
	subfloor []int
	subres   []int
}

func parseTables(setup []byte, channels int) (*tables, error) {
	if err := checkHeader(setup, headerSetup); err != nil {
		return nil, err
	}
	if channels < 1 {
		return nil, ErrBadSetup
	}
	b := newBits(setup)
	b.pos = 8 * (1 + len(magic))
	nbook, err := b.read(8)
	if err != nil {
		return nil, fmt.Errorf("%w: codebook count", err)
	}
	t := &tables{}
	for i := 0; i < int(nbook)+1; i++ {
		cb, err := unpackCodebook(b)
		if err != nil {
			return nil, fmt.Errorf("%w: codebook %d", err, i)
		}
		t.books = append(t.books, cb)
	}
	ntimes, err := b.read(6)
	if err != nil {
		return nil, fmt.Errorf("%w: times", err)
	}
	for i := 0; i < int(ntimes)+1; i++ {
		v, err := b.read(16)
		if err != nil || v != 0 {
			return nil, fmt.Errorf("%w: time %d", ErrBadSetup, i)
		}
	}
	nfloor, err := b.read(6)
	if err != nil {
		return nil, fmt.Errorf("%w: floor count", err)
	}
	for i := 0; i < int(nfloor)+1; i++ {
		ft, err := b.read(16)
		if err != nil {
			return nil, fmt.Errorf("%w: floor type %d", err, i)
		}
		fl, err := unpackFloor(b, int(ft), len(t.books))
		if err != nil {
			return nil, fmt.Errorf("%w: floor %d type %d", err, i, ft)
		}
		t.floors = append(t.floors, fl)
	}
	nres, err := b.read(6)
	if err != nil {
		return nil, fmt.Errorf("%w: residue count", err)
	}
	for i := 0; i < int(nres)+1; i++ {
		rt, err := b.read(16)
		if err != nil {
			return nil, fmt.Errorf("%w: residue type %d", err, i)
		}
		rc, err := unpackResidue(b, int(rt), len(t.books))
		if err != nil {
			return nil, fmt.Errorf("%w: residue %d type %d", err, i, rt)
		}
		t.residues = append(t.residues, rc)
	}
	nmap, err := b.read(6)
	if err != nil {
		return nil, fmt.Errorf("%w: mapping count", err)
	}
	for i := 0; i < int(nmap)+1; i++ {
		mt, err := b.read(16)
		if err != nil || mt != 0 {
			return nil, fmt.Errorf("%w: mapping type %d", ErrBadSetup, mt)
		}
		m, err := unpackMapping(b, channels, len(t.floors), len(t.residues))
		if err != nil {
			return nil, fmt.Errorf("%w: mapping %d", err, i)
		}
		t.maps = append(t.maps, m)
	}
	nmode, err := b.read(6)
	if err != nil {
		return nil, fmt.Errorf("%w: mode count", err)
	}
	for i := 0; i < int(nmode)+1; i++ {
		bf, err := b.read1()
		if err != nil {
			return nil, err
		}
		wt, err := b.read(16)
		if err != nil {
			return nil, err
		}
		tt, err := b.read(16)
		if err != nil {
			return nil, err
		}
		mp, err := b.read(8)
		if err != nil {
			return nil, err
		}
		if wt != 0 || tt != 0 || int(mp) >= len(t.maps) {
			return nil, ErrBadSetup
		}
		t.modes = append(t.modes, mode{blockflag: bf == 1, mapping: int(mp)})
	}
	fr, err := b.read1()
	if err != nil || fr != 1 {
		return nil, ErrBadSetup
	}
	return t, nil
}

func unpackMapping(b *bits, ch, nfloor, nres int) (mapping, error) {
	var m mapping
	sub, err := b.read1()
	if err != nil {
		return m, err
	}
	m.submaps = 1
	if sub == 1 {
		n, err := b.read(4)
		if err != nil {
			return m, err
		}
		m.submaps = int(n) + 1
	}
	coup, err := b.read1()
	if err != nil {
		return m, err
	}
	if coup == 1 {
		n, err := b.read(8)
		if err != nil {
			return m, err
		}
		bits := ilog(ch - 1)
		m.coupling = make([][2]int, int(n)+1)
		for i := range m.coupling {
			mag, err := b.read(bits)
			if err != nil {
				return m, err
			}
			ang, err := b.read(bits)
			if err != nil {
				return m, err
			}
			if int(mag) >= ch || int(ang) >= ch || mag == ang {
				return m, ErrBadSetup
			}
			m.coupling[i] = [2]int{int(mag), int(ang)}
		}
	}
	res, err := b.read(2)
	if err != nil || res != 0 {
		return m, ErrBadSetup
	}
	m.mux = make([]int, ch)
	if m.submaps > 1 {
		for i := 0; i < ch; i++ {
			v, err := b.read(4)
			if err != nil {
				return m, err
			}
			if int(v) >= m.submaps {
				return m, ErrBadSetup
			}
			m.mux[i] = int(v)
		}
	}
	m.subfloor = make([]int, m.submaps)
	m.subres = make([]int, m.submaps)
	for i := 0; i < m.submaps; i++ {
		if _, err := b.read(8); err != nil {
			return m, err
		}
		fl, err := b.read(8)
		if err != nil {
			return m, err
		}
		rs, err := b.read(8)
		if err != nil {
			return m, err
		}
		if int(fl) >= nfloor || int(rs) >= nres {
			return m, ErrBadSetup
		}
		m.subfloor[i] = int(fl)
		m.subres[i] = int(rs)
	}
	return m, nil
}

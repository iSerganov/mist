package vorbis

import "errors"

// resSym is one codebook symbol in residue decode order. Class words
// have class set; VQ steps carry the buffer offset they add into.
type resSym struct {
	class bool
	book  int
	entry int
	pass  int
	ch    int // type 0/1 submap channel; -1 for type 2
	off   int // spectral index (type 0/1) or interleaved work index (type 2)
	dim   int
}

type residueProg struct {
	typ2  bool
	nch   int
	chmap []int
	syms  []resSym
}

func (p residueProg) coord(s resSym, j int) (ch, spec int) {
	if p.typ2 {
		w := s.off + j
		if p.nch < 1 {
			return -1, -1
		}
		spec = w / p.nch
		ci := w % p.nch
		if ci < 0 || ci >= len(p.chmap) {
			return -1, -1
		}
		return p.chmap[ci], spec
	}
	if s.ch < 0 || s.ch >= len(p.chmap) {
		return -1, -1
	}
	return p.chmap[s.ch], s.off + j
}

type residueCfg struct {
	typ      int
	begin    int
	end      int
	partSize int
	classif  int
	classbk  int
	books    [][8]int
}

func unpackResidue(b *bits, typ, nbooks int) (residueCfg, error) {
	var r residueCfg
	r.typ = typ
	if typ > 2 {
		return r, ErrBadSetup
	}
	begin, err := b.read(24)
	if err != nil {
		return r, err
	}
	end, err := b.read(24)
	if err != nil {
		return r, err
	}
	ps, err := b.read(24)
	if err != nil {
		return r, err
	}
	cl, err := b.read(6)
	if err != nil {
		return r, err
	}
	cb, err := b.read(8)
	if err != nil {
		return r, err
	}
	r.begin, r.end = int(begin), int(end)
	r.partSize = int(ps) + 1
	r.classif = int(cl) + 1
	r.classbk = int(cb)
	if r.classbk >= nbooks || r.end < r.begin || r.partSize < 1 {
		return r, ErrBadSetup
	}
	r.books = make([][8]int, r.classif)
	cascade := make([]int, r.classif)
	for i := 0; i < r.classif; i++ {
		low, err := b.read(3)
		if err != nil {
			return r, err
		}
		flag, err := b.read1()
		if err != nil {
			return r, err
		}
		high := uint32(0)
		if flag == 1 {
			high, err = b.read(5)
			if err != nil {
				return r, err
			}
		}
		cascade[i] = int(low | (high << 3))
	}
	for i := 0; i < r.classif; i++ {
		for j := 0; j < 8; j++ {
			r.books[i][j] = -1
			if cascade[i]&(1<<uint(j)) != 0 {
				bk, err := b.read(8)
				if err != nil {
					return r, err
				}
				if int(bk) >= nbooks {
					return r, ErrBadSetup
				}
				r.books[i][j] = int(bk)
			}
		}
	}
	return r, nil
}

func decodeResidue(b *bits, r residueCfg, books []*codebook, chs []int, n int, skip []bool) ([][]float64, residueProg, error) {
	ch := len(chs)
	out := make([][]float64, ch)
	for i := 0; i < ch; i++ {
		out[i] = make([]float64, n)
	}
	prog := residueProg{typ2: r.typ == 2, nch: ch, chmap: append([]int(nil), chs...)}
	if r.end > n {
		r.end = n
	}
	if r.begin >= r.end {
		return out, prog, nil
	}
	used := 0
	for i := 0; i < ch; i++ {
		if !skip[i] {
			used++
		}
	}
	if used == 0 {
		return out, prog, nil
	}

	var err error
	if r.typ == 2 {
		err = decodeResidue2(b, r, books, ch, n, skip, out, &prog)
	} else {
		err = decodeResidue01(b, r, books, ch, n, skip, out, &prog)
	}
	return out, prog, err
}

func decodeResidue01(b *bits, r residueCfg, books []*codebook, ch, n int, skip []bool, out [][]float64, prog *residueProg) error {
	partvals := (r.end - r.begin) / r.partSize
	classwords := books[r.classbk].dim
	classes := make([][]int, ch)
	for c := 0; c < ch; c++ {
		classes[c] = make([]int, partvals)
	}
	for pass := 0; pass < 8; pass++ {
		i := 0
		for i < partvals {
			if pass == 0 {
				for c := 0; c < ch; c++ {
					if skip[c] {
						continue
					}
					cw, err := books[r.classbk].decode(b)
					if err != nil {
						if errors.Is(err, errEOP) {
							return nil
						}
						return err
					}
					prog.syms = append(prog.syms, resSym{class: true, book: r.classbk, entry: cw})
					for k := classwords - 1; k >= 0; k-- {
						if i+k < partvals {
							classes[c][i+k] = cw % r.classif
						}
						cw /= r.classif
					}
				}
			}
			for k := 0; k < classwords && i < partvals; k++ {
				off := r.begin + i*r.partSize
				end := off + r.partSize
				for c := 0; c < ch; c++ {
					if skip[c] {
						continue
					}
					bk := r.books[classes[c][i]][pass]
					if bk < 0 {
						continue
					}
					ents, err := addVQ(b, books[bk], out[c][off:end])
					if err != nil && !errors.Is(err, errEOP) {
						return err
					}
					dim := books[bk].dim
					for vi, e := range ents {
						prog.syms = append(prog.syms, resSym{
							book: bk, entry: e, pass: pass, ch: c, off: off + vi*dim, dim: dim,
						})
					}
					if errors.Is(err, errEOP) {
						return nil
					}
				}
				i++
			}
		}
	}
	return nil
}

func decodeResidue2(b *bits, r residueCfg, books []*codebook, ch, n int, skip []bool, out [][]float64, prog *residueProg) error {
	used := 0
	for i := 0; i < ch; i++ {
		if !skip[i] {
			used++
		}
	}
	if used == 0 {
		return nil
	}
	// Type 2 interleaves every submap channel into one decode buffer.
	begin, end := r.begin, r.end
	if end > n {
		end = n
	}
	partvals := (end - begin) * ch / r.partSize
	if partvals < 1 {
		return nil
	}
	classwords := books[r.classbk].dim
	classes := make([]int, partvals)
	work := make([]float64, ch*n)
	for pass := 0; pass < 8; pass++ {
		i := 0
		for i < partvals {
			if pass == 0 {
				cw, err := books[r.classbk].decode(b)
				if err != nil {
					if errors.Is(err, errEOP) {
						deinterleave2(out, work, begin, end, ch)
						return nil
					}
					return err
				}
				prog.syms = append(prog.syms, resSym{class: true, book: r.classbk, entry: cw})
				for k := classwords - 1; k >= 0; k-- {
					if i+k < partvals {
						classes[i+k] = cw % r.classif
					}
					cw /= r.classif
				}
			}
			for k := 0; k < classwords && i < partvals; k++ {
				bk := r.books[classes[i]][pass]
				off := begin*ch + i*r.partSize
				if bk >= 0 {
					ents, err := addVQ(b, books[bk], work[off:off+r.partSize])
					if err != nil && !errors.Is(err, errEOP) {
						return err
					}
					dim := books[bk].dim
					for vi, e := range ents {
						prog.syms = append(prog.syms, resSym{
							book: bk, entry: e, pass: pass, ch: -1, off: off + vi*dim, dim: dim,
						})
					}
					if errors.Is(err, errEOP) {
						deinterleave2(out, work, begin, end, ch)
						return nil
					}
				}
				i++
			}
		}
	}
	deinterleave2(out, work, begin, end, ch)
	return nil
}

func deinterleave2(out [][]float64, work []float64, begin, end, ch int) {
	for i := begin; i < end; i++ {
		for c := 0; c < ch; c++ {
			out[c][i] = work[i*ch+c]
		}
	}
}

func addVQ(b *bits, cb *codebook, dst []float64) ([]int, error) {
	step := cb.dim
	if step < 1 {
		return nil, ErrBadPacket
	}
	var ents []int
	for off := 0; off+step <= len(dst); off += step {
		e, err := cb.decode(b)
		if err != nil {
			if errors.Is(err, errEOP) {
				return ents, errEOP
			}
			return ents, err
		}
		ents = append(ents, e)
		v := cb.vq(e)
		for j := 0; j < step; j++ {
			dst[off+j] += v[j]
		}
	}
	return ents, nil
}

// sanitizePrograms repairs VQ entries that Match() perturbed to an unused
// codeword, and constrains every replacement to a codeword of the same
// bit length as orig recorded before the perturbation. Vorbis packets rely
// on running out of bits (EOP) mid-partition to mean "rest is zero"; a
// replacement with a different code length shifts that boundary and
// desyncs every residue decoded after it, even though the packet stays
// bit-valid. orig is the flat, in-order list of pre-perturbation entries
// for every non-class symbol across progs.
func sanitizePrograms(books []*codebook, progs []residueProg, orig []int) {
	idx := 0
	for pi := range progs {
		for i, s := range progs[pi].syms {
			if s.class || s.book < 0 || s.book >= len(books) {
				continue
			}
			cb := books[s.book]
			wantLen := uint8(0)
			if idx < len(orig) {
				o := orig[idx]
				if o >= 0 && o < cb.entries {
					wantLen = cb.lens[o]
				}
			}
			idx++
			if s.entry >= 0 && s.entry < cb.entries && cb.lens[s.entry] == wantLen && wantLen > 0 {
				continue
			}
			progs[pi].syms[i].entry = nearestLSBLen(cb, s.entry, wantLen)
		}
	}
}

// entryValues returns the entry of every non-class symbol across progs, in
// the same order sanitizePrograms and applyResidues walk them.
func entryValues(progs []residueProg) []int {
	var out []int
	for _, p := range progs {
		for _, s := range p.syms {
			if s.class {
				continue
			}
			out = append(out, s.entry)
		}
	}
	return out
}

// nearestLSBLen finds the used entry closest to want whose LSB matches
// want's and whose code length equals wantLen (the original entry's
// length, so the packet's bit length is unchanged). If no entry shares
// that length, or wantLen is unknown, it falls back to the closest used
// entry with the right LSB regardless of length.
func nearestLSBLen(cb *codebook, want int, wantLen uint8) int {
	bit := want & 1
	if want < 0 {
		bit = 0
	}
	best, bestD := -1, int(^uint(0)>>1)
	for _, e := range cb.used {
		if e&1 != bit || (wantLen > 0 && cb.lens[e] != wantLen) {
			continue
		}
		d := e - want
		if d < 0 {
			d = -d
		}
		if d < bestD {
			best, bestD = e, d
		}
	}
	if best >= 0 {
		return best
	}
	if wantLen > 0 {
		return nearestLSBLen(cb, want, 0)
	}
	if len(cb.used) == 0 {
		return 0
	}
	return cb.used[0]
}

func replayResidue(w *writer, books []*codebook, p residueProg) error {
	for _, s := range p.syms {
		if s.book < 0 || s.book >= len(books) {
			return ErrBadPacket
		}
		if err := books[s.book].encode(w, s.entry); err != nil {
			e := nearestClassword(books[s.book], s.entry)
			if err := books[s.book].encode(w, e); err != nil {
				return err
			}
		}
	}
	return nil
}

func encodeResidue(w *writer, r residueCfg, books []*codebook, vecs [][]float64, skip []bool, n int) error {
	if r.typ == 2 {
		return encodeResidue2(w, r, books, vecs, skip, n)
	}
	return encodeResidue01(w, r, books, vecs, skip, n)
}

func encodeResidue01(w *writer, r residueCfg, books []*codebook, vecs [][]float64, skip []bool, n int) error {
	if r.end > n {
		r.end = n
	}
	partvals := (r.end - r.begin) / r.partSize
	classwords := books[r.classbk].dim
	work := make([][]float64, len(vecs))
	for c := range vecs {
		work[c] = append([]float64(nil), vecs[c]...)
	}
	classes := make([][]int, len(vecs))
	for c := range vecs {
		classes[c] = classifyChannel(work[c], r, partvals)
	}
	for pass := 0; pass < 8; pass++ {
		i := 0
		for i < partvals {
			if pass == 0 {
				for c := range vecs {
					if skip[c] {
						continue
					}
					cw := 0
					for k := 0; k < classwords && i+k < partvals; k++ {
						cw = cw*r.classif + classes[c][i+k]
					}
					if err := books[r.classbk].encode(w, cw); err != nil {
						// fall back: try entry 0
						if err := books[r.classbk].encode(w, nearestClassword(books[r.classbk], cw)); err != nil {
							return err
						}
					}
				}
			}
			for k := 0; k < classwords && i < partvals; k++ {
				off := r.begin + i*r.partSize
				for c := range vecs {
					if skip[c] {
						continue
					}
					bk := r.books[classes[c][i]][pass]
					if bk >= 0 {
						end := off + r.partSize
						if err := subVQ(w, books[bk], work[c][off:end]); err != nil {
							return err
						}
					}
				}
				i++
			}
		}
	}
	return nil
}

func encodeResidue2(w *writer, r residueCfg, books []*codebook, vecs [][]float64, skip []bool, n int) error {
	ch := len(vecs)
	used := 0
	for i := 0; i < ch; i++ {
		if !skip[i] {
			used++
		}
	}
	if used == 0 {
		return nil
	}
	begin, end := r.begin, r.end
	if end > n {
		end = n
	}
	partvals := (end - begin) * ch / r.partSize
	classwords := books[r.classbk].dim
	work := make([]float64, ch*n)
	for i := begin; i < end; i++ {
		for c := 0; c < ch; c++ {
			work[i*ch+c] = vecs[c][i]
		}
	}
	classes := classifyInterleaved(work, r, begin, ch, partvals)
	for pass := 0; pass < 8; pass++ {
		i := 0
		for i < partvals {
			if pass == 0 {
				cw := 0
				for k := 0; k < classwords && i+k < partvals; k++ {
					cw = cw*r.classif + classes[i+k]
				}
				if err := books[r.classbk].encode(w, cw); err != nil {
					if err := books[r.classbk].encode(w, nearestClassword(books[r.classbk], cw)); err != nil {
						return err
					}
				}
			}
			for k := 0; k < classwords && i < partvals; k++ {
				bk := r.books[classes[i]][pass]
				off := begin*ch + i*r.partSize
				if bk >= 0 {
					if err := subVQ(w, books[bk], work[off:off+r.partSize]); err != nil {
						return err
					}
				}
				i++
			}
		}
	}
	return nil
}

func classifyChannel(v []float64, r residueCfg, partvals int) []int {
	out := make([]int, partvals)
	for i := 0; i < partvals; i++ {
		off := r.begin + i*r.partSize
		var mag float64
		for j := 0; j < r.partSize && off+j < len(v); j++ {
			x := v[off+j]
			if x < 0 {
				x = -x
			}
			if x > mag {
				mag = x
			}
		}
		cl := int(mag) % r.classif
		if cl < 0 {
			cl = 0
		}
		out[i] = cl
	}
	return out
}

func classifyInterleaved(work []float64, r residueCfg, begin, ch, partvals int) []int {
	out := make([]int, partvals)
	for i := 0; i < partvals; i++ {
		off := begin*ch + i*r.partSize
		var mag float64
		for j := 0; j < r.partSize && off+j < len(work); j++ {
			x := work[off+j]
			if x < 0 {
				x = -x
			}
			if x > mag {
				mag = x
			}
		}
		cl := 0
		if r.classif > 1 {
			cl = int(mag) % r.classif
		}
		out[i] = cl
	}
	return out
}

func nearestClassword(cb *codebook, want int) int {
	if want >= 0 && want < cb.entries && cb.lens[want] > 0 {
		return want
	}
	if len(cb.used) == 0 {
		return 0
	}
	return cb.used[0]
}

func subVQ(w *writer, cb *codebook, dst []float64) error {
	step := cb.dim
	if step < 1 {
		return nil
	}
	for off := 0; off+step <= len(dst); off += step {
		e := cb.nearest(dst[off : off+step])
		if err := cb.encode(w, e); err != nil {
			return err
		}
		v := cb.vq(e)
		for j := 0; j < step; j++ {
			dst[off+j] -= v[j]
		}
	}
	return nil
}

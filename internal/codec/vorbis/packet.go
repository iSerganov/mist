package vorbis

import (
	"errors"

	"github.com/iSerganov/mist/internal/codec"
)

type packetState struct {
	mode       int
	n          int
	skip       []bool
	prefix     int // bit offset where residue begins
	residueEnd int // bit offset immediately after the last decoded residue symbol
	vecs       [][]float64
	progs      []residueProg
}

func decodeAudio(s *Setup, pkt []byte) (*packetState, error) {
	if s == nil || s.tables == nil {
		return nil, ErrBadSetup
	}
	t := s.tables
	b := newBits(pkt)
	typ, err := b.read1()
	if err != nil || typ != 0 {
		return nil, ErrBadPacket
	}
	mbits := ilog(len(t.modes) - 1)
	mi, err := b.read(mbits)
	if err != nil {
		return nil, err
	}
	if int(mi) >= len(t.modes) {
		return nil, ErrBadPacket
	}
	md := t.modes[mi]
	if md.blockflag {
		if _, err := b.read(2); err != nil { // prev, next window
			return nil, err
		}
	}
	n := s.Blocksize0 / 2
	if md.blockflag {
		n = s.Blocksize1 / 2
	}
	mp := t.maps[md.mapping]
	ch := s.Channels
	skip := make([]bool, ch)
	for i := 0; i < ch; i++ {
		sub := mp.mux[i]
		used, err := readFloor(b, t.floors[mp.subfloor[sub]], t.books)
		if err != nil {
			if errors.Is(err, errEOP) {
				skip[i] = true
				continue
			}
			return nil, err
		}
		skip[i] = !used
	}
	// inverse coupling: if either coupled channel is unused, both are
	for i := len(mp.coupling) - 1; i >= 0; i-- {
		mag, ang := mp.coupling[i][0], mp.coupling[i][1]
		if skip[mag] || skip[ang] {
			skip[mag] = true
			skip[ang] = true
		}
	}
	prefix := b.pos
	vecs := make([][]float64, ch)
	for i := 0; i < ch; i++ {
		vecs[i] = make([]float64, n)
	}
	var progs []residueProg
	// residue is decoded per submap
	for sub := 0; sub < mp.submaps; sub++ {
		var chs []int
		var subSkip []bool
		for c := 0; c < ch; c++ {
			if mp.mux[c] == sub {
				chs = append(chs, c)
				subSkip = append(subSkip, skip[c])
			}
		}
		if len(chs) == 0 {
			continue
		}
		rc := t.residues[mp.subres[sub]]
		got, prog, err := decodeResidue(b, rc, t.books, chs, n, subSkip)
		if err != nil && !errors.Is(err, errEOP) {
			return nil, err
		}
		if err != nil && errors.Is(err, errEOP) && got == nil {
			return nil, err
		}
		progs = append(progs, prog)
		if got != nil {
			for i, c := range chs {
				if i < len(got) {
					copy(vecs[c], got[i])
				}
			}
		}
	}
	return &packetState{mode: int(mi), n: n, skip: skip, prefix: prefix, residueEnd: b.pos, vecs: vecs, progs: progs}, nil
}

func residuesFrom(st *packetState, rate int, books []*codebook) []codec.Residue {
	var out []codec.Residue
	for _, prog := range st.progs {
		for _, s := range prog.syms {
			if s.class {
				continue
			}
			ch, spec := prog.coord(s, 0)
			hz := 0
			if st.n > 0 && rate > 0 && spec >= 0 {
				hz = spec * rate / (st.n * 2)
			}
			unflippable := true
			if s.book >= 0 && s.book < len(books) {
				unflippable = !hasSameLenOppositeParity(books[s.book], s.entry)
			}
			out = append(out, codec.Residue{
				Channel:     ch,
				Band:        hz,
				Index:       spec,
				Value:       int32(s.entry),
				Unflippable: unflippable,
			})
		}
	}
	return out
}

// hasSameLenOppositeParity reports whether cb has a used entry other than
// entry with the same code length and the opposite LSB. LSB matching can
// only flip a symbol safely onto such an entry — anything else changes
// the codeword's bit length and desyncs decode of everything after it.
func hasSameLenOppositeParity(cb *codebook, entry int) bool {
	if entry < 0 || entry >= cb.entries || cb.lens[entry] == 0 {
		return false
	}
	length := cb.lens[entry]
	for _, e := range cb.used {
		if e != entry && cb.lens[e] == length && (e&1) != (entry&1) {
			return true
		}
	}
	return false
}

func applyResidues(st *packetState, res []codec.Residue) {
	i := 0
	for pi := range st.progs {
		for si, s := range st.progs[pi].syms {
			if s.class {
				continue
			}
			if i >= len(res) {
				return
			}
			st.progs[pi].syms[si].entry = int(res[i].Value)
			i++
		}
	}
}

func encodeAudio(s *Setup, pkt []byte, st *packetState) ([]byte, error) {
	t := s.tables
	md := t.modes[st.mode]
	mp := t.maps[md.mapping]
	w := newWriter()
	w.writeBits(pkt, st.prefix)
	ch := s.Channels
	n := st.n
	pi := 0
	for sub := 0; sub < mp.submaps; sub++ {
		var chs []int
		var subSkip []bool
		var subVec [][]float64
		for c := 0; c < ch; c++ {
			if mp.mux[c] == sub {
				chs = append(chs, c)
				subSkip = append(subSkip, st.skip[c])
				subVec = append(subVec, st.vecs[c])
			}
		}
		if len(chs) == 0 {
			continue
		}
		if pi < len(st.progs) {
			if err := replayResidue(w, t.books, st.progs[pi]); err != nil {
				return nil, err
			}
			pi++
			continue
		}
		rc := t.residues[mp.subres[sub]]
		if err := encodeResidue(w, rc, t.books, subVec, subSkip, n); err != nil {
			return nil, err
		}
	}
	// Vorbis residue decode treats running out of bits mid-partition as
	// "assume zero for the rest" (backfill). A replayed entry can have a
	// different Huffman code length than the one it replaces, so the
	// replayed residue section may end a few bits earlier or later than
	// the original did. Reproducing the original packet's own trailing
	// bits verbatim — already proven, by decoding it, not to complete one
	// symbol further — keeps that same-EOP behavior regardless of any
	// per-entry length drift upstream.
	total := len(pkt) * 8
	if st.residueEnd < total {
		w.writeBitsFrom(pkt, st.residueEnd, total-st.residueEnd)
	}
	return w.bytes(), nil
}

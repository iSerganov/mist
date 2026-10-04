package stego

import (
	"errors"
	"math"
)

// stcHeight is the constraint height h of the syndrome-trellis code: the
// trellis keeps 2^h states per cover element. Higher h finds cheaper
// flips at a cost in time and memory that doubles with every step.
const stcHeight = 7

const stcStates = 1 << stcHeight

var errNoSolution = errors.New("stego: no flip pattern carries the message")

// stcCode is a syndrome-trellis code (Filler, Judas, Fridrich 2011). The
// parity-check matrix H is m rows by m*w columns, built from one h-by-w
// submatrix repeated down the diagonal and shifted a row per block of w
// columns. The receiver recovers the message as H·y, with y the LSBs of
// the stego carrier; the sender picks the cheapest y with that syndrome.
// Carrying one bit per w cover elements this way needs several times
// fewer flips than writing each bit into a position of its own.
type stcCode struct {
	// cols[t] is column t of the submatrix; bit i is the row i below the
	// block's own row.
	cols []uint32
}

// newSTC draws the submatrix from the position key. Every column has its
// first and last row set, which makes the matrix triangular with a unit
// diagonal, so any message has a solution.
func newSTC(posKey []byte, w int) stcCode {
	ks := newKeyStream(posKey, "mist-stc-v2")
	cols := make([]uint32, w)
	for t := range cols {
		cols[t] = uint32(ks.next())&(stcStates-1) | 1 | stcStates>>1
	}
	return stcCode{cols: cols}
}

// syndrome returns H·y, the m message bits y carries.
func (c stcCode) syndrome(y []uint8, m int) []uint8 {
	w := len(c.cols)
	out := make([]uint8, m)
	var state uint32
	for b := range out {
		for t, col := range c.cols {
			if y[b*w+t]&1 == 1 {
				state ^= col
			}
		}
		out[b] = uint8(state & 1)
		state >>= 1
	}
	return out
}

// embed returns the indexes of cover whose bit must change so that the
// syndrome of the result is msg, at the least total cost. cost[j] is what
// changing cover[j] adds; len(cover) must be len(msg)*len(c.cols).
//
// It is a Viterbi search. A state holds the partial sums of the h
// syndrome rows the current block can still touch; each cover element
// either keeps its bit or flips it, and flipping XORs its column into the
// state. At the end of a block the lowest row is finished and must equal
// the message bit, which prunes every path that disagrees.
func (c stcCode) embed(cover []uint8, cost []float32, msg []uint8) ([]int, error) {
	w, m := len(c.cols), len(msg)
	if len(cover) != m*w || len(cost) != len(cover) {
		return nil, errNoSolution
	}
	inf := float32(math.Inf(1))
	cur := make([]float32, stcStates)
	nxt := make([]float32, stcStates)
	for s := range cur {
		cur[s] = inf
	}
	cur[0] = 0

	// path records, per cover element and state, whether the best way
	// into that state sets the stego bit to one.
	const words = stcStates / 64
	path := make([]uint64, len(cover)*words)

	for b, bit := range msg {
		for t, col := range c.cols {
			j := b*w + t
			// zero and one are the costs of stego bit 0 and 1. A tie goes to
			// whichever leaves the cover bit alone: breaking it towards one
			// fixed bit would flip odd values more often than even ones.
			zero, one, preferOne := cost[j], float32(0), true
			if cover[j]&1 == 0 {
				zero, one, preferOne = 0, cost[j], false
			}
			p := path[j*words : (j+1)*words]
			for s := range stcStates {
				stay := cur[s] + zero
				move := cur[s^int(col)] + one
				if move < stay || preferOne && move == stay {
					nxt[s] = move
					p[s>>6] |= 1 << (s & 63)
				} else {
					nxt[s] = stay
				}
			}
			cur, nxt = nxt, cur
		}
		for s := range stcStates / 2 {
			nxt[s] = cur[s<<1|int(bit)]
		}
		for s := stcStates / 2; s < stcStates; s++ {
			nxt[s] = inf
		}
		cur, nxt = nxt, cur
	}

	best := 0
	for s := range cur {
		if cur[s] < cur[best] {
			best = s
		}
	}
	if math.IsInf(float64(cur[best]), 1) {
		return nil, errNoSolution
	}

	var flips []int
	state := best
	for b := m - 1; b >= 0; b-- {
		state = state<<1 | int(msg[b])
		for t := w - 1; t >= 0; t-- {
			j := b*w + t
			if path[j*words+state>>6]>>(state&63)&1 == 1 {
				state ^= int(c.cols[t])
				if cover[j]&1 == 0 {
					flips = append(flips, j)
				}
			} else if cover[j]&1 == 1 {
				flips = append(flips, j)
			}
		}
	}
	return flips, nil
}

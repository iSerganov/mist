package mist

import "github.com/iSerganov/mist/internal/wire"

// planChunks decides how to place full — the fully marshaled payload —
// across a carrier's stego frames, given each frame's available room in
// bytes (as stego.Capacity / Samples.Capacity report). It always prefers
// writing full into a single frame, unmodified, over spanning: spanning is
// only attempted when no one frame is large enough by itself, so a
// message that already fits costs nothing extra and stays byte-for-byte
// compatible with a non-spanning reader.
//
// The result is one entry per frame — nil for a frame that carries no
// chunk this round, and the framed bytes to seal otherwise — or nil
// entirely when full fits nowhere, even split across every frame offered.
func planChunks(full []byte, room []int) [][]byte {
	if i, ok := firstFit(full, room); ok {
		out := make([][]byte, len(room))
		out[i] = full
		return out
	}
	return planSpan(full, room)
}

func firstFit(full []byte, room []int) (int, bool) {
	for i, r := range room {
		if r >= EnvelopeOverhead+len(full) {
			return i, true
		}
	}
	return 0, false
}

// planSpan is the multi-frame path: it walks the frames in order, filling
// each one that has room with as much of full as fits, until full is
// exhausted or the frames are. The first frame it uses opens the span
// with MarshalSpanStart, carrying the total length so a reader knows when
// reassembly is complete; every frame after that is MarshalSpanContinue,
// since the total is already known.
func planSpan(full []byte, room []int) [][]byte {
	out := make([][]byte, len(room))
	off, started := 0, false
	for i, r := range room {
		if off >= len(full) {
			break
		}
		header := wire.SpanContinueHeaderSize
		if !started {
			header = wire.SpanStartHeaderSize
		}
		avail := r - EnvelopeOverhead - header
		if avail <= 0 {
			continue
		}
		n := min(avail, len(full)-off)
		if !started {
			out[i] = wire.MarshalSpanStart(uint32(len(full)), full[off:off+n])
			started = true
		} else {
			out[i] = wire.MarshalSpanContinue(full[off : off+n])
		}
		off += n
	}
	if off < len(full) {
		return nil
	}
	return out
}

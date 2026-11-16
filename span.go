package mist

import "github.com/iSerganov/mist/internal/wire"

// spanPlan places full, the fully marshaled payload, one stego frame at a
// time as each frame's room becomes known, so the carrier is read once.
// The payload goes whole and unmodified into the first frame with any room
// when it fits there, so it stays readable by a catcher that predates
// spanning. When that frame is too small, the span starts in it and takes
// as many consecutive usable frames as it needs: the first chunk framed by
// MarshalSpanStart, which carries the total length, every later one by
// MarshalSpanContinue. A frame with no room neither starts nor continues it.
//
// With every set (WithReseal) full instead goes whole into every frame that
// holds it, and never spans.
type spanPlan struct {
	full    []byte
	every   bool
	off     int // bytes of full placed so far
	started bool
}

// place returns the framed bytes to seal into a frame with room bytes, or
// nil for filler.
func (p *spanPlan) place(room int) []byte {
	if p.every || !p.started {
		if fits(p.full, room) {
			p.started, p.off = true, len(p.full)
			return p.full
		}
		if p.every {
			return nil
		}
	}
	avail := spanRoom(room, !p.started)
	if p.off >= len(p.full) || avail <= 0 {
		return nil
	}
	chunk := p.full[p.off : p.off+min(avail, len(p.full)-p.off)]
	p.off += len(chunk)
	if p.started {
		return wire.MarshalSpanContinue(chunk)
	}
	p.started = true
	return wire.MarshalSpanStart(uint32(len(p.full)), chunk)
}

// complete reports whether all of full has been placed.
func (p *spanPlan) complete() bool { return p.off >= len(p.full) }

// fits reports whether a frame with room bytes holds full in one piece.
func fits(full []byte, room int) bool {
	return room >= EnvelopeOverhead+len(full)
}

func spanRoom(room int, first bool) int {
	header := wire.SpanContinueHeaderSize
	if first {
		header = wire.SpanStartHeaderSize
	}
	return room - EnvelopeOverhead - header
}

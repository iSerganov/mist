// Package frame models fixed-duration stego frames.
//
// A live stream has no known total length and a receiver may join mid-stream,
// so Mist divides the compressed packet stream into consecutive self-contained
// frames of duration D (a protocol constant). The same payload is re-sealed
// with a fresh ephemeral key in every frame.
//
// Frames are grouped in the packet domain, by the timestamps libav already
// carries, so Embed and Listen derive identical packet sets without either
// side reanalysing PCM. That agreement is load-bearing: one packet's
// difference changes the eligible-residue count and scrambles every
// selected position.
package frame

import (
	"time"

	"github.com/iSerganov/mist/internal/codec"
)

// Params are the audio properties needed to convert duration to samples.
type Params struct {
	SampleRate int
	Channels   int
	Duration   time.Duration
}

// Samples returns the number of interleaved PCM frames (sample-instants)
// in one stego frame for p.
func (p Params) Samples() int {
	if p.SampleRate <= 0 || p.Duration <= 0 {
		return 0
	}
	return int(p.Duration.Seconds() * float64(p.SampleRate))
}

// Group is one stego frame: the consecutive packets whose timestamps fall
// in the same window. Partial marks a group whose window did not start at
// the first packet seen — a listener that joined mid-transmission.
type Group struct {
	Index   int64
	Partial bool
	Packets []codec.Packet
}

// Grouper assembles packets into Groups as they arrive, so a live stream is
// scanned without buffering it whole.
type Grouper struct {
	window  int64
	idx     int64
	open    bool
	partial bool
	emitted bool
	pkts    []codec.Packet
}

// NewGrouper returns a Grouper for one frame window of p.
func NewGrouper(p Params) *Grouper {
	return &Grouper{window: int64(p.Samples())}
}

// Push adds pkt and returns the group it completed, if any.
func (g *Grouper) Push(pkt codec.Packet) (Group, bool) {
	if g.window <= 0 {
		return Group{}, false
	}
	idx := max(pkt.PTS, 0) / g.window
	var done Group
	var ok bool
	if g.open && idx != g.idx {
		done, ok = g.Flush()
	}
	if !g.open {
		// Packets rarely land on a window boundary, so a gap only means a
		// missed frame start for the very first group: after that the
		// transition was witnessed here.
		g.open, g.idx = true, idx
		g.partial = !g.emitted && pkt.PTS > idx*g.window
	}
	g.pkts = append(g.pkts, pkt)
	return done, ok
}

// Flush returns the group still being assembled, if any.
func (g *Grouper) Flush() (Group, bool) {
	if !g.open {
		return Group{}, false
	}
	out := Group{Index: g.idx, Partial: g.partial, Packets: g.pkts}
	g.open, g.emitted, g.pkts = false, true, nil
	return out, true
}

// GroupPackets splits pkts into stego frames.
func GroupPackets(pkts []codec.Packet, p Params) []Group {
	g := NewGrouper(p)
	out := make([]Group, 0, 1)
	for _, pkt := range pkts {
		if done, ok := g.Push(pkt); ok {
			out = append(out, done)
		}
	}
	if done, ok := g.Flush(); ok {
		out = append(out, done)
	}
	return out
}

// Capacity is the payload bytes that fit in one frame after encryption
// overhead, given nEligible coefficients and a constant embedding density.
func Capacity(nEligible int, density float64, overhead int) int {
	if nEligible <= 0 || density <= 0 {
		return 0
	}
	bytes := int(float64(nEligible)*density) / 8
	if bytes <= overhead {
		return 0
	}
	return bytes - overhead
}

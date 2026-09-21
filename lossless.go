package mist

import (
	"context"
	"errors"
	"fmt"

	"github.com/iSerganov/mist/internal/codec"
	"github.com/iSerganov/mist/internal/crypto"
	"github.com/iSerganov/mist/internal/frame"
	"github.com/iSerganov/mist/internal/stego"
)

// A lossless codec reproduces its input samples exactly, so Mist embeds
// into the PCM itself rather than into a compressed bitstream: no residue
// parse, no rewrite, and nothing that depends on the codec at all. What
// stays identical to the Vorbis path is everything that defines the
// protocol — FrameDuration windows, keyed positions, constant density,
// the payload in the first frame with room and filler in every other.
//
// The cost is the same one the Vorbis path pays: a listener that joins
// after the carrying frame recovers nothing.

// padToWindow extends pcm to a whole number of encoder sends. The encoder
// zero-pads its final send anyway and a lossless decoder hands that
// padding straight back, so Listen would see a longer last frame than
// Embed wrote into and read every position in it from the wrong place.
// Padding up front is what makes the two agree about the tail.
func padToWindow(pcm codec.PCM, window int) codec.PCM {
	if window <= 0 || pcm.NbSamples <= 0 || pcm.NbSamples%window == 0 {
		return pcm
	}
	n := (pcm.NbSamples/window + 1) * window
	for i, p := range pcm.Planes {
		if n > len(p) {
			pcm.Planes[i] = append(p, make([]float32, n-len(p))...)
		}
	}
	pcm.NbSamples = n
	return pcm
}

// embedSamples writes the sealed payload into pcm in place. scale is the
// integer grid the chosen encoder quantizes to, so the samples this
// leaves behind are ones it can represent exactly.
func (e *Emitter) embedSamples(ctx context.Context, pcm codec.PCM, scale float32, plain []byte) error {
	carried := false
	for _, w := range sampleFrames(pcm, scale) {
		if err := ctx.Err(); err != nil {
			return err
		}
		var bits stego.Bits
		if !carried && w.Capacity() >= EnvelopeOverhead+len(plain) {
			env, err := crypto.Seal(plain, e.pub)
			if err != nil {
				return err
			}
			bits, carried = env.Marshal(), true
		}
		pos, err := crypto.PositionSeed(e.pub, w.index)
		if err != nil {
			return err
		}
		// A window too short to hold even one bit is left alone, the way
		// a packet group with no eligible residues is.
		if err := stego.ApplySamples(w.Samples, pos, bits); err != nil &&
			!errors.Is(err, stego.ErrNoResidues) {
			return fmt.Errorf("%w: %v", ErrCarrier, err)
		}
	}
	if !carried {
		return noCapacity(plain)
	}
	return nil
}

// sampleFrame is one stego frame's worth of PCM, still pointing into the
// carrier's own planes so writing to it writes to the carrier. partial
// marks a window a listener only saw the tail of — the sample-domain
// twin of frame.Group.Partial, carrying no samples at all.
type sampleFrame struct {
	stego.Samples
	index   int64
	partial bool
}

// sampleFrames cuts pcm into consecutive frames. The window comes from
// frame.Params, the same function the packet-domain grouper uses, so the
// two domains cannot drift apart on where a frame begins.
func sampleFrames(pcm codec.PCM, scale float32) []sampleFrame {
	win := frameParams(pcm).Samples()
	if win <= 0 || len(pcm.Planes) == 0 {
		return nil
	}
	var out []sampleFrame
	for off := 0; off < pcm.NbSamples; off += win {
		out = append(out, sampleFrame{
			Samples: stego.Samples{
				Planes: pcm.Planes,
				Off:    off,
				N:      min(win, pcm.NbSamples-off),
				Scale:  scale,
			},
			index: int64(off / win),
		})
	}
	return out
}

// windower reassembles a decoded lossless stream into the same frames
// Embed wrote, so Listen can read their sample LSBs back. It is the
// sample-domain counterpart of frame.Grouper and shares its boundaries.
type windower struct {
	win    int
	scale  float32
	planes [][]float32
	idx    int64
	skip   int // samples still to discard before the next whole frame
	phased bool
}

func newWindower(p frame.Params, scale float32) *windower {
	return &windower{win: p.Samples(), scale: scale}
}

// push adds one decoded chunk and returns every frame it completed.
// The first chunk's timestamp sets the phase: a listener that joined
// mid-window cannot align to it, so that window is reported as partial
// and its samples dropped. There is no phase search.
func (w *windower) push(pcm codec.PCM) []sampleFrame {
	if w.win <= 0 || len(pcm.Planes) == 0 {
		return nil
	}
	var out []sampleFrame
	if !w.phased {
		w.phased = true
		pos := max(pcm.PTS, 0)
		w.idx = pos / int64(w.win)
		w.planes = make([][]float32, len(pcm.Planes))
		if rem := pos % int64(w.win); rem != 0 {
			w.skip = int(int64(w.win) - rem)
			out = append(out, sampleFrame{index: w.idx, partial: true})
			w.idx++
		}
	}
	// A decoded chunk is far smaller than a frame, so the tail being
	// discarded usually spans many of them; the remaining count has to
	// survive between pushes or everything after it lands one frame out.
	drop := min(w.skip, len(pcm.Planes[0]))
	w.skip -= drop
	for i, p := range pcm.Planes {
		if i >= len(w.planes) || drop >= len(p) {
			continue
		}
		w.planes[i] = append(w.planes[i], p[drop:]...)
	}
	return append(out, w.ready(false)...)
}

// flush returns the trailing short window, which Embed also wrote.
func (w *windower) flush() []sampleFrame { return w.ready(true) }

func (w *windower) ready(final bool) []sampleFrame {
	var out []sampleFrame
	for len(w.planes) > 0 {
		have := len(w.planes[0])
		// A short window is only a frame once the stream has ended —
		// Embed wrote that trailing sliver too.
		if have == 0 || (have < w.win && !final) {
			return out
		}
		n := min(have, w.win)
		out = append(out, sampleFrame{
			Samples: stego.Samples{Planes: w.planes, N: n, Scale: w.scale},
			index:   w.idx,
		})
		// The frame just handed out still points at these planes, so the
		// remainder must be a fresh backing array, not a reslice.
		next := make([][]float32, len(w.planes))
		for i, p := range w.planes {
			next[i] = append([]float32(nil), p[n:]...)
		}
		w.planes, w.idx = next, w.idx+1
	}
	return out
}

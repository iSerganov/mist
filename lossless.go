package mist

import (
	"errors"
	"fmt"

	"github.com/iSerganov/mist/internal/av"
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

// sampleStage is the lossless half of the pipeline. It moves each decoded
// chunk onto the encoder's grid, cuts the stream into stego frames with
// the same windower Listen reads them back with, embeds into each frame
// and hands it to the encoder. One frame of samples is held at a time.
type sampleStage struct {
	pass
	enc  *av.Encoder
	mux  *packetWriter
	win  *windower
	rate int
	read int64 // samples decoded so far
	sent int64 // samples handed to the encoder so far
}

func (s *sampleStage) push(pcm codec.PCM) error {
	if s.enc == nil {
		if err := s.open(pcm); err != nil {
			return err
		}
	}
	n := len(pcm.Planes[0])
	// One decoded chunk is one Snap cut: ffmpeg resamples a frame at a
	// time and that frame's remainder rounds differently from the rest.
	if err := s.enc.Snap(pcm.Planes, []int{n}); err != nil {
		return fmt.Errorf("%w: %v", ErrCarrier, err)
	}
	pcm.PTS = s.read
	s.read += int64(n)
	return s.embed(s.win.push(pcm))
}

func (s *sampleStage) open(pcm codec.PCM) error {
	enc, err := openEncoder(s.e.target, pcm, s.src)
	if err != nil {
		return err
	}
	s.enc, s.rate = enc, pcm.SampleRate
	s.win = newWindower(frameParams(pcm), av.SampleScale(enc.Info().SampleFmt))
	if s.dst == nil {
		return nil
	}
	s.mux, err = openMux(s.dst, enc.Info(), s.meta)
	return err
}

func (s *sampleStage) embed(frames []sampleFrame) error {
	for _, w := range frames {
		chunk := s.place.place(w.Capacity())
		if s.mux == nil {
			continue
		}
		if err := s.e.embedWindow(w, chunk); err != nil {
			return err
		}
		pkts, err := s.enc.Encode(w.pcm(s.rate, s.sent))
		if err != nil {
			return err
		}
		s.sent += int64(w.N)
		if err := s.mux.write(pkts); err != nil {
			return err
		}
		s.report()
	}
	return nil
}

func (s *sampleStage) finish() error {
	if s.enc == nil {
		return ErrCarrier
	}
	if err := s.embed(s.win.flush()); err != nil {
		return err
	}
	if s.mux == nil {
		return nil
	}
	pkts, err := s.enc.Flush()
	if err != nil {
		return err
	}
	if err := s.mux.write(pkts); err != nil {
		return err
	}
	return s.mux.finish()
}

func (s *sampleStage) close() {
	if s.mux != nil {
		s.mux.close()
	}
	if s.enc != nil {
		_ = s.enc.Close()
	}
}

// embedWindow writes one stego frame into its samples in place: chunk
// sealed for the recipient, or filler at the same density when it is nil.
func (e *Emitter) embedWindow(w sampleFrame, chunk []byte) error {
	bits, err := e.seal(chunk)
	if err != nil {
		return err
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

// pcm is the frame's samples as the encoder takes them, pts samples into
// the stream.
func (f sampleFrame) pcm(rate int, pts int64) codec.PCM {
	planes := make([][]float32, len(f.Planes))
	for i, p := range f.Planes {
		planes[i] = p[f.Off : f.Off+f.N]
	}
	return codec.PCM{
		Planes: planes, NbSamples: f.N, Channels: len(planes),
		SampleRate: rate, Format: codec.SampleFmtFLTP, PTS: pts,
	}
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
		// remainder is capped at its own length (a three-index slice):
		// cap==len forces the next append to allocate instead of
		// overwriting what the caller was just given, without copying now.
		next := make([][]float32, len(w.planes))
		for i, p := range w.planes {
			next[i] = p[n:len(p):len(p)]
		}
		w.planes, w.idx = next, w.idx+1
	}
	return out
}

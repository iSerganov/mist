package av

import (
	"encoding/binary"
	"errors"
	"math"

	"github.com/iSerganov/mist/internal/codec"
)

// defaultWindow is the send size for an encoder that accepts any frame
// length (the raw PCM codecs). It only bounds how much audio crosses the
// cgo boundary at once; the packets it produces are unaffected.
const defaultWindow = 4096

// Window is the sample count the encoder takes per send. A codec that
// accepts any length still goes through defaultWindow, so a whole carrier
// never crosses the cgo boundary in one allocation.
func (e *Encoder) Window() int {
	if e == nil || e.info.FrameSize <= 0 {
		return defaultWindow
	}
	return e.info.FrameSize
}

// Encode implements codec.Encoder. It sends every whole encoder window in
// pcm and holds what is left over for the next call, so a stream can
// arrive in pieces of any size. libav accepts a short frame only as the
// last one, and Flush is what sends it.
func (e *Encoder) Encode(pcm codec.PCM) ([]codec.Packet, error) {
	if e == nil || e.handle == nil {
		return nil, ErrClosed
	}
	w := e.Window()
	var out []codec.Packet
	for _, chunk := range splitPCM(e.join(pcm), w) {
		if chunk.NbSamples < w {
			e.held = clonePCM(chunk)
			break
		}
		pkts, err := e.sendPCM(chunk)
		out = append(out, pkts...)
		if err != nil {
			return out, err
		}
	}
	return out, nil
}

// Flush implements codec.Encoder: it sends the short last window, if
// Encode left one, and drains the encoder's tail.
func (e *Encoder) Flush() ([]codec.Packet, error) {
	if e == nil || e.handle == nil {
		return nil, ErrClosed
	}
	var out []codec.Packet
	if e.held.NbSamples > 0 {
		pkts, err := e.sendPCM(e.held)
		e.held = codec.PCM{}
		out = pkts
		if err != nil {
			return out, err
		}
	}
	if err := e.Send(Frame{}); err != nil && !errors.Is(err, ErrAgain) && !errors.Is(err, ErrEOF) {
		return out, err
	}
	pkts, err := drainEnc(e)
	return append(out, pkts...), err
}

func (e *Encoder) sendPCM(chunk codec.PCM) ([]codec.Packet, error) {
	if err := e.Send(frameFromPCM(chunk)); err != nil && !errors.Is(err, ErrAgain) {
		return nil, err
	}
	return drainEnc(e)
}

// join puts the samples the last Encode held back in front of p and
// forgets them. The result keeps the held samples' PTS.
func (e *Encoder) join(p codec.PCM) codec.PCM {
	held := e.held
	e.held = codec.PCM{}
	if held.NbSamples == 0 {
		return p
	}
	planes := pcmPlanes(p)
	total := held.NbSamples + p.NbSamples
	for i := range held.Planes {
		if i < len(planes) {
			held.Planes[i] = append(held.Planes[i], planes[i][:min(len(planes[i]), p.NbSamples)]...)
		}
		held.Planes[i] = append(held.Planes[i], make([]float32, total-len(held.Planes[i]))...)
	}
	held.NbSamples = total
	return held
}

// clonePCM copies p's planes, which otherwise alias the caller's buffer.
func clonePCM(p codec.PCM) codec.PCM {
	planes := make([][]float32, len(p.Planes))
	for i, pl := range p.Planes {
		planes[i] = append([]float32(nil), pl...)
	}
	p.Planes = planes
	return p
}

// Params returns the stream after open, including any extradata the
// matching decoder needs.
func (e *Encoder) Params() codec.Params {
	if e == nil {
		return codec.Params{}
	}
	return e.info.Params()
}

// Decode implements codec.Decoder.
func (d *Decoder) Decode(pkt codec.Packet) ([]codec.PCM, error) {
	if d == nil || d.handle == nil {
		return nil, ErrClosed
	}
	if err := d.Send(FromCodecPacket(pkt)); err != nil && !errors.Is(err, ErrAgain) && !errors.Is(err, ErrEOF) {
		return nil, err
	}
	return DrainPCM(d)
}

// DrainPCM receives every frame the decoder can currently produce.
func DrainPCM(d *Decoder) ([]codec.PCM, error) {
	var out []codec.PCM
	for {
		fr, err := d.Receive()
		if errors.Is(err, ErrAgain) || errors.Is(err, ErrEOF) {
			return out, nil
		}
		if err != nil {
			return nil, err
		}
		out = append(out, FrameToPCM(fr))
	}
}

// FrameToPCM normalises whatever sample format the decoder produced into
// the float planes everything upstream of the encoder works in.
func FrameToPCM(f Frame) codec.PCM {
	return codec.PCM{
		Planes:     f.FloatPlanes(),
		NbSamples:  f.NbSamples,
		Channels:   f.Channels,
		SampleRate: f.SampleRate,
		Format:     codec.SampleFmtFLTP,
		PTS:        f.PTS,
	}
}

func drainEnc(e *Encoder) ([]codec.Packet, error) {
	var out []codec.Packet
	for {
		pkt, err := e.Receive()
		if errors.Is(err, ErrAgain) || errors.Is(err, ErrEOF) {
			return out, nil
		}
		if err != nil {
			return nil, err
		}
		out = append(out, ToCodecPacket(pkt))
	}
}

// splitPCM cuts p into encoder-sized windows and leaves the last one
// short. Zero-padding it here would write samples the carrier never had,
// which an ordinary encode does not do; libav pads the final frame itself
// for an encoder that needs it, exactly as the ffmpeg command line does.
func splitPCM(p codec.PCM, fs int) []codec.PCM {
	if fs <= 0 || p.NbSamples <= 0 {
		return nil
	}
	planes := pcmPlanes(p)
	var out []codec.PCM
	for off := 0; off < p.NbSamples; off += fs {
		n := min(fs, p.NbSamples-off)
		chunk := make([][]float32, len(planes))
		for i, pl := range planes {
			if off+n <= len(pl) {
				chunk[i] = pl[off : off+n]
				continue
			}
			sl := make([]float32, n)
			if off < len(pl) {
				copy(sl, pl[off:])
			}
			chunk[i] = sl
		}
		out = append(out, codec.PCM{
			Planes:     chunk,
			NbSamples:  n,
			Channels:   p.Channels,
			SampleRate: p.SampleRate,
			Format:     codec.SampleFmtFLTP,
			PTS:        p.PTS + int64(off),
		})
	}
	return out
}

// frameFromPCM packs float planes for Send. The frame stays FLTP whatever
// the encoder's own sample format is: the conversion to that format is the
// C layer's, and Send reads these bytes back as float32.
func frameFromPCM(p codec.PCM) Frame {
	planes := pcmPlanes(p)
	data := make([][]byte, len(planes))
	for i, pl := range planes {
		data[i] = floatsToBytes(pl)
	}
	return Frame{
		Data:       data,
		NbSamples:  p.NbSamples,
		Channels:   p.Channels,
		SampleRate: p.SampleRate,
		Format:     codec.SampleFmtFLTP,
		PTS:        p.PTS,
	}
}

func pcmPlanes(p codec.PCM) [][]float32 {
	switch {
	case len(p.Planes) > 0:
		return p.Planes
	case len(p.Samples) > 0 && p.Channels > 0:
		return deinterleave(p.Samples, p.Channels, p.NbSamples)
	default:
		return nil
	}
}

func floatsToBytes(in []float32) []byte {
	out := make([]byte, len(in)*4)
	for i, v := range in {
		binary.LittleEndian.PutUint32(out[i*4:], math.Float32bits(v))
	}
	return out
}

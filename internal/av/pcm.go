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
// never crosses the cgo boundary in one allocation. Encode zero-pads the
// final send out to this, which callers who care where a sample ends up
// have to account for.
func (e *Encoder) Window() int {
	if e == nil || e.info.FrameSize <= 0 {
		return defaultWindow
	}
	return e.info.FrameSize
}

// Encode implements codec.Encoder: it splits pcm into encoder-sized
// windows, sends each, and returns whatever packets came back.
func (e *Encoder) Encode(pcm codec.PCM) ([]codec.Packet, error) {
	if e == nil || e.handle == nil {
		return nil, ErrClosed
	}
	var out []codec.Packet
	for _, chunk := range splitPCM(pcm, e.Window()) {
		if err := e.Send(frameFromPCM(chunk)); err != nil && !errors.Is(err, ErrAgain) {
			return out, err
		}
		pkts, err := drainEnc(e)
		out = append(out, pkts...)
		if err != nil {
			return out, err
		}
	}
	return out, nil
}

// Flush implements codec.Encoder, draining the encoder's tail.
func (e *Encoder) Flush() ([]codec.Packet, error) {
	if e == nil || e.handle == nil {
		return nil, ErrClosed
	}
	if err := e.Send(Frame{}); err != nil && !errors.Is(err, ErrAgain) && !errors.Is(err, ErrEOF) {
		return nil, err
	}
	return drainEnc(e)
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

// splitPCM cuts p into encoder-sized windows. libvorbis rejects any
// nb_samples other than frame_size, so a short tail is zero-padded.
func splitPCM(p codec.PCM, fs int) []codec.PCM {
	if fs <= 0 || p.NbSamples <= 0 {
		return nil
	}
	planes := pcmPlanes(p)
	var out []codec.PCM
	for off := 0; off < p.NbSamples; off += fs {
		chunk := make([][]float32, len(planes))
		for i, pl := range planes {
			if off+fs <= len(pl) {
				chunk[i] = pl[off : off+fs]
				continue
			}
			sl := make([]float32, fs)
			if off < len(pl) {
				copy(sl, pl[off:len(pl)])
			}
			chunk[i] = sl
		}
		out = append(out, codec.PCM{
			Planes:     chunk,
			NbSamples:  fs,
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

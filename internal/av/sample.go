package av

import (
	"encoding/binary"
	"math"

	"github.com/iSerganov/mist/internal/codec"
)

func codecSampleFmt(n int) codec.SampleFormat {
	return codec.SampleFormat(n)
}

func isPlanar(f codec.SampleFormat) bool {
	l, ok := layouts[f]
	return ok && l.planar
}

// layout describes how one libav sample format is packed, so a decoded
// frame can be read whatever format the decoder happened to produce.
// Everything upstream of the encoder works in float32.
type layout struct {
	width  int
	planar bool
	value  func([]byte) float32
}

var layouts = map[codec.SampleFormat]layout{
	codec.SampleFmtU8:   {1, false, u8Value},
	codec.SampleFmtS16:  {2, false, s16Value},
	codec.SampleFmtS32:  {4, false, s32Value},
	codec.SampleFmtFLT:  {4, false, fltValue},
	codec.SampleFmtDBL:  {8, false, dblValue},
	codec.SampleFmtU8P:  {1, true, u8Value},
	codec.SampleFmtS16P: {2, true, s16Value},
	codec.SampleFmtS32P: {4, true, s32Value},
	codec.SampleFmtFLTP: {4, true, fltValue},
	codec.SampleFmtDBLP: {8, true, dblValue},
}

func u8Value(b []byte) float32  { return (float32(b[0]) - 128) / 128 }
func s16Value(b []byte) float32 { return float32(int16(binary.LittleEndian.Uint16(b))) / 32768 }
func s32Value(b []byte) float32 { return float32(int32(binary.LittleEndian.Uint32(b))) / 2147483648 }
func fltValue(b []byte) float32 { return math.Float32frombits(binary.LittleEndian.Uint32(b)) }

func dblValue(b []byte) float32 {
	return float32(math.Float64frombits(binary.LittleEndian.Uint64(b)))
}

// FloatPlanes returns one float32 slice per channel, converting from the
// frame's own sample format. It returns nil for a format libav produced
// that this package cannot read, which callers treat as no PCM.
func (f Frame) FloatPlanes() [][]float32 {
	l, ok := layouts[f.Format]
	if !ok || f.Channels <= 0 || len(f.Data) == 0 {
		return nil
	}
	if l.planar {
		out := make([][]float32, 0, len(f.Data))
		for _, plane := range f.Data {
			out = append(out, readSamples(plane, l, f.NbSamples))
		}
		return out
	}
	packed := readSamples(f.Data[0], l, f.NbSamples*f.Channels)
	return deinterleave(packed, f.Channels, f.NbSamples)
}

// readSamples reads at most n samples; a plane's linesize can exceed the
// samples actually present because libav pads its buffers.
func readSamples(b []byte, l layout, n int) []float32 {
	if avail := len(b) / l.width; n > avail {
		n = avail
	}
	out := make([]float32, max(n, 0))
	for i := range out {
		out[i] = l.value(b[i*l.width:])
	}
	return out
}

func deinterleave(in []float32, ch, n int) [][]float32 {
	if ch <= 0 {
		return nil
	}
	out := make([][]float32, ch)
	for c := range out {
		out[c] = make([]float32, n)
		for i := 0; i < n && i*ch+c < len(in); i++ {
			out[c][i] = in[i*ch+c]
		}
	}
	return out
}

package av

import (
	"encoding/binary"
	"math"

	"github.com/iSerganov/mist/internal/codec"
)

func codecSampleFmt(n int) codec.SampleFormat {
	return codec.SampleFormat(n)
}

func isPlanar(fmt codec.SampleFormat) bool {
	return fmt == codec.SampleFmtU8P || fmt == codec.SampleFmtS16P ||
		fmt == codec.SampleFmtS32P || fmt == codec.SampleFmtFLTP ||
		fmt == codec.SampleFmtDBLP
}

func frameFloatPlanes(f Frame) [][]float32 {
	if f.Format == codec.SampleFmtFLTP && len(f.Data) > 0 {
		out := make([][]float32, len(f.Data))
		for i, b := range f.Data {
			out[i] = bytesToFloats(b)
		}
		return out
	}
	if f.Format == codec.SampleFmtFLT && len(f.Data) > 0 {
		inter := bytesToFloats(f.Data[0])
		return deinterleave(inter, f.Channels, f.NbSamples)
	}
	return nil
}

func bytesToFloats(b []byte) []float32 {
	n := len(b) / 4
	out := make([]float32, n)
	for i := 0; i < n; i++ {
		out[i] = math.Float32frombits(binary.LittleEndian.Uint32(b[i*4:]))
	}
	return out
}

func deinterleave(in []float32, ch, n int) [][]float32 {
	if ch <= 0 {
		return nil
	}
	out := make([][]float32, ch)
	for c := 0; c < ch; c++ {
		out[c] = make([]float32, n)
		for i := 0; i < n && i*ch+c < len(in); i++ {
			out[c][i] = in[i*ch+c]
		}
	}
	return out
}

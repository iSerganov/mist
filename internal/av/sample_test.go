package av

import (
	"encoding/binary"
	"math"
	"testing"

	"github.com/iSerganov/mist/internal/codec"
	"github.com/stretchr/testify/suite"
)

type SampleSuite struct {
	suite.Suite
}

func TestSampleSuite(t *testing.T) {
	suite.Run(t, &SampleSuite{})
}

func s16Bytes(v ...int16) []byte {
	b := make([]byte, len(v)*2)
	for i, x := range v {
		binary.LittleEndian.PutUint16(b[i*2:], uint16(x))
	}
	return b
}

func s32Bytes(v ...int32) []byte {
	b := make([]byte, len(v)*4)
	for i, x := range v {
		binary.LittleEndian.PutUint32(b[i*4:], uint32(x))
	}
	return b
}

func fltBytes(v ...float32) []byte {
	b := make([]byte, len(v)*4)
	for i, x := range v {
		binary.LittleEndian.PutUint32(b[i*4:], math.Float32bits(x))
	}
	return b
}

func dblBytes(v ...float64) []byte {
	b := make([]byte, len(v)*8)
	for i, x := range v {
		binary.LittleEndian.PutUint64(b[i*8:], math.Float64bits(x))
	}
	return b
}

// Decoders emit whatever format suits them — FLAC gives s32, WAV s16,
// Vorbis fltp — so every one of them has to arrive as the same float
// planes the encoder takes.
func (s *SampleSuite) TestFloatPlanes() {
	tests := []struct {
		title string
		frame Frame
		want  [][]float32
	}{
		{
			title: "u8 packed is centred on 128",
			frame: Frame{Format: codec.SampleFmtU8, Channels: 2, NbSamples: 2,
				Data: [][]byte{{128, 192, 64, 128}}},
			want: [][]float32{{0, -0.5}, {0.5, 0}},
		},
		{
			title: "s16 packed",
			frame: Frame{Format: codec.SampleFmtS16, Channels: 2, NbSamples: 2,
				Data: [][]byte{s16Bytes(0, 16384, -16384, -32768)}},
			want: [][]float32{{0, -0.5}, {0.5, -1}},
		},
		{
			title: "s32 packed",
			frame: Frame{Format: codec.SampleFmtS32, Channels: 1, NbSamples: 2,
				Data: [][]byte{s32Bytes(1<<30, -(1 << 30))}},
			want: [][]float32{{0.5, -0.5}},
		},
		{
			title: "float packed",
			frame: Frame{Format: codec.SampleFmtFLT, Channels: 2, NbSamples: 2,
				Data: [][]byte{fltBytes(0.25, -0.25, 0.75, -0.75)}},
			want: [][]float32{{0.25, 0.75}, {-0.25, -0.75}},
		},
		{
			title: "double packed",
			frame: Frame{Format: codec.SampleFmtDBL, Channels: 1, NbSamples: 2,
				Data: [][]byte{dblBytes(0.5, -1)}},
			want: [][]float32{{0.5, -1}},
		},
		{
			title: "s16 planar keeps channels apart",
			frame: Frame{Format: codec.SampleFmtS16P, Channels: 2, NbSamples: 2,
				Data: [][]byte{s16Bytes(0, 16384), s16Bytes(-16384, -32768)}},
			want: [][]float32{{0, 0.5}, {-0.5, -1}},
		},
		{
			title: "s32 planar",
			frame: Frame{Format: codec.SampleFmtS32P, Channels: 1, NbSamples: 2,
				Data: [][]byte{s32Bytes(1<<30, 0)}},
			want: [][]float32{{0.5, 0}},
		},
		{
			title: "float planar",
			frame: Frame{Format: codec.SampleFmtFLTP, Channels: 2, NbSamples: 1,
				Data: [][]byte{fltBytes(0.125), fltBytes(-0.125)}},
			want: [][]float32{{0.125}, {-0.125}},
		},
		{
			title: "u8 planar",
			frame: Frame{Format: codec.SampleFmtU8P, Channels: 1, NbSamples: 2,
				Data: [][]byte{{0, 255}}},
			want: [][]float32{{-1, 0.9921875}},
		},
		{
			title: "double planar",
			frame: Frame{Format: codec.SampleFmtDBLP, Channels: 1, NbSamples: 1,
				Data: [][]byte{dblBytes(-0.75)}},
			want: [][]float32{{-0.75}},
		},
		{
			// libav pads its buffers, so a plane can hold more than
			// NbSamples; the surplus is not audio.
			title: "padded plane is trimmed to NbSamples",
			frame: Frame{Format: codec.SampleFmtS16P, Channels: 1, NbSamples: 2,
				Data: [][]byte{s16Bytes(0, 16384, 32767, 32767)}},
			want: [][]float32{{0, 0.5}},
		},
	}
	for _, tc := range tests {
		s.Run(tc.title, func() {
			got := tc.frame.FloatPlanes()
			s.Require().Len(got, len(tc.want))
			for ch := range tc.want {
				s.Require().Len(got[ch], len(tc.want[ch]), "channel %d", ch)
				for i, want := range tc.want[ch] {
					s.InDelta(want, got[ch][i], 1e-6, "channel %d sample %d", ch, i)
				}
			}
		})
	}
}

func (s *SampleSuite) TestFloatPlanesRejects() {
	tests := []struct {
		title string
		frame Frame
	}{
		{"unknown format", Frame{Format: codec.SampleFormat(99), Channels: 1, Data: [][]byte{{0}}}},
		{"no channels", Frame{Format: codec.SampleFmtS16, Data: [][]byte{{0, 0}}}},
		{"no data", Frame{Format: codec.SampleFmtS16, Channels: 2}},
		{"unset format", Frame{Format: codec.SampleFmtNone, Channels: 1, Data: [][]byte{{0}}}},
	}
	for _, tc := range tests {
		s.Run(tc.title, func() {
			s.Nil(tc.frame.FloatPlanes())
		})
	}
}

func (s *SampleSuite) TestIsPlanar() {
	tests := []struct {
		title string
		in    codec.SampleFormat
		want  bool
	}{
		{"fltp", codec.SampleFmtFLTP, true},
		{"s16p", codec.SampleFmtS16P, true},
		{"s16", codec.SampleFmtS16, false},
		{"flt", codec.SampleFmtFLT, false},
		{"none", codec.SampleFmtNone, false},
	}
	for _, tc := range tests {
		s.Run(tc.title, func() {
			s.Equal(tc.want, isPlanar(tc.in))
		})
	}
}

package av

import (
	"bytes"
	"encoding/binary"
	"errors"
	"io"
	"os"
	"testing"

	"github.com/iSerganov/mist/internal/codec"
	"github.com/stretchr/testify/suite"
)

type StreamSuite struct {
	suite.Suite
}

func TestStreamSuite(t *testing.T) {
	suite.Run(t, &StreamSuite{})
}

func (s *StreamSuite) SetupTest() {
	if !Available() {
		s.T().Skip("libav not available")
	}
}

const streamRate = 44100

func (s *StreamSuite) TestEncodeTakesTheStreamInPieces() {
	pcm := noisePCM(3*streamRate + 123)
	tests := []struct {
		title  string
		format string
		piece  int
	}{
		{"vorbis in pieces smaller than a window", "ogg", 100},
		{"vorbis in pieces that straddle windows", "ogg", 1000},
		{"flac in pieces smaller than a block", "flac", 1000},
		{"flac in pieces larger than a block", "flac", 10_000},
		{"wav in odd pieces", "wav", 777},
	}
	for _, tc := range tests {
		s.Run(tc.title, func() {
			var whole, pieces []codec.Packet
			s.encode(tc.format, pcm, pcm.NbSamples, nil, func(p codec.Packet) { whole = append(whole, p) })
			s.encode(tc.format, pcm, tc.piece, nil, func(p codec.Packet) { pieces = append(pieces, p) })
			s.Require().NotEmpty(whole)
			s.Equal(whole, pieces)
		})
	}
}

func (s *StreamSuite) TestAWriterThatCannotSeekGetsTheSameAudio() {
	pcm := noisePCM(2*streamRate + 77)
	tests := []struct {
		title  string
		format string
	}{
		{"ogg vorbis", "ogg"},
		{"flac", "flac"},
		{"wav", "wav"},
	}
	for _, tc := range tests {
		s.Run(tc.title, func() {
			var seekable rwBuf
			var plain bytes.Buffer
			s.write(&seekable, tc.format, pcm, 4000)
			s.write(&plain, tc.format, pcm, 4000)

			want := s.decode(seekable.data)
			got := s.decode(plain.Bytes())
			s.Require().Len(got[0], pcm.NbSamples)
			s.Equal(want, got)
		})
	}
}

func (s *StreamSuite) TestHeaderIsFinishedOnlyWhereTheWriterSeeks() {
	pcm := noisePCM(streamRate)
	flacMD5 := func(b []byte) []byte { return b[26:42] }
	riffSize := func(b []byte) int { return int(binary.LittleEndian.Uint32(b[4:8])) }

	var seekable rwBuf
	var plain bytes.Buffer
	s.write(&seekable, "flac", pcm, 4000)
	s.write(&plain, "flac", pcm, 4000)
	s.NotEqual(make([]byte, 16), flacMD5(seekable.data), "a seekable FLAC carries the MD5 of its audio")
	s.Equal(make([]byte, 16), flacMD5(plain.Bytes()), "a streamed FLAC keeps the header it started with")

	seekable, plain = rwBuf{}, bytes.Buffer{}
	s.write(&seekable, "wav", pcm, 4000)
	s.write(&plain, "wav", pcm, 4000)
	s.Equal(len(seekable.data)-8, riffSize(seekable.data), "a seekable WAV states its size")
	s.NotEqual(plain.Len()-8, riffSize(plain.Bytes()), "a streamed WAV cannot go back to state its size")
}

// write encodes pcm to format in pieces of the given size and muxes the
// packets to w as they come, header first, the order a stream needs.
func (s *StreamSuite) write(w io.Writer, format string, pcm codec.PCM, piece int) {
	s.T().Helper()
	var m *Muxer
	s.encode(format, pcm, piece, func(info AudioInfo) {
		var err error
		m, err = NewMuxer(w, info)
		s.Require().NoError(err)
		s.Require().NoError(m.WriteHeader())
	}, func(p codec.Packet) {
		s.Require().NoError(m.WritePacket(FromCodecPacket(p)))
	})
	s.Require().NoError(m.WriteTrailer())
	s.Require().NoError(m.Close())
}

// encode feeds pcm to a new encoder for format in pieces of the given
// size. opened, when set, sees the encoder's parameters before any packet.
func (s *StreamSuite) encode(format string, pcm codec.PCM, piece int, opened func(AudioInfo), emit func(codec.Packet)) {
	s.T().Helper()
	f, err := FindFormat(format, "")
	s.Require().NoError(err)
	enc, err := NewEncoder(f.Info(pcm.SampleRate, pcm.Channels, AudioInfo{SampleFmt: codec.SampleFmtS16, Bits: 16}))
	s.Require().NoError(err)
	defer func() { _ = enc.Close() }()
	if opened != nil {
		opened(enc.Info())
	}
	each := func(pkts []codec.Packet, err error) {
		s.Require().NoError(err)
		for _, p := range pkts {
			emit(p)
		}
	}
	for off := 0; off < pcm.NbSamples; off += piece {
		each(enc.Encode(slicePCM(pcm, off, min(piece, pcm.NbSamples-off))))
	}
	each(enc.Flush())
}

func (s *StreamSuite) decode(b []byte) [][]float32 {
	s.T().Helper()
	d, err := OpenDemuxerReader(bytes.NewReader(b))
	s.Require().NoError(err)
	defer func() { _ = d.Close() }()
	dec, err := NewDecoder(d.Info())
	s.Require().NoError(err)
	defer func() { _ = dec.Close() }()

	var out [][]float32
	collect := func(chunks []codec.PCM, err error) {
		s.Require().NoError(err)
		for _, c := range chunks {
			if out == nil {
				out = make([][]float32, len(c.Planes))
			}
			for i, pl := range c.Planes {
				out[i] = append(out[i], pl...)
			}
		}
	}
	for {
		pkt, err := d.NextPacket()
		if errors.Is(err, ErrEOF) {
			break
		}
		s.Require().NoError(err)
		s.Require().NoError(dec.Send(pkt))
		collect(DrainPCM(dec))
	}
	s.Require().NoError(dec.Send(Packet{}))
	collect(DrainPCM(dec))
	return out
}

// noisePCM is n stereo samples of fixed-seed noise on the 16-bit grid.
func noisePCM(n int) codec.PCM {
	planes := [][]float32{make([]float32, n), make([]float32, n)}
	seed := uint32(1)
	for i := 0; i < n; i++ {
		for c := range planes {
			seed = seed*1664525 + 1013904223
			planes[c][i] = float32(int16(seed>>16)/4) / 32768
		}
	}
	return codec.PCM{
		Planes: planes, NbSamples: n, Channels: 2,
		SampleRate: streamRate, Format: codec.SampleFmtFLTP,
	}
}

func slicePCM(p codec.PCM, off, n int) codec.PCM {
	planes := make([][]float32, len(p.Planes))
	for i, pl := range p.Planes {
		planes[i] = pl[off : off+n]
	}
	p.Planes, p.NbSamples, p.PTS = planes, n, p.PTS+int64(off)
	return p
}

// An *os.File always has a Seek method, but a pipe cannot seek. A muxer
// that must go back to finish its file has to refuse it, not write a file
// nothing can open.
func (s *StreamSuite) TestAPipeIsNotTakenForAWriterThatSeeks() {
	f, err := FindFormat("m4a", "alac")
	if err != nil {
		s.T().Skipf("this FFmpeg cannot write ALAC in m4a: %v", err)
	}
	enc, err := NewEncoder(f.Info(streamRate, 2, AudioInfo{SampleFmt: codec.SampleFmtS16, Bits: 16}))
	s.Require().NoError(err)
	defer func() { _ = enc.Close() }()
	r, w, err := os.Pipe()
	s.Require().NoError(err)
	defer func() { _ = r.Close() }()
	go func() { _, _ = io.Copy(io.Discard, r) }()

	m, err := NewMuxer(w, enc.Info())
	if err == nil {
		err = m.WriteHeader()
		_ = m.Close()
	}
	_ = w.Close()

	s.Error(err)
}

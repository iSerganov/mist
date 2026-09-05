package av

import (
	"bytes"
	"encoding/binary"
	"errors"
	"io"
	"math"
	"os"
	"path/filepath"
	"testing"

	"github.com/iSerganov/mist/internal/codec"
	"github.com/stretchr/testify/suite"
)

type AVSuite struct {
	suite.Suite
}

func TestAVSuite(t *testing.T) {
	suite.Run(t, &AVSuite{})
}

func (s *AVSuite) requireLibav() {
	if !Available() {
		s.T().Skip("libav Vorbis encoder not available")
	}
}

func (s *AVSuite) TestAudioInfoParamsVorbis() {
	info := AudioInfo{
		CodecID:    CodecIDVorbis,
		SampleRate: 44100,
		Channels:   2,
		SampleFmt:  codec.SampleFmtFLTP,
		Bitrate:    128_000,
		Extradata:  []byte{1, 2, 3},
	}
	p := info.Params()
	s.Equal(codec.IDVorbis, p.ID)
	s.Equal(44100, p.SampleRate)
	s.Equal(2, p.Channels)
	s.Equal([]byte{1, 2, 3}, p.Extradata)
}

func (s *AVSuite) TestAudioInfoParamsOther() {
	tests := []struct {
		id   int
		want codec.ID
	}{
		{CodecIDPCM, codec.IDPCM},
		{CodecIDMP3, codec.IDMP3},
		{CodecIDAAC, codec.IDAAC},
		{CodecIDNone, codec.IDNone},
	}
	for _, tc := range tests {
		s.Equal(tc.want, AudioInfo{CodecID: tc.id}.Params().ID)
	}
}

func (s *AVSuite) TestInit() {
	s.requireLibav()
	s.NoError(Init())
	s.NoError(Init())
}

func (s *AVSuite) TestOpenDemuxer() {
	s.requireLibav()
	raw, info := s.encodeOgg()
	path := filepath.Join(s.T().TempDir(), "sine.ogg")
	s.Require().NoError(os.WriteFile(path, raw, 0o600))

	d, err := OpenDemuxer(path)
	s.Require().NoError(err)
	defer d.Close()
	got := d.Info()
	s.Equal(CodecIDVorbis, got.CodecID)
	s.Equal(info.SampleRate, got.SampleRate)
	s.Equal(info.Channels, got.Channels)
	s.NotEmpty(got.Extradata)
}

func (s *AVSuite) TestDemuxMuxRoundTrip() {
	s.requireLibav()
	raw, info := s.encodeOgg()

	src := bytes.NewReader(raw)
	d, err := OpenDemuxerReader(src)
	s.Require().NoError(err)
	defer d.Close()

	var pkts []Packet
	for {
		pkt, err := d.NextPacket()
		if errors.Is(err, ErrEOF) {
			break
		}
		s.Require().NoError(err)
		pkts = append(pkts, pkt)
	}
	s.Require().NotEmpty(pkts)

	var out rwBuf
	m, err := NewMuxer(&out, info)
	s.Require().NoError(err)
	s.Require().NoError(m.WriteHeader())
	for _, pkt := range pkts {
		s.Require().NoError(m.WritePacket(pkt))
	}
	s.Require().NoError(m.WriteTrailer())
	s.Require().NoError(m.Close())

	d2, err := OpenDemuxerReader(bytes.NewReader(out.data))
	s.Require().NoError(err)
	defer d2.Close()
	var again []Packet
	for {
		pkt, err := d2.NextPacket()
		if errors.Is(err, ErrEOF) {
			break
		}
		s.Require().NoError(err)
		again = append(again, pkt)
	}
	s.Require().Len(again, len(pkts))
	for i := range pkts {
		s.Equal(pkts[i].Data, again[i].Data)
	}
}

func (s *AVSuite) TestEncodeDecodePCM() {
	s.requireLibav()
	enc, err := NewEncoder(AudioInfo{
		CodecID:    CodecIDVorbis,
		SampleRate: 44100,
		Channels:   2,
		SampleFmt:  codec.SampleFmtFLTP,
		Bitrate:    64_000,
	})
	s.Require().NoError(err)
	defer enc.Close()
	info := enc.Info()
	s.NotEmpty(info.Extradata)

	in := sineFrame(44100, 2, 8192, 440)
	pkts := s.collectPackets(enc, in, shiftPTS(in, 8192), shiftPTS(in, 16384))
	s.Require().NotEmpty(pkts)

	dec, err := NewDecoder(info)
	s.Require().NoError(err)
	defer dec.Close()
	var frames []Frame
	for _, pkt := range pkts {
		s.NoError(dec.Send(pkt))
		frames = append(frames, s.recvFrames(dec)...)
	}
	s.NoError(dec.Send(Packet{}))
	frames = append(frames, s.recvFrames(dec)...)
	s.Require().NotEmpty(frames)
	s.Equal(2, frames[0].Channels)
	s.Equal(44100, frames[0].SampleRate)
	s.Greater(frameEnergy(frames[0]), 1.0)
}

func (s *AVSuite) TestCustomIO() {
	s.requireLibav()
	raw, info := s.encodeOgg()
	d, err := OpenDemuxerReader(bytes.NewReader(raw))
	s.Require().NoError(err)
	defer d.Close()
	s.Equal(info.SampleRate, d.Info().SampleRate)
	pkt, err := d.NextPacket()
	s.Require().NoError(err)
	s.NotEmpty(pkt.Data)
}

func (s *AVSuite) encodeOgg() ([]byte, AudioInfo) {
	s.T().Helper()
	enc, err := NewEncoder(AudioInfo{
		CodecID:    CodecIDVorbis,
		SampleRate: 44100,
		Channels:   2,
		SampleFmt:  codec.SampleFmtFLTP,
		Bitrate:    64_000,
	})
	s.Require().NoError(err)
	defer enc.Close()
	info := enc.Info()

	var buf rwBuf
	m, err := NewMuxer(&buf, info)
	s.Require().NoError(err)
	s.Require().NoError(m.WriteHeader())
	in := sineFrame(44100, 2, 8192, 440)
	for _, pkt := range s.collectPackets(enc, in, shiftPTS(in, 8192)) {
		s.Require().NoError(m.WritePacket(pkt))
	}
	s.Require().NoError(m.WriteTrailer())
	s.Require().NoError(m.Close())
	s.Require().NotEmpty(buf.data)
	return buf.data, info
}

func (s *AVSuite) collectPackets(enc *Encoder, frames ...Frame) []Packet {
	s.T().Helper()
	var pkts []Packet
	fs := enc.Info().FrameSize
	if fs <= 0 {
		fs = 64
	}
	for _, f := range frames {
		for _, chunk := range splitFrame(f, fs) {
			s.Require().NoError(enc.Send(chunk))
			pkts = append(pkts, s.recvPackets(enc)...)
		}
	}
	s.Require().NoError(enc.Send(Frame{}))
	pkts = append(pkts, s.recvPackets(enc)...)
	return pkts
}

func (s *AVSuite) recvPackets(enc *Encoder) []Packet {
	s.T().Helper()
	var pkts []Packet
	for {
		pkt, err := enc.Receive()
		if errors.Is(err, ErrAgain) || errors.Is(err, ErrEOF) {
			return pkts
		}
		s.Require().NoError(err)
		pkts = append(pkts, pkt)
	}
}

func (s *AVSuite) recvFrames(dec *Decoder) []Frame {
	s.T().Helper()
	var frames []Frame
	for {
		fr, err := dec.Receive()
		if errors.Is(err, ErrAgain) || errors.Is(err, ErrEOF) {
			return frames
		}
		s.Require().NoError(err)
		frames = append(frames, fr)
	}
}

func shiftPTS(f Frame, pts int64) Frame {
	f.PTS = pts
	return f
}

func splitFrame(f Frame, fs int) []Frame {
	planes := frameFloatPlanes(f)
	var out []Frame
	for off := 0; off < f.NbSamples; off += fs {
		data := make([][]byte, len(planes))
		for i, p := range planes {
			sl := make([]float32, fs)
			end := off + fs
			if end > len(p) {
				end = len(p)
			}
			if off < len(p) {
				copy(sl, p[off:end])
			}
			data[i] = floatsToLE(sl)
		}
		out = append(out, Frame{
			Data:       data,
			NbSamples:  fs,
			Channels:   f.Channels,
			SampleRate: f.SampleRate,
			Format:     codec.SampleFmtFLTP,
			PTS:        f.PTS + int64(off),
		})
	}
	return out
}

func sineFrame(rate, ch, n int, freq float64) Frame {
	data := make([][]byte, ch)
	for c := 0; c < ch; c++ {
		samples := make([]float32, n)
		for i := 0; i < n; i++ {
			samples[i] = float32(math.Sin(2 * math.Pi * freq * float64(i) / float64(rate)))
		}
		data[c] = floatsToLE(samples)
	}
	return Frame{
		Data:       data,
		NbSamples:  n,
		Channels:   ch,
		SampleRate: rate,
		Format:     codec.SampleFmtFLTP,
	}
}

func floatsToLE(in []float32) []byte {
	out := make([]byte, len(in)*4)
	for i, v := range in {
		binary.LittleEndian.PutUint32(out[i*4:], math.Float32bits(v))
	}
	return out
}

func frameEnergy(f Frame) float64 {
	var e float64
	for _, plane := range f.Data {
		for i := 0; i+4 <= len(plane); i += 4 {
			v := math.Float32frombits(binary.LittleEndian.Uint32(plane[i:]))
			e += float64(v * v)
		}
	}
	return e
}

// rwBuf is a seekable memory buffer. Ogg mux/demux both seek
// (probing, granule patch-up), so bytes.Buffer is not enough.
type rwBuf struct {
	data []byte
	off  int
}

func (b *rwBuf) Write(p []byte) (int, error) {
	end := b.off + len(p)
	if end > len(b.data) {
		b.data = append(b.data, make([]byte, end-len(b.data))...)
	}
	copy(b.data[b.off:], p)
	b.off += len(p)
	return len(p), nil
}

func (b *rwBuf) Read(p []byte) (int, error) {
	if b.off >= len(b.data) {
		return 0, io.EOF
	}
	n := copy(p, b.data[b.off:])
	b.off += n
	return n, nil
}

func (b *rwBuf) Seek(offset int64, whence int) (int64, error) {
	var n int64
	switch whence {
	case io.SeekStart:
		n = offset
	case io.SeekCurrent:
		n = int64(b.off) + offset
	case io.SeekEnd:
		n = int64(len(b.data)) + offset
	default:
		return 0, errors.New("whence")
	}
	if n < 0 {
		return 0, errors.New("negative seek")
	}
	b.off = int(n)
	return n, nil
}

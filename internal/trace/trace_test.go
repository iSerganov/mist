package trace

import (
	"bytes"
	"encoding/binary"
	"testing"

	"github.com/stretchr/testify/suite"
)

type TraceSuite struct {
	suite.Suite
}

func TestTraceSuite(t *testing.T) {
	suite.Run(t, &TraceSuite{})
}

func (s *TraceSuite) TestOggSummaryIgnoresTheSerialValue() {
	body := bytes.Repeat([]byte{1}, 8)
	a := append(oggPage(1, 100, body), oggPage(1, 200, body)...)
	b := append(oggPage(99, 100, body), oggPage(99, 200, body)...)
	var left, right Trace
	summarizeOgg(a, &left)
	summarizeOgg(b, &right)
	s.Equal(1, left.OggSerials)
	s.Equal(uint64(200), left.OggGranule)
	s.Empty(Diff(left, right))
	right.OggGranule = 201
	s.Equal([]string{"ogg granule"}, Diff(left, right))
}

func (s *TraceSuite) TestFlacStreamInfoReadsSamplesAndMD5() {
	present := flacFile(48000, []byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16})
	absent := flacFile(48000, make([]byte, 16))
	var withMD5, without Trace
	summarizeFlac(present, &withMD5)
	summarizeFlac(absent, &without)
	s.True(withMD5.Flac)
	s.Equal(int64(48000), withMD5.FlacSamples)
	s.True(withMD5.FlacMD5)
	s.False(without.FlacMD5)
	s.Equal([]string{"flac md5"}, Diff(withMD5, without))
}

func (s *TraceSuite) TestDiff() {
	base := Trace{Samples: 10, SampleFmt: "s16", NominalKbps: 100, Encoder: "Lavf"}
	tests := []struct {
		title string
		edit  func(*Trace)
		want  []string
	}{
		{"identical traces have no difference", func(*Trace) {}, nil},
		{"zero tail is a difference", func(t *Trace) { t.ZeroTail = 4 }, []string{"zero tail"}},
		{"nominal rate within slack is the same setting", func(t *Trace) { t.NominalKbps = 104 }, nil},
		{"nominal rate past slack is a difference", func(t *Trace) { t.NominalKbps = 80 }, []string{"nominal bitrate"}},
		{"file size is never a difference", func(t *Trace) { t.Bytes = 9000 }, nil},
		{"another encoder tag is a difference", func(t *Trace) { t.Encoder = "reference libFLAC" }, []string{"encoder tag"}},
	}
	for _, tc := range tests {
		s.Run(tc.title, func() {
			other := base
			tc.edit(&other)
			s.Equal(tc.want, Diff(base, other))
		})
	}
}

func (s *TraceSuite) TestZeroTailCountsSilenceOnEveryChannel() {
	tests := []struct {
		title  string
		planes [][]float32
		want   int
	}{
		{"no planes", nil, 0},
		{"no trailing silence", [][]float32{{1, 0, 1}}, 0},
		{"silence on one channel only", [][]float32{{1, 0, 0}, {1, 1, 1}}, 0},
		{"silence on every channel", [][]float32{{1, 0, 0}, {1, 1, 0}}, 1},
		{"all silent", [][]float32{{0, 0}}, 2},
	}
	for _, tc := range tests {
		s.Run(tc.title, func() {
			s.Equal(tc.want, zeroTail(tc.planes))
		})
	}
}

func (s *TraceSuite) TestExtFollowsMagic() {
	tests := []struct {
		title string
		head  string
		want  string
	}{
		{"ogg", "OggS....", ".ogg"},
		{"flac", "fLaC....", ".flac"},
		{"wav", "RIFF....", ".wav"},
		{"aiff", "FORM....", ".aiff"},
		{"unknown", "ID3.....", ".bin"},
	}
	for _, tc := range tests {
		s.Run(tc.title, func() {
			s.Equal(tc.want, Ext([]byte(tc.head)))
		})
	}
}

func oggPage(serial uint32, granule uint64, body []byte) []byte {
	page := make([]byte, 28+len(body))
	copy(page, "OggS")
	binary.LittleEndian.PutUint64(page[6:14], granule)
	binary.LittleEndian.PutUint32(page[14:18], serial)
	page[26] = 1
	page[27] = byte(len(body))
	copy(page[28:], body)
	return page
}

func flacFile(samples uint64, md5 []byte) []byte {
	info := make([]byte, 34)
	bit := 0
	writeBits(info, &bit, 4096, 16)
	writeBits(info, &bit, 4096, 16)
	writeBits(info, &bit, 0, 24)
	writeBits(info, &bit, 0, 24)
	writeBits(info, &bit, 44100, 20)
	writeBits(info, &bit, 1, 3)
	writeBits(info, &bit, 15, 5)
	writeBits(info, &bit, samples, 36)
	copy(info[18:34], md5)
	block := []byte{0x80, 0x00, 0x00, 34}
	return append(append([]byte("fLaC"), block...), info...)
}

func writeBits(dst []byte, bit *int, v uint64, n int) {
	for i := n - 1; i >= 0; i-- {
		if v&(1<<uint(i)) != 0 {
			dst[*bit/8] |= 1 << (7 - uint(*bit%8))
		}
		*bit++
	}
}

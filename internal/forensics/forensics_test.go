package forensics

import (
	"bytes"
	"encoding/base64"
	"encoding/binary"
	"encoding/hex"
	"math/rand/v2"
	"strings"
	"testing"

	"github.com/stretchr/testify/suite"
)

type ForensicsSuite struct {
	suite.Suite
}

func TestForensicsSuite(t *testing.T) {
	suite.Run(t, &ForensicsSuite{})
}

const frameSize = 417 // MPEG-1 Layer III, 128 kbps, 44.1 kHz, no padding

// mp3 is n silent MPEG-1 Layer III frames; edit may change frame i.
func mp3(n int, edit func(i int, f []byte)) []byte {
	var out []byte
	for i := range n {
		f := make([]byte, frameSize)
		copy(f, []byte{0xFF, 0xFB, 0x90, 0x04})
		if edit != nil {
			edit(i, f)
		}
		out = append(out, f...)
	}
	return out
}

func random(n int, seed uint64) []byte {
	r := rand.New(rand.NewPCG(seed, 1))
	b := make([]byte, n)
	for i := range b {
		b[i] = byte(r.Uint32())
	}
	return b
}

func id3Tag(frames ...[]byte) []byte {
	body := bytes.Join(frames, nil)
	size := len(body)
	return append([]byte{'I', 'D', '3', 3, 0, 0, byte(size >> 21 & 0x7f), byte(size >> 14 & 0x7f), byte(size >> 7 & 0x7f), byte(size & 0x7f)}, body...)
}

func id3Raw(id string, body []byte) []byte {
	h := make([]byte, 10)
	copy(h, id)
	binary.BigEndian.PutUint32(h[4:], uint32(len(body)))
	return append(h, body...)
}

// jpeg is a minimal baseline JPEG whose scan data holds a stuffed 0xFF and
// a restart marker, so only a real segment walk finds its end.
func jpeg() []byte {
	return []byte{
		0xFF, 0xD8,
		0xFF, 0xE0, 0x00, 0x04, 'J', 'F',
		0xFF, 0xDA, 0x00, 0x02,
		0x12, 0xFF, 0x00, 0x34, 0xFF, 0xD0, 0x56,
		0xFF, 0xD9,
	}
}

func riff(chunks ...[]byte) []byte {
	body := append([]byte("WAVE"), bytes.Join(chunks, nil)...)
	h := []byte("RIFF\x00\x00\x00\x00")
	binary.LittleEndian.PutUint32(h[4:], uint32(len(body)))
	return append(h, body...)
}

func riffChunk(id string, body []byte) []byte {
	h := make([]byte, 8)
	copy(h, id)
	binary.LittleEndian.PutUint32(h[4:], uint32(len(body)))
	out := append(h, body...)
	if len(body)%2 == 1 {
		out = append(out, 0)
	}
	return out
}

func flac(blocks ...[]byte) []byte {
	out := []byte("fLaC")
	for i, b := range blocks {
		hdr := uint32(len(b)-1) & 0xffffff
		hdr |= uint32(b[0]) << 24
		if i == len(blocks)-1 {
			hdr |= 0x80000000
		}
		h := make([]byte, 4)
		binary.BigEndian.PutUint32(h, hdr)
		out = append(append(out, h...), b[1:]...)
	}
	return out
}

func flacBlock(typ byte, body []byte) []byte { return append([]byte{typ}, body...) }

func flacPictureBlock(img []byte) []byte {
	var b bytes.Buffer
	u32 := func(v int) { _ = binary.Write(&b, binary.BigEndian, uint32(v)) }
	u32(3)
	u32(10)
	b.WriteString("image/jpeg")
	u32(0)
	b.Write(make([]byte, 16))
	u32(len(img))
	b.Write(img)
	return flacBlock(6, b.Bytes())
}

func oggPage(body []byte) []byte {
	p := make([]byte, 28+len(body))
	copy(p, "OggS")
	binary.LittleEndian.PutUint32(p[14:], 7)
	p[26] = 1
	p[27] = byte(len(body))
	copy(p[28:], body)
	return p
}

func mp4(atoms ...[]byte) []byte {
	return bytes.Join(append([][]byte{mp4Atom("ftyp", []byte("M4A \x00\x00\x00\x00"))}, atoms...), nil)
}

func mp4Atom(id string, body []byte) []byte {
	h := make([]byte, 8)
	binary.BigEndian.PutUint32(h, uint32(8+len(body)))
	copy(h[4:], id)
	return append(h, body...)
}

func (s *ForensicsSuite) TestContainer() {
	id3v1 := append([]byte("TAG"), make([]byte, 125)...)
	tests := []struct {
		title string
		file  []byte
		want  Level
		says  string
	}{
		{"a plain mp3", mp3(40, nil), None, ""},
		{"an mp3 with an id3v1 tag", append(mp3(40, nil), id3v1...), None, ""},
		{"bytes appended to an mp3", append(mp3(40, nil), random(500, 1)...), High, "after the last audio frame"},
		{"bytes appended after the id3v1 tag", append(append(mp3(40, nil), id3v1...), random(500, 2)...), High, "after the last audio frame"},
		{"a cut-off last frame", mp3(40, nil)[:40*frameSize-100], Low, "cut short"},
		{"junk between frames", append(append(mp3(20, nil), random(2000, 3)...), mp3(20, nil)...), High, "between audio frames"},
		{"cover art with a file appended", append(id3Tag(id3Raw("APIC", append([]byte("\x00image/jpeg\x00\x03\x00"), append(jpeg(), random(300, 4)...)...))), mp3(40, nil)...), High, "after the end of the image"},
		{"clean cover art", append(id3Tag(id3Raw("APIC", append([]byte("\x00image/jpeg\x00\x03\x00"), jpeg()...))), mp3(40, nil)...), None, ""},
		{"id3 padding that is not empty", append(id3Tag(id3Raw("TIT2", []byte("\x00song")), append([]byte{0}, random(200, 5)...)), mp3(40, nil)...), High, "padding"},
		{"base64 hidden in a comment", append(id3Tag(id3Raw("TXXX", []byte("\x00note\x00"+base64.StdEncoding.EncodeToString(random(120, 6))))), mp3(40, nil)...), Medium, "encoded binary"},
		{"a large encrypted private frame", append(id3Tag(id3Raw("PRIV", append([]byte("x\x00"), random(8000, 7)...))), mp3(40, nil)...), Medium, "PRIV"},
		{"a plain wav", riff(riffChunk("fmt ", make([]byte, 16)), riffChunk("data", make([]byte, 100))), None, ""},
		{"an unknown wav chunk", riff(riffChunk("fmt ", make([]byte, 16)), riffChunk("hide", random(400, 8)), riffChunk("data", make([]byte, 100))), Medium, `"hide"`},
		{"wav junk that is not empty", riff(riffChunk("JUNK", random(64, 9)), riffChunk("data", make([]byte, 100))), High, "padding chunk"},
		{"bytes after the wav", append(riff(riffChunk("data", make([]byte, 100))), random(64, 10)...), High, "after the end of the WAV"},
		{"empty flac padding", flac(flacBlock(0, make([]byte, 34)), flacBlock(1, make([]byte, 100))), None, ""},
		{"flac padding with data", flac(flacBlock(0, make([]byte, 34)), flacBlock(1, random(100, 11))), High, "FLAC padding"},
		{"flac cover art with a file appended", flac(flacBlock(0, make([]byte, 34)), flacPictureBlock(append(jpeg(), random(300, 12)...))), High, "after the end of the image"},
		{"a plain ogg", append(oggPage([]byte("abc")), oggPage([]byte("def"))...), None, ""},
		{"bytes after the last ogg page", append(oggPage([]byte("abc")), random(64, 13)...), High, "last Ogg page"},
		{"a plain m4a", mp4(mp4Atom("moov", make([]byte, 20)), mp4Atom("mdat", random(100, 14))), None, ""},
		{"an unknown m4a atom", mp4(mp4Atom("moov", make([]byte, 20)), mp4Atom("zzzz", random(100, 15))), Medium, `"zzzz"`},
		{"bytes after the m4a atoms", append(mp4(mp4Atom("mdat", random(100, 16))), random(64, 17)...), High, "after the end of the MP4"},
	}
	for _, tc := range tests {
		s.Run(tc.title, func() {
			c := Container(tc.file)
			s.True(c.Ran)
			s.Equal(tc.want, c.Level(), "%+v", c.Findings)
			if tc.says != "" {
				var all []string
				for _, f := range c.Findings {
					all = append(all, f.Detail)
				}
				s.Contains(strings.Join(all, "\n"), tc.says)
			}
		})
	}
}

func (s *ForensicsSuite) TestContainerSkipsUnknownFormats() {
	s.False(Container([]byte("plain text, not audio")).Ran)
}

func (s *ForensicsSuite) TestMP3Frames() {
	tests := []struct {
		title string
		file  []byte
		check string
		want  Level
	}{
		{"constant header bits", mp3(40, nil), "mp3 header bits", None},
		{"a private bit that carries data", mp3(40, func(i int, f []byte) { f[2] |= byte(i % 2) }), "mp3 header bits", High},
		{"one odd copyright bit", mp3(40, func(i int, f []byte) {
			if i == 7 {
				f[3] |= 0x08
			}
		}), "mp3 header bits", Low},
		{"empty ancillary data", mp3(40, nil), "mp3 ancillary", None},
		{"an encoder signature and padding", mp3(40, func(i int, f []byte) {
			copy(f[36:], "LAME3.100UUUUUUUU")
			if i%3 == 0 {
				copy(f[200:], "LAMEUUUU")
			}
		}), "mp3 ancillary", None},
		{"encrypted ancillary data", mp3(40, func(i int, f []byte) { copy(f[36:], random(frameSize-36, uint64(i))) }), "mp3 ancillary", High},
		{"granule lengths are context only", mp3(40, nil), "mp3 granules", None},
	}
	for _, tc := range tests {
		s.Run(tc.title, func() {
			for _, c := range MP3(tc.file) {
				if c.Name == tc.check {
					s.True(c.Ran)
					s.Equal(tc.want, c.Level(), "%+v", c.Findings)
					return
				}
			}
			s.Fail("no check named " + tc.check)
		})
	}
}

func (s *ForensicsSuite) TestSilence() {
	zeros := func(n int) []int32 { return make([]int32, n) }
	sprinkled := zeros(48000)
	for i := 100; i < len(sprinkled); i += 97 {
		sprinkled[i] = int32(1 - 2*(i%2))
	}
	dither := zeros(48000)
	r := rand.New(rand.NewPCG(1, 2))
	for i := range dither {
		dither[i] = int32(r.IntN(3)) - 1
	}
	fade := zeros(48000)
	for i := 0; i < 4000; i++ {
		if i%200 < 60 {
			fade[i] = 1
		} else if i%200 >= 100 && i%200 < 160 {
			fade[i] = -1
		}
	}
	tests := []struct {
		title  string
		planes [][]int32
		want   Level
	}{
		{"pure digital silence", [][]int32{zeros(48000)}, None},
		{"isolated ±1 in silence", [][]int32{sprinkled}, High},
		{"dense dither", [][]int32{dither}, None},
		{"the last cycles of a fade", [][]int32{fade}, None},
		{"no silence at all", [][]int32{func() []int32 {
			v := zeros(48000)
			for i := range v {
				v[i] = int32(i%200) - 100
			}
			return v
		}()}, None},
	}
	for _, tc := range tests {
		s.Run(tc.title, func() {
			s.Equal(tc.want, Silence(tc.planes, 48000).Level())
		})
	}
}

func (s *ForensicsSuite) TestEncodedText() {
	tests := []struct {
		title string
		text  string
		want  bool
	}{
		{"base64 of random bytes", base64.StdEncoding.EncodeToString(random(90, 20)), true},
		{"hex of random bytes", hex.EncodeToString(random(40, 21)), true},
		{"prose", strings.Repeat("the quick brown fox jumps over the lazy dog ", 3), false},
		{"a short id", "0123456789abcdef", false},
		{"one letter repeated", strings.Repeat("a", 100), false},
	}
	for _, tc := range tests {
		s.Run(tc.title, func() {
			s.Equal(tc.want, encodedText(tc.text))
		})
	}
}

func (s *ForensicsSuite) TestImageTail() {
	tests := []struct {
		title string
		img   []byte
		want  int
	}{
		{"a whole jpeg", jpeg(), 0},
		{"a jpeg with data after it", append(jpeg(), 1, 2, 3), 3},
		{"a jpeg with another jpeg after it", append(jpeg(), jpeg()...), len(jpeg())},
		{"not an image", []byte("hello"), -1},
	}
	for _, tc := range tests {
		s.Run(tc.title, func() {
			s.Equal(tc.want, imageTail(tc.img))
		})
	}
}

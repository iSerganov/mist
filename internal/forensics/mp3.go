package forensics

import (
	"bytes"
	"fmt"
	"strings"
)

var (
	mp3Bitrates = [2][3][16]int{
		{ // MPEG-1: layer I, II, III
			{0, 32, 64, 96, 128, 160, 192, 224, 256, 288, 320, 352, 384, 416, 448},
			{0, 32, 48, 56, 64, 80, 96, 112, 128, 160, 192, 224, 256, 320, 384},
			{0, 32, 40, 48, 56, 64, 80, 96, 112, 128, 160, 192, 224, 256, 320},
		},
		{ // MPEG-2 and 2.5
			{0, 32, 48, 56, 64, 80, 96, 112, 128, 144, 160, 176, 192, 224, 256},
			{0, 8, 16, 24, 32, 40, 48, 56, 64, 80, 96, 112, 128, 144, 160},
			{0, 8, 16, 24, 32, 40, 48, 56, 64, 80, 96, 112, 128, 144, 160},
		},
	}
	mp3Rates = [4][3]int{{11025, 12000, 8000}, {}, {22050, 24000, 16000}, {44100, 48000, 32000}}
)

// mp3Frame is one MPEG audio frame header, with the Layer III side
// information Mist's checks read.
type mp3Frame struct {
	off, size     int
	mpeg1, layer3 bool
	mono          bool
	crc           bool
	private       bool
	copyright     bool
	original      bool
	emphasis      int
	mainBegin     int
	granuleBits   []int
	info          bool
}

// parseMP3Frame reads the frame at off, or reports false when off does not
// start a valid frame that fits in b.
func parseMP3Frame(b []byte, off int) (mp3Frame, bool) {
	f, ok := parseMP3Header(b, off)
	if !ok || off+f.size > len(b) {
		return mp3Frame{}, false
	}
	if f.layer3 {
		f.sideInfo(b[off:])
	}
	return f, true
}

// parseMP3Header reads the four header bytes at off and the frame size
// they announce, whether or not the frame fits in b.
func parseMP3Header(b []byte, off int) (mp3Frame, bool) {
	if off+4 > len(b) || b[off] != 0xFF || b[off+1]&0xE0 != 0xE0 {
		return mp3Frame{}, false
	}
	h := uint32(b[off])<<24 | uint32(b[off+1])<<16 | uint32(b[off+2])<<8 | uint32(b[off+3])
	version := int(h>>19) & 3
	layer := 4 - int(h>>17)&3
	bitrateIdx := int(h>>12) & 15
	rateIdx := int(h>>10) & 3
	if version == 1 || layer == 4 || bitrateIdx == 0 || bitrateIdx == 15 || rateIdx == 3 {
		return mp3Frame{}, false
	}
	f := mp3Frame{
		off:       off,
		mpeg1:     version == 3,
		layer3:    layer == 3,
		crc:       h>>16&1 == 0,
		private:   h>>8&1 == 1,
		mono:      h>>6&3 == 3,
		copyright: h>>3&1 == 1,
		original:  h>>2&1 == 1,
		emphasis:  int(h & 3),
	}
	table := 1
	if f.mpeg1 {
		table = 0
	}
	bitrate := mp3Bitrates[table][layer-1][bitrateIdx] * 1000
	rate := mp3Rates[version][rateIdx]
	pad := int(h>>9) & 1
	switch {
	case layer == 1:
		f.size = (12*bitrate/rate + pad) * 4
	case layer == 3 && !f.mpeg1:
		f.size = 72*bitrate/rate + pad
	default:
		f.size = 144*bitrate/rate + pad
	}
	return f, f.size >= 4
}

// sideInfoSize is the length of the Layer III side information.
func (f mp3Frame) sideInfoSize() int {
	switch {
	case f.mpeg1 && f.mono:
		return 17
	case f.mpeg1:
		return 32
	case f.mono:
		return 9
	}
	return 17
}

// mainData is where the frame's main-data bytes start.
func (f mp3Frame) mainData() int {
	n := 4 + f.sideInfoSize()
	if f.crc {
		n += 2
	}
	return n
}

// sideInfo reads main_data_begin and every granule's part2_3_length, and
// marks a Xing, Info or VBRI frame, which carries a header and no audio.
func (f *mp3Frame) sideInfo(frame []byte) {
	start := 4
	if f.crc {
		start += 2
	}
	if start+f.sideInfoSize() > len(frame) {
		return
	}
	r := bitReader{b: frame[start : start+f.sideInfoSize()]}
	channels := 2
	if f.mono {
		channels = 1
	}
	granules := 1
	if f.mpeg1 {
		granules = 2
		f.mainBegin = r.read(9)
		if f.mono {
			r.read(5)
		} else {
			r.read(3)
		}
		r.read(4 * channels)
	} else {
		f.mainBegin = r.read(8)
		r.read(channels)
	}
	for range granules {
		for range channels {
			f.granuleBits = append(f.granuleBits, r.read(12))
			if f.mpeg1 {
				r.read(47)
			} else {
				r.read(51)
			}
		}
	}
	body := frame[f.mainData():min(len(frame), f.mainData()+64)]
	f.info = bytes.Contains(body, []byte("Xing")) || bytes.Contains(body, []byte("Info")) ||
		bytes.Contains(frame[:min(len(frame), 64)], []byte("VBRI"))
}

type bitReader struct {
	b   []byte
	bit int
}

func (r *bitReader) read(n int) int {
	v := 0
	for range n {
		if r.bit/8 >= len(r.b) {
			return v
		}
		v = v<<1 | int(r.b[r.bit/8]>>(7-r.bit%8)&1)
		r.bit++
	}
	return v
}

func mp3Sync(b []byte, off int) bool {
	_, ok := parseMP3Frame(b, off)
	return ok
}

// mp3Stream is an MP3 file split into its parts: the leading ID3v2 tag,
// every frame, the bytes skipped between frames, and what follows the last.
type mp3Stream struct {
	tag     []byte
	frames  []mp3Frame
	skipped int
	tail    []byte
}

// splitMP3 walks the frames. A frame only counts when the next one follows
// it directly, or nothing but end tags does, so a stray sync pattern inside
// other data is not taken for audio.
func splitMP3(b []byte) mp3Stream {
	var s mp3Stream
	off := leadingTags(b)
	s.tag = b[:off]
	end := off
	for off < len(b) {
		if f, ok := parseMP3Frame(b, off); ok && confirmed(b, off+f.size) {
			s.skipped += off - end
			s.frames = append(s.frames, f)
			off, end = off+f.size, off+f.size
			continue
		}
		limit := len(b)
		if len(s.frames) > 0 {
			limit = min(len(b), off+maxJunk)
		}
		if off = nextFrame(b, off+1, limit); off < 0 {
			break
		}
	}
	s.tail = b[end:]
	return s
}

// maxJunk is how far past a broken frame the walk looks for audio to
// resume before it takes the rest of the file as trailing data.
const maxJunk = 1 << 16

// confirmed reports that a frame ending at next is followed by another
// frame, by a final frame the file cuts short, or by nothing but fill
// bytes and end tags.
func confirmed(b []byte, next int) bool {
	if next == len(b) || mp3Sync(b, next) || truncatedFrame(b[next:]) {
		return true
	}
	rest := b[next : len(b)-trailingTags(b[next:])]
	for _, x := range rest {
		if x != 0 && x != 0xFF {
			return false
		}
	}
	return true
}

// nextFrame is the first confirmed frame in b[from:limit], or -1.
func nextFrame(b []byte, from, limit int) int {
	for i := from; i < limit; i++ {
		if b[i] != 0xFF {
			continue
		}
		if f, ok := parseMP3Frame(b, i); ok && confirmed(b, i+f.size) {
			return i
		}
	}
	return -1
}

func mp3Container(b []byte, c *Check) {
	s := splitMP3(b)
	id3v2Tags(s.tag, c)
	if len(s.frames) == 0 {
		c.add(Medium, "no MPEG audio frames found")
		return
	}
	if s.skipped > 0 {
		l := Low
		if s.skipped >= 1024 {
			l = High
		}
		c.add(l, fmt.Sprintf("%s between audio frames that belong to no frame", plural(s.skipped, "byte")))
	}
	tail := s.tail[:len(s.tail)-trailingTags(s.tail)]
	if truncatedFrame(tail) {
		c.add(Low, fmt.Sprintf("the last frame is cut short after %s", plural(len(tail), "byte")))
		return
	}
	trailing(tail, "after the last audio frame that no tag format accounts for", c)
}

// truncatedFrame reports a tail that is one frame header and less than the
// frame it announces: a cut-off download, not hidden data.
func truncatedFrame(tail []byte) bool {
	f, ok := parseMP3Header(tail, 0)
	return ok && len(tail) < f.size
}

// MP3 checks the MPEG audio frames themselves for covert channels: header
// bits encoders keep constant that change from frame to frame, ancillary
// data between one frame's audio and the next, and the granule lengths
// MP3Stego hides its bits in.
func MP3(b []byte) []Check {
	header := Check{Name: "mp3 header bits"}
	ancillary := Check{Name: "mp3 ancillary"}
	granules := Check{Name: "mp3 granules", Informational: true}
	if !mp3Sync(b, leadingTags(b)) {
		return []Check{header, ancillary, granules}
	}
	s := splitMP3(b)
	var audio []mp3Frame
	for _, f := range s.frames {
		if !f.info {
			audio = append(audio, f)
		}
	}
	if len(audio) < 16 {
		return []Check{header, ancillary, granules}
	}
	header.Ran = true
	headerBits(audio, &header)
	if audio[0].layer3 {
		ancillary.Ran, granules.Ran = true, true
		ancillaryData(b, audio, &ancillary)
		granuleLengths(audio, len(s.frames) > len(audio), &granules)
	}
	return []Check{header, ancillary, granules}
}

// headerBits counts frames whose private, copyright, original or emphasis
// field differs from the file's usual value. An encoder sets these once
// for the whole file; flipping them carries one bit per frame.
func headerBits(frames []mp3Frame, c *Check) {
	fields := []struct {
		name string
		get  func(mp3Frame) int
	}{
		{"private", func(f mp3Frame) int { return b2i(f.private) }},
		{"copyright", func(f mp3Frame) int { return b2i(f.copyright) }},
		{"original", func(f mp3Frame) int { return b2i(f.original) }},
		{"emphasis", func(f mp3Frame) int { return f.emphasis }},
	}
	for _, fd := range fields {
		counts := map[int]int{}
		for _, f := range frames {
			counts[fd.get(f)]++
		}
		usual := 0
		for _, n := range counts {
			usual = max(usual, n)
		}
		changed := len(frames) - usual
		switch {
		case changed >= 8:
			c.add(High, fmt.Sprintf("the %s bit changes in %d of %d frames; encoders set it once per file, so it can carry one bit per frame",
				fd.name, changed, len(frames)))
		case changed > 0:
			c.add(Low, fmt.Sprintf("the %s bit changes in %d of %d frames", fd.name, changed, len(frames)))
		}
	}
}

func b2i(v bool) int {
	if v {
		return 1
	}
	return 0
}

// ancillaryData gathers the bits left between one frame's main data and
// the next frame's. Encoders leave them zero, or write their own name and
// pad with 'U' (LAME writes "LAME3.100UUUU…" at whatever bit the audio
// ended on, sometimes twice). What is left after that is data no decoder
// reads.
func ancillaryData(b []byte, frames []mp3Frame, c *Check) {
	var stream []byte
	starts := make([]int, len(frames))
	for i, f := range frames {
		starts[i] = len(stream) - f.mainBegin
		stream = append(stream, b[f.off+f.mainData():f.off+f.size]...)
	}
	var gaps [][2]int
	total := 0
	for i, f := range frames {
		used := 0
		for _, bits := range f.granuleBits {
			used += bits
		}
		from := starts[i]*8 + used
		to := len(stream) * 8
		if i+1 < len(frames) {
			to = starts[i+1] * 8
		}
		if from < 0 || from >= to || to > len(stream)*8 {
			continue
		}
		gaps = append(gaps, [2]int{from, to})
		total += to - from
	}
	signature := encoderSignature(stream, gaps)
	var payload []byte
	for _, g := range gaps {
		payload = append(payload, uncovered(stream, g[0], g[1], signature)...)
	}
	c.Context = fmt.Sprintf("%s of ancillary data, %s of it neither zero nor encoder padding",
		plural(total/8, "byte"), plural(len(payload), "byte"))
	switch {
	case opaque(payload) && len(payload) >= minAncillary:
		c.add(High, fmt.Sprintf("%s of ancillary data at %.1f bits per byte, like encrypted data", plural(len(payload), "byte"), entropy(payload)))
	case len(payload) >= 4*minAncillary:
		c.add(Medium, fmt.Sprintf("%s of ancillary data that is neither zero nor encoder padding", plural(len(payload), "byte")))
	}
}

// encoderSignature is the text the encoder writes into its ancillary data:
// the printable run of at least four characters, read at every bit
// alignment, that covers the most bytes in total (count times length):
// the encoder's text repeats and is long, while the same text read at the
// wrong alignment gives short fragments that may also be printable. It is
// extended to the longest frequent run that starts with it, since a short
// gap cuts the signature off. Nil when there is none.
func encoderSignature(stream []byte, gaps [][2]int) []byte {
	counts := map[string]int{}
	for _, g := range gaps[:min(len(gaps), 2000)] {
		for shift := range 8 {
			if g[0]+shift >= g[1] {
				break
			}
			bs := bitSlice(stream, g[0]+shift, g[1])
			run := 0
			for k := 0; k <= len(bs); k++ {
				if k < len(bs) && bs[k] >= 0x20 && bs[k] <= 0x7e && bs[k] != 'U' {
					run++
					continue
				}
				if run >= 4 {
					counts[string(bs[k-run:k])]++
				}
				run = 0
			}
		}
	}
	best, score := "", 0
	for s, k := range counts {
		if k >= 4 && (k*len(s) > score || k*len(s) == score && len(s) > len(best)) {
			best, score = s, k*len(s)
		}
	}
	if best == "" {
		return nil
	}
	whole := best
	for s, k := range counts {
		if k >= 4 && len(s) > len(whole) && strings.HasPrefix(s, best) {
			whole = s
		}
	}
	return []byte(whole)
}

// minAncillary is the least ancillary payload, in bytes, that counts.
// Encoders that do not clear their bit buffers leave a few stray bytes of
// noise across thousands of frames; a hidden message is that size at least.
const minAncillary = 1024

// minPayloadBits is the shortest stretch of uncovered bits that counts.
// A gap starts and ends on any bit, so up to a byte at each edge can never
// line up with a whole padding byte; a payload leaves long stretches.
const minPayloadBits = 16

// uncovered is the bits of stream[from:to] that are neither the encoder's
// signature (whole, cut to three bytes or more, or cut by the gap's end)
// nor padding (0x00, 'U', or 'U' one bit over) at any bit
// alignment, in stretches of at least minPayloadBits, packed into bytes.
func uncovered(stream []byte, from, to int, signature []byte) []byte {
	covered := make([]bool, to-from)
	for shift := range 8 {
		if from+shift >= to {
			break
		}
		bs := bitSlice(stream, from+shift, to)
		for k := 0; k < len(bs); {
			n := 1
			if p := commonPrefix(bs[k:], signature); p >= 3 || p > 0 && k+p == len(bs) {
				n = p
			} else if bs[k] != 0 && bs[k] != 'U' && bs[k] != 0xAA {
				k++
				continue
			}
			for bit := shift + k*8; bit < shift+(k+n)*8; bit++ {
				covered[bit] = true
			}
			k += n
		}
	}
	var out []byte
	var acc, nbits int
	r := bitReader{b: stream, bit: from}
	for i := 0; i < len(covered); {
		j := i
		for j < len(covered) && covered[j] == covered[i] {
			j++
		}
		if covered[i] || j-i < minPayloadBits {
			r.bit += j - i
			i = j
			continue
		}
		for ; i < j; i++ {
			acc, nbits = acc<<1|r.read(1), nbits+1
			if nbits == 8 {
				out = append(out, byte(acc))
				acc, nbits = 0, 0
			}
		}
	}
	return out
}

// commonPrefix is how many leading bytes a and b share.
func commonPrefix(a, b []byte) int {
	n := 0
	for n < len(a) && n < len(b) && a[n] == b[n] {
		n++
	}
	return n
}

// bitSlice is the bits of b from bit from to bit to, repacked into bytes
// from the first: ancillary data starts wherever the last granule ended,
// which is rarely on a byte boundary.
func bitSlice(b []byte, from, to int) []byte {
	out := make([]byte, (to-from)/8)
	r := bitReader{b: b, bit: from}
	for i := range out {
		out[i] = byte(r.read(8))
	}
	return out
}

// granuleLengths describes part2_3_length, the field MP3Stego hides bits
// in by steering the quantizer until each length has the parity it wants.
// An honest encoder's lengths have random parity too, so without the
// encoder that made the file this is context, never a finding.
func granuleLengths(frames []mp3Frame, tagged bool, c *Check) {
	var odd, n int
	for _, f := range frames {
		for _, bits := range f.granuleBits {
			if bits == 0 {
				continue
			}
			n++
			odd += bits & 1
		}
	}
	if n == 0 {
		return
	}
	c.Context = fmt.Sprintf("%d granules, %.1f%% of lengths odd", n, 100*float64(odd)/float64(n))
	if !tagged {
		c.Context += "; no Xing/Info header, as the reference ISO encoder MP3Stego is built on writes"
	}
}

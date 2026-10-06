package forensics

import (
	"encoding/binary"
	"fmt"
	"strings"
)

// Chunks the WAV, AIFF and MP4 specifications and the common tools that
// write those files define. Anything else is data no player reads.
var (
	riffChunks = set("fmt ", "data", "fact", "LIST", "cue ", "smpl", "inst", "bext", "iXML", "id3 ", "ID3 ",
		"JUNK", "junk", "PAD ", "PEAK", "acid", "cart", "DISP", "afsp", "plst", "wavl", "slnt", "_PMX",
		"umid", "minf", "elm1", "ds64", "regn", "axml", "chna", "mext", "levl", "link", "qlty", "ResU", "FLLR",
		"C2PA", "c2pa", "DGDA", "scot", "ovwf", "LGWV", "SNDM", "MXF ", "RLND", "Fake", "olym")
	aiffChunks = set("COMM", "SSND", "MARK", "INST", "COMT", "NAME", "AUTH", "(c) ", "ANNO", "AESD", "APPL",
		"MIDI", "FVER", "CHAN", "chan", "ID3 ", "id3 ", "basc", "trns", "cate", "FLLR")
	mp4Atoms = set("ftyp", "moov", "mdat", "free", "skip", "wide", "uuid", "meta", "pdin", "moof", "mfra",
		"styp", "sidx", "ssix", "prft", "emsg")
	padding = set("JUNK", "junk", "PAD ", "FLLR", "free", "skip", "wide")
)

func set(names ...string) map[string]bool {
	m := map[string]bool{}
	for _, n := range names {
		m[n] = true
	}
	return m
}

// beyond reports bytes past the end a container declares for itself, less
// any tags players read there.
func beyond(what string, extra []byte, c *Check) {
	trailing(extra[:len(extra)-trailingTags(extra)], "after the end of the "+what, c)
}

// trailing reports bytes no frame or chunk accounts for. A run of one fill
// byte (zeros, 0xFF) is what a disc image or a sloppy writer leaves; any
// other content is data.
func trailing(extra []byte, where string, c *Check) {
	var data int
	for _, x := range extra {
		if x != 0 && x != 0xFF {
			data++
		}
	}
	switch {
	case data >= 16:
		c.add(High, fmt.Sprintf("%s %s%s", plural(len(extra), "byte"), where, entropyNote(extra)))
	case data > 0:
		c.add(Low, fmt.Sprintf("%s %s", plural(len(extra), "byte"), where))
	case len(extra) > 0:
		c.add(Low, fmt.Sprintf("%s of fill (0x00 or 0xFF) %s", plural(len(extra), "byte"), where))
	}
}

// adtsSync reports an AAC ADTS frame header at off: the 12-bit sync and a
// layer of zero, which tells it from an MPEG audio frame.
func adtsSync(b []byte, off int) bool {
	return off+7 <= len(b) && b[off] == 0xFF && b[off+1]&0xF6 == 0xF0
}

func adtsContainer(b []byte, c *Check) {
	off, frames, skipped := 0, 0, 0
	for off < len(b) {
		if size := adtsSize(b, off); size > 0 && adtsConfirmed(b, off+size) {
			off += size
			frames++
			continue
		}
		next := nextADTS(b, off+1)
		if next < 0 {
			break
		}
		skipped += next - off
		off = next
	}
	if frames == 0 {
		c.add(Medium, "no AAC frames found")
		return
	}
	if skipped > 0 {
		l := Low
		if skipped >= 1024 {
			l = High
		}
		c.add(l, fmt.Sprintf("%s between AAC frames that belong to no frame", plural(skipped, "byte")))
	}
	tail := b[off:]
	if cut := adtsCut(tail[:len(tail)-trailingTags(tail)]); cut > 0 {
		c.add(Low, fmt.Sprintf("the last frame is cut short after %s", plural(cut, "byte")))
		return
	}
	beyond("AAC frames", tail, c)
}

// adtsSize is the length of the ADTS frame at off, or 0 when there is none
// that fits in b.
func adtsSize(b []byte, off int) int {
	if !adtsSync(b, off) {
		return 0
	}
	size := int(b[off+3]&3)<<11 | int(b[off+4])<<3 | int(b[off+5])>>5
	if size < 7 || off+size > len(b) {
		return 0
	}
	return size
}

// adtsConfirmed reports that a frame ending at next is followed by another
// frame, by a final frame the file cuts short, or by nothing but fill
// bytes and end tags.
func adtsConfirmed(b []byte, next int) bool {
	rest := b[next : len(b)-trailingTags(b[next:])]
	if len(rest) == 0 || adtsSize(b, next) > 0 || adtsCut(rest) > 0 {
		return true
	}
	for _, x := range rest {
		if x != 0 && x != 0xFF {
			return false
		}
	}
	return true
}

// adtsCut is the length of b when it is one ADTS frame cut short, else 0.
func adtsCut(b []byte) int {
	if !adtsSync(b, 0) || int(b[3]&3)<<11|int(b[4])<<3|int(b[5])>>5 <= len(b) {
		return 0
	}
	return len(b)
}

// nextADTS is the first ADTS frame from from on that another frame or the
// end of the file follows, or -1.
func nextADTS(b []byte, from int) int {
	for i := from; i < len(b); i++ {
		if size := adtsSize(b, i); size > 0 && adtsConfirmed(b, i+size) {
			return i
		}
	}
	return -1
}

// chunk reports one chunk or atom by name: unknown ones, and padding that
// is not empty.
func chunk(kind, id string, body []byte, known map[string]bool, c *Check) {
	switch {
	case (id == "free" || id == "skip" || id == "wide") && (opaque(body) || len(body) >= 1024 && !zero(body)):
		c.add(Medium, fmt.Sprintf("%s %q atom of %s holds data%s", kind, id, plural(len(body), "byte"), entropyNote(body)))
	case id == "free" || id == "skip" || id == "wide":
	case padding[id] && !zero(body):
		c.add(High, fmt.Sprintf("%s padding chunk %q of %s is not empty%s", kind, id, plural(len(body), "byte"), entropyNote(body)))
	case !known[id]:
		l := Medium
		if len(body) < 64 {
			l = Low
		}
		c.add(l, fmt.Sprintf("unknown %s chunk %q of %s%s", kind, printable(id), plural(len(body), "byte"), entropyNote(body)))
	case id == "id3 " || id == "ID3 ":
		id3v2(body, c)
	}
}

func printable(id string) string {
	return strings.Map(func(r rune) rune {
		if r < 0x20 || r > 0x7e {
			return '?'
		}
		return r
	}, id)
}

func riffContainer(b []byte, c *Check) {
	declared := int(binary.LittleEndian.Uint32(b[4:8])) + 8
	if string(b[:4]) == "RF64" || declared > len(b) {
		declared = len(b)
	}
	off := 12
	for off+8 <= declared {
		id := string(b[off : off+4])
		size := int(binary.LittleEndian.Uint32(b[off+4 : off+8]))
		end := min(off+8+size, declared)
		chunk("WAV", id, b[off+8:end], riffChunks, c)
		off = end + size&1
	}
	beyond("WAV data", b[declared:], c)
}

func aiffContainer(b []byte, c *Check) {
	declared := min(int(binary.BigEndian.Uint32(b[4:8]))+8, len(b))
	off := 12
	for off+8 <= declared {
		id := string(b[off : off+4])
		size := int(binary.BigEndian.Uint32(b[off+4 : off+8]))
		end := min(off+8+size, declared)
		chunk("AIFF", id, b[off+8:end], aiffChunks, c)
		off = end + size&1
	}
	beyond("AIFF data", b[declared:], c)
}

func mp4Container(b []byte, c *Check) {
	off := 0
	for off+8 <= len(b) {
		size := int(binary.BigEndian.Uint32(b[off : off+4]))
		id := string(b[off+4 : off+8])
		head := 8
		switch size {
		case 0:
			size = len(b) - off
		case 1:
			if off+16 > len(b) {
				return
			}
			size = int(binary.BigEndian.Uint64(b[off+8 : off+16]))
			head = 16
		}
		if size < head || off+size > len(b) {
			break
		}
		chunk("MP4", id, b[off+head:off+size], mp4Atoms, c)
		off += size
	}
	beyond("MP4 atoms", b[off:], c)
}

func oggContainer(b []byte, c *Check) {
	off := 0
	serials := map[uint32]bool{}
	for off+27 <= len(b) && string(b[off:off+4]) == "OggS" {
		nseg := int(b[off+26])
		if off+27+nseg > len(b) {
			break
		}
		body := 0
		for _, l := range b[off+27 : off+27+nseg] {
			body += int(l)
		}
		if off+27+nseg+body > len(b) {
			break
		}
		serials[binary.LittleEndian.Uint32(b[off+14:off+18])] = true
		off += 27 + nseg + body
	}
	if len(serials) > 1 {
		c.add(Low, fmt.Sprintf("%d logical streams; a chained radio recording has several, a plain encode has one", len(serials)))
	}
	beyond("last Ogg page", b[off:], c)
}

func flacContainer(b []byte, c *Check) {
	off := 4
	for off+4 <= len(b) {
		hdr := binary.BigEndian.Uint32(b[off : off+4])
		last := hdr&0x80000000 != 0
		typ := int(hdr >> 24 & 0x7f)
		size := int(hdr & 0xffffff)
		body := b[off+4 : min(off+4+size, len(b))]
		switch {
		case typ == 1 && !zero(body):
			c.add(High, fmt.Sprintf("FLAC padding of %s is not empty%s", plural(len(body), "byte"), entropyNote(body)))
		case typ == 2:
			c.add(Low, fmt.Sprintf("FLAC application block %q of %s", printable(string(body[:min(4, len(body))])), plural(len(body), "byte")))
		case typ == 6:
			if img := flacPicture(body); img != nil {
				coverArt("FLAC cover art", img, c)
			}
		case typ > 6 && typ < 127:
			c.add(Medium, fmt.Sprintf("FLAC metadata block of unknown type %d, %s", typ, plural(len(body), "byte")))
		}
		off += 4 + size
		if last {
			return
		}
	}
}

// flacPicture is the image data of a PICTURE block.
func flacPicture(body []byte) []byte {
	off := 4
	for range 2 {
		if off+4 > len(body) {
			return nil
		}
		off += 4 + int(binary.BigEndian.Uint32(body[off:]))
	}
	off += 16
	if off+4 > len(body) {
		return nil
	}
	n := int(binary.BigEndian.Uint32(body[off:]))
	return body[off+4 : min(off+4+n, len(body))]
}

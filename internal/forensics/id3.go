package forensics

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"strings"
	"unicode/utf8"
)

// opaqueFrameSize is where a private or general-object ID3 frame stops
// being the small blob players and stores leave and starts to be a payload.
const opaqueFrameSize = 4096

// id3v2Size is the length of the ID3v2 tag at the start of b, header and
// footer included, or 0 when there is none.
func id3v2Size(b []byte) int {
	if len(b) < 10 || string(b[:3]) != "ID3" {
		return 0
	}
	n := 10 + syncsafe(b[6:10])
	if b[5]&0x10 != 0 {
		n += 10
	}
	return min(n, len(b))
}

func syncsafe(b []byte) int {
	return int(b[0]&0x7f)<<21 | int(b[1]&0x7f)<<14 | int(b[2]&0x7f)<<7 | int(b[3]&0x7f)
}

// id3v2Tags checks every ID3v2 tag in a run of them.
func id3v2Tags(b []byte, c *Check) {
	for len(b) > 0 {
		size := id3v2Size(b)
		if size == 0 {
			return
		}
		id3v2(b[:size], c)
		b = b[size:]
	}
}

// id3v2 checks the frames of an ID3v2.3 or 2.4 tag: pictures with data
// appended, private or object frames large enough to be a payload, text
// that reads like encoded binary, and padding that is not zero.
func id3v2(tag []byte, c *Check) {
	if len(tag) < 10 {
		return
	}
	version := tag[3]
	if version < 3 || tag[5]&0x80 != 0 {
		return
	}
	end := len(tag)
	if tag[5]&0x10 != 0 {
		end -= 10
	}
	off := 10
	if tag[5]&0x40 != 0 && off+4 <= end {
		ext := int(binary.BigEndian.Uint32(tag[off:]))
		if version == 4 {
			ext = syncsafe(tag[off : off+4])
		} else {
			ext += 4
		}
		off += ext
	}
	for off+10 <= end {
		if tag[off] == 0 {
			if pad := tag[off:end]; !zero(pad) {
				c.add(High, fmt.Sprintf("ID3 padding of %s is not empty", plural(len(pad), "byte")))
			}
			return
		}
		id := string(tag[off : off+4])
		size := int(binary.BigEndian.Uint32(tag[off+4:]))
		if version == 4 {
			size = syncsafe(tag[off+4 : off+8])
		}
		body := tag[off+10 : min(off+10+size, end)]
		id3Frame(id, body, c)
		off += 10 + size
	}
}

func id3Frame(id string, body []byte, c *Check) {
	switch {
	case id == "APIC":
		if img := apicImage(body); img != nil {
			coverArt("ID3 cover art", img, c)
		}
	case id == "PRIV" || id == "GEOB":
		if len(body) < opaqueFrameSize || mostlyText(body) {
			return
		}
		l := Low
		if opaque(body) {
			l = Medium
		}
		c.add(l, fmt.Sprintf("ID3 %s frame of %s%s", id, plural(len(body), "byte"), entropyNote(body)))
	case id == "TXXX" || id == "COMM" || id == "USLT" || id[0] == 'T':
		if s := id3Text(body); encodedText(s) {
			c.add(Medium, fmt.Sprintf("ID3 %s text of %d characters reads like encoded binary data", id, utf8.RuneCountInString(s)))
		}
	}
}

// mostlyText reports a frame that is text, such as the XMP packet Adobe
// tools store in a PRIV frame: at least 95% printable ASCII or whitespace.
func mostlyText(b []byte) bool {
	n := 0
	for _, x := range b {
		if x >= 0x20 && x <= 0x7e || x == '\n' || x == '\r' || x == '\t' {
			n++
		}
	}
	return n*100 >= len(b)*95
}

// apicImage is the picture data of an APIC frame: after the text encoding,
// the MIME type, the picture type and the description.
func apicImage(body []byte) []byte {
	if len(body) < 4 {
		return nil
	}
	enc := body[0]
	mime := bytes.IndexByte(body[1:], 0)
	if mime < 0 {
		return nil
	}
	rest := body[1+mime+2:]
	if len(rest) == 0 {
		return nil
	}
	if enc == 1 || enc == 2 {
		for i := 0; i+1 < len(rest); i += 2 {
			if rest[i] == 0 && rest[i+1] == 0 {
				return rest[i+2:]
			}
		}
		return nil
	}
	end := bytes.IndexByte(rest, 0)
	if end < 0 {
		return nil
	}
	return rest[end+1:]
}

// id3Text is a text frame's value as plain bytes, good enough to measure:
// UTF-16 loses its zero bytes, nothing is decoded properly.
func id3Text(body []byte) string {
	if len(body) < 2 {
		return ""
	}
	return strings.Map(func(r rune) rune {
		if r == 0 || r == 0xFEFF || r == 0xFFFE {
			return -1
		}
		return r
	}, string(body[1:]))
}

// encodedText reports a long value made only of a base64 or hex alphabet
// with near-uniform symbol use, which is what binary data stored as text
// looks like. Prose repeats letters and spaces; encoded data does not.
//
// The thresholds sit below what 64 random symbols measure (about 3.9 bits
// over a hex alphabet, 5.0 over base64), since a short sample never shows
// its alphabet's full entropy, and far above any word.
func encodedText(s string) bool {
	s = strings.TrimSpace(s)
	if len(s) < 64 {
		return false
	}
	hex := true
	for i := 0; i < len(s); i++ {
		ch := s[i]
		if !isBase64(ch) {
			return false
		}
		hex = hex && isHex(ch)
	}
	h := entropy([]byte(s))
	if hex {
		return h >= 3.5
	}
	return h >= 4.6
}

func isBase64(ch byte) bool {
	return isHex(ch) || ch >= 'G' && ch <= 'Z' || ch >= 'g' && ch <= 'z' ||
		ch == '+' || ch == '/' || ch == '=' || ch == '-' || ch == '_'
}

func isHex(ch byte) bool {
	return ch >= '0' && ch <= '9' || ch >= 'a' && ch <= 'f' || ch >= 'A' && ch <= 'F'
}

func entropyNote(b []byte) string {
	if opaque(b) {
		return fmt.Sprintf(", %.1f bits per byte like encrypted data", entropy(b))
	}
	return ""
}

// leadingTags is the length of the ID3v2 tags at the start of b. Streamed
// segments often carry two, one for timed metadata and one for the track.
func leadingTags(b []byte) int {
	n := 0
	for {
		size := id3v2Size(b[n:])
		if size == 0 {
			return n
		}
		n += size
	}
}

// trailingTags is how many bytes at the end of b are tags players read
// there: ID3v1, Lyrics3v2, APEv2 and MusicMatch, in any order they are
// stacked.
func trailingTags(b []byte) int {
	n := 0
	for {
		rest := b[:len(b)-n]
		switch {
		case len(rest) >= 128 && string(rest[len(rest)-128:len(rest)-125]) == "TAG":
			n += 128
		case len(rest) >= 32 && string(rest[len(rest)-32:len(rest)-24]) == "APETAGEX":
			size := int(binary.LittleEndian.Uint32(rest[len(rest)-20:]))
			flags := binary.LittleEndian.Uint32(rest[len(rest)-12:])
			if flags&0x80000000 != 0 {
				size += 32
			}
			if size > len(rest) || size < 32 {
				return n
			}
			n += size
		case len(rest) >= 15 && string(rest[len(rest)-9:]) == "LYRICS200":
			size := 0
			if _, err := fmt.Sscanf(string(rest[len(rest)-15:len(rest)-9]), "%06d", &size); err != nil || size+15 > len(rest) {
				return n
			}
			n += size + 15
		case len(rest) >= 48 && string(rest[len(rest)-48:len(rest)-29]) == "Brava Software Inc.":
			head := bytes.LastIndex(rest, []byte("18273645"))
			if head < 0 {
				return n
			}
			for head > 0 && (head < 16 || !zero(rest[head-16:head])) {
				head--
			}
			n += len(rest) - head
		default:
			return n
		}
	}
}

func plural(n int, unit string) string {
	if n == 1 {
		return "1 " + unit
	}
	return fmt.Sprintf("%d %ss", n, unit)
}

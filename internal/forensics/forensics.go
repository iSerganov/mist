// Package forensics looks for data any steganography tool may have hidden in
// an audio file outside its samples: bytes past the end of the audio, chunks
// and tags no player reads, padding that is not empty, files appended to
// cover art, and the MP3 frame fields known to carry covert bits. It reads
// bytes only. None of it is calibrated: each finding comes with a level and
// the reason for it, never a probability.
package forensics

import (
	"bytes"
	"math"
)

// Level is how strongly a finding points at hidden data.
type Level int

const (
	// None means the check found nothing out of the ordinary.
	None Level = iota
	// Low is unusual but common in honest files.
	Low
	// Medium is rare in honest files.
	Medium
	// High is data a player never reads, stored where an encoder writes none.
	High
)

// String names the level for a report.
func (l Level) String() string {
	return [...]string{"none", "low", "medium", "high"}[l]
}

// Finding is one thing a check saw, with its level.
type Finding struct {
	Level  Level
	Detail string
}

// Check is one named check and what it found. Ran is false when the check
// does not apply to the file's format. An Informational check describes
// the file and never has findings.
type Check struct {
	Name          string
	Ran           bool
	Informational bool
	Findings      []Finding
	Context       string
}

// Level is the strongest finding of the check.
func (c Check) Level() Level {
	l := None
	for _, f := range c.Findings {
		l = max(l, f.Level)
	}
	return l
}

func (c *Check) add(l Level, detail string) {
	c.Findings = append(c.Findings, Finding{Level: l, Detail: detail})
}

// Container checks the file's structure for data outside the audio stream:
// trailing bytes, unknown chunks or atoms, non-empty padding, extra tags.
func Container(b []byte) Check {
	c := Check{Name: "container"}
	audio := b[leadingTags(b):]
	switch {
	case adtsSync(audio, 0):
		c.Ran = true
		id3v2Tags(b[:len(b)-len(audio)], &c)
		adtsContainer(audio, &c)
	case bytes.HasPrefix(b, []byte("ID3")) || mp3Sync(b, 0):
		c.Ran = true
		mp3Container(b, &c)
	case bytes.HasPrefix(b, []byte("OggS")):
		c.Ran = true
		oggContainer(b, &c)
	case bytes.HasPrefix(b, []byte("fLaC")):
		c.Ran = true
		flacContainer(b, &c)
	case len(b) >= 12 && (string(b[:4]) == "RIFF" || string(b[:4]) == "RF64") && string(b[8:12]) == "WAVE":
		c.Ran = true
		riffContainer(b, &c)
	case len(b) >= 12 && string(b[:4]) == "FORM" && (string(b[8:12]) == "AIFF" || string(b[8:12]) == "AIFC"):
		c.Ran = true
		aiffContainer(b, &c)
	case len(b) >= 8 && string(b[4:8]) == "ftyp":
		c.Ran = true
		mp4Container(b, &c)
	}
	return c
}

// entropy is the Shannon entropy of b in bits per byte. Encrypted or
// compressed data sits near 8; text and padding far below.
func entropy(b []byte) float64 {
	if len(b) == 0 {
		return 0
	}
	var n [256]int
	for _, x := range b {
		n[x]++
	}
	var h float64
	for _, k := range n {
		if k == 0 {
			continue
		}
		p := float64(k) / float64(len(b))
		h -= p * math.Log2(p)
	}
	return h
}

// opaque reports whether b looks like encrypted or compressed data: long
// enough to measure, and near the 8 bits per byte random data has.
func opaque(b []byte) bool {
	return len(b) >= 256 && entropy(b) >= 7.2
}

func zero(b []byte) bool {
	for _, x := range b {
		if x != 0 {
			return false
		}
	}
	return true
}

// imageTail is how many bytes follow the end of a JPEG or PNG image, the
// place a file is appended to cover art. It is -1 for anything else.
func imageTail(img []byte) int {
	end := -1
	switch {
	case bytes.HasPrefix(img, []byte{0xFF, 0xD8}):
		end = jpegEnd(img)
	case bytes.HasPrefix(img, []byte("\x89PNG\r\n\x1a\n")):
		end = pngEnd(img)
	}
	if end < 0 {
		return -1
	}
	return len(img) - end
}

// jpegEnd walks the marker segments to the end-of-image marker. Inside
// entropy-coded data a 0xFF byte is always followed by 0x00 or a restart
// marker, so the first other marker there ends the scan.
func jpegEnd(b []byte) int {
	off := 2
	for off+2 <= len(b) {
		if b[off] != 0xFF {
			return -1
		}
		marker := b[off+1]
		switch {
		case marker == 0xD9:
			return off + 2
		case marker == 0xFF:
			off++
			continue
		case marker >= 0xD0 && marker <= 0xD7 || marker == 0x01:
			off += 2
			continue
		}
		if off+4 > len(b) {
			return -1
		}
		size := int(b[off+2])<<8 | int(b[off+3])
		off += 2 + size
		if marker != 0xDA {
			continue
		}
		for off+1 < len(b) && (b[off] != 0xFF || b[off+1] == 0x00 || b[off+1] >= 0xD0 && b[off+1] <= 0xD7) {
			off++
		}
	}
	return -1
}

// pngEnd walks the chunks to the end of IEND.
func pngEnd(b []byte) int {
	off := 8
	for off+12 <= len(b) {
		size := int(b[off])<<24 | int(b[off+1])<<16 | int(b[off+2])<<8 | int(b[off+3])
		typ := string(b[off+4 : off+8])
		off += 12 + size
		if typ == "IEND" {
			return min(off, len(b))
		}
	}
	return -1
}

// coverArt checks one embedded picture for data appended after its end.
func coverArt(where string, img []byte, c *Check) {
	switch tail := imageTail(img); {
	case tail > 16:
		c.add(High, where+": "+plural(tail, "byte")+" after the end of the image")
	case tail > 0:
		c.add(Low, where+": "+plural(tail, "byte")+" after the end of the image")
	}
}

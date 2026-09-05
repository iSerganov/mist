//go:build cgo

package av

// Custom AVIO callbacks. Handles are integers in a Go map so libav never
// stores a Go pointer (cgo forbids that). Read/write/seek are //export
// entry points called from cgo.c; they look up the slot and copy into the
// C buffer. A reader or writer that also implements io.Seeker is required
// for Ogg, because probing and granule patching seek.

/*
#include "cgo.h"
#include <stdint.h>
*/
import "C"

import (
	"io"
	"sync"
	"unsafe"
)

type ioSlot struct {
	r io.Reader
	w io.Writer
}

var (
	ioMu  sync.Mutex
	ioSeq int
	ioTab = map[int]*ioSlot{}
)

func registerIO(r io.Reader, w io.Writer) int {
	ioMu.Lock()
	defer ioMu.Unlock()
	ioSeq++
	ioTab[ioSeq] = &ioSlot{r: r, w: w}
	return ioSeq
}

func forgetIO(id int) {
	ioMu.Lock()
	delete(ioTab, id)
	ioMu.Unlock()
}

func slot(id int) *ioSlot {
	ioMu.Lock()
	defer ioMu.Unlock()
	return ioTab[id]
}

//export mist_av_go_read
func mist_av_go_read(id C.int, buf *C.uint8_t, bufSize C.int) C.int {
	s := slot(int(id))
	if s == nil || s.r == nil || bufSize <= 0 {
		return -1
	}
	p := unsafe.Slice((*byte)(unsafe.Pointer(buf)), int(bufSize))
	n, err := s.r.Read(p)
	if n > 0 {
		return C.int(n)
	}
	if err == io.EOF {
		return 0
	}
	return -1
}

//export mist_av_go_write
func mist_av_go_write(id C.int, buf *C.uint8_t, bufSize C.int) C.int {
	s := slot(int(id))
	if s == nil || s.w == nil || bufSize <= 0 {
		return -1
	}
	p := unsafe.Slice((*byte)(unsafe.Pointer(buf)), int(bufSize))
	n, err := s.w.Write(p)
	if err != nil && n == 0 {
		return -1
	}
	return C.int(n)
}

//export mist_av_go_seek
func mist_av_go_seek(id C.int, offset C.int64_t, whence C.int) C.int64_t {
	s := slot(int(id))
	if s == nil {
		return -1
	}
	var sk io.Seeker
	if s.r != nil {
		sk, _ = s.r.(io.Seeker)
	}
	if sk == nil && s.w != nil {
		sk, _ = s.w.(io.Seeker)
	}
	if sk == nil {
		return -1
	}
	w := int(whence)
	// AVSEEK_SIZE (0x10000): report length without moving the cursor.
	if w&0x10000 != 0 {
		cur, err := sk.Seek(0, io.SeekCurrent)
		if err != nil {
			return -1
		}
		end, err := sk.Seek(0, io.SeekEnd)
		_, _ = sk.Seek(cur, io.SeekStart)
		if err != nil {
			return -1
		}
		return C.int64_t(end)
	}
	n, err := sk.Seek(int64(offset), w&0xffff)
	if err != nil {
		return -1
	}
	return C.int64_t(n)
}

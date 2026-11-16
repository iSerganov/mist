//go:build cgo

package av

// Custom AVIO callbacks. Handles are integers in a Go map so libav never
// stores a Go pointer (cgo forbids that). Read/write/seek are //export
// entry points called from cgo.c; they look up the slot and copy into the
// C buffer. libav is told the stream can seek only when a seek actually
// works. A slot can instead hold a context, which libav's interrupt
// callback polls during blocking network IO.

/*
#include "cgo.h"
#include <stdint.h>
*/
import "C"

import (
	"context"
	"io"
	"sync"
	"unsafe"
)

type ioSlot struct {
	r   io.Reader
	w   io.Writer
	ctx context.Context
}

var (
	ioMu  sync.Mutex
	ioSeq int
	ioTab = map[int]*ioSlot{}
)

func registerIO(r io.Reader, w io.Writer) int {
	return register(&ioSlot{r: r, w: w})
}

func registerInterrupt(ctx context.Context) int {
	return register(&ioSlot{ctx: ctx})
}

func register(s *ioSlot) int {
	ioMu.Lock()
	defer ioMu.Unlock()
	ioSeq++
	ioTab[ioSeq] = s
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

//export mist_av_go_interrupted
func mist_av_go_interrupted(id C.int) C.int {
	s := slot(int(id))
	if s == nil || s.ctx == nil || s.ctx.Err() == nil {
		return 0
	}
	return 1
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

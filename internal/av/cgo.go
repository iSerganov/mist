//go:build cgo

package av

// cgo.go is the Go side of the libav boundary. Every C string and buffer
// is copied; libav never holds a Go pointer. Open paths register an
// integer IO handle, call into cgo.c, then copy packets/frames out with
// C.GoBytes before unref. ErrAgain is the send/receive "need more"
// condition, not a hard failure.

/*
#cgo pkg-config: libavformat libavcodec libavutil
#include "cgo.h"
#include <stdlib.h>
#include <string.h>
*/
import "C"

import (
	"fmt"
	"io"
	"unsafe"
)

const errBufLen = 256

func avInit() error {
	if rc := C.mist_av_init(); rc != C.MIST_AV_OK {
		return fmt.Errorf("%w: init", ErrInit)
	}
	return nil
}

func avOpenDemuxer(url string) (*Demuxer, error) {
	csrc := C.CString(url)
	defer C.free(unsafe.Pointer(csrc))
	errbuf := (*C.char)(C.malloc(errBufLen))
	if errbuf == nil {
		return nil, fmt.Errorf("%w: out of memory", ErrOpen)
	}
	defer C.free(unsafe.Pointer(errbuf))

	ptr := C.mist_av_demuxer_open(csrc, errbuf, errBufLen)
	if ptr == nil {
		return nil, fmt.Errorf("%w: %s", ErrOpen, C.GoString(errbuf))
	}
	d := &Demuxer{handle: unsafe.Pointer(ptr)}
	if err := avDemuxInfo(d); err != nil {
		_ = avDemuxClose(d)
		return nil, err
	}
	return d, nil
}

func avOpenDemuxerReader(r io.Reader) (*Demuxer, error) {
	id := registerIO(r, nil)
	ioh := C.mist_av_io_new(C.int(id), 0)
	if ioh == nil {
		forgetIO(id)
		return nil, fmt.Errorf("%w: io", ErrOpen)
	}
	errbuf := (*C.char)(C.malloc(errBufLen))
	if errbuf == nil {
		C.mist_av_io_free(ioh)
		forgetIO(id)
		return nil, fmt.Errorf("%w: out of memory", ErrOpen)
	}
	defer C.free(unsafe.Pointer(errbuf))
	ptr := C.mist_av_demuxer_open_io(ioh, errbuf, errBufLen)
	if ptr == nil {
		C.mist_av_io_free(ioh)
		forgetIO(id)
		return nil, fmt.Errorf("%w: %s", ErrOpen, C.GoString(errbuf))
	}
	d := &Demuxer{handle: unsafe.Pointer(ptr), closer: ioCloser{id: id, io: unsafe.Pointer(ioh)}}
	if err := avDemuxInfo(d); err != nil {
		_ = avDemuxClose(d)
		return nil, err
	}
	return d, nil
}

func avNewMuxer(w io.Writer, info AudioInfo) (*Muxer, error) {
	id := registerIO(nil, w)
	ioh := C.mist_av_io_new(C.int(id), 1)
	if ioh == nil {
		forgetIO(id)
		return nil, fmt.Errorf("%w: io", ErrOpen)
	}
	cinfo, free := cAudioInfo(info)
	defer free()
	errbuf := (*C.char)(C.malloc(errBufLen))
	if errbuf == nil {
		C.mist_av_io_free(ioh)
		forgetIO(id)
		return nil, fmt.Errorf("%w: out of memory", ErrOpen)
	}
	defer C.free(unsafe.Pointer(errbuf))
	ptr := C.mist_av_muxer_open_io(ioh, &cinfo, errbuf, errBufLen)
	if ptr == nil {
		C.mist_av_io_free(ioh)
		forgetIO(id)
		return nil, fmt.Errorf("%w: %s", ErrOpen, C.GoString(errbuf))
	}
	return &Muxer{
		handle: unsafe.Pointer(ptr),
		info:   info,
		closer: ioCloser{id: id, io: unsafe.Pointer(ioh)},
	}, nil
}

func avNewDecoder(info AudioInfo) (*Decoder, error) {
	cinfo, free := cAudioInfo(info)
	defer free()
	errbuf := (*C.char)(C.malloc(errBufLen))
	if errbuf == nil {
		return nil, fmt.Errorf("%w: out of memory", ErrOpen)
	}
	defer C.free(unsafe.Pointer(errbuf))
	ptr := C.mist_av_decoder_open(&cinfo, errbuf, errBufLen)
	if ptr == nil {
		return nil, fmt.Errorf("%w: %s", ErrOpen, C.GoString(errbuf))
	}
	return &Decoder{handle: unsafe.Pointer(ptr), info: info}, nil
}

func avNewEncoder(info AudioInfo) (*Encoder, error) {
	cinfo, free := cAudioInfo(info)
	defer free()
	errbuf := (*C.char)(C.malloc(errBufLen))
	if errbuf == nil {
		return nil, fmt.Errorf("%w: out of memory", ErrOpen)
	}
	defer C.free(unsafe.Pointer(errbuf))
	ptr := C.mist_av_encoder_open(&cinfo, errbuf, errBufLen)
	if ptr == nil {
		return nil, fmt.Errorf("%w: %s", ErrOpen, C.GoString(errbuf))
	}
	e := &Encoder{handle: unsafe.Pointer(ptr), info: info}
	if err := avEncInfo(e); err != nil {
		_ = avEncClose(e)
		return nil, err
	}
	return e, nil
}

func avEncInfo(e *Encoder) error {
	if e == nil || e.handle == nil {
		return ErrClosed
	}
	var info C.mist_av_audio_info
	if rc := C.mist_av_encoder_info((*C.mist_av_encoder)(e.handle), &info); rc != C.MIST_AV_OK {
		return mapCErr(rc, "encoder info")
	}
	e.info = goAudioInfo(info)
	if info.extradata != nil {
		C.mist_av_free(unsafe.Pointer(info.extradata))
	}
	return nil
}

func avDemuxInfo(d *Demuxer) error {
	if d == nil || d.handle == nil {
		return ErrClosed
	}
	var info C.mist_av_audio_info
	if rc := C.mist_av_demuxer_audio_info((*C.mist_av_demuxer)(d.handle), &info); rc != C.MIST_AV_OK {
		return mapCErr(rc, "audio info")
	}
	d.info = goAudioInfo(info)
	if info.extradata != nil {
		C.mist_av_free(unsafe.Pointer(info.extradata))
	}
	return nil
}

func avDemuxRead(d *Demuxer) (Packet, error) {
	if d == nil || d.handle == nil {
		return Packet{}, ErrClosed
	}
	var pkt C.mist_av_packet
	rc := C.mist_av_demuxer_read((*C.mist_av_demuxer)(d.handle), &pkt)
	if rc != C.MIST_AV_OK {
		return Packet{}, mapCErr(rc, "read")
	}
	out := goPacket(pkt)
	C.mist_av_packet_unref(&pkt)
	return out, nil
}

func avDemuxClose(d *Demuxer) error {
	if d == nil || d.handle == nil {
		return nil
	}
	C.mist_av_demuxer_close((*C.mist_av_demuxer)(d.handle))
	d.handle = nil
	if d.closer != nil {
		return d.closer.Close()
	}
	return nil
}

func avMuxHeader(m *Muxer) error {
	if m == nil || m.handle == nil {
		return ErrClosed
	}
	return mapCErr(C.mist_av_muxer_write_header((*C.mist_av_muxer)(m.handle)), "write header")
}

func avMuxWrite(m *Muxer, pkt Packet) error {
	if m == nil || m.handle == nil {
		return ErrClosed
	}
	cpkt, free := cPacket(pkt)
	defer free()
	return mapCErr(C.mist_av_muxer_write((*C.mist_av_muxer)(m.handle), &cpkt), "write")
}

func avMuxTrailer(m *Muxer) error {
	if m == nil || m.handle == nil {
		return ErrClosed
	}
	return mapCErr(C.mist_av_muxer_write_trailer((*C.mist_av_muxer)(m.handle)), "write trailer")
}

func avMuxClose(m *Muxer) error {
	if m == nil || m.handle == nil {
		return nil
	}
	C.mist_av_muxer_close((*C.mist_av_muxer)(m.handle))
	m.handle = nil
	if m.closer != nil {
		return m.closer.Close()
	}
	return nil
}

func avDecSend(d *Decoder, pkt Packet) error {
	if d == nil || d.handle == nil {
		return ErrClosed
	}
	if len(pkt.Data) == 0 {
		return mapCErr(C.mist_av_decoder_send((*C.mist_av_decoder)(d.handle), nil), "decode flush")
	}
	cpkt, free := cPacket(pkt)
	defer free()
	return mapCErr(C.mist_av_decoder_send((*C.mist_av_decoder)(d.handle), &cpkt), "decode send")
}

func avDecReceive(d *Decoder) (Frame, error) {
	if d == nil || d.handle == nil {
		return Frame{}, ErrClosed
	}
	var fr C.mist_av_frame
	rc := C.mist_av_decoder_receive((*C.mist_av_decoder)(d.handle), &fr)
	if rc != C.MIST_AV_OK {
		return Frame{}, mapCErr(rc, "decode receive")
	}
	out := goFrame(fr)
	C.mist_av_frame_unref(&fr)
	return out, nil
}

func avDecClose(d *Decoder) error {
	if d == nil || d.handle == nil {
		return nil
	}
	C.mist_av_decoder_close((*C.mist_av_decoder)(d.handle))
	d.handle = nil
	return nil
}

func avEncSend(e *Encoder, f Frame) error {
	if e == nil || e.handle == nil {
		return ErrClosed
	}
	if f.NbSamples == 0 && len(f.Data) == 0 {
		return mapCErr(C.mist_av_encoder_flush((*C.mist_av_encoder)(e.handle)), "encode flush")
	}
	planes := frameFloatPlanes(f)
	if len(planes) == 0 {
		return fmt.Errorf("%w: no pcm", ErrWrite)
	}
	arr, n, free := cFloatPlanes(planes)
	if arr == nil {
		return fmt.Errorf("%w: out of memory", ErrWrite)
	}
	defer free()
	rc := C.mist_av_encoder_send_flt(
		(*C.mist_av_encoder)(e.handle),
		(**C.float)(arr),
		C.int(n),
		C.int(f.NbSamples),
		C.int64_t(f.PTS),
	)
	return mapCErr(rc, "encode send")
}

// cFloatPlanes copies planar PCM into C heaps so the send call never
// passes a Go pointer that itself points at Go memory (cgo forbids that).
func cFloatPlanes(planes [][]float32) (unsafe.Pointer, int, func()) {
	n := len(planes)
	arr := C.malloc(C.size_t(n) * C.size_t(unsafe.Sizeof(uintptr(0))))
	if arr == nil {
		return nil, 0, func() {}
	}
	ptrs := unsafe.Slice((**C.float)(arr), n)
	bufs := make([]unsafe.Pointer, n)
	for i, p := range planes {
		sz := C.size_t(len(p)) * 4
		bufs[i] = C.malloc(sz)
		if bufs[i] == nil {
			for _, b := range bufs {
				C.free(b)
			}
			C.free(arr)
			return nil, 0, func() {}
		}
		if len(p) > 0 {
			C.memcpy(bufs[i], unsafe.Pointer(&p[0]), sz)
		}
		ptrs[i] = (*C.float)(bufs[i])
	}
	return arr, n, func() {
		for _, b := range bufs {
			C.free(b)
		}
		C.free(arr)
	}
}

func avEncReceive(e *Encoder) (Packet, error) {
	if e == nil || e.handle == nil {
		return Packet{}, ErrClosed
	}
	var pkt C.mist_av_packet
	rc := C.mist_av_encoder_receive((*C.mist_av_encoder)(e.handle), &pkt)
	if rc != C.MIST_AV_OK {
		return Packet{}, mapCErr(rc, "encode receive")
	}
	out := goPacket(pkt)
	C.mist_av_packet_unref(&pkt)
	return out, nil
}

func avEncClose(e *Encoder) error {
	if e == nil || e.handle == nil {
		return nil
	}
	C.mist_av_encoder_close((*C.mist_av_encoder)(e.handle))
	e.handle = nil
	return nil
}

type ioCloser struct {
	id int
	io unsafe.Pointer
}

func (c ioCloser) Close() error {
	if c.io != nil {
		C.mist_av_io_free((*C.mist_av_io)(c.io))
	}
	forgetIO(c.id)
	return nil
}

func mapCErr(rc C.int, op string) error {
	switch rc {
	case C.MIST_AV_OK:
		return nil
	case C.MIST_AV_EOF:
		return ErrEOF
	case C.MIST_AV_EAGAIN:
		return ErrAgain
	case C.MIST_AV_UNIMPLEMENTED:
		return fmt.Errorf("%w: %s", errUnimplemented, op)
	default:
		return fmt.Errorf("%w: %s", ErrRead, op)
	}
}

func cAudioInfo(a AudioInfo) (C.mist_av_audio_info, func()) {
	out := C.mist_av_audio_info{
		codec_id:    C.int(a.CodecID),
		sample_rate: C.int(a.SampleRate),
		channels:    C.int(a.Channels),
		sample_fmt:  C.int(a.SampleFmt),
		bitrate:     C.int64_t(a.Bitrate),
		duration_us: C.int64_t(a.DurationUs),
	}
	if len(a.Extradata) == 0 {
		return out, func() {}
	}
	p := C.CBytes(a.Extradata)
	out.extradata = (*C.uint8_t)(p)
	out.extradata_size = C.int(len(a.Extradata))
	return out, func() { C.free(p) }
}

func goAudioInfo(a C.mist_av_audio_info) AudioInfo {
	var extra []byte
	if a.extradata != nil && a.extradata_size > 0 {
		extra = C.GoBytes(unsafe.Pointer(a.extradata), a.extradata_size)
	}
	return AudioInfo{
		CodecID:    int(a.codec_id),
		SampleRate: int(a.sample_rate),
		Channels:   int(a.channels),
		SampleFmt:  codecSampleFmt(int(a.sample_fmt)),
		Bitrate:    int64(a.bitrate),
		DurationUs: int64(a.duration_us),
		Extradata:  extra,
		FrameSize:  int(a.frame_size),
	}
}

func cPacket(p Packet) (C.mist_av_packet, func()) {
	var cp C.mist_av_packet
	cp.size = C.int(len(p.Data))
	cp.stream_index = C.int(p.StreamIndex)
	cp.flags = C.int(p.Flags)
	cp.pts = C.int64_t(p.PTS)
	cp.dts = C.int64_t(p.DTS)
	cp.duration = C.int64_t(p.Duration)
	if len(p.Data) == 0 {
		return cp, func() {}
	}
	cp.data = (*C.uint8_t)(C.CBytes(p.Data))
	return cp, func() { C.free(unsafe.Pointer(cp.data)) }
}

func goPacket(p C.mist_av_packet) Packet {
	var data []byte
	if p.data != nil && p.size > 0 {
		data = C.GoBytes(unsafe.Pointer(p.data), p.size)
	}
	return Packet{
		Data:        data,
		StreamIndex: int(p.stream_index),
		Flags:       int(p.flags),
		PTS:         int64(p.pts),
		DTS:         int64(p.dts),
		Duration:    int64(p.duration),
	}
}

func goFrame(f C.mist_av_frame) Frame {
	nch := int(f.channels)
	planes := 1
	if isPlanar(codecSampleFmt(int(f.format))) {
		planes = nch
	}
	if planes > 8 {
		planes = 8
	}
	data := make([][]byte, 0, planes)
	lines := make([]int, 0, planes)
	for i := 0; i < planes; i++ {
		if f.data[i] == nil || f.linesize[i] <= 0 {
			continue
		}
		data = append(data, C.GoBytes(unsafe.Pointer(f.data[i]), f.linesize[i]))
		lines = append(lines, int(f.linesize[i]))
	}
	return Frame{
		Data:       data,
		Linesize:   lines,
		NbSamples:  int(f.nb_samples),
		Channels:   nch,
		SampleRate: int(f.sample_rate),
		Format:     codecSampleFmt(int(f.format)),
		PTS:        int64(f.pts),
	}
}

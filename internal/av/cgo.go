//go:build cgo

package av

/*
#cgo CFLAGS: -I${SRCDIR}
#include "cgo.h"
#include <stdlib.h>
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
		return mapCErr(rc, "init")
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
	_ = r
	return nil, errUnimplemented
}

func avNewMuxer(w io.Writer, info AudioInfo) (*Muxer, error) {
	_ = w
	_ = info
	return nil, errUnimplemented
}

func avNewDecoder(info AudioInfo) (*Decoder, error) {
	cinfo := cAudioInfo(info)
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
	cinfo := cAudioInfo(info)
	errbuf := (*C.char)(C.malloc(errBufLen))
	if errbuf == nil {
		return nil, fmt.Errorf("%w: out of memory", ErrOpen)
	}
	defer C.free(unsafe.Pointer(errbuf))

	ptr := C.mist_av_encoder_open(&cinfo, errbuf, errBufLen)
	if ptr == nil {
		return nil, fmt.Errorf("%w: %s", ErrOpen, C.GoString(errbuf))
	}
	return &Encoder{handle: unsafe.Pointer(ptr), info: info}, nil
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
	_ = f
	return errUnimplemented
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

func mapCErr(rc C.int, op string) error {
	switch rc {
	case C.MIST_AV_OK:
		return nil
	case C.MIST_AV_EOF:
		return ErrEOF
	case C.MIST_AV_UNIMPLEMENTED:
		return fmt.Errorf("%w: %s", errUnimplemented, op)
	default:
		return fmt.Errorf("%w: %s", ErrRead, op)
	}
}

func cAudioInfo(a AudioInfo) C.mist_av_audio_info {
	return C.mist_av_audio_info{
		codec_id:    C.int(a.CodecID),
		sample_rate: C.int(a.SampleRate),
		channels:    C.int(a.Channels),
		sample_fmt:  C.int(a.SampleFmt),
		bitrate:     C.int64_t(a.Bitrate),
		duration_us: C.int64_t(a.DurationUs),
	}
}

func goAudioInfo(a C.mist_av_audio_info) AudioInfo {
	return AudioInfo{
		CodecID:    int(a.codec_id),
		SampleRate: int(a.sample_rate),
		Channels:   int(a.channels),
		SampleFmt:  codecSampleFmt(int(a.sample_fmt)),
		Bitrate:    int64(a.bitrate),
		DurationUs: int64(a.duration_us),
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
	return Frame{
		NbSamples:  int(f.nb_samples),
		Channels:   int(f.channels),
		SampleRate: int(f.sample_rate),
		Format:     codecSampleFmt(int(f.format)),
		PTS:        int64(f.pts),
	}
}

//go:build !cgo

package av

import "io"

func avInit() error { return errUnimplemented }

func avOpenDemuxer(string) (*Demuxer, error) { return nil, errUnimplemented }

func avOpenDemuxerReader(io.Reader) (*Demuxer, error) { return nil, errUnimplemented }

func avNewMuxer(io.Writer, AudioInfo) (*Muxer, error) { return nil, errUnimplemented }

func avNewDecoder(AudioInfo) (*Decoder, error) { return nil, errUnimplemented }

func avNewEncoder(AudioInfo) (*Encoder, error) { return nil, errUnimplemented }

func avDemuxRead(*Demuxer) (Packet, error) { return Packet{}, errUnimplemented }

func avDemuxClose(*Demuxer) error { return nil }

func avMuxHeader(*Muxer) error { return errUnimplemented }

func avMuxWrite(*Muxer, Packet) error { return errUnimplemented }

func avMuxTrailer(*Muxer) error { return errUnimplemented }

func avMuxClose(*Muxer) error { return nil }

func avDecSend(*Decoder, Packet) error { return errUnimplemented }

func avDecReceive(*Decoder) (Frame, error) { return Frame{}, errUnimplemented }

func avDecClose(*Decoder) error { return nil }

func avEncSend(*Encoder, Frame) error { return errUnimplemented }

func avEncReceive(*Encoder) (Packet, error) { return Packet{}, errUnimplemented }

func avEncClose(*Encoder) error { return nil }

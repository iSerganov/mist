package av

import (
	"io"

	"github.com/iSerganov/mist/internal/codec"
)

// Init prepares the libav runtime (network, log level). Safe to call twice.
func Init() error { return avInit() }

// OpenDemuxer opens a local path or URL.
func OpenDemuxer(url string) (*Demuxer, error) { return avOpenDemuxer(url) }

// OpenDemuxerReader opens a demuxer over an arbitrary reader (Embed's carrier).
func OpenDemuxerReader(r io.Reader) (*Demuxer, error) { return avOpenDemuxerReader(r) }

// NewMuxer starts an Ogg muxer writing to w.
func NewMuxer(w io.Writer, info AudioInfo) (*Muxer, error) { return avNewMuxer(w, info) }

// NewDecoder opens a decoder for info.
func NewDecoder(info AudioInfo) (*Decoder, error) { return avNewDecoder(info) }

// NewEncoder opens an encoder for info.
func NewEncoder(info AudioInfo) (*Encoder, error) { return avNewEncoder(info) }

// NextPacket reads the next audio packet. Returns ErrEOF at end of stream.
func (d *Demuxer) NextPacket() (Packet, error) { return avDemuxRead(d) }

// Close releases the demuxer.
func (d *Demuxer) Close() error { return avDemuxClose(d) }

// WriteHeader writes the container header.
func (m *Muxer) WriteHeader() error { return avMuxHeader(m) }

// WritePacket appends one compressed packet.
func (m *Muxer) WritePacket(pkt Packet) error { return avMuxWrite(m, pkt) }

// WriteTrailer finalizes the container.
func (m *Muxer) WriteTrailer() error { return avMuxTrailer(m) }

// Close releases the muxer. It writes the trailer if that has not happened.
func (m *Muxer) Close() error { return avMuxClose(m) }

// Send feeds a compressed packet into the decoder. A zero-value packet flushes.
func (d *Decoder) Send(pkt Packet) error { return avDecSend(d, pkt) }

// Receive returns the next decoded PCM frame. Returns ErrEOF when drained.
func (d *Decoder) Receive() (Frame, error) { return avDecReceive(d) }

// Close releases the decoder.
func (d *Decoder) Close() error { return avDecClose(d) }

// Send feeds a PCM frame into the encoder. A zero-value frame flushes.
func (e *Encoder) Send(f Frame) error { return avEncSend(e, f) }

// Receive returns the next encoded packet. Returns ErrEOF when drained.
func (e *Encoder) Receive() (Packet, error) { return avEncReceive(e) }

// Close releases the encoder.
func (e *Encoder) Close() error { return avEncClose(e) }

// ToCodecPacket copies an av packet into the codec-layer view.
func ToCodecPacket(p Packet) codec.Packet {
	return codec.Packet{
		Data:     p.Data,
		PTS:      p.PTS,
		DTS:      p.DTS,
		Duration: p.Duration,
		Flags:    p.Flags,
	}
}

// FromCodecPacket copies a codec-layer packet into an av packet.
func FromCodecPacket(p codec.Packet) Packet {
	return Packet{
		Data:     p.Data,
		PTS:      p.PTS,
		DTS:      p.DTS,
		Duration: p.Duration,
		Flags:    p.Flags,
	}
}

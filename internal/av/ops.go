package av

import (
	"io"

	"github.com/iSerganov/mist/internal/codec"
)

// Init prepares the libav runtime (network, log level). Safe to call twice.
func Init() error { return avInit() }

// Available reports whether a Vorbis encoder can be opened. Tests skip when
// the tree is built with CGO_ENABLED=0 or when libav is present but the
// libvorbis encoder is not linked. A true result also means Init succeeded.
func Available() bool {
	if err := Init(); err != nil {
		return false
	}
	enc, err := NewEncoder(AudioInfo{
		CodecID:    CodecIDVorbis,
		SampleRate: 44100,
		Channels:   2,
		SampleFmt:  codec.SampleFmtFLTP,
		Bitrate:    64_000,
	})
	if err != nil {
		return false
	}
	_ = enc.Close()
	return true
}

func ensureInit() error {
	return Init()
}

// OpenDemuxer opens a local path or URL.
func OpenDemuxer(url string) (*Demuxer, error) {
	if err := ensureInit(); err != nil {
		return nil, err
	}
	return avOpenDemuxer(url)
}

// OpenDemuxerReader opens a demuxer over an arbitrary reader (Embed's carrier).
// The reader should implement io.Seeker; Ogg probing seeks.
func OpenDemuxerReader(r io.Reader) (*Demuxer, error) {
	if err := ensureInit(); err != nil {
		return nil, err
	}
	return avOpenDemuxerReader(r)
}

// NewMuxer starts an Ogg muxer writing to w.
// info.Extradata must already hold the Vorbis setup headers from an Encoder.
// w should implement io.Seeker so the muxer can patch granule positions.
func NewMuxer(w io.Writer, info AudioInfo) (*Muxer, error) {
	if err := ensureInit(); err != nil {
		return nil, err
	}
	return avNewMuxer(w, info)
}

// NewDecoder opens a decoder for info.
func NewDecoder(info AudioInfo) (*Decoder, error) {
	if err := ensureInit(); err != nil {
		return nil, err
	}
	return avNewDecoder(info)
}

// NewEncoder opens an encoder for info.
func NewEncoder(info AudioInfo) (*Encoder, error) {
	if err := ensureInit(); err != nil {
		return nil, err
	}
	return avNewEncoder(info)
}

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
// libvorbis requires exactly Info().FrameSize samples (64); larger
// windows must be split and drained via Receive between sends.
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

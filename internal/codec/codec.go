// Package codec is the boundary between PCM and a compressed bitstream.
//
// Phase 1 ships Ogg Vorbis only. WAV/PCM and MP3/AAC plug in behind
// Codec without touching crypto, wire framing, or stego position selection.
package codec

// ID identifies a compressed audio codec.
type ID int

const (
	IDNone ID = iota
	IDVorbis
	IDPCM // Phase 3
	IDMP3 // Phase 3
	IDAAC // Phase 3
)

// Params describes an audio stream.
// Extradata is the codec private blob the decoder needs at open
// (Vorbis identification/comment/setup in Xiph lacing). Encoders
// fill it after a successful open; callers pass it back to NewDecoder.
type Params struct {
	ID         ID
	SampleRate int
	Channels   int
	Bitrate    int64
	Format     SampleFormat
	Extradata  []byte
}

// SampleFormat is a PCM sample representation. Numeric values match
// libavutil's AVSampleFormat so cgo conversion is a cast.
type SampleFormat int

const (
	SampleFmtNone SampleFormat = -1
	SampleFmtU8   SampleFormat = 0
	SampleFmtS16  SampleFormat = 1
	SampleFmtS32  SampleFormat = 2
	SampleFmtFLT  SampleFormat = 3
	SampleFmtDBL  SampleFormat = 4
	SampleFmtU8P  SampleFormat = 5
	SampleFmtS16P SampleFormat = 6
	SampleFmtS32P SampleFormat = 7
	SampleFmtFLTP SampleFormat = 8
	SampleFmtDBLP SampleFormat = 9
)

// DefaultVorbis is the Phase 1 encode target.
var DefaultVorbis = Params{
	ID:         IDVorbis,
	SampleRate: 44100,
	Channels:   2,
	Bitrate:    128_000,
	Format:     SampleFmtFLTP,
}

// PCM is a window of decoded (or yet-to-be-encoded) audio.
type PCM struct {
	Samples    []float32 // interleaved if Format is packed; plane 0 if planar
	Planes     [][]float32
	NbSamples  int
	Channels   int
	SampleRate int
	Format     SampleFormat
	PTS        int64
}

// Packet is one compressed audio packet, codec payload only — no Ogg
// page wrapper. Stego bits live in here, never in container metadata.
type Packet struct {
	Data     []byte
	PTS      int64
	DTS      int64
	Duration int64
	Flags    int
}

// Residue is one quantized MDCT residue coefficient.
type Residue struct {
	Channel int
	Band    int
	Index   int
	Value   int32
}

// Codec converts between PCM and a compressed bitstream and, when the
// format allows it, exposes quantized coefficients for embedding.
type Codec interface {
	Name() string
	ID() ID
	NewEncoder(p Params) (Encoder, error)
	NewDecoder(p Params) (Decoder, error)
	Residues(pkt Packet) ([]Residue, error)
}

// Encoder turns PCM windows into compressed packets.
type Encoder interface {
	Encode(pcm PCM) ([]Packet, error)
	Flush() ([]Packet, error)
	Close() error
}

// Decoder turns compressed packets into PCM.
type Decoder interface {
	Decode(pkt Packet) ([]PCM, error)
	Close() error
}

// Muxer writes compressed packets into a container (Ogg for Phase 1).
type Muxer interface {
	Write(pkt Packet) error
	Close() error
}

// Demuxer reads compressed packets from a container.
type Demuxer interface {
	Next() (Packet, error)
	Params() Params
	Close() error
}

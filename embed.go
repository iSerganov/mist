package mist

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net/url"
	"os"

	"github.com/iSerganov/mist/internal/av"
	"github.com/iSerganov/mist/internal/codec"
	"github.com/iSerganov/mist/internal/codec/vorbis"
	"github.com/iSerganov/mist/internal/crypto"
	"github.com/iSerganov/mist/internal/frame"
	"github.com/iSerganov/mist/internal/stego"
	"github.com/iSerganov/mist/internal/wire"
)

// FrameCapacity returns the maximum payload bytes embeddable per stego
// frame at the library's fixed embedding density, before encryption
// overhead. Useful for callers who need to know the maximum message size
// supported without multi-frame spanning (Phase 2).
func FrameCapacity() int {
	return frame.Capacity(typicalEligible(), stego.Density, EnvelopeOverhead)
}

func typicalEligible() int {
	const rate, hop, n = 44100, 1024, 1024
	packets := int(FrameDuration.Seconds() * float64(rate) / hop)
	bins := 0
	for i := 0; i < n; i++ {
		hz := i * rate / (n * 2)
		if hz >= stego.DefaultBands.FromHz && hz < stego.DefaultBands.ToHz {
			bins++
		}
	}
	return bins * 2 * packets
}

func (e *Emitter) embed(ctx context.Context, r io.Reader, payload Payload) (io.ReadCloser, error) {
	if err := e.validate(ctx, payload); err != nil {
		return nil, err
	}
	pcm, params, err := decodeCarrier(r)
	if err != nil {
		return nil, err
	}
	return e.embedPCM(ctx, pcm, params, payload)
}

// embedURL decodes carrier directly via libav's own URL handling (path,
// file://, or http(s)://) instead of tunneling bytes through a Go
// io.Reader — the only way to reach a plain http(s) source, since it is
// not itself an io.Reader the caller hands us.
func (e *Emitter) embedURL(ctx context.Context, source string, payload Payload) (io.ReadCloser, error) {
	if err := e.validate(ctx, payload); err != nil {
		return nil, err
	}
	pcm, params, err := decodeCarrierURL(source)
	if err != nil {
		return nil, err
	}
	return e.embedPCM(ctx, pcm, params, payload)
}

// validate checks the arguments common to every embed entry point before
// the (expensive) carrier decode runs.
func (e *Emitter) validate(ctx context.Context, payload Payload) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if len(e.pub) != PublicKeySize {
		return ErrInvalidKey
	}
	if payload.Data == nil && payload.Type == 0 {
		return ErrInvalidPayload
	}
	return nil
}

// embedPCM opens the encoder for the Emitter's target and embeds through
// whichever domain that target allows. A lossless codec hands its samples
// back bit for bit, so the payload goes into PCM before the encoder runs;
// Vorbis does not, so it goes into the residues the encoder produced. The
// frame layout, the density and the crypto are the same either way.
func (e *Emitter) embedPCM(ctx context.Context, pcm codec.PCM, params codec.Params, payload Payload) (io.ReadCloser, error) {
	plain, err := marshalPlain(payload, e.senderPriv)
	if err != nil {
		return nil, err
	}
	enc, err := av.NewEncoder(e.target.Info(pcm.SampleRate, pcm.Channels, targetBitrate(e.target, params)))
	if err != nil {
		return nil, fmt.Errorf("%w: encoder: %v", ErrCarrier, err)
	}
	defer func() { _ = enc.Close() }()

	if e.target.Lossless {
		pcm = padToWindow(pcm, enc.Window())
		if err := e.embedSamples(ctx, pcm, av.SampleScale(enc.Info().SampleFmt), plain); err != nil {
			return nil, err
		}
		return encodeAndMux(enc, pcm, nil)
	}
	return encodeAndMux(enc, pcm, func(pkts []codec.Packet) ([]codec.Packet, error) {
		return e.embedResidues(ctx, enc.Info().Extradata, pkts, frameParams(pcm), plain)
	})
}

// encodeAndMux runs the carrier through the encoder and writes the result.
// rewrite, when set, gets every packet before muxing — the hook the Vorbis
// path embeds in, and the one a lossless path has no use for because its
// bits were already in the PCM.
func encodeAndMux(enc *av.Encoder, pcm codec.PCM, rewrite func([]codec.Packet) ([]codec.Packet, error)) (io.ReadCloser, error) {
	pkts, err := enc.Encode(pcm)
	if err != nil {
		return nil, err
	}
	flushed, err := enc.Flush()
	if err != nil {
		return nil, err
	}
	all := append(pkts, flushed...)
	if rewrite != nil {
		if all, err = rewrite(all); err != nil {
			return nil, err
		}
	}
	return muxPackets(enc.Info(), all)
}

// embedResidues is the Vorbis path: group the encoder's packets into stego
// frames and rewrite residue LSBs in each.
func (e *Emitter) embedResidues(ctx context.Context, extradata []byte, pkts []codec.Packet, fp frame.Params, plain []byte) ([]codec.Packet, error) {
	vc := vorbis.New()
	if err := vc.Load(extradata); err != nil {
		return nil, fmt.Errorf("%w: setup", err)
	}
	groups := frame.GroupPackets(pkts, fp)
	if len(groups) == 0 {
		return nil, ErrCarrier
	}
	all := make([]codec.Packet, 0, len(pkts))
	carried := false
	for _, g := range groups {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		out, done, err := e.embedGroup(vc, g, plain, !carried)
		if err != nil {
			return nil, err
		}
		carried = carried || done
		all = append(all, out...)
	}
	if !carried {
		return nil, noCapacity(plain)
	}
	return all, nil
}

// embedGroup writes one frame and reports whether the message landed in it.
// The message rides in the first frame with room for it; every other frame
// is written with CSPRNG filler at the same density, so a frame carrying
// the payload and a frame carrying nothing leave the same footprint. Only
// a frame with no eligible residues at all — a trailing sliver of encoder
// padding — passes through untouched.
func (e *Emitter) embedGroup(vc *vorbis.Codec, g frame.Group, plain []byte, carry bool) ([]codec.Packet, bool, error) {
	room, err := stego.Capacity(vc, g.Packets)
	if errors.Is(err, stego.ErrNoResidues) {
		return g.Packets, false, nil
	}
	if err != nil {
		return nil, false, fmt.Errorf("%w: %v", ErrCarrier, err)
	}

	// Nil bits make Apply fill the whole frame with CSPRNG filler.
	var bits stego.Bits
	carrying := carry && room >= EnvelopeOverhead+len(plain)
	if carrying {
		env, err := crypto.Seal(plain, e.pub)
		if err != nil {
			return nil, false, err
		}
		bits = env.Marshal()
	}
	pos, err := crypto.PositionSeed(e.pub, g.Index)
	if err != nil {
		return nil, false, err
	}
	out, err := stego.Apply(vc, pos, g.Packets, bits)
	// A group whose eligible residues fall below the density floor for even
	// one bit is left alone, the same as one with none at all — carrying is
	// already false here, since room >= EnvelopeOverhead+len(plain) implies
	// at least one slot.
	if errors.Is(err, stego.ErrNoResidues) {
		return g.Packets, false, nil
	}
	if errors.Is(err, stego.ErrCapacity) {
		return nil, false, fmt.Errorf("%w: %v", ErrInvalidPayload, err)
	}
	if err != nil {
		return nil, false, fmt.Errorf("%w: %v", ErrCarrier, err)
	}
	return out, carrying, nil
}

// Bitrate bounds for a lossy re-encode.
const (
	minBitrate     = 192_000
	maxBitrate     = 500_000
	unknownBitrate = 256_000
)

// targetBitrate picks the encode rate for a carrier of the given params.
// A lossless encoder decides its own size from the audio, and forcing a
// rate on it only makes it complain, so it is asked for none.
//
// A lossy target is given headroom above the source instead of matching
// it: Mist always re-encodes, and a second pass at the source's own rate
// compounds the loss. A lossless source reports its raw PCM rate — 1411
// kbps for CD audio — which is no target at all, so it is capped.
func targetBitrate(f av.Format, params codec.Params) int64 {
	switch {
	case f.Lossless:
		return 0
	case params.Bitrate <= 0:
		return unknownBitrate
	default:
		return min(max(params.Bitrate*3/2, minBitrate), maxBitrate)
	}
}

// noCapacity reports that no frame had room, the one embed failure that
// must be loud: the caller would otherwise get a silent no-op file.
func noCapacity(plain []byte) error {
	return fmt.Errorf("%w: %d bytes needed", ErrNoCapacity, len(plain)+EnvelopeOverhead)
}

func frameParams(pcm codec.PCM) frame.Params {
	return frame.Params{
		SampleRate: pcm.SampleRate,
		Channels:   pcm.Channels,
		Duration:   FrameDuration,
	}
}

func decodeCarrier(r io.Reader) (codec.PCM, codec.Params, error) {
	d, err := av.OpenDemuxerReader(asSeeker(r))
	if err != nil {
		return codec.PCM{}, codec.Params{}, fmt.Errorf("%w: %v", ErrCarrier, err)
	}
	defer func() { _ = d.Close() }()
	return decodeFromDemuxer(d)
}

// decodeCarrierURL decodes a carrier libav can open natively — a plain
// path, a file:// URL, or an http(s):// URL — without tunneling it
// through a Go io.Reader. This is the only route into a plain http(s)
// source, since it does not arrive as an io.Reader in the first place.
func decodeCarrierURL(source string) (codec.PCM, codec.Params, error) {
	d, err := av.OpenDemuxer(source)
	if err != nil {
		return codec.PCM{}, codec.Params{}, fmt.Errorf("%w: %v", ErrCarrier, err)
	}
	defer func() { _ = d.Close() }()
	return decodeFromDemuxer(d)
}

func decodeFromDemuxer(d *av.Demuxer) (codec.PCM, codec.Params, error) {
	info := d.Info()
	if !av.CanDecode(info) {
		return codec.PCM{}, codec.Params{},
			fmt.Errorf("%w: this FFmpeg build cannot decode %s", ErrUnsupportedCodec, codecName(info))
	}
	dec, err := av.NewDecoder(info)
	if err != nil {
		return codec.PCM{}, codec.Params{}, fmt.Errorf("%w: decoder", ErrCarrier)
	}
	defer func() { _ = dec.Close() }()
	var acc accumulator
	for {
		pkt, err := d.NextPacket()
		if errors.Is(err, av.ErrEOF) {
			break
		}
		if err != nil {
			return codec.PCM{}, codec.Params{}, fmt.Errorf("%w: %v", ErrCarrier, err)
		}
		if err := dec.Send(pkt); err != nil && !errors.Is(err, av.ErrAgain) {
			return codec.PCM{}, codec.Params{}, fmt.Errorf("%w: %v", ErrCarrier, err)
		}
		if err := acc.drain(dec); err != nil {
			return codec.PCM{}, codec.Params{}, err
		}
	}
	_ = dec.Send(av.Packet{})
	if err := acc.drain(dec); err != nil {
		return codec.PCM{}, codec.Params{}, err
	}
	if acc.n == 0 || len(acc.planes) == 0 {
		return codec.PCM{}, codec.Params{}, ErrCarrier
	}
	return acc.pcm(), info.Params(), nil
}

// accumulator collects a decoder's output into one contiguous set of
// float planes. Mist re-encodes the whole carrier, so it needs all of it.
type accumulator struct {
	planes [][]float32
	n      int
	ch     int
	rate   int
}

func (a *accumulator) drain(dec *av.Decoder) error {
	frames, err := av.DrainPCM(dec)
	if err != nil {
		return fmt.Errorf("%w: %v", ErrCarrier, err)
	}
	for _, f := range frames {
		a.add(f)
	}
	return nil
}

func (a *accumulator) add(f codec.PCM) {
	if a.ch == 0 && f.Channels > 0 {
		a.ch, a.rate = f.Channels, f.SampleRate
		a.planes = make([][]float32, a.ch)
	}
	for i, p := range f.Planes {
		if i < len(a.planes) {
			a.planes[i] = append(a.planes[i], p...)
		}
	}
	if len(f.Planes) > 0 {
		a.n += len(f.Planes[0])
	}
}

func (a *accumulator) pcm() codec.PCM {
	return codec.PCM{
		Planes:     a.planes,
		NbSamples:  a.n,
		Channels:   a.ch,
		SampleRate: a.rate,
		Format:     codec.SampleFmtFLTP,
	}
}

func codecName(info av.AudioInfo) string {
	if info.CodecName == "" {
		return "this carrier"
	}
	return info.CodecName
}

func marshalPlain(p Payload, senderPriv []byte) ([]byte, error) {
	wp := wire.Payload{Version: Version, Type: byte(p.Type), Data: p.Data}
	if len(senderPriv) > 0 {
		body, err := wire.MarshalPayload(wp)
		if err != nil {
			return nil, err
		}
		// sign the type+length+data, not including a pre-existing sig
		sig, err := crypto.Sign(senderPriv, body)
		if err != nil {
			return nil, fmt.Errorf("%w: sign", err)
		}
		wp.Signature = sig
	}
	return wire.MarshalPayload(wp)
}

func muxPackets(info av.AudioInfo, pkts []codec.Packet) (io.ReadCloser, error) {
	var buf seekBuf
	m, err := av.NewMuxer(&buf, info)
	if err != nil {
		return nil, fmt.Errorf("%w: muxer", err)
	}
	if err := m.WriteHeader(); err != nil {
		_ = m.Close()
		return nil, err
	}
	pts := int64(0)
	for _, pkt := range pkts {
		if pkt.PTS == 0 && pkt.Duration == 0 {
			pkt.PTS = pts
			pkt.DTS = pts
			pkt.Duration = 64
		}
		pts = pkt.PTS + pkt.Duration
		if err := m.WritePacket(av.FromCodecPacket(pkt)); err != nil {
			_ = m.Close()
			return nil, err
		}
	}
	if err := m.WriteTrailer(); err != nil {
		_ = m.Close()
		return nil, err
	}
	if err := m.Close(); err != nil {
		return nil, err
	}
	return io.NopCloser(bytes.NewReader(buf.data)), nil
}

func asSeeker(r io.Reader) io.ReadSeeker {
	if rs, ok := r.(io.ReadSeeker); ok {
		return rs
	}
	b, _ := io.ReadAll(r)
	return bytes.NewReader(b)
}

// openSource opens a local path or file:// URL. Embed routes http(s)://
// sources to embedURL before ever reaching this function.
func openSource(source string) (io.ReadCloser, error) {
	if source == "" {
		return nil, ErrInvalidSource
	}
	path := source
	if u, err := url.Parse(source); err == nil && u.Scheme == "file" {
		path = u.Path
	}
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrCarrier, err)
	}
	return f, nil
}

type seekBuf struct {
	data []byte
	off  int
}

func (b *seekBuf) Write(p []byte) (int, error) {
	end := b.off + len(p)
	if end > len(b.data) {
		b.data = append(b.data, make([]byte, end-len(b.data))...)
	}
	copy(b.data[b.off:], p)
	b.off += len(p)
	return len(p), nil
}

func (b *seekBuf) Read(p []byte) (int, error) {
	if b.off >= len(b.data) {
		return 0, io.EOF
	}
	n := copy(p, b.data[b.off:])
	b.off += n
	return n, nil
}

func (b *seekBuf) Seek(offset int64, whence int) (int64, error) {
	var n int64
	switch whence {
	case io.SeekStart:
		n = offset
	case io.SeekCurrent:
		n = int64(b.off) + offset
	case io.SeekEnd:
		n = int64(len(b.data)) + offset
	default:
		return 0, errors.New("whence")
	}
	if n < 0 {
		return 0, errors.New("negative seek")
	}
	b.off = int(n)
	return n, nil
}

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

// FrameCapacity returns a rough upper bound on the payload bytes
// embeddable per stego frame at the library's fixed embedding density,
// before encryption overhead. It is a planning heuristic for a typical
// carrier, not the real limit Embed enforces — EstimateCapacity reports
// that, for an actual carrier, and also accounts for Embed spanning the
// payload across frames when it does not fit in a single one.
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
	pcm, info, err := decodeCarrier(r)
	if err != nil {
		return nil, err
	}
	return e.embedPCM(ctx, pcm, info, payload)
}

// embedURL decodes carrier directly via libav's own URL handling (path,
// file://, or http(s)://) instead of tunneling bytes through a Go
// io.Reader — the only way to reach a plain http(s) source, since it is
// not itself an io.Reader the caller hands us.
func (e *Emitter) embedURL(ctx context.Context, source string, payload Payload) (io.ReadCloser, error) {
	if err := e.validate(ctx, payload); err != nil {
		return nil, err
	}
	pcm, info, err := decodeCarrierURL(source)
	if err != nil {
		return nil, err
	}
	return e.embedPCM(ctx, pcm, info, payload)
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
func (e *Emitter) embedPCM(ctx context.Context, pcm codec.PCM, info av.AudioInfo, payload Payload) (io.ReadCloser, error) {
	plain, err := marshalPlain(payload, e.senderPriv)
	if err != nil {
		return nil, err
	}
	enc, err := av.NewEncoder(e.target.Info(pcm.SampleRate, pcm.Channels, targetBitrate(e.target, info.Params())))
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
		return e.embedResidues(ctx, enc.Info(), pkts, frameParams(pcm), plain)
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
//
// Grouping needs pkts' timestamps, but Ogg does not store one per packet —
// only a granule position per page — so a demuxer reconstructs each
// packet's timestamp from that using standard Vorbis block-size
// accounting. At a genuine block-size transition (a loud/quiet transient
// in the source), that reconstruction can land a packet one window short
// or long of what the encoder itself reported, shifting it into a
// different stego frame than Embed used here. Grouping the encoder's own
// packets would then disagree with what Listen sees in the written file —
// silently, since it just looks like a broken span or a missed frame.
// canonicalPackets removes the mismatch by asking the same question
// Listen will: it muxes pkts once (unrewritten) and demuxes them straight
// back, so grouping runs on the exact packet stream — same order, same
// count, same reconstructed timestamps — any reader gets. Rewrite never
// changes a packet's size (same-Huffman-length substitution only), so
// this canonical stream's own container layout, and the timestamps a
// reader reconstructs from it, are unaffected by which frames end up
// carrying real data.
func (e *Emitter) embedResidues(ctx context.Context, info av.AudioInfo, pkts []codec.Packet, fp frame.Params, plain []byte) ([]codec.Packet, error) {
	pkts, err := canonicalPackets(info, pkts)
	if err != nil {
		return nil, err
	}
	vc := vorbis.New()
	if err := vc.Load(info.Extradata); err != nil {
		return nil, fmt.Errorf("%w: setup", err)
	}
	groups := frame.GroupPackets(pkts, fp)
	if len(groups) == 0 {
		return nil, ErrCarrier
	}
	rooms, err := groupRooms(ctx, vc, groups)
	if err != nil {
		return nil, err
	}
	plan := planChunks(plain, rooms)
	if plan == nil {
		return nil, noCapacity(plain)
	}
	all := make([]codec.Packet, 0, len(pkts))
	for i, g := range groups {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		out, err := e.embedGroup(vc, g, plan[i])
		if err != nil {
			return nil, err
		}
		all = append(all, out...)
	}
	return all, nil
}

// canonicalPackets round-trips pkts through an in-memory mux/demux pass so
// their timestamps match what any reader — Listen included — reconstructs
// from the container, rather than what the encoder itself reported. See
// embedResidues for why the two can differ.
func canonicalPackets(info av.AudioInfo, pkts []codec.Packet) ([]codec.Packet, error) {
	rc, err := muxPackets(info, pkts)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rc.Close() }()
	d, err := av.OpenDemuxerReader(asSeeker(rc))
	if err != nil {
		return nil, fmt.Errorf("%w: canonicalize: %v", ErrCarrier, err)
	}
	defer func() { _ = d.Close() }()
	out := make([]codec.Packet, 0, len(pkts))
	for {
		pkt, err := d.NextPacket()
		if errors.Is(err, av.ErrEOF) {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("%w: canonicalize: %v", ErrCarrier, err)
		}
		out = append(out, av.ToCodecPacket(pkt))
	}
	return out, nil
}

// groupRooms reports each Vorbis stego frame's real capacity in bytes —
// the same room stego.Capacity would report to Apply — without writing
// anything, so planChunks can decide where the payload goes before any
// frame is touched. A group with no eligible residues at all reports 0
// rather than failing: it is left untouched either way.
func groupRooms(ctx context.Context, vc *vorbis.Codec, groups []frame.Group) ([]int, error) {
	rooms := make([]int, len(groups))
	for i, g := range groups {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		room, err := stego.Capacity(vc, g.Packets)
		if errors.Is(err, stego.ErrNoResidues) {
			continue
		}
		if err != nil {
			return nil, fmt.Errorf("%w: %v", ErrCarrier, err)
		}
		rooms[i] = room
	}
	return rooms, nil
}

// embedGroup writes one frame. plainChunk is the framed bytes planChunks
// assigned this frame this round, or nil to fill it with CSPRNG filler at
// the same density — never left untouched unless it has no eligible
// residues at all, a trailing sliver of encoder padding.
func (e *Emitter) embedGroup(vc *vorbis.Codec, g frame.Group, plainChunk []byte) ([]codec.Packet, error) {
	var bits stego.Bits
	if plainChunk != nil {
		env, err := crypto.Seal(plainChunk, e.pub)
		if err != nil {
			return nil, err
		}
		bits = env.Marshal()
	}
	pos, err := crypto.PositionSeed(e.pub, g.Index)
	if err != nil {
		return nil, err
	}
	out, err := stego.Apply(vc, pos, g.Packets, bits)
	if errors.Is(err, stego.ErrNoResidues) {
		return g.Packets, nil
	}
	if errors.Is(err, stego.ErrCapacity) {
		return nil, fmt.Errorf("%w: %v", ErrInvalidPayload, err)
	}
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrCarrier, err)
	}
	return out, nil
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

func decodeCarrier(r io.Reader) (codec.PCM, av.AudioInfo, error) {
	d, err := av.OpenDemuxerReader(asSeeker(r))
	if err != nil {
		return codec.PCM{}, av.AudioInfo{}, fmt.Errorf("%w: %v", ErrCarrier, err)
	}
	defer func() { _ = d.Close() }()
	return decodeFromDemuxer(d)
}

// decodeCarrierURL decodes a carrier libav can open natively — a plain
// path, a file:// URL, or an http(s):// URL — without tunneling it
// through a Go io.Reader. This is the only route into a plain http(s)
// source, since it does not arrive as an io.Reader in the first place.
func decodeCarrierURL(source string) (codec.PCM, av.AudioInfo, error) {
	d, err := av.OpenDemuxer(source)
	if err != nil {
		return codec.PCM{}, av.AudioInfo{}, fmt.Errorf("%w: %v", ErrCarrier, err)
	}
	defer func() { _ = d.Close() }()
	return decodeFromDemuxer(d)
}

func decodeFromDemuxer(d *av.Demuxer) (codec.PCM, av.AudioInfo, error) {
	info := d.Info()
	if !av.CanDecode(info) {
		return codec.PCM{}, av.AudioInfo{},
			fmt.Errorf("%w: this FFmpeg build cannot decode %s", ErrUnsupportedCodec, codecName(info))
	}
	dec, err := av.NewDecoder(info)
	if err != nil {
		return codec.PCM{}, av.AudioInfo{}, fmt.Errorf("%w: decoder", ErrCarrier)
	}
	defer func() { _ = dec.Close() }()
	var acc accumulator
	for {
		pkt, err := d.NextPacket()
		if errors.Is(err, av.ErrEOF) {
			break
		}
		if err != nil {
			return codec.PCM{}, av.AudioInfo{}, fmt.Errorf("%w: %v", ErrCarrier, err)
		}
		if err := dec.Send(pkt); err != nil && !errors.Is(err, av.ErrAgain) {
			return codec.PCM{}, av.AudioInfo{}, fmt.Errorf("%w: %v", ErrCarrier, err)
		}
		if err := acc.drain(dec); err != nil {
			return codec.PCM{}, av.AudioInfo{}, err
		}
	}
	_ = dec.Send(av.Packet{})
	if err := acc.drain(dec); err != nil {
		return codec.PCM{}, av.AudioInfo{}, err
	}
	if acc.n == 0 || len(acc.planes) == 0 {
		return codec.PCM{}, av.AudioInfo{}, ErrCarrier
	}
	return acc.pcm(), info, nil
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

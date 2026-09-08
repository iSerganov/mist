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
	pcm, params, extra, err := decodeCarrier(r)
	if err != nil {
		return nil, err
	}
	return e.embedPCM(ctx, pcm, params, extra, payload)
}

// embedURL decodes carrier directly via libav's own URL handling (path,
// file://, or http(s)://) instead of tunneling bytes through a Go
// io.Reader — the only way to reach a plain http(s) source, since it is
// not itself an io.Reader the caller hands us.
func (e *Emitter) embedURL(ctx context.Context, source string, payload Payload) (io.ReadCloser, error) {
	if err := e.validate(ctx, payload); err != nil {
		return nil, err
	}
	pcm, params, extra, err := decodeCarrierURL(source)
	if err != nil {
		return nil, err
	}
	return e.embedPCM(ctx, pcm, params, extra, payload)
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

func (e *Emitter) embedPCM(ctx context.Context, pcm codec.PCM, params codec.Params, extra []byte, payload Payload) (io.ReadCloser, error) {
	plain, err := marshalPlain(payload, e.senderPriv)
	if err != nil {
		return nil, err
	}
	vc := vorbis.New()
	enc, err := vc.NewEncoder(codec.Params{
		ID:         codec.IDVorbis,
		SampleRate: params.SampleRate,
		Channels:   params.Channels,
		Bitrate:    encodeBitrate(params.Bitrate),
		Format:     codec.SampleFmtFLTP,
	})
	if err != nil {
		return nil, fmt.Errorf("%w: encoder", ErrCarrier)
	}
	defer func() { _ = enc.Close() }()
	info := enc.(interface{ Params() codec.Params }).Params()
	if err := vc.Load(info.Extradata); err != nil {
		return nil, fmt.Errorf("%w: setup", err)
	}
	_ = extra
	pkts, err := enc.Encode(pcm)
	if err != nil {
		return nil, err
	}
	flushed, err := enc.Flush()
	if err != nil {
		return nil, err
	}
	groups := frame.GroupPackets(append(pkts, flushed...), frameParams(pcm))
	if len(groups) == 0 {
		return nil, ErrCarrier
	}
	all := make([]codec.Packet, 0, len(pkts)+len(flushed))
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
		return nil, fmt.Errorf("%w: %d bytes needed", ErrNoCapacity, len(plain)+EnvelopeOverhead)
	}
	return muxPackets(info, all)
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
	if errors.Is(err, stego.ErrCapacity) {
		return nil, false, fmt.Errorf("%w: %v", ErrInvalidPayload, err)
	}
	if err != nil {
		return nil, false, fmt.Errorf("%w: %v", ErrCarrier, err)
	}
	return out, carrying, nil
}

// Bitrate bounds for the re-encode. Mist always re-encodes, and a second
// lossy pass at the source's own rate compounds the loss, so a lossy
// carrier is given headroom above its rate rather than matched to it.
const (
	minBitrate     = 192_000
	maxBitrate     = 500_000
	unknownBitrate = 256_000
)

// encodeBitrate picks the Vorbis target for a carrier of the given rate.
// A lossless source reports its raw PCM rate — 1411 kbps for CD audio —
// which is not a meaningful target for a lossy codec, so it is capped.
func encodeBitrate(source int64) int64 {
	if source <= 0 {
		return unknownBitrate
	}
	return min(max(source*3/2, minBitrate), maxBitrate)
}

func frameParams(pcm codec.PCM) frame.Params {
	return frame.Params{
		SampleRate: pcm.SampleRate,
		Channels:   pcm.Channels,
		Duration:   FrameDuration,
	}
}

func decodeCarrier(r io.Reader) (codec.PCM, codec.Params, []byte, error) {
	d, err := av.OpenDemuxerReader(asSeeker(r))
	if err != nil {
		return codec.PCM{}, codec.Params{}, nil, fmt.Errorf("%w: %v", ErrCarrier, err)
	}
	defer func() { _ = d.Close() }()
	return decodeFromDemuxer(d)
}

// decodeCarrierURL decodes a carrier libav can open natively — a plain
// path, a file:// URL, or an http(s):// URL — without tunneling it
// through a Go io.Reader. This is the only route into a plain http(s)
// source, since it does not arrive as an io.Reader in the first place.
func decodeCarrierURL(source string) (codec.PCM, codec.Params, []byte, error) {
	d, err := av.OpenDemuxer(source)
	if err != nil {
		return codec.PCM{}, codec.Params{}, nil, fmt.Errorf("%w: %v", ErrCarrier, err)
	}
	defer func() { _ = d.Close() }()
	return decodeFromDemuxer(d)
}

func decodeFromDemuxer(d *av.Demuxer) (codec.PCM, codec.Params, []byte, error) {
	info := d.Info()
	if !av.CanDecode(info) {
		return codec.PCM{}, codec.Params{}, nil,
			fmt.Errorf("%w: this FFmpeg build cannot decode %s", ErrUnsupportedCodec, codecName(info))
	}
	dec, err := av.NewDecoder(info)
	if err != nil {
		return codec.PCM{}, codec.Params{}, nil, fmt.Errorf("%w: decoder", ErrCarrier)
	}
	defer func() { _ = dec.Close() }()
	var planes [][]float32
	var n, ch, rate int
	for {
		pkt, err := d.NextPacket()
		if errors.Is(err, av.ErrEOF) {
			break
		}
		if err != nil {
			return codec.PCM{}, codec.Params{}, nil, fmt.Errorf("%w: %v", ErrCarrier, err)
		}
		if err := dec.Send(pkt); err != nil && !errors.Is(err, av.ErrAgain) {
			return codec.PCM{}, codec.Params{}, nil, fmt.Errorf("%w: %v", ErrCarrier, err)
		}
		more, c, rt, err := drainPCM(dec)
		if err != nil {
			return codec.PCM{}, codec.Params{}, nil, err
		}
		if ch == 0 && c > 0 {
			ch, rate = c, rt
			planes = make([][]float32, ch)
		}
		for i := range more {
			if i < len(planes) {
				planes[i] = append(planes[i], more[i]...)
			}
		}
		if len(more) > 0 {
			n += len(more[0])
		}
	}
	_ = dec.Send(av.Packet{})
	more, _, _, err := drainPCM(dec)
	if err != nil {
		return codec.PCM{}, codec.Params{}, nil, err
	}
	for i := range more {
		if i < len(planes) {
			planes[i] = append(planes[i], more[i]...)
		}
	}
	if len(more) > 0 {
		n += len(more[0])
	}
	if n == 0 || ch == 0 {
		return codec.PCM{}, codec.Params{}, nil, ErrCarrier
	}
	return codec.PCM{
		Planes:     planes,
		NbSamples:  n,
		Channels:   ch,
		SampleRate: rate,
		Format:     codec.SampleFmtFLTP,
	}, info.Params(), info.Extradata, nil
}

func drainPCM(dec *av.Decoder) ([][]float32, int, int, error) {
	var planes [][]float32
	var ch, rate int
	for {
		fr, err := dec.Receive()
		if errors.Is(err, av.ErrAgain) || errors.Is(err, av.ErrEOF) {
			return planes, ch, rate, nil
		}
		if err != nil {
			return nil, 0, 0, fmt.Errorf("%w: %v", ErrCarrier, err)
		}
		pcm := frameToPCM(fr)
		if ch == 0 {
			ch, rate = pcm.Channels, pcm.SampleRate
			planes = make([][]float32, ch)
		}
		for i, p := range pcm.Planes {
			if i < len(planes) {
				planes[i] = append(planes[i], p...)
			}
		}
	}
}

// frameToPCM normalises whatever sample format the decoder produced into
// the float planes the Vorbis encoder takes.
func frameToPCM(f av.Frame) codec.PCM {
	return codec.PCM{
		Planes:     f.FloatPlanes(),
		NbSamples:  f.NbSamples,
		Channels:   f.Channels,
		SampleRate: f.SampleRate,
		Format:     codec.SampleFmtFLTP,
		PTS:        f.PTS,
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

func muxPackets(p codec.Params, pkts []codec.Packet) (io.ReadCloser, error) {
	var buf seekBuf
	m, err := av.NewMuxer(&buf, av.AudioInfo{
		CodecID:    av.CodecIDVorbis,
		SampleRate: p.SampleRate,
		Channels:   p.Channels,
		SampleFmt:  p.Format,
		Bitrate:    p.Bitrate,
		Extradata:  p.Extradata,
	})
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

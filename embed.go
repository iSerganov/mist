package mist

import (
	"bytes"
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"math"
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
		Bitrate:    64_000,
		Format:     codec.SampleFmtFLTP,
	})
	if err != nil {
		return nil, fmt.Errorf("%w: encoder", ErrCarrier)
	}
	defer enc.Close()
	info := enc.(interface{ Params() codec.Params }).Params()
	if err := vc.Load(info.Extradata); err != nil {
		return nil, fmt.Errorf("%w: setup", err)
	}
	_ = extra
	frames := frame.Split(pcm.NbSamples, frame.Params{
		SampleRate: pcm.SampleRate,
		Channels:   pcm.Channels,
		Duration:   FrameDuration,
	})
	if len(frames) == 0 {
		return nil, ErrCarrier
	}
	frames = mergeTrailingCrumb(frames)
	var all []codec.Packet
	for i, fr := range frames {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		slice := slicePCM(pcm, fr.PCMStart, fr.PCMEnd)
		pkts, err := enc.Encode(slice)
		if err != nil {
			return nil, err
		}
		if i == len(frames)-1 {
			flushed, err := enc.Flush()
			if err != nil {
				return nil, err
			}
			pkts = append(pkts, flushed...)
		}
		// A trailing frame can be a handful of decoder priming/padding
		// samples rather than genuine audio (libav's Vorbis round trip
		// adds a little), too small to hold even the envelope. Pass it
		// through unmodified: the payload is still recoverable from every
		// full-sized frame, and there's no way to make a frame this small
		// look like a real one regardless of what's embedded in it.
		frameCap, err := stego.Capacity(vc, pkts)
		if err != nil && !errors.Is(err, stego.ErrNoResidues) {
			return nil, fmt.Errorf("%w: %v", ErrCarrier, err)
		}
		if err != nil || frameCap < EnvelopeOverhead {
			all = append(all, pkts...)
			continue
		}
		env, err := crypto.Seal(plain, e.pub)
		if err != nil {
			return nil, err
		}
		pos, err := crypto.PositionSeed(e.pub, fr.Index)
		if err != nil {
			return nil, err
		}
		pkts, err = stego.Apply(vc, pos, pkts, env.Marshal())
		if err != nil {
			if errors.Is(err, stego.ErrCapacity) {
				return nil, fmt.Errorf("%w: %v", ErrInvalidPayload, err)
			}
			return nil, fmt.Errorf("%w: %v", ErrCarrier, err)
		}
		all = append(all, pkts...)
	}
	return muxPackets(info, all)
}

func decodeCarrier(r io.Reader) (codec.PCM, codec.Params, []byte, error) {
	d, err := av.OpenDemuxerReader(asSeeker(r))
	if err != nil {
		return codec.PCM{}, codec.Params{}, nil, fmt.Errorf("%w: %v", ErrCarrier, err)
	}
	defer d.Close()
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
	defer d.Close()
	return decodeFromDemuxer(d)
}

func decodeFromDemuxer(d *av.Demuxer) (codec.PCM, codec.Params, []byte, error) {
	info := d.Info()
	dec, err := av.NewDecoder(info)
	if err != nil {
		return codec.PCM{}, codec.Params{}, nil, fmt.Errorf("%w: decoder", ErrCarrier)
	}
	defer dec.Close()
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

func frameToPCM(f av.Frame) codec.PCM {
	planes := make([][]float32, len(f.Data))
	for i, b := range f.Data {
		n := len(b) / 4
		pl := make([]float32, n)
		for j := 0; j < n; j++ {
			pl[j] = math.Float32frombits(binary.LittleEndian.Uint32(b[j*4:]))
		}
		planes[i] = pl
	}
	return codec.PCM{
		Planes:     planes,
		NbSamples:  f.NbSamples,
		Channels:   f.Channels,
		SampleRate: f.SampleRate,
		Format:     f.Format,
		PTS:        f.PTS,
	}
}

// crumbSamples bounds a trailing frame libav's own encode/decode round
// trip can leave behind (priming/lookahead, typically well under a
// couple thousand samples) — not genuine trailing audio. Merging it into
// the previous frame keeps a carrier that is an exact multiple of
// FrameDuration from spuriously growing an extra, near-empty stego frame
// with no room for even the protocol envelope.
const crumbSamples = 4096

// mergeTrailingCrumb folds a final frame shorter than crumbSamples into
// the one before it, when there is a previous frame to fold into.
func mergeTrailingCrumb(frames []frame.Frame) []frame.Frame {
	if len(frames) < 2 {
		return frames
	}
	last := frames[len(frames)-1]
	if last.PCMEnd-last.PCMStart >= crumbSamples {
		return frames
	}
	frames = frames[:len(frames)-1]
	prev := &frames[len(frames)-1]
	prev.PCMEnd = last.PCMEnd
	prev.Duration += last.Duration
	return frames
}

func slicePCM(p codec.PCM, start, end int) codec.PCM {
	if start < 0 {
		start = 0
	}
	if end > p.NbSamples {
		end = p.NbSamples
	}
	if end < start {
		end = start
	}
	n := end - start
	planes := make([][]float32, len(p.Planes))
	for i, pl := range p.Planes {
		if start < len(pl) {
			e := end
			if e > len(pl) {
				e = len(pl)
			}
			planes[i] = pl[start:e]
		}
	}
	return codec.PCM{
		Planes:     planes,
		NbSamples:  n,
		Channels:   p.Channels,
		SampleRate: p.SampleRate,
		Format:     codec.SampleFmtFLTP,
		PTS:        int64(start),
	}
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

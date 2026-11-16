package mist

import (
	"bytes"
	"context"
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

// openEncoder opens target for this carrier. A lossless encoder is given
// no rate at all: it decides its own size.
func openEncoder(target av.Format, pcm codec.PCM, info av.AudioInfo) (*av.Encoder, error) {
	want := target.Info(pcm.SampleRate, pcm.Channels, info)
	if !target.Lossless {
		return openVorbis(want, info)
	}
	enc, err := av.NewEncoder(want)
	if err != nil {
		return nil, fmt.Errorf("%w: encoder: %v", ErrCarrier, err)
	}
	return enc, nil
}

// Vorbis quality levels, as libvorbis and ffmpeg's -q:a number them.
const (
	minQuality      = -1
	maxQuality      = 10
	losslessQuality = 8
)

// efficiency is how many kbps of Vorbis match one kbps of a source codec
// in quality. A codec not listed counts as Vorbis's equal.
var efficiency = map[string]float64{
	"mp3":  0.75,
	"aac":  1.0,
	"opus": 1.3,
}

// openVorbis opens libvorbis in VBR mode at the whole quality level whose
// nominal rate is nearest to what keeps the carrier's quality (the higher
// one on a tie), so the stream header reads like `ffmpeg -q:a N` rather
// than an odd ABR target. A lossless source, or one that reports no rate,
// is written at q8.
// libvorbis owns the nominal rate of each level for a given sample rate
// and channel count, so each is opened and asked rather than tabled here.
func openVorbis(want, src av.AudioInfo) (*av.Encoder, error) {
	target := float64(src.Bitrate) * efficiencyOf(src.CodecName)
	keepLevel := src.Bitrate <= 0 || av.Lossless(src.NativeCodecID)
	var best *av.Encoder
	bestGap, refused := math.Inf(1), error(nil)
	for q := minQuality; q <= maxQuality; q++ {
		want.VBR, want.Quality = true, q
		enc, err := av.NewEncoder(want)
		if err != nil {
			refused = err
			continue
		}
		gap := math.Abs(float64(q - losslessQuality))
		if !keepLevel {
			gap = math.Abs(float64(nominalRate(enc)) - target)
		}
		if gap <= bestGap {
			if best != nil {
				_ = best.Close()
			}
			best, bestGap = enc, gap
			continue
		}
		_ = enc.Close()
	}
	if best == nil {
		return nil, fmt.Errorf("%w: encoder: %v", ErrCarrier, refused)
	}
	return best, nil
}

func efficiencyOf(codecName string) float64 {
	if f, ok := efficiency[codecName]; ok {
		return f
	}
	return 1
}

// nominalRate is the bitrate libvorbis wrote into the stream header, 0 if
// the header does not parse.
func nominalRate(enc *av.Encoder) int32 {
	vc := vorbis.New()
	if vc.Load(enc.Info().Extradata) != nil {
		return 0
	}
	return vc.Setup().BitrateNom
}

// encodeAndMux encodes a carrier already held in memory and muxes it with
// nothing embedded. Analysis uses it for the cover a stego file is
// compared against.
func encodeAndMux(enc *av.Encoder, pcm codec.PCM, meta av.Metadata) (io.ReadCloser, error) {
	pkts, err := enc.Encode(pcm)
	if err != nil {
		return nil, err
	}
	flushed, err := enc.Flush()
	if err != nil {
		return nil, err
	}
	return muxPackets(enc.Info(), meta, append(pkts, flushed...))
}

// residueStage is the Vorbis half of the pipeline: encode, group the
// packets into stego frames and rewrite residue LSBs in each.
//
// Grouping needs packet timestamps, but Ogg does not store one per packet —
// only a granule position per page — so a demuxer reconstructs each
// packet's timestamp from that using standard Vorbis block-size
// accounting. At a genuine block-size transition (a loud/quiet transient
// in the source), that reconstruction can land a packet one window short
// or long of what the encoder itself reported, shifting it into a
// different stego frame than Embed used here. Grouping the encoder's own
// packets would then disagree with what Listen sees in the written file —
// silently, since it just looks like a broken span or a missed frame.
//
// So the stage asks the same question Listen will. It muxes the encoder's
// packets, unrewritten, into a pipe, and a second goroutine demuxes them
// straight back and groups what it reads: the same order, count and
// reconstructed timestamps any reader gets. Rewrite never changes a
// packet's size (same-Huffman-length substitution only), so the layout of
// the file written from those groups, and the timestamps a reader
// reconstructs from it, are the loopback's own. The pipe holds nothing:
// the encoder waits until the reading side has taken what it wrote.
type residueStage struct {
	pass
	enc  *av.Encoder
	loop *packetWriter
	pipe *io.PipeWriter
	read chan error // the reading goroutine's result
	sent int64      // samples handed to the encoder so far
}

func (s *residueStage) push(pcm codec.PCM) error {
	if s.enc == nil {
		if err := s.open(pcm); err != nil {
			return err
		}
	}
	pcm.PTS = s.sent
	s.sent += int64(len(pcm.Planes[0]))
	pkts, err := s.enc.Encode(pcm)
	if err != nil {
		return err
	}
	return s.feed(pkts)
}

func (s *residueStage) open(pcm codec.PCM) error {
	enc, err := openEncoder(s.e.target, pcm, s.src)
	if err != nil {
		return err
	}
	s.enc = enc
	info := enc.Info()
	pr, pw := io.Pipe()
	s.pipe, s.read = pw, make(chan error, 1)
	go func() {
		err := s.regroup(pr, info, frameParams(pcm))
		// Unblocks an encoder still writing after the reader gave up.
		_ = pr.CloseWithError(errors.Join(errLoopback, err))
		s.read <- err
	}()
	s.loop, err = openMux(pw, info, av.Metadata{})
	return s.readerFirst(err)
}

// feed writes the encoder's packets into the loopback.
func (s *residueStage) feed(pkts []codec.Packet) error {
	return s.readerFirst(s.loop.write(pkts))
}

// readerFirst replaces a failed loopback write with the reading side's
// own error, which is the cause: the write only failed because it stopped.
func (s *residueStage) readerFirst(err error) error {
	if err == nil {
		return nil
	}
	if cause := s.stop(err); cause != nil {
		return cause
	}
	return err
}

// stop ends the loopback and returns the reading side's result.
func (s *residueStage) stop(why error) error {
	if s.pipe == nil {
		return nil
	}
	_ = s.pipe.CloseWithError(why)
	s.pipe = nil
	return <-s.read
}

func (s *residueStage) finish() error {
	if s.enc == nil {
		return ErrCarrier
	}
	pkts, err := s.enc.Flush()
	if err != nil {
		return err
	}
	if err := s.feed(pkts); err != nil {
		return err
	}
	if err := s.readerFirst(s.loop.finish()); err != nil {
		return err
	}
	return s.stop(nil)
}

func (s *residueStage) close() {
	_ = s.stop(errLoopback)
	if s.loop != nil {
		s.loop.close()
	}
	if s.enc != nil {
		_ = s.enc.Close()
	}
}

// errLoopback marks the loopback pipe closed by one side giving up.
var errLoopback = errors.New("mist: loopback closed")

// regroup reads the loopback and takes each stego frame it completes
// through placement, embedding and the output muxer.
func (s *residueStage) regroup(r io.Reader, info av.AudioInfo, fp frame.Params) error {
	d, err := av.OpenDemuxerReader(r)
	if err != nil {
		return fmt.Errorf("%w: canonicalize: %v", ErrCarrier, err)
	}
	defer func() { _ = d.Close() }()
	vc := vorbis.New()
	if err := vc.Load(info.Extradata); err != nil {
		return fmt.Errorf("%w: setup", err)
	}
	var out *packetWriter
	if s.dst != nil {
		if out, err = openMux(s.dst, info, s.meta); err != nil {
			return err
		}
		defer out.close()
	}
	g := frame.NewGrouper(fp)
	groups := 0
	emit := func(grp frame.Group, ok bool) error {
		if !ok {
			return nil
		}
		groups++
		return s.embedInto(vc, out, grp)
	}
	for {
		pkt, err := d.NextPacket()
		if errors.Is(err, av.ErrEOF) {
			break
		}
		if err != nil {
			return fmt.Errorf("%w: canonicalize: %v", ErrCarrier, err)
		}
		if err := emit(g.Push(av.ToCodecPacket(pkt))); err != nil {
			return err
		}
	}
	if err := emit(g.Flush()); err != nil {
		return err
	}
	if groups == 0 {
		return ErrCarrier
	}
	if out == nil {
		return nil
	}
	return out.finish()
}

// embedInto places, embeds and writes one stego frame. A frame with no
// eligible residues at all has no room and is written as it is.
func (s *residueStage) embedInto(vc *vorbis.Codec, out *packetWriter, grp frame.Group) error {
	room, err := stego.Capacity(vc, grp.Packets)
	if err != nil && !errors.Is(err, stego.ErrNoResidues) {
		return fmt.Errorf("%w: %v", ErrCarrier, err)
	}
	chunk := s.place.place(room)
	if out == nil {
		return nil
	}
	pkts, err := s.e.embedGroup(vc, grp, chunk)
	if err != nil {
		return err
	}
	if err := out.write(pkts); err != nil {
		return err
	}
	s.report()
	return nil
}

func readPackets(r io.Reader) (av.AudioInfo, []codec.Packet, error) {
	d, err := av.OpenDemuxerReader(asSeeker(r))
	if err != nil {
		return av.AudioInfo{}, nil, err
	}
	defer func() { _ = d.Close() }()
	var out []codec.Packet
	for {
		pkt, err := d.NextPacket()
		if errors.Is(err, av.ErrEOF) {
			return d.Info(), out, nil
		}
		if err != nil {
			return av.AudioInfo{}, nil, err
		}
		out = append(out, av.ToCodecPacket(pkt))
	}
}

// embedGroup writes one frame. plainChunk is the framed bytes the placer
// assigned this frame, or nil to fill it with CSPRNG filler at
// the same density — never left untouched unless it has no eligible
// residues at all, a trailing sliver of encoder padding.
func (e *Emitter) embedGroup(vc *vorbis.Codec, g frame.Group, plainChunk []byte) ([]codec.Packet, error) {
	bits, err := e.seal(plainChunk)
	if err != nil {
		return nil, err
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

// seal encrypts chunk for the recipient under a fresh ephemeral key. A nil
// chunk gives nil bits, which the stego layer writes as filler.
func (e *Emitter) seal(chunk []byte) (stego.Bits, error) {
	if chunk == nil {
		return nil, nil
	}
	env, err := crypto.Seal(chunk, e.pub)
	if err != nil {
		return nil, err
	}
	return env.Marshal(), nil
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

func decodeCarrier(r io.Reader) (codec.PCM, av.AudioInfo, av.Metadata, error) {
	d, err := av.OpenDemuxerReader(asSeeker(r))
	if err != nil {
		return codec.PCM{}, av.AudioInfo{}, av.Metadata{}, fmt.Errorf("%w: %v", ErrCarrier, err)
	}
	defer func() { _ = d.Close() }()
	return decodeFromDemuxer(d)
}

func decodeFromDemuxer(d *av.Demuxer) (codec.PCM, av.AudioInfo, av.Metadata, error) {
	meta, err := d.Metadata()
	if err != nil {
		return codec.PCM{}, av.AudioInfo{}, av.Metadata{}, carrierErr(err)
	}
	next := func() (av.Packet, error) {
		pkt, err := d.NextPacket()
		if err != nil && !errors.Is(err, av.ErrEOF) {
			return av.Packet{}, carrierErr(err)
		}
		return pkt, err
	}
	var acc accumulator
	err = decodeEach(d.Info(), next, func(c codec.PCM) error {
		acc.add(c)
		return nil
	})
	if err != nil {
		return codec.PCM{}, av.AudioInfo{}, av.Metadata{}, err
	}
	if acc.n == 0 || len(acc.planes) == 0 {
		return codec.PCM{}, av.AudioInfo{}, av.Metadata{}, ErrCarrier
	}
	return acc.pcm(), d.Info(), meta, nil
}

// accumulator collects a decoder's output into one contiguous set of
// float planes. Mist re-encodes the whole carrier, so it needs all of it.
type accumulator struct {
	planes [][]float32
	frames []int
	n      int
	ch     int
	rate   int
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
		a.frames = append(a.frames, len(f.Planes[0]))
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
		Frames:     a.frames,
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

// writeTags copies source metadata the way ffmpeg's CLI does: file tags
// first, then the audio stream's, dropping the product tags the CLI
// itself removes. Stream tags do not replace an encoder ident the muxer
// already wrote.
func writeTags(m *av.Muxer, meta av.Metadata) error {
	skip := map[string]struct{}{
		"creation_time": {}, "company_name": {}, "product_name": {}, "product_version": {},
	}
	for _, t := range meta.Format {
		if _, drop := skip[t.Key]; drop {
			continue
		}
		if err := m.AddTag(av.TagsFormat, t.Key, t.Value); err != nil {
			return err
		}
	}
	for _, t := range meta.Stream {
		if err := m.AddTag(av.TagsStream, t.Key, t.Value); err != nil {
			return err
		}
	}
	return nil
}

func muxPackets(info av.AudioInfo, meta av.Metadata, pkts []codec.Packet) (io.ReadCloser, error) {
	var buf seekBuf
	w, err := openMux(&buf, info, meta)
	if err != nil {
		return nil, err
	}
	defer w.close()
	if err := w.write(pkts); err != nil {
		return nil, err
	}
	if err := w.finish(); err != nil {
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

// openSource opens a local path or file:// URL. pathSource hands
// http(s):// sources to libav instead.
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

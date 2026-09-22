package mist

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/iSerganov/mist/internal/av"
	"github.com/iSerganov/mist/internal/codec"
	"github.com/iSerganov/mist/internal/codec/vorbis"
	"github.com/iSerganov/mist/internal/crypto"
	"github.com/iSerganov/mist/internal/frame"
	"github.com/iSerganov/mist/internal/stego"
	"github.com/iSerganov/mist/internal/wire"
)

// scanner turns a demuxed packet stream into authenticated payloads. It
// has one implementation per embedding domain — Vorbis residues, lossless
// PCM samples — and both decrypt through the same opener, because only
// where the bits were hidden differs.
type scanner interface {
	push(codec.Packet) (Result, bool)
	flush() (Result, bool)
	Close() error
}

// newScanner picks the domain the stream was embedded in. Extraction is
// possible wherever Emit could write: Vorbis, whose residues Mist parses,
// and any lossless codec, whose samples come back unchanged.
func newScanner(priv []byte, info av.AudioInfo, log *slog.Logger) (scanner, error) {
	pub, err := crypto.PublicFromPrivate(priv)
	if err != nil {
		return nil, ErrInvalidKey
	}
	o := &opener{priv: priv, pub: pub, log: log}
	p := info.Params()
	fp := frame.Params{SampleRate: p.SampleRate, Channels: p.Channels, Duration: FrameDuration}
	switch {
	case info.CodecID == av.CodecIDVorbis:
		vc := vorbis.New()
		if err := vc.Load(info.Extradata); err != nil {
			return nil, fmt.Errorf("%w: setup: %v", ErrCarrier, err)
		}
		return &residueScanner{opener: o, vc: vc, grouper: frame.NewGrouper(fp)}, nil
	case av.Lossless(info.NativeCodecID):
		dec, err := av.NewDecoder(info)
		if err != nil {
			return nil, fmt.Errorf("%w: decoder: %v", ErrCarrier, err)
		}
		return &sampleScanner{
			opener: o,
			dec:    dec,
			win:    newWindower(fp, av.SampleScale(info.SampleFmt)),
		}, nil
	default:
		return nil, fmt.Errorf("%w: %s is lossy and carries no recoverable bits",
			ErrUnsupportedCodec, codecName(info))
	}
}

// opener is the half of a scan that does not depend on where the bits
// came from. Every failure here is a silent skip: a frame that carries no
// payload and a frame this key cannot open must be indistinguishable.
//
// A payload that did not fit one frame arrives as consecutive chunks (see
// span.go and wire's span framing), so opener also carries the assembly
// state across calls: span, want and startIdx are empty except mid-span.
type opener struct {
	priv   []byte
	pub    []byte
	log    *slog.Logger
	joined bool

	spanning bool
	span     []byte
	want     int
	startIdx int64
}

func (o *opener) open(idx int64, bits stego.Bits, err error) (Result, bool) {
	if err != nil {
		return Result{}, false
	}
	env, err := wire.UnmarshalEnvelope(bits)
	if err != nil {
		return Result{}, false
	}
	plain, err := crypto.Open(&env, o.priv)
	if err != nil {
		return Result{}, false
	}
	if o.spanning {
		return o.resume(idx, plain)
	}
	return o.start(idx, plain)
}

// start handles a frame that is not a continuation of one already being
// assembled: either a complete, unspanned payload (today's format,
// unmodified), or the first chunk of one that needed to spread across
// several frames.
func (o *opener) start(idx int64, plain []byte) (Result, bool) {
	if p, err := wire.UnmarshalPayload(plain); err == nil {
		return Result{Payload: Payload{Type: PayloadType(p.Type), Data: p.Data}, FrameIdx: idx}, true
	}
	want, chunk, ok := wire.UnmarshalSpanStart(plain)
	if !ok {
		return Result{}, false
	}
	o.spanning, o.want, o.startIdx = true, int(want), idx
	o.span = append(o.span[:0], chunk...)
	return o.assembled()
}

// resume appends a later chunk of a payload already being assembled. A
// frame that authenticates but does not carry a valid continuation marker
// means the span was interrupted (a frame lost or corrupted in transit);
// the partial assembly is discarded rather than ever handed out.
func (o *opener) resume(idx int64, plain []byte) (Result, bool) {
	chunk, ok := wire.UnmarshalSpanContinue(plain)
	if !ok {
		o.reset()
		return Result{}, false
	}
	o.span = append(o.span, chunk...)
	return o.assembled()
}

// assembled reports the reassembled payload once every chunk has arrived.
func (o *opener) assembled() (Result, bool) {
	if len(o.span) < o.want {
		return Result{}, false
	}
	full, startIdx := o.span[:o.want], o.startIdx
	o.reset()
	p, err := wire.UnmarshalPayload(full)
	if err != nil {
		return Result{}, false
	}
	return Result{Payload: Payload{Type: PayloadType(p.Type), Data: p.Data}, FrameIdx: startIdx}, true
}

func (o *opener) reset() {
	o.spanning, o.span, o.want = false, nil, 0
}

// seed derives this frame's position key, or reports that the frame
// cannot be read at all.
func (o *opener) seed(idx int64) ([]byte, bool) {
	pos, err := crypto.PositionSeed(o.pub, idx)
	return pos, err == nil
}

// partial logs the single diagnostic a scan emits. A listener that joined
// mid-window cannot align to it, and there is no phase search.
func (o *opener) partial(idx int64) (Result, bool) {
	if !o.joined {
		o.joined = true
		o.log.Info("joined in the middle of transmission, cannot decrypt", "frame", idx)
	}
	return Result{}, false
}

// residueScanner reads bits from quantized Vorbis residues.
type residueScanner struct {
	*opener
	vc      *vorbis.Codec
	grouper *frame.Grouper
}

func (s *residueScanner) push(pkt codec.Packet) (Result, bool) {
	if g, ok := s.grouper.Push(pkt); ok {
		return s.group(g)
	}
	return Result{}, false
}

func (s *residueScanner) flush() (Result, bool) {
	if g, ok := s.grouper.Flush(); ok {
		return s.group(g)
	}
	return Result{}, false
}

func (s *residueScanner) Close() error { return nil }

func (s *residueScanner) group(g frame.Group) (Result, bool) {
	if g.Partial {
		return s.partial(g.Index)
	}
	pos, ok := s.seed(g.Index)
	if !ok {
		return Result{}, false
	}
	bits, err := stego.Recover(s.vc, pos, g.Packets)
	return s.open(g.Index, bits, err)
}

// sampleScanner reads bits from decoded PCM, which a lossless codec
// returns exactly as Embed left it.
type sampleScanner struct {
	*opener
	dec *av.Decoder
	win *windower
}

func (s *sampleScanner) push(pkt codec.Packet) (Result, bool) {
	frames, err := s.dec.Decode(pkt)
	if err != nil {
		return Result{}, false
	}
	return s.frames(frames)
}

func (s *sampleScanner) flush() (Result, bool) {
	_ = s.dec.Send(av.Packet{})
	tail, err := av.DrainPCM(s.dec)
	if err != nil {
		return Result{}, false
	}
	if res, ok := s.frames(tail); ok {
		return res, true
	}
	return s.read(s.win.flush())
}

func (s *sampleScanner) Close() error { return s.dec.Close() }

func (s *sampleScanner) frames(pcm []codec.PCM) (Result, bool) {
	for _, p := range pcm {
		if res, ok := s.read(s.win.push(p)); ok {
			return res, true
		}
	}
	return Result{}, false
}

// read scans windows in order, returning a Result once one completes: a
// window that alone authenticates a whole payload, or the last chunk of
// one assembled across several — opener tracks which. Intermediate spanned
// chunks authenticate but report no Result yet, so the loop keeps going.
func (s *sampleScanner) read(windows []sampleFrame) (Result, bool) {
	for _, w := range windows {
		if w.partial {
			s.partial(w.index)
			continue
		}
		pos, ok := s.seed(w.index)
		if !ok {
			continue
		}
		bits, err := stego.RecoverSamples(w.Samples, pos)
		if res, ok := s.open(w.index, bits, err); ok {
			return res, true
		}
	}
	return Result{}, false
}

// stream drains d in the background, emitting one Result per frame that
// decrypts. It owns d and closes it when the scan ends.
//
// A demuxer read already in flight cannot be interrupted — a live source
// may sit inside libav until the server sends more — so the scan is
// relayed through a second channel. That keeps the promise callers rely
// on: the channel they range over closes as soon as ctx does, whatever
// the reader is still waiting for.
func (c *Catcher) stream(ctx context.Context, d *av.Demuxer) (<-chan Result, error) {
	sc, err := newScanner(c.priv, d.Info(), c.logger())
	if err != nil {
		_ = d.Close()
		return nil, err
	}
	scanned := make(chan Result)
	go c.scan(ctx, d, sc, scanned)
	return relay(ctx, scanned), nil
}

// relay forwards results until in closes or ctx ends, closing its own
// channel either way. It is what lets Listen honour cancellation while the
// scan behind it is still parked inside a blocking read.
func relay(ctx context.Context, in <-chan Result) <-chan Result {
	out := make(chan Result)
	go func() {
		defer close(out)
		for {
			select {
			case res, ok := <-in:
				if !ok || !send(ctx, out, res) {
					return
				}
			case <-ctx.Done():
				return
			}
		}
	}()
	return out
}

func (c *Catcher) scan(ctx context.Context, d *av.Demuxer, sc scanner, out chan<- Result) {
	defer close(out)
	defer func() { _ = d.Close() }()
	defer func() { _ = sc.Close() }()

	r := retrier{left: c.maxRetries, backoff: c.backoff}
	for ctx.Err() == nil {
		pkt, err := d.NextPacket()
		if errors.Is(err, av.ErrEOF) {
			if res, ok := sc.flush(); ok {
				send(ctx, out, res)
			}
			return
		}
		if err != nil {
			if !r.wait(ctx) {
				return
			}
			continue
		}
		r.reset(c.maxRetries)
		if res, ok := sc.push(av.ToCodecPacket(pkt)); ok && !send(ctx, out, res) {
			return
		}
	}
}

func send(ctx context.Context, ch chan<- Result, r Result) bool {
	select {
	case ch <- r:
		return true
	case <-ctx.Done():
		return false
	}
}

// retrier absorbs transient read failures on a live source. It reports
// false once the budget is spent or ctx ends, which stops the scan.
type retrier struct {
	left    int
	backoff time.Duration
}

func (r *retrier) reset(n int) { r.left = n }

func (r *retrier) wait(ctx context.Context) bool {
	if r.left <= 0 {
		return false
	}
	r.left--
	if r.backoff <= 0 {
		return ctx.Err() == nil
	}
	t := time.NewTimer(r.backoff)
	defer t.Stop()
	select {
	case <-t.C:
		return true
	case <-ctx.Done():
		return false
	}
}

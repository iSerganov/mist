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

// scanner turns a demuxed packet stream into authenticated payloads. Every
// failure below the setup layer is a silent skip: a frame that carries no
// payload and a frame this key cannot open must be indistinguishable.
type scanner struct {
	priv    []byte
	pub     []byte
	log     *slog.Logger
	vc      *vorbis.Codec
	grouper *frame.Grouper
	joined  bool
}

func newScanner(priv []byte, info av.AudioInfo, log *slog.Logger) (*scanner, error) {
	pub, err := crypto.PublicFromPrivate(priv)
	if err != nil {
		return nil, ErrInvalidKey
	}
	if info.CodecID != av.CodecIDVorbis {
		return nil, fmt.Errorf("%w: extraction reads Vorbis residues", ErrUnsupportedCodec)
	}
	vc := vorbis.New()
	if err := vc.Load(info.Extradata); err != nil {
		return nil, fmt.Errorf("%w: setup: %v", ErrCarrier, err)
	}
	p := info.Params()
	return &scanner{
		priv: priv,
		pub:  pub,
		log:  log,
		vc:   vc,
		grouper: frame.NewGrouper(frame.Params{
			SampleRate: p.SampleRate,
			Channels:   p.Channels,
			Duration:   FrameDuration,
		}),
	}, nil
}

func (s *scanner) push(pkt codec.Packet) (Result, bool) {
	if g, ok := s.grouper.Push(pkt); ok {
		return s.open(g)
	}
	return Result{}, false
}

func (s *scanner) flush() (Result, bool) {
	if g, ok := s.grouper.Flush(); ok {
		return s.open(g)
	}
	return Result{}, false
}

func (s *scanner) open(g frame.Group) (Result, bool) {
	if g.Partial {
		if !s.joined {
			s.joined = true
			s.log.Info("joined in the middle of transmission, cannot decrypt", "frame", g.Index)
		}
		return Result{}, false
	}
	pos, err := crypto.PositionSeed(s.pub, g.Index)
	if err != nil {
		return Result{}, false
	}
	bits, err := stego.Recover(s.vc, pos, g.Packets)
	if err != nil {
		return Result{}, false
	}
	env, err := wire.UnmarshalEnvelope(bits)
	if err != nil {
		return Result{}, false
	}
	plain, err := crypto.Open(&env, s.priv)
	if err != nil {
		return Result{}, false
	}
	p, err := wire.UnmarshalPayload(plain)
	if err != nil {
		return Result{}, false
	}
	return Result{
		Payload:  Payload{Type: PayloadType(p.Type), Data: p.Data},
		FrameIdx: g.Index,
	}, true
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

func (c *Catcher) scan(ctx context.Context, d *av.Demuxer, sc *scanner, out chan<- Result) {
	defer close(out)
	defer func() { _ = d.Close() }()

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

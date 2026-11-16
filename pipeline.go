package mist

import (
	"context"
	"errors"
	"fmt"
	"io"
	"strings"

	"github.com/iSerganov/mist/internal/av"
	"github.com/iSerganov/mist/internal/codec"
)

// The embed pipeline takes a carrier through decode, embed, encode and mux
// one stego frame at a time, in one pass, so memory does not grow with the
// carrier and nothing is read twice. Each frame's share of the payload is
// decided when its room is known (spanPlan), so ErrNoCapacity can only be
// known once the carrier has ended, after the output is written.

// input is an opened carrier.
type input struct {
	d     *av.Demuxer
	close func()
	// live marks a URL that states no duration. A cancelled ctx is how it
	// normally ends, so what was already read is still written.
	live bool
}

// openPath opens a local path, a file:// URL, or an http(s):// URL. A URL
// goes to libav's own protocol layer, whose network IO gives up once ctx is
// cancelled.
func openPath(ctx context.Context, path string) (input, error) {
	if strings.HasPrefix(path, "http://") || strings.HasPrefix(path, "https://") {
		d, err := av.OpenDemuxer(ctx, path)
		if err != nil {
			return input{}, carrierErr(err)
		}
		return input{d: d, close: func() { _ = d.Close() }, live: d.Info().DurationUs <= 0}, nil
	}
	f, err := openSource(path)
	if err != nil {
		return input{}, err
	}
	c, err := openReader(f)
	if err != nil {
		_ = f.Close()
		return input{}, err
	}
	closeDemuxer := c.close
	c.close = func() { closeDemuxer(); _ = f.Close() }
	return c, nil
}

// openReader opens a carrier the caller has already opened and still owns.
func openReader(r io.Reader) (input, error) {
	d, err := av.OpenDemuxerReader(asSeeker(r))
	if err != nil {
		return input{}, carrierErr(err)
	}
	return input{d: d, close: func() { _ = d.Close() }}, nil
}

// next reads c's packets until it ends. A read cut short by ctx ends a
// live carrier the way its end of stream would.
func (c input) next(ctx context.Context) func() (av.Packet, error) {
	return func() (av.Packet, error) {
		var pkt av.Packet
		err := ctx.Err()
		if err == nil {
			pkt, err = c.d.NextPacket()
		}
		switch {
		case err == nil || errors.Is(err, av.ErrEOF):
			return pkt, err
		case ctx.Err() == nil:
			return av.Packet{}, carrierErr(err)
		case c.live:
			return av.Packet{}, av.ErrEOF
		default:
			return av.Packet{}, ctx.Err()
		}
	}
}

func carrierErr(err error) error { return fmt.Errorf("%w: %v", ErrCarrier, err) }

// placer says what each stego frame carries. place is called once per
// frame, in stream order, with the frame's room in bytes. It returns the
// framed bytes to seal into that frame, or nil for filler.
type placer interface {
	place(room int) []byte
}

// roomLog is the measuring placer: it records each room and places nothing.
type roomLog []int

func (r *roomLog) place(room int) []byte {
	*r = append(*r, room)
	return nil
}

// pass is what both stages need to know about the run they are part of.
type pass struct {
	e      *Emitter
	dst    io.Writer // nil when only measuring
	place  placer
	src    av.AudioInfo
	meta   av.Metadata
	frames int // frames the carrier's stated duration implies, 0 when unknown
	done   int
}

// report counts one more frame written.
func (p *pass) report() {
	p.done++
	total := p.frames
	if total > 0 {
		total = max(total, p.done)
	}
	p.e.progress.report(ProgressEmbed, p.done, total)
}

// stage is the half of the pipeline after the decoder, one per output
// domain. close releases what push opened, whether or not finish ran.
type stage interface {
	push(codec.PCM) error
	finish() error
	close()
}

// begin checks the request, then opens its carrier, so a bad key or
// payload fails before anything is read.
func (e *Emitter) begin(ctx context.Context, payload Payload, open func() (input, error)) (input, []byte, error) {
	if err := e.validate(ctx, payload); err != nil {
		return input{}, nil, err
	}
	plain, err := marshalPlain(payload, e.senderPriv)
	if err != nil {
		return input{}, nil, err
	}
	c, err := open()
	return c, plain, err
}

// embedTo embeds plain into c, which it closes, writing dst as it goes.
func (e *Emitter) embedTo(ctx context.Context, c input, dst io.Writer, plain []byte) error {
	plan := &spanPlan{full: plain, every: e.reseal}
	if err := e.run(ctx, c, pass{dst: dst, place: plan, frames: frameCount(c.d.Info().DurationUs)}); err != nil {
		return err
	}
	if !plan.complete() {
		return noCapacity(plain)
	}
	return nil
}

// measure reports every stego frame's room in c, which it closes, without
// writing anything.
func (e *Emitter) measure(ctx context.Context, c input) ([]int, error) {
	var rooms roomLog
	err := e.run(ctx, c, pass{place: &rooms})
	return rooms, err
}

// run is one pass over c, which it closes. A pass with no dst only
// measures: frames reach its placer, but nothing is embedded, encoded or
// written.
func (e *Emitter) run(ctx context.Context, c input, ps pass) error {
	defer c.close()
	meta, err := c.d.Metadata()
	if err != nil {
		return carrierErr(err)
	}
	ps.e, ps.src, ps.meta = e, c.d.Info(), meta
	var st stage = &residueStage{pass: ps}
	if e.target.Lossless {
		st = &sampleStage{pass: ps}
	}
	defer st.close()
	if err := decodeEach(ps.src, c.next(ctx), st.push); err != nil {
		return err
	}
	return st.finish()
}

// decodeEach decodes the packets next returns, until it reports av.ErrEOF,
// and hands fn each chunk in order.
func decodeEach(info av.AudioInfo, next func() (av.Packet, error), fn func(codec.PCM) error) error {
	if !av.CanDecode(info) {
		return fmt.Errorf("%w: this FFmpeg build cannot decode %s", ErrUnsupportedCodec, codecName(info))
	}
	dec, err := av.NewDecoder(info)
	if err != nil {
		return fmt.Errorf("%w: decoder", ErrCarrier)
	}
	defer func() { _ = dec.Close() }()
	drain := func() error {
		chunks, err := av.DrainPCM(dec)
		if err != nil {
			return carrierErr(err)
		}
		for _, c := range chunks {
			if len(c.Planes) == 0 {
				continue
			}
			if err := fn(c); err != nil {
				return err
			}
		}
		return nil
	}
	for {
		pkt, err := next()
		if errors.Is(err, av.ErrEOF) {
			break
		}
		if err != nil {
			return err
		}
		if err := dec.Send(pkt); err != nil && !errors.Is(err, av.ErrAgain) {
			return carrierErr(err)
		}
		if err := drain(); err != nil {
			return err
		}
	}
	_ = dec.Send(av.Packet{})
	return drain()
}

// packetWriter muxes packets to a writer as they are produced.
type packetWriter struct {
	m   *av.Muxer
	pts int64
}

// openMux writes the container header, with the carrier's tags, to w.
func openMux(w io.Writer, info av.AudioInfo, meta av.Metadata) (*packetWriter, error) {
	m, err := av.NewMuxer(w, info)
	if err != nil {
		return nil, fmt.Errorf("%w: muxer", err)
	}
	if err := writeTags(m, meta); err != nil {
		_ = m.Close()
		return nil, err
	}
	if err := m.WriteHeader(); err != nil {
		_ = m.Close()
		return nil, err
	}
	return &packetWriter{m: m}, nil
}

func (p *packetWriter) write(pkts []codec.Packet) error {
	for _, pkt := range pkts {
		if pkt.PTS == 0 && pkt.Duration == 0 {
			pkt.PTS = p.pts
			pkt.DTS = p.pts
			pkt.Duration = 64
		}
		p.pts = pkt.PTS + pkt.Duration
		if err := p.m.WritePacket(av.FromCodecPacket(pkt)); err != nil {
			return err
		}
	}
	return nil
}

func (p *packetWriter) finish() error {
	if err := p.m.WriteTrailer(); err != nil {
		return err
	}
	return p.m.Close()
}

func (p *packetWriter) close() { _ = p.m.Close() }

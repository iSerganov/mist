package mist

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"

	"github.com/iSerganov/mist/internal/av"
	"github.com/iSerganov/mist/internal/codec"
	"github.com/iSerganov/mist/internal/codec/vorbis"
	"github.com/iSerganov/mist/internal/crypto"
	"github.com/iSerganov/mist/internal/stego"
	"github.com/iSerganov/mist/internal/wire"
)

func extractReader(ctx context.Context, r io.Reader, priv []byte) ([]Result, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if len(priv) != PrivateKeySize {
		return nil, ErrInvalidKey
	}
	pub, err := crypto.PublicFromPrivate(priv)
	if err != nil {
		return nil, ErrInvalidKey
	}
	d, err := av.OpenDemuxerReader(asSeeker(r))
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrCarrier, err)
	}
	defer d.Close()
	vc := vorbis.New()
	if err := vc.Load(d.Info().Extradata); err != nil {
		return nil, fmt.Errorf("%w: setup", err)
	}
	var pkts []codec.Packet
	for {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		pkt, err := d.NextPacket()
		if errors.Is(err, av.ErrEOF) {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("%w: %v", ErrCarrier, err)
		}
		pkts = append(pkts, av.ToCodecPacket(pkt))
	}
	if len(pkts) == 0 {
		return nil, nil
	}
	pos, err := crypto.PositionSeed(pub, 0)
	if err != nil {
		return nil, err
	}
	bits, err := stego.Recover(vc, pos, pkts)
	if err != nil {
		return nil, nil
	}
	if res, ok := openBits(bits, priv); ok {
		return []Result{{Payload: res, FrameIdx: 0}}, nil
	}
	return nil, nil
}

func openBits(bits []byte, priv []byte) (Payload, bool) {
	for n := len(bits); n >= wire.EnvelopeOverhead; n-- {
		env, err := wire.UnmarshalEnvelope(bits[:n])
		if err != nil {
			continue
		}
		plain, err := crypto.Open(&env, priv)
		if err != nil {
			continue
		}
		wp, err := wire.UnmarshalPayload(plain)
		if err != nil {
			continue
		}
		return Payload{Type: PayloadType(wp.Type), Data: wp.Data}, true
	}
	return Payload{}, false
}

func extractFile(ctx context.Context, f *os.File, priv []byte) ([]Result, error) {
	if f == nil {
		return nil, ErrInvalidSource
	}
	if _, err := f.Seek(0, io.SeekStart); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrCarrier, err)
	}
	return extractReader(ctx, f, priv)
}

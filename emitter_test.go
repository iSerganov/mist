package mist

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/iSerganov/mist/internal/av"
	"github.com/iSerganov/mist/internal/codec"
	"github.com/iSerganov/mist/internal/codec/vorbis"
	"github.com/iSerganov/mist/internal/stego"
	"github.com/stretchr/testify/suite"
)

type EmitterSuite struct {
	audioSuite
}

func TestEmitterSuite(t *testing.T) {
	suite.Run(t, &EmitterSuite{})
}

func (s *EmitterSuite) TestNewEmitterRejectsNilKey() {
	e, err := NewEmitter(nil)
	s.Nil(e)
	s.ErrorIs(err, ErrInvalidKey)
}

func (s *EmitterSuite) TestNewEmitterCopiesPublicKey() {
	key := []byte{1, 2, 3}
	e, err := NewEmitter(key)
	s.Require().NoError(err)
	key[0] = 9
	s.Equal(byte(1), e.pub[0])
	s.Len(e.pub, 3)
}

func (s *EmitterSuite) TestWithSenderAuthCopiesKey() {
	sender := []byte{4, 5, 6}
	e, err := NewEmitter([]byte{1}, WithSenderAuth(sender))
	s.Require().NoError(err)
	sender[0] = 9
	s.Equal(byte(4), e.senderPriv[0])
}

func (s *EmitterSuite) TestFrameCapacity() {
	s.Greater(FrameCapacity(), 0)
}

func (s *EmitterSuite) TestEmbedReaderRejectsShortKey() {
	e, err := NewEmitter([]byte{1, 2, 3})
	s.Require().NoError(err)
	_, err = e.EmbedReader(context.Background(), bytes.NewReader(nil), Text("hello"))
	s.ErrorIs(err, ErrInvalidKey)
}

func (s *EmitterSuite) TestApplyRecoverSamePackets() {
	s.requireLibav()
	c := vorbis.New()
	enc, err := c.NewEncoder(codec.DefaultVorbis)
	s.Require().NoError(err)
	defer func() { _ = enc.Close() }()
	info := enc.(interface{ Params() codec.Params }).Params()
	s.Require().NoError(c.Load(info.Extradata))
	pcm := testSine(44100, 2, 8192, 440)
	pkts, err := enc.Encode(pcm)
	s.Require().NoError(err)
	fl, err := enc.Flush()
	s.Require().NoError(err)
	pkts = append(pkts, fl...)

	var nRes, nElig int
	for _, pkt := range pkts {
		if len(pkt.Data) == 0 || pkt.Data[0]&1 == 1 {
			continue
		}
		r, err := c.Residues(pkt)
		s.Require().NoError(err)
		nRes += len(r)
		nElig += len(r)
	}
	nbits := int(float64(nElig) * 0.10)
	s.T().Logf("packets=%d residues=%d eligible=%d nbits=%d (%d bytes)", len(pkts), nRes, nElig, nbits, nbits/8)
	s.T().Logf("capacity bytes=%d first_val=%d", nbits/8, func() int32 {
		for _, pkt := range pkts {
			if len(pkt.Data) == 0 || pkt.Data[0]&1 == 1 {
				continue
			}
			r, err := c.Residues(pkt)
			if err == nil && len(r) > 0 {
				return r[0].Value
			}
		}
		return 0
	}())
	s.Greater(nbits/8, 4)

	payload := make([]byte, 4)
	for i := range payload {
		payload[i] = byte(i)
	}
	pos := make([]byte, 32)
	pos[0] = 7
	out, err := stego.Apply(c, pos, pkts, payload)
	s.Require().NoError(err)
	changed, skippedIn, skippedOut := 0, 0, 0
	for i := range pkts {
		if len(pkts[i].Data) == 0 || pkts[i].Data[0]&1 == 1 {
			skippedIn++
		}
		if i < len(out) && (len(out[i].Data) == 0 || out[i].Data[0]&1 == 1) {
			skippedOut++
		}
		if i < len(out) && !bytes.Equal(pkts[i].Data, out[i].Data) {
			changed++
		}
	}
	s.T().Logf("changed=%d skipped %d→%d", changed, skippedIn, skippedOut)
	raw, err := stego.Recover(c, pos, pkts)
	s.Require().NoError(err)
	got, err := stego.Recover(c, pos, out)
	s.T().Logf("recover orig=%x out=%x", []byte(raw[:4]), []byte(got[:4]))
	s.Require().NoError(err)
	s.Require().GreaterOrEqual(len(got), len(payload))
	s.Equal(payload, []byte(got[:len(payload)]))

	rc, err := muxPackets(info, out)
	s.Require().NoError(err)
	defer func() { _ = rc.Close() }()
	ogg, err := io.ReadAll(rc)
	s.Require().NoError(err)
	d, err := av.OpenDemuxerReader(bytes.NewReader(ogg))
	s.Require().NoError(err)
	defer func() { _ = d.Close() }()
	vc := vorbis.New()
	s.Require().NoError(vc.Load(d.Info().Extradata))
	var demuxed []codec.Packet
	for {
		pkt, err := d.NextPacket()
		if err != nil {
			break
		}
		demuxed = append(demuxed, av.ToCodecPacket(pkt))
	}
	s.T().Logf("mux packets=%d demux packets=%d extra=%d/%d", len(out), len(demuxed), len(info.Extradata), len(d.Info().Extradata))
	got2, err := stego.Recover(vc, pos, demuxed)
	s.Require().NoError(err)
	s.Require().GreaterOrEqual(len(got2), len(payload))
	s.Equal(payload, []byte(got2[:len(payload)]))
}

func (s *EmitterSuite) TestEmbedRoundTrip() {
	s.requireLibav()
	pub, priv, err := GenerateKeyPair()
	s.Require().NoError(err)
	carrier := s.makeCarrier()

	em, err := NewEmitter(pub)
	s.Require().NoError(err)
	out, err := em.EmbedReader(context.Background(), bytes.NewReader(carrier), Text("hello mist"))
	s.Require().NoError(err)
	s.Require().NotNil(out)
	defer func() { _ = out.Close() }()
	ogg, err := io.ReadAll(out)
	s.Require().NoError(err)
	s.Greater(len(ogg), 100)

	// The result is a demuxable Ogg Vorbis file.
	d, err := av.OpenDemuxerReader(bytes.NewReader(ogg))
	s.Require().NoError(err)
	s.Equal(av.CodecIDVorbis, d.Info().CodecID)
	s.NoError(d.Close())

	path := s.writeTemp(ogg)
	f, err := os.Open(path)
	s.Require().NoError(err)
	defer func() { _ = f.Close() }()
	catch, err := NewCatcher(priv)
	s.Require().NoError(err)
	got, err := catch.Extract(context.Background(), f)
	s.Require().NoError(err)
	s.Require().NotEmpty(got)
	s.Equal("hello mist", string(got[0].Payload.Data))
	s.Equal(PayloadText, got[0].Payload.Type)
}

func (s *EmitterSuite) TestEmbedFile() {
	s.requireLibav()
	pub, priv, err := GenerateKeyPair()
	s.Require().NoError(err)
	path := s.writeTemp(s.makeCarrier())
	f, err := os.Open(path)
	s.Require().NoError(err)
	defer func() { _ = f.Close() }()
	em, err := NewEmitter(pub)
	s.Require().NoError(err)
	out, err := em.EmbedFile(context.Background(), f, Text("file"))
	s.Require().NoError(err)
	defer func() { _ = out.Close() }()
	ogg, err := io.ReadAll(out)
	s.Require().NoError(err)
	stegoPath := s.writeTemp(ogg)
	sf, err := os.Open(stegoPath)
	s.Require().NoError(err)
	defer func() { _ = sf.Close() }()
	catch, err := NewCatcher(priv)
	s.Require().NoError(err)
	got, err := catch.Extract(context.Background(), sf)
	s.Require().NoError(err)
	s.Require().NotEmpty(got)
	s.Equal("file", string(got[0].Payload.Data))
}

func (s *EmitterSuite) TestEmbed() {
	s.requireLibav()
	pub, _, err := GenerateKeyPair()
	s.Require().NoError(err)
	path := s.writeTemp(s.makeCarrier())
	em, err := NewEmitter(pub)
	s.Require().NoError(err)
	out, err := em.Embed(context.Background(), path, Text("path"))
	s.Require().NoError(err)
	defer func() { _ = out.Close() }()
	ogg, err := io.ReadAll(out)
	s.Require().NoError(err)
	s.Greater(len(ogg), 100)
}

func (s *EmitterSuite) TestEmbedHTTP() {
	s.requireLibav()
	pub, priv, err := GenerateKeyPair()
	s.Require().NoError(err)
	carrier := s.makeCarrier()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// libav's http protocol sends a Range request and relies on a
		// proper Content-Length/206 response to detect EOF cleanly;
		// http.ServeContent (unlike a bare w.Write) handles that.
		w.Header().Set("Content-Type", "audio/ogg")
		http.ServeContent(w, r, "carrier.ogg", time.Time{}, bytes.NewReader(carrier))
	}))
	defer srv.Close()

	em, err := NewEmitter(pub)
	s.Require().NoError(err)
	out, err := em.Embed(context.Background(), srv.URL+"/carrier.ogg", Text("over http"))
	s.Require().NoError(err)
	defer func() { _ = out.Close() }()
	ogg, err := io.ReadAll(out)
	s.Require().NoError(err)
	s.Greater(len(ogg), 100)

	path := s.writeTemp(ogg)
	f, err := os.Open(path)
	s.Require().NoError(err)
	defer func() { _ = f.Close() }()
	catch, err := NewCatcher(priv)
	s.Require().NoError(err)
	got, err := catch.Extract(context.Background(), f)
	s.Require().NoError(err)
	s.Require().NotEmpty(got)
	s.Equal("over http", string(got[0].Payload.Data))
}

func (s *EmitterSuite) TestEmbedRejectsUnknownScheme() {
	s.requireLibav()
	pub, _, err := GenerateKeyPair()
	s.Require().NoError(err)
	em, err := NewEmitter(pub)
	s.Require().NoError(err)
	_, err = em.Embed(context.Background(), "http://127.0.0.1:1/does-not-exist.ogg", Text("x"))
	s.Error(err)
}

// Skipping a frame with no room is right for a trailing sliver, but a
// carrier where every frame is skipped means nothing was embedded at all,
// and the caller must hear about it rather than get a silent no-op file.
func (s *EmitterSuite) TestEmbedRejectsCarrierWithoutCapacity() {
	s.requireLibav()
	pub, _, err := GenerateKeyPair()
	s.Require().NoError(err)
	em, err := NewEmitter(pub)
	s.Require().NoError(err)

	tests := []struct {
		title   string
		payload Payload
	}{
		{"payload larger than any frame", Text(strings.Repeat("x", 1<<20))},
	}
	for _, tc := range tests {
		s.Run(tc.title, func() {
			_, err := em.EmbedReader(context.Background(),
				bytes.NewReader(s.carrier(FrameDuration)), tc.payload)
			s.ErrorIs(err, ErrNoCapacity)
		})
	}
}

func (s *EmitterSuite) makeCarrier() []byte { return s.carrier(FrameDuration) }

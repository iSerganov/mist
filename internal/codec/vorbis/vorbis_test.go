package vorbis

import (
	"encoding/binary"
	"math"
	"testing"

	"github.com/iSerganov/mist/internal/av"
	"github.com/iSerganov/mist/internal/codec"
	"github.com/stretchr/testify/suite"
)

type VorbisSuite struct {
	suite.Suite
}

func TestVorbisSuite(t *testing.T) {
	suite.Run(t, &VorbisSuite{})
}

func (s *VorbisSuite) requireLibav() {
	if !av.Available() {
		s.T().Skip("libav Vorbis encoder not available")
	}
}

func (s *VorbisSuite) TestCodecIdentity() {
	c := New()
	s.Equal(Name, c.Name())
	s.Equal(codec.IDVorbis, c.ID())
}

func (s *VorbisSuite) TestResidues() {
	c := New()
	tests := []struct {
		title string
		data  []byte
		want  error
	}{
		{"empty", nil, ErrBadPacket},
		{"ident header", makeIdent(2, 44100, 8, 11), ErrBadPacket},
		{"comment header", []byte{headerComment, 'v', 'o', 'r', 'b', 'i', 's'}, ErrBadPacket},
		{"setup header", []byte{headerSetup, 'v', 'o', 'r', 'b', 'i', 's'}, ErrBadPacket},
		{"audio without codebooks", []byte{0x00, 0x01, 0x02}, ErrBadSetup},
	}
	for _, tc := range tests {
		s.Run(tc.title, func() {
			_, err := c.Residues(codec.Packet{Data: tc.data})
			s.ErrorIs(err, tc.want)
		})
	}
}

func (s *VorbisSuite) TestParseSetup() {
	ident := makeIdent(2, 48000, 8, 11)
	comment := []byte{headerComment, 'v', 'o', 'r', 'b', 'i', 's'}
	setup := []byte{headerSetup, 'v', 'o', 'r', 'b', 'i', 's'}

	tests := []struct {
		title            string
		ident, comm, set []byte
		want             error
	}{
		{"ok", ident, comment, setup, nil},
		{"short ident", ident[:10], comment, setup, ErrBadPacket},
		{"bad ident magic", append([]byte{1}, []byte("vorbix")...), comment, setup, ErrBadPacket},
		{"wrong ident type", append([]byte{3}, ident[1:]...), comment, setup, ErrBadPacket},
		{"zero rate", makeIdent(2, 0, 8, 11), comment, setup, ErrBadSetup},
		{"zero channels", makeIdent(0, 44100, 8, 11), comment, setup, ErrBadSetup},
		{"bad blocksize", makeIdent(2, 44100, 3, 11), comment, setup, ErrBadSetup},
		{"bad comment type", ident, []byte{1, 'v', 'o', 'r', 'b', 'i', 's'}, setup, ErrBadPacket},
		{"short comment", ident, []byte{3}, setup, ErrBadPacket},
		{"bad setup magic", ident, comment, []byte{5, 'v', 'o', 'r', 'b', 'i', 'x'}, ErrBadPacket},
	}
	for _, tc := range tests {
		s.Run(tc.title, func() {
			got, err := ParseSetup(tc.ident, tc.comm, tc.set)
			if tc.want != nil {
				s.ErrorIs(err, tc.want)
				s.Nil(got)
				return
			}
			s.Require().NoError(err)
			s.Equal(2, got.Channels)
			s.Equal(48000, got.Rate)
			s.Equal(256, got.Blocksize0)
			s.Equal(2048, got.Blocksize1)
			s.Equal(ident, got.Ident)
			s.Equal(comment, got.Comment)
			s.Equal(setup, got.Codebooks)
		})
	}
}

func (s *VorbisSuite) TestEncodeDecode() {
	s.requireLibav()
	c := New()
	enc, err := c.NewEncoder(codec.DefaultVorbis)
	s.Require().NoError(err)
	defer enc.Close()

	p := enc.(*encoder).Params()
	s.NotEmpty(p.Extradata)

	pcm := sinePCM(44100, 2, 8192, 440)
	var pkts []codec.Packet
	for i := 0; i < 3; i++ {
		out, err := enc.Encode(pcm)
		s.Require().NoError(err)
		pkts = append(pkts, out...)
	}
	flushed, err := enc.Flush()
	s.Require().NoError(err)
	pkts = append(pkts, flushed...)
	s.Require().NotEmpty(pkts)

	dec, err := c.NewDecoder(p)
	s.Require().NoError(err)
	defer dec.Close()
	var frames []codec.PCM
	for _, pkt := range pkts {
		out, err := dec.Decode(pkt)
		s.Require().NoError(err)
		frames = append(frames, out...)
	}
	out, err := dec.Decode(codec.Packet{})
	s.Require().NoError(err)
	frames = append(frames, out...)
	s.Require().NotEmpty(frames)
	s.Equal(2, frames[0].Channels)
	s.Equal(44100, frames[0].SampleRate)
	s.NotEmpty(frames[0].Planes)
}

func (s *VorbisSuite) TestLoadAndResidues() {
	s.requireLibav()
	c := New()
	enc, err := c.NewEncoder(codec.DefaultVorbis)
	s.Require().NoError(err)
	defer enc.Close()
	p := enc.(*encoder).Params()
	s.Require().NotEmpty(p.Extradata)
	n := 16
	if len(p.Extradata) < n {
		n = len(p.Extradata)
	}
	s.T().Logf("extradata len=%d head=%v", len(p.Extradata), p.Extradata[:n])
	ident, comment, setup, err := SplitExtradata(p.Extradata)
	s.Require().NoError(err)
	s.T().Logf("ident=%d t=%d comment=%d t=%d setup=%d t=%d ident28=%#x ident29=%#x",
		len(ident), ident[0], len(comment), comment[0], len(setup), setup[0], ident[28], ident[29])
	id, err := parseIdent(ident)
	s.Require().NoError(err)
	s.T().Logf("ch=%d rate=%d bs=%d/%d", id.Channels, id.Rate, id.Blocksize0, id.Blocksize1)
	_, err = ParseSetup(ident, comment, setup)
	s.Require().NoError(err)
	s.Require().NoError(c.Load(p.Extradata))
	s.Require().NotNil(c.Setup())
	s.Greater(len(c.Setup().tables.books), 0)

	pcm := sinePCM(44100, 2, 8192, 440)
	var pkts []codec.Packet
	for i := 0; i < 4; i++ {
		out, err := enc.Encode(pcm)
		s.Require().NoError(err)
		pkts = append(pkts, out...)
	}
	flushed, err := enc.Flush()
	s.Require().NoError(err)
	pkts = append(pkts, flushed...)
	s.Require().NotEmpty(pkts)

	var got []codec.Residue
	ok, fail := 0, 0
	for _, pkt := range pkts {
		res, err := c.Residues(pkt)
		if err != nil {
			fail++
			s.T().Logf("residues fail: %v len=%d b0=%d", err, len(pkt.Data), pkt.Data[0])
			continue
		}
		ok++
		if len(res) > 0 && len(got) == 0 {
			got = res
		}
	}
	s.T().Logf("residues ok=%d fail=%d", ok, fail)
	s.Greater(ok, 0)
	s.Require().NotEmpty(got)

	// Identity rewrite must decode to the same rounded residues.
	same, err := c.Residues(pkts[0])
	s.Require().NoError(err)
	rewrittenIdent, err := c.Rewrite(pkts[0], same)
	s.Require().NoError(err)
	again, err := c.Residues(rewrittenIdent)
	s.Require().NoError(err)
	s.Require().Len(again, len(same))
	for i := range same {
		s.Equal(same[i].Value, again[i].Value, "identity residue %d", i)
	}

	// Flip codebook-entry LSBs and confirm they survive Rewrite.
	flipped := 0
	want := append([]codec.Residue(nil), same...)
	for i := range want {
		want[i].Value = want[i].Value + 1
		flipped++
		if flipped == 8 {
			break
		}
	}
	s.Require().Greater(flipped, 0)
	rewritten, err := c.Rewrite(pkts[0], want)
	s.Require().NoError(err)
	got2, err := c.Residues(rewritten)
	s.Require().NoError(err)
	s.Require().Len(got2, len(want))
	okLSB := 0
	for i := range want {
		if same[i].Value == want[i].Value {
			continue
		}
		if got2[i].Value&1 == want[i].Value&1 {
			okLSB++
		}
	}
	s.Equal(flipped, okLSB)
}

func (s *VorbisSuite) TestRewriteAllEntryLSBs() {
	s.requireLibav()
	c := New()
	enc, err := c.NewEncoder(codec.DefaultVorbis)
	s.Require().NoError(err)
	defer enc.Close()
	p := enc.(*encoder).Params()
	s.Require().NoError(c.Load(p.Extradata))
	pcm := sinePCM(44100, 2, 8192, 440)
	pkts, err := enc.Encode(pcm)
	s.Require().NoError(err)
	fl, err := enc.Flush()
	s.Require().NoError(err)
	pkts = append(pkts, fl...)
	var n, okLSB, sameLen int
	for _, pkt := range pkts {
		if len(pkt.Data) == 0 || pkt.Data[0]&1 == 1 {
			continue
		}
		res, err := c.Residues(pkt)
		if err != nil || len(res) == 0 {
			continue
		}
		want := append([]codec.Residue(nil), res...)
		for i := range want {
			want[i].Value++
		}
		rew, err := c.Rewrite(pkt, want)
		s.Require().NoError(err)
		got, err := c.Residues(rew)
		s.Require().NoError(err)
		if len(got) == len(want) {
			sameLen++
		}
		for i := range want {
			if i >= len(got) {
				break
			}
			n++
			if got[i].Value&1 == want[i].Value&1 {
				okLSB++
			}
		}
	}
	s.T().Logf("packets=%d compared=%d lsb_ok=%d", sameLen, n, okLSB)
	s.Equal(n, okLSB)
}

func (s *VorbisSuite) TestVQStepCount() {
	s.requireLibav()
	c := New()
	enc, err := c.NewEncoder(codec.DefaultVorbis)
	s.Require().NoError(err)
	defer enc.Close()
	p := enc.(*encoder).Params()
	s.Require().NoError(c.Load(p.Extradata))
	pcm := sinePCM(44100, 2, 44100, 440)
	pkts, err := enc.Encode(pcm)
	s.Require().NoError(err)
	fl, err := enc.Flush()
	s.Require().NoError(err)
	pkts = append(pkts, fl...)
	var steps, inBand int
	for _, pkt := range pkts {
		if len(pkt.Data) == 0 || pkt.Data[0]&1 == 1 {
			continue
		}
		st, err := decodeAudio(c.setup, pkt.Data)
		s.Require().NoError(err)
		for _, prog := range st.progs {
			for _, sym := range prog.syms {
				if sym.class {
					continue
				}
				steps++
				ch, spec := prog.coord(sym, 0)
				hz := 0
				if st.n > 0 && c.setup.Rate > 0 && spec >= 0 {
					hz = spec * c.setup.Rate / (st.n * 2)
				}
				_ = ch
				if hz >= 8000 && hz < 16000 {
					inBand++
				}
			}
		}
	}
	s.T().Logf("vq steps=%d in-band=%d bits@10%%=%d bytes@10%%=%d", steps, inBand, inBand/10, inBand/80)
	s.Greater(steps, 0)
}

func (s *VorbisSuite) TestNewDecoderRejectsEmptyExtradata() {
	_, err := New().NewDecoder(codec.DefaultVorbis)
	s.ErrorIs(err, ErrBadSetup)
}

func makeIdent(ch int, rate uint32, bs0, bs1 byte) []byte {
	b := make([]byte, identSize)
	b[0] = headerIdent
	copy(b[1:7], magic)
	b[11] = byte(ch)
	binary.LittleEndian.PutUint32(b[12:16], rate)
	b[28] = (bs1 << 4) | (bs0 & 0x0f)
	b[29] = 1
	return b
}

func sinePCM(rate, ch, n int, freq float64) codec.PCM {
	planes := make([][]float32, ch)
	for c := 0; c < ch; c++ {
		planes[c] = make([]float32, n)
		for i := 0; i < n; i++ {
			planes[c][i] = float32(math.Sin(2 * math.Pi * freq * float64(i) / float64(rate)))
		}
	}
	return codec.PCM{
		Planes:     planes,
		NbSamples:  n,
		Channels:   ch,
		SampleRate: rate,
		Format:     codec.SampleFmtFLTP,
	}
}

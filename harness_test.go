//go:build harness

package mist

import (
	"bytes"
	"cmp"
	"context"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/iSerganov/mist/internal/av"
	"github.com/iSerganov/mist/internal/codec/vorbis"
	"github.com/iSerganov/mist/internal/quality"
	"github.com/iSerganov/mist/internal/steganalysis"
	"github.com/iSerganov/mist/internal/stego"
	"github.com/stretchr/testify/suite"
)

const (
	harnessChunk  = 1 << 16
	harnessRounds = 1000
	harnessSeed   = 1
)

var defaultHarnessFormats = []string{"ogg", "flac", "wav", "caf/alac", "wv", "tta", "aiff"}

type HarnessSuite struct {
	suite.Suite
}

func TestHarnessSuite(t *testing.T) {
	suite.Run(t, &HarnessSuite{})
}

func (s *HarnessSuite) TestMeasure() {
	if !av.Available() {
		s.T().Skip("libav not available")
	}
	corpus := os.Getenv("MIST_CORPUS")
	carriers, err := loadCarriers(corpus)
	s.Require().NoError(err)
	s.Require().NotEmpty(carriers)
	baseline, err := readReport(os.Getenv("MIST_HARNESS_BASELINE"))
	s.Require().NoError(err)
	pub, _, err := GenerateKeyPair()
	s.Require().NoError(err)

	tool := perceptualTool()
	rep := harnessReport{
		Commit:   commit(),
		Date:     time.Now().Format(time.DateOnly),
		Corpus:   cmp.Or(corpus, "synthetic"),
		Carriers: len(carriers),
	}
	if tool != nil {
		rep.Perceptual = tool.Name
	}
	for _, spec := range harnessFormats(os.Getenv("MIST_HARNESS_FORMATS")) {
		s.T().Logf("measuring %s", spec)
		rep.Formats = append(rep.Formats, measureFormat(s.T().Context(), spec, carriers, pub, tool))
	}
	md, err := rep.write(cmp.Or(os.Getenv("MIST_HARNESS_OUT"), "harness-out"), baseline)
	s.Require().NoError(err)
	s.T().Log("\n" + md)
}

func harnessFormats(spec string) []string {
	switch spec {
	case "":
		return defaultHarnessFormats
	case "all":
		var out []string
		for _, f := range Formats() {
			out = append(out, f.String())
		}
		return out
	}
	return strings.Split(spec, ",")
}

type carrierRun struct {
	clean, stego, minimal []int32
	transcode, stegoSDR   float64
	perceptual            float64
}

func measureFormat(ctx context.Context, spec string, carriers []harnessCarrier, pub []byte, tool *quality.Tool) formatReport {
	name, codecName, _ := strings.Cut(strings.TrimSpace(spec), "/")
	fr := formatReport{Format: spec}
	f, err := LookupFormat(name, codecName)
	if err != nil {
		fr.Error = err.Error()
		return fr
	}
	fr.Format = f.String()
	em, err := NewEmitter(pub, WithFormat(name), WithCodec(codecName))
	if err != nil {
		fr.Error = err.Error()
		return fr
	}
	clean, stegoScores, minimal := scores{}, scores{}, scores{}
	var transcode, stegoSDR, gap, drop []float64
	for _, c := range carriers {
		run, err := measureCarrier(ctx, em, f.Ext, c, tool)
		if err != nil {
			fr.Skipped = append(fr.Skipped, skippedCarrier{Name: c.name, Reason: err.Error()})
			continue
		}
		fr.Measured++
		clean.add(run.clean)
		stegoScores.add(run.stego)
		minimal.add(run.minimal)
		transcode = append(transcode, run.transcode)
		stegoSDR = append(stegoSDR, run.stegoSDR)
		gap = append(gap, run.transcode-run.stegoSDR)
		drop = append(drop, run.perceptual)
	}
	if fr.Measured == 0 {
		return fr
	}
	for _, d := range steganalysis.Detectors() {
		pos, neg := stegoScores[d.Name], clean[d.Name]
		lo, hi := steganalysis.AUCInterval(pos, neg, harnessRounds, harnessSeed)
		fr.Detectors = append(fr.Detectors, detectorResult{
			Name:       d.Name,
			AUC:        steganalysis.AUC(pos, neg),
			Lo:         lo,
			Hi:         hi,
			Invariance: steganalysis.AUC(pos, minimal[d.Name]),
		})
	}
	fr.Transcode = summarize(transcode, minOf)
	fr.Stego = summarize(stegoSDR, minOf)
	fr.Gap = summarize(gap, maxOf)
	if tool != nil {
		s := summarize(drop, maxOf)
		fr.PerceptualDrop = &s
	}
	return fr
}

func measureCarrier(ctx context.Context, em *Emitter, ext string, c harnessCarrier, tool *quality.Tool) (carrierRun, error) {
	data, err := c.load()
	if err != nil {
		return carrierRun{}, err
	}
	ref, info, err := decodeCarrier(bytes.NewReader(data))
	if err != nil {
		return carrierRun{}, err
	}
	clean, err := cleanTwin(em.target, data)
	if err != nil {
		return carrierRun{}, err
	}
	stegoOut, err := embedBytes(ctx, em, data, Payload{Type: PayloadText, Data: bytes.Repeat([]byte("mist"), 16)})
	if err != nil {
		return carrierRun{}, err
	}
	minimalOut, err := embedBytes(ctx, em, data, Text("m"))
	if err != nil {
		return carrierRun{}, err
	}
	cleanPCM, cleanVals, err := inspect(clean)
	if err != nil {
		return carrierRun{}, err
	}
	stegoPCM, stegoVals, err := inspect(stegoOut)
	if err != nil {
		return carrierRun{}, err
	}
	_, minimalVals, err := inspect(minimalOut)
	if err != nil {
		return carrierRun{}, err
	}
	maxLag := info.SampleRate / 10
	run := carrierRun{
		clean:     cleanVals,
		stego:     stegoVals,
		minimal:   minimalVals,
		transcode: quality.SDR(ref.Planes, cleanPCM, maxLag),
		stegoSDR:  quality.SDR(ref.Planes, stegoPCM, maxLag),
	}
	if tool != nil {
		run.perceptual, err = perceptualDrop(ctx, *tool, c.name, ext, data, clean, stegoOut)
	}
	return run, err
}

func cleanTwin(target av.Format, data []byte) ([]byte, error) {
	pcm, info, err := decodeCarrier(bytes.NewReader(data))
	if err != nil {
		return nil, err
	}
	enc, pcm, err := openEncoder(target, pcm, info)
	if err != nil {
		return nil, err
	}
	defer func() { _ = enc.Close() }()
	rc, err := encodeAndMux(enc, pcm, nil)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rc.Close() }()
	return io.ReadAll(rc)
}

func embedBytes(ctx context.Context, em *Emitter, data []byte, p Payload) ([]byte, error) {
	rc, err := em.EmbedReader(ctx, bytes.NewReader(data), p)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rc.Close() }()
	return io.ReadAll(rc)
}

func inspect(out []byte) ([][]float32, []int32, error) {
	pcm, info, err := decodeCarrier(bytes.NewReader(out))
	if err != nil {
		return nil, nil, err
	}
	if av.Lossless(info.NativeCodecID) {
		return pcm.Planes, gridValues(pcm.Planes, av.SampleScale(info.SampleFmt)), nil
	}
	_, pkts, err := readPackets(bytes.NewReader(out))
	if err != nil {
		return nil, nil, err
	}
	vc := vorbis.New()
	if err := vc.Load(info.Extradata); err != nil {
		return nil, nil, err
	}
	vals, err := stego.EligibleValues(vc, pkts)
	return pcm.Planes, vals, err
}

func gridValues(planes [][]float32, scale float32) []int32 {
	var out []int32
	for _, p := range planes {
		s := stego.Samples{Planes: [][]float32{p}, N: len(p), Scale: scale}
		for i := range s.Len() {
			out = append(out, s.At(i))
		}
	}
	return out
}

type scores map[string][]float64

func (sc scores) add(v []int32) {
	for _, d := range steganalysis.Detectors() {
		for off := 0; off < len(v); off += harnessChunk {
			sc[d.Name] = append(sc[d.Name], d.Score(v[off:min(off+harnessChunk, len(v))]))
		}
	}
}

func perceptualTool() *quality.Tool {
	for _, t := range []quality.Tool{quality.ViSQOL(), quality.PEAQ()} {
		if t.Available() {
			return &t
		}
	}
	return nil
}

func perceptualDrop(ctx context.Context, tool quality.Tool, name, ext string, carrier, clean, stegoOut []byte) (float64, error) {
	dir, err := os.MkdirTemp("", "mist-harness-*")
	if err != nil {
		return 0, err
	}
	defer func() { _ = os.RemoveAll(dir) }()
	files := map[string][]byte{
		"carrier" + filepath.Ext(name): carrier,
		"clean." + ext:                 clean,
		"stego." + ext:                 stegoOut,
	}
	for f, b := range files {
		if err := os.WriteFile(filepath.Join(dir, f), b, 0o600); err != nil {
			return 0, err
		}
	}
	ref := filepath.Join(dir, "carrier"+filepath.Ext(name))
	cleanScore, err := tool.Score(ctx, ref, filepath.Join(dir, "clean."+ext))
	if err != nil {
		return 0, err
	}
	stegoScore, err := tool.Score(ctx, ref, filepath.Join(dir, "stego."+ext))
	return cleanScore - stegoScore, err
}

func commit() string {
	out, err := exec.Command("git", "describe", "--always", "--dirty").Output()
	if err != nil {
		return "unknown"
	}
	return strings.TrimSpace(string(out))
}

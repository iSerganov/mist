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
	"runtime"
	"strconv"
	"strings"
	"sync"
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
	harnessFolds  = 5
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

	jobs, err := harnessJobs(os.Getenv("MIST_HARNESS_JOBS"))
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
		rep.Formats = append(rep.Formats, measureFormat(s.T().Context(), spec, carriers, pub, tool, jobs))
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

func harnessJobs(spec string) (int, error) {
	if spec == "" {
		return min(4, runtime.NumCPU()), nil
	}
	return strconv.Atoi(spec)
}

type carrierRun struct {
	clean, stego, minimal [][]float64
	transcode, stegoSDR   float64
	added                 float64
	kbps                  float64
	perceptual            float64
}

func measureFormat(ctx context.Context, spec string, carriers []harnessCarrier, pub []byte, tool *quality.Tool, jobs int) formatReport {
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
	runs := make([]carrierRun, len(carriers))
	errs := make([]error, len(carriers))
	sem := make(chan struct{}, jobs)
	var wg sync.WaitGroup
	for i, c := range carriers {
		wg.Go(func() {
			sem <- struct{}{}
			defer func() { <-sem }()
			runs[i], errs[i] = measureCarrier(ctx, em, f.Ext, c, tool)
		})
	}
	wg.Wait()

	var clean, stegoFeatures, minimal [][][]float64
	var transcode, stegoSDR, gap, added, drop []float64
	for i, run := range runs {
		if errs[i] != nil {
			fr.Skipped = append(fr.Skipped, skippedCarrier{Name: carriers[i].name, Reason: errs[i].Error()})
			continue
		}
		fr.Measured++
		clean = append(clean, run.clean)
		stegoFeatures = append(stegoFeatures, run.stego)
		minimal = append(minimal, run.minimal)
		transcode = append(transcode, run.transcode)
		stegoSDR = append(stegoSDR, run.stegoSDR)
		gap = append(gap, run.transcode-run.stegoSDR)
		added = append(added, run.added)
		drop = append(drop, run.perceptual)
		fr.Carriers = append(fr.Carriers, carrierResult{
			Name: carriers[i].name, Kbps: run.kbps, Transcode: num(run.transcode),
			Stego: num(run.stegoSDR), Gap: num(run.transcode - run.stegoSDR), Added: num(run.added),
		})
	}
	if fr.Measured == 0 {
		return fr
	}
	for k, d := range steganalysis.Detectors() {
		pos := column(stegoFeatures, k)
		fr.Detectors = append(fr.Detectors,
			detectorFrom(d.Name, pos, column(clean, k), steganalysis.AUC(pos, column(minimal, k))))
	}
	pos, neg := classify(stegoFeatures, clean)
	fr.Detectors = append(fr.Detectors,
		detectorFrom(classifierName, pos, neg, steganalysis.AUC(classify(stegoFeatures, minimal))))
	fr.Transcode = summarize(transcode, minOf)
	fr.Stego = summarize(stegoSDR, minOf)
	fr.Gap = summarize(gap, maxOf)
	fr.Added = summarize(added, minOf)
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
		clean:     featuresOf(cleanVals),
		stego:     featuresOf(stegoVals),
		minimal:   featuresOf(minimalVals),
		transcode: quality.SDR(ref.Planes, cleanPCM, maxLag),
		stegoSDR:  quality.SDR(ref.Planes, stegoPCM, maxLag),
		added:     quality.SDR(cleanPCM, stegoPCM, maxLag),
		kbps:      float64(len(stegoOut)) * 8 / 1000 / (float64(ref.NbSamples) / float64(info.SampleRate)),
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

func featuresOf(v []int32) [][]float64 {
	var out [][]float64
	for off := 0; off < len(v); off += harnessChunk {
		out = append(out, steganalysis.Features(v[off:min(off+harnessChunk, len(v))]))
	}
	return out
}

func column(carriers [][][]float64, k int) []float64 {
	var out []float64
	for _, chunks := range carriers {
		for _, f := range chunks {
			out = append(out, f[k])
		}
	}
	return out
}

func classify(pos, neg [][][]float64) ([]float64, []float64) {
	var x [][]float64
	var y []bool
	var groups []int
	for i := range pos {
		for _, f := range pos[i] {
			x, y, groups = append(x, f), append(y, true), append(groups, i)
		}
		for _, f := range neg[i] {
			x, y, groups = append(x, f), append(y, false), append(groups, i)
		}
	}
	var ps, ns []float64
	for i, v := range steganalysis.CrossValidate(x, y, groups, harnessFolds) {
		if y[i] {
			ps = append(ps, v)
		} else {
			ns = append(ns, v)
		}
	}
	return ps, ns
}

func detectorFrom(name string, pos, neg []float64, invariance float64) detectorResult {
	lo, hi := steganalysis.AUCInterval(pos, neg, harnessRounds, harnessSeed)
	return detectorResult{Name: name, AUC: steganalysis.AUC(pos, neg), Lo: lo, Hi: hi, Invariance: invariance}
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

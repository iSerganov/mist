//go:build harness

package mist

import (
	"bytes"
	"cmp"
	"context"
	"fmt"
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
	if _, err := exec.LookPath("ffmpeg"); err != nil {
		s.T().Skip("ffmpeg CLI not on PATH: the harness compares Mist against its output")
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
	clean, own, stego, minimal [][]float64
	transcode, stegoSDR        float64
	added                      float64
	kbps                       float64
	perceptual                 float64
	trace                      carrierTrace
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
			runs[i], errs[i] = measureCarrier(ctx, em, codecName, f.Ext, c, tool)
		})
	}
	wg.Wait()

	var clean, own, stegoFeatures, minimal [][][]float64
	var transcode, stegoSDR, gap, added, drop []float64
	for i, run := range runs {
		if errs[i] != nil {
			fr.Skipped = append(fr.Skipped, skippedCarrier{Name: carriers[i].name, Reason: errs[i].Error()})
			continue
		}
		fr.Measured++
		clean = append(clean, run.clean)
		own = append(own, run.own)
		stegoFeatures = append(stegoFeatures, run.stego)
		minimal = append(minimal, run.minimal)
		transcode = append(transcode, run.transcode)
		stegoSDR = append(stegoSDR, run.stegoSDR)
		gap = append(gap, run.transcode-run.stegoSDR)
		added = append(added, run.added)
		drop = append(drop, run.perceptual)
		fr.Traces = append(fr.Traces, run.trace)
		fr.Carriers = append(fr.Carriers, carrierResult{
			Name: carriers[i].name, Kbps: run.kbps, Transcode: num(run.transcode),
			Stego: num(run.stegoSDR), Gap: num(run.transcode - run.stegoSDR), Added: num(run.added),
		})
	}
	if fr.Measured == 0 {
		return fr
	}
	fr.Detectors = detectors(stegoFeatures, clean, minimal)
	fr.Embedding = detectors(stegoFeatures, own, minimal)
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

func measureCarrier(ctx context.Context, em *Emitter, codecName, ext string, c harnessCarrier, tool *quality.Tool) (carrierRun, error) {
	data, err := c.load()
	if err != nil {
		return carrierRun{}, err
	}
	ref, info, err := decodeCarrier(bytes.NewReader(data))
	if err != nil {
		return carrierRun{}, err
	}
	twin, err := ffmpegTwin(ctx, em.target.Container, codecName, c.name, data)
	if err != nil {
		return carrierRun{}, err
	}
	own, err := mistTwin(em.target, data)
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
	var outs [4]decoded
	for i, b := range [][]byte{twin, own, stegoOut, minimalOut} {
		if outs[i], err = inspect(b); err != nil {
			return carrierRun{}, err
		}
	}
	cleanOut, ownOut, stegoDec, minimalDec := outs[0], outs[1], outs[2], outs[3]
	maxLag := info.SampleRate / 10
	run := carrierRun{
		clean:     featuresOf(cleanOut.vals),
		own:       featuresOf(ownOut.vals),
		stego:     featuresOf(stegoDec.vals),
		minimal:   featuresOf(minimalDec.vals),
		transcode: quality.SDR(ref.Planes, ownOut.planes, maxLag),
		stegoSDR:  quality.SDR(ref.Planes, stegoDec.planes, maxLag),
		added:     quality.SDR(ownOut.planes, stegoDec.planes, maxLag),
		kbps:      stegoDec.trace.Kbps,
		trace:     carrierTrace{Name: c.name, Source: ref.NbSamples, FFmpeg: cleanOut.trace, Mist: stegoDec.trace},
	}
	if tool != nil {
		run.perceptual, err = perceptualDrop(ctx, *tool, c.name, ext, data, own, stegoOut)
	}
	return run, err
}

// ffmpegTwin encodes the carrier with the ffmpeg CLI at its own defaults,
// which is the clean file a warden without the original compares Mist's
// output against. Only the first audio stream is kept, so cover art in
// an MP3 does not turn into a video stream in the twin.
func ffmpegTwin(ctx context.Context, container, codecName, name string, data []byte) ([]byte, error) {
	dir, err := os.MkdirTemp("", "mist-twin-*")
	if err != nil {
		return nil, err
	}
	defer func() { _ = os.RemoveAll(dir) }()
	in, out := filepath.Join(dir, "carrier"+filepath.Ext(name)), filepath.Join(dir, "twin")
	if err := os.WriteFile(in, data, 0o600); err != nil {
		return nil, err
	}
	args := []string{"-nostdin", "-loglevel", "error", "-i", in, "-map", "0:a:0"}
	if codecName != "" {
		args = append(args, "-c:a", codecName)
	}
	args = append(args, "-f", container, out)
	if msg, err := exec.CommandContext(ctx, "ffmpeg", args...).CombinedOutput(); err != nil {
		return nil, fmt.Errorf("ffmpeg: %v: %s", err, bytes.TrimSpace(msg))
	}
	return os.ReadFile(out)
}

// mistTwin is the carrier through Mist's own encoder with nothing
// embedded, so comparing it with the stego copy isolates the embedding
// from everything else Mist's pipeline does differently from ffmpeg.
func mistTwin(target av.Format, data []byte) ([]byte, error) {
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

type decoded struct {
	planes [][]float32
	vals   []int32
	trace  outputTrace
}

func inspect(out []byte) (decoded, error) {
	pcm, info, err := decodeCarrier(bytes.NewReader(out))
	if err != nil {
		return decoded{}, err
	}
	d := decoded{planes: pcm.Planes, trace: traceOf(out, pcm, info)}
	if av.Lossless(info.NativeCodecID) {
		d.vals = gridValues(pcm.Planes, av.SampleScale(info.SampleFmt))
		return d, nil
	}
	_, pkts, err := readPackets(bytes.NewReader(out))
	if err != nil {
		return decoded{}, err
	}
	vc := vorbis.New()
	if err := vc.Load(info.Extradata); err != nil {
		return decoded{}, err
	}
	d.vals, err = stego.EligibleValues(vc, pkts)
	return d, err
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

// scored is a population of detector scores and the carrier each came
// from, which the interval needs: chunks of one carrier are not independent.
type scored struct {
	vals   []float64
	groups []int
}

func (s *scored) add(v float64, g int) {
	s.vals, s.groups = append(s.vals, v), append(s.groups, g)
}

// perFile averages each carrier's chunk scores into one score for the file.
func (s scored) perFile() scored {
	var out scored
	at := map[int]int{}
	var n []float64
	for i, v := range s.vals {
		j, ok := at[s.groups[i]]
		if !ok {
			j = len(out.vals)
			at[s.groups[i]] = j
			out.add(0, s.groups[i])
			n = append(n, 0)
		}
		out.vals[j] += v
		n[j]++
	}
	for j := range out.vals {
		out.vals[j] /= n[j]
	}
	return out
}

func column(carriers [][][]float64, k int) scored {
	var out scored
	for i, chunks := range carriers {
		for _, f := range chunks {
			out.add(f[k], i)
		}
	}
	return out
}

func classify(pos, neg [][][]float64) (scored, scored) {
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
	var ps, ns scored
	for i, v := range steganalysis.CrossValidate(x, y, groups, harnessFolds) {
		if y[i] {
			ps.add(v, groups[i])
		} else {
			ns.add(v, groups[i])
		}
	}
	return ps, ns
}

func detectors(pos, neg, minimal [][][]float64) []detectorResult {
	var out []detectorResult
	for k, d := range steganalysis.Detectors() {
		p := column(pos, k)
		out = append(out, detectorFrom(d.Name, p, column(neg, k), steganalysis.AUC(p.vals, column(minimal, k).vals)))
	}
	ps, ns := classify(pos, neg)
	ms, mn := classify(pos, minimal)
	return append(out, detectorFrom(classifierName, ps, ns, steganalysis.AUC(ms.vals, mn.vals)))
}

func detectorFrom(name string, pos, neg scored, invariance float64) detectorResult {
	lo, hi := steganalysis.AUCInterval(pos.vals, neg.vals, pos.groups, neg.groups, harnessRounds, harnessSeed)
	fp, fn := pos.perFile(), neg.perFile()
	flo, fhi := steganalysis.AUCInterval(fp.vals, fn.vals, fp.groups, fn.groups, harnessRounds, harnessSeed)
	return detectorResult{
		Name: name, AUC: steganalysis.AUC(pos.vals, neg.vals), Lo: lo, Hi: hi,
		FileAUC: steganalysis.AUC(fp.vals, fn.vals), FileLo: flo, FileHi: fhi,
		Invariance: invariance,
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

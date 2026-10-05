//go:build harness

package mist

import (
	"bytes"
	"cmp"
	"context"
	"fmt"
	"io"
	"math/rand/v2"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"slices"
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

// TestMerge joins the report.json files of separate runs, one per format,
// into one report. A long corpus does not fit one command's time limit, but
// the formats are independent and can run side by side.
func (s *HarnessSuite) TestMerge() {
	paths := os.Getenv("MIST_HARNESS_MERGE")
	if paths == "" {
		s.T().Skip("MIST_HARNESS_MERGE not set")
	}
	var merged *harnessReport
	for _, path := range strings.Split(paths, ",") {
		r, err := readReport(path)
		s.Require().NoError(err)
		if merged == nil {
			merged = r
			continue
		}
		merged.Formats = append(merged.Formats, r.Formats...)
	}
	for i := range merged.Formats {
		merged.Formats[i].refreshStats()
	}
	baseline, err := readReport(os.Getenv("MIST_HARNESS_BASELINE"))
	s.Require().NoError(err)
	md, err := merged.write(cmp.Or(os.Getenv("MIST_HARNESS_OUT"), "harness-out"), baseline)
	s.Require().NoError(err)
	s.T().Log("\n" + md)
}

func (s *HarnessSuite) TestMeasure() {
	if os.Getenv("MIST_HARNESS_MERGE") != "" {
		s.T().Skip("merging reports, not measuring")
	}
	if !av.Available() {
		s.T().Skip("libav not available")
	}
	if _, err := exec.LookPath("ffmpeg"); err != nil {
		s.T().Skip("ffmpeg CLI not on PATH: the harness compares Mist against its output")
	}
	corpus := os.Getenv("MIST_CORPUS")
	maxSeconds, err := strconv.Atoi(cmp.Or(os.Getenv("MIST_HARNESS_MAX_SECONDS"), "0"))
	s.Require().NoError(err)
	carriers, err := loadCarriers(corpus, maxSeconds)
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
		Corpus:   corpusLabel(corpus, maxSeconds),
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

func corpusLabel(corpus string, maxSeconds int) string {
	label := cmp.Or(corpus, "synthetic")
	if maxSeconds > 0 {
		label += fmt.Sprintf(", first %d s of each carrier", maxSeconds)
	}
	return label
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
	clean, own, stego, minimal []chunk
	transcode, stegoSDR        float64
	added                      float64
	kbps                       float64
	keyStego, keyClean         float64
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
			runs[i], errs[i] = measureCarrier(ctx, em, f.String(), codecName, f.Ext, c, tool)
		})
	}
	wg.Wait()

	var clean, own, stegoFeatures, minimal [][]chunk
	var keyStego, keyClean scored
	var categories []string
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
		categories = append(categories, carriers[i].category)
		keyStego.add(run.keyStego, i)
		keyClean.add(run.keyClean, i)
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
	fr.Detectors = append(detectors(stegoFeatures, clean, minimal), detectorFrom(keyAwareName, keyStego, keyClean, 0.5))
	fr.Scaling = scaling(stegoFeatures, clean, minimal)
	fr.Categories = byCategory(categories, stegoFeatures, clean, minimal)
	fr.Pooled = pooled(stegoFeatures, clean, minimal)
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

func measureCarrier(ctx context.Context, em *Emitter, format, codecName, ext string, c harnessCarrier, tool *quality.Tool) (carrierRun, error) {
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
	sameLevel := twin
	if !em.target.Lossless {
		level, err := mistLevel(em.target, data)
		if err != nil {
			return carrierRun{}, err
		}
		if sameLevel, err = ffmpegTwin(ctx, em.target.Container, codecName, c.name, data, "-q:a", strconv.Itoa(level)); err != nil {
			return carrierRun{}, err
		}
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
	keyStego, err := keyAwareScore(stegoOut, em.pub)
	if err != nil {
		return carrierRun{}, fmt.Errorf("key-aware: %w", err)
	}
	keyClean, err := keyAwareScore(sameLevel, em.pub)
	if err != nil {
		return carrierRun{}, fmt.Errorf("key-aware: %w", err)
	}
	var outs [5]decoded
	for i, b := range [][]byte{twin, own, stegoOut, minimalOut, sameLevel} {
		if outs[i], err = inspect(b); err != nil {
			return carrierRun{}, err
		}
	}
	defaultOut, ownOut, stegoDec, minimalDec, cleanOut := outs[0], outs[1], outs[2], outs[3], outs[4]
	if err := exportCarrier(format, c, cleanOut.vals, stegoDec.vals); err != nil {
		return carrierRun{}, err
	}
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
		keyStego:  keyStego,
		keyClean:  keyClean,
		trace:     carrierTrace{Name: c.name, Source: ref.NbSamples, FFmpeg: defaultOut.trace, Mist: stegoDec.trace},
	}
	if tool != nil {
		run.perceptual, err = perceptualDrop(ctx, *tool, c.name, ext, data, own, stegoOut)
	}
	return run, err
}

// ffmpegTwin encodes the carrier with the ffmpeg CLI at its own defaults
// plus any extra encoder arguments. That is the clean file a warden without
// the original compares Mist's output against. Only the first audio stream
// is kept, so cover art in an MP3 does not turn into a video stream.
func ffmpegTwin(ctx context.Context, container, codecName, name string, data []byte, extra ...string) ([]byte, error) {
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
	args = append(append(args, extra...), "-f", container, out)
	if msg, err := exec.CommandContext(ctx, "ffmpeg", args...).CombinedOutput(); err != nil {
		return nil, fmt.Errorf("ffmpeg: %v: %s", err, bytes.TrimSpace(msg))
	}
	return os.ReadFile(out)
}

// mistLevel is the Vorbis quality level Mist chose for the carrier, found by
// asking each level for its nominal rate and matching the one Mist wrote.
func mistLevel(target av.Format, data []byte) (int, error) {
	pcm, info, err := decodeCarrier(bytes.NewReader(data))
	if err != nil {
		return 0, err
	}
	chosen, err := openEncoder(target, pcm, info)
	if err != nil {
		return 0, err
	}
	defer func() { _ = chosen.Close() }()
	want := target.Info(pcm.SampleRate, pcm.Channels, info)
	want.VBR = true
	for q := minQuality; q <= maxQuality; q++ {
		want.Quality = q
		enc, err := av.NewEncoder(want)
		if err != nil {
			continue
		}
		nominal := nominalRate(enc)
		_ = enc.Close()
		if nominal == nominalRate(chosen) {
			return q, nil
		}
	}
	return 0, fmt.Errorf("no vorbis level has Mist's nominal rate %d", nominalRate(chosen))
}

// mistTwin is the carrier through Mist's own encoder with nothing
// embedded, so comparing it with the stego copy isolates the embedding
// from everything else Mist's pipeline does differently from ffmpeg.
func mistTwin(target av.Format, data []byte) ([]byte, error) {
	pcm, info, err := decodeCarrier(bytes.NewReader(data))
	if err != nil {
		return nil, err
	}
	enc, err := openEncoder(target, pcm, info)
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

// chunk is what the wardens see of one stretch of a file: the general
// classifier's features, led by every detector's score, and the
// second-difference Markov features the markov classifier trains on alone.
type chunk struct {
	features, markov []float64
}

func featuresOf(v []int32) []chunk {
	var out []chunk
	for off := 0; off < len(v); off += harnessChunk {
		c := v[off:min(off+harnessChunk, len(v))]
		out = append(out, chunk{features: steganalysis.Features(c), markov: steganalysis.Markov(c)})
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

func column(carriers [][]chunk, k int) scored {
	var out scored
	for i, chunks := range carriers {
		for _, c := range chunks {
			out.add(c.features[k], i)
		}
	}
	return out
}

func classify(pos, neg [][]chunk, pick func(chunk) []float64) (scored, scored) {
	var x [][]float64
	var y []bool
	var groups []int
	for i := range pos {
		for _, c := range pos[i] {
			x, y, groups = append(x, pick(c)), append(y, true), append(groups, i)
		}
		for _, c := range neg[i] {
			x, y, groups = append(x, pick(c)), append(y, false), append(groups, i)
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

// detectorScores is one detector's score for every chunk of the stego copy
// and of the copy it is compared with.
type detectorScores struct {
	name     string
	pos, neg scored
}

func scoreAll(pos, neg [][]chunk) []detectorScores {
	var out []detectorScores
	for k, d := range steganalysis.Detectors() {
		out = append(out, detectorScores{d.Name, column(pos, k), column(neg, k)})
	}
	for _, cl := range []struct {
		name string
		pick func(chunk) []float64
	}{
		{classifierName, func(c chunk) []float64 { return c.features }},
		{markovName, func(c chunk) []float64 { return c.markov }},
	} {
		ps, ns := classify(pos, neg, cl.pick)
		out = append(out, detectorScores{cl.name, ps, ns})
	}
	return out
}

func detectors(pos, neg, minimal [][]chunk) []detectorResult {
	own := scoreAll(pos, minimal)
	var out []detectorResult
	for i, sc := range scoreAll(pos, neg) {
		out = append(out, detectorFrom(sc.name, sc.pos, sc.neg, steganalysis.AUC(own[i].pos.vals, own[i].neg.vals)))
	}
	return out
}

// pooledFiles are the numbers of files a warden pools in the report.
var pooledFiles = []int{1, 3, 6}

// pooledDraws is how many random pools of each size are scored.
const pooledDraws = 500

// pooled scores what a warden who collects several files sees: the mean file
// score of k stego files against the mean of k clean ones, over random draws
// of which files. Draws overlap, so the AUC has no interval; read it as a
// trend across k.
func pooled(pos, neg, minimal [][]chunk) []pooledRow {
	if len(pos) < 2 {
		return nil
	}
	var out []pooledRow
	for _, k := range pooledFiles {
		if k > len(pos) {
			continue
		}
		row := pooledRow{Files: k}
		for _, sc := range scoreAll(pos, neg) {
			p, n := sc.pos.perFile().vals, sc.neg.perFile().vals
			row.Detectors = append(row.Detectors, pooledResult{Name: sc.name, AUC: pooledAUC(p, n, k)})
		}
		out = append(out, row)
	}
	return out
}

func pooledAUC(pos, neg []float64, k int) float64 {
	rng := rand.New(rand.NewPCG(harnessSeed, uint64(k)))
	mean := func(v []float64) float64 {
		var sum float64
		for _, i := range rng.Perm(len(v))[:k] {
			sum += v[i]
		}
		return sum / float64(k)
	}
	p, n := make([]float64, pooledDraws), make([]float64, pooledDraws)
	for i := range pooledDraws {
		p[i], n[i] = mean(pos), mean(neg)
	}
	return steganalysis.AUC(p, n)
}

// byCategory scores every detector on each corpus category alone, so a kind
// of audio that behaves unlike the rest does not average away. It returns
// nothing when the corpus has one category.
func byCategory(names []string, pos, neg, minimal [][]chunk) []categoryRow {
	seen := map[string]bool{}
	var order []string
	for _, n := range names {
		if !seen[n] {
			seen[n] = true
			order = append(order, n)
		}
	}
	if len(order) < 2 {
		return nil
	}
	slices.Sort(order)
	var out []categoryRow
	for _, cat := range order {
		var p, n, m [][]chunk
		for i, name := range names {
			if name == cat {
				p, n, m = append(p, pos[i]), append(n, neg[i]), append(m, minimal[i])
			}
		}
		out = append(out, categoryRow{Name: cat, Carriers: len(p), Detectors: detectors(p, n, m)})
	}
	return out
}

// scalingChunks are the prefix lengths, in chunks, that the report shows
// next to the whole file: about 30 s and 3 min of 44.1 kHz stereo.
var scalingChunks = []int{40, 240}

// scaling scores each detector on the first n chunks of every carrier, so a
// detector that gains on longer audio (the square-root law) shows as rising
// file AUC across the rows. A length no carrier exceeds is left out, because
// it would repeat the whole-file row.
func scaling(pos, neg, minimal [][]chunk) []scalingRow {
	longest := 0
	for _, c := range pos {
		longest = max(longest, len(c))
	}
	var out []scalingRow
	for _, n := range scalingChunks {
		if n < longest {
			out = append(out, scalingRow{Chunks: n, Detectors: detectors(prefix(pos, n), prefix(neg, n), prefix(minimal, n))})
		}
	}
	return out
}

func prefix(carriers [][]chunk, n int) [][]chunk {
	out := make([][]chunk, len(carriers))
	for i, c := range carriers {
		out[i] = c[:min(n, len(c))]
	}
	return out
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

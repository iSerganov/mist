//go:build harness

package mist

import (
	"bytes"
	"cmp"
	"context"
	"encoding/hex"
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
	harnessChunk       = 1 << 16
	harnessRounds      = 1000
	harnessSeed        = 1
	harnessFolds       = 5
	harnessPerms       = 199
	harnessRefits      = 9
	harnessPowerTarget = 0.55
	harnessPower       = 0.9
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
		run := len(merged.Manifests)
		merged.Manifests = append(merged.Manifests, r.Manifests...)
		for i := range r.Formats {
			r.Formats[i].Run += run
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
	if _, err := exec.LookPath(harnessBinary("MIST_FFMPEG", "ffmpeg")); err != nil {
		s.T().Skip("configured ffmpeg CLI is unavailable: the harness compares Mist against its output")
	}
	corpus := os.Getenv("MIST_CORPUS")
	maxSeconds, err := strconv.Atoi(cmp.Or(os.Getenv("MIST_HARNESS_MAX_SECONDS"), "0"))
	s.Require().NoError(err)
	carriers, corpusInfo, err := loadCarriers(corpus, os.Getenv("MIST_HARNESS_CORPUS_MANIFEST"), maxSeconds)
	s.Require().NoError(err)
	s.Require().NotEmpty(carriers)
	if name := os.Getenv("MIST_HARNESS_CORPUS_NAME"); name != "" {
		corpusInfo.Name = name
	}
	baseline, err := readReport(os.Getenv("MIST_HARNESS_BASELINE"))
	s.Require().NoError(err)
	pub, err := harnessPublicKey(os.Getenv("MIST_HARNESS_PUBLIC_KEY_HEX"))
	s.Require().NoError(err)

	jobs, err := harnessJobs(os.Getenv("MIST_HARNESS_JOBS"))
	s.Require().NoError(err)
	tool := perceptualTool()
	formats := harnessFormats(os.Getenv("MIST_HARNESS_FORMATS"))
	perceptual := ""
	if tool != nil {
		perceptual = tool.Name
	}
	manifest, err := makeRunManifest(carriers, corpusInfo, maxSeconds, pub, formats, jobs, perceptual)
	s.Require().NoError(err)
	rep := harnessReport{
		Commit:    manifest.Git.Describe,
		Date:      time.Now().UTC().Format(time.DateOnly),
		Corpus:    corpusLabel(corpusInfo.Name, maxSeconds),
		Carriers:  len(carriers),
		Manifests: []runManifest{manifest},
	}
	if tool != nil {
		rep.Perceptual = tool.Name
	}
	for _, spec := range formats {
		s.T().Logf("measuring %s", spec)
		rep.Formats = append(rep.Formats, measureFormat(s.T().Context(), spec, carriers, pub, tool, jobs))
	}
	md, err := rep.write(cmp.Or(os.Getenv("MIST_HARNESS_OUT"), "harness-out"), baseline)
	s.Require().NoError(err)
	s.T().Log("\n" + md)
}

func corpusLabel(name string, maxSeconds int) string {
	label := cmp.Or(name, "external")
	if maxSeconds > 0 {
		label += fmt.Sprintf(", first %d s of each carrier", maxSeconds)
	}
	return label
}

func harnessPublicKey(spec string) ([]byte, error) {
	if spec == "" {
		pub, _, err := GenerateKeyPair()
		return pub, err
	}
	pub, err := hex.DecodeString(spec)
	if err != nil {
		return nil, fmt.Errorf("MIST_HARNESS_PUBLIC_KEY_HEX: %w", err)
	}
	if len(pub) != 32 {
		return nil, fmt.Errorf("MIST_HARNESS_PUBLIC_KEY_HEX: got %d bytes, want 32", len(pub))
	}
	return pub, nil
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
	selStego, selClean         float64
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
	var keyStego, keyClean, selStego, selClean scored
	var categories []string
	var identities []scoreIdentity
	var familyOf []int
	lineageID := map[string]int{}
	var transcode, stegoSDR, gap, added, drop []float64
	for i, run := range runs {
		if errs[i] != nil {
			fr.Skipped = append(fr.Skipped, skippedCarrier{Name: carriers[i].name, Reason: redactLocalPaths(errs[i].Error())})
			continue
		}
		fr.Measured++
		clean = append(clean, run.clean)
		own = append(own, run.own)
		stegoFeatures = append(stegoFeatures, run.stego)
		minimal = append(minimal, run.minimal)
		categories = append(categories, carriers[i].category)
		group := len(identities)
		identities = append(identities, scoreIdentity{
			Carrier: carriers[i].name, Category: carriers[i].category, Lineage: carriers[i].lineage,
		})
		lineage := carriers[i].lineage
		if lineage == "" {
			lineage = carriers[i].name
		}
		fam, ok := lineageID[lineage]
		if !ok {
			fam = len(lineageID)
			lineageID[lineage] = fam
		}
		familyOf = append(familyOf, fam)
		keyStego.add(run.keyStego, group, fam)
		keyClean.add(run.keyClean, group, fam)
		selStego.add(run.selStego, group, fam)
		selClean.add(run.selClean, group, fam)
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
	fr.Meta = scoreMetadata(fr.Traces, familyOf)
	fr.Detectors, fr.Raw.Operational = analyzeDetectors(stegoFeatures, clean, minimal, identities, familyOf, true)
	fr.Detectors = append(fr.Detectors, detectorFrom(keyAwareName, keyStego, keyClean, 0.5, true))
	fr.Raw.Operational = append(fr.Raw.Operational, rawDetectorFrom(keyAwareName, keyStego, keyClean, scored{}, identities))
	fr.Detectors = append(fr.Detectors, detectorFrom(steganalysis.SelectionName, selStego, selClean, 0.5, true))
	fr.Raw.Operational = append(fr.Raw.Operational, rawDetectorFrom(steganalysis.SelectionName, selStego, selClean, scored{}, identities))
	adjustConfirmatory(fr.Detectors)
	fr.PowerFamilies, fr.PowerReached = classifierPower(fr.Raw.Operational)
	fr.Scaling = scaling(stegoFeatures, clean, minimal, familyOf)
	fr.Categories = byCategory(categories, stegoFeatures, clean, minimal, familyOf)
	fr.Pooled = pooled(stegoFeatures, clean, minimal)
	fr.Embedding, fr.Raw.Embedding = analyzeDetectors(stegoFeatures, own, minimal, identities, familyOf, false)
	fr.LeaveLineage = leaveLineageFrom(fr.Raw.Operational)
	fr.WorstCategory, fr.WorstDetector, fr.WorstFileAUC = worstCell(fr.Categories, fr.Detectors)
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
	ref, info, _, err := decodeCarrier(bytes.NewReader(data))
	if err != nil {
		return carrierRun{}, err
	}
	twin, err := ffmpegTwin(ctx, em.target.Container, codecName, c.ext, data)
	if err != nil {
		return carrierRun{}, err
	}
	sameLevel := twin
	if !em.target.Lossless {
		level, err := mistLevel(em.target, data)
		if err != nil {
			return carrierRun{}, err
		}
		if sameLevel, err = ffmpegTwin(ctx, em.target.Container, codecName, c.ext, data, "-q:a", strconv.Itoa(level)); err != nil {
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
	channels := 1
	if av.Lossless(info.NativeCodecID) {
		channels = info.Channels
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
	selStego, err := selectionScore(stegoDec.vals, em.pub)
	if err != nil {
		return carrierRun{}, fmt.Errorf("selection: %w", err)
	}
	selClean, err := selectionScore(cleanOut.vals, em.pub)
	if err != nil {
		return carrierRun{}, fmt.Errorf("selection: %w", err)
	}
	run := carrierRun{
		clean:     featuresOf(cleanOut.vals, channels),
		own:       featuresOf(ownOut.vals, channels),
		stego:     featuresOf(stegoDec.vals, channels),
		minimal:   featuresOf(minimalDec.vals, channels),
		transcode: quality.SDR(ref.Planes, ownOut.planes, maxLag),
		stegoSDR:  quality.SDR(ref.Planes, stegoDec.planes, maxLag),
		added:     quality.SDR(ownOut.planes, stegoDec.planes, maxLag),
		kbps:      stegoDec.trace.Kbps,
		keyStego:  keyStego,
		keyClean:  keyClean,
		selStego:  selStego,
		selClean:  selClean,
		trace: carrierTrace{
			Name: c.name, Source: ref.NbSamples,
			SourceRate: info.SampleRate, SourceChannels: info.Channels, SourceFmt: sampleFmtName(info.SampleFmt),
			FFmpeg: defaultOut.trace, Canonical: cleanOut.trace, Clean: ownOut.trace, Mist: stegoDec.trace,
		},
	}
	if tool != nil {
		run.perceptual, err = perceptualDrop(ctx, *tool, c.ext, ext, data, own, stegoOut)
	}
	return run, err
}

// ffmpegTwin encodes the carrier with the ffmpeg CLI at its own defaults
// plus any extra encoder arguments. That is the clean file a warden without
// the original compares Mist's output against. Only the first audio stream
// is kept, so cover art in an MP3 does not turn into a video stream.
func ffmpegTwin(ctx context.Context, container, codecName, carrierExt string, data []byte, extra ...string) ([]byte, error) {
	dir, err := os.MkdirTemp("", "mist-twin-*")
	if err != nil {
		return nil, err
	}
	defer func() { _ = os.RemoveAll(dir) }()
	in, out := filepath.Join(dir, "carrier"+carrierExt), filepath.Join(dir, "twin")
	if err := os.WriteFile(in, data, 0o600); err != nil {
		return nil, err
	}
	args := []string{"-nostdin", "-loglevel", "error", "-i", in, "-map", "0:a:0"}
	if codecName != "" {
		args = append(args, "-c:a", codecName)
	}
	args = append(append(args, extra...), "-f", container, out)
	if msg, err := exec.CommandContext(ctx, harnessBinary("MIST_FFMPEG", "ffmpeg"), args...).CombinedOutput(); err != nil {
		return nil, fmt.Errorf("ffmpeg: %s", redactLocalPaths(fmt.Sprintf("%v: %s", err, bytes.TrimSpace(msg)), dir, in, out))
	}
	return os.ReadFile(out)
}

// mistLevel is the Vorbis quality level Mist opened its encoder at for the
// carrier. It asks the encoder: libvorbis writes a nominal rate of 0 at some
// sample rates, so matching rates cannot tell the levels apart.
func mistLevel(target av.Format, data []byte) (int, error) {
	pcm, info, _, err := decodeCarrier(bytes.NewReader(data))
	if err != nil {
		return 0, err
	}
	enc, err := openEncoder(target, pcm, info)
	if err != nil {
		return 0, err
	}
	defer func() { _ = enc.Close() }()
	return enc.Info().Quality, nil
}

// mistTwin is the carrier through Mist's own encoder with nothing
// embedded, including the lossless grid snap Embed does before it
// changes a sample. Comparing it with the stego copy isolates the
// embedding from everything else Mist's pipeline does differently
// from ffmpeg.
func mistTwin(target av.Format, data []byte) ([]byte, error) {
	pcm, info, meta, err := decodeCarrier(bytes.NewReader(data))
	if err != nil {
		return nil, err
	}
	enc, err := openEncoder(target, pcm, info)
	if err != nil {
		return nil, err
	}
	defer func() { _ = enc.Close() }()
	// Embed snaps a lossless carrier onto the encoder grid before it changes
	// anything. The clean twin has to do the same, or the comparison is a
	// second resample rather than the pipeline with nothing embedded.
	if target.Lossless {
		if err := enc.Snap(pcm.Planes, pcm.Frames); err != nil {
			return nil, err
		}
	}
	rc, err := encodeAndMux(enc, pcm, meta, nil)
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
	pcm, info, _, err := decodeCarrier(bytes.NewReader(out))
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
	features, markov, rich []float64
}

func featuresOf(v []int32, channels int) []chunk {
	var out []chunk
	for off := 0; off < len(v); off += harnessChunk {
		c := v[off:min(off+harnessChunk, len(v))]
		out = append(out, chunk{
			features: steganalysis.Features(c),
			markov:   steganalysis.Markov(c),
			rich:     steganalysis.RichPlanar(c, channels),
		})
	}
	return out
}

// scored is a population of detector scores. groups is the recording, which
// perFile averages over. family is the lineage, which the interval and the
// classifier treat as the independent sample. Chunks of one recording, and
// recordings of one lineage, are not independent of each other.
type scored struct {
	vals   []float64
	groups []int
	family []int
}

func (s *scored) add(v float64, recording, family int) {
	s.vals = append(s.vals, v)
	s.groups = append(s.groups, recording)
	s.family = append(s.family, family)
}

func (s scored) familyAt(i int) int {
	if i < len(s.family) {
		return s.family[i]
	}
	if i < len(s.groups) {
		return s.groups[i]
	}
	return 0
}

// perFile averages each recording's chunk scores into one score for the file.
// Chapters stay separate files; they share a lineage through family.
func (s scored) perFile() scored {
	var out scored
	at := map[int]int{}
	var n []float64
	for i, v := range s.vals {
		j, ok := at[s.groups[i]]
		if !ok {
			j = len(out.vals)
			at[s.groups[i]] = j
			out.add(0, s.groups[i], s.familyAt(i))
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

func column(carriers [][]chunk, k int, familyOf []int) scored {
	var out scored
	for i, chunks := range carriers {
		fam := i
		if i < len(familyOf) {
			fam = familyOf[i]
		}
		for _, c := range chunks {
			out.add(c.features[k], i, fam)
		}
	}
	return out
}

// trained is the matrix a confirmatory refit has to see again: the same
// rows, labels and lineage groups the reported scores came from.
type trained struct {
	x      [][]float64
	y      []bool
	groups []int
}

func classify(pos, neg [][]chunk, pick func(chunk) []float64, familyOf []int) (scored, scored, trained) {
	var x [][]float64
	var y []bool
	var groups, recording []int
	for i := range pos {
		fam := i
		if i < len(familyOf) {
			fam = familyOf[i]
		}
		for _, c := range pos[i] {
			x = append(x, pick(c))
			y = append(y, true)
			groups = append(groups, fam)
			recording = append(recording, i)
		}
		for _, c := range neg[i] {
			x = append(x, pick(c))
			y = append(y, false)
			groups = append(groups, fam)
			recording = append(recording, i)
		}
	}
	var ps, ns scored
	for i, v := range steganalysis.NestedCrossValidate(x, y, groups, harnessFolds) {
		if y[i] {
			ps.add(v, recording[i], groups[i])
		} else {
			ns.add(v, recording[i], groups[i])
		}
	}
	return ps, ns, trained{x: x, y: y, groups: groups}
}

// detectorScores is one detector's score for every chunk of the stego copy
// and of the copy it is compared with.
type detectorScores struct {
	name     string
	pos, neg scored
	model    trained
}

type scoreIdentity struct {
	Carrier  string
	Category string
	Lineage  string
}

func scoreAll(pos, neg [][]chunk, familyOf []int) []detectorScores {
	var out []detectorScores
	for k, d := range steganalysis.Detectors() {
		out = append(out, detectorScores{name: d.Name, pos: column(pos, k, familyOf), neg: column(neg, k, familyOf)})
	}
	for _, cl := range []struct {
		name string
		pick func(chunk) []float64
	}{
		{classifierName, func(c chunk) []float64 { return c.features }},
		{markovName, func(c chunk) []float64 { return c.markov }},
		{steganalysis.RichName, func(c chunk) []float64 { return c.rich }},
	} {
		ps, ns, model := classify(pos, neg, cl.pick, familyOf)
		out = append(out, detectorScores{name: cl.name, pos: ps, neg: ns, model: model})
	}
	return out
}

func detectors(pos, neg, minimal [][]chunk, familyOf []int) []detectorResult {
	out, _ := analyzeDetectors(pos, neg, minimal, nil, familyOf, false)
	return out
}

func analyzeDetectors(pos, neg, minimal [][]chunk, identities []scoreIdentity, familyOf []int, confirm bool) ([]detectorResult, []rawDetectorScores) {
	own := scoreAll(pos, minimal, familyOf)
	var out []detectorResult
	var raw []rawDetectorScores
	for i, sc := range scoreAll(pos, neg, familyOf) {
		d := detectorFrom(sc.name, sc.pos, sc.neg, steganalysis.AUC(own[i].pos.vals, own[i].neg.vals), confirm)
		if confirm && trainedDetector(sc.name) && len(sc.model.x) > 0 {
			d.PermP = steganalysis.RefitPermutationP(sc.model.x, sc.model.y, sc.model.groups, harnessFolds, harnessRefits, harnessSeed)
		}
		out = append(out, d)
		if identities != nil {
			raw = append(raw, rawDetectorFrom(sc.name, sc.pos, sc.neg, own[i].neg, identities))
		}
	}
	return out, raw
}

func rawDetectorFrom(name string, pos, neg, minimal scored, identities []scoreIdentity) rawDetectorScores {
	out := rawDetectorScores{
		Name:  name,
		Stego: rawPopulationFrom(pos, identities),
		Clean: rawPopulationFrom(neg, identities),
	}
	if len(minimal.vals) > 0 {
		population := rawPopulationFrom(minimal, identities)
		out.Minimal = &population
	}
	return out
}

func rawPopulationFrom(scores scored, identities []scoreIdentity) rawPopulation {
	var out rawPopulation
	chunks := map[int]int{}
	for i, score := range scores.vals {
		group := scores.groups[i]
		if group < 0 || group >= len(identities) {
			continue
		}
		identity := identities[group]
		out.Chunks = append(out.Chunks, rawScore{
			Carrier: identity.Carrier, Category: identity.Category, Lineage: identity.Lineage,
			Chunk: chunks[group], Score: score,
		})
		chunks[group]++
	}
	files := scores.perFile()
	for i, score := range files.vals {
		group := files.groups[i]
		if group < 0 || group >= len(identities) {
			continue
		}
		identity := identities[group]
		out.Files = append(out.Files, rawScore{
			Carrier: identity.Carrier, Category: identity.Category, Lineage: identity.Lineage, Score: score,
		})
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
		for _, sc := range scoreAll(pos, neg, nil) {
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
func byCategory(names []string, pos, neg, minimal [][]chunk, familyOf []int) []categoryRow {
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
		var pf []int
		for i, name := range names {
			if name == cat {
				p, n, m = append(p, pos[i]), append(n, neg[i]), append(m, minimal[i])
				fam := i
				if i < len(familyOf) {
					fam = familyOf[i]
				}
				pf = append(pf, fam)
			}
		}
		out = append(out, categoryRow{Name: cat, Carriers: len(p), Detectors: detectors(p, n, m, pf)})
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
func scaling(pos, neg, minimal [][]chunk, familyOf []int) []scalingRow {
	longest := 0
	for _, c := range pos {
		longest = max(longest, len(c))
	}
	var out []scalingRow
	for _, n := range scalingChunks {
		if n < longest {
			out = append(out, scalingRow{Chunks: n, Detectors: detectors(prefix(pos, n), prefix(neg, n), prefix(minimal, n), familyOf)})
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

func detectorFrom(name string, pos, neg scored, invariance float64, confirm bool) detectorResult {
	lo, hi := steganalysis.AUCInterval(pos.vals, neg.vals, familyIDs(pos), familyIDs(neg), harnessRounds, harnessSeed)
	fp, fn := pos.perFile(), neg.perFile()
	fileAUC := steganalysis.AUC(fp.vals, fn.vals)
	flo, fhi := steganalysis.HierarchicalInterval(
		fp.vals, fn.vals, familyIDs(fp), fp.groups, familyIDs(fn), fn.groups, harnessRounds, harnessSeed,
	)
	rlo, rhi := steganalysis.AUCInterval(fp.vals, fn.vals, fp.groups, fn.groups, harnessRounds, harnessSeed)
	perm := -1.0
	if confirm && !trainedDetector(name) {
		perm = steganalysis.PairedPermutationP(fp.vals, fn.vals, familyIDs(fp), harnessPerms, harnessSeed)
	}
	return detectorResult{
		Name: name, AUC: steganalysis.AUC(pos.vals, neg.vals), Lo: lo, Hi: hi,
		FileAUC: fileAUC, FileLo: flo, FileHi: fhi,
		Detectability: steganalysis.Detectability(fileAUC),
		RecordLo:      rlo, RecordHi: rhi,
		PermP: perm, HolmP: -1, FDRP: -1,
		Invariance: invariance,
	}
}

func familyIDs(s scored) []int {
	out := make([]int, len(s.vals))
	for i := range s.vals {
		out[i] = s.familyAt(i)
	}
	return out
}

func confirmatoryDetector(name string) bool {
	return slices.Contains(steganalysis.FrozenPrimaries(), name)
}

func trainedDetector(name string) bool {
	return name == classifierName || name == markovName || name == steganalysis.RichName
}

func exploratoryDetector(name string) bool {
	switch name {
	case "chi-square", "spa", "rs":
		return true
	default:
		return false
	}
}

// adjustConfirmatory applies Holm to the preregistered family and
// Benjamini-Hochberg to the exploratory classical detectors. A detector
// whose permutation was not run stays at -1.
func adjustConfirmatory(ds []detectorResult) {
	apply := func(keep func(string) bool, adjust func([]float64) []float64, set func(*detectorResult, float64)) {
		var idx []int
		var p []float64
		for i, d := range ds {
			if keep(d.Name) && d.PermP >= 0 {
				idx = append(idx, i)
				p = append(p, d.PermP)
			}
		}
		if len(p) == 0 {
			return
		}
		adj := adjust(p)
		for j, i := range idx {
			set(&ds[i], adj[j])
		}
	}
	apply(confirmatoryDetector, steganalysis.Holm, func(d *detectorResult, v float64) { d.HolmP = v })
	apply(exploratoryDetector, steganalysis.BH, func(d *detectorResult, v float64) { d.FDRP = v })
}

func classifierPower(raw []rawDetectorScores) (families int, reached bool) {
	for _, d := range raw {
		if d.Name != classifierName {
			continue
		}
		pos, neg, _ := pairByCarrier(d.Stego.Files, d.Clean.Files)
		return steganalysis.FamiliesForPower(pos, neg, harnessPowerTarget, harnessPower, harnessSeed)
	}
	return 0, false
}

func pairByCarrier(stego, clean []rawScore) (pos, neg []float64, lineage []string) {
	at := map[string]rawScore{}
	for _, s := range clean {
		at[s.Carrier] = s
	}
	for _, s := range stego {
		c, ok := at[s.Carrier]
		if !ok {
			continue
		}
		pos = append(pos, s.Score)
		neg = append(neg, c.Score)
		lineage = append(lineage, s.Lineage)
	}
	return pos, neg, lineage
}

func leaveLineageFrom(raw []rawDetectorScores) []leaveLineageRow {
	var stego, clean []rawScore
	for _, d := range raw {
		if d.Name == classifierName {
			stego, clean = d.Stego.Files, d.Clean.Files
			break
		}
	}
	pos, neg, lin := pairByCarrier(stego, clean)
	var order []string
	groups := map[string][]int{}
	for i, name := range lin {
		if _, ok := groups[name]; !ok {
			order = append(order, name)
		}
		groups[name] = append(groups[name], i)
	}
	var out []leaveLineageRow
	for _, name := range order {
		idx := groups[name]
		if len(idx) < 2 {
			continue
		}
		p, n := pickPairs(pos, neg, idx)
		rest := complement(len(pos), idx)
		row := leaveLineageRow{
			Lineage: name, Carriers: len(idx),
			FileAUC:       steganalysis.AUC(p, n),
			Detectability: steganalysis.Detectability(steganalysis.AUC(p, n)),
			RestCarriers:  len(rest),
		}
		if len(rest) >= 2 {
			rp, rn := pickPairs(pos, neg, rest)
			row.RestFileAUC = steganalysis.AUC(rp, rn)
		}
		out = append(out, row)
	}
	return out
}

func pickPairs(pos, neg []float64, idx []int) (p, n []float64) {
	for _, i := range idx {
		p = append(p, pos[i])
		n = append(n, neg[i])
	}
	return p, n
}

func complement(n int, idx []int) []int {
	held := map[int]bool{}
	for _, i := range idx {
		held[i] = true
	}
	var out []int
	for i := range n {
		if !held[i] {
			out = append(out, i)
		}
	}
	return out
}

func worstCell(categories []categoryRow, detectors []detectorResult) (category, detector string, auc float64) {
	best := -1.0
	consider := func(cat string, d detectorResult) {
		dv := steganalysis.Detectability(d.FileAUC)
		if dv > best {
			best = dv
			category, detector, auc = cat, d.Name, d.FileAUC
		}
	}
	if len(categories) == 0 {
		for _, d := range detectors {
			consider("aggregate", d)
		}
		return category, detector, auc
	}
	for _, c := range categories {
		for _, d := range c.Detectors {
			consider(c.Name, d)
		}
	}
	return category, detector, auc
}

func perceptualTool() *quality.Tool {
	for _, t := range []quality.Tool{quality.ViSQOL(), quality.PEAQ()} {
		if t.Available() {
			return &t
		}
	}
	return nil
}

func perceptualDrop(ctx context.Context, tool quality.Tool, carrierExt, ext string, carrier, clean, stegoOut []byte) (float64, error) {
	dir, err := os.MkdirTemp("", "mist-harness-*")
	if err != nil {
		return 0, err
	}
	defer func() { _ = os.RemoveAll(dir) }()
	files := map[string][]byte{
		"carrier" + carrierExt: carrier,
		"clean." + ext:         clean,
		"stego." + ext:         stegoOut,
	}
	for f, b := range files {
		if err := os.WriteFile(filepath.Join(dir, f), b, 0o600); err != nil {
			return 0, err
		}
	}
	ref := filepath.Join(dir, "carrier"+carrierExt)
	cleanScore, err := tool.Score(ctx, ref, filepath.Join(dir, "clean."+ext))
	if err != nil {
		return 0, err
	}
	stegoScore, err := tool.Score(ctx, ref, filepath.Join(dir, "stego."+ext))
	return cleanScore - stegoScore, err
}

func commit() string {
	return inspectGit().Describe
}

func (s *HarnessSuite) TestLineageIntervalPinsASharedFamily() {
	var pos, neg scored
	pos.add(1, 0, 0)
	pos.add(1, 1, 0)
	neg.add(0, 0, 0)
	neg.add(0, 1, 0)
	d := detectorFrom("hcf-com", pos, neg, 0.5, false)
	s.Equal(1.0, d.FileAUC)
	s.Equal(1.0, d.FileLo)
	s.Equal(1.0, d.FileHi)
	s.Equal(1.0, d.Detectability)
	s.Equal(-1.0, d.PermP)
	s.Equal(-1.0, d.HolmP)
}

func (s *HarnessSuite) TestConfirmatoryHolmUsesThePermutation() {
	var pos, neg scored
	for i := range 12 {
		pos.add(1, i, i)
		neg.add(0, i, i)
	}
	d := detectorFrom("hcf-com", pos, neg, 0.5, true)
	s.Less(d.PermP, 0.05)
	ds := []detectorResult{d}
	adjustConfirmatory(ds)
	s.Equal(d.PermP, ds[0].HolmP)
	s.Equal(-1.0, ds[0].FDRP)
}

func (s *HarnessSuite) TestClassifyKeepsRecordingsInsideALineage() {
	mk := func(y float64) chunk { return chunk{features: []float64{y}, markov: []float64{y}} }
	var pos, neg [][]chunk
	var family []int
	for i := range 4 {
		pos = append(pos, []chunk{mk(1), mk(1)})
		neg = append(neg, []chunk{mk(-1), mk(-1)})
		family = append(family, i)
	}
	ps, ns, model := classify(pos, neg, func(c chunk) []float64 { return c.features }, family)
	s.Equal([]int{0, 0, 1, 1, 2, 2, 3, 3}, ps.groups)
	s.Equal([]int{0, 0, 1, 1, 2, 2, 3, 3}, ps.family)
	s.Len(ps.perFile().vals, 4)
	s.Greater(steganalysis.AUC(ps.vals, ns.vals), 0.8)
	s.Equal(family[0], model.groups[0])
}

func (s *HarnessSuite) TestLeaveLineageNeedsTwoCarriers() {
	raw := []rawDetectorScores{{
		Name: classifierName,
		Stego: rawPopulation{Files: []rawScore{
			{Carrier: "a", Lineage: "session", Score: 0.9},
			{Carrier: "b", Lineage: "session", Score: 0.8},
			{Carrier: "c", Lineage: "other", Score: 0.1},
		}},
		Clean: rawPopulation{Files: []rawScore{
			{Carrier: "a", Lineage: "session", Score: 0.2},
			{Carrier: "b", Lineage: "session", Score: 0.1},
			{Carrier: "c", Lineage: "other", Score: 0.2},
		}},
	}}
	rows := leaveLineageFrom(raw)
	s.Require().Len(rows, 1)
	s.Equal("session", rows[0].Lineage)
	s.Equal(1.0, rows[0].FileAUC)
	s.Equal(1, rows[0].RestCarriers)
	s.Equal(0.0, rows[0].RestFileAUC)
}

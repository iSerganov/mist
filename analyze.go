package mist

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"math"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/iSerganov/mist/internal/av"
	"github.com/iSerganov/mist/internal/codec"
	"github.com/iSerganov/mist/internal/codec/vorbis"
	"github.com/iSerganov/mist/internal/forensics"
	"github.com/iSerganov/mist/internal/quality"
	"github.com/iSerganov/mist/internal/steganalysis"
	"github.com/iSerganov/mist/internal/stego"
	"github.com/iSerganov/mist/internal/trace"
)

// chunkValues is how many values a warden scores at a time. The harness
// scores its populations the same way, so a file's score here is the same
// statistic the calibration holds.
const chunkValues = 1 << 16

// The trained wardens, by the names the harness reports them under.
const (
	classifierName = "classifier"
	markovName     = "markov"
)

// analysisPrior is the probability of a message before any evidence: no
// opinion either way.
const analysisPrior = 0.5

// knownClean and knownStego are what a decisive known-cover diff reports.
// Neither is 0 or 1: the diff is one comparison, and the reference the user
// named may not be the file that was really embedded into.
const (
	knownClean = 0.01
	knownStego = 0.99
)

// MaxLR is the most one stage's likelihood ratio can say either way: a
// Stage's LR lies in [1/MaxLR, MaxLR].
const MaxLR = steganalysis.MaxLR

// StageStatus says whether a stage of an Analysis ran.
type StageStatus int

const (
	// StageRan means the stage scored the file.
	StageRan StageStatus = iota
	// StageSkipped means the stage could run here but did not, because a
	// tool or a reference it needs is missing.
	StageSkipped
	// StageNotRun means the stage needs what a single-file analysis does
	// not have: a key, or the original carrier.
	StageNotRun
)

// Basis names what decided an Analysis's probability.
type Basis int

const (
	// BasisBlind is the calibrated detectors alone.
	BasisBlind Basis = iota
	// BasisFormat is a format Mist cannot write.
	BasisFormat
	// BasisKnownCover is a diff against a clean re-encode of the reference.
	BasisKnownCover
)

// Scope says what a stage looks for.
type Scope int

const (
	// ScopeMist stages look for Mist's own embedding and are calibrated.
	ScopeMist Scope = iota
	// ScopeAny stages look for data any tool may have hidden, and report a
	// Suspicion with its reasons rather than a probability.
	ScopeAny
)

// Suspicion is how strongly an uncalibrated check points at hidden data.
type Suspicion int

const (
	// SuspicionNone means nothing out of the ordinary.
	SuspicionNone Suspicion = iota
	// SuspicionLow is unusual but common in honest files.
	SuspicionLow
	// SuspicionMedium is rare in honest files.
	SuspicionMedium
	// SuspicionHigh is data no player reads, stored where encoders write none.
	SuspicionHigh
)

// String names the level for a report.
func (s Suspicion) String() string {
	return [...]string{"none", "low", "medium", "high"}[s]
}

// Stage is one step of an Analysis. For a ScopeMist stage, LR is the
// likelihood ratio against the calibration, above 1 favouring a message,
// and 0 when the stage is not evidence; AUC is how well the stage separated
// clean from stego files in the calibration run. A ScopeAny stage instead
// has a Suspicion and the Findings behind it. An Informational stage
// describes the file for context and is never evidence either way.
type Stage struct {
	Name          string
	Scope         Scope
	Status        StageStatus
	Score         float64
	LR            float64
	AUC           float64
	Note          string
	Suspicion     Suspicion
	Findings      []string
	Informational bool
}

// Analysis is how a warden reads one file. Probability is the chance of a
// Mist message, from the calibrated stages. Suspicion is the strongest
// sign of data hidden by any other tool, from the uncalibrated ones.
type Analysis struct {
	Source      Source
	Stages      []Stage
	Probability float64
	Basis       Basis
	Calibration string
	Suspicion   Suspicion
}

// AnalyzeOptions configures Analyze. Reference is the original carrier,
// when the caller has it. FFmpeg and FFprobe name the binaries the
// fingerprint stage runs, found on PATH when empty. Progress, when set,
// sees each stage as it finishes.
type AnalyzeOptions struct {
	Reference string
	FFmpeg    string
	FFprobe   string
	Progress  func(Stage)
}

// Probe reports what libav says about source without decoding it — the
// fields ffprobe shows for its audio stream. It fails for a file with no
// audio stream this FFmpeg build can decode.
func Probe(source string) (Source, error) {
	d, err := av.OpenDemuxer(source)
	if err != nil {
		return Source{}, fmt.Errorf("%w: %v", ErrCarrier, err)
	}
	defer func() { _ = d.Close() }()
	info := d.Info()
	if !av.CanDecode(info) {
		return Source{}, fmt.Errorf("%w: this FFmpeg build cannot decode %s", ErrUnsupportedCodec, codecName(info))
	}
	return sourceOf(info), nil
}

// Analyze runs every stage of the harness one file supports and reports
// the probability that it carries a Mist message. Blind, that probability
// comes from the detectors read against a calibration run, and stays near
// 0.5 wherever the harness found them at chance. With opts.Reference, the
// file is also diffed against Mist's own clean encode of that original,
// which is decisive when the two are the same recording.
func Analyze(ctx context.Context, source string, opts AnalyzeOptions) (Analysis, error) {
	cal, err := loadCalibration(calibrationJSON)
	if err != nil {
		return Analysis{}, err
	}
	raw, err := readSource(source)
	if err != nil {
		return Analysis{}, err
	}
	a := &analyzer{ctx: ctx, opts: opts, ext: filepath.Ext(source)}
	a.out.Basis = BasisBlind
	if err := a.run(raw, cal); err != nil {
		return Analysis{}, err
	}
	return a.out, nil
}

type analyzer struct {
	ctx  context.Context
	opts AnalyzeOptions
	ext  string
	out  Analysis
	lrs  []float64
}

func (a *analyzer) add(s Stage) {
	a.out.Stages = append(a.out.Stages, s)
	a.out.Suspicion = max(a.out.Suspicion, s.Suspicion)
	if s.Status == StageRan && s.LR > 0 {
		a.lrs = append(a.lrs, s.LR)
	}
	if a.opts.Progress != nil {
		a.opts.Progress(s)
	}
}

func (a *analyzer) run(raw []byte, cal calibration) error {
	suspect, err := decodeValues(raw)
	if err != nil {
		return err
	}
	a.out.Source = sourceOf(suspect.info)
	if !suspect.mistDomain() {
		a.add(Stage{Name: "format", Status: StageRan, Note: fmt.Sprintf(
			"%s is lossy and not Vorbis: Mist cannot write it, so it carries no Mist message", codecName(suspect.info))})
		a.out.Basis = BasisFormat
		return a.general(raw, suspect)
	}
	a.add(Stage{Name: "format", Status: StageRan, Note: domainNote(suspect.info)})
	format, note, ok := cal.lookup(suspect.info.CodecName, firstName(suspect.info.Container), suspect.lossless())
	if ok {
		a.out.Calibration = fmt.Sprintf("%s (%s, %d carriers, %s)", format.Format, cal.Corpus, cal.Carriers, cal.Commit)
		if note != "" {
			a.out.Calibration += "; " + note
		}
	}
	if err := a.detectors(suspect, format, ok); err != nil {
		return err
	}
	a.add(deadTail(suspect.vals))
	a.out.Probability = steganalysis.Posterior(a.lrs, analysisPrior)
	if a.opts.Reference == "" {
		a.add(a.fingerprint(raw, raw, a.ext, suspect))
		a.add(Stage{Name: "known cover", Status: StageNotRun, Note: "needs the original carrier"})
		a.add(Stage{Name: "sdr", Status: StageNotRun, Note: "needs the original carrier"})
	} else {
		ref, err := readSource(a.opts.Reference)
		if err != nil {
			return fmt.Errorf("reference: %w", err)
		}
		a.add(a.fingerprint(ref, raw, filepath.Ext(a.opts.Reference), suspect))
		if err := a.knownCover(ref, suspect); err != nil {
			return err
		}
	}
	for _, s := range []Stage{
		{Name: steganalysis.SelectionName, Status: StageNotRun, Note: "needs the recipient's public key"},
		{Name: "key-aware", Status: StageNotRun, Note: "needs the recipient's public key"},
		{Name: "invariance", Status: StageNotRun, Note: "needs two embeddings of the original"},
		{Name: "perceptual", Status: StageNotRun, Note: "needs the original and ViSQOL or PEAQ"},
	} {
		a.add(s)
	}
	return a.general(raw, suspect)
}

// general runs the checks for data any tool may have hidden: the file's
// structure, the MP3 frames when it is one, and digital silence when its
// samples are exact.
func (a *analyzer) general(raw []byte, v values) error {
	checks := append([]forensics.Check{forensics.Container(raw)}, forensics.MP3(raw)...)
	if v.lossless() && len(v.pcm.Planes) > 0 {
		checks = append(checks, forensics.Silence(planesOf(v), v.pcm.SampleRate))
	}
	for i, c := range checks {
		if err := a.ctx.Err(); err != nil {
			return err
		}
		if c.Ran || i == 0 {
			a.add(generalStage(c))
		}
	}
	if !v.lossless() {
		a.add(Stage{Name: "silence", Scope: ScopeAny, Status: StageNotRun, Note: "needs exact samples; a lossy decode is not"})
	}
	return nil
}

// generalStage turns a forensics check into a stage. forensics.Level and
// Suspicion count the same levels in the same order.
func generalStage(c forensics.Check) Stage {
	s := Stage{Name: c.Name, Scope: ScopeAny, Status: StageRan, Note: c.Context, Suspicion: Suspicion(c.Level()), Informational: c.Informational}
	if !c.Ran {
		s.Status = StageSkipped
		s.Note = "a container this check cannot read"
		return s
	}
	for _, f := range c.Findings {
		s.Findings = append(s.Findings, fmt.Sprintf("%s: %s", Suspicion(f.Level), f.Detail))
	}
	return s
}

// planesOf splits a lossless file's values back into one slice per channel.
func planesOf(v values) [][]int32 {
	n := len(v.pcm.Planes[0])
	out := make([][]int32, 0, len(v.pcm.Planes))
	for off := 0; off+n <= len(v.vals); off += n {
		out = append(out, v.vals[off:off+n])
	}
	return out
}

// detectors scores every chunk with the classical detectors and the trained
// wardens, and reads each file mean against the calibration.
func (a *analyzer) detectors(v values, format calibratedFormat, calibrated bool) error {
	chunks := featuresOf(v.vals, v.channels)
	if len(chunks) == 0 {
		for _, d := range steganalysis.Detectors() {
			a.add(Stage{Name: d.Name, Status: StageSkipped, Note: "no values to score"})
		}
		for _, w := range trainedWardens {
			a.add(Stage{Name: w.name, Status: StageSkipped, Note: "no values to score"})
		}
		return nil
	}
	for k, d := range steganalysis.Detectors() {
		if err := a.ctx.Err(); err != nil {
			return err
		}
		score := mean(chunks, func(c chunk) float64 { return c.features[k] })
		a.add(readAgainst(d.Name, score, format, calibrated))
	}
	for _, w := range trainedWardens {
		if err := a.ctx.Err(); err != nil {
			return err
		}
		ref, ok := format.stage(w.name)
		switch {
		case !calibrated || !ok || ref.Model == nil:
			a.add(Stage{Name: w.name, Status: StageSkipped, Note: "no trained model for this format"})
		case ref.Model.Dim() != len(w.pick(chunks[0])):
			a.add(Stage{Name: w.name, Status: StageSkipped, Note: "calibration is stale: rerun make calibrate"})
		default:
			score := mean(chunks, func(c chunk) float64 { return ref.Model.Score(w.pick(c)) })
			a.add(readAgainst(w.name, score, format, true))
		}
	}
	return nil
}

func readAgainst(name string, score float64, format calibratedFormat, calibrated bool) Stage {
	s := Stage{Name: name, Status: StageRan, Score: score}
	ref, ok := format.stage(name)
	if !calibrated || !ok {
		s.Note = "no calibration for this format: not evidence"
		return s
	}
	s.LR = ref.LikelihoodRatio(score)
	s.AUC = ref.AUC()
	return s
}

func deadTail(vals []int32) Stage {
	s := Stage{Name: "dead tail", Status: StageRan, Score: steganalysis.DeadTail(vals)}
	s.Note = "the end of the file is as busy as the start"
	if s.Score > 0 {
		s.Note = "the tail is constant while the head is not: embedding without filler, which Mist never does"
	}
	return s
}

// fingerprint re-encodes source with the ffmpeg CLI the way a plain user
// would, at the same settings, and diffs what the result shows without
// decoding against the suspect. Source is the suspect itself blind, and
// the original with a reference, which is the harness's own comparison.
func (a *analyzer) fingerprint(source, suspectRaw []byte, ext string, suspect values) Stage {
	s := Stage{Name: "fingerprint", Status: StageSkipped, Informational: true}
	ffmpeg, ffprobe := tool(a.opts.FFmpeg, "ffmpeg"), tool(a.opts.FFprobe, "ffprobe")
	if ffmpeg == "" || ffprobe == "" {
		s.Note = "needs ffmpeg and ffprobe on PATH"
		return s
	}
	twin, err := a.plainFFmpeg(ffmpeg, source, ext, suspect.info)
	if err != nil {
		s.Note = "ffmpeg could not re-encode it: " + err.Error()
		return s
	}
	want, err := traceBytes(a.ctx, ffprobe, twin)
	if err != nil {
		s.Note = "ffprobe could not read the re-encode: " + err.Error()
		return s
	}
	got, err := trace.Of(a.ctx, ffprobe, suspectRaw, traceInput(suspect))
	if err != nil {
		s.Note = "ffprobe could not read the file: " + err.Error()
		return s
	}
	s.Status = StageRan
	diffs := trace.Diff(want, got)
	s.Score = float64(len(diffs))
	if len(diffs) == 0 {
		s.Note = "matches a plain ffmpeg encode field for field"
		return s
	}
	s.Note = "differs from a plain ffmpeg encode in " + strings.Join(diffs, ", ") +
		"; not evidence alone, since any other encoder differs too"
	return s
}

func (a *analyzer) plainFFmpeg(ffmpeg string, source []byte, ext string, info av.AudioInfo) ([]byte, error) {
	target, err := targetOf(info)
	if err != nil {
		return nil, err
	}
	if target.Lossless {
		return trace.Encode(a.ctx, ffmpeg, source, ext, target.Container, target.CodecName)
	}
	pcm, srcInfo, err := decodeCarrier(bytes.NewReader(source))
	if err != nil {
		return nil, err
	}
	level, err := vorbisLevel(target, pcm, srcInfo)
	if err != nil {
		return nil, err
	}
	return trace.Encode(a.ctx, ffmpeg, source, ext, target.Container, "", "-q:a", strconv.Itoa(level))
}

// knownCover encodes the reference through Mist's own pipeline with
// nothing embedded and diffs the suspect's values against it. Encoding is
// deterministic, so a clean file made from the same original matches
// exactly, and a Mist file differs by sparse changes at most at Mist's rate.
func (a *analyzer) knownCover(refRaw []byte, suspect values) error {
	refPCM, refInfo, err := decodeCarrier(bytes.NewReader(refRaw))
	if err != nil {
		return fmt.Errorf("reference: %w", err)
	}
	sdr := Stage{Name: "sdr", Status: StageSkipped, Note: "the reference plays at another rate or channel count"}
	if refPCM.SampleRate == suspect.pcm.SampleRate && refPCM.Channels == suspect.pcm.Channels {
		sdr.Status = StageRan
		sdr.Score = quality.SDR(refPCM.Planes, suspect.pcm.Planes, refPCM.SampleRate/10)
		sdr.Note = "signal-to-distortion against the original, in dB"
		if math.IsInf(sdr.Score, 1) {
			sdr.Note = "the audio is sample for sample the original"
		}
	}
	cover := Stage{Name: "known cover", Status: StageSkipped}
	target, err := targetOf(suspect.info)
	if err != nil {
		cover.Note = "Mist cannot write this file's format: " + err.Error()
		a.add(cover)
		a.add(sdr)
		return nil
	}
	clean, err := plainEncode(target, refPCM, refInfo)
	if err != nil {
		return fmt.Errorf("reference: %w", err)
	}
	want, err := decodeValues(clean)
	if err != nil {
		return fmt.Errorf("reference: %w", err)
	}
	cover = coverDiff(want.vals, suspect.vals, suspect.lossless())
	switch cover.Score {
	case 0:
		a.out.Probability, a.out.Basis = knownClean, BasisKnownCover
	case 1:
		a.out.Probability, a.out.Basis = knownStego, BasisKnownCover
	}
	cover.Score = 0
	a.add(cover)
	a.add(sdr)
	return nil
}

// coverDiff compares the suspect's values with a clean encode's. Score is 0
// when they match, 1 when they differ the way Mist changes a file (a share
// no larger than stego.Density, every change ±1 for samples), and -1 when
// the reference is not the file the suspect was made from.
func coverDiff(clean, suspect []int32, lossless bool) Stage {
	s := Stage{Name: "known cover", Status: StageRan, Score: -1}
	if len(clean) != len(suspect) {
		s.Note = fmt.Sprintf("not the same recording: %d values against %d; ignored", len(clean), len(suspect))
		return s
	}
	var changed, unit int
	for i, v := range clean {
		if d := suspect[i] - v; d != 0 {
			changed++
			if d == 1 || d == -1 {
				unit++
			}
		}
	}
	if changed == 0 {
		s.Score = 0
		s.Note = fmt.Sprintf("identical to a clean encode of the original in all %d values", len(clean))
		return s
	}
	share := float64(changed) / float64(len(clean))
	pct := strconv.FormatFloat(share*100, 'g', 3, 64) + "%"
	if share > stego.Density || (lossless && unit != changed) {
		s.Note = fmt.Sprintf("%s of values differ, beyond what Mist changes: not the original of this file, "+
			"or another encoder build; ignored", pct)
		return s
	}
	s.Score = 1
	s.Note = fmt.Sprintf("%d of %d values (%s) differ from a clean encode of the original", changed, len(clean), pct)
	if lossless {
		s.Note += ", every one by exactly ±1"
	}
	return s
}

// values is a file as the wardens read it: lossless output as integer
// samples on the encoder's grid, plane after plane, and Vorbis as its
// eligible residue indices, which have no channel layout.
type values struct {
	pcm      codec.PCM
	info     av.AudioInfo
	vals     []int32
	channels int
}

func (v values) lossless() bool { return av.Lossless(v.info.NativeCodecID) }

func (v values) mistDomain() bool { return v.lossless() || v.info.CodecID == av.CodecIDVorbis }

func decodeValues(raw []byte) (values, error) {
	pcm, info, err := decodeCarrier(bytes.NewReader(raw))
	if err != nil {
		return values{}, err
	}
	v := values{pcm: pcm, info: info, channels: 1}
	if v.lossless() {
		v.vals, v.channels = gridValues(pcm.Planes, av.SampleScale(info.SampleFmt)), info.Channels
		return v, nil
	}
	if info.CodecID != av.CodecIDVorbis {
		return v, nil
	}
	_, pkts, err := readPackets(bytes.NewReader(raw))
	if err != nil {
		return values{}, err
	}
	vc := vorbis.New()
	if err := vc.Load(info.Extradata); err != nil {
		return values{}, fmt.Errorf("%w: setup: %v", ErrCarrier, err)
	}
	v.vals, err = stego.EligibleValues(vc, pkts)
	return v, err
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
// second-difference Markov and rich features the other wardens train on.
type chunk struct {
	features, markov, rich []float64
}

func featuresOf(v []int32, channels int) []chunk {
	var out []chunk
	for off := 0; off < len(v); off += chunkValues {
		c := v[off:min(off+chunkValues, len(v))]
		out = append(out, chunk{
			features: steganalysis.Features(c),
			markov:   steganalysis.Markov(c),
			rich:     steganalysis.RichPlanar(c, channels),
		})
	}
	return out
}

// trainedWardens are the logistic wardens, each with the features of a
// chunk it trains on.
var trainedWardens = []struct {
	name string
	pick func(chunk) []float64
}{
	{classifierName, func(c chunk) []float64 { return c.features }},
	{markovName, func(c chunk) []float64 { return c.markov }},
	{steganalysis.RichName, func(c chunk) []float64 { return c.rich }},
}

func mean(chunks []chunk, score func(chunk) float64) float64 {
	var sum float64
	for _, c := range chunks {
		sum += score(c)
	}
	return sum / float64(len(chunks))
}

// targetOf is the output format that would have written a file: its own
// container and codec, and for Vorbis the Ogg target Mist writes.
func targetOf(info av.AudioInfo) (av.Format, error) {
	if info.CodecID == av.CodecIDVorbis {
		return lookupFormat(DefaultFormat, "")
	}
	return lookupFormat(firstName(info.Container), info.CodecName)
}

// vorbisLevel is the Vorbis quality level Mist opens its encoder at for a
// carrier. It asks the encoder: libvorbis writes a nominal rate of 0 at
// some sample rates, so matching rates cannot tell the levels apart.
func vorbisLevel(target av.Format, pcm codec.PCM, info av.AudioInfo) (int, error) {
	enc, err := openEncoder(target, pcm, info)
	if err != nil {
		return 0, err
	}
	defer func() { _ = enc.Close() }()
	return enc.Info().Quality, nil
}

// plainEncode is the carrier through Mist's own encoder with nothing
// embedded, including the lossless grid snap Embed does before it changes
// a sample, so it differs from Embed's output by the embedding alone. It
// snaps pcm in place.
func plainEncode(target av.Format, pcm codec.PCM, info av.AudioInfo) ([]byte, error) {
	enc, err := openEncoder(target, pcm, info)
	if err != nil {
		return nil, err
	}
	defer func() { _ = enc.Close() }()
	if target.Lossless {
		if err := enc.Snap(pcm.Planes); err != nil {
			return nil, err
		}
	}
	rc, err := encodeAndMux(enc, pcm, nil)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rc.Close() }()
	return io.ReadAll(rc)
}

func traceBytes(ctx context.Context, ffprobe string, raw []byte) (trace.Trace, error) {
	v, err := decodeValues(raw)
	if err != nil {
		return trace.Trace{}, err
	}
	return trace.Of(ctx, ffprobe, raw, traceInput(v))
}

func traceInput(v values) trace.Decoded {
	return trace.Decoded{
		Planes: v.pcm.Planes, SampleRate: v.pcm.SampleRate,
		SampleFmt: v.info.SampleFmt.String(), Bitrate: v.info.Bitrate,
	}
}

func readSource(source string) ([]byte, error) {
	rc, err := openSource(source)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rc.Close() }()
	return io.ReadAll(rc)
}

func domainNote(info av.AudioInfo) string {
	if av.Lossless(info.NativeCodecID) {
		return codecName(info) + " is lossless: Mist could have written it, carrying bits in sample LSBs"
	}
	return "Vorbis: Mist could have written it, carrying bits in residue indices"
}

// firstName is the first of the comma-separated names libav gives a
// demuxer that reads several formats ("mov,mp4,m4a,...").
func firstName(names string) string {
	name, _, _ := strings.Cut(names, ",")
	return name
}

func tool(path, name string) string {
	if path == "" {
		path = name
	}
	found, err := exec.LookPath(path)
	if err != nil {
		return ""
	}
	return found
}

// Decisive reports whether a stage's likelihood ratio is at least 2 to 1
// in either direction, enough to count as a finding.
func (s Stage) Decisive() bool {
	return s.Status == StageRan && s.LR > 0 && math.Abs(math.Log(s.LR)) >= math.Log(2)
}

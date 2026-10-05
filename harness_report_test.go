//go:build harness

package mist

import (
	"cmp"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"math"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/iSerganov/mist/internal/steganalysis"
)

type harnessReport struct {
	Commit     string         `json:"commit"`
	Date       string         `json:"date"`
	Corpus     string         `json:"corpus"`
	Carriers   int            `json:"carriers"`
	Perceptual string         `json:"perceptual,omitempty"`
	Manifests  []runManifest  `json:"manifests"`
	Formats    []formatReport `json:"formats"`
}

type formatReport struct {
	Format         string            `json:"format"`
	Run            int               `json:"run"`
	Error          string            `json:"error,omitempty"`
	Measured       int               `json:"measured"`
	Skipped        []skippedCarrier  `json:"skipped,omitempty"`
	Detectors      []detectorResult  `json:"detectors,omitempty"`
	Embedding      []detectorResult  `json:"embedding_detectors,omitempty"`
	Scaling        []scalingRow      `json:"scaling,omitempty"`
	Categories     []categoryRow     `json:"categories,omitempty"`
	Pooled         []pooledRow       `json:"pooled,omitempty"`
	PowerFamilies  int               `json:"power_families"`
	PowerReached   bool              `json:"power_reached"`
	WorstCategory  string            `json:"worst_category,omitempty"`
	WorstDetector  string            `json:"worst_detector,omitempty"`
	WorstFileAUC   float64           `json:"worst_file_auc,omitempty"`
	LeaveLineage   []leaveLineageRow `json:"leave_lineage,omitempty"`
	CNN            *cnnResult        `json:"cnn,omitempty"`
	Raw            rawFormatScores   `json:"raw_scores,omitempty"`
	Traces         []carrierTrace    `json:"traces,omitempty"`
	Transcode      stat              `json:"transcode_sdr"`
	Stego          stat              `json:"stego_sdr"`
	Gap            stat              `json:"gap"`
	Added          stat              `json:"added_sdr"`
	PerceptualDrop *stat             `json:"perceptual_drop,omitempty"`
	Carriers       []carrierResult   `json:"carriers,omitempty"`
}

type rawFormatScores struct {
	Operational []rawDetectorScores `json:"operational,omitempty"`
	Embedding   []rawDetectorScores `json:"embedding,omitempty"`
}

type rawDetectorScores struct {
	Name    string         `json:"name"`
	Stego   rawPopulation  `json:"stego"`
	Clean   rawPopulation  `json:"clean"`
	Minimal *rawPopulation `json:"minimal,omitempty"`
}

type rawPopulation struct {
	Chunks []rawScore `json:"chunks,omitempty"`
	Files  []rawScore `json:"files,omitempty"`
}

type rawScore struct {
	Carrier  string  `json:"carrier"`
	Category string  `json:"category"`
	Lineage  string  `json:"lineage"`
	Chunk    int     `json:"chunk,omitempty"`
	Score    float64 `json:"score"`
}

// scalingRow is every detector's result on the first Chunks chunks of each
// carrier.
type scalingRow struct {
	Chunks    int              `json:"chunks"`
	Detectors []detectorResult `json:"detectors"`
}

// cnnResult is the external CNN warden's score for a format, read from the
// cnn.json that tools/cnn_warden writes.
type cnnResult struct {
	Files  int        `json:"files"`
	AUC    float64    `json:"auc"`
	Lo     float64    `json:"lo"`
	Hi     float64    `json:"hi"`
	Epochs int        `json:"epochs"`
	Folds  int        `json:"folds,omitempty"`
	Length int        `json:"length,omitempty"`
	Batch  int        `json:"batch,omitempty"`
	Rounds int        `json:"rounds,omitempty"`
	Seed   int        `json:"seed,omitempty"`
	Python string     `json:"python,omitempty"`
	NumPy  string     `json:"numpy,omitempty"`
	Torch  string     `json:"torch,omitempty"`
	Scores []cnnScore `json:"scores,omitempty"`
}

type cnnScore struct {
	Carrier string  `json:"carrier"`
	Clean   float64 `json:"clean"`
	Stego   float64 `json:"stego"`
}

// attachCNN adds the CNN warden's results from path, if it exists. The
// export directory names a format with "-" where the format has "/".
func (r *harnessReport) attachCNN(path string) error {
	raw, err := os.ReadFile(path)
	if errors.Is(err, fs.ErrNotExist) {
		return nil
	}
	if err != nil {
		return err
	}
	var byFormat map[string]cnnResult
	if err := json.Unmarshal(raw, &byFormat); err != nil {
		return fmt.Errorf("cnn.json: %w", err)
	}
	for i, f := range r.Formats {
		if c, ok := byFormat[strings.ReplaceAll(f.Format, "/", "-")]; ok {
			r.Formats[i].CNN = &c
		}
	}
	return nil
}

func (f formatReport) cnnTable(b *strings.Builder) {
	if f.CNN == nil {
		return
	}
	elo, ehi := detectorResult{FileLo: f.CNN.Lo, FileHi: f.CNN.Hi}.epsilonRange()
	fmt.Fprintf(b, "\n### A learned warden\n\nA small CNN trained on the first 16 chunks of each carrier, folds split by lineage (by carrier only when the export has one lineage), one score per file (`tools/cnn_warden`).\n\n"+
		"| Detector | Carriers | File AUC (95%%) | Detector-implied benchmark KL lower bound (nats, 95%%) |\n|---|---|---|---|\n| cnn | %d | %.3f (%.3f–%.3f) | %.3f (%.3f–%.3f) |\n",
		f.CNN.Files, f.CNN.AUC, f.CNN.Lo, f.CNN.Hi, epsilon(f.CNN.AUC), elo, ehi)
}

// pooledRow is each detector's AUC when the warden pools Files files.
type pooledRow struct {
	Files     int            `json:"files"`
	Detectors []pooledResult `json:"detectors"`
}

type pooledResult struct {
	Name string  `json:"name"`
	AUC  float64 `json:"auc"`
}

// categoryRow is every detector's result on one corpus category.
type categoryRow struct {
	Name      string           `json:"name"`
	Carriers  int              `json:"carriers"`
	Detectors []detectorResult `json:"detectors"`
}

// leaveLineageRow is the classifier's file AUC inside one lineage and on
// every other carrier. It is computed from scores already produced; the
// model is not retrained.
type leaveLineageRow struct {
	Lineage       string  `json:"lineage"`
	Carriers      int     `json:"carriers"`
	FileAUC       float64 `json:"file_auc"`
	Detectability float64 `json:"d_auc"`
	RestFileAUC   float64 `json:"rest_file_auc"`
	RestCarriers  int     `json:"rest_carriers"`
}

// carrierResult is one carrier's quality numbers, so a track that behaves
// unlike the rest shows up instead of vanishing into the mean.
type carrierResult struct {
	Name      string  `json:"name"`
	Kbps      float64 `json:"kbps"`
	Transcode num     `json:"transcode_sdr"`
	Stego     num     `json:"stego_sdr"`
	Gap       num     `json:"gap"`
	Added     num     `json:"added_sdr"`
}

type skippedCarrier struct {
	Name   string `json:"name"`
	Reason string `json:"reason"`
}

type detectorResult struct {
	Name          string  `json:"name"`
	AUC           float64 `json:"auc"`
	Lo            float64 `json:"auc_lo"`
	Hi            float64 `json:"auc_hi"`
	FileAUC       float64 `json:"file_auc"`
	FileLo        float64 `json:"file_auc_lo"`
	FileHi        float64 `json:"file_auc_hi"`
	Detectability float64 `json:"d_auc"`
	RecordLo      float64 `json:"record_auc_lo"`
	RecordHi      float64 `json:"record_auc_hi"`
	PermP         float64 `json:"permutation_p"`
	HolmP         float64 `json:"holm_p"`
	FDRP          float64 `json:"bh_fdr"`
	Invariance    float64 `json:"payload_invariance"`
}

// epsilon is the smallest Cachin ε that a detector reaching this AUC
// proves: |AUC−½| is at most the total variation between clean and stego,
// and Pinsker's inequality turns total variation δ into ε ≥ 2δ² nats.
func epsilon(auc float64) float64 {
	d := auc - 0.5
	return 2 * d * d
}

// epsilonRange is the ε lower bound across the file-level interval: zero
// when the interval reaches chance, else from its nearest end.
func (d detectorResult) epsilonRange() (lo, hi float64) {
	if d.FileLo <= 0.5 && d.FileHi >= 0.5 {
		lo = 0
	} else {
		lo = min(epsilon(d.FileLo), epsilon(d.FileHi))
	}
	return lo, max(epsilon(d.FileLo), epsilon(d.FileHi))
}

type stat struct {
	Mean  num `json:"mean"`
	Worst num `json:"worst"`
}

type num float64

func (n num) MarshalJSON() ([]byte, error) {
	if math.IsInf(float64(n), 0) || math.IsNaN(float64(n)) {
		return []byte("null"), nil
	}
	return json.Marshal(float64(n))
}

func (n *num) UnmarshalJSON(b []byte) error {
	if string(b) == "null" {
		*n = num(math.NaN())
		return nil
	}
	return json.Unmarshal(b, (*float64)(n))
}

func (n num) format(unit string) string {
	switch {
	case math.IsInf(float64(n), 1):
		return "∞"
	case math.IsNaN(float64(n)):
		return "—"
	}
	return fmt.Sprintf("%.2f%s", float64(n), unit)
}

// summarize takes the mean and worst of the finite values. A bit-exact
// re-encode has an infinite SDR, which says nothing about cost and would
// turn every aggregate that includes it into infinity or NaN.
func summarize(v []float64, worst func([]float64) float64) stat {
	var finite []float64
	for _, x := range v {
		if !math.IsInf(x, 0) && !math.IsNaN(x) {
			finite = append(finite, x)
		}
	}
	if len(finite) == 0 {
		return stat{Mean: num(math.NaN()), Worst: num(math.NaN())}
	}
	var sum float64
	for _, x := range finite {
		sum += x
	}
	return stat{Mean: num(sum / float64(len(finite))), Worst: num(worst(finite))}
}

// refreshStats recomputes the quality aggregates from the per-carrier
// numbers, for a report read back from JSON.
func (f *formatReport) refreshStats() {
	pick := func(get func(carrierResult) num) []float64 {
		var out []float64
		for _, c := range f.Carriers {
			out = append(out, float64(get(c)))
		}
		return out
	}
	f.Transcode = summarize(pick(func(c carrierResult) num { return c.Transcode }), minOf)
	f.Stego = summarize(pick(func(c carrierResult) num { return c.Stego }), minOf)
	f.Gap = summarize(pick(func(c carrierResult) num { return c.Gap }), maxOf)
	f.Added = summarize(pick(func(c carrierResult) num { return c.Added }), minOf)
}

func minOf(v []float64) float64 { return slices.Min(v) }

func maxOf(v []float64) float64 { return slices.Max(v) }

func readReport(path string) (*harnessReport, error) {
	if path == "" {
		return nil, nil
	}
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var r harnessReport
	return &r, json.Unmarshal(b, &r)
}

func (r harnessReport) write(dir string, baseline *harnessReport) (string, error) {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", err
	}
	cnn := cmp.Or(os.Getenv("MIST_HARNESS_CNN"), filepath.Join(dir, "cnn.json"))
	if err := r.attachCNN(cnn); err != nil {
		return "", err
	}
	js, err := json.MarshalIndent(r, "", "  ")
	if err != nil {
		return "", err
	}
	if err := os.WriteFile(filepath.Join(dir, "report.json"), append(js, '\n'), 0o644); err != nil {
		return "", err
	}
	manifestBundle := struct {
		Schema int           `json:"schema"`
		Runs   []runManifest `json:"runs"`
	}{Schema: harnessReportSchema, Runs: r.Manifests}
	manifest, err := json.MarshalIndent(manifestBundle, "", "  ")
	if err != nil {
		return "", err
	}
	if err := os.WriteFile(filepath.Join(dir, "manifest.json"), append(manifest, '\n'), 0o644); err != nil {
		return "", err
	}
	scores := struct {
		Schema  int    `json:"schema"`
		Commit  string `json:"commit"`
		Formats []struct {
			Format string          `json:"format"`
			Run    int             `json:"run"`
			Raw    rawFormatScores `json:"scores"`
		} `json:"formats"`
	}{Schema: harnessReportSchema, Commit: r.Commit}
	for _, f := range r.Formats {
		scores.Formats = append(scores.Formats, struct {
			Format string          `json:"format"`
			Run    int             `json:"run"`
			Raw    rawFormatScores `json:"scores"`
		}{Format: f.Format, Run: f.Run, Raw: f.Raw})
	}
	scoreJSON, err := json.MarshalIndent(scores, "", "  ")
	if err != nil {
		return "", err
	}
	if err := os.WriteFile(filepath.Join(dir, "scores.json"), append(scoreJSON, '\n'), 0o644); err != nil {
		return "", err
	}
	md := r.markdown(baseline)
	return md, os.WriteFile(filepath.Join(dir, "report.md"), []byte(md), 0o644)
}

const (
	gapTarget       = 0.3
	gapBudget       = 1.0
	referenceGap    = 0.18
	inaudibleSDR    = 70
	aucDetectable   = 0.1
	invarianceSlack = 0.05
	classifierName  = "classifier"
	markovName      = "markov"
)

type level int

const (
	pass level = iota
	watch
	fail
)

func (l level) mark() string { return [...]string{"✅", "⚠️", "❌"}[l] }

var detectorTargets = map[string]string{
	"chi-square": "bit overwriting (not what Mist does)",
	"spa":        "bit overwriting (not what Mist does)",
	"rs":         "bit overwriting (not what Mist does)",
	"hcf-com":    "±1 changes, which is what Mist does",
	"classifier": "anything it can learn from Mist's own output",
	"markov":     "how the waveform's curvature changes from sample to sample",
	"key-aware":  "the ephemeral key in the first frame's envelope, read with the public key alone",
}

func (f formatReport) fingerprint() (level, string) {
	odd := map[string]bool{}
	var names []string
	n := 0
	for _, t := range f.Traces {
		d := t.differences()
		if len(d) > 0 {
			n++
		}
		for _, x := range d {
			if !odd[x] {
				odd[x] = true
				names = append(names, x)
			}
		}
	}
	if n == 0 {
		return pass, fmt.Sprintf("matches ffmpeg on all %d carriers", len(f.Traces))
	}
	return fail, fmt.Sprintf("differs on %d of %d carriers: %s", n, len(f.Traces), strings.Join(names, ", "))
}

// verdict judges both the chunk and the file score and reports the worse:
// a bias too slight to show in one chunk can add up over a whole file.
func (d detectorResult) verdict() (level, string) {
	l, t := verdictOf(d.AUC, d.Lo, d.Hi)
	if fl, ft := verdictOf(d.FileAUC, d.FileLo, d.FileHi); fl > l {
		return fl, ft + " per file"
	}
	return l, t
}

func verdictOf(auc, lo, hi float64) (level, string) {
	switch {
	case lo <= 0.5 && hi >= 0.5:
		return pass, "chance"
	case math.Abs(auc-0.5) < aucDetectable:
		return watch, "faint signal"
	}
	return fail, "detectable"
}

func (d detectorResult) invariance() (level, string) {
	if math.Abs(d.Invariance-0.5) <= invarianceSlack {
		return pass, "hidden"
	}
	return fail, "message size shows"
}

func (f formatReport) detection() (level, string) {
	worst, at := pass, detectorResult{}
	var leak *detectorResult
	for _, d := range f.Detectors {
		if l, _ := d.verdict(); l > worst {
			worst, at = l, d
		}
		if l, _ := d.invariance(); l == fail && leak == nil {
			leak = &d
		}
	}
	switch {
	case worst == fail:
		return fail, fmt.Sprintf("%s detects it (AUC %.3f)", at.Name, at.AUC)
	case leak != nil:
		return fail, fmt.Sprintf("%s tells message sizes apart (%.3f)", leak.Name, leak.Invariance)
	case worst == watch:
		return watch, fmt.Sprintf("faint signal from %s (AUC %.3f)", at.Name, at.AUC)
	}
	return pass, fmt.Sprintf("chance on all %d detectors", len(f.Detectors))
}

func (f formatReport) quality() (level, string) {
	gap := float64(f.Gap.Mean)
	switch {
	case float64(f.Added.Worst) >= inaudibleSDR:
		return pass, fmt.Sprintf("inaudible: the added error is at least %s below the music", f.Added.Worst.format(" dB"))
	case gap <= gapTarget:
		return pass, fmt.Sprintf("Mist's own error sits %s below the music; embedding costs %s, within the %.1f dB target",
			f.Added.Mean.format(" dB"), f.Gap.Mean.format(" dB"), gapTarget)
	case gap <= gapBudget:
		return watch, fmt.Sprintf("Mist's own error sits %s below the music; embedding costs %s (%s error), above the %.1f dB target",
			f.Added.Mean.format(" dB"), f.Gap.Mean.format(" dB"), extraError(f.Gap.Mean), gapTarget)
	}
	return fail, fmt.Sprintf("Mist's own error sits %s below the music; embedding costs %s (%s error), over the %.1f dB target",
		f.Added.Mean.format(" dB"), f.Gap.Mean.format(" dB"), extraError(f.Gap.Mean), gapTarget)
}

func extraError(gap num) string {
	if math.IsInf(float64(gap), 0) || math.IsNaN(float64(gap)) {
		return "—"
	}
	return fmt.Sprintf("%+.0f%%", (math.Pow(10, float64(gap)/10)-1)*100)
}

func (r harnessReport) markdown(baseline *harnessReport) string {
	var b strings.Builder
	manifest := r.primaryManifest()
	fmt.Fprintf(&b, "# Mist harness report\n\nCommit `%s` · %s · corpus: %s (%d carriers, %d independent lineage groups) · perceptual metric: %s",
		r.Commit, r.Date, r.Corpus, r.Carriers, manifest.Corpus.IndependentGroups, cmp.Or(r.Perceptual, "not installed"))
	if baseline != nil {
		fmt.Fprintf(&b, " · compared with commit `%s`", baseline.Commit)
	}
	b.WriteString("\n\nEvery carrier is encoded three times: by the ffmpeg command line at its own defaults (the *clean* copy, " +
		"which is what a warden without the original would compare against), by Mist's own encoder with nothing embedded " +
		"(Mist's *own* re-encode), and by Mist with a hidden message (the *stego* copy). The report asks whether the stego copy " +
		"differs from the clean one in plain properties, whether a detector can tell them apart, and how much worse it sounds. " +
		"[How to read this report](#how-to-read-this-report) explains every number and threshold.\n\n")
	b.WriteString("## Summary\n\n| Format | Carriers | Looks like ffmpeg? | Hidden from detectors? | Audio quality |\n|---|---|---|---|---|\n")
	for _, f := range r.Formats {
		if f.Error != "" || f.Measured == 0 {
			fmt.Fprintf(&b, "| %s | %d / %d | not measured | not measured | not measured |\n", f.Format, f.Measured, f.Measured+len(f.Skipped))
			continue
		}
		fl, ft := f.fingerprint()
		dl, dt := f.detection()
		ql, qt := f.quality()
		fmt.Fprintf(&b, "| %s | %d / %d | %s %s | %s %s | %s %s |\n",
			f.Format, f.Measured, f.Measured+len(f.Skipped), fl.mark(), ft, dl.mark(), dt, ql.mark(), qt)
	}
	for _, f := range r.Formats {
		f.markdown(&b, baseline.format(f.Format))
	}
	b.WriteString(legend)
	return b.String()
}

func (r harnessReport) primaryManifest() runManifest {
	if len(r.Manifests) == 0 {
		return runManifest{}
	}
	return r.Manifests[0]
}

func detectorTable(b *strings.Builder, ds []detectorResult, base *formatReport) {
	b.WriteString("| Detector | Looks for | Chunk AUC (95%) | File AUC (95%) | D | Detector-implied benchmark KL lower bound (nats, 95%) | Adjusted p | Verdict | Message size |")
	if base != nil {
		b.WriteString(" AUC change |")
	}
	b.WriteString("\n|---|---|---|---|---|---|---|---|---|")
	if base != nil {
		b.WriteString("---|")
	}
	for _, d := range ds {
		vl, vt := d.verdict()
		il, it := d.invariance()
		elo, ehi := d.epsilonRange()
		fmt.Fprintf(b, "\n| %s | %s | %.3f (%.3f–%.3f) | %.3f (%.3f–%.3f) | %.3f | %.3f (%.3f–%.3f) | %s | %s %s | %s %s (%.3f) |",
			d.Name, detectorTargets[d.Name], d.AUC, d.Lo, d.Hi, d.FileAUC, d.FileLo, d.FileHi,
			d.Detectability, epsilon(d.FileAUC), elo, ehi, d.adjustedP(), vl.mark(), vt, il.mark(), it, d.Invariance)
		if base != nil {
			b.WriteString(" " + base.aucDelta(d) + " |")
		}
	}
}

func (d detectorResult) adjustedP() string {
	switch {
	case d.HolmP >= 0:
		return fmt.Sprintf("Holm %.3f", d.HolmP)
	case d.FDRP >= 0:
		return fmt.Sprintf("BH %.3f", d.FDRP)
	default:
		return "—"
	}
}

// pooledTable shows AUC by how many files the warden pools. A detector whose
// AUC climbs from left to right gains from collecting files, which the
// square-root law predicts at a fixed embedding rate.
func (f formatReport) pooledTable(b *strings.Builder) {
	if len(f.Pooled) == 0 {
		return
	}
	b.WriteString("\n### Does detection grow with the number of files?\n\nAUC when the warden averages a detector's score over k files drawn at random from the corpus, stego against clean. Draws overlap, so there is no interval.\n\n| Detector |")
	rule := "|---|"
	for _, r := range f.Pooled {
		fmt.Fprintf(b, " %d file(s) |", r.Files)
		rule += "---|"
	}
	b.WriteString("\n" + rule)
	for k, d := range f.Pooled[0].Detectors {
		fmt.Fprintf(b, "\n| %s |", d.Name)
		for _, r := range f.Pooled {
			fmt.Fprintf(b, " %.3f |", r.Detectors[k].AUC)
		}
	}
	b.WriteString("\n")
}

// categoryTable shows file AUC per corpus category. A category with few
// carriers has a wide interval, so the count sits in the header.
func (f formatReport) categoryTable(b *strings.Builder) {
	if len(f.Categories) == 0 {
		return
	}
	b.WriteString("\n### By kind of audio\n\nFile AUC (95%) against the clean ffmpeg copy, per public category from the corpus manifest. The count is carriers; few carriers means a wide interval.\n\n| Detector |")
	rule := "|---|"
	for _, c := range f.Categories {
		fmt.Fprintf(b, " %s (%d) |", c.Name, c.Carriers)
		rule += "---|"
	}
	b.WriteString("\n" + rule)
	for k, d := range f.Categories[0].Detectors {
		fmt.Fprintf(b, "\n| %s |", d.Name)
		for _, c := range f.Categories {
			r := c.Detectors[k]
			fmt.Fprintf(b, " %.3f (%.3f–%.3f) |", r.FileAUC, r.FileLo, r.FileHi)
		}
	}
	b.WriteString("\n")
}

func (f formatReport) recordingNote(b *strings.Builder) {
	var notes []string
	for _, d := range f.Detectors {
		fileWidth := d.FileHi - d.FileLo
		recordWidth := d.RecordHi - d.RecordLo
		if math.Abs(fileWidth-recordWidth) > 0.01 {
			notes = append(notes, fmt.Sprintf("%s %.3f–%.3f", d.Name, d.RecordLo, d.RecordHi))
		}
	}
	if len(notes) == 0 {
		return
	}
	fmt.Fprintf(b, "\n\nRecording-cluster file interval, shown where its width differs from the lineage interval by more than 0.01: %s.", strings.Join(notes, "; "))
}

func (f formatReport) powerNote(b *strings.Builder) {
	if f.PowerReached {
		fmt.Fprintf(b, "\n\nPower: shifting this run's classifier file scores to D = %.2f, about %d independent lineages give 90%% power for a lineage-cluster interval to exclude 0.5. The figure is a simulation from this run's dispersion.", harnessPowerTarget, f.PowerFamilies)
		return
	}
	if f.PowerFamilies == 0 {
		return
	}
	fmt.Fprintf(b, "\n\nPower: shifting this run's classifier file scores to D = %.2f did not reach 90%% power by %d independent lineages. The search stopped there.", harnessPowerTarget, f.PowerFamilies)
}

func (f formatReport) worstNote(b *strings.Builder) {
	if f.WorstDetector == "" {
		return
	}
	fmt.Fprintf(b, "\n\nWorst cell: %s on %s, file AUC %.3f, D %.3f.",
		f.WorstDetector, f.WorstCategory, f.WorstFileAUC, steganalysis.Detectability(f.WorstFileAUC))
}

func (f formatReport) leaveLineageTable(b *strings.Builder) {
	if len(f.LeaveLineage) == 0 {
		return
	}
	b.WriteString("\n### Leave-one-lineage\n\nClassifier file scores already computed for this run. A lineage is listed when it has at least two carriers. Rest is every other carrier, and is left blank below two.\n\n| Lineage | Carriers | Within file AUC | D | Rest carriers | Rest file AUC |\n|---|---|---|---|---|---|\n")
	for _, row := range f.LeaveLineage {
		rest := "—"
		if row.RestCarriers >= 2 {
			rest = fmt.Sprintf("%.3f", row.RestFileAUC)
		}
		fmt.Fprintf(b, "| %s | %d | %.3f | %.3f | %d | %s |\n",
			row.Lineage, row.Carriers, row.FileAUC, row.Detectability, row.RestCarriers, rest)
	}
}

// scalingTable shows file AUC by how much audio each carrier contributes. A
// detector whose AUC climbs from left to right is gaining from length, as the
// square-root law predicts at a fixed embedding rate.
func (f formatReport) scalingTable(b *strings.Builder) {
	if len(f.Scaling) == 0 {
		return
	}
	b.WriteString("\n### Does detection grow with audio?\n\nFile AUC (95%) against the clean ffmpeg copy on the first chunks of each carrier. A chunk is 65,536 values, about 0.74 s of 44.1 kHz stereo.\n\n| Detector |")
	rule := "|---|"
	for _, r := range f.Scaling {
		fmt.Fprintf(b, " First %d chunks |", r.Chunks)
		rule += "---|"
	}
	b.WriteString(" Whole file |\n" + rule + "---|")
	for k, whole := range f.Detectors {
		if k >= len(f.Scaling[0].Detectors) {
			break
		}
		fmt.Fprintf(b, "\n| %s |", whole.Name)
		for _, r := range f.Scaling {
			d := r.Detectors[k]
			fmt.Fprintf(b, " %.3f (%.3f–%.3f) |", d.FileAUC, d.FileLo, d.FileHi)
		}
		fmt.Fprintf(b, " %.3f (%.3f–%.3f) |", whole.FileAUC, whole.FileLo, whole.FileHi)
	}
	b.WriteString("\n")
}

func (f formatReport) markdown(b *strings.Builder, base *formatReport) {
	fmt.Fprintf(b, "\n## %s\n\n", f.Format)
	if f.Error != "" {
		fmt.Fprintf(b, "Not measured: %s\n", f.Error)
		return
	}
	fmt.Fprintf(b, "Measured on %d of %d carriers.\n", f.Measured, f.Measured+len(f.Skipped))
	for _, s := range f.Skipped {
		fmt.Fprintf(b, "\n- Skipped `%s`: %s", s.Name, s.Reason)
	}
	if len(f.Skipped) > 0 {
		b.WriteString("\n")
	}
	if f.Measured == 0 {
		return
	}
	fl, ft := f.fingerprint()
	b.WriteString("\n### Does it look like a plain ffmpeg encode?\n\n" +
		"The ffmpeg column is ffmpeg at its own defaults. Ogg Vorbis keeps the source's quality, so it differs from ffmpeg's default q3 whenever the carrier maps to another level.\n\n" +
		"| Carrier | Source samples | Samples (ffmpeg / Mist) | Zero tail (ffmpeg / Mist) | Sample format (ffmpeg / Mist) | Nominal kbps (ffmpeg / Mist) | Differs in |\n" +
		"|---|---|---|---|---|---|---|")
	for _, t := range f.Traces {
		fmt.Fprintf(b, "\n| %s | %d | %d / %d | %d / %d | %s / %s | %.0f / %.0f | %s |", t.Name, t.Source,
			t.FFmpeg.Samples, t.Mist.Samples, t.FFmpeg.ZeroTail, t.Mist.ZeroTail,
			t.FFmpeg.SampleFmt, t.Mist.SampleFmt, t.FFmpeg.NominalKbps, t.Mist.NominalKbps,
			cmp.Or(strings.Join(t.differences(), ", "), "—"))
	}
	fmt.Fprintf(b, "\n\n**Verdict:** %s %s.\n", fl.mark(), ft)

	b.WriteString("\n### Can a detector tell?\n\nStego copy against the clean ffmpeg copy (for Ogg Vorbis, ffmpeg at the quality level Mist chose, so only the embedding differs).\n\n")
	detectorTable(b, f.Detectors, base)
	f.recordingNote(b)
	f.powerNote(b)
	f.worstNote(b)
	dl, dt := f.detection()
	fmt.Fprintf(b, "\n\n**Verdict:** %s %s.\n", dl.mark(), dt)
	b.WriteString("\n### The embedding alone\n\nStego copy against Mist's own re-encode, so only the embedded changes differ.\n\n")
	detectorTable(b, f.Embedding, nil)
	b.WriteString("\n")
	f.scalingTable(b)
	f.categoryTable(b)
	f.leaveLineageTable(b)
	f.pooledTable(b)
	f.cnnTable(b)

	b.WriteString("\n### How much does it cost in sound?\n\n| Measure | Mean | Worst | What it means |\n|---|---|---|---|\n")
	fmt.Fprintf(b, "| Plain re-encode SDR | %s | %s | Loss of Mist's own re-encode, with nothing embedded |\n",
		f.Transcode.Mean.format(" dB"), f.Transcode.Worst.format(" dB"))
	fmt.Fprintf(b, "| Mist output SDR | %s | %s | The same, with the message embedded |\n",
		f.Stego.Mean.format(" dB"), f.Stego.Worst.format(" dB"))
	fmt.Fprintf(b, "| Embedding cost | %s | %s | Mist's own share; target ≤ %.1f dB, design-doc reference %.2f dB |\n",
		f.Gap.Mean.format(" dB"), f.Gap.Worst.format(" dB"), gapTarget, referenceGap)
	fmt.Fprintf(b, "| Extra error energy | %s | %s | Embedding cost as error added on top of a plain re-encode |\n",
		extraError(f.Gap.Mean), extraError(f.Gap.Worst))
	fmt.Fprintf(b, "| Mist's error below the music | %s | %s | Stego copy against Mist's own re-encode: what embedding alone adds |\n",
		f.Added.Mean.format(" dB"), f.Added.Worst.format(" dB"))
	if base != nil {
		fmt.Fprintf(b, "| Embedding cost, earlier run | %s | %s | Same measure at commit being compared with |\n",
			base.Gap.Mean.format(" dB"), base.Gap.Worst.format(" dB"))
	}
	if f.PerceptualDrop != nil {
		fmt.Fprintf(b, "| Perceptual drop | %s | %s | Score lost, clean minus stego: ViSQOL MOS (1–5) or PEAQ ODG (−4–0) |\n",
			f.PerceptualDrop.Mean.format(""), f.PerceptualDrop.Worst.format(""))
	}
	ql, qt := f.quality()
	fmt.Fprintf(b, "\n**Verdict:** %s %s.\n", ql.mark(), qt)

	b.WriteString("\n### Per carrier\n\n| Carrier | Output kbps | Plain re-encode SDR | Mist output SDR | Embedding cost | Mist's error below the music |\n|---|---|---|---|---|---|\n")
	for _, c := range f.Carriers {
		fmt.Fprintf(b, "| %s | %.0f | %s | %s | %s | %s |\n", c.Name, c.Kbps,
			c.Transcode.format(" dB"), c.Stego.format(" dB"), c.Gap.format(" dB"), c.Added.format(" dB"))
	}
}

const legend = `
## How to read this report

### Plain properties

Before any statistics, a file shows how long it is, whether it ends in digital zeros, what sample format it
decodes to and what bitrate its header claims. The **ffmpeg** column is the same carrier encoded by the ffmpeg
command line at its own defaults, keeping only the first audio stream. Any difference from it tells Mist's output
apart from an ordinary file outright, however well the embedding hides, so every carrier must match.

### Detectability

**AUC** is the chance that a detector, shown one clean and one stego sample, picks the stego one. The main table
compares against the ffmpeg copy, the cover a warden would actually have. **The embedding alone** compares against
Mist's own re-encode, so it isolates the embedded changes from every other way Mist's pipeline differs from ffmpeg. 0.5 is a coin
flip, which is the goal; 1.0 means it is caught every time. A value well below 0.5 is a detection too: the
detector is right, just with its sign flipped. Ogg Vorbis is scored on the residues Mist may change, lossless
formats on the decoded samples.

Each carrier is cut into chunks of 65,536 values and every chunk is scored: that is the **chunk AUC**. The
**file AUC** averages each recording's chunk scores into one score per file. Chapters of one session stay
separate files and share a lineage. The **95% file interval** redraws lineages first and then the recordings
inside a drawn lineage. The chunk interval redraws whole lineages too, never single chunks. A recording-cluster
interval is printed beside a detector only when its width differs from the lineage interval by more than 0.01.
With one recording per lineage the two intervals match. Their width follows the number of lineages, not the
number of chunks. The verdict is the worse of the chunk and file verdicts.

**D** is 0.5 + |AUC − 0.5|. A detector that is perfectly wrong, AUC 0, has the same D as a detector that is
perfectly right. The file interval and the verdict already treat an interval that sits entirely below 0.5 as
detection; D puts that on one scale.

The **detector-implied benchmark KL lower bound** applies Pinsker's inequality to the file AUC:
|AUC − ½| is at most the total variation between the benchmark clean and stego populations, so
KL ≥ 2(AUC − ½)² nats. This is only weak attack evidence for this detector and benchmark. It is not an estimate
or upper bound for Cachin's ε; a detector at chance shows that this detector found no gap, not that KL is small.

Chi-square, SPA and RS look for bits being overwritten, which Mist never does, so they are expected to sit at
chance; a rise means the embedder has drifted. They are exploratory: the report gives each a paired lineage
label-swap p-value and a Benjamini-Hochberg adjustment across the three. HCF-COM looks for ±1 changes, which
is exactly what Mist does, so it sits in the confirmatory family with the classifier, the Markov model and the
key-aware warden. That family of four is adjusted with Holm. A dash means that row was not part of the
confirmatory test (scaling, category and embedding-only tables).

The **classifier** is the adversary of record. It is a logistic regression on this run's own clean and stego
chunks, using every detector's score, the share of values at each of -3…3, and how each step between adjacent
values follows the one before it. Outer folds are lineages. An inner grouped search picks the L2 penalty from
0.001, 0.01 and 0.1, and a further held-out lineage calibrates the score. Fewer than four lineages falls back
to the fixed-penalty cross-validation. Standardisation is fit on the training rows of that fold. The primary
operational comparison repeats that whole fit under nine lineage label swaps; scaling and category rows do not.
Nine refits make the smallest attainable p-value 0.1, so that row cannot by itself clear 0.05.
Its message-size check trains a second model to tell a 64-byte message from a 1-byte one.

**Markov** is the same kind of model, trained the same way, on one richer feature set only: how the second
difference between samples, the waveform's curvature, changes from one sample to the next, with each value
clipped to -3…3. That curvature is near zero wherever the audio is smooth, so ±1 changes stand out in it more
than in the values or their steps. These are the rich-model features of audio steganalysis; a detector that
learns its own features, a CNN trained on exported chunks, is the next step beyond them and is not run here.
When it is run, its folds follow lineage as well.

**Power** asks how many independent lineages this run's classifier dispersion would need before a shift to
D = 0.55 pushed the lineage interval off 0.5 in 90% of simulations. It is not a guarantee about a future corpus.
**Worst cell** is the highest D among category rows, or among the aggregate detectors when the corpus has one
category. **Leave-one-lineage** restricts the classifier's existing file scores to each lineage that has two or
more carriers, and to the carriers that remain. It does not retrain.

**Message size** compares a 64-byte message with a 1-byte one. Mist changes the same amount of audio whatever
the message, so this should read 0.5: anything else means the message length shows.

### Audio quality

**SDR** (signal-to-distortion ratio) is how loud the music is compared with the error added to it, in dB. Higher
is better, and every 10 dB means ten times less error. Both copies are compared with the original carrier,
after lining them up in time. The **plain re-encode** is Mist's own re-encode with nothing embedded; the **embedding cost** is how
much lower the stego copy scores, which is Mist's own share. **Extra error energy** says the same thing as a
percentage: +37% means 37% more error than the plain re-encode alone.

The embedding cost is relative to the re-encode's own error, so the same perturbation reads as a larger cost on a
cleaner, higher-bitrate re-encode. **Mist's error below the music** does not depend on that: it compares the
stego copy with Mist's own re-encode, so the only difference left is what Mist changed, measured against the music
itself. The **per carrier** table lists every track with its output bitrate, so one odd track cannot hide in the
mean.

### Thresholds

| Measure | ✅ | ⚠️ | ❌ |
|---|---|---|---|
| Plain properties | every carrier matches ffmpeg | — | any carrier differs |
| Detector AUC | 95% interval includes 0.5 | interval excludes 0.5, AUC within 0.4–0.6 | AUC outside 0.4–0.6 |
| Message size | 0.45–0.55 | — | outside 0.45–0.55 |
| Embedding cost | ≤ 0.3 dB, or Mist output ≥ 70 dB SDR | 0.3–1 dB | over 1 dB |

### Reference points

- **0.3 dB embedding cost** and **AUC ≈ 0.5** are the roadmap's Phase 1 exit criteria.
- **0.18 dB** is what the design doc measured for Ogg Vorbis on one 128 kbps rock MP3.
- **70 dB SDR** puts the error far below anything audible at normal listening levels. 16-bit audio's own
  rounding noise sits about 96 dB down, which is why a lossless output can score ∞ for a plain re-encode
  (bit-exact) and pass on SDR alone.
- **A lossy carrier into a lossless output** is not bit-exact: an MP3 decodes to values between 16-bit steps,
  so the plain re-encode lands around 80–90 dB rather than ∞.
`

func (r *harnessReport) format(name string) *formatReport {
	if r == nil {
		return nil
	}
	for i := range r.Formats {
		if r.Formats[i].Format == name {
			return &r.Formats[i]
		}
	}
	return nil
}

func (f *formatReport) aucDelta(d detectorResult) string {
	for _, bd := range f.Detectors {
		if bd.Name == d.Name {
			return fmt.Sprintf("%+.3f", d.AUC-bd.AUC)
		}
	}
	return "—"
}

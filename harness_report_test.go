//go:build harness

package mist

import (
	"cmp"
	"encoding/json"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"slices"
	"strings"
)

type harnessReport struct {
	Commit     string         `json:"commit"`
	Date       string         `json:"date"`
	Corpus     string         `json:"corpus"`
	Carriers   int            `json:"carriers"`
	Perceptual string         `json:"perceptual,omitempty"`
	Formats    []formatReport `json:"formats"`
}

type formatReport struct {
	Format         string           `json:"format"`
	Error          string           `json:"error,omitempty"`
	Measured       int              `json:"measured"`
	Skipped        []skippedCarrier `json:"skipped,omitempty"`
	Detectors      []detectorResult `json:"detectors,omitempty"`
	Embedding      []detectorResult `json:"embedding_detectors,omitempty"`
	Traces         []carrierTrace   `json:"traces,omitempty"`
	Transcode      stat             `json:"transcode_sdr"`
	Stego          stat             `json:"stego_sdr"`
	Gap            stat             `json:"gap"`
	Added          stat             `json:"added_sdr"`
	PerceptualDrop *stat            `json:"perceptual_drop,omitempty"`
	Carriers       []carrierResult  `json:"carriers,omitempty"`
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
	Name       string  `json:"name"`
	AUC        float64 `json:"auc"`
	Lo         float64 `json:"auc_lo"`
	Hi         float64 `json:"auc_hi"`
	Invariance float64 `json:"payload_invariance"`
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

func summarize(v []float64, worst func([]float64) float64) stat {
	var sum float64
	for _, x := range v {
		sum += x
	}
	return stat{Mean: num(sum / float64(len(v))), Worst: num(worst(v))}
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
	js, err := json.MarshalIndent(r, "", "  ")
	if err != nil {
		return "", err
	}
	if err := os.WriteFile(filepath.Join(dir, "report.json"), append(js, '\n'), 0o644); err != nil {
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

func (d detectorResult) verdict() (level, string) {
	switch {
	case d.Lo <= 0.5 && d.Hi >= 0.5:
		return pass, "chance"
	case math.Abs(d.AUC-0.5) < aucDetectable:
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
	case float64(f.Stego.Worst) >= inaudibleSDR:
		return pass, fmt.Sprintf("inaudible: the added error is at least %s below the music", f.Stego.Worst.format(" dB"))
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
	fmt.Fprintf(&b, "# Mist harness report\n\nCommit `%s` · %s · corpus: %s (%d carriers) · perceptual metric: %s",
		r.Commit, r.Date, r.Corpus, r.Carriers, cmp.Or(r.Perceptual, "not installed"))
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

func detectorTable(b *strings.Builder, ds []detectorResult, base *formatReport) {
	b.WriteString("| Detector | Looks for | AUC | 95% interval | Verdict | Message size |")
	if base != nil {
		b.WriteString(" AUC change |")
	}
	b.WriteString("\n|---|---|---|---|---|---|")
	if base != nil {
		b.WriteString("---|")
	}
	for _, d := range ds {
		vl, vt := d.verdict()
		il, it := d.invariance()
		fmt.Fprintf(b, "\n| %s | %s | %.3f | %.3f–%.3f | %s %s | %s %s (%.3f) |",
			d.Name, detectorTargets[d.Name], d.AUC, d.Lo, d.Hi, vl.mark(), vt, il.mark(), it, d.Invariance)
		if base != nil {
			b.WriteString(" " + base.aucDelta(d) + " |")
		}
	}
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
		"| Carrier | Source samples | Samples (ffmpeg / Mist) | Zero tail (ffmpeg / Mist) | Sample format (ffmpeg / Mist) | Nominal kbps (ffmpeg / Mist) | Differs in |\n" +
		"|---|---|---|---|---|---|---|")
	for _, t := range f.Traces {
		fmt.Fprintf(b, "\n| %s | %d | %d / %d | %d / %d | %s / %s | %.0f / %.0f | %s |", t.Name, t.Source,
			t.FFmpeg.Samples, t.Mist.Samples, t.FFmpeg.ZeroTail, t.Mist.ZeroTail,
			t.FFmpeg.SampleFmt, t.Mist.SampleFmt, t.FFmpeg.NominalKbps, t.Mist.NominalKbps,
			cmp.Or(strings.Join(t.differences(), ", "), "—"))
	}
	fmt.Fprintf(b, "\n\n**Verdict:** %s %s.\n", fl.mark(), ft)

	b.WriteString("\n### Can a detector tell?\n\nStego copy against the clean ffmpeg copy.\n\n")
	detectorTable(b, f.Detectors, base)
	dl, dt := f.detection()
	fmt.Fprintf(b, "\n\n**Verdict:** %s %s.\n", dl.mark(), dt)
	b.WriteString("\n### The embedding alone\n\nStego copy against Mist's own re-encode, so only the embedded changes differ.\n\n")
	detectorTable(b, f.Embedding, nil)
	b.WriteString("\n")

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
detector is right, just with its sign flipped. Each carrier is cut into chunks of 65,536 values and every chunk
is scored, so the **95% interval** says how sure the estimate is. Ogg Vorbis is scored on the residues Mist may
change, lossless formats on the decoded samples.

Chi-square, SPA and RS look for bits being overwritten, which Mist never does, so they are expected to sit at
chance; a rise means the embedder has drifted. HCF-COM looks for ±1 changes, which is exactly what Mist does,
so it is the classical detector that matters.

The **classifier** is the adversary of record. It is a logistic regression trained on this run's own clean and
stego chunks, using every detector's score, the share of values at each of -3…3, and how each step between
adjacent values follows the one before it. It is cross-validated by carrier: every carrier is scored by a model
trained without it, so it cannot win by memorising a track. Its message-size check trains a second model to
tell a 64-byte message from a 1-byte one.

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

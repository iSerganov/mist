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
	Transcode      stat             `json:"transcode_sdr"`
	Stego          stat             `json:"stego_sdr"`
	Gap            stat             `json:"gap"`
	PerceptualDrop *stat            `json:"perceptual_drop,omitempty"`
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

func (r harnessReport) markdown(baseline *harnessReport) string {
	var b strings.Builder
	fmt.Fprintf(&b, "# Mist harness report\n\n`%s` · %s · corpus: %s (%d carriers) · perceptual: %s",
		r.Commit, r.Date, r.Corpus, r.Carriers, cmp.Or(r.Perceptual, "not installed"))
	if baseline != nil {
		fmt.Fprintf(&b, " · baseline: `%s`", baseline.Commit)
	}
	b.WriteString("\n\nAUC is how well a detector tells Mist output from a plain transcode of the same carrier; " +
		"0.5 is chance. Payload invariance compares a 64-byte payload with a 1-byte one and should also sit at 0.5. " +
		"Gap is transcode SDR minus stego SDR: what embedding costs on top of the re-encode.\n")
	for _, f := range r.Formats {
		f.markdown(&b, baseline.format(f.Format))
	}
	return b.String()
}

func (f formatReport) markdown(b *strings.Builder, base *formatReport) {
	fmt.Fprintf(b, "\n## %s\n\n", f.Format)
	if f.Error != "" {
		fmt.Fprintf(b, "Not measured: %s\n", f.Error)
		return
	}
	fmt.Fprintf(b, "Measured on %d of %d carriers.", f.Measured, f.Measured+len(f.Skipped))
	for _, s := range f.Skipped {
		fmt.Fprintf(b, " Skipped `%s`: %s.", s.Name, s.Reason)
	}
	b.WriteString("\n")
	if f.Measured == 0 {
		return
	}
	b.WriteString("\n| Detector | AUC vs clean | 95% interval | Payload invariance |")
	if base != nil {
		b.WriteString(" Δ AUC vs baseline |")
	}
	b.WriteString("\n|---|---|---|---|")
	if base != nil {
		b.WriteString("---|")
	}
	for _, d := range f.Detectors {
		fmt.Fprintf(b, "\n| %s | %.3f | %.3f–%.3f | %.3f |", d.Name, d.AUC, d.Lo, d.Hi, d.Invariance)
		if base != nil {
			b.WriteString(" " + base.aucDelta(d) + " |")
		}
	}
	b.WriteString("\n\n| Quality | Mean | Worst |\n|---|---|---|\n")
	fmt.Fprintf(b, "| Transcode SDR (carrier → clean) | %s | %s |\n", f.Transcode.Mean.format(" dB"), f.Transcode.Worst.format(" dB"))
	fmt.Fprintf(b, "| Stego SDR (carrier → stego) | %s | %s |\n", f.Stego.Mean.format(" dB"), f.Stego.Worst.format(" dB"))
	fmt.Fprintf(b, "| Gap | %s | %s |\n", f.Gap.Mean.format(" dB"), f.Gap.Worst.format(" dB"))
	if base != nil {
		fmt.Fprintf(b, "| Gap, baseline | %s | %s |\n", base.Gap.Mean.format(" dB"), base.Gap.Worst.format(" dB"))
	}
	if f.PerceptualDrop != nil {
		fmt.Fprintf(b, "| Perceptual drop (clean − stego) | %s | %s |\n", f.PerceptualDrop.Mean.format(""), f.PerceptualDrop.Worst.format(""))
	}
}

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

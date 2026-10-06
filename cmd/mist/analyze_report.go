package main

import (
	"fmt"
	"math"
	"path/filepath"
	"strings"

	"charm.land/lipgloss/v2"
	"github.com/iSerganov/mist"
)

// palette is every style the analyze screens use, in the shades that read
// on the terminal's background.
type palette struct {
	title, bold, faint, good, warn, alarm, badge lipgloss.Style
}

// styles starts dark and follows the terminal once it reports its
// background.
var styles = newPalette(true)

func newPalette(dark bool) palette {
	ld := lipgloss.LightDark(dark)
	accent := ld(lipgloss.Color("#0087AF"), lipgloss.Color("#5FD7FF"))
	caution := ld(lipgloss.Color("#AF8700"), lipgloss.Color("#FFD75F"))
	return palette{
		title: lipgloss.NewStyle().Bold(true).Foreground(accent),
		bold:  lipgloss.NewStyle().Bold(true),
		faint: lipgloss.NewStyle().Foreground(ld(lipgloss.Color("#8A8A8A"), lipgloss.Color("#7C7C7C"))),
		good:  lipgloss.NewStyle().Foreground(ld(lipgloss.Color("#00875F"), lipgloss.Color("#5FD787"))),
		warn:  lipgloss.NewStyle().Foreground(caution),
		alarm: lipgloss.NewStyle().Foreground(ld(lipgloss.Color("#D70000"), lipgloss.Color("#FF5F5F"))),
		badge: lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#000000")).Background(caution).Padding(0, 1),
	}
}

// renderReport lays an Analysis out as the report screen: the file, every
// stage with its score and evidence, and a summary of what that adds up to.
func renderReport(a mist.Analysis, t track, reference string, width int) string {
	var b strings.Builder
	fmt.Fprintf(&b, "\n  %s %s %s\n", styles.title.Render("mist"), styles.faint.Render("· analysis ·"), styles.bold.Render(filepath.Base(t.path)))
	fmt.Fprintf(&b, "  %s\n", styles.faint.Render(trackDetails(t)))
	if reference != "" {
		fmt.Fprintf(&b, "  %s %s\n", styles.faint.Render("reference"), filepath.Base(reference))
	}
	b.WriteString("\n  " + styles.bold.Render("Mist's own embedding") + styles.faint.Render("  calibrated") + "\n")
	fmt.Fprintf(&b, "  %s\n", styles.faint.Render(fmt.Sprintf("  %-16s %7s %6s  %-21s %s", "stage", "score", "auc", "evidence", "")))
	for _, s := range a.Stages {
		if s.Scope == mist.ScopeMist {
			b.WriteString(stageLine(s, width))
		}
	}
	b.WriteString("\n  " + styles.bold.Render("Any steganography tool") + styles.faint.Render("  structure and format checks, not calibrated") + "\n")
	for _, s := range a.Stages {
		if s.Scope == mist.ScopeAny {
			b.WriteString(anyLine(s, width))
		}
	}
	b.WriteString("\n")
	b.WriteString(summaryBox(a, width))
	return b.String()
}

// anyLine is one row for a check that looks for any tool: its level and
// every finding, or what it looked at when it found nothing.
func anyLine(s mist.Stage, width int) string {
	const noteAt = 20
	name := fmt.Sprintf("%-16s ", s.Name)
	switch s.Status {
	case mist.StageNotRun:
		return "  " + styles.faint.Render("· "+name+wrapNote("not run: "+s.Note, width-noteAt)) + "\n"
	case mist.StageSkipped:
		return "  " + styles.warn.Render("–") + " " + name + styles.warn.Render(wrapNote("skipped: "+s.Note, width-noteAt)) + "\n"
	}
	if s.Informational {
		return "  " + styles.faint.Render("i") + " " + name + styles.faint.Render(wrapNote("context: "+s.Note, width-noteAt)) + "\n"
	}
	if s.Suspicion == mist.SuspicionNone {
		note := "nothing hidden found"
		if s.Note != "" {
			note += "; " + s.Note
		}
		return "  " + styles.good.Render("✓") + " " + name + styles.faint.Render(wrapNote(note, width-noteAt)) + "\n"
	}
	var b strings.Builder
	for i, f := range s.Findings {
		mark, label := " ", strings.Repeat(" ", len(name))
		if i == 0 {
			mark, label = suspicionStyle(s.Suspicion).Render("●"), name
		}
		b.WriteString("  " + mark + " " + label + suspicionStyle(findingLevel(f)).Render(wrapNote(f, width-noteAt)) + "\n")
	}
	return b.String()
}

// findingLevel reads back the level a finding was written with.
func findingLevel(f string) mist.Suspicion {
	for l := mist.SuspicionHigh; l > mist.SuspicionNone; l-- {
		if strings.HasPrefix(f, l.String()+":") {
			return l
		}
	}
	return mist.SuspicionNone
}

func suspicionStyle(l mist.Suspicion) lipgloss.Style {
	switch l {
	case mist.SuspicionHigh, mist.SuspicionMedium:
		return styles.alarm
	case mist.SuspicionLow:
		return styles.warn
	}
	return styles.good
}

// stageLine is one row of the stage table: a mark, the name, and either the
// score read against the calibration or why there is none.
func stageLine(s mist.Stage, width int) string {
	const noteAt = 20
	name := fmt.Sprintf("%-16s ", s.Name)
	switch s.Status {
	case mist.StageNotRun:
		return "  " + styles.faint.Render("· "+name+wrapNote("not run: "+s.Note, width-noteAt)) + "\n"
	case mist.StageSkipped:
		return "  " + styles.warn.Render("–") + " " + name + styles.warn.Render(wrapNote("skipped: "+s.Note, width-noteAt)) + "\n"
	}
	line := "  " + styles.good.Render("✓") + " " + name
	if s.LR > 0 {
		line += fmt.Sprintf("%7.3f %6.2f  %s %s", s.Score, s.AUC, evidenceBar(s.LR), leaning(s.LR))
		return line + "\n"
	}
	if s.Score != 0 {
		line += fmt.Sprintf("%7.3f  ", s.Score)
	}
	return line + styles.faint.Render(wrapNote(s.Note, width-noteAt-9)) + "\n"
}

// evidenceBar draws a likelihood ratio around a centre line: clean to the
// left, stego to the right, full at the cap mist puts on one stage.
func evidenceBar(lr float64) string {
	const half = 10
	n := min(half, int(math.Round(math.Abs(math.Log(lr))/math.Log(mist.MaxLR)*half)))
	pad := styles.faint.Render(strings.Repeat("░", half-n))
	empty := styles.faint.Render(strings.Repeat("░", half))
	if lr < 1 {
		return pad + styles.good.Render(strings.Repeat("█", n)) + styles.faint.Render("│") + empty
	}
	return empty + styles.faint.Render("│") + styles.alarm.Render(strings.Repeat("█", n)) + pad
}

func leaning(lr float64) string {
	switch {
	case lr >= 2:
		return styles.alarm.Render(fmt.Sprintf("×%.1f stego", lr))
	case lr <= 0.5:
		return styles.good.Render(fmt.Sprintf("×%.1f clean", 1/lr))
	default:
		return styles.faint.Render("neutral")
	}
}

func summaryBox(a mist.Analysis, width int) string {
	boxWidth := max(40, min(width-4, 100))
	text := boxWidth - 8
	found, clear, notRun := findings(a)
	var b strings.Builder
	b.WriteString(styles.bold.Render("Summary") + "\n\n")
	section := func(title string, lines []string, mark string) {
		if len(lines) == 0 {
			return
		}
		b.WriteString(styles.bold.Render(title) + "\n")
		for _, l := range lines {
			wrapped := lipgloss.NewStyle().Width(text - 4).Render(l)
			b.WriteString("  " + mark + " " + strings.ReplaceAll(wrapped, "\n", "\n    ") + "\n")
		}
		b.WriteString("\n")
	}
	section("Found", found, styles.alarm.Render("●"))
	section("Not found", clear, styles.good.Render("○"))
	if len(notRun) > 0 {
		b.WriteString(styles.faint.Render("Not run: "+strings.Join(notRun, "; ")) + "\n\n")
	}
	pct := probabilityStyle(a.Probability).Bold(true).Render(fmt.Sprintf("%.0f%%", a.Probability*100))
	fmt.Fprintf(&b, "Mist message:          %s  %s\n", pct, styles.faint.Render(verdict(a)))
	level := suspicionStyle(a.Suspicion).Bold(true).Render(strings.ToUpper(a.Suspicion.String()))
	fmt.Fprintf(&b, "Other steganography:   %s  %s\n\n", level, styles.faint.Render(otherVerdict(a.Suspicion)))
	b.WriteString(styles.faint.Render(basisNote(a)))
	border := probabilityStyle(a.Probability)
	if a.Suspicion >= mist.SuspicionMedium {
		border = styles.alarm
	}
	box := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(border.GetForeground()).
		Padding(1, 2).
		Width(boxWidth)
	return lipgloss.NewStyle().MarginLeft(2).Render(box.Render(b.String())) + "\n"
}

// findings sorts the stages into what points at a message, what points
// away from one, and what could not be asked, in plain sentences.
func findings(a mist.Analysis) (found, clear, notRun []string) {
	var neutral, quiet, unusual []string
	for _, s := range a.Stages {
		switch {
		case s.Scope == mist.ScopeAny && s.Status == mist.StageRan:
			if s.Suspicion == mist.SuspicionNone && !s.Informational {
				quiet = append(quiet, s.Name)
			}
			for _, f := range s.Findings {
				l := findingLevel(f)
				text := strings.TrimPrefix(f, l.String()+": ")
				if l >= mist.SuspicionMedium {
					found = append(found, text)
				} else {
					unusual = append(unusual, text)
				}
			}
		case s.Status == mist.StageNotRun:
			notRun = append(notRun, s.Name+" ("+s.Note+")")
		case s.Status == mist.StageSkipped:
		case s.Name == "format" && a.Basis == mist.BasisFormat:
			clear = append(clear, s.Note)
		case s.Name == "known cover" && a.Basis == mist.BasisKnownCover:
			if a.Probability > 0.5 {
				found = append(found, s.Note+", the way Mist changes a file")
			} else {
				clear = append(clear, s.Note)
			}
		case s.Name == "known cover":
			clear = append(clear, "the reference did not decide it: "+s.Note)
		case s.Name == "fingerprint" && s.Score > 0:
			found = append(found, "the file "+s.Note)
		case s.Name == "fingerprint":
			clear = append(clear, "nothing in its container or tags differs from a plain ffmpeg encode")
		case s.Name == "dead tail" && s.Score > 0:
			found = append(found, s.Note)
		case s.Decisive() && s.LR > 1:
			found = append(found, fmt.Sprintf("%s scores it like the stego files it was calibrated on (×%.1f)", s.Name, s.LR))
		case s.Decisive():
			clear = append(clear, fmt.Sprintf("%s scores it like clean audio (×%.1f)", s.Name, 1/s.LR))
		case s.LR > 0:
			neutral = append(neutral, s.Name)
		}
	}
	if len(neutral) > 0 {
		clear = append(clear, strings.Join(neutral, ", ")+" cannot tell it from clean or stego audio")
	}
	if len(quiet) > 0 {
		clear = append(clear, "no data hidden outside the audio: "+strings.Join(quiet, ", "))
	}
	for _, u := range unusual {
		clear = append(clear, "unusual but common in honest files: "+u)
	}
	return found, clear, notRun
}

func otherVerdict(l mist.Suspicion) string {
	switch l {
	case mist.SuspicionHigh:
		return "data no player reads, where encoders write none"
	case mist.SuspicionMedium:
		return "rare in honest files; worth a closer look"
	case mist.SuspicionLow:
		return "unusual, but common in honest files"
	}
	return "nothing found by the structure and format checks"
}

func verdict(a mist.Analysis) string {
	switch {
	case a.Basis == mist.BasisFormat:
		return "Mist cannot write this format"
	case a.Probability >= 0.6:
		return "a message is likely"
	case a.Probability <= 0.4:
		return "a message is unlikely"
	default:
		return "no evidence either way"
	}
}

func basisNote(a mist.Analysis) string {
	switch a.Basis {
	case mist.BasisFormat:
		return "Mist message decided by format: Mist writes only Ogg Vorbis and lossless audio. " + otherNote
	case mist.BasisKnownCover:
		return "Mist message decided by the reference: a diff against a clean encode of the original. " + otherNote
	}
	if a.Calibration == "" {
		return "No calibration matches this format, so no Mist detector counts as evidence; 50% means no evidence either way. " + otherNote
	}
	return "Mist detectors read against " + a.Calibration + ". " +
		"50% means no evidence either way. Mist is built to keep these detectors at chance, so a " +
		"clean file and a stego file usually both land near it. Without the original, that is the honest answer. " + otherNote
}

// otherNote says what the other-steganography level can and cannot see.
const otherNote = "Other steganography is checked in the file's structure, its MP3 frames and its digital " +
	"silence; a tool that changes audio samples the way a good embedder does leaves nothing these checks can see."

func probabilityStyle(p float64) lipgloss.Style {
	switch {
	case p >= 0.6:
		return styles.alarm
	case p <= 0.4:
		return styles.good
	default:
		return styles.warn
	}
}

func wrapNote(s string, width int) string {
	r := []rune(s)
	if width < 20 || len(r) <= width {
		return s
	}
	return string(r[:width-1]) + "…"
}

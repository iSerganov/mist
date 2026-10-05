package quality

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
)

// Tool is an external perceptual-quality scorer run as a subprocess. Both
// files are resampled to 48 kHz 16-bit WAV with the ffmpeg CLI first,
// which is the rate ViSQOL's audio mode and PEAQ are defined at.
type Tool struct {
	Name   string
	bin    string
	ffmpeg string
	args   func(ref, deg string) []string
	score  *regexp.Regexp
}

// ViSQOL scores with Google's ViSQOL in audio mode; the score is MOS-LQO,
// from 1 (bad) to 5 (transparent).
func ViSQOL() Tool {
	return Tool{
		Name:   "visqol",
		bin:    configuredBinary("MIST_VISQOL", "visqol"),
		ffmpeg: configuredBinary("MIST_FFMPEG", "ffmpeg"),
		args: func(ref, deg string) []string {
			return []string{"--reference_file", ref, "--degraded_file", deg}
		},
		score: regexp.MustCompile(`MOS-LQO:\s*(-?[0-9.]+)`),
	}
}

// PEAQ scores with GstPEAQ's basic model; the score is the Objective
// Difference Grade, from -4 (very annoying) to 0 (imperceptible).
func PEAQ() Tool {
	return Tool{
		Name:   "peaq",
		bin:    configuredBinary("MIST_PEAQ", "peaq"),
		ffmpeg: configuredBinary("MIST_FFMPEG", "ffmpeg"),
		args: func(ref, deg string) []string {
			return []string{ref, deg}
		},
		score: regexp.MustCompile(`Objective Difference Grade:\s*(-?[0-9.]+)`),
	}
}

func configuredBinary(env, fallback string) string {
	if path := os.Getenv(env); path != "" {
		return path
	}
	return fallback
}

// Available reports whether the configured tool and ffmpeg are executable.
func (t Tool) Available() bool {
	_, errTool := exec.LookPath(t.bin)
	_, errFFmpeg := exec.LookPath(t.ffmpeg)
	return errTool == nil && errFFmpeg == nil
}

// Score rates deg against ref, both paths to any audio file ffmpeg reads.
// It returns ErrUnavailable when the tool or ffmpeg is missing.
func (t Tool) Score(ctx context.Context, ref, deg string) (float64, error) {
	if !t.Available() {
		return 0, ErrUnavailable
	}
	dir, err := os.MkdirTemp("", "mist-quality-*")
	if err != nil {
		return 0, err
	}
	defer func() { _ = os.RemoveAll(dir) }()
	refWav, degWav := filepath.Join(dir, "ref.wav"), filepath.Join(dir, "deg.wav")
	if err := resample(ctx, t.ffmpeg, ref, refWav); err != nil {
		return 0, err
	}
	if err := resample(ctx, t.ffmpeg, deg, degWav); err != nil {
		return 0, err
	}
	out, err := exec.CommandContext(ctx, t.bin, t.args(refWav, degWav)...).CombinedOutput()
	if err != nil {
		return 0, fmt.Errorf("quality: %s: %w: %s", t.Name, err, out)
	}
	m := t.score.FindSubmatch(out)
	if m == nil {
		return 0, fmt.Errorf("%w: %s", ErrNoScore, t.Name)
	}
	return strconv.ParseFloat(string(m[1]), 64)
}

func resample(ctx context.Context, ffmpeg, src, dst string) error {
	out, err := exec.CommandContext(ctx, ffmpeg, "-nostdin", "-v", "error", "-y",
		"-i", src, "-ar", "48000", "-c:a", "pcm_s16le", dst).CombinedOutput()
	if err != nil {
		return fmt.Errorf("quality: ffmpeg: %w: %s", err, out)
	}
	return nil
}

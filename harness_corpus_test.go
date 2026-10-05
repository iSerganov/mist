//go:build harness

package mist

import (
	"bytes"
	"fmt"
	"io/fs"
	"math"
	"math/rand/v2"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
)

const harnessSeconds = 16

type harnessCarrier struct {
	name     string
	category string
	load     func() ([]byte, error)
}

// rootCategory names the carriers that sit directly in the corpus directory.
const rootCategory = "(top level)"

type signal func(t float64, rng *rand.Rand) float64

func loadCarriers(dir string, maxSeconds int) ([]harnessCarrier, error) {
	if dir == "" {
		return syntheticCarriers(), nil
	}
	var out []harnessCarrier
	err := filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() || strings.HasPrefix(d.Name(), ".") {
			return err
		}
		rel, err := filepath.Rel(dir, path)
		if err != nil {
			return err
		}
		category := rootCategory
		if dir, _, nested := strings.Cut(filepath.ToSlash(rel), "/"); nested {
			category = dir
		}
		out = append(out, harnessCarrier{name: rel, category: category, load: func() ([]byte, error) { return readCarrier(path, maxSeconds) }})
		return nil
	})
	return out, err
}

// readCarrier reads path whole, or its first maxSeconds seconds when that is
// set, cut by ffmpeg without re-encoding so the carrier keeps its codec. It
// keeps hour-long tracks from filling memory, since every carrier is held
// decoded several times over.
func readCarrier(path string, maxSeconds int) ([]byte, error) {
	if maxSeconds <= 0 {
		return os.ReadFile(path)
	}
	dir, err := os.MkdirTemp("", "mist-cut-*")
	if err != nil {
		return nil, err
	}
	defer func() { _ = os.RemoveAll(dir) }()
	out := filepath.Join(dir, "cut"+filepath.Ext(path))
	args := []string{"-nostdin", "-loglevel", "error", "-i", path, "-map", "0:a:0", "-t", strconv.Itoa(maxSeconds), "-c", "copy", out}
	if msg, err := exec.Command("ffmpeg", args...).CombinedOutput(); err != nil {
		return nil, fmt.Errorf("ffmpeg cut: %v: %s", err, bytes.TrimSpace(msg))
	}
	return os.ReadFile(out)
}

func syntheticCarriers() []harnessCarrier {
	noise := wav(44100, 2, 44100*harnessSeconds)
	return []harnessCarrier{
		{name: "white-noise-44k-stereo.wav", load: func() ([]byte, error) { return noise, nil }},
		synth("shaped-noise-48k-stereo.wav", 48000, 2, 16, shapedNoise),
		synth("shaped-noise-48k-stereo-24bit.wav", 48000, 2, 24, shapedNoise),
		synth("tones-noise-44k-mono.wav", 44100, 1, 16, tonesNoise),
		synth("chirp-noise-48k-stereo.wav", 48000, 2, 16, chirpNoise),
		synth("bursts-silence-44k-stereo.wav", 44100, 2, 16, burstsSilence),
	}
}

func synth(name string, rate, ch, bits int, newSignal func() signal) harnessCarrier {
	n := rate * harnessSeconds
	width := bits / 8
	full := float64(int64(1)<<(bits-1) - 1)
	pcm := make([]byte, n*ch*width)
	rng := rand.New(rand.NewPCG(uint64(rate), uint64(ch)))
	for c := range ch {
		sig := newSignal()
		for i := range n {
			v := math.Max(-1, math.Min(1, sig(float64(i)/float64(rate), rng)))
			q := int32(math.Round(v * full))
			off := (i*ch + c) * width
			for k := range width {
				pcm[off+k] = byte(q >> (8 * k))
			}
		}
	}
	data := wavBytes(rate, ch, bits, pcm)
	return harnessCarrier{name: name, load: func() ([]byte, error) { return data, nil }}
}

func shapedNoise() signal {
	var low float64
	return func(_ float64, rng *rand.Rand) float64 {
		low = 0.95*low + 0.05*rng.NormFloat64()
		return 1.5*low + 0.05*rng.NormFloat64()
	}
}

func tonesNoise() signal {
	return func(t float64, rng *rand.Rand) float64 {
		return 0.2*math.Sin(2*math.Pi*220*t) + 0.15*math.Sin(2*math.Pi*1375*t) +
			0.1*math.Sin(2*math.Pi*7040*t) + 0.03*rng.NormFloat64()
	}
}

func chirpNoise() signal {
	const from, to = 100.0, 15000.0
	return func(t float64, rng *rand.Rand) float64 {
		phase := 2 * math.Pi * (from*t + (to-from)/(2*harnessSeconds)*t*t)
		return 0.3*math.Sin(phase) + 0.03*rng.NormFloat64()
	}
}

func burstsSilence() signal {
	return func(t float64, rng *rand.Rand) float64 {
		if int(t)%4 >= 2 {
			return 0
		}
		return 0.25 * rng.NormFloat64()
	}
}

//go:build harness

package mist

import (
	"bytes"
	"encoding/json"
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
	lineage  string
	license  string
	ext      string
	load     func() ([]byte, error)
}

type corpusDescription struct {
	Name    string
	License string
}

type corpusFile struct {
	Name     string              `json:"name"`
	License  string              `json:"license,omitempty"`
	Carriers []corpusFileCarrier `json:"carriers"`
}

type corpusFileCarrier struct {
	Path     string `json:"path"`
	ID       string `json:"id"`
	Category string `json:"category"`
	Lineage  string `json:"lineage"`
	License  string `json:"license,omitempty"`
}

type signal func(t float64, rng *rand.Rand) float64

func loadCarriers(dir, manifestPath string, maxSeconds int) ([]harnessCarrier, corpusDescription, error) {
	if dir == "" {
		if manifestPath != "" {
			return nil, corpusDescription{}, fmt.Errorf("corpus manifest needs a corpus directory")
		}
		return syntheticCarriers(), corpusDescription{Name: "synthetic", License: "generated"}, nil
	}
	if manifestPath != "" {
		return loadManifestCarriers(dir, manifestPath, maxSeconds)
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
		id := fmt.Sprintf("carrier-%04d", len(out)+1)
		source := path
		out = append(out, harnessCarrier{
			name: id, category: "external", lineage: id, license: "unspecified",
			ext: filepath.Ext(rel), load: carrierLoader(id, source, maxSeconds),
		})
		return nil
	})
	return out, corpusDescription{Name: "external", License: "unspecified"}, err
}

func loadManifestCarriers(dir, manifestPath string, maxSeconds int) ([]harnessCarrier, corpusDescription, error) {
	raw, err := os.ReadFile(manifestPath)
	if err != nil {
		return nil, corpusDescription{}, fmt.Errorf("read corpus manifest: %w", err)
	}
	var manifest corpusFile
	if err := json.Unmarshal(raw, &manifest); err != nil {
		return nil, corpusDescription{}, fmt.Errorf("parse corpus manifest: %w", err)
	}
	if manifest.Name == "" {
		return nil, corpusDescription{}, fmt.Errorf("corpus manifest: name is required")
	}
	seen := map[string]bool{}
	out := make([]harnessCarrier, 0, len(manifest.Carriers))
	for i, entry := range manifest.Carriers {
		clean := filepath.Clean(entry.Path)
		if entry.Path == "" || filepath.IsAbs(entry.Path) || clean == ".." ||
			strings.HasPrefix(clean, ".."+string(filepath.Separator)) {
			return nil, corpusDescription{}, fmt.Errorf("corpus manifest: carrier %d path must stay below the corpus directory", i)
		}
		if entry.ID == "" || strings.ContainsAny(entry.ID, `/\`) {
			return nil, corpusDescription{}, fmt.Errorf("corpus manifest: carrier %d id must be a path-free public identifier", i)
		}
		if seen[entry.ID] {
			return nil, corpusDescription{}, fmt.Errorf("corpus manifest: duplicate id %q", entry.ID)
		}
		seen[entry.ID] = true
		if entry.Category == "" || entry.Lineage == "" {
			return nil, corpusDescription{}, fmt.Errorf("corpus manifest: carrier %q needs category and lineage", entry.ID)
		}
		source := filepath.Join(dir, clean)
		license := entry.License
		if license == "" {
			license = manifest.License
		}
		out = append(out, harnessCarrier{
			name: entry.ID, category: entry.Category, lineage: entry.Lineage,
			license: license, ext: filepath.Ext(clean),
			load: carrierLoader(entry.ID, source, maxSeconds),
		})
	}
	return out, corpusDescription{Name: manifest.Name, License: manifest.License}, nil
}

func carrierLoader(id, source string, maxSeconds int) func() ([]byte, error) {
	return func() ([]byte, error) {
		data, err := readCarrier(source, maxSeconds)
		if err == nil {
			return data, nil
		}
		message := redactLocalPaths(err.Error(), source)
		return nil, fmt.Errorf("carrier %q: %s", id, message)
	}
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
	if msg, err := exec.Command(harnessBinary("MIST_FFMPEG", "ffmpeg"), args...).CombinedOutput(); err != nil {
		return nil, fmt.Errorf("ffmpeg cut: %v: %s", err, bytes.TrimSpace(msg))
	}
	return os.ReadFile(out)
}

func syntheticCarriers() []harnessCarrier {
	noise := wav(44100, 2, 44100*harnessSeconds)
	return []harnessCarrier{
		{name: "white-noise-44k-stereo", category: "synthetic", lineage: "white-noise-44k-stereo", license: "generated", ext: ".wav", load: func() ([]byte, error) { return noise, nil }},
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
	id := strings.TrimSuffix(name, filepath.Ext(name))
	return harnessCarrier{
		name: id, category: "synthetic", lineage: id, license: "generated", ext: filepath.Ext(name),
		load: func() ([]byte, error) { return data, nil },
	}
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

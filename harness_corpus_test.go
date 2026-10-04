//go:build harness

package mist

import (
	"io/fs"
	"math"
	"math/rand/v2"
	"os"
	"path/filepath"
	"strings"
)

const harnessSeconds = 16

type harnessCarrier struct {
	name string
	load func() ([]byte, error)
}

type signal func(t float64, rng *rand.Rand) float64

func loadCarriers(dir string) ([]harnessCarrier, error) {
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
		out = append(out, harnessCarrier{name: rel, load: func() ([]byte, error) { return os.ReadFile(path) }})
		return nil
	})
	return out, err
}

func syntheticCarriers() []harnessCarrier {
	noise := wav(44100, 2, 44100*harnessSeconds)
	return []harnessCarrier{
		{name: "white-noise-44k-stereo.wav", load: func() ([]byte, error) { return noise, nil }},
		synth("shaped-noise-48k-stereo.wav", 48000, 2, shapedNoise),
		synth("tones-noise-44k-mono.wav", 44100, 1, tonesNoise),
		synth("chirp-noise-48k-stereo.wav", 48000, 2, chirpNoise),
		synth("bursts-silence-44k-stereo.wav", 44100, 2, burstsSilence),
	}
}

func synth(name string, rate, ch int, newSignal func() signal) harnessCarrier {
	n := rate * harnessSeconds
	samples := make([]int16, n*ch)
	rng := rand.New(rand.NewPCG(uint64(rate), uint64(ch)))
	for c := range ch {
		sig := newSignal()
		for i := range n {
			v := math.Max(-1, math.Min(1, sig(float64(i)/float64(rate), rng)))
			samples[i*ch+c] = int16(math.Round(v * math.MaxInt16))
		}
	}
	data := wavFile(rate, ch, samples)
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

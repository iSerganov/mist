//go:build harness

package mist

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
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

const (
	splitDevelopment = "development"
	splitSealed      = "sealed"
)

type harnessCarrier struct {
	name        string
	category    string
	lineage     string
	license     string
	recipe      string
	sourceCodec string
	bitDepth    int
	sampleRate  int
	channels    int
	holdReason  string
	ext         string
	load        func() ([]byte, error)
	digest      func() (string, int, error)
}

type corpusDescription struct {
	Name    string
	License string
	HeldOut []harnessCarrier
}

type corpusFile struct {
	Name     string              `json:"name"`
	License  string              `json:"license,omitempty"`
	Carriers []corpusFileCarrier `json:"carriers"`
}

type corpusFileCarrier struct {
	Path        string `json:"path"`
	ID          string `json:"id"`
	Category    string `json:"category"`
	Lineage     string `json:"lineage"`
	License     string `json:"license,omitempty"`
	Split       string `json:"split,omitempty"`
	Recipe      string `json:"recipe,omitempty"`
	SourceCodec string `json:"source_codec,omitempty"`
	BitDepth    int    `json:"bit_depth,omitempty"`
	SampleRate  int    `json:"sample_rate,omitempty"`
	Channels    int    `json:"channels,omitempty"`
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
	if len(manifest.Carriers) == 0 {
		return nil, corpusDescription{}, fmt.Errorf("corpus manifest: no carriers")
	}
	unseal := os.Getenv("MIST_HARNESS_UNSEAL") == "1"
	seen := map[string]bool{}
	lineageSplit := map[string]string{}
	var scored, held []harnessCarrier
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
		split, err := normalizeSplit(entry.Split)
		if err != nil {
			return nil, corpusDescription{}, err
		}
		if prev, ok := lineageSplit[entry.Lineage]; ok && prev != split {
			return nil, corpusDescription{}, fmt.Errorf("corpus manifest: lineage %q is in both development and sealed", entry.Lineage)
		}
		lineageSplit[entry.Lineage] = split
		source := filepath.Join(dir, clean)
		license := entry.License
		if license == "" {
			license = manifest.License
		}
		carrier := harnessCarrier{
			name: entry.ID, category: entry.Category, lineage: entry.Lineage,
			license: license, recipe: entry.Recipe, sourceCodec: entry.SourceCodec,
			bitDepth: entry.BitDepth, sampleRate: entry.SampleRate, channels: entry.Channels,
			ext: filepath.Ext(clean), load: carrierLoader(entry.ID, source, maxSeconds),
			digest: fileDigest(entry.ID, source),
		}
		score := split == splitDevelopment
		if unseal {
			score = split == splitSealed
		}
		if score {
			scored = append(scored, carrier)
			continue
		}
		if split == splitSealed {
			carrier.holdReason = "sealed"
		} else {
			carrier.holdReason = "development-while-unsealed"
		}
		held = append(held, carrier)
	}
	if len(scored) == 0 {
		if unseal {
			return nil, corpusDescription{}, fmt.Errorf("corpus manifest: no sealed carriers to score")
		}
		return nil, corpusDescription{}, fmt.Errorf("corpus manifest: no development carriers; set MIST_HARNESS_UNSEAL=1 to score the sealed split")
	}
	return scored, corpusDescription{Name: manifest.Name, License: manifest.License, HeldOut: held}, nil
}

func normalizeSplit(split string) (string, error) {
	switch split {
	case "", splitDevelopment:
		return splitDevelopment, nil
	case splitSealed:
		return splitSealed, nil
	default:
		return "", fmt.Errorf("corpus manifest: split %q must be development or sealed", split)
	}
}

func fileDigest(id, source string) func() (string, int, error) {
	return func() (string, int, error) {
		f, err := os.Open(source)
		if err != nil {
			return "", 0, fmt.Errorf("carrier %q: %s", id, redactLocalPaths(err.Error(), source))
		}
		defer func() { _ = f.Close() }()
		h := sha256.New()
		n, err := io.Copy(h, f)
		if err != nil {
			return "", 0, fmt.Errorf("carrier %q: %s", id, redactLocalPaths(err.Error(), source))
		}
		return hex.EncodeToString(h.Sum(nil)), int(n), nil
	}
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

// conditionCatalog is the control list Phase 2 freezes. A harness run executes
// only executedConditions; the rest stay in the manifest until a later run
// asks for them, so this phase does not rescore the development baseline.
func conditionCatalog() []string {
	return []string{
		"ffmpeg-canonical",
		"mist-clean",
		"payload-minimal",
		"payload-normal",
		"payload-25",
		"payload-50",
		"payload-75",
		"payload-near-max",
		"span-below",
		"span-above",
		"signed",
		"unsigned",
		"fixed-key",
		"varied-key",
		"repeated-embeddings",
		"filler-only",
	}
}

func executedConditions() []string {
	return []string{"ffmpeg-canonical", "mist-clean", "payload-normal", "payload-minimal"}
}

type corpusPower struct {
	TargetD             float64 `json:"target_d"`
	LineagesRequired    int     `json:"lineages_required"`
	DevelopmentLineages int     `json:"development_lineages"`
	RequirementMet      bool    `json:"requirement_met"`
	Note                string  `json:"note"`
}

type externalSource struct {
	ID      string `json:"id"`
	Content string `json:"content"`
	License string `json:"license"`
	Note    string `json:"note"`
}

type corpusDocument struct {
	Name              string              `json:"name"`
	License           string              `json:"license"`
	Carriers          []corpusFileCarrier `json:"carriers"`
	Conditions        []string            `json:"conditions"`
	Power             corpusPower         `json:"power"`
	ExternalSources   []externalSource    `json:"external_sources"`
	TranscodesSkipped []string            `json:"transcodes_skipped,omitempty"`
}

type generatedCell struct {
	id, category, lineage, split  string
	rate, channels, bits, seconds int
	seed                          uint64
	signal                        func() signal
}

// generateCorpus writes a PCM development corpus and a source-disjoint sealed
// holdout. Audio stays in dir. dir must not sit inside a git checkout.
func generateCorpus(dir string) error {
	if dir == "" {
		return fmt.Errorf("corpus output directory is required")
	}
	if repo, err := insideGitRepo(dir); err != nil {
		return err
	} else if repo {
		return fmt.Errorf("corpus output must be outside a git checkout")
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	var carriers []corpusFileCarrier
	for _, cell := range phase2Cells() {
		rel := filepath.Join(cell.split, cell.id+".wav")
		path := filepath.Join(dir, rel)
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			return err
		}
		data := generateWAV(cell.rate, cell.channels, cell.bits, cell.seconds, cell.seed, cell.signal)
		if err := os.WriteFile(path, data, 0o644); err != nil {
			return err
		}
		carriers = append(carriers, corpusFileCarrier{
			Path: rel, ID: cell.id, Category: cell.category, Lineage: cell.lineage,
			License: "generated", Split: cell.split, Recipe: "generated-pcm-v1",
			SourceCodec: "pcm_s" + strconv.Itoa(cell.bits) + "le",
			BitDepth:    cell.bits, SampleRate: cell.rate, Channels: cell.channels,
		})
	}
	skipped := transcodeProvenance(dir, &carriers)
	lineages := map[string]bool{}
	for _, carrier := range carriers {
		if carrier.Split != splitSealed {
			lineages[carrier.Lineage] = true
		}
	}
	doc := corpusDocument{
		Name: "phase-2-generated", License: "generated", Carriers: carriers,
		Conditions: conditionCatalog(),
		Power: corpusPower{
			TargetD: harnessPowerTarget, LineagesRequired: 128, DevelopmentLineages: len(lineages),
			RequirementMet: false,
			Note:           "128 is the Phase 1 FLAC simulation for D = 0.55. Generated cells cover content and provenance classes; they are not that many independent real recordings.",
		},
		ExternalSources:   externalSourceNotes(),
		TranscodesSkipped: skipped,
	}
	raw, err := json.MarshalIndent(doc, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(dir, "corpus.json"), append(raw, '\n'), 0o644)
}

func phase2Cells() []generatedCell {
	frame := int(FrameDuration.Seconds())
	return []generatedCell{
		{"speech-shaped-16k-mono", "speech-shaped", "speech-shaped-16k", splitDevelopment, 16000, 1, 16, frame, 1, amNoise},
		{"sparse-tones-48k-24", "sparse", "sparse-48k", splitDevelopment, 48000, 2, 24, frame, 2, sparseTones},
		{"tonal-bed-44k", "tonal", "tonal-44k", splitDevelopment, 44100, 2, 16, frame, 3, tonesNoise},
		{"tonal-bed-44k-long", "tonal", "tonal-44k", splitDevelopment, 44100, 2, 16, 3 * frame, 3, tonesNoise},
		{"environment-48k", "environment", "environment-48k", splitDevelopment, 48000, 2, 16, frame, 4, shapedNoise},
		{"transients-44k", "transients", "transients-44k", splitDevelopment, 44100, 2, 16, frame, 5, transients},
		{"silence-heavy-44k", "silence", "silence-44k", splitDevelopment, 44100, 2, 16, frame, 6, burstsSilence},
		{"low-dynamic-44k", "low-dynamic", "low-dynamic-44k", splitDevelopment, 44100, 2, 16, frame, 7, quietNoise},
		{"multichannel-48k", "tonal", "multichannel-48k", splitDevelopment, 48000, 6, 16, frame, 8, tonesNoise},
		{"sealed-environment-48k", "environment", "sealed-environment-48k", splitSealed, 48000, 2, 16, frame, 101, shapedNoise},
		{"sealed-sparse-48k-24", "sparse", "sealed-sparse-48k", splitSealed, 48000, 2, 24, frame, 102, sparseTones},
		{"sealed-speech-16k-mono", "speech-shaped", "sealed-speech-16k", splitSealed, 16000, 1, 16, frame, 103, amNoise},
	}
}

func externalSourceNotes() []externalSource {
	return []externalSource{
		{"librispeech", "speech", "CC BY 4.0", "Acquire outside the repo. Record the license and a hash per file. Do not commit audio."},
		{"fsd50k", "environmental", "per-clip FSD50K terms", "Licenses differ by clip. Record the one that applies."},
		{"musan", "music-speech-noise", "CC BY 4.0", "Acquire outside the repo. Record the license and a hash per file."},
		{"maestro", "classical-piano", "CC BY-NC-SA 4.0", "Non-commercial. Do not treat it as a license for a product corpus."},
		{"audioset", "environmental", "YouTube terms", "Clips are not redistributable. Freeze the id list; availability changes."},
	}
}

func generateWAV(rate, ch, bits, seconds int, seed uint64, newSignal func() signal) []byte {
	n := rate * seconds
	width := bits / 8
	full := float64(int64(1)<<(bits-1) - 1)
	pcm := make([]byte, n*ch*width)
	rng := rand.New(rand.NewPCG(seed, uint64(rate)))
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
	return wavBytes(rate, ch, bits, pcm)
}

func transcodeProvenance(dir string, carriers *[]corpusFileCarrier) []string {
	ffmpeg, err := exec.LookPath(harnessBinary("MIST_FFMPEG", "ffmpeg"))
	if err != nil {
		return []string{"flac", "mp3"}
	}
	src := filepath.Join(dir, splitDevelopment, "tonal-bed-44k.wav")
	var skipped []string
	for _, spec := range []struct {
		codec, ext string
		args       []string
	}{
		{"flac", ".flac", []string{"-c:a", "flac"}},
		{"mp3", ".mp3", []string{"-c:a", "libmp3lame", "-q:a", "4"}},
	} {
		rel := filepath.Join(splitDevelopment, "tonal-bed-44k"+spec.ext)
		dst := filepath.Join(dir, rel)
		args := append([]string{"-nostdin", "-loglevel", "error", "-y", "-i", src}, spec.args...)
		args = append(args, dst)
		if msg, err := exec.Command(ffmpeg, args...).CombinedOutput(); err != nil {
			skipped = append(skipped, spec.codec)
			_ = msg
			continue
		}
		*carriers = append(*carriers, corpusFileCarrier{
			Path: rel, ID: "tonal-bed-44k-" + spec.codec, Category: "tonal", Lineage: "tonal-44k",
			License: "generated", Split: splitDevelopment, Recipe: "generated-pcm-v1-transcode-" + spec.codec,
			SourceCodec: spec.codec,
		})
	}
	return skipped
}

func insideGitRepo(dir string) (bool, error) {
	abs, err := filepath.Abs(dir)
	if err != nil {
		return false, err
	}
	for d := abs; ; d = filepath.Dir(d) {
		if _, err := os.Stat(filepath.Join(d, ".git")); err == nil {
			return true, nil
		}
		if parent := filepath.Dir(d); parent == d {
			return false, nil
		}
	}
}

func amNoise() signal {
	return func(t float64, rng *rand.Rand) float64 {
		env := 0.5 + 0.5*math.Sin(2*math.Pi*3*t)
		return 0.2 * env * rng.NormFloat64()
	}
}

func sparseTones() signal {
	return func(t float64, rng *rand.Rand) float64 {
		if math.Mod(t, 2) > 0.4 {
			return 0.01 * rng.NormFloat64()
		}
		return 0.4*math.Sin(2*math.Pi*440*t) + 0.01*rng.NormFloat64()
	}
}

func transients() signal {
	return func(t float64, rng *rand.Rand) float64 {
		m := math.Mod(t, 0.5)
		if m < 0.01 {
			return 0.8 * (1 - m/0.01) * math.Sin(2*math.Pi*2000*t)
		}
		return 0.01 * rng.NormFloat64()
	}
}

func quietNoise() signal {
	return func(_ float64, rng *rand.Rand) float64 {
		return 0.02 * rng.NormFloat64()
	}
}

func (s *HarnessSuite) TestGeneratedCorpusKeepsTheSeal() {
	s.T().Setenv("MIST_HARNESS_UNSEAL", "")
	dir := s.T().TempDir()
	s.Require().NoError(generateCorpus(dir))
	manifest := filepath.Join(dir, "corpus.json")

	dev, desc, err := loadCarriers(dir, manifest, 0)
	s.Require().NoError(err)
	s.NotEmpty(dev)
	s.NotEmpty(desc.HeldOut)
	devLineage := map[string]bool{}
	for _, carrier := range dev {
		s.NotContains(carrier.name, "/")
		devLineage[carrier.lineage] = true
	}
	for _, carrier := range desc.HeldOut {
		s.False(devLineage[carrier.lineage], carrier.lineage)
		s.Equal("sealed", carrier.holdReason)
	}
	s.Contains(idsOf(dev), "tonal-bed-44k")
	s.Contains(idsOf(dev), "tonal-bed-44k-long")
	long := findCarrier(dev, "tonal-bed-44k-long")
	short := findCarrier(dev, "tonal-bed-44k")
	s.Equal(short.lineage, long.lineage)

	s.T().Setenv("MIST_HARNESS_UNSEAL", "1")
	sealed, unsealed, err := loadCarriers(dir, manifest, 0)
	s.Require().NoError(err)
	s.NotContains(idsOf(sealed), "tonal-bed-44k")
	s.Contains(idsOf(sealed), "sealed-speech-16k-mono")
	s.NotEmpty(unsealed.HeldOut)
	for _, carrier := range unsealed.HeldOut {
		s.Equal("development-while-unsealed", carrier.holdReason)
	}
}

func (s *HarnessSuite) TestGeneratedCorpusRefusesACheckout() {
	err := generateCorpus(".")
	s.Require().Error(err)
	s.Contains(err.Error(), "outside a git checkout")
}

func (s *HarnessSuite) TestSharedLineageCannotCrossTheSeal() {
	s.T().Setenv("MIST_HARNESS_UNSEAL", "")
	dir := s.T().TempDir()
	s.Require().NoError(os.WriteFile(filepath.Join(dir, "a.wav"), wav(8000, 1, 800), 0o600))
	s.Require().NoError(os.WriteFile(filepath.Join(dir, "b.wav"), wav(8000, 1, 800), 0o600))
	s.writeCorpus(dir, corpusFile{Name: "mixed", Carriers: []corpusFileCarrier{
		{Path: "a.wav", ID: "a", Category: "tonal", Lineage: "same", Split: splitDevelopment},
		{Path: "b.wav", ID: "b", Category: "tonal", Lineage: "same", Split: splitSealed},
	}})
	_, _, err := loadCarriers(dir, filepath.Join(dir, "corpus.json"), 0)
	s.Require().Error(err)
	s.Contains(err.Error(), "both development and sealed")
}

func (s *HarnessSuite) TestHeldOutCarriersAreHashedNotDecoded() {
	s.T().Setenv("MIST_HARNESS_UNSEAL", "")
	dir := s.T().TempDir()
	body := wav(8000, 1, 800)
	s.Require().NoError(os.WriteFile(filepath.Join(dir, "a.wav"), body, 0o600))
	s.Require().NoError(os.WriteFile(filepath.Join(dir, "b.wav"), wav(8000, 1, 800), 0o600))
	s.writeCorpus(dir, corpusFile{Name: "hold", Carriers: []corpusFileCarrier{
		{Path: "a.wav", ID: "dev-a", Category: "tonal", Lineage: "dev", Split: splitDevelopment},
		{Path: "b.wav", ID: "sealed-b", Category: "environment", Lineage: "sealed", Split: splitSealed, Recipe: "external-file"},
	}})
	scored, desc, err := loadCarriers(dir, filepath.Join(dir, "corpus.json"), 0)
	s.Require().NoError(err)
	s.Equal([]string{"dev-a"}, idsOf(scored))
	manifest, err := describeCorpus(nil, desc, 0)
	s.Require().NoError(err)
	s.Require().Len(manifest.HeldOut, 1)
	s.Equal("sealed-b", manifest.HeldOut[0].ID)
	s.Equal("sealed", manifest.HeldOut[0].Reason)
	sum := sha256.Sum256(wav(8000, 1, 800))
	s.Equal(hex.EncodeToString(sum[:]), manifest.HeldOut[0].SHA256)
	s.Equal(0, manifest.IndependentGroups)
}

func (s *HarnessSuite) TestWriteGeneratedCorpus() {
	dir := os.Getenv("MIST_CORPUS_OUT")
	if dir == "" {
		s.T().Skip("MIST_CORPUS_OUT not set")
	}
	s.Require().NoError(generateCorpus(dir))
}

func (s *HarnessSuite) writeCorpus(dir string, doc corpusFile) {
	s.T().Helper()
	raw, err := json.Marshal(doc)
	s.Require().NoError(err)
	s.Require().NoError(os.WriteFile(filepath.Join(dir, "corpus.json"), raw, 0o600))
}

func idsOf(carriers []harnessCarrier) []string {
	out := make([]string, len(carriers))
	for i, carrier := range carriers {
		out[i] = carrier.name
	}
	return out
}

func findCarrier(carriers []harnessCarrier, id string) harnessCarrier {
	for _, carrier := range carriers {
		if carrier.name == id {
			return carrier
		}
	}
	return harnessCarrier{}
}

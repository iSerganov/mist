//go:build harness

package mist

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/iSerganov/mist/internal/codec"
	"github.com/iSerganov/mist/internal/steganalysis"
	"github.com/iSerganov/mist/internal/stego"
)

const harnessReportSchema = 6

type runManifest struct {
	Schema     int                `json:"schema"`
	Created    string             `json:"created"`
	Git        gitManifest        `json:"git"`
	Build      buildManifest      `json:"build"`
	Tools      toolManifest       `json:"tools"`
	Corpus     corpusManifest     `json:"corpus"`
	Protocol   protocolManifest   `json:"protocol"`
	Experiment experimentManifest `json:"experiment"`
}

type gitManifest struct {
	Commit   string `json:"commit"`
	Describe string `json:"describe"`
	Dirty    bool   `json:"dirty"`
}

type buildManifest struct {
	GoVersion    string `json:"go_version"`
	Compiler     string `json:"compiler"`
	OS           string `json:"os"`
	Architecture string `json:"architecture"`
	LogicalCPUs  int    `json:"logical_cpus"`
}

type toolManifest struct {
	FFmpeg        string `json:"ffmpeg"`
	Libavformat   string `json:"libavformat"`
	Libavcodec    string `json:"libavcodec"`
	Libavutil     string `json:"libavutil"`
	Libswresample string `json:"libswresample"`
	Libvorbis     string `json:"libvorbis"`
	Perceptual    string `json:"perceptual,omitempty"`
}

type corpusManifest struct {
	Name              string            `json:"name"`
	License           string            `json:"license,omitempty"`
	MaxSeconds        int               `json:"max_seconds,omitempty"`
	Carriers          []carrierManifest `json:"carriers"`
	IndependentGroups int               `json:"independent_groups"`
	HeldOut           []heldManifest    `json:"held_out,omitempty"`
	HeldOutGroups     int               `json:"held_out_groups,omitempty"`
}

type carrierManifest struct {
	ID              string  `json:"id"`
	Category        string  `json:"category"`
	Lineage         string  `json:"lineage"`
	License         string  `json:"license,omitempty"`
	Recipe          string  `json:"recipe,omitempty"`
	SHA256          string  `json:"sha256"`
	Bytes           int     `json:"bytes"`
	Codec           string  `json:"codec"`
	Container       string  `json:"container"`
	SampleRate      int     `json:"sample_rate"`
	Channels        int     `json:"channels"`
	SampleFmt       string  `json:"sample_format"`
	Bits            int     `json:"bits"`
	Samples         int     `json:"samples"`
	DurationSeconds float64 `json:"duration_seconds,omitempty"`
}

// heldManifest is a carrier the run did not score. Reason is "sealed" on a
// development run, or "development-while-unsealed" when the sealed split is
// the one being scored. The hash is of the file bytes; the file is not decoded.
type heldManifest struct {
	ID          string `json:"id"`
	Category    string `json:"category"`
	Lineage     string `json:"lineage"`
	License     string `json:"license,omitempty"`
	Recipe      string `json:"recipe,omitempty"`
	Reason      string `json:"reason"`
	SHA256      string `json:"sha256"`
	Bytes       int    `json:"bytes"`
	SourceCodec string `json:"source_codec,omitempty"`
	BitDepth    int    `json:"bit_depth,omitempty"`
	SampleRate  int    `json:"sample_rate,omitempty"`
	Channels    int    `json:"channels,omitempty"`
}

type protocolManifest struct {
	FrameDuration string  `json:"frame_duration"`
	Density       float64 `json:"density"`
	BandFromHz    int     `json:"band_from_hz"`
	BandToHz      int     `json:"band_to_hz"`
	PublicKeyHex  string  `json:"recipient_public_key_hex"`
	Randomness    string  `json:"randomness"`
}

type experimentManifest struct {
	Formats            []string  `json:"formats"`
	Jobs               int       `json:"jobs"`
	ChunkValues        int       `json:"chunk_values"`
	BootstrapRounds    int       `json:"bootstrap_rounds"`
	Seed               uint64    `json:"seed"`
	Folds              int       `json:"folds"`
	PayloadBytes       int       `json:"payload_bytes"`
	MinimalBytes       int       `json:"minimal_payload_bytes"`
	CleanControls      []string  `json:"clean_controls"`
	ScoreUnit          string    `json:"score_unit"`
	Classifier         string    `json:"classifier"`
	Features           []string  `json:"features"`
	PermutationRounds  int       `json:"permutation_rounds"`
	RefitRounds        int       `json:"refit_rounds"`
	NestedPenalties    []float64 `json:"nested_penalties"`
	PowerTarget        float64   `json:"power_target_d"`
	Power              float64   `json:"power"`
	PlannedConditions  []string  `json:"planned_conditions"`
	ExecutedConditions []string  `json:"executed_conditions"`
	CanonicalWorkflow  string    `json:"canonical_workflow"`
	MetadataThreshold  string    `json:"metadata_threshold"`
	MetadataFeatures   []string  `json:"metadata_features"`
	FrozenWardens      []string  `json:"frozen_wardens"`
	Oracle             string    `json:"oracle"`
}

func harnessBinary(env, fallback string) string {
	if path := os.Getenv(env); path != "" {
		return path
	}
	return fallback
}

func makeRunManifest(
	carriers []harnessCarrier,
	corpus corpusDescription,
	maxSeconds int,
	pub []byte,
	formats []string,
	jobs int,
	perceptual string,
) (runManifest, error) {
	cm, err := describeCorpus(carriers, corpus, maxSeconds)
	if err != nil {
		return runManifest{}, err
	}
	return runManifest{
		Schema:  harnessReportSchema,
		Created: time.Now().UTC().Format(time.RFC3339),
		Git:     inspectGit(),
		Build: buildManifest{
			GoVersion: runtime.Version(), Compiler: runtime.Compiler,
			OS: runtime.GOOS, Architecture: runtime.GOARCH, LogicalCPUs: runtime.NumCPU(),
		},
		Tools:  inspectTools(perceptual),
		Corpus: cm,
		Protocol: protocolManifest{
			FrameDuration: FrameDuration.String(),
			Density:       stego.Density,
			BandFromHz:    stego.DefaultBands.FromHz,
			BandToHz:      stego.DefaultBands.ToHz,
			PublicKeyHex:  hex.EncodeToString(pub),
			Randomness:    "fresh cryptographic randomness per output; exact stego bytes are intentionally nondeterministic",
		},
		Experiment: experimentManifest{
			Formats: formats, Jobs: jobs, ChunkValues: harnessChunk,
			BootstrapRounds: harnessRounds, Seed: harnessSeed, Folds: harnessFolds,
			PayloadBytes: 64, MinimalBytes: 1,
			CleanControls: []string{
				"ffmpeg-default (separate threat model; not the fingerprint verdict)",
				"ffmpeg-canonical (vorbis -q:a at Mist's level; lossless defaults)",
				"mist-clean",
			},
			ScoreUnit:  "chunks grouped by lineage; files stay one per recording",
			Classifier: "L2-regularised logistic regression, z-scored on the training fold, nested lineage-grouped CV",
			Features: []string{
				"classical detector scores",
				"histogram share at -3..3",
				"first-order SPAM transitions",
				"second-difference Markov transitions",
				"frozen rich-model summary",
			},
			FrozenWardens:      steganalysis.FrozenPrimaries(),
			Oracle:             "changed-fraction compares cover and stego sample by sample; it is not an operational warden",
			PermutationRounds:  harnessPerms,
			RefitRounds:        harnessRefits,
			NestedPenalties:    steganalysis.NestedPenaltyGrid(),
			PowerTarget:        harnessPowerTarget,
			Power:              harnessPower,
			PlannedConditions:  conditionCatalog(),
			CanonicalWorkflow:  canonicalWorkflow,
			MetadataThreshold:  metaThreshold,
			MetadataFeatures:   append([]string(nil), metaFeatureNames...),
			ExecutedConditions: executedConditions(),
		},
	}, nil
}

func describeCorpus(carriers []harnessCarrier, corpus corpusDescription, maxSeconds int) (corpusManifest, error) {
	out := corpusManifest{Name: corpus.Name, License: corpus.License, MaxSeconds: maxSeconds}
	groups := map[string]bool{}
	for _, carrier := range carriers {
		data, err := carrier.load()
		if err != nil {
			return corpusManifest{}, fmt.Errorf("manifest carrier %q: %w", carrier.name, err)
		}
		pcm, info, _, err := decodeCarrier(bytes.NewReader(data))
		if err != nil {
			return corpusManifest{}, fmt.Errorf("manifest carrier %q: %w", carrier.name, err)
		}
		sum := sha256.Sum256(data)
		entry := carrierManifest{
			ID: carrier.name, Category: carrier.category, Lineage: carrier.lineage, License: carrier.license,
			Recipe: carrier.recipe,
			SHA256: hex.EncodeToString(sum[:]), Bytes: len(data), Codec: info.CodecName,
			Container: info.Container, SampleRate: info.SampleRate, Channels: info.Channels,
			SampleFmt: sampleFormatName(info.SampleFmt), Bits: info.Bits, Samples: pcm.NbSamples,
		}
		if info.SampleRate > 0 {
			entry.DurationSeconds = float64(pcm.NbSamples) / float64(info.SampleRate)
		}
		out.Carriers = append(out.Carriers, entry)
		groups[carrier.lineage] = true
	}
	out.IndependentGroups = len(groups)
	heldGroups := map[string]bool{}
	for _, carrier := range corpus.HeldOut {
		sum, n, err := carrier.digest()
		if err != nil {
			return corpusManifest{}, fmt.Errorf("manifest held-out carrier %q: %w", carrier.name, err)
		}
		out.HeldOut = append(out.HeldOut, heldManifest{
			ID: carrier.name, Category: carrier.category, Lineage: carrier.lineage, License: carrier.license,
			Recipe: carrier.recipe, Reason: carrier.holdReason, SHA256: sum, Bytes: n,
			SourceCodec: carrier.sourceCodec, BitDepth: carrier.bitDepth,
			SampleRate: carrier.sampleRate, Channels: carrier.channels,
		})
		heldGroups[carrier.lineage] = true
	}
	out.HeldOutGroups = len(heldGroups)
	return out, nil
}

func sampleFormatName(format codec.SampleFormat) string {
	i := int(format)
	if i >= 0 && i < len(sampleFmtNames) {
		return sampleFmtNames[i]
	}
	return "unknown"
}

func inspectGit() gitManifest {
	git := harnessBinary("MIST_GIT", "git")
	commit := commandLine(git, "rev-parse", "HEAD")
	describe := commandLine(git, "describe", "--always", "--dirty")
	status := commandOutput(git, "status", "--porcelain")
	return gitManifest{Commit: commit, Describe: describe, Dirty: strings.TrimSpace(status) != ""}
}

func inspectTools(perceptual string) toolManifest {
	pkg := harnessBinary("MIST_PKG_CONFIG", "pkg-config")
	return toolManifest{
		FFmpeg:        commandLine(harnessBinary("MIST_FFMPEG", "ffmpeg"), "-version"),
		Libavformat:   commandLine(pkg, "--modversion", "libavformat"),
		Libavcodec:    commandLine(pkg, "--modversion", "libavcodec"),
		Libavutil:     commandLine(pkg, "--modversion", "libavutil"),
		Libswresample: commandLine(pkg, "--modversion", "libswresample"),
		Libvorbis:     commandLine(pkg, "--modversion", "vorbis"),
		Perceptual:    perceptual,
	}
}

func commandLine(name string, args ...string) string {
	out := commandOutput(name, args...)
	line, _, _ := strings.Cut(strings.TrimSpace(out), "\n")
	if line == "" {
		return "unknown"
	}
	return line
}

func commandOutput(name string, args ...string) string {
	out, err := exec.Command(name, args...).CombinedOutput()
	if err != nil {
		return ""
	}
	return string(out)
}

// redactLocalPaths strips machine-local directories from strings that can
// land in a published report. Manifests and skip reasons must stay
// reproducible on another machine without leaking HOME or temp paths.
func redactLocalPaths(s string, extra ...string) string {
	roots := append(append([]string{}, extra...), os.TempDir(), os.Getenv("HOME"), os.Getenv("TMPDIR"))
	for _, root := range roots {
		if root == "" || root == "/" || root == "." {
			continue
		}
		s = strings.ReplaceAll(s, root, "<path>")
		s = strings.ReplaceAll(s, filepath.ToSlash(root), "<path>")
	}
	return s
}

func (s *HarnessSuite) TestCorpusManifestUsesPublicIdentifiers() {
	dir := s.T().TempDir()
	s.Require().NoError(os.WriteFile(filepath.Join(dir, "private-recording.flac"), []byte("carrier"), 0o600))
	spec := corpusFile{
		Name: "public-corpus", License: "CC0",
		Carriers: []corpusFileCarrier{{
			Path: "private-recording.flac", ID: "carrier-a", Category: "speech", Lineage: "session-a",
		}},
	}
	raw, err := json.Marshal(spec)
	s.Require().NoError(err)
	path := filepath.Join(s.T().TempDir(), "corpus.json")
	s.Require().NoError(os.WriteFile(path, raw, 0o600))

	carriers, description, err := loadCarriers(dir, path, 0)
	s.Require().NoError(err)
	s.Require().Len(carriers, 1)
	s.Equal("public-corpus", description.Name)
	s.Equal("carrier-a", carriers[0].name)
	s.Equal(".flac", carriers[0].ext)
	s.NotContains(carriers[0].name, "private-recording")
	data, err := carriers[0].load()
	s.Require().NoError(err)
	s.Equal([]byte("carrier"), data)
}

func (s *HarnessSuite) TestCorpusManifestRejectsPathsOutsideRoot() {
	spec := corpusFile{
		Name: "public-corpus",
		Carriers: []corpusFileCarrier{{
			Path: "../private.wav", ID: "carrier-a", Category: "speech", Lineage: "session-a",
		}},
	}
	raw, err := json.Marshal(spec)
	s.Require().NoError(err)
	path := filepath.Join(s.T().TempDir(), "corpus.json")
	s.Require().NoError(os.WriteFile(path, raw, 0o600))

	_, _, err = loadCarriers(s.T().TempDir(), path, 0)
	s.Require().Error(err)
	s.Contains(err.Error(), "stay below")
}

func (s *HarnessSuite) TestRawScoresRetainPublicLineage() {
	scores := scored{vals: []float64{0.1, 0.3, 0.7}, groups: []int{0, 0, 1}}
	identities := []scoreIdentity{
		{Carrier: "carrier-a", Category: "speech", Lineage: "session-a"},
		{Carrier: "carrier-b", Category: "music", Lineage: "session-b"},
	}

	got := rawPopulationFrom(scores, identities)
	s.Require().Len(got.Chunks, 3)
	s.Require().Len(got.Files, 2)
	s.Equal("session-a", got.Chunks[0].Lineage)
	s.Equal(0.2, got.Files[0].Score)
	s.Equal("carrier-b", got.Files[1].Carrier)
}

func (s *HarnessSuite) TestExampleCorpusManifestIsPublic() {
	carriers, description, err := loadCarriers(s.T().TempDir(), "testdata/harness/corpus.example.json", 0)
	s.Require().NoError(err)
	s.Equal("example-public-corpus", description.Name)
	s.Require().Len(carriers, 2)
	s.Equal("speech-session-a", carriers[0].name)
	s.Equal("session-a", carriers[0].lineage)
	s.NotContains(carriers[0].name, "/")
	s.NotContains(carriers[0].name, string(filepath.Separator))
}

func (s *HarnessSuite) TestWalkCarriersUseAnonymousIDs() {
	dir := s.T().TempDir()
	s.Require().NoError(os.WriteFile(filepath.Join(dir, "private-title.flac"), []byte("carrier"), 0o600))

	carriers, description, err := loadCarriers(dir, "", 0)
	s.Require().NoError(err)
	s.Equal("external", description.Name)
	s.Require().Len(carriers, 1)
	s.Equal("carrier-0001", carriers[0].name)
	s.Equal("external", carriers[0].category)
	s.NotContains(carriers[0].name, "private-title")
}

func (s *HarnessSuite) TestRedactLocalPaths() {
	home := s.T().TempDir()
	s.T().Setenv("HOME", home)
	got := redactLocalPaths(home+"/Music/secret.flac failed", home+"/Music/secret.flac")
	s.NotContains(got, home)
	s.NotContains(got, "secret.flac")
	s.Contains(got, "<path>")
}

func (s *HarnessSuite) TestReportWritesManifestAndRawScores() {
	dir := s.T().TempDir()
	s.T().Setenv("MIST_HARNESS_CNN", filepath.Join(dir, "absent-cnn.json"))
	report := harnessReport{
		Commit: "abc123", Date: "2026-10-04", Corpus: "synthetic", Carriers: 1,
		Manifests: []runManifest{{
			Schema: harnessReportSchema,
			Corpus: corpusManifest{Name: "synthetic", IndependentGroups: 1},
		}},
		Formats: []formatReport{{
			Format: "flac",
			Raw:    rawFormatScores{Operational: []rawDetectorScores{{Name: "test"}}},
		}},
	}

	_, err := report.write(dir, nil)
	s.Require().NoError(err)
	for _, name := range []string{"report.json", "report.md", "manifest.json", "scores.json"} {
		_, err := os.Stat(filepath.Join(dir, name))
		s.NoError(err, name)
	}
}

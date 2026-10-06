//go:build harness

package mist

import (
	"context"
	"encoding/binary"
	"fmt"
	"hash/fnv"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/iSerganov/mist/internal/av"
	"github.com/iSerganov/mist/internal/codec"
	"github.com/iSerganov/mist/internal/steganalysis"
	"github.com/iSerganov/mist/internal/trace"
)

// canonicalWorkflow is the innocent encode the fingerprint verdict uses.
// Vorbis follows the quality level Mist already opens (ffmpeg -q:a N).
// A lossless codec has no quality knob, so the canonical encode is ffmpeg's
// own defaults. Default ffmpeg on Vorbis is a second threat model.
const canonicalWorkflow = "vorbis: ffmpeg -q:a at Mist's source-conditioned level; lossless: ffmpeg defaults"

// metaThreshold is the preregistered pass for the audio-blind classifier.
// An interval that does not contain 0.5 is a failed objective.
const metaThreshold = "metadata classifier file interval includes 0.5"

// carrierTrace sets the four files the audit compares. FFmpeg is the
// default-ffmpeg threat model. Canonical is the chosen workflow. Clean is
// Mist with nothing embedded. Mist is the stego file.
type carrierTrace struct {
	Name           string      `json:"name"`
	Source         int         `json:"source_samples"`
	SourceRate     int         `json:"source_rate,omitempty"`
	SourceChannels int         `json:"source_channels,omitempty"`
	SourceFmt      string      `json:"source_sample_fmt,omitempty"`
	FFmpeg         trace.Trace `json:"ffmpeg_default"`
	Canonical      trace.Trace `json:"ffmpeg_canonical"`
	Clean          trace.Trace `json:"mist_clean"`
	Mist           trace.Trace `json:"mist"`
}

// metaResult is one audio-blind question. Passed is the preregistered
// threshold: the file interval contains 0.5.
type metaResult struct {
	Question      string  `json:"question"`
	FileAUC       float64 `json:"file_auc,omitempty"`
	FileLo        float64 `json:"file_lo,omitempty"`
	FileHi        float64 `json:"file_hi,omitempty"`
	Detectability float64 `json:"detectability,omitempty"`
	Scored        bool    `json:"scored"`
	Passed        bool    `json:"passed"`
	Note          string  `json:"note,omitempty"`
}

// metaFeatureNames is the audio-blind vector, in order. Hashes stand in
// for strings so a tag or layout mismatch is a number the classifier can
// use. Ogg serial values are not features: ffmpeg draws them at random.
var metaFeatureNames = []string{
	"bytes", "nominal_kbps", "kbps", "samples", "zero_tail", "bits", "channels", "sample_rate",
	"packets", "mean_packet", "ogg_pages", "ogg_mean_page", "ogg_serials", "ogg_granule", "ogg_first_granule",
	"flac_md5", "flac_samples", "flac_blocks", "padding", "unknown_meta", "skip_start", "streams",
	"codec", "container", "channel_layout", "time_base", "encoder", "tag_order", "codec_tag", "profile", "sample_fmt",
}

// traceOf traces a harness output, with any ffprobe failure written into
// the trace the way the report shows it.
func traceOf(out []byte, pcm codec.PCM, info av.AudioInfo) trace.Trace {
	t, err := trace.Of(context.Background(), ffprobeBinary(), out, trace.Decoded{
		Planes: pcm.Planes, SampleRate: pcm.SampleRate, SampleFmt: info.SampleFmt.String(), Bitrate: info.Bitrate,
	})
	if err != nil {
		t.Probe = "unavailable: " + redactLocalPaths(err.Error())
	}
	return t
}

// differences is the canonical-workflow gate: Mist-clean and Mist-stego
// against the chosen ffmpeg invocation. A gap here is a failed objective.
func (t carrierTrace) differences() []string {
	return unionDiffs(trace.Diff(t.Canonical, t.Clean), trace.Diff(t.Canonical, t.Mist))
}

// defaultDifferences is the other threat model: a warden who re-encodes at
// ffmpeg's defaults. It does not decide the canonical verdict.
func (t carrierTrace) defaultDifferences() []string {
	return trace.Diff(t.FFmpeg, t.Mist)
}

func unionDiffs(a, b []string) []string {
	seen := map[string]bool{}
	var out []string
	for _, name := range append(append([]string{}, a...), b...) {
		if seen[name] {
			continue
		}
		seen[name] = true
		out = append(out, name)
	}
	return out
}

func ffprobeBinary() string {
	if p := os.Getenv("MIST_FFPROBE"); p != "" {
		return p
	}
	ff := harnessBinary("MIST_FFMPEG", "ffmpeg")
	if strings.ContainsAny(ff, `/\`) {
		sibling := filepath.Join(filepath.Dir(ff), "ffprobe")
		if st, err := os.Stat(sibling); err == nil && !st.IsDir() {
			return sibling
		}
	}
	return "ffprobe"
}

func metaFeatures(t trace.Trace) []float64 {
	return []float64{
		float64(t.Bytes), t.NominalKbps, t.Kbps, float64(t.Samples), float64(t.ZeroTail),
		float64(t.Bits), float64(t.Channels), float64(t.SampleRate),
		float64(t.Packets), float64(t.MeanPacket),
		float64(t.OggPages), float64(t.OggMeanPage), float64(t.OggSerials), float64(t.OggGranule), float64(t.OggFirst),
		boolNum(t.FlacMD5), float64(t.FlacSamples), float64(t.FlacBlocks), float64(t.Padding), float64(t.UnknownMeta),
		float64(t.SkipStart), float64(t.Streams),
		hashNum(t.Codec), hashNum(t.Format), hashNum(t.ChannelLayout), hashNum(t.TimeBase),
		hashNum(t.Encoder), hashNum(t.TagOrder), hashNum(t.CodecTag), hashNum(t.Profile), hashNum(t.SampleFmt),
	}
}

func boolNum(v bool) float64 {
	if v {
		return 1
	}
	return 0
}

func hashNum(s string) float64 {
	if s == "" {
		return 0
	}
	h := fnv.New32a()
	_, _ = h.Write([]byte(s))
	return float64(h.Sum32())
}

func scoreMetadata(traces []carrierTrace, groups []int) []metaResult {
	blank := func(note string) []metaResult {
		return []metaResult{
			{Question: "canonical-vs-mist-clean", Note: note},
			{Question: "mist-clean-vs-stego", Note: note},
		}
	}
	if len(traces) != len(groups) || len(traces) < 2 {
		return blank("fewer than two carriers")
	}
	canon := make([]trace.Trace, len(traces))
	clean := make([]trace.Trace, len(traces))
	stego := make([]trace.Trace, len(traces))
	for i, t := range traces {
		canon[i], clean[i], stego[i] = t.Canonical, t.Clean, t.Mist
	}
	return []metaResult{
		metaQuestion("canonical-vs-mist-clean", clean, canon, groups),
		metaQuestion("mist-clean-vs-stego", stego, clean, groups),
	}
}

func metaQuestion(question string, pos, neg []trace.Trace, groups []int) metaResult {
	if len(pos) != len(neg) || len(pos) != len(groups) || len(pos) < 2 {
		return metaResult{Question: question, Note: "fewer than two carriers"}
	}
	x := make([][]float64, 0, len(pos)*2)
	y := make([]bool, 0, len(pos)*2)
	g := make([]int, 0, len(pos)*2)
	for i := range pos {
		x = append(x, metaFeatures(pos[i]))
		y = append(y, true)
		g = append(g, groups[i])
		x = append(x, metaFeatures(neg[i]))
		y = append(y, false)
		g = append(g, groups[i])
	}
	scores := steganalysis.NestedCrossValidate(x, y, g, harnessFolds)
	var posS, negS []float64
	var posG, negG []int
	for i, s := range scores {
		if y[i] {
			posS = append(posS, s)
			posG = append(posG, g[i])
			continue
		}
		negS = append(negS, s)
		negG = append(negG, g[i])
	}
	auc := steganalysis.AUC(posS, negS)
	lo, hi := steganalysis.AUCInterval(posS, negS, posG, negG, harnessRounds, harnessSeed)
	return metaResult{
		Question:      question,
		FileAUC:       auc,
		FileLo:        lo,
		FileHi:        hi,
		Detectability: steganalysis.Detectability(auc),
		Scored:        true,
		Passed:        lo <= 0.5 && hi >= 0.5,
	}
}

func (s *HarnessSuite) TestFlacBlockLayoutMatchesFFmpeg() {
	if _, err := exec.LookPath(harnessBinary("MIST_FFMPEG", "ffmpeg")); err != nil {
		s.T().Skip("ffmpeg not on PATH")
	}
	data := wav(44100, 2, 44100)
	f, err := lookupFormat("flac", "")
	s.Require().NoError(err)
	mist, err := mistTwin(f, data)
	s.Require().NoError(err)
	ff, err := ffmpegTwin(context.Background(), "flac", "", ".wav", data)
	s.Require().NoError(err)
	s.Equal(ff[8:12], mist[8:12], "FLAC min/max block size")
	s.Equal(flacBlocks(ff), flacBlocks(mist))
}

func flacBlocks(b []byte) []string {
	if len(b) < 4 || string(b[:4]) != "fLaC" {
		return []string{"not flac"}
	}
	var out []string
	off := 4
	for off+4 <= len(b) {
		hdr := binary.BigEndian.Uint32(b[off : off+4])
		last := hdr&0x80000000 != 0
		typ := int((hdr >> 24) & 0x7f)
		size := int(hdr & 0xffffff)
		out = append(out, fmt.Sprintf("type=%d size=%d last=%v", typ, size, last))
		off += 4 + size
		if last || size < 0 {
			break
		}
	}
	return out
}

func (s *HarnessSuite) TestFfprobeReadsAMistFlac() {
	if _, err := exec.LookPath(ffprobeBinary()); err != nil {
		s.T().Skip("ffprobe not on PATH")
	}
	f, err := lookupFormat("flac", "")
	s.Require().NoError(err)
	out, err := mistTwin(f, wav(8000, 1, 8000))
	s.Require().NoError(err)
	t, err := trace.Of(context.Background(), ffprobeBinary(), out, trace.Decoded{})
	s.Require().NoError(err)
	s.Equal("ok", t.Probe)
	s.True(t.FlacMD5)
	s.Equal(int64(8000), t.FlacSamples)
}

func (s *HarnessSuite) TestCanonicalNominalIsNotTheDefaultThreat() {
	same := trace.Trace{Samples: 10, SampleFmt: "fltp", NominalKbps: 128}
	t := carrierTrace{Canonical: same, Clean: same, Mist: same, FFmpeg: trace.Trace{Samples: 10, SampleFmt: "fltp", NominalKbps: 80}}
	s.Empty(t.differences())
	s.Equal([]string{"nominal bitrate"}, t.defaultDifferences())
}

func (s *HarnessSuite) TestFingerprintFailsWhenMistCleanDiffers() {
	same := trace.Trace{Samples: 10, SampleFmt: "s16", NominalKbps: 100}
	clean := same
	clean.ZeroTail = 4
	f := formatReport{Measured: 1, Traces: []carrierTrace{{Canonical: same, Clean: clean, Mist: same}}}
	lv, text := f.fingerprint()
	s.Equal(fail, lv)
	s.Contains(text, "zero tail")
}

func (s *HarnessSuite) TestMetadataWardenSeparatesFileSize() {
	same := trace.Trace{Samples: 1000, SampleFmt: "s16", NominalKbps: 128, Bytes: 1000, Codec: "flac", Format: "flac"}
	bigger := same
	bigger.Bytes = 9000
	groups := []int{0, 1, 2, 3, 4, 5}
	pos := []trace.Trace{bigger, bigger, bigger, bigger, bigger, bigger}
	neg := []trace.Trace{same, same, same, same, same, same}
	got := metaQuestion("pipeline", pos, neg, groups)
	s.True(got.Scored)
	s.Greater(got.Detectability, 0.9)
	s.False(got.Passed)

	quiet := metaQuestion("match", neg, neg, groups)
	s.True(quiet.Scored)
	s.InDelta(0.5, quiet.FileAUC, 0.02)
	s.True(quiet.Passed)
}

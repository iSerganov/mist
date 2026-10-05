//go:build harness

package mist

import (
	"bytes"
	"context"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"hash/fnv"
	"math"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/iSerganov/mist/internal/av"
	"github.com/iSerganov/mist/internal/codec"
	"github.com/iSerganov/mist/internal/steganalysis"
)

// canonicalWorkflow is the innocent encode the fingerprint verdict uses.
// Vorbis follows the quality level Mist already opens (ffmpeg -q:a N).
// A lossless codec has no quality knob, so the canonical encode is ffmpeg's
// own defaults. Default ffmpeg on Vorbis is a second threat model.
const canonicalWorkflow = "vorbis: ffmpeg -q:a at Mist's source-conditioned level; lossless: ffmpeg defaults"

// metaThreshold is the preregistered pass for the audio-blind classifier.
// An interval that does not contain 0.5 is a failed objective.
const metaThreshold = "metadata classifier file interval includes 0.5"

// nominalSlack is how far two nominal bitrates may differ and still read
// as the same encoder setting rather than a different rate rule.
const nominalSlack = 0.05

// outputTrace is what a file shows without looking at samples: container
// identity, how long it is, how it ends, and the page or STREAMINFO facts
// a metadata warden can read.
type outputTrace struct {
	Samples       int     `json:"samples"`
	ZeroTail      int     `json:"zero_tail"`
	SampleFmt     string  `json:"sample_fmt"`
	NominalKbps   float64 `json:"nominal_kbps"`
	Kbps          float64 `json:"kbps"`
	Bytes         int     `json:"bytes,omitempty"`
	Codec         string  `json:"codec,omitempty"`
	Format        string  `json:"format,omitempty"`
	Channels      int     `json:"channels,omitempty"`
	SampleRate    int     `json:"sample_rate,omitempty"`
	Bits          int     `json:"bits,omitempty"`
	ChannelLayout string  `json:"channel_layout,omitempty"`
	TimeBase      string  `json:"time_base,omitempty"`
	Encoder       string  `json:"encoder,omitempty"`
	TagOrder      string  `json:"tag_order,omitempty"`
	CodecTag      string  `json:"codec_tag,omitempty"`
	Profile       string  `json:"profile,omitempty"`
	Streams       int     `json:"streams,omitempty"`
	Packets       int     `json:"packets,omitempty"`
	MeanPacket    int     `json:"mean_packet,omitempty"`
	SkipStart     int     `json:"skip_start,omitempty"`
	Probe         string  `json:"probe,omitempty"`
	Flac          bool    `json:"flac,omitempty"`
	FlacSamples   int64   `json:"flac_samples,omitempty"`
	FlacMD5       bool    `json:"flac_md5,omitempty"`
	FlacBlocks    int     `json:"flac_blocks,omitempty"`
	Padding       int     `json:"padding,omitempty"`
	UnknownMeta   int     `json:"unknown_meta,omitempty"`
	OggPages      int     `json:"ogg_pages,omitempty"`
	OggSerials    int     `json:"ogg_serials,omitempty"`
	OggMeanPage   int     `json:"ogg_mean_page,omitempty"`
	OggMeanLace   int     `json:"ogg_mean_lace,omitempty"`
	OggGranule    uint64  `json:"ogg_granule,omitempty"`
	OggFirst      uint64  `json:"ogg_first_granule,omitempty"`
}

// carrierTrace sets the four files the audit compares. FFmpeg is the
// default-ffmpeg threat model. Canonical is the chosen workflow. Clean is
// Mist with nothing embedded. Mist is the stego file.
type carrierTrace struct {
	Name           string      `json:"name"`
	Source         int         `json:"source_samples"`
	SourceRate     int         `json:"source_rate,omitempty"`
	SourceChannels int         `json:"source_channels,omitempty"`
	SourceFmt      string      `json:"source_sample_fmt,omitempty"`
	FFmpeg         outputTrace `json:"ffmpeg_default"`
	Canonical      outputTrace `json:"ffmpeg_canonical"`
	Clean          outputTrace `json:"mist_clean"`
	Mist           outputTrace `json:"mist"`
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

var sampleFmtNames = [...]string{"u8", "s16", "s32", "flt", "dbl", "u8p", "s16p", "s32p", "fltp", "dblp"}

func sampleFmtName(f codec.SampleFormat) string {
	if i := int(f); i >= 0 && i < len(sampleFmtNames) {
		return sampleFmtNames[i]
	}
	return "unknown"
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

func traceOf(out []byte, pcm codec.PCM, info av.AudioInfo) outputTrace {
	t := outputTrace{
		Samples:     pcm.NbSamples,
		ZeroTail:    zeroTail(pcm.Planes),
		SampleFmt:   "unknown",
		NominalKbps: float64(info.Bitrate) / 1000,
		Bytes:       len(out),
	}
	t.SampleFmt = sampleFmtName(info.SampleFmt)
	if pcm.NbSamples > 0 && pcm.SampleRate > 0 {
		t.Kbps = float64(len(out)) * 8 / 1000 / (float64(pcm.NbSamples) / float64(pcm.SampleRate))
	}
	enrichTrace(out, &t)
	return t
}

// zeroTail counts the trailing samples that are digital zero on every
// channel, which is what padding leaves behind.
func zeroTail(planes [][]float32) int {
	if len(planes) == 0 {
		return 0
	}
	n := 0
	for i := len(planes[0]) - 1; i >= 0; i-- {
		for _, p := range planes {
			if i < len(p) && p[i] != 0 {
				return n
			}
		}
		n++
	}
	return n
}

// differences is the canonical-workflow gate: Mist-clean and Mist-stego
// against the chosen ffmpeg invocation. A gap here is a failed objective.
func (t carrierTrace) differences() []string {
	return unionDiffs(gatedDiffs(t.Canonical, t.Clean), gatedDiffs(t.Canonical, t.Mist))
}

// defaultDifferences is the other threat model: a warden who re-encodes at
// ffmpeg's defaults. It does not decide the canonical verdict.
func (t carrierTrace) defaultDifferences() []string {
	return gatedDiffs(t.FFmpeg, t.Mist)
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

// gatedDiffs names identity fields that should match under one workflow.
// File size, packet size and page size stay out: VBR and embedding move
// them, and the metadata classifier is what scores those. Ogg serial
// values stay out because each encode draws a new one.
func gatedDiffs(a, b outputTrace) []string {
	var out []string
	if a.Samples != b.Samples {
		out = append(out, "length")
	}
	if a.ZeroTail != b.ZeroTail {
		out = append(out, "zero tail")
	}
	if a.SampleFmt != b.SampleFmt {
		out = append(out, "sample format")
	}
	if math.Abs(a.NominalKbps-b.NominalKbps) > nominalSlack*max(a.NominalKbps, b.NominalKbps) {
		out = append(out, "nominal bitrate")
	}
	differString("codec", a.Codec, b.Codec, &out)
	differString("container", a.Format, b.Format, &out)
	differInt("channels", a.Channels, b.Channels, &out)
	differInt("sample rate", a.SampleRate, b.SampleRate, &out)
	differInt("raw depth", a.Bits, b.Bits, &out)
	differString("channel layout", a.ChannelLayout, b.ChannelLayout, &out)
	differString("timebase", a.TimeBase, b.TimeBase, &out)
	differString("encoder tag", a.Encoder, b.Encoder, &out)
	differString("metadata tags", a.TagOrder, b.TagOrder, &out)
	differString("codec tag", a.CodecTag, b.CodecTag, &out)
	differString("profile", a.Profile, b.Profile, &out)
	differInt("stream count", a.Streams, b.Streams, &out)
	differInt("skip samples", a.SkipStart, b.SkipStart, &out)
	if a.Flac || b.Flac {
		if a.FlacMD5 != b.FlacMD5 {
			out = append(out, "flac md5")
		}
		if a.FlacSamples != b.FlacSamples {
			out = append(out, "flac sample count")
		}
	}
	if a.UnknownMeta != b.UnknownMeta && (a.UnknownMeta > 0 || b.UnknownMeta > 0) {
		out = append(out, "unknown metadata")
	}
	if a.OggPages > 0 || b.OggPages > 0 {
		if a.OggSerials != b.OggSerials {
			out = append(out, "ogg serials")
		}
		if a.OggGranule != b.OggGranule {
			out = append(out, "ogg granule")
		}
	}
	return out
}

func differString(name, a, b string, out *[]string) {
	if a == "" && b == "" {
		return
	}
	if a != b {
		*out = append(*out, name)
	}
}

func differInt(name string, a, b int, out *[]string) {
	if a == 0 && b == 0 {
		return
	}
	if a != b {
		*out = append(*out, name)
	}
}

func enrichTrace(out []byte, t *outputTrace) {
	summarizeOgg(out, t)
	summarizeFlac(out, t)
	applyFfprobe(out, t)
}

func summarizeOgg(b []byte, t *outputTrace) {
	if !bytes.HasPrefix(b, []byte("OggS")) {
		return
	}
	serials := map[uint32]struct{}{}
	var bodies, laces int
	var first, last uint64
	saw := false
	off := 0
	for off+27 <= len(b) && string(b[off:off+4]) == "OggS" {
		granule := binary.LittleEndian.Uint64(b[off+6 : off+14])
		serial := binary.LittleEndian.Uint32(b[off+14 : off+18])
		nseg := int(b[off+26])
		if off+27+nseg > len(b) {
			break
		}
		body := 0
		for i := range nseg {
			body += int(b[off+27+i])
		}
		if off+27+nseg+body > len(b) {
			break
		}
		serials[serial] = struct{}{}
		t.OggPages++
		bodies += body
		laces += nseg
		if granule != ^uint64(0) {
			if !saw {
				first = granule
				saw = true
			}
			last = granule
		}
		off += 27 + nseg + body
	}
	t.OggSerials = len(serials)
	if t.OggPages > 0 {
		t.OggMeanPage = bodies / t.OggPages
		t.OggMeanLace = laces / t.OggPages
	}
	if saw {
		t.OggFirst = first
		t.OggGranule = last
	}
}

func summarizeFlac(b []byte, t *outputTrace) {
	if !bytes.HasPrefix(b, []byte("fLaC")) {
		return
	}
	t.Flac = true
	off := 4
	for off+4 <= len(b) {
		hdr := binary.BigEndian.Uint32(b[off : off+4])
		last := hdr&0x80000000 != 0
		typ := int((hdr >> 24) & 0x7f)
		size := int(hdr & 0xffffff)
		off += 4
		if off+size > len(b) {
			return
		}
		body := b[off : off+size]
		off += size
		switch typ {
		case 0:
			if len(body) >= 34 {
				t.FlacSamples = int64(bitsAt(body, 108, 36))
				t.FlacMD5 = nonzero(body[18:34])
			}
		case 1:
			t.Padding += size
		default:
			if typ > 6 {
				t.UnknownMeta++
			} else {
				t.FlacBlocks++
			}
		}
		if last {
			return
		}
	}
}

func bitsAt(b []byte, bit, n int) uint64 {
	var v uint64
	for i := range n {
		p := bit + i
		if p/8 >= len(b) {
			return v
		}
		v = (v << 1) | uint64((b[p/8]>>(7-uint(p%8)))&1)
	}
	return v
}

func nonzero(b []byte) bool {
	for _, c := range b {
		if c != 0 {
			return true
		}
	}
	return false
}

func applyFfprobe(out []byte, t *outputTrace) {
	bin := ffprobeBinary()
	dir, err := os.MkdirTemp("", "mist-probe-*")
	if err != nil {
		t.Probe = "unavailable"
		return
	}
	defer func() { _ = os.RemoveAll(dir) }()
	path := filepath.Join(dir, "out"+sniffExt(out))
	if err := os.WriteFile(path, out, 0o600); err != nil {
		t.Probe = "unavailable"
		return
	}
	cmd := exec.Command(bin, "-v", "error", "-count_packets", "-show_format", "-show_streams", "-of", "json", path)
	raw, err := cmd.Output()
	if err != nil {
		msg := err.Error()
		var exit *exec.ExitError
		if errors.As(err, &exit) && len(bytes.TrimSpace(exit.Stderr)) > 0 {
			msg = string(bytes.TrimSpace(exit.Stderr))
		}
		t.Probe = "unavailable: " + redactLocalPaths(msg, dir, path)
		return
	}
	var doc probeDoc
	if err := json.Unmarshal(raw, &doc); err != nil {
		t.Probe = "unavailable: " + err.Error()
		return
	}
	t.Probe = "ok"
	if t.Format == "" {
		t.Format = doc.Format.FormatName
	}
	keys, vals := orderedTags(doc.Format.Tags)
	t.TagOrder = strings.Join(keys, ",")
	t.Streams = int(doc.Format.NbStreams)
	t.Encoder = encoderTag(vals, nil)
	if len(doc.Streams) == 0 {
		return
	}
	st := doc.Streams[0]
	t.Codec = st.CodecName
	t.CodecTag = st.CodecTag
	t.Profile = st.Profile
	t.ChannelLayout = st.ChannelLayout
	t.TimeBase = st.TimeBase
	t.Channels = int(st.Channels)
	t.SampleRate = numberInt(st.SampleRate)
	if st.Bits > 0 {
		t.Bits = int(st.Bits)
	}
	t.Packets = numberInt(st.NbRead)
	if t.Packets == 0 {
		t.Packets = numberInt(st.NbFrames)
	}
	if t.Packets > 0 && t.Bytes > 0 {
		t.MeanPacket = t.Bytes / t.Packets
	}
	t.SkipStart = int(st.InitialPadding)
	_, streamVals := orderedTags(st.Tags)
	if enc := encoderTag(vals, streamVals); enc != "" {
		t.Encoder = enc
	}
	if t.TagOrder == "" {
		sk, _ := orderedTags(st.Tags)
		t.TagOrder = strings.Join(sk, ",")
	}
}

type probeDoc struct {
	Format struct {
		FormatName string          `json:"format_name"`
		NbStreams  flexInt         `json:"nb_streams"`
		Tags       json.RawMessage `json:"tags"`
	} `json:"format"`
	Streams []probeStream `json:"streams"`
}

type probeStream struct {
	CodecName      string          `json:"codec_name"`
	CodecTag       string          `json:"codec_tag_string"`
	Profile        string          `json:"profile"`
	ChannelLayout  string          `json:"channel_layout"`
	SampleRate     json.RawMessage `json:"sample_rate"`
	Channels       flexInt         `json:"channels"`
	Bits           flexInt         `json:"bits_per_raw_sample"`
	TimeBase       string          `json:"time_base"`
	NbFrames       json.RawMessage `json:"nb_frames"`
	NbRead         json.RawMessage `json:"nb_read_packets"`
	InitialPadding flexInt         `json:"initial_padding"`
	Tags           json.RawMessage `json:"tags"`
}

// flexInt accepts the number or the numeric string ffprobe emits for the
// same field, depending on the codec.
type flexInt int

func (n *flexInt) UnmarshalJSON(b []byte) error {
	b = bytes.TrimSpace(b)
	if len(b) == 0 || string(b) == "null" {
		return nil
	}
	f, err := strconv.ParseFloat(strings.Trim(string(b), `"`), 64)
	if err != nil {
		return err
	}
	*n = flexInt(f)
	return nil
}

func numberInt(raw json.RawMessage) int {
	raw = bytes.TrimSpace(raw)
	if len(raw) == 0 || string(raw) == "null" {
		return 0
	}
	s := strings.Trim(string(raw), `"`)
	f, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return 0
	}
	return int(f)
}

func orderedTags(raw json.RawMessage) ([]string, map[string]string) {
	raw = bytes.TrimSpace(raw)
	if len(raw) == 0 || string(raw) == "null" {
		return nil, nil
	}
	dec := json.NewDecoder(bytes.NewReader(raw))
	tok, err := dec.Token()
	if err != nil || tok != json.Delim('{') {
		return nil, nil
	}
	vals := map[string]string{}
	var keys []string
	for dec.More() {
		keyTok, err := dec.Token()
		if err != nil {
			break
		}
		key, _ := keyTok.(string)
		var val any
		if err := dec.Decode(&val); err != nil {
			break
		}
		keys = append(keys, key)
		vals[key] = fmt.Sprint(val)
	}
	return keys, vals
}

func encoderTag(formatTags, streamTags map[string]string) string {
	enc := tagValue(formatTags, "encoder")
	vendor := tagValue(formatTags, "vendor")
	if streamTags != nil {
		if enc == "" {
			enc = tagValue(streamTags, "encoder")
		}
		if vendor == "" {
			vendor = tagValue(streamTags, "vendor")
		}
	}
	if vendor == "" || vendor == enc {
		return enc
	}
	if enc == "" {
		return vendor
	}
	return enc + " | " + vendor
}

func tagValue(vals map[string]string, want string) string {
	for k, v := range vals {
		if strings.EqualFold(k, want) {
			return v
		}
	}
	return ""
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

func sniffExt(b []byte) string {
	switch {
	case bytes.HasPrefix(b, []byte("OggS")):
		return ".ogg"
	case bytes.HasPrefix(b, []byte("fLaC")):
		return ".flac"
	case bytes.HasPrefix(b, []byte("RIFF")):
		return ".wav"
	case bytes.HasPrefix(b, []byte("FORM")):
		return ".aiff"
	default:
		return ".bin"
	}
}

func metaFeatures(t outputTrace) []float64 {
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
	canon := make([]outputTrace, len(traces))
	clean := make([]outputTrace, len(traces))
	stego := make([]outputTrace, len(traces))
	for i, t := range traces {
		canon[i], clean[i], stego[i] = t.Canonical, t.Clean, t.Mist
	}
	return []metaResult{
		metaQuestion("canonical-vs-mist-clean", clean, canon, groups),
		metaQuestion("mist-clean-vs-stego", stego, clean, groups),
	}
}

func metaQuestion(question string, pos, neg []outputTrace, groups []int) metaResult {
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
	var t outputTrace
	enrichTrace(out, &t)
	s.Equal("ok", t.Probe)
	s.True(t.FlacMD5)
	s.Equal(int64(8000), t.FlacSamples)
}

func (s *HarnessSuite) TestCanonicalNominalIsNotTheDefaultThreat() {
	same := outputTrace{Samples: 10, SampleFmt: "fltp", NominalKbps: 128}
	t := carrierTrace{Canonical: same, Clean: same, Mist: same, FFmpeg: outputTrace{Samples: 10, SampleFmt: "fltp", NominalKbps: 80}}
	s.Empty(t.differences())
	s.Equal([]string{"nominal bitrate"}, t.defaultDifferences())
}

func (s *HarnessSuite) TestFingerprintFailsWhenMistCleanDiffers() {
	same := outputTrace{Samples: 10, SampleFmt: "s16", NominalKbps: 100}
	clean := same
	clean.ZeroTail = 4
	f := formatReport{Measured: 1, Traces: []carrierTrace{{Canonical: same, Clean: clean, Mist: same}}}
	lv, text := f.fingerprint()
	s.Equal(fail, lv)
	s.Contains(text, "zero tail")
}

func (s *HarnessSuite) TestOggSummaryIgnoresTheSerialValue() {
	body := bytes.Repeat([]byte{1}, 8)
	a := append(oggPage(1, 100, body), oggPage(1, 200, body)...)
	b := append(oggPage(99, 100, body), oggPage(99, 200, body)...)
	var left, right outputTrace
	summarizeOgg(a, &left)
	summarizeOgg(b, &right)
	s.Equal(1, left.OggSerials)
	s.Equal(uint64(200), left.OggGranule)
	s.Empty(gatedDiffs(left, right))
	right.OggGranule = 201
	s.Equal([]string{"ogg granule"}, gatedDiffs(left, right))
}

func (s *HarnessSuite) TestFlacStreamInfoReadsSamplesAndMD5() {
	present := flacFile(48000, []byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16})
	absent := flacFile(48000, make([]byte, 16))
	var withMD5, without outputTrace
	summarizeFlac(present, &withMD5)
	summarizeFlac(absent, &without)
	s.True(withMD5.Flac)
	s.Equal(int64(48000), withMD5.FlacSamples)
	s.True(withMD5.FlacMD5)
	s.False(without.FlacMD5)
	s.Equal([]string{"flac md5"}, gatedDiffs(withMD5, without))
}

func (s *HarnessSuite) TestMetadataWardenSeparatesFileSize() {
	same := outputTrace{Samples: 1000, SampleFmt: "s16", NominalKbps: 128, Bytes: 1000, Codec: "flac", Format: "flac"}
	bigger := same
	bigger.Bytes = 9000
	groups := []int{0, 1, 2, 3, 4, 5}
	pos := []outputTrace{bigger, bigger, bigger, bigger, bigger, bigger}
	neg := []outputTrace{same, same, same, same, same, same}
	got := metaQuestion("pipeline", pos, neg, groups)
	s.True(got.Scored)
	s.Greater(got.Detectability, 0.9)
	s.False(got.Passed)

	quiet := metaQuestion("match", neg, neg, groups)
	s.True(quiet.Scored)
	s.InDelta(0.5, quiet.FileAUC, 0.02)
	s.True(quiet.Passed)
}

func oggPage(serial uint32, granule uint64, body []byte) []byte {
	page := make([]byte, 28+len(body))
	copy(page, "OggS")
	binary.LittleEndian.PutUint64(page[6:14], granule)
	binary.LittleEndian.PutUint32(page[14:18], serial)
	page[26] = 1
	page[27] = byte(len(body))
	copy(page[28:], body)
	return page
}

func flacFile(samples uint64, md5 []byte) []byte {
	info := make([]byte, 34)
	bit := 0
	writeBits(info, &bit, 4096, 16)
	writeBits(info, &bit, 4096, 16)
	writeBits(info, &bit, 0, 24)
	writeBits(info, &bit, 0, 24)
	writeBits(info, &bit, 44100, 20)
	writeBits(info, &bit, 1, 3)
	writeBits(info, &bit, 15, 5)
	writeBits(info, &bit, samples, 36)
	copy(info[18:34], md5)
	block := []byte{0x80, 0x00, 0x00, 34}
	return append(append([]byte("fLaC"), block...), info...)
}

func writeBits(dst []byte, bit *int, v uint64, n int) {
	for i := n - 1; i >= 0; i-- {
		if v&(1<<uint(i)) != 0 {
			dst[*bit/8] |= 1 << (7 - uint(*bit%8))
		}
		*bit++
	}
}

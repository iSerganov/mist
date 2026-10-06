package trace

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
)

func probe(ctx context.Context, ffprobe string, b []byte, t *Trace) error {
	dir, err := os.MkdirTemp("", "mist-probe-*")
	if err != nil {
		return err
	}
	defer func() { _ = os.RemoveAll(dir) }()
	path := filepath.Join(dir, "out"+Ext(b))
	if err := os.WriteFile(path, b, 0o600); err != nil {
		return err
	}
	cmd := exec.CommandContext(ctx, ffprobe, "-v", "error", "-count_packets", "-show_format", "-show_streams", "-of", "json", path)
	raw, err := cmd.Output()
	if err != nil {
		var exit *exec.ExitError
		if errors.As(err, &exit) && len(bytes.TrimSpace(exit.Stderr)) > 0 {
			return errors.New(string(bytes.TrimSpace(exit.Stderr)))
		}
		return err
	}
	var doc probeDoc
	if err := json.Unmarshal(raw, &doc); err != nil {
		return fmt.Errorf("ffprobe output: %w", err)
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
		return nil
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
	return nil
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

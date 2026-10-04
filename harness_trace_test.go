//go:build harness

package mist

import (
	"math"

	"github.com/iSerganov/mist/internal/av"
	"github.com/iSerganov/mist/internal/codec"
)

// nominalSlack is how far two nominal bitrates may differ and still read
// as the same encoder setting rather than a different rate rule.
const nominalSlack = 0.05

// outputTrace is what a file shows without any statistics: how long it
// is, how it ends, what it decodes to and what rate it claims.
type outputTrace struct {
	Samples     int     `json:"samples"`
	ZeroTail    int     `json:"zero_tail"`
	SampleFmt   string  `json:"sample_fmt"`
	NominalKbps float64 `json:"nominal_kbps"`
	Kbps        float64 `json:"kbps"`
}

// carrierTrace sets Mist's output beside a plain ffmpeg encode of the
// same carrier. Any field that differs tells the two apart outright.
type carrierTrace struct {
	Name   string      `json:"name"`
	Source int         `json:"source_samples"`
	FFmpeg outputTrace `json:"ffmpeg"`
	Mist   outputTrace `json:"mist"`
}

var sampleFmtNames = [...]string{"u8", "s16", "s32", "flt", "dbl", "u8p", "s16p", "s32p", "fltp", "dblp"}

func traceOf(out []byte, pcm codec.PCM, info av.AudioInfo) outputTrace {
	t := outputTrace{
		Samples:     pcm.NbSamples,
		ZeroTail:    zeroTail(pcm.Planes),
		SampleFmt:   "unknown",
		NominalKbps: float64(info.Bitrate) / 1000,
	}
	if f := int(info.SampleFmt); f >= 0 && f < len(sampleFmtNames) {
		t.SampleFmt = sampleFmtNames[f]
	}
	if pcm.NbSamples > 0 && pcm.SampleRate > 0 {
		t.Kbps = float64(len(out)) * 8 / 1000 / (float64(pcm.NbSamples) / float64(pcm.SampleRate))
	}
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

// differences names every trace field where Mist's output is not what
// ffmpeg would have written.
func (t carrierTrace) differences() []string {
	var out []string
	if t.Mist.Samples != t.FFmpeg.Samples {
		out = append(out, "length")
	}
	if t.Mist.ZeroTail != t.FFmpeg.ZeroTail {
		out = append(out, "zero tail")
	}
	if t.Mist.SampleFmt != t.FFmpeg.SampleFmt {
		out = append(out, "sample format")
	}
	if math.Abs(t.Mist.NominalKbps-t.FFmpeg.NominalKbps) > nominalSlack*max(t.Mist.NominalKbps, t.FFmpeg.NominalKbps) {
		out = append(out, "nominal bitrate")
	}
	return out
}

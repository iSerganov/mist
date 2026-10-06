package mist

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"slices"

	"github.com/iSerganov/mist/internal/steganalysis"
)

// calibrationJSON is what `make calibrate` wrote from a harness run: every
// detector's file scores on clean and stego copies of the same carriers,
// and the trained wardens refit on all of them.
//
//go:embed calibration.json
var calibrationJSON []byte

// calibration is a harness run reduced to what Analyze reads one file
// against. It names the run so a report can say where its numbers came from.
type calibration struct {
	Commit   string             `json:"commit"`
	Corpus   string             `json:"corpus"`
	Carriers int                `json:"carriers"`
	Formats  []calibratedFormat `json:"formats"`
}

// calibratedFormat is one output format's reference: Codec and Container
// as libav names them, so a suspect file's own stream can find it.
type calibratedFormat struct {
	Format    string            `json:"format"`
	Codec     string            `json:"codec"`
	Container string            `json:"container"`
	Lossless  bool              `json:"lossless"`
	Stages    []calibratedStage `json:"stages"`
}

// calibratedStage is one detector's reference populations. A trained
// warden also carries its model; its populations are out-of-fold scores
// of the same training rule, so they describe files the model never saw.
type calibratedStage struct {
	Name string `json:"name"`
	steganalysis.Reference
	Model *steganalysis.Logistic `json:"model,omitempty"`
}

func loadCalibration(raw []byte) (calibration, error) {
	var c calibration
	if err := json.Unmarshal(raw, &c); err != nil {
		return calibration{}, fmt.Errorf("calibration: %w", err)
	}
	return c, nil
}

// lookup finds the reference for a stream: the same codec first, then the
// same lossless container, then any lossless format, since every lossless
// codec carries Mist's bits the same way, in PCM sample LSBs. The note says
// when the match is not exact.
func (c calibration) lookup(codec, container string, lossless bool) (calibratedFormat, string, bool) {
	if i := slices.IndexFunc(c.Formats, func(f calibratedFormat) bool { return f.Codec == codec }); i >= 0 {
		return c.Formats[i], "", true
	}
	if !lossless {
		return calibratedFormat{}, "", false
	}
	if i := slices.IndexFunc(c.Formats, func(f calibratedFormat) bool { return f.Lossless && f.Container == container }); i >= 0 {
		return c.Formats[i], "calibrated on " + c.Formats[i].Format, true
	}
	if i := slices.IndexFunc(c.Formats, func(f calibratedFormat) bool { return f.Lossless }); i >= 0 {
		return c.Formats[i], "calibrated on " + c.Formats[i].Format, true
	}
	return calibratedFormat{}, "", false
}

func (f calibratedFormat) stage(name string) (calibratedStage, bool) {
	i := slices.IndexFunc(f.Stages, func(s calibratedStage) bool { return s.Name == name })
	if i < 0 {
		return calibratedStage{}, false
	}
	return f.Stages[i], true
}

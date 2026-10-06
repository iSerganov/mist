//go:build harness

package mist

import (
	"encoding/json"
	"os"

	"github.com/iSerganov/mist/internal/steganalysis"
)

// calibrate reduces one format's run to the reference Analyze reads a
// single file against. The classical detectors keep their file scores. Each
// trained warden is cross-validated again with the plain training rule, so
// its populations are out-of-fold scores of exactly the model it then
// refits on every chunk.
func calibrate(f Format, raw []rawDetectorScores, pos, neg [][]chunk, familyOf []int) calibratedFormat {
	out := calibratedFormat{Format: f.String(), Codec: f.Codec, Container: f.Container, Lossless: f.Lossless}
	for _, d := range raw {
		if trainedDetector(d.Name) {
			continue
		}
		out.Stages = append(out.Stages, calibratedStage{
			Name:      d.Name,
			Reference: steganalysis.Reference{Clean: fileScores(d.Clean.Files), Stego: fileScores(d.Stego.Files)},
		})
	}
	for _, w := range trainedWardens {
		m, recording := matrix(pos, neg, w.pick, familyOf)
		ps, ns := split(steganalysis.CrossValidate(m.x, m.y, m.groups, harnessFolds), m, recording)
		out.Stages = append(out.Stages, calibratedStage{
			Name:      w.name,
			Reference: steganalysis.Reference{Clean: ns.perFile().vals, Stego: ps.perFile().vals},
			Model:     steganalysis.TrainLogistic(m.x, m.y),
		})
	}
	return out
}

func fileScores(files []rawScore) []float64 {
	out := make([]float64, len(files))
	for i, f := range files {
		out[i] = f.Score
	}
	return out
}

// writeCalibration writes every calibrated format of the run. Carrier
// names stay out: a reference needs the scores, not where they came from.
func (r harnessReport) writeCalibration(path string) error {
	c := calibration{Commit: r.Commit, Corpus: r.Corpus, Carriers: r.Carriers}
	for _, f := range r.Formats {
		if f.calibration != nil {
			c.Formats = append(c.Formats, *f.calibration)
		}
	}
	raw, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, append(raw, '\n'), 0o644)
}

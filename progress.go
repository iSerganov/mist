package mist

// ProgressStage names the part of the work a Progress report describes.
type ProgressStage int

// Stages reported by an Emitter (ProgressEmbed) and a Catcher
// (ProgressScan, ProgressAssemble).
const (
	ProgressEmbed ProgressStage = iota + 1
	ProgressScan
	ProgressAssemble
)

// Progress counts frames for ProgressEmbed and ProgressScan and message
// bytes for ProgressAssemble. Total is 0 when it is not known, as on a live stream.
type Progress struct {
	Stage ProgressStage
	Done  int
	Total int
}

// ProgressFunc receives Progress reports on the goroutine doing the work,
// so it must return quickly.
type ProgressFunc func(Progress)

func (f ProgressFunc) report(stage ProgressStage, done, total int) {
	if f != nil {
		f(Progress{Stage: stage, Done: done, Total: total})
	}
}

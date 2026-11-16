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
// bytes for ProgressAssemble. Total is 0 when it is not known, as on a live
// stream. For ProgressEmbed it follows the carrier's stated duration, and
// rises to Done should the carrier run longer.
type Progress struct {
	Stage ProgressStage
	Done  int
	Total int
}

// ProgressFunc receives Progress reports one at a time, never concurrently,
// but not always on the caller's goroutine: a Vorbis embed reports from the
// goroutine writing its output. It must return quickly.
type ProgressFunc func(Progress)

func (f ProgressFunc) report(stage ProgressStage, done, total int) {
	if f != nil {
		f(Progress{Stage: stage, Done: done, Total: total})
	}
}

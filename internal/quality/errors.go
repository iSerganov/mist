package quality

import "errors"

var (
	// ErrUnavailable means the perceptual tool or ffmpeg is not on PATH.
	ErrUnavailable = errors.New("quality: perceptual tool or ffmpeg not on PATH")
	// ErrNoScore means the tool ran but printed no score this package recognises.
	ErrNoScore = errors.New("quality: no score in tool output")
)

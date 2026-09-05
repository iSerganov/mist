package av

import "errors"

var (
	ErrInit   = errors.New("av: init failed")
	ErrOpen   = errors.New("av: open failed")
	ErrRead   = errors.New("av: read failed")
	ErrWrite  = errors.New("av: write failed")
	ErrEOF    = errors.New("av: eof")
	ErrClosed = errors.New("av: closed")
	// ErrAgain is libav EAGAIN: send/receive has no output yet, not a failure.
	ErrAgain         = errors.New("av: again")
	errUnimplemented = errors.New("av: unimplemented")
)

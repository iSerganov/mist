package av

import "errors"

var (
	ErrInit   = errors.New("av: init failed")
	ErrOpen   = errors.New("av: open failed")
	ErrRead   = errors.New("av: read failed")
	ErrWrite  = errors.New("av: write failed")
	ErrEOF    = errors.New("av: eof")
	ErrClosed = errors.New("av: closed")
	// ErrInvalid is libav EINVAL from opening a codec: it refused the
	// settings it was given, not that the codec is missing.
	ErrInvalid = errors.New("av: invalid argument")
	// ErrAgain is libav EAGAIN: send/receive has no output yet, not a failure.
	ErrAgain         = errors.New("av: again")
	errUnimplemented = errors.New("av: unimplemented")
)

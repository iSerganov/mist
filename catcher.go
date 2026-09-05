package mist

import (
	"context"
	"errors"
	"io"
	"os"
	"time"
)

// CatcherOption configures NewCatcher.
type CatcherOption func(*Catcher)

// WithMaxRetries configures the maximum number of retries for failed URL connections.
func WithMaxRetries(retries int) CatcherOption {
	return func(o *Catcher) {
		o.maxRetries = retries
	}
}

// WithBackoff configures the backoff duration between retries.
func WithBackoff(backoff time.Duration) CatcherOption {
	return func(o *Catcher) {
		o.backoff = backoff
	}
}

// Catcher extracts payloads with a recipient private key.
// It is a value type: construct it once with NewCatcher and share it.
// Methods use value receivers and do not mutate the Catcher.
type Catcher struct {
	priv       []byte
	maxRetries int
	backoff    time.Duration
}

// NewCatcher returns an immutable Catcher for privKey.
// The key and any option-held secrets are copied; later writes to the
// caller's slices are not observed.
func NewCatcher(privKey []byte, opts ...CatcherOption) (*Catcher, error) {
	if privKey == nil {
		return nil, errors.New("private key is required")
	}
	res := &Catcher{
		priv: append([]byte(nil), privKey...),
	}
	for _, opt := range opts {
		opt(res)
	}
	return res, nil
}

// Listen opens source (a local file path, file:// URL, or http(s):// URL),
// continuously scans for stego frames, and sends each successfully
// decrypted payload to the returned channel.
//
// The returned channel is closed when either:
//   - the source reaches EOF (finite file or stream ended by the server), or
//   - ctx is cancelled.
//
// The caller distinguishes the two after the channel closes:
//   - ctx.Err() == nil  →  source reached EOF naturally
//   - ctx.Err() != nil  →  context was cancelled by the caller
//
// The error return covers only immediate setup failures (invalid source,
// unreadable file, malformed key). Per-frame decrypt failures are silently
// skipped — they are indistinguishable from frames carrying no payload
// and must not surface as errors.
func (c *Catcher) Listen(ctx context.Context, source string) (<-chan Result, error) {
	_ = ctx
	_ = source
	_ = c
	return nil, errUnimplemented
}

// ListenReader is Listen over an already-open bitstream (pipe, HTTP body,
// in-memory buffer). The caller retains ownership of r and must close it
// if it is an io.Closer.
func (c *Catcher) ListenReader(ctx context.Context, r io.Reader) (<-chan Result, error) {
	_ = ctx
	_ = r
	_ = c
	return nil, errUnimplemented
}

// Extract scans source synchronously and returns every frame that
// decrypts and authenticates. It stops at EOF or when ctx is cancelled.
// source stays owned by the caller.
func (c *Catcher) Extract(ctx context.Context, source *os.File) ([]Result, error) {
	_ = ctx
	_ = source
	_ = c
	return nil, errUnimplemented
}

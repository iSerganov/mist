package mist

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"
	"time"

	"github.com/iSerganov/mist/internal/av"
)

// CatcherOption configures NewCatcher.
type CatcherOption func(*Catcher)

// WithMaxRetries configures how many consecutive read failures a live
// source may produce before Listen gives up and closes the channel.
func WithMaxRetries(retries int) CatcherOption {
	return func(c *Catcher) { c.maxRetries = retries }
}

// WithBackoff configures the pause between read retries.
func WithBackoff(backoff time.Duration) CatcherOption {
	return func(c *Catcher) { c.backoff = backoff }
}

// WithLogger directs diagnostics — such as joining a stream mid-frame — to
// log. Failed decrypts are never logged: they are indistinguishable from
// frames carrying no payload. Discarded by default.
func WithLogger(log *slog.Logger) CatcherOption {
	return func(c *Catcher) { c.log = log }
}

// Catcher extracts payloads with a recipient private key. Construct it once
// with NewCatcher and share it; methods do not mutate it.
type Catcher struct {
	priv       []byte
	maxRetries int
	backoff    time.Duration
	log        *slog.Logger
}

// NewCatcher returns an immutable Catcher for privKey.
// The key and any option-held secrets are copied; later writes to the
// caller's slices are not observed.
func NewCatcher(privKey []byte, opts ...CatcherOption) (*Catcher, error) {
	if privKey == nil {
		return nil, ErrInvalidKey
	}
	c := &Catcher{priv: append([]byte(nil), privKey...)}
	for _, opt := range opts {
		opt(c)
	}
	return c, nil
}

func (c *Catcher) logger() *slog.Logger {
	if c.log != nil {
		return c.log
	}
	return slog.New(slog.DiscardHandler)
}

// Listen opens source (a local file path, file:// URL, or http(s):// URL),
// continuously scans for stego frames, and sends each successfully decrypted
// payload to the returned channel.
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
	if err := c.ready(ctx); err != nil {
		return nil, err
	}
	if source == "" {
		return nil, ErrInvalidSource
	}
	if ctx.Err() != nil {
		return closed(), nil
	}
	d, err := av.OpenDemuxer(source)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrCarrier, err)
	}
	return c.stream(ctx, d)
}

// ListenReader is Listen over an already-open bitstream (pipe, HTTP body,
// in-memory buffer). The caller retains ownership of r and must close it
// if it is an io.Closer.
func (c *Catcher) ListenReader(ctx context.Context, r io.Reader) (<-chan Result, error) {
	if err := c.ready(ctx); err != nil {
		return nil, err
	}
	if r == nil {
		return nil, ErrInvalidSource
	}
	if ctx.Err() != nil {
		return closed(), nil
	}
	d, err := av.OpenDemuxerReader(asSeeker(r))
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrCarrier, err)
	}
	return c.stream(ctx, d)
}

// Extract scans source synchronously and returns every frame that
// decrypts and authenticates. It stops at EOF or when ctx is cancelled.
// source stays owned by the caller.
func (c *Catcher) Extract(ctx context.Context, source *os.File) ([]Result, error) {
	if err := c.ready(ctx); err != nil {
		return nil, err
	}
	if source == nil {
		return nil, ErrInvalidSource
	}
	if _, err := source.Seek(0, io.SeekStart); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrCarrier, err)
	}
	ch, err := c.ListenReader(ctx, source)
	if err != nil {
		return nil, err
	}
	var out []Result
	for r := range ch {
		out = append(out, r)
	}
	return out, ctx.Err()
}

func (c *Catcher) ready(ctx context.Context) error {
	if c == nil || len(c.priv) != PrivateKeySize {
		return ErrInvalidKey
	}
	if ctx == nil {
		return ErrInvalidSource
	}
	return nil
}

func closed() <-chan Result {
	ch := make(chan Result)
	close(ch)
	return ch
}

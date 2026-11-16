package mist

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"

	"github.com/iSerganov/mist/internal/av"
)

// EmitterOption configures NewEmitter.
type EmitterOption func(*Emitter)

// WithSenderAuth attaches an Ed25519 signature over the plaintext, itself
// encrypted before embedding. Off by default — an unauthenticated ciphertext
// is shorter and one step closer to looking like nothing.
func WithSenderAuth(senderPrivKey []byte) EmitterOption {
	return func(e *Emitter) {
		e.senderPriv = append([]byte(nil), senderPrivKey...)
	}
}

// WithFormat selects the output target the way ffmpeg's -f does: a
// container short name ("flac", "wav", "ogg") or the output path whose
// extension names one ("song.flac"). Ogg Vorbis by default.
func WithFormat(name string) EmitterOption {
	return func(e *Emitter) { e.format = name }
}

// WithCodec overrides the encoder the container would default to, the way
// ffmpeg's -c:a does — an m4a holds AAC unless told to hold ALAC.
func WithCodec(name string) EmitterOption {
	return func(e *Emitter) { e.codec = name }
}

// WithEmitProgress reports each frame Embed writes to f.
func WithEmitProgress(f ProgressFunc) EmitterOption {
	return func(e *Emitter) { e.progress = f }
}

// WithReseal seals the payload again, under a fresh ephemeral key, into
// every stego frame that holds it, so a listener who joins late reads it
// from the next frame. The payload must fit one frame: a re-sealed payload
// is never spanned, and one that fits no frame is ErrNoCapacity.
func WithReseal() EmitterOption {
	return func(e *Emitter) { e.reseal = true }
}

// Emitter encrypts a payload for a recipient public key and embeds it
// into a carrier. Construct it once with NewEmitter and share it. Methods
// do not mutate the Emitter.
type Emitter struct {
	pub        []byte
	senderPriv []byte
	format     string
	codec      string
	target     av.Format
	progress   ProgressFunc
	reseal     bool
}

// NewEmitter returns an Emitter for recipientPubKey.
// The key and any option-held secrets are copied; later writes to the
// caller's slices are not observed. An output format this FFmpeg cannot
// write, or a lossy one other than Vorbis, is ErrUnsupportedCodec here
// rather than after a carrier has been decoded.
func NewEmitter(recipientPubKey []byte, opts ...EmitterOption) (*Emitter, error) {
	if recipientPubKey == nil {
		return nil, ErrInvalidKey
	}
	e := &Emitter{
		pub: append([]byte(nil), recipientPubKey...),
	}
	for _, opt := range opts {
		opt(e)
	}
	target, err := lookupFormat(e.format, e.codec)
	if err != nil {
		return nil, err
	}
	e.target = target
	return e, nil
}

// Embed opens source (a local file path, file:// URL, or http(s):// URL)
// as the carrier, encrypts payload for the recipient, and returns the
// stego stream in the Emitter's output format, complete. Every stego frame
// not carrying the payload gets filler. A live source, a URL that states
// no duration, has no end to wait for and is refused: use EmbedTo.
func (e *Emitter) Embed(ctx context.Context, source string, payload Payload) (io.ReadCloser, error) {
	if e == nil {
		return nil, ErrInvalidKey
	}
	return e.collect(ctx, payload, func() (input, error) { return openPath(ctx, source) })
}

// EmbedTo is Embed writing to dst as the carrier is read, holding about
// one stego frame of audio in memory. When dst can seek, the container
// header is completed at the end, as in a file. Otherwise it stays as
// ffmpeg leaves it on a pipe. A failure, ErrNoCapacity included, can leave
// dst written. A live source runs until ctx is cancelled, which returns nil.
func (e *Emitter) EmbedTo(ctx context.Context, source string, dst io.Writer, payload Payload) error {
	if e == nil {
		return ErrInvalidKey
	}
	if dst == nil {
		return ErrInvalidSource
	}
	c, plain, err := e.begin(ctx, payload, func() (input, error) { return openPath(ctx, source) })
	if err != nil {
		return err
	}
	return e.embedTo(ctx, c, dst, plain)
}

// EmbedReader is Embed over an already-open carrier (pipe, HTTP body,
// in-memory buffer). The caller retains ownership of carrier and must
// close it if it is an io.Closer.
func (e *Emitter) EmbedReader(ctx context.Context, carrier io.Reader, payload Payload) (io.ReadCloser, error) {
	if e == nil {
		return nil, ErrInvalidKey
	}
	if carrier == nil {
		return nil, ErrInvalidSource
	}
	return e.collect(ctx, payload, func() (input, error) { return openReader(carrier) })
}

// EmbedFile is Embed over an *os.File carrier. The file stays owned by
// the caller.
func (e *Emitter) EmbedFile(ctx context.Context, carrier *os.File, payload Payload) (io.ReadCloser, error) {
	if e == nil {
		return nil, ErrInvalidKey
	}
	if carrier == nil {
		return nil, ErrInvalidSource
	}
	return e.collect(ctx, payload, func() (input, error) {
		if _, err := carrier.Seek(0, io.SeekStart); err != nil {
			return input{}, carrierErr(err)
		}
		return openReader(carrier)
	})
}

// collect runs the pipeline into memory. Only the compressed output is
// held, and the buffer can seek, so the header is the one a file gets.
func (e *Emitter) collect(ctx context.Context, payload Payload, open func() (input, error)) (io.ReadCloser, error) {
	c, plain, err := e.begin(ctx, payload, open)
	if err != nil {
		return nil, err
	}
	if c.live {
		c.close()
		return nil, fmt.Errorf("%w: a live source has no end to wait for; use EmbedTo", ErrInvalidSource)
	}
	var buf seekBuf
	if err := e.embedTo(ctx, c, &buf, plain); err != nil {
		return nil, err
	}
	return io.NopCloser(bytes.NewReader(buf.data)), nil
}

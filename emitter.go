package mist

import (
	"context"
	"fmt"
	"io"
	"os"
	"strings"
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

// Emitter encrypts a payload for a recipient public key and embeds it
// into an Ogg Vorbis carrier. Construct it once with NewEmitter and share
// it. Methods do not mutate the Emitter.
type Emitter struct {
	pub        []byte
	senderPriv []byte
}

// NewEmitter returns an Emitter for recipientPubKey.
// The key and any option-held secrets are copied; later writes to the
// caller's slices are not observed.
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
	return e, nil
}

// Embed opens source (a local file path, file:// URL, or http(s):// URL)
// as the carrier, encrypts payload for the recipient, and returns a new
// stego Ogg stream.
//
// The payload is looped across fixed-duration stego frames, each
// re-encrypted with a fresh ephemeral X25519 key, for the full duration
// of the carrier. Suitable for both finite files and wrapping a live
// encoder output — the returned reader produces output as fast as the
// carrier does.
func (e *Emitter) Embed(ctx context.Context, source string, payload Payload) (io.ReadCloser, error) {
	if e == nil {
		return nil, ErrInvalidKey
	}
	if strings.HasPrefix(source, "http://") || strings.HasPrefix(source, "https://") {
		return e.embedURL(ctx, source, payload)
	}
	rc, err := openSource(source)
	if err != nil {
		return nil, err
	}
	defer rc.Close()
	return e.embed(ctx, rc, payload)
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
	return e.embed(ctx, carrier, payload)
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
	if _, err := carrier.Seek(0, io.SeekStart); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrCarrier, err)
	}
	return e.embed(ctx, carrier, payload)
}

package mist

import (
	"context"
	"io"
	"os"
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
	_ = ctx
	_ = source
	_ = payload
	_ = e
	return nil, errUnimplemented
}

// EmbedReader is Embed over an already-open carrier (pipe, HTTP body,
// in-memory buffer). The caller retains ownership of carrier and must
// close it if it is an io.Closer.
func (e *Emitter) EmbedReader(ctx context.Context, carrier io.Reader, payload Payload) (io.ReadCloser, error) {
	_ = ctx
	_ = carrier
	_ = payload
	_ = e
	return nil, errUnimplemented
}

// EmbedFile is Embed over an *os.File carrier. The file stays owned by
// the caller.
func (e *Emitter) EmbedFile(ctx context.Context, carrier *os.File, payload Payload) (io.ReadCloser, error) {
	_ = ctx
	_ = carrier
	_ = payload
	_ = e
	return nil, errUnimplemented
}

// FrameCapacity returns the maximum payload bytes embeddable per stego
// frame at the library's fixed embedding density, before encryption
// overhead. Useful for callers who need to know the maximum message size
// supported without multi-frame spanning (Phase 2).
func FrameCapacity() int {
	return 0
}

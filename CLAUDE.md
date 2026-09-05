# Mist

Asymmetric-key audio steganography for Go. Phase 1: text payloads in Ogg Vorbis.

Sender needs only the recipient X25519 public key. Extraction needs the private key. The payload lives in compressed Vorbis audio packets, not Ogg comments or pages. A digital recording of a live stream is enough.

## Public API (`package mist`)

```go
GenerateKeyPair() (pub, priv []byte, err error)          // X25519
GenerateSigningKeyPair() (pub, priv []byte, err error)   // Ed25519, optional
NewEmitter(pub []byte, opts ...EmitterOption) (*Emitter, error)
    Emitter.Embed(ctx, source string, payload Payload) (io.ReadCloser, error)
    Emitter.EmbedReader(ctx, carrier io.Reader, payload Payload) (io.ReadCloser, error)
    Emitter.EmbedFile(ctx, carrier *os.File, payload Payload) (io.ReadCloser, error)
NewCatcher(priv []byte, opts ...CatcherOption) (*Catcher, error)
    Catcher.Listen(ctx, source string) (<-chan Result, error)
    Catcher.ListenReader(ctx, r io.Reader) (<-chan Result, error)
    Catcher.Extract(ctx, source *os.File) ([]Result, error)  // synchronous
FrameCapacity() int
WithSenderAuth(senderPriv []byte) EmitterOption
WithMaxRetries(n int) CatcherOption
WithBackoff(d time.Duration) CatcherOption
```

`Listen` / `Embed` `source` is a path, `file://` URL, or `http(s)://` URL. Finite files close the channel on EOF; live streams run until `ctx` is cancelled. Distinguish with `ctx.Err()` after range. `Catcher` and `Emitter` copy keys at construction; do not add setters. Embed methods return `io.ReadCloser` (no `mist.Reader` type); the caller must Close it.

Do not add `ErrNoMessage`. Failed AEAD / wrong phase / empty frame are the same silent skip. Exposing "something was there but I could not decrypt" is a side channel.

## Layout

```
emitter.go        NewEmitter, Embed / EmbedReader / EmbedFile → io.ReadCloser
catcher.go        NewCatcher, Listen / ListenReader / Extract
example/          Godoc examples: keys, Emitter, Catcher
mist.go-level     payload, keys, protocol constants, errors
internal/crypto   X25519 ECDH, HKDF, ChaCha20-Poly1305, optional Ed25519
internal/wire     inner payload framing + outer envelope bytes
internal/frame    stego-frame duration, split, Listen phase search, capacity
internal/stego    LSB matching, keyed positions, constant density, Embedder/Extractor
internal/codec    Codec / Encoder / Decoder / Packet / Residue interfaces
internal/codec/vorbis  Phase 1 codec; residue parse stops before iMDCT
internal/av       cgo ↔ libavformat/libavcodec/libavutil (no stego knowledge)
```

Root must not import C. Only `internal/av` may use cgo. Crypto and framing must not import `av` or `vorbis`.

## Crypto (do not improvise)

Hybrid box, age/NaCl shape, **fresh ephemeral X25519 per stego frame**:

1. ECDH(ephemeral_priv, recipient_pub) → shared
2. HKDF(shared) → AEAD key **and** an independent position-selection key
3. ChaCha20-Poly1305 seal
4. embed `[ephemeral_pub || nonce || ciphertext||tag]`

Inner plaintext (all encrypted):

```
version u8 | type u8 | length u32be | data | optional Ed25519 sig
```

Types: `0x01` text, `0x02` image, `0x03` audio, `0x04` file. Phase 1 uses text only; do not change this layout for later types.

Implemented in `internal/crypto` + `internal/wire`. HKDF-SHA256 salt `mist-v1`, info `mist-aead-v1` / `mist-pos-v1` (32 bytes each). Seal AAD is the ephemeral public key. `Open` / AEAD failures are always `crypto.ErrOpen`. Wire version is `1`; optional Ed25519 sig is exactly 64 bytes after `data`.

## Stego invariants

- Keyed PRNG positions from the HKDF **position** subkey. Never sequential LSBs.
- **Constant density** (`stego.Density`): every encode perturbs the same fraction of eligible high-frequency residues. Short/empty payloads get CSPRNG filler. Presence and absence must have the same footprint.
- **LSB matching** (`±1`), never LSB replacement.
- Extract from the bitstream: Huffman/codebook decode only. No PCM reanalysis.
- No sync marker. `FrameDuration` (8s) is a protocol constant shared by Embed and Listen. Phase search uses AEAD as the oracle.
- Loop the same payload every frame on live streams; new ephemeral key each frame so ciphertext is not periodic.
- Phase 1 owns the full encode path. No embedding into third-party already-encoded files.

Two embed paths (decide after a spike, keep both interfaces):

- `stego.NewBlackBoxEmbedder` — unmodified libvorbisenc as an oracle
- `stego.NewPatchedEmbedder` — vendored libvorbis that exposes residues pre-pack

## CGO (`internal/av`)

Go API in `ops.go`. C ABI in `cgo.h`. `cgo.c` is real libav (pkg-config: `libavformat libavcodec libavutil`). `cgo.go` + `io.go` are `//go:build cgo`. `stub.go` (`//go:build !cgo`) keeps `CGO_ENABLED=0` type-checking; `av.Available()` is false there.

Needed surface (implemented): demux file/URL, demux `io.Reader` via AVIO, mux Ogg to `io.Writer`, decode to PCM, encode Vorbis from PCM. Custom IO uses integer handles + `//export` read/write/seek — never store a Go pointer in C. Map `mist_av_codec_id` to `AV_CODEC_ID_*` in C, not in Go. `ErrAgain` is libav `EAGAIN`.

Sample format numbers already match `AVSampleFormat`. Codec IDs do **not** — they are Mist-local (`CodecIDVorbis = 1`).

Copy packet bytes with `C.CBytes` / `C.GoBytes`. Every `Open*` has a matching `Close`. No finalizers. Ogg custom IO must be `io.Seeker`.

`CGO_ENABLED=1` needs system FFmpeg with libvorbis (`pkg-config` must find the three libs). CI installs `libavformat-dev libavcodec-dev libavutil-dev libvorbis-dev` on both lint and test.

## Status

`internal/crypto` and `internal/wire` are implemented, including `GenerateKeyPair` / `GenerateSigningKeyPair`. `internal/av` talks to real libav. `internal/codec/vorbis` wraps encode/decode and parses identification headers; residue Huffman/codebook decode is still `ErrBadSetup`. Emitter/Catcher I/O, frame, and stego are still stubs.

Suggested order: `vorbis` residue/codebook parse → `stego` extract → `frame` + `Catcher` → embed path spike → `Emitter`.

## Conventions

- `gofmt`. Tabs. MixedCaps. Getters are `Owner`, not `GetOwner`.
- Package names: one short word. No `mist.MistFoo`.
- Doc comments on every exported name, at most five lines. Wrap errors with `%w`.
- Small interfaces at the consumer (`Embedder`, `Codec`). One-method names end in `-er`.
- Tests use `github.com/stretchr/testify/suite`: one `*Suite` struct per `_test.go` file, `TestXxxSuite` entry point, every case a method. Assertions via `s.Equal` / `s.NoError` / `s.Require()`. Same package as the code under test (gocue style).
- Skipped tests are the spec for the next implementation pass — fill them, do not delete them.
- Lint with `make lint` (`golangci-lint run --timeout=5m`). CI is `.github/workflows/ci.yml`, copied from gocue: golangci-lint-action v9 / linter v2.11.4, then `go test -race` with a coverage badge on the `badges` branch. No `.golangci.yml` — default v2 config, same as gocue. CI installs libav + libvorbis so cgo lint and tests compile.

## Non-goals (Phase 1)

Analog D/A/D survival, re-encode robustness, embedding into foreign Ogg files, coercion deniability.

## Ethics

Legitimate confidential communication, watermarking, and research. Not for hiding illegal content.

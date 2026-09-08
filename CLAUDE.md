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
WithLogger(*slog.Logger) CatcherOption   // diagnostics only; never decrypt failures
```

`Listen` / `Embed` `source` is a path, `file://` URL, or `http(s)://` URL. Finite files close the channel on EOF; live streams run until `ctx` is cancelled. Distinguish with `ctx.Err()` after range. A `Listen` called with an already-cancelled context returns a closed channel and no error. `Catcher` and `Emitter` copy keys at construction; do not add setters. Embed methods return `io.ReadCloser` (no `mist.Reader` type); the caller must Close it.

Do not add `ErrNoMessage`. Failed AEAD / partial frame / empty frame are the same silent skip. Exposing "something was there but I could not decrypt" is a side channel. `ErrNoCapacity` is the opposite case and *must* be loud: it means the Emitter embedded into no frame at all, so the caller would otherwise get a silent no-op file. Telling the sender its own request failed leaks nothing to a warden.

A demuxer read already in flight cannot be interrupted, so `Listen` relays the scan through a second channel (`relay`): the channel the caller ranges over closes as soon as `ctx` does, while the scan goroutine ends when its blocked read finally returns. `ListenReader` buffers a non-seekable reader whole (`asSeeker`), so live sources belong in `Listen`, not `ListenReader`.

**Formats.** Output is always Ogg Vorbis — the payload lives in Vorbis residues. Carriers are decoded to PCM and re-encoded, and **the installed FFmpeg decides what input is acceptable**: `AudioInfo.NativeCodecID` carries libav's own `AVCodecID` verbatim and `av.CanDecode` asks libav for a decoder, so Mist keeps no whitelist. `CodecID` stays Mist-local for the codecs Mist reasons about (Vorbis), and is `CodecIDNone` for the rest. An input libav cannot decode is `ErrUnsupportedCodec`. Extraction is Vorbis-only and never re-encodes.

Decoders emit whatever sample format suits them — FLAC gives s32, WAV s16, Vorbis fltp, packed or planar — so `Frame.FloatPlanes` normalises all of them to float planes before the encoder sees them. Never reinterpret frame bytes as float32 directly; that silently produces noise for integer formats.

libav's own logging is set to `AV_LOG_FATAL`: failures already reach Go via return codes and errbuf, and cover art in a normal MP3 otherwise prints warnings into the middle of the CLI's output.

## Layout

```
cmd/mist          cobra CLI: embed / catch, colour output, signal handling
emitter.go        NewEmitter, Embed / EmbedReader / EmbedFile → io.ReadCloser
embed.go          carrier decode, frame loop, mux; FrameCapacity
catcher.go        NewCatcher, Listen / ListenReader / Extract, options
extract.go        scanner: packets → groups → authenticated Results
suite_test.go     shared test fixtures (audioSuite) for both root suites
example/          Godoc examples: keys, Emitter, Catcher
mist.go-level     payload, keys, protocol constants, errors
internal/crypto   X25519 ECDH, HKDF, ChaCha20-Poly1305, optional Ed25519
internal/wire     inner payload framing + outer envelope bytes
internal/frame    stego-frame duration, packet grouping, capacity
internal/stego    LSB matching, keyed positions, constant density, Embedder/Extractor
internal/codec    Codec / Encoder / Decoder / Packet / Residue interfaces
internal/codec/vorbis  Phase 1 codec; residue parse stops before iMDCT
internal/av       cgo ↔ libavformat/libavcodec/libavutil (no stego knowledge)
```

Root must not import C. Only `internal/av` may use cgo. Crypto and framing must not import `av` or `vorbis`.

## Crypto (do not improvise)

Hybrid box, age/NaCl shape, **fresh ephemeral X25519 per stego frame**:

1. ECDH(ephemeral_priv, recipient_pub) → shared
2. HKDF(shared) → AEAD key, an independent position-selection key, and a length mask
3. ChaCha20-Poly1305 seal
4. embed `[ephemeral_pub || masked_len u32be || nonce || ciphertext||tag || filler]`

`masked_len` is the ciphertext length XORed with the length subkey. It tells the
recipient exactly how many bytes to read instead of searching for the end, and it
must stay masked: `PositionSeed` derives from the **public** recipient key, so a
warden who knows that key can locate the bits — a cleartext length there would be
a presence test. Everything after the ciphertext is constant-density filler.

Inner plaintext (all encrypted):

```
version u8 | type u8 | length u32be | data | optional Ed25519 sig
```

Types: `0x01` text, `0x02` image, `0x03` audio, `0x04` file. Phase 1 uses text only; do not change this layout for later types.

Implemented in `internal/crypto` + `internal/wire`. HKDF-SHA256 salt `mist-v1`, info `mist-aead-v1` / `mist-pos-v1` (32 bytes each) and `mist-len-v1` (4 bytes). `PositionSeed(pub, frameIdx)` keys positions from the recipient public key plus the frame index, which the catcher recovers from packet timestamps. Seal AAD is the ephemeral public key. `Open` / AEAD failures are always `crypto.ErrOpen`. Wire version is `1`; optional Ed25519 sig is exactly 64 bytes after `data`.

## Stego invariants

- Keyed PRNG positions from the HKDF **position** subkey. Never sequential LSBs.
- **Constant density** (`stego.Density`): every encode perturbs the same fraction of eligible high-frequency residues. Short/empty payloads get CSPRNG filler. Presence and absence must have the same footprint.
- **LSB matching** (`±1`), never LSB replacement.
- Extract from the bitstream: Huffman/codebook decode only. No PCM reanalysis.
- LSB matching may only move a residue onto a codebook entry of the **same Huffman code length**. Vorbis treats running out of bits mid-partition as "rest is zero", so a different length shifts that boundary and desyncs every symbol after it. Residues with no same-length, opposite-parity sibling are marked `Unflippable` and excluded from `Eligible`.
- No sync marker. `FrameDuration` (8s) is a protocol constant shared by Embed and Listen.
- **Frames are grouped in the packet domain** (`frame.Grouper`, by packet PTS), never by re-slicing PCM. Embed and Listen must derive byte-identical packet sets: one packet's difference changes the eligible count and scrambles every position. Emitter therefore encodes the carrier once, then groups.
- A listener that joins mid-window cannot align, so it logs once and skips that frame. There is no phase search.
- **Audio quality is a hard requirement, and it is easy to destroy.** Three things protect it, all measured on real music against a plain transcode of the same PCM (22.76 dB SDR ceiling): the re-encode bitrate is derived from the source (`encodeBitrate`) rather than fixed; `Density` is 2%; and `DefaultBands` confines embedding above 6 kHz. Together they cost ~0.2 dB. Getting any of them wrong is expensive — a fixed 64 kbps plus 10% density across the full spectrum measured 5.89 dB.
- **Substitute a flipped residue by vector distance, never by index.** Codebook entries n and n+1 dequantize to unrelated spectral vectors, so honouring a bit by nudging the index swaps in a different sound. `substitute` picks the same-length, correct-parity entry whose dequantized vector is nearest the original's; `codebook.vecs` caches those vectors at parse time.
- Entry indices can go negative when `Match` steps below zero. Two's complement already gives the right parity, so read the bit from `want & 1` as-is — forcing it to zero embeds the wrong bit and corrupts recovery intermittently.
- **The payload is embedded once**, in the first frame with room for it. Every other frame is written with CSPRNG filler at the same density — never passed through untouched — so a carrying frame and an empty one leave the same footprint. Nil bits to `stego.Apply` mean "fill with filler". Only a frame with no eligible residues at all is left alone. A carrier where no frame had room is `ErrNoCapacity`.
- Consequence, accepted deliberately: a listener joining a live stream after the carrying frame recovers nothing, and losing that frame loses the message. Re-sealing per frame (fresh ephemeral each time) is what a live-stream mode would restore.
- Phase 1 owns the full encode path. No embedding into third-party already-encoded files.

PCM encode and decode use **only** ffmpeg/libav (`internal/av`). Do not add other Vorbis or Ogg libraries (no libvorbis Go bindings, no jfreymuth/vorbis, no ogg/vorbis encoders). Residue parse and rewrite are in-tree Go bitstream code on top of stock libav packets.

Two embed paths (decide after a spike, keep both interfaces):

- `stego.NewBlackBoxEmbedder` — unmodified libav/libvorbisenc as an oracle
- `stego.NewPatchedEmbedder` — same path until a vendored encoder exposes residues pre-pack

## CGO (`internal/av`)

Go API in `ops.go`. C ABI in `cgo.h`. `cgo.c` is real libav (pkg-config: `libavformat libavcodec libavutil`). `cgo.go` + `io.go` are `//go:build cgo`. `stub.go` (`//go:build !cgo`) keeps `CGO_ENABLED=0` type-checking; `av.Available()` is false there.

Needed surface (implemented): demux file/URL, demux `io.Reader` via AVIO, mux Ogg to `io.Writer`, decode to PCM, encode Vorbis from PCM. Custom IO uses integer handles + `//export` read/write/seek — never store a Go pointer in C. Map `mist_av_codec_id` to `AV_CODEC_ID_*` in C, not in Go. `ErrAgain` is libav `EAGAIN`.

Sample format numbers already match `AVSampleFormat`. Codec IDs do **not** — they are Mist-local (`CodecIDVorbis = 1`).

Copy packet bytes with `C.CBytes` / `C.GoBytes`. Every `Open*` has a matching `Close`. No finalizers. Ogg custom IO must be `io.Seeker`.

`CGO_ENABLED=1` needs system FFmpeg with libvorbis (`pkg-config` must find the three libs). CI installs `libavformat-dev libavcodec-dev libavutil-dev libvorbis-dev` on both lint and test.

## CLI (`cmd/mist`)

```
mist embed --input <file|url> --data <text> [--key pub] [--output out.ogg]
mist catch --input <file|url> --key <priv> [--timeout 30s]
```

Built on cobra. `embed` mints a keypair when `--key` is omitted and writes it
beside the output (`out.pub` / `out.key`, private mode 0600); keys are hex so
they can be inspected. Output defaults to `<input>.stego.ogg`. `catch` prints
each frame as it is recovered and exits non-zero when nothing was.

Signals (`SIGINT`/`SIGTERM`/`SIGHUP`) cancel the command's context via
`signal.NotifyContext`; SIGKILL cannot be trapped. Colour is disabled for
pipes, `TERM=dumb`, `NO_COLOR` and `--no-color`. All output goes through
`printf` in `ui.go` — tests swap the `out` writer — so add rendering there
rather than calling `fmt.Fprint*` directly (errcheck flags bare writes to an
`io.Writer`).

Carrier choice matters: quiet or purely tonal audio yields too few usable
residues and `embed` fails with `ErrNoCapacity`. Test fixtures use noise, not
a pure sine, for this reason.

## Make targets

`build` (to `bin/mist`), `embed`, `catch`, `keys`, `test`, `lint`, `clean`.
`embed` and `catch` take `INPUT=`, `DATA=`, `OUTPUT=`, `KEY=`, `TIMEOUT=`.
`keys` mints an X25519 pair with OpenSSL and writes the raw 32 bytes as hex —
OpenSSL emits PKCS#8/SPKI DER, whose last 32 bytes are the key — so its output
interoperates with `crypto/ecdh` and the CLI's own hex format.

## Status

Phase 1 is feature-complete end to end, with a `cmd/mist` CLI over it. All internal packages are implemented; `Emitter` embeds text into every whole frame and `Catcher` recovers it via `Listen` / `ListenReader` / `Extract`, including multi-frame carriers. Remaining Phase 1 gaps: `FrameCapacity()` is a heuristic that ignores the flippability ratio and so over-estimates real capacity (the true limit is enforced at `Embed` time); `Embed` over `http(s)` buffers a finite file rather than streaming a live source; a scan goroutine parked in a blocking libav read outlives its context until that read returns.

## Design principles

Follow these without being asked; they are why review comments get made.

- **Single responsibility.** One package owns one idea: `frame` owns what a stego frame *is*, `stego` owns positions and density, `crypto` owns keys, `wire` owns byte layout, `av` owns libav. When two layers must agree on something (frame boundaries), give them one shared function rather than two implementations that happen to match.
- **Policy lives above mechanism.** `internal/av` reports facts (codec is `NONE`); the `mist` layer decides that means `ErrUnsupportedCodec`. Do not push stego or protocol policy into the bindings.
- **Depend on small consumer-side interfaces** (`rewriter`, `Embedder`, `Codec`), not on concrete types. Inject collaborators — the logger arrives via `WithLogger`, defaulted to discard, never grabbed from a global.
- **Delete rather than deprecate.** No compatibility shims, no `_ = unused`, no renamed-but-kept helpers. If it is dead, remove it.
- **Prefer fewer moving parts.** Encode once and group, rather than slicing and re-encoding per frame. A magic constant that needs a paragraph to justify is usually a design smell — the trailing-crumb threshold disappeared when grouping moved to the packet domain.
- **Comment the why, never the what.** Most functions need none. Comment invariants a reader would otherwise break (why code length must be preserved, why the length field is masked).
- **Errors:** wrap with `%w`, sentinels in `errors.go`, and never leak "something was there but I couldn't read it" to a caller.

## Conventions

- `gofmt`. Tabs. MixedCaps. Getters are `Owner`, not `GetOwner`.
- Package names: one short word. No `mist.MistFoo`.
- Doc comments on every exported name, at most five lines. Wrap errors with `%w`.
- Small interfaces at the consumer (`Embedder`, `Codec`). One-method names end in `-er`.
- Tests use `github.com/stretchr/testify/suite`: one `*Suite` struct per `_test.go` file, `TestXxxSuite` entry point, every case a method. Assertions via `s.Equal` / `s.NoError` / `s.Require()`. Same package as the code under test (gocue style).
- **Table-driven cases inside suite methods**: a `tests := []struct{ title string; ... }` slice plus `s.Run(tc.title, ...)`. Shared fixtures go on an embedded base suite (`audioSuite`), not copied between files.
- Test the invariant, not the implementation: assert that a wrong key yields nothing, that every whole frame carries the payload, that the length field differs for equal payloads.
- Skipped tests are the spec for the next implementation pass — fill them, do not delete them.
- Lint with `make lint` (`golangci-lint run --timeout=5m`). CI is `.github/workflows/ci.yml`, copied from gocue: golangci-lint-action v9 / linter v2.11.4, then `go test -race` with a coverage badge on the `badges` branch. No `.golangci.yml` — default v2 config, same as gocue. CI installs libav + libvorbis so cgo lint and tests compile.

## Non-goals (Phase 1)

Analog D/A/D survival, re-encode robustness, embedding into foreign Ogg files, coercion deniability.

## Ethics

Legitimate confidential communication, watermarking, and research. Not for hiding illegal content.

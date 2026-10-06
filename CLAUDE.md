# Mist

Asymmetric-key audio steganography for Go. Phase 1: text payloads in Ogg Vorbis
or any lossless format FFmpeg can write.

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
Formats() []Format                       // what this FFmpeg build can write
LookupFormat(name, codec string) (Format, error)
EstimateCapacity(ctx, source, format, codec string) (Capacity, error)
WithSenderAuth(senderPriv []byte) EmitterOption
WithFormat(name string) EmitterOption    // container name or output path; ffmpeg's -f
WithCodec(name string) EmitterOption     // encoder override; ffmpeg's -c:a
WithMaxRetries(n int) CatcherOption
WithBackoff(d time.Duration) CatcherOption
WithLogger(*slog.Logger) CatcherOption   // diagnostics only; never decrypt failures
```

`Listen` / `Embed` `source` is a path, `file://` URL, or `http(s)://` URL. Finite files close the channel on EOF; live streams run until `ctx` is cancelled. Distinguish with `ctx.Err()` after range. A `Listen` called with an already-cancelled context returns a closed channel and no error. `Catcher` and `Emitter` copy keys at construction; do not add setters. Embed methods return `io.ReadCloser` (no `mist.Reader` type); the caller must Close it.

Do not add `ErrNoMessage`. Failed AEAD / partial frame / empty frame are the same silent skip. Exposing "something was there but I could not decrypt" is a side channel. `ErrNoCapacity` is the opposite case and *must* be loud: it means the Emitter embedded into no frame at all, so the caller would otherwise get a silent no-op file. Telling the sender its own request failed leaks nothing to a warden.

A demuxer read already in flight cannot be interrupted, so `Listen` relays the scan through a second channel (`relay`): the channel the caller ranges over closes as soon as `ctx` does, while the scan goroutine ends when its blocked read finally returns. `ListenReader` buffers a non-seekable reader whole (`asSeeker`), so live sources belong in `Listen`, not `ListenReader`.

**Formats.** Carriers are decoded to PCM and re-encoded, and **the installed FFmpeg decides what is acceptable on both sides**: `AudioInfo.NativeCodecID` carries libav's own `AVCodecID` verbatim, `av.CanDecode` asks libav for a decoder, and `av.Lossless` asks the codec descriptor for `AV_CODEC_PROP_LOSSLESS`, so Mist keeps no whitelist. `CodecID` stays Mist-local for the codecs Mist reasons about (Vorbis), and is `CodecIDNone` for the rest. An input libav cannot decode is `ErrUnsupportedCodec`.

**Two output domains, one protocol.** Vorbis carries bits in quantized residues; every lossless codec carries them in PCM sample LSBs, because a lossless encoder hands those samples back bit for bit. `av.FindFormat` resolves a target the way ffmpeg's command line does — container short name, output path extension, or encoder name, with an optional codec override (`-f` / `-c:a`) — and reports `Lossless`. The `mist` layer turns that into policy: lossless → sample domain, Vorbis → residue domain, any other lossy codec → `ErrUnsupportedCodec`, resolved in `NewEmitter` before a carrier is touched. Extraction picks the same way from the demuxed stream's own codec id, so `catch` needs no flag.

The lossless path shares everything that defines the protocol — `FrameDuration` windows, HKDF position keys, `stego.Density`, payload in the first frame with room and CSPRNG filler in every other. Only the carrier differs, behind `stego.carrier` (`Len`/`At`/`Cost`/`Flip`). Do not give it its own density, its own framing, its own code or its own crypto.

Sample embedding rides on the **integer grid the encoder quantizes to** (`av.SampleScale`). Before embedding, `Encoder.Snap` moves every carrier sample to the integer the encoder will store for it, computed by libav itself: libswresample's own conversion (the one `send_flt` uses, with this platform's rounding) plus the encoder's truncation of an s32 sample to its top 24 bits (`gridShift`). It cuts on the decoded frame lengths, because ffmpeg resamples one frame at a time and that frame's remainder rounds differently from the rest. Mist does no float→integer rounding of its own on the embed path, so everything but the ±1 changes is what a plain ffmpeg encode writes. Both sides then read `round(f*scale)` of on-grid values, so a written LSB survives encode and decode. `ApplySamples` still writes **every** sample in the window back onto that grid: the receiver reads the LSB of every covered sample, not just the changed ones. The grid is capped at 24 bits because float32 carries 24 mantissa bits and the pipeline is float throughout; a finer grid would not round-trip. `Samples.Set` clamps by **two**, not one, so the encoder's own clipping cannot take the embedded bit with it. Position k means channel `k%ch`, sample `Off+k/ch` — interleaved order, stated once in `stego.Samples` and relied on by both `sampleFrames` (embed) and `windower` (listen).

Decoders emit whatever sample format suits them — FLAC gives s32, WAV s16, Vorbis fltp, packed or planar — so `Frame.FloatPlanes` normalises all of them to float planes before the encoder sees them. Never reinterpret frame bytes as float32 directly; that silently produces noise for integer formats.

libav's own logging defaults to `AV_LOG_FATAL`: failures already reach Go via return codes and errbuf, and cover art in a normal MP3 otherwise prints warnings into the middle of the CLI's output. `MIST_AV_LOG` (`quiet`…`trace`) raises it — the only way to see what the codec layer is doing, and what `make test` sets.

## Layout

```
cmd/mist          cobra CLI: embed / catch / formats / estimate, colour output, signal handling
emitter.go        NewEmitter, Embed / EmbedReader / EmbedFile → io.ReadCloser
embed.go          carrier decode, encoder open, mux; FrameCapacity
lossless.go       sample-domain embed + windower (the lossless half of embed/extract)
format.go         Format, Formats, LookupFormat: which targets are writable
span.go           planChunks/planSpan: split a payload across frames when it doesn't fit one
estimate.go       EstimateCapacity: real per-frame and total room, without embedding anything
catcher.go        NewCatcher, Listen / ListenReader / Extract, options
extract.go        scanner interface + residueScanner / sampleScanner, shared opener
suite_test.go     shared test fixtures (audioSuite) for the root suites
harness_*_test.go `make harness` (build tag harness): corpus, measurement, report,
                  key-aware warden (harness_key_test.go), CNN export
tools/cnn_warden  Python CNN warden over the harness export; README.md
example/          Godoc examples: keys, Emitter, Catcher
mist.go-level     payload, keys, protocol constants, errors
internal/crypto   X25519 ECDH, HKDF, ChaCha20-Poly1305, optional Ed25519
internal/wire     inner payload framing + outer envelope bytes
internal/frame    stego-frame duration, packet grouping, capacity
internal/stego    syndrome-trellis code (stc.go), keyed order, constant rate; bits.go
                  holds the carrier-agnostic core, sample.go the PCM carrier
internal/codec    Codec / Encoder / Decoder / Packet / Residue interfaces
internal/codec/vorbis  residue parse/rewrite; stops before iMDCT
internal/av       cgo ↔ libavformat/libavcodec/libavutil/libswresample (no stego knowledge);
                  pcm.go is the shared PCM↔packet plumbing both domains encode with
internal/steganalysis  the warden: chi-square, SPA, RS, HCF-COM, features,
                  second-difference Markov features, cross-validated logistic
                  classifier, AUC; README.md explains each
internal/quality  lag-aligned SDR and segmental SNR, ViSQOL/PEAQ wrappers
internal/dsp      FFT shared by steganalysis and quality
```

Root must not import C. Only `internal/av` may use cgo. Crypto and framing must not import `av` or `vorbis`.

## Crypto (do not improvise)

Hybrid box, age/NaCl shape, **fresh ephemeral X25519 per stego frame**:

1. Draw an ephemeral X25519 key that has an Elligator 2 representative (about two tries); ECDH(ephemeral_priv, recipient_pub) → shared
2. HKDF(shared) → AEAD key, an independent position-selection key, and a length mask
3. ChaCha20-Poly1305 seal
4. embed `[ephemeral_rep || masked_len u32be || nonce || ciphertext||tag || filler]`

`masked_len` is the ciphertext length XORed with the length subkey. It tells the
recipient exactly how many bytes to read instead of searching for the end, and it
must stay masked: `PositionSeed` derives from the **public** recipient key, so a
warden who knows that key can locate the bits — a cleartext length there would be
a presence test. Everything after the ciphertext is constant-density filler.

`ephemeral_rep` is the Elligator 2 representative of the ephemeral key (`internal/crypto/elligator.go`), never the key. A public key in the clear has its top bit clear and lies in the prime-order subgroup, so a warden holding only the recipient public key could recover the first frame's bits, read those 32 bytes and test them: the harness's key-aware warden scored AUC 1.000 on every carrier before this. The receiver maps the representative back to the u-coordinate, which is all X25519 uses. The two top bits are random padding. Never put the raw ephemeral key, or any other structured value, in the envelope.

Inner plaintext (all encrypted):

```
version u8 | type u8 | length u32be | data | optional Ed25519 sig
```

Types: `0x01` text, `0x02` image, `0x03` audio, `0x04` file. Phase 1 uses text only; do not change this layout for later types.

Implemented in `internal/crypto` + `internal/wire`. HKDF-SHA256 salt `mist-v1`, info `mist-aead-v1` / `mist-pos-v1` (32 bytes each) and `mist-len-v1` (4 bytes). `PositionSeed(pub, frameIdx)` keys positions from the recipient public key plus the frame index, which the catcher recovers from packet timestamps. Seal AAD is the ephemeral representative as sent. `Open` / AEAD failures are always `crypto.ErrOpen`. Wire version is `1`; optional Ed25519 sig is exactly 64 bytes after `data`.

A marshaled payload that does not fit one frame is split into chunks, each independently sealed (its own ephemeral key, its own frame). `wire.MarshalSpanStart(totalLen, chunk)` frames the first one — one magic byte plus a `u32be` total length, never a valid `Payload.Version` so it can't collide with an unspanned payload — and `wire.MarshalSpanContinue(chunk)` frames every one after it with a single, different magic byte. This framing exists only when spanning is actually used: a payload that fits one frame keeps `wire.Payload`'s layout completely unmodified, so the common case pays none of it and every file already embedded stays readable. See `span.go`.

## Stego invariants

These hold in both domains. Where one is Vorbis-only it says so.

- **Syndrome-trellis code** (`stc.go`, Filler–Judas–Fridrich). A frame of n eligible values carries `m = slots(n)` bits through the first `m*(n/m)` of them in a keyed order (`Selector`, ChaCha20 keyed by the HKDF **position** subkey); the submatrix is drawn from the same key. The receiver computes the syndrome of every covered LSB, so `Len` and `At` must read the same before and after embedding. At the 1% rate this changes ~0.14 values per bit against 0.5 for one bit per position.
- **Constant rate** (`stego.Density`): every encode carries the same number of bits per eligible value. Short/empty payloads get CSPRNG filler. Presence and absence must have the same footprint.
- A trellis **tie goes to leaving the cover bit alone**. Breaking it towards a fixed stego bit flips odd values more often than even ones — a parity bias.
- Every change is **±1** (a Vorbis flip lands on the nearest opposite-parity entry), never LSB replacement. Costs (`carrier.Cost`) are the sender's alone: the receiver needs none of them.
- Vorbis: extract from the bitstream — Huffman/codebook decode only, no PCM reanalysis. Lossless: extract from decoded PCM, which is the *same* PCM Embed wrote, not an analysis of it.
- Vorbis: a flip may only move a residue onto a codebook entry of the **same Huffman code length**. Vorbis treats running out of bits mid-partition as "rest is zero", so a different length shifts that boundary and desyncs every symbol after it. Residues with no same-length, opposite-parity sibling are marked `Unflippable` and excluded from `Eligible`. `codec.Residue.FlipCost` is the squared vector distance to the nearest same-length substitute (`codebook.buildFlips`); past `stego.maxFlipCost` a flip is priced `wetCost`. `RankCost` is that distance divided by the entry's energy and raised when few substitutes exist, and that is what the trellis ranks. A symbol with no legal substitute stays out of the carrier: putting it in would change the positions the receiver walks.
- No sync marker. `FrameDuration` (8s) is a protocol constant shared by Embed and Listen.
- **Vorbis frames are grouped in the packet domain** (`frame.Grouper`, by packet PTS), never by re-slicing PCM. Embed and Listen must derive byte-identical packet sets: one packet's difference changes the eligible count and scrambles every position. Emitter therefore encodes the carrier once, then groups.
- **Embed groups from a canonicalized packet stream, not the encoder's own.** Ogg stores a granule position per *page*, not a timestamp per packet, so a demuxer reconstructs each packet's PTS from that using standard Vorbis block-size accounting — and at a genuine block-size transition, that reconstruction can differ from what the encoder itself reported by one packet's duration, shifting a packet into a different frame on read-back than Embed used to group it. `canonicalPackets` (embed.go) closes this by muxing the encoder's packets once, unrewritten, and demuxing them straight back before grouping, so grouping runs on the exact packet stream — same timestamps — any reader, including Listen, will reconstruct. Skipping this reintroduces an intermittent, silent mismatch that gets far more likely once a payload spans several frames, since every frame boundary in the span has to agree.
- **Lossless frames are windows of samples**, cut at `frame.Params.Samples()` — the same function the grouper uses, so the two domains cannot disagree about where a frame starts. `sampleFrames` cuts them for Embed and `windower` reassembles them for Listen; a window out by one sample scrambles everything in it, so those two are tested against each other.
- **Output is exactly as long as the carrier.** `splitPCM` sends the last encoder window short and never zero-pads it; libav pads a final frame itself for an encoder that needs it, the same way the `ffmpeg` CLI does. Padding in Mist made every lossless output a whole number of encoder blocks ending in digital zeros, a trace a plain encode does not leave, and `make harness` checks length and zero tail against ffmpeg.
- A listener that joins mid-window cannot align, so it logs once and skips that frame. There is no phase search.
- **Audio quality is a hard requirement, and it is easy to destroy.** For Vorbis, three things protect it: the re-encode keeps the source's quality rather than a fixed rate; `Density` is 1%; and `DefaultBands` confines embedding above 6 kHz. `openVorbis` runs libvorbis in VBR at the whole `-q:a` level whose nominal rate is nearest the source bitrate times its codec's `efficiency` (MP3 0.75, AAC 1.0, Opus 1.3, anything else 1.0; the higher level on a tie), and at q8 for a lossless or rateless source — so the header reads like an ordinary `ffmpeg -q:a N` file instead of an odd ABR target. libvorbis reports each level's nominal rate itself; Mist keeps no table of them. With one bit per position the embedding cost ~0.2 dB SDR; through the trellis code, which also prices each flip by `FlipCost`, the harness measured under 0.02 dB (both under the earlier source×1.5 ABR rule — re-run `make harness` after changing the rate rule). Capacity follows the rate: a 128 kbps MP3 carrier now encodes at q2 and holds roughly a third of what the old 192 kbps target did. Getting any of them wrong is expensive — a fixed 64 kbps plus 10% density across the full spectrum measured 5.89 dB. A lossless target has none of these problems and measures ~85 dB SDR against its carrier: the only change is ±1 on about 0.14% of the covered samples (a 1% rate at ~0.14 changes per bit), and there is no transcode loss underneath it. Never pass a bitrate to a lossless encoder — it decides its own size and complains if told otherwise.
- **Lossless embedding skips silence and nothing else.** Silence is judged per channel: a run of at least `silenceRun` (32) samples all within ±`silenceFloor` (2) is not in the carrier, and neither is a quiet run of any length touching either edge of the window, which may be the tail of a longer one. A plain encoder never puts ±1 into digital silence or the end padding, so Mist must not. The receiver re-derives the same runs from the stego file, so no ±1 may create, lengthen or join a silent run: a sample at exactly `silenceFloor+1` steps towards zero only when `mayQuiet` finds the quiet run it would join stays shorter than `silenceRun` and clear of both window edges. Flips are applied one at a time, so each check sees the samples already changed.
- Lossless **where** a change lands follows a sender-only cost: growth of the local first and second difference, discounted by the neighbourhood residual (a tone costs more than noise), plus stereo, clipping, and silence-edge terms. A floor and a ratio cap keep the cheap set from becoming a selection channel, and a dither from the position key keeps it from being a function of the waveform alone. **Direction** stays the frame histogram, √h(v+1) against √h(v−1), with that weight discounted by the residual of each step, so a close call does not walk the frame toward its peak. The sample list itself is unchanged. Random ±1 adds exactly 1 to Σv² per change; the histogram term is what keeps that drift small.
- Vorbis: **substitute a flipped residue by vector distance, never by index.** Codebook entries n and n+1 dequantize to unrelated spectral vectors, so honouring a bit by nudging the index swaps in a different sound. `substitute` picks the same-length, correct-parity entry whose dequantized vector is nearest the original's; `codebook.vecs` caches those vectors at parse time.
- Entry indices can go negative (`-3 ^ 1` is `-4`). Two's complement already gives the right parity, so read the bit from `want & 1` as-is — forcing it to zero embeds the wrong bit and corrupts recovery intermittently.
- **The payload is embedded once**, in the first frame with room for it — or, when it doesn't fit any one frame, spread across as many *consecutive* usable frames as it needs (`planChunks`/`planSpan` in span.go), each sealed independently with its own ephemeral key. Every frame not carrying a chunk this round is written with CSPRNG filler at the same density — never passed through untouched — so a carrying frame and an empty one leave the same footprint. Nil bits to `stego.Apply`/`ApplySamples` mean "fill with filler". Only a frame with no eligible residues at all is left alone. A payload that fits nowhere, even split across every usable frame, is `ErrNoCapacity`.
- Consequence, accepted deliberately: a listener joining a live stream after the carrying frame (or, for a spanned payload, after its first chunk) recovers nothing, and losing that frame — or any one frame of a span — loses the message. Re-sealing per frame (fresh ephemeral each time) is what a live-stream mode would restore.
- Phase 1 owns the full encode path. No embedding into third-party already-encoded files.
- **Never widen the sample grid past 24 bits** to chase a 32-bit format. The whole pipeline is float32; the LSB would not survive.

`av.Packet` carries `AV_PKT_DATA_SKIP_SAMPLES` (`SkipStart`, `SkipEnd`) from the demuxer to the decoder, which is what drops an MP3's encoder delay and padding, an Opus pre-skip and an Ogg page's trailing samples exactly as the ffmpeg command line does. Copy only bytes and timestamps and a lossy carrier decodes longer than a plain ffmpeg encode of it; `TestLengthMatchesFFmpeg` guards this.

PCM encode and decode use **only** ffmpeg/libav (`internal/av`). Do not add other Vorbis or Ogg libraries (no libvorbis Go bindings, no jfreymuth/vorbis, no ogg/vorbis encoders). Residue parse and rewrite are in-tree Go bitstream code on top of stock libav packets.

Two embed paths (decide after a spike, keep both interfaces):

- `stego.NewBlackBoxEmbedder` — unmodified libav/libvorbisenc as an oracle
- `stego.NewPatchedEmbedder` — same path until a vendored encoder exposes residues pre-pack

## CGO (`internal/av`)

Go API in `ops.go`. C ABI in `cgo.h`. `cgo.c` is real libav (pkg-config: `libavformat libavcodec libavutil`). `cgo.go` + `io.go` are `//go:build cgo`. `stub.go` (`//go:build !cgo`) keeps `CGO_ENABLED=0` type-checking; `av.Available()` is false there.

Needed surface (implemented): demux file/URL, demux `io.Reader` via AVIO, mux any container to `io.Writer`, decode to PCM, encode from PCM, resolve and list output formats. Custom IO uses integer handles + `//export` read/write/seek — never store a Go pointer in C. Map `mist_av_codec_id` to `AV_CODEC_ID_*` in C, not in Go. `ErrAgain` is libav `EAGAIN`.

Sample format numbers already match `AVSampleFormat`. Codec IDs do **not** — they are Mist-local (`CodecIDVorbis = 1`). `AudioInfo.Container` names the muxer (empty means `ogg`); it is preserved across `avEncInfo` so `Encoder.Info()` alone is enough to open a muxer.

The encoder picks its sample format the way the ffmpeg command line does: `fmt_score` in `cgo.c` is libavfilter's `get_fmt_score` against the carrier's own format, so a 16-bit source stays s16 and a 24-bit or float source goes to s32 wherever the encoder takes it. `bits_per_raw_sample` is the source's, capped at the format's width (ffmpeg's `enc_open`); every lossless encoder then settles s32 on 24 bits, which is why `SampleScale` stops there. `avcodec_get_supported_config` needs libavcodec ≥ 61.13.100; the `codec->sample_fmts` fallback is what keeps older CI builds compiling, since that field is gone in FFmpeg 8+. Float planes reach the encoder through **libswresample** (`swr` on the encoder, FLTP → its format), never a hand-written conversion; `mist_av_encoder_convert` runs the same conversion without sending, for `Snap`. The muxer writes the encoder's `bits_per_raw_sample` when it has one.

The muxer sets `bits_per_coded_sample` for raw-PCM containers and `bits_per_raw_sample` for codecs that store depth in their own header; TTA writes an unreadable file without the latter.

Copy packet bytes with `C.CBytes` / `C.GoBytes`. Every `Open*` has a matching `Close`. No finalizers. Ogg custom IO must be `io.Seeker`.

`CGO_ENABLED=1` needs system FFmpeg with libvorbis (`pkg-config` must find the four libs). CI installs `libavformat-dev libavcodec-dev libavutil-dev libswresample-dev libvorbis-dev` on both lint and test.

## CLI (`cmd/mist`)

```
mist embed --input <file|url> --data <text> [--key pub] [--output out.flac] [--out-codec alac]
mist catch --input <file|url> --key <priv> [--timeout 30s]
mist formats
```

Built on cobra. `embed` mints a keypair when `--key` is omitted and writes it
beside the output (`out.pub` / `out.key`, private mode 0600); keys are hex so
they can be inspected. `catch` prints each frame as it is recovered and exits
non-zero when nothing was, and needs no format flag — the stream says what it is.

**The output format follows ffmpeg's contract**: `--output`'s extension picks the
container, `--out-codec` overrides the encoder when a container holds more than
one (`alac` in an `.m4a`). `--out-codec` takes either name libav knows a codec by,
the encoder's or the codec's (`dca` or `dts`), because `formats` prints the latter.
Output defaults to `<input>.stego.ogg`. An unwritable target fails in
`LookupFormat` before the carrier is read. Source metadata is copied the
way ffmpeg copies it: the file's tags, then the mapped audio stream's
tags, dropping `creation_time`, `company_name`, `product_name`, and
`product_version`, with the encoder string left as the one the CLI writes.

Signals (`SIGINT`/`SIGTERM`/`SIGHUP`) cancel the command's context via
`signal.NotifyContext`; SIGKILL cannot be trapped. Colour is disabled for
pipes, `TERM=dumb`, `NO_COLOR` and `--no-color`. All output goes through
`printf` in `ui.go` — tests swap the `out` writer — so add rendering there
rather than calling `fmt.Fprint*` directly (errcheck flags bare writes to an
`io.Writer`).

Carrier choice matters for a **Vorbis** output: quiet or purely tonal audio
yields too few usable residues and `embed` fails with `ErrNoCapacity`. Test
fixtures use noise, not a pure sine, for this reason. A lossless output has
orders of magnitude more room and only fails on a carrier that is too short.

## Make targets

`build` (to `bin/mist`), `embed`, `catch`, `formats`, `keys`, `test`, `test-quiet`, `harness`, `cnn-warden`, `lint`, `clean`.
`test` is the loud one: `-v -race -count=1 -cover`, `GOTRACEBACK=all`, and
`MIST_AV_LOG=$(LOG)` so libav talks too; `test-quiet` is the same run without
the per-test output. 
`embed` and `catch` take `INPUT=`, `DATA=`, `OUTPUT=`, `CODEC=`, `KEY=`, `TIMEOUT=`.
`keys` mints an X25519 pair with OpenSSL and writes the raw 32 bytes as hex —
OpenSSL emits PKCS#8/SPKI DER, whose last 32 bytes are the key — so its output
interoperates with `crypto/ecdh` and the CLI's own hex format.
`harness` takes `CORPUS=`, `CORPUS_MANIFEST=`, `CORPUS_NAME=`, `FORMATS=` (a
list, or `all`), `JOBS=`, `BASELINE=`, `HARNESS_OUT=`, and tool paths
(`FFMPEG`, `FFPROBE`, `VISQOL`, `PEAQ`, `GIT`, `PKG_CONFIG`). It writes `report.md`,
`report.json`, `manifest.json` and `scores.json` under `HARNESS_OUT`. Reports
must use public carrier ids: pass a corpus manifest, or accept anonymous
`carrier-NNNN` ids. A manifest carrier with `split: sealed` is hashed and not
scored; `MIST_HARNESS_UNSEAL=1` scores that split instead and leaves the
development carriers out, so one run never trains on both. The fingerprint verdict
compares Mist with the canonical workflow: Vorbis at the `-q:a` level Mist chose,
lossless at ffmpeg's defaults. Default ffmpeg is a labeled second threat model.
`MIST_FFPROBE` (`FFPROBE`) supplies the audio-blind metadata warden. The confirmatory family is frozen: hcf-com, classifier, markov, key-aware, rich, selection (schema 6). `make corpus CORPUS_OUT=`
writes a generated PCM set plus that holdout, and the directory must sit outside
the checkout. Never commit local audio paths. It is never part of
`make test` or CI: it needs a corpus and minutes, and its numbers are read,
not asserted.
`cnn-warden` runs the harness with `MIST_HARNESS_EXPORT` (`CNN_EXPORT`),
trains `CNN_WARDEN` (default `tools/cnn_warden/train.py`) with `PYTHON`,
writes `CNN_OUT`, and runs the harness again so the report shows the result.
The analysis packages must not import `av`; the harness, in the root package, does
the decoding and hands them `[]int32` values and `[][]float32` planes.

## Status

Phase 1 is feature-complete end to end, with a `cmd/mist` CLI over it. All internal packages are implemented; `Emitter` embeds text and `Catcher` recovers it via `Listen` / `ListenReader` / `Extract`. Output is Ogg Vorbis or any lossless codec the installed FFmpeg can encode — verified end to end for FLAC, WAV, ALAC, WavPack, TTA, AIFF and CAF. `make harness` measures detectability and quality per output format, against a plain ffmpeg encode of the same carrier. A historical 84-carrier run (see `REPORT.md`; not bound to a Phase 0 manifest) found no detector's aggregate file AUC above 0.55 for FLAC or WAV, while the key-aware warden sat at chance. That is not undetectable: scored within one kind of audio, the classifier separated several categories. Bind any later claim to that run's commit, corpus name and independent lineage count. Embedding costs under 0.05 dB of SDR beyond a plain re-encode, and lossless output leaves digital silence untouched. A warden holding the original carrier can diff against it, which the harness cannot measure.

Remaining Phase 1 gaps: `FrameCapacity()` is a heuristic for the Vorbis path only — it ignores the flippability ratio and so over-estimates real capacity (the true limit is enforced at `Embed` time), and it does not describe a lossless target at all, which holds far more (`EstimateCapacity`, and the `mist estimate` command built on it, report the real number instead, at the cost of decoding and re-encoding the carrier); `Embed` over `http(s)` buffers a finite file rather than streaming a live source; a scan goroutine parked in a blocking libav read outlives its context until that read returns; the lossless path buffers the whole carrier before encoding, so `Embed` is not yet streaming there either.

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

Analog D/A/D survival, re-encode robustness, embedding into foreign already-encoded files, coercion deniability. **Known-cover attack**: a warden who holds the original carrier, or can get it (public music), re-encodes it and diffs; the embedded ±1 changes are exactly what that diff finds, in both domains, because re-encoding is deterministic. Nothing in the harness can measure this, and no embedding that changes the file can prevent it. A lossless stego file survives copying byte for byte but not re-encoding: transcoding it to MP3, or even back to Vorbis, destroys the message.

## Ethics

Legitimate confidential communication, watermarking, and research. Not for hiding illegal content.

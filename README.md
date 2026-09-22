# mist

[![CI](https://github.com/iSerganov/mist/actions/workflows/ci.yml/badge.svg)](https://github.com/iSerganov/mist/actions/workflows/ci.yml)
[![Coverage](https://img.shields.io/endpoint?url=https://raw.githubusercontent.com/iSerganov/mist/badges/coverage.json)](https://github.com/iSerganov/mist/actions/workflows/ci.yml)
[![Go Version](https://img.shields.io/badge/Go-1.27+-blue.svg)](https://golang.org/)
[![License](https://img.shields.io/badge/License-MIT-green.svg)](LICENSE)

**Asymmetric-key audio steganography for Go — a library and a command-line tool.**

A sender hides a message using nothing but the recipient's public key. Only the
matching private key can read it back. The payload lives inside the audio
itself — not in tags, comments, or container metadata — so a plain digital
recording of a stream carries the message with it.

Output is Ogg Vorbis or any lossless format your FFmpeg can write: FLAC, WAV,
ALAC, WavPack, TTA, AIFF, CAF and the rest. The extension of `--output` picks
it, exactly as it does for `ffmpeg`.

Mist is built for learning: a place to explore steganography, hybrid
cryptography, and bitstream-level audio embedding in a codebase small enough to
read in an afternoon. Contributions and design discussion are very welcome.

---

## Contents

- [mist](#mist)
  - [Contents](#contents)
  - [How it works](#how-it-works)
  - [Requirements](#requirements)
  - [Install](#install)
  - [Command line](#command-line)
    - [embed](#embed)
    - [catch](#catch)
    - [formats](#formats)
    - [estimate](#estimate)
    - [Interrupting a run](#interrupting-a-run)
  - [Library](#library)
    - [Emitter](#emitter)
    - [Catcher](#catcher)
    - [Options](#options)
    - [Errors](#errors)
  - [Choosing a carrier](#choosing-a-carrier)
    - [Audio quality](#audio-quality)
  - [Supported formats](#supported-formats)
  - [Status and limitations](#status-and-limitations)
  - [Security notes](#security-notes)
  - [Development](#development)
  - [Ethics](#ethics)
  - [Contributing](#contributing)

---

## How it works

**The message is encrypted before it is hidden.** Mist uses a hybrid box in the
shape of `age` or NaCl's `box`: a freshly generated ephemeral X25519 key agrees a
secret with the recipient's public key, HKDF-SHA256 splits that secret into an
AEAD key, a position-selection key and a length mask, and ChaCha20-Poly1305 seals
the payload. Because that sender key is ephemeral and generated per message, a
sender needs no long-term identity at all. An optional Ed25519 signature can be
added when you *do* want the recipient to know who sent it.

**The hiding place is the audio, and where exactly depends on the format.**
Mist always decodes the carrier to PCM and re-encodes it. For Ogg Vorbis it
then modifies the quantized residue values inside the resulting packets;
extraction is the mirror image and is far cheaper, Huffman-decoding the
bitstream and stopping before the inverse MDCT, so it never reanalyses PCM.
For a lossless format there is no bitstream surgery at all: a lossless encoder
returns its input samples bit for bit, so the bits go straight into the PCM
before encoding and are read back out of the decoded samples.

Everything else is identical either way. Which values get touched is chosen by
a keyed PRNG, never sequentially, and every encode perturbs the same fraction
of them whether or not there is a real message — short payloads are padded with
CSPRNG filler. Presence and absence are meant to leave the same statistical
footprint.

**Audio is divided into frames, and the message goes into one of them.** The
carrier is split into self-contained 8-second frames, and the sealed message is
written into the first frame with room for it. Every other frame is filled with
CSPRNG bytes at exactly the same density, so the frame carrying the message is
statistically indistinguishable from the ones carrying nothing — the padding is
not decoration, it is what stops the payload's location being obvious. There is
no sync marker either: both sides derive frame boundaries from the timestamps
already present in the stream.

One consequence is worth knowing: because the message lives in a single frame,
a listener who joins a live broadcast after that frame has passed recovers
nothing, and a recording that loses that frame loses the message.

## Requirements

- **Go 1.27+**
- **A C compiler and system FFmpeg**, discoverable via `pkg-config`. All codec
  I/O goes through libav; Mist deliberately adds no other Vorbis or Ogg library.

```bash
# macOS
brew install ffmpeg pkg-config

# Debian / Ubuntu
sudo apt-get install pkg-config libavformat-dev libavcodec-dev libavutil-dev libvorbis-dev
```

Building with `CGO_ENABLED=0` still type-checks against a pure-Go stub, which
keeps editors and vet happy, but every encode and decode then reports
unimplemented. Real use needs cgo.

## Install

```bash
# library
go get github.com/iSerganov/mist

# command-line tool
go install github.com/iSerganov/mist/cmd/mist@latest
```

## Command line

The `mist` binary has several commands. Colour is enabled for terminals and turned
off automatically for pipes, `TERM=dumb`, and `NO_COLOR`; `--no-color` forces it
off.

### embed

```bash
mist embed --input song.ogg --data "the eagle lands at dawn"
```

```
  mist · embed

  carrier     song.ogg
  payload     23 bytes  text
  output      song.stego.ogg  vorbis · lossy, bits in the residues
  recipient   new keypair  no --key given

  → re-encoding carrier and embedding…

  ✓ embedded into song.stego.ogg  (129.8 KiB in 273ms)

  public key  song.stego.pub  share with senders
  private key song.stego.key  keep secret — required to read the message
  ! without the private key the message cannot be recovered
```

| Flag | Short | Required | Description |
|---|---|---|---|
| `--input` | `-i` | yes | Carrier audio: path, `file://` or `http(s)://` URL |
| `--data` | `-d` | yes | Text to hide |
| `--key` | `-k` | no | Recipient **public** key file; a keypair is generated when omitted |
| `--output` | `-o` | no | Destination file; its extension picks the format (default `<input>.stego.ogg`) |
| `--out-codec` | | no | Encoder to use when the container holds more than one, e.g. `alac` for `.m4a` |

**The output format follows the `--output` extension**, the same way `ffmpeg`'s
does:

```bash
mist embed -i song.mp3 -d "the eagle lands at dawn" -o song.flac
mist embed -i song.mp3 -d "the eagle lands at dawn" -o song.m4a --out-codec alac
```

A lossless output is worth preferring when you have the disk space: it is far
roomier and very nearly transparent. Measured against the carrier, a FLAC stego
file scores about 85 dB SDR where the Vorbis path scores about 26 dB — a
lossless encode adds no loss of its own, and the only change is one step in the
last bit of two percent of the samples. Run `mist formats` to see every target
your FFmpeg build can write. An unwritable one — a lossy format other than
Vorbis, or an extension FFmpeg does not know — fails immediately, before the
carrier is even read.

When `--key` is omitted, Mist generates a keypair and writes it beside the
output, named after it: `song.stego.pub` and `song.stego.key`. Keys are
hex-encoded so they can be inspected and pasted, and the private key is written
with mode `0600`. They are written only after the embed succeeds, so a failed
run leaves nothing behind. Give the public key to anyone who should be able to
send you messages; keep the private key — without it the message is gone.

### catch

```bash
mist catch --input song.stego.ogg --key song.stego.key
```

```
  mist · catch

  source      song.stego.ogg
  key         song.stego.key
  timeout     none, read to end of stream

  → listening…
  ▸ frame 0     23 B  the eagle lands at dawn

  ✓ 1 frame(s) recovered · end of stream
```

| Flag | Short | Required | Description |
|---|---|---|---|
| `--input` | `-i` | yes | Source to read: path, `file://` or `http(s)://` URL |
| `--key` | `-k` | yes | Private key file |
| `--timeout` | `-t` | no | Go duration; `0` (the default) reads until the stream ends |

A successful scan reports one frame, since the message is embedded once. `catch`
keeps reading to the end regardless, because the carrying frame may be anywhere
in the file. A file ends by itself; a live stream runs until `--timeout` elapses
or you interrupt it. The command exits non-zero when nothing was recovered.

Frames that carry no message and frames sealed for somebody else's key are
indistinguishable, so both are skipped in silence. `catch` will never tell you
"something was here but I could not read it" — that distinction would itself be
a signal worth detecting.

`catch` takes no format flag. The stream itself says whether the bits are in
Vorbis residues or in PCM samples, so a FLAC and an Ogg are read by the same
command.

### formats

```bash
mist formats
```

```
  mist · formats

  .aif        aiff        pcm_s16be · lossless, bits in the samples
  .caf        caf         alac · lossless, bits in the samples
  .flac       flac        flac · lossless, bits in the samples
  .ogg        ogg         vorbis · lossy, bits in the residues
  .wav        wav         pcm_s16le · lossless, bits in the samples
  .wv         wv          wavpack · lossless, bits in the samples
  …
```

The list is not kept inside Mist — it is what the libraries on your machine
report. Give `embed --output` one of those extensions, and name the encoder
with `--out-codec` where a container holds more than one. `--out-codec` accepts
either name libav knows a codec by, the encoder's or the codec's, so both `dca`
and `dts` work.

### estimate

```bash
mist estimate --input song.ogg
```

```
  mist · estimate

  carrier     song.ogg
  target      ogg/vorbis  vorbis · lossy, bits in the residues

  → decoding and probing capacity…

  frames      ████████████████████████  3/3 usable  3 total
  capacity    77 B  largest single message this carrier can hold

  ✓ this carrier can be used with `mist embed`
```

| Flag | Short | Required | Description |
|---|---|---|---|
| `--input` | `-i` | yes | Carrier audio: path, `file://` or `http(s)://` URL |
| `--output` | `-o` | no | Target container, or an output path whose extension names one (default `ogg`) |
| `--out-codec` | | no | Encoder to use when the container holds more than one, e.g. `alac` for `.m4a` |

`estimate` decodes the carrier and re-encodes it for the same target `embed`
would write — `--output`/`--out-codec` pick that target the same way, and no
`--data` or `--key` is needed, since capacity does not depend on the message or
the recipient. It reports the real limit `embed` enforces, not the
[`FrameCapacity()`](#choosing-a-carrier) heuristic: for Vorbis it groups the
encoder's actual packets into stego frames and counts each one's genuinely
eligible residues, and for a lossless target it computes each frame's sample
capacity directly. A carrier that cannot fit the protocol envelope in any frame
exits non-zero with the same hint `embed` would give:

```bash
mist estimate --input tiny.wav
```

```
  mist · estimate

  carrier     tiny.wav
  target      ogg/vorbis  vorbis · lossy, bits in the residues

  → decoding and probing capacity…

  frames      ░░░░░░░░░░░░░░░░░░░░░░░░  0/1 usable  1 total
  capacity    0 B  largest single message this carrier can hold

  ✗ mist: carrier has no frame with room for the payload — quiet or tonal audio
    yields too few usable residues; try a longer or richer carrier, or a
    lossless --output format, which holds far more
```

A lossless target is cheap to check — capacity there is arithmetic over the
sample count, so no full encode is needed — while a Vorbis target costs what
`embed` costs, since only the encoder's real residues say how many are
eligible.

### Interrupting a run

Both commands install a signal handler for `SIGINT`, `SIGTERM` and `SIGHUP`,
which cancels the context they run under. `catch` stops listening and reports
whatever it recovered up to that point; `embed` abandons the run without writing
an output file or any keys, since a half-embedded carrier is of no use to
anybody. `SIGKILL` cannot be trapped by any process, so it terminates the tool
immediately as it would anything else.

## Library

Runnable Godoc examples live in the [example](example) package
(`go test ./example`).

### Emitter

```go
pub, priv, err := mist.GenerateKeyPair()

emitter, err := mist.NewEmitter(pub)
stego, err := emitter.EmbedReader(ctx, carrier, mist.Text("hello"))
defer stego.Close()
io.Copy(dst, stego)
```

`Embed(ctx, source, payload)` takes a path or URL, `EmbedReader` an open
`io.Reader`, and `EmbedFile` an `*os.File`. All three return an
`io.ReadCloser` producing a new stego stream, which the caller must close.

The output format is Ogg Vorbis unless you ask for another:

```go
emitter, err := mist.NewEmitter(pub, mist.WithFormat("song.flac"))
```

`WithFormat` takes what `ffmpeg -f` takes — a container short name, or the
output path whose extension names one — and `WithCodec` is `-c:a`, for the
containers that hold more than one encoder. An unwritable target is
`ErrUnsupportedCodec` from `NewEmitter`, before any carrier is read.
`mist.Formats()` lists what the installed FFmpeg can write, and
`mist.LookupFormat(name, codec)` resolves one without constructing an Emitter —
useful for deriving an output filename from its `Ext`.

`mist.EstimateCapacity(ctx, source, format, codec)` decodes source the way
`Embed` would for that target and reports a `Capacity` — the number of stego
frames found, how many have room for the protocol envelope, and the largest
plaintext a single `Embed` call could carry — without writing anything out or
needing a recipient key.

### Catcher

```go
catcher, err := mist.NewCatcher(priv)
results, err := catcher.Listen(ctx, "https://radio.example.com/live.ogg")
for r := range results {
    fmt.Println(r.FrameIdx, string(r.Payload.Data))
}
if ctx.Err() != nil {
    // stopped by the caller rather than by end of stream
}
```

`Listen` accepts a path or URL and streams results as they are recovered. The
channel closes when the source ends **or** when the context is cancelled; check
`ctx.Err()` afterwards to tell the two apart. `ListenReader` does the same over
an open reader, and `Extract` scans an `*os.File` synchronously and returns a
slice. Prefer `Listen` for live sources: `ListenReader` buffers a non-seekable
reader in full before it starts.

### Options

| Option | Applies to | Purpose |
|---|---|---|
| `WithSenderAuth(priv)` | Emitter | Add an Ed25519 signature over the plaintext |
| `WithFormat(name)` | Emitter | Output container, or the output path naming one (`ffmpeg -f`) |
| `WithCodec(name)` | Emitter | Encoder override for a container holding several (`ffmpeg -c:a`) |
| `WithMaxRetries(n)` | Catcher | Tolerate transient read failures on a live source |
| `WithBackoff(d)` | Catcher | Pause between those retries |
| `WithLogger(*slog.Logger)` | Catcher | Receive diagnostics such as joining mid-frame |

The logger is discarded by default and never receives failed decrypts.

### Errors

| Sentinel | Meaning |
|---|---|
| `ErrInvalidKey` | Key missing or the wrong size |
| `ErrInvalidPayload` | Payload malformed or too large for a frame |
| `ErrInvalidSource` | Empty, nil or unusable source argument |
| `ErrUnsupportedCodec` | Carrier codec Mist cannot decode, or an output format it cannot embed into |
| `ErrCarrier` | Source could not be opened, demuxed or decoded |
| `ErrNoCapacity` | No frame had room for the message, so nothing was embedded |

`ErrNoCapacity` is deliberately loud. Everything on the *reading* side fails
silently, but an `Embed` that quietly produced a file containing nothing would be
a much worse outcome than an error.

## Choosing a carrier

This matters more than it might seem — **for an Ogg Vorbis output**. A lossless
output has none of the constraints below: every sample can carry a bit, so any
audio will do and the only limit is length. If a carrier is rejected or the
quality cost bothers you, writing FLAC instead of Ogg is usually the answer.

For Vorbis, the payload rides in quantized residues, and quiet or purely tonal
audio simply does not produce many usable ones. A pure
440 Hz sine wave yields so few that an 8-second frame cannot hold even the
64-byte envelope, and `embed` fails with `ErrNoCapacity` rather than writing a
file that silently carries nothing.

Music, speech, and anything with broadband content are all comfortable carriers.
As a rough sense of scale, an 8-second frame of ordinary stereo music holds a
little over a hundred bytes — a sentence or two, not a document. Capacity is
deliberately modest: embedding is confined to the frequencies where it cannot
be heard, and every extra byte is extra distortion.

Since the message occupies a single frame, a longer carrier does not buy more
room; a *richer* one does. If your message does not fit, shorten it or pick a
carrier with more high-frequency content.

`FrameCapacity()` reports a rough upper bound for the Vorbis path and
over-estimates, because it does not account for how many residues are actually
usable. It does not describe a lossless target at all, which holds far more.
Treat it as a planning hint; the real limit is enforced when you call `Embed`.
For the real number ahead of time, without producing any output, use
`mist estimate` (or `EstimateCapacity` from Go) — see [estimate](#estimate).

### Audio quality

Mist always decodes and re-encodes, so some loss is unavoidable for a lossy
*output* — but the embedding itself should be inaudible. For Vorbis, the encode
bitrate is derived from the carrier's own rate with headroom above it, rather
than fixed, and the payload is written only above 6 kHz at a low density.

Measured on a 128 kbps MP3, against a plain FFmpeg transcode of the same
decoded audio (signal-to-distortion, higher is better):

| | SDR |
|---|---|
| plain transcode, no embedding | 22.75 dB |
| mist re-encode, no embedding | 22.76 dB |
| **mist with a message embedded** | **22.58 dB** |

So the message costs under 0.2 dB — the re-encode itself dominates, and that is
the price of the format, not of the steganography.

A lossless output removes even that. There is no transcode underneath, and the
only change is one step in the last bit of two percent of the samples:

| | SDR |
|---|---|
| **mist FLAC with a message embedded** | **85.1 dB** |

Measured against the same decoded carrier, 20 s of pink noise at 44.1 kHz
stereo.

## Supported formats

**Input is whatever your FFmpeg can decode.** The carrier is decoded to PCM and
re-encoded, so it need not already be in the output format, and Mist keeps no
list of acceptable inputs: it carries libav's own codec id through and asks
libav whether a decoder exists. MP3, FLAC, WAV, Opus, AAC/M4A, ALAC and the rest
all work if your build supports them, and a format yours cannot decode is
reported as `ErrUnsupportedCodec` naming the codec. Sample format does not
matter either — FLAC's integer samples and Vorbis's floats are normalised alike.

**Output is Ogg Vorbis or any lossless codec your FFmpeg can encode.** That
list is not Mist's either: a format qualifies when libav reports an encoder for
it and its codec descriptor carries `AV_CODEC_PROP_LOSSLESS`. `mist formats`
prints what yours offers — typically FLAC, WAV, ALAC, WavPack, TTA, AIFF, CAF,
W64, AU and the raw PCM containers.

The two kinds of output are not two features, they are two hiding places for
the same protocol:

| | Ogg Vorbis | any lossless format |
|---|---|---|
| bits live in | quantized residues | PCM sample LSBs |
| extraction | Huffman-decode, stop before the iMDCT | decode; the samples are unchanged |
| room in an 8-second frame | ~130 bytes | ~1.7 kB at 44.1 kHz stereo |
| SDR against the carrier | ~26 dB | ~85 dB |
| carrier must be | broadband; tonal audio may not fit | long enough, and nothing else |
| file size | small | large |

Frame layout, keyed positions, density, filler and crypto are identical in both.

A lossy output other than Vorbis is refused. Mist can only hide bits it can
still find afterwards, and it has no way to rewrite an MP3 or AAC bitstream —
`ErrUnsupportedCodec`, raised by `NewEmitter` before the carrier is read.

Two things follow from re-encoding. An Ogg Vorbis input is still decoded and
re-encoded, with the generation loss that implies: Mist owns the whole encode
path and does not patch somebody else's existing bitstream. And a stego file
survives being copied byte for byte, but not being transcoded — converting a
stego FLAC to MP3, or even back to Vorbis, destroys the message.

## Status and limitations

Phase 1 is complete end to end. Crypto, wire framing, the libav cgo layer,
Vorbis residue parse and rewrite, lossless sample-domain embedding, keyed
position selection, constant-density embedding, the Emitter, the Catcher, and
the CLI are all implemented and tested, including multi-frame carriers and live
streams. The lossless path is verified end to end for FLAC, WAV, ALAC, WavPack,
TTA, AIFF and CAF.

Known gaps, all recorded in [CLAUDE.md](CLAUDE.md):

- `FrameCapacity()` over-estimates and describes the Vorbis path only, as above.
- `Embed` over `http(s)` buffers a finite file rather than streaming a live source,
  and the lossless path buffers the whole carrier before encoding.
- Lossless embedding quantizes to 16 bits wherever the encoder offers that depth,
  so a 24-bit master comes back at CD depth.
- A listener that joins mid-frame cannot align to that frame; it says so once and
  resumes cleanly from the next one.
- A scan goroutine parked in a blocking libav read outlives its context until the
  read returns. `Listen`'s channel still closes immediately on cancellation, so
  callers are unaffected.

## Security notes

The design target is a passive warden who knows exactly how Mist works and has
the file, but not the private key — Kerckhoffs's principle, with all security
resting on the key. Encrypting the payload stops it being *read*; the keyed
position selection and constant embedding density are what aim to stop it being
*noticed*.

One consequence is worth spelling out: position selection derives from the
recipient's **public** key, which is not a secret. Anyone holding it can work out
which residues to look at, so everything they would find there must be
indistinguishable from noise — which is why the ciphertext length is masked with
a subkey derived from the ECDH shared secret rather than stored in the clear.

Explicitly **not** goals in Phase 1: surviving a digital-to-analog-to-digital
round trip, surviving re-encoding by a different encoder, embedding into
pre-existing files produced elsewhere, and deniability under coercion
(undetectability and deniability are different guarantees). "Undetectable" also
remains a design target rather than a proven property: a benchmark suite running
real steganalysis detectors against Mist's output does not exist yet, and
building one is an open task.

## Development

```bash
make build                       # -> bin/mist
make keys                        # X25519 keypair via OpenSSL -> keys/mist.{pub,key}
make embed INPUT=song.mp3 DATA="hello" OUTPUT=out.flac KEY=keys/mist.pub
make catch INPUT=out.flac KEY=keys/mist.key TIMEOUT=30s
make formats                     # output formats this FFmpeg build can write
make estimate INPUT=song.mp3 OUTPUT=out.flac   # capacity check, no key needed
make test                        # verbose: -v -race -cover, full tracebacks
make test LOG=trace              # ...and libav logging turned all the way up
make test-quiet                  # same run, results only
make lint                        # golangci-lint run --timeout=5m
```

`make test` is deliberately loud: every test and subtest is named, `t.Log`
output is shown, coverage is reported per package, `GOTRACEBACK=all` dumps
every goroutine on a panic, and `MIST_AV_LOG` lets libav report what the codec
layer is doing. Set `MIST_AV_LOG` on the `mist` binary too when a carrier
misbehaves — it defaults to silent so codec chatter never lands in the CLI's
output.

`make keys` writes the raw key bytes as hex, which is the format the CLI reads.
OpenSSL stores X25519 keys as PKCS#8/SPKI DER whose final 32 bytes are the key
itself, so the pair it produces is interchangeable with one `mist embed`
generates for itself.

[CLAUDE.md](CLAUDE.md) is the design reference: package layout, protocol
constants, the invariants the stego layer must preserve, and the conventions
used throughout. Read it before changing anything under `internal/`.

CI runs golangci-lint v2.11.4 and race tests with a coverage badge, matching
[gocue](https://github.com/iSerganov/gocue).

## Ethics

This project exists for legitimate confidential communication, digital
watermarking, and steganography/steganalysis research. It is not designed for,
and should not be used for, concealing illegal content or evading its detection.

## Contributing

Issues, design notes, and pull requests are very welcome — this project is meant
to be learned from and built with. If you are adding to `internal/`, please read
the invariants in [CLAUDE.md](CLAUDE.md) first; several of them are load-bearing
in non-obvious ways, and the tests exist to catch exactly those cases.

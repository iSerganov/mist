# Changelog

All notable changes to this project are documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [1.0.0]

### Changed

- **Breaking: the embedding protocol has changed, and files embedded by
  0.1.x or 0.2.x can no longer be read.** `catch` on such a file reports
  that nothing was found, just as it would for a file with no message,
  because Mist has no in-band version marker to tell the two apart.
  Re-embed any message that still needs to be read. The changes behind
  this:
  - Payload bits are carried by a syndrome-trellis code over the eligible
    values instead of one bit per keyed position with LSB matching. The
    rate stays at 2%, so capacity is unchanged, but about 0.14 values
    change per bit instead of 0.5.
  - Positions are ordered by a ChaCha20 keystream instead of
    HMAC-SHA256 counters.
  - Lossless output leaves digital silence and the end padding untouched,
    so those samples are no longer part of the carrier.
- Ogg Vorbis embedding now prices each change by how far it moves the
  residue's spectral vector and avoids the costly ones. On 13 dense music
  tracks its cost beyond a plain re-encode drops from 1.4 dB of SDR to
  under 0.02 dB.
- In lossless output the direction of each ±1 change now follows the
  frame's own sample histogram, so the histogram no longer widens with
  every change.
- Lossless capacity, as `EstimateCapacity` and `mist estimate` report it,
  no longer counts silent stretches of the carrier.
- Lossless output keeps the carrier's depth the way `ffmpeg` does, instead
  of always writing 16 bits: a 24-bit or float source is written at 24 bits
  wherever the codec allows it (FLAC, ALAC, TTA, WavPack), and a 16-bit
  source stays 16-bit. Samples are converted by libswresample, as `ffmpeg`
  converts them, so apart from the embedded ±1 changes the output matches
  a plain `ffmpeg` encode sample for sample. Building now needs
  libswresample (`libswresample-dev` on Debian/Ubuntu).
- Ogg Vorbis output is encoded in VBR at the quality level that matches the
  carrier's quality, instead of an ABR target at 1.5 times the source rate:
  the source rate is scaled by codec efficiency (MP3 ×0.75, AAC ×1,
  Opus ×1.3) and rounded to the nearest `-q:a` level, and a lossless source
  is written at q8. The stream header then reads like an ordinary
  `ffmpeg -q:a N` encode. Vorbis capacity drops with the rate: a 128 kbps
  MP3 carrier holds about a third of what it did.

### Added

- `make harness`: a local report on detectability and audio quality per
  output format, run on a music corpus or a built-in synthetic set. It
  measures four classical detectors (chi-square, SPA, RS, HCF-COM) and a
  cross-validated classifier, plus SDR against a plain re-encode, per
  carrier and on average.
- The harness compares Mist's output with the same carrier encoded by the
  `ffmpeg` command line at its own defaults, which is the clean file a
  warden would actually have. A new table sets the two side by side on
  length, trailing digital zeros, sample format and nominal bitrate, where
  any difference gives Mist away without statistics. The detectors run
  against that ffmpeg copy, and again against Mist's own re-encode to
  isolate the embedding. The harness now needs `ffmpeg` on `PATH`, and the
  synthetic corpus gains a 24-bit carrier.
- The harness scores detection per file as well as per chunk, and its 95%
  intervals are bootstrapped by carrier instead of by chunk, so they reflect
  how many tracks were measured. Each detector also reports the lower bound
  on Cachin's ε that its file-level AUC proves.

### Fixed

- Output is now exactly as long as the carrier. Mist used to zero-pad the
  carrier to a whole number of encoder blocks, so every lossless output
  ended in up to one block of digital zeros. A plain encode leaves no such
  padding, so it gave Mist away without any statistics.
- Ogg Vorbis output from a mono carrier with a high source bitrate, such
  as a mono WAV, or from a 22 kHz mono carrier, no longer fails to open
  the encoder.

## [0.2.0] - 2026-09-21

### Added

- Lossless carrier support: `Embed`/`Extract` now carry the payload in PCM
  sample LSBs for any lossless codec the installed FFmpeg can write (FLAC,
  WAV, ALAC, WavPack, TTA, AIFF, CAF), sharing the same frame layout, keyed
  positions, density and crypto as the Ogg Vorbis path.
- Multi-frame payload spanning: a message that does not fit in a single
  stego frame now spreads across as many consecutive usable frames as it
  needs, each independently sealed with its own ephemeral key, instead of
  failing outright.
- `mist estimate` CLI command and the underlying `EstimateCapacity`/
  `Capacity`/`Source` API: reports a carrier's real per-frame and total
  (spanning) capacity, plus its codec, container, sample rate, channel
  count, bitrate and duration, without embedding anything.
- `mist formats` command and `Formats`/`LookupFormat` API for discovering
  which output targets the installed FFmpeg can write.

### Fixed

- Vorbis embed now canonicalizes its packet stream (an in-memory mux and
  demux pass) before grouping it into stego frames. Ogg stores a granule
  position per page, not a timestamp per packet, so a reader reconstructs
  each packet's timestamp from that; at a genuine block-size transition,
  that reconstruction could land a packet in a different frame than the
  encoder's own accounting used, silently breaking recovery. This is most
  visible once a payload spans several frames, since every boundary in the
  span has to agree, but it could rarely affect a single-frame message too.
- Multi-channel PCM writes now target libav's `extended_data`, avoiding an
  out-of-bounds write for carriers with more than 8 channels.
- Fixed a NULL-vs-query-failure ambiguity when probing an encoder's
  supported sample formats, and a zero-length destination case in the cgo
  layer's string-copy helper.

### Changed

- `FrameCapacity()` is documented as a planning heuristic only; its doc
  comment no longer implies a Phase-2 limit now that spanning exists.

## [0.1.0] - 2026-09-08

Initial Phase 1 release: asymmetric-key steganography for Ogg Vorbis audio.

### Added

- Hybrid X25519 + HKDF-SHA256 + ChaCha20-Poly1305 encryption, with a fresh
  ephemeral key per stego frame and an optional Ed25519 sender signature.
- Ogg Vorbis payload embedding via quantized residue LSB matching at keyed
  PRNG positions, constant embedding density with CSPRNG filler, and
  band-limiting to reduce audible impact.
- `Emitter`/`Embed`/`EmbedReader`/`EmbedFile` and `Catcher`/`Listen`/
  `ListenReader`/`Extract` public API.
- `mist` CLI with `embed` and `catch` commands.

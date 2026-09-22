# Changelog

All notable changes to this project are documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

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

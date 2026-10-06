# Changelog

All notable changes to this project are documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [1.1.0]

### Added

- `mist analyze [folder]`: an interactive terminal browser for asking how a
  warden would read an audio file. It walks a folder and its subfolders,
  lists every audio file with what libav reports about it, and analyzes
  the one picked. Every format is accepted. The report answers two separate
  questions.
  - **Does it carry a Mist message?** Every stage of the harness that one
    file supports runs against it, and each score is read against a
    calibration run built into the binary. The answer is a probability,
    where 50% means no evidence either way. Mist is built to keep these
    detectors at chance, so most files land near it. A format Mist cannot
    write, such as MP3 or AAC, is 0%.
  - **Did any other tool hide data in it?** These checks look outside the
    audio samples:
    - bytes after the end of the audio
    - unknown chunks and atoms
    - padding that is not empty
    - files appended to cover art, and tags that read like encoded binary
    - MP3 header bits that change between frames
    - encrypted-looking MP3 ancillary data
    - LSB replacement in lossless digital silence

    They report a level (none, low, medium, high) with its reasons, not a
    probability, because there is no corpus of other tools' output to
    calibrate against. On 1,851 real files (radio captures, streamed AAC
    segments, rips and editor exports) they flagged two as high, both
    with genuinely unexplained data after the audio.
- `--reference`, or `r` in the list, compares the file with its original.
  The original is encoded through Mist with nothing embedded and diffed
  value by value. A clean file made from it matches exactly, and a Mist
  file differs in about 0.13% of its values, each sample by exactly ±1,
  which settles the Mist question at 1% or 99%.
- `mist.Probe` reads the stream facts ffprobe shows without decoding the
  file. `mist.Analyze` is the analysis behind the command, for use as a
  library.
- `make calibrate` runs the harness and also writes `calibration.json`,
  the reference `analyze` reads a file against. `make analyze DIR=`
  starts the browser.

### Changed

- The harness now shares its decoding, chunking, container traces and
  ffmpeg reference encodes with `analyze`, so a calibration and the
  analysis read against it always measure the same thing. Its reports are
  unchanged.
- The CLI depends on Bubble Tea v2 for the interactive screen.

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
    rate is 1%, half of 0.2's 2%, so capacity halves, but about 0.14 values
    change per bit instead of 0.5. Detectability grows with the rate: on 84
    carriers Ogg Vorbis scored file AUC 0.58 at 2% and 0.52 at 1%, and
    pooling files no longer raised it.
  - Positions are ordered by a ChaCha20 keystream instead of
    HMAC-SHA256 counters.
  - Lossless output leaves digital silence and the end padding untouched,
    so those samples are no longer part of the carrier.
  - The envelope carries an Elligator 2 representative of the ephemeral
    X25519 key instead of the key. A bare key has its top bit clear and lies
    in the prime-order subgroup, and the position seed comes from the public
    key, so a warden holding only that key could read the first frame's
    bits and test those 32 bytes: the harness's new key-aware warden scored
    AUC 1.000 on every carrier. It now scores 0.500.
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
- FLAC output now writes the sample count and MD5 into STREAMINFO after the
  encoder is flushed, and uses the same block size as `ffmpeg -c:a flac`:
  the largest power of two within a tenth of a second, rather than the
  encoder's own 105 ms block. Ogg Vorbis comments carry the same `Lavc`
  encoder tag the ffmpeg command line writes. Payload framing is unchanged,
  so a file embedded before this still extracts.

### Added

- The harness confirmatory family is frozen at six wardens: HCF-COM, the
  logistic classifier, the Markov model, the key-aware warden, a rich-model
  logistic, and a public-key selection-channel score. Fisher, stump and
  random-subspace fits of the rich vector, plus the CNN waveform,
  spectrogram and hybrid nets, are baselines with a positive control each.
  Report schema is 6. This does not change an embedded file.
- The harness fingerprint verdict compares Mist with the canonical ffmpeg
  workflow: Vorbis at the `-q:a` level Mist chose, and a lossless codec at
  ffmpeg's defaults. Default ffmpeg stays a labeled second comparison, so a
  nominal-rate gap against ffmpeg's default quality is not a failed verdict.
  An audio-blind classifier scores file metadata only, separately for
  canonical against Mist-clean and for Mist-clean against stego. Its pass is
  a file interval that includes 0.5.
- Corpus manifests can mark a sealed split. A harness run scores one split and
  records hashes of the carriers it did not score. `make corpus` writes a
  generated development set and a source-disjoint sealed holdout outside the
  repository.
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
- A second classifier in the harness, **markov**, trained on how the second
  difference between samples changes from one sample to the next, the
  rich-model features of audio steganalysis. ±1 changes stand out more in
  smooth audio's curvature than in the values the first classifier sees.
  `steganalysis.Markov` computes the features.
- A key-aware warden in the harness, `key-aware`, for the adversary the
  design assumes: one who knows the recipient's public key, recovers the first
  frame's bits and tests the envelope's leading 32 bytes for the structure of
  an X25519 public key (`steganalysis.HonestX25519`).
- The harness reports detection per corpus folder, against the number of
  chunks scored and against the number of files a warden pools, and runs
  an external CNN warden (`make cnn-warden`, `tools/cnn_warden`) whose
  result appears in the report.
- For Ogg Vorbis the harness runs the detectors against ffmpeg encoded at
  the quality level Mist chose, not at ffmpeg's default q3. Against the
  default, a carrier that maps to another level differs in nominal bitrate
  and length, and every detector that read those scored AUC 1.000; at the
  same level the embedding alone leaves a faint signal (file AUC 0.43 to
  0.54 on 13 tracks). The table of plain properties still compares with
  ffmpeg's default, so that difference stays visible.

### Fixed

- A carrier decoded from MP3, AAC, Opus or Ogg Vorbis keeps no more of its
  encoder delay, padding or pre-skip than ffmpeg keeps. Mist used to decode
  the whole stream, so on 41 of 84 carriers its output ran a few hundred
  samples longer than a plain ffmpeg encode; the demuxer's skip-samples
  side data now reaches the decoder, which trims as the ffmpeg command line
  does.
- The harness's 95% intervals now contain the AUC they are built around.
  The carrier bootstrap counted a carrier drawn twice against its own clean
  copy four times over, so on paired data the whole interval drifted off
  the estimate, and the ε bounds with it.
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

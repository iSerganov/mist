# Phase 3: remove non-steganographic fingerprints

Phase 3 decides which innocent encode Mist is supposed to resemble, audits
the container against that encode, and adds a classifier that never sees
samples. It does not change the payload protocol. It does change some
container bytes, for the reasons below.

The metal-dev baseline in `REPORT.md` was not re-run. That file remains the
Phase 1 measurement (schema 3, corpus `metal-dev`, first 300 seconds, 84
lineages). A later comparison has to repeat that corpus. The synthetic run
described here only checks the fingerprint gate.

## 3.1 Canonical workflow

The canonical innocent workflow is the one Mist already encodes:

- Vorbis: `ffmpeg -q:a N` at the source-conditioned level `openVorbis`
  selects (nearest libvorbis nominal rate after the efficiency scale, or q8
  for a lossless or rateless source).
- Lossless: ffmpeg's own defaults. A lossless encoder has no quality knob.

Default ffmpeg on Vorbis is a second threat model. A warden who re-encodes
at ffmpeg's default quality (q3, nominal rate often far from the source)
will see a nominal-bitrate gap. That gap is printed in its own column and
does not fail the canonical verdict.

This split was already in the harness, unlabeled. Detector tables for
Vorbis used the matched `-q:a` twin. The fingerprint table used ffmpeg
defaults, which is why Vorbis nominal bitrate "failed" on 73 of 84
metal-dev carriers. Step 3.1 forbids that unless both threat models are
named. Switching the embedder to default q3 would change quality and
capacity, so the encoder policy stays. The verdict moves to the workflow
the encoder already implements.

## 3.2 Trace audit

Each measured carrier now keeps four traces: ffmpeg defaults, canonical
ffmpeg, Mist with nothing embedded, and Mist with a message. The source's
sample count, rate, channel count and sample format are recorded beside
them.

Identity fields, which fail the verdict if Mist-clean or Mist-stego
disagrees with the canonical file:

- sample count, trailing digital zeros, decoded sample format, nominal
  bitrate (5% slack)
- codec, container, channel count, sample rate, raw depth, channel layout,
  timebase, encoder tag, metadata tag order, codec tag, profile, stream
  count, skip-sample count
- FLAC: an MD5 is present or not, and the STREAMINFO sample count
- Ogg: number of distinct serials, and the last granule position
- a metadata block type the FLAC spec does not define

Not equality gates, because they move with VBR and with embedding. They are
features of the metadata warden instead:

- file size, actual bitrate, packet count, mean packet size
- Ogg page count, mean page body, mean lacing, first audio granule
- FLAC padding size and the count of ordinary metadata blocks

Ogg serial values are not stored and not compared. Each encode draws a new
serial, so comparing the number would fail every file. The audit compares
how many serials a file has.

`ffprobe` supplies the container tags. `MIST_FFPROBE` overrides the binary;
otherwise the harness looks beside `MIST_FFMPEG` and then on `PATH`. A
probe failure is recorded on the trace and does not abort the carrier.
FLAC `bits_per_raw_sample` arrives as a string from ffprobe and as a number
from other codecs, so that field accepts either. STREAMINFO and Ogg pages
are parsed from the bytes, so those facts do not depend on ffprobe.

## 3.3 Audio-blind warden

Two questions, scored apart, on file metadata only:

- canonical ffmpeg against Mist-clean (pipeline fingerprint)
- Mist-clean against Mist-stego (embedding side effect)

The model is the same nested lineage-grouped logistic regression as the
audio classifier, falling back to fixed-penalty cross-validation below
four lineages. The preregistered pass is a file interval that includes
0.5. An interval that excludes 0.5 is a failed objective and is marked as
one in the summary. The threshold string is in the run manifest
(`metadata classifier file interval includes 0.5`).

Report schema is 5. The manifest names the canonical workflow, the
threshold, and the feature list.

## 3.4 Vorbis nominal rate

Against the canonical `-q:a` file, nominal rate matched on every synthetic
Vorbis carrier that encoded (256 kbps against 256 kbps). Against default
ffmpeg the same files differed (256 against 112) and in nothing else once
the encoder tag was aligned. That is the 73/84 metal-dev distinction,
reclassified: it is the default-ffmpeg threat model, not a bug in the
quality rule. The metal-dev number is not withdrawn and not re-measured.
The report keeps printing it so a reader cannot mistake the canonical pass
for "identical to default ffmpeg".

## Output bytes that changed

Three container fixes. None of them change the payload framing, the
density, the trellis, or the keys. A file embedded before this phase still
extracts.

1. FLAC STREAMINFO. The encoder writes the MD5 and the total sample count
   into its extradata only when it is flushed. `Encoder.Info` had kept the
   extradata from open time, so the muxer wrote a header with neither.
   `Info` now re-reads the encoder, and the container is preserved across
   that refresh. Custom IO also advertises `AVIO_SEEKABLE_NORMAL`, because
   the FLAC muxer will not patch STREAMINFO on trailer unless that flag is
   set. The encode-then-mux path no longer depends on the trailer patch:
   the header is written from the flushed extradata.

2. Encoder tag. The ffmpeg command line sets stream metadata to
   `Lavc<version> <codec name>` (`set_encoder_id`). Ogg and FLAC copy that
   into the comment. Mist was leaving the muxer's own `Lavf` ident, so
   every Vorbis file differed in the encoder tag even at the same `-q:a`.
   Non-PCM streams now get the CLI string. PCM is left alone: a WAV from
   ffmpeg reports `Lavf` to ffprobe, and writing the Lavc string there
   would make WAV unlike ffmpeg.

3. FLAC block size. With no frame size requested, libav's FLAC encoder
   picks a 105 ms block (4608 samples at 44.1 kHz). The CLI's filter
   instead hands it a power-of-two frame near a tenth of a second, and the
   encoder keeps a frame size it was already given. Mist now sets that
   size before open: the largest power of two no greater than
   `sample_rate/10`, and no greater than 65535. Checked against
   `ffmpeg -c:a flac` at 8, 11.025, 12, 16, 32, 44.1, 48, 88.2, 96 and
   192 kHz. Before this, a clean FLAC was hundreds of bytes away from
   ffmpeg on every synthetic carrier and the metadata warden scored file
   AUC 1. After it, the clean file size matched on all six.

The lossless clean twin in the harness now snaps samples onto the encoder
grid before encoding, which is what `Embed` does before it changes any
sample. Without that snap the twin was a second resample, not "Mist with
nothing embedded".

## Synthetic check

`MIST_HARNESS_CORPUS_NAME=synthetic`, no external corpus, formats
flac, wav and ogg, six built-in carriers of 16 seconds. This is a
fingerprint check. Six lineages cannot support a detectability claim, and
a couple of audio-detector intervals sit just off 0.5 for that reason.
Those rows are not a revision of `REPORT.md`.

| Format | Canonical identity | Metadata warden | Default ffmpeg |
|---|---|---|---|
| FLAC | match, 6/6 | both intervals include 0.5 | match (same encode) |
| WAV | match, 6/6 | both intervals include 0.5 | match (same encode) |
| Vorbis | match, 5/5 | both intervals include 0.5 | nominal bitrate differs, 5/5 |

Vorbis skipped `tones-noise-44k-mono` because that carrier had no frame
with room for the payload. That skip is capacity, not a fingerprint.

Clean FLAC size minus canonical ffmpeg was 0 on all six carriers. Stego
minus clean was 0, +1 or −2 bytes. That residual is the embedded ±1
changes inside a lossless compressor. The metadata interval for
Mist-clean against stego still included 0.5.

## Gate

- No deterministic identity-field mismatch against the canonical workflow
  on this synthetic set.
- The metadata classifier's file intervals include 0.5 on this set, for
  both questions and all three formats.
- The nominal-rate gap against default ffmpeg is an accepted, labeled
  difference. It is not a failed canonical objective, and it is not hidden.
- Ogg serial values are excluded on purpose.
- The metal-dev corpus was not re-scored. Passing this gate on six
  synthetic carriers does not move the baseline.

## Wire profile and rollback

The stego wire profile is unchanged: same framing, same density, same
code, same keys. Container bytes change for new encodes (FLAC block size
and STREAMINFO, Vorbis comment tag).

Rollback is to revert the encoder-info refresh, the FLAC frame size, the
encoder-tag write, and the harness comparison. Output returns to 105 ms
FLAC blocks, a STREAMINFO without MD5, and a Lavf tag on Vorbis. The
fingerprint verdict returns to scoring default ffmpeg.

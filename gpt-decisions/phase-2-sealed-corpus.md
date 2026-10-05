# Phase 2: a representative sealed corpus

Phase 2 adds the corpus contract the later detector work has to sit on. It
does not change the embedder, the wire format, the density, or the
syndrome-trellis code. A file embedded before this phase extracts exactly as
it did. No new detectability number is claimed, and the metal-dev baseline in
`REPORT.md` is not re-run.

The work follows `GPT-SOL-D-PLAN.md` §8. Recommended PR sequence step 16, the
blind red-team release run, stays open. This phase builds the seal that run
will use. It does not open the seal.

## What was built

`make corpus CORPUS_OUT=<directory>` writes a generated PCM corpus and
`corpus.json` into that directory. The directory must sit outside a git
checkout. The audio is not committed, and the manifest uses public ids only.

Development cells, each its own lineage except where a longer cut or a
transcode is the same source:

| Id | Content class | Provenance |
|---|---|---|
| `speech-shaped-16k-mono` | amplitude-modulated noise, not speech | 16 kHz, mono, 16-bit, one 8 s frame |
| `sparse-tones-48k-24` | sparse tones | 48 kHz, stereo, 24-bit, one frame |
| `tonal-bed-44k` | tones plus noise | 44.1 kHz, stereo, 16-bit, one frame |
| `tonal-bed-44k-long` | the same source, three frames | same lineage as `tonal-bed-44k` |
| `environment-48k` | shaped noise | 48 kHz, stereo, 16-bit |
| `transients-44k` | periodic bursts | 44.1 kHz, stereo, 16-bit |
| `silence-heavy-44k` | half silence | 44.1 kHz, stereo, 16-bit |
| `low-dynamic-44k` | quiet noise | 44.1 kHz, stereo, 16-bit |
| `multichannel-48k` | tones plus noise | 48 kHz, six channels, 16-bit |

When `ffmpeg` can encode them, the tonal bed is also written as FLAC and MP3.
Those copies share the lineage `tonal-44k`, so a transcode cannot be scored as
an independent recording or placed in the other split. A missing encoder is
recorded as `transcodes_skipped` and does not fail the corpus.

The sealed holdout is three cells with different seeds and different
lineages: environment, sparse 24-bit, and speech-shaped. Nothing in the
development split uses those lineages.

The manifest also names the public sources this generator does not fetch:
LibriSpeech, FSD50K, MUSAN, MAESTRO, and AudioSet. MAESTRO is marked
non-commercial. AudioSet is marked as an id list, not redistributable audio.
Adding any of them is a manifest edit — license, hash, lineage, split — in a
directory outside the repo.

## The seal

A carrier's `split` is `development` (the default) or `sealed`.

A normal harness run scores development carriers only. Each sealed carrier is
hashed from the file bytes and listed under `held_out` with reason `sealed`.
It is not decoded, embedded, or given to a detector.

`MIST_HARNESS_UNSEAL=1` scores the sealed split only. Development carriers are
hashed and listed with reason `development-while-unsealed`. One run never
contains both splits, so a classifier cannot train on a carrier and test on
its derivative.

A lineage that appears in both splits is a manifest error. The loader rejects
it before any file is scored. Chapters of one session stay in one split by
sharing a lineage there; they do not cross the seal.

Report schema is 4. The header states how many sealed carriers were held out,
or, on an unsealed run, that the development carriers were not scored. Schema
3 readers would otherwise treat every carrier as scored.

## Conditions

The manifest freezes the control list from the plan: canonical FFmpeg,
Mist-clean, minimal and normal payloads, 25/50/75/near-max fractions, both
sides of a span boundary, signed and unsigned, fixed and varied keys, repeated
embeddings, and filler-only.

This phase's harness run still executes four of them, which it already did:
canonical FFmpeg, Mist-clean, the normal 64-byte payload, and the 1-byte
minimal payload. The experiment manifest now names those as
`executed_conditions` and the full list as `planned_conditions`. Generating
every remaining condition for every carrier would rescore the baseline and
multiply the run. That measurement waits until a later phase asks for one of
those conditions on purpose.

Payload fractions are not byte counts. Near-max and the span edges depend on
the encoded frame, which the cover file does not know. The catalog records
the fractions. The byte count is an encode-time fact.

## The power gate

The Phase 1 FLAC simulation asked for about 128 independent lineages before a
shift to D = 0.55 would push a lineage interval off 0.5 in 90% of draws. The
generated development set has on the order of ten lineages. `corpus.json`
records `requirement_met: false` and that 128 figure, so a later reader cannot
mistake class coverage for power.

Copying the same noise 128 times would meet the count and miss the point.
Real speech, music, and environmental recordings have to be added through the
manifest, from the named public sources or another licensed set, with one
lineage per session. The metal-dev baseline remains 84 files of one content
family, each its own lineage because that run had no session manifest. It is
not this phase's sealed set, and it is not re-measured here.

## What this does not do

- It does not download datasets.
- It does not commit audio or machine paths.
- It does not inspect sealed detector scores. There are none.
- It does not run the blind red team. That is PR sequence step 16, and it
  stays open until the implementation and the release criteria are frozen.
- It does not claim the content is representative of studio music, live
  recordings, or podcasts. The generated cells are controls for those shapes:
  sparse, tonal, noisy, quiet, silent, and speech-shaped. The real recordings
  are an acquisition step the manifest is ready to hold.
- Platform and FFmpeg-version cells (Linux and macOS, x86-64 and arm64, old
  and new libav) are not generated here. A run manifest already records the
  build that scored a corpus. Matching those cells across machines is a later
  measurement, not a second embedder.

## Tests

`go test -tags harness` on the new suite methods checks that a generated
corpus keeps development and sealed lineages apart, that unsealing scores
only the sealed split, that a shared lineage is rejected, that a held-out
file is hashed and not decoded, and that the generator refuses a directory
inside a git checkout. The example manifest still parses with the new
optional fields.

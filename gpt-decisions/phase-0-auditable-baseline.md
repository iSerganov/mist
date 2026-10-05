# Phase 0: auditable baseline

This note justifies the harness and documentation changes that freeze a
reproducible detectability baseline. It is not a security proof.

## Goal

Phase 0 in `GPT-SOL-D-PLAN.md` is measurement infrastructure, not a new
embedder. Before later cost or rate work can be believed, a second machine
must be able to say exactly which commit, toolchain, corpus identity and
payload schedule produced a number.

The work also had to keep private audio and machine-local paths out of git.
Reports may be pasted into pull requests. Anything that names
`/Users/.../Downloads/...` or a song title from a private library is a leak.

## What we changed, and why

### 1. Run manifest and raw scores

The harness now writes four artefacts under `HARNESS_OUT`:

| File | Why it exists |
|---|---|
| `report.md` | Human summary, still the thing you read |
| `report.json` | Same numbers, machine-readable |
| `manifest.json` | Git commit and dirty flag, Go/OS/arch, FFmpeg and libav/libvorbis versions, public corpus name, per-carrier SHA-256, lineage, duration, protocol density/bands, payload sizes, classifier description |
| `scores.json` | Every chunk and file score with public carrier / category / lineage ids |

Markdown is a rendering. The JSON is the record. That is the only way a later
PR can say "this AUC is from commit X on corpus Y with N independent groups."

Duration is derived from decoded sample count and sample rate so a reader
does not have to infer length. Classifier and feature fields record what the
current logistic/Markov models actually use, so a future richer warden cannot
silently reuse an old interval.

We do **not** write `pkg-config --cflags` into anything that might be
committed. Those flags contain Homebrew or `/usr/local` paths. Versions are
enough to reproduce the build; include flags stay on the machine.

### 2. Public corpus identity

A directory walk used to put relative filenames and folder names into the
report. That is how album titles and local folder names leaked.

Three modes now exist:

1. **Synthetic** (`CORPUS` empty). Built-in generated noise. Public names
   such as `white-noise-44k-stereo`.
2. **Anonymous walk** (`CORPUS` set, no manifest). Files are loaded, but the
   report only sees `carrier-0001`, category `external`. The real filename
   never becomes a carrier id.
3. **Declared manifest** (`CORPUS` + `CORPUS_MANIFEST`). The JSON lists a
   public `id`, `category` and `lineage`, plus a path that must stay below
   the corpus directory. Absolute paths and `..` are rejected.

`testdata/harness/corpus.example.json` is the committed schema. It uses
fictional relative paths only. Operators copy it and fill in their own
private tree locally; that tree stays out of git.

Load errors replace the real path with `<path>` or `<carrier>` before they
can become a skip reason in `report.md`. HOME, temp directories and the
carrier path are stripped.

### 3. Configurable analysis paths

Every executable and directory the harness or CNN touches is an override.
Nothing in source assumes a music folder on this machine.

| Make variable | Environment | Default | Role |
|---|---|---|---|
| `CORPUS` | `MIST_CORPUS` | empty (synthetic) | Carrier directory |
| `CORPUS_MANIFEST` | `MIST_HARNESS_CORPUS_MANIFEST` | empty | Public id map |
| `CORPUS_NAME` | `MIST_HARNESS_CORPUS_NAME` | from manifest | Public corpus label |
| `HARNESS_OUT` | `MIST_HARNESS_OUT` | `harness-out` | Report destination |
| `BASELINE` | `MIST_HARNESS_BASELINE` | empty | Earlier `report.json` |
| `MERGE` | `MIST_HARNESS_MERGE` | empty | Join per-format reports |
| `FORMATS` | `MIST_HARNESS_FORMATS` | verified set | Output formats |
| `JOBS` | `MIST_HARNESS_JOBS` | `min(4, nCPU)` | Parallel carriers |
| `MAX_SECONDS` | `MIST_HARNESS_MAX_SECONDS` | 0 (whole file) | Prefix cut |
| `PUBLIC_KEY_HEX` | `MIST_HARNESS_PUBLIC_KEY_HEX` | fresh key | Recipient key policy |
| `FFMPEG` | `MIST_FFMPEG` | `ffmpeg` | CLI twin and cuts |
| `VISQOL` | `MIST_VISQOL` | `visqol` | Perceptual tool |
| `PEAQ` | `MIST_PEAQ` | `peaq` | Perceptual tool |
| `GIT` | `MIST_GIT` | `git` | Commit describe |
| `PKG_CONFIG` | `MIST_PKG_CONFIG` | `pkg-config` | libav versions |
| `PYTHON` | — | `python3` | CNN trainer |
| `CNN_WARDEN` | — | `tools/cnn_warden/train.py` | Trainer script |
| `CNN_EXPORT` | `MIST_HARNESS_EXPORT` | `$(HARNESS_OUT)/export` | Chunk export |
| `CNN_OUT` | `MIST_HARNESS_CNN` | `$(HARNESS_OUT)/cnn.json` | CNN scores |

`internal/quality` now honours `MIST_FFMPEG` / `MIST_VISQOL` / `MIST_PEAQ`
the same way, so a CI image with tools outside `PATH` does not need source
edits.

### 4. Git ignore

`harness-out/`, `export/`, `grok-measure/`, `keys/`, common audio suffixes
and `*.clean.i32` / `*.stego.i32` are ignored. Local carriers and CNN
exports cannot be added by accident. Committed testdata does not rely on
checked-in audio files.

`GROK-D-ANALYSIS-PLAN.md` still contains a machine-local corpus path. It
was left unstaged on purpose.

### 5. Documentation reconciliation

`DESIGN.md`, `REPORT.md`, `CLAUDE.md`, `README.md`,
`internal/steganalysis/README.md` and `tools/cnn_warden/README.md` now say
what is historical and what is current:

- Shipped density is 1% with STC, not the design doc's 2%.
- Lossless outputs and the harness exist; §5 of `DESIGN.md` is a snapshot.
- `REPORT.md` is an earlier 84-carrier run, not bound to a Phase 0
  manifest, and still uses original titles. Later runs must not.
- AUC-to-ε is a detector-implied KL lower bound, not Cachin security.
- Docs use `$CORPUS` and the example manifest. They do not mention
  `~/music` or any path on this machine.

This is marking, not rewriting history. The old numbers stay, with a date
and a warning.

## What this does **not** change

- Embedder behaviour, density, STC, silence rules, Vorbis substitution.
- Output audio files a user of `mist embed` would produce.
- Detector mathematics (Phase 1) or richer wardens (Phase 4).
- The known-cover non-goal.

A file produced by Mist today sounds and looks the same as before this
step. Only the *record of a measurement* is more honest and safer to
publish.

## Tests that lock the policy

- Manifest carriers must have a path-free public id and cannot escape the
  corpus root.
- A walk of `private-title.flac` reports `carrier-0001`, not the title.
- Skip reasons and load errors cannot retain `HOME` or the real path.
- `testdata/harness/corpus.example.json` parses and exposes only public
  names.
- A report directory always contains `manifest.json` and `scores.json`.
- Quality tools honour configured binary paths.

## How to run a publishable measurement

```bash
make harness \
  CORPUS="$CORPUS" \
  CORPUS_MANIFEST=testdata/harness/corpus.example.json \
  CORPUS_NAME=example-public-corpus \
  HARNESS_OUT="$HARNESS_OUT" \
  FORMATS=flac
```

Copy `manifest.json` and `scores.json` with the markdown if the numbers
will be cited. Do not copy the corpus.

## Follow-on

Phase 1 can now change bootstrap and grouping without arguing about which
commit a table came from. Phase 3 (ffmpeg fingerprint) should still treat
the Vorbis nominal-rate mismatch in the historical `REPORT.md` as an open
tool-level distinguisher, not as embedding evidence.

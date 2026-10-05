# CNN warden

A learned detector for the Mist harness. The harness's own detectors use
fixed features; this one learns its own, which gives a tighter lower bound
on how detectable Mist output is. It is not part of `make test` or CI.

## What it does

`make harness` with `MIST_HARNESS_EXPORT` set writes, for each carrier, the
values both its clean ffmpeg copy and its stego copy give the detectors
(the first 16 chunks of 65,536 values). `train.py` cuts those into
segments of 8,192 values, feeds a small 1-D CNN the clipped first and second
differences and the value's parity, and trains it to tell stego from clean.
It is scored like the other detectors: five folds split by lineage, falling
back to carriers when the export has one lineage, one score per file (the
mean over its segments), and a 95% interval bootstrapped by carrier. The
result goes to `cnn.json`, which the next `make harness` run picks up and
prints in the report.

## Run

Needs Python 3 with `torch` and `numpy`.

```bash
make cnn-warden CORPUS="$CORPUS" CORPUS_MANIFEST=testdata/harness/corpus.example.json FORMATS=ogg,flac,wav
```

That runs the harness with the export on, then `train.py` over what it wrote,
then the harness once more to put the result in the report. Every path is an
override (`PYTHON`, `CNN_WARDEN`, `CNN_EXPORT`, `CNN_OUT`, `HARNESS_OUT`).
By hand:

```bash
MIST_HARNESS_EXPORT="$HARNESS_OUT/export" make harness CORPUS="$CORPUS" CORPUS_MANIFEST=testdata/harness/corpus.example.json FORMATS=flac
"$PYTHON" "$CNN_WARDEN" "$HARNESS_OUT/export" --out "$HARNESS_OUT/cnn.json"
```

| Flag | Default | Meaning |
|---|---|---|
| `--out` | required | `cnn.json` to write, in the harness output directory |
| `--formats` | all | export subdirectories to score, such as `flac` or `ogg-vorbis` |
| `--folds` | 5 | cross-validation folds, split by lineage |
| `--epochs` | 6 | training epochs per fold |
| `--length` | 8192 | values per segment |
| `--rounds` | 1000 | bootstrap rounds for the interval |
| `--seed` | 1 | seed for folds, training and the bootstrap |
| `--arch` | `waveform` | `waveform`, `spectrogram`, `hybrid`, `residue`, or `all` |
| `--stereo` | off | score even/odd samples as left, right, mid and side |
| `--self-test` | off | train `waveform` on synthetic LSB replacement; exit non-zero below AUC 0.8 |

`waveform` is the net the report's primary CNN row uses. `all` keeps that row and adds the other three under `held_out`, scored on the same lineage folds, so a louder architecture is not silently selected. `residue` is the waveform net run on residue indices; a Vorbis export already is those indices. `hybrid` is a convolution plus an attention pool. Spectrogram is a short-time magnitude net. wav2vec2, AST, SSAST and BEATs are not in this suite: no positive control was run for them, and a transformer is not assumed to be stronger.

`--self-test` needs no export directory. It is the waveform net's positive control, not a measurement of Mist.

Training is slow on a CPU: with 84 carriers expect roughly 30 minutes per
format at 4 epochs. `cnn.json` is rewritten after each format, so a run that
is cut short keeps the formats it finished.

## Limits

It is a small network trained on a few thousand segments. An AUC at chance
means this network found nothing, not that nothing is there; a deeper
network or more carriers can do better. Its result belongs next to the
other detectors in the report as one more lower bound.

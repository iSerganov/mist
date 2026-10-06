# Mist harness report

> **Schema 6 measurement, 2026-10-06.**
>
> Corpus name `metal-dev`: the same 84 carriers, first 300 seconds of each, 84 independent lineage groups, one category (`external`), seed 1. Both runs used identical corpus, cap, folds and wardens, so the two are paired carrier for carrier. Public ids are `carrier-NNNN`. Perceptual tools were not installed.
>
> The confirmatory family has six members: HCF-COM, classifier, Markov, key-aware, rich and selection. This is not a Cachin ε and not a claim that the embedding is undetectable. The power simulation still asks for about 64 independent lineages (FLAC) before the worst cell is well powered; that gate is unmet.
>
> **The learned-warden (CNN) row is pending.** The CNN is being retrained on this run's export and will be attached on a re-render. FLAC is in (0.500, 0.494–0.504; the before-run was 0.499, 0.493–0.503 — chance in both).

## Headline file results

Stego against the canonical ffmpeg copy. D = 0.5 + |file AUC − 0.5|. The last column is the change in D from `38f5caa` to `355408c`, bootstrapped over the 84 paired carriers (1000 resamples, seed 1); an interval that excludes 0 is a change beyond run-to-run noise. Negative means less detectable.

| Format | Detector | File AUC (95%) | D | Adjusted p | Paired ΔD vs 38f5caa (95%) |
|---|---|---|---|---|---|
| flac | classifier | 0.552 (0.524–0.582) | 0.552 | Holm 0.600 | −0.004 (−0.015…+0.004) — n.s. |
| flac | markov | 0.539 (0.517–0.565) | 0.539 | Holm 0.600 | **−0.011 (−0.022…−0.002)** — improved |
| flac | key-aware | 0.506 (0.477–0.536) | 0.506 | Holm 1.000 | −0.030 (−0.060…+0.018) — n.s. |
| flac | cnn waveform | 0.500 (0.494–0.504) | 0.500 | — | ~0 (pending final attach) |
| wav/pcm_s16le | classifier | 0.504 (0.501–0.509) | 0.504 | Holm 0.500 | −0.002 (−0.009…+0.003) — n.s. |
| wav/pcm_s16le | key-aware | 0.500 (0.482–0.518) | 0.500 | Holm 1.000 | −0.012 (−0.024…+0.012) — n.s. |
| ogg/vorbis | hcf-com | 0.499 (0.496–0.501) | 0.501 | Holm 0.870 | **−0.007 (−0.015…−0.007)** — improved |
| ogg/vorbis | markov | 0.500 (0.498–0.503) | 0.500 | Holm 0.600 | **−0.011 (−0.020…−0.010)** — improved |
| ogg/vorbis | classifier | 0.504 (0.502–0.506) | 0.504 | Holm 0.600 | **−0.005 (−0.012…−0.003)** — improved |
| ogg/vorbis | rich | 0.501 (0.499–0.502) | 0.501 | Holm 1.000 | **−0.006 (−0.014…−0.004)** — improved |
| ogg/vorbis | rs | 0.490 (0.475–0.500) | 0.510 | BH 0.630 | +0.009 (−0.001…+0.024) — n.s. |

## What the change moved

**Improved, beyond noise:**

- **Vorbis** was the clearest win. The before-run's only Holm-significant detection — hcf-com at Holm 0.03 — is gone (Holm 0.87), and the paired ΔD excludes 0 for hcf-com, Markov, classifier and rich. All four Vorbis detectors now sit at chance. This tracks the Vorbis change: ranking a flip by vector error over the entry's own energy, and penalising entries with few legal substitutes.
- **FLAC Markov** fell from D 0.551 to 0.539, ΔD −0.011 (−0.022…−0.002). Markov reads second-difference curvature, which is exactly what the new lossless cost prices a ±1 against, so the drop is in the predicted place. It is a modest dent, not a fix: the interval is still 0.517–0.565, excluding 0.5.

**Not moved (within noise):**

- **FLAC classifier, 0.552** (0.524–0.582) is now the worst cell. ΔD −0.004 with an interval that includes 0 — statistically unchanged. It still grows with audio (0.534 on the first 40 chunks → 0.552 on the whole file) and with pooled files. The cost function steers *where* changes land but not *how many*; the 8× price cap limits how far it can steer.
- **FLAC key-aware** moved from 0.536 to 0.506, which looks large but the paired interval (−0.060…+0.018) includes 0, and key-aware does not depend on the embedder change — this is run-to-run variance on a wide interval.
- **WAV**: every detector was already at or near chance and stayed there.
- **Vorbis rs** drifted the wrong way (ΔD +0.009) but the interval includes 0; not a real regression.

**Not addressed by this change (and now the dominant signal):**

- **Pipeline fingerprint.** Mist-clean still differs from the canonical ffmpeg encode on 25/84 (FLAC), 29/84 (WAV, tags + zero tail) and 55/84 (Vorbis) carriers, all on metadata tags. The audio-blind metadata warden separates canonical ffmpeg from Mist-clean at file AUC 0.940 (FLAC), 0.743 (WAV) and 0.917 (Vorbis) — unchanged, because this commit does not touch the container/metadata path. On FLAC the embedding side-effect (Mist-clean vs stego) is also still 0.506 (0.503–0.510), excluding 0.5.

**Quality cost of the change:**

- FLAC and WAV: embedding cost unchanged (≤0.01 dB; outputs ≥71 dB SDR).
- Vorbis: the embedded error moved from 55.03 dB to 52.43 dB below the music (about 1.8× the error energy), and the embedding cost from 0.01 dB to 0.02 dB SDR. Still well inside the 0.3 dB target — moving flips onto louder entries raises their absolute error. This is the deliberate trade for the Vorbis detectability win above.

## Comparison summary

| Area | Before `38f5caa` | After `355408c` | Verdict |
|---|---|---|---|
| Vorbis detectors (worst Holm) | hcf-com Holm 0.03 (significant) | hcf-com Holm 0.87 (chance) | **improved** |
| Vorbis hcf-com / markov / classifier / rich | D 0.508 / 0.511 / 0.509 / 0.507 | D 0.501 / 0.500 / 0.504 / 0.501 | **improved** (paired, all exclude 0) |
| FLAC markov | D 0.551 | D 0.539 | **improved** (paired excludes 0) |
| FLAC classifier (worst cell overall) | D 0.556 | D 0.552 | unchanged (n.s.) |
| WAV detectors | ≤ D 0.512 | ≤ D 0.509 | unchanged (already ~chance) |
| Metadata fingerprint (FLAC/WAV/Vorbis clean-vs-canonical) | 0.940 / 0.743 / 0.917 | 0.940 / 0.743 / 0.917 | **unchanged — not addressed** |
| FLAC embedding side-effect on metadata | 0.505 | 0.506 | unchanged |
| Vorbis embedded error below music | 55.03 dB | 52.43 dB | regressed, still within target |
| FLAC / WAV embedding SDR cost | ≤ 0.02 dB | ≤ 0.01 dB | unchanged |

**Bottom line:** the branch lowered detectability where it set out to — Vorbis (all four learned/statistical detectors back to chance) and the FLAC Markov warden — at the cost of a small, in-budget drop in Vorbis embedding SDR. It did not move the FLAC classifier (0.552, still the worst detector-side cell) and does not touch the metadata pipeline fingerprint (0.74–0.94), which is now the largest separator between Mist and a plain ffmpeg encode and is independent of the embedding.

The sections below are the harness report this measurement is copied from.

Commit `355408c-dirty` · 2026-10-06 · corpus: metal-dev, first 300 s of each carrier (84 carriers, 84 independent lineage groups) · perceptual metric: not installed · compared with commit `38f5caa-dirty`

Each carrier is compared as four files: ffmpeg at its defaults, ffmpeg at the canonical workflow, Mist with nothing embedded, and Mist with a hidden message. For Vorbis the canonical workflow is `ffmpeg -q:a N` at the level Mist chose. For a lossless codec it is ffmpeg's defaults, so those two files are the same encode. The fingerprint verdict is Mist against the canonical file. Default ffmpeg is a second threat model and does not decide that verdict. Detectors for Vorbis use the canonical file, so the quality setting is not the signal. [How to read this report](#how-to-read-this-report) explains every number and threshold.

## Summary

| Format | Carriers | Canonical ffmpeg? | Metadata warden? | Hidden from detectors? | Audio quality |
|---|---|---|---|---|---|
| flac | 84 / 84 | ❌ differs from the canonical ffmpeg encode on 25 of 84 carriers: metadata tags | ❌ metadata interval excludes 0.5: canonical-vs-mist-clean, mist-clean-vs-stego | ⚠️ faint signal from chi-square (AUC 0.500) | ✅ inaudible: the added error is at least 94.06 dB below the music |
| wav/pcm_s16le | 84 / 84 | ❌ differs from the canonical ffmpeg encode on 29 of 84 carriers: metadata tags, zero tail | ❌ metadata interval excludes 0.5: canonical-vs-mist-clean | ⚠️ faint signal from hcf-com (AUC 0.500) | ✅ inaudible: the added error is at least 83.96 dB below the music |
| ogg/vorbis | 84 / 84 | ❌ differs from the canonical ffmpeg encode on 55 of 84 carriers: metadata tags | ❌ metadata interval excludes 0.5: canonical-vs-mist-clean | ⚠️ faint signal from hcf-com (AUC 0.500) | ✅ Mist's own error sits 52.43 dB below the music; embedding costs 0.02 dB, within the 0.3 dB target |

## flac

Measured on 84 of 84 carriers.

### Does it match the canonical ffmpeg encode?

Canonical means `ffmpeg -q:a N` at the level Mist chose for Vorbis, and ffmpeg's own defaults for a lossless codec. The verdict fails when Mist-clean or Mist-stego differs from that file on an identity field (length, trailing zeros, sample format, nominal rate, tags, FLAC STREAMINFO, Ogg granule and serial count). File size and packet size are left to the metadata warden. Ogg serial values are random and are not compared. Default ffmpeg is the next sentence, and it is a different threat model.

| Carrier | Source samples | Samples (canonical / Mist) | Zero tail (canonical / Mist) | Sample format (canonical / Mist) | Nominal kbps (canonical / Mist) | Canonical differs | Default ffmpeg differs |
|---|---|---|---|---|---|---|---|
| carrier-0001 | 13230191 | 13230191 / 13230191 | 0 / 0 | s32 / s32 | 0 / 0 | metadata tags | metadata tags |
| carrier-0002 | 13230191 | 13230191 / 13230191 | 0 / 0 | s32 / s32 | 0 / 0 | metadata tags | metadata tags |
| carrier-0003 | 13230191 | 13230191 / 13230191 | 0 / 0 | s32 / s32 | 0 / 0 | metadata tags | metadata tags |
| carrier-0004 | 11669231 | 11669231 / 11669231 | 0 / 0 | s32 / s32 | 0 / 0 | metadata tags | metadata tags |
| carrier-0005 | 13230191 | 13230191 / 13230191 | 0 / 0 | s32 / s32 | 0 / 0 | metadata tags | metadata tags |
| carrier-0006 | 13230191 | 13230191 / 13230191 | 0 / 0 | s32 / s32 | 0 / 0 | metadata tags | metadata tags |
| carrier-0007 | 13230191 | 13230191 / 13230191 | 0 / 0 | s32 / s32 | 0 / 0 | metadata tags | metadata tags |
| carrier-0008 | 13230191 | 13230191 / 13230191 | 0 / 0 | s32 / s32 | 0 / 0 | metadata tags | metadata tags |
| carrier-0009 | 11631215 | 11631215 / 11631215 | 288 / 288 | s32 / s32 | 0 / 0 | metadata tags | metadata tags |
| carrier-0010 | 12556271 | 12556271 / 12556271 | 30761 / 30761 | s32 / s32 | 0 / 0 | metadata tags | metadata tags |
| carrier-0011 | 13230767 | 13230767 / 13230767 | 0 / 0 | s32 / s32 | 0 / 0 | metadata tags | metadata tags |
| carrier-0012 | 13230767 | 13230767 / 13230767 | 0 / 0 | s32 / s32 | 0 / 0 | metadata tags | metadata tags |
| carrier-0013 | 13230767 | 13230767 / 13230767 | 0 / 0 | s32 / s32 | 0 / 0 | metadata tags | metadata tags |
| carrier-0014 | 13230767 | 13230767 / 13230767 | 0 / 0 | s32 / s32 | 0 / 0 | metadata tags | metadata tags |
| carrier-0015 | 2880000 | 2880000 / 2880000 | 0 / 0 | s16 / s16 | 0 / 0 | — | — |
| carrier-0016 | 5760000 | 5760000 / 5760000 | 0 / 0 | s16 / s16 | 0 / 0 | — | — |
| carrier-0017 | 5760000 | 5760000 / 5760000 | 0 / 0 | s16 / s16 | 0 / 0 | — | — |
| carrier-0018 | 28803072 | 28803072 / 28803072 | 0 / 0 | s32 / s32 | 0 / 0 | metadata tags | metadata tags |
| carrier-0019 | 28803072 | 28803072 / 28803072 | 0 / 0 | s32 / s32 | 0 / 0 | metadata tags | metadata tags |
| carrier-0020 | 27650560 | 27650560 / 27650560 | 0 / 0 | s32 / s32 | 0 / 0 | metadata tags | metadata tags |
| carrier-0021 | 27998720 | 27998720 / 27998720 | 0 / 0 | s32 / s32 | 0 / 0 | metadata tags | metadata tags |
| carrier-0022 | 24209920 | 24209920 / 24209920 | 0 / 0 | s32 / s32 | 0 / 0 | metadata tags | metadata tags |
| carrier-0023 | 28803072 | 28803072 / 28803072 | 0 / 0 | s32 / s32 | 0 / 0 | metadata tags | metadata tags |
| carrier-0024 | 24574720 | 24574720 / 24574720 | 0 / 0 | s32 / s32 | 0 / 0 | metadata tags | metadata tags |
| carrier-0025 | 26830080 | 26830080 / 26830080 | 0 / 0 | s32 / s32 | 0 / 0 | metadata tags | metadata tags |
| carrier-0026 | 24695040 | 24695040 / 24695040 | 0 / 0 | s32 / s32 | 0 / 0 | metadata tags | metadata tags |
| carrier-0027 | 18909440 | 18909440 / 18909440 | 0 / 0 | s32 / s32 | 0 / 0 | metadata tags | metadata tags |
| carrier-0028 | 28803072 | 28803072 / 28803072 | 0 / 0 | s32 / s32 | 0 / 0 | metadata tags | metadata tags |
| carrier-0029 | 7939176 | 7939176 / 7939176 | 0 / 0 | s16 / s16 | 0 / 0 | — | — |
| carrier-0030 | 8334312 | 8334312 / 8334312 | 0 / 0 | s16 / s16 | 0 / 0 | — | — |
| carrier-0031 | 8187312 | 8187312 / 8187312 | 0 / 0 | s16 / s16 | 0 / 0 | — | — |
| carrier-0032 | 7616952 | 7616952 / 7616952 | 0 / 0 | s16 / s16 | 0 / 0 | — | — |
| carrier-0033 | 7703976 | 7703976 / 7703976 | 0 / 0 | s16 / s16 | 0 / 0 | — | — |
| carrier-0034 | 7795116 | 7795116 / 7795116 | 0 / 0 | s16 / s16 | 0 / 0 | — | — |
| carrier-0035 | 8786484 | 8786484 / 8786484 | 0 / 0 | s16 / s16 | 0 / 0 | — | — |
| carrier-0036 | 8306676 | 8306676 / 8306676 | 0 / 0 | s16 / s16 | 0 / 0 | — | — |
| carrier-0037 | 7848624 | 7848624 / 7848624 | 0 / 0 | s16 / s16 | 0 / 0 | — | — |
| carrier-0038 | 7849212 | 7849212 / 7849212 | 0 / 0 | s16 / s16 | 0 / 0 | — | — |
| carrier-0039 | 8371356 | 8371356 / 8371356 | 0 / 0 | s16 / s16 | 0 / 0 | — | — |
| carrier-0040 | 7808052 | 7808052 / 7808052 | 0 / 0 | s16 / s16 | 0 / 0 | — | — |
| carrier-0041 | 7792764 | 7792764 / 7792764 | 0 / 0 | s16 / s16 | 0 / 0 | — | — |
| carrier-0042 | 7705152 | 7705152 / 7705152 | 0 / 0 | s16 / s16 | 0 / 0 | — | — |
| carrier-0043 | 7733376 | 7733376 / 7733376 | 0 / 0 | s16 / s16 | 0 / 0 | — | — |
| carrier-0044 | 8332548 | 8332548 / 8332548 | 0 / 0 | s16 / s16 | 0 / 0 | — | — |
| carrier-0045 | 7836864 | 7836864 / 7836864 | 0 / 0 | s16 / s16 | 0 / 0 | — | — |
| carrier-0046 | 7783356 | 7783356 / 7783356 | 0 / 0 | s16 / s16 | 0 / 0 | — | — |
| carrier-0047 | 8134980 | 8134980 / 8134980 | 0 / 0 | s16 / s16 | 0 / 0 | — | — |
| carrier-0048 | 7904484 | 7904484 / 7904484 | 0 / 0 | s16 / s16 | 0 / 0 | — | — |
| carrier-0049 | 7577556 | 7577556 / 7577556 | 0 / 0 | s16 / s16 | 0 / 0 | — | — |
| carrier-0050 | 8433684 | 8433684 / 8433684 | 0 / 0 | s16 / s16 | 0 / 0 | — | — |
| carrier-0051 | 8167320 | 8167320 / 8167320 | 0 / 0 | s16 / s16 | 0 / 0 | — | — |
| carrier-0052 | 7443492 | 7443492 / 7443492 | 0 / 0 | s16 / s16 | 0 / 0 | — | — |
| carrier-0053 | 7847448 | 7847448 / 7847448 | 0 / 0 | s16 / s16 | 0 / 0 | — | — |
| carrier-0054 | 6560316 | 6560316 / 6560316 | 20 / 20 | s16 / s16 | 0 / 0 | — | — |
| carrier-0055 | 7567536 | 7567536 / 7567536 | 1004 / 1004 | s32 / s32 | 0 / 0 | — | — |
| carrier-0056 | 10716829 | 10716829 / 10716829 | 2375 / 2375 | s32 / s32 | 0 / 0 | — | — |
| carrier-0057 | 10753254 | 10753254 / 10753254 | 462 / 462 | s32 / s32 | 0 / 0 | — | — |
| carrier-0058 | 8507078 | 8507078 / 8507078 | 0 / 0 | s32 / s32 | 0 / 0 | — | — |
| carrier-0059 | 9099131 | 9099131 / 9099131 | 0 / 0 | s32 / s32 | 0 / 0 | — | — |
| carrier-0060 | 10184328 | 10184328 / 10184328 | 0 / 0 | s32 / s32 | 0 / 0 | — | — |
| carrier-0061 | 9848955 | 9848955 / 9848955 | 0 / 0 | s32 / s32 | 0 / 0 | — | — |
| carrier-0062 | 10444219 | 10444219 / 10444219 | 1522 / 1522 | s32 / s32 | 0 / 0 | — | — |
| carrier-0063 | 10854001 | 10854001 / 10854001 | 0 / 0 | s32 / s32 | 0 / 0 | — | — |
| carrier-0064 | 9693408 | 9693408 / 9693408 | 0 / 0 | s32 / s32 | 0 / 0 | — | — |
| carrier-0065 | 9766958 | 9766958 / 9766958 | 0 / 0 | s32 / s32 | 0 / 0 | — | — |
| carrier-0066 | 9359938 | 9359938 / 9359938 | 0 / 0 | s32 / s32 | 0 / 0 | — | — |
| carrier-0067 | 9855000 | 9855000 / 9855000 | 8479 / 8479 | s32 / s32 | 0 / 0 | — | — |
| carrier-0068 | 9146026 | 9146026 / 9146026 | 276 / 276 | s32 / s32 | 0 / 0 | — | — |
| carrier-0069 | 10717286 | 10717286 / 10717286 | 0 / 0 | s32 / s32 | 0 / 0 | — | — |
| carrier-0070 | 8370000 | 8370000 / 8370000 | 0 / 0 | s32 / s32 | 0 / 0 | — | — |
| carrier-0071 | 10456119 | 10456119 / 10456119 | 23958 / 23958 | s32 / s32 | 0 / 0 | — | — |
| carrier-0072 | 9573914 | 9573914 / 9573914 | 7702 / 7702 | s32 / s32 | 0 / 0 | — | — |
| carrier-0073 | 10171848 | 10171848 / 10171848 | 826 / 826 | s32 / s32 | 0 / 0 | — | — |
| carrier-0074 | 9735652 | 9735652 / 9735652 | 0 / 0 | s32 / s32 | 0 / 0 | — | — |
| carrier-0075 | 9412488 | 9412488 / 9412488 | 7538 / 7538 | s32 / s32 | 0 / 0 | — | — |
| carrier-0076 | 9888000 | 9888000 / 9888000 | 0 / 0 | s32 / s32 | 0 / 0 | — | — |
| carrier-0077 | 10440000 | 10440000 / 10440000 | 0 / 0 | s32 / s32 | 0 / 0 | — | — |
| carrier-0078 | 11520002 | 11520002 / 11520002 | 0 / 0 | s32 / s32 | 0 / 0 | — | — |
| carrier-0079 | 10450747 | 10450747 / 10450747 | 21505 / 21505 | s32 / s32 | 0 / 0 | — | — |
| carrier-0080 | 8598259 | 8598259 / 8598259 | 0 / 0 | s32 / s32 | 0 / 0 | — | — |
| carrier-0081 | 11520000 | 11520000 / 11520000 | 0 / 0 | s32 / s32 | 0 / 0 | — | — |
| carrier-0082 | 11102609 | 11102609 / 11102609 | 0 / 0 | s32 / s32 | 0 / 0 | — | — |
| carrier-0083 | 8540309 | 8540309 / 8540309 | 7055 / 7055 | s32 / s32 | 0 / 0 | — | — |
| carrier-0084 | 10279847 | 10279847 / 10279847 | 7063 / 7063 | s32 / s32 | 0 / 0 | — | — |

**Verdict:** ❌ differs from the canonical ffmpeg encode on 25 of 84 carriers: metadata tags.

differs from ffmpeg defaults on 25 of 84 carriers: metadata tags. That is a separate threat model, not the canonical verdict.

### Audio-blind metadata

This classifier sees ffprobe fields, container summaries and file size. It does not see samples. Two questions are scored apart: canonical ffmpeg against Mist-clean (a pipeline fingerprint) and Mist-clean against Mist-stego (an embedding side effect). The preregistered pass is a file interval that includes 0.5. An interval that excludes 0.5 is a failed objective.

| Question | File AUC (95%) | D | Interval includes 0.5 |
|---|---|---|---|
| canonical-vs-mist-clean | 0.940 (0.900–0.970) | 0.940 | no |
| mist-clean-vs-stego | 0.506 (0.503–0.510) | 0.506 | no |

### Can a detector tell?

Stego copy against the canonical ffmpeg copy, so the quality setting is not the signal.

| Detector | Looks for | Chunk AUC (95%) | File AUC (95%) | D | Detector-implied benchmark KL lower bound (nats, 95%) | Adjusted p | Verdict | Message size | AUC change |
|---|---|---|---|---|---|---|---|---|---|
| chi-square | bit overwriting (not what Mist does) | 0.500 (0.500–0.500) | 0.502 (0.500–0.504) | 0.502 | 0.000 (0.000–0.000) | BH 0.015 | ⚠️ faint signal | ✅ hidden (0.500) | -0.000 |
| spa | bit overwriting (not what Mist does) | 0.500 (0.500–0.501) | 0.499 (0.494–0.504) | 0.501 | 0.000 (0.000–0.000) | BH 0.780 | ✅ chance | ✅ hidden (0.500) | -0.000 |
| rs | bit overwriting (not what Mist does) | 0.500 (0.499–0.500) | 0.501 (0.496–0.506) | 0.501 | 0.000 (0.000–0.000) | BH 0.780 | ✅ chance | ✅ hidden (0.500) | -0.000 |
| hcf-com | ±1 changes, which is what Mist does | 0.500 (0.500–0.500) | 0.500 (0.499–0.502) | 0.500 | 0.000 (0.000–0.000) | Holm 1.000 | ✅ chance | ✅ hidden (0.500) | -0.000 |
| classifier | anything it can learn from Mist's own output | 0.506 (0.502–0.510) | 0.552 (0.524–0.582) | 0.552 | 0.005 (0.001–0.013) | Holm 0.600 | ⚠️ faint signal | ✅ hidden (0.500) | -0.001 |
| markov | how the waveform's curvature changes from sample to sample | 0.503 (0.501–0.507) | 0.539 (0.517–0.565) | 0.539 | 0.003 (0.001–0.008) | Holm 0.600 | ⚠️ faint signal | ✅ hidden (0.500) | -0.000 |
| rich | a frozen summary of prediction error, co-occurrence and a short spectrum | 0.500 (0.500–0.500) | 0.501 (0.500–0.502) | 0.501 | 0.000 (0.000–0.000) | Holm 1.000 | ✅ chance | ✅ hidden (0.500) | -0.000 |
| key-aware | the ephemeral key in the first frame's envelope, read with the public key alone | 0.506 (0.477–0.536) | 0.506 (0.477–0.536) | 0.506 | 0.000 (0.000–0.003) | Holm 1.000 | ✅ chance | ✅ hidden (0.500) | -0.030 |
| selection | LSB bias at positions the recipient public key implies | 0.503 (0.492–0.513) | 0.503 (0.492–0.513) | 0.503 | 0.000 (0.000–0.000) | Holm 1.000 | ✅ chance | ✅ hidden (0.500) | +0.001 |

Power: shifting this run's classifier file scores to D = 0.55, about 64 independent lineages give 90% power for a lineage-cluster interval to exclude 0.5. The figure is a simulation from this run's dispersion.

Worst cell: classifier on aggregate, file AUC 0.552, D 0.552.

**Verdict:** ⚠️ faint signal from chi-square (AUC 0.500).

### The embedding alone

Stego copy against Mist's own re-encode, so only the embedded changes differ.

| Detector | Looks for | Chunk AUC (95%) | File AUC (95%) | D | Detector-implied benchmark KL lower bound (nats, 95%) | Adjusted p | Verdict | Message size |
|---|---|---|---|---|---|---|---|---|
| chi-square | bit overwriting (not what Mist does) | 0.500 (0.500–0.500) | 0.502 (0.500–0.504) | 0.502 | 0.000 (0.000–0.000) | — | ⚠️ faint signal | ✅ hidden (0.500) |
| spa | bit overwriting (not what Mist does) | 0.500 (0.500–0.501) | 0.499 (0.494–0.504) | 0.501 | 0.000 (0.000–0.000) | — | ✅ chance | ✅ hidden (0.500) |
| rs | bit overwriting (not what Mist does) | 0.500 (0.499–0.500) | 0.501 (0.496–0.506) | 0.501 | 0.000 (0.000–0.000) | — | ✅ chance | ✅ hidden (0.500) |
| hcf-com | ±1 changes, which is what Mist does | 0.500 (0.500–0.500) | 0.500 (0.499–0.502) | 0.500 | 0.000 (0.000–0.000) | — | ✅ chance | ✅ hidden (0.500) |
| classifier | anything it can learn from Mist's own output | 0.506 (0.502–0.510) | 0.552 (0.524–0.582) | 0.552 | 0.005 (0.001–0.013) | — | ⚠️ faint signal | ✅ hidden (0.500) |
| markov | how the waveform's curvature changes from sample to sample | 0.503 (0.501–0.507) | 0.539 (0.517–0.565) | 0.539 | 0.003 (0.001–0.008) | — | ⚠️ faint signal | ✅ hidden (0.500) |
| rich | a frozen summary of prediction error, co-occurrence and a short spectrum | 0.500 (0.500–0.500) | 0.501 (0.500–0.502) | 0.501 | 0.000 (0.000–0.000) | — | ✅ chance | ✅ hidden (0.500) |

### Does detection grow with audio?

File AUC (95%) against the clean ffmpeg copy on the first chunks of each carrier. A chunk is 65,536 values, about 0.74 s of 44.1 kHz stereo.

| Detector | First 40 chunks | First 240 chunks | Whole file |
|---|---|---|---|
| chi-square | 0.502 (0.500–0.504) | 0.502 (0.500–0.503) | 0.502 (0.500–0.504) |
| spa | 0.500 (0.495–0.506) | 0.499 (0.494–0.504) | 0.499 (0.494–0.504) |
| rs | 0.499 (0.492–0.506) | 0.500 (0.495–0.506) | 0.501 (0.496–0.506) |
| hcf-com | 0.501 (0.500–0.501) | 0.500 (0.499–0.501) | 0.500 (0.499–0.502) |
| classifier | 0.534 (0.513–0.558) | 0.543 (0.520–0.570) | 0.552 (0.524–0.582) |
| markov | 0.530 (0.510–0.554) | 0.537 (0.516–0.562) | 0.539 (0.517–0.565) |
| rich | 0.501 (0.500–0.503) | 0.499 (0.498–0.501) | 0.501 (0.500–0.502) |

### Does detection grow with the number of files?

AUC when the warden averages a detector's score over k files drawn at random from the corpus, stego against clean. Draws overlap, so there is no interval.

| Detector | 1 file(s) | 3 file(s) | 6 file(s) |
|---|---|---|---|
| chi-square | 0.489 | 0.482 | 0.482 |
| spa | 0.527 | 0.504 | 0.516 |
| rs | 0.528 | 0.509 | 0.512 |
| hcf-com | 0.490 | 0.484 | 0.481 |
| classifier | 0.571 | 0.555 | 0.545 |
| markov | 0.549 | 0.546 | 0.543 |
| rich | 0.495 | 0.511 | 0.504 |

### How much does it cost in sound?

| Measure | Mean | Worst | What it means |
|---|---|---|---|
| Plain re-encode SDR | 65.46 dB | 35.91 dB | Loss of Mist's own re-encode, with nothing embedded |
| Mist output SDR | 86.73 dB | 35.91 dB | The same, with the message embedded |
| Embedding cost | 0.00 dB | 0.01 dB | Mist's own share; target ≤ 0.3 dB, design-doc reference 0.18 dB |
| Extra error energy | +0% | +0% | Embedding cost as error added on top of a plain re-encode |
| Mist's error below the music | 134.26 dB | 94.06 dB | Stego copy against Mist's own re-encode: what embedding alone adds |
| Embedding cost, earlier run | 0.00 dB | 0.02 dB | Same measure at commit being compared with |

**Verdict:** ✅ inaudible: the added error is at least 94.06 dB below the music.

### Per carrier

| Carrier | Output kbps | Plain re-encode SDR | Mist output SDR | Embedding cost | Mist's error below the music |
|---|---|---|---|---|---|
| carrier-0001 | 1594 | 128.79 dB | 128.78 dB | 0.01 dB | 152.16 dB |
| carrier-0002 | 1588 | 128.91 dB | 128.90 dB | 0.01 dB | 152.27 dB |
| carrier-0003 | 1576 | 130.61 dB | 130.61 dB | 0.01 dB | 153.85 dB |
| carrier-0004 | 1545 | 128.38 dB | 128.37 dB | 0.01 dB | 151.79 dB |
| carrier-0005 | 1564 | 129.30 dB | 129.29 dB | 0.01 dB | 152.64 dB |
| carrier-0006 | 1575 | 129.70 dB | 129.70 dB | 0.01 dB | 153.04 dB |
| carrier-0007 | 1598 | 130.09 dB | 130.09 dB | 0.01 dB | 153.38 dB |
| carrier-0008 | 1550 | 130.75 dB | 130.74 dB | 0.01 dB | 154.00 dB |
| carrier-0009 | 1531 | 128.88 dB | 128.87 dB | 0.01 dB | 152.27 dB |
| carrier-0010 | 1449 | 129.02 dB | 129.01 dB | 0.01 dB | 152.43 dB |
| carrier-0011 | 1517 | 64.22 dB | 64.22 dB | 0.00 dB | 155.25 dB |
| carrier-0012 | 1661 | 50.07 dB | 50.07 dB | 0.00 dB | 154.33 dB |
| carrier-0013 | 1552 | 46.00 dB | 46.00 dB | 0.00 dB | 152.57 dB |
| carrier-0014 | 1443 | 43.92 dB | 43.92 dB | 0.00 dB | 153.12 dB |
| carrier-0015 | 967 | ∞ | 103.07 dB | ∞ | 103.07 dB |
| carrier-0016 | 1045 | ∞ | 105.26 dB | ∞ | 105.26 dB |
| carrier-0017 | 978 | ∞ | 105.99 dB | ∞ | 105.99 dB |
| carrier-0018 | 2297 | ∞ | 134.65 dB | ∞ | 134.65 dB |
| carrier-0019 | 2258 | ∞ | 132.09 dB | ∞ | 132.09 dB |
| carrier-0020 | 2593 | ∞ | 150.07 dB | ∞ | 150.07 dB |
| carrier-0021 | 2648 | ∞ | 149.80 dB | ∞ | 149.80 dB |
| carrier-0022 | 2676 | ∞ | 151.31 dB | ∞ | 151.31 dB |
| carrier-0023 | 2583 | ∞ | 148.62 dB | ∞ | 148.62 dB |
| carrier-0024 | 2685 | ∞ | 149.63 dB | ∞ | 149.63 dB |
| carrier-0025 | 2698 | ∞ | 149.99 dB | ∞ | 149.99 dB |
| carrier-0026 | 2688 | ∞ | 150.48 dB | ∞ | 150.48 dB |
| carrier-0027 | 2661 | ∞ | 150.45 dB | ∞ | 150.45 dB |
| carrier-0028 | 2639 | ∞ | 149.96 dB | ∞ | 149.96 dB |
| carrier-0029 | 465 | ∞ | 95.56 dB | ∞ | 95.56 dB |
| carrier-0030 | 463 | ∞ | 95.26 dB | ∞ | 95.26 dB |
| carrier-0031 | 460 | ∞ | 95.16 dB | ∞ | 95.16 dB |
| carrier-0032 | 467 | ∞ | 95.07 dB | ∞ | 95.07 dB |
| carrier-0033 | 465 | ∞ | 95.58 dB | ∞ | 95.58 dB |
| carrier-0034 | 457 | ∞ | 94.89 dB | ∞ | 94.89 dB |
| carrier-0035 | 464 | ∞ | 95.10 dB | ∞ | 95.10 dB |
| carrier-0036 | 469 | ∞ | 95.48 dB | ∞ | 95.48 dB |
| carrier-0037 | 466 | ∞ | 94.86 dB | ∞ | 94.86 dB |
| carrier-0038 | 463 | ∞ | 95.09 dB | ∞ | 95.09 dB |
| carrier-0039 | 462 | ∞ | 94.87 dB | ∞ | 94.87 dB |
| carrier-0040 | 466 | ∞ | 94.88 dB | ∞ | 94.88 dB |
| carrier-0041 | 465 | ∞ | 95.25 dB | ∞ | 95.25 dB |
| carrier-0042 | 472 | ∞ | 95.31 dB | ∞ | 95.31 dB |
| carrier-0043 | 460 | ∞ | 94.06 dB | ∞ | 94.06 dB |
| carrier-0044 | 468 | ∞ | 94.93 dB | ∞ | 94.93 dB |
| carrier-0045 | 481 | ∞ | 95.99 dB | ∞ | 95.99 dB |
| carrier-0046 | 457 | ∞ | 95.35 dB | ∞ | 95.35 dB |
| carrier-0047 | 468 | ∞ | 95.38 dB | ∞ | 95.38 dB |
| carrier-0048 | 467 | ∞ | 95.53 dB | ∞ | 95.53 dB |
| carrier-0049 | 461 | ∞ | 95.16 dB | ∞ | 95.16 dB |
| carrier-0050 | 475 | ∞ | 95.24 dB | ∞ | 95.24 dB |
| carrier-0051 | 461 | ∞ | 95.14 dB | ∞ | 95.14 dB |
| carrier-0052 | 463 | ∞ | 95.05 dB | ∞ | 95.05 dB |
| carrier-0053 | 462 | ∞ | 94.97 dB | ∞ | 94.97 dB |
| carrier-0054 | 472 | ∞ | 94.95 dB | ∞ | 94.95 dB |
| carrier-0055 | 1874 | 46.77 dB | 46.77 dB | 0.00 dB | 157.67 dB |
| carrier-0056 | 1947 | 36.65 dB | 36.65 dB | 0.00 dB | 159.56 dB |
| carrier-0057 | 1832 | 45.62 dB | 45.62 dB | 0.00 dB | 157.07 dB |
| carrier-0058 | 1910 | 35.91 dB | 35.91 dB | 0.00 dB | 157.95 dB |
| carrier-0059 | 1936 | 44.31 dB | 44.31 dB | 0.00 dB | 157.85 dB |
| carrier-0060 | 1872 | 58.54 dB | 58.54 dB | 0.00 dB | 154.98 dB |
| carrier-0061 | 1854 | 41.20 dB | 41.20 dB | 0.00 dB | 157.99 dB |
| carrier-0062 | 1916 | 41.33 dB | 41.33 dB | 0.00 dB | 158.43 dB |
| carrier-0063 | 1929 | 43.97 dB | 43.97 dB | 0.00 dB | 158.91 dB |
| carrier-0064 | 1768 | 68.84 dB | 68.84 dB | 0.00 dB | 154.84 dB |
| carrier-0065 | 1860 | 52.48 dB | 52.48 dB | 0.00 dB | 157.00 dB |
| carrier-0066 | 1894 | 43.08 dB | 43.08 dB | 0.00 dB | 157.72 dB |
| carrier-0067 | 1965 | 46.30 dB | 46.30 dB | 0.00 dB | 158.60 dB |
| carrier-0068 | 1927 | 38.87 dB | 38.87 dB | 0.00 dB | 159.38 dB |
| carrier-0069 | 1849 | 54.46 dB | 54.46 dB | 0.00 dB | 157.61 dB |
| carrier-0070 | 1767 | 47.32 dB | 47.32 dB | 0.00 dB | 156.79 dB |
| carrier-0071 | 1892 | 43.18 dB | 43.18 dB | 0.00 dB | 156.91 dB |
| carrier-0072 | 1921 | 45.06 dB | 45.06 dB | 0.00 dB | 158.60 dB |
| carrier-0073 | 1851 | 49.41 dB | 49.41 dB | 0.00 dB | 155.86 dB |
| carrier-0074 | 1899 | 40.06 dB | 40.06 dB | 0.00 dB | 158.32 dB |
| carrier-0075 | 1774 | 44.09 dB | 44.09 dB | 0.00 dB | 157.75 dB |
| carrier-0076 | 1984 | 39.55 dB | 39.55 dB | 0.00 dB | 158.90 dB |
| carrier-0077 | 1917 | 48.44 dB | 48.44 dB | 0.00 dB | 158.28 dB |
| carrier-0078 | 1913 | 40.85 dB | 40.85 dB | 0.00 dB | 159.21 dB |
| carrier-0079 | 1932 | 45.88 dB | 45.88 dB | 0.00 dB | 157.76 dB |
| carrier-0080 | 1937 | 46.94 dB | 46.94 dB | 0.00 dB | 157.88 dB |
| carrier-0081 | 1898 | 48.47 dB | 48.47 dB | 0.00 dB | 157.46 dB |
| carrier-0082 | 1851 | 43.09 dB | 43.09 dB | 0.00 dB | 156.76 dB |
| carrier-0083 | 1902 | 59.77 dB | 59.77 dB | 0.00 dB | 155.08 dB |
| carrier-0084 | 1813 | 41.02 dB | 41.02 dB | 0.00 dB | 158.52 dB |

## wav/pcm_s16le

Measured on 84 of 84 carriers.

### Does it match the canonical ffmpeg encode?

Canonical means `ffmpeg -q:a N` at the level Mist chose for Vorbis, and ffmpeg's own defaults for a lossless codec. The verdict fails when Mist-clean or Mist-stego differs from that file on an identity field (length, trailing zeros, sample format, nominal rate, tags, FLAC STREAMINFO, Ogg granule and serial count). File size and packet size are left to the metadata warden. Ogg serial values are random and are not compared. Default ffmpeg is the next sentence, and it is a different threat model.

| Carrier | Source samples | Samples (canonical / Mist) | Zero tail (canonical / Mist) | Sample format (canonical / Mist) | Nominal kbps (canonical / Mist) | Canonical differs | Default ffmpeg differs |
|---|---|---|---|---|---|---|---|
| carrier-0001 | 13230191 | 13230191 / 13230191 | 0 / 0 | s16 / s16 | 1411 / 1411 | metadata tags | metadata tags |
| carrier-0002 | 13230191 | 13230191 / 13230191 | 0 / 0 | s16 / s16 | 1411 / 1411 | metadata tags | metadata tags |
| carrier-0003 | 13230191 | 13230191 / 13230191 | 0 / 0 | s16 / s16 | 1411 / 1411 | metadata tags | metadata tags |
| carrier-0004 | 11669231 | 11669231 / 11669231 | 0 / 16 | s16 / s16 | 1411 / 1411 | zero tail, metadata tags | zero tail, metadata tags |
| carrier-0005 | 13230191 | 13230191 / 13230191 | 0 / 0 | s16 / s16 | 1411 / 1411 | metadata tags | metadata tags |
| carrier-0006 | 13230191 | 13230191 / 13230191 | 0 / 0 | s16 / s16 | 1411 / 1411 | metadata tags | metadata tags |
| carrier-0007 | 13230191 | 13230191 / 13230191 | 0 / 0 | s16 / s16 | 1411 / 1411 | metadata tags | metadata tags |
| carrier-0008 | 13230191 | 13230191 / 13230191 | 0 / 0 | s16 / s16 | 1411 / 1411 | metadata tags | metadata tags |
| carrier-0009 | 11631215 | 11631215 / 11631215 | 300 / 300 | s16 / s16 | 1411 / 1411 | metadata tags | metadata tags |
| carrier-0010 | 12556271 | 12556271 / 12556271 | 30781 / 30781 | s16 / s16 | 1411 / 1411 | metadata tags | metadata tags |
| carrier-0011 | 13230767 | 13230767 / 13230767 | 0 / 0 | s16 / s16 | 1411 / 1411 | metadata tags | metadata tags |
| carrier-0012 | 13230767 | 13230767 / 13230767 | 0 / 0 | s16 / s16 | 1411 / 1411 | metadata tags | metadata tags |
| carrier-0013 | 13230767 | 13230767 / 13230767 | 0 / 0 | s16 / s16 | 1411 / 1411 | metadata tags | metadata tags |
| carrier-0014 | 13230767 | 13230767 / 13230767 | 0 / 0 | s16 / s16 | 1411 / 1411 | metadata tags | metadata tags |
| carrier-0015 | 2880000 | 2880000 / 2880000 | 0 / 0 | s16 / s16 | 1536 / 1536 | — | — |
| carrier-0016 | 5760000 | 5760000 / 5760000 | 0 / 0 | s16 / s16 | 1536 / 1536 | — | — |
| carrier-0017 | 5760000 | 5760000 / 5760000 | 0 / 0 | s16 / s16 | 1536 / 1536 | — | — |
| carrier-0018 | 28803072 | 28803072 / 28803072 | 0 / 0 | s16 / s16 | 3072 / 3072 | metadata tags | metadata tags |
| carrier-0019 | 28803072 | 28803072 / 28803072 | 0 / 0 | s16 / s16 | 3072 / 3072 | metadata tags | metadata tags |
| carrier-0020 | 27650560 | 27650560 / 27650560 | 0 / 0 | s16 / s16 | 3072 / 3072 | metadata tags | metadata tags |
| carrier-0021 | 27998720 | 27998720 / 27998720 | 0 / 0 | s16 / s16 | 3072 / 3072 | metadata tags | metadata tags |
| carrier-0022 | 24209920 | 24209920 / 24209920 | 0 / 0 | s16 / s16 | 3072 / 3072 | metadata tags | metadata tags |
| carrier-0023 | 28803072 | 28803072 / 28803072 | 0 / 0 | s16 / s16 | 3072 / 3072 | metadata tags | metadata tags |
| carrier-0024 | 24574720 | 24574720 / 24574720 | 0 / 0 | s16 / s16 | 3072 / 3072 | metadata tags | metadata tags |
| carrier-0025 | 26830080 | 26830080 / 26830080 | 0 / 0 | s16 / s16 | 3072 / 3072 | metadata tags | metadata tags |
| carrier-0026 | 24695040 | 24695040 / 24695040 | 0 / 0 | s16 / s16 | 3072 / 3072 | metadata tags | metadata tags |
| carrier-0027 | 18909440 | 18909440 / 18909440 | 0 / 0 | s16 / s16 | 3072 / 3072 | metadata tags | metadata tags |
| carrier-0028 | 28803072 | 28803072 / 28803072 | 0 / 0 | s16 / s16 | 3072 / 3072 | metadata tags | metadata tags |
| carrier-0029 | 7939176 | 7939176 / 7939176 | 0 / 0 | s16 / s16 | 1411 / 1411 | — | — |
| carrier-0030 | 8334312 | 8334312 / 8334312 | 0 / 0 | s16 / s16 | 1411 / 1411 | — | — |
| carrier-0031 | 8187312 | 8187312 / 8187312 | 0 / 0 | s16 / s16 | 1411 / 1411 | — | — |
| carrier-0032 | 7616952 | 7616952 / 7616952 | 0 / 0 | s16 / s16 | 1411 / 1411 | — | — |
| carrier-0033 | 7703976 | 7703976 / 7703976 | 0 / 0 | s16 / s16 | 1411 / 1411 | — | — |
| carrier-0034 | 7795116 | 7795116 / 7795116 | 0 / 0 | s16 / s16 | 1411 / 1411 | — | — |
| carrier-0035 | 8786484 | 8786484 / 8786484 | 0 / 0 | s16 / s16 | 1411 / 1411 | — | — |
| carrier-0036 | 8306676 | 8306676 / 8306676 | 0 / 0 | s16 / s16 | 1411 / 1411 | — | — |
| carrier-0037 | 7848624 | 7848624 / 7848624 | 0 / 0 | s16 / s16 | 1411 / 1411 | — | — |
| carrier-0038 | 7849212 | 7849212 / 7849212 | 0 / 0 | s16 / s16 | 1411 / 1411 | — | — |
| carrier-0039 | 8371356 | 8371356 / 8371356 | 0 / 0 | s16 / s16 | 1411 / 1411 | — | — |
| carrier-0040 | 7808052 | 7808052 / 7808052 | 0 / 0 | s16 / s16 | 1411 / 1411 | — | — |
| carrier-0041 | 7792764 | 7792764 / 7792764 | 0 / 0 | s16 / s16 | 1411 / 1411 | — | — |
| carrier-0042 | 7705152 | 7705152 / 7705152 | 0 / 0 | s16 / s16 | 1411 / 1411 | — | — |
| carrier-0043 | 7733376 | 7733376 / 7733376 | 0 / 0 | s16 / s16 | 1411 / 1411 | — | — |
| carrier-0044 | 8332548 | 8332548 / 8332548 | 0 / 0 | s16 / s16 | 1411 / 1411 | — | — |
| carrier-0045 | 7836864 | 7836864 / 7836864 | 0 / 0 | s16 / s16 | 1411 / 1411 | — | — |
| carrier-0046 | 7783356 | 7783356 / 7783356 | 0 / 0 | s16 / s16 | 1411 / 1411 | — | — |
| carrier-0047 | 8134980 | 8134980 / 8134980 | 0 / 0 | s16 / s16 | 1411 / 1411 | — | — |
| carrier-0048 | 7904484 | 7904484 / 7904484 | 0 / 0 | s16 / s16 | 1411 / 1411 | — | — |
| carrier-0049 | 7577556 | 7577556 / 7577556 | 0 / 0 | s16 / s16 | 1411 / 1411 | — | — |
| carrier-0050 | 8433684 | 8433684 / 8433684 | 0 / 0 | s16 / s16 | 1411 / 1411 | — | — |
| carrier-0051 | 8167320 | 8167320 / 8167320 | 0 / 0 | s16 / s16 | 1411 / 1411 | — | — |
| carrier-0052 | 7443492 | 7443492 / 7443492 | 0 / 0 | s16 / s16 | 1411 / 1411 | — | — |
| carrier-0053 | 7847448 | 7847448 / 7847448 | 0 / 0 | s16 / s16 | 1411 / 1411 | — | — |
| carrier-0054 | 6560316 | 6560316 / 6560316 | 20 / 20 | s16 / s16 | 1411 / 1411 | — | — |
| carrier-0055 | 7567536 | 7567536 / 7567536 | 2973 / 2973 | s16 / s16 | 1536 / 1536 | — | — |
| carrier-0056 | 10716829 | 10716829 / 10716829 | 2698 / 2698 | s16 / s16 | 1536 / 1536 | — | — |
| carrier-0057 | 10753254 | 10753254 / 10753254 | 466 / 466 | s16 / s16 | 1536 / 1536 | — | — |
| carrier-0058 | 8507078 | 8507078 / 8507078 | 3 / 3 | s16 / s16 | 1536 / 1536 | — | — |
| carrier-0059 | 9099131 | 9099131 / 9099131 | 0 / 0 | s16 / s16 | 1536 / 1536 | — | — |
| carrier-0060 | 10184328 | 10184328 / 10184328 | 1 / 1 | s16 / s16 | 1536 / 1536 | — | — |
| carrier-0061 | 9848955 | 9848955 / 9848955 | 0 / 0 | s16 / s16 | 1536 / 1536 | — | — |
| carrier-0062 | 10444219 | 10444219 / 10444219 | 1526 / 1526 | s16 / s16 | 1536 / 1536 | — | — |
| carrier-0063 | 10854001 | 10854001 / 10854001 | 0 / 0 | s16 / s16 | 1536 / 1536 | — | — |
| carrier-0064 | 9693408 | 9693408 / 9693408 | 1 / 0 | s16 / s16 | 1536 / 1536 | zero tail | zero tail |
| carrier-0065 | 9766958 | 9766958 / 9766958 | 1 / 1 | s16 / s16 | 1536 / 1536 | — | — |
| carrier-0066 | 9359938 | 9359938 / 9359938 | 1 / 1 | s16 / s16 | 1536 / 1536 | — | — |
| carrier-0067 | 9855000 | 9855000 / 9855000 | 8479 / 8479 | s16 / s16 | 1536 / 1536 | — | — |
| carrier-0068 | 9146026 | 9146026 / 9146026 | 280 / 280 | s16 / s16 | 1536 / 1536 | — | — |
| carrier-0069 | 10717286 | 10717286 / 10717286 | 3 / 3 | s16 / s16 | 1536 / 1536 | — | — |
| carrier-0070 | 8370000 | 8370000 / 8370000 | 1 / 1 | s16 / s16 | 1536 / 1536 | — | — |
| carrier-0071 | 10456119 | 10456119 / 10456119 | 23962 / 23962 | s16 / s16 | 1536 / 1536 | — | — |
| carrier-0072 | 9573914 | 9573914 / 9573914 | 8695 / 8695 | s16 / s16 | 1536 / 1536 | — | — |
| carrier-0073 | 10171848 | 10171848 / 10171848 | 830 / 830 | s16 / s16 | 1536 / 1536 | — | — |
| carrier-0074 | 9735652 | 9735652 / 9735652 | 0 / 0 | s16 / s16 | 1536 / 1536 | — | — |
| carrier-0075 | 9412488 | 9412488 / 9412488 | 7542 / 7542 | s16 / s16 | 1536 / 1536 | — | — |
| carrier-0076 | 9888000 | 9888000 / 9888000 | 0 / 0 | s16 / s16 | 1536 / 1536 | — | — |
| carrier-0077 | 10440000 | 10440000 / 10440000 | 2 / 0 | s16 / s16 | 1536 / 1536 | zero tail | zero tail |
| carrier-0078 | 11520002 | 11520002 / 11520002 | 0 / 0 | s16 / s16 | 1536 / 1536 | — | — |
| carrier-0079 | 10450747 | 10450747 / 10450747 | 21515 / 21515 | s16 / s16 | 1536 / 1536 | — | — |
| carrier-0080 | 8598259 | 8598259 / 8598259 | 5 / 4 | s16 / s16 | 1536 / 1536 | zero tail | zero tail |
| carrier-0081 | 11520000 | 11520000 / 11520000 | 0 / 0 | s16 / s16 | 1536 / 1536 | — | — |
| carrier-0082 | 11102609 | 11102609 / 11102609 | 2 / 1 | s16 / s16 | 1536 / 1536 | zero tail | zero tail |
| carrier-0083 | 8540309 | 8540309 / 8540309 | 7390 / 7390 | s16 / s16 | 1536 / 1536 | — | — |
| carrier-0084 | 10279847 | 10279847 / 10279847 | 7068 / 7068 | s16 / s16 | 1536 / 1536 | — | — |

**Verdict:** ❌ differs from the canonical ffmpeg encode on 29 of 84 carriers: metadata tags, zero tail.

differs from ffmpeg defaults on 29 of 84 carriers: metadata tags, zero tail. That is a separate threat model, not the canonical verdict.

### Audio-blind metadata

This classifier sees ffprobe fields, container summaries and file size. It does not see samples. Two questions are scored apart: canonical ffmpeg against Mist-clean (a pipeline fingerprint) and Mist-clean against Mist-stego (an embedding side effect). The preregistered pass is a file interval that includes 0.5. An interval that excludes 0.5 is a failed objective.

| Question | File AUC (95%) | D | Interval includes 0.5 |
|---|---|---|---|
| canonical-vs-mist-clean | 0.743 (0.674–0.810) | 0.743 | no |
| mist-clean-vs-stego | 0.500 (0.500–0.500) | 0.500 | yes |

### Can a detector tell?

Stego copy against the canonical ffmpeg copy, so the quality setting is not the signal.

| Detector | Looks for | Chunk AUC (95%) | File AUC (95%) | D | Detector-implied benchmark KL lower bound (nats, 95%) | Adjusted p | Verdict | Message size | AUC change |
|---|---|---|---|---|---|---|---|---|---|
| chi-square | bit overwriting (not what Mist does) | 0.500 (0.500–0.500) | 0.501 (0.499–0.502) | 0.501 | 0.000 (0.000–0.000) | BH 0.825 | ✅ chance | ✅ hidden (0.500) | -0.000 |
| spa | bit overwriting (not what Mist does) | 0.500 (0.500–0.501) | 0.500 (0.496–0.506) | 0.500 | 0.000 (0.000–0.000) | BH 0.830 | ✅ chance | ✅ hidden (0.500) | +0.000 |
| rs | bit overwriting (not what Mist does) | 0.500 (0.499–0.501) | 0.499 (0.493–0.505) | 0.501 | 0.000 (0.000–0.000) | BH 0.825 | ✅ chance | ✅ hidden (0.500) | -0.000 |
| hcf-com | ±1 changes, which is what Mist does | 0.500 (0.500–0.500) | 0.501 (0.499–0.502) | 0.501 | 0.000 (0.000–0.000) | Holm 0.680 | ⚠️ faint signal | ✅ hidden (0.500) | +0.000 |
| classifier | anything it can learn from Mist's own output | 0.502 (0.501–0.504) | 0.504 (0.501–0.509) | 0.504 | 0.000 (0.000–0.000) | Holm 0.500 | ⚠️ faint signal | ✅ hidden (0.500) | -0.000 |
| markov | how the waveform's curvature changes from sample to sample | 0.501 (0.500–0.501) | 0.503 (0.501–0.505) | 0.503 | 0.000 (0.000–0.000) | Holm 0.500 | ⚠️ faint signal | ✅ hidden (0.500) | -0.000 |
| rich | a frozen summary of prediction error, co-occurrence and a short spectrum | 0.500 (0.500–0.500) | 0.498 (0.497–0.499) | 0.502 | 0.000 (0.000–0.000) | Holm 0.500 | ⚠️ faint signal | ✅ hidden (0.500) | +0.000 |
| key-aware | the ephemeral key in the first frame's envelope, read with the public key alone | 0.500 (0.482–0.518) | 0.500 (0.482–0.518) | 0.500 | 0.000 (0.000–0.001) | Holm 1.000 | ✅ chance | ✅ hidden (0.500) | -0.012 |
| selection | LSB bias at positions the recipient public key implies | 0.509 (0.499–0.518) | 0.509 (0.499–0.518) | 0.509 | 0.000 (0.000–0.001) | Holm 0.270 | ✅ chance | ✅ hidden (0.500) | +0.002 |

Power: shifting this run's classifier file scores to D = 0.55, about 8 independent lineages give 90% power for a lineage-cluster interval to exclude 0.5. The figure is a simulation from this run's dispersion.

Worst cell: selection on aggregate, file AUC 0.509, D 0.509.

**Verdict:** ⚠️ faint signal from hcf-com (AUC 0.500).

### The embedding alone

Stego copy against Mist's own re-encode, so only the embedded changes differ.

| Detector | Looks for | Chunk AUC (95%) | File AUC (95%) | D | Detector-implied benchmark KL lower bound (nats, 95%) | Adjusted p | Verdict | Message size |
|---|---|---|---|---|---|---|---|---|
| chi-square | bit overwriting (not what Mist does) | 0.500 (0.500–0.500) | 0.501 (0.499–0.502) | 0.501 | 0.000 (0.000–0.000) | — | ✅ chance | ✅ hidden (0.500) |
| spa | bit overwriting (not what Mist does) | 0.500 (0.500–0.501) | 0.500 (0.495–0.506) | 0.500 | 0.000 (0.000–0.000) | — | ✅ chance | ✅ hidden (0.500) |
| rs | bit overwriting (not what Mist does) | 0.500 (0.499–0.501) | 0.498 (0.493–0.505) | 0.502 | 0.000 (0.000–0.000) | — | ✅ chance | ✅ hidden (0.500) |
| hcf-com | ±1 changes, which is what Mist does | 0.500 (0.500–0.500) | 0.501 (0.500–0.502) | 0.501 | 0.000 (0.000–0.000) | — | ⚠️ faint signal | ✅ hidden (0.500) |
| classifier | anything it can learn from Mist's own output | 0.502 (0.501–0.504) | 0.504 (0.501–0.509) | 0.504 | 0.000 (0.000–0.000) | — | ⚠️ faint signal | ✅ hidden (0.500) |
| markov | how the waveform's curvature changes from sample to sample | 0.501 (0.500–0.501) | 0.503 (0.501–0.505) | 0.503 | 0.000 (0.000–0.000) | — | ⚠️ faint signal | ✅ hidden (0.500) |
| rich | a frozen summary of prediction error, co-occurrence and a short spectrum | 0.500 (0.500–0.500) | 0.498 (0.497–0.499) | 0.502 | 0.000 (0.000–0.000) | — | ⚠️ faint signal | ✅ hidden (0.500) |

### Does detection grow with audio?

File AUC (95%) against the clean ffmpeg copy on the first chunks of each carrier. A chunk is 65,536 values, about 0.74 s of 44.1 kHz stereo.

| Detector | First 40 chunks | First 240 chunks | Whole file |
|---|---|---|---|
| chi-square | 0.500 (0.498–0.501) | 0.501 (0.498–0.503) | 0.501 (0.499–0.502) |
| spa | 0.504 (0.499–0.509) | 0.502 (0.497–0.507) | 0.500 (0.496–0.506) |
| rs | 0.502 (0.496–0.509) | 0.499 (0.495–0.504) | 0.499 (0.493–0.505) |
| hcf-com | 0.501 (0.500–0.503) | 0.501 (0.500–0.503) | 0.501 (0.499–0.502) |
| classifier | 0.503 (0.500–0.506) | 0.505 (0.501–0.511) | 0.504 (0.501–0.509) |
| markov | 0.501 (0.499–0.503) | 0.504 (0.502–0.507) | 0.503 (0.501–0.505) |
| rich | 0.500 (0.499–0.501) | 0.499 (0.497–0.500) | 0.498 (0.497–0.499) |

### Does detection grow with the number of files?

AUC when the warden averages a detector's score over k files drawn at random from the corpus, stego against clean. Draws overlap, so there is no interval.

| Detector | 1 file(s) | 3 file(s) | 6 file(s) |
|---|---|---|---|
| chi-square | 0.543 | 0.491 | 0.507 |
| spa | 0.490 | 0.513 | 0.501 |
| rs | 0.484 | 0.508 | 0.490 |
| hcf-com | 0.473 | 0.482 | 0.476 |
| classifier | 0.528 | 0.520 | 0.509 |
| markov | 0.530 | 0.512 | 0.499 |
| rich | 0.487 | 0.497 | 0.528 |

### How much does it cost in sound?

| Measure | Mean | Worst | What it means |
|---|---|---|---|
| Plain re-encode SDR | 58.45 dB | 35.91 dB | Loss of Mist's own re-encode, with nothing embedded |
| Mist output SDR | 71.46 dB | 35.91 dB | The same, with the message embedded |
| Embedding cost | 0.01 dB | 0.02 dB | Mist's own share; target ≤ 0.3 dB, design-doc reference 0.18 dB |
| Extra error energy | +0% | +0% | Embedding cost as error added on top of a plain re-encode |
| Mist's error below the music | 102.73 dB | 83.96 dB | Stego copy against Mist's own re-encode: what embedding alone adds |
| Embedding cost, earlier run | 0.01 dB | 0.02 dB | Same measure at commit being compared with |

**Verdict:** ✅ inaudible: the added error is at least 83.96 dB below the music.

### Per carrier

| Carrier | Output kbps | Plain re-encode SDR | Mist output SDR | Embedding cost | Mist's error below the music |
|---|---|---|---|---|---|
| carrier-0001 | 1411 | 79.97 dB | 79.95 dB | 0.02 dB | 103.99 dB |
| carrier-0002 | 1411 | 80.09 dB | 80.07 dB | 0.02 dB | 104.10 dB |
| carrier-0003 | 1411 | 81.66 dB | 81.64 dB | 0.02 dB | 105.69 dB |
| carrier-0004 | 1411 | 79.60 dB | 79.58 dB | 0.02 dB | 103.62 dB |
| carrier-0005 | 1411 | 80.45 dB | 80.43 dB | 0.02 dB | 104.48 dB |
| carrier-0006 | 1411 | 80.85 dB | 80.83 dB | 0.02 dB | 104.87 dB |
| carrier-0007 | 1411 | 81.19 dB | 81.17 dB | 0.02 dB | 105.22 dB |
| carrier-0008 | 1411 | 81.82 dB | 81.80 dB | 0.02 dB | 105.84 dB |
| carrier-0009 | 1411 | 80.09 dB | 80.07 dB | 0.02 dB | 104.11 dB |
| carrier-0010 | 1411 | 80.24 dB | 80.22 dB | 0.02 dB | 104.29 dB |
| carrier-0011 | 1411 | 64.16 dB | 64.16 dB | 0.00 dB | 107.08 dB |
| carrier-0012 | 1411 | 50.06 dB | 50.06 dB | 0.00 dB | 106.16 dB |
| carrier-0013 | 1411 | 45.99 dB | 45.99 dB | 0.00 dB | 104.41 dB |
| carrier-0014 | 1411 | 43.92 dB | 43.92 dB | 0.00 dB | 104.96 dB |
| carrier-0015 | 1536 | ∞ | 103.07 dB | ∞ | 103.07 dB |
| carrier-0016 | 1536 | ∞ | 105.26 dB | ∞ | 105.26 dB |
| carrier-0017 | 1536 | ∞ | 105.99 dB | ∞ | 105.99 dB |
| carrier-0018 | 3072 | 62.48 dB | 62.46 dB | 0.02 dB | 86.51 dB |
| carrier-0019 | 3072 | 59.93 dB | 59.91 dB | 0.02 dB | 83.96 dB |
| carrier-0020 | 3072 | 77.91 dB | 77.89 dB | 0.02 dB | 101.90 dB |
| carrier-0021 | 3072 | 77.64 dB | 77.62 dB | 0.02 dB | 101.63 dB |
| carrier-0022 | 3072 | 79.15 dB | 79.13 dB | 0.02 dB | 103.14 dB |
| carrier-0023 | 3072 | 76.46 dB | 76.44 dB | 0.02 dB | 100.46 dB |
| carrier-0024 | 3072 | 77.47 dB | 77.45 dB | 0.02 dB | 101.46 dB |
| carrier-0025 | 3072 | 77.83 dB | 77.81 dB | 0.02 dB | 101.83 dB |
| carrier-0026 | 3072 | 78.32 dB | 78.30 dB | 0.02 dB | 102.31 dB |
| carrier-0027 | 3072 | 78.30 dB | 78.28 dB | 0.02 dB | 102.28 dB |
| carrier-0028 | 3072 | 77.81 dB | 77.79 dB | 0.02 dB | 101.81 dB |
| carrier-0029 | 1411 | ∞ | 95.55 dB | ∞ | 95.55 dB |
| carrier-0030 | 1411 | ∞ | 95.26 dB | ∞ | 95.26 dB |
| carrier-0031 | 1411 | ∞ | 95.15 dB | ∞ | 95.15 dB |
| carrier-0032 | 1411 | ∞ | 95.06 dB | ∞ | 95.06 dB |
| carrier-0033 | 1411 | ∞ | 95.57 dB | ∞ | 95.57 dB |
| carrier-0034 | 1411 | ∞ | 94.88 dB | ∞ | 94.88 dB |
| carrier-0035 | 1411 | ∞ | 95.09 dB | ∞ | 95.09 dB |
| carrier-0036 | 1411 | ∞ | 95.49 dB | ∞ | 95.49 dB |
| carrier-0037 | 1411 | ∞ | 94.86 dB | ∞ | 94.86 dB |
| carrier-0038 | 1411 | ∞ | 95.09 dB | ∞ | 95.09 dB |
| carrier-0039 | 1411 | ∞ | 94.88 dB | ∞ | 94.88 dB |
| carrier-0040 | 1411 | ∞ | 94.88 dB | ∞ | 94.88 dB |
| carrier-0041 | 1411 | ∞ | 95.24 dB | ∞ | 95.24 dB |
| carrier-0042 | 1411 | ∞ | 95.29 dB | ∞ | 95.29 dB |
| carrier-0043 | 1411 | ∞ | 94.05 dB | ∞ | 94.05 dB |
| carrier-0044 | 1411 | ∞ | 94.93 dB | ∞ | 94.93 dB |
| carrier-0045 | 1411 | ∞ | 95.98 dB | ∞ | 95.98 dB |
| carrier-0046 | 1411 | ∞ | 95.35 dB | ∞ | 95.35 dB |
| carrier-0047 | 1411 | ∞ | 95.39 dB | ∞ | 95.39 dB |
| carrier-0048 | 1411 | ∞ | 95.52 dB | ∞ | 95.52 dB |
| carrier-0049 | 1411 | ∞ | 95.15 dB | ∞ | 95.15 dB |
| carrier-0050 | 1411 | ∞ | 95.24 dB | ∞ | 95.24 dB |
| carrier-0051 | 1411 | ∞ | 95.16 dB | ∞ | 95.16 dB |
| carrier-0052 | 1411 | ∞ | 95.06 dB | ∞ | 95.06 dB |
| carrier-0053 | 1411 | ∞ | 94.97 dB | ∞ | 94.97 dB |
| carrier-0054 | 1411 | ∞ | 94.95 dB | ∞ | 94.95 dB |
| carrier-0055 | 1536 | 46.77 dB | 46.77 dB | 0.00 dB | 109.50 dB |
| carrier-0056 | 1536 | 36.64 dB | 36.64 dB | 0.00 dB | 111.39 dB |
| carrier-0057 | 1536 | 45.62 dB | 45.62 dB | 0.00 dB | 108.91 dB |
| carrier-0058 | 1536 | 35.91 dB | 35.91 dB | 0.00 dB | 109.80 dB |
| carrier-0059 | 1536 | 44.31 dB | 44.31 dB | 0.00 dB | 109.69 dB |
| carrier-0060 | 1536 | 58.52 dB | 58.52 dB | 0.00 dB | 106.82 dB |
| carrier-0061 | 1536 | 41.19 dB | 41.19 dB | 0.00 dB | 109.82 dB |
| carrier-0062 | 1536 | 41.33 dB | 41.33 dB | 0.00 dB | 110.26 dB |
| carrier-0063 | 1536 | 43.97 dB | 43.97 dB | 0.00 dB | 110.75 dB |
| carrier-0064 | 1536 | 68.66 dB | 68.66 dB | 0.00 dB | 106.68 dB |
| carrier-0065 | 1536 | 52.48 dB | 52.48 dB | 0.00 dB | 108.85 dB |
| carrier-0066 | 1536 | 43.07 dB | 43.07 dB | 0.00 dB | 109.57 dB |
| carrier-0067 | 1536 | 46.30 dB | 46.30 dB | 0.00 dB | 110.44 dB |
| carrier-0068 | 1536 | 38.86 dB | 38.86 dB | 0.00 dB | 111.21 dB |
| carrier-0069 | 1536 | 54.46 dB | 54.46 dB | 0.00 dB | 109.45 dB |
| carrier-0070 | 1536 | 47.32 dB | 47.32 dB | 0.00 dB | 108.64 dB |
| carrier-0071 | 1536 | 43.17 dB | 43.17 dB | 0.00 dB | 108.75 dB |
| carrier-0072 | 1536 | 45.05 dB | 45.05 dB | 0.00 dB | 110.43 dB |
| carrier-0073 | 1536 | 49.41 dB | 49.41 dB | 0.00 dB | 107.69 dB |
| carrier-0074 | 1536 | 40.06 dB | 40.06 dB | 0.00 dB | 110.16 dB |
| carrier-0075 | 1536 | 44.09 dB | 44.09 dB | 0.00 dB | 109.58 dB |
| carrier-0076 | 1536 | 39.54 dB | 39.54 dB | 0.00 dB | 110.73 dB |
| carrier-0077 | 1536 | 48.44 dB | 48.44 dB | 0.00 dB | 110.11 dB |
| carrier-0078 | 1536 | 40.84 dB | 40.84 dB | 0.00 dB | 111.04 dB |
| carrier-0079 | 1536 | 45.88 dB | 45.88 dB | 0.00 dB | 109.59 dB |
| carrier-0080 | 1536 | 46.94 dB | 46.94 dB | 0.00 dB | 109.72 dB |
| carrier-0081 | 1536 | 48.46 dB | 48.46 dB | 0.00 dB | 109.30 dB |
| carrier-0082 | 1536 | 43.09 dB | 43.09 dB | 0.00 dB | 108.61 dB |
| carrier-0083 | 1536 | 59.75 dB | 59.75 dB | 0.00 dB | 106.91 dB |
| carrier-0084 | 1536 | 41.01 dB | 41.01 dB | 0.00 dB | 110.36 dB |

## ogg/vorbis

Measured on 84 of 84 carriers.

### Does it match the canonical ffmpeg encode?

Canonical means `ffmpeg -q:a N` at the level Mist chose for Vorbis, and ffmpeg's own defaults for a lossless codec. The verdict fails when Mist-clean or Mist-stego differs from that file on an identity field (length, trailing zeros, sample format, nominal rate, tags, FLAC STREAMINFO, Ogg granule and serial count). File size and packet size are left to the metadata warden. Ogg serial values are random and are not compared. Default ffmpeg is the next sentence, and it is a different threat model.

| Carrier | Source samples | Samples (canonical / Mist) | Zero tail (canonical / Mist) | Sample format (canonical / Mist) | Nominal kbps (canonical / Mist) | Canonical differs | Default ffmpeg differs |
|---|---|---|---|---|---|---|---|
| carrier-0001 | 13230191 | 13230191 / 13230191 | 0 / 0 | fltp / fltp | 256 / 256 | metadata tags | nominal bitrate, metadata tags |
| carrier-0002 | 13230191 | 13230191 / 13230191 | 0 / 0 | fltp / fltp | 256 / 256 | metadata tags | nominal bitrate, metadata tags |
| carrier-0003 | 13230191 | 13230191 / 13230191 | 0 / 0 | fltp / fltp | 256 / 256 | metadata tags | nominal bitrate, metadata tags |
| carrier-0004 | 11669231 | 11669231 / 11669231 | 0 / 0 | fltp / fltp | 256 / 256 | metadata tags | nominal bitrate, metadata tags |
| carrier-0005 | 13230191 | 13230191 / 13230191 | 0 / 0 | fltp / fltp | 256 / 256 | metadata tags | nominal bitrate, metadata tags |
| carrier-0006 | 13230191 | 13230191 / 13230191 | 0 / 0 | fltp / fltp | 256 / 256 | metadata tags | nominal bitrate, metadata tags |
| carrier-0007 | 13230191 | 13230191 / 13230191 | 0 / 0 | fltp / fltp | 256 / 256 | metadata tags | nominal bitrate, metadata tags |
| carrier-0008 | 13230191 | 13230191 / 13230191 | 0 / 0 | fltp / fltp | 256 / 256 | metadata tags | nominal bitrate, metadata tags |
| carrier-0009 | 11631215 | 11631215 / 11631215 | 0 / 0 | fltp / fltp | 256 / 256 | metadata tags | nominal bitrate, metadata tags |
| carrier-0010 | 12556271 | 12556271 / 12556271 | 30255 / 30255 | fltp / fltp | 256 / 256 | metadata tags | zero tail, nominal bitrate, metadata tags |
| carrier-0011 | 13230767 | 13230767 / 13230767 | 0 / 0 | fltp / fltp | 256 / 256 | metadata tags | nominal bitrate, metadata tags |
| carrier-0012 | 13230767 | 13230767 / 13230767 | 0 / 0 | fltp / fltp | 256 / 256 | metadata tags | nominal bitrate, metadata tags |
| carrier-0013 | 13230767 | 13230767 / 13230767 | 0 / 0 | fltp / fltp | 256 / 256 | metadata tags | nominal bitrate, metadata tags |
| carrier-0014 | 13230767 | 13230767 / 13230767 | 0 / 0 | fltp / fltp | 256 / 256 | metadata tags | nominal bitrate, metadata tags |
| carrier-0015 | 2880000 | 2880000 / 2880000 | 0 / 0 | fltp / fltp | 256 / 256 | — | nominal bitrate |
| carrier-0016 | 5760000 | 5760000 / 5760000 | 0 / 0 | fltp / fltp | 256 / 256 | — | nominal bitrate |
| carrier-0017 | 5760000 | 5760000 / 5760000 | 0 / 0 | fltp / fltp | 256 / 256 | — | nominal bitrate |
| carrier-0018 | 28803072 | 28803072 / 28803072 | 0 / 0 | fltp / fltp | 4294967 / 4294967 | metadata tags | metadata tags |
| carrier-0019 | 28803072 | 28803072 / 28803072 | 0 / 0 | fltp / fltp | 4294967 / 4294967 | metadata tags | metadata tags |
| carrier-0020 | 27650560 | 27650560 / 27650560 | 0 / 0 | fltp / fltp | 4294967 / 4294967 | metadata tags | metadata tags |
| carrier-0021 | 27998720 | 27998720 / 27998720 | 0 / 0 | fltp / fltp | 4294967 / 4294967 | metadata tags | metadata tags |
| carrier-0022 | 24209920 | 24209920 / 24209920 | 0 / 0 | fltp / fltp | 4294967 / 4294967 | metadata tags | metadata tags |
| carrier-0023 | 28803072 | 28803072 / 28803072 | 0 / 0 | fltp / fltp | 4294967 / 4294967 | metadata tags | metadata tags |
| carrier-0024 | 24574720 | 24574720 / 24574720 | 0 / 0 | fltp / fltp | 4294967 / 4294967 | metadata tags | metadata tags |
| carrier-0025 | 26830080 | 26830080 / 26830080 | 0 / 0 | fltp / fltp | 4294967 / 4294967 | metadata tags | metadata tags |
| carrier-0026 | 24695040 | 24695040 / 24695040 | 0 / 0 | fltp / fltp | 4294967 / 4294967 | metadata tags | metadata tags |
| carrier-0027 | 18909440 | 18909440 / 18909440 | 0 / 0 | fltp / fltp | 4294967 / 4294967 | metadata tags | metadata tags |
| carrier-0028 | 28803072 | 28803072 / 28803072 | 0 / 0 | fltp / fltp | 4294967 / 4294967 | metadata tags | metadata tags |
| carrier-0029 | 7939176 | 7939176 / 7939176 | 0 / 0 | fltp / fltp | 256 / 256 | — | nominal bitrate |
| carrier-0030 | 8334312 | 8334312 / 8334312 | 0 / 0 | fltp / fltp | 256 / 256 | — | nominal bitrate |
| carrier-0031 | 8187312 | 8187312 / 8187312 | 0 / 0 | fltp / fltp | 256 / 256 | — | nominal bitrate |
| carrier-0032 | 7616952 | 7616952 / 7616952 | 0 / 0 | fltp / fltp | 256 / 256 | — | nominal bitrate |
| carrier-0033 | 7703976 | 7703976 / 7703976 | 0 / 0 | fltp / fltp | 256 / 256 | — | nominal bitrate |
| carrier-0034 | 7795116 | 7795116 / 7795116 | 0 / 0 | fltp / fltp | 256 / 256 | — | nominal bitrate |
| carrier-0035 | 8786484 | 8786484 / 8786484 | 0 / 0 | fltp / fltp | 256 / 256 | — | nominal bitrate |
| carrier-0036 | 8306676 | 8306676 / 8306676 | 0 / 0 | fltp / fltp | 256 / 256 | — | nominal bitrate |
| carrier-0037 | 7848624 | 7848624 / 7848624 | 0 / 0 | fltp / fltp | 256 / 256 | — | nominal bitrate |
| carrier-0038 | 7849212 | 7849212 / 7849212 | 0 / 0 | fltp / fltp | 256 / 256 | — | nominal bitrate |
| carrier-0039 | 8371356 | 8371356 / 8371356 | 0 / 0 | fltp / fltp | 256 / 256 | — | nominal bitrate |
| carrier-0040 | 7808052 | 7808052 / 7808052 | 0 / 0 | fltp / fltp | 256 / 256 | — | nominal bitrate |
| carrier-0041 | 7792764 | 7792764 / 7792764 | 0 / 0 | fltp / fltp | 256 / 256 | — | nominal bitrate |
| carrier-0042 | 7705152 | 7705152 / 7705152 | 0 / 0 | fltp / fltp | 256 / 256 | — | nominal bitrate |
| carrier-0043 | 7733376 | 7733376 / 7733376 | 0 / 0 | fltp / fltp | 256 / 256 | — | nominal bitrate |
| carrier-0044 | 8332548 | 8332548 / 8332548 | 0 / 0 | fltp / fltp | 256 / 256 | — | nominal bitrate |
| carrier-0045 | 7836864 | 7836864 / 7836864 | 0 / 0 | fltp / fltp | 256 / 256 | — | nominal bitrate |
| carrier-0046 | 7783356 | 7783356 / 7783356 | 0 / 0 | fltp / fltp | 256 / 256 | — | nominal bitrate |
| carrier-0047 | 8134980 | 8134980 / 8134980 | 0 / 0 | fltp / fltp | 256 / 256 | — | nominal bitrate |
| carrier-0048 | 7904484 | 7904484 / 7904484 | 0 / 0 | fltp / fltp | 256 / 256 | — | nominal bitrate |
| carrier-0049 | 7577556 | 7577556 / 7577556 | 0 / 0 | fltp / fltp | 256 / 256 | — | nominal bitrate |
| carrier-0050 | 8433684 | 8433684 / 8433684 | 0 / 0 | fltp / fltp | 256 / 256 | — | nominal bitrate |
| carrier-0051 | 8167320 | 8167320 / 8167320 | 0 / 0 | fltp / fltp | 256 / 256 | — | nominal bitrate |
| carrier-0052 | 7443492 | 7443492 / 7443492 | 0 / 0 | fltp / fltp | 256 / 256 | — | nominal bitrate |
| carrier-0053 | 7847448 | 7847448 / 7847448 | 0 / 0 | fltp / fltp | 256 / 256 | — | nominal bitrate |
| carrier-0054 | 6560316 | 6560316 / 6560316 | 0 / 0 | fltp / fltp | 256 / 256 | — | nominal bitrate |
| carrier-0055 | 7567536 | 7567536 / 7567536 | 368 / 368 | fltp / fltp | 256 / 256 | metadata tags | zero tail, nominal bitrate, metadata tags |
| carrier-0056 | 10716829 | 10716829 / 10716829 | 1117 / 1117 | fltp / fltp | 256 / 256 | metadata tags | zero tail, nominal bitrate, metadata tags |
| carrier-0057 | 10753254 | 10753254 / 10753254 | 0 / 0 | fltp / fltp | 256 / 256 | metadata tags | nominal bitrate, metadata tags |
| carrier-0058 | 8507078 | 8507078 / 8507078 | 0 / 0 | fltp / fltp | 256 / 256 | metadata tags | nominal bitrate, metadata tags |
| carrier-0059 | 9099131 | 9099131 / 9099131 | 0 / 0 | fltp / fltp | 256 / 256 | metadata tags | nominal bitrate, metadata tags |
| carrier-0060 | 10184328 | 10184328 / 10184328 | 0 / 0 | fltp / fltp | 256 / 256 | metadata tags | nominal bitrate, metadata tags |
| carrier-0061 | 9848955 | 9848955 / 9848955 | 0 / 0 | fltp / fltp | 256 / 256 | metadata tags | nominal bitrate, metadata tags |
| carrier-0062 | 10444219 | 10444219 / 10444219 | 635 / 635 | fltp / fltp | 256 / 256 | metadata tags | zero tail, nominal bitrate, metadata tags |
| carrier-0063 | 10854001 | 10854001 / 10854001 | 0 / 0 | fltp / fltp | 256 / 256 | metadata tags | nominal bitrate, metadata tags |
| carrier-0064 | 9693408 | 9693408 / 9693408 | 0 / 0 | fltp / fltp | 256 / 256 | metadata tags | nominal bitrate, metadata tags |
| carrier-0065 | 9766958 | 9766958 / 9766958 | 0 / 0 | fltp / fltp | 256 / 256 | metadata tags | nominal bitrate, metadata tags |
| carrier-0066 | 9359938 | 9359938 / 9359938 | 0 / 0 | fltp / fltp | 256 / 256 | metadata tags | nominal bitrate, metadata tags |
| carrier-0067 | 9855000 | 9855000 / 9855000 | 7384 / 7384 | fltp / fltp | 256 / 256 | metadata tags | zero tail, nominal bitrate, metadata tags |
| carrier-0068 | 9146026 | 9146026 / 9146026 | 0 / 0 | fltp / fltp | 256 / 256 | metadata tags | nominal bitrate, metadata tags |
| carrier-0069 | 10717286 | 10717286 / 10717286 | 0 / 0 | fltp / fltp | 256 / 256 | metadata tags | nominal bitrate, metadata tags |
| carrier-0070 | 8370000 | 8370000 / 8370000 | 0 / 0 | fltp / fltp | 256 / 256 | metadata tags | nominal bitrate, metadata tags |
| carrier-0071 | 10456119 | 10456119 / 10456119 | 22903 / 22903 | fltp / fltp | 256 / 256 | metadata tags | zero tail, nominal bitrate, metadata tags |
| carrier-0072 | 9573914 | 9573914 / 9573914 | 6874 / 6874 | fltp / fltp | 256 / 256 | metadata tags | zero tail, nominal bitrate, metadata tags |
| carrier-0073 | 10171848 | 10171848 / 10171848 | 0 / 0 | fltp / fltp | 256 / 256 | metadata tags | zero tail, nominal bitrate, metadata tags |
| carrier-0074 | 9735652 | 9735652 / 9735652 | 0 / 0 | fltp / fltp | 256 / 256 | metadata tags | nominal bitrate, metadata tags |
| carrier-0075 | 9412488 | 9412488 / 9412488 | 6088 / 6088 | fltp / fltp | 256 / 256 | metadata tags | nominal bitrate, metadata tags |
| carrier-0076 | 9888000 | 9888000 / 9888000 | 0 / 0 | fltp / fltp | 256 / 256 | metadata tags | nominal bitrate, metadata tags |
| carrier-0077 | 10440000 | 10440000 / 10440000 | 0 / 0 | fltp / fltp | 256 / 256 | metadata tags | nominal bitrate, metadata tags |
| carrier-0078 | 11520002 | 11520002 / 11520002 | 0 / 0 | fltp / fltp | 256 / 256 | metadata tags | nominal bitrate, metadata tags |
| carrier-0079 | 10450747 | 10450747 / 10450747 | 20731 / 20731 | fltp / fltp | 256 / 256 | metadata tags | nominal bitrate, metadata tags |
| carrier-0080 | 8598259 | 8598259 / 8598259 | 0 / 0 | fltp / fltp | 256 / 256 | metadata tags | nominal bitrate, metadata tags |
| carrier-0081 | 11520000 | 11520000 / 11520000 | 0 / 0 | fltp / fltp | 256 / 256 | metadata tags | nominal bitrate, metadata tags |
| carrier-0082 | 11102609 | 11102609 / 11102609 | 0 / 0 | fltp / fltp | 256 / 256 | metadata tags | nominal bitrate, metadata tags |
| carrier-0083 | 8540309 | 8540309 / 8540309 | 5589 / 5589 | fltp / fltp | 256 / 256 | metadata tags | zero tail, nominal bitrate, metadata tags |
| carrier-0084 | 10279847 | 10279847 / 10279847 | 6503 / 6503 | fltp / fltp | 256 / 256 | metadata tags | zero tail, nominal bitrate, metadata tags |

**Verdict:** ❌ differs from the canonical ffmpeg encode on 55 of 84 carriers: metadata tags.

differs from ffmpeg defaults on 84 of 84 carriers: nominal bitrate, metadata tags, zero tail. That is a separate threat model, not the canonical verdict.

### Audio-blind metadata

This classifier sees ffprobe fields, container summaries and file size. It does not see samples. Two questions are scored apart: canonical ffmpeg against Mist-clean (a pipeline fingerprint) and Mist-clean against Mist-stego (an embedding side effect). The preregistered pass is a file interval that includes 0.5. An interval that excludes 0.5 is a failed objective.

| Question | File AUC (95%) | D | Interval includes 0.5 |
|---|---|---|---|
| canonical-vs-mist-clean | 0.917 (0.867–0.954) | 0.917 | no |
| mist-clean-vs-stego | 0.500 (0.500–0.500) | 0.500 | yes |

### Can a detector tell?

Stego copy against the canonical ffmpeg copy, so the quality setting is not the signal.

| Detector | Looks for | Chunk AUC (95%) | File AUC (95%) | D | Detector-implied benchmark KL lower bound (nats, 95%) | Adjusted p | Verdict | Message size | AUC change |
|---|---|---|---|---|---|---|---|---|---|
| chi-square | bit overwriting (not what Mist does) | 0.500 (0.500–0.500) | 0.500 (0.500–0.500) | 0.500 | 0.000 (0.000–0.000) | BH 1.000 | ✅ chance | ✅ hidden (0.500) | +0.000 |
| spa | bit overwriting (not what Mist does) | 0.500 (0.500–0.500) | 0.500 (0.500–0.501) | 0.500 | 0.000 (0.000–0.000) | BH 0.630 | ✅ chance | ✅ hidden (0.500) | +0.000 |
| rs | bit overwriting (not what Mist does) | 0.500 (0.499–0.500) | 0.490 (0.475–0.500) | 0.510 | 0.000 (0.000–0.001) | BH 0.630 | ✅ chance | ✅ hidden (0.500) | -0.000 |
| hcf-com | ±1 changes, which is what Mist does | 0.500 (0.500–0.500) | 0.499 (0.496–0.501) | 0.501 | 0.000 (0.000–0.000) | Holm 0.870 | ⚠️ faint signal | ✅ hidden (0.500) | -0.001 |
| classifier | anything it can learn from Mist's own output | 0.501 (0.500–0.501) | 0.504 (0.502–0.506) | 0.504 | 0.000 (0.000–0.000) | Holm 0.600 | ⚠️ faint signal | ✅ hidden (0.500) | +0.002 |
| markov | how the waveform's curvature changes from sample to sample | 0.500 (0.500–0.501) | 0.500 (0.498–0.503) | 0.500 | 0.000 (0.000–0.000) | Holm 0.600 | ⚠️ faint signal | ✅ hidden (0.500) | +0.003 |
| rich | a frozen summary of prediction error, co-occurrence and a short spectrum | 0.500 (0.500–0.500) | 0.501 (0.499–0.502) | 0.501 | 0.000 (0.000–0.000) | Holm 1.000 | ✅ chance | ✅ hidden (0.500) | +0.001 |
| key-aware | the ephemeral key in the first frame's envelope, read with the public key alone | 0.506 (0.476–0.536) | 0.506 (0.476–0.536) | 0.506 | 0.000 (0.000–0.003) | Holm 1.000 | ✅ chance | ✅ hidden (0.500) | +0.012 |
| selection | LSB bias at positions the recipient public key implies | 0.506 (0.497–0.515) | 0.506 (0.497–0.515) | 0.506 | 0.000 (0.000–0.000) | Holm 0.780 | ✅ chance | ✅ hidden (0.500) | +0.009 |

Power: shifting this run's classifier file scores to D = 0.55, about 8 independent lineages give 90% power for a lineage-cluster interval to exclude 0.5. The figure is a simulation from this run's dispersion.

Worst cell: rs on aggregate, file AUC 0.490, D 0.510.

**Verdict:** ⚠️ faint signal from hcf-com (AUC 0.500).

### The embedding alone

Stego copy against Mist's own re-encode, so only the embedded changes differ.

| Detector | Looks for | Chunk AUC (95%) | File AUC (95%) | D | Detector-implied benchmark KL lower bound (nats, 95%) | Adjusted p | Verdict | Message size |
|---|---|---|---|---|---|---|---|---|
| chi-square | bit overwriting (not what Mist does) | 0.500 (0.500–0.500) | 0.500 (0.500–0.500) | 0.500 | 0.000 (0.000–0.000) | — | ✅ chance | ✅ hidden (0.500) |
| spa | bit overwriting (not what Mist does) | 0.500 (0.500–0.500) | 0.500 (0.500–0.501) | 0.500 | 0.000 (0.000–0.000) | — | ✅ chance | ✅ hidden (0.500) |
| rs | bit overwriting (not what Mist does) | 0.500 (0.499–0.500) | 0.490 (0.475–0.500) | 0.510 | 0.000 (0.000–0.001) | — | ✅ chance | ✅ hidden (0.500) |
| hcf-com | ±1 changes, which is what Mist does | 0.500 (0.500–0.500) | 0.499 (0.496–0.501) | 0.501 | 0.000 (0.000–0.000) | — | ⚠️ faint signal | ✅ hidden (0.500) |
| classifier | anything it can learn from Mist's own output | 0.501 (0.500–0.501) | 0.504 (0.502–0.506) | 0.504 | 0.000 (0.000–0.000) | — | ⚠️ faint signal | ✅ hidden (0.500) |
| markov | how the waveform's curvature changes from sample to sample | 0.500 (0.500–0.501) | 0.500 (0.498–0.503) | 0.500 | 0.000 (0.000–0.000) | — | ⚠️ faint signal | ✅ hidden (0.500) |
| rich | a frozen summary of prediction error, co-occurrence and a short spectrum | 0.500 (0.500–0.500) | 0.501 (0.499–0.502) | 0.501 | 0.000 (0.000–0.000) | — | ✅ chance | ✅ hidden (0.500) |

### Does detection grow with audio?

File AUC (95%) against the clean ffmpeg copy on the first chunks of each carrier. A chunk is 65,536 values, about 0.74 s of 44.1 kHz stereo.

| Detector | First 40 chunks | Whole file |
|---|---|---|
| chi-square | 0.500 (0.500–0.500) | 0.500 (0.500–0.500) |
| spa | 0.500 (0.499–0.501) | 0.500 (0.500–0.501) |
| rs | 0.495 (0.484–0.500) | 0.490 (0.475–0.500) |
| hcf-com | 0.498 (0.496–0.500) | 0.499 (0.496–0.501) |
| classifier | 0.506 (0.503–0.510) | 0.504 (0.502–0.506) |
| markov | 0.505 (0.502–0.510) | 0.500 (0.498–0.503) |
| rich | 0.501 (0.500–0.502) | 0.501 (0.499–0.502) |

### Does detection grow with the number of files?

AUC when the warden averages a detector's score over k files drawn at random from the corpus, stego against clean. Draws overlap, so there is no interval.

| Detector | 1 file(s) | 3 file(s) | 6 file(s) |
|---|---|---|---|
| chi-square | 0.500 | 0.500 | 0.500 |
| spa | 0.482 | 0.509 | 0.519 |
| rs | 0.464 | 0.485 | 0.504 |
| hcf-com | 0.523 | 0.486 | 0.502 |
| classifier | 0.533 | 0.511 | 0.495 |
| markov | 0.527 | 0.512 | 0.490 |
| rich | 0.529 | 0.511 | 0.494 |

### How much does it cost in sound?

| Measure | Mean | Worst | What it means |
|---|---|---|---|
| Plain re-encode SDR | 27.65 dB | 23.99 dB | Loss of Mist's own re-encode, with nothing embedded |
| Mist output SDR | 27.63 dB | 23.97 dB | The same, with the message embedded |
| Embedding cost | 0.02 dB | 0.03 dB | Mist's own share; target ≤ 0.3 dB, design-doc reference 0.18 dB |
| Extra error energy | +0% | +1% | Embedding cost as error added on top of a plain re-encode |
| Mist's error below the music | 52.43 dB | 45.77 dB | Stego copy against Mist's own re-encode: what embedding alone adds |
| Embedding cost, earlier run | 0.01 dB | 0.02 dB | Same measure at commit being compared with |

**Verdict:** ✅ Mist's own error sits 52.43 dB below the music; embedding costs 0.02 dB, within the 0.3 dB target.

### Per carrier

| Carrier | Output kbps | Plain re-encode SDR | Mist output SDR | Embedding cost | Mist's error below the music |
|---|---|---|---|---|---|
| carrier-0001 | 229 | 28.18 dB | 28.18 dB | 0.01 dB | 56.88 dB |
| carrier-0002 | 228 | 28.56 dB | 28.55 dB | 0.01 dB | 56.80 dB |
| carrier-0003 | 221 | 29.59 dB | 29.59 dB | 0.00 dB | 59.76 dB |
| carrier-0004 | 225 | 28.35 dB | 28.35 dB | 0.00 dB | 60.24 dB |
| carrier-0005 | 223 | 28.81 dB | 28.81 dB | 0.00 dB | 59.68 dB |
| carrier-0006 | 226 | 29.55 dB | 29.55 dB | 0.01 dB | 58.35 dB |
| carrier-0007 | 227 | 29.02 dB | 29.02 dB | 0.01 dB | 57.94 dB |
| carrier-0008 | 221 | 29.32 dB | 29.31 dB | 0.00 dB | 59.01 dB |
| carrier-0009 | 189 | 27.89 dB | 27.88 dB | 0.01 dB | 52.34 dB |
| carrier-0010 | 192 | 29.22 dB | 29.21 dB | 0.01 dB | 54.68 dB |
| carrier-0011 | 268 | 32.74 dB | 32.73 dB | 0.01 dB | 57.34 dB |
| carrier-0012 | 269 | 25.99 dB | 25.97 dB | 0.02 dB | 48.84 dB |
| carrier-0013 | 258 | 32.92 dB | 32.90 dB | 0.01 dB | 57.93 dB |
| carrier-0014 | 243 | 30.99 dB | 30.99 dB | 0.01 dB | 59.22 dB |
| carrier-0015 | 262 | 26.02 dB | 26.00 dB | 0.02 dB | 50.17 dB |
| carrier-0016 | 282 | 26.09 dB | 26.08 dB | 0.01 dB | 51.70 dB |
| carrier-0017 | 261 | 27.80 dB | 27.79 dB | 0.01 dB | 52.50 dB |
| carrier-0018 | 332 | 30.98 dB | 30.98 dB | 0.00 dB | 70.03 dB |
| carrier-0019 | 295 | 30.48 dB | 30.48 dB | 0.00 dB | 70.70 dB |
| carrier-0020 | 244 | 29.12 dB | 29.11 dB | 0.01 dB | 55.08 dB |
| carrier-0021 | 218 | 28.35 dB | 28.34 dB | 0.01 dB | 53.11 dB |
| carrier-0022 | 257 | 29.87 dB | 29.85 dB | 0.02 dB | 53.92 dB |
| carrier-0023 | 220 | 29.76 dB | 29.75 dB | 0.01 dB | 55.29 dB |
| carrier-0024 | 271 | 28.28 dB | 28.27 dB | 0.01 dB | 53.17 dB |
| carrier-0025 | 226 | 27.33 dB | 27.32 dB | 0.02 dB | 51.16 dB |
| carrier-0026 | 273 | 27.07 dB | 27.05 dB | 0.02 dB | 50.96 dB |
| carrier-0027 | 224 | 26.76 dB | 26.75 dB | 0.01 dB | 52.17 dB |
| carrier-0028 | 237 | 28.52 dB | 28.51 dB | 0.01 dB | 53.16 dB |
| carrier-0029 | 233 | 28.56 dB | 28.54 dB | 0.02 dB | 52.02 dB |
| carrier-0030 | 237 | 29.33 dB | 29.30 dB | 0.02 dB | 53.06 dB |
| carrier-0031 | 239 | 29.55 dB | 29.53 dB | 0.02 dB | 52.58 dB |
| carrier-0032 | 244 | 29.52 dB | 29.50 dB | 0.01 dB | 54.90 dB |
| carrier-0033 | 230 | 29.20 dB | 29.19 dB | 0.02 dB | 53.81 dB |
| carrier-0034 | 245 | 29.59 dB | 29.58 dB | 0.01 dB | 54.10 dB |
| carrier-0035 | 239 | 29.11 dB | 29.10 dB | 0.01 dB | 54.66 dB |
| carrier-0036 | 231 | 28.73 dB | 28.71 dB | 0.02 dB | 52.25 dB |
| carrier-0037 | 245 | 28.61 dB | 28.59 dB | 0.02 dB | 51.96 dB |
| carrier-0038 | 249 | 29.27 dB | 29.26 dB | 0.02 dB | 54.12 dB |
| carrier-0039 | 248 | 29.16 dB | 29.14 dB | 0.02 dB | 53.16 dB |
| carrier-0040 | 240 | 28.96 dB | 28.94 dB | 0.02 dB | 52.87 dB |
| carrier-0041 | 232 | 28.92 dB | 28.90 dB | 0.02 dB | 52.67 dB |
| carrier-0042 | 242 | 28.21 dB | 28.19 dB | 0.02 dB | 53.02 dB |
| carrier-0043 | 236 | 29.10 dB | 29.08 dB | 0.02 dB | 52.52 dB |
| carrier-0044 | 242 | 29.12 dB | 29.11 dB | 0.02 dB | 53.51 dB |
| carrier-0045 | 244 | 28.05 dB | 28.03 dB | 0.01 dB | 53.05 dB |
| carrier-0046 | 242 | 29.18 dB | 29.16 dB | 0.02 dB | 52.46 dB |
| carrier-0047 | 248 | 29.15 dB | 29.13 dB | 0.02 dB | 53.58 dB |
| carrier-0048 | 233 | 29.35 dB | 29.33 dB | 0.02 dB | 53.25 dB |
| carrier-0049 | 234 | 29.24 dB | 29.22 dB | 0.02 dB | 54.00 dB |
| carrier-0050 | 240 | 27.36 dB | 27.34 dB | 0.02 dB | 51.02 dB |
| carrier-0051 | 242 | 28.76 dB | 28.74 dB | 0.01 dB | 53.87 dB |
| carrier-0052 | 231 | 28.44 dB | 28.42 dB | 0.02 dB | 52.33 dB |
| carrier-0053 | 231 | 28.97 dB | 28.96 dB | 0.02 dB | 53.88 dB |
| carrier-0054 | 233 | 27.31 dB | 27.30 dB | 0.01 dB | 52.59 dB |
| carrier-0055 | 263 | 26.64 dB | 26.62 dB | 0.02 dB | 49.96 dB |
| carrier-0056 | 249 | 25.93 dB | 25.91 dB | 0.02 dB | 48.40 dB |
| carrier-0057 | 264 | 25.30 dB | 25.28 dB | 0.02 dB | 48.56 dB |
| carrier-0058 | 255 | 25.05 dB | 25.03 dB | 0.02 dB | 48.14 dB |
| carrier-0059 | 250 | 24.35 dB | 24.32 dB | 0.03 dB | 46.01 dB |
| carrier-0060 | 243 | 23.99 dB | 23.97 dB | 0.02 dB | 46.10 dB |
| carrier-0061 | 248 | 25.28 dB | 25.26 dB | 0.02 dB | 47.89 dB |
| carrier-0062 | 245 | 26.22 dB | 26.20 dB | 0.02 dB | 49.61 dB |
| carrier-0063 | 259 | 26.16 dB | 26.15 dB | 0.02 dB | 49.79 dB |
| carrier-0064 | 232 | 26.07 dB | 26.06 dB | 0.02 dB | 49.81 dB |
| carrier-0065 | 244 | 25.82 dB | 25.80 dB | 0.02 dB | 48.92 dB |
| carrier-0066 | 251 | 25.61 dB | 25.60 dB | 0.02 dB | 49.52 dB |
| carrier-0067 | 256 | 24.98 dB | 24.96 dB | 0.02 dB | 47.17 dB |
| carrier-0068 | 266 | 25.83 dB | 25.81 dB | 0.02 dB | 49.32 dB |
| carrier-0069 | 233 | 25.43 dB | 25.41 dB | 0.02 dB | 48.21 dB |
| carrier-0070 | 244 | 27.25 dB | 27.23 dB | 0.02 dB | 50.17 dB |
| carrier-0071 | 246 | 24.64 dB | 24.61 dB | 0.03 dB | 45.77 dB |
| carrier-0072 | 248 | 25.06 dB | 25.04 dB | 0.03 dB | 46.69 dB |
| carrier-0073 | 257 | 25.68 dB | 25.67 dB | 0.02 dB | 49.72 dB |
| carrier-0074 | 244 | 25.43 dB | 25.40 dB | 0.03 dB | 47.36 dB |
| carrier-0075 | 259 | 26.01 dB | 26.00 dB | 0.02 dB | 50.10 dB |
| carrier-0076 | 271 | 24.70 dB | 24.69 dB | 0.02 dB | 48.60 dB |
| carrier-0077 | 255 | 25.17 dB | 25.15 dB | 0.02 dB | 47.95 dB |
| carrier-0078 | 256 | 26.06 dB | 26.04 dB | 0.02 dB | 49.37 dB |
| carrier-0079 | 245 | 24.20 dB | 24.18 dB | 0.02 dB | 47.06 dB |
| carrier-0080 | 260 | 25.76 dB | 25.74 dB | 0.02 dB | 48.10 dB |
| carrier-0081 | 244 | 24.64 dB | 24.62 dB | 0.02 dB | 47.06 dB |
| carrier-0082 | 245 | 25.19 dB | 25.17 dB | 0.02 dB | 47.40 dB |
| carrier-0083 | 255 | 24.11 dB | 24.09 dB | 0.02 dB | 47.72 dB |
| carrier-0084 | 261 | 27.70 dB | 27.68 dB | 0.02 dB | 50.47 dB |

## How to read this report

### Plain properties

Before any statistics, a file shows how long it is, whether it ends in digital zeros, what sample format it
decodes to and what bitrate its header claims. The **ffmpeg** column is the same carrier encoded by the ffmpeg
command line at its own defaults, keeping only the first audio stream. Any difference from it tells Mist's output
apart from an ordinary file outright, however well the embedding hides, so every carrier must match.

### Detectability

**AUC** is the chance that a detector, shown one clean and one stego sample, picks the stego one. The main table
compares against the ffmpeg copy, the cover a warden would actually have. **The embedding alone** compares against
Mist's own re-encode, so it isolates the embedded changes from every other way Mist's pipeline differs from ffmpeg. 0.5 is a coin
flip, which is the goal; 1.0 means it is caught every time. A value well below 0.5 is a detection too: the
detector is right, just with its sign flipped. Ogg Vorbis is scored on the residues Mist may change, lossless
formats on the decoded samples.

Each carrier is cut into chunks of 65,536 values and every chunk is scored: that is the **chunk AUC**. The
**file AUC** averages each recording's chunk scores into one score per file. Chapters of one session stay
separate files and share a lineage. The **95% file interval** redraws lineages first and then the recordings
inside a drawn lineage. The chunk interval redraws whole lineages too, never single chunks. A recording-cluster
interval is printed beside a detector only when its width differs from the lineage interval by more than 0.01.
With one recording per lineage the two intervals match. Their width follows the number of lineages, not the
number of chunks. The verdict is the worse of the chunk and file verdicts.

**D** is 0.5 + |AUC − 0.5|. A detector that is perfectly wrong, AUC 0, has the same D as a detector that is
perfectly right. The file interval and the verdict already treat an interval that sits entirely below 0.5 as
detection; D puts that on one scale.

The **detector-implied benchmark KL lower bound** applies Pinsker's inequality to the file AUC:
|AUC − ½| is at most the total variation between the benchmark clean and stego populations, so
KL ≥ 2(AUC − ½)² nats. This is only weak attack evidence for this detector and benchmark. It is not an estimate
or upper bound for Cachin's ε; a detector at chance shows that this detector found no gap, not that KL is small.

Chi-square, SPA and RS look for bits being overwritten, which Mist never does, so they are expected to sit at
chance; a rise means the embedder has drifted. They are exploratory: the report gives each a paired lineage
label-swap p-value and a Benjamini-Hochberg adjustment across the three. HCF-COM looks for ±1 changes, which
is exactly what Mist does, so it sits in the confirmatory family with the classifier, the Markov model, the
key-aware warden, the rich model and the selection-channel warden. That family of six is adjusted with Holm.
A dash means that row was not part of the confirmatory test (scaling, category and embedding-only tables).

The **classifier** is the adversary of record. It is a logistic regression on this run's own clean and stego
chunks, using every detector's score, the share of values at each of -3…3, and how each step between adjacent
values follows the one before it. Outer folds are lineages. An inner grouped search picks the L2 penalty from
0.001, 0.01 and 0.1, and a further held-out lineage calibrates the score. Fewer than four lineages falls back
to the fixed-penalty cross-validation. Standardisation is fit on the training rows of that fold. The primary
operational comparison repeats that whole fit under nine lineage label swaps; scaling and category rows do not.
Nine refits make the smallest attainable p-value 0.1, so that row cannot by itself clear 0.05.
Its message-size check trains a second model to tell a 64-byte message from a 1-byte one.

**Markov** is the same kind of model, trained the same way, on one richer feature set only: how the second
difference between samples, the waveform's curvature, changes from one sample to the next, with each value
clipped to -3…3. That curvature is near zero wherever the audio is smooth, so ±1 changes stand out in it more
than in the values or their steps.

**Rich** is the same kind of model on a frozen summary: prediction errors through order 8, differences
through order 4, an LPC residual, decimated scales, parity, symmetrized co-occurrence, and a short
log-spaced spectrum. The harness flattens Vorbis residues to one stream, so this vector does not
condition them on packet, codebook or Huffman context. Fisher, stump and random-subspace fits of the
same vector are package baselines; this row is the logistic one. A CNN trained on exported chunks is a
separate learned warden and is not run here. When it is run, its folds follow lineage as well.

**Selection** is the LSB gap at positions implied by the recipient public key, on the first chunk at
frame index 0. It does not see sender-only costs or true span boundaries. A cover-aware changed-fraction
is recorded as an oracle in the manifest and is not a row in this table.

**Power** asks how many independent lineages this run's classifier dispersion would need before a shift to
D = 0.55 pushed the lineage interval off 0.5 in 90% of simulations. It is not a guarantee about a future corpus.
**Worst cell** is the highest D among category rows, or among the aggregate detectors when the corpus has one
category. **Leave-one-lineage** restricts the classifier's existing file scores to each lineage that has two or
more carriers, and to the carriers that remain. It does not retrain.

**Message size** compares a 64-byte message with a 1-byte one. Mist changes the same amount of audio whatever
the message, so this should read 0.5: anything else means the message length shows.

**Canonical workflow** is ffmpeg -q:a N at the Vorbis level Mist chose, and ffmpeg's defaults for a lossless
codec. The fingerprint verdict uses that file. A nominal-rate gap against ffmpeg's default quality is reported
beside it and does not fail the verdict: that warden re-encoded at a different setting. Ogg serial numbers are
random on every encode, so the audit compares how many serials a file has, not the number itself.

**Metadata warden** is a logistic regression on file metadata only, with the same nested lineage folds as the
audio classifier when there are at least four lineages. Its pass is a file interval that includes 0.5. File
size and packet size are features here; they are not required to be equal.

### Audio quality

**SDR** (signal-to-distortion ratio) is how loud the music is compared with the error added to it, in dB. Higher
is better, and every 10 dB means ten times less error. Both copies are compared with the original carrier,
after lining them up in time. The **plain re-encode** is Mist's own re-encode with nothing embedded; the **embedding cost** is how
much lower the stego copy scores, which is Mist's own share. **Extra error energy** says the same thing as a
percentage: +37% means 37% more error than the plain re-encode alone.

The embedding cost is relative to the re-encode's own error, so the same perturbation reads as a larger cost on a
cleaner, higher-bitrate re-encode. **Mist's error below the music** does not depend on that: it compares the
stego copy with Mist's own re-encode, so the only difference left is what Mist changed, measured against the music
itself. The **per carrier** table lists every track with its output bitrate, so one odd track cannot hide in the
mean.

### Thresholds

| Measure | ✅ | ⚠️ | ❌ |
|---|---|---|---|
| Canonical properties | every carrier matches the canonical ffmpeg encode | — | any identity field differs |
| Metadata warden | file interval includes 0.5 | not scored | interval excludes 0.5 |
| Detector AUC | 95% interval includes 0.5 | interval excludes 0.5, AUC within 0.4–0.6 | AUC outside 0.4–0.6 |
| Message size | 0.45–0.55 | — | outside 0.45–0.55 |
| Embedding cost | ≤ 0.3 dB, or Mist output ≥ 70 dB SDR | 0.3–1 dB | over 1 dB |

### Reference points

- **0.3 dB embedding cost** and **AUC ≈ 0.5** are the roadmap's Phase 1 exit criteria.
- **0.18 dB** is what the design doc measured for Ogg Vorbis on one 128 kbps rock MP3.
- **70 dB SDR** puts the error far below anything audible at normal listening levels. 16-bit audio's own
  rounding noise sits about 96 dB down, which is why a lossless output can score ∞ for a plain re-encode
  (bit-exact) and pass on SDR alone.
- **A lossy carrier into a lossless output** is not bit-exact: an MP3 decodes to values between 16-bit steps,
  so the plain re-encode lands around 80–90 dB rather than ∞.

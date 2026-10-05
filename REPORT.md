# Mist harness report

> **Phase 1 baseline, 2026-10-05.** 
> This run only changes how detectors are scored.
>
> Corpus name `metal-dev`: 84 carriers, the first 300 seconds of each. Public ids are `carrier-NNNN`. There is no session manifest, so each file is its own lineage (84 groups). Intervals do not account for album or session dependence. One category, `external`, so there is no per-category table and no leave-one-lineage table. Perceptual metric not installed. Seed 1. Permutation rounds 199, classifier and Markov refits 9.
>
> This is not a Cachin ε and not a claim that the embedding is undetectable. The KL column is a detector-implied lower bound for this benchmark. A file interval that excludes 0.5 is a faint signal on that detector; Holm or Benjamini-Hochberg is the confirmatory or exploratory adjustment. Machine-readable `manifest.json`, `report.json` and `scores.json` stay in the local harness output and are not in the tree. Filenames and machine paths are not recorded here.

## Headline file results

Stego against the clean ffmpeg copy. D = 0.5 + |file AUC − 0.5|.

| Format | Detector | File AUC (95%) | D | Adjusted p | KL lower bound (nats, 95%) |
|---|---|---|---|---|---|
| flac | classifier | 0.555 (0.527–0.585) | 0.555 | Holm 0.340 | 0.006 (0.001–0.015) |
| flac | markov | 0.549 (0.524–0.579) | 0.549 | Holm 0.340 | 0.005 (0.001–0.013) |
| flac | hcf-com | 0.501 (0.500–0.503) | 0.501 | Holm 0.340 | 0.000 (0.000–0.000) |
| flac | chi-square | 0.502 (0.501–0.503) | 0.502 | BH 0.015 | 0.000 (0.000–0.000) |
| wav/pcm_s16le | classifier | 0.507 (0.502–0.513) | 0.507 | Holm 0.300 | 0.000 (0.000–0.000) |
| wav/pcm_s16le | hcf-com | 0.502 (0.500–0.503) | 0.502 | Holm 0.120 | 0.000 (0.000–0.000) |
| ogg/vorbis | hcf-com | 0.508 (0.506–0.511) | 0.508 | Holm 0.020 | 0.000 (0.000–0.000) |
| ogg/vorbis | markov | 0.488 (0.483–0.492) | 0.512 | Holm 0.300 | 0.000 (0.000–0.001) |
| ogg/vorbis | classifier | 0.492 (0.490–0.493) | 0.508 | Holm 0.300 | 0.000 (0.000–0.000) |

Key-aware file AUC stays inside an interval that contains 0.5 on all three formats (FLAC 0.500, WAV 0.476, Vorbis 0.506). Message-size checks sit at 0.5. FLAC matches ffmpeg on every carrier. WAV differs in the zero tail on 5 of 84. Vorbis differs in nominal bitrate, and in the zero tail, on 73 of 84. Embedding cost on Vorbis is 0.01 dB. The power simulation asks for about 128 lineages to detect D = 0.55 on this FLAC classifier dispersion, and about 8 on WAV and Vorbis; that is a simulation, not a sample-size guarantee.

The sections below are the harness report this baseline is copied from.

# Mist harness report

Commit `b610b12-dirty` · 2026-10-05 · corpus: metal-dev, first 300 s of each carrier (84 carriers, 84 independent lineage groups) · perceptual metric: not installed

Every carrier is encoded three times: by the ffmpeg command line at its own defaults (the *clean* copy, which is what a warden without the original would compare against), by Mist's own encoder with nothing embedded (Mist's *own* re-encode), and by Mist with a hidden message (the *stego* copy). The report asks whether the stego copy differs from the clean one in plain properties, whether a detector can tell them apart, and how much worse it sounds. [How to read this report](#how-to-read-this-report) explains every number and threshold.

## Summary

| Format | Carriers | Looks like ffmpeg? | Hidden from detectors? | Audio quality |
|---|---|---|---|---|
| flac | 84 / 84 | ✅ matches ffmpeg on all 84 carriers | ⚠️ faint signal from chi-square (AUC 0.500) | ✅ inaudible: the added error is at least 94.07 dB below the music |
| wav/pcm_s16le | 84 / 84 | ❌ differs on 5 of 84 carriers: zero tail | ⚠️ faint signal from hcf-com (AUC 0.500) | ✅ inaudible: the added error is at least 83.98 dB below the music |
| ogg/vorbis | 84 / 84 | ❌ differs on 73 of 84 carriers: nominal bitrate, zero tail | ⚠️ faint signal from hcf-com (AUC 0.501) | ✅ Mist's own error sits 55.06 dB below the music; embedding costs 0.01 dB, within the 0.3 dB target |

## flac

Measured on 84 of 84 carriers.

### Does it look like a plain ffmpeg encode?

The ffmpeg column is ffmpeg at its own defaults. Ogg Vorbis keeps the source's quality, so it differs from ffmpeg's default q3 whenever the carrier maps to another level.

| Carrier | Source samples | Samples (ffmpeg / Mist) | Zero tail (ffmpeg / Mist) | Sample format (ffmpeg / Mist) | Nominal kbps (ffmpeg / Mist) | Differs in |
|---|---|---|---|---|---|---|
| carrier-0001 | 13230191 | 13230191 / 13230191 | 0 / 0 | s32 / s32 | 0 / 0 | — |
| carrier-0002 | 13230191 | 13230191 / 13230191 | 0 / 0 | s32 / s32 | 0 / 0 | — |
| carrier-0003 | 13230191 | 13230191 / 13230191 | 0 / 0 | s32 / s32 | 0 / 0 | — |
| carrier-0004 | 11669231 | 11669231 / 11669231 | 0 / 0 | s32 / s32 | 0 / 0 | — |
| carrier-0005 | 13230191 | 13230191 / 13230191 | 0 / 0 | s32 / s32 | 0 / 0 | — |
| carrier-0006 | 13230191 | 13230191 / 13230191 | 0 / 0 | s32 / s32 | 0 / 0 | — |
| carrier-0007 | 13230191 | 13230191 / 13230191 | 0 / 0 | s32 / s32 | 0 / 0 | — |
| carrier-0008 | 13230191 | 13230191 / 13230191 | 0 / 0 | s32 / s32 | 0 / 0 | — |
| carrier-0009 | 11631215 | 11631215 / 11631215 | 288 / 288 | s32 / s32 | 0 / 0 | — |
| carrier-0010 | 12556271 | 12556271 / 12556271 | 30761 / 30761 | s32 / s32 | 0 / 0 | — |
| carrier-0011 | 13230767 | 13230767 / 13230767 | 0 / 0 | s32 / s32 | 0 / 0 | — |
| carrier-0012 | 13230767 | 13230767 / 13230767 | 0 / 0 | s32 / s32 | 0 / 0 | — |
| carrier-0013 | 13230767 | 13230767 / 13230767 | 0 / 0 | s32 / s32 | 0 / 0 | — |
| carrier-0014 | 13230767 | 13230767 / 13230767 | 0 / 0 | s32 / s32 | 0 / 0 | — |
| carrier-0015 | 2880000 | 2880000 / 2880000 | 0 / 0 | s16 / s16 | 0 / 0 | — |
| carrier-0016 | 5760000 | 5760000 / 5760000 | 0 / 0 | s16 / s16 | 0 / 0 | — |
| carrier-0017 | 5760000 | 5760000 / 5760000 | 0 / 0 | s16 / s16 | 0 / 0 | — |
| carrier-0018 | 28803072 | 28803072 / 28803072 | 0 / 0 | s32 / s32 | 0 / 0 | — |
| carrier-0019 | 28803072 | 28803072 / 28803072 | 0 / 0 | s32 / s32 | 0 / 0 | — |
| carrier-0020 | 27650560 | 27650560 / 27650560 | 0 / 0 | s32 / s32 | 0 / 0 | — |
| carrier-0021 | 27998720 | 27998720 / 27998720 | 0 / 0 | s32 / s32 | 0 / 0 | — |
| carrier-0022 | 24209920 | 24209920 / 24209920 | 0 / 0 | s32 / s32 | 0 / 0 | — |
| carrier-0023 | 28803072 | 28803072 / 28803072 | 0 / 0 | s32 / s32 | 0 / 0 | — |
| carrier-0024 | 24574720 | 24574720 / 24574720 | 0 / 0 | s32 / s32 | 0 / 0 | — |
| carrier-0025 | 26830080 | 26830080 / 26830080 | 0 / 0 | s32 / s32 | 0 / 0 | — |
| carrier-0026 | 24695040 | 24695040 / 24695040 | 0 / 0 | s32 / s32 | 0 / 0 | — |
| carrier-0027 | 18909440 | 18909440 / 18909440 | 0 / 0 | s32 / s32 | 0 / 0 | — |
| carrier-0028 | 28803072 | 28803072 / 28803072 | 0 / 0 | s32 / s32 | 0 / 0 | — |
| carrier-0029 | 7939176 | 7939176 / 7939176 | 0 / 0 | s16 / s16 | 0 / 0 | — |
| carrier-0030 | 8334312 | 8334312 / 8334312 | 0 / 0 | s16 / s16 | 0 / 0 | — |
| carrier-0031 | 8187312 | 8187312 / 8187312 | 0 / 0 | s16 / s16 | 0 / 0 | — |
| carrier-0032 | 7616952 | 7616952 / 7616952 | 0 / 0 | s16 / s16 | 0 / 0 | — |
| carrier-0033 | 7703976 | 7703976 / 7703976 | 0 / 0 | s16 / s16 | 0 / 0 | — |
| carrier-0034 | 7795116 | 7795116 / 7795116 | 0 / 0 | s16 / s16 | 0 / 0 | — |
| carrier-0035 | 8786484 | 8786484 / 8786484 | 0 / 0 | s16 / s16 | 0 / 0 | — |
| carrier-0036 | 8306676 | 8306676 / 8306676 | 0 / 0 | s16 / s16 | 0 / 0 | — |
| carrier-0037 | 7848624 | 7848624 / 7848624 | 0 / 0 | s16 / s16 | 0 / 0 | — |
| carrier-0038 | 7849212 | 7849212 / 7849212 | 0 / 0 | s16 / s16 | 0 / 0 | — |
| carrier-0039 | 8371356 | 8371356 / 8371356 | 0 / 0 | s16 / s16 | 0 / 0 | — |
| carrier-0040 | 7808052 | 7808052 / 7808052 | 0 / 0 | s16 / s16 | 0 / 0 | — |
| carrier-0041 | 7792764 | 7792764 / 7792764 | 0 / 0 | s16 / s16 | 0 / 0 | — |
| carrier-0042 | 7705152 | 7705152 / 7705152 | 0 / 0 | s16 / s16 | 0 / 0 | — |
| carrier-0043 | 7733376 | 7733376 / 7733376 | 0 / 0 | s16 / s16 | 0 / 0 | — |
| carrier-0044 | 8332548 | 8332548 / 8332548 | 0 / 0 | s16 / s16 | 0 / 0 | — |
| carrier-0045 | 7836864 | 7836864 / 7836864 | 0 / 0 | s16 / s16 | 0 / 0 | — |
| carrier-0046 | 7783356 | 7783356 / 7783356 | 0 / 0 | s16 / s16 | 0 / 0 | — |
| carrier-0047 | 8134980 | 8134980 / 8134980 | 0 / 0 | s16 / s16 | 0 / 0 | — |
| carrier-0048 | 7904484 | 7904484 / 7904484 | 0 / 0 | s16 / s16 | 0 / 0 | — |
| carrier-0049 | 7577556 | 7577556 / 7577556 | 0 / 0 | s16 / s16 | 0 / 0 | — |
| carrier-0050 | 8433684 | 8433684 / 8433684 | 0 / 0 | s16 / s16 | 0 / 0 | — |
| carrier-0051 | 8167320 | 8167320 / 8167320 | 0 / 0 | s16 / s16 | 0 / 0 | — |
| carrier-0052 | 7443492 | 7443492 / 7443492 | 0 / 0 | s16 / s16 | 0 / 0 | — |
| carrier-0053 | 7847448 | 7847448 / 7847448 | 0 / 0 | s16 / s16 | 0 / 0 | — |
| carrier-0054 | 6560316 | 6560316 / 6560316 | 20 / 20 | s16 / s16 | 0 / 0 | — |
| carrier-0055 | 7567536 | 7567536 / 7567536 | 1004 / 1004 | s32 / s32 | 0 / 0 | — |
| carrier-0056 | 10716829 | 10716829 / 10716829 | 2375 / 2375 | s32 / s32 | 0 / 0 | — |
| carrier-0057 | 10753254 | 10753254 / 10753254 | 462 / 462 | s32 / s32 | 0 / 0 | — |
| carrier-0058 | 8507078 | 8507078 / 8507078 | 0 / 0 | s32 / s32 | 0 / 0 | — |
| carrier-0059 | 9099131 | 9099131 / 9099131 | 0 / 0 | s32 / s32 | 0 / 0 | — |
| carrier-0060 | 10184328 | 10184328 / 10184328 | 0 / 0 | s32 / s32 | 0 / 0 | — |
| carrier-0061 | 9848955 | 9848955 / 9848955 | 0 / 0 | s32 / s32 | 0 / 0 | — |
| carrier-0062 | 10444219 | 10444219 / 10444219 | 1522 / 1522 | s32 / s32 | 0 / 0 | — |
| carrier-0063 | 10854001 | 10854001 / 10854001 | 0 / 0 | s32 / s32 | 0 / 0 | — |
| carrier-0064 | 9693408 | 9693408 / 9693408 | 0 / 0 | s32 / s32 | 0 / 0 | — |
| carrier-0065 | 9766958 | 9766958 / 9766958 | 0 / 0 | s32 / s32 | 0 / 0 | — |
| carrier-0066 | 9359938 | 9359938 / 9359938 | 0 / 0 | s32 / s32 | 0 / 0 | — |
| carrier-0067 | 9855000 | 9855000 / 9855000 | 8479 / 8479 | s32 / s32 | 0 / 0 | — |
| carrier-0068 | 9146026 | 9146026 / 9146026 | 276 / 276 | s32 / s32 | 0 / 0 | — |
| carrier-0069 | 10717286 | 10717286 / 10717286 | 0 / 0 | s32 / s32 | 0 / 0 | — |
| carrier-0070 | 8370000 | 8370000 / 8370000 | 0 / 0 | s32 / s32 | 0 / 0 | — |
| carrier-0071 | 10456119 | 10456119 / 10456119 | 23958 / 23958 | s32 / s32 | 0 / 0 | — |
| carrier-0072 | 9573914 | 9573914 / 9573914 | 7702 / 7702 | s32 / s32 | 0 / 0 | — |
| carrier-0073 | 10171848 | 10171848 / 10171848 | 826 / 826 | s32 / s32 | 0 / 0 | — |
| carrier-0074 | 9735652 | 9735652 / 9735652 | 0 / 0 | s32 / s32 | 0 / 0 | — |
| carrier-0075 | 9412488 | 9412488 / 9412488 | 7538 / 7538 | s32 / s32 | 0 / 0 | — |
| carrier-0076 | 9888000 | 9888000 / 9888000 | 0 / 0 | s32 / s32 | 0 / 0 | — |
| carrier-0077 | 10440000 | 10440000 / 10440000 | 0 / 0 | s32 / s32 | 0 / 0 | — |
| carrier-0078 | 11520002 | 11520002 / 11520002 | 0 / 0 | s32 / s32 | 0 / 0 | — |
| carrier-0079 | 10450747 | 10450747 / 10450747 | 21505 / 21505 | s32 / s32 | 0 / 0 | — |
| carrier-0080 | 8598259 | 8598259 / 8598259 | 0 / 0 | s32 / s32 | 0 / 0 | — |
| carrier-0081 | 11520000 | 11520000 / 11520000 | 0 / 0 | s32 / s32 | 0 / 0 | — |
| carrier-0082 | 11102609 | 11102609 / 11102609 | 0 / 0 | s32 / s32 | 0 / 0 | — |
| carrier-0083 | 8540309 | 8540309 / 8540309 | 7055 / 7055 | s32 / s32 | 0 / 0 | — |
| carrier-0084 | 10279847 | 10279847 / 10279847 | 7063 / 7063 | s32 / s32 | 0 / 0 | — |

**Verdict:** ✅ matches ffmpeg on all 84 carriers.

### Can a detector tell?

Stego copy against the clean ffmpeg copy (for Ogg Vorbis, ffmpeg at the quality level Mist chose, so only the embedding differs).

| Detector | Looks for | Chunk AUC (95%) | File AUC (95%) | D | Detector-implied benchmark KL lower bound (nats, 95%) | Adjusted p | Verdict | Message size |
|---|---|---|---|---|---|---|---|---|
| chi-square | bit overwriting (not what Mist does) | 0.500 (0.500–0.500) | 0.502 (0.501–0.503) | 0.502 | 0.000 (0.000–0.000) | BH 0.015 | ⚠️ faint signal | ✅ hidden (0.500) |
| spa | bit overwriting (not what Mist does) | 0.500 (0.500–0.501) | 0.502 (0.498–0.507) | 0.502 | 0.000 (0.000–0.000) | BH 0.413 | ✅ chance | ✅ hidden (0.500) |
| rs | bit overwriting (not what Mist does) | 0.500 (0.499–0.500) | 0.501 (0.496–0.506) | 0.501 | 0.000 (0.000–0.000) | BH 0.620 | ✅ chance | ✅ hidden (0.500) |
| hcf-com | ±1 changes, which is what Mist does | 0.500 (0.500–0.500) | 0.501 (0.500–0.503) | 0.501 | 0.000 (0.000–0.000) | Holm 0.340 | ⚠️ faint signal | ✅ hidden (0.500) |
| classifier | anything it can learn from Mist's own output | 0.506 (0.503–0.510) | 0.555 (0.527–0.585) | 0.555 | 0.006 (0.001–0.015) | Holm 0.340 | ⚠️ faint signal | ✅ hidden (0.500) |
| markov | how the waveform's curvature changes from sample to sample | 0.504 (0.502–0.506) | 0.549 (0.524–0.579) | 0.549 | 0.005 (0.001–0.013) | Holm 0.340 | ⚠️ faint signal | ✅ hidden (0.500) |
| key-aware | the ephemeral key in the first frame's envelope, read with the public key alone | 0.500 (0.465–0.530) | 0.500 (0.465–0.530) | 0.500 | 0.000 (0.000–0.002) | Holm 1.000 | ✅ chance | ✅ hidden (0.500) |

Power: shifting this run's classifier file scores to D = 0.55, about 128 independent lineages give 90% power for a lineage-cluster interval to exclude 0.5. The figure is a simulation from this run's dispersion.

Worst cell: classifier on aggregate, file AUC 0.555, D 0.555.

**Verdict:** ⚠️ faint signal from chi-square (AUC 0.500).

### The embedding alone

Stego copy against Mist's own re-encode, so only the embedded changes differ.

| Detector | Looks for | Chunk AUC (95%) | File AUC (95%) | D | Detector-implied benchmark KL lower bound (nats, 95%) | Adjusted p | Verdict | Message size |
|---|---|---|---|---|---|---|---|---|
| chi-square | bit overwriting (not what Mist does) | 0.500 (0.500–0.500) | 0.502 (0.501–0.503) | 0.502 | 0.000 (0.000–0.000) | — | ⚠️ faint signal | ✅ hidden (0.500) |
| spa | bit overwriting (not what Mist does) | 0.500 (0.500–0.501) | 0.502 (0.498–0.507) | 0.502 | 0.000 (0.000–0.000) | — | ✅ chance | ✅ hidden (0.500) |
| rs | bit overwriting (not what Mist does) | 0.500 (0.499–0.500) | 0.501 (0.496–0.506) | 0.501 | 0.000 (0.000–0.000) | — | ✅ chance | ✅ hidden (0.500) |
| hcf-com | ±1 changes, which is what Mist does | 0.500 (0.500–0.500) | 0.501 (0.500–0.503) | 0.501 | 0.000 (0.000–0.000) | — | ⚠️ faint signal | ✅ hidden (0.500) |
| classifier | anything it can learn from Mist's own output | 0.506 (0.503–0.510) | 0.555 (0.527–0.585) | 0.555 | 0.006 (0.001–0.015) | — | ⚠️ faint signal | ✅ hidden (0.500) |
| markov | how the waveform's curvature changes from sample to sample | 0.504 (0.502–0.506) | 0.549 (0.524–0.579) | 0.549 | 0.005 (0.001–0.013) | — | ⚠️ faint signal | ✅ hidden (0.500) |

### Does detection grow with audio?

File AUC (95%) against the clean ffmpeg copy on the first chunks of each carrier. A chunk is 65,536 values, about 0.74 s of 44.1 kHz stereo.

| Detector | First 40 chunks | First 240 chunks | Whole file |
|---|---|---|---|
| chi-square | 0.501 (0.500–0.502) | 0.502 (0.500–0.503) | 0.502 (0.501–0.503) |
| spa | 0.498 (0.493–0.503) | 0.500 (0.497–0.504) | 0.502 (0.498–0.507) |
| rs | 0.500 (0.493–0.506) | 0.503 (0.498–0.508) | 0.501 (0.496–0.506) |
| hcf-com | 0.500 (0.500–0.501) | 0.500 (0.499–0.501) | 0.501 (0.500–0.503) |
| classifier | 0.537 (0.517–0.563) | 0.546 (0.522–0.576) | 0.555 (0.527–0.585) |
| markov | 0.532 (0.513–0.554) | 0.546 (0.522–0.573) | 0.549 (0.524–0.579) |

### Does detection grow with the number of files?

AUC when the warden averages a detector's score over k files drawn at random from the corpus, stego against clean. Draws overlap, so there is no interval.

| Detector | 1 file(s) | 3 file(s) | 6 file(s) |
|---|---|---|---|
| chi-square | 0.489 | 0.482 | 0.482 |
| spa | 0.530 | 0.505 | 0.517 |
| rs | 0.530 | 0.508 | 0.511 |
| hcf-com | 0.491 | 0.484 | 0.481 |
| classifier | 0.572 | 0.556 | 0.547 |
| markov | 0.558 | 0.549 | 0.545 |

### How much does it cost in sound?

| Measure | Mean | Worst | What it means |
|---|---|---|---|
| Plain re-encode SDR | 65.46 dB | 35.91 dB | Loss of Mist's own re-encode, with nothing embedded |
| Mist output SDR | 86.74 dB | 35.91 dB | The same, with the message embedded |
| Embedding cost | 0.00 dB | 0.02 dB | Mist's own share; target ≤ 0.3 dB, design-doc reference 0.18 dB |
| Extra error energy | +0% | +0% | Embedding cost as error added on top of a plain re-encode |
| Mist's error below the music | 134.29 dB | 94.07 dB | Stego copy against Mist's own re-encode: what embedding alone adds |

**Verdict:** ✅ inaudible: the added error is at least 94.07 dB below the music.

### Per carrier

| Carrier | Output kbps | Plain re-encode SDR | Mist output SDR | Embedding cost | Mist's error below the music |
|---|---|---|---|---|---|
| carrier-0001 | 1594 | 128.79 dB | 128.77 dB | 0.02 dB | 152.17 dB |
| carrier-0002 | 1588 | 128.91 dB | 128.89 dB | 0.02 dB | 152.29 dB |
| carrier-0003 | 1576 | 130.61 dB | 130.59 dB | 0.02 dB | 153.86 dB |
| carrier-0004 | 1545 | 128.38 dB | 128.36 dB | 0.02 dB | 151.80 dB |
| carrier-0005 | 1563 | 129.30 dB | 129.28 dB | 0.02 dB | 152.66 dB |
| carrier-0006 | 1574 | 129.70 dB | 129.68 dB | 0.02 dB | 153.05 dB |
| carrier-0007 | 1598 | 130.09 dB | 130.07 dB | 0.02 dB | 153.39 dB |
| carrier-0008 | 1549 | 130.75 dB | 130.73 dB | 0.02 dB | 154.02 dB |
| carrier-0009 | 1531 | 128.88 dB | 128.86 dB | 0.02 dB | 152.29 dB |
| carrier-0010 | 1449 | 129.02 dB | 129.00 dB | 0.02 dB | 152.45 dB |
| carrier-0011 | 1518 | 64.22 dB | 64.22 dB | 0.00 dB | 155.26 dB |
| carrier-0012 | 1663 | 50.07 dB | 50.07 dB | 0.00 dB | 154.35 dB |
| carrier-0013 | 1553 | 46.00 dB | 46.00 dB | 0.00 dB | 152.59 dB |
| carrier-0014 | 1443 | 43.92 dB | 43.92 dB | 0.00 dB | 153.13 dB |
| carrier-0015 | 968 | ∞ | 103.11 dB | ∞ | 103.11 dB |
| carrier-0016 | 1046 | ∞ | 105.28 dB | ∞ | 105.28 dB |
| carrier-0017 | 978 | ∞ | 106.00 dB | ∞ | 106.00 dB |
| carrier-0018 | 2297 | ∞ | 134.66 dB | ∞ | 134.66 dB |
| carrier-0019 | 2258 | ∞ | 132.11 dB | ∞ | 132.11 dB |
| carrier-0020 | 2593 | ∞ | 150.09 dB | ∞ | 150.09 dB |
| carrier-0021 | 2648 | ∞ | 149.81 dB | ∞ | 149.81 dB |
| carrier-0022 | 2676 | ∞ | 151.33 dB | ∞ | 151.33 dB |
| carrier-0023 | 2583 | ∞ | 148.63 dB | ∞ | 148.63 dB |
| carrier-0024 | 2685 | ∞ | 149.65 dB | ∞ | 149.65 dB |
| carrier-0025 | 2698 | ∞ | 150.01 dB | ∞ | 150.01 dB |
| carrier-0026 | 2688 | ∞ | 150.50 dB | ∞ | 150.50 dB |
| carrier-0027 | 2661 | ∞ | 150.48 dB | ∞ | 150.48 dB |
| carrier-0028 | 2638 | ∞ | 149.98 dB | ∞ | 149.98 dB |
| carrier-0029 | 466 | ∞ | 95.57 dB | ∞ | 95.57 dB |
| carrier-0030 | 463 | ∞ | 95.28 dB | ∞ | 95.28 dB |
| carrier-0031 | 461 | ∞ | 95.18 dB | ∞ | 95.18 dB |
| carrier-0032 | 467 | ∞ | 95.10 dB | ∞ | 95.10 dB |
| carrier-0033 | 467 | ∞ | 95.60 dB | ∞ | 95.60 dB |
| carrier-0034 | 457 | ∞ | 94.90 dB | ∞ | 94.90 dB |
| carrier-0035 | 464 | ∞ | 95.11 dB | ∞ | 95.11 dB |
| carrier-0036 | 470 | ∞ | 95.52 dB | ∞ | 95.52 dB |
| carrier-0037 | 466 | ∞ | 94.89 dB | ∞ | 94.89 dB |
| carrier-0038 | 464 | ∞ | 95.12 dB | ∞ | 95.12 dB |
| carrier-0039 | 462 | ∞ | 94.91 dB | ∞ | 94.91 dB |
| carrier-0040 | 466 | ∞ | 94.89 dB | ∞ | 94.89 dB |
| carrier-0041 | 465 | ∞ | 95.26 dB | ∞ | 95.26 dB |
| carrier-0042 | 473 | ∞ | 95.33 dB | ∞ | 95.33 dB |
| carrier-0043 | 459 | ∞ | 94.07 dB | ∞ | 94.07 dB |
| carrier-0044 | 468 | ∞ | 94.95 dB | ∞ | 94.95 dB |
| carrier-0045 | 482 | ∞ | 96.02 dB | ∞ | 96.02 dB |
| carrier-0046 | 458 | ∞ | 95.37 dB | ∞ | 95.37 dB |
| carrier-0047 | 469 | ∞ | 95.40 dB | ∞ | 95.40 dB |
| carrier-0048 | 467 | ∞ | 95.55 dB | ∞ | 95.55 dB |
| carrier-0049 | 461 | ∞ | 95.19 dB | ∞ | 95.19 dB |
| carrier-0050 | 476 | ∞ | 95.27 dB | ∞ | 95.27 dB |
| carrier-0051 | 461 | ∞ | 95.17 dB | ∞ | 95.17 dB |
| carrier-0052 | 463 | ∞ | 95.09 dB | ∞ | 95.09 dB |
| carrier-0053 | 464 | ∞ | 95.00 dB | ∞ | 95.00 dB |
| carrier-0054 | 473 | ∞ | 94.98 dB | ∞ | 94.98 dB |
| carrier-0055 | 1874 | 46.77 dB | 46.77 dB | 0.00 dB | 157.69 dB |
| carrier-0056 | 1948 | 36.65 dB | 36.65 dB | 0.00 dB | 159.58 dB |
| carrier-0057 | 1832 | 45.62 dB | 45.62 dB | 0.00 dB | 157.09 dB |
| carrier-0058 | 1910 | 35.91 dB | 35.91 dB | 0.00 dB | 157.98 dB |
| carrier-0059 | 1936 | 44.31 dB | 44.31 dB | 0.00 dB | 157.88 dB |
| carrier-0060 | 1873 | 58.54 dB | 58.54 dB | 0.00 dB | 155.00 dB |
| carrier-0061 | 1854 | 41.20 dB | 41.20 dB | 0.00 dB | 158.02 dB |
| carrier-0062 | 1916 | 41.33 dB | 41.33 dB | 0.00 dB | 158.45 dB |
| carrier-0063 | 1929 | 43.97 dB | 43.97 dB | 0.00 dB | 158.95 dB |
| carrier-0064 | 1769 | 68.84 dB | 68.84 dB | 0.00 dB | 154.86 dB |
| carrier-0065 | 1860 | 52.48 dB | 52.48 dB | 0.00 dB | 157.02 dB |
| carrier-0066 | 1894 | 43.08 dB | 43.08 dB | 0.00 dB | 157.75 dB |
| carrier-0067 | 1965 | 46.30 dB | 46.30 dB | 0.00 dB | 158.63 dB |
| carrier-0068 | 1927 | 38.87 dB | 38.87 dB | 0.00 dB | 159.40 dB |
| carrier-0069 | 1850 | 54.46 dB | 54.46 dB | 0.00 dB | 157.64 dB |
| carrier-0070 | 1767 | 47.32 dB | 47.32 dB | 0.00 dB | 156.82 dB |
| carrier-0071 | 1891 | 43.18 dB | 43.18 dB | 0.00 dB | 156.94 dB |
| carrier-0072 | 1921 | 45.06 dB | 45.06 dB | 0.00 dB | 158.62 dB |
| carrier-0073 | 1851 | 49.41 dB | 49.41 dB | 0.00 dB | 155.88 dB |
| carrier-0074 | 1899 | 40.06 dB | 40.06 dB | 0.00 dB | 158.35 dB |
| carrier-0075 | 1775 | 44.09 dB | 44.09 dB | 0.00 dB | 157.77 dB |
| carrier-0076 | 1984 | 39.55 dB | 39.55 dB | 0.00 dB | 158.92 dB |
| carrier-0077 | 1917 | 48.44 dB | 48.44 dB | 0.00 dB | 158.31 dB |
| carrier-0078 | 1914 | 40.85 dB | 40.85 dB | 0.00 dB | 159.24 dB |
| carrier-0079 | 1932 | 45.88 dB | 45.88 dB | 0.00 dB | 157.78 dB |
| carrier-0080 | 1938 | 46.94 dB | 46.94 dB | 0.00 dB | 157.91 dB |
| carrier-0081 | 1898 | 48.47 dB | 48.47 dB | 0.00 dB | 157.48 dB |
| carrier-0082 | 1852 | 43.09 dB | 43.09 dB | 0.00 dB | 156.78 dB |
| carrier-0083 | 1902 | 59.77 dB | 59.77 dB | 0.00 dB | 155.10 dB |
| carrier-0084 | 1813 | 41.02 dB | 41.02 dB | 0.00 dB | 158.56 dB |

## wav/pcm_s16le

Measured on 84 of 84 carriers.

### Does it look like a plain ffmpeg encode?

The ffmpeg column is ffmpeg at its own defaults. Ogg Vorbis keeps the source's quality, so it differs from ffmpeg's default q3 whenever the carrier maps to another level.

| Carrier | Source samples | Samples (ffmpeg / Mist) | Zero tail (ffmpeg / Mist) | Sample format (ffmpeg / Mist) | Nominal kbps (ffmpeg / Mist) | Differs in |
|---|---|---|---|---|---|---|
| carrier-0001 | 13230191 | 13230191 / 13230191 | 0 / 0 | s16 / s16 | 1411 / 1411 | — |
| carrier-0002 | 13230191 | 13230191 / 13230191 | 0 / 0 | s16 / s16 | 1411 / 1411 | — |
| carrier-0003 | 13230191 | 13230191 / 13230191 | 0 / 0 | s16 / s16 | 1411 / 1411 | — |
| carrier-0004 | 11669231 | 11669231 / 11669231 | 0 / 16 | s16 / s16 | 1411 / 1411 | zero tail |
| carrier-0005 | 13230191 | 13230191 / 13230191 | 0 / 0 | s16 / s16 | 1411 / 1411 | — |
| carrier-0006 | 13230191 | 13230191 / 13230191 | 0 / 0 | s16 / s16 | 1411 / 1411 | — |
| carrier-0007 | 13230191 | 13230191 / 13230191 | 0 / 0 | s16 / s16 | 1411 / 1411 | — |
| carrier-0008 | 13230191 | 13230191 / 13230191 | 0 / 0 | s16 / s16 | 1411 / 1411 | — |
| carrier-0009 | 11631215 | 11631215 / 11631215 | 300 / 300 | s16 / s16 | 1411 / 1411 | — |
| carrier-0010 | 12556271 | 12556271 / 12556271 | 30781 / 30781 | s16 / s16 | 1411 / 1411 | — |
| carrier-0011 | 13230767 | 13230767 / 13230767 | 0 / 0 | s16 / s16 | 1411 / 1411 | — |
| carrier-0012 | 13230767 | 13230767 / 13230767 | 0 / 0 | s16 / s16 | 1411 / 1411 | — |
| carrier-0013 | 13230767 | 13230767 / 13230767 | 0 / 0 | s16 / s16 | 1411 / 1411 | — |
| carrier-0014 | 13230767 | 13230767 / 13230767 | 0 / 0 | s16 / s16 | 1411 / 1411 | — |
| carrier-0015 | 2880000 | 2880000 / 2880000 | 0 / 0 | s16 / s16 | 1536 / 1536 | — |
| carrier-0016 | 5760000 | 5760000 / 5760000 | 0 / 0 | s16 / s16 | 1536 / 1536 | — |
| carrier-0017 | 5760000 | 5760000 / 5760000 | 0 / 0 | s16 / s16 | 1536 / 1536 | — |
| carrier-0018 | 28803072 | 28803072 / 28803072 | 0 / 0 | s16 / s16 | 3072 / 3072 | — |
| carrier-0019 | 28803072 | 28803072 / 28803072 | 0 / 0 | s16 / s16 | 3072 / 3072 | — |
| carrier-0020 | 27650560 | 27650560 / 27650560 | 0 / 0 | s16 / s16 | 3072 / 3072 | — |
| carrier-0021 | 27998720 | 27998720 / 27998720 | 0 / 0 | s16 / s16 | 3072 / 3072 | — |
| carrier-0022 | 24209920 | 24209920 / 24209920 | 0 / 0 | s16 / s16 | 3072 / 3072 | — |
| carrier-0023 | 28803072 | 28803072 / 28803072 | 0 / 0 | s16 / s16 | 3072 / 3072 | — |
| carrier-0024 | 24574720 | 24574720 / 24574720 | 0 / 0 | s16 / s16 | 3072 / 3072 | — |
| carrier-0025 | 26830080 | 26830080 / 26830080 | 0 / 0 | s16 / s16 | 3072 / 3072 | — |
| carrier-0026 | 24695040 | 24695040 / 24695040 | 0 / 0 | s16 / s16 | 3072 / 3072 | — |
| carrier-0027 | 18909440 | 18909440 / 18909440 | 0 / 0 | s16 / s16 | 3072 / 3072 | — |
| carrier-0028 | 28803072 | 28803072 / 28803072 | 0 / 0 | s16 / s16 | 3072 / 3072 | — |
| carrier-0029 | 7939176 | 7939176 / 7939176 | 0 / 0 | s16 / s16 | 1411 / 1411 | — |
| carrier-0030 | 8334312 | 8334312 / 8334312 | 0 / 0 | s16 / s16 | 1411 / 1411 | — |
| carrier-0031 | 8187312 | 8187312 / 8187312 | 0 / 0 | s16 / s16 | 1411 / 1411 | — |
| carrier-0032 | 7616952 | 7616952 / 7616952 | 0 / 0 | s16 / s16 | 1411 / 1411 | — |
| carrier-0033 | 7703976 | 7703976 / 7703976 | 0 / 0 | s16 / s16 | 1411 / 1411 | — |
| carrier-0034 | 7795116 | 7795116 / 7795116 | 0 / 0 | s16 / s16 | 1411 / 1411 | — |
| carrier-0035 | 8786484 | 8786484 / 8786484 | 0 / 0 | s16 / s16 | 1411 / 1411 | — |
| carrier-0036 | 8306676 | 8306676 / 8306676 | 0 / 0 | s16 / s16 | 1411 / 1411 | — |
| carrier-0037 | 7848624 | 7848624 / 7848624 | 0 / 0 | s16 / s16 | 1411 / 1411 | — |
| carrier-0038 | 7849212 | 7849212 / 7849212 | 0 / 0 | s16 / s16 | 1411 / 1411 | — |
| carrier-0039 | 8371356 | 8371356 / 8371356 | 0 / 0 | s16 / s16 | 1411 / 1411 | — |
| carrier-0040 | 7808052 | 7808052 / 7808052 | 0 / 0 | s16 / s16 | 1411 / 1411 | — |
| carrier-0041 | 7792764 | 7792764 / 7792764 | 0 / 0 | s16 / s16 | 1411 / 1411 | — |
| carrier-0042 | 7705152 | 7705152 / 7705152 | 0 / 0 | s16 / s16 | 1411 / 1411 | — |
| carrier-0043 | 7733376 | 7733376 / 7733376 | 0 / 0 | s16 / s16 | 1411 / 1411 | — |
| carrier-0044 | 8332548 | 8332548 / 8332548 | 0 / 0 | s16 / s16 | 1411 / 1411 | — |
| carrier-0045 | 7836864 | 7836864 / 7836864 | 0 / 0 | s16 / s16 | 1411 / 1411 | — |
| carrier-0046 | 7783356 | 7783356 / 7783356 | 0 / 0 | s16 / s16 | 1411 / 1411 | — |
| carrier-0047 | 8134980 | 8134980 / 8134980 | 0 / 0 | s16 / s16 | 1411 / 1411 | — |
| carrier-0048 | 7904484 | 7904484 / 7904484 | 0 / 0 | s16 / s16 | 1411 / 1411 | — |
| carrier-0049 | 7577556 | 7577556 / 7577556 | 0 / 0 | s16 / s16 | 1411 / 1411 | — |
| carrier-0050 | 8433684 | 8433684 / 8433684 | 0 / 0 | s16 / s16 | 1411 / 1411 | — |
| carrier-0051 | 8167320 | 8167320 / 8167320 | 0 / 0 | s16 / s16 | 1411 / 1411 | — |
| carrier-0052 | 7443492 | 7443492 / 7443492 | 0 / 0 | s16 / s16 | 1411 / 1411 | — |
| carrier-0053 | 7847448 | 7847448 / 7847448 | 0 / 0 | s16 / s16 | 1411 / 1411 | — |
| carrier-0054 | 6560316 | 6560316 / 6560316 | 20 / 20 | s16 / s16 | 1411 / 1411 | — |
| carrier-0055 | 7567536 | 7567536 / 7567536 | 2973 / 2973 | s16 / s16 | 1536 / 1536 | — |
| carrier-0056 | 10716829 | 10716829 / 10716829 | 2698 / 2698 | s16 / s16 | 1536 / 1536 | — |
| carrier-0057 | 10753254 | 10753254 / 10753254 | 466 / 466 | s16 / s16 | 1536 / 1536 | — |
| carrier-0058 | 8507078 | 8507078 / 8507078 | 3 / 3 | s16 / s16 | 1536 / 1536 | — |
| carrier-0059 | 9099131 | 9099131 / 9099131 | 0 / 0 | s16 / s16 | 1536 / 1536 | — |
| carrier-0060 | 10184328 | 10184328 / 10184328 | 1 / 1 | s16 / s16 | 1536 / 1536 | — |
| carrier-0061 | 9848955 | 9848955 / 9848955 | 0 / 0 | s16 / s16 | 1536 / 1536 | — |
| carrier-0062 | 10444219 | 10444219 / 10444219 | 1526 / 1526 | s16 / s16 | 1536 / 1536 | — |
| carrier-0063 | 10854001 | 10854001 / 10854001 | 0 / 0 | s16 / s16 | 1536 / 1536 | — |
| carrier-0064 | 9693408 | 9693408 / 9693408 | 1 / 0 | s16 / s16 | 1536 / 1536 | zero tail |
| carrier-0065 | 9766958 | 9766958 / 9766958 | 1 / 1 | s16 / s16 | 1536 / 1536 | — |
| carrier-0066 | 9359938 | 9359938 / 9359938 | 1 / 1 | s16 / s16 | 1536 / 1536 | — |
| carrier-0067 | 9855000 | 9855000 / 9855000 | 8479 / 8479 | s16 / s16 | 1536 / 1536 | — |
| carrier-0068 | 9146026 | 9146026 / 9146026 | 280 / 280 | s16 / s16 | 1536 / 1536 | — |
| carrier-0069 | 10717286 | 10717286 / 10717286 | 3 / 3 | s16 / s16 | 1536 / 1536 | — |
| carrier-0070 | 8370000 | 8370000 / 8370000 | 1 / 1 | s16 / s16 | 1536 / 1536 | — |
| carrier-0071 | 10456119 | 10456119 / 10456119 | 23962 / 23962 | s16 / s16 | 1536 / 1536 | — |
| carrier-0072 | 9573914 | 9573914 / 9573914 | 8695 / 8695 | s16 / s16 | 1536 / 1536 | — |
| carrier-0073 | 10171848 | 10171848 / 10171848 | 830 / 830 | s16 / s16 | 1536 / 1536 | — |
| carrier-0074 | 9735652 | 9735652 / 9735652 | 0 / 0 | s16 / s16 | 1536 / 1536 | — |
| carrier-0075 | 9412488 | 9412488 / 9412488 | 7542 / 7542 | s16 / s16 | 1536 / 1536 | — |
| carrier-0076 | 9888000 | 9888000 / 9888000 | 0 / 0 | s16 / s16 | 1536 / 1536 | — |
| carrier-0077 | 10440000 | 10440000 / 10440000 | 2 / 0 | s16 / s16 | 1536 / 1536 | zero tail |
| carrier-0078 | 11520002 | 11520002 / 11520002 | 0 / 0 | s16 / s16 | 1536 / 1536 | — |
| carrier-0079 | 10450747 | 10450747 / 10450747 | 21515 / 21515 | s16 / s16 | 1536 / 1536 | — |
| carrier-0080 | 8598259 | 8598259 / 8598259 | 5 / 4 | s16 / s16 | 1536 / 1536 | zero tail |
| carrier-0081 | 11520000 | 11520000 / 11520000 | 0 / 0 | s16 / s16 | 1536 / 1536 | — |
| carrier-0082 | 11102609 | 11102609 / 11102609 | 2 / 1 | s16 / s16 | 1536 / 1536 | zero tail |
| carrier-0083 | 8540309 | 8540309 / 8540309 | 7390 / 7390 | s16 / s16 | 1536 / 1536 | — |
| carrier-0084 | 10279847 | 10279847 / 10279847 | 7068 / 7068 | s16 / s16 | 1536 / 1536 | — |

**Verdict:** ❌ differs on 5 of 84 carriers: zero tail.

### Can a detector tell?

Stego copy against the clean ffmpeg copy (for Ogg Vorbis, ffmpeg at the quality level Mist chose, so only the embedding differs).

| Detector | Looks for | Chunk AUC (95%) | File AUC (95%) | D | Detector-implied benchmark KL lower bound (nats, 95%) | Adjusted p | Verdict | Message size |
|---|---|---|---|---|---|---|---|---|
| chi-square | bit overwriting (not what Mist does) | 0.500 (0.500–0.500) | 0.501 (0.499–0.502) | 0.501 | 0.000 (0.000–0.000) | BH 0.585 | ✅ chance | ✅ hidden (0.500) |
| spa | bit overwriting (not what Mist does) | 0.500 (0.499–0.500) | 0.498 (0.494–0.502) | 0.502 | 0.000 (0.000–0.000) | BH 0.465 | ✅ chance | ✅ hidden (0.500) |
| rs | bit overwriting (not what Mist does) | 0.500 (0.499–0.501) | 0.502 (0.498–0.507) | 0.502 | 0.000 (0.000–0.000) | BH 0.495 | ✅ chance | ✅ hidden (0.501) |
| hcf-com | ±1 changes, which is what Mist does | 0.500 (0.500–0.500) | 0.502 (0.500–0.503) | 0.502 | 0.000 (0.000–0.000) | Holm 0.120 | ⚠️ faint signal | ✅ hidden (0.500) |
| classifier | anything it can learn from Mist's own output | 0.502 (0.501–0.504) | 0.507 (0.502–0.513) | 0.507 | 0.000 (0.000–0.000) | Holm 0.300 | ⚠️ faint signal | ✅ hidden (0.500) |
| markov | how the waveform's curvature changes from sample to sample | 0.501 (0.500–0.501) | 0.503 (0.501–0.505) | 0.503 | 0.000 (0.000–0.000) | Holm 0.300 | ⚠️ faint signal | ✅ hidden (0.500) |
| key-aware | the ephemeral key in the first frame's envelope, read with the public key alone | 0.476 (0.441–0.506) | 0.476 (0.441–0.506) | 0.524 | 0.001 (0.000–0.007) | Holm 0.300 | ✅ chance | ✅ hidden (0.500) |

Power: shifting this run's classifier file scores to D = 0.55, about 8 independent lineages give 90% power for a lineage-cluster interval to exclude 0.5. The figure is a simulation from this run's dispersion.

Worst cell: key-aware on aggregate, file AUC 0.476, D 0.524.

**Verdict:** ⚠️ faint signal from hcf-com (AUC 0.500).

### The embedding alone

Stego copy against Mist's own re-encode, so only the embedded changes differ.

| Detector | Looks for | Chunk AUC (95%) | File AUC (95%) | D | Detector-implied benchmark KL lower bound (nats, 95%) | Adjusted p | Verdict | Message size |
|---|---|---|---|---|---|---|---|---|
| chi-square | bit overwriting (not what Mist does) | 0.500 (0.500–0.500) | 0.501 (0.499–0.502) | 0.501 | 0.000 (0.000–0.000) | — | ✅ chance | ✅ hidden (0.500) |
| spa | bit overwriting (not what Mist does) | 0.500 (0.499–0.500) | 0.498 (0.494–0.502) | 0.502 | 0.000 (0.000–0.000) | — | ✅ chance | ✅ hidden (0.500) |
| rs | bit overwriting (not what Mist does) | 0.500 (0.499–0.501) | 0.502 (0.497–0.507) | 0.502 | 0.000 (0.000–0.000) | — | ✅ chance | ✅ hidden (0.501) |
| hcf-com | ±1 changes, which is what Mist does | 0.500 (0.500–0.500) | 0.502 (0.500–0.503) | 0.502 | 0.000 (0.000–0.000) | — | ⚠️ faint signal | ✅ hidden (0.500) |
| classifier | anything it can learn from Mist's own output | 0.502 (0.501–0.504) | 0.507 (0.502–0.513) | 0.507 | 0.000 (0.000–0.000) | — | ⚠️ faint signal | ✅ hidden (0.500) |
| markov | how the waveform's curvature changes from sample to sample | 0.501 (0.500–0.501) | 0.503 (0.501–0.505) | 0.503 | 0.000 (0.000–0.000) | — | ⚠️ faint signal | ✅ hidden (0.500) |

### Does detection grow with audio?

File AUC (95%) against the clean ffmpeg copy on the first chunks of each carrier. A chunk is 65,536 values, about 0.74 s of 44.1 kHz stereo.

| Detector | First 40 chunks | First 240 chunks | Whole file |
|---|---|---|---|
| chi-square | 0.503 (0.500–0.508) | 0.501 (0.500–0.503) | 0.501 (0.499–0.502) |
| spa | 0.503 (0.498–0.508) | 0.501 (0.497–0.505) | 0.498 (0.494–0.502) |
| rs | 0.501 (0.494–0.508) | 0.503 (0.498–0.507) | 0.502 (0.498–0.507) |
| hcf-com | 0.501 (0.500–0.502) | 0.501 (0.499–0.502) | 0.502 (0.500–0.503) |
| classifier | 0.502 (0.499–0.505) | 0.507 (0.503–0.514) | 0.507 (0.502–0.513) |
| markov | 0.501 (0.499–0.503) | 0.507 (0.503–0.512) | 0.503 (0.501–0.505) |

### Does detection grow with the number of files?

AUC when the warden averages a detector's score over k files drawn at random from the corpus, stego against clean. Draws overlap, so there is no interval.

| Detector | 1 file(s) | 3 file(s) | 6 file(s) |
|---|---|---|---|
| chi-square | 0.543 | 0.491 | 0.507 |
| spa | 0.487 | 0.511 | 0.498 |
| rs | 0.488 | 0.511 | 0.494 |
| hcf-com | 0.474 | 0.482 | 0.476 |
| classifier | 0.530 | 0.521 | 0.509 |
| markov | 0.530 | 0.513 | 0.499 |

### How much does it cost in sound?

| Measure | Mean | Worst | What it means |
|---|---|---|---|
| Plain re-encode SDR | 58.45 dB | 35.91 dB | Loss of Mist's own re-encode, with nothing embedded |
| Mist output SDR | 71.47 dB | 35.91 dB | The same, with the message embedded |
| Embedding cost | 0.01 dB | 0.02 dB | Mist's own share; target ≤ 0.3 dB, design-doc reference 0.18 dB |
| Extra error energy | +0% | +0% | Embedding cost as error added on top of a plain re-encode |
| Mist's error below the music | 102.75 dB | 83.98 dB | Stego copy against Mist's own re-encode: what embedding alone adds |

**Verdict:** ✅ inaudible: the added error is at least 83.98 dB below the music.

### Per carrier

| Carrier | Output kbps | Plain re-encode SDR | Mist output SDR | Embedding cost | Mist's error below the music |
|---|---|---|---|---|---|
| carrier-0001 | 1411 | 79.97 dB | 79.95 dB | 0.02 dB | 104.00 dB |
| carrier-0002 | 1411 | 80.09 dB | 80.07 dB | 0.02 dB | 104.12 dB |
| carrier-0003 | 1411 | 81.66 dB | 81.64 dB | 0.02 dB | 105.70 dB |
| carrier-0004 | 1411 | 79.60 dB | 79.58 dB | 0.02 dB | 103.65 dB |
| carrier-0005 | 1411 | 80.45 dB | 80.43 dB | 0.02 dB | 104.48 dB |
| carrier-0006 | 1411 | 80.85 dB | 80.83 dB | 0.02 dB | 104.89 dB |
| carrier-0007 | 1411 | 81.19 dB | 81.18 dB | 0.02 dB | 105.22 dB |
| carrier-0008 | 1411 | 81.82 dB | 81.80 dB | 0.02 dB | 105.85 dB |
| carrier-0009 | 1411 | 80.09 dB | 80.07 dB | 0.02 dB | 104.13 dB |
| carrier-0010 | 1411 | 80.24 dB | 80.22 dB | 0.02 dB | 104.29 dB |
| carrier-0011 | 1411 | 64.16 dB | 64.16 dB | 0.00 dB | 107.10 dB |
| carrier-0012 | 1411 | 50.06 dB | 50.06 dB | 0.00 dB | 106.18 dB |
| carrier-0013 | 1411 | 45.99 dB | 45.99 dB | 0.00 dB | 104.43 dB |
| carrier-0014 | 1411 | 43.92 dB | 43.92 dB | 0.00 dB | 104.97 dB |
| carrier-0015 | 1536 | ∞ | 103.10 dB | ∞ | 103.10 dB |
| carrier-0016 | 1536 | ∞ | 105.29 dB | ∞ | 105.29 dB |
| carrier-0017 | 1536 | ∞ | 106.01 dB | ∞ | 106.01 dB |
| carrier-0018 | 3072 | 62.48 dB | 62.46 dB | 0.02 dB | 86.54 dB |
| carrier-0019 | 3072 | 59.93 dB | 59.91 dB | 0.02 dB | 83.98 dB |
| carrier-0020 | 3072 | 77.91 dB | 77.89 dB | 0.02 dB | 101.92 dB |
| carrier-0021 | 3072 | 77.64 dB | 77.62 dB | 0.02 dB | 101.64 dB |
| carrier-0022 | 3072 | 79.15 dB | 79.13 dB | 0.02 dB | 103.16 dB |
| carrier-0023 | 3072 | 76.46 dB | 76.44 dB | 0.02 dB | 100.47 dB |
| carrier-0024 | 3072 | 77.47 dB | 77.45 dB | 0.02 dB | 101.49 dB |
| carrier-0025 | 3072 | 77.83 dB | 77.81 dB | 0.02 dB | 101.84 dB |
| carrier-0026 | 3072 | 78.32 dB | 78.30 dB | 0.02 dB | 102.34 dB |
| carrier-0027 | 3072 | 78.30 dB | 78.28 dB | 0.02 dB | 102.31 dB |
| carrier-0028 | 3072 | 77.81 dB | 77.79 dB | 0.02 dB | 101.82 dB |
| carrier-0029 | 1411 | ∞ | 95.58 dB | ∞ | 95.58 dB |
| carrier-0030 | 1411 | ∞ | 95.29 dB | ∞ | 95.29 dB |
| carrier-0031 | 1411 | ∞ | 95.19 dB | ∞ | 95.19 dB |
| carrier-0032 | 1411 | ∞ | 95.10 dB | ∞ | 95.10 dB |
| carrier-0033 | 1411 | ∞ | 95.60 dB | ∞ | 95.60 dB |
| carrier-0034 | 1411 | ∞ | 94.91 dB | ∞ | 94.91 dB |
| carrier-0035 | 1411 | ∞ | 95.11 dB | ∞ | 95.11 dB |
| carrier-0036 | 1411 | ∞ | 95.52 dB | ∞ | 95.52 dB |
| carrier-0037 | 1411 | ∞ | 94.88 dB | ∞ | 94.88 dB |
| carrier-0038 | 1411 | ∞ | 95.11 dB | ∞ | 95.11 dB |
| carrier-0039 | 1411 | ∞ | 94.91 dB | ∞ | 94.91 dB |
| carrier-0040 | 1411 | ∞ | 94.89 dB | ∞ | 94.89 dB |
| carrier-0041 | 1411 | ∞ | 95.27 dB | ∞ | 95.27 dB |
| carrier-0042 | 1411 | ∞ | 95.33 dB | ∞ | 95.33 dB |
| carrier-0043 | 1411 | ∞ | 94.07 dB | ∞ | 94.07 dB |
| carrier-0044 | 1411 | ∞ | 94.96 dB | ∞ | 94.96 dB |
| carrier-0045 | 1411 | ∞ | 96.02 dB | ∞ | 96.02 dB |
| carrier-0046 | 1411 | ∞ | 95.37 dB | ∞ | 95.37 dB |
| carrier-0047 | 1411 | ∞ | 95.42 dB | ∞ | 95.42 dB |
| carrier-0048 | 1411 | ∞ | 95.55 dB | ∞ | 95.55 dB |
| carrier-0049 | 1411 | ∞ | 95.18 dB | ∞ | 95.18 dB |
| carrier-0050 | 1411 | ∞ | 95.27 dB | ∞ | 95.27 dB |
| carrier-0051 | 1411 | ∞ | 95.18 dB | ∞ | 95.18 dB |
| carrier-0052 | 1411 | ∞ | 95.08 dB | ∞ | 95.08 dB |
| carrier-0053 | 1411 | ∞ | 94.98 dB | ∞ | 94.98 dB |
| carrier-0054 | 1411 | ∞ | 94.99 dB | ∞ | 94.99 dB |
| carrier-0055 | 1536 | 46.77 dB | 46.77 dB | 0.00 dB | 109.52 dB |
| carrier-0056 | 1536 | 36.64 dB | 36.64 dB | 0.00 dB | 111.43 dB |
| carrier-0057 | 1536 | 45.62 dB | 45.62 dB | 0.00 dB | 108.93 dB |
| carrier-0058 | 1536 | 35.91 dB | 35.91 dB | 0.00 dB | 109.80 dB |
| carrier-0059 | 1536 | 44.31 dB | 44.31 dB | 0.00 dB | 109.72 dB |
| carrier-0060 | 1536 | 58.52 dB | 58.52 dB | 0.00 dB | 106.85 dB |
| carrier-0061 | 1536 | 41.19 dB | 41.19 dB | 0.00 dB | 109.85 dB |
| carrier-0062 | 1536 | 41.33 dB | 41.33 dB | 0.00 dB | 110.28 dB |
| carrier-0063 | 1536 | 43.97 dB | 43.97 dB | 0.00 dB | 110.77 dB |
| carrier-0064 | 1536 | 68.66 dB | 68.66 dB | 0.00 dB | 106.70 dB |
| carrier-0065 | 1536 | 52.48 dB | 52.48 dB | 0.00 dB | 108.88 dB |
| carrier-0066 | 1536 | 43.07 dB | 43.07 dB | 0.00 dB | 109.58 dB |
| carrier-0067 | 1536 | 46.30 dB | 46.30 dB | 0.00 dB | 110.47 dB |
| carrier-0068 | 1536 | 38.86 dB | 38.86 dB | 0.00 dB | 111.24 dB |
| carrier-0069 | 1536 | 54.46 dB | 54.46 dB | 0.00 dB | 109.48 dB |
| carrier-0070 | 1536 | 47.32 dB | 47.32 dB | 0.00 dB | 108.66 dB |
| carrier-0071 | 1536 | 43.17 dB | 43.17 dB | 0.00 dB | 108.77 dB |
| carrier-0072 | 1536 | 45.05 dB | 45.05 dB | 0.00 dB | 110.46 dB |
| carrier-0073 | 1536 | 49.41 dB | 49.41 dB | 0.00 dB | 107.72 dB |
| carrier-0074 | 1536 | 40.06 dB | 40.06 dB | 0.00 dB | 110.19 dB |
| carrier-0075 | 1536 | 44.09 dB | 44.09 dB | 0.00 dB | 109.62 dB |
| carrier-0076 | 1536 | 39.54 dB | 39.54 dB | 0.00 dB | 110.76 dB |
| carrier-0077 | 1536 | 48.44 dB | 48.44 dB | 0.00 dB | 110.15 dB |
| carrier-0078 | 1536 | 40.84 dB | 40.84 dB | 0.00 dB | 111.07 dB |
| carrier-0079 | 1536 | 45.88 dB | 45.88 dB | 0.00 dB | 109.62 dB |
| carrier-0080 | 1536 | 46.94 dB | 46.94 dB | 0.00 dB | 109.76 dB |
| carrier-0081 | 1536 | 48.46 dB | 48.46 dB | 0.00 dB | 109.33 dB |
| carrier-0082 | 1536 | 43.09 dB | 43.09 dB | 0.00 dB | 108.63 dB |
| carrier-0083 | 1536 | 59.75 dB | 59.75 dB | 0.00 dB | 106.93 dB |
| carrier-0084 | 1536 | 41.01 dB | 41.01 dB | 0.00 dB | 110.38 dB |

## ogg/vorbis

Measured on 84 of 84 carriers.

### Does it look like a plain ffmpeg encode?

The ffmpeg column is ffmpeg at its own defaults. Ogg Vorbis keeps the source's quality, so it differs from ffmpeg's default q3 whenever the carrier maps to another level.

| Carrier | Source samples | Samples (ffmpeg / Mist) | Zero tail (ffmpeg / Mist) | Sample format (ffmpeg / Mist) | Nominal kbps (ffmpeg / Mist) | Differs in |
|---|---|---|---|---|---|---|
| carrier-0001 | 13230191 | 13230191 / 13230191 | 0 / 0 | fltp / fltp | 112 / 256 | nominal bitrate |
| carrier-0002 | 13230191 | 13230191 / 13230191 | 0 / 0 | fltp / fltp | 112 / 256 | nominal bitrate |
| carrier-0003 | 13230191 | 13230191 / 13230191 | 0 / 0 | fltp / fltp | 112 / 256 | nominal bitrate |
| carrier-0004 | 11669231 | 11669231 / 11669231 | 0 / 0 | fltp / fltp | 112 / 256 | nominal bitrate |
| carrier-0005 | 13230191 | 13230191 / 13230191 | 0 / 0 | fltp / fltp | 112 / 256 | nominal bitrate |
| carrier-0006 | 13230191 | 13230191 / 13230191 | 0 / 0 | fltp / fltp | 112 / 256 | nominal bitrate |
| carrier-0007 | 13230191 | 13230191 / 13230191 | 0 / 0 | fltp / fltp | 112 / 256 | nominal bitrate |
| carrier-0008 | 13230191 | 13230191 / 13230191 | 0 / 0 | fltp / fltp | 112 / 256 | nominal bitrate |
| carrier-0009 | 11631215 | 11631215 / 11631215 | 0 / 0 | fltp / fltp | 112 / 256 | nominal bitrate |
| carrier-0010 | 12556271 | 12556271 / 12556271 | 30511 / 30255 | fltp / fltp | 112 / 256 | zero tail, nominal bitrate |
| carrier-0011 | 13230767 | 13230767 / 13230767 | 0 / 0 | fltp / fltp | 112 / 256 | nominal bitrate |
| carrier-0012 | 13230767 | 13230767 / 13230767 | 0 / 0 | fltp / fltp | 112 / 256 | nominal bitrate |
| carrier-0013 | 13230767 | 13230767 / 13230767 | 0 / 0 | fltp / fltp | 112 / 256 | nominal bitrate |
| carrier-0014 | 13230767 | 13230767 / 13230767 | 0 / 0 | fltp / fltp | 112 / 256 | nominal bitrate |
| carrier-0015 | 2880000 | 2880000 / 2880000 | 0 / 0 | fltp / fltp | 112 / 256 | nominal bitrate |
| carrier-0016 | 5760000 | 5760000 / 5760000 | 0 / 0 | fltp / fltp | 112 / 256 | nominal bitrate |
| carrier-0017 | 5760000 | 5760000 / 5760000 | 0 / 0 | fltp / fltp | 112 / 256 | nominal bitrate |
| carrier-0018 | 28803072 | 28803072 / 28803072 | 0 / 0 | fltp / fltp | 4294967 / 4294967 | — |
| carrier-0019 | 28803072 | 28803072 / 28803072 | 0 / 0 | fltp / fltp | 4294967 / 4294967 | — |
| carrier-0020 | 27650560 | 27650560 / 27650560 | 0 / 0 | fltp / fltp | 4294967 / 4294967 | — |
| carrier-0021 | 27998720 | 27998720 / 27998720 | 0 / 0 | fltp / fltp | 4294967 / 4294967 | — |
| carrier-0022 | 24209920 | 24209920 / 24209920 | 0 / 0 | fltp / fltp | 4294967 / 4294967 | — |
| carrier-0023 | 28803072 | 28803072 / 28803072 | 0 / 0 | fltp / fltp | 4294967 / 4294967 | — |
| carrier-0024 | 24574720 | 24574720 / 24574720 | 0 / 0 | fltp / fltp | 4294967 / 4294967 | — |
| carrier-0025 | 26830080 | 26830080 / 26830080 | 0 / 0 | fltp / fltp | 4294967 / 4294967 | — |
| carrier-0026 | 24695040 | 24695040 / 24695040 | 0 / 0 | fltp / fltp | 4294967 / 4294967 | — |
| carrier-0027 | 18909440 | 18909440 / 18909440 | 0 / 0 | fltp / fltp | 4294967 / 4294967 | — |
| carrier-0028 | 28803072 | 28803072 / 28803072 | 0 / 0 | fltp / fltp | 4294967 / 4294967 | — |
| carrier-0029 | 7939176 | 7939176 / 7939176 | 0 / 0 | fltp / fltp | 112 / 256 | nominal bitrate |
| carrier-0030 | 8334312 | 8334312 / 8334312 | 0 / 0 | fltp / fltp | 112 / 256 | nominal bitrate |
| carrier-0031 | 8187312 | 8187312 / 8187312 | 0 / 0 | fltp / fltp | 112 / 256 | nominal bitrate |
| carrier-0032 | 7616952 | 7616952 / 7616952 | 0 / 0 | fltp / fltp | 112 / 256 | nominal bitrate |
| carrier-0033 | 7703976 | 7703976 / 7703976 | 0 / 0 | fltp / fltp | 112 / 256 | nominal bitrate |
| carrier-0034 | 7795116 | 7795116 / 7795116 | 0 / 0 | fltp / fltp | 112 / 256 | nominal bitrate |
| carrier-0035 | 8786484 | 8786484 / 8786484 | 0 / 0 | fltp / fltp | 112 / 256 | nominal bitrate |
| carrier-0036 | 8306676 | 8306676 / 8306676 | 0 / 0 | fltp / fltp | 112 / 256 | nominal bitrate |
| carrier-0037 | 7848624 | 7848624 / 7848624 | 0 / 0 | fltp / fltp | 112 / 256 | nominal bitrate |
| carrier-0038 | 7849212 | 7849212 / 7849212 | 0 / 0 | fltp / fltp | 112 / 256 | nominal bitrate |
| carrier-0039 | 8371356 | 8371356 / 8371356 | 0 / 0 | fltp / fltp | 112 / 256 | nominal bitrate |
| carrier-0040 | 7808052 | 7808052 / 7808052 | 0 / 0 | fltp / fltp | 112 / 256 | nominal bitrate |
| carrier-0041 | 7792764 | 7792764 / 7792764 | 0 / 0 | fltp / fltp | 112 / 256 | nominal bitrate |
| carrier-0042 | 7705152 | 7705152 / 7705152 | 0 / 0 | fltp / fltp | 112 / 256 | nominal bitrate |
| carrier-0043 | 7733376 | 7733376 / 7733376 | 0 / 0 | fltp / fltp | 112 / 256 | nominal bitrate |
| carrier-0044 | 8332548 | 8332548 / 8332548 | 0 / 0 | fltp / fltp | 112 / 256 | nominal bitrate |
| carrier-0045 | 7836864 | 7836864 / 7836864 | 0 / 0 | fltp / fltp | 112 / 256 | nominal bitrate |
| carrier-0046 | 7783356 | 7783356 / 7783356 | 0 / 0 | fltp / fltp | 112 / 256 | nominal bitrate |
| carrier-0047 | 8134980 | 8134980 / 8134980 | 0 / 0 | fltp / fltp | 112 / 256 | nominal bitrate |
| carrier-0048 | 7904484 | 7904484 / 7904484 | 0 / 0 | fltp / fltp | 112 / 256 | nominal bitrate |
| carrier-0049 | 7577556 | 7577556 / 7577556 | 0 / 0 | fltp / fltp | 112 / 256 | nominal bitrate |
| carrier-0050 | 8433684 | 8433684 / 8433684 | 0 / 0 | fltp / fltp | 112 / 256 | nominal bitrate |
| carrier-0051 | 8167320 | 8167320 / 8167320 | 0 / 0 | fltp / fltp | 112 / 256 | nominal bitrate |
| carrier-0052 | 7443492 | 7443492 / 7443492 | 0 / 0 | fltp / fltp | 112 / 256 | nominal bitrate |
| carrier-0053 | 7847448 | 7847448 / 7847448 | 0 / 0 | fltp / fltp | 112 / 256 | nominal bitrate |
| carrier-0054 | 6560316 | 6560316 / 6560316 | 0 / 0 | fltp / fltp | 112 / 256 | nominal bitrate |
| carrier-0055 | 7567536 | 7567536 / 7567536 | 2160 / 368 | fltp / fltp | 112 / 256 | zero tail, nominal bitrate |
| carrier-0056 | 10716829 | 10716829 / 10716829 | 1373 / 1117 | fltp / fltp | 112 / 256 | zero tail, nominal bitrate |
| carrier-0057 | 10753254 | 10753254 / 10753254 | 0 / 0 | fltp / fltp | 112 / 256 | nominal bitrate |
| carrier-0058 | 8507078 | 8507078 / 8507078 | 0 / 0 | fltp / fltp | 112 / 256 | nominal bitrate |
| carrier-0059 | 9099131 | 9099131 / 9099131 | 0 / 0 | fltp / fltp | 112 / 256 | nominal bitrate |
| carrier-0060 | 10184328 | 10184328 / 10184328 | 0 / 0 | fltp / fltp | 112 / 256 | nominal bitrate |
| carrier-0061 | 9848955 | 9848955 / 9848955 | 0 / 0 | fltp / fltp | 112 / 256 | nominal bitrate |
| carrier-0062 | 10444219 | 10444219 / 10444219 | 507 / 635 | fltp / fltp | 112 / 256 | zero tail, nominal bitrate |
| carrier-0063 | 10854001 | 10854001 / 10854001 | 0 / 0 | fltp / fltp | 112 / 256 | nominal bitrate |
| carrier-0064 | 9693408 | 9693408 / 9693408 | 0 / 0 | fltp / fltp | 112 / 256 | nominal bitrate |
| carrier-0065 | 9766958 | 9766958 / 9766958 | 0 / 0 | fltp / fltp | 112 / 256 | nominal bitrate |
| carrier-0066 | 9359938 | 9359938 / 9359938 | 0 / 0 | fltp / fltp | 112 / 256 | nominal bitrate |
| carrier-0067 | 9855000 | 9855000 / 9855000 | 7896 / 7384 | fltp / fltp | 112 / 256 | zero tail, nominal bitrate |
| carrier-0068 | 9146026 | 9146026 / 9146026 | 0 / 0 | fltp / fltp | 112 / 256 | nominal bitrate |
| carrier-0069 | 10717286 | 10717286 / 10717286 | 0 / 0 | fltp / fltp | 112 / 256 | nominal bitrate |
| carrier-0070 | 8370000 | 8370000 / 8370000 | 0 / 0 | fltp / fltp | 112 / 256 | nominal bitrate |
| carrier-0071 | 10456119 | 10456119 / 10456119 | 23415 / 22903 | fltp / fltp | 112 / 256 | zero tail, nominal bitrate |
| carrier-0072 | 9573914 | 9573914 / 9573914 | 7130 / 6874 | fltp / fltp | 112 / 256 | zero tail, nominal bitrate |
| carrier-0073 | 10171848 | 10171848 / 10171848 | 8 / 0 | fltp / fltp | 112 / 256 | zero tail, nominal bitrate |
| carrier-0074 | 9735652 | 9735652 / 9735652 | 0 / 0 | fltp / fltp | 112 / 256 | nominal bitrate |
| carrier-0075 | 9412488 | 9412488 / 9412488 | 6088 / 6088 | fltp / fltp | 112 / 256 | nominal bitrate |
| carrier-0076 | 9888000 | 9888000 / 9888000 | 0 / 0 | fltp / fltp | 112 / 256 | nominal bitrate |
| carrier-0077 | 10440000 | 10440000 / 10440000 | 0 / 0 | fltp / fltp | 112 / 256 | nominal bitrate |
| carrier-0078 | 11520002 | 11520002 / 11520002 | 0 / 0 | fltp / fltp | 112 / 256 | nominal bitrate |
| carrier-0079 | 10450747 | 10450747 / 10450747 | 20731 / 20731 | fltp / fltp | 112 / 256 | nominal bitrate |
| carrier-0080 | 8598259 | 8598259 / 8598259 | 0 / 0 | fltp / fltp | 112 / 256 | nominal bitrate |
| carrier-0081 | 11520000 | 11520000 / 11520000 | 0 / 0 | fltp / fltp | 112 / 256 | nominal bitrate |
| carrier-0082 | 11102609 | 11102609 / 11102609 | 0 / 0 | fltp / fltp | 112 / 256 | nominal bitrate |
| carrier-0083 | 8540309 | 8540309 / 8540309 | 5973 / 5589 | fltp / fltp | 112 / 256 | zero tail, nominal bitrate |
| carrier-0084 | 10279847 | 10279847 / 10279847 | 5735 / 6503 | fltp / fltp | 112 / 256 | zero tail, nominal bitrate |

**Verdict:** ❌ differs on 73 of 84 carriers: nominal bitrate, zero tail.

### Can a detector tell?

Stego copy against the clean ffmpeg copy (for Ogg Vorbis, ffmpeg at the quality level Mist chose, so only the embedding differs).

| Detector | Looks for | Chunk AUC (95%) | File AUC (95%) | D | Detector-implied benchmark KL lower bound (nats, 95%) | Adjusted p | Verdict | Message size |
|---|---|---|---|---|---|---|---|---|
| chi-square | bit overwriting (not what Mist does) | 0.500 (0.500–0.500) | 0.500 (0.500–0.500) | 0.500 | 0.000 (0.000–0.000) | BH 1.000 | ✅ chance | ✅ hidden (0.500) |
| spa | bit overwriting (not what Mist does) | 0.500 (0.500–0.500) | 0.500 (0.500–0.501) | 0.500 | 0.000 (0.000–0.000) | BH 1.000 | ✅ chance | ✅ hidden (0.500) |
| rs | bit overwriting (not what Mist does) | 0.500 (0.500–0.501) | 0.495 (0.484–0.501) | 0.505 | 0.000 (0.000–0.001) | BH 1.000 | ✅ chance | ✅ hidden (0.500) |
| hcf-com | ±1 changes, which is what Mist does | 0.501 (0.501–0.501) | 0.508 (0.506–0.511) | 0.508 | 0.000 (0.000–0.000) | Holm 0.020 | ⚠️ faint signal | ✅ hidden (0.500) |
| classifier | anything it can learn from Mist's own output | 0.499 (0.498–0.499) | 0.492 (0.490–0.493) | 0.508 | 0.000 (0.000–0.000) | Holm 0.300 | ⚠️ faint signal | ✅ hidden (0.500) |
| markov | how the waveform's curvature changes from sample to sample | 0.497 (0.497–0.498) | 0.488 (0.483–0.492) | 0.512 | 0.000 (0.000–0.001) | Holm 0.300 | ⚠️ faint signal | ✅ hidden (0.500) |
| key-aware | the ephemeral key in the first frame's envelope, read with the public key alone | 0.506 (0.482–0.530) | 0.506 (0.482–0.530) | 0.506 | 0.000 (0.000–0.002) | Holm 1.000 | ✅ chance | ✅ hidden (0.500) |

Power: shifting this run's classifier file scores to D = 0.55, about 8 independent lineages give 90% power for a lineage-cluster interval to exclude 0.5. The figure is a simulation from this run's dispersion.

Worst cell: markov on aggregate, file AUC 0.488, D 0.512.

**Verdict:** ⚠️ faint signal from hcf-com (AUC 0.501).

### The embedding alone

Stego copy against Mist's own re-encode, so only the embedded changes differ.

| Detector | Looks for | Chunk AUC (95%) | File AUC (95%) | D | Detector-implied benchmark KL lower bound (nats, 95%) | Adjusted p | Verdict | Message size |
|---|---|---|---|---|---|---|---|---|
| chi-square | bit overwriting (not what Mist does) | 0.500 (0.500–0.500) | 0.500 (0.500–0.500) | 0.500 | 0.000 (0.000–0.000) | — | ✅ chance | ✅ hidden (0.500) |
| spa | bit overwriting (not what Mist does) | 0.500 (0.500–0.500) | 0.500 (0.500–0.501) | 0.500 | 0.000 (0.000–0.000) | — | ✅ chance | ✅ hidden (0.500) |
| rs | bit overwriting (not what Mist does) | 0.500 (0.500–0.501) | 0.495 (0.484–0.501) | 0.505 | 0.000 (0.000–0.001) | — | ✅ chance | ✅ hidden (0.500) |
| hcf-com | ±1 changes, which is what Mist does | 0.501 (0.501–0.501) | 0.508 (0.506–0.511) | 0.508 | 0.000 (0.000–0.000) | — | ⚠️ faint signal | ✅ hidden (0.500) |
| classifier | anything it can learn from Mist's own output | 0.499 (0.498–0.499) | 0.492 (0.490–0.493) | 0.508 | 0.000 (0.000–0.000) | — | ⚠️ faint signal | ✅ hidden (0.500) |
| markov | how the waveform's curvature changes from sample to sample | 0.497 (0.497–0.498) | 0.488 (0.483–0.492) | 0.512 | 0.000 (0.000–0.001) | — | ⚠️ faint signal | ✅ hidden (0.500) |

### Does detection grow with audio?

File AUC (95%) against the clean ffmpeg copy on the first chunks of each carrier. A chunk is 65,536 values, about 0.74 s of 44.1 kHz stereo.

| Detector | First 40 chunks | Whole file |
|---|---|---|
| chi-square | 0.500 (0.500–0.500) | 0.500 (0.500–0.500) |
| spa | 0.500 (0.500–0.501) | 0.500 (0.500–0.501) |
| rs | 0.500 (0.485–0.515) | 0.495 (0.484–0.501) |
| hcf-com | 0.507 (0.506–0.510) | 0.508 (0.506–0.511) |
| classifier | 0.492 (0.489–0.494) | 0.492 (0.490–0.493) |
| markov | 0.489 (0.485–0.492) | 0.488 (0.483–0.492) |

### Does detection grow with the number of files?

AUC when the warden averages a detector's score over k files drawn at random from the corpus, stego against clean. Draws overlap, so there is no interval.

| Detector | 1 file(s) | 3 file(s) | 6 file(s) |
|---|---|---|---|
| chi-square | 0.500 | 0.500 | 0.500 |
| spa | 0.482 | 0.509 | 0.517 |
| rs | 0.463 | 0.497 | 0.512 |
| hcf-com | 0.532 | 0.489 | 0.505 |
| classifier | 0.520 | 0.508 | 0.495 |
| markov | 0.515 | 0.508 | 0.487 |

### How much does it cost in sound?

| Measure | Mean | Worst | What it means |
|---|---|---|---|
| Plain re-encode SDR | 27.65 dB | 23.99 dB | Loss of Mist's own re-encode, with nothing embedded |
| Mist output SDR | 27.64 dB | 23.98 dB | The same, with the message embedded |
| Embedding cost | 0.01 dB | 0.02 dB | Mist's own share; target ≤ 0.3 dB, design-doc reference 0.18 dB |
| Extra error energy | +0% | +0% | Embedding cost as error added on top of a plain re-encode |
| Mist's error below the music | 55.06 dB | 48.34 dB | Stego copy against Mist's own re-encode: what embedding alone adds |

**Verdict:** ✅ Mist's own error sits 55.06 dB below the music; embedding costs 0.01 dB, within the 0.3 dB target.

### Per carrier

| Carrier | Output kbps | Plain re-encode SDR | Mist output SDR | Embedding cost | Mist's error below the music |
|---|---|---|---|---|---|
| carrier-0001 | 229 | 28.18 dB | 28.18 dB | 0.01 dB | 57.23 dB |
| carrier-0002 | 228 | 28.56 dB | 28.55 dB | 0.01 dB | 57.76 dB |
| carrier-0003 | 221 | 29.59 dB | 29.59 dB | 0.00 dB | 60.12 dB |
| carrier-0004 | 225 | 28.35 dB | 28.35 dB | 0.00 dB | 60.51 dB |
| carrier-0005 | 223 | 28.81 dB | 28.81 dB | 0.00 dB | 60.00 dB |
| carrier-0006 | 226 | 29.55 dB | 29.55 dB | 0.00 dB | 58.87 dB |
| carrier-0007 | 227 | 29.02 dB | 29.02 dB | 0.00 dB | 58.65 dB |
| carrier-0008 | 221 | 29.32 dB | 29.31 dB | 0.00 dB | 60.33 dB |
| carrier-0009 | 189 | 27.89 dB | 27.89 dB | 0.01 dB | 55.52 dB |
| carrier-0010 | 192 | 29.22 dB | 29.21 dB | 0.01 dB | 57.07 dB |
| carrier-0011 | 268 | 32.74 dB | 32.73 dB | 0.01 dB | 59.39 dB |
| carrier-0012 | 269 | 25.99 dB | 25.98 dB | 0.01 dB | 53.83 dB |
| carrier-0013 | 258 | 32.92 dB | 32.91 dB | 0.01 dB | 61.15 dB |
| carrier-0014 | 243 | 30.99 dB | 30.99 dB | 0.00 dB | 64.18 dB |
| carrier-0015 | 262 | 26.02 dB | 26.01 dB | 0.01 dB | 52.18 dB |
| carrier-0016 | 282 | 26.09 dB | 26.08 dB | 0.01 dB | 52.94 dB |
| carrier-0017 | 261 | 27.80 dB | 27.79 dB | 0.01 dB | 54.25 dB |
| carrier-0018 | 332 | 30.98 dB | 30.98 dB | 0.00 dB | 72.29 dB |
| carrier-0019 | 295 | 30.48 dB | 30.48 dB | 0.00 dB | 74.24 dB |
| carrier-0020 | 244 | 29.12 dB | 29.12 dB | 0.00 dB | 59.86 dB |
| carrier-0021 | 218 | 28.35 dB | 28.35 dB | 0.01 dB | 57.64 dB |
| carrier-0022 | 257 | 29.87 dB | 29.86 dB | 0.01 dB | 58.06 dB |
| carrier-0023 | 220 | 29.76 dB | 29.76 dB | 0.00 dB | 61.03 dB |
| carrier-0024 | 271 | 28.28 dB | 28.28 dB | 0.01 dB | 57.55 dB |
| carrier-0025 | 226 | 27.33 dB | 27.33 dB | 0.01 dB | 55.61 dB |
| carrier-0026 | 273 | 27.07 dB | 27.06 dB | 0.01 dB | 55.34 dB |
| carrier-0027 | 224 | 26.76 dB | 26.76 dB | 0.01 dB | 56.03 dB |
| carrier-0028 | 237 | 28.52 dB | 28.52 dB | 0.00 dB | 58.79 dB |
| carrier-0029 | 233 | 28.56 dB | 28.55 dB | 0.01 dB | 55.18 dB |
| carrier-0030 | 237 | 29.33 dB | 29.32 dB | 0.01 dB | 56.65 dB |
| carrier-0031 | 239 | 29.55 dB | 29.54 dB | 0.01 dB | 55.05 dB |
| carrier-0032 | 244 | 29.52 dB | 29.51 dB | 0.01 dB | 57.14 dB |
| carrier-0033 | 230 | 29.20 dB | 29.19 dB | 0.01 dB | 55.45 dB |
| carrier-0034 | 245 | 29.59 dB | 29.58 dB | 0.01 dB | 56.80 dB |
| carrier-0035 | 239 | 29.11 dB | 29.10 dB | 0.01 dB | 55.77 dB |
| carrier-0036 | 231 | 28.73 dB | 28.72 dB | 0.01 dB | 55.82 dB |
| carrier-0037 | 245 | 28.61 dB | 28.60 dB | 0.01 dB | 54.95 dB |
| carrier-0038 | 249 | 29.27 dB | 29.27 dB | 0.01 dB | 57.71 dB |
| carrier-0039 | 248 | 29.16 dB | 29.15 dB | 0.01 dB | 56.63 dB |
| carrier-0040 | 240 | 28.96 dB | 28.95 dB | 0.01 dB | 56.16 dB |
| carrier-0041 | 232 | 28.92 dB | 28.91 dB | 0.01 dB | 55.62 dB |
| carrier-0042 | 242 | 28.21 dB | 28.20 dB | 0.01 dB | 55.91 dB |
| carrier-0043 | 236 | 29.10 dB | 29.08 dB | 0.01 dB | 54.74 dB |
| carrier-0044 | 242 | 29.12 dB | 29.11 dB | 0.01 dB | 55.58 dB |
| carrier-0045 | 244 | 28.05 dB | 28.04 dB | 0.01 dB | 56.07 dB |
| carrier-0046 | 242 | 29.18 dB | 29.17 dB | 0.01 dB | 56.81 dB |
| carrier-0047 | 248 | 29.15 dB | 29.14 dB | 0.01 dB | 56.84 dB |
| carrier-0048 | 233 | 29.35 dB | 29.34 dB | 0.01 dB | 56.40 dB |
| carrier-0049 | 234 | 29.24 dB | 29.23 dB | 0.01 dB | 56.59 dB |
| carrier-0050 | 240 | 27.36 dB | 27.34 dB | 0.01 dB | 52.65 dB |
| carrier-0051 | 242 | 28.76 dB | 28.75 dB | 0.01 dB | 55.64 dB |
| carrier-0052 | 231 | 28.44 dB | 28.43 dB | 0.01 dB | 57.03 dB |
| carrier-0053 | 231 | 28.97 dB | 28.97 dB | 0.01 dB | 56.41 dB |
| carrier-0054 | 233 | 27.31 dB | 27.30 dB | 0.01 dB | 53.81 dB |
| carrier-0055 | 263 | 26.64 dB | 26.63 dB | 0.01 dB | 52.40 dB |
| carrier-0056 | 249 | 25.93 dB | 25.92 dB | 0.01 dB | 51.36 dB |
| carrier-0057 | 264 | 25.30 dB | 25.29 dB | 0.02 dB | 49.39 dB |
| carrier-0058 | 255 | 25.05 dB | 25.04 dB | 0.01 dB | 49.74 dB |
| carrier-0059 | 250 | 24.35 dB | 24.34 dB | 0.01 dB | 49.47 dB |
| carrier-0060 | 243 | 23.99 dB | 23.98 dB | 0.02 dB | 48.34 dB |
| carrier-0061 | 248 | 25.28 dB | 25.27 dB | 0.02 dB | 49.71 dB |
| carrier-0062 | 245 | 26.22 dB | 26.21 dB | 0.01 dB | 52.50 dB |
| carrier-0063 | 259 | 26.16 dB | 26.15 dB | 0.01 dB | 52.40 dB |
| carrier-0064 | 232 | 26.07 dB | 26.06 dB | 0.01 dB | 52.55 dB |
| carrier-0065 | 244 | 25.82 dB | 25.80 dB | 0.01 dB | 50.99 dB |
| carrier-0066 | 251 | 25.61 dB | 25.60 dB | 0.01 dB | 51.05 dB |
| carrier-0067 | 256 | 24.98 dB | 24.97 dB | 0.01 dB | 50.29 dB |
| carrier-0068 | 266 | 25.83 dB | 25.81 dB | 0.01 dB | 50.61 dB |
| carrier-0069 | 233 | 25.43 dB | 25.42 dB | 0.01 dB | 50.92 dB |
| carrier-0070 | 244 | 27.25 dB | 27.24 dB | 0.01 dB | 53.34 dB |
| carrier-0071 | 246 | 24.64 dB | 24.63 dB | 0.01 dB | 49.25 dB |
| carrier-0072 | 248 | 25.06 dB | 25.05 dB | 0.02 dB | 49.39 dB |
| carrier-0073 | 257 | 25.68 dB | 25.67 dB | 0.01 dB | 50.70 dB |
| carrier-0074 | 244 | 25.43 dB | 25.42 dB | 0.01 dB | 51.26 dB |
| carrier-0075 | 259 | 26.01 dB | 26.00 dB | 0.01 dB | 51.04 dB |
| carrier-0076 | 271 | 24.70 dB | 24.69 dB | 0.02 dB | 48.95 dB |
| carrier-0077 | 255 | 25.17 dB | 25.16 dB | 0.01 dB | 50.01 dB |
| carrier-0078 | 256 | 26.06 dB | 26.05 dB | 0.01 dB | 51.73 dB |
| carrier-0079 | 245 | 24.20 dB | 24.19 dB | 0.01 dB | 49.49 dB |
| carrier-0080 | 260 | 25.76 dB | 25.75 dB | 0.02 dB | 50.29 dB |
| carrier-0081 | 244 | 24.64 dB | 24.63 dB | 0.01 dB | 49.97 dB |
| carrier-0082 | 245 | 25.19 dB | 25.18 dB | 0.01 dB | 50.86 dB |
| carrier-0083 | 255 | 24.11 dB | 24.10 dB | 0.01 dB | 49.49 dB |
| carrier-0084 | 261 | 27.70 dB | 27.69 dB | 0.01 dB | 53.61 dB |

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
is exactly what Mist does, so it sits in the confirmatory family with the classifier, the Markov model and the
key-aware warden. That family of four is adjusted with Holm. A dash means that row was not part of the
confirmatory test (scaling, category and embedding-only tables).

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
than in the values or their steps. These are the rich-model features of audio steganalysis; a detector that
learns its own features, a CNN trained on exported chunks, is the next step beyond them and is not run here.
When it is run, its folds follow lineage as well.

**Power** asks how many independent lineages this run's classifier dispersion would need before a shift to
D = 0.55 pushed the lineage interval off 0.5 in 90% of simulations. It is not a guarantee about a future corpus.
**Worst cell** is the highest D among category rows, or among the aggregate detectors when the corpus has one
category. **Leave-one-lineage** restricts the classifier's existing file scores to each lineage that has two or
more carriers, and to the carriers that remain. It does not retrain.

**Message size** compares a 64-byte message with a 1-byte one. Mist changes the same amount of audio whatever
the message, so this should read 0.5: anything else means the message length shows.

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
| Plain properties | every carrier matches ffmpeg | — | any carrier differs |
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

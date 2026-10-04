# Mist harness report

perceptual metric: not installed

Every carrier is encoded three times: by the ffmpeg command line at its own defaults (the *clean* copy, which is what a warden without the original would compare against), by Mist's own encoder with nothing embedded (Mist's *own* re-encode), and by Mist with a hidden message (the *stego* copy). The report asks whether the stego copy differs from the clean one in plain properties, whether a detector can tell them apart, and how much worse it sounds. [How to read this report](#how-to-read-this-report) explains every number and threshold.

## Summary

| Format | Carriers | Looks like ffmpeg? | Hidden from detectors? | Audio quality |
|---|---|---|---|---|
| ogg/vorbis | 13 / 13 | ❌ differs on 13 of 13 carriers: nominal bitrate, length, zero tail | ❌ rs detects it (AUC 0.519) | ✅ Mist's own error sits 65.08 dB below the music; embedding costs 0.01 dB, within the 0.3 dB target |
| flac | 13 / 13 | ❌ differs on 10 of 13 carriers: sample format, zero tail | ❌ chi-square detects it (AUC 0.966) | ✅ inaudible: the added error is at least 85.48 dB below the music |
| wav/pcm_s16le | 13 / 13 | ❌ differs on 5 of 13 carriers: zero tail | ⚠️ faint signal from hcf-com (AUC 0.500) | ✅ inaudible: the added error is at least 85.48 dB below the music |

## ogg/vorbis

Measured on 13 of 13 carriers.

### Does it look like a plain ffmpeg encode?

| Carrier | Source samples | Samples (ffmpeg / Mist) | Zero tail (ffmpeg / Mist) | Sample format (ffmpeg / Mist) | Nominal kbps (ffmpeg / Mist) | Differs in |
|---|---|---|---|---|---|---|
| 02 - Ride the Lightning.mp3 | 18406656 | 18407616 / 18407616 | 0 / 0 | fltp / fltp | 112 / 480 | nominal bitrate |
| 03 - Disposable Heroes.mp3 | 24553728 | 24553792 / 24553792 | 0 / 0 | fltp / fltp | 112 / 480 | nominal bitrate |
| 04 - No Remorse.mp3 | 17521920 | 17522880 / 17522880 | 0 / 0 | fltp / fltp | 112 / 480 | nominal bitrate |
| 06 - For Whom the Bell Tolls.mp3 | 11669760 | 11670592 / 11670592 | 0 / 0 | fltp / fltp | 112 / 480 | nominal bitrate |
| 07 - The Four Horsemen.mp3 | 14614272 | 14614720 / 14614720 | 0 / 0 | fltp / fltp | 112 / 480 | nominal bitrate |
| 08 - Fade to Black.mp3 | 19317888 | 19318208 / 19318336 | 0 / 0 | fltp / fltp | 112 / 480 | length, nominal bitrate |
| 09 - Seek & Destroy.mp3 | 18188928 | 18189376 / 18189376 | 0 / 0 | fltp / fltp | 112 / 480 | nominal bitrate |
| 10 - Whiplash.mp3 | 13644288 | 13644864 / 13644864 | 0 / 0 | fltp / fltp | 112 / 480 | nominal bitrate |
| 11 - Fight Fire with Fire.mp3 | 11631744 | 11632448 / 11631744 | 0 / 256 | fltp / fltp | 112 / 480 | length, zero tail, nominal bitrate |
| 14 - Motorbreath.mp3 | 12556800 | 12556992 / 12557248 | 30720 / 30720 | fltp / fltp | 112 / 480 | length, nominal bitrate |
| Pillars_of_the_Sky.wav | 2880000 | 2880192 / 2880960 | 0 / 0 | fltp / fltp | 112 / 500 | length, nominal bitrate |
| Starlight_Ascent.wav | 5760000 | 5760064 / 5760320 | 0 / 0 | fltp / fltp | 112 / 500 | length, nominal bitrate |
| Stellar_Ascent.wav | 5760000 | 5760960 / 5760192 | 0 / 0 | fltp / fltp | 112 / 500 | length, nominal bitrate |

**Verdict:** ❌ differs on 13 of 13 carriers: nominal bitrate, length, zero tail.

### Can a detector tell?

Stego copy against the clean ffmpeg copy.

| Detector | Looks for | Chunk AUC (95%) | File AUC (95%) | ε ≥ (nats, 95%) | Verdict | Message size | AUC change |
|---|---|---|---|---|---|---|---|
| chi-square | bit overwriting (not what Mist does) | 0.500 (0.500–0.501) | 0.538 (0.500–0.615) | 0.003 (0.000–0.027) | ✅ chance | ✅ hidden (0.500) | +0.000 |
| spa | bit overwriting (not what Mist does) | 0.523 (0.500–0.563) | 0.615 (0.500–0.731) | 0.027 (0.000–0.107) | ✅ chance | ✅ hidden (0.500) | +0.000 |
| rs | bit overwriting (not what Mist does) | 0.519 (0.500–0.550) | 0.654 (0.538–0.808) | 0.047 (0.003–0.189) | ❌ detectable per file | ✅ hidden (0.499) | -0.001 |
| hcf-com | ±1 changes, which is what Mist does | 1.000 (1.000–1.000) | 1.000 (1.000–1.000) | 0.500 (0.500–0.500) | ❌ detectable | ✅ hidden (0.500) | +0.000 |
| classifier | anything it can learn from Mist's own output | 1.000 (0.999–1.000) | 1.000 (1.000–1.000) | 0.500 (0.500–0.500) | ❌ detectable | ✅ hidden (0.500) | -0.000 |

**Verdict:** ❌ rs detects it (AUC 0.519).

### The embedding alone

Stego copy against Mist's own re-encode, so only the embedded changes differ.

| Detector | Looks for | Chunk AUC (95%) | File AUC (95%) | ε ≥ (nats, 95%) | Verdict | Message size |
|---|---|---|---|---|---|---|
| chi-square | bit overwriting (not what Mist does) | 0.500 (0.500–0.500) | 0.497 (0.473–0.500) | 0.000 (0.000–0.001) | ✅ chance | ✅ hidden (0.500) |
| spa | bit overwriting (not what Mist does) | 0.499 (0.498–0.500) | 0.491 (0.441–0.500) | 0.000 (0.000–0.007) | ✅ chance | ✅ hidden (0.500) |
| rs | bit overwriting (not what Mist does) | 0.499 (0.497–0.500) | 0.488 (0.426–0.497) | 0.000 (0.000–0.011) | ⚠️ faint signal per file | ✅ hidden (0.499) |
| hcf-com | ±1 changes, which is what Mist does | 0.497 (0.495–0.498) | 0.462 (0.391–0.450) | 0.003 (0.005–0.024) | ⚠️ faint signal | ✅ hidden (0.500) |
| classifier | anything it can learn from Mist's own output | 0.501 (0.500–0.502) | 0.527 (0.473–0.609) | 0.001 (0.000–0.024) | ⚠️ faint signal | ✅ hidden (0.500) |

### How much does it cost in sound?

| Measure | Mean | Worst | What it means |
|---|---|---|---|
| Plain re-encode SDR | 35.97 dB | 35.07 dB | Loss of Mist's own re-encode, with nothing embedded |
| Mist output SDR | 35.97 dB | 35.06 dB | The same, with the message embedded |
| Embedding cost | 0.01 dB | 0.01 dB | Mist's own share; target ≤ 0.3 dB, design-doc reference 0.18 dB |
| Extra error energy | +0% | +0% | Embedding cost as error added on top of a plain re-encode |
| Mist's error below the music | 65.08 dB | 61.24 dB | Stego copy against Mist's own re-encode: what embedding alone adds |
| Embedding cost, earlier run | 0.01 dB | 0.01 dB | Same measure at commit being compared with |

**Verdict:** ✅ Mist's own error sits 65.08 dB below the music; embedding costs 0.01 dB, within the 0.3 dB target.

### Per carrier

| Carrier | Output kbps | Plain re-encode SDR | Mist output SDR | Embedding cost | Mist's error below the music |
|---|---|---|---|---|---|
| 02 - Ride the Lightning.mp3 | 391 | 35.22 dB | 35.21 dB | 0.00 dB | 65.00 dB |
| 03 - Disposable Heroes.mp3 | 392 | 35.73 dB | 35.73 dB | 0.00 dB | 65.93 dB |
| 04 - No Remorse.mp3 | 381 | 35.87 dB | 35.87 dB | 0.00 dB | 67.84 dB |
| 06 - For Whom the Bell Tolls.mp3 | 385 | 35.07 dB | 35.06 dB | 0.00 dB | 67.77 dB |
| 07 - The Four Horsemen.mp3 | 383 | 35.56 dB | 35.55 dB | 0.00 dB | 67.55 dB |
| 08 - Fade to Black.mp3 | 386 | 36.20 dB | 36.20 dB | 0.00 dB | 66.49 dB |
| 09 - Seek & Destroy.mp3 | 390 | 35.94 dB | 35.93 dB | 0.00 dB | 66.17 dB |
| 10 - Whiplash.mp3 | 379 | 36.20 dB | 36.20 dB | 0.00 dB | 67.78 dB |
| 11 - Fight Fire with Fire.mp3 | 334 | 36.30 dB | 36.29 dB | 0.01 dB | 62.49 dB |
| 14 - Motorbreath.mp3 | 335 | 36.45 dB | 36.45 dB | 0.01 dB | 63.86 dB |
| Pillars_of_the_Sky.wav | 484 | 36.25 dB | 36.24 dB | 0.01 dB | 61.25 dB |
| Starlight_Ascent.wav | 504 | 36.17 dB | 36.16 dB | 0.01 dB | 61.24 dB |
| Stellar_Ascent.wav | 473 | 36.70 dB | 36.69 dB | 0.01 dB | 62.67 dB |

## flac

Measured on 13 of 13 carriers.

### Does it look like a plain ffmpeg encode?

| Carrier | Source samples | Samples (ffmpeg / Mist) | Zero tail (ffmpeg / Mist) | Sample format (ffmpeg / Mist) | Nominal kbps (ffmpeg / Mist) | Differs in |
|---|---|---|---|---|---|---|
| 02 - Ride the Lightning.mp3 | 18406656 | 18406656 / 18406656 | 0 / 0 | s32 / s16 | 0 / 0 | sample format |
| 03 - Disposable Heroes.mp3 | 24553728 | 24553728 / 24553728 | 0 / 0 | s32 / s16 | 0 / 0 | sample format |
| 04 - No Remorse.mp3 | 17521920 | 17521920 / 17521920 | 0 / 365 | s32 / s16 | 0 / 0 | zero tail, sample format |
| 06 - For Whom the Bell Tolls.mp3 | 11669760 | 11669760 / 11669760 | 0 / 451 | s32 / s16 | 0 / 0 | zero tail, sample format |
| 07 - The Four Horsemen.mp3 | 14614272 | 14614272 / 14614272 | 0 / 0 | s32 / s16 | 0 / 0 | sample format |
| 08 - Fade to Black.mp3 | 19317888 | 19317888 / 19317888 | 0 / 0 | s32 / s16 | 0 / 0 | sample format |
| 09 - Seek & Destroy.mp3 | 18188928 | 18188928 / 18188928 | 0 / 0 | s32 / s16 | 0 / 0 | sample format |
| 10 - Whiplash.mp3 | 13644288 | 13644288 / 13644288 | 486 / 701 | s32 / s16 | 0 / 0 | zero tail, sample format |
| 11 - Fight Fire with Fire.mp3 | 11631744 | 11631744 / 11631744 | 288 / 503 | s32 / s16 | 0 / 0 | zero tail, sample format |
| 14 - Motorbreath.mp3 | 12556800 | 12556800 / 12556800 | 30761 / 31874 | s32 / s16 | 0 / 0 | zero tail, sample format |
| Pillars_of_the_Sky.wav | 2880000 | 2880000 / 2880000 | 0 / 0 | s16 / s16 | 0 / 0 | — |
| Starlight_Ascent.wav | 5760000 | 5760000 / 5760000 | 0 / 0 | s16 / s16 | 0 / 0 | — |
| Stellar_Ascent.wav | 5760000 | 5760000 / 5760000 | 0 / 0 | s16 / s16 | 0 / 0 | — |

**Verdict:** ❌ differs on 10 of 13 carriers: sample format, zero tail.

### Can a detector tell?

Stego copy against the clean ffmpeg copy.

| Detector | Looks for | Chunk AUC (95%) | File AUC (95%) | ε ≥ (nats, 95%) | Verdict | Message size | AUC change |
|---|---|---|---|---|---|---|---|
| chi-square | bit overwriting (not what Mist does) | 0.966 (0.906–0.999) | 0.970 (0.846–1.000) | 0.443 (0.240–0.500) | ❌ detectable | ✅ hidden (0.500) | +0.001 |
| spa | bit overwriting (not what Mist does) | 0.219 (0.195–0.263) | 0.112 (0.000–0.254) | 0.300 (0.121–0.500) | ❌ detectable | ✅ hidden (0.500) | -0.001 |
| rs | bit overwriting (not what Mist does) | 0.283 (0.263–0.316) | 0.041 (0.000–0.178) | 0.421 (0.208–0.500) | ❌ detectable | ✅ hidden (0.499) | -0.002 |
| hcf-com | ±1 changes, which is what Mist does | 0.964 (0.903–0.999) | 0.805 (0.645–1.000) | 0.186 (0.042–0.500) | ❌ detectable | ✅ hidden (0.500) | +0.000 |
| classifier | anything it can learn from Mist's own output | 0.961 (0.890–0.999) | 0.911 (0.740–1.000) | 0.338 (0.115–0.500) | ❌ detectable | ✅ hidden (0.500) | +0.000 |

**Verdict:** ❌ chi-square detects it (AUC 0.966).

### The embedding alone

Stego copy against Mist's own re-encode, so only the embedded changes differ.

| Detector | Looks for | Chunk AUC (95%) | File AUC (95%) | ε ≥ (nats, 95%) | Verdict | Message size |
|---|---|---|---|---|---|---|
| chi-square | bit overwriting (not what Mist does) | 0.500 (0.499–0.500) | 0.494 (0.435–0.538) | 0.000 (0.000–0.008) | ✅ chance | ✅ hidden (0.500) |
| spa | bit overwriting (not what Mist does) | 0.499 (0.497–0.501) | 0.515 (0.426–0.604) | 0.000 (0.000–0.021) | ✅ chance | ✅ hidden (0.500) |
| rs | bit overwriting (not what Mist does) | 0.499 (0.498–0.500) | 0.456 (0.367–0.527) | 0.004 (0.000–0.035) | ✅ chance | ✅ hidden (0.499) |
| hcf-com | ±1 changes, which is what Mist does | 0.500 (0.499–0.500) | 0.515 (0.462–0.586) | 0.000 (0.000–0.015) | ✅ chance | ✅ hidden (0.500) |
| classifier | anything it can learn from Mist's own output | 0.501 (0.500–0.501) | 0.533 (0.509–0.609) | 0.002 (0.000–0.024) | ⚠️ faint signal | ✅ hidden (0.500) |

### How much does it cost in sound?

| Measure | Mean | Worst | What it means |
|---|---|---|---|
| Plain re-encode SDR | ∞ | 85.62 dB | Loss of Mist's own re-encode, with nothing embedded |
| Mist output SDR | 89.88 dB | 85.48 dB | The same, with the message embedded |
| Embedding cost | ∞ | ∞ | Mist's own share; target ≤ 0.3 dB, design-doc reference 0.18 dB |
| Extra error energy | — | — | Embedding cost as error added on top of a plain re-encode |
| Mist's error below the music | 101.16 dB | 99.90 dB | Stego copy against Mist's own re-encode: what embedding alone adds |
| Embedding cost, earlier run | — | — | Same measure at commit being compared with |

**Verdict:** ✅ inaudible: the added error is at least 85.48 dB below the music.

### Per carrier

| Carrier | Output kbps | Plain re-encode SDR | Mist output SDR | Embedding cost | Mist's error below the music |
|---|---|---|---|---|---|
| 02 - Ride the Lightning.mp3 | 873 | 85.79 dB | 85.65 dB | 0.14 dB | 100.33 dB |
| 03 - Disposable Heroes.mp3 | 867 | 86.13 dB | 85.99 dB | 0.14 dB | 100.67 dB |
| 04 - No Remorse.mp3 | 846 | 87.10 dB | 86.96 dB | 0.14 dB | 101.61 dB |
| 06 - For Whom the Bell Tolls.mp3 | 840 | 85.62 dB | 85.48 dB | 0.14 dB | 100.16 dB |
| 07 - The Four Horsemen.mp3 | 845 | 86.10 dB | 85.96 dB | 0.14 dB | 100.63 dB |
| 08 - Fade to Black.mp3 | 868 | 87.14 dB | 87.00 dB | 0.14 dB | 101.65 dB |
| 09 - Seek & Destroy.mp3 | 891 | 87.07 dB | 86.93 dB | 0.14 dB | 101.57 dB |
| 10 - Whiplash.mp3 | 839 | 87.72 dB | 87.58 dB | 0.14 dB | 102.19 dB |
| 11 - Fight Fire with Fire.mp3 | 826 | 86.11 dB | 85.96 dB | 0.14 dB | 100.63 dB |
| 14 - Motorbreath.mp3 | 751 | 86.28 dB | 86.14 dB | 0.14 dB | 100.82 dB |
| Pillars_of_the_Sky.wav | 968 | ∞ | 99.90 dB | ∞ | 99.90 dB |
| Starlight_Ascent.wav | 1046 | ∞ | 102.10 dB | ∞ | 102.10 dB |
| Stellar_Ascent.wav | 978 | ∞ | 102.83 dB | ∞ | 102.83 dB |

## wav/pcm_s16le

Measured on 13 of 13 carriers.

### Does it look like a plain ffmpeg encode?

| Carrier | Source samples | Samples (ffmpeg / Mist) | Zero tail (ffmpeg / Mist) | Sample format (ffmpeg / Mist) | Nominal kbps (ffmpeg / Mist) | Differs in |
|---|---|---|---|---|---|---|
| 02 - Ride the Lightning.mp3 | 18406656 | 18406656 / 18406656 | 0 / 0 | s16 / s16 | 1411 / 1411 | — |
| 03 - Disposable Heroes.mp3 | 24553728 | 24553728 / 24553728 | 0 / 0 | s16 / s16 | 1411 / 1411 | — |
| 04 - No Remorse.mp3 | 17521920 | 17521920 / 17521920 | 1 / 365 | s16 / s16 | 1411 / 1411 | zero tail |
| 06 - For Whom the Bell Tolls.mp3 | 11669760 | 11669760 / 11669760 | 0 / 451 | s16 / s16 | 1411 / 1411 | zero tail |
| 07 - The Four Horsemen.mp3 | 14614272 | 14614272 / 14614272 | 0 / 0 | s16 / s16 | 1411 / 1411 | — |
| 08 - Fade to Black.mp3 | 19317888 | 19317888 / 19317888 | 0 / 0 | s16 / s16 | 1411 / 1411 | — |
| 09 - Seek & Destroy.mp3 | 18188928 | 18188928 / 18188928 | 0 / 0 | s16 / s16 | 1411 / 1411 | — |
| 10 - Whiplash.mp3 | 13644288 | 13644288 / 13644288 | 492 / 701 | s16 / s16 | 1411 / 1411 | zero tail |
| 11 - Fight Fire with Fire.mp3 | 11631744 | 11631744 / 11631744 | 300 / 503 | s16 / s16 | 1411 / 1411 | zero tail |
| 14 - Motorbreath.mp3 | 12556800 | 12556800 / 12556800 | 30781 / 31874 | s16 / s16 | 1411 / 1411 | zero tail |
| Pillars_of_the_Sky.wav | 2880000 | 2880000 / 2880000 | 0 / 0 | s16 / s16 | 1536 / 1536 | — |
| Starlight_Ascent.wav | 5760000 | 5760000 / 5760000 | 0 / 0 | s16 / s16 | 1536 / 1536 | — |
| Stellar_Ascent.wav | 5760000 | 5760000 / 5760000 | 0 / 0 | s16 / s16 | 1536 / 1536 | — |

**Verdict:** ❌ differs on 5 of 13 carriers: zero tail.

### Can a detector tell?

Stego copy against the clean ffmpeg copy.

| Detector | Looks for | Chunk AUC (95%) | File AUC (95%) | ε ≥ (nats, 95%) | Verdict | Message size | AUC change |
|---|---|---|---|---|---|---|---|
| chi-square | bit overwriting (not what Mist does) | 0.501 (0.499–0.502) | 0.494 (0.438–0.544) | 0.000 (0.000–0.008) | ✅ chance | ✅ hidden (0.501) | +0.001 |
| spa | bit overwriting (not what Mist does) | 0.500 (0.491–0.509) | 0.538 (0.385–0.710) | 0.003 (0.000–0.088) | ✅ chance | ✅ hidden (0.499) | -0.001 |
| rs | bit overwriting (not what Mist does) | 0.496 (0.490–0.502) | 0.426 (0.278–0.562) | 0.011 (0.000–0.098) | ✅ chance | ✅ hidden (0.500) | -0.003 |
| hcf-com | ±1 changes, which is what Mist does | 0.500 (0.496–0.504) | 0.426 (0.308–0.467) | 0.011 (0.002–0.074) | ⚠️ faint signal per file | ✅ hidden (0.500) | +0.001 |
| classifier | anything it can learn from Mist's own output | 0.507 (0.497–0.516) | 0.621 (0.485–0.781) | 0.029 (0.000–0.158) | ✅ chance | ✅ hidden (0.501) | +0.002 |

**Verdict:** ⚠️ faint signal from hcf-com (AUC 0.500).

### The embedding alone

Stego copy against Mist's own re-encode, so only the embedded changes differ.

| Detector | Looks for | Chunk AUC (95%) | File AUC (95%) | ε ≥ (nats, 95%) | Verdict | Message size |
|---|---|---|---|---|---|---|
| chi-square | bit overwriting (not what Mist does) | 0.501 (0.500–0.502) | 0.500 (0.444–0.553) | 0.000 (0.000–0.006) | ✅ chance | ✅ hidden (0.501) |
| spa | bit overwriting (not what Mist does) | 0.499 (0.496–0.501) | 0.497 (0.414–0.580) | 0.000 (0.000–0.015) | ✅ chance | ✅ hidden (0.499) |
| rs | bit overwriting (not what Mist does) | 0.499 (0.498–0.501) | 0.462 (0.367–0.533) | 0.003 (0.000–0.035) | ✅ chance | ✅ hidden (0.500) |
| hcf-com | ±1 changes, which is what Mist does | 0.499 (0.499–0.500) | 0.485 (0.408–0.538) | 0.000 (0.000–0.017) | ✅ chance | ✅ hidden (0.500) |
| classifier | anything it can learn from Mist's own output | 0.501 (0.500–0.501) | 0.521 (0.473–0.592) | 0.001 (0.000–0.017) | ⚠️ faint signal | ✅ hidden (0.501) |

### How much does it cost in sound?

| Measure | Mean | Worst | What it means |
|---|---|---|---|
| Plain re-encode SDR | ∞ | 85.62 dB | Loss of Mist's own re-encode, with nothing embedded |
| Mist output SDR | 89.88 dB | 85.48 dB | The same, with the message embedded |
| Embedding cost | ∞ | ∞ | Mist's own share; target ≤ 0.3 dB, design-doc reference 0.18 dB |
| Extra error energy | — | — | Embedding cost as error added on top of a plain re-encode |
| Mist's error below the music | 101.16 dB | 99.91 dB | Stego copy against Mist's own re-encode: what embedding alone adds |
| Embedding cost, earlier run | — | — | Same measure at commit being compared with |

**Verdict:** ✅ inaudible: the added error is at least 85.48 dB below the music.

### Per carrier

| Carrier | Output kbps | Plain re-encode SDR | Mist output SDR | Embedding cost | Mist's error below the music |
|---|---|---|---|---|---|
| 02 - Ride the Lightning.mp3 | 1411 | 85.79 dB | 85.65 dB | 0.14 dB | 100.33 dB |
| 03 - Disposable Heroes.mp3 | 1411 | 86.13 dB | 85.99 dB | 0.14 dB | 100.67 dB |
| 04 - No Remorse.mp3 | 1411 | 87.10 dB | 86.96 dB | 0.14 dB | 101.60 dB |
| 06 - For Whom the Bell Tolls.mp3 | 1411 | 85.62 dB | 85.48 dB | 0.14 dB | 100.16 dB |
| 07 - The Four Horsemen.mp3 | 1411 | 86.10 dB | 85.96 dB | 0.14 dB | 100.63 dB |
| 08 - Fade to Black.mp3 | 1411 | 87.14 dB | 87.00 dB | 0.14 dB | 101.65 dB |
| 09 - Seek & Destroy.mp3 | 1411 | 87.07 dB | 86.93 dB | 0.14 dB | 101.57 dB |
| 10 - Whiplash.mp3 | 1411 | 87.72 dB | 87.58 dB | 0.14 dB | 102.20 dB |
| 11 - Fight Fire with Fire.mp3 | 1411 | 86.11 dB | 85.96 dB | 0.14 dB | 100.63 dB |
| 14 - Motorbreath.mp3 | 1411 | 86.28 dB | 86.14 dB | 0.14 dB | 100.83 dB |
| Pillars_of_the_Sky.wav | 1536 | ∞ | 99.91 dB | ∞ | 99.91 dB |
| Starlight_Ascent.wav | 1536 | ∞ | 102.10 dB | ∞ | 102.10 dB |
| Stellar_Ascent.wav | 1536 | ∞ | 102.83 dB | ∞ | 102.83 dB |

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
**file AUC** averages each carrier's chunk scores into one score per file, which is what a warden holding whole
files would use; a bias too slight to show in one chunk can add up over a file. The **95% intervals** are
bootstrapped by carrier: each round redraws whole carriers, with all their chunks, because chunks of one track are
not independent. Their width reflects how many carriers there are, not how many chunks. The verdict is the worse
of the chunk and file verdicts.

**ε ≥** is what the file AUC proves about Mist in Cachin's sense, where a scheme is ε-secure when the relative
entropy between clean and stego files is at most ε. A detector's |AUC − ½| is at most the total variation between
the two, and Pinsker's inequality turns that into ε ≥ 2(AUC − ½)² nats. It is a **lower bound** only: a detector
at chance shows that this detector found no gap, not that ε is small. The bracket applies the same formula across
the file AUC's interval.

Chi-square, SPA and RS look for bits being overwritten, which Mist never does, so they are expected to sit at
chance; a rise means the embedder has drifted. HCF-COM looks for ±1 changes, which is exactly what Mist does,
so it is the classical detector that matters.

The **classifier** is the adversary of record. It is a logistic regression trained on this run's own clean and
stego chunks, using every detector's score, the share of values at each of -3…3, and how each step between
adjacent values follows the one before it. It is cross-validated by carrier: every carrier is scored by a model
trained without it, so it cannot win by memorising a track. Its message-size check trains a second model to
tell a 64-byte message from a 1-byte one.

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

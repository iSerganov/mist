# Mist harness report

 perceptual metric: not installed

Every carrier is encoded three times: by the ffmpeg command line at its own defaults (the *clean* copy, which is what a warden without the original would compare against), by Mist's own encoder with nothing embedded (Mist's *own* re-encode), and by Mist with a hidden message (the *stego* copy). The report asks whether the stego copy differs from the clean one in plain properties, whether a detector can tell them apart, and how much worse it sounds. [How to read this report](#how-to-read-this-report) explains every number and threshold.

## Summary

| Format | Carriers | Looks like ffmpeg? | Hidden from detectors? | Audio quality |
|---|---|---|---|---|
| ogg/vorbis | 13 / 13 | ❌ differs on 13 of 13 carriers: nominal bitrate, length | ❌ hcf-com detects it (AUC 1.000) | ✅ Mist's own error sits 54.01 dB below the music; embedding costs 0.01 dB, within the 0.3 dB target |
| flac | 13 / 13 | ✅ matches ffmpeg on all 13 carriers | ⚠️ faint signal from chi-square (AUC 0.500) | ✅ inaudible: the added error is at least 99.89 dB below the music |
| wav/pcm_s16le | 13 / 13 | ✅ matches ffmpeg on all 13 carriers | ⚠️ faint signal from chi-square (AUC 0.500) | ✅ inaudible: the added error is at least 79.57 dB below the music |

## ogg/vorbis

Measured on 13 of 13 carriers.

### Does it look like a plain ffmpeg encode?

| Carrier | Source samples | Samples (ffmpeg / Mist) | Zero tail (ffmpeg / Mist) | Sample format (ffmpeg / Mist) | Nominal kbps (ffmpeg / Mist) | Differs in |
|---|---|---|---|---|---|---|
| 02 - Ride the Lightning.mp3 | 18406656 | 18407616 / 18407616 | 0 / 0 | fltp / fltp | 112 / 256 | nominal bitrate |
| 03 - Disposable Heroes.mp3 | 24553728 | 24553792 / 24553792 | 0 / 0 | fltp / fltp | 112 / 256 | nominal bitrate |
| 04 - No Remorse.mp3 | 17521920 | 17522880 / 17522880 | 0 / 0 | fltp / fltp | 112 / 256 | nominal bitrate |
| 06 - For Whom the Bell Tolls.mp3 | 11669760 | 11670592 / 11670592 | 0 / 0 | fltp / fltp | 112 / 256 | nominal bitrate |
| 07 - The Four Horsemen.mp3 | 14614272 | 14614720 / 14614720 | 0 / 0 | fltp / fltp | 112 / 256 | nominal bitrate |
| 08 - Fade to Black.mp3 | 19317888 | 19318208 / 19318208 | 0 / 0 | fltp / fltp | 112 / 256 | nominal bitrate |
| 09 - Seek & Destroy.mp3 | 18188928 | 18189376 / 18189376 | 0 / 0 | fltp / fltp | 112 / 256 | nominal bitrate |
| 10 - Whiplash.mp3 | 13644288 | 13644864 / 13644864 | 0 / 0 | fltp / fltp | 112 / 256 | nominal bitrate |
| 11 - Fight Fire with Fire.mp3 | 11631744 | 11632448 / 11632448 | 0 / 0 | fltp / fltp | 112 / 256 | nominal bitrate |
| 14 - Motorbreath.mp3 | 12556800 | 12556992 / 12557248 | 30720 / 30720 | fltp / fltp | 112 / 256 | length, nominal bitrate |
| Pillars_of_the_Sky.wav | 2880000 | 2880192 / 2880192 | 0 / 0 | fltp / fltp | 112 / 256 | nominal bitrate |
| Starlight_Ascent.wav | 5760000 | 5760064 / 5760320 | 0 / 0 | fltp / fltp | 112 / 256 | length, nominal bitrate |
| Stellar_Ascent.wav | 5760000 | 5760960 / 5760192 | 0 / 0 | fltp / fltp | 112 / 256 | length, nominal bitrate |

**Verdict:** ❌ differs on 13 of 13 carriers: nominal bitrate, length.

### Can a detector tell?

Stego copy against the clean ffmpeg copy.

| Detector | Looks for | Chunk AUC (95%) | File AUC (95%) | ε ≥ (nats, 95%) | Verdict | Message size |
|---|---|---|---|---|---|---|
| chi-square | bit overwriting (not what Mist does) | 0.500 (0.500–0.500) | 0.500 (0.500–0.500) | 0.000 (0.000–0.000) | ✅ chance | ✅ hidden (0.500) |
| spa | bit overwriting (not what Mist does) | 0.501 (0.500–0.502) | 0.538 (0.500–0.611) | 0.003 (0.000–0.025) | ✅ chance | ✅ hidden (0.500) |
| rs | bit overwriting (not what Mist does) | 0.501 (0.500–0.502) | 0.538 (0.500–0.611) | 0.003 (0.000–0.025) | ✅ chance | ✅ hidden (0.499) |
| hcf-com | ±1 changes, which is what Mist does | 1.000 (1.000–1.000) | 1.000 (1.000–1.000) | 0.500 (0.500–0.500) | ❌ detectable | ✅ hidden (0.501) |
| classifier | anything it can learn from Mist's own output | 1.000 (1.000–1.000) | 1.000 (1.000–1.000) | 0.500 (0.500–0.500) | ❌ detectable | ✅ hidden (0.500) |
| markov | how the waveform's curvature changes from sample to sample | 1.000 (1.000–1.000) | 1.000 (1.000–1.000) | 0.500 (0.500–0.500) | ❌ detectable | ✅ hidden (0.500) |

**Verdict:** ❌ hcf-com detects it (AUC 1.000).

### The embedding alone

Stego copy against Mist's own re-encode, so only the embedded changes differ.

| Detector | Looks for | Chunk AUC (95%) | File AUC (95%) | ε ≥ (nats, 95%) | Verdict | Message size |
|---|---|---|---|---|---|---|
| chi-square | bit overwriting (not what Mist does) | 0.500 (0.500–0.500) | 0.500 (0.500–0.500) | 0.000 (0.000–0.000) | ✅ chance | ✅ hidden (0.500) |
| spa | bit overwriting (not what Mist does) | 0.500 (0.500–0.500) | 0.497 (0.491–0.500) | 0.000 (0.000–0.000) | ✅ chance | ✅ hidden (0.500) |
| rs | bit overwriting (not what Mist does) | 0.499 (0.498–0.500) | 0.462 (0.394–0.500) | 0.003 (0.000–0.023) | ✅ chance | ✅ hidden (0.499) |
| hcf-com | ±1 changes, which is what Mist does | 0.508 (0.505–0.510) | 0.544 (0.537–0.576) | 0.004 (0.003–0.012) | ⚠️ faint signal | ✅ hidden (0.501) |
| classifier | anything it can learn from Mist's own output | 0.491 (0.488–0.496) | 0.444 (0.401–0.463) | 0.006 (0.003–0.019) | ⚠️ faint signal | ✅ hidden (0.500) |
| markov | how the waveform's curvature changes from sample to sample | 0.484 (0.478–0.492) | 0.432 (0.355–0.462) | 0.009 (0.003–0.042) | ⚠️ faint signal | ✅ hidden (0.500) |

### How much does it cost in sound?

| Measure | Mean | Worst | What it means |
|---|---|---|---|
| Plain re-encode SDR | 28.30 dB | 26.02 dB | Loss of Mist's own re-encode, with nothing embedded |
| Mist output SDR | 28.29 dB | 26.00 dB | The same, with the message embedded |
| Embedding cost | 0.01 dB | 0.02 dB | Mist's own share; target ≤ 0.3 dB, design-doc reference 0.18 dB |
| Extra error energy | +0% | +1% | Embedding cost as error added on top of a plain re-encode |
| Mist's error below the music | 54.01 dB | 48.78 dB | Stego copy against Mist's own re-encode: what embedding alone adds |

**Verdict:** ✅ Mist's own error sits 54.01 dB below the music; embedding costs 0.01 dB, within the 0.3 dB target.

### Per carrier

| Carrier | Output kbps | Plain re-encode SDR | Mist output SDR | Embedding cost | Mist's error below the music |
|---|---|---|---|---|---|
| 02 - Ride the Lightning.mp3 | 226 | 28.08 dB | 28.07 dB | 0.01 dB | 54.11 dB |
| 03 - Disposable Heroes.mp3 | 225 | 28.75 dB | 28.74 dB | 0.01 dB | 54.91 dB |
| 04 - No Remorse.mp3 | 218 | 29.32 dB | 29.31 dB | 0.01 dB | 57.04 dB |
| 06 - For Whom the Bell Tolls.mp3 | 226 | 28.37 dB | 28.36 dB | 0.01 dB | 57.16 dB |
| 07 - The Four Horsemen.mp3 | 220 | 28.71 dB | 28.70 dB | 0.01 dB | 56.47 dB |
| 08 - Fade to Black.mp3 | 224 | 29.52 dB | 29.51 dB | 0.01 dB | 55.42 dB |
| 09 - Seek & Destroy.mp3 | 226 | 28.88 dB | 28.87 dB | 0.01 dB | 55.17 dB |
| 10 - Whiplash.mp3 | 220 | 29.27 dB | 29.27 dB | 0.01 dB | 56.92 dB |
| 11 - Fight Fire with Fire.mp3 | 189 | 27.91 dB | 27.89 dB | 0.01 dB | 52.17 dB |
| 14 - Motorbreath.mp3 | 192 | 29.20 dB | 29.18 dB | 0.01 dB | 53.87 dB |
| Pillars_of_the_Sky.wav | 262 | 26.02 dB | 26.00 dB | 0.02 dB | 48.78 dB |
| Starlight_Ascent.wav | 282 | 26.09 dB | 26.07 dB | 0.02 dB | 49.49 dB |
| Stellar_Ascent.wav | 261 | 27.80 dB | 27.78 dB | 0.02 dB | 50.65 dB |

## flac

Measured on 13 of 13 carriers.

### Does it look like a plain ffmpeg encode?

| Carrier | Source samples | Samples (ffmpeg / Mist) | Zero tail (ffmpeg / Mist) | Sample format (ffmpeg / Mist) | Nominal kbps (ffmpeg / Mist) | Differs in |
|---|---|---|---|---|---|---|
| 02 - Ride the Lightning.mp3 | 18406656 | 18406656 / 18406656 | 0 / 0 | s32 / s32 | 0 / 0 | — |
| 03 - Disposable Heroes.mp3 | 24553728 | 24553728 / 24553728 | 0 / 0 | s32 / s32 | 0 / 0 | — |
| 04 - No Remorse.mp3 | 17521920 | 17521920 / 17521920 | 0 / 0 | s32 / s32 | 0 / 0 | — |
| 06 - For Whom the Bell Tolls.mp3 | 11669760 | 11669760 / 11669760 | 0 / 0 | s32 / s32 | 0 / 0 | — |
| 07 - The Four Horsemen.mp3 | 14614272 | 14614272 / 14614272 | 0 / 0 | s32 / s32 | 0 / 0 | — |
| 08 - Fade to Black.mp3 | 19317888 | 19317888 / 19317888 | 0 / 0 | s32 / s32 | 0 / 0 | — |
| 09 - Seek & Destroy.mp3 | 18188928 | 18188928 / 18188928 | 0 / 0 | s32 / s32 | 0 / 0 | — |
| 10 - Whiplash.mp3 | 13644288 | 13644288 / 13644288 | 486 / 486 | s32 / s32 | 0 / 0 | — |
| 11 - Fight Fire with Fire.mp3 | 11631744 | 11631744 / 11631744 | 288 / 288 | s32 / s32 | 0 / 0 | — |
| 14 - Motorbreath.mp3 | 12556800 | 12556800 / 12556800 | 30761 / 30761 | s32 / s32 | 0 / 0 | — |
| Pillars_of_the_Sky.wav | 2880000 | 2880000 / 2880000 | 0 / 0 | s16 / s16 | 0 / 0 | — |
| Starlight_Ascent.wav | 5760000 | 5760000 / 5760000 | 0 / 0 | s16 / s16 | 0 / 0 | — |
| Stellar_Ascent.wav | 5760000 | 5760000 / 5760000 | 0 / 0 | s16 / s16 | 0 / 0 | — |

**Verdict:** ✅ matches ffmpeg on all 13 carriers.

### Can a detector tell?

Stego copy against the clean ffmpeg copy.

| Detector | Looks for | Chunk AUC (95%) | File AUC (95%) | ε ≥ (nats, 95%) | Verdict | Message size |
|---|---|---|---|---|---|---|
| chi-square | bit overwriting (not what Mist does) | 0.500 (0.500–0.500) | 0.506 (0.487–0.524) | 0.000 (0.000–0.001) | ⚠️ faint signal | ✅ hidden (0.500) |
| spa | bit overwriting (not what Mist does) | 0.498 (0.497–0.499) | 0.432 (0.350–0.490) | 0.009 (0.000–0.045) | ⚠️ faint signal | ✅ hidden (0.497) |
| rs | bit overwriting (not what Mist does) | 0.502 (0.500–0.503) | 0.550 (0.509–0.613) | 0.005 (0.000–0.026) | ⚠️ faint signal | ✅ hidden (0.501) |
| hcf-com | ±1 changes, which is what Mist does | 0.500 (0.500–0.500) | 0.494 (0.482–0.506) | 0.000 (0.000–0.001) | ⚠️ faint signal | ✅ hidden (0.500) |
| classifier | anything it can learn from Mist's own output | 0.503 (0.501–0.504) | 0.527 (0.509–0.540) | 0.001 (0.000–0.003) | ⚠️ faint signal | ✅ hidden (0.503) |
| markov | how the waveform's curvature changes from sample to sample | 0.500 (0.500–0.501) | 0.509 (0.485–0.528) | 0.000 (0.000–0.002) | ✅ chance | ✅ hidden (0.500) |

**Verdict:** ⚠️ faint signal from chi-square (AUC 0.500).

### The embedding alone

Stego copy against Mist's own re-encode, so only the embedded changes differ.

| Detector | Looks for | Chunk AUC (95%) | File AUC (95%) | ε ≥ (nats, 95%) | Verdict | Message size |
|---|---|---|---|---|---|---|
| chi-square | bit overwriting (not what Mist does) | 0.500 (0.500–0.500) | 0.506 (0.487–0.524) | 0.000 (0.000–0.001) | ⚠️ faint signal | ✅ hidden (0.500) |
| spa | bit overwriting (not what Mist does) | 0.498 (0.497–0.499) | 0.432 (0.350–0.490) | 0.009 (0.000–0.045) | ⚠️ faint signal | ✅ hidden (0.497) |
| rs | bit overwriting (not what Mist does) | 0.502 (0.500–0.503) | 0.550 (0.509–0.613) | 0.005 (0.000–0.026) | ⚠️ faint signal | ✅ hidden (0.501) |
| hcf-com | ±1 changes, which is what Mist does | 0.500 (0.500–0.500) | 0.494 (0.482–0.506) | 0.000 (0.000–0.001) | ⚠️ faint signal | ✅ hidden (0.500) |
| classifier | anything it can learn from Mist's own output | 0.503 (0.501–0.504) | 0.527 (0.509–0.540) | 0.001 (0.000–0.003) | ⚠️ faint signal | ✅ hidden (0.503) |
| markov | how the waveform's curvature changes from sample to sample | 0.500 (0.500–0.501) | 0.509 (0.485–0.528) | 0.000 (0.000–0.002) | ✅ chance | ✅ hidden (0.500) |

### How much does it cost in sound?

| Measure | Mean | Worst | What it means |
|---|---|---|---|
| Plain re-encode SDR | ∞ | 128.38 dB | Loss of Mist's own re-encode, with nothing embedded |
| Mist output SDR | 122.89 dB | 99.89 dB | The same, with the message embedded |
| Embedding cost | ∞ | ∞ | Mist's own share; target ≤ 0.3 dB, design-doc reference 0.18 dB |
| Extra error energy | — | — | Embedding cost as error added on top of a plain re-encode |
| Mist's error below the music | 138.46 dB | 99.89 dB | Stego copy against Mist's own re-encode: what embedding alone adds |

**Verdict:** ✅ inaudible: the added error is at least 99.89 dB below the music.

### Per carrier

| Carrier | Output kbps | Plain re-encode SDR | Mist output SDR | Embedding cost | Mist's error below the music |
|---|---|---|---|---|---|
| 02 - Ride the Lightning.mp3 | 1578 | 128.57 dB | 128.53 dB | 0.04 dB | 148.80 dB |
| 03 - Disposable Heroes.mp3 | 1573 | 128.92 dB | 128.88 dB | 0.04 dB | 149.14 dB |
| 04 - No Remorse.mp3 | 1551 | 129.97 dB | 129.93 dB | 0.04 dB | 150.11 dB |
| 06 - For Whom the Bell Tolls.mp3 | 1545 | 128.38 dB | 128.34 dB | 0.04 dB | 148.63 dB |
| 07 - The Four Horsemen.mp3 | 1551 | 128.88 dB | 128.84 dB | 0.04 dB | 149.11 dB |
| 08 - Fade to Black.mp3 | 1573 | 129.99 dB | 129.95 dB | 0.04 dB | 150.14 dB |
| 09 - Seek & Destroy.mp3 | 1596 | 129.94 dB | 129.90 dB | 0.04 dB | 150.07 dB |
| 10 - Whiplash.mp3 | 1544 | 130.61 dB | 130.57 dB | 0.04 dB | 150.73 dB |
| 11 - Fight Fire with Fire.mp3 | 1531 | 128.87 dB | 128.83 dB | 0.04 dB | 149.12 dB |
| 14 - Motorbreath.mp3 | 1449 | 129.02 dB | 128.98 dB | 0.04 dB | 149.27 dB |
| Pillars_of_the_Sky.wav | 968 | ∞ | 99.89 dB | ∞ | 99.89 dB |
| Starlight_Ascent.wav | 1046 | ∞ | 102.10 dB | ∞ | 102.10 dB |
| Stellar_Ascent.wav | 978 | ∞ | 102.82 dB | ∞ | 102.82 dB |

## wav/pcm_s16le

Measured on 13 of 13 carriers.

### Does it look like a plain ffmpeg encode?

| Carrier | Source samples | Samples (ffmpeg / Mist) | Zero tail (ffmpeg / Mist) | Sample format (ffmpeg / Mist) | Nominal kbps (ffmpeg / Mist) | Differs in |
|---|---|---|---|---|---|---|
| 02 - Ride the Lightning.mp3 | 18406656 | 18406656 / 18406656 | 0 / 0 | s16 / s16 | 1411 / 1411 | — |
| 03 - Disposable Heroes.mp3 | 24553728 | 24553728 / 24553728 | 0 / 0 | s16 / s16 | 1411 / 1411 | — |
| 04 - No Remorse.mp3 | 17521920 | 17521920 / 17521920 | 1 / 1 | s16 / s16 | 1411 / 1411 | — |
| 06 - For Whom the Bell Tolls.mp3 | 11669760 | 11669760 / 11669760 | 0 / 0 | s16 / s16 | 1411 / 1411 | — |
| 07 - The Four Horsemen.mp3 | 14614272 | 14614272 / 14614272 | 0 / 0 | s16 / s16 | 1411 / 1411 | — |
| 08 - Fade to Black.mp3 | 19317888 | 19317888 / 19317888 | 0 / 0 | s16 / s16 | 1411 / 1411 | — |
| 09 - Seek & Destroy.mp3 | 18188928 | 18188928 / 18188928 | 0 / 0 | s16 / s16 | 1411 / 1411 | — |
| 10 - Whiplash.mp3 | 13644288 | 13644288 / 13644288 | 492 / 492 | s16 / s16 | 1411 / 1411 | — |
| 11 - Fight Fire with Fire.mp3 | 11631744 | 11631744 / 11631744 | 300 / 300 | s16 / s16 | 1411 / 1411 | — |
| 14 - Motorbreath.mp3 | 12556800 | 12556800 / 12556800 | 30781 / 30781 | s16 / s16 | 1411 / 1411 | — |
| Pillars_of_the_Sky.wav | 2880000 | 2880000 / 2880000 | 0 / 0 | s16 / s16 | 1536 / 1536 | — |
| Starlight_Ascent.wav | 5760000 | 5760000 / 5760000 | 0 / 0 | s16 / s16 | 1536 / 1536 | — |
| Stellar_Ascent.wav | 5760000 | 5760000 / 5760000 | 0 / 0 | s16 / s16 | 1536 / 1536 | — |

**Verdict:** ✅ matches ffmpeg on all 13 carriers.

### Can a detector tell?

Stego copy against the clean ffmpeg copy.

| Detector | Looks for | Chunk AUC (95%) | File AUC (95%) | ε ≥ (nats, 95%) | Verdict | Message size |
|---|---|---|---|---|---|---|
| chi-square | bit overwriting (not what Mist does) | 0.500 (0.499–0.500) | 0.512 (0.497–0.527) | 0.000 (0.000–0.001) | ⚠️ faint signal | ✅ hidden (0.499) |
| spa | bit overwriting (not what Mist does) | 0.499 (0.497–0.501) | 0.497 (0.435–0.554) | 0.000 (0.000–0.008) | ✅ chance | ✅ hidden (0.498) |
| rs | bit overwriting (not what Mist does) | 0.500 (0.499–0.502) | 0.538 (0.465–0.627) | 0.003 (0.000–0.032) | ✅ chance | ✅ hidden (0.500) |
| hcf-com | ±1 changes, which is what Mist does | 0.500 (0.500–0.500) | 0.509 (0.490–0.528) | 0.000 (0.000–0.002) | ✅ chance | ✅ hidden (0.500) |
| classifier | anything it can learn from Mist's own output | 0.500 (0.499–0.501) | 0.509 (0.484–0.541) | 0.000 (0.000–0.003) | ✅ chance | ✅ hidden (0.501) |
| markov | how the waveform's curvature changes from sample to sample | 0.500 (0.499–0.500) | 0.509 (0.491–0.528) | 0.000 (0.000–0.002) | ✅ chance | ✅ hidden (0.499) |

**Verdict:** ⚠️ faint signal from chi-square (AUC 0.500).

### The embedding alone

Stego copy against Mist's own re-encode, so only the embedded changes differ.

| Detector | Looks for | Chunk AUC (95%) | File AUC (95%) | ε ≥ (nats, 95%) | Verdict | Message size |
|---|---|---|---|---|---|---|
| chi-square | bit overwriting (not what Mist does) | 0.500 (0.499–0.500) | 0.512 (0.497–0.527) | 0.000 (0.000–0.001) | ⚠️ faint signal | ✅ hidden (0.499) |
| spa | bit overwriting (not what Mist does) | 0.499 (0.497–0.501) | 0.497 (0.435–0.554) | 0.000 (0.000–0.008) | ✅ chance | ✅ hidden (0.498) |
| rs | bit overwriting (not what Mist does) | 0.500 (0.499–0.502) | 0.538 (0.465–0.627) | 0.003 (0.000–0.032) | ✅ chance | ✅ hidden (0.500) |
| hcf-com | ±1 changes, which is what Mist does | 0.500 (0.500–0.500) | 0.509 (0.490–0.528) | 0.000 (0.000–0.002) | ✅ chance | ✅ hidden (0.500) |
| classifier | anything it can learn from Mist's own output | 0.500 (0.499–0.501) | 0.509 (0.484–0.541) | 0.000 (0.000–0.003) | ✅ chance | ✅ hidden (0.501) |
| markov | how the waveform's curvature changes from sample to sample | 0.500 (0.499–0.500) | 0.509 (0.491–0.528) | 0.000 (0.000–0.002) | ✅ chance | ✅ hidden (0.499) |

### How much does it cost in sound?

| Measure | Mean | Worst | What it means |
|---|---|---|---|
| Plain re-encode SDR | ∞ | 79.60 dB | Loss of Mist's own re-encode, with nothing embedded |
| Mist output SDR | 85.33 dB | 79.57 dB | The same, with the message embedded |
| Embedding cost | ∞ | ∞ | Mist's own share; target ≤ 0.3 dB, design-doc reference 0.18 dB |
| Extra error energy | — | — | Embedding cost as error added on top of a plain re-encode |
| Mist's error below the music | 101.41 dB | 99.90 dB | Stego copy against Mist's own re-encode: what embedding alone adds |

**Verdict:** ✅ inaudible: the added error is at least 79.57 dB below the music.

### Per carrier

| Carrier | Output kbps | Plain re-encode SDR | Mist output SDR | Embedding cost | Mist's error below the music |
|---|---|---|---|---|---|
| 02 - Ride the Lightning.mp3 | 1411 | 79.77 dB | 79.74 dB | 0.04 dB | 100.64 dB |
| 03 - Disposable Heroes.mp3 | 1411 | 80.11 dB | 80.07 dB | 0.04 dB | 100.97 dB |
| 04 - No Remorse.mp3 | 1411 | 81.09 dB | 81.05 dB | 0.04 dB | 101.95 dB |
| 06 - For Whom the Bell Tolls.mp3 | 1411 | 79.60 dB | 79.57 dB | 0.04 dB | 100.47 dB |
| 07 - The Four Horsemen.mp3 | 1411 | 80.08 dB | 80.05 dB | 0.04 dB | 100.94 dB |
| 08 - Fade to Black.mp3 | 1411 | 81.12 dB | 81.08 dB | 0.04 dB | 101.98 dB |
| 09 - Seek & Destroy.mp3 | 1411 | 81.05 dB | 81.02 dB | 0.04 dB | 101.91 dB |
| 10 - Whiplash.mp3 | 1411 | 81.70 dB | 81.66 dB | 0.04 dB | 102.56 dB |
| 11 - Fight Fire with Fire.mp3 | 1411 | 80.09 dB | 80.05 dB | 0.04 dB | 100.95 dB |
| 14 - Motorbreath.mp3 | 1411 | 80.24 dB | 80.20 dB | 0.04 dB | 101.12 dB |
| Pillars_of_the_Sky.wav | 1536 | ∞ | 99.90 dB | ∞ | 99.90 dB |
| Starlight_Ascent.wav | 1536 | ∞ | 102.09 dB | ∞ | 102.09 dB |
| Stellar_Ascent.wav | 1536 | ∞ | 102.82 dB | ∞ | 102.82 dB |

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

**Markov** is the same kind of model, trained the same way, on one richer feature set only: how the second
difference between samples, the waveform's curvature, changes from one sample to the next, with each value
clipped to -3…3. That curvature is near zero wherever the audio is smooth, so ±1 changes stand out in it more
than in the values or their steps. These are the rich-model features of audio steganalysis; a detector that
learns its own features, a CNN trained on exported chunks, is the next step beyond them and is not run here.

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

# Mist harness report

perceptual metric: not installed

Every carrier is encoded three times: by the ffmpeg command line at its own defaults (the *clean* copy, which is what a warden without the original would compare against), by Mist's own encoder with nothing embedded (Mist's *own* re-encode), and by Mist with a hidden message (the *stego* copy). The report asks whether the stego copy differs from the clean one in plain properties, whether a detector can tell them apart, and how much worse it sounds. [How to read this report](#how-to-read-this-report) explains every number and threshold.

## Summary

| Format | Carriers | Looks like ffmpeg? | Hidden from detectors? | Audio quality |
|---|---|---|---|---|
| ogg/vorbis | 13 / 13 | ❌ differs on 13 of 13 carriers: nominal bitrate, length, zero tail | ❌ hcf-com detects it (AUC 1.000) | ✅ Mist's own error sits 65.73 dB below the music; embedding costs 0.00 dB, within the 0.3 dB target |
| flac | 13 / 13 | ❌ differs on 13 of 13 carriers: length, zero tail, sample format | ❌ chi-square detects it (AUC 0.998) | ✅ inaudible: the added error is at least 81.59 dB below the music |
| wav/pcm_s16le | 13 / 13 | ❌ differs on 13 of 13 carriers: length, zero tail | ✅ chance on all 5 detectors | ✅ inaudible: the added error is at least 81.59 dB below the music |

## ogg/vorbis

Measured on 13 of 13 carriers.

### Does it look like a plain ffmpeg encode?

| Carrier | Source samples | Samples (ffmpeg / Mist) | Zero tail (ffmpeg / Mist) | Sample format (ffmpeg / Mist) | Nominal kbps (ffmpeg / Mist) | Differs in |
|---|---|---|---|---|---|---|
| 01 - Creeping Death.mp3 | 16319232 | 16319424 / 16319424 | 0 / 0 | fltp / fltp | 112 / 480 | nominal bitrate |
| 02 - Ride the Lightning.mp3 | 18406656 | 18407616 / 18407616 | 0 / 0 | fltp / fltp | 112 / 480 | nominal bitrate |
| 03 - Disposable Heroes.mp3 | 24553728 | 24553792 / 24553792 | 0 / 0 | fltp / fltp | 112 / 480 | nominal bitrate |
| 04 - No Remorse.mp3 | 17521920 | 17522880 / 17522880 | 0 / 0 | fltp / fltp | 112 / 480 | nominal bitrate |
| 06 - For Whom the Bell Tolls.mp3 | 11669760 | 11670592 / 11670592 | 0 / 0 | fltp / fltp | 112 / 480 | nominal bitrate |
| 07 - The Four Horsemen.mp3 | 14614272 | 14614720 / 14614720 | 0 / 0 | fltp / fltp | 112 / 480 | nominal bitrate |
| 08 - Fade to Black.mp3 | 19317888 | 19318208 / 19318336 | 0 / 0 | fltp / fltp | 112 / 480 | length, nominal bitrate |
| 09 - Seek & Destroy.mp3 | 18188928 | 18189376 / 18189376 | 0 / 0 | fltp / fltp | 112 / 480 | nominal bitrate |
| 10 - Whiplash.mp3 | 13644288 | 13644864 / 13644864 | 0 / 0 | fltp / fltp | 112 / 480 | nominal bitrate |
| 11 - Fight Fire with Fire.mp3 | 11631744 | 11632448 / 11631744 | 0 / 256 | fltp / fltp | 112 / 480 | length, zero tail, nominal bitrate |
| 12 - Kirk Doodle.mp3 | 7276032 | 7276992 / 7276480 | 0 / 0 | fltp / fltp | 112 / 480 | length, nominal bitrate |
| 13 - Am I Evil.mp3 | 9809280 | 9809600 / 9809280 | 0 / 0 | fltp / fltp | 112 / 480 | length, nominal bitrate |
| 14 - Motorbreath.mp3 | 12556800 | 12556992 / 12557248 | 30720 / 30720 | fltp / fltp | 112 / 480 | length, nominal bitrate |

**Verdict:** ❌ differs on 13 of 13 carriers: nominal bitrate, length, zero tail.

### Can a detector tell?

Stego copy against the clean ffmpeg copy.

| Detector | Looks for | AUC | 95% interval | Verdict | Message size |
|---|---|---|---|---|---|
| chi-square | bit overwriting (not what Mist does) | 0.500 | 0.500–0.501 | ✅ chance | ✅ hidden (0.500) |
| spa | bit overwriting (not what Mist does) | 0.500 | 0.500–0.500 | ✅ chance | ✅ hidden (0.500) |
| rs | bit overwriting (not what Mist does) | 0.500 | 0.500–0.501 | ✅ chance | ✅ hidden (0.500) |
| hcf-com | ±1 changes, which is what Mist does | 1.000 | 1.000–1.000 | ❌ detectable | ✅ hidden (0.500) |
| classifier | anything it can learn from Mist's own output | 1.000 | 1.000–1.000 | ❌ detectable | ✅ hidden (0.500) |

**Verdict:** ❌ hcf-com detects it (AUC 1.000).

### The embedding alone

Stego copy against Mist's own re-encode, so only the embedded changes differ.

| Detector | Looks for | AUC | 95% interval | Verdict | Message size |
|---|---|---|---|---|---|
| chi-square | bit overwriting (not what Mist does) | 0.500 | 0.499–0.501 | ✅ chance | ✅ hidden (0.500) |
| spa | bit overwriting (not what Mist does) | 0.500 | 0.500–0.500 | ✅ chance | ✅ hidden (0.500) |
| rs | bit overwriting (not what Mist does) | 0.500 | 0.500–0.501 | ✅ chance | ✅ hidden (0.500) |
| hcf-com | ±1 changes, which is what Mist does | 0.497 | 0.480–0.515 | ✅ chance | ✅ hidden (0.500) |
| classifier | anything it can learn from Mist's own output | 0.501 | 0.484–0.520 | ✅ chance | ✅ hidden (0.500) |

### How much does it cost in sound?

| Measure | Mean | Worst | What it means |
|---|---|---|---|
| Plain re-encode SDR | 35.63 dB | 34.20 dB | Loss of Mist's own re-encode, with nothing embedded |
| Mist output SDR | 35.62 dB | 34.20 dB | The same, with the message embedded |
| Embedding cost | 0.00 dB | 0.01 dB | Mist's own share; target ≤ 0.3 dB, design-doc reference 0.18 dB |
| Extra error energy | +0% | +0% | Embedding cost as error added on top of a plain re-encode |
| Mist's error below the music | 65.73 dB | 62.52 dB | Stego copy against Mist's own re-encode: what embedding alone adds |

**Verdict:** ✅ Mist's own error sits 65.73 dB below the music; embedding costs 0.00 dB, within the 0.3 dB target.

### Per carrier

| Carrier | Output kbps | Plain re-encode SDR | Mist output SDR | Embedding cost | Mist's error below the music |
|---|---|---|---|---|---|
| 01 - Creeping Death.mp3 | 348 | 34.41 dB | 34.40 dB | 0.01 dB | 63.68 dB |
| 02 - Ride the Lightning.mp3 | 391 | 35.22 dB | 35.21 dB | 0.00 dB | 65.16 dB |
| 03 - Disposable Heroes.mp3 | 392 | 35.73 dB | 35.73 dB | 0.00 dB | 66.00 dB |
| 04 - No Remorse.mp3 | 381 | 35.87 dB | 35.87 dB | 0.00 dB | 67.84 dB |
| 06 - For Whom the Bell Tolls.mp3 | 385 | 35.07 dB | 35.06 dB | 0.00 dB | 67.86 dB |
| 07 - The Four Horsemen.mp3 | 383 | 35.56 dB | 35.55 dB | 0.00 dB | 67.55 dB |
| 08 - Fade to Black.mp3 | 386 | 36.20 dB | 36.20 dB | 0.00 dB | 66.33 dB |
| 09 - Seek & Destroy.mp3 | 390 | 35.94 dB | 35.93 dB | 0.00 dB | 66.15 dB |
| 10 - Whiplash.mp3 | 379 | 36.20 dB | 36.20 dB | 0.00 dB | 67.87 dB |
| 11 - Fight Fire with Fire.mp3 | 334 | 36.30 dB | 36.29 dB | 0.01 dB | 62.52 dB |
| 12 - Kirk Doodle.mp3 | 320 | 34.20 dB | 34.20 dB | 0.00 dB | 67.23 dB |
| 13 - Am I Evil.mp3 | 330 | 36.00 dB | 35.99 dB | 0.01 dB | 62.52 dB |
| 14 - Motorbreath.mp3 | 335 | 36.45 dB | 36.45 dB | 0.01 dB | 63.82 dB |

## flac

Measured on 13 of 13 carriers.

### Does it look like a plain ffmpeg encode?

| Carrier | Source samples | Samples (ffmpeg / Mist) | Zero tail (ffmpeg / Mist) | Sample format (ffmpeg / Mist) | Nominal kbps (ffmpeg / Mist) | Differs in |
|---|---|---|---|---|---|---|
| 01 - Creeping Death.mp3 | 16319232 | 16319232 / 16321536 | 0 / 2304 | s32 / s16 | 0 / 0 | length, zero tail, sample format |
| 02 - Ride the Lightning.mp3 | 18406656 | 18406656 / 18408960 | 0 / 2304 | s32 / s16 | 0 / 0 | length, zero tail, sample format |
| 03 - Disposable Heroes.mp3 | 24553728 | 24553728 / 24556032 | 0 / 2304 | s32 / s16 | 0 / 0 | length, zero tail, sample format |
| 04 - No Remorse.mp3 | 17521920 | 17521920 / 17524224 | 0 / 2669 | s32 / s16 | 0 / 0 | length, zero tail, sample format |
| 06 - For Whom the Bell Tolls.mp3 | 11669760 | 11669760 / 11672064 | 0 / 2755 | s32 / s16 | 0 / 0 | length, zero tail, sample format |
| 07 - The Four Horsemen.mp3 | 14614272 | 14614272 / 14616576 | 0 / 2304 | s32 / s16 | 0 / 0 | length, zero tail, sample format |
| 08 - Fade to Black.mp3 | 19317888 | 19317888 / 19321344 | 0 / 3456 | s32 / s16 | 0 / 0 | length, zero tail, sample format |
| 09 - Seek & Destroy.mp3 | 18188928 | 18188928 / 18192384 | 0 / 3456 | s32 / s16 | 0 / 0 | length, zero tail, sample format |
| 10 - Whiplash.mp3 | 13644288 | 13644288 / 13644288 | 486 / 701 | s32 / s16 | 0 / 0 | zero tail, sample format |
| 11 - Fight Fire with Fire.mp3 | 11631744 | 11631744 / 11635200 | 288 / 3959 | s32 / s16 | 0 / 0 | length, zero tail, sample format |
| 12 - Kirk Doodle.mp3 | 7276032 | 7276032 / 7276032 | 0 / 0 | s32 / s16 | 0 / 0 | sample format |
| 13 - Am I Evil.mp3 | 9809280 | 9809280 / 9810432 | 0 / 1152 | s32 / s16 | 0 / 0 | length, zero tail, sample format |
| 14 - Motorbreath.mp3 | 12556800 | 12556800 / 12556800 | 30761 / 31874 | s32 / s16 | 0 / 0 | zero tail, sample format |

**Verdict:** ❌ differs on 13 of 13 carriers: length, zero tail, sample format.

### Can a detector tell?

Stego copy against the clean ffmpeg copy.

| Detector | Looks for | AUC | 95% interval | Verdict | Message size |
|---|---|---|---|---|---|
| chi-square | bit overwriting (not what Mist does) | 0.998 | 0.997–0.999 | ❌ detectable | ✅ hidden (0.500) |
| spa | bit overwriting (not what Mist does) | 0.195 | 0.187–0.202 | ❌ detectable | ✅ hidden (0.500) |
| rs | bit overwriting (not what Mist does) | 0.265 | 0.257–0.273 | ❌ detectable | ✅ hidden (0.500) |
| hcf-com | ±1 changes, which is what Mist does | 0.999 | 0.998–1.000 | ❌ detectable | ✅ hidden (0.500) |
| classifier | anything it can learn from Mist's own output | 0.999 | 0.998–0.999 | ❌ detectable | ✅ hidden (0.500) |

**Verdict:** ❌ chi-square detects it (AUC 0.998).

### The embedding alone

Stego copy against Mist's own re-encode, so only the embedded changes differ.

| Detector | Looks for | AUC | 95% interval | Verdict | Message size |
|---|---|---|---|---|---|
| chi-square | bit overwriting (not what Mist does) | 0.500 | 0.497–0.503 | ✅ chance | ✅ hidden (0.500) |
| spa | bit overwriting (not what Mist does) | 0.499 | 0.490–0.509 | ✅ chance | ✅ hidden (0.500) |
| rs | bit overwriting (not what Mist does) | 0.499 | 0.490–0.509 | ✅ chance | ✅ hidden (0.500) |
| hcf-com | ±1 changes, which is what Mist does | 0.500 | 0.490–0.511 | ✅ chance | ✅ hidden (0.500) |
| classifier | anything it can learn from Mist's own output | 0.500 | 0.489–0.510 | ✅ chance | ✅ hidden (0.500) |

### How much does it cost in sound?

| Measure | Mean | Worst | What it means |
|---|---|---|---|
| Plain re-encode SDR | 85.98 dB | 81.73 dB | Loss of Mist's own re-encode, with nothing embedded |
| Mist output SDR | 85.84 dB | 81.59 dB | The same, with the message embedded |
| Embedding cost | 0.14 dB | 0.14 dB | Mist's own share; target ≤ 0.3 dB, design-doc reference 0.18 dB |
| Extra error energy | +3% | +3% | Embedding cost as error added on top of a plain re-encode |
| Mist's error below the music | 100.54 dB | 96.40 dB | Stego copy against Mist's own re-encode: what embedding alone adds |

**Verdict:** ✅ inaudible: the added error is at least 81.59 dB below the music.

### Per carrier

| Carrier | Output kbps | Plain re-encode SDR | Mist output SDR | Embedding cost | Mist's error below the music |
|---|---|---|---|---|---|
| 01 - Creeping Death.mp3 | 809 | 84.70 dB | 84.56 dB | 0.14 dB | 99.33 dB |
| 02 - Ride the Lightning.mp3 | 872 | 85.79 dB | 85.65 dB | 0.14 dB | 100.35 dB |
| 03 - Disposable Heroes.mp3 | 867 | 86.13 dB | 85.99 dB | 0.14 dB | 100.68 dB |
| 04 - No Remorse.mp3 | 846 | 87.10 dB | 86.96 dB | 0.14 dB | 101.62 dB |
| 06 - For Whom the Bell Tolls.mp3 | 840 | 85.62 dB | 85.48 dB | 0.14 dB | 100.20 dB |
| 07 - The Four Horsemen.mp3 | 845 | 86.10 dB | 85.96 dB | 0.14 dB | 100.66 dB |
| 08 - Fade to Black.mp3 | 867 | 87.14 dB | 87.00 dB | 0.14 dB | 101.67 dB |
| 09 - Seek & Destroy.mp3 | 891 | 87.07 dB | 86.93 dB | 0.14 dB | 101.59 dB |
| 10 - Whiplash.mp3 | 839 | 87.72 dB | 87.58 dB | 0.14 dB | 102.23 dB |
| 11 - Fight Fire with Fire.mp3 | 826 | 86.11 dB | 85.97 dB | 0.14 dB | 100.66 dB |
| 12 - Kirk Doodle.mp3 | 714 | 81.73 dB | 81.59 dB | 0.14 dB | 96.40 dB |
| 13 - Am I Evil.mp3 | 794 | 86.27 dB | 86.13 dB | 0.14 dB | 100.83 dB |
| 14 - Motorbreath.mp3 | 751 | 86.28 dB | 86.14 dB | 0.14 dB | 100.85 dB |

## wav/pcm_s16le

Measured on 13 of 13 carriers.

### Does it look like a plain ffmpeg encode?

| Carrier | Source samples | Samples (ffmpeg / Mist) | Zero tail (ffmpeg / Mist) | Sample format (ffmpeg / Mist) | Nominal kbps (ffmpeg / Mist) | Differs in |
|---|---|---|---|---|---|---|
| 01 - Creeping Death.mp3 | 16319232 | 16319232 / 16322560 | 0 / 3328 | s16 / s16 | 1411 / 1411 | length, zero tail |
| 02 - Ride the Lightning.mp3 | 18406656 | 18406656 / 18407424 | 0 / 768 | s16 / s16 | 1411 / 1411 | length, zero tail |
| 03 - Disposable Heroes.mp3 | 24553728 | 24553728 / 24555520 | 0 / 1792 | s16 / s16 | 1411 / 1411 | length, zero tail |
| 04 - No Remorse.mp3 | 17521920 | 17521920 / 17522688 | 1 / 1133 | s16 / s16 | 1411 / 1411 | length, zero tail |
| 06 - For Whom the Bell Tolls.mp3 | 11669760 | 11669760 / 11673600 | 0 / 4291 | s16 / s16 | 1411 / 1411 | length, zero tail |
| 07 - The Four Horsemen.mp3 | 14614272 | 14614272 / 14614528 | 0 / 256 | s16 / s16 | 1411 / 1411 | length, zero tail |
| 08 - Fade to Black.mp3 | 19317888 | 19317888 / 19320832 | 0 / 2944 | s16 / s16 | 1411 / 1411 | length, zero tail |
| 09 - Seek & Destroy.mp3 | 18188928 | 18188928 / 18190336 | 0 / 1408 | s16 / s16 | 1411 / 1411 | length, zero tail |
| 10 - Whiplash.mp3 | 13644288 | 13644288 / 13647872 | 492 / 4285 | s16 / s16 | 1411 / 1411 | length, zero tail |
| 11 - Fight Fire with Fire.mp3 | 11631744 | 11631744 / 11632640 | 300 / 1399 | s16 / s16 | 1411 / 1411 | length, zero tail |
| 12 - Kirk Doodle.mp3 | 7276032 | 7276032 / 7278592 | 0 / 2560 | s16 / s16 | 1411 / 1411 | length, zero tail |
| 13 - Am I Evil.mp3 | 9809280 | 9809280 / 9809920 | 0 / 640 | s16 / s16 | 1411 / 1411 | length, zero tail |
| 14 - Motorbreath.mp3 | 12556800 | 12556800 / 12558336 | 30781 / 33410 | s16 / s16 | 1411 / 1411 | length, zero tail |

**Verdict:** ❌ differs on 13 of 13 carriers: length, zero tail.

### Can a detector tell?

Stego copy against the clean ffmpeg copy.

| Detector | Looks for | AUC | 95% interval | Verdict | Message size |
|---|---|---|---|---|---|
| chi-square | bit overwriting (not what Mist does) | 0.499 | 0.497–0.502 | ✅ chance | ✅ hidden (0.500) |
| spa | bit overwriting (not what Mist does) | 0.503 | 0.494–0.513 | ✅ chance | ✅ hidden (0.499) |
| rs | bit overwriting (not what Mist does) | 0.497 | 0.487–0.505 | ✅ chance | ✅ hidden (0.499) |
| hcf-com | ±1 changes, which is what Mist does | 0.498 | 0.487–0.509 | ✅ chance | ✅ hidden (0.500) |
| classifier | anything it can learn from Mist's own output | 0.509 | 0.498–0.519 | ✅ chance | ✅ hidden (0.500) |

**Verdict:** ✅ chance on all 5 detectors.

### The embedding alone

Stego copy against Mist's own re-encode, so only the embedded changes differ.

| Detector | Looks for | AUC | 95% interval | Verdict | Message size |
|---|---|---|---|---|---|
| chi-square | bit overwriting (not what Mist does) | 0.500 | 0.497–0.502 | ✅ chance | ✅ hidden (0.500) |
| spa | bit overwriting (not what Mist does) | 0.500 | 0.490–0.509 | ✅ chance | ✅ hidden (0.499) |
| rs | bit overwriting (not what Mist does) | 0.499 | 0.490–0.508 | ✅ chance | ✅ hidden (0.499) |
| hcf-com | ±1 changes, which is what Mist does | 0.500 | 0.489–0.510 | ✅ chance | ✅ hidden (0.500) |
| classifier | anything it can learn from Mist's own output | 0.500 | 0.490–0.511 | ✅ chance | ✅ hidden (0.500) |

### How much does it cost in sound?

| Measure | Mean | Worst | What it means |
|---|---|---|---|
| Plain re-encode SDR | 85.98 dB | 81.73 dB | Loss of Mist's own re-encode, with nothing embedded |
| Mist output SDR | 85.84 dB | 81.59 dB | The same, with the message embedded |
| Embedding cost | 0.14 dB | 0.14 dB | Mist's own share; target ≤ 0.3 dB, design-doc reference 0.18 dB |
| Extra error energy | +3% | +3% | Embedding cost as error added on top of a plain re-encode |
| Mist's error below the music | 100.54 dB | 96.40 dB | Stego copy against Mist's own re-encode: what embedding alone adds |

**Verdict:** ✅ inaudible: the added error is at least 81.59 dB below the music.

### Per carrier

| Carrier | Output kbps | Plain re-encode SDR | Mist output SDR | Embedding cost | Mist's error below the music |
|---|---|---|---|---|---|
| 01 - Creeping Death.mp3 | 1411 | 84.70 dB | 84.56 dB | 0.14 dB | 99.33 dB |
| 02 - Ride the Lightning.mp3 | 1411 | 85.79 dB | 85.65 dB | 0.14 dB | 100.35 dB |
| 03 - Disposable Heroes.mp3 | 1411 | 86.13 dB | 85.99 dB | 0.14 dB | 100.68 dB |
| 04 - No Remorse.mp3 | 1411 | 87.10 dB | 86.96 dB | 0.14 dB | 101.63 dB |
| 06 - For Whom the Bell Tolls.mp3 | 1411 | 85.62 dB | 85.48 dB | 0.14 dB | 100.19 dB |
| 07 - The Four Horsemen.mp3 | 1411 | 86.10 dB | 85.96 dB | 0.14 dB | 100.66 dB |
| 08 - Fade to Black.mp3 | 1411 | 87.14 dB | 87.00 dB | 0.14 dB | 101.67 dB |
| 09 - Seek & Destroy.mp3 | 1411 | 87.07 dB | 86.93 dB | 0.14 dB | 101.59 dB |
| 10 - Whiplash.mp3 | 1411 | 87.72 dB | 87.58 dB | 0.14 dB | 102.23 dB |
| 11 - Fight Fire with Fire.mp3 | 1411 | 86.11 dB | 85.97 dB | 0.14 dB | 100.66 dB |
| 12 - Kirk Doodle.mp3 | 1411 | 81.73 dB | 81.59 dB | 0.14 dB | 96.40 dB |
| 13 - Am I Evil.mp3 | 1411 | 86.27 dB | 86.13 dB | 0.14 dB | 100.83 dB |
| 14 - Motorbreath.mp3 | 1411 | 86.28 dB | 86.14 dB | 0.14 dB | 100.85 dB |

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
detector is right, just with its sign flipped. Each carrier is cut into chunks of 65,536 values and every chunk
is scored, so the **95% interval** says how sure the estimate is. Ogg Vorbis is scored on the residues Mist may
change, lossless formats on the decoded samples.

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

# Mist harness report

perceptual metric: not installed

Every carrier is re-encoded twice: once plainly (the *clean* copy) and once with a hidden message (the *stego* copy). The report asks whether a detector can tell the two apart, and how much worse the stego copy sounds. [How to read this report](#how-to-read-this-report) explains every number and threshold.

## Summary

| Format | Carriers | Hidden from detectors? | Audio quality |
|---|---|---|---|
| ogg/vorbis | 13 / 13 | ✅ chance on all 5 detectors | ✅ Mist's own error sits 65.73 dB below the music; embedding costs 0.00 dB, within the 0.3 dB target |
| flac | 13 / 13 | ✅ chance on all 5 detectors | ✅ inaudible: the added error is at least 81.59 dB below the music |
| wav/pcm_s16le | 13 / 13 | ✅ chance on all 5 detectors | ✅ inaudible: the added error is at least 81.59 dB below the music |

## ogg/vorbis

Measured on 13 of 13 carriers.

### Can a detector tell?

| Detector | Looks for | AUC | 95% interval | Verdict | Message size |
|---|---|---|---|---|---|
| chi-square | bit overwriting (not what Mist does) | 0.500 | 0.499–0.501 | ✅ chance | ✅ hidden (0.500) |
| spa | bit overwriting (not what Mist does) | 0.500 | 0.500–0.500 | ✅ chance | ✅ hidden (0.500) |
| rs | bit overwriting (not what Mist does) | 0.500 | 0.499–0.501 | ✅ chance | ✅ hidden (0.500) |
| hcf-com | ±1 changes, which is what Mist does | 0.497 | 0.480–0.515 | ✅ chance | ✅ hidden (0.500) |
| classifier | anything it can learn from Mist's own output | 0.502 | 0.484–0.520 | ✅ chance | ✅ hidden (0.500) |

**Verdict:** ✅ chance on all 5 detectors.

### How much does it cost in sound?

| Measure | Mean | Worst | What it means |
|---|---|---|---|
| Plain re-encode SDR | 35.63 dB | 34.20 dB | Loss any re-encode causes, Mist or not |
| Mist output SDR | 35.62 dB | 34.20 dB | The same, with the message embedded |
| Embedding cost | 0.00 dB | 0.01 dB | Mist's own share; target ≤ 0.3 dB, design-doc reference 0.18 dB |
| Extra error energy | +0% | +0% | Embedding cost as error added on top of a plain re-encode |
| Mist's error below the music | 65.73 dB | 62.27 dB | Stego copy against the clean one: what Mist alone adds |

**Verdict:** ✅ Mist's own error sits 65.73 dB below the music; embedding costs 0.00 dB, within the 0.3 dB target.

### Per carrier

| Carrier | Output kbps | Plain re-encode SDR | Mist output SDR | Embedding cost | Mist's error below the music |
|---|---|---|---|---|---|
| 01 - Creeping Death.mp3 | 348 | 34.41 dB | 34.40 dB | 0.01 dB | 63.50 dB |
| 02 - Ride the Lightning.mp3 | 391 | 35.22 dB | 35.21 dB | 0.00 dB | 65.24 dB |
| 03 - Disposable Heroes.mp3 | 392 | 35.73 dB | 35.73 dB | 0.00 dB | 65.99 dB |
| 04 - No Remorse.mp3 | 381 | 35.87 dB | 35.87 dB | 0.00 dB | 67.80 dB |
| 06 - For Whom the Bell Tolls.mp3 | 385 | 35.07 dB | 35.06 dB | 0.00 dB | 67.70 dB |
| 07 - The Four Horsemen.mp3 | 383 | 35.56 dB | 35.55 dB | 0.00 dB | 67.54 dB |
| 08 - Fade to Black.mp3 | 386 | 36.20 dB | 36.20 dB | 0.00 dB | 66.57 dB |
| 09 - Seek & Destroy.mp3 | 390 | 35.94 dB | 35.93 dB | 0.00 dB | 66.06 dB |
| 10 - Whiplash.mp3 | 379 | 36.20 dB | 36.20 dB | 0.00 dB | 67.91 dB |
| 11 - Fight Fire with Fire.mp3 | 334 | 36.30 dB | 36.29 dB | 0.01 dB | 62.27 dB |
| 12 - Kirk Doodle.mp3 | 320 | 34.20 dB | 34.20 dB | 0.00 dB | 67.33 dB |
| 13 - Am I Evil.mp3 | 330 | 36.00 dB | 35.99 dB | 0.01 dB | 62.50 dB |
| 14 - Motorbreath.mp3 | 335 | 36.45 dB | 36.45 dB | 0.01 dB | 64.06 dB |

## flac

Measured on 13 of 13 carriers.

### Can a detector tell?

| Detector | Looks for | AUC | 95% interval | Verdict | Message size |
|---|---|---|---|---|---|
| chi-square | bit overwriting (not what Mist does) | 0.500 | 0.497–0.503 | ✅ chance | ✅ hidden (0.500) |
| spa | bit overwriting (not what Mist does) | 0.500 | 0.491–0.510 | ✅ chance | ✅ hidden (0.501) |
| rs | bit overwriting (not what Mist does) | 0.498 | 0.489–0.508 | ✅ chance | ✅ hidden (0.500) |
| hcf-com | ±1 changes, which is what Mist does | 0.500 | 0.490–0.510 | ✅ chance | ✅ hidden (0.500) |
| classifier | anything it can learn from Mist's own output | 0.501 | 0.490–0.511 | ✅ chance | ✅ hidden (0.501) |

**Verdict:** ✅ chance on all 5 detectors.

### How much does it cost in sound?

| Measure | Mean | Worst | What it means |
|---|---|---|---|
| Plain re-encode SDR | 85.98 dB | 81.73 dB | Loss any re-encode causes, Mist or not |
| Mist output SDR | 85.84 dB | 81.59 dB | The same, with the message embedded |
| Embedding cost | 0.14 dB | 0.14 dB | Mist's own share; target ≤ 0.3 dB, design-doc reference 0.18 dB |
| Extra error energy | +3% | +3% | Embedding cost as error added on top of a plain re-encode |
| Mist's error below the music | 100.54 dB | 96.40 dB | Stego copy against the clean one: what Mist alone adds |

**Verdict:** ✅ inaudible: the added error is at least 81.59 dB below the music.

### Per carrier

| Carrier | Output kbps | Plain re-encode SDR | Mist output SDR | Embedding cost | Mist's error below the music |
|---|---|---|---|---|---|
| 01 - Creeping Death.mp3 | 809 | 84.70 dB | 84.56 dB | 0.14 dB | 99.33 dB |
| 02 - Ride the Lightning.mp3 | 873 | 85.79 dB | 85.65 dB | 0.14 dB | 100.35 dB |
| 03 - Disposable Heroes.mp3 | 867 | 86.13 dB | 85.99 dB | 0.14 dB | 100.67 dB |
| 04 - No Remorse.mp3 | 846 | 87.10 dB | 86.96 dB | 0.14 dB | 101.62 dB |
| 06 - For Whom the Bell Tolls.mp3 | 840 | 85.62 dB | 85.48 dB | 0.14 dB | 100.18 dB |
| 07 - The Four Horsemen.mp3 | 845 | 86.10 dB | 85.96 dB | 0.14 dB | 100.66 dB |
| 08 - Fade to Black.mp3 | 868 | 87.14 dB | 87.00 dB | 0.14 dB | 101.66 dB |
| 09 - Seek & Destroy.mp3 | 891 | 87.07 dB | 86.93 dB | 0.14 dB | 101.59 dB |
| 10 - Whiplash.mp3 | 839 | 87.72 dB | 87.58 dB | 0.14 dB | 102.22 dB |
| 11 - Fight Fire with Fire.mp3 | 826 | 86.11 dB | 85.97 dB | 0.14 dB | 100.66 dB |
| 12 - Kirk Doodle.mp3 | 714 | 81.73 dB | 81.59 dB | 0.14 dB | 96.40 dB |
| 13 - Am I Evil.mp3 | 794 | 86.27 dB | 86.13 dB | 0.14 dB | 100.82 dB |
| 14 - Motorbreath.mp3 | 751 | 86.28 dB | 86.14 dB | 0.14 dB | 100.85 dB |

## wav/pcm_s16le

Measured on 13 of 13 carriers.

### Can a detector tell?

| Detector | Looks for | AUC | 95% interval | Verdict | Message size |
|---|---|---|---|---|---|
| chi-square | bit overwriting (not what Mist does) | 0.500 | 0.498–0.503 | ✅ chance | ✅ hidden (0.501) |
| spa | bit overwriting (not what Mist does) | 0.500 | 0.491–0.509 | ✅ chance | ✅ hidden (0.501) |
| rs | bit overwriting (not what Mist does) | 0.499 | 0.490–0.508 | ✅ chance | ✅ hidden (0.499) |
| hcf-com | ±1 changes, which is what Mist does | 0.500 | 0.488–0.510 | ✅ chance | ✅ hidden (0.500) |
| classifier | anything it can learn from Mist's own output | 0.500 | 0.490–0.511 | ✅ chance | ✅ hidden (0.499) |

**Verdict:** ✅ chance on all 5 detectors.

### How much does it cost in sound?

| Measure | Mean | Worst | What it means |
|---|---|---|---|
| Plain re-encode SDR | 85.98 dB | 81.73 dB | Loss any re-encode causes, Mist or not |
| Mist output SDR | 85.84 dB | 81.59 dB | The same, with the message embedded |
| Embedding cost | 0.14 dB | 0.14 dB | Mist's own share; target ≤ 0.3 dB, design-doc reference 0.18 dB |
| Extra error energy | +3% | +3% | Embedding cost as error added on top of a plain re-encode |
| Mist's error below the music | 100.54 dB | 96.41 dB | Stego copy against the clean one: what Mist alone adds |

**Verdict:** ✅ inaudible: the added error is at least 81.59 dB below the music.

### Per carrier

| Carrier | Output kbps | Plain re-encode SDR | Mist output SDR | Embedding cost | Mist's error below the music |
|---|---|---|---|---|---|
| 01 - Creeping Death.mp3 | 1411 | 84.70 dB | 84.56 dB | 0.14 dB | 99.33 dB |
| 02 - Ride the Lightning.mp3 | 1411 | 85.79 dB | 85.65 dB | 0.14 dB | 100.35 dB |
| 03 - Disposable Heroes.mp3 | 1411 | 86.13 dB | 85.99 dB | 0.14 dB | 100.68 dB |
| 04 - No Remorse.mp3 | 1411 | 87.10 dB | 86.96 dB | 0.14 dB | 101.62 dB |
| 06 - For Whom the Bell Tolls.mp3 | 1412 | 85.62 dB | 85.48 dB | 0.14 dB | 100.19 dB |
| 07 - The Four Horsemen.mp3 | 1411 | 86.10 dB | 85.96 dB | 0.14 dB | 100.65 dB |
| 08 - Fade to Black.mp3 | 1411 | 87.14 dB | 87.00 dB | 0.14 dB | 101.66 dB |
| 09 - Seek & Destroy.mp3 | 1411 | 87.07 dB | 86.93 dB | 0.14 dB | 101.58 dB |
| 10 - Whiplash.mp3 | 1412 | 87.72 dB | 87.58 dB | 0.14 dB | 102.22 dB |
| 11 - Fight Fire with Fire.mp3 | 1411 | 86.11 dB | 85.97 dB | 0.14 dB | 100.66 dB |
| 12 - Kirk Doodle.mp3 | 1412 | 81.73 dB | 81.59 dB | 0.14 dB | 96.41 dB |
| 13 - Am I Evil.mp3 | 1411 | 86.27 dB | 86.13 dB | 0.14 dB | 100.82 dB |
| 14 - Motorbreath.mp3 | 1411 | 86.28 dB | 86.14 dB | 0.14 dB | 100.85 dB |

## How to read this report

### Detectability

**AUC** is the chance that a detector, shown one clean and one stego sample, picks the stego one. 0.5 is a coin
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
after lining them up in time. The **plain re-encode** is what any re-encode costs; the **embedding cost** is how
much lower the stego copy scores, which is Mist's own share. **Extra error energy** says the same thing as a
percentage: +37% means 37% more error than the plain re-encode alone.

The embedding cost is relative to the re-encode's own error, so the same perturbation reads as a larger cost on a
cleaner, higher-bitrate re-encode. **Mist's error below the music** does not depend on that: it compares the
stego copy with the clean one, so the only difference left is what Mist changed, measured against the music
itself. The **per carrier** table lists every track with its output bitrate, so one odd track cannot hide in the
mean.

### Thresholds

| Measure | ✅ | ⚠️ | ❌ |
|---|---|---|---|
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

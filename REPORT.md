# Mist harness report

perceptual metric: not installed

Every carrier is re-encoded twice: once plainly (the *clean* copy) and once with a hidden message (the *stego* copy). The report asks whether a detector can tell the two apart, and how much worse the stego copy sounds. [How to read this report](#how-to-read-this-report) explains every number and threshold.

## Summary

| Format | Carriers | Hidden from detectors? | Audio quality |
|---|---|---|---|
| ogg/vorbis | 13 / 13 | ✅ chance on all 4 detectors | ❌ embedding costs 1.37 dB (+37% error), over the 0.3 dB target |
| flac | 13 / 13 | ✅ chance on all 4 detectors | ✅ inaudible: the added error is at least 81.24 dB below the music |

## ogg/vorbis

Measured on 13 of 13 carriers.

### Can a detector tell?

| Detector | Looks for | AUC | 95% interval | Verdict | Message size |
|---|---|---|---|---|---|
| chi-square | bit overwriting (not what Mist does) | 0.500 | 0.499–0.501 | ✅ chance | ✅ hidden (0.500) |
| spa | bit overwriting (not what Mist does) | 0.500 | 0.500–0.500 | ✅ chance | ✅ hidden (0.500) |
| rs | bit overwriting (not what Mist does) | 0.500 | 0.499–0.501 | ✅ chance | ✅ hidden (0.500) |
| hcf-com | ±1 changes, which is what Mist does | 0.517 | 0.500–0.535 | ✅ chance | ✅ hidden (0.500) |

**Verdict:** ✅ chance on all 4 detectors.

### How much does it cost in sound?

| Measure | Mean | Worst | What it means |
|---|---|---|---|
| Plain re-encode SDR | 35.63 dB | 34.20 dB | Loss any re-encode causes, Mist or not |
| Mist output SDR | 34.26 dB | 33.16 dB | The same, with the message embedded |
| Embedding cost | 1.37 dB | 1.98 dB | Mist's own share; target ≤ 0.3 dB, design-doc reference 0.18 dB |
| Extra error energy | +37% | +58% | Embedding cost as error added on top of a plain re-encode |

**Verdict:** ❌ embedding costs 1.37 dB (+37% error), over the 0.3 dB target.

## flac

Measured on 13 of 13 carriers.

### Can a detector tell?

| Detector | Looks for | AUC | 95% interval | Verdict | Message size |
|---|---|---|---|---|---|
| chi-square | bit overwriting (not what Mist does) | 0.500 | 0.497–0.503 | ✅ chance | ✅ hidden (0.500) |
| spa | bit overwriting (not what Mist does) | 0.503 | 0.494–0.512 | ✅ chance | ✅ hidden (0.503) |
| rs | bit overwriting (not what Mist does) | 0.498 | 0.490–0.509 | ✅ chance | ✅ hidden (0.501) |
| hcf-com | ±1 changes, which is what Mist does | 0.500 | 0.490–0.511 | ✅ chance | ✅ hidden (0.500) |

**Verdict:** ✅ chance on all 4 detectors.

### How much does it cost in sound?

| Measure | Mean | Worst | What it means |
|---|---|---|---|
| Plain re-encode SDR | 85.98 dB | 81.73 dB | Loss any re-encode causes, Mist or not |
| Mist output SDR | 85.49 dB | 81.24 dB | The same, with the message embedded |
| Embedding cost | 0.49 dB | 0.49 dB | Mist's own share; target ≤ 0.3 dB, design-doc reference 0.18 dB |
| Extra error energy | +12% | +12% | Embedding cost as error added on top of a plain re-encode |

**Verdict:** ✅ inaudible: the added error is at least 81.24 dB below the music.

## How to read this report

### Detectability

**AUC** is the chance that a detector, shown one clean and one stego sample, picks the stego one. 0.5 is a coin
flip, which is the goal; 1.0 means it is caught every time. A value well below 0.5 is a detection too: the
detector is right, just with its sign flipped. Each carrier is cut into chunks of 65,536 values and every chunk
is scored, so the **95% interval** says how sure the estimate is. Ogg Vorbis is scored on the residues Mist may
change, lossless formats on the decoded samples.

Chi-square, SPA and RS look for bits being overwritten, which Mist never does, so they are expected to sit at
chance; a rise means the embedder has drifted. HCF-COM looks for ±1 changes, which is exactly what Mist does,
so it is the detector that matters.

**Message size** compares a 64-byte message with a 1-byte one. Mist changes the same amount of audio whatever
the message, so this should read 0.5: anything else means the message length shows.

### Audio quality

**SDR** (signal-to-distortion ratio) is how loud the music is compared with the error added to it, in dB. Higher
is better, and every 10 dB means ten times less error. Both copies are compared with the original carrier,
after lining them up in time. The **plain re-encode** is what any re-encode costs; the **embedding cost** is how
much lower the stego copy scores, which is Mist's own share. **Extra error energy** says the same thing as a
percentage: +37% means 37% more error than the plain re-encode alone.

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

# Phase 4: freeze stronger wardens

Phase 4 freezes the wardens an embedder change will be scored against. It
does not change the embedder, the payload, or a container. A file embedded
before this phase extracts the same way, and a new file is byte-for-byte
the same kind of output Phase 3 already produced.

`REPORT.md` was not re-run. It remains the metal-dev measurement from
before these wardens existed (schema 3 at the time, corpus name
`metal-dev`, first 300 seconds, 84 lineages, commit `b610b12-dirty`).
That report's Holm family has four members. Schema 6 has six. The two
Holm columns are not comparable number for number.

## What was frozen

The confirmatory Holm family, in `steganalysis.FrozenPrimaries`:

1. `hcf-com` — ±1 / LSB matching.
2. `classifier` — nested logistic on detector scores, the local histogram and first-order transitions.
3. `markov` — nested logistic on second-difference transitions.
4. `key-aware` — whether the first frame's 32-byte representative looks like a raw X25519 public key.
5. `rich` — nested logistic on the new summary vector.
6. `selection` — LSB-rate gap at positions implied by the recipient public key.

Chi-square, SPA and RS stay exploratory (Benjamini-Hochberg). Nine full
refits remain the permutation for the three trained models, so their
smallest p-value is still 0.1. The other three use the 199-draw paired
lineage swap.

The rich vector is a summary, not a full spatial rich model, so a harness
chunk stays trainable. It holds prediction errors of orders 1–8,
differences of orders 1–4, an order-8 LPC residual energy, decimation by
2 (both phases) and by 4, parity-conditioned differences, the LSB mean,
order-2 and order-3 co-occurrence symmetrized under sign flip and time
reversal, entropy and peak of an order-4 co-occurrence, spectral flatness,
eight log-spaced bands, a wrapped phase difference and a group-delay
proxy, and mid/side when the caller passes a channel count. Log-spaced
bands stand in for a mel or constant-Q filterbank. They are not one.

Vorbis packet, partition, codebook, band and Huffman conditioning is not
in this vector. The harness flattens residues to `[]int32` before the
analysis packages see them, and those packages must not import `av` or
`vorbis`. Claiming that conditioning without the structure would be a
false warden. A later export can add it; this phase does not pretend it
is there.

`TrainFLD`, `TrainStumps` and `TrainSubspace` fit the same vector. They
are package baselines with their own positive control. They are not extra
Holm rows: three models on one vector would be one question counted three
times. The report's `rich` row is the nested logistic, the same protocol
as Markov.

## Selection channel and the oracle

`selection` uses `crypto.PositionSeed(pub, 0)` and `stego.Covered`, which
is the same count the embedder uses at `stego.Density`. The first harness
chunk stands in for frame 0. The harness value stream is not cut on the
silence mask Embed uses, so this score is an operational approximation.
It does not receive sender-only costs or true span boundaries.

`ChangedFraction` compares cover and stego sample by sample. That needs
the cover. It is named in the manifest as `oracle` and it is not a row
in the detector table. It bounds how concentrated a later cost function
could become. It is not a realistic attacker.

## Key-aware cases

Current envelopes carry an Elligator representative, so `HonestX25519`
is expected to reject them. The control that the warden still works is a
real X25519 public key, which it accepts. Added checks, without a corpus:

- 64 seals to one recipient are pairwise distinct representatives, and the top bit is not stuck clear. A representative interpreted as a public key passes `HonestX25519` only at the rate of a random string (about 1 in 32), so the test allows a handful of hits and rejects a rate that would make the check a presence test.
- `Open` with a different private key fails.
- `Open` with the matching private key succeeds.
- Position seeds for frame 0 and frame 1 differ.
- Seals to two recipients differ.
- A payload that carries an Ed25519 signature still does not put a raw public key in the envelope.
- Span-start and span-continue magic bytes are outside the payload version.

## CNN tooling

`tools/cnn_warden/train.py` keeps the waveform net as the default, so
`make cnn-warden` does not get several times slower. `--arch all` also
trains a spectrogram net, a convolution-plus-attention hybrid, and the
same waveform net under the name `residue` (the integers in a Vorbis
export are already residue indices). Extra architectures are written as
`held_out` and printed beside the primary row. The loudest one is not
promoted. `--stereo` scores even/odd samples as left, right, mid and
side; it is off by default because a Vorbis export is not interleaved
PCM. `--self-test` trains the waveform net on synthetic LSB replacement
and exits non-zero below AUC 0.8. On this machine that control scored
1.000 (16 synthetic carriers, not Mist output).

wav2vec2, AST, SSAST and BEATs are not in the frozen suite. No positive
control was run, and a transformer is not assumed to be a stronger
warden than the nets that did pass a control.

## Positive controls

These live in `internal/steganalysis` and run under `go test`, plus the
CNN self-test when torch imports. Each control is a case the warden is
supposed to catch. Passing it is not a measurement of Mist.

| Control | Warden | Carrier |
|---|---|---|
| LSB replacement at rate 1 | classifier on `Features` | peaked, short |
| LSB matching at rate 1 | Markov, rich logistic, FLD, stumps, subspace | amplitude-8 sine plus unit noise |
| LSB matching at rate 0.5 | HCF-COM | the existing peaked fixture |
| random ±1 index neighbor | HCF-COM | peaked indices |
| raw X25519 public key | `HonestX25519` | `ecdh` public bytes; `0xff`×32 is rejected |
| planted LSBs at listed positions | `ParityGap` | smooth series |
| clear u32 length below 2^20 | `PlausibleLength` | four bytes |
| constant tail, varying head | `DeadTail` | no-filler stand-in |
| constant file-size gap | logistic on one feature | metadata stand-in |
| cover vs stego sample gap | `ChangedFraction` | oracle, about a quarter of samples at rate 0.5 |

The matching control uses a low-amplitude carrier on purpose. ±1 is
invisible to these summaries inside an amplitude-800 sine, and a control
the feature cannot see would only prove the test was unfair. Music-scale
audio is what the harness measures later. The unit control only shows
the warden is not dead.

## Wire profile

Unchanged. No edit under `internal/crypto`, `internal/wire`, `internal/stego`
beyond exporting `Covered` (the density count the warden must share with
the embedder), or `internal/av`. Rollback is deleting the new warden
files and the harness rows that name them. Embedded files do not need
to be regenerated.

## What this is not

This phase does not hide a message. Hiding starts when the sender's
costs change (Phase 5, PR sequence step 8). The frozen suite is enough
to start that code: the wardens, the folds and the positive controls
exist, and an embedder edit can be scored against them.

It is not enough to claim the embedding got harder to detect. That claim
needs a paired harness on corpus name `metal-dev`, the same 300 second
cap and the same 84 lineages, run after the Phase 3 container changes,
with schema 6, before and after the embedder edit. The synthetic checks
in earlier phases are not that baseline. The Phase 2 power count of
about 128 lineages is still unmet, and the blind red-team (PR step 16)
stays open.

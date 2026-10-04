# Steganalysis techniques

This package holds the warden's toolkit: detectors that score a stream of quantized
values for traces of LSB embedding, and the AUC that says how well a detector
separates stego output from clean output. The detectors see only `[]int32`, so
they work for every output format Mist can write. The harness hands them whichever
values that format embeds in:

| Output | Embedding domain | Values the detectors see |
|---|---|---|
| Ogg Vorbis | quantized residues | residue codebook entry indices inside `DefaultBands` (≥ 6 kHz) |
| Any lossless codec FFmpeg can write (FLAC, WAV, ALAC, WavPack, TTA, AIFF, CAF, …) | PCM sample LSBs | decoded samples on the encoder's integer grid (`av.SampleScale`), interleaved |

Every detector returns a score in `[0, 1]`. Higher means the stream is more likely
to carry data. A detector with nothing to measure, such as an empty or constant
stream, returns `0`.

| Detector | Targets | Score | Best on |
|---|---|---|---|
| [`ChiSquare`](#pairs-of-values-chi-square--westfeld--pfitzmann) | LSB replacement, high rate | p-value that the 2k / 2k+1 pairs are equalised | peaked histograms |
| [`SPA`](#sample-pair-analysis--dumitrescu-wu--wang) | LSB replacement, any rate | estimated embedding rate p | smooth, correlated signals |
| [`RS`](#rs-analysis--fridrich-goljan--du) | LSB replacement, any rate | estimated embedding rate p | smooth, correlated signals |
| [`HCF`](#hcf-centre-of-mass--harmsen--pearlman) | ±1 embedding (LSB matching) | 1 − normalised HCF centre of mass | peaked histograms |
| [`CrossValidate`](#logistic-classifier--the-adversary-of-record) | anything learnable from Mist's own output | out-of-fold probability of stego | whatever the training data holds |

**Why the first three matter to Mist even though it does not use LSB replacement.**
Mist embeds by LSB *matching* (±1). On lossless outputs that is literal ±1 on
the sample. On Vorbis it substitutes the nearest same-length codebook entry with
the right parity, which changes the index by an amount that is not always ±1. Chi-square, SPA and RS
model LSB *replacement* (`x → x ^ 1`). They are expected to sit at chance against
Mist. If one of them climbs above chance, the embedder has drifted towards
replacement. HCF, together with the trained classifier, is the detector that
actually targets what Mist does.

Throughout, *rate* p is the fraction of values that carry a message bit. Under
replacement, half of those values happen to hold the right bit already, so a
fraction p/2 of all LSBs is actually flipped.

---

## Pairs-of-values chi-square — Westfeld & Pfitzmann

**Idea.** LSB replacement maps 2k and 2k+1 onto each other. Once enough bits are
written, the two counts in each "pair of values" converge to their mean. Natural
histograms, especially peaked ones, rarely have that property.

**Statistic.** Histogram the values into pairs keyed by `x >> 1`, so negative
values pair up correctly in two's complement. For each pair whose expected count
is at least 5:

$$
e_k = \frac{h_{2k} + h_{2k+1}}{2}, \qquad
\chi^2 = \sum_k \frac{(h_{2k} - e_k)^2}{e_k}, \qquad
\text{score} = 1 - P\!\left(\tfrac{\text{df}}{2}, \tfrac{\chi^2}{2}\right)
$$

`df` is the number of retained pairs minus one, and `P` is the regularised lower
incomplete gamma function, i.e. the chi-square CDF. A score near 1 means the pairs
look equalised, which is the embedding signature.

**Limits.** It detects only near-full, sequential replacement; random partial
embedding at a few percent leaves it at 0. On wide, flat histograms adjacent bins
are already nearly equal, so clean data scores near 1. That is a false positive
inherent to the test, so read it only through an AUC against matched clean
carriers.

> A. Westfeld, A. Pfitzmann. *Attacks on Steganographic Systems.* Information
> Hiding 1999, LNCS 1768.

---

## Sample pair analysis — Dumitrescu, Wu & Wang

**Idea.** Look at adjacent pairs `(u, v)`. In a natural signal, odd-difference
pairs split evenly between "the smaller value is odd" (set X) and "the smaller
value is even" (set Y). LSB replacement breaks that balance in a precisely
predictable way. Pairs with `u = v` turn into `(2k, 2k+1)` pairs, which land in Y.
Solving for the rate that would produce the observed imbalance estimates p.

**Sets.** Over all adjacent pairs:

- **C₀**: pairs with `u >> 1 == v >> 1` (same pair of values). LSB flips cannot
  move a pair out of this set.
- **D₀**: pairs with `u == v`.
- **X**: pairs with an odd difference and an odd smaller value.
- **Y**: pairs with an odd difference and an even smaller value.

**Estimator.** Assume |X| = |Y| in the cover. With `s = |Y| − |X|` measured on
the stego stream, the expected counts under flip probability p/2 give

$$
\frac{|C_0|}{2}\,p^2 \;-\; \bigl(|D_0| + s\bigr)\,p \;+\; s \;=\; 0
$$

and the score is the smaller root, clamped to `[0, 1]`. This is Ker's
trace-set form, summed over every difference class m, re-derived here; p = 1
is a root in expectation, since a fully randomised stream has |D₀| ≈ |C₀|/2. For small p it reduces to `p ≈ s / |D₀|`.
When sampling noise makes the discriminant negative, near p = 1, the real part
of the complex roots is used, following Ker.

**Limits.** It needs correlated neighbours, so it is noisy on independent, peaked
data. It is blind to ±1 matching because matching moves values both ways and keeps
|X| ≈ |Y|.

> S. Dumitrescu, X. Wu, Z. Wang. *Detection of LSB Steganography via Sample Pair
> Analysis.* IEEE Trans. Signal Processing 51(7), 2003.
> A. D. Ker. *A General Framework for Structural Steganalysis of LSB
> Replacement.* Information Hiding 2005.

---

## RS analysis — Fridrich, Goljan & Du

**Idea.** Split the stream into groups of 4 and measure each group's roughness
`f(G) = Σ |gᵢ₊₁ − gᵢ|`. Apply a flipping mask to the group and compare
roughness before and after. If the group gets rougher it is *Regular*; if it gets
smoother it is *Singular*. In a smooth cover, flipping usually adds noise, so
Regular groups outnumber Singular ones. LSB replacement erodes that gap for the
positive flip but not for the shifted flip, and the asymmetry measures p.

**Flips.** `F₁: x → x ^ 1` pairs (2k, 2k+1). `F₋₁: x → ((x + 1) ^ 1) − 1` pairs
(2k−1, 2k). The mask is `M = [0, 1, 1, 0]`, applied as `+M` for F₁ and as `−M`
for F₋₁.

**Estimator.** Let `d = R − S` be the regular-minus-singular fraction. Measure it
four times: on the stream (`d₀`, `d₋₀`) and on the stream with every LSB
inverted (`d₁`, `d₋₁`), each under `+M` and `−M` respectively. Then solve

$$
2(d_1 + d_0)\,z^2 + (d_{-0} - d_{-1} - d_1 - 3d_0)\,z + d_0 - d_{-0} = 0,
\qquad p = \frac{z}{z - \tfrac12}
$$

taking the root `z` of smaller magnitude. The score is p clamped to `[0, 1]`, with
the same real-part rule as SPA.

**Limits.** It rests on the same smoothness assumption as SPA and underestimates
on independent, peaked data. It is blind to ±1 matching.

> J. Fridrich, M. Goljan, R. Du. *Reliable Detection of LSB Steganography in
> Color and Grayscale Images.* ACM Workshop on Multimedia and Security, 2001.

---

## HCF centre of mass — Harmsen & Pearlman

**Idea.** ±1 embedding adds independent noise to every value it touches. The
stego histogram is therefore the cover histogram *convolved* with the noise
distribution `[p/4, 1 − p/2, p/4]`. In the Fourier domain, convolution is
multiplication by a low-pass response. The histogram characteristic function
(HCF), the DFT of the histogram, loses energy at high frequencies, and its centre
of mass shifts towards zero.

**Statistic.** Histogram the values over `[min, max]` and zero-pad to a power of
two N. Take the FFT H. Then

$$
\text{COM} = \frac{\sum_{k=1}^{N/2} k\,|H[k]|}{\sum_{k=1}^{N/2} |H[k]|},
\qquad \text{score} = 1 - \frac{\text{COM}}{N/2}
$$

so a lower COM, the embedding signature, gives a higher score.

**Limits.** A stream spanning 2^20 or more distinct values (a wide 24-bit
sample grid, or a single outlier) is not scored and returns `0`, which bounds the
FFT at 16 MiB. The histogram is zero-padded to a power of two, so the score shifts
slightly when the value range crosses one, by about 5·10⁻³ in the tests; compare it
only between carriers of similar range. The absolute COM depends heavily on the content. It is meaningful
only in comparison with a matched clean carrier, which is exactly how the harness
uses it, through AUC. The effect is strongest on peaked histograms, such as
Vorbis residue indices, and weak on wide ones, such as 16-bit PCM from a lossless
output. Ker's calibrated variant, which compares against a downsampled copy, is
the known improvement if this stays weak.

> J. Harmsen, W. Pearlman. *Steganalysis of Additive Noise Modelable Information
> Hiding.* Proc. SPIE 5020, 2003.
> A. D. Ker. *Steganalysis of LSB Matching in Grayscale Images.* IEEE Signal
> Processing Letters 12(6), 2005.

---

## Logistic classifier — the adversary of record

**Idea.** A fixed detector tests one hypothesis about what embedding changes. A
classifier trained on Mist's own clean and stego output learns whatever does
change, including traces nobody thought to test for. The roadmap treats it as
the adversary of record, so its AUC is the one the exit criterion is judged by.

**Features.** `Features` turns a chunk into one vector:

- every classical detector's score, in `Detectors` order;
- the share of values at each of −3…3, because ±1 embedding flattens a peaked
  histogram;
- first-order SPAM transitions. Each difference between adjacent values is
  truncated to ±3, giving 7 bins, and the features are the 7 × 7 probabilities
  of one bin given the bin before it. ±1 noise blurs how one step follows the
  next. (Pevný, Bas and Fridrich.)

**Model.** L2-regularised logistic regression on z-scored features, trained by
full-batch gradient descent for a fixed number of steps. Training is
deterministic.

**Validation.** `CrossValidate` splits samples into folds **by group**; the
harness uses the carrier as the group. Every carrier is scored by a model trained
without it, so the classifier cannot win by recognising a track it has already
seen. The out-of-fold scores then go through `AUC` and `AUCInterval` like any
detector's.

**Limits.** A linear model on 60 hand-built features is a modest adversary. It
matches HCF on synthetic ±1 embedding and beats it at higher rates, but a deep
model on richer features would be stronger. With few carriers, its AUC is noisier
than the classical detectors'.

> T. Pevný, P. Bas, J. Fridrich. *Steganalysis by Subtractive Pixel Adjacency
> Matrix.* IEEE Trans. Information Forensics and Security 5(2), 2010.

---

## AUC and its interval

**AUC** is the probability that a randomly chosen stego score exceeds a randomly
chosen clean score, with ties counting half. It is computed exactly from the
Mann-Whitney U statistic with average ranks for ties:

$$
\text{AUC} = \frac{R_+ - n_+(n_+ + 1)/2}{n_+\,n_-}
$$

`0.5` is chance, `1` is a perfect detector, and `0` is a perfect detector with
its sign flipped. A value well below 0.5 is still a detection. For Mist the goal
is **AUC ≈ 0.5 for every detector**.

**`AUCInterval`** is a percentile cluster bootstrap. Every score carries a group,
and the harness uses the carrier as the group. Each round redraws whole groups
with replacement, taking every positive and negative score of each, recomputes
the AUC, and reports the 2.5th and 97.5th percentiles. Chunks of one track are
correlated, and clean and stego chunks of one track are paired, so resampling
chunks one by one would make the interval far too narrow: its width would follow
the number of chunks rather than the number of tracks. It is deterministic for a
given seed, so reports can be compared run to run. With fewer than one round it
returns the AUC itself as both bounds.

**From AUC to ε.** Cachin calls a scheme ε-secure when the relative entropy
D(P_C ‖ P_S) between clean and stego files is at most ε. Any detector's
|AUC − ½| is at most the total variation δ between the two, and Pinsker's
inequality gives δ ≤ √(ε/2), so a measured AUC proves ε ≥ 2(AUC − ½)² nats. This
is only ever a lower bound: an AUC at chance says the detector found nothing, not
that ε is small.

> J. A. Hanley, B. J. McNeil. *The Meaning and Use of the Area under a Receiver
> Operating Characteristic (ROC) Curve.* Radiology 143, 1982.
> B. Efron, R. Tibshirani. *An Introduction to the Bootstrap.* Chapman & Hall, 1993.
> C. Cachin. *An Information-Theoretic Model for Steganography.* Information and
> Computation 192(1), 2004.

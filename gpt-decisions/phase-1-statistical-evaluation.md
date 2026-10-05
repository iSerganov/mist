# Phase 1: correct the statistical evaluation

Phase 1 changes how a harness run scores and reports detectors. It does not
change the embedder, the wire format, the density, the syndrome-trellis code,
or any sender cost. A file embedded before this phase extracts exactly as it
did. No new detectability number is claimed: this document does not include a
corpus run.

The work follows `GPT-SOL-D-PLAN.md` §7 and recommended PR sequence step 2.
Phase 0 (step 1) was already in place: public corpus ids, a run manifest, and
raw scores. This phase consumes that lineage field.

## What “independent” means here

A chunk is not an independent draw. Chunks of one recording share the source,
the encoder, and the embedded changes. Recordings cut from one session share
those too: chapters, a second payload, a second key, or a transcode of the
same take. Treating them as separate carriers makes a bootstrap interval and a
cross-validation fold look more precise than the corpus is.

The independent unit is the lineage recorded in the corpus manifest. The
harness assigns each distinct lineage string a dense integer as carriers are
measured. A blank lineage uses the public carrier id, so two carriers that
omitted the field do not collapse into one family by accident.

An anonymous directory walk, the path with no manifest, still gives each file
its own lineage (`carrier-NNNN`, the same id Phase 0 already uses). That is
anti-conservative when several files in one folder are chapters of one
session: the interval will be too narrow. The manifest is how a run says they
belong together. This phase does not try to discover that from the audio.

Acoustic near-duplicate detection was considered and left out. A fingerprint
would invent groups the corpus author did not declare, and a false merge
changes every interval and every fold. Step 1.1’s “detect near-duplicates”
waits until a corpus actually needs it. Until then the declared lineage is the
strongest group the harness will use.

## Nested grouped evaluation

`CrossValidate` is unchanged: folds by group, fixed L2 penalty `1e-2`,
standardisation fit on the training rows of that fold. `TrainLogistic` now
calls `TrainLogisticPenalty` with that same penalty, so existing classifier
tests keep their model.

`NestedCrossValidate` is what the harness reports.

- The outer split is `assignFolds` on the lineage id. Every chunk of every
  recording in a lineage stays in one fold. The reported score for a chunk is
  produced by a model that did not train on that lineage.
- On the training lineages, an inner grouped cross-validation picks the L2
  penalty from `1e-3`, `1e-2`, `1e-1` by log loss. The grid is
  `NestedPenaltyGrid` and is copied into `manifest.json`, so a report names
  the search that produced it.
- One further training lineage, the last id in sorted order, is held out to
  fit a Platt line `sigmoid(a + b · logit(p))`. The slope is floored at
  `1e-3`. A negative or zero slope would reverse or erase the score the
  classifier just learned. The floor keeps calibration from doing that.
- Fewer than four distinct lineages cannot spare an inner search and a
  calibration holdout. The call falls back to `CrossValidate` at the protocol
  penalty. Two lineages, which is what a tiny synthetic corpus often is, stay
  on the old path.
- If a fold’s training rows are empty or single-class, that fold is scored
  `0.5` rather than fitting a model that has nothing to separate.

The same nested call scores the classifier and the Markov model everywhere
they appear: the primary table, the embedding-only table, scaling prefixes,
and per-category rows. Those tables are not a different model with a secretly
fixed penalty. What they do not repeat is the confirmatory refit below.

Stored score groups remain the recording index. `perFile` still averages
chunks of one recording into one file score. Two chapters of one session stay
two files and share a family id. Merging them would hide a chapter that
separates and a chapter that does not.

## Confirmatory tests, and what is exploratory

Two permutation procedures, both swapping labels of a whole lineage so a
recording’s stego score and its clean score move together.

**Score permutation.** `PairedPermutationP` swaps the already computed file
scores. 199 permutations plus the observed labelling, so the p-value is
`k / 200` and cannot be zero. The smallest value is `0.005`. This is cheap.
It is used for every classical detector and for the key-aware row on the
primary operational comparison (stego against the clean ffmpeg copy).

**Refit permutation.** `RefitPermutationP` repeats `NestedCrossValidate` after
the same kind of swap. Feature rows are unchanged; only the labels move, which
is the null that the model’s scores do not depend on which copy is stego.
This runs only for the classifier and the Markov model, only on that primary
operational comparison, and only for 9 permutations. Scaling, category,
embedding-only, and pooled rows do not refit.

Nine was chosen because each refit retrains the nested model on every chunk.
On a real corpus that is minutes per format, which is the cost this phase
accepted. It has a resolution limit that the report legend states: the
smallest attainable p-value is `1/10 = 0.1`. A classifier row cannot, by
itself, clear 0.05. Raising the count is a later decision once a corpus run
shows the refit is worth the time. The test is still the right null — the
pipeline refits — it is just coarse.

The observed labelling is counted in both procedures. A perfect separation
that survives every shuffle still has a positive p-value.

**Holm** adjusts the confirmatory family so its family-wise error stays
controlled: `hcf-com`, `classifier`, `markov`, `key-aware`. HCF-COM is in that
family because it is the classical detector aimed at ±1 changes. The trained
models are in it because they are the adversaries of record. The key-aware
row is in it because a public-key structure leak is a presence test, not an
exploratory search.

**Benjamini-Hochberg** adjusts the exploratory three: `chi-square`, `spa`,
`rs`. They model LSB replacement, which Mist does not do. They stay in the
report so a drift toward replacement is visible, and they are not asked to
share the confirmatory error budget.

Any other row leaves both adjusted p-values at `-1`, which the markdown
prints as a dash. That includes every detector on the scaling, category, and
embedding-only tables. A dash means “this row was not in the confirmatory
test”, not “p = 0”.

Comparing two detector versions with a paired cluster interval was not added.
The report already prints a point AUC change against a baseline report. A
paired interval needs both runs’ file scores aligned by carrier. `scores.json`
now keeps those scores, so the comparison can be built when the first embedder
change is measured against a frozen baseline. Doing it inside a phase that
does not change the embedder would have nothing to compare.

## Hierarchical interval

The primary file interval is `HierarchicalInterval`. Each round draws lineages
with replacement, then draws that lineage’s recordings with replacement, and
scores the result with the same pair weights as `AUCInterval`: a recording
drawn once contributes its paired comparison once, and between-recording pairs
are scaled by `n/(n−1)`.

When every lineage is a single recording, the inner draw is skipped and the
random sequence is the one `AUCInterval` uses for the same seed. The two
intervals are then equal, not merely close. A unit test locks that.

The chunk interval is a lineage-cluster `AUCInterval`. It is not a bootstrap
of chunks one by one, and it is not a per-recording cluster once several
recordings share a lineage. Segment-level confidence intervals are not
reported. That was a Phase 1 gate, and the chunk column would have been the
place they leaked back in.

The recording-cluster file interval is still computed and stored
(`record_auc_lo`, `record_auc_hi`). The markdown prints it only when its width
differs from the lineage interval by more than 0.01. With one recording per
lineage the widths match and the note stays absent. When chapters share a
lineage, the lineage interval is the one the verdict uses; the recording
interval is there so a reader can see how much independence was being assumed.

One limitation is deliberate. The `n/(n−1)` scale keeps using the number of
recordings, including in the hierarchical rounds. Changing it to the number of
families would have made the one-recording case disagree with `AUCInterval`.
The hierarchy changes which recordings are drawn together. It does not
re-derive that scale.

## Detectability

`Detectability` is D = 0.5 + |AUC − 0.5|. AUC 0.1 and AUC 0.9 are the same
detection with opposite signs. The file-AUC column stays signed, so the
direction is visible. D is the column that does not reward a reversed
detector.

The verdict already treated an interval lying entirely below 0.5 as detection.
The KL lower bound was already `2(AUC − ½)²`, which does not depend on sign.
D is the missing orientation-aware summary, not a second verdict rule.

If the file interval contains 0.5, the KL lower bound stays 0, as in Phase 0.
That bound remains a detector-implied lower bound on the benchmark divergence.
It is not an estimate of Cachin’s ε and it is not evidence that the scheme is
undetectable.

## Power

`FamiliesForPower` answers a planning question: how many independent paired
lineages would this run’s classifier dispersion need before a shift to D = 0.55
pushed a 95% lineage-cluster interval off 0.5 in 90% of simulations.

The procedure:

1. Centre each pilot pair so the mean stego-minus-clean difference is zero.
2. Binary-search a common shift until the pilot AUC is within 0.03 of 0.55.
   The tolerance is 0.03 because a handful of pairs moves AUC in steps of
   `1/(n²)`; 0.02 is smaller than some of those steps.
3. If no shift lands that close, return 0 and “not reached”. A constant pilot
   jumps from AUC 0.5 to AUC 1 and cannot represent D = 0.55. The report then
   omits the power sentence.
4. Otherwise resample the shifted pairs at n = 8, 16, …, 256. Each size gets
   30 simulations. Each simulation’s interval is `AUCInterval` with one family
   per pair and 40 bootstrap rounds, which is the hierarchical interval in the
   one-recording case.
5. The smallest n whose intervals exclude 0.5 at least 90% of the time is the
   answer. If none do, the report says the search stopped at 256.

The target 0.55 is the engineering effect named in the plan, not a security
claim. The result is a simulation from this run’s file scores. It is not a
guarantee about a future corpus, and it is not a proof that the current corpus
is powered. Seeds are the harness seed, recorded in the manifest next to the
target and the 0.9 power.

## Worst cell and leave-one-lineage

Aggregate AUC is how a separated category disappears. After the per-category
table, the report names the cell with the largest D: category, detector, file
AUC, D. The scan covers every detector in every category, not only the
classifier. A corpus with one category does not build that table
(`byCategory` returns nothing below two categories); the worst cell is then
the aggregate detector with the largest D, labeled `aggregate`.

Leave-one-lineage does not retrain. It restricts the classifier’s file scores,
which are already out of fold, to each lineage that has at least two carriers,
and it reports the AUC of the carriers that remain when at least two remain.
A lineage of one carrier is omitted. One pair has AUC 0, 0.5, or 1 and reads
like a finding when it is a coin flip. The model is not refit without that
lineage: with five folds a lineage shares its fold with others, so these
scores are out-of-fold for the lineage but not a pure leave-one-lineage
training run. The table says so. A true refit per held-out lineage would
multiply the nested training by the number of lineages and was not accepted
for this phase.

## CNN folds

`tools/cnn_warden/train.py` assigns folds by the `lineage` field in the export
JSON. Every recording of one lineage stays in one fold. A single lineage
cannot be split, so the script falls back to a shuffled carrier fold, which is
what it did before. The CNN’s own interval is still a bootstrap over files,
not `HierarchicalInterval`. The plan’s Phase 1 change for that tool was the
fold. Moving its interval to the hierarchical bootstrap belongs with the pass
that makes the CNN a primary row.

## Manifest

Report schema is 3. Schema 2 remains the Phase 0 document: public ids, raw
scores, and a classifier described as carrier-grouped. Schema 3 adds the
fields a reader needs to see that the evaluation moved:

- `score_unit`: chunks grouped by lineage; files stay one per recording
- `classifier`: nested lineage-grouped CV, z-scored on the training fold
- `permutation_rounds`: 199
- `refit_rounds`: 9
- `nested_penalties`: the grid above
- `power_target_d`: 0.55
- `power`: 0.9

`seed` is still the harness seed (1). Bootstrap, both permutations, and the
power simulation use it. Raw scores still carry the public carrier, category,
and lineage. Nothing in the manifest is a local path.

## Tests

`go test ./internal/steganalysis` covers:

- D folds 0.1 and 0.9 onto 0.9
- a one-recording-per-family hierarchical interval equals `AUCInterval` for the
  same seed
- separated families pin the hierarchical interval at 1
- Holm’s step-down and Benjamini-Hochberg’s step-up on a hand-computed triple
- a paired permutation is 1 when the two populations match, and small when
  twelve families are perfectly separated
- nested CV matches `CrossValidate` below four groups, is deterministic, and
  separates eight held-out groups
- a strong shift is reachable at the power target used in that test (0.9, so
  the simulation can finish on a handful of families); a constant pilot is not

`go test -tags harness` on the new suite methods covers the wiring that does
not encode audio: a shared family pins the file interval, Holm is applied to a
confirmatory permutation, the classifier keeps one row per recording inside a
lineage, and leave-one-lineage ignores a lineage of one carrier. The existing
raw-score test still checks public ids. `make harness` was not run. There is
no new AUC to quote.

## What this does not show

A lineage-aware interval that contains 0.5 means this detector, on this
corpus, did not separate the copies beyond chance. It does not mean the KL
divergence is small, and it does not mean a warden with the original carrier
would fail. Known-cover detection and transcoding survival stay non-goals.
The next phases are the ones that can change the output file.

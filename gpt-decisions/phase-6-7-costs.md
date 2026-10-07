# Phases 6 and 7: sender-only costs

Phase 5's planner split was skipped. `Cost` and `Flip` were already the
sender-only half of the carrier interface, and `Len` / `At` were already
the receiver-visible half. A second interface would not change a bit.

## What changed

Lossless position cost is the growth of the first and second difference
when the sample steps by one, divided by the neighbourhood's own mean
absolute difference. A small residual is a tone, so the same ±1 costs
more there than in noise. Stereo disturbance, distance to the clip, and
distance to a silent run are added on top. None of those terms add or
remove a sample from the carrier.

The price then passes through a floor (`0.25`) and a ratio cap (`8`),
and a ±5% dither from the position key. A free sample would be a
selection channel. A cost that is a pure function of the waveform would
be one too, for a warden who knows the rule.

Direction stays the histogram draw, √h(v+1) against √h(v−1), with each
side discounted by its residual. Taking the minimum residual every time
walked a Laplace frame toward its peak and shrank Σv² by several units
per change. The discount is mild (`residual / 32`) so a close call still
follows the histogram. The drift bound in the sample suite is 0.75 per
change, under the +1 an unsteered ±1 adds.

Vorbis still refuses a symbol with no same-length, opposite-parity
codeword. Leaving those symbols in the carrier would change `slots` and
the keyed order, and files already embedded would not extract. Among
legal symbols, `RankCost` is the squared vector distance divided by the
entry's energy, times `1 + 1/n` for `n` legal substitutes. The trellis
ranks that. `FlipCost` stays the raw distance, and a distance past
`maxFlipCost` is still wet. The substitute applied at rewrite is still
the nearest legal vector: every candidate of one symbol shares the
energy and the neighbourhood, so the rank does not change which of them
wins. The parsed packet is cached on the codec and dropped before the
programs are rewritten, so the rewrite does not decode the packet a
second time and a later read cannot see the substitutes.

## What this does not say

No metal-dev harness was run after the change. There is no new file AUC,
and this is not a claim that any warden got weaker. The unit tests check
the mechanism: noise is cheaper than a smooth ramp, the cost ratio stays
inside the cap, the dither follows the key, the eligible index list is
unchanged by embedding, a step that shrinks the residual outranks one
that grows it, a louder codebook entry ranks cheaper than a quiet one
with a worse vector, and an identity Vorbis rewrite still round-trips.
Extraction of older files depends only on the unchanged position list.

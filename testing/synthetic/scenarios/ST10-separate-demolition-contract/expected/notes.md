# ST10-separate-demolition-contract notes

The ST10 base with one change: the owners engage a licensed demolisher under a separate contract (Stage 1), working
under the same CDC, and hand the site to the builder clear and level for the new dwelling (Stage 2). No profile value
differs from the base (the trigger lists no profile keys). 18 documents in 4 waves: d01, d04, d05, d09, d11, d12 and d13 changed.

## Passing mentions checked

Searched every base `content`, `carries` and title block for "builder's contract", "included", "builder" next to
demolition, and asbestos removal. Only d01 and d13 carried the negation "Demolition and asbestos removal are included
in the builder's contract"; d05 now also obtains the demolition quotations. d09 and d11 put the removal scope in
the builder's tender and the builder's work plan; both now name the demolition quotation and the demolisher. d04 now
has the disconnections ordered before the demolisher starts, with the demolition quotations alongside the tender.
d12 adds that the demolisher removes the old footings and slabs and the builder's earthworks cover fill removal and
backfill (the old-footings trap sentence is unchanged). d07 (sediment control "from the start of demolition") stays
true. The d01 budget sentence ("including demolition") stays true: it is the owners' total for both
contracts; its follow-on sentence now says the figure covers both contracts. The project summary is updated.

## Traps

- d01 `stage-1-demolition` (new; d01 becomes adversarial): "Stage 1: demolition by a separately engaged licensed
  demolisher" matches the `staged` trigger; `flag.staged` stays no. DOMAIN.md §10 trap 7 says a demolition-then-build
  sequence is not staged delivery, and the checker allows no profile change for this trigger. Per review finding 1
  the negation in the same paragraph reads "Demolition before the builder starts is not staged construction; the
  new house is built by one builder under one contract". Per ruling 2, an app "yes" is an app defect (the staged
  asks text should exclude demolition before the building works).
- Base traps carry over.

## No added documents

The demolisher is engaged after the CDC, so its quotation falls outside this corpus (brief to first consultant
invoices). No demolition trade id exists either (shape gap), so a quotation could not be keyed to a package.

## What the scorer should see

Shape trigger `separate-demolition-contract`: cost lines "Demolition contract (separate)" and the owner-held
provisional sum; program rows to engage the demolisher and hand over a clear site. The shape records that the app
produces nothing extra (`fr.demolition-1a` already fires in the base), so these are expected misses.

## Missed by design

As the base.

## Shape comment

- The shape's `why` now says "Stage 1" wakes the staged question and the expected answer stays no (ruling 2).

## Key correction after run `20261106-t10-t12` (6 November 2026)

- `proj.lga` also accepts "Sutherland". The ST12 keys for the same council already accept it, as the ST11
  keys accept "Parramatta" for the City of Parramatta: the keys accept the area name without its "Shire" or
  "City of". The run read "Sutherland" from "Sutherland Shire Council". Whether the app should keep "Shire"
  is noted in FINDINGS (F22).

# ST10-base render review

Reviewer: independent read, 2026-11-06. Corpus not edited.

## Verdict: FIX

Lint: `ok  testing/synthetic/scenarios/ST10-base` (18 documents, every `carries[].says` present).

## What a PM concludes from the documents alone

A knock-down rebuild at 22 Marrakai Close, Caringbah South (Lot 31 DP 216847, Sutherland Shire, R2, 619.8 m2,
street to the west, rear to the east). A 1962 brick-veneer and fibro house, rear garage and shed are demolished after
licensed removal of bonded asbestos, and replaced by an architect-designed two-storey Class 1a house with an
attached 10a double garage: 302 m2 GFA, 4 bedrooms, 3 bathrooms, all-electric. One CDC lodged by the architect;
select tender to three builders on the HIA lump-sum contract including demolition; budget $1,520,000 ex GST; high
quality; owners move out; single owner; unrestricted access; Class M, no groundwater, slab on ground; no OSD, kerb
discharge; trees retained. Consultants: architect $86,000, structural $8,600, civil $10,500 ex ($11,550 inc),
geotechnical $4,100, hazmat $2,650, BASIX $2,100, all ex GST unless noted. Invoices: hazmat $2,915, architect
Stage 1 $20,900 (ABN differs from the proposal), geotechnical $4,510.

This matches `expected/profile.json`, `fees.json` and `invoices.json` on every value. Title blocks match
`document_register.json`; d02, d07 and d18 are messy tables; d03, d11, d12, d13 and d17 are clean plus their
declared traps. The findings are about stray triggers and one geometry error.

## Findings

1. **d12 `Geotechnical report SLG-26-0461 - 22 Marrakai Close.md`, line 61 (writer departure b): the old footings
   are on the wrong boundary.** The survey (d03) puts the street on the west (NBN pit at the north-western corner,
   driveway along the north to a garage in the rear north-eastern corner), so the eastern boundary is the rear
   boundary, where only the old garage stands, beside the spotted gum and jacaranda. "Old brick strip footings
   along the eastern boundary are to be excavated" therefore means digging beside the two rear trees, which a PM
   reads as a tree impact and which contradicts d13 "all works are outside tree protection zones". It also does
   not fit "old brick strip footings" of the house, whose rear wall is about 13 m from those trees. Change, in the
   corpus and in the manifest trap `old-footings-boundary` `what`:
   "The old brick strip footings along the southern side boundary are to be excavated and removed after
   demolition."
   (The house has a path along its southern side, so its strip footings sit near that boundary; the
   `excavat…boundar` trigger still fires.)

2. **d12, lines 59 to 61: the trap sits directly under its own negation.** "no excavation near the boundaries"
   followed by the next bullet saying footings along a boundary are to be excavated reads as a contradiction no
   geotechnical engineer would leave. Move the old-footings sentence (with the fill sentence that follows it) out of
   the `## Footings and slab` list into the first paragraph of `## Confirmation after demolition`. The carries text
   is unchanged.

3. **d03 `Detail and boundary survey report - 22 Marrakai Close.md`, line 72: undeclared gas trigger.** "There is a
   gas main in the street, with the meter on the northern side wall of the house" matches the `gas_supply` trigger
   `gas (main|supply…)` and states an existing natural gas connection in a read report, so Jev can answer
   `natural_gas_main` against `none` from d01 and d13. Only d13's gas sentence is a declared trap. Change the corpus
   and the manifest d03 `content` to: "A gas meter is on the northern side wall of the house, supplied from the
   street." (no trigger match; the abolition in d13 still makes sense). If the hit is wanted, declare it instead as
   a d03 trap with expect `det.gas_supply stays none`.

4. **d06 `Structural engineering fee proposal PKE-2604.md`, line 22 (writer departure a).** "No transfer structures
   are expected…" is ordinary engineering prose and the lint does not flag it, but the render brief bans the word
   outright and the key folder is named `expected/`. Change the manifest d06 carries and the corpus to:
   "No transfer structures are needed for a two-storey timber-framed house." The `transfer_structures` trigger
   (`transfer structure`) still fires and the negation is unchanged. Keep d11 "unexpected-finds procedure"; that is
   the industry term.

5. **d13 `Architectural design statement - 22 Marrakai Close.md`, line 33-34: undeclared contamination hit.** "no
   environmental sensitivities such as bushland, waterways or contamination" matches the `potentially_contaminated_land`
   trigger `contaminat` (DOMAIN.md §10 trap 5). It is a carries line and a fair negation, so no corpus change; add it
   to the d13 traps (expect `det.potentially_contaminated_land` stays false from d02) so a miss is attributed
   correctly.

6. **d13, lines 66 to 73: the Trees section reads as a checklist.** Three one-line paragraphs repeat that nothing is
   affected. Join them into one paragraph in the same order (garage sentence; "No trees are affected; all works are
   outside tree protection zones."; "No neighbouring or street trees are affected by the works."). Wording unchanged.

## Writer departure c (d12 "could not be investigated")

Accepted. The manifest `content` is guidance, not carries; "could not be tested" would put the banned word "test"
into the corpus, and "investigated" is what a geotechnical engineer writes.

## Realism

All 18 read as real documents. d07's "effective from the start of demolition" in the fee table is slightly odd but
plausible for an ESCP line. Apart from finding 6, none read as written for a test.

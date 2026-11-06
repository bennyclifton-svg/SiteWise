# ST11-three-plus-townhouses render review (independent read)

Reviewer: senior PM read of `corpus/` against `manifest.json`, `expected/`, the T11 shape and DOMAIN.md §10.
This is a variant of ST11-base. The read covered the 9 rewritten documents (d01, d05, d06, d07, d10, d13,
d14, d15, d16), the added d20 and d21, and the three documents the writer changed outside
`changed_from_base` (d04, d12, d17). The other documents are byte-identical to the ST11-base final corpus,
which has its own review.

## Verdict: FIX

Five findings. Two are serious:

- The invented lot areas on A-101 cannot hold the drawn layout (finding 1).
- "Common property" is strata language in a community title scheme (finding 2).

Every fee, invoice and profile value a PM would take from the documents matches the key.

## Step 1: lint

`python testing/synthetic/tools/lint_corpus.py testing/synthetic/scenarios/ST11-three-plus-townhouses`
gives `ok  ST11-three-plus-townhouses`.

## Step 2: what a PM concludes before opening the key

- **Project:** demolish the 1958 house and garage. Build multi dwelling housing: one attached row of
  4 two-storey, two-bedroom townhouses, each Class 1a with a Class 10a single garage. GFA is 460 m2 in total,
  115 m2 each. The townhouses face a common driveway along the southern boundary, with one crossing.
- **Subdivision:** community title into proposed Lots 1 to 4 and a shared driveway/bin parcel.
- **Site:** Lot 9 DP 236154, 700.4 m2, R2, City of Parramatta, climate zone 5. Work type new.
- **Pathway:** one DA, which the architect lodges. Permissibility relies on the Housing SEPP low-rise
  provisions (verified: false). No cl 4.6, determined under delegation.
- **Site conditions, hazmat, owner and contract:** as the base. Vacant during the works, mid quality,
  architect-led tender, HIA lump sum, $1,650,000 ex GST, target start 1 February 2027 (email only).
- **Consultants:** the base set plus traffic (d20, $6,500 ex GST) and waste (d21, $2,500 ex GST).
- **Fees and invoices:** as the base.

## Step 3: comparison with the key

The conclusions match `profile.json` (dwellings 4, average dwelling size 115, subclass townhouses),
`fees.json` (8 submissions, d20 650000 and d21 250000 cents ex GST) and `invoices.json`. The base's d18
`arithmetic` flag is unchanged. d20 and d21 match `document_register.json` (WTE-FP-0391 on 18 March 2026 and
BWC-0817 on 19 March 2026, with matching titles).

**Lot areas against the subdivision type and the dwelling count.** A-101 gives 4 x 128.0 + 188.4 = 700.4 m2.
That equals the survey area. Four lots match 4 dwellings. A separate shared parcel is right for community
title, and wrong for the base's Torrens subdivision, which had no shared land. The subdivision type follows
the trigger's own evidence ("community title subdivision into 4 lots with a common driveway"). The geometry
fails on two points: the setback (finding 1), and the strata word "common property" (finding 2).

## Findings

### 1. d14 and d15 A-101: equal 128.0 m2 lots do not fit the 6.0 m front setback (serious)

- **Files:** `Site and subdivision plan A-101 Rev A.md` and `... Rev B.md`, line 33: "Proposed Lots 1 to 4:
  128.0 m2 each; common property 188.4 m2".
- **What is wrong:** the arithmetic works. A 4.57 m driveway strip over the 41.2 m depth gives 188.4 m2,
  leaving a strip 12.43 m wide. Four equal lots are then each 10.30 m long along the driveway. Lot 1 also
  carries the 6.0 m front setback (line 20, and the d16 DCP table), about 74.6 m2. That leaves about 53 m2
  for a townhouse whose ground floor plus single garage is about 75 m2 (115 m2 over two storeys, plus about
  18 m2 of garage). A PM would query this at once.
- **Exact change:** in d14 and d15 line 33, write "Proposed Lot 1 174.0 m2, Lots 2 and 3 99.4 m2 each,
  Lot 4 139.2 m2; [shared parcel, see finding 2] 188.4 m2". Those areas are units 8.0 m wide on the 12.43 m
  strip: Lot 1 is the 6.0 m setback plus 8.0 m, and Lot 4 takes the 11.2 m remainder. They total 512.0 m2,
  and the site total stays 700.4 m2. In the manifest, d14 `content` should say that the lot areas are unequal
  and that Lot 1 includes the front setback. Neither field is scored (`avg_dwelling_size_sqm` reads the area
  schedule, which does not change).

### 2. "Common property" is strata vocabulary in a community title scheme (serious)

- **Files:** these lines use "common property":
  - d01 line 10 ("the driveway and bin area held as common property")
  - d13 line 18
  - d14 and d15 lines 17, 22 and 33 (d14 line 23 as well)
  - d16 line 41 ("Lots 1 to 4 and common property")

  The manifest `project.summary` and the d14 `content` also use it, and so does `expected/notes.md` line 4.
- **What is wrong:** community title in NSW (Community Land Development Act 2021, verified: false) holds
  shared land as **association property**, run by a community association under a management statement.
  "Common property" belongs to a strata scheme (Strata Schemes Development Act 2015) and its owners
  corporation. The shape's sibling trigger `strata-subdivision` ("Strata (or community title) subdivision
  instead of Torrens") uses "two lots and common property" and "common property under a ... strata scheme" as
  its evidence. A reader would take this corpus as a strata-like scheme that the manifest does not plan, and
  the key carries none of that trigger's adds. The subdivision type itself is consistent with the trigger and
  the dwelling count. Only the word is wrong.
- **Exact change:** replace "common property" with "association property" in every place listed above. Keep
  "common driveway". In d14 and d15, write "dividing the site into proposed Lots 1 to 4 and the association
  property lot". Alternatively, if the owner prefers strata, add `strata-subdivision` to the manifest
  `triggers` and to the key. Do not leave the mixed form.

### 3. The manifest does not record the d04, d12 and d17 changes, and notes.md says they are unchanged (moderate)

- **Files:** these differ from the ST11-base final corpus:
  - `RE Timing for the build.md` lines 9 and 10: "A dual occupancy with a subdivision" became "Townhouses
    with a subdivision".
  - `Geotechnical report BRG-26-0655 - 18 Wattlebird Street.md` line 19: "attached pair of" became
    "attached row of two-storey townhouses". Lines 55 to 57: "both dwellings" became "all the townhouses",
    and the thickened footing now goes "under each separating wall between the townhouses".
  - `Tax invoice FLA-INV-1031.md` line 13: "Project: Dual occupancy and subdivision" became
    "Project: Townhouses and subdivision".

  All three edits are correct; the base wording would contradict the variant. `expected/notes.md` lines 15
  and 16 say that "d02, d03, d04, d08, d09, d11, d12 and the invoices never named the dwelling count or the
  dual occupancy, so they are unchanged". That is false for d04, d12 and d17.
- **Exact change:** append to the manifest `changed_from_base`:
  - `{"doc": "d04", "change": "changed", "why": "Architect's reply says 'Townhouses with a subdivision' instead of 'A dual occupancy with a subdivision'; target start unchanged"}`
  - `{"doc": "d12", "change": "changed", "why": "Introduction says 'attached row of two-storey townhouses'; footing advice covers all the townhouses with a thickened footing under each separating wall; classification and traps unchanged"}`
  - `{"doc": "d17", "change": "changed", "why": "Project line says 'Townhouses and subdivision' instead of 'Dual occupancy and subdivision'; amounts unchanged"}`

  In the d12 `content`, change "a thickened footing under the separating wall" to "a thickened footing under
  each separating wall". In notes.md, remove d04 and d12 from the "unchanged" list. Replace "and the invoices"
  with "d18 and d19". Then add "d04, d12 and d17 had passing dual-occupancy wording reworded".

### 4. d03 and d13: the large gum "near the rear boundary" cannot stay "well clear" of a full-depth row (moderate; latent `trees-affected`)

- **Files:** `Detail and boundary survey report ...md` line 54 (T1: a large gum near the rear boundary,
  16 m high, 10 m spread) and `Architectural design statement ...md` line 66 ("the large gum near the rear
  boundary is retained well clear of the works").
- **What is wrong:** in the base, the two dwellings sat at the front with rear yards. Here the row of 4 runs
  the depth of the lot, so Townhouse 4 and its courtyard sit about 3 to 11 m from the rear boundary. A 16 m gum
  there almost certainly has its TPZ over the Lot 4 footprint (AS 4970, verified: false). A PM would ask for
  an arborist, which is the `trees-affected` trigger this scenario does not plan.
- **Exact change:** change d03 line 54 location to "In the neighbouring rear yard, about 9 m beyond the rear
  boundary". Change d13 line 66 to "...; the large gum in the neighbouring rear yard is well clear of
  Townhouse 4". Record d03 in `changed_from_base`, and update the manifest d03 `content` and the
  `trees-present-not-affected` trap `what` to match.

### 5. d20 and d21: swept paths "for waste collection" with kerbside bins (minor, realism)

- **Files:** `Traffic and parking fee proposal - Wayfinder.md` line 31 (item 3 "...and for waste
  collection"), and `Waste management plan fee proposal - Binley.md` line 39 ("Swept paths for collection
  vehicles (by the traffic engineer)").
- **What is wrong:** d16 and d21 both say the bins are wheeled to the kerb for council's kerbside collection,
  so no collection vehicle enters the site and no swept path is needed for one. The text follows the
  trigger's `must_include` "Swept paths for the common driveway and waste collection", so the key wording
  stays.
- **Exact change:** in d21 line 39, change the exclusion to "Swept paths for the common driveway (by the
  traffic engineer)". In d20 line 31, change item 3 to "Swept paths for the common driveway and garages, and
  for the bin wheel-out route at the crossing". Keep "waste collection" in the item title if the RFP scorer
  needs the phrase.

## Step 4: stray triggers

Shape `triggers[].evidence` searched: strata, right of carriageway, battle-axe, two buildings, vacant site,
no subdivision, easements, sewer main in the lot, TPZ, BAL, flood, heritage area and lot size below the
minimum.

- The only literal hits are planned: the d01 vacant site, and the d03 "No easements" negation.
- "Strata" does not appear. The near-miss is "common property" (finding 2), and the latent tree impact is
  finding 4.

DOMAIN.md §10 traps, all declared:

- d13 "building a row" (`building-a-row`).
- d06, d12 and d13 separating or party wall (`party-wall-not-neighbour`); d12 now says "each separating
  wall", which still fires.
- d01 and d13 rainwater tank.
- d03 sewer main in the street.
- d05 and d10 Stage 1 to 4.
- "4 lots" in d01, d13 and d16 fires the dwellings trigger with the true count 4. notes.md records this.
- "Dual occupancy" survives only in d02's land-use table, as in the base.

## Step 5: title blocks

- d20 and d21 are `clean`. Each has `Title:`, a number and `Date:` before the first `##`, and one `#` equal to
  `Title:`. The values match the register.
- The rewritten clean documents keep their base title blocks. Only d01 retitled: "Owner's brief: townhouses
  at 18 Wattlebird Street", matching the manifest and the register.
- d14 and d15 are unchanged apart from the project line.
- The messy d07 keeps its table block and GST-inclusive figures. The adversarial d12 and d13 keep their traps
  in the body.

## Step 6: realism

The rewrites read naturally. The single crossing, bin store and mailboxes, and the "row does not read as a
repeated unit" language are what an architect writes for townhouses. The d16 traffic and waste sections
are typical SEE paragraphs. d20 and d21 read like small-firm quotes.

Beyond findings 1, 4 and 5, the notes already concede that $1,650,000 and an $85,000 architect's fee are
light for 4 dwellings. Two-bedroom units make the 460 m2 GFA believable. Nothing reads as written for a test.

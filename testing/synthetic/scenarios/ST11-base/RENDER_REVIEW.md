# ST11-base render review (independent read)

Reviewer: senior PM read of `corpus/` against `manifest.json`, `expected/` and the T11 shape and DOMAIN.md §10.

## Verdict: FIX

Seven findings. Two are serious. In the survey, the street tree sits inside the frontage beside a new crossing (finding 1). Read documents hit `party-wall` and `rainwater-tank` keywords that their manifest entries do not declare (findings 2 and 3). The documents carry no wrong figure. Every fee, invoice and profile value a PM would take from them matches the key.

## Step 1: lint

`python testing/synthetic/tools/lint_corpus.py testing/synthetic/scenarios/ST11-base` gives `ok`.

## Step 2: what a PM concludes before opening the key

- Project: demolish a 1958 single-storey brick house and fibro garage. Build an attached, side-by-side pair of
  two-storey, four-bedroom townhouses (two Class 1a, each with a Class 10a garage, 460 m2 GFA), with Torrens
  subdivision into proposed Lots 1 and 2. Lot 9 DP 236154, 700.4 m2, R2, City of Parramatta, NSW. Climate zone 5.
- Work type: new. Pathway: one DA, which the architect lodges. The land is CDC-eligible but the DA was chosen. No
  panel, no cl 4.6, neighbour notification only.
- Site: unrestricted access, standard sensitivity. Not heritage, flood or bush fire land, and no acid sulfate
  soils or contamination record. Class M. No groundwater, no basement, no excavation next to neighbours. Sewer and
  water reticulated, no gas. OSD is required and none exists. No easements.
- Hazmat: bonded asbestos and lead paint, removed before demolition. The loose-fill register search is negative.
- Owner and contract: single owner. Vacant during the works (the lease ends 31 October 2026). Mid quality.
  Traditional architect-led tender, HIA lump sum. Budget $1,650,000 ex GST. Target start 1 February 2027 (stated
  in the email only).
- Consultants: architect, town planner, structural, civil/stormwater, geotechnical, hazmat and surveyor (survey
  done, no fee in the corpus).
- Fees (ex GST): architect $85,000 (+ CA option $18,000, $220/h), structural $9,800 ($360/visit), civil $13,000
  ($14,300 inc GST), geotech $4,100 ($450/visit), hazmat $2,600 (+ $750 option) and planner $11,000.
- Invoices: FLA-INV-1031 $16,000 + $1,600 = $17,600. BRG-0522 $4,100 + $410, but it states $4,610, an
  arithmetic error. Greyfield 2207 $2,600 + $260 = $2,860.

## Step 3: comparison with the key

These conclusions match `profile.json`, `fees.json` and `invoices.json` value for value, including the d18
`arithmetic` flag. Findings 1 and 5 put two of those values at risk.

## Findings

### 1. d03 survey: the street tree is inside the frontage beside the northern crossing (serious; the writer's question 4)

- File: `Detail and boundary survey report - 18 Wattlebird Street.md` line 55: `T2 | Street tree on the verge |
  About 6 m north of the centre of the frontage, at the kerb line | 7 m | 5 m`.
- What is wrong: the frontage is 17.0 m, so the tree is 2.5 m from the northern side boundary, inside the
  northern lot's 8.5 m frontage. The survey lists only trees with trunks over 300 mm, so the AS 4970 TPZ is at
  least 3.6 m (12 x DBH). Each lot gets its own crossing (d13, d14, d15 and d16). Any 3 m crossing on the northern
  lot then lies within about 3 m of the trunk, which is inside the TPZ. A PM would not accept d13's "No neighbouring
  or street trees are affected by the works". This is near-verbatim the `trees-affected` evidence ("the second
  driveway encroaches into the tree protection zone of the street tree"). The manifest also contradicts itself: the
  d03 trap says the tree is "well clear of the frontage", but the d03 content says "about 6 m north of the
  frontage centre".
- Exact change: in the manifest d03 `content`, replace "street tree on the verge about 6 m north of the frontage
  centre" with "street tree on the verge in front of the northern neighbour, about 8 m beyond the northern side
  boundary, well clear of both new crossings". In d03 line 55, change the location cell to `On the verge in front
  of the neighbouring property to the north, about 8 m beyond the northern side boundary, at the kerb line`. Then
  the base's "trees not affected" holds, and the trap text and content agree.

### 2. d06 and d12: undeclared `party-wall` hits in documents the app reads (serious; the writer's question 2)

- Files: `Structural engineering fee proposal CSE-2607.md` line 28 ("the thickened footing under the separating
  wall") and `Geotechnical report BRG-26-0655 - 18 Wattlebird Street.md` line 57 ("a thickened footing under the
  separating wall between the two dwellings").
- What is wrong: `scope_defaults.yaml` `implied_by party-wall` (`\b(?:party|separating|common) walls?\b`) runs in
  code on every passage the label stage reads (`internal/jobs/worker.go`, `ImpliedReadings`). d06 is `commercial`
  and d12 is `report`. Both kinds are read (DISCOVERY.md §5.2), so each one alone suggests
  `site.adjoining-structures`. Only d13 declares the trap. The hits in d14 and d15 are not findings, because
  drawings are not read unless the user sets them to read. The wording is right for an engineer, so keep it.
- Exact change: in the manifest, add to d06 `traps` and to d12 `traps`:
  `{"id": "party-wall-not-neighbour", "what": "'the separating wall' (the footing under the wall between the two
  new dwellings) matches implied_by party-wall (DOMAIN.md §10 trap 1).", "expect": "No site.adjoining-structures
  work item"}`. In `expected/notes.md` under Traps, add "d06 and d12 `party-wall-not-neighbour` (the thickened
  footing under the separating wall)". Do not change the corpus.

### 3. d01 brief: an undeclared `rainwater-tank` hit (serious, the same class as finding 2)

- File: `Owner brief - 18 Wattlebird Street.md` line 48: "a BASIX rainwater tank for each dwelling".
- What is wrong: this matches the `det.water_supply_source` trigger `rain ?water tanks?`, which d13 declares as
  `rainwater-tank`. d01 is a `design_brief`, so it is read, but its manifest `traps` list is empty. The manifest
  content asked for the sentence, so this is a declaration gap, not a writer error.
- Exact change: in the manifest, add to d01 `traps`: `{"id": "rainwater-tank", "what": "'a BASIX rainwater tank
  for each dwelling' matches the water_supply_source trigger 'rain ?water tanks?'.", "expect":
  "det.water_supply_source stays reticulated"}`. In `notes.md`, add d01 to the `rainwater-tank` trap line.

### 4. d08 geotech proposal: "by arrangement with the tenants" (moderate; the writer's question 1)

- File: `Geotechnical investigation fee proposal - Basalt Ridge.md` line 25.
- Ruling: the writer is right to depart. The manifest's "drilled after the house is vacated but before demolition"
  cannot happen: the lease runs to 31 October 2026, and d12 records fieldwork on 14 April 2026 with a report on
  20 April. **The manifest content should change.** "Tenants", though, is a new occupancy cue in an early `## Scope`
  passage of a `commercial` document, and that passage gets header questions (`HeaderRouted`, ordinal ≤ 12 and
  section "Scope"). Jev could then answer `hdr.cond.operational_constraints` as `live_environment` or
  `partial_occupation` against d01's `vacant`. The d09 proposal and the d11 report already use the neutral
  "managing agent".
- Exact change: in the manifest d08 `content`, replace "drilled after the house is vacated but before demolition,
  clear of the house" with "drilled before demolition, clear of the house, with access to the front and rear yards
  arranged through the owners' managing agent". In d08 lines 24 and 25, change "...to about 3 m or refusal, by
  arrangement with the tenants for access to the front and rear yards." to "...to about 3 m or refusal. Access to
  the front and rear yards will be arranged through your managing agent."

### 5. d13 design statement: "exceeds the code's height standard" (moderate; the writer's question 3)

- File: `Architectural design statement - 18 Wattlebird Street.md` line 71.
- What is wrong: d02 states no LEP height control. Planning certificates do not, so the documents do not
  contradict each other directly. The domain problem is that the Low Rise Housing Diversity Code's dual-occupancy
  height limit and the usual Parramatta R2 height of buildings control are, as far as I know, both 9 m
  (verified: false). A design over the code height is then very likely over the LEP height too. That needs a
  cl 4.3 assessment and a cl 4.6 request, but d13 and d16 both say no cl 4.6, and the d16 SEE never mentions
  height. A PM would ask about it at once, so the `flag.cl46 = no` key rests on a shaky premise. DOMAIN.md §9
  (cdc-low-rise-code row) put this wording in the base, so the writer followed the domain.
- Exact change: give the DA a reason that does not touch an LEP development standard. In the manifest d13
  `carries` `hdr.cond.planning` `says` and in d13 line 71, replace "The design exceeds the code's height standard,
  so a DA will be lodged" with "The design does not meet the code's landscaped area standard, so a DA will be
  lodged". Make the same change in the DOMAIN.md §9 cdc-low-rise-code row's base wording. Optionally, add
  "The proposal complies with the LEP height of buildings and floor space ratio standards." to the d16
  `### Minimum lot size and frontage` section, without numbers.

### 6. d12 geotech report: "the old house slab" contradicts the hazmat report (minor, realism)

- File: `Geotechnical report BRG-26-0655 - 18 Wattlebird Street.md` line 66 ("remove topsoil and the old house
  slab"), and line 65 ("the old footings, slab, paths and fill").
- What is wrong: d11 says the house has "timber floors on a subfloor", and d03 puts its floor at RL 98.45 against
  ground at about RL 97.8 to 98.2. The house has no slab; only the garage does (d03, RL 98.10).
- Exact change: in d12 line 66, change "the old house slab" to "the old garage slab". In the manifest d12
  `excavate-to-depth` trap `what`, make the same change. Keep the phrase "excavate to a depth of 300 mm", because
  it is the trap.

### 7. `expected/document_register.json` d12: the strict lifecycle is still there (minor, key)

- `REVIEW.md` finding 4 asked for d12 `lifecycle` to become `{"accept": ["design", "consultant"]}`, as for d03
  and d11. The key still says `"design"`. Apply it, because the key is wrong, not the render.

## Step 4: stray triggers

I searched for every `triggers[].evidence` phrase in the shape: strata, right of carriageway, battle-axe,
4 townhouses, two buildings, vacant site, no subdivision, interallotment easement, a sewer main crossing the lot,
a sewer extension, a TPZ, BAL, flood, heritage conservation area, ANEF, rail noise and lot size below the minimum.
None is in the corpus. The near-miss is the d03 tree position (finding 1).

DOMAIN.md §10 keyword hits:

- Declared: d03 sewer main and trees. d05 and d10 Stage 1 to 4. d12 "excavate to a depth". d13 "building a dual",
  separating, party, fire-rated and acoustic wall, and rainwater tank. d02 Low Rise Housing Diversity Code.
- Undeclared: d06 and d12 separating wall (finding 2) and d01 rainwater tank (finding 3).
- Harmless because those documents are not read for the profile: d14 and d15 separating wall and rainwater tank,
  and d17 "Stage 1".
- Not a trap: "Water main in Wattlebird Street" in d03 is covered by the same `sewer-main-in-street` reading
  (authority_asset stays no). Consider naming it in that trap's `what`.

## Step 5: title blocks

Every `clean` document has its label lines before the first `##`, and one `#` heading that equals `Title:`. The
d17 and d18 invoices and the d14 and d15 drawings carry `Title:` plus a matching `#`. Numbers, revisions, titles
and dates all match `document_register.json`. The `messy` documents have the tiers' typical features:

- d02: a council table block and fixed council wording.
- d07: a table block, GST-inclusive figures and unheaded sections.
- d19: a table block and a fixed-width invoice.

The `adversarial` documents (d03, d11, d12 and d13) have clean title blocks with their traps in the body, as their
tier intends.

## Step 6: realism

The documents read like real ones: the council certificate wording, the stage-based fee letters, the hazmat sample
table, and the geotech profile and footing advice. Two passages read as written for the test:

- d13 `## Planning`. "No public exhibition or community consultation is proposed beyond neighbour notification"
  and "determined by council staff under delegation, not the local planning panel" are SEE language. An architect's
  design statement would rarely say this. Acceptable for an adversarial document.
- d03 services, "No Sydney Water assets are within the property; the sewer main is in the street". A surveyor would
  say "no sewer within the lot". Acceptable.

Otherwise, findings 1, 5 and 6 are the realism defects.

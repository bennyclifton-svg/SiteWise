# ST13 review (T13 mixed-use-shop-top)

Scenarios: ST13-base, ST13-rise-three-storeys, ST13-contaminated-land.

## Verdict: FIX

Checker: `python testing/synthetic/tools/validate_expected.py testing/synthetic/scenarios/ST13-*` gives `ok` for
all three. The rise variant passes only because five keys are held at base values that the variant's own documents
contradict (finding 1). That is the shape problem the author raised. It is settled below.

What this review covered: all seven brief steps. I diffed each variant's manifest, profile, fees, procurement and
pointer files against the base. I re-added every fee table and invoice, checked every ABN checksum (all nine are
valid), and read every `title_block` against DISCOVERY.md §4.3.

## Findings

### 1. ST13-rise-three-storeys: five scale and budget keys held at base values (serious; settles the shape problem)

- What is wrong: `triggers[rise-three-storeys].profile` in the shape is
  `[det.rise_in_storeys, hdr.scale.total_storeys, det.effective_height, det.sprinklered]`. Cutting six storeys to
  three necessarily changes the residential level count, the unit count, the GFA, the parking and the budget. The
  checker lets only listed keys move, so `expected/profile.json` holds `hdr.scale.tower_levels` 5,
  `hdr.scale.residential_units` 42, `hdr.scale.total_gfa_sqm` 5500, `hdr.scale.car_parks` 55 and
  `proj.budget_stated` 2450000000. Meanwhile d01, d13, d14 and d15 state 2, 18, 2,700 m2, 30 and $10,800,000. A
  correct app would score wrong on all five. `rfps.json` already carries the true 1080000000, so the set
  contradicts itself.
- Ruling: **change the shape.** This is one trigger, not several. A three-storey shop-top has fewer apartment levels,
  apartments, floor area and parking by definition. Holding $24.5M for a 2,700 m2 building would be about $9,000/m2,
  which no PM would accept. Precedents: T05 lists `hdr.scale.added_gfa_sqm` and `hdr.scale.gfa_sqm` with
  `hdr.scale.storeys`, T09 lists `proj.budget_stated` in a trigger's `profile`, and the ST11 review ruled the same
  way. `hdr.scale.podium_levels` (1), `hdr.scale.retail_gla_sqm` (600) and `hdr.scale.site_sqm` stay at the base
  value because the documents keep them.
- Exact changes (fix round, by the author):
  1. `testing/synthetic/types/T13-mixed-use-shop-top/shape.yaml`, trigger `rise-three-storeys`:
     `profile: [det.rise_in_storeys, hdr.scale.total_storeys, det.effective_height, det.sprinklered]` ->
     `profile: [det.rise_in_storeys, hdr.scale.total_storeys, det.effective_height, det.sprinklered, hdr.scale.tower_levels, hdr.scale.residential_units, hdr.scale.total_gfa_sqm, hdr.scale.car_parks, proj.budget_stated]`.
     All five keys are already accepted profile keys in the base, so `validate_shape.py` will take them.
  2. ST13-rise-three-storeys `expected/profile.json`: key the true three-storey values:
     - `hdr.scale.tower_levels` -> `{"value": 2, "sources": ["d01"], "read_by_design": true}`
     - `hdr.scale.residential_units` -> `{"value": 18, "sources": ["d01", "d13"], "read_by_design": true}`
     - `hdr.scale.total_gfa_sqm` -> `{"value": 2700, "sources": ["d01"], "read_by_design": true}`
     - `hdr.scale.car_parks` -> `{"value": 30, "sources": ["d01", "d14", "d15"], "read_by_design": true}`
     - `proj.budget_stated` -> `{"value": 1080000000, "sources": ["d01"], "read_by_design": true}`
  3. ST13-rise-three-storeys `expected/notes.md`: replace the "Keys held at the base value" section and table with
     a "Changed keys" list covering all nine keys. Keep the sentence that fees and invoices stay at base amounts.
  4. ST13-base `expected/notes.md`: delete the last shape comment ("`rise-three-storeys` lists too few profile
     keys").
  5. Re-run the checker. It should pass with the true values.

### 2. ST13-contaminated-land `expected/notes.md`: say plainly that the procurement delta comes only from d19 (moderate)

- Confirmed correct: `det.potentially_contaminated_land` false -> true is the only profile change. Sources d02, d05,
  d13 and d19 are all read kinds, and each passage contains `contaminat` (d05 in its Conclusions section). d19 is
  cued as a proposal ("Fee proposal" heading), adds up ($38,500 + $14,000 + $22,000 = $74,500 ex GST), and is keyed
  in `fees.json`, `document_register.json` and `procurement.json` `from_proposals`. The notes already say the package
  rule `complexity_additions[2]` can never fire (reference/index/procurement.md line 36).
- What is unclear: the base shape already expects `consultant.environmental` (`need: usually`, built_in no). So the
  base scores a miss on it, and the variant scores a match only because d19 opens a package. `score_procurement`
  matches packages and suggestions alike and records `state: package`. The base-to-variant procurement change
  therefore tests the proposal drop, not the reading of the trigger. Also, "If the harness scores suggestions
  separately, expect a miss there" does not describe how the scorer works.
- Exact change: in the Procurement bullet, replace the last two sentences with: "The base shape already lists
  `consultant.environmental` (need: usually), and the app misses it in the base. In this variant it is matched only
  as a package opened by the d19 proposal (`state: package`), never as a suggestion. The procurement change between
  base and variant comes from d19, not from `det.potentially_contaminated_land`."
- Shape note (no scenario change): the trigger's `adds.procurement: [consultant.environmental]` repeats a base row.
  It does no harm, and the shape author can leave it.

### 3. ST13-rise-three-storeys `manifest.json` d01: the building-class sentence lost "mixed-use" (minor)

- What is wrong: in the base, `hdr.building_class` is carried by "a new six-storey mixed-use building: ground-floor
  retail with 42 apartments". The variant rewrote this to "a new three-storey shop-top housing building: a
  ground-floor shop with two levels of apartments above (18 apartments)". DOMAIN.md §10 says "shop-top housing" is
  a planning term that fires nothing. The trigger evidence is already carried by the `hdr.subclass` sentence, so
  this second rewrite is a change the trigger does not need. "A ground-floor shop" in the singular also contradicts
  the three shell tenancies that d01 keeps.
- Exact change: `hdr.building_class` says -> "Halcyon Eastwood Pty Ltd will develop a new three-storey mixed-use
  building: ground-floor retail with 18 apartments on the two levels above". Leave the `hdr.subclass` evidence
  sentence as it is.

### 4. ST13-* `expected/document_register.json` d11: the lifecycle key is stricter than for the matching reports (minor)

- What is wrong: d11 (geotechnical report) expects `"design"` only. The other consultant reports in the set, d03,
  d05 and d12, accept `["design", "consultant"]`, and so do the T11 geotechnical reports after their review. No
  rule settles the lifecycle of a report: d11 has no `Drawing number` line and no caption.
- Exact change: in all three scenarios, set d11 `lifecycle` -> `{"accept": ["design", "consultant"]}`.

### 5. ST13-base and ST13-rise-three-storeys `manifest.json` d02 and d05: contamination denials are undeclared traps (minor)

- What is wrong: d05 denies "service station, dry cleaner ... underground tanks", which are the contaminated-land
  trigger's own evidence words. d02 says "no record that the land is contaminated or potentially contaminated".
  Both fire `contaminat` and depend on Jev reading the negation, but both have `traps: []`. The rise variant copies
  both documents unchanged.
- Exact change: add to d05 `traps`:
  `{"id": "contamination-denied", "what": "The site history names 'service station', 'dry cleaner' and 'underground tanks' only to deny them; the conclusion fires 'contaminat'.", "expect": "det.potentially_contaminated_land stays false"}`.
  Add to d02 `traps`:
  `{"id": "potentially-contaminated-negated", "what": "'no record that the land is contaminated or potentially contaminated' contains the phrase it denies.", "expect": "det.potentially_contaminated_land stays false"}`.
  Mention both in the base notes' Traps list.

### 6. ST13-contaminated-land `manifest.json` d05: the service station sits "behind the shops" on the dwelling lot (minor)

- What is wrong: d03 puts the dwelling on No. 40 and the shops on Nos. 42 and 44. d05 says "the southern part of
  No. 40, now the open car park behind the shops". A renderer may move the shops or the dwelling to make this fit.
- Exact change: "the rear part of No. 40, now the open car park behind the dwelling".

### 7. Notes only, no change

- Rise variant fees: the checker holds every fee at the base amount. The architect's $424,000 to tender is 3.9% of
  $10.8M, which is heavy but possible. The existing note covers it.
- Rise variant effective height: the base's 18.9 m implies about 2.3 m below the retail floor (the site fall), which
  would give about 9.6 m at three storeys, not 7.6 m. Both are under every 12 m and 25 m threshold, so no answer
  changes.
- `det.type_of_construction` stays A in the rise variant. That is right under C2D4, because the top storey is
  Class 2 and Table C2D2 gives A at a rise of 3 (verified: false).

## Step 2: faithful to the shape (base)

Every T13 trigger is denied or absent. The denials are effective height under 25 m (d12), no heritage (d02), local
road (d13), FSR and height complied with (d13), seepage only (d11), low-risk PSI (d05), not flood-affected (d02), no
rail (d02), no Sydney Water assets in the site (d03), D&C (d01), three shell tenancies, Local Planning Panel, CIV
under $30M, no trees affected (d13), no acid sulfate soils (d02), neighbour on shale with no underpinning (d11), and
only a vehicle crossing (d08). The keyword traps in DOMAIN.md §10 are handled: singular "existing building", "built
in 1965", "one building", no "Stage 1", the sewer main in the lane, street trees paired with "not affected", the
planner lodges, the design review panel, strata subdivision and public exhibition. "Lodge the strata building bond"
in d01 incidentally fires `architect_lodges`, and the notes say so. Apart from finding 5, no accidental trigger.

## Step 3: register keys (DISCOVERY.md §4.2, §4.3)

- Label-line documents (d01, d03, d05, d06, d07, d09 to d13, and d19): every number label is accepted (`Ref`,
  `Report no`, `Proposal no`). Every number matches the pattern, every date format is accepted, and the one `#`
  heading is identical to `Title:`. d05 and d11 give rev 0, and d12 gives rev A. The rules settle what the key
  expects.
- d04 (email): title from the first `Subject`, date from the latest header (3 March), no number. Correct.
- d14 and d15 (drawings): `Drawing number`, `Revision`, `Title` and `Date` with no `#` heading. The lifecycle is
  `design` by rule. The supersession from d15 to d14 cannot be written today, and the notes say so (§4.5).
- d02 and d08 (table title blocks): the title comes from the `#` heading, but the number and date are in the table
  and are lost. The key keeps the true values, and the base notes say they will be missed. Acceptable.
- Invoices d16 to d18 go to the invoice drop, so they have no register row. Correct (§2.3).

## Step 4: profile keys

Every base value is supported by a `carries` entry in a read kind. `read_by_design` is false only for the two
targets in the d04 email. The values that also appear on drawings (GLA, parking) have d01 as a read source. Each
determinant passage matches its regex: `building classification`, `rise in storeys`, `effective height`,
`sprinkler`, `climate zone`, `traffic noise`, `\bsewer`, `mains water`, `natural gas`, `\bexisting building\b`,
`built in 1965`, `contaminat` and `heritage`. The flags use the shape's `scenario_inputs` phrasing. The known misses
(`proj.regulated_design` with no reader, podium and tower levels in prose only, novation option on every
consultant) are in the notes. Apart from finding 1, no profile key is wrong.

## Step 5: fees and invoices

The fee tables add up: architect $424,000, structural $103,000, geotechnical $26,000, planner $54,500, and d19
$74,500, all ex GST. The civil figures are GST-inclusive and were divided by 1.1: $15,400 / $20,900 / $16,500 /
$52,800 / option $27,500 give $14,000 / $19,000 / $15,000 / $48,000 / $25,000, with `gst_inc_normalised`, a real
flag (`read.go` FlagGSTInc). The invoices add up: d16 $68,000 + $6,800 = $74,800. d17 is $9,500 + 50% of $28,000 =
$23,500, plus $2,350 = $25,850. d18 is $20,200 + $2,020 printed as $22,420, which raises `arithmetic` only, because
its GST is exactly 10% of ex GST (`ledger.go` lines 306-313). There are no state flags in the keys, as brief 03
requires.

## Step 6: variants

- Contaminated-land: changes d02, d05 and d13 and adds d19. Each uses the shape's evidence wording ("service station
  until 1992; underground tanks may remain", "a dry cleaner operated in the ground-floor shop", "the PSI recommends
  a detailed site investigation"). Nothing else moves. See findings 2 and 6.
- Rise-three-storeys: the evidence follows the shape's wording, and every changed document follows from the cut.
  The keys are wrong until finding 1 is fixed. See finding 3.

## Step 7: realism

A PM would recognise this as a real Eastwood shop-top job: a brief, a s10.7 certificate, a survey, a PSI, five
consultant proposals, a geotechnical report, a BCA report, an SEE, two revisions of a drawing and three invoices,
from February to May 2026 ahead of a July DA. The pieces fit NSW practice:

- The Local Planning Panel is right for Housing SEPP Chapter 4 development, both at six storeys and at three storeys
  with 18 apartments.
- In the base the developer lodges the strata building bond. At a rise of 3, HBCF applies instead and the 30-space
  car park needs no sprinklers. That matches the shape's `why`.

The fees are plausible for a $24.5M D&C: architect 1.7% to tender (2.8% with post-novation), structural $195,000 in
total, civil $73,000 ex GST in total, planner $54,500, geotechnical $26,000, and DSI/RAP/validation $74,500.

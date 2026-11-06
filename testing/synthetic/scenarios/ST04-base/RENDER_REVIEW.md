# ST04-base render review (brief 04r)

Reviewer: independent (senior NSW PM). Date: 7 November 2026. Corpus, keys, tools and code were not edited.

**Verdict: FIX.** The 19 documents read as one coherent job and nearly every keyed value is stated where the key says.
Two changes matter. First, a stray d10 exclusion line asks the intersection-modelling question with wording that
implies counts are in scope. Second, `fees.json` gives all six firms a phone and an address that no document prints.
The other findings are key gaps or small wording fixes.

## 1. Lint

`python testing/synthetic/tools/lint_corpus.py testing/synthetic/scenarios/ST04-base` gave `ok  testing/synthetic/scenarios/ST04-base`
(exit 0).

## 2. What I concluded before opening expected/

- **Project:** Redgum Logistics Estates Pty Ltd, the sole owner. The work is a new single-storey warehouse (Class 7b,
  7,500 m2, 12 m clear, 12 recessed docks, 11 m top of storage, ESFR) and a single-storey office (Class 5, 500 m2).
  GFA is 8,000 m2. The site is a vacant benched estate lot of 16,000 m2 (16,012 m2 by survey) at 14 Ironstone Circuit,
  Erskine Park, Lot 21 DP 1289934. Rise in storeys 1, Type C, climate zone 6, large isolated building under C3D4.
- **Pathway:** a local DA to Penrith City Council, determined by staff under delegation. The land is E4 General
  Industrial. CDC is available on the land but not for this building, which exceeds the IBBC height standard. There
  is no cl 4.6 variation, no planning panel and no TfNSW referral. The planner lodges, not the architect.
- **Procurement:** an AS 4902 design and construct lump sum. The architect, structural engineer, civil engineer and
  landscape architect are novated.
- **Site:** none of these apply: heritage, bush fire, flood controls, acid sulfate soils, contamination (estate
  validation report), coastal, rail, groundwater, trees, basement, adjoining excavation, transfer structures, gas,
  Sydney Water assets or existing OSD. Sewer and water are reticulated. Power is from the grid via a kiosk. There is a
  3 m drainage easement on the east boundary. The geotech report gives site class M for the office slab, moderately
  saline soils, exposure B1 and a pavement CBR of 3%.
- **Consultants:**
  - Fee proposals in: architect, structural, civil, geotech, planner and traffic.
  - Reports with no fee in the set: surveyor and BCA.
  - Named but not quoted: landscape.
  - A fire engineer is implied by the travel-distance performance solution.
  - Services design can be left to the D&C contractor.
- **Budget:** $14,800,000 ex GST.
- **Fees (ex GST):** architect $240,000 (option $165,000, $240/h); structural $76,000 (option $58,000, $220/h); civil
  $88,000 inc GST, which is $80,000 ex GST (option $49,500 inc, $45,000 ex); geotech $24,000 (option $2,800); planner
  $38,000 ($230/h); traffic $19,000 ($210/h).
- **Invoices:**
  - MHA-INV-0311: $38,000 + $3,800 = $41,800.
  - Westbourne 1187: $17,500 + $1,750 = $19,250.
  - SHG-INV-0507: $18,800 + $2,820 = $21,620. The GST is charged at 15%.

## 3. Comparison with expected/

`profile.json`, `fees.json` (totals, stages, options, rates, GST basis), `invoices.json`, `procurement.json` and the
program and RFP facts all agree with my reading, except for findings 2 and 3 below. The planned 16,000 against 16,012
m2 conflict is accepted by the key, and the d19 `gst_rate` flag is right.

## Findings

1. **d10 `Lanefield Traffic fee proposal LT-0326.md`, line 59: "Traffic counts beyond one weekday morning and afternoon
   peak at the estate entry".** This is a stray trigger.
   - It fires `flag.intersection_modelling` through `traffic (?:count|model)` (`knowledge/profile/flag_readings.yaml:68`).
     The question asks "does the passage state that intersection counts or modelling are required?"
   - As an exclusion, the line implies that a count at the estate entry is in scope. So Jev can fairly answer yes,
     against the key's no and against the d10 SIDRA denial. That gives a conflict or a wrong value.
   - The wording also echoes the shape's `traffic-generating-development` evidence ("intersection counts at the
     estate entry").
   - The manifest plans neither the line nor the trap, and notes.md does not declare it.
   - **Change:** replace the bullet with `- Parking demand surveys at comparable sites`. Keep the other exclusion. The
     d04 email's "traffic counts" can stay because d04 is filed as correspondence and is not read.

2. **`expected/fees.json`: firm phone and address do not match any document, for all six submissions.** The app reads
   both (`internal/proposals/read.go` `FieldFirmPhone` and `FieldFirmAddress`). The scorer checks only the total and
   the firm name today (`harness/score.py score_fees`), but the key is still wrong for anyone who scores letterheads.
   The manifest fixes no phone or address. The documents are consistent with each other (d05 matches d17, d09 matches
   d18, d08 matches d19), so the key should follow the documents. **Change** in fees.json:

   | doc | phone: key → document | address: key → document |
   |---|---|---|
   | d05 | 02 9687 4410 → 02 9687 3304 | Level 2, 18 Smith Street, Parramatta NSW 2150 → Level 3, 41 Macquarie Street, Parramatta NSW 2150 |
   | d06 | 02 9622 1875 → 02 9893 4417 | Suite 4, 9 Main Street, Blacktown NSW 2148 → Suite 7, 12 Argyle Street, Parramatta NSW 2150 |
   | d07 | 02 4721 6630 → 02 4721 5583 | Level 1, 302 High Street, Penrith NSW 2750 → Level 2, 9 Station Street, Penrith NSW 2750 |
   | d08 | 02 9604 2287 → 02 4731 8862 | Unit 6, 41 Victoria Street, Smithfield NSW 2164 → Unit 3, 25 Coombes Drive, Penrith NSW 2750 |
   | d09 | 02 9635 7702 → 02 9623 1478 | Suite 12, 85 George Street, Parramatta NSW 2150 → Suite 2, 70 Queen Street, St Marys NSW 2760 |
   | d10 | 02 8814 3390 → 02 4722 9035 | Level 5, 10 Hunter Street, Parramatta NSW 2150 → Level 1, 5 Lawson Street, Penrith NSW 2750 |

   (If the owner prefers to keep the key, change the six letterheads and the three invoice letterheads to the key's
   values instead. Do not mix the two.)

3. **`expected/profile.json` omits two values that d11 states and the app reads.**
   - d11 says "classified as site class M (moderately reactive) under AS 2870" and "The soils are moderately saline".
   - `det.site_class` (trigger `site class(ification)?`) and `det.saline_soil` (trigger `salin(e|ity)`) are both live
     profile determinants (`knowledge/determinants.yaml`, `profile_group: site`). d08's scope also fires both
     without a value.
   - ST01-base keys `det.site_class`.
   - **Change:** add `"det.site_class": {"value": "M", "sources": ["d11"], "read_by_design": true}` and
     `"det.saline_soil": {"value": true, "sources": ["d11"], "read_by_design": true}`. Or, if they are left out on
     purpose, say so in notes.md.

4. **d13 `Statement of environmental effects - 14 Ironstone Circuit.md`, line 60: heading "## Flooding and stormwater".**
   *Can it make the base read as flood affected?*
   - Not by code. `det.flood_hazard` is a boolean that Jev decides. The only flood statement in the section is the
     planned denial on line 62.
   - But the heading travels as the `Section` of every passage under it (`internal/jobs/worker.go` stored passages,
     `passageExcerpt`), and the determinant trigger `\bflood` reads that excerpt. So the OSD paragraph on lines 64-65
     also gets a flood question even though its own text says nothing about flooding. The correct answer there is
     none.
   - The same heading also invites a `site.flooding` label on that paragraph.
   - The exposure is small but unplanned. **Change:** rename the heading to `## Stormwater`. The line 62 sentence
     still contains "flood", so d13 stays a source for `det.flood_hazard`.

5. **d13, line 15: "the consultants' reports lodged with the application".**
   - This fires `flag.architect_lodges` (`\blodge(?:d|s|ment)?\b`) in the same sentence that names Marrick Hale
     Architects.
   - notes.md does not declare it as an incidental.
   - **Change:** "lodged" → "submitted".

6. **d12 `BCA report CBR-26-071 - 14 Ironstone Circuit.md`, lines 34-35 against lines 67-68.**
   - Lines 34-35 say compliant perimeter vehicular access "is to be provided around the building". Lines 67-68 say a
     performance solution covers "the perimeter vehicular access".
   - A PM would ask which is true. Lines 67-68 are the `fire_specialist_systems` carry and must stay.
   - **Change** lines 34-35 to: "The building is treated as a large isolated building under C3D4. Perimeter vehicular
     access at least 6 m wide is provided around the building, within 18 m of it, except across the office frontage,
     where it is addressed by a performance solution."

7. **d15 and d16 (`A-1100 Rev A/Rev B - Site plan.md`), line 24: "Sprinkler tank and pump room near the north-eastern
   corner".**
   - d12 says "two fire water tanks and a pump room".
   - **Change** both sheets to: "- Two fire water tanks and pump room near the north-eastern corner".

## 4. The writer's noted trigger-like words: checked, no change

- **d02 s10.7 item headings** (heritage, bush fire, flood, contaminated land, loose-fill asbestos, coastal
  protection, rail corridors, road widening). Each heading is followed straight away by its planned denial, which is
  the manifest's carry. "Asbestos", "acid sulfate" (a planned carry) and "road widening" fire nothing unplanned. The
  certificate is messy tier, as planned.
- **"Gross pollutant trap"** (d07 line 29, d14 lines 41-44). It matches no trigger. `stormwater_quality` needs
  `pollutant (?:load|reduction)`, and "reductions in gross pollutants" does not match. It is a real term (brief 04
  allows real phrases), and lint passes.
- **"Test pits"** (d08, d11). These match no trigger. The d11 "excavated to a depth" basement trap is planned.
- **d14 "1% AEP event".** It is not a flood trigger (`\bflood` only), and it is normal OSD wording.
- **Other hits:**
  - d01's access sentence fires `flag.arborist_offsite_access` ("access … neighbour"). It is an arborist-only
    question, unkeyed and unscored. Harmless.
  - d13 "stormwater quality treatment" fires `stormwater_quality`, which agrees with the key (yes).
  - All other trigger and DOMAIN §10 hits are the manifest's carries or declared traps: floodlights, public
    exhibition, planning panel, TfNSW, SIDRA, bore logs, groundwater, hydrocarbon contamination, staged, multiple
    buildings, the CDC item and the planner lodging.
  - No shape `triggers[].evidence` wording appears, apart from finding 1.

## 5. Title blocks

- **Clean documents:** every clean document (d01, d03-d06, d08, d12, d15-d17) has `Label: value` lines before its
  first `##`, one `#` heading equal to `Title:`, and number, revision, title and date equal to
  `document_register.json`.
- **Adversarial documents:** d09-d11, d13, d14 and d19 have clean blocks plus their declared trap.
- **Messy documents:** d02 and d07 have table title blocks with the `#` outside the table. d18 is a table invoice.
- **Email:** d04 has `From/To/Subject/Date` and gives "RE: Program and DA timing" and 4 March 2026.
- **Supersession:** A-1100 Rev B supersedes Rev A in the key, and the revision table matches.

## 6. Realism

- **Overall:** the documents read like the real thing. Letterheads, fee tables, GST statements, invoices with bank
  details, the certificate's council wording, the geotech profile (estate fill over residual clay over Bringelly
  Shale) and the civil report all ring true.
- **Passages that read as written for a test.** These are all manifest carries, so I propose no change, but a real
  PM would notice:
  - d01, "Approvals and procurement" (lines 100-108): a run of denials ("This is one building; we are not planning
    multiple buildings on the lot", "no staged construction or staged handover", line 69 "No Connecting with Country
    input is required").
  - d11, line 27: "no basement is proposed and the building is slab on ground" in a fieldwork paragraph.
  - d12, "Fire engineering": three back-to-back negations.
- **Optional, outside this review:** d01 line 85 "the code's height standard" names no code; d09 supplies it.

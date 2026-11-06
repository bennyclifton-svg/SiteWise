# ST13-rise-three-storeys render review

Reviewer: independent read of `corpus/` (18 documents; 10 rewritten from ST13-base, 8 byte-identical copies) against
`manifest.json` and `expected/`. The base corpus is final and reviewed (`../ST13-base/RENDER_REVIEW.md`); this review
covers the ten rewrites (d01, d06, d07, d09-d15) and whether any copied document contradicts the three-storey version.

## Verdict: FIX

No scored value is at risk. Every scored figure is stated once and consistently: 3 storeys, rise in storeys 3, 18
apartments, about 2,700 m2 GFA, 30 spaces, $10,800,000 ex GST, effective height 7.6 m, not sprinklered. The fixes
are two technical errors a PM or certifier would catch: the C2D4 note in the BCA report (finding 1) and the geotech
rock depth against the site fall (finding 2). There is also one planning point to verify (finding 3) and three
small realism edits.

## 1. Lint

`python testing/synthetic/tools/lint_corpus.py testing/synthetic/scenarios/ST13-rise-three-storeys` gives
`ok  testing/synthetic/scenarios/ST13-rise-three-storeys`.

(If you run it with a relative `.` path from inside the scenario folder, it reports "base ST13-base: manifest.json
unreadable", because `Path('.').parent` is `.`. That is a tool quirk, not a corpus fault.)

## 2. A PM's reading, before opening `expected/`

- **Project.** Halcyon Eastwood Pty Ltd is the sole owner and decision maker. It is building a new three-storey
  shop-top building at 40-44 Halloran Street, Eastwood: three shell shops (600 m2 GLA; Rev B splits tenancy 2) under
  18 apartments on two levels, over one basement of 30 spaces. GFA is about 2,700 m2 on a 1,802.4 m2 site. The 1965
  shops and dwelling are demolished. The site is vacant during the works, access is constrained, and it is one
  building in one stage.
- **Class.** Class 2, 6 and 7a. Rise in storeys 3, effective height 7.6 m, not sprinklered, climate zone 5. The DBP
  Act applies (Class 2). BASIX covers the apartments and Section J the retail. Fire safety is DTS. Type of
  construction: the whole building is Type A under C2D4, because the top storey is Class 2. d12 says otherwise; see
  finding 1.
- **Planning.** A DA to Ryde, determined by the Local Planning Panel. Housing SEPP Chapter 4 and the ADG apply,
  which is right at 3 storeys. No clause 4.6. The planner lodges. Not heritage, flood, bushfire, acid sulfate,
  coastal or rail. Local road, no TfNSW, no heavy vehicles, no SIDRA, no green travel plan. Traffic noise affects the
  site. Street trees are not affected. Strata subdivision comes later.
- **Procurement and insurance.** D&C, AS 4902, lump sum, design team novated. The builder provides HBCF because the
  rise is 3 storeys; there is no strata building bond.
- **Consultants.** As the base: architect, structural, civil, geotechnical, planner, surveyor, environmental and BCA.
  Acoustic and services engineers are implied.
- **Site.** One basement, excavated about 3.8 m below Halloran Street. Fill and clay over weathered shale, with
  medium-strength shale below. No groundwater, only seepage. Piled retention to No. 38, no underpinning, anchors need
  the neighbours' consent. OSD is needed. Services are reticulated. No contamination.
- **Budget.** $10,800,000 ex GST, construction only. That is about $4,000/m2 of GFA, plausible.
- **Fees and invoices.** Unchanged from the base, and they match `fees.json` and `invoices.json` (including the d18
  arithmetic error). Architect: $424,000 to tender plus a $264,000 option, about 6.4% of the budget. Structural:
  $195,000 all in, about 1.8%. Both are at the top of the range for a $10.8M job but believable. `notes.md` keeps them
  on purpose, so no change.

## 3. Comparison with `expected/`

Every value in `profile.json` matches my reading, with the right sources. The changed keys are `det.rise_in_storeys`
3 (d01, d12), `hdr.scale.total_storeys` 3, `tower_levels` 2, `residential_units` 18 (d01, d13), `total_gfa_sqm`
2700, `car_parks` 30 (d01, d14, d15), `det.effective_height` 7.6, `det.sprinklered` false and `proj.budget_stated`
1080000000. Each is stated once, with no competing figure anywhere in the corpus.

I grepped all 18 documents for the six-storey values: six, five levels, 42, 55, 5,500, 24.5, 18.9, 7 m and 12.5 m.
None remain. No copied document (d02-d05, d08, d16-d18) mentions storeys, apartment numbers, parking numbers,
sprinklers, the bond or excavation depth. d08's OSD tank "under the basement ramp" and d05's "basement excavation"
spoil still fit one basement.

`derived` gives `det.type_of_construction` A. That is right for the building, but d12 contradicts it (finding 1).

**Checks the dispatcher asked for:**

- **HBCF in place of the strata building bond (d01): right.** The HBCF exemption for multi-storey buildings needs a
  rise of more than 3 storeys and two or more dwellings. The Strata Building Bond and Inspections Scheme covers the
  residential buildings above that line (4 or more storeys). At a rise of 3, HBCF applies and the bond does not.
  When a developer engages a builder, the builder takes out HBCF and must give the certificate before doing work or
  taking payment, so "before starting work" is fair. (Not checked against the instrument text: the shape's R4
  stays `verified: false`.)
- **Effective height 7.6 m, not sprinklered (d12): right.** 7.6 m is two floor-to-floor heights, e.g. a 4.5 m retail
  floor plus a 3.1 m residential floor. No document states floor heights, so nothing contradicts it. Class 2
  sprinklers start at a rise of 4. A 30-space car park is under the car-park threshold, and 600 m2 of retail
  triggers none.
- **Type A/B and the C2D4 note (d12): wrong.** See finding 1.
- **Geotech (d11): internally consistent but contradicts the survey.** See finding 2.

## 4. Stray triggers

I searched the corpus for every `triggers[].evidence` phrase in `shape.yaml` and the DOMAIN.md §10 phrases. Every
hit is a planned base denial: "no groundwater", "no underpinning", "Sydney Water sewer main in the rear lane",
street trees not affected, no classified road, no clause 4.6, the design review panel, public exhibition, strata
subdivision, "one building", "no staged construction" and the planner lodging. The only shape evidence present is the
planned `rise-three-storeys` wording in d01. The negation "Six storeys: ground-floor retail with five residential
levels above" appears nowhere. No new hits.

## 5. Title blocks

All ten rewritten documents keep the base title blocks, and the eight copies are identical. In the clean documents
(d01, d06, d07, d09, d12, d14, d15), the label lines come before the first `##`, there is one `#` heading equal to
`Title:`, and the numbers, revisions and dates match `document_register.json`. Rev B (d15) supersedes d14. The
adversarial d10, d11 and d13 carry their traps as the manifest plans.

## 6. Realism

The rewrites read as the base did: like the real documents. Three small exceptions are findings 4-6.

## Findings

1. **d12 `BCA report NBR-26-118 - 40-44 Halloran Street.md`, lines 33-36: the C2D4 note is wrong in law and
   contradicts `derived` det.type_of_construction A.** Table C2D2 does give Type A for Class 2 and Type B for Class 6
   at a rise of 3. But C2D4 (buildings of multiple classification) settles the building by applying the top storey's
   classification to every storey. Here the top storey is Class 2, so the whole building is Type A. The report
   instead says each part is "set by the rise in storeys of the whole building", which leaves the Class 6 part at
   Type B. A certifier would not write that. The repo's own `knowledge/clusters/determinants/VERIFICATION.md` and
   `knowledge/tables/type_of_construction.yaml` also describe C2D4 as the top-storey rule. A mixed type would need C2D5
   separation by a fire wall, and the parts here are separated by floors.
   Replace lines 33-36 with:
   > Effective height: 7.6 m. The effective height is less than 25 m. With a rise in storeys of 3, Table C2D2 gives
   > Type A construction for the Class 2 part and Type B for the Class 6 part. Under C2D4 the classification of the
   > top storey (Class 2) is applied to all storeys, so the whole building requires Type A construction.

   Make the matching edits outside the corpus: the manifest d12 `content` ("Type A construction for the Class 2 part
   and Type B for the Class 6 part, the building to be assessed under C2D4" becomes "Type A for the Class 2 part and
   Type B for Class 6 under Table C2D2; C2D4 makes the whole building Type A"), the shape `rise-three-storeys` `why`
   ("so C2D4 decides the building" becomes "and C2D4 (top-storey classification) makes the whole building Type A"),
   and the last sentence of `expected/notes.md`. Keep `verified: false` until someone reads it in the instrument.

2. **d11 `Geotechnical report BRG-26-0412 - 40-44 Halloran Street.md`, line 37 with lines 45-46 and 62: the rock
   depth does not fit the site fall.** All four boreholes are at the rear of the site (rear car park and rear yard).
   d03 says the ground falls about 2.3 m from Wattle Lane to Halloran Street, so the boreholes start about 2.3 m above
   the street. Bulk excavation at "3.8 m below Halloran Street" is therefore about 6.1 m below the borehole collars.
   With medium-strength shale at "about 4.0" m in those holes, the excavation would cut about 2 m into
   medium-strength rock. It would not stop "just above" it, and the pad footings would sit well below the top of the
   rock, not "just below" it. That changes excavation method and cost (rock hammering and sawing, not large
   excavators). The base value of 6.0 m fits the new design exactly: 6.0 − 2.3 = 3.7 m below the street, so the
   3.8 m excavation reaches the top of the medium-strength shale.
   Changes:
   - Line 37: `| Ashfield Shale | Moderately weathered to fresh shale, medium strength | about 6.0 |`
   - Under the table, add: "Depths are below the existing surface at the boreholes, at the rear of the site, which is
     about 2.3 m above Halloran Street."
   - Lines 45-46: "will extend through the weathered shale to about the top of the medium strength shale."
   - Line 62: "Pad footings founded on medium strength shale at or just below the base of the excavation ..."

   The 9 m boreholes (d09, d11) still reach about 3 m below bulk excavation level, enough for 1.5 m pile sockets. The
   manifest d11 `content` ("Ashfield Shale of medium strength from about 4 m") needs the same edit. Nothing scored
   changes.

3. **d01 line 64, d10 line 37, d13 line 88: verify that a 3-storey proposal still goes to the Local Planning Panel
   (`verified: false`).** I recall that the Local Planning Panels Direction refers Chapter 4 (former SEPP 65)
   development to the panel only when it is 4 or more storeys. If that is right, a 3-storey shop-top needs another
   reason: for example, a contentious application (ten or more unique objections) or a variation of more than 10%.
   The base reason (6 storeys) has gone, and no document gives a new one. `notes.md` keeps the panel on purpose, and
   `flag.panel` yes is stated in three documents, so scoring is safe. A Sydney PM would still ask why. If the
   Direction confirms the 4-storey line, add one sentence to d13 `## Determination`, for example: "Council advised
   at the pre-lodgement meeting that, given the objections it expects from the Halloran Street neighbours, it
   proposes to refer the application to the Local Planning Panel for determination." This keeps the variant to one
   change. The alternative is to flip `flag.panel` to no, which changes d01, d10 (and its $4,500 fee line) and d13.
   That decision is the owner's.

4. **d13 `Statement of environmental effects`, line 27: "(a rise in storeys of 3)" reads as written for a test.**
   Rise in storeys is an NCC term that planners do not use in an SEE, and d13 is not a source for
   `det.rise_in_storeys` (d01 and d12 are). Change to: "construct a three-storey mixed-use building, being shop-top
   housing: three ground-floor shops with 18 apartments on the two levels above." Drop the parenthesis from the
   manifest d13 `content` too.

5. **d01 line 19 and d06 line 22: "a ground-floor shop" (singular) against three shops elsewhere.** d01 line 19
   copies the shape evidence verbatim ("ground-floor shop with two levels of apartments above"). d06 says "18
   apartments over a ground-floor shop". Every other document says three tenancies. Change d01 to "ground-floor
   shops with two levels of apartments above" and d06 to "18 apartments over three ground-floor shops". If lint
   matches `says` verbatim, make the same edit in the manifest `carries` for d01 `hdr.subclass` and in the d06
   `content`. The shape evidence string can stay: it describes the trigger, not the text.

6. **d14/d15 line 19/20: "lift core (two lifts)" was copied from the 42-apartment base.** Eighteen apartments on two
   levels would normally get one lift; a PM would query a second lift on cost. Change to "lift core (one lift)" in
   both drawings, or leave it if the owner wants the base drawings untouched. It is not scored.

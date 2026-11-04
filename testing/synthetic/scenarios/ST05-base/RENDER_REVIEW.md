# Render review: ST05-base

Reviewer: independent (brief 04r), 2026-11-04. Corpus: 19 documents in `corpus/`.

## Verdict: **FIX**

Seven findings. Two of them matter. Finding 1: the SEE gives a reason for the DA that the other
documents contradict. Finding 2: the owner brief's "main works" leaves room for the owners to be in
the house for part of the job, and the key's `vacant` relies on that sentence. The rest are wording
that reads as written for a test, one setback inconsistency and one unplanned keyword. Of the
writer's three departures, (1) and (2) are accepted and (3) is finding 1.

## 1. Lint

`python testing/synthetic/tools/lint_corpus.py testing/synthetic/scenarios/ST05-base` → `ok`, exit 0.

## 2. What a PM concludes from the documents alone

- **Job:** a single-storey rear addition of 55 m2 (kitchen, living, dining) and internal alterations
  to a 140 m2 brick house built in 1958, for Tomas and Leah Ferreira at 41 Tallowwood Avenue, Concord
  NSW 2137 (Lot 23 DP 30418). The bathroom and laundry are redone and two loadbearing walls come out
  on steel beams. The 1980s skillion and laundry lean-to are demolished. The result is 195 m2. Class 1a,
  work type extend (with alterations), mid-range quality, the owners are the only decision makers and
  they move out (see finding 2).
- **Planning:** DA to the City of Canada Bay, determined under delegation, neighbour notification only,
  no cl 4.6. R2, LEP 2013, 8.5 m height and 0.5:1 FSR (proposed 0.35:1). The land is CDC-eligible. The
  SEE's reason for a DA does not hold up (finding 1). Not heritage, flood, bushfire, coastal, rail,
  contaminated or acid sulfate land.
- **Site:** 557.4 m2, 12.2 m frontage, 45.7 m deep, near level. Street parking only, no rear lane,
  a 1.0 m side path is the only way to the rear: urban constrained. Class M, no groundwater, shallow
  strip and pad footings to match the existing 0.5 m strip footing, no basement, no excavation next
  to neighbours. Sewer, water and gas mains in the street, no Sydney Water asset in the lot, no OSD
  now or required, gravity to the kerb. Natural gas kept for the cooktop. NCC climate zone 5. A
  frangipani at the back fence and a street brush box, neither affected. Hazmat survey negative for
  asbestos and lead; loose-fill register clear.
- **Procurement:** architect-led traditional lump-sum tender to three or four builders, HIA NSW Lump
  Sum contract, one continuous job. Budget $640,000 ex GST. Owner wants a start by 1 March 2027,
  stated only in the d04 email.
- **Consultants:** architect, surveyor, structural, civil/stormwater, geotechnical, hazmat and BASIX.
  A certifier is needed but has no document. No arborist, heritage or planner.
- **Fees:** architect $44,000 ex GST (9.5k/12.5k/19k/3k), $11,000 CA option, $210/h. Structural
  $6,800 ex GST (1.2k/0.8k/4.8k), $330 per visit. Civil $7,260 inc GST = $6,600 ex GST
  (2.2k/3.2k/1.2k), CCTV at cost. Geotechnical $3,200 ex GST, $420 per footing visit. Hazmat $2,300
  ex GST, $650 clearance option. BASIX $1,200 ex GST (950 + 250).
- **Invoices:**
  - KHA-INV-0412: $9,500 + $950 = $10,450 for Stage 1 at 100%.
  - CS-2291: $2,300 + $345 = $2,645. The total adds up, but the GST is 15%. The correct GST is $230.
  - 1088: $2,200 + $220 = $2,420 for the stormwater concept plan.

## 3. Compared with `expected/`

My conclusions match `profile.json`, `fees.json` and `invoices.json` on every keyed value. That
includes `gst_inc_normalised` on d07 and `gst_rate` (not `arithmetic`) on d18. The differences are
the doubts in findings 1 and 2. In both, the keyed value still comes out right, but a careful reader
stops on the sentence.

## 4. Stray triggers

The planned traps were all found where the manifest puts them:
- d02: CDC-eligible land
- d03: sewer main in the street, and the listed trees
- d07: OSD negated
- d11: fibre cement with no asbestos, and the loose-fill register
- d12: "excavate to a depth of 300 mm"
- d13: Dulux Heritage, flood test, "build over the footprint"
- d18: 15% GST

Also checked and harmless:
- "Stage 1 … Stage 4" in d05 and d17 is declared in `notes.md`.
- "lodge"/"lodgement" occurs in d01, d04, d05, d10 and d15. The key is `architect_lodges` = yes.
- "No public road works are required" (d07) is the planned negation for `flag.public_road`.
- "existing occupation" in d03 is surveyor's language for fences, and "occupation certificate" in
  d07 is a certificate. Neither says who lives in the house.
- "paved terrace" (d07, d13 to d16) is not a party-wall pattern. `party-wall` matches only
  `party|separating|common wall`.

Unplanned hits: finding 7 ("secondary dwellings").

## 5. Title blocks

Clean documents d01, d05, d06, d08, d09, d10, d14, d15, d16, d17 and d18 each have `Label: value`
lines before the first `##` and one `#` heading equal to `Title:`. Numbers, revisions and dates match
`document_register.json` (d14/d15 A-201 Rev A 6 May and Rev B 27 May; d16 KHA-SEE-01, 28 May). d04 is
the two-message email header its entry declares. Messy d02, d07 and d19 use the Markdown-table blocks
their entries declare. d02's `# Planning Certificate` matches the register title. Adversarial d03,
d11, d12 and d13 carry their planned traps, and their title blocks are clean and match the register.

## 6. The writer's three departures

1. **d09 "pre-demolition survey" for "pre-demolition and refurbishment survey": accept.** NSW
   hazmat consultants commonly title a survey before demolition and strip-out "pre-demolition". The
   scope line still covers every affected area, the roof space, the subfloor and the eaves. Keeping
   "refurbishment" out of a read document is a fair guard against the extend-versus-refurb conflict
   (DOMAIN.md §10). No change. Update the d09 manifest `content` to "pre-demolition survey" so the
   manifest and the corpus agree.
2. **"Title:" lines on d17 and d18, and "Subtotal $2,300.00 ex GST": accept.** This follows ST01
   finding 9. The `says` strings ("Subtotal $2,300.00", "Total $10,450.00 incl GST") are still
   contained verbatim. No change. Update both `title_block` entries to start with "Title: Tax invoice".
3. **No rear-setback row in the SEE table: not acceptable as it stands.** See finding 1. The missing
   row is a symptom of a deeper problem: the stated rear-setback failure is not credible on this lot.

## Findings

1. **d16 `Statement of environmental effects - 41 Tallowwood Avenue.md`, line 12: "A DA is required
   because the addition does not meet the Housing Code rear setback standard."** Nothing in the corpus
   supports this. The lot is 45.7 m deep. The addition extends the house only about 6.5 m and keeps
   "a large garden behind" (d13). The frangipani "near the back fence" is 16 m from the existing rear
   wall (d03), so roughly 9 to 10 m of yard remains. A single-storey addition with that much yard
   would meet any Housing Code rear setback. The compliance table then lists side setback, landscaped
   area and so on, but has no rear-setback row. So the one control the SEE says fails is the one it
   never shows. A PM, or council, would ask why this is not a CDC. Adding a row would mean inventing
   a Housing Code number (AGENTS.md: no invented rule numbers), and the geometry would still
   contradict it.
   **Change:** replace the second sentence of line 12 with "The owners have chosen a development
   application rather than a complying development certificate, so that the design is assessed against
   council's DCP." The pathway stays an unambiguous `da`, and the d02 CDC-eligible trap still has its
   counterweight. Update the d16 `carries` `says` for `hdr.cond.planning` to match. No table row is
   then needed. (If the scenario must show a Housing Code failure, choose one the documents support
   and add its row with the standard marked from the instrument. Rear setback is not it.)

2. **d01 `Owner brief - 41 Tallowwood Avenue.md`, line 22: "We will move out of the house for the
   duration of the main works. The owners will vacate the house for the duration of the works."** "Main
   works" implies some works happen while the owners are still in, which reads toward the
   `occupied-during-works` trigger. The second sentence switches to the third person in a first-person
   brief and repeats the first. It reads as pasted from the shape's negation list. The key's `vacant`
   rests on d01 alone.
   **Change:** replace both sentences with "We will move out of the house for the whole of the works,
   so it will be empty from the day the builder starts until handover." Update the d01 `carries`
   `says` for `hdr.cond.operational_constraints` to match.

3. **d13 `Architectural design statement - 41 Tallowwood Avenue.md`, lines 32-33: "The minimum side
   setback to the addition is 1.5 m, on the western side; the 1.0 m side path on the eastern side is
   kept."** If 1.5 m is the minimum and it is on the west, the eastern setback must be larger. Yet the
   existing house leaves only a 1.0 m path on the east. A reader cannot tell whether the addition's
   eastern wall is at 1.0 m or set in. The SEE says "1.5 m to the addition".
   **Change:** use "The addition is set 1.5 m from both side boundaries, inside the line of the
   existing eastern wall, so the 1.0 m side path on the eastern side is kept." This keeps the
   `wall-near-boundary` negation ("1.5 m") and agrees with d16.

4. **Realism: d01 sentences shaped for the answer key.** Each change keeps the trigger words. Update
   the matching manifest `says` with each one.
   - Lines 14-15: "The works are alterations and additions to an existing building (a single-storey
     brick dwelling). The house was built in 1958 and still has…" → "The existing building is a
     single-storey brick house. It was built in 1958 and still has…" This keeps `existing building`
     and "built in 1958".
   - Lines 19-20: "The house stays single storey: a single-storey rear addition, with no change to the
     front…" → "The house will stay single storey, with no change to the front…" The previous sentence
     already says "single-storey rear addition".
   - Line 53: "No basement or cellar. There are not multiple buildings: just the one house." → "No
     basement or cellar, and no other buildings on the lot: just the one house."

5. **d13, line 30: "The site is a standard suburban lot with no environmental sensitivities."** This
   is the same key-shaped phrase ST01 finding 8 flagged.
   **Change:** "The site is a standard suburban lot with no environmental sensitivities such as
   bushland, waterways or contamination." The d13 `says` for `hdr.cond.environmental_sensitivity`
   stays a substring, so no manifest change is needed.

6. **d03 `Detail and boundary survey report - 41 Tallowwood Avenue.md`, lines 30-31: "Narrow 12.2 m
   frontage with street parking only, no rear lane, and side access … along a 1.0 m path…"** This is a
   verbless fragment in an otherwise formal surveyor's report, and it reads as the `says` string
   dropped in.
   **Change:** "The lot has a narrow 12.2 m frontage with street parking only and no rear lane. Side
   access to the rear yard is along a 1.0 m path on the eastern side of the house." Update the d03
   `says` for `hdr.cond.access_constraints` to match.

7. **d02 `Planning certificate s10.7 - Lot 23 DP 30418.md`, line 38: "Permitted with consent: dwelling
   houses; dual occupancies; secondary dwellings; home businesses."** "Secondary dwelling" is the
   DOMAIN.md §10 granny-flat (T12) keyword. It is unplanned, and it sits in a document that is read
   for the profile. This is low risk, because it is a permitted-use list, and it is real council
   wording.
   **Change:** either drop the permitted-use sentence (item 2 still states the zone, height and FSR),
   or declare it as a d02 trap with `expect` "hdr.subclass stays house".

The remaining documents read like the real thing:
- the s10.7 certificate
- the hazmat proposal, report and invoice
- the geotechnical proposal and report (the test pit at the existing footing is a good touch)
- the structural, civil and BASIX quotes
- the drawing sheets
- the email thread

Their amounts, dates and sequence are consistent: survey, fee proposals, hazmat and geotechnical
fieldwork, invoices, design statement, Rev B "for DA lodgement", SEE, then the civil invoice.

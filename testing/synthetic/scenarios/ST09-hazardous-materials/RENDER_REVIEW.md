# Render review: ST09-hazardous-materials

Reviewer: independent (brief 04r), 2026-11-04. Corpus: 18 documents in `corpus/` (base 16 + d17, d18).

## Verdict: **FIX**

Five findings. No keyed profile value is at risk. Two matter:

- **Finding 1.** The defect report d08 is dated 14 April 2026. The survey d18 was sampled at the same
  opening-up and reported on 8 April. Yet d08 still calls the membrane and mastics "of unknown
  composition", says a survey "is needed", and leaves the asbestos removal out of Option B and the
  budget. A PM would read d08 as stale.
- **Finding 2.** The d18 trap is described wrongly. "Asbestos remediation" does not match the
  contamination trigger. Only "not a contaminated land assessment" does.

The variant differs from the base only where `changed_from_base` says. I checked this with a
file-by-file diff against `ST09-base/corpus`: d01, d03, d08, d09, d10 and d13 changed, and d17 and d18
were added. No "constructed in 2008", "2008 drawings" or "2019 survey found no asbestos" is left in any
document. The only "2008" left is the SEPP title in d02 ("SEPP (Exempt and Complying Development
Codes) 2008"). That is not a construction year, and it does not match the `existing_building_year`
triggers (`\b(?:built|constructed) in \d{4}\b`, `\byear (?:built|of construction|constructed)\b`).

## 1. Lint

`python testing/synthetic/tools/lint_corpus.py testing/synthetic/scenarios/ST09-hazardous-materials` →
`ok`, exit 0.

## 2. What a PM concludes from the documents alone

- **Job, planning, site, procurement, consultants:** as the base. The one change is that the building
  was constructed in **1986**, with the strata plan registered the same year. It is a 6-storey,
  42-apartment Class 2 strata building in Hurstville. All 42 balconies get exempt, like-for-like
  waterproofing renewal, tendered as an AS 4000 lump sum. The budget is $1,680,000 ex GST, and the
  committee wants a start by 1 February 2027.
- **Hazardous materials (the variant):**
  - The asbestos register lists bonded fibre cement soffits (d03, d08).
  - The d18 survey (sampled 24–25 March, reported 8 April) confirms chrysotile in the soffit sheets.
    No asbestos was found in the membrane or mastics.
  - All 42 balconies' soffits are to be removed by a licensed removalist. The SafeWork NSW notice and
    a clearance certificate per balcony come before membrane works (d09, d18).
  - Soffits are reinstated (d09).
  - d13 makes the licensed asbestos removalist a nominated subcontractor.
- **Consultants:** remedial building consultant, structural engineer and **hazardous materials
  consultant** (Fibrecheck). There is no environmental or contaminated-land input. The land is not
  contaminated (d02), and d18 says it is not a contaminated land assessment.
- **Fees:** as the base, plus Fibrecheck at $7,000 ex GST ($3,900 / $1,100 / $600 / $1,400). Further
  clearance visits are $180 each ex GST.
- **Invoices:** as the base. d14 is $26,950. d15 is $7,150 and carries the ABN typo. d16 is $7,480.

## 3. Compared with `expected/`

My conclusions match `profile.json` on every keyed value. That includes `det.existing_building_year` =
1986 (d01, d03, d08, d10, d18) and `det.potentially_contaminated_land` = false (d02). They also match
`invoices.json`, `procurement.json` (d17 → consultant.hazmat) and `document_register.json` for d17 and
d18.

`fees.json` d17 matches on its total, GST basis and the four items. It leaves out the $180 per-visit
rate (finding 4).

## 4. Stray triggers and the contamination check

Checked against the app's real triggers:

- **`det.potentially_contaminated_land`** (`knowledge/determinants.yaml`) has a single trigger,
  `contaminat`. In the corpus it hits only in two places:
  - d02 l.66, "no record that the land is contaminated" (base);
  - d18 l.23, "it is not a contaminated land assessment" (the planned trap).

  Both are explicit denials, so the worst Jev answer is `not_stated`. Neither can set true.
- **"Asbestos remediation"** (d18 l.50) **and "remediation"** (d01 title and l.7, d08 l.15) match no
  determinant trigger and no flag.
  - No `implied_by` pattern in `knowledge/profile/scope_defaults.yaml` covers remediat, asbestos,
    soffit or hazard.
  - `site.contamination` excludes "Asbestos and hazardous materials inside existing buildings"
    (`knowledge/clusters/structure/systems.yaml`).
  - `cost_remediation` is a client decision (`knowledge/rfp/skeleton.yaml`), not a reading.

  So this wording cannot set a contamination value or a remediation flag. `hdr.work_type` =
  remediation is keyed anyway.
- **Hazmat trigger evidence** appears where it is planned:
  - "constructed in 1986": d01, d03, d08, d10, d18;
  - the asbestos register on the soffits: d03 l.28, d08 l.54, d17, d18 l.20;
  - "of unknown composition": d08 l.55–56, and d17 in other words.

  No negation phrase remains.
- **Strip-out and demolition.** `\bdemoli(?:tion|sh)\b` still hits only d09 l.47, as planned. d09's
  new hazmat section does not use "demolish" or "strip-out", apart from the d18 and d08 phrase
  "before strip-out", which matches `\bstrip[- ]?out\b`. That phrase implies `site.demolition`, which
  the trigger adds anyway, so it is harmless here.
- **New §10 traps.** "No asbestos detected" (d18 l.37–38, l.44–46) refers to the membrane and mastic
  results, not the building, and no flag reads it. d17 and d18 add no flood, basement, bathroom,
  bedroom, kitchen or stairs word.

## 5. Title blocks

- d17 (clean) and d18 (adversarial) each have `Label: value` lines before the first `##`, and one `#`
  heading equal to `Title:`.
- Their numbers (FHM-P-0309; FHM-R-0309 rev 0) and dates (9 March, 8 April 2026) match
  `document_register.json`.
- d18 carries its planned trap.
- The six changed documents keep the base title blocks.

## Findings

1. **d08 `Defect investigation report PBC-R-26014-01.md` is stale against d18 (serious, realism and
   timeline).**
   - d08 is dated 14 April 2026. Its l.19–21 say the four balconies were opened on 24–25 March, the
     same days d18 (8 April) sampled them.
   - Yet d08 l.54–57 says the membrane and mastics "are of unknown composition" and that a survey "is
     needed before strip-out".
   - Option B (l.67–70) and the budget (l.76–79) do not mention licensed asbestos removal, soffit
     reinstatement or clearance.

   A PM would ask why the lead consultant ignored a survey received six days earlier.

   **Change:** replace l.54–57 with:

   > "7. **Hazardous materials.** The common property asbestos register lists bonded fibre cement
   > sheet in the balcony soffits, and the original bituminous membrane and mastic sealants were of
   > unknown composition. Fibrecheck Hazardous Materials sampled all three at the four opened
   > balconies; their survey FHM-R-0309 (8 April 2026) confirms asbestos in the soffit sheets and
   > found none in the membrane or mastics. The soffits of all 42 balconies are to be removed by a
   > licensed asbestos removalist, with clearance before membrane works."

   Append to Option B: "Remove the asbestos soffit sheets under licence and reinstate the soffits."
   Add to the budget's "This allows for" list: "licensed asbestos removal and clearance". Keep
   $1,680,000, as `notes.md` keeps the base budget.

   Update the d08 `content` and `why` in the manifest. The trigger's "unknown composition" evidence
   stays in d08 (now past tense) and in d17.

2. **The d18 `not-contaminated-land` trap description is wrong (manifest and `notes.md`).**
   - The manifest says "asbestos remediation" matches the contamination trigger `contaminat`. It does
     not. The only d18 hit is l.23 "not a contaminated land assessment".
   - "Remediation" reaches nothing in the profile. No determinant trigger, flag or `implied_by`
     pattern matches it, and `site.contamination` excludes building asbestos.

   **Change:** in the d18 trap `what` and in `notes.md` Traps, write: "States 'This survey covers
   building materials only; it is not a contaminated land assessment'; matches the contamination
   trigger `contaminat`. 'Asbestos remediation' (l.50) is the §10 'Remediation' word; it matches no
   trigger and tests only the Jev reading of the passage." Keep the expect.

3. **d18 `Hazardous materials survey FHM-R-0309.md` l.50–51 reads as written for the trap.** "We
   recommend asbestos remediation of the soffits: the bonded fibre cement soffit sheets ... are
   removed" puts a colon on a list item. It is also the only line in d18 over 110 characters.

   **Change:** "1. Asbestos remediation of the soffits: a licensed asbestos removalist removes the bonded
   fibre cement soffit sheets of all 42 balconies before or with the strip-out of the balcony above."
   Wrap it at about 100 characters. The phrase "asbestos remediation" is kept.

4. **`expected/fees.json` d17: the per-visit rate is missing.** d17 l.49 has "Further visits ...
   charged at $180 per visit, excluding GST". The key's d17 `prices` hold only the four stages. The
   base keys the equivalent rates for d05, d06 and d07 (for example d07 "Additional inspection, per
   visit" at 48000).

   **Change:** add `{"label": "Further clearance visits, per visit", "kind": "rate", "state":
   "priced", "amount_cents": 18000}` to d17 prices.

5. **d09 `Remedial specification PBC-SPEC-26014 Rev A.md` l.41: "Reinstate the soffits to match the
   existing".** The existing soffits are asbestos cement. A specification names the new material, and
   RW-101 Rev B shows no soffit detail.

   **Change:** "5. Once the balcony above is cleared, reinstate the soffits with new 6 mm fibre cement
   sheet (asbestos-free), fixed and painted to match the existing."

Everything else reads like the real thing: d17 is a typical hazmat proposal, and d18 has the right
method (PLM with dispersion staining, NATA lab), results table, extension to all balconies, and
limitations. The new d09 section and the d13 eligibility item are also sound. Dates are consistent:
d17 9 March → opening-up 24–25 March → d18 8 April → d09 2 June.

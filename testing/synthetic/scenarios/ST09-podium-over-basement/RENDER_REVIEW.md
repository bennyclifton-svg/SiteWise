# Render review: ST09-podium-over-basement

Reviewer: independent (brief 04r), 2026-11-04. Corpus: 18 documents in `corpus/` (16 from the base, d17
and d18 added).

## Verdict: **FIX**

Six findings. Two matter for scoring:

- **Finding 1.** d10 is a `flag.basement` "no" source. It says "no excavation, basement or ground works
  are involved". That is untrue: the podium works close car park bays, add temporary drainage in the
  basement, and (d18) may replace the pipe runs hung from the car park ceiling. It also says "no
  excavation" twice.
- **Finding 2.** Six documents mention the basement without a declared trap. Three of them (d09, d10
  and d13) are read for the profile.

The NCC class line (d08: "Class 2, with a Class 7a basement car park") and the $1,960,000 budget (d08)
match `profile.json`. No "no basement", "no podium", "at-grade" or $1,680,000 wording is left anywhere.
Fees and invoices are unchanged in amount, as the notes say.

## 1. Lint

`python testing/synthetic/tools/lint_corpus.py testing/synthetic/scenarios/ST09-podium-over-basement` →
`ok`, exit 0.

## 2. What a PM concludes from the documents alone

- **Job:** remediation of an existing building. The Owners - Strata Plan No. 912345 are renewing,
  like for like, the waterproofing of all 42 leaking balconies and of the ground-floor communal
  courtyard podium. The podium sits over a single-level basement car park (B1, 48 bays) and leaks into
  bays 14 to 18. The building is Kurrajong Apartments, 41 Kurrajong Parade, Hurstville NSW 2220. It is
  6 storeys with 42 units, built in 2008, and is out of warranty. Its NCC class is 2, with a Class 7a
  car park (d08), and its rise in storeys is 6. The building is fully occupied and the car park stays
  in use. The client is a strata scheme with a committee, a managing agent and two general meetings.
- **Planning:** exempt development, with no DA and no CC. The DBP Act applies. The land is Georges
  River, zone R4. There is no heritage, flood, contamination, bush fire or rail issue. The land is
  CDC-eligible, but that is not the pathway.
- **Site and structure:** the balcony slabs are sound. The podium slab (250 mm) carries the
  reinstated load without strengthening. There are no transfer structures and no OSD. Nothing is
  excavated and the basement structure is not altered. Delivery is one continuous stage.
- **Procurement:** traditional, with AS 4000-1997 and a lump sum to five remedial builders. A schedule
  of rates covers provisional quantities only. The budget is $1,960,000 ex GST, about $280,000 of it for
  the podium. The target start is 1 February 2027.
- **Consultants:** a remedial building consultant (lead), a structural engineer, a landscape architect
  and a hydraulic engineer. No geotechnical engineer.
- **Fees:**
  - Parapet: $81,000 ex GST, with a $6,800 disbursement, $650 per meeting and $210 per hour.
  - Halyard: $99,000 inc GST, which is $90,000 ex GST, with $242 per hour inc GST, which is $220 ex GST.
  - Karrak: $17,000 ex GST, with $480 per visit.
  - Groundswell: $9,400 ex GST (3.8k, 2.2k, 1.6k and 1.8k).
  - Cascade: $8,500 ex GST (2.9k, 4.1k and 1.5k).
- **Invoices:** the same as the base. PBC-INV-1187 is $26,950. 3307 is $7,150, and its ABN, ending 250,
  does not match the one in the quote, ending 205. PBC-INV-1192 is $7,480.

## 3. Compared with `expected/`

`profile.json`, `fees.json` and `invoices.json` match every conclusion above:

- `det.ncc_class` is ["2", "7a"] and `proj.budget_stated` is 196000000. The `rfps.json` budget is the
  same.
- The d17 and d18 stages and totals match.
- The d06 address and hourly rate carry the fixes from the base review.

The one key difference is in `derived` (finding 3).

**Diff against ST09-base:**

- d02, d04, d11, d12, d14, d15 and d16 are byte-identical to the base.
- Every other change is in a document listed in `changed_from_base`.
- The base review's findings 1 and 3 to 7 have been carried into this variant.

## 4. Stray triggers and keyword traps

Each planned trap appears where the manifest puts it:

| Doc | Trap | Where it appears |
|---|---|---|
| d01 | podium-over-basement | lines 17, 20 and 33 |
| d01 | rfp-cued-as-proposal | line 70 |
| d02 | cdc-eligible-land | line 46 |
| d03 | podium-over-basement | lines 20, 72 and 79 |
| d03 | bathroom-below | line 49, the only "bathroom" |
| d03 | bedroom-sill | line 56, the only "bedroom" |
| d05 | fee stages | lines 35 and 41-49 |
| d07 | podium-over-basement | lines 25 and 32 |
| d07 | fire-stairs | line 31, the only "stairs" |
| d08 | podium-over-basement | lines 30, 36, 59 and 89 |
| d08 | flood-test | lines 20, 22, 47 and 60 |
| d08 | remediation | line 15 |
| d09 | flood-test-hold-point | lines 78 and 104 |
| d09 | strip-out | lines 32 and 36 |
| d13 | schedule-of-rates | line 18 |
| d15 | abn-typo | line 10 |
| d18 | podium-over-basement-quote | line 17 |

The following words appear nowhere in the corpus: kitchen, ensuite, laundry, masonry, brickwork,
repointing, common wall, party wall, cladding, facade, scaffold, at-grade, groundwater, rail (outside
d02) and contaminat (outside d02).

The basement hits that the manifest does not plan are listed in finding 2.

Two hits carry over from the base and are planned:

- The `existing ... on-site detention` OSD wording in d09, line 54.
- The `excavat ... adjoining boundaries` wording in d10, line 64.

## 5. Title blocks

Clean documents:

- d10, d11, d12 and d17 each have `Label: value` lines before the first `##` and one `#` heading equal
  to `Title:`. Their number, revision and date match `document_register.json`. RW-101 Rev B supersedes
  Rev A.
- d14 and d16 have the same structure.
- d04 has its email header.

Messy documents d02, d06 and d15 use table blocks.

Adversarial documents d01, d03, d05, d07, d08, d09, d13 and d18 have readable label lines that match
the register.

## Findings

1. **d10 `Structural assessment KCE-R-2611-01.md`, lines 64-65.** The text reads "no excavation,
   basement or ground works are involved ... The works include no excavation; the existing basement
   car park structure is not altered." Basement works are involved: d09 Podium item 6 closes bays and
   adds temporary drainage, and d18 item 2 may replace the pipe runs in the car park. "No excavation"
   is also said twice. This passage is a source for `flag.basement` and `flag.excavation_adjoining`.
   **Change:** replace both lines with: "The works are confined to the balconies and the podium
   surface, with no excavation or ground works and nothing done near the adjoining boundaries or
   neighbouring structures. The existing basement car park structure is not altered." The
   `excavat ... adjoining` trigger still matches, at about 55 characters apart. Update the d10
   `content`.

2. **Unplanned `\bbasement` hits.** The basement appears without a declared trap in:
   - d05, line 22;
   - d06, line 19;
   - d09, lines 17 and 75;
   - d10, lines 17, 49 and 65;
   - d13, lines 15 and 60;
   - d17, line 18.

   d09, d10 and d13 are read for the profile. The wording is right for the job, so keep it.
   **Change:** add a `podium-over-basement` trap to d05, d06, d09, d10, d13 and d17 in
   `manifest.json`, each with `"expect": "flag.basement is not yes"`. Update the trap list in
   `expected/notes.md` to match.

3. **`expected/profile.json`, `derived`: `det.ncc_class` ["2"].** This contradicts the keyed value
   ["2", "7a"]. The 7a part is read from d08 and cannot be derived from "apartments".
   **Change:** remove the `det.ncc_class` entry from `derived` in this variant, and say why in
   `notes.md`. Keep the `det.existing_building` entry.

4. **d01 `Consultant brief`, lines 55-56, against d05 lines 63-65 and d06 lines 29-30.** Both
   competing proposals, dated 23 and 25 February, "understand the owners corporation will engage a
   landscape architect and a hydraulic engineer directly". Nothing they could have read says so: the
   brief names only a structural engineer.
   **Change:** after d01 line 56, add: "For the courtyard podium, the owners corporation will engage a
   landscape architect and a hydraulic engineer directly; the consultant coordinates their input."
   Update the d01 `content`.

5. **d09 `Remedial specification`, contradictions with the podium scope.**
   - Line 54 says "The existing building stormwater system ... is not altered". But Podium item 5 adds
     overflows, and d18 may replace the pipe runs and connections in the car park.
     **Change:** "3. Apart from the podium drainage below the courtyard, the existing building
     stormwater system, which has no on-site detention, is not altered." The OSD trigger wording is
     unchanged.
   - Lines 29-30, Delivery, say "all 42 balconies are renewed ... in one continuous stage" and leave
     out the podium.
     **Change:** "all 42 balconies and the courtyard podium are renewed under one contract in one
     continuous stage."

6. **Realism and edit seams.**
   - **d08, line 85:** "for Option B, balconies and podium (construction cost of the works including
     preliminaries and provisional sums)" reads as a definition written for the key.
     **Change:** lines 85-87 → "We recommend the owners corporation adopts a construction budget of
     $1,960,000 excluding GST for Option B. This allows for the remedial works to all 42 balconies,
     about $280,000 for the courtyard podium, preliminaries, a provisional sum for concrete repairs
     found on opening up, and a 10% contingency. It excludes consultant fees."
   - **Unwrapped lines from the edit:** d01 lines 17 and 44, d03 line 18, d07 lines 32 and 49, d08
     lines 85-89 and 114, and d10 lines 64-65 run to 130-200 characters. Every other line wraps at
     about 105 characters, so the seams show where text was edited.
     **Change:** rewrap these lines to match the rest of the document.

Everything else reads like the real thing:

- The landscape and hydraulic proposals (d17, d18) are realistic in scope and price, and their dates
  follow from a direct engagement.
- The car park leak log, the podium findings and the podium slab check read like real documents.
- The fee and GST arithmetic is correct.

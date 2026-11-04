# Render review: ST01-base

Reviewer: independent (brief 04r), 2026-10-30. Corpus: 17 documents in `corpus/`.

## Verdict: **FIX**

Nine findings. Two of them matter for scoring. Finding 1 is an unplanned keyword trap in a document
that carries `hdr.scale.storeys`. Finding 2 is a neighbour's tree that makes the key's arborist "no"
doubtful to a PM. The rest are smaller ambiguities and wording that reads as written for a test.

## 1. Lint

`python testing/synthetic/tools/lint_corpus.py testing/synthetic/scenarios/ST01-base` → `ok`, exit 0.

## 2. What a PM concludes from the documents alone

- **Job:** a new two-storey, four-bedroom detached house with a double garage, for Dan and Mai Nguyen at
  27 Corella Rise, Denistone NSW 2114 (Lot 14 DP 1302865). City of Ryde, zone R2. Vacant, grassed lot
  of 650.2 m2 with a gentle 0.9 m fall. GFA 298 m2 against a 300 m2 target. Three bathrooms plus a
  powder room. Class 1a with a Class 10a garage. Quality is high. The owners are the single
  decision-makers.
- **Planning:** a DA to council, determined under delegation, with neighbour notification only. The
  land is CDC-eligible, but the design misses the Housing Code side setback. No cl 4.6 variation.
  Not heritage, flood, bushfire, coastal, rail, contaminated or acid sulfate land.
- **Site:** Class M (AS 2870). No groundwater to 3 m. Raft slab, no basement, no excavation next to
  neighbours. Sewer and water mains are in the street, with no Sydney Water asset in the lot. No OSD.
  Gravity drainage to the kerb. The house is all-electric (no gas). NCC climate zone 5. The blue gum
  is retained. A jacaranda stands on No. 25 next to the northern boundary (see finding 2).
- **Procurement:** an architect-led traditional lump-sum tender on the HIA NSW Lump Sum Contract.
  Budget $1,450,000 ex GST. The owner wants a start of 1 February 2027, stated only in the d04 email.
- **Consultants:** architect, surveyor, structural, civil/stormwater, geotechnical and BASIX/NatHERS.
  A certifier will be needed, but no certifier document is in the corpus. A PM would ask about an
  arborist (finding 2).
- **Fees:** architect $80,000 ex GST (18/22/34/6k), with a $15,000 contract administration option
  and a $220/h rate. Structural $8,000 ex GST (1.5k + 6.5k, certificate included, $350 per visit).
  Civil $9,900 inc GST, which is $9,000 ex GST (3k/4.5k/1.5k). Geotechnical $3,800 ex GST, with
  footing inspections at $440 per visit. BASIX $2,000 ex GST (1,650 + 350).
- **Invoices:**
  - RG-0418: $3,800 + $380, stated as $4,280. The correct total is $4,180, so this is an
    arithmetic error.
  - INV-2307: $18,000 + $1,800 = $19,800 for Stage 1 at 100%.
  - 5521: $1,500 + $150 = $1,650 for concept and DA structural advice.

## 3. Compared with `expected/`

My conclusions match `profile.json`, `fees.json` and `invoices.json` on every keyed value. That
includes the `arithmetic` flag on d15 and `gst_inc_normalised` on d07. The only differences are the
doubts in findings 2 to 4: places where a careful reader could reach a different value from the key.

## 4. Stray triggers

The planned traps were all found where the manifest puts them:
- d02: CDC-eligible land
- d03: sewer main in the street
- d07: OSD negated
- d10: "excavate to a depth of 300 mm" and H2
- d11: garden beds and "No demolition"

Hits the manifest does not plan are findings 1, 4 and 5. "Lodge" (trap 6) occurs in d01, d05, d11,
d13 and d14. That is harmless because the key is `architect_lodges` = yes. d14's "public exhibition"
is negated, and it is a planned carry for `flag.community`.

## 5. Title blocks

Clean documents d01, d05, d06, d08, d09, d12, d13 and d14 each have `Label: value` lines before the
first `##` and one `#` heading equal to `Title:`. Their numbers, revisions and dates match
`document_register.json`.

d04 is an email header, as its manifest entry says. Messy documents d02, d07 and d17 use the
Markdown-table blocks their manifest entries declare. Adversarial documents d03, d10 and d11 carry
their planned traps. For d15 and d16, see finding 9.

## Findings

1. **d11 `Architectural design statement - 27 Corella Rise.md`, line 22: "under a single-storey garage
   roof that ties into the main roof".** This is DOMAIN.md §10 trap 13: `storeys` matches
   "single-storey garage" on a two-storey house. The manifest does not plan it, and d11 is a source
   for `hdr.scale.storeys`. It also contradicts d01 ("Double garage (2 spaces) under the main roof").
   **Change:** replace with "under a lower roof that ties into the main roof".

2. **d03 `Detail and boundary survey report - 27 Corella Rise.md`, line 54: T2 "Jacaranda … On No. 25,
   next to the northern boundary | 8 m | 7 m".** A neighbour's tree with a 3.5 m canopy radius sits at
   the boundary. The house is 1.2 m (ground floor) and 1.8 m (first floor) off that side. A PM would
   doubt d11's "No neighbouring or street trees are affected" and ask for an arborist. Yet the key
   has `flag.arborist_offsite_trees` = no, and the base must not look like the `trees-affected`
   variant.
   **Change:** set the location cell to "On No. 25, near its rear corner, about 8 m from the common
   boundary".

3. **d07 `Brennan Civil - stormwater fee - Corella Rise.md`, lines 28-29: "No public road works beyond
   the standard vehicle crossing."** The hedge says there are some road works. A new crossing is
   road-reserve work under a Roads Act s138 consent (DOMAIN.md §2 `site.public-domain`). A reader
   could answer `flag.public_road` yes, but the key says no.
   **Change:** use "No public road works are required." The exclusions already hand the crossing to
   the builder. Update the d07 `carries` `says` for `flag.public_road` to match.

4. **d09 `BASIX and NatHERS fee proposal - Thermacheck.md`, lines 19-20: "NCC climate zone 5 (NatHERS
   climate zone 56)".** The sentence that carries `det.climate_zone` holds two zone numbers, and the
   key accepts only "5". This ambiguity is unplanned.
   **Change:** use "The site is in NCC climate zone 5." and update the d09 `carries` `says` to match.
   d11 still carries the value. Alternatively, keep the sentence and declare it a planned trap.

5. **d01 `Owner brief - 27 Corella Rise.md`, line 69: "No Connecting with Country or Aboriginal
   cultural heritage input is required for this lot."** "heritage" is an unplanned trap 12 hit in a
   document that is read for the profile. No owner writes this sentence.
   **Change:** use "We do not need any Connecting with Country input for this lot." This keeps
   `flag.country` = no.

6. **d13 `Site plan A-101 Rev B.md`, line 27: "side setbacks 1.2 m ground floor, 1.8 m first floor".**
   This contradicts d14 ("1.5 m south, 1.8 m north") and its own revision note ("first-floor northern
   side setback increased").
   **Change:** use "side setbacks 1.2 m ground floor; first floor 1.5 m south, 1.8 m north".

7. **d10 `Geotechnical report RG-26-0388 - 27 Corella Rise.md`, lines 64-66: "…this is a timber
   treatment matter and does not change the site class."** The report explains the trap's answer.
   This reads as written for a test, and a geotechnical engineer would not comment on framing
   treatment this way.
   **Change:** replace lines 63-66 after "Building Technology File 18." with "We understand the frame
   will be H2 treated pine with termite management to AS 3660.1; termite protection is outside the
   scope of this report." This keeps H2 next to the site class.

8. **Realism: sentences shaped for the answer key.** Each change keeps the signal. Update the
   matching manifest `says` with each one.
   - d01 line 15: drop "(single owner)". The sentence before it already says this.
   - d01 line 27: "New build on a vacant lot." → "It will be a new build on our vacant lot."
   - d01 line 67: "HIA NSW Lump Sum Contract with the builder." → "We expect to sign the HIA NSW Lump
     Sum Contract with the builder."
   - d07 line 28: "No NorBE assessment applies." NorBE applies only in the Sydney drinking water
     catchment, which Ryde is not in, so a civil engineer would not raise it. Use "The site is
     outside the drinking water catchment, so no NorBE assessment applies."
   - d11 line 16: "A standard suburban lot with no environmental sensitivities." → "The lot is a
     standard suburban lot with no environmental sensitivities such as bushland, waterways or
     contamination."

9. **d15 `Tax invoice RG-0418.md` and d16 `Tax invoice INV-2307 Harbourline.md` (clean): no `Title:`
   line, and the first line ("Tax invoice") differs in case from the `#` heading ("Tax Invoice").**
   The manifest declares this block. Neither document is in the register, so this is low priority.
   **Change:** change line 1 to "Title: Tax invoice" and the heading to "# Tax invoice" in both,
   unless the invoice reader is meant to see a bare first line.

Everything else reads like the real thing. The s10.7 certificate (d02), the SEE (d14), the survey
report, the fee proposals and the invoices are credible in form, amounts and dates.

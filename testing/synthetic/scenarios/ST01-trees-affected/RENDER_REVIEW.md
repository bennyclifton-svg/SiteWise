# Render review: ST01-trees-affected

Reviewer: independent (brief 04r), 2026-10-30. Corpus: 19 documents in `corpus/` (base 17 + d18, d19).

## Verdict: **FIX**

Six findings. None of them changes a keyed value. Two matter:

- **Finding 1.** The design statement and the AIA both say the arborist's measures are on the DA
  drawings. Site plan Rev B, the DA issue, does not show them.
- **Finding 2.** d18 and d19 plainly propose root pruning. The key does not say what
  `flag.arborist_pruning` should be, and that flag drives an arborist RFP line.

The variant differs from the base only where `changed_from_base` says (d03, d11, d14 changed; d18, d19
added). I checked this with a file-by-file diff against `ST01-base/corpus`. No document still says that
no trees are affected. The AS 4970 figures are internally consistent (see section 5).

## 1. Lint

`python testing/synthetic/tools/lint_corpus.py testing/synthetic/scenarios/ST01-trees-affected` →
`ok`, exit 0. Run from the repo root. Run from the scenario folder, the base path does not resolve and
it gives false ABN/lot errors.

## 2. What a PM concludes from the documents alone

- **Job, planning, site, procurement:** as the base. A new two-storey, 4-bed Class 1a house with a
  Class 10a garage, GFA 298 m2, on a vacant 650.2 m2 R2 lot at Ryde. The application is a DA under
  delegation (the design fails the Housing Code side setback). The site is Class M, with no basement
  and no OSD. Gravity drainage goes to the kerb. The house is all-electric. Procurement is a
  traditional HIA lump-sum tender, budget $1,450,000 ex GST, and the owners want to start on
  1 February 2027.
- **Trees (the variant):**
  - Tree 1 is the blue gum on the lot. It has a major encroachment of 22% of its TPZ, and the
    terrace edge comes inside its SRZ. It is retained by a pier-and-suspended-beam terrace edge.
  - Tree 2 is the neighbour's jacaranda at No. 25. It has a minor encroachment of about 9% of its
    TPZ, outside the SRZ, from the driveway cut. It is retained by hand digging and root pruning
    under supervision.
  - There are two hold points.
  - Both arborist flags are **yes**. The project needs an arborist package, site.trees retain, and
    tree protection in the cost plan and the program.
- **Consultants:** architect, surveyor, structural, civil, geotechnical, BASIX/NatHERS, and
  **arborist**. A certifier is still needed but has no document.
- **Fees:** as the base, plus the arborist at $3,400 ex GST ($1,900 / $700 / $800).
- **Invoices:** as the base. RG-0418 has an arithmetic error ($4,280 stated against $4,180).
  INV-2307 is $19,800 and 5521 is $1,650.

## 3. Compared with `expected/`

My conclusions match `profile.json`, `fees.json` (including d18) and `invoices.json` on every keyed
value.

`hdr.cond.environmental_sensitivity` = standard is still defensible. Retained garden trees on a suburban
lot are not an environmental sensitivity in that taxonomy's sense.

The only gaps are keys that the corpus now evidences but the key does not mention (finding 2).

## 4. Stray triggers

The `trees-affected` evidence phrases appear in d11, d14, d18 and d19, as planned.

Neither negation phrase appears anywhere. "No trees on or next to the site are affected" is gone from
d11, and no document says the blue gum is unaffected. Site plans d12 and d13 still say only "blue gum
... to be retained". That is consistent with the variant.

The other triggers (cdc, bushfire, flood, heritage, OSD, sewer-in-lot, basement, pool, gas, old
structure, acid sulfate, coastal, sloping) hit only the planned base traps and negations.

The new documents introduce no §10 trap. "Cut ... 400 mm below existing ground" does not match
`excavat\w* to a depth`.

These new hits are not planned:

- **`arborist_pruning` triggers.** These phrases match the trigger:
  - d18 l.48: "root pruning"
  - d19 l.26, l.69, l.77: "pruned"
  - d19 l.102: "prunes roots"
  - d19 l.21: "No tree removal is proposed"

  This is finding 2.
- **"Piers".** d11 l.80, d14 l.98 and d19 l.73: "bored piers with a suspended beam". This could read
  as `substructure.piles`. That is defensible but unkeyed (finding 5).

## 5. Arborist figures (AS 4970)

These are all consistent:

- **TPZ radius** = 12 × DBH:
  - 1.20 m gives 14.4 m. The area is π·14.4² = 651 m2. This is under the 15 m cap.
  - 0.45 m gives 5.4 m. The area is 91.6 m2, stated as 92 m2.
- **SRZ radius** = (D·50)^0.42 × 0.64:
  - 3.9 m implies a buttress diameter D ≈ 1.48 m.
  - 2.6 m implies D ≈ 0.56 m.

  Both are plausible against the DBH, but neither D is stated (finding 4).
- **Jacaranda geometry.** The trunk is 2.0 m from the boundary, so the TPZ crosses the boundary by
  3.4 m and the SRZ by 0.6 m, both as stated. The driveway is 1.8 m inside the boundary, which puts it
  3.8 m from the trunk. That gives:
  - an encroachment width of 5.4 − 3.8 = 1.6 m, as stated;
  - a chord of 2·√(5.4² − 3.8²) = 7.7 m, stated as "about 7.5 m";
  - a segment area of 8.4 m2 = 9.2% of the TPZ, stated as "about 9%".

  It is under 10% and outside the SRZ, so the encroachment is minor, as stated.
- **Blue gum.** The terrace edge is 3.3 m from the trunk, inside the 3.9 m SRZ. With the encroachment
  above 10% and inside the SRZ, it is major. A 22% total for the house at the 11 m rear setback (7.8 m
  from the trunk) plus the terrace is plausible. "Last 1.5 m of the slab on piers" covers the 0.6 m of
  SRZ intrusion.
- **Cross-document checks.**
  - The survey (d03) and the AIA (d19) agree on location, height and spread for both trees.
  - d11 and d14 match the AIA on the design response (piers and beam, hand digging, fencing).
  - The d18 stages match the d19 hold points.

## 6. Title blocks

- d18 and d19 are clean. Their label lines come before the first `##`, and each has one `#` heading
  equal to its `Title:`.
- Their numbers (CLA-0927; CLA-AIA-0927 rev 1) and dates match `document_register.json`.
- d03, d11 and d14 keep the base title blocks unchanged.

## 7. Realism

- d19 reads like a real consultant's AIA.
- d18 is close, but it states the AIA's conclusions before any measurement (finding 3).

## Findings

1. **`Site plan A-101 Rev B.md` (d13) contradicts d11 and d19 (serious).**
   - d11 l.82–83 says the AIA's "recommendations are shown on the drawings".
   - d19 l.83–84 says the changes "will be shown on the DA drawings".
   - Rev B, "Amended for DA lodgement" five days after the AIA, shows only "Existing blue gum ... to
     be retained". It shows no jacaranda, no TPZ or SRZ, no protection fencing and no pier-and-beam
     edge.

   A PM would raise an RFI.

   **Change:** add d13 to the manifest's `changed_from_base` with the reason "DA site plan shows the
   arborist's tree protection". In d13:
   - after l.20, add:
     - "- Neighbour's jacaranda on No. 25 near the northern boundary, with its TPZ and SRZ"
     - "- Tree protection zones, protection fencing and the pier-and-beam rear terrace edge from
       arboricultural impact assessment CLA-AIA-0927"
   - change the Rev B description (l.42) to "Amended for DA lodgement: first-floor northern side
     setback increased; rainwater tank moved; tree protection from arborist's assessment added".

   Leave d12 (Rev A, 4 May) as is. It predates the arborist.

2. **The pruning flag is evidenced but not keyed (serious for RFP scoring).**
   - d18 l.48 has "root pruning".
   - d19 l.26 ("hand digging and root pruning under supervision"), l.76–77 ("Roots up to 50 mm may
     be cleanly pruned by the project arborist") and l.102 propose root pruning.
   - `knowledge/profile/flag_readings.yaml` `arborist_pruning` fires on `\bprun`. It asks whether
     pruning, removal or clearance is proposed, and it drives `knowledge/rfp/arborist.yaml` `pruning`.

   **Change:** add `flag.arborist_pruning` to `expected/profile.json`: `{"value": "yes", "sources":
   ["d18","d19"], "read_by_design": true}`. Note in `notes.md` that root pruning is proposed while
   no canopy pruning or removal is.

   If the owner rules that root pruning is not "pruning" for this flag, record "no". Do not leave it
   unkeyed.

   Also consider keying `flag.arborist_offsite_access`. d18 l.75–76 and d19 l.34–35 say the
   jacaranda is assessed from the client's side, so no permission is confirmed or needed.

3. **`Arborist fee proposal - Canopy Line.md` l.25–27 states measured conclusions before any
   measurement (realism).** A fee proposal cannot yet know that the slab edge is inside the SRZ.

   **Change:** l.23 should end "...close to the proposed works. From the survey and site plan Rev A:".
   Keep both evidence phrases as written, so the carried wording is unchanged. Make the first bullet a
   sentence like the second: "- The driveway cut encroaches into the tree protection zone of the
   neighbour's jacaranda at No. 25."

4. **`Arboricultural impact assessment CLA-0927.md` §3–4: the SRZ cannot be checked.** l.30–31 says
   the diameter above the root buttress was measured, but no value is given, and the SRZ depends on it.

   **Change:** add a column "Diameter above buttress" to the §4 table, with 1.48 m (Tree 1) and
   0.56 m (Tree 2).

5. **Piers are introduced without a key decision (minor).** d11 l.79–80, d14 l.98 and d19 l.73 put
   the terrace edge on "bored piers". A reader may add `substructure.piles` and a piers cost line. The
   expected scope and cost plan have neither, and the structural fee (d06, March) predates the change.

   **Change:** none to the corpus. Add a line to `notes.md` saying whether a piles work item for the
   terrace edge is accepted or ignored by the scorer.

6. **The d19 filename uses the proposal number (minor).** `Arboricultural impact assessment
   CLA-0927.md` carries CLA-0927, which is d18's proposal number. The report number is CLA-AIA-0927.
   An app reading the filename may register the wrong number.

   **Change:** rename it to `Arboricultural impact assessment CLA-AIA-0927.md` (manifest d19
   `filename` too). Or keep it as a deliberate, realistic trap and add it to d19 `traps`.

Optional: `Statement of environmental effects` l.16 lists the documents to read with the SEE but omits
the AIA. Add ", the arboricultural impact assessment and tree protection plan" after "the survey".

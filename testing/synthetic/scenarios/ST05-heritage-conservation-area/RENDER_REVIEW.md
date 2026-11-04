# Render review: ST05-heritage-conservation-area

Reviewer: independent (brief 04r), 2026-11-04. Corpus: 22 documents in `corpus/`. These are the base
d01–d19, with d02, d05, d13, d15 and d16 changed, plus the added d20, d21 and d22.

## Verdict: **FIX**

There are two findings. Finding 1 matters: the two added fee proposals each state an hourly rate that
`fees.json` does not hold. The fee reader will find a priced rate in d20 and in d21 that the key does
not expect. Finding 2 is a realism point in the d15 revision note. The heritage change is coherent
across d02, d05, d13, d16, d20, d21 and d22. No read document still denies heritage or implies that
complying development is available. Every base amount is unchanged.

## 1. Lint

`python testing/synthetic/tools/lint_corpus.py testing/synthetic/scenarios/ST05-heritage-conservation-area`
returned `ok` with exit code 0. It was run from the repo root.

## 2. What a PM concludes from the documents alone

- **Job:** a single-storey 55 m2 rear kitchen, living and dining addition to a 140 m2 brick house
  built in 1958. The house goes to 195 m2 after the works. The bathroom and laundry are redone and two
  loadbearing walls are removed on steel beams. The owners are Tomas and Leah Ferreira, at
  41 Tallowwood Avenue, Concord NSW 2137 (Lot 23 DP 30418). The council is the City of Canada Bay and
  the zone is R2. The lot is 557.4 m2 with a 12.2 m frontage and no rear lane. Side access is a 1.0 m
  path.
- **Classification and terms:** Class 1a, residential house, extend. Finishes are mid-range. The owners
  are the only decision makers and the house will be vacant during the works. The job is single-stage
  and has one building.
- **Heritage:** the site is within the Tallowwood Estate Heritage Conservation Area (LEP Schedule 5).
  The land is not a heritage item. The house is a neutral infill building, not a contributory one
  (d02, d13, d16 and d22). Council's heritage officer gave pre-DA advice: keep the Federation
  streetscape intact and use recessive materials. The addition sits behind the ridge and cannot be
  seen from the street. The heritage impact statement (HIS) finds the impact acceptable.
- **Pathway:** DA to the City of Canada Bay. The s10.7 certificate says Housing Code complying
  development may not be carried out because the land is in an HCA (d02 item 4), and the SEE agrees
  (d16). The architect lodges on the Portal (d05, and the d21 exclusions). The town planner writes the
  SEE. There is no clause 4.6 variation and no panel. Neighbours are notified, with no exhibition.
- **Site:** not bushfire, flood, coastal, rail, contaminated or acid sulfate land. There is no OSD, no
  stormwater quality treatment, no NorBE, no public road works and no authority asset in the lot. The
  site is Class M, with no basement and no groundwater. No trees are affected. The climate zone is 5 and
  BASIX applies. Gas is kept.
- **Consultants:** architect, structural, civil, geotechnical, hazmat, BASIX, surveyor, heritage
  consultant, town planner, and the certifier for the CC.
- **Budget and contract:** $640,000 ex GST for construction, under the HIA NSW Lump Sum contract. The
  procurement route is a traditional tender to three or four builders.
- **Fees:**
  - Architect: $44,000 ex GST. Contract administration is an $11,000 option and the rate is $210/h.
  - Structural: $6,800 ex GST.
  - Civil: $7,260 inc GST, which is $6,600 ex GST.
  - Geotechnical: $3,200 ex GST.
  - Hazmat: $2,300 ex GST.
  - BASIX: $1,200 ex GST.
  - Heritage: $4,700 ex GST, plus **$185/h** for further work.
  - Planner: $6,500 ex GST, plus **$195/h** for additional work.
- **Invoices:**
  - KHA-INV-0412 is $9,500 + $950 = $10,450.
  - CS-2291 is $2,300 + $345 = $2,645. The GST is at 15%.
  - 1088 is $2,200 + $220 = $2,420.

## 3. Comparison with `expected/`

- **`profile.json`:** agrees with every conclusion above. `det.heritage_status` is
  `conservation_area`, from d02, d13, d16 and d22. `hdr.cond.planning` is `da`, from d01, d05 and d16.
  - d20 and d21 also state the HCA, and d21 states that a DA is needed. These are extra sources that
    agree with the key, so they do no harm.
  - `hdr.cond.environmental_sensitivity = standard` comes from d13 line 30. That is correct: the
    taxonomy has no option for a built-heritage area.
- **`invoices.json`:** unchanged from the base and agrees.
- **`fees.json`:** agrees on every total and stage, except the hourly rates in d20 and d21
  (finding 1). The key holds a rate row for d05 ($210) and for the per-visit rates in d06 and d08, but
  none for d20 or d21.

**Diff against ST05-base.**
- Only d02, d05, d13, d15 and d16 differ, and only as `changed_from_base` says.
- The planning certificate negation is replaced by the HCA listing, and item 4 now says complying
  development is not available.
- The architect's scope adds the heritage-sensitive schedule and hands the SEE to the planner.
- In d13, "Dulux Heritage" is dropped. The base phrase "1940s to 1960s houses" is replaced by the
  Federation streetscape.
- The SEE's old line "owners have chosen a DA rather than a CDC" is replaced by "complying development
  is not available".
- The other 14 base documents are byte-identical to the base.

## 4. Stray triggers and keyword traps

I searched the corpus for "heritage", "conservation", "complying", "CDC", "Federation", "streetscape",
"character", "exempt" and "certifier", for the `shape.yaml` trigger evidence and for DOMAIN §10.
- **Heritage words:** they appear only in d02, d05, d13, d15, d16, d20, d21 and d22. None of the
  unchanged documents (d01, d03, d04, d06–d12, d14 and the invoices) mentions heritage or
  complying development.
- **The two planned CDC negations:** d02 item 4 ("may not be carried out … within a heritage
  conservation area") and d16 line 12 ("complying development is not available").
- **"certifier":** appears only in the d05 exclusions, where it means the certifier for the CC.
- **"exempt development may be carried out":** d02 item 5 is ordinary certificate wording and does not
  name the Housing Code.
- **Base traps:** "flood test", "build over the footprint", "sewer main in the street", "OSD is not
  required" and "built in 1958" are all planned and unchanged.
- **"Lodge" trap:** the d21 exclusion "Lodgement … stays with the architect" agrees with
  `flag.architect_lodges = yes`.

No unplanned trigger evidence was found.

## 5. Title blocks

- **Clean documents:** d05, d16, d20, d21 and d22 have label lines before the first `##` and one `#`
  heading that equals `Title:`. d15 has the drawing label lines. In every one, the number, revision
  and date match `document_register.json`. d22 uses `Report date:`, which `markdown_identity.go` reads
  as a date. d15 Rev B supersedes d14.
- **Messy document:** d02 keeps the table identity block as planned. Its LGA sentence ends with a full
  stop and is followed by a blank line.
- **Adversarial document:** d13 keeps the `flood-test-shower` and `build-over-footprint` traps.

## 6. Realism

The heritage material reads like the real thing:
- The HIS follows the NSW statement-of-heritage-impact layout: the area's significance, the house as a
  neutral building, visibility, form, materials and fabric, then recommendations.
- The heritage fee proposal and the planner's fee proposal are ordinary small-practice proposals.
- The SEE compliance table adds the HCA rows naturally.

Finding 2 is the only exception.

## Findings

1. **`Heritage consultant fee proposal - Morrow Heritage.md`, lines 58–59, and
   `Town planning fee proposal - Carrick Planning.md`, lines 58–59.**
   - d20 says "charged at $185 per hour excluding GST" and d21 says "charged at $195 per hour excluding
     GST".
   - The manifest content did not plan either rate, and `fees.json` has no row for them. The fee reader
     will find a priced rate in each proposal that the key counts as extra. Every other proposal with
     an hourly or per-visit rate has a `rate` row in the key, for example d05's $210.
   - Hourly rates are normal in these proposals, so keep the corpus as it is and change the key.
   - **Change** `expected/fees.json` d20 `prices`: add
     `{"label": "Additional work: hourly rate", "kind": "rate", "state": "priced", "amount_cents": 18500}`.
   - **Change** `expected/fees.json` d21 `prices`: add
     `{"label": "Additional work: hourly rate", "kind": "rate", "state": "priced", "amount_cents": 19500}`.
   - Add "further work at $185 per hour" to the d20 manifest `content`, and "additional work at $195 per
     hour" to the d21 manifest `content`.

2. **`Proposed ground floor plan A-201 Rev B.md`, line 41.**
   - The note reads: "Amended for DA lodgement following council's pre-DA heritage advice: laundry
     moved; terrace steps relocated".
   - Both changes are at the rear or inside the house, out of sight of the street. A PM would ask what
     heritage advice moved a laundry.
   - **Change** it to: "Amended for DA lodgement following council's pre-DA advice (planning and
     heritage): laundry moved; terrace steps relocated". Update the d15 `content` in `manifest.json` to
     match. The note still cites the heritage advice that `changed_from_base` asks for.

**Observations (no change needed):**
- **Order of documents:** d13 (20 May) and the HIS d22 (25 May) both describe the design as following
  the pre-DA advice. The plan revised "for DA lodgement" is Rev B, dated 27 May. That order is normal:
  the HIS assesses the near-final DA set, and the lodgement set is issued after it.
- **Long lines:** d02 line 57, d13 lines 32 and 50, and d16 lines 12, 21, 68 and 99 are long, unwrapped
  lines where the evidence sentences were inserted. They are invisible once rendered.
- **Housing Code wording:** the d02 item 4 wording is plausible certificate language. The underlying
  exclusion in the Housing Code for land in an HCA stays `verified: false`, as `shape.yaml` records.

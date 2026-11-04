# Render review: ST01-cdc-pathway

Reviewer: independent (brief 04r), 2026-10-30. Corpus: 16 documents in `corpus/` (d14 SEE removed).

## Verdict: **FIX**

Four findings, all small. Two matter. Finding 1 is a DA idea (clause 4.6) left in the design statement,
which is a source for `flag.cl46`. Finding 2 is the phrase "development application" in the architect's
fee proposal, which a keyword reader can take as a DA. The other two are wording that reads as written
for a test. The variant changes nothing outside `changed_from_base`, and every amount is unchanged.

## 1. Lint

`python testing/synthetic/tools/lint_corpus.py testing/synthetic/scenarios/ST01-cdc-pathway` → `ok`, exit 0.
Run from the repo root as the brief says. If you run it with `.` from inside the scenario folder, it
cannot resolve `base`, so it reports false ABN and lot errors.

## 2. What a PM concludes from the documents alone

- **Job:** a new, detached, two-storey, four-bedroom house with a double garage, for Dan and Mai Nguyen
  at 27 Corella Rise, Denistone NSW 2114 (Lot 14 DP 1302865). It is in the City of Ryde, zone R2. The
  lot is vacant and grassed, 650.2 m2, with a gentle 0.9 m fall. GFA is 298 m2 against a 300 m2 target.
  There are 3 bathrooms plus a powder room. The building is Class 1a with a Class 10a garage. Quality
  is high and the owners are the only decision makers. The site is single-stage and unrestricted.
- **Pathway:** a CDC under the Housing Code from a registered certifier (d01, d04, d05, d11). The s10.7
  certificate confirms that Housing Code complying development may be carried out on the land. There is
  no DA, no SEE, no construction certificate, no planning panel and no exhibition. The architect lodges
  on the Portal (d05). Contributions are still payable (d05 exclusions).
- **Site:** not heritage, bushfire, flood, coastal, rail, contaminated or acid sulfate land. There is
  no OSD, no stormwater quality treatment, no NorBE and no public road works. Gravity drains to the
  kerb. The sewer and water mains are in the street and there are no Sydney Water assets in the lot.
  The site is Class M with no groundwater and no basement. The 300 mm topsoil strip is not a basement.
  The blue gum is retained and the neighbour's jacaranda is far away, so there is no tree impact.
  Climate zone 5 and BASIX apply. The house is all-electric with no gas.
- **Consultants:** architect, structural, civil, geotechnical, BASIX/NatHERS, surveyor (already
  engaged) and certifier. No town planner.
- **Budget and contract:** $1,450,000 ex GST for construction. HIA NSW Lump Sum contract. Traditional
  tender. Target start 1 Feb 2027.
- **Fees:**
  - Architect: $80,000 ex. Contract administration is a $15,000 option. The rate is $220/h.
  - Structural: $8,000 ex. Inspections are $350 per visit.
  - Civil: $9,900 inc GST, which is $9,000 ex.
  - Geotechnical: $3,800 ex. Inspections are $440.
  - BASIX: $2,000 ex.
- **Invoices:**
  - RG-0418 is $3,800 + $380. Its stated total of $4,280 is wrong: the correct sum is $4,180.
  - INV-2307 is $18,000 + $1,800 = $19,800.
  - 5521 is $1,500 + $150 = $1,650.

## 3. Comparison with `expected/`

`profile.json`, `fees.json` and `invoices.json` agree with every conclusion above:
- `hdr.cond.planning = cdc`.
- The RG-0418 arithmetic flag.
- The d07 GST-inclusive figure normalised to ex GST.
- The fee labels carry "CDC" and "certifier".

None of the differences comes from a wrong value. Findings 1 and 2 are wording risks around sources
the key relies on.

**Diff against ST01-base.** Each changed line is in a document listed in `changed_from_base`:
- d01, d03, d04, d05, d06, d07, d08, d09, d11, d13 and d17 change as the manifest says.
- d14 is removed.
- d02, d10, d12, d15 and d16 are byte-identical to the base.

A search for `DA`, `development application`, `construction certificate`, `CC`, `SEE`, `council`,
`merit` and `DCP` (outside the s10.7 list of instruments) finds no stray DA or CC except findings 1 and 2.

## 4. Stray triggers and keyword traps

The search covered the `shape.yaml` trigger evidence and DOMAIN §10. Every hit is planned and is the
same as in the base:
- "No demolition is required" (d11).
- "sewer main in the street" (d03).
- "excavate to a depth of 300 mm" and "H2 treated pine" (d10).
- "exempt from on-site detention" (d07).
- "not a heritage item" and "not subject to flood related development controls" (d02).
- "lodge/lodgement" (d04, d05, d13).
- "garden beds" (d11).

The s10.7 "Complying development under the Housing Code may be carried out" trap now agrees with the
pathway, as the notes say. No unplanned trigger evidence was found.

## 5. Title blocks

- **Clean documents:** d01, d05, d06, d08, d09, d12, d13, d15 and d16 have label lines before the first
  `##` and one `#` heading equal to `Title:`. d04 has the planned email header. Number, revision and
  date match `document_register.json`. d13 Rev B supersedes d12.
- **Messy documents:** d02, d07 and d17 use letterhead, table and preformatted layouts as planned.
- **Adversarial documents:** d03, d10 and d11 carry the planned negations and traps.

## 6. Realism

All documents read as real ones. The CDC rewording is natural in most of them: in the certifier timing
email (d04), in "Design certificate for the certifier" (d06) and in "Certifier and council inspection
fees" (d07). Findings 3 and 4 are the exceptions.

## Findings

1. **`Architectural design statement - 27 Corella Rise.md`, line 60.** The text reads: "The design
   complies with the height and floor space ratio standards, so no clause 4.6 variation is sought."
   Clause 4.6 is a DA mechanism, so it cannot apply to complying development. The LEP height and FSR
   standards (9.5 m, 0.5:1) are also not the CDC tests. To a PM the sentence implies a DA, and it is
   the `flag.cl46` source.
   **Change** lines 60–62 to: "As complying development, no clause 4.6 variation is sought or possible.
   The house is within the 8.5 m Housing Code height limit and the gross floor area is well below the
   Housing Code maximum for a lot of this size." Update d11's `flag.cl46` `says` in `manifest.json` to
   match.

2. **`Fee proposal - Architectural services - 27 Corella Rise.md`, lines 24–25.** The text reads:
   "lodgement of the complying development application on the NSW Planning Portal". Once the line wrap
   is joined, "lodgement … development application" matches the DA lodgement patterns in
   `knowledge/profile/project_details.yaml` (lines 77–78). It also reads as "DA" to any keyword reader.
   This document is a `hdr.cond.planning` source.
   **Change** it to: "lodgement of the application for a complying development certificate with the
   certifier on the NSW Planning Portal". `flag.architect_lodges` still fires on "lodgement".

3. **`Owner brief - 27 Corella Rise.md`, lines 61–63.** The text reads: "…approved by a registered
   certifier:\nComplying development certificate under the Housing Code." The colon followed by a
   capitalised fragment reads as an inserted evidence phrase.
   **Change** it to: "…so we would like it approved by a registered certifier." followed by a new line
   with "Approval: Complying development certificate under the Housing Code." This keeps the exact
   evidence string.

4. **`BASIX and NatHERS fee proposal - Thermacheck.md`, line 52.** The text reads: "We issue the CDC
   certificates within five working days". The assessor does not issue CDCs.
   **Change** it to: "We issue the NatHERS and BASIX certificates for the CDC within five working days
   of receiving complete information."

Observation, no change needed: d11 line 68 says "the certifier gives the neighbour notice before work".
For a Housing Code new dwelling, neighbours are also notified before the CDC is determined. Adding that
risks a "notification" program row that the shape removes, so leave it as written.

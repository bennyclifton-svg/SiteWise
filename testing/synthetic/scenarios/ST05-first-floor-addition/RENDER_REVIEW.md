# Render review: ST05-first-floor-addition

Reviewer: independent (brief 04r), 2026-11-04. Corpus: 19 documents in `corpus/`, the same ids as ST05-base.

## Verdict: **FIX**

Three findings. One matters. Finding 1 is a pair of roof areas in the civil fee proposal (d07, a read
document): "The rear addition adds about 30 m² of new roof". A reader looking for the added area can take
30 m² as a competing `hdr.scale.added_gfa_sqm` candidate against the key's 55. Finding 2 is a word in the
owner's brief that only makes sense against the base scenario. Finding 3 is a figure in the SEE that the
variant left behind. There is no leftover single-storey or ground-floor-only wording anywhere, and every
amount is unchanged.

## 1. Lint

`python testing/synthetic/tools/lint_corpus.py testing/synthetic/scenarios/ST05-first-floor-addition` → `ok`,
exit 0 (run from the repo root).

## 2. What a PM concludes from the documents alone

- **Job:** alterations and additions to a 1958 brick house for Tomas and Leah Ferreira at 41 Tallowwood
  Avenue, Concord NSW 2137 (Lot 23 DP 30418). It is in the City of Canada Bay, zone R2. The work is a rear
  kitchen, living and dining addition on the skillion footprint, a first-floor main bedroom and ensuite
  over the rear rooms with a new stair, the bathroom and laundry redone, and two loadbearing walls removed
  on steel beams. The rear roof comes off, the builder provides a temporary roof, and the walls carrying
  the first floor are underpinned. The house becomes two storeys. It is Class 1a. GFA is 140 m2 existing,
  55 m2 added and 195 m2 after the works (d13, d16; d14/d15 split it 30 m2 ground and 25 m2 first floor).
  The site is 557.4 m2 with a 12.2 m frontage, a 1.0 m side path and no rear lane. The owners move out,
  are the sole decision makers and want mid-range finishes.
- **Pathway:** a DA to the City of Canada Bay (d01, d05, d16), chosen over a CDC although the s10.7 says
  Housing Code complying development may be carried out. The architect lodges on the Portal and writes the
  SEE. The application is decided under delegation, with no cl 4.6, panel or exhibition. Height is 7.6 m
  against 8.5 m. FSR is 0.35:1, which agrees with 195/557.4.
- **Site:** not heritage, bushfire, flood, coastal, rail, contaminated or acid sulfate land. There is no
  OSD (existing or required), no stormwater quality treatment, no NorBE and no road works. The sewer and
  water mains are in the street. The site is Class M with no groundwater and no basement. Existing footings
  are brick strips at 0.5 m, and underpinning takes 150 kPa. There is no tree impact and no
  transfer structure. No asbestos was found. The house keeps natural gas. Climate zone 5 and BASIX apply.
- **Consultants:** architect, structural, civil, geotechnical, hazmat, BASIX and the surveyor (already
  engaged), with a certifier for the CC. No town planner.
- **Budget and contract:** $640,000 ex GST for construction. HIA NSW Lump Sum contract. Traditional
  tender to three or four builders. Target start 1 March 2027 (in the email only).
- **Fees:** architect $44,000 ex, with contract administration a $11,000 option and $210/h. Structural
  $9,600 ex, with inspections $330 per visit. Civil $7,260 inc GST, which is $6,600 ex. Geotechnical
  $3,900 ex, with inspections $420. Hazmat $2,300 ex, with a $650 clearance option. BASIX $1,200 ex.
- **Invoices:** KHA-INV-0412 is $9,500 + $950 = $10,450. CS-2291 is $2,300 + $345 = $2,645; the GST is
  15%, not 10%. LCS 1088 is $2,200 + $220 = $2,420.

## 3. Comparison with `expected/`

`profile.json`, `fees.json` and `invoices.json` agree with every conclusion above: `hdr.scale.storeys = 2`
from d01, d13 and d16; existing 140, added 55 and total 195; `hdr.cond.planning = da`;
`flag.transfer_structures = no` from d06; the d07 GST-inclusive total normalised to $6,600; and the CS-2291
`gst_rate` flag. No value is wrong. Finding 1 is a wording risk next to the added-area key.

**Diff against ST05-base.** Exactly the ten documents in `changed_from_base` differ (d01, d05, d06, d07,
d08, d12, d13, d14, d15, d16). The other nine are byte-identical to the base. Each change does what the
manifest says. Title-block lines are unchanged apart from the `Project:` line in d08 and d12 ("Rear and
first-floor addition"), which is correct.

**Leftover storey wording.** A search for `single-storey`, `single storey`, `single level`, `storey`,
`upper`, `ground floor` and `m2`/`m²` finds no single-storey or ground-floor-only wording. Every storey
mention says two storeys or a first floor. The hazmat report (d11) still lists "Roof space above the rear
rooms", which suits the roof removal. The only area figures that compete with 55 m2 are the d07 roof areas
(finding 1). The 30/25 split on d14/d15 sits inside "Addition 55 m2 (…)", so it does not compete.

## 4. Stray triggers and keyword traps

The search covered the `shape.yaml` trigger evidence and DOMAIN §10. Every hit is planned:
- The `first-floor-addition` evidence appears verbatim in d01 and d13. The base negation "Single-storey
  addition only" is gone.
- "No transfer structures are required; the first floor is carried on new steel beams" (d06) is the
  planned near-miss in notes.md.
- "underpin*" appears in d06, d08, d12 and d13. The variant plans it (`substructure.underpinning` new).
- The base traps are unchanged:
  - "sewer main in the street" (d03).
  - "OSD is not required" (d07).
  - "fibre-cement" and "loose-fill … no listing" (d11).
  - "excavate to a depth of 300 mm" (d12).
  - "Dulux Heritage", "Flood test the shower" and "build over the footprint" (d13).
  - "complying development may be carried out" (d02).
- "temporary roof" (d06 exclusions, d13) is planned as `structure.temporary-works`.

## 5. Title blocks

The clean documents have `Label: value` lines before the first `##` and one `#` heading equal to
`Title:`. d02 and d07 keep their messy Markdown-table blocks. The d03, d11 and d12 adversarial documents
carry their traps. Nothing changed from the base register.

## 6. Realism

The changed documents read like the real thing:
- The structural scope (capacity check, underpinning, a temporary roof by the builder) is right.
- The geotechnical test pits at three footings are right.
- The SEE height and privacy sections are right.
- The Rev B "stair turned at a half landing" is right.

The trigger sentence "New first-floor addition over the existing house containing…" is a verbless
fragment in prose (d01 line 17, d13 line 20). It is fixed by `shape.yaml` evidence, so it is not a
finding. Findings 1 and 2 cover what reads as written for a test.

## Findings

1. **d07 `Lindqvist Civil - stormwater fee - Tallowwood Ave.md`, line 22.** "The rear addition adds about
   30 m² of new roof and the first floor replaces about 35 m² of the existing rear roof; all roof water is
   drained to the existing system, plus the new paved terrace at the back of the house."
   - **Problem:** d07 is a read document. "addition adds about 30 m²" is a competing added-area candidate
     against 55 m2. A third area (35 m²) also sits next to it. The splice also leaves "plus the new paved
     terrace" hanging off the wrong clause.
   - **Change:** replace the sentence with "The rear addition has a new low-pitched roof, and the first
     floor replaces part of the existing rear roof. Roof water from both, and from the new paved terrace
     at the back of the house, is drained to the existing system."
   - **Manifest:** in `manifest.json` d07 `content`, replace the quoted roof-area sentence with the same
     text.
2. **d01 `Owner brief - 41 Tallowwood Avenue.md`, line 19.** "What we have in mind is a smaller rear
   addition".
   - **Problem:** "smaller" compares with nothing the owners have said. It only makes sense against the
     base scenario's larger addition, so the sentence reads as written for a test.
   - **Change:** replace "a smaller rear addition" with "a modest rear addition".
   - **Manifest:** make the same change in the d01 `hdr.work_type` `says`.
3. **d16 `Statement of environmental effects - 41 Tallowwood Avenue.md`, line 47.** "Landscaped area
   (DCP) | 40% | 46% | Yes".
   - **Problem:** the figure is unchanged from the base. The ground-floor addition is now about 25 m2
     smaller, and the owners' stated reason is to keep the backyard, so the landscaped area should rise
     by about 4.5% of 557.4 m2. A planner would notice. The figure is not scored.
   - **Change:** replace `46%` with `50%`.

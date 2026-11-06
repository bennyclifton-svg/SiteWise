# ST11-cdc-low-rise-code render review (independent read)

Reviewer: senior PM read of `corpus/` against `manifest.json`, `expected/`, the T11 shape and DOMAIN.md §10.
Focus: the seven rewritten documents (d01, d04, d05, d06, d07, d13, d15), the removal of d10 and d16, and copied
base wording that still implies a DA.

## Verdict: FIX

Four findings. One is serious: d13 says the proposed lots meet the code's lot size and frontage standards, but the
proposed lots are 350.2 m2 with about 8.5 m frontage each. The same sentence frames cl 4.6 as "not needed", but a
CDC cannot vary a standard at all (finding 1). No document still says DA, development application, council's
determination, construction certificate, SEE or town planner. Every fee, invoice and profile value matches the key.

## Step 1: lint

`python testing/synthetic/tools/lint_corpus.py testing/synthetic/scenarios/ST11-cdc-low-rise-code` gives
`ok  ST11-cdc-low-rise-code`.

## Step 2: what a PM concludes before opening the key

- Project: as the base. Demolish a 1958 house and garage. Build an attached side-by-side pair of two-storey,
  four-bedroom townhouses: two Class 1a dwellings, each with a Class 10a garage, 460 m2 GFA. Torrens subdivision
  into proposed Lots 1 and 2. Lot 9 DP 236154, 700.4 m2, R2, City of Parramatta, climate zone 5. Work type: new.
- Pathway: a CDC under the Low Rise Housing Diversity Code, determined by a registered certifier and lodged by the
  architect. The subdivision gets its own CDC. There is no panel, no cl 4.6 and no exhibition, and neighbours get
  notice. d02 confirms that the LRHDC applies to the land. d02's "Permitted with consent" is standard s10.7
  land-use-table wording and does not imply a DA.
- Site, hazmat, owner and contract: unchanged from the base (vacant site, single owner, mid quality, traditional,
  HIA, $1,650,000 ex GST, target start 1 February 2027 from the email only). OSD is required and none exists.
- Consultants: architect, structural, civil/stormwater, geotechnical, hazmat and surveyor. No town planner; d05's
  exclusions no longer list one. The certifier is named in d01, d05, d06, d07 and d13, but no certifier fee is in
  the corpus.
- Fees and invoices: identical to the base except the planner, who is gone. Architect $85,000, structural $9,800,
  civil $14,300 inc GST, geotech $4,100, hazmat $2,600. The invoices are unchanged, and the d18 arithmetic error
  is still there.

## Step 3: comparison with the key

The conclusions match `profile.json` (only `hdr.cond.planning` is changed, from da to cdc, and the d16 sources are
removed), `fees.json` (labels renamed to CDC, amounts unchanged, no planner) and `invoices.json` (byte-identical
to the base). `procurement.json` drops the planner. `document_register.json` drops d10 and d16 and gives d13 and
d15 the status "for CDC". Finding 1 puts `flag.cl46` on a shaky sentence. Finding 4 is a defect in the key's notes.

## Findings

### 1. d13: the proposed lots "meet the code's minimum lot size and frontage standards" (serious, domain)

- File: `Architectural design statement - 18 Wattlebird Street.md` line 73: "Both proposed lots meet the code's
  minimum lot size and frontage standards, so no clause 4.6 variation is needed."
- What is wrong:
  - The dual-occupancy lot standards in the code apply to the parent lot, as I understand them (about
    400 m2 and 12 m frontage, verified: false). d14 and d15 give each proposed lot 350.2 m2, and half of the 17 m
    frontage is about 8.5 m. A PM reading line 73 against the site plan would conclude that the lots fail the
    standard.
  - "No clause 4.6 variation is needed" implies that a variation was available. Under a CDC it is not: a
    certifier cannot approve any departure from a standard.
  - The sentence is copied base wording ("exceed the minimum lot size ... so no clause 4.6 variation is sought")
    with "the code's" put in.
- Exact change: in d13 line 73 and in the manifest d13 `carries` `flag.cl46` `says`, replace the sentence with
  "The parent lot meets the code's minimum lot size and frontage for a dual occupancy. A complying development
  certificate cannot vary a development standard, so no clause 4.6 variation is involved." `flag.cl46` stays no.

### 2. d05: "shadow diagrams" in Stage 2 is DA wording carried over (minor, copied base wording)

- File: `Fee proposal - Architectural services - 18 Wattlebird Street.md` line 31.
- What is wrong: the manifest d05 content for Stage 2 lists construction-level drawings, demolition and waste plans
  and the subdivision plan, with no shadow diagrams. Shadow diagrams were a DA deliverable in the base, assessed
  against the DCP's solar access controls. A dual-occupancy CDC is assessed against the code's numeric standards
  (verified: false). It is harmless to scoring but reads as a DA leftover.
- Exact change: in line 31, replace "Construction-level drawings for the certifier, shadow diagrams, demolition and
  waste management plans" with "Construction-level drawings for the certifier, BASIX coordination, demolition and
  waste management plans".

### 3. d05 Stage 3 overlaps Stage 2 once the CDC needs construction-level drawings (minor, realism)

- File: same file, line 32 ("Construction drawings, specification and finishes schedules for the builder and
  tender"), read against line 31 and d04 lines 9 to 11 ("most of the documentation has to be done before we
  lodge").
- What is wrong: Stage 2 already pays for construction-level drawings, and Stage 3 pays $38,000 for "construction
  drawings" again. A client would ask why the drawings are paid for twice. The base split (DA drawings, then CC and
  tender drawings) made sense. The CDC split needs Stage 3 to name what is left.
- Exact change: in line 32, change the services cell to "Detailed construction drawings and details, specification
  and finishes schedules for the builder and tender, coordinated with the engineers". Amounts unchanged.

### 4. `expected/notes.md`: a stale paragraph says dwellings 2 -> 4 (moderate, key notes)

- File: `expected/notes.md`, the paragraph "`hdr.scale.dwellings` (2 -> 4) and `hdr.scale.avg_dwelling_size_sqm`
  (230 -> 115) differ from the base ...".
- What is wrong: it was copied from the three-plus-townhouses variant. It contradicts the line above it ("Only
  `hdr.cond.planning` differs from the base") and `profile.json`, which keeps 2 and 230.
- Exact change: delete that paragraph.

## Step 4: stray triggers

- `cdc-low-rise-code` evidence: both phrases are in d13 line 70, and the d01 line 40 sentence is close to the
  first. The trigger's negations ("A development application will be lodged", "does not meet the code's landscaped
  area standard") are not in the corpus.
- I searched the corpus for every other trigger's evidence (strata, right of carriageway, battle-axe,
  "4 two-storey", two buildings, vacant lot, no subdivision, interallotment easement, a sewer main in the lot, TPZ,
  BAL, heritage conservation area, ANEF, rail noise). There are no hits except d02's negative heritage line and the
  d12 "landscaped areas" grading advice. Neither is a trigger.
- DA leftovers: I searched for DA, development application, council's determination or approval, construction
  certificate, CC, SEE, environmental effects, town planner, Halden, Section 4.55 and notification. None of these
  appears in any document, the unchanged d03, d08, d09, d11, d12, d14 and d17 to d19 included. d02's "Duty
  planner" is council staff. The remaining "council" mentions are correct under a CDC: d01 and d05 council fees and
  contributions, d07 council's OSD policy, crossing standard drawings and inspection fees.
- DOMAIN.md §10: declared hits are as in the base (d01 and d13 rainwater tank, d06, d12 and d13 separating wall,
  d05 Stage 1 to 4, d13 "building a dual occupancy" and fire-rated wall). "Lodge" in d01, d04 and d05 supports
  `flag.architect_lodges = yes`. "Public exhibition" and "local planning panel" in d13 line 73 are negated and
  carry the expected no answers.

## Step 5: title blocks

d01, d05, d06, d13 and d15 have `Label: value` lines before the first `##` and one `#` heading equal to `Title:`.
Number, revision, date and issue status ("For CDC" in d13 and d15) match `document_register.json`. d04 is an
email thread as in the base. d07 remains messy: a table block, GST-inclusive figures and unheaded sections.
d13 remains adversarial: a clean block with its traps in the body.

## Step 6: realism

The rewrites read like real CDC-path documents. The d04 reply's point that a CDC front-loads the documentation is
what an architect would say. d06 "design certificates for the certifier" and d07 "for the CDC application" are
natural.

d13 `## Planning` line 73 still reads as written for the test. "Not by council or the local planning panel" and
"No public exhibition or community consultation applies" are things an architect would not normally write in a
design statement for a certifier. I accept them for an adversarial document because they carry the `flag.panel`
and `flag.community` keys. Finding 1 fixes the one sentence there that is also wrong.

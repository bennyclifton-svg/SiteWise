# Render review: ST03-base

Reviewer: independent (brief 04r), 2026-11-07. Corpus: 18 documents in `corpus/`. Type: T03-office-fitout.

## Verdict: **FIX**

Five findings. None of them changes a profile, register, invoice or stated-total key, and the
harness's fee score (stated total and firm name) is unaffected. Findings 1 to 3 are priced rows the
manifest does not plan and `fees.json` does not key. Findings 4 and 5 are dating and sequence slips
that a PM would notice. Every planned trap appears where the manifest puts it. No implied-scope
pattern (wet room, kitchen, robe, bedroom, stair, party wall, strip-out, masonry, energy rating)
fires anywhere except the planned traps in d11 and d13.

The writer's declared departure, the d09 CDC row wording, is judged under "Fee key check" below. The
key still matches, and no change is needed.

## 1. Lint

`python testing/synthetic/tools/lint_corpus.py testing/synthetic/scenarios/ST03-base` gives `ok`,
exit 0.

## 2. What a PM concludes from the documents alone (before opening `expected/`)

- **Job:** Avenor Capital Partners Pty Ltd fits out the whole of Level 14 (1,200 m2 NLA, single
  tenancy) of Wenlock Place, 211 Wenlock Terrace, North Sydney NSW 2060 (Lot 12 DP 1184520, North
  Sydney LGA, zone E2 Commercial Centre). It is a commercial office refurbishment, a tenant fit-out
  of an existing floor handed over in base-building condition. There is no strip-out, demolition,
  structure, stair or new amenities. The kitchenette is not a kitchen. Make-good is a future lease
  obligation.
- **Building:** a 24-storey A-grade tower built in 2008, Class 5, rise in storeys 24, Type A,
  sprinklered, climate zone 5, no gas to the office floors, and no Sydney Water assets in the
  tenancy. The other floors stay occupied, and noisy works and deliveries are after hours only. The
  base building's DA 2004/0617 and its NABERS rating are the landlord's, not this job's.
- **Pathway:** a CDC from a private certifier. The s10.7 certificate's "may be carried out" is
  eligibility only. There is no clause 4.6 and no panel, and the DBP Act does not apply.
  Section J is DTS and BASIX does not apply.
- **Site conditions:** not heritage (the heritage item is next door at 205), not bush fire prone,
  no flood controls, not acid sulfate, no contamination record, no rail corridor, no road widening.
  There are no basement, excavation, transfer structures, trees, road reserve works or OSD.
  Environmental sensitivity is standard.
- **Procurement:** traditional lump sum on full documentation under AS 4000-1997, three to five
  tenderers from the landlord's list, not staged. High quality. The services engineers carry their
  scope through construction.
- **Budget:** $2,400,000 ex GST for the fit-out contract. Tenant-supplied items are loose
  furniture and workstations, AV equipment and IT active equipment.
- **Consultants:** interior designer (lead), multi-services engineer (MEHF), BCA, access, private
  certifier and QS. No architect, structural, civil or geotechnical engineer.
- **Fees (ex GST):**
  - Corrigan Hale $130,000 in six named phases, plus a $12,000 FF&E option and $210/h.
  - Meridale $68,200 inc GST = $62,000 ex GST.
  - Tallowood $10,000, plus a $2,400 option.
  - Clearpath $5,500, plus $180/h.
  - Harbourgate $17,000, plus $350 per extra inspection.
  - Baxter Lane $16,500, plus $190/h.
- **Invoices:**
  - CHI-INV-0418 (30 April 2026): $14,000 + $1,400 = $15,400.
  - TBR-INV-1187 (15 May 2026): $3,500 + $350 = $3,850.
  - Meridale 2291 (29 May 2026): $6,000 + $600, printed as $6,900. It is misadded by $300 and
    should be $6,600.
- **Program:** access and contractor start on 7 September 2026 (email only).

## 3. Against `expected/`

**Profile.** Every keyed value matches my reading, including the five `not_read` keys, which the
documents do state (floor plate 1,200, grade A, one tenancy, Type A, DBP not applicable). Each
negated flag has its carried sentence. `flag.fire_specialist_systems: no` rests on the d11 facade,
cladding, timber or structural sentence, which matches that flag's trigger. Profile keys: no
difference.

**Invoices.** All three match on number, date, ABN, ex GST, GST and total. The d17 `arithmetic`
flag is the planned misadded total. Invoice keys: no difference.

**Fees.** Firms, ABNs, contacts, emails, phones, addresses, stated totals and GST bases all match.
d06's inc-GST stages ÷ 1.1 give 6,000 / 18,000 / 24,000 / 3,000 / 11,000 = $62,000, as keyed.
Every table sums to its stated total. Findings 1 to 3 are the only differences.

### Fee key check: d09 CDC row (the writer's declared departure)

- **What differs:** the document's row 1 reads "Assessment and determination of the complying
  development certificate, including the fire safety schedule". `fees.json` labels it "CDC
  assessment and determination with the fire safety schedule".
- **Why the document is right:** the manifest's d09 `carries` requires "Assessment and
  determination of the complying development certificate" word for word. The writer had to use it.
  The key label is the manifest `content` paraphrase.
- **Why the key still matches:**
  - The row is the only $9,800 cell. It is priced, and it is a stage. Its meaning (CDC assessment
    and determination, including the fire safety schedule) is identical.
  - `harness/score.py` `score_fees` scores only `stated_total_cents` and `firm_name`. Price labels
    are not machine-matched.
  - The same paraphrasing already exists elsewhere in the keys: d09 row 2 drops "(commencement and
    final)", d10 row 1 drops the bracketed list, and d07's option drops ", including responses ...".
- **Change:** none required. If the owner wants key labels to be verbatim, change the d09 row 1
  label to "Assessment and determination of the complying development certificate, including the
  fire safety schedule". This is optional and has no scoring effect.

## 4. Stray triggers (shape `triggers[].evidence`, DOMAIN.md §10)

All the hits below are planned:

| Hit | Document | Planned by |
|---|---|---|
| DA history and OC 2008 | d03 | trap `base-building-da` |
| NABERS base-building rating | d03 | trap `base-building-nabers` |
| 24-storey | d03 | trap `storeys-of-building` |
| Sprinklered throughout | d03, d11 | trap `sprinklered-not-works` |
| Fire stairs | d11 | trap `fire-stairs` |
| BASIX not applicable | d11 | trap `basix-not-applicable` |
| No strip-out or demolition | d13 | trap `no-strip-out` |
| Core bathrooms | d13 | trap `core-bathrooms` |
| Make good at expiry | d01 | trap `make-good-at-expiry` |
| Kitchenette | d01 | trap `kitchenette-not-kitchen` |
| Heritage item next door | d02 | trap `heritage-next-door` |
| Complying development may be carried out | d02 | trap `cdc-land-eligible` |
| Loose-fill asbestos (not listed) | d02 | content |
| "No performance solution is proposed" | d11 | content |
| Accessible sanitary facilities are base building | d12 | content |
| Not targeting Green Star or NABERS | d01 | carried |
| No staged construction | d01 | carried |

None of the following appears:

- a comms room with cooling, racks or a UPS (the comms room is plain);
- slab coring or floor boxes (power poles are used);
- heavy storage loads or compactus;
- D&C wording;
- exempt-development wording outside the SEPP's name;
- "common wall", or "kitchen" alone;
- asbestos in the tenancy;
- decant or swing space.

No fee table uses "Stage N". Stray triggers: no finding.

## 5. Title blocks

- **Clean documents** (d04, d05, d07, d08, d09, d10, d12, d14, d15, d16, d18): the label lines come
  before the first `##`, and the one `#` heading equals `Title:`. Numbers, revisions, titles and
  dates match `document_register.json`. The labels used (`Ref`, `Proposal no`, `Quote no`,
  `Report no`, `Drawing number`, `Revision`, `Date`) are all on DISCOVERY §4.3's read list.
- **d04** is an email: `From`, `To`, `Subject` and `Date` come first, the quoted block repeats them,
  and the latest date is 24 February 2026.
- **Messy documents:**
  - d02's identity is in tables, and its one heading is "Planning Certificate".
  - d06's identity is in a table above the heading, with "Dear Daniel" after it.
  - d17's identity is in a table.
  - Each is what its tier says.
- **Adversarial documents** (d01, d03, d11, d13): clean label lines that agree with the register,
  and the traps are written as `traps[].what` describes.

## 6. Realism

These read as real documents: a PM's brief, a landlord's fit-out guide, a council s10.7 certificate
in council form (Codes SEPP cl 1.17A/1.19, $133 fee, LFAI item), consultant proposals with
assumptions and validity periods, NCC-literate BCA and access reports, drawing stand-ins with
revision tables, and plain tax invoices. "Test fit" is the ordinary fit-out term and is fine. The
one passage that reads as written for a test is the d01 "For the avoidance of doubt" list of nine
negations ("not multiple buildings", "no Connecting with Country", "no existing OSD"). Every line of
it is carried, so it stays; a real brief would carry two or three of these.

## Findings

1. **d08 `Access consultancy fee proposal - Clearpath - Wenlock Place.md`, L44-45: unplanned
   hourly rate.**
   - **Wrong:** "Additional site visits are charged at $180 per hour ex GST." The manifest's d08
     content plans no rate, and `fees.json` keys none. The proposal reader may store it as a
     `rate` cell that the key does not have. d05's rate is planned and keyed, so the keys are now
     inconsistent.
   - **Change:** delete the sentence "Additional site visits are charged at $180 per hour ex GST."
     The paragraph ends "payable within 14 days."
   - **Alternative:** keep it and add `{"label": "Additional site visits, hourly rate", "kind":
     "rate", "state": "priced", "amount_cents": 18000}` to d08's prices.

2. **d09 `Certifier fee proposal HGC-26-0219.md`, L44: unplanned per-inspection rate.**
   - **Wrong:** "additional inspections caused by incomplete work, at $350 each ex GST" is a priced
     rate in the Exclusions list. It is neither planned nor keyed.
   - **Change:** shorten the bullet to "- additional inspections caused by incomplete work".
   - **Alternative:** add `{"label": "Additional inspections, per inspection", "kind": "rate",
     "state": "priced", "amount_cents": 35000}` to d09's prices.

3. **d10 `Quantity surveying fee proposal - Baxter Lane.md`, L44-45: unplanned hourly rate.**
   - **Wrong:** "Additional estimates are charged at $190 per hour ex GST." It is neither planned
     nor keyed.
   - **Change:** delete the sentence. The paragraph ends "payable within 14 days."
   - **Alternative:** add `{"label": "Additional estimates, hourly rate", "kind": "rate", "state":
     "priced", "amount_cents": 19000}` to d10's prices.

4. **d11 L14 and d12 L13: the BCA and access reports review a "design development layout" that is
   not yet drawn.**
   - **Wrong:** the reports are dated 22 and 24 April 2026. The concept report is dated 15 April,
     and ID-101 Rev A ("For landlord concept approval") is dated 29 April. A PM would ask which
     layout was reviewed. Rev B, the CDC issue, follows on 13 May.
   - **Change:** in d11 L14 change "the design development layout of the Level 14 fit-out" to "the
     concept layout of the Level 14 fit-out". In d12 L13 change "the design development layout of
     the Level 14 tenancy" to "the concept layout of the Level 14 tenancy".
   - This does not affect scoring.

5. **d13 `Concept design report - Avenor Level 14 fit-out.md`, L28: comms room location
   contradicts ID-101 Rev A.**
   - **Wrong:** the concept report (15 April) puts "a comms room beside the riser". Rev A (29 April)
     shows it "on the south side of the floor", and Rev B moves it "next to the north riser after
     landlord comments". In that sequence the riser position is new at Rev B.
   - **Change:** at d13 L28 change "- a comms room beside the riser" to "- a comms room on the south
     side of the floor".
   - This does not affect scoring.

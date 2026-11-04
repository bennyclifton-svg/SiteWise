# Render review: ST09-base

Reviewer: independent (brief 04r), 2026-11-04. Corpus: 16 documents in `corpus/`.

## Verdict: **FIX**

Seven findings. Two of them matter for scoring. In finding 1, d13 is read for the profile and says
"a rise of more than three storeys" on a 6-storey building. That is an unplanned storeys number. In
finding 2, `fees.json` leaves out Halyard's hourly rate, but it keys the hourly rates of the other two
firms. Every planned trap appears where the manifest puts it. No unplanned implied-scope word
(kitchen, masonry, brickwork, common wall, ensuite, a second bathroom, bedroom or stairs) appears
anywhere.

## 1. Lint

`python testing/synthetic/tools/lint_corpus.py testing/synthetic/scenarios/ST09-base` → `ok`, exit 0.

## 2. What a PM concludes from the documents alone

- **Job:** remediation, not new work. The owners corporation (The Owners - Strata Plan No. 912345)
  renews the waterproofing of all 42 leaking balconies like for like. The building is Kurrajong
  Apartments, 41 Kurrajong Parade, Hurstville NSW 2220: a 6-storey residential flat building (Class 2,
  rise in storeys 6) of 42 units, built in 2008. It is out of statutory warranty and no claim is
  pursued. The frame is reinforced concrete with cantilevered balcony slabs. The car park is at grade,
  with no basement and no podium. The building stays fully occupied throughout. The client is a strata
  scheme run by a committee and a managing agent, and two general meetings stand in the path.
- **Planning:** exempt development (like for like, no change to the appearance), with no DA and no CC.
  The DBP Act applies, so regulated designs and declarations are needed. The land is Georges River,
  zone R4. It is not heritage, flood-controlled, contaminated, bush fire prone or next to a rail
  corridor. The land is CDC-eligible, but that is not the pathway for these works.
- **Site and structure:** the slabs are sound with no spalling. The balustrade posts are re-fixed with
  side brackets and the glass panels are retained. Set-down is adequate and the door frames are
  retained. There is no plant on the balconies and no asbestos. Nothing is excavated, nothing is built
  over a sewer or water main, and there are no transfer structures. No OSD exists. Delivery is one
  continuous stage.
- **Procurement:** traditional. The consultant designs the works and tenders them to five remedial
  builders on an AS 4000-1997 lump sum, with a schedule of rates for provisional quantities only.
  Budget $1,680,000 ex GST. The committee wants a start by 1 February 2027.
- **Consultants:** a remedial building consultant (lead) and a remedial structural engineer. No
  geotechnical, hydraulic, mechanical, environmental or certifier input.
- **Fees:**
  - Parapet $81,000 ex GST (24.5k, 18k, 6.5k and 32k). Opening-up disbursement estimated at $6,800.
    Meetings $650 each. Hourly rate $210.
  - Halyard $99,000 inc GST, which is $90,000 ex GST (27k, 20k, 8k and 35k). Opening-up is included.
    Hourly rate $242 inc GST, which is $220 ex GST.
  - Karrak $17,000 ex GST (6.5k, 5.8k, 0.7k and 4k). Extra inspections $480 each.
- **Invoices:**
  - PBC-INV-1187: $24,500 + $2,450 = $26,950.
  - 3307: $6,500 + $650 = $7,150. Its ABN, 77 650 369 250, differs from the quote's 77 650 369 205.
  - PBC-INV-1192: $6,800 + $680 = $7,480.

## 3. Compared with `expected/`

My conclusions match `profile.json` on every keyed value. They match `invoices.json` too, including
`abn_invalid` on d15. They match `fees.json` on totals, GST basis, stages and `gst_inc_normalised` on
d06. The one difference is Halyard's hourly rate, which the key omits (finding 2).

## 4. Stray triggers

Each planned trap appears where the manifest puts it:

| Doc | Trap | Where it appears |
|---|---|---|
| d01 | rfp-cued-as-proposal | line 66 |
| d02 | cdc-eligible-land | line 46 |
| d03 | bathroom-below | line 48, the only "bathroom" in the corpus |
| d03 | bedroom-sill | line 55, the only "bedroom" |
| d05 | fee stages | lines 32 and 38-46 |
| d07 | fire-stairs | line 30, the only "stairs" |
| d08 | flood-test | lines 20 and 45, both in the planned passage |
| d08 | remediation | line 15 |
| d09 | flood-test-hold-point | line 85 |
| d09 | strip-out | lines 30 and 34 |
| d13 | schedule-of-rates | lines 17 and 40 |
| d15 | abn-typo | line 10 |

"Basement" appears only in the four planned `flag.basement` negations (d01, d07, d08 and d10). "Flood"
appears only in d08, d09 and the d02 source. "Contaminat" appears only in d02. "Demolish" appears only
in d09. No document contains kitchen, masonry, brickwork, repointing, common wall, ensuite, cladding,
facade, scaffold, condenser or spalling, apart from the negations "no spalling".

The following hits are not declared as traps. All are negations or harmless:

- **Statutory warranty, out of warranty:** d03 line 17 and d08 lines 89-92. Planned in the content.
- **"Hazardous materials survey ... found no asbestos":** d03 line 28 and d08 line 54. This is a
  common negation listed in DOMAIN.md §10.
- **"Pitched metal roof ... outside the scope":** d08 line 36. Planned.
- **"Barrier height ... at least 1 m":** d10 line 43. Planned.
- **"No plant or services":** d03 line 32 and d08 line 56.
- **"Stage 1" in d14 line 20:** an invoice, so it is not read for the profile.
- **"Railway Parade" in d06:** see finding 6.
- **"Rise of more than three storeys" in d13:** see finding 1.

## 5. Title blocks

Clean documents:

- d10, d11 and d12 each have `Label: value` lines before the first `##` and one `#` heading equal to
  `Title:`. Their numbers, revisions and dates match `document_register.json`. RW-101 Rev B
  supersedes Rev A.
- d14 and d16 have the same structure. They are not in the register.
- d04 has the email header its manifest entry declares.

Messy documents d02, d06 and d15 use the Markdown-table blocks their manifest entries declare.

The adversarial documents (d01, d03, d05, d07, d08, d09 and d13) have readable label lines that match
the register, and each carries its planned trap.

## Findings

1. **d13 `Conditions of tender PBC-T-26014.md`, lines 52-53: "HBCF cover is not required because the
   building is a multi-storey building with a rise of more than three storeys."** d13 is read for the
   profile, and this sentence puts a second storey number ("three storeys") next to the 6 keyed for
   `hdr.scale.storeys` and `det.rise_in_storeys`. It also comes close to the `low-rise-hbcf` trigger
   wording. The manifest does not declare it as a trap.
   **Change:** use "HBCF cover is not required: the multi-storey building exemption applies (rise in
   storeys of 6)." Update the d13 `content` to match.

2. **`expected/fees.json`, d06 prices: Halyard's hourly rate is missing.** d06 line 44 states
   "Additional meetings and services at $242 per hour including GST". The key includes the d05 and d07
   rates but not this one, so a correct reading would score as an extra price.
   **Change:** add `{"label": "Additional meetings and services, hourly rate", "kind": "rate",
   "state": "priced", "amount_cents": 22000}` to d06 prices. That is $242 inc GST normalised to $220 ex
   GST, like the stages.

3. **d03 `Building information and leak history - Kurrajong Apartments.md`, line 37 against lines 44
   and 54.** Line 37 says "only the first report is listed". Yet balcony 206 is listed twice (08/08/2023
   and 16/12/2024 "Further drips"). The two 206 rows also name the upper, source balcony. Every other
   row names the balcony where the leak was reported.
   **Change:** line 44 → `| 08/08/2023 | 106 | Water running down the slab edge from balcony 206 |`;
   line 54 → `| 16/12/2024 | 306 | Water at the door sill |`. The log keeps 23 rows. In line 48, drop
   the redundant "in 2024", and keep the rest of the planned trap sentence.

4. **d01 `Consultant brief - Kurrajong Apartments balconies.md`, line 66: "by 20 February 2026".**
   All three proposals are dated after this deadline: d05 on 23 February, d06 on 25 February and d07 on
   26 February. A PM would read every proposal as late.
   **Change:** use "by 27 February 2026", and update the d01 `content` and the `rfp-cued-as-proposal`
   trap text to match.

5. **d09 `Remedial specification PBC-SPEC-26014 Rev A.md`, lines 50-51: "one new overflow per
   balcony, discharging clear of the balcony edge and connected to the existing downpipes as shown on
   RW-101".** An overflow cannot both discharge clear of the edge and connect to the downpipes. RW-101
   Rev B Detail 6 shows only "discharging clear of the slab edge".
   **Change:** use "Install one new overflow per balcony through the balustrade line, discharging clear
   of the slab edge, as shown on RW-101 Detail 6."

6. **d06 `Halyard fee submission - Kurrajong Parade.md`, line 2: "7 Railway Parade, Kogarah".** This
   is an unplanned `\brail` hit. The risk is low because d06 is dropped on a procurement row and
   `flag.rail` is keyed from d02. Still, base documents should not touch the rail flag.
   **Change:** use "Level 2, 7 Montgomery Street, Kogarah NSW 2217", and update the d06 address in
   `fees.json`. Alternatively, declare it as a trap.

7. **Realism: sentences shaped for the answer key.** Each change keeps the signal. Update each matching
   manifest `says`.
   - **d10, lines 52-55, "Other matters": "No excavation is proposed next to adjoining boundaries ...
     The works include no basement or excavation."** A balcony slab report would not say this. Change
     it to: "## Other matters / The works are confined to the balconies; no excavation, basement or
     ground works are involved, and nothing is done near the adjoining boundaries or neighbouring
     structures."
   - **d10, lines 21-22: "cantilevered reinforced concrete slabs of the existing building,
     cantilevering about 2.4 m".** Change it to: "The balconies are reinforced concrete slabs of the
     existing building, cantilevering about 2.4 m from the edge of each floor plate."
   - **d09, lines 26 and 52-53: "Authority assets" and the OSD and sewer sentences.** These say the same
     thing twice, and a balcony specification would not mention OSD. Delete line 53. Change line 52 to:
     "Outlets connect to the existing downpipes. The building stormwater system, which has no on-site
     detention, is not altered."
   - **d08, lines 26 and 30.** "The building is 6 storeys above ground." repeats line 25. "The works are
     to an existing building" reads like a form field. Delete line 26. Change line 30 to: "Existing
     building; these are remedial works."
   - **d01, line 34: "remediation, not new work".** Change it to: "to investigate and rectify the
     leaking balcony waterproofing of the existing building." The title and d08 still carry
     `hdr.work_type`.

Everything else reads like the real thing. That includes:

- the s10.7 certificate, the email thread, the leak log, the proposals, the structural report, the
  tender conditions and the invoices;
- the fee arithmetic and the GST on every invoice;
- the dates, from investigation to report, invoices and tender.

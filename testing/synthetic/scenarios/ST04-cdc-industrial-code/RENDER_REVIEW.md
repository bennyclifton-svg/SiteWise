# ST04-cdc-industrial-code: render review

Reviewer: independent read (brief 04r), 7 November 2026. Base: ST04-base (corpus final and reviewed).

## Verdict: PASS

There are no blocking findings. The nine rewritten documents read as a credible complying development
certificate (CDC) job, and nothing in the corpus makes the planning pathway a conflict. There are four
findings: one stale manifest line, one owner note about a base fact, and two optional realism additions.

## Checks run

1. **Lint.** `python testing/synthetic/tools/lint_corpus.py testing/synthetic/scenarios/ST04-cdc-industrial-code`
   printed `ok`, exit 0.
2. **Copies.** `cmp` against `ST04-base/corpus` shows all 7 copies are byte-identical: d02 planning
   certificate, d03 survey, d08 geotechnical fee, d11 geotechnical report, d15 A-1100 Rev A, d17 and d19
   invoices.
3. **Diff of the nine rewritten documents against the base.** Each one changes only DA to CDC wording.
   - d01: the CDC sentence, the code-standards sentence, the certifier determination sentence, the s7.12
     contributions line, and the reworded clause 4.6 and community sentences.
   - d04: subject and body.
   - d05: CDC documentation phase; the development manager lodges.
   - d06, d07, d10, d12, d14: "for the CDC".
   - d10: certifier RFIs.
   - d16: "Amended for CDC".

   Every amount is unchanged, and fees.json phase labels follow the new wording. A search for `DA`,
   `development application`, `development consent`, `council staff`, `planning panel`, `planner` and
   `Westbourne` finds only the planned d01 sentence ("no planning panel"). It also finds d02's
   "permitted with consent", which is judged below.
4. **Stray triggers.** I searched the nine documents for the shape.yaml trigger evidence and the DOMAIN.md §10
   traps: flood, bore, excavated to a depth, building [a-d], stage N / staged, road reserve, TfNSW / traffic
   generating, contaminat, heritage, public exhibition, demolition, basement, kitchen, and storage or racking
   within a sentence of fire.
   - Every hit is a planned carry or trap: d01 no existing building / no basement / one continuous stage
     / staff kitchen / validation report, d10 TfNSW denial, d14 hydrocarbon contamination.
   - The `cdc-industrial-code` negation phrases ("A development application will be lodged with council",
     "exceeds the code's height standard") are absent.
   - d05's "will lodge the CDC application" is the planned `flag.architect_lodges` (no) source: the
     development manager lodges, not the architect.
5. **Title blocks.**
   - d01, d05, d06, d10, d12, d14 and d16 have label lines before the first `##` and one `#` heading equal
     to `Title:`.
   - d07 is a messy table title block, as its tier says.
   - d04 is an email header with the subject "RE: Program and CDC timing", which matches the register.
   - Numbers, revisions and dates match `expected/document_register.json`. The d12, d14 and d16 issue status
     "for CDC" follows the same pattern as the base's "for DA". On d16 it is literal ("Issue: Amended for
     CDC"); on d12 and d14 it is `unsupported`.

## What a PM concludes before opening expected/

- **Building and work.** A Class 7b warehouse with a Class 5 office, rise in storeys 1, Type C construction.
  New build on a vacant, benched 1.6 ha E4 lot. GFA 8,000 m2 (7,500 + 500), 12 m clear, 12 recessed docks,
  ESFR, large isolated building with performance solutions for travel distance and perimeter access.
- **Planning pathway.** CDC under Codes SEPP Part 5A (Industrial and Business Buildings Code), determined by
  the registered certifier, with s7.12 contributions. No DA, no panel, no clause 4.6, no SEE and no town
  planner. This is stated in d01 and d05 and supported by d04, d06, d07, d10, d12, d14 and d16.
- **Procurement.** D&C lump sum under AS 4902. The architect, structural engineer, civil engineer and
  landscape architect are novated.
- **Consultants.** Architect, structural, civil, geotechnical, traffic, BCA and surveyor documents are in the
  corpus, and a landscape architect is named for novation. No town planner. A certifier is implied.
- **Site conditions.** None of these apply: heritage, bushfire, flood control, acid sulfate soils,
  contamination, coastal, rail. Not in a drinking water catchment. Sewer and water connections are provided.
  There is a 3 m drainage easement on the east side. No groundwater. Site class M, moderately saline soils.
  Traffic is below the Schedule 3 thresholds and the site has no classified road frontage.
- **Budget.** $14,800,000 ex GST. Contributions, land and fees are excluded.
- **Fees.**

  | Discipline | Total | Option |
  |---|---|---|
  | Architect | $240,000 ex GST | $165,000 |
  | Structural | $76,000 ex GST | $58,000 |
  | Civil | $88,000 inc GST ($80,000 ex GST) | $49,500 inc GST |
  | Geotechnical | $24,000 ex GST | $2,800 |
  | Traffic | $19,000 ex GST | — |

- **Invoices.** MHA-INV-0311: $38,000 + $3,800 = $41,800. SHG-INV-0507: $18,800 + $2,820 at 15% = $21,620,
  which is a GST rate error.

This matches `expected/profile.json` (`hdr.cond.planning: cdc` from d01 and d05; no conflicts),
`fees.json` and `invoices.json` with no differences. The d07 inc-GST normalisation is exact:
35,200, 30,800, 22,000 and 49,500, each divided by 1.1.

## d02 "permitted with consent": judged, no change

Item 2 says "Warehouse or distribution centres are permitted with consent in the zone". This is standard
s10.7 wording for the land-use table, and it is the precondition for complying development, not a rival
pathway. The code applies only where the use is permitted in the zone under the LEP. Item 4 then says the
Industrial and Business Buildings Code may be carried out on the land. Read together, the certificate supports
the CDC. The key keeps d02 out of `hdr.cond.planning`'s sources (trap `cdc-available-on-land`), so the
certificate cannot set the pathway either way. Leave it as is.

## Is the CDC lawful and credible for this warehouse? Yes, on the base facts

Source: DPHI, *Complying Development for Business and Industry: A Pocket Guide* (May 2023), pp. 11-13. This
is a secondary source, not the instrument, so the standard stays `verified: false` under AGENTS.md rule 5. For
E4 General Industrial new buildings it gives these limits:

| Standard | Code limit | This project | Result |
|---|---|---|---|
| Gross floor area | 50,000 m2 | 8,000 m2 | Under |
| Height | The greater of the LEP height or 18 m | 12 m clear, single storey; the ridge is plausibly about 14-15 m | Under 18 m |
| Floor space ratio | LEP, otherwise 1:1 | 0.5:1 (8,000 m2 on 16,000 m2) | Under |
| Front setback | DCP, otherwise 10 m or the neighbours' average to a non-classified road | d01 and d05 state compliance with a landscaped setback | Met as stated |

The guide also lists eligibility conditions and land exclusions. The base facts clear every one:

- the use is permitted in the zone (d02);
- no heritage item or conservation area (d02);
- not acid sulfate soils class 1 or 2 (d02, not mapped);
- not bush fire prone (d02);
- no significant contamination (d01 validation, d02);
- not unsewered land in a drinking water catchment (d03 sewer junction, d07 not in a catchment);
- no critical habitat or biodiversity agreement (d01 no ecological constraints);
- no concurrence (d10 below Schedule 3, no TfNSW referral);
- not designated development;
- no environment protection licence (d01: dangerous goods only in minor quantities).

The code also bars building over a registered easement. The 3 m eastern drainage easement (d03) receives the
site's stormwater (d14), and nothing places the building on it. Nothing in the base facts excludes the CDC.

## Findings

1. **`manifest.json` `project.summary`, phrase "Local DA determined by council staff, then an AS 4902 design
   and construct lump sum…"** The summary was carried over from the base and contradicts the variant's own
   trigger and documents. No corpus document repeats it, so there is no scoring effect, but a later renderer
   or reviewer could be misled.
   Change: "Complying development certificate under the Industrial and Business Buildings Code (Codes SEPP
   Part 5A) determined by a registered certifier, then an AS 4902 design and construct lump sum…". This is an
   owner or manifest edit; this review does not make it.

2. **Owner note on a base fact: ST04-base d01, "The building exceeds the code's height standard, so a DA is
   required"** (also the shape.yaml `cdc-industrial-code` negation).
   - The variant reverses this without changing any dimension. Under the 2023 guide the E4 height standard
     is the greater of the LEP height or 18 m, so a single-storey warehouse with 12 m clear height does not
     plausibly exceed it.
   - The CDC variant is therefore the more credible one. The base's stated reason for a DA is doubtful.
   - In this variant, `expected/notes.md` ("Whether a 12 m clear-height warehouse meets the code's height
     standard is `verified: false`") can cite the guide. It must stay `verified: false` until it is checked
     against the instrument.
   - For the base, consider giving the DA a reason the code really imposes. Examples: a voluntary DA for an
     unresolved performance solution, or a site that does not meet a code standard. Rule 5 applies: confirm
     the clause first. No change in this variant.

3. **Optional realism, d01 `## Approvals and procurement`, after "…so the certifier can determine it."** The
   height claim is asserted with no figure. A PM would expect the overall height to appear next to the code
   claim. Suggested added sentence:

   "The ridge is about 14.5 m above finished ground, under the code's 18 m maximum building height."

   It hits no trigger. It is deliberately not a `Label: value` line, so it cannot compete with
   `Clear height: 12 m` for `hdr.scale.clear_height_m`. If it is added, add it to d01's content note as
   `verified: false`. Not required for PASS.

4. **Optional realism, d12 BCA report, first paragraph, "…with the performance solutions noted, for the
   certifier's determination of the CDC."** A CDC cannot leave performance solutions to a later construction
   certificate. A real CDC-stage BCA report would say how and when they are resolved. Suggested addition to
   that paragraph, outside `## Fire engineering` so it does not touch the reviewed `fire_specialist_systems`
   wording:

   "The performance solutions will be documented in a fire engineering report and resolved before the CDC
   is determined."

   Keep "storage" and "racking" out of the sentence. Fire and Rescue NSW consultation could also be named,
   but only after the clause in the Development Certification and Fire Safety Regulation 2021 is confirmed
   (rule 5). Not required for PASS.

## Realism

All nine rewritten documents read like the real thing. The CDC wording is idiomatic: "certified scheme",
"conditions of the complying development certificate", "certifier requests for information", "neighbour notice
is given before work starts", "s7.12 contributions still payable under the CDC".

One phrase reads slightly as written for a test: d01 "no clause 4.6 variation request is required". Under a
CDC, clause 4.6 is not available at all. The sentence is the planned `flag.cl46` carry and is not false, so it
stays.

The d04 key `proj.target_da_lodgement` now holds a CDC lodgement date. This is noted in `expected/notes.md`
and missed by design.

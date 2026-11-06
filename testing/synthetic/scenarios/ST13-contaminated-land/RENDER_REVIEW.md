# ST13-contaminated-land: independent render review

Reviewer: independent senior PM read, 6 November 2026. Brief: `testing/synthetic/briefs/04r-render-review.md`.
Scope: the variant. The ST13-base corpus is final and already reviewed (`ST13-base/RENDER_REVIEW.md`), so this read
looks hardest at the rewritten d02, d05 and d13, the added d19, and whether any copied document still denies
contamination.

## Verdict: FIX

Five findings. One is moderate: the rewritten PSI moves the rear car park onto No. 40, which three unchanged
documents contradict. The other four are minor wording fixes. The contamination signal is clean. No copied document
still denies contamination, and the key's only change (`det.potentially_contaminated_land` true) is carried by d02,
d05, d13 and d19.

## 1. Lint

`python testing/synthetic/tools/lint_corpus.py testing/synthetic/scenarios/ST13-contaminated-land` gives
`ok  testing/synthetic/scenarios/ST13-contaminated-land`.

## 2. What a PM concludes from the documents alone

- New six-storey mixed-use (shop-top) building: three shell shops, 42 apartments, two basement levels with 55 spaces.
  Class 2/6/7a, sprinklered, rise in storeys 6. Three existing buildings (two 1965 shops and a dwelling) are to be
  demolished.
- DA to the City of Ryde, decided by the Local Planning Panel. E1 Local Centre, no heritage, bushfire, flood or acid
  sulfate soils. D&C with novation, AS 4902 lump sum, DBP Act applies.
- **Site condition:** potentially contaminated land. Council's s10.7 records a service station on No. 40 until 1992.
  The PSI finds a vent stub and a patched slab (a likely tank pit), TRH and BTEX above HSL A in two pits, and a former
  dry cleaner at No. 42 (1971-2004) that needs soil vapour sampling. It recommends a DSI, then a RAP, with an
  accredited site auditor engaged before determination. The SEE carries the same route into the DA. Basement 7 m in
  shale, no groundwater (seepage only), excavation next to No. 38 and No. 46.
- Consultants: architect, structural, civil, geotechnical, town planner, BCA, surveyor, and now an **environmental
  consultant** (Kestrel, from its own proposal). The client engages the **site auditor** directly. That is a separate
  appointment that is not yet priced.
- Fees: the base six, plus Kestrel KE-FP-0611: DSI $38,500, RAP $14,000, validation $22,000, **total $74,500 ex GST**.
  Option: test pits, $6,800. Site auditor excluded. Invoices are unchanged from the base: Calder Rowe $74,800, Ashgrove
  $25,850, Basalt Ridge $22,420 with the mistyped total.

## 3. Comparison with `expected/`

`profile.json`, `fees.json` (d19 entry: firm, ABN, contact, phone, address, the three stages, the option and the
excluded site auditor) and `invoices.json` all agree with the reading above. I found no difference in a keyed value.
`hdr.cond.environmental_sensitivity: standard` stays as the base. It is read from d01, and contamination is a
determinant here, not a sensitivity, so that is consistent.

## 4. Stray triggers

The `contaminated-land` evidence phrases appear only where the manifest plans them: d05 (all three), d19 ("a dry
cleaner operated in the ground-floor shop") and d13 ("the PSI recommends a detailed site investigation"). The negation
"The PSI found a low risk of contamination…" is gone from every document. No other T13 trigger's evidence appears. The
only §10 trap word in the corpus is "groundwater" in d11, which is base and planned. d02's "(a) The land is not
significantly contaminated land" is the statutory s59(2) wording, and the next paragraph resolves it: "The land is
potentially contaminated land". It is not a denial. d19's "lodged with the development application" adds one more
`lodge` match. The base already has many, so the key is unaffected (see finding 5, which removes it anyway).

**Contamination denials in copied documents:** none. d01, d03, d04, d06-d12 and d14-d18 do not mention contamination.
d11's spoil paragraph only says spoil is classified before disposal.

## 5. Title blocks

- d05 (clean): Title, Report no KE-26-0310, Revision 0, Report date 10 March 2026, Prepared for. There is one `#`
  heading that matches Title. All values match the register.
- d19 (clean): Title, Proposal no KE-FP-0611, Date 27 March 2026, To. There is one matching `#` heading. All values
  match the register.
- d13 (adversarial): the title block is unchanged from the base and matches APL-26-031-SEE, 22 May 2026.
- d02 (messy): a table title block with PC-2026-01187 and 10/02/2026, and the `#` heading "Planning Certificate". This
  is correct for its tier.

## 6. The writer's two departures

**(1) Kestrel's address and phone in d05 changed to match d19 and fees.json: accept.** One firm with one named
principal cannot have two offices and two direct numbers in the same corpus three weeks apart. `fees.json` holds the
Parramatta address and 02 9806 4471 as the key for d19, so d05 must agree with d19, not the other way round. d05 is not
a fee document, so nothing is scored from its letterhead. ST13-base and ST13-rise-three-storeys keep the Burwood
letterhead. That is harmless because each scenario's corpus is loaded on its own.

**(2) Rear car park: d05 is wrong; change d05, not d09.** See finding 1.

## Findings

1. **d05: the PSI puts the car park behind No. 40, against d03, d09 and d11 (moderate).**
   The base corpus says three times that the car park is at the rear of the shops and the yard is behind the dwelling.
   d03: "The rear of Nos. 42 and 44 is a sealed car park; the rear of No. 40 is a grassed yard". d09 line 26 and d11
   lines 26-27 put BH1-BH2 in the "rear car park of Nos. 42 and 44" and BH3-BH4 in the "rear yard of No. 40". The base
   review set that layout on purpose, and those three documents are final. The rewritten d05 alone says "now the open
   car park behind the dwelling" and "rear car park of No. 40". A PM reading the set would ask which is right.
   The error is upstream. `ST13-base/REVIEW.md` finding 6 proposed "now the open car park behind the dwelling" while
   fixing "behind the shops", and that wording went into the manifest's d05 `content` and `expected/notes.md`. The
   tank pit stays on No. 40, as d02 and d19 require. Only the surface changes.
   - d05 line 31: "The rear part of No. 40, now the open car park behind the dwelling," -> "The rear part of No. 40,
     now the grassed yard behind the dwelling,"
   - d05 lines 44-46: "In the rear car park of No. 40 there is a cut-off vent pipe stub against the rear fence and a
     rectangular patched slab of about 6 m by 3 m," -> "In the rear yard of No. 40 there is a cut-off vent pipe stub
     against the rear fence and a rectangular concrete slab of about 6 m by 3 m in the grass,"
   - d05 line 51: "Six shallow test pits were dug through the rear car park and yard" -> "Six shallow test pits were
     dug in the rear yard of No. 40 and through the car park behind the shops"
   - d05 lines 55-56 and 58 then read correctly, because "the patched slab" becomes "the slab". Change "nearest the
     patched slab" to "nearest the concrete slab".
   - d19 line 58: "beyond the patched slab" -> "beyond the concrete slab".
   - Manifest d05 `content`: "now the open car park behind the dwelling" -> "now the grassed yard behind the dwelling",
     and "a patched slab in the car park" -> "a concrete slab in the rear yard of No. 40".
   - `expected/notes.md` line 3: "(now the car park behind the dwelling)" -> "(now the grassed yard behind the
     dwelling)".
   - Do not change d09, d11 or d03.

2. **d13: "Chapter 4 of the SEPP" points to the wrong SEPP (minor).**
   The paragraph just before it is the Housing SEPP, Chapter 4. The new paragraph opens "Chapter 4 of the SEPP applies
   because the proposal introduces residential use." and only names Resilience and Hazards in its second sentence, so
   it reads as two drafts pasted together. Keep the manifest's `says` verbatim and move the applicability clause after
   it. Lines 46-47 become: "**Contamination.** SEPP (Resilience and Hazards) 2021 Chapter 4: the land is potentially
   contaminated. The site was a service station until 1992 and a dry cleaner operated in the shop at No. 42; the PSI
   recommends a detailed site investigation, and a remediation action plan and site audit statement will show the land
   can be made suitable for residential use. Chapter 4 applies because the proposal introduces residential use."

3. **d05 line 40: the summary sentence sits mid-history and reads as inserted (minor).**
   "The site was a service station until 1992; underground tanks may remain. A dry cleaner … The PSI recommends a
   detailed site investigation to define both." follows a bullet list and comes before the inspection and sampling it
   anticipates. A consultant would not put its recommendation there. It also says "the site" right after a bullet that
   correctly limits the service station to the rear of No. 40. Move the whole sentence, unchanged, so that it opens
   `## Conclusions`, before "The land is potentially contaminated." The verbatim `says` stays, and the sentence then
   reads as a summary.

4. **d05 Site history no longer says when the shops and dwelling were built (minor).**
   The rewrite dropped the base bullet "The shops on Nos. 42 and 44 and the present dwelling on No. 40 appear by
   1965…". The history now goes from "a single dwelling with open yards" (1943, 1951) to a service station from the
   mid-1950s, and the shops never appear. The last paragraph's "The shops and dwelling date from the 1960s" then has
   nothing to rest on. d01 says the shops were built in 1965. After the service-station bullet (line 34), insert:
   "- The shops on Nos. 42 and 44 and the present dwelling on No. 40 appear by 1965 and are unchanged in form in every
   later photograph."

5. **d19 and d13 disagree on when the RAP reaches council (minor).**
   d19 line 40 says the RAP "will be prepared for review by the site auditor and lodged with the development
   application", and line 78 says it will "accompany the development application". The later SEE (22 May) says only
   the PSI accompanies the application, the DSI is under way, and the auditor will be engaged before determination. A
   PM would reconcile these to the SEE, the later and controlling document. Both edits also remove one `lodge` match.
   - d19 line 40: "and lodged with the development application." -> "and provided to council before the development
     application is determined."
   - d19 lines 77-78: "in time for the remediation action plan to accompany the development application." -> "so that
     the remediation action plan can be with council before the development application is determined."

## Checked, no finding

- d02 reads like a real s10.7 certificate. Item 8 now follows the CLM Act s59(2) form, (a)-(c), with council's record
  of the service station after it. It is correct not to mention the dry cleaner, because that is not on council's
  record.
- d19 reads like a real environmental proposal: GPR sweep, twelve bores, four sub-slab vapour points, a conceptual
  site model, remediation by excavation that fits the basement, auditor excluded, physical works by the contractor.
  The single-row "Excluded item | Site auditor | Excluded" table is a little tidy, but it duplicates a real exclusion
  bullet, so I would leave it.
- d11 (unchanged) drilled BH3-BH4 in the rear yard of No. 40, which is the former service station area, and logged
  no odour. A real geotechnical log there might note hydrocarbon odour. The report does not deny contamination,
  though, and its fill (0.0-0.6 m) is above the PSI's 0.9 m odour. Leave it unchanged, as `notes.md` intends.
- d13's expected conditions (remediation and validation before the construction certificate, site audit statement
  before occupation) are the usual Ryde-style conditions. They are consistent with remediation done as an early-works
  excavation.

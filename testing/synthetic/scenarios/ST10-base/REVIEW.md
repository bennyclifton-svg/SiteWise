# ST10 review: knock-down rebuild (base, da-pathway, separate-demolition-contract)

Reviewer: independent (brief 03r), 2026-11-06. Scenarios and shape not edited.

## Verdict: FIX

The keys are right. Four wording changes to manifests reduce the chance that a reasonable app reading is
scored as wrong. No expected value changes.

## Checks

1. **Checker.** `python testing/synthetic/tools/validate_expected.py testing/synthetic/scenarios/ST10-*`:
   `ok` for all three scenarios.
2. **Faithful to the shape.** The base matches the shape's base profile. It also keys `det.site_class` M (d12) and
   `flag.arborist_offsite_trees` no (d13), which the shape lacks (see ruling 1). Declared traps cover the
   regex hits I found: staged (d05 fee stages), authority_asset (d03), osd (d03, d07), excavation_adjoining (d12),
   contamination (d11), gas (d13), and demolish near trees (d13). d02 complying-development eligibility is also
   declared.
3. **Register keys.** I checked every title block against DISCOVERY §4.3. Label lines settle the number,
   revision, title and date for d01, d03, d05, d06 and d08 to d15. `Ref`, `Proposal no`, `Quote no`,
   `Report no` and `Document number` are all number labels. d04 is an email, so its date is the latest header,
   3 March. In d02 and d07 the number and date sit in tables, so only the `#` heading title is read. The notes
   say this. d15 superseding d14 is keyed, and the notes say the app cannot link it today. Invoices go to the
   ledger, not the register.
4. **Profile keys.** Each value is carried by a read document. The d04 email holds the only statement of
   `proj.target_construction_start` and `proj.regulated_design`, so both are correctly `read_by_design: false`.
   `det.site_class` has a reader (`knowledge/determinants.yaml` site_class, trigger `site class(ification)?`), so
   it is correctly `read_by_design: true`.
5. **Fees and invoices.** The d07 GST-inclusive figures convert to 180,000 + 420,000 + 300,000 + 150,000 =
   1,050,000 cents, which is $11,550 / 1.1. Every other total adds up. `gst_inc_normalised` and `abn_invalid` are
   real flags (`internal/proposals/read.go`, `internal/invoices/ledger.go`). All eight ABNs pass the checksum
   except the deliberate d17 transposition.
6. **Variants.** In da-pathway only `hdr.cond.planning` moves (cdc to da), and d21 adds sources. In
   separate-demolition-contract no expected value moves; only the d05 Stage 4 fee label changes, and the amounts
   stay the same. Both variants use the shape's evidence verbatim.
7. **Realism.** The job reads as a credible Sutherland Shire knock-down rebuild under a CDC. The dates follow a
   sensible order: brief, s10.7, survey, fee proposals, reports, CDC set, then the first invoices. The fees are
   plausible for a $1.52M high-end house. The architect's $86k to tender is about 5.7%, and civil at $10.5k with
   CCTV is high but credible.

## Findings

1. **ST10-separate-demolition-contract d01: the staged negation is weak and may sit in another passage.**
   The new sentence "there is no staged construction *of the house*" concedes that demolition is a stage. The
   `staged` asks text (`knowledge/profile/flag_readings.yaml`) excludes fee, payment, design and approval stages,
   but not a demolition contract. Change the negation to: "Demolition before the builder starts is not staged
   construction; the new house is built by one builder under one contract." Put it in the same paragraph as the
   Stage 1 / Stage 2 sentence, so the one passage Jev reads holds both. Keep `flag.staged` "no" (see ruling 2).
2. **ST10-da-pathway d13 and d21: an undeclared `community` trap.** The text "Neighbour notification under
   council's community participation plan; no public exhibition" wakes `community` through the trigger
   `public exhibition`. Neighbour notification can reasonably be read as consultation. "no" is right for the RFP
   flag, but the passage is ambiguous. Reword to: "No community consultation is required beyond council's
   standard neighbour notification; no public exhibition." Alternatively, declare it as a trap in both documents.
3. **ST10-base d07 (and both variants): the public-road statement contradicts the scope.** The proposal says "No
   public road works are required", yet the builder rebuilds the vehicle crossing. That crossing is work in the
   road reserve under a council s138 approval, and it appears in the shape's `site.public-domain` work item.
   Keep the key "no", because no civil road design is in scope. Reword to: "The civil scope includes no public
   road works; the builder rebuilds the vehicle crossing to council's standard drawing under council's
   approval."
4. **ST10-base d12 (and both variants): a minor realism problem.** The text says "No groundwater was encountered
   in the boreholes to 3.0 m", but sandstone sits at about 2.6 m. Reword to "to refusal on sandstone at about
   2.6 m". No key changes.

## Rulings on the author's shape questions

1. **Add `det.site_class: M` and `flag.arborist_offsite_trees: "no"` to the T10 base profile: yes.** R10's
   reason is "for a new house the normal base case", and a knock-down rebuild is a new house on a new slab with
   new footings. The scenarios already key both values from read documents (d12, d13), so no scenario changes.
   `flag.arborist_offsite_trees` is already listed by the `trees-affected` trigger. R10 names T01 only, so the
   owner should confirm before the shape change is marked reviewed.
2. **"Stage 1" in separate-demolition-contract: the expectation is correct. Keep it as a trap; do not reword
   it away.** The phrase is the shape's own evidence, and check 6 requires it verbatim. It is also how owners
   actually write. DOMAIN.md §10 trap 7 holds that demolition-then-build on one house is not staged delivery. The
   `staged` flag only switches on the architecture RFP's staged and multi-building option and workshops
   (`knowledge/rfp/architecture.yaml`), and a pre-start demolition contract needs neither. If the app answers
   "yes", score it as an app defect, not a key error: the `staged` asks text should exclude demolition or site
   clearance before the building works. That is a knowledge change outside this review. Strengthen the negation
   as in finding 1. The shape's `why` ("'Stage 1' wakes flag.staged") should read "wakes the staged question;
   the expected answer stays no".

# Run 20261107-t03-t04-t06: T03, T04 and T06

First live run of the office fit-out (T03), the warehouse (T04) and the
sprinkler pump and tank upgrade with fit-out (T06): a base and two variants
each.

- **App:** local main `0aee9a4` plus another session's uncommitted edits
  (`internal/httpapi/profile.go` scope-decision display field,
  `knowledge/profile/layout.yaml`, `internal/knowledge/layout_test.go`, web
  display files). None of them changes a reading, scope row, package, cost
  line, program row or RFP position.
- **How it ran:** `serve.sh fresh` (empty database), live Jev, one scenario
  at a time (`run1.log` to `run3.log`). No HTTP errors; every RFP draft
  returned 201.
- **Scores:** `field_score.*` (register, profile, invoices, fees) and
  `shape_score.*` (procurement, cost plan, program, RFPs) in each scenario
  folder. Detail and code locations: `FINDINGS.md`, section "From run
  `runs/20261107-t03-t04-t06`" (F43 to F50).

## Outcome

- **Money is right everywhere.** Every fee total, firm name and invoice
  amount is correct in all nine runs, with no fabricated amounts. The three
  planted invoice faults are all flagged and nothing else is: T03 bad
  arithmetic, T04 GST at 15 %, T06 GST at half rate.
- **Every variant's changed value is read:** planning `da` (T03, T06) and
  `cdc` (T04), and `det.flood_hazard` true (T04). Each variant gets the
  right program skeleton.
- **One wrong profile value per run.** T03: occupied tower read as
  `live_environment` not `partial_occupation` (owner question). T04: a
  vehicle crossing read as public road works (owner question). T06: access
  read as restricted hours, not urban constrained (owner question).
- **The biggest new defect is F43:** on a refurb, works whose action is
  below the 0.85 floor or conflicting are treated as existing and drop out
  of scope. T03 loses its partitions, ceilings, joinery, floors, lighting
  and cabling from the program; T06 loses the fire water and sprinklers, and
  the pump-tank-only job puts its whole $1.2M on Hydraulic.
- **F47:** every new-build program is undated (T04 here, and T02, T10, T13
  re-checked), because the empty Investigation stage counts as a leaf of
  unknown length. Earlier "no links violated" results on new builds were
  not actually scorable.
- **RFP new failures 7–14 per run,** most of them stale certifier/BCA keys
  written before F9, triggers that do not restate `expect`, and the public
  road question.

## Per scenario

| Scenario | Profile correct / wrong | Fees (total, firm) | Invoices | Procurement | Cost plan | Program matched | Links violated | RFP new failures |
|---|---|---|---|---|---|---|---|---|
| ST03-base | 43 / 1 | 6/6, 6/6 | 3/3 | 9/13 | 21/38 | 29/71 (41 %) | 0 | 7 |
| ST03-da-pathway | 44 / 1 | 7/7, 7/7 | 3/3 | 10/14 | 24/39 | 33/73 (45 %) | 0 | 11 |
| ST03-strip-out | 43 / 1 | 6/6, 6/6 | 3/3 | 9/13 | 22/39 | 25/74 (34 %) | 0 | 7 |
| ST04-base | 64 / 1 | 6/6, 6/6 | 3/3 | 13/20 | 33/49 | 51/95 (54 %) | undated (F47) | 9 |
| ST04-cdc | 64 / 1 | 5/5, 5/5 | 2/2 | 12/19 | 31/49 | 49/91 (54 %) | undated (F47) | 14 |
| ST04-flood | 63 / 1 | 6/6, 6/6 | 3/3 | 13/20 | 33/51 | 51/97 (53 %) | undated (F47) | 12 |
| ST06-base | 43 / 1 | 7/7, 7/7 | 3/3 | 7/9 | 16/36 | 24/85 (28 %) | 1 (scorer) | 12 |
| ST06-tank-consent | 43 / 1 | 8/8, 8/8 | 3/3 | 9/11 | 17/37 | 26/86 (30 %) | 0 | 14 |
| ST06-pump-tank-only | 44 / 1 | 6/6, 6/6 | 3/3 | 6/7 | 13/28 | 18/57 (32 %) | 0 | 12 |

Notes on the table:
- T04 also has a planned site-area conflict (16,000 against 16,012 m²),
  scored `conflict_true`.
- T06 has three `conflict_true` values: occupation, `flag.staged`, and the
  client, which F46 makes a conflict and which blocks five RFPs.
- The one T06 "violated" link is a start-to-start link the scorer reads as
  finish-to-start (scorer limit).
- Filing: discipline 14–17 correct per run; T06 files the FS-101 fire
  services drawings as fire engineer (F49).

## What worked

- Fee totals, firm names, invoice amounts and invoice flags: all correct.
- F9 holds: the certifier prices `b2_cdc` on a CDC. F20 holds: no CDC job
  has a DA fees line. F37 holds on a fresh database.
- The flood reaches the civil RFP (flood report, evacuation plan, trunk
  drainage).
- On the fit-outs the tenant is the client, and the landlord's base-building
  NABERS rating does not set a sustainability target.
- T04 now reads the survey's site area and most "no" flags (`cl46`,
  `community`, `staged`, `multi_building`, `panel`, `basement`,
  `osd_existing`).

## New app findings (detail in FINDINGS.md)

- **F43** Refurb works with a below-floor or conflicting action are filed
  as existing and leave scope (`internal/profile/existing.go`
  `existingPresence`).
- **F44** The three T03 program-order defects (fire stopping on day 0,
  services after ceilings, the sprinkler-to-ceilings link dropped as a loop)
  are still in the code; confirmed offline, hidden live by F43.
- **F45** `implied_by` scope suggestions fire on denials ("bathrooms ... are
  not in scope" adds wet-area works; "No strip-out" adds demolition).
- **F46** A "Name, Role, Company" addressee line is offered whole as the
  client, making the T06 client a conflict.
- **F47** New-build programs have no dates: the empty Investigation stage
  is an unknown-length leaf.
- **F48** "Local government area: North Sydney" is not an LGA candidate.
- **F49** Fire services drawings file as fire engineer.
- **F50** Interior Design is suggested on a services-only refurb.

Repeats: F16 (rise in storeys only from a BCA report), F18 on a refurb
(access review passages set doors to investigate), F29 ("Establish the
site" has no predecessor), the F41 family (`flood_exposure` keys a field
that does not exist; no effect here), some missed "no" flags, and
`proj.regulated_design` (scored by design).

## Key and shape issues (recommended, not applied)

- T03 and T06 certifier and BCA keys predate F9 and should be re-keyed
  (7 RFP failures per T03 run, 4 per T06 run).
- Variant skeletons and `expect` restatements for T03 da-pathway, T04
  cdc-industrial-code and flood-affected, and T06 tank-needs-consent.
- T06 `position.existing` should be `absent`, not `ask`.
- T03 and T06 `hdr.cond.quality_level` applies only to residential: mark it
  not read by design.
- From rendering: ST03-da-pathway should key `flag.architect_lodges` no
  (the checker refuses a key the base lacks); ST04 keys `det.site_class` M
  and `det.saline_soil` true, which the T04 shape lacks.

No key was changed after the run.

## Owner questions

1. Is a vehicle crossing "work in a public road"? (T04; 9 RFP failures per
   run hang on it.)
2. F43: on a refurb, should an included system with an unsettled works
   action stay in scope?
3. Should a vacant floor in an occupied tower accept both
   `partial_occupation` and `live_environment`, and T06 access both
   `restricted_hours` and `urban_constrained`?
4. Should a combined services proposal fold into the discipline packages
   rather than add a "Services Engineer" package (T03 carries five services
   fee lines)?
5. ST04-base gives "over the Industrial and Business Buildings Code height"
   as its reason for a DA, but E4 height is the LEP height or 18 m, so a
   12 m warehouse is under it. Keep the reason or reword it?
6. Move T03 `det.type_of_construction` to `expected_derived`?

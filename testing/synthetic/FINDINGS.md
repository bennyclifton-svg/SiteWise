# Findings

From pilot runs. Each entry: what was expected, what the app did, the
suspected cause, and where to look. A key is changed only when it was
wrong, with the reason in that scenario's `notes.md`. "Suspected" means not
yet traced in the code.

Run `runs/20261031-pilot` (T01 base, CDC and trees variants; app at
`e233aa6` plus the owner's uncommitted working tree), 31 October 2026.

## Status after the fixes (4 November 2026)

Fixed and confirmed by re-running all nine pilot scenarios
(`runs/20261104-after-fixes/REPORT.md`): F1, F2 (LGA and address), F3, F4,
F5, F7, F8, F13, F14, F15, F16, the care-of client and the table invoice
date. F9 to F12 are fixed in the RFP knowledge and rules (unit tests and
answer keys; inspection and BASIX clause numbers stay `verified: false`).
F6 was fixed on 2 November but is fragile: in one of nine runs the BASIX
proposal filed blank at Jev 0.59. Plan:
`docs/plans/2026-11-04-synthetic-findings-fixes.md`.

Still open (not confirmed as app defects): T01's missed "no" flags, gas
supply, contract form and existing building; `flag.authority_asset`,
`flag.community` and access constraints as conflicts; `proj.regulated_design`
(no reading exists); remediation has no skeleton.

## Status after the fixes (6 November 2026)

Fixed and confirmed by re-running all nine T10 to T12 scenarios
(`runs/20261106-t10-t12-after-fixes/REPORT.md`): F17 (where the survey's
reading clears the floor), F18, F19's old-house votes and apartment
fallback, F20, F21, F22, F28 and F30. F23 is fixed in code; the re-run
reused the firm saved in the first run. F25 is an alias only and barely
changes filing, because Jev sees only the label. Plan:
`docs/plans/2026-11-06-knockdown-two-buildings.md`.

Still open: F6; F24, F26, F27 and F29.

## Status after the fixes (7 November 2026)

Owner rulings R11 to R14 (`reference/OWNER_RULINGS.md`). Fixed and
confirmed by re-running the eight live T02, T13 and T17 scenarios on a
fresh database (`runs/20261107-t02-t13-t17-after-fixes/REPORT.md`): F31,
F32 (units, shop area, site area; R14), F34, F35 (R12, majority by
documents at 2x), F36, F37, F38, F39, F40, F41, F42, F27 (R12 counts
documents) and R13 (architect brief always, masterplan only when large).
F33 is fixed in code, but `flag.cl46` stays blank live with the other
missed "no" flags. Plan: `docs/plans/2026-11-07-synthetic-findings-f31-f42.md`.

Still open: the missed "no" flags (T13 `cl46`, `public_road`, `community`,
`architect_lodges`); `proj.regulated_design`; the survey's site-area line;
F6, F24, F26 and F29.

Owner ruling on the T11 subclass (6 November 2026): a dual occupancy is a
house, because there is no strata scheme. Townhouses are a registered
strata or community scheme with shared common property. The T11 base and
CDC keys now expect `house`; the four-townhouse variant keeps `townhouses`.
The subclass criteria carry the rule (`profile-13`).

## Status after the fixes (7 November 2026, T03, T04 and T06)

Re-ran the nine T03, T04 and T06 scenarios unchanged on a fresh database
(`runs/20261107-t03-t04-t06-after-fixes/`; app at local main `bd24ab1`
plus another session's uncommitted display edits). Both runs are scored
with the same current shapes and keys. The T06 sprinkler rows were
corrected after the re-run (see below), and both T06 runs were re-scored.
Plan: `docs/plans/2026-11-07-synthetic-findings-f43-f50.md`.

**Fixed and confirmed:** F45, F46, F47, F48, F49 and F50. F43 and F44 are
fixed as far as the run can show, with a residual for F43 (F51).

- **F43 (R16), mostly fixed.** Systems filed as existing fall from 18 to 2
  (T03 base), 7 to 2 (T06 base) and 2 to 1 (pump-tank-only).
  - T03 now has partitions, ceilings, glazing, joinery, floors, lighting,
    cabling, switchboards and comms in scope. Its cost plans gain
    Electrical and Communications lines, and its programs gain partitions,
    ceilings, joinery, luminaires, cabling and floors.
  - T06 now has `fire-active.fire-water` and `sprinklers` in scope.
    ST06-pump-tank-only splits its $1.2M as Active fire protection
    $800,000 and Hydraulic $400,000 (was all Hydraulic). T06 base no longer
    puts $1.8M on Mechanical.
  - Still filed as existing: `structure.alterations` in all T06 runs and
    T03 painting in all three runs (F51).
- **F44, fixed where the run reaches it.** Services rough-in, ductwork,
  sprinkler pipework and the pressure-test hold point now all precede the
  ceilings. There is no `link_skipped` note, and "Alter and test
  sprinklers" follows the ceilings. Fire stopping is not seen live: no run
  reads `fire-passive.penetrations` (a known gap), so the fire-stopping
  order rests on the unit test.
- **F45.** T03 no longer gets wet-area waterproofing, tiling, cold or hot
  water from "bathrooms ... are not in scope", nor demolition from "No
  strip-out or demolition is required". `hydraulic.sanitary` stays in
  scope, correctly: the kitchenette waste is stated. The "fire stairs"
  suggestion of stairs and ramps is still there (not a denial, so out of
  this fix).
- **F46.** `proj.client` is "Corvina Property Holdings Pty Ltd" in all
  three T06 runs. T06 RFP new failures fall from 5 to 0.
- **F47.** Every program is dated: T04 "All 52 durations known" (was 52 of
  53, no dates). The DA variants ST03-da-pathway and ST06-tank-needs-consent
  lose their last unknown row as well. All 175 previously unknown links are
  now scored.
- **F48.** `proj.lga` is North Sydney in ST03-base and strip-out (was
  blank).
- **F49.** FS-101 Rev A and B file as fire services in all three T06 runs.
  The options report (d12), blank before, now files too. T06 discipline is
  all correct (was 3 wrong or missed per run).
- **F50.** ST06-pump-tank-only has no Interior Design package, fee line or
  RFP. Its procurement false positives fall from 5 to 4.

| Scenario | Profile correct (missed, conflict) | Cost plan matched | Program matched | Links violated / unknown | RFP new failures | Fees (total, firm) |
|---|---|---|---|---|---|---|
| ST03-base | 44 (6, 0) → 45 (5, 0) | 21 → 22 / 38 | 29 → 39 / 71 | 0 / 0 → 1 / 0 | 0 → 0 | 6/6, 6/6 both |
| ST03-da-pathway | 45 (5, 0) → 45 (5, 0) | 24 → 24 / 39 | 33 → 40 / 73 | 0 / 12 → 1 / 0 | 0 → 0 | 7/7, 7/7 both |
| ST03-strip-out | 44 (6, 0) → 45 (5, 0) | 22 → 23 / 39 | 25 → 34 / 74 | 0 / 0 → 0 / 0 | 0 → 0 | 6/6, 6/6 both |
| ST04-base | 65 (5, 1) → same | 33 → 33 / 49 | 51 → 51 / 95 | 0 / 52 → 3 / 0 | 0 → 0 | 6/6, 6/6 both |
| ST04-cdc | 65 (5, 1) → same | 32 → 32 / 49 | 49 → 49 / 91 | 0 / 47 → 4 / 0 | 0 → 0 | 5/5, 5/5 both |
| ST04-flood | 64 (5, 2) → same | 33 → 33 / 51 | 51 → 51 / 97 | 0 / 52 → 3 / 0 | 0 → 0 | 6/6, 6/6 both |
| ST06-base | 44 (2, 3) → 45 (2, 2) | 16 → 19 / 36 | 24 → 30 / 86 | 1 / 0 → 1 / 0 | 5 → 0 | 7/7, 7/7 both |
| ST06-pump-tank-only | 45 (2, 2) → 46 (2, 1) | 13 → 14 / 28 | 22 → 22 / 62 | 0 / 0 → 0 / 0 | 5 → 0 | 6/6, 6/6 both |
| ST06-tank-consent | 44 (2, 3) → 44 (2, 3) | 18 → 22 / 38 | 26 → 32 / 87 | 0 / 12 → 1 / 0 | 5 → 0 | 8/8, 8/8 both |

Notes on the table:
- Firm names, invoice amounts and invoice flags are unchanged and all
  correct. The only field outcomes that changed are `proj.lga` (T03),
  `proj.client` and the d12, d16 and d17 disciplines (T06), and the
  scorer case below.
- T04 conflicts are the planned site-area conflict, plus
  `environmental_sensitivity` in the flood variant. T06 conflicts are
  occupation and `flag.staged`.
- ST06-tank-consent's third "conflict" is `hdr.work_type` `conflict_false`.
  This is a scorer limit, not an app defect: the app shows refurb (amber,
  correct) with two losing alternatives, extend (the SEE) and remediation
  (the Calder fee proposal). `score_profile` (`harness/score.py:180`) scores
  any cell with more than one alternative as a conflict and checks only
  the alternatives, never the value. Before the re-run there was one
  alternative, so it scored correct.
- Regression check: none in any scored area. T03 base and da-pathway lose
  their "Tenancy signage" cost line, but that match came from the "01 Site
  works" line, which only F45's false demolition created. `site.signage`
  is not in scope in either run.

**Every violated link explained.**
- *ST03 base and da-pathway: "Fit off electrical and hydraulic services"
  after "Install joinery".* App defect (F52). Neither fit-off row links to
  the joinery.
- *T04 (all three): "Install erosion and sediment controls" after
  "Establish the site", and "Bulk earthworks" after the erosion controls.*
  App defect (F53, F29 family). All three rows start on construction day
  0, in parallel.
- *ST04-base and flood: "Construction certificate: early works" after
  "Coordinate consultant documentation".* Known gap, already in the T04
  shape. The app's single CC sits in approvals (day 160), before D&C
  procurement and the contractor's documentation (days 235-255).
- *ST04-cdc: "Prepare construction documentation" and "Coordinate
  consultant documentation" after "Novate the design consultants".* Known
  gap. D&C plus CDC falls back to `sk.generic` (named in the variant), which
  puts construction documentation (days 75-95) before procurement and
  novation (185-200). The variant inherits the base's D&C "after novation"
  links. Owner question: on a D&C CDC, is the CDC documentation done
  before tender, so that these links should be variant removals?
- *T06 base and tank-consent: "Coordinate consultant documentation" after
  "Prepare fit-out documentation".* Scorer limit, already known: the app's
  link is start-to-start, and the scorer reads `after` as finish-to-start.
- *T06 base and tank-consent: the sprinkler pressure-test hold point after
  "Alter and test sprinklers".* Shape error, now corrected. Since F44 the
  app makes the pipework row, then the hold point, the ceilings and the
  fit-off row, which is right. The T06 shape still had one row, labelled
  like the app's fit-off row, so the scorer paired them. T06 now has the
  two rows that T03 has (`types/T06-sprinkler-pump-tank-upgrade/DOMAIN.md`,
  last section). That violation is gone after re-scoring.

**Still open.**
- F51, F52 and F53 (new, below).
- ST03 strip-out variant: walls, joinery and floors read `remove` (amber),
  so only the strip-out is seen. No new partitions, joinery or floor rows
  are made, though the documents strip and then renew them.
- `site.demolition`'s action is blank in the strip-out variant. It stays
  in scope, but takes the refurb default `alter`, so `fr.strip-out-fitout`
  (which needs `remove`) never fires. No run has strip-out rows (F51
  knock-on).
- T06 still reads no partitions, joinery, doors or painting for level 3.
  The concept report never lists them, so this is a reading or corpus
  question, not D-27. T06 base keeps 20 built-in program misses.
- T06 has no fire-water, pump or tank construction rows (a known gap). The
  commercial fit-out split gives Active fire protection only $128,571 of
  T06 base's $2.7M (a known gap).
- Repeats: F16, F18 (doors as Investigation rows), F29, the F41 family,
  the missed "no" flags, `proj.regulated_design`, and the "fire stairs"
  stairs-and-ramps suggestion.
- Scorer limits: `after` is read as finish-to-start, and a cell with two or
  more alternatives is scored as a conflict even when its value is
  settled.

## App defects

**F1. Update profile response cut off while the app is busy.** *(2 Nov: improved by an owner fix, still 3 of 12 waves, only the first Update on a new project; see `runs/20261102-fixes-1-4-6`.)* In 8 of 12
waves, `POST /projects/{id}/profile/read` returned 200 and then the
connection was reset partway through the body (about 217 KB). The read was
still queued, because that happens before the response is written. Idle,
the same call succeeds every time. No panic or error appears in the server
log. Suspected: the profile view is assembled while a profile event
rewrites it, and something fails after the headers are sent
(`internal/httpapi/profile.go` `writeProfileQueued`; `writeJSON` drops the
encode error). A user would see the Update button fail intermittently.

**F2. Council area read with a trailing word.** `proj.lga` came back as a
conflict between "City of Ryde Section" and "City of Ryde Duty" (base,
trees), and as the single wrong value "City of Ryde Section" (CDC). The
planning certificate says "City of Ryde" and then a section heading or
"Duty". Suspected: the LGA reading takes the words after the name.
`knowledge/profile/project_details.yaml` (lga), `internal/profile/details.go`.

**F3. GST-inclusive fee proposal kept inc GST.** The civil fee proposal
(d07) states $9,900 inc GST. The app's stated total is $9,900; the key is
$9,000 ex GST, with `gst_inc_normalised` expected. All three scenarios.
`internal/proposals/read.go`.

**F4. Procurement route not read.** *(2 Nov: cause found; still open.)*
"Architect-led traditional lump-sum tender to three or four residential
builders" in the owner's brief leaves `hdr.cond.procurement_route` blank.
Knock-on: the program uses `sk.generic` instead of `sk.new-traditional-da`,
in all three scenarios, which accounts for much of the program's missed and
out-of-order activity.

*Cause.* The route question is never asked for that passage. Header
questions (class, type, work type, every `hdr.cond.*`) go only to passages
that `HeaderRouted` accepts (`internal/profile/questions.go`): ordinal ≤ 12,
a section matching `headerSection` (description, outline, scope,
introduction, project details, contract…), or a "the project comprises…"
sentence. The brief's sentence is passage 15 under "## How we want to
build", so it fails all three. Checked with `cmd/profile-eval -live`: the
same text under "Contract and procurement" gets the route question, and
Jev answers `traditional` with confidence 1.0. Under its real heading, the
question is not in the request.

*Suggested fix (owner's call, lane A).* Route the procurement-route
question on its own cue words, not only on position: a passage matching
words like tender, lump sum, design and construct, D&C, ECI, managing
contractor or novat(ed/ion) gets `hdr.cond.procurement_route`, wherever it
sits (the intent-routing pattern the code already cites). Adding
"procurement|tender" to `headerSection` alone would not catch this brief,
whose heading has neither word.

**F5. Invoice with an arithmetic error flagged as a GST-rate error.** d15
states ex $3,800, GST $380 and a total of $4,280 (it should be $4,180).
The app read ex GST as $3,900 (total minus GST) and raised `gst_rate`, not
`arithmetic`. In the trees run the same invoice raised `amount_unknown`
and left the amounts blank. Run-to-run variation on the same bytes.
`internal/invoices/ledger.go` `Check`, `internal/invoices/read.go`.

**F6. The BASIX assessor is not recognised as a discipline.** *(2 Nov: fixed; verified in all three scenarios.)* The
BASIX/NatHERS fee proposal (d09) gets no discipline (Jev 0.59, blank). So
no BASIX package is created from it, and it misses both procurement and
the cost plan. The knowledge has `consultant.basix`, but the filing
discipline question does not pick it.

**F7. The planning certificate is filed as a town planner's document.** d02,
a council s10.7 certificate, gets discipline `consultant.town_planner`
(amber 0.73). The key is `none`: council is not a consultant.

**F8. The supplier's name is never read from an invoice.** All nine
invoices (three per scenario) have a blank `supplier`: Jev answers none at
confidence 1.0, though each invoice names the firm in its heading and
beside its ABN. So no invoice links to its package, even where a package
for that firm exists from its fee proposal. `internal/invoices/read.go`
(supplier question). Found 2 Nov after the scorer stopped counting state
flags as false alarms.

**F9. The certifier RFP has no CDC pathway.** On the CDC variant, every
principal-certifier stage (B stages) is excluded and the RFP blocks on
`pathway_unsupported`: `knowledge/rfp/bca-certification.yaml` lists cdc in
`unsupported_pathways`, and the B stages are gated to da and ssd. A CDC
house needs exactly this appointment. (RFP triage A5.)

**F10. The certifier RFP leaves out the Class 1 and 10 critical-stage
inspections and the BASIX check at the OC** (all scenarios; owner ruling
R7). Related: a certifier package still opens as BCA advisory (a1 to a3),
because `bca_certification` is a client scope decision. Whether a house
should default to certification is an owner call. (RFP triage A1.)

**F11. `flag.mixed_use` is yes for a house with a garage.** Class 1a plus
10a counts as "parts with different classes" (`internal/rfp/drivers.go`
around line 304), so the architect RFP of an ordinary house asks for a
"Class and use map by part". Class 10 alongside one other class should
not make a project mixed use. (RFP triage A3.)

**F12. The structural RFP has no row for footings that avoid a tree's
structural root zone** (trees variant, owner ruling R9). (RFP triage A4.)

### From run `runs/20261104-t05-t09` (T05 and T09; app at `c56c9de`), 4 November 2026

Repeats on the new types: F1 (Update cut off in the first wave), F2,
F3, F4 (route blank in both T05 runs, so the program opens on
`sk.generic`), F7 (the s10.7 certificate is filed as town planner in T05
and T09) and F8 (no invoice supplier in all six invoices).

F2 is wider than the council area. The site address has the same
trailing-word problem: "41 Tallowwood Avenue Date", "41 Tallowwood Avenue.
It" and "41 Tallowwood Avenue. We" sit beside the right address, so
`site.address` becomes a conflict (T05 base and first floor).

**F13. A wrong ABN on an invoice can never be flagged.** The T09 structural
invoice prints `ABN | 77 650 369 250`, which fails the checksum. The app
stored no ABN and raised no `abn_invalid`. Cause (traced):
`internal/invoices/harvest.go` puts a checksum-failing ABN into
`ev.BadABNs` and skips it as a candidate; nothing reads `BadABNs`. The
issuer ABN is only ever Jev's pick among valid candidates, so the
`abn_invalid` check in `ledger.go` (`inv.ABN != "" && !ValidABN`) cannot
fire on a read invoice, only on a hand-typed one. Suggested fix (lane A):
when the only ABN printed beside "ABN" fails the checksum, store it and
let the ledger flag it. The same invoice's date ("| Date | 05/05/2026 |")
was also left blank, Jev confidence 1.0.

**F14. Stated fee total missed in two common layouts** (all three T09
proposals; traced offline with `Harvest` and `NewDraft` on the corpus).
- A total row whose words sit in the second column, with the first cell
  empty (`| | Total $17,000 excluding GST | $17,000 |`, under a
  Stage/Item column), is not marked `Total`. With no total candidate, the
  stated-total question is never asked, so the field is blank, not
  merely unconfirmed. T05's `| | **Total** | **$6,800** |` worked; the
  difference is the amount inside the label cell.
- A total stated only inc GST ("Total $99,000 including GST") is dropped
  by `settleTotal` (`a.GST != "inc"`), and nothing converts it to ex GST,
  so it is blank too. This is the other side of F3 (where an inc figure
  was kept as if ex).
`internal/proposals/read.go` `settleTotal` and the stated-total question
guard (`len(totals) > 0`). The T09 hazmat proposal (`| | Total $7,000
excluding GST | $7,000 |`) is a fourth case; its firm was also left blank
although the letterhead prints "Fibrecheck Hazardous Materials Pty Ltd"
and a valid ABN (not traced).

**F15. One dissenting passage blanks a value, however lopsided the
evidence.** `hdr.work_type` had 42 passages for `extend` and 5 for `refurb`
(the internal alterations) and was left blank (red), in T05 base and first
floor. Same pattern in T09: `fact.contract_basis` 13 `lump_sum` to 3
`schedule_of_rates` (the planned "schedule of rates" trap), and
`det.existing_building` 2 to 1 (the 1 is "The existing building stormwater
system … is not altered", read as "not an existing building"). Worst case, on the very value a
variant tests: in T05 heritage, `det.heritage_status` had 14 passages for
`conservation_area` (the s10.7 certificate, SEE, HIS, design statement,
three fee proposals) and 1 for `none` ("… 1958 house. None of it has
heritage significance", about the rear lean-to), and was left blank.
Knock-on in
T05: with work type blank and the route blank (F4), the program opens on
`sk.generic`, not `sk.extend-traditional`. Whether a strong majority
should settle a choice, or alterations plus additions should count as
`extend`, is an owner call (lane A: profile trust).

**F16. Rise in storeys is read only from a BCA or NCC report.** The T09
defect report says "The building has a rise in storeys of 6"; the field
(`stated_in: BCA or NCC compliance report`) stayed blank with no
sources. Remediation jobs rarely have a BCA report, so the class 2 DBP
and HBCF rules that hang off storeys get nothing.

Minor: `proj.client` read as "The Owners - Strata Plan No. 912345, c/-
Rhiannon Teale, Northgate Strata Management" (the care-of party is kept).
The predicted cut at " - " did not happen.

### From run `runs/20261106-t10-t12` (T10, T11, T12; app at `0ee5b22`), 6 November 2026

Run report: `runs/20261106-t10-t12/REPORT.md`.

**Repeats.**
- F6: the T10 BASIX fee proposal files blank in all three runs (Jev 0.31
  to 0.35). It was borderline in the pilots; here it fails every time.
- The T01 open items are still open: missed "no" flags (groundwater,
  multi_building and excavation_adjoining in T12), unread
  `proj.regulated_design`, and `flag.community` read from "neighbour
  notification" (T11).

A theme runs through F17 to F19: on a knock-down, the documents describe
two buildings, and some readings take the house being demolished as the
project.

**F17. `det.existing_building` is false or blank on knock-down and dual
occupancy jobs.**
- *Expected:* true. The shapes key demolition as work to an existing
  building (T10 REVIEW §2.4).
- *Actual:* `stated_false` in all three T10 runs and in T11 base and
  three-plus. In T10 it came from "This is a new house: we will knock down
  the existing house…" (0.91). In T11 it came from "The site contains an
  existing building … to be demolished" (0.61). T11 CDC was blank (0.56).
- *Cause:* the question's criteria say nothing about demolition, so Jev
  reads "new house" or "to be demolished" as a denial.
- *Location:* `knowledge/determinants.yaml` `existing_building`.
- *Knock-on:* the RFP `existing_building` module is off, which drops the
  architect's `existing_interfaces` and the civil `retained_services` (12
  RFP new failures).
- *Owner call:* does demolition count? If yes, the question needs a
  criterion. If no, the T10 and T11 keys change.

**F18. Consultant scope and old-house passages set work actions to
`investigate`.**
- *Expected:* `new` for the new building's systems.
- *Actual (ST10-base, `work_items`):* windows, wall linings, ceilings,
  floor finishes, footings and lighting are `investigate`. They came from
  "lead paint on the window frames…" and "the old brick strip footings …
  are to be excavated". In ST12-base, footings, slab and timber framing are
  `investigate`, from "Inspections of the footings and the slab
  reinforcement before the pour" and the structural fee scope.
- *Cause (traced to the data, not the code):* a document action replaces
  the work-type default even when it describes a consultant's service or
  the demolished house.
- *Location:* the system action reading in `internal/profile` (action
  shape) and the work items it writes.
- *Knock-on:* the 1a construction fragments need `new`, so most program
  built-in misses follow. Missed rows include excavate and pour footings,
  under-slab drainage, slab, wall frames, trusses, windows, plasterboard
  and ceilings; generic rows such as "Pad and strip footings" appear
  instead. Every RFP also gets a seven-item "Investigations" list.

**F19. The T11 subclass is a conflict, and the residential fallback is
the apartment scope list.**
- *Expected:* `hdr.subclass` townhouses, and only house-scale systems in
  scope.
- *Actual:* "house" has 10 sources against 11 for "townhouses", in all
  three T11 runs. The "house" sources are hazmat, survey and geotechnical
  passages about the existing house. With the subclass blank, the scope
  defaults use `categories: residential`, which is the Class 2 list
  (`*id001`): passenger lifts, lift shafts, hydrants, accessible access
  and livable housing. The package scope additions then add Vertical
  Transport, Fire Services and Access consultants, and the harness made
  RFPs for all three in every T11 run, not only CDC.
  `hdr.scale.avg_dwelling_size_sqm` has no row at all.
- *Location:* `knowledge/profile/scope_defaults.yaml` `categories`;
  `knowledge/works/package_defaults.yaml` scope additions.

**F20. "DA fees and contributions" is kept on every CDC job (traced).**
- *Expected:* the `da-fees` starter is dropped when `hdr.cond.planning` is
  cdc.
- *Actual:* it is kept with "Applicability is unknown" in all six CDC runs.
- *Cause:* `costSuggestInput` reads profile values with `bool_and(value_state='known')`.
  Profile rows store `set` (7,447 rows) or `absent`, never `known`. So every
  field-gated starter sees an empty value and is kept as unknown.
- *Location:* `internal/store/cost_suggest.go` `costSuggestInput`.
- *Note:* `cost_budget.go` uses `value_state='set'` correctly. The shapes
  assumed the starter is dropped.

**F21. A suburb whose second word is a direction is cut (traced).**
- *Expected:* "22 Marrakai Close, Caringbah South NSW 2229".
- *Actual:* "22 Marrakai Close, Caringbah", in all T10 runs, although every
  source prints the full address.
- *Cause:* `addressCandidates` keeps a second suburb word only after a
  `suburbFirst` word ("South Hurstville"). In "Caringbah South", the first
  word is not in that list, so the address ends at "Caringbah", and the
  state and postcode go with it.
- *Location:* `internal/profile/details.go` `addressCandidates`.

**F22. The client is blank when a proposal is addressed "To: agent, for
client" (traced).**
- *Expected:* "Hamish and Rosa Delaney".
- *Actual:* a conflict. Six sources read the right name. Three read the
  whole line, "Ines Hart, Marlowe Hart Architects, for (or on behalf of)
  Hamish and Rosa Delaney", from the T10 geotechnical, hazmat and
  structural proposals.
- *Cause:* `addresseeCandidates` offers the whole "To:" line. It cuts only
  at c/- and "as trustee for", not at "for" or "on behalf of".
- *Location:* `internal/profile/addressee.go`.
- *Knock-on:* all five T10 RFPs block on `details` (Client).

**F23. The letterhead's descriptor line is read as the firm name
(traced).**
- *Expected:* "Drummond Civil Pty Ltd".
- *Actual:* "Civil and stormwater engineers", in all T10 runs.
- *Cause:* `letterhead()` walks up from the ABN line and takes the first
  line that matches `companyRe`. "engineers" matches, and that line sits
  between "DRUMMOND CIVIL PTY LTD" and the address. Checked offline with
  `Harvest`: the letterhead name is the descriptor, and `HeaderFirm` is
  right.
- *Location:* `internal/proposals/letterhead.go` `letterhead`.

**F24. The fee total is lost when a proposal quotes the cost of work
(traced offline).**
- *Expected:* a stated total of $4,700 for the T12 certifier (d10).
- *Actual:* blank in all three T12 runs, with GST basis "unknown".
- *Cause:* the proposal's notes print "$250,000 or more including GST" and
  "$231,000 including GST", for the long service levy threshold and the
  owner's cost of work. `Harvest` keeps both as fees:
  - `settleTotal` refuses the total because a larger "fee" is on the page;
  - the GST basis is split, three ex against two inc;
  - both figures become priced quote rows ($227,272.73 and $210,000).
  The stated total and the GST basis then go to Jev, and the run stored
  neither.
- *Location:* `internal/proposals/harvest.go` (amount kinds),
  `internal/proposals/read.go` `settleTotal`.

**F25. "Building designer" is not a discipline alias.**
- *Expected:* `consultant.architect`. T12's designer is a building
  designer, as on most granny flats.
- *Actual:* the fee proposal, design statement and site plans file with
  no discipline, at Jev 0.35 to 0.52. That is four of five design
  documents in each T12 run, and one site plan filed as `none`.
- *Location:* `data/intake/disciplines.json` `consultant.architect`
  aliases.
- *Knock-on:* in T12 base there is no architect package from the fee
  proposal, so the designer's invoice gets no supplier.
- *Related:* the T12 stormwater fee proposal (d07, "Garrick Stormwater")
  filed as hydraulic, not civil, at 0.66 to 0.70. The T12 DA SEE filed as
  `consultant.building`.

**F26. A house connected to the street water main is read as building
next to an authority asset.**
- *Expected:* `flag.authority_asset` no.
- *Actual:* yes in ST11-cdc. It is a conflict in ST10-da, ST10-sep and all
  three T12 runs, and blank in T11 base and three-plus.
- *Cause:* the trigger `(?:sewer|water) (?:main|asset)` matches "Water main
  in Wattlebird Street; the house is connected to it". Jev answers yes at
  0.60 to 0.82. The denial in the next bullet ("No Sydney Water assets are
  within the property") scored 0.37 to 0.43 when it lost.
- *Location:* `knowledge/profile/flag_readings.yaml` `authority_asset`. This
  was listed in pilots as "conflict, check wording"; the wording is now
  traced.
- *Knock-on:* the architect `water_asset` and the structural
  `authority_asset` stage are included in eight runs (16 RFP new failures).

**F27. The F15 majority rule counts passages, not documents.**
- *Expected:* `hdr.cond.access_constraints` unrestricted. The survey says
  so at 0.99.
- *Actual:* `urban_constrained`, in all T10 runs. It rests on five passages
  from two geotechnical documents, about a small track-mounted rig and the
  site classification (0.37 to 0.91).
- *Location:* `internal/profile/reconcile.go` `lopsidedChoice`, which counts
  `len(byValue[v])`. This is a side effect of the F15 fix.
- *Knock-on:* the surveyor RFP's `access` position.

**F28. The reinforcement hold point comes after the pour (traced).**
- *Expected:* the inspection before "Reinforce and pour slab".
- *Actual:* "Reinforce and pour slab" → "Hold point: Reinforcement
  inspection release identifying the pour" → "Cure slab". Seen in five
  runs, and once in the T01 pilot.
- *Cause:* `holdPointRows` puts a hold after the last activity tagged with
  the interface's `after` system (`structure.concrete`). The one
  fr.slab-on-ground-1a activity is tagged with both `structure.concrete`
  and `substructure.slab-on-ground`, so the hold lands after the pour.
- *Location:* `internal/program/generate.go` `holdPointRows`;
  `knowledge/clusters/structure/interfaces.yaml` hold point.

**F29. The house construction fragments float apart (suspected: missing
attach links).** Seen in all nine runs; T05 in the pilots showed the same.
- "Establish the site" has no predecessor or successor, and starts the same
  day as erosion control.
- The fit-offs (sanitary, water, electrical) link only to their rough-ins.
  They finish before plasterboard and joinery start.
- Hydraulic rough-in follows "Cure slab", before the frame and roof.
- Site stormwater is not after gutters and downpipes.
- Wrap and cladding are not after the windows.

Location: `knowledge/program/fragments.yaml` (fr.site-establishment-1a,
fr.hydraulic-*-1a, fr.electrical-1a, fr.stormwater-1a, fr.cladding).

**F30. A dangerous-goods process is read on houses.**
- *Expected:* no `hdr.cond.hazardous_process` row. The taxonomy limits it
  to industrial.
- *Actual:* "Dangerous goods", from the demolition and asbestos passages.
  It is printed in seven RFP briefs across T10, T11 and T12, and is hidden
  in the profile view.
- *Location:* `knowledge/profile/taxonomy.yaml` `hazardous_process`
  `applies_to`, which the header question routing appears to ignore
  (suspected).

**Harness: two RFP drafts returned 409** (Hazmat in ST10-da, BASIX in
ST12-sewer).
- Both happened while three projects in the same organisation were
  drafting.
- No reading jobs were queued, and the ensure call uses the last completed
  profile, so this is not `reading_incomplete`.
- Suspected: `refreshReport` gave up after three serialisation failures
  and returned `ErrVersionConflict`.
- Those reports have no draft. Run scenarios one at a time, or log the 409
  body.

### From run `runs/20261106-t02-t13-t17` (T02, T13, T17; app at `6284fa1` plus another session's uncommitted Profile display edits), 6 November 2026

The run report is `runs/20261106-t02-t13-t17/REPORT.md`.

**Repeats.**
- `proj.regulated_design` is still unread. The civil RFP blocks on it in
  every T02 and T13 run. The structural RFP also blocks in
  ST02-traditional and in every T17 run.
- Missed "no" flags (the T01 open item). `flag.architect_lodges` is blank
  in T13, although the architect says the town planner lodges.
- F5 is back, for an ex-GST line labelled "Fee" (F31).
- F23's letterhead fix is defeated by a directory name match (F37).
- F24's guard loses another total (F38).
- F6 and F25 family: discipline is blank for planning documents (F40).
- F15 and F27 family: neither free text nor flags settle on a majority
  (F35, F36).

**F31. The F5 fix misses an ex-GST line labelled "Fee" (traced).**
- *Expected:* the T17 fire engineer's invoice, K-26-0412, is $14,500 ex GST
  plus $1,450 GST. It prints a total of $16,850, which should be $15,950.
  The ledger should flag `arithmetic`.
- *Actual:* ex GST is $15,400, `gst_rate` is raised and `arithmetic` is
  missed. This happened in all three T17 runs.
- *Cause:* `settlePrinted` keeps printed amounts only when the ex-GST label
  is "subtotal", "ex GST", "excl" or "before GST". That test is `exish`, at
  `internal/invoices/read.go:262-326`. The invoice's ex-GST line says "Fee",
  so this route fails.
  - Jev then picks the total and the GST.
  - `Resolve` sets ex GST to total less GST (`read.go:755`).
  - The ledger sees ex + GST = total, so it raises no `arithmetic`. It sees
    $1,450 ≠ 10 % of $15,400, so it raises `gst_rate`
    (`internal/invoices/ledger.go:305-313`).
- *Contrast:* T13's arithmetic invoice says "Sub-total ex GST" and was
  right.
- *Location:* `internal/invoices/read.go` `exish` and `printedAmounts`.

**F32. Most subclass scale fields are never read (traced).**
- *Expected:* T13 `residential_units` 42, `retail_gla_sqm` 600,
  `podium_levels` 1, `tower_levels` 5 and `site_sqm` 1,800. T02
  `avg_unit_size_sqm` 80.
- *Actual:* all blank, with no sources, in every run.
- *Cause:* `HarvestOwn` asks a scale question only when the field's trigger
  matches (`internal/profile/harvest.go:121-125`). These fields have no
  `triggers:` (`knowledge/profile/taxonomy.yaml:131-135`, `1426-1440` and
  the other mixed subclasses). Of the 147 distinct scale fields, 127 have
  none. `scaleCandidates` has an extractor for `residential_units`
  (`harvest.go:214`), but it is never reached.
- *The same facts are read under generic keys:* units 42 (18 in the
  three-storey variant), `gla_sqm` 600, `total_storeys` 6.
- *`site_sqm`, separately:*
  - The T13 brief and architect say "consolidated site … about 1,800 m2",
    which misses the triggers "site area" and "lot size"
    (`taxonomy.yaml:40-42`).
  - The survey's "Site area by survey:" ends a line with a colon, so the
    figure is cut into the next passage with the label as context
    (`internal/jobs/source.go:207`).
  - No survey produced a site-area reading, in T13 or in T02 (where label
    and figure share a line). Not traced further.
- *Location:* `knowledge/profile/taxonomy.yaml` `scale_fields`.
- *Owner call:* add triggers or aliases, or key these as unread.

**F33. A trigger phrase broken by a line wrap is missed (traced).**
- *Expected:* `flag.cl46` no. The T13 SEE says "…no clause⏎4.6 variation
  request is required."
- *Actual:* blank, in all three T13 runs.
- *Cause:* the trigger `clause 4\.6` has a literal space, and its literal
  prefilter checks the raw passage text, which keeps the newline
  (`internal/knowledge/profile.go:349-398`; `knowledge/profile/flag_readings.yaml:71-73`).
- *Scope:* any multi-word trigger can miss on text extracted from a PDF.
- *Knock-on:* the town planner's `variation_request` stage is priced as an
  option, which is 1 RFP new failure per run.
- *Location:* trigger matching in `internal/knowledge/profile.go`.
  Normalising whitespace before matching would cover every trigger.

**F34. "Road traffic noise … is low" is read as a noise-affected site
(traced).**
- *Expected:* `det.noise_affected_site` false. This is the planned d13 trap.
- *Actual:* `stated_true` in all three T02 runs, from the acoustic report,
  at 0.73 to 0.82.
- *Cause:* the trigger matches "traffic noise", and the criteria do not say
  that a low or compliant level is a denial
  (`knowledge/determinants.yaml:809-830`).
- *Knock-on:* small. One unforeseen record (T02 shape).

**F35. Two spellings of the owners corporation make the client a conflict
(traced).**
- *Expected:* "The Owners - Strata Plan No. 87654".
- *Actual:* a red conflict, in all three T17 runs. Eleven sources say "The
  Owners - Strata Plan No. 87654". One, the audit's "Owner: Strata Plan
  87654, managed by…", says "Strata Plan 87654".
- *Cause:* `mergeExtendedText` merges only when one value is the first
  words of the other (`internal/profile/reconcile.go:541-578`).
  `lopsidedChoice` excludes free-text details (`reconcile.go:498-519`).
- *Knock-on:* every T17 RFP that needs the client blocks on `details`:
  architect, fire engineer, structural, town planner and QS. That is 5 of
  the 9 RFP new failures in base, and 4 of 9 in CDC.
- *Owner call:* see F27. Should a strong majority settle free text?

**F36. An existing basement is read as basement works (traced).**
- *Expected:* `flag.basement` no. This is the T17 d04 trap; the works are
  external walls only.
- *Actual:* a conflict between no (2 sources) and yes (1 source, "The
  building is wholly residential above the basement car park", 0.98).
- *Cause:* the question asks whether the project "includes a basement"
  (`knowledge/profile/flag_readings.yaml:26-28`), and an existing building
  does. Flags have no majority rule.
- *Knock-on:* structural `dilapidation`, `retention_design` and
  `temporary_works_interfaces` fall back to option, which shows as
  included. That is 3 RFP new failures per T17 run.

**F37. A directory firm matched by name overwrites the ABN printed on the
proposal (traced).**
- *Expected:* the T02 civil proposal is from Northfall Civil Engineering,
  ABN 36 979 312 905.
- *Actual:* it is attached to the T10 firm saved as "Civil and stormwater
  engineers" (Drummond's descriptor, saved before F23 was fixed). That firm
  brings ABN 29 559 935 012, a Taren Point address and Kate Drummond as
  contact.
- *Same defect elsewhere:* the T02 and T13 Basalt Ridge proposals print ABNs
  49 518 838 449 and 65 534 253 982. Both took the T11 Basalt Ridge firm,
  ABN 27 811 849 176.
- *Cause:*
  - `settleFirm` looks up the directory by every name found on the page,
    including Northfall's descriptor line (`internal/proposals/read.go:276-303`).
  - `FirmLookup` tries the ABN first, then the name key
    (`internal/store/submissions.go:314-345`).
  - A name match then fills every firm field by rule, though the page
    prints a different valid ABN.
- *Location:* `settleFirm` and `FirmLookup`. A name match should be refused
  when the page's only valid ABN differs from the directory firm's (lane A).
- *Harness:* the directory is organisation-wide, so this depends on run
  order (see "Harness" in the report).

**F38. A larger optional fee blocks the stated total (traced).**
- *Expected:* $156,000 for the T02 structural proposal (d06). It prints a
  `| | **Total** | **$156,000** |` row and "Total $156,000 + GST".
- *Actual:* blank, in base and groundwater.
- *Cause:* the page also offers "Option: post-novation services … $210,000
  ex GST". `settleTotal` refuses a total when any larger fee is on the page
  (`internal/proposals/read.go:486-490`).
- *Contrast:* the traditional variant has no larger option and was right.
- *Related:* the same guard as F24 (there, the cost of work).
- *Location:* `settleTotal`. Optional or conditional fees should not count
  as larger fees.

**F39. A quantity surveyor's "cost estimate" service is read as a total
(traced).**
- *Expected:* $45,000 ex GST for the T17 QS proposal (d11). It prints
  "Total fee $49,500 including GST" in a plain-text table.
- *Actual:* no stated total, in all three T17 runs.
- *Cause:* "DA cost estimate report (registered quantity surveyor) $3,300"
  matches `totalWords` `(?:cost|fee) estimate`
  (`internal/proposals/harvest.go:246, 390`). That gives two distinct
  totals, so `settleTotal` sets none.
- *Second effect:* the listed appointment sum is $42,000, because the
  $3,000 row is treated as a total. The correct sum is $45,000.
- *Location:* `internal/proposals/harvest.go` `totalWords`.

**F40. Planning documents file with no discipline, and BCA reports file as
certifier.**
- *Expected:* `consultant.town_planner` and `consultant.bca`.
- *Actual:*
  - planning documents with no discipline: the T02 Hollis fee proposal
    (0.50), the T13 SEE (0.53 to 0.54), and the T17 Ridgeway fee proposal
    (0.56) and SEE (0.53);
  - the BCA assessment report filed as `consultant.certifier` in T02 and
    T13 (0.92 to 0.97).
- *Knock-on:* the T02 town planner's invoice gets no supplier, because no
  firm is linked to the town planner package.
- *Location:* `data/intake/disciplines.json` (same family as F6 and F25).
  Not traced further.

**F41. The contamination package rule can never fire (traced; already
noted in the knowledge index).**
- *Expected:* an environmental consultant suggested when land is
  potentially contaminated.
- *Actual:* in ST13-contaminated-land the environmental package came only
  from the d19 proposal (`origin: document`, "proposal received"). There is
  no suggestion, although `det.potentially_contaminated_land` is
  `stated_true`.
- *Cause:* the rule keys `field: contamination_level`
  (`knowledge/works/package_defaults.yaml:135-141`). That is neither a
  determinant nor a taxonomy condition, so `packageFieldValue` rejects
  every value (`internal/procurement/defaults.go:160-186`). See
  `reference/index/procurement.md:36`.
- *Location:* key the rule on `potentially_contaminated_land`.

**F42. No reinforcement hold point on a basement building (traced to the
data).**
- *Expected:* a reinforcement inspection before the slab pours (T13 shape,
  built in).
- *Actual:* no reinforcement hold in any T13 run. There is no fr.slab-2
  row; the basement slab and floor-cycle pours have no hold.
- *Cause:* `if.reinforcement-inspected-before-pour` sequences
  `structure.concrete` before `substructure.slab-on-ground`
  (`knowledge/clusters/structure/interfaces.yaml:250-256`). No
  slab-on-ground system is in scope on a building with a basement.
- *Location:* the interface's systems. The T13 shape's built-in claim also
  changes (see below).

**Shape and key gaps (recommended, not applied).**
- *T02 `traditional-procurement`:*
  - add `skeleton: sk.new-traditional-da`;
  - remove "Prepare construction documentation before tender" and
    "Construction certificate before award" from `adds.program`. They are
    base rows the app makes ("Prepare construction documentation",
    "Construction certificate"), reworded past the 0.75 match.
- *T02 RFPs:* add to architect `app_differs`
  `{key: stage.brief_masterplan, app: include, expect: absent}`, and to
  certifier `app_differs`
  `{key: stage.a4_construction, app: option, expect: absent}`.
- *T13 architect:* `blocking: ["fact:budget"]` becomes `[]`. The budget is
  stated, as in T10 to T12.
- *T13 `app_differs`* (`expect: absent`, or `exclude` for stages):
  - bca: b2, b3 and b4 stages (option); `existing_assessment` (ask);
  - certifier: a2 and a3 (ask); a4 (option); `existing_assessment` (ask);
  - hydraulic: `treatment` and `process` (ask); `staging` (option);
  - fire services: `plant` and `upgrade` (ask); `remediation` (O);
  - mechanical: stage `investigation` (ask); `staging` and `remediation`
    (O);
  - electrical: stage `investigation` and `dry_fire` (ask);
  - geotechnical: `asset_exposure`, `ass` and `handback` (ask);
  - surveyor: `buried` and `sewer` (ask).
  - That is 24 of the 28 base RFP new failures.
- *T13 program:* the reinforcement-hold and ground-slab rows become
  `built_in: "no"` (F42).
- *T13 keys:*
  - `podium_levels` and `tower_levels`: `read_by_design: false`;
  - `residential_units` and `retail_gla_sqm`: the owner's choice (F32).
- *T17 `cdc-pathway`:*
  - add "Town planner" to `removes.cost_plan`;
  - rename "Prepare and lodge the CDC" to the app's "Prepare and lodge the
    complying development certificate";
  - change `built_in.program` to `"partial"`.
- *ST17 key, d01:* if the owner agrees, the council fire safety order may
  also be `statutory_instrument`.

**Scorer limits.**
- A trigger cannot change `rfps[].expect` (known). This gives 16 false new
  failures in ST02-traditional, `earthworks_suitability` in ST02-groundwater
  and ST13-contaminated, and `planning_response` in ST17-cdc.
- `blocking.slot:project:assessment_quantities` (the design vehicle, typed
  by the user) should be user input in `user_input_check`
  (`tools/score_shapes.py:46-52`). This is 1 per T02 and T13 run.

**Harness.**
- The firm directory is organisation-wide and keeps firms from earlier runs
  (F37).
- Fictional firm names repeat across types: Basalt Ridge Geotechnical is in
  T02, T11 and T13, with three ABNs.

**Owner questions.**
- Is a CDC lawful for an 8-storey Class 2 recladding? The Codes SEPP Fire
  Safety Code lists three categories, and cladding is not one. The
  ST17-cdc premise is doubtful and is kept as an app test.
- Should free text and flags settle on a strong majority (F35, F36, F27)?
- Should a single-building architect RFP always price "Brief and
  masterplan" (`architecture.yaml:62-66`)?

### From run `runs/20261107-t03-t04-t06` (T03, T04, T06; app at `0aee9a4` plus uncommitted edits), 7 November 2026

The run report is `runs/20261107-t03-t04-t06/REPORT.md`.

The uncommitted edits were in `internal/httpapi/profile.go` (the
`scope_decision` display field), `knowledge/profile/layout.yaml`,
`internal/knowledge/layout_test.go` and web display files. None of them
changes a reading, a scope row, a package, a cost line, a program row or an
RFP position. The run used a fresh database.

**Repeats.**
- F16: T06 `det.rise_in_storeys` is blank. "The building has a rise in
  storeys of 6" is printed only in the annual fire safety statement, which
  is not a `stated_in` source.
- F18 on a refurb: the access consultant's review passages set accessible
  access, egress doors and internal doors to `investigate`
  (0.94 on internal doors). The program then shows them as Investigation
  rows, and no door-hanging row is made. The knock-down fix
  (`oldBuildingAction`) applies only to new work.
- F29: "Establish the site" has no predecessor in all nine runs (fit-out
  and 7b fragments too).
- F41 family: `complexity_additions` `flood_exposure`
  (`knowledge/works/package_defaults.yaml:164-170`) keys a field the
  profile does not have, so it can never fire. Here this has no effect:
  civil is a baseline consultant, and the flood reaches the civil RFP
  through `det.flood_hazard`.
- Missed flags are fewer, but some remain:
  - T04 now reads `cl46`, `community`, `staged`, `multi_building`, `panel`,
    `basement` and `osd_existing` as no;
  - still missed: T04 `architect_lodges`, T03 `authority_asset`, and T04
    `fire_specialist_systems` (no at 0.42, under the floor);
  - T04 `stormwater_quality` (yes) was never stored. Its triggers match the
    MUSIC sentences. Suspected: the question asks whether *council
    requires* modelling, and the documents say only that targets are
    demonstrated with MUSIC.
- `proj.regulated_design` is still unread. The keys now score it by design.

**F43. On a refurb, a system with no settled works action is filed as
existing and leaves scope (traced).**
- *Expected:* the works the documents describe are in scope:
  - T03: partitions, ceilings, glazing, joinery, paint, floors, lighting and
    cabling;
  - T06: the fire water and sprinklers (pumps and tank).
- *Actual:* each has `sys.<leaf>.presence` included (amber), a blank or red
  `.action`, and `sys.<leaf>.existing` present, so `scope.<leaf>` is never
  written. Systems filed as existing:
  - T03 base: 18;
  - T03 strip-out: 17, including sprinklers, detection, air conditioning
    and demolition;
  - T06 base: 7, including `fire-active.fire-water` and
    `fire-active.sprinklers`;
  - ST06-pump-tank-only: 2, which leaves one system in scope
    (`hydraulic.backflow-prevention`).
- *Cause:* D-27, `existingPresence` (`internal/profile/existing.go:83-98`),
  treats an included system with no applied action on a refurb as existing.
  `withScope` then skips it (`internal/profile/build.go:190-211`). The
  action is blank in two ways:
  - *Below the floor.* The action floor is 0.85
    (`data/profile/thresholds.json` `shapes.action`). T03's "## Interiors"
    list got `new` 0.59 (walls), 0.73 (joinery), 0.52 (glazing) and 0.43
    (floors), and `alter` 0.74 (ceilings). All were dropped.
  - *Conflict.* T06 fire water got upgrade (0.99, 0.98, 0.92, 0.87), alter
    0.91, retain 0.97 (from "The hydrant system is unaffected") and
    investigate 0.91. Sprinklers got upgrade 0.99 against alter 1.00. Both
    reconcile red with value "". `appliedWorkRow` rejects a red row, so the
    action counts as unstated.
- *Knock-on:*
  - T03 program: the construction stage has no partitions, ceilings, doors,
    joinery, paint, floors, luminaires, cabling or fire stopping. The
    strip-out variant has 7 construction rows and no strip-out.
  - T03 cost plan: no Electrical or Communications lines.
  - T06 program: 3 construction rows and 25 built-in misses per run.
  - T06 cost plan: Mechanical gets $1.8M of the $2.7M.
  - ST06-pump-tank-only: the whole $1.2M goes to "Hydraulic services", and
    there is no "Active fire protection" line.
- *Location:* `existingPresence`, with the action floor and red-action
  handling. Two possible fixes (owner's call, lane A):
  - treat a conflict among works actions (new, alter, upgrade, replace) as
    `several`, which `withScope` already handles;
  - do not apply D-27 to a presence read from a passage that lists the
    works.

**F44. The three T03 REVIEW program-order defects are still in the code
(confirmed offline at `0aee9a4`; masked live by F43).**
- *Method:* `Generate` was run over the real catalogue with the T03 shape's
  33 work items, refurb, cdc and traditional (throwaway `go test -overlay`;
  nothing was written in the repo).
- *Results:*
  - "Seal service penetrations" has no predecessor and starts on
    construction day 0. Its hold point precedes partitions, ceilings, doors,
    joinery "Procure and deliver" and "Install", painting and floors.
  - Ceilings start on day 8. Lighting (day 11) and ductwork (day 14) follow
    the ceilings. Sanitary rough-in follows the doors, and electrical
    rough-in (day 25) follows the hot-water unit.
  - Note `link_skipped` appears: "Link from Hold point: a recorded pressure
    test of the sprinkler pipework to interiors.ceilings left out: it would
    make a loop".
- *Locations:* as in `types/T03-office-fitout/REVIEW.md` findings 1 to 3
  and the T03 shape gaps (lines 768-770):
  - `internal/program/generate.go` `chainLinks`;
  - `knowledge/clusters/fire/interfaces.yaml`
    `if.fire-stopping-before-concealment`;
  - `knowledge/program/fragments.yaml` `fr.fire-stopping`,
    `fr.sprinklers-fitout`, and the services fragments that attach after
    `fr.strip-out-fitout`.
- *Also seen live:* with no anchor, the services chain in library order.
  Sanitary is followed by cold water, hot water, ductwork, commissioning,
  then sprinklers and detection, so the sprinklers start after mechanical
  commissioning.

**F45. `implied_by` scope suggestions fire on denials (traced).**
- *Expected:* nothing suggested from a sentence that excludes the work.
- *Actual:*
  - "The base-building bathrooms on the core are not in scope" (T03 d13)
    suggests wet-area waterproofing, tiling, sanitary, cold and hot water
    and local exhaust. The program gains "Wet-area waterproofing" and "Wall
    and floor tiling".
  - "No strip-out or demolition is required" suggests `site.demolition`.
  - "fire stairs" suggests stairs and ramps. This trap is already named in
    the T06 shape.
- *Cause:* `implied_by` patterns (`knowledge/profile/scope_defaults.yaml`
  `wet-room`, `strip-out`, `stair`) match words with no negation check
  (`withImpliedScope`, `internal/profile/build.go:246-269`).
- *Note:* "kitchenette" does not match `\bkitchens?\b`, so the fit-out's
  real kitchenette implies nothing.

**F46. A "To:" line of the form "Name, Role, Company" is offered whole as
the client (traced; F22 family).**
- *Expected:* "Corvina Property Holdings Pty Ltd".
- *Actual:* a red conflict in all three T06 runs.
  - "Corvina Property Holdings Pty Ltd" has 3 sources ("Client:" lines).
  - "Megan Holt, Asset Manager, Corvina Property Holdings Pty Ltd" has 5
    sources (certifier, interior designer and hazmat "To:" lines).
- *Cause:* `addresseeCandidates` (`internal/profile/addressee.go:30-53`)
  offers the whole line. It cuts only at c/-, "for" or "on behalf of", and
  "as trustee for". It does not cut a leading person and role.
- *Knock-on:* the fire services, structural, electrical, mechanical and
  certifier RFPs block on `details`. That is 5 RFP new failures per run.

**F47. New-build programs have no dates (traced).**
- *Expected:* computed start and finish for every row (owner direction,
  25 October: every activity dated).
- *Actual:* in all three T04 runs every activity's `es` and `ef` is null,
  `finish.typical` is null, and the toolbar reads "Indicative: 52 of 53
  durations known". The same holds in ST02-base ("90 of 91"), ST13-base
  ("94 of 95") and ST10-base ("73 of 74"): zero dated activities in each.
- *Cause:*
  - On a new building the skeleton's Investigation stage gets no rows
    (`stageRows` keeps every skeleton stage, `internal/program/generate.go:475-483`;
    `investigationRows` adds none).
  - A stage with no children is not a summary row, so `status` and
    `leafDuration` treat it as a leaf with no duration
    (`internal/program/engine.go:243-249`, `:581-589`).
  - Unknown then spreads to every successor (`engine.go:288-303`).
  - Refurb programs (T03, T06) have Investigation rows, and are dated.
- *Knock-on:* the scorer reports program links as "unknown" (T04: 52). The
  earlier "no program link violated" for T02 and T13 was not scorable.
- *Location:* drop empty stages or give them zero duration (lane A).

**F48. "Local government area: North Sydney." is not an LGA candidate
(traced).**
- *Expected:* `proj.lga` North Sydney in ST03-base and strip-out.
- *Actual:* blank. The only candidate was "For the General Manager, North
  Sydney Council" in the signature (0.26 to 0.28).
- *Cause:*
  - `lgaAreaRe` takes the name *before* "local government area"
    (`internal/profile/details.go:44`). The planning certificate's label
    form puts the name after the colon.
  - The heading "NORTH SYDNEY COUNCIL" is upper case, and `lgaCouncil`
    matches only "Council".
- *Contrast:* ST03-da-pathway and T06 read it from "lodged with North
  Sydney Council" and "Issued by North Sydney Council" (1.0).

**F49. Fire services drawings file as fire engineer.**
- *Expected:* `consultant.fire_services` (the key also accepts hydraulic).
- *Actual:* FS-101 Rev A and Rev B file as `consultant.fire_engineer` at
  0.79 to 0.85, in all three T06 runs. The fire services options report
  (d12) is blank (0.38 to 0.42).
- *Location:* `data/intake/disciplines.json`, same family as F6, F25 and
  F40. Not traced further.

**F50. The refurb baseline suggests Interior Design on a services-only
upgrade.**
- *Expected:* no interior designer in ST06-pump-tank-only, which has no
  fit-out.
- *Actual:* an Interior Design package (origin calculation), its fee line
  and an RFP.
- *Cause:* `knowledge/works/package_defaults.yaml` baselines[4] adds
  Interior Design to every commercial, institution or mixed refurb.
  baselines[2] adds Architect the same way; the T06 shape already names
  that one.

**Shape and key gaps (recommended, not applied).**
- *T03 and T06 certifier and BCA:* the keys predate F9. They expect
  `blocking: [pathway_unsupported]` and every B stage absent. The app now
  prices `b2_cdc`, `b3` and `b4`, and `b1` as a boundary option (shown
  included).
  - Re-key them, as was done for the T12 certifier on 6 November.
  - For the BCA consultant, add the B stages as `app_differs` (option).
  - That is 7 RFP new failures per T03 run and 4 per T06 run.
- *Skeletons:*
  - T03 `da-pathway`: `skeleton: sk.refurb-traditional-da`;
  - T06 `tank-needs-consent`: `skeleton: sk.refurb-traditional-da`;
  - T04 `cdc-industrial-code`: `skeleton: sk.generic` (its `why` already
    says so).
- *Triggers that should restate `expect`:*
  - T03 `da-pathway` and T06 `tank-needs-consent`: certifier and BCA
    `a2_planning` and `b2_construction_approval` are in;
  - T04 `cdc-industrial-code`: architect `planning_response`, traffic
    `assessment_response` and `conditions_review`, BCA `a2_planning` and
    certifier `b2_construction_approval` are excluded, and `b2_cdc` is
    included;
  - T04 `flood-affected`: civil `flood_report`, `flood_evacuation` and
    `trunk_drainage` are I, and the complex service-level rows are I.
- *T06 `position.existing`* (fire services, electrical, mechanical): the key
  has `app: ask`. The T03 keys, checked by a Go run, have `app: absent`.
  Use absent.
- *T04 `cdc-industrial-code` cost plan:* alias "CDC fee" to "Construction
  certificate and certifier fees", as T03 and T06 do, instead of removing
  that line.
- *T06 `tank-needs-consent`:* add "Town planner" to `adds.cost_plan`.
- *T06 `pump-tank-only`:* keep the generic design rows ("Confirm the
  brief", "Prepare the concept design", "Client review of the concept",
  "Develop the design"). Remove only "Prepare fit-out documentation".
- *T03 and T06 `hdr.cond.quality_level`:* the field applies to residential
  only, so it is never captured on these types. Mark the keys
  `read_by_design: false`.
- *No key was changed.* The disputes below are owner questions, not clear
  key errors.

**Scorer limits.**
- `after` is read as finish-to-start. The app's SS link "Prepare fit-out
  documentation" → "Coordinate consultant documentation" scores as violated
  (T06 base).
- Program links are unscorable while F47 leaves every date unknown.

**What worked.**
- Every fee total, firm name and invoice amount is right in all nine runs.
- The planted invoice flags are all raised, with no false flags:
  - T03 `arithmetic`;
  - T04 `gst_rate` at 15 %;
  - T06 `gst_rate` at half rate.
- Every variant's trigger value is read: planning `da` and `cdc`, and
  `det.flood_hazard` true.
- The flood reaches the civil RFP (flood report, evacuation plan, trunk
  drainage).
- F9 is confirmed: the certifier prices `b2_cdc` on a CDC.
- F20 holds: no CDC job has a DA fees line.
- F37 holds on a fresh database.
- On the fit-outs, the tenant is the client, and the landlord's NABERS
  rating does not set a sustainability target.
- The survey's site area (16,012 m²) is now read (T04).

**Owner questions.**
- Is a vehicle crossing "work in a public road"? T04 says "No works in the
  public road other than the two heavy-duty vehicle crossings".
  - The app reads `flag.public_road` yes (0.68), and the civil RFP goes
    complex with public road works: 9 RFP failures per T04 run.
  - The key says no.
- F43: on a refurb, should an included system with an unsettled action stay
  in scope?
- Should a vacant floor in an occupied tower accept both
  `partial_occupation` and `live_environment` (T03; T06 accepts both)?
  Should T06 access accept both `restricted_hours` and `urban_constrained`?
- Should a combined services proposal fold into the four discipline
  packages, rather than add a "Services Engineer" package beside them?
  T03's cost plan carries five services fee lines.

**Found in the re-run `runs/20261107-t03-t04-t06-after-fixes` (app at
`bd24ab1`).** F47 now dates every program, so links that were unscorable
before are now scored.

**F51. The F43 fix still files two kinds of unsettled refurb works as
existing (traced).**
- *Expected (R16, and the plan's direction):* an included system on a
  refurb is filed as existing only when its works action settles on a
  non-works value.
- *Actual:*
  - *A conflict with `several`.* T06 `structure.alterations` (all three
    runs) is red with alternatives `investigate` (from the slab-scanning
    and assessment passages) and `several` (from the structural fee table).
    `unsettledWorksAction` (`internal/profile/existing.go:66-81`) counts a
    red row as unsettled only when an alternative is in `scopeAction`
    (`internal/profile/build.go:311`: new, replace, upgrade, alter, repair,
    remove). `several` is not in that list, so D-27 files the system as
    existing. The result: no "Alter and protect retained structure" row
    and no structure cost line, on a job with tank plinths and core holes.
  - *No action row at all.* T03 `interiors.painting` (all three runs:
    "painting throughout" in the concept report's "## Interiors" list,
    presence 0.99), `mechanical.local-exhaust` (all three),
    `access-egress.exits` (da-pathway and strip-out), and da-pathway
    `interiors.floor-finishes`; also T06 `interiors.internal-glazing`.
    Each has presence included and no `.action` row, so
    `unsettledWorksAction` never sees them. `existingPresence` (`:114-129`)
    then finds no action and files them as existing. Why no action row is
    stored for a works-list passage is suspected, not traced.
- *Knock-on:* there is no "Paint the fit-out" row in any T03 program.
  Separately, an unsettled system that stays in scope takes the refurb
  default `alter` (`internal/works/workitems.go` `LegacyDefaultAction`).
  So an unsettled `site.demolition` never meets `fr.strip-out-fitout`'s
  `remove`, and the strip-out variant has no strip-out rows.
- *Location:* `unsettledWorksAction` (add `several`), and `existingPresence`
  for a presence with no action reading (lane A).

**F52. Services fit-off is not linked after the joinery (traced).**
- *Expected:* in T03, sanitary fit-off (kitchenette sink and taps) and
  electrical fit-off (power to joinery) follow "Install joinery". The T06
  shape expects the same.
- *Actual:* in ST03-base and da-pathway the two "Fit off" rows start on
  days 233-235, but the joinery install finishes on day 248.
  - `fr.hydraulic-sanitary-fitout` and `fr.electrical-fitout`
    (`knowledge/program/fragments.yaml:2179`, `:2227`) link fit-off only to
    their own rough-in (FS plus the close-in lag).
  - `fr.joinery-fitout` (`:2360`) has no `before` to either fit-off.
- *Location:* add `after: [fr.joinery-fitout]` to both fit-off activities
  (lane A, program library).

**F53. A new warehouse's site start rows all start together (traced; F29
family).**
- *Expected (T04):* establish the site, then erosion and sediment
  controls, then bulk earthworks.
- *Actual:* in all three T04 runs, "Establish the site", "Erosion,
  sediment and dust control" and "Earthworks, cut and fill" have no
  predecessor. All three start on the first day of construction (day 265,
  or 200 for the CDC variant), in parallel.
  - `fr.site-establishment-7b` and `fr.erosion-control` both attach
    `after: stage_start`.
  - `fr.earthworks-7b` attaches after `fr.demolition-7b`. On a greenfield
    site that fragment is absent, and it has no attach of its own, so
    `presentAnchor` (`internal/program/generate.go:1009-1026`) falls back
    to `stage_start`.
- *Location:* the attach anchors of these three fragments, with F29
  (lane A, program library).

## Known gaps confirmed (already named in the shape)

- Electrical, hydraulic and mechanical consultants are suggested for a
  house (`package_defaults.yaml` baselines[0], scope_additions[2]), with
  matching cost lines plus "Active fire protection".
- No BASIX, landscape or QS package, no head contract (needs accepted work
  items). Missing cost lines: long service levy, HBCF insurance, s138 road
  fees, preliminaries and margin, PC and provisional sums, owner supply,
  balustrades.
- CDC: the town planner, "DA fees and contributions" and "Construction
  certificate and certifier fees" stay in. *(6 Nov: the DA fees line is a
  defect, F20, not knowledge.)*
- Trees: no "Tree protection" cost line.
- Supersession is never written (A-101 Rev A → Rev B), as DISCOVERY.md
  §4.5 predicted. Lifecycle is mostly blank, as Jev lifecycle answers are
  never applied.
- Title blocks laid out as a table (the messy tier) give no number or date,
  by design.
- T10 to T12 (run `20261106-t10-t12`):
  - No hazmat RFP template. The hazmat RFP is the generic draft, so none
    of its phrases match.
  - No BASIX, QS or building-consultant package.
  - Demolition is one row, with no services-off, clearance or SafeWork
    holds.
  - No subdivision content (T11).
  - Construction certificate before construction documentation (T11).
  - No construction-documentation stage on the CDC skeleton, so client
    sign-off for lodgement comes before construction documents (T10,
    T12).
  - Air conditioning after the linings.

## Key or harness issues (not app defects)

- **Invoice flags `no_supplier`, `unmapped` and `period`.** *(Fixed 2
  Nov, in the scorer.)* They follow from the project's state, not the
  document: nothing is allocated or confirmed in a harness run, and the
  upload month is after the invoice date. `harness/score.py` `state_flags`
  now expects them, and brief 03 says keys list only document-caused
  flags. No key changed.
- **Conflicts on site area (650 vs 650.2 m²), GFA (300 vs 298 m²), access
  and authority assets.** The documents genuinely disagree, by design in
  some cases. The app shows them as conflicts that include the true value,
  which is acceptable (Provisional Q4).
- **Missed "no" flags** (`sustainability_target`, `groundwater`, `staged`,
  `multi_building`), `det.existing_building`, `det.gas_supply`,
  `fact.contract_form` and `proj.regulated_design`: to be checked whether
  the documents' wording matches the app's reading triggers before calling
  them app misses.
- **RFPs.** *(Triaged 2 Nov: `runs/20261102-fixes-1-4-6/RFP_TRIAGE.md`.)*
  The ~100 "new failures" per scenario were mostly not app defects. The
  scorer now reads a position the app marks "not required by the current
  brief" as absent (18 per scenario), and reports blocks that wait on user
  input (allowances, issue fields, terms, client scope decisions) as
  "awaiting user input", not failures (about 50). The T01 shape's phrases are
  now keyed to template rows where one exists, and its structural
  inspection schedule is I, not O. Base new failures: 105 to 34, all
  knock-ons of blank profile values (14 from F4). The real RFP defects are
  F9 to F12.

- **RFP scores after scope by stage.** *(Fixed 4 Nov, in the scorer.)*
  Since 3 November every RFP discipline sets `scope_by_stage`, and
  `placeServices` (`internal/rfp`) shows a rules `option` or `ask` as
  `include` (positions `O`/`ask` as `I`; `done` as `exclude`). The scorer
  translated `expect` but not `app_differs.app`, which a static check over
  the 15 shapes put at about 401 false new failures and 482 false
  "fixed". `score_shapes.py` now translates both; a difference the layout
  hides counts as ok. Not yet handled: a position whose home stage is
  excluded shows as X. The shapes keep the rules' answers and need no
  change. **Owner question:** under scope by stage, an open client
  question (`ask`) is issued as included and priced; a key that calls it
  absent now stays a known difference.

- **T10 to T12 (6 Nov).**
  - **Scorer fixes.**
    - *Skeleton.* `score_shapes.py` let only `program.skeleton` set the
      expected skeleton. A trigger may now set `skeleton:`
      (`validate_shape.py` checks it). The T10 and T12 `da-pathway` and
      T11 `cdc-low-rise-code` triggers set theirs, so their skeleton
      "mismatches" were the scorer's. T01 `cdc-pathway` and T07
      heritage have the same gap; add `skeleton:` there too.
    - *`no_supplier`.* `harness/score.py` expected the flag on every
      invoice. Since F8 the reader picks a supplier, and `ledger.go`
      raises `no_supplier` only when there is no supplier package. The
      flag is now expected only when the app read no supplier (27
      false "missed" in this run; the pilots' after-fix scores had the
      same artefact).
  - **Shape fixes (reasons in each DOMAIN.md).**
    - *Architect budget block.* `expect.blocking: ["fact:budget"]` became
      `[]`: the budget is stated (SCORING.md §7.5, as T01). This was 9
      false new failures.
    - *DA cost lines.* The pathway triggers now add the CC fee line, and
      T12 adds the town planner line. The DA's correct CC line had been
      scored as a false positive.
    - *Building designer.* The T12 "Building designer" line has
      `aliases: ["Architect"]`.
    - *T12 certifier.* It no longer expects the pre-F9
      `pathway_unsupported` block.
    - *T10 certifier.* The four client-flag positions are keyed as in
      T11.
  - **Key fix.** ST10 `proj.lga` also accepts "Sutherland", as ST12
    does for the same council. The app keeps the area name without
    "Shire" or "City of" (`lgaTrailing`). The owner confirmed this
    convention on 6 November 2026.
  - **Still a scorer limitation.**
    - A trigger cannot change `rfps[].expect`. So in ST12-sewer, the
      architect `water_asset` and structural `authority_asset` rows the
      trigger wants count as new failures. So do the planning-response
      and B2 stage differences in the pathway variants.
    - `proj.regulated_design` has no reading, but the ST11 and ST12 keys
      derive `read_by_design: true`, so it scores as missed.
  - **Not app defects.**
    - The T12 surveyor invoice gets no supplier because no surveyor
      package exists when it is read. The surveyor is only a suggestion
      until the cost plan is prepared. The ledger asks the user to add
      the package; this is by design.
    - The T10 occupancy conflict is genuine: geotechnical fieldwork ran
      while the house was occupied. GFA 300 against 302 m² and site area
      700 against 700.4 m² are planned conflicts.
    - The T10 geotechnical `g4_planning` is a priced option after the
      boundary. The T10 shape keys it absent, an owner call.
    - The T10 and T12 certifier `b1_preliminary` is keyed absent, while
      T11 keys it as an option.
    - d04 (the architect's email) is filed as architect, where the key
      says none. This is the same question as T05 d04.
  - **Owner questions.**
    - Does demolition make `det.existing_building` true (F17)?
    - Should one strong document outweigh several weak passages (F27)?

## What worked

- Kind and title: 100% across 43 filed documents. Status 100%.
- The trigger values each variant changes were all read correctly:
  planning `da` and `cdc`, both tree flags `no`/`yes`, and site class M.
- Profile: 44 or 45 values correct out of 60 per scenario.
- Invoices: number, total and GST read correctly on every clean invoice.
- The arborist package was proposed in the trees variant.

# Walsh ingestion fixes — implementation plan

> For Claude: execute task by task, test first, in this session. Owner is
> away and asked for no questions (2026-10-27); decisions below are made in
> prose and can be reversed.

**Goal.** Fix every finding of the 2026-10-27 review of Walsh Renovation's
five documents (sections 1–4 of that review): scope the documents imply,
values read wrongly, the three ingestion causes, and the small set of facts
worth structuring.

**Lanes.** Profile readings, scope, budget figures and awards are lane A:
the existing answer keys (Meridian, Chen) and tests must still pass and the
budgets in `bench/budgets.json` stand. Labels and the Profile appointed-firm
line are lane B.

**Jev.** No new fan-out on any interactive path. New questions join the
existing background profile fan-out (one call per passage state,
https://docs.typesafe.ai/patterns/fan-out). Code finds candidates, Jev picks
one or none (https://docs.typesafe.ai/cookbooks/pre_parsed_value_extraction_cookbook);
code parses dates and money. Implied scope is a code lookup and is shown as a
suggestion, never as evidence. Question wording changes bump the question
version (https://docs.typesafe.ai/concepts/how-to-build-with-system-one).

**Unknown stays unknown.** No rate, contingency amount, duration or firm is
invented. Where a document is ambiguous the app says so.

## Decisions made without the owner

- D1 GFA. Three scale fields: existing GFA, GFA added by the works, and
  proposed GFA (the building after the works; the existing `gfa_sqm` key,
  relabelled, so RFPs keep reading it). Code sorts each area candidate by its
  own words ("existing" / "addition, extension, added, additional" / other).
  When proposed is unknown and existing and added are both stated, Go
  computes proposed = existing + added (origin calculation). The Program's
  floor-area driver uses a stated planning value first, then for an
  extension the added area, then the proposed GFA, then its 250 m²
  assumption.
- D2 An applied action (new, replace, upgrade, alter, repair, remove) on a
  system puts that system in scope; retain and investigate do not.
- D3 Implied scope ("ensuite" implies wet-area waterproofing, …) comes from
  a draft knowledge file and is a *suggested* scope row with the source
  words in its note. The user's out always wins.
- D4 A fee proposal superseded by the firm's own engagement letter is
  `superseded` (by the letter), not `unsuccessful`.
- D5 Award lines keep the letter's own wording; the mapped stage stays the
  stage link. A wrong stage suggestion is fixed at the mapping, not hidden.
- D6 Firms named in a document dropped on a package (an invited builder, a
  consultant already advising) are shown on the package row as "Named"
  firms. Nothing is awarded and no money moves.
- D7 Owner targets are three project details (target DA lodgement, target
  construction start, maximum months on site); the builder's preliminary
  estimate is one detail (cost range and months). The Program shows each
  target beside the programmed date; the Cost Plan shows the estimate beside
  the budget. Neither changes a computed date or a budget.
- D8 PI insurance: only the cover limit and the period end are kept, on the
  submission. An expired period shows "request a current certificate".
- D9 The live-occupation allowance ($25–40k) stays text in the estimate's
  source: it sits inside the builder's range, so a cost line would double
  count.
- D10 Verification of procurement fixes runs on a new dev project
  "Walsh check" with the same five files, so the owner's Walsh procurement
  history is not rewritten. Walsh's profile is re-read (the owner's own
  Update action); user values stand.

## Tasks

Each task: write the failing test, make it pass, run the package tests.
Tests run with `SITEWISE_TEST_DATABASE_URL=postgres://sitewise@127.0.0.1:5433/sitewise_test?sslmode=disable`
and Go from `.tools/go/bin`.

### Section 1 — scope

**T1 Room and element words route a passage to its systems.**
`internal/jobs/worker.go` `explicitEvidenceRoutes`: kitchen, bathroom,
ensuite, laundry, WC/powder room, robe → interiors and hydraulic (and
mechanical for wet rooms and kitchens); party wall / separating wall → site,
fire-passive, interiors; strip-out / demolition → site; masonry, repoint,
sandstone → structure; BASIX → envelope. Routing only: Jev still answers
inclusion.
Test: `internal/jobs` — the brief's design-objectives passage and the
builder's caveat passage get the interiors / hydraulic / site / fire-passive
leaf questions.

**T2 An applied action puts the system in scope.**
`internal/profile/build.go` `withScope`: a whole-project `sys.<leaf>.action`
row in band amber/green/user with an action other than retain/investigate
adds `scope.<leaf>` in, unless presence says not included or the user said
out.
Test: `internal/profile/scope_test.go` — stairs alter (amber) and sanitary
upgrade (amber) give scope rows; retain gives none.

**T3 Implied scope from the documents' own words.**
New `knowledge/profile/scope_terms.yaml` (draft): term regex → systems.
The reading job stores one code fact per matched passage
(`implied.<leaf>`, decided_by rule, excerpt = the matched sentence); Build
turns them into suggested scope rows with note "Implied by '<term>'".
Test: knowledge load test; profile Build test (ensuite → wet-area
waterproofing, tiling, sanitary, cold and hot water, local exhaust; kitchen
→ joinery; party wall → adjoining structures, fire compartments, acoustic
separation; strip-out → demolition; repoint → masonry; BASIX → thermal
performance; new bedroom → internal doors).

**T4 A party-wall signal.**
`knowledge/clusters/delivery/signals.yaml`: `sig.party-wall-stated`
("does the passage state work to, or a tie-in with, a wall shared with a
neighbouring building?"), runs_on site, structure, fire-passive, interiors.
`python tools/check_knowledge.py` passes.

**T5 Unknowns stay unknown.**
Test: Walsh fixture build has no scope row for hazardous materials,
switchboards, landscape, termite management or underpinning.

### Section 2 — values

**T6 GFA existing / added / proposed** (D1). `knowledge/profile/taxonomy.yaml`
scale fields, `internal/profile/harvest.go` (candidate sort),
`internal/profile/build.go` (derived proposed), `internal/store/program_prepare.go`
(driver). Tests: harvest ("~95 m² addition GFA" → added only), build
(existing 120 + added 95 → proposed 215 calculation), program prepare.

**T7 LGA without the planning instrument.** `internal/profile/details.go`
`lgaCandidates`: drop trailing LEP / DCP. Test: "City of Sydney LEP/DCP
apply" → "City of Sydney".

**T8 Architect lodges.** `knowledge/profile/flag_readings.yaml`: the
question also accepts an architect's stated services that run through to
lodgement. Question version bump. Live check on Walsh.

**T9 Budget framing.** `internal/store/cost_budget.go`: the budget's source
excerpt is read in Go. "Including contingency" → the spread rationale says
the budget includes a contingency the document does not quantify;
"additional contingency" → the contingency rationale says it is held in
addition to the budget; GST basis not stated → said in both. Test: store
cost test with Walsh's wording.

**T10 Letter date.** `internal/proposals/letterhead.go` `headerDate`: a
bare date line in the header counts. Test: engagement letter → 2026-03-03.

**T11 Fee proposal superseded by the letter** (D4).
(a) firm from a sign-off ("Director, Atelier North") when there is no
letterhead; (b) award marks other submissions of the same firm in the
package `superseded` by the awarded one. Tests: proposals reader, store
award test.

**T12 Stage names and mapping** (D5). `internal/store/appointment_scope.go`:
row label keeps the source wording; the stage mapping prefers the stage Jev
priced the fee against (`price:stage:*` cell) over a partial coverage
guess. Test: Walsh letter fixture maps Construction documentation →
For-construction documents and Construction administration → Contract
administration, labels unchanged.

**T13 Monthly wording.** `internal/proposals/offer.go`: a fee line whose
own row says monthly / per month / × N months, where the lines add to the
printed total only as lump sums, gets a check note. Test: Walsh letter.

**T14 Invalid ABN said, not hidden.** `internal/proposals/read.go`: a
printed ABN that fails the checksum adds flag `abn_invalid` and the
printed value; the award firm step and comparison show it. Test: reader.

**T15 Named firms on a package** (D6). Submissions that are not proposals
keep their firm; the package overview lists them as named firms with the
source file. Procurement row shows "Named: …". Tests: store overview,
browser check on Walsh check.

### Section 4 — structure a little

**T16 Owner targets** (D7). `knowledge/profile/project_details.yaml`
`target_da_lodgement`, `target_construction_start`, `max_construction_months`
(pre-parsed candidates: month-year, early/mid/late-year, "N months"). Go
parses each to a latest date or a month count. Program read returns
`targets` (target, programmed date or months, met / later / unknown);
Program shows them in its header. Tests: details candidates, parse, program
read.

**T17 PI limit and expiry** (D8). Migration adds `pi_limit_cents`,
`pi_expires_on` to submissions; the reader parses the professional
indemnity passage in Go; procurement row and comparison show expiry.
Test: reader on Walsh letter → $5,000,000, 2026-06-30, expired on
2026-10-27.

**T18 Builder's preliminary estimate** (D7, D9). Detail
`preliminary_estimate` (money range candidates) and
`preliminary_construction_months`; Cost Plan totals show the range beside
the budget; Program targets show the months. Tests: details, cost totals.

**T19 Appointed firms on Profile.** Profile API returns awarded packages
with firm and contact; Profile shows "Appointed: Atelier North Pty Ltd
(Lucia Ferraro)". Test: httpapi profile test.

### Verification

V1 `go test ./...` (note pre-existing failures named in memory: Program and
profile-layout tests, hold-point CRLF), `python tools/check_knowledge.py`,
`npm --prefix web run build`, `npm --prefix web test`.
V2 Answer keys: Meridian and Chen scorers stay at or above their floors.
V3 Live: rebuild the dev server, re-read Walsh's profile with Jev; record
each item before/after in `docs/evidence/2026-10-27-walsh-ingestion.md`.
V4 Live: create "Walsh check", upload the five files, read, drop the fee
proposal and letter on Architect, award by stage, drop the builder email on
Head contract and the heritage advice on Heritage; record results.
V5 Timing: profile build, procurement overview and cost plan read against
`bench/budgets.json`.

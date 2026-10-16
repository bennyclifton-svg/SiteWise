# Procurement from fee proposal to award: implementation plan

Date: 16 October 2026. Lane A (fees a PM awards and certifies). Source:
`docs/plans/2026-10-16-procurement-process-to-award` (the owner's plan). This
note maps its placeholders to the repo, records what the agent decided while
the owner was away, and is the task list the build follows. Every decision
marked **assumed** can be reversed without a migration unless noted.

## 1. What the repo already has (the plan's "for the coding agent" question)

| Plan asks | Found | Consequence |
|---|---|---|
| How the RFP stores the returnable schedules and the scope by stage | Not stored as rows. `rfp.Resolve` (pure) turns discipline knowledge + frozen drivers + `report_settings` into `Resolved`; `rfp.Sections` prints stages, options and returnables as text blocks. Settings are per report draft. | Phase 0 adds `rfp.Schedule(Resolved)` and persists its rows per package (`schedule_items`), synced from the current RFP draft where they are used. The stage table is printed from the same rows. |
| Is there a firms or suppliers table | No. `packages.supplier_abn` only (invoice spec). | New org-wide `firms`. Award writes `packages.supplier_abn` so the invoice supplier rule settles by ABN. |
| Do cost lines support a parent link | Yes: `cost_item_revisions.parent_item_id`; a parent is `posting=false` and the grid shows it as a group. The posting trigger forbids any value on a group. | Award turns the package's fee line into a group that keeps its **budget**; a migration lets a group hold a budget only. Children hold the contract value (`commitment`). |
| Kind "commercial document" | `data/intake/kinds.json` already has `commercial` ("pricing or payments"). | No kind change. The subtype (fee proposal / tender / not a proposal) is a proposal question. |

## 2. Decisions taken while the owner was away

1. **The procurement questions do not ride in the filing call (assumed; deviation).**
   The filing gate replays every recorded Jev request byte for byte
   (`internal/eval/replay.go`); adding questions to the filing call changes
   every request and invalidates the intake answer key, the recordings and the
   bench. Instead, a filed document that is commercial or carries a proposal
   cue (filename or heading: fee proposal, fee submission, tender, quotation,
   proposal) gets its **own** fan-out straight after filing: one call that
   asks subtype, package (only when rules cannot settle it), firm identity
   (only when the directory cannot), every price and rate mapping and every
   non-price criterion. Filing itself is unchanged and stays inside 1 s / 2 s.
   The submission row is created when filing commits, so the proposal shows
   under its package at filing time ("Reading…"); the firm and prices follow
   in about a second.
2. **Package by rule.** The document's filed discipline (any band, including
   provisional amber) with exactly one live package of that discipline settles
   the package. Dropping on a package row settles it as the user's act. Two
   packages of one discipline: Jev picks between them in the same call.
3. **Firm by rule.** A checksum-valid ABN that matches a directory firm settles
   the firm and its details fill from the directory with no firm questions. A
   normalised name equal to one directory firm also settles it. Otherwise Jev
   picks the name among harvested candidates (letterhead, signature block,
   "Pty Ltd" lines, email domain) and code creates a **draft** firm (amber).
4. **Thresholds.** Proposal questions have their own version (`proposal-1`) and
   calibrated thresholds in `data/proposal/thresholds.json`, fitted by
   `cmd/proposal-eval` on real fee proposals with the same policy as invoices
   (Wilson lower bound z = 1.645, green ≥ 0.95, amber ≥ 0.70; agent's
   proposal, owner to sign off). Uncalibrated shapes stay blank.
5. **Phase 4's generative model is not wired.** AGENTS.md non-negotiable #1
   forbids it until the owner amends it; the plan says the owner amends it
   first. Phase 4 is built with the model **off**: Jev picks the passages for
   each criterion, code collects departures, the PM rates, and code drafts a
   factual recommendation from computed price facts only (no prose
   generation). The validator (cited ids exist, every summary cites, every
   number matches a computed value or a cited passage) and the cache table are
   built and tested against planted failures, so wiring a model later is one
   adapter and the owner's AGENTS.md change.
6. **Default schedule.** A package with no RFP gets the discipline's default
   schedule by resolving its RFP with no settings; a discipline with no RFP
   knowledge (trades, some specialists) gets a generic schedule: one lump-sum
   price item, reimbursables, the skeleton's rate roles and returnables.
7. **Rates.** R3 roles come from each discipline's R3 note ("Roles: a · b · c");
   a discipline without one uses the skeleton's default roles.
8. **Money.** Integer cents, AUD, ex GST, in Go. A proposal priced inc GST is
   normalised by code (÷ 1.1 to the cent) with a note. A percentage fee is
   computed from the cost plan's construction budget total when it is known
   and shown with its basis; otherwise it stays unknown.

## 3. Data model (migration `032_procurement_award.sql`)

Every table carries `org_id`; every foreign key is composite with `org_id`.

```
schedule_items(org_id, project_id, package_id, key, kind, label, detail,
  ordinal, base bool, retired_at, updated_at)
  kind: stage | option | rate | criterion | reimbursable
  PRIMARY KEY (org_id, package_id, key)

firms(org_id, id, name, abn, address, contact, email, phone, status
  draft|confirmed, created_at, version)            -- org-wide, no project
  UNIQUE (org_id, abn) WHERE abn IS NOT NULL

submissions(org_id, id, project_id, package_id null, firm_id null,
  document_id, kind fee_proposal|tender, status reading|received|
  needs_review|awarded|unsuccessful|superseded|withdrawn|not_proposal,
  received_on date, supersedes, stated_total_cents, gst_basis ex|inc|unknown,
  option_note, retry_after, read_attempts, created_at, version)
  UNIQUE (org_id, document_id)

submission_decisions(org_id, submission_id, field, value, band, decided_by,
  question_version, confidence, proposed_value)     -- like invoice_decisions
submission_prices(org_id, id, submission_id, item_key null, label,
  amount_cents null, state priced|not_priced|included|excluded|rates_only,
  basis text, page, passage, band, decided_by)
  item_key null = an "other item" the firm offered unasked
submission_passages(org_id, submission_id, id, heading, page, text)
submission_responses(org_id, submission_id, item_key, passage_ids,
  summary, rating strong|acceptable|concern null, decided_by)
comparison_reports(org_id, id, project_id, package_id, input_sha256,
  model, output jsonb, recommendation, edited_by, signed_off_by, signed_off_at)
awards(org_id, id, project_id, package_id, submission_id, parent_item_id,
  parent_label_before, parent_created bool, created_at, undone_at)
cost_line_sources(org_id, project_id, cost_item_id, award_id, item_key null,
  price_id null)
packages: + awarded_firm_id, awarded_submission_id
cost_posting_value(): a group line may hold a budget value only.
```

## 4. Flow

```
drop (register or package row)
 → existing upload + filing (unchanged call, 1 s / 2 s)
 → Runner.AfterFile(doc, text): commercial or proposal cue
     → store.OpenSubmission (package by drop or rule; "reading"; event)
     → proposalReader.Start (own slots, background priority)
         extract ≤ 12 pages → proposals.Harvest (money rows, totals, GST
         wording, rates, %, ABNs, emails, phones, names, sections)
         → rules: subtype, package, firm (ABN / name), GST basis
         → ONE Jev fan-out: what is still open + item mappings + criteria
         → code: leftovers → other items, ex GST, sum check, flags
         → store.CommitSubmissionRead (decisions never over a user's;
           supersede an earlier submission from the same firm; event)
As built (routes under /api/projects/{p}):
POST  /procurement/packages/{pkg}/submissions?name=   drop on a row
GET   /procurement/board                              firm chips per row
GET   /procurement/packages/{pkg}/prices              price table (+ /export xlsx)
PUT   /submissions/{s}/cells/{key}                    a user's cell
PATCH /submissions/{s}                                firm, package, status, date
GET   /submissions/{s}/file                           the PDF, inline
POST  /procurement/packages/{pkg}/firms               add a firm by hand
POST  /procurement/packages/{pkg}/award/preview       step one: the lines
POST  /procurement/packages/{pkg}/award               step two: commit
DELETE /procurement/packages/{pkg}/award              undo (refused once invoiced)
GET|PUT /procurement/packages/{pkg}/report            report, recommendation
PUT   .../submissions/{s}/ratings/{criterion}         a user's rating
GET   /api/reports/{r}/response-form                  the RFP's response form
GET   /api/firms?q=                                   the org's firm directory
```

## 5. Budgets (bench/budgets.json; proposed ones need owner sign-off)

| Path | p50 / p90 ms |
|---|---|
| filing a proposal (whole_intake, unchanged) | 1,000 / 2,000 |
| `proposal_read` prices read, from filing commit | 3,000 / 6,000 (proposed) |
| `price_table` five firms | 150 / 300 (proposed) |
| `award_write` | 200 / 500 (proposed) |
| `procurement_report` cached reopen | 150 / 300 (proposed) |
| `submission_write` cell or firm edit | 100 / 250 (Procurement write budget) |

## 6. Tasks

Phase 0
- [x] P0.1 `rfp.Schedule(Resolved)`: stage, element, option, reimbursable,
      rate and criterion items with stable keys; tests incl. "changing a stage
      or option changes the items with no code change".
- [x] P0.2 Stage table printed from the schedule rows (same rows = same scope).
- [x] P0.3 Returnables as tables (R2–R9), wording aligned with the report rows;
      page-count limit (O1, three pages) still holds; goldens refreshed.
- [x] P0.4 Migration + `schedule_items` synced from the package's current
      RFP draft where the rows are used (writing them on every RFP edit cost
      the setting click ~8 ms against a 100 ms budget); default schedule for
      a package without an RFP.

Phase 1
- [x] P1.1 Migration: firms, submissions, decisions, passages; isolation test.
- [x] P1.2 `internal/proposals`: cues, harvest (firm candidates, money rows,
      sections), rules, questions, apply, finish.
- [x] P1.3 Runner.AfterFile hook, proposal reader, retry sweep, resume.
- [x] P1.4 Store: open/commit submission, supersede, firm directory match.
- [x] P1.5 API + Procurement page firm columns, Add firm, drop on row,
      Review/Compare, proposal side panel (PDF beside fields).

Phase 2
- [x] P2.1 Price harvest, mapping questions, leftovers, GST and sum checks,
      percentage and rates-only cases (unit tests).
- [x] P2.2 Price table API + screen (review with one firm, compare with
      several), user correction, totals recomputed in Go, xlsx export.
- [x] P2.3 Answer key from real fee proposals (≥ 30, private), live run,
      thresholds.

Phase 3
- [x] P3.1 Migration: awards, cost_line_sources, package award columns, group
      budget trigger.
- [x] P3.2 `costs.Summarize` and the invoice summary roll children up to a
      budgeted group; grid shows group budget, contract, variance.
- [x] P3.3 Award preview + commit in one transaction; combine rows; undo
      refused once an invoice is allocated; other submissions unsuccessful.
- [x] P3.4 Invoice: awarded ABN settles the package; line options are the
      package's child lines, described with the schedule items they hold.

Phase 4
- [x] P4.1 Criteria passages (Jev, in the same proposal fan-out), departures.
- [x] P4.2 Report page (A4 landscape print), ratings, recommendation drafted
      by code from facts, sign-off; cached by input hash.
- [x] P4.3 Validator for a future model draft, tested with planted failures.

Gate
- [x] Bench paths, Playwright (drop three, compare, correct a cell, award,
      cost plan shows children, undo), full Go suite, evidence note.

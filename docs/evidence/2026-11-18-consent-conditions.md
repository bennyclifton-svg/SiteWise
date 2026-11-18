# Consent conditions in the register: what was measured

Date: 18 November 2026. Branch `wave/conditions` (from local `main` with the
register and RFIs; `main` merged in once more at the end for the project
report and guest access; not merged back, not pushed, not deployed). Plan:
`docs/plans/2026-11-17-consent-conditions.md`. Everything here was run on
the development machine against private PostgreSQL 17 schemas
(`wave_conditions`, `wave_conditions_e2e`), with stand-ins for Jev and the
model. The live server was not contacted, and no live Jev or model call
was made. The `conditions` group of the command key is a draft the owner
has not reviewed. The consent answer key does not exist yet: its format is
in `data/eval/consent/README.md` and its folder is ignored by git; no case
was invented.

## 1. What was built, by task

| Task | What | Where |
|---|---|---|
| 0. Key and fixtures | `.gitignore` line for `data/eval/consent/private/`; the key's format and loader (`LoadKeys`, `SplitScore`, `PagesFile`); the public Seven Hills notice copied to `testdata`. | `data/eval/consent/README.md`, `internal/consent/key.go` |
| 1. Condition kind | `condition` (open, evidence filed, satisfied, not applicable; open while open or evidence filed). `ConditionDetails` with unknown fields refused and the text kept to 12,000 characters. An edit keeps the notice's number, heading, words, source, evidence and RFP line; it changes the gate, who acts and evidence needed only. Evidence is linked by Go (`link_evidence`), and an open condition becomes evidence filed. | `internal/delivery/condition.go`, `items.go`, `internal/store/consent.go`, `delivery_edits.go`, `delivery_register.go` |
| 2. Go split | `Detect` (a notice marker and three numbered conditions) and `Split`: numbers in sequence per prefix (`12`, `A3`, `B1`; a jump of up to four so a scan's gap still finds the next; "Condition 13" written out may start anywhere); stage headings in capitals (a short paragraph may sit before the first condition) or naming a stage; running headers and footers dropped at page edges; "Advisory notes" ends the list; Markdown tables with a No. column; consent and modification references as printed; the determination date by rule (a modification's own date first); "(Amended … MOD/2023/0177)" and "(Deleted …)" markers with the modification named; gaps as ranges. | `internal/consent/split.go` |
| 3. Jev call | One background fan-out per condition: `party` (24 options), `system` (13 systems and none), `evidence` (noul), and `timing` only when the heading left the gate open. Options in a fixed order (a hand-built JSON object; a Go map would sort them). Bands at TypeSafe's starting floors, amber at most; a timeout is grey. Fingerprint over version, state and questions. Parties checked against the filing vocabulary. `cmd/consent-eval` writes a notice's pages and scores splits and picks against the key; it only reports. | `internal/consent/read.go`, `cmd/consent-eval` |
| 4. Job and store | Job kind `consent` on the always-on lane, sixteen in flight. `UpsertConditions` in one transaction under the lease: new numbers proposed; a proposed row takes newer words and picks; an accepted row only flagged ("Changed by MOD/2023/0177"; "changed since satisfied, check"); deleted is a flag proposing not applicable; an older notice never overwrites a newer one; a removed row is not brought back. A read the PM asked for is one change that PM can undo. A document filed as a Statutory Instrument that Go recognises is queued after its text is stored. | `internal/jobs/consent.go`, `internal/store/consent_read.go`, `worker.go`, `loop.go` |
| 5. Read request | `POST /projects/{id}/delivery/imports` {document_id}: 202 queued; 422 "No numbered conditions found" (or no text yet, or over 600); 409 while queued or running; 400 for a malformed id; 404 for another org's or project's document. The register read carries each read's status, count, gaps or error, and the Profile's consent number, date and lapse date. | `internal/httpapi/delivery.go`, `projects.go`, `speed.go` |
| 6. Operations, chat, key | See section 3. Attention gains `register.conditions_due` and `register.conditions_open_at_gate` on `reports.ConditionOpenBefore`, the definition the named filters use. | `internal/httpapi/ops_consent.go`, `ops_reads.go`, `chat_register.go`, `chat_view.go`, `attention.go`, `internal/reports/attention.go`, `data/eval/chat/` |
| 7. Program gates | One gate hold point per gate with live, applicable conditions (`consent:before_cc` …), in the outline too, through the existing review (source `consent`), with no count in its label. Linked before the certificate activity; with none (the generic outline has none), the gate closes the stage before construction and holds it, or for occupation and subdivision follows the last construction activity, with a generation note. `consent_gate_open` when a gate is released with conditions open. Release readings skip gates (no interface). | `internal/program/consent.go`, `generate.go`, `review.go`, `internal/store/program_prepare.go` |
| 8. Evidence | In the one `register` job and its one Jev request: Go's "Condition 13" / "Conditions 4 and 5" rule (with the consent reference when the project has two), then one noul per condition the rule left over a shortlist of at most 12. Readings in `delivery_readings` (field `evidence`); Jev's wait for `evidence_amber` in `data/register/thresholds.json`. | `internal/jobs/register.go`, `internal/store/delivery_match.go` |
| 9. Screen | Conditions in the one table, grouped by gate; No., Due (gate), Status in words, amber "to check"; the strip under the table with the council's words, Open page N, how each value was decided, possible evidence, Link evidence, Satisfied, Add to RFP; Read conditions in the three-dot menu; the header line. | `web/src/Delivery.tsx`, `deliveryApi.ts`, `Project.tsx`, `register.css` |
| 10. Bench | `TestConsentBudget`, `TestConsentReadGoSideBudget`, `TestSplitBudget`; the intake bench seeds 300 conditions. Rows `delivery_import` and `consent_read`. | `internal/httpapi/register_budget_test.go`, `internal/jobs/consent_test.go`, `cmd/intake-bench/delivery.go`, `bench/budgets.json` |

**Migration 068** widens the delivery kind and status checks with
`condition` and its statuses, `delivery_readings.field` with `evidence`,
and `program_changes.source` with `consent`; adds an index on condition
rows and `jobs.requested_by` and `jobs.outcome`. It rewrites no check that
067 or 070 own (`project_changes` area and via, report kinds), so it gives
the same result before or after them.

## 2. Speed

Development machine, private schema, stand-ins; thirty samples per path
unless said.

| Budget row | Budget p50 / p90 | Measured p50 / p90 | Held by |
|---|---|---|---|
| `delivery_read`, 300 condition rows | 100 / 250 ms | 12 / 14 ms | `TestConsentBudget` |
| `delivery_write`, a condition's gate through `POST /ops` | 100 / 250 ms | 14 / 15 ms | `TestConsentBudget` |
| `delivery_import`, 120-condition notice | 300 / 1,000 ms | 9 / 10 ms | `TestConsentBudget` |
| Go split, 25 pages, 102 conditions | 20 / 50 ms | 5 / 10 ms | `TestSplitBudget` |
| `consent_read`, Go half only (instant stand-in), 120 conditions, ten samples | 5,000 / 10,000 ms | 71 / 91 ms | `TestConsentReadGoSideBudget` |
| `consent_read` with live Jev | 5,000 / 10,000 ms | not measured | `cmd/consent-eval -live` (not run) |
| `register_match` with conditions | 2,000 / 4,000 ms | not re-measured | plan 4's row |
| `whole_intake` | 1,000 / 2,000 ms | not re-measured | the auto-queue is one kind lookup and Go's split in the `full_text` job, after the text is stored |

## 3. Chat and conditions

| | From chat |
|---|---|
| Reads | `read(delivery)` with part `condition` or a gate (`before_cc` …): No. (the printed number), item, With, gate, status with "evidence needed", "possible evidence", "to check" and any flag; the consent's facts; each read's count and missing numbers. Printing a condition marks the turn as having read outside text (the printed heading is the council's words). |
| At once, with Undo | `delivery.edit` on a condition: gate, who acts, discipline, evidence needed, evidence filed or not applicable, `link_evidence`. |
| Waits for a click | `delivery.satisfy` (and `delivery.edit` with status satisfied is refused from chat); `delivery.accept_evidence`; `delivery.read_notice` (a prepared read); `reports.add_scope_line` (the RFP's settings are not captured by the change log, so it has no Undo; the screen's Add to RFP applies it at once). |
| Opened filled in | Nothing: sending conditions out is plan 1's. |
| Screen | The `register` table's named filters `open_before_cc`, `open_before_oc`, `no_one`, and `to_check` (now including conditions SiteWise read). Rows are named "Condition 14". |

**The key.** `conditions` group, seven draft cases (`cn1` to `cn7`): two
reads, the "nobody owns" filter, rows selected by hand given to the
landscape architect, and three that wait for the click (Add to RFP,
satisfied, read the notice again). The house fixture's consent is read by
the real route and job with a stand-in for Jev. `TestScreenCasesScoredOffline`
scores all seven with the stand-in model; all pass.

## 4. The split on the owner's notices (not an accuracy measure)

`cmd/consent-eval -pages -list` was run locally on four private notices,
with the pages written to a scratch folder outside the repository and no
call to any outside service. It is a check that the rules fit their
layouts, not a scored result: there is no key.

| Notice | Pages | Conditions found | Note |
|---|---|---|---|
| Petersham, 18 October 2016 | 25 | 103, no gaps | The notice prints conditions 1 to 103; the plan says 102. Five stage headings in capitals, two with a paragraph before the first condition; advisory notes end the list. No printed headings per condition. |
| Petersham MOD 2018 | 25 | 103 | One "Deleted" (89); its own date read as 11 October 2018. |
| Petersham MOD 2023 (0177) | 32 | 103 | Markers on 1, 4, 8, 9, 11, 50, 51, 62, 65, 89, each naming its modification; date 4 August 2023. A plan schedule's "04 August 2023" lines were first read as condition 4 until bare numbers before a month were refused. |
| Camp Y amenities, 8 November 2021 | 13 | 41, no gaps | Mixed-case stage headings and "Operational Conditions"; every condition's printed heading read. |

The scanned Bankstown consent was not tried (it needs the OCR text).

## 5. What was not measured

- No live Jev call: the picks' accuracy is unmeasured, the answer key is
  empty, and every pick stays amber at most. No cut-off is proposed.
- No live model run of the `conditions` group.
- No owner review of the key format, the fixture rows, the screen or its
  wording.
- `tools/check.ps1` was not run; the intake bench was not run end to end.
- The screen was checked by Playwright only, not by eye.

## 6. Where the build differs from the plan

- **Petersham has 103 conditions**, not 102 (section 4).
- **There is no "Authorities" filing value.** The auto-read keys on the
  Statutory Instrument kind and Go's `Detect`; Read conditions offers every
  filed document, Statutory Instruments first.
- **Evidence is linked through `delivery.edit` (`link_evidence`)**, not
  `delivery.link`, which is the delivery dependencies operation.
- **`reports.add_scope_line` stays click-only from chat** and applies with
  no Undo from the screen, because `report_settings` has no capture
  trigger yet. The line goes to the RFP's own stage: design finalisation
  or delivery when the RFP has it, else the last stage before delivery,
  else the last stage (the arborist RFP has neither). Add to RFP is
  disabled when the row has no package; a package with no RFP draft is
  refused with "Create the RFP first." rather than disabled.
- **A read queued after filing is not one undoable change**: no one asked
  for it and documents carry no uploader, so it has no actor; the PM
  removes rows in one bulk action. A read the PM asked for is one change
  (`jobs.requested_by`).
- **The read's outcome is kept on the job** (`jobs.outcome`): count, gaps,
  references; the plan had no home for gaps.
- **Gates with no certificate activity** close the stage before
  construction (so Adopt does not chain them after the work they hold), or
  follow the last construction activity for occupation and subdivision.
- **The gate date for attention** is the gate hold point's start, or
  before Adopt the certificate activity's start; with neither there is no
  date and no attention item.
- **"Open page 12" opens the notice**; the viewer does not jump to a page.
- **The Petersham, Camp Y and Bankstown key tests** skip until the owner's
  key and the notices' pages files exist (`TestPrivateKeySplitsExactly`).
- **Size.** About 3,400 lines of product Go (the plan estimated 1,010) and
  about 180 of TSX, plus the eval tool and test helpers.

## 7. Known limits and what waits

- **The owner:** the answer key (Petersham, the modifications, Camp Y,
  Bankstown, about 20 consultant documents as evidence), a reviewed cut-off
  for evidence picks, and a look at the screen.
- **Plan 1 (correspondence):** emailing conditions out; notices from
  messages.
- **Plan 5 (report):** its register section now lists consent conditions,
  with their gate as the target; counting open conditions per gate is the
  report's to add.
- **OCR:** a scanned notice is read from its OCR text once it is stored;
  whether the OCR path re-runs the auto-read was not traced, so Read
  conditions is the sure way.

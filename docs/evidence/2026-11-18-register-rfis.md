# The register and RFIs on the building: what was measured

Date: 18 November 2026. Branch `wave/register` (from local `main`, with
`main` merged in once so the shared `Attention` function was present; not
merged back, not pushed, not deployed). Plan:
`docs/plans/2026-11-17-rfis-review-comments.md`. Everything here was run on
the development machine against a private PostgreSQL 17 schema
(`wave_register`), with stand-ins for Jev and the model. The live server
was not contacted, and no live Jev or model call was made. The command
key's new `register` group and the Question A answer key are drafts the
owner has not reviewed; the answer key has no cases yet.

## 1. What was built, by task

| Task | What | Where |
|---|---|---|
| 1. Kinds, statuses, details | `rfi` (draft, open, answered, closed, withdrawn; open while draft, open or answered) and `notice` (open, closed, withdrawn). `RFIDetails` and `NoticeDetails` with unknown fields refused and bounds. A kind change to or from `rfi` is refused. `SystemID`, `DeterminantID` on rows. `With` is "You" for an answered RFI. `DueDate` is one call to the wave's shared `program.BusinessDayAfter` (written here; index agreement 1). | `internal/delivery/`, `internal/program/calendar.go` |
| 2. Migration and store | Migration 066 (below). RFIs numbered `max+1` under the project lock, backed by a unique index; due date from the raised date and the response time on the Program's calendar (Monday to Friday without one); a due date the PM typed survives a later change to the raised date; With filled from the package's awarded firm; system from the work item, or from a trade package whose `builds` fall in one top-level system. Systems, determinants and answer documents are checked against the catalogue, the org and the project. The read leaves out Program's activities and milestones (an accepted hold point stays), lists only kinds with rows, adds With, working days late and proposed readings, and with a system the also-open line. Accepting a reading is an ordinary edit (`accept_reading`, values set in Go); a dismissal alone leaves the row's version. Three tables are captured for Undo; an undo touches the delivery revision and sends the event. | `internal/db/migrations/066_register.sql`, `internal/store/delivery_register.go`, `delivery.go`, `delivery_edits.go`, `change_undo.go` |
| 3. Routes and speed paths | `GET /delivery?kind=&system=` (timed on `delivery_system_read` when a system is named); `PUT /delivery/settings` (`delivery_write`). Wrong org: 404. | `internal/httpapi/delivery.go`, `projects.go`, `speed.go` |
| 4. The post-filing job | `register` is queued beside `queueVisual` in the filing commit only when the project has an Open RFI. `ClaimJob` holds it until `full_text` is done. Go's rule matches `RFI-007`, `RFI 7`, `RFI #7`, `RFI No. 7` or the sender's ref, any document kind, shown at once (amber). For correspondence only, one Jev fan-out asks Question A (`register.answers.r<n>`, one noul per RFI, at most 12, text cut to 2,400 characters around "RFI" or "response"). With no approved cut-off every Jev reading is stored blank and never shown; unresolved or failed is grey; a failed call fails the job so it retries. Runs on the always-on lane. | `internal/store/delivery_match.go`, `internal/jobs/register.go`, `worker.go`, `loop.go`, `internal/db/queries.sql(.go)` |
| 5. The screen | The idle nav slot is Register. One table, six columns, no tabs; Kind filter (only kinds with rows, "· N to check"), System filter in building order with the also-open line, "Showing … · Clear"; select, F2 or double-click to edit, arrows, Ctrl+A, Esc, right-click for bulk system, close, withdraw, reopen, remove. The grey row raises an RFI dated today in Sydney. Response time in the three-dot menu. A strip under the table: question, ref, raised date, answer, documents, sources, status history, and each possible answer with Accept or Dismiss in the amber "to check" state. Every write is `POST /ops` (area `delivery`), so each has one Undo. Registered with `chatView.ts` as `register`. "Raise RFI from this document" on the document register (Go only). Empty: "Nothing is recorded as owed yet." | `web/src/Delivery.tsx`, `deliveryApi.ts`, `register.css`, `LeftNav.tsx`, `Project.tsx`, `Register.tsx`, `chatView.ts`, `Chat.tsx` |
| 6. Operations, chat, key | See section 3. `reports.Attention` gains `register.rfi_overdue` and `register.to_check`; `reports.RFIOverdue` is the one definition the `overdue` filter also uses. | `internal/httpapi/chat_register.go`, `ops_more.go`, `chat_view.go`, `chat_screen.go`, `chat_tools.go`, `chat_more.go`, `attention.go`, `internal/reports/attention.go`, `data/eval/chat/` |
| 7. Bench and budgets | `delivery_system_read` and `register_match` rows; the gate test; `TestRegisterBudget`, `TestRegisterMatchBudget`; the intake bench samples all four paths. The answer key's shape (`data/eval/register/answers.json`, empty) and `jobs.ProposeRegisterCutoff`. | `bench/budgets.json`, `internal/latency/gate_test.go`, `cmd/intake-bench/delivery.go`, `internal/jobs/register_eval.go` |

**Migration 066** widens `project_delivery_items_kind_check` and
`_status_check` (named, so plan 6 drops and re-adds them with
`condition` and its statuses), adds `number`, `system_id`,
`determinant_id` and `delivery_numbered`, the indexes
`delivery_number_uq` and `delivery_system_idx`, the tables
`delivery_settings` and `delivery_readings`, the `delivery` area on
`project_changes`, and capture triggers on all three tables. The
foreign-key sweep inventory gained the three new composite keys.

## 2. Speed

Development machine, private schema, stand-ins; thirty samples per path
(sixty writes), each more than five calls on a connection.

| Budget row | Budget p50 / p90 | Measured p50 / p90 | Held by |
|---|---|---|---|
| `delivery_read`, 1,000 rows, 100 with readings | 100 / 250 ms | 27 / 33 ms | `TestRegisterBudget` |
| `delivery_system_read`, the same with the also-open line | 150 / 300 ms | 26 / 29 ms | `TestRegisterBudget` |
| `delivery_write`, add and edit through `POST /ops` | 100 / 250 ms | 11 / 29 ms | `TestRegisterBudget` |
| `register_match`, rule step, 40 Open RFIs | 2,000 / 4,000 ms | 12 / 14 ms | `TestRegisterMatchBudget` |
| `whole_intake` | 1,000 / 2,000 ms | not re-measured | the job is one `INSERT … WHERE EXISTS` inside the filing commit |

`register_match` with the Jev step was not measured: that is one
background fan-out, and no live call was made. The intake bench itself
was not run over the private corpus; its delivery step compiles and
samples the four paths.

## 3. Chat and the register

| | From chat |
|---|---|
| Reads | `read(delivery)` with `part` = a kind, a top-level system id (adds the also-open line) or `overdue`. Each row: kind, No., item, With, due, working days late, status, system, possible answers by reading id. Possible answers are document titles, so a read that prints them marks the turn as having read outside text. |
| At once, with Undo | `delivery.add` (an RFI saved as Draft; question, ref, raised_on as their own fields; a raised date inside `details` is refused so the figure check sees it), `delivery.edit` (close, withdraw, reopen, With, Due, System, wording, `dismiss_readings`; an item may be named `RFI-004`), `delivery.settings` (the response time; a number in the PM's words). |
| Waits for a click | `delivery.accept`: the card names the document and the RFI. `accept_reading` sent through `delivery.edit` is refused unless the screen sent it. |
| Opened filled in | Send is plan 1's and is not built: chat says the fixed words and the turn is logged as declined (`s23`). |
| Screen | The `view` tool's `register` table: select, only, clear, open; named filters `overdue`, `with_me` (answered, and the PM's own drafts), `to_check`. "Open the register" is an exact phrase with no model call. |

**The key.** `register` group, eight draft cases (`rg1` to `rg8`): two
reads, raising an RFI with no invented date, closing one by number,
setting the system on a row selected by hand, the overdue filter,
accepting a possible answer (prepared), the response time. The house
fixture gained two RFIs and a structural sketch that cites one, matched by
Go's rule when seeded. `TestScreenCasesScoredOffline` scores all eight
with the stand-in model; all pass, as do the 23 screen cases. The two
reads are scored on Go's half only (nothing changed, no figure of the
model's own); their words are the live run's to judge. `s23` used to ask
for the RFI table, which now exists; it now asks to send an RFI.

## 4. What was not measured

- No live Jev call. Question A's accuracy is unmeasured; the answer key
  is empty and its picks stay hidden. No cut-off is proposed.
- No live model run of the `register` group.
- No owner review of the key, the fixture rows, the screen or the
  wording.
- The full `tools/check.ps1` gate was not run (shared schema, private
  corpora); the intake bench was not run over the private corpus.
- The screen was checked by Playwright only, not by eye in a browser.

## 5. Where the build differs from the plan

- **The response time is a small table, not `projects.rfi_response_days`.**
  The plan has setting it apply at once with Undo; a capture trigger on
  `projects` would capture every project write, so `delivery_settings`
  (one row per project) carries it.
- **Readings carry a `basis`** ("cites RFI-007"), so the PM sees why a
  rule matched.
- **Accepting is `accept_reading` on the edit**, with status, the
  document and the date set in Go, rather than three fields the screen
  sends. It is the same edit route; the values cannot come from the model.
- **The date answered is the document's upload date** (Sydney). A filed
  date separate from the upload date was not found on the document.
- **Late is in working days** on the project's calendar, and the overdue
  filter and the attention rule both read it, so a due date on a Friday
  is not late on the Saturday.
- **Candidates are Open RFIs only** (not Draft): a draft has not been
  sent, so nothing can answer it. The job is queued only when an Open RFI
  exists.
- **A notice cannot name a message yet.** Messages are plan 1's; a
  `message_id` is refused until there is a table to check it against.
- **The trade package's system** is used only when its `builds` fall in
  one top-level system; otherwise the PM picks.
- **"With me"** is answered RFIs and the drafts the signed-in PM raised.
- **The key's ids** are `rg1`–`rg8`; `g1`–`g6` were already Program's.

## 6. Known limits and what waits

- **Plan 1 (correspondence):** Send, the thread link for the rule step,
  "Add to register" on a message, and notices from messages.
- **Plan 6 (conditions):** adds `condition` to the two checks (and to
  `delivery_numbered` if conditions carry numbers), its evidence nouls to
  the same `register` job and request, and `evidence` to
  `delivery_readings.field`.
- **Plan 5 (report):** the report item in the register's toolbar.
- **The owner:** the Question A key (60 documents or more), an approved
  cut-off in `data/register/thresholds.json`, and a look at the screen.
- Main Works exclusions are one count, "not sorted by system". Child
  systems are not offered. An RFI's documents open in the centre viewer
  only when they are in the project's document list.

# The register and RFIs on the building: implementation plan

Date 17 November 2026; revised 18 November. Lane A (records a user trusts:
due dates, answer links, org isolation). Status: built 18 November 2026 on
branch `wave/register` (Tasks 1 to 7 and "From chat"; migration 066), not
merged. What was built, measured and left open, and where the build
differs from this text, is in `docs/evidence/2026-11-18-register-rfis.md`.
The answer key for Question A is empty until the owner supplies it.
Moved 20 November 2026 (owner-approved wireframe, branch
`wave/register-panel`, not merged): the register is the right-hand panel's
Register side, off the left nav; a row opens as a sheet in the middle
panel; an RFI's To, Cc and Subject are saved and its Send is on the sheet;
its history reads the RFI's mail thread with the register. See
`docs/evidence/2026-11-20-register-panel.md`.

**Revised 18 November** after the review in `2026-11-17-next-wave-index.md`.
The register is one table with no kind tabs. "With" replaces "To". Review
comments (sheet import, PDF markups, a Jev call per comment) are deferred.
A notice confirmed from a message is a row. Chat reaches everything
("From chat"). The file keeps its name so links hold. The earlier text is
at git ref `refs/backup/next-wave-plans-2026-11-18`.

## In one page (product requirements)

**Outcome.** The PM opens Register and sees one table of what is owed on
the project: RFIs, notices and consent conditions (plan 6), each with who
holds it now, when it is due and the part of the building it is about. They
type an RFI in the grey new row. SiteWise numbers it (RFI-007), fills
"With" from the package's appointed firm, and works out the response due
date. When the consultant's reply is filed (dropped, or emailed in under
plan 1), the RFI row shows "Possible answer: <document>". One click accepts
it: Answered, linked to that document, with Undo. Choosing "Envelope" in
the System filter shows what is open on the façade, plus a line saying how
many specification unknowns and tender exclusions sit on the same system,
with links.

**Why.** Time-to-decision and answer trust. Procore and Aconex file an RFI
against a document. SiteWise files it against the part of the building it
is about, beside the specification and the tenders for that part. Reply
matching saves the PM opening every email to tick off the register.

**Where it lives.** The left nav's idle placeholder
(`web/src/LeftNav.tsx:38`, id `plan`, labelled "Report", disabled, marked
Planned) becomes **Register**. `web/src/Delivery.tsx` is rewritten as its
one table. No other new screen. No tabs.

**One table.** Every kind shares six columns:

| Column | RFI | Notice | Condition (plan 6) |
|---|---|---|---|
| No. | RFI-007 | — | 46 |
| Item | The subject | The message's subject | The printed heading |
| With | Who holds it now | Who must act | Who acts |
| Due | The response date | The date the PM typed | The gate ("Before CC") |
| Status | Draft, Open, Answered, Closed | Open, Closed | Open, Evidence filed, Satisfied |
| System | The building system | — | The building system |

A Kind filter lists only the kinds that have rows. The default view is
what is open. A detail strip under the table carries what is particular to
a kind: an RFI's question, answer and documents; a notice's message; a
condition's words. Actions, risks, issues, decisions and approvals stay as
kinds and show when they have rows. Activities and milestones are
Program's and are not listed here.

**"With" says who holds it now.** It is Procore's ball in court in one
column. While an RFI is Draft or Open it shows the firm it went to. While
it is Answered it shows "You", because the next move (pass the answer on,
or close it) is the PM's. The rule is Go's, computed at read; the PM can
type over the firm.

**What reads what.**

| Step | Who |
|---|---|
| Numbers, response due, days overdue, the "With" firm, the also-open line | Go. Dates use the Program's working calendar. |
| The system on a new row | Go, from its package or work item; else the PM's pick. No Jev. |
| The reply cites "RFI-007", "RFI 7" or the sender's ref, or plan 1 threads it under the message that sent the RFI | Go rule. Any document kind. Shows at once. |
| Which Open RFIs a correspondence document answers | Jev, one background fan-out per document (Question A), shared with plan 6's "which open condition does this evidence": both question sets ride one request on the one document state |
| RFI wording drafted from a short note | The LLM, in chat only (decision 5) |

*Question A, `register.answers.r<n>`.* A noul (a yes/no question that
returns a probability) for each candidate RFI, all in one request.

- **State:** `document` (title, kind, discipline, sender text, and up to
  2,400 characters that Go cuts around "RFI" or "response") and
  `rfis.r1…r12` (number, ref, subject, the question's first 400 characters,
  who it went to).
- **Instruction:** "Using `document` and `rfis.r<n>`, does the document
  reply to that request, giving the information, decision or instruction it
  asks for?"
- **True:** the document states the answer, or says it is attached.
  **False:** it is about something else, only mentions the topic, asks its
  own question, or is the request itself.
- **Candidates (chosen by Go):** Open status, the same discipline's package
  first, then the most recent; at most 12, minus any the rule matched.
- **Why one noul per RFI:** one letter often answers several, and TypeSafe
  says to "add one Noul per option when several can apply"
  (https://docs.typesafe.ai/patterns/companion-noul). It also avoids a
  Choice's lean towards its first option. Go caps the state because
  "accuracy falls as the state grows"
  (https://docs.typesafe.ai/model-jaggedness/jev-1.13). It is one request
  (https://docs.typesafe.ai/patterns/fan-out), and Jev compares no dates.
- **Answer key:** `data/eval/register/answers.json` (new), owner-reviewed:
  at least 60 filed documents with their Open RFIs and which ones each
  answers, negatives included. Sources: the private corpus RFI threads
  (`data/eval/intake/private/folder-batches/034-camp-y-traffic/`) and the
  owner's projects. The threshold is on the probability of the action
  taken (https://docs.typesafe.ai/confidence). Until it is approved,
  suggestions are stored but not shown.

**Speed budgets** (`bench/budgets.json`):

| Path | p50 / p90 ms | Row |
|---|---|---|
| Register read, 1,000 rows with readings | 100 / 250 | `delivery_read` (reuse) |
| Create, edit, accept, dismiss, the response time | 100 / 250 | `delivery_write` (reuse) |
| A screen batch (bulk close, set system) and its Undo | 100 / 250 | `ops_apply`, `change_undo` (reuse) |
| Read with a System filter and the also-open line | 150 / 300 | `delivery_system_read` (new) |
| Document filed → possible answer written (background) | 2,000 / 4,000 | `register_match` (new; shared with plan 6, first to land adds it) |
| Filing | 1,000 / 2,000 | `whole_intake` (unchanged; the job is queued inside the commit) |

**Done means.** (1) An RFI typed in the new row gets the next number and a
due date five working days on (weekends and Program holidays skipped)
within `delivery_write`. (2) A reply email citing "RFI-007" files; within
about 3 s RFI-007 shows a possible answer; Accept makes it Answered and
With reads "You"; Undo puts it back. (3) System = Envelope lists what is
open there with the specification and exclusion counts. (4) A project with
RFIs only shows no Kind filter entries for kinds it does not have. (5) New
bench rows pass; `whole_intake`, `TestWriteRoutesHaveOperations`,
`TestOnlyChatReachesTheModel` and the store isolation tests still pass;
`PRODUCT.md` and AGENTS.md say the register is back and that its reads and
writes call neither Jev nor the LLM. (6) The chat cases in "From chat" pass.

**Out of scope.** Sending an RFI (plan 1; chat opens Send filled in). A
guest answering (a consultant replies to the email; plan 1 threads it).
Review comments: comment sheets, PDF markups and a system pick per comment
are deferred until RFIs and conditions have run on two projects. Labelling
each Main Works exclusion with a system. Child systems (top-level only). An
RFI PDF form. Turning a builder's RFI document into a row by Jev: the PM
uses "Raise RFI from this document", which is Go only. Contract time bars
on a notice (the PM types its date).

**Owner decisions needed** (recommended default first). The plan takes its
defaults for 2 to 7; decision 1 is the index's decision 2.

1. **Register home and name:** switch on the idle nav slot as "Register",
   one table with no kind tabs. Alternatives: the name "Project Plan"
   (AGENTS.md and PRODUCT.md use it for the placeholder); or a "Register"
   tab on Program, with no nav change.
2. **RFI response time:** five working days, shown blue (typical) until the
   PM sets the project's own in the register's three-dot menu or in chat.
3. **Working days:** the Program's calendar; Monday to Friday when there is
   no program, and the cell says so.
4. **Jev answer suggestions** stay hidden until the key and cut-off are
   approved. Rule matches show from day one.
5. **LLM drafting: chat only, no register button.** Jev cannot draft ("Jev
   is not trained to generate text", jev-1.13 page). Chat's `delivery.add`
   takes the drafted wording and saves it as Draft, so nothing leaves until
   the PM sends it. No screen read or write reaches the model.
6. **Numbering:** RFI-001 per project, never reused. The sender's number
   shows as "Ref". Notices carry no number.
7. **Answered stays open** until the PM closes it; the PM can reopen it.

## What we reuse

| Need | Existing thing | File |
|---|---|---|
| Rows, kinds, links, provenance, version | `project_delivery_items` | migration `019_delivery.sql`, `internal/delivery/items.go:46` |
| Create, edit, retire, conflicts | `CreateDelivery`, `PatchDelivery` | `internal/store/delivery.go:62`, `delivery_edits.go:19` |
| Referenced documents with page and excerpt | `source_refs` → `resolveScopeSources` | `internal/store/package_scope.go:221` |
| Open versus closed | `delivery.Open`; `deliveryOpenSQL` (the new open statuses are not in its closed list, so it is unchanged) | `items.go:73`, `internal/store/delivery_references.go` |
| Routes and chat operations | GET/POST/PATCH `/delivery`; `delivery.add`, `delivery.edit`, `delivery.link`, `delivery.verify` | `internal/httpapi/delivery.go`, `ops_more.go:349` |
| One Undo per screen action | `POST /ops` and the change capture trigger | `internal/httpapi/ops.go:167`, `internal/store/change_undo.go:23`, `063_chat_capture.sql` |
| Implications feeding the register | Accepted approvals and hold points already land as rows | `internal/store/delivery_accept.go:36` |
| A post-filing job queued in the commit | `queueVisual`; the always-on lane | `internal/store/visual_identity.go:90`, `filing.go:296`, `internal/jobs/loop.go:19` |
| Working days | `program.NewCalendar(...).DateOf(n)` | `internal/program/calendar.go:142`, `:297` |
| Today; the "With" firm | `todaySydney()`; `AppointedFirms` | `internal/httpapi/invoices.go:237`, `internal/store/appointed.go:22` |
| Trade → systems | `builds:` via `Catalog.Trade` | `knowledge/works/trades.yaml`, `internal/knowledge/trades.go:38` |
| Specification unknowns by system | Rows carry `SystemID` and `Inclusion` | `internal/specifications/schedule.go:76` |
| Systems in building order | The profile's groups, as Implications uses them | `web/src/Implications.tsx:39` |
| Table conventions | Selection, inline editor, row menu, states | `web/src/table/selection.ts`, `table/InlineCellEditor.tsx`, `table/RowMenu.tsx`, `states.css`, `cost-tables.css`; template `Accommodation.tsx` |
| Chat moving the screen | plan 0's `view` tool and `chatView.ts` registry | `2026-11-18-chat-agency.md`, Task 2 |
| Bench and browser test | Delivery bench and spec | `cmd/intake-bench/delivery.go`, `web/tests/delivery.spec.ts` |

## What is new

| Thing | Why an existing one cannot carry it | Est. |
|---|---|---|
| Kinds `rfi`, `notice`; statuses `draft`, `answered`; columns `number`, `system_id`, `determinant_id` | Not a table: the register holds every field except a number and the building link. | 30 SQL |
| Table `delivery_readings` | `proposals` (migration 016) is keyed to knowledge records (`ic/cq/uc`) and a rule fingerprint. `program_readings` belongs to a program. A suggestion kept in a row's `details` cannot be removed by cascade when its source document is deleted. | 15 SQL |
| `projects.rfi_response_days` | One project setting; there is no project settings table. | 1 SQL |
| `PUT /projects/{id}/delivery/settings` | Sets the response time; `PATCH /projects/{id}` is rename. An operation (`delivery.settings`), so chat can set it. | 25 Go |
| Job kind `register` (shared with plan 6) | The post-filing reading for this register. Jobs are per document, which fits. | 150 Go |
| New screens | None (the nav slot is decision 1) | — |

## Data (migration sketch; take the next free number, the wave shares it)

```sql
-- Check the generated constraint names with \d first. Plan 6 adds 'condition' to the same check.
ALTER TABLE project_delivery_items DROP CONSTRAINT project_delivery_items_kind_check,
 ADD CONSTRAINT project_delivery_items_kind_check CHECK (kind IN ('activity','milestone','action','risk','issue','decision','approval','rfi','notice'));
ALTER TABLE project_delivery_items DROP CONSTRAINT project_delivery_items_status_check,
 ADD CONSTRAINT project_delivery_items_status_check CHECK (status IN (<the 18 existing>,'draft','answered'));
ALTER TABLE project_delivery_items ADD COLUMN number integer CHECK (number > 0),
 ADD COLUMN system_id text CHECK (length(system_id) BETWEEN 1 AND 200),
 ADD COLUMN determinant_id text CHECK (length(determinant_id) BETWEEN 1 AND 200),
 ADD CONSTRAINT delivery_numbered CHECK ((number IS NOT NULL) = (kind = 'rfi'));
CREATE UNIQUE INDEX delivery_number_uq ON project_delivery_items(org_id,project_id,kind,number) WHERE number IS NOT NULL;
CREATE INDEX delivery_system_idx ON project_delivery_items(org_id,project_id,system_id) WHERE system_id IS NOT NULL;
ALTER TABLE projects ADD COLUMN rfi_response_days smallint CHECK (rfi_response_days BETWEEN 1 AND 60);
CREATE TABLE delivery_readings (
 org_id uuid NOT NULL REFERENCES orgs(id) ON DELETE CASCADE,
 id uuid NOT NULL, project_id uuid NOT NULL, item_id uuid NOT NULL, document_id uuid NOT NULL,
 field text NOT NULL CHECK (field IN ('answers')),  -- plan 6 adds 'evidence'
 decided_by text NOT NULL CHECK (decided_by IN ('rule','jev')),
 probability double precision, band text NOT NULL CHECK (band IN ('amber','blank','grey')),
 question_version text NOT NULL DEFAULT '',
 status text NOT NULL DEFAULT 'proposed' CHECK (status IN ('proposed','accepted','dismissed')),
 created_at timestamptz NOT NULL DEFAULT now(),
 PRIMARY KEY (org_id,id), UNIQUE (org_id,item_id,document_id,field),
 FOREIGN KEY (org_id,project_id,item_id) REFERENCES project_delivery_items(org_id,project_id,id) ON DELETE CASCADE,
 FOREIGN KEY (org_id,document_id) REFERENCES documents(org_id,id) ON DELETE CASCADE);
CREATE TRIGGER capture_change AFTER INSERT OR UPDATE OR DELETE ON project_delivery_items FOR EACH ROW EXECUTE FUNCTION capture_change('version');
CREATE TRIGGER capture_change AFTER INSERT OR UPDATE OR DELETE ON delivery_readings FOR EACH ROW EXECUTE FUNCTION capture_change();
```

How the existing columns carry the new kinds:

- **RFI.** `title` is the subject. `owner_text` and `package_id` say who it
  went to. `target_date` is the response due date; `actual_date` is the
  date answered. Details: `question`, `ref`, `raised_by`, `raised_on`,
  `answer`, `answer_documents`.
- **Notice.** `title` is the message's subject. `owner_text` is who must
  act. `target_date` is the date the PM typed. Details: `message_id` (plan
  1's message), `notice` (the kind of notice, from plan 1's options),
  `received_on`.

## Tasks

**Task 1. Kinds, statuses, details** (~100 Go). Files:
`internal/delivery/items.go`, `patch.go`, `items_test.go`. RFI statuses:
draft, open, answered, closed, withdrawn; it stays open while draft, open
or answered. Notice statuses: open, closed, withdrawn. Add `RFIDetails` and
`NoticeDetails` to `validateDetails`, with `DisallowUnknownFields` and
bounds. A kind change to or from `rfi` is refused, because numbers belong
to one kind. Add `SystemID` and `DeterminantID` to `Content` and `Patch`.
`With(item)` returns "You" for an answered RFI, else `owner_text`.
`DueDate(cal, raisedOn, days)` is a one-line call to the shared
`program.BusinessDayAfter` (index, agreement 1; whichever of this plan and
plan 3 lands first writes it), or nil when an input is unknown. **Test:**
status and `Open` tables; unknown fields rejected; a kind change refused;
`With` for each status; due dates for a Friday raise, Christmas as a
Program holiday, no program, and no raised date.

**Task 2. Migration and store** (~200 Go). Files: the migration;
`internal/store/delivery.go`, `delivery_edits.go`, new
`delivery_readings.go`, `change_undo.go`.

- `CreateDelivery` numbers rfi rows `max+1` under `lockProject`. It sets
  `target_date` from `raised_on` plus the response days unless the user
  typed one, and fills an empty `owner_text` from `AppointedFirms`. It sets
  `system_id` from the linked work item's system, or from a trade package
  whose `builds` fall in one top-level system. It checks that system,
  determinant, document and message ids belong to this org, this project
  and the catalogue.
- `ReadDelivery` leaves out activities and milestones. It adds the proposed
  readings (rule and amber only), `with`, and `late_days` for Open RFIs,
  with today from `todaySydney()`. With a system named, it adds the
  also-open line: schedule rows under that system with an unknown inclusion
  or field, and the `tender_exclusions` of trade packages that build under
  it. Main Works exclusions are one count, "not sorted by system".
- **Accepting is an ordinary edit.** An edit that sets a proposed reading's
  value (answered status, `answer_documents`, `actual_date` = the
  document's filed date, else its upload date) marks that reading accepted
  in the same transaction. `dismiss_readings` on a patch leaves the row's
  version alone.
- Add `captured` entries for area `delivery`, and an after-undo touch that
  bumps the delivery revision and sends the event.

**Test** (`internal/store/delivery_test.go`): 20 concurrent creates number
1 to 20. Another org's document id gives `ErrNotFound`. A typed due date
survives a change to the raised date. An edit accepts the matching reading.
A dismissal keeps the version. An ops batch of three edits is undone as one
change. A deleted document's readings go. A notice with a message source
reads back with its message.

**Task 3. Routes and speed paths** (~40 Go). Files:
`internal/httpapi/delivery.go`, `projects.go`, `speed.go`.
`GET /delivery?system=&kind=` maps to `delivery_system_read` when a system
is named. `PUT /delivery/settings` maps to `delivery_write`. **Test:** a
wrong-org request gets 404.

**Task 4. The post-filing job** (~150 Go). Files:
`internal/store/filing.go`, new `internal/jobs/register.go`, `worker.go`,
`loop.go`.

- `queueRegister`, next to `queueVisual`, inserts the job only
  `WHERE EXISTS` an Open RFI (or, with plan 6, an open condition) in the
  project. Plan 6 uses this same job and queue point for its evidence
  step (its separate `consent` read is queued after `full_text`).
  Whichever lands first creates the job kind. The job must find the
  document's text; if `full_text` has not finished, it waits for it rather
  than reading an empty text.
- The job reads the text with the worker's `Text`. The rule step matches
  each Open RFI's number (`RFI[-\s#]*0*7`), its `ref`, or plan 1's thread
  link. The Jev step asks Question A, for correspondence only, on the RFIs
  the rule left, in the same request as plan 6's condition nouls.
- Bands come from `data/register/thresholds.json`, which is absent until
  the owner approves one.
- The job kind goes into the always-on lane.
- Email bodies (plan 1) file as correspondence and take this path
  unchanged.

**Test** (fake asker): a drawing citing "RFI-007" gives a rule reading. An
uncited letter sends one request with at most 12 nouls. A grey answer is
stored grey. A project with no Open RFI queues no job. The filing commit's
timing test is unchanged.

**Task 5. The register screen** (~320 TSX, replacing 135). Files:
`web/src/Delivery.tsx` (rewritten), `deliveryApi.ts`, `LeftNav.tsx` (enable
`plan` as Register), `Project.tsx` (view wiring), `Register.tsx` (one
row-menu entry on the document register: "Raise RFI from this document"),
`web/tests/delivery.spec.ts`. Read `web/DESIGN.md` first.

- One table, the six columns above, for every kind. No tab strip.
- A Kind filter listing only kinds with rows, with "· N to check" beside a
  kind that has proposals; a System filter in building order, with the
  also-open line under it. Both use the "Showing … · Clear" line the
  repository uses.
- The table follows `Accommodation.tsx`: select, F2 to edit in place, and
  right-click for bulk set system, status, close or withdraw.
- A grey new row adds an RFI. The response time is in the three-dot menu.
- A detail strip shows the selected row's question, answer, documents
  (Open shows them in the centre viewer), sources, status history, and
  Accept or Dismiss on each proposal, in the amber "to check" state.
- Every write goes through `POST /ops` (area `delivery`), so each action
  has one Undo.
- The table registers with plan 0's `chatView.ts` (`select`, `only`,
  `open`).

**Test:** Playwright covers create, number and due date, accepting a
seeded reading, With reading "You", and Undo; keyboard selection; a 409
retry; a project with one kind shows no other kind.

**Task 6. Operations, chat reads and the key** (~60 Go). See "From chat".
Files: `internal/httpapi/ops_more.go`, `ops_routes.go`, `chat_more.go`,
`data/eval/chat/gen_cases.py`. **Test:** `TestWriteRoutesHaveOperations`
passes with no route of this plan pending; the six key cases.

**Task 7. Bench, budgets, answer key** (~90 Go). Files:
`cmd/intake-bench/delivery.go` (RFI create and edit, a 1,000-row read, a
system read), `bench/budgets.json` and `internal/latency/gate_test.go` (the
two new rows), and `data/eval/register/answers.json` with a scorer run
against live Jev. The scorer proposes an amber cut-off for the owner.

**Totals:** about 640 Go, 320 TSX and 45 SQL, tests extra. One new route,
one new table, no new screen.

## From chat

Plan 0's rule table governs. Every write route here is an operation.

- **Reads.** `read(delivery)` prints each row's kind, number, item, with,
  due, late days, status, system label and possible answers; it takes a
  kind, a system or "overdue". With a system named, it adds the also-open
  line.
- **At once, with Undo** (the capture trigger in this plan's migration, so
  these stop being prepared): `delivery.add`, `delivery.edit` (close,
  withdraw, reopen; set With, Due and System; `dismiss_readings`),
  `delivery.settings` (the response time). `raised_on`, Due and the
  response time are `Figures`: a date or a number of days must be in the
  PM's words.
- **Waits for a click.** Accepting a possible answer: the PM vouches that
  the document answers the RFI.
- **Drafted in chat.** An RFI's wording from a short note, saved as Draft.
- **Opened filled in.** Send (plan 1), with the RFI and the documents it
  cites. Before plan 1 lands chat says so and the turn is logged as
  declined.
- **Screen.** Filter by kind, system and who holds it; select rows; open a
  row's detail.
- **Named filters.** Overdue; with me (Answered, and my Drafts); to check.
- **Key cases.** "What is open on the façade"; "what is with the
  structural engineer and how late"; "raise an RFI to the structural
  engineer about the basement wall waterproofing" (Draft, no invented
  date); "close RFI-004"; "set these to Envelope" (rows selected by hand);
  "accept the answer on RFI-007" (prepared).

## Edge cases

| Case | What happens |
|---|---|
| The answering document is deleted | Its readings go (cascade). An accepted link reads "Document removed". The RFI stays Answered. |
| One letter answers RFIs 7, 8 and 12 | One reading per RFI, each accepted on its own row. |
| Two documents answer one RFI | Both are offered; the second accepted joins `answer_documents`. |
| The filed document is the RFI itself | Excluded: it is already one of that RFI's sources. |
| The reply quotes the builder's number | The rule matches `ref` as well as ours. |
| Jev times out | The reading is stored grey and not shown; the job retries within its attempts. |
| No Open RFIs | No job; filing is unchanged. |
| Two RFIs are created at once | The project lock orders them; the unique index backs it. |
| The PM typed the due date | It is kept when the raised date changes. No raised date: Due stays blank. |
| A notice's message is discarded or its project address rotated | The row stays, with "message removed" where the link was. |
| An id from another org, or an unknown system | 404 or 422; nothing is written. |
| Chat drafts an RFI | It is saved as Draft; dates must be in the user's message (`Figures`). |
| A Main Works tender lists 40 exclusions | One count per system, "not sorted by system", linked to the tender. |
| A project with no rows of any kind | One line: "Nothing is recorded as owed yet", and the grey new row. |

## Risks and sibling conflicts

- **No answer key at launch.** Jev suggestions stay hidden until there are
  approved cut-offs. Day one rests on the cited-number rule, Go links and
  the PM's picks.
- **Plan 6, same check and same job.** Plan 6 adds `condition` and three
  statuses to the same two checks; the second migration must list both
  plans' values. Both plans read a filed document for open register items:
  one job kind and one request, and one bench row (`register_match`).
- **Plan 5.** The project report opens from this register's toolbar; plan 5
  adds the item.
- **Plan 1.** Its reply threading feeds this plan's rule step. Its "Add to
  register" button on a message calls `delivery.add` with the message as
  source. Before plan 1 lands there are no notices and the kind never
  shows.
- **One table for unlike things.** If a kind later needs a column the
  others cannot share, it goes in the detail strip first. A seventh column
  needs a reason stronger than one kind's convenience.
- **Background load.** Email intake raises the volume of correspondence. Jev
  is asked only about correspondence, at most 12 nouls each, in a lane that
  cannot take filing's reserved slots.
- **Old rows.** Existing action, risk, issue, decision and approval rows
  appear when the register returns. Old activity and milestone rows do not.
- **The also-open line's cost.** If it misses `delivery_system_read`, it
  becomes counts with links, computed only when a system is chosen.

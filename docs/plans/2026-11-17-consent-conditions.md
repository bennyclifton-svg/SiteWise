# Consent conditions register: implementation plan

Date 17 November 2026; revised 18 November. Lane A (filing, rule numbers from the instrument,
hold points a PM trusts). Status: built 18 November 2026 on branch
`wave/conditions` (not merged, not pushed); every task done, with the
deviations in `docs/evidence/2026-11-18-consent-conditions.md`. Accuracy
on real notices is not measured: the owner's key is not yet written.

**Revised 18 November** after the review in `2026-11-17-next-wave-index.md`.
The scope stands. Conditions are a kind in the one register table, not a
tab. This plan builds the read route (plan 4's comment sheets are
deferred). Chat reaches everything ("From chat"). The earlier text is at
git ref `refs/backup/next-wave-plans-2026-11-18`.

## In one page (product requirements)

**Outcome.** The PM drops a notice of determination into the repository as
today. It files in about a second (Statutory Instrument, Authorities). A few
seconds later the shared register shows **Consent conditions**: every
numbered condition, grouped by when it must be met, with its number, printed
heading, the council's own words and a link to the page. Each row shows who
acts (discipline, and package when there is one), the building system,
whether evidence is needed, and a status: open, evidence filed, satisfied,
not applicable. What SiteWise read shows as "to check" until the PM edits
it. A later document that looks like a condition's evidence shows on that
row as "Possible evidence" with a Link button. Program offers one hold point
per gate ("Consent conditions before the construction certificate") through
its existing review. A condition given to a consultant goes onto their RFP as
a scope line in one click.

**Why.** Time-to-decision and answer trust. A missed condition costs a
construction certificate, a start or an occupation certificate. Today the PM
retypes 40 to 120 conditions into a spreadsheet. SiteWise reads them once,
keeps the council's words, and ties each to a package, a system and a
Program gate. Filing speed is untouched: the read runs after filing.

**Where it lives. New screens: none.**
- Register: conditions are a kind in the one table plan 4
  (`2026-11-17-rfis-review-comments.md`) builds from `web/src/Delivery.tsx`.
  There is no tab. The Kind filter shows "Conditions" once a notice is read;
  rows group by gate; the shared columns carry them: No. (46), Item (the
  printed heading), With (who acts), Due (the gate: "Before CC"), Status,
  System. The council's words, the page link and the evidence sit in the
  detail strip.
- Program: gate hold point rows via the existing review (Adopt or Keep).
  Release stays the existing click.
- RFP: an existing custom scope line (`custom:<stage>` setting).
- Profile: consent number, date and lapse date are already read
  (`knowledge/profile/project_facts.yaml`); the register header shows them.

**What reads what.**
- Filing: unchanged, one filing fan-out as today.
- Go: spots a notice; splits conditions by numbering and stage headings;
  maps a heading to a stage; reads the consent reference and determination
  date; lists gaps in the numbering; matches a modification to the
  conditions it changes; finds the package for a discipline; places Program
  gates; counts what is open; writes the RFP line from a fixed template.
- Jev: one background fan-out per condition (below). When a later document
  files, plan 4's one post-filing request: a Go rule first (the document
  cites "Condition 13"), then one Noul per shortlisted open condition, riding
  in the same request as plan 4's RFI Nouls (task 8).
- LLM: none.

**Jev questions, one fan-out per condition** (`consent-read-1`, background).
State: printed stage heading, number, heading, and the first 3,500
characters of the text, as the specification import does
(`internal/specifications/import.go:160`). Pattern:
[speculative fan-out](https://docs.typesafe.ai/patterns/fan-out), all
questions on one state in one request. Code splits, Jev picks
([how to build](https://docs.typesafe.ai/concepts/how-to-build-with-system-one):
"Keep deterministic work in code"). Instructions say the condition is
evidence, not an instruction.

| Id | Type | Asked when | Options |
|---|---|---|---|
| `timing` | Choice | Only when the heading did not settle it ("General", none, unknown wording) | before the construction certificate · before work starts · during works · before occupation · before the subdivision certificate · ongoing after occupation · throughout (no single gate) |
| `party` | Choice | Always | 24: architect, structural, civil, hydraulic, electrical, mechanical, fire services, fire engineer, landscape, arborist, acoustic, traffic, geotechnical, environmental, hazardous materials, heritage, access, BCA, certifier, surveyor, BASIX, waste (ids from `data/intake/disciplines.json`), builder (`trade.main_works`), applicant (fees, levies, bonds, title steps) |
| `system` | Choice | Always | The 13 systems in `knowledge/systems.yaml`, plus `none` (not about the building: fees, hours, signs, title) |
| `evidence` | Noul | Always | true: something must be submitted, certified, paid or approved to show it is met; false: a standing rule about how work is done or the building is used |

Bands follow [confidence](https://docs.typesafe.ai/confidence) and
[routing by probability](https://docs.typesafe.ai/patterns/probability-routing).
No green and nothing auto-accepted until the key is owner-reviewed. Until
then (owner decision 9) a Choice at 0.5 or above shows amber "to check",
lower stays blank; a
Noul at 0.7 or above is yes, at 0.3 or below is no, between stays blank
(TypeSafe's starting points); a timeout is grey. Options sit in a fixed
order because Jev "leans toward the option that comes first"
([jev-1.13](https://docs.typesafe.ai/model-jaggedness/jev-1.13)); the key
measures that. All option lists are far under the 255 limit
([API](https://docs.typesafe.ai/api)).

**Speed budgets.**

| Path | p50 / p90 | Bench row |
|---|---|---|
| Register read with 300 condition rows | 100 / 250 ms | `delivery_read` (existing; bench seeds 300 conditions) |
| Edit, accept, link evidence, satisfy, add by hand | 100 / 250 ms | `delivery_write` (existing) |
| "Read conditions" request (queues the job) | 300 / 1,000 ms | new `delivery_import` (this plan's route; the `report_assemble` numbers) |
| Background read, 120-condition notice, job lease to last row | 5,000 / 10,000 ms | new `consent_read` (excludes queue wait, as `visual_identity` does) |
| Go split, 25-page 102-condition notice | 20 / 50 ms | `TestSplitBudget` in `internal/consent` |
| Evidence suggestion after a document files | 2,000 / 4,000 ms | `register_match`, plan 4's row, shared (first to land adds it) |
| Program read and generate with gates | 100 / 250, 300 / 800 ms | `program_read`, `program_generate` (existing) |
| Filing | 1,000 / 2,000 ms | `whole_intake`, unchanged (the read is queued after `full_text`) |

Throughput: 120 calls at 16 in flight, about 0.25 to 0.3 s each with gzip
(`docs/evidence/2026-11-11-profile-update-speed.md`), is about 8 rounds,
roughly 2.5 s. These calls are smaller than a profile label call.

**Done means.**
1. The Petersham 2016 notice gives 102 rows, 1 to 102 in order, each stage
   from its heading, advisory notes left out, each row linked to its page.
   The split matches the key exactly: no merges, splits or misses.
2. The Camp Y amenities notice gives 41 rows with their printed headings.
3. The 2023 modification, dropped later, adds no duplicates: changed
   conditions are flagged "Changed by MOD/2023/0177", deleted ones proposed
   as not applicable, and the PM's edits stay as they were.
4. The scanned Bankstown consent is read from its OCR text, and missing
   numbers are listed ("Conditions 14 and 22 were not found; check the scan").
5. After Adopt, Program shows gate hold points linked before the
   construction and occupation certificate activities; release is the
   existing click.
6. Add to RFP puts one line on the package's RFP draft. A landscape plan
   filed later shows as possible evidence on the landscape condition.
7. Budgets hold. `TestWriteRoutesHaveOperations`,
   `TestOnlyChatReachesTheModel` and an org-isolation test on the read
   request pass.

**Out of scope.** A new filing kind. Conditions in other approvals (Roads
Act s138, Sydney Water, complying development certificates). Any summary or
rewording of a condition. Contributions or bonds into the Cost Plan. Due
dates on the register (Program dates the gates). Guest evidence uploads
(plan 2 has no such link). Emailing conditions out (plan 1). RFP returnables. Any status
change SiteWise makes by itself.

**Owner decisions needed** (recommended default in bold). The plan takes
its defaults for 1 to 8; the owner decides 9, which is the index's
decision 9.
1. **A new `condition` kind**, not `approval`. Approval statuses (not
   submitted … approved) describe an application; a condition is an
   obligation. The DA itself can stay an `approval` row.
2. **Read automatically after filing** when Go recognises a notice filed as
   a Statutory Instrument, with a Read conditions button for anything else.
   (Alternative: click only.)
3. **One hold point per gate, at most five rows, shown in the outline
   program too.** (Alternative: one per condition, which floods a 20 to 30
   row outline.)
4. **No hold point for "during works", "ongoing" or "throughout"
   conditions.** They are obligations, not gates.
5. **RFP: a scope line at the gate's stage** (before the certificate or
   before work starts to design finalisation, the rest to delivery; the PM
   can move it), not a returnable.
6. **Only the PM marks a condition satisfied.** Evidence never satisfies
   it; from chat it waits for the click.
7. **Version 1 covers NSW development consents and s4.55 modifications.**
8. **Money in a condition stays as the council's text in version 1.**
9. **Show condition picks amber from day one** at TypeSafe's starting
   floors, never green, because every pick sits on a proposed row (as the
   specification import does). The alternative is blank until the owner
   approves cut-offs in plan 4's `data/register/thresholds.json`. Evidence
   suggestions follow plan 4: rule matches at once, Jev ones withheld until
   approved. The owner reviews the draft key either way.

## What we reuse

| Need | Existing thing | File |
|---|---|---|
| Upload, file, OCR a scan, cache source pages | intake, `full_text`, `ocr` jobs | `internal/store/filing.go:293`, `internal/store/ocr.go`, `internal/jobs/worker.go:184` |
| Background read, one fan-out per candidate, 16 in flight | specification import job, `eachPassage` | `internal/jobs/specifications.go:14`, `internal/jobs/worker.go:31`, `:908` |
| Register rows, org-scoped, proposals, versions | `project_delivery_items` | `internal/delivery/items.go:46`, `internal/store/delivery.go:23`, `019_delivery.sql` |
| Accept by editing | any PATCH sets accepted and origin user | `internal/store/delivery_edits.go:19` |
| Add a condition by hand | `POST /projects/{id}/delivery` | `internal/httpapi/delivery.go` |
| Register screen, bulk right-click, one Undo per action | plan 4's register over `POST /ops` (area delivery) and the change log | `web/src/Delivery.tsx`, `web/src/deliveryApi.ts`, `internal/store/change.go` |
| System column and filter | plan 4's `system_id` column | plan 4, migration sketch |
| Suggestions that vanish with their document | plan 4's `delivery_readings` | plan 4, migration sketch |
| Post-filing read of a new document | plan 4's `register` job and `queueRegister` beside `queueVisual` | `internal/store/filing.go`, plan 4, task 4 |
| Hold points, adopt or keep, release click | Program generate, review, release | `internal/program/generate.go:875`, `internal/program/review.go:129`, `internal/store/program_holdpoints.go:132`, `internal/httpapi/program_review.go:146` |
| Consent date, lapse check | Profile facts, `consent_lapse` | `knowledge/profile/project_facts.yaml`, `internal/program/checks.go:100` |
| RFP scope line | `custom:<stage>` setting (300 characters, 40 a stage) | `internal/rfp/settings.go:342`, `PUT /reports/{r}/settings/{key}` |
| Chat edits, click-only changes | `delivery.edit`, `Click` operations | `internal/httpapi/ops_more.go:352` |
| Route classing | `noOperation` | `internal/httpapi/ops_routes.go:9` |
| Join a question to one fan-out | hold-point release join | `internal/jobs/holdpoint.go:88` |
| Systems, disciplines | catalogue, vocabulary | `knowledge/systems.yaml`, `data/intake/disciplines.json` |

## What is new

| Thing | Why existing cannot carry it | Est. lines |
|---|---|---|
| `condition` kind, statuses, details | Approval statuses describe an application | 60 Go, 15 SQL |
| `internal/consent`: split, call, shortlist | A parser plus Jev calls; `internal/delivery` validates records and stays Jev-free | 360 Go |
| Job kind `consent` | `jobs.kind` takes any value; no migration | 80 Go |
| Condition upsert and reconcile in the store | Modification rules are specific to consents | 220 Go |
| `POST /projects/{id}/delivery/imports` {document_id}; read status in `getDelivery` | No register route takes a document | 60 Go |
| Operations `delivery.satisfy`, `delivery.accept_evidence`, `delivery.read_notice`, `reports.add_scope_line` | Satisfied and accepting evidence are vouches, like releasing a hold point; a read is started by a click; Add to RFP is a pending "reports" route today | 70 Go |
| Program gates and one check | Interface hold points cannot name a consent | 110 Go |
| Conditions in the register: grouping by gate, the detail strip, actions | Plan 4 owns the table; this adds one kind's grouping and actions | 200 TSX |
| `cmd/consent-eval` | Split and pick scoring against the key | 200 Go (tool) |

No new table or left-nav entry; one new route. It extends plan 4's
`delivery_readings` table and `register` job; if this plan lands first, it
builds those exactly as plan 4 sketches them.

## Data

Migration at the next free number, `NNN_consent_conditions.sql` (check the
generated constraint names first). Plan 4 edits the same two checks; the
second to land rebases the lists.

```sql
ALTER TABLE project_delivery_items DROP CONSTRAINT project_delivery_items_kind_check,
 ADD CONSTRAINT project_delivery_items_kind_check CHECK (kind IN
 ('activity','milestone','action','risk','issue','decision','approval','condition'));
ALTER TABLE project_delivery_items DROP CONSTRAINT project_delivery_items_status_check,
 ADD CONSTRAINT project_delivery_items_status_check CHECK (status IN (/* existing 18 */,
 'evidence_filed','satisfied','not_applicable'));
ALTER TABLE program_changes DROP CONSTRAINT program_changes_source_check,
 ADD CONSTRAINT program_changes_source_check CHECK (source IN
 ('profile','work_items','procurement','library','consent'));
-- plan 4's table: evidence suggestions for conditions
ALTER TABLE delivery_readings DROP CONSTRAINT delivery_readings_field_check,
 ADD CONSTRAINT delivery_readings_field_check CHECK (field IN ('answers','evidence'));
```

A condition row:
- `title` "Condition 46 · <printed heading>"; with no heading, Go takes the
  opening words, cut to 120 characters.
- `status` open, evidence_filed, satisfied or not_applicable;
  `delivery.Open` is false for the last two.
- `package_id` set by Go only when exactly one live package has the picked
  discipline.
- `source_proposal_key` `consent:<consent ref or document id>:<number>`;
  the existing unique index stops duplicates, retired rows included.
- Job-written rows are `origin` document, `review_status` proposed; any PM
  edit makes them user and accepted, as today.
- `system_id`: plan 4's column, so its System filter lists conditions too.
  Plan 4's integer `number` stays null: a condition number can be "A1".
- `details` (strict decode): `consent_ref`, `number` (as printed: "46",
  "A1"), `heading`, `text` (verbatim, up to 12,000 characters), `stage`,
  `stage_heading`, `discipline`, `evidence_needed` (null is unknown),
  `evidence` [{document_id, linked_by, linked_at}], `source` {document_id,
  page}, `rfp` {report_id, stage, line_id}.
- `provenance` (not edited by the PM): `reading` {fingerprint,
  question_version, each field's value, band, decided_by}, `notices`
  [{document_id, determined_on, change}].
- Evidence suggestions are `delivery_readings` rows, field `evidence`, so
  they go when their document is deleted.

**Cached by source.** The reading and its fingerprint (version, state and
questions hashed, as `WorkspaceFingerprint` does) live in the row's
provenance; a re-read with the same fingerprint makes no call. So no cache
table: `specification_import_readings` belongs to imports, `passage_calls`
is keyed by passage.

## Answer key

Real documents stay in `D:\AI Projects\Test Data`, outside the repo. The key
goes in `data/eval/consent/private/` (a new `.gitignore` line; the repo is
public): per condition, number, page, stage, party, system, evidence needed.
No condition text.

| Document | What it tests |
|---|---|
| Petersham notice of determination, 18 Oct 2016 (`Petersham\03 ANX Q PPR\DA, MODs & STAMPED PLANS\01 DA 2016 (Original)\`): 25 pages, 102 conditions, upper-case stage headings, a "Reason:" line each, advisory notes after | Split, heading rule, "General" and "Site works", picks |
| Petersham modifications, 11 Oct 2018 and 4 Aug 2023 (`02 MOD 2018`, `04 MOD 2023`): conditions restated with "Deleted" markers | Matching a modification, deleted and changed |
| Camp Y amenities notice, Hawkesbury, 8 Nov 2021 (`Block B\03 RFT\AMENITES BLOCK 2\…\DA 0167-21 Notice of Determination.pdf`): 41 conditions with printed headings, mixed-case stage headings, "Operational Conditions" | A second council's layout, the ongoing stage |
| Bankstown consent, 31 Dec 2014 (`delivery-bankstown\05 DA Consent\`), image only, and the builder's responsibility matrix (`data/eval/intake/private/folder-batches/051-bankstown-da-matrix/`) | OCR path and gaps; the matrix is an independent human labelling of stage and who acts for about 70 conditions |
| Seven Hills synthetic notice (`docs/landing_demo_corpus/seven-hills/landing-cut/1-project-folder/Notice of determination DA-2025-0733.md`) | Public fixture, table form |
| Petersham s138 roads approval (`…\CIVIL\S138 Step 1 Approval Notice.pdf`) | Negative: not a consent, not auto-read |

An agent drafts the key; the owner reviews it; thresholds stay withheld
until then. The split must be exact on every key document before any pick is
scored.

## Tasks

**0. Key and fixtures.** Add the `.gitignore` line, draft
`data/eval/consent/private/*.yaml`, copy the Seven Hills notice into
`internal/consent/testdata/`. Proof: task 2's loader test. Lines: key files.

**1. Condition kind.** `internal/delivery/items.go` (`Statuses`, `Open`,
`ConditionDetails` in `validateDetails`), the migration, statuses in
`web/src/Delivery.tsx` and `web/src/deliveryApi.ts`. Proof: unit tests for
statuses, `Open`, unknown detail fields and over-long text refused; a store
test creates a condition via `POST /delivery`; wrong org gets 404. About 60
Go, 15 SQL, 5 TSX.

**2. Go split.** `internal/consent/split.go`. `Detect(pages)` needs a notice
marker ("notice of determination", "development consent", "determination
no.") and at least three numbered conditions. `Split(pages)` returns the
consent reference, the determination date (by rule; used only to order
notices), the conditions (number, heading, text, page, stage heading, stage,
deleted or changed marker) and the gaps. Heading words map to a stage, in
this order: "subdivision certificate"; "occupation"; "construction
certificate" with "prior" or "before"; "commenc", "prior to construction",
"prior to any works"; "during", "site works"; "operational", "ongoing".
"Deferred commencement" maps to before work starts, flagged. "General", no
heading or other wording goes to Jev; "Advisory notes" or "Notes" ends the
conditions. A number counts only as the next in its sequence (`12`, `A3`,
`B1`), so a cross-reference is not a condition; `(a)` and `i)` stay inside;
repeated footers ("Document Set ID", "Page 5 of 25") drop; a Markdown table
with a No. column reads as rows. Proof: the public fixture test; private key
tests that skip without the key (102 of 102, 41 of 41, the modification
markers); `TestSplitBudget`. About 200 Go.

**3. Jev call.** `internal/consent/read.go`: `Call(c)` builds the questions
above, leaving out `timing` when the heading settled it; `Apply` bands each
answer; the party list is checked against `intakevocab.Disciplines` at load.
Proof: no timing question when settled; `jev.CheckRequestSize` passes at
3,500 characters; stable fingerprint. `cmd/consent-eval` replays recorded
answers against the key and only reports. About 100 Go.

**4. Job and store.** `internal/jobs/consent.go` (the pattern of
`internal/jobs/specifications.go`), a `Perform` case and a queue at the end
of `fullText` in `internal/jobs/worker.go`, and `internal/store/consent.go`.
`UpsertConditions` is one transaction: a new number is a proposed row; the
same text is left alone; changed text replaces a proposed row but only flags
an accepted one; "Deleted" proposes not applicable by flag, never by
status; an older notice never overwrites a newer one; a row the PM deleted
is not brought back. It ends with `finishDeliveryWrite` (the existing
event). With plan 4's change-capture trigger on the table, the job opens one
change ("Read 102 conditions from <notice>"), so one Undo takes a read back
out. Proof: store tests for a second read making zero Jev calls (a
counting fake asker), the modification cases, an accepted edit kept, a lost
lease, org isolation. About 300 Go (80 job, 220 store).

**5. Read request and read status.** New
`POST /projects/{id}/delivery/imports` {document_id}: when `consent.Detect`
is true it queues the `consent` job; otherwise 422 "No numbered conditions
found". 404 for another org's or project's document; 409 while leased. It
is an operation, `delivery.read_notice`, `Click: true`, if the owner takes
plan 0's decision 1; otherwise it is `noOperation` ("reads a document into
rows and queues a background read; chat starts no read"). `getDelivery`
adds `consents: [{document_id, title, status, conditions, gaps, error}]`
from the `jobs` row and a count. The delivery write routes join
`programReviewRoutes`; in any case `GET /program` already kicks the review,
so the gates are fresh whenever Program opens. Budget `delivery_import`.
Proof: handler tests (wrong org, wrong project, leased, not a notice) and
`TestWriteRoutesHaveOperations`. About 60 Go.

**6. Operations, chat reads and the key.** See "From chat".
`internal/httpapi/ops_more.go`: `delivery.satisfy` (`Click: true`, "mark a
consent condition satisfied: you are saying the certifier or council
accepted it"); `delivery.edit` refuses status satisfied on a condition and
points to the click; `delivery.accept_evidence` (`Click: true`);
`reports.add_scope_line` (appends one fixed-template line to the package's
RFP draft at `custom:<stage>` and records `details.rfp`; applied at once
with Undo once `report_settings` is captured, else prepared).
`internal/httpapi/chat_more.go`: `read(delivery)` with kind condition.
Proof: an operations test (satisfied refused, then prepared; a scope line
added and undone) and the six key cases. About 70 Go.

**7. Program gates.** `internal/program/generate.go`, `review.go`,
`checks.go`, `internal/store/program_prepare.go`. `GenInput.ConsentGates`
holds the live, applicable conditions per gate stage, from one more query in
the existing `readProgramGenInput` batch. One hold point per gate with any
condition: key `consent:<stage>`, label "Consent conditions before the
construction certificate", linked finish-to-start to the
`fr.construction-certificate` activity, the first construction row, the
`fr.occupation-certificate` activity, or a subdivision certificate activity
when one exists. Added in the outline too. The label carries no count, so it
never churns the review; the hold point detail shows "3 of 14 open".
`SourceOf` maps "consent:" to a new `SourceConsent`. A check
`consent_gate_open` fires only when a gate is released with conditions open.
Gates have no interface, so the passage release reading
(`DocumentHoldPoints`) skips them and asks nothing new. Proof: generate tests
(placed and linked; none without conditions; existing goldens unchanged), a
review test (an add from source consent), and the existing release test on a
gate. About 110 Go.

**8. Evidence suggestion, in plan 4's job.** Plan 4's `register` job
(queued by `queueRegister` beside `queueVisual` in
`internal/store/filing.go`, only when the project has open RFIs or open
conditions) gets this plan's two steps.
- Rule: the document cites "Condition 13" or "Conditions 4 and 5" with the
  consent reference, or is filed under Authorities. That makes a rule
  reading, shown at once.
- Jev, for any kind but photo and unknown: one Noul per condition the rule
  left, `register.evidence.c<n>`, over a Go shortlist of at most 12 open
  conditions with evidence not false (discipline match first, then shared
  title words, as `WorkspaceImportCall` shortlists). One Noul per option
  lets one document evidence several conditions
  ([companion Noul](https://docs.typesafe.ai/patterns/companion-noul): "Ask
  one Noul per option when a message may match several"). State
  `conditions.c1…c12` holds number, heading, stage and the first 400
  characters. True: the document is the plan, report, certificate, approval
  or receipt the condition asks for, or says it is met. False: it only
  mentions the topic.
- These Nouls ride in the same request as plan 4's Question A, on the one
  `document` state. Go cuts that excerpt around "RFI", "response" or
  "condition", else it takes the opening text.
- Readings go to `delivery_readings` (field `evidence`). Bands come from
  plan 4's `data/register/thresholds.json`, so Jev suggestions wait for the
  owner's cut-off and rule ones do not. A suggestion never changes a status.

Proof: shortlist cap and order, rule match on "Condition 13", no Jev call
when nothing is open, no status change, and readings gone when the document
is deleted. The key maps about 20 Petersham consultant documents to the
conditions they evidence. About 90 Go.

**9. Conditions in the register.** Plan 4's table in
`web/src/Delivery.tsx`; no tab and no new column. Conditions use the shared
columns (No., Item, With, Due, Status, System) and group by gate in order.
Status says what is needed in words: "Open · evidence needed", "Possible
evidence · Link", "Evidence filed", "Satisfied". Writes go through plan 4's
`POST /ops` path (area delivery), so each action has one Undo, and its
right-click batch sets status, package or system on many rows at once. The
detail strip shows the council's text and "Open page 12" (the existing
viewer); a "to check" chip says how each value was decided (rule, Jev or
you). Above the group: the Profile's consent number, date and lapse date,
plus "Read 96 of 102", the gaps or the error. Actions: Link evidence
(filed-document picker; sets evidence filed), Possible evidence → Link,
Satisfied, Read conditions (unread Authorities documents), and Add to RFP
(disabled with "Create the RFP first" when there is no draft). Acceptance
gates nothing: proposed rows count for the gates. Proof: a browser test on
the Seven Hills fixture (filter, edit, link, add to RFP). About 200 TSX.

**10. Bench and eval.** `cmd/intake-bench/delivery.go` seeds 300 conditions
and samples `delivery_read` and `delivery_write`. Add `consent_read` and
`delivery_import` to `bench/budgets.json`, measured by `cmd/consent-eval -live` on the
102-condition notice. About 30 Go plus the 200-line tool.

**Totals:** about 1,010 lines of product Go (a third of it the split and
its rules), 200 TSX, 20 SQL, and a 200-line eval tool.

## From chat

Plan 0's rule table governs (`2026-11-18-chat-agency.md`).

- **Reads.** `read(delivery)` with kind condition: number, heading, gate,
  who acts, system, whether evidence is needed, status and possible
  evidence; the consent's number, date and lapse date; how many conditions
  were read and which numbers are missing. The council's words are outside
  text, so a change in a turn that read them waits for a click.
- **At once, with Undo.** `delivery.edit` (who, package, system, gate;
  evidence filed or not applicable); `delivery.link` (evidence);
  `reports.add_scope_line`.
- **Waits for a click.** `delivery.satisfy`; `delivery.accept_evidence`;
  `delivery.read_notice` (plan 0, decision 1).
- **Screen.** Filter by gate, who acts and status; select rows; open the
  notice at a condition's page.
- **Named filters.** Open before the construction certificate; open before
  occupation; with no one to act; to check.
- **Key cases.** "What is still open before the construction certificate
  and who has it"; "which conditions need the arborist"; "show the ones
  nobody owns"; "give these to the landscape architect" (rows selected by
  hand); "put those on her RFP"; "mark condition 46 satisfied" (prepared).

## Edge cases

| Case | What happens |
|---|---|
| Scanned notice | The existing OCR job runs first; the split reads OCR text; gaps listed; the PM adds missing ones by hand |
| No numbered conditions (refusal, bare notice) | `Detect` false, nothing written, the strip says so |
| Numbering 1., 1), A1, B3; cross-references | Sequence per prefix; a cross-reference is not a condition |
| Condition over 12,000 characters | Kept to 12,000 with "continues on page N"; Jev sees the first 3,500 |
| Modification read before or after the original | Matched by consent reference and number; newer notice wins; accepted rows only flagged |
| Consent reference not found | Keyed by document; strip says "Could not tell which consent this changes" |
| Two consents on one project | Separate references; the filter picks one |
| Notice read twice or re-read | Same fingerprint, zero calls |
| Jev timeout or circuit open | Rule fields written, picks grey, the job retries; the strip shows what waits |
| Discipline with no package, or two | Who shows the discipline; the package stays blank for the PM |
| No certificate activity in Program | The gate sits at its stage start, with a generation note |
| Satisfied, then a modification changes the text | Status kept; flagged "changed since satisfied, check" |
| Money, hours or dates in the text | Kept as the council's words; never parsed into Cost Plan or Program |
| Notice filed as Correspondence | Not auto-read; Read conditions offered for Authorities documents |
| Notice deleted after the read | Rows stay, marked "notice deleted"; the PM retires them if wanted |
| Evidence document deleted | Its suggestions go (cascade); a linked evidence entry shows "document deleted" |
| More than 600 conditions | Refused: "split the document", as the specification import does past 1,000 |
| Wrong org or project on the route or an evidence link | 404, or 422 for a document outside the project |
| Chat tries to mark satisfied | Prepared for the PM's click |
| RFP already issued | The line goes to the draft only (existing report rules) |

## Risks

- **Split rules fit too few layouts.** The key covers three councils and a
  scan; gaps always show; a missing condition can be added by hand.
- **OCR misreads numbers on old scans.** Gaps show it; nothing is guessed.
- **Option order bias in `party`.** Measured on the key; amber until
  reviewed.
- **Plan 4 slips.** Task 9 then ships as a filter on `Delivery.tsx`, reached
  from the consent number on the Profile. Still no new left-nav entry.
- **Migration collision.** Plans 4 and 6 edit the same checks; rebase on
  merge.
- **Worker queue.** The worker runs one job at a time, so a consent read can
  wait behind a long profile update; the strip says "Queued".
- **Shared request grows.** Condition and RFI Nouls share one `document`
  state, and "accuracy falls as the state grows with content unrelated to the
  decision" (jev-1.13). Both keys measure it. If it costs accuracy, split
  into one background call per register: still one per document per
  register, never on a click.
- **Siblings.** Plan 4: the `register` job, `delivery_readings` (field
  `evidence`), `system_id` and the two checks are shared (the import
  route is this plan's);
  the second plan to land rebases. Plan 5 counts open conditions per gate
  from these rows for its Registers section. Plan 2: nothing; a guest
  sees no register row.

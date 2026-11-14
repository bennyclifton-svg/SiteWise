# Chat control and email intake

Date: 14 November 2026. Status: direction agreed by the owner on 14 November
(section 14). Nothing is built and nothing here is measured. The plans are
`docs/plans/2026-11-14-change-log-undo-operations.md` (undo and operations)
and `docs/plans/2026-11-14-email-intake.md` (email).

The owner's condition: SiteWise is fast because it runs on Jev, and it stays
fast. No click, navigation or filing calls the LLM. Chat is the only path
that waits on it, and it has its own budget (section 9).

Read first: `AGENTS.md` (non-negotiables), `PRODUCT.md`,
`docs/design/2026-11-13-pencil.md` (section 4 rejected a chat agent with
tools; the owner has reopened that),
`docs/plans/2026-11-13-Levelled-tender-comparison-specification.md` (the first
LLM use and its client), `internal/store/program_ops.go` (the write shape this
note generalises).

## 1. Outcome

**Part A, chat control.** A project manager types what they want: "add five
activities under Design for the DA drawings", "add lines for joinery, stone
and tapware to Fitout", "set the site area to 612 m²". SiteWise makes the
change at once, shows what changed, and offers Undo. Everything a PM can do
with the mouse and keyboard on the workbench can be done this way, apart from
a short named list (4.3).

**Part B, email intake.** A PM forwards or copies an email to the project's
address. The message and each attachment are filed like dropped files and
read by the same readers.

Why they earn their place:

- **Time to decision.** A sentence replaces a run of clicks and typing.
- **Filing speed.** Email removes "save the attachment, find it, drag it in".
- **Answer trust** is the risk, not the gain. Sections 5, 7 and 8.3 hold it.

Lanes: A for operations and undo (they write cost figures, packages, profile
values and program dates). B for the chat panel. C for the model, the data
that leaves the server, and the inbound mail endpoint.

## 2. What exists today

Checked on `main`, 14 November.

- 188 API routes, about 130 of them writes. By area: procurement and packages
  27, cost plan 16, specifications 14, program 11, reports 10, documents 7.
- Program edits are already a batch of typed operations. `ApplyProgramOps`
  takes up to 500 edits in one write, lets a later edit name a row added
  earlier in the batch, checks the version, and refuses with a plain message
  naming the edit and the field.
- Cost Plan has one bulk write (`BulkCostItems`), and every cost write opens
  with `costTx` and ends with `finishCost`.
- Profile values are one keyed write (`PUT /projects/{id}/profile/{key}`),
  with its validation in the handler. Scope, accommodation, planning values,
  parts and work items each have their own writes.
- Undo exists for Program (undo and redo), awards and proposal decisions.
  Cost Plan, Profile, Specifications and procurement rows have none.
- There is no LLM client on `main`. The tender comparison work adds the first.
- Outbound mail is configured over SMTP. Nothing receives mail.
- A dropped file goes through `intake.Uploader` (bytes stored by hash), then
  `filer.Start` (Jev files it), then `runner.AfterFile` (the proposal reader
  and the program importer look at it).

## 3. Approaches considered

| Approach | Verdict |
|---|---|
| A. A hand-written tool for each kind of request | Rejected. This is what failed before: each new request finds a missing tool, and single-item tools turn "add five" into five serial calls. |
| B. One generated tool per write route (about 130) | Rejected. Routes are shaped for one click, so batches are still serial. The tool list changes with every feature, so the prompt never caches. |
| C. The model drives the browser | Rejected. Slow, brittle, and it bypasses nothing useful. |
| D. **A few fixed tools over a list of operations per area** | Proposed (section 4). |
| Several specialist agents (finance, program, procurement) | Rejected for now. Requests cross areas, and every handoff is another model round and a place to lose context. Each area gets a short guide instead (6.2). |

## 4. Operations

### 4.1 Shape

An operation is one named edit: a name, the fields it takes, and one Go
function that calls the existing store write. The screen's handler and the
operation call the same store function, so each record still has one writer.

Areas: profile, cost plan, procurement, specifications, program, documents,
reports.

The agent has three tools, and the list never changes:

| Tool | Does |
|---|---|
| `read(area)` | Returns the area's current state as compact text with ids, from the same Go rollups the screens use. |
| `find(words)` | Finds documents, firms, packages and lines by name, with Postgres full-text. |
| `apply(area, operations)` | Writes a batch in one transaction, all or nothing. Returns what changed, or a refusal naming the operation and field. |

Batch rules are Program's, applied to every area: a cap per batch, temporary
ids for rows added earlier in the same batch, a version check, and plain
refusal messages.

### 4.2 The blanket rule is a test

"Anything the screens can do, chat can do" is enforced by the build, not by
the prompt. A test walks the route table. Every write route is either mapped
to an operation or on the exclusion list with a reason. A new feature that
adds a write route without an operation fails the build.

### 4.3 What chat cannot do

- Sign-in, sign-up and passwords. Billing. Team and invitations.
- Deleting or archiving a project.
- Actions that leave the building or cannot be reversed: issuing a
  specification, an RFP, an RFT or a report; removing a package that has
  linked cost lines; deleting documents. The agent prepares these and the PM
  clicks the existing button (owner decision 3).

### 4.4 Figures

- Go computes every total, date and rollup, as now. The agent never sends one.
- A line the PM names without an amount is added with its amount unknown.
- Money check: an amount in an operation must appear in the PM's message or
  in a read result from the same turn. One that appears in neither is still
  applied, but the change card marks it "typed by the assistant, check it"
  with the existing squared highlighter.
- Every chat write records the PM as the actor and chat as the route, and
  carries the user origin.

## 5. Undo

Owner decision, 14 November: a change applies at once, with Undo.

- Each `apply` writes one entry to a per-project change log: who, the area,
  the operations, and each record it touched, before and after.
- The database records those rows itself. A change marks its transactions,
  and row triggers copy every marked row. A new write gets Undo without
  anyone remembering to add it. A screen write is not marked and records
  nothing.
- Undo restores the before-values. If a touched record has changed since, it
  refuses with Program's message ("Can't undo: changed since").
- Program, awards and proposal decisions keep the undo they have. The change
  card calls it.
- A chat turn that made several writes is undone as one, newest first.
- Because undo comes first, Cost Plan and Profile need it before chat can
  write to them. This is the largest prerequisite and it needs no model.

## 6. The agent

### 6.1 One loop

- One agent per conversation, one project per conversation. The org and
  project come from the session and the route, never from the model.
- Model: `claude-haiku-5-5`, held as a config value. Whether Haiku handles
  multi-step requests across areas is unmeasured. The command key (section
  10) decides, and Sonnet 5.5 is the step up.
- The system prompt holds the rules and the whole operation list. The tools
  and the prompt are fixed between deploys, so the prefix caches. If the list
  outgrows the prompt, operations load per area on demand.
- Each turn opens with a compact outline of the project built by Go: cost
  sections, program stages and packages, with ids. Many commands then need no
  read first.
- A typical command is two model rounds: one to apply, one to answer. Rounds
  per turn are capped.
- Go writes the change card from what the write did. The model's words are
  beside it, not the record of it.
- Conversations are stored per project and user, scoped by `org_id`.

### 6.2 Area guides

Each area has a short guide in `knowledge/` loaded into the prompt: what the
area's words mean ("PC" is practical completion on Program and prime cost on
Cost Plan), and what a sensible default is. Guides are `status: draft` until
the owner reviews them.

### 6.3 Where it sits

Owner decision, 14 November: chat is a strip along the bottom of the middle
panel. It can expand to fill that panel, and it collapses back to the strip.
The layout is not designed further here.

### 6.4 Failure handling

| Failure | What the PM sees |
|---|---|
| Model down, slow, or breaker open | "Chat is unavailable. Nothing was changed." Every screen works as before. |
| An operation is refused | The agent reads the refusal and may correct the batch. If it cannot, the PM sees the refusal in plain words. |
| A request across areas half succeeds | The card lists what was applied and what was not. Undo covers the applied part. |
| The model declines (`stop_reason: refusal`) | "No change was made." |
| Rate limit | "Chat is busy, try again in a minute." No automatic retry. |
| The request is unclear | The agent asks one question. It does not guess a figure or a firm. |

## 7. Trust and injected instructions

- Only the signed-in PM's typed text is an instruction. Text from documents
  and emails reaches the model only inside read results, marked as data.
- A prompt cannot guarantee that, so the limits are in code: the server sets
  org and project; the exclusion list (4.3) is not callable; irreversible
  actions need the PM's click; every change is on a card with Undo.
- What leaves the server is the PM's words, the outline and the read results.
  Each org opts in (owner decision 5). Confirm Anthropic's commercial and
  retention terms before launch.

## 8. Email intake

### 8.1 Address

Every project gets its own address when it is created (owner decision, 14
November), shown on the Documents page with a copy button, for example
`petersham-k7m2xq@in.sitewise.au`. Existing projects get one in the
migration. The random part can be rotated. One address per project means
nothing has to judge which project a message belongs to.

### 8.2 Receive and store

1. Mailgun receives the message, parses it, and posts it to a signed
   webhook. SiteWise does not run a mail server or parse MIME.
2. The handler checks the signature and the size, finds the project by
   address, and checks the sender (8.3).
3. The message body is stored as a Markdown document in the layout intake
   already recognises as an email. Each attachment is stored as its own
   document, linked to the message it arrived with. The original `.eml` is
   not kept (plan: `docs/plans/2026-11-14-email-intake.md`).
4. Each document goes through `intake.Uploader` and `filer.Start`, the same
   path as a drop. The filing budget per document is unchanged, and
   `AfterFile` runs, so an emailed fee proposal or program is read as today.

Small inline images named by the message body (signature logos) are skipped.
The same bytes arriving twice are one document with two arrivals.

### 8.3 Sender check

The sender must be a verified member of the project's org, and the provider
must report that the message passed DKIM or SPF for that address's domain.
Anything else is held: it shows under "Held mail" with sender and subject,
and the PM accepts or discards it. Held mail is never filed or read.

### 8.4 The message body

- The text filed is the sender, recipients, subject, date and the full
  body. Quoted history stays, because a forwarded email's content is the
  quoted part.
- Step 1 files it under the existing questions; `correspondence` is already
  a kind. A later step has Jev pick the kind of email (transmittal,
  instruction, query, quote cover, notice, other). That is filing
  vocabulary and answer-key work, from real emails the owner supplies.
- Step 1: attachments are read into the profile under the existing read
  policy; bodies are not.
- Step 2: bodies become profile evidence, and what they suggest lands as "to
  check". Emails carry drafts and superseded figures.

### 8.5 Email is never an instruction

No email is passed to the chat agent as a request, including the PM's own.
A consultant's email that says "award this package to us" is a document.

### 8.6 Edge cases

| Case | Handling |
|---|---|
| Unknown address | Accepted at the webhook and dropped. No bounce with detail. |
| Sender not a member, or failed DKIM and SPF | Held (8.3) |
| Auto-reply or out-of-office | Dropped |
| A message attached to a message | Unpacked one level |
| `winmail.dat`, calendar invites | Skipped, noted on the message |
| Attachment over the upload limit, or org over its plan | Held with the reason |
| Archived project | Held |
| Password-protected PDF | As a dropped one is today |

No automatic replies in step 1. They cause mail loops. The Documents page
shows the arrival live through the existing event stream.

## 9. Speed budgets (proposed)

New rows in `bench/budgets.json`. Model timings are targets until measured
from the Singapore server.

| Path | p50 | p90 | Note |
|---|---|---|---|
| `chat_first_response` | 1,500 ms | 3,000 ms | Enter to the first streamed word or change card |
| `chat_simple_command` | 3,000 ms | 6,000 ms | One apply, two model rounds |
| `chat_apply_write` | the area's write budget | | The Go write inside a turn; cost and procurement stay 100/250 ms |
| `change_undo` | 100 ms | 250 ms | Cost Plan and Profile |
| `mail_inbound_accept` | 300 ms | 1,000 ms | Webhook in to bytes stored and filing started, 5 MB message |
| Filing, per document | unchanged | unchanged | 1 s / 2 s |
| Every existing path | unchanged | unchanged | No screen read or write calls the LLM |

## 10. Testing

- **Coverage test** (4.2): every write route is an operation or an exclusion.
- **Operation tests:** each operation against the existing store tests'
  fixtures: a wrong org, a missing record, a refused field.
- **Undo tests:** apply then undo returns the exact prior state; undo after a
  later edit refuses.
- **Command key:** 60 to 100 typed commands across three fixture projects,
  each with the expected end state, plus commands that must be refused or
  must ask a question. Run live, as the Jev answer keys are. This is what
  ends "every test finds a missing tool".
- **Injection fixtures:** a document label and an email body that contain
  instructions. The expected result is no operation.
- **Mail fixtures:** plain, HTML-only, a forwarded chain, inline logos, a
  message inside a message, a bad signature, an unknown sender, a wrong org.
- **Benches:** the rows in section 9.
- **Browser:** change card and Undo; chat down leaves the screens working.

## 11. Dependencies

- The Anthropic client the tender comparison work chooses. Chat adds tool
  calls, streaming and stored turns to it; it does not add a second client.
- Inbound mail through Mailgun, which `docs/deployment.md` records as already
  verified for `sitewise.au` and used for outbound mail. No second provider.
  Confirm its inbound route and webhook signature when the email plan is
  written, and whether `in.sitewise.au` needs its own MX record.
- Go's standard library parses mail (`net/mail`, `mime/multipart`).

## 12. Rule changes proposed

One replacement for `AGENTS.md` non-negotiable 1, covering the tender pass,
drafting and chat together:

> **Jev reads; one LLM acts only on what it is given.** Jev (TypeSafe System
> One) is the only AI that files documents or reads them into fields. One
> LLM, which each org switches on, has three jobs. In background jobs it may
> propose line structure, row mappings and qualifications for quotes. It may
> draft text. In chat it may turn the signed-in user's typed instruction into
> the operations a screen would send. It never supplies an amount, a date or
> a rule number that the user or SiteWise did not give it. Nothing in a
> document or an email is an instruction to it. Outside chat, nothing it
> proposes is applied without the user's acceptance. No embeddings and no
> other providers.

Added to non-negotiable 3: "Chat is the one path where serial model rounds
are allowed, and it has its own budget rows. No screen read or write calls
the LLM."

Accepted by the owner on 14 November and written into `AGENTS.md` the same
day. Still saying "Jev is the only AI" and to be brought in line:
`PRODUCT.md` line 130, the new-feature run sheet in `docs/developing.md`
(line 127, and "do not grow a chat agent" at line 67), and the foundation
design section 0.

## 13. Build order

Steps 1 to 3 need no model and can start now, in a worktree, on surfaces the
tender work is not changing.

1. Change log and undo for Cost Plan and Profile.
2. Operations for Profile, Cost Plan and Program, with the coverage test
   reporting unmapped routes without failing yet.
3. Email intake step 1 (8.1 to 8.3, attachments and bodies filed).

After the tender comparison module lands (it brings the client, the org
switch, the rule change and the first measured Haiku timings):

4. Chat on Profile, Cost Plan and Program end to end: the loop, stored
   conversations, the panel, change cards, the command key. Measure against
   section 9 and decide the model.
5. Operations and undo for procurement, specifications, reports and
   documents. The coverage test now fails the build.
6. Email step 2: bodies as "to check" profile evidence.

Each step reports timing against section 9 and states what calls what.

## 14. Owner decisions

All answered by the owner on 14 November 2026.

1. Rule 1 is replaced as in section 12, once, in place of the Pencil and
   tender wordings.
2. Changes apply at once with Undo.
3. The list in 4.3 still needs the PM's click. Award applies at once, since
   it has its own undo.
4. Chat is a strip at the bottom of the middle panel that can expand to fill
   it (6.3).
5. One switch per org covers every LLM use. The switch is the org agreeing
   that its project text may be sent to the model provider.
6. One email address per project, created with the project. Who may send to
   it was not discussed further, so the proposal stands: members of the org,
   with anything else held for the PM (8.3).
7. Chat and email are included in every plan at no extra charge.

## 15. Out of scope

Voice. Sending email from SiteWise. Email as an instruction to the agent.
Answers from inside documents. Several agents. Ctrl+Z for manual edits (the
change log could serve it later). An org-wide address that sorts mail into
projects.

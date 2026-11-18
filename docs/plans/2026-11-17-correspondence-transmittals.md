# Correspondence and transmittals: implementation plan

Date 17 November 2026; revised 18 November. Lane A (a record of what was
sent and received), with lane C's rules on the send path (mail leaves the
server; untrusted input). Status: plan only, nothing built.

**Revised 18 November** after the review in `2026-11-17-next-wave-index.md`.
A message is a record; what is owed is a register row, so Notice, Due and
Status are no longer columns here. Contract time bars are deferred. Each
message offers one next step. A document shows who holds which revision.
Messages are frozen. Chat reaches everything ("From chat"). The earlier
text is at git ref `refs/backup/next-wave-plans-2026-11-18`.

It sits on inbound email intake (`docs/plans/2026-11-14-email-intake.md`,
design `docs/design/2026-11-14-chat-control-and-email-intake.md` §8) and does
not re-plan it. It owns sending mail out; plans 3, 4 and 5 call its one send
path. Guests (plan 2, `2026-11-17-guest-access.md`) get links for large
sends; this plan does not design them.

## In one page (product requirements)

**Outcome.** The repository's Messages toggle (email intake, owner 15 Nov)
becomes the correspondence record. Each row shows In or Out, number (sent
ones: TR-001), date, the other party, subject and package. Replies sit
under the message they answer, and a message that has been answered says
Replied. To send, the PM selects documents in the repository, right-clicks
Send…, picks firms or types addresses, adds a note and clicks Send. A
numbered transmittal sheet goes with the documents and the message lands in
the record as Out. Nothing in the record can be edited or deleted.

A message that looks like an RFI, a claim or a notice shows one chip and
one button: "Possible delay notice · Add to register". One click puts it
where it belongs (the register, the claims, the variations) with the
message as its source. A document's Filing column says who was sent it;
when it is superseded, the new revision says "3 firms hold revision B" with
Send… filled in for them.

**Why.** Answer trust: the record shows what was sent, to whom, in which
revision, and it cannot be altered afterwards. Filing speed: sent and
received mail is filed with no dragging. Time-to-decision: a claim or a
notice in the inbox is one click from the place that tracks it.

**Where it lives.** No new screen, no new left-nav entry. Record: columns
on the Messages list in the repository panel (`web/src/MailIn.tsx`, made by
email intake Task 9). Send: a "Send…" item in the repository's right-click
menu (`web/src/Register.tsx:302`) opening a dialog on `ConfirmDialog`.
Reply: a button on an opened message, same dialog. Who holds a document: one
line in the document viewer's Filing column.

**What reads what.** Jev: one choice question, `notice`, joined to the
existing filing fan-out of an email body, so no new call (Task 2). Go:
threading by `Message-ID`, `In-Reply-To`, `References`; direction; firm and
package by sender address; the Replied mark; who holds which revision;
numbering; the sheet; MIME and SMTP. LLM: none on any screen path; in chat
it may draft the note that the PM then sends. Email is never an instruction
(design §8.5); only the PM's click changes a record.

**Speed budgets.**

| Path | p50 / p90 | Bench row |
|---|---|---|
| Record read, 1,000 messages, or one document's transmittals | 100 / 250 ms | `mail_read` (email intake, reused) |
| Record edit (package, dismiss the chip) | 100 / 250 ms | `mail_decide` (reused) |
| Send, click to 202: check, render sheet, store, queue | 100 / 250 ms | `transmittal_send` (new; the `report_issue` budget) |
| Queued to accepted by the mail service, 10 MB attached | 3,000 / 10,000 ms, background | `mail_send` (new; bench on a local SMTP sink) |
| Filing an email body (the pick joins the fan-out) | 1 s / 2 s | `whole_intake`, `jev_admission_request` unchanged |
| Inbound webhook (adds thread and firm lookups) | 300 / 1,000 ms | `mail_inbound_accept` unchanged |

**Done means.** Tests in Tasks 1–7 pass; `tools/check.ps1` passes; the
bench shows the rows above in budget; on the live server the owner sends a
transmittal of two drawings to a second mailbox, replies, and sees the reply
under it and the transmittal marked Replied; one of the drawings is then
superseded and its new revision names the second mailbox as holding the old
one. No route edits or deletes a filed or sent message.

**Out of scope.** Inbound plumbing. Delivery tracking ("Sent" means the
mail service accepted it). Drafts, templates, scheduled sends, letterhead
letters. Reading attachments for notices. Contract time bars: a knowledge
file of clause periods and a due date worked out from it are deferred
(the periods sit in copyrighted forms that special conditions amend; the PM
types the date on the register row). Security of Payment dates (plan 3).
Guest links (plan 2). Approval workflows, distribution lists. A shared
record across organisations: this is the PM's own record.

**Owner decisions** (recommended default given). The plan takes its
defaults for 3, 4, 6 and 7; 1 and 2 are in the index.
1. **Outside senders.** File, not hold, an authenticated message that
   replies to one SiteWise sent for this project, or comes from a firm
   address linked to this project (invited, quoted or awarded). Hold the
   rest. (Amendment A1.)
2. **From address.** "Org name via SiteWise" at the project's own address;
   Reply-To the PM and the project address; Cc the PM. One DNS step:
   verify `in.sitewise.au` for sending in Mailgun. Fallback: From
   `SITEWISE_MAIL_FROM`, same Reply-To.
3. **Attach or link.** Attach up to 10 MB in total; above that, links
   (plan 2). Until plan 2 lands, the send is refused with the size.
4. **Numbering.** One series per project, TR-001, for every message sent
   from SiteWise, never reused. Received messages get no number.
5. **What a message may be.** Request for information; variation claim;
   extension of time or delay notice; payment claim; notice of dispute;
   direction or instruction; another notice under the contract; none. Every
   pick shows "to check" until an answer key of about 50 real messages sets
   a threshold.
6. **Send cap.** 100 messages and 500 recipients per org per day; the
   sender's email must be confirmed (as for uploads).
7. **A member's own mail.** A team member's message copied to the project
   address is Out. One whose subject starts "Fwd:" or "FW:" is In, from the
   original sender when Go finds the forwarded header block.

## Prerequisite and amendments to email intake

Email intake Tasks 0–9 land first. Task 3 here (sending) needs only its
Task 2 (tables and addresses). Amendments to that plan:

- **A1** (decision 1). Task 6 step 7: after the member check, an
  authenticated sender is filed when `In-Reply-To` or `References` names a
  `message_id` this project sent, or when the address equals a linked
  firm's `email`. A domain match links a firm (Task 1) but never admits.
- **A2.** `RecordMail` keeps `To`, `Cc`, `In-Reply-To`, `References`
  (already in `mailin.Message`); Task 1 adds the columns and rules.
- **A3.** Task 6 step 10: write each part's `document_id` as soon as
  `Uploader.File` returns, before `deps.Filer.Start`, so the reading finds
  its message.
- **A4.** Class its write routes in `internal/httpapi/ops_routes.go`
  (`TestWriteRoutesHaveOperations`): the webhook is `noOperation` (a
  provider webhook). Accept, discard and rotate follow plan 0's rule table:
  accept starts Jev filing, so it is a prepared read if the owner takes
  plan 0's decision 1; discard cannot be put back, so it waits for a click;
  rotate changes the project's address and stays `noOperation`.
- **A5.** Task 0 also confirms `message-headers` carries `In-Reply-To` and
  `References`, and verifies `in.sitewise.au` for sending.
- **A6.** Rotate warns: "Replies to messages sent from the old address will
  not arrive."
- **A7.** No route deletes a filed message. Discard applies to held mail
  only.

## What we reuse

| Need | Existing thing | File |
|---|---|---|
| Mail tables, address, webhook, Messages list | Email intake | `docs/plans/2026-11-14-email-intake.md` Tasks 2–9 |
| Email bodies settle as `correspondence` by rule | `markdownEmail`, `email_headers` | `internal/intake/markdown_identity.go:119`, `internal/intake/decisions.go:186` |
| One fan-out per filing; joining a question to it | `Draft.open`, `callFrom`, `catalogQuestion`; hold-point precedent | `internal/intake/draft.go:189`, `internal/intake/questions.go:240`, `:273`, `internal/jobs/holdpoint.go:88` |
| Raw Jev pick before any threshold | `Draft.Choices` | `internal/intake/draft.go:139` |
| One filing transaction | `Store.CommitFiling` | `internal/store/filing.go:175` |
| Document schedule; corrections win; foreign ids refused | `reportDocumentScheduleSelection`, `rfp.DocumentRows` | `internal/store/report_documents.go:15`, `internal/rfp/sections.go:135` |
| Which document supersedes which | the supersession link on a filing | `internal/store/filing.go` |
| One-page PDF in the house style | `export.Doc`, `export.PDF` | `internal/export/doc.go`, `internal/export/pdf.go:61` |
| SMTP settings; Mailgun on 587 | `config.SMTP*`, `smtpMail` | `internal/config/config.go`, `cmd/sitewise/main.go:447`, `docs/deployment.md` |
| Firms with email; who is on which package | `firms`, `package_invitations`, `submissions.firm_id`, `GET /firms` | `internal/db/migrations/032_procurement_award.sql:33`, `052_package_invitations.sql`, `internal/httpapi/submissions.go:739` |
| Session, origin, project, confirmed email | `memberSession`, `originOK`, `visibleProject`, `uploadAllowed` | `internal/httpapi/projects.go:600`, `:623`, `documents.go:432`, `billing.go:295` |
| Dialog; firm search | `ConfirmDialog` (takes children); firm search | `web/src/ConfirmDialog.tsx`, `web/src/FirmNameEditor.tsx` |
| Adding a register row, a claim, a variation | `delivery.add` (plan 4); `CreateInvoice` (plan 3); `POST …/variations` (plan 3) | sibling plans |
| Chat's inbox tool (placeholder today) | `messages` | `internal/httpapi/chat_tools.go:109` |
| Chat opening a dialog filled in | plan 0's `view` tool and `chatView.ts` | `2026-11-18-chat-agency.md`, Task 2 |

## What is new

| Thing | Why existing can't carry it | Est. lines |
|---|---|---|
| Columns on `mail_messages` (no new table) | Record fields belong on the message row | 40 SQL |
| `internal/correspondence` (the pick's options, MIME, sender) | `auth.SMTP.send` is private, plain text, one recipient, no attachments or thread headers | 250 Go |
| `POST /projects/{id}/transmittals` | Nothing sends mail with documents | 170 Go |
| `PATCH /projects/{id}/mail/{message}` | Email intake only accepts or discards | 50 Go |
| `GET /projects/{id}/mail/{message}/sheet` | The sheet is not a repository document (filing it would call Jev) | 25 Go |
| Who holds which revision (a `?document=` read on the mail route) | Nothing ties a sent document to its later revisions | 60 Go, 30 TSX |
| `web/src/SendDialog.tsx` | No dialog sends mail | 180 TSX |

Totals: 0 new tables, 3 new endpoints, 0 new screens, 1 new component, 1 new
Go package. About 910 lines of Go and 390 of TSX, tests extra.

## Data

Next free migration number (the wave shares it).

```sql
-- Correspondence record over the mail tables
-- (docs/plans/2026-11-17-correspondence-transmittals.md).
ALTER TABLE mail_messages
 ALTER COLUMN webhook_token DROP NOT NULL,   -- a message SiteWise sent has none
 ADD COLUMN direction text NOT NULL DEFAULT 'in' CHECK (direction IN ('in','out')),
 ADD COLUMN to_addrs text NOT NULL DEFAULT '' CHECK (length(to_addrs) <= 4000),
 ADD COLUMN cc_addrs text NOT NULL DEFAULT '' CHECK (length(cc_addrs) <= 4000),
 ADD COLUMN thread_id uuid,                  -- first message of the thread
 ADD COLUMN firm_id uuid,
 ADD COLUMN package_id uuid,
 ADD COLUMN linked_by text NOT NULL DEFAULT '' CHECK (linked_by IN ('','rule','user')),
 ADD COLUMN notice_read text NOT NULL DEFAULT '',   -- Jev's pick, a proposal
 ADD COLUMN notice_confidence real,
 ADD COLUMN pick_state text NOT NULL DEFAULT ''     -- what the PM did with the pick
  CHECK (pick_state IN ('','used','dismissed')),
 ADD COLUMN number integer CHECK (number > 0),      -- TR-001; sent by SiteWise only
 ADD COLUMN body text NOT NULL DEFAULT '' CHECK (length(body) <= 20000),
 ADD COLUMN send_status text NOT NULL DEFAULT ''
  CHECK (send_status IN ('','queued','sent','failed')),
 ADD COLUMN send_attempts integer NOT NULL DEFAULT 0,
 ADD COLUMN sent_at timestamptz,
 ADD CONSTRAINT mail_sent_shape CHECK ((webhook_token IS NULL) = (number IS NOT NULL)
  AND (number IS NULL OR direction = 'out')),
 ADD FOREIGN KEY (org_id,firm_id) REFERENCES firms(org_id,id) ON DELETE SET NULL (firm_id),
 ADD FOREIGN KEY (org_id,project_id,package_id) REFERENCES packages(org_id,project_id,id)
  ON DELETE SET NULL (package_id);
-- notice_read: CHECK against the eight option ids plus ''.
UPDATE mail_messages SET thread_id = id;
ALTER TABLE mail_messages ALTER COLUMN thread_id SET NOT NULL;
CREATE UNIQUE INDEX mail_number_uq ON mail_messages(org_id,project_id,number) WHERE number IS NOT NULL;
CREATE INDEX mail_thread ON mail_messages(org_id,project_id,thread_id);
CREATE INDEX mail_queued ON mail_messages(sent_at) WHERE send_status = 'queued';
CREATE INDEX mail_parts_document ON mail_parts(org_id,document_id) WHERE document_id IS NOT NULL;
CREATE INDEX firms_email ON firms(org_id, lower(email)) WHERE email <> '';
-- The PM's two edits (package, pick_state) are captured for Undo, as 063 does for other tables.
CREATE TRIGGER capture_change AFTER UPDATE ON mail_messages FOR EACH ROW EXECUTE FUNCTION capture_change();
```

There is no reply status column. Replied is computed at read: a message
has a later message from the other direction in its thread.

A sent message's parts reuse `mail_parts`: part 0 is the sheet (bytes, no
document), parts 1… the documents sent (`document_id` and the file's hash).
A part with no `document_id` keeps its bytes (email intake Task 2), so a
document deleted after sending still downloads from the transmittal.

## Tasks

Files marked (EI) are made by email intake; (new) by this plan.

**Task 1. Record fields at record time** (140 Go, 150 test).
Files: migration (new), `internal/store/mail.go` (EI, `RecordMail`),
`internal/mailin/mailin.go` (EI). In `RecordMail`'s transaction:
- *Thread:* `In-Reply-To`, then `References` last to first, against this
  project's `message_id`s; the first found gives `thread_id`, else its own.
- *Direction* (decision 7): a member and not a forward is `out`; else `in`.
  `mailin.Forwarded(subject, body)` returns the original `From`, or none.
- *Firm and package:* exact `lower(email)` match among the org's firms,
  else the one firm whose email domain is the sender's (never a free-mail
  domain; a fixed list in Go). That firm's packages on this project
  (invited, quoted, awarded): exactly one sets `package_id`; several leave
  it blank; no firm takes the thread's package. For `out`, the recipients.

Test `internal/store/mail_register_test.go` (new): threads by
`In-Reply-To`, and by `References` when that is unknown; a reply before its
parent starts its own thread; exact and domain firm match; gmail never
matches by domain; two packages leave it blank; a member's copy is `out`;
"FW:" is `in` from the original sender; another org's firm never matches.

**Task 2. The pick in the filing fan-out** (95 Go, 100 test).
Files: `internal/correspondence/notice.go` (new), `internal/intake/draft.go`,
`internal/intake/file.go`, `internal/store/filing.go`.
- In `Draft.open`, when `markdownEmail(d.text)` (kind already settled by
  rule), add choice question `notice`, one judgement: "Which of these is
  this message, under the contract?" Options in order: `none`, `rfi`,
  `direction`, `variation_claim`, `eot_delay`, `payment_claim`, `dispute`,
  `other_notice`. Each description is `{covers, not_for}`, the same fields
  for every option
  ([how to build](https://docs.typesafe.ai/concepts/how-to-build-with-system-one));
  `not_for` names the traps (a mention of a variation, a request for a
  quote, a forwarded notice already answered, a question in passing).
- Joined to the existing call, as the speculative fan-out pattern
  ([patterns](https://docs.typesafe.ai/patterns)) and the hold-point
  precedent do; the version gains `+notice-1` (`internal/intake/draft.go:87`).
  No second call. A timeout leaves it unread; `jev_retry` asks again.
- `Draft.Apply` skips `notice`: it never writes `decisions`.
  `Draft.Notice(result)` takes the raw pick from `Choices`. `CommitFiling`
  takes `Notice *NoticeRead` and, in its transaction, sets `notice_read`
  and `notice_confidence` on the message whose part 0 is this document (a
  dropped `.md` has none: no write).
- No threshold exists, so the pick is shown "to check" and never creates
  anything ([confidence](https://docs.typesafe.ai/confidence): act
  automatically only in a calibrated high band). The answer key is also run
  with options reversed, since Jev leans to the first option
  ([jev-1.13](https://docs.typesafe.ai/model-jaggedness/jev-1.13)). Key:
  about 50 real messages the owner labels, half of them notices or RFIs,
  private in `data/eval/notice/`; owner-supplied, not in this build.

Tests: an email body's call carries `notice` beside the filing questions in
one Jev call; a PDF's does not; `notice` never reaches `decisions`; grey
leaves `notice_read` empty; the pick lands on the right message.

**Task 3. The one send path** (200 Go in `correspondence`, 195 in `httpapi`, 220 test).
Files: `internal/correspondence/mime.go`, `send.go` (new);
`internal/httpapi/transmittals.go` (new); `projects.go`, `ops_routes.go`,
`server.go`, `cmd/sitewise/main.go` (wiring); `bench/budgets.json`.
- For plans 3, 4, 5: `queueOutbound(ctx, deps, session, projectID,
  store.Outbound) (store.MailMessage, error)` with `To, Cc, Subject, Body,
  DocumentIDs, Files` (bytes already stored, such as an issued report's
  `ExportSHA256`), `InReplyTo` (a message id), `PackageID`. Callers keep
  the message id on their own record.
- In order: confirmed email; live project; daily cap; 1–20 addresses
  parsed by `net/mail`, no line breaks; documents through
  `reportDocumentScheduleSelection`; ≤ 10 MB; under `lockProject` the next
  `number`, the sheet (`export.PDF`: project, number, date, from, to, the
  `rfp.DocumentRows` table, the note), `blobs.Put`, the `out` row `queued`
  with its parts; commit; a `mail` event; start the send; 202. Budget
  `transmittal_send`.
- `mime.go`: `multipart/mixed` with stdlib `mime/multipart`,
  `quotedprintable`, `base64`; `Message-ID <uuid@domain>`, `In-Reply-To`,
  `References`; names via `mime.FormatMediaType`. `send.go`: a `Sender`
  interface (SMTP, and a test sink like `auth.Sink`), `Start` in a
  goroutine, a minute sweep for `queued` rows older than two minutes, five
  tries with backoff, then `failed`. Retry is the same route with
  `{"retry": "<message id>"}`.
- `POST /projects/{id}/transmittals` is `noOperation`, as report issue is
  (`internal/httpapi/ops_routes.go:47`): "sending leaves the building; the
  PM presses Send on the screen (design 4.3); chat opens Send filled in".
  The sheet route downloads part 0.

Tests: foreign, other-project, duplicate and missing ids refused as in
`internal/httpapi/report_transmittal_test.go`; CR/LF refused; over 10 MB
413; two concurrent sends get TR-001 and TR-002; the sink gets one MIME
message with the sheet first, the documents, `Message-ID`, Reply-To, Cc; a
sink failure retries then fails; after a restart the sweep sends a queued
row once; a reply joins its thread; the cap returns 429.

**Task 4. Record read and edit** (110 Go, 120 test).
Files: `internal/store/mail.go` (EI), `internal/httpapi/mail.go` (EI),
`internal/httpapi/projects.go` (routes, `routePaths`), `ops_more.go`.
- `GET /projects/{id}/mail` adds per row: `direction`, `number`, `with`
  (sender for In, recipients for Out), `package`, `firm`, `notice_read`,
  `pick_state`, `replied`, `send_status`, `thread_id`. `?unanswered=1`
  lists messages with no reply. One query. Budget `mail_read`.
- `PATCH /projects/{id}/mail/{message}` with `version`: `package_id`
  (`linked_by='user'`) and `pick_state`. Nothing else on a message can
  change. Budget `mail_decide`. Both are operations (`mail.link`,
  `mail.dismiss_pick`), applied at once with Undo.

Tests: another org 404; stale version 409; a user's package survives a
later reply; held messages are not editable; a PATCH naming any other
field is refused; no DELETE route exists for a filed message
(`RoutePatterns` test).

**Task 5. Who holds which revision** (60 Go, 60 test).
Files: `internal/store/mail.go`, `internal/httpapi/mail.go`.
`GET /projects/{id}/mail?document=<id>` returns, from `mail_parts`, each
transmittal that carried this document or a revision it supersedes: number,
date, recipients, and which revision went. For the newest revision it also
returns `holding_older`: the firms and addresses sent an older revision and
never sent this one. Inside `mail_read`. Tests: a three-revision chain; a
firm sent B then C is not listed; a deleted older revision still counts
(the part keeps its hash); another project's document 404.

**Task 6. Record, Send and Reply in the browser** (lane B; 390 TSX, 120 test).
Files: `web/src/MailIn.tsx` (EI), `web/src/SendDialog.tsx` (new),
`web/src/Register.tsx` (menu item), the document viewer's Filing column,
`web/src/api.ts`, `web/tests/correspondence.spec.ts` (new). Read
`web/DESIGN.md` first.
- Columns: In/Out, No., Date, With, Subject, Package; replies indent under
  the first message; a replied message says Replied. Package edits in
  place as register cells do. No Notice, Due or Status column, and no count
  on the toggle.
- The chip: when `notice_read` is set and `pick_state` is empty the row
  shows "Possible delay notice · to check" with one button and "Not one".
  The button is Add to register (RFI, direction, delay notice, dispute,
  other notice; plan 4's `delivery.add` with this message as source), Read
  as progress claim (plan 3), or Add variation (plan 3). A button whose
  plan has not landed is not drawn. After either click `pick_state` is
  saved and the chip goes.
- Send…: To and Cc take typed addresses and firms with an email; subject
  "TR-00n <project>"; a note; documents with total size. "Sending…", then
  "Sent" on the `mail` event, or "Not sent" with Retry. Reply: same dialog,
  To the sender, "Re:" subject. The dialog registers with plan 0's
  `chatView.ts` so chat can open it filled in.
- Filing column: "Sent: TR-004 to Acme, 3 Nov". On a revision with
  `holding_older`: "3 firms hold revision B · Send…", which opens the
  dialog with this document and those firms.

**Task 7. Operations, chat reads and the key** (50 Go, 60 test). See "From
chat". Files: `internal/httpapi/chat_tools.go`, `ops_more.go`,
`ops_routes.go`, `data/eval/chat/gen_cases.py`.

**Task 8. Links for large sends, after plan 2** (60 Go). Over 10 MB, each
recipient gets one `documents` grant through plan 2's `CreateGuestGrant`
(its cap: 200 documents a link), and the link goes in the message body
instead of attachments; parts gain `sent_as` (migration then). Plan 2
defines the link and what a guest sees.

**Task 9. Bench, gate, documents.** Bench `transmittal_send` and
`mail_send`; re-run `mail_read`, `mail_decide`, `whole_intake`. `AGENTS.md`:
record read and edit, send and sheet do not call Jev; list this plan under
"Before changing one of these surfaces". `PRODUCT.md`: one paragraph.
`docs/deployment.md`: the sending domain. Evidence note with timings.

## From chat

Plan 0's rule table governs. Every write route here is an operation or has
a `noOperation` reason that plan 0's table gives.

- **Reads.** `messages` (filled by email intake and the chat plan) adds
  direction, number, with, package, Replied and Jev's pick, printed by Go;
  one message's body by id; with a document named, who was sent it and who
  holds an older revision.
- **At once, with Undo.** `mail.link` (a package); `mail.dismiss_pick`.
- **Opened filled in.** Send and Reply, through `view`: documents, firms,
  Cc, subject and a note the model drafts. The PM edits and presses Send.
  Chat never sends. A turn that read a message body is outside text, which
  does not stop a dialog being opened: nothing is written until the press.
- **Screen.** Show Messages; select and filter messages; open a message.
- **Named filters.** No reply yet; not sent; from a firm; on a package.
- **Key cases.** "Who has the old structural drawings"; "what has Acme not
  answered"; "link these to Roofing" (messages selected by hand); "send the
  basement drawings to the structural engineer" (opens Send, sends
  nothing); "reply that we accept" (opens Reply with the note drafted);
  "delete that email" (refused: the record is frozen).

## Edge cases

| Case | Handling |
|---|---|
| Reply arrives before its parent; Outlook drops `References` | Own thread, not re-threaded; `In-Reply-To` alone threads |
| Firm domain, unknown address; free-mail sender | Firm linked, not admitted (A1); free mail matches exact address only |
| Firm on two packages | Firm shown, Package blank for the PM |
| Jev timeout; ordinary email picked as a notice | No chip; or "to check", cleared by "Not one" |
| The chip's button, then the register row is deleted | The message keeps `pick_state` used; the PM can add again from the message's menu |
| Email tells SiteWise to do something | Nothing; it is a document (design §8.5) |
| Document deleted after sending | Transmittal still lists and downloads it |
| A document sent, then superseded twice | The newest revision lists who holds each older one |
| A recipient typed as an address, not a firm | Listed by address under "hold revision B" |
| Unfiled or foreign document; over 10 MB; bad address | 400; 413 with the size; 400 |
| Mail service down; restart mid-send | Retry five times, then "Not sent" with Retry; sweep resends |
| Archived project; unconfirmed email; daily cap | 409, 403, 429 in plain words |
| Address rotated | Replies to earlier sends are dropped; rotate warns (A6) |
| Someone asks to delete or edit a message | Refused on every path: the record is frozen |
| Wrong org on any route | 404 |

## Risks

- **Sent is not delivered**: Mailgun keeps bounces (a failed-delivery
  webhook is a follow-up). A crash after the provider accepted a message
  sends it twice on the sweep; the same `Message-ID` lets clients collapse it.
- **One sending domain for all orgs.** Confirmed email, a daily cap and no
  automatic replies limit abuse; one org can still hurt the domain's name.
- **The pick is unmeasured**; a miss shows nothing, so the PM still reads
  the mail. **Outside mail now files** (A1) and counts against the monthly
  document allowance.
- **Frozen cuts both ways.** A message sent to the wrong firm stays in the
  record. That is the point of the record; the PM sends a correction.
- **"Holds revision B" is what SiteWise sent,** not what a firm received
  some other way. The line says "sent", never "has".

## Conflicts with sibling plans

- Plans 3, 4, 5 send through `queueOutbound` and keep the message id;
  plan 5 passes its issued PDF in `Files`. Plan 2 owns links (Task 8).
- Plan 4 holds what is owed. A notice's due date lives on its register
  row; this plan computes no due date.
- Plan 3 owns "Read as progress claim" and "Add variation"; this plan only
  draws their buttons.
- Open with plan 2: whether an RFT sent by transmittal to an invited firm
  also gives that firm plan 2's `package` link. Recommended: no, until the
  owner decides; the PM shares the package link from Procurement.

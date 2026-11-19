# Email Intake Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Every project has its own email address; a message sent to it by a
member of the project's org is filed like dropped files (the message as one
document, each attachment as another), and mail from anyone else is held for
the PM to accept or discard.

**Architecture:** Mailgun, already verified for `sitewise.au`, receives the
mail, parses it, and posts it to one signed webhook. The handler checks the
signature, finds the project from the address, checks the sender, stores each
part by content hash, and hands accepted parts to the existing upload commit
and filer. The message body becomes a Markdown document in the layout intake
already recognises as an email, so it files with no new Jev question.

**Tech Stack:** Go standard library (`crypto/hmac`, `mime/multipart`,
`net/mail`), PostgreSQL 17, the existing `intake.Uploader`, `filer` and event
broker, React for one line and one list. No new dependency. No LLM.

**Design:** `docs/design/2026-11-14-chat-control-and-email-intake.md`,
section 8, build-order step 3. Owner decisions, 14 November 2026: one address
per project, created with the project; included in every plan.

**Status:** Tasks 1 to 10 built 19 November 2026 on branch `wave/mail`
(not merged, not deployed), to Mailgun's documentation and against
stand-ins: **Task 0 has not been run** and nothing has been sent to or
received from Mailgun. The feature stays off until both settings are set.
See "Task 0 findings: not yet run" and "Result" below, and
`docs/evidence/2026-11-19-mail.md`. Task 11 is the owner's.

**Update, 15 November 2026 (owner):** messages are listed in the repository
beside documents, behind a Documents | Messages toggle at its head. Task 9
therefore lists every message for the project, filed and held, newest
first; opening one shows its body and links to the documents its
attachments became; `GET /projects/{id}/mail` returns filed messages too,
paged. The chat agent reads the same list
(`docs/plans/2026-11-15-chat-agent.md`, Task 6). Migrations 055 to 060 are
now taken on `main`: use the next free number.

**Update, 18 November 2026:** `docs/plans/2026-11-17-correspondence-transmittals.md`
builds on this plan and amends it (its A1 to A7): outside senders who
reply to something SiteWise sent are filed, not held; `To`, `Cc`,
`In-Reply-To` and `References` are kept; a part's `document_id` is written
before filing starts; the write routes are classed for chat; no route
deletes a filed message. Read that section before building Tasks 0, 2, 6
and 9 here.

---

## Feature statement (AGENTS.md run sheet)

- **Outcome:** a PM forwards an email with two drawings to the project's
  address and sees three documents arrive and file in the repository, without
  opening SiteWise to drag anything.
- **Why:** filing speed. It removes "save the attachment, find it, drag it".
- **Lane:** C (a public endpoint with no session, untrusted input, a new
  secret), with lane A's rules for filing and org isolation.
- **Jev:** no new call. Each accepted document is filed by the existing
  fan-out, one per document, as a drop is. The webhook, the held-mail list,
  accept, discard and rotate do not call Jev.
- **Budgets (new rows):** `mail_inbound_accept` p50 ≤ 300 ms, p90 ≤ 1,000 ms
  (signed post of a 5 MB message with two PDFs, to the 200 response).
  `mail_read` p50 ≤ 100 ms, p90 ≤ 250 ms. `mail_decide` p50 ≤ 100 ms,
  p90 ≤ 250 ms (accept a held message with three attachments). Filing stays
  p50 ≤ 1 s and p90 ≤ 2 s per document. Project create and project open keep
  their recorded budgets.
- **Done means:** the tests in Tasks 2 to 9 pass, `tools/check.ps1` passes,
  the bench reports the three new rows inside budget, and on the live server
  a real email from the owner's mailbox files its attachments (Task 11).
- **Edge cases:** see the table under Task 6.
- **Out of scope:** email as an instruction to anything; reading message
  bodies into the profile (bodies arrive with profile reading off); a Jev
  question for the kind of email, and its answer key; replies or receipts
  sent by SiteWise; outside senders on an allow-list; keeping the original
  `.eml`; unpacking a message attached to a message; an org-wide address.

## Where this departs from the design note

| Design note | This plan | Why |
|---|---|---|
| The raw message is stored as a document | Mailgun parses the message; the body is stored as a Markdown document and the raw MIME is not kept | No MIME parser to write or secure. The PM still has the original in their mailbox. |
| Body text with quoted history cut | The full plain-text body | A forwarded email's content is the quoted part. |
| Jev picks the kind of email | The existing filing questions only (`correspondence` is already a kind in `data/intake/kinds.json`) | A new question needs an answer key from real emails. It is a follow-up. |

## What exists (read before starting)

| Thing | Where |
|---|---|
| Upload: bytes by hash, then one commit | `internal/intake/upload.go` (`Uploader.Upload`), `Store.CommitIntake` in `internal/store/store.go:630` |
| The upload handler: plan check, commit, count, start filing | `uploadFile`, `internal/httpapi/documents.go:146` |
| Plan allowance written against an HTTP response | `uploadAllowed`, `internal/httpapi/billing.go:285` |
| Supported formats; anything else is stored `not_filed` | `ReasonFor`, `formatOf`, `internal/intake/run.go:27` |
| A Markdown file that starts `From:`, `To:`, `Subject:`, `Date:` is treated as an email: title from the subject, date from the header | `markdownEmail`, `internal/intake/markdown_identity.go:25`, `:119` |
| Profile reading per document: `auto`, `read`, `skip` | `SetProfileReading`, `internal/store/reading.go:28` |
| A signed webhook with no session | `stripeWebhook`, `internal/httpapi/billing.go:247` |
| Project create | `createProject`, `internal/httpapi/projects.go:485`; `Store.CreateProject`, `store.go:245` |
| Project delete, and how unreferenced bytes are removed | `internal/store/delete.go:138`; `deps.Blobs.Remove` with `Store.BlobReferenced` in `internal/httpapi/projects.go:545` |
| Events to the browser | `broker.Commit`, used in `internal/httpapi/server.go:445` |
| Mail settings and the live provider | `internal/config/config.go`, `docs/deployment.md` (Mailgun) |

Mailgun facts this plan relies on, from its documentation:

- A route's `forward()` to a URL posts the parsed message:
  `application/x-www-form-urlencoded`, or `multipart/form-data` when there
  are attachments. Fields: `recipient`, `sender` (the envelope sender),
  `from`, `subject`, `body-plain`, `body-html`, `stripped-text`,
  `message-headers`, `attachment-count`, `attachment-1`…, `content-id-map`,
  `timestamp`, `token`, `signature`.
- The signature is hex HMAC-SHA256 of `timestamp` followed by `token`, keyed
  with the account's HTTP webhook signing key.
- A 200 response is success. 406 is a rejection with no retry. Any other
  code is retried for 8 hours.
- Inbound SPF and DKIM results are the headers `X-Mailgun-Spf` (Pass,
  Neutral, Fail, SoftFail) and `X-Mailgun-Dkim-Check-Result` (Pass, Fail);
  `X-Mailgun-Sflag: Yes` marks spam. Spam filtering must be switched on for
  the domain for these to appear.

Sources: <https://documentation.mailgun.com/docs/mailgun/user-manual/receive-forward-store/receive-http>,
<https://documentation.mailgun.com/docs/mailgun/user-manual/webhooks/securing-webhooks>,
<https://documentation.mailgun.com/docs/mailgun/user-manual/receive-forward-store/route-filters>,
<https://documentation.mailgun.com/docs/mailgun/user-manual/receive-forward-store/spam-filter>.

---

### Task 0: Confirm Mailgun's behaviour (gate, needs the owner's Mailgun login)

No code. The owner, or an agent with the owner present, does this in the
Mailgun control panel. Use dummy content only: the test posts go to a
third-party inspection page.

**Step 1:** Decide the receiving domain. Recommended: `in.sitewise.au`, with
its own MX records from Mailgun's panel, so the root domain stays free for
ordinary mailboxes later. The alternative is the root domain with a prefix
(`p-…@sitewise.au`) and no DNS change.

**Step 2:** Switch on spam filtering for that domain. Create a temporary
route that forwards to Mailgun's inspection tool (linked from the
receive-http page).

**Step 3:** Send four messages to a test address on the domain: one from
Gmail with a PDF; one from the owner's work mailbox with a PDF and its usual
signature logo; one HTML-only message; one with a forged `From`.

**Step 4:** Record in this plan, under "Task 0 findings":

1. Whether `X-Mailgun-Spf` and `X-Mailgun-Dkim-Check-Result` appear in
   `message-headers` on a forwarded post, and their values for the forged
   message.
2. Whether `body-plain` is filled for the HTML-only message.
3. The largest message Mailgun accepts, and how `recipient` is written when
   the address was in Bcc.

**Gate:** if finding 1 is "no", stop. The sender check in Task 3 cannot be
built as written. Bring the owner the two choices: verify DKIM in Go from a
raw-MIME route (a new dependency and a MIME parser), or accept the private
address plus a member's `From` as enough (weaker: someone who knows both can
forge a message).

### Task 0 findings: not yet run

Nothing below has been observed on a real Mailgun post. Since the security
review of 19 November 2026 the sender check no longer reads any header the
sender can write (SPF and `X-Mailgun-*` results are not trusted): a sender
is authenticated only by a DKIM signature that verifies in Go over the raw
message and is aligned with the From domain's registrable domain. Anything
else is held as "Could not confirm the sender", never filed. Each fact sits
in one function, so a different finding changes that function and its
tests and nothing else.

| # | What the owner records (Step 3's messages: Gmail with a PDF, the work mailbox with its logo, HTML-only, a forged From) | Function it touches | Today's assumption |
|---|---|---|---|
| 1 | The route's forward URL ends in `mime` (`https://sitewise.au/api/mail/inbound/mime`) and the post carries the raw message in `body-mime` | `mailin.RawField`, `readMailPost` | As Mailgun documents the mime variant |
| 1b | The raw bytes are exact enough for DKIM to verify: the Gmail and work messages are filed (authenticated), the forged one held | `mailin.Verify` | Verifies; if not, members' mail is held until fixed |
| 2 | `body-plain` is filled for the HTML-only message (parsed route only) | `mailin.PlainBody` | The raw message is read first; HTML's text is the fallback |
| 3a | The largest message Mailgun accepts and posts | `mailin.MaxPostBytes` (40 MB) | Mailgun's documented 25 MB message limit, plus encoding |
| 3b | How `recipient` is written when the address was in Bcc, and when two project addresses share one message | `mailin.RecipientLocalPart` | The address received for, comma-separated when several |
| 4 | Whether a retry reuses its first signature or is signed afresh | `mailin.FreshFor` (9 h) | Reused; if signed afresh, 15 minutes |
| 5 | Where the small fields sit relative to file parts in a multipart post | `readMailPost` | Signature fields first; otherwise files are spooled, then checked |
| — | Spam filtering is on for the receiving domain, so `X-Mailgun-Sflag` is present | `mailin.Dropped` | Present when on; any "Yes" drops |
| 6 | Whether the raw route's `body-mime` carries `X-Mailgun-Sflag` (Mailgun may add it to the parsed fields only) | `mailin.Dropped`, `readMessage` | Carried in `body-mime`. If not, spam is not dropped but still never filed unless DKIM-authenticated; a sender writing the header can only drop its own mail |
| 7 | A DKIM-signed member message (Gmail, the work mailbox) signs From, Subject, Message-ID, Date and Content-Type, and In-Reply-To on a reply | `mailin.Trust` | Gmail and Microsoft 365 sign all of them; a signature without them holds the message, without In-Reply-To the reply does not thread |

Also for the owner: the Caddy source-address limit on `/api/mail/inbound*`
from Mailgun's published ranges (a go-live gate in Task 11), and a signing key used for this webhook
only (`docs/deployment.md`, "Project email"). Mail goes out from
`SITEWISE_MAIL_FROM` as "SiteWise" (`fromAddress`), with Reply-To the PM and
the project address.

### Task 1: Worktree and baseline

Create a worktree on a new branch `mail/intake` (superpowers:using-git-worktrees),
`git merge main`, copy the private data the gate needs, run
`tools/check.ps1`, and note the bench figures for project create, project
open and `whole_intake`.

Run Go tests with:

```powershell
$env:SITEWISE_TEST_DATABASE_URL = 'postgres://sitewise@127.0.0.1:5433/sitewise_test?sslmode=disable'
$env:PATH = "$PWD\.tools\go\bin;$env:PATH"
go test -p 1 ./internal/httpapi -run <Name> -count=1
```

### Task 2: Addresses and the mail tables

**Files:**
- Create: `internal/db/migrations/056_mail_intake.sql` (take the next free
  number), `internal/store/mail.go`
- Modify: `internal/store/store.go:245` (`CreateProject`),
  `internal/store/delete.go` (`DeleteProject`), the query behind
  `Store.BlobReferenced` (*verify* its file)
- Test: `internal/store/mail_test.go`

**Step 1: Write the failing tests.**

- `TestProjectGetsMailAddress`: `CreateProject` then `MailAddress` returns a
  local part matching `^[a-z0-9-]+-[0-9a-f]{10}$` that starts with the
  project's name in lower case. Two projects named the same get different
  addresses. `ProjectByMailAddress` finds the project in any letter case and
  returns `ErrNotFound` for an unknown address.
- `TestRotateMailAddress`: after `RotateMailAddress` the old local part is
  not found and the new one is.
- `TestBlobReferencedCountsHeldMail`: a hash named only by a `mail_parts`
  row is referenced; after the message row is deleted it is not.
- `TestDeleteProjectRemovesMail`: deleting the project removes its address,
  messages and parts, and lists the held parts' hashes in
  `DeleteResult.Unreferenced`.

**Step 2:** Run. Expected: FAIL, `MailAddress` undefined.

**Step 3: Write the migration.**

```sql
-- One inbound address per project, made when the project is. The random
-- part keeps the address private; a member can rotate it.
CREATE TABLE project_mail_addresses (
 org_id uuid NOT NULL,
 project_id uuid NOT NULL,
 local_part text NOT NULL CHECK (local_part ~ '^[a-z0-9-]{12,64}$'),
 created_at timestamptz NOT NULL DEFAULT now(),
 PRIMARY KEY (org_id,project_id),
 UNIQUE (local_part),
 FOREIGN KEY (org_id,project_id) REFERENCES projects(org_id,id) ON DELETE CASCADE
);
INSERT INTO project_mail_addresses(org_id,project_id,local_part)
SELECT org_id,id,
 COALESCE(NULLIF(trim(both '-' from left(regexp_replace(lower(name),'[^a-z0-9]+','-','g'),24)),''),'project')
 || '-' || substr(replace(gen_random_uuid()::text,'-',''),1,10)
FROM projects;

-- One row per message received for a project. A held message waits for a
-- member; a discarded one is deleted.
CREATE TABLE mail_messages (
 org_id uuid NOT NULL,
 id uuid NOT NULL,
 project_id uuid NOT NULL,
 message_id text NOT NULL,
 webhook_token text NOT NULL,
 sender text NOT NULL,
 sender_name text NOT NULL DEFAULT '',
 subject text NOT NULL DEFAULT '',
 received_at timestamptz NOT NULL DEFAULT now(),
 status text NOT NULL CHECK (status IN ('filed','held')),
 held_reason text NOT NULL DEFAULT ''
  CHECK (held_reason IN ('','unknown_sender','unverified_sender','plan_limit','archived','held_full')),
 decided_by uuid,
 PRIMARY KEY (org_id,id),
 UNIQUE (org_id,project_id,message_id),
 UNIQUE (webhook_token),
 FOREIGN KEY (org_id,project_id) REFERENCES projects(org_id,id) ON DELETE CASCADE,
 CHECK (status <> 'held' OR held_reason <> '')
);
CREATE INDEX mail_messages_held ON mail_messages(org_id,project_id,received_at DESC) WHERE status='held';

-- The parts of a message: 0 is the body document, then each attachment.
-- A skipped part keeps its name and the reason, not its bytes.
CREATE TABLE mail_parts (
 org_id uuid NOT NULL,
 message_id uuid NOT NULL,
 n integer NOT NULL CHECK (n >= 0),
 filename text NOT NULL,
 sha256 bytea CHECK (octet_length(sha256)=32),
 byte_size bigint NOT NULL DEFAULT 0,
 skipped text NOT NULL DEFAULT '' CHECK (skipped IN ('','inline_image','calendar','winmail','not_kept')),
 document_id uuid,
 PRIMARY KEY (org_id,message_id,n),
 FOREIGN KEY (org_id,message_id) REFERENCES mail_messages(org_id,id) ON DELETE CASCADE,
 FOREIGN KEY (org_id,document_id) REFERENCES documents(org_id,id) ON DELETE SET NULL (document_id),
 CHECK ((skipped = '') = (sha256 IS NOT NULL))
);
CREATE INDEX mail_parts_sha ON mail_parts(sha256) WHERE sha256 IS NOT NULL AND document_id IS NULL;
```

**Step 4: Implement** in `internal/store/mail.go`: `MailAddress`,
`ProjectByMailAddress` (returns org and project ids and whether the project
is archived), `RotateMailAddress`, and `newMailLocalPart(name)` (the same
slug rule as the migration, then ten hex characters from `crypto/rand`).
`CreateProject` inserts the address in its existing transaction.
`BlobReferenced` also counts `mail_parts` rows with no `document_id`.
`DeleteProject` adds those hashes to `res.Unreferenced` before the project
row goes.

**Step 5:** Run the tests and
`go test -p 1 ./internal/store ./internal/httpapi -run "Project|Delete" -count=1`.
Expected: PASS.

**Step 6:** Commit: `Mail intake: a private address per project, and the mail tables`.

### Task 3: Read and judge one message (pure functions)

**Files:**
- Create: `internal/mailin/mailin.go`
- Test: `internal/mailin/mailin_test.go`

No database and no HTTP here: small functions over values, so the untrusted
input is handled in one place that is easy to test.

**Step 1: Write the failing tests** (table tests):

- `TestVerifySignature`: a known key, timestamp and token verify; a changed
  token, a changed timestamp, an empty key and an upper-case signature are
  handled (the last one verifies).
- `TestSenderAddress`: `Jo Chen <Jo.Chen@Example.com>` gives
  `jo.chen@example.com` and the name; two addresses, no address and a
  malformed header give an error.
- `TestAuthenticated`: with From `pm@example.com`, each row states headers,
  the envelope sender, and the expected result:

  | SPF header | Envelope sender | DKIM header | `DKIM-Signature d=` | Result |
  |---|---|---|---|---|
  | Pass | `pm@example.com` | none | none | true |
  | Pass | `bounce@mail.example.com` | none | none | true |
  | Pass | `x@attacker.net` | none | none | false |
  | Fail | `pm@example.com` | Pass | `example.com` | true |
  | none | `pm@example.com` | Pass | `attacker.net` | false |
  | none | `pm@example.com` | none | none | false |
  | SoftFail | `pm@example.com` | Fail | `example.com` | false |

- `TestDropped`: `X-Mailgun-Sflag: Yes`, `Auto-Submitted: auto-replied`,
  `Precedence: bulk` and `X-Autoreply` are dropped; `Auto-Submitted: no` and
  an ordinary message are not.
- `TestSkipPart`: an image under 100 KB whose content id is in
  `content-id-map` is `inline_image`; a larger inline image is kept; `.ics`
  and `text/calendar` are `calendar`; `winmail.dat` is `winmail`; a PDF is
  kept.
- `TestBodyDocument`: the Markdown starts with `From:`, `To:`, `Subject:`
  and `Date:` lines in the layout of the email fixture in
  `internal/intake/markdown_identity_test.go`, the date is written
  `14 November 2026`, a subject with a line break or a leading `#` cannot
  add a header line or a heading, and the filename is
  `<subject> (email 2026-11-14).md`, cleaned and at most 255 bytes, or
  `Email 2026-11-14.md` when the subject is empty.

**Step 2:** Run. Expected: FAIL, package does not exist.

**Step 3: Implement.**

```go
// VerifySignature checks Mailgun's HMAC over timestamp then token.
// https://documentation.mailgun.com/docs/mailgun/user-manual/webhooks/securing-webhooks
func VerifySignature(key, timestamp, token, signature string) bool {
	if key == "" || timestamp == "" || token == "" {
		return false
	}
	mac := hmac.New(sha256.New, []byte(key))
	mac.Write([]byte(timestamp + token))
	want := hex.EncodeToString(mac.Sum(nil))
	return hmac.Equal([]byte(want), []byte(strings.ToLower(signature)))
}

// Message is one parsed inbound post, before any decision.
type Message struct {
	Recipient, EnvelopeSender string
	FromAddress, FromName     string
	To, Cc, Subject           string
	Date                      time.Time // zero when the header is missing or unreadable
	MessageID                 string
	Headers                   [][2]string
	BodyPlain                 string
	InlineIDs                 map[string]bool // attachment field names in content-id-map
}
```

`Authenticated(m)` is true when SPF passed and the envelope sender's domain
matches the From domain, or DKIM passed and a `DKIM-Signature` header's `d=`
matches the From domain. Two domains match when they are equal or one ends
with a dot and the other. Missing headers mean false. Write this rule in the
function comment, with the limit that Mailgun reports one DKIM result for a
message that may carry several signatures.

The body document is the header lines, a blank line, then `BodyPlain`. Use
the date as the `Date` header wrote it, without converting the zone. When
the header is missing, use the time received.

**Step 4:** Run the tests. Expected: PASS.

**Step 5:** Commit: `mailin: verify, read and judge one inbound message`.

### Task 4: File bytes that are already stored (refactor)

**Files:**
- Modify: `internal/intake/upload.go:59-100`
- Test: `internal/intake/upload_test.go`

**Step 1:** Write the failing test `TestFileStoredBlob`: `blobs.Put` some
bytes, then `Uploader.File(ctx, org, project, "report.pdf", blob)` returns
the same `Filing` that `Upload` returns for the same bytes and name; an
unsupported name is committed `not_filed` with `ReasonFor`'s reason; an
empty name is `ErrFilename`.

**Step 2:** Run. Expected: FAIL, `File` undefined.

**Step 3:** Split `Upload` into the `blobs.Put` call and a new exported
`File(ctx, orgID, projectID, filename string, blob files.Blob) (Filing, error)`
(*verify* the type `blobs.Put` returns) that does everything after the put,
including `ReasonFor`. `Upload` calls `File`. No behaviour changes for a drop.

**Step 4:** Run `go test -p 1 ./internal/intake -count=1`. Expected: PASS.

**Step 5:** Commit: `intake: file a blob that is already stored`.

### Task 5: Record, list and decide mail in the store

**Files:**
- Modify: `internal/store/mail.go`
- Test: `internal/store/mail_test.go`

**Step 1: Write the failing tests.**

- `TestRecordMailIsIdempotent`: the same webhook token twice, and the same
  `Message-Id` for the project twice, each record one message and report
  the repeat.
- `TestHeldMailCaps`: the 51st held message for a project, or held bytes
  past 500 MB, is recorded with reason `held_full` and every part
  `not_kept`.
- `TestHeldMailExpires`: recording a message deletes that project's held
  messages older than 30 days and returns their hashes for removal.
- `TestHeldMailIsScoped`: `HeldMail`, `MarkMailFiled` and `DiscardMail` with
  another org, or another project of the same org, return `ErrNotFound`.

**Step 2:** Run. Expected: FAIL.

**Step 3: Implement** `RecordMail` (one transaction under `lockProject`:
expire, check repeats, apply the caps, insert the message and parts),
`HeldMail` (newest first, with parts), `MarkMailFiled` (status `filed`,
`decided_by`, each part's `document_id`), `SetMailHeld` (a message that hit
the plan limit while filing), and `DiscardMail` (delete, returning the
hashes).

**Step 4:** Run the tests. Expected: PASS.

**Step 5:** Commit: `Store: record, hold and decide inbound mail`.

### Task 6: The webhook

**Files:**
- Create: `internal/httpapi/mail.go`
- Modify: `internal/httpapi/projects.go` (route `POST /mail/inbound`,
  `routePaths` name `mail_inbound_accept`), `internal/httpapi/billing.go`
  (split the plan check out of `uploadAllowed` into
  `uploadAllowance(ctx, deps, orgID) (ok bool, message string, err error)`;
  `uploadAllowed` keeps its verified-email check and calls it),
  `internal/httpapi/server.go` and `cmd/sitewise/main.go` (wiring),
  `internal/config/config.go`, `deploy/sitewise.env.example`,
  `bench/budgets.json`
- Test: `internal/httpapi/mail_test.go`, `internal/config/config_test.go`

Config: `SITEWISE_MAIL_IN_DOMAIN` and `SITEWISE_MAILGUN_SIGNING_KEY`. Both
set or both blank; one without the other is a load error. Blank means the
feature is off: the webhook answers 404 and no address is shown. The key
never appears in an error or a log line.

**Step 1: Write the failing tests** with a helper that builds a signed post
(`multipart/form-data`, or urlencoded when there are no files):

| Case | Expected |
|---|---|
| Member's message, two PDFs, SPF aligned | 200; three documents in the project (the body `.md` and both PDFs); the body document's profile reading is `skip`; a `mail` event is written; the month's document count rose by three |
| The same post again (same token) | 200; still three documents |
| A new token, same `Message-Id` | 200; still one message |
| No attachments (urlencoded) | 200; one document |
| Sender is not a member of the project's org | 200; no documents; one held message, reason `unknown_sender` |
| Sender is a member of a different org | Held, `unknown_sender` |
| Member whose email is not confirmed | Held, `unknown_sender` |
| Member, but SPF and DKIM absent or not aligned | Held, `unverified_sender` |
| Org is at its monthly document limit | Held, `plan_limit`; nothing filed |
| Project is archived | Held, `archived` |
| Bad signature | 401; nothing stored |
| Timestamp older than 24 hours | 406; nothing stored |
| Address not known, or at another domain | 200; nothing stored |
| Spam flag, or an auto-reply | 200; nothing stored |
| Signature logo inline, under 100 KB | Its part is recorded as `inline_image`; no document |
| An `.exe` attachment | A document stored `not_filed`, as a dropped one is |
| Body larger than the 40 MB cap | 413; nothing stored |
| Feature not configured | 404 |
| The database fails while recording | 500, so Mailgun retries; a retry then succeeds once |

**Step 2:** Run. Expected: FAIL with 404.

**Step 3: Implement `mailInbound`**, in this order:

1. Not configured: 404.
2. Take one of two slots; none free: 503 (Mailgun retries).
3. `http.MaxBytesReader` at 40 MB (raise or lower to Task 0's finding), then
   `r.ParseMultipartForm(8 << 20)`; `http.ErrNotMultipart` is not an error.
   `defer r.MultipartForm.RemoveAll()`.
4. `mailin.VerifySignature`. Bad: 401. Old timestamp: 406.
5. Find the recipient at the configured domain and the project. None: 200.
6. Dropped kinds (spam, auto-reply): 200.
7. Decide: `archived`, then sender (a confirmed member of this org with
   that address, else `unknown_sender`), then `mailin.Authenticated`
   (else `unverified_sender`).
8. Build the parts: the body document when the body has text, then each
   attachment that `SkipPart` keeps. `deps.Blobs.Put` each kept part.
9. `RecordMail`. A repeat: 200. Remove any expired hashes it returned.
10. Accepted: for each part, check `uploadAllowance` (over: `SetMailHeld`
    with `plan_limit` and stop), `Uploader.File`, set the body document's
    profile reading to `skip`, count the document when it was created, and
    `deps.Filer.Start` when it is pending. Then `MarkMailFiled`.
11. Write a `mail` event for the org and wake the broker. Log one line:
    org, project, outcome, part count. Never log the subject, addresses or
    the key.
12. 200.

Add the budget row
`{"name":"mail_inbound_accept","p50_us":300000,"p90_us":1000000,"note":"signed Mailgun post of a 5 MB message with two PDFs, to the 200 response; filing of each document runs after and keeps its own budget; no Jev here"}`.

**Step 4:** Run the tests. Expected: PASS.

**Step 5:** Commit: `Mail intake webhook: verify, hold or file`.

### Task 7: Read, accept, discard, rotate

**Files:**
- Modify: `internal/httpapi/mail.go`, `internal/httpapi/projects.go`
  (routes and `routePaths`), `bench/budgets.json`
- Test: `internal/httpapi/mail_test.go`

Routes, all with `originOK`, `memberSession` and `visibleProject`:

- `GET /projects/{id}/mail` returns
  `{"address":"petersham-…@in.sitewise.au","held":[{"id","sender","sender_name","subject","received_at","reason","parts":[{"filename","byte_size","skipped"}]}]}`.
  With the feature off, `address` is empty.
- `POST /projects/{id}/mail/{message}/accept` files a held message as Task 6
  step 10 does, with the caller as `decided_by`. A message held as
  `held_full` has nothing to file and returns 409.
- `POST /projects/{id}/mail/{message}/discard` deletes it and removes bytes
  nothing else references.
- `POST /projects/{id}/mail/address` rotates the address.

**Step 1: Write the failing tests:** accept turns a held message into
documents and empties the list; accept over the plan limit returns 402 with
the plan's message and leaves it held; discard removes the message and its
unshared bytes, and keeps bytes a filed document also uses; a member of
another org gets 404 on every route; a malformed id gets 404; after rotate,
a post to the old address stores nothing and a post to the new one files.

**Step 2:** Run. Expected: FAIL with 404.

**Step 3:** Implement. Budget rows `mail_read` (100/250 ms) and
`mail_decide` (100/250 ms).

**Step 4:** Run the tests. Expected: PASS.

**Step 5:** Commit: `Mail intake: held list, accept, discard, rotate`.

### Task 8: The body document files as an email

**Files:**
- Test: `internal/intake/markdown_identity_test.go`

**Step 1:** Write `TestMailBodyDocumentFilesAsEmail`: pass the output of
`mailin`'s body document through the Markdown extractor and
`markdownIdentity`. Assert the title candidate is the subject and the date
candidate is the header date; for a forwarded thread with an older quoted
header block, the date is the latest one.

**Step 2:** Run. If it fails, change the body document's layout in
`internal/mailin`, not the intake rules.

**Step 3:** Commit: `Mail body documents are recognised as emails by intake`.

### Task 9: The address and held mail in the repository (lane B)

**Files:**
- Modify: `web/src/api.ts`, `web/src/Project.tsx` (the repository panel)
- Create: `web/src/MailIn.tsx`
- Test: `web/tests/mail-in.spec.ts`, and the e2e server's fixtures
  (`web/tests/e2eserver/main.go`)

Read `web/DESIGN.md` and the register's existing rows first. Do not restyle
the repository.

- One quiet line at the head of the repository: "Email documents to" then
  the address, with a copy button. Nothing shows when the address is empty.
- When mail is held: a count on that line that opens a list. Each row shows
  the sender, the subject, when it arrived, its attachments by name, and the
  reason in plain words ("Not from your team", "Could not confirm the
  sender", "Monthly document limit reached", "Project is archived", "Held
  mail is full"). Two actions: Accept and Discard. Discard asks once.
- The list is read with `GET /projects/{id}/mail` after the documents list,
  so opening a project is no slower, and again on a `mail` event.

**Step 1:** Write the failing browser test: the address shows and copies; a
held message lists with its reason; Accept makes its documents appear in the
register; Discard removes it; with the feature off the line is absent.

**Step 2 to 4:** Run it red, build, run it green, run the existing
repository browser tests.

**Step 5:** Commit: `Repository: the project's email address and held mail`.

### Task 10: Bench, gate, documents

**Files:**
- Modify: `cmd/intake-bench` (*verify* where upload is benched and follow
  it), `AGENTS.md`, `PRODUCT.md`, `docs/deployment.md`, `docs/operations.md`

**Step 1:** Bench `mail_inbound_accept`, `mail_read` and `mail_decide`
through the real server. Run the bench.

**Step 2:** Run `tools/check.ps1`. Expected: PASS.

**Step 3:** `AGENTS.md`: add to the paths that do not call Jev: "The mail
webhook, the held-mail list, accept, discard and rotating the address do not
call Jev; each accepted document is filed as a dropped one is." Add this
plan under "Before changing one of these surfaces". `PRODUCT.md`: one
paragraph on the address. `docs/deployment.md` and `docs/operations.md`: the
receiving domain, the route, the two settings, and that the key is the
account's webhook signing key. No secret in any file.

**Step 4:** Add a "Result" section here: what was built, the tests, timing
against the three budgets, and project create, project open and
`whole_intake` before and after. Report in plain language what calls what
and what could fail in production. Commit.

### Task 11: Go live (owner steps, after merge and redeploy)

1. DNS at Crazy Domains: the MX records Mailgun's panel shows for the
   receiving domain.
2. Mailgun: spam filtering on for that domain; one route,
   `match_recipient(".*@in\.sitewise\.au")` with
   `forward("https://sitewise.au/api/mail/inbound/mime")` and `stop()`.
3. **Gate:** the Caddy rule limiting `/api/mail/inbound*` to Mailgun's
   published source ranges is in place (`docs/deployment.md`, "Project
   email"), and a post from any other address answers 404. Mail does not
   go live without it.
4. `/etc/sitewise/sitewise.env`: `SITEWISE_MAIL_IN_DOMAIN` and
   `SITEWISE_MAILGUN_SIGNING_KEY`. Restart. `SITEWISE_MAIL_SEND` stays off
   while Stripe is in test mode.
5. From the owner's mailbox, send a message with a PDF to a project's
   address. It files. From a personal address that is not on the team, send
   another. It is held. Accept it.

---

## Result (19 November 2026)

Built on `wave/mail` in the worktree `sitewise-agency-docs`, from local
`main` at 6f563f2; migrations 075 (this plan) and 076 (correspondence).
Everything was run against a private schema of `sitewise_test`, hand-written
posts in Mailgun's documented shape signed with a test key, a stand-in Jev
and a mail sink. **Nothing was run against real Mailgun**; Task 0 and Task 11
remain the owner's. Differences from this plan, each the smallest change
that keeps its intent (the evidence note lists them with reasons):

- The local part is `p-` and 32 random hex characters, not the project's
  name: an address never names the org or project.
- Every refusal (bad signature, stale or future timestamp, unreadable or
  over 40 MB) answers the same 406 "not accepted", not 401 or 413: one
  answer, and Mailgun does not retry a refusal.
- Every signed token is recorded (`mail_webhook_tokens`), not only those of
  stored messages: Mailgun signs only timestamp and token, so a captured
  post could otherwise be replayed with another body.
- Held reason `busy`: more than 60 filed messages a project an hour are held,
  so a flood cannot spend Jev reads or the allowance.
- The plan limit is checked once per message before it is recorded (as an
  upload's is checked before its bytes); a message never goes from filed
  back to held.
- Accept is a prepared read for chat (A4, plan 0 decision 1); discard waits
  for a click; rotate is the PM's on the screen.
- `GET /projects/{id}/mail` returns `{address, held, messages}` (the 15
  November update), not `{address, held: [...]}`.

Timing is in the evidence note (`TestMailBudgets`). The bench binary
(`cmd/intake-bench`) was not extended or run over the private corpus.

## Risks to watch

- **(Superseded 19 November 2026: the sender check is now DKIM verified in
  Go over the raw message; see "Task 0 findings".)** Sender checking rested on Mailgun's SPF and DKIM headers. Task 0
  settles whether they arrive. They report one DKIM result for a message
  that may carry several signatures, so an aligned pass is strong evidence,
  not proof. The address is private, held mail is never filed or read, and
  no email is ever an instruction.
- **A leaked address invites junk.** Anyone the PM copies learns the
  address. Their mail is held, capped at 50 messages and 500 MB a project,
  and expires after 30 days. Rotating the address cuts it off.
- **Filing accuracy on email bodies is unmeasured.** The intake answer key
  holds no emails. The body files under the existing questions; a wrong
  kind or discipline is corrected in the register as for any document.
- **Mailgun retries for 8 hours.** A message is recorded once per webhook
  token and once per `Message-Id`, so a retry after a slow response does
  not file twice.
- **The original message is not kept.** If the owner wants the `.eml` in
  the register, that is a raw-MIME route and a parser: a follow-up.

## Follow-ups, not in this plan

- A Jev question for the kind of email (transmittal, instruction, query,
  quote cover, notice), with an answer key from about 50 real emails.
- Message bodies as profile evidence that lands "to check" (design 8.4).
- An allow-list of outside senders per project.
- Unpacking a message attached to a message; keeping the original `.eml`.
- A receipt to the sender.

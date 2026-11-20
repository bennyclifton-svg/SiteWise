# Guest access by link: implementation plan

Date 17 November 2026; revised 18 November. Lane C (security), holding lane A's org-isolation
rules. Status: built 18 November 2026 on branch `wave/guests` (not merged, not
deployed); what was built, measured and left open is in
`docs/evidence/2026-11-18-guest-access.md`. Needs its security review before
it ships (index, build order).

**Revised 18 November** after the review in `2026-11-17-next-wave-index.md`.
Three kinds of link only (documents, report, package): grants for RFI
answers and comments are gone, with the `respond` and `item_id` columns.
Chat reaches everything ("From chat"). The earlier text is at git ref
`refs/backup/next-wave-plans-2026-11-18`.

## In one page (product requirements)

**Outcome.** The PM shares one thing with one person outside the org, by
link: a set of documents, an issued report, or (for a firm invited on a
Procurement row) the issued RFP or RFT with a place to send its quote. The
person opens the link, sees only that thing on one small page, downloads
it, and, if a firm, drops its quote. The quote lands on that firm's invited
cell and is filed and read exactly as a PM drop there is. On each shared
thing the PM sees who has a link, how often and when they last opened it,
and revokes it in one click. Nothing else in SiteWise answers to a link.

**Why.** Time to decision: quotes now arrive by email and get dragged onto
the row; a firm that drops its own quote puts it on the right cell at once.
Answer trust: a guest sees only issued, frozen material, never a draft, a
price or another firm.

**Where it lives.** No new left-nav entry. "Share…" in the register's
right-click menu (`web/src/Register.tsx:302`); "Share…" beside "Issued
reports" (`web/src/Reports.tsx:268`); a link button on an invited firm's
cell, both Procurement tabs (`InvitedFirm`, `web/src/Procurement.tsx:579`),
which then reads "Invited · link opened 14 Nov". One new screen, outside
the workbench: the guest page at `/g` (Trace sheet, no nav, no account
menu). A guest cannot use the workbench, so no existing surface can hold it.

**Link model: recommend (a), a link with no account.**

| | (a) Link, no account | (b) Light guest account by email |
|---|---|---|
| Friction for a busy subcontractor | Click, look, drop the quote | A password per builder |
| Speed | One indexed lookup per request | Same, plus sign-in |
| Forwarded link | Holder gets that one thing (Edge cases) | Useless without the password |
| Fit with today's accounts | A separate path; touches nothing | One person is one org (`internal/store/team.go:66`); a guest in `users`/`memberships` passes `memberSession` (`internal/httpapi/projects.go:600`), which opens every project in the org, and counts as a seat |

"Signed" is done as team invites and resets are: a random 256-bit token kept
only as its SHA-256 (`internal/auth/tokens.go`); revoke and "last opened"
need the row anyway. We reuse the token, fragment link, rate limiter and
cookie shape, but **not** `invites`, `sessions` or `memberships`: every
member handler trusts those, so a guest in them would be a leak everywhere.

**What reads what.** Jev: no new question. A guest's quote takes the
existing upload, filing (one identity fan-out) and proposal fan-out after
filing (`proposalReader.Filed`, `internal/httpapi/submissions.go:70`), with
the same question versions and withheld thresholds
([patterns](https://docs.typesafe.ai/patterns): one fan-out per state;
[confidence](https://docs.typesafe.ai/confidence): bands unchanged); a test
proves the call count equals a PM drop's. LLM: none new; the tender pass
follows only when the org has switched it on, as today. Go: the grant check,
what a grant may fetch, upload limits, the change-log record.

**Speed budgets.**

| Path | p50 / p90 ms | Bench row |
|---|---|---|
| Guest page open (grant check + page read); link exchange | 100 / 250 | new `guest_open` |
| Share create, list, revoke | 100 / 250 | `team` ("team list, invite, revoke, remove") |
| Guest quote, upload to filed / to read | 1,000 / 2,000; 3,000 / 6,000 | `whole_intake`; `proposal_read` (unchanged path) |
| Procurement board with link state | 150 / 300 | `procurement_board` |
| Guest download | not timed (streams, as `downloadDocument`) | its grant check is `guest_open`'s |

**Done means.**
1. Three drawings, Share…, client's email, Copy link. The client sees three
   rows and downloads each; a fourth drawing's id in the URL gives 404.
2. On the Roofing RFT row, the link button on "Acme Roofing · Invited";
   Acme downloads the RFT and its transmittal drawings and drops
   `quote.pdf`. The PM's cell goes to "Reading proposal…" then Acme's total.
   Acme sees "Received quote.pdf, 17 Nov 2026" and nothing more.
3. PM revokes; Acme's next click: "This link has expired or been withdrawn."
4. The guest sweep passes. Seats and limits read the same with 50 links.

**Out of scope.** Guest accounts and sign-in. Guests seeing prices, other
firms, the board, cost plan, program, profile, chat or live events. A guest
answering an RFI or commenting on a document (a consultant replies to the
email; plan 1 threads it and plan 4 matches it). Sending mail (plan 1). The response form for guests. An
org-wide list of links. A tender close time apart from link expiry.

**Owner decisions** (recommended default in brackets). The plan takes its
defaults for 2, 3, 5, 7, 8, 9 and 10; the owner decides 1, 4 and 6.
1. Link model. [(a) link, no account.]
2. Link life. [Documents and reports 30 days, RFP/RFT 21; PM sets 1 to 180.]
3. Who shares and revokes. [Any org member, as any member drops quotes now.]
4. Reports and addenda. [A report link shows every issued version, newest
   first, never a draft; a new RFT issue reaches every firm holding a link.
   Alternative: pin the link to the issue shared.]
5. Response form workbook. [Not offered in v1: `ReadResponseForm`
   (`internal/store/procurement_schedule.go:298`) builds it from the live
   draft, so a firm could see unissued edits. Later: freeze it at issue.]
6. Sending the link. [Copy link only, into the PM's own email; plan 1's
   transmittal carries it once built. Alternative: SiteWise mails it as
   `SendTeamInvite` does, about 30 lines.]
7. Guest upload limits. [PDF, Word, Excel; 50 MB; 20 a link an hour; counted
   toward the org's documents and storage as a PM drop is.]
8. Quotes after award. [Refused: "This tender has been awarded."]
9. Archived project. [Its links pause. `projects.archived_at` gains a reader.]
10. Forwarded links. [Show opens and last opened; no lock to one browser.]

## What we reuse

| Need | Existing thing | File |
|---|---|---|
| Token, hash at rest; fragment link | `auth.NewToken`/`HashToken`; invite `token=` handling | `internal/auth/tokens.go`, `web/src/App.tsx` |
| Rate limits, client IP, cookie | `authLimits.allow`, `clientIP`, `setSessionCookie` | `internal/httpapi/accounts.go:284`, `:253`, `:240` |
| Upload, filing, read | as `uploadSubmission`: `Uploader.Upload`, `Filer.Start`, `ProposalReader.Start` | `internal/httpapi/submissions.go:461` |
| Quote on an invited cell | `OpenSubmissionFor` with a firm, `presetSubmissionFirm` | `internal/store/submissions.go:97`, `:165` |
| Invited firms | `package_invitations` (052), `boardInvitations` | `internal/store/package_invitations.go:228` |
| Billing allowance and count | `uploadAllowed`, `countDocument` | `internal/httpapi/billing.go:295`, `:357` |
| Document identity, issues, RFT of a package | `reportDocumentScheduleSelection`, `reportHistory`, `packageRFPReport` | `internal/store/report_documents.go:15`, `report_history.go:18`, `report_rfp_group.go:103` |
| Frozen transmittal ids; issued PDF | block basis `document_ids`; `ReadIssuedReport` | `internal/rfp/sections.go:155`, `internal/store/report_issue.go:171` |
| Download writer | body of `downloadDocument`, `downloadReport` | `internal/httpapi/documents.go:19`, `report_issue.go:46` |
| Change log; undo only by its actor | `project_changes` (056), `UndoChange` | `internal/store/change.go:14`, `change_undo.go:56` |
| PM route checks | `deliverySession` | `internal/httpapi/delivery.go:10` |
| Route table, classing, timing | `apiRoutes`, `routePaths`, `noOperation` | `internal/httpapi/projects.go:137`, `:329`, `ops_routes.go` |
| Isolation sweep | `TestNextWaveRouteAccessSweep` | `internal/httpapi/access_sweep_test.go:29` |
| Dialog; sheet and band | `ConfirmDialog` (plain tone, children); `trace-home`, `TraceBand` | `web/src/ConfirmDialog.tsx`, `web/src/TracePage.tsx` |

## What is new

| Thing | Why an existing one cannot carry it | Est. lines |
|---|---|---|
| Table `guest_grants` | `invites`/`sessions`/`memberships` are trusted by every member handler and counted as seats | 45 SQL |
| Cookie `sitewise_guest`, `guestGrant()` | A second cookie name keeps the paths apart by construction: member routes read only `sitewise_session` | in `guest.go` |
| 5 guest routes under `/guest` | Every existing route requires membership, and must keep doing so | 160 Go |
| 3 share routes | Team invites make members; a share makes no person at all | 80 Go |
| `ShareDialog.tsx`, `GuestPage.tsx` | No dialog makes links; the guest page is the one screen a guest sees | 310 TSX |

## Data

One migration at the next free number, `<n>_guest_grants.sql`:

```sql
-- One person outside the org sees one shared thing; an invited firm may send
-- its quote. No user, membership, invite or session: never a seat, never
-- admitted by a member route. document_ids has no key: a deleted document
-- drops off. Not captured: sharing is not an operation and is not undone.
CREATE TABLE guest_grants (
 org_id uuid NOT NULL REFERENCES orgs(id) ON DELETE CASCADE,
 id uuid NOT NULL,
 project_id uuid NOT NULL,
 token_hash bytea NOT NULL UNIQUE CHECK (octet_length(token_hash)=32),
 kind text NOT NULL CHECK (kind IN ('documents','report','package')),  -- only a package link takes a quote
 shared_with text NOT NULL CHECK (length(trim(shared_with)) BETWEEN 1 AND 254),
 document_ids uuid[] NOT NULL DEFAULT '{}' CHECK (cardinality(document_ids) <= 200),
 report_id uuid, package_id uuid, firm_id uuid,
 created_by uuid NOT NULL,
 created_at timestamptz NOT NULL DEFAULT now(),
 expires_at timestamptz NOT NULL,
 revoked_at timestamptz,
 opens integer NOT NULL DEFAULT 0,
 last_opened_at timestamptz,
 PRIMARY KEY (org_id,id),
 FOREIGN KEY (org_id,project_id) REFERENCES projects(org_id,id) ON DELETE CASCADE,
 FOREIGN KEY (org_id,created_by) REFERENCES users(org_id,id) DEFERRABLE INITIALLY DEFERRED,
 FOREIGN KEY (org_id,project_id,report_id) REFERENCES reports(org_id,project_id,id) ON DELETE CASCADE,
 FOREIGN KEY (org_id,project_id,package_id) REFERENCES packages(org_id,project_id,id) ON DELETE CASCADE,
 -- A firm's link lives and dies with its invited cell; a corrected name keeps it.
 FOREIGN KEY (org_id,package_id,firm_id) REFERENCES package_invitations(org_id,package_id,firm_id)
  ON DELETE CASCADE ON UPDATE CASCADE,
 CHECK (CASE kind
  WHEN 'documents' THEN cardinality(document_ids)>0 AND report_id IS NULL AND package_id IS NULL AND firm_id IS NULL
  WHEN 'report'    THEN report_id IS NOT NULL AND cardinality(document_ids)=0 AND package_id IS NULL AND firm_id IS NULL
  WHEN 'package'   THEN package_id IS NOT NULL AND firm_id IS NOT NULL AND report_id IS NULL AND cardinality(document_ids)=0 END)
);
CREATE INDEX guest_grants_project ON guest_grants(org_id,project_id,created_at DESC);

-- A guest's write is recorded with the grant as actor; no member can undo it.
ALTER TABLE project_changes DROP CONSTRAINT project_changes_via_check;
ALTER TABLE project_changes ADD CONSTRAINT project_changes_via_check CHECK (via IN ('chat','screen','guest'));
```

## Tasks

**1. Migration.** As above. Add the five composite keys' fixtures and
rejections to `internal/db/migrate_fk_sweep_test.go`, which inventories
every new composite reference. Test: one grant of each kind inserts; a
mixed kind is refused; removing the invitation deletes the grant;
`CorrectInvitation` (`package_invitations.go:109`) keeps it on the new firm.
Est. 45 SQL, 40 test.

**2. Store: grants.** New `internal/store/guest.go`.
`CreateGuestGrant(ctx, org, project, actor, in, tokenHash, now)` in one
transaction: project live, not archived; documents 1 to 200, all filed in
this project, else `ErrNotFound` without saying which (as
`transmittalSelection`, `report_rfp.go:306`); a report with an issue; for a
package, the invitation on a live package (as `InviteFirm` checks,
`package_invitations.go:27`) and an issued RFP/RFT; life 1 to 180 days;
`in.Replaces` revokes an older grant in the same transaction ("New link").
`GuestGrantByToken(ctx, hash, now)` is one query; unknown, revoked, expired
and archived all give `ErrNotFound`. Plus `ListGuestGrants`,
`RevokeGuestGrant` (row lock `FOR UPDATE`), `TouchGuestGrant`.
Tests `TestGuestGrantScope` (sibling-project, foreign, unissued and
uninvited targets refused, nothing written) and `TestGuestGrantLife`.
Est. 120 Go, 120 test.

**3. Store: what a grant may fetch.** Same file. `GuestPage(ctx, g)`, one
read-only transaction: project, sharer, then `reportDocumentScheduleSelection`
(corrected identity), `reportHistory`, or for a package `packageRFPReport`
+ `reportHistory`, the latest issue's transmittal ids and this firm's own
submissions here (filename and date only). `GuestDocument(ctx, g, doc)`
admits a documents grant's ids or a package's transmittal ids from any
issue; `GuestIssue(ctx, g, version)` admits only an issued version, via
`ReadIssuedReport`. Test `TestGuestFetchRules`: each kind its own and
nothing else; a draft version, another firm's quote and a non-transmittal
document give `ErrNotFound`. Est. 100 Go, 100 test.

**4. Store: the guest's quote.** Split `OpenSubmissionFor`
(`submissions.go:97`) into a wrapper and `openSubmissionTx`; no behaviour
change, every procurement test still passes (lane A).
`OpenGuestQuote(ctx, g, documentID, created, now)` in one transaction: lock
the grant `FOR SHARE` and recheck it is live (`ErrGuestGone`); refuse an
awarded package; refuse a document that existed before and is not this
firm's quote here (never re-point another firm's quote); `openSubmissionTx`
with the grant's package and firm; insert one `project_changes` row (via
`guest`, actor the grant id, area `procurement`, ops
`[{"op":"guest_quote","document_id","submission_id","filename"}]`). Test
`TestOpenGuestQuote`: firm preset as a PM drop does; change row present;
revoked, awarded and foreign-firm cases refused with that submission
unchanged; the PM's undo of the change gives 404 (`UndoChange` matches its
actor). Est. 60 Go, 90 test.

**5. HTTP: guest routes.** New `internal/httpapi/guest.go`, a `guestRoutes`
table merged in `apiRoutes` as `accountRoutes` is.
- `guestGrant(w, r, deps, write)` is **the one check every guest route
  passes**: reads only `sitewise_guest`, hashes it, `GuestGrantByToken`; a
  miss clears the cookie and gives 401 "This link has expired or been
  withdrawn."; a write checks `originOK` and the per-link limit. Org,
  project and target come from the grant, never the request.
- `POST /guest/session` {token}: 30 per IP per 15 min; cookie HttpOnly,
  SameSite=Strict, Path=/api/guest, expiring with the grant; opens + 1.
  `GET /guest`: the page; sets last opened.
- `GET /guest/documents/{doc}/file`, `GET /guest/issues/{version}/file`:
  `GuestDocument`/`GuestIssue`, then `serveFile`, one writer extracted from
  `downloadDocument` and `downloadReport`.
- `POST /guest/quote?name=`: package grants only; PDF, DOCX or XLSX by name
  before a byte is read; 50 MB `MaxBytesReader`; `orgUploadAllowed` (split
  from `uploadAllowed`, members keep the verified-email rule); `Upload` into
  the grant's project; `OpenGuestQuote`; on `ErrGuestGone` delete a
  just-created document (`internal/store/delete.go:27`) before filing, so no
  Jev call, and give 401; else `Filer.Start` or `ProposalReader.Start` and
  `countDocument` exactly as `uploadSubmission`. Answers `{filename,
  received_on}` only.
- `noOperation`: `POST /guest/session` "a guest opening a link; chat never
  signs anyone in"; `POST /guest/quote` "a guest's file upload; the guest
  page has no chat". Both `GET /guest` and the exchange time to
  `guest_open`. Est. 160 Go.

**6. HTTP: share routes and the board.** New `shares.go`:
`POST /projects/{id}/shares`, `GET /projects/{id}/shares`,
`DELETE /projects/{id}/shares/{share}` through `deliverySession`. Create
answers `{share, link}` once; link = `PublicOrigin + "/g#t=" + raw`; 100
creates per org an hour. POST is `noOperation`: "sharing outside the org
is the PM's press of Create link; chat opens Share filled in". DELETE is an
operation, `shares.revoke`, `Click: true`: a revoked link cannot be put
back. `routePaths`
to `pathTeam`. `boardInvitations` left-joins the newest live grant;
`FirmCard` (`internal/store/comparison.go:403`) gains `link_opened_on` and
`linked`. Test: the three routes join the existing sweep's list (signed out
401, wrong origin 403, foreign org 404, sibling project 404).
Est. 80 Go, 15 test.

**7. Isolation and behaviour tests.** New `guest_access_test.go`; new
`export_test.go` (package `httpapi`) exporting `RoutePatterns()` from
`apiRoutes()`.
- `TestGuestReachesOnlyItsGrant`. Fixture: the access sweep's, plus firms A
  and B invited on a package, B's priced quote, an issued RFT whose
  transmittal holds d1, a private d2, a sibling project, a foreign org. The
  client holds only A's guest cookie. (a) Every pattern outside `/guest/`,
  filled with real ids, gives 401, 403 or 404 (the public sign-in, invite,
  verify and Stripe routes are listed by name and must not give 2xx); a
  snapshot of the sweep's tables plus `submissions`, `submission_prices`,
  `package_invitations`, `guest_grants`, `project_changes` is unchanged; no
  Jev call. (b) Guest routes with ids outside the grant (d2, sibling and
  foreign documents, a random id, the RFT draft version, B's quote) give
  404, and no body holds B's name or amount, d2's name or the sibling's
  name; a `/guest/` pattern with no case here fails. (c) A documents grant
  cannot post a quote, nor a package grant fetch its file. (d) Revoked or
  expired: 401, cookie cleared. (e) Wrong origin on a guest write: 403.
- `TestGuestQuoteIsAPMDrop`: one file dropped by the PM and by A's guest on
  twin projects: same Jev calls, same decisions (user, green), same card.
- `TestGuestUploadLimits` (`.exe` 400 with no `files` row; 51 MB 413; 21st
  an hour 429; plan cap 402); `TestGuestRevokedMidUpload` (no document,
  submission or job left); `TestGuestsAreNotSeats` (Free org, 50 links:
  `GET /team` and `POST /team/invites` unchanged). Est. 380 test.

**8. Web: Share.** New `web/src/ShareDialog.tsx`; calls in
`web/src/submissionApi.ts` and `web/src/api.ts`. On `ConfirmDialog`: "Shared
with" (prefilled with the firm's directory email), "Link works until",
Create link, then the link and Copy. Below: everyone with a link to this
thing (shared with, by whom, opened n times, last opened, works until,
Revoke in one click). An issue that discloses the budget shows a warning
line first. Entry points as in Where it lives; the invited-cell button is
disabled with "Issue the RFT first" until it is issued. The dialog
registers with plan 0's `chatView.ts` so chat can open it filled in.
Playwright
`web/tests/guest-share.spec.ts`: share from an invited cell, open in a fresh
context, drop a PDF, see the PM's cell change, revoke, link refused.
Est. 140 TSX, 30 CSS, 60 test.

**9. Web: the guest page.** New `web/src/GuestPage.tsx`; `web/src/main.tsx`
renders it (its own lazy chunk) when the path is `/g`, before `App` mounts,
so no session or account call is made. On load: take `t=` from the
fragment, clear it from the address bar, post it to `/api/guest/session`,
then `GET /api/guest`; with no fragment the cookie serves. `trace-home`
sheet, Wordmark, no account menu; `TraceBand` titled with what was shared,
one line "Project · from pm@… to you@… · works until 8 Dec 2026"; a
documents table (number, title, revision, Download); issues (Issue 2,
14 Nov 2026, Download); for a firm, "Your quote": a drop zone and button
("PDF, Word or Excel, up to 50 MB") and "Received" rows. Each refusal is one
plain sentence. Est. 170 TSX, 30 CSS.

**10. Bench and docs.** `bench/budgets.json` row `guest_open`
100000/250000; the `want` map in `internal/latency/gate_test.go`;
`cmd/intake-bench/guest.go` samples link exchange and page open on a package
link with a 50-document transmittal (as `cmd/intake-bench/award.go:264`
samples `procurement_invite`). `PRODUCT.md`: replace "No sharing yet." with
one sentence. `AGENTS.md`: guest page open, guest downloads and share
create/list/revoke join the paths that do not call Jev; a guest quote is
filing plus the existing proposal fan-out. Est. 70 Go.

Totals: about 610 Go (20 of it the chat read and the revoke operation), 40
SQL, 310 TSX, 60 CSS, 800 test lines.

## From chat

Plan 0's rule table governs (`2026-11-18-chat-agency.md`).

- **Reads.** `read(shares)`: who holds a link to what, who made it and
  when, how often and when it was last opened, when it ends.
  `read(procurement)` adds each invited firm's link state. Go prints both;
  a link's token is never in a read.
- **Waits for a click.** `shares.revoke`.
- **Opened filled in.** Share, through `view`, with the thing, the person
  and the life. The PM presses Create link. Chat never makes a link.
- **Named filter** on Procurement: invited firms who have not opened their
  link.
- **Key cases.** "Who has not opened the roofing RFT"; "who can see the
  October report"; "share the RFT with Acme" (opens Share, makes no link);
  "revoke Acme's link" (prepared); "remind the firms who have not opened
  it" (opens Send to those firms, after plan 1); "share the cost plan with
  the client" (refused: only documents, an issued report or an RFT can be
  shared).
- **The guest page has no chat strip.**

## Edge cases

| Case | What happens |
|---|---|
| Forwarded link | Works for its holder, inside its one thing. The page names who it was sent to; the PM sees opens and last opened; Revoke stops it at the next request. A forwarded RFT link can drop a quote only on that firm's cell; a later quote supersedes the earlier as a revised quote does, the earlier stays with `superseded_by` (`submissions.go:789`), and the PM can detach. |
| Expired link | 401, cookie cleared: "This link has expired or been withdrawn. Ask the person who sent it for a new one." Same words for unknown, expired, revoked and archived: no oracle. |
| Revoked mid-upload | Bytes land; `OpenGuestQuote` rechecks under the row lock and refuses; the new document is deleted before filing, so no Jev call; 401. A document that existed before is never deleted. |
| Huge file | 413 at 50 MB before anything is committed (the `files.ErrTooLarge` path). |
| Wrong file type | 400 by name before a byte is read: "Send a PDF, Word or Excel file." A renamed file that will not extract is stored, not filed, as a PM drop is. |
| Firm uploading to another firm's cell | Not possible by route: the quote route takes no package or firm. Bytes identical to another firm's quote (or any project file not this firm's own quote) are answered as a received quote and nothing is stored or attached (second review, finding 4); that quote is untouched. |
| Enumeration of tokens | 256-bit tokens, SHA-256 at rest, unique index; 30 exchanges per IP per 15 min; answers carry no ids; any id a guest sends is checked against the grant (404). |
| Rate limits | Exchange 30 per IP per 15 min; reads 300 per link an hour; quotes 20 per link an hour; share creates 100 per org an hour. In memory, per process, as the account limits are. |
| PM opens a guest link in their own browser | Two cookie names; member routes read only `sitewise_session`, guest routes only `sitewise_guest` (Path=/api/guest). Neither gives the other's access. |
| Second link in one browser | Replaces the first cookie; clicking the first link again restores it. |
| Invitation removed or corrected | Removed: the link goes with it (an undo restores the firm, not the link). Corrected name with the same name key (case, spacing, "Pty Ltd"): the link follows the cell. A different name or firm: the link is withdrawn (second review, finding 1). |
| Package retired, project deleted or archived | Refused (retired), gone (cascade), paused (decision 9). |
| RFT re-issued (addendum) | The new issue shows first, earlier issues below. |
| Shared document deleted or superseded | Deleted: drops off, its URL 404. Superseded: the guest keeps the revision shared; the PM shares the new one. |
| Quote after award | 409 "This tender has been awarded." |
| Org at its documents or storage cap | 402: "The project team's upload allowance is used up. Let pm@… know." |
| Jev slow or down on a quote | As a PM drop: grey or needs review on the PM's cell; the guest sees "Received". |
| Cross-site request, framing | Origin check on guest writes, SameSite=Strict, `frame-ancestors 'none'` (`secureHeaders`, `internal/httpapi/server.go:580`). |

## Risks

1. A link is a bearer secret. Scope, expiry, revoke and the open count
   contain it; they do not remove it. If proof of the person is needed
   later, an emailed code on first open is the next step, not (b).
2. Two auth paths. A guest route that forgets `guestGrant` fails sweep (b);
   a new member route is covered by sweep (a) automatically.
3. Splitting `OpenSubmissionFor` touches every PM drop; the procurement and
   invited-firm tests are the guard.

## For sibling plans

- **Plan 4** (`2026-11-17-rfis-review-comments.md`): nothing. A guest
  writes no register row.
- **Plan 1** (`2026-11-17-correspondence-transmittals.md`) owns sending. Its
  Task 8 (large sends) makes one `documents` grant per recipient through
  `CreateGuestGrant`; 200 documents a link is the cap. **Overlap to settle:**
  whether a firm's RFT transmittal also makes the firm's `package` link.
- **Plan 5** (`2026-11-17-monthly-report.md`): a report link shows every
  issued version, newest first, never a draft (decision 4; index agreement
  6). **Plans 3, 6:** nothing.

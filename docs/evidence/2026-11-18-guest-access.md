# Guest access by link: what was built and measured

Date: 18 November 2026. Branch `wave/guests`, from local `main` at 158326f
(not merged, not deployed). Plan: `docs/plans/2026-11-17-guest-access.md`.
Lane C. Everything here was run on the development machine against a
private PostgreSQL 17 schema (`wave_guests` in `sitewise_test`); the live
server was not contacted and no live model or Jev call was made. The
chat key's new `shares` group is draft and not owner-reviewed. The plan
asks for a security review before this ships; none has been done.

## 1. What was built, by task

| Task | What | Where |
|---|---|---|
| 1. Migration | `067_guest_grants.sql`: one link to one thing (`documents`, `report`, `package`), token kept only as its SHA-256 (unique), no `respond` column, no item grants. Five composite keys; a package link cascades with its invitation and follows a corrected firm (`ON UPDATE CASCADE`). The change log accepts `via 'guest'`. The FK sweep inventories the five keys, and its fixtures now insert `package_invitations` (with a spare firm for that table's own control). | `internal/db/migrations/067_guest_grants.sql`, `internal/db/migrate_fk_sweep_test.go`, `internal/db/testdata/next_wave_foreign_keys.json` |
| 2. Store: grants | `CreateGuestGrant` in one transaction under the project lock: project live and not archived; 1 to 200 documents all filed in this project; a report with an issue; a live package with the firm invited and an issued RFP or RFT; life 1 to 180 days (default 30, a package 21); `Replaces` revokes an older link in the same step. Anything outside the project is `ErrNotFound` without saying which. `GuestGrantByToken` is one query; unknown, revoked, expired, archived project and retired package are all `ErrGuestGone`. `ListGuestGrants`, `RevokeGuestGrant` (row lock), `TouchGuestGrant`, `ShareContext`. | `internal/store/guest.go` |
| 3. Store: what a grant may fetch | `GuestPage` in one read-only transaction; `GuestDocumentFile` admits a documents link's ids, or a package link's transmittal ids from any issue of its RFP or RFT; `GuestIssueFile` admits only an issued version. | same |
| 4. Store: the quote | `OpenSubmissionFor` split into a wrapper and `openSubmissionTx` (no behaviour change). `OpenGuestQuote` rechecks the grant `FOR SHARE` under the project lock, refuses an awarded package, never re-points a document that existed before unless it is already this firm's quote here, places it with the firm preset as a PM drop on that cell, and writes one `project_changes` row (via `guest`, actor the grant id, area `procurement`, op `guest_quote`). | `internal/store/submissions.go`, `internal/store/guest.go` |
| 5. HTTP: guest routes | Five routes under `/guest`. `guestGrant` is the one check: it reads only `sitewise_guest`, hashes it, finds the live grant on every request, and takes org, project and target from the grant. Exchange `POST /guest/session`; page `GET /guest`; downloads through one `serveFile` writer; `POST /guest/quote` (type by name before a byte, 50 MB, 5 a link an hour and 10 for its life, `orgUploadAllowed`, upload, `OpenGuestQuote`, delete before filing if the link went, then `Filer.Start` or `ProposalReader.Start` exactly as `uploadSubmission`). | `internal/httpapi/guest.go`, `internal/httpapi/billing.go`, `internal/httpapi/accounts.go` |
| 6. HTTP: share routes and the board | `POST`, `GET`, `DELETE /projects/{id}/shares[/{share}]` through `deliverySession`, timed as `team`. Create answers `{share, link}` once; 100 creates per org an hour. `DELETE` is the operation `shares.revoke`, `Click`. The board's cards gain `linked`, `link_opened_on` and `tender_issued`; link changes and a guest's first open write a procurement event so the PM's cell moves live. | `internal/httpapi/shares.go`, `internal/httpapi/ops_shares.go`, `internal/store/comparison.go` |
| 7. Isolation and behaviour tests | `TestGuestReachesOnlyItsGrant` (every route pattern, via `export_test.go`), `TestGuestGrantScope`, `TestGuestGrantLife`, `TestGuestFetchRules`, `TestOpenGuestQuote`, `TestGuestRevokedMidUpload`, `TestGuestQuoteIsAPMDrop`, `TestGuestUploadLimits`, `TestGuestsAreNotSeatsAndCountTowardTheCap`; the share routes join `TestNextWaveRouteAccessSweep`. | `internal/httpapi/guest_*_test.go`, `access_sweep_test.go` |
| 8. Web: Share | `ShareDialog.tsx` on `ConfirmDialog` (plain tone): Shared with, Link works until, Create link, then the link and Copy; a warning when the newest issue prints the budget; everyone with a link (by whom, opens, last opened, works until, Revoke). One `ShareHost` per project registers it with `chatView.ts` as `share`; the register's right-click menu (Share…), Issued reports (Share…) and an invited firm's cell open it the same way chat does. The cell reads "Invited · link opened 14 Nov". | `web/src/ShareDialog.tsx`, `Register.tsx`, `Reports.tsx`, `Procurement.tsx`, `Project.tsx`, `Chat.tsx` |
| 9. Web: the guest page | `GuestPage.tsx`, its own chunk, rendered by `main.tsx` at `/g` before `App` mounts: posts the fragment token once, clears it from the address bar, then shows the band (project · from · to · works until), issues newest first, documents, and for a firm "Your quote" with a drop zone and "Received" rows. Trace sheet, Wordmark only: no nav, no account menu, no chat strip, no sign-up. | `web/src/GuestPage.tsx`, `web/src/main.tsx`, `trace-home.css` |
| 10. Bench and docs | `guest_open` 100 / 250 ms in `bench/budgets.json` and the gate test; `TestGuestOpenBudget`; the intake bench's `guestLinks` step; PRODUCT.md, AGENTS.md, the plan's status line. | `bench/budgets.json`, `internal/latency/gate_test.go`, `internal/httpapi/guest_budget_test.go`, `cmd/intake-bench/guest.go` |
| From chat | `read(shares)` (Go prints who holds a link to what, by whom, opens, last opened, end, state; never a token); `read(procurement)` adds "link opened 14 Nov" or "link not opened" on an invited firm; the named filter `link_not_opened` on Procurement; Share opened filled in through `view` (`dialog: share`), every id checked by Go first, anything but documents, an issued report or an issued RFP or RFT refused; `shares.revoke` prepared for the click. Eight draft key cases (`ga1` to `ga8`); five scored offline. The prompt gains one line. | `internal/httpapi/chat_shares.go`, `chat_view.go`, `chat_more.go`, `chat_read.go`, `knowledge/chat/system.md`, `data/eval/chat/`, `cmd/chat-eval` |

## 2. Speed

`TestGuestOpenBudget`: 20 rounds on a package link whose issued RFT
transmittal holds 50 documents; each round makes a link, exchanges it in a
fresh browser, reads the page, reads the board, lists and revokes. Local
test database, two runs.

| Budget row | Budget p50 / p90 | Run 1 p50 / p90 | Run 2 p50 / p90 |
|---|---|---|---|
| `guest_open` (exchange and page) | 100 / 250 ms | 1.8 / 3.4 ms | 2.7 / 4.5 ms |
| `team` (share create, list, revoke) | 100 / 250 ms | 1.3 / 1.8 ms | 1.6 / 2.4 ms |
| `procurement_board` (with link state) | 150 / 300 ms | 2.2 / 4.1 ms | 3.0 / 4.1 ms |

The intake bench (`cmd/intake-bench`, `guestLinks`) samples the same paths
through the real server; `TestBenchEndToEnd` passed with the step in
place, but the bench itself was not run over the private corpus. A guest
quote takes the unchanged `whole_intake` and `proposal_read` paths; they
were not re-measured. Downloads stream and are not timed (their grant
check is `guest_open`'s).

## 3. What was not measured

- No live model run; the `shares` group's model-only cases (`ga1`, `ga3`,
  `ga8`) are unscored.
- No owner review of the key or of the page's words.
- No browser check by the owner. The Playwright spec ran headless.
- No security review (the index asks for one before plan 2 ships).
- The full `tools/check.ps1` gate was not run.

## 4. Known limits

- A link is a bearer secret. Scope, expiry, revoke and the open count
  contain it; they do not prove who holds it (plan, risk 1).
- A guest cannot reach the model or Jev directly, but a quote starts the
  same background reads a PM's upload does: filing (Jev), the proposal read
  (Jev) and, when the org has switched the model on, the tender pass. A
  link therefore spends the org's reads; it is capped at 5 quotes an hour
  and 10 for its whole life, and a guest never takes the org's last tenth
  of its monthly documents or storage (review M3).
- The rate limits are in memory, per process, as the account limits are:
  a restart clears them. Failed link checks (exchange or cookie) count 30
  per address per 15 minutes, an IPv6 client keyed by its /64; a valid link
  is never blocked by them (review L2). Reads are 300 per link an hour,
  quotes 5, share creates 100 per org an hour. The limiter holds at most
  50,000 keys and evicts at most 1,000 per insert (review L1).
- A guest's upload lands held (not filed, reason `guest_hold`, no intake
  job). Only `OpenGuestQuote`, after rechecking the link under the project
  lock, releases it to filing in the transaction that places it. A crash
  or a failed clean-up leaves a held document that neither the filer nor
  `Server.Resume` will file; it shows as not filed in the register until
  someone deletes it.
- A corrected firm name keeps the link (the firm's id does not change). A
  swap to a different firm withdraws the old firm's links first (review
  M2); the foreign key keeps `ON UPDATE CASCADE` because a swap moves the
  invitation's firm id and the withdrawn rows still reference it.
- "Shared with" is a label the PM types; SiteWise sends no mail (plan 1
  owns sending). "Remind the firms" answers with the fixed words.
- The 402 message tells the guest to let "the person who sent this link"
  know, not the sharer's address.

## 5. Threats

After the security review of 18 November 2026 (no Critical or High; three
Medium and five Low, all fixed, each test shown to fail without its fix).

| Who | Can | Cannot | Held by |
|---|---|---|---|
| Holder of a leaked link | See and download the one shared thing until it ends or is withdrawn; a firm's link can drop up to 10 quotes (5 an hour) on that firm's own cell only | Reach any other document (only the transmittal block's ids, review note), an issue's draft, another firm's quote or price, another project or org, any member route, chat; push the org's PMs to the plan cap | `TestGuestReachesOnlyItsGrant` (a), (b), (c); `TestOpenGuestQuote`; `TestGuestDownloadsOnlyTheTransmittal`; `TestGuestQuoteLifetime`; `TestGuestUploadLimits`; `TestGuestsAreNotSeatsAndCountTowardTheCap` |
| A firm holding two links in one browser | Use whichever link its page was opened on | Upload to or download from the other link from a stale tab (409 "This page is for another link. Reload.") | `TestGuestPageIsTiedToItsLink` (M1) |
| A firm whose cell was swapped to another firm | Nothing: its link is withdrawn in the swap | Upload as, or see the quotes of, the firm now in the cell | `TestGuestGrantLife` (M2) |
| Someone guessing tokens | 30 misses per address (IPv6 per /64) per 15 minutes; 256-bit tokens | Learn whether a token was ever valid: wrong, expired, revoked, archived and retired give one 401 body; lock out a valid guest behind the same address | `TestGuestReachesOnlyItsGrant` (d), `TestGuestDeadLinksOneAnswer`, `TestGuestMissesBlockOnlyMisses` (L2), `TestLimitAddressKeysIPv6By64` |
| Someone flooding the limiter with addresses | Fill it to its cap | Grow it without bound or slow sign-in | `TestLimiterHasAHardCap` (L1) |
| Holder of a revoked or expired link | Nothing: the next request is 401 and the cookie is cleared; an upload in flight places nothing and is never filed, even after a crash | Re-exchange the token; keep a cached page (no-store) | `TestGuestRevokedMidUpload`, `TestGuestUploadIsHeldUntilTheRecheck`, `TestResumeLeavesAHeldGuestUpload` (L4), `TestGuestGrantLife` |
| Holder of a link made by a removed team member | Nothing: the member's links are withdrawn with the membership | See the removed member's name on a live page | `TestRemovedMembersLinksStop` (L3) |
| A firm whose tender was awarded on a combined row | Nothing more | Quote after the award stored on the group | `TestGuestQuoteRefusedAfterAGroupAward` (L5) |
| Signed-in user of another org | Nothing on this project: share list, create and revoke give 404 | Make a link to another org's records; make more than 100 links an hour | `TestGuestGrantScope`, `TestNextWaveRouteAccessSweep`, `TestShareCreatesPerOrg` |
| A PM's own browser with a guest cookie | Two cookie names; member routes read only `sitewise_session`, guest routes only `sitewise_guest` (Path=/api/guest) | Gain either side's access from the other | (a) of the sweep sends the guest cookie to every member route |
| A cross-site page or a network observer | Nothing: guest writes check Origin, the cookie is SameSite=Strict and, in production, Secure; pages cannot be framed | Post a quote or exchange a token from another origin; read the cookie over plain HTTP in production | `TestGuestReachesOnlyItsGrant` (e) and its header checks, `TestGuestCookieSecureInProduction` |
| One link reading in a loop | 300 reads an hour | More | `TestGuestReadsPerLink` |

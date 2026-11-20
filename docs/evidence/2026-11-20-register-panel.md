# The register in the right-hand panel: what was built and measured

Date: 20 November 2026. Branch `wave/register-panel` (not merged, not
pushed, not deployed). Owner-approved wireframe, described in the brief of
that date. Run on the development machine against private PostgreSQL
schemas (`wave_regpanel`, `wave_regpanel_e2e`) with the e2e server's mail
sink and stand-in model; no live Jev, model, Mailgun or SMTP call was made.

## What changed, in plain words

- **The right-hand panel has a switch**, Documents | Register, in a band at
  its top. Documents is the repository as it was. The choice is kept in the
  viewer's browser (`localStorage`, wrapped in try/catch). Register is off
  the left nav; the nav entry carried no count or mark, so nothing moved
  with it. An old link with `?view=register` opens the Profile with the
  Register beside it.
- **The list is narrow**: No. | Item | With | Due. The status is a chip in
  the Item cell (it drops under the title when the title would get less
  than 8rem); System is a filter and, with "Set system", is edited in the
  Item cell. The filters row (kind, system, Show closed), the grey "Raise an
  RFI…" row, the row menu (three dots and right-click), keyboard selection,
  Ctrl+A, Escape, multi-select, double-click edits of Item, With and Due,
  F2 (title), gate groups, consent read lines and Read conditions stay.
  Project report is in the band.
- **A row opens as one sheet in the middle panel** (click, Enter, the row
  menu's Open, or chat's `view open`); Back returns to the middle view that
  was there. An RFI's sheet: a header (`RFI-003 · Title`, a chip that says
  who holds it, a Due chip, three dots, Send), the Profile card's
  label|value grid (To, Cc, Subject, Raised with the response time; Their
  ref only when the RFI has a ref or was raised by someone else), the
  question, the Transmittal band (unchanged reuse of `RfpTransmittal`) and
  a History band. A notice shows With, Due, its notice details and history;
  a consent condition shows today's `ConditionDetail` unchanged in function,
  plus its history.
- **History** is built in the browser from records Go returns, oldest
  first: raised, each status change (`provenance.status_history`), what went
  out (to whom, documents, TR number, "Open what was sent"), each reply in
  the thread (sender, first lines as plain text, attachments, "Open the
  reply", "Use as the answer"), the accepted answer documents, a typed
  answer (editable), and possible answers (amber, Open / Accept / Dismiss).
  A ring in the accent colour is what went out, an amber dot what came in,
  a green dot the answer. With mail off it shows raised, status changes and
  the accepted documents only, and reads complete.
- **Send** on the sheet posts to the same `POST /projects/{id}/transmittals`
  with To, Cc, the subject, the question as the note, the transmittal's
  documents and `rfi`. Nothing new on the send path. A field the PM changed
  is saved first. The button's title says what goes ("Send to 1 address
  with 2 documents"); the line under it shows that, or the server's refusal
  in its own words, or why Send is off.
- **Saved fields**: `to`, `cc` (lists) and `subject` are RFI details, saved
  through `delivery.edit` with Undo. No migration (details is JSON).
  `internal/delivery` checks them with the send route's own functions
  (`correspondence.ParseRecipients`, `CleanHeader`): one address an entry,
  no line break or control character, at most 20 in all, subject one line
  of at most 300.

### What calls what

- `GET /projects/{id}/delivery` now also returns each sent RFI's `thread`
  (one extra query for the whole register, org and project on every join,
  at most 50 messages a thread, reply text cut to three lines in Go) and
  `send_off` when the server has no outbox. No Jev, no model.
- `delivery.edit` and `delivery.add` resolve To and Cc through Send's
  existing resolver (`resolveAddresses`, split out of `sendAddresses`), so
  a firm's name becomes its recorded address.
- Chat: an address it puts in To or Cc must be in the PM's own message or
  be the recorded address of a project firm the PM named (`chosenAddress`,
  shared with Send's guard); otherwise the batch is refused. `view` dialog
  `send` with `rfi` opens that RFI's sheet filled in (`rfi-sheet` in the
  screen registry); `view show register` switches the panel; `select`,
  `only`, `clear` act on the list; `open` opens the sheet. The screen line
  names an open register row ("Open: register row …").
- No new write route.

### What could fail

- An RFI whose `sent_message_id` points at a message in another project
  gets no thread (the join carries the project); tested.
- A reply with no possible answer (the register job has not run, or it was
  dismissed): "Use as the answer" makes the same edit by hand
  (`answer_documents`, Answered, and the reply's own Sydney date as the date
  answered when none is set).
- The register's list is mounted once opened and hidden after; its chat
  table registers only while it is shown, and chat shows it first.

## Speed

`TestRegisterBudget` (30 samples, the test's own count; budget
`delivery_read` p50 ≤ 100 ms, p90 ≤ 250 ms). The test now also seeds 50
sent RFIs with two replies each (150 thread messages).

| | delivery_read p50 | p90 |
|---|---|---|
| Before (base code, the test as it was) | 26.3 ms | 29.7 ms |
| After, 150 thread messages, statistics analysed (two runs) | 28.8 / 29.1 ms | 30.8 / 31.2 ms |
| After, same data, tables never analysed | 38.6 ms | 39.4 ms |

`delivery_system_read` after: p50 25.0 ms, p90 26.3 to 27.6 ms (budget
150/300). `delivery_write` after: p50 10.2 ms, p90 26.2 ms. The test now
runs `ANALYZE` on the four tables after seeding, as a live database keeps
statistics; without it the planner guessed one row and the read took about
10 ms more. A first version of the thread query (correlated subqueries per
message) added about 46 ms and was rewritten: it now starts from the sent
message ids Go already holds, counts parts by a lateral lookup on
mail_parts' key, and matches a reply's possible answer in Go. All within
budget.

## Tests

- `go build ./... && go vet ./...`: clean.
- `go test -count=1 -p 1 ./...` (schema `wave_regpanel`): 45 packages ok;
  `internal/store` fails only `TestRestoreFactsDetectCostContentChanges`
  (expected under a non-public schema). `cmd/chat-eval` failed on the new
  key cases rg9 and rg10 until the offline stand-in got their scripts; it
  then passed (`ok sitewise/cmd/chat-eval 24.8s`). New tests:
  `TestRFIAddressing` (internal/delivery), `TestRFIAddressingSavedWithUndo`,
  `TestRegisterReadCarriesRFIThreads`, `TestChatRFIAddressesAreThePMs`,
  `TestChatOpensTheRFISheetAndSeesIt` (internal/httpapi).
- `python tools/check_knowledge.py`: 0 errors, 1 warning (the existing
  `flood_exposure` warning in `knowledge/works/package_defaults.yaml`).
- `npm --prefix web run test`: 274 passed, 3 failed (the known
  `program/engine.test.ts` random-03/14/44); the new
  `registerHistory.test.ts` passes. `npx tsc -b --noEmit`: clean, no `.js`
  emitted into `web/src`.
- Playwright (schema `wave_regpanel_e2e`), the specs this change touches:
  20 passed, 0 failed (delivery 3, conditions 1, project-report 1,
  correspondence 2, chat-view 7, chat 4, guest-share 1, register-edit 1).
- The full Playwright suite on this branch: 85 passed, 69 failed, 10
  skipped before it was stopped near the end (the intake spec's worker was
  killed by the stop). The failures are in program, program-edit,
  program-review, rfp-*, tender-*, profile, procurement-award,
  navigation-refresh, reports, invoices and others. The same subset run on
  the base commit's web code (f949971, before this change) failed 79 of 79
  tests in navigation-refresh, procurement-award, hydraulic-rfp,
  profile-run, profile, program*, reports, proposal-reading and
  rfp-alignment, so those failures were there before this change and are
  not caused by it. The other failing files (rfp-allowances, rfp-editor,
  rfp-export, rfp-group, rfp-scope-lines, rfp-terms, specifications,
  tender-*, variations, visual-identity, work-layout, works, invoices) were
  not re-run on the base code.

## Screenshots (1650 × 920)

- `docs/evidence/img/2026-11-20-register-panel-register.png`: RFI-001 open,
  Register on the right, after Send (history: raised, sent).
- `docs/evidence/img/2026-11-20-register-panel-documents.png`: the same
  RFI with Documents on the right, scrolled to the history (raised, sent,
  the reply quoted with "Use as the answer").
- `docs/evidence/img/2026-11-20-register-panel-answered.png`: after "Use as
  the answer" (green dot, Answered).

Checked by eye: no overlap or clipped text in the sheet. The Due chip
wraps to a second header line at this width; long titles and With are cut
with an ellipsis in the narrow list (the full title is the cell's tooltip).

## Owner decisions open

1. With messages on the project, the Documents side shows a second
   Documents | Messages toggle under the panel's switch, so two tabs read
   "Documents". Fold Messages into the panel switch, or leave it?
2. "Raised by" shows `raised_by` when set; who in the team raised it is not
   shown (the record keeps a user id; naming members needs one more read).
3. Answered early or late is not shown: Go does not compute it for an
   answered RFI, and the browser does not compute dates.
4. Chat's Send for an RFI uses its To, Cc and Subject only; a drafted note
   or a document list from chat is not used, because the question is the
   note and the transmittal is the documents.

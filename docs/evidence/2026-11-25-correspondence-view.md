# Correspondence view, raised claims, Report and breadcrumbs: evidence

Date: 25 November 2026. Plan: `docs/plans/2026-11-23-correspondence-between-parties.md`,
section 19. Lanes A (register rules: direction, With) and B (layout).

## What changed

- **Direction.** An EOT claim, a notice of delay and a letter are either
  received or raised by the host (`details.direction` = `out`). A raised
  item is with the party the row names. Once their result or reply is
  recorded, it is with the host. Its due date is the other party's period
  from the day it was lodged, worked out as before. There is no migration:
  the field lives in `details`, and a blank direction reads as received.
  Go: `delivery.Raised`, `checkDirection`, `delivery.With`; chat takes
  `direction` on `delivery.add` and `delivery.edit`.
- **Correspondence.** On the left nav in Specifications' place
  (Specifications stays reachable by `?view=specifications`). It has two
  tabs, Register and Messages. The right-hand panel is the documents only.
  - The register is one list: Type, No., Subject, Raised by, With, Issued,
    Due, Status.
  - A heading click sorts, and a second click reverses. A dragged heading
    moves its column, and the order is kept in `localStorage`. The three-dot
    menu has Reset columns.
  - A click on a row drills in to its sheet. A click on With, Due or Status
    only selects; a double-click there edits; F2 edits the subject.
  - The sheet is laid out in RFP bands: Particulars, the message (Question,
    Claim, Notice or Letter), the answer or result, and Responses and
    history.
- **Report.** The project report is its own nav view under Correspondence,
  with RFP section bands. Wording is edited by clicking it, as on an RFP,
  and calculation notes are hidden, as on an RFP.
- **Breadcrumbs.** `BackButton` is now a trail. Every step above the open
  view is a link back, and the last step names what is open. The trail is
  used for the register sheet, documents, the price table, the comparison
  report (Procurement › package › Comparison report) and RFPs.
- **Fix.** A 409 on a register write was wiped by the successful re-read
  that followed it, so the PM never saw it. Now a good read clears only a
  failed read's message.

Nothing here calls Jev or the model.

## Timing (store test, 1,000 rows, 20 samples)

| Path | p50 | p90 | Budget |
| --- | --- | --- | --- |
| delivery_read | 30.6 ms | 32.7 ms | 100 / 250 ms |
| delivery_write | 2.0 ms | 2.5 ms | 100 / 250 ms |

## Tests

| Run | Result |
| --- | --- |
| `go test ./internal/delivery ./internal/store -run Register` | Pass, including the new `TestRegisterRaisedClaimDirection` |
| `go test -p 1 ./internal/httpapi ./internal/reports ./cmd/chat-eval` | Pass. `TestMailBudgets` failed once, on one `mail_read` sample of 390 ms (p50 1.6 ms), then passed twice in a row; this change does not touch the mail read |
| Browser: delivery (5, one new for a raised claim), conditions, correspondence, project report, specifications, chat-view (7) | Pass, except three that already failed on `main` at `9472e1a`: correspondence's "mail" address check and both procurement-award tests (the second now passes its breadcrumb and stops at its older failure) |

## Screens

Screenshots were taken on the dev server (Hale 2) with Playwright, since
Chrome was not connected. A raised EOT claim was added, checked and undone.
The correspondence spec writes `docs/evidence/img/2026-11-25-correspondence-*.png`.

## Not done

- The builder's own time bar for lodging a claim.
- Sending a raised item through mail.
- Outside parties signing in (slices 3 to 5).

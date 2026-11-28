# Evidence: payment schedules in the owner's format

Plan: `docs/plans/2026-11-26-payment-schedules.md` (all eight slices built
26 to 28 November 2026, not committed). Every recommended owner decision in
section 14 was taken as given while the owner was away; none is reviewed.
Nothing was marked verified or reviewed.

## What was measured

`TestVariationsBudget` (`internal/httpapi/claims_test.go`), p50 / p90 in
milliseconds against the recorded budget of 100 / 250, local Postgres 17 on
the development machine, 27 November 2026:

| Path | Bench row | p50 | p90 |
|---|---|---|---|
| One sheet, `GET /invoices?sheet=` | `schedule_read` | 34 | 35 |
| The tab's read, `GET /invoices` | `variations_read` | 31 | 36 |
| Assess by percent or dollars; retention; Accept claimed | `claim_assess` | 39 | 41 |
| Breakdown add, edit, rebuild | `breakdown_write` | 32 | 33 |
| Contract terms (the award's PATCH) | `variation_write` | 28 | 33 |
| Issue: snapshot, PDF in the new layout, file row | `payment_schedule_issue` | 27 | 34 |

`transmittal_send` and `mail_send` were not re-measured for a claim's Send;
it adds one indexed read and one decision row to the existing send.

## What was not measured

- The bench fixture was not grown to 12 contracts of 80 breakdown lines and
  24 earlier claims (section 10). The rows above are on the test fixture of
  one contract with two lines. An 80-line schedule is laid out by
  `TestScheduleOfEightyLinesRunsOn` (three pages, no row cut) but not timed.
- No real progress claim was read. The claim key
  (`data/eval/claim/private/`) is still empty, so `claim-1` picks stay
  blank, as before.
- The chat cases cl9 to cl13 are draft and scored only against the offline
  stand-in model (`TestScreenCasesScoredOffline`), not a live model.
- Send ran only against the test sink (`internal/httpapi` and the browser
  suite). Mailgun has still not been used (mail plan, Task 0).
- No Act row other than NSW is verified, so on most projects the printed
  line is "This is a payment schedule."

## Tests

- Go: `internal/invoices/schedule_test.go` (the golden's figures with generic
  labels), `internal/reports/payment_schedule_sheet_test.go`,
  `internal/export/schedule_test.go`, `internal/store/breakdown_test.go`,
  `internal/store/claim_invoice_test.go`, `internal/httpapi/schedule_test.go`,
  `internal/httpapi/schedule_send_test.go`.
- Browser: six tests in `web/tests/progress-claims.spec.ts` (drop to issue,
  the breakdown, terms and retention, Send, placing an invoice), plus
  `variations.spec.ts`, `invoices.spec.ts` and `costs.spec.ts`.

## Seen on the page

The issued PDF of a sample sheet (two works lines, one cut with its
reason, a provisional sum marked final, a variation, retention at 5% of the
works, two payment rows) prints in two A4 pages in the house table style:
details grid, one table per section headed by its own name, figures right
aligned, totals bold, the approval row with room to sign.

## Known failure, not from this work

`web/tests/correspondence.spec.ts` "the project's address, a held message
accepted…" fails on committed code (`d9e371b`): the project's mail address
is named after the project ("mail_house") and the test expects a random
one. It was not changed here.

The golden documents behind this plan carry a real client's contract
figures; keep `docs/paysch-report-golden/`, the plan and the wireframe out
of any public push.

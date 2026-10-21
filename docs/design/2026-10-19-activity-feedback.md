# Activity feedback

Lane B. A shared, subtle indicator makes pending work legible across the
workbench, especially batches of documents. It improves time-to-decision by
distinguishing a file waiting its turn from work already underway.

## Contract

- A 14px cyan open ring rotates once per second beside cyan status text.
  Active document rows have a faint cyan tint. A muted static clock means
  queued or waiting. Colour denotes processing here, never confidence or acceptance.
- Uploads start queued and become active only when the existing upload pump
  starts the request. Real byte progress remains visible. At 100%, the copy
  says "Finishing upload" until the server replies.
- OCR stages, full-text job states, profile active counts and specification
  import leases distinguish waiting from reading. Image review derives `reading` from a live worker lease in the existing
  status query. It says "Reading document images" while leased, and "Waiting
  for image reading" while queued or after lease expiry. Both register and
  review-panel polling continue during reading so a missed completion event
  cannot leave the ring running. Pending initial filing likewise has no
  worker-start signal and says "Waiting for filing". Do not infer progress
  from time elapsed.
- Invoice and proposal rows use their existing `reading` status. No new
  stored worker state, endpoint, polling frequency, or Jev call is introduced.
- The register summary counts visible documents once, preferring active text
  preparation when another job for the same document is waiting. It describes
  current work, not a fabricated batch-completion percentage.
- Loading and saving indicators delay the icon by 300ms with CSS, reserving
  its space immediately. Completion and failure remove activity immediately;
  they never wait for an animation. Accepted values and review decisions do
  not change.
- Reduced motion removes rotation. Icons are hidden from assistive technology;
  words and existing live regions convey status. The register announces one
  aggregate status rather than a live announcement per row. Printing hides
  the icons.

## Coverage and boundaries

Shared `Activity` / `Pending` components cover the register and expanded filing,
OCR, image review, profile and sources, invoice uploads and ledger, proposal
uploads and price tables, specification imports, loading messages, existing
pending buttons and save feedback. Cost, invoice, price and specification
writes expose a local saving indicator. Confirmation and award actions use
the same treatment. Browser-managed download links retain native browser
feedback: the app does not receive a reliable download-completion signal.

No processing, money, classification, acceptance, persistence or permission
changes. No new dependency. Existing speed budgets remain binding: filing
p50/p90 1,000/2,000ms, profile edits 50/150ms, other named paths as recorded
in `bench/budgets.json`. This change adds no network round trip or artificial
completion delay.

## Verification

`web/tests/activity.spec.ts` exercises upload-slot queuing, failed transfers,
queued-to-reading and reading-to-complete transitions, honest image-review
copy, reduced motion, and desktop/mobile captures. Existing image-review and
invoice tests cover explicit acceptance and the complete invoice workflow.
Production build and whitespace checks pass. The selected 21 browser scenarios
pass: 20 in the final grouped run, followed by the profile-progress scenario
after updating its expected wording to include the new counts. Desktop/mobile
register captures were inspected; the filename separator was clarified and
the final captures confirmed it.

Local replay benchmark (184 files, two rounds, four concurrent uploads):

| Path | p50 / p90 ms | Budget p50 / p90 ms |
|---|---|---|
| Filing | 351 / 791 | 1,000 / 2,000 |
| Invoice intake | 261 / 353 | 1,000 / 2,000 |
| Proposal reading | 1,792 / 1,966 | 3,000 / 6,000 |
| Profile edit | 34 / 130 | 50 / 150 |
| Cost read / write | 2 / 5; 4 / 6 | 100 / 250 each |
| Specification read / write | 6 / 9; 2 / 3 | 100 / 250 each |

The full gate **did not pass**: `spec_home_profile_edit` measured 61 / 77ms,
exceeding its 50ms p50 budget. Text extraction p90 349ms and deterministic
rules p90 3.2ms also exceeded the component targets reported for VPS
validation. No backend code was changed for activity feedback. These are local
replay measurements, not production or live-provider evidence. Full results
are in the ignored `tmp/activity-bench-results.json` and samples in
`tmp/activity-bench-samples.json`.

Follow-up: the owner screenshot exposed that image reading had no active UI
state. A failing store regression reproduced a live lease still appearing as
queued. Both review and register reads now expose live leases as reading;
queued, expired and failed jobs are covered. Four focused browser scenarios
pass, including queued → reading → ready with no SSE, colour distinction,
row tint, reduced motion and desktop/mobile captures. Focused store, intake
and authenticated endpoint tests pass. No new dependency.

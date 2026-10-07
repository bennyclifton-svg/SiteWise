# WP-45a proposal review controls

Lane A. Outcome: inspect a saved proposal and make an explicit planning
decision, reducing the time between seeing a requirement and allocating it.
This extends the existing workbench; it does not enable the evaluator on
profile edits or claim M1 completion. No new service, dependency or Jev call.
Existing proposal API budgets remain p50/p90 100/250 ms. Full evaluation still
has to pass the separate profile edit/rebuild budgets before runtime wiring.

## Implemented

`web/src/Proposals.tsx` and `proposalApi.ts` consume the saved ranked proposal
API. The default list preserves the server's critical-item policy and exposes
Show all. A bounded list keeps the reader reachable on mobile. Reasons include
knowledge and interface identity, triggering work with location labels and
source excerpts, determinant values, rule-reference qualification and recorded
signals. Draft knowledge, proposed triggering work and changed inputs remain
visible. An empty saved projection is explicitly not a completeness claim.

Acceptance is a deliberate form submission. Investigation/work proposals create
planning work; discipline proposals create planned services packages;
obligations require a chosen package, with optional explicit work, role and
stage; approvals and hold points create uncompleted delivery requirements.
Retained-work assignments require a works package and maintain-operation or
protect role. The server remains authoritative for all target validation.
Incomplete work targets cannot be accepted in the UI.

Dismissal preserves an optional rationale. Undo shows the agreed consequence:
retire an untouched/unreferenced created record, refuse edited or used records.
Decision versions and input fingerprints go back unchanged to the API. A
changed saved proposal blocks a pending review; choices remain visible until
the user cancels and starts again against the current reason. Switching project
views preserves an unfinished review. Failed loads remain distinct from empty
results and disable decisions until recovery.

## Validation

- Production TypeScript/Vite build passes (bundle 346.83 kB, gzip 101.05 kB).
- Seven Playwright proposal scenarios pass against the real Go API and
  PostgreSQL: investigation, discipline, obligation, approval and hold-point
  acceptance/undo; stable investigation identity on reacceptance; edited-item
  undo refusal; dismissal; retained choices across navigation; stale-input
  submission block; failed-load recovery. The stale response is deliberately
  intercepted to simulate changed inputs; mutation scenarios use the real API.
- The first investigation run exposed a test loading race; waiting for the
  rendered list before checking Show all fixes the test. All seven pass after
  that correction. Four adjacent Reports/Works/Delivery browser tests also pass.
- The isolated `web/tests/e2eserver` compiles. Its new diagnostic fixture route
  uses a freshly loaded catalogue per request, changes synthetic labels/kinds
  only there, and explicitly invokes a rebuild. It is not in the production
  server and the executable refuses databases other than `sitewise_test`.
- Detector returned no findings for the new proposal files. Desktop/mobile
  captures and stale/blocked-undo states are under ignored `.tools/` as
  `wp45-proposals-*.png`. The skill's independent finish review returned
  **ship**, with no material fixes. No new visual identity or concept exercise.
- `git diff --check` passes. No claim of new live Jev, VPS or release evidence.

The unchanged endpoint implementation has local 40-cycle benchmark evidence in
`bench/results/2026-10-05-profile-site-reuse-diagnostic.json`: proposal read
5.028/6.369 ms, work accept 8.277/9.161, dismiss 7.810/9.007, work undo
9.627/12.025, delivery accept 3.675/4.333 and delivery undo 7.184/10.367.
All are within their 100/250 ms API budgets. These are server timings from the
prior retained implementation, not a new browser-render benchmark. The new
view fetches saved proposals, packages, works and profile in parallel; no
evaluation happens in those reads. The overall diagnostic still fails the
independent Spec Home profile-edit median gate. Automatic proposal rebuilding,
owner-reviewed critical recall/usefulness and full M1 acceptance remain open.

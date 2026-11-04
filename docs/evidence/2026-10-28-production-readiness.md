# Production-readiness cleanup evidence

28 October 2026. Decision: **release remains blocked** until the gates below
are satisfied. This cleanup does not constitute a production deployment.
The item-by-item critique and implementation scope are in
`docs/plans/2026-10-28-production-readiness.md`. The GPT/LLM assistant proposal
was explicitly excluded by the owner. No dependency or runtime AI was added.

## Review method and delivered behavior

Checked the supplied review against PRODUCT, the foundation design, current
code and the relevant surface notes. Two independent, read-only interface
reviews and the Impeccable detector supported the critique; detector output is
advisory, not a release gate. It reported 78 findings, mostly existing token
and typography conventions. Desktop, narrow-pane and 390px browser checks
provided the implementation evidence. This is not a complete accessibility or
security certification.

- Removed the inactive composer and Chats destination; kept functioning
  Profile feedback. Project Plan remains disabled and visibly planned.
- Made project navigation collapsible on small screens. The project switcher
  is a searchable, bounded disclosure with All projects and Escape support.
- Added local project-name search, a separate actions disclosure retaining
  delete confirmation, and recoverable project/catalog loading errors.
- Added real sign-out: same-origin DELETE, organisation/user-scoped session
  revocation, expired HttpOnly cookie, cancellation of that session's SSE
  streams and in-memory UI reset. A failed request leaves a retry action.
  Replayed revoked cookies fail; other sessions remain valid. SSE also ends
  at session expiry. Pending project/catalog responses cannot restore data
  after sign-out.
- An untitled file shows its filename and Needs title review; this never
  writes an accepted title. Existing OCR progress remains visible.
- Profile responds to its pane width, selected choices have a non-colour
  mark, sources disclose their count, and streamed updates preserve a typed
  correction. New appointment flags without an explicit layout home remain
  visible in Project rather than silently disappearing.
- Download has a visible label; theme controls and focused skip link are
  readable. Developer diagnostics remain hidden unless a dev endpoint was
  confirmed. DESIGN now describes the actual font and shared token system.
- Fixed desktop excess scroll caused by an offscreen live region; fixed
  frozen cost identities intercepting edits in narrow panes. Financial
  calculations and unknown-value semantics are unchanged.
- Worker lanes now receive fresh initialisation state instead of copying a
  mutex-bearing `sync.Once`. Fixed CRLF-sensitive fixture setup, isolated
  the Program import test's empty-stage scenario from the reviewed library, supplied
  required synthetic RFP-ledger data and made optional Python rendering
  imports lazy for pure helper tests.

The checkout already contained extensive uncommitted feature work. It was
preserved. The total Git diff therefore includes work outside this cleanup.

## Verification

Full command output is retained under `tmp/readiness-*.log` in this checkout.
Those diagnostic files and screenshots are local, ignored artifacts; this
document records the durable conclusions.

| Check | Result |
| --- | --- |
| Vite/TypeScript production build | Pass; existing large-chunk advisory remains (about 694 kB before compression). |
| Frontend unit tests | 234 tests in 10 files pass. |
| Go vet | Pass after worker-copy correction. |
| Python helper tests | 46 pass after portable fixture/lazy-import corrections. |
| Strict knowledge validation | 0 errors; 2 existing mapping warnings. |
| Logout security/SSE and upload-reconnect checks | Pass, including final logout rerun; timing recorded below. |
| Backend corrections | Profile placement, Program import, all jobs tests and all knowledge tests pass. `TestProgramRoutes` still fails on its obsolete empty-library expectation; Meridian accuracy still fails. |
| Focused browser smoke | Initial corrected run: 8 pass. Final expanded cleanup regression: 17 pass, including all Profile tests, OCR states/retry, cost selection/edit, mobile cost removal, load recovery, sign-out, light/dark layout and draft protection. |
| Full browser suite | 128 tests: 76 pass, 41 fail, 11 previously skipped. Seven of those failures pass on corrected reruns; 34 remain unresolved. This is not a green full-suite result. |
| Program touch regressions | Long-press drag/swipe and one-finger scrolling pass after opening the mobile disclosure and supplying enough synthetic rows to require scrolling. No screenshot baseline was silently regenerated. |
| Installed OCR gate | Pass, 20 filings and 20 detail recoveries. Filing p50/p90 3355/3531ms; recovery 3389/3623ms; budget 10000/20000ms. Includes 2000ms polling allowance and replayed provider latency. |
| Installed visual identity gate | Pass, 20 samples: foreground filing 370/378ms against 1000/2000ms; visual reading 932/976ms against 10000/20000ms. Replayed provider latency, queue excluded. |
| Invoice replay | Pass: 90 cases, zero wrong green values. |
| Intake replay | Fail: 129 missing recorded responses for the current question version. |
| Profile replay | Fail: stale Bankstown recording; separate Hale replay also reports stale recording. |
| Proposal replay | Fail: missing `data/eval/proposal/private/answer-key.json`. |
| Complete latency benchmark | Fail: `spec_home_profile_edit` p50 54.779ms exceeds 50ms (p90 62.311ms). Filing and normal Profile edit budgets pass. See details below. |

Measured local timings (milliseconds, p50/p90): sign-out 0.585/0.647 against
100/250 (20 samples); cost disclosure 28.9/32.9 against 100/250; invoice
discipline write 8.3/9.5 against 50/100; document field correction 14.6/19.7
against 50/150; project rename 2.1/2.7 against 50/150. Program first chart
paint was 77.3ms against 150ms; drag p95 3.2ms against 16ms; fit-to-screen
maximum 13.5ms against 16ms. These are local checks with synthetic/replayed
inputs, not live VPS evidence. The focused sign-out timing ran while the
broader browser suite was active.

The complete replay benchmark exercised 184 files twice, concurrency 4,
2 background callers and 10% deliberately degraded requests. Whole filing:
p50 410.159ms / p90 1030.791ms, budget 1000/2000ms. Profile edits:
40.210/139.660ms, budget 50/150ms. The gate exited 1 for the
specification-home edit above. It also reported component overruns for text
extraction (p90 567.981ms vs 250ms), deterministic rules (1.081/4.001ms vs
1/1ms), and commit/SSE enqueue (p50 5.143ms vs 5ms); these component gates
are judged on the target VPS. Files: `tmp/readiness-bench.json`,
`tmp/readiness-bench-samples.json`, `tmp/readiness-bench.log`. Its early stages
overlapped browser/OCR verification; no target-host performance claim is made.
The benchmark's replayed latency is independent of the stale accuracy
recordings and does not repair or validate their answers.

## Release blockers and limits

1. Refresh the private intake/Profile recordings against the current Jev
   question versions, preserving independently verified answer keys. The
   Meridian Go regression also reports an incorrect automatic acoustic
   separation value despite meeting its aggregate score floor (39/46).
   Do not relax the threshold or edit the answer key to conceal this.
2. Restore the private proposal answer key/corpus required by its evaluator.
3. Resolve the remaining broad browser regressions and run the full suite
   green. Some Program editing tests assume generated rows that the current
   reviewed library does not provide; procurement tests also have expectations
   inconsistent with the current shell. Outstanding groups: Program editing,
   reading/review, stage plan, print and screenshot baselines; procurement
   award and report navigation; proposal reading/quoted fee scope; saved
   reports; specification status selectors. These are not declared harmless
   merely because they predate this cleanup. The complete failing test names
   and errors are retained in `tmp/readiness-browser-full.log`.
4. Resolve the specification-home Profile edit budget failure, repeat the
   complete latency gate in isolation, then run the live release gate on the
   intended VPS. Local replay timings do not measure Jev latency.
5. Complete the operational gates in `docs/operations.md`, including live
   production configuration and off-site restore rehearsal. No live service,
   mail delivery, production data or backup restore was modified or certified.

The missing private evaluation inputs cannot be replaced with invented
answers. Existing failing gates remain visible; no assertion or budget was
relaxed to obtain a green report.

## Reproduce

The canonical complete gate is `tools/check.ps1`; it fails closed at the first
failure. This review ran its components separately to expose later failures
instead of stopping at the first Go regression. The final UI rerun was:

```powershell
npm.cmd --prefix web run build
npm.cmd --prefix web run test:e2e -- production-readiness.spec.ts cost-selection.spec.ts ocr-retry.spec.ts ocr.spec.ts procurement-cost-remove.spec.ts profile.spec.ts
npm.cmd --prefix web run test:e2e -- program.spec.ts program-edit.spec.ts -g 'touch'
npm.cmd --prefix web run test:e2e -- program.spec.ts -g 'one-finger'
```

The first touch rerun exposed a synthetic fixture too short to scroll; the
last command passes after using enough rows to exercise the original scroll
assertion. The first OCR gate passed its timing assertions but failed to write
a relative output path from the Go package directory; the final run used an
absolute path and passed. Neither failed attempt is counted as a passing run.

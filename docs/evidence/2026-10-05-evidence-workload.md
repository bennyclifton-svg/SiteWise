# Evidence-call workload after K4, and the restored profile replay gate

Date: 5 October 2026. Package WP-00 (`docs/plans/2026-10-04-next-wave-agent-work-packages.md`). Decision D-17 resolved under owner authorisation A9 (`docs/plans/2026-10-04-next-wave-implementation-plan.md` §0.1).

## What was broken

At `f0c170a` both profile replays failed before scoring:

- `go run ./cmd/profile-eval` → `bankstown-page9: evidence recording is stale`
- The Hale replay → `hale-description: evidence recording is stale`

At `02094db`, with the same recordings, the source suite scored 15/15.

## Cause (measured)

A throwaway diagnostic built each source case's evidence call at both commits, exactly as the replay builds it, and compared the question IDs. It lives in git-ignored `tmp/calldiff`.

- **Label calls:** unchanged.
- **Evidence calls:** gained 5 to 23 questions per case. All 50 added IDs are failure-mode detectors (`fm:fm.*`), from the K4 Pass B merges. No question was removed.

The replay fingerprints the whole call, so the old recordings no longer matched. Answer quality had not regressed; it simply could not be checked.

Hale cases read their passage from the private brief PDF. The diagnostic skipped them, but the replay failure has the same cause.

| Case | Questions, 02094db | Questions, f0c170a | Call bytes, 02094db | Call bytes, f0c170a |
| - | - | - | - | - |
| bankstown-page9 | 105 | 113 | 95,200 | 98,577 |
| bankstown-water | 73 | 88 | 47,645 | 53,808 |
| bankstown-fire | 44 | 49 | 36,693 | 38,745 |
| bankstown-air | 75 | 98 | 54,612 | 64,698 |
| bankstown-nonpotable | 73 | 88 | 46,733 | 52,896 |
| commercial-local-exclusion | 75 | 98 | 54,366 | 64,452 |
| Every system labelled at once (bound, not a real passage) | 961 | 1,652 | 633,823 | 936,634 |

## Limits checked

- [Models](https://docs.typesafe.ai/models): jev-1.13 allows 64k tokens per request (`state` plus all questions), and 32k for `state` plus the longest question.
- [API](https://docs.typesafe.ai/api): no limit on the number of questions per call.
- [Fan-out](https://docs.typesafe.ai/patterns/fan-out): "putting all of the questions your system needs in a single request"; questions are evaluated in parallel.

The recorded `bankstown-page9` evidence call used 21,476 input tokens for 95,200 bytes, about 4.4 bytes per token. The largest real call is therefore about 22k tokens at HEAD, well within 64k.

**Risk recorded, not fixed here:** the all-systems bound was already over 64k at `02094db` (about 144k tokens) and is about 210k at HEAD. A passage whose labels span many families gets every leaf of each family plus all matching detectors. Packages that add questions to this call (WP-22 action, WP-27 signals) must measure their worst real passage against 64k.

## Decision D-17

Option **(a), re-record**. The enlarged calls stay within the documented limit, and one call per passage keeps TypeSafe's recommended pattern and the `AGENTS.md` rule of one fan-out per state. Option (b), a separate detector stage, would add a Jev call per passage for no measured benefit.

## Re-recording (live Jev, owner authorisation A1)

| Suite | Readings correct | Applied automatically | Forbidden applied | Input / output tokens before | Input / output tokens after | Jev latency p50 / p90 |
| - | - | - | - | - | - | - |
| Source | 15/15 | 15/15 | 0 | 105,732 / 23,867 | 120,485 / 28,245 (+14% input) | 276 / 327 ms |
| Hale | 12/12 | 12/12 | 0 | 200,906 / 43,770 | 212,755 / 47,909 (+6% input) | 296 / 410 ms |

Both replays now pass at HEAD without `-live`. The recordings stay in git-ignored `data/eval/profile/private/`. The previous recordings are kept outside the repo.

## Effect on users

The live passage cache (`passage_calls`) uses the same whole-call fingerprint. The next "Update project profile" on any existing project therefore re-reads every selected document's evidence stage once, with calls about 5-20% larger in input tokens on these cases. Filing is not affected: reading is a separate background job that the user starts.

## Follow-up: filing gates (F28, F29), owner delegated "you choose"

**F28 (filing accuracy baseline): accepted.** The baseline is the replay result after the live re-recording (`data/eval/intake/results/baseline.json`, hashes and aggregate metrics only). `intake-eval -replay` now passes. Accepting it sets a **regression floor, not a quality approval**: held-out title accuracy is 0.29, with 15 confident wrong titles, and that remains a known weakness to improve separately.

**F29 (replayed bench): root cause found; replay gate still red.** The 52 unrecorded requests are ordinary filing calls (date, discipline, kind, number, title), not OCR.

- `internal/eval/cases.go` files only page 1 of a multi-sheet drawing file. Its comment, "Intake reads the first page only", predates drawing-set expansion (migrations 006/007).
- The bench uploads whole files, so the app files every sheet, and those sheet requests were never recorded.
- The fix is for the intake eval to file, or at least record, every sheet the app files. That is filing work outside this wave, recorded as a follow-up.

**Live bench (A1) instead, dev machine, `-live`:** 184 files × 2 rounds, concurrency 4, 2 background Jev callers (241 calls), 10% of Jev requests stalled to their deadline.

| Path | p50 | p90 | Budget p50/p90 | Result |
| - | - | - | - | - |
| whole_intake (filing) | 377 ms | 1,245 ms | 1,000 / 2,000 | ok |
| project_profile_read | 8.5 ms | 12.0 ms | 50 / 150 | ok |
| profile_edit | 16.7 ms | 37.7 ms | 50 / 150 | ok |
| identity_text_extraction | 56 ms | 365 ms | 80 / 250 | p90 over |
| deterministic_field_rules | 1.0 ms | 3.1 ms | 1 / 1 | over (micro-budget; flaky on this host, F26) |
| jev_admission_request | 236 ms | 1,200 ms | 350 / 800 | p90 over (p90 is the stalled calls hitting the 1.2 s deadline) |

User-facing filing stays within budget under background reading (NW-REQ-272, 330, locally). The component overruns concern filing code this wave has not changed. Release evidence still needs the VPS (WP-71).

## M0: whole-file replay restored (AT-35), component gates reported

Run under the review amendment (plan §1.1 M0, §8.6; package WP-00 follow-up). Commit `d7f6699`.

**Change.** `internal/eval` now splits multi-page drawing sets into sheets exactly as `intake.ExpandDrawing` does, and records one Jev call per sheet. Each set's outcome is reported.

**Live re-recording (owner's standing permission for Jev calls):**

- 218 calls, 0 errors (129 before; the difference is 89 sheet calls).
- Drawing sets: 10 published, 1 stopped on a page not confirmed as a drawing (reported, not dropped).
- Scored metrics are unchanged field by field from the accepted baseline.
- Held-out title accuracy remains **0.29, with 15 confident wrong titles**. That is a known defect, not a quality standard.
- No new baseline was accepted. `intake-eval -replay` passes against the existing baseline.

**Replayed bench (`cmd/intake-bench`), dev machine with 12 logical CPUs; 184 files × 2 rounds, concurrency 4, 2 background Jev callers, 10% stalled.** There were **0 unrecorded requests**, so AT-35 is met (52 before this change).

| Path | p50 | p90 | Budget p50/p90 | Result |
| - | - | - | - | - |
| whole_intake | 331 ms | 700 ms | 1,000 / 2,000 | ok |
| identity_text_extraction | 58 ms | 316 ms | 80 / 250 | **FAIL** p90 |
| candidate_harvesting | 2.0 ms | 4.9 ms | 5 / 10 | ok |
| deterministic_field_rules | 1.06 ms | 3.6 ms | 1 / 1 | **FAIL** |
| jev_admission_request | 237 ms | 353 ms | 350 / 800 | ok |
| commit_sse_enqueue | 3.0 ms | 4.0 ms | 5 / 15 | ok |
| project_profile_read | 10.0 ms | 12.0 ms | 50 / 150 | ok |
| profile_edit | 34.9 ms | 40.2 ms | 50 / 150 | ok |
| other API paths | | | | ok |

**The two failures are pre-existing, not regressions.** The first replayed bench (`5ac5e4d`, 1 October) already broke them, with extraction p90 553 ms and rules p90 3.0 ms. Filing code is unchanged by this wave.

**Diagnosis, sequential and with no concurrency (throwaway tool, same 184 files):**

- **Extraction:** p50 28 ms, **p90 155 ms**, within budget. The slowest files are 0.8-1.3 MB drawing PDFs (petersham-hyd-h-201: 379 ms). The bench overrun is contention under load on this laptop.
- **Harvest plus rules:** p50 2.0 ms, p90 7.0 ms. The timings cluster on whole milliseconds (1.9994, 6.9998, 8.0137 ms…), which suggests this host's clock resolution is close to the 1 ms rules budget. That is a suspicion, not a proof.

**Gate status.** The required bench gate is still red on these two component budgets. Under plan §1.1 and work-packages §2.4, that blocks new Lane A merges. WP-12 is implemented and tested on branch `nw/wp-12-values` but **not merged**. Resolving the gate is an owner decision:

- (a) optimise extraction under load and the rule path to meet the current budgets; or
- (b) state where the component budgets are measured (for example the target VPS, per §8.3), and how a dev-host bench result counts.

Changing a budget is owner-only.

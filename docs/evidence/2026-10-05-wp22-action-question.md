# WP-22: action question and bounded evidence requests

Date: 5 October 2026. Lane A. Code implemented in the supplied `main` working
tree on base/head `80a5a4244623109b5da70b14859c303a46e056b8`. Not merged or
Verified: changed-question replays and live workload measurements are blocked
on the private-data transfer approval described below.

Contracts: implementation plan D-17/D-36 and §4.5, plan commit
`ceb67518a03701a7df948fe2f384cce816e0e404`.

## Changes

The existing evidence fan-out now includes a separate ten-option action
question for relevant system leaves, alongside presence and provider. Eight
action descriptions and boundaries, plus `several` and `not_stated`, come from
`knowledge/works/actions.yaml`. Wording is not duplicated in Go. The question
reads the existing `text` field; section/context only resolve its subject.
Q-WP-22-1 needs no field rename: the actual request includes both `text` and
`excerpt`, so the PRG's requested `text` reference is valid.

The label state resolves families and the evidence state resolves their
leaves. The action therefore joins the existing speculative leaf fan-out;
requiring a leaf already resolved before that call would omit first-read
actions or require another round trip. No new request, job kind or hot-path
Jev call is added.

Reconciliation uses an independent `action` threshold shape. Its ten-option
amber and green floors are null pending its own keyed calibration; no floor
is copied from presence or provider. Readings remain stored and unresolved.
Tests supply a separate action floor to verify application: it changes only
untouched proposed work, retains action SourceRefs, does not establish scope
on its own, and never overwrites an accepted item. Conflicting document action
is exposed in the compatibility scope note. `several` is explicitly Needs
mapping, never a work-item action; `not_stated` leaves the work-type default.

Question version advances `profile-3` to `profile-4`. Whole-call fingerprints
change, including the shared version on the label stage, so the next selected
document update rereads those states. Existing private recordings are retained
unchanged and correctly fail stale-recording checks. The evaluator's new
`-measure` mode inspects current requests with recorded labels without a model
call or an accuracy claim.

The client and background cache entry point preflight request size before
admission/transport. They reject the whole state, never truncate questions or
silently split it into calls. Errors include counts only, no document text.
Previously committed facts survive a failed oversized reading; the job reports
the failure and source text stays available for mapping.

The preflight estimates tokens from marshalled bytes, using the earlier
measured English ratio (about 4.4 bytes/token), rounded conservatively to four
and leaving 4k total/2k longest-context headroom. It is **not an exact provider
tokenizer or a guarantee for arbitrary input**. Provider validation is still
authoritative. The all-systems synthetic request is rejected locally; both
standing real fixture suites pass preflight. Actual added provider tokens and
latency remain unmeasured until the authorized transfer can run.

No new dependency, service, rule number or knowledge approval. A store test's
global pending-work assertion was corrected to examine its own tenants,
because benchmark fixture rows can legitimately remain in the test database.
Production drawing behavior is unchanged.

## Documentation followed

- [Fan-out](https://docs.typesafe.ai/patterns/fan-out): independent questions
  over the same state share one request.
- [How to build](https://docs.typesafe.ai/concepts/how-to-build-with-system-one):
  typed atomic questions, explicit state paths, code owns control flow.
- [Confidence](https://docs.typesafe.ai/confidence): separate thresholds by
  question and option count; missing calibration does not imply certainty.
- [Jev 1.13 limitations](https://docs.typesafe.ai/model-jaggedness/jev-1.13):
  literal boundaries, no generation/arithmetic, limited relevant context.
- [Models](https://docs.typesafe.ai/models) and [API](https://docs.typesafe.ai/api):
  64k total tokens, 32k state plus longest question, 255 choice options.

Read on 5 October 2026. The how-to page was unavailable through web extraction;
an approved read-only HTTPS fetch succeeded and was inspected locally.

## Verification and blocking approval

The full Go suite passed after adding the action integration, ownership,
Needs-mapping, preflight and rollback coverage. Follow-up tests for Jev,
jobs, store and the evaluator also pass after the final request-size and
first-read action changes.

Current request measurement, without private text in the output:

| Suite | Label + evidence calls | Added action questions | Added JSON bytes | Largest request bytes | Largest estimated input tokens |
| - | - | - | - | - | - |
| Source | 14 | 91 | 219,794 | 222,721 | 55,681 |
| Hale | 22 | 187 | 455,354 | 214,675 | 53,669 |

Counts use the unchanged recorded labels. They are request measurements, not
live execution counts or actual added provider tokens. Both label and evidence
fingerprints advance because of the shared question version. Repository-wide
document counts are not inferred from this small fixture set.

The first local timing run failed: profile edit p50 90.523 ms (50 ms budget),
profile rebuild p90 725.413 ms (300 ms). Retained as
`bench/results/2026-10-05-wp22-first.json`, with raw samples in ignored
`.tools/wp22-first-samples.json`. Nine early rebuilds were over 300 ms, then
the normal cluster returned to roughly 60–80 ms; loaded HTTP edits also had
large early delays. No budget was relaxed. A controlled follow-up run is
recorded separately; even a passing follow-up does not erase these outliers.

The controlled final run passed the development endpoint/end-to-end gate:
profile edit p50/p90 37.3/115.0 ms (50/150 budget), rebuild 67.7/71.1 ms
(100/300), works read 1.1/2.0 ms and write 7.5/10.8 ms (100/250).
Filing rounds were 381.2/1,245.2 and 381.3/1,305.9 ms (1,000/2,000).
Exact results: `bench/results/2026-10-05-wp22.json`. No benchmark-time
compilation or diagnostic runs competed with the final run. The first run's
outliers remain unexplained; this is local timing evidence, not target-VPS
release acceptance. Component limits reported over by the benchmark remain
visible in the result file and require the planned target-VPS checks.

Commands (dedicated test database and bundled Go runtime):

```powershell
go test -p 1 ./...
go test -p 1 ./internal/jev ./internal/jobs ./internal/store ./cmd/profile-eval
go run ./cmd/profile-eval -measure
go run ./cmd/profile-eval -measure -cases data/eval/profile/private/hale-cases.json -recording data/eval/profile/private/hale-recording.json
go run ./cmd/profile-eval
go run ./cmd/profile-eval -cases data/eval/profile/private/hale-cases.json -recording data/eval/profile/private/hale-recording.json
go run ./cmd/intake-bench -out bench/results/2026-10-05-wp22.json -samples-out .tools/wp22-samples.json
```

Source replay: `bankstown-page9: recording is stale`.
Hale replay: `hale-description: recording is stale`.
These are **red gates**, not waived or replaced by synthetic results.

Automatic approval review rejected the attempted live re-recording command:
it would send private source/Hale fixture passages to `api.typesafe.ai` using
the local Jev key, and the reviewer required explicit authorization for that
payload and destination. No live request executed and no credential was
printed. A precise approval question is pending in this chat. The plan's
historical A1 permission was insufficient for that automatic review; no
alternate path or indirect execution was attempted.

Continue independent local packages while this approval is pending. After
approval, remeasure against the final questions, back up the existing private
recordings locally, re-record both suites, replay unchanged keys, and report
actual calls/tokens/latency. Do not mark the package Verified or merge with
these gates red. Owner-reviewed action calibration is also still outstanding.

## Traceability

Implemented in code, pending integration/review: NW-REQ-009, 061, 066, 115
(action application), 123, 124, 125, 127, 128, 129, 130, 270, 367, 387.
NW-REQ-126's independent threshold shape and fail-closed behavior are present;
calibration/owner acceptance is not done. NW-REQ-131/381 workload measurement
is partial (local request counts and size only). NW-REQ-315/385 and AT-09/25/33
remain partial until real replay, workload and the wider quality gates pass.

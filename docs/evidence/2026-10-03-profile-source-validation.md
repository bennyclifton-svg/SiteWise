# Project profile source validation

Implemented the [source-record design](../design/2026-10-03-profile-source-records.md).

The existing Bankstown PPR was reprocessed on the development service using
Jev 1.13.0. All 83 PDF pages produced text; 1,747 source passages were saved
and classified. All extraction, label and evidence jobs completed. The old
500-passage truncation is gone. Exact source text, page links and unresolved
requirements are now inspectable in the profile.

The complete run captured eight storeys, 33 sole occupancy units and the
31 December 2014 consent date. Basement levels no longer become parking-space
counts. A final audit found ceiling dimensions and electrical fittings offered
as bedroom counts; a count-specific harvesting rule and regression test now
reject those candidates. The saved profile was refreshed using checkpoints.

This is processing coverage, not a claim of complete semantic recall. The first
complete run classified 915 passages as background/reference, mapped 127, and
retained 705 for mapping or review. These include requirements outside existing
profile fields and uncertain readings. Jev classifications can still be wrong;
source text remains available regardless. Machine values remain provisional
and user-entered values remain authoritative.

## Checks

- `go test ./...` passed with the isolated `sitewise_test` database; targeted
  profile/worker tests passed again after the bedroom-count correction.
- Web production build passed. The three profile browser scenarios passed
  across the verification runs, including missed completion events and source
  filters, original-page links and 390px mobile layout. The Windows test-server
  cleanup hung after results and was stopped; test assertions had completed.
- Private replay gate: 15/15 expected readings captured, 14/15 automatically
  applied, zero forbidden values applied. The remaining correct reading was
  below the provisional confidence floor and retained for review.
- Eight checked cases contain six PPR excerpts and two synthetic controls.
  This small suite is a regression check, not threshold calibration or a
  whole-document accuracy estimate. Source cases and recordings are ignored
  under `data/eval/profile/private/` because they contain private source text.
- Recorded Jev call p50 253ms / p90 282ms; 68,708 input and 19,166 output tokens
  for that suite. These are provider-call timings, not full-document latency.
  Family routing reduced suite input by approximately 75% versus asking every
  leaf question in the first stage.
- Source-read API tests enforce local p50 50ms / p90 150ms budgets. Coverage,
  pagination, organization isolation, exact offsets, final-page survival,
  cached replay and lease renewal are covered by regression tests.

Run `go run ./cmd/profile-eval` for private deterministic replay. `-live`
records new paid TypeSafe answers; it requires the configured Jev credential.
Do not publish the private fixtures or recordings.

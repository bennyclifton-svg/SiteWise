# WP-23: part-based location and fact placement

Date: 5 October 2026. Lane A. Local changes on base/head
`80a5a4244623109b5da70b14859c303a46e056b8`; no commit, merge or deployment.
Plan: `ceb67518a03701a7df948fe2f384cce816e0e404`, D-26 and §4.6.

## Outcome

The existing label-stage `source.scope` question now offers the site's named
parts, each described by label and kind, plus whole-project, specific,
multiple and not-stated options. Code orders parts by ID and maps `p1...pn`
back to the exact labels. The implicit whole part is excluded so its lazy
creation cannot invalidate an otherwise identical retry.

The worker reads parts once per stage. An applied location is copied to
`profile_facts.part_label`; the evidence stage reuses the matching cached label
call rather than asking another question. Parts and applied placement are
included in request fingerprints. After a rename, the old location cache is
not reused and old unmatched labels fall back to the whole part, as specified.
That fallback is a known limitation, not an inferred new location.

Located evidence cannot turn green, even with multiple agreeing sources.
Explicit user values remain final. Location floors use their own option count,
`location.n<k>`; the new entries are null pending owner calibration. No floor
means no automatic placement, rather than borrowing the old generic 0.6 floor.
This follows https://docs.typesafe.ai/confidence. Question version is now
`profile-5`; no answer key or knowledge status changed.

## Checks and re-read cost

- Focused tests cover exact-label mapping, similar labels/kinds, whole-only
  options, stable ordering, independent floors, generic/unknown answers,
  amber-only evidence, rename fallback, retry cache reuse, tenant/site isolation
  and label-to-evidence placement without another call.
- The initial full Go suite passed with the dedicated database.
- Local request measurement completed for both private replay suites. This
  measures JSON request sizes only, not actual tokens, latency or accuracy.
  Source/Hale requests remain within the preflight estimate. Actual re-read
  cost and accuracy remain unverified until fresh live recordings are allowed.
- The source and Hale recordings remain stale after the question change.
  Automatic approval review rejected sending their private passages to
  `api.typesafe.ai` without explicit payload/destination authorization. The
  question remains pending; no live calls were made or indirect path attempted.
- No new dependency, service, migration or model round trip.

## Performance investigation

The first two original-workload runs failed. They are retained as
`bench/results/2026-10-05-wp23-first.json` and
`bench/results/2026-10-05-wp23-repeat-before-fix.json`.
Profile-edit medians were 89.6 and 89.2 ms against 50 ms; the second rebuild
p90 was 717.8 ms against 300 ms. No budget was relaxed.

A small database fixture isolated write overhead: pure profile computation
took 1–2 ms, work-item synchronization about 30 ms and 278 profile INSERTs
about 29 ms. The same edit took about 67 ms. Replacing individual work upserts
with one recordset INSERT reduced it to about 47 ms; replacing the profile
INSERT batch with transactional COPY reduced it to 25–28 ms. Existing
conflict guards, org/site ownership and database constraints are preserved.
The first full run after that change passed edit p50/p90 at 37.5/61.4 ms but
still failed rebuild p90 at 699.8 ms. It is retained as
`bench/results/2026-10-05-wp23-after-bulk.json`.

The separate rebuild spike was then traced to a cached PostgreSQL generic
plan prepared while `passage_sources` was nearly empty. It performed a
sequential scan with about 6,755 rejected rows **for each of 1,153 facts**,
taking roughly 675 ms and 188,205 buffer hits. A fresh connection's query
plan was fast, which is why simply explaining the SQL initially missed it.
The captured prepared plan is in ignored `.tools/wp23-prepared.log`.
JIT was unavailable on this PostgreSQL build; pure computation, large
work-item writes and a fresh generic plan did not explain this second defect.

The existing stale-statistics snapshot regression now prepares the query on
empty temporary tables before loading 1,000 facts and 6,999 passage sources.
It failed before the fix at 1,053.3/1,107.0 ms p50/p90. A first aggregation
approach passed that test but added complexity and did not address the full
HTTP response: source coverage had the same cached-plan vulnerability. Its
6,000-passage regression timed out after preparing the query before upload.

The final correction is smaller: **only these two size-sensitive queries use
`pgx.QueryExecModeExec`**, keeping bound parameters and a single round trip
while planning each execution. The existing SQL retains its exact-ID lateral
lookups and statement-level snapshot. Other queries keep their statement cache.
This uses the existing pgx dependency and changes no server setting or query
semantics. The existing tests still verify missing-source retention, original
metadata, tenant isolation, superseded-document exclusion and all coverage
counts. The cache-before-upload tests now pass at 11.3/12.0 ms for snapshots
and 19.4/19.7 ms for coverage (`.tools/wp23-both-plans-green.log`).

Diagnostic runs include extra instrumentation, and one reduced run has
insufficient samples; they are retained as diagnostic evidence, never release
or final timing proof. Their names contain `diagnostic`. The original workload
is rerun without instrumentation for final acceptance.

The diagnostic followed the local diagnose skill's reproduce/measure/fix loop.
Existing project design/plan documents supplied its context; no issue-tracker
or agent-instruction reconfiguration was needed for this authorized AFK work.
Temporary instrumentation was removed from production; the timing and import
harnesses were moved to ignored `.tools/wp23_diagnostic_test.go`,
`.tools/wp23_cold_diagnostic_test.go` and `.tools/wp23_debug.go`.
The final full Go suite passes (`.tools/wp23-complete-go.log`). The original
184-file, two-round workload also passes without instrumentation; final
results are in `bench/results/2026-10-05-wp23.json` and
`.tools/wp23-complete-bench.log`. Final p50/p90 milliseconds:

| Path | Measured | Budget |
| - | - | - |
| Profile edit, existing benchmark project | 40.8 / 66.0 | 50 / 150 |
| Spec Home code-only rebuild | 43.6 / 45.4 | 100 / 300 |
| Works read | 1.0 / 2.0 | 100 / 250 |
| Works write | 6.0 / 8.0 | 100 / 250 |
| Filing | 328.4 / 929.6 | 1,000 / 2,000 |

The **full Spec Home HTTP edit is now 80.3/82.3 ms**, maximum 84.4 ms. Before
the response fix it was 89.4/3,984.0 ms, maximum 4,117.1 ms; that run remains
in `bench/results/2026-10-05-wp23-before-response-fix.json`. D-18 reports this
large-project diagnostic separately from the existing profile-edit benchmark.
The cached-plan coverage regression explained and corrected the slow response,
instead of treating the earlier green rebuild gate as sufficient evidence.

Development component limits remain over for identity extraction p90
384.8 ms (250 budget), and deterministic rules about 1.0/4.0 ms (1/1 budget).
Those limits are judged on the target VPS under the existing policy, not
waived. Private replay, calibration and owner/reviewer gates remain below.

## Traceability and remaining gates

NW-REQ-009/061/066/074/132/133/134/135/270/367/387 have local implementation
and focused tests. NW-REQ-381's actual reread cost remains unverified;
NW-REQ-315/385 require integration/review and green required gates. AT-10's
location portion is covered, while its wider package/cost behavior belongs to
later work. Owner threshold calibration, fresh private replays, target-VPS
timing and independent review remain; this handoff does not mark the
package Verified.

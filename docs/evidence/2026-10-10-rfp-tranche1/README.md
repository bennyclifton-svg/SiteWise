# Consultant RFP tranche 1: build evidence

Date: 10 October 2026. Branch `rfp/golden` (worktree `../sitewise-rfp`).
Plan: [2026-10-09-consultant-rfp-build.md](../../plans/2026-10-09-consultant-rfp-build.md).
Content changes for review: [phase-a-changes.md](../../rfp-golden/phase-a-changes.md).

## What a user can do now

Reports → RFP → a services package whose discipline is Architect, Town
planner, Traffic, Civil or Structural → Create → Assemble. The draft is built
from that discipline's rules (`knowledge/rfp/`), not the generic RFP. Beside
the draft, "Finish this RFP" lists what blocks issue first:

- confirm the appointment;
- client quantities;
- issue details;
- unknown or conflicting facts;
- source changes under a setting.

After those come the stages, facts, allowances, I/O/X positions and weights.
Every choice saves on change and survives Refresh and the next draft after
issue. A package titled "Civil" without a discipline id is offered the civil
RFP and switches only when the user confirms.

Issue is refused (`rfp_open_checks`) while a blocking item is open. It is also
refused (`unreviewed_clauses`) while a discipline's rules are still `draft`;
all five are draft, so nothing can be issued until the owner reviews them.

## Checks run

| Check | Result |
|---|---|
| `go test ./internal/...` | pass (all packages) |
| Selection answer keys (12 scenarios, 27 scenario × discipline runs), `internal/rfp/answerkey_test.go` | pass |
| Markdown snapshots, Arthur St (5) and Yarramundi (2), `internal/rfp/testdata/` | pass; owner comparison with `docs/rfp-golden/*/rfp.md` still to do |
| Store: settings, isolation, stale version, refresh survival, source-change conflict, issue gates, issued settings immutable, carry-forward after issue | pass (`internal/store/report_settings_test.go`) |
| HTTP: PUT/DELETE settings (200/400/404/409/422), access sweep, discipline suggestion | pass |
| FK sweep with the new `report_settings` table | pass |
| `python tools/check_knowledge.py` | 0 errors, 2 warnings (both existed before) |
| `tools/test_check_knowledge.py` | pass |
| Playwright full suite | 32 passed. On earlier full runs, `work-layout` and one report alert assertion each failed once; both pass alone and on rerun, and neither is changed here. |
| Design detector on the new UI | no warnings in new code; one advisory (16px heading, matches existing report headings) |

## Page counts (issue renderer, body size, nothing dropped; limit 3)

| Scenario / discipline | Pages | | Scenario / discipline | Pages |
|---|---|---|---|---|
| arthur-st / architecture | 2.62 | | mixed-2-6-7a / architecture | 2.56 |
| arthur-st / civil | 2.72 | | mixed-2-6-7a / civil | 2.57 |
| arthur-st / structural | 2.58 | | mixed-2-6-7a / structural | 2.52 |
| arthur-st / town-planning | 2.45 | | existing-3-9c-repair / architecture | 2.53 |
| arthur-st / traffic | 2.28 | | live-hospital / architecture | 2.58 |
| yarramundi / architecture | 2.74 | | unknown-facts / architecture | 2.77 |
| yarramundi / civil | 2.78 | | planning-advice-only / town-planning | 1.60 |

All 27 runs fit within three pages; the longest is 2.78.

## Speed (dev host, `intake-bench -files 25`, replayed Jev; gate passed)

| Path | p50 | p90 | Budget |
|---|---:|---:|---|
| report_setting (new) | 54.4 ms | 56.2 ms | 100 / 250 ms |
| report_assemble | 8.2 ms | 12.0 ms | 300 / 1,000 ms |
| report_read | 16.7 ms | 17.8 ms | 100 / 250 ms |
| report_write | 5.0 ms | 6.1 ms | 100 / 250 ms |
| report_issue | 15.2 ms | 16.8 ms | 100 / 250 ms |
| whole_intake | 425 ms | 911 ms | 1,000 / 2,000 ms |

On populated scenarios, resolve plus sections takes about 1 ms at p90. The
page-fit PDF render takes 32 ms at p50 and 36 ms at p90. A setting change
writes, re-resolves from the frozen drivers and returns the view with its
page fit; it never reloads the project and never calls Jev.

Not measured: the RFP paths while filing runs concurrently. The component
budgets judged on the VPS (extraction and rules) were over on this host, as
they were before this work.

## Driver report

[driver-report.md](driver-report.md) lists, for each scenario and
discipline, which facts came from the profile, which were set for the RFP and
which stay unknown (O9). The most common unknowns are site flags that the
profile does not ask about yet: OSD, public road, excavation next to
boundaries, panel and objections. They are candidates for profile questions.

## Open for the owner

1. Review [phase-a-changes.md](../../rfp-golden/phase-a-changes.md) and the
   Markdown snapshots, discipline by discipline.
2. Approve or correct the twelve answer keys.
3. Mark each discipline `status: reviewed` with `reviewed_by` and
   `reviewed_on` when its rules are right. Issue unblocks per discipline.
4. Regulated design: the NSW applicability rule is unverified. Until a
   verified rule exists, the user records the answer per project (O5).
5. Unsupported planning pathways (CDC, REF, modification, planning proposal,
   permit) need their own stage sets before a planner RFP can issue (O6).
6. Sensitive content: `docs/rfp-golden/` (fee ledgers from real proposals)
   and the answer keys (real project names) are committed on this branch.
   The repository is public; decide what may be pushed before merging.

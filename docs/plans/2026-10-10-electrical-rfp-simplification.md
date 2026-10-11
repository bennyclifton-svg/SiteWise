# Electrical RFP simplification

Owner direction approved 10 October 2026. Lane A: reduce time to a usable RFP while retaining saved wording, source traceability and issue checks. No new dependency, service or Jev call.

Create opens an assembled draft from saved project information. It does not update Profile. Reopening an existing draft does not regenerate it; Refresh explicitly incorporates saved changes while retaining protected wording. Incomplete reading remains visible but does not interrupt drafting. An issued report remains immutable and the next draft is explicitly requested. Concurrent Create requests reuse one draft; repeatable-read conflicts retry after locking the report.

## Audit and scope

Electrical had nine unconditional, blank compound quantity controls. Architecture, structural, civil, town planning and traffic already used default and conditional allowances. Mechanical, hydraulic and fire services share Electrical's older pattern; their templates are outside this change.

Electrical now has editable defaults of four design meetings and one consolidated review round per appointed design stage, and one accessible investigation visit when applicable. Construction inspections need a common quantity only when construction is included in the base appointment. Detailed asset, test, duration and authority schedules belong in separately described consultant responses, never a single number combining unlike units.

Unknown authority assistance, generation/EV and ICT are explicit priced options. Conflicting evidence and material responsibility questions still block issue. All issue decisions remain available in an expandable summary. Old saved compound counts remain visible with their values and require review before removal; no guessed conversion occurs. Existing saved drafts adopt the new template on Refresh. Knowledge remains draft pending owner review.

## Verification

- Production frontend build passes.
- Knowledge checker: zero errors; two existing works-default mapping warnings.
- Full RFP suite, selection answer keys and golden render checks pass.
- Full report and store suites pass; focused API/store regressions cover direct assembly, idempotence, simultaneous requests, source changes, protected edits and wrong-organisation access.
- Six procurement/report Playwright scenarios pass, including immediate Electrical content, collapsed decisions, editable defaults, mobile width, protected wording, issue history and PDF download.
- Broader HTTP API tests retain the pre-existing TestProfilePlacesInputsInBuildOrder failure (unmapped profile flag homes).
- Speed budgets: report writes 100/250 ms p50/p90, assembly 300/1000 ms, filing 1000/2000 ms. Recorded local benchmark results are in .tools/electrical-bench-results.json and .tools/electrical-bench.log; these are development measurements, not live Jev/VPS release evidence.

No actual Petersham profile, RFP wording or issued PDF was edited by this implementation.

Final recorded development timings (p50/p90): report writes 8.3/48.0 ms against 100/250; assembly 12.8/15.5 ms against 300/1000; reads 1.1/22.8 ms against 100/250; whole filing 18.2/639.6 ms against 1000/2000. The overall gate still fails the existing Spec Home profile-edit p50 (51.7 ms against 50 ms). Provider component timings/sample counts are reported for target-VPS verification, not claimed as a release pass.

The local development app was rebuilt and restarted; HTTP 200 and the new frontend asset were verified on port 8080. Existing Petersham Electrical draft: click Refresh draft once to adopt these rules while retaining protected wording.

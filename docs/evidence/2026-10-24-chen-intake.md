# Chen Residence intake repair

24 October 2026. Lane A: document identity and filing trust.

## Change

Markdown now has a format-specific identity harvester. It reads literal leading
headings and labelled document references, versions and issue/report dates.
It does not treat licence/ABN/job numbers, inspection dates, body references or
attachment metadata as the uploaded document's identity. Conflicting values
remain unresolved. The original bytes and candidate source offsets are retained.

An email requires a From/To/Subject header block. Its kind is Correspondence,
its title is the first message subject, and its filing date is the latest
explicit message-header calendar date. Code compares dates; an incomplete
thread does not establish that date. Email body and attachment revisions and
numbers are not document identity. Number/revision remain absent unless the
document itself supplies them. The register column now says Doc No.

Unresolved classification still uses the existing single Jev fan-out. No new
provider, service, dependency, confidence cutoff or green AI acceptance was
introduced. This follows the existing [pre-parsed extraction pattern](https://docs.typesafe.ai/cookbooks/pre_parsed_value_extraction_cookbook):
code supplies literal candidates and Jev chooses only unresolved values.

## Saved results

All 12 source files were inspected and their persisted decisions rechecked.
Deterministic repairs were committed through the store's filing transaction,
scoped to Chen's project and org, with expected decision versions and user
correction protection. Existing Jev classifications were retained. No private
source text is included in this note; local before/after snapshots are in
ignored `tmp/chen-review-records.json` and `tmp/chen-after.json`.

| Field | Before | After |
|---|---|---|
| Titles | 0/12 | 12/12 |
| Kinds | 8/12 | 12/12; four email threads now Correspondence |
| Dates | 4/12 | 11/12; inspection-only date removed |
| Document references | 1, incorrect licence number | 3 explicit references |
| Revision | Email incorrectly assigned v1.1 | Email blank; programme v0.1 |
| Discipline | 9/12 | 6/12 evidenced; three unsupported assignments cleared in the approved live recheck |
| Lifecycle | 0/12 | Unchanged; no invented values |

The planning memo's Town Planner classification remains questionable and is
not claimed as fixed. Automatic approval review rejected the live Jev recheck
because of private document transfer; explicit transfer approval was requested.
The blocked probe did not send document excerpts. The local server was rebuilt
and restarted with normal OCR/network access; backlog profile reading remains
disabled pending approval. The repaired rows were verified in the running UI.

## Verification

- Identity and intake suites passed.
- Added regression cases cover own metadata, licence-only sources, job numbers,
  unrelated body/attachment references, conflicting labels, invalid dates,
  code fences, literal source spans and user corrections including clearing.
- Web production build and all four intake/register-edit browser tests passed.
- 20 Markdown filings with an unavailable-provider fixture: p50 24.15 ms,
  p90 66.98 ms, against 1,000/2,000 ms. This is degraded-path evidence only.
- Whole-filing benchmark: 24 corpus uploads, four concurrent uploads, two
  background callers, recorded provider latency and 10% injected stalls;
  p50 663.178 ms, p90 1,526.259 ms, max 1,609.342 ms. Filing gate passed.
  This corpus benchmark is not a live Chen/TypeSafe or target-VPS measurement.
- Overall benchmark failed on profile_edit (51.644/199.338 ms against 50/150)
  and spec_home_profile_edit p50 (54.392 ms against 50). Some release-only
  component budgets were also exceeded on this development host.
- Broader store suite failed TestPrepareProgramIsIdempotentAndLinksPackageStages
  and TestProgramReview in the pre-existing modified Program area; no Program
  source was changed for this intake task.
- Recorded intake accuracy replay could not complete: 129 requests had no
  recorded response. No accuracy-gate pass or new calibration is claimed.

Logs and samples: `tmp/chen-tests.log`, `tmp/chen-focused-tests.log`,
`tmp/chen-browser-tests.log`, `tmp/chen-intake-replay.log`,
`tmp/chen-bench-results.json`, `tmp/chen-bench-samples.json`.

## Approved live recheck

The owner explicitly approved sending the 12 intake excerpts to TypeSafe Jev.
The first live preview reproduced the subject/issuer confusion for the planning
memo, and the programme kind abstained on that run. The Markdown-only issuer
question now asks for an explicitly evidenced author/sender profession, not a
subject, recipient, recommended consultant or attachment author. Its request
and saved decision version is `intake-77+markdown-1`; PDF/DOCX/XLSX questions
are unchanged. The existing provisional amber cutoffs are retained, not promoted
to calibrated or green acceptance. The explicit leading Master Programme/Program
heading is a deterministic Schedule lookup.

Two live passes through the real filing runner persisted all 12 files with
identical field values and bands on both passes. Each pass reads the stored
file, extracts identity, makes one real Jev request, commits decisions and
enqueues the filing event. User decisions remain protected by the existing
versioned store transaction. The planning memo, programme and owner appointment
email now abstain on discipline: their own sources do not state the issuer's
profession. This is deliberate uncertainty, not a failed reading.

24 live samples: p50 **216.110 ms**, p90 **244.215 ms**, maximum **317.946 ms**,
against **1,000 / 2,000 ms**. Exactly 24 Jev calls, zero call failures, no grey
results. Timing includes the runner and a saved-decision verification read;
it excludes the administrative reset to pending, upload/network transfer,
HTTP dispatch queue and client receipt/rendering. The prior full HTTP/SSE replay
benchmark remains separate evidence for that envelope. These are sequential
local re-filings, not target-VPS burst-load release measurements.

Final identity/intake suites passed again after the classification refinement.
The final build is deployed locally. Automatic review refused restoring queued
background readings outside the approved Chen scope, so the server retains
`-background-backlog=false`; this does not prevent new uploads or explicit
profile updates. No additional private backlog was sent as part of this task.
Local live output: `tmp/chen-live-after.json`; final test log:
`tmp/chen-live-tests.log`. Previous broader gate limitations above remain.
Original upload timing was not recorded, so no before/after speed delta is claimed.

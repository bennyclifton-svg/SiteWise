# WP-K4 investigation proposal wording

Date: 6 October 2026. Status: bounded draft-catalogue correction implemented.

Lane A. Outcome: an investigation proposal asks the reviewer to establish the
condition and required response instead of directing physical work based only
on a broad applicability predicate. No new questions, services, dependencies,
record IDs or numeric requirements.

## Review and changes

Inventoried the catalogue's 361 investigation labels, then selected labels
beginning with explicit physical-action verbs: Provide, Remove, Clear, Clean,
Shore, Lower, Upgrade, Replace, Form, Resize, Articulate, Land, Reject, Isolate
or Size. Read the cited batch-2 rows for all 47 selected records, including all
rows of merged records. Recast each selected label as a check, survey or
assessment of the source's stated physical constraint. Selection is bounded;
this is not a claim that a verb filter establishes engineering correctness or
finds every unsuitable label.

Examples: "Shore the trench" becomes an assessment of ground stability and
temporary-support requirements; "Clear the path of the columns that exist"
becomes a survey of columns and check of the vehicle swept path. The trade-waste
record's new wording covers both pit covers and pipe cover, matching its two
source rows. Hazardous-material records request assessment and management
requirements rather than prescribing removal methods.

Changed records by cluster: envelope 2, fire 1, services-wet-air 28, structure
16. The complete ID, source, old label and new label inventory is
`docs/unforeseen/k4-investigation-label-review.json`, explicitly unreviewed.
The source dataset and triage rows are untouched.

Compared parsed records before/after after normalising the intended label
change: every other field is identical. This preserves predicates, attachments,
signals, contract wording, severity, provenance and draft status. Signal
questions and their runtime routing are unchanged. Record hashes and proposal
fingerprints may change as intended; previous decisions do not silently approve
changed proposal content.

## Validation and limits

`python tools/check_knowledge.py --strict`: zero errors; three existing
package-default mapping warnings; 2,000 dataset rows and zero pending.
`go test -p 1 ./internal/knowledge ./internal/works`: passes (3.422/1.859 s).
Refreshed the current K4 hash inventory. No test asserts that these draft
records are useful or legally correct.

This wording-only change adds no runtime operations or Jev questions. No new
latency result is claimed. Budgets remain filing 1,000/2,000 ms, profile edit
50/150 ms and rebuild 100/300 ms (p50/p90); the existing latency failure remains
open and blocks a passing merge/release claim.

WP-K4 is incomplete. Remaining wording beyond this selection, loose merges,
generated contracts, severities, predicates, suppression adequacy and owner
review still need attention. K6 usefulness and M1 approval are not established.

## Follow-up: premature design choices

Examined a further 38 labels beginning with Design, Redesign, Set, Keep, Choose,
Select, Match, Plan, Agree, Resolve, Do not, Detail, Fit or Open. Read each
cited batch-2 row, including merged rows. Corrected 35 more labels: one
electrical-comms, one fire, nine services-wet-air and 24 structure. The exact
review inventory now contains 82 corrections in total. Only the label field
changed in each record, checked by parsed before/after comparison.

Retained three investigative labels: comparing hazardous-area equipment ratings
with the zone drawing; opening up to identify an existing ceiling build-up;
and opening enough ceiling to inform a concealed-structure provisional sum.
Their verbs alone do not make them premature construction instructions.

For merged records, the new labels preserve all source concerns: forklift
routes include both joints and pits; existing-floor assessment includes fire
and acoustic performance; crossover review includes geometry and sight
distance. These are requests for assessment, not new technical requirements.

Follow-up strict validation: zero errors, the same three warnings, 2,000 rows
and zero pending. Knowledge and works tests pass (3.421/1.837 seconds). Refreshed
the current catalogue hashes. No new latency claim, dependency, runtime
question, content approval or milestone completion follows from this pass.

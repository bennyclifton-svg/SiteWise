# Bankstown processing status

Lane A defect: show the actual Project Profile job state so users can retry
stopped processing. Profile reads retain the p50 50ms / p90 150ms budget.
No new dependency, automatic retry, or change to profile results.

## Profile button

Bankstown had one failed label job (`jev circuit open`) and a queued evidence
job behind it, with no active processing. The UI treated any pending job as
active and displayed "Updating project profile" alongside the stopped warning.

The profile read now counts distinct documents with unexpired processing
leases, scoped to the org and project. Failed work takes precedence and offers
"Retry project profile"; queued work without a live lease says "Profile update
queued". Only live processing says "Updating project profile". Existing polling
refreshes this state if an SSE completion event is missed.

Validation: store and HTTP API tests pass; web build passes; two browser
regressions pass (failed work with a queued dependency, and missed completion
SSE). Store regression covers queued, active and expired leases. Thirty local
Bankstown HTTP reads, including PowerShell client overhead, measured p50
48.45ms / p90 54.65ms. After restart, the live API reports pending=1, active=0,
failed=1; `/dev/build` reports both server and web current.

## OCR diagnosis (parser changes remain proposed)

CC-06 and CC-07 are saved as filed with `ocr_review`. OCR reads `LEVEL 2` and
`LEVEL 3` beside `DRAWING NAME`, but harvested candidates do not recognise them
as the owning title. Filename candidates include the trailing revision and
compete with those printed titles. Printed drawing numbers are split into
`DWG.No. CC-` and separate `06` / `07` runs, which the general harvester does
not join. Metadata Jev cutoffs remain uncalibrated, so those ambiguous fields
are withheld; a high confidence value alone does not authorize applying them.
CC-06's discipline confidence was below its existing amber threshold.

Deleting and uploading the same bytes does not fix field selection. Proposed
follow-up: evidence-based label geometry and fragmented-number parsing with
these two documents as regression cases, then recovery for missing fields on
already-filed documents while preserving user edits. No thresholds were lowered
and no Bankstown documents or jobs were reset during this diagnosis.

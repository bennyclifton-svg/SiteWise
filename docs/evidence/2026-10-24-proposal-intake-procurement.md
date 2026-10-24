# Evidence: fee proposals into Procurement (Kavanagh), 24 October 2026

Plan: `docs/plans/2026-10-24-proposal-intake-to-procurement.md`. Branch
`intake/proposal-procurement`. Fixtures: the four synthetic Kavanagh fee
proposals in `internal/proposals/testdata/kavanagh/`.

## End to end (this branch's server, fresh database, live Jev)

Four Markdown files uploaded to a new project with no packages, then one
"Update project profile".

| | Before (main, 13:33) | After (branch) |
|---|---|---|
| Filing time per file | ~0.25 s | 0.21–0.27 s (budget p50 1 s) |
| Kind / lifecycle | commercial amber / blank | Commercial / Consultant, green by rule |
| Number | blank ×4 | QUA-KAV-2601, CST-KAV-0126, FLH-KAV-014 green; VCA-KAV-26-07 blank (number shape) |
| Procurement rows | none at read time; 4 proposals "need a package" | Architect, Structural, Hydraulic, Quantity Surveyor (origin document); 0 unassigned |
| Firm name | `**Quoin Architecture Pty Ltd**`; Vertex none | clean name ×4, green by rule |
| Firm address | blank ×4 | office address ×4, green by rule |
| GST basis | unknown ×4 | ex ×4 |
| Fee on the row | — | stated total ×4 ($96,000 / $41,800 / $32,500 / $45,000), marked stated |
| Received date | upload day | proposal date (20–24 Jan 2026) |
| Stage fees (architect) | none | Concept $24,000 amber, Design development $43,200, Construction support $28,800 green; $24,000 offered for Concept and Schematic, counted once, flagged |
| Client | never asked | Kavanagh Property Holdings Pty Ltd as trustee for the Kavanagh Family Trust (4 documents) |
| Pool | blank | included (2 documents) |
| Retaining walls | blank | included (site and building) |
| State evidence | consultants' office addresses only | also each proposal's site block |
| Label/evidence calls | 52, 382k input tokens | 48, 355k (signature and billing blocks not sent; more, smaller passages) |
| Work type | new (amber) | **conflict**: the QS scope "cost advisory services" read as advisory 0.96 |

The work-type conflict is the consultant-service-versus-project confusion
that the profile-9 header wording in progress elsewhere addresses ("a
consultant's advice is not an advisory-only project"). Re-check after merge.

## Live question evaluation (`data/eval/profile/kavanagh/`)

Seven passage cases, `go run ./cmd/profile-eval -live -cases
data/eval/profile/kavanagh/cases.json`.

| Question version | Correct | Forbidden applied | Notes |
|---|---|---|---|
| profile-5 (main) | 3/7 | 3 | pool not stated 0.73 (architect), 0.52 (structural); landscape and survey "not included" from fee exclusions |
| profile-10 (branch) | 4/5 | 0 | pool included 0.88 / 0.79; a generic "lift design … excluded" list does not include lifts |

Remaining miss: the hydraulic "pool backwash interfaces" passage labels
pools at 0.49, under the uncalibrated 0.5 floor; not lowered. Recorded call
latency p50 241 ms, p90 300 ms.

## Tests

New: `internal/proposals/kavanagh_test.go`, `internal/store/proposal_package_test.go`,
`internal/intake/proposal_label_test.go`, `internal/intake/fee_proposal_test.go`,
`internal/jobs/passages_test.go`, `internal/profile/addressee_test.go`,
`internal/profile/postcode_test.go`. Proposals, intake, jobs, profile and
`cmd/...` (including the intake benchmark) pass.

Failing on main HEAD before this branch and unchanged by it:
`TestPrepareProgramIsIdempotentAndLinksPackageStages`,
`TestProgramQuestionAnswerAndNotSure` (store),
`TestProfilePlacesInputsInBuildOrder`, `TestProgramRoutes`,
`TestProgramQuestionRoute` (httpapi), `TestProgramLibraryInRepoLoads`
(knowledge). The in-progress program work fixes them.

Not run: the real-proposal false-green gate (`cmd/proposal-eval`); its
private answer key is not on disk. The letterhead rule fires only with one
ABN in the document and no addressee label near it.

## Not done

- LGA from the postcode: needs a postcode-to-council dataset (ABS
  correspondence); not downloaded without the owner's approval.
- Retaining-wall keys stay separate by design (see the plan).

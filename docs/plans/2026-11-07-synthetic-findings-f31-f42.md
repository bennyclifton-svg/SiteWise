# Fixes for the synthetic-corpus findings F31 to F42, and the T02, T13 and T17 recommendations

Date: 7 November 2026. Owner direction: "please fix all recommendations",
with answers to the four questions from run `20261106-t02-t13-t17`.
Findings: `testing/synthetic/FINDINGS.md` (F31 to F42 and the shape, scorer
and harness lists under that run). Evidence:
`testing/synthetic/runs/20261106-t02-t13-t17/REPORT.md`.

App fixes are lane A. Each is fixed only as far as its finding, test first.
Existing tests, answer keys and speed budgets must still pass. No new
dependency and no new Jev round trip: every reading change rides the
passage's existing fan-out. After the merge, the T02, T13 and T17
scenarios are re-run unchanged to measure each fix.

## Owner answers (recorded as rulings R11 to R14)

- **R11 (CDC for a Class 2 recladding):** more likely a DA. The ST17
  `cdc-pathway` variant is retired: its folder and run stay as a record,
  it leaves the T17 `variants` list and is not re-run.
- **R12 (majority settling):** yes. Free text and flags settle on a strong
  majority, like choices (F15).
- **R13 (architect brief and masterplan):** the brief is typical on every
  project, so it is always priced. A masterplan is only for large
  projects.
- **R14 (mixed-use scale):** apartment count and shop area are key metrics
  for shop-top mixed use, so the app must read them.

## Direction choices (decided before code)

**R12, majority by documents.** A choice, flag or free-text detail takes
its leading value when that value comes from at least 3 distinct
documents and at least 3 times as many documents as all other values
combined. Passages are counted per document (one document, one vote; this
also fixes F27). The other values stay visible as alternatives and the band
is amber. Anything closer stays a conflict, as today. Numbers and money
keep today's rule.

**R13, brief always, masterplan on large projects.** The architect's first
stage is the brief on every project. The masterplan lines (options,
report) are included only on a large project: more than one building or a
staged project (`flag.multi_building` or `flag.staged` yes). The stage
name follows what is included ("Brief" or "Brief and masterplan").

**R14, ask the mixed-use scale fields on their own words.** The subclass
scale fields that carry a project's key metric get `triggers:`: apartment
or unit counts ("apartments", "units", "dwellings") and shop or retail
area ("GLA", "retail area", "shop area", "lettable area"). The site-area
triggers also catch "site … m2" wording such as "consolidated site of
about 1,800 m2". Podium and tower levels stay unasked; their keys become
`read_by_design: false`.

## Work split

| Group | Findings | Area |
|---|---|---|
| Invoices | F31 | `internal/invoices` (`exish`, `printedAmounts`) |
| Fee proposals | F37, F38, F39 | `internal/proposals` (`settleFirm`, `settleTotal`, `totalWords`), `internal/store` `FirmLookup` |
| Profile reading | F32 (R14), F33, F34, F36 question | `knowledge/profile/*`, `knowledge/determinants.yaml`, `internal/knowledge/profile.go` trigger matching |
| Profile resolution | R12 (F35, F36 conflict, F27) | `internal/profile/reconcile.go` |
| Filing | F40 | `data/intake/disciplines.json` |
| Procurement and program knowledge | F41, F42 | `knowledge/works/package_defaults.yaml`, `knowledge/clusters/structure/interfaces.yaml` |
| RFPs | R13 | `knowledge/rfp/architecture.yaml`, `internal/rfp` if a line condition is needed |
| Testing (no app code) | shape and key gaps, scorer limits, harness, R11 | `testing/synthetic/` |

Profile reading owns the question-version bump; profile resolution
changes no question. Nobody edits `knowledge/profile/layout.yaml` or
`internal/httpapi/profile.go`, which hold another session's uncommitted
work.

## Testing changes

- **Shapes and keys:** every recommendation under "Shape and key gaps" in
  FINDINGS (T02 traditional `skeleton:` and reworded adds, T02 RFP
  `app_differs`, T13 architect `blocking: []`, the 24 T13 `app_differs`,
  T13 program rows `built_in: "no"` until F42 lands, T13 keys, ST17 d01
  also accepting `statutory_instrument`). After the app fixes, `built_in`
  claims that change are updated. T01 `cdc-pathway` and T07 heritage get
  `skeleton:` (the T10–T12 note).
- **Scorer:** a trigger may change `rfps[].expect` (positions and stages),
  and the design vehicle block (`slot:project:assessment_quantities`)
  counts as user input.
- **Harness:** `serve.sh fresh` recreates the `sitewise_synth` database so
  a run starts with an empty firm directory, and T13's Basalt Ridge firm is
  renamed so no fictional firm repeats across types with a different ABN.
- **R11:** ST17 `cdc-pathway` retired as above; R11 to R14 recorded in
  `reference/OWNER_RULINGS.md`.

## Budgets

Filing stays p50 ≤ 1 s and p90 ≤ 2 s; every other path keeps its
`bench/budgets.json` budget. The intake bench gate is run before and after
the merge.

## Done

Every group merged to local main with its tests; the Go suite passes apart
from the two failures that pre-date this work (TestMeridianAnswerKey,
TestProgramRoutes); the bench gate passes; the eight live T02, T13 and T17
scenarios are re-run into `runs/20261107-t02-t13-t17-after-fixes/` and
each fix is confirmed or reported.

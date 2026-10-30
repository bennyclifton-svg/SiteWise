# Scoring app output against a shape

`tools/score_shapes.py` compares what SiteWise built for a scenario with
that type's `types/<id>-<slug>/shape.yaml`. It covers the procurement table,
the cost plan, the program and the RFPs. It reads saved API responses and
never calls the app. Profile, register, fee and invoice fields are scored
separately against `expected/` (Task 7).

```
python testing/synthetic/tools/score_shapes.py \
  --shape testing/synthetic/types/T01-new-house/shape.yaml \
  --snapshot <run>/<scenario>/snapshot [--trigger cdc-pathway] [--json <run>/<scenario>/score.json]
```

The output is a JSON score (`sitewise-synthetic-score/1`) plus a short
summary. **There is no blended number.** Each area reports every category
on its own: matched, missed, extra, false positive, fabricated, violated,
new failure, known difference and fixed. A miss also carries the shape's
`built_in` claim:

- **`missed_built_in`** (the shape says `yes`): the app should have
  produced the item and did not. This is an app bug.
- **`missed_known_gap`** (`no` or `partial`): a knowledge gap the shape
  already names.

An area the snapshot does not hold is marked `not_captured`, with every
expected item missed. It is never passed by default.

## 1. Saved-output format (the harness writes this)

One folder per scenario run. Each file is the **unmodified JSON body** of
one API response. The harness must not reshape these files.

| File | Endpoint | Route |
|---|---|---|
| `packages.json` | `GET /projects/{id}/packages` | `internal/httpapi/projects.go:204` (`getPackages`, `internal/httpapi/packages.go:13`) |
| `cost_plan.json` | `GET /projects/{id}/cost-plan`, or the body of `POST /projects/{id}/cost-plan/prepare` (`{"plan":…,"totals":…}`, unwrapped by the scorer) | `projects.go:150`, `:159` |
| `program.json` | `GET /projects/{id}/program` (`404 {"empty":true}` when there is no program, scored as `empty`) | `projects.go:236` |
| `reports/<report id>.json` | `GET /reports/{r}` for each `kind: rfp` row of `GET /projects/{id}/reports` | `projects.go:193`, `:192` |

A single combined file `{"packages":…,"cost_plan":…,"program":…,"reports":[…]}`
is also accepted.

### Fields the scorer relies on

**Procurement:** `PackageOverview`, `internal/store/packages.go:20`.
- `items[]` (`procurement.Package`, `internal/procurement/packages.go:13`):
  `id`, `kind`, `works_scope`, `discipline_id`, `title`,
  `lifecycle_status` (`cancelled` is ignored) and `retired_at` (retired
  items are ignored).
- `suggestions[]` (`internal/procurement/defaults.go:10`): `title`, `kind`.
  A suggestion has no discipline id. Its key is
  `default:services:<title>` (`defaults.go:55`). The title is mapped to a
  discipline through the label and aliases in `data/intake/disciplines.json`.
- `proposed_works` (`internal/procurement/proposed.go:32`): counted as
  `trade.main_works` when it has no `package_id`.
- A works item with `works_scope: head_contract` and no `discipline_id`
  also counts as `trade.main_works`.

**Cost plan:** `costs.Plan`, `internal/costs/plan.go:22`.
- `items[]` (`Item` `:68`, which embeds `Content` `:48`): `cost_item_id`,
  `label`, `code`, `line_kind` (`fee`, `works` or `project_wide`),
  `category`, `package_id`, `parent_item_id`, `posting`, `source_key`
  (`works:<system>`, `pw:<starter id>` or `fee:<package id>`), `origin`,
  `rate` and `quantity`.
- `values{budget|commitment|claimed_to_date}` (`Value` `:80`): `amount`,
  `low`, `high`, `value_state`, `origin`, `rationale` and `provenance`.
- `suggestions[]` (`internal/costs/suggest.go:20`) have the same content
  plus `source_key` and `parent_key`. Pending suggestions count as app
  lines, tagged `state: suggestion`. Dismissed lines are ignored.
- The section comes from the line kind:
  - `fee` → `consultants`;
  - `works` → `construction`;
  - `project_wide` with `category: authority_fees` → `fees_and_charges`;
  - any other `project_wide` → `contingency`.
- A fee line's discipline is its package's `discipline_id`.

**Program:** `ProgramView` = `ProgramState` + `ProgramExtras`
(`internal/store/program_prepare.go:73`, `internal/store/program.go:127`).
- `activities[]` (`program.go:68`): `id`, `parent_id`, `kind` (`stage`,
  `activity`, `milestone` or `hold_point`), `label`, `catalogue_stage_id`
  (on stages), `fragment_id` and `interface_id`.
- `links[]` (`program.go:97`): `pred_id` and `succ_id`.
- `computed.activities` (`internal/program/model.go:105`): an object keyed
  by activity id with `es`/`ef` working-day indices and `unknown`
  (`Times.MarshalJSON`, `model.go:161`; all null when unknown).
- `computed.cycle`.
- The skeleton is `skeleton.id` (`program_prepare.go:32`), else
  `program.skeleton_id`.

**RFPs:** `ReportView`, `internal/store/reports.go:48`.
- `report.kind` and `report.package_id`. The package id gives the
  discipline through `packages.json`.
- `rfp.discipline` is the RFP template id, for example `structural`
  (`internal/rfp/resolve.go:263`).
- `rfp.groups[]` (`internal/rfp/controls.go:11`):
  - `id: stages` holds the `stage:<key>` controls (`:172`), with `value`
    one of `include`, `option`, `exclude`, `done` or `ask` (`resolve.go:17`);
  - `id: elements` holds the `element:<key>` controls (`:179`);
  - `id: positions` holds the `position:<key>` controls (`:190`), with
    `value` `I`, `O`, `X` or `ask`. A missing control means `absent`.
- `rfp.checks[]` (`resolve.go:111`): `id` and `blocking`.
- `draft.sections[].blocks[]` (`internal/reports/draft.go:10`, `:33`,
  `:39`): `label`, `text` and `table.rows`. Sections with `internal: true`
  are review lists, never issued, and are skipped.

## 2. Expectations for a variant

`--trigger <id>` scores the base shape plus that trigger's `adds`, minus its
`removes`.

- **Removed items become not expected.** A town planner still suggested
  under `cdc-pathway` counts as a false positive.
- **Added items are expected.** They take the trigger's `built_in` claim
  for that area.
- **Links follow the removals.** An `after` link to a removed activity is
  dropped.
- **RFP additions merge.** Added `must_include` phrases join the existing
  RFP for that discipline, or create one.

The flag can be repeated for multi-trigger scenarios. An unknown trigger id
is an error.

## 3. Matching

### Normalisation

- Labels are lower-cased.
- A leading line code is removed (`F04 `, `07 `, `13.1 `).
- Bracketed asides are removed: "Town planner (optional)" becomes
  "town planner".
- `&` becomes "and", and `_` becomes a space.
- Everything that is not a letter or a digit becomes a space.

For token comparisons, stop words are dropped and a plural `s` is stripped.
Matching is one-to-one: each app line serves at most one expected line.

### Rules, in order

Each rule runs over all expected items before the next rule starts.

**Cost plan:**
1. `app_label` or `aliases` on the shape line equals the app's
   "code label" or label.
2. The normalised labels are equal.
3. The source anchor matches:
   - `starter_lines.yaml <id>` matches `source_key: pw:<id>`;
   - `elements.yaml code NN` matches a posting `works` line with code `NN`.
4. Discipline, for consultants only. The longest discipline label or alias
   inside the shape label (so "Quantity surveyor" beats "surveyor") equals
   the fee line's package discipline.
5. The app label starts with the shape label, within the same section
   (for example `Exclusions: …`).

A match by rule 1, 2 or 3 in a different section is kept and listed under
`section_mismatch`. Unmatched lines are then matched against
`not_expected_cost_plan` with the same rules, as false positives. The rest
are `extra`, except non-posting group lines, which are counted under
`groups` because their children are scored.

**Procurement:** match on the exact discipline id first. Next, match on
an id in the entry's `aliases` (other discipline ids that also count, for
example `consultant.esd` with `aliases: [consultant.basix]`; see
`reference/CONVENTIONS.md` C2). `matched_by` says which rule matched. A kind
that differs is listed under `kind_mismatch`. Unmatched app packages are matched against
`not_expected_procurement` (false positives), with the same two rules. The
rest are `extra`.

**Program activities** (stage rows are not activities):
1. Equal normalised label.
2. One of the shape's `aliases` equals the app label (`matched_by: alias`).
3. Interface: the shape `source` anchor `if.<id>` equals `interface_id`.
4. Fragment: equal `fragment_id`, picking the closest label. This matters
   because the app gives a single-activity fragment the work item's title
   (`internal/program/generate.go:649`).
5. Similar label: token Jaccard of at least 0.75.

## 4. Rules by area

### Cost plan

- **Matched, missed, extra and false positive**, as in §3.
- **Fabrication.** Every amount, low or high value on every line and metric
  is classified with `classify_amount`:
  - **document**:
    - a budget spread whose provenance has `budget_origin: document`
      (`internal/store/cost_budget.go:156`);
    - a brief figure whose provenance `origin` is `document` (`:262`);
    - an award commitment whose rationale starts "Awarded to"
      (`internal/store/awards.go:484`), which names the firm's file.
  - **user**: a value with `origin: user` that is not an award. This
    includes typed values, quantity × rate and an acknowledged benchmark.
    It is reported, and should be zero in a harness run, because the
    harness never types.
  - **fabricated**: any other figure. That includes an `assumption` with no
    spread or brief provenance, or a spread or brief resting on a
    `calculation` or `assumption` row. A `rate` or `quantity` on a line
    whose `origin` is not `user` is also fabricated.

  The rule is that **unknown stays unknown**: any fabricated amount is a
  failure.

### Procurement

The table is the union of three sources: created packages (`state:
package`), live suggestions (`suggested`) and an unplaced head-contract
proposal (`proposed_works`). The shape's `need` is copied onto each miss.

### Program

- **Skeleton:** `match` is true when the app's skeleton id equals
  `program.skeleton`. A shape value of `missing` means no skeleton is
  expected.
- **Activities:**
  - matched, missed, extra and false positive (activities a trigger
    removes);
  - `wrong_stage`, when the matched row's parent stage
    (`catalogue_stage_id`) differs from the shape's `stage`.
- **`after` links:** each link with both ends matched is checked against
  the computed dates. It **holds** when the activity's `es` is no earlier
  than the predecessor's `ef`, which is a finish-to-start reading of the
  app's own dates, whatever the app's link type.
  - A side with unknown times gives **unknown**, never a pass.
  - A missed side gives **unscored**, with the reason.
- **Orphans:** a non-stage row that no link touches. The app chains
  unlinked rows inside a stage (`generate.go:976`), so an orphan is a row
  that only floats from its stage start.
- **Cycles:** every loop found by a depth-first search over `links[]`, plus
  `computed.cycle` when the app reports one.

### RFPs

- **Finding the RFP.** Each rule runs over every expected RFP before the
  next rule starts:
  1. the report's package discipline equals the entry's `discipline`;
  2. the package discipline is one of the entry's `aliases`;
  3. the template ids match.

  Discipline comes first because two disciplines can share a template:
  `consultant.bca` and `consultant.certifier` both use `bca-certification`
  (`reference/CONVENTIONS.md` C3). An expected RFP is `missing` otherwise (always so for
  `template: null`). An RFP for a removed discipline is `not_expected`. Any
  other RFP is `extra`.
- **`expect`** uses the same keys and meanings as the Go answer-key test
  (`internal/rfp/answerkey_test.go:40`):
  - `stages_absent` and `elements_absent` mean state `exclude`;
  - `stages_ask` means `ask`;
  - a position `absent` means no control.

  Each key is classified:
  - **ok**: the app equals `expect`;
  - **known_difference**: the app differs, but exactly as `app_differs`
    predicts;
  - **fixed**: `app_differs` predicted a difference, and the app now equals
    `expect`. That is good news, and the shape should be updated;
  - **new_failure**: anything else. That includes a stage or element key
    the app does not have, which signals template drift, and every
    `blocking` id that is missing or unexpected.

  `text.*` keys in `app_differs` cannot be checked mechanically, and are
  listed under `not_checked`.
- **`must_include` and `must_not_include`.** Each item is a plain string or
  `{text, key}`. The score records where each one was found.
  1. **A keyed item** (`key: stage.<k>`, `position.<k>` or `element.<k>`) is
     decided by that control's state alone (`where: key:<key>`). An exact
     key beats word overlap. A keyed row that the app does not have counts
     as absent.

  Unkeyed phrases are prose, so they are matched approximately:
  2. A leading `Option:` is dropped.
  3. If at least 75 % of the phrase's words are in a stage, element or
     position control label, the control's state decides:
     - `include`, `option`, `I` or `O` means present;
     - `exclude`, `done` or `X` means absent;
     - `ask` means unknown.

     This keeps a printed "Excluded" row from counting as the item being in
     scope.
  4. Otherwise the phrase is present if 75 % of its words are in one issued
     text block or table row. Rows marked X, Excluded or Not in this RFP
     are skipped.

  A failure on a control that `app_differs` predicts is reported as
  `known`. Unknown results (an `ask` control, or no draft) are listed
  apart; they are neither a pass nor a failure. The optionality an
  `Option:` prefix asks for is checked only when the phrase lands on a
  control.

## 5. What today's API cannot score

- **`authority`** under trigger `adds`. No endpoint lists authority
  applications.
- **`text.*` differences.** Prose deliverables need an owner's read.
- **Which document a figure came from.** Spread and brief provenance say a
  figure came from *a* document-origin profile row, not which document.
  Award rationales name the file.
- **Optional status of an unkeyed `must_include`** (see §4). A keyed item is decided by its row's state.
- **`work_items`, `profile`, `expected_derived` and `scenario_inputs`.**
  These belong to the field comparisons (`GET /projects/{id}/works`,
  `/profile`), not to this scorer.

## 6. Owner rubric: practical, logical, concise

The owner records this per type in `scenarios/<scenario>/expected/rubric.md`
after reading the app's cost plan, program and RFP set for the base
scenario. It does not feed the machine score. Score each of the three
outputs on each criterion (1–5), with one sentence of reason.

| Score | Practical (a PM could use it tomorrow) | Logical (order and dependencies make sense) | Concise (nothing a PM would strike out) |
|---|---|---|---|
| 5 | Usable as issued; nothing to add for this job | Every sequence and stage is how a competent PM would run it | No line, activity or clause to delete |
| 4 | One or two small additions before use | One minor sequencing or placement fix | One or two items to delete |
| 3 | Usable as a starting draft; several additions | Several fixes, but the critical path is right | Noticeable padding (generic or other-type items) |
| 2 | Misses things a PM would see at once (a consultant, a statutory step) | A critical-path error (an approval after work it gates) | So much noise the real items are hard to find |
| 1 | Not usable for this job | The logic is wrong or circular | Mostly irrelevant content |

`rubric.md` form:

```
# Rubric: <scenario id>  (owner, <date>)
| Output | Practical | Logical | Concise | Reason |
|---|---|---|---|---|
| Cost plan | | | | |
| Program | | | | |
| RFPs | | | | |
```

Agents never fill in rubric scores.

## 7. Decisions (provisional until the owner confirms)

Taken by the orchestrator on 30 October 2026.

1. **The harness may press `POST /projects/{id}/cost-plan/prepare`, `POST
   /projects/{id}/program/prepare` and `POST /projects/{id}/reports`** (one
   RFP per services package). It does so in its own synthetic test
   organisation only. These are the user's clicks that bring the cost plan,
   program and RFPs into being, and prepare also adopts suggested
   consultants as packages. Without them, those areas score `not_captured`
   or `empty`.
2. **A `must_include` or `must_not_include` item may be `{text, key}`.** The
   key is decided first (§4). Plain strings keep the 75 % word match.
3. **`aliases: [...]` is allowed on cost-plan lines and program
   activities.** Both forms are optional. `validate_shape.py` checks them,
   and brief 01 documents them.
4. **Discipline `aliases: [<discipline id>, ...]` are allowed on
   procurement, `not_expected_procurement` and rfps entries**
   (`reference/CONVENTIONS.md` C2). `validate_shape.py` checks that each one
   is a discipline id.
5. **T01 architecture `expect.blocking` is now `[]`.** The scenario states
   the budget, so nothing should block.

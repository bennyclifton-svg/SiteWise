# ASMM and NATSPEC scope views: implementation plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.
> Use superpowers:test-driven-development for every Go task. Read every section
> before Task 1. Stop at each **STOP** marker and do what it says. Do not skip one.

**Goal:** On Profile → Scope and works, a user picks **View scope by: Systems | ASMM | NATSPEC**
and sees the same scope grouped and numbered under the chosen classification.
Jev proposes both classifications for every building system once, offline. The
owner reviews them. A user can change either classification on a project.

**Architecture:** Classification is a **view over the building systems**,
the same way consultant disciplines are (AGENTS.md rule 4). Nothing about the
scope record changes. Every scope item is already a `work_items` row with a
stable id and a `system_id`, and every system already has a `scope.<leaf>`
profile row. Cost lines and package scope already point at `work_item_id`.
So we add four things:

1. Two catalogue files: the ASMM codes and the NATSPEC worksections, with
   edition ids.
2. One crosswalk file that maps every live system to codes in both catalogues.
   An offline tool fills it with one Jev fan-out per system. The owner reviews it.
3. A small per-project override table for user edits.
4. A per-user view preference and a toggle in the Profile UI.

The toggle never calls Jev and never changes a fact. Switching views only
regroups rows that are already loaded.

**Tech stack:** Go 1.27, PostgreSQL 17 (pgx, sqlc), React 19 + Vite, Playwright,
Python knowledge checker, Jev `jev-1.13.0` through `internal/jev`. No new
dependency.

---

## 0. Read this first

### 0.1 Where this plan departs from the pasted spec, and why

The owner's spec (pasted on 2026-10-12) is the requirement. This plan meets it
with these changes. If you find yourself drifting back towards the spec's literal
wording on any of these points, stop.

| Spec says | This plan does | Why |
|---|---|---|
| "Refactor the scope data model: one shared scope record" | No refactor. `work_items` + `scope.<leaf>` already are that record, with stable ids that cost lines and packages reference. | A refactor here would put filing, costs and packages at risk (lane A) and gain nothing. |
| "Jev analyses each scope item once" per project | Jev classifies each **system** (~150 of them) **once, offline**, into `knowledge/classifications/crosswalk.yaml`. A project looks the codes up. | Every scope item carries a `system_id` from a fixed catalogue, so the answer does not depend on the project. AGENTS.md rule 3 says "nothing judged while a user waits if it can be precomputed". Lookups cost 0 ms and 0 Jev calls. Answers are the same on every project, and the owner can review them once. |
| Each assignment stores source `Jev` or `User` | Jev assignments live in the crosswalk (`source: jev` or `owner`). User assignments live in the per-project override table (`source: user`). The API merges them and reports `source` for each one. | If Jev rows were copied into every project, they would go stale whenever the crosswalk improved. |
| "On reclassification, preserve user edits" | A user override always wins over the crosswalk. Re-running the crosswalk tool never touches the overrides table. | This preserves user edits by construction. |
| "Shared scope edits flag classifications for review" | An override is flagged **review** when the catalogue edition it was made against differs from the current one. The API computes this flag; it is not stored. | System descriptions are catalogue facts, not project facts. The only change that can invalidate a user's code is a new catalogue edition. |
| Downstream PMP/RFP/RFT/cost/programme integration | Phase 5 delivers the API contract and a design note. Each downstream screen gets its own later plan. | AGENTS.md says a direction choice (what the next screen should be) is decided in prose before code. |
| Two-way toggle ASMM / NATSPEC | Three-way: **Systems** / ASMM / NATSPEC, with Systems as the default (see O3). | Presets, the Requirements link and the compliance inputs are anchored to building systems. Systems is the model, and the other two are views over it. |

### 0.2 Non-negotiables (from AGENTS.md; break one and the work is rejected)

- **Jev is the only AI.** No LLM, embedding or other model, at runtime or in tools.
- **Do not type classification codes from memory.** Every code and label comes from
  the source documents the owner provides (Task 0). A code you "know" is
  still a guess. Leave unverified entries as `verified: false`.
- Every database row is scoped by `org_id`. Every query has `org_id = $1`.
- Speed budgets (Section 0.5) are gates.
- Run `python tools/check_knowledge.py --strict` after every edit under `knowledge/`.
- Write `status: draft` on everything. Only the owner writes `reviewed`.
- Python edits on this Windows machine turn LF files into CRLF. Open files with
  `newline=''`, or use the Edit tool.

### 0.3 Environment

- Work in a git worktree: `git worktree add ../sitewise-classify -b scope/classification`.
  Other sessions share the main tree.
- Go and sqlc are in `D:\AI Projects\sitewise\.tools`. See `tools/check.ps1` for
  PATH, GOCACHE and the test database URL. Run Go tests with `-p 1`.
- Test database: `SITEWISE_TEST_DATABASE_URL=postgres://sitewise@127.0.0.1:5433/sitewise_test?sslmode=disable`.
  If another session is using 5433, use a private one as the 2026-10-03 plans did.
- Full gate: `pwsh tools/check.ps1`. To build the SPA, run `npm --prefix web run build`
  through bash, because `dev.ps1 -Build` is broken in pwsh.
- **This repository is public.** Never commit private corpus text, and never commit a
  licensed source document (see O4).

### 0.4 Glossary for this plan

- **System**: a node in `knowledge/systems.yaml` (13 top-level) or
  `knowledge/clusters/*/systems.yaml` (leaves such as `envelope.windows-doors`). Load
  them with `knowledge.Load(...)`, then `cat.System(id)`, `cat.TopSystems()` and `cat.Leaves()`.
- **Scheme**: `asmm` or `natspec`.
- **Catalogue**: the list of codes for one scheme and edition.
- **Crosswalk**: system → codes, for both schemes.
- **Primary code**: where a system's row is listed in that view. **Additional codes** are
  shown as chips on the row and never move it.
- **Override**: a user's per-project replacement of one system's codes for one scheme.
- **Unresolved**: no code passed the checks. The row stays visible under "Unclassified".

### 0.5 Budgets (gates)

| Path | p50 / p90 | How it is measured |
|---|---|---|
| `GET /projects/{id}/profile` | unchanged from today (`pathProfileRead`) | existing latency gate; classification adds map lookups only |
| `PUT /projects/{id}/profile/classification` | 50 / 150 ms | timed as `pathProfileEdit`, same as `profile_edit` in `internal/latency/gate_test.go` |
| `PUT /me/preferences` | 50 / 150 ms | timed as `pathProfileEdit` |
| Toggle switch in the browser | < 100 ms, no network wait | client regroups from data already loaded; the preference save runs in the background |
| Jev on any user click | **zero calls** | Jev runs only in the offline tool |

### 0.6 Owner decisions (STOP: get answers before Task 1)

Ask the owner these questions, record the answers in Section 7 of this file, and commit.
The recommended answers are in bold. If the owner says "go with your
recommendations", use the bold ones.

- **O1. Which document is "ASMM"?** The *Australian Standard Method of Measurement*
  (ASMM 5th ed., now ANZSMM 2018) is a **trade/work-section measurement**
  standard. The present 01–13 rail and `knowledge/costs/elements.yaml` are closer to the
  AIQS **elemental** classification (Substructure, Columns, Upper floors, External walls, …,
  published in the *Australian Cost Management Manual*). The pasted spec warns that earlier
  tables "mixed classification structures"; this is that mix. Options: (a) ANZSMM 2018
  trade sections; (b) **AIQS elemental (ACMM) codes, labelled "ASMM / AIQS elemental"**
  if the owner means what QS cost plans use; (c) both, as three schemes. The code below
  uses a scheme id (`asmm`) and a display name read from the catalogue file, so the answer
  changes only data and labels.
- **O2. NATSPEC edition.** Which NATSPEC release (for example "NATSPEC 2026-04")? The
  owner supplies the worksection list for that release.
- **O3. Views.** **Systems | ASMM | NATSPEC, defaulting to Systems**, or exactly two
  views. If two, the Systems view is removed from the toggle only. Its code stays,
  because Implications and Costs use it.
- **O4. Licensing (public repo).** Codes with short titles are reference data. Scope
  notes copied from NATSPEC or AIQS are licensed text. **Commit codes and titles only.
  Write `usage` notes in our own words. Keep source PDFs in `data/reference/classifications/`,
  which is git-ignored.** The owner confirms that committing codes and titles is
  acceptable.
- **O5. Where a multi-mapped system is listed.** **Under its primary code only, with
  the additional codes as chips** (no duplicate rows, so nothing is counted twice), or under every code
  with a "also under" marker. The plan builds the first option.

### 0.7 Task 0 (owner, STOP): provide the catalogue sources

The agent cannot do this task. The owner puts these files in `data/reference/classifications/`
(create it, and add `data/reference/classifications/` to `.gitignore` before anything
is copied there):

- the NATSPEC worksection list for the O2 edition (PDF or spreadsheet), and
- the ASMM/AIQS code list for the O1 choice.

Until the owner does this, you may do Tasks 1–3 (formats, checker and loader, with
test fixtures). You may not do Task 4 (filling real catalogues) or anything after it.

---

## Phase 1: Catalogues

### Task 1: Document the catalogue format in the schema

**Files:** modify `knowledge/SCHEMA.md` (append a section at the end).

**Step 1:** Append this section, word for word:

````markdown
## Classifications (2026-10-12)

See `docs/plans/2026-10-12-asmm-natspec-scope-views.md`. ASMM and NATSPEC are
views over the systems, like disciplines. They never replace the systems.

**Catalogues** `knowledge/classifications/<scheme>.yaml`, one per scheme
(`asmm`, `natspec`):

```yaml
version: 1
scheme: natspec                 # file name without .yaml
name: NATSPEC                   # shown on screen
edition: "NATSPEC 2026-04"      # exact release id from the source; changes flag user overrides for review
status: draft
sources:
  - {reference: natspec-worksections-2026-04.pdf, anchor: "Worksection list"}   # file in data/reference/classifications/ (git-ignored)
groups:                         # top level of the hierarchy, in the source's order
  - {code: "04", label: Enclosure}
codes:
  - code: "0451"
    label: Windows and glazed doors      # title exactly as the source prints it
    group: "04"
    usage: >-                            # our own words, never copied text; read by Jev
      Specifying windows, glazed doors and their glazing as products installed in walls.
    verified: false                      # true only once the owner checks code and title against the source
```

Rules: `code` unique in the file and matches the scheme pattern (`natspec`:
`^\d{4}$`, `asmm`: set in the checker after O1); every `group` exists; `label`
non-empty; `usage` 1–300 characters; `edition` non-empty.

**Crosswalk** `knowledge/classifications/crosswalk.yaml`: one entry for every
live system, top-level and leaf:

```yaml
version: 1
status: draft
editions: {asmm: "<edition>", natspec: "NATSPEC 2026-04"}   # must equal the catalogues' editions
entries:
  - system: envelope.windows-doors
    natspec: {primary: "0451", also: ["0455"], source: jev, p: {"0451": 0.97, "0455": 0.81}}
    asmm:    {primary: null, also: [], source: jev}           # null primary = unresolved
```

`source` is `jev` (written by `cmd/classify-crosswalk`) or `owner` (hand
edited; the tool never overwrites an `owner` entry). `primary` is a catalogue
code or null; `also` holds codes, without `primary`. `p` is optional, written by the
tool, and is the Jev noul probability for each code. Deprecated systems are not listed.
````

**Step 2:** Commit: `git commit -m "Schema: classification catalogues and crosswalk (lane A)"`.

### Task 2: Checker support

**Files:** modify `tools/check_knowledge.py`; modify `tools/test_check_knowledge.py`.

**Step 1: Write failing tests** in `tools/test_check_knowledge.py`, following the style of
the tests already there (read the file first). Cover each case:
duplicate code → error; group not in `groups` → error; NATSPEC code `451` (3 digits) →
error; missing `edition` → error; crosswalk entry for an unknown system → error;
crosswalk missing a live system → error; `primary` not in the catalogue → error;
`primary` repeated in `also` → error; crosswalk `editions` not equal to the catalogue `edition`
→ error; a valid pair of fixture catalogues and crosswalk → no error.

**Step 2:** Run `python -m pytest tools/test_check_knowledge.py -q -p no:cacheprovider`
and expect the new tests to FAIL.

**Step 3: Implement** `check_classifications(report, known)` in `check_knowledge.py`,
modelled on `check_cost_structure` (line ~1345). Use the existing `known` dict
of system ids and the deprecated map. Call it from `main()` after the other catalogue checks.
**If `knowledge/classifications/` does not exist, do nothing.** The check must pass
before Task 4.

**Step 4:** Run the tests and `python tools/check_knowledge.py --strict`. Both must PASS.

**Step 5:** Commit: `Knowledge checker validates classification catalogues (lane A)`.

### Task 3: Go loader

**Files:** create `internal/knowledge/classifications.go`,
`internal/knowledge/classifications_test.go`; modify the catalogue loader in
`internal/knowledge/load.go` to call `loadClassifications(root)` (find where
`loadCostElements` is called and add the call next to it).

**Step 1: Write failing tests** with `t.TempDir()` fixture files, copying how
`internal/knowledge/cost_elements_test.go` builds a temporary knowledge root:

- `TestClassificationsLoad`: valid catalogues + crosswalk; `cat.Scheme("natspec").Edition`
  is the fixture edition; `cat.Classes("envelope.windows-doors", "natspec")` returns
  primary `0451`, also `[0455]`, source `jev`.
- `TestClassificationsRejectUnknownCode`: a crosswalk primary that is not in the catalogue fails `Load`.
- `TestClassificationsRejectMissingSystem`: a live system with no entry fails `Load`.
- `TestClassificationsAbsent`: with no `classifications/` directory, `Load` succeeds and
  `cat.Schemes()` is empty. Today's repository must keep loading.

**Step 2:** Run `go test ./internal/knowledge -run Classif -p 1` and expect FAIL.

**Step 3: Implement.** Keep it small:

```go
package knowledge

// Classifications are views over the systems (ASMM, NATSPEC). Codes come
// from the owner's source documents; Jev's crosswalk is drafted offline by
// cmd/classify-crosswalk and reviewed by the owner.

type ClassCode struct {
	Code     string `yaml:"code" json:"code"`
	Label    string `yaml:"label" json:"label"`
	Group    string `yaml:"group" json:"group"`
	Usage    string `yaml:"usage" json:"-"`
	Verified bool   `yaml:"verified" json:"verified"`
}

type ClassScheme struct {
	ID      string      `json:"id"`
	Name    string      `json:"name"`
	Edition string      `json:"edition"`
	Groups  []ClassCode `json:"groups"`
	Codes   []ClassCode `json:"codes"`
	byCode  map[string]ClassCode
}

// SystemClasses is one system's codes in one scheme. Primary "" is unresolved.
type SystemClasses struct {
	Primary string             `yaml:"primary" json:"primary"`
	Also    []string           `yaml:"also" json:"also"`
	Source  string             `yaml:"source" json:"source"` // jev | owner (catalogue); user (project override)
	P       map[string]float64 `yaml:"p" json:"-"`
}

func (c *Catalog) Schemes() []ClassScheme                  { /* in fixed order asmm, natspec, present only */ }
func (c *Catalog) Scheme(id string) (ClassScheme, bool)    { /* lookup */ }
func (s ClassScheme) Code(code string) (ClassCode, bool)   { /* byCode lookup */ }
func (c *Catalog) Classes(system, scheme string) SystemClasses { /* crosswalk lookup; zero value if absent */ }
```

Validation in `loadClassifications` repeats the checker's rules: the checker guards
authors, and the loader guards the binary. Use `c.unmarshal` (strict YAML,
as `loadCostElements` does). Return `fmt.Errorf("invalid classification %s: ...")`.

**Step 4:** Run the tests and expect PASS. Then run `go test ./internal/knowledge -p 1`; the whole package must PASS.

**Step 5:** Commit: `Load classification catalogues and crosswalk (lane A)`.

### Task 4: Fill the catalogues from the owner's sources (needs Task 0)

**Files:** create `knowledge/classifications/natspec.yaml`, `knowledge/classifications/asmm.yaml`.

**Step 1:** Read the source documents in `data/reference/classifications/`. Transcribe
**every** code with its title and group exactly as printed. Do not shorten or
"tidy" a title.
**Step 2:** For each code, write `usage` in your own words: one sentence on what work
a specifier or QS puts under it, plus what it excludes if the source makes that
clear. Jev reads this, so make it literal and concrete (see Task 6).
**Step 3:** Set `verified: false` everywhere and `status: draft`.
**Step 4:** Write a small script in your scratchpad (not the repo) that counts
codes per group in your YAML. Compare the counts by eye with the source's table of
contents and write the counts in the commit message.
**Step 5:** Run `python tools/check_knowledge.py --strict` and expect PASS. The crosswalk
does not exist yet, so the crosswalk checks do not run (make sure the checker allows
catalogues without a crosswalk: it errors only if `crosswalk.yaml` exists and is wrong).
**Step 6:** Commit: `Classification catalogues transcribed from <editions> (lane A, unverified)`.

**STOP.** Ask the owner to spot-check 20 random codes against the source before
Phase 2. The owner, not you, sets `verified: true`.

---

## Phase 2: The crosswalk, drafted by Jev

### How Jev is used here, and why it is used this way

The job: for each system, which codes in each scheme cover its work? That
calls for judgement over unstructured descriptions (`describes`, `excludes`, `usage`),
which is what Jev is for. Code handles the rest. Each line below is a TypeSafe doc
you must follow:

- **One fan-out per system** with every question about that system in one request:
  both schemes, both the primary pick and the per-code checks
  ([fan-out](https://docs.typesafe.ai/patterns/fan-out)). One call answers both
  classification sets, as the spec asks.
- **Code shortlists, Jev picks**
  ([pre-parsed extraction](https://docs.typesafe.ai/cookbooks/pre_parsed_value_extraction_cookbook),
  [intent routing](https://docs.typesafe.ai/patterns/intent-routing)). A hand-written
  `candidates.yaml` maps each **top-level** system to the catalogue groups its leaves
  may fall under. Code expands those groups into candidate codes. Make the shortlist
  generous: a missed group means Jev can never pick the right code.
- **A choice question for the primary code**, with the shortlisted codes plus `none`.
  The [API](https://docs.typesafe.ai/api) allows at most 255 options
  (`jev.MaxChoiceOptions`). If a shortlist is longer, the tool fails for that system, and
  you narrow `candidates.yaml`.
- **A noul for each shortlisted code** ("is this code also used for this work?") gives
  the additional codes. Nouls carry probability only
  ([confidence](https://docs.typesafe.ai/confidence)).
- **A threshold per question type, chosen by the cost of a wrong answer, set in code,
  never in the question**
  ([confidence routing](https://docs.typesafe.ai/patterns/confidence-routing)).
  Choice confidence depends on the number of options, so the primary threshold cannot
  be reused for the nouls, and it is calibrated on the gold set (Task 7).
- **Literal and atomic wording that names state fields in backticks, with no
  counting and no generation**
  ([jev-1.13 limitations](https://docs.typesafe.ai/model-jaggedness/jev-1.13)).
  Jev returns ids. Code checks every id against the catalogue and writes the YAML.
- **Code checks consistency.** If the primary pick is not also a noul pass, or a noul
  passes for a code in a different group from the primary, mark the system unresolved
  for review. Do not ask Jev a second time; the [how to build](https://docs.typesafe.ai/concepts/how-to-build-with-system-one)
  guide puts this control flow in code.

### Task 5: Candidate shortlist file

**Files:** create `knowledge/classifications/candidates.yaml`; extend the checker
(the group exists in that scheme; every top-level system is listed) and the Go loader.

```yaml
version: 1
status: draft
# Catalogue groups a top-level system's work may fall under. Code expands
# these into the codes Jev chooses among. Generous on purpose: a missing group
# means Jev can never pick the right code.
systems:
  - system: envelope
    natspec: ["04", "05"]     # example only: use real group codes from Task 4
    asmm: ["..."]
```

Use TDD as in Task 2 and Task 3. Commit: `Classification candidate groups per system (lane A)`.

### Task 6: `cmd/classify-crosswalk`

**Files:** create `cmd/classify-crosswalk/main.go`, `cmd/classify-crosswalk/questions.go`,
`cmd/classify-crosswalk/questions_test.go`, `cmd/classify-crosswalk/merge.go`,
`cmd/classify-crosswalk/merge_test.go`.

**What it does:**

```
go run ./cmd/classify-crosswalk -knowledge knowledge -out knowledge/classifications/crosswalk.yaml \
   [-only envelope.windows-doors,...] [-record tmp/crosswalk-rec.jsonl | -replay tmp/crosswalk-rec.jsonl] \
   [-gold data/eval/classification/gold.json]
```

1. Load the catalogue (`knowledge.Load`).
2. For each live system (top-level and leaf, skipping deprecated ones), build one
   `jev.Call` with `Priority: jev.PriorityBackground`, a fixed `QuestionVersion`
   (`"classify-1"`), and `Deadline: 10 * time.Second`.
3. Run up to 8 systems concurrently. The client's background slots cap this as well.
4. Check the answers in code (below) and merge them into the existing crosswalk.
   **Never change an entry whose scheme block has `source: owner`.**
5. Write the YAML in a stable order (system ids in catalogue order, codes sorted),
   with LF line endings.
6. Print a summary: systems run, resolved per scheme, unresolved, and tokens used.
7. With `-gold`, also print primary accuracy and additional-code precision/recall
   for the gold systems, and exit non-zero on a regression against `-baseline` if
   one is given.

For the API key and client options, copy `cmd/corpus-eval/main.go` around line 266
(`jev.New(jev.Options{APIKey: key, Model: config.PinnedJevModel, ...})`). For
recording and replay, use `eval.NewRecorder` the way corpus-eval does, so that tests and
re-runs need no network.

**State** (only what the questions need; do not include unrelated systems):

```json
{
  "system": {"id": "envelope.windows-doors", "label": "...", "describes": "...", "excludes": "...",
             "part_of": "Building envelope"},
  "natspec": [{"code": "0451", "title": "Windows and glazed doors", "usage": "..."}],
  "asmm":    [{"code": "...",  "title": "...", "usage": "..."}]
}
```

**Questions** (`questions.go`, a pure function `Questions(sys, natspec, asmm []knowledge.ClassCode) map[string]jev.Question`):

- `natspec.primary`: `choice`.
  Instructions: ``"Using `system` and the worksections in `natspec`, which one worksection would a specification writer use for most of the work described in `system.describes`? Work in `system.excludes` belongs elsewhere. Choose none if no listed worksection is used for that work."``
  Criteria: `{"<code>": "<code> <title>: <usage>", ..., "none": "No listed worksection is used for most of the work in `system.describes`."}`
- `natspec.uses.<code>`: `noul`, one per shortlisted code.
  Instructions: ``"Using `system` and worksection <code> <title> in `natspec`, would a specification for the work in `system.describes` include clauses from worksection <code>?"``
  Criteria: `{"true": "Some of the work in `system.describes` is specified under <code> (<usage>).", "false": "None of the work in `system.describes` is specified under <code>, or only work listed in `system.excludes` is."}`
- `asmm.primary` and `asmm.uses.<code>`: the same pattern, worded for the O1 choice
  ("which one <ASMM name> code would a quantity surveyor use to measure / cost most of
  the work in `system.describes`?").

Check the wording against the authoring rules in `knowledge/SCHEMA.md` → "Authoring rules": atomic,
literal, positive, criteria agreeing with instructions, no thresholds.

**Validation and merge** (`merge.go`, pure, unit-tested):

```go
// Thresholds are provisional until Task 7 calibrates them on the gold set.
// Noul answers carry probability only; choice answers carry confidence that
// depends on option count, so the two are never compared.
type Thresholds struct{ Primary, Also float64 }

func Resolve(scheme knowledge.ClassScheme, answers map[string]jev.Answer, prefix string, t Thresholds) knowledge.SystemClasses
```

Rules in `Resolve`:
1. Primary = `answers[prefix+".primary"].Choice` if it is a catalogue code, its confidence
   is ≥ `t.Primary`, and `answers[prefix+".uses."+code].Noul` ≥ `t.Also`. Otherwise
   primary is `""` (unresolved).
2. Also = every code whose `uses` noul is ≥ `t.Also`, excluding primary, sorted.
3. If primary is `""`, keep `also` empty as well. An unresolved row shows as unresolved,
   not half-classified.
4. Ignore an answer for an unknown code (the client already drops invalid choices; check anyway).
5. A missing answer (timeout or unresolved in `jev.Result.Unresolved`) makes the system unresolved for that scheme.
6. `P` records the noul probability of every code in the result.

**Tests** (`merge_test.go`, `questions_test.go`): primary below threshold → unresolved;
primary `none` → unresolved; primary whose own noul fails → unresolved; an unknown
code is ignored; `also` excludes primary and is sorted; an `owner` entry survives a merge
unchanged; every option id in the choice criteria is a catalogue code or `none`; a
shortlist above 255 returns an error; question keys are stable and sorted (so recordings
replay).

Commit: `Offline crosswalk drafting with one Jev fan-out per system (lane A)`.

### Task 7: Gold set, calibration and the live run

**Files:** create `data/eval/classification/gold.json` (public: it holds only
system ids and codes, and no private text); update `crosswalk.yaml`; create
`docs/evidence/2026-10-xx-crosswalk-run.md`.

**Step 1 (owner, STOP):** Ask the owner for primary + additional codes for 25 systems
spread across all 13 top-level systems: `{"system": "...", "natspec": {"primary": "...", "also": []}, "asmm": {...}}`.
Do not write these answers yourself.

**Step 2:** Run the tool live on the gold systems only (`-only ... -record ...`).
The owner allows live Jev calls (standing authorisation). Sweep thresholds over the
recording with `-replay`: Primary in {0.5, 0.6, 0.7, 0.8, 0.9} and Also in {0.5 … 0.9}.
Pick the pair with the highest primary accuracy, then the best precision on `also` at
recall ≥ 0.8. A wrong primary moves a row to the wrong group, which is visible and
annoying. A wrong `also` is only a chip, so the primary threshold should be the stricter one.
Write the table and your choice in the evidence file.

**Step 3:** Run all systems live with the chosen thresholds and commit the
crosswalk with `status: draft` and every scheme block `source: jev`.

**Step 4:** Run `python tools/check_knowledge.py --strict`, `go test ./internal/knowledge -p 1`.

**Step 5:** Commit: `Jev-drafted ASMM and NATSPEC crosswalk; primary accuracy N% on 25-system gold (lane A)`.

**STOP.** The owner reviews the crosswalk. Owner fixes are written with `source: owner`.

---

## Phase 3: Per-project overrides, preference and API

### Task 8: Migration

**Files:** create `internal/db/migrations/021r_scope_classifications.sql`. Check the
directory first and use the next free letter after the last `021*` file.

```sql
-- A user's classification of one system on one project, per scheme. Absent
-- means the catalogue crosswalk applies. Rows are keyed by system, not by
-- code: costs, packages and work items never point at a classification, so
-- changing a code can never move money or scope (docs/plans/2026-10-12-asmm-natspec-scope-views.md).
CREATE TABLE scope_classifications (
 org_id uuid NOT NULL REFERENCES orgs(id) ON DELETE CASCADE,
 project_id uuid NOT NULL,
 system_id text NOT NULL CHECK (length(system_id) BETWEEN 1 AND 200),
 scheme text NOT NULL CHECK (scheme IN ('asmm','natspec')),
 primary_code text CHECK (primary_code IS NULL OR length(primary_code) BETWEEN 1 AND 20),
 also_codes text[] NOT NULL DEFAULT '{}' CHECK (cardinality(also_codes) <= 20),
 edition text NOT NULL CHECK (length(edition) BETWEEN 1 AND 100),
 updated_by uuid NOT NULL,
 updated_at timestamptz NOT NULL DEFAULT now(),
 version bigint NOT NULL DEFAULT 1 CHECK (version > 0),
 PRIMARY KEY (org_id, project_id, system_id, scheme),
 FOREIGN KEY (org_id, project_id) REFERENCES projects(org_id, id) ON DELETE CASCADE,
 FOREIGN KEY (org_id, updated_by) REFERENCES users(org_id, id) DEFERRABLE INITIALLY DEFERRED
);

-- How a user likes to see scope. One row per user; absent means 'systems'.
CREATE TABLE user_preferences (
 org_id uuid NOT NULL,
 user_id uuid NOT NULL,
 scope_view text NOT NULL DEFAULT 'systems' CHECK (scope_view IN ('systems','asmm','natspec')),
 updated_at timestamptz NOT NULL DEFAULT now(),
 PRIMARY KEY (org_id, user_id),
 FOREIGN KEY (org_id, user_id) REFERENCES users(org_id, id) ON DELETE CASCADE
);
```

Check that `projects` has `UNIQUE (org_id, id)`; other FKs such as `work_items` reference it,
so it should. Look at how `internal/db/migrate_fk_sweep_test.go` and
`internal/db/testdata/next_wave_foreign_keys.json` list tenant FKs, and add the new
FKs there if that test requires it. Run `go test ./internal/db -p 1` and expect PASS.

Commit: `Migration: per-project classification overrides and user scope view (lane A)`.

### Task 9: Store

**Files:** create `internal/store/classification.go`, `internal/store/classification_test.go`.

```go
type ClassOverride struct {
	SystemID, Scheme, Primary, Edition string
	Also    []string
	Version int64
}

// ClassOverrides returns the project's overrides, keyed system|scheme.
func (s *Store) ClassOverrides(ctx context.Context, org, project string) (map[string]ClassOverride, error)

// SetClassOverride upserts one override. expected 0 means "must not exist yet";
// otherwise the row's version must match, or ErrConflict.
func (s *Store) SetClassOverride(ctx context.Context, org, project, user string, o ClassOverride, expected int64) (ClassOverride, error)

// ResetClassOverride deletes it; the crosswalk applies again.
func (s *Store) ResetClassOverride(ctx context.Context, org, project, system, scheme string) error

func (s *Store) ScopeView(ctx context.Context, org, user string) (string, error)  // "systems" when absent
func (s *Store) SetScopeView(ctx context.Context, org, user, view string) error
```

Follow the style of the existing store files (raw pgx queries are fine; see `works.go`).
Every query filters `org_id = $1` **and** checks that the project belongs to the org:
`SetClassOverride` must return `ErrNotFound` for a project of another org. Look at how
`internal/store/reading.go` handles that.

**Tests** (real test DB, same helpers as `reading_test.go`): set → read back;
version conflict → `ErrConflict`; a project in another org → `ErrNotFound`, nothing written;
reset → gone; deleting the project cascades; the scope view defaults to `systems`;
an invalid view is rejected by the CHECK.

Commit: `Store classification overrides and scope view (lane A)`.

### Task 10: Merge overrides into the profile JSON, and build the views on the server

Grouping happens in Go so `go test` can cover it. The web app has no unit test
runner and must not get one (no new dependency). The profile JSON carries all three
groupings, so the toggle is instant.

**Files:** create `internal/profile/classify.go`, `internal/profile/classify_test.go`;
modify `internal/httpapi/profile.go` (the `systems` building block around lines 358–392 and
the response struct); modify `web/src/profileApi.ts` (types only).

**Step 1: Pure function, failing tests first.**

```go
// Effective returns a system's codes in one scheme: the user's override if
// any, else the crosswalk. Review is true when the override was made against
// an older catalogue edition.
func Effective(cat *knowledge.Catalog, overrides map[string]store.ClassOverride, system, scheme string) ClassCell

type ClassCell struct {
	Primary string   `json:"primary"`           // "" = unresolved
	Also    []string `json:"also"`
	Source  string   `json:"source"`            // jev | owner | user
	Review  bool     `json:"review,omitempty"`
	Version int64    `json:"version,omitempty"` // override version for the next write
}

// View groups the system rows under one scheme's groups and codes. A row is
// listed once, under its primary code; rows with no primary go to a final
// "unclassified" entry that is always present when non-empty. Groups and
// codes keep the catalogue's order. Only codes with at least one row appear.
func View(cat *knowledge.Catalog, scheme string, rows []RowRef) []ViewEntry

type RowRef struct{ Leaf, Top, Primary string }
type ViewEntry struct {
	ID    string      `json:"id"`    // group code, or "unclassified"
	Code  string      `json:"code"`
	Label string      `json:"label"`
	Codes []ViewCode  `json:"codes"` // sub-headings inside the pane
}
type ViewCode struct {
	Code   string   `json:"code"`
	Label  string   `json:"label"`
	Leaves []string `json:"leaves"`
}
```

If the `profile` package must not import `store`, define a small `ClassOverride` input type
in `profile` and convert in httpapi. Do whatever the package boundaries already do, and look
before choosing.

Tests:
- an override wins and reports `source: user`;
- an override with an old edition gives `review: true`;
- a system that is not in the crosswalk is unresolved;
- `View` lists each leaf exactly once across all entries (**no double counting**, even
  when `also` holds other codes);
- an unresolved leaf appears under `unclassified`;
- entry order follows the catalogue order;
- the total number of leaves across `View` equals the number of input rows, for both schemes
  (the invariant that switching views preserves the scope).

**Step 2: Wire it into the profile handler.** Load overrides once per request
(`ClassOverrides`, one query). On each `systemRowJSON`, add
`Classes map[string]ClassCell \`json:"classes,omitempty"\`` for each present scheme. On the
response, add:

```go
Views   map[string][]profile.ViewEntry `json:"views,omitempty"`   // "asmm", "natspec"
Schemes []knowledge.ClassScheme       `json:"schemes,omitempty"` // names, editions, codes for the editor
ScopeView string                      `json:"scope_view"`        // the signed-in user's preference
```

Also give each top-level system group a `Classes` value, so compliance inputs can be placed
in a classification view (see Task 13).

If `cat.Schemes()` is empty (no catalogue yet), leave all of these out. The page must look
exactly as it does today.

**Step 3:** Mirror the new fields in `web/src/profileApi.ts` (`ClassCell`, `ViewEntry`,
`ViewCode`, `ClassScheme`; `SystemRow.classes?`; `Profile.views?`, `schemes?`, `scope_view`).

**Step 4:** Run `go test ./internal/profile ./internal/httpapi -p 1` and `npm --prefix web run build`. Both must PASS.

Commit: `Profile carries ASMM and NATSPEC classes and grouped views (lane A)`.

### Task 11: Routes

**Files:** modify `internal/httpapi/projects.go` (route map ~line 154 and the timing map
~line 248), `internal/httpapi/profile.go` (handlers); create
`internal/httpapi/classification_test.go`.

- `PUT /projects/{id}/profile/classification`, timed `pathProfileEdit`. Body:
  `{"system": "...", "scheme": "asmm|natspec", "primary": "0451" | null, "also": [...], "version": 0}`
  or `{"system": "...", "scheme": "...", "reset": true}`. Validation **in code, at the
  boundary**: the system exists and is not deprecated; the scheme exists; primary and every
  `also` code exist in that scheme's catalogue; no duplicates; primary is not in `also`; at most 20 codes.
  Store the scheme's **current** edition. Return the rebuilt profile JSON, as the other
  profile edits do. Responses: 400 for invalid input, 404 for an unknown or foreign project,
  409 for a version conflict. Check the origin as the other PUTs do.
  **Register the route before** `PUT /projects/{id}/profile/{key}` if the mux would
  otherwise match `{key}`. Look at how `profile/scope` avoids that clash and do the same.
- `PUT /me/preferences`, timed `pathProfileEdit`. Body `{"scope_view": "systems|asmm|natspec"}`.
  Returns 204. Check the origin.

**Tests:** each 400 case; foreign org → 404 and nothing written (copy the pattern in
`access_sweep_test.go`, and **add the new routes to the access sweep** if it lists
routes); a valid edit changes only that scheme (the other scheme's cell is unchanged);
reset returns to the crosswalk value; a preference round-trip shows up in the next
`GET /profile` as `scope_view`.

Run `go test ./internal/httpapi ./internal/latency -p 1` and expect PASS.

Commit: `Routes to edit a system's classification and save the scope view (lane A)`.

---

## Phase 4: The toggle and editing (Profile → Scope and works)

The UI is `web/src/ScopeWorks.tsx`. Read the whole file first. Today the rail
(`<nav className="sw-rail">`) lists Planning (0), then the 13 systems numbered 1–13,
then "Other work". The pane lists the chosen entry's `SystemLine` rows.

### Task 12: View toggle and rail

**Files:** modify `web/src/ScopeWorks.tsx`, `web/src/Profile.tsx` (pass `scope_view` and a
setter), `web/src/profileApi.ts` (`setScopeView`), `web/src/styles.css`.

1. In `.sw-head`, add a segmented control labelled **"View scope by:"** with buttons
   Systems / {schemes[0].name} / {schemes[1].name}. Use
   `role="radiogroup"`, with `role="radio"` and `aria-checked` on each button. Leave the control
   out entirely when `data.schemes` is empty.
2. State: `const [view, setView] = useState(data.scope_view || "systems")`. On click, set
   the state immediately (instant), then `profileApi.setScopeView(view)` in the background.
   If that fails, keep the choice and show nothing. A lost preference is harmless.
3. Rail entries in a classification view:
   `Planning and approvals` first (unchanged), then one entry per `data.views[view]` entry,
   showing **the group code** in `.sw-num` (instead of the index `i`) and the group label, then
   `Unclassified` if present, then `Other work` if there are orphans. Counts work as today
   (`on/total` of rows in the entry).
4. Pane in a classification view: for the chosen entry, render one sub-heading per
   `ViewCode` (`<h4>{code} {label}</h4>`), each followed by its `SystemLine` rows (look the rows up
   by leaf from `data.systems`). Reuse `SystemLine` unchanged.
5. Selecting: `selected` holds the entry id. When the view changes, select the first non-planning
   entry that contains the previously selected leaf's group if possible, otherwise the first entry.
6. The scope tick, presets, "Reset to defaults" and work items behave **exactly** as before,
   because they act on leaves, which are the same in every view.

### Task 13: Inputs in classification views

Flags and compliance inputs are homed to top-level systems
(`knowledge/profile/layout.yaml` `homes`). In a classification view, show a system's
inputs under the entry that holds its top-level system's **primary** group (from the
top-level `Classes` added in Task 10). If that top-level system is unresolved, show them
under Planning. Change `entryInputs(data, id)` so it takes the view into account; do not
duplicate it.

### Task 14: Editing a classification

**Files:** `web/src/ScopeWorks.tsx` (the `SystemLine` detail panel), `web/src/profileApi.ts`
(`setClassification`), `web/src/styles.css`.

In a row's opened detail, add a **Classification** block with one line per scheme:

```
NATSPEC   0451 Windows and glazed doors   + 0455 Door hardware     [Jev] [Change] [Reset]
ASMM      Unclassified                                              [Change]
```

- A source badge reads `Jev`, `Reviewed` (for `owner`) or `You` (for `user`). A `review` flag shows
  "Catalogue updated: check".
- **Change** opens an inline picker: a text filter over `schemes[x].codes` (code or label
  contains the typed text), a list to pick the primary from, and checkboxes for additional codes. Save
  calls `setClassification` and updates from the returned profile. An edit changes only that scheme.
- **Reset** appears only when `source === "user"` and returns the row to the crosswalk.
- On a 409 conflict, reload the profile and show "Changed elsewhere: reloaded".
- Keyboard: the picker is a `listbox`, Escape closes it, and focus returns to Change.

Run `npm --prefix web run build` and expect no type errors.

### Task 15: End-to-end test

**Files:** create `web/tests/classification.spec.ts`; check `web/tests/e2eserver`, which may need
fixture catalogues if the test knowledge root has none.

Cover:
1. The toggle is visible; switching to NATSPEC shows group codes in the rail; the
   "N systems in scope" count is **the same in all three views**.
2. A ticked system stays ticked after switching views and back.
3. Change a system's NATSPEC primary → it moves to the new group in NATSPEC view; its
   ASMM cell is unchanged; reload the page → still moved (it persists).
4. Reset → back under its Jev group.
5. Reload the page → the last chosen view is restored (preference).
6. A system with no primary is listed under Unclassified, never hidden.
7. Costs page totals are identical before and after steps 3–4 (scope and money never move).

Run `npm --prefix web run test:e2e -- classification.spec.ts`. Watch for the
known e2e port-4173 teardown problem: kill a stray server on 4173 if the run hangs.

Commit (Tasks 12–15 can be one commit if they land together):
`View scope by Systems, ASMM or NATSPEC, with per-project edits (lane A)`.

---

## Phase 5: Downstream (contract now, screens later)

### Task 16: Design note and API contract

**Files:** create `docs/design/2026-10-12-scope-classification.md`.

Write in prose. **Do not build any of the screens listed here.** Each one gets its own
plan once the owner chooses the direction. Cover:

- **The contract.** Downstream code asks
  `profile.Effective(cat, overrides, system, scheme)` for any work item's `system_id`.
  Quantities, budgets, package scope and cost lines stay keyed to `work_item_id` and
  `system_id`, **never to a code**. Group-by is presentation only. Totals are always
  summed over work items, never over codes. A row is listed once, under its primary code, so
  nothing is counted twice.
- **Cost plan.** `knowledge/costs/elements.yaml` codes 01–13 could become the ASMM/AIQS
  elemental codes, or the Costs grid could gain the same "View by" toggle. This is an owner
  decision (O1's answer decides which fits better).
- **Trade RFTs.** NATSPEC worksection numbers cite the specification sections a trade
  package covers (`package_scope_items` → work items → systems → NATSPEC).
- **Consultant RFPs.** Scope tables can cite NATSPEC worksections per discipline.
- **Programme and PMP.** Group activities and scope statements by the user's view.
- **Review of issued documents.** Issued reports are frozen (the `report_settings`
  immutability trigger). A classification edit after issue marks the next draft "classification
  changed since issue", never the issued version. Describe this; do not build it.
- **Document evidence (later, code only, no Jev).** NATSPEC project specifications print
  worksection numbers in their headings. A regex over passage `section` paths can show
  which worksections a project's spec actually contains. That is free evidence for the project
  and a check on the crosswalk. Name it as a follow-up plan.

Commit: `Design note: how classification views feed costs, RFP, RFT and programme`.

---

## Phase 6: Verify and report

### Task 17: Full gate and report

1. `python tools/check_knowledge.py --strict`: PASS.
2. `pwsh tools/check.ps1`: PASS, including the latency gate.
3. Measure `PUT /projects/{id}/profile/classification` and `GET /projects/{id}/profile` on
   the dev server against Section 0.5 (the existing `/speed` route reports timings), and
   write the p50/p90 into the evidence file.
4. Petersham check on the dev server (`:8080 /dev/login`; **never** send the owner an
   invite-token link): toggle all three views, change one classification, reload.
5. Check every spec test in Section 6 below against the test that proves it.
6. Report to the owner in plain language: what calls what, what can fail (catalogue file
   missing → the toggle is hidden; Jev unavailable → only the offline tool is affected), timing
   against budget, no new dependency.

Merge to local `main` (standing authorisation). **Do not push.** Never mark anything `reviewed`.

---

## 6. The spec's release tests, mapped

| Spec test | Proved by |
|---|---|
| Jev populates both systems with valid codes | Task 6 merge tests (unknown codes dropped); Task 3 loader rejects invalid crosswalk; Task 7 gold accuracy |
| Switching views preserves scope, quantities and totals | Task 10 `View` leaf-count invariant; Task 15 steps 1, 2, 7 |
| Manual edits persist and survive later classifier runs | Task 9 store tests; Task 15 step 3; the tool never touches `scope_classifications` and never overwrites `owner` entries (Task 6 test) |
| Multiple mappings do not double-count | Task 10 "each leaf exactly once" test; Task 15 step 7 |
| Unclassified items remain visible | Task 10 unclassified test; Task 15 step 6 |
| Shared scope changes flag downstream documents without overwriting approved content | Task 10 `review` flag on an edition change; downstream flagging is designed in Task 16 and built later |
| BCA/NCC class, subtype and work type stay separate Profiler fields | Untouched: no task edits `taxonomy.yaml` or the header fields |

## 7. Owner decisions (fill in before Task 1)

Answered by the owner on 2026-10-12:

- **O1: ANZSMM 2018.** The scheme id stays `asmm` in code; the display `name` is
  "ANZSMM 2018". ANZSMM is a paid AIQS/NZIQS document, so the owner supplies the PDF in
  `data/reference/classifications/`. There is no free official copy, and you must not
  use unofficial uploads (Scribd, Studocu) or type codes from memory.
- **O2: the latest NATSPEC, which is the National Classification List of April 2026.**
  Already done: `data/reference/classifications/natspec-2026-04.yaml` (473 worksections,
  121 subgroups, 19 workgroups) was transcribed by script from the free NATSPEC PDF (saved
  beside it, git-ignored) and checked against a second extraction. For Task 4, copy it
  to `knowledge/classifications/natspec.yaml`. It adds a `subgroups:` list and a
  `subgroup` on each code; extend the schema, checker and loader to accept them. Codes take
  their parent from the document, not the code prefix (0001 sits under 001). Workgroups
  marked `aus_spec: true` (civil and maintenance) are never shown in the scope view.
- **O3: two views, ANZSMM | NATSPEC, with ANZSMM the default.** ANZSMM replaces today's
  0–13 rail as the main way scope is organised on screen ("ANZSMM becomes the
  systems"). Remove the Systems option from the toggle (Task 12) and number the rail with
  ANZSMM codes. Recommended, pending the owner's confirmation: the building systems in
  `knowledge/systems.yaml` stay as the internal engine, because Jev's reading questions,
  NCC rules, determinants, presets and work items are keyed to them. Only the screen
  changes. **Do not rename or delete system ids.**
- **O4: licensing.** NATSPEC publishes its list free, so its codes and titles are fine to
  commit. For ANZSMM, commit section numbers and titles only, and write `usage` notes in
  our own words. Source PDFs are git-ignored (`data/reference/classifications/*.pdf`).
- **O5: recommended default** (each system listed once, under its main code, with the
  other codes shown as tags). Explained to the owner; awaiting a final answer.

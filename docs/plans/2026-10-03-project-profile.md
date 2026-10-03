# Project Profile and One-Line Register Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Draft a project profile (project header, parts, systems checklist,
compliance) from filed documents using Jev only, let the user polish it, and
compress the document list into a one-line register.

**Architecture:** Background stages after filing add profile questions to the
existing per-passage Jev fan-outs (labelling, then evidence). Code harvests
candidates, reconciles readings across documents into precomputed
`profile_rows`, and the page reads them in one SQL query. User values are
final. The UI becomes a two-column project page: register (one third) and
profile (two thirds).

**Tech Stack:** Go single binary (net/http, pgx, sqlc), PostgreSQL 17, YAML
knowledge (`gopkg.in/yaml.v3`), React 19 + Vite + TypeScript, Playwright,
Python checker for `knowledge/`.

**Design (read first):** `docs/design/2026-10-03-project-profile.md`. Also
`AGENTS.md`, `docs/design/2026-09-29-foundation-design.md`, `knowledge/SCHEMA.md`.

## Status (3 October 2026, branch `profile/impl`)

**Done and committed on `profile/impl`** (worktree `D:\AI Projects\sitewise-profile`):
P01 schema and checker, P06 systems, P07 determinants, P08 `knowledge/profile/`,
P09 loader, P10 harvest, P11 questions, P12 reconcile, P04 store (migration 007),
P03 worker in `serve`, P13 facts and rebuild, P14 API with budgets and bench samples,
P05 and P15 UI (left nav, profile, register), and P02 (manifest and draft
answer keys). `go test ./...`, the strict knowledge check,
the web build and both Playwright specs pass. A live smoke run on the Hale brief
profiled it in 30 s with no Jev errors (`docs/evidence/2026-10-03-profile-smoke/`).

**Not done (next agent):**
- **P02** is drafted: `data/eval/profile/answer-keys/*.yaml` (unreviewed agent
  drafts; every id checked against knowledge). The owner reviews them (O1).
- **P16** `cmd/profile-eval` harness (record, replay, score, no-wrong-green gate).
- **P17** live run over all five projects, then tuning. Known gaps: "Not
  applicable" absences are not yet read as *not included*, and MHE battery
  charging was read as EV charging.
- **O1 (owner):** review the answer keys, approve the provisional floors
  (`approved_by_owner`), and review the `type_of_construction` and
  `compartment_limits` tables and their rules. Until then derived values read
  "Awaiting owner review".
- **Merge:** the branch is based on `main` at `3ad28d5` and does not include
  the corpus-hardening session's uncommitted work. Merge that work first.
  Expect conflicts in `web/src/DocumentRow.tsx`, `web/src/Project.tsx`,
  `web/src/api.ts` (`EVENT_KINDS`), `internal/httpapi/projects.go` (route
  map), `internal/httpapi/server.go` (Options), `cmd/intake-bench/main.go`
  and migration numbering (this branch uses `007`; hardening adds `006`). The
  drawing-sheet UI (sheet rows, "Sheet n of N") must be carried into
  `Register.tsx`'s detail row. `DocumentRow` itself is now unused (its
  `FieldCell`, `HeadState`, `FIELDS` and `REASONS` are reused); delete it after
  the merge.
- Tests for this branch ran against a private Postgres on port 5434, because
  test files require a database named exactly `sitewise_test` and port 5433
  is shared with the other session.

**Deviations from the plan below, all deliberate:**
- Layout: the owner asked for the register in the **right** third and a
  **left nav** with a project switcher (after Clerk). The profile is in the
  middle.
- No separate debounced `profile` job: the `jobs` table needs a document id.
  The profile rebuilds in code at the end of each document's label and
  evidence stages, serialised per project by an advisory lock. A user edit
  runs the same rebuild and returns the whole profile.
- Profile store queries use pgx directly (JSONB and batches), not sqlc.
- `profile_facts` has no assertion column. Assertions are stored as their own
  facts (`det.<id>.assertion`) and paired by passage in `Reconcile`.
- Register rows can be open several at once (simpler, and the e2e flow needs it).
- `ncc_class` is asked per mentioned option (`det.ncc_class.<opt>`), because a
  choice cannot return a set.
- Scale fields are asked as `hdr.scale.<key>` on any passage where their
  trigger matched.

---

## 0. Rules for every agent

1. **Jev only.** No LLM, embedding, chat or other AI call anywhere, including
   tests and tools. Jev picks; code harvests, computes, reconciles and writes.
   Every new Jev question cites its TypeSafe page in a code comment
   ([fan-out](https://docs.typesafe.ai/patterns/fan-out),
   [pre-parsed extraction](https://docs.typesafe.ai/cookbooks/pre_parsed_value_extraction_cookbook),
   [intent routing](https://docs.typesafe.ai/patterns/intent-routing),
   [confidence](https://docs.typesafe.ai/confidence),
   [jev-1.13 limitations](https://docs.typesafe.ai/model-jaggedness/jev-1.13),
   [how to build](https://docs.typesafe.ai/concepts/how-to-build-with-system-one)).
   Jev never counts, compares, converts units, does dates or writes text.
2. **No new Jev round trips.** Profile questions join the two existing
   per-passage calls (`LabelCall`, `EvidenceCall` in `internal/jobs/worker.go`).
   All profile Jev work runs at `jev.PriorityBackground`.
3. **Nothing judged while a user waits.** GET and PUT on the profile read and
   write precomputed rows only.
4. **TDD.** Write the failing test, watch it fail, implement, watch it pass,
   commit. Small commits. Never weaken an existing assertion to get green.
5. **Knowledge rules.** Everything you write in `knowledge/` is
   `status: draft`. Only the owner sets `reviewed`. Run
   `python tools/check_knowledge.py --strict` after every knowledge edit.
   Numbers come from the instrument itself, never from seed prose.
6. **Copy data from `../clerk`, never code.**
7. **A dependency needs a reason** in the commit message.
8. **Stay inside your task's "Owns" list.** If you must touch another file,
   stop and record why in your task notes. Shared contracts (section 2) change
   only by editing this plan first.

### Concurrent work you must not disturb

A corpus-hardening session is editing these files in the main working tree,
uncommitted, as of 3 October 2026. **Do not edit them until the owner says
that session has committed**, then branch from that commit:

`internal/intake/**`, `internal/identity/**`, `internal/eval/**`,
`cmd/corpus-eval/**`, `data/intake/**`, `data/eval/intake/**`,
`internal/store/filing.go`, `internal/store/views.go`, `internal/store/sheets.go`,
`internal/httpapi/documents.go`, `internal/httpapi/projects.go`,
`internal/httpapi/server.go`, `internal/httpapi/devlogin*.go`,
`internal/db/migrations/006_drawing_sheets.sql`, `web/src/DocumentRow.tsx`,
`web/src/Project.tsx`, `web/src/api.ts`, `testdata/identity/**`, `tools/*corpus*`.

### Workspace setup (PowerShell, once per task)

```powershell
$Main = 'D:\AI Projects\sitewise'
git -C $Main worktree add "$Main-$TASK" -b "profile/$TASK" main   # $TASK e.g. 'p01'
Set-Location "$Main-$TASK"
New-Item -ItemType Junction -Path .tools -Target "$Main\.tools" | Out-Null   # Go, Postgres, sqlc live here (git-ignored)
$env:PATH = "$PWD\.tools\go\bin;$PWD\.tools\go-path\bin;$env:PATH"
$env:GOCACHE = "$PWD\.tools\go-cache"; $env:GOPATH = "$PWD\.tools\go-path"; $env:GOTOOLCHAIN = 'local'
$env:SITEWISE_TEST_DATABASE_URL = 'postgres://sitewise@127.0.0.1:5433/sitewise_test?sslmode=disable'
```

PostgreSQL must be running on 5433 (`tools/dev.ps1` starts it). Web tasks
also run `npm --prefix web ci` in the worktree.

### Commands

| What | Command |
|---|---|
| Knowledge check | `python tools/check_knowledge.py --strict` |
| Checker tests | `python -m unittest discover -s tools -p 'test_*.py' -v` |
| Go tests (one package) | `go test ./internal/profile/ -run TestName -v` |
| Go tests (all) | `go test ./...` |
| Regenerate sqlc | `sqlc generate` (binary in `.tools\go-path\bin`) |
| Web build | `npm --prefix web run build` |
| E2E | `npm --prefix web run test:e2e` (build first) |
| Full gate | `tools/check.ps1` |

### Merging

Merge to `main` in dependency order (section 3). Rebase on `main` before
merging. Resolve conflicts by keeping both sides' intent; re-run your task's
tests and `go test ./...` after every rebase.

---

## 1. Task list and who should take it

| ID | Task | Depends on | Blocked by hardening session | Agent level |
|---|---|---|---|---|
| P01 | Knowledge schema contract and checker | none | no | standard |
| P02 | Eval manifest and draft answer keys (5 projects) | none (P01 for source ids) | no | needs construction judgement |
| P03 | Wire the background worker into `serve` | none | no | standard |
| P04 | Profile tables, store and isolation tests | none | yes (migration number, store files) | standard |
| P05 | One-line register UI | none | **yes** (`DocumentRow.tsx`, `Project.tsx`) | standard (UI) |
| P06 | Systems: add, widen, deprecate | P01 | no | needs construction judgement |
| P07 | Determinants: add, triggers, `stated_in`, compartment derivation | P01 | no | needs construction judgement |
| P08 | `knowledge/profile/`: Clerk taxonomy, project facts, typical systems | P01 | no | standard |
| P09 | Go loader for determinants and `knowledge/profile/` | P01 (fixtures); real data after P06–P08 | no | standard |
| P10 | Candidate harvesters (code) | P09 | no | standard |
| P11 | Profile Jev questions in the two fan-outs | P09, P10, P03 | no | standard |
| P12 | Reconciliation (pure code) | P09 | no | standard |
| P13 | Profile stage job and persistence | P03, P04, P11, P12 | no | standard |
| P14 | Profile HTTP API, SSE event, speed budgets | P04, P12, P13 | yes (`projects.go`, `server.go`) | standard |
| P15 | Profile UI and two-column layout | P05, P14 | yes (`Project.tsx`) | standard (UI) |
| P16 | Profile eval harness (replay and live) | P02, P13 | no | standard |
| P17 | Live run, provisional floors, evidence write-up | all | no | needs judgement + owner |
| O1 | **Owner:** review answer keys, approve provisional floors, review `type_of_construction` and `compartment_limits` tables and their rules | P02, P07 | — | owner only |

Waves: **Now:** P01, P02, P03 (and P04/P05 once the hardening session
commits). **After P01:** P06, P07, P08, P09 in parallel. **Then:** P10, P12
in parallel; P11 after P10. **Then:** P13, P14, P15, P16. **Last:** P17.

---

## 2. Shared contracts

Every task codes against these. Change them here first, then in code.

### 2.1 Knowledge YAML additions (P01 defines, P06–P09 use)

- `status` may be `draft`, `reviewed` or `deprecated`. A `deprecated` record
  must have `replaced_by: <id of the same kind>`, which exists and is not
  deprecated. Strict check: no `systems`, `runs_on`, `from`, `to` or
  `attaches_to` anywhere may name a deprecated id.
- A `sources` item may be `{document: <id>, anchor: "<exact line>"}`. The
  `<id>` must exist in `data/eval/profile/manifest.json` (`documents[].id`).
  The anchor is recorded but cannot be checked (the corpus is private).
- Determinants gain optional fields:

```yaml
triggers:                      # RE2 patterns, matched case-insensitively by code.
  - '\bBAL[- ]?(LOW|12\.5|19|29|40|FZ)\b'   # A question is asked for a passage only
  - 'bush ?fire attack level'               # when at least one trigger matches it.
stated_in:                     # where this is usually stated; ids must exist in
  - {kind: report, discipline: consultant.bushfire, label: Bushfire assessment report}  # data/intake/kinds.json and disciplines.json
profile_group: site            # classification | site | services | fire ; omit = not shown in the profile
```

- New directory `knowledge/profile/` with three files (P08 writes them,
  P01 makes the checker validate them):
  - `taxonomy.yaml`: `building_classes` (each: `id`, `label`, `subclasses`
    with `id`, `label`, `ncc_class`, `scale_fields` with `key`, `label`,
    `type`, `unit`, `basis_required`, `triggers`), `work_types` (`id`, `label`),
    `conditions` (`key`, `label`, `options` with `id`, `label`). Every record
    has `status` and `sources`.
  - `project_facts.yaml`: list key `facts`, same shape as determinants
    (`id`, `label`, `value`, `extraction`, `options`, `triggers`, `question`,
    `status`, `sources`).
  - `typical_systems.yaml`: list key `typical`, each
    `{subclass: <taxonomy subclass id>, work_type: <id>, systems: [<leaf system ids>], status, sources}`.

### 2.2 Question ids and wording (P11 builds, P12/P13/P16 read)

| Question id | Type | Options (criteria keys) | Asked in |
|---|---|---|---|
| `det.<determinant_id>` | choice | pre_parsed: `c1`…`cN` (code candidates) + `none`; choice: the determinant's option ids + `not_stated` | LabelCall, only when a trigger matched |
| `det.<determinant_id>.assertion` | choice | `stated`, `required`, `allowance`, `not_stated` | LabelCall, with its determinant |
| `fact.<fact_id>` | choice | as for determinants | LabelCall, only when a trigger matched |
| `hdr.building_class` | choice | taxonomy class ids + `not_stated` | LabelCall, header-routed passages only |
| `hdr.subclass` | choice | all taxonomy subclass ids + `not_stated` | same |
| `hdr.work_type` | choice | work type ids + `not_stated` | same |
| `hdr.cond.<key>` | choice | condition option ids + `not_stated` | same |
| `hdr.scale.<key>` | choice | `c1`…`cN` (code candidates) + `none` | LabelCall, only when the scale field's trigger matched and code found candidates (any passage, not only header-routed) |
| `sys.<leaf_id>.presence` | choice | `included`, `not_included`, `not_stated` | EvidenceCall, leaves the passage is labelled with |
| `sys.<leaf_id>.provider` | choice | `contractor`, `owner`, `others`, `not_stated` | same |

**Header routing (code):** a passage is header-routed when its document kind
is `design_brief`, `specification`, `report`, `contract` or `commercial`
(quotes and tenders) and either its
`section` matches `(?i)description|outline|scope|introduction|project details|key development`
or its ordinal is below 12.

**Wording** (copy exactly; `<label>` is the record's `label`):

- Assertion: instructions ``Using `text`, how does the passage present the <label>?``
  - `stated`: "It states the value as a fact about this project or site, such as an assessment result, a certificate or a measured value."
  - `required`: "It requires the value: a brief, consent, contract or specification says the building must have or achieve it."
  - `allowance`: "It treats the value as an assumption or a pricing allowance, for example 'we have allowed for' or 'assumed'."
  - `not_stated`: "The passage does not give a value for it."
- Presence: instructions ``Using `text`, does the passage say whether the completed project will have <label>?``
  - `included`: "It says the project will have it: it is specified, required, priced, allowed for, or supplied by any party."
  - `not_included`: "It says the project will not have it: not applicable, not required, or not part of the project."
  - `not_stated`: "It mentions it without saying whether the project will have it, or only excludes it from one party's price."
- Provider: instructions ``Using `text`, who does the passage say provides <label>?``
  - `contractor`: "The builder, contractor or tenderer provides it as part of their works or price."
  - `owner`: "The owner, client or principal supplies or arranges it."
  - `others`: "An authority, developer, separate contractor or other named party provides it."
  - `not_stated`: "The passage does not say who provides it."
- Pre-parsed criteria: `cN` → `"<verbatim value> (in: '<up to 80 chars around it>')"`;
  `none` → `"The passage does not state the <label>, or no candidate is it."`
- Header: instructions ``Using `text` and `document`, which <label> does the passage state for this project?``,
  each option's criterion is its taxonomy label plus its subclass/NCC hint,
  `not_stated` → `"The passage does not state it."`

**State** for both calls stays `passageState` plus one field:
`"candidates": {"det.floor_area": [{"id":"c1","value":"2,000sqm","context":"…"}], …}`.
Omit the field when empty. Questions name `text`, `document` or `candidates`
explicitly.

### 2.3 Thresholds file (P03 creates, P13/P17 use)

`data/profile/thresholds.json`:

```json
{
  "question_version": "profile-1",
  "status": "provisional",
  "approved_by_owner": false,
  "label_min_noul": 0.5,
  "shapes": {
    "presence":   {"options": 3, "amber": 0.6, "green": null},
    "provider":   {"options": 4, "amber": 0.6, "green": null},
    "assertion":  {"options": 4, "amber": 0.6, "green": null},
    "header":     {"amber": 0.6, "green": null},
    "determinant":{"amber": 0.6, "green": null}
  }
}
```

`green: null` means green is withheld. While `approved_by_owner` is false the
server still runs, but the profile shows "Provisional thresholds" in its
status line. A missing file means nothing is applied, and the profile says
so.

### 2.4 Database (P04 creates; P13/P14 use)

`internal/db/migrations/007_project_profile.sql` (use the next free number on
your base commit):

```sql
CREATE TABLE project_parts (
    org_id uuid NOT NULL,
    id uuid NOT NULL,
    project_id uuid NOT NULL,
    label text NOT NULL,
    kind text NOT NULL,
    ncc_class text,
    created_at timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (org_id, id),
    UNIQUE (org_id, project_id, label),
    FOREIGN KEY (org_id, project_id) REFERENCES projects (org_id, id) ON DELETE CASCADE,
    CHECK (kind IN ('whole', 'building', 'part', 'storey', 'compartment', 'tenancy', 'outbuilding'))
);

-- One Jev or rule reading of one question in one passage. Facts are replaced
-- per document when its evidence stage reruns; they are never edited.
CREATE TABLE profile_facts (
    org_id uuid NOT NULL,
    id uuid NOT NULL,
    project_id uuid NOT NULL,
    document_id uuid NOT NULL,
    passage_id uuid,
    question_id text NOT NULL,
    value text NOT NULL,
    unit text,
    basis text,
    part_label text,
    excerpt text,
    confidence double precision,
    decided_by text NOT NULL,
    question_version text NOT NULL,
    created_at timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (org_id, id),
    FOREIGN KEY (org_id, project_id) REFERENCES projects (org_id, id) ON DELETE CASCADE,
    FOREIGN KEY (org_id, document_id) REFERENCES documents (org_id, id) ON DELETE CASCADE,
    CHECK (decided_by IN ('jev', 'rule')),
    CHECK (excerpt IS NULL OR char_length(excerpt) <= 120)
);
CREATE INDEX profile_facts_by_project ON profile_facts (org_id, project_id, question_id);
CREATE INDEX profile_facts_by_document ON profile_facts (org_id, document_id);

-- The user's word. Final: no stage ever writes here.
CREATE TABLE profile_user_values (
    org_id uuid NOT NULL,
    project_id uuid NOT NULL,
    part_id uuid NOT NULL,
    key text NOT NULL,
    value text,
    note text,
    user_id uuid NOT NULL,
    version bigint NOT NULL DEFAULT 1,
    updated_at timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (org_id, project_id, part_id, key),
    FOREIGN KEY (org_id, part_id) REFERENCES project_parts (org_id, id) ON DELETE CASCADE,
    CHECK (note IS NULL OR char_length(note) <= 120)
);

-- Precomputed view the page reads in one query. Rewritten by the profile
-- stage and by a user edit, inside one transaction each.
CREATE TABLE profile_rows (
    org_id uuid NOT NULL,
    project_id uuid NOT NULL,
    part_id uuid NOT NULL,
    key text NOT NULL,
    value text,
    band text NOT NULL,
    assertion text,
    note text,
    sources jsonb NOT NULL DEFAULT '[]',
    alternatives jsonb NOT NULL DEFAULT '[]',
    computed_at timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (org_id, project_id, part_id, key),
    FOREIGN KEY (org_id, part_id) REFERENCES project_parts (org_id, id) ON DELETE CASCADE,
    CHECK (band IN ('green', 'amber', 'red', 'blank', 'suggested', 'user', 'unchecked'))
);

CREATE TABLE profile_builds (
    org_id uuid NOT NULL,
    project_id uuid NOT NULL,
    built_at timestamptz,
    pending_documents integer NOT NULL DEFAULT 0,
    thresholds_version text,
    PRIMARY KEY (org_id, project_id),
    FOREIGN KEY (org_id, project_id) REFERENCES projects (org_id, id) ON DELETE CASCADE
);
```

Profile keys: `hdr.*`, `fact.*`, `det.<id>`, and for systems
`sys.<leaf_id>.presence`, `sys.<leaf_id>.provider`, `sys.<leaf_id>.note`.
Every project gets one `kind = 'whole'` part labelled `Whole project`,
created on first profile write or read.

### 2.5 Go types (P12 defines in `internal/profile/types.go`)

```go
package profile

// Fact is one stored reading (a profile_facts row plus document context).
type Fact struct {
	QuestionID   string  // "det.bal", "sys.hydraulic.gas.presence", "hdr.subclass"
	Value        string  // option id, or the verbatim candidate value
	Unit, Basis  string
	Assertion    string  // from the matching ".assertion" fact; "" when not asked
	PartLabel    string
	Excerpt      string
	Confidence   *float64
	DecidedBy    string  // "jev" | "rule"
	DocumentID   string
	PassageID    string
	DocumentKind string  // data/intake/kinds.json id
	Superseded   bool
}

type Part struct{ ID, Label, Kind, NCCClass string }

type UserValue struct {
	PartID, Key string
	Value       *string // nil = cleared by the user
	Note        string
}

type Source struct {
	DocumentID, PassageID, Excerpt string
	Confidence                     *float64
}

type Alternative struct {
	Value   string
	Sources []Source
}

type Row struct {
	PartID, Key, Value string
	Band               string // green|amber|red|blank|suggested|user|unchecked
	Assertion          string
	Note               string
	Sources            []Source
	Alternatives       []Alternative
	Tenders            string // "", "consistent", "differ"
}

type Thresholds struct {
	LabelMinNoul float64
	Amber        map[string]float64  // shape -> floor
	Green        map[string]*float64 // shape -> nil when withheld
	Approved     bool
	Version      string
}

type Input struct {
	Parts       []Part
	Facts       []Fact
	User        []UserValue
	Thresholds  Thresholds
	Suggested   []string // leaf system ids from typical_systems for the chosen subclass + work type
}

// Reconcile is pure: no I/O, no Jev, deterministic order.
func Reconcile(in Input, cat Catalog) []Row
```

`Catalog` is the small interface P09 exposes (determinants, facts, taxonomy,
leaf systems, deprecations, `Derive`).

### 2.6 HTTP contract (P14 serves, P15 consumes)

`GET /api/projects/{id}/profile` → 200:

```json
{
  "project_id": "uuid",
  "built_at": "2026-10-03T10:00:00Z",
  "pending_documents": 0,
  "thresholds": {"version": "profile-1", "provisional": true},
  "parts": [{"id": "uuid", "label": "Whole project", "kind": "whole", "ncc_class": null}],
  "header": [
    {"key": "hdr.subclass", "label": "Subclass", "part_id": "uuid", "value": "warehouse",
     "band": "amber", "options": [{"id": "warehouse", "label": "Warehouse (Class 7b)"}],
     "sources": [{"document_id": "uuid", "passage_id": "uuid", "excerpt": "single level warehouse tenancy", "confidence": 0.82}],
     "alternatives": []}
  ],
  "systems": [
    {"id": "hydraulic", "label": "Hydraulic", "rows": [
      {"leaf": "hydraulic.gas", "label": "Fuel gas", "part_id": "uuid",
       "presence": {"value": "included", "band": "amber", "tenders": "consistent", "sources": [], "alternatives": []},
       "provider": {"value": "contractor", "band": "red", "tenders": "differ", "sources": [], "alternatives": []},
       "note": {"value": "Two 45 kg LPG gas bottles", "band": "amber", "sources": []},
       "shown_by_default": true}
    ]}
  ],
  "compliance": [
    {"group": "site", "rows": [
      {"key": "det.bal", "label": "Bushfire attack level (AS 3959)", "part_id": "uuid",
       "value": "BAL-LOW", "assertion": "allowance", "band": "amber",
       "sources": [], "alternatives": [],
       "stated_in": ["Bushfire assessment report"], "derived": null}
    ]},
    {"group": "fire", "rows": [
      {"key": "det.type_of_construction", "label": "Type of Construction required", "part_id": "uuid",
       "value": null, "band": "blank", "derived": {"rule": "rule.ncc.type-of-construction", "state": "unknown", "reason": "unreviewed"}}
    ]}
  ]
}
```

`PUT /api/projects/{id}/profile/{key}` body `{"part_id":"uuid","value":"included"|null,"note":"…"}`
→ 200 with the updated row object (same shape as above). Validation at the
boundary: the key exists; the value is one of its options, or a number for
numeric keys, or ≤ 200 characters for text; the note is ≤ 120 characters. 422
otherwise. Writes `profile_user_values` and that key's `profile_rows` row in
one transaction. A change to `hdr.subclass` or `hdr.work_type` also rewrites
the `suggested` band on systems rows in the same transaction (code lookup in
`typical_systems.yaml`).

`POST /api/projects/{id}/parts` `{"label","kind","ncc_class"}` → 201 part.
`PATCH /api/projects/{id}/parts/{part}` `{"label"?, "kind"?, "ncc_class"?}` → 200 part.
There is no delete; a wrong part is renamed or its kind changed.

SSE: new event kind `profile`, payload `{"project_id": "uuid"}`. The client
refetches the profile.

Speed paths (add to `bench/budgets.json` and `routePaths`):
`project_profile_read` (p50 50 ms, p90 150 ms) for GET;
`profile_edit` (p50 50 ms, p90 150 ms) for PUT and the parts routes.

---

## 3. Tasks

Each task: read the design doc, then your card. "Done when" is the bar.

### Task P01: Knowledge schema contract and checker

**Owns:** `knowledge/SCHEMA.md`, `tools/check_knowledge.py`, `tools/test_check_knowledge.py`, `tools/fixtures/profile/**` (create).

**Step 1: Write failing checker tests** in `tools/test_check_knowledge.py`,
one per rule in section 2.1. Use small fixture trees under
`tools/fixtures/profile/` (copy the pattern the existing tests use):

- `test_deprecated_requires_replaced_by`: a deprecated system with no `replaced_by` → error.
- `test_replaced_by_must_exist_and_not_be_deprecated`.
- `test_strict_rejects_reference_to_deprecated_system`: a rule `systems:` naming it → error under `--strict`.
- `test_document_source_must_be_in_manifest`: `{document: x, anchor: y}` with `x` absent from the fixture manifest → error; present → ok.
- `test_triggers_must_compile`: an invalid regex → error. (Python `re` is close enough to RE2 for this check; also reject look-behind `(?<` and backreferences `\1`, which RE2 lacks.)
- `test_stated_in_ids_exist`: unknown kind or discipline → error (read `data/intake/kinds.json` and `disciplines.json`).
- `test_profile_group_values`.
- `test_profile_dir_shapes`: `taxonomy.yaml`, `project_facts.yaml`, `typical_systems.yaml` required keys; `typical_systems` system ids must be existing, non-deprecated leaves; subclass and work type ids must exist in `taxonomy.yaml`.

**Step 2: Run** `python -m unittest discover -s tools -p 'test_*.py' -v`. Expected: the new tests FAIL.

**Step 3: Implement** in `tools/check_knowledge.py`: add `deprecated` to
`STATUSES`; the `replaced_by` checks; accept the `document` source form,
checked against `data/eval/profile/manifest.json` when it exists (when the
file is missing, any `document` source is an error under `--strict`); trigger
compilation; `stated_in` and `profile_group` validation; `knowledge/profile/`
loading and checks. Keep every existing check.

**Step 4: Run** the tests and `python tools/check_knowledge.py --strict`.
Expected: all PASS; 0 errors on the real tree.

**Step 5: Update `knowledge/SCHEMA.md`.** Document every section 2.1 field with
a YAML example. Add the presence, provider and assertion question templates
from section 2.2 under "Questions (Jev)", with the authoring rules they
follow (atomic, literal, positive). Add a short "Profile files" section for
`knowledge/profile/`.

**Step 6: Commit** `Extend the knowledge schema for the project profile: deprecation, document sources, determinant triggers and stated_in, and knowledge/profile files.`

**Done when:** the checker tests pass, the strict check passes on the real
tree, and SCHEMA.md documents every new field.

---

### Task P02: Eval manifest and draft answer keys

**Owns:** `data/eval/profile/**` (create).

This is reading work. Read every document in full; do not skim.

**Step 1: Manifest.** Create `data/eval/profile/manifest.json`:

```json
{
  "root_env": "SITEWISE_PROFILE_CORPUS",
  "root_default": "D:/AI Projects/Test Data",
  "projects": [
    {"id": "newham", "documents": ["newham-spec"]},
    {"id": "hale", "documents": ["hale-brief"]},
    {"id": "petersham", "documents": ["petersham-ppr"]},
    {"id": "rutherford", "documents": ["rutherford-bpb"]},
    {"id": "mornington", "documents": ["morn-coastal", "morn-montique", "morn-toussaint"]}
  ],
  "documents": [
    {"id": "newham-spec", "path": "Newham Kit/Specification.docx", "sha256": "<compute>"},
    {"id": "hale-brief", "path": "Hale/Design/Design Brief [P1].pdf", "sha256": "<compute>"},
    {"id": "petersham-ppr", "path": "Petersham/03 ANX Q PPR/ANX Q PPR [E].pdf", "sha256": "<compute>"},
    {"id": "rutherford-bpb", "path": "procurement-industrial-new/BUILDING PERFORMANCE BRIEF [B].pdf", "sha256": "<compute>"},
    {"id": "morn-coastal", "path": "Mornington/Coastal-Builders.pdf", "sha256": "<compute>"},
    {"id": "morn-montique", "path": "Mornington/Montique.pdf", "sha256": "<compute>"},
    {"id": "morn-toussaint", "path": "Mornington/Toussaint.pdf", "sha256": "<compute>"}
  ]
}
```

Compute hashes with `Get-FileHash -Algorithm SHA256`. Never copy the
documents into the repo. Do not record names, phone numbers or email
addresses from them anywhere.

**Step 2: One answer key per project** at
`data/eval/profile/answer-keys/<project>.yaml`:

```yaml
schema_version: 1
reviewed: false              # only the owner changes this
label_source: agent-draft
project: hale
header:
  hdr.building_class: {accept: [industrial], evidence: "hale-brief §3.01: single level warehouse tenancy"}
  hdr.subclass: {accept: [warehouse, logistics_ecommerce], evidence: "..."}
  hdr.work_type: {accept: [extend], evidence: "hale-brief §1.01: proposed extension works (Unit 10)"}
facts:
  fact.contract_form: {accept: [], blank_ok: true, evidence: "not stated in the brief"}
systems:                     # only leaves a document speaks to; others are expected blank
  fire-active.sprinklers: {presence: included, provider: not_stated, evidence: "§9.01 ESFR to AS2118.1"}
  electrical.solar-pv: {presence: not_included, evidence: "§7 Solar PV System: Not applicable"}
  hydraulic.gas: {presence: not_included, evidence: "§10.03 Natural Gas: Not applicable"}
determinants:
  det.storage_height: {accept: [], blank_ok: true, evidence: "Min. top of storage height: TBC"}
  det.bal: {accept: [], blank_ok: true, evidence: "not stated"}
```

Rules: `accept: []` with `blank_ok: true` means any fill is wrong. Record
the assertion where it matters (`assertion: allowance` for Newham's BAL-LOW).
For Mornington record where tenderers differ (provider of solar: Montique
owner, Coastal and Toussaint contractor; electrical supply phases; tank
sizes). Use "storeys" (scale field), not `rise_in_storeys`, unless a document
says "rise in storeys". Seed Newham, Hale and Petersham header fields from
`../clerk/data/eval/jev/answer-keys/profile-setup.yaml`. It covers other
input documents, so re-check each value against the single document listed
here.

**Step 3: README.** `data/eval/profile/README.md`: what each key covers,
that keys are unreviewed drafts, and how scoring treats `blank_ok`.

**Step 4: Commit** `Add the profile evaluation manifest and draft answer keys for Newham, Hale, Petersham, Rutherford and Mornington (unreviewed).`

**Done when:** five keys exist, every entry has evidence with a section or
page reference, and the manifest hashes match the files.

---

### Task P03: Wire the background worker into `serve`

**Owns:** `cmd/sitewise/main.go` (serve wiring only), `internal/jobs/loop.go`,
`internal/jobs/loop_test.go`, `internal/jobs/fulltext.go`,
`internal/jobs/fulltext_test.go`, `data/profile/thresholds.json`.

Today `serve` never runs `jobs.Worker`, so filed documents never get
passages, labels or evidence, and `MinNoul` is zero, which accepts nothing.

**Step 1: Failing tests.**
- `TestLoopRunsJobsForEachOrgWithBacklog`: a fake store reports two orgs with
  backlog; the loop calls `Once` for each until `ClaimJob` reports none, then
  waits for its poll interval. Use the existing `fakeAsk` style from
  `worker_test.go`.
- `TestLoopStopsOnContextCancel`.
- `TestLoopSkipsWhenJevCircuitOpen`: Ask returns `jev.ErrCircuitOpen`; the job
  fails with backoff and the loop does not spin.
- `TestFullTextUsesWholeDocument`: with a three-page PDF fixture from
  `testdata/identity/` (read only, do not edit), the text includes page 3.
- `TestPassageSectionIsNearestHeading`: a numbered or upper-case heading line
  becomes `Section` for the passages after it.

**Step 2: Run** `go test ./internal/jobs/ -v`. Expected: FAIL.

**Step 3: Implement.**
- `loop.go`: `func Run(ctx context.Context, w *Worker, orgs func(ctx context.Context) ([]string, error), poll time.Duration)`.
  It lists orgs with backlog via a store method (add
  `OrgsWithBackgroundJobs` in a **new** file `internal/store/background.go`,
  not `jobs.go`). It runs one job at a time per process, so foreground
  filing keeps its slots. Background Jev calls already cannot take the
  interactive reserve.
- `fulltext.go`: build `Worker.Text` from `identity.Extract` with limits large
  enough for whole documents (pages and bytes caps as constants, with the
  reason in a comment). Do **not** edit `internal/identity`. Store the
  heading line before each passage as its section. To keep sections, extend
  `SplitPassages` with a sibling `SplitSections(text) []Passage` in
  `fulltext.go`; keep the old function's behaviour.
- `data/profile/thresholds.json`: exactly section 2.3.
- `cmd/sitewise/main.go`: load thresholds (fail to start on malformed JSON;
  run with nothing applied when the file is missing, and log it), set
  `MinNoul = label_min_noul` only when `approved_by_owner` is true or when the
  `-profile-provisional` flag is passed, start `jobs.Run` in a goroutine tied
  to the server's shutdown context.

**Step 4: Run** `go test ./internal/jobs/ ./cmd/sitewise/ -v`, then
`go test ./...`. Expected: PASS.

**Step 5: Commit** `Run background passages, labelling and evidence in serve, read whole documents for passages, and add provisional profile thresholds behind an explicit flag.`

**Done when:** a document filed through the dev server gets passages and
(with the flag) labels, visible in the database, and all tests pass.

---

### Task P04: Profile tables, store and isolation tests

**Blocked until the hardening session commits** (migration numbering and
store files). **Owns:** `internal/db/migrations/007_project_profile.sql`,
`internal/store/profile.sql`, `internal/store/profile.go`,
`internal/store/profile_test.go`, `sqlc.yaml` (add the query file only),
generated `internal/db/profile.sql.go`.

**Step 1: Failing tests** in `internal/store/profile_test.go` against
`SITEWISE_TEST_DATABASE_URL`:
- `TestWholePartCreatedOnce`: `EnsureWholePart` twice → one row.
- `TestReplaceDocumentFactsIsIdempotent`: replacing facts for a document
  removes that document's old facts only.
- `TestUserValueWriteUpdatesRowInOneTransaction`: after `SetUserValue`, the
  matching `profile_rows` row has `band = 'user'`, and a forced error rolls
  both back.
- `TestProfileReadIsOneQuery`: `ReadProfile` returns parts, rows and build
  state; count queries with a pgx tracer and assert at most 2.
- `TestProfileOrgIsolation`: every new method with another org's ids returns
  nothing or `ErrNotFound`, never data. Follow `internal/store/isolation_test.go`.

**Step 2: Run** `go test ./internal/store/ -run Profile -v`. Expected: FAIL.

**Step 3: Implement.** Migration from section 2.4 verbatim. Queries in
`internal/store/profile.sql`; add the file to `sqlc.yaml`; run
`sqlc generate`. Methods: `EnsureWholePart`, `CreatePart`, `UpdatePart`,
`ListParts`, `ReplaceDocumentFacts`, `ProjectFacts`, `SetUserValue`,
`UserValues`, `WriteProfileRows` (replace all rows of a project except
`band='user'` rows, in one transaction), `ReadProfile`, `MarkProfileBuilt`,
`SetPendingDocuments`. Every query filters by `org_id`.

**Step 4: Run** the store tests, then `go test ./...`. Expected: PASS.

**Step 5: Commit** `Add project profile tables and store methods with one-query reads and org isolation tests.`

---

### Task P05: One-line register UI

**Blocked until the hardening session commits** `web/src/*`. **Owns:**
`web/src/Register.tsx` (create), `web/src/RegisterRow.tsx` (create),
`web/src/FieldCells.tsx` (create; moved from `DocumentRow.tsx`),
`web/src/DocumentRow.tsx` (delete after moving), `web/src/Project.tsx`
(page chrome only), `web/src/styles.css`, `web/tests/intake.spec.ts`,
`web/tests/register.spec.ts` (create).

Follow design section 6 exactly. Reference look:
`../clerk/frontend/src/components/project/DocumentRepositoryPanel.tsx`,
lines ~1396–1700 (table, colgroup widths, sortable headers). Copy the look,
not the code. Use this repo's tokens in `styles.css`; no new dependencies.

**Step 1: Failing e2e tests** in `web/tests/register.spec.ts`, using the
existing e2e server and recorded Jev from `intake.spec.ts`:
- `one row per document`: after dropping two fixtures, exactly two `tbody tr`
  rows with No., Title, Rev, Date and Disc. cells.
- `no tally, legend or drop panel`: none of the old texts ("to check",
  "stored, not filed", "How to read each box") are visible.
- `expand shows eight editable fields`: Enter on a row shows a detail row with
  Doc no., Rev, Title, Date, Kind, Discipline, Lifecycle and Supersedes;
  Escape closes it and focus returns to the row.
- `correct a field from the expanded row`: edit Rev, save, the row's Rev
  updates and the cell shows the user mark.
- `mark has text`: a row with an amber field exposes an accessible name such
  as "1 field to check".
- `sort by No.`: clicking the header orders rows by number; again reverses.
- `upload progress row`: during upload the row shows the filename and a
  progress element.
- `live state hidden when live`: no "Live" text; when the stream drops,
  "Reconnecting…" appears.

**Step 2: Run** `npm --prefix web run build; npm --prefix web run test:e2e`. Expected: new tests FAIL.

**Step 3: Implement.**
- Move `FieldCell`, `CellEditor`, `Why`, `stateOf` and `REASONS` into
  `FieldCells.tsx` unchanged.
- `Register.tsx`: header ("Documents", count, "Add files" button, live
  warning only when not live), a `<table>` with `<colgroup>` widths from the
  design, sticky `<thead>` with sort buttons (`aria-sort`), and body rows.
  It holds `expandedKey` and `sort`.
- `RegisterRow.tsx`: one `<tr>` per row, about 32 px, with cells truncating
  plus a `title` attribute. The Mark cell renders the worst state among the
  row's fields as an icon plus visually hidden text. The expanded detail is
  a second `<tr>` with one `<td colSpan>` holding the eight `FieldCell`s in a
  compact grid, Download, and the sheet link.
- Drawing sets: the source row renders as a group row with a disclosure
  button; children follow it, open by default.
- `Project.tsx`: remove the tally, drop panel and legend; keep page-wide drop
  handlers and the hidden file input (triggered by "Add files"); render
  `<Register>` in a left column `<section>` sized one third
  (`grid-template-columns: minmax(320px, 1fr) 2fr` at ≥1100 px, one column
  below). Leave the right column as an empty `<section aria-label="Project profile">`
  for P15.
- Update `intake.spec.ts` selectors for the new markup. Keep every existing
  assertion's meaning.

**Step 4: Run** the build and e2e. Expected: PASS. Check by hand at 1440 px,
1100 px and 390 px wide; screenshot each into the PR description.

**Step 5: Commit** `Compress the document list into a one-line sortable register a third of the page wide, with expandable rows for correction; remove the tally, legend and drop panel.`

---

### Task P06: Systems: add, widen, deprecate

**Owns:** `knowledge/clusters/*/systems.yaml`, references to the deprecated id
in `knowledge/clusters/*/{rules,interfaces,failure_modes}.yaml`,
`knowledge/MERGE.md`.

Design section 5 rows S1–S10 are the specification. For each new leaf write
`id`, `parent`, `label`, `describes` (literal: what a passage about it talks
about) and `excludes` (the near-misses and where they belong), `status: draft`
and `sources`. Prefer a seed heading from `../clerk/data/seed/`; search with
`rg -n -i "<term>" ../clerk/data/seed`. Use `{document: <id>, anchor: …}`
only when no seed covers it (bollards, eyewash, salinity).

| Leaf | Cluster file | Must exclude (examples) |
|---|---|---|
| `mechanical.heating-appliances` | services-wet-air | gas pipework and gas flues as plumbing (`hydraulic.gas`), hydronic plant (`mechanical.central-plant`) |
| `envelope.vehicle-doors` | envelope | pedestrian external doors (`windows-doors`), fire shutters (`fire-passive.openings`) |
| `site.loading-docks` | structure | pavements and swept paths (`site.vehicle-access`) |
| `site.impact-protection` | structure | balustrades and fall barriers (`access-egress.barriers`) |
| `site.waste-storage` | structure | trade waste plumbing (`hydraulic.trade-waste`) |
| `site.public-domain` | structure | on-site driveways (`site.vehicle-access`), accessible paths (`access-egress`) |
| `site.signage` | structure | exit signs (`fire-active.emergency-lighting`), accessible signage (`access-egress`) |
| `interiors.ffe` | envelope | built-in joinery (`interiors.joinery`), sanitary fixtures (`hydraulic.sanitary`) |

Widen `hydraulic.cold-water` (private potable sources: rainwater tanks used
for drinking, bores, treatment and filtration). Adjust
`hydraulic.non-potable-water` excludes to point drinking-water tanks to
`cold-water`. Widen `mechanical.local-exhaust` (battery-charging area
ventilation) and `hydraulic.specialist-process` (emergency showers and
eyewash). Deprecate `fire-passive.bushfire-construction` with
`replaced_by: envelope.bushfire-construction`. Move every reference to it,
then merge any boundary text it had that the envelope record lacks, such as
BAL assessment results, into the envelope record.

Keep each parent's children mutually exclusive and at most 40. Re-read
sibling `excludes` so a new leaf is not claimed by an older sibling's
`describes`.

**Verify:** `python tools/check_knowledge.py --strict` → 0 errors. Then
`go test ./internal/knowledge/ ./internal/jobs/` → PASS (the loader must
still load).

**Commit** `Add heating appliances, vehicle doors, loading docks, impact protection, waste storage, public domain, signage and FF&E systems; widen water and exhaust boundaries; deprecate the duplicate bushfire construction system.`

---

### Task P07: Determinants: add, triggers, `stated_in`, compartment derivation

**Owns:** `knowledge/determinants.yaml`, `knowledge/clusters/fire/rules.yaml`
(compartment `derives` only), `knowledge/clusters/determinants/VERIFICATION.md`
(notes only).

1. Add `water_supply_source` (choice: `reticulated`, `rainwater_tank`, `bore`,
   `mixed`), `sewer_connection` (choice: `reticulated`, `onsite_treatment`,
   `pressure_sewer`), `gas_supply` (choice: `natural_gas_main`, `lpg`,
   `none`), `saline_soil` (choice: `stated_true`, `stated_false`,
   `not_stated`), `storage_height` (number, m, `pre_parsed`, scope
   compartment), and `heritage_status` (choice: `none`, `conservation_area`,
   `local_item`, `state_register`). Deprecate `heritage` with
   `replaced_by: heritage_status`.
2. Give `triggers`, `stated_in` and `profile_group` to every determinant the
   profile shows. At minimum: `ncc_class`, `rise_in_storeys`,
   `effective_height`, `floor_area`, `sprinklered`, `state`, `climate_zone`,
   `bal`, `bushfire_prone_land`, `wind_class`, `wind_region`, `site_class`,
   `saline_soil`, `flood_hazard`, `flood_planning_level`, `heritage_status`,
   `acid_sulfate_soils_mapped`, `potentially_contaminated_land`,
   `termite_prone_area`, `noise_affected_site`, `water_supply_source`,
   `sewer_connection`, `gas_supply`, `storage_height`, `ncc_edition`,
   `type_of_construction` (derived, no triggers). Triggers must be narrow:
   `rise_in_storeys` triggers only on `rise in storeys`, never on "5 storey".
3. Every trigger gets a test line in P10's table, so list in your commit
   notes, per trigger, one real line from a research document that should
   match and one that should not.
4. Compartment limits: read `internal/knowledge/evaluate.go` (`Derive`,
   `lookup`). If one rule can give only one table output, add
   `derives` to `rule.ncc.fire-compartment-limits` giving
   `compartment_max_floor_area` and add a sibling rule giving
   `compartment_max_volume`, both from table `compartment_limits` with inputs
   `ncc_class` and `type_of_construction`. Add the two derived determinants
   (`derived: true`, `by: <rule>`). Do not change the table rows.
5. Do not mark anything reviewed.

**Verify:** strict check → 0 errors.

**Commit** `Add water, sewer, gas, saline soil, storage height and heritage status determinants; give profile determinants triggers, stated_in and groups; derive compartment limits from the verified table.`

---

### Task P08: `knowledge/profile/`: Clerk taxonomy, project facts, typical systems

**Owns:** `knowledge/profile/taxonomy.yaml`, `knowledge/profile/project_facts.yaml`,
`knowledge/profile/typical_systems.yaml`.

1. `taxonomy.yaml`: transcribe `../clerk/data/taxonomy/building-classes.json`
   (classes, subclasses with `ncc_class`, scale fields), `work_types`, and
   these universal conditions from `complexity-dimensions.json`: `planning`
   (add option `planning_permit` for VIC and other states), `procurement_route`,
   `access_constraints`, `operational_constraints`, `stakeholder_complexity`,
   `environmental_sensitivity`, plus the industrial overlays `power_source` and
   `hazardous_process`. **Leave out** `bushfire_exposure`, `flood_exposure`,
   `heritage_status`, `contamination_level` and `water_source`; determinants
   own those. Strip cost-uplift text such as `(+10-20%)` from labels. Add
   `unit` and `basis_required: true` to area scale fields (`gfa_sqm`,
   `gla_sqm`, `nla_sqm`, `site_sqm`), and narrow `triggers` to every scale
   field (for example `storeys`: `(\d+|single|double|two|three)[ -]?stor(e)?y`;
   `gla_sqm`: `GLA|gross lettable`). Sources:
   `{seed: <the clerk seed the taxonomy cites>, anchor: …}` where one exists;
   otherwise cite `../clerk/docs/pmp2.0/pmp2.md` headings via a seed copy
   only if the checker allows. If it does not, stop and ask the owner which
   source form to use.
2. `project_facts.yaml`: `consent_number` (pre_parsed, triggers on
   `\b(DA|CDC|SSD)[- /]?\d`, `development consent`), `consent_date`,
   `consent_lapse_date` (pre_parsed dates; code normalises to ISO),
   `contract_form` (choice: `as4902`, `as4000`, `as2124`, `abic`,
   `domestic_hia`, `domestic_mba`, `other`), `contract_basis` (choice:
   `lump_sum`, `cost_plus`, `gmp`, `schedule_of_rates`, `other`),
   `defects_liability_period` (pre_parsed months), `design_life` (pre_parsed
   years). Use the question wording style of section 2.2.
3. `typical_systems.yaml`: for each `(subclass, work_type)` pair present in
   `../clerk/data/taxonomy/typical-work-scopes.json`, map Clerk work-scope
   items to leaf system ids. Example: Clerk `roofing` →
   `envelope.roof-coverings`; `hydraulic_plumbing` → `hydraulic.cold-water`,
   `hydraulic.hot-water`, `hydraulic.sanitary`. Add `house × new` from the
   Newham and Mornington documents, and `warehouse × new|extend` from Hale and
   Rutherford, so a thin brief gets a useful list. When unsure, leave a
   system out; a suggestion the user must untick costs more than one they
   tick.

**Verify:** strict check → 0 errors.

**Commit** `Add the project profile taxonomy from Clerk data, project facts and typical systems per subclass and work type (draft).`

---

### Task P09: Go loader for determinants and `knowledge/profile/`

**Owns:** `internal/knowledge/determinants.go`, `internal/knowledge/profile.go`,
`internal/knowledge/determinants_test.go`, `internal/knowledge/profile_test.go`,
`internal/knowledge/testdata/profile/**`, plus `load.go` changes limited to
calling the new loaders and handling `deprecated`.

The loader does not read `determinants.yaml` today.

**Step 1: Failing tests** with a fixture tree in
`internal/knowledge/testdata/profile/`:
- `TestLoadDeterminantsWithTriggers`: triggers compile as Go `regexp` with
  `(?i)`; a bad pattern fails `Load` with the determinant id in the error.
- `TestDeprecatedSystemsAreHiddenFromChoices`: `Children(parent)` excludes
  deprecated systems; `Resolve(id)` maps a deprecated id to `replaced_by`.
- `TestTaxonomyAndTypicalSystems`: `Typical("house","new")` returns the leaf
  ids; an unknown pair returns nil.
- `TestProfileDeterminantsByGroup`: only determinants with `profile_group`.
- `TestRealKnowledgeLoads`: `Load("../../knowledge")` succeeds (this test
  goes green once P06–P08 merge).

**Step 2: Run** `go test ./internal/knowledge/ -v`. Expected: FAIL.

**Step 3: Implement** types `Determinant` (`ID, Label, Value, Unit,
Extraction, Options, Derived, By, Triggers []*regexp.Regexp, StatedIn,
ProfileGroup, Question, Status, ReplacedBy`), `ProjectFact` (same shape),
`Taxonomy`, and `Catalog` methods: `Determinants()`, `Determinant(id)`,
`ProjectFacts()`, `Taxonomy()`, `Typical(subclass, workType)`, `Resolve(id)`,
`Leaves()` (non-deprecated leaf systems in id order).

**Step 4: Run** `go test ./internal/knowledge/ ./internal/jobs/ -v`. Expected: PASS
(except `TestRealKnowledgeLoads` until P06–P08 merge; mark it with `t.Skip`
only while those are unmerged, and remove the skip in the same PR that
merges them).

**Step 5: Commit** `Load determinants, project facts, taxonomy and typical systems from knowledge, and hide deprecated systems from Jev choices.`

---

### Task P10: Candidate harvesters (code)

**Owns:** `internal/profile/harvest.go`, `internal/profile/harvest_test.go`.

**Step 1: Failing table test** `TestHarvest`. Each case is a real line from
a research document, the expected triggered keys, and the expected
candidates:

| Line (verbatim) | Triggered | Candidates |
|---|---|---|
| `We have allowed for Bush Fire Attack Level Low to all sites.` | `det.bal` | none (choice determinant) |
| `Thermally Broken windows/doors, with BAL 40 compliance` | `det.bal` | none |
| `The size of the fire compartments must remain under 2,000sqm so that Type C construction can be achieved with a FRL of 90/90/90.` | `det.floor_area` | `2,000sqm` → value `2000`, unit `m2` |
| `Construction of a 5 storey development consisting of a residential flat building` | `hdr.scale.storeys` (not `det.rise_in_storeys`) | `5` |
| `Development consent number: DA201500704, and Section 96: DA201500704.01` | `fact.consent_number` | `DA201500704`, `DA201500704.01` |
| `Consent will lapse: 17.10.2023` | `fact.consent_lapse_date` | `17.10.2023` → `2023-10-17` |
| `Design & Construct Contract AS 4902-2000 (Amendment 1)` | `fact.contract_form` | none |
| `Supply and install Aqua Nova Advance Septic Treatment System.` | `det.sewer_connection` | none |
| `Two 45 kg LPG gas bottles, gas bottle connections and gas lines` | `det.gas_supply` | none |
| `All building areas are measured as Gross Lettable Area (GLA)` + a later line `1,920 m2` in the same passage | `hdr.scale.gla_sqm` | `1,920 m2` → `1920`, basis `GLA` |
| `TOTAL GROSS FLOOR 4,013` | none | none (no unit: a known miss, kept as a test) |
| `Min. top of storage height TBC` | `det.storage_height` trigger | no candidates → **no question** |
| `three-storey brick residential apartment building` (a neighbour) | `hdr.scale.storeys` | `3` (Jev must reject it; code does not judge) |
| `Building Type: Single storey` | `hdr.scale.storeys` | `Single storey` → `1` |

**Step 2: Run** `go test ./internal/profile/ -run TestHarvest -v`. Expected: FAIL.

**Step 3: Implement** `func Harvest(p Passage, cat Catalog) Harvested`,
which returns triggered keys and `map[questionID][]Candidate{ID, Value,
Context, Norm, Unit, Basis}`. Number parsing strips thousands separators;
area units are `m2`, `m²`, `sqm`, `square metres`; the basis comes from
`GFA|GLA|NLA|gross floor|gross lettable|net lettable|site area` in the same
passage. Dates `dd.mm.yyyy`, `dd/mm/yyyy` and `d Month yyyy` are normalised
to ISO in `Norm`; `Value` stays verbatim. Cap at 12 candidates per key per
passage, keeping the first ones. A pre_parsed key with zero candidates is not
triggered. Over-find rather than under-find; Jev picks
([pre-parsed extraction](https://docs.typesafe.ai/cookbooks/pre_parsed_value_extraction_cookbook)).

**Step 4: Run.** Expected: PASS. Add a benchmark `BenchmarkHarvest` over a
2,000-rune passage; it must stay under 1 ms per passage on the dev box.

**Step 5: Commit** `Harvest profile candidates and triggers from passages in code, with cases from the research documents.`

---

### Task P11: Profile Jev questions in the two fan-outs

**Owns:** `internal/profile/questions.go`, `internal/profile/questions_test.go`,
`internal/jobs/worker.go` (only `LabelCall`, `EvidenceCall`, the passage
struct and their callers), `internal/jobs/worker_test.go` (new cases only).

**Step 1: Failing tests.**
- `TestLabelCallAddsTriggeredDeterminantsOnly`: a passage with "BAL 40"
  gets `det.bal` and `det.bal.assertion`, and no other `det.*`.
- `TestPreParsedOptionsAreCandidatesPlusNone`: criteria keys are `c1…cN` and
  `none`; `state.candidates["det.floor_area"]` matches.
- `TestHeaderQuestionsOnlyOnRoutedPassages`: ordinal 3 of a `design_brief`
  gets `hdr.*`; ordinal 40 in section "Fire Services" does not; a `drawing`
  never does.
- `TestEvidenceCallAddsPresenceAndProviderForLabelledLeaves`: labels
  `[hydraulic, hydraulic.gas]` → `sys.hydraulic.gas.presence` and
  `.provider`; never for top-level ids or unlabelled leaves.
- `TestNoDeprecatedIdsOffered`.
- `TestWordingMatchesContract`: instructions and criteria equal section 2.2
  exactly for one example of each question type.
- `TestOneCallPerStageStillHolds`: `BackgroundCalls` for N passages still
  produces at most 2N calls.

**Step 2: Run** `go test ./internal/profile/ ./internal/jobs/ -v`. Expected: FAIL.

**Step 3: Implement** builders `LabelQuestions(p, h Harvested, cat)` and
`EvidenceQuestions(p, labels, cat)` returning `map[string]jev.Question` plus
the candidate state, and merge them into `LabelCall` and `EvidenceCall`.
Cite the TypeSafe pages in comments next to each builder. Bump
`questionVersion` to `"knowledge.v2+profile-1"`.

**Step 4: Run.** Expected: PASS.

**Step 5: Commit** `Ask profile questions inside the existing label and evidence fan-outs: triggered determinants with assertion, routed header choices, and presence and provider per labelled leaf.`

---

### Task P12: Reconciliation (pure code)

**Owns:** `internal/profile/types.go`, `internal/profile/reconcile.go`,
`internal/profile/reconcile_test.go`.

**Step 1: Failing table tests** (section 2.5 types). Each name states the rule:
- `TestUserValueWins`: a user value plus contrary amber facts → band `user`,
  the user's value.
- `TestSupersededFactsIgnored`.
- `TestBelowAmberFloorIsBlankButKept`: confidence 0.4 with floor 0.6 → band
  `blank`, value empty, the source still listed.
- `TestNilConfidenceIsBlank`.
- `TestGreenWithheldWhenNull`.
- `TestPresenceConflictIsRed`: one `included`, one `not_included` (both
  above floor, different documents) → `red`, both in `Alternatives`, value empty.
- `TestTendersConsistent` (Mornington): three `commercial` documents, all
  `included` for `electrical.solar-pv` → amber, `Tenders: "consistent"`.
- `TestTendersDiffer` (Mornington): provider `owner` (one tender) vs
  `contractor` (two) → `red`, `Tenders: "differ"`, alternatives ordered by
  support.
- `TestAllowanceShownNotDerived` (Newham): `det.bal` = `BAL-LOW` with
  assertion `allowance` → row amber with assertion `allowance`; a derivation
  that needs `bal` gets no `bal` input.
- `TestDerivationUsesStatedAndUserOnly` (Rutherford): user `ncc_class=7b`,
  user `rise_in_storeys=1`, with the rule and tables marked `reviewed` in a
  fixture catalog → `type_of_construction = C`; compartment max area `2000`,
  volume `12000`. With the real draft statuses → unknown with reason
  `unreviewed`.
- `TestSuggestedNeverOutranksEvidence`: suggested `hydraulic.gas` plus a
  `not_included` fact above floor → `not_included`, amber.
- `TestSuggestedOnly`: no facts → band `suggested`, value `included`.
- `TestPartLabelExactMatch`: a fact with `part_label` equal to a part's label
  goes to that part; otherwise to the whole part.
- `TestTwoValuesOnOnePartIsConflict`.
- `TestNoteIsVerbatimExcerpt`: the note equals a source excerpt and is cut
  at 120 runes on a rune boundary.
- `TestDeterministicOrder`.

**Step 2: Run** `go test ./internal/profile/ -run Reconcile -v`. Expected: FAIL.

**Step 3: Implement** `Reconcile`. Independence rule: documents of kind
`commercial` count once together for support ("consistent"). Every other
document counts once. Support for a value is the number of independent
documents above the amber floor. No I/O, no Jev, no time calls.

**Step 4: Run.** Expected: PASS.

**Step 5: Commit** `Reconcile profile facts in code: user values final, conflicts red, tenders not independent, allowances never derived, suggestions below evidence.`

---

### Task P13: Profile stage job and persistence

**Owns:** `internal/jobs/profile.go`, `internal/jobs/profile_test.go`,
`internal/store/store.go` (add `JobKindProfile` constant only),
`internal/store/background.go` (from P03, add a debounce helper).

**Step 1: Failing tests** (real test DB, fake Jev):
- `TestEvidenceStageWritesFactsAndEnqueuesProfile`: after the evidence stage,
  `profile_facts` hold the document's readings (values verbatim; confidence
  from `Answer.Confidence`; noul probability is never stored as confidence)
  and one `profile` job exists for the project.
- `TestProfileJobDebounced`: three documents finishing evidence → one
  pending profile job.
- `TestProfileJobWritesRowsAndEmitsEvent`: the job reconciles and writes
  `profile_rows`, sets `profile_builds.built_at`, and emits an SSE `profile`
  event with the project id.
- `TestUserRowsSurviveRebuild`.
- `TestMissingThresholdsAppliesNothing`: facts are stored and rows are
  `blank`; `profile_builds` records `thresholds_version = ''`.
- `TestJevDownKeepsLastRows`: Ask fails; the previous rows are unchanged and
  the job retries with backoff.

**Step 2: Run** `go test ./internal/jobs/ -run Profile -v`. Expected: FAIL.

**Step 3: Implement.** In the evidence stage, after `SetPassageEvidence`,
convert profile answers to facts (`ReplaceDocumentFacts` once per document,
at the end of the stage), then enqueue a debounced `profile` job. Do the same
for profile answers from the label stage (keep them in memory across the
document's passages, or store them in a second replace call; both stages
replace only their own `question_id` prefixes). The profile job loads parts,
facts, user values, supersession and thresholds, then calls
`profile.Reconcile` and `WriteProfileRows`. Record per document: pages,
passages, Jev calls, elapsed time and errors, in the job's event payload.

**Step 4: Run** `go test ./...`. Expected: PASS.

**Step 5: Commit** `Store profile readings per document and rebuild each project's profile in a debounced background job that emits a profile event.`

---

### Task P14: Profile HTTP API, SSE event, speed budgets

**Blocked until the hardening session commits** `projects.go` and
`server.go`. **Owns:** `internal/httpapi/profile.go`,
`internal/httpapi/profile_test.go`, the route and `routePaths` entries in
`internal/httpapi/projects.go` (a few lines only), `internal/events` (only if
a new event kind needs registering), `web/src/api.ts` (`EVENT_KINDS` only),
`bench/budgets.json`, `cmd/intake-bench/**` (add samples for the two new
paths).

**Step 1: Failing tests** (`httptest`, real test DB):
- `TestGetProfileShape`: the JSON matches section 2.6, including
  `pending_documents`, `thresholds.provisional` and `stated_in`.
- `TestPutValidatesAtBoundary`: unknown key → 422; option not allowed → 422;
  note over 120 → 422; another org's project → 404.
- `TestPutSubclassRewritesSuggestions`: PUT `hdr.subclass=house` with
  `hdr.work_type=new` → the response and a following GET show `suggested`
  rows for `typical_systems` of (house, new).
- `TestPartsCreateAndRename`; no DELETE route exists.
- `TestProfileRoutesAreTimed`: the speed recorder sees `project_profile_read`
  and `profile_edit`.
- `TestOriginCheckOnWrites`: same origin rules as existing writes.

**Step 2: Run** `go test ./internal/httpapi/ -run Profile -v`. Expected: FAIL.

**Step 3: Implement** handlers using `memberSession` and `readJSON` like
`projects.go`. Add the two budget paths to `bench/budgets.json` and sample
them in `cmd/intake-bench` (20 warm samples each against a seeded project
with 200 profile rows).

**Step 4: Run** `go test ./...` and
`go run ./cmd/intake-bench -manifest data/eval/intake/manifest.json -budgets bench/budgets.json`.
Expected: PASS within budget.

**Step 5: Commit** `Serve the project profile and user edits from precomputed rows within 50/150 ms budgets, with parts and a profile event.`

---

### Task P15: Profile UI and two-column layout

**Owns:** `web/src/Profile.tsx`, `web/src/ProfileHeader.tsx`,
`web/src/SystemsChecklist.tsx`, `web/src/CompliancePanel.tsx`,
`web/src/profileApi.ts` (create all), `web/src/Project.tsx` (mount the
profile in the right column, handle the `profile` event, nothing else),
`web/src/styles.css` (profile styles), `web/src/icons.tsx` (add Suggested
and Conflict icons), `web/tests/profile.spec.ts` (create),
`web/tests/e2eserver/**` (recorded profile answers only).

Design sections 3 and 6 are the specification. Bands reuse the register's
icons: from the document or confirmed, check, not set, not checked, set by
you. Add **Suggested** (outline icon, text "Typical for <subclass>") and
**Conflict** (red icon, text "Sources differ"). Colour never carries meaning
alone.

**Step 1: Failing e2e tests:**
- `profile shows three sections and a parts bar`.
- `choose class, subclass, work type → suggested systems appear` (thin brief).
- `tick a suggested system → it becomes set by you` and survives reload.
- `untick an evidenced system → set by you; the source is still listed`.
- `conflict shows both values with their documents; choosing one resolves it`.
- `source link jumps to and expands the register row`.
- `compliance row with nothing stated shows "Usually in: …"`.
- `allowance is labelled`: Newham-style BAL-LOW shows "Allowance".
- `keyboard`: the tri-state presence control is a radio group (arrow keys)
  and focus stays put when a `profile` event refreshes data.
- `provisional thresholds note` appears when `thresholds.provisional`.

**Step 2: Run** the build and e2e. Expected: FAIL.

**Step 3: Implement.**
- `profileApi.ts`: `getProfile`, `putValue`, `createPart`, `updatePart`,
  typed from section 2.6.
- `Profile.tsx`: fetches, keeps state, refetches on the `profile` event,
  and shows the status line (built time, "N documents still being read",
  provisional note).
- `ProfileHeader.tsx`: class, subclass (filtered by class), work type and
  scale fields (number inputs with unit and basis), conditions, project
  facts.
- `SystemsChecklist.tsx`: groups by top-level system with collapsible
  headings. Each leaf row has a tri-state radio group, a provider select, a
  note input (maxLength 120), the band mark, and a sources popover listing
  document title and excerpt with a "Show in register" action. A "Show all
  systems" toggle reveals the rest.
- `CompliancePanel.tsx`: groups Classification, Site, Services and Fire; for
  each row, the value editor (select or number), assertion label, band,
  sources, "Usually in: …" when empty, and for derived rows the rule state
  ("Awaiting owner review", "Pending: table not verified").
- Parts bar: list, add (label, kind, NCC class), rename.

**Step 4: Run** the build and e2e. Expected: PASS. Check by hand at 1440,
1100 and 390 px; screenshots in the PR.

**Step 5: Commit** `Show the project profile beside the register: header, parts, systems checklist and compliance, editable in place with provenance.`

---

### Task P16: Profile eval harness (replay and live)

**Owns:** `cmd/profile-eval/**` (create), `data/eval/profile/results/**`,
`tools/check.ps1` (one added step).

**Step 1: Failing tests** in `cmd/profile-eval/main_test.go`: scoring of
`accept`, `blank_ok` and presence; `-replay` with a recorded file
reproduces identical scores; a missing recording fails closed; no green
may be wrong (exit code 1 if so).

**Step 2: Implement.**
- `-live` runs the real pipeline over the manifest documents with
  `SITEWISE_JEV_API_KEY` and records every Jev request and response to
  `data/eval/profile/results/recording-<date>.json` (requests contain
  document text, so this file is git-ignored).
- `-replay` replays it.
- Report per project: correct without user action, wrong and applied,
  blank, presence precision and recall, and the share against the 80%
  target. Also report elapsed time, calls and passages per document.
- Add `go run ./cmd/profile-eval -replay` to `tools/check.ps1` after
  intake-eval.

**Step 3: Run** `go test ./cmd/profile-eval/ -v`. Expected: PASS.

**Step 4: Commit** `Add the profile evaluation harness with live recording, deterministic replay and a no-wrong-green gate.`

---

### Task P17: Live run, provisional floors, evidence write-up

**Needs the owner.** After O1 has reviewed the answer keys:

1. `go run ./cmd/profile-eval -live`, then `-replay` to confirm determinism.
2. Write `docs/evidence/2026-10-XX-profile-first-live/README.md`: scores per
   project and field, every wrong-applied value with its passage, latency
   per document, and Jev call counts. Never round up; list misses.
3. Propose amber floors per question shape from the run. Do not set green
   unless a shape has at least 30 keyed answers
   ([confidence routing](https://docs.typesafe.ai/patterns/confidence-routing)).
   The owner approves by setting `approved_by_owner: true`.
4. Fix only what the evidence shows: triggers (P07), harvesters (P10),
   wording (P11). Each fix gets a failing test from the real line first.

**Commit** the evidence and any approved threshold change separately.

---

## 4. Definition of done (whole feature)

- `tools/check.ps1` passes, including the profile replay and both new speed
  paths.
- Jev is the only AI. Search the diff for `openai`, `anthropic`, `claude`,
  `gpt`, `llm` and `embedding`: zero hits in code.
- The profile is filled for the five projects at or near 80% correct without
  user action, with no wrong green. The evidence write-up says where it falls
  short.
- The register is one line per document at one third of the page, with every
  previous capability reachable (correct, retry, download, sheets).
- The owner has reviewed the answer keys and the thresholds.

# Profile scope, document reading selection and deletion: implementation plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Build `docs/design/2026-10-03-profile-scope.md`: a per-document
profile reading setting with bulk selection, permanent bulk deletion, and a
scope-led profile whose systems and compliance rows come from the schema.

**Architecture:** Three changes, committed separately. (1) A `profile_read`
column on documents plus a kind rule in `data/profile/reading.json`; job
queueing filters in SQL, reconciliation filters in `profile.Build`. (2) An
org-scoped transactional delete that removes documents (everything cascades),
then unreferenced file rows, then unreferenced blobs under the blob store's
publish lock. (3) Scope = defaults from `knowledge/profile/scope_defaults.yaml`
∪ document-evidenced systems ∪ user `scope.*` values − user removals, stored as
`scope.<leaf>` profile rows at build time; compliance relevance is computed in
code from rules (`systems`, `applies_when`, `derives`) and the new determinant
`systems` field.

**Tech stack:** Go 1.27, PostgreSQL 17 (pgx, sqlc), React + Vite, Playwright,
Python knowledge checker. No new dependency.

**Environment:** worktree `D:\AI Projects\sitewise-scope`, branch
`profile/scope`. Private test database on port 5434
(`SITEWISE_TEST_DATABASE_URL=postgres://sitewise@127.0.0.1:5434/sitewise_test?sslmode=disable`)
so tests do not collide with the shared 5433. Go and sqlc from
`D:\AI Projects\sitewise\.tools`. Run Go tests with `-p 1`.

**Not changed:** Jev question wording and `profile.QuestionVersion` (labels
only), filing, intake recordings.

---

## Change 1: profile reading setting (lane A, register lane B)

### Task 1.1: Read policy in the profile package

**Files:** create `data/profile/reading.json`, `internal/profile/reading.go`,
`internal/profile/reading_test.go`.

`reading.json`:

```json
{
  "version": 1,
  "note": "Kinds the profile reads when a document's setting is automatic. Drawings, photos, correspondence and unclassified documents are skipped: on the dev corpus drawings were ~60% of Jev calls and the source of every wrong header value on Spec Home (docs/design/2026-10-03-profile-scope.md).",
  "read_kinds": ["design_brief", "specification", "report", "contract", "commercial", "certificate", "statutory_instrument", "schedule"]
}
```

`ReadPolicy{kinds map[string]bool}`; `LoadReadPolicy(path) (ReadPolicy, error)`
rejects an empty list and unknown JSON; `Reads(kind, setting string) bool`:
`read` → true, `skip` → false, otherwise `kinds[kind]`; `Kinds() []string`
sorted; `Loaded() bool`. `ValidSetting(s)` for `auto|read|skip`.

Tests: auto drawing false, auto specification true, read drawing true, skip
specification false, unknown kind false, load rejects empty list.

### Task 1.2: Reconciliation ignores unread documents

**Files:** modify `internal/profile/reconcile.go` (`Fact.ReadSetting`,
`Input.Read ReadPolicy`), `internal/profile/build.go`; test in
`internal/profile/reconcile_test.go`.

`Build` drops facts whose document the policy does not read, before
reconciling. A zero policy keeps every fact (tools and tests that predate the
setting). Facts with `DecidedBy == "user"` are not document facts and are
never dropped.

Test: a drawing fact and a specification fact for the same key; with the
default policy only the specification value shows; with the drawing set to
`read` both count (conflict).

### Task 1.3: Migration and store

**Files:** create `internal/db/migrations/010_profile_reading.sql`; modify
`internal/store/views.sql` (+ `profile_read` in both document view queries),
regenerate sqlc, `internal/store/views.go` (`DocumentView.ProfileRead`),
`internal/store/profile.go` (`readSnapshot` selects `d.profile_read`;
`RequestProfileRead(ctx, org, project, readKinds)`; `ReadProfile(ctx, org,
project, readKinds)` counts), create `internal/store/reading.go`
(`SetProfileReading`), test `internal/store/reading_test.go`.

```sql
ALTER TABLE documents ADD COLUMN profile_read text NOT NULL DEFAULT 'auto'
  CHECK (profile_read IN ('auto', 'read', 'skip'));
```

Readable predicate used by every reading query (`$k` = read kinds):
`d.profile_read = 'read' OR (d.profile_read = 'auto' AND COALESCE((SELECT
value FROM decisions WHERE org_id = d.org_id AND document_id = d.id AND field
= 'kind'), '') = ANY($k::text[]))`.

- `RequestProfileRead`: the stale-source requeue still requeues `full_text`
  for every stale document, but `label`/`evidence` only for readable ones;
  failed-job retry, restamp and label insert are restricted to readable
  documents.
- `SetProfileReading(ctx, org, project, ids, setting, readKinds) (int, error)`:
  one transaction; expands drawing-set sources to their sheets
  (`drawing_sheets.source_id`); every id must be in the project or the call
  returns `ErrNotFound` and changes nothing; deletes `queued` and `failed`
  `label`/`evidence` jobs of documents that are now unreadable (never
  `leased` or `done`; `done` results stay cached for re-inclusion).
- `ReadProfile` adds `ReadDocuments`, `SkippedDocuments`, `SkippedKind` (most
  common kind among skipped); `UnreadDocuments` counts readable documents
  only.

Tests: setting validation; wrong org → not found and nothing changed; mixed
valid/foreign ids → not found, nothing changed; sheet expansion; queued label
job removed on skip, done job kept; `RequestProfileRead` queues a
specification but not a drawing, and queues the drawing once set to `read`;
counts.

### Task 1.4: API and wiring

**Files:** modify `internal/httpapi/projects.go` (route + timing),
`internal/httpapi/profile.go` (body fields, rebuild helper),
`internal/httpapi/documents.go` (`/catalog` adds `profile_read_kinds`),
`internal/httpapi/server.go` and `Deps` (`ProfileReading profile.ReadPolicy`),
`internal/jobs/worker.go` (`Worker.Reading` into `profile.Input`),
`cmd/sitewise/main.go` (load `data/profile/reading.json`, fail start if
invalid); tests in `internal/httpapi/profile_test.go`.

Route: `PUT /projects/{id}/documents/profile-read`, body
`{"document_ids": [...], "setting": "auto|read|skip"}`, origin checked, at
most 1,000 ids, returns the profile JSON (rebuild is code only), timed as
`profile_edit`. Bad setting or empty ids → 400; foreign or unknown ids → 404.

Profile JSON adds `read_documents`, `skipped_documents`, `skipped_kind`.

### Task 1.5: Register selection, Profile column, bulk bar

**Files:** modify `web/src/api.ts` (`Doc.profile_read`, `Catalog.profile_read_kinds`,
`api.setProfileReading`), `web/src/Register.tsx`, `web/src/Project.tsx`
(apply setting results), `web/src/Profile.tsx` (summary line),
`web/src/profileApi.ts`, `web/src/styles.css`; e2e in
`web/tests/profile.spec.ts` (and the e2e server if it needs the route).

- A narrow checkbox column on the left selects rows. Click a checkbox selects
  it and sets the anchor; Shift-click selects the range from the anchor;
  Ctrl/Cmd-click on a row toggles its selection without opening it; a header
  checkbox selects all visible document rows. Plain row click still opens the
  detail (the register's existing behaviour).
- Profile column: ✓ (read) or – (not read), with a dot and "your choice" title
  when `profile_read` is not `auto`. Clicking it toggles that row between read
  and skip.
- With rows selected a bulk bar shows "N selected · Read for profile · Don't
  read · Reset to automatic · Delete · Clear".
- Profile summary: "Reading 12 of 48 documents · 36 not read (mostly
  Drawing)" with a "Show" link that selects nothing but filters the register
  to not-read rows (a `filter` prop).

### Task 1.6: Bench and commit

Add a timed `PUT …/documents/profile-read` sample under `profile_edit` in
`cmd/intake-bench` `profileReadEdit`. Run all checks; commit.

---

## Change 2: delete documents (lane C)

### Task 2.1: Blob removal under the publish lock

**Files:** modify `internal/files/store.go`; test `internal/files/store_test.go`.

- `Put` of bytes already stored touches the blob's modification time (under
  the publish lock), so a blob being re-used by an upload is "young".
- `Remove(sum []byte, minAge time.Duration, referenced func() (bool, error)) (bool, error)`:
  under the same lock; keeps the blob if it is younger than `minAge` or
  `referenced` says true; missing blob is not an error.

Tests: removes an old unreferenced blob; keeps a referenced one; keeps a young
one; a `Put` of existing bytes makes it young.

### Task 2.2: Store delete

**Files:** create `internal/store/delete.go`, test `internal/store/delete_test.go`.

`DeleteDocuments(ctx, org, project, ids, userID) (DeleteResult, error)` in one
transaction under the project's profile advisory lock:

1. Expand drawing-set sources to their sheets.
2. Every requested id must belong to the org and project, else `ErrNotFound`.
3. Collect `file_id`s; `DELETE FROM documents` (passages, decisions, jobs,
   facts, sources, supersessions, sheets cascade; a superseded prior revision
   becomes current because its supersession row goes).
4. Delete file rows in the project that no remaining document references;
   return their hashes.
5. Append one `deleted` event per document (payload `{document_id, project_id,
   user_id}`).

`BlobReferenced(ctx, sum) (bool, error)`: any file row in any org with that
hash (the blob directory is shared).

Tests: delete one; delete a set deletes its sheets; deleting a sheet keeps
the set; shared file row kept while another document uses it; hash returned
only when the last file row goes; superseded prior becomes current; foreign
org id → not found and nothing deleted; mixed ids → nothing deleted; events
written.

### Task 2.3: API

**Files:** modify `internal/httpapi/projects.go`, `internal/httpapi/documents.go`
(handler), `internal/httpapi/speed.go` (`pathDocumentDelete =
"document_delete"`), `bench/budgets.json` (`document_delete` 500/1,000 ms),
`cmd/intake-bench/main.go` (time deletes of bench documents at the end);
tests in `internal/httpapi/server_test.go` or a new `delete_test.go`.

`POST /projects/{id}/documents/delete`, body `{"document_ids": [...]}`,
origin checked, member session, at most 200 ids. After commit: remove blobs
(min age 2 minutes; a younger blob is left as an orphan for a later sweep),
rebuild the profile in code, return `{"deleted": [ids]}`. Add `"deleted"` to
the event kinds the SSE stream forwards.

### Task 2.4: Register delete

**Files:** `web/src/Register.tsx`, `web/src/Project.tsx` (`removed` action,
`deleted` event), `web/src/api.ts`, `web/src/ConfirmDialog.tsx` (native
`<dialog>`), styles; e2e.

- Bin column at the far right; bin in its header deletes the selection
  (disabled with no selection). The bulk bar's Delete opens the same dialog.
- Dialog: "Delete 12 documents, including A-101 Ground Floor Plan, A-102 …?
  This can't be undone." Buttons: Cancel (default focus), Delete.
- After success: rows leave, selection clears, announcement "12 documents
  deleted."

Commit change 2.

---

## Change 3: scope and relevance from the schema (lane A, labels B)

### Task 3.1: Schema data

**Files:** modify `knowledge/profile/taxonomy.yaml` (refurb label
"Refurbishment / fit-out / upgrade"), create
`knowledge/profile/scope_defaults.yaml` (replaces `typical_systems.yaml`: the
seven lists move into `classes:`; add `categories:` lists for every building
category; `presets: [typical_fit_out]`; `empty_work_types: [refurb,
remediation, advisory]`), delete `knowledge/profile/typical_systems.yaml`,
modify `knowledge/determinants.yaml` (`systems:` on profile determinants),
`knowledge/SCHEMA.md` (document both).

Lookup: `new`, `extend` → class list for that work type, else class `new`
list, else category list; empty work types → none.

### Task 3.2: Knowledge loader and checker

**Files:** `internal/knowledge/load.go` (`Rule.Systems`, `Rule.AppliesWhen any`),
`internal/knowledge/profile.go` (`Determinant.Systems`, scope defaults loader
replacing typical, `ScopeDefaults(category, class, work)`, `Preset(id)`),
create `internal/knowledge/relevance.go` + test; `tools/check_knowledge.py`
(validate `scope_defaults.yaml` ids, determinant `systems` refs, warning
"profile determinant is read by no rule and has no systems"), its test.

`Relevant(scope []string, values map[string]string) Relevance` with
`Determinants map[string]bool`, `Rules []string`. Rule relevant when one of
its `systems` covers or is covered by a scope system, and its `applies_when`
is not false. Predicate evaluation is three-valued (true/false/unknown):
`any_of`, `eq`, `is`, `gt`, `gte`, `lt`, `lte` over `values`;
`system_present` over scope; `all`, `any`, `not`. Missing value → unknown.
Determinants from relevant rules' predicate `det` references and `derives`
inputs/gives, plus determinants whose `systems` touch scope, plus core
(`state`, `ncc_class`, `ncc_edition`, `compliance_pathway` if present).

Tests: sprinkler-pump scope (industrial; `fire-active.sprinklers` only) shows
`storage_height`, not `bal`, `termite_prone_area`, `site_class`; fit-out
scope; new house shows `site_class`, `bal`; a false predicate drops a rule;
an unknown keeps it.

### Task 3.3: Scope in the profile build

**Files:** `internal/profile/scope.go` + test, `internal/profile/build.go`,
`internal/profile/reconcile.go` (suggested from scope defaults).

Build: reconcile → header values → defaults (`category`, `class`, `work`) →
scope = defaults ∪ document-included leaves (whole-part presence `included`
in amber/green/user) ∪ user `scope.<leaf> = in` − user `scope.<leaf> = out`.
Emit `scope.<leaf>` rows (`value` in, band `default|document|user`) for every
in-scope leaf, and for user-removed leaves that a document includes a row
with value `out`, band `user` and a note naming the source. Suggested
presence comes only from defaults that are in scope.

### Task 3.4: API

**Files:** `internal/httpapi/profile.go`: header labels "Building category",
"Building class", "Work type"; compliance row label for `ncc_class` is "NCC
class"; systems rows get `in_scope`, `scope_origin`, `scope_note`;
`shown_by_default` = in scope; compliance fields get `relevant`; profile JSON
gets `presets`. New route `PUT /projects/{id}/profile/scope`, body
`{"systems": {"<leaf>": "in"|"out"|null}}` (null removes the user choice),
validated against catalog leaves, one transaction, rebuild, timed as
`profile_edit`. `putProfileValue` rejects `scope.` keys (one write path).

### Task 3.5: Profile UI

**Files:** `web/src/Profile.tsx`, `web/src/profileApi.ts`, `web/src/ScopePicker.tsx`, styles; e2e.

Scope of works section between Project and Systems: one line per top-level
system with "n of m", a group checkbox (all leaves in or out), expandable
leaf checkboxes with origin marks (Typical / From documents / You), and the
"Typical fit-out" preset button. Systems list shows in-scope rows; the
"Show all systems" toggle stays. Compliance hides non-relevant rows unless
"Show everything" is ticked; a line says "Showing 9 of 28 compliance items
for this scope".

### Task 3.6: Bench, checks, evidence, commit

Time a scope PUT under `profile_edit`. Run every check. Record timings in
`docs/evidence/2026-10-03-profile-scope.md`. Commit.

---

## Verification (end)

- `python tools/check_knowledge.py --strict`
- `go test -p 1 ./...` against port 5434
- `npm --prefix web run build`, profile and intake Playwright specs
- Budgets: profile read, profile edit, delete measured locally (HTTP tests
  with the real store) and recorded; the full replay bench needs refreshed
  intake recordings (known, not caused here).

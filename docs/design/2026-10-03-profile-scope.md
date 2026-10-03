# Profile scope, document selection and deletion

Date: 3 October 2026. Owner: Benny Clifton. Status: agreed in design session;
not built. Extends `docs/design/2026-10-03-project-profile.md`.

## 1. Why

The profile shows every system (about 150 leaves) and every determinant for
every project. A sprinkler pump replacement in a warehouse sees BAL, termites,
salinity and site class; a commercial fit-out sees substructure and site
works. That is cognitive overload, and none of it comes from the schema.

Profile reading is also slow and noisy. "Update project profile" reads every
document with text. On the dev database, drawings were 6,510 of about 10,500
label calls and 1,740 of about 3,300 evidence calls, yet only 2.4% of drawing
passages mapped to anything (specifications 17%, the Hale brief 45%). Every
wrong applied header value on Spec Home came from a drawing:
`garage_spaces = 230` (a wall dimension), `bedrooms = 1` ("MIN. 1 BEDROOM"),
`beds = 3` (a room label).

Outcome: the profile shows what the works touch, read from the documents the
user chooses. Serves answer trust and time-to-decision.

## 2. Names and work types

Labels only; stored ids are unchanged.

| Was | Now | Examples |
|---|---|---|
| Building class | Building category | Residential, Commercial, Industrial, Institution |
| Building type / subclass | Building class | House, Apartments, Warehouse, Office |
| Work type | Work type | |

The compliance row for the NCC classification is always labelled "NCC class",
never "Class". "Division" (ABS) was considered and rejected: the taxonomy is
Clerk's, not the ABS codes, so the name would claim an alignment we do not
have. ABS codes can be added later as a field.

Work types in `knowledge/profile/taxonomy.yaml`:

| id | Label |
|---|---|
| `new` | New build |
| `extend` | Extension / addition |
| `refurb` | Refurbishment / fit-out / upgrade |
| `remediation` | Remediation / rectification |
| `advisory` | Advisory services |

## 3. Scope of works

A project holds a set of **in-scope systems**: the systems the works touch.
The checklist shows only these; compliance follows from them (section 4).
"Show everything" remains one click away.

**Defaults** come from `knowledge/profile/scope_defaults.yaml`, which replaces
`typical_systems.yaml` (its seven lists carry over):

- `new`, `extend`: the building class's list; a class without one uses its
  building category's list, so every class has defaults from about five
  category lists.
- `refurb`: empty. The scope picker offers a one-click "Typical fit-out"
  preset (interiors, mechanical, electrical, comms-security, fire services,
  hydraulic fixtures), defined in the same file.
- `remediation`, `advisory`: empty.

**Picker.** Grouped by the 13 top-level systems; tick a group or open it and
tick leaves.

**Stored scope.** A per-project table: `org_id`, `project_id`, `system_id`,
`state` (`in`, `out`), `origin` (`default`, `user`, `document`).

- Changing building class or work type re-applies `default` rows only.
- An included document adds a system as `document` (amber, with its source).
  Documents never remove a system.
- A user choice is final. A document that says a user-removed system is
  included shows a notice ("Spec §4.2 says included"), not a change.
- A deprecated system id is shown under its `replaced_by`.

## 4. Relevance from the schema

Code computes relevance at profile build time and stores it with the rows.
Reads do no extra work.

1. **Rules.** A rule is relevant when its `systems` touches an in-scope
   system; a top-level id such as `fire-passive` matches any in-scope leaf
   under it. Its `applies_when` is evaluated against values the profile
   holds: a false predicate drops the rule; an unknown value keeps it. Silence
   never hides a row.
2. **Determinants.** A profile determinant is relevant when a relevant rule
   reads it (`applies_when` `det:` references, `derives.inputs`,
   `derives.gives`) **or** one of its own `systems` is in scope.
3. **Core rows** always show: `state`, `ncc_class`, `ncc_edition`,
   `compliance_pathway`.

**Schema change.** Profile determinants gain `systems: [system ids]`. Rules
alone cannot carry relevance: on 3 October 16 of the 28 profile determinants
were read by no rule (`acid_sulfate_soils_mapped`, `effective_height`,
`flood_hazard`, `flood_planning_level`, `gas_supply`, `heritage_status`,
`ncc_edition`, `potentially_contaminated_land`, `saline_soil`,
`sewer_connection`, `site_class`, `storage_height`, `termite_prone_area`,
`water_supply_source`, `wind_class`, `wind_region`). Examples:
`site_class: [substructure]`, `storage_height: [fire-active.sprinklers]`,
`gas_supply: [hydraulic]`.

**Checker.** `tools/check_knowledge.py` warns on a profile determinant that
no rule reads and that has no `systems`: it can never become relevant. This
makes a wrong profile a fixable schema record.

**Reading is unchanged.** Jev still asks about every system in included
documents, because that is how documents add systems to scope. Narrowing
reading to scope is out of scope for now.

## 5. Choosing which documents the profile reads

Each document has a profile reading setting: `auto`, `read`, `skip`.

- `auto` follows a kind rule in data (`data/profile/reading.json`): read
  `design_brief`, `specification`, `report`, `contract`, `commercial`,
  `certificate`, `statutory_instrument`, `schedule`; skip `drawing`, `photo`,
  `correspondence`, `unknown`. Code routes; no Jev call decides it
  ([intent routing](https://docs.typesafe.ai/patterns/intent-routing)).
- `read` and `skip` are user values and final. A kind correction changes
  only `auto` documents.

Filing still extracts text and splits passages for every document. The
setting controls Jev reading only, so turning a document on needs no
re-extraction.

**Update project profile** queues reading only for documents that resolve to
read and have not been read at the current version.

**Turning a document off** removes its facts from the profile immediately.
Its cached Jev results are kept (`passage_calls`), and the rebuild is code
only, so turning it back on restores them with no Jev calls.

**Register.** A narrow Profile column (tick or dash; a dot marks a user
override). Click selects, Shift-click selects a range, Ctrl-click toggles one
row, a header checkbox selects all visible rows. With rows selected, a bulk
bar offers Read for profile, Don't read, Reset to automatic, Delete. A
drawing-set group row applies to all its sheets. The profile shows one line,
for example "Reading 12 of 48 documents · 36 not read (mostly drawings)",
linking to the register filtered to them.

## 6. Deleting documents

A bin column at the far right of the register deletes its row; a bin in the
header deletes the selection. Both, and the bulk bar's Delete, open one
confirmation naming the count and the first titles ("Delete 12 documents,
including A-101 Ground Floor Plan…? This can't be undone.").

Deletion is permanent and org-scoped, in one transaction:

- removes the document, its passages, readings, facts, jobs and decisions;
- removes the stored file only when no other document in the org has the
  same content hash;
- a deleted drawing-set source deletes its sheets; a deleted sheet deletes
  only itself;
- a revision the deleted document superseded becomes current again;
- records an event: who, what, when;
- rebuilds the profile in code.

## 7. Speed budgets

Added to `bench/budgets.json`; the build fails when broken.

| Path | p50 | p90 |
|---|---|---|
| `project_profile_read` (unchanged) | 50 ms | 150 ms |
| `profile_edit`, including a scope change | 50 ms | 150 ms |
| `profile_rebuild`, code only, Spec Home | 100 ms | 300 ms |
| `profile_reading_setting`, bulk, up to 200 documents | 50 ms | 150 ms |
| `document_delete`, bulk, up to 50 documents | 500 ms | 1,000 ms |

Filing keeps p50 ≤ 1 s and p90 ≤ 2 s and is not touched.

## 8. Failure handling

- Bulk setting and bulk delete are all or nothing.
- Another org's document id, or an already deleted one, is not found; nothing
  changes.
- A background job on a deleted document fails its writes and is dropped,
  not retried.
- A missing `scope_defaults.yaml` entry gives an empty default scope, never
  an error.

## 9. Testing

- Relevance (Go, fixed scopes): industrial sprinkler pump replacement
  (sprinklers only; no BAL, termite or site class rows), commercial fit-out,
  new house. Predicate false drops a rule; unknown keeps it.
- Store: user override beats `auto`; a kind correction keeps it; delete with
  a shared content hash keeps the file; superseded revision restored; org
  isolation for setting and delete.
- Knowledge checker: the new warning.
- Playwright: range and toggle selection, bulk setting, bulk delete with
  confirmation.

## 10. Existing data

Every document starts as `auto`. Drawings already read leave the profile at
once (Spec Home loses `garage_spaces = 230` and the bedroom counts). Each
project's scope is seeded from its defaults plus the systems its included
documents already evidence.

## 11. Delivery

Three changes, each its own lane:

1. Reading setting, kind rule, register selection and bulk bar (A).
2. Deletion (C: it touches stored files).
3. Scope, relevance, determinant `systems`, checker warning, renames and the
   picker (A; labels and picker B).

## 12. Out of scope

Narrowing Jev reading to in-scope systems; ABS codes; class-specific default
lists beyond the existing seven; the profile answer-key scorer (recommended
separately); a recently-deleted view or undo.

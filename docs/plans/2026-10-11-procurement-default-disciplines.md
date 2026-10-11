# Procurement: every discipline the project needs, listed by default

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

Status: plan only, nothing built.

**Goal:** A user opens Procurement and the table already lists every
consultant the project needs (each service separately, plus Vertical
Transport and Fire Services when the scope has lifts or sprinklers). The PM
removes the lines they don't need and adds any that are missing. They never
build the list from scratch.

**Why:** It cuts time-to-decision. Today a fit-out with an unread building
class shows no consultants, and a commercial job shows one "Services
Engineer" row, which no RFP is written for. Every service the PM needs has
to be added by hand.

**Lane:** A, product core: this changes what the table proposes. Slice 1
(the shield) is lane B. These must still pass: the existing procurement,
package and proposal tests, the RFP golden and answer-key tests, the
org-isolation sweeps (`internal/httpapi/access_sweep_test.go`), and the
`packages_read` budget (p50 100 ms / p90 250 ms).

**Architecture:** Suggestions stay transient and computed from data, as
they are today (`procurement.SuggestedPackages`). Nothing calls Jev and
nothing new is stored, except a dismissal when the user removes a
suggested row. Three inputs feed the list, unioned:

1. **Work type**: a list that applies to any building class.
2. **Building class(es)**: top-ups for each class known. The classes come
   from `hdr.building_class` and the multi-choice NCC classes in
   `det.ncc_class` (for example "5,6" means commercial).
3. **Scope of works**: each in-scope system (`scope.<leaf>` = `in` on the
   whole-site part) adds its consultant.

The rules live in `knowledge/works/package_defaults.yaml`. Code only reads
them, matches and unions.

**Tech stack:** Go (`internal/knowledge`, `internal/procurement`,
`internal/profile`, `internal/store`, `internal/httpapi`), React
(`web/src/Procurement.tsx`), Playwright.

---

## Decisions

Confirmed by the owner, 2026-10-11:

- **D1. No single "Services Engineer" suggestion.** Electrical, Hydraulic,
  Mechanical and Fire Services are suggested separately. One firm holding
  all four gets four RFPs, which is fine. "Services Engineer" stays
  available under Add discipline.
- **D2. Work type alone is enough to suggest.** An unknown building class
  never empties the list.
- **D3. More than one building class is normal.** Every known class adds
  its top-up.
- **D4. Scope drives specialists**: a lift brings Vertical Transport,
  sprinklers bring Fire Services, and so on.
- **D5. The novation shield icon goes.** The route line above the table
  already says whether consultants are novated.
- **D6. The PM corrects by delete and add.** A suggested row can be removed.
  A removed row (suggested or created) is not suggested again on that
  project.

Agent's proposals, for the owner to review (all knowledge stays
`status: draft`):

- **D7. Multiple classes are entered through `det.ncc_class`.** That field is
  already multi-choice (NCC classes 1a to 10) and always shown in the
  profile, so the header needs no change. `hdr.building_class` stays one
  category. The NCC classes map to categories through
  `knowledge/profile/taxonomy.yaml`, using each subclass's `ncc_class`.
- **D8. Draft lists** (Task 2), chosen to be generous, so the PM deletes
  rather than adds:

  | Work type | Any class | Non-residential top-up |
  |---|---|---|
  | new, extend | Architect, Structural, Civil, Town Planner, Geotechnical, Surveyor, Certifier, Electrical, Hydraulic | Traffic, BCA, Access, Mechanical, Fire Services |
  | refurb (fit-out) | Architect, Certifier, Electrical, Hydraulic | Interior Design, BCA, Access, Mechanical, Fire Services |
  | remediation, advisory | (none, as today) | (none) |

  Infrastructure new/extend keeps today's list: Civil, Structural,
  Geotechnical, Surveyor, Environmental, Town Planner. The complexity
  additions (heritage, BAL, flood and so on) are unchanged.

  Scope additions (a rule naming a top-level system matches every leaf
  under it):

  | Scope system in | Consultant |
  |---|---|
  | `vertical-transport` | Vertical Transport |
  | `fire-active.sprinklers`, `.hydrants`, `.fire-water`, `.detection`, `.occupant-warning` | Fire Services |
  | `mechanical` | Mechanical |
  | `hydraulic` | Hydraulic |
  | `electrical`, `fire-active.emergency-lighting` | Electrical |

  Smoke alarms and portable extinguishers add nothing: the electrician or
  the builder covers them. Comms, security and structure scope add nothing
  yet; that would widen this plan.
- **D9. Existing projects:** a live "Services Engineer" package is kept. The
  split suggestions appear beside it, and the PM removes whichever they
  don't want.

## Out of scope

- No Jev change, no new reading, no new service or dependency.
- No change to the profile header, RFP content, cost plan autofill or the
  head-contract (Main Works) proposal.
- Firms and the Comparison column stay as placeholders.
- Trade suggestions from scope (for example a Fire Services Trade).
  Follow-up.

## Speed

`GET /projects/{id}/packages` and the Procurement view stay within the
`packages_read` budget: p50 ≤ 100 ms, p90 ≤ 250 ms. The added work is
reading the `scope.%` rows in the same query, plus an in-memory union over
fewer than 100 rows and fewer than 60 rules. Filing is untouched
(p50 ≤ 1 s / p90 ≤ 2 s).

## Edge cases the tests must prove

- Work type known, building class blank: the any-class list.
- Two categories at once (`det.ncc_class` "2,6"): the union, with no
  duplicates.
- Conflicting `hdr.building_class` across rows: today that empties the
  list. It must now fall back to the classes that are still known and the
  any-class list.
- No work type and no scope: an empty list. Nothing is invented.
- Scope in, but no work type: the scope additions still show.
- A suggested row is removed, then the list is reloaded: it stays gone. A
  different project is unaffected.
- A created package is removed: it is not suggested again.
- Wrong org: dismissing returns 404 and writes nothing.
- A consultant name in the YAML that matches no discipline label in
  `data/intake/disciplines.json`: the catalogue fails to load. This is the
  guard that stops "Services Engineer"-style drift.

---

## Slice 1: remove the shield (lane B)

### Task 1: drop the novation icon from Procurement rows

**Files:**
- Modify: `web/src/Procurement.tsx:10` (import) and `:152` (the `novation-mark` span)
- Modify: `web/src/icons.tsx:102-110` (delete `IconShield` if nothing else uses it; it is only used here today)
- Modify: `web/src/reports.css:193` (delete `.novation-mark`)
- Test: the existing Procurement Playwright spec (`web/e2e/`; grep for `Novated`)

**Step 1:** Grep `web/e2e` for `Novated` / `novation-mark`. Change any
assertion that expects the icon so it expects no icon, and run it to
confirm it fails.

**Step 2:** Remove the span, the import, the icon and the CSS. Keep
`novation` on the `Row` type only if something else reads it; otherwise
remove it.

**Step 3:** `cd web && npx tsc -b && npx playwright test <procurement spec>`.
Expect a pass. Rebuild `web/dist` through bash, not `dev.ps1 -Build`.

**Step 4:** Commit: `Procurement rows drop the novation shield; the route line says it (lane B)`.

---

## Slice 2: the knowledge (lane A)

### Task 2: package_defaults v2, with any-class baselines and scope additions

**Files:**
- Modify: `knowledge/works/package_defaults.yaml`
- Modify: `internal/knowledge/package_defaults.go`
- Test: `internal/knowledge/package_defaults_test.go` (create it if absent)
- Modify: `knowledge/SCHEMA.md` (document the two new shapes beside the existing package-defaults section)

**Shape:**

```yaml
version: 2
status: draft
baselines:
- work_types: [new, extend]          # no building_classes: any class
  consultants: [Architect, Structural, Civil, Town Planner, Geotechnical, Surveyor, Certifier, Electrical, Hydraulic]
- building_classes: [industrial, commercial, institution, mixed]
  work_types: [new, extend]
  consultants: [Traffic, BCA, Access, Mechanical, Fire Services]
# ... the rest as in D8
scope_additions:
- systems: [vertical-transport]
  consultants: [Vertical Transport]
- systems: [fire-active.sprinklers, fire-active.hydrants, fire-active.fire-water, fire-active.detection, fire-active.occupant-warning]
  consultants: [Fire Services]
# ... the rest as in D8
complexity_additions: # unchanged
```

**Step 1: write the failing tests.**
- `TestPackageDefaultsNamesAreDisciplines`: load `../../knowledge`, then
  check that every baseline, scope and complexity consultant equals a
  `consultant.*` label in `data/intake/disciplines.json`.
- `TestPackageDefaultsRejectsUnknownName`: a fixture copy whose consultant
  is `Services Engineerz` fails to load.
- `TestPackageDefaultsRejectsUnknownSystem`: a fixture whose scope system
  is `lifts.magic` fails to load.

**Step 2:** `go test ./internal/knowledge -run PackageDefaults`. Expect a
FAIL.

**Step 3: implement.**
- Add `ScopeAdditions []PackageScopeDefault` to the struct
  (`Systems []string`, `Consultants []string`).
- Accept `version: 2`.
- Allow an empty `building_classes`, but every listed class must still
  exist.
- Validate each consultant name against the discipline labels (the same
  `intakevocab.Disciplines` source that `cost_elements.go:59` reads).
- Validate each system id against `c.systems` or the leaves.
- Edit the YAML to the D8 lists and keep `status: draft`.

**Step 4:** `go test ./internal/knowledge/... && python tools/check_knowledge.py`.
Expect both to pass.

**Step 5:** Commit: `Package defaults name real disciplines, apply without a class, add scope additions (lane A)`.

---

## Slice 3: the suggestion logic (lane A)

### Task 3: profile facts give classes (plural) and scope

**Files:**
- Modify: `internal/profile/package_defaults.go:37-64`
- Test: `internal/profile/package_defaults_test.go`

**Step 1: write the failing tests.**

```go
// two categories from the NCC classes, plus the header category
classes, types, scope := PackageBaselineFacts(cat, rows)
// rows: hdr.building_class=commercial (green), det.ncc_class="2,6" (user),
//       hdr.work_type=new, scope.vertical-transport.passenger-lifts=in,
//       scope.mechanical.ventilation=out
// want classes [commercial residential], types [new],
//      scope [vertical-transport.passenger-lifts]
```

- Conflicting `hdr.building_class` rows: the conflicting header value is
  dropped, and the NCC-derived classes stay.
- A blank or assumption row adds nothing. Keep `appliedWorkRow` for
  `hdr.*` and `det.*`.
- A scope row counts when its value is `in`, whatever its band. That
  matches how Profile shows scope (`internal/httpapi/profile.go:345`).

**Step 2:** `go test ./internal/profile -run PackageBaselineFacts`. Expect a
FAIL.

**Step 3: implement.**
- Change the signature to
  `PackageBaselineFacts(cat *knowledge.Catalog, rows []Row) (classes, workTypes, scope []string)`.
- Map NCC classes to categories by walking `cat.Taxonomy().BuildingClasses`
  and their subclasses' `ncc_class`.
- Return everything sorted and deduped.

**Step 4:** Expect the tests to pass. Commit with Task 4.

### Task 4: SuggestedPackages unions work type, classes and scope

**Files:**
- Modify: `internal/procurement/defaults.go` (both functions)
- Modify: `internal/procurement/defaults_test.go` (update the call sites; today's baseline counts change)
- Modify: `internal/store/packages.go:61-93` and `readPackageFacts` (add `OR (p.kind='whole' AND r.key LIKE 'scope.%')`)

**Step 1: write the failing tests** in `defaults_test.go`, against the real
catalogue:
- `refurb` with no class gives Architect, Certifier, Electrical, Hydraulic.
  No "Services Engineer".
- `refurb` with `[commercial]` also adds Interior Design, BCA, Access,
  Mechanical and Fire Services.
- `new` with `[commercial, residential]` gives the union, each title once.
- No work type, scope `[vertical-transport.passenger-lifts, fire-active.sprinklers]`
  gives exactly Vertical Transport and Fire Services. Their `MatchedFields`
  name the system, for example `{"scope": "fire-active.sprinklers"}`, so
  the row can say why later.
- No work type and no scope gives an empty list.
- A package titled " electrical " (live **or retired**) suppresses the
  Electrical suggestion (D6).
- A key in the new `dismissed` set is left out (input for Task 5).

**Step 2:** `go test ./internal/procurement`. Expect a FAIL.

**Step 3: implement.** Replace the inputs with

```go
type SuggestionInputs struct {
	Classes, WorkTypes, Scope []string
	Facts                     map[string]string
	Existing                  []Package       // live and retired
	Dismissed                 map[string]bool // proposal keys
}
func SuggestedPackages(cat *knowledge.Catalog, in SuggestionInputs) []PackageSuggestion
```

- A baseline matches when one of its work types is in `WorkTypes`, and
  either its `BuildingClasses` is empty or it shares a class.
- A scope rule matches when a scope id equals a rule entry or starts with
  `entry + "."`.
- `taken` now includes retired packages. Keys keep the
  `default:services:<Title>` form, so `ProposalDiscipline` keeps working
  unchanged.
- In `packageOverview`, read retired packages too: check `readPackages`.
  If it filters them, add a separate titles query rather than change what
  `o.Items` returns.

**Step 4:** `go test ./internal/procurement ./internal/profile ./internal/store ./internal/httpapi`.
Expect a pass. Then run the `packages_read` latency gate
(`go test ./internal/latency -run Gate`, or the `intake-bench` packages
case) and record p50/p90.

**Step 5:** Commit: `Procurement suggests every service, by work type, each class and the scope (lane A)`.

---

## Slice 4: delete a suggested line (lane A)

### Task 5: dismiss a package suggestion

**Files:**
- Modify: `internal/store/package_proposed.go` (add `DismissPackageProposal`)
- Modify: `internal/store/packages.go` (load the dismissed keys into `SuggestionInputs.Dismissed`)
- Modify: `internal/httpapi/proposals.go:~200` (on the dismiss path, fall back to the package dismissal when `DismissProposal` returns `ErrProposalUnavailable`, mirroring the accept fallback at line 200)
- Test: `internal/store/package_proposed_test.go`, `internal/httpapi/procurement_test.go`, and an entry in `access_sweep_test.go`

**Step 1: check first.** Read `proposal_decisions` in
`internal/db/migrations/016_proposals.sql` and `017a_*`, to see whether
`record_id` and `inputs_fingerprint` allow a package suggestion's key.
- If they do, write a `dismissed` decision there, with
  `inputs_fingerprint` = the suggestion's `KnowledgeVersion`.
- If they don't, stop and say why before adding a migration. A migration
  is a lane A schema change and needs the owner's nod.

**Step 2: write the failing tests.**
- Dismissing `default:services:Certifier` removes it from
  `ReadPackageProposals` for that project only.
- A wrong-org call returns 404 and adds no row.
- An unknown key returns 404.
- Dismissing twice is idempotent.

**Step 3:** Implement. Scope every query to `org_id` and `project_id`, and
take the project lock as `AcceptPackageProposal` does.

**Step 4:** `go test ./internal/store ./internal/httpapi`. Expect a pass.

**Step 5:** Commit: `A suggested consultant can be removed and stays removed (lane A)`.

### Task 6: Remove on suggested rows

**Files:**
- Modify: `web/src/Procurement.tsx:160-166` (show the `⋯` menu on suggested rows with **Remove** only; it calls the proposal dismiss endpoint and reloads)
- Modify: `web/src/procurementApi.ts` (add `dismiss(project, key, fingerprint)`)
- Test: the Procurement Playwright spec

**Step 1: write the failing e2e test.** On a seeded commercial fit-out:
1. Electrical, Hydraulic, Mechanical, Fire Services, BCA, Access and
   Certifier rows show.
2. Remove Certifier.
3. Reload: it is gone, and Add discipline still offers Certifier.

**Step 2:** Implement with no confirm dialog. The row holds no data, and
the user can add it back from Add discipline.

**Step 3:** `npx tsc -b && npx playwright test <spec>`. Expect a pass.
Rebuild `web/dist`.

**Step 4:** Commit: `Suggested Procurement rows can be removed (lane A)`.

---

## Done looks like

On Petersham (or any fit-out with the building class unread) the
Procurement table lists, with no clicks:
- Architect, Certifier, Electrical and Hydraulic.
- Mechanical and Fire Services when that scope is in.
- Vertical Transport when a lift is in scope.

There's no shield anywhere. Removing Certifier sticks. All the gates above
pass, and `packages_read` p50/p90 is reported against 100/250 ms.

## Follow-ups (named, not built)

- Trade suggestions from scope (Fire Services Trade, Mechanical Services).
- Comms/security and structure scope adding ICT, Security and Structural.
- A "why suggested" hint from `MatchedFields`.
- Owner review of the D8 lists (marking them `reviewed` is the owner's job).

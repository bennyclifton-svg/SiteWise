# Procurement table: packages and RFP/RFT in one screen

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** A user opens Procurement, adds a discipline, and clicks **Create**.
That's the whole flow: no Packages tab, no report-type choice, no
"create a package first".

**Architecture:** A package is still the record that holds scope, stages and
discipline, and an RFP/RFT is still built from one package. The user just
never creates a package as a separate step. Procurement becomes one table
with one row per package, grouped into Consultants and Trades.

- **Choosing a discipline** in a new row saves a package. The server works
  out the title, kind and novation.
- **Create** makes the report in one request: an RFP for a consultant row,
  an RFT for a trade row. On a suggested row, the same request also accepts
  the proposal.
- **View** opens the existing report editor full width, with a back link.
- **The row menu** opens that package's scope editor, which moves here from
  the Packages tab, and can remove the row.

**Tech stack:**
- Go: `internal/procurement`, `internal/store`, `internal/httpapi`.
- React: new `web/src/Procurement.tsx`, `web/src/PackageScope.tsx` and
  `web/src/procurementApi.ts`; `web/src/Reports.tsx` becomes the report view
  only.
- Playwright.

**Lane:** A, product core. It creates packages and reports a user relies on.
These must still pass: the existing package and report API tests, the RFP
golden and answer-key tests, and the `packages_write` and `report_write`
budgets.

---

## Owner decisions (2026-10-10)

- **P1. One Procurement screen; the Packages tab is deleted.** The
  following move inline:
  - Scope editing goes in the row `⋯` menu, under **Scope**.
  - Gaps become one line above the table.
  - The procurement-route notice becomes one line above the table.
- **P2. No report-type choice.** A consultant discipline makes an RFP and a
  trade discipline makes an RFT. Supply packages (`supplier.*`) can't be
  added from the table. A legacy supply package shows with "—" in the
  RFP/RFT column.
- **P3. Add discipline adds a blank row** with one dropdown, grouped by
  Consultants and Trades. Choosing a discipline saves the package straight
  away.
- **P4. Profile suggestions show as faint "suggested" rows.** Clicking
  **Create** on one accepts it and creates its report in one request.
  Suggested rows include the consultant baseline and the head-contract
  (Main Works) proposal.
- **P5. Firm 1–3 and Comparison are placeholders.** They're shown disabled
  and nothing is stored. Tenderers are a later design.
- **P6. PMP leaves the table** and will get its own tab later, per D4. Until
  then, one line under the table reads "Project management plan:
  Create/View", so existing PMPs stay reachable.

## Out of scope

- Firms and tenderers.
- Tender comparison.
- Export and copy icons.
- Landscape/portrait choice.
- Split or merge of packages.
- Reordering rows.
- A PMP tab.
- Changing how an RFP or RFT is assembled.

## Budgets

| Path | Route | p50 | p90 |
|---|---|---|---|
| `packages_write` | `POST /projects/{id}/procurement/rows` | 100 ms | 250 ms |
| `report_write` | `POST /projects/{id}/procurement/create` | 100 ms | 250 ms |
| `packages_read`, `report_read` | table load (two parallel GETs) | 100 ms | 250 ms |

- No Jev on any of these paths. Tests assert `jevHits == 0`.
- Create is one round trip, including the accept-and-create on a suggested
  row.
- The table load is two GETs in parallel, never serial.

## Edge cases (must be handled and tested)

- **Wrong org:** returns 404 on both new routes.
- **Unknown discipline id, or a `supplier.*` id:** returns 422.
- **The same discipline added twice in a project:** returns 409. The
  dropdown also hides disciplines already used.
- **Create on a package that already has a report:** returns the existing
  report with status 200, not a second one. This covers a double-click.
- **Create with a proposal key that is no longer proposed:** returns 404.
  The table reloads.
- **Body with both or neither of `package_id` and `proposal_key`:** returns
  400.
- **Retired package:** returns 404.
- **Legacy package with no `discipline_id`:** grouped by kind, with the
  title as its label. Services go under Consultants; works and supply go
  under Trades.

## Done

- Packages tab gone.
- Procurement shows the table.
- The Petersham structural row creates and opens the structural RFP.
- `go test -p 1 ./...` passes.
- All Playwright specs pass, rewritten where they used the Packages tab or
  the Report type dropdown.
- The bench gate passes for `packages_write` and `report_write`.

## Workspace

Build in a worktree: `../sitewise-procurement`, branch
`procurement/table`. It needs the junctions the RFP build used, for
`.tools`, `web/node_modules` and `data/eval/*/private` (see
`docs/plans/2026-10-10-rfp-single-report.md`).

---

### Task 1: Map a discipline to a package

**Files:**
- Create: `internal/procurement/discipline.go`
- Test: `internal/procurement/discipline_test.go`

**Step 1: Write the failing test**

```go
package procurement

import "testing"

func TestDisciplinePackage(t *testing.T) {
	cases := []struct{ id, label, kind, scope string; ok bool }{
		{"consultant.structural", "Structural", "services", "", true},
		{"trade.main_works", "Main Works", "works", "head_contract", true},
		{"trade.electrical", "Electrical Services", "works", "trade", true},
		{"supplier.facade", "Facade supply", "", "", false},
		{"other.thing", "Other", "", "", false},
	}
	for _, c := range cases {
		p, ok := DisciplinePackage(c.id, c.label)
		if ok != c.ok || p.Kind != c.kind || p.WorksScope != c.scope {
			t.Fatalf("%s: %+v %v", c.id, p, ok)
		}
		if ok && (p.DisciplineID != c.id || p.Title != c.label || p.LifecycleStatus != "planned") {
			t.Fatalf("%s: %+v", c.id, p)
		}
	}
}

func TestProposalDiscipline(t *testing.T) {
	ds := []Discipline{
		{ID: "consultant.services", Label: "Services", Aliases: []string{"Services Engineer"}},
		{ID: "consultant.town_planner", Label: "Town Planner"},
		{ID: "trade.main_works", Label: "Main Works"},
	}
	for key, want := range map[string]string{
		"default:services:Services Engineer": "consultant.services",
		"default:services:town planner":      "consultant.town_planner",
		WorksProposalKey:                     "trade.main_works",
		"default:services:Unknown":           "",
	} {
		if got := ProposalDiscipline(key, ds); got != want {
			t.Fatalf("%s: got %q want %q", key, got, want)
		}
	}
}
```

**Step 2: Run it and confirm it fails**

Run: `go test ./internal/procurement/ -run 'DisciplinePackage|ProposalDiscipline'`

Expected: FAIL, with `undefined: DisciplinePackage`.

**Step 3: Implement**

```go
package procurement

import "strings"

// Discipline is one entry of the filing discipline list
// (data/intake/disciplines.json), passed in by the caller so this package
// stays free of the intake loader.
type Discipline struct {
	ID      string
	Label   string
	Aliases []string
}

// DisciplinePackage is the package a Procurement row makes for a discipline.
// Consultants are services (RFP); trades are works (RFT), and Main Works is
// the head contract. Suppliers have no RFP/RFT, so they are refused.
func DisciplinePackage(id, label string) (Package, bool) {
	p := Package{DisciplineID: id, Title: label, LifecycleStatus: "planned"}
	switch {
	case strings.HasPrefix(id, "consultant."):
		p.Kind = "services"
	case id == "trade.main_works":
		p.Kind, p.WorksScope = "works", "head_contract"
	case strings.HasPrefix(id, "trade."):
		p.Kind, p.WorksScope = "works", "trade"
	default:
		return Package{}, false
	}
	return p, true
}

// ProposalDiscipline names the discipline a proposal stands for, so an
// accepted suggestion gets the discipline RFP without a confirm step.
// Consultant proposal keys end in the roster name; an unknown name yields "".
func ProposalDiscipline(key string, ds []Discipline) string {
	if key == WorksProposalKey {
		return "trade.main_works"
	}
	name, ok := strings.CutPrefix(key, "default:services:")
	if !ok {
		return ""
	}
	for _, d := range ds {
		if strings.EqualFold(d.Label, name) {
			return d.ID
		}
		for _, a := range d.Aliases {
			if strings.EqualFold(a, name) {
				return d.ID
			}
		}
	}
	return ""
}
```

**Step 4: Run it and confirm it passes**

Run: `go test ./internal/procurement/`

Expected: PASS.

**Step 5: Commit** with the message `Map a discipline to its package kind (lane A)`.

---

### Task 2: Store creates a discipline row, one per discipline

**Files:**
- Modify: `internal/store/packages.go`
  - Extract the profile query from `packageOverview` into `readPackageFacts`.
  - Add `ErrDuplicatePackage`.
  - Add `CreateDisciplinePackage`.
- Modify: `internal/httpapi/packages.go`, in `packageError`, to map
  `ErrDuplicatePackage` to 409.

**Step 1: Refactor without changing behaviour**

1. Move the query and scan loop in `packageOverview` (the `rows, err :=
   tx.Query(... 'hdr.work_type' ...)` block) into this function:
   ```go
   // readPackageFacts reads the profile rows packages are proposed from.
   func readPackageFacts(ctx context.Context, tx pgx.Tx, org, project string) ([]profile.Row, error)
   ```
2. Have `packageOverview` call `readPackageFacts`.
3. Run `go test ./internal/store/ -run Package`.

Expected: PASS, unchanged.

**Step 2: Add the store method**

```go
var ErrDuplicatePackage = errors.New("discipline already has a package")

// CreateDisciplinePackage saves the package a Procurement row stands for.
// The caller supplies a package from procurement.DisciplinePackage; novation
// follows the procurement route, as for accepted proposals. One live
// package per discipline keeps the table one row per discipline.
func (s *Store) CreateDisciplinePackage(ctx context.Context, org, project, actor string, p procurement.Package) (procurement.Package, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return p, err
	}
	defer tx.Rollback(ctx)
	if err := lockProject(ctx, tx, org, project); err != nil {
		return p, err
	}
	var exists, taken bool
	if err := tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM projects WHERE org_id=$1::uuid AND id=$2::uuid),
 EXISTS(SELECT 1 FROM packages WHERE org_id=$1::uuid AND project_id=$2::uuid AND discipline_id=$3 AND retired_at IS NULL)`, org, project, p.DisciplineID).Scan(&exists, &taken); err != nil {
		return p, err
	}
	if !exists {
		return p, ErrNotFound
	}
	if taken {
		return p, ErrDuplicatePackage
	}
	if err := packageActor(ctx, tx, org, actor); err != nil {
		return p, err
	}
	facts, err := readPackageFacts(ctx, tx, org, project)
	if err != nil {
		return p, err
	}
	p.Novation = p.Kind == "services" && procurement.BuilderDesigns(profile.PackageComplexityFacts(facts)["procurement_route"])
	if err := procurement.ValidatePackage(p); err != nil {
		return p, fmt.Errorf("%w: %v", ErrInvalidPackage, err)
	}
	p.Stages = procurement.DefaultStages(p, s.workCatalog())
	provenance, _ := json.Marshal(map[string]string{"actor": actor, "discipline": p.DisciplineID})
	if p, err = s.insertProposedPackage(ctx, tx, org, project, p, provenance); err != nil {
		return p, err
	}
	// A discipline the user picked is their choice, not a calculation.
	if _, err := tx.Exec(ctx, `UPDATE packages SET origin='user' WHERE org_id=$1::uuid AND id=$2::uuid`, org, p.ID); err != nil {
		return p, err
	}
	p.Origin = "user"
	return p, s.finishPackagesWrite(ctx, tx, org, project)
}
```

**Step 3: Make `insertProposedPackage` write `discipline_id`**

Today it drops `discipline_id`. Add `discipline_id` to its INSERT as
`NULLIF($10,'')`, and pass `p.DisciplineID`.

**Step 4: Map the new error in `packageError`**

Add this case before the 23505 case:

```go
case errors.Is(err, store.ErrDuplicatePackage):
	http.Error(w, "discipline already has a package", http.StatusConflict)
```

**Step 5: Check everything still passes**

Run: `go test -p 1 ./internal/store/ ./internal/httpapi/ -run 'Package|Report'`

Expected: PASS. The new method is covered through the API in Task 4.

**Step 6: Commit** with the message `Store one package per discipline row (lane A)`.

---

### Task 3: Store creates the report in one transaction

**Files:**
- Modify: `internal/store/reports.go`
- Modify: `internal/store/package_proposed.go`

**Step 1: Split out the parts of `AcceptProposedPackage` and `CreateReport` that don't commit**

1. In `package_proposed.go`, rename the body of `AcceptProposedPackage`
   after `packageActor` into this function:
   ```go
   func (s *Store) acceptProposalTx(ctx context.Context, tx pgx.Tx, org, project, actor, key, disciplineID string) (procurement.Package, error)
   ```
   It returns before `finishPackagesWrite`.
2. Set `DisciplineID: disciplineID` on both package literals it builds.
3. `AcceptProposedPackage` keeps its signature. It passes `""` as the
   discipline, then calls `finishPackagesWrite`.
4. In `reports.go`, move the INSERT in `CreateReport` into this function:
   ```go
   func insertReport(ctx context.Context, tx pgx.Tx, org, project, kind, pkg, name string) (Report, error)
   ```
   `CreateReport` calls it.

Run: `go test -p 1 ./internal/store/ -run 'Package|Report'`

Expected: PASS, unchanged.

**Step 2: Add the one-request create**

```go
// ProcurementCreate makes the RFP or RFT for one Procurement row. A
// suggested row passes its proposal key, and the proposal is accepted in the
// same transaction, so Create is one round trip. A package that already has
// a report returns that report (created=false): a double-click never makes
// two.
func (s *Store) ProcurementCreate(ctx context.Context, org, project, actor, packageID, proposalKey, disciplineID string) (procurement.Package, Report, bool, error) {
	var p procurement.Package
	var r Report
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return p, r, false, err
	}
	defer tx.Rollback(ctx)
	if err := lockProject(ctx, tx, org, project); err != nil {
		return p, r, false, err
	}
	if err := packageActor(ctx, tx, org, actor); err != nil {
		return p, r, false, err
	}
	accepted := proposalKey != ""
	if accepted {
		p, err = s.acceptProposalTx(ctx, tx, org, project, actor, proposalKey, disciplineID)
	} else {
		p, err = readPackage(ctx, tx, org, project, packageID)
	}
	if errors.Is(err, pgx.ErrNoRows) {
		return p, r, false, ErrNotFound
	}
	if err != nil {
		return p, r, false, err
	}
	if p.RetiredAt != nil {
		return p, r, false, ErrNotFound
	}
	kind := map[string]string{"services": "rfp", "works": "rft"}[p.Kind]
	if kind == "" {
		return p, r, false, ErrInvalidReport
	}
	var raw []byte
	err = tx.QueryRow(ctx, `SELECT to_jsonb(r) FROM reports r WHERE org_id=$1::uuid AND project_id=$2::uuid AND package_id=$3::uuid ORDER BY id LIMIT 1`, org, project, p.ID).Scan(&raw)
	if err == nil {
		return p, r, false, json.Unmarshal(raw, &r)
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return p, r, false, err
	}
	if r, err = insertReport(ctx, tx, org, project, kind, p.ID, p.Title); err != nil {
		return p, r, false, err
	}
	if accepted {
		if err := BumpRevision(ctx, tx, org, project, "packages"); err != nil {
			return p, r, false, err
		}
	}
	return p, r, true, s.finishReportWrite(ctx, tx, org, r, "", "drafting")
}
```

Check before you write it: how `finishPackagesWrite` appends its
`packages` event. If the Packages SSE consumers need the event as well as
the revision bump, append the `packages` event here too. Copy the two
lines from `finishPackagesWrite`; don't call it, because it commits.

**Step 3: Check everything still passes**

Run: `go test -p 1 ./internal/store/`

Expected: PASS.

**Step 4: Commit** with the message `Accept a proposal and create its report in one transaction (lane A)`.

---

### Task 4: Two API routes

**Files:**
- Create: `internal/httpapi/procurement.go`
- Modify: `internal/httpapi/projects.go`. Add the two routes, and map them
  in `routePaths`:
  - `POST /projects/{id}/procurement/rows` → `pathPackagesWrite`
  - `POST /projects/{id}/procurement/create` → `pathReportWrite`
- Test: `internal/httpapi/procurement_test.go`

**Step 1: Write the failing test**

```go
package httpapi_test

import (
	"net/http"
	"sitewise/internal/procurement"
	"testing"
)

func TestProcurementRowsAndCreate(t *testing.T) {
	a := newApp(t, withProfile(t))
	m := a.member(t, newUUID(t))
	project := m.createProject(t, "Procurement")
	rows := "/api/projects/" + project + "/procurement/rows"
	create := "/api/projects/" + project + "/procurement/create"

	var p procurement.Package
	m.call(t, http.MethodPost, rows, []byte(`{"discipline_id":"consultant.structural"}`), http.StatusCreated, &p)
	if p.Kind != "services" || p.DisciplineID != "consultant.structural" || p.Title != "Structural" || p.Origin != "user" {
		t.Fatalf("%+v", p)
	}
	m.call(t, http.MethodPost, rows, []byte(`{"discipline_id":"consultant.structural"}`), http.StatusConflict, nil)
	m.call(t, http.MethodPost, rows, []byte(`{"discipline_id":"supplier.facade"}`), http.StatusUnprocessableEntity, nil)
	m.call(t, http.MethodPost, rows, []byte(`{"discipline_id":"consultant.nope"}`), http.StatusUnprocessableEntity, nil)
	var trade procurement.Package
	m.call(t, http.MethodPost, rows, []byte(`{"discipline_id":"trade.electrical"}`), http.StatusCreated, &trade)
	if trade.Kind != "works" || trade.WorksScope != "trade" {
		t.Fatalf("%+v", trade)
	}

	type out struct {
		Package procurement.Package `json:"package"`
		Report  struct{ ID, Kind, PackageID string } `json:"report"`
	}
	var first, again, rft out
	m.call(t, http.MethodPost, create, []byte(`{"package_id":"`+p.ID+`"}`), http.StatusCreated, &first)
	if first.Report.Kind != "rfp" || first.Report.PackageID != p.ID {
		t.Fatalf("%+v", first)
	}
	m.call(t, http.MethodPost, create, []byte(`{"package_id":"`+p.ID+`"}`), http.StatusOK, &again)
	if again.Report.ID != first.Report.ID {
		t.Fatal("second create made a second report")
	}
	m.call(t, http.MethodPost, create, []byte(`{"package_id":"`+trade.ID+`"}`), http.StatusCreated, &rft)
	if rft.Report.Kind != "rft" {
		t.Fatalf("%+v", rft)
	}

	// A suggested row: accepted and created in one request, with its discipline.
	var overview struct{ Suggestions []procurement.PackageSuggestion `json:"suggestions"` }
	m.getJSON(t, "/api/projects/"+project+"/packages", &overview)
	if len(overview.Suggestions) == 0 {
		t.Skip("profile fixture proposes no consultants")
	}
	var sug out
	m.call(t, http.MethodPost, create, []byte(`{"proposal_key":"`+overview.Suggestions[0].Key+`"}`), http.StatusCreated, &sug)
	if sug.Package.DisciplineID == "" || sug.Report.Kind != "rfp" {
		t.Fatalf("%+v", sug)
	}
	m.call(t, http.MethodPost, create, []byte(`{"proposal_key":"`+overview.Suggestions[0].Key+`"}`), http.StatusNotFound, nil)

	for _, bad := range []string{`{}`, `{"package_id":"` + p.ID + `","proposal_key":"x"}`} {
		m.call(t, http.MethodPost, create, []byte(bad), http.StatusBadRequest, nil)
	}
	foreign := a.member(t, newUUID(t))
	foreign.call(t, http.MethodPost, rows, []byte(`{"discipline_id":"consultant.civil"}`), http.StatusNotFound, nil)
	foreign.call(t, http.MethodPost, create, []byte(`{"package_id":"`+p.ID+`"}`), http.StatusNotFound, nil)
	if a.jevHits.Load() != 0 {
		t.Fatal("procurement rows called Jev")
	}
}
```

Before you write the test, check that the `withProfile` fixture yields a
building class and work type the baselines propose. If it doesn't, add one
consultant through a profile write, the way `package_proposed_test.go`
does. Remove the `t.Skip` once the fixture is sure.

**Step 2: Run it and confirm it fails**

Run: `go test -p 1 ./internal/httpapi/ -run TestProcurementRowsAndCreate`

Expected: FAIL with a 404 or 405 on the new route.

**Step 3: Implement**

```go
package httpapi

import (
	"net/http"

	"sitewise/internal/procurement"
	"sitewise/internal/store"
)

// disciplines adapts the filing discipline list for procurement.
func disciplines(deps Deps) []procurement.Discipline {
	out := make([]procurement.Discipline, 0, len(deps.Catalog.Disciplines))
	for _, d := range deps.Catalog.Disciplines {
		out = append(out, procurement.Discipline{ID: d.ID, Label: d.Label, Aliases: d.Aliases})
	}
	return out
}

// postProcurementRow saves the package for a discipline chosen in the
// Procurement table. Only the id is trusted; title and kind come from the
// discipline list.
func postProcurementRow(w http.ResponseWriter, r *http.Request, deps Deps) {
	if !originOK(r, deps.PublicOrigin) {
		http.Error(w, "origin rejected", http.StatusForbidden)
		return
	}
	session, ok := memberSession(w, r, deps)
	if !ok {
		return
	}
	project := r.PathValue("id")
	if !uuidPattern.MatchString(project) {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}
	var body struct {
		DisciplineID string `json:"discipline_id"`
	}
	if err := readProposalJSON(w, r, deps.MaxBodyBytes, &body); err != nil || body.DisciplineID == "" || len(body.DisciplineID) > 100 {
		http.Error(w, "invalid body", http.StatusBadRequest)
		return
	}
	var p procurement.Package
	found := false
	for _, d := range deps.Catalog.Disciplines {
		if d.ID == body.DisciplineID {
			p, found = procurement.DisciplinePackage(d.ID, d.Label)
			break
		}
	}
	if !found {
		http.Error(w, "unknown or unsupported discipline", http.StatusUnprocessableEntity)
		return
	}
	p, err := deps.Store.CreateDisciplinePackage(r.Context(), session.OrgID, project, session.UserID, p)
	if err != nil {
		packageError(w, err)
		return
	}
	if deps.Broker != nil {
		deps.Broker.Wake(session.OrgID)
	}
	writeJSON(w, http.StatusCreated, p)
}

// postProcurementCreate makes a row's RFP or RFT in one request; the report
// kind follows the package kind, never the client.
func postProcurementCreate(w http.ResponseWriter, r *http.Request, deps Deps) {
	if !originOK(r, deps.PublicOrigin) {
		http.Error(w, "origin rejected", http.StatusForbidden)
		return
	}
	session, ok := memberSession(w, r, deps)
	if !ok {
		return
	}
	project := r.PathValue("id")
	if !uuidPattern.MatchString(project) {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}
	var body struct {
		PackageID   string `json:"package_id"`
		ProposalKey string `json:"proposal_key"`
	}
	if err := readProposalJSON(w, r, deps.MaxBodyBytes, &body); err != nil || (body.PackageID == "") == (body.ProposalKey == "") || len(body.ProposalKey) > 300 || body.PackageID != "" && !uuidPattern.MatchString(body.PackageID) {
		http.Error(w, "invalid body", http.StatusBadRequest)
		return
	}
	discipline := ""
	if body.ProposalKey != "" {
		discipline = procurement.ProposalDiscipline(body.ProposalKey, disciplines(deps))
	}
	p, report, created, err := deps.Store.ProcurementCreate(r.Context(), session.OrgID, project, session.UserID, body.PackageID, body.ProposalKey, discipline)
	if err != nil {
		reportError(w, err)
		return
	}
	status := http.StatusOK
	if created {
		status = http.StatusCreated
		if deps.Broker != nil {
			deps.Broker.Wake(session.OrgID)
		}
	}
	writeJSON(w, status, struct {
		Package procurement.Package `json:"package"`
		Report  store.Report        `json:"report"`
	}{p, report})
}
```

Register both routes in `projects.go`, next to `POST /projects/{id}/packages`,
and add both to `routePaths` as listed under Files.

**Step 4: Run it and confirm it passes**

Run: `go test -p 1 ./internal/httpapi/`

Expected: PASS, including the committed-budgets gate test. That test fails
if a route has no budget path.

**Step 5: Commit** with the message `Procurement rows and one-request RFP/RFT create (lane A)`.

---

### Task 5: Web API client

**Files:**
- Create: `web/src/procurementApi.ts`

```ts
import { request } from "./api";
import type { PackageRecord } from "./packageApi";
import type { SavedReport } from "./reportApi";

export const procurementApi = {
  addRow: (project: string, disciplineId: string) => request<PackageRecord & { discipline_id?: string }>("POST", `/projects/${project}/procurement/rows`, { discipline_id: disciplineId }),
  /** Exactly one of package_id or proposal_key. Returns the existing report when the row already has one. */
  create: (project: string, from: { package_id: string } | { proposal_key: string }) => request<{ package: PackageRecord; report: SavedReport }>("POST", `/projects/${project}/procurement/create`, from),
  retire: (project: string, pkg: string, version: number) => request<PackageRecord>("PATCH", `/projects/${project}/packages/${pkg}`, { retired: true, version }),
};
```

Also add `discipline_id?: string` and `retired_at?: string` to
`PackageRecord` in `web/src/packageApi.ts`.

Check: `.\node_modules\.bin\tsc.cmd --noEmit`, run in `web`, has no
errors.

**Commit** with the message `Procurement API client (lane A)`.

---

### Task 6: Move the scope editor out of Packages

**Files:**
- Create: `web/src/PackageScope.tsx`
- Delete (in Task 8): `web/src/Packages.tsx`

1. Move the scope parts of `Packages.tsx` into
   `export function PackageScope({ projectId, pkg, tick, onSignedOut }: { projectId: string; pkg: PackageRecord; tick: number; onSignedOut: () => void })`.
   That means:
   - the `scope`, `works`, `input`, `editing`, `busy`, `error` and
     `reload` state;
   - the `fail`, `saveScope` and `workTitle` helpers;
   - the "Assigned scope" section and the "Add scope" / "Edit scope
     wording" section.

   Keep the markup and labels byte-for-byte, so the Playwright selectors
   still match. The heading stays `<h2>{pkg.title}</h2>`.
2. Have it load works itself, using `packageApi.works`, along with
   `packageApi.scope` and the project id.
3. Don't change behaviour. This is a move.

Check: `tsc --noEmit` passes.

**Commit** with the message `Move the package scope editor into its own component (lane B)`.

---

### Task 7: Reports.tsx becomes the report view only

**Files:**
- Modify: `web/src/Reports.tsx`

1. Change the signature to
   `export function Reports({ reportId, tick, onBack, onSignedOut }: { reportId: string; tick: number; onBack: () => void; onSignedOut: () => void })`.
2. Delete the library-only parts:
   - the `items`, `kind`, `packageId` and `loading` state, and their load
     effect;
   - `create()` and `rfpLabel()`;
   - `availablePackages` and its effect;
   - the whole `<div className="report-library">` block.
3. Replace `selected` with `reportId` throughout. Delete the `setSelected`
   calls.
4. `confirmDiscipline` needs the package version. Fetch it inside the
   function:
   ```ts
   const pkg = (await reportApi.packages(view.report.project_id ?? "")).items.find(p => p.id === view.report.package_id);
   ```
   If `SavedReport` lacks `project_id`, add it to the type; the server's
   `Report` already sends it.

   This runs only on that one click, so it isn't a hot path.
5. Put a back link at the top of the reader:
   `<button type="button" className="btn btn-small report-back" onClick={onBack}>← Procurement</button>`.
   Disable it while `editor` is open, so unsaved wording isn't dropped.
6. Remove the two-column grid. The reader is full width (Task 9 CSS).

Check: `tsc --noEmit` passes.

**Commit** with the message `Report view opens full width from Procurement (lane B)`.

---

### Task 8: The Procurement table

**Files:**
- Create: `web/src/Procurement.tsx`
- Modify: `web/src/Project.tsx`. Remove the Packages button, the
  `showPackages`/`packagesOpened` state and the `<Packages>` mount. Mount
  `<Procurement>` where `<Reports>` was, and pass `onOpenProfile` as
  Packages had it.
- Delete: `web/src/Packages.tsx`

`Procurement.tsx` holds a view state:
`{ at: "table" } | { at: "report", id } | { at: "scope", pkg }`.

The table view does the following.

- **Load:** `Promise.all([packageApi.list(projectId), reportApi.list(projectId), api.catalog()])`,
  rerun on `tick` and `reload`. The catalog is fetched once; keep it in its
  own effect with `[]` deps.
- **Rows:**
  - live packages (`!retired_at`);
  - then suggestions, as `suggested`;
  - then `proposed_works`, only when it has no `package_id`, as a
    suggested "Main Works" row.
- **Grouping:** Consultants holds rows of kind `services`. Trades holds
  rows of kind `works` and `supply`. Sort within each group by title.
- **Discipline cell:** the catalog label for `discipline_id`, else the
  package title. A shield icon is drawn when `novation` is set; check
  `icons.tsx` for one before adding any.
- **Firm 1–3 cells:** `<input disabled placeholder="Add firm" aria-label="Firm 1 for {title}" title="Firms come later" />`.
  Nothing is saved.
- **RFP / RFT cell:**
  - A live services or works package with a report shows **View**, which
    opens the report view.
  - A live package without a report shows **Create**, which calls
    `procurementApi.create({ package_id })` and then opens the report.
  - A suggested row shows **Create**, which calls
    `procurementApi.create({ proposal_key })` and then opens the report.
  - A supply package shows "—".
  - Disable only the clicked row's button while it saves, using a
    `busyRow` key.
- **Comparison cell:** disabled text "Compare".
- **`⋯` menu** (a `<details>`):
  - **Scope** opens the scope view.
  - **Remove** opens `ConfirmDialog`, then calls `procurementApi.retire`.
  - Suggested rows have no menu.
- **Add discipline:**
  - Appends one draft row containing a `<select aria-label="Discipline">`,
    an **Add row** button and a **Cancel** button.
  - The options are two `<optgroup>`s, "Consultants" (`consultant.*`) and
    "Trades" (`trade.*`). Leave out ids already used by a live package.
  - **Add row** calls `procurementApi.addRow`, then reloads.
  - On a 409, say "That discipline already has a row." and reload.
- **Above the table:**
  - One route line: the same three messages as Packages, plus the
    "Answer on the profile" button.
  - One gaps line:
    - "Every accepted work item is covered." when there are none;
    - otherwise, for example, "2 work items have a gap:", followed by the
      same `gapWords` text, comma-joined.
- **Below the table:** "Project management plan:"
  - **View** when a `kind === "pmp"` report exists;
  - otherwise **Create**, which calls `reportApi.create(projectId, "", "pmp")`
    and opens it.
- **Errors:**
  - 401 calls `onSignedOut`.
  - 404 on create says "This suggestion changed. The table has been
    refreshed." and reloads.
  - Anything else says "Could not save. Try again."
- **Section:** `<section aria-label="Procurement">` with `<h1>Procurement</h1>`.
- **Narrow screens:** below 700px the table scrolls inside
  `.procurement-scroll` (`overflow-x: auto`), so the page itself never
  scrolls sideways.

The report view renders `<Reports reportId tick onBack={() => setView({ at: "table" })} onSignedOut />`.

The scope view renders a back button, then
`<PackageScope projectId pkg tick onSignedOut />`.

Check:
1. `tsc --noEmit` passes.
2. `vite build` passes.
3. The pages still load in `tools/dev.ps1`. Rebuild `web/dist` via bash if
   `-Build` is broken; that's a known gotcha.

**Commit** with the message `Procurement table replaces the Packages tab and report library (lane A)`.

---

### Task 9: Styles

**Files:**
- Modify: `web/src/reports.css`

1. Add the classes the table needs:
   - `.procurement-table` (borders, and group header rows in muted small
     caps);
   - `.procurement-scroll`;
   - `.row-suggested` (muted text and a dashed left border);
   - `.firm-cell input:disabled`;
   - `.report-back`;
   - a full-width `.reports` reader that has no library column.
2. Use the existing tokens and button classes. Match the screenshot's
   density. Don't add new colours.

**Commit** with the message `Style the Procurement table (lane B)`.

---

### Task 10: Playwright

**Files:**
- Create: `web/tests/procurement.spec.ts`. It replaces `packages.spec.ts`.
- Modify: `web/tests/reports.spec.ts`, `rfp-editor.spec.ts`,
  `report-issue.spec.ts`, `works.spec.ts` and `delivery.spec.ts`. Each one
  that clicks "Packages", "Report type", "Accept …" or "Create RFP draft"
  must now drive the table.
- Delete: `web/tests/packages.spec.ts`

`procurement.spec.ts` covers the following.

1. On a new project with one work item ("Rooftop chiller", as in the old
   spec):
   - Procurement shows the route line and the gap line naming Rooftop
     chiller.
   - A suggested "Main Works" row shows **Create**.
2. Clicking **Create** on Main Works opens "RFT — Head contract", or
   whatever title the accepted proposal has. **← Procurement** returns.
   The row now shows **View**, and the gap line says everything is
   covered.
3. **Add discipline**, then choose Structural, then **Add row**: a
   Structural row appears under Consultants. **Create** opens an RFP with
   a structural heading.
4. Adding Structural again isn't offered: the option isn't in the
   dropdown.
5. In the `⋯` menu, **Scope** opens the scope editor with the package
   heading.
6. **Remove**, then confirm: the row is gone.
7. The PMP line: **Create** opens a PMP, and **Assemble draft** is visible.
   This replaces `report-issue.spec.ts`'s dropdown path.
8. At a 390px viewport, `scrollWidth <= innerWidth`.

Run the following in `web`: `npx playwright test`

Expected: all specs pass.

**Commit** with the message `Drive Procurement through the table in the browser specs (lane A)`.

---

### Task 11: Gate and record

1. Run `go test -p 1 ./...`. Everything must pass.
2. Run `python tools/check_knowledge.py`. It must pass, even though
   knowledge is untouched.
3. Run all Playwright specs.
4. Run the bench for `packages_write`, `report_write`, `packages_read` and
   `report_read`, the same way as the RFP single-report gate. Each one
   must be within budget. Record p50 and p90 in this plan's Status.
5. Add a Status section to this plan. It covers:
   - the commits;
   - any departures from the plan;
   - the timings;
   - the owner check that's pending: the Petersham structural row,
     created from the table, in the browser.
6. Commit with the message `Record the Procurement table build status`.
7. Merge to local main. This is allowed under the standing authorisation;
   don't push.

## What calls what (for the owner)

- **Table load:** two GETs that already exist, `/packages` and `/reports`,
  plus the cached `/catalog`. No new read path.
- **Choosing a discipline:**
  - Calls `POST /procurement/rows`.
  - The server looks up the label in `disciplines.json` and creates one
    package with default stages. Novation follows the route.
  - It can fail if the discipline is already used (409) or unknown or
    supply (422).
- **Create:**
  - Calls `POST /procurement/create`.
  - In one transaction it optionally accepts the proposal, then either
    finds the row's existing report or inserts a new one. The report kind
    comes from the package kind.
  - It can fail if the proposal changed (404) or the package is retired
    (404).
- **View:** opens the unchanged report editor. Assembly, settings, facts
  and issue all work as before.
- **Remove:** calls the existing `PATCH /packages/{pkg}` with
  `retired: true`. Any report on that package then shows "This package is
  retired", as today.
- **No new dependency. No Jev.**

# Change Log, Undo and Operations Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** A batch of named operations can be applied to a project's Cost Plan,
Profile values or Program in one call, and any such change can be undone,
with no model involved.

**Architecture:** A change applied through the new `apply` entry point marks
its database transactions. Row triggers on the Cost Plan and Profile value
tables copy each marked row before and after into a per-project change log.
Undo replays that log newest first, after checking every row is still as the
change left it. Operations are a registry of named edits that call the store
writes the screens already call; a test fails when a write route has no
operation and no stated reason.

**Tech Stack:** Go, PostgreSQL 17 (PL/pgSQL triggers, `jsonb`), pgx v5. No new
dependency.

**Design:** `docs/design/2026-11-14-chat-control-and-email-intake.md`,
sections 4 and 5, build-order steps 1 and 2. This plan is the groundwork the
chat agent needs. It does not call a model and adds no screen.

**Status:** written 14 November 2026 from reading `main`. Nothing in it has
been compiled or run. Names marked *verify* were not opened.

---

## Feature statement (AGENTS.md run sheet)

- **Outcome:** five cost lines can be added in one call and removed again in
  one call; a profile value set through an operation can be put back.
- **Why:** time-to-decision. The owner decided on 14 November that a chat
  change applies at once with Undo, so undo has to exist before chat can
  write. Program, awards and proposal decisions have undo today; Cost Plan
  and Profile do not.
- **Lane:** A (cost figures and profile values).
- **Speed is the owner's condition.** A screen write is not marked, so it
  does no new work beyond one trigger check per row. The existing cost and
  profile benches must pass unchanged, and their before and after figures are
  reported (Task 10).
- **Budgets (new rows):** `ops_apply` p50 ≤ 100 ms, p90 ≤ 250 ms (five cost
  lines added in one call). `change_undo` p50 ≤ 100 ms, p90 ≤ 250 ms.
- **Done means:** the tests in Tasks 1 to 9 pass, `tools/check.ps1` passes,
  and the bench reports both new rows inside budget with `costs_write`,
  `costs_bulk` and the profile edit path no slower than before.
- **Edge cases:** another org; another project in the same org; a row changed
  after the change (refuse); a cost plan frozen after the change (refuse); a
  second undo (refuse); an operation that fails mid-batch (nothing stays
  applied); an unknown operation or field; a batch over the cap.
- **Out of scope:** the model, the chat panel, the `read` and `find` tools,
  redo, Ctrl+Z for screen edits, operations for scope, accommodation,
  planning values, parts, work items, procurement, specifications, reports
  and documents (they stay on the pending list in Task 6), email.

## What exists (read before starting)

| Thing | Where |
|---|---|
| Every cost write opens with `costTx` and ends with `finishCost` | `internal/store/costs.go:140`, `:189` |
| Cost tables, frozen-plan trigger | `internal/db/migrations/021_cost_plan.sql` |
| Profile value write and reset | `SetUserValue`, `DeleteUserValue` in `internal/store/profile.go:301`, `:394` |
| Profile value validation lives in the handler | `putProfileValue`, `internal/httpapi/profile.go:556` |
| Program batch write, its error shape and its own undo | `internal/store/program_ops.go`, `program_undo.go` |
| Route table (a local map inside `Handler`) and budget names | `internal/httpapi/projects.go:111`, `routePaths` near `:329` |
| Routes that wake the program review | `internal/httpapi/program_review.go:147` |
| HTTP test helpers: `newApp`, `withProfile`, `a.member`, `m.createProject`, `m.call`, `m.getJSON` | `internal/httpapi/planning_test.go`, `access_sweep_test.go` |
| Test database and the full gate | `tools/check.ps1` |

Run Go tests with the repo toolchain and the test database:

```powershell
$env:SITEWISE_TEST_DATABASE_URL = 'postgres://sitewise@127.0.0.1:5433/sitewise_test?sslmode=disable'
$env:PATH = "$PWD\.tools\go\bin;$env:PATH"
go test -p 1 ./internal/store -run <Name> -count=1
```

---

### Task 0: Worktree and baseline

**Step 1:** Create a worktree on a new branch `chat/change-log` (use
superpowers:using-git-worktrees). Agent worktrees start from `origin/main`,
which is behind local `main`: run `git merge main` first.

**Step 2:** Copy the private data the gate needs (see the next-wave gate
note in project memory), then run `tools/check.ps1`.

**Step 3:** Record the bench figures for `costs_write`, `costs_bulk`,
`costs_delete` and the profile edit path. These are the "before" numbers for
Task 10. If the gate already fails on `main`, write down which checks fail
and stop for the owner.

### Task 1: The change log and the capture trigger

**Files:**
- Create: `internal/db/migrations/055_project_changes.sql` (take the next
  free number; another session may have used 055)
- Test: `internal/store/change_capture_test.go`

**Step 1: Write the failing test.** `TestCaptureRecordsMarkedWritesOnly`,
using a direct `pgx` connection to `SITEWISE_TEST_DATABASE_URL` and a project
with one cost line (build it with the fixture calls the cost store tests
use):

- With no mark set, update the line's label in a transaction. Assert
  `project_changes` and `project_change_rows` have no rows for the project.
- In a new transaction run
  `SELECT set_config('sitewise.change', $1, true)` with the JSON mark below,
  update the label and commit. Assert one `project_changes` row and one
  `project_change_rows` row with `op='UPDATE'`, `old->>'label'` the old label
  and `new->>'label'` the new one.
- Marked again, update only the row's `version`. Assert no new row (a
  version bump alone is not a change).
- Marked with another project's id in the mark, update the label. Assert the
  statement fails with SQLSTATE `23514`.

**Step 2:** Run it. Expected: FAIL, relation `project_changes` does not exist.

**Step 3: Write the migration.**

```sql
-- One row per change applied through the operations entry point. Rows hold
-- each record the change wrote, before and after, so it can be undone.
CREATE TABLE project_changes (
 org_id uuid NOT NULL REFERENCES orgs(id) ON DELETE CASCADE,
 project_id uuid NOT NULL,
 id uuid NOT NULL,
 n bigint GENERATED ALWAYS AS IDENTITY,
 actor uuid NOT NULL,
 via text NOT NULL CHECK (via IN ('chat','screen')),
 turn_id uuid,
 area text NOT NULL CHECK (area IN ('cost_plan','profile')),
 ops jsonb NOT NULL DEFAULT '[]' CHECK (jsonb_typeof(ops)='array'),
 undone_at timestamptz,
 at timestamptz NOT NULL DEFAULT now(),
 PRIMARY KEY (org_id,id),
 FOREIGN KEY (org_id,project_id) REFERENCES projects(org_id,id) ON DELETE CASCADE
);
CREATE INDEX project_changes_recent ON project_changes(org_id,project_id,n DESC);

CREATE TABLE project_change_rows (
 org_id uuid NOT NULL,
 change_id uuid NOT NULL,
 n bigint GENERATED ALWAYS AS IDENTITY,
 tbl text NOT NULL,
 op text NOT NULL CHECK (op IN ('INSERT','UPDATE','DELETE')),
 old jsonb,
 new jsonb,
 PRIMARY KEY (org_id,change_id,n),
 FOREIGN KEY (org_id,change_id) REFERENCES project_changes(org_id,id) ON DELETE CASCADE,
 CHECK ((op='INSERT')=(old IS NULL)), CHECK ((op='DELETE')=(new IS NULL))
);

-- The mark is transaction-local JSON: {"id","org","project","actor","via",
-- "turn","area"}. With no mark this returns at once, so a screen write pays
-- one setting lookup per row and nothing else. Trigger arguments name the
-- columns a later write may bump without it counting as a change.
CREATE FUNCTION capture_change() RETURNS trigger LANGUAGE plpgsql AS $$
DECLARE
 mark text := current_setting('sitewise.change', true);
 c jsonb; o jsonb; n jsonb; r jsonb; ignore text[] := '{}';
BEGIN
 IF mark IS NULL OR mark = '' THEN RETURN NULL; END IF;
 c := mark::jsonb;
 FOR i IN 0..TG_NARGS-1 LOOP ignore := ignore || TG_ARGV[i]; END LOOP;
 IF TG_OP <> 'INSERT' THEN o := to_jsonb(OLD); END IF;
 IF TG_OP <> 'DELETE' THEN n := to_jsonb(NEW); END IF;
 IF TG_OP = 'UPDATE' AND (o - ignore) = (n - ignore) THEN RETURN NULL; END IF;
 r := COALESCE(n, o);
 IF r->>'org_id' <> c->>'org' OR (r ? 'project_id' AND r->>'project_id' <> c->>'project') THEN
  RAISE EXCEPTION 'change crosses its project' USING ERRCODE='23514';
 END IF;
 INSERT INTO project_changes(org_id,project_id,id,actor,via,turn_id,area)
 VALUES ((c->>'org')::uuid,(c->>'project')::uuid,(c->>'id')::uuid,(c->>'actor')::uuid,
  c->>'via',NULLIF(c->>'turn','')::uuid,c->>'area')
 ON CONFLICT (org_id,id) DO NOTHING;
 INSERT INTO project_change_rows(org_id,change_id,tbl,op,old,new)
 VALUES ((c->>'org')::uuid,(c->>'id')::uuid,TG_TABLE_NAME,TG_OP,o,n);
 RETURN NULL;
END $$;

CREATE TRIGGER capture_change AFTER INSERT OR UPDATE OR DELETE ON cost_items
 FOR EACH ROW EXECUTE FUNCTION capture_change();
CREATE TRIGGER capture_change AFTER INSERT OR UPDATE OR DELETE ON cost_item_revisions
 FOR EACH ROW EXECUTE FUNCTION capture_change('version');
CREATE TRIGGER capture_change AFTER INSERT OR UPDATE OR DELETE ON cost_values
 FOR EACH ROW EXECUTE FUNCTION capture_change('version');
CREATE TRIGGER capture_change AFTER INSERT OR UPDATE OR DELETE ON scope_cost_links
 FOR EACH ROW EXECUTE FUNCTION capture_change();
CREATE TRIGGER capture_change AFTER INSERT OR UPDATE OR DELETE ON cost_suggestion_dismissals
 FOR EACH ROW EXECUTE FUNCTION capture_change();
CREATE TRIGGER capture_change AFTER INSERT OR UPDATE OR DELETE ON cost_plan_versions
 FOR EACH ROW EXECUTE FUNCTION capture_change('version');
CREATE TRIGGER capture_change AFTER INSERT OR UPDATE OR DELETE ON profile_user_values
 FOR EACH ROW EXECUTE FUNCTION capture_change('version','updated_at');
```

Check before running: every table above has an `org_id` column; none has a
generated column (a generated column cannot be restored from an image; if
one exists, stop and say so). Check `cost_line_sources` and any table added
after migration 021 that a cost write in `internal/store/cost*.go` touches;
add its trigger here.

**Step 4:** Run the test. Expected: PASS.

**Step 5:** Find where a project is deleted (*verify*:
`internal/store`, the function behind `DELETE /projects/{id}`) and confirm
the new tables go with it. Run `go test -p 1 ./internal/store ./internal/httpapi -run Delete -count=1`.

**Step 6:** Commit: `Change log tables and capture trigger for cost and profile values`.

### Task 2: Mark a store call's transactions

**Files:**
- Create: `internal/store/change.go`
- Modify: `internal/store/store.go:136-160` (the `pool` field and `Open`)
- Test: `internal/store/change_test.go`

**Step 1: Write the failing test.** `TestMarkedCostWriteIsCaptured`: call
`PatchCostItem` with `store.WithChange(ctx, store.Change{...})` and assert
one change with rows for `cost_item_revisions`; call it again with a plain
context and assert no second change.

**Step 2:** Run. Expected: FAIL, `store.WithChange` undefined.

**Step 3: Implement.**

```go
// Change marks every transaction begun with its context, so the capture
// triggers record the rows it writes under one undoable change.
type Change struct {
	ID, OrgID, ProjectID, Actor string
	Via    string // "chat" or "screen"
	TurnID string // groups one chat turn's changes; may be empty
	Area   string // "cost_plan" or "profile"
}

type changeKey struct{}

func WithChange(ctx context.Context, c Change) context.Context {
	return context.WithValue(ctx, changeKey{}, c)
}

// markedPool is the pool every store function uses. Begin sets the
// transaction-local mark when the context carries a change.
type markedPool struct{ *pgxpool.Pool }

func (p markedPool) Begin(ctx context.Context) (pgx.Tx, error) {
	return p.BeginTx(ctx, pgx.TxOptions{})
}

func (p markedPool) BeginTx(ctx context.Context, o pgx.TxOptions) (pgx.Tx, error) {
	tx, err := p.Pool.BeginTx(ctx, o)
	if err != nil {
		return nil, err
	}
	c, ok := ctx.Value(changeKey{}).(Change)
	if !ok || o.AccessMode == pgx.ReadOnly {
		return tx, nil
	}
	mark, _ := json.Marshal(map[string]string{"id": c.ID, "org": c.OrgID, "project": c.ProjectID,
		"actor": c.Actor, "via": c.Via, "turn": c.TurnID, "area": c.Area})
	if _, err := tx.Exec(ctx, `SELECT set_config('sitewise.change',$1,true)`, string(mark)); err != nil {
		tx.Rollback(ctx)
		return nil, err
	}
	return tx, nil
}
```

Change `Store.pool` to `markedPool` and wrap the pool in `Open`. The 144
`s.pool.Begin` call sites compile unchanged. Anything that needs the raw
pool takes `s.pool.Pool`.

A write made with `s.pool.Exec` outside a transaction is not marked and is
not captured. Task 4's apply-then-undo tests catch any operation that writes
that way; move that write into a transaction when they do.

**Step 4:** Run the test, then `go build ./...` and
`go test -p 1 ./internal/store -count=1`. Expected: PASS.

**Step 5:** Commit: `Store: mark the transactions of a change`.

### Task 3: Undo a change

**Files:**
- Create: `internal/store/change_undo.go`
- Test: `internal/store/change_undo_test.go`

**Step 1: Write the failing tests.**

- `TestCapturedTablesMatchDatabase`: the Go registry below names exactly the
  tables that carry the `capture_change` trigger (`pg_trigger`), each `Key`
  equals the table's primary key (`pg_index`), and each `Ignore` equals the
  trigger's arguments. This stops the registry and the migration drifting.
- `TestUndoRestoresCostEdit`: read the plan; marked `PatchCostItem` changes a
  label; `UndoChange`; read again. The two reads are equal once `version`
  fields are removed.
- `TestUndoRestoresAddedAndDeletedLines`: marked `CreateCostItem` then undo
  leaves no line; marked `BulkCostItems` delete of a group line with children
  and values, then undo, restores every line and value.
- `TestUndoRefusals`: a second undo returns `ErrChangeUndone`; after an
  unmarked edit of the same line, undo returns `ErrChangedSince` and writes
  nothing; another org or another project gets `ErrNotFound`; another user
  in the same org gets `ErrNotFound`; after `BaselineCostPlan` freezes the
  plan, undo returns `ErrChangedSince`.

**Step 2:** Run. Expected: FAIL, `UndoChange` undefined.

**Step 3: Implement.**

```go
type capturedTable struct {
	Key    []string // primary key columns
	Ignore []string // the trigger's arguments
	Area   string   // which finisher runs after an undo: "costs" or "profile"
}

// captured must match migration 055; TestCapturedTablesMatchDatabase checks it.
var captured = map[string]capturedTable{ /* one entry per trigger in Task 1 */ }

var (
	ErrChangeUndone = errors.New("already undone")
	ErrChangedSince = errors.New(MsgUndoChanged) // "Can't undo: changed since"
)

// UndoChange puts back every row a change wrote, newest first, if each row
// is still as the change left it. Only the user who made it may undo it.
func (s *Store) UndoChange(ctx context.Context, org, project, actor, id string) ([]string, error)
```

Inside one transaction, with a plain (unmarked) context:

1. `lockProject`. Select the change `FOR UPDATE` by `org_id`, `project_id`,
   `id` and `actor`. No row: `ErrNotFound`. `undone_at` set: `ErrChangeUndone`.
2. Load its rows ordered by `n`. Refuse any `tbl` not in `captured` (a
   table name never reaches SQL from row data).
3. Changed-since check. For each table and key, take the last row. If its
   `op` is `DELETE` the record must be absent. Otherwise
   `to_jsonb(current) - ignore` must equal `new - ignore`. Any miss:
   `ErrChangedSince`.
4. Replay from the highest `n` down:
   - `INSERT`: delete the record by key.
   - `DELETE`: `INSERT INTO <tbl> SELECT * FROM jsonb_populate_record(NULL::<tbl>, $1)` with `old`.
   - `UPDATE`: set every column except the key and the ignored columns from
     `jsonb_populate_record(NULL::<tbl>, $1)`; where `version` is ignored,
     set `version = version + 1`. Read the column list once per table from
     `information_schema.columns`.
5. A constraint error during replay (SQLSTATE class 23, which includes the
   frozen-plan trigger) rolls back and returns `ErrChangedSince`.
6. Set `undone_at = clock_timestamp()`.
7. Run the finisher for each area touched. Costs: split the body of
   `finishCost` into `touchCost` (bump the draft's version, `BumpRevision`
   `"costs"`, append the `costs` event) and call it; `finishCost` calls it
   too. Profile: call what `SetUserValue` calls before it commits
   (*verify*: `finishProfileWriteLocked`, `internal/store/revisions.go:98`).
8. Commit and return the areas touched.

**Step 4:** Run the tests. Expected: PASS.

**Step 5:** Commit: `Store: undo a captured change`.

### Task 4: The undo route

**Files:**
- Create: `internal/httpapi/changes.go`
- Modify: `internal/httpapi/projects.go` (route map near `:111`, `routePaths`
  near `:329`), `internal/httpapi/program_review.go:147` (the review map,
  because undoing a profile value changes what the program is built from),
  `bench/budgets.json`
- Test: `internal/httpapi/changes_test.go`

**Step 1: Write the failing test.** `TestChangeUndoRoute`: a marked change
made through the store, then `POST /api/projects/{id}/changes/{change}/undo`
returns 200 `{"undone":true,"areas":["costs"]}`. A second call returns 409.
After a later edit it returns 422 with
`{"error":"changed_since","message":"Can't undo: changed since"}`. A member
of another org gets 404. A malformed change id gets 404.

**Step 2:** Run. Expected: FAIL with 404 on the first call.

**Step 3: Implement** `postChangeUndo` with the session check `patchCostItem`
uses (`deliverySession(w, r, d, true)`), `uuidPattern` on both ids, and the
error mapping above. Add the route, its `routePaths` name `change_undo`, the
review-map entry, and the budget row
`{"name":"change_undo","p50_us":100000,"p90_us":250000,"note":"undo one applied change: rows restored and the area's rollups refreshed in one transaction; no Jev"}`.

**Step 4:** Run the test. Expected: PASS.

**Step 5:** Commit: `Undo route for applied changes`.

### Task 5: Route table as a function (refactor only)

**Files:**
- Modify: `internal/httpapi/projects.go:87-300`

**Step 1:** Move the `routes` map literal out of `Handler` into
`func apiRoutes() map[string]func(http.ResponseWriter, *http.Request, Deps)`,
including the account and billing routes merged at `:284-289`. `Handler`
calls it. No behaviour changes.

**Step 2:** Run `go test -p 1 ./internal/httpapi -count=1`. Expected: PASS,
as before the move.

**Step 3:** Commit: `httpapi: route table as a function`.

### Task 6: Operations registry and the coverage test

**Files:**
- Create: `internal/httpapi/ops.go`, `internal/httpapi/ops_routes.go`
- Test: `internal/httpapi/ops_coverage_internal_test.go`

**Step 1: Write the failing test.**

```go
// Every write route is an operation, is excluded with a reason, or is
// pending with its area. A new write route fails here until it is classed.
func TestWriteRoutesHaveOperations(t *testing.T) {
	covered := map[string]bool{}
	for _, op := range operations() {
		for _, r := range op.Routes {
			covered[r] = true
		}
	}
	routes := apiRoutes()
	pending := 0
	for pattern := range routes {
		if strings.HasPrefix(pattern, "GET ") {
			continue
		}
		homes := 0
		if covered[pattern] {
			homes++
		}
		if noOperation[pattern] != "" {
			homes++
		}
		if pendingOperation[pattern] != "" {
			homes++
			pending++
		}
		if homes != 1 {
			t.Errorf("%s: want exactly one of operation, noOperation, pendingOperation; got %d", pattern, homes)
		}
	}
	for _, list := range []map[string]string{noOperation, pendingOperation} {
		for pattern := range list {
			if routes[pattern] == nil {
				t.Errorf("%s is listed but is not a route", pattern)
			}
		}
	}
	for pattern := range covered {
		if routes[pattern] == nil {
			t.Errorf("an operation names %s, which is not a route", pattern)
		}
	}
	t.Logf("%d write routes are pending an operation", pending)
}
```

**Step 2:** Run. Expected: FAIL, `operations` undefined.

**Step 3: Implement.**

```go
// operation is one named edit. Run does what the screen's handler does for
// the same edit and calls the same store write.
type operation struct {
	Name   string          // "cost_plan.add_line"
	About  string          // one sentence, read by the agent later
	Fields json.RawMessage // JSON Schema of the fields
	Routes []string        // the write routes this stands for
	Click  bool            // prepared here, done by the user's click (design 4.3)
	Run    func(ctx context.Context, d Deps, who opActor, project string, fields json.RawMessage) (summary string, err error)
}

type opActor struct{ OrgID, UserID string }

// opError is a refused batch, in Program's shape.
type opError struct {
	Index   int    `json:"op_index"`
	Field   string `json:"field,omitempty"`
	Message string `json:"message"`
}
```

`ops_routes.go` holds `noOperation` (pattern to reason: sign-in, sign-up,
passwords, billing, Stripe, team and invitations, session end, file upload,
project delete and archive, the undo route itself, `POST /projects/{id}/ops`)
and `pendingOperation` (pattern to area) for every other write route. Start
with `operations()` returning nil and every remaining route pending, so the
test passes; each later task moves routes from pending to an operation.

**Step 4:** Run the test. Expected: PASS, logging about 115 pending.

**Step 5:** Commit: `Operations registry and write-route coverage test`.

### Task 7: Apply a batch

**Files:**
- Modify: `internal/httpapi/ops.go`, `internal/httpapi/projects.go` (route,
  `routePaths` name `ops_apply`), `bench/budgets.json`
- Test: `internal/httpapi/ops_apply_test.go`

**Step 1: Write the failing tests** against
`POST /api/projects/{id}/ops` with body
`{"area":"cost_plan","turn_id":"","ops":[{"op":"...","fields":{...}}]}`,
using the first real operation from Task 8 (write Task 8's `add_line` here if
needed to have one):

- Two valid operations return 200 with
  `{"change_id":"…","undo":{"kind":"change"},"results":[{"op":"…","summary":"…"},…]}`.
- An unknown operation, an operation from another area, a `Click` operation,
  an empty batch and a batch of 51 each return 422 as an `opError` and write
  nothing.
- A batch whose second operation is refused returns 422 with `op_index: 1`,
  and the plan reads exactly as before the call (the first operation was
  undone).
- Another org gets 404.

**Step 2:** Run. Expected: FAIL with 404.

**Step 3: Implement** `applyOps(ctx, d, who, project, area, turnID, calls)`:
validate the batch; make a change id with the store's id function; run each
operation in order under `store.WithChange`; on the first error call
`UndoChange` for what was captured and return the `opError` with that index;
then store the batch as sent in `project_changes.ops`. Add the handler, the
route and the budget row
`{"name":"ops_apply","p50_us":100000,"p90_us":250000,"note":"five cost lines added in one operations call, each the existing cost write; no Jev"}`.

A batch is several store transactions, so "nothing stays applied" is done by
undoing, not by one transaction. Say so in the function comment.

**Step 4:** Run the tests. Expected: PASS.

**Step 5:** Commit: `Apply a batch of operations with undo on failure`.

### Task 8: Cost Plan operations

**Files:**
- Create: `internal/httpapi/ops_costs.go`
- Modify: `internal/httpapi/ops_routes.go` (move each route out of pending)
- Test: `internal/httpapi/ops_costs_test.go`

Each operation fills `CostWrite` itself from the current draft
(`plan_version_id` and `version`), so a caller never sends a version. Each
decodes its fields with unknown fields refused, and calls one store function.

| Operation | Store call | Route it stands for |
|---|---|---|
| `cost_plan.add_line` | `CreateCostItem` | `POST …/cost-plan/items` |
| `cost_plan.edit_line` | `PatchCostItem` | `PATCH …/cost-plan/items/{item}` |
| `cost_plan.set_amount` | `PutCostValue` | `PUT …/items/{item}/values/{metric}` |
| `cost_plan.delete_lines`, `duplicate_lines`, `set_lines` | `BulkCostItems` | `POST …/items/bulk`, `DELETE …/items/{item}` |
| `cost_plan.reorder` | `ReorderCostItems` | `PUT …/cost-plan/order` |
| `cost_plan.subdivide` | `SubdivideCostItem` | `POST …/items/{item}/subdivide` |
| `cost_plan.settings` | `SaveCostSettings` | `PUT …/cost-plan` |
| `cost_plan.accept_suggestions` | `AcceptCostSuggestions` | both `…/suggestions/…accept` routes |
| `cost_plan.dismiss_suggestion`, `restore_suggestion` | `DismissCostSuggestion` | both `…/dismiss` routes |
| `cost_plan.link_scope` | `LinkScopeCost` | `PUT …/cost-plan/links` |
| `cost_plan.apply_benchmark` | `ApplyCostBenchmark` | `POST …/items/{item}/benchmarks` |
| `cost_plan.prepare` | `PrepareCostPlan` | `POST …/cost-plan/prepare` |
| `cost_plan.freeze` (`Click: true`) | none | `POST …/cost-plan/baseline` |

`set_amount` takes an amount only from its fields; it never calculates one
(the existing `calculate` flag stays a screen feature until chat is designed
to use it).

**For each operation, in this order:**

1. Write the failing test: apply it through `POST …/ops`, assert the plan
   shows the edit, call the undo route, assert the plan equals the plan
   before once `version` fields are removed. For `delete_lines` include a
   line that has a value and a dismissed default.
2. Run it. Expected: FAIL, unknown operation.
3. Write the operation and move its route out of `pendingOperation`.
4. Run it and `TestWriteRoutesHaveOperations`. Expected: PASS. If the undo
   leaves a difference, a write touched a table with no trigger or wrote
   outside a transaction: add the trigger (new migration) or move the write.
5. Commit, one operation or one small group per commit.

### Task 9: Profile value operation and Program pass-through

**Files:**
- Modify: `internal/httpapi/profile.go:556-653`
- Create: `internal/httpapi/ops_profile.go`, `internal/httpapi/ops_program.go`
- Test: `internal/httpapi/ops_profile_test.go`, `internal/httpapi/ops_program_test.go`

**Step 1 (refactor only):** Move everything in `putProfileValue` after the
body is decoded (validation, the favourable-value rule, scope lookup,
`EnsureWholePart`, the store call) into
`setProfileValue(ctx, deps, who, projectID, key string, in profileValueInput) (current int64, status int, msg string)`.
The handler calls it and maps the result as now. Run
`go test -p 1 ./internal/httpapi -run Profile -count=1`. Expected: PASS
unchanged. Commit.

**Step 2:** Test then write `profile.set_value` (fields: `key`, `part_id`,
`value`, `note`, `unknown`, `reset`). It calls `setProfileValue` with origin
`user`, and fills the row version itself. Tests: set then undo restores the
profile row; a reset then undo restores the user's value; an unknown key and
a favourable planning value without the user's word are refused with the
handler's own messages. Move `PUT /projects/{id}/profile/{key}` out of
pending. Commit.

**Step 3:** Test then write `program.edit`. For area `program`, `applyOps`
passes the whole batch to `ApplyProgramOps` as one call (it is already one
transaction with its own undo), reading the program id and version itself,
and returns `"undo":{"kind":"program"}`. A refused batch returns Program's
own `op_index`, `field` and `message`. Test: five `add` rows under a stage in
one call, then `POST …/program/undo` removes all five. Move
`POST /projects/{id}/program/ops` out of pending. Commit.

### Task 10: Bench, gate and report

**Files:**
- Modify: `cmd/intake-bench` (*verify* where `costs_bulk` is benched and
  follow it), `AGENTS.md` (add the operations call and undo to the paths
  that do not call Jev)

**Step 1:** Add bench cases for `ops_apply` (five `cost_plan.add_line` in one
call) and `change_undo` (undo that change). Run the bench.

**Step 2:** Run `tools/check.ps1`. Expected: PASS.

**Step 3:** Add a "Result" section to this plan, as the other plans have:
what was built, the tests, and timing. Report `ops_apply` and `change_undo`
against budget, and `costs_write`, `costs_bulk`, `costs_delete` and the
profile edit path before (Task 0) and after. If any existing path is slower
beyond bench noise, stop and say what the trigger costs before merging.

**Step 4:** Report in plain language what calls what and what could fail in
production, then commit.

---

## Risks to watch

- **Trigger cost on screen writes.** One function call per written row that
  returns after a setting lookup. Task 10 measures it; the gate is the
  existing budgets.
- **Writes outside a transaction** are not captured (Task 2). The
  apply-then-undo test for each operation is the guard.
- **A new table a cost or profile write starts using** has no trigger until
  someone adds one. The same tests catch it for existing operations;
  `TestCapturedTablesMatchDatabase` keeps the registry honest.
- **Restore order and foreign keys.** Replaying newest first undoes child
  rows before parents. A record something else now depends on (an invoice
  mapped to a new line) fails the restore and is reported as changed since.

## Follow-ups, not in this plan

- Operations and capture for scope, accommodation, planning values, parts
  and work items; then procurement, specifications, reports and documents
  after the tender comparison work lands.
- An Undo for screen edits on Cost Plan (mark the handlers' contexts).
- The `read` and `find` tools, the agent loop and the chat strip.

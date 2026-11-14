package store

import (
	"context"
	"encoding/json"
	"errors"
	"sort"
	"strconv"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

type capturedTable struct {
	Key    []string // primary key columns
	Ignore []string // the trigger's arguments: columns that may move without counting
	Area   string   // which finisher runs after an undo: "costs", "profile" or "procurement"
}

// captured names every table that carries the capture_change trigger. It
// must match the migrations; TestCapturedTablesMatchDatabase checks it.
var captured = map[string]capturedTable{
	"cost_items":                 {Key: []string{"org_id", "id"}, Area: "costs"},
	"cost_item_revisions":        {Key: []string{"org_id", "plan_version_id", "cost_item_id"}, Ignore: []string{"version"}, Area: "costs"},
	"cost_values":                {Key: []string{"org_id", "plan_version_id", "cost_item_id", "metric"}, Ignore: []string{"version"}, Area: "costs"},
	"scope_cost_links":           {Key: []string{"org_id", "project_id", "plan_version_id", "package_scope_item_id", "cost_item_id"}, Area: "costs"},
	"cost_suggestion_dismissals": {Key: []string{"org_id", "project_id", "source_key"}, Area: "costs"},
	"cost_plan_versions":         {Key: []string{"org_id", "id"}, Ignore: []string{"version"}, Area: "costs"},
	"profile_user_values":        {Key: []string{"org_id", "id"}, Ignore: []string{"version", "updated_at"}, Area: "profile"},
	"tender_lines":               {Key: []string{"org_id", "id"}, Ignore: []string{"updated_at"}, Area: "procurement"},
	"tender_adjustments":         {Key: []string{"org_id", "id"}, Ignore: []string{"updated_at"}, Area: "procurement"},
	"tender_adjustment_amounts":  {Key: []string{"org_id", "adjustment_id", "submission_id"}, Ignore: []string{"updated_at"}, Area: "procurement"},
}

var (
	// ErrChangeUndone: the change was already undone.
	ErrChangeUndone = errors.New("already undone")
	// ErrChangedSince: a row the change wrote has changed since, so putting
	// it back would lose a later edit.
	ErrChangedSince = errors.New(MsgUndoChanged)
)

type changeRow struct {
	Table    string
	Op       string
	Old, New []byte
}

// UndoChange puts back every row a change wrote, newest first, if each row
// is still as the change left it. Only the user who made it may undo it.
// It returns the areas it touched.
func (s *Store) UndoChange(ctx context.Context, org, project, actor, id string) ([]string, error) {
	// The undo itself is not a change: its transaction is never marked.
	tx, err := s.pool.Pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)
	if err := lockProject(ctx, tx, org, project); err != nil {
		return nil, err
	}
	var undone bool
	err = tx.QueryRow(ctx, `SELECT undone_at IS NOT NULL FROM project_changes WHERE org_id=$1::uuid AND project_id::text=$2 AND id::text=$3 AND actor::text=$4 FOR UPDATE`,
		org, project, id, actor).Scan(&undone)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	if undone {
		return nil, ErrChangeUndone
	}
	rows, err := tx.Query(ctx, `SELECT tbl, op, old, new FROM project_change_rows WHERE org_id=$1::uuid AND change_id::text=$2 ORDER BY n`, org, id)
	if err != nil {
		return nil, err
	}
	var log []changeRow
	for rows.Next() {
		var r changeRow
		if err := rows.Scan(&r.Table, &r.Op, &r.Old, &r.New); err != nil {
			rows.Close()
			return nil, err
		}
		// A table name never reaches SQL from row data unless it is registered.
		if _, ok := captured[r.Table]; !ok {
			rows.Close()
			return nil, errors.New("change log names a table that is not captured")
		}
		log = append(log, r)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return nil, err
	}
	// Changed-since check: the last image of each record must be what is there now.
	last := map[string]changeRow{}
	var order []string
	for _, r := range log {
		image := r.New
		if r.Op == "DELETE" {
			image = r.Old
		}
		k := r.Table + "|" + keyOf(captured[r.Table], image)
		if _, seen := last[k]; !seen {
			order = append(order, k)
		}
		last[k] = r
	}
	for _, k := range order {
		r := last[k]
		t := captured[r.Table]
		image := r.New
		if r.Op == "DELETE" {
			image = r.Old
		}
		where, args := keyWhere(t, image, 1)
		var same *bool
		if r.Op == "DELETE" {
			var exists bool
			if err := tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM `+r.Table+` t WHERE `+where+`)`, args...).Scan(&exists); err != nil {
				return nil, err
			}
			ok := !exists
			same = &ok
		} else {
			args = append(args, string(r.New), t.Ignore)
			n := len(args)
			var ok bool
			err := tx.QueryRow(ctx, `SELECT (to_jsonb(t) - $`+itoa(n)+`::text[]) = ($`+itoa(n-1)+`::jsonb - $`+itoa(n)+`::text[]) FROM `+r.Table+` t WHERE `+where, args...).Scan(&ok)
			if errors.Is(err, pgx.ErrNoRows) {
				ok = false
			} else if err != nil {
				return nil, err
			}
			same = &ok
		}
		if !*same {
			return nil, ErrChangedSince
		}
	}
	// Replay newest first, so child rows go back before their parents need them.
	columns := map[string][]string{}
	for i := len(log) - 1; i >= 0; i-- {
		r := log[i]
		t := captured[r.Table]
		var err error
		switch r.Op {
		case "INSERT":
			where, args := keyWhere(t, r.New, 1)
			_, err = tx.Exec(ctx, `DELETE FROM `+r.Table+` t WHERE `+where, args...)
		case "DELETE":
			_, err = tx.Exec(ctx, `INSERT INTO `+r.Table+` SELECT * FROM jsonb_populate_record(NULL::`+r.Table+`, $1::jsonb)`, string(r.Old))
		case "UPDATE":
			cols := columns[r.Table]
			if cols == nil {
				if cols, err = tableColumns(ctx, tx, r.Table); err != nil {
					return nil, err
				}
				columns[r.Table] = cols
			}
			var sets []string
			for _, c := range cols {
				switch {
				case contains(t.Key, c):
				case c == "version" && contains(t.Ignore, c):
					sets = append(sets, `version = t.version + 1`)
				case contains(t.Ignore, c):
				default:
					sets = append(sets, c+` = r.`+c)
				}
			}
			where, args := keyWhere(t, r.Old, 2)
			args = append([]any{string(r.Old)}, args...)
			_, err = tx.Exec(ctx, `UPDATE `+r.Table+` t SET `+strings.Join(sets, ", ")+` FROM jsonb_populate_record(NULL::`+r.Table+`, $1::jsonb) r WHERE `+where, args...)
		}
		if err != nil {
			// A constraint that refuses the restore (a frozen plan, a record
			// something else now depends on) means things changed since.
			var pg *pgconn.PgError
			if errors.As(err, &pg) && strings.HasPrefix(pg.Code, "23") {
				return nil, ErrChangedSince
			}
			return nil, err
		}
	}
	if _, err := tx.Exec(ctx, `UPDATE project_changes SET undone_at = clock_timestamp() WHERE org_id=$1::uuid AND id::text=$2`, org, id); err != nil {
		return nil, err
	}
	touched := map[string]bool{}
	for _, r := range log {
		touched[captured[r.Table].Area] = true
	}
	var areas []string
	for a := range touched {
		areas = append(areas, a)
	}
	sort.Strings(areas)
	for _, a := range areas {
		var err error
		switch a {
		case "costs":
			err = s.touchCostsAfterUndo(ctx, tx, org, project, log)
		case "profile":
			err = s.touchProfileAfterUndo(ctx, tx, org, project)
		case "procurement":
			err = s.refreshTenderAfterUndo(ctx, tx, org, project, log)
		}
		if err != nil {
			var pg *pgconn.PgError
			if errors.As(err, &pg) && strings.HasPrefix(pg.Code, "23") {
				return nil, ErrChangedSince
			}
			return nil, err
		}
	}
	return areas, tx.Commit(ctx)
}

func itoa(n int) string { return strconv.Itoa(n) }

func contains(list []string, v string) bool {
	for _, x := range list {
		if x == v {
			return true
		}
	}
	return false
}

// keyOf is a record's primary key as text, from its row image.
func keyOf(t capturedTable, image []byte) string {
	var m map[string]any
	_ = json.Unmarshal(image, &m)
	parts := make([]string, len(t.Key))
	for i, k := range t.Key {
		b, _ := json.Marshal(m[k])
		parts[i] = string(b)
	}
	return strings.Join(parts, "|")
}

// keyWhere is "t.k1::text = $n AND ..." for a record's primary key.
func keyWhere(t capturedTable, image []byte, first int) (string, []any) {
	var m map[string]any
	_ = json.Unmarshal(image, &m)
	var conds []string
	var args []any
	for i, k := range t.Key {
		v := ""
		switch x := m[k].(type) {
		case string:
			v = x
		default:
			b, _ := json.Marshal(x)
			v = string(b)
		}
		conds = append(conds, "t."+k+"::text = $"+itoa(first+i))
		args = append(args, v)
	}
	return strings.Join(conds, " AND "), args
}

func tableColumns(ctx context.Context, tx pgx.Tx, table string) ([]string, error) {
	rows, err := tx.Query(ctx, `SELECT column_name FROM information_schema.columns WHERE table_schema = current_schema() AND table_name = $1 ORDER BY ordinal_position`, table)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var cols []string
	for rows.Next() {
		var c string
		if err := rows.Scan(&c); err != nil {
			return nil, err
		}
		cols = append(cols, c)
	}
	return cols, rows.Err()
}

// rowField reads one text field from the images of the rows a change wrote.
func rowField(log []changeRow, tables map[string]string) []string {
	seen := map[string]bool{}
	var out []string
	for _, r := range log {
		field, ok := tables[r.Table]
		if !ok {
			continue
		}
		for _, image := range [][]byte{r.Old, r.New} {
			var m map[string]any
			if len(image) == 0 || json.Unmarshal(image, &m) != nil {
				continue
			}
			if v, ok := m[field].(string); ok && v != "" && !seen[v] {
				seen[v] = true
				out = append(out, v)
			}
		}
	}
	return out
}

// touchCostsAfterUndo is what every cost write does before it commits:
// bump each touched plan's version, the costs revision, and tell screens.
func (s *Store) touchCostsAfterUndo(ctx context.Context, tx pgx.Tx, org, project string, log []changeRow) error {
	plans := rowField(log, map[string]string{"cost_plan_versions": "id", "cost_item_revisions": "plan_version_id", "cost_values": "plan_version_id",
		"scope_cost_links": "plan_version_id", "cost_suggestion_dismissals": "plan_version_id", "cost_items": "plan_version_id"})
	for _, id := range plans {
		if err := s.touchCost(ctx, tx, org, project, id); err != nil {
			return err
		}
	}
	if len(plans) == 0 {
		return BumpRevision(ctx, tx, org, project, "costs")
	}
	return nil
}

// touchProfileAfterUndo is what SetUserValue does before it commits.
func (s *Store) touchProfileAfterUndo(ctx context.Context, tx pgx.Tx, org, project string) error {
	if err := bumpRevisionLocked(ctx, tx, org, project, "profile_inputs"); err != nil {
		return err
	}
	if b := s.profileBuild; b != nil {
		if b.proposalError != nil {
			return b.proposalError
		}
		return s.rebuildProfileProjectionLocked(ctx, tx, org, project, *b, b.proposalEvaluator)
	}
	return nil
}

// refreshTenderAfterUndo brings everything derived from tender lines and
// adds and subs back in step: the tender sum, flags, price cells and
// suggestions of every submission and package the change touched.
func (s *Store) refreshTenderAfterUndo(ctx context.Context, tx pgx.Tx, org, project string, log []changeRow) error {
	subs := rowField(log, map[string]string{"tender_lines": "submission_id", "tender_adjustment_amounts": "submission_id"})
	pkgs := map[string]bool{}
	for _, id := range rowField(log, map[string]string{"tender_adjustments": "package_id"}) {
		pkgs[id] = true
	}
	for _, sub := range subs {
		var pkg string
		err := tx.QueryRow(ctx, `SELECT COALESCE(package_id::text,'') FROM submissions WHERE org_id=$1::uuid AND project_id::text=$2 AND id::text=$3`, org, project, sub).Scan(&pkg)
		if errors.Is(err, pgx.ErrNoRows) || pkg == "" {
			continue
		}
		if err != nil {
			return err
		}
		if err := refreshTender(ctx, tx, org, project, pkg, sub, nil); err != nil {
			return err
		}
		pkgs[pkg] = true
	}
	for pkg := range pkgs {
		if err := syncSuggestions(ctx, tx, org, project, pkg); err != nil {
			return err
		}
		if err := touchTender(ctx, tx, s, org, project, pkg, "", "undone"); err != nil {
			return err
		}
	}
	return nil
}

// CapturedTables is the registry of captured tables, for the test that
// holds it to the database.
func CapturedTables() map[string]struct{ Key, Ignore []string } {
	out := map[string]struct{ Key, Ignore []string }{}
	for name, t := range captured {
		out[name] = struct{ Key, Ignore []string }{t.Key, t.Ignore}
	}
	return out
}

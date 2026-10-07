package db

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"maps"
	"os"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

type sweepFK struct {
	table, name, parent, definition string
	columns, parentColumns          []string
}

// Inventory migrations rather than a hand-maintained list: a new composite
// reference must acquire a fixture and an exact-constraint rejection here.
func TestNextWaveCompositeForeignKeys(t *testing.T) {
	ctx := context.Background()
	cfg, err := pgx.ParseConfig(os.Getenv("SITEWISE_TEST_DATABASE_URL"))
	if err != nil || cfg.Database != "sitewise_test" {
		t.Fatal("dedicated sitewise_test database required")
	}
	conn, err := pgx.ConnectConfig(ctx, cfg)
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close(ctx)
	tx, err := conn.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer tx.Rollback(ctx)
	run := func(sql string, args ...any) {
		t.Helper()
		if _, err := tx.Exec(ctx, sql, args...); err != nil {
			t.Fatal(err)
		}
	}
	schema := fmt.Sprintf("fk_sweep_%d", time.Now().UnixNano())
	run("CREATE SCHEMA " + pgx.Identifier{schema}.Sanitize())
	run("SET LOCAL search_path TO " + pgx.Identifier{schema}.Sanitize() + ",public")
	inventory := func() map[string]sweepFK {
		rows, err := tx.Query(ctx, `SELECT child.relname,c.conname,parent.relname,pg_get_constraintdef(c.oid),
 ARRAY(SELECT a.attname::text FROM unnest(c.conkey) WITH ORDINALITY k(num,ord) JOIN pg_attribute a ON a.attrelid=c.conrelid AND a.attnum=k.num ORDER BY k.ord),
 ARRAY(SELECT a.attname::text FROM unnest(c.confkey) WITH ORDINALITY k(num,ord) JOIN pg_attribute a ON a.attrelid=c.confrelid AND a.attnum=k.num ORDER BY k.ord)
 FROM pg_constraint c JOIN pg_class child ON child.oid=c.conrelid JOIN pg_class parent ON parent.oid=c.confrelid
 WHERE c.contype='f' AND cardinality(c.conkey)>1 AND c.connamespace=$1::regnamespace`, schema)
		if err != nil {
			t.Fatal(err)
		}
		defer rows.Close()
		out := map[string]sweepFK{}
		for rows.Next() {
			var f sweepFK
			if err := rows.Scan(&f.table, &f.name, &f.parent, &f.definition, &f.columns, &f.parentColumns); err != nil {
				t.Fatal(err)
			}
			out[f.name] = f
		}
		if err := rows.Err(); err != nil {
			t.Fatal(err)
		}
		return out
	}
	names, err := fs.Glob(migrationFiles, "migrations/*.sql")
	if err != nil {
		t.Fatal(err)
	}
	sort.Strings(names)
	var before map[string]sweepFK
	for _, name := range names {
		if before == nil && name >= "migrations/011_sites.sql" {
			before = inventory()
		}
		body, err := migrationFiles.ReadFile(name)
		if err != nil {
			t.Fatal(err)
		}
		run(string(body))
	}
	after := inventory()
	id := func(n int) string { return fmt.Sprintf("91000000-0000-4000-8000-%012d", n) }
	type row = map[string]any
	insert := func(q pgx.Tx, table string, data row) error {
		keys := make([]string, 0, len(data))
		for key := range data {
			keys = append(keys, key)
		}
		sort.Strings(keys)
		columns, values := []string{}, []string{}
		for _, key := range keys {
			name := pgx.Identifier{key}.Sanitize()
			columns = append(columns, name)
			values = append(values, "r."+name)
		}
		raw, err := json.Marshal(data)
		if err != nil {
			return err
		}
		_, err = q.Exec(ctx, "INSERT INTO "+pgx.Identifier{table}.Sanitize()+" ("+strings.Join(columns, ",")+") SELECT "+strings.Join(values, ",")+" FROM jsonb_populate_record(NULL::"+pgx.Identifier{table}.Sanitize()+",$1::jsonb) r", raw)
		return err
	}
	fixtures := []map[string]row{}
	for n := 1; n <= 3; n++ {
		org := id(1)
		if n == 3 {
			org = id(2)
		}
		if n != 2 {
			run(`INSERT INTO orgs(id,name) VALUES($1,'FK sweep')`, org)
		}
		p, s, u, part, w, pkg, stage, scope, d, report, version, planning := id(100+n), id(200+n), id(300+n), id(400+n), id(500+n), id(600+n), id(700+n), id(800+n), id(900+n), id(1000+n), id(1100+n), id(1200+n)
		recordID := fmt.Sprintf("ic.seed%d", n)
		proposalKey := recordID + "||||0"
		f := map[string]row{
			"users":                   {"org_id": org, "id": u, "email": fmt.Sprintf("fk%d@example.test", n)},
			"sites":                   {"org_id": org, "id": s, "label": "Site"},
			"projects":                {"org_id": org, "id": p, "site_id": s, "name": "Project"},
			"files":                   {"org_id": org, "id": id(1800 + n), "project_id": p, "sha256": "\\x" + strings.Repeat("01", 32), "byte_size": 1, "media_type": "text/plain"},
			"documents":               {"org_id": org, "id": id(1900 + n), "project_id": p, "file_id": id(1800 + n), "filename": "source.txt", "status": "filed"},
			"profile_facts":           {"org_id": org, "id": id(2000 + n), "project_id": p, "document_id": id(1900 + n), "question_id": "hdr.work_type", "value": "new", "decided_by": "rule", "question_version": "scope-test"},
			"project_parts":           {"org_id": org, "id": part, "site_id": s, "created_by_project_id": p, "label": "Part", "kind": "part"},
			"work_items":              {"org_id": org, "id": w, "project_id": p, "site_id": s, "part_id": part, "system_id": "sys.fire", "action": "new", "inclusion": "included", "title": "Work", "origin": "user", "review_status": "accepted_for_planning"},
			"packages":                {"org_id": org, "id": pkg, "project_id": p, "kind": "services", "title": "Package", "lifecycle_status": "planned", "origin": "user", "review_status": "accepted_for_planning"},
			"package_stages":          {"org_id": org, "id": stage, "project_id": p, "package_id": pkg, "stage_id": "design", "label": "Design", "ordinal": 0, "origin": "user"},
			"package_scope_items":     {"org_id": org, "id": scope, "project_id": p, "package_id": pkg, "item_kind": "obligation", "user_text": "Scope", "inclusion": "included", "origin": "user", "review_status": "accepted_for_planning"},
			"project_delivery_items":  {"org_id": org, "id": d, "project_id": p, "kind": "milestone", "title": "Milestone", "status": "planned", "origin": "user", "review_status": "accepted_for_planning"},
			"reports":                 {"org_id": org, "id": report, "project_id": p, "package_id": pkg, "kind": "rfp", "title": "Report"},
			"report_versions":         {"org_id": org, "id": version, "project_id": p, "report_id": report, "number": 1, "status": "draft", "source_revisions": row{}, "reporting_date": "2026-10-06", "template_id": "rfp", "template_version": 1, "sections": []any{}},
			"profile_planning_values": {"org_id": org, "id": planning, "project_id": p, "site_id": s, "part_id": part, "key": "seed", "scope": "project", "value_state": "unknown", "origin": "user", "review_status": "accepted_for_planning", "meaning": "stated", "version": 1},
			"proposals":               {"org_id": org, "project_id": p, "site_id": s, "key": proposalKey, "record_kind": "ic", "record_id": recordID, "proposal_index": 0, "kind": "obligation", "label": "Proposal", "reason": row{}, "specificity": 0, "draft": true, "unaccepted_triggers": false, "inputs_fingerprint": strings.Repeat("a", 64), "knowledge_version": "seed", "state": "open"},
			"proposal_decisions":      {"org_id": org, "id": id(1400 + n), "project_id": p, "proposal_key": proposalKey, "record_id": recordID, "decision": "dismissed", "inputs_fingerprint": strings.Repeat("a", 64), "actor": u},
			"profile_user_values":     {"org_id": org, "id": id(1500 + n), "project_id": p, "site_id": s, "part_id": part, "key": "seed", "value": "yes", "user_id": u},
			"profile_rows":            {"org_id": org, "project_id": p, "site_id": s, "part_id": part, "key": "seed", "band": "blank"},
			"project_revisions":       {"org_id": org, "project_id": p},
			"proposal_triggers":       {"org_id": org, "project_id": p, "proposal_key": proposalKey, "work_item_id": w},
			"delivery_dependencies":   {"org_id": org, "project_id": p, "predecessor_id": d, "successor_id": id(950 + n)},
			"report_edits":            {"org_id": org, "report_version_id": version, "target_id": "seed", "text": "Edit", "base_content_sha256": strings.Repeat("a", 64), "user_id": u},
			"report_references":       {"org_id": org, "report_version_id": version, "citation_id": "U1", "label": "U", "anchor_id": "seed", "basis": row{}},
		}
		f["passages"] = row{"org_id": org, "id": id(2400 + n), "document_id": id(1900 + n), "ordinal": 0, "body": "Source"}
		f["passage_sources"] = row{"org_id": org, "passage_id": id(2400 + n), "document_id": id(1900 + n)}
		f["passage_calls"] = row{"org_id": org, "passage_id": id(2400 + n), "document_id": id(1900 + n), "stage": "evidence", "fingerprint": "fixture", "result": map[string]any{}}
		costVersion, costID, unusedCostID := id(2100+n), id(2200+n), id(2300+n)
		f["cost_plan_versions"] = row{"org_id": org, "project_id": p, "id": costVersion, "revision": 1, "status": "draft", "currency": "AUD", "tax_basis": "ex_tax"}
		f["cost_plans"] = row{"org_id": org, "project_id": p, "draft_version_id": costVersion}
		f["cost_items"] = row{"org_id": org, "project_id": p, "id": costID, "created_in_version_id": costVersion}
		f["cost_item_revisions"] = row{"org_id": org, "project_id": p, "plan_version_id": costVersion, "cost_item_id": costID, "label": "Line", "line_kind": "works", "work_item_id": w, "posting": true, "origin": "user", "meaning": "allowance"}
		f["cost_values"] = row{"org_id": org, "project_id": p, "plan_version_id": costVersion, "cost_item_id": costID, "metric": "estimate", "value_state": "unknown", "origin": "user", "meaning": "allowance"}
		f["scope_cost_links"] = row{"org_id": org, "project_id": p, "plan_version_id": costVersion, "cost_item_id": costID, "package_scope_item_id": scope}
		for _, table := range []string{"users", "sites", "projects", "files", "documents", "passages", "project_parts", "work_items", "packages", "package_stages", "package_scope_items", "project_delivery_items", "reports", "report_versions", "profile_planning_values", "proposals", "cost_plan_versions", "cost_items", "cost_item_revisions"} {
			if err := insert(tx, table, f[table]); err != nil {
				t.Fatal(table, err)
			}
		}
		unusedCost := maps.Clone(f["cost_items"])
		unusedCost["id"] = unusedCostID
		if err := insert(tx, "cost_items", unusedCost); err != nil {
			t.Fatal(err)
		}
		second := maps.Clone(f["project_delivery_items"])
		second["id"] = id(950 + n)
		if err := insert(tx, "project_delivery_items", second); err != nil {
			t.Fatal(err)
		}
		fixtures = append(fixtures, f)
	}
	// A new project needs an unused site for its positive-control insertion.
	run(`INSERT INTO sites(org_id,id,label) VALUES($1,$2,'Unused')`, id(1), id(9900))
	keys := []string{}
	for name, f := range after {
		if old, ok := before[name]; !ok || old.definition != f.definition {
			keys = append(keys, name)
		}
	}
	sort.Strings(keys)
	if len(keys) == 0 {
		t.Fatal("no next-wave foreign keys discovered")
	}
	// Pin the reviewed inventory too: discovery alone would silently stop
	// testing a constraint removed or weakened by a later migration.
	rawInventory, err := os.ReadFile("testdata/next_wave_foreign_keys.json")
	if err != nil {
		t.Fatal(err)
	}
	var expected map[string]string
	if err := json.Unmarshal(rawInventory, &expected); err != nil {
		t.Fatal(err)
	}
	actual := map[string]string{}
	for _, name := range keys {
		actual[name] = after[name].definition
	}
	if !maps.Equal(expected, actual) {
		t.Fatal("next-wave foreign-key contract changed; review the migration and update its coverage inventory explicitly")
	}
	for _, name := range keys {
		f := after[name]
		t.Run(f.name, func(t *testing.T) {
			base, ok := fixtures[0][f.table]
			if !ok {
				t.Fatalf("missing fixture for %s", f.table)
			}
			base = maps.Clone(base)
			if _, ok := base["id"]; ok {
				base["id"] = id(9999)
			}
			switch f.table {
			case "cost_plan_versions":
				base["revision"] = 2
				base["status"] = "baseline"
				base["frozen_at"] = "2026-10-07T00:00:00Z"
			case "cost_item_revisions":
				base["cost_item_id"] = id(2301)
				if strings.Contains(f.definition, "package_stage_id") {
					base["line_kind"] = "fee"
					base["work_item_id"] = nil
					base["package_id"] = fixtures[0]["packages"]["id"]
					base["package_stage_id"] = fixtures[0]["package_stages"]["id"]
				}
			case "projects":
				base["site_id"] = id(9900)
			case "project_parts":
				base["label"] = "New part"
			case "package_stages":
				base["label"] = "New stage"
			case "profile_planning_values", "profile_user_values", "profile_rows":
				base["key"] = "sweep"
			case "report_versions":
				base["number"] = 2
			case "proposals":
				base["record_id"] = "ic.sweep"
				base["key"] = "ic.sweep||||0"
			}
			if f.name == "project_parts_org_id_site_id_fkey" {
				base["created_by_project_id"] = nil
			}
			// Every invalid attempt must start from an insertion known to be valid.
			check := func(data row) error {
				save, err := tx.Begin(ctx)
				if err != nil {
					return err
				}
				defer save.Rollback(ctx)
				err = insert(save, f.table, data)
				if err == nil {
					_, err = save.Exec(ctx, "SET CONSTRAINTS ALL IMMEDIATE")
				}
				return err
			}
			if err := check(base); err != nil {
				t.Fatalf("invalid positive control: %v", err)
			}
			var lastErr error
			// Try each non-org component, then combinations, using existing records
			// from a different project or tenant. Never count an unrelated FK failure.
			targets := fixtures[2:]
			if len(f.columns) > 2 {
				targets = fixtures[1:]
			}
			for _, foreign := range targets {
				matched := false
				parent, ok := foreign[f.parent]
				if !ok {
					t.Fatalf("missing parent fixture %s", f.parent)
				}
				for mask := 1; mask < (1 << len(f.columns)); mask++ {
					data := maps.Clone(base)
					changed := false
					for i, col := range f.columns {
						if mask&(1<<i) == 0 || col == "org_id" {
							continue
						}
						value, ok := parent[f.parentColumns[i]]
						if !ok {
							t.Fatalf("missing parent column %s.%s", f.parent, f.parentColumns[i])
						}
						if strings.HasPrefix(col, "created_") && f.table == "proposal_decisions" {
							kind := map[string]string{"created_work_item_id": "work_item", "created_package_id": "package", "created_scope_item_id": "package_scope_item", "created_delivery_item_id": "delivery_item"}[col]
							data["created_record_type"] = kind
							data["created_record_id"] = value
							data["decision"] = "accepted"
						} else {
							data[col] = value
						}
						changed = true
					}
					if !changed {
						continue
					}
					if f.table == "proposals" {
						part := ""
						if v, ok := data["target_part_id"]; ok {
							part = v.(string)
						}
						data["key"] = "ic.sweep|||" + part + "|0"
					}
					if (f.table == "package_scope_items" || f.table == "project_delivery_items") && data["stage_id"] != nil && data["package_id"] == nil {
						data["package_id"] = fixtures[0]["packages"]["id"]
					}
					lastErr = check(data)
					var pgErr *pgconn.PgError
					if errors.As(lastErr, &pgErr) && pgErr.Code == "23503" && pgErr.ConstraintName == f.name {
						matched = true
						break
					}
				}
				if !matched {
					t.Fatalf("no rejection by %s itself for parent org %v project %v; last result: %v", f.name, parent["org_id"], parent["project_id"], lastErr)
				}
			}
		})
	}
	if !t.Failed() {
		t.Logf("verified %d new or changed composite foreign keys", len(keys))
	}
}

package db

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

// The program tables hold the two-tier, link and constraint rules that a
// bug in the store must never get past (plan section 5).
func TestProgramMigrationConstraints(t *testing.T) {
	ctx := context.Background()
	dsn := os.Getenv("SITEWISE_TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("dedicated test database required")
	}
	cfg, err := pgx.ParseConfig(dsn)
	if err != nil || cfg.Database != "sitewise_test" {
		t.Fatal("dedicated test database required")
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
	schema := pgx.Identifier{fmt.Sprintf("program_%d", time.Now().UnixNano())}.Sanitize()
	run("CREATE SCHEMA " + schema)
	run("SET LOCAL search_path TO " + schema + ",public")
	names, _ := fs.Glob(migrationFiles, "migrations/*.sql")
	sort.Strings(names)
	for _, name := range names {
		body, _ := migrationFiles.ReadFile(name)
		run(string(body))
	}
	id := func(n int) string { return fmt.Sprintf("33000000-0000-4000-8000-%012d", n) }
	// Org n (1 or 2) owns site n+2, project n+4, package n+6, package stage n+8.
	for i := 1; i <= 2; i++ {
		run(`INSERT INTO orgs(id,name) VALUES($1,'Program')`, id(i))
		run(`INSERT INTO sites(org_id,id,label) VALUES($1,$2,'Site')`, id(i), id(i+2))
		run(`INSERT INTO projects(org_id,id,site_id,name) VALUES($1,$2,$3,'Project')`, id(i), id(i+4), id(i+2))
		run(`INSERT INTO packages(org_id,id,project_id,kind,title,lifecycle_status,origin,review_status) VALUES($1,$2,$3,'services','Architect','planned','user','accepted_for_planning')`, id(i), id(i+6), id(i+4))
		run(`INSERT INTO package_stages(org_id,id,project_id,package_id,stage_id,label,ordinal,origin) VALUES($1,$2,$3,$4,'design','Design',0,'user')`, id(i), id(i+8), id(i+4), id(i+6))
	}
	bad := func(code, sql string, args ...any) {
		t.Helper()
		save, err := tx.Begin(ctx)
		if err != nil {
			t.Fatal(err)
		}
		defer save.Rollback(ctx)
		_, err = save.Exec(ctx, sql, args...)
		if err == nil {
			_, err = save.Exec(ctx, "SET CONSTRAINTS ALL IMMEDIATE")
		}
		var pgErr *pgconn.PgError
		if !errors.As(err, &pgErr) || pgErr.Code != code {
			t.Fatalf("want %s, got %v for %s %v", code, err, sql, args)
		}
	}

	program := `INSERT INTO programs(org_id,id,project_id,library_version,created_by_surface,working_week) VALUES($1,$2,$3,'lib',$4,$5)`
	run(program, id(1), id(20), id(5), "program", 31)
	run(program, id(2), id(21), id(6), "procurement", 31)
	bad("23505", program, id(1), id(22), id(5), "program", 31)  // one program per project
	bad("23503", program, id(2), id(22), id(5), "program", 31)  // another org's project
	bad("23514", program, id(1), id(22), id(5), "schedule", 31) // unknown surface
	bad("23514", program, id(1), id(22), id(5), "program", 0)   // empty working week
	bad("23514", program, id(1), id(22), id(5), "program", 128) // beyond Sunday
	bad("23514", `UPDATE programs SET stage_revision=0 WHERE id=$1`, id(20))
	bad("23514", `UPDATE programs SET start_origin='jev' WHERE id=$1`, id(20))

	act := `INSERT INTO program_activities(org_id,id,program_id,parent_id,ordinal,kind,label,catalogue_stage_id,duration_days,duration_low,duration_high,constraint_kind,constraint_date,review_status,origin,source_key,dismissed_reason) VALUES($1,$2,$3,$4,0,$5,$6,$7,$8,$9,$10,$11,$12,'proposed','library',$13,$14)`
	type row struct {
		org, id, program, parent, kind, label, catalogue string
		dur, low, high                                   any
		ck, cd, key, reason                              any
	}
	ins := func(r row) []any {
		var parent, catalogue any
		if r.parent != "" {
			parent = r.parent
		}
		if r.catalogue != "" {
			catalogue = r.catalogue
		}
		ck := r.ck
		if ck == nil {
			ck = "asap"
		}
		return []any{r.org, r.id, r.program, parent, r.kind, r.label, catalogue, r.dur, r.low, r.high, ck, r.cd, r.key, r.reason}
	}
	stage := row{org: id(1), id: id(30), program: id(20), kind: "stage", label: "Design", catalogue: "design", key: "skeleton:sk.generic:design"}
	run(act, ins(stage)...)
	child := row{org: id(1), id: id(31), program: id(20), parent: id(30), kind: "activity", label: "Concept design", dur: 10, low: 8, high: 15}
	run(act, ins(child)...)
	other := row{org: id(2), id: id(32), program: id(21), kind: "stage", label: "Design", catalogue: "design"}
	run(act, ins(other)...)
	with := func(r row, edit func(*row)) []any { r.id = id(39); edit(&r); return ins(r) }
	// Two tiers.
	bad("23514", act, with(stage, func(r *row) { r.parent = id(30) })...)
	bad("23514", act, with(child, func(r *row) { r.parent = "" })...)
	// Catalogue stage ids belong to stages only, and every stage has one.
	bad("23514", act, with(child, func(r *row) { r.catalogue = "other" })...)
	bad("23514", act, with(stage, func(r *row) { r.catalogue = ""; r.key = nil })...)
	// Labels, durations and ranges.
	bad("23514", act, with(child, func(r *row) { r.label = "   " })...)
	bad("23514", act, with(child, func(r *row) { r.label = strings.Repeat("x", 201) })...)
	bad("23514", act, with(child, func(r *row) { r.dur = 3651 })...)
	bad("23514", act, with(child, func(r *row) { r.dur = -1 })...)
	bad("23514", act, with(child, func(r *row) { r.high = nil })...)
	bad("23514", act, with(child, func(r *row) { r.low, r.high = 9, 8 })...)
	bad("23514", act, with(child, func(r *row) { r.kind = "milestone"; r.low, r.high = nil, nil; r.dur = 5 })...)
	run(act, with(child, func(r *row) { r.kind = "milestone"; r.low, r.high = nil, nil; r.dur = 0; r.id = id(33) })...)
	// Constraints come with a date exactly when they are not asap.
	bad("23514", act, with(child, func(r *row) { r.cd = "2027-03-01" })...)
	bad("23514", act, with(child, func(r *row) { r.ck = "snet" })...)
	bad("23514", act, with(child, func(r *row) { r.ck = "alap"; r.cd = "2027-03-01" })...)
	run(act, with(child, func(r *row) { r.ck = "mso"; r.cd = "2027-03-01"; r.id = id(34) })...)
	// A dismissal reason is for hold points only.
	bad("23514", act, with(child, func(r *row) { r.reason = "Not needed" })...)
	run(act, with(child, func(r *row) {
		r.kind = "hold_point"
		r.reason = "Not needed"
		r.dur = 0
		r.low, r.high = nil, nil
		r.id = id(35)
	})...)
	// The parent is a row of the same org.
	bad("23503", act, with(child, func(r *row) { r.parent = id(32) })...)
	bad("23503", act, with(child, func(r *row) { r.program = id(21) })...)
	// Source keys are unique among live rows only.
	bad("23505", act, with(stage, func(r *row) {})...)
	run(`UPDATE program_activities SET retired_at=now() WHERE id=$1`, id(30))
	run(act, with(stage, func(r *row) { r.id = id(36) })...)
	run(`UPDATE program_activities SET retired_at=NULL,source_key=NULL WHERE id=$1`, id(30))

	link := `INSERT INTO program_links(org_id,id,program_id,pred_id,succ_id,type,lag_days,origin) VALUES($1,$2,$3,$4,$5,$6,$7,$8)`
	run(link, id(1), id(40), id(20), id(31), id(33), "FS", 0, "chain")
	bad("23514", link, id(1), id(41), id(20), id(31), id(31), "FS", 0, "user")   // a row never links to itself
	bad("23514", link, id(1), id(41), id(20), id(33), id(34), "FS", 2, "chain")  // chains have no lag
	bad("23514", link, id(1), id(41), id(20), id(33), id(34), "SS", 0, "chain")  // chains are FS
	bad("23514", link, id(1), id(41), id(20), id(33), id(34), "XX", 0, "user")   // unknown type
	bad("23514", link, id(1), id(41), id(20), id(33), id(34), "FS", 366, "user") // lag beyond a year
	bad("23514", link, id(1), id(41), id(20), id(33), id(34), "FS", 0, "jev")    // unknown origin
	bad("23505", link, id(1), id(41), id(20), id(31), id(33), "SS", 0, "user")   // one live link per pair
	bad("23503", link, id(1), id(41), id(20), id(32), id(33), "FS", 0, "user")   // another org's row
	run(link, id(1), id(41), id(20), id(33), id(34), "FS", -365, "user")
	run(`UPDATE program_links SET retired_at=now() WHERE id=$1`, id(40))
	run(link, id(1), id(42), id(20), id(31), id(33), "SS", 2, "user")
	// A retired row still holds its links: rows are retired, never deleted.
	bad("23503", `DELETE FROM program_activities WHERE id=$1`, id(33))

	day := `INSERT INTO program_calendar_days(org_id,program_id,day,kind,label,origin) VALUES($1,$2,$3,$4,$5,$6)`
	run(day, id(1), id(20), "2027-04-02", "public_holiday", "Good Friday", "library")
	bad("23505", day, id(1), id(20), "2027-04-02", "rdo", "RDO", "user")
	bad("23514", day, id(1), id(20), "2027-04-03", "holiday", "Easter", "user")
	bad("23514", day, id(1), id(20), "2027-04-03", "rdo", " ", "user")
	bad("23514", day, id(1), id(20), "2027-04-03", "rdo", "RDO", "document")
	bad("23503", day, id(1), id(21), "2027-04-03", "rdo", "RDO", "user")

	run(`INSERT INTO program_ops(org_id,program_id,seq,actor,ops,inverse) VALUES($1,$2,1,$3,'[]','[]')`, id(1), id(20), id(99))
	bad("23505", `INSERT INTO program_ops(org_id,program_id,seq,actor,ops,inverse) VALUES($1,$2,1,$3,'[]','[]')`, id(1), id(20), id(99))

	change := `INSERT INTO program_changes(org_id,id,program_id,source,kind,source_key,proposal,summary,default_choice,status) VALUES($1,$2,$3,$4,$5,$6,'{}',$7,$8,$9)`
	run(change, id(1), id(50), id(20), "profile", "add", "hold:if.x", "Add hold point", "adopt", "pending")
	bad("23505", change, id(1), id(51), id(20), "profile", "add", "hold:if.x", "Add hold point", "adopt", "pending")
	run(change, id(1), id(51), id(20), "profile", "add", "hold:if.x", "Add hold point", "adopt", "superseded")
	bad("23514", change, id(1), id(52), id(20), "jev", "add", "k", "Add", "adopt", "pending")
	bad("23514", change, id(1), id(52), id(20), "profile", "move", "k", "Add", "adopt", "pending")
	bad("23514", change, id(1), id(52), id(20), "profile", "add", "k", " ", "adopt", "pending")
	bad("23514", change, id(1), id(52), id(20), "profile", "add", "k", "Add", "maybe", "pending")

	reading := `INSERT INTO program_readings(org_id,id,program_id,source,field,band,decided_by,status) VALUES($1,$2,$3,$4,'start',$5,$6,$7)`
	run(reading, id(1), id(60), id(20), "proposal_r7", "green", "rule", "proposed")
	bad("23514", reading, id(1), id(61), id(20), "email", "green", "rule", "proposed")
	bad("23514", reading, id(1), id(61), id(20), "proposal_r7", "red", "rule", "proposed")
	bad("23514", reading, id(1), id(61), id(20), "proposal_r7", "green", "llm", "proposed")
	bad("23514", reading, id(1), id(61), id(20), "proposal_r7", "green", "rule", "applied")

	// Package stages point at a program stage of their own org, and pin a revision.
	run(`UPDATE package_stages SET program_stage_id=$2,stage_revision=1 WHERE id=$1`, id(9), id(30))
	bad("23503", `UPDATE package_stages SET program_stage_id=$2 WHERE id=$1`, id(10), id(30))
	run(`UPDATE report_versions SET stage_revision=1 WHERE false`)

	// Deleting the project takes the program and its links with it, even
	// while a package stage points at a program stage.
	run("SAVEPOINT project_delete")
	run(`DELETE FROM projects WHERE org_id=$1 AND id=$2`, id(1), id(5))
	run("SET CONSTRAINTS ALL IMMEDIATE")
	var left int
	if err := tx.QueryRow(ctx, `SELECT (SELECT count(*) FROM programs WHERE org_id=$1)+(SELECT count(*) FROM program_activities WHERE org_id=$1)+(SELECT count(*) FROM program_links WHERE org_id=$1)`, id(1)).Scan(&left); err != nil || left != 0 {
		t.Fatalf("project delete left %d program rows: %v", left, err)
	}
	run("ROLLBACK TO SAVEPOINT project_delete")

	run(`DELETE FROM orgs WHERE id=$1`, id(1))
	run("SET CONSTRAINTS ALL IMMEDIATE")
	for _, table := range []string{"programs", "program_activities", "program_links", "program_calendar_days", "program_ops", "program_changes", "program_readings"} {
		var n int
		if err := tx.QueryRow(ctx, "SELECT count(*) FROM "+table+" WHERE org_id=$1", id(1)).Scan(&n); err != nil || n != 0 {
			t.Fatalf("%s survived delete: %d %v", table, n, err)
		}
		if err := tx.QueryRow(ctx, "SELECT count(*) FROM "+table+" WHERE org_id=$1", id(2)).Scan(&n); err != nil || (n == 0 && (table == "programs" || table == "program_activities")) {
			t.Fatalf("%s lost the other org's rows: %d %v", table, n, err)
		}
	}
}

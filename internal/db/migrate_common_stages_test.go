package db

import (
	"context"
	"fmt"
	"io/fs"
	"os"
	"sort"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
)

// Migration 047 (seven project stages, 7 November 2026): saved catalogue
// stage ids move to their new stage. Construction documentation becomes
// design finalisation after a novation or on a contractor-led program, and
// design development otherwise. Labels stay as the user saw them.
func TestCommonStagesMigrationMovesSavedStages(t *testing.T) {
	ctx := context.Background()
	cfg, e := pgx.ParseConfig(os.Getenv("SITEWISE_TEST_DATABASE_URL"))
	if e != nil || cfg.Database != "sitewise_test" {
		t.Fatal("dedicated test database required")
	}
	conn, e := pgx.ConnectConfig(ctx, cfg)
	if e != nil {
		t.Fatal(e)
	}
	defer conn.Close(ctx)
	tx, e := conn.Begin(ctx)
	if e != nil {
		t.Fatal(e)
	}
	defer tx.Rollback(ctx)
	must := func(sql string, args ...any) {
		t.Helper()
		if _, err := tx.Exec(ctx, sql, args...); err != nil {
			t.Fatal(sql, err)
		}
	}
	schema := pgx.Identifier{fmt.Sprintf("stages_%d", time.Now().UnixNano())}.Sanitize()
	must("CREATE SCHEMA " + schema)
	must("SET LOCAL search_path TO " + schema + ",public")
	names, e := fs.Glob(migrationFiles, "migrations/*.sql")
	if e != nil {
		t.Fatal(e)
	}
	sort.Strings(names)
	const migration = "migrations/047_common_stages.sql"
	for _, name := range names {
		if name >= migration {
			break
		}
		raw, _ := migrationFiles.ReadFile(name)
		must(string(raw))
	}
	const org = "47000000-0000-4000-8000-000000000001"
	must(`INSERT INTO orgs(id,name) VALUES($1,'Fixture')`, org)
	for _, p := range []string{"trad", "dc"} {
		must(`INSERT INTO sites(org_id,id,label) VALUES($1,md5($2||'site')::uuid,'Site')`, org, p)
		must(`INSERT INTO projects(org_id,id,name,site_id) VALUES($1,md5($2)::uuid,$2,md5($2||'site')::uuid)`, org, p)
	}
	must(`INSERT INTO packages(org_id,id,project_id,kind,title,novation,lifecycle_status,origin,review_status,meaning,provenance)
 VALUES($1,md5('arch')::uuid,md5('trad')::uuid,'services','Architect','option','planned','user','accepted_for_planning','stated','{}')`, org)
	stages := []struct{ id, old, phase, want string }{
		{"s1", "investigation", "pre", "brief_concept"},
		{"s2", "concept_design", "pre", "brief_concept"},
		{"s3", "approval_documentation", "pre", "planning"},
		{"s4", "construction_documentation", "pre", "design_development"},
		{"s5", "construction_documentation", "post", "design_finalisation"},
		{"s6", "procurement", "none", "tender_documentation"},
		{"s7", "construction_stage_inspections", "post", "delivery"},
		{"s8", "defects", "post", "delivery"},
		{"s9", "other", "none", "other"},
	}
	for i, s := range stages {
		must(`INSERT INTO package_stages(org_id,id,project_id,package_id,stage_id,label,ordinal,novation_phase,origin)
 VALUES($1,md5($2)::uuid,md5('trad')::uuid,md5('arch')::uuid,$3,$2,$4,$5,'user')`, org, s.id, s.old, i, s.phase)
	}
	for _, p := range []struct{ project, skeleton string }{{"trad", "sk.new-traditional-da"}, {"dc", "sk.new-dc-da"}} {
		must(`INSERT INTO programs(org_id,id,project_id,skeleton_id,library_version,created_by_surface) VALUES($1,md5($2||'program')::uuid,md5($2)::uuid,$3,'test','program')`, org, p.project, p.skeleton)
		for i, old := range []string{"concept_design", "construction_documentation", "construction", "other"} {
			must(`INSERT INTO program_activities(org_id,id,program_id,ordinal,kind,label,catalogue_stage_id,review_status,origin)
 VALUES($1,md5($2||$3)::uuid,md5($2||'program')::uuid,$4,'stage',$3,$3,'accepted','user')`, org, p.project, old, i)
		}
		must(`INSERT INTO program_readings(org_id,id,program_id,source,catalogue_stage_id,field,band,decided_by)
 VALUES($1,md5($2||'reading')::uuid,md5($2||'program')::uuid,'proposal_r7','construction_documentation','duration','green','rule')`, org, p.project)
	}
	must("SET CONSTRAINTS ALL IMMEDIATE")
	raw, _ := migrationFiles.ReadFile(migration)
	must(string(raw))

	for _, s := range stages {
		var got, label string
		if e := tx.QueryRow(ctx, `SELECT stage_id,label FROM package_stages WHERE org_id=$1 AND id=md5($2)::uuid`, org, s.id).Scan(&got, &label); e != nil || got != s.want || label != s.id {
			t.Fatalf("package stage %s (%s, %s): %q %q %v", s.id, s.old, s.phase, got, label, e)
		}
	}
	for project, want := range map[string]map[string]string{
		"trad": {"concept_design": "brief_concept", "construction_documentation": "design_development", "construction": "delivery", "other": "other"},
		"dc":   {"concept_design": "brief_concept", "construction_documentation": "design_finalisation", "construction": "delivery", "other": "other"},
	} {
		for old, w := range want {
			var got, label string
			if e := tx.QueryRow(ctx, `SELECT catalogue_stage_id,label FROM program_activities WHERE org_id=$1 AND id=md5($2||$3)::uuid`, org, project, old).Scan(&got, &label); e != nil || got != w || label != old {
				t.Fatalf("%s program stage %s: %q %q %v", project, old, got, label, e)
			}
		}
		var got string
		if e := tx.QueryRow(ctx, `SELECT catalogue_stage_id FROM program_readings WHERE org_id=$1 AND id=md5($2||'reading')::uuid`, org, project).Scan(&got); e != nil || got != want["construction_documentation"] {
			t.Fatalf("%s reading: %q %v", project, got, e)
		}
	}
}

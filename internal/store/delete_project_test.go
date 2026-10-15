package store_test

import (
	"bytes"
	"context"
	"errors"
	"testing"

	"sitewise/internal/store"
)

const (
	doomedProject = "33333333-3333-4333-8333-333333333301"
	doomedDoc     = "33333333-3333-4333-8333-333333333321"
	doomedShared  = "33333333-3333-4333-8333-333333333322"
	baseVersion = "33333333-3333-4333-8333-333333333332"
	costItem    = "33333333-3333-4333-8333-333333333341"
	specIssue     = "33333333-3333-4333-8333-333333333351"
	doomedReport  = "33333333-3333-4333-8333-333333333361"
	reportVersion = "33333333-3333-4333-8333-333333333362"
	exportFile    = "33333333-3333-4333-8333-333333333371"
)

func TestDeleteProjectRemovesEverythingThatBelongsToIt(t *testing.T) {
	ctx := context.Background()
	st := profileStore(t)
	if err := st.CreateProject(ctx, orgA, doomedProject, "To delete"); err != nil {
		t.Fatal(err)
	}
	if _, err := st.EnsureWholePart(ctx, orgA, doomedProject); err != nil {
		t.Fatal(err)
	}
	addDocument(t, st, orgA, doomedProject, doomedDoc, 0x61, "specification")
	if err := st.CreateFile(ctx, orgA, store.File{ID: doomedShared, ProjectID: doomedProject, SHA256: bytes32(0xab), ByteSize: 8, MediaType: "application/pdf"}); err != nil {
		t.Fatal(err)
	}
	if err := st.EnqueueStage(ctx, orgA, doomedDoc, store.JobKindLabel); err != nil {
		t.Fatal(err)
	}
	if _, err := st.AppendEvent(ctx, orgA, "profile", doomedDoc, `{"project_id":"`+doomedProject+`"}`); err != nil {
		t.Fatal(err)
	}
	if _, err := st.AppendEvent(ctx, orgA, "profile", docA, `{"project_id":"`+projectA+`"}`); err != nil {
		t.Fatal(err)
	}
	pool := rawPool(t)
	if _, err := pool.Exec(ctx, `
INSERT INTO profile_user_values (org_id, site_id, part_id, key, value, user_id, scope, value_state)
SELECT p.org_id, p.site_id, pp.id, 'det.bal', 'BAL-29', $3::uuid, 'site', 'set'
FROM projects p
JOIN project_parts pp ON pp.org_id = p.org_id AND pp.site_id = p.site_id AND pp.kind = 'whole'
WHERE p.org_id = $1::uuid AND p.id = $2::uuid`, orgA, doomedProject, userA); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `
INSERT INTO cost_plan_versions (org_id, project_id, id, revision, status, tax_basis)
VALUES ($1::uuid, $2::uuid, $3::uuid, 1, 'draft', 'ex_tax')`, orgA, doomedProject, baseVersion); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `
INSERT INTO cost_items (org_id, project_id, id, created_in_version_id)
VALUES ($1::uuid, $2::uuid, $3::uuid, $4::uuid)`, orgA, doomedProject, costItem, baseVersion); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `
INSERT INTO cost_item_revisions (org_id, project_id, plan_version_id, cost_item_id, label, line_kind, category, origin, meaning)
VALUES ($1::uuid, $2::uuid, $3::uuid, $4::uuid, 'Contingency', 'project_wide', 'other', 'user', 'allowance')`, orgA, doomedProject, baseVersion, costItem); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `
INSERT INTO cost_plans (org_id, project_id, draft_version_id, baseline_version_id)
VALUES ($1::uuid, $2::uuid, $3::uuid, $3::uuid)`, orgA, doomedProject, baseVersion); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `
UPDATE cost_plan_versions SET status = 'baseline', frozen_at = now()
WHERE org_id = $1::uuid AND id = $2::uuid`, orgA, baseVersion); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `
INSERT INTO specification_issues (org_id, project_id, id, number, title, purpose, snapshot, created_by)
VALUES ($1::uuid, $2::uuid, $3::uuid, 1, 'Schedule', 'preliminary', '{}', $4::uuid)`, orgA, doomedProject, specIssue, userA); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `
INSERT INTO specification_history (org_id, project_id, entity_id, entity_kind, actor, content)
VALUES ($1::uuid, $2::uuid, $3::uuid, 'type', $4::uuid, '{}')`, orgA, doomedProject, specIssue, userA); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `
INSERT INTO reports (org_id, id, project_id, kind, title)
VALUES ($1::uuid, $2::uuid, $3::uuid, 'pmp', 'Plan')`, orgA, doomedReport, doomedProject); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `
INSERT INTO report_versions (org_id, id, report_id, project_id, number, status, reporting_date, source_revisions, template_id, template_version, sections)
VALUES ($1::uuid, $2::uuid, $3::uuid, $4::uuid, 1, 'draft', '2026-10-14', '{}', 'pmp', 1, '[]')`, orgA, reportVersion, doomedReport, doomedProject); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `
INSERT INTO report_edits (org_id, report_version_id, target_id, text, base_content_sha256, user_id)
VALUES ($1::uuid, $2::uuid, 'intro', 'Kept wording', repeat('a', 64), $3::uuid)`, orgA, reportVersion, userA); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `
INSERT INTO files (org_id, id, project_id, sha256, byte_size, media_type)
VALUES ($1::uuid, $2::uuid, $3::uuid, decode(repeat('cd', 32), 'hex'), 12, 'application/pdf')`, orgA, exportFile, doomedProject); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `
UPDATE report_versions
SET status = 'issued', snapshot = '{}', snapshot_sha256 = repeat('ab', 32),
    export_file_sha256 = decode(repeat('cd', 32), 'hex'), issued_at = now(), issued_by = $3::uuid
WHERE org_id = $1::uuid AND id = $2::uuid`, orgA, reportVersion, userA); err != nil {
		t.Fatal(err)
	}

	var site string
	if err := pool.QueryRow(ctx, `SELECT site_id::text FROM projects WHERE org_id=$1 AND id=$2`, orgA, doomedProject).Scan(&site); err != nil {
		t.Fatal(err)
	}
	if _, err := st.DeleteProject(ctx, orgB, doomedProject); !errors.Is(err, store.ErrNotFound) {
		t.Fatalf("another org: %v", err)
	}
	if _, err := pool.Exec(ctx, `DELETE FROM cost_item_revisions WHERE org_id=$1 AND project_id=$2`, orgA, doomedProject); err == nil {
		t.Fatal("a frozen cost line was deleted on its own")
	}
	if _, err := pool.Exec(ctx, `DELETE FROM specification_issues WHERE org_id=$1 AND id=$2`, orgA, specIssue); err == nil {
		t.Fatal("an issued specification was deleted on its own")
	}
	if _, err := pool.Exec(ctx, `DELETE FROM report_versions WHERE org_id=$1 AND id=$2`, orgA, reportVersion); err == nil {
		t.Fatal("an issued report was deleted on its own")
	}

	res, err := st.DeleteProject(ctx, orgA, doomedProject)
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Deleted) != 1 || res.Deleted[0] != doomedProject {
		t.Fatalf("deleted %+v", res.Deleted)
	}
	if _, err := st.DeleteProject(ctx, orgA, doomedProject); !errors.Is(err, store.ErrNotFound) {
		t.Fatalf("second delete: %v", err)
	}

	gone := []struct {
		name string
		q    string
		args []any
	}{
		{"project", `SELECT count(*) FROM projects WHERE org_id=$1 AND id=$2`, []any{orgA, doomedProject}},
		{"site", `SELECT count(*) FROM sites WHERE org_id=$1 AND id=$2`, []any{orgA, site}},
		{"parts", `SELECT count(*) FROM project_parts WHERE org_id=$1 AND site_id=$2`, []any{orgA, site}},
		{"site values", `SELECT count(*) FROM profile_user_values WHERE org_id=$1 AND site_id=$2`, []any{orgA, site}},
		{"documents", `SELECT count(*) FROM documents WHERE org_id=$1 AND project_id=$2`, []any{orgA, doomedProject}},
		{"files", `SELECT count(*) FROM files WHERE org_id=$1 AND project_id=$2`, []any{orgA, doomedProject}},
		{"jobs", `SELECT count(*) FROM jobs WHERE org_id=$1 AND document_id=$2`, []any{orgA, doomedDoc}},
		{"costs", `SELECT count(*) FROM cost_plan_versions WHERE org_id=$1 AND project_id=$2`, []any{orgA, doomedProject}},
		{"issues", `SELECT count(*) FROM specification_issues WHERE org_id=$1 AND project_id=$2`, []any{orgA, doomedProject}},
		{"history", `SELECT count(*) FROM specification_history WHERE org_id=$1 AND project_id=$2`, []any{orgA, doomedProject}},
		{"reports", `SELECT count(*) FROM reports WHERE org_id=$1 AND project_id=$2`, []any{orgA, doomedProject}},
		{"events", `SELECT count(*) FROM events WHERE org_id=$1 AND payload::jsonb->>'project_id'=$2`, []any{orgA, doomedProject}},
	}
	for _, row := range gone {
		var n int
		if err := pool.QueryRow(ctx, row.q, row.args...).Scan(&n); err != nil || n != 0 {
			t.Fatalf("%s left %d (%v)", row.name, n, err)
		}
	}
	var kept int
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM projects WHERE org_id=$1 AND id=$2`, orgA, projectA).Scan(&kept); err != nil || kept != 1 {
		t.Fatal("the other project in the org was removed")
	}
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM events WHERE org_id=$1 AND payload::jsonb->>'project_id'=$2`, orgA, projectA).Scan(&kept); err != nil || kept != 1 {
		t.Fatalf("the other project's event was removed: %d %v", kept, err)
	}
	if documentExists(t, orgB, docB) == false {
		t.Fatal("another org's document was removed")
	}
	used, err := st.BlobReferenced(ctx, bytes32(0xab))
	if err != nil || !used {
		t.Fatalf("shared file referenced=%v %v", used, err)
	}
	if used, err := st.BlobReferenced(ctx, bytes32(0x61)); err != nil || used {
		t.Fatalf("deleted file referenced=%v %v", used, err)
	}
	export := bytes.Repeat([]byte{0xcd}, 32)
	foundExport := false
	for _, sum := range res.Unreferenced {
		if bytes.Equal(sum, export) {
			foundExport = true
		}
	}
	if !foundExport {
		t.Fatal("issued export hash was not returned for removal")
	}
	if used, err := st.BlobReferenced(ctx, export); err != nil || used {
		t.Fatalf("issued export still referenced=%v %v", used, err)
	}
}

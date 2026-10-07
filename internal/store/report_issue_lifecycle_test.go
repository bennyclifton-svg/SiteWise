package store_test

import (
	"context"
	"encoding/json"
	"sitewise/internal/files"
	"sitewise/internal/reports"
	"sitewise/internal/store"
	"strings"
	"testing"
)

// Keep the issue fixture deliberately short and user-authored. This tests the
// immutable lifecycle without pretending the draft knowledge is owner-reviewed.
func TestReportIssueHashAndResetSurviveNextDraftRefresh(t *testing.T) {
	ctx := context.Background()
	s, b, _ := workStore(t)
	if e := s.RebuildProfile(ctx, orgA, projectA, "", b.Compute); e != nil {
		t.Fatal(e)
	}
	r, e := s.CreateReport(ctx, orgA, projectA, userA, "pmp", "")
	if e != nil {
		t.Fatal(e)
	}
	d, e := s.RefreshReport(ctx, orgA, r.ID, userA, "test", true)
	if e != nil {
		t.Fatal(e)
	}
	d, e = s.EditReport(ctx, orgA, r.ID, userA, "project", "Protected wording", d.Version)
	if e != nil {
		t.Fatal(e)
	}
	var block reports.Block
	for _, section := range d.Sections {
		for _, b := range section.Blocks {
			if b.ID == "project" {
				block = b
			}
		}
	}
	block.Basis = json.RawMessage(`{"long_key":{"b":9007199254740993,"a":"exact"},"z":2}`)
	raw, _ := json.Marshal([]reports.Section{{ID: "brief", Title: "Brief", Blocks: []reports.Block{block}}})
	pool := rawPool(t)
	if _, e = pool.Exec(ctx, `UPDATE report_versions SET sections=$3::jsonb WHERE org_id=$1::uuid AND id=$2::uuid`, orgA, d.ID, raw); e != nil {
		t.Fatal(e)
	}
	blobs, e := files.Open(t.TempDir(), 20<<20)
	if e != nil {
		t.Fatal(e)
	}
	issued, e := s.IssueReport(ctx, orgA, r.ID, userA, "test", store.IssueOptions{Version: d.Version, ReportingDate: "2026-10-07", AcceptStale: true, StaleReason: "Fixture lifecycle"}, blobs)
	if e != nil {
		t.Fatal(e)
	}
	saved, e := s.ReadIssuedReport(ctx, orgA, r.ID, issued.ID)
	if e != nil {
		t.Fatal(e)
	}
	_, hash, e := reports.CanonicalSnapshot(saved.Snapshot)
	if e != nil || hash != issued.SnapshotSHA256 {
		t.Fatal("persisted hash differs", hash, issued.SnapshotSHA256, e)
	}
	if _, e = pool.Exec(ctx, `UPDATE report_versions SET reporting_date='2026-12-01' WHERE org_id=$1::uuid AND id=$2::uuid`, orgA, issued.ID); e == nil {
		t.Fatal("immutable update accepted")
	}
	if _, e = pool.Exec(ctx, `DELETE FROM report_versions WHERE org_id=$1::uuid AND id=$2::uuid`, orgA, issued.ID); e == nil {
		t.Fatal("immutable delete accepted")
	}
	d, e = s.RefreshReport(ctx, orgA, r.ID, userA, "test", true)
	if e != nil {
		t.Fatal(e)
	}
	d, e = s.ResetReportEdit(ctx, orgA, r.ID, userA, "project", d.Version)
	if e != nil {
		t.Fatal(e)
	}
	for n := 0; n < 2; n++ {
		d, e = s.RefreshReport(ctx, orgA, r.ID, userA, "test", true)
		if e != nil {
			t.Fatal(e)
		}
		for _, section := range d.Sections {
			for _, block := range section.Blocks {
				if block.ID == "project" && (block.Edited || block.Text == "Protected wording") {
					t.Fatal("reset edit resurrected", block)
				}
			}
		}
	}
	if e = s.DeleteOrg(ctx, orgA); e != nil {
		t.Fatal("tenant issue cleanup", e)
	}
}

func TestReportDocumentScheduleOmitsSupersededRevisions(t *testing.T) {
	ctx := context.Background()
	s, _, _ := workStore(t)
	pool := rawPool(t)
	newFile := "11111111-1111-4111-8111-aaaaaaaaab11"
	newDoc := "11111111-1111-4111-8111-aaaaaaaaab21"
	if _, e := pool.Exec(ctx, `INSERT INTO files(org_id,id,project_id,sha256,byte_size,media_type) VALUES($1::uuid,$2::uuid,$3::uuid,decode(repeat('9a',32),'hex'),10,'application/pdf')`, orgA, newFile, projectA); e != nil {
		t.Fatal(e)
	}
	if _, e := pool.Exec(ctx, `INSERT INTO documents(org_id,id,project_id,file_id,filename,status,document_number,revision) VALUES($1::uuid,$2::uuid,$3::uuid,$4::uuid,'current.pdf','filed','TEST-CURRENT','B')`, orgA, newDoc, projectA, newFile); e != nil {
		t.Fatal(e)
	}
	if _, e := pool.Exec(ctx, `INSERT INTO supersessions(org_id,document_id,prior_document_id) VALUES($1::uuid,$2::uuid,$3::uuid)`, orgA, newDoc, docA); e != nil {
		t.Fatal(e)
	}
	r, e := s.CreateReport(ctx, orgA, projectA, userA, "pmp", "")
	if e != nil {
		t.Fatal(e)
	}
	draft, e := s.RefreshReport(ctx, orgA, r.ID, userA, "test", true)
	if e != nil {
		t.Fatal(e)
	}
	found := false
	for _, section := range draft.Sections {
		for _, b := range section.Blocks {
			if b.ID == "document:"+docA {
				t.Fatal("superseded document issued as current")
			}
			if b.ID == "document:"+newDoc {
				found = true
				if !strings.Contains(b.Text, "document ID "+newDoc) {
					t.Fatal("document identity missing from standalone report body")
				}
			}
		}
	}
	if !found {
		t.Fatal("current revision missing")
	}
}

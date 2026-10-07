package store_test

import (
	"context"
	"errors"
	"strings"
	"testing"

	"sitewise/internal/procurement"
	"sitewise/internal/reports"
	"sitewise/internal/store"
)

func TestReportDraftRefreshEditsIsolationAndPending(t *testing.T) {
	ctx := context.Background()
	s, b, part := workStore(t)
	workType := "refurb"
	if _, err := s.SetUserValue(ctx, orgA, projectA, part, userA, "hdr.work_type", store.UserWrite{Value: &workType, State: "set", Scope: "project", Origin: "assumption", Meaning: "stated", Note: "Owner briefing pending"}); err != nil {
		t.Fatal(err)
	}
	p, err := s.CreatePackage(ctx, orgA, projectA, userA, procurement.Package{Kind: "services", Title: "Engineering", LifecycleStatus: "planned"})
	if err != nil {
		t.Fatal(err)
	}
	if err := s.RebuildProfile(ctx, orgA, projectA, "", b.Compute); err != nil {
		t.Fatal(err)
	}
	r, err := s.CreateReport(ctx, orgA, projectA, userA, "rfp", p.ID)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := s.CreateReport(ctx, orgB, projectA, userB, "rfp", p.ID); !errors.Is(err, store.ErrNotFound) {
		t.Fatalf("foreign create %v", err)
	}
	if _, err := s.ReadReport(ctx, orgB, r.ID, "test-build"); !errors.Is(err, store.ErrNotFound) {
		t.Fatalf("foreign read %v", err)
	}
	d, err := s.RefreshReport(ctx, orgA, r.ID, userA, "test-build", true)
	if err != nil {
		t.Fatal(err)
	}
	if d.ID == "" || len(d.Sections) != 7 || d.Status != "draft" || d.SourceRevisions.AppBuild != "test-build" {
		t.Fatalf("draft %+v", d)
	}
	view, err := s.ReadReport(ctx, orgA, r.ID, "test-build")
	if err != nil || view.Draft.ID != d.ID {
		t.Fatalf("read %+v %v", view, err)
	}
	if len(d.References) == 0 || len(view.Draft.References) != len(d.References) {
		t.Fatal("references not persisted with draft")
	}
	if len(d.MaterialAssumptions) == 0 {
		t.Fatal("assumptions omitted")
	}
	assumption := d.MaterialAssumptions[0]
	if !strings.Contains(assumption.Basis.Text, userA) || !strings.Contains(assumption.Basis.Text, "Owner briefing pending") || strings.Contains(assumption.Basis.Text, "Recorded: not recorded") {
		t.Fatal("missing assumption audit", assumption)
	}
	refsByID := map[string]bool{}
	for _, ref := range view.Draft.References {
		refsByID[ref.ID] = true
		if ref.Basis.Text == "" || len(ref.Basis.Source) == 0 {
			t.Fatal("incomplete printable reference", ref)
		}
	}
	for _, section := range view.Draft.Sections {
		for _, block := range section.Blocks {
			if len(block.CitationIDs) == 0 {
				t.Fatal("uncited block", block.ID)
			}
			for _, id := range block.CitationIDs {
				if !refsByID[id] {
					t.Fatal("dangling citation", id)
				}
			}
		}
	}
	target := "package:" + p.ID
	edited, err := s.EditReport(ctx, orgA, r.ID, userA, target, "Protected appointment wording", d.Version)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := s.EditReport(ctx, orgA, r.ID, userA, target, "Stale edit", d.Version); !errors.Is(err, store.ErrVersionConflict) {
		t.Fatalf("stale edit %v", err)
	}
	view, err = s.ReadReport(ctx, orgA, r.ID, "test-build")
	if err != nil {
		t.Fatal(err)
	}
	protected := false
	for _, ref := range view.Draft.References {
		if ref.AnchorID == target && ref.Basis.ProtectedEdit && ref.Basis.UserID == userA && !ref.Basis.UpdatedAt.IsZero() {
			protected = true
		}
	}
	if !protected {
		t.Fatal("protected user citation not persisted")
	}
	if _, err := s.EditReport(ctx, orgB, r.ID, userB, target, "Foreign edit", edited.Version); !errors.Is(err, store.ErrNotFound) {
		t.Fatalf("foreign edit %v", err)
	}
	title := "Changed engineering"
	if _, err := s.PatchPackage(ctx, orgA, projectA, p.ID, userA, procurement.PackagePatch{Version: 1, Title: &title}); err != nil {
		t.Fatal(err)
	}
	view, err = s.ReadReport(ctx, orgA, r.ID, "test-build")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(strings.Join(view.Stale, ","), "packages") {
		t.Fatal("package change not stale", view.Stale)
	}
	d, err = s.RefreshReport(ctx, orgA, r.ID, userA, "test-build", true)
	if err != nil {
		t.Fatal(err)
	}
	if d.ID != edited.ID {
		t.Fatal("refresh lost draft identity")
	}
	found := false
	for _, section := range d.Sections {
		for _, block := range section.Blocks {
			if block.ID == target {
				found = true
				if !block.Conflict || block.Text != "Protected appointment wording" {
					t.Fatal("edit lost", block)
				}
			}
		}
	}
	if !found {
		t.Fatal("protected target missing")
	}
	pool := rawPool(t)
	if _, err := pool.Exec(ctx, `INSERT INTO jobs(org_id,id,document_id,kind,status) VALUES($1::uuid,gen_random_uuid(),$2::uuid,'evidence','queued') ON CONFLICT(org_id,document_id,kind) DO UPDATE SET status='queued'`, orgA, docA); err != nil {
		t.Fatal(err)
	}
	before := d.Version
	if _, err := s.RefreshReport(ctx, orgA, r.ID, userA, "test-build", false); !errors.Is(err, reports.ErrReadingIncomplete) {
		t.Fatalf("pending reading %v", err)
	}
	view, err = s.ReadReport(ctx, orgA, r.ID, "test-build")
	if err != nil || view.Draft.Version != before {
		t.Fatalf("failed refresh mutated draft %+v %v", view, err)
	}
	if _, err := s.RefreshReport(ctx, orgA, r.ID, userA, "test-build", true); err != nil {
		t.Fatal("explicit completed state", err)
	}
	retired := true
	if _, err := s.PatchPackage(ctx, orgA, projectA, p.ID, userA, procurement.PackagePatch{Version: 2, Retired: &retired}); err != nil {
		t.Fatal(err)
	}
	if _, err := s.RefreshReport(ctx, orgA, r.ID, userA, "test-build", true); !errors.Is(err, store.ErrReportPackageRetired) {
		t.Fatalf("retired refresh %v", err)
	}
	view, err = s.ReadReport(ctx, orgA, r.ID, "test-build")
	if err != nil || !strings.Contains(strings.Join(view.Stale, ","), "package_retired") {
		t.Fatalf("historical draft unavailable %+v %v", view, err)
	}
}

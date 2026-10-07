package store_test

import (
	"context"
	"errors"
	"testing"

	"sitewise/internal/store"
)

func TestDrawingSheetsAtomicPublicationAndIsolation(t *testing.T) {
	ctx := context.Background()
	st := openHealthStore(t)
	for _, id := range []string{orgA, orgB} {
		if err := st.DeleteOrg(ctx, id); err != nil {
			t.Fatal(err)
		}
	}
	t.Cleanup(func() {
		for _, id := range []string{orgA, orgB} {
			_ = st.DeleteOrg(ctx, id)
		}
	})
	if err := seedTenants(ctx, st); err != nil {
		t.Fatal(err)
	}
	up, err := st.CommitIntake(ctx, orgA, store.CommitIntake{ProjectID: projectA, SHA256: bytes32(31), ByteSize: 100, MediaType: "application/pdf", Filename: "pack.pdf"})
	if err != nil {
		t.Fatal(err)
	}
	kind := store.DecisionWrite{Field: "kind", Value: "drawing", Band: "amber", DecidedBy: "jev", QuestionVersion: "test-version"}
	// Re-filing stored bytes may encounter an already queued/completed stage.
	if err := st.EnqueueStage(ctx, orgA, up.DocumentID, store.JobKindFullText); err != nil {
		t.Fatal(err)
	}
	_, err = st.CommitFiling(ctx, orgA, up.DocumentID, store.CommitFiling{PDFPages: 2, Decisions: []store.DecisionWrite{kind}})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := st.DrawingExpansion(ctx, orgB, up.DocumentID); !errors.Is(err, store.ErrNotFound) {
		t.Fatal("cross-org expansion", err)
	}
	pending, err := st.PendingDrawingExpansions(ctx)
	owned := 0
	for _, p := range pending {
		if p.OrgID == orgA || p.OrgID == orgB {
			owned++
			if p.DocumentID != up.DocumentID {
				t.Fatalf("unexpected tenant expansion: %+v", p)
			}
		}
	}
	// The global worker scan may include other fixtures, including the
	// workload benchmark. Assert isolation against this test's own tenants.
	if err != nil || owned != 1 {
		t.Fatalf("durable work: %+v %v", pending, err)
	}
	sheets := []store.SheetWrite{{SHA256: bytes32(32), ByteSize: 40, Decisions: []store.DecisionWrite{kind}}, {SHA256: bytes32(33), ByteSize: 50, Decisions: []store.DecisionWrite{kind}}}
	if err := st.PublishDrawingSheets(ctx, orgB, up.DocumentID, sheets); !errors.Is(err, store.ErrNotFound) {
		t.Fatal("cross-org publish", err)
	}
	bad := append([]store.SheetWrite{}, sheets...)
	bad[1].Decisions = nil
	if err := st.PublishDrawingSheets(ctx, orgA, up.DocumentID, bad); err == nil {
		t.Fatal("published mixed set")
	}
	doc, err := st.GetDocument(ctx, orgA, up.DocumentID)
	if err != nil || doc.Status != "filed" {
		t.Fatal("source changed on failure", doc, err)
	}
	// Fail after the first child has been inserted, exercising rollback rather
	// than merely checking pre-transaction validation.
	collision := append([]store.SheetWrite{}, sheets...)
	for i := range collision {
		collision[i].Decisions = []store.DecisionWrite{kind,
			{Field: "number", Value: "D-1", Band: "green", DecidedBy: "rule"},
			{Field: "revision", Value: "A", Band: "green", DecidedBy: "rule"}}
	}
	collision[1].PriorID = "invalid-uuid"
	if err := st.PublishDrawingSheets(ctx, orgA, up.DocumentID, collision); err == nil {
		t.Fatal("invalid second sheet unexpectedly published")
	}
	before, err := st.ProjectDocumentViews(ctx, orgA, projectA)
	if err != nil || len(before) != 2 {
		t.Fatalf("partial children survived rollback: %+v %v", before, err)
	}
	doc, err = st.GetDocument(ctx, orgA, up.DocumentID)
	if err != nil || doc.Status != "filed" {
		t.Fatal("source not restored on rollback", doc, err)
	}
	// Two deliveries can finish together after a restart or repeated trigger.
	// Pages sharing their printed number/revision must remain separate rows.
	for i := range sheets {
		sheets[i].Decisions = collision[i].Decisions
	}
	done := make(chan error, 2)
	for i := 0; i < 2; i++ {
		go func() { done <- st.PublishDrawingSheets(ctx, orgA, up.DocumentID, sheets) }()
	}
	for i := 0; i < 2; i++ {
		if err := <-done; err != nil {
			t.Fatal(err)
		}
	}
	views, err := st.ProjectDocumentViews(ctx, orgA, projectA)
	if err != nil {
		t.Fatal(err)
	}
	if len(views) != 4 {
		t.Fatalf("duplicate sheets: got %d (seed, source, two sheets)", len(views))
	}
	for _, view := range views {
		for _, field := range view.Fields {
			if field.Field == "kind" && field.DecidedBy == "jev" && field.QuestionVersion != "test-version" {
				t.Fatalf("persisted classifier version missing from listing: %+v", field)
			}
		}
	}
	exp, err := st.DrawingExpansion(ctx, orgA, up.DocumentID)
	if err != nil || exp.Status != "complete" {
		t.Fatalf("%+v %v", exp, err)
	}
	pending, err = st.PendingDrawingExpansions(ctx)
	owned = 0
	for _, p := range pending {
		if p.OrgID == orgA || p.OrgID == orgB {
			owned++
		}
	}
	if err != nil || owned != 0 {
		t.Fatal("completed work still pending", pending, err)
	}
	var child string
	priorPages := map[int]string{}
	for _, v := range views {
		if v.SourceID == up.DocumentID {
			child = v.ID
			priorPages[v.SheetPage] = v.ID
		}
	}
	assertSheetSupersessionScope(t, st, kind, priorPages)
	if err := st.CorrectDecision(ctx, orgA, child, "title", "Owner title"); err != nil {
		t.Fatal(err)
	}
	writes := []store.DecisionWrite{
		{Field: "number", Value: "S-7", Band: "green", DecidedBy: "rule", ExpectedVersion: 1},
		{Field: "revision", Value: "3", Band: "green", DecidedBy: "rule", ExpectedVersion: 1},
		{Field: "title", Value: "Parsed title", Band: "green", DecidedBy: "rule"},
	}
	if err := st.RefreshSheetMetadata(ctx, orgB, child, writes); err == nil {
		t.Fatal("cross-org refresh")
	}
	if err := st.RefreshSheetMetadata(ctx, orgA, up.DocumentID, writes); err == nil {
		t.Fatal("refreshed source container")
	}
	if err := st.RefreshSheetMetadata(ctx, orgA, child, writes); err != nil {
		t.Fatal(err)
	}
	v, err := st.DocumentView(ctx, orgA, child)
	if err != nil || v.Number != "S-7" || v.Revision != "3" {
		t.Fatalf("refreshed identity %+v %v", v, err)
	}
	for _, f := range v.Fields {
		if f.Field == "title" && (f.Value != "Owner title" || f.DecidedBy != "user") {
			t.Fatal("overwrote owner correction", f)
		}
	}
	writes[0].Value = "STALE"
	if err := st.RefreshSheetMetadata(ctx, orgA, child, writes); err != nil {
		t.Fatal(err)
	}
	v, err = st.DocumentView(ctx, orgA, child)
	if err != nil || v.Number != "S-7" {
		t.Fatal("stale parser overwrote number", v, err)
	}
}

func assertSheetSupersessionScope(t *testing.T, st *store.Store, kind store.DecisionWrite, priors map[int]string) {
	t.Helper()
	ctx := context.Background()
	up, err := st.CommitIntake(ctx, orgA, store.CommitIntake{ProjectID: projectA, SHA256: bytes32(34), ByteSize: 100, MediaType: "application/pdf", Filename: "pack-revision-b.pdf"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err = st.CommitFiling(ctx, orgA, up.DocumentID, store.CommitFiling{PDFPages: 2, Decisions: []store.DecisionWrite{kind}}); err != nil {
		t.Fatal(err)
	}
	var sheets []store.SheetWrite
	for i := 0; i < 2; i++ {
		sheets = append(sheets, store.SheetWrite{SHA256: bytes32(byte(35 + i)), ByteSize: 40, PriorID: priors[2], Decisions: []store.DecisionWrite{kind,
			{Field: "number", Value: "D-1", Band: "green", DecidedBy: "rule"},
			{Field: "revision", Value: "B", Band: "green", DecidedBy: "rule"}}})
	}
	if err := st.PublishDrawingSheets(ctx, orgA, up.DocumentID, sheets); err != nil {
		t.Fatal(err)
	}
	views, err := st.ProjectDocumentViews(ctx, orgA, projectA)
	if err != nil {
		t.Fatal(err)
	}
	links, err := st.OrgSupersessions(ctx, orgA)
	if err != nil {
		t.Fatal(err)
	}
	for _, v := range views {
		if v.SourceID != up.DocumentID {
			continue
		}
		linked := false
		for _, link := range links {
			linked = linked || link[0] == v.ID
		}
		if linked != (v.SheetPage == 2) {
			t.Fatalf("page %d supersession crosses physical sheet identity: %v", v.SheetPage, links)
		}
	}
}

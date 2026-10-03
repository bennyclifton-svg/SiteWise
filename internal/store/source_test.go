package store_test

import (
	"context"
	"sitewise/internal/store"
	"testing"
	"time"
)

func TestSourceRecordsIsolationPaginationAndCoverage(t *testing.T) {
	st := profileStore(t)
	ctx := context.Background()
	src := store.DocumentSource{Pages: 83, EmptyPages: []int{40}, Source: []store.SourcePage{{Page: 83, Text: "Handover."}}}
	for i := 0; i < 75; i++ {
		src.Units = append(src.Units, store.SourceUnit{Page: 83, Body: "Handover.", Section: "Handover", End: 9})
	}
	if err := st.ReplaceSource(ctx, orgA, docA, src); err != nil {
		t.Fatal(err)
	}
	view, err := st.SourceRecords(ctx, orgA, projectA, "", "", 0)
	if err != nil || len(view.Records) != 50 || !view.More {
		t.Fatalf("page1 %+v %v", view, err)
	}
	next, err := st.SourceRecords(ctx, orgA, projectA, "", "", 50)
	if err != nil || len(next.Records) != 25 || next.More {
		t.Fatalf("page2 %+v %v", next, err)
	}
	if _, err := st.SourceRecords(ctx, orgB, projectA, "", "", 0); err != store.ErrNotFound {
		t.Fatalf("foreign read %v", err)
	}
	c, err := st.SourceCoverage(ctx, orgA, projectA)
	if err != nil || len(c) != 1 || c[0].Units != 75 || c[0].Pages != 83 || !c[0].Current {
		t.Fatalf("coverage %+v %v", c, err)
	}
	if err := st.SetSourceReading(ctx, orgA, view.Records[0].ID, store.SourceReading{Category: "requirement", Outcome: "needs_mapping"}); err != nil {
		t.Fatal(err)
	}
	if err := st.SetPassageSystems(ctx, orgA, view.Records[0].ID, []string{"hydraulic.hot-water", "hydraulic.cold-water"}); err != nil {
		t.Fatal(err)
	}
	filtered, err := st.SourceRecords(ctx, orgA, projectA, "hydraulic.hot-water", "", 0)
	if err != nil || len(filtered.Records) != 1 {
		t.Fatalf("filter %+v %v", filtered, err)
	}
}

func TestOldProfileUpgradeWaitsForExtractionAndLabels(t *testing.T) {
	st := profileStore(t)
	ctx := context.Background()
	pool := rawPool(t)
	if err := st.ReplacePassages(ctx, orgA, docA, []string{"Old truncated text"}); err != nil {
		t.Fatal(err)
	}
	for _, k := range []string{store.JobKindFullText, store.JobKindLabel, store.JobKindEvidence} {
		if err := st.EnqueueStage(ctx, orgA, docA, k); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := pool.Exec(ctx, `UPDATE jobs SET status='done' WHERE org_id=$1 AND document_id=$2`, orgA, docA); err != nil {
		t.Fatal(err)
	}
	if _, err := st.RequestProfileRead(ctx, orgA, projectA, nil); err != nil {
		t.Fatal(err)
	}
	if _, err := st.ClaimJob(ctx, orgA, time.Minute, []string{store.JobKindEvidence}); err != store.ErrIdle {
		t.Fatalf("evidence raced upgrade: %v", err)
	}
	if _, err := st.ClaimJob(ctx, orgA, time.Minute, []string{store.JobKindLabel}); err != store.ErrIdle {
		t.Fatalf("labels raced extraction: %v", err)
	}
	if _, err := st.ClaimJob(ctx, orgA, time.Minute, []string{store.JobKindFullText}); err != nil {
		t.Fatalf("old extraction not refreshed: %v", err)
	}
}

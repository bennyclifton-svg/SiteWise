package store_test

import (
	"context"
	"testing"
	"time"

	"sitewise/internal/profile"
)

func TestEnsureWholePartConcurrentCreation(t *testing.T) {
	s := profileStore(t)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	pool := rawPool(t)
	tx, err := pool.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer tx.Rollback(ctx)
	var pid int
	if err := tx.QueryRow(ctx, `SELECT pg_backend_pid()`).Scan(&pid); err != nil {
		t.Fatal(err)
	}
	var id string
	if err := tx.QueryRow(ctx, `INSERT INTO project_parts(org_id,id,site_id,created_by_project_id,label,kind)
SELECT org_id,gen_random_uuid(),site_id,id,'Concurrent whole','whole' FROM projects WHERE org_id=$1::uuid AND id=$2::uuid RETURNING id::text`, orgA, projectA).Scan(&id); err != nil {
		t.Fatal(err)
	}
	type result struct {
		part profile.Part
		err  error
	}
	done := make(chan result, 1)
	go func() { p, e := s.EnsureWholePart(ctx, orgA, projectA); done <- result{p, e} }()
	// Observe the uniqueness wait before committing the competing insert.
	// A sleep alone would not prove the request took its earlier snapshot.
	ticker := time.NewTicker(5 * time.Millisecond)
	defer ticker.Stop()
	for {
		var waiting bool
		if err := pool.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM pg_stat_activity WHERE $1::int=ANY(pg_blocking_pids(pid)))`, pid).Scan(&waiting); err != nil {
			t.Fatal(err)
		}
		if waiting {
			break
		}
		select {
		case got := <-done:
			t.Fatalf("request did not wait for competing insert: %+v", got)
		case <-ctx.Done():
			t.Fatal(ctx.Err())
		case <-ticker.C:
		}
	}
	if err := tx.Commit(ctx); err != nil {
		t.Fatal(err)
	}
	select {
	case got := <-done:
		if got.err != nil || got.part.ID != id || got.part.Label != "Concurrent whole" {
			t.Fatalf("concurrent creation: %+v", got)
		}
	case <-ctx.Done():
		t.Fatal(ctx.Err())
	}
	var n int
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM project_parts WHERE org_id=$1::uuid AND site_id=(SELECT site_id FROM projects WHERE org_id=$1::uuid AND id=$2::uuid) AND kind='whole'`, orgA, projectA).Scan(&n); err != nil {
		t.Fatal(err)
	}
	if n != 1 {
		t.Fatalf("whole parts: %d", n)
	}
}

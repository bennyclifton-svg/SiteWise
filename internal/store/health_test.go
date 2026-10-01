package store_test

import (
	"context"
	"errors"
	"os"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"sitewise/internal/store"
)

func TestBacklogRoleAndPing(t *testing.T) {
	ctx := context.Background()
	st := openHealthStore(t)
	for _, orgID := range []string{orgA, orgB} {
		if err := st.DeleteOrg(ctx, orgID); err != nil {
			t.Fatal(err)
		}
	}
	t.Cleanup(func() {
		for _, orgID := range []string{orgA, orgB} {
			_ = st.DeleteOrg(context.Background(), orgID)
		}
	})
	if err := seedTenants(ctx, st); err != nil {
		t.Fatal(err)
	}
	if err := st.Ping(ctx); err != nil {
		t.Fatal(err)
	}

	if role, err := st.Role(ctx, orgA, userA); err != nil || role != "owner" {
		t.Fatalf("role %q %v", role, err)
	}
	if _, err := st.Role(ctx, orgA, userB); !errors.Is(err, store.ErrNotFound) {
		t.Fatalf("cross-org role %v", err)
	}

	if err := st.EnqueueStage(ctx, orgA, docA, store.JobKindFullText); err != nil {
		t.Fatal(err)
	}
	if err := st.EnqueueStage(ctx, orgB, docB, store.JobKindFullText); err != nil {
		t.Fatal(err)
	}
	// Age by the database clock: org A's job is ten minutes old.
	pool := rawPool(t)
	if _, err := pool.Exec(ctx, `UPDATE jobs SET created_at = now() - interval '10 minutes' WHERE org_id = $1`, orgA); err != nil {
		t.Fatal(err)
	}

	a, err := st.OrgBacklog(ctx, orgA)
	if err != nil {
		t.Fatal(err)
	}
	got := backlogOf(a, store.JobKindFullText)
	if got.Queued != 1 || got.Oldest < 10*time.Minute || got.Oldest > 11*time.Minute {
		t.Fatalf("org A backlog %+v", a)
	}
	b, err := st.OrgBacklog(ctx, orgB)
	if err != nil {
		t.Fatal(err)
	}
	if got := backlogOf(b, store.JobKindFullText); got.Queued != 1 || got.Oldest > time.Minute {
		t.Fatalf("org B backlog %+v", b)
	}

	// The process view spans orgs; other tests may leave work of their own.
	all, err := st.Backlog(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if got := backlogOf(all, store.JobKindFullText); got.Queued < 2 || got.Oldest < 10*time.Minute {
		t.Fatalf("process backlog %+v", all)
	}

	// Finished work is not backlog.
	if _, err := pool.Exec(ctx, `UPDATE jobs SET status = 'done' WHERE org_id = $1`, orgA); err != nil {
		t.Fatal(err)
	}
	a, err = st.OrgBacklog(ctx, orgA)
	if err != nil {
		t.Fatal(err)
	}
	if got := backlogOf(a, store.JobKindFullText); got.Queued != 0 {
		t.Fatalf("done job counted %+v", a)
	}

	st.Close()
	if err := st.Ping(ctx); err == nil {
		t.Fatal("ping on a closed pool succeeded")
	}
}

func backlogOf(list []store.Backlog, kind string) store.Backlog {
	for _, b := range list {
		if b.Kind == kind {
			return b
		}
	}
	return store.Backlog{Kind: kind}
}

func openHealthStore(t *testing.T) *store.Store {
	t.Helper()
	dsn := testDSN(t)
	ctx := context.Background()
	st, err := store.Open(ctx, dsn)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(st.Close)
	if err := st.Migrate(ctx); err != nil {
		t.Fatal(err)
	}
	return st
}

func rawPool(t *testing.T) *pgxpool.Pool {
	t.Helper()
	pool, err := pgxpool.New(context.Background(), testDSN(t))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(pool.Close)
	return pool
}

func testDSN(t *testing.T) string {
	t.Helper()
	dsn := os.Getenv("SITEWISE_TEST_DATABASE_URL")
	if dsn == "" {
		t.Fatal("SITEWISE_TEST_DATABASE_URL is required")
	}
	name, err := databaseName(dsn)
	if err != nil {
		t.Fatal(err)
	}
	if name != "sitewise_test" {
		t.Fatalf("refusing to use database %q", name)
	}
	return dsn
}

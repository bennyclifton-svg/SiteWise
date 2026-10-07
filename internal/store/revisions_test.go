package store_test

import (
	"context"
	"errors"
	"sync"
	"testing"

	"sitewise/internal/profile"
	"sitewise/internal/store"
)

func TestRevisionBumpPreservesInitializationAndDomainCounters(t *testing.T) {
	ctx := context.Background()
	profileStore(t)
	pool := rawPool(t)
	if _, err := pool.Exec(ctx, `DELETE FROM project_revisions WHERE org_id=$1::uuid AND project_id=$2::uuid`, orgA, projectA); err != nil {
		t.Fatal(err)
	}
	domains := []string{"profile_inputs", "works", "packages", "delivery", "costs", "reports"}
	for round := int64(1); round <= 2; round++ {
		for index, domain := range domains {
			tx, err := pool.Begin(ctx)
			if err != nil {
				t.Fatal(err)
			}
			err = store.BumpRevision(ctx, tx, orgA, projectA, domain)
			if err != nil {
				_ = tx.Rollback(ctx)
				t.Fatal(err)
			}
			if err := tx.Commit(ctx); err != nil {
				t.Fatal(err)
			}
			var counters [6]int64
			var version int64
			if err := pool.QueryRow(ctx, `SELECT profile_inputs,works,packages,delivery,costs,reports,version FROM project_revisions WHERE org_id=$1::uuid AND project_id=$2::uuid`, orgA, projectA).Scan(&counters[0], &counters[1], &counters[2], &counters[3], &counters[4], &counters[5], &version); err != nil {
				t.Fatal(err)
			}
			for n, count := range counters {
				want := round - 1
				if n <= index {
					want = round
				}
				if count != want {
					t.Fatalf("domain %s count %d want %d", domains[n], count, want)
				}
			}
			if version != 2+(round-1)*6+int64(index) {
				t.Fatalf("version %d", version)
			}
		}
	}
	for _, org := range []string{orgB, orgA} {
		tx, err := pool.Begin(ctx)
		if err != nil {
			t.Fatal(err)
		}
		domain := "works"
		if org == orgA {
			domain = "unexpected"
		}
		err = store.BumpRevision(ctx, tx, org, projectA, domain)
		_ = tx.Rollback(ctx)
		if err == nil || (org == orgB && !errors.Is(err, store.ErrNotFound)) {
			t.Fatalf("invalid bump: %v", err)
		}
	}
}

func TestRevisionAtomicityConcurrencyAndFreshness(t *testing.T) {
	ctx := context.Background()
	raw := profileStore(t)
	part, err := raw.EnsureWholePart(ctx, orgA, projectA)
	if err != nil {
		t.Fatal(err)
	}
	build := store.ProfileBuild{KnowledgeVersion: "knowledge-1", QuestionVersion: "questions-1", ThresholdsVersion: "thresholds-1",
		Compute: func(s store.ProfileSnapshot) []profile.Row {
			rows := []profile.Row{}
			for _, u := range s.User {
				rows = append(rows, profile.Row{PartID: u.PartID, Key: u.Key, Value: *u.Value, Band: "user"})
			}
			return rows
		}}
	st := raw.WithProfile(build)
	write := store.UserWrite{Value: strPtr("house"), Scope: "site"}
	if _, err := st.SetUserValue(ctx, orgA, projectA, part.ID, userA, "hdr.subclass", write); err != nil {
		t.Fatal(err)
	}
	before, err := st.ReadProfile(ctx, orgA, projectA, nil)
	if err != nil || before.Revision != 1 || len(before.InputFingerprint) != 64 || len(before.StaleFor(build)) != 0 {
		t.Fatalf("initial build: %+v %v", before, err)
	}
	// A failure after deleting old rows must roll back the value, counter,
	// build and event, retaining the last complete state.
	broken := build
	broken.Compute = func(s store.ProfileSnapshot) []profile.Row {
		return []profile.Row{{PartID: part.ID, Key: "bad", Band: "invalid"}}
	}
	write.Value = strPtr("warehouse")
	if _, err := raw.WithProfile(broken).SetUserValue(ctx, orgA, projectA, part.ID, userA, "hdr.subclass", write); err == nil {
		t.Fatal("expected rebuild failure")
	}
	after, err := st.ReadProfile(ctx, orgA, projectA, nil)
	if err != nil || after.Revision != before.Revision || after.CurrentInputs != before.CurrentInputs || after.InputFingerprint != before.InputFingerprint || after.Rows[0].Value != "house" {
		t.Fatalf("partial commit: %+v %v", after, err)
	}
	const writers = 12
	var wg sync.WaitGroup
	failures := make(chan error, writers)
	for i := 0; i < writers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, err := st.SetUserValue(ctx, orgA, projectA, part.ID, userA, "hdr.subclass", write)
			failures <- err
		}()
	}
	wg.Wait()
	close(failures)
	for err := range failures {
		if err != nil {
			t.Fatal(err)
		}
	}
	after, err = st.ReadProfile(ctx, orgA, projectA, nil)
	if err != nil || after.Revision != before.Revision+writers || after.CurrentInputs.ProfileInputs != before.CurrentInputs.ProfileInputs+writers {
		t.Fatalf("lost counter: %+v %v", after, err)
	}
	if _, err := st.ReadProfile(ctx, orgB, projectA, nil); !errors.Is(err, store.ErrNotFound) {
		t.Fatalf("foreign revision: %v", err)
	}
	// Unconfigured writers invalidate without claiming a rebuild completed.
	if _, err := raw.SetProfileReading(ctx, orgA, projectA, []string{docA}, "skip", nil); err != nil {
		t.Fatal(err)
	}
	stale, err := st.ReadProfile(ctx, orgA, projectA, nil)
	if err != nil || len(stale.StaleFor(build)) != 1 || stale.StaleFor(build)[0] != "profile_inputs" {
		t.Fatalf("missing staleness: %v %v", stale.StaleFor(build), err)
	}
	changed := build
	changed.KnowledgeVersion = "knowledge-2"
	if len(after.StaleFor(changed)) != 1 {
		t.Fatal("knowledge update not stale")
	}
	if err := st.RebuildProfile(ctx, orgA, projectA, build.ThresholdsVersion, build.Compute); err != nil {
		t.Fatal(err)
	}
	fresh, _ := st.ReadProfile(ctx, orgA, projectA, nil)
	if len(fresh.StaleFor(build)) != 0 {
		t.Fatal("rebuild remained stale")
	}
	if err := st.RebuildProfile(ctx, orgA, projectA, build.ThresholdsVersion, build.Compute); err != nil {
		t.Fatal(err)
	}
	repeated, _ := st.ReadProfile(ctx, orgA, projectA, nil)
	if repeated.Revision != fresh.Revision+1 || repeated.InputFingerprint != fresh.InputFingerprint || repeated.CurrentInputs != fresh.CurrentInputs {
		t.Fatal("repeat rebuild changed inputs")
	}
	reopened, err := store.Open(ctx, testDSN(t))
	if err != nil {
		t.Fatal(err)
	}
	defer reopened.Close()
	recovered, err := reopened.ReadProfile(ctx, orgA, projectA, nil)
	if err != nil || recovered.Revision != repeated.Revision || recovered.InputFingerprint != repeated.InputFingerprint || recovered.CurrentInputs != repeated.CurrentInputs {
		t.Fatal("reopen lost completed build")
	}

}

func TestConcurrentRefreshRequestsCoalesce(t *testing.T) {
	st := profileStore(t)
	ctx := context.Background()
	if err := st.EnqueueStage(ctx, orgA, docA, store.JobKindFullText); err != nil {
		t.Fatal(err)
	}
	var wg sync.WaitGroup
	counts := make(chan int64, 2)
	errs := make(chan error, 2)
	for i := 0; i < 2; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			n, err := st.RequestProfileRead(ctx, orgA, projectA, nil)
			counts <- n
			errs <- err
		}()
	}
	wg.Wait()
	close(counts)
	close(errs)
	for err := range errs {
		if err != nil {
			t.Fatal(err)
		}
	}
	var total int64
	for n := range counts {
		total += n
	}
	if total != 1 {
		t.Fatalf("queued %d job sets", total)
	}
}

func TestDomainRevisionsAreScopedAndTransactional(t *testing.T) {
	st := profileStore(t)
	ctx := context.Background()
	pool := rawPool(t)
	before, err := st.ReadProfile(ctx, orgA, projectA, nil)
	if err != nil {
		t.Fatal(err)
	}
	tx, err := pool.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer tx.Rollback(ctx)
	for _, domain := range []string{"profile_inputs", "works", "packages", "delivery", "costs", "reports"} {
		if err := store.BumpRevision(ctx, tx, orgA, projectA, domain); err != nil {
			t.Fatal(err)
		}
	}
	if err := tx.Commit(ctx); err != nil {
		t.Fatal(err)
	}
	after, err := st.ReadProfile(ctx, orgA, projectA, nil)
	want := before.CurrentInputs
	want.ProfileInputs++
	want.Works++
	want.Packages++
	want.Delivery++
	want.Costs++
	want.Reports++
	if err != nil || after.CurrentInputs != want {
		t.Fatalf("domain counters: %+v %v", after.CurrentInputs, err)
	}
	for _, domain := range []string{"profile_inputs", "unexpected"} {
		tx, err := pool.Begin(ctx)
		if err != nil {
			t.Fatal(err)
		}
		err = store.BumpRevision(ctx, tx, orgB, projectA, domain)
		tx.Rollback(ctx)
		if err == nil {
			t.Fatal("foreign/invalid domain accepted")
		}
	}
	tx, err = pool.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if err := store.BumpRevision(ctx, tx, orgA, projectA, "works"); err != nil {
		t.Fatal(err)
	}
	tx.Rollback(ctx)
	rolledBack, err := st.ReadProfile(ctx, orgA, projectA, nil)
	if err != nil || rolledBack.CurrentInputs != want {
		t.Fatal("rolled back revision persisted")
	}
	if _, err := pool.Exec(ctx, `INSERT INTO project_revisions(org_id,project_id) VALUES($1::uuid,$2::uuid)`, orgB, projectA); err == nil {
		t.Fatal("cross-org revision FK accepted")
	}
}

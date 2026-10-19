package store_test

import (
	"context"
	"fmt"
	"math/rand"
	"sort"
	"testing"
	"time"

	"sitewise/internal/store"
)

// largeProgram is 50 stages of 19 activities (1,000 rows) with 2,000
// links: the stage and activity chains plus forward links between
// activities, so it stays a loop-free graph.
func largeProgram() store.ProgramCreate {
	in := store.ProgramCreate{Surface: "program", LibraryVersion: "timing", StartDate: "2027-03-01", StartOrigin: "user"}
	var acts []string
	parent := map[string]string{}
	for s := 0; s < 50; s++ {
		sk := fmt.Sprintf("s%d", s)
		in.Rows = append(in.Rows, store.ProgramRowInput{Key: sk, Kind: "stage", Label: fmt.Sprintf("Stage %d", s), CatalogueStageID: "other", ReviewStatus: "proposed", Origin: "library"})
		if s > 0 {
			in.Links = append(in.Links, store.ProgramLinkInput{PredKey: fmt.Sprintf("s%d", s-1), SuccKey: sk, Origin: "chain"})
		}
		for a := 0; a < 19; a++ {
			ak := fmt.Sprintf("s%da%d", s, a)
			in.Rows = append(in.Rows, store.ProgramRowInput{Key: ak, ParentKey: sk, Kind: "activity", Label: fmt.Sprintf("Activity %d.%d", s, a), DurationDays: ip(1 + (s*19+a)%9), DurationOrigin: "library", ReviewStatus: "proposed", Origin: "library"})
			if a > 0 {
				in.Links = append(in.Links, store.ProgramLinkInput{PredKey: fmt.Sprintf("s%da%d", s, a-1), SuccKey: ak, Origin: "chain"})
			}
			acts = append(acts, ak)
			parent[ak] = sk
		}
	}
	r := rand.New(rand.NewSource(1))
	seen := map[[2]string]bool{}
	for len(in.Links) < 2000 {
		i := r.Intn(len(acts) - 45)
		j := i + 2 + r.Intn(40)
		p := [2]string{acts[i], acts[j]}
		if seen[p] {
			continue
		}
		seen[p] = true
		in.Links = append(in.Links, store.ProgramLinkInput{PredKey: p[0], SuccKey: p[1], Type: []string{"FS", "SS", "FF"}[r.Intn(3)], LagDays: r.Intn(5) - 1, Origin: "user"})
	}
	return in
}

// TestProgramWriteTiming measures one move on a 1,000-activity program
// through the store: lock and load, validate, apply, recompute, write the
// diff and the undo log, commit (the program_write budget is p50 100 ms,
// p90 250 ms including HTTP, plan 11.1).
func TestProgramWriteTiming(t *testing.T) {
	ctx := context.Background()
	s := profileStore(t)
	if _, err := s.CreateProgram(ctx, orgA, projectA, largeProgram()); err != nil {
		t.Fatal(err)
	}
	st, err := s.LoadProgram(ctx, orgA, projectA)
	if err != nil {
		t.Fatal(err)
	}
	if len(st.Activities) != 1000 || len(st.Links) != 2000 || st.Computed.Cycle != nil {
		t.Fatalf("%d rows %d links cycle %v", len(st.Activities), len(st.Links), st.Computed.Cycle)
	}
	target := rowID(t, st, "Activity 25.10")
	var writes, reads []time.Duration
	for k := 0; k < 41; k++ {
		days := 1
		if k%2 == 1 {
			days = -1
		}
		start := time.Now()
		res, err := s.ApplyProgramOps(ctx, orgA, projectA, userA, st.Program.ID, st.Program.Version, []store.ProgramOp{{Op: "move", IDs: []string{target}, Days: &days}})
		if err != nil {
			t.Fatal(err)
		}
		writes = append(writes, time.Since(start))
		st = res.ProgramState
		start = time.Now()
		if _, err := s.LoadProgram(ctx, orgA, projectA); err != nil {
			t.Fatal(err)
		}
		reads = append(reads, time.Since(start))
	}
	pct := func(d []time.Duration, p float64) time.Duration {
		d = append([]time.Duration(nil), d[1:]...) // the first call warms the pool
		sort.Slice(d, func(i, j int) bool { return d[i] < d[j] })
		return d[int(p*float64(len(d)-1))]
	}
	t.Logf("store move on 1,000 activities / 2,000 links: p50 %v p90 %v; load p50 %v p90 %v",
		pct(writes, .5), pct(writes, .9), pct(reads, .5), pct(reads, .9))
	if pct(writes, .9) > 250*time.Millisecond {
		t.Fatalf("a single move takes p90 %v, over the 250 ms program_write budget", pct(writes, .9))
	}
}

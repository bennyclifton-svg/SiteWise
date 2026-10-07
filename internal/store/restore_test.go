package store_test

import (
	"context"
	"testing"

	"sitewise/internal/store"
)

func TestRestoreFacts(t *testing.T) {
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
	if _, err := st.AppendEvent(ctx, orgA, "filing", docA, "{}"); err != nil {
		t.Fatal(err)
	}

	facts, err := st.RestoreFacts(ctx)
	if err != nil {
		t.Fatal(err)
	}
	a := orgFacts(t, facts, orgA)
	want := store.OrgCounts{OrgID: orgA, Users: 1, Memberships: 1, Projects: 1, Files: 1, Documents: 1, Decisions: 1, Passages: 1, Events: 1}
	if a != want {
		t.Fatalf("org A %+v want %+v", a, want)
	}
	if b := orgFacts(t, facts, orgB); b.Events != 0 || b.Documents != 1 {
		t.Fatalf("org B %+v", b)
	}
	if facts.ForeignKeys < 15 || facts.Unvalidated != 0 {
		t.Fatalf("constraints %d unvalidated %d", facts.ForeignKeys, facts.Unvalidated)
	}
	if len(facts.Migrations) == 0 || facts.Migrations[0] != "001_init.sql" {
		t.Fatalf("migrations %v", facts.Migrations)
	}
	for _, c := range facts.CrossOrg {
		if c.Rows != 0 {
			t.Fatalf("cross-org rows %+v", c)
		}
	}
	if len(facts.CrossOrg) == 0 {
		t.Fatal("no cross-org checks ran")
	}

	// An event naming another org's document breaks the boundary; nothing
	// in the schema prevents it, so the check must find it.
	if _, err := st.AppendEvent(ctx, orgA, "filing", docB, "{}"); err != nil {
		t.Fatal(err)
	}
	facts, err = st.RestoreFacts(ctx)
	if err != nil {
		t.Fatal(err)
	}
	var crossed int64
	for _, c := range facts.CrossOrg {
		crossed += c.Rows
	}
	if crossed == 0 {
		t.Fatalf("cross-org event not found: %+v", facts.CrossOrg)
	}
}

func orgFacts(t *testing.T, f store.RestoreFacts, orgID string) store.OrgCounts {
	t.Helper()
	for _, o := range f.Orgs {
		if o.OrgID == orgID {
			return o
		}
	}
	t.Fatalf("org %s missing", orgID)
	return store.OrgCounts{}
}

func TestRestoreFactsDetectCostContentChanges(t *testing.T) {
	ctx := context.Background()
	s := profileStore(t)
	p, e := s.CreateCostItem(ctx, orgA, projectA, userA, ci("Restore allowance"))
	if e != nil {
		t.Fatal(e)
	}
	before, e := s.RestoreFacts(ctx)
	if e != nil {
		t.Fatal(e)
	}
	again, e := s.RestoreFacts(ctx)
	if e != nil {
		t.Fatal(e)
	}
	find := func(f store.RestoreFacts) store.RestoreTableFacts {
		for _, r := range f.Tables {
			if r.Table == "cost_values" && r.OrgID == orgA {
				return r
			}
		}
		t.Fatal("cost values omitted")
		return store.RestoreTableFacts{}
	}
	if find(before) != find(again) {
		t.Fatal("stable content changed digest")
	}
	pool := rawPool(t)
	if _, e = pool.Exec(ctx, `UPDATE cost_values SET amount='123.45' WHERE org_id=$1::uuid AND plan_version_id=$2::uuid`, orgA, p.ID); e != nil {
		t.Fatal(e)
	}
	after, e := s.RestoreFacts(ctx)
	if e != nil {
		t.Fatal(e)
	}
	a, b := find(before), find(after)
	if a.Rows != b.Rows || a.SHA256 == b.SHA256 {
		t.Fatal("same-count value corruption missed", a, b)
	}
}

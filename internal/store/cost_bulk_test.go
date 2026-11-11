package store_test

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"sitewise/internal/costs"
	"sitewise/internal/store"
)

// Bulk actions on selected Cost Plan lines, each one write in one
// transaction: set category and exclusion, duplicate, delete.
func TestCostBulkSetDuplicateDelete(t *testing.T) {
	ctx := context.Background()
	s := profileStore(t)
	var p costs.Plan
	for _, label := range []string{"Alpha", "Beta", "Gamma", "Delta"} {
		in := ci(label)
		in.CostWrite = cw(p)
		var err error
		if p, err = s.CreateCostItem(ctx, orgA, projectA, userA, in); err != nil {
			t.Fatal(err)
		}
	}
	id := func(p costs.Plan, label string) string {
		t.Helper()
		for _, i := range p.Items {
			if i.Label == label {
				return i.ID
			}
		}
		t.Fatalf("no line %s", label)
		return ""
	}
	item := func(p costs.Plan, label string) costs.Item {
		for _, i := range p.Items {
			if i.Label == label {
				return i
			}
		}
		return costs.Item{}
	}
	// Beta holds a contract: a copy never carries it.
	var err error
	if p, err = s.PutCostValue(ctx, orgA, projectA, id(p, "Beta"), userA, "commitment",
		store.CostValueInput{PlanVersionID: p.ID, PlanVersion: p.Version, Value: cv("80.00")}); err != nil {
		t.Fatal(err)
	}

	yes, escalation := true, "escalation"
	if p, err = s.BulkCostItems(ctx, orgA, projectA, userA, store.CostBulkInput{CostWrite: cw(p), Op: "set",
		IDs: []string{id(p, "Alpha"), id(p, "Beta")}, Excluded: &yes, Assign: &escalation}); err != nil {
		t.Fatal(err)
	}
	for _, label := range []string{"Alpha", "Beta"} {
		if i := item(p, label); !i.Excluded || i.Category != "escalation" {
			t.Errorf("%s = excluded %v category %q", label, i.Excluded, i.Category)
		}
	}
	if i := item(p, "Gamma"); i.Excluded || i.Category != "contingency" {
		t.Errorf("an unselected line changed: %+v", i)
	}

	if p, err = s.BulkCostItems(ctx, orgA, projectA, userA, store.CostBulkInput{CostWrite: cw(p), Op: "duplicate", IDs: []string{id(p, "Beta")}}); err != nil {
		t.Fatal(err)
	}
	cp := item(p, "Beta (copy)")
	if cp.ID == "" || cp.Origin != "user" || cp.SourceKey != "" || cp.Category != "escalation" || !cp.Excluded {
		t.Fatalf("copy = %+v", cp)
	}
	if b := cp.Values["budget"]; b.Amount == nil || strings.TrimSuffix(string(*b.Amount), ".00") != "100" {
		t.Errorf("copy budget = %+v", b)
	}
	if c, ok := cp.Values["commitment"]; ok && c.Amount != nil {
		t.Errorf("copy carries the contract: %+v", c)
	}
	order := []string{}
	for _, i := range p.Items {
		order = append(order, i.Label)
	}
	at := -1
	for n, l := range order {
		if l == "Beta" {
			at = n
		}
	}
	if at+1 >= len(order) || order[at+1] != "Beta (copy)" {
		t.Errorf("copy placed at %v", order)
	}

	before := p.Version
	start := time.Now()
	if p, err = s.BulkCostItems(ctx, orgA, projectA, userA, store.CostBulkInput{CostWrite: cw(p), Op: "delete",
		IDs: []string{id(p, "Alpha"), id(p, "Gamma"), cp.ID}}); err != nil {
		t.Fatal(err)
	}
	t.Logf("bulk delete of 3 lines: %s", time.Since(start))
	if len(p.Items) != 2 || p.Version != before+1 || item(p, "Beta").ID == "" || item(p, "Delta").ID == "" {
		t.Fatalf("after delete: %d lines, version %d (was %d)", len(p.Items), p.Version, before)
	}

	stale := store.CostWrite{PlanVersionID: p.ID, Version: p.Version - 1}
	cases := []struct {
		name string
		org  string
		in   store.CostBulkInput
		want error
	}{
		{"stale plan", orgA, store.CostBulkInput{CostWrite: stale, Op: "delete", IDs: []string{id(p, "Beta")}}, store.ErrVersionConflict},
		{"another org", orgB, store.CostBulkInput{CostWrite: cw(p), Op: "delete", IDs: []string{id(p, "Beta")}}, store.ErrNotFound},
		{"unknown line", orgA, store.CostBulkInput{CostWrite: cw(p), Op: "delete", IDs: []string{"00000000-0000-4000-8000-000000000999"}}, store.ErrNotFound},
		{"no lines", orgA, store.CostBulkInput{CostWrite: cw(p), Op: "delete"}, costs.ErrInvalid},
		{"twice", orgA, store.CostBulkInput{CostWrite: cw(p), Op: "delete", IDs: []string{id(p, "Beta"), id(p, "Beta")}}, costs.ErrInvalid},
		{"unknown op", orgA, store.CostBulkInput{CostWrite: cw(p), Op: "paint", IDs: []string{id(p, "Beta")}}, costs.ErrInvalid},
		{"nothing to set", orgA, store.CostBulkInput{CostWrite: cw(p), Op: "set", IDs: []string{id(p, "Beta")}}, costs.ErrInvalid},
		{"bad category", orgA, store.CostBulkInput{CostWrite: cw(p), Op: "set", IDs: []string{id(p, "Beta")}, Assign: strPtr("marble")}, costs.ErrInvalid},
	}
	for _, c := range cases {
		user := userA
		if c.org == orgB {
			user = userB
		}
		if _, err := s.BulkCostItems(ctx, c.org, projectA, user, c.in); !errors.Is(err, c.want) {
			t.Errorf("%s: %v, want %v", c.name, err, c.want)
		}
	}
}

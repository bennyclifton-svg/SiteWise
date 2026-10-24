package store_test

import (
	"context"
	"errors"
	"reflect"
	"sitewise/internal/costs"
	"sitewise/internal/store"
	"sort"
	"testing"
	"time"
)

func TestCostOrderPreservesValuesAndFrozenHistory(t *testing.T) {
	ctx := context.Background()
	s := profileStore(t)
	var p costs.Plan
	for _, label := range []string{"Alpha", "Beta", "Gamma"} {
		in := ci(label)
		in.CostWrite = cw(p)
		var err error
		p, err = s.CreateCostItem(ctx, orgA, projectA, userA, in)
		if err != nil {
			t.Fatal(err)
		}
	}
	ids := []string{p.Items[2].ID, p.Items[0].ID, p.Items[1].ID}
	before, _ := costs.Summarize(p, "overall")
	stale := cw(p)
	var elapsed []time.Duration
	for n := 0; n < 20; n++ {
		start := time.Now()
		next, err := s.ReorderCostItems(ctx, orgA, projectA, userA, store.CostOrderInput{CostWrite: cw(p), IDs: ids})
		if err != nil {
			t.Fatal(err)
		}
		p = next
		elapsed = append(elapsed, time.Since(start))
	}
	sort.Slice(elapsed, func(i, j int) bool { return elapsed[i] < elapsed[j] })
	t.Logf("cost reorder p50=%s p90=%s; budget 100ms/250ms", elapsed[9], elapsed[17])
	if elapsed[9] > 100*time.Millisecond || elapsed[17] > 250*time.Millisecond {
		t.Fatal("cost write speed budget exceeded")
	}
	after, _ := costs.Summarize(p, "overall")
	if !reflect.DeepEqual(before, after) {
		t.Fatal("reorder changed money")
	}
	check := func(plan costs.Plan) {
		t.Helper()
		for pos, id := range ids {
			for _, item := range plan.Items {
				if item.ID == id && (item.DisplayOrder == nil || *item.DisplayOrder != pos+1) {
					t.Fatal("order lost", item)
				}
			}
		}
	}
	check(p)
	if _, err := s.ReorderCostItems(ctx, orgA, projectA, userA, store.CostOrderInput{CostWrite: stale, IDs: ids}); !errors.Is(err, store.ErrVersionConflict) {
		t.Fatal("stale write", err)
	}
	if _, err := s.ReorderCostItems(ctx, orgB, projectA, userB, store.CostOrderInput{CostWrite: cw(p), IDs: ids}); !errors.Is(err, store.ErrNotFound) {
		t.Fatal("foreign write", err)
	}
	for _, bad := range [][]string{{ids[0]}, {ids[0], ids[0], ids[1]}, {"missing", ids[1], ids[2]}} {
		if _, err := s.ReorderCostItems(ctx, orgA, projectA, userA, store.CostOrderInput{CostWrite: cw(p), IDs: bad}); err == nil {
			t.Fatal("invalid order accepted", bad)
		}
	}
	other := ci("Council charges")
	other.Category = "authority_fees"
	other.CostWrite = cw(p)
	withOther, createErr := s.CreateCostItem(ctx, orgA, projectA, userA, other)
	if createErr != nil {
		t.Fatal(createErr)
	}
	p = withOther
	var otherID string
	for _, item := range p.Items {
		if item.Label == "Council charges" {
			otherID = item.ID
		}
	}
	if _, err := s.ReorderCostItems(ctx, orgA, projectA, userA, store.CostOrderInput{CostWrite: cw(p), IDs: append(append([]string{}, ids...), otherID)}); !errors.Is(err, costs.ErrInvalid) {
		t.Fatal("cross-section move", err)
	}
	baseline := p.ID
	var err error
	p, err = s.BaselineCostPlan(ctx, orgA, projectA, userA, cw(p))
	if err != nil {
		t.Fatal(err)
	}
	check(p)
	frozen, err := s.ReadCostPlan(ctx, orgA, projectA, baseline)
	if err != nil {
		t.Fatal(err)
	}
	check(frozen)
	if _, err = s.ReorderCostItems(ctx, orgA, projectA, userA, store.CostOrderInput{CostWrite: cw(frozen), IDs: ids}); !errors.Is(err, store.ErrFrozenCostPlan) {
		t.Fatal("frozen write", err)
	}
}

package store_test

import (
	"context"
	"encoding/json"
	"sitewise/internal/costs"
	"sitewise/internal/store"
	"strings"
	"testing"
)

func TestPMPUsesCostSummaryAndFreezesLedgerInputs(t *testing.T) {
	ctx := context.Background()
	s, b, _ := workStore(t)
	if err := s.RebuildProfile(ctx, orgA, projectA, "", b.Compute); err != nil {
		t.Fatal(err)
	}
	p, err := s.CreateCostItem(ctx, orgA, projectA, userA, ci("Risk allowance"))
	if err != nil {
		t.Fatal(err)
	}
	p, err = s.PutCostValue(ctx, orgA, projectA, p.Items[0].ID, userA, "estimate", store.CostValueInput{PlanVersionID: p.ID, PlanVersion: p.Version, Value: costs.Value{ValueState: "known", Low: cm("120.00"), High: cm("180.00"), Origin: "user", Meaning: "allowance"}})
	if err != nil {
		t.Fatal(err)
	}
	r, err := s.CreateReport(ctx, orgA, projectA, userA, "pmp", "")
	if err != nil {
		t.Fatal(err)
	}
	d, err := s.RefreshReport(ctx, orgA, r.ID, userA, "test", true)
	if err != nil {
		t.Fatal(err)
	}
	found, foundRange := false, false
	for _, section := range d.Sections {
		for _, b := range section.Blocks {
			if b.ID == "cost:summary:forecast" {
				foundRange = b.Text == "120.00–180.00 AUD"
			}
			if strings.HasPrefix(b.ID, "cost:line:") || strings.Contains(b.Text, "price to be returned") {
				t.Fatal("tender schedule leaked into PMP", b)
			}
			if b.ID == "cost:variance" {
				var basis struct {
					Inputs json.RawMessage `json:"inputs"`
				}
				if err := json.Unmarshal(b.Basis, &basis); err != nil {
					t.Fatal(err)
				}
				if !strings.Contains(string(basis.Inputs), p.ID) || !strings.Contains(string(basis.Inputs), p.Items[0].ID) || !strings.Contains(string(basis.Inputs), "100.00") {
					t.Fatal("summary inputs not frozen", string(basis.Inputs))
				}
				found = true
			}
		}
	}
	if !found {
		t.Fatal("cost variance missing")
	}
	if !foundRange {
		t.Fatal("known cost range was hidden or called incomplete")
	}
}

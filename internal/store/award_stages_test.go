package store_test

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"sitewise/internal/procurement"
	"sitewise/internal/proposals"
	"sitewise/internal/store"
)

// A letter staged as "Fixed fee $1,000: Concept $400, Documents $600" can be
// awarded as its two stages, but only when they add up to the confirmed fee.
func TestAwardByStageFromQuotedFees(t *testing.T) {
	ctx := context.Background()
	s, _, _ := workStore(t)
	row, _ := procurement.DisciplinePackage("consultant.town_planner", "Town Planner")
	pkg, err := s.CreateDisciplinePackage(ctx, orgA, projectA, userA, row)
	if err != nil {
		t.Fatal(err)
	}
	firm, err := s.CreateFirm(ctx, orgA, store.FirmFields{Name: "Staged Studio"})
	if err != nil {
		t.Fatal(err)
	}
	id, err := s.AddFirmSubmission(ctx, orgA, projectA, pkg.ID, firm, time.Now())
	if err != nil {
		t.Fatal(err)
	}
	amt := func(s string) *string { return &s }
	review := proposals.QuoteReview{Coverage: map[string]proposals.QuoteCoverage{}, Issues: []string{}, Fees: []proposals.QuotedFee{
		{ID: "a1", Label: "Fixed professional fee", Amount: amt("1000.00"), Role: proposals.RoleTotal, Source: "Fixed fee $1,000 staged as follows:"},
		{ID: "a2", Label: "Concept", Amount: amt("400.00"), Role: proposals.RoleComponent, Source: "| Concept | $400 |"},
		{ID: "a3", Label: "Documents", Amount: amt("600.00"), Role: proposals.RoleComponent, Source: "| Documents | $600 |"},
		{ID: "a4", Label: "Renders", Amount: amt("50.00"), Role: "Conditional / optional fee", Source: "Optional renders $50"},
	}}
	raw, _ := json.Marshal(review)
	pool := rawPool(t)
	if _, err := pool.Exec(ctx, `UPDATE submissions SET quote_review=$3::jsonb WHERE org_id=$1 AND id=$2`, orgA, id, string(raw)); err != nil {
		t.Fatal(err)
	}
	confirm := func(amount string) {
		t.Helper()
		if err := s.SetSubmissionCell(ctx, orgA, projectA, id, "comparison:appointment", store.CellInput{State: "priced", Amount: amount, Note: "Full appointment per letter"}); err != nil {
			t.Fatal(err)
		}
	}
	stages := func() []proposals.QuotedFee {
		t.Helper()
		table, err := s.ReadPriceTable(ctx, orgA, projectA, pkg.ID)
		if err != nil {
			t.Fatal(err)
		}
		return table.Columns[0].AwardStages
	}
	byStage := []store.AwardLine{{Label: "Concept", QuoteIDs: []string{"a2"}}, {Label: "Documents", QuoteIDs: []string{"a3"}}}

	// Stages that do not add to the confirmed fee are not offered.
	confirm("900")
	if got := stages(); got != nil {
		t.Fatalf("stages offered against a different fee: %+v", got)
	}
	if _, err := s.PreviewAward(ctx, orgA, projectA, pkg.ID, store.AwardInput{SubmissionID: id, Lines: byStage}); !errors.Is(err, store.ErrInvalidInput) {
		t.Fatalf("stage award against a different fee: %v", err)
	}

	confirm("1000")
	if got := stages(); len(got) != 2 || got[0].ID != "a2" || got[1].ID != "a3" {
		t.Fatalf("stages %+v", got)
	}
	refused := map[string][]store.AwardLine{
		"a stage left out":        {{Label: "Concept", QuoteIDs: []string{"a2"}}},
		"the total with a stage":  {{Label: "Fee", QuoteIDs: []string{"a1"}}, {Label: "Documents", QuoteIDs: []string{"a3"}}},
		"an optional fee":         {{Label: "Concept", QuoteIDs: []string{"a2"}}, {Label: "Documents", QuoteIDs: []string{"a3", "a4"}}},
		"a stage twice":           {{Label: "Concept", QuoteIDs: []string{"a2"}}, {Label: "Again", QuoteIDs: []string{"a2", "a3"}}},
		"stages with the fee row": {{Label: "Concept", QuoteIDs: []string{"a2"}}, {Label: "Fee", Keys: []string{"comparison:appointment"}}},
		"an unknown source line":  {{Label: "Concept", QuoteIDs: []string{"a2"}}, {Label: "Documents", QuoteIDs: []string{"a3", "zz"}}},
	}
	for name, lines := range refused {
		if _, err := s.PreviewAward(ctx, orgA, projectA, pkg.ID, store.AwardInput{SubmissionID: id, Lines: lines}); !errors.Is(err, store.ErrInvalidInput) {
			t.Fatalf("%s allowed: %v", name, err)
		}
	}
	if _, err := s.PreviewAward(ctx, orgA, projectA, pkg.ID, store.AwardInput{SubmissionID: id, Lines: byStage, Total: "999.00"}); !errors.Is(err, store.ErrInvalidInput) {
		t.Fatalf("wrong running total allowed: %v", err)
	}
	// Combining stages under one name is still the PM's choice.
	combined, err := s.PreviewAward(ctx, orgA, projectA, pkg.ID, store.AwardInput{SubmissionID: id, Lines: []store.AwardLine{{Label: "Design", QuoteIDs: []string{"a2", "a3"}}}})
	if err != nil || combined.Total != 100000 || len(combined.Lines) != 1 {
		t.Fatalf("combined %+v %v", combined, err)
	}
	// The single confirmed fee line still works.
	if _, err := s.PreviewAward(ctx, orgA, projectA, pkg.ID, store.AwardInput{SubmissionID: id, Lines: []store.AwardLine{{Label: "Professional fee", Keys: []string{"comparison:appointment"}}}}); err != nil {
		t.Fatalf("one-total award: %v", err)
	}

	v, err := s.Award(ctx, orgA, projectA, pkg.ID, userA, store.AwardInput{SubmissionID: id, Lines: byStage, Total: "1000.00"})
	if err != nil || v.Total != 100000 || len(v.Lines) != 2 || v.Lines[0].Cents != 40000 || v.Lines[1].Cents != 60000 {
		t.Fatalf("award %+v %v", v, err)
	}
	plan, err := s.ReadCostPlan(ctx, orgA, projectA, "")
	if err != nil {
		t.Fatal(err)
	}
	got := map[string]string{}
	for _, it := range plan.Items {
		if it.ParentItemID == v.ParentItemID {
			c := it.Values["commitment"]
			if c.Amount == nil {
				t.Fatalf("no commitment on %s", it.Label)
			}
			got[it.Label] = string(*c.Amount)
			if _, ok := it.Values["budget"]; ok {
				t.Fatalf("award wrote a budget on %s", it.Label)
			}
		}
	}
	if len(got) != 2 || got["Concept"] != "400.00" || got["Documents"] != "600.00" {
		t.Fatalf("children %v", got)
	}
	award, err := s.ReadAward(ctx, orgA, projectA, pkg.ID)
	if err != nil || award == nil || len(award.Lines) != 2 || award.Total != 100000 || len(award.Lines[0].QuoteIDs) != 1 || award.Lines[0].QuoteIDs[0] != "a2" {
		t.Fatalf("read award %+v %v", award, err)
	}
	if err := s.UndoAward(ctx, orgA, projectA, pkg.ID, userA); err != nil {
		t.Fatal(err)
	}
	if award, _ := s.ReadAward(ctx, orgA, projectA, pkg.ID); award != nil {
		t.Fatalf("undo left %+v", award)
	}
}

package store

import (
	"strings"
	"testing"

	"sitewise/internal/costs"
)

// The spread says what the budget's words say about contingency and GST,
// and never quantifies a contingency the document leaves open (Walsh brief,
// 2026-10-27).
func TestBudgetFramingNote(t *testing.T) {
	b := &costs.ConstructionBudget{Stated: costs.StatedBudget{Text: "$920,000", Amount: "920000.00", GST: "not_stated"}}
	note := budgetFramingNote(b, "t confirmed:** working budget ceiling **$920,000** (including contingency for latent con")
	if !strings.Contains(note, "includes a contingency the document does not quantify") || !strings.Contains(note, "does not say whether the budget includes GST") {
		t.Fatalf("Walsh framing note %q", note)
	}
	ex := &costs.ConstructionBudget{Stated: costs.StatedBudget{Text: "$2.4m ex GST", Amount: "2400000.00", GST: "ex"}}
	if note := budgetFramingNote(ex, "The construction budget is $2.4m ex GST."); note != "" {
		t.Fatalf("a plain ex-GST budget gained a note %q", note)
	}
}

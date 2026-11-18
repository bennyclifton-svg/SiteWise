package reports

import "testing"

// Plan 4's rules: open RFIs past their response date, and register rows
// with a possible answer to check.
func TestAttentionRegisterRules(t *testing.T) {
	in := AttentionInput{Today: "2026-12-01", Delivery: []AttentionRecord{
		{ID: "late", Kind: "rfi", Status: "open", TargetDate: "2026-11-27", LateDays: 2},
		{ID: "draft-late", Kind: "rfi", Status: "draft", TargetDate: "2026-11-27", LateDays: 2},
		{ID: "answered", Kind: "rfi", Status: "answered", TargetDate: "2026-11-20", LateDays: 0, ToCheck: false},
		{ID: "weekend", Kind: "rfi", Status: "open", TargetDate: "2026-11-29"},
		{ID: "check", Kind: "rfi", Status: "open", TargetDate: "2026-12-10", ToCheck: true},
		{ID: "risk", Kind: "risk", Status: "open", LateDays: 3},
	}}
	items := Attention(in)
	over, ok := AttentionFor(items, RuleRFIOverdue)
	if !ok || over.Count != 2 || over.IDs[0] != "late" || over.IDs[1] != "draft-late" || over.Text() != "2 RFIs are past the response date" {
		t.Fatalf("overdue %+v", over)
	}
	check, ok := AttentionFor(items, RuleRegisterToCheck)
	if !ok || check.Count != 1 || check.IDs[0] != "check" || check.Text() != "1 register row has a possible answer to check" {
		t.Fatalf("to check %+v", check)
	}
	if _, ok := AttentionFor(Attention(AttentionInput{Today: "2026-12-01"}), RuleRFIOverdue); ok {
		t.Fatal("no rows, no item")
	}
}

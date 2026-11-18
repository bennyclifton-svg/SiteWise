package reports

import "testing"

// Plan 6's rules: consent conditions open before a gate the Program dates
// within the window, and ones still open at a gate already reached.
func TestAttentionConditionRules(t *testing.T) {
	for _, tc := range []struct {
		name        string
		r           AttentionRecord
		due, atGate bool
	}{
		{"due in the window", AttentionRecord{Kind: "condition", Status: "open", Gate: "before_cc", GateDate: "2026-12-20"}, true, false},
		{"evidence filed is still open", AttentionRecord{Kind: "condition", Status: "evidence_filed", Gate: "before_cc", GateDate: "2026-12-29"}, true, false},
		{"past the window", AttentionRecord{Kind: "condition", Status: "open", Gate: "before_cc", GateDate: "2026-12-30"}, false, false},
		{"gate reached", AttentionRecord{Kind: "condition", Status: "open", Gate: "before_occupation", GateDate: "2026-12-01"}, false, true},
		{"satisfied", AttentionRecord{Kind: "condition", Status: "satisfied", Gate: "before_cc", GateDate: "2026-12-02"}, false, false},
		{"not applicable", AttentionRecord{Kind: "condition", Status: "not_applicable", Gate: "before_cc", GateDate: "2026-11-02"}, false, false},
		{"no Program date: unknown stays unknown", AttentionRecord{Kind: "condition", Status: "open", Gate: "before_cc"}, false, false},
		{"no gate", AttentionRecord{Kind: "condition", Status: "open", GateDate: "2026-12-02"}, false, false},
		{"an RFI is not a condition", AttentionRecord{Kind: "rfi", Status: "open", Gate: "before_cc", GateDate: "2026-12-02"}, false, false},
	} {
		if got := conditionDue(tc.r, "2026-12-01"); got != tc.due {
			t.Errorf("%s: due %v", tc.name, got)
		}
		if got := conditionAtGate(tc.r, "2026-12-01"); got != tc.atGate {
			t.Errorf("%s: at gate %v", tc.name, got)
		}
	}
	items := Attention(AttentionInput{Today: "2026-12-01", Delivery: []AttentionRecord{
		{ID: "a", Kind: "condition", Status: "open", Gate: "before_cc", GateDate: "2026-12-10"},
		{ID: "b", Kind: "condition", Status: "open", Gate: "before_cc", GateDate: "2026-12-10"},
		{ID: "c", Kind: "condition", Status: "open", Gate: "before_work", GateDate: "2026-11-30"},
	}})
	due, ok := AttentionFor(items, RuleConditionsDue)
	if !ok || due.Count != 2 || due.Text() != "2 consent conditions are open before a gate in the next 28 days" {
		t.Fatalf("due %+v", due)
	}
	at, ok := AttentionFor(items, RuleConditionsAtGate)
	if !ok || at.Count != 1 || at.IDs[0] != "c" || at.Text() != "1 consent condition is still open at a gate the Program has reached" {
		t.Fatalf("at gate %+v", at)
	}
	if !ConditionOpenBefore(AttentionRecord{Kind: "condition", Status: "open", Gate: "before_cc"}, "before_cc") || ConditionOpenBefore(AttentionRecord{Kind: "condition", Status: "open", Gate: "before_cc"}, "before_occupation") {
		t.Fatal("open before")
	}
}

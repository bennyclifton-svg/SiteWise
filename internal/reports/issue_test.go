package reports

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestIssueBudgetDisclosureDoesNotLeakCitationBasis(t *testing.T) {
	s := []Section{{ID: "fee_return", Blocks: []Block{{ID: "cost:line:stable-id", Label: "Design fee", Text: "Return price against stable-id", Origin: "calculation", ReviewStatus: "accepted_for_planning", Meaning: "requirement", Basis: json.RawMessage(`{"cost_item_id":"stable-id","stage_id":"design","budget":{"amount":"12734.19"}}`)}}}}
	for _, disclose := range []bool{false, true} {
		out, refs, err := IssueContent(s, "rfp", disclose)
		if err != nil {
			t.Fatal(err)
		}
		raw, _ := json.Marshal(map[string]any{"sections": out, "references": refs})
		if strings.Contains(string(raw), "12734.19") != disclose {
			t.Fatal("budget disclosure mismatch", disclose)
		}
		if !strings.Contains(string(raw), "stable-id") || !strings.Contains(string(raw), "design") {
			t.Fatal("pricing identity lost")
		}
	}
}

func TestChangesSinceIssueIncludesDeletedWording(t *testing.T) {
	before := []Section{{Blocks: []Block{{ID: "old", Label: "Obligation", Text: "Keep supply live"}, {ID: "same", Text: "old"}}}}
	after := []Section{{Blocks: []Block{{ID: "same", Text: "new"}, {ID: "added", Text: "new scope"}}}}
	got := ChangesSinceIssue(before, after)
	if len(got) != 3 || got[1].Kind != "removed" {
		t.Fatal(got)
	}
}

func TestCanonicalSnapshotNestedObjectOrderAndExactNumbers(t *testing.T) {
	snapshot := IssueSnapshot{SchemaVersion: 1, ReportID: "report", VersionID: "version", Kind: "rfp", Title: "RFP", RendererVersion: "renderer", Sections: []Section{{ID: "brief", Blocks: []Block{{ID: "a", Basis: json.RawMessage(`{"long_key":{"b":9007199254740993,"a":"safe"},"z":2}`)}}}}}
	first, hash, e := CanonicalSnapshot(snapshot)
	if e != nil {
		t.Fatal(e)
	}
	snapshot.Sections[0].Blocks[0].Basis = json.RawMessage(`{"z":2,"long_key":{"a":"safe","b":9007199254740993}}`)
	second, hash2, e := CanonicalSnapshot(snapshot)
	if e != nil || hash != hash2 || string(first) != string(second) {
		t.Fatal("order-dependent hash", hash, hash2, e)
	}
	if !strings.Contains(string(second), "9007199254740993") {
		t.Fatal("number changed")
	}
}

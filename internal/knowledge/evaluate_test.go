package knowledge_test

import (
	"encoding/json"
	"strings"
	"testing"

	"sitewise/internal/knowledge"
)

func TestDerivationsStayUnknown(t *testing.T) {
	cat, err := knowledge.Load(writeFixture(t))
	if err != nil {
		t.Fatal(err)
	}
	same := knowledge.Scope{Level: "building", Ref: "b1"}
	ok := []knowledge.Fact{
		{Determinant: "ncc_class", Value: "5", Scope: same, DocumentID: "d1", PassageID: "p1"},
		{Determinant: "rise_in_storeys", Value: "4", Scope: same, DocumentID: "d1", PassageID: "p1"},
	}
	got := cat.Derive("rule.demo.type", ok)
	if got.State != knowledge.StateDetermined || got.Value != "A" {
		t.Fatalf("%+v", got)
	}

	repo := loadRepo(t)
	draft := repo.Derive("rule.ncc.type-of-construction", ok)
	if draft.State != knowledge.StateUnknown || draft.Reason != "unreviewed" {
		t.Fatalf("%+v", draft)
	}
	pending := repo.Derive("rule.ncc.energy-monitoring", ok)
	if pending.State != knowledge.StateUnknown || pending.Reason != "pending" {
		t.Fatalf("%+v", pending)
	}

	cases := []struct {
		rule   string
		facts  []knowledge.Fact
		reason string
	}{
		{"rule.demo.missing", ok, "missing_table"},
		{"rule.demo.pending", ok, "pending"},
		{"rule.demo.draft", ok, "unreviewed"},
		{"rule.demo.draft-table", []knowledge.Fact{
			{Determinant: "ncc_class", Value: "5", Scope: same, DocumentID: "d1", PassageID: "p1"},
		}, "unreviewed"},
		{"rule.demo.type", []knowledge.Fact{
			{Determinant: "ncc_class", Value: "5", Scope: knowledge.Scope{Level: "building", Ref: "b1"}, DocumentID: "d1", PassageID: "p1"},
			{Determinant: "ncc_class", Value: "5", Scope: knowledge.Scope{Level: "part", Ref: "retail"}, DocumentID: "d2", PassageID: "p2"},
			{Determinant: "rise_in_storeys", Value: "4", Scope: knowledge.Scope{Level: "building", Ref: "b1"}, DocumentID: "d1", PassageID: "p1"},
		}, "mixed_scope"},
		{"rule.demo.type", []knowledge.Fact{
			{Determinant: "ncc_class", Value: "5", Scope: same, DocumentID: "d1", PassageID: "p1"},
			{Determinant: "ncc_class", Value: "6", Scope: same, DocumentID: "d2", PassageID: "p2"},
			{Determinant: "rise_in_storeys", Value: "4", Scope: same, DocumentID: "d1", PassageID: "p1"},
		}, "conflict"},
	}
	for _, tc := range cases {
		t.Run(tc.reason, func(t *testing.T) {
			got := cat.Derive(tc.rule, tc.facts)
			if got.State != knowledge.StateUnknown || got.Reason != tc.reason {
				t.Fatalf("%+v", got)
			}
			if tc.reason == "conflict" || tc.reason == "mixed_scope" {
				if len(got.Provenance) < 2 {
					t.Fatalf("provenance %+v", got.Provenance)
				}
			}
		})
	}
}

func TestFirePilotIsCoverage(t *testing.T) {
	cat := loadRepo(t)
	present := []string{
		"fire-active.fire-water",
		"fire-active.sprinklers",
		"electrical",
		"electrical.standby-power",
		"electrical.switchboards",
		"structure",
		"fire-active.detection",
		"mechanical.fire-mode-air-systems",
		"mechanical",
	}
	yes := true
	no := false
	report := cat.FirePilot(present, []knowledge.Answer{
		{QuestionID: "if:if.fire-water-supplies-sprinklers#0", Addressed: &yes, DocumentID: "d1", PassageID: "p1"},
		{QuestionID: "if:if.fire-water-supplies-sprinklers#0", Addressed: &no, DocumentID: "d2", PassageID: "p2"},
	})
	if len(report.Edges) == 0 || len(report.Coverage) == 0 {
		t.Fatalf("%+v", report)
	}
	var water knowledge.QuestionCoverage
	for _, coverage := range report.Coverage {
		if !coverage.Draft {
			t.Fatalf("draft coverage %+v", coverage)
		}
		for _, q := range coverage.Questions {
			if q.ID == "if:if.fire-water-supplies-sprinklers#0" {
				water = q
			}
			if q.State != knowledge.StateUnknown && q.State != knowledge.StateAddressed && q.State != knowledge.StateNotAddressed {
				t.Fatalf("state %s", q.State)
			}
		}
	}
	if water.State != knowledge.StateUnknown || len(water.Provenance) != 2 {
		t.Fatalf("%+v", water)
	}
	body, err := json.Marshal(report)
	if err != nil {
		t.Fatal(err)
	}
	text := strings.ToLower(string(body))
	if strings.Contains(text, "complies") || strings.Contains(text, "compliant") {
		t.Fatalf("pilot claimed compliance: %s", body)
	}
}

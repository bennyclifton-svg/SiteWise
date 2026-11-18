package delivery

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestConditionStatusesAndOpen(t *testing.T) {
	for _, tc := range []struct {
		status string
		open   bool
	}{{"open", true}, {"evidence_filed", true}, {"satisfied", false}, {"not_applicable", false}} {
		c := Content{Kind: "condition", Title: "Condition 46 · Landscape plan", Status: tc.status, Details: json.RawMessage(`{"number":"46"}`)}
		if err := Validate(c); err != nil {
			t.Fatalf("%s: %v", tc.status, err)
		}
		if Open("condition", tc.status) != tc.open {
			t.Errorf("Open(condition, %s) = %v", tc.status, !tc.open)
		}
	}
	if DefaultStatus("condition") != "open" {
		t.Fatal("default status")
	}
	long := strings.Repeat("x", MaxConditionText+1)
	for _, details := range []string{
		`{"number":"46","invented":true}`,
		`{"number":"46","text":"` + long + `"}`,
		`{"number":"46","stage":"before_lunch"}`,
		`{"number":"46","evidence":[{"document_id":"not-a-uuid"}]}`,
		`{"number":"46","source":{"document_id":"x","page":2}}`,
	} {
		if Validate(Content{Kind: "condition", Status: "open", Title: "T", Details: json.RawMessage(details)}) == nil {
			t.Errorf("accepted %s", details[:min(60, len(details))])
		}
	}
	if Validate(Content{Kind: "condition", Status: "answered", Title: "T", Details: json.RawMessage(`{}`)}) == nil {
		t.Error("an RFI status was accepted on a condition")
	}
	ok := `{"consent_ref":"DA/2025/0733","number":"A1","heading":"Landscape plan","text":"` + strings.Repeat("y", MaxConditionText) + `","stage":"before_cc","evidence_needed":true,"evidence":[{"document_id":"0b5f1c1e-1111-4111-8111-111111111111","linked_by":"u","linked_at":"2026-11-18"}],"source":{"document_id":"0b5f1c1e-1111-4111-8111-111111111112","page":12}}`
	if err := Validate(Content{Kind: "condition", Status: "open", Title: "T", Details: json.RawMessage(ok)}); err != nil {
		t.Fatal(err)
	}
}

func TestConditionTitleAndEditKeepsTheInstrument(t *testing.T) {
	if got := ConditionTitle("46", "Landscape plan", "ignored"); got != "Condition 46 · Landscape plan" {
		t.Fatalf("title %q", got)
	}
	if got := ConditionTitle("12", "", strings.Repeat("word ", 60)); len([]rune(got)) > 140 || !strings.HasPrefix(got, "Condition 12 · word word") || !strings.HasSuffix(got, "…") {
		t.Fatalf("title from text %q", got)
	}
	yes := true
	before := ConditionDetails{Number: "46", Heading: "Landscape plan", Text: "The council's words.", Stage: GateConstructionCertificate,
		Evidence: []ConditionEvidence{{DocumentID: "0b5f1c1e-1111-4111-8111-111111111111"}}}
	after := ConditionDetails{Number: "47", Heading: "Renamed", Text: "Reworded", Stage: GateOccupation, Discipline: "consultant.landscape", EvidenceNeeded: &yes}
	got := MergeConditionEdit(before, after)
	if got.Number != "46" || got.Heading != "Landscape plan" || got.Text != "The council's words." || len(got.Evidence) != 1 {
		t.Fatalf("the instrument's words changed: %+v", got)
	}
	if got.Stage != GateOccupation || got.Discipline != "consultant.landscape" || got.EvidenceNeeded == nil || !*got.EvidenceNeeded {
		t.Fatalf("the PM's edit was lost: %+v", got)
	}
	if !HeldAt(GateConstructionCertificate) || HeldAt(GateDuringWorks) || HeldAt(GateOngoing) || HeldAt(GateThroughout) {
		t.Fatal("gates held at")
	}
}

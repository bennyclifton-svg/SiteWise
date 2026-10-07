package delivery

import (
	"encoding/json"
	"testing"
)

func TestDeliveryKindStatusAndDetails(t *testing.T) {
	for _, kind := range []string{"activity", "milestone", "action", "risk", "issue", "decision", "approval"} {
		c := Content{Kind: kind, Title: "Record", Status: DefaultStatus(kind), Details: json.RawMessage(`{}`)}
		if err := Validate(c); err != nil {
			t.Fatalf("%s: %v", kind, err)
		}
		c.Status = "invented"
		if Validate(c) == nil {
			t.Fatal("unknown status", kind)
		}
	}
	for _, tc := range []struct{ kind, status, details string }{
		{"risk", "approved", `{}`},
		{"approval", "submitted", `{"progress":100}`},
		{"approval", "approved", `{"submitted_on":"2026-10-06","determined_on":"2026-10-05"}`},
		{"risk", "open", `{"likelihood":5}`},
		{"activity", "in_progress", `null`},
		{"decision", "decided", `{"options":["A"]}`},
		{"decision", "decided", `{"options":["A"],"chosen":"B"}`},
		{"decision", "open", `{"options":["A","A"]}`},
	} {
		if err := Validate(Content{Kind: tc.kind, Status: tc.status, Title: "Test", Details: json.RawMessage(tc.details)}); err == nil {
			t.Fatalf("accepted %+v", tc)
		}
	}
	if err := Validate(Content{Kind: "decision", Status: "decided", Title: "Choose", Details: json.RawMessage(`{"options":["A","B"],"chosen":"B"}`)}); err != nil {
		t.Fatal(err)
	}
}

func TestDeliveryDatesAndReferenceShape(t *testing.T) {
	c := Content{Kind: "milestone", Title: "Handover", Status: "planned", Details: json.RawMessage(`{}`)}
	for _, value := range []string{"2026-02-29", "2026-1-01", "0000-01-01", "today", "2026-01-01T00:00:00Z", ""} {
		c.TargetDate = &value
		if Validate(c) == nil {
			t.Fatal("invalid date accepted", value)
		}
	}
	valid := "2028-02-29"
	c.TargetDate = &valid
	if err := Validate(c); err != nil {
		t.Fatal(err)
	}
	c.StageID = "stage"
	if Validate(c) == nil {
		t.Fatal("stage without package")
	}
	for _, tc := range []struct{ kind, status string }{{"approval", "approved"}, {"milestone", "achieved"}, {"action", "complete"}, {"risk", "closed"}, {"decision", "decided"}, {"issue", "resolved"}} {
		if Open(tc.kind, tc.status) {
			t.Fatal("terminal status counted open", tc)
		}
	}
}

package delivery

import (
	"encoding/json"
	"testing"

	"sitewise/internal/program"
)

func TestRegisterKindsStatusesAndOpen(t *testing.T) {
	for _, tc := range []struct {
		kind, status string
		open         bool
	}{
		{"rfi", "draft", true}, {"rfi", "open", true}, {"rfi", "answered", true},
		{"rfi", "closed", false}, {"rfi", "withdrawn", false},
		{"notice", "open", true}, {"notice", "closed", false}, {"notice", "withdrawn", false},
	} {
		c := Content{Kind: tc.kind, Title: "Basement wall waterproofing", Status: tc.status, Details: json.RawMessage(`{}`)}
		if err := Validate(c); err != nil {
			t.Fatalf("%s %s: %v", tc.kind, tc.status, err)
		}
		if Open(tc.kind, tc.status) != tc.open {
			t.Errorf("Open(%s, %s) = %v", tc.kind, tc.status, !tc.open)
		}
	}
	if DefaultStatus("rfi") != "draft" || DefaultStatus("notice") != "open" {
		t.Fatal("default statuses")
	}
	for _, tc := range []struct{ kind, status, details string }{
		{"rfi", "decided", `{}`},
		{"notice", "answered", `{}`},
		{"rfi", "open", `{"question":"x","invented":1}`},
		{"rfi", "open", `{"raised_on":"18/11/2026"}`},
		{"rfi", "open", `{"answer_documents":["not-a-uuid"]}`},
		{"rfi", "open", `{"answer_documents":["0b5f1c1e-1111-4111-8111-111111111111","0b5f1c1e-1111-4111-8111-111111111111"]}`},
		{"notice", "open", `{"message_id":"m1"}`},
		{"notice", "open", `{"time_bar":"2026-12-01"}`},
	} {
		if Validate(Content{Kind: tc.kind, Status: tc.status, Title: "T", Details: json.RawMessage(tc.details)}) == nil {
			t.Errorf("accepted %+v", tc)
		}
	}
	ok := Content{Kind: "rfi", Status: "open", Title: "T", Details: json.RawMessage(`{"question":"Which membrane?","ref":"B-12","raised_on":"2026-11-20","answer_documents":["0b5f1c1e-1111-4111-8111-111111111111"]}`)}
	if err := Validate(ok); err != nil {
		t.Fatal(err)
	}
	if Validate(Content{Kind: "risk", Status: "open", Title: "T", DeterminantID: "d", Details: json.RawMessage(`{}`)}) == nil {
		t.Error("a determinant without its system was accepted")
	}
}

func TestRegisterKindChangeRefused(t *testing.T) {
	for _, tc := range []struct {
		from, to string
		ok       bool
	}{{"rfi", "risk", false}, {"issue", "rfi", false}, {"rfi", "rfi", true}, {"risk", "issue", true}, {"notice", "action", true}} {
		if (CheckKindChange(tc.from, tc.to) == nil) != tc.ok {
			t.Errorf("CheckKindChange(%s, %s)", tc.from, tc.to)
		}
	}
}

func TestRegisterWith(t *testing.T) {
	for status, want := range map[string]string{"draft": "Structural Co", "open": "Structural Co", "answered": "You", "closed": "Structural Co", "withdrawn": "Structural Co"} {
		if got := With(Content{Kind: "rfi", Status: status, OwnerText: "Structural Co"}); got != want {
			t.Errorf("With(rfi %s) = %q, want %q", status, got, want)
		}
	}
	if With(Content{Kind: "notice", Status: "open", OwnerText: "Builder"}) != "Builder" {
		t.Error("a notice is with whoever must act")
	}
}

func TestRegisterDueDate(t *testing.T) {
	friday, wednesday := "2026-11-20", "2026-12-23"
	if d := DueDate(program.Calendar{}, &friday, 5); d == nil || *d != "2026-11-27" {
		t.Errorf("a Friday raise: %v", d)
	}
	christmas := program.Calendar{Days: []program.CalendarDay{{Day: "2026-12-25", Kind: program.DayPublicHoliday}}}
	if d := DueDate(christmas, &wednesday, 2); d == nil || *d != "2026-12-28" {
		t.Errorf("Christmas as a Program holiday: %v", d)
	}
	if d := DueDate(program.Calendar{}, &wednesday, 2); d == nil || *d != "2026-12-25" {
		t.Errorf("no program, Monday to Friday: %v", d)
	}
	if DueDate(program.Calendar{}, nil, 5) != nil {
		t.Error("no raised date must leave Due blank")
	}
}

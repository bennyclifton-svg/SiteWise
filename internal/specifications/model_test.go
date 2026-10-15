package specifications

import (
	"strings"
	"testing"
)

func TestValueRolesUnknownAndZero(t *testing.T) {
	zero := 0.0
	a := Assertion{Key: "width", Meaning: "selection", Value: Value{State: "known", Number: &zero, Unit: "mm"}}
	if err := ValidateAssertion(a); err != nil {
		t.Fatal(err)
	}
	a.Value.State = "unknown"
	if ValidateAssertion(a) == nil {
		t.Fatal("unknown silently retained a numeric value")
	}
	a.Value = Value{State: "unknown"}
	if err := ValidateAssertion(a); err != nil {
		t.Fatal(err)
	}
	a.Value = Value{State: "known", Number: &zero, Unit: "L/min"}
	if ValidateAssertion(a) == nil {
		t.Fatal("dimension accepted a flow unit")
	}
}

func TestScheduleKeepsLocalExceptionsAndMeanings(t *testing.T) {
	s := Snapshot{Types: []Type{{ID: "paint", Code: "P1", Title: "Paint", Status: "active"}}, Applications: []Application{{ID: "wall", TypeID: "paint", Location: "Office", Surface: "wall", Selection: "selected"}, {ID: "ceiling", TypeID: "paint", Location: "Office", Surface: "ceiling", Selection: "selected"}}, Assertions: []Assertion{
		{TypeID: "paint", Key: "sheen", Meaning: "selection", Status: "accepted", Value: Value{State: "known", Text: "Low sheen"}},
		{TypeID: "paint", ApplicationID: "ceiling", Key: "sheen", Meaning: "selection", Status: "accepted", Value: Value{State: "known", Text: "Flat"}},
		{TypeID: "paint", ApplicationID: "ceiling", Key: "sheen", Meaning: "selection", Status: "proposed", Value: Value{State: "known", Text: "Gloss"}},
		{TypeID: "paint", Key: "performance", Meaning: "requirement", Status: "accepted", Value: Value{State: "known", Text: "Required test"}},
		{TypeID: "paint", Key: "performance", Meaning: "capability", Status: "accepted", Value: Value{State: "known", Text: "Supplier claim"}},
	}}
	rows := Assemble(s)
	for _, r := range rows {
		var sheen string
		for _, f := range r.Fields {
			if f.Key == "sheen" {
				sheen = f.Value
			}
		}
		if r.Surface == "wall" && sheen != "Low sheen" {
			t.Fatal(r)
		}
		if r.Surface == "ceiling" && (sheen != "Flat" || r.Pending != 1) {
			t.Fatal(r)
		}
		if len(r.Fields) != 3 {
			t.Fatal("requirement and capability collapsed", r)
		}
	}
	s.Types[0].Title = "=HYPERLINK(\"bad\")"
	csv, err := CSV(s)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(csv), "'=HYPERLINK") {
		t.Fatal("spreadsheet formula was not escaped")
	}
}

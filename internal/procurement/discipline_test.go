package procurement

import "testing"

func TestDisciplinePackage(t *testing.T) {
	cases := []struct {
		id, label, kind, scope string
		ok                     bool
	}{
		{"consultant.structural", "Structural", "services", "", true},
		{"trade.main_works", "Main Works", "works", "head_contract", true},
		{"trade.electrical", "Electrical Services", "works", "trade", true},
		{"supplier.facade", "Facade supply", "", "", false},
		{"other.thing", "Other", "", "", false},
	}
	for _, c := range cases {
		p, ok := DisciplinePackage(c.id, c.label)
		if ok != c.ok || p.Kind != c.kind || p.WorksScope != c.scope {
			t.Fatalf("%s: %+v %v", c.id, p, ok)
		}
		if ok && (p.DisciplineID != c.id || p.Title != c.label || p.LifecycleStatus != "planned") {
			t.Fatalf("%s: %+v", c.id, p)
		}
	}
}

func TestProposalDiscipline(t *testing.T) {
	ds := []Discipline{
		{ID: "consultant.services", Label: "Services", Aliases: []string{"Services Engineer"}},
		{ID: "consultant.town_planner", Label: "Town Planner"},
		{ID: "trade.main_works", Label: "Main Works"},
	}
	for key, want := range map[string]string{
		"default:services:Services Engineer": "consultant.services",
		"default:services:town planner":      "consultant.town_planner",
		WorksProposalKey:                     "trade.main_works",
		"default:services:Unknown":           "",
	} {
		if got := ProposalDiscipline(key, ds); got != want {
			t.Fatalf("%s: got %q want %q", key, got, want)
		}
	}
}

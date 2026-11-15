package httpapi

import (
	"strings"
	"testing"
)

func TestPrefetchFor(t *testing.T) {
	for _, c := range []struct{ text, want string }{
		// Questions: the areas their words name, earliest first.
		{"What's the total budget?", "cost_plan"},
		{"How much is left on mechanical and what share is spent?", "cost_plan"},
		{"When does the Planning stage finish?", "program"},
		{"What's the site area and the zone?", "profile"},
		{"Who has tendered for structural and what did they quote?", "procurement"},
		{"What needs my attention?", "attention"},
		{"What's the most risky aspect of this project?", "attention implications"},
		{"show me the invoices for mechanical", "invoices"},
		{"Which packages have no firms invited yet?", "procurement"},
		{"What happens to completion if the budget goes up?", "program cost_plan"},
		// At most two areas.
		{"What is the budget, the program finish and the zone?", "cost_plan program"},
		// A question that names no area is left to the model.
		{"Are trees involved in this project?", ""},
		// Commands are never read ahead, whatever they name.
		{"Add lines for joinery, stone and tapware to Fitout. Joinery is $45,000.", ""},
		{"Set the budget for mechanical to $120,000", ""},
		{"Increase the contingency by 10%", ""},
		{"", ""},
		// A word is matched whole: "classify" is not "class", "blot" is not "lot".
		{"Can you classify the blot?", ""},
	} {
		var got []string
		for _, p := range prefetchFor(c.text) {
			if p.Call.Name != "read" || !strings.Contains(string(p.Call.Input), `"area":"`+p.Area+`"`) {
				t.Fatalf("%q: a prefetch is a read of its area, got %+v", c.text, p)
			}
			got = append(got, p.Area)
		}
		if strings.Join(got, " ") != c.want {
			t.Errorf("%q: got %q, want %q", c.text, strings.Join(got, " "), c.want)
		}
	}
}

// Every area read ahead is one the read tool knows.
func TestPrefetchAreasAreReadAreas(t *testing.T) {
	known := map[string]bool{}
	for _, a := range readAreas {
		known[a] = true
	}
	for area := range prefetchWords {
		if !known[area] {
			t.Errorf("%q is not a read area", area)
		}
	}
}

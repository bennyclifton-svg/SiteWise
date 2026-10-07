package profile_test

import (
	"encoding/json"
	"sitewise/internal/jev"
	"sitewise/internal/profile"
	"strings"
	"testing"
)

func TestLocationOptionsAndCountSpecificThreshold(t *testing.T) {
	parts := []profile.Part{{ID: "whole", Label: "Whole project", Kind: "whole"}, {ID: "2", Label: "Plant room east", Kind: "part"}, {ID: "1", Label: "Plant room", Kind: "plant_area"}}
	q, options := profile.LocationOptions(parts)
	criteria := q.Criteria.(map[string]string)
	if len(criteria) != 6 || options["p1"].ID != "1" || !strings.Contains(criteria["p1"], "plant_area") || !strings.Contains(criteria["p2"], "Plant room east") {
		t.Fatalf("options %+v %+v", options, criteria)
	}
	th := profile.Thresholds{Amber: map[string]float64{"location.n5": .5, "presence": .5}}
	a := jev.Answer{Type: jev.TypeChoice, Choice: "p1", Confidence: conf(.99)}
	if _, part := th.AppliedLocation(a, parts); part != "" {
		t.Fatal("borrowed a threshold from another option count")
	}
	th.Amber["location.n6"] = .8
	if scope, part := th.AppliedLocation(a, parts); scope != "Plant room" || part != scope {
		t.Fatalf("placement %q %q", scope, part)
	}
	for _, choice := range []string{"specific", "multiple", "whole_project", "not_stated", "p999"} {
		a.Choice = choice
		if _, part := th.AppliedLocation(a, parts); part != "" {
			t.Fatalf("invented a part for %s", choice)
		}
	}
	base, _ := profile.LocationOptions(parts[:1])
	if len(base.Criteria.(map[string]string)) != 4 {
		t.Fatal("whole part duplicates generic option")
	}
	q2, _ := profile.LocationOptions([]profile.Part{parts[2], parts[0], parts[1]})
	b1, _ := json.Marshal(q)
	b2, _ := json.Marshal(q2)
	if string(b1) != string(b2) {
		t.Fatal("DB ordering changed location criteria")
	}
}

func TestLocatedFactsAreAmberAndRenamesFallBack(t *testing.T) {
	cat := repoCatalog(t)
	f := jevFact("sys.hydraulic.gas.presence", "included", "one", .99)
	f.PartLabel = "Building B"
	g := f
	g.DocumentID = "two"
	in := input(f, g)
	in.Thresholds.Green["presence"] = conf(.8)
	if r := row(t, profile.Reconcile(in, cat), "p-b", f.QuestionID); r.Band != "amber" {
		t.Fatalf("located evidence turned green %+v", r)
	}
	in.Parts[1].Label = "Renamed building"
	rows := profile.Reconcile(in, cat)
	noRow(t, rows, "p-b", f.QuestionID)
	if r := row(t, rows, whole, f.QuestionID); r.Band != "amber" {
		t.Fatal("stale location gained certainty")
	}
}

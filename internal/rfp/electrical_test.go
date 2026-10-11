package rfp

import (
	"strings"
	"testing"
)

func TestElectricalPricingDefaultsAndLegacyCounts(t *testing.T) {
	d, sk := discipline(t, "electrical-services"), skeleton(t)
	drivers := facts(t, "class=5", "work_type=new", "procurement=traditional", "pathway=da")
	settings := set("boundary:range", `{"start":"inception","end":"tender"}`)
	r := Resolve(d, sk, drivers, settings)
	values := map[string]string{}
	for _, a := range r.Allowances {
		values[a.Key] = a.Value
	}
	if values["design_meetings"] != "4" || values["design_review_rounds"] != "1" {
		t.Fatalf("pricing defaults: %+v", values)
	}
	for _, c := range r.Checks {
		if c.Blocking && strings.HasPrefix(c.ID, "allowance:") {
			t.Fatalf("design blocked by count: %+v", c)
		}
	}
	for _, p := range r.Positions {
		if (p.Key == "generation" || p.Key == "ict") && p.Value != "O" {
			t.Fatalf("unknown specialist service: %+v", p)
		}
	}
	settings["allowance:design_meetings"] = set("x", `"6"`)["x"]
	r = Resolve(d, sk, drivers, settings)
	for _, a := range r.Allowances {
		if a.Key == "design_meetings" && (a.Value != "6" || a.Source != "user") {
			t.Fatalf("override lost: %+v", a)
		}
	}
	settings["allowance:extent"] = set("x", `"12"`)["x"]
	r = Resolve(d, sk, drivers, settings)
	found := false
	for _, c := range r.Checks {
		if c.ID == "orphan:allowance:extent" && c.Blocking && strings.Contains(c.Text, "12") {
			found = true
		}
	}
	if !found {
		t.Fatal("saved compound count silently discarded")
	}
	drivers.Facts["flag.elec_generation"] = Fact{State: Conflict, Values: []string{"yes", "no"}}
	r = Resolve(d, sk, drivers, settings)
	for _, p := range r.Positions {
		if p.Key == "generation" && p.Value != Ask {
			t.Fatalf("conflict treated as optional: %+v", p)
		}
	}
	settings["boundary:range"] = set("x", `{"start":"inception","end":"construction"}`)["x"]
	settings["stage:construction"] = set("x", `"include"`)["x"]
	r = Resolve(d, sk, drivers, settings)
	found = false
	for _, c := range r.Checks {
		if c.ID == "allowance:construction_visits" && c.Blocking {
			found = true
		}
	}
	if !found {
		t.Fatal("base construction appointment must specify visits")
	}
}

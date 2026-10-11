package rfp

import (
	"strings"
	"testing"
)

func TestHydraulicPilotSelectionsAndLegacyChoices(t *testing.T) {
	d, sk := discipline(t, "hydraulic-services"), skeleton(t)
	drivers := facts(t, "class=2", "work_type=new", "procurement=traditional", "pathway=da")
	r := Resolve(d, sk, drivers, nil)
	if r.HasCheck("boundary") || r.Stage("design").State != Include || r.Stage("tender").State != Option {
		t.Fatalf("design-first appointment: %+v", r.Boundary)
	}
	for _, key := range []string{"rainwater", "pumps", "roof", "interfaces"} {
		if p := r.Position(key); p.Value != "I" || p.Assumption == "" {
			t.Fatalf("%s must carry its pricing basis: %+v", key, p)
		}
	}
	if r.Position("alternative_sources").Value != "O" {
		t.Fatal("unusual sources should be separately priced until specified")
	}
	settings := set("stage:design", `"option"`)
	r = Resolve(d, sk, drivers, settings)
	if r.Position("rainwater").Value != "O" || r.Position("domestic").Value != "O" {
		t.Fatal("optional design leaked into the base fee")
	}
	unknown := set("position:rainwater", `"ask"`)
	unknown["stage:planning"] = set("x", `"ask"`)["x"]
	for key, value := range unknown {
		if err := ValidateSetting(d, sk, drivers, key, value.Value); err != nil {
			t.Fatal(err)
		}
	}
	u := Resolve(d, sk, drivers, unknown)
	if u.Position("rainwater").Value != Ask || !u.HasCheck("scope:rainwater") || !u.HasCheck("scope-stage:planning") {
		t.Fatal("explicit unknown must persist and block issue")
	}
	legacy := Resolve(d, sk, drivers, set("position:domestic", `"X"`))
	if legacy.Position("sanitary").Value != "X" {
		t.Fatal("old combined water/sanitary selection lost")
	}
	settings["position:rainwater"] = set("x", `"X"`)["x"]
	r = Resolve(d, sk, drivers, settings)
	if r.Position("rainwater").Value != "X" || !r.HasCheck("unallocated:rainwater") {
		t.Fatal("explicit exclusion lost, or missing responsibility hidden")
	}
	drivers.Facts["flag.hyd_rainwater"] = Fact{State: Conflict, Values: []string{"yes", "no"}}
	r = Resolve(d, sk, drivers, nil)
	if r.Position("rainwater").Value != Ask || !r.HasCheck("fact:flag.hyd_rainwater") {
		t.Fatal("conflicting sources became a pricing assumption")
	}
	settings["position:sources"] = set("x", `"I"`)["x"]
	r = Resolve(d, sk, drivers, settings)
	found := false
	for _, c := range r.Blocking() {
		found = found || c.ID == "orphan:position:sources"
	}
	if !found {
		t.Fatal("old combined source selection silently reinterpreted")
	}
}

func TestHydraulicPilotCombinedScheduleAndDates(t *testing.T) {
	d, sk := discipline(t, "hydraulic-services"), skeleton(t)
	drivers := facts(t, "class=2", "work_type=new", "procurement=traditional", "pathway=da")
	settings := set("field:issued", `"2026-10-12"`)
	settings["field:closes"] = set("x", `"2026-10-30"`)["x"]
	settings["weight:price"] = set("x", `99`)["x"]
	r := Resolve(d, sk, drivers, settings)
	if r.HasCheck("fields") || r.HasCheck("weights") {
		t.Fatal("removed issue fields or evaluation still block issue")
	}
	sections, err := Sections(r, consultantTemplate(t))
	if err != nil {
		t.Fatal(err)
	}
	if sections[0].ID != "issue_dates" || sections[2].ID != "positions" {
		t.Fatal("dates and scope are not first")
	}
	for _, section := range sections {
		if section.ID == "role" || section.ID == "evaluation" {
			t.Fatalf("redundant section: %s", section.ID)
		}
		for _, block := range section.Blocks {
			if block.ID == "rfp:returnables:r1" {
				t.Fatal("duplicate pricing schedule")
			}
			if block.ID == "rfp:stages" && (block.Table == nil || len(block.Table.Columns) != 5) {
				t.Fatal("stage summary and pricing not combined")
			}
		}
	}
	text := Markdown(sections)
	for _, forbidden := range []string{"Your role", "This appointment buys", "Contact ", "Questions to", "Confirm you can start by", "section 5", "Section 5 allocation"} {
		if strings.Contains(text, forbidden) {
			t.Fatalf("obsolete text remains: %s", forbidden)
		}
	}
	controls := BuildControls(r, d, sk, settings)
	for _, g := range controls.Groups {
		if g.ID == "weights" {
			t.Fatal("evaluation controls remain")
		}
		if g.ID == "issue" && (len(g.Controls) != 2 || g.Section != "issue_dates") {
			t.Fatal("issue details are not just top dates")
		}
	}
	settings["field:closes"] = set("x", `"2026-10-01"`)["x"]
	if !Resolve(d, sk, drivers, settings).HasCheck("proposal_dates") {
		t.Fatal("closing before release must be corrected")
	}
}

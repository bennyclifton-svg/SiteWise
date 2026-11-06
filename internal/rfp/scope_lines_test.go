package rfp

import (
	"encoding/json"
	"strings"
	"testing"
)

// Scope by stage (owner direction, 3 November 2026): every line starts
// included, a line can be excluded on its own, the user can add lines of
// their own, and a stage is only in or out.
func TestScopeLinesIncludeExcludeAndOwnLines(t *testing.T) {
	d, sk := discipline(t, "architecture"), skeleton(t)
	drivers := facts(t, "class=1a", "work_type=new", "procurement=traditional", "pathway=cdc")
	own, _ := json.Marshal([]CustomLine{{ID: "u1", Text: "Survey coordination", Value: "I"}, {ID: "u2", Text: "Pool fencing details", Value: "X"}})
	settings := set("line:concept.area_schedule", `"X"`)
	settings["custom:concept"] = Setting{Value: own}
	r := Resolve(d, sk, drivers, settings)

	concept := r.Stage("concept")
	if concept.State != Include || len(concept.Lines) != 8 {
		t.Fatalf("concept = %s with %d lines", concept.State, len(concept.Lines))
	}
	values := map[string]string{}
	for _, l := range concept.Lines {
		values[l.Key] = l.Value
	}
	if values["site_inspection"] != "I" || values["area_schedule"] != "X" || values["u1"] != "I" || values["u2"] != "X" {
		t.Fatalf("lines = %v", values)
	}
	// The stage's deliverable is what it buys: included lines only.
	if strings.Contains(concept.Text, "area schedule") || strings.Contains(concept.Text, "Pool fencing") || !strings.Contains(concept.Text, "Survey coordination") {
		t.Fatalf("concept text = %q", concept.Text)
	}
	for _, s := range r.Stages {
		if s.State == Option || s.State == Ask || s.State == Done {
			t.Errorf("stage %s is %s; scope by stage is only in or out", s.Key, s.State)
		}
	}
	for _, p := range r.Positions {
		if p.Value != "I" && p.Value != "X" {
			t.Errorf("position %s is %s", p.Key, p.Value)
		}
		if p.Value == "X" && p.Party != "" {
			t.Errorf("excluded %s names a party: %s", p.Key, p.Party)
		}
	}

	sections, err := Sections(r, consultantTemplate(t))
	if err != nil {
		t.Fatal(err)
	}
	text := Markdown(sections)
	for _, want := range []string{"| Concept |  | Included |  |", "|  | Area schedule | Excluded |  |", "|  | Survey coordination | Included |  |", "| Total fee |"} {
		if !strings.Contains(text, want) {
			t.Errorf("printed RFP lacks %q", want)
		}
	}
	if strings.Contains(text, "Options total") {
		t.Error("scope by stage prints no options total")
	}
}

// A stage after an appointment range the user set stays out, so a
// planning-only appointment is not priced for documentation.
func TestScopeByStageKeepsUserRange(t *testing.T) {
	d, sk := discipline(t, "architecture"), skeleton(t)
	drivers := facts(t, "class=1a", "work_type=new", "procurement=traditional", "pathway=cdc")
	r := Resolve(d, sk, drivers, set("boundary:range", `{"start":"concept","end":"planning_approval"}`))
	if s := r.Stage("design_development"); s.State != Exclude {
		t.Fatalf("design development after the range = %s", s.State)
	}
	if s := r.Stage("planning_approval"); s.State != Include {
		t.Fatalf("planning approval in the range = %s", s.State)
	}
}

func TestCustomLinesAreValidated(t *testing.T) {
	d, sk := discipline(t, "architecture"), skeleton(t)
	drivers := facts(t, "class=1a")
	ok := `[{"id":"u1","text":"Survey coordination","value":"I"}]`
	if err := ValidateSetting(d, sk, drivers, "custom:concept", json.RawMessage(ok)); err != nil {
		t.Fatalf("valid lines refused: %v", err)
	}
	if err := ValidateSetting(d, sk, drivers, "line:concept.area_schedule", json.RawMessage(`"X"`)); err != nil {
		t.Fatalf("line choice refused: %v", err)
	}
	for name, raw := range map[string]string{
		"slot":         `[{"id":"u1","text":"Up to {allowance:x}","value":"I"}]`,
		"blank":        `[{"id":"u1","text":"  ","value":"I"}]`,
		"duplicate id": `[{"id":"u1","text":"A","value":"I"},{"id":"u1","text":"B","value":"I"}]`,
		"bad id":       `[{"id":"concept","text":"A","value":"I"}]`,
		"option":       `[{"id":"u1","text":"A","value":"O"}]`,
		"long":         `[{"id":"u1","text":"` + strings.Repeat("a", 301) + `","value":"I"}]`,
	} {
		if ValidateSetting(d, sk, drivers, "custom:concept", json.RawMessage(raw)) == nil {
			t.Errorf("%s accepted", name)
		}
	}
	for _, key := range []string{"custom:nostage", "line:concept.nope", "line:nostage.site_inspection"} {
		if ValidateSetting(d, sk, drivers, key, json.RawMessage(`"I"`)) == nil {
			t.Errorf("%s accepted", key)
		}
	}
	if ValidateSetting(d, sk, drivers, "line:concept.area_schedule", json.RawMessage(`"O"`)) == nil {
		t.Error("a line accepted an option")
	}
	// Disciplines not yet by stage take no line settings.
	h := ruled(t, "hydraulic-services")
	if ValidateSetting(h, sk, drivers, "custom:"+h.Stages[0].Key, json.RawMessage(ok)) == nil {
		t.Error("custom lines accepted for a discipline not by stage")
	}
}

// R13 (owner ruling, 7 November 2026): the brief is typical on every
// project, so the architect always prices it; a masterplan is only for a
// large project (more than one building, or staged). The stage name follows
// what is included.
func TestArchitectBriefAlwaysMasterplanOnlyWhenLarge(t *testing.T) {
	d, sk := discipline(t, "architecture"), skeleton(t)
	lineKeys := func(s StageResult) string {
		keys := []string{}
		for _, l := range s.Lines {
			keys = append(keys, l.Key)
		}
		return strings.Join(keys, ",")
	}
	for _, c := range []struct {
		name  string
		facts []string
	}{
		{"single house", []string{"class=1a", "work_type=new", "procurement=traditional", "pathway=cdc"}},
		{"apartment building", []string{"class=2", "work_type=new", "procurement=traditional", "pathway=da", "flag.multi_building=no", "flag.staged=no"}},
	} {
		s := Resolve(d, sk, facts(t, c.facts...), Settings{}).Stage("brief_masterplan")
		if s.State != Include || s.Name != "Brief" || lineKeys(s) != "brief,strategy_workshop" {
			t.Errorf("%s: %s %q lines %s", c.name, s.State, s.Name, lineKeys(s))
		}
		if strings.Contains(s.Text, "masterplan") {
			t.Errorf("%s prices a masterplan: %q", c.name, s.Text)
		}
	}
	for _, flag := range []string{"flag.multi_building=yes", "flag.staged=yes"} {
		r := Resolve(d, sk, facts(t, "class=9b", "work_type=new", "procurement=traditional", "pathway=da", flag), Settings{})
		s := r.Stage("brief_masterplan")
		if s.State != Include || s.Name != "Brief and masterplan" || lineKeys(s) != "brief,masterplan_options,masterplan_report,strategy_workshop" {
			t.Errorf("%s: %s %q lines %s", flag, s.State, s.Name, lineKeys(s))
		}
	}
	// Excluding both masterplan lines on a large project names the stage Brief.
	r := Resolve(d, sk, facts(t, "class=9b", "work_type=new", "flag.staged=yes"), set("line:brief_masterplan.masterplan_options", `"X"`, "line:brief_masterplan.masterplan_report", `"X"`))
	if s := r.Stage("brief_masterplan"); s.Name != "Brief" {
		t.Errorf("masterplan excluded: %q", s.Name)
	}
}

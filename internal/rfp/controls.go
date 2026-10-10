package rfp

import (
	"strings"

	"sitewise/internal/knowledge"
)

// Controls is the editor's view of a resolved RFP: the decisions that block
// issue first, then grouped controls, each with its value and source.
type Controls struct {
	Discipline string    `json:"discipline"`
	Title      string    `json:"title"`
	Status     string    `json:"status"`
	Scale      Scale     `json:"scale"`
	Boundary   BoundaryC `json:"boundary"`
	Decisions  []Check   `json:"decisions"`
	Checks     []Check   `json:"checks"`
	Groups     []Group   `json:"groups"`
}

type Scale struct {
	Value  string `json:"value"`
	Source string `json:"source"`
	Why    string `json:"why"`
	Set    bool   `json:"set"`
}

type BoundaryC struct {
	Boundary
	Stages []Choice `json:"stages"`
	Set    bool     `json:"set"`
}

type Group struct {
	ID       string    `json:"id"`
	Title    string    `json:"title"`
	Controls []Control `json:"controls"`
}

type Choice struct {
	Value string `json:"value"`
	Label string `json:"label"`
}

// Control kinds: choice, tristate, parts, number, text, date, stage,
// element, count, position, weight.
type Control struct {
	Key     string   `json:"key"`
	Label   string   `json:"label"`
	Kind    string   `json:"kind"`
	Value   any      `json:"value"`
	Options []Choice `json:"options,omitempty"`
	Source  string   `json:"source"`
	Why     string   `json:"why,omitempty"`
	Help    string   `json:"help,omitempty"`
	Locked  bool     `json:"locked,omitempty"`
	Set     bool     `json:"set"`
	Party   string   `json:"party,omitempty"`
	Owner   string   `json:"owner,omitempty"`
	// Home is the kind of Profile home a fact: control writes to (detail,
	// header, site, determinant), or where it is changed instead
	// (cost_plan, link, derived; then Locked).
	Home string `json:"home,omitempty"`
}

// BuildControls is pure: it reads the resolved RFP, the rules and the
// stored settings, and never the project.
func BuildControls(r Resolved, d knowledge.RFPDiscipline, sk knowledge.RFPSkeleton, settings Settings) Controls {
	isSet := func(key string) bool { _, ok := settings[key]; return ok }
	c := Controls{Discipline: r.Discipline, Title: r.Title, Status: r.Status, Decisions: r.Blocking(), Checks: r.Checks}
	c.Scale = Scale{Value: r.Scale, Source: r.ScaleSource, Why: r.ScaleWhy, Set: isSet("scale")}
	c.Boundary = BoundaryC{Boundary: r.Boundary, Set: isSet("boundary:range")}
	for _, s := range d.Stages {
		c.Boundary.Stages = append(c.Boundary.Stages, Choice{Value: s.Key, Label: s.Name})
	}
	used := usedDrivers(d, sk)
	project := Group{ID: "project", Title: "Project"}
	options := func(list []knowledge.RFPOption) []Choice {
		out := []Choice{}
		for _, o := range list {
			out = append(out, Choice{Value: o.Key, Label: o.Label})
		}
		return out
	}
	// A project fact is read from Profile and an edit writes it there (D1),
	// so it is never "set" on the RFP.
	fact := func(key, label, kind string, opts []Choice, help string) {
		f := r.Fact(key)
		var v any = nil
		if f.State == Known && len(f.Values) > 0 {
			v = f.Values[0]
		}
		why := ""
		if f.State == Conflict {
			why = "Sources disagree: " + strings.Join(f.Values, " vs ")
		}
		home := FactHome(sk, r, key)
		switch home.Kind {
		case HomeCostPlan:
			help = "Set in the cost plan."
		case HomeLink:
			help = "Change it in Profile."
		case HomeDerived:
			help = "Follows from the parts' classes in Profile."
		}
		project.Controls = append(project.Controls, Control{Key: "fact:" + key, Label: label, Kind: kind, Value: v, Options: opts, Source: sourceOf(f), Why: why, Help: help, Home: home.Kind, Locked: !home.Writable()})
	}
	fact("work_type", "Work type", "choice", options(sk.WorkTypes), "")
	fact("pathway", "Planning pathway", "choice", options(sk.Pathways), "")
	fact("procurement", "Procurement route", "choice", options(sk.Procurement), "")
	if used["regulated_design"] {
		fact("regulated_design", "Regulated design applies", "choice", []Choice{{"yes", "Yes, the regime applies"}, {"no", "No, recorded advice says it does not"}}, "Record the professional advice or verified rule you relied on. The class alone does not decide it.")
	}
	parts := []Choice{}
	selected := []string{}
	for _, p := range r.AllParts {
		label := p.Label
		if len(p.Classes) > 0 {
			label += " (Class " + strings.Join(p.Classes, ", ") + ")"
		} else {
			label += " (class not established; change it in Profile)"
		}
		parts = append(parts, Choice{Value: p.ID, Label: label})
	}
	for _, p := range r.Parts {
		selected = append(selected, p.ID)
	}
	project.Controls = append(project.Controls, Control{Key: "parts:affected", Label: "Parts this appointment covers", Kind: "parts", Value: selected, Options: parts, Source: map[bool]string{true: "user", false: "profile"}[isSet("parts:affected")], Set: isSet("parts:affected")})
	for _, key := range []string{"site_area", "gfa", "budget"} {
		if used[key] || key != "budget" {
			fact(key, factLabels[key], "number", nil, "")
		}
	}
	for _, f := range sk.Flags {
		if !used["flag."+f.Key] {
			continue
		}
		fact("flag."+f.Key, f.Label, "tristate", []Choice{{"yes", "Yes"}, {"no", "No"}, {"unknown", "Not known"}}, "")
	}
	for _, f := range sk.Fields {
		if f.Group != "project" {
			continue
		}
		v := r.Fields[f.Key]
		if f.Profile != "" {
			home := FactHome(sk, r, f.Key)
			project.Controls = append(project.Controls, Control{Key: "fact:" + f.Key, Label: f.Label, Kind: f.Kind, Value: v.Value, Source: v.Source, Help: f.Help, Home: home.Kind})
			continue
		}
		project.Controls = append(project.Controls, Control{Key: "field:" + f.Key, Label: f.Label, Kind: f.Kind, Value: v.Value, Source: v.Source, Help: f.Help, Set: isSet("field:" + f.Key)})
	}
	c.Groups = append(c.Groups, project)

	stages := Group{ID: "stages", Title: "Stages"}
	stateOptions := []Choice{{Include, "In the base fee"}, {Option, "Priced as an option"}, {Exclude, "Not in this RFP"}, {Done, "Done by others"}}
	for _, s := range r.Stages {
		stages.Controls = append(stages.Controls, Control{Key: "stage:" + s.Key, Label: s.Name, Kind: "stage", Value: s.State, Options: stateOptions, Source: s.Source, Why: s.Why, Locked: s.Locked, Set: isSet("stage:" + s.Key), Help: s.ShownWhen})
	}
	c.Groups = append(c.Groups, stages)
	if len(r.Elements) > 0 {
		elements := Group{ID: "elements", Title: "Priced elements"}
		for _, e := range r.Elements {
			elements.Controls = append(elements.Controls, Control{Key: "element:" + e.Key, Label: e.Code + " " + e.Name, Kind: "element", Value: e.State, Options: stateOptions[:3], Source: e.Source, Why: e.Why, Set: isSet("element:" + e.Key)})
		}
		c.Groups = append(c.Groups, elements)
	}
	allowances := Group{ID: "allowances", Title: "Allowances"}
	for _, a := range r.Allowances {
		allowances.Controls = append(allowances.Controls, Control{Key: "allowance:" + a.Key, Label: a.Label, Kind: "count", Value: a.Value, Source: a.Source, Owner: a.Owner, Set: isSet("allowance:" + a.Key), Help: a.Text})
	}
	c.Groups = append(c.Groups, allowances)
	positions := Group{ID: "positions", Title: "Scope position"}
	for _, p := range r.Positions {
		why := p.Why
		if p.Assumption != "" {
			why = p.Assumption
		}
		positions.Controls = append(positions.Controls, Control{Key: "position:" + p.Key, Label: p.Label, Kind: "position", Value: p.Value, Options: []Choice{{"I", "Included"}, {"O", "Optional"}, {"X", "Excluded"}}, Source: p.Source, Why: why, Party: p.Party, Set: isSet("position:" + p.Key)})
	}
	c.Groups = append(c.Groups, positions)
	issue := Group{ID: "issue", Title: "Issue details"}
	for _, f := range sk.Fields {
		if f.Group != "issue" {
			continue
		}
		v := r.Fields[f.Key]
		issue.Controls = append(issue.Controls, Control{Key: "field:" + f.Key, Label: f.Label, Kind: f.Kind, Value: v.Value, Source: v.Source, Help: f.Help, Set: isSet("field:" + f.Key)})
	}
	for _, w := range r.Weights {
		issue.Controls = append(issue.Controls, Control{Key: "weight:" + w.Key, Label: w.Label + " (%)", Kind: "weight", Value: w.Value, Source: w.Source, Set: isSet("weight:" + w.Key)})
	}
	c.Groups = append(c.Groups, issue)
	return c
}

func sourceOf(f Fact) string {
	switch {
	case f.Source == "user":
		return "user"
	case f.State == Conflict:
		return "conflict"
	case f.State != Known:
		return "unknown"
	}
	return f.Source
}

// usedDrivers lists the facts a discipline's rules test, so the editor shows
// only the flags and facts that can change this RFP.
func usedDrivers(d knowledge.RFPDiscipline, sk knowledge.RFPSkeleton) map[string]bool {
	used := map[string]bool{}
	visit := func(w knowledge.RFPWhen) {
		for _, clause := range w {
			for key, values := range clause {
				if key == "flag" {
					for _, v := range values {
						used["flag."+v] = true
					}
					continue
				}
				if fact, ok := overFacts[key]; ok {
					used[fact] = true
					continue
				}
				used[key] = true
			}
		}
	}
	modules := map[string]bool{}
	cell := func(c *knowledge.RFPCell) {
		if c != nil {
			visit(c.When)
		}
	}
	visit(d.Scale.Complex)
	for _, b := range d.Boundary {
		visit(b.When)
	}
	for _, s := range d.Stages {
		visit(s.When)
		visit(s.Suggest)
		visit(s.IncludeWhen)
		modules[s.Module] = true
	}
	for _, e := range d.Elements {
		visit(e.When)
	}
	for _, a := range d.Allowances {
		visit(a.When)
		modules[a.Module] = true
	}
	for _, p := range d.Positions {
		visit(p.When)
		visit(p.Suggest)
		cell(p.Value)
		cell(p.Simple)
		cell(p.Complex)
		modules[p.Module] = true
	}
	for _, rows := range [][]knowledge.RFPRow{d.ProjectRows, d.Role, d.Terms, d.ReturnableNotes} {
		for _, row := range rows {
			visit(row.When)
			modules[row.Module] = true
		}
	}
	for _, m := range sk.Modules {
		if modules[m.Key] {
			visit(m.When)
		}
	}
	return used
}

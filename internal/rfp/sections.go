package rfp

import (
	"encoding/json"
	"fmt"
	"strings"

	"sitewise/internal/knowledge"
	"sitewise/internal/reports"
)

// BudgetBlockID is the one RFP block that carries the internal budget. Issue
// strips it unless the user discloses the budget.
const BudgetBlockID = "rfp:budget"

var feeBasisLabel = map[string]string{"fixed": "Lump sum", "time_to_ceiling": "Time to the ceiling", "per_unit": "Rate per unit", "option": "Option"}

type builder struct {
	r       Resolved
	base    map[string]any
	draft   bool
	section map[string]*reports.Section
	order   []string
}

// Sections renders a resolved RFP into the template's sections, followed by
// the internal Before-issue section. Every block of a draft discipline is
// provisional, which keeps it from being issued (report_issue.go).
func Sections(r Resolved, template knowledge.ReportTemplate) ([]reports.Section, error) {
	b := &builder{r: r, draft: r.Status != "reviewed", section: map[string]*reports.Section{}}
	b.base = map[string]any{"method": "rfp_discipline", "discipline": r.Discipline, "version": r.Version, "status": r.Status}
	for _, s := range template.Sections {
		sec := reports.Section{ID: s.ID, Title: s.Title, Essential: s.Essential, Blocks: []reports.Block{}}
		b.section[s.ID] = &sec
		b.order = append(b.order, s.ID)
	}
	for _, id := range []string{"project", "role", "stages", "allowances", "positions", "terms", "evaluation", "returnables"} {
		if b.section[id] == nil {
			return nil, fmt.Errorf("consultant RFP template missing section %s", id)
		}
	}
	b.project()
	b.role()
	b.stages()
	b.allowances()
	b.positions()
	b.terms()
	b.evaluation()
	b.returnables()
	out := []reports.Section{}
	for _, id := range b.order {
		out = append(out, *b.section[id])
	}
	return append(out, b.review()), nil
}

func (b *builder) add(section, id, label, text string, table *reports.Table, extra map[string]any) {
	basis := map[string]any{}
	for k, v := range b.base {
		basis[k] = v
	}
	for k, v := range extra {
		basis[k] = v
	}
	raw, _ := json.Marshal(basis)
	block := reports.Block{ID: id, Label: label, Text: text, Origin: "calculation", ReviewStatus: "proposed", Meaning: "requirement", Basis: raw, Provisional: b.draft, Table: table}
	b.section[section].Blocks = append(b.section[section].Blocks, block)
}

func doc(columns []string, rows [][]string) *reports.Table {
	return &reports.Table{Columns: columns, Rows: rows, Display: "document"}
}

func (b *builder) field(key string) string {
	if v := b.r.Fields[key].Value; v != "" {
		return v
	}
	return "___"
}

func (b *builder) project() {
	header := strings.Join([]string{b.r.Heading, "Ref " + b.field("ref") + " · Issued " + b.field("issued") + " · Closes " + b.field("closes") + " · Contact " + b.field("contact")}, "\n")
	b.add("project", "rfp:header", "", header, nil, map[string]any{"fields": b.r.Fields})
	rows := [][]string{}
	for _, row := range b.r.Project {
		if row.Label == "" {
			b.add("project", "rfp:project:intro", "", row.Text, nil, map[string]any{"facts": b.r.Facts, "parts": b.r.Parts})
			continue
		}
		rows = append(rows, []string{row.Label, row.Text})
	}
	b.add("project", "rfp:project", "", "", doc([]string{"Project", "Detail"}, rows), map[string]any{"rows": b.r.Project, "facts": b.r.Facts})
	if b.r.Budget != "" {
		b.add("project", BudgetBlockID, "Construction budget", "$"+formatNumber(b.r.Budget)+" "+b.budgetBasis(), nil, map[string]any{"budget": b.r.Budget})
	}
}

func (b *builder) budgetBasis() string {
	if f, ok := b.r.Facts["budget"]; ok && f.Detail != "" {
		return f.Detail
	}
	return "excl. GST"
}

func (b *builder) role() {
	lines := []string{}
	for _, row := range b.r.Role {
		lines = append(lines, row.Text)
	}
	b.add("role", "rfp:role", "", strings.Join(lines, "\n"), nil, map[string]any{"rows": b.r.Role})
}

func (b *builder) stages() {
	main, options := [][]string{}, [][]string{}
	for _, s := range b.r.Stages {
		text := s.Text
		if s.Responsibility != "" {
			text += ". " + s.Responsibility
		}
		switch s.State {
		case Include:
			if s.Assumption != "" {
				text += " Assumption: " + s.Assumption
			}
			main = append(main, []string{s.Name, text, feeBasisLabel[s.FeeBasis]})
		case Ask:
			main = append(main, []string{s.Name, "To confirm before issue: " + text, feeBasisLabel[s.FeeBasis]})
		case Option:
			why := s.Assumption
			if after, ok := strings.CutPrefix(s.Why, "Follows "); ok && why == "" {
				why = "With the " + strings.ToLower(after)
			}
			if why == "" {
				why = map[string]string{"boundary": "Later stage; the client may commission it", "default": "Client opts in"}[s.Source]
			}
			if why == "" {
				why = "Client opts in"
			}
			options = append(options, []string{s.Name, text, why})
		}
	}
	b.add("stages", "rfp:stages", "", b.r.StageIntro, doc([]string{"Stage", "Deliverable at sign-off", "Pricing"}, main), map[string]any{"stages": b.r.Stages, "boundary": b.r.Boundary})
	if len(options) > 0 {
		b.add("stages", "rfp:stages:options", "", "Price these separately; they are not in the base fee.", doc([]string{"Option", "Deliverable", "Why it is optional"}, options), map[string]any{"stages": b.r.Stages})
	}
	if len(b.r.Elements) > 0 {
		rows := [][]string{}
		for _, e := range b.r.Elements {
			switch e.State {
			case Include:
				rows = append(rows, []string{e.Code, e.Name, e.Text})
			case Option:
				rows = append(rows, []string{e.Code, e.Name + " (option)", e.Text + " " + e.Assumption})
			case Ask:
				rows = append(rows, []string{e.Code, e.Name + " (to confirm)", e.Text})
			}
		}
		b.add("stages", "rfp:elements", "", b.r.ElementNote, doc([]string{"Element", "Name", "Includes"}, rows), map[string]any{"elements": b.r.Elements})
	}
}

func (b *builder) allowances() {
	rows := [][]string{}
	for i := 0; i < len(b.r.Allowances); i += 2 {
		row := []string{b.r.Allowances[i].Label, b.r.Allowances[i].Text, "", ""}
		if i+1 < len(b.r.Allowances) {
			row[2], row[3] = b.r.Allowances[i+1].Label, b.r.Allowances[i+1].Text
		}
		rows = append(rows, row)
	}
	b.add("allowances", "rfp:allowances", "", b.r.AllowNote, doc([]string{"Item", "Allowance", "Item", "Allowance"}, rows), map[string]any{"allowances": b.r.Allowances})
}

func (b *builder) positions() {
	rows := [][]string{}
	for _, p := range b.r.Positions {
		v := p.Value
		switch p.Value {
		case "X":
			v = "X: " + p.Party
		case Ask:
			v = "To confirm before issue"
		}
		if p.Assumption != "" {
			v += ". " + p.Assumption
		}
		rows = append(rows, []string{p.Label, v})
	}
	scale := "Service levels in this table are set for a " + b.r.Scale + " project"
	if b.r.ScaleSource == "rule" && b.r.Scale == "complex" {
		scale += " (" + b.r.ScaleWhy + ")"
	}
	text := strings.Join([]string{b.r.Legend, scale + ". Technical items follow the project's own facts.", b.r.Departures}, "\n")
	b.add("positions", "rfp:positions", "", text, doc([]string{"Item", "Position"}, rows), map[string]any{"positions": b.r.Positions, "scale": b.r.Scale, "scale_source": b.r.ScaleSource})
}

func (b *builder) terms() {
	lines := []string{}
	for _, t := range b.r.Terms {
		lines = append(lines, "- "+t.Text)
	}
	b.add("terms", "rfp:terms", "", strings.Join(lines, "\n"), nil, map[string]any{"rows": b.r.Terms})
}

func (b *builder) evaluation() {
	parts := []string{}
	for _, w := range b.r.Weights {
		parts = append(parts, fmt.Sprintf("%s %d%%", w.Label, w.Value))
	}
	b.add("evaluation", "rfp:evaluation", "", strings.Join(parts, " · ")+". "+b.r.EvalNote, nil, map[string]any{"weights": b.r.Weights})
}

func (b *builder) returnables() {
	lines := []string{}
	for _, r := range b.r.Returnables {
		if r.Key == "r1" {
			continue
		}
		lines = append(lines, r.Label+": "+r.Text)
	}
	var r1 Row
	for _, r := range b.r.Returnables {
		if r.Key == "r1" {
			r1 = r
		}
	}
	b.add("returnables", "rfp:returnables:r1", "", r1.Label+": "+r1.Text, b.feeTable(), map[string]any{"stages": b.r.Stages, "elements": b.r.Elements})
	b.add("returnables", "rfp:returnables", "", strings.Join(lines, "\n"), nil, map[string]any{"rows": b.r.Returnables})
}

// feeTable is the R1 price schedule: base stages, then options, each with a
// total, so an option is never rolled into the base fee.
func (b *builder) feeTable() *reports.Table {
	elements := []ElementResult{}
	for _, e := range b.r.Elements {
		if e.State == Include || e.State == Option || e.State == Ask {
			elements = append(elements, e)
		}
	}
	grid := false
	for _, s := range b.r.Stages {
		grid = grid || (s.ByElement && len(elements) > 0)
	}
	columns := []string{"Stage", "Fee $ excl. GST", "Weeks", "Hours"}
	blank := func(label string) []string { return []string{label, "", "", ""} }
	if grid {
		columns = []string{"Stage"}
		for _, e := range elements {
			columns = append(columns, e.Code)
		}
		columns = append(columns, "Total $", "Weeks")
		blank = func(label string) []string { return append([]string{label}, make([]string, len(elements)+2)...) }
	}
	stageRow := func(s StageResult, label string) []string {
		row := blank(label)
		if grid && !s.ByElement {
			for i := range elements {
				row[i+1] = "—"
			}
		}
		return row
	}
	rows := [][]string{}
	for _, s := range b.r.Stages {
		if s.State != Include && s.State != Ask {
			continue
		}
		label := s.Name
		if s.FeeBasis == "time_to_ceiling" {
			label += " (ceiling: hours × R3 rate)"
		}
		if s.FeeBasis == "per_unit" {
			label += " (rate per unit × quantity)"
		}
		rows = append(rows, stageRow(s, label))
	}
	rows = append(rows, blank("Base fee total"))
	optionRows := [][]string{}
	for _, s := range b.r.Stages {
		if s.State == Option {
			optionRows = append(optionRows, stageRow(s, "Option: "+s.Name))
		}
	}
	for _, p := range b.r.Positions {
		if p.Value == "O" {
			row := blank("Option: " + shortLabel(p.Label))
			if grid {
				for i := range elements {
					row[i+1] = "—"
				}
			}
			optionRows = append(optionRows, row)
		}
	}
	if len(optionRows) > 0 {
		rows = append(rows, optionRows...)
		rows = append(rows, blank("Options total"))
	}
	return doc(columns, rows)
}

// shortLabel keeps a fee row readable: the item up to its first clause.
func shortLabel(label string) string {
	for _, sep := range []string{"; ", ": "} {
		if i := strings.Index(label, sep); i > 0 {
			label = label[:i]
		}
	}
	return label
}

func (b *builder) review() reports.Section {
	sec := reports.Section{ID: reports.ReviewSectionID, Title: "Before issue — resolve or accept", Internal: true, Blocks: []reports.Block{}}
	add := func(id, label, text string, extra map[string]any) {
		basis := map[string]any{}
		for k, v := range b.base {
			basis[k] = v
		}
		for k, v := range extra {
			basis[k] = v
		}
		raw, _ := json.Marshal(basis)
		sec.Blocks = append(sec.Blocks, reports.Block{ID: id, Label: label, Text: text, Origin: "calculation", ReviewStatus: "proposed", Meaning: "stated", Basis: raw, Provisional: true})
	}
	if b.draft {
		add("rfp:check:discipline_review", "Rules await owner review", "The "+strings.ToLower(b.r.Title)+" rules are a draft. Drafting and editing work; issue stays blocked until the owner marks them reviewed.", map[string]any{"blocking": true})
	}
	for _, c := range b.r.Checks {
		label := c.Label
		if c.Blocking {
			label = "Blocks issue — " + label
		}
		add("rfp:check:"+c.ID, label, c.Text, map[string]any{"check": c, "blocking": c.Blocking})
	}
	return sec
}

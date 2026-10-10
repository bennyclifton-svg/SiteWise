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
const BudgetBlockID = reports.RFPBudgetBlockID

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
	// The PDF title names the discipline; the header carries the issue details.
	header := "Ref " + b.field("ref") + " · Issued " + b.field("issued") + " · Closes " + b.field("closes") + " · Contact " + b.field("contact")
	b.add("project", "rfp:header", "", header, nil, map[string]any{"fields": b.r.Fields})
	// Page one is the project, the same on every RFP: the opening sentence,
	// the grid in the Profile card's columns, the site conditions with every
	// answer, then the other shared rows. Each cell keeps its control key in
	// the basis so the screen can edit the value where it prints.
	rest := []Row{}
	for _, row := range b.r.Project {
		if row.Label == "" {
			b.add("project", "rfp:project:intro", "", row.Text, nil, map[string]any{"facts": b.r.Facts, "parts": b.r.AllParts})
			continue
		}
		rest = append(rest, row)
	}
	b.add("project", "rfp:project:grid", "", "", b.grid(), map[string]any{"grid": b.r.Grid, "facts": b.r.Facts})
	if len(b.r.Conditions) > 0 {
		b.add("project", "rfp:project:conditions", "", b.conditionLines(), nil, map[string]any{"conditions": b.r.Conditions})
	}
	for _, row := range rest {
		b.add("project", "rfp:project:"+row.Key, "", row.Label+": "+row.Text, nil, map[string]any{"row": row})
	}
	if b.r.Budget != "" {
		b.add("project", BudgetBlockID, "Construction budget", "$"+formatNumber(b.r.Budget)+" "+b.budgetBasis(), nil, map[string]any{"budget": b.r.Budget})
	}
}

// grid lays the project grid out in the Profile card's three columns, each
// cell "Label: value", so a long value wraps across a third of the page
// rather than a sixth. A shorter column leaves its cell blank.
func (b *builder) grid() *reports.Table {
	columns := []string{}
	height := 0
	for _, c := range b.r.Grid {
		columns = append(columns, c.Title)
		height = max(height, len(c.Cells))
	}
	rows := make([][]string, height)
	for i := range rows {
		for _, c := range b.r.Grid {
			cell := ""
			if i < len(c.Cells) {
				cell = c.Cells[i].Label + ": " + c.Cells[i].Text
			}
			rows[i] = append(rows[i], cell)
		}
	}
	return doc(columns, rows)
}

// conditionLines prints every answer, one full-width line per answer, so
// page one is no longer than the table it replaced (owner decision O1:
// three pages). Each line names its answer, so it needs no heading.
func (b *builder) conditionLines() string {
	by := map[string][]string{}
	for _, c := range b.r.Conditions {
		by[c.Answer] = append(by[c.Answer], c.Label)
	}
	lines := []string{}
	for _, a := range []struct{ answer, lead string }{
		{"Yes", "Conditions that apply"}, {"No", "Conditions that do not apply"},
		{"To be confirmed", "Conditions to be confirmed (sources disagree)"}, {"Not known", "Conditions not yet known"},
	} {
		if len(by[a.answer]) > 0 {
			lines = append(lines, a.lead+": "+strings.Join(by[a.answer], "; ")+".")
		}
	}
	return strings.Join(lines, "\n")
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
	// The discipline's own project rows describe this appointment; they
	// follow the role in the same block.
	for _, row := range b.r.Appointment {
		if row.Label != "" {
			lines = append(lines, row.Label+": "+row.Text)
			continue
		}
		lines = append(lines, row.Text)
	}
	b.add("role", "rfp:role", "", strings.Join(lines, "\n"), nil, map[string]any{"rows": b.r.Role, "appointment": b.r.Appointment})
}

func (b *builder) stages() {
	rows := [][]string{}
	options := [][]string{}
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
			rows = append(rows, []string{s.Name, text})
		case Ask:
			rows = append(rows, []string{s.Name, "To confirm before issue: " + text})
		case Option:
			why := s.Assumption
			if after, ok := strings.CutPrefix(s.Why, "Follows "); ok && why == "" {
				why = "With the " + strings.ToLower(after) + "."
			}
			if why == "" && s.Source == "boundary" {
				why = "A later stage the client may commission."
			}
			if why == "" {
				why = "The client opts in."
			}
			options = append(options, []string{s.Name + " (option)", text + ". " + why})
		}
	}
	// Options follow the base stages in one table: an option is scope the
	// client may buy, priced on its own line in R1.
	rows = append(rows, options...)
	b.add("stages", "rfp:stages", "", b.r.StageIntro+" Options are priced on their own and are not in the base fee.", doc([]string{"Stage", "Deliverable at sign-off"}, rows), map[string]any{"stages": b.r.Stages, "boundary": b.r.Boundary})
	if len(b.r.Elements) > 0 {
		lines := []string{b.r.ElementNote}
		for _, e := range b.r.Elements {
			switch e.State {
			case Include:
				lines = append(lines, e.Code+" "+e.Name+": "+e.Text+".")
			case Option:
				lines = append(lines, e.Code+" "+e.Name+" (option): "+e.Text+". "+e.Assumption)
			case Ask:
				lines = append(lines, e.Code+" "+e.Name+" (to confirm before issue): "+e.Text+".")
			}
		}
		b.add("stages", "rfp:elements", "", strings.Join(lines, "\n"), nil, map[string]any{"elements": b.r.Elements})
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

// positions prints each canonical item under its position: included,
// optional or excluded with the party who does it. A grouped list reads like
// the golden I/O/X table and, unlike a table row, can break across pages.
func (b *builder) positions() {
	groups := map[string][]string{}
	for _, p := range b.r.Positions {
		item := p.Label
		if p.Value == "X" {
			item += " (" + p.Party + ")"
		}
		// An included item's label already states it; only an option or a
		// decision carries the condition it waits on.
		if p.Assumption != "" && p.Value != "I" {
			item += " (" + compactAssumption(p.Assumption) + ")"
		}
		groups[p.Value] = append(groups[p.Value], item)
	}
	scale := "Service levels are set for a " + b.r.Scale + " project"
	if b.r.ScaleSource == "rule" && b.r.Scale == "complex" {
		scale += " (" + b.r.ScaleWhy + ")"
	}
	lines := []string{scale + "; technical items follow the project's own facts."}
	for _, g := range []struct{ value, title string }{
		{"I", "Included in the fee"},
		{"O", "Optional: price each on its own line in R1"},
		{"X", "Excluded: the party named does it"},
		{Ask, "To confirm before issue"},
	} {
		if items := groups[g.value]; len(items) > 0 {
			lines = append(lines, g.title+": "+strings.Join(items, "; ")+".")
		}
	}
	lines = append(lines, b.r.Departures)
	b.add("positions", "rfp:positions", "", strings.Join(lines, "\n"), nil, map[string]any{"positions": b.r.Positions, "scale": b.r.Scale, "scale_source": b.r.ScaleSource})
	if len(b.r.ScopeSchedule) > 0 {
		intro := []string{}
		rows := [][]string{}
		for _, row := range b.r.ScopeSchedule {
			if row.Label == "" {
				intro = append(intro, row.Text)
			} else {
				rows = append(rows, []string{row.Label, row.Text})
			}
		}
		var table *reports.Table
		if len(rows) > 0 {
			table = doc([]string{"System / interface", "Allocation and acceptance basis"}, rows)
		}
		b.add("positions", "rfp:scope_schedule", "Responsibility schedule", strings.Join(intro, "\n"), table, map[string]any{"rows": b.r.ScopeSchedule})
	}
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
	// Options follow the base total and are totalled on their own, as in the
	// golden R1: each option stage by name, then the section 5 options.
	options := 0
	for _, s := range b.r.Stages {
		if s.State == Option {
			rows = append(rows, stageRow(s, "Option: "+s.Name))
			options++
		}
	}
	for _, p := range b.r.Positions {
		if p.Value == "O" {
			rows = append(rows, blank("Each optional item in section 5, one line each"))
			options++
			break
		}
	}
	if options > 0 {
		rows = append(rows, blank("Options total"))
	}
	return doc(columns, rows)
}

// compactAssumption shortens a printed pricing assumption for the grouped
// position list, where the group already says how the item is priced.
func compactAssumption(a string) string {
	a = strings.TrimSuffix(a, ".")
	for _, prefix := range []string{"Price as an option: ", "Priced as an option ", "Included: "} {
		a = strings.TrimPrefix(a, prefix)
	}
	return strings.Replace(a, "needed only if ", "if ", 1)
}

// shortLabel keeps a fee row readable: the item up to its first clause.
func shortLabel(label string) string {
	for _, sep := range []string{"; "} {
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

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
	for _, id := range []string{"project", "role", "stages", "allowances", "positions", "terms", "returnables"} {
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
	b.returnables()
	b.consultantLayout()
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
	printed := [][]string{}
	height := 0
	for _, c := range b.r.Grid {
		columns = append(columns, c.Title)
		cells := []string{}
		for _, cell := range c.Cells {
			if !cell.Empty {
				cells = append(cells, cell.Label+": "+cell.Text)
			}
		}
		printed = append(printed, cells)
		height = max(height, len(cells))
	}
	rows := make([][]string, height)
	for i := range rows {
		for _, cells := range printed {
			cell := ""
			if i < len(cells) {
				cell = cells[i]
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

func (b *builder) stages() { b.consultantStages() }

func (b *builder) allowances() {
	rows := [][]string{}
	for _, a := range b.r.Allowances {
		rows = append(rows, []string{a.Label, a.Text})
	}
	b.add("allowances", "rfp:allowances", "", b.r.AllowNote, doc([]string{"Item", "Allowance"}, rows), map[string]any{"allowances": b.r.Allowances})
}

// positions prints each canonical item under its position: included,
// optional or excluded with the party who does it. A grouped list reads like
// the golden I/O/X table and, unlike a table row, can break across pages.
func (b *builder) positions() { b.consultantServices() }

func (b *builder) terms() {
	lines := []string{}
	for _, t := range b.r.Terms {
		lines = append(lines, "- "+t.Text)
	}
	b.add("terms", "rfp:terms", "", strings.Join(lines, "\n"), nil, map[string]any{"rows": b.r.Terms})
}

func (b *builder) returnables() {
	lines := []string{}
	for _, r := range b.r.Returnables {
		if r.Key == "r1" {
			continue
		}
		lines = append(lines, r.Label+": "+r.Text)
	}

	b.add("returnables", "rfp:returnables", "", strings.Join(lines, "\n"), nil, map[string]any{"rows": b.r.Returnables})
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

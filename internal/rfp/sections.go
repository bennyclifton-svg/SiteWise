package rfp

import (
	"encoding/json"
	"fmt"
	"slices"
	"strings"
	"unicode"

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
	opt     partOptions
}

// partOptions render a discipline as one part of a combined engagement's
// RFP (group.go). The zero value is the discipline's own RFP, unchanged.
type partOptions struct {
	// part prices nothing in the stage table: the engagement's one fee
	// schedule asks for every discipline's fees.
	part bool
	// moved lists services another discipline in the engagement does; they
	// print once, under that discipline.
	moved map[string]bool
}

// Sections renders a resolved RFP into the template's sections, followed by
// the internal Before-issue section. Every block of a draft discipline is
// provisional, which keeps it from being issued (report_issue.go).
func Sections(r Resolved, template knowledge.ReportTemplate) ([]reports.Section, error) {
	return sections(r, template, partOptions{})
}

func sections(r Resolved, template knowledge.ReportTemplate, opt partOptions) ([]reports.Section, error) {
	b := &builder{r: r, draft: r.Status != "reviewed", section: map[string]*reports.Section{}, opt: opt}
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
		if row.Key != "information" {
			b.add("project", "rfp:project:"+row.Key, "", row.Label+": "+row.Text, nil, map[string]any{"row": row})
		}
	}
	b.transmittal()
	if b.r.Budget != "" {
		b.add("project", BudgetBlockID, "Construction budget", "$"+formatNumber(b.r.Budget)+" "+b.budgetBasis(), nil, map[string]any{"budget": b.r.Budget})
	}
}

// DocumentRow is one repository document as an RFP schedule prints it.
type DocumentRow struct {
	ID, Number, Title, Revision, Date, Discipline string
}

// DocumentRows reads the saved document identity of each chosen document.
// A row whose identity cannot be read is left out.
func DocumentRows(items []reports.BriefValue) []DocumentRow {
	_, disciplines := vocabLabels()
	out := []DocumentRow{}
	for _, item := range items {
		var basis struct {
			Sources []map[string]string `json:"sources"`
		}
		if json.Unmarshal(item.Basis, &basis) != nil || len(basis.Sources) == 0 {
			continue
		}
		s := basis.Sources[0]
		discipline := s["discipline"]
		if label := disciplines[discipline]; label != "" {
			discipline = label
		}
		out = append(out, DocumentRow{ID: item.ID, Number: s["document_number"], Title: s["title"], Revision: s["revision"], Date: s["date"], Discipline: discipline})
	}
	return out
}

func (b *builder) transmittal() {
	rows := [][]string{}
	ids := []string{}
	for _, d := range DocumentRows(b.r.Transmittal) {
		rows = append(rows, []string{d.Number, d.Title, d.Revision, d.Date, d.Discipline})
		ids = append(ids, d.ID)
	}
	text := ""
	// Keep explicitly authored information as a note alongside the schedule.
	if b.r.Fields["information"].Value != "" {
		text = b.r.Fields["information"].Value
	}
	b.add("project", "rfp:project:information", "Transmittal", text, doc([]string{"Doc number", "Title", "Rev", "Date", "Discipline"}, rows), map[string]any{"document_ids": ids})
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

// routeRoles are the role rows about the procurement route (novation, D&C
// handover). With a design team table they follow it, beside the stage
// pricing, so the role opens with what the consultant is there to achieve.
var routeRoles = map[string]bool{"novation": true, "novation_option": true, "dnc_handover": true}

func (b *builder) role() {
	lines, route := []string{}, []string{}
	for _, row := range b.r.Role {
		switch {
		case row.Text == "":
			// An empty slot, such as no subconsultants, prints nothing.
		case routeRoles[row.Key]:
			route = append(route, row.Text)
		default:
			lines = append(lines, row.Text)
		}
	}
	// The discipline's own project rows describe this appointment; they
	// follow the role.
	for _, row := range b.r.Appointment {
		if row.Label != "" {
			route = append(route, row.Label+": "+row.Text)
			continue
		}
		route = append(route, row.Text)
	}
	// A head contract names the trades the client lets separately, so the
	// tenderer prices attendance on them.
	separate := []string{}
	for _, s := range b.r.Stages {
		if s.State == Exclude && s.Source == "package" && s.By != "" && !slices.Contains(separate, s.By) {
			separate = append(separate, s.By)
		}
	}
	if len(separate) > 0 {
		route = append(route, "The client lets these separately: "+strings.Join(separate, ", ")+". Coordinate with them and give them access as the contract provides.")
	}
	// The owner removed the Design team table (7 Nov 2026): the consultant
	// team lives on the project profile, not in each RFP.
	basis := map[string]any{"rows": b.r.Role, "appointment": b.r.Appointment}
	b.add("role", "rfp:role", "", strings.Join(append(lines, route...), "\n"), nil, basis)
}

func (b *builder) stages() { b.consultantStages() }

func (b *builder) allowances() {
	// Two pairs of columns (owner direction, 8 November 2026): read down
	// the left pair, then the right, so the table takes half the length.
	half := (len(b.r.Allowances) + 1) / 2
	rows := make([][]string, half)
	for i, a := range b.r.Allowances {
		row := i % half
		if rows[row] == nil {
			rows[row] = []string{"", "", "", ""}
		}
		col := 0
		if i >= half {
			col = 2
		}
		rows[row][col], rows[row][col+1] = allowanceItem(a), allowanceQty(a)
	}
	b.add("allowances", "rfp:allowances", "", b.r.AllowNote, doc([]string{"Item", "Qty", "Item", "Qty"}, rows), map[string]any{"allowances": b.r.Allowances, "layout": "allowance_pairs"})
}

// allowanceItem is an allowance's item cell: its label, then what it covers.
func allowanceItem(a AllowanceResult) string {
	if a.Detail == "" {
		return a.Label
	}
	detail := a.Detail
	// "Per stage" reads on as "per stage"; an acronym such as "LOD" keeps its case.
	if r := []rune(detail); len(r) > 1 && unicode.IsUpper(r[0]) && unicode.IsLower(r[1]) {
		detail = string(unicode.ToLower(r[0])) + string(r[1:])
	}
	return a.Label + ": " + detail
}

// allowanceQty is the Qty cell: the count, a blank to fill, or a note that
// the firm proposes it.
func allowanceQty(a AllowanceResult) string {
	switch {
	case a.Value != "":
		return a.Value
	case a.Owner == "consultant":
		return "Proposed in R2"
	}
	return "___"
}

// positions prints each canonical item under its position: included,
// optional or excluded with the party who does it. A grouped list reads like
// the golden I/O/X table and, unlike a table row, can break across pages.
func (b *builder) positions() { b.consultantServices() }

// terms prints the numbered terms (owner, 8 Nov 2026): firms and the client
// cite a term by its number. The agreement is named by its documents'
// titles in term 1; the documents themselves travel in the transmittal, so
// no document table prints here. The basis keeps the chosen ids so the
// editor can show them selected in the repository.
func (b *builder) terms() {
	lines := []string{}
	for _, t := range b.r.Terms {
		lines = append(lines, "- "+t.Text)
	}
	ids := []string{}
	for _, d := range DocumentRows(b.r.Agreement) {
		ids = append(ids, d.ID)
	}
	b.add("terms", "rfp:terms", "", strings.Join(lines, "\n"), nil, map[string]any{"rows": b.r.Terms, "layout": "numbered_terms", "agreement_ids": ids})
}

// returnables prints one row per schedule in the order the comparison
// report reads them back. The response form carries a fill-in table for
// each (form.go); a proposal returned on it reads item by item.
func (b *builder) returnables() {
	rows := [][]string{}
	for _, r := range b.r.Returnables {
		if r.Key == "r1" {
			continue
		}
		rows = append(rows, []string{r.Label, r.Text})
	}
	b.add("returnables", "rfp:returnables", "", b.words().formNote, doc([]string{"Schedule", "What to return"}, rows), map[string]any{"rows": b.r.Returnables})
}

// ResponseFormNote tells the firm how to reply.
const ResponseFormNote = "Reply in this order. The response form (Excel, issued with this RFP) has a table for each schedule and every fee row of section 2; return it with your proposal."

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

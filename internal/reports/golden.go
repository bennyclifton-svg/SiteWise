package reports

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"

	"sitewise/internal/delivery"
	"sitewise/internal/invoices"
)

// The project report in the owner's format
// (docs/plans/2026-11-26-project-report-golden-format.md): the owner's own
// sections in the owner's order, each table holding rows from records,
// rows the PM owns, and the template's placeholder rows. Every generated
// figure is a screen's own (ContractSummary, the register, the variation
// register); unknown prints "TBD", never 0. Nothing here calls Jev or a
// model. The earlier sections stay on screen as the PM's review lists.

// Row kinds in a report table.
const (
	RowRecord      = "record"      // built by Go from a record at every Update
	RowOwn         = "report"      // the PM's own; Update never touches it
	RowPlaceholder = "placeholder" // the template's row, grey until typed in
	RowHistory     = "history"     // an earlier period's row, as issued
)

// Cell tones: what a fill says on the printed page.
const (
	ToneDone = "done" // green: approved, determined
	ToneKey  = "key"  // yellow: the key computed date
)

// RowFacts say what one table row is. Every field is omitted when empty,
// so tables built before this change hash and freeze as they did.
type RowFacts struct {
	Key  string `json:"key,omitempty"`
	Kind string `json:"kind,omitempty"`
	// ID is the report_rows row laid over this one, when the PM has one.
	ID string `json:"id,omitempty"`
	// New: the row's key was not in the last issue. Changed marks the
	// cells whose words differ from the last issue's.
	New     bool     `json:"new,omitempty"`
	Changed []bool   `json:"changed,omitempty"`
	Tones   []string `json:"tones,omitempty"`
	// Typed marks the cells the PM typed over; Base holds SiteWise's own
	// words under each, so the screen can offer them back.
	Typed []bool   `json:"typed,omitempty"`
	Base  []string `json:"base,omitempty"`
	// Flag is "moved" (SiteWise's value under a typed cell has changed
	// since it was typed) or "missing" (the record is gone). Issue waits.
	Flag   string `json:"flag,omitempty"`
	Source string `json:"source,omitempty"` // a document a prepared fill read
	// Pos is where the row stands among the table's rows; a move sets it.
	Pos float64 `json:"pos,omitempty"`
}

// GoldenInput is what the owner's sections read beyond MonthlyInput.
type GoldenInput struct {
	Title      string
	Client     string
	OrgName    string
	Address    string
	Lot        string
	LGA        string
	Consent    string
	ConsentOn  string
	Period     Period
	Issues     []IssueRecord
	Contracts  []invoices.ContractSummary
	Unawarded  []string // live works packages with no award, by title
	Packages   map[string]string
	Agreed     string // the agreed date for practical completion; "" unknown
	Adjusted   string // agreed plus days granted, as the register works it out
	Systems    []GoldenSystem
	SystemDocs map[string]int // documents filed this period, by system
}

// Period is the window a draft reports on.
type Period struct {
	From, To, Label string
}

// IssueRecord is one issued report, for the revision history.
type IssueRecord struct {
	Number int
	Date   string
	By     string
}

// GoldenSystem is a top-level system of the building, in building order.
type GoldenSystem struct {
	ID, Label string
}

// GoldenSections are the owner's sections in order. sectionTitles carries
// the template's titles.
func (m monthly) goldenSections(titles map[string]string) []Section {
	g := m.in.Golden
	if g == nil {
		g = &GoldenInput{}
	}
	sec := func(id string, blocks ...Block) Section {
		return Section{ID: id, Title: titles[id], Essential: true, Blocks: blocks}
	}
	out := []Section{
		sec("cover", m.coverTable(g)),
		sec("quality", m.revisionTable(g), m.distributionTable(), m.block("quality:note", "Controlled document", m.controlledNote(g), map[string]any{"method": "golden_text"})),
		sec("context", m.contextTable(g)),
		sec("summary", m.summaryTable(g)),
		sec("statutory", m.approvalTable(g), m.certificateTable()),
		sec("programme", m.programmeBlocks(g)...),
		sec("cost_planning", m.costTable(g)),
		sec("design", m.designTable(g), m.surveysTable()),
		sec("site", m.siteTable()),
		sec("change", m.variationTable(g)),
		sec("insurances", m.insuranceTable()),
		sec("attach_a", m.scheduleList(g)),
		sec("attach_b", m.attachmentB()),
	}
	return out
}

// table builds a block holding one editable table. rows and facts line up.
func (m monthly) table(id, label string, columns []string, rows [][]string, facts []RowFacts) Block {
	b := m.block(id, label, "", map[string]any{"method": "golden_table", "table": id})
	b.Table = &Table{ID: id, Columns: columns, Rows: rows, Display: "document", Meta: facts}
	return b
}

type rowSet struct {
	rows  [][]string
	facts []RowFacts
}

func (r *rowSet) add(key, kind string, cells ...string) *RowFacts {
	r.rows = append(r.rows, cells)
	r.facts = append(r.facts, RowFacts{Key: key, Kind: kind})
	return &r.facts[len(r.facts)-1]
}

// placeholders adds the template's named rows, and blank ones trailing.
func (r *rowSet) placeholders(table string, width int, named [][]string, blanks int) {
	for i, cells := range named {
		row := make([]string, width)
		copy(row, cells)
		r.add(fmt.Sprintf("ph:%s:%d", table, i), RowPlaceholder, row...)
	}
	for i := 0; i < blanks; i++ {
		r.add(fmt.Sprintf("ph:%s:blank:%d", table, i), RowPlaceholder, make([]string, width)...)
	}
}

func tbd(s string) string {
	if strings.TrimSpace(s) == "" {
		return "TBD"
	}
	return s
}

func shortDay(iso string) string {
	t, err := time.Parse("2006-01-02", iso)
	if err != nil {
		return iso
	}
	return t.Format("2 Jan 2006")
}

func dayOrTBD(p *string) string {
	if p == nil || *p == "" {
		return "TBD"
	}
	return shortDay(*p)
}

func moneyOrTBD(p *int64) string {
	if p == nil {
		return "TBD"
	}
	return invoices.DisplayCents(*p)
}

// ---- cover and quality

func (m monthly) coverTable(g *GoldenInput) Block {
	var r rowSet
	r.add("cover:title", RowRecord, "Title", tbd(g.Title))
	r.add("cover:project", RowRecord, "Project", tbd(m.in.ProjectName))
	r.add("cover:period", RowRecord, "Period", tbd(g.Period.Label))
	r.add("cover:for", RowRecord, "Prepared for", tbd(g.Client))
	r.add("cover:by", RowRecord, "Prepared by", tbd(g.OrgName))
	// The cover's image: drop one on this row (slice 5).
	r.placeholders("cover", 2, [][]string{{"Image", ""}}, 0)
	return m.table("cover", "Cover", []string{"Item", "Detail"}, r.rows, r.facts)
}

func (m monthly) revisionTable(g *GoldenInput) Block {
	var r rowSet
	for _, is := range g.Issues {
		r.add(fmt.Sprintf("issue:%d", is.Number), RowRecord, fmt.Sprintf("Issue %d", is.Number), shortDay(is.Date), tbd(is.By))
	}
	r.add(fmt.Sprintf("issue:%d", m.in.Number), RowRecord, fmt.Sprintf("Issue %d", m.in.Number), "TBD", "TBD")
	return m.table("revisions", "Revision history", []string{"Revision", "Date", "Issued by"}, r.rows, r.facts)
}

func (m monthly) distributionTable() Block {
	var r rowSet
	r.placeholders("distribution", 4, nil, 2)
	return m.table("distribution", "Distribution", []string{"Name", "Organisation", "Position", "Copy"}, r.rows, r.facts)
}

func (m monthly) controlledNote(g *GoldenInput) string {
	who, by := tbd(g.Client), tbd(g.OrgName)
	return fmt.Sprintf("This report was prepared by %s for %s. It is a controlled document; copies not on the distribution list are not kept up to date.", by, who)
}

// ---- 01 site and project context

func (m monthly) contextTable(g *GoldenInput) Block {
	var r rowSet
	r.add("site:address", RowRecord, "Address", tbd(g.Address))
	r.add("site:lot", RowRecord, "Lot and DP", tbd(g.Lot))
	r.add("site:lga", RowRecord, "Local authority", tbd(g.LGA))
	consent := tbd(g.Consent)
	if g.Consent != "" && g.ConsentOn != "" {
		consent += ", " + shortDay(g.ConsentOn)
	}
	r.add("site:consent", RowRecord, "Development consent", consent)
	r.placeholders("context", 2, [][]string{{"Development description", "TBD"}}, 0)
	return m.table("context", "Site and project context", []string{"Item", "Detail"}, r.rows, r.facts)
}

// ---- 02 executive summary: this period's row, then earlier ones as issued

func (m monthly) summaryTable(g *GoldenInput) Block {
	var r rowSet
	label := tbd(g.Period.Label)
	r.add(fmt.Sprintf("period:%d", m.in.Number), RowRecord, label, m.periodBullets(g))
	if prev := m.prevTable("summary"); prev != nil {
		for i, row := range prev.Rows {
			if i < len(prev.Meta) && prev.Meta[i].Key != "" && len(row) == 2 {
				r.add(prev.Meta[i].Key, RowHistory, row...)
			}
		}
	}
	return m.table("summary", "Executive summary", []string{"Period", "Summary"}, r.rows, r.facts)
}

// periodBullets are this period's events from the records, under one
// heading per contract and one for the project. The narrative is the PM's.
func (m monthly) periodBullets(g *GoldenInput) string {
	start := m.periodStart()
	in := func(d *string) bool { return d != nil && *d != "" && (start == "" || *d > start) }
	var project []string
	for _, it := range m.in.Delivery {
		switch it.Kind {
		case "eot_claim":
			var d delivery.EOTClaimDetails
			_ = json.Unmarshal(it.Details, &d)
			if in(d.ReceivedOn) {
				project = append(project, fmt.Sprintf("- %s received%s", itemName(it), claimedDays(d.DaysClaimed)))
			}
			if d.Result.Decided() && in(d.Result.AssessedOn) {
				project = append(project, fmt.Sprintf("- %s assessed: %s", itemName(it), resultWords(d.Result)))
			}
		case "delay_notice":
			var d delivery.DelayNoticeDetails
			_ = json.Unmarshal(it.Details, &d)
			if in(d.ReceivedOn) {
				project = append(project, fmt.Sprintf("- %s received: %s", itemName(it), strings.TrimSpace(it.Title)))
			}
		}
	}
	for _, p := range m.in.Packages {
		if p.AwardedOn != "" && (start == "" || p.AwardedOn > start) {
			project = append(project, fmt.Sprintf("- %s awarded to %s", p.Title, tbd(p.AwardedTo)))
		}
	}
	var parts []string
	for _, c := range g.Contracts {
		var lines []string
		if m.in.Ledger != nil {
			for _, va := range m.in.Ledger.Variations {
				if va.AwardID == c.AwardID && va.SubmittedOn != "" && (start == "" || va.SubmittedOn > start) {
					lines = append(lines, fmt.Sprintf("- %s %s submitted", va.Number, va.Title))
				}
			}
		}
		if c.Latest != nil && (start == "" || c.Latest.IssuedOn > start) {
			lines = append(lines, fmt.Sprintf("- Payment schedule %s issued", c.Latest.Number))
		}
		if len(lines) > 0 {
			parts = append(parts, "**"+c.Title+"**\n"+strings.Join(lines, "\n"))
		}
	}
	if len(project) > 0 {
		parts = append(parts, "**Project**\n"+strings.Join(project, "\n"))
	}
	if len(parts) == 0 {
		return "Nil to report from SiteWise's records this period."
	}
	return strings.Join(parts, "\n")
}

func itemName(it delivery.Item) string {
	prefix := map[string]string{"eot_claim": "EOT", "delay_notice": "NOD", "rfi": "RFI", "letter": "LTR"}[it.Kind]
	if it.Number != nil && prefix != "" {
		return fmt.Sprintf("%s-%d", prefix, *it.Number)
	}
	if it.Title != "" {
		return it.Title
	}
	return "A register row"
}

func claimedDays(n *int) string {
	if n == nil {
		return ""
	}
	return fmt.Sprintf(", %d days claimed", *n)
}

func resultWords(r *delivery.EOTResult) string {
	switch {
	case r.Outcome == delivery.OutcomeRejected:
		return "rejected"
	case r.DaysGranted != nil:
		return fmt.Sprintf("%d days granted", *r.DaysGranted)
	}
	return r.Outcome
}

// prevTable is a table of the last issue, by its id.
func (m monthly) prevTable(id string) *Table {
	if m.in.Previous == nil {
		return nil
	}
	for _, s := range m.in.Previous.Sections {
		for _, b := range s.Blocks {
			if b.Table != nil && b.Table.ID == id {
				return b.Table
			}
		}
	}
	return nil
}

// ---- 03 statutory approvals

// approvalPlaceholders are the owner's standing list (golden report,
// section 03). Row headings, not rule citations; draft until reviewed.
var approvalPlaceholders = []string{"Energy distributor", "Arborist", "Ecology", "Wastewater", "Long service levy", "Transport", "Land registry", "Water", "Stormwater", "Public domain", "Fire service"}

func (m monthly) approvalTable(g *GoldenInput) Block {
	var r rowSet
	if g.Consent != "" {
		f := r.add("approval:consent", RowRecord, tbd(g.LGA), "Development consent", g.Consent, "", tbd(shortDay(g.ConsentOn)), "Approved")
		if g.ConsentOn != "" {
			f.Tones = []string{"", "", "", "", ToneDone, ""}
		}
	}
	for _, it := range m.in.Delivery {
		if it.Kind != "approval" {
			continue
		}
		var d delivery.ApprovalDetails
		_ = json.Unmarshal(it.Details, &d)
		f := r.add("approval:"+it.ID, RowRecord, tbd(d.Authority), tbd(it.Title), tbd(d.Reference), dayOrTBD(d.SubmittedOn), dayOrTBD(d.DeterminedOn), tbd(statusWords(it.Status)))
		if d.DeterminedOn != nil && *d.DeterminedOn != "" {
			f.Tones = []string{"", "", "", "", ToneDone, ""}
		}
	}
	var named [][]string
	for _, a := range approvalPlaceholders {
		named = append(named, []string{a, "Nil to report", "", "", "TBD", ""})
	}
	r.placeholders("approvals", 6, named, 0)
	return m.table("approvals", "Authority approvals", []string{"Authority", "Approval", "Reference", "Submitted", "Determined", "Status"}, r.rows, r.facts)
}

func statusWords(s string) string {
	return strings.ReplaceAll(capitalise(s), "_", " ")
}

func (m monthly) certificateTable() Block {
	var r rowSet
	r.placeholders("certificates", 4, nil, 2)
	return m.table("certificates", "Construction certificate", []string{"Certificate", "Certifier", "Date", "Status"}, r.rows, r.facts)
}

// ---- 04 programme review

func (m monthly) programmeBlocks(g *GoldenInput) []Block {
	var out []Block
	main := ""
	for _, c := range g.Contracts {
		if c.Kind == "works" {
			main = c.AwardID
			break
		}
	}
	for _, c := range g.Contracts {
		if c.Kind != "works" {
			continue
		}
		var r rowSet
		r.placeholders("milestones:"+c.AwardID, 5, [][]string{{"Contract execution", "TBD"}, {"Commencement", "TBD"}}, 0)
		if c.AwardID == main {
			r.add("pc:original", RowRecord, "Date for practical completion", tbd(shortDay(g.Agreed)), "", "", "")
			for _, it := range m.in.Delivery {
				if it.Kind != "eot_claim" {
					continue
				}
				var d delivery.EOTClaimDetails
				_ = json.Unmarshal(it.Details, &d)
				granted, status := "TBD", tbd(statusWords(it.Status))
				if d.Result.Decided() {
					granted = "0"
					if d.Result.DaysGranted != nil {
						granted = fmt.Sprint(*d.Result.DaysGranted)
					}
					status = resultWords(d.Result)
				}
				claimed := "TBD"
				if d.DaysClaimed != nil {
					claimed = fmt.Sprint(*d.DaysClaimed)
				}
				r.add("eot:"+it.ID, RowRecord, itemName(it)+" "+strings.TrimSpace(it.Title), "", claimed, granted, status)
			}
			r.placeholders("eot", 5, nil, 1)
			f := r.add("pc:adjusted", RowRecord, "Adjusted date for practical completion", tbd(shortDay(g.Adjusted)), "", "", "")
			f.Tones = []string{"", ToneKey, "", "", ""}
		} else {
			r.placeholders("pc:"+c.AwardID, 5, [][]string{{"Date for practical completion", "TBD"}}, 0)
		}
		out = append(out, m.table("milestones:"+c.AwardID, c.Title, []string{c.Title, "Date", "Days claimed", "Days approved", "Status"}, r.rows, r.facts))
	}
	if len(out) == 0 {
		var r rowSet
		r.placeholders("milestones", 5, [][]string{{"Date for practical completion", "TBD"}}, 0)
		out = append(out, m.table("milestones", "Milestones", []string{"Contract", "Date", "Days claimed", "Days approved", "Status"}, r.rows, r.facts))
	}
	var nod rowSet
	for _, it := range m.in.Delivery {
		if it.Kind != "delay_notice" {
			continue
		}
		var d delivery.DelayNoticeDetails
		_ = json.Unmarshal(it.Details, &d)
		nod.add("nod:"+it.ID, RowRecord, itemName(it)+" "+strings.TrimSpace(it.Title), dayOrTBD(d.ReceivedOn), tbd(d.Cause), tbd(statusWords(it.Status)))
	}
	nod.placeholders("nod", 4, nil, 1)
	out = append(out, m.table("nod", "Notices of delay", []string{"Notice", "Received", "Narrative", "Status"}, nod.rows, nod.facts))
	var lead rowSet
	for _, p := range m.in.Packages {
		if p.Kind == "supply" {
			lead.add("pkg:"+p.ID, RowRecord, p.Title, tbd(p.AwardedTo), tbd(packageStatus(p)), "TBD")
		}
	}
	lead.placeholders("long_lead", 4, nil, 2)
	out = append(out, m.table("long_lead", "Long lead items", []string{"Item", "Supplier", "Status", "Lead time"}, lead.rows, lead.facts))
	return out
}

// ---- 05 cost planning

func (m monthly) costTable(g *GoldenInput) Block {
	var r rowSet
	var o, v, a int64
	known := true
	for _, c := range g.Contracts {
		if c.Kind != "works" {
			continue
		}
		vars := c.Variations
		r.add("contract:"+c.AwardID, RowRecord, c.Title, moneyOrTBD(c.Original), moneyOrTBD(&vars), moneyOrTBD(c.Adjusted))
		if c.Original == nil || c.Adjusted == nil {
			known = false
		} else {
			o, v, a = o+*c.Original, v+c.Variations, a+*c.Adjusted
		}
	}
	// A package not yet awarded has no contract sum: the total is not
	// known either, and no unawarded firm's price is printed.
	for _, t := range g.Unawarded {
		r.add("unawarded:"+t, RowRecord, t, "TBD", "TBD", "TBD")
		known = false
	}
	if len(r.rows) == 0 {
		r.placeholders("cost", 4, [][]string{{"Head contract", "TBD", "TBD", "TBD"}}, 0)
	} else if known {
		r.add("cost:total", RowRecord, "Total", invoices.DisplayCents(o), invoices.DisplayCents(v), invoices.DisplayCents(a))
	} else {
		r.add("cost:total", RowRecord, "Total", "TBD", "TBD", "TBD")
	}
	b := m.table("cost_planning", "Cost planning", []string{"Major package", "Original ($)", "Variation ($)", "Adjusted ($)"}, r.rows, r.facts)
	b.Text = "Adjusted is original plus approved variations, plus the adjustment on each provisional sum marked final. Figures ex GST, as the Cost Plan's Variations tab shows them."
	return b
}

// ---- 06 design management

func (m monthly) designTable(g *GoldenInput) Block {
	var r rowSet
	open := map[string]int{}
	for _, it := range m.in.Delivery {
		if it.Kind == "rfi" && it.SystemID != "" && it.Status != "closed" && it.Status != "answered" {
			open[it.SystemID]++
		}
	}
	for _, s := range g.Systems {
		var lines []string
		if n := g.SystemDocs[s.ID]; n > 0 {
			lines = append(lines, "- "+plural(n, "document", "documents")+" filed this period")
		}
		if n := open[s.ID]; n > 0 {
			lines = append(lines, "- "+plural(n, "open RFI", "open RFIs"))
		}
		text := "Nil to report"
		if len(lines) > 0 {
			text = strings.Join(lines, "\n")
		}
		r.add("system:"+s.ID, RowRecord, s.Label, text)
	}
	return m.table("design", "Design management", []string{"Discipline", "Summary and risks"}, r.rows, r.facts)
}

func (m monthly) surveysTable() Block {
	var r rowSet
	r.placeholders("surveys", 3, nil, 2)
	return m.table("surveys", "Reports, surveys and shop drawings", []string{"Item", "By", "Status"}, r.rows, r.facts)
}

// ---- 07 site activity

func (m monthly) siteTable() Block {
	var r rowSet
	engaged := func(kind string) string {
		var lines []string
		for _, p := range m.in.Packages {
			if p.Kind == kind && p.AwardedTo != "" {
				lines = append(lines, fmt.Sprintf("- %s: %s", p.Title, p.AwardedTo))
			}
		}
		sort.Strings(lines)
		if len(lines) == 0 {
			return "Nil to report"
		}
		return "Engaged:\n" + strings.Join(lines, "\n")
	}
	r.placeholders("site", 2, [][]string{{"Safety and induction", "Nil to report"}}, 0)
	r.add("site:consultants", RowRecord, "Consultant procurement", engaged("services"))
	r.add("site:trades", RowRecord, "Trade procurement", engaged("works"))
	next := []string{}
	for _, b := range m.next() {
		if b.Text != "" && b.ID != "" && !strings.HasPrefix(b.ID, "comment:") {
			next = append(next, "- "+b.Text)
		}
	}
	works := "Nil to report"
	if len(next) > 0 {
		works = strings.Join(next, "\n")
	}
	r.add("site:works", RowRecord, "Site conditions and works", works)
	r.placeholders("site_tail", 2, [][]string{{"PCG meetings", "Nil to report"}, {"Quality and ITP", "Nil to report"}}, 0)
	var open []string
	for _, it := range m.in.Delivery {
		switch it.Kind {
		case "action", "issue", "risk", "decision", "rfi":
		default:
			continue
		}
		if it.Status == "closed" || it.Status == "answered" || it.Status == "done" {
			continue
		}
		line := "- " + itemName(it)
		if it.Kind != "rfi" || it.Number == nil {
			line = "- " + strings.TrimSpace(it.Title)
		} else {
			line += " " + strings.TrimSpace(it.Title)
		}
		if it.With != "" {
			line += " (with " + it.With + ")"
		}
		if it.TargetDate != nil && *it.TargetDate != "" {
			line += ", due " + shortDay(*it.TargetDate)
		}
		open = append(open, line)
	}
	issues := "Nil to report"
	if len(open) > 0 {
		issues = strings.Join(open, "\n")
	}
	r.add("site:open", RowRecord, "Open issues and actions", issues)
	return m.table("site", "Site activity", []string{"Topic", "Summary"}, r.rows, r.facts)
}

// ---- 08 variation and change control

func (m monthly) variationTable(g *GoldenInput) Block {
	var r rowSet
	if m.in.Ledger != nil {
		for _, va := range m.in.Ledger.Variations {
			// Approved when approved, else the PM's estimate, else the amount
			// claimed, labelled; a variation with none is TBC, never 0.
			amount, basis := va.Approved, "approved"
			if va.Status != invoices.VariationApproved {
				amount, basis = va.Forecast()
			}
			value := "TBC"
			if amount != nil {
				value = invoices.DisplayCents(*amount)
				if basis != "" && basis != "approved" {
					value += " (" + basis + ")"
				}
			}
			r.add("var:"+va.ID, RowRecord, tbd(g.Packages[va.AwardID]), va.Number, tbd(va.Title), statusWords(va.Status), value, "")
		}
	}
	r.placeholders("change", 6, nil, 2)
	return m.table("change", "Variation and change control", []string{"Package", "No.", "Description", "Status", "Amount ($)", "Fund"}, r.rows, r.facts)
}

// ---- 09 insurances

func (m monthly) insuranceTable() Block {
	var r rowSet
	r.placeholders("insurances", 4, [][]string{{"Contract works insurance", "TBD", "TBD", ""}, {"Public liability", "TBD", "TBD", ""}, {"Workers compensation", "TBD", "TBD", ""}, {"Security or bank guarantee", "TBD", "TBD", ""}}, 0)
	return m.table("insurances", "Insurances, security and bonds", []string{"Item", "Detail", "Expiry", "Status"}, r.rows, r.facts)
}

// ---- attachments

func (m monthly) scheduleList(g *GoldenInput) Block {
	var r rowSet
	for _, c := range g.Contracts {
		if c.Latest == nil {
			continue
		}
		inc := c.Latest.IncGST
		r.add("schedule:"+c.AwardID, RowRecord, c.Title, c.Latest.Number, c.Latest.Period, shortDay(c.Latest.IssuedOn), invoices.DisplayCents(inc))
	}
	if len(r.rows) == 0 {
		r.placeholders("schedules", 5, [][]string{{"Nil to report"}}, 0)
	}
	return m.table("schedules", "Payment schedules", []string{"Contract", "Claim", "Period", "Issued", "Value inc GST ($)"}, r.rows, r.facts)
}

func (m monthly) attachmentB() Block {
	var r rowSet
	r.placeholders("attach_b", 2, [][]string{{"TBD", ""}}, 0)
	return m.table("attach_b", "Attachment B", []string{"Document", "Note"}, r.rows, r.facts)
}

// MarkNew compares each table with the last issue's: a row whose key was
// not there is new, and a cell whose words differ is changed. The first
// report marks nothing.
func MarkNew(sections []Section, previous []Section) {
	if previous == nil {
		return
	}
	prior := map[string]map[string][]string{}
	for _, s := range previous {
		for _, b := range s.Blocks {
			if b.Table == nil || b.Table.ID == "" {
				continue
			}
			rows := map[string][]string{}
			for i, row := range b.Table.Rows {
				if i < len(b.Table.Meta) && b.Table.Meta[i].Key != "" {
					rows[b.Table.Meta[i].Key] = row
				}
			}
			prior[b.Table.ID] = rows
		}
	}
	for si := range sections {
		for bi := range sections[si].Blocks {
			t := sections[si].Blocks[bi].Table
			if t == nil || t.ID == "" {
				continue
			}
			was, compared := prior[t.ID]
			if !compared {
				continue
			}
			for i := range t.Meta {
				f := &t.Meta[i]
				if f.Kind == RowPlaceholder || f.Kind == RowHistory || f.Key == "" {
					continue
				}
				old, ok := was[f.Key]
				if !ok {
					f.New = true
					continue
				}
				changed := make([]bool, len(t.Rows[i]))
				any := false
				for c, cell := range t.Rows[i] {
					if c >= len(old) || old[c] != cell {
						changed[c], any = true, true
					}
				}
				if any {
					f.Changed = changed
				}
			}
		}
	}
}

// PeriodLabel names a period from its dates: "October 2026" for one whole
// calendar month, "3 to 17 November 2026", "20 October to 5 November
// 2026", or "To 27 November 2026" for a first report with no start.
func PeriodLabel(from, to string) string {
	t, err := time.Parse("2006-01-02", to)
	if err != nil {
		return ""
	}
	f, err := time.Parse("2006-01-02", from)
	if from == "" || err != nil {
		return "To " + t.Format("2 January 2006")
	}
	if f.Day() == 1 && f.Year() == t.Year() && f.Month() == t.Month() && t.AddDate(0, 0, 1).Day() == 1 {
		return t.Format("January 2006")
	}
	switch {
	case f.Year() == t.Year() && f.Month() == t.Month():
		return fmt.Sprintf("%d to %s", f.Day(), t.Format("2 January 2006"))
	case f.Year() == t.Year():
		return fmt.Sprintf("%s to %s", f.Format("2 January"), t.Format("2 January 2006"))
	}
	return fmt.Sprintf("%s to %s", f.Format("2 January 2006"), t.Format("2 January 2006"))
}

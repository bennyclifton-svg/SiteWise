package reports

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"

	"sitewise/internal/costs"
	"sitewise/internal/delivery"
	"sitewise/internal/figures"
	"sitewise/internal/invoices"
)

// The project report (docs/plans/2026-11-17-monthly-report.md): what a client
// reads each period, assembled in Go from the figures the Cost Plan, Program
// and Procurement screens show. Every sentence is Go's; the PM's comments are
// protected edits over the comment blocks. Nothing here calls Jev or a model.

const (
	KindMonthly       = "monthly"
	MonthlyTemplateID = "tpl.monthly-report"
	MonthlyTitle      = "Project report"
	// NextWindowDays is the "next period" window from the as-at date (owner
	// decision 14's default): the next issue date is not known in advance.
	NextWindowDays = 28
	// List caps keep the issued PDF inside four pages; each capped list says
	// how many more are in SiteWise.
	maxPackageRows   = 15
	maxRegisterItems = 5
	maxMilestones    = 10
	maxNext          = 10
	maxMoved         = 12
)

// Basis methods. A block whose method differs from the previous issue's is
// "not compared" (a template or method change between reports).
const (
	methodCost       = "monthly_cost_v1"
	methodCompletion = "monthly_program_v1"
	methodPackages   = "monthly_packages_v1"
	methodRegister   = "monthly_register_v1"
)

// MonthlyInput is everything one draft reads, already loaded by the store on
// the draft's snapshot. A nil part was not there (no cost plan, no Program).
type MonthlyInput struct {
	ProjectName string
	// AsAt is the Sydney date of this Update: the figures are as at it.
	AsAt string
	// Number is the issue this draft becomes.
	Number   int
	Previous *MonthlyPrevious
	// Costs is costs.Summarize(plan, "overall"), PackageCosts by package.
	Costs        *costs.Totals
	PackageCosts *costs.Totals
	// Ledger is the invoice ledger as the Cost Plan grid reads it; Received
	// is each invoice's Sydney date of receipt.
	Ledger   *AttentionLedger
	Received map[string]string
	Program  *MonthlyProgram
	Packages []MonthlyPackage
	Delivery []delivery.Item
	// Documents are the project's filings with the date each was received.
	Documents       []MonthlyDocument
	PendingReadings int
	FailedReadings  int
	// Attention is what reports.Attention reads, so this report, chat and
	// the named filters cannot disagree.
	Attention AttentionInput
	// Golden is what the owner's sections read beyond the above; nil on a
	// template without them.
	Golden *GoldenInput
}

// MonthlyPrevious is the last issued project report: its number, its
// reporting date (this period's start) and its frozen sections.
type MonthlyPrevious struct {
	Number   int
	Date     string
	Sections []Section
}

// MonthlyProgram is the Program as the engine computed it on this read.
type MonthlyProgram struct {
	Relative bool
	Cycle    bool
	Complete bool
	// Typical, Low and High are ISO dates on a calendar program, "" on a
	// relative one; the labels are what the Program screen shows.
	Typical, Low, High                string
	TypicalLabel, LowLabel, HighLabel string
	TypicalIndex                      *int
	Known, Total                      int
	Activities                        []MonthlyActivity
}

// MonthlyActivity is one Program row with its computed dates.
type MonthlyActivity struct {
	ID, StageID, Kind, Label string
	// Start and Finish are ISO dates, "" when unknown or relative; the
	// labels are what the screen shows ("Week 12" on a relative program).
	Start, Finish           string
	StartLabel, FinishLabel string
	Critical, Unknown       bool
}

// MonthlyPackage is one live procurement row.
type MonthlyPackage struct {
	ID, Title, Kind, Lifecycle string
	Invited, Quoted, ToCheck   int
	AwardedTo, AwardedOn       string
	// IssuedOn is the latest issue date of the package's RFP or RFT.
	IssuedOn string
}

// MonthlyDocument is one filing: when it arrived and how it is filed.
type MonthlyDocument struct {
	ID, Received, Discipline, Status string
}

// IssueTitle is what a report's issued PDF is titled: "Project report No. 4"
// for the project report, the saved title for any other kind.
func IssueTitle(kind, title string, number int) string {
	if kind == KindMonthly && number > 0 {
		return fmt.Sprintf("%s No. %d", title, number)
	}
	return title
}

// AssembleMonthly builds the draft's sections in the template's order, one
// small function per section, then applies the PM's protected edits.
func AssembleMonthly(in MonthlyInput, sectionTitles map[string]string, edits []Edit) ([]Section, error) {
	var basisErr error
	m := monthly{in: in, err: &basisErr}
	// The owner's format (template version 2): the owner's sections first;
	// the earlier ones follow as the PM's review lists, never issued.
	var front []Section
	if _, ok := sectionTitles["cover"]; ok {
		front = m.goldenSections(sectionTitles)
	}
	order := []struct {
		id    string
		build func() []Block
	}{
		{"glance", m.glance}, {"attention", m.attention}, {"changes", nil}, {"cost", m.cost},
		{"program", m.program}, {"procurement", m.procurement}, {"registers", m.registers}, {"next", m.next},
	}
	out := make([]Section, 0, len(order))
	changes := -1
	for _, s := range order {
		title, ok := sectionTitles[s.id]
		if !ok {
			return nil, fmt.Errorf("report template missing %s", s.id)
		}
		section := Section{ID: s.id, Title: title, Essential: true, Blocks: []Block{}}
		if s.build == nil {
			changes = len(out)
		} else {
			section.Blocks = s.build()
		}
		out = append(out, section)
	}
	// Changes compare this draft's own blocks with the last issue's.
	out[changes].Blocks = m.changes(out)
	if basisErr != nil {
		return nil, basisErr
	}
	if front != nil {
		for i := range out {
			out[i].Internal = true
		}
		out = append(front, out...)
	}
	return ApplyEdits(out, edits)
}

type monthly struct {
	in  MonthlyInput
	err *error
}

func (m monthly) block(id, label, text string, basis map[string]any) Block {
	raw, err := json.Marshal(basis)
	if err != nil && *m.err == nil {
		*m.err = fmt.Errorf("report basis %s: %w", id, err)
	}
	return Block{ID: id, Label: label, Text: text, Origin: "calculation", ReviewStatus: "proposed", Meaning: "stated", Basis: raw}
}

// comment is a section's comment block. Its generated text is Go's sentence
// for the section; its basis holds the section's figures and the period
// start, so a comment written against other figures or another period is
// flagged by the existing conflict rule and holds issue until it is kept,
// rewritten or restored.
func (m monthly) comment(section, text string, figures map[string]any) Block {
	label := "Comment"
	if section == "summary" {
		label = "Summary"
	}
	return m.block("comment:"+section, label, text, map[string]any{"method": "monthly_comment", "section": section, "period_start": m.periodStart(), "figures": figures})
}

func (m monthly) periodStart() string {
	if m.in.Previous == nil {
		return ""
	}
	return m.in.Previous.Date
}

// ---- At a glance

func (m monthly) glance() []Block {
	cost, costFig := m.costSentence()
	prog, progFig := m.programSentence()
	proc := m.procurementSentence()
	shown, more := AttentionShown(Attention(m.in.Attention))
	attention := "Nothing is flagged by SiteWise's records."
	if n := len(shown) + more; n > 0 {
		attention = plural(n, "item needs", "items need") + " attention; they are listed below."
	}
	summary := strings.Join([]string{cost, prog, proc, attention}, " ")
	out := []Block{m.comment("summary", summary, map[string]any{"cost": costFig, "program": progFig, "attention": len(shown) + more})}
	period := fmt.Sprintf("To date, as at %s. This is the first project report.", LongDate(m.in.AsAt))
	if p := m.in.Previous; p != nil {
		period = fmt.Sprintf("Since report %d (%s) to %s.", p.Number, LongDate(p.Date), LongDate(m.in.AsAt))
	}
	out = append(out, m.block("glance:period", "Period", period, map[string]any{"method": "monthly_period", "start": m.periodStart(), "as_at": m.in.AsAt}))
	if m.in.PendingReadings > 0 || m.in.FailedReadings > 0 {
		var parts []string
		if n := m.in.PendingReadings; n > 0 {
			parts = append(parts, plural(n, "document is", "documents are")+" still being read")
		}
		if n := m.in.FailedReadings; n > 0 {
			parts = append(parts, plural(n, "document", "documents")+" could not be read")
		}
		out = append(out, m.block("glance:reading", "Documents", asSentence(strings.Join(parts, "; "))+" Figures from them are not in this report yet.",
			map[string]any{"method": "saved_reading_status", "pending": m.in.PendingReadings, "failed": m.in.FailedReadings}))
	}
	return out
}

// ---- Needs attention

func (m monthly) attention() []Block {
	shown, more := AttentionShown(Attention(m.in.Attention))
	if len(shown) == 0 {
		return []Block{m.block("attention:none", "", "Nothing is flagged by SiteWise's records. This is a list of explicit states, not a judgement of risk.", map[string]any{"method": "attention_rules"})}
	}
	var out []Block
	seen := map[string]bool{}
	for _, it := range shown {
		subject := "all"
		if len(it.IDs) == 1 {
			subject = it.IDs[0]
		}
		id := "attention:" + it.Rule + ":" + subject
		for n := 2; seen[id]; n++ {
			id = fmt.Sprintf("attention:%s:%s:%d", it.Rule, subject, n)
		}
		seen[id] = true
		text := it.Sentence(func(f AttentionFigure) string { return figures.Dollars(string(f.Amount)) }) + "."
		out = append(out, m.block(id, it.Source, text, map[string]any{"method": "attention_rules", "rule": it.Rule, "count": it.Count, "ids": it.IDs, "figures": it.Figures}))
	}
	if more > 0 {
		out = append(out, m.block("attention:more", "", fmt.Sprintf("And %d more in SiteWise.", more), map[string]any{"method": "attention_rules", "more": more}))
	}
	return out
}

// ---- Cost

// costSections are the grid's rows in its order, then Unallocated and Total.
func (m monthly) cost() []Block {
	t := m.in.Costs
	if t == nil {
		return []Block{
			m.block("cost:none", "", "No cost plan yet.", map[string]any{"method": methodCost}),
			m.comment("cost", "No cost plan yet.", nil),
		}
	}
	tax := "ex GST"
	if t.TaxBasis == "inc_tax" {
		tax = "inc GST"
	}
	l := m.in.Ledger
	claimed := func(section string) string {
		if l == nil {
			return "No invoices"
		}
		return figures.Dollars(string(centsMoney(l.Summary.Sections[section])))
	}
	remaining := func(section string) string {
		if l == nil {
			return "Not known"
		}
		if !l.Summary.Comparable {
			return "Not known"
		}
		if v, ok := l.Summary.SectionsContractRemaining[section]; ok {
			return figures.Dollars(string(centsMoney(v)))
		}
		return "Not known"
	}
	rows := [][]string{}
	fig := map[string]any{}
	for _, section := range CostSectionOrder {
		s := t.Sections[section]
		rows = append(rows, []string{CostSectionNames[section], TotalText(s["budget"]), TotalText(s["commitment"]), TotalText(s["forecast"]), claimed(section), remaining(section)})
		fig[section] = costFigures(s, l, section)
	}
	if l != nil && l.Summary.Unallocated.Invoiced != 0 {
		rows = append(rows, []string{"Unallocated invoices", "—", "—", "—", figures.Dollars(string(centsMoney(l.Summary.Unallocated.Invoiced))), "—"})
	}
	totalClaimed, totalRemaining := "No invoices", "Not known"
	if l != nil {
		totalClaimed = figures.Dollars(string(centsMoney(l.Summary.Total)))
		if l.Summary.ContractRemaining != nil && l.Summary.Comparable {
			totalRemaining = figures.Dollars(string(centsMoney(*l.Summary.ContractRemaining)))
		}
	}
	rows = append(rows, []string{"Total", TotalText(t.Overall["budget"]), TotalText(t.Overall["commitment"]), TotalText(t.Overall["forecast"]), totalClaimed, totalRemaining})
	fig["total"] = costFigures(t.Overall, l, "")
	note := "Budget, approved contract and forecast final cost are " + tax + ", as the cost plan holds them. Claimed to date is invoiced to date from the invoice ledger, ex GST; remaining is approved contract less invoiced."
	if l != nil && !l.Summary.Comparable {
		note += " The cost plan's amounts include GST, so remaining is not known rather than mixed."
	}
	if unknown := t.Overall["forecast"].Unknown; unknown > 0 {
		note += " " + plural(unknown, "line has", "lines have") + " no forecast, so totals marked known are lower bounds."
	}
	table := m.block("cost:table", "", note, map[string]any{"method": methodCost, "tax_basis": t.TaxBasis, "figures": fig})
	table.Table = &Table{Columns: []string{"Section", "Budget", "Approved contract", "Forecast final cost", "Claimed to date", "Remaining"}, Rows: rows, Display: "document"}
	out := []Block{table}
	if l != nil {
		out = append(out, m.invoicesThisPeriod(*l))
	}
	vtext, vfig := m.variationLine()
	out = append(out, m.block("cost:variations", "", vtext, map[string]any{"method": "variations_register_v1", "figures": vfig}))
	text, f := m.costSentence()
	return append(out, m.comment("cost", text, f))
}

// variationLine is the variation register's totals, as the Variations tab
// shows them: approved, and what the open ones are expected to cost (the
// PM's estimate, else the amount claimed). Unknown stays unknown.
func (m monthly) variationLine() (string, map[string]any) {
	var vs []invoices.Variation
	if m.in.Ledger != nil {
		vs = m.in.Ledger.Variations
	}
	var approved, forecast int64
	nApproved, nOpen, unknown := 0, 0, 0
	for _, v := range vs {
		switch {
		case v.Status == invoices.VariationApproved:
			nApproved++
			if v.Approved != nil {
				approved += *v.Approved
			}
		case v.Open():
			nOpen++
			if f, _ := v.Forecast(); f != nil {
				forecast += *f
			} else {
				unknown++
			}
		}
	}
	fig := map[string]any{"approved": approved, "approved_count": nApproved, "open_count": nOpen, "open_forecast": forecast, "open_unknown": unknown}
	if nApproved == 0 && nOpen == 0 {
		return "No variation is recorded in SiteWise.", fig
	}
	dollars := func(c int64) string { return "$" + invoices.DisplayCents(c) }
	var parts []string
	if nApproved > 0 {
		parts = append(parts, fmt.Sprintf("%d approved, %s", nApproved, dollars(approved)))
	}
	if nOpen > 0 {
		open := fmt.Sprintf("%d open, %s expected", nOpen, dollars(forecast))
		if unknown > 0 {
			open += fmt.Sprintf(" (%d with no amount yet)", unknown)
		}
		parts = append(parts, open)
	}
	return "Variations: " + strings.Join(parts, "; ") + ", ex GST.", fig
}

// costFigures are one row's exact figures, kept in the basis so the next
// report can compare them.
func costFigures(s map[string]costs.Total, l *AttentionLedger, section string) map[string]any {
	out := map[string]any{}
	for _, metric := range []string{"budget", "commitment", "forecast"} {
		t := s[metric]
		out[metric] = exactTotal(t)
		// The known part and the unknown count, so a partly known total
		// still shows what moved without pretending it is the total.
		if t.Lines > 0 && t.Amount == nil && t.Unknown > 0 && t.Unknown < t.Lines {
			out[metric+"_known"] = string(t.KnownSubtotal)
			out[metric+"_unknown"] = t.Unknown
		}
	}
	if l != nil {
		v := l.Summary.Total
		if section != "" {
			v = l.Summary.Sections[section]
		}
		out["claimed"] = string(centsMoney(v))
	}
	return out
}

// exactTotal is a total's amount when it is complete, else nil: an unknown
// is never compared as a number.
func exactTotal(t costs.Total) any {
	if t.Lines == 0 || t.Amount == nil || !t.Complete {
		return nil
	}
	return string(*t.Amount)
}

func (m monthly) invoicesThisPeriod(l AttentionLedger) Block {
	start := m.periodStart()
	n, review := 0, 0
	var ex int64
	for _, inv := range l.Invoices {
		if start != "" && m.in.Received[inv.ID] < start {
			continue
		}
		if !invoices.Counted(inv) {
			continue
		}
		n++
		ex += invoices.CountedEx(inv)
		if l.Status[inv.ID] == invoices.StatusNeedsReview {
			review++
		}
	}
	since := "to date"
	if start != "" {
		since = "since " + LongDate(start)
	}
	text := fmt.Sprintf("No invoices received %s.", since)
	if n > 0 {
		text = fmt.Sprintf("%s received %s, %s ex GST", plural(n, "invoice", "invoices"), since, figures.Dollars(string(centsMoney(ex))))
		if review > 0 {
			text += fmt.Sprintf("; %d %s review", review, map[bool]string{true: "needs", false: "need"}[review == 1])
		}
		text += "."
	}
	return m.block("cost:invoices", "Invoices", text, map[string]any{"method": "monthly_invoices", "since": start, "count": n, "ex_gst": string(centsMoney(ex)), "needs_review": review})
}

// costSentence is the status words (owner decision 7): under or over budget
// only where both totals are known, else "not yet known".
func (m monthly) costSentence() (string, map[string]any) {
	t := m.in.Costs
	if t == nil {
		return "There is no cost plan yet.", nil
	}
	b, f := t.Overall["budget"], t.Overall["forecast"]
	tax := "ex GST"
	if t.TaxBasis == "inc_tax" {
		tax = "inc GST"
	}
	fig := map[string]any{"budget": exactTotal(b), "forecast": exactTotal(f)}
	switch {
	case f.Lines == 0:
		return "The cost plan has no lines in its totals yet.", fig
	case exactTotal(f) == nil:
		return fmt.Sprintf("Forecast final cost is not yet known: %s, and the known part is %s %s.", plural(f.Unknown, "line has no forecast", "lines have no forecast"), figures.Dollars(string(f.KnownSubtotal)), tax), fig
	case exactTotal(b) == nil:
		return fmt.Sprintf("Forecast final cost is %s %s; whether it is within budget is not yet known, because the total budget is not complete.", figures.Dollars(string(*f.Amount)), tax), fig
	}
	diff := costs.Sub(*f.Amount, *b.Amount)
	switch c := costs.Compare(diff, "0"); {
	case c < 0:
		return fmt.Sprintf("Forecast final cost is %s %s, %s under budget.", figures.Dollars(string(*f.Amount)), tax, figures.Dollars(strings.TrimPrefix(string(diff), "-"))), fig
	case c > 0:
		return fmt.Sprintf("Forecast final cost is %s %s, %s over budget.", figures.Dollars(string(*f.Amount)), tax, figures.Dollars(string(diff))), fig
	}
	return fmt.Sprintf("Forecast final cost is %s %s, equal to the budget.", figures.Dollars(string(*f.Amount)), tax), fig
}

// TotalText is a total as the Cost Plan grid shows it: "—" for no lines,
// "Not known" when every line is unknown, the known part and the count of
// unknown lines when some are, a range when the lines hold ranges. Never 0
// for unknown.
func TotalText(t costs.Total) string {
	switch {
	case t.Lines == 0:
		return "—"
	case t.Unknown == t.Lines:
		return "Not known"
	case t.Amount != nil:
		return figures.Dollars(string(*t.Amount))
	case t.Unknown > 0:
		return fmt.Sprintf("%s known; %d unknown", figures.Dollars(string(t.KnownSubtotal)), t.Unknown)
	case t.Low != nil && t.High != nil:
		return figures.Dollars(string(*t.Low)) + "–" + figures.Dollars(string(*t.High))
	}
	return "Not known"
}

// ---- Program

func (m monthly) program() []Block {
	p := m.in.Program
	if p == nil {
		return []Block{m.block("program:none", "", "No Program yet.", map[string]any{"method": methodCompletion}), m.comment("program", "There is no Program yet.", nil)}
	}
	text, fig := m.programSentence()
	milestones := map[string]any{}
	for _, a := range p.Activities {
		if a.Kind == "milestone" || a.Kind == "hold_point" {
			milestones[a.ID] = map[string]any{"label": a.Label, "kind": a.Kind, "date": a.Start, "label_date": a.StartLabel, "unknown": a.Unknown}
		}
	}
	completion := m.block("program:completion", "Completion", completionText(p), map[string]any{"method": methodCompletion, "relative": p.Relative, "cycle": p.Cycle, "complete": p.Complete,
		"typical": p.Typical, "low": p.Low, "high": p.High, "typical_label": p.TypicalLabel, "typical_index": p.TypicalIndex, "known": p.Known, "total": p.Total, "milestones": milestones})
	out := []Block{completion}
	if stages := criticalStages(p); len(stages) > 0 {
		out = append(out, m.block("program:critical", "Critical path", "The critical path runs through "+joinAnd(stages)+".", map[string]any{"method": "program_critical", "stages": stages}))
	}
	if rows, more := m.milestoneRows(p); len(rows) > 0 {
		b := m.block("program:milestones", "Milestones and hold points", fmt.Sprintf("Due in the next %d days or moved since the last report.", NextWindowDays), map[string]any{"method": "program_milestones", "more": more})
		b.Table = &Table{Columns: []string{"Milestone", "Planned date", "Since last report"}, Rows: rows, Display: "document"}
		if more > 0 {
			b.Text += fmt.Sprintf(" And %d more in SiteWise.", more)
		}
		out = append(out, b)
	}
	out = append(out, m.block("program:basis", "", "Dates are planned dates computed from durations and links; SiteWise records no construction progress.", map[string]any{"method": "program_planned_dates"}))
	return append(out, m.comment("program", text, fig))
}

func completionText(p *MonthlyProgram) string {
	switch {
	case p.Cycle:
		return "The Program's links form a loop, so no dates are computed. Fix the loop on the Program screen."
	case p.TypicalLabel == "":
		return "Completion is not yet known: no activity has a duration."
	}
	text := "Planned completion " + p.TypicalLabel
	if !p.Complete {
		text = "Planned completion at least " + p.TypicalLabel
	}
	if p.LowLabel != "" && p.HighLabel != "" && (p.LowLabel != p.TypicalLabel || p.HighLabel != p.TypicalLabel) {
		text += " (range " + p.LowLabel + " to " + p.HighLabel + ")"
	}
	text += "."
	if !p.Complete {
		text += fmt.Sprintf(" %d of %d activities have a duration; the rest are unknown, so this is a lower bound.", p.Known, p.Total)
	}
	if p.Relative {
		text += " The Program has no start date, so it is counted in weeks."
	}
	return text
}

func (m monthly) programSentence() (string, map[string]any) {
	p := m.in.Program
	if p == nil {
		return "There is no Program yet.", nil
	}
	fig := map[string]any{"typical": p.Typical, "typical_index": p.TypicalIndex, "complete": p.Complete}
	text := completionText(p)
	if c := m.completionMove(); c != "" {
		text += " " + c
	}
	return text, fig
}

// completionMove is "That is 9 days later than report 3." or "".
func (m monthly) completionMove() string {
	prev := m.prevBlock("program:completion")
	if prev == nil || m.in.Program == nil {
		return ""
	}
	days, unit, ok := completionDelta(prev, m.in.Program)
	if !ok {
		return ""
	}
	n := m.in.Previous.Number
	switch {
	case days > 0:
		return fmt.Sprintf("That is %s later than report %d.", plural(days, unit, unit+"s"), n)
	case days < 0:
		return fmt.Sprintf("That is %s earlier than report %d.", plural(-days, unit, unit+"s"), n)
	}
	return fmt.Sprintf("Unchanged since report %d.", n)
}

type completionBasis struct {
	Method       string                    `json:"method"`
	Relative     bool                      `json:"relative"`
	Cycle        bool                      `json:"cycle"`
	Typical      string                    `json:"typical"`
	TypicalLabel string                    `json:"typical_label"`
	TypicalIndex *int                      `json:"typical_index"`
	Milestones   map[string]milestoneBasis `json:"milestones"`
}

type milestoneBasis struct {
	Label     string `json:"label"`
	Date      string `json:"date"`
	LabelDate string `json:"label_date"`
	Unknown   bool   `json:"unknown"`
}

// completionDelta compares the typical finish with a previous report's:
// calendar days on a dated Program, working days on a relative one. Not ok
// when either is unknown or they are of different kinds.
func completionDelta(prev *Block, now *MonthlyProgram) (int, string, bool) {
	var b completionBasis
	if json.Unmarshal(prev.Basis, &b) != nil || b.Method != methodCompletion || b.Cycle || now.Cycle || b.Relative != now.Relative {
		return 0, "", false
	}
	if now.Relative {
		if b.TypicalIndex == nil || now.TypicalIndex == nil {
			return 0, "", false
		}
		return *now.TypicalIndex - *b.TypicalIndex, "working day", true
	}
	d, ok := dayDiff(b.Typical, now.Typical)
	return d, "day", ok
}

func criticalStages(p *MonthlyProgram) []string {
	labels := map[string]string{}
	var order []string
	critical := map[string]bool{}
	for _, a := range p.Activities {
		if a.Kind == "stage" {
			labels[a.ID] = a.Label
			order = append(order, a.ID)
			if a.Critical {
				critical[a.ID] = true
			}
		} else if a.Critical && !a.Unknown {
			critical[a.StageID] = true
		}
	}
	var out []string
	for _, id := range order {
		if critical[id] {
			out = append(out, labels[id])
		}
	}
	return out
}

// milestoneRows are milestones and hold points due in the window or moved
// since the last report, date order, capped.
func (m monthly) milestoneRows(p *MonthlyProgram) ([][]string, int) {
	prev := map[string]milestoneBasis{}
	if b := m.prevBlock("program:completion"); b != nil {
		var cb completionBasis
		if json.Unmarshal(b.Basis, &cb) == nil && cb.Method == methodCompletion {
			prev = cb.Milestones
		}
	}
	end := addDays(m.in.AsAt, NextWindowDays)
	type row struct {
		date  string
		cells []string
	}
	var rows []row
	for _, a := range p.Activities {
		if a.Kind != "milestone" && a.Kind != "hold_point" || a.Unknown {
			continue
		}
		moved := ""
		if old, ok := prev[a.ID]; ok && old.Date != "" && a.Start != "" && old.Date != a.Start {
			if d, ok := dayDiff(old.Date, a.Start); ok {
				moved = movedDays(d)
			}
		}
		due := a.Start != "" && a.Start >= m.in.AsAt && a.Start <= end
		if !due && moved == "" {
			continue
		}
		if moved == "" {
			moved = "Unchanged"
			if _, ok := prev[a.ID]; !ok && m.in.Previous != nil {
				moved = "New"
			}
		}
		rows = append(rows, row{a.Start, []string{a.Label, a.StartLabel, moved}})
	}
	sort.SliceStable(rows, func(i, j int) bool { return rows[i].date < rows[j].date })
	more := 0
	if len(rows) > maxMilestones {
		more = len(rows) - maxMilestones
		rows = rows[:maxMilestones]
	}
	out := make([][]string, len(rows))
	for i, r := range rows {
		out[i] = r.cells
	}
	return out, more
}

func movedDays(d int) string {
	switch {
	case d > 0:
		return plural(d, "day", "days") + " later"
	case d < 0:
		return plural(-d, "day", "days") + " earlier"
	}
	return "Unchanged"
}

// ---- Procurement

type packageBasis struct {
	Title     string `json:"title"`
	Status    string `json:"status"`
	Invited   int    `json:"invited"`
	Quoted    int    `json:"quoted"`
	AwardedTo string `json:"awarded_to"`
	Contract  any    `json:"contract"`
}

func (m monthly) procurement() []Block {
	if len(m.in.Packages) == 0 {
		return []Block{m.block("proc:none", "", "No procurement packages yet.", map[string]any{"method": methodPackages}), m.comment("procurement", "There are no procurement packages yet.", nil)}
	}
	basis := map[string]packageBasis{}
	rows := [][]string{}
	for _, p := range m.in.Packages {
		contract := m.packageContract(p)
		pb := packageBasis{Title: p.Title, Status: packageStatus(p), Invited: p.Invited, Quoted: p.Quoted, AwardedTo: p.AwardedTo, Contract: contract}
		basis[p.ID] = pb
		shownContract := "—"
		if s, ok := contract.(string); ok {
			shownContract = figures.Dollars(s)
		} else if p.AwardedTo != "" {
			shownContract = "Not known"
		}
		awarded := p.AwardedTo
		if awarded == "" {
			awarded = "—"
		}
		rows = append(rows, []string{p.Title, pb.Status, fmt.Sprint(p.Invited + p.Quoted), fmt.Sprint(p.Quoted), awarded, shownContract})
	}
	more := 0
	if len(rows) > maxPackageRows {
		more = len(rows) - maxPackageRows
		rows = rows[:maxPackageRows]
	}
	text := "Firms counts every firm named on the row; quotes are those received. A contract sum is the cost plan's approved contract for an awarded package; no unawarded firm's price is shown."
	if more > 0 {
		text += fmt.Sprintf(" And %d more packages in SiteWise.", more)
	}
	b := m.block("proc:table", "", text, map[string]any{"method": methodPackages, "packages": basis})
	b.Table = &Table{Columns: []string{"Package", "Status", "Firms", "Quotes", "Awarded to", "Contract sum"}, Rows: rows, Display: "document"}
	return []Block{b, m.comment("procurement", m.procurementSentence(), map[string]any{"packages": basis})}
}

// packageContract is the approved contract the cost plan holds for an
// awarded package, exactly, or nil when it is not complete or not awarded.
func (m monthly) packageContract(p MonthlyPackage) any {
	if p.AwardedTo == "" || m.in.PackageCosts == nil {
		return nil
	}
	g, ok := m.in.PackageCosts.Groups[p.ID]
	if !ok {
		return nil
	}
	return exactTotal(g["commitment"])
}

func packageStatus(p MonthlyPackage) string {
	switch {
	case p.AwardedTo != "":
		s := "Awarded"
		if p.AwardedOn != "" {
			s += " " + LongDate(p.AwardedOn)
		}
		return s
	case p.Quoted > 0:
		return plural(p.Quoted, "quote received", "quotes received")
	case p.Invited > 0:
		return "Invited"
	case p.IssuedOn != "":
		return "Issued " + LongDate(p.IssuedOn)
	}
	return map[string]string{"proposed": "Proposed", "planned": "Planned", "procuring": "Procuring", "appointed": "Appointed", "closed": "Closed"}[p.Lifecycle]
}

func (m monthly) procurementSentence() string {
	if len(m.in.Packages) == 0 {
		return "There are no procurement packages yet."
	}
	awarded, quoting, waiting := 0, 0, 0
	for _, p := range m.in.Packages {
		switch {
		case p.AwardedTo != "":
			awarded++
		case p.Quoted > 0:
			quoting++
		case p.Invited > 0:
			waiting++
		}
	}
	return fmt.Sprintf("Of %s, %d %s awarded, %d %s quotes and %d %s waiting for quotes.", plural(len(m.in.Packages), "package", "packages"),
		awarded, isAre(awarded), quoting, hasHave(quoting), waiting, isAre(waiting))
}

// ---- Registers

// registerKinds are the register's kinds the report lists, in the
// register's own order (RFIs, notices and plan 6's consent conditions
// first), with their plural names.
//
// The correspondence kinds (EOT claims, delay notices, letters) are quiet:
// listed when the project has rows, never named in the "No … are
// recorded" line (index rule 5).
var registerKinds = []struct {
	kind, many string
	quiet      bool
}{
	{"rfi", "RFIs", false}, {"eot_claim", "EOT claims", true}, {"delay_notice", "delay notices", true}, {"letter", "letters", true},
	{"notice", "notices", false}, {"condition", "consent conditions", false}, {"action", "actions", false}, {"risk", "risks", false},
	{"issue", "issues", false}, {"decision", "decisions", false}, {"approval", "approvals", false},
}

type registerBasis struct {
	Title  string `json:"title"`
	Status string `json:"status"`
	Open   bool   `json:"open"`
}

func (m monthly) registers() []Block {
	byKind := map[string][]delivery.Item{}
	for _, it := range m.in.Delivery {
		if it.RetiredAt == nil {
			byKind[it.Kind] = append(byKind[it.Kind], it)
		}
	}
	var out []Block
	var none []string
	summary := []string{}
	for _, k := range registerKinds {
		items := byKind[k.kind]
		if len(items) == 0 {
			if !k.quiet {
				none = append(none, k.many)
			}
			continue
		}
		open, overdue := 0, 0
		basis := map[string]registerBasis{}
		var listed []delivery.Item
		for _, it := range items {
			isOpen := delivery.Open(it.Kind, it.Status)
			basis[it.ID] = registerBasis{Title: it.Title, Status: it.Status, Open: isOpen}
			if !isOpen {
				continue
			}
			open++
			if it.TargetDate != nil && *it.TargetDate < m.in.AsAt {
				overdue++
			}
			listed = append(listed, it)
		}
		// Risks oldest first (owner decision 6): no ranking, by target date
		// then title as the delivery read orders them.
		text := fmt.Sprintf("%s open", plural(open, strings.TrimSuffix(k.many, "s"), k.many))
		if k.kind == "rfi" {
			text = fmt.Sprintf("%d open", open)
		}
		if overdue > 0 {
			text += fmt.Sprintf(", %d past the target date", overdue)
		}
		text += fmt.Sprintf(" of %d recorded.", len(items))
		if k.kind == delivery.KindEOTClaim {
			granted := 0
			for _, it := range items {
				granted += delivery.DaysGranted(it.Content)
			}
			text += fmt.Sprintf(" %s granted.", plural(granted, "day", "days"))
		}
		more := 0
		if len(listed) > maxRegisterItems {
			more = len(listed) - maxRegisterItems
			listed = listed[:maxRegisterItems]
		}
		if more > 0 {
			text += fmt.Sprintf(" And %d more open in SiteWise.", more)
		}
		b := m.block("reg:"+k.kind+":summary", capitalise(k.many), text, map[string]any{"method": methodRegister, "kind": k.kind, "items": basis})
		if len(listed) > 0 {
			rows := [][]string{}
			for _, it := range listed {
				target := "—"
				if it.TargetDate != nil {
					target = LongDate(*it.TargetDate)
				}
				if it.Kind == "condition" {
					// A condition's target is its gate; the Program dates it.
					if g := delivery.GateWords[delivery.Condition(it.Content).Stage]; g != "" {
						target = g
					}
				}
				owner := it.OwnerText
				if owner == "" {
					owner = "—"
				}
				rows = append(rows, []string{it.Title, strings.ReplaceAll(it.Status, "_", " "), target, owner})
			}
			b.Table = &Table{Columns: []string{"Item", "Status", "Target", "With"}, Rows: rows, Display: "document"}
		}
		out = append(out, b)
		summary = append(summary, fmt.Sprintf("%d %s open", open, k.many))
	}
	if len(none) > 0 {
		out = append(out, m.block("reg:none", "", "No "+joinOr(none)+" are recorded in SiteWise.", map[string]any{"method": methodRegister, "none": none}))
	}
	text := "No register items are recorded in SiteWise."
	if len(summary) > 0 {
		text = capitalise(joinAnd(summary)) + "."
	}
	return append(out, m.comment("registers", text, map[string]any{"open": summary}))
}

// ---- Next period

func (m monthly) next() []Block {
	end := addDays(m.in.AsAt, NextWindowDays)
	type row struct{ date, what, when, source string }
	var rows []row
	if p := m.in.Program; p != nil && !p.Relative && !p.Cycle {
		for _, a := range p.Activities {
			if a.Kind == "stage" || a.Unknown {
				continue
			}
			if a.Start != "" && a.Start >= m.in.AsAt && a.Start <= end {
				verb := "Starts"
				if a.Kind == "milestone" || a.Kind == "hold_point" {
					verb = "Due"
				}
				rows = append(rows, row{a.Start, a.Label, verb + " " + LongDate(a.Start), "Program"})
			} else if a.Finish != "" && a.Finish >= m.in.AsAt && a.Finish <= end {
				rows = append(rows, row{a.Finish, a.Label, "Finishes " + LongDate(a.Finish), "Program"})
			}
		}
	}
	for _, it := range m.in.Delivery {
		if it.RetiredAt != nil || !delivery.Open(it.Kind, it.Status) || it.TargetDate == nil {
			continue
		}
		if d := *it.TargetDate; d >= m.in.AsAt && d <= end {
			rows = append(rows, row{d, it.Title, "Target " + LongDate(d), "Register, " + it.Kind})
		}
	}
	for _, p := range m.in.Packages {
		if p.AwardedTo == "" && p.Invited > 0 && p.Quoted == 0 {
			rows = append(rows, row{"9999", p.Title, "Quotes awaited", "Procurement"})
		}
	}
	sort.SliceStable(rows, func(i, j int) bool { return rows[i].date < rows[j].date })
	text := fmt.Sprintf("From %s to %s.", LongDate(m.in.AsAt), LongDate(end))
	if p := m.in.Program; p != nil && p.Relative {
		text += " The Program has no start date, so none of its rows is dated in this window."
	}
	if len(rows) == 0 {
		return []Block{m.block("next:none", "", text+" Nothing is dated in this window in SiteWise.", map[string]any{"method": "monthly_next", "from": m.in.AsAt, "to": end}), m.comment("next", "Nothing is dated in the next period.", nil)}
	}
	more := 0
	if len(rows) > maxNext {
		more = len(rows) - maxNext
		rows = rows[:maxNext]
	}
	if more > 0 {
		text += fmt.Sprintf(" And %d more in SiteWise.", more)
	}
	cells := [][]string{}
	for _, r := range rows {
		cells = append(cells, []string{r.what, r.when, r.source})
	}
	b := m.block("next:table", "", text, map[string]any{"method": "monthly_next", "from": m.in.AsAt, "to": end, "more": more})
	b.Table = &Table{Columns: []string{"What", "When", "From"}, Rows: cells, Display: "document"}
	return []Block{b, m.comment("next", fmt.Sprintf("%s in the next %d days.", plural(len(rows)+more, "item is due", "items are due"), NextWindowDays), map[string]any{"count": len(rows) + more})}
}

// ---- Helpers

func (m monthly) prevBlock(id string) *Block {
	if m.in.Previous == nil {
		return nil
	}
	for i := range m.in.Previous.Sections {
		for j := range m.in.Previous.Sections[i].Blocks {
			if b := &m.in.Previous.Sections[i].Blocks[j]; b.ID == id {
				return b
			}
		}
	}
	return nil
}

// LongDate writes an ISO date as "17 November 2026"; anything else as it is.
func LongDate(iso string) string {
	t, err := time.Parse("2006-01-02", iso)
	if err != nil {
		return iso
	}
	return t.Format("2 January 2006")
}

func addDays(iso string, n int) string {
	t, err := time.Parse("2006-01-02", iso)
	if err != nil {
		return iso
	}
	return t.AddDate(0, 0, n).Format("2006-01-02")
}

// dayDiff is b less a in calendar days.
func dayDiff(a, b string) (int, bool) {
	x, e1 := time.Parse("2006-01-02", a)
	y, e2 := time.Parse("2006-01-02", b)
	if e1 != nil || e2 != nil {
		return 0, false
	}
	return int(y.Sub(x).Hours() / 24), true
}

func joinAnd(items []string) string {
	switch len(items) {
	case 0:
		return ""
	case 1:
		return items[0]
	}
	return strings.Join(items[:len(items)-1], ", ") + " and " + items[len(items)-1]
}

func joinOr(items []string) string {
	if len(items) < 2 {
		return joinAnd(items)
	}
	return strings.Join(items[:len(items)-1], ", ") + " or " + items[len(items)-1]
}

func lowerFirst(s string) string {
	if s == "" {
		return s
	}
	return strings.ToLower(s[:1]) + s[1:]
}

func capitalise(s string) string {
	if s == "" {
		return s
	}
	return strings.ToUpper(s[:1]) + s[1:]
}

func asSentence(s string) string { return capitalise(s) + "." }

func isAre(n int) string {
	if n == 1 {
		return "is"
	}
	return "are"
}

func hasHave(n int) string {
	if n == 1 {
		return "has"
	}
	return "have"
}

// CompletionOf is the Program's planned completion for the attention rule,
// or nil when it is not known.
func CompletionOf(p *MonthlyProgram) *AttentionCompletion {
	if p == nil || p.Cycle || p.TypicalLabel == "" {
		return nil
	}
	return &AttentionCompletion{Relative: p.Relative, Date: p.Typical, Index: p.TypicalIndex, Label: p.TypicalLabel}
}

// LastCompletion is the planned completion an issued project report printed,
// read from its frozen program:completion block; nil when it printed none.
func LastCompletion(number int, sections []Section) *AttentionCompletion {
	b := findBlock(sections, "program:completion")
	if b == nil {
		return nil
	}
	var cb completionBasis
	if json.Unmarshal(b.Basis, &cb) != nil || cb.Method != methodCompletion || cb.Cycle || cb.TypicalLabel == "" {
		return nil
	}
	return &AttentionCompletion{Number: number, Relative: cb.Relative, Date: cb.Typical, Index: cb.TypicalIndex, Label: cb.TypicalLabel}
}

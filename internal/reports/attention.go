package reports

import (
	"fmt"
	"sort"
	"strings"

	"sitewise/internal/costs"
	"sitewise/internal/figures"
	"sitewise/internal/invoices"
)

// Attention is the one Go answer to "what needs attention"
// (docs/plans/2026-11-17-monthly-report.md, Task 5; wave rule 2 in
// docs/plans/2026-11-17-next-wave-index.md). Chat's attention read, the
// named filters, the monthly report and the left nav's marks all read it, so
// they cannot disagree. Every rule is an explicit state in a record, never a
// judgement of risk, and a figure that is unknown is never "over".
//
// It is pure: the caller loads the rollups and passes them in. A nil or
// empty part of the input means that part was not loaded or has no rows,
// and its rules say nothing.

// Rule ids. They are stable: a named filter, a nav mark or a report section
// keys on them. Three different things are called "over budget" on the
// screens; each has its own id and its own words so none reads as another.
const (
	RuleImplicationOpen       = "implications.open_critical"
	RuleSectionContractOver   = "cost.section_contract_over_budget"
	RuleForecastOver          = "cost.forecast_over_budget"
	RuleLineClaimedOver       = "cost.line_claimed_over_budget"
	RuleCostNoBudget          = "cost.no_budget"
	RulePackageInvitedNoQuote = "procurement.invited_no_quote"
	RulePackageQuotesToCheck  = "procurement.quotes_to_check"
	RuleInvoiceNeedsReview    = "invoices.needs_review"
	RuleInvoiceUnallocated    = "invoices.unallocated"
	RuleDocumentNotFiled      = "documents.not_filed"
	RuleProgramNoDuration     = "program.no_duration"
	RuleApprovalPastTarget    = "approvals.submitted_past_target"
)

// MaxAttentionShown is how many items a list shows before "and N more".
const MaxAttentionShown = 10

// AttentionInput is what Attention reads, already loaded. Later plans add a
// field and a rule each, and nothing else:
//   - plan 1 (correspondence): messages awaiting a reply;
//   - plan 3 (variations and claims): payment schedules due or past due;
//   - plan 4 (register): open RFIs past their due date;
//   - plan 5 (monthly report): completion later than the last issued report;
//   - plan 6 (consent conditions): conditions due in the window.
type AttentionInput struct {
	// Today is the Sydney date, YYYY-MM-DD, for rules with a due date.
	Today        string
	Implications []AttentionImplication
	// Costs is costs.Summarize(plan, "overall"); nil when there is no plan.
	Costs    *costs.Totals
	Packages []AttentionPackage
	// Ledger is the invoice ledger as the Cost Plan grid reads it; nil when
	// it was not loaded.
	Ledger    *AttentionLedger
	Documents []AttentionDocument
	Program   []AttentionActivity
	// Delivery is the live delivery records (retired ones left out).
	Delivery []AttentionRecord
}

// AttentionImplication is one implication proposal and whether it is decided.
type AttentionImplication struct {
	Key      string
	Label    string
	Rank     int
	Severity string
	Critical bool
	Decided  bool
}

// AttentionPackage is one live procurement package and its firm cells.
type AttentionPackage struct {
	ID    string
	Title string
	Firms []AttentionFirm
}

// AttentionFirm is one firm cell on the procurement board: a quote when
// Quoted, else a firm invited and not yet quoting.
type AttentionFirm struct {
	Quoted      bool
	NeedsReview bool
}

// AttentionLedger is the parts of the invoice ledger the rules read.
type AttentionLedger struct {
	Invoices []invoices.Invoice
	// Status is each invoice's shown status (invoices.Status).
	Status  map[string]string
	Summary invoices.Summary
	// Lines are the budget lines in Summary.Lines' order.
	Lines  []invoices.BudgetLine
	Groups map[string]invoices.GroupSummary
}

// AttentionDocument is one document on the register.
type AttentionDocument struct {
	ID     string
	Status string
}

// AttentionActivity is one Program row.
type AttentionActivity struct {
	ID           string
	Kind         string
	DurationDays *int
}

// AttentionRecord is one live delivery record (risk, approval, action…).
type AttentionRecord struct {
	ID         string
	Kind       string
	Status     string
	TargetDate string
}

// AttentionFigure is one money figure in a sentence, ex GST as its screen
// holds it. Chat tags it; the report prints it.
type AttentionFigure struct {
	Label  string
	Amount costs.Money
}

// AttentionItem is one thing that needs attention.
type AttentionItem struct {
	Rule   string
	Source string
	// Count is the number the sentence states: lines, documents, invoices,
	// firms or quotes; 1 for an item about one thing (a section, an
	// implication).
	Count int
	// IDs are the rows the item is about, in the screen's order: cost line,
	// document, Program row, invoice, package or delivery record ids, an
	// implication's key, or a cost section's key. Empty for the whole plan.
	IDs []string
	// Words is the sentence with a %s for each figure, in order. Use
	// Sentence or Text, not Words, to show it.
	Words   string
	Figures []AttentionFigure
}

// Sentence fills the item's figures with show's form of each.
func (it AttentionItem) Sentence(show func(AttentionFigure) string) string {
	args := make([]any, len(it.Figures))
	for i, f := range it.Figures {
		args[i] = show(f)
	}
	return fmt.Sprintf(it.Words, args...)
}

// Text is the sentence with plain figures (no $, no tags).
func (it AttentionItem) Text() string {
	return it.Sentence(func(f AttentionFigure) string {
		if p, ok := figures.Plain(string(f.Amount)); ok {
			return p
		}
		return string(f.Amount)
	})
}

// AttentionShown is the first MaxAttentionShown items and how many more
// there are, for a list that ends "and N more".
func AttentionShown(items []AttentionItem) ([]AttentionItem, int) {
	if len(items) <= MaxAttentionShown {
		return items, 0
	}
	return items[:MaxAttentionShown], len(items) - MaxAttentionShown
}

// AttentionFor returns the item for a rule that has one item (every rule but the
// per-section, per-package and per-implication ones), or false.
func AttentionFor(items []AttentionItem, rule string) (AttentionItem, bool) {
	for _, it := range items {
		if it.Rule == rule {
			return it, true
		}
	}
	return AttentionItem{}, false
}

// CostSectionOrder and CostSectionNames are the Cost Plan's sections in the
// grid's order and their names.
var (
	CostSectionOrder = []string{"construction", "consultants", "contingency", "authority"}
	CostSectionNames = map[string]string{"authority": "Authority fees", "consultants": "Consultants", "construction": "Construction", "contingency": "Contingency and project-wide"}
)

// Attention returns every item, in a fixed order: implications, cost,
// procurement, invoices, documents, Program, delivery records. It ranks
// nothing beyond that order. Callers cap what they show with AttentionShown.
func Attention(in AttentionInput) []AttentionItem {
	var out []AttentionItem
	out = append(out, implicationItems(in.Implications)...)
	out = append(out, costItems(in.Costs)...)
	if in.Ledger != nil {
		out = appendIf(out, lineClaimedOverItem(*in.Ledger))
	}
	out = append(out, packageItems(in.Packages)...)
	if in.Ledger != nil {
		out = appendIf(out, invoiceReviewItem(*in.Ledger))
		out = appendIf(out, invoiceUnallocatedItem(*in.Ledger))
	}
	out = appendIf(out, documentItem(in.Documents))
	out = appendIf(out, programItem(in.Program))
	out = appendIf(out, approvalItem(in.Delivery, in.Today))
	return out
}

func appendIf(out []AttentionItem, it AttentionItem) []AttentionItem {
	if it.Count > 0 {
		out = append(out, it)
	}
	return out
}

// maxImplications keeps a long draft checklist from filling the list: the
// eight highest ranked open critical or high implications.
const maxImplications = 8

func implicationItems(items []AttentionImplication) []AttentionItem {
	open := append([]AttentionImplication(nil), items...)
	sort.SliceStable(open, func(i, j int) bool { return open[i].Rank < open[j].Rank })
	var out []AttentionItem
	for _, it := range open {
		if it.Decided || len(out) == maxImplications || !(it.Critical || it.Severity == "high") {
			continue
		}
		flag := "severity " + it.Severity
		if it.Critical {
			flag = "critical"
		}
		out = append(out, AttentionItem{Rule: RuleImplicationOpen, Source: "Implications, draft checklist", Count: 1, IDs: []string{it.Key},
			Words: "Open implication, " + esc(flag) + ": " + esc(it.Label)})
	}
	return out
}

// costItems are the Cost Plan's rules from its own totals.
func costItems(t *costs.Totals) []AttentionItem {
	if t == nil {
		return nil
	}
	var out []AttentionItem
	// A section's approved contract above its budget, on known subtotals.
	// A section with no known budget is never over it.
	for _, section := range CostSectionOrder {
		s, ok := t.Sections[section]
		if !ok {
			continue
		}
		b, c := s["budget"], s["commitment"]
		if b.KnownSubtotal == "" || c.KnownSubtotal == "" {
			continue
		}
		if costs.Compare(c.KnownSubtotal, b.KnownSubtotal) > 0 && costs.Compare(b.KnownSubtotal, "0") > 0 {
			name := CostSectionNames[section]
			out = append(out, AttentionItem{Rule: RuleSectionContractOver, Source: "Cost Plan", Count: 1, IDs: []string{section},
				Words:   esc(name) + ": contract %s is over budget %s by %s",
				Figures: []AttentionFigure{{name + ", contract", c.KnownSubtotal}, {name + ", budget", b.KnownSubtotal}, {name + ", contract less budget", costs.Sub(c.KnownSubtotal, b.KnownSubtotal)}}})
		}
	}
	// The forecast final cost above the budget, only when both totals are
	// complete: an unknown or a range on either side says nothing.
	b, f := t.Overall["budget"], t.Overall["forecast"]
	if b.Complete && f.Complete && b.Amount != nil && f.Amount != nil && costs.Compare(*b.Amount, "0") > 0 && costs.Compare(*f.Amount, *b.Amount) > 0 {
		out = append(out, AttentionItem{Rule: RuleForecastOver, Source: "Cost Plan", Count: 1,
			Words:   "Forecast final cost %s is over the total budget %s by %s",
			Figures: []AttentionFigure{{"Forecast final cost", *f.Amount}, {"Total budget", *b.Amount}, {"Forecast less budget", costs.Sub(*f.Amount, *b.Amount)}}})
	}
	if n := len(t.NoBudget); n > 0 {
		out = append(out, AttentionItem{Rule: RuleCostNoBudget, Source: "Cost Plan", Count: n, IDs: append([]string(nil), t.NoBudget...),
			Words: plural(n, "cost line has", "cost lines have") + " no budget amount"})
	}
	return out
}

// lineClaimedOverItem is the rows the Cost Plan tags "over budget": claimed
// to date (invoices, ex GST) above the line's budget, as the ledger flags
// it. A posting line's flag is its own; a group's is its group summary's.
// A line with an unknown budget is never over it. This is not the section
// rule above (contract against budget): which one "over budget" means is an
// open owner question, so both stand, worded apart.
func lineClaimedOverItem(l AttentionLedger) AttentionItem {
	var ids []string
	seen := map[string]bool{}
	for i, s := range l.Summary.Lines {
		if s.Archived || i >= len(l.Lines) {
			continue
		}
		over := s.OverBudget
		if l.Lines[i].Group {
			over = l.Groups[s.ID].OverBudget
		}
		if over && !seen[s.ID] {
			ids = append(ids, s.ID)
			seen[s.ID] = true
		}
	}
	var rest []string
	for id, g := range l.Groups {
		if g.OverBudget && !seen[id] {
			rest = append(rest, id)
		}
	}
	sort.Strings(rest)
	ids = append(ids, rest...)
	return AttentionItem{Rule: RuleLineClaimedOver, Source: "Cost Plan, invoices", Count: len(ids), IDs: ids,
		Words: plural(len(ids), "cost line has", "cost lines have") + " claims to date above the line's budget"}
}

// packageItems are per live package: firms invited with no quote, and
// quotes with readings to check.
func packageItems(pkgs []AttentionPackage) []AttentionItem {
	var out []AttentionItem
	for _, p := range pkgs {
		invited, quoted, review := PackageQuotes(p.Firms)
		if invited > 0 && quoted == 0 {
			out = append(out, AttentionItem{Rule: RulePackageInvitedNoQuote, Source: "Procurement", Count: invited, IDs: []string{p.ID},
				Words: esc(p.Title) + ": " + plural(invited, "firm", "firms") + " invited and no quote received"})
		}
		if review > 0 {
			out = append(out, AttentionItem{Rule: RulePackageQuotesToCheck, Source: "Procurement", Count: review, IDs: []string{p.ID},
				Words: esc(p.Title) + ": " + plural(review, "quote has", "quotes have") + " readings to check"})
		}
	}
	return out
}

// PackageQuotes counts a package's firm cells: firms invited with no quote,
// quotes received, and quotes with readings to check. The no_quote filter
// counts with it too.
func PackageQuotes(firms []AttentionFirm) (invited, quoted, review int) {
	for _, f := range firms {
		if f.Quoted {
			quoted++
		} else {
			invited++
		}
		if f.NeedsReview {
			review++
		}
	}
	return invited, quoted, review
}

func invoiceReviewItem(l AttentionLedger) AttentionItem {
	var ids []string
	for _, inv := range l.Invoices {
		if l.Status[inv.ID] == invoices.StatusNeedsReview {
			ids = append(ids, inv.ID)
		}
	}
	return AttentionItem{Rule: RuleInvoiceNeedsReview, Source: "Invoices", Count: len(ids), IDs: ids,
		Words: plural(len(ids), "invoice needs", "invoices need") + " review"}
}

// invoiceUnallocatedItem is the counted invoices with an amount on no live
// cost line: what the grid's Unallocated row holds.
func invoiceUnallocatedItem(l AttentionLedger) AttentionItem {
	line := map[string]bool{}
	for _, s := range l.Summary.Lines {
		line[s.ID] = true
	}
	var ids []string
	for _, inv := range l.Invoices {
		if !invoices.Counted(inv) {
			continue
		}
		var held int64
		for _, a := range invoices.Shares(inv) {
			if a.LineID != "" && line[a.LineID] && a.Amount != nil {
				held += *a.Amount
			}
		}
		if held != *inv.Ex {
			ids = append(ids, inv.ID)
		}
	}
	return AttentionItem{Rule: RuleInvoiceUnallocated, Source: "Invoices", Count: len(ids), IDs: ids,
		Words:   plural(len(ids), "invoice has", "invoices have") + " amounts on no cost line; the Unallocated row holds %s",
		Figures: []AttentionFigure{{"Unallocated, invoiced to date", centsMoney(l.Summary.Unallocated.Invoiced)}}}
}

// The documents, Program and package sentences keep chat's wording of
// 18 November, plural even for one ("1 documents"): changing chat's words
// is outside this change. Only the cost line count was made singular.

// DocumentNotFiled is a document filing gave up on, with a reason: the
// register's "not filed" row.
func DocumentNotFiled(status string) bool { return status == "not_filed" }

func documentItem(docs []AttentionDocument) AttentionItem {
	var ids []string
	for _, d := range docs {
		if DocumentNotFiled(d.Status) {
			ids = append(ids, d.ID)
		}
	}
	return AttentionItem{Rule: RuleDocumentNotFiled, Source: "Documents", Count: len(ids), IDs: ids,
		Words: plural(len(ids), "document", "documents") + " could not be filed or read"}
}

// ProgramNoDuration is an activity with no duration.
func ProgramNoDuration(kind string, days *int) bool { return kind == "activity" && days == nil }

func programItem(acts []AttentionActivity) AttentionItem {
	var ids []string
	for _, a := range acts {
		if ProgramNoDuration(a.Kind, a.DurationDays) {
			ids = append(ids, a.ID)
		}
	}
	return AttentionItem{Rule: RuleProgramNoDuration, Source: "Program", Count: len(ids), IDs: ids,
		Words: plural(len(ids), "Program row has", "Program rows have") + " no duration, so the finish date is a lower bound"}
}

// approvalItem is approvals submitted, not yet determined, whose target
// date has passed. A record with no target date is never past it.
func approvalItem(records []AttentionRecord, today string) AttentionItem {
	var ids []string
	for _, r := range records {
		if r.Kind == "approval" && r.Status == "submitted" && r.TargetDate != "" && today != "" && r.TargetDate < today {
			ids = append(ids, r.ID)
		}
	}
	return AttentionItem{Rule: RuleApprovalPastTarget, Source: "Delivery records", Count: len(ids), IDs: ids,
		Words: plural(len(ids), "approval is", "approvals are") + " submitted, not determined and past the target date"}
}

// plural is "1 cost line has" or "3 cost lines have".
func plural(n int, one, many string) string {
	if n == 1 {
		return "1 " + one
	}
	return fmt.Sprintf("%d %s", n, many)
}

// esc keeps a record's own words (a title, a label) from being read as a
// format verb in Words.
func esc(s string) string { return strings.ReplaceAll(s, "%", "%%") }

// centsMoney is whole cents as Money, sign kept once.
func centsMoney(c int64) costs.Money {
	sign := ""
	if c < 0 {
		sign, c = "-", -c
	}
	return costs.Money(fmt.Sprintf("%s%d.%02d", sign, c/100, c%100))
}

package invoices

import (
	"fmt"
	"sort"
	"strconv"
	"time"
)

// Stored statuses. Filed and needs review are both "read"; which one shows
// is decided from the ledger at read time, so a flag that depends on another
// invoice (a duplicate) is never stale.
const (
	StatusReading     = "reading"
	StatusFiled       = "filed"
	StatusNeedsReview = "needs_review"
	StatusNotRead     = "not_read"
	StatusVoid        = "void"
	StatusSuperseded  = "superseded"
)

// Allocation is one share of an invoice charged to one cost line. A nil
// LineID is Unallocated. A nil Amount on an invoice's only allocation is the
// whole ex-GST amount; a split states every share.
type Allocation struct {
	ID        string `json:"id"`
	LineID    string `json:"line_id"`
	Amount    *int64 `json:"-"`
	Label     string `json:"-"`
	DecidedBy string `json:"decided_by"`
	Band      string `json:"band"`
}

// Invoice is one ledger row as stored.
type Invoice struct {
	ID             string
	Filename       string
	Status         string
	NotReadReason  string
	Kind           string
	Number         string
	Date           string
	Period         string
	PackageID      string
	ABN            string
	Currency       string
	Ex, GST, Total *int64
	InvoiceCount   int
	SupersededBy   string
	DuplicateOK    bool
	CreatedAt      time.Time
	Allocations    []Allocation
	Decisions      map[string]Decision
}

// BudgetLine is a cost plan line invoices can be charged to. Archived lines
// were removed from the plan but still hold allocations; their spend counts.
type BudgetLine struct {
	ID        string
	Label     string
	Section   string
	Category  string
	PackageID string
	Budget    *int64
	Archived  bool
	// Parent is the awarded group this line belongs to. The group holds the
	// budget; the line's spend rolls up to it.
	Parent string
	// Group is a line whose budget stands for its children's.
	Group bool
}

// Counted reports whether an invoice is in the ledger totals: read, an
// invoice or credit note, in AUD, with a known ex-GST amount.
func Counted(inv Invoice) bool {
	switch inv.Status {
	case StatusVoid, StatusSuperseded, StatusReading, StatusNotRead:
		return false
	}
	return (inv.Kind == "invoice" || inv.Kind == "credit_note") && (inv.Currency == "" || inv.Currency == "AUD") && inv.Ex != nil
}

// Shares are the invoice's amounts per line: one allocation with no amount
// is the whole ex-GST amount.
func Shares(inv Invoice) []Allocation {
	if len(inv.Allocations) == 1 && inv.Allocations[0].Amount == nil && inv.Ex != nil {
		a := inv.Allocations[0]
		v := *inv.Ex
		a.Amount = &v
		return []Allocation{a}
	}
	return inv.Allocations
}

// LineSummary is one cost plan line's spend from the ledger.
type LineSummary struct {
	ID         string `json:"id"`
	Label      string `json:"label"`
	Section    string `json:"section"`
	Category   string `json:"category"`
	Archived   bool   `json:"archived"`
	Budget     *int64 `json:"-"`
	Invoiced   int64  `json:"-"`
	Remaining  *int64 `json:"-"`
	Percent    string `json:"percent_spent"`
	OverBudget bool   `json:"over_budget"`
	Invoices   int    `json:"invoices"`
}

// Summary is the cost plan's view of the ledger. Total is every counted
// invoice's ex GST; Unallocated is Total less what lines hold, so the lines
// and Unallocated always add up to Total.
type Summary struct {
	Lines       []LineSummary
	Unallocated LineSummary
	Total       int64
	// Budget is the sum of the known line budgets; BudgetUnknown counts live
	// lines without one, and while any is unknown the overall remaining is too.
	Budget        int64
	BudgetUnknown int
	Remaining     *int64
	Percent       string
	Counted       int
	Excluded      int
	// Sections is invoiced to date per cost plan section, for subtotal rows.
	Sections map[string]int64
	// Comparable is false when the plan's budgets include GST: invoiced to
	// date stays ex GST and remaining is unknown rather than mixed.
	Comparable bool
}

// Summarize computes the summary from the ledger. taxBasis is the cost
// plan's ("ex_tax" or "inc_tax").
func Summarize(lines []BudgetLine, ledger []Invoice, taxBasis string) Summary {
	s := Summary{Comparable: taxBasis != "inc_tax", Sections: map[string]int64{}}
	index := map[string]int{}
	for _, l := range lines {
		index[l.ID] = len(s.Lines)
		s.Lines = append(s.Lines, LineSummary{ID: l.ID, Label: l.Label, Section: l.Section, Category: l.Category, Archived: l.Archived, Budget: l.Budget})
	}
	s.Unallocated = LineSummary{ID: "", Label: "Unallocated", Section: "unallocated"}
	var held int64
	for _, inv := range ledger {
		if !Counted(inv) {
			s.Excluded++
			continue
		}
		s.Counted++
		s.Total += *inv.Ex
		seen := map[string]bool{}
		for _, a := range Shares(inv) {
			i, ok := index[a.LineID]
			if a.LineID == "" || !ok || a.Amount == nil {
				continue
			}
			s.Lines[i].Invoiced += *a.Amount
			s.Sections[s.Lines[i].Section] += *a.Amount
			held += *a.Amount
			if !seen[a.LineID] {
				s.Lines[i].Invoices++
				seen[a.LineID] = true
			}
		}
		if len(seen) == 0 {
			s.Unallocated.Invoices++
		}
	}
	s.Unallocated.Invoiced = s.Total - held
	// A group's spend is its own and its children's; a child's budget is the
	// group's, so it neither counts nor leaves the total unknown.
	parentOf := map[string]string{}
	for _, l := range lines {
		if l.Parent != "" {
			if _, ok := index[l.Parent]; ok {
				parentOf[l.ID] = l.Parent
			}
		}
	}
	for i := range s.Lines {
		if p, ok := parentOf[s.Lines[i].ID]; ok {
			g := &s.Lines[index[p]]
			g.Invoiced += s.Lines[i].Invoiced
			g.Invoices += s.Lines[i].Invoices
		}
	}
	for i := range s.Lines {
		l := &s.Lines[i]
		if _, child := parentOf[l.ID]; child {
			continue
		}
		if l.Budget == nil {
			if !l.Archived {
				s.BudgetUnknown++
			}
			continue
		}
		s.Budget += *l.Budget
		if s.Comparable {
			r := *l.Budget - l.Invoiced
			l.Remaining = &r
			l.Percent = percent(l.Invoiced, *l.Budget)
			l.OverBudget = l.Invoiced > *l.Budget
		}
	}
	if s.BudgetUnknown == 0 && len(s.Lines) > 0 && s.Comparable {
		r := s.Budget - s.Total
		s.Remaining = &r
		s.Percent = percent(s.Total, s.Budget)
	}
	return s
}

// percent is spent over budget to one decimal place, or "" when the budget
// is zero or negative.
func percent(spent, budget int64) string {
	if budget <= 0 {
		return ""
	}
	tenths := (spent*1000 + budget/2) / budget
	if spent < 0 {
		tenths = (spent*1000 - budget/2) / budget
	}
	sign := ""
	if tenths < 0 {
		sign, tenths = "-", -tenths
	}
	return sign + strconv.FormatInt(tenths/10, 10) + "." + strconv.FormatInt(tenths%10, 10)
}

// Flag is one reconciliation finding. A flag never keeps an invoice out of
// the ledger; Info flags are hints that do not ask for review.
type Flag struct {
	Code   string `json:"code"`
	Reason string `json:"reason"`
	Info   bool   `json:"info,omitempty"`
}

// required are the fields a read invoice needs before it can show as filed.
var required = []string{FieldKind, FieldNumber, FieldDate, FieldSupplier, FieldTotal, FieldGST, FieldLine}

// Check runs every invoice check in code. packages maps a package id to the
// ABN recorded for it; today is the date in Sydney as YYYY-MM-DD.
func Check(ledger []Invoice, packageABN map[string]string, today string) map[string][]Flag {
	out := map[string][]Flag{}
	add := func(id, code, reason string, info bool) {
		out[id] = append(out[id], Flag{Code: code, Reason: reason, Info: info})
	}
	live := func(inv Invoice) bool {
		return inv.Status != StatusVoid && inv.Status != StatusSuperseded && inv.Status != StatusReading
	}
	supplier := func(inv Invoice) string {
		if inv.PackageID != "" {
			return "p:" + inv.PackageID
		}
		if inv.ABN != "" {
			return "a:" + inv.ABN
		}
		return ""
	}
	byNumber := map[string][]Invoice{}
	byLook := map[string][]Invoice{}
	for _, inv := range ledger {
		if !live(inv) || supplier(inv) == "" {
			continue
		}
		if inv.Number != "" {
			k := supplier(inv) + "|" + NormalizeNumber(inv.Number)
			byNumber[k] = append(byNumber[k], inv)
		}
		if inv.Total != nil && inv.Date != "" {
			k := supplier(inv) + "|" + strconv.FormatInt(*inv.Total, 10) + "|" + inv.Date
			byLook[k] = append(byLook[k], inv)
		}
	}
	for _, inv := range ledger {
		id := inv.ID
		switch inv.Status {
		case StatusVoid, StatusSuperseded, StatusReading:
			continue
		case StatusNotRead:
			reason := "Not read: type the fields from the PDF"
			if inv.NotReadReason == NotReadUnreadable {
				reason = "Not read: the PDF's text can't be read; type the fields"
			}
			add(id, "not_read", reason, false)
		}
		if live(inv) && supplier(inv) != "" && !inv.DuplicateOK {
			if inv.Number != "" {
				for _, o := range byNumber[supplier(inv)+"|"+NormalizeNumber(inv.Number)] {
					if o.ID != id {
						add(id, "duplicate", "Possible duplicate of "+o.Number, false)
						break
					}
				}
			}
			if inv.Total != nil && inv.Date != "" {
				for _, o := range byLook[supplier(inv)+"|"+strconv.FormatInt(*inv.Total, 10)+"|"+inv.Date] {
					if o.ID != id && NormalizeNumber(o.Number) != NormalizeNumber(inv.Number) {
						add(id, "look_alike", "Looks like "+orFile(o), false)
						break
					}
				}
			}
		}
		if inv.Ex != nil && inv.GST != nil && inv.Total != nil && abs(*inv.Ex+*inv.GST-*inv.Total) > 1 {
			add(id, "arithmetic", "Totals don't add up", false)
		}
		if inv.Ex != nil && inv.GST != nil {
			switch {
			case *inv.GST == 0 && *inv.Ex != 0:
				add(id, "gst_free", "No GST charged", true)
			case abs(GSTOf(*inv.Ex)-*inv.GST) > 1:
				add(id, "gst_rate", "GST isn't 10%", false)
			}
		}
		if inv.ABN != "" {
			if !ValidABN(inv.ABN) {
				add(id, "abn_invalid", "ABN invalid", false)
			} else if want := packageABN[inv.PackageID]; inv.PackageID != "" && want != "" && want != inv.ABN {
				add(id, "abn_differs", "ABN differs from supplier", false)
			}
		}
		if len(inv.Allocations) > 1 && inv.Ex != nil {
			var sum int64
			for _, a := range inv.Allocations {
				if a.Amount != nil {
					sum += *a.Amount
				}
			}
			if sum != *inv.Ex {
				add(id, "split_total", "Split doesn't add up", false)
			}
		}
		if inv.Date != "" && inv.Period != "" && inv.Date[:7] < inv.Period[:7] {
			add(id, "period", fmt.Sprintf("Dated %s, in %s's period", monthName(inv.Date), monthName(inv.Period)), true)
		}
		if inv.Date != "" && today != "" && inv.Date > today {
			add(id, "odd_date", "Check the date", false)
		}
		if inv.Kind == "credit_note" {
			add(id, "credit_note", "Credit note: reduces the line's spend", true)
		}
		if inv.Kind == "not_invoice" {
			add(id, "not_invoice", "Not an invoice: out of totals until you confirm it", false)
		}
		if inv.Currency != "" && inv.Currency != "AUD" {
			add(id, "currency", "In "+inv.Currency+": out of totals until entered in AUD", false)
		}
		if inv.InvoiceCount > 1 {
			add(id, "many", fmt.Sprintf("May contain %d invoices", inv.InvoiceCount), false)
		}
		if inv.Status != StatusNotRead && inv.Ex == nil {
			add(id, "amount_unknown", "Amount unknown: out of totals", false)
		}
		unmapped := len(inv.Allocations) == 0
		for _, a := range inv.Allocations {
			if a.LineID == "" {
				unmapped = true
			}
		}
		if unmapped && inv.Status != StatusNotRead {
			add(id, "unmapped", "No activity line: sits in Unallocated", false)
		}
		if inv.PackageID == "" && inv.Status != StatusNotRead && inv.Kind != "not_invoice" {
			add(id, "no_supplier", "No matching package: pick the supplier, or add it in Procurement", false)
		}
	}
	return out
}

func orFile(inv Invoice) string {
	if inv.Number != "" {
		return inv.Number
	}
	return inv.Filename
}

func monthName(iso string) string {
	t, err := time.Parse("2006-01-02", iso)
	if err != nil {
		return iso
	}
	return t.Format("January")
}

// Status is what a row shows: stored statuses as they are, and a read
// invoice as filed only when every required field is green or the user's,
// and no check asks for review.
func Status(inv Invoice, flags []Flag) string {
	if inv.Status != StatusFiled {
		return inv.Status
	}
	for _, f := range flags {
		if !f.Info {
			return StatusNeedsReview
		}
	}
	for _, f := range required {
		d, ok := inv.Decisions[f]
		if !ok || d.Value == "" && d.DecidedBy != DecidedByUser {
			return StatusNeedsReview
		}
		if d.DecidedBy != DecidedByUser && d.Band != BandGreen {
			return StatusNeedsReview
		}
	}
	return StatusFiled
}

// SortNewest orders the ledger newest first, the order rows arrive in.
func SortNewest(ledger []Invoice) {
	sort.SliceStable(ledger, func(i, j int) bool {
		if !ledger[i].CreatedAt.Equal(ledger[j].CreatedAt) {
			return ledger[i].CreatedAt.After(ledger[j].CreatedAt)
		}
		return ledger[i].ID > ledger[j].ID
	})
}

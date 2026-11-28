package invoices

import (
	"sort"
	"strings"
	"time"
)

// A payment schedule as one sheet (docs/plans/2026-11-26-payment-schedules.md,
// slice 1): the contract's breakdown with what is complete to date, what was
// certified before and what this claim pays, retention, and the payment
// list. Everything here is worked out from the ledger's own claim figures
// (ClaimLedger), so the sheet, the tab, chat and the report cannot disagree.
// Pure: no store, no Jev, no model.

// Sections of a sheet, in the order they print.
const (
	SectionWorks       = "works"
	SectionProvisional = "provisional"
	SectionVariations  = "variations"
	// SectionUnplaced holds the claim's rows that sit on no line of the
	// contract. They are in the ledger's totals, so they are in the sheet's.
	SectionUnplaced = "unplaced"
)

// Retention bases: what the contract's rate is a percent of.
const (
	RetentionBaseWorks  = "works"   // the works sub total
	RetentionBaseLet    = "let"     // works plus provisional sums
	RetentionBaseToDate = "to_date" // the contract sum to date
)

// BreakdownLine is one line of a contract's breakdown: a works line or a
// provisional sum.
type BreakdownLine struct {
	ID string
	// Key is the ledger key of the claim rows on the line (BreakdownKey).
	// Empty means the line is a cost line and keys as one: a contract whose
	// breakdown has not been stored is its cost lines.
	Key     string
	Section string
	Label   string
	Note    string
	Amount  *int64 // total let, or the provisional sum; nil unknown
	Final   bool   // a provisional sum the PM marked settled at what was completed
	// FinalOn is the day it was marked final (YYYY-MM-DD), when known.
	FinalOn string
	// Retired: taken off the breakdown. It still shows on a schedule while
	// it has certified money or a row, and is no longer part of what is let.
	Retired bool
}

func (l BreakdownLine) key() string {
	if l.Key != "" {
		return l.Key
	}
	return lineKey(l.ID)
}

// TenderPart is one counted line of the awarded firm's tender on the rows a
// contract line holds.
type TenderPart struct {
	Label, Kind string
	Amount      int64
}

// BreakdownFromTender splits a contract line into the firm's own lines:
// base lines are works, provisional sums and prime cost items are
// provisional. The split is kept only when it adds up to the line's
// contract value to the cent; otherwise the line stays whole. One base
// line is the line itself, so it is no split.
func BreakdownFromTender(commitment *int64, parts []TenderPart) ([]BreakdownLine, bool) {
	if commitment == nil || len(parts) == 0 {
		return nil, false
	}
	var sum int64
	out := make([]BreakdownLine, 0, len(parts))
	for _, p := range parts {
		amount := p.Amount
		l := BreakdownLine{Section: SectionWorks, Label: p.Label, Amount: &amount}
		switch p.Kind {
		case "ps", "pc":
			l.Section = SectionProvisional
		case "base":
		default:
			return nil, false
		}
		sum += p.Amount
		out = append(out, l)
	}
	if sum != *commitment || len(out) == 1 && out[0].Section == SectionWorks {
		return nil, false
	}
	return out, true
}

// RetentionTerms are a contract's retention as typed once; a nil rate is
// none typed.
type RetentionTerms struct {
	RateBP *int
	Base   string
}

// SheetClaim is one claim of the contract with what the payment list prints.
type SheetClaim struct {
	ID, Number, Period            string
	InvoiceNo, InvoiceDate        string
	IssuedOn, ReportID, VersionID string // its latest issued schedule; empty when none
	// Received is the day the claim was served (YYYY-MM-DD).
	Received string
	Rows     []ClaimRow
	Figures  ClaimFigures
	// Late: its schedule date passed with none served, so it counts at the
	// claimed amounts.
	Late bool
}

// SheetInput is a contract and its claims; ClaimID is the claim the sheet
// is for.
type SheetInput struct {
	Lines      []BreakdownLine
	Variations []Variation
	Terms      RetentionTerms
	Claims     []SheetClaim
	ClaimID    string
}

// SheetLine is one line of a section. Completed is to date and Prior what
// was certified before this claim; ThisClaim is the difference. Completed
// is nil while the claim's row on the line is not assessed.
type SheetLine struct {
	Key           string
	RowID         string // the claim's row on the line; "" when the claim left it out
	Label, Note   string
	TotalLet      *int64
	Completed     *int64
	Percent       string // Completed of TotalLet to one decimal; "" when either is unknown
	Prior         int64
	ThisClaim     *int64
	ClaimedToDate *int64 // what the contractor claims to date; nil when the claim left it out
	BelowClaimed  bool
	Reason        string
	NotApproved   bool // a claimed variation with no approved sum
	// OverBy is how far Completed is over TotalLet: on a provisional sum
	// the excess needs a variation. Nil when it is not over.
	OverBy *int64
	// Final is a provisional sum marked final; FinalClaimed says a claim
	// received after that moves it all the same.
	Final, FinalClaimed bool
	Retired             bool
}

// SheetSection is one section with its sub totals; a sub total is nil while
// any line's figure is unknown.
type SheetSection struct {
	Kind                           string
	Lines                          []SheetLine
	TotalLet, Completed, ThisClaim *int64
	Prior                          int64
}

// SheetRetention is the contract's retention at this claim. Held is what
// this schedule withholds (a return is negative), as the ledger holds it.
// Limit and Percent need the contract's terms.
type SheetRetention struct {
	// RateBP and BaseKind are the contract's terms, when typed; Base is
	// what the rate is a percent of, when it is known.
	RateBP     *int
	BaseKind   string
	Base       *int64
	Limit      *int64
	HeldBefore int64
	Held       *int64
	Percent    string // this claim's movement as a percent of the base
	HeldToDate *int64
	// Open: the contract has retention terms and this claim has not yet
	// said what it holds or returns. A schedule is not issued while it is.
	Open bool
}

// PaymentRow is one claim in the payment list.
type PaymentRow struct {
	ClaimID, Period, Number string
	InvoiceNo, InvoiceDate  string
	Ex, GST, Inc            *int64
	Current                 bool // the claim the sheet is for
	Unserved                bool // counted at the claimed amount: no schedule was served in time
}

// PaymentTotals are the payment list's totals: the sum of its rows (owner
// decision 3), nil while any row is unknown.
type PaymentTotals struct {
	Ex, GST, Inc *int64
}

// ScheduleSheet is a payment schedule's figures.
type ScheduleSheet struct {
	ClaimID                        string
	Sections                       []SheetSection
	TotalLet, Completed, ThisClaim *int64
	Prior                          int64
	Retention                      SheetRetention
	// Net is the total claim including retention, ex GST.
	Net, GST, Payable *int64
	Payments          []PaymentRow
	Paid              PaymentTotals
	// PaidInFull: every row is assessed and none is below what was claimed.
	PaidInFull bool
	// Notes are what the store has to say about the contract's records,
	// such as a breakdown that does not add up to its cost line.
	Notes []string
}

// ScheduleRef is an issued payment schedule, for the report's attachment.
type ScheduleRef struct {
	ClaimID, Number, Period, IssuedOn, ReportID, VersionID string
	ExGST, GST, IncGST                                     int64
}

// ContractSummary is one contract's header figures: the one rollup the
// tab, chat and the project report read.
type ContractSummary struct {
	PackageID, AwardID, Title, Firm, Kind string
	Original                              *int64 // works plus provisional sums as let; nil while any line is unknown
	Variations                            int64  // approved variations
	Provisional                           int64  // on provisional sums marked final: completed to date less the sum
	TotalLet                              *int64 // Original plus Variations: the sheet's total let
	Adjusted                              *int64 // TotalLet plus Provisional
	Certified                             int64  // completed to date over counted claims, ex GST, before retention
	Retention                             int64  // held now
	Paid                                  int64  // the payment list's total ex GST
	Latest                                *ScheduleRef
}

// BuildSheet lays a claim out on its contract's breakdown. A claim that is
// not one of the contract's gives an empty sheet.
func BuildSheet(in SheetInput) ScheduleSheet {
	claims := certifiedOrder(in.Claims)
	var cur *SheetClaim
	for i := range claims {
		if claims[i].ID == in.ClaimID {
			cur = &claims[i]
		}
	}
	s := ScheduleSheet{ClaimID: in.ClaimID}
	if cur == nil {
		return s
	}
	fig := func(i int) RowFigures {
		if i < len(cur.Figures.Rows) {
			return cur.Figures.Rows[i]
		}
		return RowFigures{}
	}
	before := cur.Figures.Before
	rows := map[string][]int{} // row key -> the claim's rows on it
	s.PaidInFull = cur.Figures.Unassessed == 0
	for i, r := range cur.Rows {
		if fig(i).BelowClaimed {
			s.PaidInFull = false
		}
		if r.Kind != RowRetention {
			rows[rowKey(r)] = append(rows[rowKey(r)], i)
		}
	}
	used := map[string]bool{}
	// place builds a breakdown line from the claim's rows on its key. Where
	// the claim left the line out, what was certified stands.
	place := func(key, label, note string, let *int64) SheetLine {
		used[key] = true
		l := SheetLine{Key: key, Label: label, Note: note, TotalLet: let, Prior: before[key]}
		if len(rows[key]) == 0 {
			done, nothing := l.Prior, int64(0)
			l.Completed, l.ThisClaim = &done, &nothing
		} else {
			l.fill(cur.Rows, fig, rows[key])
		}
		if l.Completed != nil && let != nil {
			l.Percent = percent(*l.Completed, *let)
			if over := *l.Completed - *let; over > 0 {
				l.OverBy = &over
			}
		}
		return l
	}
	works, provisional := SheetSection{Kind: SectionWorks}, SheetSection{Kind: SectionProvisional}
	for _, bl := range in.Lines {
		// A retired line shows only while it has history on this contract.
		if bl.Retired && len(rows[bl.key()]) == 0 && before[bl.key()] == 0 {
			continue
		}
		l := place(bl.key(), bl.Label, bl.Note, bl.Amount)
		l.Final, l.Retired = bl.Final && bl.Section == SectionProvisional, bl.Retired
		// Claimed again: the sum was final before this claim came in, and
		// the claim moves it all the same. The claim it was settled on is not.
		l.FinalClaimed = l.Final && l.ThisClaim != nil && *l.ThisClaim != 0 && bl.FinalOn != "" && cur.Received != "" && bl.FinalOn < cur.Received
		if bl.Section == SectionProvisional {
			provisional.Lines = append(provisional.Lines, l)
		} else {
			works.Lines = append(works.Lines, l)
		}
	}
	variations := SheetSection{Kind: SectionVariations}
	for _, v := range in.Variations {
		key := variationKey(v.ID)
		approved := v.Status == VariationApproved && v.Approved != nil
		if !approved && len(rows[key]) == 0 {
			continue
		}
		l := place(key, strings.TrimSpace(v.Number+" "+v.Title), "", nil)
		if approved {
			l.TotalLet = v.Approved
			if l.Completed != nil {
				l.Percent = percent(*l.Completed, *v.Approved)
				if over := *l.Completed - *v.Approved; over > 0 {
					l.OverBy = &over
				}
			}
		} else {
			l.NotApproved = true
		}
		variations.Lines = append(variations.Lines, l)
	}
	unplaced := SheetSection{Kind: SectionUnplaced}
	for i, r := range cur.Rows {
		if r.Kind == RowRetention || used[rowKey(r)] {
			continue
		}
		l := SheetLine{RowID: r.ID, Label: r.Label}
		l.fill(cur.Rows, fig, []int{i})
		unplaced.Lines = append(unplaced.Lines, l)
	}
	letKnown, doneKnown := true, true
	var let, done, now int64
	for _, sec := range []*SheetSection{&works, &provisional, &variations, &unplaced} {
		if len(sec.Lines) == 0 && sec.Kind != SectionWorks {
			continue
		}
		sec.total()
		// An unplaced row is not part of what was let.
		if sec.Kind != SectionUnplaced {
			if sec.TotalLet == nil {
				letKnown = false
			} else {
				let += *sec.TotalLet
			}
		}
		if sec.Completed == nil {
			doneKnown = false
		} else {
			done, now = done+*sec.Completed, now+*sec.ThisClaim
		}
		s.Prior += sec.Prior
		s.Sections = append(s.Sections, *sec)
	}
	if letKnown {
		s.TotalLet = &let
	}
	if doneKnown {
		s.Completed, s.ThisClaim = &done, &now
	}
	s.Retention = retention(in.Terms, *cur, fig, before[retentionKey], works.TotalLet, provisional.TotalLet, s.TotalLet)
	s.Net, s.GST, s.Payable = cur.Figures.Net, cur.Figures.GST, cur.Figures.Payable
	s.Payments = paymentRows(claims, cur.ID)
	s.Paid = paymentTotals(s.Payments)
	return s
}

// fill takes the line's figures from the claim's rows on it. With one row
// Completed is its assessed to date. Completed is Prior plus what the rows
// schedule, so the sheet always adds up to the ledger's gross.
func (l *SheetLine) fill(rows []ClaimRow, fig func(int) RowFigures, on []int) {
	l.RowID = rows[on[0]].ID
	var now, claimed int64
	known, claimedKnown := true, true
	for _, i := range on {
		r, f := rows[i], fig(i)
		if f.Scheduled == nil {
			known = false
		} else {
			now += *f.Scheduled
		}
		switch {
		case r.ToDate != nil:
			claimed += *r.ToDate - f.CertifiedBefore
		case f.Claimed != nil:
			claimed += *f.Claimed
		default:
			claimedKnown = false
		}
		l.BelowClaimed = l.BelowClaimed || f.BelowClaimed
		if l.Reason == "" {
			l.Reason = r.Reason
		}
	}
	if known {
		done := l.Prior + now
		l.Completed, l.ThisClaim = &done, &now
	}
	if claimedKnown {
		c := l.Prior + claimed
		l.ClaimedToDate = &c
	}
}

// total sums a section; a sum is nil while any line's figure is unknown.
func (sec *SheetSection) total() {
	letKnown, doneKnown := true, true
	var let, done, now int64
	for _, l := range sec.Lines {
		switch {
		case l.Retired:
			// No longer part of what is let.
		case l.TotalLet == nil:
			letKnown = false
		default:
			let += *l.TotalLet
		}
		if l.Completed == nil {
			doneKnown = false
		} else {
			done, now = done+*l.Completed, now+*l.ThisClaim
		}
		sec.Prior += l.Prior
	}
	if letKnown && sec.Kind != SectionUnplaced {
		sec.TotalLet = &let
	}
	if doneKnown {
		sec.Completed, sec.ThisClaim = &done, &now
	}
}

// retention is what the claim's retention row holds or returns, against the
// contract's terms where they are typed. A claim with no retention row
// moves nothing.
func retention(terms RetentionTerms, c SheetClaim, fig func(int) RowFigures, heldBefore int64, works, provisional, toDate *int64) SheetRetention {
	nothing := int64(0)
	r := SheetRetention{HeldBefore: heldBefore, Held: &nothing, RateBP: terms.RateBP}
	said := false
	for i, row := range c.Rows {
		if row.Kind == RowRetention {
			r.Held = fig(i).Scheduled
			said = r.Held != nil
		}
	}
	r.Open = terms.RateBP != nil && !said
	if terms.RateBP != nil {
		r.BaseKind = terms.Base
	}
	if r.Held != nil {
		to := heldBefore + *r.Held
		r.HeldToDate = &to
	}
	var base *int64
	switch terms.Base {
	case RetentionBaseWorks:
		base = works
	case RetentionBaseLet:
		if works != nil && provisional != nil {
			n := *works + *provisional
			base = &n
		} else if works != nil && provisional == nil {
			base = works
		}
	case RetentionBaseToDate:
		base = toDate
	}
	if terms.RateBP == nil || base == nil || *base <= 0 {
		return r
	}
	limit := (*base*int64(*terms.RateBP) + 5000) / 10000
	r.Base, r.Limit = base, &limit
	if r.Held != nil {
		r.Percent = percent(abs(*r.Held), *base)
	}
	return r
}

// certifiedOrder is the claims in the ledger's certified order.
func certifiedOrder(claims []SheetClaim) []SheetClaim {
	out := append([]SheetClaim(nil), claims...)
	sort.SliceStable(out, func(i, j int) bool { return out[i].Figures.Order < out[j].Figures.Order })
	return out
}

// paymentRows are the counted claims in certified order, up to and
// including upTo, which is listed whether it counts yet or not. With no
// upTo it is every counted claim.
func paymentRows(claims []SheetClaim, upTo string) []PaymentRow {
	var out []PaymentRow
	for _, c := range claims {
		current := upTo != "" && c.ID == upTo
		if !c.Figures.Counts && !current {
			continue
		}
		row := PaymentRow{ClaimID: c.ID, Period: c.Period, Number: c.Number, InvoiceNo: c.InvoiceNo, InvoiceDate: c.InvoiceDate, Current: current}
		if c.Late && c.Figures.Counts {
			ex := c.Figures.Ex
			gst := GSTOf(ex)
			inc := ex + gst
			row.Ex, row.GST, row.Inc, row.Unserved = &ex, &gst, &inc, true
		} else {
			row.Ex, row.GST, row.Inc = c.Figures.Net, c.Figures.GST, c.Figures.Payable
		}
		out = append(out, row)
		if current {
			break
		}
	}
	return out
}

func paymentTotals(rows []PaymentRow) PaymentTotals {
	var ex, gst, inc int64
	for _, r := range rows {
		if r.Ex == nil || r.GST == nil || r.Inc == nil {
			return PaymentTotals{}
		}
		ex, gst, inc = ex+*r.Ex, gst+*r.GST, inc+*r.Inc
	}
	return PaymentTotals{Ex: &ex, GST: &gst, Inc: &inc}
}

// SummariseContract works out a contract's header figures. contract is its
// cost lines' contract values: the contract sum, approved variations,
// certified and retention are ContractTotals' and ContractClaims' over
// them, so no screen's figure moves. The breakdown supplies only the
// provisional sums marked final.
func SummariseContract(contract []*int64, lines []BreakdownLine, vs []Variation, claims []SheetClaim) ContractSummary {
	claims = certifiedOrder(claims)
	var c ContractSummary
	sum, approved, toDate := ContractTotals(contract, vs)
	c.Original, c.Variations, c.TotalLet = sum, approved, toDate
	figures := make([]ClaimFigures, 0, len(claims))
	var certified map[string]int64 // to date by row key, after the last counted claim
	for _, cl := range claims {
		figures = append(figures, cl.Figures)
		if cl.Figures.Counts {
			certified = cl.Figures.After
		}
		if cl.VersionID != "" {
			ref := ScheduleRef{ClaimID: cl.ID, Number: cl.Number, Period: cl.Period, IssuedOn: cl.IssuedOn, ReportID: cl.ReportID, VersionID: cl.VersionID}
			if f := cl.Figures; f.Net != nil && f.GST != nil && f.Payable != nil {
				ref.ExGST, ref.GST, ref.IncGST = *f.Net, *f.GST, *f.Payable
			}
			c.Latest = &ref
		}
	}
	c.Certified, c.Retention = ContractClaims(figures)
	for _, l := range lines {
		if l.Section == SectionProvisional && l.Final && !l.Retired && l.Amount != nil {
			c.Provisional += certified[l.key()] - *l.Amount
		}
	}
	if toDate != nil {
		adjusted := *toDate + c.Provisional
		c.Adjusted = &adjusted
	}
	for _, r := range paymentRows(claims, "") {
		if r.Ex != nil {
			c.Paid += *r.Ex
		}
	}
	return c
}

// PeriodLabel is a date's claim period as a schedule prints it: AUG26. Go
// prints it from the date, so a period cannot be mistyped.
func PeriodLabel(date string) string {
	t, err := time.Parse("2006-01-02", date)
	if err != nil {
		return ""
	}
	return strings.ToUpper(t.Format("Jan")) + t.Format("06")
}

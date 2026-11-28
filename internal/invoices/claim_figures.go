package invoices

import (
	"fmt"
	"maps"
	"sort"
	"time"
)

// A claim's figures (plan Tasks 6 and 7), all worked out here. Per row the
// schedule pays the PM's assessed to date less what SiteWise certified
// before; the claim counts in Claimed to Date once its schedule is issued,
// at those scheduled amounts ex GST and before retention (owner decision
// 3). A claim whose schedule date has passed unserved counts at the
// claimed amounts instead. Retention changes what is paid, never the
// certified cost.

// ClaimInput is one claim as the ledger holds it.
type ClaimInput struct {
	ID        string
	PackageID string
	Created   time.Time
	Rows      []ClaimRow
	// Issued is a payment schedule issued for it; IssuedAt orders the
	// certified history.
	Issued   bool
	IssuedAt time.Time
	// Late: the Act's date for the schedule passed with none served, under
	// a verified row (ScheduleLapsed).
	Late          bool
	ReferenceDate string
	// The builder's printed GST and total, for the arithmetic checks.
	GST, Total *int64
}

// RowFigures are one row's working.
type RowFigures struct {
	CertifiedBefore int64
	Claimed         *int64 // this claim, as printed (or to date less previous)
	Scheduled       *int64 // assessed to date less certified before; nil until assessed
	BelowClaimed    bool
}

// ClaimFigures are one claim's totals and checks.
type ClaimFigures struct {
	Rows []RowFigures
	// Gross is the scheduled contract and variation rows; nil while any is
	// unassessed. Retention is what the schedule withholds this time (a
	// release is negative).
	Gross, Retention, Net, GST, Payable *int64
	ClaimedGross                        *int64
	Unassessed, MissingReasons          int
	Flags                               []Flag
	// Counts says the claim is in Claimed to Date: Shares per line, Ex
	// their sum.
	Counts bool
	Shares []Allocation
	Ex     int64
	// Order is the claim's place in certified order. Before and After are
	// what was certified to date on its contract, by row key, going into
	// the claim and coming out of it: a payment schedule prints every line,
	// and a line the claim left out still has its prior.
	Order         int
	Before, After map[string]int64
}

// lineKey is a cost line's row key, BreakdownKey a breakdown line's and
// variationKey a variation's.
func lineKey(id string) string      { return "l:" + id }
func variationKey(id string) string { return "v:" + id }

// BreakdownKey is the ledger key of rows on a stored breakdown line.
func BreakdownKey(id string) string { return "b:" + id }

const retentionKey = "retention"

func rowKey(r ClaimRow) string {
	switch {
	case r.Kind == RowRetention:
		return retentionKey
	case r.VariationID != "":
		return variationKey(r.VariationID)
	case r.BreakdownID != "":
		return BreakdownKey(r.BreakdownID)
	case r.LineID != "":
		return lineKey(r.LineID)
	}
	return ""
}

// ClaimLedger works out every claim of a project in certified order:
// issued claims by issue time, then the rest. variationLine maps a
// variation to its cost line and variationStatus to its status;
// lineToDate is each line's contract sum to date (nil unknown).
func ClaimLedger(claims []ClaimInput, variationLine, variationStatus map[string]string, lineToDate map[string]*int64) map[string]ClaimFigures {
	ordered := append([]ClaimInput(nil), claims...)
	sort.SliceStable(ordered, func(i, j int) bool {
		a, b := ordered[i], ordered[j]
		if a.Issued != b.Issued {
			return a.Issued
		}
		if a.Issued && !a.IssuedAt.Equal(b.IssuedAt) {
			return a.IssuedAt.Before(b.IssuedAt)
		}
		if !a.Created.Equal(b.Created) {
			return a.Created.Before(b.Created)
		}
		return a.ID < b.ID
	})
	certified := map[string]map[string]int64{} // package -> row key -> certified to date
	references := map[string]int{}
	for _, c := range claims {
		if c.ReferenceDate != "" {
			references[c.PackageID+"|"+c.ReferenceDate]++
		}
	}
	out := map[string]ClaimFigures{}
	for order, c := range ordered {
		if certified[c.PackageID] == nil {
			certified[c.PackageID] = map[string]int64{}
		}
		before := certified[c.PackageID]
		f := claimFigures(c, before, variationLine, variationStatus, lineToDate)
		f.Order, f.Before = order, maps.Clone(before)
		if c.ReferenceDate != "" && references[c.PackageID+"|"+c.ReferenceDate] > 1 {
			f.Flags = append(f.Flags, Flag{Code: "same_reference", Reason: "Another claim on this contract names the same reference date"})
		}
		if f.Counts {
			for i, r := range c.Rows {
				k := rowKey(r)
				if k == "" {
					continue
				}
				switch {
				case c.Late && f.Rows[i].Claimed != nil:
					before[k] = f.Rows[i].CertifiedBefore + *f.Rows[i].Claimed
				case !c.Late && r.Assessed != nil:
					before[k] = *r.Assessed
				}
			}
		}
		f.After = maps.Clone(before)
		out[c.ID] = f
	}
	return out
}

func claimFigures(c ClaimInput, before map[string]int64, variationLine, variationStatus map[string]string, lineToDate map[string]*int64) ClaimFigures {
	f := ClaimFigures{Rows: make([]RowFigures, len(c.Rows))}
	var gross, retention, claimedGross int64
	grossKnown, claimedKnown := true, true
	var printedRows, printedRetention int64
	for i, r := range c.Rows {
		rf := RowFigures{CertifiedBefore: before[rowKey(r)], Claimed: r.Claimed()}
		if r.Assessed != nil {
			s := *r.Assessed - rf.CertifiedBefore
			rf.Scheduled = &s
		}
		if r.ToDate != nil && r.Previous != nil && r.ThisClaim != nil && *r.ToDate-*r.Previous != *r.ThisClaim {
			f.Flags = append(f.Flags, Flag{Code: "row_arithmetic", Reason: fmt.Sprintf("%s: to date less previous is not this claim", r.Label)})
		}
		if r.Previous != nil && *r.Previous != rf.CertifiedBefore && r.Kind != RowRetention {
			f.Flags = append(f.Flags, Flag{Code: "previous_differs", Reason: fmt.Sprintf("%s: the builder's previous (%s) is not SiteWise's certified (%s)", r.Label, DisplayCents(*r.Previous), DisplayCents(rf.CertifiedBefore))})
		}
		switch r.Kind {
		case RowRetention:
			if r.ThisClaim != nil {
				printedRetention = *r.ThisClaim
			}
			if rf.Scheduled != nil {
				retention = *rf.Scheduled
			}
		default:
			if r.ThisClaim != nil {
				printedRows += *r.ThisClaim
			}
			if rf.Claimed != nil {
				claimedGross += *rf.Claimed
			} else {
				claimedKnown = false
			}
			if rf.Scheduled != nil {
				gross += *rf.Scheduled
			} else {
				grossKnown = false
			}
			line := r.LineID
			if r.VariationID != "" {
				line = variationLine[r.VariationID]
				if st := variationStatus[r.VariationID]; st != VariationApproved {
					f.Flags = append(f.Flags, Flag{Code: "unapproved_variation", Reason: fmt.Sprintf("%s: claims a variation that is not approved", r.Label)})
				}
			}
			if limit, ok := lineToDate[line]; ok && limit != nil && r.ToDate != nil && *r.ToDate > *limit {
				f.Flags = append(f.Flags, Flag{Code: "over_contract", Reason: fmt.Sprintf("%s: claimed to date is more than the line's contract sum to date", r.Label)})
			}
		}
		if r.Assessed == nil {
			f.Unassessed++
		} else {
			claimedToDate := r.ToDate
			if claimedToDate == nil && rf.Claimed != nil {
				n := rf.CertifiedBefore + *rf.Claimed
				claimedToDate = &n
			}
			rf.BelowClaimed = claimedToDate != nil && *r.Assessed < *claimedToDate
			if r.Kind == RowRetention {
				// Retention to date is held as a positive amount whatever
				// sign the claim printed; holding more than claimed pays less.
				rf.BelowClaimed = claimedToDate != nil && *r.Assessed > abs(*claimedToDate)
			}
			if rf.BelowClaimed && r.Reason == "" {
				f.MissingReasons++
			}
		}
		f.Rows[i] = rf
	}
	if claimedKnown {
		f.ClaimedGross = &claimedGross
	}
	if c.Total != nil && c.GST != nil && len(c.Rows) > 0 && printedRows != 0 {
		// Retention prints as a negative row or as a positive deduction.
		net := printedRows - abs(printedRetention)
		if abs(net+*c.GST-*c.Total) > 1 && abs(printedRows+printedRetention+*c.GST-*c.Total) > 1 {
			f.Flags = append(f.Flags, Flag{Code: "claim_total", Reason: "Rows less retention plus GST is not the claim's total"})
		}
	}
	if grossKnown {
		g, ret := gross, retention
		net := g - ret
		gst := GSTOf(net)
		if c.GST != nil && *c.GST == 0 {
			gst = 0
		}
		pay := net + gst
		f.Gross, f.Retention, f.Net, f.GST, f.Payable = &g, &ret, &net, &gst, &pay
	}
	f.Counts = c.Issued && !c.Late && grossKnown || c.Late
	if !f.Counts {
		return f
	}
	byLine := map[string]int64{}
	var order []string
	for i, r := range c.Rows {
		if r.Kind == RowRetention {
			continue
		}
		amount := f.Rows[i].Scheduled
		if c.Late {
			amount = f.Rows[i].Claimed
		}
		if amount == nil {
			continue
		}
		line := r.LineID
		if r.VariationID != "" {
			line = variationLine[r.VariationID]
		}
		if _, seen := byLine[line]; !seen {
			order = append(order, line)
		}
		byLine[line] += *amount
		f.Ex += *amount
	}
	for _, line := range order {
		v := byLine[line]
		f.Shares = append(f.Shares, Allocation{LineID: line, Amount: &v, DecidedBy: DecidedByRule, Band: BandGreen})
	}
	return f
}

// ContractClaims are a contract's certified to date (what its counted
// claims add to Claimed to Date) and the retention held (what its counted
// claims' schedules withheld, less releases).
func ContractClaims(claims []ClaimFigures) (certified, retention int64) {
	for _, f := range claims {
		if !f.Counts {
			continue
		}
		certified += f.Ex
		if f.Retention != nil {
			retention += *f.Retention
		}
	}
	return certified, retention
}

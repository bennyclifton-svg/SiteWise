package store

import (
	"context"
	"fmt"
	"sort"

	"github.com/jackc/pgx/v5"

	"sitewise/internal/invoices"
)

// A contract's summary and a claim's payment schedule sheet, from the
// ledger read (docs/plans/2026-11-26-payment-schedules.md, slice 1 and
// section 13). Nothing is read here: the ledger's snapshot already holds
// the contracts, variations and claims, and invoices.BuildSheet and
// invoices.SummariseContract do the arithmetic. The tab, chat and the
// project report all call ContractSummaries, so they cannot disagree.

// ContractSummaries are the live contracts' header figures, in the
// ledger's contract order.
func ContractSummaries(v LedgerView) []invoices.ContractSummary {
	byID := invoicesByID(v)
	out := make([]invoices.ContractSummary, 0, len(v.Contracts))
	for _, c := range v.Contracts {
		in := contractSheet(v, c, byID)
		// The contract sum stays the cost lines' own, as every screen has
		// shown it; the breakdown supplies the provisional sums.
		sums := make([]*int64, 0, len(c.Lines))
		for _, l := range c.Lines {
			sums = append(sums, l.Amount)
		}
		s := invoices.SummariseContract(sums, in.Lines, in.Variations, in.Claims)
		s.PackageID, s.AwardID, s.Title, s.Firm, s.Kind = c.PackageID, c.AwardID, c.Title, c.Firm, c.Kind
		out = append(out, s)
	}
	return out
}

// ClaimSheet is one claim's payment schedule laid out on its contract;
// false when the claim is not on a live contract of this ledger.
func ClaimSheet(v LedgerView, claimID string) (invoices.ScheduleSheet, bool) {
	pkg := ""
	for _, cl := range v.Claims {
		if cl.InvoiceID == claimID {
			pkg = cl.PackageID
		}
	}
	for _, c := range v.Contracts {
		if pkg != "" && c.PackageID == pkg {
			in := contractSheet(v, c, invoicesByID(v))
			in.ClaimID = claimID
			sheet := invoices.BuildSheet(in)
			for _, g := range c.Gaps {
				side, by := "under", -g.Diff
				if g.Diff > 0 {
					side, by = "over", g.Diff
				}
				sheet.Notes = append(sheet.Notes, fmt.Sprintf("Breakdown is $%s %s the contract line %s.", invoices.DisplayCents(by), side, g.Label))
			}
			return sheet, true
		}
	}
	return invoices.ScheduleSheet{}, false
}

// ClaimPrints are the live claims as their schedules print them: each
// claim's number on its contract, its period and its tax invoice.
func ClaimPrints(v LedgerView) map[string]invoices.SheetClaim {
	byID := invoicesByID(v)
	out := map[string]invoices.SheetClaim{}
	for _, c := range v.Contracts {
		for _, sc := range contractSheet(v, c, byID).Claims {
			out[sc.ID] = sc
		}
	}
	return out
}

func invoicesByID(v LedgerView) map[string]invoices.Invoice {
	out := make(map[string]invoices.Invoice, len(v.Invoices))
	for _, inv := range v.Invoices {
		out[inv.ID] = inv
	}
	return out
}

// contractSheet is a contract as the sheet reads it: its breakdown, its
// variations and its claims.
func contractSheet(v LedgerView, c Contract, byID map[string]invoices.Invoice) invoices.SheetInput {
	in := invoices.SheetInput{Terms: invoices.RetentionTerms{RateBP: c.RetentionBP, Base: c.RetentionBase}}
	for _, b := range c.Breakdown {
		in.Lines = append(in.Lines, invoices.BreakdownLine{ID: b.ID, Key: b.key(), Section: b.Section, Label: b.Label, Note: b.Note, Amount: b.Amount, Final: b.Final, FinalOn: b.FinalOn, Retired: b.Retired})
	}
	for _, va := range v.Variations {
		if va.AwardID == c.AwardID {
			in.Variations = append(in.Variations, va)
		}
	}
	var claims []ClaimView
	for _, cl := range v.Claims {
		if cl.PackageID == c.PackageID {
			claims = append(claims, cl)
		}
	}
	numbers := claimNumbers(claims)
	for _, cl := range claims {
		sc := invoices.SheetClaim{ID: cl.InvoiceID, Number: numbers[cl.InvoiceID], Received: cl.Served, Rows: cl.Rows, Figures: cl.Figures, Late: cl.Late}
		// The period is printed from a date, never typed as words: the
		// claim's own period (read, or the month it arrived, and the PM's to
		// correct), else its reference date, else the day it was served.
		for _, date := range []string{byID[cl.InvoiceID].Period, cl.ReferenceDate, cl.Served} {
			if sc.Period = invoices.PeriodLabel(date); sc.Period != "" {
				break
			}
		}
		// The invoice is the linked one; else the claim's own, when it is its
		// own tax invoice; else none yet. Nothing is copied between rows.
		switch {
		case len(cl.SettledBy) > 0:
			tax := byID[cl.SettledBy[0]]
			sc.InvoiceNo, sc.InvoiceDate = tax.Number, tax.Date
		case cl.TaxInvoice:
			own := byID[cl.InvoiceID]
			sc.InvoiceNo, sc.InvoiceDate = own.Number, own.Date
		}
		if n := len(cl.Issued); n > 0 {
			last := cl.Issued[n-1]
			sc.IssuedOn, sc.ReportID, sc.VersionID = last.IssuedOn, last.ReportID, last.VersionID
		}
		in.Claims = append(in.Claims, sc)
	}
	return in
}

// claimNumbers are a contract's claim numbers: the contractor's where the
// claim printed one, else the claim's place on the contract in the order
// served (owner decision 11).
func claimNumbers(claims []ClaimView) map[string]string {
	order := make([]int, len(claims))
	for i := range order {
		order[i] = i
	}
	sort.SliceStable(order, func(a, b int) bool { return claims[order[a]].Served < claims[order[b]].Served })
	out := make(map[string]string, len(claims))
	for place, i := range order {
		if out[claims[i].InvoiceID] = claims[i].Number; claims[i].Number == "" {
			out[claims[i].InvoiceID] = fmt.Sprintf("%02d", place+1)
		}
	}
	return out
}

// ScheduleHeader is what a payment schedule's header, statement and
// approval block print (plan 5.5), every value with where it came from.
// Nothing here is read by Jev or the model.
type ScheduleHeader struct {
	Project, Address string
	Parties
	Contractor, ContractorABN, ContractorEmail string
	ContractDate                               string
	// Act is the Act's name. It is printed only from a row the owner has
	// verified, with no section number.
	Act string
	// Terms are the six party fields as typed, for the form that edits them.
	Terms DeliveryTerms
	// Warnings are company numbers that do not read as an ABN or an ACN.
	Warnings []string
}

// ScheduleHeader reads the header for a claim's schedule. actor is the
// signed-in user, whose email stands for the approver until a name is typed.
func (s *Store) ScheduleHeader(ctx context.Context, org, project, actor string, v LedgerView, claimID string) (ScheduleHeader, error) {
	tx, err := s.pool.BeginTx(ctx, pgx.TxOptions{AccessMode: pgx.ReadOnly})
	if err != nil {
		return ScheduleHeader{}, err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	return s.scheduleHeader(ctx, tx, org, project, actor, v, claimID)
}

func (s *Store) scheduleHeader(ctx context.Context, tx pgx.Tx, org, project, actor string, v LedgerView, claimID string) (ScheduleHeader, error) {
	h := ScheduleHeader{Project: v.Project}
	var orgName, email string
	if err := tx.QueryRow(ctx, `SELECT COALESCE(s.address, ''), o.name, COALESCE((SELECT email FROM users u WHERE u.org_id=p.org_id AND u.id::text=$3), '')
FROM projects p JOIN sites s ON s.org_id=p.org_id AND s.id=p.site_id JOIN orgs o ON o.id=p.org_id
WHERE p.org_id=$1::uuid AND p.id=$2::uuid`, org, project, actor).Scan(&h.Address, &orgName, &email); err != nil {
		return h, err
	}
	client, err := wholeProfileValue(ctx, tx, org, project, "proj.client")
	if err != nil {
		return h, err
	}
	if h.Terms, err = deliveryTerms(ctx, tx, org, project); err != nil {
		return h, err
	}
	h.Parties = h.Terms.Parties(client, orgName, email)
	for _, cl := range v.Claims {
		if cl.InvoiceID != claimID {
			continue
		}
		for _, c := range v.Contracts {
			if c.PackageID == cl.PackageID {
				h.Contractor, h.ContractorABN, h.ContractorEmail, h.ContractDate = c.Firm, c.FirmABN, c.FirmEmail, c.ContractDate
			}
		}
	}
	state, err := projectState(ctx, tx, org, project)
	if err != nil {
		return h, err
	}
	// The Act is named only from a verified row, and on a services contract
	// only when the PM switched it on (owner decision 8).
	actOn := true
	for _, cl := range v.Claims {
		if cl.InvoiceID != claimID {
			continue
		}
		for _, c := range v.Contracts {
			if c.PackageID == cl.PackageID {
				actOn = c.ActOn()
			}
		}
	}
	if cat := s.workCatalog(); cat != nil && state != "" && actOn {
		if a, ok := cat.SOPAct(state); ok && a.Verified {
			h.Act = a.Act
		}
	}
	for _, n := range []string{h.PrincipalABN, h.AssessorCompanyNo} {
		if w := invoices.CompanyNumberCheck(n); w != "" {
			h.Warnings = append(h.Warnings, w)
		}
	}
	return h, nil
}

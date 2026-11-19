package httpapi

import (
	"context"
	"fmt"
	"strings"

	"sitewise/internal/costs"
	"sitewise/internal/invoices"
)

// read(variations): the Variations and claims tab as text, every figure
// Go's and tagged for the model to quote (variations plan, "From chat").
// For each date, the inputs it was worked out from.
func readVariationsArea(ctx context.Context, d Deps, who opActor, project string, r *reader) error {
	v, err := d.Store.InvoiceLedger(ctx, who.OrgID, project, todaySydney())
	if err != nil {
		return err
	}
	if len(v.Contracts) == 0 {
		r.line("Variations and claims: no live contract. Variations and progress claims sit on an awarded contract.")
		return nil
	}
	money := func(p *int64, label string) string {
		if p == nil {
			return "unknown"
		}
		m := costs.Money(invoices.FormatCents(*p))
		return r.fig(plainMoney(&m), dollars(m), label, "Variations and claims")
	}
	r.line("Variations and claims, ex GST. Every amount and date is SiteWise's.")
	for _, c := range v.Contracts {
		amounts := make([]*int64, 0, len(c.Lines))
		for _, l := range c.Lines {
			amounts = append(amounts, l.Amount)
		}
		var vs []invoices.Variation
		for _, va := range v.Variations {
			if va.AwardID == c.AwardID {
				vs = append(vs, va)
			}
		}
		sum, approved, toDate := invoices.ContractTotals(amounts, vs)
		certified, held := invoices.ContractClaims(claimFigures(v, c.PackageID))
		firm := ""
		if c.Firm != "" {
			firm = " (" + c.Firm + ")"
		}
		r.line("Contract %s%s%s:", r.id(c.PackageID), c.Title, firm)
		r.line(" contract sum %s; approved variations %s; contract sum to date %s", money(sum, c.Title+", contract sum"), money(&approved, c.Title+", approved variations"), money(toDate, c.Title+", contract sum to date"))
		if c.Claims() {
			period := func(days *int, basis string) string {
				if days == nil {
					return "not entered"
				}
				return fmt.Sprintf("%d %s days", *days, basis)
			}
			r.line(" certified to date %s; retention held %s; contract periods: payment schedule %s, payment %s", money(&certified, c.Title+", certified to date"), money(&held, c.Title+", retention held"),
				period(c.ScheduleDays, c.ScheduleBasis), period(c.PaymentDays, c.PaymentBasis))
		}
		var lines []string
		for _, l := range c.Lines {
			lines = append(lines, r.id(l.ID)+l.Label)
		}
		r.line(" lines: %s", strings.Join(lines, "; "))
		if len(vs) == 0 {
			r.line(" variations: none recorded")
		}
		for _, va := range vs {
			f, basis := va.Forecast()
			parts := []string{va.Status, "claimed " + money(va.Claimed, va.Number+", claimed")}
			if va.Estimate != nil {
				parts = append(parts, "PM's estimate "+money(va.Estimate, va.Number+", PM's estimate"))
			}
			if va.Approved != nil {
				parts = append(parts, "approved "+money(va.Approved, va.Number+", approved"))
			}
			if f != nil {
				parts = append(parts, "forecast "+money(f, va.Number+", forecast")+" ("+basis+")")
			}
			if va.TimeDays != nil {
				parts = append(parts, fmt.Sprintf("time claimed %d days", *va.TimeDays))
			}
			r.line(" variation %s%s %s · %s", r.id(va.ID), va.Number, va.Title, strings.Join(parts, " · "))
		}
		for _, cl := range v.Claims {
			if cl.PackageID != c.PackageID {
				continue
			}
			fg := cl.Figures
			served := cl.Served
			if !cl.ServedConfirmed {
				served += " (the drop day; to confirm)"
			}
			state := "assessing"
			switch {
			case len(cl.Issued) > 0 && cl.ScheduleServed != "":
				state = "schedule served " + cl.ScheduleServed
			case len(cl.Issued) > 0:
				state = "schedule issued " + cl.Issued[len(cl.Issued)-1].IssuedOn + ", not yet recorded as served"
			case fg.Unassessed == 0:
				state = "assessed, not issued"
			}
			r.line(" claim %s%s · served %s · %s · claimed %s · scheduled %s · payable %s", r.id(cl.InvoiceID), orNone(cl.Number), served, state,
				money(fg.ClaimedGross, "claim "+cl.Number+", claimed"), money(fg.Gross, "claim "+cl.Number+", scheduled"), money(fg.Payable, "claim "+cl.Number+", payable"))
			for _, due := range []struct {
				what string
				d    invoices.DueDate
			}{{"payment schedule due", cl.Schedule}, {"payment due", cl.Payment}} {
				date := due.d.Date
				if date == "" {
					date = "unknown"
				}
				check := ""
				if due.d.ToCheck {
					check = " (to check)"
				}
				r.line("  %s %s%s. Worked out from: %s %s", due.what, date, check, due.d.Words, strings.Join(due.d.Notes, " "))
			}
			for i, row := range cl.Rows {
				rf := fg.Rows[i]
				line := row.Label
				if row.Kind == invoices.RowRetention {
					line = "retention"
				}
				r.line("  row %d %s%s · claimed to date %s · certified before %s · assessed to date %s · this schedule %s%s", i+1, r.id(row.ID), line,
					money(row.ToDate, "claim "+cl.Number+", "+row.Label+", claimed to date"), money(&rf.CertifiedBefore, "claim "+cl.Number+", "+row.Label+", certified before"),
					money(row.Assessed, "claim "+cl.Number+", "+row.Label+", assessed"), money(rf.Scheduled, "claim "+cl.Number+", "+row.Label+", scheduled"), reasonOf(row.Reason))
			}
		}
	}
	return nil
}

func orNone(s string) string {
	if s == "" {
		return "(no number)"
	}
	return s
}

func reasonOf(s string) string {
	if s == "" {
		return ""
	}
	return " · reason: " + s
}

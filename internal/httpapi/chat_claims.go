package httpapi

import (
	"context"
	"fmt"
	"strings"

	"sitewise/internal/costs"
	"sitewise/internal/invoices"
	"sitewise/internal/store"
)

// read(variations): the Variations and Progress claims tabs as text, every figure
// Go's and tagged for the model to quote (variations plan, "From chat").
// For each date, the inputs it was worked out from.
func readVariationsArea(ctx context.Context, d Deps, who opActor, project string, r *reader) error {
	v, err := d.Store.InvoiceLedger(ctx, who.OrgID, project, todaySydney())
	if err != nil {
		return err
	}
	if len(v.Contracts) == 0 {
		r.line("Variations and progress claims: no live contract. Variations and progress claims sit on an awarded contract.")
		return nil
	}
	money := func(p *int64, label string) string {
		if p == nil {
			return "unknown"
		}
		m := costs.Money(invoices.FormatCents(*p))
		return r.fig(plainMoney(&m), dollars(m), label, "Variations and claims")
	}
	r.line("Variations and progress claims, ex GST. Every amount and date is SiteWise's.")
	summaries := store.ContractSummaries(v)
	for i, c := range v.Contracts {
		var vs []invoices.Variation
		for _, va := range v.Variations {
			if va.AwardID == c.AwardID {
				vs = append(vs, va)
			}
		}
		s := summaries[i]
		sum, approved, toDate, certified, held := s.Original, s.Variations, s.TotalLet, s.Certified, s.Retention
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
		// The breakdown, once it is more than the cost lines: what a payment
		// schedule prints, with each provisional sum and its final mark.
		var breakdown []string
		for _, b := range c.Breakdown {
			if !b.Stored || b.Retired {
				continue
			}
			word := b.Label
			if b.Section == invoices.SectionProvisional {
				word += " (provisional sum"
				if b.Final {
					word += ", final"
				}
				word += ")"
			}
			breakdown = append(breakdown, r.id(b.ID)+word+" "+money(b.Amount, c.Title+", "+b.Label+", sum"))
		}
		if len(breakdown) > 0 {
			r.line(" breakdown: %s", strings.Join(breakdown, "; "))
		}
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
			readSheet(r, v, cl, money)
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

// readSheet prints a claim's payment schedule as its sheet shows it: each
// line's total let, percent and dollars complete, less prior and this
// claim; retention; and the payment list. Every figure is the sheet's.
func readSheet(r *reader, v store.LedgerView, cl store.ClaimView, money func(*int64, string) string) {
	sheet, ok := store.ClaimSheet(v, cl.InvoiceID)
	if !ok {
		return
	}
	n := store.ClaimPrints(v)[cl.InvoiceID].Number
	r.line("  schedule %s, as its sheet prints it:", n)
	for _, sec := range sheet.Sections {
		for _, l := range sec.Lines {
			pct := "unknown"
			if l.Percent != "" {
				pct = l.Percent + "%"
			}
			prior := l.Prior
			r.line("   %s %s · total let %s · complete %s, %s · less prior %s · this claim %s%s", sec.Kind, l.Label, money(l.TotalLet, "schedule "+n+", "+l.Label+", total let"), pct,
				money(l.Completed, "schedule "+n+", "+l.Label+", completed"), money(&prior, "schedule "+n+", "+l.Label+", less prior"), money(l.ThisClaim, "schedule "+n+", "+l.Label+", this claim"), reasonOf(l.Reason))
		}
	}
	ret := sheet.Retention
	terms := "no retention terms entered"
	if ret.RateBP != nil {
		terms = fmt.Sprintf("retention %d.%02d%% of %s, limit %s", *ret.RateBP/100, *ret.RateBP%100, ret.BaseKind, money(ret.Limit, "schedule "+n+", retention limit"))
	}
	before := ret.HeldBefore
	r.line("   %s; held before %s; this claim holds %s; held to date %s", terms, money(&before, "schedule "+n+", retention held before"), money(ret.Held, "schedule "+n+", retention this claim"), money(ret.HeldToDate, "schedule "+n+", retention held to date"))
	r.line("   total claim inc retention %s; GST %s; payable %s", money(sheet.Net, "schedule "+n+", net"), money(sheet.GST, "schedule "+n+", GST"), money(sheet.Payable, "schedule "+n+", payable"))
	for _, p := range sheet.Payments {
		invoice := p.InvoiceNo
		if invoice == "" {
			invoice = "awaiting invoice"
		}
		r.line("   payment list %s claim %s · invoice %s · %s ex GST · %s inc GST", p.Period, p.Number, invoice, money(p.Ex, "payment "+p.Number+", ex GST"), money(p.Inc, "payment "+p.Number+", inc GST"))
	}
}

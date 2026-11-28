package httpapi

import (
	"strconv"

	"sitewise/internal/invoices"
	"sitewise/internal/store"
)

// A payment schedule as a sheet, riding on the ledger read
// (docs/plans/2026-11-26-payment-schedules.md, slice 1):
// GET /projects/{id}/invoices?sheet={claim} adds that claim's sheet to the
// reply. Every figure is Go's; nothing here calls Jev or the model.

type sheetLineJSON struct {
	Key           string     `json:"key"`
	RowID         string     `json:"row_id"`
	Label         string     `json:"label"`
	Note          string     `json:"note"`
	TotalLet      moneyField `json:"total_let"`
	Completed     moneyField `json:"completed"`
	Percent       string     `json:"percent"`
	Prior         moneyField `json:"prior"`
	ThisClaim     moneyField `json:"this_claim"`
	ClaimedToDate moneyField `json:"claimed_to_date"`
	BelowClaimed  bool       `json:"below_claimed"`
	Reason        string     `json:"reason"`
	NotApproved   bool       `json:"not_approved"`
	// OverBy is how far completed is over the sum; null when it is not.
	OverBy       moneyField `json:"over_by"`
	Final        bool       `json:"final"`
	FinalClaimed bool       `json:"final_claimed"`
	Retired      bool       `json:"retired"`
}

type sheetSectionJSON struct {
	Kind      string          `json:"kind"`
	Lines     []sheetLineJSON `json:"lines"`
	TotalLet  moneyField      `json:"total_let"`
	Completed moneyField      `json:"completed"`
	Prior     moneyField      `json:"prior"`
	ThisClaim moneyField      `json:"this_claim"`
}

type sheetRetentionJSON struct {
	// The contract's terms, when typed: the rate as a percent ("5"), what it
	// is a percent of (works, let or to_date) and that base in dollars.
	RatePercent string     `json:"rate_percent"`
	BaseKind    string     `json:"base_kind"`
	Base        moneyField `json:"base"`
	// Open: this claim has not yet said what it holds or returns.
	Open       bool       `json:"open"`
	Limit      moneyField `json:"limit"`
	HeldBefore moneyField `json:"held_before"`
	// Held is what this schedule withholds; a return is negative.
	Held       moneyField `json:"held"`
	Percent    string     `json:"percent"`
	HeldToDate moneyField `json:"held_to_date"`
}

type paymentRowJSON struct {
	ClaimID     string     `json:"claim_id"`
	Period      string     `json:"period"`
	Number      string     `json:"number"`
	InvoiceNo   string     `json:"invoice_no"`
	InvoiceDate string     `json:"invoice_date"`
	Ex          moneyField `json:"ex"`
	GST         moneyField `json:"gst"`
	Inc         moneyField `json:"inc"`
	Current     bool       `json:"current"`
	Unserved    bool       `json:"unserved"`
}

type paymentTotalsJSON struct {
	Ex  moneyField `json:"ex"`
	GST moneyField `json:"gst"`
	Inc moneyField `json:"inc"`
}

type sheetJSON struct {
	ClaimID    string             `json:"claim_id"`
	Sections   []sheetSectionJSON `json:"sections"`
	TotalLet   moneyField         `json:"total_let"`
	Completed  moneyField         `json:"completed"`
	Prior      moneyField         `json:"prior"`
	ThisClaim  moneyField         `json:"this_claim"`
	Retention  sheetRetentionJSON `json:"retention"`
	Net        moneyField         `json:"net"`
	GST        moneyField         `json:"gst"`
	Payable    moneyField         `json:"payable"`
	Payments   []paymentRowJSON   `json:"payments"`
	Paid       paymentTotalsJSON  `json:"paid"`
	PaidInFull bool               `json:"paid_in_full"`
	Notes      []string           `json:"notes"`
	// Header is the schedule's parties, contract date, Act line and approver.
	Header *sheetHeaderJSON `json:"header,omitempty"`
}

// sheetHeaderJSON is what the schedule's header prints, each party with
// whether the PM typed it, and the six typed fields for the form.
type sheetHeaderJSON struct {
	Project           string            `json:"project"`
	Address           string            `json:"address"`
	Principal         string            `json:"principal"`
	PrincipalABN      string            `json:"principal_abn"`
	PrincipalEmail    string            `json:"principal_email"`
	PrincipalTyped    bool              `json:"principal_typed"`
	Contractor        string            `json:"contractor"`
	ContractorABN     string            `json:"contractor_abn"`
	ContractorEmail   string            `json:"contractor_email"`
	AssessorOrg       string            `json:"assessor_org"`
	AssessorCompanyNo string            `json:"assessor_company_no"`
	AssessorTyped     bool              `json:"assessor_typed"`
	Approver          string            `json:"approver"`
	ApproverTyped     bool              `json:"approver_typed"`
	ContractDate      string            `json:"contract_date"`
	Act               string            `json:"act"`
	Warnings          []string          `json:"warnings"`
	Typed             map[string]string `json:"typed"`
}

func headerBody(h store.ScheduleHeader) *sheetHeaderJSON {
	text := func(p *string) string {
		if p == nil {
			return ""
		}
		return *p
	}
	return &sheetHeaderJSON{Project: h.Project, Address: h.Address, Principal: h.Principal, PrincipalABN: h.PrincipalABN, PrincipalEmail: h.PrincipalEmail,
		PrincipalTyped: h.PrincipalTyped, Contractor: h.Contractor, ContractorABN: h.ContractorABN, ContractorEmail: h.ContractorEmail,
		AssessorOrg: h.AssessorOrg, AssessorCompanyNo: h.AssessorCompanyNo, AssessorTyped: h.AssessorTyped, Approver: h.Approver, ApproverTyped: h.ApproverTyped,
		ContractDate: h.ContractDate, Act: h.Act, Warnings: append([]string{}, h.Warnings...),
		Typed: map[string]string{"principal_name": text(h.Terms.PrincipalName), "principal_abn": text(h.Terms.PrincipalABN), "principal_email": text(h.Terms.PrincipalEmail),
			"assessor_org": text(h.Terms.AssessorOrg), "assessor_company_no": text(h.Terms.AssessorCompanyNo), "assessor_name": text(h.Terms.AssessorName)}}
}

// ratePercent prints basis points as a percent: 500 is "5", 250 is "2.5".
func ratePercent(bp *int) string {
	if bp == nil {
		return ""
	}
	return strconv.FormatFloat(float64(*bp)/100, 'f', -1, 64)
}

type scheduleRefJSON struct {
	ClaimID   string `json:"claim_id"`
	Number    string `json:"number"`
	Period    string `json:"period"`
	IssuedOn  string `json:"issued_on"`
	ReportID  string `json:"report_id"`
	VersionID string `json:"version_id"`
	ExGST     string `json:"ex_gst"`
	GST       string `json:"gst"`
	IncGST    string `json:"inc_gst"`
}

func scheduleRefBody(r *invoices.ScheduleRef) *scheduleRefJSON {
	if r == nil {
		return nil
	}
	return &scheduleRefJSON{ClaimID: r.ClaimID, Number: r.Number, Period: r.Period, IssuedOn: r.IssuedOn, ReportID: r.ReportID, VersionID: r.VersionID,
		ExGST: invoices.FormatCents(r.ExGST), GST: invoices.FormatCents(r.GST), IncGST: invoices.FormatCents(r.IncGST)}
}

func sheetBody(s invoices.ScheduleSheet) *sheetJSON {
	out := &sheetJSON{ClaimID: s.ClaimID, Sections: []sheetSectionJSON{}, TotalLet: cents(s.TotalLet), Completed: cents(s.Completed), Prior: centsValue(s.Prior),
		ThisClaim: cents(s.ThisClaim), Net: cents(s.Net), GST: cents(s.GST), Payable: cents(s.Payable), Payments: []paymentRowJSON{},
		Paid: paymentTotalsJSON{Ex: cents(s.Paid.Ex), GST: cents(s.Paid.GST), Inc: cents(s.Paid.Inc)}, PaidInFull: s.PaidInFull,
		Retention: sheetRetentionJSON{Limit: cents(s.Retention.Limit), HeldBefore: centsValue(s.Retention.HeldBefore), Held: cents(s.Retention.Held),
			Percent: s.Retention.Percent, HeldToDate: cents(s.Retention.HeldToDate), RatePercent: ratePercent(s.Retention.RateBP),
			BaseKind: s.Retention.BaseKind, Base: cents(s.Retention.Base), Open: s.Retention.Open}}
	for _, sec := range s.Sections {
		j := sheetSectionJSON{Kind: sec.Kind, Lines: []sheetLineJSON{}, TotalLet: cents(sec.TotalLet), Completed: cents(sec.Completed), Prior: centsValue(sec.Prior), ThisClaim: cents(sec.ThisClaim)}
		for _, l := range sec.Lines {
			j.Lines = append(j.Lines, sheetLineJSON{Key: l.Key, RowID: l.RowID, Label: l.Label, Note: l.Note, TotalLet: cents(l.TotalLet), Completed: cents(l.Completed),
				Percent: l.Percent, Prior: centsValue(l.Prior), ThisClaim: cents(l.ThisClaim), ClaimedToDate: cents(l.ClaimedToDate), BelowClaimed: l.BelowClaimed,
				Reason: l.Reason, NotApproved: l.NotApproved, OverBy: cents(l.OverBy), Final: l.Final, FinalClaimed: l.FinalClaimed, Retired: l.Retired})
		}
		out.Sections = append(out.Sections, j)
	}
	out.Notes = append([]string{}, s.Notes...)
	for _, p := range s.Payments {
		out.Payments = append(out.Payments, paymentRowJSON{ClaimID: p.ClaimID, Period: p.Period, Number: p.Number, InvoiceNo: p.InvoiceNo, InvoiceDate: p.InvoiceDate,
			Ex: cents(p.Ex), GST: cents(p.GST), Inc: cents(p.Inc), Current: p.Current, Unserved: p.Unserved})
	}
	return out
}

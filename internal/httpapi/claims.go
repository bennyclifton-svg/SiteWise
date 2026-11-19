package httpapi

import (
	"errors"
	"net/http"
	"strings"

	"sitewise/internal/invoices"
	"sitewise/internal/store"
)

// Progress claims on the Variations and claims tab
// (docs/plans/2026-11-17-variations-progress-claims.md, Tasks 7 and 9). No
// route here calls Jev or the model; every figure in a reply is Go's.

// Budget paths for claims (bench/budgets.json).
const (
	pathClaimAssess          = "claim_assess"
	pathPaymentScheduleIssue = "payment_schedule_issue"
)

// putClaimLines saves an assessment or Accept claimed.
func putClaimLines(w http.ResponseWriter, r *http.Request, deps Deps) {
	session, ok := invoiceSession(w, r, deps)
	if !ok {
		return
	}
	var in store.ClaimAssess
	if readProposalJSON(w, r, deps.MaxBodyBytes, &in) != nil {
		http.Error(w, "invalid body", http.StatusBadRequest)
		return
	}
	for _, row := range in.Rows {
		if !uuidPattern.MatchString(row.ID) {
			http.Error(w, "invalid row", http.StatusBadRequest)
			return
		}
	}
	if err := deps.Store.AssessClaim(r.Context(), session.OrgID, r.PathValue("id"), session.UserID, r.PathValue("inv"), in); err != nil {
		claimError(w, deps, err)
		return
	}
	invoiceWritten(w, r, deps, session)
}

// postPaymentSchedule issues the claim's payment schedule. Only the PM's
// press of Issue calls it: chat opens the claim's panel and never issues.
func postPaymentSchedule(w http.ResponseWriter, r *http.Request, deps Deps) {
	session, ok := invoiceSession(w, r, deps)
	if !ok {
		return
	}
	issued, err := deps.Store.IssuePaymentSchedule(r.Context(), session.OrgID, r.PathValue("id"), session.UserID, r.PathValue("inv"), todaySydney(), deps.Blobs)
	if err != nil {
		claimError(w, deps, err)
		return
	}
	if deps.Broker != nil {
		deps.Broker.Wake(session.OrgID)
	}
	writeJSON(w, http.StatusCreated, issued)
}

// patchAwardPeriods types a contract's own payment schedule and payment
// periods.
func patchAwardPeriods(w http.ResponseWriter, r *http.Request, deps Deps) {
	if !uuidPattern.MatchString(r.PathValue("pkg")) {
		http.NotFound(w, r)
		return
	}
	session, ok := deliverySession(w, r, deps, true)
	if !ok {
		return
	}
	var in store.ContractPeriods
	if readProposalJSON(w, r, deps.MaxBodyBytes, &in) != nil {
		http.Error(w, "invalid body", http.StatusBadRequest)
		return
	}
	if err := deps.Store.SetContractPeriods(r.Context(), session.OrgID, r.PathValue("id"), session.UserID, r.PathValue("pkg"), in); err != nil {
		claimError(w, deps, err)
		return
	}
	invoiceWritten(w, r, deps, session)
}

func claimError(w http.ResponseWriter, deps Deps, err error) {
	switch {
	case errors.Is(err, store.ErrInvalidClaim):
		msg := strings.TrimPrefix(err.Error(), store.ErrInvalidClaim.Error()+": ")
		http.Error(w, strings.ToUpper(msg[:1])+msg[1:]+".", http.StatusUnprocessableEntity)
	case errors.Is(err, store.ErrClaimIssued):
		http.Error(w, "This claim's payment schedule is issued, so its rows can no longer change.", http.StatusConflict)
	default:
		invoiceError(w, deps, err)
	}
}

type claimRowJSON struct {
	ID              string     `json:"id"`
	Ordinal         int        `json:"ordinal"`
	Kind            string     `json:"kind"`
	Label           string     `json:"label"`
	LineID          string     `json:"line_id"`
	Line            string     `json:"line"`
	VariationID     string     `json:"variation_id"`
	Variation       string     `json:"variation"`
	MappedBy        string     `json:"mapped_by"`
	Band            string     `json:"band"`
	ToDate          moneyField `json:"to_date"`
	Previous        moneyField `json:"previous"`
	ThisClaim       moneyField `json:"this_claim"`
	CertifiedBefore moneyField `json:"certified_before"`
	Claimed         moneyField `json:"claimed"`
	Assessed        moneyField `json:"assessed_to_date"`
	Scheduled       moneyField `json:"scheduled"`
	BelowClaimed    bool       `json:"below_claimed"`
	Reason          string     `json:"reason"`
	Version         int64      `json:"version"`
}

type issuedJSON struct {
	store.IssuedSchedule
	File string `json:"file"`
}

type claimJSON struct {
	InvoiceID       string           `json:"invoice_id"`
	PackageID       string           `json:"package_id"`
	Number          string           `json:"number"`
	Served          string           `json:"served_on"`
	ServedConfirmed bool             `json:"served_confirmed"`
	ReferenceDate   string           `json:"reference_date"`
	ScheduleServed  string           `json:"schedule_served_on"`
	Claimed         moneyField       `json:"claimed"`
	Scheduled       moneyField       `json:"scheduled"`
	Retention       moneyField       `json:"retention"`
	Net             moneyField       `json:"net"`
	GST             moneyField       `json:"gst"`
	Payable         moneyField       `json:"payable"`
	Unassessed      int              `json:"unassessed"`
	MissingReasons  int              `json:"missing_reasons"`
	Counts          bool             `json:"counts"`
	Late            bool             `json:"late"`
	Alert           string           `json:"alert"`
	ScheduleDue     invoices.DueDate `json:"schedule_due"`
	PaymentDue      invoices.DueDate `json:"payment_due"`
	Issued          []issuedJSON     `json:"issued"`
	SettledBy       []string         `json:"settled_by"`
	Flags           []invoices.Flag  `json:"flags"`
	Rows            []claimRowJSON   `json:"rows"`
}

// claimFigures are one contract's claims' figures.
func claimFigures(v store.LedgerView, pkg string) []invoices.ClaimFigures {
	var out []invoices.ClaimFigures
	for _, c := range v.Claims {
		if c.PackageID == pkg {
			out = append(out, c.Figures)
		}
	}
	return out
}

func claimsBody(v store.LedgerView) []claimJSON {
	lines, vars := map[string]string{}, map[string]string{}
	for _, c := range v.Contracts {
		for _, l := range c.Lines {
			lines[l.ID] = l.Label
		}
	}
	for _, va := range v.Variations {
		vars[va.ID] = va.Number + " " + va.Title
	}
	out := []claimJSON{}
	for _, c := range v.Claims {
		f := c.Figures
		j := claimJSON{InvoiceID: c.InvoiceID, PackageID: c.PackageID, Number: c.Number, Served: c.Served, ServedConfirmed: c.ServedConfirmed,
			ReferenceDate: c.ReferenceDate, ScheduleServed: c.ScheduleServed, Claimed: cents(f.ClaimedGross), Scheduled: cents(f.Gross),
			Retention: cents(f.Retention), Net: cents(f.Net), GST: cents(f.GST), Payable: cents(f.Payable), Unassessed: f.Unassessed,
			MissingReasons: f.MissingReasons, Counts: f.Counts, Late: c.Late, Alert: c.Alert, ScheduleDue: c.Schedule, PaymentDue: c.Payment,
			Issued: []issuedJSON{}, SettledBy: append([]string{}, c.SettledBy...), Flags: append([]invoices.Flag{}, f.Flags...), Rows: []claimRowJSON{}}
		if j.ScheduleDue.Notes == nil {
			j.ScheduleDue.Notes = []string{}
		}
		if j.PaymentDue.Notes == nil {
			j.PaymentDue.Notes = []string{}
		}
		for _, is := range c.Issued {
			j.Issued = append(j.Issued, issuedJSON{IssuedSchedule: is, File: "/api/reports/" + is.ReportID + "/versions/" + is.VersionID + "/file"})
		}
		for i, r := range c.Rows {
			rf := invoices.RowFigures{}
			if i < len(f.Rows) {
				rf = f.Rows[i]
			}
			before := rf.CertifiedBefore
			j.Rows = append(j.Rows, claimRowJSON{ID: r.ID, Ordinal: r.Ordinal, Kind: r.Kind, Label: r.Label, LineID: r.LineID, Line: lines[r.LineID],
				VariationID: r.VariationID, Variation: vars[r.VariationID], MappedBy: r.MappedBy, Band: r.Band, ToDate: cents(r.ToDate),
				Previous: cents(r.Previous), ThisClaim: cents(r.ThisClaim), CertifiedBefore: cents(&before), Claimed: cents(rf.Claimed),
				Assessed: cents(r.Assessed), Scheduled: cents(rf.Scheduled), BelowClaimed: rf.BelowClaimed, Reason: r.Reason, Version: r.Version})
		}
		out = append(out, j)
	}
	return out
}

package httpapi

import (
	"encoding/json"
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
		// A row is named by its id, or by the line, variation or retention
		// the claim left out.
		named := row.Retention || row.Line != nil && uuidPattern.MatchString(*row.Line) || row.Variation != nil && uuidPattern.MatchString(*row.Variation)
		if row.ID == "" && !named || row.ID != "" && !uuidPattern.MatchString(row.ID) {
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
	// The body is the contract's periods, its terms (date and retention), or
	// both. A group that is left out is not touched.
	var raw map[string]json.RawMessage
	if readProposalJSON(w, r, deps.MaxBodyBytes, &raw) != nil {
		http.Error(w, "invalid body", http.StatusBadRequest)
		return
	}
	periods, terms := false, false
	for k := range raw {
		switch k {
		case "schedule_days", "schedule_basis", "payment_days", "payment_basis":
			periods = true
		case "contract_date", "retention_percent", "retention_base", "act_applies":
			terms = true
		default:
			http.Error(w, "invalid body", http.StatusBadRequest)
			return
		}
	}
	whole, _ := json.Marshal(raw)
	var p store.ContractPeriods
	var t store.ContractTerms
	if !periods && !terms || json.Unmarshal(whole, &p) != nil || json.Unmarshal(whole, &t) != nil {
		http.Error(w, "invalid body", http.StatusBadRequest)
		return
	}
	if periods {
		if err := deps.Store.SetContractPeriods(r.Context(), session.OrgID, r.PathValue("id"), session.UserID, r.PathValue("pkg"), p); err != nil {
			claimError(w, deps, err)
			return
		}
	}
	if terms {
		if err := deps.Store.SetContractTerms(r.Context(), session.OrgID, r.PathValue("id"), session.UserID, r.PathValue("pkg"), t); err != nil {
			claimError(w, deps, err)
			return
		}
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

type claimSendJSON struct {
	MessageID string `json:"message_id"`
	Status    string `json:"status"`
	SentOn    string `json:"sent_on"`
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
	// As the schedule prints the claim: its number on the contract, its
	// period (AUG26) and its tax invoice.
	// Note is the PM's own words on the schedule. RetentionOpen: the
	// contract has retention terms and this claim has not said what it holds.
	Note          string `json:"schedule_note"`
	RetentionOpen bool   `json:"retention_open"`
	// ScheduleServedConfirmed: the PM typed or confirmed the day served.
	// Send is the message the issued schedule last went out on.
	ScheduleServedConfirmed bool `json:"schedule_served_confirmed"`
	// TaxInvoice: the claim is its own tax invoice. AwaitingInvoice: no
	// invoice is linked to it yet.
	TaxInvoice      bool           `json:"tax_invoice"`
	AwaitingInvoice bool           `json:"awaiting_invoice"`
	Send            *claimSendJSON `json:"schedule_send,omitempty"`
	DisplayNumber   string         `json:"display_number"`
	Period          string         `json:"period"`
	InvoiceNo       string         `json:"invoice_no"`
	InvoiceDate     string         `json:"invoice_date"`
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
	prints := store.ClaimPrints(v)
	terms := map[string]bool{} // contracts with retention terms
	for _, c := range v.Contracts {
		terms[c.PackageID] = c.RetentionBP != nil
	}
	for _, c := range v.Claims {
		f := c.Figures
		j := claimJSON{InvoiceID: c.InvoiceID, PackageID: c.PackageID, Number: c.Number, Served: c.Served, ServedConfirmed: c.ServedConfirmed,
			ReferenceDate: c.ReferenceDate, ScheduleServed: c.ScheduleServed, Claimed: cents(f.ClaimedGross), Scheduled: cents(f.Gross),
			Retention: cents(f.Retention), Net: cents(f.Net), GST: cents(f.GST), Payable: cents(f.Payable), Unassessed: f.Unassessed,
			MissingReasons: f.MissingReasons, Counts: f.Counts, Late: c.Late, Alert: c.Alert, ScheduleDue: c.Schedule, PaymentDue: c.Payment,
			Issued: []issuedJSON{}, SettledBy: append([]string{}, c.SettledBy...), Flags: append([]invoices.Flag{}, f.Flags...), Rows: []claimRowJSON{}}
		j.DisplayNumber, j.Note = c.Number, c.Note
		j.ScheduleServedConfirmed = c.ScheduleServedConfirmed
		j.TaxInvoice, j.AwaitingInvoice = c.TaxInvoice, c.AwaitingInvoice
		if c.Send != nil {
			j.Send = &claimSendJSON{MessageID: c.Send.MessageID, Status: c.Send.Status, SentOn: c.Send.SentOn}
		}
		if terms[c.PackageID] {
			j.RetentionOpen = true
			for _, r := range c.Rows {
				if r.Kind == invoices.RowRetention && r.Assessed != nil {
					j.RetentionOpen = false
				}
			}
		}
		if p, ok := prints[c.InvoiceID]; ok {
			j.DisplayNumber, j.Period, j.InvoiceNo, j.InvoiceDate = p.Number, p.Period, p.InvoiceNo, p.InvoiceDate
		}
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

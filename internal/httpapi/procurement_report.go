package httpapi

import (
	"net/http"
	"sitewise/internal/invoices"
	"strings"

	"sitewise/internal/proposals"
	"sitewise/internal/store"
)

// getProcurementReport is the one-page report: the price table, the
// non-price matrix with the passages Jev picked, the PM's ratings and the
// recommendation. With no model enabled the draft is the code's factual
// summary of the price table; the PM writes the recommendation.
func getProcurementReport(w http.ResponseWriter, r *http.Request, deps Deps) {
	if !pathIDs(w, r, "pkg") {
		return
	}
	session, ok := deliverySession(w, r, deps, false)
	if !ok {
		return
	}
	writeProcurementReport(w, r, deps, session)
}

func writeProcurementReport(w http.ResponseWriter, r *http.Request, deps Deps, session store.Session) {
	t, err := deps.Store.ReadPriceTable(r.Context(), session.OrgID, r.PathValue("id"), r.PathValue("pkg"))
	if err != nil {
		procurementError(w, deps, err)
		return
	}
	rep, err := deps.Store.ReadComparisonReport(r.Context(), session.OrgID, r.PathValue("id"), r.PathValue("pkg"), t, "")
	if err != nil {
		procurementError(w, deps, err)
		return
	}
	labels := map[string]string{}
	for _, c := range rep.Criteria {
		labels[c.Key] = c.Label
	}
	var facts []proposals.FirmFacts
	for i, c := range t.Columns {
		name := c.Submission.Filename
		if c.Firm != nil {
			name = c.Firm.Name
		}
		f := proposals.FirmFacts{Name: name, BaseTotal: c.Totals.BaseTotal, KnownBase: c.Totals.KnownBase, NotPriced: c.Totals.NotPriced, Unread: c.Totals.Unread, Lowest: c.Totals.Lowest, Gap: c.Totals.Gap, Ratings: map[string]string{}}
		if i < len(rep.Columns) {
			for k, resp := range rep.Columns[i].Responses {
				if resp.Rating != "" {
					f.Ratings[labels[k]] = resp.Rating
				}
			}
		}
		facts = append(facts, f)
	}
	rep.Draft = proposals.DraftRecommendation(t.Title, facts)
	for _, col := range t.Columns {
		if col.Quote != nil && !t.Builder {
			lines := []string{"Quoted packages, alternatives and breakdowns are not additive. Confirm equivalent scope before selecting a firm."}
			for _, c := range t.Columns {
				name := c.Submission.Filename
				if c.Firm != nil {
					name = c.Firm.Name
				}
				fee := c.Cells["comparison:appointment"]
				if fee.Cents == nil {
					lines = append(lines, name+": comparison fee not confirmed.")
				} else {
					lines = append(lines, name+": confirmed comparison fee $"+invoices.DisplayCents(*fee.Cents)+" ex GST. Scope: "+fee.Basis)
				}
			}
			lines = append(lines, "Recommendation for "+t.Title+": [state the firm to appoint and why].")
			rep.Draft = strings.Join(lines, "\n")
			break
		}
	}

	award, err := deps.Store.ReadAward(r.Context(), session.OrgID, r.PathValue("id"), r.PathValue("pkg"))
	if err != nil {
		procurementError(w, deps, err)
		return
	}
	writeJSON(w, http.StatusOK, struct {
		Table  tableJSON              `json:"table"`
		Report store.ComparisonReport `json:"report"`
	}{tableBody(t, award), rep})
}

func putResponseRating(w http.ResponseWriter, r *http.Request, deps Deps) {
	if !pathIDs(w, r, "pkg", "sub") {
		return
	}
	session, ok := deliverySession(w, r, deps, true)
	if !ok {
		return
	}
	var body struct {
		Rating string `json:"rating"`
	}
	if readProposalJSON(w, r, deps.MaxBodyBytes, &body) != nil {
		http.Error(w, "invalid body", http.StatusBadRequest)
		return
	}
	if err := deps.Store.RateResponse(r.Context(), session.OrgID, r.PathValue("id"), r.PathValue("sub"), r.PathValue("key"), body.Rating, session.UserID); err != nil {
		procurementError(w, deps, err)
		return
	}
	if deps.Broker != nil {
		deps.Broker.Wake(session.OrgID)
	}
	writeProcurementReport(w, r, deps, session)
}

func putRecommendation(w http.ResponseWriter, r *http.Request, deps Deps) {
	if !pathIDs(w, r, "pkg") {
		return
	}
	session, ok := deliverySession(w, r, deps, true)
	if !ok {
		return
	}
	var body struct {
		Text    string `json:"text"`
		SignOff bool   `json:"sign_off"`
	}
	if readProposalJSON(w, r, deps.MaxBodyBytes, &body) != nil {
		http.Error(w, "invalid body", http.StatusBadRequest)
		return
	}
	if err := deps.Store.SaveRecommendation(r.Context(), session.OrgID, r.PathValue("id"), r.PathValue("pkg"), body.Text, session.UserID, body.SignOff); err != nil {
		procurementError(w, deps, err)
		return
	}
	if deps.Broker != nil {
		deps.Broker.Wake(session.OrgID)
	}
	writeProcurementReport(w, r, deps, session)
}

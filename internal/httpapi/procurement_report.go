package httpapi

import (
	"net/http"

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

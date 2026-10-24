package httpapi

import (
	"errors"
	"log"
	"net/http"
	"sitewise/internal/costs"
	"sitewise/internal/store"
)

func costError(w http.ResponseWriter, e error) {
	switch {
	case errors.Is(e, costs.ErrInvalid):
		http.Error(w, e.Error(), 422)
	case errors.Is(e, store.ErrFrozenCostPlan):
		writeJSON(w, 409, map[string]string{"error": "frozen"})
	case errors.Is(e, store.ErrVersionConflict):
		http.Error(w, "cost plan changed", 409)
	case errors.Is(e, store.ErrNotFound):
		http.Error(w, "not found", 404)
	default:
		http.Error(w, "cost operation failed", 500)
	}
}
func getCostPlan(w http.ResponseWriter, r *http.Request, d Deps) {
	s, ok := deliverySession(w, r, d, false)
	if !ok {
		return
	}
	p, e := d.Store.ReadCostPlan(r.Context(), s.OrgID, r.PathValue("id"), r.URL.Query().Get("version"))
	if e != nil {
		costError(w, e)
		return
	}
	writeJSON(w, 200, p)
}
func getCostTotals(w http.ResponseWriter, r *http.Request, d Deps) {
	s, ok := deliverySession(w, r, d, false)
	if !ok {
		return
	}
	by := r.URL.Query().Get("by")
	if by == "" {
		by = "overall"
	}
	p, e := d.Store.CostTotals(r.Context(), s.OrgID, r.PathValue("id"), r.URL.Query().Get("version"), by)
	if e != nil {
		costError(w, e)
		return
	}
	writeJSON(w, 200, p)
}
func costResult(w http.ResponseWriter, d Deps, org string, p costs.Plan, e error) {
	if e != nil {
		costError(w, e)
		return
	}
	totals, err := costs.Summarize(p, "overall")
	if err != nil {
		costError(w, err)
		return
	}
	p.Totals = &totals
	if d.Broker != nil {
		d.Broker.Wake(org)
	}
	writeJSON(w, 200, p)
}
func postCostItem(w http.ResponseWriter, r *http.Request, d Deps) {
	s, ok := deliverySession(w, r, d, true)
	if !ok {
		return
	}
	var in store.CostItemInput
	if readProposalJSON(w, r, d.MaxBodyBytes, &in) != nil {
		http.Error(w, "invalid body", 400)
		return
	}
	p, e := d.Store.CreateCostItem(r.Context(), s.OrgID, r.PathValue("id"), s.UserID, in)
	// A fee line by stage creates the stage plan when the project has none
	// (plan 7.6); the line itself is already saved.
	if e == nil && in.LineKind == "fee" && in.PackageStageID != "" {
		ensureProgram(r, d, s.OrgID, r.PathValue("id"), "cost_plan")
	}
	costResult(w, d, s.OrgID, p, e)
}
func patchCostItem(w http.ResponseWriter, r *http.Request, d Deps) {
	s, ok := deliverySession(w, r, d, true)
	if !ok {
		return
	}
	var in store.CostItemInput
	if readProposalJSON(w, r, d.MaxBodyBytes, &in) != nil {
		http.Error(w, "invalid body", 400)
		return
	}
	p, e := d.Store.PatchCostItem(r.Context(), s.OrgID, r.PathValue("id"), r.PathValue("item"), s.UserID, in)
	costResult(w, d, s.OrgID, p, e)
}
func putCostValue(w http.ResponseWriter, r *http.Request, d Deps) {
	s, ok := deliverySession(w, r, d, true)
	if !ok {
		return
	}
	var in store.CostValueInput
	if readProposalJSON(w, r, d.MaxBodyBytes, &in) != nil {
		http.Error(w, "invalid body", 400)
		return
	}
	p, e := d.Store.PutCostValue(r.Context(), s.OrgID, r.PathValue("id"), r.PathValue("item"), s.UserID, r.PathValue("metric"), in)
	costResult(w, d, s.OrgID, p, e)
}
func putCostSettings(w http.ResponseWriter, r *http.Request, d Deps) {
	s, ok := deliverySession(w, r, d, true)
	if !ok {
		return
	}
	var in store.CostSettingsInput
	if readProposalJSON(w, r, d.MaxBodyBytes, &in) != nil {
		http.Error(w, "invalid body", 400)
		return
	}
	p, e := d.Store.SaveCostSettings(r.Context(), s.OrgID, r.PathValue("id"), s.UserID, in)
	costResult(w, d, s.OrgID, p, e)
}
func postCostBaseline(w http.ResponseWriter, r *http.Request, d Deps) {
	s, ok := deliverySession(w, r, d, true)
	if !ok {
		return
	}
	var in store.CostWrite
	if readProposalJSON(w, r, d.MaxBodyBytes, &in) != nil {
		http.Error(w, "invalid body", 400)
		return
	}
	p, e := d.Store.BaselineCostPlan(r.Context(), s.OrgID, r.PathValue("id"), s.UserID, in)
	costResult(w, d, s.OrgID, p, e)
}
func postCostSubdivision(w http.ResponseWriter, r *http.Request, d Deps) {
	s, ok := deliverySession(w, r, d, true)
	if !ok {
		return
	}
	var in store.CostSubdivision
	if readProposalJSON(w, r, d.MaxBodyBytes, &in) != nil {
		http.Error(w, "invalid body", 400)
		return
	}
	p, e := d.Store.SubdivideCostItem(r.Context(), s.OrgID, r.PathValue("id"), r.PathValue("item"), s.UserID, in)
	costResult(w, d, s.OrgID, p, e)
}
func putScopeCostLink(w http.ResponseWriter, r *http.Request, d Deps) {
	s, ok := deliverySession(w, r, d, true)
	if !ok {
		return
	}
	var in store.CostLinkInput
	if readProposalJSON(w, r, d.MaxBodyBytes, &in) != nil {
		http.Error(w, "invalid body", 400)
		return
	}
	if !uuidPattern.MatchString(in.ScopeItemID) || !uuidPattern.MatchString(in.CostItemID) {
		http.Error(w, "invalid reference", 422)
		return
	}
	p, e := d.Store.LinkScopeCost(r.Context(), s.OrgID, r.PathValue("id"), s.UserID, in)
	costResult(w, d, s.OrgID, p, e)
}
func deleteCostItem(w http.ResponseWriter, r *http.Request, d Deps) {
	s, ok := deliverySession(w, r, d, true)
	if !ok {
		return
	}
	var in store.CostWrite
	if readProposalJSON(w, r, d.MaxBodyBytes, &in) != nil {
		http.Error(w, "invalid body", 400)
		return
	}
	p, e := d.Store.RemoveCostItem(r.Context(), s.OrgID, r.PathValue("id"), r.PathValue("item"), s.UserID, in)
	costResult(w, d, s.OrgID, p, e)
}
func getCostBenchmarks(w http.ResponseWriter, r *http.Request, d Deps) {
	s, ok := deliverySession(w, r, d, false)
	if !ok {
		return
	}
	out, e := d.Store.ReadCostBenchmarks(r.Context(), s.OrgID, r.PathValue("id"), r.PathValue("item"), r.URL.Query().Get("geography"), r.URL.Query().Get("quality"))
	if e != nil {
		costError(w, e)
		return
	}
	writeJSON(w, 200, map[string]any{"suggestions": out, "message": "Only reviewed benchmarks matching the plan price date, currency, tax basis, geography, quality, units and known applicability are offered."})
}
func postCostBenchmark(w http.ResponseWriter, r *http.Request, d Deps) {
	s, ok := deliverySession(w, r, d, true)
	if !ok {
		return
	}
	var in store.CostBenchmarkInput
	if readProposalJSON(w, r, d.MaxBodyBytes, &in) != nil {
		http.Error(w, "invalid body", 400)
		return
	}
	p, e := d.Store.ApplyCostBenchmark(r.Context(), s.OrgID, r.PathValue("id"), r.PathValue("item"), s.UserID, in)
	costResult(w, d, s.OrgID, p, e)
}

func postCostSuggestion(w http.ResponseWriter, r *http.Request, d Deps) {
	session, ok := deliverySession(w, r, d, true)
	if !ok {
		return
	}
	var in store.CostSuggestionInput
	if readProposalJSON(w, r, d.MaxBodyBytes, &in) != nil {
		http.Error(w, "invalid body", 400)
		return
	}
	p, e := d.Store.AcceptCostSuggestions(r.Context(), session.OrgID, r.PathValue("id"), session.UserID, r.PathValue("key"), in)
	costResult(w, d, session.OrgID, p, e)
}
func dismissCostSuggestion(w http.ResponseWriter, r *http.Request, d Deps) {
	session, ok := deliverySession(w, r, d, true)
	if !ok {
		return
	}
	var in store.CostWrite
	if readProposalJSON(w, r, d.MaxBodyBytes, &in) != nil {
		http.Error(w, "invalid body", 400)
		return
	}
	p, e := d.Store.DismissCostSuggestion(r.Context(), session.OrgID, r.PathValue("id"), session.UserID, r.PathValue("key"), r.Method == "DELETE", in)
	costResult(w, d, session.OrgID, p, e)
}

func prepareCostPlan(w http.ResponseWriter, r *http.Request, d Deps) {
	session, ok := deliverySession(w, r, d, true)
	if !ok {
		return
	}
	var in struct{}
	if readProposalJSON(w, r, d.MaxBodyBytes, &in) != nil {
		http.Error(w, "invalid body", 400)
		return
	}
	// Proposed consultants become packages first, so their fee lines are
	// prepared in the same request (owner decision 3, 2026-10-21).
	// A failed adoption leaves the consultants as proposals; it must never
	// stop the cost plan opening.
	adopted, e := d.Store.AdoptServiceSuggestions(r.Context(), session.OrgID, r.PathValue("id"), session.UserID, disciplines(d))
	if e != nil {
		if errors.Is(e, store.ErrNotFound) {
			costError(w, e)
			return
		}
		log.Printf("cost plan prepare: adopting proposed consultants: %v", e)
		adopted = 0
	}
	result, e := d.Store.PrepareCostPlan(r.Context(), session.OrgID, r.PathValue("id"), session.UserID)
	if e != nil {
		costError(w, e)
		return
	}
	if (adopted > 0 || result.Changed) && d.Broker != nil {
		d.Broker.Wake(session.OrgID)
	}
	writeJSON(w, 200, result)
}

package httpapi

import (
	"errors"
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

package httpapi_test

import (
	"encoding/json"
	"net/http"
	"testing"

	"sitewise/internal/costs"
)

// Bulk actions on selected Cost Plan lines through the API: one request, one
// plan version; a wrong org, a stale plan and bad input are refused.
func TestCostBulkRoute(t *testing.T) {
	a := newApp(t, withProfile(t))
	m := a.member(t, newUUID(t))
	project := m.createProject(t, "Bulk cost lines")
	base := "/api/projects/" + project + "/cost-plan"
	var plan costs.Plan
	for _, label := range []string{"Alpha", "Beta", "Gamma"} {
		body, _ := json.Marshal(map[string]any{"plan_version_id": plan.ID, "version": plan.Version, "label": label, "line_kind": "project_wide",
			"category": "contingency", "posting": true, "origin": "user", "meaning": "allowance"})
		m.call(t, http.MethodPost, base+"/items", body, http.StatusOK, &plan)
	}
	ids := func(labels ...string) []string {
		var out []string
		for _, l := range labels {
			for _, i := range plan.Items {
				if i.Label == l {
					out = append(out, i.ID)
				}
			}
		}
		return out
	}
	bulk := func(who *member, in map[string]any, want int) {
		t.Helper()
		in["plan_version_id"], in["version"] = plan.ID, plan.Version
		raw, _ := json.Marshal(in)
		var out costs.Plan
		var dest any
		if want == http.StatusOK {
			dest = &out
		}
		who.call(t, http.MethodPost, base+"/items/bulk", raw, want, dest)
		if want == http.StatusOK {
			plan = out
		}
	}
	bulk(m, map[string]any{"op": "set", "ids": ids("Alpha", "Beta"), "assign": "escalation", "excluded": true}, http.StatusOK)
	for _, i := range plan.Items {
		if (i.Label == "Alpha" || i.Label == "Beta") != (i.Excluded && i.Category == "escalation") {
			t.Errorf("%s: excluded %v category %s", i.Label, i.Excluded, i.Category)
		}
	}
	bulk(m, map[string]any{"op": "duplicate", "ids": ids("Gamma")}, http.StatusOK)
	if len(ids("Gamma (copy)")) != 1 {
		t.Fatalf("no copy in %d lines", len(plan.Items))
	}
	version := plan.Version
	bulk(m, map[string]any{"op": "delete", "ids": ids("Alpha", "Beta", "Gamma (copy)")}, http.StatusOK)
	if len(plan.Items) != 1 || plan.Version != version+1 || plan.Totals == nil {
		t.Fatalf("after delete: %d lines, version %d (was %d)", len(plan.Items), plan.Version, version)
	}

	bulk(m, map[string]any{"op": "delete", "ids": []string{"not-an-id"}}, http.StatusUnprocessableEntity)
	bulk(m, map[string]any{"op": "paint", "ids": ids("Gamma")}, http.StatusUnprocessableEntity)
	bulk(m, map[string]any{"op": "delete", "ids": []string{}}, http.StatusUnprocessableEntity)
	bulk(m, map[string]any{"op": "set", "ids": ids("Gamma"), "assign": "marble"}, http.StatusUnprocessableEntity)
	stale := map[string]any{"op": "delete", "ids": ids("Gamma")}
	raw, _ := json.Marshal(map[string]any{"plan_version_id": plan.ID, "version": plan.Version - 1, "op": stale["op"], "ids": stale["ids"]})
	m.call(t, http.MethodPost, base+"/items/bulk", raw, http.StatusConflict, nil)
	other := a.member(t, newUUID(t))
	bulk(other, map[string]any{"op": "delete", "ids": ids("Gamma")}, http.StatusNotFound)
}

package httpapi_test

import (
	"encoding/json"
	"net/http"
	"sort"
	"testing"
	"time"

	"sitewise/internal/costs"
)

func TestCostDisclosureReadBudgetAndIsolation(t *testing.T) {
	a := newApp(t, withProfile(t))
	m := a.member(t, newUUID(t))
	project := m.createProject(t, "Group summaries")
	base := "/api/projects/" + project + "/cost-plan"
	var plan costs.Plan
	add := func(label, parent string, posting bool, amount string) string {
		input := map[string]any{"plan_version_id": plan.ID, "version": plan.Version, "label": label, "parent_item_id": parent, "line_kind": "project_wide", "category": "contingency", "posting": posting, "origin": "user", "meaning": "allowance"}
		if amount != "" {
			input["values"] = map[string]any{"commitment": map[string]any{"amount": amount, "value_state": "known", "origin": "user", "meaning": "stated"}}
		}
		raw, _ := json.Marshal(input)
		m.call(t, http.MethodPost, base+"/items", raw, http.StatusOK, &plan)
		for _, item := range plan.Items {
			if item.Label == label {
				return item.ID
			}
		}
		t.Fatal("missing created line")
		return ""
	}
	g := add("Architect", "", false, "")
	for n, amount := range []string{"14500.00", "24000.00", "22000.00", "26000.00", "6500.00", "3500.00"} {
		add(string(rune('A'+n)), g, true, amount)
	}
	var totals costs.Totals
	m.getJSON(t, base+"/totals", &totals)
	if *totals.Rollups[g].Values["commitment"].Amount != "96500.00" || *totals.Overall["commitment"].Amount != "96500.00" {
		t.Fatal(totals)
	}
	for _, path := range []struct {
		name, url string
		p50, p90  time.Duration
	}{
		{"cost totals with groups", base + "/totals", 100 * time.Millisecond, 250 * time.Millisecond},
		{"invoice ledger with groups", "/api/projects/" + project + "/invoices", 150 * time.Millisecond, 300 * time.Millisecond},
	} {
		var samples []time.Duration
		for n := 0; n < 20; n++ {
			start := time.Now()
			m.call(t, http.MethodGet, path.url, nil, http.StatusOK, nil)
			samples = append(samples, time.Since(start))
		}
		sort.Slice(samples, func(i, j int) bool { return samples[i] < samples[j] })
		t.Logf("%s p50=%s p90=%s; budget=%s/%s", path.name, samples[9], samples[17], path.p50, path.p90)
		if samples[9] > path.p50 || samples[17] > path.p90 {
			t.Fatal("read budget exceeded")
		}
		foreign := a.member(t, newUUID(t))
		foreign.call(t, http.MethodGet, path.url, nil, http.StatusNotFound, nil)
	}
	if a.jevHits.Load() != 0 {
		t.Fatal("disclosure called Jev")
	}
}

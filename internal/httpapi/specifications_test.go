package httpapi_test

import (
	"encoding/json"
	"net/http"
	"testing"
	"time"

	"sitewise/bench"
	"sitewise/internal/latency"
	"sitewise/internal/specifications"
)

func TestSpecificationAPIAndSpeedBudgets(t *testing.T) {
	a := newApp(t, withProfile(t))
	m := a.member(t, newUUID(t))
	project := m.createProject(t, "Synthetic specification workflow")
	base := "/api/projects/" + project + "/specifications"
	var item specifications.Type
	m.call(t, http.MethodPost, base+"/types", []byte(`{"code":"P1","title":"Interior paint","kind":"finish","status":"active"}`), http.StatusCreated, &item)
	var app specifications.Application
	input, _ := json.Marshal(map[string]any{"type_id": item.ID, "location": "Office", "surface": "wall", "action": "repair", "inclusion": "included", "selection": "selected"})
	m.call(t, http.MethodPost, base+"/applications", input, http.StatusCreated, &app)
	var value specifications.Assertion
	input, _ = json.Marshal(map[string]any{"type_id": item.ID, "key": "sheen", "meaning": "selection", "value": map[string]string{"state": "known", "text": "Low sheen"}})
	m.call(t, http.MethodPost, base+"/assertions", input, http.StatusCreated, &value)
	m.call(t, http.MethodPost, base+"/assertions/"+value.ID+"/review", []byte(`{"version":1,"status":"accepted"}`), http.StatusOK, &value)
	foreign := a.member(t, newUUID(t))
	for _, path := range []string{base, base + "/export", base + "/issues"} {
		foreign.call(t, http.MethodGet, path, nil, http.StatusNotFound, nil)
	}
	foreign.call(t, http.MethodPost, base+"/types", []byte(`{"title":"Wrong org","kind":"finish"}`), http.StatusNotFound, nil)
	for _, path := range []string{"/cells", "/coverage", "/update", "/types/" + item.ID + "/adopt", "/types/" + item.ID + "/link"} {
		foreign.call(t, http.MethodPost, base+path, []byte(`{}`), http.StatusNotFound, nil)
	}
	m.call(t, http.MethodPost, base+"/types", []byte(`{"title":"Bad","kind":"unknown"}`), http.StatusUnprocessableEntity, nil)
	samples := map[string][]int64{}
	var cell specifications.Assertion
	var coverage specifications.Coverage
	for i := 0; i < 20; i++ {
		start := time.Now()
		input, _ = json.Marshal(map[string]any{"type_id": item.ID, "key": "material", "meaning": "requirement", "value": map[string]string{"state": "known", "text": "Synthetic finish"}, "expected_accepted_id": cell.ID})
		m.call(t, http.MethodPost, base+"/cells", input, http.StatusCreated, &cell)
		samples["specification_cell"] = append(samples["specification_cell"], time.Since(start).Microseconds())
		coverage.SystemID = "interiors"
		coverage.Mode = "managed_here"
		coverage.Maintainer = "Project manager"
		input, _ = json.Marshal(coverage)
		start = time.Now()
		m.call(t, http.MethodPost, base+"/coverage", input, http.StatusCreated, &coverage)
		samples["specification_coverage"] = append(samples["specification_coverage"], time.Since(start).Microseconds())
		start = time.Now()
		m.call(t, http.MethodPost, base+"/update", nil, http.StatusAccepted, nil)
		samples["specification_update"] = append(samples["specification_update"], time.Since(start).Microseconds())
		start = time.Now()
		item.Description = "Synthetic performance sample"
		input, _ = json.Marshal(item)
		m.call(t, http.MethodPut, base+"/types/"+item.ID, input, http.StatusOK, &item)
		samples["specification_write"] = append(samples["specification_write"], time.Since(start).Microseconds())
		var view struct {
			Snapshot specifications.Snapshot `json:"snapshot"`
			Rows     []specifications.Row    `json:"rows"`
		}
		start = time.Now()
		m.getJSON(t, base, &view)
		samples["specification_read"] = append(samples["specification_read"], time.Since(start).Microseconds())
		if len(view.Rows) != 1 || len(view.Rows[0].Fields) != 2 {
			t.Fatal("schedule did not assemble accepted values", view.Rows)
		}
		input, _ = json.Marshal(map[string]any{"title": "Synthetic schedule", "purpose": "preliminary", "revision": view.Snapshot.Revision})
		var issue specifications.Issue
		start = time.Now()
		m.call(t, http.MethodPost, base+"/issues", input, http.StatusCreated, &issue)
		samples["specification_issue"] = append(samples["specification_issue"], time.Since(start).Microseconds())
		start = time.Now()
		m.call(t, http.MethodGet, base+"/issues/"+issue.ID+"/export?format=pdf", nil, http.StatusOK, nil)
		samples["specification_export"] = append(samples["specification_export"], time.Since(start).Microseconds())
		foreign.call(t, http.MethodGet, base+"/issues/"+issue.ID+"/export?format=pdf", nil, http.StatusNotFound, nil)
	}
	budgets, err := bench.Budgets()
	if err != nil {
		t.Fatal(err)
	}
	for _, b := range budgets.Paths {
		if b.Name == "specification_write" {
			for _, name := range []string{"specification_cell", "specification_coverage", "specification_update"} {
				p50, _ := latency.Percentile(samples[name], .5)
				p90, _ := latency.Percentile(samples[name], .9)
				t.Logf("%s p50=%.2fms p90=%.2fms", name, float64(p50)/1000, float64(p90)/1000)
				if p50 > b.P50US || p90 > b.P90US {
					t.Fatalf("%s exceeds write budget", name)
				}
			}
		}
		if v, ok := samples[b.Name]; ok {
			p50, _ := latency.Percentile(v, .5)
			p90, _ := latency.Percentile(v, .9)
			t.Logf("%s p50=%.2fms p90=%.2fms", b.Name, float64(p50)/1000, float64(p90)/1000)
			if p50 > b.P50US || p90 > b.P90US {
				t.Fatalf("%s exceeds budget", b.Name)
			}
		}
	}
	if a.jevHits.Load() != 0 {
		t.Fatal("authoring or rendering called Jev")
	}
}

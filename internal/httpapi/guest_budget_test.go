package httpapi_test

import (
	"encoding/json"
	"fmt"
	"net/http"
	"path/filepath"
	"testing"
	"time"

	"sitewise/internal/latency"
)

// TestGuestOpenBudget holds guest_open (the link exchange and the guest
// page on a package link whose transmittal holds 50 documents), team (share
// create, list and revoke) and procurement_board (the board with link
// state) to their rows in bench/budgets.json, measured on the local test
// database (docs/plans/2026-11-17-guest-access.md, Task 10).
func TestGuestOpenBudget(t *testing.T) {
	f := newGuestFixture(t)
	var ids []string
	for i := 0; i < 50; i++ {
		ids = append(ids, f.document(t, f.m, f.project, fmt.Sprintf("A-%03d Transmitted sheet.pdf", 200+i)))
	}
	f.issueRFT(t, f.rftDraft, ids)
	budgets, err := latency.LoadBudgets(filepath.Join("..", "..", "bench", "budgets.json"))
	if err != nil {
		t.Fatal(err)
	}
	samples := map[string][]int64{}
	timed := func(path string, run func()) {
		start := time.Now()
		run()
		samples[path] = append(samples[path], time.Since(start).Microseconds())
	}
	base := "/api/projects/" + f.project + "/shares"
	hits := f.a.jevHits.Load()
	for i := 0; i < budgets.MinSamples; i++ {
		var created shareCreated
		body, _ := json.Marshal(map[string]any{"kind": "package", "package_id": f.pkg, "firm_id": f.firmA, "shared_with": "acme@example.test"})
		timed("team", func() { f.m.call(t, http.MethodPost, base, body, http.StatusCreated, &created) })
		token := created.Link[len(created.Link)-64:]
		g := f.guest(t, "")
		exchange, _ := json.Marshal(map[string]string{"token": token})
		timed("guest_open", func() {
			if code, raw := g.raw(t, http.MethodPost, "/api/guest/session", exchange, true); code != http.StatusNoContent {
				t.Fatalf("exchange %d %s", code, raw)
			}
		})
		var page struct {
			Documents []json.RawMessage `json:"documents"`
		}
		timed("guest_open", func() {
			code, raw := g.raw(t, http.MethodGet, "/api/guest", nil, false)
			if code != http.StatusOK || json.Unmarshal([]byte(raw), &page) != nil || len(page.Documents) != 50 {
				t.Fatalf("page %d %d", code, len(page.Documents))
			}
		})
		timed("procurement_board", func() { f.m.getJSON(t, "/api/projects/"+f.project+"/procurement/board", new(json.RawMessage)) })
		timed("team", func() { f.m.getJSON(t, base+"?package="+f.pkg, new(json.RawMessage)) })
		timed("team", func() { f.m.call(t, http.MethodDelete, base+"/"+created.Share.ID, nil, http.StatusOK, nil) })
	}
	var scoped latency.Budgets
	scoped.MinSamples = budgets.MinSamples
	for _, p := range budgets.Paths {
		if _, ok := samples[p.Name]; ok {
			scoped.Paths = append(scoped.Paths, p)
		}
	}
	for name, got := range samples {
		p50, _ := latency.Percentile(got, 0.5)
		p90, _ := latency.Percentile(got, 0.9)
		t.Logf("%s: %d samples, p50 %d us, p90 %d us", name, len(got), p50, p90)
	}
	code, report := latency.Gate(scoped, samples, true)
	if code != 0 {
		t.Fatal(report)
	}
	if f.a.jevHits.Load() != hits {
		t.Fatal("a guest open or a share called Jev")
	}
}

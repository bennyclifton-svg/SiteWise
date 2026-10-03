package httpapi

import (
	"net/http"
	"sync"
	"time"

	"sitewise/internal/latency"
)

// Budget paths recorded by the server itself; the names match
// bench/budgets.json. Stage paths come from the intake Observer.
const (
	pathWholeIntake     = "whole_intake"
	pathDocumentList    = "project_document_list"
	pathFieldCorrection = "field_correction"
	pathInviteAuth      = "project_invite_auth"
	pathHealthSpeed     = "health_speed"
	pathProfileRead     = "project_profile_read"
	pathProfileEdit     = "profile_edit"
	pathDocumentDelete  = "document_delete"
)

// speedWindow is how many recent observations each path keeps. The speed
// page reports recent behaviour, not the whole process lifetime.
const speedWindow = 1024

// recorder keeps a ring of recent durations per budget path. Observe is called
// from concurrent filings and requests and holds the lock only to store.
type recorder struct {
	mu    sync.Mutex
	paths map[string]*ring
}

type ring struct {
	us    [speedWindow]int64
	next  int
	n     int
	total int64
}

func newRecorder() *recorder {
	return &recorder{paths: map[string]*ring{}}
}

// Observe records one duration for path.
func (r *recorder) Observe(path string, d time.Duration) {
	r.mu.Lock()
	defer r.mu.Unlock()
	w := r.paths[path]
	if w == nil {
		w = &ring{}
		r.paths[path] = w
	}
	w.us[w.next] = d.Microseconds()
	w.next = (w.next + 1) % speedWindow
	if w.n < speedWindow {
		w.n++
	}
	w.total++
}

type speedReport struct {
	Window     int         `json:"window"`
	MinSamples int         `json:"min_samples"`
	Paths      []speedPath `json:"paths"`
}

// speedPath is one path's recent percentiles. WithinBudget is null until
// the window holds the gate's minimum sample count.
type speedPath struct {
	Name         string  `json:"name"`
	N            int     `json:"n"`
	Total        int64   `json:"total"`
	P50MS        float64 `json:"p50_ms"`
	P90MS        float64 `json:"p90_ms"`
	BudgetP50MS  float64 `json:"budget_p50_ms,omitempty"`
	BudgetP90MS  float64 `json:"budget_p90_ms,omitempty"`
	WithinBudget *bool   `json:"within_budget"`
}

// report lists every budgeted path, recorded or not, then any other
// recorded path. Percentiles use the gate's nearest-rank rule.
func (r *recorder) report(budgets latency.Budgets) speedReport {
	r.mu.Lock()
	snap := make(map[string][]int64, len(r.paths))
	totals := make(map[string]int64, len(r.paths))
	for name, w := range r.paths {
		snap[name] = append([]int64(nil), w.us[:w.n]...)
		totals[name] = w.total
	}
	r.mu.Unlock()

	rep := speedReport{Window: speedWindow, MinSamples: budgets.MinSamples}
	seen := map[string]bool{}
	for _, b := range budgets.Paths {
		seen[b.Name] = true
		p := summarizePath(b.Name, snap[b.Name], totals[b.Name])
		p.BudgetP50MS = ms(b.P50US)
		p.BudgetP90MS = ms(b.P90US)
		if p.N >= budgets.MinSamples {
			ok := p.P50MS <= p.BudgetP50MS && p.P90MS <= p.BudgetP90MS
			p.WithinBudget = &ok
		}
		rep.Paths = append(rep.Paths, p)
	}
	for name, s := range snap {
		if !seen[name] {
			rep.Paths = append(rep.Paths, summarizePath(name, s, totals[name]))
		}
	}
	return rep
}

func summarizePath(name string, us []int64, total int64) speedPath {
	p := speedPath{Name: name, N: len(us), Total: total}
	if len(us) == 0 {
		return p
	}
	p50, _ := latency.Percentile(us, 0.5)
	p90, _ := latency.Percentile(us, 0.9)
	p.P50MS, p.P90MS = ms(p50), ms(p90)
	return p
}

func ms(us int64) float64 { return float64(us) / 1000 }

// getSpeed serves recorded percentiles. It reads memory only: no database
// query and no Jev call, so the page cannot slow the paths it reports.
func getSpeed(w http.ResponseWriter, r *http.Request, deps Deps) {
	if _, ok := ownerSession(w, r, deps); !ok {
		return
	}
	if deps.Speed == nil {
		http.Error(w, "speed not recorded", http.StatusServiceUnavailable)
		return
	}
	writeJSON(w, http.StatusOK, deps.Speed.report(deps.Budgets))
}

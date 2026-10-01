package httpapi

import (
	"context"
	"errors"
	"slices"
	"strings"
	"testing"
	"time"

	"sitewise/internal/jev"
	"sitewise/internal/latency"
	"sitewise/internal/store"
)

var healthNow = time.Date(2026, 10, 1, 9, 0, 0, 0, time.UTC)

func healthyChecker() *healthChecker {
	return &healthChecker{
		ping: func(context.Context) error { return nil },
		jev: func() jev.Status {
			return jev.Status{Circuit: jev.CircuitClosed, Reached: healthNow.Add(-10 * time.Second)}
		},
		now: func() time.Time { return healthNow },
	}
}

func backlogOf(list ...store.Backlog) func(context.Context) ([]store.Backlog, error) {
	return func(context.Context) ([]store.Backlog, error) { return list, nil }
}

func TestHealthStatus(t *testing.T) {
	cases := []struct {
		name    string
		edit    func(*healthChecker)
		backlog func(context.Context) ([]store.Backlog, error)
		want    string
		reason  string
	}{
		{name: "healthy", want: healthOK},
		{
			name: "database down",
			edit: func(h *healthChecker) {
				h.ping = func(context.Context) error { return errors.New("connection refused") }
			},
			backlog: func(context.Context) ([]store.Backlog, error) { return nil, errors.New("connection refused") },
			want:    healthDown,
			reason:  "database",
		},
		{
			name: "stale jev probe",
			edit: func(h *healthChecker) {
				h.jev = func() jev.Status {
					return jev.Status{Circuit: jev.CircuitClosed, Reached: healthNow.Add(-jevStaleAfter - time.Second)}
				}
			},
			want:   healthDegraded,
			reason: "jev",
		},
		{
			name: "jev never reached",
			edit: func(h *healthChecker) {
				h.jev = func() jev.Status { return jev.Status{Circuit: jev.CircuitClosed} }
			},
			want:   healthDegraded,
			reason: "jev",
		},
		{
			name: "jev status not wired",
			edit: func(h *healthChecker) { h.jev = nil },
			want: healthDegraded, reason: "jev",
		},
		{
			name: "open circuit",
			edit: func(h *healthChecker) {
				h.jev = func() jev.Status { return jev.Status{Circuit: jev.CircuitOpen, Reached: healthNow} }
			},
			want:   healthDegraded,
			reason: "circuit",
		},
		{
			name:    "intake backlog",
			backlog: backlogOf(store.Backlog{Kind: store.JobKindIntake, Queued: 2, Oldest: intakeBacklogLimit + time.Second}),
			want:    healthDegraded,
			reason:  "intake",
		},
		{
			name:    "background backlog within limit",
			backlog: backlogOf(store.Backlog{Kind: store.JobKindFullText, Queued: 40, Oldest: backgroundBacklogLimit - time.Second}),
			want:    healthOK,
		},
		{
			name:    "background backlog over limit",
			backlog: backlogOf(store.Backlog{Kind: store.JobKindFullText, Queued: 40, Oldest: backgroundBacklogLimit + time.Second}),
			want:    healthDegraded,
			reason:  "full_text",
		},
		{
			name:    "backlog unreadable",
			backlog: func(context.Context) ([]store.Backlog, error) { return nil, errors.New("timeout") },
			want:    healthDegraded,
			reason:  "backlog",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			h := healthyChecker()
			if tc.edit != nil {
				tc.edit(h)
			}
			backlog := tc.backlog
			if backlog == nil {
				backlog = backlogOf()
			}
			got := h.check(context.Background(), backlog)
			if got.Status != tc.want {
				t.Fatalf("status %s want %s: %+v", got.Status, tc.want, got)
			}
			if tc.reason == "" && len(got.Reasons) != 0 {
				t.Fatalf("unexpected reasons %v", got.Reasons)
			}
			if tc.reason != "" && !slices.ContainsFunc(got.Reasons, func(r string) bool { return strings.Contains(r, tc.reason) }) {
				t.Fatalf("reasons %v lack %q", got.Reasons, tc.reason)
			}
		})
	}
}

func TestHealthBoundsAHungDatabase(t *testing.T) {
	h := healthyChecker()
	h.ping = func(ctx context.Context) error {
		<-ctx.Done()
		return ctx.Err()
	}
	slow := func(ctx context.Context) ([]store.Backlog, error) {
		<-ctx.Done()
		return nil, ctx.Err()
	}
	start := time.Now()
	got := h.check(context.Background(), slow)
	if took := time.Since(start); took > 2*dbCheckTimeout {
		t.Fatalf("check took %s", took)
	}
	if got.Status != healthDown {
		t.Fatalf("status %s", got.Status)
	}
}

func TestRecorderKeepsARecentWindow(t *testing.T) {
	r := newRecorder()
	for i := range speedWindow + 100 {
		// The first 100 samples are slow and fall out of the window.
		d := time.Millisecond
		if i < 100 {
			d = time.Second
		}
		r.Observe("whole_intake", d)
	}
	budgets := latency.Budgets{MinSamples: 20, Paths: []latency.PathBudget{
		{Name: "whole_intake", P50US: 1_000_000, P90US: 2_000_000},
		{Name: "health_speed", P50US: 25_000, P90US: 100_000},
	}}
	rep := r.report(budgets)
	whole := pathReport(t, rep, "whole_intake")
	if whole.N != speedWindow || whole.Total != speedWindow+100 {
		t.Fatalf("window %+v", whole)
	}
	if whole.P90MS != 1 || whole.WithinBudget == nil || !*whole.WithinBudget {
		t.Fatalf("percentiles %+v", whole)
	}
	if health := pathReport(t, rep, "health_speed"); health.N != 0 || health.WithinBudget != nil {
		t.Fatalf("unrecorded path %+v", health)
	}
}

func TestRecorderFlagsABreachedBudget(t *testing.T) {
	r := newRecorder()
	for range 20 {
		r.Observe("health_speed", 200*time.Millisecond)
	}
	rep := r.report(latency.Budgets{MinSamples: 20, Paths: []latency.PathBudget{
		{Name: "health_speed", P50US: 25_000, P90US: 100_000},
	}})
	if got := pathReport(t, rep, "health_speed"); got.WithinBudget == nil || *got.WithinBudget {
		t.Fatalf("breach not flagged %+v", got)
	}
}

func pathReport(t *testing.T, rep speedReport, name string) speedPath {
	t.Helper()
	for _, p := range rep.Paths {
		if p.Name == name {
			return p
		}
	}
	t.Fatalf("no %s in %+v", name, rep.Paths)
	return speedPath{}
}

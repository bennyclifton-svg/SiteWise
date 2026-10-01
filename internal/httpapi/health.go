package httpapi

import (
	"context"
	"errors"
	"fmt"
	"log"
	"math"
	"net/http"
	"sync"
	"time"

	"sitewise/internal/jev"
	"sitewise/internal/store"
)

const (
	healthOK       = "ok"
	healthDegraded = "degraded"
	healthDown     = "down"

	// jevStaleAfter is four missed probes at the 30 s probe interval serve
	// uses. Older than this, the provider is treated as unreachable.
	jevStaleAfter = 2 * time.Minute
	// intakeBacklogLimit: a filing runs within its 15 s foreground timeout,
	// so a pending filing twice that old is stuck, not slow.
	intakeBacklogLimit = 30 * time.Second
	// backgroundBacklogLimit: background stages may queue behind a burst,
	// but work this old is not draining.
	backgroundBacklogLimit = 15 * time.Minute
	// dbCheckTimeout bounds each database read so health answers even when
	// the database hangs.
	dbCheckTimeout = 500 * time.Millisecond
)

// healthChecker computes status from the database, the Jev client's own
// state and the job backlog. Nothing here calls Jev.
type healthChecker struct {
	ping func(context.Context) error
	// jev is nil when the Jev client does not report status; health then
	// reports the provider as unknown rather than healthy.
	jev func() jev.Status
	now func() time.Time
}

type healthReport struct {
	Status   string          `json:"status"`
	Reasons  []string        `json:"reasons"`
	Database databaseHealth  `json:"database"`
	Jev      jevHealth       `json:"jev"`
	Backlog  []backlogHealth `json:"backlog"`
}

type databaseHealth struct {
	OK        bool    `json:"ok"`
	LatencyMS float64 `json:"latency_ms"`
}

type jevHealth struct {
	Circuit string `json:"circuit"`
	// ReachedAgeS is null when the provider has not answered since start.
	ReachedAgeS *float64 `json:"reached_age_s"`
	Stale       bool     `json:"stale"`
}

type backlogHealth struct {
	Kind    string  `json:"kind"`
	Queued  int64   `json:"queued"`
	OldestS float64 `json:"oldest_s"`
}

// check evaluates every signal. backlog is the process view for public
// health and the caller's org for the detailed view.
func (h *healthChecker) check(ctx context.Context, backlog func(context.Context) ([]store.Backlog, error)) healthReport {
	rep := healthReport{Status: healthOK, Reasons: []string{}, Backlog: []backlogHealth{}}
	degrade := func(reason string) {
		rep.Reasons = append(rep.Reasons, reason)
		if rep.Status == healthOK {
			rep.Status = healthDegraded
		}
	}

	pingCtx, cancel := context.WithTimeout(ctx, dbCheckTimeout)
	start := time.Now()
	err := h.ping(pingCtx)
	cancel()
	rep.Database.LatencyMS = float64(time.Since(start).Microseconds()) / 1000
	if err != nil {
		rep.Status = healthDown
		rep.Reasons = append(rep.Reasons, "database unreachable")
	} else {
		rep.Database.OK = true
	}

	rep.Jev = h.jevHealth()
	switch {
	case rep.Jev.Circuit == "unknown":
		degrade("jev status unknown")
	case rep.Jev.ReachedAgeS == nil:
		degrade("jev not reached since start")
	case rep.Jev.Stale:
		degrade(fmt.Sprintf("jev probe stale %.0fs", *rep.Jev.ReachedAgeS))
	}
	if rep.Jev.Circuit == jev.CircuitOpen || rep.Jev.Circuit == jev.CircuitHalfOpen {
		degrade("jev circuit " + rep.Jev.Circuit)
	}

	// A down database cannot report its backlog; the reason is already given.
	if rep.Database.OK {
		blCtx, cancel := context.WithTimeout(ctx, dbCheckTimeout)
		list, err := backlog(blCtx)
		cancel()
		if err != nil {
			degrade("backlog unreadable")
		}
		for _, b := range list {
			rep.Backlog = append(rep.Backlog, backlogHealth{Kind: b.Kind, Queued: b.Queued, OldestS: math.Round(b.Oldest.Seconds()*10) / 10})
			limit := backgroundBacklogLimit
			if b.Kind == store.JobKindIntake {
				limit = intakeBacklogLimit
			}
			if b.Queued > 0 && b.Oldest > limit {
				degrade(fmt.Sprintf("%s backlog oldest %.0fs", b.Kind, b.Oldest.Seconds()))
			}
		}
	}
	return rep
}

func (h *healthChecker) jevHealth() jevHealth {
	if h.jev == nil {
		return jevHealth{Circuit: "unknown", Stale: true}
	}
	s := h.jev()
	out := jevHealth{Circuit: s.Circuit}
	if s.Reached.IsZero() {
		out.Stale = true
		return out
	}
	age := h.now().Sub(s.Reached)
	secs := math.Round(age.Seconds()*10) / 10
	out.ReachedAgeS = &secs
	out.Stale = age > jevStaleAfter
	return out
}

// publicHealth answers with a status word only, for Caddy and uptime checks:
// 200 while the app can serve (ok or degraded), 503 when the database is
// down. A change of status is logged with its reasons for the operator.
type publicHealth struct {
	checker *healthChecker
	backlog func(context.Context) ([]store.Backlog, error)
	log     *log.Logger
	observe func(string, time.Duration)

	mu   sync.Mutex
	last string
}

func (p *publicHealth) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	start := time.Now()
	rep := p.checker.check(r.Context(), p.backlog)
	p.noteChange(rep)
	status := http.StatusOK
	if rep.Status == healthDown {
		status = http.StatusServiceUnavailable
	}
	w.Header().Set("Cache-Control", "no-store")
	writeJSON(w, status, map[string]string{"status": rep.Status})
	if p.observe != nil {
		p.observe(pathHealthSpeed, time.Since(start))
	}
}

func (p *publicHealth) noteChange(rep healthReport) {
	p.mu.Lock()
	changed := rep.Status != p.last
	p.last = rep.Status
	p.mu.Unlock()
	if changed {
		p.log.Printf("health %s reasons=%q", rep.Status, rep.Reasons)
	}
}

// getHealth is the detailed view for an org owner. Backlog is the owner's
// org only, so no tenant learns another's queue.
func getHealth(w http.ResponseWriter, r *http.Request, deps Deps) {
	session, ok := ownerSession(w, r, deps)
	if !ok {
		return
	}
	if deps.Health == nil {
		http.Error(w, "health not configured", http.StatusServiceUnavailable)
		return
	}
	rep := deps.Health.check(r.Context(), func(ctx context.Context) ([]store.Backlog, error) {
		return deps.Store.OrgBacklog(ctx, session.OrgID)
	})
	writeJSON(w, http.StatusOK, rep)
}

// ownerSession admits members whose role in the session org is owner.
func ownerSession(w http.ResponseWriter, r *http.Request, deps Deps) (store.Session, bool) {
	session, ok := memberSession(w, r, deps)
	if !ok {
		return store.Session{}, false
	}
	role, err := deps.Store.Role(r.Context(), session.OrgID, session.UserID)
	if err != nil && !errors.Is(err, store.ErrNotFound) {
		http.Error(w, "membership failed", http.StatusInternalServerError)
		return store.Session{}, false
	}
	if role != "owner" {
		http.Error(w, "forbidden", http.StatusForbidden)
		return store.Session{}, false
	}
	return session, true
}

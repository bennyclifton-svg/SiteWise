package httpapi

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"sync"
	"time"

	"sitewise/internal/auth"
	"sitewise/internal/events"
	"sitewise/internal/knowledge"
	engine "sitewise/internal/program"
	"sitewise/internal/store"
)

// The ask-first review (plan docs/plans/2026-10-18-program.md section 7.4):
// the program_review job, GET /program/changes and POST
// /program/changes/decide. No Jev anywhere: the review regenerates in code
// and diffs by source key.
//
// Trigger. The job is a debounced goroutine owned by the server, like the
// invoice and proposal readers, kicked after a successful write on the
// routes that change its inputs (Profile values, work items, packages and
// package stages) and after every program write (so a pending item an edit
// resolved is superseded). The events broker cannot carry this: it is a
// per-org client cursor that most Profile, work item and package writes do
// not publish to. A leased job in internal/jobs would survive a restart,
// but the review is cheap and idempotent, so GET /program kicks it as well:
// opening the program reviews what changed while the server was down, what
// a background Profile reading changed, and a new library version, with
// nothing judged while the user waits. A kick lost to a restart is
// therefore recovered the next time anyone opens the program.

const (
	// eventProgramReview tells open views the pending review set changed;
	// the program version did not, so program.changed would be ignored.
	eventProgramReview = "program.review"
	reviewDebounce     = 2 * time.Second
	reviewTimeout      = 30 * time.Second
)

// programReviewer runs the review for a project once its inputs have been
// quiet for the debounce delay. Kicks are keyed by project id (globally
// unique); the org comes from the session that made the change, resolved in
// the background so the kick adds nothing to the request.
type programReviewer struct {
	delay time.Duration
	run   func(ctx context.Context, sessionID, project string)

	mu      sync.Mutex
	pending map[string]*pendingReview
	wg      sync.WaitGroup
}

type pendingReview struct {
	timer   *time.Timer
	session string
}

func newProgramReviewer(st *store.Store, cat *knowledge.Catalog, broker *events.Broker, lg *log.Logger, delay time.Duration) *programReviewer {
	if delay <= 0 {
		delay = reviewDebounce
	}
	return &programReviewer{delay: delay, pending: map[string]*pendingReview{},
		run: func(ctx context.Context, sessionID, project string) {
			sess, _, err := st.LookupSession(ctx, sessionID)
			if err != nil {
				return // signed out since; the next open reviews again
			}
			out, err := st.ReviewProgram(ctx, sess.OrgID, project, cat)
			if err != nil {
				lg.Printf("program review project=%s: %v", project, err)
				return
			}
			if out.Changed {
				publishReview(ctx, broker, lg, sess.OrgID, project, out.ProgramID, out.Pending)
			}
		}}
}

// Kick (re)starts the project's debounce timer: a burst of edits makes one
// review, delay after the last of them.
func (r *programReviewer) Kick(sessionID, project string) {
	if r == nil || project == "" {
		return
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if p, ok := r.pending[project]; ok && p.timer.Stop() {
		p.session = sessionID
		p.timer.Reset(r.delay)
		return
	}
	p := &pendingReview{session: sessionID}
	r.wg.Add(1)
	p.timer = time.AfterFunc(r.delay, func() {
		defer r.wg.Done()
		r.mu.Lock()
		if r.pending[project] == p {
			delete(r.pending, project)
		}
		session := p.session
		r.mu.Unlock()
		ctx, cancel := context.WithTimeout(context.Background(), reviewTimeout)
		defer cancel()
		r.run(ctx, session, project)
	})
	r.pending[project] = p
}

// Wait blocks until every scheduled review has run.
func (r *programReviewer) Wait() { r.wg.Wait() }

// Stop drops reviews not yet started and waits for running ones. A dropped
// review is recovered the next time the program is opened.
func (r *programReviewer) Stop() {
	if r == nil {
		return
	}
	r.mu.Lock()
	for project, p := range r.pending {
		if p.timer.Stop() {
			r.wg.Done()
		}
		delete(r.pending, project)
	}
	r.mu.Unlock()
	r.wg.Wait()
}

func publishReview(ctx context.Context, broker *events.Broker, lg *log.Logger, org, project, programID string, pending int) {
	if broker == nil {
		return
	}
	payload, _ := json.Marshal(map[string]any{"project_id": project, "program_id": programID, "changes_pending": pending})
	if _, err := broker.Commit(ctx, org, eventProgramReview, "", string(payload)); err != nil {
		lg.Printf("program review event: %v", err)
	}
}

// programReviewRoutes are the routes whose success can change the review:
// the program's inputs (Profile, planning values, work items, packages and
// their stages), program writes, and opening the program.
var programReviewRoutes = map[string]bool{
	"PUT /projects/{id}/profile/{key}":                   true,
	"PUT /projects/{id}/profile/scope":                   true,
	"POST /projects/{id}/parts":                          true,
	"PATCH /projects/{id}/parts/{part}":                  true,
	"PUT /projects/{id}/planning/{key}":                  true,
	"DELETE /projects/{id}/planning/{key}":               true,
	"POST /projects/{id}/proposals/{key}/accept":         true,
	"POST /projects/{id}/proposals/{key}/dismiss":        true,
	"DELETE /projects/{id}/proposals/{key}/decision":     true,
	"POST /projects/{id}/documents/delete":               true,
	"POST /projects/{id}/works":                          true,
	"PATCH /projects/{id}/works/{wi}":                    true,
	"POST /projects/{id}/works/{wi}/split":               true,
	"POST /projects/{id}/works/{wi}/retire":              true,
	"POST /projects/{id}/packages":                       true,
	"PATCH /projects/{id}/packages/{pkg}":                true,
	"POST /projects/{id}/packages/{pkg}/stages":          true,
	"PATCH /projects/{id}/packages/{pkg}/stages/{stage}": true,
	"POST /projects/{id}/packages/proposed/accept":       true,
	"POST /projects/{id}/procurement/rows":               true,
	"POST /projects/{id}/procurement/create":             true,
	"POST /projects/{id}/procurement/dismiss":            true,
	"GET /projects/{id}/program":                         true,
	"POST /projects/{id}/program/ops":                    true,
	"POST /projects/{id}/program/undo":                   true,
	"POST /projects/{id}/program/redo":                   true,
	"POST /projects/{id}/program/accept":                 true,
	"POST /projects/{id}/program/question":               true,
	"POST /projects/{id}/program/changes/decide":         true,
}

// statusWriter remembers the status a handler wrote.
type statusWriter struct {
	http.ResponseWriter
	status int
}

func (s *statusWriter) WriteHeader(code int) {
	if s.status == 0 {
		s.status = code
	}
	s.ResponseWriter.WriteHeader(code)
}

func (s *statusWriter) Write(b []byte) (int, error) {
	if s.status == 0 {
		s.status = http.StatusOK
	}
	return s.ResponseWriter.Write(b)
}

// reviewHook kicks the review after a successful request on a review
// route; other routes are returned unchanged.
func reviewHook(pattern string, h func(http.ResponseWriter, *http.Request, Deps)) func(http.ResponseWriter, *http.Request, Deps) {
	if !programReviewRoutes[pattern] {
		return h
	}
	return func(w http.ResponseWriter, r *http.Request, d Deps) {
		if d.ProgramReview == nil {
			h(w, r, d)
			return
		}
		sw := &statusWriter{ResponseWriter: w}
		h(sw, r, d)
		if sw.status < 200 || sw.status >= 300 {
			return
		}
		if c, err := r.Cookie(auth.SessionCookie); err == nil && c.Value != "" {
			d.ProgramReview.Kick(c.Value, r.PathValue("id"))
		}
	}
}

// getProgramChanges is GET /program/changes: the pending review items.
func getProgramChanges(w http.ResponseWriter, r *http.Request, d Deps) {
	s, ok := programSession(w, r, d, false)
	if !ok {
		return
	}
	list, err := d.Store.ProgramChanges(r.Context(), s.OrgID, r.PathValue("id"))
	if err != nil {
		programError(w, r, d, s.OrgID, store.ProgramResult{}, err)
		return
	}
	writeJSON(w, http.StatusOK, list)
}

// postProgramDecide is POST /program/changes/decide {"version",
// "decisions":[{"id","choice"}]} or {"version","all":"adopt"}. Adopted
// changes apply as one undoable batch; the response is the program view.
func postProgramDecide(w http.ResponseWriter, r *http.Request, d Deps) {
	s, ok := programSession(w, r, d, true)
	if !ok {
		return
	}
	var body struct {
		Version   int64                   `json:"version"`
		Decisions []store.ProgramDecision `json:"decisions"`
		All       string                  `json:"all"`
	}
	if readProposalJSON(w, r, d.MaxBodyBytes, &body) != nil {
		http.Error(w, "invalid body", http.StatusBadRequest)
		return
	}
	if body.Version < 1 {
		programInvalid(w, "version", "The program version is required.")
		return
	}
	if body.All != "" && body.All != engine.ChoiceAdopt {
		programInvalid(w, "all", `"all" is "adopt".`)
		return
	}
	ctx, project := r.Context(), r.PathValue("id")
	res, err := d.Store.DecideProgramChanges(ctx, s.OrgID, project, s.UserID, body.Version, body.Decisions, body.All == engine.ChoiceAdopt)
	if err != nil {
		programError(w, r, d, s.OrgID, res, err)
		return
	}
	view, err := d.Store.ResultView(ctx, s.OrgID, project, res, d.Knowledge)
	if err != nil {
		programError(w, r, d, s.OrgID, res, err)
		return
	}
	programChanged(r, d, s.OrgID, res.Program)
	publishReview(ctx, d.Broker, d.Log, s.OrgID, project, res.Program.ID, view.ChangesPending)
	writeJSON(w, http.StatusOK, view)
}

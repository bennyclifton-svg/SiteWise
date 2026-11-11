package httpapi

// The profile's run strip reads (docs/design/2026-11-10-run-time-feedback.md):
// document progress, the passages just read, and what an update changes
// elsewhere. Reads only; none calls Jev.

import (
	"errors"
	"net/http"
	"strings"
	"time"

	"sitewise/internal/store"
)

// maxFindDocuments bounds one finds read: the documents being read at once.
const maxFindDocuments = 20

// runProject is the shared front of the three reads: a member, a project id
// shaped like one, and the profile routes switched on.
func runProject(w http.ResponseWriter, r *http.Request, deps Deps) (store.Session, string, bool) {
	session, ok := memberSession(w, r, deps)
	if !ok {
		return session, "", false
	}
	project := r.PathValue("id")
	if !uuidPattern.MatchString(project) || deps.Knowledge == nil {
		http.NotFound(w, r)
		return session, "", false
	}
	return session, project, true
}

func writeRun(w http.ResponseWriter, r *http.Request, v any, err error) {
	if errors.Is(err, store.ErrNotFound) {
		http.NotFound(w, r)
		return
	}
	if err != nil {
		http.Error(w, "read failed", http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusOK, v)
}

func getProfileProgress(w http.ResponseWriter, r *http.Request, deps Deps) {
	session, project, ok := runProject(w, r, deps)
	if !ok {
		return
	}
	progress, err := deps.Store.ProfileProgress(r.Context(), session.OrgID, project)
	writeRun(w, r, progress, err)
}

// getProfileFinds reads ?docs=id,id (the documents being read) and ?after=
// (RFC 3339; the last time the browser saw). Without after it reads the last
// thirty seconds.
func getProfileFinds(w http.ResponseWriter, r *http.Request, deps Deps) {
	session, project, ok := runProject(w, r, deps)
	if !ok {
		return
	}
	var docs []string
	for _, id := range strings.Split(r.URL.Query().Get("docs"), ",") {
		if id = strings.TrimSpace(id); id == "" {
			continue
		}
		if !uuidPattern.MatchString(id) {
			http.Error(w, "invalid document", http.StatusBadRequest)
			return
		}
		docs = append(docs, id)
	}
	if len(docs) > maxFindDocuments {
		http.Error(w, "too many documents", http.StatusBadRequest)
		return
	}
	after := time.Now().Add(-30 * time.Second)
	if v := r.URL.Query().Get("after"); v != "" {
		t, err := time.Parse(time.RFC3339Nano, v)
		if err != nil {
			http.Error(w, "invalid after", http.StatusBadRequest)
			return
		}
		after = t
	}
	finds, err := deps.Store.ProfileFinds(r.Context(), session.OrgID, project, docs, after)
	writeRun(w, r, map[string]any{"finds": finds}, err)
}

// getProfileNext is timed on the packages read's budget (100/250 ms): the
// proposals it counts are that read's and the cost plan's.
func getProfileNext(w http.ResponseWriter, r *http.Request, deps Deps) {
	session, project, ok := runProject(w, r, deps)
	if !ok {
		return
	}
	next, err := deps.Store.ProfileNext(r.Context(), session.OrgID, project)
	writeRun(w, r, next, err)
}

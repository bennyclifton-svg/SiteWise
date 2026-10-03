package httpapi

import (
	"errors"
	"net/http"
	"sitewise/internal/store"
	"strconv"
)

func getProfileSources(w http.ResponseWriter, r *http.Request, deps Deps) {
	session, ok := memberSession(w, r, deps)
	if !ok {
		return
	}
	project := r.PathValue("id")
	if !uuidPattern.MatchString(project) || deps.Knowledge == nil {
		http.NotFound(w, r)
		return
	}
	offset := 0
	if v := r.URL.Query().Get("offset"); v != "" {
		n, err := strconv.Atoi(v)
		if err != nil || n < 0 || n > 1000000 {
			http.Error(w, "invalid offset", 400)
			return
		}
		offset = n
	}
	system := r.URL.Query().Get("system")
	if system != "" {
		if _, ok := deps.Knowledge.System(system); !ok {
			http.Error(w, "unknown system", 400)
			return
		}
	}
	outcome := r.URL.Query().Get("outcome")
	switch outcome {
	case "", "needs_mapping", "mapped", "background", "pending":
	default:
		http.Error(w, "invalid outcome", 400)
		return
	}
	records, err := deps.Store.SourceRecords(r.Context(), session.OrgID, project, system, outcome, offset)
	if errors.Is(err, store.ErrNotFound) {
		http.NotFound(w, r)
		return
	}
	if err != nil {
		http.Error(w, "source read failed", 500)
		return
	}
	writeJSON(w, http.StatusOK, records)
}

package httpapi

import (
	"errors"
	"net/http"
	"sitewise/internal/reports"
	"sitewise/internal/store"
)

func reportSession(w http.ResponseWriter, r *http.Request, deps Deps, write bool) (store.Session, bool) {
	if write && !originOK(r, deps.PublicOrigin) {
		http.Error(w, "origin rejected", http.StatusForbidden)
		return store.Session{}, false
	}
	session, ok := memberSession(w, r, deps)
	if !ok {
		return session, false
	}
	if !uuidPattern.MatchString(r.PathValue("r")) {
		http.Error(w, "not found", http.StatusNotFound)
		return session, false
	}
	return session, true
}

func postReport(w http.ResponseWriter, r *http.Request, deps Deps) {
	if !originOK(r, deps.PublicOrigin) {
		http.Error(w, "origin rejected", http.StatusForbidden)
		return
	}
	session, ok := memberSession(w, r, deps)
	if !ok {
		return
	}
	project := r.PathValue("id")
	if !uuidPattern.MatchString(project) {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}
	if _, err := deps.Store.GetProject(r.Context(), session.OrgID, project); err != nil {
		reportError(w, err)
		return
	}
	var body struct {
		Kind      string `json:"kind"`
		PackageID string `json:"package_id"`
	}
	if err := readProposalJSON(w, r, deps.MaxBodyBytes, &body); err != nil {
		http.Error(w, "invalid body", http.StatusBadRequest)
		return
	}
	report, err := deps.Store.CreateReport(r.Context(), session.OrgID, project, session.UserID, body.Kind, body.PackageID)
	if err != nil {
		reportError(w, err)
		return
	}
	if deps.Broker != nil {
		deps.Broker.Wake(session.OrgID)
	}
	writeJSON(w, http.StatusCreated, report)
}

func getReport(w http.ResponseWriter, r *http.Request, deps Deps) {
	session, ok := reportSession(w, r, deps, false)
	if !ok {
		return
	}
	view, err := deps.Store.ReadReport(r.Context(), session.OrgID, r.PathValue("r"), deps.ReportBuild)
	if err != nil {
		reportError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, view)
}

func getReports(w http.ResponseWriter, r *http.Request, deps Deps) {
	session, ok := memberSession(w, r, deps)
	if !ok {
		return
	}
	project := r.PathValue("id")
	if !uuidPattern.MatchString(project) {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}
	items, err := deps.Store.ListReports(r.Context(), session.OrgID, project)
	if err != nil {
		reportError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, items)
}

func refreshReport(w http.ResponseWriter, r *http.Request, deps Deps) {
	session, ok := reportSession(w, r, deps, true)
	if !ok {
		return
	}
	var body struct {
		UseLastCompleted bool `json:"use_last_completed"`
	}
	if err := readProposalJSON(w, r, deps.MaxBodyBytes, &body); err != nil {
		http.Error(w, "invalid body", http.StatusBadRequest)
		return
	}
	draft, err := deps.Store.RefreshReport(r.Context(), session.OrgID, r.PathValue("r"), session.UserID, deps.ReportBuild, body.UseLastCompleted)
	if err != nil {
		reportError(w, err)
		return
	}
	if deps.Broker != nil {
		deps.Broker.Wake(session.OrgID)
	}
	writeJSON(w, http.StatusOK, draft)
}

func editReport(w http.ResponseWriter, r *http.Request, deps Deps) {
	session, ok := reportSession(w, r, deps, true)
	if !ok {
		return
	}
	var body struct {
		Text    string `json:"text"`
		Version int64  `json:"version"`
	}
	if err := readProposalJSON(w, r, deps.MaxBodyBytes, &body); err != nil {
		http.Error(w, "invalid body", http.StatusBadRequest)
		return
	}
	draft, err := deps.Store.EditReport(r.Context(), session.OrgID, r.PathValue("r"), session.UserID, r.PathValue("target"), body.Text, body.Version)
	if err != nil {
		reportError(w, err)
		return
	}
	if deps.Broker != nil {
		deps.Broker.Wake(session.OrgID)
	}
	writeJSON(w, http.StatusOK, draft)
}

func reportError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, store.ErrNotFound):
		http.Error(w, "not found", http.StatusNotFound)
	case errors.Is(err, reports.ErrReadingIncomplete):
		writeJSON(w, http.StatusConflict, map[string]any{"error": "reading_incomplete", "message": "Reading is pending, failed or needs refresh. Wait for refresh or explicitly use the last completed state.", "choices": []string{"wait", "use_last_completed"}})
	case errors.Is(err, store.ErrReportPackageRetired):
		http.Error(w, "report package is retired", http.StatusConflict)
	case errors.Is(err, store.ErrVersionConflict):
		http.Error(w, "draft changed; reload before editing", http.StatusConflict)
	case errors.Is(err, store.ErrInvalidReport):
		http.Error(w, "invalid or unavailable report operation", http.StatusUnprocessableEntity)
	default:
		http.Error(w, "report operation failed", http.StatusInternalServerError)
	}
}

func resetReportEdit(w http.ResponseWriter, r *http.Request, deps Deps) {
	session, ok := reportSession(w, r, deps, true)
	if !ok {
		return
	}
	var body struct {
		Version int64 `json:"version"`
	}
	if err := readProposalJSON(w, r, deps.MaxBodyBytes, &body); err != nil {
		http.Error(w, "invalid body", http.StatusBadRequest)
		return
	}
	draft, err := deps.Store.ResetReportEdit(r.Context(), session.OrgID, r.PathValue("r"), session.UserID, r.PathValue("target"), body.Version)
	if err != nil {
		reportError(w, err)
		return
	}
	if deps.Broker != nil {
		deps.Broker.Wake(session.OrgID)
	}
	writeJSON(w, http.StatusOK, draft)
}

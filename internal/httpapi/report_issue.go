package httpapi

import (
	"errors"
	"io"
	"net/http"
	"sitewise/internal/files"
	"sitewise/internal/reports/render"
	"sitewise/internal/store"
)

func issueReport(w http.ResponseWriter, r *http.Request, deps Deps) {
	session, ok := reportSession(w, r, deps, true)
	if !ok {
		return
	}
	var body store.IssueOptions
	if err := readProposalJSON(w, r, deps.MaxBodyBytes, &body); err != nil {
		http.Error(w, "invalid body", http.StatusBadRequest)
		return
	}
	issue, err := deps.Store.IssueReport(r.Context(), session.OrgID, r.PathValue("r"), session.UserID, deps.ReportBuild, body, deps.Blobs)
	if err != nil {
		switch {
		case errors.Is(err, render.ErrOverflow):
			writeJSON(w, http.StatusUnprocessableEntity, map[string]string{"error": "report_overflow", "message": err.Error()})
		case errors.Is(err, render.ErrContent):
			writeJSON(w, http.StatusUnprocessableEntity, map[string]string{"error": "report_content", "message": err.Error()})
		case errors.Is(err, store.ErrReportStale):
			writeJSON(w, http.StatusConflict, map[string]string{"error": "report_stale", "message": err.Error()})
		case errors.Is(err, store.ErrReportUnreviewed):
			writeJSON(w, http.StatusUnprocessableEntity, map[string]string{"error": "unreviewed_clauses", "message": err.Error()})
		default:
			reportError(w, err)
		}
		return
	}
	if deps.Broker != nil {
		deps.Broker.Wake(session.OrgID)
	}
	writeJSON(w, http.StatusCreated, issue)
}

func downloadReport(w http.ResponseWriter, r *http.Request, deps Deps) {
	session, ok := reportSession(w, r, deps, false)
	if !ok {
		return
	}
	if !uuidPattern.MatchString(r.PathValue("version")) {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}
	issue, err := deps.Store.ReadIssuedReport(r.Context(), session.OrgID, r.PathValue("r"), r.PathValue("version"))
	if err != nil {
		reportError(w, err)
		return
	}
	if deps.Blobs == nil {
		http.Error(w, "export unavailable", http.StatusServiceUnavailable)
		return
	}
	file, err := deps.Blobs.Open(issue.ExportSHA256)
	if err != nil {
		if errors.Is(err, files.ErrNotFound) {
			http.Error(w, "export file unavailable", http.StatusNotFound)
		} else {
			http.Error(w, "export file unavailable", http.StatusInternalServerError)
		}
		return
	}
	defer file.Close()
	w.Header().Set("Content-Type", "application/pdf")
	w.Header().Set("Content-Disposition", `attachment; filename="sitewise-report.pdf"`)
	w.Header().Set("Cache-Control", "private, no-store")
	_, _ = io.Copy(w, file)
}

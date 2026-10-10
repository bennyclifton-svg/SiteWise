package httpapi

import (
	"encoding/json"
	"errors"
	"net/http"
	"regexp"
	"strings"

	"sitewise/internal/store"
)

var settingKeyPattern = regexp.MustCompile(`^[a-z_]{1,20}(:[a-z0-9_.]{1,100})?$`)

// putReportSetting stores one consultant RFP choice. The body carries the
// value and the draft version the user saw, so a stale editor gets 409.
func putReportSetting(w http.ResponseWriter, r *http.Request, deps Deps) {
	writeReportSetting(w, r, deps, false)
}

// deleteReportSetting removes a choice, which restores the rule.
func deleteReportSetting(w http.ResponseWriter, r *http.Request, deps Deps) {
	writeReportSetting(w, r, deps, true)
}

func writeReportSetting(w http.ResponseWriter, r *http.Request, deps Deps, remove bool) {
	session, ok := reportSession(w, r, deps, true)
	if !ok {
		return
	}
	key := r.PathValue("key")
	if !settingKeyPattern.MatchString(key) {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid_setting", "message": "unknown setting"})
		return
	}
	var body struct {
		Value   json.RawMessage `json:"value"`
		Version int64           `json:"version"`
	}
	if err := readProposalJSON(w, r, deps.MaxBodyBytes, &body); err != nil {
		http.Error(w, "invalid body", http.StatusBadRequest)
		return
	}
	value := body.Value
	if remove {
		value = nil
	} else if len(value) == 0 || string(value) == "null" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid_setting", "message": "a value is required; delete the setting to restore the rule"})
		return
	}
	draft, err := deps.Store.SetReportSetting(r.Context(), session.OrgID, r.PathValue("r"), session.UserID, key, value, body.Version)
	if err != nil {
		if errors.Is(err, store.ErrInvalidSetting) {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid_setting", "message": "that value is not allowed for this setting"})
			return
		}
		reportError(w, err)
		return
	}
	if deps.Broker != nil {
		deps.Broker.Wake(session.OrgID)
	}
	view, err := deps.Store.ReadReport(r.Context(), session.OrgID, r.PathValue("r"), deps.ReportBuild)
	if err != nil {
		reportError(w, err)
		return
	}
	view.Draft = &draft
	writeJSON(w, http.StatusOK, view)
}

var factNamePattern = regexp.MustCompile(`^[a-z_]{1,40}(\.[a-z0-9_]{1,60})?$`)

// putReportFact saves a project fact edited in a consultant RFP to its one
// home in Profile and returns the refreshed view (owner decision D1). A
// null value clears the user's word so the documents show again.
func putReportFact(w http.ResponseWriter, r *http.Request, deps Deps) {
	session, ok := reportSession(w, r, deps, true)
	if !ok {
		return
	}
	name := r.PathValue("fact")
	if !factNamePattern.MatchString(name) {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid_setting", "message": "unknown fact"})
		return
	}
	var body struct {
		Value   *string `json:"value"`
		Version int64   `json:"version"`
	}
	if err := readProposalJSON(w, r, deps.MaxBodyBytes, &body); err != nil {
		http.Error(w, "invalid body", http.StatusBadRequest)
		return
	}
	if body.Value != nil {
		v := strings.TrimSpace(*body.Value)
		body.Value = &v
		if v == "" {
			body.Value = nil
		}
	}
	draft, err := deps.Store.SetReportFact(r.Context(), session.OrgID, r.PathValue("r"), session.UserID, deps.ReportBuild, name, body.Value, body.Version)
	if err != nil {
		if errors.Is(err, store.ErrInvalidSetting) {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid_setting", "message": "that value is not allowed for this fact"})
			return
		}
		reportError(w, err)
		return
	}
	if deps.Broker != nil {
		deps.Broker.Wake(session.OrgID)
	}
	view, err := deps.Store.ReadReport(r.Context(), session.OrgID, r.PathValue("r"), deps.ReportBuild)
	if err != nil {
		reportError(w, err)
		return
	}
	view.Draft = &draft
	writeJSON(w, http.StatusOK, view)
}

// getRFPDisciplines lists the consultant RFP disciplines, so the Reports
// screen can show which packages get a discipline RFP and suggest one for a
// package that has none confirmed. It reads knowledge only.
func getRFPDisciplines(w http.ResponseWriter, r *http.Request, deps Deps) {
	if _, ok := memberSession(w, r, deps); !ok {
		return
	}
	type item struct {
		ID            string   `json:"id"`
		Title         string   `json:"title"`
		Status        string   `json:"status"`
		DisciplineIDs []string `json:"discipline_ids"`
		TitleAliases  []string `json:"title_aliases"`
	}
	out := []item{}
	if deps.Knowledge != nil {
		for _, id := range deps.Knowledge.RFPDisciplines() {
			d, _ := deps.Knowledge.RFP(id)
			out = append(out, item{ID: d.ID, Title: d.Title, Status: d.Status, DisciplineIDs: d.DisciplineIDs, TitleAliases: d.TitleAliases})
		}
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": out})
}

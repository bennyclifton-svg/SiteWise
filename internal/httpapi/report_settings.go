package httpapi

import (
	"encoding/json"
	"errors"
	"mime"
	"net/http"
	"regexp"
	"strings"

	"sitewise/internal/export"
	"sitewise/internal/knowledge"
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
		// View names the combined engagement's report the editor shows. A
		// member's setting then returns that report, recomposed in the same
		// transaction, so the editor needs one round trip.
		View string `json:"view"`
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
	draft, group, err := deps.Store.SetReportSettingInGroup(r.Context(), session.OrgID, r.PathValue("r"), session.UserID, key, value, body.Version)
	answerReportSetting(w, r, deps, session.OrgID, draft, group, body.View, err)
}

// postReportSettings stores several choices in one transaction, such as
// scope lines deleted together. Each change's null value deletes it.
func postReportSettings(w http.ResponseWriter, r *http.Request, deps Deps) {
	session, ok := reportSession(w, r, deps, true)
	if !ok {
		return
	}
	var body struct {
		Changes []store.SettingChange `json:"changes"`
		Version int64                 `json:"version"`
		View    string                `json:"view"`
	}
	if err := readProposalJSON(w, r, deps.MaxBodyBytes, &body); err != nil {
		http.Error(w, "invalid body", http.StatusBadRequest)
		return
	}
	if len(body.Changes) == 0 || len(body.Changes) > store.MaxSettingChanges {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid_setting", "message": "send between one and 200 changes"})
		return
	}
	for i, c := range body.Changes {
		if !settingKeyPattern.MatchString(c.Key) {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid_setting", "message": "unknown setting"})
			return
		}
		if len(c.Value) == 0 || string(c.Value) == "null" {
			body.Changes[i].Value = nil
		}
	}
	draft, group, err := deps.Store.SetReportSettingsInGroup(r.Context(), session.OrgID, r.PathValue("r"), session.UserID, body.Changes, body.Version)
	answerReportSetting(w, r, deps, session.OrgID, draft, group, body.View, err)
}

// answerReportSetting returns the report a setting write changed: the
// combined engagement's when the editor shows it, else this one's.
func answerReportSetting(w http.ResponseWriter, r *http.Request, deps Deps, org string, draft store.ReportDraft, group, show string, err error) {
	if err != nil {
		if errors.Is(err, store.ErrInvalidSetting) {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid_setting", "message": "that value is not allowed for this setting"})
			return
		}
		reportError(w, err)
		return
	}
	if deps.Broker != nil {
		deps.Broker.Wake(org)
	}
	// Only the group the store recomposed may be returned instead.
	if show != "" && show == group && group != r.PathValue("r") {
		view, err := deps.Store.ReadReport(r.Context(), org, group, deps.ReportBuild)
		if err != nil {
			reportError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, view)
		return
	}
	view, err := deps.Store.ReadReport(r.Context(), org, r.PathValue("r"), deps.ReportBuild)
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
			// The list names the consultant RFPs a title can suggest; the
			// general fallbacks and trade RFTs are chosen by discipline.
			if d.Works() || d.ID == knowledge.GeneralConsultantRFP {
				continue
			}
			out = append(out, item{ID: d.ID, Title: d.Title, Status: d.Status, DisciplineIDs: d.DisciplineIDs, TitleAliases: d.TitleAliases})
		}
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": out})
}

// getReportExport downloads the current draft as a PDF or a Word document
// laid out as the screen shows it (owner direction, 8 November 2026). It
// reads only; no Jev call. An issued RFP's frozen PDF is a separate file.
func getReportExport(w http.ResponseWriter, r *http.Request, deps Deps) {
	session, ok := reportSession(w, r, deps, false)
	if !ok {
		return
	}
	format := r.URL.Query().Get("format")
	if format != "pdf" && format != "docx" {
		http.Error(w, "format must be pdf or docx", http.StatusBadRequest)
		return
	}
	data, err := deps.Store.ReadReportExport(r.Context(), session.OrgID, r.PathValue("r"))
	if err != nil {
		reportError(w, err)
		return
	}
	doc := export.FromSections(data.Title, data.Project, data.Date, data.Sections)
	var body []byte
	contentType := "application/pdf"
	if format == "pdf" {
		body, err = export.PDF(doc)
	} else {
		contentType = "application/vnd.openxmlformats-officedocument.wordprocessingml.document"
		body, err = export.DOCX(doc)
	}
	if errors.Is(err, export.ErrContent) {
		writeJSON(w, http.StatusUnprocessableEntity, map[string]string{"error": "export_content", "message": err.Error()})
		return
	}
	if err != nil {
		reportError(w, err)
		return
	}
	name := safeFileName(strings.TrimSpace(data.Project+" "+data.Title)) + "." + format
	w.Header().Set("Content-Type", contentType)
	w.Header().Set("Content-Disposition", mime.FormatMediaType("attachment", map[string]string{"filename": name}))
	w.Header().Set("Cache-Control", "no-store")
	_, _ = w.Write(body)
}

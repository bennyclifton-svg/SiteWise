package httpapi

import (
	"errors"
	"net/http"

	"sitewise/internal/store"
)

// Stage-sync routes (program plan docs/plans/2026-10-18-program.md 7.6 and
// 8): GET shows a package's or an RFP's differences from the program's
// current stage plan, by stable stage id; POST {choice, revision} applies
// Update or Keep. Every route is scoped to the session's org; another org's
// or another project's records are a bare 404. revision pins the program
// stage revision the user saw: a newer one is 409 with the current view. No
// route calls Jev.

const pathStageSyncWrite = "stage_sync_write"

type stageSyncBody struct {
	Choice   string `json:"choice"`
	Revision int64  `json:"revision"`
}

func readStageSyncBody(w http.ResponseWriter, r *http.Request, d Deps) (stageSyncBody, bool) {
	var body stageSyncBody
	if readProposalJSON(w, r, d.MaxBodyBytes, &body) != nil {
		http.Error(w, "invalid body", http.StatusBadRequest)
		return body, false
	}
	if body.Choice != store.StageSyncUpdate && body.Choice != store.StageSyncKeep {
		writeJSON(w, http.StatusUnprocessableEntity, map[string]string{"error": "Choice is update or keep.", "field": "choice"})
		return body, false
	}
	if body.Revision < 1 {
		writeJSON(w, http.StatusUnprocessableEntity, map[string]string{"error": "The stage revision is required.", "field": "revision"})
		return body, false
	}
	return body, true
}

// stageSyncError maps store errors; a conflict carries the current view.
func stageSyncError(w http.ResponseWriter, current store.StageSync, err error) {
	switch {
	case errors.Is(err, store.ErrNotFound):
		http.Error(w, "not found", http.StatusNotFound)
	case errors.Is(err, store.ErrVersionConflict):
		writeJSON(w, http.StatusConflict, map[string]any{"error": "The program's stages changed; showing the latest differences.", "current": current})
	case errors.Is(err, store.ErrInvalidPackage), errors.Is(err, store.ErrInvalidReport):
		http.Error(w, "invalid stage sync", http.StatusUnprocessableEntity)
	default:
		http.Error(w, "stage sync failed", http.StatusInternalServerError)
	}
}

// packageSyncSession checks the member, the origin on a write and that
// both ids are well formed and the project is the caller org's.
func packageSyncSession(w http.ResponseWriter, r *http.Request, d Deps, write bool) (store.Session, bool) {
	if write {
		return packageEditSession(w, r, d)
	}
	s, ok := memberSession(w, r, d)
	if !ok {
		return s, false
	}
	for _, name := range []string{"id", "pkg"} {
		if !uuidPattern.MatchString(r.PathValue(name)) {
			http.Error(w, "not found", http.StatusNotFound)
			return s, false
		}
	}
	return s, true
}

func getPackageStageSync(w http.ResponseWriter, r *http.Request, d Deps) {
	s, ok := packageSyncSession(w, r, d, false)
	if !ok {
		return
	}
	out, err := d.Store.ReadPackageStageSync(r.Context(), s.OrgID, r.PathValue("id"), r.PathValue("pkg"))
	if err != nil {
		stageSyncError(w, out, err)
		return
	}
	writeJSON(w, http.StatusOK, out)
}

func postPackageStageSync(w http.ResponseWriter, r *http.Request, d Deps) {
	s, ok := packageSyncSession(w, r, d, true)
	if !ok {
		return
	}
	body, ok := readStageSyncBody(w, r, d)
	if !ok {
		return
	}
	out, err := d.Store.SyncPackageStages(r.Context(), s.OrgID, r.PathValue("id"), r.PathValue("pkg"), s.UserID, body.Choice, body.Revision)
	if err != nil {
		stageSyncError(w, out, err)
		return
	}
	if d.Broker != nil {
		d.Broker.Wake(s.OrgID)
	}
	writeJSON(w, http.StatusOK, out)
}

func getReportStageSync(w http.ResponseWriter, r *http.Request, d Deps) {
	s, ok := reportSession(w, r, d, false)
	if !ok {
		return
	}
	out, err := d.Store.ReadReportStageSync(r.Context(), s.OrgID, r.PathValue("r"))
	if err != nil {
		stageSyncError(w, out, err)
		return
	}
	writeJSON(w, http.StatusOK, out)
}

func postReportStageSync(w http.ResponseWriter, r *http.Request, d Deps) {
	s, ok := reportSession(w, r, d, true)
	if !ok {
		return
	}
	body, ok := readStageSyncBody(w, r, d)
	if !ok {
		return
	}
	out, err := d.Store.SyncReportStages(r.Context(), s.OrgID, r.PathValue("r"), s.UserID, d.ReportBuild, body.Choice, body.Revision)
	if err != nil {
		stageSyncError(w, out, err)
		return
	}
	if d.Broker != nil {
		d.Broker.Wake(s.OrgID)
	}
	writeJSON(w, http.StatusOK, out)
}

// ensureProgram creates the program from a downstream surface that needs
// stages first (plan 7.6). It runs after the request's own write has
// succeeded, so ownership is already proven. A failure is logged and never
// fails that write, which has committed; the next surface tries again.
func ensureProgram(r *http.Request, d Deps, org, project, surface string) {
	if d.Knowledge == nil {
		return
	}
	if err := d.Store.EnsureProgram(r.Context(), org, project, surface, d.Knowledge); err != nil && d.Log != nil {
		d.Log.Printf("program prepare from %s: %v", surface, err)
	}
}

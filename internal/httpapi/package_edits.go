package httpapi

import (
	"net/http"
	"sitewise/internal/procurement"
	"sitewise/internal/store"
)

func packageEditSession(w http.ResponseWriter, r *http.Request, deps Deps) (store.Session, bool) {
	if !originOK(r, deps.PublicOrigin) {
		http.Error(w, "origin rejected", http.StatusForbidden)
		return store.Session{}, false
	}
	session, ok := memberSession(w, r, deps)
	if !ok {
		return session, false
	}
	for _, name := range []string{"id", "pkg"} {
		if !uuidPattern.MatchString(r.PathValue(name)) {
			http.Error(w, "not found", http.StatusNotFound)
			return session, false
		}
	}
	if stage := r.PathValue("stage"); stage != "" && !uuidPattern.MatchString(stage) {
		http.Error(w, "not found", http.StatusNotFound)
		return session, false
	}
	if _, err := deps.Store.GetProject(r.Context(), session.OrgID, r.PathValue("id")); err != nil {
		packageError(w, err)
		return session, false
	}
	return session, true
}

func patchPackage(w http.ResponseWriter, r *http.Request, deps Deps) {
	session, ok := packageEditSession(w, r, deps)
	if !ok {
		return
	}
	var patch procurement.PackagePatch
	if err := readProposalJSON(w, r, deps.MaxBodyBytes, &patch); err != nil {
		http.Error(w, "invalid body", http.StatusBadRequest)
		return
	}
	p, err := deps.Store.PatchPackage(r.Context(), session.OrgID, r.PathValue("id"), r.PathValue("pkg"), session.UserID, patch)
	if err != nil {
		packageError(w, err)
		return
	}
	if deps.Broker != nil {
		deps.Broker.Wake(session.OrgID)
	}
	writeJSON(w, http.StatusOK, p)
}

func postPackageStage(w http.ResponseWriter, r *http.Request, deps Deps) {
	session, ok := packageEditSession(w, r, deps)
	if !ok {
		return
	}
	var body struct {
		StageID       string `json:"stage_id"`
		Label         string `json:"label"`
		Ordinal       int    `json:"ordinal"`
		NovationPhase string `json:"novation_phase"`
	}
	if err := readProposalJSON(w, r, deps.MaxBodyBytes, &body); err != nil {
		http.Error(w, "invalid body", http.StatusBadRequest)
		return
	}
	if body.NovationPhase == "" {
		body.NovationPhase = "none"
	}
	stage, err := deps.Store.CreatePackageStage(r.Context(), session.OrgID, r.PathValue("id"), r.PathValue("pkg"), session.UserID, procurement.Stage{StageID: body.StageID, Label: body.Label, Ordinal: body.Ordinal, NovationPhase: body.NovationPhase})
	if err != nil {
		packageError(w, err)
		return
	}
	if deps.Broker != nil {
		deps.Broker.Wake(session.OrgID)
	}
	writeJSON(w, http.StatusCreated, stage)
}

func patchPackageStage(w http.ResponseWriter, r *http.Request, deps Deps) {
	session, ok := packageEditSession(w, r, deps)
	if !ok {
		return
	}
	var patch procurement.StagePatch
	if err := readProposalJSON(w, r, deps.MaxBodyBytes, &patch); err != nil {
		http.Error(w, "invalid body", http.StatusBadRequest)
		return
	}
	stage, err := deps.Store.PatchPackageStage(r.Context(), session.OrgID, r.PathValue("id"), r.PathValue("pkg"), r.PathValue("stage"), session.UserID, patch)
	if err != nil {
		packageError(w, err)
		return
	}
	if deps.Broker != nil {
		deps.Broker.Wake(session.OrgID)
	}
	writeJSON(w, http.StatusOK, stage)
}

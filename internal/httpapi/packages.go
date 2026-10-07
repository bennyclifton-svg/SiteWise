package httpapi

import (
	"errors"
	"net/http"
	"strings"

	"github.com/jackc/pgx/v5/pgconn"
	"sitewise/internal/procurement"
	"sitewise/internal/store"
)

func getPackages(w http.ResponseWriter, r *http.Request, deps Deps) {
	session, ok := memberSession(w, r, deps)
	if !ok {
		return
	}
	project := r.PathValue("id")
	if !uuidPattern.MatchString(project) {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}
	items, suggestions, err := deps.Store.ReadPackageOverview(r.Context(), session.OrgID, project)
	if err != nil {
		packageError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, struct {
		ProjectID   string                          `json:"project_id"`
		Items       []procurement.Package           `json:"items"`
		Suggestions []procurement.PackageSuggestion `json:"suggestions"`
	}{project, items, suggestions})
}

func postPackage(w http.ResponseWriter, r *http.Request, deps Deps) {
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
		packageError(w, err)
		return
	}
	var body struct {
		Kind            string `json:"kind"`
		WorksScope      string `json:"works_scope"`
		DisciplineID    string `json:"discipline_id"`
		Title           string `json:"title"`
		Novation        bool   `json:"novation"`
		LifecycleStatus string `json:"lifecycle_status"`
		Stages          []struct {
			StageID       string `json:"stage_id"`
			Label         string `json:"label"`
			Ordinal       int    `json:"ordinal"`
			NovationPhase string `json:"novation_phase"`
		} `json:"stages"`
	}
	if err := readProposalJSON(w, r, deps.MaxBodyBytes, &body); err != nil {
		http.Error(w, "invalid body", http.StatusBadRequest)
		return
	}
	if len(body.Stages) > 100 {
		http.Error(w, "too many stages", http.StatusUnprocessableEntity)
		return
	}
	if body.LifecycleStatus == "" {
		body.LifecycleStatus = "planned"
	}
	p := procurement.Package{Kind: body.Kind, WorksScope: body.WorksScope, DisciplineID: body.DisciplineID, Title: strings.TrimSpace(body.Title), Novation: body.Novation, LifecycleStatus: body.LifecycleStatus}
	if body.Stages != nil {
		p.Stages = []procurement.Stage{}
		for _, s := range body.Stages {
			phase := s.NovationPhase
			if phase == "" {
				phase = "none"
			}
			p.Stages = append(p.Stages, procurement.Stage{StageID: s.StageID, Label: strings.TrimSpace(s.Label), Ordinal: s.Ordinal, NovationPhase: phase})
		}
	}
	p, err := deps.Store.CreatePackage(r.Context(), session.OrgID, project, session.UserID, p)
	if err != nil {
		packageError(w, err)
		return
	}
	if deps.Broker != nil {
		deps.Broker.Wake(session.OrgID)
	}
	writeJSON(w, http.StatusCreated, p)
}

func packageError(w http.ResponseWriter, err error) {
	var pgErr *pgconn.PgError
	switch {
	case errors.Is(err, store.ErrNotFound):
		http.Error(w, "not found", http.StatusNotFound)
	case errors.Is(err, store.ErrInvalidPackage):
		http.Error(w, err.Error(), http.StatusUnprocessableEntity)
	case errors.Is(err, store.ErrVersionConflict):
		http.Error(w, "package changed", http.StatusConflict)
	case errors.As(err, &pgErr) && pgErr.Code == "23505":
		http.Error(w, "duplicate package stage, responsibility or proposal", http.StatusConflict)
	default:
		http.Error(w, "package operation failed", http.StatusInternalServerError)
	}
}

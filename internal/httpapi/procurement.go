package httpapi

import (
	"errors"
	"net/http"
	"strings"

	"github.com/jackc/pgx/v5/pgconn"

	"sitewise/internal/procurement"
	"sitewise/internal/store"
)

// disciplines adapts the filing discipline list for procurement.
func disciplines(deps Deps) []procurement.Discipline {
	out := make([]procurement.Discipline, 0, len(deps.Catalog.Disciplines))
	for _, d := range deps.Catalog.Disciplines {
		out = append(out, procurement.Discipline{ID: d.ID, Label: d.Label, Aliases: d.Aliases})
	}
	return out
}

// postProcurementRow saves the package for a discipline chosen in the
// Procurement table. Only the id is trusted; title and kind come from the
// discipline list.
func postProcurementRow(w http.ResponseWriter, r *http.Request, deps Deps) {
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
	var body struct {
		DisciplineID string `json:"discipline_id"`
	}
	if err := readProposalJSON(w, r, deps.MaxBodyBytes, &body); err != nil || body.DisciplineID == "" || len(body.DisciplineID) > 100 {
		http.Error(w, "invalid body", http.StatusBadRequest)
		return
	}
	var p procurement.Package
	found := false
	for _, d := range deps.Catalog.Disciplines {
		if d.ID == body.DisciplineID {
			p, found = procurement.DisciplinePackage(d.ID, d.Label)
			break
		}
	}
	if !found {
		http.Error(w, "unknown or unsupported discipline", http.StatusUnprocessableEntity)
		return
	}
	p, err := deps.Store.CreateDisciplinePackage(r.Context(), session.OrgID, project, session.UserID, p)
	if err != nil {
		packageError(w, err)
		return
	}
	if deps.Broker != nil {
		deps.Broker.Wake(session.OrgID)
	}
	writeJSON(w, http.StatusCreated, p)
}

// postProcurementDismiss removes a suggested row. Only the key is trusted;
// the store recomputes the suggestion and refuses one not offered now.
func postProcurementDismiss(w http.ResponseWriter, r *http.Request, deps Deps) {
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
	var body struct {
		ProposalKey string `json:"proposal_key"`
	}
	if err := readProposalJSON(w, r, deps.MaxBodyBytes, &body); err != nil || !strings.HasPrefix(body.ProposalKey, "default:services:") || len(body.ProposalKey) > 300 {
		http.Error(w, "invalid body", http.StatusBadRequest)
		return
	}
	if err := deps.Store.DismissPackageSuggestion(r.Context(), session.OrgID, project, session.UserID, body.ProposalKey); err != nil {
		packageError(w, err)
		return
	}
	if deps.Broker != nil {
		deps.Broker.Wake(session.OrgID)
	}
	w.WriteHeader(http.StatusNoContent)
}

// postProcurementCreate makes a row's RFP or RFT in one request; the report
// kind follows the package kind, never the client.
func postProcurementCreate(w http.ResponseWriter, r *http.Request, deps Deps) {
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
	var body struct {
		PackageID   string `json:"package_id"`
		ProposalKey string `json:"proposal_key"`
	}
	if err := readProposalJSON(w, r, deps.MaxBodyBytes, &body); err != nil || (body.PackageID == "") == (body.ProposalKey == "") || len(body.ProposalKey) > 300 || body.PackageID != "" && !uuidPattern.MatchString(body.PackageID) {
		http.Error(w, "invalid body", http.StatusBadRequest)
		return
	}
	discipline := ""
	if body.ProposalKey != "" {
		discipline = procurement.ProposalDiscipline(body.ProposalKey, disciplines(deps))
	}
	p, report, created, err := deps.Store.ProcurementCreate(r.Context(), session.OrgID, project, session.UserID, body.PackageID, body.ProposalKey, discipline)
	if err != nil {
		procurementCreateError(w, err)
		return
	}
	if report.CurrentDraftVersionID == "" {
		draft, err := deps.Store.EnsureReportDraft(r.Context(), session.OrgID, report.ID, session.UserID, deps.ReportBuild)
		if err != nil {
			reportError(w, err)
			return
		}
		report.CurrentDraftVersionID = draft.ID
	}
	status := http.StatusOK
	if created {
		status = http.StatusCreated
	}
	// A repeat on an accepted head contract can still add scope lines, so
	// listeners are woken whenever the request may have written.
	if (created || body.ProposalKey != "") && deps.Broker != nil {
		deps.Broker.Wake(session.OrgID)
	}
	writeJSON(w, status, struct {
		Package procurement.Package `json:"package"`
		Report  store.Report        `json:"report"`
	}{p, report})
}

// procurementCreateError maps the package half of Create (accepting a
// proposal) like the Packages routes, and the rest like the report routes.
func procurementCreateError(w http.ResponseWriter, err error) {
	var pgErr *pgconn.PgError
	if errors.Is(err, store.ErrInvalidPackage) || errors.Is(err, store.ErrDuplicatePackage) || errors.As(err, &pgErr) && pgErr.Code == "23505" {
		packageError(w, err)
		return
	}
	reportError(w, err)
}

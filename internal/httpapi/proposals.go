package httpapi

import (
	"encoding/json"
	"errors"
	"github.com/jackc/pgx/v5/pgconn"
	"io"
	"net/http"
	"regexp"
	"unicode/utf8"

	"sitewise/internal/store"
	"sitewise/internal/works"
)

var proposalFingerprintPattern = regexp.MustCompile(`^[0-9a-f]{64}$`)

func getProposals(w http.ResponseWriter, r *http.Request, deps Deps) {
	session, ok := memberSession(w, r, deps)
	if !ok {
		return
	}
	project := r.PathValue("id")
	if !uuidPattern.MatchString(project) {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}
	show := r.URL.Query().Get("show")
	if show != "" && show != "all" {
		http.Error(w, "show must be all or omitted", http.StatusBadRequest)
		return
	}
	items, err := deps.Store.ReadProposals(r.Context(), session.OrgID, project)
	if err != nil {
		proposalError(w, err)
		return
	}
	total := len(items)
	if show != "all" {
		count := deps.ProposalShowCount
		if count <= 0 {
			count = works.DefaultProposalShowCount
		}
		ranked := make([]works.Proposal, len(items))
		for i := range items {
			ranked[i] = items[i].Proposal
		}
		visible := works.VisibleProposals(ranked, count)
		keys := make(map[string]bool, len(visible))
		for _, p := range visible {
			keys[p.Key] = true
		}
		selected := []store.ProposalView{}
		for _, p := range items {
			if keys[p.Key] {
				selected = append(selected, p)
			}
		}
		items = selected
	}
	writeJSON(w, http.StatusOK, struct {
		ProjectID string               `json:"project_id"`
		Items     []store.ProposalView `json:"items"`
		Total     int                  `json:"total"`
	}{project, items, total})
}

func acceptProposal(w http.ResponseWriter, r *http.Request, deps Deps) {
	writeProposalDecision(w, r, deps, true)
}

func dismissProposal(w http.ResponseWriter, r *http.Request, deps Deps) {
	writeProposalDecision(w, r, deps, false)
}

func undoProposalDecision(w http.ResponseWriter, r *http.Request, deps Deps) {
	if !originOK(r, deps.PublicOrigin) {
		http.Error(w, "origin rejected", http.StatusForbidden)
		return
	}
	session, ok := memberSession(w, r, deps)
	if !ok {
		return
	}
	project, key := r.PathValue("id"), r.PathValue("key")
	if !uuidPattern.MatchString(project) || key == "" || len(key) > 1000 || !utf8.ValidString(key) {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}
	if _, err := deps.Store.GetProject(r.Context(), session.OrgID, project); err != nil {
		proposalError(w, err)
		return
	}
	var body struct {
		Version int64 `json:"version"`
	}
	if err := readProposalJSON(w, r, deps.MaxBodyBytes, &body); err != nil {
		http.Error(w, "invalid body", http.StatusBadRequest)
		return
	}
	if err := deps.Store.UndoProposalDecision(r.Context(), session.OrgID, project, key, session.UserID, body.Version); err != nil {
		proposalError(w, err)
		return
	}
	if deps.Broker != nil {
		deps.Broker.Wake(session.OrgID)
	}
	w.WriteHeader(http.StatusNoContent)
}

func writeProposalDecision(w http.ResponseWriter, r *http.Request, deps Deps, accept bool) {
	if !originOK(r, deps.PublicOrigin) {
		http.Error(w, "origin rejected", http.StatusForbidden)
		return
	}
	session, ok := memberSession(w, r, deps)
	if !ok {
		return
	}
	project, key := r.PathValue("id"), r.PathValue("key")
	if !uuidPattern.MatchString(project) || key == "" || len(key) > 1000 || !utf8.ValidString(key) {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}
	// Resolve ownership before parsing the body, matching other project writes.
	if _, err := deps.Store.GetProject(r.Context(), session.OrgID, project); err != nil {
		proposalError(w, err)
		return
	}
	var fingerprint, rationale string
	var scopeChoice store.ScopeAcceptance
	if accept {
		var body struct {
			InputsFingerprint string `json:"inputs_fingerprint"`
			store.ScopeAcceptance
		}
		if err := readProposalJSON(w, r, deps.MaxBodyBytes, &body); err != nil {
			http.Error(w, "invalid body", http.StatusBadRequest)
			return
		}
		fingerprint = body.InputsFingerprint
		scopeChoice = body.ScopeAcceptance
	} else {
		var body struct {
			InputsFingerprint string `json:"inputs_fingerprint"`
			Rationale         string `json:"rationale"`
		}
		if err := readProposalJSON(w, r, deps.MaxBodyBytes, &body); err != nil {
			http.Error(w, "invalid body", http.StatusBadRequest)
			return
		}
		fingerprint, rationale = body.InputsFingerprint, body.Rationale
	}
	if !proposalFingerprintPattern.MatchString(fingerprint) || !utf8.ValidString(rationale) || utf8.RuneCountInString(rationale) > 200 {
		http.Error(w, "invalid proposal decision", http.StatusUnprocessableEntity)
		return
	}
	if accept {
		if scopeChoice.Role == "" {
			item, err := deps.Store.AcceptDeliveryProposal(r.Context(), session.OrgID, project, key, session.UserID, fingerprint, store.DeliveryAcceptance{PackageID: scopeChoice.PackageID, WorkItemID: scopeChoice.WorkItemID, StageID: scopeChoice.StageID})
			if !errors.Is(err, store.ErrProposalUnavailable) {
				if err != nil {
					proposalError(w, err)
					return
				}
				if deps.Broker != nil {
					deps.Broker.Wake(session.OrgID)
				}
				writeJSON(w, http.StatusOK, item)
				return
			}
		}
		if scopeChoice.PackageID != "" {
			if !uuidPattern.MatchString(scopeChoice.PackageID) {
				http.Error(w, "invalid package", http.StatusUnprocessableEntity)
				return
			}
			item, err := deps.Store.AcceptScopeProposal(r.Context(), session.OrgID, project, key, session.UserID, fingerprint, scopeChoice)
			if err != nil {
				proposalError(w, err)
				return
			}
			if deps.Broker != nil {
				deps.Broker.Wake(session.OrgID)
			}
			writeJSON(w, http.StatusOK, item)
			return
		}
		if scopeChoice.WorkItemID != "" || scopeChoice.Role != "" || scopeChoice.StageID != "" {
			http.Error(w, "package required for assignment", http.StatusUnprocessableEntity)
			return
		}
		item, err := deps.Store.AcceptProposal(r.Context(), session.OrgID, project, key, session.UserID, fingerprint)
		if errors.Is(err, store.ErrProposalUnavailable) {
			p, err := deps.Store.AcceptPackageProposal(r.Context(), session.OrgID, project, key, session.UserID, fingerprint)
			if err != nil {
				proposalError(w, err)
				return
			}
			if deps.Broker != nil {
				deps.Broker.Wake(session.OrgID)
			}
			writeJSON(w, http.StatusOK, p)
			return
		}
		if err != nil {
			proposalError(w, err)
			return
		}
		if deps.Broker != nil {
			deps.Broker.Wake(session.OrgID)
		}
		writeJSON(w, http.StatusOK, item)
		return
	}
	decision, err := deps.Store.DismissProposal(r.Context(), session.OrgID, project, key, session.UserID, fingerprint, rationale)
	if err != nil {
		proposalError(w, err)
		return
	}
	if deps.Broker != nil {
		deps.Broker.Wake(session.OrgID)
	}
	writeJSON(w, http.StatusOK, decision)
}

func readProposalJSON(w http.ResponseWriter, r *http.Request, limit int64, body any) error {
	r.Body = http.MaxBytesReader(w, r.Body, limit)
	defer r.Body.Close()
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	if err := dec.Decode(body); err != nil {
		return err
	}
	if err := dec.Decode(new(any)); err != io.EOF {
		return errors.New("expected one JSON object")
	}
	return nil
}

func proposalError(w http.ResponseWriter, err error) {
	var pgErr *pgconn.PgError
	switch {
	case errors.As(err, &pgErr) && pgErr.Code == "23505":
		http.Error(w, "responsibility already exists in this package", http.StatusConflict)
	case errors.Is(err, store.ErrNotFound):
		http.Error(w, "not found", http.StatusNotFound)
	case errors.Is(err, store.ErrVersionConflict):
		http.Error(w, "proposal inputs or decision changed", http.StatusConflict)
	case errors.Is(err, store.ErrProposalUndoBlocked):
		http.Error(w, "created record has been edited or used; undo is unavailable", http.StatusConflict)
	case errors.Is(err, store.ErrProposalUnavailable):
		http.Error(w, "proposal target is not available for acceptance", http.StatusUnprocessableEntity)
	case errors.Is(err, store.ErrInvalidProposalDecision), errors.Is(err, store.ErrInvalidWork), errors.Is(err, store.ErrInvalidPackage), errors.Is(err, store.ErrInvalidDelivery):
		http.Error(w, "invalid proposal decision", http.StatusUnprocessableEntity)
	default:
		http.Error(w, "proposal operation failed", http.StatusInternalServerError)
	}
}

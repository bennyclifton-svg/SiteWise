package httpapi

import (
	"errors"
	"fmt"
	"io"
	"net/http"

	"sitewise/internal/files"
	"sitewise/internal/specifications"
	"sitewise/internal/store"
)

func specificationError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, store.ErrNotFound):
		http.Error(w, "not found", http.StatusNotFound)
	case errors.Is(err, store.ErrVersionConflict):
		http.Error(w, "Specification changed. Reload before applying this decision.", http.StatusConflict)
	case errors.Is(err, store.ErrInvalidSpecification):
		http.Error(w, err.Error(), http.StatusUnprocessableEntity)
	default:
		http.Error(w, "Specification request failed.", http.StatusInternalServerError)
	}
}

func getSpecifications(w http.ResponseWriter, r *http.Request, d Deps) {
	session, ok := memberSession(w, r, d)
	if !ok {
		return
	}
	project := r.PathValue("id")
	if !uuidPattern.MatchString(project) {
		http.NotFound(w, r)
		return
	}
	snap, err := d.Store.ReadSpecifications(r.Context(), session.OrgID, project)
	if err != nil {
		specificationError(w, err)
		return
	}
	systems := []map[string]string{}
	imports, err := d.Store.ListSpecificationImports(r.Context(), session.OrgID, project)
	if err != nil {
		specificationError(w, err)
		return
	}
	if d.Knowledge != nil {
		for _, s := range d.Knowledge.BuildingOrderSystems() {
			if s.Status != "deprecated" {
				systems = append(systems, map[string]string{"id": s.ID, "label": s.Label})
			}
		}
	}
	writeJSON(w, http.StatusOK, map[string]any{"snapshot": snap, "attributes": specifications.Definitions(), "systems": systems, "sections": specifications.Sections(d.Knowledge, snap), "rows": specifications.Assemble(snap), "imports": imports})
}

func writeSpecification(w http.ResponseWriter, r *http.Request, d Deps) {
	if !originOK(r, d.PublicOrigin) {
		http.Error(w, "origin rejected", http.StatusForbidden)
		return
	}
	session, ok := memberSession(w, r, d)
	if !ok {
		return
	}
	project := r.PathValue("id")
	if !uuidPattern.MatchString(project) {
		http.NotFound(w, r)
		return
	}
	if _, err := d.Store.GetProject(r.Context(), session.OrgID, project); err != nil {
		specificationError(w, err)
		return
	}
	var out any
	var err error
	status := http.StatusCreated
	decode := func(v any) bool {
		if readProposalJSON(w, r, d.MaxBodyBytes, v) != nil {
			http.Error(w, "invalid specification body", http.StatusBadRequest)
			return false
		}
		return true
	}
	switch r.Pattern {
	case "POST /projects/{id}/specifications/cells":
		var in store.SpecificationCell
		if !decode(&in) {
			return
		}
		out, err = d.Store.SaveSpecificationCell(r.Context(), session.OrgID, project, session.UserID, in)
	case "POST /projects/{id}/specifications/types/{type}/link":
		var in struct {
			Target  string `json:"target_type_id"`
			Version int64  `json:"version"`
		}
		if !decode(&in) {
			return
		}
		err = d.Store.LinkSpecificationCandidate(r.Context(), session.OrgID, project, session.UserID, r.PathValue("type"), in.Target, in.Version)
		out = map[string]string{"status": "linked"}
	case "POST /projects/{id}/specifications/update":
		var count int64
		count, err = d.Store.RequestSpecificationUpdate(r.Context(), session.OrgID, project, session.UserID)
		out = map[string]any{"queued": count}
		status = http.StatusAccepted
	case "POST /projects/{id}/specifications/coverage":
		var in specifications.Coverage
		if !decode(&in) {
			return
		}
		out, err = d.Store.SaveSpecificationCoverage(r.Context(), session.OrgID, project, session.UserID, in)
	case "POST /projects/{id}/specifications/types/{type}/adopt":
		var in struct {
			Version int64  `json:"version"`
			Reason  string `json:"reason"`
		}
		if !decode(&in) {
			return
		}
		out, err = d.Store.AdoptSpecification(r.Context(), session.OrgID, project, session.UserID, r.PathValue("type"), in.Version, in.Reason)
	case "POST /projects/{id}/specifications/imports":
		var in struct {
			DocumentID   string `json:"document_id"`
			TargetTypeID string `json:"target_type_id"`
		}
		if !decode(&in) {
			return
		}
		err = d.Store.RequestSpecificationImport(r.Context(), session.OrgID, project, session.UserID, in.DocumentID, in.TargetTypeID)
		out = map[string]string{"status": "queued"}
		status = http.StatusAccepted
	case "POST /projects/{id}/specifications/types", "PUT /projects/{id}/specifications/types/{type}":
		var in specifications.Type
		if !decode(&in) {
			return
		}
		in.ID = r.PathValue("type")
		if in.ID != "" {
			status = http.StatusOK
		}
		out, err = d.Store.SaveSpecificationType(r.Context(), session.OrgID, project, session.UserID, in)
	case "POST /projects/{id}/specifications/applications", "PUT /projects/{id}/specifications/applications/{application}":
		var in specifications.Application
		if !decode(&in) {
			return
		}
		in.ID = r.PathValue("application")
		if in.ID != "" {
			status = http.StatusOK
		}
		out, err = d.Store.SaveSpecificationApplication(r.Context(), session.OrgID, project, session.UserID, in)
	case "POST /projects/{id}/specifications/assertions":
		var in specifications.Assertion
		if !decode(&in) {
			return
		}
		out, err = d.Store.AddSpecificationAssertion(r.Context(), session.OrgID, project, session.UserID, in)
	case "POST /projects/{id}/specifications/assertions/{assertion}/review":
		var in store.SpecificationReview
		if !decode(&in) {
			return
		}
		out, err = d.Store.ReviewSpecificationAssertion(r.Context(), session.OrgID, project, session.UserID, r.PathValue("assertion"), in)
		status = http.StatusOK
	case "POST /projects/{id}/specifications/relationships":
		var in specifications.Relationship
		if !decode(&in) {
			return
		}
		out, err = d.Store.AddSpecificationRelationship(r.Context(), session.OrgID, project, session.UserID, in)
	case "POST /projects/{id}/specifications/issues":
		var in struct {
			Title    string `json:"title"`
			Purpose  string `json:"purpose"`
			Revision int64  `json:"revision"`
		}
		if !decode(&in) {
			return
		}
		out, err = d.Store.IssueSpecification(r.Context(), session.OrgID, project, session.UserID, in.Title, in.Purpose, in.Revision, d.Blobs)
	default:
		http.NotFound(w, r)
		return
	}
	if err != nil {
		specificationError(w, err)
		return
	}
	if d.Broker != nil {
		d.Broker.Wake(session.OrgID)
	}
	writeJSON(w, status, out)
}

func getSpecificationIssues(w http.ResponseWriter, r *http.Request, d Deps) {
	session, ok := memberSession(w, r, d)
	if !ok {
		return
	}
	project := r.PathValue("id")
	if !uuidPattern.MatchString(project) {
		http.NotFound(w, r)
		return
	}
	issues, err := d.Store.ListSpecificationIssues(r.Context(), session.OrgID, project)
	if err != nil {
		specificationError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, issues)
}

func exportSpecification(w http.ResponseWriter, r *http.Request, d Deps) {
	session, ok := memberSession(w, r, d)
	if !ok {
		return
	}
	project := r.PathValue("id")
	if !uuidPattern.MatchString(project) {
		http.NotFound(w, r)
		return
	}
	var snap specifications.Snapshot
	var err error
	name := "specification-draft.csv"
	if id := r.PathValue("issue"); id != "" {
		var issue specifications.Issue
		issue, err = d.Store.GetSpecificationIssue(r.Context(), session.OrgID, project, id)
		if err == nil && r.URL.Query().Get("format") == "pdf" {
			if d.Blobs == nil || len(issue.ExportSHA256) == 0 {
				http.Error(w, "PDF export unavailable", http.StatusNotFound)
				return
			}
			file, openErr := d.Blobs.Open(issue.ExportSHA256)
			if openErr != nil {
				status := http.StatusInternalServerError
				if errors.Is(openErr, files.ErrNotFound) {
					status = http.StatusNotFound
				}
				http.Error(w, "PDF export unavailable", status)
				return
			}
			defer file.Close()
			w.Header().Set("Content-Type", "application/pdf")
			w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="specification-issue-%d.pdf"`, issue.Number))
			_, _ = io.Copy(w, file)
			return
		}
		snap = issue.Snapshot
		name = fmt.Sprintf("specification-issue-%d.csv", issue.Number)
	} else {
		snap, err = d.Store.ReadSpecifications(r.Context(), session.OrgID, project)
	}
	if err != nil {
		specificationError(w, err)
		return
	}
	body, err := specifications.CSV(snap)
	if err != nil {
		specificationError(w, err)
		return
	}
	w.Header().Set("Content-Type", "text/csv; charset=utf-8")
	w.Header().Set("Content-Disposition", `attachment; filename="`+name+`"`)
	w.Write(body)
}

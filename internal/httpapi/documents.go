package httpapi

import (
	"encoding/json"
	"errors"
	"io"
	"mime"
	"net/http"
	"regexp"
	"strings"
	"time"

	"sitewise/internal/files"
	"sitewise/internal/intake"
	"sitewise/internal/store"
)

func downloadDocument(w http.ResponseWriter, r *http.Request, deps Deps) {
	session, ok := memberSession(w, r, deps)
	if !ok {
		return
	}
	id := r.PathValue("id")
	if !uuidPattern.MatchString(id) {
		http.NotFound(w, r)
		return
	}
	doc, err := deps.Store.GetDocument(r.Context(), session.OrgID, id)
	if errors.Is(err, store.ErrNotFound) {
		http.NotFound(w, r)
		return
	}
	if err != nil || deps.Blobs == nil {
		http.Error(w, "read failed", 500)
		return
	}
	file, err := deps.Store.GetFile(r.Context(), session.OrgID, doc.FileID)
	if err != nil {
		http.Error(w, "read failed", 500)
		return
	}
	f, err := deps.Blobs.Open(file.SHA256)
	if err != nil {
		http.Error(w, "file unavailable", 500)
		return
	}
	defer f.Close()
	w.Header().Set("Content-Type", file.MediaType)
	disposition := "attachment"
	if r.URL.Query().Get("view") == "1" && file.MediaType == "application/pdf" {
		disposition = "inline"
	}
	w.Header().Set("Content-Disposition", mime.FormatMediaType(disposition, map[string]string{"filename": doc.Filename}))
	_, _ = io.Copy(w, f)
}

// uuidPattern rejects malformed ids before they reach a uuid cast, so a bad
// path is a 404 rather than a database error.
var uuidPattern = regexp.MustCompile(`^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$`)

// editable are the fields a person may correct here. Supersession is a link
// between filings and is not typed in as a value.
var editable = map[string]bool{
	intake.FieldKind:       true,
	intake.FieldDiscipline: true,
	intake.FieldLifecycle:  true,
	intake.FieldNumber:     true,
	intake.FieldRevision:   true,
	intake.FieldTitle:      true,
	intake.FieldDate:       true,
}

type documentList struct {
	// Cursor is read before the list. Resuming the event stream from it
	// replays anything committed while the list was read.
	Cursor    int64                `json:"cursor"`
	Project   store.Project        `json:"project"`
	Documents []store.DocumentView `json:"documents"`
}

type option struct {
	ID    string `json:"id"`
	Label string `json:"label"`
}

type catalogBody struct {
	Kinds       []option `json:"kinds"`
	Disciplines []option `json:"disciplines"`
	Lifecycle   []option `json:"lifecycle"`
	// ProfileReadKinds are the kinds the profile reads when a document's
	// setting is automatic, so the register can show the resolved state.
	ProfileReadKinds []string `json:"profile_read_kinds"`
}

func checkSession(w http.ResponseWriter, r *http.Request, deps Deps) {
	if _, ok := memberSession(w, r, deps); ok {
		w.WriteHeader(http.StatusNoContent)
	}
}

func listProjects(w http.ResponseWriter, r *http.Request, deps Deps) {
	session, ok := memberSession(w, r, deps)
	if !ok {
		return
	}
	projects, err := deps.Store.ListProjects(r.Context(), session.OrgID)
	if err != nil {
		http.Error(w, "read failed", http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusOK, projects)
}

func listDocuments(w http.ResponseWriter, r *http.Request, deps Deps) {
	session, ok := memberSession(w, r, deps)
	if !ok {
		return
	}
	project, ok := visibleProject(w, r, deps, session.OrgID)
	if !ok {
		return
	}
	cursor, err := deps.Store.LatestEventID(r.Context(), session.OrgID)
	if err != nil {
		http.Error(w, "read failed", http.StatusInternalServerError)
		return
	}
	docs, err := deps.Store.ProjectDocumentViews(r.Context(), session.OrgID, project.ID)
	if err != nil {
		http.Error(w, "read failed", http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusOK, documentList{Cursor: cursor, Project: project, Documents: docs})
}

// uploadFile streams the body to content-addressed storage, commits the
// filing, and starts intake without holding the request open for Jev. The
// result arrives as an event. A re-drop of stored bytes returns the existing
// filing and restarts intake if it had stopped.
func uploadFile(w http.ResponseWriter, r *http.Request, deps Deps) {
	if !originOK(r, deps.PublicOrigin) {
		http.Error(w, "origin rejected", http.StatusForbidden)
		return
	}
	session, ok := memberSession(w, r, deps)
	if !ok {
		return
	}
	project, ok := visibleProject(w, r, deps, session.OrgID)
	if !ok {
		return
	}
	name := strings.TrimSpace(r.URL.Query().Get("name"))
	body := http.MaxBytesReader(w, r.Body, deps.MaxUploadBytes)
	defer body.Close()
	filing, err := deps.Uploader.Upload(r.Context(), session.OrgID, project.ID, intake.Upload{
		Filename: name,
		Body:     body,
		Reason:   intake.ReasonFor(name),
	})
	var tooBig *http.MaxBytesError
	switch {
	case errors.As(err, &tooBig), errors.Is(err, files.ErrTooLarge):
		http.Error(w, "file is larger than the upload limit", http.StatusRequestEntityTooLarge)
		return
	case err != nil && r.Context().Err() != nil:
		return
	case errors.Is(err, intake.ErrFilename):
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	case err != nil:
		deps.Log.Printf("upload failed org=%s: %v", session.OrgID, err)
		http.Error(w, "upload failed", http.StatusInternalServerError)
		return
	}
	if filing.Status == store.StatusPending {
		deps.Filer.Start(session.OrgID, filing.DocumentID)
	}
	view, err := deps.Store.DocumentView(r.Context(), session.OrgID, filing.DocumentID)
	if err != nil {
		http.Error(w, "read failed", http.StatusInternalServerError)
		return
	}
	status := http.StatusOK
	if filing.Created {
		status = http.StatusCreated
	}
	writeJSON(w, status, view)
}

func getDocument(w http.ResponseWriter, r *http.Request, deps Deps) {
	session, ok := memberSession(w, r, deps)
	if !ok {
		return
	}
	view, ok := visibleDocument(w, r, deps, session.OrgID)
	if !ok {
		return
	}
	writeJSON(w, http.StatusOK, view)
}

// retryFiling restarts intake for stored bytes that are still pending, so an
// interrupted filing never needs the file uploaded again.
func retryFiling(w http.ResponseWriter, r *http.Request, deps Deps) {
	if !originOK(r, deps.PublicOrigin) {
		http.Error(w, "origin rejected", http.StatusForbidden)
		return
	}
	session, ok := memberSession(w, r, deps)
	if !ok {
		return
	}
	view, ok := visibleDocument(w, r, deps, session.OrgID)
	if !ok {
		return
	}
	if view.Status == store.StatusNotFiled && (view.Reason == "no_text_layer" || strings.HasPrefix(view.Reason, "ocr_")) {
		if err := deps.Store.RetryOCR(r.Context(), session.OrgID, view.ID); err != nil {
			http.Error(w, "OCR retry failed", http.StatusConflict)
			return
		}
		deps.Broker.Wake(session.OrgID)
		view, err := deps.Store.DocumentView(r.Context(), session.OrgID, view.ID)
		if err != nil {
			http.Error(w, "read failed", 500)
			return
		}
		writeJSON(w, http.StatusAccepted, view)
		return
	}
	if view.Expansion != nil && view.Expansion.Status == "review" {
		if err := deps.Store.RetryDrawingExpansion(r.Context(), session.OrgID, view.ID); err != nil {
			http.Error(w, "retry failed", 500)
			return
		}
	}
	if view.Status == store.StatusPending || (view.Expansion != nil && view.Expansion.Status != "complete") {
		deps.Filer.Start(session.OrgID, view.ID)
	}
	writeJSON(w, http.StatusAccepted, view)
}

func reprocessDetails(w http.ResponseWriter, r *http.Request, deps Deps) {
	if !originOK(r, deps.PublicOrigin) {
		http.Error(w, "origin rejected", http.StatusForbidden)
		return
	}
	session, ok := memberSession(w, r, deps)
	if !ok {
		return
	}
	view, ok := visibleDocument(w, r, deps, session.OrgID)
	if !ok {
		return
	}
	if err := deps.Store.ReprocessOCRDetails(r.Context(), session.OrgID, view.ID); err != nil {
		http.Error(w, "Missing-detail recovery is not available for this document. Refresh and check its current details.", http.StatusConflict)
		return
	}
	deps.Broker.Wake(session.OrgID)
	view, err := deps.Store.DocumentView(r.Context(), session.OrgID, view.ID)
	if err != nil {
		http.Error(w, "read failed", 500)
		return
	}
	writeJSON(w, http.StatusAccepted, view)
}

func correctField(w http.ResponseWriter, r *http.Request, deps Deps) {
	if !originOK(r, deps.PublicOrigin) {
		http.Error(w, "origin rejected", http.StatusForbidden)
		return
	}
	session, ok := memberSession(w, r, deps)
	if !ok {
		return
	}
	id := r.PathValue("id")
	field := r.PathValue("field")
	if !uuidPattern.MatchString(id) {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}
	if !editable[field] {
		http.Error(w, "field cannot be corrected", http.StatusBadRequest)
		return
	}
	var body struct {
		Value *string `json:"value"`
	}
	if err := readJSON(w, r, deps.MaxBodyBytes, &body); err != nil || body.Value == nil {
		http.Error(w, "value required", http.StatusBadRequest)
		return
	}
	value := strings.TrimSpace(*body.Value)
	if !inCatalog(deps.Catalog, field, value) {
		http.Error(w, "value is not in the vocabulary", http.StatusBadRequest)
		return
	}
	err := deps.Service.Correct(r.Context(), session.OrgID, id, field, value)
	if errors.Is(err, store.ErrNotFound) {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}
	if err != nil {
		http.Error(w, "correction failed", http.StatusBadRequest)
		return
	}
	deps.Broker.Wake(session.OrgID)
	view, err := deps.Store.DocumentView(r.Context(), session.OrgID, id)
	if err != nil {
		http.Error(w, "read failed", http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusOK, view)
}

func getCatalog(w http.ResponseWriter, r *http.Request, deps Deps) {
	if _, ok := memberSession(w, r, deps); !ok {
		return
	}
	out := catalogBody{}
	for _, k := range deps.Catalog.Kinds {
		out.Kinds = append(out.Kinds, option{ID: k.ID, Label: k.Label})
	}
	for _, d := range deps.Catalog.Disciplines {
		out.Disciplines = append(out.Disciplines, option{ID: d.ID, Label: d.Label})
	}
	for _, a := range deps.Catalog.Lifecycle {
		out.Lifecycle = append(out.Lifecycle, option{ID: a.ID, Label: a.Label})
	}
	out.ProfileReadKinds = deps.ProfileReading.Kinds()
	w.Header().Set("Cache-Control", "private, max-age=300")
	writeJSON(w, http.StatusOK, out)
}

// inCatalog accepts an empty value (the person says there is none) or an id
// from the closed vocabulary. Free-text fields take any value.
func inCatalog(cat intake.Catalog, field, value string) bool {
	if value == "" {
		return true
	}
	switch field {
	case intake.FieldKind:
		for _, k := range cat.Kinds {
			if k.ID == value {
				return true
			}
		}
		return false
	case intake.FieldDiscipline:
		for _, d := range cat.Disciplines {
			if d.ID == value {
				return true
			}
		}
		return false
	case intake.FieldLifecycle:
		for _, a := range cat.Lifecycle {
			if a.ID == value {
				return true
			}
		}
		return false
	default:
		return len(value) <= 500
	}
}

func visibleProject(w http.ResponseWriter, r *http.Request, deps Deps, orgID string) (store.Project, bool) {
	id := r.PathValue("id")
	if !uuidPattern.MatchString(id) {
		http.Error(w, "not found", http.StatusNotFound)
		return store.Project{}, false
	}
	project, err := deps.Store.GetProject(r.Context(), orgID, id)
	if errors.Is(err, store.ErrNotFound) {
		http.Error(w, "not found", http.StatusNotFound)
		return store.Project{}, false
	}
	if err != nil {
		http.Error(w, "read failed", http.StatusInternalServerError)
		return store.Project{}, false
	}
	return project, true
}

func visibleDocument(w http.ResponseWriter, r *http.Request, deps Deps, orgID string) (store.DocumentView, bool) {
	id := r.PathValue("id")
	if !uuidPattern.MatchString(id) {
		http.Error(w, "not found", http.StatusNotFound)
		return store.DocumentView{}, false
	}
	view, err := deps.Store.DocumentView(r.Context(), orgID, id)
	if errors.Is(err, store.ErrNotFound) {
		http.Error(w, "not found", http.StatusNotFound)
		return store.DocumentView{}, false
	}
	if err != nil {
		http.Error(w, "read failed", http.StatusInternalServerError)
		return store.DocumentView{}, false
	}
	return view, true
}

func writeJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}

const (
	maxDeleteIDs = 200
	// blobMinAge keeps bytes an upload may be about to reference; a younger
	// unreferenced blob stays as an orphan until a later sweep.
	blobMinAge = 2 * time.Minute
)

// deleteDocuments permanently deletes a selection from the register, after
// the user confirmed. The database deletion is one transaction; stored bytes
// go afterwards, only when no org references them. A blob that cannot be
// removed stays on disk unreferenced and the deletion still stands.
func deleteDocuments(w http.ResponseWriter, r *http.Request, deps Deps) {
	if !originOK(r, deps.PublicOrigin) {
		http.Error(w, "origin rejected", http.StatusForbidden)
		return
	}
	session, ok := memberSession(w, r, deps)
	if !ok {
		return
	}
	projectID := r.PathValue("id")
	if !uuidPattern.MatchString(projectID) {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}
	var body struct {
		DocumentIDs []string `json:"document_ids"`
	}
	if err := readJSON(w, r, deps.MaxBodyBytes, &body); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	if len(body.DocumentIDs) == 0 || len(body.DocumentIDs) > maxDeleteIDs {
		http.Error(w, "delete 1 to 200 documents at a time", http.StatusBadRequest)
		return
	}
	for _, id := range body.DocumentIDs {
		if !uuidPattern.MatchString(id) {
			http.Error(w, "not found", http.StatusNotFound)
			return
		}
	}
	res, err := deps.Store.DeleteDocuments(r.Context(), session.OrgID, projectID, body.DocumentIDs, session.UserID)
	if errors.Is(err, store.ErrNotFound) {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}
	if err != nil {
		http.Error(w, "delete failed", http.StatusInternalServerError)
		return
	}
	for _, sum := range res.Unreferenced {
		if _, err := deps.Blobs.Remove(sum, blobMinAge, func() (bool, error) {
			return deps.Store.BlobReferenced(r.Context(), sum)
		}); err != nil {
			deps.Log.Printf("delete: stored file kept: %v", err)
		}
	}
	if deps.Knowledge != nil {
		if err := rebuildProfile(r, deps, session.OrgID, projectID); err != nil {
			deps.Log.Printf("delete: profile rebuild failed: %v", err)
		}
	}
	writeJSON(w, http.StatusOK, map[string][]string{"deleted": res.Deleted})
}

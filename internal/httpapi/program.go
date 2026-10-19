package httpapi

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"

	engine "sitewise/internal/program"
	"sitewise/internal/store"
)

// Program routes (plan docs/plans/2026-10-18-program.md section 8), all
// under /projects/{id}/program and scoped to the session's org like the
// cost plan. None calls Jev: generation, the question and every edit are
// code over stored values. Every response is the full view (8.2), so the
// client never merges.

const (
	pathProgramRead     = "program_read"
	pathProgramWrite    = "program_write"
	pathProgramGenerate = "program_generate"

	// eventProgramChanged tells open views to refetch when the version is
	// newer than theirs (8.3).
	eventProgramChanged = "program.changed"

	msgProgramConflict = "Someone else changed the program; showing their version."
	msgQuestionGone    = "That question is no longer asked."
	maxQuestionValue   = 200
)

// programError maps store errors. A missing program, project or another
// org's ids are all a bare 404: nothing says which. A stale version is 409
// with the current view so the client can replace its state.
func programError(w http.ResponseWriter, r *http.Request, d Deps, org string, res store.ProgramResult, err error) {
	var opErr *store.ProgramOpError
	switch {
	case errors.As(err, &opErr):
		writeProgramOpError(w, opErr)
	case errors.Is(err, store.ErrVersionConflict):
		current, verr := d.Store.ProgramView(r.Context(), org, r.PathValue("id"), res.ProgramState, d.Knowledge)
		if verr != nil {
			http.Error(w, "program operation failed", http.StatusInternalServerError)
			return
		}
		writeJSON(w, http.StatusConflict, map[string]any{"error": msgProgramConflict, "current": current})
	case errors.Is(err, store.ErrNotFound):
		http.Error(w, "not found", http.StatusNotFound)
	default:
		d.Log.Printf("program: %v", err)
		http.Error(w, "program operation failed", http.StatusInternalServerError)
	}
}

func writeProgramOpError(w http.ResponseWriter, e *store.ProgramOpError) {
	body := map[string]any{"error": e.Message, "field": e.Field, "op_index": e.Index, "cycle": e.Cycle, "held": e.Held}
	if e.Cycle == nil {
		body["cycle"] = []string{}
	}
	if e.Held == nil {
		body["held"] = []string{}
	}
	writeJSON(w, http.StatusUnprocessableEntity, body)
}

// programInvalid is a 422 for a body field the store never sees.
func programInvalid(w http.ResponseWriter, field, message string) {
	writeProgramOpError(w, &store.ProgramOpError{Index: -1, Field: field, Message: message})
}

// programChanged publishes the committed version. The event is durable
// before subscribers wake; a failure to publish never fails the write,
// which has committed: a reconnecting client reads the state anyway.
func programChanged(r *http.Request, d Deps, org string, p store.Program) {
	if d.Broker == nil {
		return
	}
	payload, _ := json.Marshal(map[string]any{"project_id": p.ProjectID, "program_id": p.ID, "version": p.Version})
	if _, err := d.Broker.Commit(r.Context(), org, eventProgramChanged, "", string(payload)); err != nil {
		d.Log.Printf("program event: %v", err)
	}
}

// writeProgramResult renders a committed write and announces it.
func writeProgramResult(w http.ResponseWriter, r *http.Request, d Deps, org string, res store.ProgramResult) {
	view, err := d.Store.ResultView(r.Context(), org, r.PathValue("id"), res, d.Knowledge)
	if err != nil {
		programError(w, r, d, org, res, err)
		return
	}
	programChanged(r, d, org, res.Program)
	writeJSON(w, http.StatusOK, view)
}

// programSession is the cost plan's session check: member, origin on
// writes, a well-formed project id in the caller's org.
func programSession(w http.ResponseWriter, r *http.Request, d Deps, write bool) (store.Session, bool) {
	s, ok := deliverySession(w, r, d, write)
	if !ok {
		return s, false
	}
	if d.Knowledge == nil {
		http.Error(w, "not found", http.StatusNotFound)
		return s, false
	}
	return s, true
}

// getProgram is GET /program: 404 {"empty": true} for a project of the
// caller's org with no program yet.
func getProgram(w http.ResponseWriter, r *http.Request, d Deps) {
	s, ok := programSession(w, r, d, false)
	if !ok {
		return
	}
	st, err := d.Store.LoadProgram(r.Context(), s.OrgID, r.PathValue("id"))
	if errors.Is(err, store.ErrNotFound) {
		writeJSON(w, http.StatusNotFound, map[string]bool{"empty": true})
		return
	}
	if err != nil {
		programError(w, r, d, s.OrgID, store.ProgramResult{}, err)
		return
	}
	view, err := d.Store.ProgramView(r.Context(), s.OrgID, r.PathValue("id"), st, d.Knowledge)
	if err != nil {
		programError(w, r, d, s.OrgID, store.ProgramResult{}, err)
		return
	}
	writeJSON(w, http.StatusOK, view)
}

// postProgramPrepare is POST /program/prepare {"surface"}: creates the
// program when none exists, then answers as GET with "created".
func postProgramPrepare(w http.ResponseWriter, r *http.Request, d Deps) {
	s, ok := programSession(w, r, d, true)
	if !ok {
		return
	}
	var body struct {
		Surface string `json:"surface"`
	}
	if readProposalJSON(w, r, d.MaxBodyBytes, &body) != nil {
		http.Error(w, "invalid body", http.StatusBadRequest)
		return
	}
	if !store.ValidProgramSurface(body.Surface) {
		programInvalid(w, "surface", "Surface is program, procurement or cost_plan.")
		return
	}
	st, created, err := d.Store.PrepareProgram(r.Context(), s.OrgID, r.PathValue("id"), body.Surface, d.Knowledge)
	if err != nil {
		programError(w, r, d, s.OrgID, store.ProgramResult{}, err)
		return
	}
	view, err := d.Store.ProgramView(r.Context(), s.OrgID, r.PathValue("id"), st, d.Knowledge)
	if err != nil {
		programError(w, r, d, s.OrgID, store.ProgramResult{}, err)
		return
	}
	if created {
		programChanged(r, d, s.OrgID, st.Program)
	}
	writeJSON(w, http.StatusOK, struct {
		store.ProgramView
		Created bool `json:"created"`
	}{view, created})
}

// postProgramOps is POST /program/ops {"program_id","version","ops"}.
func postProgramOps(w http.ResponseWriter, r *http.Request, d Deps) {
	s, ok := programSession(w, r, d, true)
	if !ok {
		return
	}
	var body struct {
		ProgramID string            `json:"program_id"`
		Version   int64             `json:"version"`
		Ops       []store.ProgramOp `json:"ops"`
	}
	if readProposalJSON(w, r, d.MaxBodyBytes, &body) != nil {
		http.Error(w, "invalid body", http.StatusBadRequest)
		return
	}
	if !uuidPattern.MatchString(body.ProgramID) {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}
	if body.Version < 1 {
		programInvalid(w, "version", "The program version is required.")
		return
	}
	res, err := d.Store.ApplyProgramOps(r.Context(), s.OrgID, r.PathValue("id"), s.UserID, body.ProgramID, body.Version, body.Ops)
	if err != nil {
		programError(w, r, d, s.OrgID, res, err)
		return
	}
	writeProgramResult(w, r, d, s.OrgID, res)
}

// postProgramUndo is POST /program/undo and /redo {"version"}.
func postProgramUndo(w http.ResponseWriter, r *http.Request, d Deps) {
	s, ok := programSession(w, r, d, true)
	if !ok {
		return
	}
	var body struct {
		Version int64 `json:"version"`
	}
	if readProposalJSON(w, r, d.MaxBodyBytes, &body) != nil {
		http.Error(w, "invalid body", http.StatusBadRequest)
		return
	}
	if body.Version < 1 {
		programInvalid(w, "version", "The program version is required.")
		return
	}
	flip := d.Store.UndoProgram
	if strings.HasSuffix(r.URL.Path, "/redo") {
		flip = d.Store.RedoProgram
	}
	res, err := flip(r.Context(), s.OrgID, r.PathValue("id"), s.UserID, body.Version)
	if err != nil {
		programError(w, r, d, s.OrgID, res, err)
		return
	}
	writeProgramResult(w, r, d, s.OrgID, res)
}

// postProgramAccept is POST /program/accept {"version", "ids"} or
// {"version", "all": true}.
func postProgramAccept(w http.ResponseWriter, r *http.Request, d Deps) {
	s, ok := programSession(w, r, d, true)
	if !ok {
		return
	}
	var body struct {
		Version int64    `json:"version"`
		IDs     []string `json:"ids"`
		All     bool     `json:"all"`
	}
	if readProposalJSON(w, r, d.MaxBodyBytes, &body) != nil {
		http.Error(w, "invalid body", http.StatusBadRequest)
		return
	}
	if body.Version < 1 {
		programInvalid(w, "version", "The program version is required.")
		return
	}
	for _, id := range body.IDs {
		if !uuidPattern.MatchString(id) {
			programInvalid(w, "ids", store.MsgNotInProgram)
			return
		}
	}
	res, err := d.Store.AcceptProgram(r.Context(), s.OrgID, r.PathValue("id"), s.UserID, body.Version, body.IDs, body.All)
	if err != nil {
		programError(w, r, d, s.OrgID, res, err)
		return
	}
	writeProgramResult(w, r, d, s.OrgID, res)
}

// postProgramQuestion is POST /program/question {"version","key","value"}
// (plan 7.3). "not_sure" dismisses the question until its Profile value
// changes. Any other value is written to the Profile through the Profile
// edit path (the same check and SetUserValue as PUT /profile/{key}, with
// origin user), then the program changes it causes apply at once as one
// undoable batch with a toast.
func postProgramQuestion(w http.ResponseWriter, r *http.Request, d Deps) {
	s, ok := programSession(w, r, d, true)
	if !ok {
		return
	}
	var body struct {
		Version int64  `json:"version"`
		Key     string `json:"key"`
		Value   string `json:"value"`
	}
	if readProposalJSON(w, r, d.MaxBodyBytes, &body) != nil {
		http.Error(w, "invalid body", http.StatusBadRequest)
		return
	}
	body.Value = strings.TrimSpace(body.Value)
	if body.Version < 1 {
		programInvalid(w, "version", "The program version is required.")
		return
	}
	if len([]rune(body.Value)) > maxQuestionValue {
		programInvalid(w, "value", "value is longer than 200 characters")
		return
	}
	ctx, project := r.Context(), r.PathValue("id")
	st, err := d.Store.LoadProgram(ctx, s.OrgID, project)
	if err != nil {
		programError(w, r, d, s.OrgID, store.ProgramResult{}, err)
		return
	}
	// The question is recomputed, never trusted from the client: the key
	// must be the one asked now, and the value one of its options.
	view, err := d.Store.ProgramView(ctx, s.OrgID, project, st, d.Knowledge)
	if err != nil {
		programError(w, r, d, s.OrgID, store.ProgramResult{}, err)
		return
	}
	if st.Program.Version != body.Version {
		writeJSON(w, http.StatusConflict, map[string]any{"error": msgProgramConflict, "current": view})
		return
	}
	q := view.Question
	if q == nil || q.Key != body.Key {
		programInvalid(w, "key", msgQuestionGone)
		return
	}
	profileKey := store.ProgramProfileKey(*q)
	if body.Value == "not_sure" {
		st, err := d.Store.DismissProgramQuestion(ctx, s.OrgID, project, body.Version, profileKey)
		if err != nil {
			programError(w, r, d, s.OrgID, store.ProgramResult{ProgramState: st}, err)
			return
		}
		fb := store.ProgramResult{ProgramState: st}
		fb.Toasts = []string{"We won't ask about " + strings.ToLower(q.Label) + " again until it changes in the Profile."}
		writeProgramResult(w, r, d, s.OrgID, fb)
		return
	}
	if !questionValueOK(*q, body.Value) {
		programInvalid(w, "value", "That is not an answer to this question.")
		return
	}
	// The Profile stores a yes/no determinant as stated_true or
	// stated_false; the question's values are the engine's true and false.
	value := body.Value
	if det, ok := d.Knowledge.Determinant(q.Key); ok && q.Kind == engine.QuestionDeterminant && det.Value == "boolean" {
		value = map[string]string{"true": "stated_true", "false": "stated_false"}[value]
	}
	if msg := validProfileValue(d.Knowledge, profileKey, &value, ""); msg != "" {
		programInvalid(w, "value", msg)
		return
	}
	scope := d.Knowledge.KeyScope(profileKey)
	if scope == "" {
		scope = "project"
	}
	whole, err := d.Store.EnsureWholePart(ctx, s.OrgID, project)
	if err != nil {
		programError(w, r, d, s.OrgID, store.ProgramResult{}, err)
		return
	}
	if _, err := d.Store.SetUserValue(ctx, s.OrgID, project, whole.ID, s.UserID, profileKey, store.UserWrite{
		Value: &value, State: "set", Origin: "user", Meaning: "stated", Scope: scope}); err != nil {
		programError(w, r, d, s.OrgID, store.ProgramResult{}, err)
		return
	}
	res, _, err := d.Store.ApplyProgramAnswer(ctx, s.OrgID, project, s.UserID, body.Version, d.Knowledge, *q, body.Value)
	if err != nil {
		programError(w, r, d, s.OrgID, res, err)
		return
	}
	writeProgramResult(w, r, d, s.OrgID, res)
}

// questionValueOK accepts one of the question's values, or a number for a
// number question.
func questionValueOK(q engine.Question, v string) bool {
	if q.Number {
		_, err := strconv.ParseFloat(v, 64)
		return err == nil
	}
	return contains(q.Values, v)
}

package httpapi

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"sitewise/internal/costs"
	"sitewise/internal/store"
)

// operation is one named edit. Run does what the screen's handler does for
// the same edit and calls the same store write, so each record still has
// one writer (docs/plans/2026-11-14-change-log-undo-operations.md, task 6).
type operation struct {
	Name  string // "cost_plan.add_line": the area, a dot, the edit
	About string // one sentence, read by the chat agent
	// Fields lists the fields in plain words for the agent's prompt.
	Fields string
	// Figures names the fields that hold an amount ("money"), a duration
	// ("duration"), a date ("date") or a typed value that may hold a number
	// ("value"). Chat refuses a figure the PM did not give (review change 2).
	Figures map[string]string
	Routes  []string // the write routes this stands for
	// Click: the PM vouches for something by doing this, so chat prepares
	// it and the PM clicks (review change 4).
	Click bool
	// NoUndo: the change log cannot put this back (its tables carry no
	// capture trigger), so it is never recorded as an undoable change.
	NoUndo bool
	// Hold says why this call must wait for a click even though the
	// operation usually applies at once ("" means apply). It reads; it
	// never writes.
	Hold func(ctx context.Context, d Deps, who opActor, project string, fields json.RawMessage) (string, error)
	Run  func(ctx context.Context, d Deps, who opActor, project string, fields json.RawMessage) (summary string, err error)
	// Batch takes every call of a batch in one store write that has its own
	// undo (Program, tender tables). An operation has Run or Batch.
	Batch func(ctx context.Context, d Deps, who opActor, project string, in opBatch) (opApplied, error)
}

type opActor struct{ OrgID, UserID string }

type opCall struct {
	Op     string          `json:"op"`
	Fields json.RawMessage `json:"fields"`
}

// opBatch is one apply: operations of one area, all or nothing.
type opBatch struct {
	Area   string   `json:"area"`
	TurnID string   `json:"turn_id"`
	Calls  []opCall `json:"ops"`
	// Via is "chat" or "screen". Clicked is true when the PM's own click
	// sent the batch, which is what a Click operation waits for.
	Via     string `json:"-"`
	Clicked bool   `json:"-"`
}

type opResult struct {
	Op      string `json:"op"`
	Summary string `json:"summary"`
}

// opUndo says which undo reverses the batch: the change log's, or Program's own.
type opUndo struct {
	Kind string `json:"kind"` // "change", "program" or "none"
}

type opApplied struct {
	ChangeID string     `json:"change_id,omitempty"`
	Undo     opUndo     `json:"undo"`
	Results  []opResult `json:"results"`
}

// opError is a refused batch, in Program's shape. NeedsClick is not a
// fault: the batch is right but waits for the PM's click, and Message says
// why.
type opError struct {
	Index      int    `json:"op_index"`
	Field      string `json:"field,omitempty"`
	Message    string `json:"message"`
	NeedsClick bool   `json:"needs_click,omitempty"`
	status     int
}

func (e *opError) Error() string { return e.Message }

// refuse is an operation's plain refusal of its own fields.
func refuse(field, format string, args ...any) error {
	return &opError{Field: field, Message: fmt.Sprintf(format, args...)}
}

const maxOpsPerBatch = 50

var opAreas = []string{"cost_plan", "profile", "program", "procurement", "invoices", "delivery", "specifications", "reports", "documents"}

// operations is the registry. The order is the order the agent reads.
func operations() []operation {
	var all []operation
	all = append(all, costOperations()...)
	all = append(all, forwardedOperations("cost_plan")...)
	all = append(all, profileOperations()...)
	all = append(all, roomOperations()...)
	all = append(all, forwardedOperations("profile")...)
	all = append(all, programOperations()...)
	all = append(all, forwardedOperations("program")...)
	all = append(all, procurementOperations()...)
	for _, area := range opAreas[3:] {
		all = append(all, forwardedOperations(area)...)
	}
	return all
}

func operationNamed(name string) (operation, bool) {
	for _, op := range operations() {
		if op.Name == name {
			return op, true
		}
	}
	return operation{}, false
}

func opArea(name string) string {
	area, _, _ := strings.Cut(name, ".")
	return area
}

// checkBatch names the first thing wrong with a batch before anything is
// written.
func checkBatch(in opBatch) ([]operation, *opError) {
	known := false
	for _, a := range opAreas {
		known = known || a == in.Area
	}
	if !known {
		return nil, &opError{Index: -1, Field: "area", Message: "Areas are " + strings.Join(opAreas, ", ") + "."}
	}
	if len(in.Calls) == 0 || len(in.Calls) > maxOpsPerBatch {
		return nil, &opError{Index: -1, Field: "ops", Message: fmt.Sprintf("Send between 1 and %d operations.", maxOpsPerBatch)}
	}
	ops := make([]operation, len(in.Calls))
	for i, call := range in.Calls {
		op, ok := operationNamed(call.Op)
		switch {
		case !ok:
			return nil, &opError{Index: i, Field: "op", Message: "There is no operation named " + call.Op + "."}
		case opArea(op.Name) != in.Area:
			return nil, &opError{Index: i, Field: "op", Message: op.Name + " belongs to area " + opArea(op.Name) + "; send it in its own apply."}
		case op.Click && !in.Clicked:
			return nil, &opError{Index: i, NeedsClick: true, Message: "This is yours to confirm: " + op.About}
		case i > 0 && (op.Batch != nil || ops[0].Batch != nil) && op.Name != in.Calls[0].Op:
			return nil, &opError{Index: i, Field: "op", Message: op.Name + " cannot share a batch with " + in.Calls[0].Op + "; send it in its own apply."}
		}
		ops[i] = op
	}
	return ops, nil
}

// applyOps applies a batch or none of it. A batch of plain operations is
// several store transactions, so "nothing stays applied" is done by undoing
// what was captured, not by one transaction: each operation runs under the
// same marked change, and the first refusal undoes that change.
func applyOps(ctx context.Context, d Deps, who opActor, project string, in opBatch) (opApplied, *opError) {
	ops, bad := checkBatch(in)
	if bad != nil {
		return opApplied{}, bad
	}
	if !in.Clicked {
		for i, op := range ops {
			if op.Hold == nil {
				continue
			}
			why, err := op.Hold(ctx, d, who, project, in.Calls[i].Fields)
			if err != nil {
				return opApplied{}, asOpError(i, err)
			}
			if why != "" {
				return opApplied{}, &opError{Index: i, NeedsClick: true, Message: why}
			}
		}
	}
	if ops[0].Batch != nil {
		out, err := ops[0].Batch(ctx, d, who, project, in)
		if err != nil {
			return opApplied{}, asOpError(-1, err)
		}
		return out, nil
	}
	via := in.Via
	if via == "" {
		via = "screen"
	}
	change := store.Change{ID: store.NewChangeID(), OrgID: who.OrgID, ProjectID: project, Actor: who.UserID, Via: via, TurnID: in.TurnID, Area: in.Area}
	marked := store.WithChange(ctx, change)
	out := opApplied{ChangeID: change.ID, Undo: opUndo{Kind: "change"}}
	for i, op := range ops {
		run := marked
		if op.Click || op.NoUndo {
			// What the PM vouches for by a click (a freeze, a release) is
			// not put back by Undo, so it is not captured as a change.
			run = ctx
		}
		summary, err := op.Run(run, d, who, project, in.Calls[i].Fields)
		if err != nil {
			// Put back what the earlier operations wrote. A change that
			// captured nothing does not exist, which is fine.
			if _, undoErr := d.Store.UndoChange(ctx, who.OrgID, project, who.UserID, change.ID); undoErr != nil && !errors.Is(undoErr, store.ErrNotFound) {
				d.Log.Printf("ops: undo after a refused batch failed: %v", undoErr)
				return opApplied{}, &opError{Index: i, Message: "The change failed part way and could not be put back. Check " + in.Area + ".", status: http.StatusInternalServerError}
			}
			return opApplied{}, asOpError(i, err)
		}
		out.Results = append(out.Results, opResult{Op: op.Name, Summary: summary})
	}
	raw, _ := json.Marshal(in.Calls)
	if err := d.Store.SetChangeOps(ctx, who.OrgID, change.ID, raw); err != nil && !errors.Is(err, store.ErrNotFound) {
		d.Log.Printf("ops: storing the batch failed: %v", err)
	} else if errors.Is(err, store.ErrNotFound) {
		// Nothing was captured (every operation was a no-op), so there is
		// nothing to undo.
		out.ChangeID, out.Undo.Kind = "", "none"
	}
	return out, nil
}

// asOpError turns what an operation returned into the refusal the caller
// sees, with plain words for the store's errors.
func asOpError(index int, err error) *opError {
	var own *opError
	var prog *store.ProgramOpError
	switch {
	case errors.As(err, &own):
		out := *own
		if index >= 0 {
			out.Index = index
		}
		if out.status == 0 {
			out.status = http.StatusUnprocessableEntity
		}
		return &out
	case errors.As(err, &prog):
		return &opError{Index: prog.Index, Field: prog.Field, Message: prog.Message, status: http.StatusUnprocessableEntity}
	case errors.Is(err, store.ErrNotFound):
		return &opError{Index: index, Message: "That is not in this project.", status: http.StatusNotFound}
	case errors.Is(err, store.ErrFrozenCostPlan):
		return &opError{Index: index, Message: "The cost plan is frozen; it cannot be edited.", status: http.StatusConflict}
	case errors.Is(err, store.ErrVersionConflict):
		return &opError{Index: index, Message: "This changed while the edit was being made. Read it again.", status: http.StatusConflict}
	case errors.Is(err, costs.ErrInvalid):
		return &opError{Index: index, Message: "The cost plan does not accept that: " + strings.TrimPrefix(err.Error(), costs.ErrInvalid.Error()+": "), status: http.StatusUnprocessableEntity}
	case errors.Is(err, store.ErrInvalidInput):
		return &opError{Index: index, Message: err.Error(), status: http.StatusUnprocessableEntity}
	}
	return &opError{Index: index, Message: "The change could not be made.", status: http.StatusInternalServerError}
}

// decodeFields reads an operation's fields and refuses any it does not know.
func decodeFields(raw json.RawMessage, into any) error {
	if len(bytes.TrimSpace(raw)) == 0 {
		raw = json.RawMessage(`{}`)
	}
	dec := json.NewDecoder(bytes.NewReader(raw))
	dec.DisallowUnknownFields()
	if err := dec.Decode(into); err != nil {
		msg := err.Error()
		if name, ok := strings.CutPrefix(msg, "json: unknown field "); ok {
			return refuse(strings.Trim(name, `"`), "%s is not a field of this operation.", name)
		}
		return refuse("", "The fields are not in the shape this operation takes.")
	}
	return nil
}

// byRef finds one row by its id or by the short id the chat reads show
// (the first eight characters). Two rows sharing a short id is refused
// rather than guessed.
func byRef[T any](rows []T, id func(T) string, ref, what string) (T, error) {
	var zero, found T
	ref = strings.ToLower(strings.TrimSpace(ref))
	n := 0
	for _, r := range rows {
		full := id(r)
		if full == ref {
			return r, nil
		}
		if len(ref) >= 6 && strings.HasPrefix(full, ref) {
			found = r
			n++
		}
	}
	switch n {
	case 1:
		return found, nil
	case 0:
		return zero, refuse("", "There is no %s with id %q here. Use an id from a read.", what, ref)
	}
	return zero, refuse("", "The id %q matches more than one %s; use the full id.", ref, what)
}

func shortID(id string) string {
	if len(id) > 8 {
		return id[:8]
	}
	return id
}

// postOps is POST /projects/{id}/ops: a screen or a prepared chat change
// applying a batch with the caller's own session.
func postOps(w http.ResponseWriter, r *http.Request, d Deps) {
	s, ok := deliverySession(w, r, d, true)
	if !ok {
		return
	}
	var in opBatch
	if readProposalJSON(w, r, d.MaxBodyBytes, &in) != nil {
		http.Error(w, "invalid body", http.StatusBadRequest)
		return
	}
	if in.TurnID != "" && !uuidPattern.MatchString(in.TurnID) {
		http.Error(w, "invalid turn", http.StatusUnprocessableEntity)
		return
	}
	in.Via = "screen"
	out, bad := applyOps(withOpRequest(r.Context(), r), d, opActor{s.OrgID, s.UserID}, r.PathValue("id"), in)
	if bad != nil {
		writeOpError(w, bad)
		return
	}
	if d.Broker != nil {
		d.Broker.Wake(s.OrgID)
	}
	writeJSON(w, http.StatusOK, out)
}

func writeOpError(w http.ResponseWriter, e *opError) {
	status := e.status
	if status == 0 || e.NeedsClick {
		status = http.StatusUnprocessableEntity
	}
	writeJSON(w, status, e)
}

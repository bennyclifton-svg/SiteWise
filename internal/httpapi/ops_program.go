package httpapi

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math/bits"
	"strings"

	"sitewise/internal/figures"
	"sitewise/internal/store"
)

// Program operations. program.edit is passed whole to ApplyProgramOps: it
// is already one transaction with its own undo, a later edit may name a row
// added earlier in the batch, and every date is computed in Go.

func programOperations() []operation {
	return []operation{
		{Name: "program.edit", About: "One Program edit. Edits of a batch apply together; a later edit may name an earlier add by its temp_id. Go computes every date.",
			Fields: `op (add|update|delete|move|link_add|link_delete|set_start|set_working_week); ` +
				`add: parent_id (the stage), label, or labels (a list, for several rows alike), or label with count (5 makes "label 1" to "label 5"), kind (activity|milestone|stage), duration, temp_id; ` +
				`update: id, label, duration; delete: ids; link_add: pred_id, succ_id, type (FS), lag_days; ` +
				`move: id, days; set_start: start_date (YYYY-MM-DD); set_working_week: working_week (31 is Monday to Friday, 63 Monday to Saturday, 127 every day). ` +
				`duration is written as the PM said it ("2 weeks", "10 days"); Go converts it on the program's working week`,
			Figures: map[string]string{"duration": "duration", "duration_days": "number", "days": "number", "lag_days": "number",
				"start_date": "date", "constraint_date": "date", "day": "date", "working_week": "number", "count": "number"},
			Routes: []string{"POST /projects/{id}/program/ops"}, Batch: programBatch},
		{Name: "program.release_hold_point", About: "release a hold point: you are saying its inspection or approval was given.", Click: true,
			Fields: `id (the hold point)`, Routes: []string{"POST /projects/{id}/program/hold-points/release"}, Run: programRelease},
	}
}

// programEdit is a ProgramOp as chat sends it: ids may be short, and a
// duration may be written in words.
type programEdit struct {
	store.ProgramOp
	Duration *string `json:"duration,omitempty"`
	// Labels adds one row per label, alike in every other field.
	Labels []string `json:"labels,omitempty"`
	// Count adds that many rows named label 1, label 2 and so on: "five
	// activities for the DA drawings" is then a dozen tokens, not fifty.
	Count int `json:"count,omitempty"`
}

func loadProgramForOps(ctx context.Context, d Deps, who opActor, project string) (store.ProgramState, error) {
	if d.Knowledge == nil {
		return store.ProgramState{}, refuse("", "Program is not available.")
	}
	st, err := d.Store.LoadProgram(ctx, who.OrgID, project)
	if errors.Is(err, store.ErrNotFound) {
		// The screen creates the program when it is first opened; chat
		// does the same so "add an activity" works on a new project.
		if err := d.Store.EnsureProgram(ctx, who.OrgID, project, "program", d.Knowledge); err != nil {
			return st, err
		}
		st, err = d.Store.LoadProgram(ctx, who.OrgID, project)
	}
	return st, err
}

func programBatch(ctx context.Context, d Deps, who opActor, project string, in opBatch) (opApplied, error) {
	st, err := loadProgramForOps(ctx, d, who, project)
	if err != nil {
		return opApplied{}, err
	}
	// The working week is stored as a set of weekdays; a duration in weeks
	// needs how many days that is.
	week := bits.OnesCount8(uint8(st.Program.WorkingWeek))
	var ops []store.ProgramOp
	var from []int // the call each edit came from
	temps := map[string]bool{}
	for i, call := range in.Calls {
		var e programEdit
		dec := json.NewDecoder(bytes.NewReader(call.Fields))
		dec.DisallowUnknownFields()
		if err := dec.Decode(&e); err != nil {
			return opApplied{}, &opError{Index: i, Message: "The fields are not in the shape a Program edit takes."}
		}
		if e.WorkingWeek != nil && e.Op == "set_working_week" {
			week = bits.OnesCount8(uint8(*e.WorkingWeek))
		}
		if e.Duration != nil {
			days, ok := figures.WorkingDays(*e.Duration, week)
			if !ok {
				return opApplied{}, &opError{Index: i, Field: "duration", Message: fmt.Sprintf("%q is not a duration. Write it like \"2 weeks\" or \"10 days\".", *e.Duration)}
			}
			e.DurationDays = store.OptInt{Set: true, Value: &days}
		}
		// Short ids become full ids; a temp_id from this batch is left alone.
		full := func(ref *string) *opError {
			if *ref == "" || temps[*ref] || uuidPattern.MatchString(*ref) {
				return nil
			}
			a, err := byRef(st.Activities, func(a store.ProgramActivity) string { return a.ID }, *ref, "Program row")
			if err != nil {
				return &opError{Index: i, Field: "id", Message: err.Error()}
			}
			*ref = a.ID
			return nil
		}
		refs := []*string{&e.ID, &e.ParentID, &e.AfterID, &e.PredID, &e.SuccID}
		for j := range e.IDs {
			refs = append(refs, &e.IDs[j])
		}
		for _, ref := range refs {
			if bad := full(ref); bad != nil {
				return opApplied{}, bad
			}
		}
		if e.Count != 0 {
			if e.Op != "add" || e.Label == nil || len(e.Labels) > 0 || e.TempID != "" || e.Count < 2 || e.Count > 50 {
				return opApplied{}, &opError{Index: i, Field: "count", Message: "count is for add with one label: it adds 2 to 50 rows named label 1, label 2 and so on."}
			}
			for n := 1; n <= e.Count; n++ {
				e.Labels = append(e.Labels, fmt.Sprintf("%s %d", *e.Label, n))
			}
			e.Label = nil
		}
		if len(e.Labels) > 0 && (e.Op != "add" || e.Label != nil || e.TempID != "") {
			return opApplied{}, &opError{Index: i, Field: "labels", Message: "labels is for add, in place of label and temp_id."}
		}
		if len(e.Labels) == 0 {
			e.Labels = []string{""}
		}
		for j, label := range e.Labels {
			one := e.ProgramOp
			if label != "" {
				l := label
				one.Label = &l
			}
			if one.Op == "add" && one.TempID == "" {
				// Program wants a name for each new row; chat need not invent one.
				one.TempID = fmt.Sprintf("new-%d-%d", i+1, j+1)
			}
			if one.TempID != "" {
				temps[one.TempID] = true
			}
			ops = append(ops, one)
			from = append(from, i)
		}
	}
	res, err := d.Store.ApplyProgramOps(ctx, who.OrgID, project, who.UserID, st.Program.ID, st.Program.Version, ops)
	if err != nil {
		// Program names the edit by its place in what it was sent; point
		// back at the call it came from.
		var bad *store.ProgramOpError
		if errors.As(err, &bad) && bad.Index >= 0 && bad.Index < len(from) {
			return opApplied{}, &opError{Index: from[bad.Index], Field: bad.Field, Message: bad.Message, status: 422}
		}
		return opApplied{}, err
	}
	if d.Broker != nil {
		payload, _ := json.Marshal(map[string]any{"project_id": res.Program.ProjectID, "program_id": res.Program.ID, "version": res.Program.Version})
		if _, err := d.Broker.Commit(ctx, who.OrgID, eventProgramChanged, "", string(payload)); err != nil {
			d.Log.Printf("program event: %v", err)
		}
	}
	out := opApplied{Undo: opUndo{Kind: "program"}}
	label := map[string]string{}
	for _, a := range res.Activities {
		label[a.ID] = a.Label
	}
	for i, op := range ops {
		out.Results = append(out.Results, opResult{Op: in.Calls[from[i]].Op, Summary: programSummary(op, label, st)})
	}
	return out, nil
}

func programSummary(op store.ProgramOp, label map[string]string, before store.ProgramState) string {
	name := func(id string) string {
		if l := label[id]; l != "" {
			return `"` + l + `"`
		}
		for _, a := range before.Activities {
			if a.ID == id {
				return `"` + a.Label + `"`
			}
		}
		return "a row"
	}
	days := ""
	if op.DurationDays.Set && op.DurationDays.Value != nil {
		days = fmt.Sprintf(", %d working days", *op.DurationDays.Value)
	}
	switch op.Op {
	case "add":
		l := ""
		if op.Label != nil {
			l = *op.Label
		}
		return fmt.Sprintf("Added %q under %s%s.", l, name(op.ParentID), days)
	case "update":
		return fmt.Sprintf("Updated %s%s.", name(op.ID), days)
	case "delete":
		var names []string
		for _, id := range append([]string{op.ID}, op.IDs...) {
			if id != "" {
				names = append(names, name(id))
			}
		}
		return "Deleted " + strings.Join(names, ", ") + "."
	case "link_add":
		return fmt.Sprintf("Linked %s to start after %s.", name(op.SuccID), name(op.PredID))
	case "set_start":
		if op.StartDate != nil {
			return "Set the program start to " + *op.StartDate + "."
		}
	}
	return "Program edit: " + op.Op + "."
}

func programRelease(ctx context.Context, d Deps, who opActor, project string, raw json.RawMessage) (string, error) {
	var f struct {
		ID string `json:"id"`
	}
	if err := decodeFields(raw, &f); err != nil {
		return "", err
	}
	st, err := loadProgramForOps(ctx, d, who, project)
	if err != nil {
		return "", err
	}
	a, err := byRef(st.Activities, func(a store.ProgramActivity) string { return a.ID }, f.ID, "hold point")
	if err != nil {
		return "", err
	}
	if _, err := d.Store.ReleaseHoldPoint(ctx, who.OrgID, project, who.UserID, st.Program.Version, a.ID, ""); err != nil {
		return "", err
	}
	return fmt.Sprintf("Released the hold point %q.", a.Label), nil
}

package httpapi

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"sitewise/internal/agent"
)

// The view tool (docs/plans/2026-11-18-chat-agency.md, Task 2): chat moves
// the screen. A move changes no record, so it has no Undo and no click.

// namedFilter computes, in Go, the rows of a set that is a rule, not a
// judgement ("cost lines with no budget"). Task 4 of the agency plan fills
// viewFilters; until then every filter name is refused.
type namedFilter func(ctx context.Context, d Deps, who opActor, project string) ([]string, error)

var viewFilters = map[string]map[string]namedFilter{}

// viewDialogs are the dialogs chat may open, filled in. It opens them; the
// PM presses the button inside.
var viewDialogs = map[string]string{"award": "the award dialog"}

// viewIn is the view tool's input.
type viewIn struct {
	Do     string            `json:"do"`
	Table  string            `json:"table"`
	IDs    []string          `json:"ids"`
	Filter string            `json:"filter"`
	Dialog string            `json:"dialog"`
	Fields map[string]string `json:"fields"`
	Final  bool              `json:"final"`
}

// view is the tool that moves the screen. It checks the table, every id and
// the dialog, and returns the move as a view event. It writes nothing, so
// it is done at once even in a turn that read a document.
func (b *chatBox) view(ctx context.Context, input json.RawMessage) agent.Output {
	var in viewIn
	if err := json.Unmarshal(input, &in); err != nil {
		return toolError("The view input is not valid.")
	}
	v, bad := checkView(ctx, b.d, b.who, b.project, in)
	if bad != "" {
		return toolError("%s", bad)
	}
	return agent.Output{Text: "Done on the PM's screen: " + v.Line, Views: []agent.View{v}, Final: in.Final}
}

// checkView turns the model's view call into a move the strip can make, or
// says what is wrong with it.
func checkView(ctx context.Context, d Deps, who opActor, project string, in viewIn) (agent.View, string) {
	if in.Dialog != "" || in.Do == "open" && in.Table == "" {
		return checkDialog(ctx, d, who, project, in)
	}
	t, ok := viewTables[in.Table]
	if !ok {
		return agent.View{}, "Tables are " + strings.Join(viewTableNames, ", ") + "."
	}
	v := agent.View{Do: in.Do, Table: in.Table}
	if in.Filter != "" {
		if _, ok := viewFilters[in.Table][in.Filter]; !ok {
			return agent.View{}, fmt.Sprintf("There is no named filter %q for %s yet. Pick the rows by id from a read instead.", in.Filter, t.many)
		}
	}
	switch in.Do {
	case "show":
		v.Line = "Showed the " + t.many + "."
		return v, ""
	case "clear":
		v.Line = "Cleared the " + t.many + " back to all rows."
		return v, ""
	case "select", "only", "open":
	default:
		return agent.View{}, "do is show, open, select, only or clear."
	}
	if len(in.IDs) == 0 {
		return agent.View{}, "Name the rows by id."
	}
	if len(in.IDs) > maxScreenIDs {
		return agent.View{}, fmt.Sprintf("That is %d rows; chat moves at most %d at once. Narrow the set.", len(in.IDs), maxScreenIDs)
	}
	if in.Do == "open" && len(in.IDs) != 1 {
		return agent.View{}, "open takes one id."
	}
	rows, err := tableRows(ctx, d, who, project, in.Table)
	if err != nil {
		d.Log.Printf("chat view %s failed: %v", in.Table, err)
		return agent.View{}, "The " + t.many + " could not be read."
	}
	for _, ref := range in.IDs {
		full, err := byRef(rows, func(id string) string { return id }, ref, t.one)
		if err != nil {
			return agent.View{}, err.Error() + " Nothing on screen was changed."
		}
		if !containsID(v.IDs, full) {
			v.IDs = append(v.IDs, full)
		}
	}
	switch in.Do {
	case "select":
		v.Line = "Selected " + t.count(len(v.IDs)) + "."
	case "only":
		v.Line = "Showing only " + t.count(len(v.IDs)) + "."
	default:
		v.Line = "Opened " + t.one + " " + shortID(v.IDs[0]) + "."
	}
	return v, ""
}

// checkDialog fills a dialog from ids Go has checked. The award dialog
// opens on one firm's column of one package; the PM presses Award.
func checkDialog(ctx context.Context, d Deps, who opActor, project string, in viewIn) (agent.View, string) {
	if in.Do != "open" {
		return agent.View{}, "A dialog is opened with do open."
	}
	what, ok := viewDialogs[in.Dialog]
	if !ok {
		return agent.View{}, "The dialog chat can open is award (fields package and column)."
	}
	pkgs, err := d.Store.ReadPackages(ctx, who.OrgID, project)
	if err != nil {
		return agent.View{}, "Procurement could not be read."
	}
	board, _, err := d.Store.ProcurementBoard(ctx, who.OrgID, project)
	if err != nil {
		return agent.View{}, "Procurement could not be read."
	}
	var live []string
	title := map[string]string{}
	for _, p := range pkgs {
		if p.RetiredAt == nil {
			live = append(live, p.ID)
			title[p.ID] = p.Title
		}
	}
	pkg, err := byRef(live, func(id string) string { return id }, in.Fields["package"], "package")
	if err != nil {
		return agent.View{}, err.Error()
	}
	var columns []string
	firm := map[string]string{}
	for _, c := range board[pkg] {
		if c.SubmissionID != "" {
			columns = append(columns, c.SubmissionID)
			firm[c.SubmissionID] = c.FirmName
		}
	}
	col, err := byRef(columns, func(id string) string { return id }, in.Fields["column"], "quoted firm (column) on that package")
	if err != nil {
		return agent.View{}, err.Error()
	}
	return agent.View{Do: "open", Dialog: in.Dialog, Fields: map[string]string{"package": pkg, "column": col},
		Line: fmt.Sprintf("Opened %s for %s on %s. Press Award there to award it.", what, firm[col], title[pkg])}, ""
}

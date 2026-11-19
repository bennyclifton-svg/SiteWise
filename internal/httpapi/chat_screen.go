package httpapi

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"sitewise/internal/store"
)

// Chat that can see and move the screen (docs/plans/2026-11-18-chat-agency.md,
// Task 1). The strip says what is on screen with each turn, and the view
// tool (chat_view.go) moves it. Neither changes a record, so neither has
// Undo or a click. Every id is checked against this org and project before
// the model sees it or the strip acts on it.

// maxScreenIDs is how many selected ids one turn carries. A larger
// selection is sent as a count, and chat refuses to act on it rather than
// act on a part.
const maxScreenIDs = 200

// viewTable is one table chat can see and move.
type viewTable struct {
	one, many string // "cost line", "cost lines"
	area      string // the operations area its rows belong to
	view      string // the workbench view that shows it
}

var viewTables = map[string]viewTable{
	"documents":   {"document", "documents", "documents", "documents"},
	"cost_lines":  {"cost line", "cost lines", "cost_plan", "costs"},
	"program":     {"Program row", "Program rows", "program", "program"},
	"procurement": {"package", "packages", "procurement", "procurement"},
	"register":    {"register row", "register rows", "delivery", "register"},
}

var viewTableNames = []string{"documents", "cost_lines", "program", "procurement", "register"}

func (t viewTable) count(n int) string {
	if n == 1 {
		return "1 " + t.one
	}
	return fmt.Sprintf("%d %s", n, t.many)
}

// viewNames are the workbench views the strip may report, with the words
// the model reads.
var viewNames = map[string]string{"profile": "Profile", "costs": "Cost Plan", "procurement": "Procurement", "specifications": "Specifications", "program": "Program", "register": "Register"}

// tableRows lists the full ids of a table's rows in this project, read with
// the org from the session.
func tableRows(ctx context.Context, d Deps, who opActor, project, table string) ([]string, error) {
	var ids []string
	switch table {
	case "documents":
		views, err := d.Store.ProjectDocumentViews(ctx, who.OrgID, project)
		if err != nil {
			return nil, err
		}
		for _, v := range views {
			ids = append(ids, v.ID)
		}
	case "cost_lines":
		p, err := d.Store.ReadCostPlan(ctx, who.OrgID, project, "")
		if errors.Is(err, store.ErrNotFound) {
			return nil, nil
		}
		if err != nil {
			return nil, err
		}
		for _, i := range p.Items {
			ids = append(ids, i.ID)
		}
	case "program":
		st, err := d.Store.LoadProgram(ctx, who.OrgID, project)
		if errors.Is(err, store.ErrNotFound) {
			return nil, nil
		}
		if err != nil {
			return nil, err
		}
		for _, a := range st.Activities {
			ids = append(ids, a.ID)
		}
	case "procurement":
		pkgs, err := d.Store.ReadPackages(ctx, who.OrgID, project)
		if err != nil {
			return nil, err
		}
		for _, p := range pkgs {
			if p.RetiredAt == nil {
				ids = append(ids, p.ID)
			}
		}
	case "register":
		items, err := d.Store.ReadDelivery(ctx, who.OrgID, project)
		if err != nil {
			return nil, err
		}
		for _, it := range items {
			ids = append(ids, it.ID)
		}
	case "messages":
		list, err := d.Store.MailMessages(ctx, who.OrgID, project, store.MailFilter{})
		if err != nil {
			return nil, err
		}
		for _, m := range list {
			ids = append(ids, m.ID)
		}
	default:
		return nil, fmt.Errorf("no table %q", table)
	}
	return ids, nil
}

// screenIn is the strip's report of the screen when Enter was pressed.
type screenIn struct {
	View string `json:"view"`
	Open *struct {
		Kind string `json:"kind"`
		ID   string `json:"id"`
	} `json:"open"`
	Selected *struct {
		Table string   `json:"table"`
		IDs   []string `json:"ids"`
		Count int      `json:"count"`
	} `json:"selected"`
}

// screenSel is the selection as Go checked it.
type screenSel struct {
	Table string
	IDs   []string // full ids, all in this project
	Count int      // how many the PM had selected
	Gone  int      // sent but not in this project now
}

// tooMany is a selection larger than one turn carries.
func (s *screenSel) tooMany() bool { return s != nil && s.Count > maxScreenIDs }

// readScreen checks what the strip sent and writes the one line the model
// reads. Ids not in this project and org are dropped before anything is
// written; the line carries ids and counts, never a title or any document
// text, so it is not outside text.
func readScreen(ctx context.Context, d Deps, who opActor, project string, in *screenIn) (string, *screenSel) {
	if in == nil {
		return "", nil
	}
	var parts []string
	if name, ok := viewNames[in.View]; ok {
		parts = append(parts, "On screen: "+name+".")
	}
	if in.Open != nil && in.Open.Kind == "document" && in.Open.ID != "" {
		if docs, err := tableRows(ctx, d, who, project, "documents"); err == nil && containsID(docs, in.Open.ID) {
			parts = append(parts, "Open: document "+shortID(in.Open.ID)+".")
		}
	}
	var sel *screenSel
	if s := in.Selected; s != nil && len(s.IDs) > 0 {
		t, ok := viewTables[s.Table]
		rows, err := tableRows(ctx, d, who, project, s.Table)
		if ok && err == nil {
			sent := s.IDs
			if len(sent) > maxScreenIDs {
				sent = sent[:maxScreenIDs]
			}
			sel = &screenSel{Table: s.Table, Count: max(s.Count, len(s.IDs))}
			for _, id := range sent {
				if containsID(rows, id) && !containsID(sel.IDs, id) {
					sel.IDs = append(sel.IDs, id)
				}
			}
			sel.Gone = len(sent) - len(sel.IDs)
			parts = append(parts, selectionLine(t, sel))
		}
	}
	if sel == nil {
		parts = append(parts, "Nothing is selected.")
	}
	return strings.Join(parts, " "), sel
}

func selectionLine(t viewTable, sel *screenSel) string {
	if sel.tooMany() {
		return fmt.Sprintf("Selected: %s, more than chat can act on at once. If the PM means these, ask them to narrow the selection to %d or fewer; do not act on a part.", t.count(sel.Count), maxScreenIDs)
	}
	line := "Selected: " + t.count(len(sel.IDs))
	if len(sel.IDs) > 0 {
		short := make([]string, len(sel.IDs))
		for i, id := range sel.IDs {
			short[i] = shortID(id)
		}
		line += " (" + strings.Join(short, ", ") + ")"
	}
	line += "."
	if sel.Gone > 0 {
		verb := "are"
		if sel.Gone == 1 {
			verb = "is"
		}
		line += fmt.Sprintf(" %d of the %d %s no longer there.", sel.Gone, sel.Gone+len(sel.IDs), verb)
	}
	return line
}

func containsID(ids []string, id string) bool {
	for _, x := range ids {
		if x == id {
			return true
		}
	}
	return false
}

// selectionWords are the words that point at what is selected. A batch in
// the selection's area, in a turn whose message says one of them, is
// refused when the selection was too large to send.
var selectionWords = []string{"these", "those", "them", "selected", "selection", "the lot", "all of them"}

func pointsAtSelection(text string) bool {
	text = " " + strings.ToLower(text) + " "
	for _, w := range selectionWords {
		if strings.Contains(text, " "+w+" ") || strings.Contains(text, " "+w+".") || strings.Contains(text, " "+w+",") || strings.Contains(text, " "+w+"?") || strings.Contains(text, " "+w+"!") {
			return true
		}
	}
	return false
}

// tooManySelected says why a batch must not run on a selection larger than
// one turn carries, or "" when it may.
func tooManySelected(sel *screenSel, area, asked string) string {
	if !sel.tooMany() || viewTables[sel.Table].area != area || !pointsAtSelection(asked) {
		return ""
	}
	return fmt.Sprintf("%s are selected, more than chat can act on at once. Narrow the selection to %d or fewer and ask again.", viewTables[sel.Table].count(sel.Count), maxScreenIDs)
}

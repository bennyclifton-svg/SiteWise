package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"sitewise/internal/agent"
	"sitewise/internal/reports"
)

// The view tool (docs/plans/2026-11-18-chat-agency.md, Task 2): chat moves
// the screen. A move changes no record, so it has no Undo and no click.

// viewFilters are the named filters per table (chat_filters.go): sets that
// are a rule, not a judgement, computed in Go so the model picks them in
// one round. A later plan adds its own here ("awaiting a reply", "overdue",
// "to check"); the tool description lists whatever is in this map.
var viewFilters = map[string]map[string]namedFilter{
	"cost_lines": {
		"no_budget":   {rows: costNoBudget, shown: "with no budget", none: "Every cost line in the totals has a budget."},
		"over_budget": {rows: costOverBudget, shown: "over budget on claims to date", none: "No cost line has claimed more than its budget."},
		"excluded":    {rows: costExcluded, shown: "excluded from the totals", none: "No cost line is excluded."},
	},
	"documents": {
		"not_filed":  {rows: attentionRule(attentionDocuments, reports.RuleDocumentNotFiled), shown: "not filed", none: "No document is marked not filed."},
		"not_read":   {rows: docFilter(isNotRead), shown: "the profile does not read", none: "The profile reads every document."},
		"superseded": {rows: docFilter(isSuperseded), shown: "superseded by a newer revision", none: "No document has been superseded."},
	},
	"program": {
		"no_duration": {rows: attentionRule(attentionProgram, reports.RuleProgramNoDuration), shown: "with no duration", none: "Every Program activity has a duration."},
		"critical":    {rows: programFilter(programCritical), shown: "on the critical path", none: "No Program row is on the critical path."},
	},
	"procurement": {
		"no_quote": {rows: packageNoQuote, shown: "with no quote", none: "Every package has at least one quote."},
	},
	"register": {
		"overdue":  {rows: registerFilter(registerOverdue), shown: "past the response date", none: "No RFI is past its response date."},
		"with_me":  {rows: registerFilter(registerWithMe), shown: "with you (answered, or your drafts)", none: "Nothing on the register is with you."},
		"to_check": {rows: registerFilter(registerToCheck), shown: "with a possible answer to check", none: "No register row has a possible answer to check."},
	},
}

// viewDialogs are the dialogs chat may open, filled in. It opens them; the
// PM presses the button inside.
var viewDialogs = map[string]string{"award": "the award dialog", "report": "the project report"}

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
		return agent.View{}, "Tables are " + strings.Join(viewTableNames, ", ") + ". If the PM asked for another table, say \"" + declineWords + "\" and name the screen; nothing on screen was changed."
	}
	if in.Filter != "" {
		return checkFilter(ctx, d, who, project, in, t)
	}
	v := agent.View{Do: in.Do, Table: in.Table}
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
		return agent.View{}, "The dialogs chat can open are award (fields package and column) and report (field issue yes to open its issue form)."
	}
	if in.Dialog == "report" {
		return checkReportDialog(ctx, d, who, project, in)
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

// checkFilter turns a filter name into the move it makes.
func checkFilter(ctx context.Context, d Deps, who opActor, project string, in viewIn, t viewTable) (agent.View, string) {
	f, ok := viewFilters[in.Table][in.Filter]
	if !ok {
		return agent.View{}, fmt.Sprintf("There is no filter %q for %s. The filters for %s are %s. Any other set: pick the rows by id from a read.", in.Filter, t.many, t.many, strings.Join(filterNames(in.Table), ", "))
	}
	if in.Do != "select" && in.Do != "only" {
		return agent.View{}, "A filter goes with do select or only."
	}
	if len(in.IDs) > 0 {
		return agent.View{}, "Give a filter or ids, not both."
	}
	ids, err := f.rows(ctx, d, who, project)
	var refusal filterRefusal
	if errors.As(err, &refusal) {
		return agent.View{}, refusal.Error() + " Nothing on screen was changed."
	}
	if err != nil {
		d.Log.Printf("chat filter %s %s failed: %v", in.Table, in.Filter, err)
		return agent.View{}, "The " + t.many + " could not be read."
	}
	if len(ids) == 0 {
		// Nothing to select or show alone: the table is shown and the line
		// says so, so the PM is not left with an empty table.
		return agent.View{Do: "show", Table: in.Table, Line: f.none}, ""
	}
	if len(ids) > maxScreenIDs {
		return agent.View{}, fmt.Sprintf("There are %s %s, more than chat can move at once (%d). Nothing on screen was changed; ask the PM to narrow it.", t.count(len(ids)), f.shown, maxScreenIDs)
	}
	v := agent.View{Do: in.Do, Table: in.Table, IDs: ids, Filter: in.Filter, Shown: f.shown}
	if in.Do == "select" {
		v.Line = "Selected " + t.count(len(ids)) + " " + f.shown + "."
	} else {
		v.Line = "Showing " + t.count(len(ids)) + " " + f.shown + "."
	}
	return v, ""
}

// checkReportDialog opens the project report in its reader, with its issue
// form when asked. Issuing leaves the building, so the PM presses Issue.
func checkReportDialog(ctx context.Context, d Deps, who opActor, project string, in viewIn) (agent.View, string) {
	r, err := d.Store.MonthlyReport(ctx, who.OrgID, project)
	if err != nil {
		return agent.View{}, "There is no project report yet. Prepare it first with apply reports.update. Nothing on screen was changed."
	}
	fields := map[string]string{"report": r.ID}
	line := "Opened the project report."
	if strings.EqualFold(strings.TrimSpace(in.Fields["issue"]), "yes") {
		fields["issue"] = "yes"
		line = "Opened the project report's issue form. Press Freeze and issue PDF there to issue it."
	}
	return agent.View{Do: "open", Dialog: "report", Fields: fields, Line: line}, ""
}

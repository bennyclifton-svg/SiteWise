package httpapi

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"sitewise/internal/reports"
	"sitewise/internal/store"
)

// The project report's tables, row by row (project report plan, section
// 7): add a row of the PM's own, type over a cell, keep their words against
// a moved value, hide, move, remove, give a row back to SiteWise, and set
// the period. Nothing here calls Jev or the model.

func init() {
	for _, route := range []string{"POST /reports/{r}/rows", "PATCH /reports/{r}/rows/{row}", "DELETE /reports/{r}/rows/{row}", "PUT /reports/{r}/period"} {
		routePaths[route] = pathReportWrite
	}
}

var reportRowRoutes = map[string]func(http.ResponseWriter, *http.Request, Deps){
	"POST /reports/{r}/rows":         postReportRow,
	"PATCH /reports/{r}/rows/{row}":  patchReportRow,
	"DELETE /reports/{r}/rows/{row}": deleteReportRow,
	"PUT /reports/{r}/period":        putReportPeriod,
}

type rowBody struct {
	store.RowWrite
	Version int64 `json:"version"`
}

// rowReply is the draft with the change in it, and the row it wrote.
type rowReply struct {
	Draft store.ReportDraft `json:"draft"`
	Row   string            `json:"row"`
}

func postReportRow(w http.ResponseWriter, r *http.Request, deps Deps) {
	writeReportRow(w, r, deps, "")
}

func patchReportRow(w http.ResponseWriter, r *http.Request, deps Deps) {
	if !uuidPattern.MatchString(r.PathValue("row")) {
		http.NotFound(w, r)
		return
	}
	writeReportRow(w, r, deps, r.PathValue("row"))
}

func writeReportRow(w http.ResponseWriter, r *http.Request, deps Deps, row string) {
	session, ok := reportSession(w, r, deps, true)
	if !ok {
		return
	}
	var body rowBody
	if err := readProposalJSON(w, r, deps.MaxBodyBytes, &body); err != nil || len(body.Cells) > 40 || len(body.Keep) > 40 {
		http.Error(w, "invalid body", http.StatusBadRequest)
		return
	}
	if row != "" {
		body.Row, body.Key = row, ""
	}
	if body.Table == "" && body.Row == "" {
		http.Error(w, "a table or a row is needed", http.StatusBadRequest)
		return
	}
	draft, id, err := deps.Store.WriteReportRow(r.Context(), session.OrgID, r.PathValue("r"), session.UserID, body.Version, body.RowWrite)
	if err != nil {
		reportRowError(w, err)
		return
	}
	if deps.Broker != nil {
		deps.Broker.Wake(session.OrgID)
	}
	writeJSON(w, http.StatusOK, rowReply{Draft: draft, Row: id})
}

func deleteReportRow(w http.ResponseWriter, r *http.Request, deps Deps) {
	session, ok := reportSession(w, r, deps, true)
	if !ok {
		return
	}
	var body struct {
		Version int64 `json:"version"`
	}
	if err := readProposalJSON(w, r, deps.MaxBodyBytes, &body); err != nil {
		http.Error(w, "invalid body", http.StatusBadRequest)
		return
	}
	draft, err := deps.Store.DeleteReportRow(r.Context(), session.OrgID, r.PathValue("r"), session.UserID, r.PathValue("row"), body.Version)
	if err != nil {
		reportRowError(w, err)
		return
	}
	if deps.Broker != nil {
		deps.Broker.Wake(session.OrgID)
	}
	writeJSON(w, http.StatusOK, rowReply{Draft: draft})
}

func putReportPeriod(w http.ResponseWriter, r *http.Request, deps Deps) {
	session, ok := reportSession(w, r, deps, true)
	if !ok {
		return
	}
	var body struct {
		store.ReportPeriod
		Version int64 `json:"version"`
	}
	if err := readProposalJSON(w, r, deps.MaxBodyBytes, &body); err != nil {
		http.Error(w, "invalid body", http.StatusBadRequest)
		return
	}
	draft, err := deps.Store.SetReportPeriod(r.Context(), session.OrgID, r.PathValue("r"), session.UserID, deps.ReportBuild, body.Version, body.ReportPeriod)
	if err != nil {
		reportRowError(w, err)
		return
	}
	if deps.Broker != nil {
		deps.Broker.Wake(session.OrgID)
	}
	writeJSON(w, http.StatusOK, draft)
}

func reportRowError(w http.ResponseWriter, err error) {
	if errors.Is(err, store.ErrReportRow) {
		http.Error(w, "That row, cell or period cannot be saved: check the table, the column and the dates.", http.StatusUnprocessableEntity)
		return
	}
	reportError(w, err)
}

// ---- operations (chat; the screen sends the routes itself)

// reportTables finds the project report's draft for an operation and its
// tables by id.
func reportDraftFor(ctx context.Context, d Deps, who opActor, project string) (store.Report, *store.ReportDraft, error) {
	r, err := projectReport(ctx, d, who, project, "")
	if err != nil {
		return r, nil, err
	}
	view, err := d.Store.ReadReport(ctx, who.OrgID, r.ID, d.ReportBuild)
	if err != nil {
		return r, nil, err
	}
	if view.Draft == nil {
		return r, nil, refuse("", "The project report has no draft yet. Update it first (reports.update).")
	}
	return r, view.Draft, nil
}

func findTable(dr *store.ReportDraft, id string) (*reports.Table, []string) {
	var ids []string
	for _, s := range dr.Sections {
		for _, b := range s.Blocks {
			if b.Table != nil && b.Table.ID != "" && b.Table.Meta != nil {
				if b.Table.ID == id {
					return b.Table, nil
				}
				ids = append(ids, b.Table.ID)
			}
		}
	}
	return nil, ids
}

// rowOf finds a row by its key, its id, or its first cell's words.
func rowOf(t *reports.Table, ref string) (int, error) {
	ref = strings.TrimSpace(ref)
	match := -1
	for i, f := range t.Meta {
		if f.Key == ref || f.ID != "" && f.ID == ref {
			return i, nil
		}
		if len(t.Rows[i]) > 0 && strings.EqualFold(strings.TrimSpace(t.Rows[i][0]), ref) {
			if match >= 0 {
				return -1, refuse("row", "Two rows of %s start %q; name it by its key from read report.", t.ID, ref)
			}
			match = i
		}
	}
	if match < 0 {
		return -1, refuse("row", "%s has no row %q. read report (area report) lists each row's key.", t.ID, ref)
	}
	return match, nil
}

// columnOf finds a column by its index or its heading.
func columnOf(t *reports.Table, ref string) (int, error) {
	if n, err := strconv.Atoi(strings.TrimSpace(ref)); err == nil && n >= 0 && n < len(t.Columns) {
		return n, nil
	}
	for i, c := range t.Columns {
		if strings.EqualFold(strings.TrimSpace(c), strings.TrimSpace(ref)) {
			return i, nil
		}
	}
	return -1, refuse("column", "%s's columns are %s.", t.ID, strings.Join(t.Columns, ", "))
}

func rowTarget(t *reports.Table, i int) map[string]any {
	f := t.Meta[i]
	if f.ID != "" && (f.Kind == reports.RowOwn && f.Key == "" || f.Key == "") {
		return map[string]any{"row": f.ID}
	}
	return map[string]any{"table": t.ID, "key": f.Key}
}

func reportRowOperations() []operation {
	tableField := `table (a table id from read report, e.g. approvals, eot, change)`
	setCell := forward{Name: "reports.set_cell", Undo: true,
		About:   "Type over one cell of the project report's tables, or put SiteWise's own words back (clear true). The PM's words stay through Update; when SiteWise's value under them moves, the cell waits for the PM to keep theirs or take SiteWise's. A record (an EOT result, a variation, a contract sum) is changed at source with that screen's operation, not here.",
		Fields:  tableField + `; row (its key or its first cell's words); column (its heading or index); text (the words; a number in them must be one the PM gave or a read printed); clear (true takes SiteWise's words back); source (a document id the words were read from)`,
		Figures: map[string]string{"text": "value"},
		Route:   "POST /reports/{r}/rows", Also: []string{"PATCH /reports/{r}/rows/{row}"},
		Send: func(ctx context.Context, d Deps, who opActor, project string, f map[string]any) (sent, error) {
			r, dr, err := reportDraftFor(ctx, d, who, project)
			if err != nil {
				return sent{}, err
			}
			t, ids := findTable(dr, fieldText(f, "table"))
			if t == nil {
				return sent{}, refuse("table", "The report's tables are %s.", strings.Join(ids, ", "))
			}
			i, err := rowOf(t, fieldText(f, "row"))
			if err != nil {
				return sent{}, err
			}
			col, err := columnOf(t, fieldText(f, "column"))
			if err != nil {
				return sent{}, err
			}
			body := rowTarget(t, i)
			var text any = fieldText(f, "text")
			if clear, _ := f["clear"].(bool); clear {
				text = nil
			} else if fieldText(f, "text") == "" {
				return sent{}, refuse("text", "text is needed, or clear true.")
			}
			body["cells"] = map[string]any{strconv.Itoa(col): text}
			body["version"] = dr.Version
			if src := fieldText(f, "source"); src != "" {
				body["source"] = src
			}
			if row, ok := body["row"].(string); ok {
				delete(body, "row")
				return sent{Route: "PATCH /reports/{r}/rows/{row}", Path: map[string]string{"r": r.ID, "row": row}, Body: body}, nil
			}
			return sent{Path: map[string]string{"r": r.ID}, Body: body}, nil
		},
		Done: func(f map[string]any) string {
			if clear, _ := f["clear"].(bool); clear {
				return fmt.Sprintf("Put SiteWise's words back in %s, %s.", fieldText(f, "table"), fieldText(f, "row"))
			}
			return fmt.Sprintf("Wrote %s, %s, %s on the project report.", fieldText(f, "table"), fieldText(f, "row"), fieldText(f, "column"))
		},
	}
	addRow := forward{Name: "reports.add_row", Undo: true,
		About:   "Add a row of the PM's own to one of the project report's tables. Update never changes it.",
		Fields:  tableField + `; cells (a list of words, one per column in order); after (a row key to put it after; leave out for the end); source (a document id the words were read from)`,
		Figures: map[string]string{"cells": "value"},
		Route:   "POST /reports/{r}/rows",
		Send: func(ctx context.Context, d Deps, who opActor, project string, f map[string]any) (sent, error) {
			r, dr, err := reportDraftFor(ctx, d, who, project)
			if err != nil {
				return sent{}, err
			}
			t, ids := findTable(dr, fieldText(f, "table"))
			if t == nil {
				return sent{}, refuse("table", "The report's tables are %s.", strings.Join(ids, ", "))
			}
			list, _ := f["cells"].([]any)
			if len(list) == 0 || len(list) > len(t.Columns) {
				return sent{}, refuse("cells", "cells is a list of up to %d words: %s.", len(t.Columns), strings.Join(t.Columns, ", "))
			}
			cells := map[string]any{}
			for i, v := range list {
				cells[strconv.Itoa(i)] = strings.TrimSpace(fmt.Sprint(v))
			}
			body := map[string]any{"table": t.ID, "cells": cells, "version": dr.Version}
			if after := fieldText(f, "after"); after != "" {
				i, err := rowOf(t, after)
				if err != nil {
					return sent{}, err
				}
				body["after"] = t.Meta[i].Key
				if t.Meta[i].Key == "" {
					body["after"] = t.Meta[i].ID
				}
			}
			if src := fieldText(f, "source"); src != "" {
				body["source"] = src
			}
			return sent{Path: map[string]string{"r": r.ID}, Body: body}, nil
		},
		Done: func(f map[string]any) string {
			return "Added a row to " + fieldText(f, "table") + " on the project report."
		},
	}
	rowAction := func(name, about string, body func(t *reports.Table, i int) (sent, error)) forward {
		return forward{Name: name, Undo: true, About: about,
			Fields: tableField + `; row (its key or its first cell's words)` + map[bool]string{true: `; direction (up|down)`}[name == "reports.move_row"],
			Route:  "POST /reports/{r}/rows", Also: []string{"PATCH /reports/{r}/rows/{row}", "DELETE /reports/{r}/rows/{row}"},
			Send: func(ctx context.Context, d Deps, who opActor, project string, f map[string]any) (sent, error) {
				r, dr, err := reportDraftFor(ctx, d, who, project)
				if err != nil {
					return sent{}, err
				}
				t, ids := findTable(dr, fieldText(f, "table"))
				if t == nil {
					return sent{}, refuse("table", "The report's tables are %s.", strings.Join(ids, ", "))
				}
				// A hidden row is not in the table: restoring names it by key.
				if name == "reports.restore_row" {
					for h, key := range t.HiddenKeys {
						if key == strings.TrimSpace(fieldText(f, "row")) && h < len(t.HiddenIDs) {
							return sent{Route: "DELETE /reports/{r}/rows/{row}", Path: map[string]string{"r": r.ID, "row": t.HiddenIDs[h]}, Body: map[string]any{"version": dr.Version}}, nil
						}
					}
				}
				i, err := rowOf(t, fieldText(f, "row"))
				if err != nil {
					return sent{}, err
				}
				out, err := body(t, i)
				if err != nil {
					return sent{}, err
				}
				if out.Path == nil {
					out.Path = map[string]string{}
				}
				out.Path["r"] = r.ID
				if m, ok := out.Body.(map[string]any); ok {
					m["version"] = dr.Version
					if dir := fieldText(f, "direction"); name == "reports.move_row" {
						if dir != "up" && dir != "down" {
							return sent{}, refuse("direction", "direction is up or down.")
						}
						m["move"] = dir
					}
				}
				return out, nil
			},
			Done: func(f map[string]any) string {
				return map[string]string{"reports.remove_row": "Removed", "reports.restore_row": "Gave SiteWise back", "reports.move_row": "Moved"}[name] + " " + fieldText(f, "row") + " in " + fieldText(f, "table") + "."
			}}
	}
	remove := rowAction("reports.remove_row", "Remove a row from the project report's table: a row of the PM's own is deleted; a row SiteWise made is hidden, and the table says how many are hidden.", func(t *reports.Table, i int) (sent, error) {
		f := t.Meta[i]
		if f.Kind == reports.RowOwn && f.Key == "" {
			return sent{Route: "DELETE /reports/{r}/rows/{row}", Path: map[string]string{"row": f.ID}, Body: map[string]any{}}, nil
		}
		b := rowTarget(t, i)
		b["hidden"] = true
		return sent{Body: b}, nil
	})
	restore := rowAction("reports.restore_row", "Give a row SiteWise made back to SiteWise: the PM's words over it, its place and its hidden mark go.", func(t *reports.Table, i int) (sent, error) {
		f := t.Meta[i]
		if f.ID == "" || f.Key == "" {
			return sent{}, refuse("row", "That row is SiteWise's already, or is the PM's own (reports.remove_row deletes it).")
		}
		return sent{Route: "DELETE /reports/{r}/rows/{row}", Path: map[string]string{"row": f.ID}, Body: map[string]any{}}, nil
	})
	move := rowAction("reports.move_row", "Move a row of the project report's table one place up or down.", func(t *reports.Table, i int) (sent, error) {
		return sent{Body: rowTarget(t, i)}, nil
	})
	period := forward{Name: "reports.period", Click: true,
		About:   "Set the period the project report's draft reports on. The draft is rebuilt for the new window; it waits for the PM's click.",
		Fields:  `from (the first day, YYYY-MM-DD; leave out for "to date"); to (the last day); label (words such as "October 2026"; leave out and SiteWise names it from the dates)`,
		Figures: map[string]string{"from": "date", "to": "date"},
		Route:   "PUT /reports/{r}/period",
		Send: func(ctx context.Context, d Deps, who opActor, project string, f map[string]any) (sent, error) {
			r, dr, err := reportDraftFor(ctx, d, who, project)
			if err != nil {
				return sent{}, err
			}
			return sent{Path: map[string]string{"r": r.ID}, Body: map[string]any{"from": fieldText(f, "from"), "to": fieldText(f, "to"), "label": fieldText(f, "label"), "version": dr.Version}}, nil
		},
		Done: func(f map[string]any) string { return "Set the project report's period." },
	}
	return []operation{setCell.operation(), addRow.operation(), remove.operation(), restore.operation(), move.operation(), period.operation(), placeImageOperation()}
}

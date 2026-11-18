package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"sitewise/internal/reports"
	"sitewise/internal/store"
)

// The project report's operations (docs/plans/2026-11-17-monthly-report.md,
// "From chat" and Task 10). Each runs the screen's own route with the
// caller's session. A report route is keyed by the report, not the
// project, so each first finds the report among this project's own
// reports: an id from another project or org is refused before any route
// runs. Issuing is not here: it leaves the building, so chat opens the
// reader's issue form and the PM presses Issue (ops_routes.go).

// commentSections are the sections that carry a comment block.
var commentSections = []string{"summary", "cost", "program", "procurement", "registers", "next"}

func reportOperations() []operation {
	comment := forward{Name: "reports.comment", Undo: true,
		About:   "Write or clear the PM's comment under one section of the project report. Every figure in the text must be one a read printed this turn or the PM typed.",
		Fields:  `section (` + strings.Join(commentSections, "|") + `); text (the comment); clear (true restores SiteWise's own sentence); report (id; leave out for the project report)`,
		Figures: map[string]string{"text": "value"},
		Route:   "PUT /reports/{r}/edits/{target}", Also: []string{"DELETE /reports/{r}/edits/{target}"},
		Send: func(ctx context.Context, d Deps, who opActor, project string, f map[string]any) (sent, error) {
			r, err := projectReport(ctx, d, who, project, fieldText(f, "report"))
			if err != nil {
				return sent{}, err
			}
			section := fieldText(f, "section")
			if !containsID(commentSections, section) {
				return sent{}, refuse("section", "section is %s.", strings.Join(commentSections, ", "))
			}
			view, err := d.Store.ReadReport(ctx, who.OrgID, r.ID, d.ReportBuild)
			if err != nil {
				return sent{}, err
			}
			if view.Draft == nil {
				return sent{}, refuse("", "The project report has no draft yet. Update it first (reports.update).")
			}
			target := "comment:" + section
			var block *reports.Block
			for i := range view.Draft.Sections {
				for j := range view.Draft.Sections[i].Blocks {
					if view.Draft.Sections[i].Blocks[j].ID == target {
						block = &view.Draft.Sections[i].Blocks[j]
					}
				}
			}
			if block == nil {
				return sent{}, refuse("section", "This report has no %s comment.", section)
			}
			path := map[string]string{"r": r.ID, "target": target}
			if clear, _ := f["clear"].(bool); clear {
				if !block.Edited {
					return sent{}, refuse("clear", "The %s comment is already SiteWise's own sentence.", section)
				}
				return sent{Route: "DELETE /reports/{r}/edits/{target}", Path: path, Body: map[string]any{"version": view.Draft.Version}}, nil
			}
			text := fieldText(f, "text")
			if text == "" {
				return sent{}, refuse("text", "text is needed, or clear true.")
			}
			return sent{Path: path, Body: map[string]any{"text": text, "version": view.Draft.Version}}, nil
		},
		Done: func(f map[string]any) string {
			if clear, _ := f["clear"].(bool); clear {
				return "Cleared the " + fieldText(f, "section") + " comment on the project report; it shows SiteWise's sentence again."
			}
			return "Wrote the " + fieldText(f, "section") + " comment on the project report."
		},
	}
	return []operation{{
		Name:   "reports.update",
		About:  "Prepare the project report: create it if the project has none, then build its draft from the screens' figures (Update). The draft is derived from the records, so there is nothing to undo.",
		Fields: `report (id; leave out for the project report)`,
		Routes: []string{"POST /reports/{r}/draft"},
		// Update rebuilds a draft from saved records and keeps every comment:
		// it changes no record of the PM's, so it applies at once and has no
		// Undo, as Update on the screen.
		NoUndo: true,
		Run:    updateReport,
	}, comment.operation()}
}

func updateReport(ctx context.Context, d Deps, who opActor, project string, raw json.RawMessage) (string, error) {
	var in struct {
		Report string `json:"report"`
	}
	if err := decodeFields(raw, &in); err != nil {
		return "", err
	}
	r, err := projectReport(ctx, d, who, project, in.Report)
	if errors.Is(err, store.ErrNotFound) && in.Report == "" {
		// The screen's own create: one project report per project.
		status, body, err := callRoute(ctx, d, project, "POST /projects/{id}/reports", sent{Body: map[string]string{"kind": reports.KindMonthly}})
		if err != nil {
			return "", err
		}
		if status != http.StatusCreated {
			return "", routeRefusal(status, body)
		}
		if err := json.Unmarshal(body, &r); err != nil {
			return "", err
		}
	} else if err != nil {
		return "", err
	}
	status, body, err := callRoute(ctx, d, project, "POST /reports/{r}/draft", sent{Path: map[string]string{"r": r.ID}, Body: map[string]bool{"use_last_completed": true}})
	if err != nil {
		return "", err
	}
	if status != http.StatusOK {
		return "", routeRefusal(status, body)
	}
	var draft store.ReportDraft
	if err := json.Unmarshal(body, &draft); err != nil {
		return "", err
	}
	what := "the project report"
	if r.Kind != reports.KindMonthly {
		what = r.Title
	}
	return fmt.Sprintf("Updated %s: draft %d, from the screens' figures as at today.", what, draft.Number), nil
}

// projectReport finds a report of this project: the project report when ref
// is empty, else the report whose id ref is. ErrNotFound when the project
// has no project report; another project's or org's id is never found.
func projectReport(ctx context.Context, d Deps, who opActor, project, ref string) (store.Report, error) {
	if ref == "" {
		return d.Store.MonthlyReport(ctx, who.OrgID, project)
	}
	list, err := d.Store.ListReports(ctx, who.OrgID, project)
	if err != nil {
		return store.Report{}, err
	}
	return byRef(list, func(r store.Report) string { return r.ID }, ref, "report")
}

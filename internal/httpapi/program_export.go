package httpapi

import (
	"bytes"
	"mime"
	"net/http"

	engine "sitewise/internal/program"
	"sitewise/internal/store"
	"sitewise/internal/xlsx"
)

const pathProgramExport = "program_export"

// getProgramExport is GET /program/export?format=xlsx|csv&view=expanded|
// collapsed (plan 8, WP-P11): the program as a table in the view the
// browser is showing, expanded by default. It reads the stored inputs, the
// engine computes the dates as on any read, and internal/program formats
// the rows. No Jev. Another org's project, or one with no program, is a
// bare 404.
func getProgramExport(w http.ResponseWriter, r *http.Request, d Deps) {
	s, ok := programSession(w, r, d, false)
	if !ok {
		return
	}
	q := r.URL.Query()
	format, view := q.Get("format"), q.Get("view")
	if format == "" {
		format = "xlsx"
	}
	if view == "" {
		view = engine.ViewExpanded
	}
	if format != "xlsx" && format != "csv" || view != engine.ViewExpanded && view != engine.ViewCollapsed {
		http.Error(w, "format is xlsx or csv; view is expanded or collapsed", http.StatusBadRequest)
		return
	}
	ctx, project := r.Context(), r.PathValue("id")
	p, err := d.Store.GetProject(ctx, s.OrgID, project)
	if err != nil {
		programError(w, r, d, s.OrgID, store.ProgramResult{}, err)
		return
	}
	st, err := d.Store.LoadProgram(ctx, s.OrgID, project)
	if err != nil {
		programError(w, r, d, s.OrgID, store.ProgramResult{}, err)
		return
	}
	origin := make(map[string]string, len(st.Activities))
	for _, a := range st.Activities {
		origin[a.ID] = a.Origin
	}
	rows := engine.ExportRows(st.Input(), st.Computed, origin, view)
	name := safeFileName(p.Name) + " program"
	var buf bytes.Buffer
	if format == "csv" {
		if err := engine.WriteExportCSV(&buf, rows); err != nil {
			programError(w, r, d, s.OrgID, store.ProgramResult{}, err)
			return
		}
		w.Header().Set("Content-Type", "text/csv; charset=utf-8")
		name += ".csv"
	} else {
		if err := xlsx.WriteXLSX(&buf, []xlsx.Sheet{engine.ExportSheet(rows)}); err != nil {
			programError(w, r, d, s.OrgID, store.ProgramResult{}, err)
			return
		}
		w.Header().Set("Content-Type", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
		name += ".xlsx"
	}
	w.Header().Set("Content-Disposition", mime.FormatMediaType("attachment", map[string]string{"filename": name}))
	_, _ = w.Write(buf.Bytes())
}

package specifications

import (
	"fmt"
	"strings"
	"time"

	"sitewise/internal/reports"
	"sitewise/internal/reports/render"
)

// PDF renders the same accepted values and unresolved decisions as the screen.
// It uses the existing embedded fonts and renderer, without a new dependency.
func PDF(s Snapshot, title, purpose string, date time.Time) ([]byte, error) {
	section := reports.Section{ID: "specifications", Title: s.ProjectName}
	section.Blocks = append(section.Blocks, reports.Block{Text: "Issue purpose: " + purpose + ". Unresolved choices and source claims require review before construction."})
	for _, c := range s.Coverage {
		section.Blocks = append(section.Blocks, reports.Block{ID: c.ID, Label: c.SystemID + " — specification coverage", Text: CoverageText(c)})
	}
	for _, row := range Assemble(s) {
		lines := []string{row.Location + "; " + strings.TrimSpace(row.Surface+" "+row.Action) + "; " + row.Inclusion + "; " + row.Selection}
		if row.Authority == "external" {
			lines = append(lines, "Maintained externally. Confirmed readings do not transfer specification responsibility or approve a product.")
		}
		if row.ReviewStatus == "proposed" {
			lines = append(lines, "Item identity awaits review.")
		}
		if row.Quantity != nil {
			lines = append(lines, fmt.Sprintf("Quantity: %g %s", *row.Quantity, row.Unit))
		}
		if row.Scheme != "" {
			lines = append(lines, "Scheme: "+row.Scheme)
		}
		if row.OptionGroup != "" {
			lines = append(lines, "Alternative group: "+row.OptionGroup)
		}
		if row.Description != "" {
			lines = append(lines, row.Description)
		}
		for _, f := range row.Fields {
			line := f.Label + " (" + f.Meaning + "): " + f.Value
			if f.Source.DocumentID != "" {
				line += fmt.Sprintf(" [Document %s, page %d", f.Source.DocumentID, f.Source.Page)
				if f.Source.Location != "" {
					line += "; " + f.Source.Location
				}
				if f.Source.Unavailable {
					line += "; source unavailable"
				}
				if f.Source.Stale {
					line += "; revision review required"
				}
				line += "]"
			}
			lines = append(lines, line)
		}
		lines = append(lines, row.Relationships...)
		if row.Pending > 0 {
			lines = append(lines, fmt.Sprintf("%d proposed values await review.", row.Pending))
		}
		section.Blocks = append(section.Blocks, reports.Block{ID: row.TypeID + "/" + row.ApplicationID, Label: strings.TrimSpace(row.Code + " " + row.Title), Text: strings.Join(lines, "\n")})
	}
	out, err := render.PDF(reports.IssueSnapshot{SchemaVersion: 1, Kind: "specification", Title: title, ReportingDate: date.Format("2006-01-02"), RendererVersion: render.Version, SourceRevisions: reports.SourceState{TemplateID: "tpl.specifications"}, Sections: []reports.Section{section}})
	return out.PDF, err
}

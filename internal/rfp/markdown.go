package rfp

import (
	"strings"

	"sitewise/internal/reports"
)

// Markdown renders report sections in the golden RFP layout: headings,
// paragraphs and pipe tables. Tests compare it with reviewed snapshots.
func Markdown(sections []reports.Section) string {
	var b strings.Builder
	for _, s := range sections {
		if len(s.Blocks) == 0 {
			continue
		}
		b.WriteString("### " + s.Title + "\n\n")
		for _, block := range s.Blocks {
			text := strings.TrimSpace(block.Text)
			if block.Label != "" {
				text = "**" + block.Label + "**: " + text
			}
			if text != "" {
				b.WriteString(text + "\n\n")
			}
			if block.Table != nil {
				b.WriteString(markdownTable(*block.Table))
				b.WriteString("\n")
			}
		}
	}
	return b.String()
}

func markdownTable(t reports.Table) string {
	var b strings.Builder
	cell := func(s string) string { return strings.ReplaceAll(strings.ReplaceAll(s, "|", `\|`), "\n", " ") }
	row := func(cells []string) {
		parts := make([]string, len(cells))
		for i, c := range cells {
			parts[i] = cell(c)
		}
		b.WriteString("| " + strings.Join(parts, " | ") + " |\n")
	}
	row(t.Columns)
	sep := make([]string, len(t.Columns))
	for i := range sep {
		sep[i] = "---"
	}
	b.WriteString("|" + strings.Join(sep, "|") + "|\n")
	for _, r := range t.Rows {
		row(r)
	}
	return b.String()
}

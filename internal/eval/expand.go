package eval

import (
	"bytes"
	"context"
	"os"

	"sitewise/internal/identity"
	"sitewise/internal/intake"
	"sitewise/internal/store"
)

// maxSheets matches the application's drawing-set limit (intake.ExpandDrawing).
const maxSheets = 200

// Sheet expansion outcomes. A drawing set that stops keeps its reason, so a
// failure is counted rather than silently dropped (D-36, AT-35).
const (
	SheetsPublished  = "published"
	SheetsTooMany    = "too_many_sheets"
	SheetsNoText     = "page_without_text"
	SheetsNotDrawing = "page_not_drawing"
)

// SheetRun is what the expansion of one filed multi-page drawing did.
type SheetRun struct {
	Pages   int    `json:"pages"`
	Asked   int    `json:"asked"`
	Outcome string `json:"outcome"`
}

// expands reports whether the application would split this filed case into
// sheets: a multi-page PDF filed as a drawing with an amber or green band
// (store.CommitFiling).
func expands(c Case, pages int, decisions []intake.Decision) bool {
	if c.Format != "pdf" || pages <= 1 {
		return false
	}
	for _, d := range decisions {
		if d.Field == intake.FieldKind && d.Value == "drawing" && (d.Band == "amber" || d.Band == "green") {
			return true
		}
	}
	return false
}

// expand files every sheet of a drawing set the way intake.ExpandDrawing
// does: the same single-page extraction, a draft with no filename, one Jev
// call per sheet and the same stop rules. The calls are recorded and replayed
// with the others, so a whole-file replay contains every request the
// application makes. Sheet answers are counted, not scored: the answer keys
// score only first pages today.
func (e Evaluator) expand(ctx context.Context, c Case, pages int, priors []store.NumberedDocument, run *RunResult) (*SheetRun, error) {
	out := &SheetRun{Pages: pages}
	if pages > maxSheets {
		out.Outcome = SheetsTooMany
		return out, nil
	}
	f, err := os.Open(c.Path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	info, err := f.Stat()
	if err != nil {
		return nil, err
	}
	for page := 1; page <= pages; page++ {
		var pdf bytes.Buffer
		if _, err := identity.PDFSheet(ctx, f, info.Size(), page, maxSheets, &pdf); err != nil {
			return nil, err
		}
		text, err := identity.Extract(ctx, "pdf", bytes.NewReader(pdf.Bytes()), int64(pdf.Len()), identity.Limits{MaxPages: 1})
		if err != nil {
			return nil, err
		}
		if !text.TextLayer {
			out.Outcome = SheetsNoText
			return out, nil
		}
		draft := intake.NewDraft(e.Catalog, "", text, intake.Harvest("", text), nil)
		draft.Plan(priors, c.ID)
		if call, ok := draft.Call(); ok {
			if e.Deadline > 0 {
				call.Deadline = e.Deadline
			}
			result, err := e.Asker.Ask(ctx, call)
			run.Calls++
			out.Asked++
			if err != nil {
				if ctx.Err() != nil {
					return nil, ctx.Err()
				}
				run.Errors++
			}
			draft.Apply(result, err, e.Thresholds)
		}
		kind, band := "", ""
		for _, d := range draft.Decisions() {
			if d.Field == intake.FieldKind {
				kind, band = d.Value, d.Band
			}
		}
		if !store.DrawingSetPageKind(kind, band) {
			out.Outcome = SheetsNotDrawing
			return out, nil
		}
	}
	out.Outcome = SheetsPublished
	return out, nil
}

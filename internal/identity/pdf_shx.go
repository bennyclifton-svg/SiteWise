package identity

import (
	"bytes"
	"context"
	"io"
	"math"
	"strings"

	"github.com/klippa-app/go-pdfium"
	"github.com/klippa-app/go-pdfium/enums"
	"github.com/klippa-app/go-pdfium/references"
	"github.com/klippa-app/go-pdfium/requests"
)

// AutoCAD exports SHX glyphs as drawing paths, with their searchable text in
// annotations. Ordinary review comments are not part of the printed sheet.
func pdfPageRuns(ctx context.Context, inst pdfium.Pdfium, doc references.FPDF_DOCUMENT, pageIndex, maxRuns int) ([]Run, error) {
	runs, err := pdfPageTextRuns(ctx, inst, doc, pageIndex, maxRuns)
	if err != nil || maxRuns <= 0 || len(runs) >= maxRuns {
		return runs, err
	}
	page := requests.Page{ByIndex: &requests.PageByIndex{Document: doc, Index: pageIndex}}
	count, err := inst.FPDFPage_GetAnnotCount(&requests.FPDFPage_GetAnnotCount{Page: page})
	if err != nil {
		return nil, malformed(err)
	}
	// Bound annotation traversal independently of the returned identity budget.
	nativeCount := len(runs)
	printedStamp := false
	for i := 0; i < min(count.Count, 4096); i++ {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		a, err := inst.FPDFPage_GetAnnot(&requests.FPDFPage_GetAnnot{Page: page, Index: i})
		if err != nil {
			continue
		}
		subtype, subtypeErr := inst.FPDFAnnot_GetSubtype(&requests.FPDFAnnot_GetSubtype{Annotation: a.Annotation})
		if subtypeErr == nil && subtype.Subtype == enums.FPDF_ANNOT_SUBTYPE_STAMP {
			printedStamp = true
		}
		run, ok := pdfSHXRun(inst, a.Annotation, pageIndex+1)
		inst.FPDFPage_CloseAnnot(&requests.FPDFPage_CloseAnnot{Annotation: a.Annotation})
		if ok {
			duplicate := false
			for _, old := range runs {
				if old.Text == run.Text && math.Abs(old.Source.X-run.Source.X) < 3 && math.Abs(old.Source.Y-run.Source.Y) < 3 {
					duplicate = true
					break
				}
			}
			if !duplicate {
				runs = append(runs, run)
			}
		}
	}
	if printedStamp {
		// Flatten only this in-memory reader document, never the stored original.
		// Printed approval/regulated-design stamps contain visible text in an
		// appearance stream which the ordinary page text API does not expose.
		if flat, err := pdfPrintedStampRuns(ctx, inst, doc, pageIndex, maxRuns); err == nil {
			for _, r := range flat {
				known := false
				for _, native := range runs[:nativeCount] {
					if native.Text == r.Text && math.Abs(native.Source.X-r.Source.X) < 1 && math.Abs(native.Source.Y-r.Source.Y) < 1 {
						known = true
						break
					}
				}
				if !known {
					r.Source.Annotation = "Printed stamp appearance"
					runs = append(runs, r)
				}
			}
		}
	}
	if len(runs) > maxRuns {
		// Adding CAD text must not displace an existing native title block.
		// A page already at its native budget returns above without annotation
		// traversal. Annotations fill available space, never displace native text.
		annotationBudget := maxRuns - nativeCount
		nativeBudget := maxRuns - annotationBudget
		kept := append([]Run(nil), runs[:nativeBudget/2]...)
		kept = append(kept, runs[nativeCount-(nativeBudget-nativeBudget/2):nativeCount]...)
		kept = append(kept, runs[nativeCount:nativeCount+annotationBudget/2]...)
		kept = append(kept, runs[len(runs)-(annotationBudget-annotationBudget/2):]...)
		runs = kept
	}
	return runs, nil
}

// PDFium flatten updates the content stream; reopen the single-page copy to
// parse its new text objects. Never serialize or alter the entire source pack.
func pdfPrintedStampRuns(ctx context.Context, inst pdfium.Pdfium, doc references.FPDF_DOCUMENT, index, maxRuns int) ([]Run, error) {
	copyDoc, err := inst.FPDF_CreateNewDocument(&requests.FPDF_CreateNewDocument{})
	if err != nil {
		return nil, err
	}
	defer inst.FPDF_CloseDocument(&requests.FPDF_CloseDocument{Document: copyDoc.Document})
	if _, err = inst.FPDF_ImportPagesByIndex(&requests.FPDF_ImportPagesByIndex{Source: doc, Destination: copyDoc.Document, PageIndices: []int{index}}); err != nil {
		return nil, err
	}
	page := requests.Page{ByIndex: &requests.PageByIndex{Document: copyDoc.Document, Index: 0}}
	if _, err = inst.FPDFPage_Flatten(&requests.FPDFPage_Flatten{Page: page, Usage: requests.FPDFPage_FlattenUsagePrint}); err != nil {
		return nil, err
	}
	var body bytes.Buffer
	writer := &pdfSheetWriter{ctx: ctx, dst: &body, remaining: 32 << 20}
	if _, err = inst.FPDF_SaveAsCopy(&requests.FPDF_SaveAsCopy{Document: copyDoc.Document, Flags: requests.SaveFlagNoIncremental, FileWriter: writer}); err != nil {
		return nil, err
	}
	if writer.err != nil {
		return nil, writer.err
	}
	data := body.Bytes()
	if len(data) == 0 {
		return nil, io.ErrUnexpectedEOF
	}
	flat, err := inst.OpenDocument(&requests.OpenDocument{File: &data})
	if err != nil {
		return nil, err
	}
	defer inst.FPDF_CloseDocument(&requests.FPDF_CloseDocument{Document: flat.Document})
	runs, err := pdfPageTextRuns(ctx, inst, flat.Document, 0, maxRuns)
	for i := range runs {
		runs[i].Source.Page = index + 1
	}
	return runs, err
}

func pdfSHXRun(inst pdfium.Pdfium, annotation references.FPDF_ANNOTATION, page int) (Run, bool) {
	label, err := inst.FPDFAnnot_GetStringValue(&requests.FPDFAnnot_GetStringValue{Annotation: annotation, Key: "T"})
	if err != nil || label.Value != "AutoCAD SHX Text" {
		return Run{}, false
	}
	text, err := inst.FPDFAnnot_GetStringValue(&requests.FPDFAnnot_GetStringValue{Annotation: annotation, Key: "Contents"})
	if err != nil || len(text.Value) > 4096 || strings.TrimSpace(text.Value) == "" {
		return Run{}, false
	}
	box, err := inst.FPDFAnnot_GetRect(&requests.FPDFAnnot_GetRect{Annotation: annotation})
	if err != nil {
		return Run{}, false
	}
	x, right := math.Min(float64(box.Rect.Left), float64(box.Rect.Right)), math.Max(float64(box.Rect.Left), float64(box.Rect.Right))
	y, top := math.Min(float64(box.Rect.Bottom), float64(box.Rect.Top)), math.Max(float64(box.Rect.Bottom), float64(box.Rect.Top))
	if right <= x || top <= y {
		return Run{}, false
	}
	return Run{Text: trimKept(text.Value), Source: Source{Page: page, X: x, Y: y, Width: right - x, Height: top - y, Annotation: "AutoCAD SHX Text"}}, true
}

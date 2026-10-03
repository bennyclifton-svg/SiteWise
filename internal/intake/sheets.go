package intake

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"

	"sitewise/internal/identity"
	"sitewise/internal/jev"
	"sitewise/internal/store"
)

// ExpandDrawing runs off the upload clock. Every sheet is judged independently
// in one background fan-out; only a wholly admitted drawing set is published.
// https://docs.typesafe.ai/patterns/fan-out
func (r *Runner) ExpandDrawing(ctx context.Context, orgID, documentID string) error {
	expansion, err := r.store.DrawingExpansion(ctx, orgID, documentID)
	if errors.Is(err, store.ErrNotFound) {
		return nil
	}
	if err != nil {
		return err
	}
	if expansion.Status != "pending" {
		return nil
	}
	if expansion.PageCount > 200 {
		return r.store.ReviewDrawingExpansion(ctx, orgID, documentID, "This drawing set exceeds the 200-sheet limit. The original is retained.")
	}
	doc, err := r.store.GetDocument(ctx, orgID, documentID)
	if err != nil {
		return err
	}
	file, err := r.store.GetFile(ctx, orgID, doc.FileID)
	if err != nil {
		return err
	}
	path, err := r.blobs.Path(file.SHA256)
	if err != nil {
		return err
	}
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()
	sheets := make([]store.SheetWrite, 0, expansion.PageCount)
	project, err := r.store.ProjectDocuments(ctx, orgID, doc.ProjectID)
	if err != nil {
		return err
	}
	links, err := r.store.OrgSupersessions(ctx, orgID)
	if err != nil {
		return err
	}
	for page := 1; page <= expansion.PageCount; page++ {
		if err := ctx.Err(); err != nil {
			return err
		}
		var pdf bytes.Buffer
		count, err := identity.PDFSheet(ctx, f, file.ByteSize, page, 200, &pdf)
		if err != nil {
			return err
		}
		if count != expansion.PageCount {
			return errors.New("PDF page count changed")
		}
		text, err := identity.Extract(ctx, "pdf", bytes.NewReader(pdf.Bytes()), int64(pdf.Len()), identity.Limits{MaxPages: 1})
		if err != nil {
			return err
		}
		if !text.TextLayer {
			return r.store.ReviewDrawingExpansion(ctx, orgID, documentID, fmt.Sprintf("Page %d has no readable text. The original is retained; sheets have not been published.", page))
		}
		// Do not carry the pack filename's number, revision or title into every
		// child. The source filename remains available through provenance.
		draft := NewDraft(r.svc.catalog, "", text, Harvest("", text), nil)
		draft.Plan(project, documentID)
		if call, ok := draft.Call(); ok {
			call.Priority = jev.PriorityBackground
			result, err := r.svc.jev.Ask(ctx, call)
			if err != nil {
				return err
			}
			draft.Apply(result, nil, r.svc.thresholds)
		}
		kind := draft.by[FieldKind]
		if !store.DrawingSetPageKind(kind.Value, kind.Band) {
			return r.store.ReviewDrawingExpansion(ctx, orgID, documentID, fmt.Sprintf("Page %d was not confirmed as a drawing or schedule within this drawing set. The original is retained as one document.", page))
		}
		draft.fillBlanks()
		blob, err := r.blobs.Put(ctx, bytes.NewReader(pdf.Bytes()))
		if err != nil {
			return err
		}
		sheets = append(sheets, store.SheetWrite{SHA256: blob.SHA256, ByteSize: blob.Size, Decisions: decisionWrites(draft.by, nil), PriorID: draft.Link(links, documentID)})
	}
	return r.store.PublishDrawingSheets(ctx, orgID, documentID, sheets)
}

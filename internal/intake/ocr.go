package intake

import (
	"context"
	"errors"
	"sitewise/internal/identity"
	"sitewise/internal/store"
	"strings"
	"time"
)

// RunOCR is a single background pass, leased by the existing jobs worker.
// Crashes resume from the durable queue; ordinary OCR failures finish with a
// specific explanation rather than looping over an unreadable document.
func (r *Runner) RunOCR(ctx context.Context, orgID, documentID string) error {
	doc, err := r.store.GetDocument(ctx, orgID, documentID)
	if err != nil {
		return err
	}
	missingOnly := doc.Status == store.StatusFiled && store.OCRDetailsActive(doc.Reason)
	if !missingOnly && (doc.Status != store.StatusPending || !strings.HasPrefix(doc.Reason, "ocr_")) {
		return nil
	}
	progress := func(stage string) error {
		if missingOnly {
			return r.store.OCRDetailsProgress(ctx, orgID, documentID, stage)
		}
		return r.store.OCRProgress(ctx, orgID, documentID, "ocr_"+stage)
	}
	fail := func(reason string) error {
		if missingOnly {
			return r.store.OCRDetailsProgress(ctx, orgID, documentID, strings.TrimPrefix(reason, "ocr_"))
		}
		return r.notFiled(ctx, orgID, documentID, reason)
	}
	if err := progress("reading"); err != nil {
		return err
	}
	if r.OCR == nil {
		return fail("ocr_unavailable")
	}
	file, err := r.store.GetFile(ctx, orgID, doc.FileID)
	if err != nil {
		return err
	}
	path, err := r.blobs.Path(file.SHA256)
	if err != nil {
		return err
	}
	started := time.Now()
	text, err := r.OCR(ctx, path)
	r.svc.observe("ocr_extraction", time.Since(started))
	if ctx.Err() != nil {
		return ctx.Err()
	}
	if err != nil {
		reason := "ocr_failed"
		if errors.Is(err, identity.ErrOCRUnavailable) {
			reason = "ocr_unavailable"
		}
		if errors.Is(err, identity.ErrTooLarge) {
			reason = "ocr_limit"
		}
		return fail(reason)
	}
	if len(text.Runs) == 0 {
		return fail("ocr_no_text")
	}
	text.OCR = true
	if err := progress("classifying"); err != nil {
		return err
	}
	_, err = r.svc.file(ctx, orgID, documentID, text, missingOnly)
	if err != nil && missingOnly && ctx.Err() == nil {
		return fail("ocr_failed")
	}
	return err
}

package intake

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"

	"sitewise/internal/files"
	"sitewise/internal/identity"
	"sitewise/internal/store"
)

// Reasons a stored upload is not filed. The bytes are kept and the reason is
// shown to the user.
const (
	ReasonUnsupported = "unsupported_format"
	ReasonNoText      = "no_text_layer"
	ReasonUnreadable  = "unreadable"
	ReasonTooLarge    = "too_large"
)

// ReasonFor is the not_filed reason for a filename intake cannot read, or
// empty when the format is supported.
func ReasonFor(filename string) string {
	if formatOf(filename) == "" {
		return ReasonUnsupported
	}
	return ""
}

func formatOf(filename string) string {
	switch strings.ToLower(filepath.Ext(filename)) {
	case ".pdf":
		return "pdf"
	case ".docx":
		return "docx"
	case ".xlsx":
		return "xlsx"
	default:
		return ""
	}
}

// Runner is the foreground path from stored bytes to a committed filing.
type Runner struct {
	blobs  *files.Store
	store  *store.Store
	svc    *Service
	limits identity.Limits
}

// NewRunner binds the blob store and the filing service.
func NewRunner(blobs *files.Store, st *store.Store, svc *Service) *Runner {
	return &Runner{blobs: blobs, store: st, svc: svc, limits: identity.DefaultLimits()}
}

// Run reads identity text from the stored blob and files the document. A file
// with no readable text is kept and marked not_filed. A document that is no
// longer pending is left as it is, so a repeated run is safe.
func (r *Runner) Run(ctx context.Context, orgID, documentID string) error {
	doc, err := r.store.GetDocument(ctx, orgID, documentID)
	if err != nil {
		return err
	}
	if doc.Status != store.StatusPending {
		return nil
	}
	format := formatOf(doc.Filename)
	if format == "" {
		return r.notFiled(ctx, orgID, documentID, ReasonUnsupported)
	}
	text, err := r.identity(ctx, orgID, doc.FileID, format)
	switch {
	case errors.Is(err, identity.ErrTooLarge):
		return r.notFiled(ctx, orgID, documentID, ReasonTooLarge)
	case errors.Is(err, identity.ErrMalformed):
		return r.notFiled(ctx, orgID, documentID, ReasonUnreadable)
	case err != nil:
		return err
	case !text.TextLayer:
		return r.notFiled(ctx, orgID, documentID, ReasonNoText)
	}
	_, err = r.svc.File(ctx, orgID, documentID, text)
	return err
}

// identity opens the blob by path rather than files.Store.Open: Put hashed
// these bytes moments ago, and re-hashing a large drawing would spend the
// extraction budget on a check that cannot fail.
func (r *Runner) identity(ctx context.Context, orgID, fileID, format string) (identity.Text, error) {
	file, err := r.store.GetFile(ctx, orgID, fileID)
	if err != nil {
		return identity.Text{}, err
	}
	path, err := r.blobs.Path(file.SHA256)
	if err != nil {
		return identity.Text{}, err
	}
	f, err := os.Open(path)
	if err != nil {
		return identity.Text{}, err
	}
	defer f.Close()
	return identity.Extract(ctx, format, f, file.ByteSize, r.limits)
}

func (r *Runner) notFiled(ctx context.Context, orgID, documentID, reason string) error {
	err := r.store.MarkNotFiled(ctx, orgID, documentID, reason)
	if errors.Is(err, store.ErrNotFound) {
		return nil
	}
	return err
}

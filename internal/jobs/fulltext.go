package jobs

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"sitewise/internal/identity"
	"sitewise/internal/store"
)

// fullLimits read a whole document for passages, not just its identity
// pages. Caps bound memory on a pathological file; a 200-page report fits.
var fullLimits = identity.Limits{
	MaxBytes:  512 << 20,
	MaxPages:  200,
	MaxSheets: 20,
	MaxRows:   5000,
	MaxCols:   60,
	MaxRuns:   250000,
}

// BlobPaths resolves a stored blob to a local path.
type BlobPaths interface {
	Path(sum []byte) (string, error)
}

// FullText returns Worker.Text: the document's whole text, with a blank line
// between pages and before headings so passages break at them.
func FullText(st *store.Store, blobs BlobPaths) func(ctx context.Context, orgID, documentID string) (string, error) {
	return func(ctx context.Context, orgID, documentID string) (string, error) {
		doc, err := st.GetDocument(ctx, orgID, documentID)
		if err != nil {
			return "", err
		}
		format := strings.TrimPrefix(strings.ToLower(filepath.Ext(doc.Filename)), ".")
		if format != "pdf" && format != "docx" && format != "xlsx" {
			return "", errors.New("unsupported format for full text")
		}
		file, err := st.GetFile(ctx, orgID, doc.FileID)
		if err != nil {
			return "", err
		}
		path, err := blobs.Path(file.SHA256)
		if err != nil {
			return "", err
		}
		f, err := os.Open(path)
		if err != nil {
			return "", err
		}
		defer f.Close()
		text, err := identity.Extract(ctx, format, f, file.ByteSize, fullLimits)
		if err != nil {
			return "", err
		}
		return joinRuns(text.Runs), nil
	}
}

func joinRuns(runs []identity.Run) string {
	var b strings.Builder
	page := -1
	for _, r := range runs {
		line := strings.TrimSpace(r.Text)
		if line == "" {
			continue
		}
		if b.Len() > 0 {
			if r.Source.Page != page || r.Source.Heading || looksLikeHeading(line) {
				b.WriteString("\n\n")
			} else {
				b.WriteString("\n")
			}
		}
		page = r.Source.Page
		b.WriteString(line)
	}
	return b.String()
}

var headingRe = regexp.MustCompile(`^(\d+(\.\d+)*\.?\s+[A-Za-z]|[A-Z][A-Z0-9 &/,'()-]{3,}$)`)

// looksLikeHeading is a numbered heading ("7.13. Garbage Systems") or a short
// upper-case line ("FIRE SERVICES").
func looksLikeHeading(line string) bool {
	return len(line) <= 80 && headingRe.MatchString(line)
}

// sectionOf is the heading a passage starts with, if any.
func sectionOf(body string) string {
	first, _, _ := strings.Cut(strings.TrimSpace(body), "\n")
	first = strings.TrimSpace(first)
	if looksLikeHeading(first) {
		return first
	}
	return ""
}

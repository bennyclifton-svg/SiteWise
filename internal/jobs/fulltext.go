package jobs

import (
	"context"
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
	source := FullSource(st, blobs)
	return func(ctx context.Context, orgID, documentID string) (string, error) {
		src, err := source(ctx, orgID, documentID)
		if err != nil {
			return "", err
		}
		var parts []string
		for _, p := range src.Source {
			parts = append(parts, p.Text)
		}
		return strings.Join(parts, "\n\n"), nil
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
	return len(line) <= 80 && headingRe.MatchString(line) && !quantityHeading.MatchString(line) && !tableCodeHeading.MatchString(line) && line != "TOTAL"
}

// PDF table cells can resemble numbered or all-caps headings.
var quantityHeading = regexp.MustCompile(`(?i)^\d+(?:\.\d+)?\s*(?:m2|m²|mm|m|sqm|kpa|mpa|a|kn|t)\b`)
var tableCodeHeading = regexp.MustCompile(`^[A-Z]\d+(?:,\s*[A-Z]\d+)*$`)

// sectionOf is the heading a passage starts with, if any.
func sectionOf(body string) string {
	first, _, _ := strings.Cut(strings.TrimSpace(body), "\n")
	first = strings.TrimSpace(first)
	if looksLikeHeading(first) {
		return first
	}
	return ""
}

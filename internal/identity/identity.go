// Package identity reads the text a filing decision is allowed to see.
// It records where that text came from. It does not decide document fields.
package identity

import (
	"archive/zip"
	"context"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"path"
	"strings"
	"unicode/utf8"
)

const maxZipMembers = 64

// ErrMalformed means the container or its XML/PDF structure cannot be read.
var ErrMalformed = errors.New("malformed document")

// ErrTooLarge means a byte, page, or zip-member limit stopped extraction.
var ErrTooLarge = errors.New("document exceeds identity limit")

// Limits bound how much of a file is identity text.
// Zero numeric limits are replaced by DefaultLimits. An unspecified MaxPages
// also enables the default control-page probe.
type Limits struct {
	// MaxBytes caps PDF bytes loaded/read and the sum of decompressed zip members.
	// Larger PDFs use bounded range reads instead of loading the whole file.
	MaxBytes int64
	// MaxPages is how many leading PDF pages are identity pages.
	MaxPages int
	// ReadControlPage permits one additional explicit document-control page
	// after a sparse report cover (or its repeated publisher cover).
	// At most two further pages are probed; drawing sheets never trigger it.
	ReadControlPage bool
	// MaxSheets is how many leading workbook sheets are read.
	MaxSheets int
	// MaxRows caps spreadsheet rows and DOCX table rows.
	MaxRows int
	// MaxCols caps spreadsheet columns.
	MaxCols int
	// MaxRuns caps returned text runs so one dense page cannot dominate the budget.
	MaxRuns int
}

// DefaultLimits is the identity region: the first page or sheet, not the
// whole file. Later full text is a background job.
func DefaultLimits() Limits {
	return Limits{
		MaxBytes:        32 << 20,
		MaxPages:        1,
		ReadControlPage: true,
		MaxSheets:       1,
		MaxRows:         40,
		MaxCols:         16,
		MaxRuns:         400,
	}
}

func (l Limits) norm() Limits {
	d := DefaultLimits()
	if l.MaxBytes <= 0 {
		l.MaxBytes = d.MaxBytes
	}
	if l.MaxPages <= 0 {
		l.MaxPages = d.MaxPages
		l.ReadControlPage = d.ReadControlPage
	}
	if l.MaxSheets <= 0 {
		l.MaxSheets = d.MaxSheets
	}
	if l.MaxRows <= 0 {
		l.MaxRows = d.MaxRows
	}
	if l.MaxCols <= 0 {
		l.MaxCols = d.MaxCols
	}
	if l.MaxRuns <= 0 {
		l.MaxRuns = d.MaxRuns
	}
	return l
}

// Source locates a run. It is provenance, not a filing decision.
type Source struct {
	Annotation string // named CAD text annotation; ordinary comments are excluded
	// PDF page coordinates in points, origin bottom-left. Zero size means
	// geometry is unavailable. They allow captions to bind to their own cell.
	X, Y, Width, Height float64
	Page                int // 1-based PDF page
	Rotation            int // degrees counterclockwise, snapped to a right angle when close
	Sheet               string
	Cell                string // anchor, such as A1
	Merge               string // A1:C1 when the anchor is a merged cell
	Table               int    // 1-based DOCX table; 0 is not a table
	Row                 int    // 1-based table row or sheet row
	Col                 int    // 1-based table column or sheet column
	Heading             bool   // paragraph style is a heading style
	Cached              bool   // spreadsheet value is the stored formula cache, not a calculated result
}

// Run is one piece of identity text.
type Run struct {
	Text   string
	Source Source
}

// Text is the identity text of one file.
// TextLayer is false for a readable PDF page that has no text (a scan or a blank).
type Text struct {
	PageCount int // physical PDF pages; zero for other formats
	Format    string
	TextLayer bool
	Runs      []Run
}

// Extract reads identity text from a PDF, DOCX, or XLSX.
// format is "pdf", "docx", or "xlsx". r is the file bytes of the given size.
func Extract(ctx context.Context, format string, r io.ReaderAt, size int64, limits Limits) (Text, error) {
	if err := ctx.Err(); err != nil {
		return Text{}, err
	}
	limits = limits.norm()
	var (
		got Text
		err error
	)
	switch strings.ToLower(format) {
	case "pdf":
		got, err = extractPDF(ctx, r, size, limits)
	case "docx":
		got, err = extractDOCX(ctx, r, size, limits)
	case "xlsx":
		got, err = extractXLSX(ctx, r, size, limits)
	default:
		return Text{}, fmt.Errorf("%w: format %q", ErrMalformed, format)
	}
	if err != nil {
		return Text{}, err
	}
	got.Format = strings.ToLower(format)
	got.TextLayer = len(got.Runs) > 0
	return got, nil
}

func readAtMost(r io.ReaderAt, size, max int64) ([]byte, error) {
	if size <= 0 {
		return nil, ErrMalformed
	}
	if size > max {
		return nil, ErrTooLarge
	}
	buf := make([]byte, size)
	n, err := r.ReadAt(buf, 0)
	if int64(n) != size {
		if err != nil && err != io.EOF {
			return nil, err
		}
		return nil, ErrMalformed
	}
	return buf, nil
}

func openPackage(r io.ReaderAt, size, maxBytes int64) (*zip.Reader, error) {
	if size <= 0 {
		return nil, ErrMalformed
	}
	if size > maxBytes {
		return nil, ErrTooLarge
	}
	zr, err := zip.NewReader(r, size)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrMalformed, err)
	}
	if len(zr.File) > maxZipMembers {
		return nil, ErrTooLarge
	}
	var declared uint64
	for _, f := range zr.File {
		if err := safeZipName(f.Name); err != nil {
			return nil, err
		}
		if f.UncompressedSize64 > uint64(maxBytes) || declared > uint64(maxBytes)-f.UncompressedSize64 {
			return nil, ErrTooLarge
		}
		declared += f.UncompressedSize64
	}
	return zr, nil
}

func safeZipName(name string) error {
	if name == "" || strings.Contains(name, "\\") || strings.ContainsRune(name, 0) {
		return fmt.Errorf("%w: zip entry", ErrMalformed)
	}
	if path.IsAbs(name) {
		return fmt.Errorf("%w: zip entry", ErrMalformed)
	}
	clean := path.Clean(name)
	if clean == ".." || strings.HasPrefix(clean, "../") {
		return fmt.Errorf("%w: zip entry", ErrMalformed)
	}
	return nil
}

func zipMember(zr *zip.Reader, name string) *zip.File {
	for _, f := range zr.File {
		if f.Name == name {
			return f
		}
	}
	return nil
}

// budgetReader stops a zip member once the remaining decompressed budget is spent
// and checks cancellation between reads.
type budgetReader struct {
	r    io.Reader
	ctx  context.Context
	left int64
	n    int
}

func (b *budgetReader) Read(p []byte) (int, error) {
	b.n++
	if b.n%16 == 0 {
		if err := b.ctx.Err(); err != nil {
			return 0, err
		}
	}
	if b.left <= 0 {
		return 0, ErrTooLarge
	}
	if int64(len(p)) > b.left {
		p = p[:b.left]
	}
	n, err := b.r.Read(p)
	b.left -= int64(n)
	return n, err
}

func openMember(ctx context.Context, f *zip.File, budget int64) (io.ReadCloser, *budgetReader, error) {
	if err := ctx.Err(); err != nil {
		return nil, nil, err
	}
	if f.UncompressedSize64 > uint64(budget) {
		return nil, nil, ErrTooLarge
	}
	rc, err := f.Open()
	if err != nil {
		return nil, nil, fmt.Errorf("%w: %v", ErrMalformed, err)
	}
	br := &budgetReader{r: rc, ctx: ctx, left: budget}
	return rc, br, nil
}

func newXML(r io.Reader) *xml.Decoder {
	dec := xml.NewDecoder(r)
	// Office files are UTF-8. Refuse a charset that would pull in a converter.
	dec.CharsetReader = func(charset string, _ io.Reader) (io.Reader, error) {
		return nil, fmt.Errorf("%w: charset %s", ErrMalformed, charset)
	}
	return dec
}

func trimKept(s string) string {
	if !utf8.ValidString(s) {
		s = strings.ToValidUTF8(s, "")
	}
	return strings.TrimSpace(s)
}

func malformed(err error) error {
	if err == nil {
		return ErrMalformed
	}
	if errors.Is(err, ErrMalformed) || errors.Is(err, ErrTooLarge) || errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return err
	}
	return fmt.Errorf("%w: %v", ErrMalformed, err)
}

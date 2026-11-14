package main

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"strings"

	"sitewise/internal/identity"
	"sitewise/internal/tender"
)

// reader holds what turning a file into pages and page images needs. ocr is
// nil when Tesseract or the Python helper is not set up.
type reader struct{ ocr *identity.OCR }

func newReader(getenv func(string) string) reader {
	exe, py := getenv("SITEWISE_TESSERACT"), getenv("SITEWISE_OCR_PYTHON")
	if exe == "" {
		exe = "tesseract"
	}
	if py == "" {
		py = "python"
	}
	ocr, err := identity.NewOCR(exe, getenv("SITEWISE_TESSDATA"), py)
	if err != nil {
		return reader{}
	}
	return reader{ocr: ocr}
}

// pages extracts a file the way the product does: text layer first, then
// OCR for the pages that have none.
func (r reader) pages(ctx context.Context, path string, b []byte) ([]tender.Page, bool, error) {
	format := strings.TrimPrefix(strings.ToLower(filepath.Ext(path)), ".")
	if format == "markdown" {
		format = "md"
	}
	text, err := identity.Extract(ctx, format, bytes.NewReader(b), int64(len(b)), tender.ExtractLimits())
	if err != nil {
		return nil, false, err
	}
	var missing []int
	if format == "pdf" {
		missing = identity.MissingTextPages(text)
		if len(missing) > 0 && r.ocr != nil {
			if read, err := r.ocr.ReadMissing(ctx, path, text, tender.MaxImagePages); err == nil || len(read.Runs) > len(text.Runs) {
				text = read
			}
		}
	}
	return tender.PagesOf(text, missing), format == "pdf", nil
}

// render is the tender.Renderer over the OCR helper's Python.
func (r reader) render() tender.Renderer {
	if r.ocr == nil {
		return nil
	}
	return func(ctx context.Context, path string, pages []int) (map[int]string, error) {
		images, err := r.ocr.Pages(ctx, path, pages)
		if err != nil {
			return nil, err
		}
		out := map[int]string{}
		for _, img := range images {
			out[img.Page] = img.JPEG
		}
		return out, nil
	}
}

func readFile(path string) ([]byte, error) { return os.ReadFile(path) }

package tender

import (
	"strings"

	"sitewise/internal/identity"
)

// ExtractLimits reads every page of a tender, not just its identity pages.
func ExtractLimits() identity.Limits {
	l := identity.DefaultLimits()
	l.MaxPages = 400
	l.ReadControlPage = false
	l.MaxRuns = 120000
	l.MaxSheets = 12
	l.MaxRows = 2000
	return l
}

// PagesOf turns extracted text into pages. ocr lists the pages whose text
// came from OCR (identity.MissingTextPages before the OCR call).
func PagesOf(text identity.Text, ocr []int) []Page {
	n := text.PageCount
	if n == 0 {
		n = 1
	}
	parts := make([][]string, n+1)
	for _, run := range text.Runs {
		p := run.Source.Page
		if text.PageCount == 0 || p < 1 || p > n {
			p = 1
		}
		parts[p] = append(parts[p], run.Text)
	}
	fromOCR := map[int]bool{}
	for _, p := range ocr {
		fromOCR[p] = true
	}
	pages := make([]Page, 0, n)
	for p := 1; p <= n; p++ {
		pages = append(pages, Page{Number: p, Text: strings.Join(parts[p], "\n"), OCR: fromOCR[p]})
	}
	return pages
}

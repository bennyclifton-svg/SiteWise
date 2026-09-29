package identity

import (
	"context"
	"io"
	"math"
	"sync"

	"github.com/klippa-app/go-pdfium"
	"github.com/klippa-app/go-pdfium/references"
	"github.com/klippa-app/go-pdfium/requests"
	"github.com/klippa-app/go-pdfium/webassembly"
	"github.com/tetratelabs/wazero"
)

// PDFium binding is the embedded WebAssembly build of
// github.com/klippa-app/go-pdfium v1.18.0 (MIT). PDFium and Wazero are Apache-2.0.
// The module is compiled into this process, so intake stays one Go binary
// and does not need CGO or a sidecar.
//
// Measured 2026-09-29 on Windows 10 amd64, AMD Ryzen 5 3600, Go 1.27.1.
// Warm, which compiles the module and opens one worker, took 1.54 s and
// 104 MiB of system memory (24 MiB on the Go heap). Warm extraction of
// testdata/identity/identity-page.pdf over 20 samples: p50 8.0 ms, p90 9.1 ms.
// Five benchmark runs sat near 8.3 ms/op. The gate is 80/250 ms, so the WASM
// packaging stays.
// Call Warm at process start; that cold compile is not part of an intake.
//
// An empty FSConfig is passed so the default init cannot mount the host drive
// into the sandbox. Documents are opened from bytes.

var (
	pdfOnce sync.Once
	pdfPool pdfium.Pool
	pdfInit error
)

// Warm compiles the embedded PDFium module and opens one worker.
// Later Extract calls reuse that worker.
func Warm(ctx context.Context) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	_, err := pdfiumPool()
	return err
}

func pdfiumPool() (pdfium.Pool, error) {
	pdfOnce.Do(func() {
		// Reuse one worker. Compiling the module is the cold cost; a warm
		// extract must not instantiate PDFium again. One worker matches the
		// single-file identity path and caps WASM memory growth.
		pdfPool, pdfInit = webassembly.Init(webassembly.Config{
			MinIdle:      1,
			MaxIdle:      1,
			MaxTotal:     1,
			ReuseWorkers: true,
			FSConfig:     wazero.NewFSConfig(),
			Stdout:       io.Discard,
			Stderr:       io.Discard,
		})
	})
	if pdfInit != nil {
		return nil, pdfInit
	}
	return pdfPool, nil
}

func extractPDF(ctx context.Context, r io.ReaderAt, size int64, limits Limits) (Text, error) {
	body, err := readAtMost(r, size, limits.MaxBytes)
	if err != nil {
		return Text{}, err
	}
	if len(body) < 5 || string(body[:5]) != "%PDF-" {
		return Text{}, ErrMalformed
	}
	if err := ctx.Err(); err != nil {
		return Text{}, err
	}
	pool, err := pdfiumPool()
	if err != nil {
		return Text{}, err
	}
	inst, err := pool.GetInstanceWithContext(ctx)
	if err != nil {
		return Text{}, err
	}
	defer inst.Close()

	doc, err := inst.OpenDocument(&requests.OpenDocument{File: &body})
	if err != nil {
		return Text{}, malformed(err)
	}
	defer inst.FPDF_CloseDocument(&requests.FPDF_CloseDocument{Document: doc.Document})

	count, err := inst.FPDF_GetPageCount(&requests.FPDF_GetPageCount{Document: doc.Document})
	if err != nil {
		return Text{}, malformed(err)
	}
	pages := count.PageCount
	if pages > limits.MaxPages {
		pages = limits.MaxPages
	}
	var runs []Run
	for page := 0; page < pages; page++ {
		if err := ctx.Err(); err != nil {
			return Text{}, err
		}
		pageRuns, err := pdfPageRuns(ctx, inst, doc.Document, page, limits.MaxRuns-len(runs))
		if err != nil {
			return Text{}, err
		}
		runs = append(runs, pageRuns...)
		if len(runs) >= limits.MaxRuns {
			runs = runs[:limits.MaxRuns]
			break
		}
	}
	return Text{Runs: runs}, nil
}

func pdfPageRuns(ctx context.Context, inst pdfium.Pdfium, doc references.FPDF_DOCUMENT, pageIndex, maxRuns int) ([]Run, error) {
	if maxRuns <= 0 {
		return nil, nil
	}
	page := requests.Page{ByIndex: &requests.PageByIndex{Document: doc, Index: pageIndex}}
	loaded, err := inst.FPDFText_LoadPage(&requests.FPDFText_LoadPage{Page: page})
	if err != nil {
		return nil, malformed(err)
	}
	defer inst.FPDFText_ClosePage(&requests.FPDFText_ClosePage{TextPage: loaded.TextPage})

	counted, err := inst.FPDFText_CountChars(&requests.FPDFText_CountChars{TextPage: loaded.TextPage})
	if err != nil {
		return nil, malformed(err)
	}
	if counted.Count <= 0 {
		return nil, nil
	}
	// Rects are lines, not characters. The structured-char helper crosses into
	// WASM once per glyph, which misses the identity budget on a normal page.
	rects, err := inst.FPDFText_CountRects(&requests.FPDFText_CountRects{
		TextPage:   loaded.TextPage,
		StartIndex: 0,
		Count:      counted.Count,
	})
	if err != nil {
		return nil, malformed(err)
	}
	if rects.Count < 0 {
		return nil, ErrMalformed
	}
	runs := make([]Run, 0, rects.Count)
	n := rects.Count
	if n > maxRuns {
		n = maxRuns
	}
	for i := 0; i < n; i++ {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		rect, err := inst.FPDFText_GetRect(&requests.FPDFText_GetRect{
			TextPage: loaded.TextPage,
			Index:    i,
		})
		if err != nil {
			return nil, malformed(err)
		}
		bounded, err := inst.FPDFText_GetBoundedText(&requests.FPDFText_GetBoundedText{
			TextPage: loaded.TextPage,
			Left:     rect.Left,
			Top:      rect.Top,
			Right:    rect.Right,
			Bottom:   rect.Bottom,
		})
		if err != nil {
			return nil, malformed(err)
		}
		text := cleanPDFLine(bounded.Text)
		if text == "" {
			continue
		}
		rotation := rectRotation(inst, loaded.TextPage, rect.Left, rect.Top, rect.Right, rect.Bottom)
		runs = append(runs, Run{
			Text: text,
			Source: Source{
				Page:     pageIndex + 1,
				Rotation: rotation,
			},
		})
	}
	return runs, nil
}

func rectRotation(inst pdfium.Pdfium, textPage references.FPDF_TEXTPAGE, left, top, right, bottom float64) int {
	cx := (left + right) / 2
	cy := (top + bottom) / 2
	tol := math.Min(math.Abs(right-left), math.Abs(top-bottom)) / 2
	if tol < 2 {
		tol = 2
	}
	at, err := inst.FPDFText_GetCharIndexAtPos(&requests.FPDFText_GetCharIndexAtPos{
		TextPage:   textPage,
		X:          cx,
		Y:          cy,
		XTolerance: tol,
		YTolerance: tol,
	})
	if err != nil || at.CharIndex < 0 {
		return 0
	}
	ang, err := inst.FPDFText_GetCharAngle(&requests.FPDFText_GetCharAngle{
		TextPage: textPage,
		Index:    at.CharIndex,
	})
	if err != nil {
		// A missing angle still leaves the text. Rotation stays upright.
		return 0
	}
	return degreesCCW(float64(ang.CharAngle))
}

func degreesCCW(rad float64) int {
	if math.IsNaN(rad) || math.IsInf(rad, 0) {
		return 0
	}
	n := int(math.Round(rad * 180 / math.Pi))
	n %= 360
	if n < 0 {
		n += 360
	}
	for _, axis := range []int{0, 90, 180, 270} {
		d := n - axis
		if d < 0 {
			d = -d
		}
		if d <= 2 {
			return axis
		}
	}
	if n >= 358 {
		return 0
	}
	return n
}

func cleanPDFLine(s string) string {
	// PDFium line breaks are not part of the line's text.
	out := make([]rune, 0, len(s))
	for _, r := range s {
		if r == '\n' || r == '\r' || r == 0 {
			continue
		}
		out = append(out, r)
	}
	return trimKept(string(out))
}

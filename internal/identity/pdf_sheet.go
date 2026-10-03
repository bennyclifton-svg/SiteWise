package identity

import (
	"context"
	"io"

	"github.com/klippa-app/go-pdfium/requests"
)

// PDFSheet copies one physical page into a standalone PDF. It makes no
// judgement about whether the source is a drawing set. Callers must establish
// that separately, and retain the original document and page provenance.
// Page is one-based. Each call releases the PDFium worker before returning,
// so background splitting can yield between sheets to interactive extraction.
func PDFSheet(ctx context.Context, r io.ReaderAt, size int64, page, maxPages int, dst io.Writer) (int, error) {
	if err := ctx.Err(); err != nil {
		return 0, err
	}
	if page < 1 || maxPages < 1 || dst == nil {
		return 0, ErrMalformed
	}
	// Bound total source reads, including page resources copied during save.
	if size <= 0 || size > 512<<20 {
		return 0, ErrTooLarge
	}
	source := &pdfBudgetReader{ctx: ctx, reader: io.NewSectionReader(r, 0, size), remaining: 512 << 20}
	pool, err := pdfiumPool()
	if err != nil {
		return 0, err
	}
	inst, err := pool.GetInstanceWithContext(ctx)
	if err != nil {
		return 0, err
	}
	defer inst.Close()
	doc, err := inst.OpenDocument(&requests.OpenDocument{FileReader: source, FileReaderSize: size})
	if err != nil {
		if source.err != nil {
			return 0, source.err
		}
		return 0, malformed(err)
	}
	defer inst.FPDF_CloseDocument(&requests.FPDF_CloseDocument{Document: doc.Document})
	count, err := inst.FPDF_GetPageCount(&requests.FPDF_GetPageCount{Document: doc.Document})
	if err != nil {
		return 0, malformed(err)
	}
	if count.PageCount > maxPages {
		return count.PageCount, ErrTooLarge
	}
	if page > count.PageCount {
		return count.PageCount, ErrMalformed
	}
	sheet, err := inst.FPDF_CreateNewDocument(&requests.FPDF_CreateNewDocument{})
	if err != nil {
		return count.PageCount, err
	}
	defer inst.FPDF_CloseDocument(&requests.FPDF_CloseDocument{Document: sheet.Document})
	_, err = inst.FPDF_ImportPagesByIndex(&requests.FPDF_ImportPagesByIndex{
		Source: doc.Document, Destination: sheet.Document, PageIndices: []int{page - 1},
	})
	if err != nil {
		return count.PageCount, malformed(err)
	}
	writer := &pdfSheetWriter{ctx: ctx, dst: dst, remaining: 128 << 20}
	_, err = inst.FPDF_SaveAsCopy(&requests.FPDF_SaveAsCopy{
		Document: sheet.Document, Flags: requests.SaveFlagNoIncremental, FileWriter: writer,
	})
	if source.err != nil {
		return count.PageCount, source.err
	}
	if writer.err != nil {
		return count.PageCount, writer.err
	}
	if err != nil {
		return count.PageCount, err
	}
	return count.PageCount, ctx.Err()
}

type pdfSheetWriter struct {
	ctx       context.Context
	dst       io.Writer
	remaining int
	err       error
}

func (w *pdfSheetWriter) Write(p []byte) (int, error) {
	if w.err == nil {
		w.err = w.ctx.Err()
	}
	if w.err == nil && len(p) > w.remaining {
		w.err = ErrTooLarge
	}
	if w.err != nil {
		return 0, w.err
	}
	n, err := w.dst.Write(p)
	w.remaining -= n
	if err == nil && n != len(p) {
		err = io.ErrShortWrite
	}
	w.err = err
	return n, err
}

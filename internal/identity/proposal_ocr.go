package identity

import (
	"context"
	_ "embed"
	"encoding/json"
	"errors"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
)

//go:embed proposal_ocr.py
var proposalOCRScript string

// MissingTextPages includes image pages and pages with only a native footer.
// A native page elsewhere in the PDF is not evidence that its fee appendix was read.
func MissingTextPages(text Text) []int {
	counts := map[int]int{}
	for _, run := range text.Runs {
		counts[run.Source.Page] += len(strings.TrimSpace(run.Text))
	}
	var pages []int
	for p := 1; p <= text.PageCount; p++ {
		if counts[p] < 40 {
			pages = append(pages, p)
		}
	}
	return pages
}

type proposalOCRPage struct {
	Page          int
	Width, Height float64
	TSV           string
}

// ReadMissingPages reuses Tesseract outside filing, at most four image pages.
// Cache only successful extraction beside the content-addressed source. No model
// is involved and no PDF is rewritten. Limits leave explicit incomplete coverage.
func (o *OCR) ReadMissingPages(ctx context.Context, path string, native Text) (Text, error) {
	return o.ReadMissing(ctx, path, native, ocrBatch)
}

// ocrBatch is how many pages one run of the helper reads: its output is
// capped, so a long scan is read a batch at a time.
const ocrBatch = 4

// ReadMissing is ReadMissingPages with its own page limit. The tender pass
// runs in the background and checks every amount against page text, so it
// reads up to the pages it may send, where a proposal read stops at four.
// A batch that fails leaves its pages without text and marks the result
// incomplete; the batches before it are kept.
func (o *OCR) ReadMissing(ctx context.Context, path string, native Text, limit int) (Text, error) {
	pages := MissingTextPages(native)
	if len(pages) == 0 {
		return native, nil
	}
	native.Incomplete = len(pages) > limit
	pages = pages[:min(len(pages), limit)]
	var runs []Run
	read := map[int]bool{}
	var failed error
	for start := 0; start < len(pages); start += ocrBatch {
		batch := pages[start:min(start+ocrBatch, len(pages))]
		// The first batch keeps the cache name a four-page read always had.
		cache := path + ".proposal-ocr-v2-" + strconv.Itoa(len(batch)) + ".json"
		if start > 0 {
			cache = path + ".proposal-ocr-v2-p" + strconv.Itoa(batch[0]) + "-" + strconv.Itoa(len(batch)) + ".json"
		}
		got, err := o.readBatch(ctx, path, batch, cache)
		if err != nil {
			failed = err
			native.Incomplete = true
			break
		}
		runs = append(runs, got...)
		for _, p := range batch {
			read[p] = true
		}
	}
	if len(read) == 0 {
		return native, failed
	}
	for _, run := range native.Runs {
		if !read[run.Source.Page] {
			runs = append(runs, run)
		}
	}
	native.Runs = runs
	native.OCR = true
	return native, failed
}

// readBatch reads a few pages through the helper, or from its cache.
func (o *OCR) readBatch(ctx context.Context, path string, pages []int, cache string) ([]Run, error) {
	requested, _ := json.Marshal(pages)
	var data []byte
	file, err := os.Open(cache)
	cached := err == nil
	if cached {
		data, err = io.ReadAll(io.LimitReader(file, (2<<20)+1))
		file.Close()
		cached = err == nil && len(data) <= 2<<20
	}
	if !cached {
		cmd := exec.CommandContext(ctx, o.python, "-c", proposalOCRScript, path, o.executable, o.tessdata, string(requested))
		var output limitedOCRBuffer
		cmd.Stdout = &output
		if err = cmd.Run(); err != nil {
			return nil, err
		}
		data = output.Bytes()
	}
	var recovered []proposalOCRPage
	if len(data) > 4<<20 || json.Unmarshal(data, &recovered) != nil || len(recovered) != len(pages) {
		return nil, errors.New("invalid proposal OCR output")
	}
	var runs []Run
	for i, p := range recovered {
		if p.Page != pages[i] {
			return nil, errors.New("invalid proposal OCR page")
		}
		got, err := parseOCRPage([]byte(p.TSV), p.Width, p.Height, 200.0/72)
		if err != nil {
			return nil, err
		}
		for j := range got {
			got[j].Source.Page = p.Page
		}
		runs = append(runs, got...)
	}
	if !cached {
		// A stopped worker must not leave a truncated cache for the next read.
		if tmp, err := os.CreateTemp(filepath.Dir(cache), ".proposal-ocr-*"); err == nil {
			name := tmp.Name()
			_, writeErr := tmp.Write(data)
			closeErr := tmp.Close()
			if writeErr == nil && closeErr == nil {
				_ = os.Rename(name, cache)
			}
			_ = os.Remove(name)
		}
	}
	return runs, nil
}

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
	pages := MissingTextPages(native)
	if len(pages) == 0 {
		return native, nil
	}
	native.Incomplete = len(pages) > 4
	pages = pages[:min(len(pages), 4)]
	requested, _ := json.Marshal(pages)
	cache := path + ".proposal-ocr-v2-" + strconv.Itoa(len(pages)) + ".json"
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
			native.Incomplete = true
			return native, err
		}
		data = output.Bytes()
	}
	var recovered []proposalOCRPage
	if len(data) > 4<<20 || json.Unmarshal(data, &recovered) != nil || len(recovered) != len(pages) {
		native.Incomplete = true
		return native, errors.New("invalid proposal OCR output")
	}
	var runs []Run
	read := map[int]bool{}
	for i, p := range recovered {
		if p.Page != pages[i] {
			native.Incomplete = true
			return native, errors.New("invalid proposal OCR page")
		}
		got, err := parseOCRPage([]byte(p.TSV), p.Width, p.Height, 200.0/72)
		if err != nil {
			native.Incomplete = true
			return native, err
		}
		for j := range got {
			got[j].Source.Page = p.Page
		}
		runs = append(runs, got...)
		read[p.Page] = true
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
	for _, run := range native.Runs {
		if !read[run.Source.Page] {
			runs = append(runs, run)
		}
	}
	native.Runs = runs
	native.OCR = true
	return native, nil
}

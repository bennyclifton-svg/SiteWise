package identity

import (
	"bytes"
	"context"
	_ "embed"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

var ErrOCRUnavailable = errors.New("OCR is unavailable")

//go:embed ocr_render.py
var ocrRenderer string

// OCR uses native PDFium outside the text-extraction worker. The existing
// WASM renderer took 29 seconds on the Bankstown sheet; native rendering
// avoids holding normal uploads behind that work.
type OCR struct{ executable, tessdata, python string }

func NewOCR(executable, tessdata, python string) (*OCR, error) {
	path, err := exec.LookPath(executable)
	if err != nil {
		return nil, fmt.Errorf("%w: tesseract executable", ErrOCRUnavailable)
	}
	py, err := exec.LookPath(python)
	if err != nil {
		return nil, fmt.Errorf("%w: Python executable", ErrOCRUnavailable)
	}
	if tessdata != "" {
		if _, err := os.Stat(filepath.Join(tessdata, "eng.traineddata")); err != nil {
			return nil, fmt.Errorf("%w: English language data", ErrOCRUnavailable)
		}
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := exec.CommandContext(ctx, py, "-c", "import pypdfium2, PIL").Run(); err != nil {
		return nil, fmt.Errorf("%w: Python PDFium/Pillow packages", ErrOCRUnavailable)
	}
	return &OCR{executable: path, tessdata: tessdata, python: py}, nil
}

// Extract reads the first identity page only, never the entire pack. Both
// subprocesses share a deadline and temporary files are removed on all exits.
func (o *OCR) Extract(ctx context.Context, path string) (Text, error) {
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	info, err := os.Stat(path)
	if err != nil {
		return Text{}, err
	}
	if info.Size() > 32<<20 {
		return Text{}, ErrTooLarge
	}
	dir, err := os.MkdirTemp("", "sitewise-ocr-")
	if err != nil {
		return Text{}, err
	}
	defer os.RemoveAll(dir)
	bitmap := filepath.Join(dir, "page.png")
	render := exec.CommandContext(ctx, o.python, "-c", ocrRenderer, path, bitmap)
	var meta limitedOCRBuffer
	render.Stdout = &meta
	if err := render.Run(); err != nil {
		if ctx.Err() != nil {
			return Text{}, ctx.Err()
		}
		var exit *exec.ExitError
		if errors.As(err, &exit) && exit.ExitCode() == 3 {
			return Text{}, ErrTooLarge
		}
		return Text{}, fmt.Errorf("OCR rendering failed: %w", err)
	}
	var size struct {
		Width, Height, CanvasWidth, CanvasHeight float64
		Pages                                    int
		Regions                                  []struct{ X, Y, Width, Height, Top, PixelHeight float64 }
	}
	if err := json.Unmarshal(meta.Bytes(), &size); err != nil {
		return Text{}, err
	}
	if size.Width <= 0 || size.Height <= 0 || size.Pages < 1 {
		return Text{}, ErrMalformed
	}
	args := []string{bitmap, "stdout", "-l", "eng", "--oem", "1", "--psm", "11", "--dpi", "150"}
	if o.tessdata != "" {
		args = append(args, "--tessdata-dir", o.tessdata)
	}
	args = append(args, "-c", "tessedit_create_tsv=1", "-c", "tessedit_create_txt=0")
	cmd := exec.CommandContext(ctx, o.executable, args...)
	cmd.Env = append(os.Environ(), "OMP_THREAD_LIMIT=1")
	var output limitedOCRBuffer
	cmd.Stdout = &output
	if err := cmd.Run(); err != nil {
		if ctx.Err() != nil {
			return Text{}, ctx.Err()
		}
		return Text{}, fmt.Errorf("OCR process failed: %w", err)
	}
	const scale = 150.0 / 72
	runs, err := parseOCR(output.Bytes(), size.CanvasWidth, size.CanvasHeight, scale)
	if err != nil {
		return Text{}, err
	}
	mapped := runs[:0]
	for _, run := range runs {
		top := (size.CanvasHeight - run.Source.Y - run.Source.Height) * scale
		for _, region := range size.Regions {
			if top >= region.Top-1 && top+run.Source.Height*scale <= region.Top+region.PixelHeight+1 {
				run.Source.X += region.X - 20/scale
				run.Source.Y = region.Y + region.Height - (top-region.Top)/scale - run.Source.Height
				mapped = append(mapped, run)
				break
			}
		}
	}
	runs = mapped
	return Text{Format: "pdf", OCR: true, PageCount: size.Pages, Runs: runs}, err
}

// Bound subprocess output as well as input dimensions. OCR input is untrusted.
type limitedOCRBuffer struct{ bytes.Buffer }

func (b *limitedOCRBuffer) Write(p []byte) (int, error) {
	if b.Len()+len(p) > 2<<20 {
		return 0, ErrTooLarge
	}
	return b.Buffer.Write(p)
}

func parseOCR(data []byte, width, height, scale float64) ([]Run, error) {
	if !bytes.HasPrefix(data, []byte("level\tpage_num\t")) {
		return nil, errors.New("invalid OCR TSV header")
	}
	var runs []Run
	lastLine := ""
	for _, lineData := range strings.Split(strings.TrimSpace(string(data)), "\n")[1:] {
		row := strings.SplitN(strings.TrimSuffix(lineData, "\r"), "\t", 12)
		if len(row) != 12 {
			return nil, errors.New("invalid OCR TSV")
		}
		var err error
		if row[0] != "5" || strings.TrimSpace(row[11]) == "" {
			continue
		}
		v := make([]float64, 4)
		for i := range v {
			v[i], err = strconv.ParseFloat(row[6+i], 64)
			if err != nil || v[i] < 0 || math.IsNaN(v[i]) || math.IsInf(v[i], 0) {
				return nil, errors.New("invalid OCR coordinates")
			}
		}
		x, y, w, h := v[0]/scale, height-(v[1]+v[3])/scale, v[2]/scale, v[3]/scale
		if x+w > width+1 || y < -1 || h > height+1 {
			return nil, errors.New("OCR coordinates outside page")
		}
		line := strings.Join(row[1:5], "/")
		// Split wide gaps into cells, preserving captions and their geometry.
		if len(runs) > 0 && line == lastLine {
			prev := &runs[len(runs)-1]
			if x-prev.Source.X-prev.Source.Width < math.Max(h, prev.Source.Height)*1.5 {
				prev.Text += " " + row[11]
				top := math.Max(prev.Source.Y+prev.Source.Height, y+h)
				prev.Source.Width = x + w - prev.Source.X
				prev.Source.Y = math.Min(prev.Source.Y, y)
				prev.Source.Height = top - prev.Source.Y
				continue
			}
		}
		if len(runs) >= 10000 {
			return nil, ErrTooLarge
		}
		runs = append(runs, Run{Text: row[11], Source: Source{Page: 1, X: x, Y: y, Width: w, Height: h}})
		lastLine = line
	}
	if len(runs) > 400 {
		runs = append(runs[:200:200], runs[len(runs)-200:]...)
	}
	return runs, nil
}

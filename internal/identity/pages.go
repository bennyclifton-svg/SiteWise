package identity

import (
	"bufio"
	"bytes"
	"context"
	_ "embed"
	"encoding/json"
	"errors"
	"os/exec"
	"strconv"
	"strings"
)

//go:embed pages_render.py
var pagesRenderer string

// PageImage is one whole page as a JPEG.
type PageImage struct {
	Page int    `json:"page"`
	JPEG string `json:"jpeg"` // base64
}

const (
	pageLongEdge  = 1568
	pageQuality   = 80
	pageOutputMax = 48 << 20
)

type cappedBuffer struct {
	bytes.Buffer
	max int
}

func (b *cappedBuffer) Write(p []byte) (int, error) {
	if b.Len()+len(p) > b.max {
		return 0, errors.New("page image output limit")
	}
	return b.Buffer.Write(p)
}

// Pages renders the listed pages (1-based) of a PDF. A page that does not
// exist is left out. It makes no judgement about what the pages hold.
func (o *OCR) Pages(ctx context.Context, path string, pages []int) ([]PageImage, error) {
	if o == nil {
		return nil, ErrOCRUnavailable
	}
	if len(pages) == 0 {
		return nil, nil
	}
	list := make([]string, len(pages))
	for i, p := range pages {
		list[i] = strconv.Itoa(p)
	}
	cmd := exec.CommandContext(ctx, o.python, "-c", pagesRenderer, path, strconv.Itoa(pageLongEdge), strconv.Itoa(pageQuality), strings.Join(list, ","))
	buf := &cappedBuffer{max: pageOutputMax}
	cmd.Stdout = buf
	if err := cmd.Run(); err != nil {
		return nil, errors.New("page rendering unavailable")
	}
	var out []PageImage
	sc := bufio.NewScanner(&buf.Buffer)
	sc.Buffer(make([]byte, 0, 1<<20), 8<<20)
	for sc.Scan() {
		var img PageImage
		if err := json.Unmarshal(sc.Bytes(), &img); err != nil {
			return nil, ErrMalformed
		}
		out = append(out, img)
	}
	return out, sc.Err()
}

package identity

import (
	"bytes"
	"context"
	_ "embed"
	"encoding/json"
	"errors"
	"os/exec"
	"strconv"
)

//go:embed visual_render.py
var visualRenderer string

type VisualImage struct {
	Label string `json:"label"`
	Page  int    `json:"page"`
	Data  string `json:"data"`
}
type VisualLine struct {
	Text string `json:"text"`
	Page int    `json:"page"`
}
type VisualEvidence struct {
	Images  []VisualImage `json:"images"`
	Lines   []VisualLine  `json:"lines"`
	Pages   int           `json:"pages"`
	Drawing bool          `json:"drawing"`
}
type visualBuffer struct{ bytes.Buffer }

func (b *visualBuffer) Write(p []byte) (int, error) {
	if b.Len()+len(p) > 4<<20 {
		return 0, errors.New("visual output limit")
	}
	return b.Buffer.Write(p)
}

func (o *OCR) Visual(ctx context.Context, path string, drawing, needOCR bool) (VisualEvidence, error) {
	if o == nil {
		return VisualEvidence{}, ErrOCRUnavailable
	}
	cmd := exec.CommandContext(ctx, o.python, "-c", visualRenderer, path, strconv.FormatBool(drawing), strconv.FormatBool(needOCR), o.executable, o.tessdata)
	var buf visualBuffer
	cmd.Stdout = &buf
	if err := cmd.Run(); err != nil {
		return VisualEvidence{}, errors.New("visual rendering unavailable")
	}
	var out VisualEvidence
	if err := json.Unmarshal(buf.Bytes(), &out); err != nil {
		return out, err
	}
	if len(out.Images) == 0 || len(out.Images) > 3 {
		return out, ErrMalformed
	}
	return out, nil
}

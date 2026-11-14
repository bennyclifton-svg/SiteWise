package identity

import (
	"bytes"
	"context"
	"encoding/base64"
	"image/jpeg"
	"os"
	"path/filepath"
	"testing"

	"github.com/signintech/gopdf"
)

func TestPagesRendersJPEGs(t *testing.T) {
	python := os.Getenv("SITEWISE_OCR_PYTHON")
	if python == "" {
		t.Skip("set SITEWISE_OCR_PYTHON to render pages")
	}
	pdf := gopdf.GoPdf{}
	pdf.Start(gopdf.Config{PageSize: *gopdf.PageSizeA4})
	pdf.AddPage()
	pdf.AddPage()
	path := filepath.Join(t.TempDir(), "two.pdf")
	if err := pdf.WritePdf(path); err != nil {
		t.Fatal(err)
	}
	o := &OCR{python: python}
	got, err := o.Pages(context.Background(), path, []int{1, 2, 99})
	if err != nil || len(got) != 2 || got[0].Page != 1 || got[1].Page != 2 {
		t.Fatalf("%d pages, err %v", len(got), err)
	}
	for _, img := range got {
		data, err := base64.StdEncoding.DecodeString(img.JPEG)
		if err != nil {
			t.Fatal(err)
		}
		cfg, err := jpeg.DecodeConfig(bytes.NewReader(data))
		if err != nil || max(cfg.Width, cfg.Height) < 1567 || max(cfg.Width, cfg.Height) > 1569 {
			t.Fatalf("long edge %d, err %v", max(cfg.Width, cfg.Height), err)
		}
	}
}

func TestPagesWithoutOCRIsUnavailable(t *testing.T) {
	var o *OCR
	if _, err := o.Pages(context.Background(), "x.pdf", []int{1}); err != ErrOCRUnavailable {
		t.Fatalf("err %v", err)
	}
}

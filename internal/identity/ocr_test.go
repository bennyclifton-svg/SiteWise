package identity

import (
	"context"
	"os"
	"testing"
	"time"
)

func TestOCRTSVGeometryAndBounds(t *testing.T) {
	data := []byte("level\tpage_num\tblock_num\tpar_num\tline_num\tword_num\tleft\ttop\twidth\theight\tconf\ttext\n5\t1\t1\t1\t1\t1\t10\t20\t25\t10\t95\tLEVEL\n5\t1\t1\t1\t1\t2\t40\t20\t10\t10\t96\t2\n5\t1\t1\t1\t1\t3\t100\t20\t10\t10\t96\tF\n")
	runs, err := parseOCR(data, 200, 100, 1)
	if err != nil || len(runs) != 2 || runs[0].Text != "LEVEL 2" || runs[0].Source.Y != 70 || runs[1].Source.X != 100 {
		t.Fatalf("runs=%+v err=%v", runs, err)
	}
	if _, err := parseOCR(data, 20, 20, 1); err == nil {
		t.Fatal("accepted out-of-page OCR")
	}
	if _, err := NewOCR("missing-sitewise-tesseract-executable", "", "python"); err == nil {
		t.Fatal("accepted missing binary")
	}
}

func TestOCRInstalled(t *testing.T) {
	path := os.Getenv("SITEWISE_OCR_TEST_PDF")
	if path == "" {
		t.Skip("set SITEWISE_OCR_TEST_PDF for installed OCR smoke test")
	}
	o, err := NewOCR(os.Getenv("SITEWISE_TESSERACT"), os.Getenv("SITEWISE_TESSDATA"), os.Getenv("SITEWISE_OCR_PYTHON"))
	if err != nil {
		t.Fatal(err)
	}
	start := time.Now()
	text, err := o.Extract(context.Background(), path)
	if err != nil {
		t.Fatal(err)
	}
	if !text.OCR || text.TextLayer || len(text.Runs) == 0 {
		t.Fatalf("bad OCR result %+v", text)
	}
	t.Logf("OCR %s, %d runs", time.Since(start), len(text.Runs))
}

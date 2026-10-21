package identity

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"
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

func TestMissingTextPagesFindsMixedFeeAppendix(t *testing.T) {
	text := Text{PageCount: 3, Runs: []Run{{Text: strings.Repeat("Native proposal text ", 5), Source: Source{Page: 1}}, {Text: "Page 2", Source: Source{Page: 2}}}}
	if got := MissingTextPages(text); !reflect.DeepEqual(got, []int{2, 3}) {
		t.Fatalf("missing pages %v", got)
	}
}

func TestProposalOCRRetainsTheMiddleOfDensePages(t *testing.T) {
	var data strings.Builder
	data.WriteString("level\tpage_num\tblock_num\tpar_num\tline_num\tword_num\tleft\ttop\twidth\theight\tconf\ttext\n")
	for n := 1; n <= 600; n++ {
		word := "text"
		if n == 300 {
			word = "$32,000"
		}
		fmt.Fprintf(&data, "5\t1\t1\t1\t%d\t1\t10\t%d\t40\t1\t95\t%s\n", n, n, word)
	}
	runs, err := parseOCRPage([]byte(data.String()), 100, 800, 1)
	if err != nil || len(runs) != 600 || runs[299].Text != "$32,000" {
		t.Fatalf("lost middle-page fee: %d runs, %v", len(runs), err)
	}
}

func TestProposalOCRCacheMergesSourcePages(t *testing.T) {
	path := filepath.Join(t.TempDir(), "source")
	tsv := "level\tpage_num\tblock_num\tpar_num\tline_num\tword_num\tleft\ttop\twidth\theight\tconf\ttext\n5\t1\t1\t1\t1\t1\t10\t20\t40\t10\t95\t$32,000\n"
	data, _ := json.Marshal([]proposalOCRPage{{Page: 2, Width: 100, Height: 100, TSV: tsv}})
	if err := os.WriteFile(path+".proposal-ocr-v2-1.json", data, 0600); err != nil {
		t.Fatal(err)
	}
	native := Text{PageCount: 2, TextLayer: true, Runs: []Run{{Text: strings.Repeat("Native text ", 5), Source: Source{Page: 1}}}}
	got, err := (&OCR{}).ReadMissingPages(context.Background(), path, native)
	if err != nil || !got.OCR || got.Incomplete || !got.TextLayer || len(got.Runs) != 2 || got.Runs[0].Source.Page != 2 || got.Runs[1].Source.Page != 1 {
		t.Fatalf("lost native or physical page evidence: %+v %v", got, err)
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

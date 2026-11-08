// Command pdfpng renders each page of a PDF to a PNG, to look at an
// export the way a reader would. It uses the PDFium build the server
// already embeds (internal/identity).
//
//	go run ./tools/pdfpng in.pdf outdir [dpi]
package main

import (
	"fmt"
	"image/png"
	"io"
	"os"
	"path/filepath"
	"strconv"

	"github.com/klippa-app/go-pdfium/requests"
	"github.com/klippa-app/go-pdfium/webassembly"
	"github.com/tetratelabs/wazero"
)

func main() {
	if len(os.Args) < 3 {
		fmt.Fprintln(os.Stderr, "usage: pdfpng in.pdf outdir [dpi]")
		os.Exit(2)
	}
	dpi := 110
	if len(os.Args) > 3 {
		dpi, _ = strconv.Atoi(os.Args[3])
	}
	if err := run(os.Args[1], os.Args[2], dpi); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run(in, dir string, dpi int) error {
	data, err := os.ReadFile(in)
	if err != nil {
		return err
	}
	pool, err := webassembly.Init(webassembly.Config{MinIdle: 1, MaxIdle: 1, MaxTotal: 1, FSConfig: wazero.NewFSConfig(), Stdout: io.Discard, Stderr: io.Discard})
	if err != nil {
		return err
	}
	defer pool.Close()
	inst, err := pool.GetInstance(0)
	if err != nil {
		return err
	}
	doc, err := inst.OpenDocument(&requests.OpenDocument{File: &data})
	if err != nil {
		return err
	}
	count, err := inst.FPDF_GetPageCount(&requests.FPDF_GetPageCount{Document: doc.Document})
	if err != nil {
		return err
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	for i := 0; i < count.PageCount; i++ {
		page, err := inst.RenderPageInDPI(&requests.RenderPageInDPI{Page: requests.Page{ByIndex: &requests.PageByIndex{Document: doc.Document, Index: i}}, DPI: dpi})
		if err != nil {
			return err
		}
		name := filepath.Join(dir, fmt.Sprintf("page-%d.png", i+1))
		f, err := os.Create(name)
		if err != nil {
			return err
		}
		err = png.Encode(f, page.Result.Image)
		page.Cleanup()
		f.Close()
		if err != nil {
			return err
		}
		fmt.Println(name)
	}
	return nil
}

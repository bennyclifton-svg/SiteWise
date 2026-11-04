package invoices

import (
	"bytes"
	"context"
	"os"
	"sitewise/internal/identity"
	"sitewise/internal/jev"
	"testing"
	"time"
)

func TestMarkdownInvoice(t *testing.T) {
	raw, err := os.ReadFile("testdata/quoin.md")
	if err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"quoin.md", "quoin.MARKDOWN"} {
		t.Run(name, func(t *testing.T) {
			text, err := identity.Extract(context.Background(), FormatOf(name), bytes.NewReader(raw), int64(len(raw)), ExtractLimits())
			if err != nil {
				t.Fatal(err)
			}
			ask := &fakeAsk{res: jev.Result{Answers: map[string]jev.Answer{
				FieldSupplier: {Type: jev.TypeChoice, Choice: "p1", Confidence: conf(.99)},
				FieldABN:      {Type: jev.TypeChoice, Choice: "c1", Confidence: conf(.99)},
			}}}
			pc := Context{Packages: []Package{{ID: "arch", Title: "Architect", Kind: "services"}}, Lines: []Line{{ID: "line", Label: "Architect", PackageID: "arch"}}}
			out, err := Read(context.Background(), ask, calibrated(FieldSupplier, FieldABN), name, text, pc, nil, time.Now(), "")
			if err != nil {
				t.Fatal(err)
			}
			if out.NotRead != "" || out.Number != "QUA-2601" || out.Date != "2026-03-18" || out.Kind != "invoice" || out.Ex == nil || *out.Ex != 2400000 || out.GST == nil || *out.GST != 240000 || out.Total == nil || *out.Total != 2640000 || out.PackageID != "arch" || out.LineID != "line" {
				t.Fatalf("reading: %+v", out)
			}
		})
	}
}

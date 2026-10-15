package sample

import (
	"bytes"
	"context"
	"testing"
	"time"

	"sitewise/internal/identity"
	"sitewise/internal/invoices"
)

// The synthetic layout must read the way the rules expect, or every test
// built on it measures nothing.
func TestSampleReadsByRule(t *testing.T) {
	for _, inv := range Batch(1, 6) {
		if !invoices.ValidABN(inv.ABN) {
			t.Fatalf("invalid ABN %s", inv.ABN)
		}
		body := PDF(inv)
		text, err := identity.Extract(context.Background(), "pdf", bytes.NewReader(body), int64(len(body)), invoices.ExtractLimits())
		if err != nil {
			t.Fatal(err)
		}
		r := invoices.NewDraft(invoices.Harvest(inv.Number+".pdf", text), invoices.Context{}, nil).Finish(time.Now())
		if r.Number != inv.Number || r.Kind != "invoice" || r.Ex == nil || *r.Ex != inv.Ex || *r.GST != inv.GST || r.Date == "" {
			t.Fatalf("%+v from %+v", r, inv)
		}
	}
}

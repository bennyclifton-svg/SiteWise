package httpapi_test

import (
	"context"
	"strings"
	"sync"
	"testing"

	"sitewise/internal/httpapi"
	"sitewise/internal/jev"
	"sitewise/internal/proposals"
	psample "sitewise/internal/proposals/sample"
	tsample "sitewise/internal/tender/sample"
)

// recordingJev notes every field the old reader asks about.
type recordingJev struct {
	mu     sync.Mutex
	fields []string
}

func (r *recordingJev) Ask(ctx context.Context, call jev.Call) (jev.Result, error) {
	r.mu.Lock()
	for f := range call.Questions {
		r.fields = append(r.fields, f)
	}
	r.mu.Unlock()
	return psample.Jev{}.Ask(ctx, call)
}

func (r *recordingJev) asked() (prices, others int) {
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, f := range r.fields {
		if f == proposals.FieldStatedTotal || strings.HasPrefix(f, "price:stage:") || strings.HasPrefix(f, "price:option:") {
			prices++
		} else {
			others++
		}
	}
	return prices, others
}

// With the server setting on and the org's switch on, Jev is not asked the
// prices the tender pass reads, and the table still carries them. With the
// org's switch off the old reading is unchanged. The firm is invented.
func TestSkipJevPricesWhenTheTenderPassReads(t *testing.T) {
	rec := &recordingJev{}
	_, m, _, base, labels := tenderSetup(t, withTender(tsample.Model{}, tinyRender), func(o *httpapi.Options) {
		o.ProposalJev = rec
		o.TenderSkipJevPrices = true
	})

	// Switch off: the old reader asks its price questions as before.
	before := m.dropTender(t, base, "Invented Planning Pty Ltd", labels)
	m.waitTender(t, base, before, "")
	if prices, _ := rec.asked(); prices == 0 {
		t.Fatal("with the switch off the old reader asked no price")
	}

	rec.mu.Lock()
	rec.fields = nil
	rec.mu.Unlock()
	m.modelSwitch(t, true, 200)
	sub := m.dropTender(t, base, "Invented Surveying Pty Ltd", labels)
	table := m.waitTender(t, base, sub, "read")
	prices, others := rec.asked()
	if prices != 0 || others == 0 {
		t.Fatalf("Jev was asked %d price questions and %d others", prices, others)
	}
	c := table.column(sub)
	if c == nil || money(t, c.TenderSum) != float64(len(labels))*1000 {
		t.Fatalf("the tender column %+v", c)
	}
}

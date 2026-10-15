package main

import (
	"bufio"
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"

	"sitewise/internal/eval"
	"sitewise/internal/invoices"
	"sitewise/internal/invoices/sample"
	"sitewise/internal/jev"
	"sitewise/internal/store"
)

// invoiceModel stands in for System One on invoice questions in a replayed
// bench: the recordings hold filing calls, not these synthetic invoices. It
// answers nothing (every question stays blank) and takes the time a recorded
// call took, in turn, so the latency model matches filing's.
type invoiceModel struct {
	mu  sync.Mutex
	lat []time.Duration
	i   int
}

func newInvoiceModel(records []eval.Record) *invoiceModel {
	m := &invoiceModel{}
	for _, r := range records {
		if r.Error == "" && r.LatencyUS > 0 {
			m.lat = append(m.lat, time.Duration(r.LatencyUS)*time.Microsecond)
		}
	}
	if len(m.lat) == 0 {
		m.lat = []time.Duration{300 * time.Millisecond}
	}
	return m
}

func (m *invoiceModel) Ask(ctx context.Context, _ jev.Call) (jev.Result, error) {
	m.mu.Lock()
	d := m.lat[m.i%len(m.lat)]
	m.i++
	m.mu.Unlock()
	select {
	case <-time.After(d):
		return jev.Result{Answers: map[string]jev.Answer{}}, nil
	case <-ctx.Done():
		return jev.Result{}, ctx.Err()
	}
}

// invoiceWatch is the client's view of the event stream: when each invoice
// was first seen read.
type invoiceWatch struct {
	mu   sync.Mutex
	read map[string]chan struct{}
}

func (w *invoiceWatch) wait(id string) chan struct{} {
	w.mu.Lock()
	defer w.mu.Unlock()
	ch, ok := w.read[id]
	if !ok {
		ch = make(chan struct{})
		w.read[id] = ch
	}
	return ch
}

func (w *invoiceWatch) seen(id, status string) {
	if status == "" || status == invoices.StatusReading {
		return
	}
	ch := w.wait(id)
	w.mu.Lock()
	defer w.mu.Unlock()
	select {
	case <-ch:
	default:
		close(ch)
	}
}

func (a *apiClient) invoiceStream(ctx context.Context, project string, w *invoiceWatch) (func(), error) {
	// Start from the org's current cursor: only this run's events matter.
	resp, err := a.do(ctx, http.MethodGet, "/projects/"+project+"/documents", nil, 0)
	if err != nil {
		return nil, err
	}
	var list struct {
		Cursor int64 `json:"cursor"`
	}
	err = json.NewDecoder(resp.Body).Decode(&list)
	resp.Body.Close()
	if err != nil {
		return nil, err
	}
	ctx, cancel := context.WithCancel(ctx)
	resp, err = a.do(ctx, http.MethodGet, "/events?after="+strconv.FormatInt(list.Cursor, 10), nil, 0)
	if err != nil || resp.StatusCode != http.StatusOK {
		cancel()
		if err == nil {
			resp.Body.Close()
			err = fmt.Errorf("event stream status %d", resp.StatusCode)
		}
		return nil, err
	}
	done := make(chan struct{})
	go func() {
		defer close(done)
		sc := bufio.NewScanner(resp.Body)
		sc.Buffer(make([]byte, 0, 64<<10), 4<<20)
		for sc.Scan() {
			line := sc.Text()
			if !strings.HasPrefix(line, "data: ") {
				continue
			}
			var ev struct {
				Kind    string `json:"kind"`
				Payload struct {
					InvoiceID string `json:"invoice_id"`
					Status    string `json:"status"`
				} `json:"payload"`
			}
			if json.Unmarshal([]byte(strings.TrimPrefix(line, "data: ")), &ev) == nil && ev.Kind == store.EventInvoices {
				w.seen(ev.Payload.InvoiceID, ev.Payload.Status)
			}
		}
	}()
	return func() { cancel(); resp.Body.Close(); <-done }, nil
}

func (a *apiClient) uploadInvoice(ctx context.Context, project string, inv sample.Invoice) (string, error) {
	body := sample.PDF(inv)
	resp, err := a.do(ctx, http.MethodPost, "/projects/"+project+"/invoices?name="+url.QueryEscape(inv.Number+".pdf"), bytes.NewReader(body), int64(len(body)))
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusCreated {
		return "", fmt.Errorf("invoice upload %d: %s", resp.StatusCode, raw)
	}
	var out struct {
		ID string `json:"id"`
	}
	if err := json.Unmarshal(raw, &out); err != nil {
		return "", err
	}
	return out.ID, nil
}

func waitRead(ctx context.Context, ch chan struct{}) error {
	select {
	case <-ch:
		return nil
	case <-time.After(30 * time.Second):
		return fmt.Errorf("invoice not read within 30 s")
	case <-ctx.Done():
		return ctx.Err()
	}
}

// invoiceProject makes a project with consultant packages and a prepared
// cost plan, so invoice reads have suppliers and lines to choose from.
func (a *apiClient) invoiceProject(ctx context.Context, name string) (string, []string, error) {
	project, err := a.createProject(ctx, name)
	if err != nil {
		return "", nil, err
	}
	for _, title := range []string{"Architect", "Structural engineer", "Surveyor"} {
		body, _ := json.Marshal(map[string]string{"kind": "services", "title": title})
		resp, err := a.do(ctx, http.MethodPost, "/projects/"+project+"/packages", bytes.NewReader(body), int64(len(body)))
		if err != nil {
			return "", nil, err
		}
		resp.Body.Close()
		if resp.StatusCode != http.StatusCreated {
			return "", nil, fmt.Errorf("package create %d", resp.StatusCode)
		}
	}
	resp, err := a.do(ctx, http.MethodPost, "/projects/"+project+"/cost-plan/prepare", bytes.NewReader([]byte(`{}`)), 2)
	if err != nil {
		return "", nil, err
	}
	raw, _ := io.ReadAll(resp.Body)
	resp.Body.Close()
	var ready struct {
		Plan struct {
			Items []struct {
				ID      string `json:"cost_item_id"`
				Posting bool   `json:"posting"`
			} `json:"items"`
		} `json:"plan"`
	}
	if err := json.Unmarshal(raw, &ready); err != nil {
		return "", nil, fmt.Errorf("prepare cost plan: %w: %s", err, raw)
	}
	var lines []string
	for _, it := range ready.Plan.Items {
		if it.Posting {
			lines = append(lines, it.ID)
		}
	}
	return project, lines, nil
}

// invoices measures the ledger paths: one invoice from upload to its row,
// a batch of 20 to the last row, and read, re-map and export at 1,000 rows.
func (a *apiClient) invoices(ctx context.Context, st *store.Store, n int) error {
	project, _, err := a.invoiceProject(ctx, "Invoice intake benchmark")
	if err != nil {
		return err
	}
	watch := &invoiceWatch{read: map[string]chan struct{}{}}
	stop, err := a.invoiceStream(ctx, project, watch)
	if err != nil {
		return err
	}
	defer stop()
	next := 1
	for i := 0; i < n; i++ {
		inv := sample.Batch(next, 1)[0]
		next++
		start := time.Now()
		id, err := a.uploadInvoice(ctx, project, inv)
		if err != nil {
			return err
		}
		if err := waitRead(ctx, watch.wait(id)); err != nil {
			return err
		}
		a.samples.Add("invoice_intake", time.Since(start))
	}
	for r := 0; r < max(n, 20); r++ {
		batch := sample.Batch(next, 20)
		next += 20
		start := time.Now()
		ids := make([]string, len(batch))
		errs := make([]error, len(batch))
		var wg sync.WaitGroup
		for i := range batch {
			wg.Add(1)
			go func() {
				defer wg.Done()
				ids[i], errs[i] = a.uploadInvoice(ctx, project, batch[i])
				if errs[i] == nil {
					errs[i] = waitRead(ctx, watch.wait(ids[i]))
				}
			}()
		}
		wg.Wait()
		for _, err := range errs {
			if err != nil {
				return err
			}
		}
		a.samples.Add("invoice_batch", time.Since(start))
	}

	big, lines, err := a.invoiceProject(ctx, "Invoice ledger of 1,000")
	if err != nil {
		return err
	}
	if len(lines) < 2 {
		return fmt.Errorf("invoice bench needs two cost lines, has %d", len(lines))
	}
	ids, err := seedLedger(ctx, st, big, lines, 1000)
	if err != nil {
		return err
	}
	base := "/projects/" + big + "/invoices"
	for i := 0; i < n; i++ {
		raw, err := a.timed(ctx, "invoices_read", http.MethodGet, base, nil, http.StatusOK)
		if err != nil {
			return err
		}
		if i == 0 {
			var v struct {
				Invoices []json.RawMessage `json:"invoices"`
			}
			if err := json.Unmarshal(raw, &v); err != nil || len(v.Invoices) != 1000 {
				return fmt.Errorf("ledger read returned %d invoices: %v", len(v.Invoices), err)
			}
		}
		body, _ := json.Marshal(map[string]any{"fields": map[string]string{"line": lines[i%2]}})
		if _, err := a.timed(ctx, "invoice_write", http.MethodPatch, base+"/"+ids[i%len(ids)], body, http.StatusOK); err != nil {
			return err
		}
		if _, err := a.timed(ctx, "invoice_export", http.MethodGet, base+"/export", nil, http.StatusOK); err != nil {
			return err
		}
	}
	return nil
}

// seedLedger writes n read invoices straight to the store: the bench times
// reading the ledger, not filling it.
func seedLedger(ctx context.Context, st *store.Store, project string, lines []string, n int) ([]string, error) {
	ids := make([]string, 0, n)
	now := time.Now()
	for i := 0; i < n; i++ {
		sum := sha256.Sum256([]byte(fmt.Sprintf("bench ledger %s %d", project, i)))
		created, err := st.CreateInvoice(ctx, benchOrg, project, store.InvoiceUpload{SHA256: sum[:], Size: 1000, MediaType: "application/pdf", Filename: fmt.Sprintf("INV-%05d.pdf", i)}, now)
		if err != nil {
			return nil, err
		}
		ex := int64(100000 + i*137)
		d := func(f, v string) invoices.Decision {
			return invoices.Decision{Field: f, Value: v, Band: invoices.BandGreen, DecidedBy: invoices.DecidedByRule}
		}
		out := invoices.Outcome{InvoiceCount: 1}
		out.Decisions = []invoices.Decision{d(invoices.FieldKind, "invoice"), d(invoices.FieldNumber, fmt.Sprintf("INV-%05d", i)),
			d(invoices.FieldDate, "2026-10-01"), d(invoices.FieldTotal, strconv.FormatInt(ex+invoices.GSTOf(ex), 10)),
			d(invoices.FieldGST, strconv.FormatInt(invoices.GSTOf(ex), 10)), d(invoices.FieldLine, lines[i%len(lines)]),
			d(invoices.FieldPeriod, invoices.PeriodOf(now)), d(invoices.FieldCurrency, "AUD")}
		if _, err := st.CommitInvoiceRead(ctx, benchOrg, created.ID, out, false, now); err != nil {
			return nil, err
		}
		ids = append(ids, created.ID)
	}
	return ids, nil
}

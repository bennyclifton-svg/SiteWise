package httpapi

import (
	"bytes"
	"context"
	"errors"
	"io"
	"log"
	"mime"
	"net/http"
	"os"
	"sort"
	"strings"
	"sync"
	"time"

	"sitewise/internal/events"
	"sitewise/internal/files"
	"sitewise/internal/identity"
	"sitewise/internal/invoices"
	"sitewise/internal/jev"
	"sitewise/internal/store"
)

// Budget paths for the invoice ledger (bench/budgets.json).
const (
	pathInvoiceIntake = "invoice_intake"
	pathInvoicesRead  = "invoices_read"
	pathInvoiceWrite  = "invoice_write"
	pathInvoiceExport = "invoice_export"
)

// invoiceReader reads stored invoices outside the upload request, one
// goroutine per invoice, bounded by its own slots. Its Jev calls are
// interactive, so background work cannot take their reserve.
type invoiceReader struct {
	store      *store.Store
	blobs      *files.Store
	ask        invoices.Asker
	thresholds invoices.Thresholds
	broker     *events.Broker
	log        *log.Logger
	observe    func(string, time.Duration)
	timeout    time.Duration
	slots      chan struct{}
	now        func() time.Time
	wg         sync.WaitGroup

	mu       sync.Mutex
	inflight map[string]struct{}
}

// Start reads one invoice unless it is already being read. began is when
// its upload arrived, so invoice_intake covers upload to ledger row.
func (r *invoiceReader) Start(orgID, invoiceID string, began time.Time, recheck bool) {
	key := orgID + "/" + invoiceID
	r.mu.Lock()
	if _, busy := r.inflight[key]; busy {
		r.mu.Unlock()
		return
	}
	r.inflight[key] = struct{}{}
	r.mu.Unlock()
	r.wg.Add(1)
	go func() {
		defer r.wg.Done()
		defer func() {
			r.mu.Lock()
			delete(r.inflight, key)
			r.mu.Unlock()
		}()
		r.slots <- struct{}{}
		defer func() { <-r.slots }()
		ctx, cancel := context.WithTimeout(context.Background(), r.timeout)
		defer cancel()
		committed, err := r.read(ctx, orgID, invoiceID, recheck)
		if err != nil {
			r.log.Printf("invoice read stopped org=%s invoice=%s: %v", orgID, invoiceID, err)
		}
		r.broker.Wake(orgID)
		if committed && !recheck && !began.IsZero() && r.observe != nil {
			r.observe(pathInvoiceIntake, time.Since(began))
		}
	}()
}

func (r *invoiceReader) read(ctx context.Context, orgID, invoiceID string, recheck bool) (bool, error) {
	in, err := r.store.InvoiceForReading(ctx, orgID, invoiceID, recheck)
	if err != nil {
		return false, err
	}
	if !recheck && in.Status != invoices.StatusReading {
		return false, nil
	}
	out, err := r.outcome(ctx, in, recheck)
	if err != nil {
		return false, err
	}
	return r.store.CommitInvoiceRead(ctx, orgID, invoiceID, out, recheck, r.now())
}

func (r *invoiceReader) outcome(ctx context.Context, in store.InvoiceReadInput, recheck bool) (invoices.Outcome, error) {
	format := invoices.FormatOf(in.Filename)
	if format == "" {
		return invoices.Outcome{NotRead: invoices.NotReadUnsupported}, nil
	}
	path, err := r.blobs.Path(in.SHA256)
	if err != nil {
		return invoices.Outcome{}, err
	}
	f, err := os.Open(path)
	if err != nil {
		return invoices.Outcome{}, err
	}
	text, err := identity.Extract(ctx, format, f, in.Size, invoices.ExtractLimits())
	f.Close()
	switch {
	case errors.Is(err, identity.ErrTooLarge):
		return invoices.Outcome{NotRead: invoices.NotReadTooLarge}, nil
	case errors.Is(err, identity.ErrMalformed):
		return invoices.Outcome{NotRead: invoices.NotReadUnreadable}, nil
	case err != nil:
		return invoices.Outcome{}, err
	}
	priority := jev.PriorityInteractive
	if recheck {
		priority = jev.PriorityBackground
	}
	return invoices.Read(ctx, r.ask, r.thresholds, in.Filename, text, in.Context, in.Keep, in.CreatedAt, priority)
}

// RunInvoiceRechecks re-asks grey questions in the background until ctx
// ends. A re-check never replaces a value already read or typed.
func (s *Server) RunInvoiceRechecks(ctx context.Context, every time.Duration) {
	t := time.NewTicker(every)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
		}
		now := time.Now()
		due, err := s.store.DueInvoiceRechecks(ctx, now)
		if err != nil {
			continue
		}
		for _, ref := range due {
			if ok, err := s.store.ClaimInvoiceRecheck(ctx, ref, now); err == nil && ok {
				s.invoices.Start(ref.OrgID, ref.InvoiceID, time.Time{}, true)
			}
		}
	}
}

// uploadInvoice stores the bytes, creates the ledger row and starts the read
// without holding the request open for Jev. The row arrives as an event.
func uploadInvoice(w http.ResponseWriter, r *http.Request, deps Deps) {
	began := time.Now()
	session, ok := deliverySession(w, r, deps, true)
	if !ok {
		return
	}
	name := strings.TrimSpace(r.URL.Query().Get("name"))
	if name == "" || len(name) > 255 || strings.ContainsAny(name, "/\\") || strings.IndexFunc(name, func(r rune) bool { return r < 0x20 || r == 0x7f }) >= 0 {
		http.Error(w, "a file name is required", http.StatusBadRequest)
		return
	}
	body := http.MaxBytesReader(w, r.Body, deps.MaxUploadBytes)
	defer body.Close()
	blob, err := deps.Blobs.Put(r.Context(), body)
	var tooBig *http.MaxBytesError
	switch {
	case errors.As(err, &tooBig), errors.Is(err, files.ErrTooLarge):
		http.Error(w, "file is larger than the upload limit", http.StatusRequestEntityTooLarge)
		return
	case err != nil && r.Context().Err() != nil:
		return
	case err != nil:
		deps.Log.Printf("invoice upload failed org=%s: %v", session.OrgID, err)
		http.Error(w, "upload failed", http.StatusInternalServerError)
		return
	}
	up := store.InvoiceUpload{SHA256: blob.SHA256, Size: blob.Size, MediaType: invoiceMediaType(name), Filename: name, UserID: session.UserID}
	if invoices.FormatOf(name) == "" {
		up.NotRead = invoices.NotReadUnsupported
	} else if blob.Size == 0 {
		up.NotRead = invoices.NotReadUnreadable
	}
	created, err := deps.Store.CreateInvoice(r.Context(), session.OrgID, r.PathValue("id"), up, time.Now())
	switch {
	case errors.Is(err, store.ErrNotFound):
		http.NotFound(w, r)
		return
	case errors.Is(err, store.ErrMetadata), errors.Is(err, invoices.ErrInvalid):
		http.Error(w, "invalid upload", http.StatusUnprocessableEntity)
		return
	case err != nil:
		deps.Log.Printf("invoice create failed org=%s: %v", session.OrgID, err)
		http.Error(w, "upload failed", http.StatusInternalServerError)
		return
	}
	if created.Status == invoices.StatusReading && deps.InvoiceReader != nil {
		deps.InvoiceReader.Start(session.OrgID, created.ID, began, false)
	}
	if deps.Broker != nil {
		deps.Broker.Wake(session.OrgID)
	}
	status := http.StatusOK
	if created.Created {
		status = http.StatusCreated
	}
	writeJSON(w, status, created)
}

func invoiceMediaType(name string) string {
	switch invoices.FormatOf(name) {
	case "pdf":
		return "application/pdf"
	case "docx":
		return "application/vnd.openxmlformats-officedocument.wordprocessingml.document"
	case "xlsx":
		return "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet"
	}
	return "application/octet-stream"
}

// todaySydney is the date the odd-date check compares against.
func todaySydney() string {
	return time.Now().In(invoices.Sydney).Format("2006-01-02")
}

func getInvoices(w http.ResponseWriter, r *http.Request, deps Deps) {
	session, ok := deliverySession(w, r, deps, false)
	if !ok {
		return
	}
	v, err := deps.Store.InvoiceLedger(r.Context(), session.OrgID, r.PathValue("id"), todaySydney())
	if err != nil {
		invoiceError(w, deps, err)
		return
	}
	writeJSON(w, http.StatusOK, ledgerBody(v))
}

func patchInvoice(w http.ResponseWriter, r *http.Request, deps Deps) {
	session, ok := invoiceSession(w, r, deps)
	if !ok {
		return
	}
	var e store.InvoiceEdit
	if readProposalJSON(w, r, deps.MaxBodyBytes, &e) != nil || len(e.Fields) > 20 {
		http.Error(w, "invalid body", http.StatusBadRequest)
		return
	}
	if err := deps.Store.EditInvoice(r.Context(), session.OrgID, r.PathValue("id"), r.PathValue("inv"), session.UserID, e, time.Now()); err != nil {
		invoiceError(w, deps, err)
		return
	}
	invoiceWritten(w, r, deps, session)
}

func putInvoiceAllocations(w http.ResponseWriter, r *http.Request, deps Deps) {
	session, ok := invoiceSession(w, r, deps)
	if !ok {
		return
	}
	var body struct {
		Shares []store.ShareInput `json:"shares"`
	}
	if readProposalJSON(w, r, deps.MaxBodyBytes, &body) != nil {
		http.Error(w, "invalid body", http.StatusBadRequest)
		return
	}
	if err := deps.Store.SplitInvoice(r.Context(), session.OrgID, r.PathValue("id"), r.PathValue("inv"), session.UserID, body.Shares, time.Now()); err != nil {
		invoiceError(w, deps, err)
		return
	}
	invoiceWritten(w, r, deps, session)
}

// invoiceWritten answers a write with the recomputed ledger, so the panel
// shows the new summary without waiting for the event.
func invoiceWritten(w http.ResponseWriter, r *http.Request, deps Deps, session store.Session) {
	if deps.Broker != nil {
		deps.Broker.Wake(session.OrgID)
	}
	v, err := deps.Store.InvoiceLedger(r.Context(), session.OrgID, r.PathValue("id"), todaySydney())
	if err != nil {
		invoiceError(w, deps, err)
		return
	}
	writeJSON(w, http.StatusOK, ledgerBody(v))
}

func invoiceSession(w http.ResponseWriter, r *http.Request, deps Deps) (store.Session, bool) {
	if !uuidPattern.MatchString(r.PathValue("inv")) {
		http.NotFound(w, r)
		return store.Session{}, false
	}
	return deliverySession(w, r, deps, true)
}

func invoiceError(w http.ResponseWriter, deps Deps, err error) {
	switch {
	case errors.Is(err, store.ErrNotFound):
		http.Error(w, "not found", http.StatusNotFound)
	case errors.Is(err, store.ErrInvoiceSplit):
		http.Error(w, "this invoice is split; change its shares instead", http.StatusConflict)
	case errors.Is(err, invoices.ErrInvalid):
		http.Error(w, err.Error(), http.StatusUnprocessableEntity)
	default:
		deps.Log.Printf("invoice operation failed: %v", err)
		http.Error(w, "invoice operation failed", http.StatusInternalServerError)
	}
}

// downloadInvoice serves the stored file. view=1 shows a PDF inline beside
// the fields, framed only by this application's own pages.
func downloadInvoice(w http.ResponseWriter, r *http.Request, deps Deps) {
	if !uuidPattern.MatchString(r.PathValue("inv")) {
		http.NotFound(w, r)
		return
	}
	session, ok := deliverySession(w, r, deps, false)
	if !ok {
		return
	}
	file, name, err := deps.Store.InvoiceFile(r.Context(), session.OrgID, r.PathValue("id"), r.PathValue("inv"))
	if err != nil {
		invoiceError(w, deps, err)
		return
	}
	f, err := deps.Blobs.Open(file.SHA256)
	if err != nil {
		http.Error(w, "file unavailable", http.StatusInternalServerError)
		return
	}
	defer f.Close()
	disposition := "attachment"
	if r.URL.Query().Get("view") == "1" && file.MediaType == "application/pdf" {
		disposition = "inline"
		w.Header().Set("Content-Security-Policy", "default-src 'none'; frame-ancestors 'self'")
	}
	w.Header().Set("Content-Type", file.MediaType)
	w.Header().Set("Content-Disposition", mime.FormatMediaType(disposition, map[string]string{"filename": name}))
	_, _ = io.Copy(w, f)
}

// exportInvoices writes the workbook (or one sheet as CSV) of what the
// Invoices tab's filters show; the Cost plan sheet is always the whole plan.
func exportInvoices(w http.ResponseWriter, r *http.Request, deps Deps) {
	session, ok := deliverySession(w, r, deps, false)
	if !ok {
		return
	}
	q := r.URL.Query()
	format, sheet := q.Get("format"), q.Get("sheet")
	if format == "" {
		format = "xlsx"
	}
	if format != "xlsx" && format != "csv" || format == "csv" && sheet != "cost_plan" && sheet != "invoices" {
		http.Error(w, "format is xlsx, or csv with sheet cost_plan or invoices", http.StatusBadRequest)
		return
	}
	v, err := deps.Store.InvoiceLedger(r.Context(), session.OrgID, r.PathValue("id"), todaySydney())
	if err != nil {
		invoiceError(w, deps, err)
		return
	}
	f := invoices.Filter{Supplier: q.Get("supplier"), Month: q.Get("month"), Line: q.Get("line"), Status: q.Get("status")}
	nm := ledgerNames(v)
	meta := invoices.ExportMeta{Project: v.Project, Exported: time.Now(), Filters: f.String(func(id string) string { return nm.supplier(id) }, func(id string) string { return nm.line(id) })}
	var rows []invoices.ExportRow
	for _, inv := range v.Invoices {
		status := v.Status[inv.ID]
		for _, row := range invoices.Rows(inv) {
			if !f.Match(inv, row, status) {
				continue
			}
			rows = append(rows, invoices.ExportRow{Number: inv.Number, Supplier: nm.supplier(inv.PackageID), Date: inv.Date, Period: inv.Period,
				Line: nm.line(row.LineID), Status: invoices.StatusWord(status), Filename: inv.Filename, Ex: row.Ex, GST: row.GST, Total: row.Total})
		}
	}
	sheets := invoices.Workbook(meta, v.Summary, rows)
	base := safeFileName(v.Project) + " invoices " + time.Now().In(invoices.Sydney).Format("2006-01-02")
	var buf bytes.Buffer
	if format == "csv" {
		i := 0
		if sheet == "invoices" {
			i = 1
		}
		if err := invoices.WriteCSV(&buf, sheets[i]); err != nil {
			invoiceError(w, deps, err)
			return
		}
		w.Header().Set("Content-Type", "text/csv; charset=utf-8")
		w.Header().Set("Content-Disposition", mime.FormatMediaType("attachment", map[string]string{"filename": base + " " + strings.ReplaceAll(sheet, "_", " ") + ".csv"}))
	} else {
		if err := invoices.WriteXLSX(&buf, sheets); err != nil {
			invoiceError(w, deps, err)
			return
		}
		w.Header().Set("Content-Type", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
		w.Header().Set("Content-Disposition", mime.FormatMediaType("attachment", map[string]string{"filename": base + ".xlsx"}))
	}
	_, _ = w.Write(buf.Bytes())
}

func safeFileName(s string) string {
	s = strings.Map(func(r rune) rune {
		if r < 0x20 || strings.ContainsRune(`<>:"/\|?*`, r) {
			return ' '
		}
		return r
	}, s)
	s = strings.TrimSpace(s)
	if s == "" {
		return "project"
	}
	if len(s) > 80 {
		s = s[:80]
	}
	return s
}

type ledgerNameIndex struct {
	packages map[string]string
	lines    map[string]string
}

func ledgerNames(v store.LedgerView) ledgerNameIndex {
	n := ledgerNameIndex{packages: v.Titles, lines: map[string]string{}}
	for _, l := range v.Lines {
		n.lines[l.ID] = l.Label
	}
	return n
}

func (n ledgerNameIndex) supplier(id string) string { return n.packages[id] }

func (n ledgerNameIndex) line(id string) string {
	if id == "" || id == "unallocated" {
		return "Unallocated"
	}
	if l, ok := n.lines[id]; ok {
		return l
	}
	return "Removed line"
}

// The ledger as JSON. Money is an exact decimal string; null is unknown.

type moneyField = *string

func cents(p *int64) moneyField {
	if p == nil {
		return nil
	}
	s := invoices.FormatCents(*p)
	return &s
}

func centsValue(n int64) moneyField {
	return cents(&n)
}

type invoiceLineJSON struct {
	ID         string     `json:"id"`
	Label      string     `json:"label"`
	Section    string     `json:"section"`
	Category   string     `json:"category"`
	Archived   bool       `json:"archived"`
	Budget     moneyField `json:"budget"`
	Invoiced   moneyField `json:"invoiced"`
	Remaining  moneyField `json:"remaining"`
	Percent    string     `json:"percent_spent"`
	OverBudget bool       `json:"over_budget"`
	Invoices   int        `json:"invoices"`
}

type invoiceSummaryJSON struct {
	Lines         []invoiceLineJSON     `json:"lines"`
	Unallocated   invoiceLineJSON       `json:"unallocated"`
	Total         moneyField            `json:"invoiced"`
	Budget        moneyField            `json:"budget"`
	BudgetUnknown int                   `json:"budget_unknown"`
	Remaining     moneyField            `json:"remaining"`
	Percent       string                `json:"percent_spent"`
	Counted       int                   `json:"counted"`
	Excluded      int                   `json:"excluded"`
	Comparable    bool                  `json:"comparable"`
	Sections      map[string]moneyField `json:"sections"`
}

type invoiceFieldJSON struct {
	Value      string   `json:"value"`
	Band       string   `json:"band"`
	DecidedBy  string   `json:"decided_by"`
	Confidence *float64 `json:"confidence,omitempty"`
}

type invoiceAllocationJSON struct {
	ID        string     `json:"id"`
	LineID    string     `json:"line_id"`
	Line      string     `json:"line"`
	Amount    moneyField `json:"amount"`
	DecidedBy string     `json:"decided_by"`
	Band      string     `json:"band"`
}

type invoiceRowJSON struct {
	Share  int        `json:"share"`
	Shares int        `json:"shares"`
	LineID string     `json:"line_id"`
	Line   string     `json:"line"`
	Ex     moneyField `json:"ex_gst"`
	GST    moneyField `json:"gst"`
	Total  moneyField `json:"total"`
	Band   string     `json:"band"`
	By     string     `json:"decided_by"`
}

type invoiceJSON struct {
	ID            string                      `json:"id"`
	Filename      string                      `json:"filename"`
	Status        string                      `json:"status"`
	NotReadReason string                      `json:"not_read_reason,omitempty"`
	Kind          string                      `json:"kind"`
	Number        string                      `json:"number"`
	Date          string                      `json:"date"`
	Period        string                      `json:"period"`
	SupplierID    string                      `json:"supplier_id"`
	Supplier      string                      `json:"supplier"`
	ABN           string                      `json:"abn"`
	Currency      string                      `json:"currency"`
	Ex            moneyField                  `json:"ex_gst"`
	GST           moneyField                  `json:"gst"`
	Total         moneyField                  `json:"total"`
	InvoiceCount  int                         `json:"invoice_count"`
	SupersededBy  string                      `json:"superseded_by,omitempty"`
	DuplicateOK   bool                        `json:"duplicate_ok"`
	Counted       bool                        `json:"counted"`
	CreatedAt     time.Time                   `json:"created_at"`
	Flags         []invoices.Flag             `json:"flags"`
	Fields        map[string]invoiceFieldJSON `json:"fields"`
	Allocations   []invoiceAllocationJSON     `json:"allocations"`
	Rows          []invoiceRowJSON            `json:"rows"`
}

type invoiceLedgerJSON struct {
	Project  string             `json:"project"`
	TaxBasis string             `json:"tax_basis"`
	Summary  invoiceSummaryJSON `json:"summary"`
	Invoices []invoiceJSON      `json:"invoices"`
	Packages []invoices.Package `json:"packages"`
	Lines    []invoiceLineJSON  `json:"lines"`
}

func lineBody(l invoices.LineSummary) invoiceLineJSON {
	return invoiceLineJSON{ID: l.ID, Label: l.Label, Section: l.Section, Category: l.Category, Archived: l.Archived, Budget: cents(l.Budget),
		Invoiced: centsValue(l.Invoiced), Remaining: cents(l.Remaining), Percent: l.Percent, OverBudget: l.OverBudget, Invoices: l.Invoices}
}

func ledgerBody(v store.LedgerView) invoiceLedgerJSON {
	n := ledgerNames(v)
	out := invoiceLedgerJSON{Project: v.Project, TaxBasis: v.TaxBasis, Invoices: []invoiceJSON{}, Packages: v.Packages, Lines: []invoiceLineJSON{}}
	if out.Packages == nil {
		out.Packages = []invoices.Package{}
	}
	s := v.Summary
	out.Summary = invoiceSummaryJSON{Lines: []invoiceLineJSON{}, Unallocated: lineBody(s.Unallocated), Total: centsValue(s.Total), Budget: centsValue(s.Budget), BudgetUnknown: s.BudgetUnknown,
		Remaining: cents(s.Remaining), Percent: s.Percent, Counted: s.Counted, Excluded: s.Excluded, Comparable: s.Comparable, Sections: map[string]moneyField{}}
	for k, n := range s.Sections {
		out.Summary.Sections[k] = centsValue(n)
	}
	for _, l := range s.Lines {
		out.Summary.Lines = append(out.Summary.Lines, lineBody(l))
		out.Lines = append(out.Lines, invoiceLineJSON{ID: l.ID, Label: l.Label, Section: l.Section, Category: l.Category, Archived: l.Archived})
	}
	for _, inv := range v.Invoices {
		j := invoiceJSON{ID: inv.ID, Filename: inv.Filename, Status: v.Status[inv.ID], NotReadReason: inv.NotReadReason, Kind: inv.Kind, Number: inv.Number,
			Date: inv.Date, Period: inv.Period, SupplierID: inv.PackageID, Supplier: n.supplier(inv.PackageID), ABN: inv.ABN, Currency: inv.Currency,
			Ex: cents(inv.Ex), GST: cents(inv.GST), Total: cents(inv.Total), InvoiceCount: inv.InvoiceCount, SupersededBy: inv.SupersededBy,
			DuplicateOK: inv.DuplicateOK, Counted: invoices.Counted(inv), CreatedAt: inv.CreatedAt, Flags: v.Flags[inv.ID], Fields: map[string]invoiceFieldJSON{}}
		if j.Flags == nil {
			j.Flags = []invoices.Flag{}
		}
		for f, d := range inv.Decisions {
			j.Fields[f] = invoiceFieldJSON{Value: d.Value, Band: d.Band, DecidedBy: d.DecidedBy, Confidence: d.Confidence}
		}
		bands := map[string]invoices.Allocation{}
		for _, a := range inv.Allocations {
			j.Allocations = append(j.Allocations, invoiceAllocationJSON{ID: a.ID, LineID: a.LineID, Line: n.line(a.LineID), Amount: cents(a.Amount), DecidedBy: a.DecidedBy, Band: a.Band})
			bands[a.LineID] = a
		}
		for _, row := range invoices.Rows(inv) {
			a := bands[row.LineID]
			j.Rows = append(j.Rows, invoiceRowJSON{Share: row.Share, Shares: row.Shares, LineID: row.LineID, Line: n.line(row.LineID), Ex: cents(row.Ex), GST: cents(row.GST), Total: cents(row.Total), Band: a.Band, By: a.DecidedBy})
		}
		out.Invoices = append(out.Invoices, j)
	}
	sort.SliceStable(out.Invoices, func(i, j int) bool { return out.Invoices[i].CreatedAt.After(out.Invoices[j].CreatedAt) })
	return out
}

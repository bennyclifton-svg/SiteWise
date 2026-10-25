package httpapi

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"mime"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"sitewise/internal/events"
	"sitewise/internal/files"
	"sitewise/internal/identity"
	"sitewise/internal/intake"
	"sitewise/internal/invoices"
	"sitewise/internal/proposals"
	"sitewise/internal/rfp"
	"sitewise/internal/store"
	"sitewise/internal/xlsx"
)

// Budget paths for procurement (bench/budgets.json).
const (
	pathProposalRead      = "proposal_read"
	pathPriceTable        = "price_table"
	pathSubmissionWrite   = "submission_write"
	pathAwardWrite        = "award_write"
	pathProcurementReport = "procurement_report"
)

// proposalReader reads a filed proposal outside the filing request: one
// goroutine per submission, bounded by its own slots, at background
// priority so it never takes filing's interactive reserve.
type proposalReader struct {
	ocr        func(context.Context, string, identity.Text) (identity.Text, error)
	store      *store.Store
	blobs      *files.Store
	ask        proposals.Asker
	thresholds proposals.Thresholds
	broker     *events.Broker
	log        *log.Logger
	observe    func(string, time.Duration)
	timeout    time.Duration
	slots      chan struct{}
	now        func() time.Time
	wg         sync.WaitGroup

	mu       sync.Mutex
	inflight map[string]struct{}
	// after runs once a read commits: the proposal_program read queued
	// after this read finishes (plan 10.3). Nil runs nothing.
	after func(orgID, submissionID string)
	// disciplines names each filing discipline as Procurement shows it, so
	// a proposal's package can be made before it is read.
	disciplines map[string]string
}

// Filed is the filing hook: a proposal-like document gets a submission and
// a read. It returns at once.
func (r *proposalReader) Filed(orgID, documentID string, text identity.Text) {
	began := r.now()
	r.wg.Add(1)
	go func() {
		defer r.wg.Done()
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		id, err := r.store.SubmissionForDocument(ctx, orgID, documentID)
		if err != nil {
			r.log.Printf("proposal lookup org=%s doc=%s: %v", orgID, documentID, err)
			return
		}
		if id == "" {
			doc, err := r.store.GetDocument(ctx, orgID, documentID)
			if err != nil {
				return
			}
			kind, _, _ := r.store.FiledKind(ctx, orgID, documentID)
			if !proposals.Cue(doc.Filename, text, kind) {
				return
			}
			if id, err = r.store.OpenSubmission(ctx, orgID, doc.ProjectID, documentID, "", began); err != nil {
				r.log.Printf("proposal open org=%s doc=%s: %v", orgID, documentID, err)
				return
			}
			// A proposal for a consultancy with no row yet gets one, so the
			// read lands it under that consultancy with its firm and fee.
			if _, _, err := r.store.EnsureProposalPackage(ctx, orgID, doc.ProjectID, documentID, func(id string) string { return r.disciplines[id] }); err != nil {
				r.log.Printf("proposal package org=%s doc=%s: %v", orgID, documentID, err)
			}
			r.broker.Wake(orgID)
		}
		r.Start(orgID, id, began, false)
	}()
}

// Start reads one submission unless it is already being read.
func (r *proposalReader) Start(orgID, id string, began time.Time, recheck bool) {
	key := orgID + "/" + id
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
		committed, err := r.read(ctx, orgID, id, recheck)
		if err != nil {
			r.log.Printf("proposal read stopped org=%s submission=%s: %v", orgID, id, err)
		}
		r.broker.Wake(orgID)
		if committed && !recheck && !began.IsZero() && r.observe != nil {
			r.observe(pathProposalRead, time.Since(began))
		}
		if committed && r.after != nil {
			r.after(orgID, id)
		}
	}()
}

func (r *proposalReader) read(ctx context.Context, orgID, id string, recheck bool) (bool, error) {
	in, err := r.store.SubmissionForReading(ctx, orgID, id, recheck)
	if err != nil {
		return false, err
	}
	if !recheck && in.Status != store.SubmissionReading {
		return false, nil
	}
	out, err := r.outcome(ctx, orgID, in)
	if err != nil {
		return false, err
	}
	return r.store.CommitSubmissionRead(ctx, orgID, id, out, recheck, r.now())
}

func (r *proposalReader) outcome(ctx context.Context, orgID string, in store.SubmissionReadInput) (proposals.Outcome, error) {
	format := proposalFormat(in.Filename)
	if format == "" {
		return proposals.Outcome{NotRead: proposals.NotReadUnsupported}, nil
	}
	path, err := r.blobs.Path(in.SHA256)
	if err != nil {
		return proposals.Outcome{}, err
	}
	f, err := os.Open(path)
	if err != nil {
		return proposals.Outcome{}, err
	}
	text, err := identity.Extract(ctx, format, f, in.Size, proposals.ExtractLimits())
	f.Close()
	switch {
	case errors.Is(err, identity.ErrTooLarge), errors.Is(err, identity.ErrMalformed):
		return proposals.Outcome{NotRead: proposals.NotReadUnreadable, Reading: proposals.Reading{Flags: []string{proposals.FlagNoText}}}, nil
	case err != nil:
		return proposals.Outcome{}, err
	}
	pc := in.Context
	if format == "pdf" && len(identity.MissingTextPages(text)) > 0 {
		text.Incomplete = true
		if r.ocr != nil {
			text, err = r.ocr(ctx, path, text)
			if err != nil {
				text.Incomplete = true
			}
		}
	}
	pc.Lookup = r.store.FirmLookup(ctx, orgID)
	return proposals.Read(ctx, r.ask, r.thresholds, in.Filename, text, pc, in.Keep)
}

// ProposalFiled is also used by the durable OCR worker after identity filing.
func (s *Server) ProposalFiled(orgID, documentID string, text identity.Text) {
	s.proposals.Filed(orgID, documentID, text)
}

func proposalFormat(name string) string {
	switch strings.ToLower(name[strings.LastIndex(name, ".")+1:]) {
	case "pdf":
		return "pdf"
	case "docx":
		return "docx"
	case "xlsx":
		return "xlsx"
	// Filing accepts Markdown, so a Markdown proposal is read too, not
	// left as "needs a package" with nothing read.
	case "md", "markdown":
		return "md"
	}
	return ""
}

// RunProposalRechecks re-asks grey proposal reads in the background.
func (s *Server) RunProposalRechecks(ctx context.Context, every time.Duration) {
	t := time.NewTicker(every)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
		}
		now := time.Now()
		due, err := s.store.DueSubmissionRechecks(ctx, now)
		if err != nil {
			continue
		}
		for _, ref := range due {
			if ok, err := s.store.ClaimSubmissionRecheck(ctx, ref, now); err == nil && ok {
				s.proposals.Start(ref.OrgID, ref.SubmissionID, time.Time{}, true)
			}
		}
	}
}

// ProposalReader starts and resumes proposal reads.
type ProposalReader interface {
	Start(orgID, id string, began time.Time, recheck bool)
}

func procurementError(w http.ResponseWriter, deps Deps, err error) {
	switch {
	case errors.Is(err, store.ErrNotFound):
		http.Error(w, "not found", http.StatusNotFound)
	case errors.Is(err, store.ErrInvalidInput):
		http.Error(w, err.Error(), http.StatusUnprocessableEntity)
	case errors.Is(err, store.ErrConflict), errors.Is(err, store.ErrVersionConflict):
		http.Error(w, err.Error(), http.StatusConflict)
	case errors.Is(err, store.ErrInvalidReport):
		http.Error(w, "this report has no response form", http.StatusNotFound)
	default:
		deps.Log.Printf("procurement operation failed: %v", err)
		http.Error(w, "procurement operation failed", http.StatusInternalServerError)
	}
}

func pathIDs(w http.ResponseWriter, r *http.Request, names ...string) bool {
	for _, n := range names {
		if !uuidPattern.MatchString(r.PathValue(n)) {
			http.NotFound(w, r)
			return false
		}
	}
	return true
}

// JSON shapes. Money is an exact decimal string; null is unknown.

type priceCellJSON struct {
	proposals.Cell
	Amount moneyField `json:"amount"`
}

type totalsJSON struct {
	BaseTotal    moneyField `json:"base_total"`
	KnownBase    moneyField `json:"known_base"`
	NotPriced    int        `json:"not_priced"`
	Unread       int        `json:"unread"`
	OptionsTotal moneyField `json:"options_total"`
	OthersTotal  moneyField `json:"others_total"`
	StatedTotal  moneyField `json:"stated_total"`
	Gap          moneyField `json:"gap"`
	Lowest       bool       `json:"lowest"`
}

type columnJSON struct {
	AppointmentTotal *string                       `json:"appointment_total"`
	Quote            *proposals.QuoteReview        `json:"quote,omitempty"`
	AwardStages      []proposals.QuotedFee         `json:"award_stages,omitempty"`
	Builder          *proposals.BuilderReview      `json:"builder,omitempty"`
	UndoAlignment    bool                          `json:"undo_alignment"`
	Submission       store.Submission              `json:"submission"`
	Firm             *store.Firm                   `json:"firm"`
	Cells            map[string]priceCellJSON      `json:"cells"`
	Others           []priceCellJSON               `json:"others"`
	Totals           totalsJSON                    `json:"totals"`
	NeedsReview      bool                          `json:"needs_review"`
	Fields           map[string]proposals.Decision `json:"fields"`
}

type tableJSON struct {
	store.PriceTable
	Columns []columnJSON `json:"columns"`
	Award   *awardJSON   `json:"award"`
}

func cellOf(c proposals.Cell) priceCellJSON { return priceCellJSON{Cell: c, Amount: cents(c.Cents)} }

func tableBody(t store.PriceTable, award *store.AwardView) tableJSON {
	out := tableJSON{PriceTable: t, Columns: []columnJSON{}}
	for _, c := range t.Columns {
		col := columnJSON{Submission: c.Submission, Firm: c.Firm, Cells: map[string]priceCellJSON{}, Others: []priceCellJSON{}, NeedsReview: c.NeedsReview, Fields: map[string]proposals.Decision{}}
		col.UndoAlignment = c.UndoAlignment
		col.Builder = c.Builder
		col.Quote = c.Quote
		col.AwardStages = c.AwardStages
		col.AppointmentTotal = cents(c.AppointmentTotal)
		for k, v := range c.Cells {
			col.Cells[k] = cellOf(v)
		}
		for _, o := range c.Others {
			col.Others = append(col.Others, cellOf(o))
		}
		for _, f := range proposals.FirmFields {
			if d, ok := c.Decisions[f]; ok {
				col.Fields[f] = d
			}
		}
		tt := c.Totals
		col.Totals = totalsJSON{BaseTotal: cents(tt.BaseTotal), KnownBase: centsValue(tt.KnownBase), NotPriced: tt.NotPriced, Unread: tt.Unread,
			OptionsTotal: centsValue(tt.OptionsTotal), OthersTotal: centsValue(tt.OthersTotal), StatedTotal: cents(c.Submission.StatedTotal), Gap: cents(tt.Gap), Lowest: tt.Lowest}
		out.Columns = append(out.Columns, col)
	}
	if award != nil {
		a := awardBody(*award)
		out.Award = &a
	}
	return out
}

func getPriceTable(w http.ResponseWriter, r *http.Request, deps Deps) {
	if !pathIDs(w, r, "pkg") {
		return
	}
	session, ok := deliverySession(w, r, deps, false)
	if !ok {
		return
	}
	writePriceTable(w, r, deps, session, http.StatusOK)
}

func writePriceTable(w http.ResponseWriter, r *http.Request, deps Deps, session store.Session, status int) {
	t, err := deps.Store.ReadPriceTable(r.Context(), session.OrgID, r.PathValue("id"), r.PathValue("pkg"))
	if err != nil {
		procurementError(w, deps, err)
		return
	}
	award, err := deps.Store.ReadAward(r.Context(), session.OrgID, r.PathValue("id"), r.PathValue("pkg"))
	if err != nil {
		procurementError(w, deps, err)
		return
	}
	writeJSON(w, status, tableBody(t, award))
}

type cardJSON struct {
	store.FirmCard
	BaseTotal   moneyField `json:"base_total"`
	KnownBase   moneyField `json:"known_base"`
	StatedTotal moneyField `json:"stated_total"`
}

func getProcurementBoard(w http.ResponseWriter, r *http.Request, deps Deps) {
	session, ok := deliverySession(w, r, deps, false)
	if !ok {
		return
	}
	board, loose, err := deps.Store.ProcurementBoard(r.Context(), session.OrgID, r.PathValue("id"))
	if err != nil {
		procurementError(w, deps, err)
		return
	}
	out := struct {
		Packages   map[string][]cardJSON `json:"packages"`
		Unassigned []cardJSON            `json:"unassigned"`
	}{Packages: map[string][]cardJSON{}, Unassigned: []cardJSON{}}
	conv := func(c store.FirmCard) cardJSON {
		return cardJSON{FirmCard: c, BaseTotal: cents(c.BaseTotal), KnownBase: centsValue(c.KnownBase), StatedTotal: cents(c.StatedTotal)}
	}
	for pkg, cards := range board {
		for _, c := range cards {
			out.Packages[pkg] = append(out.Packages[pkg], conv(c))
		}
	}
	for _, c := range loose {
		out.Unassigned = append(out.Unassigned, conv(c))
	}
	writeJSON(w, http.StatusOK, out)
}

// uploadSubmission files a proposal dropped on a package row: it goes into
// the document register like any upload, and the package is the user's.
func linkProposalDocument(w http.ResponseWriter, r *http.Request, deps Deps) {
	if !pathIDs(w, r, "pkg") {
		return
	}
	session, ok := deliverySession(w, r, deps, true)
	if !ok {
		return
	}
	var body struct {
		DocumentID string `json:"document_id"`
	}
	if readProposalJSON(w, r, deps.MaxBodyBytes, &body) != nil || !uuidPattern.MatchString(body.DocumentID) {
		http.Error(w, "invalid document", http.StatusBadRequest)
		return
	}
	doc, err := deps.Store.GetDocument(r.Context(), session.OrgID, body.DocumentID)
	if err != nil {
		procurementError(w, deps, err)
		return
	}
	if doc.ProjectID != r.PathValue("id") {
		http.NotFound(w, r)
		return
	}
	if doc.Status != store.StatusFiled {
		http.Error(w, "Wait for the document to finish filing.", http.StatusConflict)
		return
	}
	id, err := deps.Store.OpenSubmission(r.Context(), session.OrgID, doc.ProjectID, doc.ID, r.PathValue("pkg"), time.Now())
	if err != nil {
		procurementError(w, deps, err)
		return
	}
	if deps.ProposalReader != nil {
		deps.ProposalReader.Start(session.OrgID, id, time.Now(), false)
	}
	if deps.Broker != nil {
		deps.Broker.Wake(session.OrgID)
	}
	writeJSON(w, http.StatusOK, map[string]string{"submission_id": id, "document_id": doc.ID})
}

func uploadSubmission(w http.ResponseWriter, r *http.Request, deps Deps) {
	if !pathIDs(w, r, "pkg") {
		return
	}
	session, ok := deliverySession(w, r, deps, true)
	if !ok {
		return
	}
	target := r.URL.Query().Get("submission_id")
	if target != "" && !uuidPattern.MatchString(target) {
		http.Error(w, "invalid submission_id", http.StatusBadRequest)
		return
	}
	if err := deps.Store.CheckSubmissionUpload(r.Context(), session.OrgID, r.PathValue("id"), r.PathValue("pkg"), target); err != nil {
		procurementError(w, deps, err)
		return
	}
	name := strings.TrimSpace(r.URL.Query().Get("name"))
	body := http.MaxBytesReader(w, r.Body, deps.MaxUploadBytes)
	defer body.Close()
	filing, err := deps.Uploader.Upload(r.Context(), session.OrgID, r.PathValue("id"), intake.Upload{Filename: name, Body: body, Reason: intake.ReasonFor(name)})
	var tooBig *http.MaxBytesError
	switch {
	case errors.As(err, &tooBig), errors.Is(err, files.ErrTooLarge):
		http.Error(w, "file is larger than the upload limit", http.StatusRequestEntityTooLarge)
		return
	case err != nil && r.Context().Err() != nil:
		return
	case errors.Is(err, intake.ErrFilename):
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	case err != nil:
		deps.Log.Printf("proposal upload failed org=%s: %v", session.OrgID, err)
		http.Error(w, "upload failed", http.StatusInternalServerError)
		return
	}
	var id string
	if target == "" {
		id, err = deps.Store.OpenSubmission(r.Context(), session.OrgID, r.PathValue("id"), filing.DocumentID, r.PathValue("pkg"), time.Now())
	} else {
		id, err = deps.Store.AttachSubmissionDocument(r.Context(), session.OrgID, r.PathValue("id"), r.PathValue("pkg"), target, filing.DocumentID, time.Now())
	}
	if err != nil {
		procurementError(w, deps, err)
		return
	}
	if filing.Status == store.StatusPending {
		deps.Filer.Start(session.OrgID, filing.DocumentID)
	} else if deps.ProposalReader != nil {
		// Bytes already filed: read now with the package the user chose.
		deps.ProposalReader.Start(session.OrgID, id, time.Now(), target == "")
	}
	if deps.Broker != nil {
		deps.Broker.Wake(session.OrgID)
	}
	writeJSON(w, http.StatusCreated, map[string]string{"submission_id": id, "document_id": filing.DocumentID})
}

// postFirmSubmission adds a directory firm, or a new one, under a package
// with no file; its prices are typed in the table.
func postFirmSubmission(w http.ResponseWriter, r *http.Request, deps Deps) {
	if !pathIDs(w, r, "pkg") {
		return
	}
	session, ok := deliverySession(w, r, deps, true)
	if !ok {
		return
	}
	var body struct {
		FirmID string            `json:"firm_id"`
		Firm   *store.FirmFields `json:"firm"`
	}
	if readProposalJSON(w, r, deps.MaxBodyBytes, &body) != nil || (body.FirmID == "") == (body.Firm == nil) || body.FirmID != "" && !uuidPattern.MatchString(body.FirmID) {
		http.Error(w, "firm_id or firm is required", http.StatusBadRequest)
		return
	}
	firm := body.FirmID
	if body.Firm != nil {
		f, err := deps.Store.CreateFirm(r.Context(), session.OrgID, *body.Firm)
		if err != nil {
			procurementError(w, deps, err)
			return
		}
		firm = f
	}
	if _, err := deps.Store.AddFirmSubmission(r.Context(), session.OrgID, r.PathValue("id"), r.PathValue("pkg"), firm, time.Now()); err != nil {
		procurementError(w, deps, err)
		return
	}
	if deps.Broker != nil {
		deps.Broker.Wake(session.OrgID)
	}
	writePriceTable(w, r, deps, session, http.StatusCreated)
}

func putSubmissionCell(w http.ResponseWriter, r *http.Request, deps Deps) {
	if !pathIDs(w, r, "sub") {
		return
	}
	session, ok := deliverySession(w, r, deps, true)
	if !ok {
		return
	}
	var in store.CellInput
	if readProposalJSON(w, r, deps.MaxBodyBytes, &in) != nil {
		http.Error(w, "invalid body", http.StatusBadRequest)
		return
	}
	key := r.PathValue("key")
	if len(key) == 0 || len(key) > 200 {
		http.NotFound(w, r)
		return
	}
	if err := deps.Store.SetSubmissionCell(r.Context(), session.OrgID, r.PathValue("id"), r.PathValue("sub"), key, in); err != nil {
		procurementError(w, deps, err)
		return
	}
	submissionWritten(w, r, deps, session)
}

func putComparisonRow(w http.ResponseWriter, r *http.Request, deps Deps) {
	if !pathIDs(w, r, "pkg") {
		return
	}
	session, ok := deliverySession(w, r, deps, true)
	if !ok {
		return
	}
	var in store.ComparisonRowInput
	if readProposalJSON(w, r, deps.MaxBodyBytes, &in) != nil {
		http.Error(w, "invalid body", http.StatusBadRequest)
		return
	}
	if err := deps.Store.SetComparisonRow(r.Context(), session.OrgID, r.PathValue("id"), r.PathValue("pkg"), in); err != nil {
		procurementError(w, deps, err)
		return
	}
	if deps.Broker != nil {
		deps.Broker.Wake(session.OrgID)
	}
	writePriceTable(w, r, deps, session, http.StatusOK)
}

func postSubmissionAlignment(w http.ResponseWriter, r *http.Request, deps Deps) {
	if !pathIDs(w, r, "sub") {
		return
	}
	session, ok := deliverySession(w, r, deps, true)
	if !ok {
		return
	}
	var in store.AlignmentInput
	if readProposalJSON(w, r, deps.MaxBodyBytes, &in) != nil {
		http.Error(w, "invalid body", http.StatusBadRequest)
		return
	}
	if err := deps.Store.AlignSubmission(r.Context(), session.OrgID, r.PathValue("id"), r.PathValue("sub"), in); err != nil {
		procurementError(w, deps, err)
		return
	}
	submissionWritten(w, r, deps, session)
}

func patchSubmission(w http.ResponseWriter, r *http.Request, deps Deps) {
	if !pathIDs(w, r, "sub") {
		return
	}
	session, ok := deliverySession(w, r, deps, true)
	if !ok {
		return
	}
	var e store.SubmissionEdit
	if readProposalJSON(w, r, deps.MaxBodyBytes, &e) != nil {
		http.Error(w, "invalid body", http.StatusBadRequest)
		return
	}
	if e.PackageID != nil && *e.PackageID != "" && !uuidPattern.MatchString(*e.PackageID) || e.FirmID != nil && *e.FirmID != "" && !uuidPattern.MatchString(*e.FirmID) {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}
	if err := deps.Store.EditSubmission(r.Context(), session.OrgID, r.PathValue("id"), r.PathValue("sub"), e); err != nil {
		procurementError(w, deps, err)
		return
	}
	submissionWritten(w, r, deps, session)
}

// submissionWritten answers with the submission's package table so the
// screen shows the recomputed totals without waiting for the event.
func submissionWritten(w http.ResponseWriter, r *http.Request, deps Deps, session store.Session) {
	if deps.Broker != nil {
		deps.Broker.Wake(session.OrgID)
	}
	pkg, err := deps.Store.SubmissionPackage(r.Context(), session.OrgID, r.PathValue("id"), r.PathValue("sub"))
	if err != nil {
		procurementError(w, deps, err)
		return
	}
	if pkg == "" {
		writeJSON(w, http.StatusOK, map[string]string{"submission_id": r.PathValue("sub")})
		return
	}
	r.SetPathValue("pkg", pkg)
	writePriceTable(w, r, deps, session, http.StatusOK)
}

func downloadSubmission(w http.ResponseWriter, r *http.Request, deps Deps) {
	if !pathIDs(w, r, "sub") {
		return
	}
	session, ok := deliverySession(w, r, deps, false)
	if !ok {
		return
	}
	file, name, err := deps.Store.SubmissionFile(r.Context(), session.OrgID, r.PathValue("id"), r.PathValue("sub"))
	if err != nil {
		procurementError(w, deps, err)
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

func getFirms(w http.ResponseWriter, r *http.Request, deps Deps) {
	session, ok := memberSession(w, r, deps)
	if !ok {
		return
	}
	q := r.URL.Query().Get("q")
	if len(q) > 100 {
		http.Error(w, "query too long", http.StatusBadRequest)
		return
	}
	firms, err := deps.Store.ReadFirms(r.Context(), session.OrgID, q)
	if err != nil {
		procurementError(w, deps, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"firms": firms})
}

// getResponseForm writes the RFP's response form as a workbook.
func getResponseForm(w http.ResponseWriter, r *http.Request, deps Deps) {
	if !pathIDs(w, r, "r") {
		return
	}
	session, ok := memberSession(w, r, deps)
	if !ok {
		return
	}
	form, err := deps.Store.ReadResponseForm(r.Context(), session.OrgID, r.PathValue("r"))
	if err != nil {
		procurementError(w, deps, err)
		return
	}
	var buf bytes.Buffer
	if err := xlsx.WriteXLSX(&buf, rfp.ResponseForm(form.Header, form.Items, form.Returnables)); err != nil {
		procurementError(w, deps, err)
		return
	}
	w.Header().Set("Content-Type", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
	w.Header().Set("Content-Disposition", mime.FormatMediaType("attachment", map[string]string{"filename": safeFileName(form.Header.Title) + " response form.xlsx"}))
	_, _ = w.Write(buf.Bytes())
}

// exportPriceTable writes the price table as a workbook in the screen's
// layout: items down, firms across, totals computed in Go.
func exportPriceTable(w http.ResponseWriter, r *http.Request, deps Deps) {
	if !pathIDs(w, r, "pkg") {
		return
	}
	session, ok := deliverySession(w, r, deps, false)
	if !ok {
		return
	}
	t, err := deps.Store.ReadPriceTable(r.Context(), session.OrgID, r.PathValue("id"), r.PathValue("pkg"))
	if err != nil {
		procurementError(w, deps, err)
		return
	}
	var buf bytes.Buffer
	if err := xlsx.WriteXLSX(&buf, []xlsx.Sheet{priceSheet(t)}); err != nil {
		procurementError(w, deps, err)
		return
	}
	w.Header().Set("Content-Type", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
	w.Header().Set("Content-Disposition", mime.FormatMediaType("attachment", map[string]string{"filename": safeFileName(t.Title) + " price comparison " + store.SydneyDate(time.Now()) + ".xlsx"}))
	_, _ = w.Write(buf.Bytes())
}

// priceSheet lays a price table out as the screen does.
func priceSheet(t store.PriceTable) xlsx.Sheet {
	bold := func(s string) xlsx.Cell { return xlsx.Cell{Text: s, Bold: true} }
	sheet := xlsx.Sheet{Name: "Price comparison", Widths: []float64{50}, TitleRow: 1}
	head := []xlsx.Cell{bold(t.Title + " — item (ex GST)")}
	for _, c := range t.Columns {
		name := c.Submission.Filename
		if c.Firm != nil {
			name = c.Firm.Name
		}
		head = append(head, bold(name))
		sheet.Widths = append(sheet.Widths, 20)
	}
	sheet.Rows = append(sheet.Rows, head)
	cell := func(c proposals.Cell, ok bool) xlsx.Cell {
		switch {
		case !ok || c.State == "":
			return xlsx.Cell{Text: "Not read"}
		case c.State == proposals.StatePriced && c.Cents != nil:
			v := *c.Cents
			return xlsx.Cell{Cents: &v}
		case c.State == proposals.StatePriced:
			return xlsx.Cell{Text: "Priced, amount unknown"}
		case c.State == proposals.StateIncluded:
			return xlsx.Cell{Text: "Included"}
		case c.State == proposals.StateExcluded:
			return xlsx.Cell{Text: "Excluded"}
		}
		return xlsx.Cell{Text: "Not priced"}
	}
	row := func(label string, isBold bool, f func(store.FirmColumn) xlsx.Cell) {
		r := []xlsx.Cell{{Text: label, Bold: isBold}}
		for _, c := range t.Columns {
			r = append(r, f(c))
		}
		sheet.Rows = append(sheet.Rows, r)
	}
	quoted := false
	for _, c := range t.Columns {
		quoted = quoted || c.Quote != nil
	}
	if quoted && !t.Builder {
		row("Quoted fees — source amounts, not additive", true, func(c store.FirmColumn) xlsx.Cell {
			var lines []string
			if c.Quote != nil {
				for _, fee := range c.Quote.Fees {
					amount := fee.Printed + " (GST unclear)"
					if fee.Amount != nil {
						amount = "$" + *fee.Amount + " ex GST"
					}
					lines = append(lines, amount+" — "+fee.Label+" — "+fee.Role+fmt.Sprintf(" (page %d)", fee.Page))
				}
			}
			return xlsx.Cell{Text: strings.Join(lines, "\n")}
		})
		row("Confirmed comparison fee", true, func(c store.FirmColumn) xlsx.Cell { v, ok := c.Cells["comparison:appointment"]; return cell(v, ok) })
		row("Confirmed scope and source", false, func(c store.FirmColumn) xlsx.Cell { return xlsx.Cell{Text: c.Cells["comparison:appointment"].Basis} })
		for _, it := range t.Items {
			if !it.Price() || it.Key == "comparison:appointment" {
				continue
			}
			row(it.Label+" — scope coverage", false, func(c store.FirmColumn) xlsx.Cell {
				if own, ok := c.Cells[it.Key]; ok && own.DecidedBy == proposals.ByUser {
					return xlsx.Cell{Text: own.State + " — confirmed by you. " + own.Basis}
				}
				if c.Quote == nil {
					return xlsx.Cell{Text: "Coverage unclear"}
				}
				coverage := c.Quote.Coverage[it.Key]
				label := coverage.Status
				for _, fee := range c.Quote.Fees {
					if fee.ID == coverage.FeeID {
						label += ": " + fee.Label
					}
				}
				return xlsx.Cell{Text: label}
			})
		}
		return sheet
	}
	itemRow := func(it store.ScheduleItem, label string) {
		row(label, false, func(c store.FirmColumn) xlsx.Cell { v, ok := c.Cells[it.Key]; return cell(v, ok) })
		hasNote := false
		for _, c := range t.Columns {
			if c.Cells[it.Key].Basis != "" {
				hasNote = true
				break
			}
		}
		if hasNote {
			row(label+" — scope / allowance", false, func(c store.FirmColumn) xlsx.Cell { return xlsx.Cell{Text: c.Cells[it.Key].Basis} })
		}
	}
	if t.Builder {
		row("Quoted total ex GST (source reading; review before use)", true, func(c store.FirmColumn) xlsx.Cell {
			if c.Builder == nil || c.Builder.ExGST == nil {
				return xlsx.Cell{Text: "Unresolved"}
			}
			v, ok, err := invoices.ParseCents(*c.Builder.ExGST)
			if !ok || err != nil {
				return xlsx.Cell{Text: "Unresolved"}
			}
			return xlsx.Cell{Cents: &v}
		})
		row("Quoted total basis and source", false, func(c store.FirmColumn) xlsx.Cell {
			if c.Builder == nil {
				return xlsx.Cell{}
			}
			return xlsx.Cell{Text: c.Builder.Basis + "\n" + strings.Join(c.Builder.Sources, "\n") + "\n" + strings.Join(c.Builder.Issues, "\n")}
		})
	}
	for _, it := range t.Items {
		if it.Price() && it.Base {
			itemRow(it, it.Label)
		}
	}
	totalLabel := "Base fee total"
	if t.Builder {
		totalLabel = "Comparison total (separate from quoted total)"
	}
	row(totalLabel, true, func(c store.FirmColumn) xlsx.Cell {
		if c.Totals.BaseTotal == nil {
			return xlsx.Cell{Text: "Unknown", Bold: true}
		}
		v := *c.Totals.BaseTotal
		return xlsx.Cell{Cents: &v, Bold: true}
	})
	row("Items not priced", false, func(c store.FirmColumn) xlsx.Cell {
		n := float64(c.Totals.NotPriced)
		return xlsx.Cell{Number: &n}
	})
	for _, it := range t.Items {
		if it.Price() && !it.Base {
			itemRow(it, "Option: "+it.Label)
		}
	}
	if !t.Builder {
		row("Fees awaiting alignment", false, func(c store.FirmColumn) xlsx.Cell {
			var parts []string
			for _, o := range c.Others {
				if o.Cents != nil {
					parts = append(parts, "$"+*cents(o.Cents)+" "+o.Label)
				}
			}
			return xlsx.Cell{Text: strings.Join(parts, "; ")}
		})
	}
	if t.Builder && len(t.Columns) > 0 && t.Columns[0].Builder != nil {
		for _, section := range t.Columns[0].Builder.Sections {
			row(section.Label+" — source detail, not additive", false, func(c store.FirmColumn) xlsx.Cell {
				var text []string
				if c.Builder != nil {
					for _, s := range c.Builder.Sections {
						if s.Key == section.Key {
							for _, d := range s.Details {
								text = append(text, d.Relationship+": "+d.Text)
							}
						}
					}
				}
				if len(text) == 0 {
					return xlsx.Cell{Text: "No matching wording found"}
				}
				return xlsx.Cell{Text: strings.Join(text, "\n")}
			})
		}
	}
	for _, it := range t.Items {
		if it.Kind == rfp.ItemRate {
			unit := it.Unit
			if unit == "" {
				unit = "hour"
			}
			itemRow(it, it.Label+" (per "+unit+")")
		}
	}
	return sheet
}

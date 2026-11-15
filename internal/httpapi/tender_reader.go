package httpapi

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"sync"
	"time"

	"sitewise/internal/events"
	"sitewise/internal/files"
	"sitewise/internal/identity"
	"sitewise/internal/llm"
	"sitewise/internal/store"
	"sitewise/internal/tender"
)

// Budget paths for the tender pass (bench/budgets.json).
const (
	pathTenderRead     = "tender_read"
	pathTenderReadLong = "tender_read_long"
	pathTenderEdit     = "tender_edit"
	pathChangeUndo     = "change_undo"
)

// Words the screen shows when a pass fails. Never a library's own error.
const (
	tenderUnreachable = "The model could not be reached. It will be tried again."
	tenderGaveUp      = "The model could not be reached."
	tenderListedNone  = "The model listed no prices. It will be tried again."
	tenderDeclined    = "The model declined to read this file."
	tenderNoImages    = "Page images cannot be made on this server."
	tenderWrongKind   = "This kind of file cannot be read for prices."
	tenderUnreadable  = "This file could not be opened."
)

// tenderReader runs the tender pass on a submission after its first read
// has committed: one goroutine per submission, two at a time. It is the
// only place a tender page leaves the server, and it runs only for a
// submission QueueTenderRead accepted, which needs the org's switch.
type tenderReader struct {
	store   *store.Store
	blobs   *files.Store
	ask     llm.Asker
	render  tender.Renderer
	model   string
	ocr     func(context.Context, string, identity.Text) (identity.Text, error)
	broker  *events.Broker
	log     *log.Logger
	observe func(string, time.Duration)
	timeout time.Duration
	slots   chan struct{}
	now     func() time.Time
	wg      sync.WaitGroup

	mu       sync.Mutex
	inflight map[string]struct{}
}

// Queue asks for a pass when the org's switch is on and the submission is
// ready, and starts it. It is the hook after the first read commits and
// after a submission is placed in a package.
func (r *tenderReader) Queue(orgID, id string, began time.Time) {
	if r == nil {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	ok, err := r.store.QueueTenderRead(ctx, orgID, id, false)
	if err != nil {
		r.log.Printf("tender queue org=%s submission=%s: %v", orgID, id, err)
		return
	}
	if ok {
		r.Start(orgID, id, began)
	}
}

// Start runs one pass unless that submission is already being read.
func (r *tenderReader) Start(orgID, id string, began time.Time) {
	if r == nil {
		return
	}
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
		lines, committed, err := r.read(ctx, orgID, id)
		if err != nil {
			r.log.Printf("tender read stopped org=%s submission=%s: %v", orgID, id, err)
		}
		r.broker.Wake(orgID)
		if committed && !began.IsZero() && r.observe != nil {
			path := pathTenderRead
			if lines > 100 {
				path = pathTenderReadLong
			}
			r.observe(path, time.Since(began))
		}
	}()
}

func (r *tenderReader) read(ctx context.Context, orgID, id string) (lines int, committed bool, err error) {
	claimed, err := r.store.ClaimTenderRead(ctx, orgID, id, r.now())
	if err != nil || !claimed {
		return 0, false, err
	}
	// From here every way out records a state: a claimed read left as
	// "reading" would never be tried again.
	raw, info, pages, rows, stated, err := r.pass(ctx, orgID, id)
	if err != nil {
		return 0, false, r.fail(orgID, id, err)
	}
	reading := tender.Reconcile(raw, pages, rows, stated)
	// The commit must land even when the pass used most of the time allowed.
	commitCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 20*time.Second)
	defer cancel()
	if err := r.store.CommitTenderRead(commitCtx, orgID, id, reading, tender.PromptVersion, info.Cut || info.Dropped > 0, r.now()); err != nil {
		return 0, false, r.fail(orgID, id, err)
	}
	return len(reading.Lines), true, nil
}

var errTenderKind = errors.New("unsupported file kind")
var errTenderFile = errors.New("unreadable file")

func (r *tenderReader) pass(ctx context.Context, orgID, id string) (tender.RawReading, tender.PassInfo, []tender.Page, []tender.Row, *int64, error) {
	var none tender.RawReading
	in, err := r.store.TenderInput(ctx, orgID, id)
	if err != nil {
		return none, tender.PassInfo{}, nil, nil, nil, err
	}
	format := proposalFormat(in.Filename)
	if format == "" {
		return none, tender.PassInfo{}, nil, nil, nil, errTenderKind
	}
	// The cause goes to the log; the screen is told "This file could not be opened."
	path, err := r.blobs.Path(in.SHA256)
	if err != nil {
		return none, tender.PassInfo{}, nil, nil, nil, fmt.Errorf("%w: no stored file: %v", errTenderFile, err)
	}
	f, err := os.Open(path)
	if err != nil {
		return none, tender.PassInfo{}, nil, nil, nil, fmt.Errorf("%w: open: %v", errTenderFile, err)
	}
	text, err := identity.Extract(ctx, format, f, in.Size, tender.ExtractLimits())
	f.Close()
	if err != nil {
		if errors.Is(err, context.DeadlineExceeded) || errors.Is(err, context.Canceled) {
			return none, tender.PassInfo{}, nil, nil, nil, err
		}
		return none, tender.PassInfo{}, nil, nil, nil, fmt.Errorf("%w: extract: %v", errTenderFile, err)
	}
	var missing []int
	if format == "pdf" {
		// A page with no text layer goes through the existing OCR first, so
		// that every amount can be checked against text (rule R3).
		if missing = identity.MissingTextPages(text); len(missing) > 0 && r.ocr != nil {
			if read, err := r.ocr(ctx, path, text); err == nil {
				text = read
			}
		}
	}
	pages := tender.PagesOf(text, missing)
	raw, info, err := tender.Pass(ctx, r.ask, r.render, r.model, path, format == "pdf", pages, in.Rows)
	return raw, info, pages, in.Rows, in.StatedEx, err
}

// tenderFailure says, for the cause of a failed pass and how many times it
// has now been tried, what the screen is told and how long to wait before
// trying again. wait is zero when it is not tried again. off is true when
// the pass turned out to be switched off, which is not a failure.
func tenderFailure(cause error, attempts int) (words string, wait time.Duration, off bool) {
	switch {
	case errors.Is(cause, llm.ErrOff):
		return "", 0, true
	case errors.Is(cause, tender.ErrNoRenderer):
		return tenderNoImages, 0, false
	case errors.Is(cause, llm.ErrRefused):
		return tenderDeclined, 0, false
	case errors.Is(cause, errTenderKind):
		return tenderWrongKind, 0, false
	case errors.Is(cause, errTenderFile), errors.Is(cause, store.ErrNotFound):
		return tenderUnreadable, 0, false
	}
	// Anything else may pass on a later try: three more, further apart.
	waits := []time.Duration{30 * time.Second, 2 * time.Minute, 10 * time.Minute}
	if attempts < 1 || attempts > len(waits) {
		return tenderGaveUp, 0, false
	}
	if errors.Is(cause, tender.ErrEmpty) {
		return tenderListedNone, waits[attempts-1], false
	}
	return tenderUnreachable, waits[attempts-1], false
}

// fail records why a pass stopped, in plain words, and when to try again.
func (r *tenderReader) fail(orgID, id string, cause error) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	attempts, _ := r.store.TenderAttempts(ctx, orgID, id)
	words, wait, off := tenderFailure(cause, attempts)
	if off {
		return r.store.ResetTenderRead(ctx, orgID, id)
	}
	var retry *time.Time
	if wait > 0 {
		at := r.now().Add(wait)
		retry = &at
	}
	return errors.Join(cause, r.store.FailTenderRead(ctx, orgID, id, words, retry))
}

// RunTenderRetries starts failed passes whose retry time has come.
func (s *Server) RunTenderRetries(ctx context.Context, every time.Duration) {
	if s.tenders == nil {
		return
	}
	t := time.NewTicker(every)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
		}
		due, err := s.store.DueTenderRetries(ctx, time.Now())
		if err != nil {
			continue
		}
		for _, ref := range due {
			s.tenders.Start(ref.OrgID, ref.SubmissionID, time.Time{})
		}
	}
}

// postTenderRead is "Read again" and "Try again": it re-runs the pass for
// one firm. The manager's own lines are kept and the column goes back to
// "to check".
func postTenderRead(w http.ResponseWriter, r *http.Request, deps Deps) {
	if !pathIDs(w, r, "sub") {
		return
	}
	session, ok := deliverySession(w, r, deps, true)
	if !ok {
		return
	}
	pkg, err := deps.Store.SubmissionPackage(r.Context(), session.OrgID, r.PathValue("id"), r.PathValue("sub"))
	if err != nil {
		procurementError(w, deps, err)
		return
	}
	if deps.TenderReader == nil {
		http.Error(w, "This server is not set up to use a model.", http.StatusConflict)
		return
	}
	on, err := deps.Store.ModelEnabled(r.Context(), session.OrgID)
	if err != nil {
		procurementError(w, deps, err)
		return
	}
	if !on {
		http.Error(w, "The model is switched off for this organisation.", http.StatusConflict)
		return
	}
	queued, err := deps.Store.QueueTenderRead(r.Context(), session.OrgID, r.PathValue("sub"), true)
	if err != nil {
		procurementError(w, deps, err)
		return
	}
	if !queued || pkg == "" {
		http.Error(w, "This submission cannot be read for prices yet.", http.StatusConflict)
		return
	}
	deps.TenderReader.Start(session.OrgID, r.PathValue("sub"), time.Now())
	submissionWritten(w, r, deps, session)
}

// putAccountModel is the org's one switch for sending project text to the
// model provider. Only the owner turns it.
func putAccountModel(w http.ResponseWriter, r *http.Request, deps Deps) {
	if !originOK(r, deps.PublicOrigin) {
		http.Error(w, "origin rejected", http.StatusForbidden)
		return
	}
	session, ok := ownerSession(w, r, deps)
	if !ok {
		return
	}
	var in struct {
		Enabled *bool `json:"enabled"`
	}
	if readProposalJSON(w, r, deps.MaxBodyBytes, &in) != nil || in.Enabled == nil {
		http.Error(w, "invalid body", http.StatusBadRequest)
		return
	}
	// The one switch covers the tender pass and chat; either makes it real.
	if deps.TenderReader == nil && deps.Chat == nil {
		http.Error(w, "This server is not set up to use a model.", http.StatusConflict)
		return
	}
	if err := deps.Store.SetModelEnabled(r.Context(), session.OrgID, session.UserID, *in.Enabled, time.Now()); err != nil {
		readFailed(w, deps, "account model", err)
		return
	}
	view, err := modelView(r.Context(), deps, session.OrgID)
	if err != nil {
		readFailed(w, deps, "account model", err)
		return
	}
	writeJSON(w, http.StatusOK, view)
}

type modelJSON struct {
	Available bool       `json:"available"`
	Enabled   bool       `json:"enabled"`
	EnabledAt *time.Time `json:"enabled_at"`
}

func modelView(ctx context.Context, deps Deps, orgID string) (modelJSON, error) {
	at, err := deps.Store.ModelEnabledAt(ctx, orgID)
	return modelJSON{Available: deps.TenderReader != nil, Enabled: at != nil, EnabledAt: at}, err
}

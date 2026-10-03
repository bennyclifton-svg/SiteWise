package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"io/fs"
	"log"
	"net/http"
	"strings"
	"sync"
	"time"

	"sitewise/bench"
	"sitewise/internal/events"
	"sitewise/internal/files"
	"sitewise/internal/identity"
	"sitewise/internal/intake"
	"sitewise/internal/jev"
	"sitewise/internal/knowledge"
	"sitewise/internal/profile"
	"sitewise/internal/store"
)

const (
	// defaultFilingTimeout bounds one foreground filing after the upload
	// request has returned. The Jev call inside it has its own 1.2 s deadline.
	defaultFilingTimeout = 15 * time.Second
	// defaultFilingSlots caps concurrent foreground filings. Jev admission
	// still enforces its own interactive reserve.
	defaultFilingSlots = 16
)

// Options is the process wiring for one server.
type Options struct {
	OCR            func(context.Context, string) (identity.Text, error)
	Store          *store.Store
	Blobs          *files.Store
	Jev            intake.Asker
	Catalog        intake.Catalog
	Thresholds     intake.Thresholds
	Static         fs.FS
	PublicOrigin   string
	SecureCookie   bool
	MaxUploadBytes int64
	Log            *log.Logger
	FilingTimeout  time.Duration
	// Observe receives per-path filing latency; nil records nothing.
	Observe intake.Observer
	// DevLogin mounts GET /dev/login, which signs a visitor in as the local
	// owner. serve enables it only on a loopback address outside production.
	DevLogin bool
	// Knowledge and ProfileThresholds serve the project profile. Nil
	// knowledge disables the profile routes (404).
	Knowledge         *knowledge.Catalog
	ProfileThresholds profile.Thresholds
}

// Server is the API under /api, the event stream, and the embedded SPA.
type Server struct {
	handler   http.Handler
	store     *store.Store
	filer     *filer
	closing   chan struct{}
	closeOnce sync.Once
}

// New composes the upload, filing and event path. Nothing here starts a
// goroutine until a filing is started or resumed.
func New(opts Options) (*Server, error) {
	if opts.Store == nil || opts.Blobs == nil || opts.Jev == nil || opts.Static == nil {
		return nil, errors.New("server is not configured")
	}
	if opts.PublicOrigin == "" {
		return nil, errors.New("public origin is required for origin checks")
	}
	if opts.Log == nil {
		opts.Log = log.New(io.Discard, "", 0)
	}
	if opts.FilingTimeout <= 0 {
		opts.FilingTimeout = defaultFilingTimeout
	}
	svc, err := intake.NewService(opts.Store, opts.Jev, opts.Catalog, opts.Thresholds)
	if err != nil {
		return nil, err
	}
	budgets, err := bench.Budgets()
	if err != nil {
		return nil, err
	}
	speed := newRecorder()
	svc.Observe(func(path string, d time.Duration) {
		speed.Observe(path, d)
		if opts.Observe != nil {
			opts.Observe(path, d)
		}
	})
	checker := &healthChecker{ping: opts.Store.Ping, now: time.Now}
	// A client that reports its own state feeds health; any other Asker
	// leaves Jev reported as unknown, never as healthy.
	if s, ok := opts.Jev.(interface{ Status() jev.Status }); ok {
		checker.jev = s.Status
	}
	broker := events.NewBroker(opts.Store)
	runner := intake.NewRunner(opts.Blobs, opts.Store, svc)
	runner.OCR = opts.OCR
	f := &filer{
		observe: speed.Observe,
		run:     runner.Run,
		expand:  runner.ExpandDrawing,
		expansionFailed: func(orgID, documentID string) {
			_ = opts.Store.ReviewDrawingExpansion(context.Background(), orgID, documentID, "Sheet processing stopped. The original is retained. You can retry processing.")
		},
		splitSlots: make(chan struct{}, 1),
		broker:     broker,
		log:        opts.Log,
		timeout:    opts.FilingTimeout,
		slots:      make(chan struct{}, defaultFilingSlots),
		inflight:   map[string]struct{}{},
	}
	if opts.OCR != nil {
		f.isOCR = func(ctx context.Context, orgID, documentID string) bool {
			doc, err := opts.Store.GetDocument(ctx, orgID, documentID)
			return err == nil && strings.HasPrefix(doc.Reason, "ocr_")
		}
	}
	closing := make(chan struct{})
	api := Handler(Deps{
		Store:          opts.Store,
		Blobs:          opts.Blobs,
		PublicOrigin:   opts.PublicOrigin,
		Log:            opts.Log,
		SecureCookie:   opts.SecureCookie,
		MaxUploadBytes: opts.MaxUploadBytes,
		Uploader:       intake.NewUploader(opts.Blobs, opts.Store),
		Service:        svc,
		Catalog:        opts.Catalog,
		Broker:         broker,
		Filer:          f,
		Closing:        closing,
		Health:         checker,
		Speed:          speed,
		Budgets:        budgets,

		Knowledge:         opts.Knowledge,
		ProfileThresholds: opts.ProfileThresholds,
	})
	mux := http.NewServeMux()
	mux.Handle("/healthz", &publicHealth{checker: checker, backlog: opts.Store.Backlog, log: opts.Log, observe: speed.Observe})
	mux.Handle("/api/", http.StripPrefix("/api", noStore(api)))
	app := spa(opts.Static)
	if opts.DevLogin {
		mux.Handle("GET /dev/login", noStore(devLogin(opts.Store, opts.SecureCookie, opts.Log)))
		// tools/dev.ps1 starts serve in the repo root, so "." is the source tree.
		mux.Handle("GET /dev/build", noStore(devBuild(".", time.Now())))
		app = localAppSession(app, opts.Store, opts.SecureCookie, opts.Log)
	}
	mux.Handle("/", app)
	return &Server{handler: secureHeaders(mux), store: opts.Store, filer: f, closing: closing}, nil
}

// CloseStreams ends open event streams. Clients reconnect from their cursor.
func (s *Server) CloseStreams() {
	s.closeOnce.Do(func() { close(s.closing) })
}

// ServeHTTP serves the whole application.
func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.handler.ServeHTTP(w, r)
}

// Resume restarts filings that were stored but not filed when the process
// last stopped. Each keeps its own org.
func (s *Server) Resume(ctx context.Context) (int, error) {
	pending, err := s.store.PendingIntake(ctx)
	if err != nil {
		return 0, err
	}
	for _, p := range pending {
		s.filer.start(p.OrgID, p.DocumentID, false)
	}
	expansions, err := s.store.PendingDrawingExpansions(ctx)
	if err != nil {
		return len(pending), err
	}
	for _, p := range expansions {
		s.filer.startExpansion(p.OrgID, p.DocumentID)
	}
	return len(pending) + len(expansions), nil
}

// Wait blocks until started filings finish or ctx ends.
func (s *Server) Wait(ctx context.Context) error {
	done := make(chan struct{})
	go func() {
		s.filer.wg.Wait()
		close(done)
	}()
	select {
	case <-done:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

// filer runs foreground filings outside the upload request, so a closed tab
// does not abort a filing whose bytes are already stored.
type filer struct {
	isOCR func(context.Context, string, string) bool
	// observe records whole_intake for filings started by an upload or retry.
	observe         func(string, time.Duration)
	run             func(ctx context.Context, orgID, documentID string) error
	expand          func(ctx context.Context, orgID, documentID string) error
	splitSlots      chan struct{}
	expansionFailed func(orgID, documentID string)
	broker          *events.Broker
	log             *log.Logger
	timeout         time.Duration
	slots           chan struct{}
	wg              sync.WaitGroup

	mu       sync.Mutex
	inflight map[string]struct{}
}

// Start files the document unless a filing for it is already running. A
// failure is written as an event so the client can offer a retry.
func (f *filer) Start(orgID, documentID string) {
	f.start(orgID, documentID, true)
}

// start times the filing from here to its event being written when timed.
// Resumed filings are not timed: their clock would start at process restart.
// This is the server's share of whole_intake; the bench's client-side figure
// also includes the final upload byte and SSE delivery.
func (f *filer) start(orgID, documentID string, timed bool) {
	began := time.Now()
	key := orgID + "/" + documentID
	f.mu.Lock()
	if _, busy := f.inflight[key]; busy {
		f.mu.Unlock()
		return
	}
	f.inflight[key] = struct{}{}
	f.mu.Unlock()
	f.wg.Add(1)
	go func() {
		defer f.wg.Done()
		defer func() {
			f.mu.Lock()
			delete(f.inflight, key)
			f.mu.Unlock()
		}()
		f.slots <- struct{}{}
		defer func() { <-f.slots }()
		ctx, cancel := context.WithTimeout(context.Background(), f.timeout)
		err := f.run(ctx, orgID, documentID)
		cancel()
		if err != nil {
			f.log.Printf("filing stopped org=%s document=%s: %v", orgID, documentID, err)
			payload, _ := json.Marshal(map[string]string{"document_id": documentID, "status": store.StatusPending})
			if _, err := f.broker.Commit(context.Background(), orgID, "filing_failed", documentID, string(payload)); err != nil {
				f.log.Printf("filing_failed event not written org=%s: %v", orgID, err)
			}
		}
		f.broker.Wake(orgID)
		if err == nil {
			f.startExpansion(orgID, documentID)
		}
		if timed && f.observe != nil {
			elapsed := time.Since(began)
			// A queue handoff is not a completed filing. OCR has its own gate;
			// do not make whole_intake look faster by counting queued scans.
			if f.isOCR == nil || !f.isOCR(context.Background(), orgID, documentID) {
				f.observe(pathWholeIntake, elapsed)
			}
		}
	}()
}

// Expansion never occupies a foreground filing slot while waiting for later sheets.
func (f *filer) startExpansion(orgID, documentID string) {
	if f.expand == nil {
		return
	}
	f.wg.Add(1)
	go func() {
		defer f.wg.Done()
		f.splitSlots <- struct{}{}
		defer func() { <-f.splitSlots }()
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
		defer cancel()
		if err := f.expand(ctx, orgID, documentID); err != nil {
			f.log.Printf("drawing expansion stopped org=%s document=%s: %v", orgID, documentID, err)
			if f.expansionFailed != nil {
				f.expansionFailed(orgID, documentID)
			}
		}
		f.broker.Wake(orgID)
	}()
}

// spa serves built files, and index.html for client routes. Hashed assets
// are immutable; index.html is revalidated so a deploy is picked up.
func spa(static fs.FS) http.Handler {
	files := http.FileServerFS(static)
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet && r.Method != http.MethodHead {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		name := strings.TrimPrefix(r.URL.Path, "/")
		if name != "" && name != "index.html" {
			if info, err := fs.Stat(static, name); err == nil && !info.IsDir() {
				if strings.HasPrefix(name, "assets/") {
					w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
				}
				files.ServeHTTP(w, r)
				return
			}
			if strings.HasPrefix(name, "assets/") {
				http.NotFound(w, r)
				return
			}
		}
		index, err := fs.ReadFile(static, "index.html")
		if err != nil {
			http.Error(w, "web build missing: run npm --prefix web run build", http.StatusServiceUnavailable)
			return
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Header().Set("Cache-Control", "no-cache")
		_, _ = w.Write(index)
	})
}

func noStore(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "no-store")
		next.ServeHTTP(w, r)
	})
}

// secureHeaders keeps the page to its own origin. The SPA uses no external
// scripts, fonts or frames.
func secureHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		h := w.Header()
		h.Set("Content-Security-Policy", "default-src 'self'; img-src 'self' data:; object-src 'none'; base-uri 'none'; frame-ancestors 'none'; form-action 'self'")
		h.Set("X-Content-Type-Options", "nosniff")
		h.Set("Referrer-Policy", "no-referrer")
		next.ServeHTTP(w, r)
	})
}

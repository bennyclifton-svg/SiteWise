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

	"sitewise/internal/events"
	"sitewise/internal/files"
	"sitewise/internal/intake"
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
	broker := events.NewBroker(opts.Store)
	runner := intake.NewRunner(opts.Blobs, opts.Store, svc)
	f := &filer{
		run:      runner.Run,
		broker:   broker,
		log:      opts.Log,
		timeout:  opts.FilingTimeout,
		slots:    make(chan struct{}, defaultFilingSlots),
		inflight: map[string]struct{}{},
	}
	closing := make(chan struct{})
	api := Handler(Deps{
		Store:          opts.Store,
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
	})
	mux := http.NewServeMux()
	mux.Handle("/api/", http.StripPrefix("/api", noStore(api)))
	mux.Handle("/", spa(opts.Static))
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
		s.filer.Start(p.OrgID, p.DocumentID)
	}
	return len(pending), nil
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
	run     func(ctx context.Context, orgID, documentID string) error
	broker  *events.Broker
	log     *log.Logger
	timeout time.Duration
	slots   chan struct{}
	wg      sync.WaitGroup

	mu       sync.Mutex
	inflight map[string]struct{}
}

// Start files the document unless a filing for it is already running. A
// failure is written as an event so the client can offer a retry.
func (f *filer) Start(orgID, documentID string) {
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

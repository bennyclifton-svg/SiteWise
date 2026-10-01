// Command e2eserver runs the real server against the sitewise_test database
// with a recorded Jev, for the Playwright intake flow. It is never deployed.
//
// Recorded answers pick each question's first option. Test-only thresholds
// make kind green, discipline amber and lifecycle blank so every band shows;
// a filename starting "slow-" makes Jev miss its deadline so fields go grey.
package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"log"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"os/signal"
	"sort"
	"strings"
	"sync"
	"time"

	"sitewise/internal/auth"
	"sitewise/internal/config"
	"sitewise/internal/files"
	"sitewise/internal/httpapi"
	"sitewise/internal/identity"
	"sitewise/internal/intake"
	"sitewise/internal/jev"
	"sitewise/internal/store"
	"sitewise/web"
)

// Fixed org ids so each run deletes the previous run's rows first.
var orgs = map[string]string{
	"a": "e2e00000-0000-4000-8000-00000000000a",
	"b": "e2e00000-0000-4000-8000-00000000000b",
}

var confidence = map[string]float64{
	intake.FieldKind:       0.95,
	intake.FieldDiscipline: 0.7,
	intake.FieldLifecycle:  0.3,
}

func main() {
	addr := flag.String("addr", "127.0.0.1:4173", "listen address")
	data := flag.String("data", "data/intake", "intake vocabulary directory")
	flag.Parse()
	if err := run(*addr, *data); err != nil {
		log.Fatal(err)
	}
}

func run(addr, data string) error {
	dsn := os.Getenv("SITEWISE_TEST_DATABASE_URL")
	u, err := url.Parse(dsn)
	if err != nil || strings.TrimPrefix(u.Path, "/") != "sitewise_test" {
		return errors.New("SITEWISE_TEST_DATABASE_URL must name the sitewise_test database")
	}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()
	st, err := store.Open(ctx, dsn)
	if err != nil {
		return err
	}
	defer st.Close()
	if err := st.Migrate(ctx); err != nil {
		return err
	}
	for name, id := range orgs {
		if err := st.DeleteOrg(ctx, id); err != nil {
			return err
		}
		if err := st.CreateOrg(ctx, id, "E2E org "+strings.ToUpper(name)); err != nil {
			return err
		}
	}
	dir, err := os.MkdirTemp("", "sitewise-e2e-")
	if err != nil {
		return err
	}
	defer os.RemoveAll(dir)
	blobs, err := files.Open(dir, 50<<20)
	if err != nil {
		return err
	}
	cat, err := intake.LoadCatalog(data)
	if err != nil {
		return err
	}
	// Warm PDFium as serve does, so filed-in times match production startup.
	if err := identity.Warm(ctx); err != nil {
		return err
	}
	fake := httptest.NewServer(http.HandlerFunc(recorded))
	defer fake.Close()
	client, err := jev.New(jev.Options{
		BaseURL: fake.URL,
		APIKey:  "e2e",
		Logger:  slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelWarn})),
	})
	if err != nil {
		return err
	}
	srv, err := httpapi.New(httpapi.Options{
		Store:          st,
		Blobs:          blobs,
		Jev:            client,
		Catalog:        cat,
		Thresholds:     thresholds(cat),
		Static:         web.Dist(),
		PublicOrigin:   "http://" + addr,
		MaxUploadBytes: 50 << 20,
		Log:            log.New(os.Stderr, "e2e ", log.LstdFlags),
	})
	if err != nil {
		return err
	}
	cut := &streamCutter{cancels: map[int]context.CancelFunc{}}
	mux := http.NewServeMux()
	mux.HandleFunc("POST /__e2e/invite", func(w http.ResponseWriter, r *http.Request) {
		invite(w, r, st)
	})
	mux.HandleFunc("POST /__e2e/drop-streams", func(w http.ResponseWriter, r *http.Request) {
		cut.dropAll()
		w.WriteHeader(http.StatusNoContent)
	})
	mux.Handle("/", cut.wrap(srv))
	hs := &http.Server{Addr: addr, Handler: mux, ReadHeaderTimeout: 10 * time.Second}
	hs.RegisterOnShutdown(srv.CloseStreams)
	go func() {
		<-ctx.Done()
		shutdown, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = hs.Shutdown(shutdown)
	}()
	fmt.Printf("e2e server on http://%s\n", addr)
	if err := hs.ListenAndServe(); !errors.Is(err, http.ErrServerClosed) {
		return err
	}
	return nil
}

// streamCutter ends open event streams on demand, standing in for a network
// drop: browser offline emulation does not cut a stream that is already open.
type streamCutter struct {
	mu      sync.Mutex
	next    int
	cancels map[int]context.CancelFunc
}

func (c *streamCutter) wrap(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/events" {
			next.ServeHTTP(w, r)
			return
		}
		ctx, cancel := context.WithCancel(r.Context())
		c.mu.Lock()
		id := c.next
		c.next++
		c.cancels[id] = cancel
		c.mu.Unlock()
		defer func() {
			c.mu.Lock()
			delete(c.cancels, id)
			c.mu.Unlock()
			cancel()
		}()
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func (c *streamCutter) dropAll() {
	c.mu.Lock()
	defer c.mu.Unlock()
	for _, cancel := range c.cancels {
		cancel()
	}
}

func invite(w http.ResponseWriter, r *http.Request, st *store.Store) {
	orgID, ok := orgs[r.URL.Query().Get("org")]
	if !ok {
		http.Error(w, "org must be a or b", http.StatusBadRequest)
		return
	}
	email := "member@" + r.URL.Query().Get("org") + ".e2e.test"
	raw, err := auth.CreateInvite(r.Context(), st, &auth.Sink{}, orgID, email, "member", time.Now().Add(time.Hour), nil)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]string{"token": raw})
}

// thresholds are test data only: production cut-offs stay unknown until the
// calibration run. Option counts match each catalog question exactly.
func thresholds(cat intake.Catalog) intake.Thresholds {
	cut := func(n int) intake.Threshold {
		g, a := 0.9, 0.6
		return intake.Threshold{Green: &g, Amber: &a, Options: &n}
	}
	return intake.Thresholds{
		QuestionVersion: intake.QuestionVersion,
		Reconciliation:  "e2e fixture thresholds; not calibrated",
		Questions: map[string][]intake.Threshold{
			intake.FieldKind:       {cut(len(cat.Kinds))},
			intake.FieldDiscipline: {cut(len(cat.Disciplines))},
			intake.FieldLifecycle:  {cut(len(cat.Lifecycle))},
		},
	}
}

func recorded(w http.ResponseWriter, r *http.Request) {
	var call struct {
		State struct {
			Filename string `json:"filename"`
		} `json:"state"`
		Questions map[string]struct {
			Criteria map[string]string `json:"criteria"`
		} `json:"questions"`
	}
	if err := json.NewDecoder(r.Body).Decode(&call); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	if strings.HasPrefix(strings.ToLower(call.State.Filename), "slow-") {
		select {
		case <-r.Context().Done():
		case <-time.After(3 * time.Second):
		}
		return
	}
	answers := map[string]any{}
	for id, q := range call.Questions {
		opts := make([]string, 0, len(q.Criteria))
		for opt := range q.Criteria {
			opts = append(opts, opt)
		}
		sort.Strings(opts)
		if len(opts) == 0 {
			continue
		}
		probs := map[string]float64{}
		for _, opt := range opts {
			probs[opt] = 0
		}
		probs[opts[0]] = 1
		conf, ok := confidence[id]
		if !ok {
			conf = 0.5
		}
		answers[id] = map[string]any{"type": "choice", "choice": opts[0], "probabilities": probs, "confidence": conf}
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{
		"model":   config.PinnedJevModel,
		"answers": answers,
		"usage":   map[string]int{"input_tokens": 10, "output_tokens": 2},
	})
}

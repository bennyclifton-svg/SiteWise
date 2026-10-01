// Command intake-bench measures every budgeted path through the real server
// and fails when a budget in bench/budgets.json is broken.
//
//	go run ./cmd/intake-bench -manifest data/eval/intake/manifest.json -budgets bench/budgets.json
//
// It drops the verified corpus into fresh projects over HTTP, timing each
// filing from the final upload byte to its event on an open SSE stream, so
// queue time, errors and grey (degraded) filings are all in the samples.
// Stage timings come from the intake Observer inside the same filings.
//
// Jev is live with -live, or replays the recorded provider latency per
// document. A replay is a timing model, not a measurement of the provider:
// the release gate is -live on the intended VPS. Results record the target,
// hardware, concurrency, background load and degraded share.
package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"log"
	"log/slog"
	"net"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"sitewise/internal/auth"
	"sitewise/internal/config"
	"sitewise/internal/eval"
	"sitewise/internal/files"
	"sitewise/internal/httpapi"
	"sitewise/internal/identity"
	"sitewise/internal/intake"
	"sitewise/internal/jev"
	"sitewise/internal/latency"
	"sitewise/internal/store"
	"sitewise/web"
)

// benchOrg is fixed so each run deletes the previous run's rows first.
const benchOrg = "be4c0000-0000-4000-8000-000000000001"

const (
	// pathWhole is timed by the bench client, not inside the server.
	pathWhole = "whole_intake"
	// pathHealth has no server endpoint yet. It stays in the budget file and
	// fails the gate as missing samples rather than being dropped from it.
	pathHealth = "health_speed"
)

type options struct {
	manifest    string
	budgets     string
	data        string
	recordings  string
	samplesOut  string
	out         string
	target      string
	live        bool
	rounds      int
	concurrency int
	background  int
	degradePct  int
	apiSamples  int
	maxFiles    int
	corpusRoot  string
	keysRoot    string
}

func main() {
	os.Exit(run(os.Args[1:], os.Getenv, os.Stdout, os.Stderr))
}

func run(args []string, getenv func(string) string, stdout, stderr io.Writer) int {
	var o options
	fs := flag.NewFlagSet("intake-bench", flag.ContinueOnError)
	fs.SetOutput(stderr)
	fs.StringVar(&o.manifest, "manifest", "data/eval/intake/manifest.json", "evaluation manifest naming the corpus")
	fs.StringVar(&o.budgets, "budgets", "bench/budgets.json", "latency budgets")
	fs.StringVar(&o.data, "data", "data/intake", "intake vocabulary and thresholds directory")
	fs.StringVar(&o.recordings, "recordings", "", "recorded Jev exchanges (default <private_dir>/recordings.jsonl)")
	fs.StringVar(&o.samplesOut, "samples-out", "bench/samples.json", "microsecond samples, readable by `sitewise gate`")
	fs.StringVar(&o.out, "out", "bench/results/latest.json", "results metadata")
	fs.StringVar(&o.target, "target", "dev", "name of the host under test; release evidence needs the intended VPS")
	fs.BoolVar(&o.live, "live", false, "call System One (SITEWISE_JEV_API_KEY) instead of replaying recorded latency")
	fs.IntVar(&o.rounds, "rounds", 2, "fresh projects, each receiving every corpus file; the first is the cold round")
	fs.IntVar(&o.concurrency, "concurrency", 4, "parallel uploads")
	fs.IntVar(&o.background, "background", 2, "concurrent background Jev callers during the uploads")
	fs.IntVar(&o.degradePct, "degrade", 10, "percent of Jev requests stalled past the deadline")
	fs.IntVar(&o.apiSamples, "api-samples", 40, "samples per API path")
	fs.IntVar(&o.maxFiles, "files", 0, "limit corpus files per round (0 is all)")
	fs.StringVar(&o.corpusRoot, "corpus-root", "", "override the manifest corpus_root")
	fs.StringVar(&o.keysRoot, "keys-root", "", "override the manifest keys_root")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	if o.rounds < 1 || o.concurrency < 1 || o.background < 0 || o.degradePct < 0 || o.degradePct > 50 || o.apiSamples < 1 {
		fmt.Fprintln(stderr, "rounds, concurrency and api-samples must be positive; background >= 0; degrade 0-50")
		return 2
	}
	code, err := bench(context.Background(), o, getenv, stdout)
	if err != nil {
		fmt.Fprintln(stderr, "intake-bench:", err)
		return 1
	}
	return code
}

// Results is the committed bench metadata. Raw samples go to -samples-out.
type Results struct {
	SchemaVersion int                   `json:"schema_version"`
	GeneratedAt   time.Time             `json:"generated_at"`
	Target        string                `json:"target"`
	Host          Host                  `json:"host"`
	Jev           string                `json:"jev"`
	Model         string                `json:"model"`
	Workload      Workload              `json:"workload"`
	Outcomes      map[string]int        `json:"outcomes"`
	Paths         map[string]PathResult `json:"paths"`
	WholeByRound  map[string]PathResult `json:"whole_intake_by_round"`
	Missing       []string              `json:"missing_paths"`
	Gate          GateVerdict           `json:"gate"`
	Notes         []string              `json:"notes"`
}

// Host is the hardware the samples came from.
type Host struct {
	OS         string `json:"os"`
	Arch       string `json:"arch"`
	CPUs       int    `json:"cpus"`
	GOMAXPROCS int    `json:"gomaxprocs"`
	CPU        string `json:"cpu,omitempty"`
	Go         string `json:"go"`
}

// Workload is what ran while the samples were taken.
type Workload struct {
	Files             int    `json:"files_per_round"`
	Rounds            int    `json:"rounds"`
	Concurrency       int    `json:"concurrency"`
	BackgroundCallers int    `json:"background_jev_callers"`
	BackgroundCalls   int64  `json:"background_jev_calls"`
	DegradePercent    int    `json:"degrade_percent"`
	StalledRequests   int64  `json:"stalled_jev_requests"`
	ColdRound         string `json:"cold_round"`
}

// PathResult is one path's measured percentiles against its budget.
type PathResult struct {
	N     int   `json:"n"`
	P50US int64 `json:"p50_us"`
	P90US int64 `json:"p90_us"`
	MaxUS int64 `json:"max_us"`
	Pass  *bool `json:"pass,omitempty"`
}

// GateVerdict is latency.Gate's exit code and report.
type GateVerdict struct {
	Code   int    `json:"code"`
	Report string `json:"report"`
}

func bench(ctx context.Context, o options, getenv func(string) string, stdout io.Writer) (int, error) {
	m, _, err := eval.LoadManifest(o.manifest)
	if err != nil {
		return 0, err
	}
	if o.corpusRoot != "" {
		m.CorpusRoot = o.corpusRoot
	}
	if o.keysRoot != "" {
		m.KeysRoot = o.keysRoot
	}
	if o.recordings == "" {
		o.recordings = filepath.Join(m.PrivateDir, "recordings.jsonl")
	}
	budgets, err := latency.LoadBudgets(o.budgets)
	if err != nil {
		return 0, err
	}
	set, err := eval.BuildCases(m)
	if err != nil {
		return 0, fmt.Errorf("corpus failed verification:\n%w", err)
	}
	cases := set.Cases
	if o.maxFiles > 0 && o.maxFiles < len(cases) {
		cases = cases[:o.maxFiles]
	}

	dsn, err := benchDSN(getenv)
	if err != nil {
		return 0, err
	}
	st, err := store.Open(ctx, dsn)
	if err != nil {
		return 0, err
	}
	defer st.Close()
	if err := st.Migrate(ctx); err != nil {
		return 0, err
	}
	if err := st.DeleteOrg(ctx, benchOrg); err != nil {
		return 0, err
	}
	if err := st.CreateOrg(ctx, benchOrg, "Intake bench"); err != nil {
		return 0, err
	}
	defer st.DeleteOrg(context.Background(), benchOrg)

	blobDir, err := os.MkdirTemp("", "sitewise-bench-")
	if err != nil {
		return 0, err
	}
	defer os.RemoveAll(blobDir)
	blobs, err := files.Open(blobDir, 200<<20)
	if err != nil {
		return 0, err
	}
	cat, err := intake.LoadCatalog(o.data)
	if err != nil {
		return 0, err
	}
	thresholds, err := intake.LoadThresholds(o.data)
	if err != nil {
		return 0, err
	}
	if err := identity.Warm(ctx); err != nil {
		return 0, err
	}

	client, transport, replayer, records, err := jevClient(o, getenv)
	if err != nil {
		return 0, err
	}
	if o.live {
		if err := client.Warm(ctx); err != nil {
			return 0, fmt.Errorf("jev warm-up: %w", err)
		}
	}

	samples := newCollector()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return 0, err
	}
	origin := "http://" + ln.Addr().String()
	srv, err := httpapi.New(httpapi.Options{
		Store:          st,
		Blobs:          blobs,
		Jev:            client,
		Catalog:        cat,
		Thresholds:     thresholds,
		Static:         web.Dist(),
		PublicOrigin:   origin,
		MaxUploadBytes: 200 << 20,
		Log:            log.New(io.Discard, "", 0),
		Observe:        samples.Add,
	})
	if err != nil {
		return 0, err
	}
	hs := &http.Server{Handler: srv, ReadHeaderTimeout: 10 * time.Second}
	go hs.Serve(ln)
	defer func() {
		srv.CloseStreams()
		shut, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = hs.Shutdown(shut)
	}()

	api := &apiClient{origin: origin, samples: samples}
	if err := api.signIn(ctx, st, o.apiSamples); err != nil {
		return 0, err
	}
	arrived := newArrivals()
	stream, err := api.stream(ctx, 0, arrived.record)
	if err != nil {
		return 0, err
	}
	defer stream()

	stopBackground := background(client, records, o.background)
	outcomes := map[string]int{}
	byRound := map[string]*collector{}
	var projects []string
	var documents []string
	for round := 1; round <= o.rounds; round++ {
		project, err := api.createProject(ctx, fmt.Sprintf("Bench round %d", round))
		if err != nil {
			stopBackground()
			return 0, err
		}
		projects = append(projects, project)
		rc := newCollector()
		byRound[fmt.Sprint(round)] = rc
		docs, err := api.dropAll(ctx, project, cases, o.concurrency, arrived, samples, rc, outcomes)
		if err != nil {
			stopBackground()
			return 0, err
		}
		documents = append(documents, docs...)
	}
	bgCalls := stopBackground()

	if err := api.listDocuments(ctx, projects, o.apiSamples); err != nil {
		return 0, err
	}
	if err := api.correct(ctx, documents, o.apiSamples); err != nil {
		return 0, err
	}
	if err := api.reconnect(ctx, arrived.cursor(), o.apiSamples); err != nil {
		return 0, err
	}
	if replayer != nil {
		if n := replayer.misses.Load(); n > 0 {
			return 0, fmt.Errorf("%d Jev requests had no recorded document; re-record with intake-eval -live", n)
		}
	}
	grey, err := countGrey(ctx, st, projects)
	if err != nil {
		return 0, err
	}
	outcomes["grey_filings"] = grey

	snapshot := samples.Snapshot()
	if err := writeJSON(o.samplesOut, snapshot); err != nil {
		return 0, err
	}
	code, report := latency.Gate(budgets, snapshot)
	res := Results{
		SchemaVersion: 1,
		GeneratedAt:   time.Now().UTC(),
		Target:        o.target,
		Host:          host(),
		Jev:           jevMode(o.live),
		Model:         config.PinnedJevModel,
		Workload: Workload{
			Files:             len(cases),
			Rounds:            o.rounds,
			Concurrency:       o.concurrency,
			BackgroundCallers: o.background,
			BackgroundCalls:   bgCalls,
			DegradePercent:    o.degradePct,
			StalledRequests:   transport.stall.Load(),
			ColdRound:         "1",
		},
		Outcomes:     outcomes,
		Paths:        map[string]PathResult{},
		WholeByRound: map[string]PathResult{},
		Gate:         GateVerdict{Code: code, Report: report},
		Notes: []string{
			"whole_intake runs from the final upload byte read by the client transport to the filing's terminal event on an open SSE stream, over loopback.",
			"Stage paths are timed inside the same filings by the intake Observer and include failed and degraded runs.",
			"Round 1 is the first filing of each file in a process with PDFium and the Jev connection already warmed, as serve does at startup.",
			pathHealth + " has no endpoint until Task 12, so it has no samples and fails the gate as missing.",
		},
	}
	if !o.live {
		res.Notes = append(res.Notes, "Jev latency is the recorded provider latency replayed per document, not a live measurement. Release evidence is a -live run on the intended VPS.")
	}
	for _, b := range budgets.Paths {
		got := snapshot[b.Name]
		if len(got) == 0 {
			res.Missing = append(res.Missing, b.Name)
			continue
		}
		pr := summarize(got)
		pass := pr.N >= budgets.MinSamples && pr.P50US <= b.P50US && pr.P90US <= b.P90US
		pr.Pass = &pass
		res.Paths[b.Name] = pr
	}
	for round, rc := range byRound {
		if got := rc.Snapshot()[pathWhole]; len(got) > 0 {
			res.WholeByRound[round] = summarize(got)
		}
	}
	if err := writeJSON(o.out, res); err != nil {
		return 0, err
	}
	printResults(stdout, budgets, res)
	return code, nil
}

func jevMode(live bool) string {
	if live {
		return "live"
	}
	return "replayed-recorded-latency"
}

// benchDSN refuses any database but the dedicated test or bench database:
// the bench deletes and recreates its org.
func benchDSN(getenv func(string) string) (string, error) {
	dsn := getenv("SITEWISE_BENCH_DATABASE_URL")
	if dsn == "" {
		dsn = getenv("SITEWISE_TEST_DATABASE_URL")
	}
	u, err := url.Parse(dsn)
	if dsn == "" || err != nil {
		return "", errors.New("SITEWISE_BENCH_DATABASE_URL or SITEWISE_TEST_DATABASE_URL is required")
	}
	switch strings.TrimPrefix(u.Path, "/") {
	case "sitewise_test", "sitewise_bench":
		return dsn, nil
	default:
		return "", errors.New("the bench database must be sitewise_test or sitewise_bench")
	}
}

func jevClient(o options, getenv func(string) string) (*jev.Client, *degrading, *stateReplayer, []eval.Record, error) {
	every := int64(0)
	if o.degradePct > 0 {
		every = int64(100 / o.degradePct)
	}
	records, err := readRecordings(o.recordings)
	if err != nil && !o.live {
		return nil, nil, nil, nil, err
	}
	var (
		next     http.RoundTripper
		key      = "replay"
		replayer *stateReplayer
	)
	if o.live {
		key = strings.TrimSpace(getenv("SITEWISE_JEV_API_KEY"))
		if key == "" {
			return nil, nil, nil, nil, errors.New("SITEWISE_JEV_API_KEY is required for -live")
		}
		next = jev.NewTransport()
	} else {
		replayer, err = newStateReplayer(records)
		if err != nil {
			return nil, nil, nil, nil, err
		}
		next = replayer
	}
	d := &degrading{next: next, every: every}
	client, err := jev.New(jev.Options{
		APIKey:    key,
		Model:     config.PinnedJevModel,
		Transport: d,
		Logger:    slog.New(slog.DiscardHandler),
	})
	if err != nil {
		return nil, nil, nil, nil, err
	}
	return client, d, replayer, records, nil
}

func readRecordings(file string) ([]eval.Record, error) {
	f, err := os.Open(file)
	if errors.Is(err, os.ErrNotExist) {
		return nil, fmt.Errorf("no recordings at %s: run intake-eval -live first, or bench with -live", file)
	}
	if err != nil {
		return nil, err
	}
	defer f.Close()
	return eval.ReadRecordings(f)
}

// background keeps callers asking recorded questions at background priority
// until stopped, so foreground filings compete with labeling work for slots.
// stop returns how many background calls were made.
func background(client *jev.Client, records []eval.Record, callers int) (stop func() int64) {
	var calls []jev.Call
	for _, rec := range records {
		if call, ok := backgroundCall(rec); ok {
			calls = append(calls, call)
		}
	}
	var n atomic.Int64
	ctx, cancel := context.WithCancel(context.Background())
	var wg sync.WaitGroup
	if len(calls) > 0 {
		for i := 0; i < callers; i++ {
			wg.Add(1)
			go func(i int) {
				defer wg.Done()
				for j := i; ctx.Err() == nil; j += callers {
					_, _ = client.Ask(ctx, calls[j%len(calls)])
					n.Add(1)
				}
			}(i)
		}
	}
	return func() int64 {
		cancel()
		wg.Wait()
		return n.Load()
	}
}

func backgroundCall(rec eval.Record) (jev.Call, bool) {
	var req struct {
		State     json.RawMessage               `json:"state"`
		Questions map[string]backgroundQuestion `json:"questions"`
	}
	if err := json.Unmarshal([]byte(rec.Request), &req); err != nil || len(req.Questions) == 0 {
		return jev.Call{}, false
	}
	qs := make(map[string]jev.Question, len(req.Questions))
	for id, q := range req.Questions {
		qs[id] = jev.Question{Type: q.Type, Instructions: q.Instructions, Criteria: q.Criteria}
	}
	return jev.Call{State: req.State, Questions: qs, Priority: jev.PriorityBackground, QuestionVersion: intake.QuestionVersion}, true
}

type backgroundQuestion struct {
	Type         string          `json:"type"`
	Instructions json.RawMessage `json:"instructions"`
	Criteria     json.RawMessage `json:"criteria"`
}

// countGrey counts filings with any grey field: the degraded outcomes.
func countGrey(ctx context.Context, st *store.Store, projects []string) (int, error) {
	n := 0
	for _, p := range projects {
		list, err := st.ProjectDocumentViews(ctx, benchOrg, p)
		if err != nil {
			return 0, err
		}
		for _, d := range list {
			for _, f := range d.Fields {
				if f.Band == intake.BandGrey {
					n++
					break
				}
			}
		}
	}
	return n, nil
}

func summarize(us []int64) PathResult {
	p50, _ := latency.Percentile(us, 0.5)
	p90, _ := latency.Percentile(us, 0.9)
	var max int64
	for _, v := range us {
		if v > max {
			max = v
		}
	}
	return PathResult{N: len(us), P50US: p50, P90US: p90, MaxUS: max}
}

func host() Host {
	h := Host{OS: runtime.GOOS, Arch: runtime.GOARCH, CPUs: runtime.NumCPU(), GOMAXPROCS: runtime.GOMAXPROCS(0), Go: runtime.Version()}
	if id := os.Getenv("PROCESSOR_IDENTIFIER"); id != "" {
		h.CPU = id
	} else if body, err := os.ReadFile("/proc/cpuinfo"); err == nil {
		for _, line := range strings.Split(string(body), "\n") {
			if strings.HasPrefix(line, "model name") {
				if _, v, ok := strings.Cut(line, ":"); ok {
					h.CPU = strings.TrimSpace(v)
					break
				}
			}
		}
	}
	return h
}

func printResults(w io.Writer, budgets latency.Budgets, r Results) {
	fmt.Fprintf(w, "target %s, jev %s, %d files x %d rounds, concurrency %d, background %d callers (%d calls), degrade %d%% (%d stalled)\n",
		r.Target, r.Jev, r.Workload.Files, r.Workload.Rounds, r.Workload.Concurrency, r.Workload.BackgroundCallers, r.Workload.BackgroundCalls, r.Workload.DegradePercent, r.Workload.StalledRequests)
	fmt.Fprintf(w, "outcomes %v\n", r.Outcomes)
	for _, b := range budgets.Paths {
		p, ok := r.Paths[b.Name]
		if !ok {
			fmt.Fprintf(w, "  %-28s no samples\n", b.Name)
			continue
		}
		verdict := "FAIL"
		if p.Pass != nil && *p.Pass {
			verdict = "ok"
		}
		fmt.Fprintf(w, "  %-28s n=%-4d p50 %7.1f ms (budget %6.1f)  p90 %7.1f ms (budget %6.1f)  %s\n",
			b.Name, p.N, ms(p.P50US), ms(b.P50US), ms(p.P90US), ms(b.P90US), verdict)
	}
	for round, p := range r.WholeByRound {
		fmt.Fprintf(w, "  whole_intake round %s: n=%d p50 %.1f ms p90 %.1f ms\n", round, p.N, ms(p.P50US), ms(p.P90US))
	}
	if r.Gate.Code != 0 {
		fmt.Fprintf(w, "gate failed (code %d):\n%s", r.Gate.Code, r.Gate.Report)
		return
	}
	fmt.Fprintln(w, "gate passed")
}

func ms(us int64) float64 { return float64(us) / 1000 }

func writeJSON(file string, v any) error {
	if err := os.MkdirAll(filepath.Dir(file), 0o755); err != nil {
		return err
	}
	body, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(file, append(body, '\n'), 0o644)
}

// apiClient is one signed-in browser session against the bench server.
type apiClient struct {
	origin  string
	http    *http.Client
	samples *collector
}

func (a *apiClient) do(ctx context.Context, method, path string, body io.Reader, size int64) (*http.Response, error) {
	req, err := http.NewRequestWithContext(ctx, method, a.origin+"/api"+path, body)
	if err != nil {
		return nil, err
	}
	if body != nil {
		req.ContentLength = size
	}
	req.Header.Set("Origin", a.origin)
	if body != nil && !strings.Contains(path, "/files?") {
		req.Header.Set("Content-Type", "application/json")
	}
	return a.http.Do(req)
}

func (a *apiClient) timed(ctx context.Context, path, method, url string, body []byte, want int) ([]byte, error) {
	var r io.Reader
	if body != nil {
		r = bytes.NewReader(body)
	}
	start := time.Now()
	resp, err := a.do(ctx, method, url, r, int64(len(body)))
	if err != nil {
		return nil, err
	}
	out, err := io.ReadAll(resp.Body)
	resp.Body.Close()
	a.samples.Add(path, time.Since(start))
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != want {
		return nil, fmt.Errorf("%s %s: status %d: %s", method, url, resp.StatusCode, bytes.TrimSpace(out))
	}
	return out, nil
}

// signIn consumes n invitations, timing each token consumption, and keeps
// the last session.
func (a *apiClient) signIn(ctx context.Context, st *store.Store, n int) error {
	for i := 0; i < n; i++ {
		raw, err := auth.CreateInvite(ctx, st, &auth.Sink{}, benchOrg, fmt.Sprintf("bench-%d@bench.test", i), "member", time.Now().Add(time.Hour), nil)
		if err != nil {
			return err
		}
		jar, err := cookiejar.New(nil)
		if err != nil {
			return err
		}
		a.http = &http.Client{Jar: jar, Timeout: 60 * time.Second}
		body, _ := json.Marshal(map[string]string{"token": raw})
		if _, err := a.timed(ctx, "project_invite_auth", http.MethodPost, "/session", body, http.StatusNoContent); err != nil {
			return err
		}
	}
	return nil
}

func (a *apiClient) createProject(ctx context.Context, name string) (string, error) {
	body, _ := json.Marshal(map[string]string{"name": name})
	out, err := a.timed(ctx, "project_invite_auth", http.MethodPost, "/projects", body, http.StatusCreated)
	if err != nil {
		return "", err
	}
	var got struct {
		ID string `json:"id"`
	}
	if err := json.Unmarshal(out, &got); err != nil || got.ID == "" {
		return "", fmt.Errorf("create project: %s", out)
	}
	return got.ID, nil
}

// stream opens the org event stream and feeds every event to on. The
// returned func closes it.
func (a *apiClient) stream(ctx context.Context, after int64, on func(event, time.Time)) (func(), error) {
	ctx, cancel := context.WithCancel(ctx)
	resp, err := a.do(ctx, http.MethodGet, fmt.Sprintf("/events?after=%d", after), nil, 0)
	if err != nil {
		cancel()
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		resp.Body.Close()
		cancel()
		return nil, fmt.Errorf("event stream status %d", resp.StatusCode)
	}
	done := make(chan struct{})
	go func() {
		defer close(done)
		_ = readEvents(resp.Body, on)
	}()
	return func() {
		cancel()
		resp.Body.Close()
		<-done
	}, nil
}

// dropAll uploads every case into project with bounded concurrency and
// records each filing's whole-intake time.
func (a *apiClient) dropAll(ctx context.Context, project string, cases []eval.Case, concurrency int, arrived *arrivals, all, round *collector, outcomes map[string]int) ([]string, error) {
	type result struct {
		doc  string
		kind string
		err  error
	}
	jobs := make(chan eval.Case)
	results := make(chan result)
	var wg sync.WaitGroup
	for i := 0; i < concurrency; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for c := range jobs {
				doc, kind, err := a.drop(ctx, project, c, arrived, all, round)
				results <- result{doc, kind, err}
			}
		}()
	}
	go func() {
		for _, c := range cases {
			jobs <- c
		}
		close(jobs)
		wg.Wait()
		close(results)
	}()
	var docs []string
	var first error
	for r := range results {
		if r.err != nil {
			if first == nil {
				first = r.err
			}
			continue
		}
		outcomes[r.kind]++
		docs = append(docs, r.doc)
	}
	return docs, first
}

func (a *apiClient) drop(ctx context.Context, project string, c eval.Case, arrived *arrivals, all, round *collector) (string, string, error) {
	body, err := os.ReadFile(c.Path)
	if err != nil {
		return "", "", err
	}
	lb := newLastByte(body)
	resp, err := a.do(ctx, http.MethodPost, "/projects/"+project+"/files?name="+url.QueryEscape(c.Filename()), lb, int64(len(body)))
	if err != nil {
		return "", "", err
	}
	out, err := io.ReadAll(resp.Body)
	resp.Body.Close()
	if err != nil {
		return "", "", err
	}
	if resp.StatusCode != http.StatusCreated {
		return "", "", fmt.Errorf("upload %s: status %d: %s", c.ID, resp.StatusCode, bytes.TrimSpace(out))
	}
	var view struct {
		ID string `json:"id"`
	}
	if err := json.Unmarshal(out, &view); err != nil || view.ID == "" {
		return "", "", fmt.Errorf("upload %s: %s", c.ID, out)
	}
	wait, cancel := context.WithTimeout(ctx, 60*time.Second)
	defer cancel()
	at, kind, err := arrived.await(wait, view.ID)
	if err != nil {
		return "", "", fmt.Errorf("upload %s: no filing event: %w", c.ID, err)
	}
	select {
	case <-lb.done:
	default:
		return "", "", fmt.Errorf("upload %s: final byte not observed", c.ID)
	}
	d := at.Sub(lb.at)
	all.Add(pathWhole, d)
	round.Add(pathWhole, d)
	return view.ID, kind, nil
}

func (a *apiClient) listDocuments(ctx context.Context, projects []string, n int) error {
	for i := 0; i < n; i++ {
		p := projects[i%len(projects)]
		if _, err := a.timed(ctx, "project_document_list", http.MethodGet, "/projects/"+p+"/documents", nil, http.StatusOK); err != nil {
			return err
		}
	}
	return nil
}

// correct writes n user corrections to the number field.
func (a *apiClient) correct(ctx context.Context, documents []string, n int) error {
	if len(documents) == 0 {
		return errors.New("no filed documents to correct")
	}
	for i := 0; i < n; i++ {
		doc := documents[i%len(documents)]
		body, _ := json.Marshal(map[string]string{"value": fmt.Sprintf("BENCH-%d", i)})
		if _, err := a.timed(ctx, "field_correction", http.MethodPut, "/documents/"+doc+"/fields/"+intake.FieldNumber, body, http.StatusOK); err != nil {
			return err
		}
	}
	return nil
}

// reconnect opens n streams a few events behind the latest cursor and times
// the first caught-up event.
func (a *apiClient) reconnect(ctx context.Context, cursor int64, n int) error {
	after := cursor - 5
	if after < 0 {
		after = 0
	}
	for i := 0; i < n; i++ {
		got := make(chan struct{}, 1)
		start := time.Now()
		closeStream, err := a.stream(ctx, after, func(event, time.Time) {
			select {
			case got <- struct{}{}:
			default:
			}
		})
		if err != nil {
			return err
		}
		select {
		case <-got:
			a.samples.Add("sse_reconnect", time.Since(start))
		case <-time.After(10 * time.Second):
			closeStream()
			return errors.New("reconnect: no catch-up event")
		}
		closeStream()
	}
	return nil
}

// Command e2eserver runs the real server against the sitewise_test database
// with a recorded Jev, for the Playwright intake flow. It is never deployed.
//
// Recorded answers pick each question's first option. Test-only thresholds
// make kind green, discipline amber and lifecycle blank so every band shows;
// a filename starting "slow-" makes Jev miss its deadline so fields go grey.
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
	"net/http/httptest"
	"net/url"
	"os"
	"os/signal"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/jackc/pgx/v5"

	"sitewise/internal/auth"
	"sitewise/internal/chatfix"
	"sitewise/internal/config"
	"sitewise/internal/files"
	"sitewise/internal/httpapi"
	"sitewise/internal/identity"
	"sitewise/internal/intake"
	"sitewise/internal/invoices"
	"sitewise/internal/invoices/sample"
	"sitewise/internal/jev"
	"sitewise/internal/jobs"
	"sitewise/internal/knowledge"
	"sitewise/internal/profile"
	"sitewise/internal/proposals"
	sample2 "sitewise/internal/proposals/sample"
	"sitewise/internal/store"
	tendersample "sitewise/internal/tender/sample"
	"sitewise/internal/works"
	"sitewise/internal/xlsx"
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
	library := flag.String("program-library", "fixture", `program library: "fixture" (a temp copy of knowledge/ with the test fragment and duration below) or "repo" (knowledge/ as committed)`)
	flag.Parse()
	if *library != "fixture" && *library != "repo" {
		log.Fatal(`-program-library is "fixture" or "repo"`)
	}
	if err := run(*addr, *data, *library == "fixture"); err != nil {
		log.Fatal(err)
	}
}

func run(addr, data string, programFixture bool) error {
	host, _, err := net.SplitHostPort(addr)
	if err != nil || net.ParseIP(host) == nil || !net.ParseIP(host).IsLoopback() {
		return errors.New("e2e server must bind a loopback address")
	}
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
	// The profile panel reads the real knowledge with provisional floors.
	// The program library is the committed one, or with programFixture a
	// temp copy holding one test fragment and one reviewed test duration, so
	// the browser suite has a question to answer and work-item changes to
	// review (plan 11.5 steps 3 and 13). The repo's knowledge/ is never edited.
	root := "knowledge"
	if programFixture {
		if root, err = programLibrary(dir); err != nil {
			return err
		}
		// The fixture has work-item rows only, which the outline leaves out;
		// the editing suite needs activities to edit.
		store.SetProgramOutline(false)
	}
	building, err := knowledge.Load(root)
	if err != nil {
		return err
	}
	profileTh, err := profile.LoadThresholds("data/profile/thresholds.json")
	if err != nil {
		return err
	}
	reading, err := profile.LoadReadPolicy("data/profile/reading.json")
	if err != nil {
		return err
	}
	chat, err := e2eChat()
	if err != nil {
		return err
	}
	srv, err := httpapi.New(httpapi.Options{
		// Chat is answered by a stand-in that reads the PM's words only.
		Chat:              chat,
		Knowledge:         building,
		ProfileThresholds: profileTh,
		ProfileReading:    reading,
		Store:             st,
		Blobs:             blobs,
		Jev:               client,
		Catalog:           cat,
		Thresholds:        thresholds(cat),
		InvoiceThresholds: invoiceThresholds(),
		// Proposals are read by a stand-in that takes the amount printed
		// beside each row's own label, with every band applied.
		ProposalJev:        sample2.Jev{Delay: 150 * time.Millisecond},
		ProposalThresholds: proposalThresholds(),
		// The tender pass is answered by a stand-in that lists each line of
		// page text ending in a dollar amount. No page leaves this process.
		TenderLLM:   tendersample.Model{Delay: 150 * time.Millisecond},
		TenderModel: "claude-haiku-5-5",
		TenderRender: func(_ context.Context, _ string, pages []int) (map[int]string, error) {
			out := map[int]string{}
			for _, p := range pages {
				out[p] = "/9j/4AAQSkZJRg=="
			}
			return out, nil
		},
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
	mux.HandleFunc("POST /__e2e/shutdown", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
		stop()
	})
	mux.HandleFunc("POST /__e2e/invite", func(w http.ResponseWriter, r *http.Request) {
		invite(w, r, st)
	})
	// Synthetic tax invoices for the ledger suite; no real supplier.
	mux.HandleFunc("GET /__e2e/invoice.pdf", func(w http.ResponseWriter, r *http.Request) {
		n, err := strconv.Atoi(r.URL.Query().Get("n"))
		if err != nil || n < 0 || n > 100000 {
			http.Error(w, "n is required", http.StatusBadRequest)
			return
		}
		w.Header().Set("Content-Type", "application/pdf")
		_, _ = w.Write(sample.PDF(sample.Batch(n, 1)[0]))
	})
	// Synthetic fee proposals: ?firm=Name&abn=n&lines=Label~cents,Label~-1&rates=Role~cents&extra=Label~cents
	mux.HandleFunc("GET /__e2e/proposal.pdf", func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()
		n, _ := strconv.Atoi(q.Get("abn"))
		p := sample2.Proposal{Firm: q.Get("firm"), ABN: sample.ABNFor(n), Email: "office@" + strings.ToLower(strings.Fields(q.Get("firm") + " firm")[0]) + ".test"}
		parse := func(v string) []sample2.Line {
			var out []sample2.Line
			for _, part := range strings.Split(v, ",") {
				label, cents, ok := strings.Cut(part, "~")
				if !ok {
					continue
				}
				c, _ := strconv.ParseInt(cents, 10, 64)
				out = append(out, sample2.Line{Label: label, Cents: c})
			}
			return out
		}
		p.Lines, p.Rates = parse(q.Get("lines")), parse(q.Get("rates"))
		if e := parse(q.Get("extra")); len(e) == 1 {
			p.Extra = &e[0]
		}
		w.Header().Set("Content-Type", "application/pdf")
		_, _ = w.Write(sample2.PDF(p))
	})
	// Synthetic program files for the readings suite:
	// ?rows=Label~start~finish,Label~start~finish with dates as typed text,
	// and an optional fourth field, the predecessors cell ("1", "1SS+2d";
	// rows number from 1, as program tools do without an ID column; a
	// semicolon separates several). The column is written only when a row
	// has one.
	mux.HandleFunc("GET /__e2e/program.xlsx", func(w http.ResponseWriter, r *http.Request) {
		txt := func(s string) xlsx.Cell { return xlsx.Cell{Text: s} }
		var data [][]string
		withPred := false
		for _, part := range strings.Split(r.URL.Query().Get("rows"), ",") {
			f := strings.Split(part, "~")
			if (len(f) != 3 && len(f) != 4) || strings.TrimSpace(f[0]) == "" {
				continue
			}
			withPred = withPred || len(f) == 4
			data = append(data, f)
		}
		head := []xlsx.Cell{txt("Task Name"), txt("Start"), txt("Finish")}
		if withPred {
			head = append(head, txt("Predecessors"))
		}
		rows := [][]xlsx.Cell{head}
		for _, f := range data {
			row := []xlsx.Cell{txt(f[0]), txt(f[1]), txt(f[2])}
			if withPred {
				pred := ""
				if len(f) == 4 {
					pred = f[3]
				}
				row = append(row, txt(pred))
			}
			rows = append(rows, row)
		}
		w.Header().Set("Content-Type", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
		_ = xlsx.WriteXLSX(w, []xlsx.Sheet{{Name: "Program", Rows: rows}})
	})
	// Marks an RFP or RFT draft issued, as the issue route leaves it, with a
	// synthetic PDF, so a guest link has an issue to share. The RFP issue
	// route's clause gates are not what the guest suite is about.
	mux.HandleFunc("POST /__e2e/issue-rft", func(w http.ResponseWriter, r *http.Request) {
		blob, err := blobs.Put(r.Context(), bytes.NewReader(sample2.PDF(sample2.Proposal{Firm: "Request for tender"})))
		if err != nil {
			http.Error(w, err.Error(), 500)
			return
		}
		conn, err := pgx.Connect(r.Context(), dsn)
		if err != nil {
			http.Error(w, err.Error(), 500)
			return
		}
		defer conn.Close(r.Context())
		report := r.URL.Query().Get("report")
		tag, err := conn.Exec(r.Context(), `UPDATE report_versions v SET status='issued', snapshot='{}'::jsonb, snapshot_sha256=repeat('a',64), export_file_sha256=$2,
 issued_at=now(), issued_by=(SELECT u.id FROM users u WHERE u.org_id=v.org_id ORDER BY u.created_at LIMIT 1)
 FROM reports r WHERE r.org_id=v.org_id AND r.id=v.report_id AND r.id::text=$1 AND v.id=r.current_draft_version_id AND r.kind IN ('rfp','rft')`, report, blob.SHA256)
		if err != nil || tag.RowsAffected() != 1 {
			http.Error(w, fmt.Sprintf("no draft to issue: %v", err), 400)
			return
		}
		if _, err := conn.Exec(r.Context(), `UPDATE reports SET current_draft_version_id=NULL WHERE id::text=$1`, report); err != nil {
			http.Error(w, err.Error(), 500)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	})
	mux.HandleFunc("POST /__e2e/drop-streams", func(w http.ResponseWriter, r *http.Request) {
		cut.dropAll()
		w.WriteHeader(http.StatusNoContent)
	})
	// Only the dedicated, non-deployed browser test server exposes this explicit
	// diagnostic rebuild. Investigation uses the unchanged runtime catalogue so
	// automatic refresh is exercised through acceptance and undo as in production.
	mux.HandleFunc("POST /__e2e/proposals", func(w http.ResponseWriter, r *http.Request) {
		fixture, err := knowledge.Load("knowledge")
		if err != nil {
			http.Error(w, err.Error(), 500)
			return
		}
		kind := r.URL.Query().Get("kind")
		switch kind {
		case "investigation", "work_item", "discipline", "obligation", "approval", "hold_point":
		default:
			http.Error(w, "invalid fixture kind", 400)
			return
		}
		for i := range fixture.InterfaceConsequences() {
			record := &fixture.InterfaceConsequences()[i]
			if record.ID == "ic.loads-investigate-supported" && kind != "investigation" {
				record.Propose.Kind = kind
				record.Propose.Label = "Synthetic " + kind + " proposal"
			}
		}
		evaluator, err := works.NewEvaluator(fixture)
		if err != nil {
			http.Error(w, err.Error(), 500)
			return
		}
		build := store.ProfileBuild{Catalog: fixture, KnowledgeVersion: fixture.Version(), Compute: func(s store.ProfileSnapshot) []profile.Row {
			return profile.Build(profile.Input{Parts: s.Parts, Facts: s.Facts, User: s.User, Planning: s.Planning, Thresholds: profileTh, Read: reading}, fixture)
		}}
		if err := st.WithProfile(build).RebuildProposals(r.Context(), orgs["a"], r.URL.Query().Get("project"), evaluator); err != nil {
			http.Error(w, err.Error(), 500)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	})
	// A synthetic reply that cites an RFI, filed and matched as the
	// register's post-filing job would match it (Go's rule; no Jev).
	mux.HandleFunc("POST /__e2e/register-reply", func(w http.ResponseWriter, r *http.Request) {
		if err := registerReply(r.Context(), st, orgs["a"], r.URL.Query().Get("project"), r.URL.Query().Get("text")); err != nil {
			http.Error(w, err.Error(), 500)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	})
	// Consent conditions (plan 6): file a notice's text as a Statutory
	// Instrument, and run the queued consent reads with a stand-in for Jev.
	mux.HandleFunc("POST /__e2e/file-document", func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(io.LimitReader(r.Body, 1<<20))
		q := r.URL.Query()
		if _, err := chatfix.SeedDocuments(r.Context(), st, orgs["a"], q.Get("project"), []chatfix.Document{{Title: q.Get("title"), Kind: q.Get("kind"), Pages: []string{string(body)}}}); err != nil {
			http.Error(w, err.Error(), 500)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	})
	mux.HandleFunc("POST /__e2e/run-consent", func(w http.ResponseWriter, r *http.Request) {
		if err := runConsent(r.Context(), st, orgs["a"]); err != nil {
			http.Error(w, err.Error(), 500)
			return
		}
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

// Test fixture only: the fragment, its labels and every number below are
// invented for the browser suite and are not library content. Gas work on
// a new building brings two activities with a reviewed 3–5–8 day duration;
// with the work type unknown the fragment is conditional (low 0), so the
// strip asks for the work type and answering narrows the finish range.
const fixtureFragments = `version: 1
status: draft
sources: [{design: test-fixture-only, anchor: "# e2e"}]
fragments:
  - id: fr.e2e-gas
    label: Gas service
    stage: delivery
    systems: [hydraulic.gas]
    applies_when: {det: work_type, eq: new}
    activities:
      - {key: rough-in, label: Gas rough-in, systems: [hydraulic.gas], duration: du.e2e-gas}
      - {key: fit-off, label: Gas fit-off, systems: [hydraulic.gas], duration: du.e2e-gas}
    links:
      - {from: rough-in, to: fit-off, type: FS, lag: null}
    attach: {after: stage_start}
    status: draft
    sources: [{design: test-fixture-only, anchor: "# e2e"}]
`

const fixtureDurations = `version: 1
status: draft
sources: [{design: test-fixture-only, anchor: "# e2e"}]
durations:
  - id: du.e2e-gas
    version: 1
    basis: fixed
    unit_days: working
    low: "3"
    typical: "5"
    high: "8"
    geography: Test
    limitations: []
    status: reviewed
    sources: [{design: test-fixture-only, anchor: "# e2e"}]
`

// programLibrary copies knowledge/ under dir and replaces the program
// fragments and durations with the test fixture, returning the copy's root.
func programLibrary(dir string) (string, error) {
	root := filepath.Join(dir, "knowledge")
	if err := os.CopyFS(root, os.DirFS("knowledge")); err != nil {
		return "", err
	}
	for name, body := range map[string]string{"fragments.yaml": fixtureFragments, "durations.yaml": fixtureDurations} {
		if err := os.WriteFile(filepath.Join(root, "program", name), []byte(body), 0o600); err != nil {
			return "", err
		}
	}
	return root, nil
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
// invoiceThresholds make the recorded middling answers amber, so the
// ledger suite sees Jev's picks as values to check.
func invoiceThresholds() invoices.Thresholds {
	lo, hi := 2, 255
	g, a := 0.9, 0.4
	cut := []invoices.Threshold{{Options: &lo, MaxOptions: &hi, Green: &g, Amber: &a}}
	return invoices.Thresholds{QuestionVersion: invoices.QuestionVersion, Calibration: "e2e fixture thresholds; not calibrated",
		Questions: map[string][]invoices.Threshold{invoices.FieldABN: cut, invoices.FieldSupplier: cut, invoices.FieldLine: cut, invoices.FieldDate: cut}}
}

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
			Criteria map[string]json.RawMessage `json:"criteria"`
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

// proposalThresholds apply every proposal answer green, so the browser
// suite sees the read as it would with a calibrated question.
func proposalThresholds() proposals.Thresholds {
	lo, hi, g := 2, 255, 0.5
	th := proposals.Thresholds{QuestionVersion: proposals.QuestionVersion, Questions: map[string][]proposals.Threshold{}}
	for _, f := range []string{"price", "rate", "criterion", "firm", "kind", "package", "gst_basis", "stated_total"} {
		th.Questions[f] = []proposals.Threshold{{Options: &lo, MaxOptions: &hi, Green: &g, Amber: &g}}
	}
	return th
}

func registerReply(ctx context.Context, st *store.Store, org, project, text string) error {
	if _, err := chatfix.SeedDocuments(ctx, st, org, project, []chatfix.Document{{Title: "Engineer's reply", Kind: "drawing", Pages: []string{text}}}); err != nil {
		return err
	}
	docs, err := st.ProjectDocumentViews(ctx, org, project)
	if err != nil {
		return err
	}
	for _, d := range docs {
		if err := st.QueueRegister(ctx, org, d.ID); err != nil {
			return err
		}
	}
	for _, kind := range []string{store.JobKindFullText, store.JobKindRegister} {
		w := &jobs.Worker{Store: st}
		for {
			job, err := st.ClaimJobSince(ctx, org, time.Minute, []string{kind}, time.Time{})
			if errors.Is(err, store.ErrIdle) {
				break
			}
			if err != nil {
				return err
			}
			if kind == store.JobKindRegister {
				if err := w.Perform(ctx, job); err != nil {
					return err
				}
			}
			if err := st.CompleteJob(ctx, job.OrgID, job.ID, job.Token, store.EventWrite{Kind: "job", DocumentID: job.DocumentID, Payload: `{"kind":"` + kind + `","status":"done"}`}); err != nil {
				return err
			}
		}
	}
	return nil
}

// consentAsk is the browser suite's stand-in for Jev on a consent read: a
// condition that names the arborist is the arborist's; every other party
// is below the floor; evidence is needed; the system is the site.
type consentAsk struct{}

func (consentAsk) Ask(_ context.Context, call jev.Call) (jev.Result, error) {
	state, _ := call.State.(map[string]any)
	cond, _ := state["condition"].(map[string]any)
	text, _ := cond["text"].(string)
	conf := func(v float64) *float64 { return &v }
	party, p := "applicant", 0.3
	if strings.Contains(strings.ToLower(text), "arborist") {
		party, p = "consultant.arborist", 0.8
	}
	return jev.Result{Answers: map[string]jev.Answer{
		"party":    {Type: jev.TypeChoice, Choice: party, Confidence: conf(p)},
		"system":   {Type: jev.TypeChoice, Choice: "site", Confidence: conf(0.8)},
		"evidence": {Type: jev.TypeNoul, Noul: 0.9},
		"timing":   {Type: jev.TypeChoice, Choice: "before_cc", Confidence: conf(0.8)},
	}}, nil
}

func runConsent(ctx context.Context, st *store.Store, org string) error {
	for _, kind := range []string{store.JobKindFullText, store.JobKindConsent} {
		w := &jobs.Worker{Store: st, Ask: consentAsk{}}
		for {
			job, err := st.ClaimJobSince(ctx, org, time.Minute, []string{kind}, time.Time{})
			if errors.Is(err, store.ErrIdle) {
				break
			}
			if err != nil {
				return err
			}
			if kind == store.JobKindConsent {
				if err := w.Perform(ctx, job); err != nil {
					return err
				}
			}
			if err := st.CompleteJob(ctx, job.OrgID, job.ID, job.Token, store.EventWrite{Kind: "job", DocumentID: job.DocumentID, Payload: `{"kind":"` + kind + `","status":"done"}`}); err != nil {
				return err
			}
		}
	}
	return nil
}

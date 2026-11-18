package jobs_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"sitewise/internal/delivery"
	"sitewise/internal/jev"
	"sitewise/internal/jobs"
	"sitewise/internal/store"
)

// The register's reply match (docs/plans/2026-11-17-rfis-review-comments.md,
// Task 4), with a stand-in for Jev: no live call is made.

type registerAsk struct {
	mu    sync.Mutex
	calls []jev.Call
	yes   map[string]float64 // key → probability; others unresolved
	fail  error
}

func (a *registerAsk) Ask(_ context.Context, call jev.Call) (jev.Result, error) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.calls = append(a.calls, call)
	if a.fail != nil {
		return jev.Result{}, a.fail
	}
	out := jev.Result{Answers: map[string]jev.Answer{}}
	for key := range call.Questions {
		if p, ok := a.yes[key]; ok {
			out.Answers[key] = jev.Answer{Type: jev.TypeNoul, Noul: p}
		} else {
			out.Unresolved = append(out.Unresolved, key)
		}
	}
	return out, nil
}

type registerRun struct {
	st           *store.Store
	org, project string
	rfis         map[int]string // number → id
}

func registerProject(t *testing.T, open int) registerRun {
	t.Helper()
	st := openStore(t)
	ctx := context.Background()
	r := registerRun{st: st, org: newID(t), project: newID(t), rfis: map[int]string{}}
	seedOrg(t, st, r.org, r.project)
	user := newID(t)
	if err := st.CreateUser(ctx, r.org, user, "pm@example.test"); err != nil {
		t.Fatal(err)
	}
	for n := 1; n <= open; n++ {
		item, err := st.CreateDelivery(ctx, r.org, r.project, user, store.DeliveryInput{Content: delivery.Content{Kind: "rfi", Status: "open", Title: fmt.Sprintf("Question %d", n), OwnerText: "Structural Co",
			Details: json.RawMessage(fmt.Sprintf(`{"question":"Question %d","ref":"SC-%02d"}`, n, n))}})
		if err != nil {
			t.Fatal(err)
		}
		r.rfis[*item.Number] = item.ID
	}
	return r
}

func (r registerRun) file(t *testing.T, kind, text string) string {
	t.Helper()
	ctx := context.Background()
	doc := seedDoc(t, r.st, r.org, r.project, store.StatusFiled)
	for field, value := range map[string]string{"kind": kind, "title": "Reply"} {
		if err := r.st.CreateDecision(ctx, r.org, store.Decision{ID: newID(t), DocumentID: doc, Field: field, Value: value, Band: "green", DecidedBy: "rule"}); err != nil {
			t.Fatal(err)
		}
	}
	if err := r.st.ReplaceSource(ctx, r.org, doc, store.DocumentSource{Source: []store.SourcePage{{Text: text}}, Units: []store.SourceUnit{{Body: text, Page: 1}}}); err != nil {
		t.Fatal(err)
	}
	if err := r.st.QueueRegister(ctx, r.org, doc); err != nil {
		t.Fatal(err)
	}
	return doc
}

// run claims the queued register job and performs it under its lease.
func (r registerRun) run(t *testing.T, w *jobs.Worker) error {
	t.Helper()
	ctx := context.Background()
	job, err := r.st.ClaimJobSince(ctx, r.org, time.Minute, []string{store.JobKindRegister}, time.Time{})
	if err != nil {
		return err
	}
	return w.Perform(ctx, job)
}

func (r registerRun) readings(t *testing.T) map[string]string {
	t.Helper()
	pool, err := pgxpool.New(context.Background(), os.Getenv("SITEWISE_TEST_DATABASE_URL"))
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()
	rows, err := pool.Query(context.Background(), `SELECT item_id::text, decided_by||':'||band FROM delivery_readings WHERE org_id=$1::uuid`, r.org)
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	out := map[string]string{}
	for rows.Next() {
		var item, v string
		if err := rows.Scan(&item, &v); err != nil {
			t.Fatal(err)
		}
		out[item] = v
	}
	return out
}

func TestRegisterRuleMatchOnADrawing(t *testing.T) {
	r := registerProject(t, 8)
	ask := &registerAsk{}
	r.file(t, "drawing", "Revised detail issued in response to RFI 7. Membrane lapped 150 mm.")
	if err := r.run(t, &jobs.Worker{Store: r.st, Ask: ask}); err != nil {
		t.Fatal(err)
	}
	got := r.readings(t)
	if len(got) != 1 || got[r.rfis[7]] != "rule:amber" {
		t.Fatalf("readings %v", got)
	}
	if len(ask.calls) != 0 {
		t.Fatal("a drawing was sent to Jev")
	}
	view, err := r.st.ReadRegister(context.Background(), r.org, r.project, store.RegisterQuery{}, "2026-11-20")
	if err != nil {
		t.Fatal(err)
	}
	for _, it := range view.Items {
		if (len(it.Readings) == 1) != (it.ID == r.rfis[7]) {
			t.Fatalf("shown readings on %d: %v", *it.Number, it.Readings)
		}
	}
	// The sender's ref matches too.
	r.file(t, "drawing", "Our ref SC-03: see the revised section.")
	if err := r.run(t, &jobs.Worker{Store: r.st, Ask: ask}); err != nil {
		t.Fatal(err)
	}
	if r.readings(t)[r.rfis[3]] != "rule:amber" {
		t.Fatal("the ref was not matched")
	}
}

func TestRegisterUncitedLetterAsksOnce(t *testing.T) {
	r := registerProject(t, 14)
	ask := &registerAsk{yes: map[string]float64{"register.answers.r1": 0.93, "register.answers.r2": 0.2}}
	r.file(t, "correspondence", "Dear PM, the membrane to the basement wall shall be the sheet system shown. "+strings.Repeat("Further words. ", 400))
	if err := r.run(t, &jobs.Worker{Store: r.st, Ask: ask}); err != nil {
		t.Fatal(err)
	}
	if len(ask.calls) != 1 || len(ask.calls[0].Questions) != 12 {
		t.Fatalf("calls %d, questions %d", len(ask.calls), len(ask.calls[0].Questions))
	}
	state, _ := json.Marshal(ask.calls[0].State)
	if len(state) > 6000 {
		t.Fatalf("state too large: %d bytes", len(state))
	}
	got := r.readings(t)
	if len(got) != 12 || got[r.rfis[14]] != "jev:blank" || got[r.rfis[12]] != "jev:grey" {
		t.Fatalf("readings %v", got)
	}
	// Without an approved cut-off nothing Jev picked is shown.
	view, _ := r.st.ReadRegister(context.Background(), r.org, r.project, store.RegisterQuery{}, "2026-11-20")
	for _, it := range view.Items {
		if len(it.Readings) != 0 {
			t.Fatalf("a Jev reading was shown without a cut-off: %v", it.Readings)
		}
	}
	// With one, the pick above it is amber.
	r.file(t, "correspondence", "Dear PM, regarding the membrane, the sheet system is confirmed.")
	if err := r.run(t, &jobs.Worker{Store: r.st, Ask: ask, RegisterThresholds: jobs.RegisterThresholds{Amber: 0.8}}); err != nil {
		t.Fatal(err)
	}
	view, _ = r.st.ReadRegister(context.Background(), r.org, r.project, store.RegisterQuery{}, "2026-11-20")
	shown := 0
	for _, it := range view.Items {
		shown += len(it.Readings)
	}
	if shown != 1 {
		t.Fatalf("amber readings shown %d", shown)
	}
}

func TestRegisterJevFailureStoresGreyAndRetries(t *testing.T) {
	r := registerProject(t, 2)
	ask := &registerAsk{fail: errors.New("deadline exceeded")}
	r.file(t, "correspondence", "Dear PM, please find attached.")
	if err := r.run(t, &jobs.Worker{Store: r.st, Ask: ask}); err == nil {
		t.Fatal("a failed call must fail the job so it retries")
	}
	got := r.readings(t)
	if len(got) != 2 || got[r.rfis[1]] != "jev:grey" {
		t.Fatalf("readings %v", got)
	}
}

func TestRegisterNoOpenRFIQueuesNothing(t *testing.T) {
	r := registerProject(t, 0)
	r.file(t, "correspondence", "RFI-001 answered.")
	if err := r.run(t, &jobs.Worker{Store: r.st}); !errors.Is(err, store.ErrIdle) {
		t.Fatalf("a project with no Open RFI queued a job: %v", err)
	}
}

func TestRegisterExcerptAndRule(t *testing.T) {
	rfis := []store.RegisterRFI{{ID: "a", Number: 7}, {ID: "b", Number: 17}, {ID: "c", Number: 70, Ref: "B-12"}}
	hits, left := jobs.RuleMatches("Response to RFI-007 and our ref b-12.", rfis)
	if len(hits) != 2 || hits[0].ItemID != "a" || hits[1].ItemID != "c" || len(left) != 1 || left[0].ID != "b" {
		t.Fatalf("hits %v left %v", hits, left)
	}
	for _, text := range []string{"RFI #7", "rfi 0007", "RFI No. 7", "RFI-7."} {
		if h, _ := jobs.RuleMatches(text, rfis[:1]); len(h) != 1 {
			t.Errorf("%q did not match RFI 7", text)
		}
	}
	for _, text := range []string{"RFI-077", "RFI 71", "the RFI process"} {
		if h, _ := jobs.RuleMatches(text, rfis[:1]); len(h) != 0 {
			t.Errorf("%q matched RFI 7", text)
		}
	}
}

package jobs_test

import (
	"context"
	"errors"
	"strings"
	"sync"
	"testing"
	"time"

	"sitewise/internal/identity"
	"sitewise/internal/jev"
	"sitewise/internal/jobs"
	"sitewise/internal/store"
)

func TestSourceRetainsAllPagesOffsetsAndListContext(t *testing.T) {
	text := identity.Text{Format: "pdf", PageCount: 83}
	for p := 1; p <= 83; p++ {
		if p == 40 {
			continue
		}
		text.Runs = append(text.Runs, identity.Run{Text: "SPECIFIED WORK\nThe Contractor must supply:\n(a) hot water;\n(b) cold water.\n" + strings.Repeat("Long text. ", 220), Source: identity.Source{Page: p}})
	}
	src := jobs.SourceFromText(text)
	if src.Pages != 83 || len(src.EmptyPages) != 1 || src.EmptyPages[0] != 40 {
		t.Fatalf("coverage %+v", src.EmptyPages)
	}
	seen := make(map[int]string)
	cold := false
	for _, u := range src.Units {
		if u.Body != src.Source[u.Page-1].Text[u.Start:u.End] {
			t.Fatal("source offsets do not round-trip")
		}
		seen[u.Page] += u.Body
		if strings.Contains(u.Body, "(b) cold water") && strings.Contains(u.Context, "Contractor must supply") {
			cold = true
		}
	}
	for _, p := range src.Source {
		if strings.Join(strings.Fields(p.Text), "") != strings.Join(strings.Fields(seen[p.Page]), "") {
			t.Fatalf("lost content on page %d", p.Page)
		}
	}
	if !cold {
		t.Fatal("list item lost its responsible party")
	}
	if seen[83] == "" {
		t.Fatal("last page lost")
	}
}

func TestTableRowsKeepColumnHeadingsAndDocumentOrder(t *testing.T) {
	src := jobs.SourceFromText(identity.Text{Format: "docx", Runs: []identity.Run{
		{Text: "Before table"},
		{Text: "System", Source: identity.Source{Table: 1, Row: 1, Col: 1}},
		{Text: "Provider", Source: identity.Source{Table: 1, Row: 1, Col: 2}},
		{Text: "Hot water", Source: identity.Source{Table: 1, Row: 2, Col: 1}},
		{Text: "Owner", Source: identity.Source{Table: 1, Row: 2, Col: 2}},
		{Text: "After table"},
	}})
	if len(src.Source) != 4 || src.Source[0].Text != "Before table" || src.Source[3].Text != "After table" {
		t.Fatalf("table moved out of document order: %+v", src.Source)
	}
	found := false
	for _, u := range src.Units {
		if strings.Contains(u.Body, "Hot water") && strings.Contains(u.Body, "Owner") && strings.Contains(u.Context, "System\tProvider") {
			found = true
		}
	}
	if !found {
		t.Fatal("table row lost headings")
	}
}

type checkpointAsk struct {
	mu     sync.Mutex
	failed bool
	calls  map[string]int
}

func (f *checkpointAsk) Ask(_ context.Context, c jev.Call) (jev.Result, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	text := c.State.(map[string]any)["text"].(string)
	f.calls[text]++
	if strings.Contains(text, "FAIL") && !f.failed {
		f.failed = true
		return jev.Result{}, errors.New("temporary provider failure")
	}
	conf := 0.9
	r := jev.Result{Answers: map[string]jev.Answer{}}
	for id, q := range c.Questions {
		a := jev.Answer{Type: q.Type, Confidence: &conf}
		if q.Type == jev.TypeChoice {
			a.Choice = "not_stated"
			if id == "source.category" {
				a.Choice = "requirement"
			}
		}
		r.Answers[id] = a
	}
	return r, nil
}

func TestCompletedPassagesAreReusedOnRetry(t *testing.T) {
	st := openStore(t)
	ctx := context.Background()
	org, project := newID(t), newID(t)
	seedOrg(t, st, org, project)
	doc := seedDoc(t, st, org, project, store.StatusFiled)
	src := store.DocumentSource{Source: []store.SourcePage{{Text: "First requirement."}}, Units: []store.SourceUnit{{Body: "First requirement."}}}
	if err := st.ReplaceSource(ctx, org, doc, src); err != nil {
		t.Fatal(err)
	}
	ask := &checkpointAsk{calls: map[string]int{}}
	w := &jobs.Worker{Store: st, Ask: ask, Catalog: loadKnowledge(t), MinNoul: 0.5}
	job := store.ClaimedJob{OrgID: org, DocumentID: doc, Kind: store.JobKindLabel}
	if err := w.Perform(ctx, job); err != nil {
		t.Fatal(err)
	}
	if err := w.Perform(ctx, job); err != nil {
		t.Fatal(err)
	}
	if ask.calls["First requirement."] != 1 {
		t.Fatalf("paid to reread completed passage: %v", ask.calls)
	}
	records, err := st.SourceRecords(ctx, org, project, "", "needs_mapping", 0)
	if err != nil || len(records.Records) != 1 {
		t.Fatalf("unmapped requirement disappeared: %+v %v", records, err)
	}
}

func TestLeaseRenewalKeepsLongReadOwned(t *testing.T) {
	st := openStore(t)
	ctx := context.Background()
	org, project := newID(t), newID(t)
	seedOrg(t, st, org, project)
	doc := seedDoc(t, st, org, project, store.StatusFiled)
	if err := st.EnqueueStage(ctx, org, doc, store.JobKindFullText); err != nil {
		t.Fatal(err)
	}
	entered := make(chan struct{})
	release := make(chan struct{})
	w := &jobs.Worker{Store: st, Lease: 90 * time.Millisecond, Text: func(context.Context, string, string) (string, error) { close(entered); <-release; return "Text", nil }}
	done := make(chan error, 1)
	go func() { done <- w.Once(ctx, org) }()
	<-entered
	time.Sleep(180 * time.Millisecond)
	_, err := st.ClaimJob(ctx, org, time.Second, []string{store.JobKindFullText})
	close(release)
	if got := <-done; got != nil {
		t.Fatal(got)
	}
	if !errors.Is(err, store.ErrIdle) {
		t.Fatalf("live lease stolen: %v", err)
	}
}

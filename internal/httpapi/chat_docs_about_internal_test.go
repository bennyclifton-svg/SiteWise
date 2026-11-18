package httpapi

import (
	"context"
	"crypto/rand"
	"encoding/json"
	"fmt"
	"io"
	"log"
	mrand "math/rand/v2"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"

	"sitewise/internal/chatfix"
	"sitewise/internal/knowledge"
	"sitewise/internal/latency"
	"sitewise/internal/store"
)

func TestTitleNames(t *testing.T) {
	for _, c := range []struct {
		title string
		terms []string
		want  bool
	}{
		{"Basement excavation plan", []string{"basement"}, true},
		{"Basements and car park", []string{"basement"}, true},
		{"Car park", []string{"basements"}, false},
		{"Compiled drawings", []string{"pile"}, false},
		{"Pile layout", []string{"piles"}, true},
		{"Acme letterhead", []string{"acm"}, false},
		{"ACM register", []string{"acm"}, true},
		{"S-001 Basement slab", []string{"basement slab"}, true},
		{"Slab plan", []string{"basement slab"}, false},
	} {
		if got := titleNames(c.title, c.terms); got != c.want {
			t.Errorf("titleNames(%q, %v) = %v", c.title, c.terms, got)
		}
	}
	for in, want := range map[string]string{"the basement": "basement", "all the basement documents": "basement", "Basement, the shoring": "basement,shoring", "the documents": ""} {
		if got := aboutWords(in); got != want {
			t.Errorf("aboutWords(%q) = %q, want %q", in, got, want)
		}
	}
}

// TestDocumentsAboutBudget holds chat_tool_read for the documents tool with
// about on the plan's size: 2,000 documents and 200,000 passages, every
// passage labelled with a building system as the profile's reading would.
// The figure is the tool's own Go work: the document list, the one counting
// query and the ranking. The gate runs it with go test.
func TestDocumentsAboutBudget(t *testing.T) {
	dsn := os.Getenv("SITEWISE_TEST_DATABASE_URL")
	if dsn == "" {
		t.Fatal("SITEWISE_TEST_DATABASE_URL is required")
	}
	if u, err := url.Parse(dsn); err != nil || strings.TrimPrefix(u.Path, "/") != "sitewise_test" {
		t.Fatalf("refusing to use database %q", dsn)
	}
	ctx := context.Background()
	st, err := store.Open(ctx, dsn)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(st.Close)
	if err := st.Migrate(ctx); err != nil {
		t.Fatal(err)
	}
	org, project := benchUUID(t), benchUUID(t)
	if err := st.CreateOrg(ctx, org, "About bench"); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = st.DeleteOrg(context.Background(), org) })
	if err := st.CreateProject(ctx, org, project, "About bench"); err != nil {
		t.Fatal(err)
	}

	began := time.Now()
	docs := seedAboutDocs(t, st, org, project, 2000)
	seedAboutPassages(t, dsn, org, docs, 200_000)
	t.Logf("fixture: 2,000 documents (%d with text), 200,000 passages in %s", len(docs), time.Since(began).Round(time.Second))

	synonyms, systems, err := LoadChatWords(filepath.Join("..", "..", "knowledge"))
	if err != nil {
		t.Fatal(err)
	}
	cat, err := knowledge.Load(filepath.Join("..", "..", "knowledge"))
	if err != nil {
		t.Fatal(err)
	}
	b := &chatBox{d: Deps{Store: st, Log: log.New(io.Discard, "", 0), Knowledge: cat, Chat: &Chat{Synonyms: synonyms, SystemTerms: systems}},
		who: opActor{OrgID: org}, project: project}
	subjects := []string{"the basement", "roof", "fire, smoke", "the basement"}
	var samples []int64
	for i := 0; i < 24; i++ {
		raw, _ := json.Marshal(map[string]any{"about": subjects[i%len(subjects)], "select": true})
		start := time.Now()
		out := b.documents(ctx, raw)
		samples = append(samples, time.Since(start).Microseconds())
		if out.IsError || !strings.Contains(out.Text, "select_ids: ") {
			t.Fatalf("documents about %q: %s", subjects[i%len(subjects)], out.Text[:min(len(out.Text), 600)])
		}
		if i == 0 {
			t.Logf("first lines: %s", strings.Join(strings.SplitN(out.Text, "\n", 4)[1:3], " / "))
		}
	}
	budgets, err := latency.LoadBudgets(filepath.Join("..", "..", "bench", "budgets.json"))
	if err != nil {
		t.Fatal(err)
	}
	scoped := latency.Budgets{MinSamples: budgets.MinSamples}
	for _, p := range budgets.Paths {
		if p.Name == "chat_tool_read" {
			scoped.Paths = append(scoped.Paths, p)
		}
	}
	if len(scoped.Paths) != 1 {
		t.Fatal("bench/budgets.json lacks chat_tool_read")
	}
	p50, _ := latency.Percentile(samples, .5)
	p90, _ := latency.Percentile(samples, .9)
	t.Logf("chat_tool_read (documents about, 2,000 documents) p50=%d us p90=%d us", p50, p90)
	if code, report := latency.Gate(scoped, map[string][]int64{"chat_tool_read": samples}, true); code != 0 {
		t.Fatalf("budget broken: %s", report)
	}
}

func benchUUID(t *testing.T) string {
	t.Helper()
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		t.Fatal(err)
	}
	b[6], b[8] = b[6]&0x0f|0x40, b[8]&0x3f|0x80
	return fmt.Sprintf("%x-%x-%x-%x-%x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:])
}

// aboutTopics are the bench's titles, each with the building system the
// profile's reading would label most of its passages with and the subject
// word its text uses. One in twenty is about the basement.
var aboutTopics = []struct{ title, system, word string }{
	{"Ground floor plan", "interiors", ""}, {"Roof plan", "roof", "roof"}, {"Elevations", "envelope", ""}, {"Sections", "structure", ""},
	{"Basement plan", "substructure.basement-retention", "basement"}, {"Fire services layout", "fire", "fire"},
	{"Hydraulic layout", "hydraulic", ""}, {"Mechanical layout", "mechanical", "smoke"}, {"Door schedule", "interiors", ""},
	{"Finishes schedule", "interiors", ""}, {"Electrical layout", "electrical", ""}, {"Site plan", "site", ""},
	{"Landscape plan", "site", ""}, {"Stair details", "structure", ""}, {"Facade details", "envelope", ""},
	{"Structural notes", "structure", "basement"}, {"Slab set out", "substructure", ""}, {"Ceiling plan", "interiors", ""},
	{"Wall types", "interiors", "fire"}, {"Window schedule", "envelope", ""},
}

type benchDoc struct {
	id    string
	topic int
}

// seedAboutDocs files n documents through the ordinary filing path, eight
// at a time. One in forty is a scan with no text, so some scans are
// basement plans that only their title can find.
func seedAboutDocs(t *testing.T, st *store.Store, org, project string, n int) []benchDoc {
	t.Helper()
	var mu sync.Mutex
	var out []benchDoc
	var wg sync.WaitGroup
	errs := make(chan error, 8)
	for w := 0; w < 8; w++ {
		wg.Add(1)
		go func(w int) {
			defer wg.Done()
			var docs []chatfix.Document
			for i := w; i < n; i += 8 {
				d := chatfix.Document{Title: fmt.Sprintf("%s %04d", aboutTopics[i%len(aboutTopics)].title, i), Kind: "drawing", Number: fmt.Sprintf("A-%04d", i)}
				if i%40 != 4 {
					d.Pages = []string{"Drawing notes."}
				}
				docs = append(docs, d)
			}
			got, err := chatfix.SeedDocuments(context.Background(), st, org, project, docs)
			if err != nil {
				errs <- err
				return
			}
			mu.Lock()
			for i, d := range docs {
				if d.Pages != nil {
					out = append(out, benchDoc{got[d.Title], (w + 8*i) % len(aboutTopics)})
				}
			}
			mu.Unlock()
		}(w)
	}
	wg.Wait()
	close(errs)
	for err := range errs {
		t.Fatal(err)
	}
	return out
}

// seedAboutPassages adds passages of random words to the documents with
// text, until there are about total in all. A document's subject word is in
// a third of its passages, and any subject word strays into one passage in
// two hundred elsewhere. Every passage carries one system label: its
// document's system six times in ten, another system otherwise.
func seedAboutPassages(t *testing.T, dsn, org string, docs []benchDoc, total int) {
	t.Helper()
	ctx := context.Background()
	conn, err := pgx.Connect(ctx, dsn)
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close(ctx)
	var vocab []string
	for i := 0; i < 3000; i++ {
		vocab = append(vocab, fmt.Sprintf("word%d", i))
	}
	subjects := []string{"basement", "roof", "fire", "smoke"}
	others := []string{"structure", "roof", "fire", "mechanical", "hydraulic", "electrical", "envelope", "interiors", "site"}
	rng := mrand.New(mrand.NewPCG(1, 2))
	var passages, labels [][]any
	per := (total - len(docs)) / len(docs)
	for _, doc := range docs {
		topic := aboutTopics[doc.topic]
		for k := 0; k < per; k++ {
			id := benchUUID(t)
			words := make([]string, 40)
			for j := range words {
				words[j] = vocab[rng.IntN(len(vocab))]
			}
			if topic.word != "" && rng.IntN(3) == 0 {
				words[rng.IntN(len(words))] = topic.word
			}
			if rng.IntN(200) == 0 {
				words[rng.IntN(len(words))] = subjects[rng.IntN(len(subjects))]
			}
			system := topic.system
			if rng.IntN(10) >= 6 {
				system = others[rng.IntN(len(others))]
			}
			passages = append(passages, []any{org, id, doc.id, 1000 + k, strings.Join(words, " ")})
			labels = append(labels, []any{org, id, system})
		}
	}
	if _, err := conn.CopyFrom(ctx, pgx.Identifier{"passages"}, []string{"org_id", "id", "document_id", "ordinal", "body"}, pgx.CopyFromRows(passages)); err != nil {
		t.Fatal(err)
	}
	if _, err := conn.CopyFrom(ctx, pgx.Identifier{"passage_systems"}, []string{"org_id", "passage_id", "system_id"}, pgx.CopyFromRows(labels)); err != nil {
		t.Fatal(err)
	}
	// As autovacuum would leave them: statistics for the planner, and the
	// visibility map the index-only scans rely on.
	for _, table := range []string{"passages", "passage_systems", "documents"} {
		if _, err := conn.Exec(ctx, "VACUUM ANALYZE "+table); err != nil {
			t.Fatal(err)
		}
	}
}

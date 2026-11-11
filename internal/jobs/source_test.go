package jobs_test

import (
	"context"
	"errors"
	"fmt"
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

// A builder's quote exported from an estimating tool: the bullet glyph of a
// wrapped item is emitted after its text, list headings are plain short
// lines set left of the list, and every page ends with the firm's footer.
func TestBulletListKeepsBulletsHeadingsAndFooterApart(t *testing.T) {
	line := func(page int, x, y float64, s string) identity.Run {
		return identity.Run{Text: s, Source: identity.Source{Page: page, X: x, Y: y, Width: 100, Height: 11.6}}
	}
	text := identity.Text{Format: "pdf", PageCount: 4}
	for p := 1; p <= 4; p++ {
		if p > 1 {
			text.Runs = append(text.Runs,
				line(p, 36.9, 646, fmt.Sprintf("Trade %d", p)),
				line(p, 62.3, 630, fmt.Sprintf("• PERMITS for trade %d are included.", p)),
				line(p, 36.9, 60, "Seaview Builders"),
				line(p, 36.9, 50, "Po Box 12 Seaview VIC 3999"),
				line(p, 36.9, 40, "ABN: 51824753556"),
				line(p, 500, 30, fmt.Sprintf("Page %d of 4", p)),
			)
			continue
		}
		text.Runs = append(text.Runs,
			line(p, 36.9, 646, "Demolition"),
			line(p, 77.2, 630, "SCOPE OF WORKS - A single-story brick veneer house on timber and concrete floors with steel roofing."),
			line(p, 77.2, 620, "No vegetation is included. No site earthworks are"),
			line(p, 77.2, 610, "included. Site to be track rolled."),
			line(p, 62.3, 630, "•"),
			line(p, 62.3, 600, "• PERMITS - Building Permit is included."),
			line(p, 36.9, 590, "Roofing & Cladding"),
			line(p, 62.3, 575, "• Supply and fit Colorbond to roof and walls."),
			line(p, 77.2, 560, "Includes proctor wrap, Colorbond ridges and trims,"),
			line(p, 77.2, 550, "Colorbond 90mm downpipes."),
			line(p, 62.3, 560, "•"),
			line(p, 36.9, 60, "Seaview Builders"),
			line(p, 36.9, 50, "Po Box 12 Seaview VIC 3999"),
			line(p, 36.9, 40, "ABN: 51824753556"),
			line(p, 500, 30, fmt.Sprintf("Page %d of 4", p)),
		)
	}
	src := jobs.SourceFromText(text)
	var page1 []store.SourceUnit
	for _, u := range src.Units {
		if u.Body != src.Source[u.Page-1].Text[u.Start:u.End] {
			t.Fatal("source offsets do not round-trip")
		}
		if strings.Trim(u.Body, " \n•") == "" {
			t.Fatalf("a bare bullet is a unit: %q", u.Body)
		}
		if u.Page == 1 {
			page1 = append(page1, u)
		}
	}
	find := func(units []store.SourceUnit, words string) store.SourceUnit {
		t.Helper()
		for _, u := range units {
			if strings.Contains(u.Body, words) {
				return u
			}
		}
		t.Fatalf("no unit holds %q", words)
		return store.SourceUnit{}
	}
	scope := find(page1, "SCOPE OF WORKS")
	if scope.Section != "Demolition" || !strings.Contains(scope.Body, "• SCOPE OF WORKS") || strings.Contains(scope.Body, "PERMITS") || !strings.Contains(scope.Body, "track rolled") {
		t.Fatalf("scope unit %+v", scope)
	}
	permits := find(page1, "PERMITS")
	if permits.Section != "Demolition" || strings.Contains(permits.Body, "Roofing") {
		t.Fatalf("a heading rode on the item above it: %+v", permits)
	}
	wrap := find(page1, "proctor wrap")
	if wrap.Section != "Roofing & Cladding" || !strings.Contains(wrap.Body, "• Includes proctor wrap") || strings.Contains(wrap.Body, "Seaview Builders") {
		t.Fatalf("wrapped item %+v", wrap)
	}
	// The footer is kept, apart from the works; a repeat is not read again.
	if first := find(page1, "Po Box 12"); first.Category != "" || !strings.Contains(first.Body, "Page 1 of 4") {
		t.Fatalf("first footer %+v", first)
	}
	for _, u := range src.Units {
		if u.Page > 1 && strings.Contains(u.Body, "Po Box 12") && u.Category != store.RepeatedFurniture {
			t.Fatalf("repeated footer on page %d is read again: %+v", u.Page, u)
		}
		if u.Page > 1 && strings.Contains(u.Body, "PERMITS") && u.Category != "" {
			t.Fatalf("works marked background: %+v", u)
		}
	}
}

// A priced table: each row is one unit under the table's title, which is
// only marked by its larger type. A numbered row is not a heading.
func TestPricedTableRowsAreUnitsUnderTheirTitle(t *testing.T) {
	text := identity.Text{Format: "pdf", PageCount: 1}
	y := 700.0
	cell := func(x, h float64, s string) {
		text.Runs = append(text.Runs, identity.Run{Text: s, Source: identity.Source{Page: 1, X: x, Y: y, Width: 80, Height: h}})
	}
	cell(247, 17.4, "Quoted Categories")
	y -= 20
	cell(65, 11.6, "Description of category")
	cell(504, 11.6, "Total incl. GST")
	for _, row := range [][3]string{{"19", "Security / Intercom / Data - See Specifications", "$35,000.00"}, {"20", "Solar Panels & Battery System - as per quotation provided", "$90,576.51"}, {"33", "Shed Retaining Walls", "$20,867.72"}} {
		y -= 20
		cell(36, 11.9, row[0])
		cell(65, 11.9, row[1])
		cell(505, 11.9, row[2])
	}
	y -= 20
	cell(58, 11.6, "4.12 Rock for gravel driveway west to East/West of house")
	cell(434, 11.6, "6.000 Ea")
	cell(521, 11.6, "$3,960.00")
	src := jobs.SourceFromText(text)
	var solar, rock *store.SourceUnit
	for i, u := range src.Units {
		if strings.Contains(u.Body, "Solar Panels") {
			solar = &src.Units[i]
		}
		if strings.Contains(u.Body, "4.12 Rock") {
			rock = &src.Units[i]
		}
	}
	if solar == nil || solar.Section != "Quoted Categories" || !strings.Contains(solar.Body, "$90,576.51") || strings.Contains(solar.Body, "Security") || strings.Contains(solar.Body, "Retaining") {
		t.Fatalf("solar row %+v", solar)
	}
	if rock == nil || rock.Section != "Quoted Categories" || !strings.Contains(rock.Body, "$3,960.00") {
		t.Fatalf("numbered row %+v", rock)
	}
}

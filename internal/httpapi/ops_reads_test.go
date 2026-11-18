package httpapi_test

import (
	"context"
	"net/http"
	"os"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"

	"sitewise/internal/chatfix"
	"sitewise/internal/llm"
	"sitewise/internal/store"
)

// Plan 0 Task 6 (owner decision 1, 18 November 2026): chat prepares a Jev
// read as a card and the PM's click queues it. The turn never calls Jev;
// the click runs the screen's own route, which only queues.

func testSQL(t *testing.T, query string, args ...any) {
	t.Helper()
	ctx := context.Background()
	conn, err := pgx.Connect(ctx, os.Getenv("SITEWISE_TEST_DATABASE_URL"))
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close(ctx)
	if _, err := conn.Exec(ctx, query, args...); err != nil {
		t.Fatal(err)
	}
}

func testScan(t *testing.T, query string, args []any, into ...any) {
	t.Helper()
	ctx := context.Background()
	conn, err := pgx.Connect(ctx, os.Getenv("SITEWISE_TEST_DATABASE_URL"))
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close(ctx)
	if err := conn.QueryRow(ctx, query, args...).Scan(into...); err != nil {
		t.Fatalf("%s: %v", query, err)
	}
}

// projectJobs is every job of the project's documents as "kind status",
// sorted, so two moments can be compared.
func projectJobs(t *testing.T, org, project string) []string {
	t.Helper()
	ctx := context.Background()
	conn, err := pgx.Connect(ctx, os.Getenv("SITEWISE_TEST_DATABASE_URL"))
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close(ctx)
	rows, err := conn.Query(ctx, `SELECT j.id::text||' '||j.kind||' '||j.status FROM jobs j JOIN documents d ON d.org_id=j.org_id AND d.id=j.document_id WHERE j.org_id=$1::uuid AND d.project_id=$2::uuid`, org, project)
	if err != nil {
		t.Fatal(err)
	}
	out, err := pgx.CollectRows(rows, pgx.RowTo[string])
	if err != nil {
		t.Fatal(err)
	}
	sort.Strings(out)
	return out
}

type appliedCard struct {
	Kind     string   `json:"kind"`
	Lines    []string `json:"lines"`
	ChangeID string   `json:"change_id"`
	Undo     string   `json:"undo"`
}

// prepareRead runs a turn that only calls apply with one read operation. The
// turn read nothing, and the read is still prepared, never applied.
func (ta *toolApp) prepareRead(t *testing.T, area, name string, fields map[string]any) chatEvent {
	t.Helper()
	jev := ta.a.jevHits.Load()
	events := ta.run(t, "Please do it", tool("apply", map[string]any{"area": area, "ops": []map[string]any{op(name, fields)}, "final": true}))
	for _, e := range events {
		if e.Type == "change" {
			t.Fatalf("%s was applied at once: %+v", name, e.Card)
		}
	}
	for _, e := range events {
		if e.Type == "prepared" {
			break
		}
		if e.Type == "done" {
			t.Fatalf("%s: no card; results %q", name, ta.results)
		}
	}
	card := cardOf(t, events, "prepared")
	if card.Card.Why == "" || len(card.Card.Lines) != 1 {
		t.Fatalf("%s card %+v", name, card.Card)
	}
	if ta.a.jevHits.Load() != jev {
		t.Fatalf("%s: preparing the card called Jev", name)
	}
	return card
}

// click applies a prepared card as the PM's click does. The applied card
// offers no Undo, and the click called no Jev.
func (ta *toolApp) click(t *testing.T, card chatEvent) appliedCard {
	t.Helper()
	jev := ta.a.jevHits.Load()
	var out appliedCard
	ta.m.call(t, http.MethodPost, "/api/projects/"+ta.project+"/chat/prepared/"+card.Card.Prepared+"/apply", []byte(`{}`), http.StatusOK, &out)
	if out.Kind != "change" || out.Undo != "none" || out.ChangeID != "" || len(out.Lines) != 1 {
		t.Fatalf("the applied read offers an undo or says nothing: %+v", out)
	}
	if ta.a.jevHits.Load() != jev {
		t.Fatal("the click called Jev while the PM waited")
	}
	return out
}

// refusedRead runs a turn whose read is refused before any card exists.
func (ta *toolApp) refusedRead(t *testing.T, area, name string, fields map[string]any, want string) {
	t.Helper()
	_, before := changeCounts(t, ta.m.orgID)
	events := ta.run(t, "Please do it", tool("apply", map[string]any{"area": area, "ops": []map[string]any{op(name, fields)}, "final": true}))
	for _, e := range events {
		if e.Type == "prepared" || e.Type == "change" {
			t.Fatalf("%s: a %s card was made", name, e.Type)
		}
	}
	if len(ta.results) != 1 || !strings.Contains(ta.results[0], "Refused") || !strings.Contains(ta.results[0], want) {
		t.Fatalf("%s: results %q, want %q", name, ta.results, want)
	}
	if _, after := changeCounts(t, ta.m.orgID); after != before {
		t.Fatalf("%s: a prepared change was stored", name)
	}
}

func cardSays(t *testing.T, card chatEvent, words ...string) {
	t.Helper()
	for _, w := range words {
		if !strings.Contains(card.Card.Lines[0], w) {
			t.Fatalf("card %q does not say %q", card.Card.Lines[0], w)
		}
	}
}

func TestChatPreparesDocumentReads(t *testing.T) {
	ta := newToolApp(t)
	ctx := context.Background()
	org, project := ta.m.orgID, ta.project
	ids, err := chatfix.SeedDocuments(ctx, ta.a.store, org, project, []chatfix.Document{
		{Title: "Development consent", Kind: "statutory_instrument", Number: "DA-2025-0733", Pages: []string{"Condition 1. Build what was approved."}},
		{Title: "Survey scan", Kind: "drawing"},
		{Title: "Architectural set", Kind: "drawing"},
	})
	if err != nil {
		t.Fatal(err)
	}
	consent, scan, set := ids["Development consent"], ids["Survey scan"], ids["Architectural set"]
	// A scan whose filing stopped for want of a text layer.
	up, err := ta.a.store.CommitIntake(ctx, org, store.CommitIntake{ProjectID: project, SHA256: []byte(strings.Repeat("s", 32)), ByteSize: 1000, MediaType: "application/pdf", Filename: "Old scan.pdf"})
	if err != nil {
		t.Fatal(err)
	}
	stopped := up.DocumentID
	if err := ta.a.store.MarkNotFiled(ctx, org, stopped, "no_text_layer"); err != nil {
		t.Fatal(err)
	}
	// A scan filed with details missing, and a set the split stopped on.
	testSQL(t, `UPDATE documents SET reason='ocr_review' WHERE org_id=$1::uuid AND id=$2::uuid`, org, scan)
	testSQL(t, `INSERT INTO drawing_expansions(org_id,source_id,page_count,status,reason) VALUES($1::uuid,$2::uuid,3,'review',$3)`, org, set, store.UnconfirmedPageReason(2))

	// Another project of this org, and another org, each with a document.
	elsewhere := ta.m.createProject(t, "Elsewhere")
	theirs, err := chatfix.SeedDocuments(ctx, ta.a.store, org, elsewhere, []chatfix.Document{{Title: "Elsewhere consent", Kind: "statutory_instrument"}})
	if err != nil {
		t.Fatal(err)
	}
	other := ta.a.member(t, newUUID(t))
	otherProject := other.createProject(t, "Theirs")
	foreign, err := chatfix.SeedDocuments(ctx, ta.a.store, other.orgID, otherProject, []chatfix.Document{{Title: "Their consent", Kind: "statutory_instrument"}})
	if err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"documents.read_images", "documents.read_missing_details", "documents.split_set", "documents.retry_filing"} {
		ta.refusedRead(t, "documents", name, map[string]any{"document": theirs["Elsewhere consent"]}, "There is no document")
		ta.refusedRead(t, "documents", name, map[string]any{"document": foreign["Their consent"]}, "There is no document")
	}
	ta.refusedRead(t, "profile", "profile.document_reading", map[string]any{"documents": []string{consent, foreign["Their consent"]}, "setting": "read"}, "There is no document")

	// Image review: the job the screen's button queues, once. Filing queued
	// one already; it is refused, then marked as having failed so it can be
	// asked for again.
	ta.refusedRead(t, "documents", "documents.read_images", map[string]any{"document": consent}, "already queued")
	testSQL(t, `UPDATE visual_identity SET status='unavailable' WHERE org_id=$1::uuid AND document_id=$2::uuid`, org, consent)
	testSQL(t, `UPDATE jobs SET status='failed' WHERE org_id=$1::uuid AND document_id=$2::uuid AND kind='visual_identity'`, org, consent)
	card := ta.prepareRead(t, "documents", "documents.read_images", map[string]any{"document": consent[:8]})
	cardSays(t, card, "DA-2025-0733 Development consent", "page images", "Jev credits", "can't be undone")
	var status string
	if review, err := ta.a.store.VisualIdentity(ctx, org, consent); err != nil || review.Status != "unavailable" {
		t.Fatalf("preparing the card queued the image review: %+v %v", review, err)
	}
	ta.click(t, card)
	testScan(t, `SELECT v.status||'/'||j.status FROM visual_identity v JOIN jobs j ON j.org_id=v.org_id AND j.document_id=v.document_id AND j.kind='visual_identity' WHERE v.org_id=$1::uuid AND v.document_id=$2::uuid`, []any{org, consent}, &status)
	if status != "queued/queued" {
		t.Fatalf("image review after the click: %s", status)
	}
	ta.refusedRead(t, "documents", "documents.read_images", map[string]any{"document": consent}, "already queued")

	// Missing details of a scan.
	card = ta.prepareRead(t, "documents", "documents.read_missing_details", map[string]any{"document": scan})
	cardSays(t, card, "Survey scan", "Jev credits")
	testScan(t, `SELECT reason FROM documents WHERE org_id=$1::uuid AND id=$2::uuid`, []any{org, scan}, &status)
	if status != "ocr_review" {
		t.Fatalf("preparing the card changed the document: %s", status)
	}
	ta.click(t, card)
	testScan(t, `SELECT d.reason||'/'||j.status FROM documents d JOIN jobs j ON j.org_id=d.org_id AND j.document_id=d.id AND j.kind='ocr' WHERE d.org_id=$1::uuid AND d.id=$2::uuid`, []any{org, scan}, &status)
	if status != "ocr_details_queued/queued" {
		t.Fatalf("missing details after the click: %s", status)
	}
	ta.refusedRead(t, "documents", "documents.read_missing_details", map[string]any{"document": scan}, "already reading")
	ta.refusedRead(t, "documents", "documents.read_missing_details", map[string]any{"document": consent}, "no missing details")

	// A stopped filing starts again through the OCR queue.
	card = ta.prepareRead(t, "documents", "documents.retry_filing", map[string]any{"document": stopped})
	cardSays(t, card, "Old scan.pdf", "can't be undone")
	ta.click(t, card)
	testScan(t, `SELECT d.status||'/'||d.reason||'/'||j.status FROM documents d JOIN jobs j ON j.org_id=d.org_id AND j.document_id=d.id AND j.kind='ocr' WHERE d.org_id=$1::uuid AND d.id=$2::uuid`, []any{org, stopped}, &status)
	if status != "pending/ocr_queued/queued" {
		t.Fatalf("retry after the click: %s", status)
	}
	ta.refusedRead(t, "documents", "documents.retry_filing", map[string]any{"document": stopped}, "already queued")
	ta.refusedRead(t, "documents", "documents.retry_filing", map[string]any{"document": consent}, "no stopped filing")

	// A drawing set the automatic split stopped on.
	ta.refusedRead(t, "documents", "documents.split_set", map[string]any{"document": consent}, "not a drawing set")
	card = ta.prepareRead(t, "documents", "documents.split_set", map[string]any{"document": set})
	cardSays(t, card, "Architectural set", "every sheet", "Jev credits")
	ta.click(t, card)
	var manual bool
	testScan(t, `SELECT manual FROM drawing_expansions WHERE org_id=$1::uuid AND source_id=$2::uuid`, []any{org, set}, &manual)
	if !manual {
		t.Fatal("the split was not queued")
	}
	ta.refusedRead(t, "documents", "documents.split_set", map[string]any{"document": set}, "split")

	// The profile's reading setting, as the register's selection sends it.
	card = ta.prepareRead(t, "profile", "profile.document_reading", map[string]any{"documents": []string{consent[:8]}, "setting": "skip"})
	cardSays(t, card, "Stop the profile reading DA-2025-0733 Development consent")
	ta.click(t, card)
	testScan(t, `SELECT profile_read FROM documents WHERE org_id=$1::uuid AND id=$2::uuid`, []any{org, consent}, &status)
	if status != "skip" {
		t.Fatalf("profile reading %s", status)
	}
	ta.refusedRead(t, "profile", "profile.document_reading", map[string]any{"documents": []string{scan}, "setting": "sometimes"}, "read, skip or auto")
	testSQL(t, `UPDATE documents SET profile_read='auto' WHERE org_id=$1::uuid AND id=$2::uuid`, org, consent)

	// A profile update and a specification update queue what the screen's
	// own buttons queue: pressing the button after the click adds nothing.
	for _, read := range []struct{ area, op, route, says string }{
		{"profile", "profile.update", "/profile/read", "Update the project profile"},
		{"specifications", "specifications.update", "/specifications/update", "specification schedule"},
	} {
		before := projectJobs(t, org, project)
		card = ta.prepareRead(t, read.area, read.op, nil)
		cardSays(t, card, read.says, "Jev credits")
		if got := projectJobs(t, org, project); strings.Join(got, ",") != strings.Join(before, ",") {
			t.Fatalf("%s: preparing the card queued work", read.op)
		}
		ta.click(t, card)
		after := projectJobs(t, org, project)
		if read.op == "specifications.update" && strings.Join(after, ",") == strings.Join(before, ",") {
			t.Fatal("the specification read queued nothing")
		}
		status := ta.m.status(t, http.MethodPost, "/api/projects/"+project+read.route, nil)
		if status != http.StatusOK && status != http.StatusAccepted {
			t.Fatalf("%s screen route: %d", read.op, status)
		}
		if again := projectJobs(t, org, project); strings.Join(again, ",") != strings.Join(after, ",") {
			t.Fatalf("%s: the screen's button queued more than the click did:\n%v\n%v", read.op, after, again)
		}
	}
	if ta.a.jevHits.Load() != 0 {
		t.Fatalf("Jev was called %d times", ta.a.jevHits.Load())
	}
}

// blockingAsker holds every model call until released, so a route that
// waited on the model would not return.
type blockingAsker struct {
	release chan struct{}
	calls   chan struct{}
}

func (b *blockingAsker) Ask(ctx context.Context, req llm.Request) (llm.Response, error) {
	b.calls <- struct{}{}
	select {
	case <-b.release:
	case <-ctx.Done():
		return llm.Response{}, ctx.Err()
	}
	return (&countingAsker{}).Ask(ctx, req)
}

func TestChatPreparesProcurementReads(t *testing.T) {
	ask := &blockingAsker{release: make(chan struct{}), calls: make(chan struct{}, 8)}
	ta := newToolApp(t, greenProposals, withTender(ask, tinyRender))
	ctx := context.Background()
	org, project := ta.m.orgID, ta.project
	var pkg struct {
		ID    string `json:"id"`
		Title string `json:"title"`
	}
	ta.m.call(t, http.MethodPost, "/api/projects/"+project+"/procurement/rows", []byte(`{"discipline_id":"consultant.town_planner"}`), http.StatusCreated, &pkg)
	ids, err := chatfix.SeedDocuments(ctx, ta.a.store, org, project, []chatfix.Document{{Title: "Planning fee proposal", Kind: "proposal"}})
	if err != nil {
		t.Fatal(err)
	}
	doc := ids["Planning fee proposal"]
	other := ta.a.member(t, newUUID(t))
	otherProject := other.createProject(t, "Theirs")
	var theirPkg struct {
		ID string `json:"id"`
	}
	other.call(t, http.MethodPost, "/api/projects/"+otherProject+"/procurement/rows", []byte(`{"discipline_id":"consultant.town_planner"}`), http.StatusCreated, &theirPkg)
	foreign, err := chatfix.SeedDocuments(ctx, ta.a.store, other.orgID, otherProject, []chatfix.Document{{Title: "Their proposal", Kind: "proposal"}})
	if err != nil {
		t.Fatal(err)
	}

	// A proposal placed on a package: refused for another org's ids.
	ta.refusedRead(t, "procurement", "procurement.read_proposal", map[string]any{"package": theirPkg.ID, "document": doc}, "package")
	ta.refusedRead(t, "procurement", "procurement.read_proposal", map[string]any{"package": pkg.ID, "document": foreign["Their proposal"]}, "There is no document")
	card := ta.prepareRead(t, "procurement", "procurement.read_proposal", map[string]any{"package": pkg.ID[:8], "document": doc[:8]})
	cardSays(t, card, "Planning fee proposal", pkg.Title, "can't be undone")
	if sub, _ := ta.a.store.SubmissionForDocument(ctx, org, doc); sub != "" {
		t.Fatal("preparing the card opened a submission")
	}
	ta.click(t, card)
	sub, err := ta.a.store.SubmissionForDocument(ctx, org, doc)
	if err != nil || sub == "" {
		t.Fatalf("no submission after the click: %v", err)
	}
	var placed string
	testScan(t, `SELECT COALESCE(package_id::text,'') FROM submissions WHERE org_id=$1::uuid AND id=$2::uuid`, []any{org, sub}, &placed)
	if placed != pkg.ID {
		t.Fatalf("the submission is on %q, want %q", placed, pkg.ID)
	}
	ta.refusedRead(t, "procurement", "procurement.read_proposal", map[string]any{"package": pkg.ID, "document": doc}, "already on")

	// The tender pass again: a quote that has been read.
	var labels []string
	var table priceTableBody
	ta.m.getJSON(t, "/api/projects/"+project+"/procurement/packages/"+pkg.ID+"/prices", &table)
	for _, it := range table.Items {
		if it.Base {
			labels = append(labels, it.Label)
		}
	}
	ta.m.modelSwitch(t, true, http.StatusOK)
	base := "/api/projects/" + project + "/procurement/packages/" + pkg.ID
	quote := ta.m.dropTender(t, base, "Invented Planning Pty Ltd", labels)
	<-ask.calls
	ask.release <- struct{}{}
	ta.m.waitTender(t, base, quote, "read")
	ta.refusedRead(t, "procurement", "procurement.read_tender_again", map[string]any{"package": pkg.ID, "firm": "Nobody Pty Ltd"}, "not exactly one firm")
	card = ta.prepareRead(t, "procurement", "procurement.read_tender_again", map[string]any{"package": pkg.ID, "firm": "Invented Planning"})
	cardSays(t, card, "Invented Planning Pty Ltd", "again", "can't be undone")
	var state string
	testScan(t, `SELECT tender_state FROM submissions WHERE org_id=$1::uuid AND id=$2::uuid`, []any{org, quote}, &state)
	if state != store.TenderRead {
		t.Fatalf("preparing the card queued the pass: %s", state)
	}
	// The click returns while the model is held: it only queued the pass.
	done := make(chan appliedCard, 1)
	go func() { done <- ta.click(t, card) }()
	select {
	case <-done:
	case <-time.After(10 * time.Second):
		t.Fatal("the click waited on the model")
	}
	testScan(t, `SELECT tender_state FROM submissions WHERE org_id=$1::uuid AND id=$2::uuid`, []any{org, quote}, &state)
	if state != store.TenderPending && state != store.TenderReading {
		t.Fatalf("the pass was not queued: %s", state)
	}
	ta.refusedRead(t, "procurement", "procurement.read_tender_again", map[string]any{"package": pkg.ID, "firm": "Invented Planning"}, "already being read")
	<-ask.calls
	close(ask.release)
	ta.m.waitTender(t, base, quote, "read")

}

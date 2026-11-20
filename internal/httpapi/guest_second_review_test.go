package httpapi_test

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"reflect"
	"sort"
	"testing"
	"time"

	psample "sitewise/internal/proposals/sample"
	"sitewise/internal/store"
)

// The second security review of guest access (20 November 2026). Each test
// here fails without its fix.

// Finding 1: a bare firm renamed in place to a wholly different name is a
// different firm under the same id, so its link goes, as a swap's does. A
// correction of case, spacing or "Pty Ltd" keeps the link.
func TestGuestRenamedFirmLosesItsLink(t *testing.T) {
	f := newGuestFixture(t)
	link, token := f.share(t, map[string]any{"kind": "package", "package_id": f.pkg, "firm_id": f.firmA, "shared_with": "acme@example.test"})
	acme := f.guest(t, token)
	base := "/api/projects/" + f.project + "/procurement/packages/" + f.pkg + "/invitations/"
	var inv struct {
		FirmID   string `json:"firm_id"`
		FirmName string `json:"firm_name"`
	}
	live := func() int {
		return f.count(t, `SELECT count(*) FROM guest_grants WHERE org_id=$1::uuid AND id=$2::uuid AND revoked_at IS NULL`, f.m.orgID, link.Share.ID)
	}
	// The same name key: the link stays.
	f.m.call(t, http.MethodPut, base+f.firmA, []byte(`{"name":"ACME  planning."}`), http.StatusOK, &inv)
	if inv.FirmID != f.firmA || live() != 1 {
		t.Fatalf("a spelling correction moved the firm or withdrew its link: %+v, live %d", inv, live())
	}
	if code, _ := acme.raw(t, http.MethodGet, "/api/guest", nil, false); code != http.StatusOK {
		t.Fatalf("link after a spelling correction = %d", code)
	}
	// A wholly different name on the same bare firm: Zulu now sits in the
	// cell under Acme's id, so Acme's link must stop.
	f.m.call(t, http.MethodPut, base+f.firmA, []byte(`{"name":"Zulu Town Planners"}`), http.StatusOK, &inv)
	if inv.FirmID != f.firmA || inv.FirmName != "Zulu Town Planners" {
		t.Fatalf("the rename was not in place: %+v", inv)
	}
	if live() != 0 {
		t.Fatal("renaming the firm to another name left its link live")
	}
	if code, body := acme.raw(t, http.MethodGet, "/api/guest", nil, false); code != http.StatusUnauthorized || bytes.Contains([]byte(body), []byte("Zulu")) {
		t.Fatalf("Acme's link after the rename = %d %q", code, body)
	}
	quote := psample.PDF(psample.Proposal{Firm: "Acme Planning", Lines: []psample.Line{{Label: "Fee", Cents: 100}}})
	if code, _ := acme.raw(t, http.MethodPost, "/api/guest/quote?name=quote.pdf", quote, true); code != http.StatusUnauthorized {
		t.Fatalf("Acme's upload as Zulu = %d", code)
	}
	// Zulu's own new link works.
	_, zuluToken := f.share(t, map[string]any{"kind": "package", "package_id": f.pkg, "firm_id": f.firmA, "shared_with": "zulu@example.test"})
	if code, _ := f.guest(t, zuluToken).raw(t, http.MethodGet, "/api/guest", nil, false); code != http.StatusOK {
		t.Fatalf("Zulu's new link = %d", code)
	}
}

// Finding 2: a document a guest placed is filing plus the proposal read. The
// filing commit queues no visual identity, register reading or sheet split
// for it, and the full_text step's consent queue skips it; the same filing
// of a PM's document queues each. The PM's own clicks still reach it.
func TestGuestDocumentGetsNoAutomaticReads(t *testing.T) {
	f := newGuestFixture(t)
	ctx := context.Background()
	// An Open RFI makes the register job due on filing.
	f.m.call(t, http.MethodPost, "/api/projects/"+f.project+"/delivery", []byte(`{"kind":"rfi","title":"Basement wall waterproofing","status":"open","system_id":"substructure","details":{"question":"Which membrane?","raised_on":"2026-11-20"}}`), http.StatusCreated, nil)
	link, _ := f.share(t, map[string]any{"kind": "package", "package_id": f.pkg, "firm_id": f.firmA, "shared_with": "acme@example.test"})
	commit := func(t *testing.T, name, reason string) string {
		t.Helper()
		body := psample.PDF(psample.Proposal{Firm: name, Lines: []psample.Line{{Label: "Fee", Cents: 321}}})
		blob, err := f.blobs.Put(ctx, bytes.NewReader(body))
		if err != nil {
			t.Fatal(err)
		}
		in, err := f.a.store.CommitIntake(ctx, f.m.orgID, store.CommitIntake{ProjectID: f.project, SHA256: blob.SHA256, ByteSize: blob.Size, MediaType: "application/pdf", Filename: name + ".pdf", Reason: reason})
		if err != nil || !in.Created {
			t.Fatalf("intake %+v %v", in, err)
		}
		return in.DocumentID
	}
	// The guest's: held, then placed and released by OpenGuestQuote.
	guestDoc := commit(t, "Crafted drawing set", store.GuestHoldReason)
	g := store.GuestGrant{ID: link.Share.ID, OrgID: f.m.orgID, ProjectID: f.project, Kind: store.GuestPackage, PackageID: f.pkg, FirmID: f.firmA, CreatedBy: f.user}
	if q, err := f.a.store.OpenGuestQuote(ctx, g, guestDoc, true, time.Now()); err != nil || !q.Queued {
		t.Fatalf("guest quote %+v %v", q, err)
	}
	pmDoc := commit(t, "PM drawing set", "")
	// Both filed as a multi-sheet drawing, as Jev might file a crafted PDF.
	for _, doc := range []string{guestDoc, pmDoc} {
		if _, err := f.a.store.CommitFiling(ctx, f.m.orgID, doc, store.CommitFiling{PDFPages: 150, Decisions: []store.DecisionWrite{
			{Field: "kind", Value: "drawing", Band: "green", DecidedBy: "jev", QuestionVersion: "test"}}}); err != nil {
			t.Fatal(err)
		}
		if err := f.a.store.QueueConsentAfterFiling(ctx, f.m.orgID, doc); err != nil {
			t.Fatal(err)
		}
	}
	jobs := func(doc string) []string {
		rows, err := f.conn.Query(ctx, `SELECT kind FROM jobs WHERE org_id=$1::uuid AND document_id=$2::uuid AND kind NOT IN ('intake','full_text') ORDER BY kind`, f.m.orgID, doc)
		if err != nil {
			t.Fatal(err)
		}
		defer rows.Close()
		out := []string{}
		for rows.Next() {
			var k string
			if err := rows.Scan(&k); err != nil {
				t.Fatal(err)
			}
			out = append(out, k)
		}
		return out
	}
	expansion := func(doc string) string {
		var status string
		if err := f.conn.QueryRow(ctx, `SELECT status FROM drawing_expansions WHERE org_id=$1::uuid AND source_id=$2::uuid`, f.m.orgID, doc).Scan(&status); err != nil {
			t.Fatal(err)
		}
		return status
	}
	if got, want := jobs(pmDoc), []string{"consent", "register", "visual_identity"}; !reflect.DeepEqual(got, want) || expansion(pmDoc) != "pending" {
		t.Fatalf("the PM's document: jobs %v (want %v), expansion %s", got, want, expansion(pmDoc))
	}
	if got := jobs(guestDoc); len(got) != 0 {
		t.Fatalf("a guest's document queued %v", got)
	}
	if s := expansion(guestDoc); s != "review" {
		t.Fatalf("a guest's drawing set is %s, want review", s)
	}
	pending, err := f.a.store.PendingDrawingExpansions(ctx)
	if err != nil {
		t.Fatal(err)
	}
	for _, p := range pending {
		if p.DocumentID == guestDoc {
			t.Fatal("a guest's drawing set is waiting to be split")
		}
	}
	// The PM's own clicks still reach it.
	if err := f.a.store.RetryDrawingExpansion(ctx, f.m.orgID, guestDoc); err != nil || expansion(guestDoc) != "pending" {
		t.Fatalf("Retry processing on a guest's set: %v, %s", err, expansion(guestDoc))
	}
	if err := f.a.store.QueueVisualIdentity(ctx, f.m.orgID, guestDoc); err != nil {
		t.Fatal(err)
	}
	if err := f.a.store.QueueConsent(ctx, f.m.orgID, f.project, guestDoc, f.user); err != nil {
		t.Fatal(err)
	}
	if got, want := jobs(guestDoc), []string{"consent", "visual_identity"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("after the PM's clicks: %v, want %v", got, want)
	}
}

// Finding 3: a valid link's exchanges are limited per link, and the PM's
// cell hears "link opened" once an hour however often the link is opened.
func TestGuestExchangeLimitedPerLink(t *testing.T) {
	f := newGuestFixture(t)
	link, token := f.share(t, map[string]any{"kind": "package", "package_id": f.pkg, "firm_id": f.firmA, "shared_with": "acme@example.test"})
	events := func() int {
		return f.count(t, `SELECT count(*) FROM events WHERE org_id=$1::uuid AND kind=$2 AND payload::jsonb->>'state'='link_opened' AND payload::jsonb->>'package_id'=$3`, f.m.orgID, store.EventProcurement, f.pkg)
	}
	body := []byte(`{"token":"` + token + `"}`)
	browser := &guest{f: f, client: &http.Client{}}
	for i := 0; i < 30; i++ {
		if code, raw := browser.raw(t, http.MethodPost, "/api/guest/session", body, true); code != http.StatusNoContent {
			t.Fatalf("exchange %d = %d %s", i+1, code, raw)
		}
	}
	if code, _ := browser.raw(t, http.MethodPost, "/api/guest/session", body, true); code != http.StatusTooManyRequests {
		t.Fatalf("31st exchange in the hour = %d", code)
	}
	if n := events(); n != 1 {
		t.Fatalf("%d link-opened events for 30 exchanges, want 1", n)
	}
	if n := f.count(t, `SELECT opens FROM guest_grants WHERE org_id=$1::uuid AND id=$2::uuid`, f.m.orgID, link.Share.ID); n != 30 {
		t.Fatalf("opens = %d, want 30", n)
	}
	// Another link is not held by this one's limit.
	_, other := f.share(t, map[string]any{"kind": "documents", "document_ids": []string{f.d1}, "shared_with": "client@example.test"})
	if g := f.guest(t, other); g.share == "" {
		t.Fatal("another link's exchange was refused")
	}
}

// Finding 4: bytes that match a file the project already holds, other than
// this link's own quote, are answered exactly as a received quote, and
// nothing is stored, attached or changed.
func TestGuestDuplicateBytesLookLikeAReceipt(t *testing.T) {
	f := newGuestFixture(t)
	ctx := context.Background()
	_, token := f.share(t, map[string]any{"kind": "package", "package_id": f.pkg, "firm_id": f.firmA, "shared_with": "acme@example.test"})
	acme := f.guest(t, token)
	// A real new quote's answer, for its shape.
	quote := psample.PDF(psample.Proposal{Firm: "Acme Planning", Lines: []psample.Line{{Label: "Fee", Cents: 9900000}}})
	code, real := acme.raw(t, http.MethodPost, "/api/guest/quote?name=quote.pdf", quote, true)
	if code != http.StatusCreated {
		t.Fatalf("new quote = %d %s", code, real)
	}
	var sub string
	if err := f.conn.QueryRow(ctx, `SELECT id::text FROM submissions WHERE org_id=$1::uuid AND firm_id=$2::uuid AND package_id=$3::uuid`, f.m.orgID, f.firmA, f.pkg).Scan(&sub); err != nil {
		t.Fatal(err)
	}
	f.waitRead(t, sub)
	time.Sleep(300 * time.Millisecond)
	keys := func(raw string) []string {
		var m map[string]any
		if err := json.Unmarshal([]byte(raw), &m); err != nil {
			t.Fatalf("%s: %v", raw, err)
		}
		out := []string{}
		for k := range m {
			out = append(out, k)
		}
		sort.Strings(out)
		return out
	}
	stored := func(t *testing.T, sha []byte) []byte {
		t.Helper()
		blob, err := f.blobs.Open(sha)
		if err != nil {
			t.Fatal(err)
		}
		defer blob.Close()
		raw, _ := io.ReadAll(blob)
		return raw
	}
	var bravoSHA, privateSHA []byte
	if err := f.conn.QueryRow(ctx, `SELECT f.sha256 FROM submissions s JOIN documents d ON d.org_id=s.org_id AND d.id=s.document_id JOIN files f ON f.org_id=d.org_id AND f.id=d.file_id WHERE s.org_id=$1::uuid AND s.id=$2::uuid`, f.m.orgID, f.subB).Scan(&bravoSHA); err != nil {
		t.Fatal(err)
	}
	if err := f.conn.QueryRow(ctx, `SELECT f.sha256 FROM documents d JOIN files f ON f.org_id=d.org_id AND f.id=d.file_id WHERE d.org_id=$1::uuid AND d.id=$2::uuid`, f.m.orgID, f.d2).Scan(&privateSHA); err != nil {
		t.Fatal(err)
	}
	before := f.snapshot(t)
	today := store.SydneyDate(time.Now())
	// A rival's quote and the PM's own private file.
	for name, sha := range map[string][]byte{"rival.pdf": bravoSHA, "private.pdf": privateSHA} {
		code, body := acme.raw(t, http.MethodPost, "/api/guest/quote?name="+name, stored(t, sha), true)
		if code != http.StatusCreated || !reflect.DeepEqual(keys(body), keys(real)) {
			t.Fatalf("%s = %d %s, want 201 shaped as %s", name, code, body, real)
		}
		var got store.GuestQuote
		if err := json.Unmarshal([]byte(body), &got); err != nil || got.Filename != name || got.ReceivedOn != today {
			t.Fatalf("%s answered %s", name, body)
		}
	}
	after := f.snapshot(t)
	for table, rows := range before {
		if table == "jobs" || table == "guest_grants" {
			continue // the first quote's reads move on in the background
		}
		if after[table] != rows {
			t.Errorf("%s changed for a duplicate upload", table)
		}
	}
	// The firm's own repeat is still its plain receipt.
	if code, body := acme.raw(t, http.MethodPost, "/api/guest/quote?name=quote.pdf", quote, true); code != http.StatusCreated {
		t.Fatalf("own repeat = %d %s", code, body)
	}
}

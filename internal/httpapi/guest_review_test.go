package httpapi_test

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"testing"
	"time"

	"sitewise/internal/httpapi"
	psample "sitewise/internal/proposals/sample"
	"sitewise/internal/store"
)

// The security review of 18 November 2026 (M1-M3, L1-L5 and its notes).
// Each test here fails without its fix.

// M1: one cookie per browser. A page opened on link A, after link B was
// opened in another tab, may not upload to or download from link B.
func TestGuestPageIsTiedToItsLink(t *testing.T) {
	f := newGuestFixture(t)
	_, tokenA := f.share(t, map[string]any{"kind": "package", "package_id": f.pkg, "firm_id": f.firmA, "shared_with": "acme@example.test"})
	// Link B: another package's RFT, for another firm (it could be any org's).
	var other struct {
		ID string `json:"id"`
	}
	f.m.call(t, http.MethodPost, "/api/projects/"+f.project+"/procurement/rows", []byte(`{"discipline_id":"consultant.architect"}`), http.StatusCreated, &other)
	var inv struct {
		FirmID string `json:"firm_id"`
	}
	f.m.call(t, http.MethodPost, "/api/projects/"+f.project+"/procurement/packages/"+other.ID+"/invitations", []byte(`{"name":"Other Architects"}`), http.StatusCreated, &inv)
	var rfp store.Report
	f.m.call(t, http.MethodPost, "/api/projects/"+f.project+"/reports", []byte(`{"kind":"rfp","package_id":"`+other.ID+`"}`), http.StatusCreated, &rfp)
	var d store.ReportDraft
	f.m.call(t, http.MethodPost, "/api/reports/"+rfp.ID+"/draft", []byte(`{"use_last_completed":true}`), http.StatusOK, &d)
	f.issueRFT(t, d.ID, []string{f.d1})
	_, tokenB := f.share(t, map[string]any{"kind": "package", "package_id": other.ID, "firm_id": inv.FirmID, "shared_with": "other@example.test"})
	tab := f.guest(t, tokenA) // the page shows link A
	shareA := tab.share
	// Link B opened in another tab of the same browser replaces the cookie.
	body, _ := json.Marshal(map[string]string{"token": tokenB})
	if code, _ := tab.raw(t, http.MethodPost, "/api/guest/session", body, true); code != http.StatusNoContent {
		t.Fatal("second link not exchanged")
	}
	// Back in tab A: the drop carries link A's id and is refused.
	docs := f.count(t, `SELECT count(*) FROM documents WHERE org_id=$1::uuid`, f.m.orgID)
	tab.share = shareA
	quote := psample.PDF(psample.Proposal{Firm: "Acme Planning", Lines: []psample.Line{{Label: "Fee", Cents: 100}}})
	code, msg := tab.raw(t, http.MethodPost, "/api/guest/quote?name=quote.pdf", quote, true)
	if code != http.StatusConflict || !strings.Contains(msg, "This page is for another link. Reload.") {
		t.Fatalf("cross-link upload = %d %q", code, msg)
	}
	if f.count(t, `SELECT count(*) FROM documents WHERE org_id=$1::uuid`, f.m.orgID) != docs || f.count(t, `SELECT count(*) FROM project_changes WHERE org_id=$1::uuid AND via='guest'`, f.m.orgID) != 0 ||
		f.count(t, `SELECT count(*) FROM submissions WHERE org_id=$1::uuid AND package_id=$2::uuid`, f.m.orgID, other.ID) != 0 {
		t.Fatal("a cross-link upload wrote something")
	}
	if code, _ := tab.raw(t, http.MethodGet, "/api/guest/documents/"+f.d1+"/file", nil, false); code != http.StatusConflict {
		t.Fatalf("cross-link download = %d", code)
	}
	// With no link id at all, downloads and uploads are refused too.
	tab.share = ""
	if code, _ := tab.raw(t, http.MethodGet, "/api/guest/documents/"+f.d1+"/file", nil, false); code != http.StatusConflict {
		t.Fatalf("download with no link id = %d", code)
	}
	// The page reloaded on link B works on link B.
	var page store.GuestPageView
	_, raw := tab.raw(t, http.MethodGet, "/api/guest", nil, false)
	if err := json.Unmarshal([]byte(raw), &page); err != nil || page.Share == shareA {
		t.Fatalf("reloaded page %s", raw)
	}
	if code, _ := tab.raw(t, http.MethodGet, "/api/guest/documents/"+f.d1+"/file?share="+page.Share, nil, false); code != http.StatusOK {
		t.Fatalf("download on its own link = %d", code)
	}
}

// M3: a link places at most GuestQuoteLifetime quotes, ever, and a guest
// never takes the org's last tenth of its allowance (the cap case is in
// TestGuestsAreNotSeatsAndCountTowardTheCap).
func TestGuestQuoteLifetime(t *testing.T) {
	f := newGuestFixture(t)
	link, token := f.share(t, map[string]any{"kind": "package", "package_id": f.pkg, "firm_id": f.firmA, "shared_with": "acme@example.test"})
	acme := f.guest(t, token)
	// Ten placed quotes (the review's number, written out so the test does
	// not follow the constant).
	for i := 0; i < 10; i++ {
		if _, err := f.conn.Exec(context.Background(), `INSERT INTO project_changes (org_id,project_id,id,actor,via,area) VALUES ($1::uuid,$2::uuid,gen_random_uuid(),$3::uuid,'guest','procurement')`, f.m.orgID, f.project, link.Share.ID); err != nil {
			t.Fatal(err)
		}
	}
	files := f.count(t, `SELECT count(*) FROM files WHERE org_id=$1::uuid`, f.m.orgID)
	quote := psample.PDF(psample.Proposal{Firm: "Acme Planning", Lines: []psample.Line{{Label: "Fee", Cents: 100}}})
	code, msg := acme.raw(t, http.MethodPost, "/api/guest/quote?name=quote.pdf", quote, true)
	if code != http.StatusConflict || !strings.Contains(msg, "Ask the person who sent it for a new link") {
		t.Fatalf("11th quote = %d %q", code, msg)
	}
	if f.count(t, `SELECT count(*) FROM files WHERE org_id=$1::uuid`, f.m.orgID) != files {
		t.Fatal("a quote past the cap stored its bytes")
	}
	// Under the lock the same cap holds (the route's check is the cheap one).
	g := store.GuestGrant{ID: link.Share.ID, OrgID: f.m.orgID, ProjectID: f.project, Kind: store.GuestPackage, PackageID: f.pkg, FirmID: f.firmA}
	if _, err := f.a.store.OpenGuestQuote(context.Background(), g, f.d2, true, time.Now()); err != store.ErrGuestQuoteLimit {
		t.Fatalf("store past the cap = %v", err)
	}
}

// L2: misses from one address are blocked; a valid link behind the same
// address still works.
func TestGuestMissesBlockOnlyMisses(t *testing.T) {
	f := newGuestFixture(t)
	_, token := f.share(t, map[string]any{"kind": "documents", "document_ids": []string{f.d1}, "shared_with": "client@example.test"})
	client := f.guest(t, token)
	wrong := &guest{f: f, client: &http.Client{}}
	bad := []byte(`{"token":"` + strings.Repeat("ab", 32) + `"}`)
	for i := 0; i < 30; i++ {
		if code, _ := wrong.raw(t, http.MethodPost, "/api/guest/session", bad, true); code != http.StatusUnauthorized {
			t.Fatalf("miss %d = %d", i+1, code)
		}
	}
	if code, _ := wrong.raw(t, http.MethodPost, "/api/guest/session", bad, true); code != http.StatusTooManyRequests {
		t.Fatalf("31st miss = %d", code)
	}
	// A guessed token sent as a cookie is a miss too.
	cookie := http.Header{"Cookie": {httpapi.GuestCookie + "=" + strings.Repeat("ef", 32)}}
	resp := wrong.send(t, http.MethodGet, "/api/guest", nil, false, cookie)
	resp.Body.Close()
	if resp.StatusCode != http.StatusTooManyRequests {
		t.Fatalf("cookie miss after the block = %d", resp.StatusCode)
	}
	// The real link from the same address: page, download and a fresh exchange.
	if code, _ := client.raw(t, http.MethodGet, "/api/guest", nil, false); code != http.StatusOK {
		t.Fatalf("valid link behind a blocked address = %d", code)
	}
	if code, _ := client.raw(t, http.MethodGet, "/api/guest/documents/"+f.d1+"/file", nil, false); code != http.StatusOK {
		t.Fatalf("download behind a blocked address = %d", code)
	}
	if again := f.guest(t, token); again.share == "" {
		t.Fatal("exchange behind a blocked address refused")
	}
}

// The per-link read limit: 300 an hour.
func TestGuestReadsPerLink(t *testing.T) {
	f := newGuestFixture(t)
	_, token := f.share(t, map[string]any{"kind": "documents", "document_ids": []string{f.d1}, "shared_with": "client@example.test"})
	client := f.guest(t, token) // one page read already
	for i := 1; i < 300; i++ {
		if code, _ := client.raw(t, http.MethodGet, "/api/guest", nil, false); code != http.StatusOK {
			t.Fatalf("read %d = %d", i+1, code)
		}
	}
	if code, _ := client.raw(t, http.MethodGet, "/api/guest", nil, false); code != http.StatusTooManyRequests {
		t.Fatalf("301st read = %d", code)
	}
}

// Share creates per org: 100 an hour.
func TestShareCreatesPerOrg(t *testing.T) {
	f := newGuestFixture(t)
	body, _ := json.Marshal(map[string]any{"kind": "documents", "document_ids": []string{f.d1}, "shared_with": "client@example.test"})
	for i := 0; i < 100; i++ {
		f.m.call(t, http.MethodPost, "/api/projects/"+f.project+"/shares", body, http.StatusCreated, nil)
	}
	if code := f.m.status(t, http.MethodPost, "/api/projects/"+f.project+"/shares", body); code != http.StatusTooManyRequests {
		t.Fatalf("101st share = %d", code)
	}
}

// Archived, retired, revoked and wrong links get the same 401 body.
func TestGuestDeadLinksOneAnswer(t *testing.T) {
	f := newGuestFixture(t)
	_, docToken := f.share(t, map[string]any{"kind": "documents", "document_ids": []string{f.d1}, "shared_with": "client@example.test"})
	_, pkgToken := f.share(t, map[string]any{"kind": "package", "package_id": f.pkg, "firm_id": f.firmA, "shared_with": "acme@example.test"})
	answer := func(token string) (int, string) {
		return (&guest{f: f, client: &http.Client{}}).raw(t, http.MethodPost, "/api/guest/session", []byte(`{"token":"`+token+`"}`), true)
	}
	_, wrong := answer(strings.Repeat("9a", 32))
	if _, err := f.conn.Exec(context.Background(), `UPDATE packages SET retired_at=now() WHERE org_id=$1::uuid AND id=$2::uuid`, f.m.orgID, f.pkg); err != nil {
		t.Fatal(err)
	}
	code, retired := answer(pkgToken)
	if code != 401 || retired != wrong {
		t.Fatalf("retired package %d %q, wrong %q", code, retired, wrong)
	}
	f.m.call(t, http.MethodPut, "/api/projects/"+f.project+"/archived", []byte(`{"archived":true}`), http.StatusNoContent, nil)
	code, archived := answer(docToken)
	if code != 401 || archived != wrong {
		t.Fatalf("archived project %d %q, wrong %q", code, archived, wrong)
	}
}

// L3: a removed member's links stop with the membership.
func TestRemovedMembersLinksStop(t *testing.T) {
	f := newGuestFixture(t)
	ctx := context.Background()
	colleague := newUUID(t)
	if _, err := f.conn.Exec(ctx, `INSERT INTO users (org_id,id,email) VALUES ($1::uuid,$2::uuid,$3)`, f.m.orgID, colleague, colleague+"@example.test"); err != nil {
		t.Fatal(err)
	}
	if _, err := f.conn.Exec(ctx, `INSERT INTO memberships (org_id,user_id,role) VALUES ($1::uuid,$2::uuid,'member')`, f.m.orgID, colleague); err != nil {
		t.Fatal(err)
	}
	theirs, err := f.a.store.CreateGuestGrant(ctx, f.m.orgID, f.project, colleague, store.GuestGrantInput{Kind: "documents", DocumentIDs: []string{f.d1}, SharedWith: "client@example.test"}, make32(1), time.Now())
	if err != nil {
		t.Fatal(err)
	}
	mine, _ := f.share(t, map[string]any{"kind": "documents", "document_ids": []string{f.d1}, "shared_with": "other@example.test"})
	f.m.call(t, http.MethodDelete, "/api/team/members/"+colleague, nil, http.StatusNoContent, nil)
	if n := f.count(t, `SELECT count(*) FROM guest_grants WHERE org_id=$1::uuid AND id=$2::uuid AND revoked_at IS NOT NULL`, f.m.orgID, theirs.ID); n != 1 {
		t.Fatal("a removed member's link is still live")
	}
	if n := f.count(t, `SELECT count(*) FROM guest_grants WHERE org_id=$1::uuid AND id=$2::uuid AND revoked_at IS NULL`, f.m.orgID, mine.Share.ID); n != 1 {
		t.Fatal("another member's link was withdrawn")
	}
}

func make32(b byte) []byte {
	out := make([]byte, 32)
	for i := range out {
		out[i] = b
	}
	return out
}

// L4: a guest's upload left held by a crash is never filed by Resume.
func TestResumeLeavesAHeldGuestUpload(t *testing.T) {
	f := newGuestFixture(t)
	ctx := context.Background()
	body := psample.PDF(psample.Proposal{Firm: "Crashed quote", Lines: []psample.Line{{Label: "Fee", Cents: 1}}})
	blob, err := f.blobs.Put(ctx, strings.NewReader(string(body)))
	if err != nil {
		t.Fatal(err)
	}
	in, err := f.a.store.CommitIntake(ctx, f.m.orgID, store.CommitIntake{ProjectID: f.project, SHA256: blob.SHA256, ByteSize: blob.Size, MediaType: "application/pdf", Filename: "crashed.pdf", Reason: store.GuestHoldReason})
	if err != nil {
		t.Fatal(err)
	}
	hits := f.a.jevHits.Load()
	if _, err := f.a.srv.Resume(ctx); err != nil {
		t.Fatal(err)
	}
	time.Sleep(500 * time.Millisecond)
	var status string
	if err := f.conn.QueryRow(ctx, `SELECT status FROM documents WHERE org_id=$1::uuid AND id=$2::uuid`, f.m.orgID, in.DocumentID).Scan(&status); err != nil || status != store.StatusNotFiled {
		t.Fatalf("held upload after Resume: %s %v", status, err)
	}
	if f.a.jevHits.Load() != hits {
		t.Fatal("Resume sent a held guest upload to Jev")
	}
}

// L5: an award on the group a package was combined into closes the
// member's link to quotes.
func TestGuestQuoteRefusedAfterAGroupAward(t *testing.T) {
	f := newGuestFixture(t)
	ctx := context.Background()
	_, token := f.share(t, map[string]any{"kind": "package", "package_id": f.pkg, "firm_id": f.firmA, "shared_with": "acme@example.test"})
	acme := f.guest(t, token)
	var other struct {
		ID string `json:"id"`
	}
	f.m.call(t, http.MethodPost, "/api/projects/"+f.project+"/procurement/rows", []byte(`{"discipline_id":"consultant.architect"}`), http.StatusCreated, &other)
	var group struct {
		ID string `json:"id"`
	}
	body, _ := json.Marshal(map[string]any{"package_ids": []string{f.pkg, other.ID}, "title": "Design team"})
	f.m.call(t, http.MethodPost, "/api/projects/"+f.project+"/packages/combine", body, http.StatusCreated, &group)
	if group.ID == "" || group.ID == f.pkg {
		t.Fatalf("combine gave %+v", group)
	}
	prep := f.m.do(t, http.MethodPost, "/api/projects/"+f.project+"/cost-plan/prepare", []byte(`{}`))
	prep.Body.Close()
	// B's quote now sits on the group; the award is stored on the group.
	if _, err := f.conn.Exec(ctx, `UPDATE submissions SET package_id=$3::uuid WHERE org_id=$1::uuid AND id=$2::uuid`, f.m.orgID, f.subB, group.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := f.conn.Exec(ctx, `WITH item AS (INSERT INTO cost_items (org_id,project_id,id,created_in_version_id)
 SELECT org_id, project_id, gen_random_uuid(), draft_version_id FROM cost_plans WHERE org_id=$1::uuid AND project_id=$2::uuid RETURNING id)
INSERT INTO awards (org_id,id,project_id,package_id,submission_id,parent_item_id,parent_created,created_by) SELECT $1::uuid,gen_random_uuid(),$2::uuid,$3::uuid,$4::uuid,item.id,false,$5::uuid FROM item`, f.m.orgID, f.project, group.ID, f.subB, f.user); err != nil {
		t.Fatal(err)
	}
	quote := psample.PDF(psample.Proposal{Firm: "Acme Planning", Lines: []psample.Line{{Label: "Fee", Cents: 5}}})
	if code, msg := acme.raw(t, http.MethodPost, "/api/guest/quote?name="+url.QueryEscape("late.pdf"), quote, true); code != http.StatusConflict || !strings.Contains(msg, "awarded") {
		t.Fatalf("quote after a group award = %d %q", code, msg)
	}
}

// Note on transmittalIDs: only the transmittal block's ids are
// downloadable; a document id in any other block of an issue is not.
func TestGuestDownloadsOnlyTheTransmittal(t *testing.T) {
	f := newGuestFixture(t)
	_, token := f.share(t, map[string]any{"kind": "package", "package_id": f.pkg, "firm_id": f.firmA, "shared_with": "acme@example.test"})
	acme := f.guest(t, token)
	sections, _ := json.Marshal([]map[string]any{{"id": "project", "title": "Project", "blocks": []map[string]any{
		{"id": "rfp:project:information", "label": "Transmittal", "text": "", "basis": map[string]any{"document_ids": []string{f.d3}}},
		{"id": "rfp:project:other", "label": "Other", "text": "", "basis": map[string]any{"document_ids": []string{f.d2}}}}}})
	// An addendum (the RFT's second issue) with a transmittal and one more block.
	f.issueSections(t, f.rftDraft, string(sections))
	for doc, want := range map[string]int{f.d3: 200, f.d2: 404} {
		if code, _ := acme.raw(t, http.MethodGet, "/api/guest/documents/"+doc+"/file", nil, false); code != want {
			t.Fatalf("%s = %d, want %d", doc, code, want)
		}
	}
}

var _ = fmt.Sprint

// Agreement 6 of the wave index, with the project report (kind monthly, no
// package): a report link shows every issued version newest first and
// nothing of the draft or its comments, through the ordinary report link.
func TestGuestSeesTheProjectReportsIssuesOnly(t *testing.T) {
	f := newGuestFixture(t)
	f.m.ops(t, f.project, "reports", http.StatusOK, op("reports.update", map[string]any{}))
	report := projectReportID(t, f.m, f.project)
	issue := func(t *testing.T) string {
		t.Helper()
		var view store.ReportView
		f.m.getJSON(t, "/api/reports/"+report, &view)
		body, _ := json.Marshal(store.IssueOptions{Version: view.Draft.Version, ReportingDate: "2026-11-18", AcceptStale: true, StaleReason: "Guest fixture with missing profile acknowledged"})
		var issued store.IssuedReport
		f.m.call(t, http.MethodPost, "/api/reports/"+report+"/issue", body, http.StatusCreated, &issued)
		return issued.ID
	}
	first := issue(t)
	f.m.ops(t, f.project, "reports", http.StatusOK, op("reports.update", map[string]any{}))
	second := issue(t)
	// The next draft, with a comment the PM has not issued.
	f.m.ops(t, f.project, "reports", http.StatusOK, op("reports.update", map[string]any{}))
	f.m.ops(t, f.project, "reports", http.StatusOK, op("reports.comment", map[string]any{"section": "cost", "text": "Unissued draft remark about the contingency."}))
	var view store.ReportView
	f.m.getJSON(t, "/api/reports/"+report, &view)
	_, token := f.share(t, map[string]any{"kind": "report", "report_id": report, "shared_with": "client@example.test"})
	client := f.guest(t, token)
	code, raw := client.raw(t, http.MethodGet, "/api/guest", nil, false)
	var page store.GuestPageView
	if code != http.StatusOK || json.Unmarshal([]byte(raw), &page) != nil {
		t.Fatalf("page %d %s", code, raw)
	}
	if page.Kind != "report" || len(page.Issues) != 2 || page.Issues[0].ID != second || page.Issues[1].ID != first || len(page.Documents) != 0 || page.Quote != nil {
		t.Fatalf("project report page %+v", page)
	}
	if strings.Contains(raw, "Unissued draft remark") || strings.Contains(raw, view.Draft.ID) {
		t.Fatalf("the page shows the draft: %s", raw)
	}
	for path, want := range map[string]int{"/api/guest/issues/" + first + "/file": 200, "/api/guest/issues/" + second + "/file": 200, "/api/guest/issues/" + view.Draft.ID + "/file": 404} {
		if code, _ := client.raw(t, http.MethodGet, path, nil, false); code != want {
			t.Fatalf("%s = %d, want %d", path, code, want)
		}
	}
}

package httpapi_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"os"
	"reflect"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"sitewise/internal/files"
	"sitewise/internal/httpapi"
	psample "sitewise/internal/proposals/sample"
	"sitewise/internal/store"
)

// syncBuffer is the server log, read by the test while the server writes.
type syncBuffer struct {
	mu sync.Mutex
	b  bytes.Buffer
}

func (s *syncBuffer) Write(p []byte) (int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.b.Write(p)
}

func (s *syncBuffer) String() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.b.String()
}

// guestFixture is the plan's Task 7 fixture: a project with a package that
// has firms A (invited) and B (a priced quote), an issued RFT whose
// transmittal holds d1, a private d2, a sibling project and a foreign org.
type guestFixture struct {
	a                *app
	m, foreign       *member
	blobs            *files.Store
	conn             *pgx.Conn
	logs             *syncBuffer
	user             string
	project, sibling string
	foreignProject   string
	d1, d2, d3       string
	dSib, dForeign   string
	pkg              string
	firmA, firmB     string
	subB             string
	rft, rftIssue    string
	rftDraft         string
	pmp, pmpIssue    string
}

const (
	firmBName   = "Bravo Planning"
	firmBAmount = "123,456"
)

func newGuestFixture(t *testing.T, adjust ...func(*httpapi.Options)) *guestFixture {
	t.Helper()
	ctx := context.Background()
	blobs, err := files.Open(t.TempDir(), 100<<20)
	if err != nil {
		t.Fatal(err)
	}
	logs := &syncBuffer{}
	opts := append([]func(*httpapi.Options){withProfile(t), greenProposals, func(o *httpapi.Options) {
		o.Blobs = blobs
		o.MaxUploadBytes = 200 << 20
		o.Log = log.New(logs, "", 0)
	}}, adjust...)
	f := &guestFixture{a: newApp(t, opts...), blobs: blobs, logs: logs}
	f.m, f.foreign = f.a.member(t, newUUID(t)), f.a.member(t, newUUID(t))
	f.project = f.m.createProject(t, "Guest project")
	f.sibling = f.m.createProject(t, "Sibling secret project")
	f.foreignProject = f.foreign.createProject(t, "Foreign secret project")
	f.conn, err = pgx.Connect(ctx, os.Getenv("SITEWISE_TEST_DATABASE_URL"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { f.conn.Close(context.Background()) })
	if err := f.conn.QueryRow(ctx, `SELECT id::text FROM users WHERE org_id=$1::uuid LIMIT 1`, f.m.orgID).Scan(&f.user); err != nil {
		t.Fatal(err)
	}
	f.d1 = f.document(t, f.m, f.project, "A-101 Transmitted plan.pdf")
	f.d2 = f.document(t, f.m, f.project, "Private cost advice.pdf")
	f.d3 = f.document(t, f.m, f.project, "A-102 Shared section.pdf")
	f.dSib = f.document(t, f.m, f.sibling, "Sibling secret drawing.pdf")
	f.dForeign = f.document(t, f.foreign, f.foreignProject, "Foreign secret drawing.pdf")

	var pkg struct {
		ID string `json:"id"`
	}
	f.m.call(t, http.MethodPost, "/api/projects/"+f.project+"/procurement/rows", []byte(`{"discipline_id":"consultant.town_planner"}`), http.StatusCreated, &pkg)
	f.pkg = pkg.ID
	base := "/api/projects/" + f.project + "/procurement/packages/" + f.pkg
	var inv struct {
		FirmID string `json:"firm_id"`
	}
	f.m.call(t, http.MethodPost, base+"/invitations", []byte(`{"name":"Acme Planning"}`), http.StatusCreated, &inv)
	f.firmA = inv.FirmID
	f.m.call(t, http.MethodPost, base+"/invitations", []byte(`{"name":"`+firmBName+`"}`), http.StatusCreated, &inv)
	f.firmB = inv.FirmID
	var table priceTableBody
	f.m.getJSON(t, base+"/prices", &table)
	var lines []psample.Line
	for _, it := range table.Items {
		if it.Base {
			lines = append(lines, psample.Line{Label: it.Label, Cents: 12345600})
			break
		}
	}
	pdf := psample.PDF(psample.Proposal{Firm: firmBName, Lines: lines})
	var up struct {
		SubmissionID string `json:"submission_id"`
	}
	f.m.call(t, http.MethodPost, base+"/submissions?firm_id="+f.firmB+"&name="+url.QueryEscape("Bravo fee.pdf"), pdf, http.StatusCreated, &up)
	f.subB = up.SubmissionID
	f.waitRead(t, f.subB)

	// An issued PMP gives a real issued PDF; the RFT's issue reuses its bytes.
	var pmp store.Report
	f.m.call(t, http.MethodPost, "/api/projects/"+f.project+"/reports", []byte(`{"kind":"pmp"}`), http.StatusCreated, &pmp)
	f.pmp = pmp.ID
	var d store.ReportDraft
	f.m.call(t, http.MethodPost, "/api/reports/"+f.pmp+"/draft", []byte(`{"use_last_completed":true}`), http.StatusOK, &d)
	body, _ := json.Marshal(store.IssueOptions{Version: d.Version, ReportingDate: "2026-10-07", AcceptStale: true, StaleReason: "Guest fixture with missing profile acknowledged"})
	var issued store.IssuedReport
	f.m.call(t, http.MethodPost, "/api/reports/"+f.pmp+"/issue", body, http.StatusCreated, &issued)
	f.pmpIssue = issued.ID

	var rft store.Report
	f.m.call(t, http.MethodPost, "/api/projects/"+f.project+"/reports", []byte(`{"kind":"rfp","package_id":"`+f.pkg+`"}`), http.StatusCreated, &rft)
	f.rft = rft.ID
	f.m.call(t, http.MethodPost, "/api/reports/"+f.rft+"/draft", []byte(`{"use_last_completed":true}`), http.StatusOK, &d)
	f.rftIssue = d.ID
	f.issueRFT(t, f.rftIssue, []string{f.d1})
	f.m.call(t, http.MethodPost, "/api/reports/"+f.rft+"/draft", []byte(`{"use_last_completed":true}`), http.StatusOK, &d)
	f.rftDraft = d.ID
	if f.rftDraft == f.rftIssue {
		t.Fatal("the RFT's issue was reused as its draft")
	}
	return f
}

// document stores a filed PDF with real bytes, with no filing call.
func (f *guestFixture) document(t *testing.T, actor *member, project, name string) string {
	t.Helper()
	ctx := context.Background()
	blob, err := f.blobs.Put(ctx, bytes.NewReader(psample.PDF(psample.Proposal{Firm: name})))
	if err != nil {
		t.Fatal(err)
	}
	id, fileID := newUUID(t), newUUID(t)
	if err := f.a.store.CreateFile(ctx, actor.orgID, store.File{ID: fileID, ProjectID: project, SHA256: blob.SHA256, ByteSize: blob.Size, MediaType: "application/pdf"}); err != nil {
		t.Fatal(err)
	}
	if err := f.a.store.CreateDocument(ctx, actor.orgID, store.Document{ID: id, ProjectID: project, FileID: fileID, Filename: name, Status: "filed"}); err != nil {
		t.Fatal(err)
	}
	return id
}

// issueRFT marks a draft issued with a transmittal holding ids, as the issue
// route would leave it; its PDF is the PMP issue's bytes. The RFP issue
// route's clause gates are not what these tests are about.
func (f *guestFixture) issueRFT(t *testing.T, version string, ids []string) {
	t.Helper()
	sections, _ := json.Marshal([]map[string]any{{"id": "project", "title": "Project", "blocks": []map[string]any{
		{"id": "rfp:project:information", "label": "Transmittal", "text": "", "basis": map[string]any{"document_ids": ids}}}}})
	ctx := context.Background()
	if _, err := f.conn.Exec(ctx, `UPDATE report_versions v SET status='issued', sections=$3::jsonb, snapshot='{}'::jsonb, snapshot_sha256=repeat('a',64),
 export_file_sha256=(SELECT export_file_sha256 FROM report_versions WHERE org_id=$1::uuid AND id=$5::uuid), issued_at=now(), issued_by=$4::uuid
 WHERE org_id=$1::uuid AND id=$2::uuid`, f.m.orgID, version, string(sections), f.user, f.pmpIssue); err != nil {
		t.Fatal(err)
	}
	if _, err := f.conn.Exec(ctx, `UPDATE reports SET current_draft_version_id=NULL WHERE org_id=$1::uuid AND id=$2::uuid`, f.m.orgID, f.rft); err != nil {
		t.Fatal(err)
	}
}

func (f *guestFixture) waitRead(t *testing.T, sub string) {
	t.Helper()
	base := "/api/projects/" + f.project + "/procurement/packages/" + f.pkg
	deadline := time.Now().Add(15 * time.Second)
	for {
		var table priceTableBody
		f.m.getJSON(t, base+"/prices", &table)
		for _, c := range table.Columns {
			if c.Submission.ID == sub && c.Submission.Status != "reading" {
				return
			}
		}
		if time.Now().After(deadline) {
			t.Fatalf("submission %s not read", sub)
		}
		time.Sleep(50 * time.Millisecond)
	}
}

type shareCreated struct {
	Share store.ShareView `json:"share"`
	Link  string          `json:"link"`
}

// share makes a link as the PM and returns it with its token.
func (f *guestFixture) share(t *testing.T, in map[string]any) (shareCreated, string) {
	t.Helper()
	body, _ := json.Marshal(in)
	var out shareCreated
	f.m.call(t, http.MethodPost, "/api/projects/"+f.project+"/shares", body, http.StatusCreated, &out)
	_, token, ok := strings.Cut(out.Link, "/g#t=")
	if !ok || len(token) != 64 {
		t.Fatalf("link %q", out.Link)
	}
	return out, token
}

// guest is a browser holding only a guest link: no session.
type guest struct {
	f      *guestFixture
	client *http.Client
	token  string
}

func (f *guestFixture) guest(t *testing.T, token string) *guest {
	t.Helper()
	jar, err := cookiejar.New(nil)
	if err != nil {
		t.Fatal(err)
	}
	g := &guest{f: f, client: &http.Client{Jar: jar}, token: token}
	if token != "" {
		body, _ := json.Marshal(map[string]string{"token": token})
		if code, raw := g.raw(t, http.MethodPost, "/api/guest/session", body, true); code != http.StatusNoContent {
			t.Fatalf("exchange = %d: %s", code, raw)
		}
	}
	return g
}

// raw sends one request with the guest's jar; origin sets this app's Origin.
func (g *guest) raw(t *testing.T, method, path string, body []byte, origin bool) (int, string) {
	t.Helper()
	resp := g.send(t, method, path, body, origin, nil)
	defer resp.Body.Close()
	out, _ := io.ReadAll(resp.Body)
	return resp.StatusCode, string(out)
}

func (g *guest) send(t *testing.T, method, path string, body []byte, origin bool, header http.Header) *http.Response {
	t.Helper()
	req, err := http.NewRequest(method, g.f.a.url+path, bytes.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	if origin {
		req.Header.Set("Origin", g.f.a.url)
	}
	for k, v := range header {
		req.Header[k] = v
	}
	resp, err := g.client.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	return resp
}

func (f *guestFixture) count(t *testing.T, sql string, args ...any) int {
	t.Helper()
	var n int
	if err := f.conn.QueryRow(context.Background(), sql, args...).Scan(&n); err != nil {
		t.Fatal(err)
	}
	return n
}

func (f *guestFixture) snapshot(t *testing.T) map[string]string {
	t.Helper()
	out := map[string]string{}
	for _, table := range []string{"projects", "documents", "files", "packages", "package_invitations", "submissions", "submission_prices", "submission_decisions",
		"reports", "report_versions", "report_edits", "guest_grants", "project_changes", "cost_items", "cost_values", "awards", "firms", "jobs",
		"work_items", "program_activities", "profile_user_values", "chat_conversations", "chat_turns", "users", "memberships", "sessions", "invites"} {
		var raw []byte
		err := f.conn.QueryRow(context.Background(), `SELECT COALESCE(jsonb_agg(to_jsonb(r) - 'opens' - 'last_opened_at' ORDER BY to_jsonb(r)::text),'[]'::jsonb) FROM `+pgx.Identifier{table}.Sanitize()+` r WHERE org_id=ANY($1::uuid[])`, []string{f.m.orgID, f.foreign.orgID}).Scan(&raw)
		if err != nil {
			t.Fatal(table, err)
		}
		out[table] = string(raw)
	}
	return out
}

// publicRoutes answer without a session. A guest cookie must still never
// make one of them succeed.
var publicRoutes = map[string]bool{
	"POST /signup": true, "POST /signin": true, "POST /password/forgot": true, "POST /password/reset": true,
	"POST /session": true, "POST /invites/accept": true, "POST /email/verify": true, "POST /email/verify/resend": true,
	"POST /stripe/webhook": true,
}

// fill puts real ids into a route pattern: the project's own records, so a
// guest is refused for who it is, never for a malformed id.
func (f *guestFixture) fill(pattern string) string {
	_, path, _ := strings.Cut(pattern, " ")
	ids := map[string]string{"id": f.project, "pkg": f.pkg, "sub": f.subB, "r": f.rft, "version": f.rftIssue, "doc": f.d2, "firm": f.firmB}
	if strings.HasPrefix(path, "/documents/") {
		ids["id"] = f.d1
	}
	if strings.HasPrefix(path, "/team/") {
		ids["id"] = f.user
	}
	for {
		open := strings.Index(path, "{")
		if open < 0 {
			return path
		}
		end := strings.Index(path[open:], "}") + open
		name := path[open+1 : end]
		v, ok := ids[name]
		if !ok {
			v = "00000000-0000-4000-8000-000000000001"
		}
		path = path[:open] + v + path[end+1:]
	}
}

// TestGuestReachesOnlyItsGrant is the plan's Task 7 sweep.
func TestGuestReachesOnlyItsGrant(t *testing.T) {
	f := newGuestFixture(t)
	pkgLink, tokenA := f.share(t, map[string]any{"kind": "package", "package_id": f.pkg, "firm_id": f.firmA, "shared_with": "acme@example.test"})
	docLink, tokenDocs := f.share(t, map[string]any{"kind": "documents", "document_ids": []string{f.d1, f.d3}, "shared_with": "client@example.test"})
	if pkgLink.Share.ExpiresOn != store.SydneyDate(time.Now().Add(21*24*time.Hour)) || docLink.Share.ExpiresOn != store.SydneyDate(time.Now().Add(30*24*time.Hour)) {
		t.Fatalf("default lives %s and %s", pkgLink.Share.ExpiresOn, docLink.Share.ExpiresOn)
	}
	hits := f.a.jevHits.Load()
	before := f.snapshot(t)
	acme := f.guest(t, tokenA)

	// (a) Every route outside /guest, with real ids and only the guest
	// cookie, is refused.
	t.Run("member_routes_refuse_the_guest_cookie", func(t *testing.T) {
		cookie := http.Header{"Cookie": {httpapi.GuestCookie + "=" + tokenA}}
		bare := &guest{f: f, client: &http.Client{}}
		n := 0
		for _, pattern := range httpapi.RoutePatterns() {
			method, _, _ := strings.Cut(pattern, " ")
			if strings.HasPrefix(pattern, method+" /guest") {
				continue
			}
			path := "/api" + f.fill(pattern)
			resp := bare.send(t, method, path, []byte(`{}`), true, cookie)
			raw, _ := io.ReadAll(resp.Body)
			resp.Body.Close()
			code := resp.StatusCode
			if publicRoutes[pattern] {
				if code >= 200 && code < 300 {
					t.Errorf("%s with a guest cookie = %d", pattern, code)
				}
			} else if code != 401 && code != 403 && code != 404 {
				t.Errorf("%s with a guest cookie = %d: %.120s", pattern, code, raw)
			}
			n++
		}
		t.Logf("%d member and public routes refused the guest cookie", n)
	})
	if after := f.snapshot(t); !reflect.DeepEqual(before, after) {
		for table, raw := range before {
			if raw != after[table] {
				t.Errorf("refused guest requests changed %s", table)
			}
		}
	}
	if f.a.jevHits.Load() != hits {
		t.Fatal("the guest sweep called Jev")
	}

	// (b) Guest routes with ids outside the grant give 404 and say nothing.
	secrets := []string{firmBName, firmBAmount, "123456", "Private cost advice", "Sibling secret", "Foreign secret", tokenA, tokenDocs, f.subB, f.d2}
	quiet := func(t *testing.T, what, body string) {
		t.Helper()
		for _, s := range secrets {
			if strings.Contains(body, s) {
				t.Fatalf("%s disclosed %q: %.200s", what, s, body)
			}
		}
	}
	covered := map[string]bool{"POST /guest/session": true, "POST /guest/quote": true}
	probes := []struct {
		pattern, path string
		want          int
	}{
		{"GET /guest", "/api/guest", 200},
		{"GET /guest/documents/{doc}/file", "/api/guest/documents/" + f.d1 + "/file", 200},
		{"GET /guest/documents/{doc}/file", "/api/guest/documents/" + f.d2 + "/file", 404},
		{"GET /guest/documents/{doc}/file", "/api/guest/documents/" + f.d3 + "/file", 404},
		{"GET /guest/documents/{doc}/file", "/api/guest/documents/" + f.dSib + "/file", 404},
		{"GET /guest/documents/{doc}/file", "/api/guest/documents/" + f.dForeign + "/file", 404},
		{"GET /guest/documents/{doc}/file", "/api/guest/documents/" + newUUID(t) + "/file", 404},
		{"GET /guest/documents/{doc}/file", "/api/guest/documents/not-a-uuid/file", 404},
		{"GET /guest/issues/{version}/file", "/api/guest/issues/" + f.rftIssue + "/file", 200},
		{"GET /guest/issues/{version}/file", "/api/guest/issues/" + f.rftDraft + "/file", 404},
		{"GET /guest/issues/{version}/file", "/api/guest/issues/" + f.pmpIssue + "/file", 404},
		{"GET /guest/issues/{version}/file", "/api/guest/issues/" + newUUID(t) + "/file", 404},
	}
	for _, p := range probes {
		covered[p.pattern] = true
		code, body := acme.raw(t, http.MethodGet, p.path, nil, false)
		if code != p.want {
			t.Fatalf("%s = %d, want %d: %.200s", p.path, code, p.want, body)
		}
		if !strings.HasPrefix(body, "%PDF") {
			quiet(t, p.path, body)
		}
	}
	for _, pattern := range httpapi.RoutePatterns() {
		method, _, _ := strings.Cut(pattern, " ")
		if strings.HasPrefix(pattern, method+" /guest") && !covered[pattern] {
			t.Errorf("%s has no case in the guest sweep", pattern)
		}
	}
	var page store.GuestPageView
	_, body := acme.raw(t, http.MethodGet, "/api/guest", nil, false)
	if err := json.Unmarshal([]byte(body), &page); err != nil {
		t.Fatal(err)
	}
	if page.Kind != "package" || len(page.Documents) != 1 || page.Documents[0].ID != f.d1 || len(page.Issues) != 1 || page.Issues[0].ID != f.rftIssue || page.Quote == nil || page.To != "acme@example.test" || page.Project != "Guest project" {
		t.Fatalf("package page %+v", page)
	}

	// (c) A documents link takes no quote and opens no issue; it opens its own.
	client := f.guest(t, tokenDocs)
	if code, _ := client.raw(t, http.MethodPost, "/api/guest/quote?name=quote.pdf", []byte("%PDF-1.4"), true); code != 404 {
		t.Fatalf("documents link took a quote: %d", code)
	}
	for path, want := range map[string]int{"/api/guest/documents/" + f.d1 + "/file": 200, "/api/guest/documents/" + f.d3 + "/file": 200, "/api/guest/documents/" + f.d2 + "/file": 404, "/api/guest/issues/" + f.rftIssue + "/file": 404} {
		if code, body := client.raw(t, http.MethodGet, path, nil, false); code != want {
			t.Fatalf("documents link %s = %d, want %d: %.100s", path, code, want, body)
		}
	}

	// (e) A guest write from another origin is refused.
	if code, _ := acme.raw(t, http.MethodPost, "/api/guest/quote?name=quote.pdf", []byte("%PDF-1.4"), false); code != 403 {
		t.Fatalf("cross-origin quote = %d", code)
	}
	if code, _ := (&guest{f: f, client: &http.Client{}}).raw(t, http.MethodPost, "/api/guest/session", []byte(`{"token":"`+tokenA+`"}`), false); code != 403 {
		t.Fatalf("cross-origin exchange = %d", code)
	}

	// (d) Wrong, expired and revoked links get one plain answer, and the
	// cookie is cleared.
	gone := func(t *testing.T, g *guest, what string) string {
		t.Helper()
		resp := g.send(t, http.MethodGet, "/api/guest", nil, false, nil)
		raw, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		if resp.StatusCode != 401 || !strings.Contains(resp.Header.Get("Set-Cookie"), httpapi.GuestCookie+"=;") || resp.Header.Get("Referrer-Policy") != "no-referrer" {
			t.Fatalf("%s: %d %q %q", what, resp.StatusCode, resp.Header.Get("Set-Cookie"), raw)
		}
		return string(raw)
	}
	f.m.call(t, http.MethodDelete, "/api/projects/"+f.project+"/shares/"+pkgLink.Share.ID, nil, http.StatusOK, nil)
	revoked := gone(t, acme, "revoked")
	if _, err := f.conn.Exec(context.Background(), `UPDATE guest_grants SET expires_at=now()-interval '1 minute' WHERE org_id=$1::uuid AND id=$2::uuid`, f.m.orgID, docLink.Share.ID); err != nil {
		t.Fatal(err)
	}
	expired := gone(t, client, "expired")
	wrong := &guest{f: f, client: &http.Client{}}
	code, wrongExchange := wrong.raw(t, http.MethodPost, "/api/guest/session", []byte(`{"token":"`+strings.Repeat("ab", 32)+`"}`), true)
	if code != 401 || revoked != expired || expired != wrongExchange || !strings.Contains(revoked, "expired or been withdrawn") {
		t.Fatalf("answers differ: %d %q %q %q", code, revoked, expired, wrongExchange)
	}
	if code, _ := acme.raw(t, http.MethodPost, "/api/guest/session", []byte(`{"token":"`+tokenA+`"}`), true); code != 401 {
		t.Fatalf("a revoked link exchanged again: %d", code)
	}
	if logs := f.logs.String(); strings.Contains(logs, tokenA) || strings.Contains(logs, tokenDocs) {
		t.Fatal("a token reached the server log")
	}
	if f.a.jevHits.Load() != hits {
		t.Fatal("guest reads called Jev")
	}
}

// A share is the PM's: its routes refuse the signed out, another origin,
// another org and another project, and a target outside the project.
func TestGuestGrantScope(t *testing.T) {
	f := newGuestFixture(t)
	base := "/api/projects/" + f.project + "/shares"
	grants := func() int {
		return f.count(t, `SELECT count(*) FROM guest_grants WHERE org_id=ANY($1::uuid[])`, []string{f.m.orgID, f.foreign.orgID})
	}
	// Another package with an invited firm and no issued RFP.
	var other struct {
		ID string `json:"id"`
	}
	f.m.call(t, http.MethodPost, "/api/projects/"+f.project+"/procurement/rows", []byte(`{"discipline_id":"consultant.architect"}`), http.StatusCreated, &other)
	var inv struct {
		FirmID string `json:"firm_id"`
	}
	f.m.call(t, http.MethodPost, "/api/projects/"+f.project+"/procurement/packages/"+other.ID+"/invitations", []byte(`{"name":"Unissued Architects"}`), http.StatusCreated, &inv)
	var unissued store.Report
	f.m.call(t, http.MethodPost, "/api/projects/"+f.project+"/reports", []byte(`{"kind":"pmp"}`), http.StatusCreated, &unissued)
	refused := map[string]map[string]any{
		"sibling document":     {"kind": "documents", "document_ids": []string{f.d1, f.dSib}, "shared_with": "x@example.test"},
		"foreign document":     {"kind": "documents", "document_ids": []string{f.dForeign}, "shared_with": "x@example.test"},
		"random document":      {"kind": "documents", "document_ids": []string{newUUID(t)}, "shared_with": "x@example.test"},
		"report with no issue": {"kind": "report", "report_id": unissued.ID, "shared_with": "x@example.test"},
		"uninvited firm":       {"kind": "package", "package_id": f.pkg, "firm_id": newUUID(t), "shared_with": "x@example.test"},
		"unissued RFP":         {"kind": "package", "package_id": other.ID, "firm_id": inv.FirmID, "shared_with": "x@example.test"},
		"another org's report": {"kind": "report", "report_id": newUUID(t), "shared_with": "x@example.test"},
	}
	for name, body := range refused {
		raw, _ := json.Marshal(body)
		if got := f.m.status(t, http.MethodPost, base, raw); got != http.StatusNotFound {
			t.Errorf("%s = %d, want 404", name, got)
		}
	}
	for name, body := range map[string]string{
		"mixed kind":  `{"kind":"documents","document_ids":["` + f.d1 + `"],"report_id":"` + f.pmp + `","shared_with":"x@example.test"}`,
		"cost plan":   `{"kind":"cost_plan","shared_with":"x@example.test"}`,
		"no one":      `{"kind":"report","report_id":"` + f.pmp + `","shared_with":"  "}`,
		"181 days":    `{"kind":"report","report_id":"` + f.pmp + `","shared_with":"x@example.test","days":181}`,
		"0 is not -1": `{"kind":"report","report_id":"` + f.pmp + `","shared_with":"x@example.test","days":-1}`,
		"unknown key": `{"kind":"report","report_id":"` + f.pmp + `","shared_with":"x@example.test","respond":true}`,
	} {
		if got := f.m.status(t, http.MethodPost, base, []byte(body)); got != http.StatusBadRequest {
			t.Errorf("%s = %d, want 400", name, got)
		}
	}
	if n := grants(); n != 0 {
		t.Fatalf("refused shares wrote %d grants", n)
	}
	ok, _ := f.share(t, map[string]any{"kind": "report", "report_id": f.pmp, "shared_with": "client@example.test", "days": 1})
	body, _ := json.Marshal(map[string]any{"kind": "report", "report_id": f.pmp, "shared_with": "client@example.test"})
	for name, status := range map[string]int{
		"signed out":     (&member{app: f.a, client: &http.Client{}}).status(t, http.MethodPost, base, body),
		"foreign org":    f.foreign.status(t, http.MethodPost, base, body),
		"sibling":        f.m.status(t, http.MethodPost, "/api/projects/"+f.sibling+"/shares", body),
		"list foreign":   f.foreign.status(t, http.MethodGet, base, nil),
		"revoke foreign": f.foreign.status(t, http.MethodDelete, base+"/"+ok.Share.ID, nil),
		"revoke sibling": f.m.status(t, http.MethodDelete, "/api/projects/"+f.sibling+"/shares/"+ok.Share.ID, nil),
	} {
		want := http.StatusNotFound
		if name == "signed out" {
			want = http.StatusUnauthorized
		}
		if status != want {
			t.Errorf("%s = %d, want %d", name, status, want)
		}
	}
	req, _ := http.NewRequest(http.MethodPost, f.a.url+base, bytes.NewReader(body))
	req.Header.Set("Origin", "https://untrusted.example")
	if resp, err := f.m.client.Do(req); err != nil || resp.StatusCode != http.StatusForbidden {
		t.Fatalf("wrong origin: %v %v", resp, err)
	}
	var list struct {
		Shares []store.ShareView `json:"shares"`
		Issued bool              `json:"issued"`
	}
	f.m.getJSON(t, base+"?report="+f.pmp, &list)
	if len(list.Shares) != 1 || !list.Issued || list.Shares[0].SharedBy == "" || list.Shares[0].Opens != 0 || !list.Shares[0].Live {
		t.Fatalf("list %+v", list)
	}
	raw := f.m.do(t, http.MethodGet, base, nil)
	listed, _ := io.ReadAll(raw.Body)
	raw.Body.Close()
	if strings.Contains(string(listed), "token") {
		t.Fatalf("the list holds a token field: %s", listed)
	}
	if n := grants(); n != 1 {
		t.Fatalf("%d grants after the refusals, want 1", n)
	}
}

// The table's own rules: one grant of each kind inserts; a mixed kind is
// refused; removing the invitation removes the firm's link; correcting the
// firm's name keeps it on the corrected firm.
func TestGuestGrantLife(t *testing.T) {
	f := newGuestFixture(t)
	ctx := context.Background()
	link, token := f.share(t, map[string]any{"kind": "package", "package_id": f.pkg, "firm_id": f.firmA, "shared_with": "acme@example.test"})
	f.share(t, map[string]any{"kind": "report", "report_id": f.pmp, "shared_with": "client@example.test"})
	f.share(t, map[string]any{"kind": "documents", "document_ids": []string{f.d1}, "shared_with": "client@example.test"})
	_, err := f.conn.Exec(ctx, `INSERT INTO guest_grants (org_id,id,project_id,token_hash,kind,shared_with,document_ids,report_id,created_by,expires_at)
VALUES ($1::uuid,gen_random_uuid(),$2::uuid,decode(repeat('ee',32),'hex'),'documents','x',ARRAY[$3::uuid],$4::uuid,$5::uuid,now()+interval '1 day')`, f.m.orgID, f.project, f.d1, f.pmp, f.user)
	if err == nil || !strings.Contains(err.Error(), "guest_grants_check") {
		t.Fatalf("mixed kind inserted: %v", err)
	}
	// A new link for the same thing replaces the old one in one step.
	replaced, newToken := f.share(t, map[string]any{"kind": "package", "package_id": f.pkg, "firm_id": f.firmA, "shared_with": "acme@example.test", "replaces": link.Share.ID})
	if code, _ := (&guest{f: f, client: &http.Client{}}).raw(t, http.MethodPost, "/api/guest/session", []byte(`{"token":"`+token+`"}`), true); code != 401 {
		t.Fatalf("the replaced link still works: %d", code)
	}
	acme := f.guest(t, newToken)
	// Correcting the firm's name keeps the link on its cell.
	base := "/api/projects/" + f.project + "/procurement/packages/" + f.pkg + "/invitations/"
	var inv struct {
		FirmID string `json:"firm_id"`
	}
	f.m.call(t, http.MethodPut, base+f.firmA, []byte(`{"name":"Acme Planning Pty Ltd"}`), http.StatusOK, &inv)
	if n := f.count(t, `SELECT count(*) FROM guest_grants WHERE org_id=$1::uuid AND id=$2::uuid AND firm_id=$3::uuid`, f.m.orgID, replaced.Share.ID, inv.FirmID); n != 1 {
		t.Fatal("the corrected firm lost its link")
	}
	if code, _ := acme.raw(t, http.MethodGet, "/api/guest", nil, false); code != 200 {
		t.Fatalf("link after a correction = %d", code)
	}
	// Removing the invitation takes the link with it.
	f.m.call(t, http.MethodDelete, base+inv.FirmID, nil, http.StatusNoContent, nil)
	if n := f.count(t, `SELECT count(*) FROM guest_grants WHERE org_id=$1::uuid AND kind='package'`, f.m.orgID); n != 0 {
		t.Fatalf("%d package links outlived their invitation", n)
	}
	if code, _ := acme.raw(t, http.MethodGet, "/api/guest", nil, false); code != 401 {
		t.Fatalf("link after its invitation went = %d", code)
	}
	// An archived project pauses its links; unarchived, they work again.
	docs, docToken := f.share(t, map[string]any{"kind": "documents", "document_ids": []string{f.d3}, "shared_with": "client@example.test"})
	client := f.guest(t, docToken)
	f.m.call(t, http.MethodPut, "/api/projects/"+f.project+"/archived", []byte(`{"archived":true}`), http.StatusNoContent, nil)
	if code, _ := client.raw(t, http.MethodGet, "/api/guest", nil, false); code != 401 {
		t.Fatalf("archived project's link = %d", code)
	}
	f.m.call(t, http.MethodPut, "/api/projects/"+f.project+"/archived", []byte(`{"archived":false}`), http.StatusNoContent, nil)
	client = f.guest(t, docToken)
	if code, _ := client.raw(t, http.MethodGet, "/api/guest", nil, false); code != 200 {
		t.Fatalf("unarchived project's link = %d", code)
	}
	var views struct {
		Shares []store.ShareView `json:"shares"`
	}
	f.m.getJSON(t, "/api/projects/"+f.project+"/shares?document="+f.d3, &views)
	if len(views.Shares) != 1 || views.Shares[0].ID != docs.Share.ID || views.Shares[0].Opens != 2 || views.Shares[0].LastOpenedOn == "" {
		t.Fatalf("opens not recorded: %+v", views.Shares)
	}
	// A deleted document drops off the link.
	f.m.call(t, http.MethodPost, "/api/projects/"+f.project+"/documents/delete", []byte(`{"document_ids":["`+f.d3+`"]}`), http.StatusOK, nil)
	if code, _ := client.raw(t, http.MethodGet, "/api/guest/documents/"+f.d3+"/file", nil, false); code != 404 {
		t.Fatalf("deleted document = %d", code)
	}
	var page store.GuestPageView
	_, body := client.raw(t, http.MethodGet, "/api/guest", nil, false)
	if json.Unmarshal([]byte(body), &page) != nil || len(page.Documents) != 0 {
		t.Fatalf("page after delete %s", body)
	}
}

// TestGuestFetchRules: a report link shows every issued version newest
// first and never a draft; an RFT re-issue reaches the firm's link, and a
// document from an earlier issue's transmittal stays downloadable.
func TestGuestFetchRules(t *testing.T) {
	f := newGuestFixture(t)
	_, token := f.share(t, map[string]any{"kind": "package", "package_id": f.pkg, "firm_id": f.firmA, "shared_with": "acme@example.test"})
	acme := f.guest(t, token)
	f.issueRFT(t, f.rftDraft, []string{f.d3})
	var page store.GuestPageView
	_, body := acme.raw(t, http.MethodGet, "/api/guest", nil, false)
	if err := json.Unmarshal([]byte(body), &page); err != nil {
		t.Fatal(err)
	}
	if len(page.Issues) != 2 || page.Issues[0].ID != f.rftDraft || page.Issues[1].ID != f.rftIssue || len(page.Documents) != 1 || page.Documents[0].ID != f.d3 {
		t.Fatalf("addendum page %+v", page)
	}
	for _, d := range []string{f.d1, f.d3} {
		if code, _ := acme.raw(t, http.MethodGet, "/api/guest/documents/"+d+"/file", nil, false); code != 200 {
			t.Fatalf("transmittal document %s = %d", d, code)
		}
	}
	if code, _ := acme.raw(t, http.MethodGet, "/api/guest/documents/"+f.d2+"/file", nil, false); code != 404 {
		t.Fatal("a document outside every transmittal was served")
	}
	// A report link: issued versions only.
	_, reportToken := f.share(t, map[string]any{"kind": "report", "report_id": f.pmp, "shared_with": "client@example.test"})
	client := f.guest(t, reportToken)
	var d store.ReportDraft
	f.m.call(t, http.MethodPost, "/api/reports/"+f.pmp+"/draft", []byte(`{"use_last_completed":true}`), http.StatusOK, &d)
	_, body = client.raw(t, http.MethodGet, "/api/guest", nil, false)
	page = store.GuestPageView{}
	if err := json.Unmarshal([]byte(body), &page); err != nil || len(page.Issues) != 1 || page.Issues[0].ID != f.pmpIssue || page.Quote != nil || len(page.Documents) != 0 {
		t.Fatalf("report page %s", body)
	}
	for path, want := range map[string]int{"/api/guest/issues/" + f.pmpIssue + "/file": 200, "/api/guest/issues/" + d.ID + "/file": 404, "/api/guest/issues/" + f.rftIssue + "/file": 404} {
		if code, _ := client.raw(t, http.MethodGet, path, nil, false); code != want {
			t.Fatalf("%s = %d, want %d", path, code, want)
		}
	}
	if code, _ := client.raw(t, http.MethodPost, "/api/guest/quote?name=q.pdf", []byte("%PDF"), true); code != 404 {
		t.Fatal("a report link took a quote")
	}
}

func boardCard(t *testing.T, f *guestFixture, m *member, project, pkg, firm string) map[string]any {
	t.Helper()
	var board struct {
		Packages map[string][]map[string]any `json:"packages"`
	}
	m.getJSON(t, "/api/projects/"+project+"/procurement/board", &board)
	for _, c := range board.Packages[pkg] {
		if c["firm_id"] == firm {
			return c
		}
	}
	return nil
}

// TestOpenGuestQuote: Acme's quote lands on Acme's invited cell, is filed and
// read as a PM drop is, is recorded as a change no member can undo, and a
// revoked or awarded link takes nothing.
func TestOpenGuestQuote(t *testing.T) {
	f := newGuestFixture(t)
	link, token := f.share(t, map[string]any{"kind": "package", "package_id": f.pkg, "firm_id": f.firmA, "shared_with": "acme@example.test"})
	if c := boardCard(t, f, f.m, f.project, f.pkg, f.firmA); c == nil || c["linked"] != true || c["link_opened_on"] != nil || c["tender_issued"] != true {
		t.Fatalf("invited card before the link was opened %+v", c)
	}
	acme := f.guest(t, token)
	if c := boardCard(t, f, f.m, f.project, f.pkg, f.firmA); c == nil || c["link_opened_on"] != store.SydneyDate(time.Now()) {
		t.Fatalf("invited card after the link was opened %+v", c)
	}
	quote := psample.PDF(psample.Proposal{Firm: "Acme Planning", Lines: []psample.Line{{Label: "Fee", Cents: 9900000}}})
	code, body := acme.raw(t, http.MethodPost, "/api/guest/quote?name="+url.QueryEscape("quote.pdf"), quote, true)
	if code != http.StatusCreated || !strings.Contains(body, `"filename":"quote.pdf"`) || strings.Contains(body, "submission") || strings.Contains(body, "document") {
		t.Fatalf("quote = %d %s", code, body)
	}
	var sub string
	if err := f.conn.QueryRow(context.Background(), `SELECT id::text FROM submissions WHERE org_id=$1::uuid AND firm_id=$2::uuid AND package_id=$3::uuid`, f.m.orgID, f.firmA, f.pkg).Scan(&sub); err != nil {
		t.Fatal(err)
	}
	f.waitRead(t, sub)
	if c := boardCard(t, f, f.m, f.project, f.pkg, f.firmA); c == nil || c["submission_id"] != sub || c["firm_name"] != "Acme Planning" {
		t.Fatalf("Acme's cell after its quote %+v", c)
	}
	var change string
	if err := f.conn.QueryRow(context.Background(), `SELECT id::text FROM project_changes WHERE org_id=$1::uuid AND project_id=$2::uuid AND via='guest' AND actor=$3::uuid AND area='procurement' AND ops->0->>'op'='guest_quote' AND ops->0->>'submission_id'=$4`, f.m.orgID, f.project, link.Share.ID, sub).Scan(&change); err != nil {
		t.Fatalf("no change row: %v", err)
	}
	f.m.call(t, http.MethodPost, "/api/projects/"+f.project+"/changes/"+change+"/undo", nil, http.StatusNotFound, nil)
	var page store.GuestPageView
	_, body = acme.raw(t, http.MethodGet, "/api/guest", nil, false)
	if json.Unmarshal([]byte(body), &page) != nil || page.Quote == nil || len(page.Quote.Received) != 1 || page.Quote.Received[0].Filename != "quote.pdf" {
		t.Fatalf("received %s", body)
	}
	// The same bytes again are the same quote; B's bytes are refused and B's
	// quote is untouched.
	if code, _ := acme.raw(t, http.MethodPost, "/api/guest/quote?name=quote.pdf", quote, true); code != http.StatusCreated {
		t.Fatalf("same quote again = %d", code)
	}
	var bravoBytes []byte
	var bravo string
	if err := f.conn.QueryRow(context.Background(), `SELECT f.sha256, to_jsonb(s)::text FROM submissions s JOIN documents d ON d.org_id=s.org_id AND d.id=s.document_id JOIN files f ON f.org_id=d.org_id AND f.id=d.file_id WHERE s.org_id=$1::uuid AND s.id=$2::uuid`, f.m.orgID, f.subB).Scan(&bravoBytes, &bravo); err != nil {
		t.Fatal(err)
	}
	blob, err := f.blobs.Open(bravoBytes)
	if err != nil {
		t.Fatal(err)
	}
	raw, _ := io.ReadAll(blob)
	blob.Close()
	if code, body := acme.raw(t, http.MethodPost, "/api/guest/quote?name=mine.pdf", raw, true); code != http.StatusConflict || strings.Contains(body, firmBName) {
		t.Fatalf("another firm's bytes = %d %s", code, body)
	}
	var after string
	if err := f.conn.QueryRow(context.Background(), `SELECT to_jsonb(s)::text FROM submissions s WHERE s.org_id=$1::uuid AND s.id=$2::uuid`, f.m.orgID, f.subB).Scan(&after); err != nil || after != bravo {
		t.Fatalf("B's quote changed: %v", err)
	}
	// Awarded: refused, and the page says so.
	// The award route's line rules are not what this is about: a live award
	// row on B is what the guest route checks.
	prep := f.m.do(t, http.MethodPost, "/api/projects/"+f.project+"/cost-plan/prepare", []byte(`{}`))
	prep.Body.Close()
	if _, err := f.conn.Exec(context.Background(), `WITH item AS (INSERT INTO cost_items (org_id,project_id,id,created_in_version_id)
 SELECT org_id, project_id, gen_random_uuid(), draft_version_id FROM cost_plans WHERE org_id=$1::uuid AND project_id=$2::uuid RETURNING id)
INSERT INTO awards (org_id,id,project_id,package_id,submission_id,parent_item_id,parent_created,created_by) SELECT $1::uuid,gen_random_uuid(),$2::uuid,$3::uuid,$4::uuid,item.id,false,$5::uuid FROM item`, f.m.orgID, f.project, f.pkg, f.subB, f.user); err != nil {
		t.Fatal(err)
	}
	other := psample.PDF(psample.Proposal{Firm: "Acme Planning", Lines: []psample.Line{{Label: "Fee", Cents: 9800000}}})
	if code, body := acme.raw(t, http.MethodPost, "/api/guest/quote?name=late.pdf", other, true); code != http.StatusConflict || !strings.Contains(body, "awarded") {
		t.Fatalf("quote after award = %d %s", code, body)
	}
	if n := f.count(t, `SELECT count(*) FROM documents WHERE org_id=$1::uuid AND filename='late.pdf'`, f.m.orgID); n != 0 {
		t.Fatal("a refused quote left its document")
	}
	_, body = acme.raw(t, http.MethodGet, "/api/guest", nil, false)
	if !strings.Contains(body, "This tender has been awarded.") {
		t.Fatalf("page after award %s", body)
	}
}

// TestGuestRevokedMidUpload: the link is withdrawn while the bytes land.
// Nothing is placed, the new document is deleted before any filing, and no
// Jev call is made for it.
func TestGuestRevokedMidUpload(t *testing.T) {
	f := newGuestFixture(t)
	link, token := f.share(t, map[string]any{"kind": "package", "package_id": f.pkg, "firm_id": f.firmA, "shared_with": "acme@example.test"})
	acme := f.guest(t, token)
	hits := f.a.jevHits.Load()
	pr, pw := io.Pipe()
	req, _ := http.NewRequest(http.MethodPost, f.a.url+"/api/guest/quote?name=slow.pdf", pr)
	req.Header.Set("Origin", f.a.url)
	done := make(chan *http.Response, 1)
	go func() {
		resp, err := acme.client.Do(req)
		if err != nil {
			done <- nil
			return
		}
		done <- resp
	}()
	quote := psample.PDF(psample.Proposal{Firm: "Acme Planning", Lines: []psample.Line{{Label: "Fee", Cents: 7700000}}})
	_, _ = pw.Write(quote[:len(quote)/2])
	time.Sleep(300 * time.Millisecond)
	f.m.call(t, http.MethodDelete, "/api/projects/"+f.project+"/shares/"+link.Share.ID, nil, http.StatusOK, nil)
	_, _ = pw.Write(quote[len(quote)/2:])
	pw.Close()
	resp := <-done
	if resp == nil {
		t.Fatal("upload failed")
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("revoked mid-upload = %d", resp.StatusCode)
	}
	for table, sql := range map[string]string{
		"documents":   `SELECT count(*) FROM documents WHERE org_id=$1::uuid AND filename='slow.pdf'`,
		"submissions": `SELECT count(*) FROM submissions WHERE org_id=$1::uuid AND filename='slow.pdf'`,
		"jobs":        `SELECT count(*) FROM jobs j WHERE j.org_id=$1::uuid AND NOT EXISTS(SELECT 1 FROM documents d WHERE d.org_id=j.org_id AND d.id=j.document_id)`,
		"changes":     `SELECT count(*) FROM project_changes WHERE org_id=$1::uuid AND via='guest'`,
	} {
		if n := f.count(t, sql, f.m.orgID); n != 0 {
			t.Errorf("%s left: %d", table, n)
		}
	}
	time.Sleep(200 * time.Millisecond)
	if f.a.jevHits.Load() != hits {
		t.Fatal("a refused quote reached Jev")
	}
}

// TestGuestQuoteIsAPMDrop: one file dropped by the PM on an invited cell and
// by the firm's guest on a twin project: the same Jev calls, the same
// decisions and the same card.
func TestGuestQuoteIsAPMDrop(t *testing.T) {
	f := newGuestFixture(t)
	_, token := f.share(t, map[string]any{"kind": "package", "package_id": f.pkg, "firm_id": f.firmA, "shared_with": "acme@example.test"})
	acme := f.guest(t, token)
	// The twin: the same row and invited firm on the sibling project.
	var twin struct {
		ID string `json:"id"`
	}
	f.m.call(t, http.MethodPost, "/api/projects/"+f.sibling+"/procurement/rows", []byte(`{"discipline_id":"consultant.town_planner"}`), http.StatusCreated, &twin)
	twinBase := "/api/projects/" + f.sibling + "/procurement/packages/" + twin.ID
	var inv struct {
		FirmID string `json:"firm_id"`
	}
	f.m.call(t, http.MethodPost, twinBase+"/invitations", []byte(`{"firm_id":"`+f.firmA+`"}`), http.StatusCreated, &inv)
	quote := psample.PDF(psample.Proposal{Firm: "Acme Planning", Lines: []psample.Line{{Label: "Fee", Cents: 9900000}}})
	wait := func(project, pkg string) map[string]any {
		deadline := time.Now().Add(15 * time.Second)
		for {
			c := boardCard(t, f, f.m, project, pkg, f.firmA)
			if c != nil && c["status"] != "reading" && c["status"] != "invited" {
				return c
			}
			if time.Now().After(deadline) {
				t.Fatalf("not read: %+v", c)
			}
			time.Sleep(50 * time.Millisecond)
		}
	}
	start := f.a.jevHits.Load()
	f.m.call(t, http.MethodPost, twinBase+"/submissions?firm_id="+f.firmA+"&name=quote.pdf", quote, http.StatusCreated, nil)
	pm := wait(f.sibling, twin.ID)
	time.Sleep(200 * time.Millisecond)
	pmCalls := f.a.jevHits.Load() - start
	start = f.a.jevHits.Load()
	if code, body := acme.raw(t, http.MethodPost, "/api/guest/quote?name=quote.pdf", quote, true); code != http.StatusCreated {
		t.Fatalf("guest quote = %d %s", code, body)
	}
	g := wait(f.project, f.pkg)
	time.Sleep(200 * time.Millisecond)
	guestCalls := f.a.jevHits.Load() - start
	if pmCalls != guestCalls {
		t.Fatalf("Jev calls: PM drop %d, guest %d", pmCalls, guestCalls)
	}
	for _, k := range []string{"status", "firm_name", "base_total", "stated_total", "needs_review", "filename", "flags"} {
		if fmt.Sprint(pm[k]) != fmt.Sprint(g[k]) {
			t.Errorf("%s: PM drop %v, guest %v", k, pm[k], g[k])
		}
	}
	decisions := func(project string) string {
		var out string
		if err := f.conn.QueryRow(context.Background(), `SELECT string_agg(d.field||'='||d.decided_by||'/'||d.band, ',' ORDER BY d.field) FROM submission_decisions d JOIN submissions s ON s.org_id=d.org_id AND s.id=d.submission_id
WHERE s.org_id=$1::uuid AND s.project_id=$2::uuid AND s.firm_id=$3::uuid AND s.filename='quote.pdf'`, f.m.orgID, project, f.firmA).Scan(&out); err != nil {
			t.Fatal(err)
		}
		return out
	}
	if a, b := decisions(f.sibling), decisions(f.project); a != b || !strings.Contains(a, "firm_name=user/green") {
		t.Fatalf("decisions differ:\nPM    %s\nguest %s", a, b)
	}
}

// TestGuestUploadLimits: the type by name before a byte is read, 50 MB,
// 20 a link an hour.
func TestGuestUploadLimits(t *testing.T) {
	f := newGuestFixture(t)
	_, token := f.share(t, map[string]any{"kind": "package", "package_id": f.pkg, "firm_id": f.firmA, "shared_with": "acme@example.test"})
	acme := f.guest(t, token)
	files := func() int { return f.count(t, `SELECT count(*) FROM files WHERE org_id=$1::uuid`, f.m.orgID) }
	before := files()
	if code, body := acme.raw(t, http.MethodPost, "/api/guest/quote?name=setup.exe", []byte("MZ"), true); code != 400 || !strings.Contains(body, "PDF, Word or Excel") {
		t.Fatalf(".exe = %d %s", code, body)
	}
	if code, _ := acme.raw(t, http.MethodPost, "/api/guest/quote", []byte("%PDF"), true); code != 400 {
		t.Fatalf("no name = %d", code)
	}
	big := bytes.Repeat([]byte("x"), 51<<20)
	if code, body := acme.raw(t, http.MethodPost, "/api/guest/quote?name=huge.pdf", big, true); code != 413 || !strings.Contains(body, "50 MB") {
		t.Fatalf("51 MB = %d %s", code, body)
	}
	if files() != before {
		t.Fatal("a refused upload wrote a file row")
	}
	quote := psample.PDF(psample.Proposal{Firm: "Acme Planning", Lines: []psample.Line{{Label: "Fee", Cents: 100}}})
	// Three uploads above counted toward the hour (the type is checked first,
	// so only huge.pdf did); the same bytes again are the same quote.
	for i := 1; i < 20; i++ {
		if code, body := acme.raw(t, http.MethodPost, "/api/guest/quote?name=quote.pdf", quote, true); code != http.StatusCreated {
			t.Fatalf("upload %d = %d %s", i, code, body)
		}
	}
	if code, _ := acme.raw(t, http.MethodPost, "/api/guest/quote?name=quote.pdf", quote, true); code != 429 {
		t.Fatalf("21st upload in the hour = %d", code)
	}
}

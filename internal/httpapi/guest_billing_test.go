package httpapi_test

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"sitewise/internal/files"
	"sitewise/internal/httpapi"
	psample "sitewise/internal/proposals/sample"
	"sitewise/internal/store"
)

// A Free org with billing on: one project, a package with an invited firm
// and an issued RFT. Fifty links make no seat and leave the team as it was;
// a firm's quote at the documents cap is refused in the guest's own words.
func TestGuestsAreNotSeatsAndCountTowardTheCap(t *testing.T) {
	ctx := context.Background()
	blobs, err := files.Open(t.TempDir(), 100<<20)
	if err != nil {
		t.Fatal(err)
	}
	f := &guestFixture{blobs: blobs, a: newApp(t, withProfile(t), greenProposals, func(o *httpapi.Options) {
		o.Blobs = blobs
		o.Billing = &httpapi.Billing{}
	})}
	f.m = f.a.member(t, newUUID(t))
	f.project = f.m.createProject(t, "Free guest project")
	if f.conn, err = pgx.Connect(ctx, os.Getenv("SITEWISE_TEST_DATABASE_URL")); err != nil {
		t.Fatal(err)
	}
	defer f.conn.Close(ctx)
	if err := f.conn.QueryRow(ctx, `SELECT id::text FROM users WHERE org_id=$1::uuid LIMIT 1`, f.m.orgID).Scan(&f.user); err != nil {
		t.Fatal(err)
	}
	var pkg struct {
		ID string `json:"id"`
	}
	f.m.call(t, http.MethodPost, "/api/projects/"+f.project+"/procurement/rows", []byte(`{"discipline_id":"consultant.town_planner"}`), http.StatusCreated, &pkg)
	f.pkg = pkg.ID
	var inv struct {
		FirmID string `json:"firm_id"`
	}
	f.m.call(t, http.MethodPost, "/api/projects/"+f.project+"/procurement/packages/"+f.pkg+"/invitations", []byte(`{"name":"Acme Planning"}`), http.StatusCreated, &inv)
	f.firmA = inv.FirmID
	var pmp store.Report
	f.m.call(t, http.MethodPost, "/api/projects/"+f.project+"/reports", []byte(`{"kind":"pmp"}`), http.StatusCreated, &pmp)
	f.pmp = pmp.ID
	var d store.ReportDraft
	f.m.call(t, http.MethodPost, "/api/reports/"+f.pmp+"/draft", []byte(`{"use_last_completed":true}`), http.StatusOK, &d)
	body, _ := json.Marshal(store.IssueOptions{Version: d.Version, ReportingDate: "2026-10-07", AcceptStale: true, StaleReason: "Guest billing fixture with missing profile acknowledged"})
	var issued store.IssuedReport
	f.m.call(t, http.MethodPost, "/api/reports/"+f.pmp+"/issue", body, http.StatusCreated, &issued)
	f.pmpIssue = issued.ID
	var rft store.Report
	f.m.call(t, http.MethodPost, "/api/projects/"+f.project+"/reports", []byte(`{"kind":"rfp","package_id":"`+f.pkg+`"}`), http.StatusCreated, &rft)
	f.rft = rft.ID
	f.m.call(t, http.MethodPost, "/api/reports/"+f.rft+"/draft", []byte(`{"use_last_completed":true}`), http.StatusOK, &d)
	f.issueRFT(t, d.ID, nil)

	team := func() string {
		var raw json.RawMessage
		f.m.getJSON(t, "/api/team", &raw)
		return string(raw)
	}
	usage := func() store.Usage {
		u, err := f.a.store.Usage(ctx, f.m.orgID, time.Now())
		if err != nil {
			t.Fatal(err)
		}
		return u
	}
	teamBefore, usageBefore := team(), usage()
	inviteBefore := f.m.status(t, http.MethodPost, "/api/team/invites", []byte(`{"email":"colleague@example.test"}`))
	var token string
	for i := 0; i < 50; i++ {
		_, token = f.share(t, map[string]any{"kind": "package", "package_id": f.pkg, "firm_id": f.firmA, "shared_with": fmt.Sprintf("acme%d@example.test", i)})
	}
	if team() != teamBefore || usage().Seats != usageBefore.Seats || usage().Projects != usageBefore.Projects {
		t.Fatalf("links changed the team: %s / %+v", team(), usage())
	}
	if got := f.m.status(t, http.MethodPost, "/api/team/invites", []byte(`{"email":"colleague@example.test"}`)); got != inviteBefore {
		t.Fatalf("team invite with 50 links = %d, before %d", got, inviteBefore)
	}
	acme := f.guest(t, token)
	// A guest stops at 90% of the month's documents (45 of 50 on Free), so it
	// never takes the last uploads from the org's own PMs (review M3).
	for usage().DocumentsThisMonth < 45 {
		if err := f.a.store.CountDocument(ctx, f.m.orgID, time.Now()); err != nil {
			t.Fatal(err)
		}
	}
	quote := psample.PDF(psample.Proposal{Firm: "Acme Planning", Lines: []psample.Line{{Label: "Fee", Cents: 100}}})
	code, msg := acme.raw(t, http.MethodPost, "/api/guest/quote?name=quote.pdf", quote, true)
	if code != http.StatusPaymentRequired || !strings.Contains(msg, "upload allowance is used up") || strings.Contains(msg, "Free") {
		t.Fatalf("quote at the cap = %d %s", code, msg)
	}
	if n := f.count(t, `SELECT count(*) FROM documents WHERE org_id=$1::uuid AND filename='quote.pdf'`, f.m.orgID); n != 0 {
		t.Fatal("a quote over the cap was stored")
	}
}

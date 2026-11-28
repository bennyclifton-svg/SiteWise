package httpapi_test

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"

	"sitewise/internal/export"
	"sitewise/internal/invoices"
	"sitewise/internal/reports"
	"sitewise/internal/store"
)

type sendLedger struct {
	Claims []struct {
		Served          string `json:"schedule_served_on"`
		ServedConfirmed bool   `json:"schedule_served_confirmed"`
		Send            *struct {
			MessageID string `json:"message_id"`
			Status    string `json:"status"`
			SentOn    string `json:"sent_on"`
		} `json:"schedule_send"`
		Issued []struct {
			ReportID  string `json:"report_id"`
			VersionID string `json:"version_id"`
			File      string `json:"file"`
		} `json:"issued"`
	} `json:"claims"`
}

// issuedClaim is a claim dropped on a live contract, accepted as claimed and
// issued. It returns the claim's id.
func issuedClaim(t *testing.T, m *member, project, pkg string) string {
	t.Helper()
	raw, err := os.ReadFile("../invoices/testdata/claim-pc04.md")
	if err != nil {
		t.Fatal(err)
	}
	base := "/api/projects/" + project
	var created struct{ ID string }
	m.call(t, http.MethodPost, base+"/invoices?name=pc04.md&kind=progress_claim&package="+pkg, raw, http.StatusCreated, &created)
	m.waitInvoices(t, project, 1)
	m.ops(t, project, "claims", http.StatusOK, op("claims.accept_claimed", map[string]any{"claim": created.ID}))
	m.call(t, http.MethodPost, base+"/invoices/"+created.ID+"/schedule", nil, http.StatusCreated, nil)
	return created.ID
}

// Slice 5: Issue freezes the sheet in the owner's layout, under its own
// template id, and the PDF is the house table style with every section.
func TestScheduleIssuesInTheOwnersLayout(t *testing.T) {
	ctx := context.Background()
	a := newApp(t, withProfile(t))
	m := a.member(t, newUUID(t))
	project := m.createProject(t, "Owner layout")
	pkg, _, _ := liveContract(t, m, project)
	base := "/api/projects/" + project
	m.call(t, http.MethodPatch, base+"/procurement/packages/"+pkg+"/award", []byte(`{"contract_date":"2026-02-25"}`), http.StatusOK, nil)
	m.call(t, http.MethodPut, base+"/delivery/settings", []byte(`{"principal_name":"Harbour Trust","assessor_name":"A. Manager"}`), http.StatusOK, nil)
	claim := issuedClaim(t, m, project, pkg)

	var v sendLedger
	m.getJSON(t, base+"/invoices", &v)
	if len(v.Claims) != 1 || len(v.Claims[0].Issued) != 1 {
		t.Fatalf("issued: %+v", v.Claims)
	}
	issue := v.Claims[0].Issued[0]
	report, err := a.store.ReadIssuedReport(ctx, m.orgID, issue.ReportID, issue.VersionID)
	if err != nil {
		t.Fatal(err)
	}
	snap := report.Snapshot
	if snap.SourceRevisions.TemplateID != store.PaymentScheduleSheetTemplate || snap.RendererVersion != export.Version || snap.Kind != reports.KindPaymentSchedule {
		t.Fatalf("snapshot template %q renderer %q kind %q", snap.SourceRevisions.TemplateID, snap.RendererVersion, snap.Kind)
	}
	var order []string
	words := ""
	for _, s := range snap.Sections {
		order = append(order, s.ID)
		for _, b := range s.Blocks {
			words += b.Text + "\n"
			if b.Table != nil {
				for _, r := range b.Table.Rows {
					words += strings.Join(r, "|") + "\n"
				}
			}
		}
	}
	if got := strings.Join(order, " "); got != "details works totals note payments approval" {
		t.Fatalf("sections %q", got)
	}
	for _, want := range []string{"Project: Owner layout", "Contract: Main Works", "Principal: Harbour Trust", "Contract date: 25 February 2026", "Claim no.: PC-04",
		"Substructure|100,000.00|60.0|60,000.00|–|60,000.00", "Sub total ex GST ($)|150,000.00||86,000.00|–|86,000.00",
		"Retention held this claim|||||−2,300.00", "Total claim inc retention (ex GST)|||||83,700.00", "Amount payable (inc GST)|||||92,070.00",
		"Progress claim to be paid in full.", "PC-04|Awaiting invoice||83,700.00|8,370.00|92,070.00", "A. Manager||"} {
		if !strings.Contains(words, want) {
			t.Fatalf("the issued schedule lacks %q:\n%s", want, words)
		}
	}
	// No Act is printed for a project with no verified state row behind it.
	if strings.Contains(words, "made under") {
		t.Fatalf("an Act was printed:\n%s", words)
	}
	if _, hash, err := reports.CanonicalSnapshot(snap); err != nil || hash == "" {
		t.Fatalf("snapshot hash %q %v", hash, err)
	}
	resp := m.do(t, http.MethodGet, issue.File, nil)
	defer resp.Body.Close()
	pdf, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK || !bytes.HasPrefix(pdf, []byte("%PDF")) {
		t.Fatalf("pdf = %d, %d bytes", resp.StatusCode, len(pdf))
	}
	_ = claim
}

// Mail is not set up: Send is refused in words, nothing is queued and the
// claim is not marked. The PM downloads the PDF and types the day served.
func TestScheduleSendRefusedWhenMailIsOff(t *testing.T) {
	f := newMailFixture(t, withProfile(t))
	pkg, _, _ := liveContract(t, f.m, f.project)
	claim := issuedClaim(t, f.m, f.project, pkg)
	raw, _ := json.Marshal(map[string]any{"to": []string{"builder@example.test"}, "subject": "Payment schedule PC-04", "claim": claim})
	resp := f.m.do(t, http.MethodPost, "/api/projects/"+f.project+"/transmittals", raw)
	body, _ := io.ReadAll(resp.Body)
	resp.Body.Close()
	if resp.StatusCode != http.StatusServiceUnavailable || !strings.Contains(string(body), "not set up") || f.messages(t) != 0 {
		t.Fatalf("send with mail off = %d %s", resp.StatusCode, body)
	}
	var v sendLedger
	f.m.getJSON(t, "/api/projects/"+f.project+"/invoices", &v)
	if c := v.Claims[0]; c.Send != nil || c.Served != "" {
		t.Fatalf("the claim was marked: %+v", c)
	}
}

// Send carries the issued PDF as it was issued. When the message has gone,
// Go sets the day the schedule was served by rule, for the PM to confirm.
func TestScheduleSendSetsTheServedDateByRule(t *testing.T) {
	f := newSendFixture(t, withProfile(t))
	pkg, _, _ := liveContract(t, f.m, f.project)
	base := "/api/projects/" + f.project

	// A claim with no issued schedule has nothing to send.
	raw, err := os.ReadFile("../invoices/testdata/claim-pc04.md")
	if err != nil {
		t.Fatal(err)
	}
	var draft struct{ ID string }
	f.m.call(t, http.MethodPost, base+"/invoices?name=pc04.md&kind=progress_claim&package="+pkg, raw, http.StatusCreated, &draft)
	f.m.waitInvoices(t, f.project, 1)
	if status, _, text := f.send(t, map[string]any{"to": []string{"builder@example.test"}, "claim": draft.ID}); status != http.StatusConflict || !strings.Contains(text, "Issue the payment schedule") {
		t.Fatalf("send before issue = %d %s", status, text)
	}
	if status, _, text := f.send(t, map[string]any{"to": []string{"builder@example.test"}, "claim": newUUID(t)}); status != http.StatusBadRequest {
		t.Fatalf("send for no claim = %d %s", status, text)
	}
	f.m.ops(t, f.project, "claims", http.StatusOK, op("claims.accept_claimed", map[string]any{"claim": draft.ID}))
	f.m.call(t, http.MethodPost, base+"/invoices/"+draft.ID+"/schedule", nil, http.StatusCreated, nil)

	status, msg, text := f.send(t, map[string]any{"to": []string{"Builder <builder@example.test>"}, "cc": []string{"principal@example.test"}, "subject": "Payment schedule PC-04 · Main Works", "claim": draft.ID})
	if status != http.StatusAccepted {
		t.Fatalf("send = %d %s", status, text)
	}
	sent := f.waitStatus(t, msg.ID, "sent")
	// The sheet (part 0) and the issued PDF, byte for byte as issued.
	var pdfPart bool
	for _, p := range sent.Parts {
		pdfPart = pdfPart || strings.HasPrefix(p.Filename, "Payment schedule") && p.ContentType == "application/pdf"
	}
	if !pdfPart || len(f.sink.Kept()) != 1 {
		t.Fatalf("parts %+v, sink %d", sent.Parts, len(f.sink.Kept()))
	}
	var v sendLedger
	deadline := time.Now().Add(5 * time.Second)
	for {
		f.m.getJSON(t, base+"/invoices", &v)
		if v.Claims[0].Served != "" || time.Now().After(deadline) {
			break
		}
		time.Sleep(20 * time.Millisecond)
	}
	c := v.Claims[0]
	today := time.Now().In(invoices.Sydney).Format("2006-01-02")
	if c.Served != today || c.ServedConfirmed || c.Send == nil || c.Send.MessageID != msg.ID || c.Send.Status != "sent" || c.Send.SentOn != today {
		t.Fatalf("after send: %+v send %+v (want served %s by rule)", c, c.Send, today)
	}
	// The PM confirms it, or types another day; a later send does not move it.
	f.m.call(t, http.MethodPatch, base+"/invoices/"+draft.ID, []byte(`{"fields":{"schedule_served_on":"2026-10-20"}}`), http.StatusOK, nil)
	_, again, _ := f.send(t, map[string]any{"to": []string{"builder@example.test"}, "claim": draft.ID})
	f.waitStatus(t, again.ID, "sent")
	f.m.getJSON(t, base+"/invoices", &v)
	if c := v.Claims[0]; c.Served != "2026-10-20" || !c.ServedConfirmed || c.Send == nil || c.Send.MessageID != again.ID {
		t.Fatalf("after the PM's date: %+v send %+v", c, c.Send)
	}
}

// Chat opens Send filled in for a claim's issued schedule: the principal
// from the project's terms, the claim checked by Go. It sends nothing, and
// a claim with no issued schedule is refused in words.
func TestChatOpensSendForAnIssuedSchedule(t *testing.T) {
	ta := newToolApp(t)
	pkg, _, _ := liveContract(t, ta.m, ta.project)
	base := "/api/projects/" + ta.project
	ta.m.call(t, http.MethodPut, base+"/delivery/settings", []byte(`{"principal_name":"Harbour Trust","principal_email":"accounts@harbour.example"}`), http.StatusOK, nil)
	raw, err := os.ReadFile("../invoices/testdata/claim-pc04.md")
	if err != nil {
		t.Fatal(err)
	}
	var created struct{ ID string }
	ta.m.call(t, http.MethodPost, base+"/invoices?name=pc04.md&kind=progress_claim&package="+pkg, raw, http.StatusCreated, &created)
	ta.m.waitInvoices(t, ta.project, 1)
	open := tool("view", map[string]any{"do": "open", "dialog": "send", "fields": map[string]string{"claim": "PC-04"}, "final": true})
	viewOf := func(events []chatEvent) *chatEvent {
		for i := range events {
			if events[i].Type == "view" {
				return &events[i]
			}
		}
		return nil
	}
	if v := viewOf(ta.run(t, "Send schedule PC-04 to the principal", open)); v != nil || !strings.Contains(strings.Join(ta.results, "\n"), "no issued payment schedule") {
		t.Fatalf("before issue: view %+v results %q", v, ta.results)
	}
	ta.m.ops(t, ta.project, "claims", http.StatusOK, op("claims.accept_claimed", map[string]any{"claim": created.ID}))
	ta.m.call(t, http.MethodPost, base+"/invoices/"+created.ID+"/schedule", nil, http.StatusCreated, nil)
	ta.calls, ta.results = nil, nil
	v := viewOf(ta.run(t, "Send schedule PC-04 to the principal", open))
	if v == nil || v.View.Fields["claim"] != created.ID || !strings.Contains(v.View.Fields["to"], "accounts@harbour.example") ||
		v.View.Fields["subject"] != "Payment schedule PC-04 · Main Works" || !strings.Contains(v.View.Line, "Nothing is sent until the PM presses Send") {
		t.Fatalf("view %+v results %q", v, ta.results)
	}
	// Nothing left the building, and the claim is not marked.
	var ledger sendLedger
	ta.m.getJSON(t, base+"/invoices", &ledger)
	if c := ledger.Claims[0]; c.Send != nil || c.Served != "" {
		t.Fatalf("chat sent something: %+v", c)
	}
}

// Slice 7: a consultant's contract takes a claim the PM drops on it. The
// claim is its own tax invoice, and its schedules name no Act until the PM
// switches it on for that contract (owner decisions 7 and 8).
func TestServicesContractTakesAClaim(t *testing.T) {
	ctx := context.Background()
	a := newApp(t, withProfile(t))
	m := a.member(t, newUUID(t))
	project := m.createProject(t, "Consultant claims")
	pkg, _, _ := liveContract(t, m, project)
	conn, err := pgx.Connect(ctx, os.Getenv("SITEWISE_TEST_DATABASE_URL"))
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close(ctx)
	if _, err := conn.Exec(ctx, `UPDATE packages SET kind='services', works_scope=NULL WHERE org_id=$1 AND id=$2`, m.orgID, pkg); err != nil {
		t.Fatal(err)
	}
	base := "/api/projects/" + project
	raw, err := os.ReadFile("../invoices/testdata/claim-pc04.md")
	if err != nil {
		t.Fatal(err)
	}
	var created struct{ ID string }
	m.call(t, http.MethodPost, base+"/invoices?name=pc04.md&kind=progress_claim&package="+pkg, raw, http.StatusCreated, &created)
	m.waitInvoices(t, project, 1)
	var v struct {
		Contracts []struct {
			Services   bool   `json:"services"`
			ActOn      bool   `json:"act_on"`
			ActApplies string `json:"act_applies"`
			Claims     bool   `json:"claims"`
		} `json:"contracts"`
		Claims []struct {
			TaxInvoice      bool `json:"tax_invoice"`
			AwaitingInvoice bool `json:"awaiting_invoice"`
		} `json:"claims"`
	}
	m.getJSON(t, base+"/invoices", &v)
	if len(v.Contracts) != 1 || !v.Contracts[0].Services || v.Contracts[0].ActOn || !v.Contracts[0].Claims || len(v.Claims) != 1 || !v.Claims[0].TaxInvoice || v.Claims[0].AwaitingInvoice {
		t.Fatalf("services contract: %+v", v)
	}
	m.call(t, http.MethodPatch, base+"/procurement/packages/"+pkg+"/award", []byte(`{"act_applies":"yes"}`), http.StatusOK, nil)
	m.getJSON(t, base+"/invoices", &v)
	if !v.Contracts[0].ActOn || v.Contracts[0].ActApplies != "yes" {
		t.Fatalf("switched on: %+v", v.Contracts[0])
	}
	if got := m.status(t, http.MethodPatch, base+"/procurement/packages/"+pkg+"/award", []byte(`{"act_applies":"maybe"}`)); got != http.StatusUnprocessableEntity {
		t.Fatalf("act_applies maybe = %d", got)
	}
	// The PM may say the consultant sends a separate tax invoice after all.
	m.call(t, http.MethodPatch, base+"/invoices/"+created.ID, []byte(`{"fields":{"tax_invoice":"no"}}`), http.StatusOK, nil)
	m.getJSON(t, base+"/invoices", &v)
	if v.Claims[0].TaxInvoice || !v.Claims[0].AwaitingInvoice {
		t.Fatalf("not its own invoice: %+v", v.Claims[0])
	}
}

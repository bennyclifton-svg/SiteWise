package httpapi_test

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"sitewise/internal/httpapi"
	"sitewise/internal/store"
)

// The record's read and its two edits (correspondence plan, Task 4), and
// who holds which revision (Task 5).

func (f *mailFixture) pkg(t *testing.T, discipline string) string {
	t.Helper()
	var pkg struct {
		ID string `json:"id"`
	}
	f.m.call(t, http.MethodPost, "/api/projects/"+f.project+"/procurement/rows", []byte(`{"discipline_id":"`+discipline+`"}`), http.StatusCreated, &pkg)
	return pkg.ID
}

func (f *mailFixture) message(t *testing.T, messageID string) store.MailMessage {
	t.Helper()
	var m store.MailMessage
	f.m.getJSON(t, "/api/projects/"+f.project+"/mail/"+f.heldID(t, messageID), &m)
	return m
}

func (f *mailFixture) patch(t *testing.T, id string, body any) int {
	t.Helper()
	raw, _ := json.Marshal(body)
	return f.m.status(t, http.MethodPatch, "/api/projects/"+f.project+"/mail/"+id, raw)
}

func TestMailRecordEdits(t *testing.T) {
	f := newMailFixture(t)
	roof := f.pkg(t, "consultant.structural")
	foreign := f.a.member(t, newUUID(t))
	p := f.memberPost("Our letter")
	f.send(t, p)
	m := f.message(t, p.messageID)
	if m.Direction != "out" || m.PackageID != "" {
		t.Fatalf("%+v", m)
	}
	// Another org: 404. A stale version: 409. Another field: refused.
	raw, _ := json.Marshal(map[string]any{"version": m.Version, "package_id": roof})
	if got := foreign.status(t, http.MethodPatch, "/api/projects/"+f.project+"/mail/"+m.ID, raw); got != http.StatusNotFound {
		t.Fatalf("foreign = %d", got)
	}
	if got := f.patch(t, m.ID, map[string]any{"version": m.Version + 7, "package_id": roof}); got != http.StatusConflict {
		t.Fatalf("stale = %d", got)
	}
	for _, other := range []map[string]any{{"version": m.Version, "subject": "Changed"}, {"version": m.Version, "body": "x"}, {"version": m.Version, "sender": "x@y"}, {"version": m.Version}} {
		if got := f.patch(t, m.ID, other); got != http.StatusBadRequest {
			t.Fatalf("patch %v = %d", other, got)
		}
	}
	if got := f.patch(t, m.ID, map[string]any{"version": m.Version, "package_id": newUUID(t)}); got != http.StatusNotFound {
		t.Fatalf("a package of no project = %d", got)
	}
	if got := f.patch(t, m.ID, map[string]any{"version": m.Version, "package_id": roof}); got != http.StatusOK {
		t.Fatalf("link = %d", got)
	}
	m = f.message(t, p.messageID)
	if m.PackageID != roof || m.LinkedBy != "user" {
		t.Fatalf("after link %+v", m)
	}
	// The user's package survives a later reply in the thread.
	reply := f.memberPost("Re: Our letter")
	reply.headers = append(reply.headers, [2]string{"In-Reply-To", p.messageID})
	f.send(t, reply)
	if again := f.message(t, p.messageID); again.PackageID != roof {
		t.Fatalf("reply moved the package: %+v", again)
	}
	// A held message is not editable.
	held := f.memberPost("From outside")
	held.from, held.envelope = "x@outside.test", "x@outside.test"
	f.send(t, held)
	h := f.message(t, held.messageID)
	if got := f.patch(t, h.ID, map[string]any{"version": h.Version, "pick_state": "dismissed"}); got != http.StatusConflict {
		t.Fatalf("held = %d", got)
	}
	// No route deletes a filed message.
	for _, pattern := range httpapi.RoutePatterns() {
		if strings.HasPrefix(pattern, "DELETE ") && strings.Contains(pattern, "/mail") {
			t.Fatalf("a DELETE route on mail: %s", pattern)
		}
	}
}

// The operations: mail.link applies at once and Undo puts it back.
func TestMailLinkOperationUndoes(t *testing.T) {
	f := newMailFixture(t)
	roof := f.pkg(t, "consultant.structural")
	p := f.memberPost("Our letter")
	f.send(t, p)
	m := f.message(t, p.messageID)
	body, _ := json.Marshal(map[string]any{"area": "mail", "ops": []map[string]any{{"op": "mail.link", "fields": map[string]any{"message": m.ID[:8], "package": roof}}}})
	var applied struct {
		ChangeID string `json:"change_id"`
		Undo     struct {
			Kind string `json:"kind"`
		} `json:"undo"`
	}
	f.m.call(t, http.MethodPost, "/api/projects/"+f.project+"/ops", body, http.StatusOK, &applied)
	if applied.Undo.Kind != "change" || f.message(t, p.messageID).PackageID != roof {
		t.Fatalf("applied %+v", applied)
	}
	f.m.call(t, http.MethodPost, "/api/projects/"+f.project+"/changes/"+applied.ChangeID+"/undo", nil, http.StatusOK, nil)
	if after := f.message(t, p.messageID); after.PackageID != "" {
		t.Fatalf("undo left %+v", after)
	}
	// A filed message is never accepted or discarded from chat either.
	discard, _ := json.Marshal(map[string]any{"area": "mail", "ops": []map[string]any{{"op": "mail.discard", "fields": map[string]any{"message": m.ID}}}})
	resp := f.m.do(t, http.MethodPost, "/api/projects/"+f.project+"/ops", discard)
	resp.Body.Close()
	if resp.StatusCode == http.StatusOK {
		t.Fatal("mail.discard applied without a click")
	}
}

func TestWhoHoldsWhichRevision(t *testing.T) {
	ctx := context.Background()
	f := newSendFixture(t)
	a := f.doc(t, f.m, f.project, "S-201 A.pdf", "S-201", "A", []byte("%PDF rev A"))
	b := f.doc(t, f.m, f.project, "S-201 B.pdf", "S-201", "B", []byte("%PDF rev B"))
	c := f.doc(t, f.m, f.project, "S-201 C.pdf", "S-201", "C", []byte("%PDF rev C"))
	for _, link := range [][2]string{{b, a}, {c, b}} {
		if _, err := f.conn.Exec(ctx, `INSERT INTO supersessions(org_id,document_id,prior_document_id) VALUES ($1::uuid,$2::uuid,$3::uuid)`, f.m.orgID, link[0], link[1]); err != nil {
			t.Fatal(err)
		}
	}
	firm := f.pkg(t, "consultant.structural")
	var inv struct {
		FirmID string `json:"firm_id"`
	}
	f.m.call(t, http.MethodPost, "/api/projects/"+f.project+"/procurement/packages/"+firm+"/invitations", []byte(`{"name":"Acme Engineers"}`), http.StatusCreated, &inv)
	if _, err := f.conn.Exec(ctx, `UPDATE firms SET email='acme@acme.example' WHERE id=$1::uuid`, inv.FirmID); err != nil {
		t.Fatal(err)
	}
	sendTo := func(to string, doc string) {
		t.Helper()
		status, m, text := f.send(t, map[string]any{"to": []string{to}, "document_ids": []string{doc}})
		if status != http.StatusAccepted {
			t.Fatalf("send = %d %s", status, text)
		}
		f.waitStatus(t, m.ID, "sent")
	}
	sendTo("acme@acme.example", a)  // Acme holds A
	sendTo("bolt@bolt.example", b)  // Bolt holds B
	sendTo("later@cell.example", b) // sent B then C: not listed
	sendTo("later@cell.example", c)
	read := func(doc string) store.DocumentTransmittals {
		var out store.DocumentTransmittals
		f.m.getJSON(t, "/api/projects/"+f.project+"/mail?document="+doc, &out)
		return out
	}
	got := read(c)
	if !got.Newest || len(got.Sends) != 4 {
		t.Fatalf("sends %+v", got)
	}
	holders := map[string]string{}
	for _, h := range got.HoldingOlder {
		holders[h.Address] = h.Revision + "|" + h.Firm
	}
	if holders["acme@acme.example"] != "A|Acme Engineers" || holders["bolt@bolt.example"] != "B|" || len(holders) != 2 {
		t.Fatalf("holders %v", holders)
	}
	if old := read(a); old.Newest || len(old.HoldingOlder) != 0 || len(old.Sends) != 1 {
		t.Fatalf("an older revision %+v", old)
	}
	// A deleted older revision still counts: the part keeps what went.
	if _, err := f.a.store.DeleteDocuments(ctx, f.m.orgID, f.project, []string{a}, f.user); err != nil {
		t.Fatal(err)
	}
	got = read(c)
	if len(got.Sends) != 4 || len(got.HoldingOlder) != 2 {
		t.Fatalf("after deleting A: %+v", got)
	}
	// Another project's document: 404.
	sibling := f.m.createProject(t, "Sibling")
	other := f.doc(t, f.m, sibling, "Other.pdf", "X-1", "A", []byte("%PDF other"))
	if got := f.m.status(t, http.MethodGet, "/api/projects/"+f.project+"/mail?document="+other, nil); got != http.StatusNotFound {
		t.Fatalf("other project = %d", got)
	}
	if got := f.m.status(t, http.MethodGet, fmt.Sprintf("/api/projects/%s/mail?document=x", f.project), nil); got != http.StatusNotFound {
		t.Fatalf("malformed = %d", got)
	}
}

// A notice on the register may now name its message (the register's wait on
// plan 1): only a filed message of this project, and its pick is used.
func TestRegisterNoticeFromAMessage(t *testing.T) {
	f := newMailFixture(t)
	foreign := newMailFixture(t)
	p := f.memberPost("Notice of delay")
	f.send(t, p)
	m := f.message(t, p.messageID)
	fp := foreign.memberPost("Theirs")
	foreign.send(t, fp)
	theirs := foreign.message(t, fp.messageID)
	notice := func(message string) int {
		body, _ := json.Marshal(map[string]any{"kind": "notice", "title": "Delay notice", "status": "open",
			"details": map[string]any{"message_id": message, "notice": "Extension of time or delay notice", "received_on": nil}})
		return f.m.status(t, http.MethodPost, "/api/projects/"+f.project+"/delivery", body)
	}
	if got := notice(theirs.ID); got == http.StatusCreated {
		t.Fatal("a notice named another org's message")
	}
	if got := notice(m.ID); got != http.StatusCreated {
		t.Fatalf("notice from a message = %d", got)
	}
	if after := f.message(t, p.messageID); after.PickState != "used" {
		t.Fatalf("pick state %q", after.PickState)
	}
}

// Plan 3's two chip actions, through their own routes: a filed attachment
// read as the contract's progress claim, and a variation that links the
// message's body document.
func TestMailChipActionsReachPlan3(t *testing.T) {
	ctx := context.Background()
	f := newMailFixture(t)
	pkg, lines, _ := liveContract(t, f.m, f.project)
	claim := mailFile{field: "attachment-1", name: "pc04.md", ctype: "text/markdown", body: []byte("# Progress claim 4\n\nSubstructure $10,000\n")}
	p := f.memberPost("Progress claim 4", claim)
	f.send(t, p)
	var body, attached string
	if err := f.conn.QueryRow(ctx, `SELECT max(CASE WHEN p.n=0 THEN p.document_id::text END), max(CASE WHEN p.n=1 THEN p.document_id::text END)
FROM mail_parts p JOIN mail_messages m ON m.org_id=p.org_id AND m.id=p.message_id WHERE m.message_id=$1`, p.messageID).Scan(&body, &attached); err != nil {
		t.Fatal(err)
	}
	for _, doc := range []string{body, attached} {
		deadline := time.Now().Add(10 * time.Second)
		for {
			var status string
			_ = f.conn.QueryRow(ctx, `SELECT status FROM documents WHERE id=$1::uuid`, doc).Scan(&status)
			if status != "pending" || time.Now().After(deadline) {
				break
			}
			time.Sleep(50 * time.Millisecond)
		}
	}
	// Read as progress claim: the attachment's own bytes, on the contract.
	resp := f.m.do(t, http.MethodGet, "/api/documents/"+attached+"/file", nil)
	raw, _ := io.ReadAll(resp.Body)
	resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("file = %d", resp.StatusCode)
	}
	f.m.call(t, http.MethodPost, "/api/projects/"+f.project+"/invoices?name=pc04.md&kind=progress_claim&package="+pkg, raw, http.StatusCreated, nil)
	if n := f.count(t, `SELECT count(*) FROM invoice_decisions WHERE org_id=$1::uuid AND field='kind' AND value='progress_claim'`, f.m.orgID); n != 1 {
		t.Fatalf("claims %d", n)
	}
	// Add variation: the message's body document linked.
	vb, _ := json.Marshal(map[string]any{"package": pkg, "line": lines[0], "title": "Progress claim 4", "link_documents": []string{body}})
	if got := f.m.status(t, http.MethodPost, "/api/projects/"+f.project+"/variations", vb); got >= 300 {
		t.Fatalf("variation = %d", got)
	}
	if n := f.count(t, `SELECT count(*) FROM variation_documents WHERE org_id=$1::uuid AND document_id=$2::uuid`, f.m.orgID, body); n != 1 {
		t.Fatalf("linked %d", n)
	}
}

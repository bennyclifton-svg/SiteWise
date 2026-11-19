package httpapi_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime"
	"mime/multipart"
	"net/http"
	"net/mail"
	"strings"
	"sync"
	"testing"
	"time"

	"sitewise/internal/correspondence"
	"sitewise/internal/httpapi"
	"sitewise/internal/store"
)

// The one send path (correspondence plan, Task 3), against a sink: nothing
// here reaches a mail service.

const mailTestFrom = "sitewise@example.test"

type sendFixture struct {
	*mailFixture
	sink *correspondence.Sink
}

func newSendFixture(t *testing.T, adjust ...func(*httpapi.Options)) *sendFixture {
	t.Helper()
	sink := &correspondence.Sink{}
	opts := append([]func(*httpapi.Options){func(o *httpapi.Options) {
		o.MailSender, o.MailFrom, o.MailRetryDelay = sink, mailTestFrom, 5*time.Millisecond
	}}, adjust...)
	return &sendFixture{mailFixture: newMailFixture(t, opts...), sink: sink}
}

// doc stores a filed document with real bytes and no filing call.
func (f *sendFixture) doc(t *testing.T, m *member, project, name, number, revision string, body []byte) string {
	t.Helper()
	ctx := context.Background()
	blob, err := f.blobs.Put(ctx, bytes.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	id, fileID := newUUID(t), newUUID(t)
	if err := f.a.store.CreateFile(ctx, m.orgID, store.File{ID: fileID, ProjectID: project, SHA256: blob.SHA256, ByteSize: blob.Size, MediaType: "application/pdf"}); err != nil {
		t.Fatal(err)
	}
	if err := f.a.store.CreateDocument(ctx, m.orgID, store.Document{ID: id, ProjectID: project, FileID: fileID, Filename: name, Status: "filed", Number: number, Revision: revision}); err != nil {
		t.Fatal(err)
	}
	return id
}

func (f *sendFixture) send(t *testing.T, body map[string]any) (int, store.MailMessage, string) {
	t.Helper()
	raw, _ := json.Marshal(body)
	resp := f.m.do(t, http.MethodPost, "/api/projects/"+f.project+"/transmittals", raw)
	defer resp.Body.Close()
	text, _ := io.ReadAll(resp.Body)
	var m store.MailMessage
	_ = json.Unmarshal(text, &m)
	return resp.StatusCode, m, string(text)
}

func (f *sendFixture) waitStatus(t *testing.T, id, want string) store.MailMessage {
	t.Helper()
	deadline := time.Now().Add(10 * time.Second)
	for {
		var m store.MailMessage
		f.m.getJSON(t, "/api/projects/"+f.project+"/mail/"+id, &m)
		if m.SendStatus == want {
			return m
		}
		if time.Now().After(deadline) {
			t.Fatalf("send status %q, want %q", m.SendStatus, want)
		}
		time.Sleep(20 * time.Millisecond)
	}
}

func TestTransmittalSendsOneMIMEMessage(t *testing.T) {
	f := newSendFixture(t)
	a := f.doc(t, f.m, f.project, "A-101 Basement.pdf", "A-101", "B", []byte("%PDF basement plan"))
	b := f.doc(t, f.m, f.project, "A-102 Section.pdf", "A-102", "C", []byte("%PDF section"))
	status, m, text := f.send(t, map[string]any{"to": []string{"Eng Smith <eng@consult.example>"}, "subject": "TR-001 Mail project", "note": "Please review.\nRegards", "document_ids": []string{a, b}})
	if status != http.StatusAccepted || m.Number == nil || *m.Number != 1 || m.Direction != "out" {
		t.Fatalf("send = %d %s", status, text)
	}
	f.waitStatus(t, m.ID, "sent")
	kept := f.sink.Kept()
	if len(kept) != 1 {
		t.Fatalf("sink holds %d messages", len(kept))
	}
	got := kept[0]
	if got.From != mailTestFrom || len(got.To) != 2 {
		t.Fatalf("envelope from %q to %v", got.From, got.To)
	}
	msg, err := mail.ReadMessage(bytes.NewReader(got.Msg))
	if err != nil {
		t.Fatal(err)
	}
	h := msg.Header
	if h.Get("From") != `"SiteWise" <`+mailTestFrom+`>` && h.Get("From") != "SiteWise <"+mailTestFrom+">" || !strings.Contains(h.Get("Reply-To"), f.email) || !strings.Contains(h.Get("Reply-To"), f.address) ||
		!strings.Contains(h.Get("Cc"), f.email) || !strings.HasPrefix(h.Get("Message-ID"), "<") || h.Get("X-Auto-Response-Suppress") == "" {
		t.Fatalf("headers %v", h)
	}
	_, params, _ := mime.ParseMediaType(h.Get("Content-Type"))
	r := multipart.NewReader(msg.Body, params["boundary"])
	var names []string
	for {
		p, err := r.NextPart()
		if err == io.EOF {
			break
		}
		if err != nil {
			t.Fatal(err)
		}
		if n := p.FileName(); n != "" {
			names = append(names, n)
		}
	}
	if len(names) != 3 || names[0] != "Transmittal TR-001.pdf" || names[1] != "A-101 Basement.pdf" || names[2] != "A-102 Section.pdf" {
		t.Fatalf("attachments %v", names)
	}
	// The sheet downloads from the record; nothing about it can change.
	resp := f.m.do(t, http.MethodGet, "/api/projects/"+f.project+"/mail/"+m.ID+"/sheet", nil)
	sheet, _ := io.ReadAll(resp.Body)
	resp.Body.Close()
	if resp.StatusCode != http.StatusOK || !bytes.HasPrefix(sheet, []byte("%PDF")) {
		t.Fatalf("sheet = %d", resp.StatusCode)
	}
	// Frozen: no route edits or deletes a sent message.
	if got := f.m.status(t, http.MethodPost, "/api/projects/"+f.project+"/mail/"+m.ID+"/discard", nil); got != http.StatusConflict {
		t.Fatalf("discard sent = %d", got)
	}
	// A reply from the engineer joins its thread (A1 files it) and the
	// transmittal says Replied.
	reply := f.memberPost("Re: TR-001")
	reply.from, reply.envelope = "eng@consult.example", "eng@consult.example"
	reply.headers = append(reply.headers, [2]string{"In-Reply-To", sentMessageID(t, f.mailFixture, m.ID)})
	f.mailFixture.send(t, reply)
	l := f.list(t)
	var sent, answer store.MailMessage
	for _, x := range l.Messages {
		if x.ID == m.ID {
			sent = x
		}
		if x.Subject == "Re: TR-001" {
			answer = x
		}
	}
	if answer.Status != "filed" || answer.ThreadID != sent.ThreadID || !sent.Replied {
		t.Fatalf("reply %+v sent %+v", answer, sent)
	}
}

func sentMessageID(t *testing.T, f *mailFixture, id string) string {
	t.Helper()
	var mid string
	if err := f.conn.QueryRow(context.Background(), `SELECT message_id FROM mail_messages WHERE id=$1::uuid`, id).Scan(&mid); err != nil {
		t.Fatal(err)
	}
	return mid
}

func TestTransmittalRefusals(t *testing.T) {
	ctx := context.Background()
	f := newSendFixture(t)
	foreign := f.a.member(t, newUUID(t))
	foreignProject := foreign.createProject(t, "Foreign")
	sibling := f.m.createProject(t, "Sibling")
	mine := f.doc(t, f.m, f.project, "Mine.pdf", "M-1", "A", []byte("%PDF mine"))
	other := f.doc(t, foreign, foreignProject, "Theirs.pdf", "T-1", "A", []byte("%PDF theirs"))
	sib := f.doc(t, f.m, sibling, "Sibling.pdf", "S-1", "A", []byte("%PDF sibling"))
	to := []string{"eng@consult.example"}
	for name, c := range map[string]struct {
		body map[string]any
		want int
	}{
		"foreign document":       {map[string]any{"to": to, "document_ids": []string{other}}, http.StatusBadRequest},
		"other project document": {map[string]any{"to": to, "document_ids": []string{sib}}, http.StatusBadRequest},
		"duplicate":              {map[string]any{"to": to, "document_ids": []string{mine, mine}}, http.StatusBadRequest},
		"missing":                {map[string]any{"to": to, "document_ids": []string{newUUID(t)}}, http.StatusBadRequest},
		"malformed":              {map[string]any{"to": to, "document_ids": []string{"x"}}, http.StatusBadRequest},
		"no recipient":           {map[string]any{"to": []string{}}, http.StatusBadRequest},
		"bad address":            {map[string]any{"to": []string{"not an address"}}, http.StatusBadRequest},
		"CR in address":          {map[string]any{"to": []string{"eng@consult.example\r\nBcc: victim@example.net"}}, http.StatusBadRequest},
		"LF in subject":          {map[string]any{"to": to, "subject": "Hi\nBcc: victim@example.net"}, http.StatusBadRequest},
		"21 addresses":           {map[string]any{"to": manyAddresses(21)}, http.StatusBadRequest},
		"foreign reply":          {map[string]any{"to": to, "in_reply_to": newUUID(t)}, http.StatusBadRequest},
		"unknown issue":          {map[string]any{"to": to, "issue": newUUID(t)}, http.StatusBadRequest},
		"unknown RFI":            {map[string]any{"to": to, "rfi": newUUID(t)}, http.StatusBadRequest},
	} {
		if status, _, text := f.send(t, c.body); status != c.want {
			t.Fatalf("%s = %d %s", name, status, text)
		}
	}
	if n := f.count(t, `SELECT count(*) FROM mail_messages WHERE org_id=$1::uuid`, f.m.orgID); n != 0 {
		t.Fatalf("a refused send recorded %d messages", n)
	}
	// Another org cannot send on this project.
	raw, _ := json.Marshal(map[string]any{"to": to})
	if got := foreign.status(t, http.MethodPost, "/api/projects/"+f.project+"/transmittals", raw); got != http.StatusNotFound {
		t.Fatalf("foreign send = %d", got)
	}
	// Over 10 MB: 413 with the size.
	big := f.doc(t, f.m, f.project, "Big.pdf", "B-1", "A", bytes.Repeat([]byte("x"), 11<<20))
	if status, _, text := f.send(t, map[string]any{"to": to, "document_ids": []string{big}}); status != http.StatusRequestEntityTooLarge || !strings.Contains(text, "MB") {
		t.Fatalf("over 10 MB = %d %s", status, text)
	}
	// Unconfirmed email: 403.
	if _, err := f.conn.Exec(ctx, `UPDATE users SET email_verified=false WHERE org_id=$1::uuid`, f.m.orgID); err != nil {
		t.Fatal(err)
	}
	if status, _, _ := f.send(t, map[string]any{"to": to}); status != http.StatusForbidden {
		t.Fatalf("unconfirmed = %d", status)
	}
	if _, err := f.conn.Exec(ctx, `UPDATE users SET email_verified=true WHERE org_id=$1::uuid`, f.m.orgID); err != nil {
		t.Fatal(err)
	}
	// The daily cap: 429.
	if _, err := f.conn.Exec(ctx, `INSERT INTO mail_messages(org_id,id,project_id,message_id,sender,to_addrs,status,direction,thread_id,number,send_status)
SELECT $1::uuid, g.id, $2::uuid, '<cap-'||g.n||'@x>', 'x@example.test', 'a@b.example', 'filed', 'out', g.id, g.n, 'sent'
FROM (SELECT gen_random_uuid() AS id, n FROM generate_series(1,100) n) g`, f.m.orgID, sibling); err != nil {
		t.Fatal(err)
	}
	if status, _, _ := f.send(t, map[string]any{"to": to}); status != http.StatusTooManyRequests {
		t.Fatalf("cap = %d", status)
	}
	// An archived project: 409.
	if _, err := f.conn.Exec(ctx, `DELETE FROM mail_messages WHERE org_id=$1::uuid AND project_id=$2::uuid`, f.m.orgID, sibling); err == nil {
		t.Fatal("a sent message was deleted")
	}
	f.m.call(t, http.MethodPut, "/api/projects/"+f.project+"/archived", []byte(`{"archived":true}`), http.StatusNoContent, nil)
	if status, _, _ := f.send(t, map[string]any{"to": to}); status != http.StatusConflict && status != http.StatusTooManyRequests {
		t.Fatalf("archived = %d", status)
	}
}

func manyAddresses(n int) []string {
	out := make([]string, n)
	for i := range out {
		out[i] = fmt.Sprintf("a%d@example.net", i)
	}
	return out
}

func TestTransmittalNumbersUnderConcurrency(t *testing.T) {
	f := newSendFixture(t)
	var wg sync.WaitGroup
	nums := make([]int, 2)
	for i := range nums {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			_, m, _ := f.send(t, map[string]any{"to": []string{"eng@consult.example"}, "subject": fmt.Sprintf("Concurrent %d", i)})
			if m.Number != nil {
				nums[i] = *m.Number
			}
		}(i)
	}
	wg.Wait()
	if !(nums[0] == 1 && nums[1] == 2) && !(nums[0] == 2 && nums[1] == 1) {
		t.Fatalf("numbers %v", nums)
	}
}

func TestTransmittalRetriesThenFails(t *testing.T) {
	f := newSendFixture(t)
	f.sink.SetFail(errors.New("421 service not available"))
	_, m, text := f.send(t, map[string]any{"to": []string{"eng@consult.example"}, "subject": "Will fail"})
	if m.ID == "" {
		t.Fatal(text)
	}
	failed := f.waitStatus(t, m.ID, "failed")
	if !strings.Contains(failed.SendError, "421") {
		t.Fatalf("error %q", failed.SendError)
	}
	var attempts int
	_ = f.conn.QueryRow(context.Background(), `SELECT send_attempts FROM mail_messages WHERE id=$1::uuid`, m.ID).Scan(&attempts)
	if attempts != store.MailSendAttempts {
		t.Fatalf("attempts %d", attempts)
	}
	f.sink.SetFail(nil)
	if status, _, _ := f.send(t, map[string]any{"retry": m.ID}); status != http.StatusAccepted {
		t.Fatalf("retry = %d", status)
	}
	f.waitStatus(t, m.ID, "sent")
	if status, _, _ := f.send(t, map[string]any{"retry": m.ID}); status != http.StatusConflict {
		t.Fatalf("retry a sent message = %d", status)
	}
	if len(f.sink.Kept()) != 1 {
		t.Fatalf("sent %d times", len(f.sink.Kept()))
	}
}

// A message queued when the process stopped is sent once by the sweep.
func TestOutboxSweepSendsAQueuedMessageOnce(t *testing.T) {
	ctx := context.Background()
	f := newSendFixture(t)
	m, err := f.a.store.QueueOutbound(ctx, f.m.orgID, store.Outbound{ProjectID: f.project, SentBy: f.user, From: "Org via SiteWise <" + mailTestFrom + ">",
		To: []string{"eng@consult.example"}, Subject: "Queued before a restart"}, time.Now())
	if err != nil {
		t.Fatal(err)
	}
	if _, err := f.a.srv.Resume(ctx); err != nil {
		t.Fatal(err)
	}
	if _, err := f.a.srv.Resume(ctx); err != nil {
		t.Fatal(err)
	}
	f.waitStatus(t, m.ID, "sent")
	time.Sleep(100 * time.Millisecond)
	if n := len(f.sink.Kept()); n != 1 {
		t.Fatalf("sent %d times", n)
	}
}

func TestTransmittalOffWithoutASender(t *testing.T) {
	f := newMailFixture(t)
	raw, _ := json.Marshal(map[string]any{"to": []string{"eng@consult.example"}})
	resp := f.m.do(t, http.MethodPost, "/api/projects/"+f.project+"/transmittals", raw)
	resp.Body.Close()
	if resp.StatusCode != http.StatusServiceUnavailable || f.messages(t) != 0 {
		t.Fatalf("no sender = %d", resp.StatusCode)
	}
}

// Send for an RFI (the register's wait on plan 1): the RFI keeps the message
// it went out on, and a reply in that thread is a candidate the rule step
// matches without the RFI's number in its text.
func TestRFISendThreadsItsReply(t *testing.T) {
	ctx := context.Background()
	f := newSendFixture(t)
	body, _ := json.Marshal(map[string]any{"kind": "rfi", "title": "Footing depth", "status": "open", "details": map[string]any{"question": "Confirm the footing depth.", "raised_on": "2026-11-10"}})
	var rfi struct {
		ID string `json:"id"`
	}
	f.m.call(t, http.MethodPost, "/api/projects/"+f.project+"/delivery", body, http.StatusCreated, &rfi)
	status, m, text := f.send(t, map[string]any{"to": []string{"eng@consult.example"}, "subject": "RFI-001 Footing depth", "note": "Confirm the footing depth.", "rfi": rfi.ID})
	if status != http.StatusAccepted {
		t.Fatalf("send = %d %s", status, text)
	}
	var sent string
	if err := f.conn.QueryRow(ctx, `SELECT details->>'sent_message_id' FROM project_delivery_items WHERE id=$1::uuid`, rfi.ID).Scan(&sent); err != nil || sent != m.ID {
		t.Fatalf("sent message %q %v", sent, err)
	}
	reply := f.memberPost("Re: footing")
	reply.from, reply.envelope, reply.body = "eng@consult.example", "eng@consult.example", "It is 1.2 m."
	reply.headers = append(reply.headers, [2]string{"In-Reply-To", sentMessageID(t, f.mailFixture, m.ID)})
	f.mailFixture.send(t, reply)
	var doc string
	if err := f.conn.QueryRow(ctx, `SELECT p.document_id::text FROM mail_parts p JOIN mail_messages x ON x.org_id=p.org_id AND x.id=p.message_id WHERE x.message_id=$1 AND p.n=0`, reply.messageID).Scan(&doc); err != nil {
		t.Fatal(err)
	}
	var rfis []store.RegisterRFI
	var err error
	for deadline := time.Now().Add(10 * time.Second); time.Now().Before(deadline); time.Sleep(50 * time.Millisecond) {
		if _, rfis, err = f.a.store.RegisterCandidates(ctx, f.m.orgID, doc); err == nil {
			break
		}
	}
	if err != nil || len(rfis) != 1 || !rfis[0].Threaded {
		t.Fatalf("candidates %+v %v", rfis, err)
	}
	// An RFI of another project is refused.
	other := f.m.createProject(t, "Other")
	var foreignRFI struct {
		ID string `json:"id"`
	}
	f.m.call(t, http.MethodPost, "/api/projects/"+other+"/delivery", body, http.StatusCreated, &foreignRFI)
	if status, _, _ := f.send(t, map[string]any{"to": []string{"eng@consult.example"}, "rfi": foreignRFI.ID}); status != http.StatusBadRequest {
		t.Fatalf("another project's RFI = %d", status)
	}
}

// H3: From is SiteWise only; the org and the member are named in a fixed
// footer. Every org together has a daily cap.
func TestTransmittalFromAndGlobalCap(t *testing.T) {
	ctx := context.Background()
	f := newSendFixture(t, func(o *httpapi.Options) { o.MailGlobalPerDay = 2 })
	if _, err := f.conn.Exec(ctx, `UPDATE orgs SET name='Acme <attacker@evil.test> Pty' WHERE id=$1::uuid`, f.m.orgID); err != nil {
		t.Fatal(err)
	}
	_, m, text := f.send(t, map[string]any{"to": []string{"eng@consult.example"}, "subject": "One", "note": "Hello"})
	if m.ID == "" {
		t.Fatal(text)
	}
	f.waitStatus(t, m.ID, "sent")
	msg, err := mail.ReadMessage(bytes.NewReader(f.sink.Kept()[0].Msg))
	if err != nil {
		t.Fatal(err)
	}
	from, _ := mail.ParseAddress(msg.Header.Get("From"))
	if from == nil || from.Name != "SiteWise" || from.Address != mailTestFrom {
		t.Fatalf("From %q", msg.Header.Get("From"))
	}
	var body string
	_ = f.conn.QueryRow(ctx, `SELECT body FROM mail_messages WHERE id=$1::uuid`, m.ID).Scan(&body)
	if !strings.Contains(body, "Sent by "+f.email+" of Acme") {
		t.Fatalf("footer %q", body)
	}
	// Another org's send counts toward the same cap.
	other := newSendFixture(t, func(o *httpapi.Options) { o.MailGlobalPerDay = 2 })
	_ = other
	if _, err := f.conn.Exec(ctx, `INSERT INTO mail_messages(org_id,id,project_id,message_id,sender,to_addrs,status,direction,thread_id,number,send_status)
VALUES ($1::uuid, gen_random_uuid(), $2::uuid, '<g@x>', 'x@example.test', 'a@b.example', 'filed', 'out', gen_random_uuid(), 99, 'sent')`, other.m.orgID, other.project); err != nil {
		t.Fatal(err)
	}
	if status, _, text := f.send(t, map[string]any{"to": []string{"eng@consult.example"}, "subject": "Three"}); status != http.StatusTooManyRequests || !strings.Contains(text, "SiteWise has sent") {
		t.Fatalf("global cap = %d %s", status, text)
	}
}

// H3: on a free plan, sending needs an account a week old, then 10 messages
// and 50 recipients a day.
func TestTransmittalFreePlan(t *testing.T) {
	ctx := context.Background()
	f := newSendFixture(t, func(o *httpapi.Options) { o.Billing = &httpapi.Billing{} })
	to := []string{"eng@consult.example"}
	if status, _, text := f.send(t, map[string]any{"to": to}); status != http.StatusForbidden || !strings.Contains(text, "7 days") {
		t.Fatalf("new free account = %d %s", status, text)
	}
	if _, err := f.conn.Exec(ctx, `UPDATE users SET created_at=now()-interval '8 days' WHERE org_id=$1::uuid`, f.m.orgID); err != nil {
		t.Fatal(err)
	}
	if status, _, text := f.send(t, map[string]any{"to": manyAddresses(20)}); status != http.StatusAccepted {
		t.Fatalf("a week-old free account = %d %s", status, text)
	}
	for i := 0; i < 9; i++ {
		f.send(t, map[string]any{"to": to, "subject": fmt.Sprintf("n%d", i)})
	}
	if status, _, _ := f.send(t, map[string]any{"to": to}); status != http.StatusTooManyRequests {
		t.Fatalf("the 11th free send = %d", status)
	}
}

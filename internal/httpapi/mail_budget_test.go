package httpapi_test

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"path/filepath"
	"testing"
	"time"

	"sitewise/internal/latency"
	"sitewise/internal/store"
)

// TestMailBudgets holds the mail rows of bench/budgets.json on the local test
// database (email intake Task 10, correspondence Task 9): the webhook with a
// 5 MB message of two PDFs, the record read with 1,000 messages, accept and
// the two edits, and Send with two documents. Mail leaves through a sink.
// mail_send (queued to accepted) is logged, not gated: the sink accepts at
// once, so it measures this process, not a mail service.
func TestMailBudgets(t *testing.T) {
	ctx := context.Background()
	f := newSendFixture(t)
	budgets, err := latency.LoadBudgets(filepath.Join("..", "..", "bench", "budgets.json"))
	if err != nil {
		t.Fatal(err)
	}
	n := budgets.MinSamples
	if n < 10 {
		n = 10 // a cached generic plan after five calls must be inside the sample
	}
	samples := map[string][]int64{}
	timed := func(path string, run func()) {
		start := time.Now()
		run()
		samples[path] = append(samples[path], time.Since(start).Microseconds())
	}
	// 1,000 messages on the record.
	if _, err := f.conn.Exec(ctx, `INSERT INTO mail_messages(org_id,id,project_id,message_id,webhook_token,sender,subject,body,status,direction,thread_id,received_at)
SELECT $1::uuid, g.id, $2::uuid, '<bench-'||g.n||'@x.test>', 'bench-token-'||g.id, 'sender'||g.n||'@x.test', 'Bench message '||g.n, 'Body', 'filed', 'in', g.id, now() - (g.n||' minutes')::interval
FROM (SELECT gen_random_uuid() AS id, n FROM generate_series(1,1000) n) g`, f.m.orgID, f.project); err != nil {
		t.Fatal(err)
	}

	half := make([]byte, 5<<20/2)
	for i := 0; i < n; i++ {
		// The webhook: a member's 5 MB message with two PDFs. Distinct bytes
		// each time, so nothing is a repeat.
		a := append([]byte(fmt.Sprintf("%%PDF-1.4 bench %d a ", i)), half...)
		b := append([]byte(fmt.Sprintf("%%PDF-1.4 bench %d b ", i)), half...)
		p := f.memberPost(fmt.Sprintf("Bench %d", i), mailFile{field: "attachment-1", name: "A.pdf", ctype: "application/pdf", body: a}, mailFile{field: "attachment-2", name: "B.pdf", ctype: "application/pdf", body: b})
		// The post is written before the clock starts: the budget is the
		// server's, from the request to the 200.
		ctype, body := encodePost(f.fields(p), p.files)
		timed("mail_inbound_accept", func() {
			if got := f.raw(t, ctype, body); got != http.StatusOK {
				t.Fatalf("post = %d", got)
			}
		})
	}
	var first store.MailMessage
	for i := 0; i < n; i++ {
		timed("mail_read", func() {
			var l mailListBody
			f.m.getJSON(t, "/api/projects/"+f.project+"/mail", &l)
			if len(l.Messages) != 1000 {
				t.Fatalf("read %d messages", len(l.Messages))
			}
			first = l.Messages[0]
		})
		timed("mail_read", func() { f.m.getJSON(t, "/api/projects/"+f.project+"/mail/"+first.ID, new(json.RawMessage)) })
	}
	// Accept a held message with three attachments; edit a filed one's pick.
	for i := 0; i < n; i++ {
		p := f.memberPost(fmt.Sprintf("Held %d", i), pdfFile(1, fmt.Sprintf("H%d-1.pdf", i)), pdfFile(2, fmt.Sprintf("H%d-2.pdf", i)), pdfFile(3, fmt.Sprintf("H%d-3.pdf", i)))
		p.from, p.envelope = "someone@outside.test", "someone@outside.test"
		f.mailFixture.send(t, p)
		id := f.heldID(t, p.messageID)
		timed("mail_decide", func() {
			f.m.call(t, http.MethodPost, "/api/projects/"+f.project+"/mail/"+id+"/accept", nil, http.StatusOK, nil)
		})
		m := f.message(t, p.messageID)
		timed("mail_decide", func() {
			if got := f.patch(t, m.ID, map[string]any{"version": m.Version, "pick_state": "dismissed"}); got != http.StatusOK {
				t.Fatalf("patch = %d", got)
			}
		})
	}
	// Send with two documents, to the 202.
	docA := f.doc(t, f.m, f.project, "S-1.pdf", "S-1", "A", []byte("%PDF send a"))
	docB := f.doc(t, f.m, f.project, "S-2.pdf", "S-2", "A", []byte("%PDF send b"))
	var sent []string
	for i := 0; i < n; i++ {
		timed("transmittal_send", func() {
			status, m, text := f.send(t, map[string]any{"to": []string{"engineer@consult.example"}, "subject": fmt.Sprintf("Bench send %d", i), "document_ids": []string{docA, docB}})
			if status != http.StatusAccepted {
				t.Fatalf("send = %d %s", status, text)
			}
			sent = append(sent, m.ID)
		})
	}
	var queued []int64
	for _, id := range sent {
		m := f.waitStatus(t, id, "sent")
		if m.SentAt != nil {
			var q store.MailMessage
			f.m.getJSON(t, "/api/projects/"+f.project+"/mail/"+id, &q)
			queued = append(queued, m.SentAt.Sub(q.ReceivedAt).Microseconds())
		}
	}
	if p50, err := latency.Percentile(queued, 0.5); err == nil {
		p90, _ := latency.Percentile(queued, 0.9)
		t.Logf("mail_send (sink, not gated): %d samples, p50 %d us, p90 %d us", len(queued), p50, p90)
	}
	var scoped latency.Budgets
	scoped.MinSamples = budgets.MinSamples
	for _, p := range budgets.Paths {
		if _, ok := samples[p.Name]; ok {
			scoped.Paths = append(scoped.Paths, p)
		}
	}
	for name, got := range samples {
		p50, _ := latency.Percentile(got, 0.5)
		p90, _ := latency.Percentile(got, 0.9)
		t.Logf("%s: %d samples, p50 %d us, p90 %d us", name, len(got), p50, p90)
	}
	if code, report := latency.Gate(scoped, samples, true); code != 0 {
		t.Fatal(report)
	}
	// The webhook, the record and Send never call Jev while a request waits;
	// filing (the accepted documents) runs after and is not counted here.

}

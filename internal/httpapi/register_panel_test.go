package httpapi_test

import (
	"context"
	"net/http"
	"os"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"

	"sitewise/internal/chatfix"
	"sitewise/internal/delivery"
	"sitewise/internal/store"
)

// The register panel (owner, 20 November 2026): an RFI's To, Cc and Subject
// are saved on the row through delivery.edit with Undo, checked as the send
// route checks them, and the register read carries each sent RFI's thread.

type panelItem struct {
	delivery.Item
	Details struct {
		To      []string `json:"to"`
		Cc      []string `json:"cc"`
		Subject string   `json:"subject"`
	} `json:"details"`
}

func readPanel(t *testing.T, m *member, project string) (map[string]panelItem, string) {
	t.Helper()
	var view struct {
		Items   []panelItem `json:"items"`
		SendOff string      `json:"send_off"`
	}
	m.getJSON(t, "/api/projects/"+project+"/delivery", &view)
	out := map[string]panelItem{}
	for _, it := range view.Items {
		out[it.Title] = it
	}
	return out, view.SendOff
}

func testPool(t *testing.T) *pgxpool.Pool {
	t.Helper()
	pool, err := pgxpool.New(context.Background(), os.Getenv("SITEWISE_TEST_DATABASE_URL"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(pool.Close)
	return pool
}

func TestRFIAddressingSavedWithUndo(t *testing.T) {
	a := newApp(t, withProfile(t))
	m := a.member(t, newUUID(t))
	project := m.createProject(t, "RFI addressing")
	pool := testPool(t)
	if _, err := pool.Exec(context.Background(), `INSERT INTO firms(org_id,id,name,name_key,email,status) VALUES ($1::uuid,gen_random_uuid(),'Northline Structural','northlinestructural','eng@northline.test','draft')`, m.orgID); err != nil {
		t.Fatal(err)
	}
	m.ops(t, project, "delivery", http.StatusOK, op("delivery.add", map[string]any{"kind": "rfi", "title": "Slab edge", "raised_on": "2026-11-20"}))

	// The screen sends lists; a firm's name resolves to its recorded address.
	saved := m.ops(t, project, "delivery", http.StatusOK, op("delivery.edit", map[string]any{"item": "RFI-001",
		"to": []string{"Northline Structural", "site@builder.test"}, "cc": []string{"pm@team.test"}, "subject": "RFI-001 Slab edge, please"}))
	items, off := readPanel(t, m, project)
	got := items["Slab edge"].Details
	if strings.Join(got.To, "|") != `"Northline Structural" <eng@northline.test>|site@builder.test` || strings.Join(got.Cc, "|") != "pm@team.test" || got.Subject != "RFI-001 Slab edge, please" {
		t.Fatalf("saved %+v", got)
	}
	// Without an outbox the read says Send is off, in plain words.
	if off == "" {
		t.Fatal("the read does not say sending is off")
	}
	// What the send route refuses, the edit refuses, and nothing changes.
	twentyOne := make([]string, 21)
	for i := range twentyOne {
		twentyOne[i] = "a" + strings.Repeat("x", i) + "@x.test"
	}
	for name, fields := range map[string]map[string]any{
		"a line break":    {"to": []string{"eng@x.test\r\nBcc: spy@x.test"}},
		"not an address":  {"cc": []string{"nobody at all"}},
		"21 addresses":    {"to": twentyOne},
		"a subject break": {"subject": "Two\nlines"},
	} {
		fields["item"] = "RFI-001"
		m.ops(t, project, "delivery", http.StatusUnprocessableEntity, op("delivery.edit", fields))
		items, _ = readPanel(t, m, project)
		if after := items["Slab edge"].Details; strings.Join(after.To, "|") != strings.Join(got.To, "|") || after.Subject != got.Subject {
			t.Fatalf("%s changed the row: %+v", name, after)
		}
	}
	// Undo puts the row back as it was: no addresses.
	m.undo(t, project, saved.ChangeID, http.StatusOK)
	items, _ = readPanel(t, m, project)
	if d := items["Slab edge"].Details; len(d.To) != 0 || len(d.Cc) != 0 || d.Subject != "" {
		t.Fatalf("after undo %+v", d)
	}
	if store.MailMaxRecipients != delivery.MaxRecipients {
		t.Fatal("the RFI's address limit is not the send route's")
	}
}

// The thread: what went out and the filed replies in its thread, from one
// query for the whole register; another project's RFI naming this
// project's message gets nothing.
func TestRegisterReadCarriesRFIThreads(t *testing.T) {
	a := newApp(t, withProfile(t))
	m := a.member(t, newUUID(t))
	project := m.createProject(t, "RFI thread")
	other := m.createProject(t, "Other job")
	ctx := context.Background()
	docs, err := chatfix.SeedDocuments(ctx, a.store, m.orgID, project, []chatfix.Document{
		{Title: "Detail", Kind: "drawing", Pages: []string{"Detail."}},
		{Title: "Re RFI-001", Kind: "correspondence", Pages: []string{"Use the sheet membrane."}},
	})
	if err != nil {
		t.Fatal(err)
	}
	m.ops(t, project, "delivery", http.StatusOK, op("delivery.add", map[string]any{"kind": "rfi", "title": "Membrane", "raised_on": "2026-11-20"}))
	m.ops(t, other, "delivery", http.StatusOK, op("delivery.add", map[string]any{"kind": "rfi", "title": "Theirs", "raised_on": "2026-11-20"}))
	pool := testPool(t)
	out, in := newUUID(t), newUUID(t)
	sum := strings.Repeat("ab", 32)
	for _, q := range []struct {
		sql  string
		args []any
	}{
		{`INSERT INTO mail_messages(org_id,id,project_id,message_id,webhook_token,sender,to_addrs,status,direction,thread_id,number,send_status,received_at)
 VALUES ($1::uuid,$2::uuid,$3::uuid,'<out@e2e>',NULL,'sitewise@x.test','eng@consult.test','filed','out',$2::uuid,1,'sent',now()-interval '2 hours')`, []any{m.orgID, out, project}},
		{`INSERT INTO mail_parts(org_id,message_id,n,filename,sha256) VALUES ($1::uuid,$2::uuid,0,'TR-001.pdf',decode($3,'hex'))`, []any{m.orgID, out, sum}},
		{`INSERT INTO mail_parts(org_id,message_id,n,filename,sha256,document_id) VALUES ($1::uuid,$2::uuid,1,'detail.pdf',decode($3,'hex'),$4::uuid)`, []any{m.orgID, out, sum, docs["Detail"]}},
		{`INSERT INTO mail_messages(org_id,id,project_id,message_id,webhook_token,sender,sender_name,body,status,direction,thread_id,received_at)
 VALUES ($1::uuid,$2::uuid,$3::uuid,'<in@e2e>','tok-` + in + `','eng@consult.test','Consult Eng',$4,'filed','in',$5::uuid,now())`, []any{m.orgID, in, project, "Use the sheet membrane.\n\nLaps 150 mm.\n\nOn Mon, PM wrote:\n> Which membrane?", out}},
		{`INSERT INTO mail_parts(org_id,message_id,n,filename,sha256,document_id) VALUES ($1::uuid,$2::uuid,0,'body.txt',decode($3,'hex'),$4::uuid),($1::uuid,$2::uuid,1,'photo.jpg',decode($3,'hex'),NULL)`, []any{m.orgID, in, sum, docs["Re RFI-001"]}},
		{`UPDATE project_delivery_items SET details = details || jsonb_build_object('sent_message_id',$2::text) WHERE org_id=$1::uuid AND kind='rfi'`, []any{m.orgID, out}},
		{`INSERT INTO delivery_readings(org_id,id,project_id,item_id,document_id,field,decided_by,band,basis)
 SELECT org_id,gen_random_uuid(),project_id,id,$3::uuid,'answers','rule','amber','a reply in the RFI''s thread' FROM project_delivery_items WHERE org_id=$1::uuid AND project_id=$2::uuid`, []any{m.orgID, project, docs["Re RFI-001"]}},
	} {
		if _, err := pool.Exec(ctx, q.sql, q.args...); err != nil {
			t.Fatal(err)
		}
	}
	items, _ := readPanel(t, m, project)
	thread := items["Membrane"].Thread
	if len(thread) != 2 {
		t.Fatalf("thread %+v", thread)
	}
	sent, reply := thread[0], thread[1]
	if sent.Direction != "out" || sent.With != "eng@consult.test" || sent.Documents != 1 || sent.Number == nil || *sent.Number != 1 || sent.SendStatus != "sent" {
		t.Fatalf("sent %+v", sent)
	}
	if reply.Direction != "in" || reply.With != "Consult Eng" || reply.Excerpt != "Use the sheet membrane.\nLaps 150 mm." || reply.Documents != 1 || reply.BodyDocument != docs["Re RFI-001"] || reply.Reading == "" {
		t.Fatalf("reply %+v", reply)
	}
	theirs, _ := readPanel(t, m, other)
	if got := theirs["Theirs"].Thread; len(got) != 0 {
		t.Fatalf("another project's RFI read this project's mail: %+v", got)
	}
}

// Chat may set an RFI's To only to an address the PM typed, or the
// recorded address of a project firm the PM named (Send's guard).
func TestChatRFIAddressesAreThePMs(t *testing.T) {
	ta := newToolApp(t)
	pool := testPool(t)
	ctx := context.Background()
	firm, pkg := newUUID(t), newUUID(t)
	for _, q := range []struct {
		sql  string
		args []any
	}{
		{`INSERT INTO firms(org_id,id,name,name_key,email,status) VALUES ($1::uuid,$2::uuid,'Northline Structural','northlinestructural','eng@northline.test','draft')`, []any{ta.m.orgID, firm}},
		{`INSERT INTO packages(org_id,id,project_id,kind,title,lifecycle_status,origin,review_status) VALUES ($1::uuid,$2::uuid,$3::uuid,'services','Structure','planned','user','accepted_for_planning')`, []any{ta.m.orgID, pkg, ta.project}},
		{`INSERT INTO package_invitations(org_id,project_id,package_id,firm_id) VALUES ($1::uuid,$2::uuid,$3::uuid,$4::uuid)`, []any{ta.m.orgID, ta.project, pkg, firm}},
	} {
		if _, err := pool.Exec(ctx, q.sql, q.args...); err != nil {
			t.Fatal(err)
		}
	}
	ta.m.ops(t, ta.project, "delivery", http.StatusOK, op("delivery.add", map[string]any{"kind": "rfi", "title": "Slab edge", "raised_on": "2026-11-20"}))
	edit := func(message string, fields map[string]any) []chatEvent {
		fields["item"] = "RFI-001"
		return ta.run(t, message, tool("apply", map[string]any{"area": "delivery", "ops": []map[string]any{op("delivery.edit", fields)}, "final": true}))
	}
	applied := func(events []chatEvent) bool {
		for _, e := range events {
			if e.Type == "change" {
				return true
			}
		}
		return false
	}
	// An address the model made up is refused.
	if applied(edit("Address RFI-001 to the engineer", map[string]any{"to": []string{"collect@outside.test"}})) || !strings.Contains(ta.results[0], "not in the PM's message") {
		t.Fatalf("an invented address was used: %q", ta.results)
	}
	// The PM's own address, and a project firm the PM named, are used.
	if !applied(edit("Send RFI-001 to site@builder.test", map[string]any{"to": []string{"site@builder.test"}})) {
		t.Fatalf("the PM's address was refused: %q", ta.results)
	}
	if !applied(edit("Address RFI-001 to Northline Structural", map[string]any{"to": "Northline Structural"})) {
		t.Fatalf("a firm the PM named was refused: %q", ta.results)
	}
	items, _ := readPanel(t, ta.m, ta.project)
	if to := items["Slab edge"].Details.To; len(to) != 1 || !strings.Contains(to[0], "eng@northline.test") {
		t.Fatalf("to %v", to)
	}
}

// Chat's Send for an RFI opens that RFI's sheet (the screen does it, from
// the rfi field), and the screen line names an open register row.
func TestChatOpensTheRFISheetAndSeesIt(t *testing.T) {
	ta := newToolApp(t)
	ta.m.ops(t, ta.project, "delivery", http.StatusOK, op("delivery.add", map[string]any{"kind": "rfi", "title": "Slab edge", "raised_on": "2026-11-20"}))
	items, _ := readPanel(t, ta.m, ta.project)
	id := items["Slab edge"].ID
	events := ta.run(t, "Send RFI-001 to site@builder.test", tool("view", map[string]any{"do": "open", "dialog": "send", "fields": map[string]string{"rfi": "RFI-001", "to": "site@builder.test", "subject": "RFI-001 Slab edge"}, "final": true}))
	var view *chatEvent
	for i := range events {
		if events[i].Type == "view" {
			view = &events[i]
		}
	}
	if view == nil || view.View.Fields["rfi"] != id || view.View.Fields["to"] != "site@builder.test" || !strings.Contains(view.View.Line, "Opened RFI-001 to send") {
		t.Fatalf("events %+v results %q", view, ta.results)
	}
	ta.calls = nil
	ta.m.chatOn(t, ta.project, "", "What is open?", map[string]any{"view": "profile", "open": map[string]any{"kind": "register", "id": id}})
	if line := onScreenOf(ta.model.sent[len(ta.model.sent)-1]); !strings.Contains(line, "Open: register row "+id[:8]) {
		t.Fatalf("screen line %q", line)
	}
}

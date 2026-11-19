package store_test

import (
	"context"
	"testing"
	"time"

	"sitewise/internal/store"
)

// Record fields worked out at record time (correspondence plan, Task 1).

const (
	firmAcme   = "11111111-1111-4111-8111-aaaaaaaaf001"
	firmBolt   = "11111111-1111-4111-8111-aaaaaaaaf002"
	firmGmail  = "11111111-1111-4111-8111-aaaaaaaaf003"
	firmOther  = "22222222-2222-4222-8222-bbbbbbbbf004"
	pkgRoof    = "11111111-1111-4111-8111-aaaaaaaae001"
	pkgFacade  = "11111111-1111-4111-8111-aaaaaaaae002"
	pkgForeign = "22222222-2222-4222-8222-bbbbbbbbe003"
)

func seedMailFirms(t *testing.T, st *store.Store) {
	t.Helper()
	ctx := context.Background()
	pool := rawPool(t)
	for _, q := range []struct {
		sql  string
		args []any
	}{
		{`INSERT INTO packages(org_id,id,project_id,kind,title,lifecycle_status,origin,review_status) VALUES ($1,$2,$3,'services','Roofing','planned','user','accepted_for_planning'),($1,$4,$3,'services','Facade','planned','user','accepted_for_planning')`, []any{orgA, pkgRoof, projectA, pkgFacade}},
		{`INSERT INTO packages(org_id,id,project_id,kind,title,lifecycle_status,origin,review_status) VALUES ($1,$2,$3,'services','Theirs','planned','user','accepted_for_planning')`, []any{orgB, pkgForeign, projectB}},
		{`INSERT INTO firms(org_id,id,name,name_key,email,status) VALUES ($1,$2,'Acme Roofing','acmeroofing','Estimating@Acme.example','draft'),($1,$3,'Bolt Facades','boltfacades','info@bolt.example','draft'),($1,$4,'Solo','solo','solo.builder@gmail.com','draft')`, []any{orgA, firmAcme, firmBolt, firmGmail}},
		{`INSERT INTO firms(org_id,id,name,name_key,email,status) VALUES ($1,$2,'Other org firm','otherorgfirm','someone@other.example','draft')`, []any{orgB, firmOther}},
		{`INSERT INTO package_invitations(org_id,project_id,package_id,firm_id) VALUES ($1,$2,$3,$4),($1,$2,$3,$5),($1,$2,$6,$5)`, []any{orgA, projectA, pkgRoof, firmAcme, firmBolt, pkgFacade}},
	} {
		if _, err := pool.Exec(ctx, q.sql, q.args...); err != nil {
			t.Fatal(err)
		}
	}
	_ = st
}

func recordFrom(t *testing.T, st *store.Store, token, messageID, sender, inReplyTo, refs string) store.MailMessage {
	t.Helper()
	ctx := context.Background()
	in := mailIn(token, messageID, "")
	in.Sender, in.InReplyTo, in.References = sender, inReplyTo, refs
	in.ReceivedAt = time.Now()
	r, err := st.RecordMail(ctx, orgA, in)
	if err != nil {
		t.Fatal(err)
	}
	m, err := st.MailMessageByID(ctx, orgA, projectA, r.ID)
	if err != nil {
		t.Fatal(err)
	}
	return m
}

func TestMailThreads(t *testing.T) {
	st := profileStore(t)
	seedMailFirms(t, st)
	first := recordFrom(t, st, "th-1", "<a1@x.example>", "pm@x.example", "", "")
	if first.ThreadID != first.ID {
		t.Fatalf("a first message starts its own thread: %+v", first)
	}
	reply := recordFrom(t, st, "th-2", "<a2@x.example>", "pm@x.example", "<a1@x.example>", "")
	if reply.ThreadID != first.ID {
		t.Fatal("In-Reply-To did not thread")
	}
	// Outlook drops In-Reply-To's parent here: References, last to first.
	byRefs := recordFrom(t, st, "th-3", "<a3@x.example>", "pm@x.example", "<unknown@x.example>", "<a1@x.example> <a2@x.example>")
	if byRefs.ThreadID != first.ID {
		t.Fatal("References did not thread")
	}
	early := recordFrom(t, st, "th-4", "<b2@x.example>", "pm@x.example", "<b1@x.example>", "")
	if early.ThreadID != early.ID {
		t.Fatal("a reply before its parent should start its own thread")
	}
	parent := recordFrom(t, st, "th-5", "<b1@x.example>", "pm@x.example", "", "")
	again, _ := st.MailMessageByID(context.Background(), orgA, projectA, early.ID)
	if parent.ThreadID != parent.ID || again.ThreadID != early.ID {
		t.Fatal("a late parent re-threaded its earlier reply")
	}
}

func TestMailFirmAndPackage(t *testing.T) {
	st := profileStore(t)
	seedMailFirms(t, st)
	exact := recordFrom(t, st, "fp-1", "<f1@acme.example>", "estimating@acme.example", "", "")
	if exact.FirmID != firmAcme || exact.PackageID != pkgRoof || exact.Package != "Roofing" || exact.LinkedBy != "rule" {
		t.Fatalf("exact match %+v", exact)
	}
	domain := recordFrom(t, st, "fp-2", "<f2@acme.example>", "jo@acme.example", "", "")
	if domain.FirmID != firmAcme {
		t.Fatalf("domain match %+v", domain)
	}
	two := recordFrom(t, st, "fp-3", "<f3@bolt.example>", "info@bolt.example", "", "")
	if two.FirmID != firmBolt || two.PackageID != "" {
		t.Fatalf("a firm on two packages leaves the package blank: %+v", two)
	}
	gmail := recordFrom(t, st, "fp-4", "<f4@gmail.example>", "someone.else@gmail.com", "", "")
	if gmail.FirmID != "" {
		t.Fatalf("free mail matched by domain: %+v", gmail)
	}
	solo := recordFrom(t, st, "fp-5", "<f5@gmail.example>", "solo.builder@gmail.com", "", "")
	if solo.FirmID != firmGmail {
		t.Fatalf("free mail exact address: %+v", solo)
	}
	foreign := recordFrom(t, st, "fp-6", "<f6@other.example>", "someone@other.example", "", "")
	if foreign.FirmID != "" {
		t.Fatalf("another org's firm matched: %+v", foreign)
	}
	// No firm: the thread's package.
	follow := recordFrom(t, st, "fp-7", "<f7@x.example>", "pm@x.example", "<f1@acme.example>", "")
	if follow.FirmID != "" || follow.PackageID != pkgRoof {
		t.Fatalf("thread package %+v", follow)
	}
	// A member's own copy is Out; its firm is the recipient's.
	in := mailIn("fp-8", "<f8@x.example>", "")
	in.Direction, in.Sender, in.To = "out", "pm@x.example", "Acme <estimating@acme.example>"
	r, err := st.RecordMail(context.Background(), orgA, in)
	if err != nil {
		t.Fatal(err)
	}
	out, _ := st.MailMessageByID(context.Background(), orgA, projectA, r.ID)
	if out.Direction != "out" || out.FirmID != firmAcme || out.With != in.To {
		t.Fatalf("out %+v", out)
	}
	// Replied is computed: the first Acme message has a later Out in its thread? No:
	// fp-8 is its own thread. A reply from the team to f1 marks it Replied.
	reply := mailIn("fp-9", "<f9@x.example>", "")
	reply.Direction, reply.Sender, reply.InReplyTo = "out", "pm@x.example", "<f1@acme.example>"
	if _, err := st.RecordMail(context.Background(), orgA, reply); err != nil {
		t.Fatal(err)
	}
	answered, _ := st.MailMessageByID(context.Background(), orgA, projectA, exact.ID)
	unanswered, _ := st.MailMessageByID(context.Background(), orgA, projectA, two.ID)
	if !answered.Replied || unanswered.Replied {
		t.Fatalf("replied %v %v", answered.Replied, unanswered.Replied)
	}
	list, err := st.MailMessages(context.Background(), orgA, projectA, store.MailFilter{Unanswered: true, FirmID: firmBolt})
	if err != nil || len(list) != 1 || list[0].ID != two.ID {
		t.Fatalf("unanswered from Bolt: %+v %v", list, err)
	}
}

func TestMailOutsideAdmitted(t *testing.T) {
	ctx := context.Background()
	st := profileStore(t)
	seedMailFirms(t, st)
	for _, c := range []struct {
		sender, inReplyTo string
		want              bool
	}{
		{"estimating@acme.example", "", true},   // a firm's exact address on this project
		{"jo@acme.example", "", false},          // a domain links, never admits
		{"someone@other.example", "", false},    // another org's firm
		{"stranger@nowhere.example", "", false}, // nobody
		{"stranger@nowhere.example", "<x@y>", false},
	} {
		ok, err := st.MailOutsideAdmitted(ctx, orgA, projectA, c.sender, c.inReplyTo, "")
		if err != nil || ok != c.want {
			t.Errorf("%s: got %v %v", c.sender, ok, err)
		}
	}
	if _, err := st.MailOutsideAdmitted(ctx, orgB, projectA, "estimating@acme.example", "", ""); err != nil {
		t.Fatal(err)
	} else if ok, _ := st.MailOutsideAdmitted(ctx, orgB, projectA, "estimating@acme.example", "", ""); ok {
		t.Fatal("admitted through another org")
	}
}

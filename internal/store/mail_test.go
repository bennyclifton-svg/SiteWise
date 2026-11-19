package store_test

import (
	"context"
	"errors"
	"regexp"
	"strings"
	"testing"
	"time"

	"sitewise/internal/store"
)

var localPartPattern = regexp.MustCompile(`^p-[0-9a-f]{32}$`)

func TestProjectGetsMailAddress(t *testing.T) {
	ctx := context.Background()
	st := profileStore(t)
	const second = "11111111-1111-4111-8111-aaaaaaaaab01"
	if err := st.CreateProject(ctx, orgA, second, "Project"); err != nil {
		t.Fatal(err)
	}
	a, err := st.MailAddress(ctx, orgA, projectA)
	if err != nil {
		t.Fatal(err)
	}
	b, err := st.MailAddress(ctx, orgA, second)
	if err != nil {
		t.Fatal(err)
	}
	if !localPartPattern.MatchString(a) || !localPartPattern.MatchString(b) || a == b {
		t.Fatalf("addresses %q %q", a, b)
	}
	if strings.Contains(a, "project") {
		t.Fatalf("the address names the project: %q", a)
	}
	got, err := st.ProjectByMailAddress(ctx, strings.ToUpper(a))
	if err != nil || got.OrgID != orgA || got.ProjectID != projectA || got.Archived {
		t.Fatalf("lookup %+v %v", got, err)
	}
	if _, err := st.ProjectByMailAddress(ctx, "p-00000000000000000000000000000000"); !errors.Is(err, store.ErrNotFound) {
		t.Fatalf("unknown address: %v", err)
	}
	if _, err := st.MailAddress(ctx, orgB, projectA); !errors.Is(err, store.ErrNotFound) {
		t.Fatalf("another org's project: %v", err)
	}
	// A project row made without CreateProject gets its address on first read.
	pool := rawPool(t)
	if _, err := pool.Exec(ctx, `DELETE FROM project_mail_addresses WHERE org_id=$1::uuid AND project_id=$2::uuid`, orgA, second); err != nil {
		t.Fatal(err)
	}
	if c, err := st.MailAddress(ctx, orgA, second); err != nil || !localPartPattern.MatchString(c) || c == b {
		t.Fatalf("lazy address %q %v", c, err)
	}
}

func TestRotateMailAddress(t *testing.T) {
	ctx := context.Background()
	st := profileStore(t)
	old, err := st.MailAddress(ctx, orgA, projectA)
	if err != nil {
		t.Fatal(err)
	}
	fresh, err := st.RotateMailAddress(ctx, orgA, projectA)
	if err != nil || fresh == old || !localPartPattern.MatchString(fresh) {
		t.Fatalf("rotate %q %v", fresh, err)
	}
	if _, err := st.ProjectByMailAddress(ctx, old); !errors.Is(err, store.ErrNotFound) {
		t.Fatalf("old address still found: %v", err)
	}
	if got, err := st.ProjectByMailAddress(ctx, fresh); err != nil || got.ProjectID != projectA {
		t.Fatalf("new address %+v %v", got, err)
	}
	if _, err := st.RotateMailAddress(ctx, orgB, projectA); !errors.Is(err, store.ErrNotFound) {
		t.Fatalf("rotate another org's project: %v", err)
	}
}

func mailIn(token, messageID, reason string, parts ...store.MailPartIn) store.MailIn {
	return store.MailIn{ProjectID: projectA, MessageID: messageID, Token: token, Sender: "someone@example.net",
		Subject: "Drawings", Body: "See attached.", ReceivedAt: time.Now(), HeldReason: reason, Parts: parts}
}

func part(name string, b byte, size int64) store.MailPartIn {
	return store.MailPartIn{Filename: name, ContentType: "application/pdf", SHA256: bytes32(b), Size: size}
}

func TestRecordMailIsIdempotent(t *testing.T) {
	ctx := context.Background()
	st := profileStore(t)
	first, err := st.RecordMail(ctx, orgA, mailIn("tok-1", "<m1@example.net>", "", part("a.pdf", 0x71, 10)))
	if err != nil || first.Repeat || first.Status != "filed" {
		t.Fatalf("first %+v %v", first, err)
	}
	again, err := st.RecordMail(ctx, orgA, mailIn("tok-1", "<m1@example.net>", ""))
	if err != nil || !again.Repeat {
		t.Fatalf("same token %+v %v", again, err)
	}
	same, err := st.RecordMail(ctx, orgA, mailIn("tok-2", "<m1@example.net>", ""))
	if err != nil || !same.Repeat || same.ID != first.ID {
		t.Fatalf("same Message-Id %+v %v", same, err)
	}
	list, err := st.MailMessages(ctx, orgA, projectA, store.MailFilter{})
	if err != nil || len(list) != 1 || len(list[0].Parts) != 1 {
		t.Fatalf("list %+v %v", list, err)
	}
	// A token seen by the webhook for a post that stored nothing is also a repeat.
	seen, err := st.SeenWebhookToken(ctx, "tok-3")
	if err != nil || seen {
		t.Fatalf("first sight %v %v", seen, err)
	}
	if r, err := st.RecordMail(ctx, orgA, mailIn("tok-3", "<m3@example.net>", "")); err != nil || !r.Repeat {
		t.Fatalf("token seen before %+v %v", r, err)
	}
}

func TestHeldMailCaps(t *testing.T) {
	ctx := context.Background()
	st := profileStore(t)
	for i := 0; i < store.MailHeldMax; i++ {
		id := string(rune('a'+i%26)) + string(rune('a'+i/26))
		if _, err := st.RecordMail(ctx, orgA, mailIn("held-"+id, "<h"+id+"@example.net>", store.MailUnknownSender, part("x.pdf", 0x72, 1))); err != nil {
			t.Fatal(err)
		}
	}
	full, err := st.RecordMail(ctx, orgA, mailIn("held-full", "<hfull@example.net>", store.MailUnknownSender, part("y.pdf", 0x73, 5)))
	if err != nil || full.HeldReason != store.MailHeldFull || len(full.Unkept) != 1 {
		t.Fatalf("51st held %+v %v", full, err)
	}
	m, err := st.MailMessageByID(ctx, orgA, projectA, full.ID)
	if err != nil || m.Parts[0].Skipped != "not_kept" {
		t.Fatalf("parts %+v %v", m, err)
	}
	// Bytes past 500 MB are held full too, on a project with room for messages.
	big, err := st.RecordMail(ctx, orgB, store.MailIn{ProjectID: projectB, MessageID: "<big@example.net>", Token: "big", Sender: "x@example.net",
		HeldReason: store.MailUnknownSender, Parts: []store.MailPartIn{{Filename: "huge.pdf", SHA256: bytes32(0x74), Size: store.MailHeldBytes + 1}}})
	if err != nil || big.HeldReason != store.MailHeldFull {
		t.Fatalf("bytes cap %+v %v", big, err)
	}
}

func TestHeldMailExpires(t *testing.T) {
	ctx := context.Background()
	st := profileStore(t)
	old := mailIn("old", "<old@example.net>", store.MailUnknownSender, part("old.pdf", 0x75, 3))
	old.ReceivedAt = time.Now().AddDate(0, 0, -31)
	if _, err := st.RecordMail(ctx, orgA, old); err != nil {
		t.Fatal(err)
	}
	r, err := st.RecordMail(ctx, orgA, mailIn("new", "<new@example.net>", store.MailUnknownSender))
	if err != nil || len(r.Expired) != 1 || r.Expired[0][0] != 0x75 {
		t.Fatalf("expire %+v %v", r, err)
	}
	if ref, err := st.BlobReferenced(ctx, bytes32(0x75)); err != nil || ref {
		t.Fatalf("expired bytes still referenced: %v %v", ref, err)
	}
}

func TestHeldMailIsScoped(t *testing.T) {
	ctx := context.Background()
	st := profileStore(t)
	const other = "11111111-1111-4111-8111-aaaaaaaaab02"
	if err := st.CreateProject(ctx, orgA, other, "Other"); err != nil {
		t.Fatal(err)
	}
	r, err := st.RecordMail(ctx, orgA, mailIn("scoped", "<s@example.net>", store.MailUnknownSender, part("s.pdf", 0x76, 3)))
	if err != nil {
		t.Fatal(err)
	}
	for _, c := range []struct{ org, project string }{{orgB, projectA}, {orgA, other}, {orgB, projectB}} {
		if _, err := st.HeldForDecision(ctx, c.org, c.project, r.ID); !errors.Is(err, store.ErrNotFound) {
			t.Fatalf("read from %v: %v", c, err)
		}
		if err := st.MarkMailFiled(ctx, c.org, c.project, r.ID, ""); !errors.Is(err, store.ErrNotFound) {
			t.Fatalf("mark from %v: %v", c, err)
		}
		if _, err := st.DiscardMail(ctx, c.org, c.project, r.ID); !errors.Is(err, store.ErrNotFound) {
			t.Fatalf("discard from %v: %v", c, err)
		}
		list, err := st.MailMessages(ctx, c.org, c.project, store.MailFilter{})
		if err != nil || len(list) != 0 {
			t.Fatalf("list from %v: %+v %v", c, list, err)
		}
	}
	if _, err := st.HeldForDecision(ctx, orgA, projectA, "not-a-uuid"); !errors.Is(err, store.ErrNotFound) {
		t.Fatalf("malformed id: %v", err)
	}
}

func TestBlobReferencedCountsHeldMail(t *testing.T) {
	ctx := context.Background()
	st := profileStore(t)
	r, err := st.RecordMail(ctx, orgA, mailIn("ref", "<ref@example.net>", store.MailUnknownSender, part("r.pdf", 0x77, 3)))
	if err != nil {
		t.Fatal(err)
	}
	if ref, err := st.BlobReferenced(ctx, bytes32(0x77)); err != nil || !ref {
		t.Fatalf("held part not referenced: %v %v", ref, err)
	}
	sums, err := st.DiscardMail(ctx, orgA, projectA, r.ID)
	if err != nil || len(sums) != 1 {
		t.Fatalf("discard %v %v", sums, err)
	}
	if ref, err := st.BlobReferenced(ctx, bytes32(0x77)); err != nil || ref {
		t.Fatalf("discarded part still referenced: %v %v", ref, err)
	}
}

// A filed message is a record: no edit, no delete, no discard (A7).
func TestFiledMailIsFrozen(t *testing.T) {
	ctx := context.Background()
	st := profileStore(t)
	r, err := st.RecordMail(ctx, orgA, mailIn("frozen", "<f@example.net>", "", part("f.pdf", 0x78, 3)))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := st.DiscardMail(ctx, orgA, projectA, r.ID); !errors.Is(err, store.ErrMailNotHeld) {
		t.Fatalf("discard filed: %v", err)
	}
	pool := rawPool(t)
	for _, q := range []string{
		`UPDATE mail_messages SET subject='changed' WHERE org_id=$1::uuid AND id=$2::uuid`,
		`UPDATE mail_messages SET status='held', held_reason='busy' WHERE org_id=$1::uuid AND id=$2::uuid`,
		`DELETE FROM mail_messages WHERE org_id=$1::uuid AND id=$2::uuid`,
		`UPDATE mail_parts SET filename='x.pdf' WHERE org_id=$1::uuid AND message_id=$2::uuid`,
		`DELETE FROM mail_parts WHERE org_id=$1::uuid AND message_id=$2::uuid`,
	} {
		if _, err := pool.Exec(ctx, q, orgA, r.ID); err == nil {
			t.Fatalf("allowed: %s", q)
		}
	}
}

func TestDeleteProjectRemovesMail(t *testing.T) {
	ctx := context.Background()
	st := profileStore(t)
	const doomed = "11111111-1111-4111-8111-aaaaaaaaab03"
	if err := st.CreateProject(ctx, orgA, doomed, "Doomed"); err != nil {
		t.Fatal(err)
	}
	local, err := st.MailAddress(ctx, orgA, doomed)
	if err != nil {
		t.Fatal(err)
	}
	in := mailIn("doomed-1", "<d1@example.net>", "", part("d.pdf", 0x79, 3))
	in.ProjectID = doomed
	if _, err := st.RecordMail(ctx, orgA, in); err != nil {
		t.Fatal(err)
	}
	held := mailIn("doomed-2", "<d2@example.net>", store.MailUnknownSender, part("e.pdf", 0x7a, 3))
	held.ProjectID = doomed
	if _, err := st.RecordMail(ctx, orgA, held); err != nil {
		t.Fatal(err)
	}
	res, err := st.DeleteProject(ctx, orgA, doomed)
	if err != nil {
		t.Fatal(err)
	}
	found := map[byte]bool{}
	for _, s := range res.Unreferenced {
		found[s[0]] = true
	}
	if !found[0x79] || !found[0x7a] {
		t.Fatalf("unreferenced %v", res.Unreferenced)
	}
	if _, err := st.ProjectByMailAddress(ctx, local); !errors.Is(err, store.ErrNotFound) {
		t.Fatalf("address survived: %v", err)
	}
	if ref, _ := st.BlobReferenced(ctx, bytes32(0x7a)); ref {
		t.Fatal("held bytes still referenced")
	}
}

package store_test

import (
	"context"
	"errors"
	"net/url"
	"os"
	"strings"
	"testing"

	"sitewise/internal/store"
)

const (
	orgA = "11111111-1111-4111-8111-111111111111"
	orgB = "22222222-2222-4222-8222-222222222222"

	userA = "11111111-1111-4111-8111-aaaaaaaaaaaa"
	userB = "22222222-2222-4222-8222-bbbbbbbbbbbb"

	projectA = "11111111-1111-4111-8111-aaaaaaaaaa01"
	projectB = "22222222-2222-4222-8222-bbbbbbbbbb02"

	fileA = "11111111-1111-4111-8111-aaaaaaaaaa11"
	fileB = "22222222-2222-4222-8222-bbbbbbbbbb12"

	docA = "11111111-1111-4111-8111-aaaaaaaaaa21"
	docB = "22222222-2222-4222-8222-bbbbbbbbbb22"

	decisionA = "11111111-1111-4111-8111-aaaaaaaaaa31"
	decisionB = "22222222-2222-4222-8222-bbbbbbbbbb32"

	passageA = "11111111-1111-4111-8111-aaaaaaaaaa41"
	passageB = "22222222-2222-4222-8222-bbbbbbbbbb42"

	jobA     = "11111111-1111-4111-8111-aaaaaaaaaa51"
	jobCross = "11111111-1111-4111-8111-aaaaaaaaaa52"

	inviteA = "11111111-1111-4111-8111-aaaaaaaaaa61"
	inviteB = "22222222-2222-4222-8222-bbbbbbbbbb62"

	sessionA = "11111111-1111-4111-8111-aaaaaaaaaa71"
	sessionB = "22222222-2222-4222-8222-bbbbbbbbbb72"

	forgedFile = "11111111-1111-4111-8111-aaaaaaaaaa81"
)

func TestOrgIsolation(t *testing.T) {
	dsn := os.Getenv("SITEWISE_TEST_DATABASE_URL")
	if dsn == "" {
		t.Fatal("SITEWISE_TEST_DATABASE_URL is required")
	}
	databaseName, err := databaseName(dsn)
	if err != nil {
		t.Fatal(err)
	}
	if databaseName != "sitewise_test" {
		t.Fatalf("refusing to use database %q", databaseName)
	}

	ctx := context.Background()
	st, err := store.Open(ctx, dsn)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(st.Close)
	if err := st.Migrate(ctx); err != nil {
		t.Fatal(err)
	}
	if err := st.Migrate(ctx); err != nil {
		t.Fatalf("migrate again: %v", err)
	}

	for _, orgID := range []string{orgA, orgB} {
		if err := st.DeleteOrg(ctx, orgID); err != nil {
			t.Fatal(err)
		}
	}
	t.Cleanup(func() {
		for _, orgID := range []string{orgA, orgB} {
			_ = st.DeleteOrg(context.Background(), orgID)
		}
	})

	if err := seedTenants(ctx, st); err != nil {
		t.Fatal(err)
	}

	shared := bytes32(0xab)

	t.Run("read", func(t *testing.T) {
		if _, err := st.GetDocument(ctx, orgA, docB); !errors.Is(err, store.ErrNotFound) {
			t.Fatalf("got %v", err)
		}
		got, err := st.GetDocument(ctx, orgA, docA)
		if err != nil {
			t.Fatal(err)
		}
		if got.ID != docA || got.ProjectID != projectA {
			t.Fatalf("%+v", got)
		}
	})

	t.Run("update", func(t *testing.T) {
		if err := st.UpdateDocumentStatus(ctx, orgA, docB, "filed"); err == nil {
			t.Fatal("expected error")
		}
		got, err := st.GetDocument(ctx, orgB, docB)
		if err != nil {
			t.Fatal(err)
		}
		if got.Status != "pending" {
			t.Fatalf("status = %s", got.Status)
		}
	})

	t.Run("download", func(t *testing.T) {
		if _, err := st.GetFile(ctx, orgA, fileB); !errors.Is(err, store.ErrNotFound) {
			t.Fatalf("got %v", err)
		}
		got, err := st.FindFileByHash(ctx, orgA, projectA, shared)
		if err != nil {
			t.Fatal(err)
		}
		if got.ID != fileA {
			t.Fatalf("file id = %s", got.ID)
		}
		other, err := st.FindFileByHash(ctx, orgB, projectB, shared)
		if err != nil {
			t.Fatal(err)
		}
		if other.ID != fileB {
			t.Fatalf("file id = %s", other.ID)
		}
		if _, err := st.FindFileByHash(ctx, orgA, projectB, shared); !errors.Is(err, store.ErrNotFound) {
			t.Fatalf("forged project hash lookup: %v", err)
		}
	})

	t.Run("supersede", func(t *testing.T) {
		if err := st.Supersede(ctx, orgA, docA, docB); err == nil {
			t.Fatal("expected error")
		}
		if err := st.Supersede(ctx, orgA, docB, docA); err == nil {
			t.Fatal("expected error")
		}
		got, err := st.GetDocument(ctx, orgA, docA)
		if err != nil {
			t.Fatal(err)
		}
		if got.SupersedesID != "" {
			t.Fatalf("supersedes = %s", got.SupersedesID)
		}
		other, err := st.GetDocument(ctx, orgB, docB)
		if err != nil {
			t.Fatal(err)
		}
		if other.SupersedesID != "" {
			t.Fatalf("supersedes = %s", other.SupersedesID)
		}
	})

	t.Run("stream", func(t *testing.T) {
		feed, err := st.ListStream(ctx, orgA, projectA)
		if err != nil {
			t.Fatal(err)
		}
		if len(feed) != 1 || feed[0].DocumentID != docA {
			t.Fatalf("%+v", feed)
		}
		forged, err := st.ListStream(ctx, orgA, projectB)
		if err != nil {
			t.Fatal(err)
		}
		if len(forged) != 0 {
			t.Fatalf("%+v", forged)
		}
		if _, err := st.GetPassage(ctx, orgA, passageB); !errors.Is(err, store.ErrNotFound) {
			t.Fatalf("got %v", err)
		}
		if _, err := st.GetInvite(ctx, orgA, inviteB); !errors.Is(err, store.ErrNotFound) {
			t.Fatalf("invite: %v", err)
		}
		if _, err := st.GetSession(ctx, orgA, sessionB); !errors.Is(err, store.ErrNotFound) {
			t.Fatalf("session: %v", err)
		}
	})

	t.Run("enqueue", func(t *testing.T) {
		if err := st.EnqueueJob(ctx, orgA, jobCross, docB, "intake"); err == nil {
			t.Fatal("expected error")
		}
		if err := st.EnqueueJob(ctx, orgA, jobA, docA, "intake"); err != nil {
			t.Fatal(err)
		}
		jobsA, err := st.ListJobs(ctx, orgA)
		if err != nil {
			t.Fatal(err)
		}
		if len(jobsA) != 1 || jobsA[0].ID != jobA || jobsA[0].DocumentID != docA {
			t.Fatalf("%+v", jobsA)
		}
		jobsB, err := st.ListJobs(ctx, orgB)
		if err != nil {
			t.Fatal(err)
		}
		for _, job := range jobsB {
			if job.ID == jobA || job.ID == jobCross || job.DocumentID == docA {
				t.Fatalf("%+v", jobsB)
			}
		}
	})

	t.Run("forged project", func(t *testing.T) {
		if err := st.CreateFile(ctx, orgA, store.File{
			ID:        forgedFile,
			ProjectID: projectB,
			SHA256:    bytes32(0xcd),
			ByteSize:  4,
			MediaType: "application/pdf",
		}); err == nil {
			t.Fatal("expected error")
		}
		if _, err := st.GetFile(ctx, orgA, forgedFile); !errors.Is(err, store.ErrNotFound) {
			t.Fatalf("org A: %v", err)
		}
		if _, err := st.GetFile(ctx, orgB, forgedFile); !errors.Is(err, store.ErrNotFound) {
			t.Fatalf("org B: %v", err)
		}
	})

	t.Run("cross org membership", func(t *testing.T) {
		if err := st.CreateMembership(ctx, orgA, userB, "member"); err == nil {
			t.Fatal("expected error")
		}
	})
}

func seedTenants(ctx context.Context, st *store.Store) error {
	shared := bytes32(0xab)
	type tenant struct {
		org, user, email, project, file, doc, decision, passage, invite, session string
	}
	tenants := []tenant{
		{orgA, userA, "a@example.com", projectA, fileA, docA, decisionA, passageA, inviteA, sessionA},
		{orgB, userB, "b@example.com", projectB, fileB, docB, decisionB, passageB, inviteB, sessionB},
	}
	for _, tn := range tenants {
		if err := st.CreateOrg(ctx, tn.org, "Org "+tn.org); err != nil {
			return err
		}
		if err := st.CreateUser(ctx, tn.org, tn.user, tn.email); err != nil {
			return err
		}
		if err := st.CreateMembership(ctx, tn.org, tn.user, "owner"); err != nil {
			return err
		}
		if err := st.CreateInvite(ctx, tn.org, store.Invite{
			ID:        tn.invite,
			Email:     tn.email,
			TokenHash: bytes32(0x11),
		}); err != nil {
			return err
		}
		if err := st.CreateSession(ctx, tn.org, tn.session, tn.user); err != nil {
			return err
		}
		if err := st.CreateProject(ctx, tn.org, tn.project, "Project"); err != nil {
			return err
		}
		if err := st.CreateFile(ctx, tn.org, store.File{
			ID:        tn.file,
			ProjectID: tn.project,
			SHA256:    shared,
			ByteSize:  8,
			MediaType: "application/pdf",
		}); err != nil {
			return err
		}
		if err := st.CreateDocument(ctx, tn.org, store.Document{
			ID:        tn.doc,
			ProjectID: tn.project,
			FileID:    tn.file,
			Filename:  "plan.pdf",
			Status:    "pending",
			Number:    "A-100",
			Revision:  "P1",
		}); err != nil {
			return err
		}
		if err := st.CreateDecision(ctx, tn.org, store.Decision{
			ID:         tn.decision,
			DocumentID: tn.doc,
			Field:      "kind",
			Value:      "drawing",
			Band:       "green",
			DecidedBy:  "rule",
		}); err != nil {
			return err
		}
		if err := st.AddPassage(ctx, tn.org, store.Passage{
			ID:         tn.passage,
			DocumentID: tn.doc,
			Ordinal:    1,
			Body:       "title block",
		}); err != nil {
			return err
		}
	}
	return nil
}

func databaseName(dsn string) (string, error) {
	u, err := url.Parse(dsn)
	if err != nil {
		return "", err
	}
	name := strings.TrimPrefix(u.Path, "/")
	if name == "" {
		return "", errors.New("database name missing from SITEWISE_TEST_DATABASE_URL")
	}
	return name, nil
}

func bytes32(b byte) []byte {
	out := make([]byte, 32)
	for i := range out {
		out[i] = b
	}
	return out
}

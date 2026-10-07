package store_test

import (
	"context"
	"errors"
	"testing"

	"github.com/jackc/pgx/v5/pgconn"
	"sitewise/internal/profile"
	"sitewise/internal/store"
)

func TestProfileFactDocumentBelongsToProject(t *testing.T) {
	ctx := context.Background()
	s := profileStore(t)
	const otherProject = "33333333-3333-4333-8333-222222222222"
	if err := s.CreateProject(ctx, orgA, otherProject, "Other project"); err != nil {
		t.Fatal(err)
	}
	pool := rawPool(t)
	for _, tc := range []struct{ name, project, document string }{
		{"same_org_other_project", otherProject, docA},
		{"foreign_org_document", projectA, docB},
	} {
		t.Run(tc.name, func(t *testing.T) {
			tx, err := pool.Begin(ctx)
			if err != nil {
				t.Fatal(err)
			}
			defer tx.Rollback(ctx)
			_, err = tx.Exec(ctx, `INSERT INTO profile_facts(org_id,id,project_id,document_id,question_id,value,decided_by,question_version)
 VALUES($1::uuid,gen_random_uuid(),$2::uuid,$3::uuid,'hdr.work_type','new','rule','ownership-test')`, orgA, tc.project, tc.document)
			if err == nil {
				_, err = tx.Exec(ctx, "SET CONSTRAINTS ALL IMMEDIATE")
			}
			var pgErr *pgconn.PgError
			if !errors.As(err, &pgErr) || pgErr.Code != "23503" {
				t.Fatalf("fact accepted a document outside its project: %v", err)
			}
		})
	}
	// Normal worker writes choose the source document's actual project.
	if err := s.ReplaceDocumentFacts(ctx, orgA, docA, []string{"hdr."}, profile.QuestionVersion,
		[]store.StoredFact{{QuestionID: "hdr.work_type", Value: "refurb", DecidedBy: "rule"}}); err != nil {
		t.Fatal(err)
	}
	var count int
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM profile_facts WHERE org_id=$1::uuid AND project_id=$2::uuid AND document_id=$3::uuid`, orgA, projectA, docA).Scan(&count); err != nil || count != 1 {
		t.Fatalf("valid document fact lost: %d %v", count, err)
	}
	// Keep the established document-deletion cascade, including its facts.
	if _, err := s.DeleteDocuments(ctx, orgA, projectA, []string{docA}, userA); err != nil {
		t.Fatal(err)
	}
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM profile_facts WHERE org_id=$1::uuid AND document_id=$2::uuid`, orgA, docA).Scan(&count); err != nil || count != 0 {
		t.Fatalf("deleted document retained facts: %d %v", count, err)
	}
}

func TestProfileFactPassageMetadataMatchesDocument(t *testing.T) {
	ctx := context.Background()
	s := profileStore(t)
	const otherFile = "44444444-4444-4444-8444-111111111111"
	const otherDoc = "44444444-4444-4444-8444-222222222222"
	if err := s.CreateFile(ctx, orgA, store.File{ID: otherFile, ProjectID: projectA, SHA256: bytes32(0x55), ByteSize: 8, MediaType: "text/plain"}); err != nil {
		t.Fatal(err)
	}
	if err := s.CreateDocument(ctx, orgA, store.Document{ID: otherDoc, ProjectID: projectA, FileID: otherFile, Filename: "other.txt", Status: "filed"}); err != nil {
		t.Fatal(err)
	}
	for _, source := range []struct{ doc, location string }{{docA, "Own location"}, {otherDoc, "Unrelated location"}} {
		if err := s.ReplaceSource(ctx, orgA, source.doc, store.DocumentSource{Source: []store.SourcePage{{Text: "Saved evidence"}}, Units: []store.SourceUnit{{Body: "Saved evidence", Page: 7, Location: source.location}}}); err != nil {
			t.Fatal(err)
		}
	}
	own, err := s.DocumentPassages(ctx, orgA, docA)
	if err != nil || len(own) != 1 {
		t.Fatalf("own source: %+v %v", own, err)
	}
	other, err := s.DocumentPassages(ctx, orgA, otherDoc)
	if err != nil || len(other) != 1 {
		t.Fatalf("other source: %+v %v", other, err)
	}
	// Historical locators can outlive passages. Inject a mismatched locator
	// as well: retaining a fact must not borrow another document's location.
	if err := s.ReplaceDocumentFacts(ctx, orgA, docA, []string{"det."}, profile.QuestionVersion, []store.StoredFact{
		{QuestionID: "det.own", PassageID: own[0].ID, Value: "saved", DecidedBy: "rule"},
		{QuestionID: "det.other", PassageID: other[0].ID, Value: "saved", DecidedBy: "rule"},
		{QuestionID: "det.missing", PassageID: "44444444-4444-4444-8444-333333333333", Value: "saved", DecidedBy: "rule"},
	}); err != nil {
		t.Fatal(err)
	}
	snapshot, err := s.ProfileInput(ctx, orgA, projectA)
	if err != nil || len(snapshot.Facts) != 3 {
		t.Fatalf("historical facts lost: %+v %v", snapshot.Facts, err)
	}
	for _, fact := range snapshot.Facts {
		if fact.QuestionID == "det.own" {
			if fact.Location != "Own location" || fact.Page != 7 {
				t.Fatal("matching passage metadata lost", fact)
			}
		} else if fact.Location != "" || fact.Page != 0 || fact.Section != "" || fact.StartOffset != 0 || fact.EndOffset != 0 {
			t.Fatal("fact borrowed another document's source metadata", fact)
		}
	}
}

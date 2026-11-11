package store_test

import (
	"context"
	"errors"
	"os"
	"testing"

	"sitewise/internal/store"
)

const projectA2 = "11111111-1111-4111-8111-aaaaaaaaaa02"

// The Projects page reads one org's projects, latest activity first, with the
// site address, filing count and archive flag; another org's are invisible.
func TestProjectSummaries(t *testing.T) {
	dsn := os.Getenv("SITEWISE_TEST_DATABASE_URL")
	if dsn == "" {
		t.Fatal("SITEWISE_TEST_DATABASE_URL is required")
	}
	if name, err := databaseName(dsn); err != nil || name != "sitewise_test" {
		t.Fatalf("refusing to use database %q (%v)", name, err)
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
	clean := func() {
		for _, orgID := range []string{orgA, orgB} {
			_ = st.DeleteOrg(context.Background(), orgID)
		}
	}
	clean()
	t.Cleanup(clean)
	if err := seedTenants(ctx, st); err != nil {
		t.Fatal(err)
	}
	if err := st.CreateProject(ctx, orgA, projectA2, "Second"); err != nil {
		t.Fatal(err)
	}

	ids := func(list []store.ProjectSummary) []string {
		out := make([]string, len(list))
		for i, p := range list {
			out[i] = p.ID
		}
		return out
	}

	got, err := st.ProjectSummaries(ctx, orgA)
	if err != nil {
		t.Fatal(err)
	}
	// Second was created after the first project's only filing.
	if want := []string{projectA2, projectA}; !equal(ids(got), want) {
		t.Fatalf("order %v, want %v", ids(got), want)
	}
	if got[1].Documents != 1 || got[1].LastFiled == nil || got[0].Documents != 0 || got[0].LastFiled != nil {
		t.Fatalf("counts %+v", got)
	}

	// A new filing moves its project to the top; a typed address shows.
	const file2 = "11111111-1111-4111-8111-aaaaaaaaaa12"
	if err := st.CreateFile(ctx, orgA, store.File{ID: file2, ProjectID: projectA, SHA256: bytes32(0xcd), ByteSize: 8, MediaType: "application/pdf"}); err != nil {
		t.Fatal(err)
	}
	if err := st.CreateDocument(ctx, orgA, store.Document{ID: "11111111-1111-4111-8111-aaaaaaaaaa22", ProjectID: projectA, FileID: file2, Filename: "elevations.pdf", Status: "filed", Number: "A-200", Revision: "P1"}); err != nil {
		t.Fatal(err)
	}
	site, err := st.ProjectSite(ctx, orgA, projectA)
	if err != nil {
		t.Fatal(err)
	}
	address := "14 Hale Street, Petersham"
	if _, err := st.UpdateSite(ctx, orgA, site.ID, site.Version, nil, &address, nil); err != nil {
		t.Fatal(err)
	}
	got, err = st.ProjectSummaries(ctx, orgA)
	if err != nil {
		t.Fatal(err)
	}
	if want := []string{projectA, projectA2}; !equal(ids(got), want) {
		t.Fatalf("order after filing %v, want %v", ids(got), want)
	}
	if got[0].Documents != 2 || got[0].Address != address {
		t.Fatalf("first row %+v", got[0])
	}

	// Archive keeps the project and flags it; unarchive clears the flag.
	if err := st.SetProjectArchived(ctx, orgA, projectA, true); err != nil {
		t.Fatal(err)
	}
	if got, _ = st.ProjectSummaries(ctx, orgA); !got[0].Archived || got[0].Documents != 2 {
		t.Fatalf("archived row %+v", got[0])
	}
	if err := st.SetProjectArchived(ctx, orgA, projectA, false); err != nil {
		t.Fatal(err)
	}
	if got, _ = st.ProjectSummaries(ctx, orgA); got[0].Archived {
		t.Fatalf("unarchived row %+v", got[0])
	}

	// Another org sees only its own, and cannot archive ours.
	other, err := st.ProjectSummaries(ctx, orgB)
	if err != nil {
		t.Fatal(err)
	}
	if want := []string{projectB}; !equal(ids(other), want) {
		t.Fatalf("org B sees %v", ids(other))
	}
	if err := st.SetProjectArchived(ctx, orgB, projectA, true); !errors.Is(err, store.ErrNotFound) {
		t.Fatalf("cross-org archive: %v", err)
	}
}

func equal(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

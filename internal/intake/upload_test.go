package intake_test

import (
	"bytes"
	"context"
	"crypto/sha256"
	"errors"
	"io"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"sitewise/internal/files"
	"sitewise/internal/intake"
	"sitewise/internal/store"
)

const (
	orgA = "aaaaaaa4-aaaa-4aaa-8aaa-aaaaaaaaaaa1"
	orgB = "bbbbbbb4-bbbb-4bbb-8bbb-bbbbbbbbbbb2"

	projectA  = "aaaaaaa4-aaaa-4aaa-8aaa-aaaaaaaaaa11"
	projectA2 = "aaaaaaa4-aaaa-4aaa-8aaa-aaaaaaaaaa12"
	projectB  = "bbbbbbb4-bbbb-4bbb-8bbb-bbbbbbbbbb21"

	seedFile = "aaaaaaa4-aaaa-4aaa-8aaa-aaaaaaaaaa31"
)

func TestDBFailureAfterRename(t *testing.T) {
	dir := t.TempDir()
	blobs := openBlobs(t, dir, 1<<20)
	up := intake.NewUploader(blobs, downDB{})
	_, err := up.Upload(context.Background(), orgA, projectA, intake.Upload{
		Filename: "plan.pdf",
		Body:     bytes.NewReader([]byte("durable bytes")),
	})
	if err == nil || !strings.Contains(err.Error(), "database unavailable") {
		t.Fatalf("got %v", err)
	}
	onDisk, err := blobs.List()
	if err != nil {
		t.Fatal(err)
	}
	if len(onDisk) != 1 {
		t.Fatalf("blobs = %d", len(onDisk))
	}
	rc, err := blobs.Open(onDisk[0])
	if err != nil {
		t.Fatal(err)
	}
	got, err := io.ReadAll(rc)
	rc.Close()
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != "durable bytes" {
		t.Fatalf("blob %q", got)
	}
	if temps(t, dir) != 0 {
		t.Fatal("temp file left after rename")
	}

	n, err := intake.RecoverOrphans(blobs, onDisk)
	if err != nil {
		t.Fatal(err)
	}
	if n != 0 {
		t.Fatalf("removed a live hash: %d", n)
	}
	n, err = intake.RecoverOrphans(blobs, nil)
	if err != nil {
		t.Fatal(err)
	}
	if n != 1 {
		t.Fatalf("removed %d", n)
	}
	if _, err := blobs.Open(onDisk[0]); !errors.Is(err, files.ErrNotFound) {
		t.Fatalf("got %v", err)
	}
}

func TestHashCollisionInMetadata(t *testing.T) {
	ctx := context.Background()
	st, blobs, up := openIntake(t)
	seedProject(t, st, orgA, projectA)
	body := []byte("specification text")
	sum := sha256.Sum256(body)
	if err := st.CreateFile(ctx, orgA, store.File{
		ID:        seedFile,
		ProjectID: projectA,
		SHA256:    sum[:],
		ByteSize:  int64(len(body) + 9),
		MediaType: "application/pdf",
	}); err != nil {
		t.Fatal(err)
	}

	_, err := up.Upload(ctx, orgA, projectA, intake.Upload{
		Filename: "spec.pdf",
		Body:     bytes.NewReader(body),
	})
	if !errors.Is(err, store.ErrMetadata) {
		t.Fatalf("got %v", err)
	}
	got, err := st.FindFileByHash(ctx, orgA, projectA, sum[:])
	if err != nil {
		t.Fatal(err)
	}
	if got.ID != seedFile || got.ByteSize != int64(len(body)+9) {
		t.Fatalf("metadata changed: %+v", got)
	}
	jobs, err := st.ListJobs(ctx, orgA)
	if err != nil {
		t.Fatal(err)
	}
	if len(jobs) != 0 {
		t.Fatalf("jobs %+v", jobs)
	}
	rc, err := blobs.Open(sum[:])
	if err != nil {
		t.Fatal(err)
	}
	rc.Close()
}

func TestConcurrentDuplicateUpload(t *testing.T) {
	ctx := context.Background()
	st, blobs, up := openIntake(t)
	seedProject(t, st, orgA, projectA)
	body := bytes.Repeat([]byte("drawing"), 40)

	const n = 8
	var wg sync.WaitGroup
	filings := make([]intake.Filing, n)
	errs := make([]error, n)
	start := make(chan struct{})
	for i := range n {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start
			filings[i], errs[i] = up.Upload(ctx, orgA, projectA, intake.Upload{
				Filename: "A-100.pdf",
				Body:     bytes.NewReader(body),
			})
		}()
	}
	close(start)
	wg.Wait()

	for i, err := range errs {
		if err != nil {
			t.Fatalf("upload %d: %v", i, err)
		}
	}
	first := filings[0]
	created := 0
	for i, filing := range filings {
		if filing.FileID != first.FileID || filing.DocumentID != first.DocumentID || filing.JobID != first.JobID {
			t.Fatalf("upload %d = %+v, want %+v", i, filing, first)
		}
		if filing.Created {
			created++
		}
		if filing.Status != store.StatusPending || filing.JobID == "" {
			t.Fatalf("%+v", filing)
		}
	}
	if created != 1 {
		t.Fatalf("created %d filings", created)
	}
	jobs, err := st.ListJobs(ctx, orgA)
	if err != nil {
		t.Fatal(err)
	}
	if len(jobs) != 1 || jobs[0].ID != first.JobID || jobs[0].DocumentID != first.DocumentID || jobs[0].Kind != store.JobKindIntake {
		t.Fatalf("%+v", jobs)
	}
	onDisk, err := blobs.List()
	if err != nil {
		t.Fatal(err)
	}
	if len(onDisk) != 1 {
		t.Fatalf("blobs = %d", len(onDisk))
	}
}

func TestRedropReturnsExistingFiling(t *testing.T) {
	ctx := context.Background()
	st, _, up := openIntake(t)
	seedProject(t, st, orgA, projectA)
	body := []byte("revision p1")
	first, err := up.Upload(ctx, orgA, projectA, intake.Upload{
		Filename: "A-100.pdf",
		Body:     bytes.NewReader(body),
	})
	if err != nil {
		t.Fatal(err)
	}
	second, err := up.Upload(ctx, orgA, projectA, intake.Upload{
		Filename: `..\other-name.pdf`,
		Body:     bytes.NewReader(body),
	})
	if err != nil {
		t.Fatal(err)
	}
	if second.Created || second.FileID != first.FileID || second.DocumentID != first.DocumentID || second.JobID != first.JobID {
		t.Fatalf("second %+v", second)
	}
	if second.Filename != "A-100.pdf" {
		t.Fatalf("filename %s", second.Filename)
	}
	jobs, err := st.ListJobs(ctx, orgA)
	if err != nil {
		t.Fatal(err)
	}
	if len(jobs) != 1 {
		t.Fatalf("jobs %+v", jobs)
	}
}

func TestOtherProjectAssociatesSameBlob(t *testing.T) {
	ctx := context.Background()
	st, blobs, up := openIntake(t)
	seedProject(t, st, orgA, projectA)
	if err := st.CreateProject(ctx, orgA, projectA2, "Other"); err != nil {
		t.Fatal(err)
	}
	body := []byte("shared specification")
	first, err := up.Upload(ctx, orgA, projectA, intake.Upload{
		Filename: "spec.docx",
		Body:     bytes.NewReader(body),
	})
	if err != nil {
		t.Fatal(err)
	}
	second, err := up.Upload(ctx, orgA, projectA2, intake.Upload{
		Filename: "spec.docx",
		Body:     bytes.NewReader(body),
	})
	if err != nil {
		t.Fatal(err)
	}
	if !second.Created || second.FileID == first.FileID || second.DocumentID == first.DocumentID || second.JobID == first.JobID {
		t.Fatalf("second %+v", second)
	}
	if !bytes.Equal(first.SHA256, second.SHA256) {
		t.Fatal("hash diverged")
	}
	onDisk, err := blobs.List()
	if err != nil {
		t.Fatal(err)
	}
	if len(onDisk) != 1 {
		t.Fatalf("blobs = %d", len(onDisk))
	}
	jobs, err := st.ListJobs(ctx, orgA)
	if err != nil {
		t.Fatal(err)
	}
	if len(jobs) != 2 {
		t.Fatalf("jobs %+v", jobs)
	}
}

func TestCrossOrgExistenceIsNotDisclosed(t *testing.T) {
	ctx := context.Background()
	st, blobs, up := openIntake(t)
	seedProject(t, st, orgA, projectA)
	seedProject(t, st, orgB, projectB)
	body := []byte("confidential drawing")
	filedA, err := up.Upload(ctx, orgA, projectA, intake.Upload{
		Filename: "secret.pdf",
		Body:     bytes.NewReader(body),
	})
	if err != nil {
		t.Fatal(err)
	}
	filedB, err := up.Upload(ctx, orgB, projectB, intake.Upload{
		Filename: "secret.pdf",
		Body:     bytes.NewReader(body),
	})
	if err != nil {
		t.Fatal(err)
	}
	if filedA.FileID == filedB.FileID || filedA.DocumentID == filedB.DocumentID || filedA.JobID == filedB.JobID {
		t.Fatalf("shared ids a=%+v b=%+v", filedA, filedB)
	}
	if _, err := st.GetFile(ctx, orgA, filedB.FileID); !errors.Is(err, store.ErrNotFound) {
		t.Fatalf("file: %v", err)
	}
	if _, err := st.GetDocument(ctx, orgA, filedB.DocumentID); !errors.Is(err, store.ErrNotFound) {
		t.Fatalf("document: %v", err)
	}
	_, err = up.Upload(ctx, orgB, projectA, intake.Upload{
		Filename: "secret.pdf",
		Body:     bytes.NewReader(body),
	})
	if !errors.Is(err, store.ErrNotFound) {
		t.Fatalf("got %v", err)
	}
	msg := err.Error()
	for _, secret := range []string{orgA, filedA.FileID, filedA.DocumentID, filedA.JobID} {
		if strings.Contains(msg, secret) {
			t.Fatalf("error disclosed %q: %s", secret, msg)
		}
	}
	onDisk, err := blobs.List()
	if err != nil {
		t.Fatal(err)
	}
	if len(onDisk) != 1 {
		t.Fatalf("blobs = %d", len(onDisk))
	}
	live, err := st.ListContentHashes(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if !containsHash(live, filedA.SHA256) {
		t.Fatal("recovery hashes omitted a stored blob")
	}
	n, err := intake.RecoverOrphans(blobs, live)
	if err != nil {
		t.Fatal(err)
	}
	if n != 0 {
		t.Fatalf("removed a referenced blob: %d", n)
	}
}

func TestFilenameIsNotStoragePath(t *testing.T) {
	ctx := context.Background()
	st, blobs, up := openIntake(t)
	seedProject(t, st, orgA, projectA)
	filed, err := up.Upload(ctx, orgA, projectA, intake.Upload{
		Filename: `..\..\secret.pdf`,
		Body:     bytes.NewReader([]byte("%PDF-1.7")),
	})
	if err != nil {
		t.Fatal(err)
	}
	if filed.Filename != "secret.pdf" {
		t.Fatalf("filename %s", filed.Filename)
	}
	err = filepath.WalkDir(blobRoot(t, blobs, filed.SHA256), func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if strings.Contains(strings.ToLower(d.Name()), "secret") {
			t.Fatalf("stored path uses the filename: %s", path)
		}
		return nil
	})
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		t.Fatal(err)
	}
	doc, err := st.GetDocument(ctx, orgA, filed.DocumentID)
	if err != nil {
		t.Fatal(err)
	}
	if doc.Filename != "secret.pdf" {
		t.Fatalf("document filename %s", doc.Filename)
	}
	file, err := st.GetFile(ctx, orgA, filed.FileID)
	if err != nil {
		t.Fatal(err)
	}
	if file.MediaType != "application/pdf" {
		t.Fatalf("media type %s", file.MediaType)
	}
}

func TestUnreadablePersistedNotFiled(t *testing.T) {
	ctx := context.Background()
	st, blobs, up := openIntake(t)
	seedProject(t, st, orgA, projectA)

	scanned, err := up.Upload(ctx, orgA, projectA, intake.Upload{
		Filename: "scan.pdf",
		Body:     bytes.NewReader([]byte("%PDF scanned image")),
		Reason:   "scanned",
	})
	if err != nil {
		t.Fatal(err)
	}
	if scanned.Status != store.StatusNotFiled || scanned.Reason != "scanned" || scanned.JobID != "" || !scanned.Created {
		t.Fatalf("%+v", scanned)
	}
	doc, err := st.GetDocument(ctx, orgA, scanned.DocumentID)
	if err != nil {
		t.Fatal(err)
	}
	if doc.Status != store.StatusNotFiled || doc.Reason != "scanned" || doc.Filename != "scan.pdf" {
		t.Fatalf("%+v", doc)
	}
	rc, err := blobs.Open(scanned.SHA256)
	if err != nil {
		t.Fatal(err)
	}
	rc.Close()

	again, err := up.Upload(ctx, orgA, projectA, intake.Upload{
		Filename: "scan.pdf",
		Body:     bytes.NewReader([]byte("%PDF scanned image")),
		Reason:   "scanned",
	})
	if err != nil {
		t.Fatal(err)
	}
	if again.Created || again.DocumentID != scanned.DocumentID || again.Reason != "scanned" {
		t.Fatalf("%+v", again)
	}

	empty, err := up.Upload(ctx, orgA, projectA, intake.Upload{
		Filename: "blank.xlsx",
		Body:     bytes.NewReader(nil),
	})
	if err != nil {
		t.Fatal(err)
	}
	if empty.Status != store.StatusNotFiled || empty.Reason != "empty" || empty.JobID != "" {
		t.Fatalf("%+v", empty)
	}
	unread, err := up.Upload(ctx, orgA, projectA, intake.Upload{
		Filename: "notes.docx",
		Body:     bytes.NewReader([]byte("not a zip")),
		Reason:   "unreadable",
	})
	if err != nil {
		t.Fatal(err)
	}
	if unread.Status != store.StatusNotFiled || unread.Reason != "unreadable" {
		t.Fatalf("%+v", unread)
	}
	jobs, err := st.ListJobs(ctx, orgA)
	if err != nil {
		t.Fatal(err)
	}
	if len(jobs) != 0 {
		t.Fatalf("jobs %+v", jobs)
	}
}

type downDB struct{}

func (downDB) CommitIntake(context.Context, string, store.CommitIntake) (store.CommittedIntake, error) {
	return store.CommittedIntake{}, errors.New("database unavailable")
}

func openIntake(t *testing.T) (*store.Store, *files.Store, *intake.Uploader) {
	t.Helper()
	st := storeFrom(t)
	blobs := openBlobs(t, t.TempDir(), 1<<20)
	return st, blobs, intake.NewUploader(blobs, st)
}

func storeFrom(t *testing.T) *store.Store {
	t.Helper()
	dsn := os.Getenv("SITEWISE_TEST_DATABASE_URL")
	if dsn == "" {
		t.Fatal("SITEWISE_TEST_DATABASE_URL is required")
	}
	name, err := databaseName(dsn)
	if err != nil {
		t.Fatal(err)
	}
	if name != "sitewise_test" {
		t.Fatalf("refusing to use database %q", name)
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
	return st
}

func seedProject(t *testing.T, st *store.Store, orgID, projectID string) {
	t.Helper()
	ctx := context.Background()
	if err := st.DeleteOrg(ctx, orgID); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_ = st.DeleteOrg(context.Background(), orgID)
	})
	if err := st.CreateOrg(ctx, orgID, "Org"); err != nil {
		t.Fatal(err)
	}
	if err := st.CreateProject(ctx, orgID, projectID, "Project"); err != nil {
		t.Fatal(err)
	}
}

func openBlobs(t *testing.T, dir string, max int64) *files.Store {
	t.Helper()
	st, err := files.Open(dir, max)
	if err != nil {
		t.Fatal(err)
	}
	return st
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

func containsHash(hashes [][]byte, want []byte) bool {
	for _, hash := range hashes {
		if bytes.Equal(hash, want) {
			return true
		}
	}
	return false
}

func temps(t *testing.T, dir string) int {
	t.Helper()
	n := 0
	tmp := filepath.Join(dir, "tmp")
	entries, err := os.ReadDir(tmp)
	if errors.Is(err, os.ErrNotExist) {
		return 0
	}
	if err != nil {
		t.Fatal(err)
	}
	for _, entry := range entries {
		if !entry.IsDir() {
			n++
		}
	}
	return n
}

func blobRoot(t *testing.T, blobs *files.Store, sha []byte) string {
	t.Helper()
	path, err := blobs.Path(sha)
	if err != nil {
		t.Fatal(err)
	}
	return filepath.Dir(filepath.Dir(path))
}

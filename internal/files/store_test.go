package files_test

import (
	"bytes"
	"context"
	"crypto/sha256"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"sitewise/internal/files"
)

func TestPutUsesContentHash(t *testing.T) {
	dir := t.TempDir()
	st := openFiles(t, dir, 1024)
	body := []byte("title block A-100")
	blob, err := st.Put(context.Background(), bytes.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	sum := sha256.Sum256(body)
	if !bytes.Equal(blob.SHA256, sum[:]) || blob.Size != int64(len(body)) {
		t.Fatalf("%+v", blob)
	}
	path, err := st.Path(blob.SHA256)
	if err != nil {
		t.Fatal(err)
	}
	if filepath.Base(path) != encode(blob.SHA256) {
		t.Fatalf("path %s", path)
	}
	if strings.Contains(path, "secret") || strings.Contains(path, "plan.pdf") {
		t.Fatalf("path uses a filename: %s", path)
	}
	rc, err := st.Open(blob.SHA256)
	if err != nil {
		t.Fatal(err)
	}
	defer rc.Close()
	got, err := io.ReadAll(rc)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, body) {
		t.Fatalf("read %q", got)
	}
}

func TestInterruptedStream(t *testing.T) {
	dir := t.TempDir()
	st := openFiles(t, dir, 1024)
	_, err := st.Put(context.Background(), &errAfter{n: 4, err: io.ErrClosedPipe, payload: []byte("partial-upload-bytes")})
	if !errors.Is(err, io.ErrClosedPipe) {
		t.Fatalf("got %v", err)
	}
	if filesLeft(t, dir) != 0 {
		t.Fatal("interrupted upload left a file behind")
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err = st.Put(ctx, bytes.NewReader([]byte("late")))
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("got %v", err)
	}
	if filesLeft(t, dir) != 0 {
		t.Fatal("canceled upload left a file behind")
	}
}

func TestHashCollisionInMetadata(t *testing.T) {
	dir := t.TempDir()
	st := openFiles(t, dir, 1024)
	body := []byte("the real drawing bytes")
	sum := sha256.Sum256(body)
	path, err := st.Path(sum[:])
	if err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	planted := []byte("metadata claims this hash but the bytes differ")
	if err := os.WriteFile(path, planted, 0o644); err != nil {
		t.Fatal(err)
	}
	_, err = st.Put(context.Background(), bytes.NewReader(body))
	if !errors.Is(err, files.ErrHashMismatch) {
		t.Fatalf("got %v", err)
	}
	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, planted) {
		t.Fatalf("existing blob was replaced: %q", got)
	}
	if _, err := st.Open(sum[:]); !errors.Is(err, files.ErrHashMismatch) {
		t.Fatalf("open got %v", err)
	}
}

func TestExistingHashIsImmutable(t *testing.T) {
	dir := t.TempDir()
	st := openFiles(t, dir, 1024)
	body := []byte("same bytes twice")
	first, err := st.Put(context.Background(), bytes.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	path, err := st.Path(first.SHA256)
	if err != nil {
		t.Fatal(err)
	}
	before, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	time.Sleep(20 * time.Millisecond)
	second, err := st.Put(context.Background(), bytes.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(first.SHA256, second.SHA256) || second.Size != first.Size {
		t.Fatalf("second %+v", second)
	}
	after, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if !before.ModTime().Equal(after.ModTime()) || before.Size() != after.Size() {
		t.Fatal("matching hash was rewritten")
	}
	listed, err := st.List()
	if err != nil {
		t.Fatal(err)
	}
	if len(listed) != 1 {
		t.Fatalf("blobs = %d", len(listed))
	}
}

func TestBoundedStream(t *testing.T) {
	dir := t.TempDir()
	st := openFiles(t, dir, 4)
	if _, err := st.Put(context.Background(), bytes.NewReader([]byte("1234"))); err != nil {
		t.Fatal(err)
	}
	_, err := st.Put(context.Background(), bytes.NewReader([]byte("12345")))
	if !errors.Is(err, files.ErrTooLarge) {
		t.Fatalf("got %v", err)
	}
	listed, err := st.List()
	if err != nil {
		t.Fatal(err)
	}
	if len(listed) != 1 {
		t.Fatalf("blobs = %d", len(listed))
	}
}

func TestConcurrentPutSameBytes(t *testing.T) {
	dir := t.TempDir()
	st := openFiles(t, dir, 1024)
	body := bytes.Repeat([]byte("dup"), 30)
	var wg sync.WaitGroup
	errCh := make(chan error, 8)
	for range 8 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, err := st.Put(context.Background(), bytes.NewReader(body))
			errCh <- err
		}()
	}
	wg.Wait()
	close(errCh)
	for err := range errCh {
		if err != nil {
			t.Fatal(err)
		}
	}
	listed, err := st.List()
	if err != nil {
		t.Fatal(err)
	}
	if len(listed) != 1 {
		t.Fatalf("blobs = %d", len(listed))
	}
}

func TestRemoveUnreferenced(t *testing.T) {
	dir := t.TempDir()
	st := openFiles(t, dir, 1024)
	keep, err := st.Put(context.Background(), bytes.NewReader([]byte("keep")))
	if err != nil {
		t.Fatal(err)
	}
	drop, err := st.Put(context.Background(), bytes.NewReader([]byte("drop")))
	if err != nil {
		t.Fatal(err)
	}
	n, err := st.RemoveUnreferenced([][]byte{keep.SHA256})
	if err != nil {
		t.Fatal(err)
	}
	if n != 1 {
		t.Fatalf("removed %d", n)
	}
	rc, err := st.Open(keep.SHA256)
	if err != nil {
		t.Fatal(err)
	}
	rc.Close()
	if _, err := st.Open(drop.SHA256); !errors.Is(err, files.ErrNotFound) {
		t.Fatalf("got %v", err)
	}
}

func openFiles(t *testing.T, dir string, max int64) *files.Store {
	t.Helper()
	st, err := files.Open(dir, max)
	if err != nil {
		t.Fatal(err)
	}
	return st
}

func filesLeft(t *testing.T, dir string) int {
	t.Helper()
	n := 0
	err := filepath.WalkDir(dir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !d.IsDir() {
			n++
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	return n
}

func encode(sum []byte) string {
	const hexdigits = "0123456789abcdef"
	out := make([]byte, len(sum)*2)
	for i, b := range sum {
		out[i*2] = hexdigits[b>>4]
		out[i*2+1] = hexdigits[b&0x0f]
	}
	return string(out)
}

type errAfter struct {
	n       int
	payload []byte
	err     error
	read    int
}

func (e *errAfter) Read(p []byte) (int, error) {
	if e.read >= e.n {
		return 0, e.err
	}
	n := min(len(p), e.n-e.read, len(e.payload)-e.read)
	copy(p, e.payload[e.read:e.read+n])
	e.read += n
	return n, nil
}

package files

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"syscall"
	"time"
)

// ErrHashMismatch means bytes at a content-addressed path do not hash to that path.
var ErrHashMismatch = errors.New("blob content does not match its hash")

// ErrTooLarge means the upload exceeded the configured size limit.
var ErrTooLarge = errors.New("upload exceeds size limit")

// ErrNotFound means no blob is stored for the hash.
var ErrNotFound = errors.New("blob not found")

// Blob is one immutable object. Its path is derived from SHA256.
type Blob struct {
	SHA256 []byte
	Size   int64
}

// Store writes content-addressed blobs under a single directory.
// Temporary files live inside that directory so the final rename stays on one filesystem.
type Store struct {
	root    string
	tmp     string
	max     int64
	publish [256]sync.Mutex
	// reused records when Put last returned an existing blob, by hex hash.
	reused sync.Map
}

// Open creates the blob directory. maxBytes is the maximum accepted upload.
func Open(dir string, maxBytes int64) (*Store, error) {
	if strings.TrimSpace(dir) == "" {
		return nil, errors.New("file directory is required")
	}
	if maxBytes <= 0 {
		return nil, errors.New("upload size limit must be positive")
	}
	root, err := filepath.Abs(dir)
	if err != nil {
		return nil, err
	}
	tmp := filepath.Join(root, "tmp")
	if err := os.MkdirAll(tmp, 0o755); err != nil {
		return nil, err
	}
	return &Store{root: root, tmp: tmp, max: maxBytes}, nil
}

// Path returns the blob location for a hash. The uploaded filename is never an input.
func (s *Store) Path(sum []byte) (string, error) {
	if len(sum) != sha256.Size {
		return "", errors.New("sha256 must be 32 bytes")
	}
	name := hex.EncodeToString(sum)
	path := filepath.Join(s.root, name[:2], name)
	rel, err := filepath.Rel(s.root, path)
	if err != nil || strings.HasPrefix(rel, "..") {
		return "", errors.New("blob path escapes storage root")
	}
	return path, nil
}

// Put streams r to a temporary file, hashes it, and renames it into place.
// An existing blob whose bytes match the hash is left unchanged.
func (s *Store) Put(ctx context.Context, r io.Reader) (Blob, error) {
	if err := ctx.Err(); err != nil {
		return Blob{}, err
	}
	tmp, err := os.CreateTemp(s.tmp, "up-")
	if err != nil {
		return Blob{}, err
	}
	tmpName := tmp.Name()
	keep := false
	defer func() {
		if !keep {
			_ = os.Remove(tmpName)
		}
	}()

	h := sha256.New()
	n, err := io.Copy(io.MultiWriter(tmp, h), &bounded{ctx: ctx, r: r, max: s.max})
	if err != nil {
		_ = tmp.Close()
		return Blob{}, err
	}
	if err := tmp.Sync(); err != nil {
		_ = tmp.Close()
		return Blob{}, err
	}
	if err := tmp.Close(); err != nil {
		return Blob{}, err
	}
	sum := h.Sum(nil)
	final, err := s.Path(sum)
	if err != nil {
		return Blob{}, err
	}
	if err := os.MkdirAll(filepath.Dir(final), 0o755); err != nil {
		return Blob{}, err
	}
	// Concurrent duplicates must not both observe a missing path and then
	// replace it while another upload inspects it. Windows rejects that open
	// during replacement. Serialize publication by hash prefix; streaming and
	// hashing stay parallel, and an existing immutable blob is never replaced.
	lock := &s.publish[sum[0]]
	lock.Lock()
	defer lock.Unlock()
	switch err := inspect(final, sum); {
	case err == nil:
		// Remember the re-use: Remove keeps a blob re-used recently, so a
		// deletion cannot take bytes an upload is about to commit a reference
		// to. The file itself is never touched.
		s.reused.Store(hex.EncodeToString(sum), time.Now())
		return Blob{SHA256: bytes.Clone(sum), Size: n}, nil
	case errors.Is(err, os.ErrNotExist):
	default:
		return Blob{}, err
	}
	if err := os.Rename(tmpName, final); err != nil {
		if vErr := inspect(final, sum); vErr == nil {
			return Blob{SHA256: bytes.Clone(sum), Size: n}, nil
		} else if !errors.Is(vErr, os.ErrNotExist) {
			return Blob{}, vErr
		}
		return Blob{}, err
	}
	keep = true
	if err := syncDir(filepath.Dir(final)); err != nil {
		return Blob{}, err
	}
	if err := syncDir(s.root); err != nil {
		return Blob{}, err
	}
	return Blob{SHA256: bytes.Clone(sum), Size: n}, nil
}

// Open returns the blob only when its bytes hash to sum.
func (s *Store) Open(sum []byte) (io.ReadCloser, error) {
	path, err := s.Path(sum)
	if err != nil {
		return nil, err
	}
	if err := inspect(path, sum); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return os.Open(path)
}

// List returns hashes stored under the blob root.
func (s *Store) List() ([][]byte, error) {
	var out [][]byte
	err := filepath.WalkDir(s.root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			if path != s.root && d.Name() == "tmp" {
				return filepath.SkipDir
			}
			return nil
		}
		name := d.Name()
		if len(name) != sha256.Size*2 {
			return nil
		}
		sum, err := hex.DecodeString(name)
		if err != nil || len(sum) != sha256.Size {
			return nil
		}
		if filepath.Base(filepath.Dir(path)) != name[:2] {
			return nil
		}
		out = append(out, sum)
		return nil
	})
	if err != nil {
		return nil, err
	}
	return out, nil
}

// RemoveUnreferenced deletes blobs whose hashes are not in live.
// A hash still referenced by any org must be included; this store has no tenant index.
func (s *Store) RemoveUnreferenced(live [][]byte) (int, error) {
	keep := make(map[string]struct{}, len(live))
	for _, sum := range live {
		if len(sum) != sha256.Size {
			return 0, errors.New("sha256 must be 32 bytes")
		}
		keep[hex.EncodeToString(sum)] = struct{}{}
	}
	have, err := s.List()
	if err != nil {
		return 0, err
	}
	removed := 0
	for _, sum := range have {
		if _, ok := keep[hex.EncodeToString(sum)]; ok {
			continue
		}
		path, err := s.Path(sum)
		if err != nil {
			return removed, err
		}
		if err := os.Remove(path); err != nil && !errors.Is(err, os.ErrNotExist) {
			return removed, err
		}
		removed++
	}
	return removed, nil
}

// Remove deletes the blob for sum when it is at least minAge old and
// referenced reports no reference. Both checks run under the publish lock that
// Put holds, and Put records when it re-uses an existing blob, so bytes an
// upload is about to reference are kept. A missing blob is not an error. The blob directory is
// shared by every org, so referenced must look across orgs.
func (s *Store) Remove(sum []byte, minAge time.Duration, referenced func() (bool, error)) (bool, error) {
	path, err := s.Path(sum)
	if err != nil {
		return false, err
	}
	lock := &s.publish[sum[0]]
	lock.Lock()
	defer lock.Unlock()
	info, err := os.Lstat(path)
	if errors.Is(err, os.ErrNotExist) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	if time.Since(info.ModTime()) < minAge {
		return false, nil
	}
	key := hex.EncodeToString(sum)
	if at, ok := s.reused.Load(key); ok && time.Since(at.(time.Time)) < minAge {
		return false, nil
	}
	used, err := referenced()
	if err != nil || used {
		return false, err
	}
	if err := os.Remove(path); err != nil && !errors.Is(err, os.ErrNotExist) {
		return false, err
	}
	s.reused.Delete(key)
	return true, nil
}

// inspect reports whether path is the immutable blob for want.
// A hash mismatch leaves the file in place.
func inspect(path string, want []byte) error {
	info, err := os.Lstat(path)
	if err != nil {
		return err
	}
	if info.Mode()&os.ModeSymlink != 0 {
		return fmt.Errorf("blob path is a symlink")
	}
	if !info.Mode().IsRegular() {
		return fmt.Errorf("blob path is not a file")
	}
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()
	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return err
	}
	if !bytes.Equal(h.Sum(nil), want) {
		return ErrHashMismatch
	}
	return nil
}

func syncDir(dir string) error {
	f, err := os.Open(dir)
	if err != nil {
		return err
	}
	defer f.Close()
	err = f.Sync()
	if err == nil || dirSyncUnsupported(err) {
		return nil
	}
	return err
}

func dirSyncUnsupported(err error) bool {
	if errors.Is(err, syscall.ENOTSUP) || errors.Is(err, syscall.EINVAL) || errors.Is(err, syscall.EPERM) {
		return true
	}
	var errno syscall.Errno
	if errors.As(err, &errno) {
		// Windows refuses FlushFileBuffers on a directory handle.
		return errno == 1 || errno == 5
	}
	return false
}

type bounded struct {
	ctx context.Context
	r   io.Reader
	max int64
	n   int64
}

func (b *bounded) Read(p []byte) (int, error) {
	if err := b.ctx.Err(); err != nil {
		return 0, err
	}
	remain := b.max - b.n + 1
	if remain <= 0 {
		return 0, ErrTooLarge
	}
	if int64(len(p)) > remain {
		p = p[:remain]
	}
	n, err := b.r.Read(p)
	b.n += int64(n)
	if b.n > b.max {
		return n, ErrTooLarge
	}
	return n, err
}

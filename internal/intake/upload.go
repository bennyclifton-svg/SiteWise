package intake

import (
	"context"
	"errors"
	"fmt"
	"io"
	"path/filepath"
	"strings"

	"sitewise/internal/files"
	"sitewise/internal/store"
)

// ErrFilename means the uploaded name is empty or too long. It is the
// caller's to fix, not a storage failure.
var ErrFilename = errors.New("invalid filename")

// Upload is one dropped file. Reason is empty when the file should be filed.
// A non-empty reason stores the bytes and records status not_filed.
type Upload struct {
	Filename string
	Body     io.Reader
	Reason   string
}

// Filing is the durable project association for an upload.
type Filing struct {
	FileID     string
	DocumentID string
	JobID      string
	Status     string
	Reason     string
	Filename   string
	SHA256     []byte
	Size       int64
	Created    bool
}

// Committer persists the association and intake job. Implementations must
// commit those rows in one transaction.
type Committer interface {
	CommitIntake(context.Context, string, store.CommitIntake) (store.CommittedIntake, error)
}

// Uploader stores bytes by hash, then asks Committer to file them.
type Uploader struct {
	blobs *files.Store
	db    Committer
}

// NewUploader binds content-addressed storage to the filing transaction.
func NewUploader(blobs *files.Store, db Committer) *Uploader {
	return &Uploader{blobs: blobs, db: db}
}

// Upload writes the body, then commits the org's project association.
// A database failure after the rename leaves the blob for RecoverOrphans.
func (u *Uploader) Upload(ctx context.Context, orgID, projectID string, up Upload) (Filing, error) {
	if u.blobs == nil || u.db == nil {
		return Filing{}, errors.New("upload storage is not configured")
	}
	name, err := cleanFilename(up.Filename)
	if err != nil {
		return Filing{}, err
	}
	reason, err := cleanReason(up.Reason)
	if err != nil {
		return Filing{}, err
	}
	blob, err := u.blobs.Put(ctx, up.Body)
	if err != nil {
		return Filing{}, err
	}
	if blob.Size == 0 && reason == "" {
		reason = "empty"
	}
	committed, err := u.db.CommitIntake(ctx, orgID, store.CommitIntake{
		ProjectID: projectID,
		SHA256:    blob.SHA256,
		ByteSize:  blob.Size,
		MediaType: mediaType(name),
		Filename:  name,
		Reason:    reason,
	})
	if err != nil {
		return Filing{}, err
	}
	return Filing{
		FileID:     committed.FileID,
		DocumentID: committed.DocumentID,
		JobID:      committed.JobID,
		Status:     committed.Status,
		Reason:     committed.Reason,
		Filename:   committed.Filename,
		SHA256:     committed.SHA256,
		Size:       committed.ByteSize,
		Created:    committed.Created,
	}, nil
}

// RecoverOrphans removes blobs that no live association references.
// live must include every org's hashes: the blob directory is shared.
func RecoverOrphans(blobs *files.Store, live [][]byte) (int, error) {
	if blobs == nil {
		return 0, errors.New("file store is required")
	}
	return blobs.RemoveUnreferenced(live)
}

func cleanFilename(name string) (string, error) {
	name = filepath.Base(strings.TrimSpace(name))
	name = strings.TrimSpace(name)
	if name == "" || name == "." || name == ".." {
		return "", fmt.Errorf("%w: filename is required", ErrFilename)
	}
	if len(name) > 255 {
		return "", fmt.Errorf("%w: filename is too long", ErrFilename)
	}
	return name, nil
}

func cleanReason(reason string) (string, error) {
	reason = strings.TrimSpace(reason)
	if reason == "" {
		return "", nil
	}
	if len(reason) > 64 {
		return "", errors.New("invalid not_filed reason")
	}
	for _, r := range reason {
		if r < 0x20 || r == 0x7f {
			return "", errors.New("invalid not_filed reason")
		}
	}
	return reason, nil
}

func mediaType(name string) string {
	switch strings.ToLower(filepath.Ext(name)) {
	case ".pdf":
		return "application/pdf"
	case ".docx":
		return "application/vnd.openxmlformats-officedocument.wordprocessingml.document"
	case ".xlsx":
		return "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet"
	default:
		return "application/octet-stream"
	}
}

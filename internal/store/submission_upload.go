package store

import (
	"context"
	"errors"
	"github.com/jackc/pgx/v5"
	"sitewise/internal/proposals"
	"time"
)

// CheckSubmissionUpload rejects an inaccessible package before storing bytes.
// An existing column can receive its missing source without creating a duplicate firm.
func (s *Store) CheckSubmissionUpload(ctx context.Context, org, project, pkg, sub string) error {
	if err := packageInProject(ctx, s.pool, org, project, pkg); err != nil {
		return err
	}
	if sub == "" {
		return nil
	}
	v, err := readSubmission(ctx, s.pool, org, sub)
	if err != nil {
		return err
	}
	return uploadTarget(v, project, pkg)
}

func uploadTarget(v Submission, project, pkg string) error {
	if v.ProjectID != project || v.PackageID != pkg {
		return ErrNotFound
	}
	if v.DocumentID != "" || (v.Status != SubmissionReceived && v.Status != SubmissionNeedsReview) {
		return ErrConflict
	}
	return nil
}

// AttachSubmissionDocument retains the column's user corrections, and starts a
// fresh machine reading. The source has already been filed through normal intake.
func (s *Store) AttachSubmissionDocument(ctx context.Context, org, project, pkg, sub, doc string, now time.Time) (string, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return "", err
	}
	defer tx.Rollback(ctx)
	var locked string
	if err = tx.QueryRow(ctx, `SELECT id::text FROM submissions WHERE org_id=$1::uuid AND id=$2::uuid FOR UPDATE`, org, sub).Scan(&locked); errors.Is(err, pgx.ErrNoRows) {
		return "", ErrNotFound
	} else if err != nil {
		return "", err
	}
	v, err := readSubmission(ctx, tx, org, sub)
	if err != nil {
		return "", err
	}
	if err = uploadTarget(v, project, pkg); err != nil {
		return "", err
	}
	var filename string
	if err = tx.QueryRow(ctx, `SELECT filename FROM documents WHERE org_id=$1::uuid AND project_id=$2::uuid AND id=$3::uuid`, org, project, doc).Scan(&filename); errors.Is(err, pgx.ErrNoRows) {
		return "", ErrNotFound
	} else if err != nil {
		return "", err
	}
	var exists bool
	if err = tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM submissions WHERE org_id=$1::uuid AND document_id=$2::uuid)`, org, doc).Scan(&exists); err != nil {
		return "", err
	}
	if exists {
		return "", ErrConflict
	}
	if _, err = tx.Exec(ctx, `UPDATE submissions SET document_id=$3::uuid,filename=$4,status='reading',flags='[]',retry_after=NULL,read_attempts=0,updated_at=$5,version=version+1 WHERE org_id=$1::uuid AND id=$2::uuid`, org, sub, doc, filename, now); err != nil {
		return "", err
	}
	if err = upsertSubmissionDecision(ctx, tx, org, project, sub, proposals.Decision{Field: proposals.FieldPackage, Value: pkg, Band: proposals.BandGreen, DecidedBy: proposals.ByUser}, true); err != nil {
		return "", err
	}
	if v.FirmID != "" {
		f, err := readFirm(ctx, tx, org, v.FirmID)
		if err != nil {
			return "", err
		}
		if err = upsertSubmissionDecision(ctx, tx, org, project, sub, proposals.Decision{Field: proposals.FieldFirmName, Value: f.Name, Band: proposals.BandGreen, DecidedBy: proposals.ByUser}, true); err != nil {
			return "", err
		}
	}
	if err = procurementEvent(ctx, tx, s, org, project, sub, "reading"); err != nil {
		return "", err
	}
	return sub, tx.Commit(ctx)
}

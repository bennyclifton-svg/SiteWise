package store

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/jackc/pgx/v5"
)

// DeleteResult is what a deletion removed. Unreferenced holds the content
// hashes of file rows that went with it; their blobs may be removed once no
// org references them (BlobReferenced).
type DeleteResult struct {
	Deleted      []string
	Unreferenced [][]byte
}

// DeleteDocuments permanently deletes documents in one project, all or
// nothing. A drawing set takes its sheets with it; a sheet goes alone.
// Passages, decisions, jobs, readings, facts, sources and supersessions
// cascade, so a revision the deleted document superseded becomes current
// again. Each document has its own file row (documents_file_uq), which goes
// with it; the bytes may still be shared with other projects or orgs. It
// runs under the project's profile lock so a rebuild never reads half a
// deletion, and records one "deleted" event per document.
func (s *Store) DeleteDocuments(ctx context.Context, orgID, projectID string, ids []string, userID string) (DeleteResult, error) {
	var res DeleteResult
	if len(ids) == 0 {
		return res, ErrNotFound
	}
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return res, err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if _, err := tx.Exec(ctx, `SELECT pg_advisory_xact_lock(hashtextextended($1 || '/' || $2, 0))`, orgID, projectID); err != nil {
		return res, err
	}
	var found int
	if err := tx.QueryRow(ctx, `
SELECT count(*) FROM documents
WHERE org_id = $1::uuid AND project_id = $2::uuid AND id::text = ANY($3::text[])`, orgID, projectID, ids).Scan(&found); err != nil {
		return res, err
	}
	if found != len(unique(ids)) {
		return res, ErrNotFound
	}
	rows, err := tx.Query(ctx, `
WITH chosen AS (
  SELECT id FROM documents WHERE org_id = $1::uuid AND project_id = $2::uuid AND id::text = ANY($3::text[])
  UNION
  SELECT ds.document_id FROM drawing_sheets ds
  WHERE ds.org_id = $1::uuid AND ds.source_id::text = ANY($3::text[])
)
DELETE FROM documents d
WHERE d.org_id = $1::uuid AND d.project_id = $2::uuid AND d.id IN (SELECT id FROM chosen)
RETURNING d.id::text, d.file_id::text`, orgID, projectID, ids)
	if err != nil {
		return res, err
	}
	var fileIDs []string
	for rows.Next() {
		var id, file string
		if err := rows.Scan(&id, &file); err != nil {
			rows.Close()
			return res, err
		}
		res.Deleted = append(res.Deleted, id)
		fileIDs = append(fileIDs, file)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return res, err
	}
	frows, err := tx.Query(ctx, `
DELETE FROM files f
WHERE f.org_id = $1::uuid AND f.project_id = $2::uuid AND f.id::text = ANY($3::text[])
  AND NOT EXISTS (SELECT 1 FROM documents d WHERE d.org_id = f.org_id AND d.file_id = f.id)
  AND NOT EXISTS (SELECT 1 FROM invoices i WHERE i.org_id = f.org_id AND i.file_id = f.id)
RETURNING f.sha256`, orgID, projectID, fileIDs)
	if err != nil {
		return res, err
	}
	for frows.Next() {
		var sum []byte
		if err := frows.Scan(&sum); err != nil {
			frows.Close()
			return res, err
		}
		res.Unreferenced = append(res.Unreferenced, sum)
	}
	frows.Close()
	if err := frows.Err(); err != nil {
		return res, err
	}
	q := s.q.WithTx(tx)
	for _, id := range res.Deleted {
		payload, _ := json.Marshal(map[string]string{"document_id": id, "project_id": projectID, "user_id": userID})
		if _, err := appendEvent(ctx, q, orgID, "deleted", id, string(payload)); err != nil {
			return res, err
		}
	}
	return res, s.finishProfileWrite(ctx, tx, orgID, projectID)
}

// DeleteProject permanently removes one project and everything that belongs
// to it: documents, files, profile, cost plan (including a frozen baseline),
// procurement, delivery, specifications (including an issue and its history),
// reports (including an issued version) and the project's events. Version 1
// gives each project its own site, so the site and its parts go too when no
// project still uses them. Bytes shared with another project stay. Frozen
// and issued rows stay immutable outside this transaction.
func (s *Store) DeleteProject(ctx context.Context, orgID, projectID string) (DeleteResult, error) {
	var res DeleteResult
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return res, err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if _, err := tx.Exec(ctx, `SELECT pg_advisory_xact_lock(hashtextextended($1 || '/' || $2, 0))`, orgID, projectID); err != nil {
		return res, err
	}
	var siteID string
	err = tx.QueryRow(ctx, `
SELECT site_id::text FROM projects WHERE org_id = $1::uuid AND id = $2::uuid FOR UPDATE`, orgID, projectID).Scan(&siteID)
	if errors.Is(err, pgx.ErrNoRows) {
		return res, ErrNotFound
	}
	if err != nil {
		return res, err
	}
	// The immutability triggers allow this transaction to remove frozen and
	// issued rows. The setting is local, so it ends with the transaction.
	if _, err := tx.Exec(ctx, `SELECT set_config('sitewise.deleting_project', '1', true)`); err != nil {
		return res, err
	}
	rows, err := tx.Query(ctx, `SELECT sha256 FROM files WHERE org_id = $1::uuid AND project_id = $2::uuid`, orgID, projectID)
	if err != nil {
		return res, err
	}
	for rows.Next() {
		var sum []byte
		if err := rows.Scan(&sum); err != nil {
			rows.Close()
			return res, err
		}
		res.Unreferenced = append(res.Unreferenced, sum)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return res, err
	}
	// Cost rows reference the project without ON DELETE CASCADE, and a frozen
	// baseline refuses an ordinary delete. They go before the project row.
	for _, q := range []string{
		`DELETE FROM scope_cost_links WHERE org_id = $1::uuid AND project_id = $2::uuid`,
		`DELETE FROM cost_values WHERE org_id = $1::uuid AND project_id = $2::uuid`,
		`DELETE FROM cost_item_revisions WHERE org_id = $1::uuid AND project_id = $2::uuid`,
		`DELETE FROM cost_items WHERE org_id = $1::uuid AND project_id = $2::uuid`,
		`DELETE FROM cost_plans WHERE org_id = $1::uuid AND project_id = $2::uuid`,
		`DELETE FROM cost_plan_versions WHERE org_id = $1::uuid AND project_id = $2::uuid`,
	} {
		if _, err := tx.Exec(ctx, q, orgID, projectID); err != nil {
			return res, err
		}
	}
	if _, err := tx.Exec(ctx, `
DELETE FROM events e
WHERE e.org_id = $1::uuid
  AND (
    e.document_id IN (SELECT d.id FROM documents d WHERE d.org_id = e.org_id AND d.project_id = $2::uuid)
    OR (left(e.payload, 1) = '{' AND e.payload::jsonb->>'project_id' = $2::text)
  )`, orgID, projectID); err != nil {
		return res, err
	}
	tag, err := tx.Exec(ctx, `DELETE FROM projects WHERE org_id = $1::uuid AND id = $2::uuid`, orgID, projectID)
	if err != nil {
		return res, err
	}
	if tag.RowsAffected() != 1 {
		return res, ErrNotFound
	}
	if _, err := tx.Exec(ctx, `
DELETE FROM sites s
WHERE s.org_id = $1::uuid AND s.id = $2::uuid
  AND NOT EXISTS (SELECT 1 FROM projects p WHERE p.org_id = s.org_id AND p.site_id = s.id)`, orgID, siteID); err != nil {
		return res, err
	}
	res.Deleted = []string{projectID}
	return res, tx.Commit(ctx)
}

// BlobReferenced reports whether any file row in any org has this content
// hash. The blob directory is shared across orgs, so a blob may be removed
// only when this is false. It reveals nothing to a caller: the answer only
// decides whether bytes stay on disk.
func (s *Store) BlobReferenced(ctx context.Context, sum []byte) (bool, error) {
	var used bool
	err := s.pool.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM files WHERE sha256 = $1)`, sum).Scan(&used)
	return used, err
}

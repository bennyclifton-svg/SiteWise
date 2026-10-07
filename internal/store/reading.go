package store

import (
	"context"
	"errors"
	"fmt"
)

// ErrBadReadSetting rejects a reading setting other than auto, read or skip.
var ErrBadReadSetting = errors.New("reading setting must be auto, read or skip")

// readableSQL is the predicate for "the profile reads document d". param is
// the placeholder holding the automatically read kinds; a NULL list applies
// no kind filter, so only an explicit skip is excluded (tools and tests that
// predate the setting). The kind is the document's filed kind decision.
func readableSQL(param string) string {
	return fmt.Sprintf(`(d.profile_read = 'read' OR (d.profile_read = 'auto' AND (%[1]s::text[] IS NULL OR
  COALESCE((SELECT k.value FROM decisions k WHERE k.org_id = d.org_id AND k.document_id = d.id AND k.field = 'kind'), '') = ANY(%[1]s::text[]))))`, param)
}

// SetProfileReading records the user's reading setting for documents in one
// project, all or nothing. A drawing set's sheets follow their set. Reading
// that has not started for a document the profile no longer reads is
// withdrawn; finished readings stay, so turning a document back on costs no
// Jev call. It returns how many documents changed.
func (s *Store) SetProfileReading(ctx context.Context, orgID, projectID string, ids []string, setting string, readKinds []string) (int, error) {
	if setting != "auto" && setting != "read" && setting != "skip" {
		return 0, ErrBadReadSetting
	}
	if len(ids) == 0 {
		return 0, ErrNotFound
	}
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return 0, err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if err := lockProject(ctx, tx, orgID, projectID); err != nil {
		return 0, err
	}
	var found int
	if err := tx.QueryRow(ctx, `
SELECT count(*) FROM documents
WHERE org_id = $1::uuid AND project_id = $2::uuid AND id::text = ANY($3::text[])`, orgID, projectID, ids).Scan(&found); err != nil {
		return 0, err
	}
	if found != len(unique(ids)) {
		return 0, ErrNotFound
	}
	tag, err := tx.Exec(ctx, `
WITH chosen AS (
  SELECT id FROM documents WHERE org_id = $1::uuid AND project_id = $2::uuid AND id::text = ANY($3::text[])
  UNION
  SELECT ds.document_id FROM drawing_sheets ds
  WHERE ds.org_id = $1::uuid AND ds.source_id::text = ANY($3::text[])
)
UPDATE documents SET profile_read = $4
WHERE org_id = $1::uuid AND project_id = $2::uuid AND id IN (SELECT id FROM chosen)`, orgID, projectID, ids, setting)
	if err != nil {
		return 0, err
	}
	if _, err := tx.Exec(ctx, `
DELETE FROM jobs j USING documents d
WHERE j.org_id = $1::uuid AND d.org_id = j.org_id AND d.id = j.document_id AND d.project_id = $2::uuid
  AND j.kind IN ('label', 'evidence') AND j.status IN ('queued', 'failed')
  AND NOT `+readableSQL("$3"), orgID, projectID, nilIfEmpty(readKinds)); err != nil {
		return 0, err
	}
	return int(tag.RowsAffected()), s.finishProfileWrite(ctx, tx, orgID, projectID)
}

func unique(ids []string) []string {
	seen := map[string]bool{}
	var out []string
	for _, id := range ids {
		if !seen[id] {
			seen[id] = true
			out = append(out, id)
		}
	}
	return out
}

// nilIfEmpty turns an empty kind list into SQL NULL (no kind filter).
func nilIfEmpty(kinds []string) []string {
	if len(kinds) == 0 {
		return nil
	}
	return kinds
}

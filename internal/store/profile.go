package store

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"

	"sitewise/internal/profile"
)

// Profile queries use pgx directly: JSONB rows and per-project batches are
// clearer here than generated wrappers. Every statement is scoped by org_id.

const wholePartLabel = "Whole project"

// ProfileView is what the profile page reads: parts, precomputed rows and
// how fresh they are.
type ProfileView struct {
	Parts             []profile.Part
	Rows              []profile.Row
	BuiltAt           *time.Time
	ThresholdsVersion string
	PendingDocuments  int
}

// ProfileSnapshot is the input one rebuild reconciles, read under the
// project's profile lock.
type ProfileSnapshot struct {
	Parts []profile.Part
	Facts []profile.Fact
	User  []profile.UserValue
}

// EnsureWholePart creates the project's "Whole project" part once.
func (s *Store) EnsureWholePart(ctx context.Context, orgID, projectID string) (profile.Part, error) {
	return ensureWholePart(ctx, s.pool, orgID, projectID)
}

func ensureWholePart(ctx context.Context, q interface {
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
}, orgID, projectID string) (profile.Part, error) {
	var p profile.Part
	err := q.QueryRow(ctx, `
WITH ins AS (
  INSERT INTO project_parts (org_id, id, project_id, label, kind)
  SELECT $1::uuid, $3::uuid, $2::uuid, $4, 'whole'
  WHERE EXISTS (SELECT 1 FROM projects WHERE org_id = $1::uuid AND id = $2::uuid)
  ON CONFLICT (org_id, project_id, label) DO NOTHING
  RETURNING id::text, label, kind, COALESCE(ncc_class, ''))
SELECT * FROM ins
UNION ALL
SELECT id::text, label, kind, COALESCE(ncc_class, '') FROM project_parts
WHERE org_id = $1::uuid AND project_id = $2::uuid AND kind = 'whole'
LIMIT 1`, orgID, projectID, newID(), wholePartLabel).Scan(&p.ID, &p.Label, &p.Kind, &p.NCCClass)
	if errors.Is(err, pgx.ErrNoRows) {
		return profile.Part{}, ErrNotFound
	}
	return p, err
}

// CreatePart adds a part to a project in the org.
func (s *Store) CreatePart(ctx context.Context, orgID, projectID, label, kind, nccClass string) (profile.Part, error) {
	p := profile.Part{ID: newID(), Label: label, Kind: kind, NCCClass: nccClass}
	tag, err := s.pool.Exec(ctx, `
INSERT INTO project_parts (org_id, id, project_id, label, kind, ncc_class)
SELECT $1::uuid, $2::uuid, $3::uuid, $4, $5, NULLIF($6, '')
WHERE EXISTS (SELECT 1 FROM projects WHERE org_id = $1::uuid AND id = $3::uuid)`,
		orgID, p.ID, projectID, label, kind, nccClass)
	if err != nil {
		return profile.Part{}, err
	}
	if tag.RowsAffected() == 0 {
		return profile.Part{}, ErrNotFound
	}
	return p, nil
}

// UpdatePart renames a part or changes its kind or class. Nil leaves a field.
func (s *Store) UpdatePart(ctx context.Context, orgID, projectID, partID string, label, kind, nccClass *string) (profile.Part, error) {
	var p profile.Part
	err := s.pool.QueryRow(ctx, `
UPDATE project_parts SET
  label = COALESCE($4, label),
  kind = CASE WHEN kind = 'whole' THEN kind ELSE COALESCE($5, kind) END,
  ncc_class = CASE WHEN $6::text IS NULL THEN ncc_class ELSE NULLIF($6, '') END
WHERE org_id = $1::uuid AND project_id = $2::uuid AND id = $3::uuid
RETURNING id::text, label, kind, COALESCE(ncc_class, '')`,
		orgID, projectID, partID, label, kind, nccClass).Scan(&p.ID, &p.Label, &p.Kind, &p.NCCClass)
	if errors.Is(err, pgx.ErrNoRows) {
		return profile.Part{}, ErrNotFound
	}
	return p, err
}

// StoredFact is a profile reading to store for one document.
type StoredFact struct {
	PassageID  string
	QuestionID string
	Value      string
	Unit       string
	Basis      string
	PartLabel  string
	Excerpt    string
	Confidence *float64
	DecidedBy  string
}

// ReplaceDocumentFacts replaces one document's facts whose question id
// starts with prefix, so a rerun stage never duplicates its readings.
func (s *Store) ReplaceDocumentFacts(ctx context.Context, orgID, documentID, prefix, questionVersion string, facts []StoredFact) error {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	var projectID string
	if err := tx.QueryRow(ctx, `SELECT project_id::text FROM documents WHERE org_id = $1::uuid AND id = $2::uuid`,
		orgID, documentID).Scan(&projectID); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrNotFound
		}
		return err
	}
	if _, err := tx.Exec(ctx, `DELETE FROM profile_facts WHERE org_id = $1::uuid AND document_id = $2::uuid AND starts_with(question_id, $3)`,
		orgID, documentID, prefix); err != nil {
		return err
	}
	batch := &pgx.Batch{}
	for _, f := range facts {
		if !strings.HasPrefix(f.QuestionID, prefix) {
			continue
		}
		batch.Queue(`
INSERT INTO profile_facts (org_id, id, project_id, document_id, passage_id, question_id, value, unit, basis, part_label,
  excerpt, confidence, decided_by, question_version)
VALUES ($1::uuid, $2::uuid, $3::uuid, $4::uuid, NULLIF($5, '')::uuid, $6, $7, $8, $9, $10, $11, $12, $13, $14)`,
			orgID, newID(), projectID, documentID, f.PassageID, f.QuestionID, f.Value, f.Unit, f.Basis, f.PartLabel,
			cutRunes(f.Excerpt, 120), f.Confidence, f.DecidedBy, questionVersion)
	}
	if batch.Len() > 0 {
		if err := tx.SendBatch(ctx, batch).Close(); err != nil {
			return err
		}
	}
	return tx.Commit(ctx)
}

// SetUserValue records the user's word for one key. The caller rebuilds the
// rows (RebuildProfile) so the page reads it at once.
func (s *Store) SetUserValue(ctx context.Context, orgID, projectID, partID, userID, key string, value *string, note string) error {
	tag, err := s.pool.Exec(ctx, `
INSERT INTO profile_user_values (org_id, project_id, part_id, key, value, note, user_id)
SELECT $1::uuid, $2::uuid, $3::uuid, $4, $5, $6, $7::uuid
WHERE EXISTS (SELECT 1 FROM project_parts WHERE org_id = $1::uuid AND project_id = $2::uuid AND id = $3::uuid)
ON CONFLICT (org_id, project_id, part_id, key) DO UPDATE
SET value = EXCLUDED.value, note = EXCLUDED.note, user_id = EXCLUDED.user_id,
    version = profile_user_values.version + 1, updated_at = now()`,
		orgID, projectID, partID, key, value, cutRunes(note, 120), userID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

// RebuildProfile reconciles one project under a per-project lock, writes
// every row, records the build and appends a profile event, in one
// transaction. Concurrent rebuilds and edits serialise on the lock, so a
// rebuild never drops a user value written after it started.
func (s *Store) RebuildProfile(ctx context.Context, orgID, projectID, thresholdsVersion string,
	compute func(ProfileSnapshot) []profile.Row) error {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if _, err := tx.Exec(ctx, `SELECT pg_advisory_xact_lock(hashtextextended($1 || '/' || $2, 0))`, orgID, projectID); err != nil {
		return err
	}
	whole, err := ensureWholePart(ctx, tx, orgID, projectID)
	if err != nil {
		return err
	}
	snap, err := readSnapshot(ctx, tx, orgID, projectID)
	if err != nil {
		return err
	}
	if len(snap.Parts) == 0 {
		snap.Parts = []profile.Part{whole}
	}
	rows := compute(snap)
	if _, err := tx.Exec(ctx, `DELETE FROM profile_rows WHERE org_id = $1::uuid AND project_id = $2::uuid`, orgID, projectID); err != nil {
		return err
	}
	batch := &pgx.Batch{}
	for _, r := range rows {
		sources, _ := json.Marshal(nonNil(r.Sources))
		alts, _ := json.Marshal(nonNilAlts(r.Alternatives))
		var derived []byte
		if r.Derived != nil {
			derived, _ = json.Marshal(r.Derived)
		}
		batch.Queue(`
INSERT INTO profile_rows (org_id, project_id, part_id, key, value, band, assertion, note, tenders, sources, alternatives, derived)
VALUES ($1::uuid, $2::uuid, $3::uuid, $4, $5, $6, $7, $8, $9, $10, $11, $12)`,
			orgID, projectID, r.PartID, r.Key, r.Value, r.Band, r.Assertion, cutRunes(r.Note, 120), r.Tenders, sources, alts, derived)
	}
	if batch.Len() > 0 {
		if err := tx.SendBatch(ctx, batch).Close(); err != nil {
			return err
		}
	}
	if _, err := tx.Exec(ctx, `
INSERT INTO profile_builds (org_id, project_id, built_at, thresholds_version) VALUES ($1::uuid, $2::uuid, now(), $3)
ON CONFLICT (org_id, project_id) DO UPDATE SET built_at = now(), thresholds_version = EXCLUDED.thresholds_version`,
		orgID, projectID, thresholdsVersion); err != nil {
		return err
	}
	payload, _ := json.Marshal(map[string]string{"project_id": projectID})
	if _, err := appendEvent(ctx, s.q.WithTx(tx), orgID, "profile", "", string(payload)); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

type rowQuerier interface {
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
}

func readSnapshot(ctx context.Context, q rowQuerier, orgID, projectID string) (ProfileSnapshot, error) {
	var snap ProfileSnapshot
	parts, err := readParts(ctx, q, orgID, projectID)
	if err != nil {
		return snap, err
	}
	snap.Parts = parts
	rows, err := q.Query(ctx, `
SELECT f.question_id, f.value, f.unit, f.basis, f.part_label, f.excerpt, f.confidence, f.decided_by,
       f.document_id::text, COALESCE(f.passage_id::text, ''),
       COALESCE((SELECT d.value FROM decisions d WHERE d.org_id = f.org_id AND d.document_id = f.document_id AND d.field = 'kind'), ''),
       EXISTS (SELECT 1 FROM supersessions s WHERE s.org_id = f.org_id AND s.prior_document_id = f.document_id)
FROM profile_facts f
WHERE f.org_id = $1::uuid AND f.project_id = $2::uuid
ORDER BY f.document_id, f.passage_id, f.question_id`, orgID, projectID)
	if err != nil {
		return snap, err
	}
	for rows.Next() {
		var f profile.Fact
		if err := rows.Scan(&f.QuestionID, &f.Value, &f.Unit, &f.Basis, &f.PartLabel, &f.Excerpt, &f.Confidence,
			&f.DecidedBy, &f.DocumentID, &f.PassageID, &f.DocumentKind, &f.Superseded); err != nil {
			rows.Close()
			return snap, err
		}
		snap.Facts = append(snap.Facts, f)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return snap, err
	}
	urows, err := q.Query(ctx, `
SELECT part_id::text, key, value, note FROM profile_user_values
WHERE org_id = $1::uuid AND project_id = $2::uuid ORDER BY part_id, key`, orgID, projectID)
	if err != nil {
		return snap, err
	}
	defer urows.Close()
	for urows.Next() {
		var u profile.UserValue
		if err := urows.Scan(&u.PartID, &u.Key, &u.Value, &u.Note); err != nil {
			return snap, err
		}
		snap.User = append(snap.User, u)
	}
	return snap, urows.Err()
}

func readParts(ctx context.Context, q rowQuerier, orgID, projectID string) ([]profile.Part, error) {
	rows, err := q.Query(ctx, `
SELECT id::text, label, kind, COALESCE(ncc_class, '') FROM project_parts
WHERE org_id = $1::uuid AND project_id = $2::uuid ORDER BY kind <> 'whole', created_at, label`, orgID, projectID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []profile.Part
	for rows.Next() {
		var p profile.Part
		if err := rows.Scan(&p.ID, &p.Label, &p.Kind, &p.NCCClass); err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	return out, rows.Err()
}

// ProfileInput reads a project's snapshot without locking, for a dry run.
func (s *Store) ProfileInput(ctx context.Context, orgID, projectID string) (ProfileSnapshot, error) {
	return readSnapshot(ctx, s.pool, orgID, projectID)
}

// ReadProfile returns the precomputed profile. It never reconciles.
func (s *Store) ReadProfile(ctx context.Context, orgID, projectID string) (ProfileView, error) {
	var v ProfileView
	var exists bool
	err := s.pool.QueryRow(ctx, `
SELECT EXISTS (SELECT 1 FROM projects WHERE org_id = $1::uuid AND id = $2::uuid),
       (SELECT built_at FROM profile_builds WHERE org_id = $1::uuid AND project_id = $2::uuid),
       COALESCE((SELECT thresholds_version FROM profile_builds WHERE org_id = $1::uuid AND project_id = $2::uuid), ''),
       (SELECT count(DISTINCT j.document_id) FROM jobs j JOIN documents d ON d.org_id = j.org_id AND d.id = j.document_id
        WHERE j.org_id = $1::uuid AND d.project_id = $2::uuid AND j.status IN ('queued', 'leased')
          AND j.kind IN ('full_text', 'label', 'evidence'))`,
		orgID, projectID).Scan(&exists, &v.BuiltAt, &v.ThresholdsVersion, &v.PendingDocuments)
	if err != nil {
		return v, err
	}
	if !exists {
		return v, ErrNotFound
	}
	if v.Parts, err = readParts(ctx, s.pool, orgID, projectID); err != nil {
		return v, err
	}
	rows, err := s.pool.Query(ctx, `
SELECT part_id::text, key, value, band, assertion, note, tenders, sources, alternatives, derived
FROM profile_rows WHERE org_id = $1::uuid AND project_id = $2::uuid ORDER BY part_id, key`, orgID, projectID)
	if err != nil {
		return v, err
	}
	defer rows.Close()
	for rows.Next() {
		var r profile.Row
		var sources, alts, derived []byte
		if err := rows.Scan(&r.PartID, &r.Key, &r.Value, &r.Band, &r.Assertion, &r.Note, &r.Tenders, &sources, &alts, &derived); err != nil {
			return v, err
		}
		_ = json.Unmarshal(sources, &r.Sources)
		_ = json.Unmarshal(alts, &r.Alternatives)
		if len(derived) > 0 {
			r.Derived = &profile.Derived{}
			_ = json.Unmarshal(derived, r.Derived)
		}
		v.Rows = append(v.Rows, r)
	}
	return v, rows.Err()
}

func nonNil(s []profile.Source) []profile.Source {
	if s == nil {
		return []profile.Source{}
	}
	return s
}

func nonNilAlts(a []profile.Alternative) []profile.Alternative {
	if a == nil {
		return []profile.Alternative{}
	}
	return a
}

func cutRunes(s string, n int) string {
	r := []rune(s)
	if len(r) <= n {
		return s
	}
	return string(r[:n])
}

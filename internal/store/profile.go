package store

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/jackc/pgx/v5"

	"sitewise/internal/profile"
	"sitewise/internal/works"
)

// Profile queries use pgx directly: JSONB rows and per-project batches are
// clearer here than generated wrappers. Every statement is scoped by org_id.

const wholePartLabel = "Whole project"

// ProfileView is what the profile page reads: parts, precomputed rows and
// how fresh they are.
type ProfileView struct {
	Revision              int64
	InputFingerprint      string
	KnowledgeVersion      string
	QuestionVersion       string
	ReadKinds             []string
	Inputs, CurrentInputs Revisions
	Coverage              []SourceCoverage
	Parts                 []profile.Part
	Rows                  []profile.Row
	BuiltAt               *time.Time
	ThresholdsVersion     string
	PendingDocuments      int
	// ActiveDocuments counts live leases, never merely queued or expired work.
	ActiveDocuments int
	// UnreadDocuments have their text split but have not been asked for
	// reading; the user's "Update project profile" queues them.
	UnreadDocuments int
	// ReadDocuments and SkippedDocuments split the project's documents by
	// whether the profile reads them; SkippedKind is the most common kind
	// among the skipped ones.
	ReadDocuments    int
	SkippedDocuments int
	SkippedKind      string
	FailedDocuments  int
	PaymentRequired  bool
}

// ProfileSnapshot is the input one rebuild reconciles, read under the
// project's profile lock.
type ProfileSnapshot struct {
	Site      Site
	Documents []json.RawMessage
	WorkItems []json.RawMessage // WP-20 populates this with its authoritative set.
	Parts     []profile.Part
	Facts     []profile.Fact
	User      []profile.UserValue
	Planning  []profile.PlanningValue
}

// projectSiteSQL is the site of project $2 in org $1. Parts belong to the
// site; a project reaches them through its site (migration 011).
const projectSiteSQL = `(SELECT site_id FROM projects WHERE org_id = $1::uuid AND id = $2::uuid)`

// EnsureWholePart creates the "Whole project" part of the project's site once.
func (s *Store) EnsureWholePart(ctx context.Context, orgID, projectID string) (profile.Part, error) {
	return ensureWholePart(ctx, s.pool, orgID, projectID)
}

func ensureWholePart(ctx context.Context, q interface {
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
}, orgID, projectID string) (profile.Part, error) {
	var p profile.Part
	read := func() error {
		return q.QueryRow(ctx, `SELECT id::text,label,kind,COALESCE(ncc_class,'') FROM project_parts
WHERE org_id=$1::uuid AND site_id=`+projectSiteSQL+` AND kind='whole'`, orgID, projectID).
			Scan(&p.ID, &p.Label, &p.Kind, &p.NCCClass)
	}
	// Existing parts need no insert attempt on every edit and rebuild.
	if err := read(); !errors.Is(err, pgx.ErrNoRows) {
		return p, err
	}
	err := q.QueryRow(ctx, `
  INSERT INTO project_parts (org_id, id, site_id, created_by_project_id, label, kind)
  SELECT $1::uuid, $3::uuid, p.site_id, p.id, $4, 'whole'
  FROM projects p WHERE p.org_id = $1::uuid AND p.id = $2::uuid
  ON CONFLICT (org_id, site_id) WHERE kind = 'whole' DO NOTHING
  RETURNING id::text, label, kind, COALESCE(ncc_class, '')`, orgID, projectID, newID(), wholePartLabel).Scan(&p.ID, &p.Label, &p.Kind, &p.NCCClass)
	if errors.Is(err, pgx.ErrNoRows) {
		// A concurrent insert can win after our INSERT's snapshot was taken.
		// Read again in a new statement to see the committed winner.
		err = read()
	}
	if errors.Is(err, pgx.ErrNoRows) {
		return profile.Part{}, ErrNotFound
	}
	return p, err
}

// CreatePart adds a part to the site of a project in the org.
func (s *Store) CreatePart(ctx context.Context, orgID, projectID, label, kind, nccClass string) (profile.Part, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return profile.Part{}, err
	}
	defer tx.Rollback(ctx)
	if err := lockProject(ctx, tx, orgID, projectID); err != nil {
		return profile.Part{}, err
	}

	p := profile.Part{ID: newID(), Label: label, Kind: kind, NCCClass: nccClass}
	tag, err := tx.Exec(ctx, `
INSERT INTO project_parts (org_id, id, site_id, created_by_project_id, label, kind, ncc_class)
SELECT $1::uuid, $3::uuid, p.site_id, p.id, $4, $5, NULLIF($6, '')
FROM projects p WHERE p.org_id = $1::uuid AND p.id = $2::uuid`,
		orgID, projectID, p.ID, label, kind, nccClass)
	if err != nil {
		return profile.Part{}, err
	}
	if tag.RowsAffected() == 0 {
		return profile.Part{}, ErrNotFound
	}
	return p, s.finishProfileWrite(ctx, tx, orgID, projectID)
}

// UpdatePart renames a part or changes its kind or class. Nil leaves a field.
func (s *Store) UpdatePart(ctx context.Context, orgID, projectID, partID string, label, kind, nccClass *string) (profile.Part, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return profile.Part{}, err
	}
	defer tx.Rollback(ctx)
	if err := lockProject(ctx, tx, orgID, projectID); err != nil {
		return profile.Part{}, err
	}

	var p profile.Part
	err = tx.QueryRow(ctx, `
UPDATE project_parts SET
  label = COALESCE($4, label),
  kind = CASE WHEN kind = 'whole' THEN kind ELSE COALESCE($5, kind) END,
  ncc_class = CASE WHEN $6::text IS NULL THEN ncc_class ELSE NULLIF($6, '') END
WHERE org_id = $1::uuid AND site_id = `+projectSiteSQL+` AND id = $3::uuid
RETURNING id::text, label, kind, COALESCE(ncc_class, '')`,
		orgID, projectID, partID, label, kind, nccClass).Scan(&p.ID, &p.Label, &p.Kind, &p.NCCClass)
	if errors.Is(err, pgx.ErrNoRows) {
		return profile.Part{}, ErrNotFound
	}
	if err != nil {
		return p, err
	}
	return p, s.finishProfileWrite(ctx, tx, orgID, projectID)
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
// starts with any of prefixes, so a rerun stage never duplicates readings.
func (s *Store) ReplaceDocumentFacts(ctx context.Context, orgID, documentID string, prefixes []string, questionVersion string, facts []StoredFact) error {
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
	if err := lockProject(ctx, tx, orgID, projectID); err != nil {
		return err
	}
	for _, prefix := range prefixes {
		if _, err := tx.Exec(ctx, `DELETE FROM profile_facts WHERE org_id = $1::uuid AND document_id = $2::uuid AND starts_with(question_id, $3)`,
			orgID, documentID, prefix); err != nil {
			return err
		}
	}
	batch := &pgx.Batch{}
	for _, f := range facts {
		if !hasAnyPrefix(f.QuestionID, prefixes) {
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
	return s.finishProfileWrite(ctx, tx, orgID, projectID)
}

// UserWrite is one edit of the user's word on a profile key (plan §4.1, §4.3).
type UserWrite struct {
	Value *string
	// State is set, cleared (the user blanked it) or unknown (explicitly
	// unresolved); Value is nil unless State is set.
	State string
	Note  string
	// Origin is user or assumption; Meaning is stated, requirement,
	// allowance or forecast.
	Origin, Meaning string
	// Scope is site or project, from the key-scope registry (D-04). A site
	// value belongs to the site and is seen by every project on it.
	Scope string
	// Version is the version the client edited from (0 for a new value).
	// Nil skips the check, for clients that do not send one.
	Version *int64
}

// lockProject serialises writes to one project with its rebuilds. A site
// value is also guarded per project: version 1 has one project per site
// (projects_one_per_site_v1); a shared site needs a site lock here too.
func lockProject(ctx context.Context, tx pgx.Tx, orgID, projectID string) error {
	_, err := tx.Exec(ctx, `SELECT pg_advisory_xact_lock(hashtextextended($1 || '/' || $2, 0))`, orgID, projectID)
	return err
}

// userValueOwner resolves the project's site and checks the part is on it.
func userValueOwner(ctx context.Context, tx pgx.Tx, orgID, projectID, partID string) (string, error) {
	var siteID string
	err := tx.QueryRow(ctx, `
SELECT p.site_id::text FROM projects p
JOIN project_parts pp ON pp.org_id = p.org_id AND pp.site_id = p.site_id AND pp.id = $3::uuid
WHERE p.org_id = $1::uuid AND p.id = $2::uuid`, orgID, projectID, partID).Scan(&siteID)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", ErrNotFound
	}
	return siteID, err
}

// ownerSQL selects one key's row by its owner: $1 org, $2 the site or the
// project, $3 part, $4 key. Each scope is its own literal query so the
// matching partial unique index is used; ownerArgs gives its arguments.
func ownerSQL(scope string) (string, error) {
	switch scope {
	case "site":
		return `org_id = $1::uuid AND scope = 'site' AND site_id = $2::uuid AND part_id = $3::uuid AND key = $4`, nil
	case "project":
		return `org_id = $1::uuid AND scope = 'project' AND project_id = $2::uuid AND part_id = $3::uuid AND key = $4`, nil
	}
	return "", fmt.Errorf("user value scope %q", scope)
}

func ownerArgs(scope, orgID, projectID, siteID, partID, key string) []any {
	owner := projectID
	if scope == "site" {
		owner = siteID
	}
	return []any{orgID, owner, partID, key}
}

// currentUserValue returns the stored version of a key (0 when absent).
func currentUserValue(ctx context.Context, tx pgx.Tx, orgID, projectID, siteID, partID, key, scope string) (int64, error) {
	where, err := ownerSQL(scope)
	if err != nil {
		return 0, err
	}
	var version int64
	err = tx.QueryRow(ctx, `SELECT version FROM profile_user_values WHERE `+where+` FOR UPDATE`,
		ownerArgs(scope, orgID, projectID, siteID, partID, key)...).Scan(&version)
	if errors.Is(err, pgx.ErrNoRows) {
		return 0, nil
	}
	return version, err
}

// SetUserValue records the user's word for one key and returns its new
// version. A stale expected version returns the current one with
// ErrVersionConflict. An omitted origin or meaning keeps the stored one, so
// an edit never turns an assumption back into a stated fact (D-06). The
// configured store rebuilds the rows in this transaction; an unconfigured
// store leaves the projection explicitly stale.
func (s *Store) SetUserValue(ctx context.Context, orgID, projectID, partID, userID, key string, w UserWrite) (int64, error) {
	if _, err := ownerSQL(w.Scope); err != nil {
		return 0, err
	}
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return 0, err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if err := lockProject(ctx, tx, orgID, projectID); err != nil {
		return 0, err
	}
	if err := packageActor(ctx, tx, orgID, userID); err != nil {
		return 0, err
	}
	siteID, err := userValueOwner(ctx, tx, orgID, projectID, partID)
	if err != nil {
		return 0, err
	}
	current, err := currentUserValue(ctx, tx, orgID, projectID, siteID, partID, key, w.Scope)
	if err != nil {
		return 0, err
	}
	if w.Version != nil && *w.Version != current {
		return current, ErrVersionConflict
	}
	// A key the registry has moved to the other scope keeps one row only.
	other := "project"
	if w.Scope == "project" {
		other = "site"
	}
	otherWhere, _ := ownerSQL(other)
	// Version validation has completed under the project lock. These two
	// mutations are independent of their results, so share one round trip.
	batch := &pgx.Batch{}
	batch.Queue(`DELETE FROM profile_user_values WHERE `+otherWhere,
		ownerArgs(other, orgID, projectID, siteID, partID, key)...)
	var project any = projectID
	if w.Scope == "site" {
		project = nil
	}
	var version int64
	batch.Queue(`
INSERT INTO profile_user_values (org_id, id, project_id, site_id, part_id, scope, key, value, value_state, note,
  origin, meaning, user_id)
VALUES ($1::uuid, gen_random_uuid(), $2::uuid, $3::uuid, $4::uuid, $5, $6, $7, $8, $9,
  COALESCE($10::text, 'user'), COALESCE($11::text, 'stated'), $12::uuid)
ON CONFLICT `+userValueConflict(w.Scope)+` DO UPDATE
SET value = EXCLUDED.value, value_state = EXCLUDED.value_state, note = EXCLUDED.note,
    origin = COALESCE($10::text, profile_user_values.origin), meaning = COALESCE($11::text, profile_user_values.meaning),
    user_id = EXCLUDED.user_id, version = profile_user_values.version + 1, updated_at = now()
RETURNING version`,
		orgID, project, siteID, partID, w.Scope, key, w.Value, orDefault(w.State, "set"), cutRunes(w.Note, 120),
		nilIfBlank(w.Origin), nilIfBlank(w.Meaning), userID)
	results := tx.SendBatch(ctx, batch)
	defer results.Close()
	if _, err = results.Exec(); err != nil {
		return 0, err
	}
	err = results.QueryRow().Scan(&version)
	if err != nil {
		return 0, err
	}
	if err = results.Close(); err != nil {
		return 0, err
	}
	return version, s.finishProfileWriteLocked(ctx, tx, orgID, projectID)
}

func userValueConflict(scope string) string {
	if scope == "site" {
		return `(org_id, site_id, part_id, key) WHERE scope = 'site'`
	}
	return `(org_id, project_id, part_id, key) WHERE scope = 'project'`
}

func orDefault(v, d string) string {
	if v == "" {
		return d
	}
	return v
}

func nilIfBlank(v string) any {
	if v == "" {
		return nil
	}
	return v
}

// DeleteUserValue removes the user's word for one key, so evidence shows
// again. A stale expected version returns the current version with
// ErrVersionConflict.
func (s *Store) DeleteUserValue(ctx context.Context, orgID, projectID, partID, key, scope string, version *int64) (int64, error) {
	where, err := ownerSQL(scope)
	if err != nil {
		return 0, err
	}
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return 0, err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if err := lockProject(ctx, tx, orgID, projectID); err != nil {
		return 0, err
	}
	siteID, err := userValueOwner(ctx, tx, orgID, projectID, partID)
	if err != nil {
		return 0, err
	}
	current, err := currentUserValue(ctx, tx, orgID, projectID, siteID, partID, key, scope)
	if err != nil {
		return 0, err
	}
	if version != nil && *version != current {
		return current, ErrVersionConflict
	}
	if _, err := tx.Exec(ctx, `DELETE FROM profile_user_values WHERE `+where,
		ownerArgs(scope, orgID, projectID, siteID, partID, key)...); err != nil {
		return 0, err
	}
	return 0, s.finishProfileWriteLocked(ctx, tx, orgID, projectID)
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
	b := ProfileBuild{ThresholdsVersion: thresholdsVersion, QuestionVersion: profile.QuestionVersion, Compute: compute}
	if s.profileBuild != nil {
		b = *s.profileBuild
		b.ThresholdsVersion = thresholdsVersion
		b.Compute = compute
	}
	if err := s.rebuildProfileTx(ctx, tx, orgID, projectID, b); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func (s *Store) rebuildProfileTx(ctx context.Context, tx pgx.Tx, orgID, projectID string, b ProfileBuild) error {
	if b.proposalError != nil {
		return b.proposalError
	}
	return s.rebuildProfileProjectionTx(ctx, tx, orgID, projectID, b, b.proposalEvaluator)
}

func (s *Store) rebuildProfileProjectionTx(ctx context.Context, tx pgx.Tx, orgID, projectID string, b ProfileBuild, evaluator *works.Evaluator) error {
	if b.Compute == nil {
		return errors.New("profile compute is not configured")
	}
	if err := lockProject(ctx, tx, orgID, projectID); err != nil {
		return err
	}
	return s.rebuildProfileProjectionLocked(ctx, tx, orgID, projectID, b, evaluator)
}

// rebuildProfileProjectionLocked requires the project's transaction advisory lock.
// Keep the locking entry point for standalone rebuilds and unaudited callers.
func (s *Store) rebuildProfileProjectionLocked(ctx context.Context, tx pgx.Tx, orgID, projectID string, b ProfileBuild, evaluator *works.Evaluator) error {

	if b.Compute == nil {
		return errors.New("profile compute is not configured")
	}
	snap, err := readSnapshot(ctx, tx, orgID, projectID)
	if err != nil {
		return err
	}

	hasWhole := false
	for _, part := range snap.Parts {
		hasWhole = hasWhole || part.Kind == "whole"
	}
	if !hasWhole {
		whole, err := ensureWholePart(ctx, tx, orgID, projectID)
		if err != nil {
			return err
		}
		snap.Parts = append([]profile.Part{whole}, snap.Parts...)
	}
	if err := readFingerprintInputs(ctx, tx, orgID, projectID, &snap); err != nil {
		return err
	}
	rows := b.Compute(snap)
	if b.Catalog != nil {
		if err := s.syncProposedWorks(ctx, tx, orgID, projectID, snap.Site.ID, profile.ProposedWorks(projectID, rows, snap.Parts, b.Catalog), b.Catalog); err != nil {
			return err
		}
	}
	items, err := readWorkItems(ctx, tx, orgID, projectID)
	if err != nil {
		return err
	}
	var proposalInputs works.ProposalInput
	proposalFingerprint, previousProposalFingerprint := "", ""
	rebuildProposals := evaluator != nil
	if evaluator != nil {
		proposalInputs, err = profile.ProposalInputs(rows, snap.Parts, items, b.Catalog)
		if err != nil {
			return err
		}
		proposalInputs, proposalFingerprint, err = evaluator.MaterializeInput(proposalInputs)
		if err != nil {
			return err
		}
		if err = tx.QueryRow(ctx, `SELECT COALESCE((SELECT inputs->>'proposal_fingerprint' FROM profile_builds WHERE org_id=$1::uuid AND project_id=$2::uuid),'')`, orgID, projectID).Scan(&previousProposalFingerprint); err != nil {
			return err
		}
		rebuildProposals = proposalFingerprint != previousProposalFingerprint

	}
	snap.WorkItems = workFingerprint(items)
	rows = profile.ProjectWorkScope(rows, items)
	// Compute from immutable inputs while this goroutine writes the projection.
	// Only this goroutine touches the transaction. Join on every exit so a
	// failed write cannot leave background work beyond the rebuild's lifetime.
	var fingerprint string
	var fingerprintErr error
	var proposals []works.Proposal
	var proposalErr error
	var computing sync.WaitGroup
	computing.Go(func() { fingerprint, fingerprintErr = profileFingerprint(snap, b) })
	if rebuildProposals {
		computing.Go(func() {
			proposals, proposalErr = evaluator.Evaluate(proposalInputs)
		})
	}
	defer computing.Wait()
	// The project lock freezes decisions and projections. Read their small
	// indexes while the evaluator runs; only this goroutine uses the transaction.
	var proposalStored proposalPersistenceInput
	if rebuildProposals {
		proposalStored, err = readProposalPersistenceInput(ctx, tx, orgID, projectID)
		if err != nil {
			return err
		}
	}

	inputs, err := readRevisions(ctx, tx, orgID, projectID)
	if err != nil {
		return err
	}
	rawInputs, err := json.Marshal(struct {
		Revisions
		ReadKinds           []string `json:"read_kinds"`
		ProposalFingerprint string   `json:"proposal_fingerprint,omitempty"`
	}{inputs, b.ReadKinds, proposalFingerprint})
	if err != nil {
		return err
	}

	if err := writeProfileProjection(ctx, tx, orgID, projectID, snap.Site.ID, rows); err != nil {
		return err
	}

	computing.Wait()
	if fingerprintErr != nil {
		return fingerprintErr
	}
	if rebuildProposals {
		if proposalErr != nil {
			return proposalErr
		}
		if err := writePreparedProposals(ctx, tx, orgID, projectID, snap.Site.ID, proposals, proposalStored); err != nil {
			return err
		}
	}
	var revision int64
	err = tx.QueryRow(ctx, `
INSERT INTO profile_builds (org_id, project_id, built_at, thresholds_version, revision, input_fingerprint, knowledge_version, question_version, inputs)
VALUES ($1::uuid,$2::uuid,now(),$3,1,$4,$5,$6,$7)
ON CONFLICT (org_id,project_id) DO UPDATE SET built_at=now(), thresholds_version=EXCLUDED.thresholds_version,
 revision=profile_builds.revision+1, input_fingerprint=EXCLUDED.input_fingerprint, knowledge_version=EXCLUDED.knowledge_version,
 question_version=EXCLUDED.question_version, inputs=EXCLUDED.inputs RETURNING revision`,
		orgID, projectID, b.ThresholdsVersion, fingerprint, b.KnowledgeVersion, b.QuestionVersion, rawInputs).Scan(&revision)
	if err != nil {
		return err
	}
	payload, _ := json.Marshal(map[string]any{"project_id": projectID, "revision": revision})
	_, err = appendEvent(ctx, s.q.WithTx(tx), orgID, "profile", "", string(payload))

	return err
}

type rowQuerier interface {
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
}

func readSnapshot(ctx context.Context, q rowQuerier, orgID, projectID string) (ProfileSnapshot, error) {
	var snap ProfileSnapshot
	// Non-nil marks a complete all-document fingerprint read, even for an empty
	// project. The provenance rows and fingerprint metadata share this statement.
	snap.Documents = []json.RawMessage{}
	parts, err := readParts(ctx, q, orgID, projectID)
	if err != nil {
		return snap, err
	}
	snap.Parts = parts
	// OFFSET 0 keeps each lateral lookup parameterised by its exact ID instead
	// of flattening into joins that multiply scans under stale statistics (F31).
	// Return each fact, document and passage metadata record once. Match them
	// in Go so stale estimates cannot turn the joins into repeated scans.
	// One statement also keeps facts and source metadata on the same snapshot
	// when a background extraction replaces passages during a rebuild.
	// Replan this size-sensitive query for each execution: a generic plan
	// cached before upload can retain a full source-table scan per fact.
	// Exec uses bound parameters in one round trip, without changing the
	// connection's statement-cache policy for other queries.
	snapshotSQL := `
WITH facts AS MATERIALIZED (
 SELECT * FROM profile_facts WHERE org_id=$1::uuid AND project_id=$2::uuid
), source_documents AS MATERIALIZED (
 SELECT ids.document_id, doc.profile_read, doc.filename, doc.document_number, doc.revision, fl.sha256,
 COALESCE((SELECT value FROM decisions WHERE org_id=$1::uuid AND document_id=ids.document_id AND field='kind'),'') AS kind,
 EXISTS(SELECT 1 FROM supersessions WHERE org_id=$1::uuid AND prior_document_id=ids.document_id) AS superseded
 FROM (SELECT id AS document_id FROM documents WHERE org_id=$1::uuid AND project_id=$2::uuid) ids
 LEFT JOIN LATERAL (
 SELECT profile_read,filename,document_number,revision,file_id FROM documents WHERE org_id=$1::uuid AND id=ids.document_id OFFSET 0
 ) doc ON true
 LEFT JOIN LATERAL (
 SELECT sha256 FROM files WHERE org_id=$1::uuid AND id=doc.file_id OFFSET 0
 ) fl ON true
)
-- Row tags let one statement carry facts and each distinct source once.
SELECT 'fact',f.id::text,f.question_version,f.question_id,f.value,f.unit,f.basis,f.part_label,f.excerpt,f.confidence,f.decided_by,
       f.document_id::text,COALESCE(f.passage_id::text, ''),
       ''::text,'auto'::text,false,''::text,''::text,''::text,''::text,0::integer,''::text,''::text,0::integer,0::integer,NULL::jsonb
FROM facts f
UNION ALL
SELECT 'document',''::text,''::text,''::text,''::text,''::text,''::text,''::text,''::text,NULL::double precision,''::text,
       doc.document_id::text,''::text,
       COALESCE(doc.kind,''),COALESCE(doc.profile_read,'auto'),COALESCE(doc.superseded,false),
       COALESCE(encode(doc.sha256,'hex'),''),COALESCE(doc.filename,''),COALESCE(doc.document_number,''),COALESCE(doc.revision,''),0::integer,''::text,''::text,0::integer,0::integer,
       jsonb_build_object('id',doc.document_id,'read',doc.profile_read,
       'filename',doc.filename,'number',doc.document_number,'revision',doc.revision,
       'kind',doc.kind,'superseded',doc.superseded,'sha256',encode(doc.sha256,'hex'))
FROM source_documents doc
UNION ALL
SELECT 'passage',''::text,''::text,''::text,''::text,''::text,''::text,''::text,''::text,NULL::double precision,''::text,ps.document_id::text,
       ps.passage_id::text,''::text,'auto'::text,false,''::text,''::text,''::text,''::text,
       ps.page,ps.location,ps.section,ps.start_offset,ps.end_offset,NULL::jsonb
FROM passage_sources ps
WHERE ps.org_id=$1::uuid AND ps.passage_id=ANY(ARRAY(SELECT DISTINCT passage_id FROM facts WHERE passage_id IS NOT NULL))`
	rows, err := q.Query(ctx, snapshotSQL, pgx.QueryExecModeExec, orgID, projectID)
	if err != nil {
		return snap, err
	}
	documents := map[string]profile.Fact{}
	// Historical passage IDs may no longer resolve. Even when an ID exists,
	// its location belongs only to facts from that same source document.
	type sourceKey struct{ document, passage string }
	passages := map[sourceKey]profile.Fact{}
	for rows.Next() {
		var f profile.Fact
		var kind string
		var documentJSON json.RawMessage
		if err := rows.Scan(&kind, &f.ID, &f.QuestionVersion, &f.QuestionID, &f.Value, &f.Unit, &f.Basis, &f.PartLabel, &f.Excerpt, &f.Confidence,
			&f.DecidedBy, &f.DocumentID, &f.PassageID, &f.DocumentKind, &f.ReadSetting, &f.Superseded,
			&f.FileSHA256, &f.Filename, &f.DocumentNumber, &f.Revision, &f.Page, &f.Location, &f.Section,
			&f.StartOffset, &f.EndOffset, &documentJSON); err != nil {
			rows.Close()
			return snap, err
		}
		switch kind {
		case "fact":
			snap.Facts = append(snap.Facts, f)
		case "document":
			documents[f.DocumentID] = f
			snap.Documents = append(snap.Documents, documentJSON)
		case "passage":
			passages[sourceKey{f.DocumentID, f.PassageID}] = f
		}
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return snap, err
	}
	for i := range snap.Facts {
		f := &snap.Facts[i]
		d := documents[f.DocumentID]
		f.DocumentKind, f.ReadSetting, f.Superseded = d.DocumentKind, orDefault(d.ReadSetting, "auto"), d.Superseded
		f.FileSHA256, f.Filename, f.DocumentNumber, f.Revision = d.FileSHA256, d.Filename, d.DocumentNumber, d.Revision
		p := passages[sourceKey{f.DocumentID, f.PassageID}]
		f.Page, f.Location, f.Section, f.StartOffset, f.EndOffset = p.Page, p.Location, p.Section, p.StartOffset, p.EndOffset
	}
	sort.SliceStable(snap.Facts, func(i, j int) bool {
		a, b := snap.Facts[i], snap.Facts[j]
		if a.DocumentID != b.DocumentID {
			return a.DocumentID < b.DocumentID
		}
		if a.PassageID != b.PassageID {
			// Match PostgreSQL's ascending UUID order, including NULLS LAST.
			if a.PassageID == "" {
				return false
			}
			if b.PassageID == "" {
				return true
			}
			return a.PassageID < b.PassageID
		}
		return a.QuestionID < b.QuestionID
	})
	urows, err := q.Query(ctx, `
SELECT part_id::text, key, value, note, value_state, origin, meaning, version, scope, review_status FROM profile_user_values
WHERE org_id = $1::uuid AND ((scope = 'project' AND project_id = $2::uuid)
   OR (scope = 'site' AND site_id = `+projectSiteSQL+`))
ORDER BY part_id, key`, orgID, projectID)
	if err != nil {
		return snap, err
	}
	defer urows.Close()
	for urows.Next() {
		var u profile.UserValue
		if err := urows.Scan(&u.PartID, &u.Key, &u.Value, &u.Note, &u.State, &u.Origin, &u.Meaning, &u.Version, &u.Scope, &u.ReviewStatus); err != nil {
			return snap, err
		}
		snap.User = append(snap.User, u)
	}
	if err := urows.Err(); err != nil {
		return snap, err
	}
	urows.Close()
	planning, err := readPlanning(ctx, q, orgID, projectID, false)
	for _, p := range planning {
		snap.Planning = append(snap.Planning, p.PlanningValue)
	}
	return snap, err
}

func readParts(ctx context.Context, q rowQuerier, orgID, projectID string) ([]profile.Part, error) {
	rows, err := q.Query(ctx, `
SELECT id::text, label, kind, COALESCE(ncc_class, '') FROM project_parts
WHERE org_id = $1::uuid AND site_id = `+projectSiteSQL+` ORDER BY kind <> 'whole', created_at, label`, orgID, projectID)
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
// readKinds are the automatically read kinds; nil applies no kind filter.
func (s *Store) ReadProfile(ctx context.Context, orgID, projectID string, readKinds []string) (ProfileView, error) {
	var v ProfileView
	tx, err := s.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.RepeatableRead, AccessMode: pgx.ReadOnly})
	if err != nil {
		return v, err
	}
	defer tx.Rollback(ctx)
	return readProfileTx(ctx, tx, orgID, projectID, readKinds)
}

// Report assembly shares this read inside its own consistent snapshot.
func readProfileTx(ctx context.Context, tx pgx.Tx, orgID, projectID string, readKinds []string) (ProfileView, error) {

	var v ProfileView
	var err error
	// Resolve document IDs before reading jobs/decisions: stale statistics must
	// not turn the edit response into repeated project-wide scans (F31).
	var exists bool
	err = tx.QueryRow(ctx, `
SELECT EXISTS (SELECT 1 FROM projects WHERE org_id = $1::uuid AND id = $2::uuid),
       (SELECT built_at FROM profile_builds WHERE org_id = $1::uuid AND project_id = $2::uuid),
       COALESCE((SELECT thresholds_version FROM profile_builds WHERE org_id = $1::uuid AND project_id = $2::uuid), ''),
       (SELECT count(DISTINCT j.document_id) FROM jobs j
        WHERE j.org_id = $1::uuid AND j.document_id = ANY(ARRAY(SELECT id FROM documents WHERE org_id=$1::uuid AND project_id=$2::uuid)) AND j.status IN ('queued', 'leased')
          AND j.kind IN ('full_text', 'label', 'evidence')),
       (SELECT count(DISTINCT j.document_id) FROM jobs j
        WHERE j.org_id=$1::uuid AND j.document_id = ANY(ARRAY(SELECT id FROM documents WHERE org_id=$1::uuid AND project_id=$2::uuid)) AND j.status='leased' AND j.locked_until > now()
          AND j.kind IN ('full_text','label','evidence')),
       (SELECT count(*) FROM documents d WHERE d.org_id = $1::uuid AND d.project_id = $2::uuid AND `+readableSQL("$3")+`
          AND EXISTS (SELECT 1 FROM jobs f WHERE f.org_id = d.org_id AND f.document_id = d.id AND f.kind = 'full_text' AND f.status = 'done')
          AND NOT EXISTS (SELECT 1 FROM jobs l WHERE l.org_id = d.org_id AND l.document_id = d.id AND l.kind = 'label')),
       (SELECT count(DISTINCT j.document_id) FROM jobs j
        WHERE j.org_id=$1::uuid AND j.document_id = ANY(ARRAY(SELECT id FROM documents WHERE org_id=$1::uuid AND project_id=$2::uuid)) AND j.status='failed' AND j.kind IN ('full_text','label','evidence')),
       EXISTS (SELECT 1 FROM jobs j
        WHERE j.org_id=$1::uuid AND j.document_id = ANY(ARRAY(SELECT id FROM documents WHERE org_id=$1::uuid AND project_id=$2::uuid)) AND j.status='failed' AND j.kind IN ('label','evidence') AND j.last_error LIKE '%status 402%')`,
		orgID, projectID, nilIfEmpty(readKinds)).Scan(&exists, &v.BuiltAt, &v.ThresholdsVersion, &v.PendingDocuments, &v.ActiveDocuments, &v.UnreadDocuments, &v.FailedDocuments, &v.PaymentRequired)
	if err != nil {
		return v, err
	}
	if !exists {
		return v, ErrNotFound
	}
	if err = tx.QueryRow(ctx, `
WITH readable_documents AS MATERIALIZED (
 SELECT d.id, `+readableSQL("$3")+` AS readable
 FROM documents d WHERE d.org_id=$1::uuid AND d.project_id=$2::uuid
)
SELECT count(*) FILTER (WHERE readable), count(*) FILTER (WHERE NOT readable),
       COALESCE((SELECT k.value FROM decisions k WHERE k.org_id=$1::uuid AND k.field='kind'
                 AND k.document_id = ANY(ARRAY(SELECT id FROM readable_documents WHERE NOT readable))
                 GROUP BY k.value ORDER BY count(*) DESC, k.value LIMIT 1), '')
FROM readable_documents`,
		orgID, projectID, nilIfEmpty(readKinds)).Scan(&v.ReadDocuments, &v.SkippedDocuments, &v.SkippedKind); err != nil {
		return v, err
	}
	if err = readBuildState(ctx, tx, orgID, projectID, &v); err != nil {
		return v, err
	}
	if v.Coverage, err = sourceCoverage(ctx, tx, orgID, projectID); err != nil {
		return v, err
	}
	if v.Parts, err = readParts(ctx, tx, orgID, projectID); err != nil {
		return v, err
	}
	rows, err := tx.Query(ctx, `
SELECT part_id::text, key, value, band, assertion, note, tenders, sources, alternatives, derived,
       scope, origin, review_status, meaning, value_state, user_version
FROM profile_rows WHERE org_id = $1::uuid AND project_id = $2::uuid ORDER BY part_id, key`, orgID, projectID)
	if err != nil {
		return v, err
	}
	defer rows.Close()
	for rows.Next() {
		var r profile.Row
		var sources, alts, derived []byte
		if err := rows.Scan(&r.PartID, &r.Key, &r.Value, &r.Band, &r.Assertion, &r.Note, &r.Tenders, &sources, &alts, &derived,
			&r.Scope, &r.Origin, &r.ReviewStatus, &r.Meaning, &r.ValueState, &r.UserVersion); err != nil {
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

func hasAnyPrefix(s string, prefixes []string) bool {
	for _, p := range prefixes {
		if strings.HasPrefix(s, p) {
			return true
		}
	}
	return false
}

// RequestProfileRead remembers the request even while text is being prepared.
// The job claimer waits for extraction to finish before starting labels. Filing never
// queues reading itself: the user asks for it, so a profile update is a
// deliberate act and an upload costs no Jev calls beyond filing. Reading jobs
// already queued are restamped so a worker started with -background-backlog
// =false, which claims only jobs created after it started, picks them up.
// Only documents the profile reads are queued (readKinds; nil applies no
// kind filter); every document's text is still prepared.
func (s *Store) RequestProfileRead(ctx context.Context, orgID, projectID string, readKinds []string) (int64, error) {
	kinds := nilIfEmpty(readKinds)
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return 0, err
	}
	defer tx.Rollback(ctx)
	if err := lockProject(ctx, tx, orgID, projectID); err != nil {
		return 0, err
	}
	// Upgrade old extractions only on an explicit update. Never race a live reader.
	if _, err := tx.Exec(ctx, `WITH stale AS MATERIALIZED (
 SELECT d.id FROM documents d LEFT JOIN document_sources ds ON ds.org_id=d.org_id AND ds.document_id=d.id
 WHERE d.org_id=$1::uuid AND d.project_id=$2::uuid AND COALESCE(ds.version,'')<>$3
 AND EXISTS(SELECT 1 FROM passages p WHERE p.org_id=d.org_id AND p.document_id=d.id)
 AND NOT EXISTS(SELECT 1 FROM jobs j WHERE j.org_id=d.org_id AND j.document_id=d.id AND j.status IN ('queued','leased'))
 ) UPDATE jobs j SET status='queued',attempts=0,last_error='',run_after=now(),created_at=now()
 FROM documents d WHERE d.org_id=j.org_id AND d.id=j.document_id
 AND j.org_id=$1::uuid AND j.document_id IN (SELECT id FROM stale)
 AND (j.kind='full_text' OR (j.kind IN ('label','evidence') AND `+readableSQL("$4")+`))`, orgID, projectID, SourceVersion, kinds); err != nil {
		return 0, err
	}
	// An explicit click retries failed work; completed readings are retained.
	if _, err := tx.Exec(ctx, `UPDATE jobs j SET status='queued', attempts=0, last_error='', run_after=now(), created_at=now()
FROM documents d WHERE j.org_id=$1::uuid AND d.org_id=j.org_id AND d.id=j.document_id AND d.project_id=$2::uuid
 AND j.status='failed' AND (j.kind='full_text' OR (j.kind IN ('label','evidence') AND `+readableSQL("$3")+`))`, orgID, projectID, kinds); err != nil {
		return 0, err
	}
	if _, err := tx.Exec(ctx, `
UPDATE jobs j SET created_at = now()
FROM documents d
WHERE j.org_id = $1::uuid AND d.org_id = j.org_id AND d.id = j.document_id AND d.project_id = $2::uuid
  AND j.status = 'queued' AND (j.kind = 'full_text' OR (j.kind IN ('label', 'evidence') AND `+readableSQL("$3")+`))`, orgID, projectID, kinds); err != nil {
		return 0, err
	}
	tag, err := tx.Exec(ctx, `
INSERT INTO jobs (org_id, id, document_id, kind, status, priority)
SELECT d.org_id, gen_random_uuid(), d.id, 'label', 'queued', 0
FROM documents d
WHERE d.org_id = $1::uuid AND d.project_id = $2::uuid AND `+readableSQL("$3")+`
  AND EXISTS (SELECT 1 FROM jobs f WHERE f.org_id = d.org_id AND f.document_id = d.id AND f.kind = 'full_text' AND f.status IN ('queued', 'leased', 'done'))
ON CONFLICT (org_id, document_id, kind) DO NOTHING`, orgID, projectID, kinds)
	if err != nil {
		return 0, err
	}
	return tag.RowsAffected(), tx.Commit(ctx)
}

// SetScope records the user's scope choices on the whole project in one
// transaction: "in" or "out" per system key, or nil to remove the choice so
// defaults and documents decide again. A configured store rebuilds in the
// same transaction.
func (s *Store) SetScope(ctx context.Context, orgID, projectID, partID, userID string, choices map[string]*string) error {
	return s.setWorkScope(ctx, orgID, projectID, partID, userID, choices)
}

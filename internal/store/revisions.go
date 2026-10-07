package store

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"slices"
	"sort"

	"github.com/jackc/pgx/v5"
	"sitewise/internal/knowledge"
	"sitewise/internal/profile"
	"sitewise/internal/works"
)

// ProfileBuild is immutable process wiring. It contains no model caller.
type ProfileBuild struct {
	Catalog           *knowledge.Catalog
	KnowledgeVersion  string
	QuestionVersion   string
	ThresholdsVersion string
	ReadKinds         []string
	Compute           func(ProfileSnapshot) []profile.Row
	proposalEvaluator *works.Evaluator
	proposalError     error
}

// WithProfile returns a store sharing the pool, with code-only reconciliation
// attached to profile writes. It never mutates the original store's wiring.
func (s *Store) WithProfile(build ProfileBuild) *Store {
	copy := *s
	build.ReadKinds = append([]string(nil), build.ReadKinds...)
	sort.Strings(build.ReadKinds)
	if build.Catalog != nil {
		build.proposalEvaluator, build.proposalError = works.NewEvaluator(build.Catalog)
	}
	copy.profileBuild = &build
	return &copy
}

type Revisions struct {
	ProfileInputs int64 `json:"profile_inputs"`
	Works         int64 `json:"works"`
	Packages      int64 `json:"packages"`
	Delivery      int64 `json:"delivery"`
	Costs         int64 `json:"costs"`
	Reports       int64 `json:"reports"`
}

// BumpRevision belongs inside the authoritative write transaction. Future
// domain stores use this same bounded list; identifiers never come from SQL input.
func BumpRevision(ctx context.Context, tx pgx.Tx, orgID, projectID, domain string) error {
	switch domain {
	case "profile_inputs", "works", "packages", "delivery", "costs", "reports":
	default:
		return fmt.Errorf("unknown revision domain %q", domain)
	}
	if err := lockProject(ctx, tx, orgID, projectID); err != nil {
		return err
	}
	return bumpRevisionLocked(ctx, tx, orgID, projectID, domain)
}

// bumpRevisionLocked requires this transaction to hold the project advisory lock.
func bumpRevisionLocked(ctx context.Context, tx pgx.Tx, orgID, projectID, domain string) error {
	switch domain {
	case "profile_inputs", "works", "packages", "delivery", "costs", "reports":
	default:
		return fmt.Errorf("unknown revision domain %q", domain)
	}
	// Initialize and increment in one statement. A newly created counter keeps
	// the same version (2) as the former default-row insert followed by update.
	var revision int64
	err := tx.QueryRow(ctx, `INSERT INTO project_revisions (org_id,project_id,`+domain+`,version)
SELECT org_id,id,1,2 FROM projects WHERE org_id=$1::uuid AND id=$2::uuid
ON CONFLICT (org_id,project_id) DO UPDATE SET `+domain+`=project_revisions.`+domain+`+1,
version=project_revisions.version+1,updated_at=now() RETURNING `+domain,
		orgID, projectID).Scan(&revision)
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrNotFound
	}
	return err
}

func (s *Store) finishProfileWrite(ctx context.Context, tx pgx.Tx, orgID, projectID string) error {
	if err := lockProject(ctx, tx, orgID, projectID); err != nil {
		return err
	}
	return s.finishProfileWriteLocked(ctx, tx, orgID, projectID)
}

// finishProfileWriteLocked requires the caller to hold the project's advisory
// lock in this transaction, from before reading or changing authoritative input.
func (s *Store) finishProfileWriteLocked(ctx context.Context, tx pgx.Tx, orgID, projectID string) error {
	if err := bumpRevisionLocked(ctx, tx, orgID, projectID, "profile_inputs"); err != nil {
		return err
	}
	if b := s.profileBuild; b != nil {
		if b.proposalError != nil {
			return b.proposalError
		}
		if err := s.rebuildProfileProjectionLocked(ctx, tx, orgID, projectID, *b, b.proposalEvaluator); err != nil {
			return err
		}
	}
	return tx.Commit(ctx)
}

func lockDocumentProject(ctx context.Context, tx pgx.Tx, orgID, documentID string) (string, error) {
	var projectID string
	err := tx.QueryRow(ctx, `SELECT project_id::text FROM documents WHERE org_id=$1::uuid AND id=$2::uuid`, orgID, documentID).Scan(&projectID)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", ErrNotFound
	}
	if err != nil {
		return "", err
	}
	return projectID, lockProject(ctx, tx, orgID, projectID)
}

// Fingerprints exclude counters, timestamps and job status: a retry of the
// same input has the same fingerprint even though the build revision advances.
func profileFingerprint(snap ProfileSnapshot, b ProfileBuild) (string, error) {
	// Optimistic-lock versions advance on a repeated save too. They are not
	// changed evidence or values; the domain counters track those writes.
	snap.Site.Version = 0
	snap.User = append([]profile.UserValue(nil), snap.User...)
	for i := range snap.User {
		snap.User[i].Version = 0
	}
	snap.Planning = append([]profile.PlanningValue(nil), snap.Planning...)
	for i := range snap.Planning {
		snap.Planning[i].Version = 0
	}
	components := []any{snap.Parts, snap.Facts, snap.User, snap.Planning, snap.Documents, snap.WorkItems, b.ReadKinds}
	canonical := make([]any, 0, len(components)+5)
	// Version the representation: hash complete rows before sorting instead of
	// quoting all evidence text a second time inside a JSON string array.
	canonical = append(canonical, "profile-input-v2")
	for _, component := range components {
		rows, err := sortedFingerprintRows(component)
		if err != nil {
			return "", err
		}
		canonical = append(canonical, rows)
	}
	canonical = append(canonical, snap.Site, b.ThresholdsVersion, b.QuestionVersion, b.KnowledgeVersion)
	raw, err := json.Marshal(canonical)
	if err != nil {
		return "", err
	}
	hash := sha256.Sum256(raw)
	return hex.EncodeToString(hash[:]), nil
}

// Each digest covers the entire encoded row. Sorting retains input-order
// independence and duplicate multiplicity without retaining large quoted rows.
func sortedFingerprintRows(component any) ([]string, error) {
	rows := reflect.ValueOf(component)
	out := make([]string, rows.Len())
	for i := range out {
		raw, err := json.Marshal(rows.Index(i).Interface())
		if err != nil {
			return nil, err
		}
		hash := sha256.Sum256(raw)
		out[i] = hex.EncodeToString(hash[:])
	}
	sort.Strings(out)
	return out, nil
}

func readRevisions(ctx context.Context, tx pgx.Tx, orgID, projectID string) (Revisions, error) {
	var r Revisions
	err := tx.QueryRow(ctx, `SELECT profile_inputs, works, packages, delivery, costs, reports
FROM project_revisions WHERE org_id=$1::uuid AND project_id=$2::uuid`, orgID, projectID).
		Scan(&r.ProfileInputs, &r.Works, &r.Packages, &r.Delivery, &r.Costs, &r.Reports)
	if errors.Is(err, pgx.ErrNoRows) {
		return r, nil
	}
	return r, err
}

func readFingerprintInputs(ctx context.Context, tx pgx.Tx, orgID, projectID string, snap *ProfileSnapshot) error {
	err := tx.QueryRow(ctx, `SELECT s.id::text,s.label,s.address,s.lot,s.version FROM sites s
JOIN projects p ON p.org_id=s.org_id AND p.site_id=s.id WHERE p.org_id=$1::uuid AND p.id=$2::uuid`, orgID, projectID).
		Scan(&snap.Site.ID, &snap.Site.Label, &snap.Site.Address, &snap.Site.Lot, &snap.Site.Version)
	if err != nil {
		return err
	}
	if snap.Documents != nil {
		return nil // readSnapshot already read every document in this transaction.
	}
	rows, err := tx.Query(ctx, `SELECT jsonb_build_object('id',d.id,'read',d.profile_read,
'filename',d.filename,'number',d.document_number,'revision',d.revision,
'kind',COALESCE((SELECT value FROM decisions WHERE org_id=d.org_id AND document_id=d.id AND field='kind'),''),
'superseded',EXISTS(SELECT 1 FROM supersessions WHERE org_id=d.org_id AND prior_document_id=d.id),
'sha256',(SELECT encode(sha256,'hex') FROM files WHERE org_id=d.org_id AND id=d.file_id))
FROM documents d WHERE d.org_id=$1::uuid AND d.project_id=$2::uuid ORDER BY d.id`, orgID, projectID)
	if err != nil {
		return err
	}
	defer rows.Close()
	for rows.Next() {
		var raw json.RawMessage
		if err := rows.Scan(&raw); err != nil {
			return err
		}
		snap.Documents = append(snap.Documents, raw)
	}
	return rows.Err()
}

func readBuildState(ctx context.Context, tx pgx.Tx, orgID, projectID string, v *ProfileView) error {
	var inputs []byte
	err := tx.QueryRow(ctx, `SELECT COALESCE(b.revision,0),COALESCE(b.input_fingerprint,''),COALESCE(b.knowledge_version,''),COALESCE(b.question_version,''),b.inputs,
COALESCE(r.profile_inputs,0),COALESCE(r.works,0),COALESCE(r.packages,0),COALESCE(r.delivery,0),COALESCE(r.costs,0),COALESCE(r.reports,0)
FROM (SELECT 1) singleton
LEFT JOIN profile_builds b ON b.org_id=$1::uuid AND b.project_id=$2::uuid
LEFT JOIN project_revisions r ON r.org_id=$1::uuid AND r.project_id=$2::uuid`, orgID, projectID).Scan(&v.Revision, &v.InputFingerprint, &v.KnowledgeVersion, &v.QuestionVersion, &inputs,
		&v.CurrentInputs.ProfileInputs, &v.CurrentInputs.Works, &v.CurrentInputs.Packages, &v.CurrentInputs.Delivery, &v.CurrentInputs.Costs, &v.CurrentInputs.Reports)
	if err != nil {
		return err
	}
	if len(inputs) > 0 {
		var policy struct {
			ReadKinds []string `json:"read_kinds"`
		}
		if err := json.Unmarshal(inputs, &policy); err != nil {
			return err
		}
		v.ReadKinds = policy.ReadKinds
		if err := json.Unmarshal(inputs, &v.Inputs); err != nil {
			return err
		}
	}
	return nil
}

// StaleFor names explicit dependencies only. Report kinds add their own
// dependency lists when their assemblers exist (WP-41).
func (v ProfileView) StaleFor(b ProfileBuild) []string {
	stale := []string{}
	if !slices.Equal(v.ReadKinds, b.ReadKinds) {
		stale = append(stale, "reading_policy")
	}
	if v.Revision == 0 {
		stale = append(stale, "not_built")
	}
	if v.Inputs.ProfileInputs != v.CurrentInputs.ProfileInputs {
		stale = append(stale, "profile_inputs")
	}
	if v.Inputs.Works != v.CurrentInputs.Works {
		stale = append(stale, "works")
	}
	if v.KnowledgeVersion != b.KnowledgeVersion {
		stale = append(stale, "knowledge_version")
	}
	if v.QuestionVersion != b.QuestionVersion {
		stale = append(stale, "question_version")
	}
	if v.ThresholdsVersion != b.ThresholdsVersion {
		stale = append(stale, "thresholds_version")
	}
	return stale
}

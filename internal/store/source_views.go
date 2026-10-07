package store

import (
	"context"

	"github.com/jackc/pgx/v5"
)

type SourceCoverage struct {
	DocumentID   string `json:"document_id"`
	Filename     string `json:"filename"`
	Pages        int    `json:"pages"`
	EmptyPages   []int  `json:"empty_pages"`
	Current      bool   `json:"current"`
	Units        int    `json:"units"`
	Labelled     int    `json:"labelled"`
	Evidence     int    `json:"evidence"`
	NeedsMapping int    `json:"needs_mapping"`
	Mapped       int    `json:"mapped"`
	Background   int    `json:"background"`
}

func (s *Store) SourceCoverage(ctx context.Context, org, project string) ([]SourceCoverage, error) {
	return sourceCoverage(ctx, s.pool, org, project)
}

func sourceCoverage(ctx context.Context, q rowQuerier, org, project string) ([]SourceCoverage, error) {
	// Batch the passage IDs per document so outcome/evidence counts need one
	// lookup per document rather than a correlated lookup per passage. Replan
	// for the current upload size instead of retaining an empty-project plan.
	rows, err := q.Query(ctx, `SELECT d.id::text,d.filename,COALESCE(ds.pages,0),COALESCE(ds.empty_pages,'{}'),COALESCE(ds.version=$3,false),
 p.units,c.labelled,e.evidence,c.needs_mapping,c.mapped,c.background
 FROM documents d
 LEFT JOIN LATERAL (
 SELECT pages,empty_pages,version FROM document_sources WHERE org_id=d.org_id AND document_id=d.id OFFSET 0
 ) ds ON true
 CROSS JOIN LATERAL (
 SELECT count(*) AS units,array_agg(id) AS ids FROM passages WHERE org_id=d.org_id AND document_id=d.id
 ) p
 CROSS JOIN LATERAL (
 SELECT count(*) FILTER(WHERE outcome<>'pending') AS labelled,
 count(*) FILTER(WHERE outcome='needs_mapping') AS needs_mapping,
 count(*) FILTER(WHERE outcome='mapped') AS mapped,count(*) FILTER(WHERE outcome='background') AS background
 FROM passage_sources WHERE org_id=d.org_id AND passage_id=ANY(p.ids) AND outcome<>'pending'
 ) c
 CROSS JOIN LATERAL (
 SELECT count(*) AS evidence FROM passage_calls WHERE org_id=d.org_id AND passage_id=ANY(p.ids) AND stage='evidence'
 ) e
 WHERE d.org_id=$1::uuid AND d.project_id=$2::uuid
 AND NOT EXISTS(SELECT 1 FROM supersessions ss WHERE ss.org_id=d.org_id AND ss.prior_document_id=d.id)
 ORDER BY d.filename,d.id`, pgx.QueryExecModeExec, org, project, SourceVersion)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []SourceCoverage{}
	for rows.Next() {
		var c SourceCoverage
		if err = rows.Scan(&c.DocumentID, &c.Filename, &c.Pages, &c.EmptyPages, &c.Current, &c.Units, &c.Labelled, &c.Evidence, &c.NeedsMapping, &c.Mapped, &c.Background); err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	return out, rows.Err()
}

type SourceRecord struct {
	ID         string   `json:"id"`
	DocumentID string   `json:"document_id"`
	Filename   string   `json:"filename"`
	Ordinal    int      `json:"ordinal"`
	Text       string   `json:"text"`
	Page       int      `json:"page"`
	Location   string   `json:"location"`
	Section    string   `json:"section"`
	Context    string   `json:"context"`
	Category   string   `json:"category"`
	Provider   string   `json:"provider"`
	Scope      string   `json:"scope"`
	Outcome    string   `json:"outcome"`
	Confidence *float64 `json:"confidence,omitempty"`
	Keys       []string `json:"keys"`
	Unresolved []string `json:"unresolved"`
	Systems    []string `json:"systems"`
}
type SourceRecords struct {
	Records []SourceRecord `json:"records"`
	More    bool           `json:"more"`
}

// SourceRecords is bounded, scoped and stable in document/ordinal order.
// Needs mapping includes unfinished units so failures never hide source text.
func (s *Store) SourceRecords(ctx context.Context, org, project, system, outcome string, offset int) (SourceRecords, error) {
	out := SourceRecords{Records: []SourceRecord{}}
	var exists bool
	if err := s.pool.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM projects WHERE org_id=$1::uuid AND id=$2::uuid)`, org, project).Scan(&exists); err != nil {
		return out, err
	}
	if !exists {
		return out, ErrNotFound
	}
	rows, err := s.pool.Query(ctx, `SELECT p.id::text,d.id::text,d.filename,p.ordinal,p.body,
 COALESCE(ps.page,0),COALESCE(ps.location,''),COALESCE(ps.section,''),COALESCE(ps.context,''),COALESCE(ps.category,'unresolved'),COALESCE(ps.provider,''),COALESCE(ps.scope,''),COALESCE(ps.outcome,'pending'),ps.confidence,COALESCE(ps.mapped_keys,'{}'),COALESCE(ps.unresolved,'{}'),
 ARRAY(SELECT system_id FROM passage_systems ss WHERE ss.org_id=p.org_id AND ss.passage_id=p.id ORDER BY system_id)
 FROM passages p JOIN documents d ON d.org_id=p.org_id AND d.id=p.document_id
 LEFT JOIN passage_sources ps ON ps.org_id=p.org_id AND ps.passage_id=p.id
 WHERE d.org_id=$1::uuid AND d.project_id=$2::uuid
 AND NOT EXISTS(SELECT 1 FROM supersessions ss WHERE ss.org_id=d.org_id AND ss.prior_document_id=d.id)
 AND ($3='' OR EXISTS(SELECT 1 FROM passage_systems ss WHERE ss.org_id=p.org_id AND ss.passage_id=p.id AND ss.system_id=$3))
 AND ($4='' OR ($4='needs_mapping' AND COALESCE(ps.outcome,'pending') IN ('needs_mapping','pending')) OR ps.outcome=$4)
 ORDER BY d.id,p.ordinal LIMIT 51 OFFSET $5`, org, project, system, outcome, offset)
	if err != nil {
		return out, err
	}
	defer rows.Close()
	for rows.Next() {
		var r SourceRecord
		if err = rows.Scan(&r.ID, &r.DocumentID, &r.Filename, &r.Ordinal, &r.Text, &r.Page, &r.Location, &r.Section, &r.Context, &r.Category, &r.Provider, &r.Scope, &r.Outcome, &r.Confidence, &r.Keys, &r.Unresolved, &r.Systems); err != nil {
			return out, err
		}
		out.Records = append(out.Records, r)
	}
	if len(out.Records) > 50 {
		out.More = true
		out.Records = out.Records[:50]
	}
	return out, rows.Err()
}

// Merge evidence results without discarding low-confidence or missing answers.
func (s *Store) MergeSourceEvidence(ctx context.Context, org, id string, keys, unresolved []string) error {
	if keys == nil {
		keys = []string{}
	}
	if unresolved == nil {
		unresolved = []string{}
	}
	_, err := s.pool.Exec(ctx, `UPDATE passage_sources SET
 mapped_keys=ARRAY(SELECT DISTINCT unnest(mapped_keys || $3::text[])),
 unresolved=ARRAY(SELECT DISTINCT unnest(unresolved || $4::text[])),
 outcome=CASE WHEN cardinality(unresolved || $4::text[])>0 THEN 'needs_mapping' WHEN category NOT IN ('reference','background','unresolved') AND cardinality(mapped_keys || $3::text[])>0 THEN 'mapped' ELSE outcome END
 WHERE org_id=$1::uuid AND passage_id=$2::uuid`, org, id, keys, unresolved)
	return err
}

// Extend the active lease while a long document is being processed.
func (s *Store) RenewBackgroundLease(ctx context.Context, job ClaimedJob, seconds float64) error {
	tag, err := s.pool.Exec(ctx, `UPDATE jobs SET locked_until=now()+make_interval(secs=>$4) WHERE org_id=$1::uuid AND id=$2::uuid AND lease_token=$3::uuid AND status='leased'`, job.OrgID, job.ID, job.Token, seconds)
	if err == nil && tag.RowsAffected() == 0 {
		return ErrLeaseLost
	}
	return err
}

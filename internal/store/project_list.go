package store

import (
	"context"
	"time"
)

// ProjectSummary is one row of the Projects page: enough to pick the job
// that needs you without opening it. Every figure is a count or a date the
// database already holds; nothing is judged here.
type ProjectSummary struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	// Address is the typed site address, else the profile's reading of it;
	// empty when neither is known.
	Address string `json:"address"`
	// Documents counts filings still in the register (not void, superseded
	// or replaced by their split sheets).
	Documents int `json:"documents"`
	// SourcesDiffer counts profile values whose sources disagree (red band).
	SourcesDiffer int       `json:"sources_differ"`
	CreatedAt     time.Time `json:"created_at"`
	// LastFiled is the newest filing, nil before the first one.
	LastFiled *time.Time `json:"last_filed,omitempty"`
	Archived  bool       `json:"archived"`
}

// ProjectSummaries lists orgID's projects, most recent activity first: the
// newest filing, or creation when nothing is filed yet. One query; each
// aggregate reads one project's rows through its (org_id, project_id) index.
func (s *Store) ProjectSummaries(ctx context.Context, orgID string) ([]ProjectSummary, error) {
	rows, err := s.pool.Query(ctx, `
SELECT p.id::text, p.name, COALESCE(NULLIF(s.address, ''), r.address, ''),
       d.n, r.differ, p.created_at, d.last, p.archived_at IS NOT NULL
FROM projects p
JOIN sites s ON s.org_id = p.org_id AND s.id = p.site_id
CROSS JOIN LATERAL (
  SELECT count(*) FILTER (WHERE status NOT IN ('void', 'superseded', 'split'))::int AS n,
         max(created_at) AS last
  FROM documents WHERE org_id = p.org_id AND project_id = p.id
) d
CROSS JOIN LATERAL (
  SELECT count(*) FILTER (WHERE band = 'red')::int AS differ,
         max(value) FILTER (WHERE key = 'site.address' AND value <> '') AS address
  FROM profile_rows WHERE org_id = p.org_id AND project_id = p.id
) r
WHERE p.org_id = $1::uuid
ORDER BY GREATEST(p.created_at, COALESCE(d.last, p.created_at)) DESC, p.id`, orgID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []ProjectSummary{}
	for rows.Next() {
		var p ProjectSummary
		if err := rows.Scan(&p.ID, &p.Name, &p.Address, &p.Documents, &p.SourcesDiffer, &p.CreatedAt, &p.LastFiled, &p.Archived); err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	return out, rows.Err()
}

// SetProjectArchived puts a project away or brings it back. Nothing in the
// project changes. Another org's project is ErrNotFound.
func (s *Store) SetProjectArchived(ctx context.Context, orgID, projectID string, archived bool) error {
	result, err := s.pool.Exec(ctx, `
UPDATE projects SET archived_at = CASE WHEN $3 THEN COALESCE(archived_at, now()) END
WHERE org_id = $1::uuid AND id = $2::uuid`, orgID, projectID, archived)
	if err != nil {
		return err
	}
	if result.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

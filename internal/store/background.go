package store

import "context"

// OrgsWithBackgroundJobs lists orgs that have queued passage, label or
// evidence jobs due now. It is the only cross-org read the worker makes, and
// it returns ids only.
func (s *Store) OrgsWithBackgroundJobs(ctx context.Context) ([]string, error) {
	rows, err := s.pool.Query(ctx, `
SELECT DISTINCT org_id::text FROM jobs
WHERE status = 'queued' AND run_after <= now() AND kind = ANY($1)
ORDER BY 1`, []string{JobKindFullText, JobKindLabel, JobKindEvidence})
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		out = append(out, id)
	}
	return out, rows.Err()
}

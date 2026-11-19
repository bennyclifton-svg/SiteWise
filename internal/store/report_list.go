package store

import (
	"context"
	"encoding/json"
)

// ListReports lists the project's reports. A payment schedule is reached
// through its claim on the Cost Plan, never as a package's report.
func (s *Store) ListReports(ctx context.Context, org, project string) ([]Report, error) {
	if _, err := s.GetProject(ctx, org, project); err != nil {
		return nil, err
	}
	rows, err := s.pool.Query(ctx, `SELECT to_jsonb(r) FROM reports r WHERE org_id=$1::uuid AND project_id=$2::uuid AND kind <> 'payment_schedule' ORDER BY title,id`, org, project)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Report{}
	for rows.Next() {
		var raw []byte
		if err := rows.Scan(&raw); err != nil {
			return nil, err
		}
		var r Report
		if err := json.Unmarshal(raw, &r); err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

package store

import (
	"context"
)

// OrgCounts is one org's row counts, compared before and after a restore.
type OrgCounts struct {
	OrgID       string `json:"org_id"`
	Users       int64  `json:"users"`
	Memberships int64  `json:"memberships"`
	Projects    int64  `json:"projects"`
	Files       int64  `json:"files"`
	Documents   int64  `json:"documents"`
	Decisions   int64  `json:"decisions"`
	Passages    int64  `json:"passages"`
	Events      int64  `json:"events"`
}

// CrossOrg is the number of rows whose parent is not in the row's own org.
type CrossOrg struct {
	Reference string `json:"reference"`
	Rows      int64  `json:"rows"`
}

// RestoreFacts is what a restore rehearsal compares and checks. It holds
// counts and catalog state only, never row content.
type RestoreFacts struct {
	Orgs        []OrgCounts `json:"orgs"`
	ForeignKeys int64       `json:"foreign_keys"`
	Unvalidated int64       `json:"unvalidated_foreign_keys"`
	Migrations  []string    `json:"migrations"`
	CrossOrg    []CrossOrg  `json:"cross_org"`
}

// RestoreFacts reads the whole database. It is an operator check run on the
// host, not a tenant read, and is never served over HTTP.
func (s *Store) RestoreFacts(ctx context.Context) (RestoreFacts, error) {
	var f RestoreFacts
	orgs, err := s.q.OrgCounts(ctx)
	if err != nil {
		return f, err
	}
	for _, o := range orgs {
		f.Orgs = append(f.Orgs, OrgCounts(o))
	}
	fk, err := s.q.ForeignKeyState(ctx)
	if err != nil {
		return f, err
	}
	f.ForeignKeys, f.Unvalidated = fk.Total, fk.Unvalidated
	cross, err := s.q.CrossOrgRows(ctx)
	if err != nil {
		return f, err
	}
	for _, c := range cross {
		f.CrossOrg = append(f.CrossOrg, CrossOrg(c))
	}
	// schema_migrations is created by the migrator, outside the sqlc schema.
	rows, err := s.pool.Query(ctx, `SELECT version FROM schema_migrations ORDER BY version`)
	if err != nil {
		return f, err
	}
	defer rows.Close()
	for rows.Next() {
		var v string
		if err := rows.Scan(&v); err != nil {
			return f, err
		}
		f.Migrations = append(f.Migrations, v)
	}
	return f, rows.Err()
}

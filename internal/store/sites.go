package store

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
)

// ErrVersionConflict means the caller edited from a stale version. The
// current record is returned with it so the client can show what changed.
var ErrVersionConflict = errors.New("version conflict")

// Site is the lasting record of one address or campus. A project is an
// intervention on one site; version 1 has one project per site (migration 011).
type Site struct {
	ID      string `json:"id"`
	Label   string `json:"label"`
	Address string `json:"address"`
	Lot     string `json:"lot"`
	Version int64  `json:"version"`
}

// ProjectSite returns the site of a project in the org.
func (s *Store) ProjectSite(ctx context.Context, orgID, projectID string) (Site, error) {
	var site Site
	err := s.pool.QueryRow(ctx, `
SELECT s.id::text, s.label, s.address, s.lot, s.version
FROM projects p JOIN sites s ON s.org_id = p.org_id AND s.id = p.site_id
WHERE p.org_id = $1::uuid AND p.id = $2::uuid`, orgID, projectID).
		Scan(&site.ID, &site.Label, &site.Address, &site.Lot, &site.Version)
	if errors.Is(err, pgx.ErrNoRows) {
		return Site{}, ErrNotFound
	}
	return site, err
}

// UpdateSite changes a site's label, address or lot when version is the
// stored version. Nil leaves a field. A stale version returns the current
// site with ErrVersionConflict; another org's site is ErrNotFound.
func (s *Store) UpdateSite(ctx context.Context, orgID, siteID string, version int64, label, address, lot *string) (Site, error) {
	var site Site
	err := s.pool.QueryRow(ctx, `
UPDATE sites SET
  label = COALESCE($4, label),
  address = COALESCE($5, address),
  lot = COALESCE($6, lot),
  version = version + 1,
  updated_at = now()
WHERE org_id = $1::uuid AND id = $2::uuid AND version = $3
RETURNING id::text, label, address, lot, version`, orgID, siteID, version, label, address, lot).
		Scan(&site.ID, &site.Label, &site.Address, &site.Lot, &site.Version)
	if err == nil {
		return site, nil
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return Site{}, err
	}
	err = s.pool.QueryRow(ctx, `
SELECT id::text, label, address, lot, version FROM sites WHERE org_id = $1::uuid AND id = $2::uuid`, orgID, siteID).
		Scan(&site.ID, &site.Label, &site.Address, &site.Lot, &site.Version)
	if errors.Is(err, pgx.ErrNoRows) {
		return Site{}, ErrNotFound
	}
	if err != nil {
		return Site{}, err
	}
	return site, ErrVersionConflict
}

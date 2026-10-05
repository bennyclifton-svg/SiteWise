package store

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"

	"sitewise/internal/profile"
)

// PlanningValue is one stored planning value: the reconciled shape plus
// its row id, owner and when it last changed (superseded or created).
type PlanningValue struct {
	profile.PlanningValue
	ID, Scope string
	UpdatedAt time.Time
}

// PlanningWrite records one planning value. Kind is the registry type
// (integer, number, boolean, choice or text) and decides the typed column.
type PlanningWrite struct {
	PartID, Key, Scope, Kind string
	State                    string
	Value                    *string
	RangeLow, RangeHigh      *float64
	Unit                     string
	Origin, Meaning          string
	Rationale, Limitations   string
	// Version is the live version the client edited from (0 for none).
	Version int64
}

// planningOwner is planningSQL's owner filter: $1 org, $2 the site or the
// project, $3 part, $4 key, live rows only.
func planningOwner(scope string) (string, error) {
	switch scope {
	case "site":
		return `org_id = $1::uuid AND scope = 'site' AND site_id = $2::uuid AND part_id = $3::uuid AND key = $4 AND review_status <> 'superseded'`, nil
	case "project":
		return `org_id = $1::uuid AND scope = 'project' AND project_id = $2::uuid AND part_id = $3::uuid AND key = $4 AND review_status <> 'superseded'`, nil
	}
	return "", fmt.Errorf("planning value scope %q", scope)
}

// livePlanning locks and returns the live row's id and version ("", 0 when
// none) and the highest version the key has had, so history stays ordered.
func livePlanning(ctx context.Context, tx pgx.Tx, orgID, projectID, siteID, partID, key, scope string) (string, int64, int64, error) {
	where, err := planningOwner(scope)
	if err != nil {
		return "", 0, 0, err
	}
	args := ownerArgs(scope, orgID, projectID, siteID, partID, key)
	var id string
	var version int64
	err = tx.QueryRow(ctx, `SELECT id::text, version FROM profile_planning_values WHERE `+where+` FOR UPDATE`, args...).Scan(&id, &version)
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return "", 0, 0, err
	}
	owner, _ := ownerSQL(scope)
	var latest int64
	err = tx.QueryRow(ctx, `SELECT COALESCE(max(version), 0) FROM profile_planning_values WHERE `+owner, args...).Scan(&latest)
	return id, version, latest, err
}

// SetPlanningValue supersedes the live value of a key, if any, with a new
// row and returns its version. Superseded rows are kept as history. A stale
// expected version returns the live version with ErrVersionConflict. The
// caller rebuilds the profile.
func (s *Store) SetPlanningValue(ctx context.Context, orgID, projectID, userID string, w PlanningWrite) (int64, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return 0, err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if err := lockProject(ctx, tx, orgID, projectID); err != nil {
		return 0, err
	}
	siteID, err := userValueOwner(ctx, tx, orgID, projectID, w.PartID)
	if err != nil {
		return 0, err
	}
	prior, current, latest, err := livePlanning(ctx, tx, orgID, projectID, siteID, w.PartID, w.Key, w.Scope)
	if err != nil {
		return 0, err
	}
	if w.Version != current {
		return current, ErrVersionConflict
	}
	if err := retireOtherScope(ctx, tx, orgID, projectID, siteID, w.PartID, w.Key, w.Scope); err != nil {
		return 0, err
	}
	var text, numeric, boolean any
	if w.State == "set" && w.Value != nil {
		switch w.Kind {
		case "integer", "number":
			numeric = *w.Value
		case "boolean":
			boolean = *w.Value == "true"
		default:
			text = *w.Value
		}
	}
	var project any = projectID
	if w.Scope == "site" {
		project = nil
	}
	// A user's own planning value is accepted for planning; code proposes.
	review := "accepted_for_planning"
	if w.Origin == "calculation" {
		review = "proposed"
	}
	// Retire the live row before inserting its successor: one live row per key.
	if prior != "" {
		if err := supersede(ctx, tx, orgID, prior, nil); err != nil {
			return 0, err
		}
	}
	var id string
	err = tx.QueryRow(ctx, `
INSERT INTO profile_planning_values (org_id, site_id, project_id, scope, part_id, key, value_state,
  value_text, value_numeric, value_bool, range_low, range_high, unit, origin, review_status, meaning,
  rationale, limitations, user_id, version)
VALUES ($1::uuid, $2::uuid, $3::uuid, $4, $5::uuid, $6, $7, $8::text, $9::numeric, $10::boolean, $11, $12, $13,
  $14, $15, $16, $17, $18, $19::uuid, $20)
RETURNING id::text`,
		orgID, siteID, project, w.Scope, w.PartID, w.Key, w.State, text, numeric, boolean, w.RangeLow, w.RangeHigh,
		w.Unit, w.Origin, review, w.Meaning, w.Rationale, w.Limitations, nilIfBlank(userID), latest+1).Scan(&id)
	if err != nil {
		return 0, err
	}
	if prior != "" {
		if _, err := tx.Exec(ctx, `UPDATE profile_planning_values SET superseded_by = $3::uuid
WHERE org_id = $1::uuid AND id = $2::uuid`, orgID, prior, id); err != nil {
			return 0, err
		}
	}
	return latest + 1, tx.Commit(ctx)
}

// retireOtherScope supersedes a live value the registry has since moved to
// the other scope, so a key keeps one live value (as SetUserValue does).
func retireOtherScope(ctx context.Context, tx pgx.Tx, orgID, projectID, siteID, partID, key, scope string) error {
	other := "project"
	if scope == "project" {
		other = "site"
	}
	where, err := planningOwner(other)
	if err != nil {
		return err
	}
	_, err = tx.Exec(ctx, `UPDATE profile_planning_values SET review_status = 'superseded', superseded_at = now() WHERE `+where,
		ownerArgs(other, orgID, projectID, siteID, partID, key)...)
	return err
}

// supersede retires a live row; next is the row that replaced it, if any.
func supersede(ctx context.Context, tx pgx.Tx, orgID, id string, next any) error {
	_, err := tx.Exec(ctx, `UPDATE profile_planning_values
SET review_status = 'superseded', superseded_at = now(), superseded_by = $3::uuid
WHERE org_id = $1::uuid AND id = $2::uuid`, orgID, id, next)
	return err
}

// WithdrawPlanningValue supersedes the live value of a key with nothing; it
// is kept as history, never deleted. It returns ErrNotFound when the key
// has no live value and ErrVersionConflict, with the live version, when the
// expected version is stale.
func (s *Store) WithdrawPlanningValue(ctx context.Context, orgID, projectID, partID, key, scope string, version int64) (int64, error) {
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
	prior, current, _, err := livePlanning(ctx, tx, orgID, projectID, siteID, partID, key, scope)
	if err != nil {
		return 0, err
	}
	if prior == "" {
		return 0, ErrNotFound
	}
	if version != current {
		return current, ErrVersionConflict
	}
	if err := supersede(ctx, tx, orgID, prior, nil); err != nil {
		return 0, err
	}
	return 0, tx.Commit(ctx)
}

// PlanningValues lists a project's planning values: its own and its site's.
// History (superseded rows) is included only when history is true.
func (s *Store) PlanningValues(ctx context.Context, orgID, projectID string, history bool) ([]PlanningValue, error) {
	return readPlanning(ctx, s.pool, orgID, projectID, history)
}

func readPlanning(ctx context.Context, q rowQuerier, orgID, projectID string, history bool) ([]PlanningValue, error) {
	rows, err := q.Query(ctx, `
SELECT id::text, part_id::text, key, scope, value_state,
       COALESCE(value_text, trim_scale(value_numeric)::text, value_bool::text),
       range_low::float8, range_high::float8, unit, origin, review_status, meaning, rationale, limitations,
       version, COALESCE(superseded_at, created_at)
FROM profile_planning_values
WHERE org_id = $1::uuid AND ($3 OR review_status <> 'superseded')
  AND ((scope = 'project' AND project_id = $2::uuid) OR (scope = 'site' AND site_id = `+projectSiteSQL+`))
ORDER BY part_id, key, version`, orgID, projectID, history)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []PlanningValue
	for rows.Next() {
		var p PlanningValue
		if err := rows.Scan(&p.ID, &p.PartID, &p.Key, &p.Scope, &p.State, &p.Value, &p.RangeLow, &p.RangeHigh,
			&p.Unit, &p.Origin, &p.ReviewStatus, &p.Meaning, &p.Rationale, &p.Limitations, &p.Version, &p.UpdatedAt); err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	return out, rows.Err()
}

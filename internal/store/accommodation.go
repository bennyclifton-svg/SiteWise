package store

import (
	"context"
	"fmt"
	"regexp"
	"slices"
	"strconv"
	"strings"

	"github.com/jackc/pgx/v5"

	"sitewise/internal/profile"
)

// The Accommodation schedule's edits
// (docs/plans/2026-11-11-accommodation-schedule-lines.md). A line is named
// by its generated key ("offices#3") or, for a line the user added, its row
// id. Every write locks the project and is scoped to the org; no Jev.

// lineKeyPattern is a generated line's key, as the table's check states it.
var lineKeyPattern = regexp.MustCompile(`^[a-z][a-z0-9_]*#[1-9][0-9]{0,2}$`)

// accommodationColumns maps a field the user edits to its column.
var accommodationColumns = map[string]string{"name": "name", "level": "level", "area": "area_sqm", "comment": "comment"}

// AccommodationLines reads the user's edits to a project's schedule, oldest
// first.
func (s *Store) AccommodationLines(ctx context.Context, org, project string) ([]profile.StoredLine, error) {
	if !validUUID(org) || !validUUID(project) {
		return nil, ErrNotFound
	}
	rows, err := s.pool.Query(ctx, `SELECT id::text, coalesce(line_key,''), name, level, coalesce(area_sqm::text,''), comment, set_fields, position, split, removed
FROM accommodation_lines WHERE org_id=$1::uuid AND project_id=$2::uuid ORDER BY created_at, id`, org, project)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []profile.StoredLine
	for rows.Next() {
		var l profile.StoredLine
		if err := rows.Scan(&l.ID, &l.Key, &l.Name, &l.Level, &l.Area, &l.Comment, &l.Set, &l.Position, &l.Split, &l.Removed); err != nil {
			return nil, err
		}
		out = append(out, l)
	}
	return out, rows.Err()
}

// SetAccommodationField records the user's value for one field on each line.
func (s *Store) SetAccommodationField(ctx context.Context, org, project string, refs []string, field, value string) error {
	col, ok := accommodationColumns[field]
	if !ok {
		return ErrInvalidInput
	}
	ids, keys, err := lineRefs(refs)
	if err != nil {
		return err
	}
	param := "$4::text"
	if field == "area" {
		param = "NULLIF($4,'')::numeric"
	}
	merged := `(SELECT array_agg(DISTINCT f ORDER BY f) FROM unnest(accommodation_lines.set_fields || ARRAY[$3::text]) f)`
	return s.accommodationTx(ctx, org, project, func(tx pgx.Tx) error {
		if len(keys) > 0 {
			if _, err := tx.Exec(ctx, fmt.Sprintf(`INSERT INTO accommodation_lines (org_id, project_id, line_key, %[1]s, set_fields)
SELECT $1::uuid, $2::uuid, k, %[2]s, ARRAY[$3::text] FROM unnest($5::text[]) k
ON CONFLICT (org_id, project_id, line_key) DO UPDATE SET %[1]s = excluded.%[1]s, set_fields = %[3]s, updated_at = now()`, col, param, merged),
				org, project, field, value, keys); err != nil {
				return err
			}
		}
		if len(ids) == 0 {
			return nil
		}
		tag, err := tx.Exec(ctx, fmt.Sprintf(`UPDATE accommodation_lines SET %[1]s = %[2]s, set_fields = %[3]s, updated_at = now()
WHERE org_id=$1::uuid AND project_id=$2::uuid AND id = ANY($5::uuid[]) AND line_key IS NULL`, col, param, merged),
			org, project, field, value, ids)
		return foundAll(tag.RowsAffected(), len(ids), err)
	})
}

// ResetAccommodationLines hands generated lines back to the documents: the
// user's fields and removal go; the line keeps its place. A line the user
// added has nothing to go back to and is left as it is.
func (s *Store) ResetAccommodationLines(ctx context.Context, org, project string, refs []string) error {
	_, keys, err := lineRefs(refs)
	if err != nil {
		return err
	}
	return s.accommodationTx(ctx, org, project, func(tx pgx.Tx) error {
		_, err := tx.Exec(ctx, `UPDATE accommodation_lines SET name='', level='', area_sqm=NULL, comment='', set_fields='{}', removed=false, updated_at=now()
WHERE org_id=$1::uuid AND project_id=$2::uuid AND line_key = ANY($3::text[])`, org, project, keys)
		return err
	})
}

// RemoveAccommodationLines takes lines out: a generated line is marked
// removed so the next reading does not bring it back; a line the user
// added is deleted.
func (s *Store) RemoveAccommodationLines(ctx context.Context, org, project string, refs []string) error {
	ids, keys, err := lineRefs(refs)
	if err != nil {
		return err
	}
	return s.accommodationTx(ctx, org, project, func(tx pgx.Tx) error {
		if len(keys) > 0 {
			if _, err := tx.Exec(ctx, `INSERT INTO accommodation_lines (org_id, project_id, line_key, removed)
SELECT $1::uuid, $2::uuid, k, true FROM unnest($3::text[]) k
ON CONFLICT (org_id, project_id, line_key) DO UPDATE SET removed = true, updated_at = now()`, org, project, keys); err != nil {
				return err
			}
		}
		if len(ids) == 0 {
			return nil
		}
		tag, err := tx.Exec(ctx, `DELETE FROM accommodation_lines WHERE org_id=$1::uuid AND project_id=$2::uuid AND id = ANY($3::uuid[]) AND line_key IS NULL`, org, project, ids)
		return foundAll(tag.RowsAffected(), len(ids), err)
	})
}

// SplitAccommodationLine turns one line that stands for many rooms into a
// numbered line per room. Only a space's first line splits.
func (s *Store) SplitAccommodationLine(ctx context.Context, org, project, key string) error {
	if _, n, ok := profile.ParseLineKey(key); !ok || n != 1 || !lineKeyPattern.MatchString(key) {
		return ErrInvalidInput
	}
	return s.accommodationTx(ctx, org, project, func(tx pgx.Tx) error {
		_, err := tx.Exec(ctx, `INSERT INTO accommodation_lines (org_id, project_id, line_key, split) VALUES ($1::uuid, $2::uuid, $3, true)
ON CONFLICT (org_id, project_id, line_key) DO UPDATE SET split = true, updated_at = now()`, org, project, key)
		return err
	})
}

// PlaceAccommodationLines stores the table's order. order lists every line
// shown, in the new order; "new:<i>" stands for added[i], a line the user
// adds there. It returns the new lines' ids in added's order.
func (s *Store) PlaceAccommodationLines(ctx context.Context, org, project string, order []string, added []profile.StoredLine) ([]string, error) {
	if len(order) == 0 || len(order) > 2000 {
		return nil, ErrInvalidInput
	}
	at := make([]int, len(added))
	for i := range at {
		at[i] = -1
	}
	var refs []string
	var refAt []float64
	for i, ref := range order {
		if n, ok := strings.CutPrefix(ref, "new:"); ok {
			k, err := strconv.Atoi(n)
			if err != nil || k < 0 || k >= len(added) || at[k] >= 0 {
				return nil, ErrInvalidInput
			}
			at[k] = i
			continue
		}
		refs = append(refs, ref)
		refAt = append(refAt, float64(i))
	}
	for i, l := range added {
		if at[i] < 0 || !validLineFields(l) {
			return nil, ErrInvalidInput
		}
	}
	var ids, keys []string
	var idAt, keyAt []float64
	for i, ref := range refs {
		switch {
		case validUUID(ref):
			ids, idAt = append(ids, ref), append(idAt, refAt[i])
		case lineKeyPattern.MatchString(ref):
			keys, keyAt = append(keys, ref), append(keyAt, refAt[i])
		default:
			return nil, ErrInvalidInput
		}
	}
	var created []string
	err := s.accommodationTx(ctx, org, project, func(tx pgx.Tx) error {
		if len(keys) > 0 {
			if _, err := tx.Exec(ctx, `INSERT INTO accommodation_lines (org_id, project_id, line_key, position)
SELECT $1::uuid, $2::uuid, k, p FROM unnest($3::text[], $4::float8[]) AS t(k, p)
ON CONFLICT (org_id, project_id, line_key) DO UPDATE SET position = excluded.position, updated_at = now()`, org, project, keys, keyAt); err != nil {
				return err
			}
		}
		if len(ids) > 0 {
			tag, err := tx.Exec(ctx, `UPDATE accommodation_lines a SET position = t.p, updated_at = now()
FROM unnest($3::uuid[], $4::float8[]) AS t(id, p)
WHERE a.org_id=$1::uuid AND a.project_id=$2::uuid AND a.id = t.id AND a.line_key IS NULL`, org, project, ids, idAt)
			if err := foundAll(tag.RowsAffected(), len(ids), err); err != nil {
				return err
			}
		}
		for i, l := range added {
			var id string
			set := slices.Clone(l.Set)
			slices.Sort(set)
			if err := tx.QueryRow(ctx, `INSERT INTO accommodation_lines (org_id, project_id, name, level, area_sqm, comment, set_fields, position)
VALUES ($1::uuid, $2::uuid, $3, $4, NULLIF($5,'')::numeric, $6, $7::text[], $8) RETURNING id::text`,
				org, project, l.Name, l.Level, l.Area, l.Comment, slices.Compact(set), float64(at[i])).Scan(&id); err != nil {
				return err
			}
			created = append(created, id)
		}
		return nil
	})
	return created, err
}

// accommodationTx runs one schedule write under the project's lock, after
// checking the project is the org's.
func (s *Store) accommodationTx(ctx context.Context, org, project string, fn func(pgx.Tx) error) error {
	if !validUUID(org) || !validUUID(project) {
		return ErrNotFound
	}
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if err := lockProject(ctx, tx, org, project); err != nil {
		return err
	}
	var mine bool
	if err := tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM projects WHERE org_id=$1::uuid AND id=$2::uuid)`, org, project).Scan(&mine); err != nil {
		return err
	}
	if !mine {
		return ErrNotFound
	}
	if err := fn(tx); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

// lineRefs splits line references into added lines' ids and generated keys.
func lineRefs(refs []string) (ids, keys []string, err error) {
	if len(refs) == 0 || len(refs) > 2000 {
		return nil, nil, ErrInvalidInput
	}
	for _, r := range refs {
		switch {
		case validUUID(r):
			ids = append(ids, r)
		case lineKeyPattern.MatchString(r):
			if _, _, ok := profile.ParseLineKey(r); !ok {
				return nil, nil, ErrInvalidInput
			}
			keys = append(keys, r)
		default:
			return nil, nil, ErrInvalidInput
		}
	}
	return ids, keys, nil
}

func validLineFields(l profile.StoredLine) bool {
	for _, f := range l.Set {
		if _, ok := accommodationColumns[f]; !ok {
			return false
		}
	}
	if l.Area != "" {
		if v, err := strconv.ParseFloat(l.Area, 64); err != nil || v < 0 {
			return false
		}
	}
	return len(l.Name) <= 120 && len(l.Level) <= 80 && len(l.Comment) <= 500
}

// foundAll turns an update that missed some rows into ErrNotFound: a line
// id from another project or org.
func foundAll(affected int64, want int, err error) error {
	if err != nil {
		return err
	}
	if affected != int64(want) {
		return ErrNotFound
	}
	return nil
}

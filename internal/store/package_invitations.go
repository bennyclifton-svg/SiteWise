package store

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
)

// Invitation is a firm the PM named on a Procurement row before its quote
// arrived (docs/plans/2026-11-11-procurement-invited-firms.md). It is not a
// proposal: nothing is priced, compared or awarded from it.
type Invitation struct {
	PackageID string `json:"package_id"`
	FirmID    string `json:"firm_id"`
	FirmName  string `json:"firm_name"`
}

// InviteFirm names a firm on a live package: a directory firm by id, or a
// typed name. A grouped member's invitation is its group's, as its
// proposals are. Inviting a firm twice is one invitation.
func (s *Store) InviteFirm(ctx context.Context, org, project, pkg, firmID, name string) (Invitation, error) {
	name = strings.TrimSpace(name)
	if (firmID == "") == (name == "") || firmID != "" && !validUUID(firmID) || len(name) > 200 {
		return Invitation{}, ErrInvalidInput
	}
	if !validUUID(pkg) {
		return Invitation{}, ErrNotFound
	}
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return Invitation{}, err
	}
	defer tx.Rollback(ctx)
	// The project lock keeps two nominations of one new name to one firm.
	if err := lockProject(ctx, tx, org, project); err != nil {
		return Invitation{}, err
	}
	var live bool
	if err := tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM packages WHERE org_id=$1::uuid AND project_id=$2::uuid AND id=$3::uuid AND retired_at IS NULL)`, org, project, pkg).Scan(&live); err != nil {
		return Invitation{}, err
	}
	if !live {
		return Invitation{}, ErrNotFound
	}
	if pkg, err = routeToGroup(ctx, tx, org, project, pkg); err != nil {
		return Invitation{}, err
	}
	var f Firm
	if firmID != "" {
		f, err = readFirm(ctx, tx, org, firmID)
	} else {
		f, err = firmNamed(ctx, tx, org, name)
	}
	if err != nil {
		return Invitation{}, err
	}
	if _, err := tx.Exec(ctx, `INSERT INTO package_invitations (org_id,project_id,package_id,firm_id) VALUES ($1::uuid,$2::uuid,$3::uuid,$4::uuid) ON CONFLICT DO NOTHING`, org, project, pkg, f.ID); err != nil {
		return Invitation{}, err
	}
	if err := invitationEvent(ctx, tx, s, org, project, pkg, "invited"); err != nil {
		return Invitation{}, err
	}
	return Invitation{PackageID: pkg, FirmID: f.ID, FirmName: f.Name}, tx.Commit(ctx)
}

// RemoveInvitation takes a firm's invitation off a row. On a group's row it
// also finds one made on a member before the consultants were combined.
func (s *Store) RemoveInvitation(ctx context.Context, org, project, pkg, firmID string) error {
	if !validUUID(pkg) || !validUUID(firmID) {
		return ErrNotFound
	}
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	if err := lockProject(ctx, tx, org, project); err != nil {
		return err
	}
	if pkg, err = routeToGroup(ctx, tx, org, project, pkg); err != nil {
		return err
	}
	tag, err := tx.Exec(ctx, `DELETE FROM package_invitations i USING packages p
WHERE i.org_id=$1::uuid AND i.project_id=$2::uuid AND i.firm_id=$4::uuid
  AND p.org_id=i.org_id AND p.project_id=i.project_id AND p.id=i.package_id AND (p.id=$3::uuid OR p.group_id=$3::uuid)`, org, project, pkg, firmID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	if err := invitationEvent(ctx, tx, s, org, project, pkg, "uninvited"); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

// CorrectInvitation puts the firm the PM meant in an invited firm's cell,
// keeping the cell's place on the row. A firm that is only a name typed for
// this row is renamed (or dropped, when a directory firm is chosen), so the
// misspelling leaves the directory; a firm with details, quotes or other
// rows keeps its name and the cell takes the corrected firm.
func (s *Store) CorrectInvitation(ctx context.Context, org, project, pkg, oldFirm, firmID, name string) (Invitation, error) {
	name = strings.TrimSpace(name)
	if (firmID == "") == (name == "") || firmID != "" && !validUUID(firmID) || len(name) > 200 {
		return Invitation{}, ErrInvalidInput
	}
	if !validUUID(pkg) || !validUUID(oldFirm) {
		return Invitation{}, ErrNotFound
	}
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return Invitation{}, err
	}
	defer tx.Rollback(ctx)
	if err := lockProject(ctx, tx, org, project); err != nil {
		return Invitation{}, err
	}
	if pkg, err = routeToGroup(ctx, tx, org, project, pkg); err != nil {
		return Invitation{}, err
	}
	onRow := func(firm string) (bool, error) {
		var ok bool
		err := tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM package_invitations i JOIN packages p ON p.org_id=i.org_id AND p.project_id=i.project_id AND p.id=i.package_id
WHERE i.org_id=$1::uuid AND i.project_id=$2::uuid AND i.firm_id=$4::uuid AND p.retired_at IS NULL AND (p.id=$3::uuid OR p.group_id=$3::uuid))`, org, project, pkg, firm).Scan(&ok)
		return ok, err
	}
	if ok, err := onRow(oldFirm); err != nil || !ok {
		if err == nil {
			err = ErrNotFound
		}
		return Invitation{}, err
	}
	// Bare: no details, no quotes, invited on this row only.
	var bare bool
	if err := tx.QueryRow(ctx, `SELECT f.abn IS NULL AND f.address='' AND f.contact='' AND f.email='' AND f.phone=''
  AND NOT EXISTS(SELECT 1 FROM submissions s WHERE s.org_id=f.org_id AND s.firm_id=f.id)
  AND NOT EXISTS(SELECT 1 FROM package_invitations i JOIN packages p ON p.org_id=i.org_id AND p.project_id=i.project_id AND p.id=i.package_id
    WHERE i.org_id=f.org_id AND i.firm_id=f.id AND NOT (i.project_id=$3::uuid AND $4::uuid IN (p.id, COALESCE(p.group_id, p.id))))
FROM firms f WHERE f.org_id=$1::uuid AND f.id=$2::uuid`, org, oldFirm, project, pkg).Scan(&bare); err != nil {
		return Invitation{}, err
	}
	var f Firm
	if firmID != "" {
		if f, err = readFirm(ctx, tx, org, firmID); err != nil {
			return Invitation{}, err
		}
	} else {
		key := nameKeyOrName(name)
		err := tx.QueryRow(ctx, `SELECT id::text,name FROM firms WHERE org_id=$1::uuid AND name_key=$2 AND id<>$4::uuid
ORDER BY lower(name)=lower($3) DESC, status='confirmed' DESC, created_at LIMIT 1`, org, key, name, oldFirm).Scan(&f.ID, &f.Name)
		switch {
		case err == nil:
		case !errors.Is(err, pgx.ErrNoRows):
			return Invitation{}, err
		case bare:
			f = Firm{ID: oldFirm, Name: name}
			if _, err := tx.Exec(ctx, `UPDATE firms SET name=$3, name_key=$4, updated_at=now(), version=version+1 WHERE org_id=$1::uuid AND id=$2::uuid`, org, oldFirm, name, key); err != nil {
				return Invitation{}, err
			}
		default:
			if f, err = insertFirm(ctx, tx, org, name); err != nil {
				return Invitation{}, err
			}
		}
	}
	if f.ID != oldFirm {
		if ok, err := onRow(f.ID); err != nil || ok {
			if err == nil {
				err = fmt.Errorf("%w: %s is already on this row", ErrConflict, f.Name)
			}
			return Invitation{}, err
		}
		// The invitation keeps its time, so the firm keeps its cell.
		if _, err := tx.Exec(ctx, `UPDATE package_invitations i SET firm_id=$5::uuid FROM packages p
WHERE i.org_id=$1::uuid AND i.project_id=$2::uuid AND i.firm_id=$4::uuid
  AND p.org_id=i.org_id AND p.project_id=i.project_id AND p.id=i.package_id AND (p.id=$3::uuid OR p.group_id=$3::uuid)`, org, project, pkg, oldFirm, f.ID); err != nil {
			return Invitation{}, err
		}
		if bare {
			if _, err := tx.Exec(ctx, `DELETE FROM firms WHERE org_id=$1::uuid AND id=$2::uuid`, org, oldFirm); err != nil {
				return Invitation{}, err
			}
		}
	}
	if err := invitationEvent(ctx, tx, s, org, project, pkg, "invited"); err != nil {
		return Invitation{}, err
	}
	return Invitation{PackageID: pkg, FirmID: f.ID, FirmName: f.Name}, tx.Commit(ctx)
}

// firmNamed is the directory firm a typed name means: one whose name key
// matches (case, spacing and "Pty Ltd" aside), the exact name and a
// confirmed firm first. Otherwise the name becomes a new confirmed firm,
// as Add firm in Prices makes one.
func firmNamed(ctx context.Context, tx pgx.Tx, org, name string) (Firm, error) {
	key := nameKeyOrName(name)
	var f Firm
	err := tx.QueryRow(ctx, `SELECT id::text,name FROM firms WHERE org_id=$1::uuid AND name_key=$2
ORDER BY lower(name)=lower($3) DESC, status='confirmed' DESC, created_at LIMIT 1`, org, key, name).Scan(&f.ID, &f.Name)
	if err == nil || !errors.Is(err, pgx.ErrNoRows) {
		return f, err
	}
	return insertFirm(ctx, tx, org, name)
}

// insertFirm adds a typed name to the directory as a confirmed firm.
func insertFirm(ctx context.Context, tx pgx.Tx, org, name string) (Firm, error) {
	f := Firm{ID: newID(), Name: name, Status: "confirmed"}
	_, err := tx.Exec(ctx, `INSERT INTO firms (org_id,id,name,name_key,status) VALUES ($1::uuid,$2::uuid,$3,$4,'confirmed')`, org, f.ID, name, nameKeyOrName(name))
	return f, err
}

// invitedFirm is one invitation as the board shows it.
type invitedFirm struct {
	firm, name string
	at         time.Time
}

// boardInvitations is each live row's invited firms in the order they were
// named. A grouped member's invitations show on its group's row.
func boardInvitations(ctx context.Context, tx pgx.Tx, org, project string) (map[string][]invitedFirm, error) {
	rows, err := tx.Query(ctx, `SELECT COALESCE(p.group_id, i.package_id)::text, i.firm_id::text, f.name, i.created_at
FROM package_invitations i
JOIN packages p ON p.org_id=i.org_id AND p.project_id=i.project_id AND p.id=i.package_id
JOIN firms f ON f.org_id=i.org_id AND f.id=i.firm_id
WHERE i.org_id=$1::uuid AND i.project_id=$2::uuid AND p.retired_at IS NULL
ORDER BY i.created_at, i.firm_id`, org, project)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := map[string][]invitedFirm{}
	seen := map[string]bool{}
	for rows.Next() {
		var pkg string
		var f invitedFirm
		if err := rows.Scan(&pkg, &f.firm, &f.name, &f.at); err != nil {
			return nil, err
		}
		// One firm named on two members of a group is one cell on the group.
		if !seen[pkg+"/"+f.firm] {
			seen[pkg+"/"+f.firm] = true
			out[pkg] = append(out[pkg], f)
		}
	}
	return out, rows.Err()
}

// firmsJoined is when each firm first appeared on each row with a proposal,
// whatever has become of it since: a revised quote keeps the first one's cell.
func firmsJoined(ctx context.Context, tx pgx.Tx, org, project string) (map[string]time.Time, error) {
	rows, err := tx.Query(ctx, `SELECT package_id::text, firm_id::text, min(created_at) FROM submissions
WHERE org_id=$1::uuid AND project_id=$2::uuid AND package_id IS NOT NULL AND firm_id IS NOT NULL GROUP BY 1,2`, org, project)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := map[string]time.Time{}
	for rows.Next() {
		var pkg, firm string
		var at time.Time
		if err := rows.Scan(&pkg, &firm, &at); err != nil {
			return nil, err
		}
		out[pkg+"/"+firm] = at
	}
	return out, rows.Err()
}

// placeFirms is a row's cards in Firm 1, 2, 3 order: an invited firm with
// no card of its own gets an invited card, and each firm sits where it first
// joined the row, invited or first received. A card whose firm is not read
// yet sits where it was uploaded.
func placeFirms(pkg string, cards []FirmCard, invited []invitedFirm, joined map[string]time.Time) []FirmCard {
	on := map[string]time.Time{}
	for _, f := range invited {
		on[f.firm] = f.at
	}
	has := map[string]bool{}
	for i := range cards {
		c := &cards[i]
		if c.FirmID == "" {
			continue
		}
		has[c.FirmID] = true
		if at, ok := joined[pkg+"/"+c.FirmID]; ok && at.Before(c.joined) {
			c.joined = at
		}
		if at, ok := on[c.FirmID]; ok && at.Before(c.joined) {
			c.joined = at
		}
	}
	for _, f := range invited {
		if !has[f.firm] {
			cards = append(cards, FirmCard{FirmID: f.firm, FirmName: f.name, Status: CardInvited, InvitedOn: SydneyDate(f.at), Flags: []string{}, joined: f.at})
		}
	}
	sort.SliceStable(cards, func(i, j int) bool { return cards[i].joined.Before(cards[j].joined) })
	return cards
}

// invitationEvent wakes the Procurement page, as a proposal's events do.
func invitationEvent(ctx context.Context, tx pgx.Tx, s *Store, org, project, pkg, state string) error {
	payload, _ := json.Marshal(map[string]string{"project_id": project, "package_id": pkg, "state": state})
	_, err := appendEvent(ctx, s.q.WithTx(tx), org, EventProcurement, "", string(payload))
	return err
}

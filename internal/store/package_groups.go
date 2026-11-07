package store

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"unicode/utf8"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"sitewise/internal/costs"
	"sitewise/internal/procurement"
)

// Combined engagements, slice 1 (docs/plans/2026-11-07-combined-engagements-slice1.md):
// several consultant rows bought under one appointment. The members keep
// their rows, disciplines and RFP choices; the group holds the proposals and
// one parent fee line with each member's line beneath it. Every action is
// one transaction under the project lock, which award also takes, so the
// award checks below cannot race an award.

var (
	ErrPackageGrouped = errors.New("package is in a combined engagement")
	ErrPackageAwarded = errors.New("undo the award first")
)

// groupMember is what combining needs to know about one row.
type groupMember struct {
	ID, Kind, Role, GroupID, Title string
	Retired                        bool
}

// CombinePackages makes a group of the given consultant rows. title is the
// group's name; empty joins the members' names in the order given.
func (s *Store) CombinePackages(ctx context.Context, org, project, actor string, ids []string, title string) (procurement.Package, error) {
	var group procurement.Package
	ids = distinct(ids)
	if len(ids) < 2 {
		return group, fmt.Errorf("%w: choose two or more consultants to combine", ErrInvalidPackage)
	}
	if len(ids) > 50 {
		return group, fmt.Errorf("%w: too many packages", ErrInvalidPackage)
	}
	for _, id := range ids {
		if !validUUID(id) {
			return group, ErrNotFound
		}
	}
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return group, err
	}
	defer tx.Rollback(ctx)
	if err := lockProject(ctx, tx, org, project); err != nil {
		return group, err
	}
	if err := packageActor(ctx, tx, org, actor); err != nil {
		return group, err
	}
	members, err := readGroupMembers(ctx, tx, org, project, ids)
	if err != nil {
		return group, err
	}
	if len(members) != len(ids) {
		return group, ErrNotFound
	}
	names := make([]string, 0, len(ids))
	for _, m := range members {
		switch {
		case m.Retired:
			return group, ErrNotFound
		case m.Kind != "services" || m.Role != procurement.RoleSingle:
			return group, fmt.Errorf("%w: only consultant rows can be combined", ErrInvalidPackage)
		case m.GroupID != "":
			return group, ErrPackageGrouped
		}
		names = append(names, m.Title)
	}
	if err := refuseLiveAward(ctx, tx, org, project, ids); err != nil {
		return group, err
	}
	title = strings.TrimSpace(title)
	if title == "" {
		title = defaultGroupTitle(names)
	}
	group = procurement.Package{Kind: "services", Role: procurement.RoleGroup, Title: title, Novation: procurement.NovationNo, LifecycleStatus: "planned"}
	if err := procurement.ValidatePackage(group); err != nil {
		return group, fmt.Errorf("%w: %v", ErrInvalidPackage, err)
	}
	group.ID = newID()
	provenance, _ := json.Marshal(map[string]any{"actor": actor, "combined": ids})
	if _, err := tx.Exec(ctx, `INSERT INTO packages(org_id,id,project_id,kind,title,novation,lifecycle_status,origin,review_status,meaning,provenance,role) VALUES($1::uuid,$2::uuid,$3::uuid,'services',$4,'no','planned','user','accepted_for_planning','stated',$5::jsonb,'group')`, org, group.ID, project, title, provenance); err != nil {
		return group, err
	}
	if _, err := tx.Exec(ctx, `UPDATE packages SET group_id=$3::uuid,version=version+1 WHERE org_id=$1::uuid AND project_id=$2::uuid AND id=ANY($4::uuid[])`, org, project, group.ID, ids); err != nil {
		return group, err
	}
	// Proposals already in hand become the group's; each remembers its member.
	if _, err := tx.Exec(ctx, `UPDATE submissions SET group_member_id=package_id,package_id=$3::uuid,version=version+1,updated_at=now() WHERE org_id=$1::uuid AND project_id=$2::uuid AND package_id=ANY($4::uuid[]) AND status<>'awarded'`, org, project, group.ID, ids); err != nil {
		return group, err
	}
	plan, err := nestMemberFeeLines(ctx, tx, org, project, actor, group, ids)
	if err != nil {
		return group, err
	}
	if group, err = readPackage(ctx, tx, org, project, group.ID); err != nil {
		return group, err
	}
	return group, s.finishGroupWrite(ctx, tx, org, project, plan)
}

// UncombinePackages releases every member of a group and retires the group.
func (s *Store) UncombinePackages(ctx context.Context, org, project, actor, groupID string) error {
	return s.groupWrite(ctx, org, project, actor, groupID, func(tx pgx.Tx, target groupMember) (string, error) {
		if target.Role != procurement.RoleGroup {
			return "", fmt.Errorf("%w: not a combined engagement", ErrInvalidPackage)
		}
		return s.releaseMembers(ctx, tx, org, project, actor, target.ID, nil)
	})
}

// RemoveFromGroup takes one member out of its group. A group left with one
// member is no group, so the last two are released together.
func (s *Store) RemoveFromGroup(ctx context.Context, org, project, actor, memberID string) error {
	return s.groupWrite(ctx, org, project, actor, memberID, func(tx pgx.Tx, target groupMember) (string, error) {
		if target.GroupID == "" {
			return "", fmt.Errorf("%w: not in a combined engagement", ErrInvalidPackage)
		}
		var remaining int
		if err := tx.QueryRow(ctx, `SELECT count(*) FROM packages WHERE org_id=$1::uuid AND project_id=$2::uuid AND group_id=$3::uuid`, org, project, target.GroupID).Scan(&remaining); err != nil {
			return "", err
		}
		if remaining <= 2 {
			return s.releaseMembers(ctx, tx, org, project, actor, target.GroupID, nil)
		}
		return s.releaseMembers(ctx, tx, org, project, actor, target.GroupID, []string{target.ID})
	})
}

// groupWrite runs one uncombine or remove: the project lock, the target row
// in this org and project, then the change and its announcements.
func (s *Store) groupWrite(ctx context.Context, org, project, actor, id string, change func(pgx.Tx, groupMember) (string, error)) error {
	if !validUUID(id) {
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
	if err := packageActor(ctx, tx, org, actor); err != nil {
		return err
	}
	rows, err := readGroupMembers(ctx, tx, org, project, []string{id})
	if err != nil {
		return err
	}
	if len(rows) != 1 || rows[0].Retired {
		return ErrNotFound
	}
	plan, err := change(tx, rows[0])
	if err != nil {
		return err
	}
	return s.finishGroupWrite(ctx, tx, org, project, plan)
}

// releaseMembers takes members (all of them when nil) out of the group:
// their fee lines go back to the top level with their budgets, their
// proposals go back to them, and an emptied group loses its line and row.
// It returns the cost plan it changed, or "".
func (s *Store) releaseMembers(ctx context.Context, tx pgx.Tx, org, project, actor, groupID string, members []string) (string, error) {
	if err := refuseLiveAward(ctx, tx, org, project, []string{groupID}); err != nil {
		return "", err
	}
	var all []string
	var title string
	if err := tx.QueryRow(ctx, `SELECT COALESCE(array_agg(id::text ORDER BY id),'{}'),(SELECT title FROM packages WHERE org_id=$1::uuid AND project_id=$2::uuid AND id=$3::uuid) FROM packages WHERE org_id=$1::uuid AND project_id=$2::uuid AND group_id=$3::uuid`, org, project, groupID).Scan(&all, &title); err != nil {
		return "", err
	}
	whole := members == nil
	if whole {
		members = all
	}
	plan, err := unnestMemberFeeLines(ctx, tx, org, project, groupID, members, whole)
	if err != nil {
		return "", err
	}
	if _, err := tx.Exec(ctx, `UPDATE submissions SET package_id=group_member_id,group_member_id=NULL,version=version+1,updated_at=now() WHERE org_id=$1::uuid AND project_id=$2::uuid AND package_id=$3::uuid AND group_member_id=ANY($4::uuid[])`, org, project, groupID, members); err != nil {
		return "", err
	}
	if whole {
		// A proposal filed on the group itself has no member to go back to:
		// it waits for a package like any unplaced proposal.
		if _, err := tx.Exec(ctx, `UPDATE submissions SET package_id=NULL,version=version+1,updated_at=now() WHERE org_id=$1::uuid AND project_id=$2::uuid AND package_id=$3::uuid AND status<>'awarded'`, org, project, groupID); err != nil {
			return "", err
		}
	} else if renamed, ok := followDefaultTitle(ctx, tx, org, project, title, all, members); ok {
		if _, err := tx.Exec(ctx, `UPDATE packages SET title=$4,version=version+1 WHERE org_id=$1::uuid AND project_id=$2::uuid AND id=$3::uuid`, org, project, groupID, renamed); err != nil {
			return "", err
		}
	}
	if _, err := tx.Exec(ctx, `UPDATE packages SET group_id=NULL,version=version+1 WHERE org_id=$1::uuid AND project_id=$2::uuid AND id=ANY($3::uuid[])`, org, project, members); err != nil {
		return "", err
	}
	if whole {
		// Retired, not deleted: packages are never deleted, so history that
		// names the group keeps resolving.
		if _, err := tx.Exec(ctx, `UPDATE packages SET retired_at=now(),version=version+1,provenance=provenance||jsonb_build_object('uncombined_by',$4::text) WHERE org_id=$1::uuid AND project_id=$2::uuid AND id=$3::uuid`, org, project, groupID, actor); err != nil {
			return "", err
		}
	}
	return plan, nil
}

// followDefaultTitle renames a group still carrying its default name after
// members leave; a name the PM chose stays.
func followDefaultTitle(ctx context.Context, tx pgx.Tx, org, project, title string, all, leaving []string) (string, bool) {
	before, err := memberTitles(ctx, tx, org, project, all, nil)
	if err != nil || defaultGroupTitle(before) != title {
		return "", false
	}
	after, err := memberTitles(ctx, tx, org, project, all, leaving)
	if err != nil {
		return "", false
	}
	return defaultGroupTitle(after), true
}

// memberTitles lists the titles of ids, less skip, in the order the group
// was combined (its provenance), falling back to title order.
func memberTitles(ctx context.Context, tx pgx.Tx, org, project string, ids, skip []string) ([]string, error) {
	rows, err := tx.Query(ctx, `SELECT p.title FROM packages p LEFT JOIN packages g ON g.org_id=p.org_id AND g.project_id=p.project_id AND g.id=p.group_id
LEFT JOIN LATERAL (SELECT ord FROM jsonb_array_elements_text(COALESCE(g.provenance->'combined','[]'::jsonb)) WITH ORDINALITY AS c(id,ord) WHERE c.id=p.id::text) o ON true
WHERE p.org_id=$1::uuid AND p.project_id=$2::uuid AND p.id=ANY($3::uuid[]) AND NOT p.id=ANY($4::uuid[]) ORDER BY o.ord NULLS LAST,p.title,p.id`, org, project, ids, append([]string{}, skip...))
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []string{}
	for rows.Next() {
		var t string
		if err := rows.Scan(&t); err != nil {
			return nil, err
		}
		out = append(out, t)
	}
	return out, rows.Err()
}

// nestMemberFeeLines gives the group a parent fee line in the draft and moves
// each member's top-level fee line under it, values untouched. A draft
// without member lines is left for prepare, which nests new lines itself.
// Frozen baselines are other plan versions and are not touched.
func nestMemberFeeLines(ctx context.Context, tx pgx.Tx, org, project, actor string, group procurement.Package, members []string) (string, error) {
	plan, err := readCostPlan(ctx, tx, org, project, "")
	if err != nil || plan.Status != "draft" {
		return "", err
	}
	in := map[string]bool{}
	for _, m := range members {
		in[m] = true
	}
	var moving []string
	order := -1
	for _, it := range plan.Items {
		if it.LineKind == "fee" && in[it.PackageID] && it.ParentItemID == "" {
			moving = append(moving, it.ID)
			if it.DisplayOrder != nil && (order < 0 || *it.DisplayOrder < order) {
				order = *it.DisplayOrder
			}
		}
	}
	if len(moving) == 0 {
		return "", nil
	}
	line := newID()
	content := costs.Content{Label: group.Title, LineKind: "fee", PackageID: group.ID, Posting: false, Origin: "user", Meaning: "allowance",
		Rationale: "Combined engagement: one appointment holding its members' fee lines."}
	if err := costs.ValidateContent(content); err != nil {
		return "", err
	}
	if _, err := tx.Exec(ctx, `INSERT INTO cost_items(org_id,project_id,id,created_in_version_id) VALUES($1::uuid,$2::uuid,$3::uuid,$4::uuid)`, org, project, line, plan.ID); err != nil {
		return "", err
	}
	if err := writeCostContent(ctx, tx, org, project, plan.ID, actor, line, content, false); err != nil {
		return "", err
	}
	// The source key stops prepare suggesting a second group line; the
	// display order puts the group where its first member was.
	if _, err := tx.Exec(ctx, `UPDATE cost_item_revisions SET source_key=$4,provenance=provenance||CASE WHEN $5>=0 THEN jsonb_build_object('display_order',$5::int) ELSE '{}'::jsonb END WHERE org_id=$1::uuid AND plan_version_id=$2::uuid AND cost_item_id=$3::uuid`, org, plan.ID, line, "fee:"+group.ID, order); err != nil {
		return "", err
	}
	if _, err := tx.Exec(ctx, `UPDATE cost_item_revisions SET parent_item_id=$3::uuid,version=version+1 WHERE org_id=$1::uuid AND plan_version_id=$2::uuid AND cost_item_id=ANY($4::uuid[])`, org, plan.ID, line, moving); err != nil {
		return "", err
	}
	return plan.ID, nil
}

// unnestMemberFeeLines moves members' lines from under the group's line back
// to the top level, values untouched. When the whole group goes, its line
// goes too, unless the PM put figures or other lines on it: those are not
// thrown away silently.
func unnestMemberFeeLines(ctx context.Context, tx pgx.Tx, org, project, groupID string, members []string, whole bool) (string, error) {
	plan, err := readCostPlan(ctx, tx, org, project, "")
	if err != nil || plan.Status != "draft" {
		return "", err
	}
	parents := map[string]bool{}
	for _, it := range plan.Items {
		if it.LineKind == "fee" && it.PackageID == groupID && it.ParentItemID == "" {
			parents[it.ID] = true
		}
	}
	if len(parents) == 0 {
		return "", nil
	}
	in := map[string]bool{}
	for _, m := range members {
		in[m] = true
	}
	var moving []string
	for _, it := range plan.Items {
		if !parents[it.ParentItemID] {
			continue
		}
		if in[it.PackageID] {
			moving = append(moving, it.ID)
		} else if whole {
			return "", fmt.Errorf("%w: move the other lines under %s before uncombining", ErrInvalidPackage, it.Label)
		}
	}
	if len(moving) > 0 {
		if _, err := tx.Exec(ctx, `UPDATE cost_item_revisions SET parent_item_id=NULL,version=version+1 WHERE org_id=$1::uuid AND plan_version_id=$2::uuid AND cost_item_id=ANY($3::uuid[])`, org, plan.ID, moving); err != nil {
			return "", err
		}
	}
	if whole {
		ids := []string{}
		for _, it := range plan.Items {
			if !parents[it.ID] {
				continue
			}
			for _, v := range it.Values {
				if v.ValueState == "known" {
					return "", fmt.Errorf("%w: clear the figures on %s before uncombining", ErrInvalidPackage, it.Label)
				}
			}
			ids = append(ids, it.ID)
		}
		for _, q := range []string{
			`DELETE FROM scope_cost_links WHERE org_id=$1::uuid AND plan_version_id=$2::uuid AND cost_item_id=ANY($3::uuid[])`,
			`DELETE FROM cost_values WHERE org_id=$1::uuid AND plan_version_id=$2::uuid AND cost_item_id=ANY($3::uuid[])`,
			`DELETE FROM cost_item_revisions WHERE org_id=$1::uuid AND plan_version_id=$2::uuid AND cost_item_id=ANY($3::uuid[])`,
		} {
			if _, err := tx.Exec(ctx, q, org, plan.ID, ids); err != nil {
				return "", err
			}
		}
	}
	if len(moving) == 0 && !whole {
		return "", nil
	}
	return plan.ID, nil
}

// finishGroupWrite announces packages and, when lines moved, the cost plan,
// then commits.
func (s *Store) finishGroupWrite(ctx context.Context, tx pgx.Tx, org, project, plan string) error {
	if plan == "" {
		return s.finishPackagesWrite(ctx, tx, org, project)
	}
	if err := s.packagesChanged(ctx, tx, org, project); err != nil {
		return err
	}
	_, err := s.finishCost(ctx, tx, org, project, plan)
	return err
}

func readGroupMembers(ctx context.Context, tx pgx.Tx, org, project string, ids []string) ([]groupMember, error) {
	rows, err := tx.Query(ctx, `SELECT id::text,kind,role,COALESCE(group_id::text,''),title,retired_at IS NOT NULL FROM packages WHERE org_id=$1::uuid AND project_id=$2::uuid AND id=ANY($3::uuid[]) FOR UPDATE`, org, project, ids)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	byID := map[string]groupMember{}
	for rows.Next() {
		var m groupMember
		if err := rows.Scan(&m.ID, &m.Kind, &m.Role, &m.GroupID, &m.Title, &m.Retired); err != nil {
			return nil, err
		}
		byID[m.ID] = m
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	// Keep the caller's order: it is the order of the default name.
	out := []groupMember{}
	for _, id := range ids {
		if m, ok := byID[strings.ToLower(id)]; ok {
			out = append(out, m)
		}
	}
	return out, nil
}

// refuseLiveAward is checked inside the action's transaction, after the
// project lock, so an award cannot land between the check and the write.
func refuseLiveAward(ctx context.Context, tx pgx.Tx, org, project string, ids []string) error {
	var live bool
	if err := tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM awards WHERE org_id=$1::uuid AND project_id=$2::uuid AND package_id=ANY($3::uuid[]) AND undone_at IS NULL)`, org, project, ids).Scan(&live); err != nil {
		return err
	}
	if live {
		return ErrPackageAwarded
	}
	return nil
}

// defaultGroupTitle joins the members' names: "A and B", "A, B and C". A
// name too long for a package title falls back to a plain one.
func defaultGroupTitle(names []string) string {
	t := ""
	switch len(names) {
	case 0:
	case 1:
		t = names[0]
	default:
		t = strings.Join(names[:len(names)-1], ", ") + " and " + names[len(names)-1]
	}
	if t == "" || utf8.RuneCountInString(t) > 200 {
		return "Combined engagement"
	}
	return t
}

func distinct(ids []string) []string {
	seen := map[string]bool{}
	out := []string{}
	for _, id := range ids {
		key := strings.ToLower(strings.TrimSpace(id))
		if !seen[key] {
			seen[key] = true
			out = append(out, key)
		}
	}
	return out
}

func validUUID(s string) bool {
	var u pgtype.UUID
	return len(s) == 36 && u.Scan(s) == nil
}

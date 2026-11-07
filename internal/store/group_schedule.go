package store

import (
	"context"
	"errors"
	"strings"

	"github.com/jackc/pgx/v5"
	"sitewise/internal/procurement"
	"sitewise/internal/proposals"
	"sitewise/internal/rfp"
)

// Combined engagements, slice 3 (docs/plans/2026-11-07-combined-engagements-slice3.md):
// a group's proposals, price table and award. This file gives the group its
// schedule (each member's fee rows, then combined stages) and routes a
// proposal to the group by code. No Jev call here.

// servicesDisciplines are the disciplines a combined "Services Engineer"
// filing (consultant.services) covers.
var servicesDisciplines = map[string]bool{
	"consultant.mechanical": true, "consultant.electrical": true, "consultant.hydraulic": true, "consultant.fire_services": true,
}

const servicesFiling = "consultant.services"

// groupPackage reports whether pkg is a live combined engagement in this
// org and project.
func groupPackage(ctx context.Context, q querier, org, project, pkg string) (bool, error) {
	var ok bool
	err := q.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM packages WHERE org_id=$1::uuid AND project_id=$2::uuid AND id::text=$3 AND role='group' AND retired_at IS NULL)`, org, project, pkg).Scan(&ok)
	return ok, err
}

// routeToGroup is the package a proposal placed on pkg belongs to: a
// grouped member's proposals are the group's, so they land there.
func routeToGroup(ctx context.Context, q querier, org, project, pkg string) (string, error) {
	if pkg == "" {
		return pkg, nil
	}
	var group string
	err := q.QueryRow(ctx, `SELECT COALESCE(group_id::text,'') FROM packages WHERE org_id=$1::uuid AND project_id=$2::uuid AND id::text=$3`, org, project, pkg).Scan(&group)
	if errors.Is(err, pgx.ErrNoRows) || err == nil && group == "" {
		return pkg, nil
	}
	return group, err
}

// groupMembers is a group's members as the reader and the award use them:
// the layout (keys, names, words) and each member's own live schedule.
type groupMembers struct {
	Packages  []procurement.Package
	Layout    proposals.GroupLayout
	Schedules [][]rfp.ScheduleItem
}

// groupMembersOf loads a group's members in Procurement order. With sync,
// each member's own schedule is brought up to date and read (its RFP
// draft, as the group RFP prints); without, only the layout is built.
func (s *Store) groupMembersOf(ctx context.Context, tx pgx.Tx, org, project, group string, sync bool) (groupMembers, error) {
	var g groupMembers
	members, err := readGroupMemberPackages(ctx, tx, org, project, group)
	if err != nil {
		return g, err
	}
	cat := s.workCatalog()
	used := map[string]bool{}
	for _, m := range members {
		own := []rfp.ScheduleItem{}
		if sync {
			if err := s.ensureSchedule(ctx, tx, org, project, m.ID); err != nil {
				return g, err
			}
			items, err := readSchedule(ctx, tx, org, m.ID)
			if err != nil {
				return g, err
			}
			for _, it := range items {
				if !it.Retired && !it.Custom {
					own = append(own, it.ScheduleItem)
				}
			}
		}
		part := rfp.GroupPart{PackageID: m.ID, PackageTitle: m.Title, DisciplineID: m.DisciplineID}
		title := m.Title
		if cat != nil {
			if d, ok := cat.RFPForDiscipline(m.DisciplineID); ok {
				part.Templated, part.Discipline = true, d
				if d.Title != "" {
					title = d.Title
				}
			}
		}
		key := proposals.MemberKeyFor(rfp.MemberKey(part))
		for used[key] {
			// Two members of one discipline file: keep their rows apart.
			key += "-2"
		}
		used[key] = true
		words := append(rfp.MemberWords(part), strings.ToLower(m.Title))
		g.Packages = append(g.Packages, m)
		g.Layout.Members = append(g.Layout.Members, proposals.GroupMember{Key: key, Title: title, Words: words})
		g.Schedules = append(g.Schedules, own)
	}
	return g, nil
}

// ensureGroupSchedule makes the group's rows its members' fee rows and the
// combined stages. A member that left keeps its rows retired, so cells read
// onto them keep their labels and show as left the group.
func (s *Store) ensureGroupSchedule(ctx context.Context, tx pgx.Tx, org, project, group string) (groupMembers, error) {
	g, err := s.groupMembersOf(ctx, tx, org, project, group, true)
	if err != nil {
		return g, err
	}
	schedule := proposals.GroupSchedule(g.Layout, g.Schedules)
	g.Layout = g.Layout.WithCombined(schedule)
	return g, upsertSchedule(ctx, tx, org, project, group, schedule, ScheduleFromDefault)
}

// servesDiscipline is true when a proposal filed as discipline d belongs to
// package p by rule: its own discipline, or a group holding it.
func servesDiscipline(p proposals.Package, d string) bool {
	if d == "" {
		return false
	}
	if p.Discipline == d {
		return true
	}
	for _, x := range p.Disciplines {
		if x == d {
			return true
		}
	}
	return false
}

// groupCandidates turns the project's packages into the reader's
// candidates: a grouped member is not one (its group is), and a group
// carries its members' disciplines, plus the combined services filing when
// every member is a service it covers. member maps each member to its group.
func groupCandidates(rows []candidateRow) (out []proposals.Package, member map[string]string) {
	member = map[string]string{}
	disciplines := map[string][]string{}
	services := map[string]bool{}
	for _, r := range rows {
		if r.GroupID == "" {
			continue
		}
		member[r.ID] = r.GroupID
		if r.Discipline != "" {
			disciplines[r.GroupID] = append(disciplines[r.GroupID], r.Discipline)
		}
		if _, seen := services[r.GroupID]; !seen {
			services[r.GroupID] = true
		}
		services[r.GroupID] = services[r.GroupID] && servicesDisciplines[r.Discipline]
	}
	for _, r := range rows {
		if r.GroupID != "" {
			continue
		}
		p := proposals.Package{ID: r.ID, Title: r.Title, Discipline: r.Discipline}
		if r.Role == procurement.RoleGroup {
			p.Disciplines = disciplines[r.ID]
			if services[r.ID] {
				p.Disciplines = append(p.Disciplines, servicesFiling)
			}
		}
		out = append(out, p)
	}
	return out, member
}

// candidateRow is one live package as the reader's context loads it.
type candidateRow struct {
	ID, Title, Discipline, Role, GroupID string
}

// GroupTable is a group's price table layout: its disciplines in
// Procurement order, each with the key its rows carry.
type GroupTable struct {
	Members []GroupTableMember `json:"members"`
	// Left are rows of members no longer in the group: their cells show,
	// and are left out of award until the PM places them again.
	Left []ScheduleItem `json:"left"`
}

// GroupTableMember is one discipline of the group's table.
type GroupTableMember struct {
	PackageID string `json:"package_id"`
	Key       string `json:"key"`
	Title     string `json:"title"`
}

func groupTable(g groupMembers) *GroupTable {
	t := &GroupTable{Members: []GroupTableMember{}, Left: []ScheduleItem{}}
	for i, m := range g.Layout.Members {
		t.Members = append(t.Members, GroupTableMember{PackageID: g.Packages[i].ID, Key: m.Key, Title: m.Title})
	}
	return t
}

// left is true for a member row whose member is no longer in the group.
func (t *GroupTable) left(key string) bool {
	member, _, combined := proposals.SplitItemKey(key)
	if member == "" || combined {
		return false
	}
	for _, m := range t.Members {
		if m.Key == member {
			return false
		}
	}
	return true
}

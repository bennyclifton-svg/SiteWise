package store

import (
	"context"
	"sort"

	"github.com/jackc/pgx/v5"
	"sitewise/internal/procurement"
)

// Combined engagements, slice 4 (docs/plans/2026-11-07-combined-engagements-slice4.md):
// a group has one set of stage lines, one per stage any member includes,
// in program order. They are ordinary package stages on the group row, so
// the existing stage sync, the group's RFP and its award follow them with
// no code of their own. Members' stage rows are never rewritten: they are
// inert while grouped and live again once released. Packages add no
// program rows (program plan 13.5.1), so nothing here touches the program's
// dates or links. No Jev.

// memberStagesSQL is every live member's live stages that still follow the
// program. A stage kept after its program stage was deleted stays the
// member's, held by the member's own records.
const memberStagesSQL = `SELECT s.stage_id,COALESCE(s.program_stage_id::text,'') FROM packages p
JOIN package_stages s ON s.org_id=p.org_id AND s.project_id=p.project_id AND s.package_id=p.id
WHERE p.org_id=$1::uuid AND p.project_id=$2::uuid AND p.group_id=$3::uuid AND p.retired_at IS NULL AND s.retired_at IS NULL AND s.removed_from_program IS NULL`

// syncGroupStages brings the group's lines to its members' union: missing
// lines are added through insertPackageStage (which links them to the
// program and records the plan), lines no member includes are retired
// unless something holds them, and the rest keep their ids and versions.
// Runs inside the combine or remove transaction, under the project lock.
func (s *Store) syncGroupStages(ctx context.Context, tx pgx.Tx, org, project, groupID string) error {
	group, err := readPackage(ctx, tx, org, project, groupID)
	if err != nil {
		return err
	}
	plan, err := readStagePlan(ctx, tx, org, project, false)
	if err != nil {
		return err
	}
	wanted, err := s.memberStageUnion(ctx, tx, org, project, group, plan)
	if err != nil {
		return err
	}
	byKey := map[string]int{}
	for i, st := range wanted {
		byKey[stageKey(st, plan)] = i
	}
	ids := make([]string, len(wanted))
	extra := []string{}
	for _, st := range group.Stages {
		if st.RemovedFromProgram != nil {
			extra = append(extra, st.ID)
			continue
		}
		if i, ok := byKey[stageKey(st, plan)]; ok && ids[i] == "" {
			ids[i] = st.ID
			continue
		}
		held, err := stageHeld(ctx, tx, org, project, groupID, st.ID, false)
		if err != nil {
			return err
		}
		if held {
			extra = append(extra, st.ID)
			continue
		}
		if _, err := tx.Exec(ctx, `UPDATE package_stages SET retired_at=now(),version=version+1 WHERE org_id=$1::uuid AND project_id=$2::uuid AND package_id=$3::uuid AND id=$4::uuid`, org, project, groupID, st.ID); err != nil {
			return err
		}
	}
	for i, st := range wanted {
		if ids[i] != "" {
			continue
		}
		st.Ordinal = i
		made, err := s.insertPackageStage(ctx, tx, org, group, st, plan)
		if err != nil {
			return err
		}
		ids[i] = made.ID
	}
	// Program order for the lines, then any held lines after them.
	_, err = tx.Exec(ctx, `UPDATE package_stages s SET ordinal=o.n-1,version=s.version+1
FROM unnest($4::uuid[]) WITH ORDINALITY AS o(id,n)
WHERE s.org_id=$1::uuid AND s.project_id=$2::uuid AND s.package_id=$3::uuid AND s.id=o.id AND s.ordinal<>o.n-1`, org, project, groupID, append(ids, extra...))
	return err
}

// memberStageUnion is one stage per program stage (or, with no program,
// per catalogue stage) that any member includes, labelled as the program
// labels it, in program order, then catalogue order. A label appears once,
// as in a single package's defaults.
func (s *Store) memberStageUnion(ctx context.Context, tx pgx.Tx, org, project string, group procurement.Package, plan *stagePlan) ([]procurement.Stage, error) {
	rows, err := tx.Query(ctx, memberStagesSQL, org, project, group.ID)
	if err != nil {
		return nil, err
	}
	type found struct{ stageID, programStageID string }
	var all []found
	for rows.Next() {
		var f found
		if err := rows.Scan(&f.stageID, &f.programStageID); err != nil {
			rows.Close()
			return nil, err
		}
		all = append(all, f)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return nil, err
	}
	catalogue := map[string]int{}
	labels := map[string]string{}
	if cat := s.workCatalog(); cat != nil {
		for i, d := range cat.DeliveryStages() {
			catalogue[d.ID], labels[d.ID] = i, d.Label
		}
	}
	out := []procurement.Stage{}
	seenKey, seenLabel := map[string]bool{}, map[string]bool{}
	for _, f := range all {
		st := procurement.Stage{StageID: f.stageID, Label: labels[f.stageID], NovationPhase: "none", Origin: "calculation"}
		if plan != nil {
			if _, live := plan.stage(f.programStageID); live {
				st.ProgramStageID = f.programStageID
			}
		}
		// Takes the program's label and link, or links by catalogue stage.
		if err := s.linkPackageStage(group, &st, plan); err != nil {
			return nil, err
		}
		key := stageKey(st, plan)
		if seenKey[key] || seenLabel[st.Label] {
			continue
		}
		seenKey[key], seenLabel[st.Label] = true, true
		out = append(out, st)
	}
	rank := func(st procurement.Stage) (int, int) {
		p := len(catalogue) + 1
		if plan != nil {
			if i := plan.index(st.ProgramStageID); i >= 0 {
				p = i
			} else {
				p = len(plan.Stages)
			}
		}
		c, ok := catalogue[st.StageID]
		if !ok {
			c = len(catalogue)
		}
		return p, c
	}
	sort.SliceStable(out, func(i, j int) bool {
		pi, ci := rank(out[i])
		pj, cj := rank(out[j])
		if pi != pj {
			return pi < pj
		}
		return ci < cj
	})
	return out, nil
}

// stageKey is what makes two stages the same line: the live program stage
// they point at, else their catalogue stage.
func stageKey(st procurement.Stage, plan *stagePlan) string {
	if _, live := plan.stage(st.ProgramStageID); live {
		return "program:" + st.ProgramStageID
	}
	return "catalogue:" + st.StageID
}

// groupedMember reports the group a package belongs to, or "".
func groupedMember(ctx context.Context, tx pgx.Tx, org, project, pkg string) (string, error) {
	var group string
	err := tx.QueryRow(ctx, `SELECT COALESCE(group_id::text,'') FROM packages WHERE org_id=$1::uuid AND project_id=$2::uuid AND id::text=$3`, org, project, pkg).Scan(&group)
	if err == pgx.ErrNoRows {
		return "", nil
	}
	return group, err
}

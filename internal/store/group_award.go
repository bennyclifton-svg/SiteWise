package store

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/jackc/pgx/v5"
	"sitewise/internal/costs"
	"sitewise/internal/invoices"
	"sitewise/internal/proposals"
)

// Award on a combined engagement (slice 3). One award row on the group;
// each discipline's priced cells become lines under that member's fee line
// and each combined cell a line under the group's line, so the group line's
// roll-up is the one contract sum. Contract values come only from the
// firm's priced cells. No Jev call.

// groupAwardLine is a confirmed line and the member it sits under ("" for
// the group's own line, which holds combined stages).
type groupAwardLine struct {
	AwardLineView
	Member string
}

// groupAwardPlan is a checked group award: every figure from the stored cells.
type groupAwardPlan struct {
	View    AwardView
	Sub     Submission
	Firm    Firm
	Title   string
	Lines   []groupAwardLine
	Members groupMembers
}

// planGroupAward resolves the PM's lines against the group's cells. Each line
// holds one discipline's cells, or combined cells only; every priced base
// cell is awarded (options may be left out). Unplaced amounts, source quote
// stages and rows of members who left are not awarded: the PM places them
// first, so no figure lands under a guessed discipline.
func (s *Store) planGroupAward(ctx context.Context, tx pgx.Tx, org, project, group string, in AwardInput) (groupAwardPlan, error) {
	var g groupAwardPlan
	sub, err := readSubmission(ctx, tx, org, in.SubmissionID)
	if err != nil {
		return g, err
	}
	if sub.ProjectID != project || sub.PackageID != group {
		return g, ErrNotFound
	}
	if sub.Status != SubmissionReceived && sub.Status != SubmissionNeedsReview {
		return g, ErrInvalidInput
	}
	if sub.FirmID == "" {
		return g, fmt.Errorf("%w: name the appointed firm on this proposal before awarding", ErrInvalidInput)
	}
	if g.Firm, err = readFirm(ctx, tx, org, sub.FirmID); err != nil {
		return g, err
	}
	if err := tx.QueryRow(ctx, `SELECT title FROM packages WHERE org_id=$1::uuid AND project_id=$2::uuid AND id::text=$3 AND retired_at IS NULL`, org, project, group).Scan(&g.Title); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return g, ErrNotFound
		}
		return g, err
	}
	if in.Appointment || len(in.Lines) == 0 || len(in.Lines) > 100 {
		return g, ErrInvalidInput
	}
	if g.Members, err = s.groupMembersOf(ctx, tx, org, project, group, false); err != nil {
		return g, err
	}
	owner := map[string]string{} // member key -> member package
	titles := map[string]string{"": g.Title}
	for i, m := range g.Members.Layout.Members {
		owner[m.Key] = g.Members.Packages[i].ID
		titles[g.Members.Packages[i].ID] = m.Title
	}
	schedule, err := readSchedule(ctx, tx, org, group)
	if err != nil {
		return g, err
	}
	live := map[string]ScheduleItem{}
	for _, it := range schedule {
		member, _, combined := proposals.SplitItemKey(it.Key)
		if !it.Retired && it.Price() && (combined || owner[member] != "") {
			live[it.Key] = it
		}
	}
	cells, err := pricedCells(ctx, tx, org, sub.ID)
	if err != nil {
		return g, err
	}
	used := map[string]bool{}
	for _, l := range in.Lines {
		label := strings.TrimSpace(l.Label)
		if label == "" || len(label) > 200 || len(l.Keys) == 0 || len(l.PriceIDs)+len(l.QuoteIDs)+len(l.AppointmentIDs) != 0 {
			return g, ErrInvalidInput
		}
		line := groupAwardLine{AwardLineView: AwardLineView{Label: label, Keys: []string{}, PriceIDs: []string{}, QuoteIDs: []string{}}}
		for i, k := range l.Keys {
			c, priced := cells[k]
			if _, ok := live[k]; !ok || !priced || used[k] {
				return g, ErrInvalidInput
			}
			member, _, combined := proposals.SplitItemKey(k)
			parent := ""
			if !combined {
				parent = owner[member]
			}
			if i > 0 && parent != line.Member {
				return g, ErrInvalidInput
			}
			used[k] = true
			line.Member = parent
			line.Cents += c
			line.Keys = append(line.Keys, k)
		}
		line.Under = titles[line.Member]
		g.Lines = append(g.Lines, line)
		g.View.Lines = append(g.View.Lines, line.AwardLineView)
		g.View.Total += line.Cents
	}
	for k, it := range live {
		if _, priced := cells[k]; priced && it.Base && !used[k] {
			return g, ErrInvalidInput
		}
	}
	if in.Total != "" {
		c, ok, err := invoices.ParseCents(in.Total)
		if err != nil || !ok || c != g.View.Total {
			return g, ErrInvalidInput
		}
	}
	g.Sub = sub
	g.View.SubmissionID, g.View.FirmName, g.View.ParentLabel = sub.ID, g.Firm.Name, g.Title+": "+g.Firm.Name
	return g, nil
}

// pricedCells are a submission's priced cells with a known amount, by key.
func pricedCells(ctx context.Context, tx pgx.Tx, org, sub string) (map[string]int64, error) {
	rows, err := tx.Query(ctx, `SELECT item_key, amount_cents FROM submission_prices WHERE org_id=$1::uuid AND submission_id::text=$2 AND item_key IS NOT NULL AND state='priced' AND amount_cents IS NOT NULL`, org, sub)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := map[string]int64{}
	for rows.Next() {
		var k string
		var c int64
		if err := rows.Scan(&k, &c); err != nil {
			return nil, err
		}
		out[k] = c
	}
	return out, rows.Err()
}

// previewGroupAward is PreviewAward for a group.
func (s *Store) previewGroupAward(ctx context.Context, tx pgx.Tx, org, project, group string, in AwardInput) (AwardView, error) {
	g, err := s.planGroupAward(ctx, tx, org, project, group, in)
	if err != nil {
		return g.View, err
	}
	if parent, err := packageFeeLine(ctx, tx, org, project, group); err != nil {
		return g.View, err
	} else if parent != nil {
		g.View.Budget = budgetCents(*parent)
	}
	return g.View, nil
}

// lineBefore is one fee line the group award changed, as it was.
type lineBefore struct {
	ItemID  string `json:"item_id"`
	Created bool   `json:"created"`
	parentBefore
}

// groupBefore is what undo restores: the package state and every line the
// award turned into a parent (the group's line first).
type groupBefore struct {
	Group       bool         `json:"group"`
	Lifecycle   string       `json:"lifecycle_status"`
	SupplierABN string       `json:"supplier_abn"`
	Lines       []lineBefore `json:"lines"`
}

// awardGroup writes a group award in the caller's transaction and commits.
func (s *Store) awardGroup(ctx context.Context, tx pgx.Tx, p costs.Plan, org, project, group, actor string, in AwardInput) (AwardView, error) {
	g, err := s.planGroupAward(ctx, tx, org, project, group, in)
	if err != nil {
		return g.View, err
	}
	before := groupBefore{Group: true}
	if err := tx.QueryRow(ctx, `SELECT lifecycle_status, COALESCE(supplier_abn,'') FROM packages WHERE org_id=$1::uuid AND id::text=$2`, org, group).Scan(&before.Lifecycle, &before.SupplierABN); err != nil {
		return g.View, err
	}
	plan, err := readCostPlan(ctx, tx, org, project, p.ID)
	if err != nil {
		return g.View, err
	}
	// The group's line, then each member's line that receives stages.
	groupLine, _, err := s.awardParent(ctx, tx, plan, org, project, actor, group, "", g.Title, true, &before)
	if err != nil {
		return g.View, err
	}
	parents := map[string]costs.Item{"": groupLine}
	// Every member's line becomes part of the one contract: a discipline
	// whose stages were all priced combined keeps its line, with nothing of
	// its own beneath it, rather than an open line with no contract.
	for i, m := range g.Members.Packages {
		receives := false
		for _, l := range g.Lines {
			receives = receives || l.Member == m.ID
		}
		line, ok, err := s.awardParent(ctx, tx, plan, org, project, actor, m.ID, groupLine.ID, g.Members.Layout.Members[i].Title, receives, &before)
		if err != nil {
			return g.View, err
		}
		if ok {
			parents[m.ID] = line
		}
	}
	raw, _ := json.Marshal(before)
	awardID := newID()
	if _, err := tx.Exec(ctx, `INSERT INTO awards (org_id,id,project_id,package_id,submission_id,parent_item_id,parent_created,parent_before,created_by)
VALUES ($1::uuid,$2::uuid,$3::uuid,$4::uuid,$5::uuid,$6::uuid,$7,$8::jsonb,$9::uuid)`, org, awardID, project, group, g.Sub.ID, groupLine.ID, before.Lines[0].Created, string(raw), actor); err != nil {
		return g.View, err
	}
	rationale := clipText("Awarded to "+g.Firm.Name+" ("+g.Sub.Filename+")", 1000)
	count := map[string]int{}
	sums := map[string]int64{}
	for i := range g.Lines {
		l := &g.Lines[i]
		parent := parents[l.Member]
		count[l.Member]++
		sums[l.Member] += l.Cents
		code := parent.Code
		if code == "" {
			code = "A"
		}
		id := newID()
		c := costs.Content{ParentItemID: parent.ID, Code: clipText(code+"."+strconv.Itoa(count[l.Member]), 100), Label: l.Label, LineKind: "fee", PackageID: parent.PackageID, Posting: true, Origin: "user", Meaning: "stated", Rationale: rationale}
		if err := s.insertCostItem(ctx, tx, org, project, p.ID, actor, id, c); err != nil {
			return g.View, err
		}
		amount := costs.Money(invoices.FormatCents(l.Cents))
		if err := writeCostValue(ctx, tx, org, project, p.ID, id, actor, "commitment", costs.Value{ValueState: "known", Amount: &amount, Origin: "user", Meaning: "stated", Rationale: rationale}); err != nil {
			return g.View, err
		}
		for _, k := range l.Keys {
			if _, err := tx.Exec(ctx, `INSERT INTO cost_line_sources (org_id,id,project_id,award_id,cost_item_id,item_key,label,ordinal)
SELECT $1::uuid,$2::uuid,$3::uuid,$4::uuid,$5::uuid,$6,COALESCE((SELECT label FROM schedule_items WHERE org_id=$1::uuid AND package_id=$7::uuid AND key=$6),''),$8`, org, newID(), project, awardID, id, k, group, i); err != nil {
				return g.View, err
			}
		}
		l.CostItemID = id
		g.View.Lines[i].CostItemID = id
	}
	// A blank budget adopts the contract beneath its own line, so the group
	// line's roll-up (its own budget plus its members') is never counted
	// twice. A budget the PM set is never changed.
	for member, parent := range parents {
		if sums[member] == 0 || !blankBudget(parent.Values["budget"]) {
			continue
		}
		amount := costs.Money(invoices.FormatCents(sums[member]))
		adopted := costs.Value{ValueState: "known", Amount: &amount, Origin: "user", Meaning: "stated", Rationale: clipText(adoptedBudget+" awarded to "+g.Firm.Name, 1000)}
		if err := writeCostValue(ctx, tx, org, project, p.ID, parent.ID, actor, "budget", adopted); err != nil {
			return g.View, err
		}
	}
	g.View.ParentItemID = groupLine.ID
	if b := budgetCents(groupLine); b != nil {
		g.View.Budget = b
	}
	abn := g.Firm.ABN
	if abn != "" && !invoices.ValidABN(abn) {
		abn = ""
	}
	if _, err := tx.Exec(ctx, `UPDATE packages SET lifecycle_status='appointed', supplier_abn=COALESCE(NULLIF($3,''), supplier_abn), version=version+1 WHERE org_id=$1::uuid AND id::text=$2`, org, group, abn); err != nil {
		return g.View, err
	}
	if _, err := tx.Exec(ctx, `UPDATE submissions SET status=CASE WHEN id=$3::uuid THEN 'awarded' ELSE 'unsuccessful' END, updated_at=now(), version=version+1
WHERE org_id=$1::uuid AND package_id::text=$2 AND (id=$3::uuid OR status IN ('received','needs_review'))`, org, group, g.Sub.ID); err != nil {
		return g.View, err
	}
	if err := BumpRevision(ctx, tx, org, project, "packages"); err != nil {
		return g.View, err
	}
	if err := procurementEvent(ctx, tx, s, org, project, g.Sub.ID, "awarded"); err != nil {
		return g.View, err
	}
	_, err = s.finishCost(ctx, tx, org, project, p.ID)
	return g.View, err
}

// awardParent finds or makes the fee line a group award writes under: the
// group's top-level line (under == ""), or a member's line beneath it. A
// line the PM already split is their structure, so award refuses it. A
// posting line becomes a parent keeping only its budget; undo restores it.
// A missing line is made only when create is set; ok is false otherwise.
func (s *Store) awardParent(ctx context.Context, tx pgx.Tx, plan costs.Plan, org, project, actor, pkg, under, label string, create bool, before *groupBefore) (costs.Item, bool, error) {
	var found *costs.Item
	for i, it := range plan.Items {
		if it.LineKind == "fee" && it.PackageID == pkg && it.ParentItemID == under {
			found = &plan.Items[i]
			if it.SourceKey == "fee:"+pkg {
				break
			}
		}
	}
	if found == nil && !create {
		return costs.Item{}, false, nil
	}
	if found == nil {
		id := newID()
		c := costs.Content{ParentItemID: under, Label: label, LineKind: "fee", PackageID: pkg, Posting: false, Origin: "user", Meaning: "stated"}
		if err := s.insertCostItem(ctx, tx, org, project, plan.ID, actor, id, c); err != nil {
			return costs.Item{}, false, err
		}
		before.Lines = append(before.Lines, lineBefore{ItemID: id, Created: true})
		return costs.Item{ID: id, Content: c, Values: map[string]costs.Value{}}, true, nil
	}
	lb := lineBefore{ItemID: found.ID, parentBefore: parentBefore{Label: found.Label, Posting: found.Posting, Values: found.Values, Content: found.Content}}
	if found.Posting {
		if hasChildren(plan, found.ID) {
			return costs.Item{}, false, ErrConflict
		}
		if _, err := tx.Exec(ctx, `DELETE FROM cost_values WHERE org_id=$1::uuid AND plan_version_id=$2::uuid AND cost_item_id=$3::uuid AND metric<>'budget'`, org, plan.ID, found.ID); err != nil {
			return costs.Item{}, false, err
		}
		if _, err := tx.Exec(ctx, `UPDATE cost_item_revisions SET posting=false, quantity=NULL, rate=NULL, unit=NULL, rate_basis=NULL, version=version+1
WHERE org_id=$1::uuid AND plan_version_id=$2::uuid AND cost_item_id=$3::uuid`, org, plan.ID, found.ID); err != nil {
			return costs.Item{}, false, err
		}
	} else if under != "" {
		// A member line already split into parts by the PM.
		return costs.Item{}, false, ErrConflict
	}
	before.Lines = append(before.Lines, lb)
	return *found, true, nil
}

func hasChildren(plan costs.Plan, id string) bool {
	for _, it := range plan.Items {
		if it.ParentItemID == id {
			return true
		}
	}
	return false
}

// undoGroupAward reverses a group award while no invoice rests on its lines:
// its stage lines go, each parent line is restored as it was (an adopted
// budget back to blank, one the PM has since typed stays), lines it made
// are removed, and the group and its proposals return to where they were.
func (s *Store) undoGroupAward(ctx context.Context, tx pgx.Tx, p costs.Plan, org, project, group, actor string) error {
	var awardID, sub string
	var raw []byte
	err := tx.QueryRow(ctx, `SELECT id::text, submission_id::text, parent_before FROM awards
WHERE org_id=$1::uuid AND project_id=$2::uuid AND package_id::text=$3 AND undone_at IS NULL FOR UPDATE`, org, project, group).Scan(&awardID, &sub, &raw)
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrNotFound
	}
	if err != nil {
		return err
	}
	children, err := awardChildren(ctx, tx, org, awardID)
	if err != nil {
		return err
	}
	var before groupBefore
	if err := json.Unmarshal(raw, &before); err != nil {
		return err
	}
	touched := append([]string{}, children...)
	for _, l := range before.Lines {
		touched = append(touched, l.ItemID)
	}
	var invoiced bool
	if err := tx.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM invoice_allocations WHERE org_id=$1::uuid AND cost_item_id::text = ANY($2::text[]))`, org, touched).Scan(&invoiced); err != nil {
		return err
	}
	if invoiced {
		return ErrAwardLocked
	}
	if err := deleteCostLines(ctx, tx, org, p.ID, children); err != nil {
		return err
	}
	// Members first, the group's line last: a made line is removed only once
	// nothing is left beneath it.
	for i := len(before.Lines) - 1; i >= 0; i-- {
		if err := restoreAwardParent(ctx, tx, org, project, p.ID, before.Lines[i]); err != nil {
			return err
		}
	}
	if _, err := tx.Exec(ctx, `UPDATE awards SET undone_at=now() WHERE org_id=$1::uuid AND id=$2::uuid`, org, awardID); err != nil {
		return err
	}
	lifecycle := before.Lifecycle
	if lifecycle == "" {
		lifecycle = "procuring"
	}
	if _, err := tx.Exec(ctx, `UPDATE packages SET lifecycle_status=$3, supplier_abn=NULLIF($4,''), version=version+1 WHERE org_id=$1::uuid AND id::text=$2`, org, group, lifecycle, before.SupplierABN); err != nil {
		return err
	}
	if _, err := tx.Exec(ctx, `UPDATE submissions SET status='received', updated_at=now(), version=version+1 WHERE org_id=$1::uuid AND package_id::text=$2 AND status IN ('awarded','unsuccessful')`, org, group); err != nil {
		return err
	}
	if err := BumpRevision(ctx, tx, org, project, "packages"); err != nil {
		return err
	}
	if err := procurementEvent(ctx, tx, s, org, project, sub, "award_undone"); err != nil {
		return err
	}
	_, err = s.finishCost(ctx, tx, org, project, p.ID)
	return err
}

func deleteCostLines(ctx context.Context, tx pgx.Tx, org, plan string, ids []string) error {
	for _, q := range []string{
		`DELETE FROM cost_values WHERE org_id=$1::uuid AND plan_version_id=$2::uuid AND cost_item_id::text = ANY($3::text[])`,
		`DELETE FROM cost_item_revisions WHERE org_id=$1::uuid AND plan_version_id=$2::uuid AND cost_item_id::text = ANY($3::text[])`,
	} {
		if _, err := tx.Exec(ctx, q, org, plan, ids); err != nil {
			return err
		}
	}
	return nil
}

// restoreAwardParent puts one parent line back as the award found it.
func restoreAwardParent(ctx context.Context, tx pgx.Tx, org, project, plan string, l lineBefore) error {
	if l.Created {
		var held bool
		if err := tx.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM cost_item_revisions WHERE org_id=$1::uuid AND plan_version_id=$2::uuid AND parent_item_id=$3::uuid)`, org, plan, l.ItemID).Scan(&held); err != nil || held {
			return err
		}
		return deleteCostLines(ctx, tx, org, plan, []string{l.ItemID})
	}
	if _, err := tx.Exec(ctx, `UPDATE cost_item_revisions SET posting=$4, label=$5, quantity=$6::text::numeric, rate=$7::text::numeric, unit=NULLIF($8,''), rate_basis=NULLIF($9,''), version=version+1
WHERE org_id=$1::uuid AND plan_version_id=$2::uuid AND cost_item_id=$3::uuid`, org, plan, l.ItemID, l.Posting, l.Label, l.Content.Quantity, l.Content.Rate, l.Content.Unit, l.Content.RateBasis); err != nil {
		return err
	}
	var rationale string
	if err := tx.QueryRow(ctx, `SELECT COALESCE((SELECT rationale FROM cost_values WHERE org_id=$1::uuid AND plan_version_id=$2::uuid AND cost_item_id=$3::uuid AND metric='budget'),'')`, org, plan, l.ItemID).Scan(&rationale); err != nil {
		return err
	}
	_, had := l.Values["budget"]
	restoreBudget := strings.HasPrefix(rationale, adoptedBudget)
	if !had || restoreBudget {
		if _, err := tx.Exec(ctx, `DELETE FROM cost_values WHERE org_id=$1::uuid AND plan_version_id=$2::uuid AND cost_item_id=$3::uuid AND metric='budget'`, org, plan, l.ItemID); err != nil {
			return err
		}
	}
	for m, val := range l.Values {
		if m == "budget" && !restoreBudget {
			continue
		}
		if _, err := tx.Exec(ctx, `INSERT INTO cost_values(org_id,project_id,plan_version_id,cost_item_id,metric,value_state,amount,low,high,as_of,origin,meaning,rationale,provenance)
VALUES($1::uuid,$2::uuid,$3::uuid,$4::uuid,$5,$6,$7::text::numeric,$8::text::numeric,$9::text::numeric,$10::text::date,$11,$12,$13,$14::jsonb)
ON CONFLICT(org_id,plan_version_id,cost_item_id,metric) DO NOTHING`, org, project, plan, l.ItemID, m, val.ValueState, val.Amount, val.Low, val.High, val.AsOf, val.Origin, val.Meaning, val.Rationale, jsonOrEmpty(val.Provenance)); err != nil {
			return err
		}
	}
	return nil
}

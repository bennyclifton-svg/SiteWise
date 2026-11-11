package store

import (
	"context"
	"slices"

	"github.com/jackc/pgx/v5"

	"sitewise/internal/costs"
)

// CostBulkInput is one action on the Cost Plan lines the user selected:
// delete them, duplicate them, or set their category or package and whether
// they count in totals. It is one write: one transaction, one plan version.
// No Jev; money is not computed here.
type CostBulkInput struct {
	CostWrite
	Op  string   `json:"op"` // delete | duplicate | set
	IDs []string `json:"ids"`
	// Assign is a category for project-wide lines, or a package for fee and
	// works lines ("" hands a works line back to automatic assignment).
	Assign *string `json:"assign,omitempty"`
	// Excluded takes the lines out of totals, or puts them back.
	Excluded *bool `json:"excluded,omitempty"`
}

// maxBulkLines bounds one action; a whole plan is well under it.
const maxBulkLines = 1000

// BulkCostItems applies one action to several lines in one transaction.
// Deletion follows the one-line delete: a group line takes its children,
// frozen history stays, and a removed default stays dismissed.
func (s *Store) BulkCostItems(ctx context.Context, org, project, actor string, in CostBulkInput) (costs.Plan, error) {
	if len(in.IDs) == 0 || len(in.IDs) > maxBulkLines {
		return costs.Plan{}, costs.ErrInvalid
	}
	switch in.Op {
	case "delete", "duplicate":
	case "set":
		if in.Assign == nil && in.Excluded == nil {
			return costs.Plan{}, costs.ErrInvalid
		}
	default:
		return costs.Plan{}, costs.ErrInvalid
	}
	tx, p, e := s.costTx(ctx, org, project, actor, in.CostWrite)
	if e != nil {
		return p, e
	}
	defer tx.Rollback(ctx)
	byID := map[string]costs.Item{}
	for _, i := range p.Items {
		byID[i.ID] = i
	}
	picked := map[string]bool{}
	for _, id := range in.IDs {
		if picked[id] {
			return p, costs.ErrInvalid
		}
		if _, ok := byID[id]; !ok {
			return p, ErrNotFound
		}
		picked[id] = true
	}
	switch in.Op {
	case "delete":
		e = s.removeCostItems(ctx, tx, org, project, actor, p, picked)
	case "set":
		e = s.setCostItems(ctx, tx, org, project, actor, p, in)
	case "duplicate":
		e = s.duplicateCostItems(ctx, tx, org, project, actor, p, picked)
	}
	if e != nil {
		return p, e
	}
	return s.finishCost(ctx, tx, org, project, p.ID)
}

// editableContent is a line's content as an edit sends it back, as the grid
// does: a line tied to a work item names no system of its own, and a
// calculated origin is the user's assumption.
func editableContent(i costs.Item) costs.Content {
	c := i.Content
	if c.Origin == "calculation" {
		c.Origin = "assumption"
	}
	if c.WorkItemID != "" {
		c.SystemID = ""
	}
	return c
}

func (s *Store) setCostItems(ctx context.Context, tx pgx.Tx, org, project, actor string, p costs.Plan, in CostBulkInput) error {
	for _, item := range p.Items {
		if !slices.Contains(in.IDs, item.ID) {
			continue
		}
		c := editableContent(item)
		if in.Excluded != nil {
			c.Excluded = *in.Excluded
		}
		if in.Assign != nil {
			switch c.LineKind {
			case "project_wide":
				c.Category = *in.Assign
			default:
				c.PackageID, c.PackageStageID = *in.Assign, ""
			}
		}
		if e := costs.ValidateContent(c); e != nil {
			return e
		}
		if e := s.costTargets(ctx, tx, org, project, p.ID, item.ID, c); e != nil {
			return e
		}
		if e := writeCostContent(ctx, tx, org, project, p.ID, actor, item.ID, c, true); e != nil {
			return e
		}
	}
	return nil
}

// duplicateCostItems adds a copy of each line straight after it: its label
// marked as a copy, its category or package, quantities and budget
// allowance. Never its contract or what has been claimed against it: those
// belong to the original's award and invoices. A group line is not copied;
// its lines are.
func (s *Store) duplicateCostItems(ctx context.Context, tx pgx.Tx, org, project, actor string, p costs.Plan, picked map[string]bool) error {
	sections := costs.CostSections(p.Items)
	copies := map[string]string{} // original id -> copy id
	for _, item := range p.Items {
		if !picked[item.ID] {
			continue
		}
		if !item.Posting {
			return costs.ErrInvalid
		}
		c := editableContent(item)
		if c.WorkItemID != "" {
			// The copy is the user's own line for the same system, not a
			// second claim on the work item.
			c.WorkItemID, c.SystemID = "", item.SystemID
		}
		c.Label = copyLabel(c.Label)
		c.Code = ""
		c.Origin = "user"
		switch c.Meaning {
		case "stated", "requirement", "allowance", "forecast":
		default:
			c.Meaning = "allowance"
		}
		id := newID()
		if e := s.insertCostItem(ctx, tx, org, project, p.ID, actor, id, c); e != nil {
			return e
		}
		if b, ok := item.Values["budget"]; ok {
			v := costs.Value{Amount: b.Amount, Low: b.Low, High: b.High, ValueState: b.ValueState, AsOf: b.AsOf,
				Origin: "user", Meaning: "allowance", Rationale: "Copied from " + item.Label}
			if e := writeCostValue(ctx, tx, org, project, p.ID, id, actor, "budget", v); e != nil {
				return e
			}
		}
		copies[item.ID] = id
	}
	// Each copy sits straight after its original among its siblings.
	done := map[string]bool{}
	for _, item := range p.Items {
		if copies[item.ID] == "" || done[item.ParentItemID+"|"+sections[item.ID]] {
			continue
		}
		set := item.ParentItemID + "|" + sections[item.ID]
		done[set] = true
		var order []string
		for _, sib := range p.Items {
			if sib.ParentItemID != item.ParentItemID || sections[sib.ID] != sections[item.ID] {
				continue
			}
			order = append(order, sib.ID)
			if cp := copies[sib.ID]; cp != "" {
				order = append(order, cp)
			}
		}
		if _, e := tx.Exec(ctx, `UPDATE cost_item_revisions r SET provenance=r.provenance||jsonb_build_object('display_order',o.position::int,'order_actor',$4::text),version=r.version+1 FROM unnest($5::text[]) WITH ORDINALITY AS o(id,position) WHERE r.org_id=$1::uuid AND r.project_id=$2::uuid AND r.plan_version_id=$3::uuid AND r.cost_item_id::text=o.id`, org, project, p.ID, actor, order); e != nil {
			return e
		}
	}
	return nil
}

// copyLabel marks a copy, within the label limit.
func copyLabel(label string) string {
	const suffix = " (copy)"
	if len(label)+len(suffix) > 500 {
		label = label[:500-len(suffix)]
	}
	return label + suffix
}

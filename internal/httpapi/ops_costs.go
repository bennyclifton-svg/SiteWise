package httpapi

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"sitewise/internal/costs"
	"sitewise/internal/figures"
	"sitewise/internal/store"
)

// Cost Plan operations. Each reads the current draft itself for the plan
// version, so a caller never sends one, and calls the store write its
// screen route calls.

var costSections = []string{"construction", "consultants", "contingency", "authority"}

func costOperations() []operation {
	return []operation{
		{Name: "cost_plan.add_line", About: "Add one cost line. A line with no amount is added with its amount unknown.",
			Fields:  `label, or labels (a list) for several lines in the same place with no amount; section (construction|consultants|contingency|authority); system (construction: a system id); package (consultants: a package id); category (contingency: contingency|escalation|exclusions|other); amount (optional budget, ex or inc GST as the plan is)`,
			Figures: map[string]string{"amount": "money"}, Routes: []string{"POST /projects/{id}/cost-plan/items"}, Run: costAddLine},
		{Name: "cost_plan.edit_line", About: "Rename one cost line.",
			Fields: `line (id); label`, Routes: []string{"PATCH /projects/{id}/cost-plan/items/{item}"}, Run: costEditLine},
		{Name: "cost_plan.set_amount", About: "Set or clear a line's budget or contract amount.",
			Fields:  `line (id); metric (budget|contract); amount, or unknown: true to clear it`,
			Figures: map[string]string{"amount": "money"}, Routes: []string{"PUT /projects/{id}/cost-plan/items/{item}/values/{metric}"}, Run: costSetAmount},
		{Name: "cost_plan.delete_lines", About: "Delete cost lines. A group line takes its lines with it.",
			Fields: `lines (ids)`, Routes: []string{"DELETE /projects/{id}/cost-plan/items/{item}"}, Hold: costDeleteHold, Run: costBulk("delete")},
		{Name: "cost_plan.duplicate_lines", About: "Copy cost lines; each copy is named \"(copy)\".",
			Fields: `lines (ids)`, Run: costBulk("duplicate")},
		{Name: "cost_plan.set_lines", About: "Take lines out of the totals or put them back, or move them to a category or package.",
			Fields: `lines (ids); excluded (true takes them out of totals); assign (a category, or a package id)`,
			Routes: []string{"POST /projects/{id}/cost-plan/items/bulk"}, Run: costBulk("set")},
		{Name: "cost_plan.freeze", About: "freeze the cost plan as a baseline.", Click: true,
			Fields: `(none)`, Routes: []string{"POST /projects/{id}/cost-plan/baseline"}, Run: costFreeze},
	}
}

func draftPlan(ctx context.Context, d Deps, who opActor, project string) (costs.Plan, error) {
	return d.Store.ReadCostPlan(ctx, who.OrgID, project, "")
}

func costLine(p costs.Plan, ref string) (costs.Item, error) {
	return byRef(p.Items, func(i costs.Item) string { return i.ID }, ref, "cost line")
}

// dollars writes an amount for a summary: "$45,000".
func dollars(m costs.Money) string { return figures.Dollars(string(m)) }

// typedAmount reads an amount field as the PM wrote it.
func typedAmount(field, s string) (costs.Money, error) {
	a, ok := figures.Amount(s)
	if !ok || strings.HasPrefix(a, "-") {
		return "", refuse(field, "%q is not an amount. Write it as a number, for example 45000.", s)
	}
	return costs.Money(a), nil
}

func costAddLine(ctx context.Context, d Deps, who opActor, project string, raw json.RawMessage) (string, error) {
	var f struct {
		Label    string   `json:"label"`
		Labels   []string `json:"labels"`
		Section  string   `json:"section"`
		System   string   `json:"system"`
		Package  string   `json:"package"`
		Category string   `json:"category"`
		Amount   *string  `json:"amount"`
	}
	if err := decodeFields(raw, &f); err != nil {
		return "", err
	}
	// labels adds several lines to the same place in one operation, so
	// "add stone and tapware" is short to write and quick to apply.
	labels := f.Labels
	if f.Label != "" {
		labels = append([]string{f.Label}, labels...)
	}
	for i := range labels {
		if labels[i] = strings.TrimSpace(labels[i]); labels[i] == "" {
			return "", refuse("label", "A cost line needs a name.")
		}
	}
	switch {
	case len(labels) == 0:
		return "", refuse("label", "A cost line needs a name.")
	case len(labels) > 1 && f.Amount != nil:
		return "", refuse("amount", "An amount belongs to one line. Send the line with an amount on its own.")
	case len(labels) > 30:
		return "", refuse("labels", "Add at most 30 lines in one operation.")
	}
	p, err := draftPlan(ctx, d, who, project)
	if err != nil {
		return "", err
	}
	c := costs.Content{Label: labels[0], Posting: true, Origin: "user", Meaning: "allowance"}
	where := ""
	section := strings.ToLower(f.Section)
	if section == "" {
		switch {
		case f.System != "":
			section = "construction"
		case f.Package != "":
			section = "consultants"
		case f.Category == "authority_fees":
			section = "authority"
		case f.Category != "":
			section = "contingency"
		}
	}
	switch section {
	case "construction":
		var names []string
		for _, s := range p.Systems {
			names = append(names, s.ID+" ("+s.Label+")")
			if strings.EqualFold(f.System, s.ID) || strings.EqualFold(f.System, s.Label) {
				c.SystemID, where = s.ID, "Construction ("+s.Label+")"
			}
		}
		if c.SystemID == "" {
			return "", refuse("system", "A construction line belongs to one system: %s.", strings.Join(names, ", "))
		}
		c.LineKind = "works"
	case "consultants":
		pkgs, err := d.Store.ReadPackages(ctx, who.OrgID, project)
		if err != nil {
			return "", err
		}
		pkg, err := packageByRef(pkgs, f.Package)
		if err != nil {
			return "", refuse("package", "A consultant line belongs to a package. %s", err.Error())
		}
		c.LineKind, c.PackageID, where = "fee", pkg.ID, "Consultants ("+pkg.Title+")"
	case "contingency":
		c.LineKind, c.Category, where = "project_wide", f.Category, "Contingency"
		if c.Category == "" {
			c.Category = "contingency"
		}
		if c.Category == "authority_fees" {
			return "", refuse("category", "Authority fees go in section authority.")
		}
	case "authority":
		c.LineKind, c.Category, where = "project_wide", "authority_fees", "Authority fees"
	default:
		return "", refuse("section", "Sections are %s.", strings.Join(costSections, ", "))
	}
	in := store.CostItemInput{CostWrite: store.CostWrite{PlanVersionID: p.ID, Version: p.Version}, Content: c}
	amount := "amount unknown"
	if f.Amount != nil {
		m, err := typedAmount("amount", *f.Amount)
		if err != nil {
			return "", err
		}
		in.Values = map[string]costs.Value{"budget": {Amount: &m, ValueState: "known", Origin: "user", Meaning: "allowance"}}
		amount = "budget " + dollars(m)
	}
	var names []string
	for _, label := range labels {
		in.Content.Label = label
		p, err = d.Store.CreateCostItem(ctx, who.OrgID, project, who.UserID, in)
		if err != nil {
			return "", err
		}
		// Each write moves the plan on; the next one names the new version.
		in.CostWrite = store.CostWrite{PlanVersionID: p.ID, Version: p.Version}
		names = append(names, `"`+label+`"`)
	}
	return fmt.Sprintf("Added %s to %s, %s.", strings.Join(names, ", "), where, amount), nil
}

func costEditLine(ctx context.Context, d Deps, who opActor, project string, raw json.RawMessage) (string, error) {
	var f struct {
		Line  string `json:"line"`
		Label string `json:"label"`
	}
	if err := decodeFields(raw, &f); err != nil {
		return "", err
	}
	if strings.TrimSpace(f.Label) == "" {
		return "", refuse("label", "A cost line needs a name.")
	}
	p, err := draftPlan(ctx, d, who, project)
	if err != nil {
		return "", err
	}
	item, err := costLine(p, f.Line)
	if err != nil {
		return "", err
	}
	c := item.Content
	old := c.Label
	c.Label = strings.TrimSpace(f.Label)
	// The grid does the same: a calculated line that is edited by hand
	// becomes an assumption the user owns.
	if c.Origin == "calculation" {
		c.Origin = "assumption"
	}
	in := store.CostItemInput{CostWrite: store.CostWrite{PlanVersionID: p.ID, Version: p.Version}, Content: c}
	if _, err := d.Store.PatchCostItem(ctx, who.OrgID, project, item.ID, who.UserID, in); err != nil {
		return "", err
	}
	return fmt.Sprintf("Renamed %q to %q.", old, c.Label), nil
}

func costSetAmount(ctx context.Context, d Deps, who opActor, project string, raw json.RawMessage) (string, error) {
	var f struct {
		Line    string  `json:"line"`
		Metric  string  `json:"metric"`
		Amount  *string `json:"amount"`
		Unknown bool    `json:"unknown"`
	}
	if err := decodeFields(raw, &f); err != nil {
		return "", err
	}
	metric, meaning := "budget", "allowance"
	switch strings.ToLower(f.Metric) {
	case "", "budget":
	case "contract", "commitment":
		metric, meaning = "commitment", "stated"
	default:
		return "", refuse("metric", "Chat sets a line's budget or contract amount. Claimed to date comes from invoices.")
	}
	if (f.Amount == nil) == !f.Unknown {
		return "", refuse("amount", "Give an amount, or unknown: true to clear it.")
	}
	p, err := draftPlan(ctx, d, who, project)
	if err != nil {
		return "", err
	}
	item, err := costLine(p, f.Line)
	if err != nil {
		return "", err
	}
	current := item.Values[metric]
	v := costs.Value{ValueState: "unknown", Origin: "user", Meaning: meaning, Version: current.Version}
	said := "unknown"
	if f.Amount != nil {
		m, err := typedAmount("amount", *f.Amount)
		if err != nil {
			return "", err
		}
		v.Amount, v.ValueState, said = &m, "known", dollars(m)
	}
	in := store.CostValueInput{PlanVersionID: p.ID, PlanVersion: p.Version, Value: v}
	if _, err := d.Store.PutCostValue(ctx, who.OrgID, project, item.ID, who.UserID, metric, in); err != nil {
		return "", err
	}
	name := map[string]string{"budget": "budget", "commitment": "contract amount"}[metric]
	return fmt.Sprintf("Set the %s of %q to %s.", name, item.Label, said), nil
}

type costLinesFields struct {
	Lines    []string `json:"lines"`
	Excluded *bool    `json:"excluded"`
	Assign   *string  `json:"assign"`
}

func costLines(p costs.Plan, refs []string) ([]costs.Item, error) {
	if len(refs) == 0 {
		return nil, refuse("lines", "Name at least one line by its id.")
	}
	var out []costs.Item
	for _, ref := range refs {
		item, err := costLine(p, ref)
		if err != nil {
			return nil, err
		}
		out = append(out, item)
	}
	return out, nil
}

func costBulk(action string) func(context.Context, Deps, opActor, string, json.RawMessage) (string, error) {
	return func(ctx context.Context, d Deps, who opActor, project string, raw json.RawMessage) (string, error) {
		var f costLinesFields
		if err := decodeFields(raw, &f); err != nil {
			return "", err
		}
		if action != "set" && (f.Excluded != nil || f.Assign != nil) {
			return "", refuse("", "This operation takes only lines.")
		}
		if action == "set" && f.Excluded == nil && f.Assign == nil {
			return "", refuse("excluded", "Say what to set: excluded, or assign.")
		}
		p, err := draftPlan(ctx, d, who, project)
		if err != nil {
			return "", err
		}
		items, err := costLines(p, f.Lines)
		if err != nil {
			return "", err
		}
		in := store.CostBulkInput{CostWrite: store.CostWrite{PlanVersionID: p.ID, Version: p.Version}, Op: action, Excluded: f.Excluded, Assign: f.Assign}
		var labels []string
		for _, i := range items {
			in.IDs = append(in.IDs, i.ID)
			labels = append(labels, `"`+i.Label+`"`)
		}
		if _, err := d.Store.BulkCostItems(ctx, who.OrgID, project, who.UserID, in); err != nil {
			return "", err
		}
		names := strings.Join(labels, ", ")
		switch {
		case action == "delete":
			return "Deleted " + names + ".", nil
		case action == "duplicate":
			return "Copied " + names + ".", nil
		case f.Excluded != nil && *f.Excluded:
			return "Took " + names + " out of the totals.", nil
		case f.Excluded != nil:
			return "Put " + names + " back in the totals.", nil
		}
		return "Moved " + names + ".", nil
	}
}

// costDeleteHold makes a delete wait for a click when it is large, when it
// empties the plan, or when it touches a line with a contract or claimed
// amount (review change 4).
func costDeleteHold(ctx context.Context, d Deps, who opActor, project string, raw json.RawMessage) (string, error) {
	var f costLinesFields
	if err := decodeFields(raw, &f); err != nil {
		return "", err
	}
	p, err := draftPlan(ctx, d, who, project)
	if err != nil {
		return "", err
	}
	items, err := costLines(p, f.Lines)
	if err != nil {
		return "", err
	}
	doomed := map[string]bool{}
	for _, i := range items {
		doomed[i.ID] = true
	}
	// A group line takes its lines with it, so count those too.
	for grew := true; grew; {
		grew = false
		for _, i := range p.Items {
			if !doomed[i.ID] && doomed[i.ParentItemID] {
				doomed[i.ID], grew = true, true
			}
		}
	}
	for _, i := range p.Items {
		if !doomed[i.ID] {
			continue
		}
		for _, metric := range []string{"commitment", "claimed_to_date"} {
			if v := i.Values[metric]; v.ValueState == "known" {
				return fmt.Sprintf("%q has a contract or claimed amount, so deleting it waits for your click.", i.Label), nil
			}
		}
	}
	switch {
	case len(doomed) == len(p.Items) && len(doomed) > 1:
		return fmt.Sprintf("This deletes every line of the cost plan (%d), so it waits for your click.", len(doomed)), nil
	case len(doomed) > 3:
		// The plan says more than ten. The live run showed "clear the plan"
		// arriving as a delete of most lines, a few at a time, so the line
		// is drawn lower: four or more wait for a click.
		return fmt.Sprintf("This deletes %d lines, so it waits for your click.", len(doomed)), nil
	case len(p.Items) >= 4 && len(doomed)*2 >= len(p.Items):
		return fmt.Sprintf("This deletes %d of the plan's %d lines, so it waits for your click.", len(doomed), len(p.Items)), nil
	}
	return "", nil
}

func costFreeze(ctx context.Context, d Deps, who opActor, project string, raw json.RawMessage) (string, error) {
	var f struct{}
	if err := decodeFields(raw, &f); err != nil {
		return "", err
	}
	p, err := draftPlan(ctx, d, who, project)
	if err != nil {
		return "", err
	}
	if _, err := d.Store.BaselineCostPlan(ctx, who.OrgID, project, who.UserID, store.CostWrite{PlanVersionID: p.ID, Version: p.Version}); err != nil {
		return "", err
	}
	return "Froze the cost plan as a baseline.", nil
}

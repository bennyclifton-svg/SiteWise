package httpapi

import (
	"context"
	"errors"
	"sort"
	"strings"

	"sitewise/internal/profile"
	"sitewise/internal/reports"
	"sitewise/internal/store"
)

// Named filters (docs/plans/2026-11-18-chat-agency.md, Task 4). For a set
// that is a rule, not a judgement, the model names the filter and Go
// computes the rows. Where a filter is an attention rule it reads
// reports.Attention itself (attentionRule), so chat, the report and the
// screen cannot disagree. The model supplies no amount and no date.

// namedFilter is one rule-based set on one table.
type namedFilter struct {
	rows func(ctx context.Context, d Deps, who opActor, project string) ([]string, error)
	// shown follows the count on the "Showing" line, written to read after
	// both "1 cost line" and "6 cost lines".
	shown string
	// none is the sentence when no row matches, written by hand.
	none string
}

// filterRefusal is a filter that cannot be computed now, in words for the
// PM ("the Program has a loop").
type filterRefusal string

func (f filterRefusal) Error() string { return string(f) }

// filterNames lists a table's filters, sorted, so the tool description
// stays the same from one request to the next.
func filterNames(table string) []string {
	var names []string
	for name := range viewFilters[table] {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

// filterList is every table's filters for the view tool's description.
func filterList() string {
	var parts []string
	for _, table := range viewTableNames {
		if names := filterNames(table); len(names) > 0 {
			parts = append(parts, table+": "+strings.Join(names, ", "))
		}
	}
	return strings.Join(parts, "; ")
}

// Cost lines.

// costNoBudget is the lines Summarize counts as having no budget amount:
// attention's cost.no_budget rule, read from reports.Attention.
var costNoBudget = attentionRule(attentionCosts, reports.RuleCostNoBudget)

// costOverBudget is the rows the Cost Plan tags "over budget": claimed to
// date (invoices, ex GST) above the budget, from the invoice ledger the grid
// reads. It is attention's cost.line_claimed_over_budget rule, not the
// section rule (contract over budget), in the table's order.
func costOverBudget(ctx context.Context, d Deps, who opActor, project string) ([]string, error) {
	ids, err := attentionRule(attentionLedger, reports.RuleLineClaimedOver)(ctx, d, who, project)
	if err != nil || len(ids) == 0 {
		return nil, err
	}
	over := map[string]bool{}
	for _, id := range ids {
		over[id] = true
	}
	return inTableOrder(ctx, d, who, project, "cost_lines", over)
}

// costExcluded is the lines the grid tags "excluded".
func costExcluded(ctx context.Context, d Deps, who opActor, project string) ([]string, error) {
	p, err := d.Store.ReadCostPlan(ctx, who.OrgID, project, "")
	if errors.Is(err, store.ErrNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var ids []string
	for _, i := range p.Items {
		if i.Excluded {
			ids = append(ids, i.ID)
		}
	}
	return ids, nil
}

// Documents.

// profileReadsDoc is whether the profile reads a document: the register's
// "excluded from profile reading" filter and the Profile's "filed but not
// read for the profile" count. The zero policy reads every document unless
// it is set to skip, as the store's readableSQL does.
func profileReadsDoc(p profile.ReadPolicy, v store.DocumentView) bool {
	if !p.Loaded() {
		return v.ProfileRead != profile.ReadSkip
	}
	kind := ""
	for _, f := range v.Fields {
		if f.Field == "kind" {
			kind = f.Value
		}
	}
	return p.Reads(kind, v.ProfileRead)
}

// supersededBy maps each superseded document to the one that replaces it,
// from the documents' own supersedes link. Chat's document list uses it.
func supersededBy(views []store.DocumentView) map[string]string {
	in := map[string]bool{}
	for _, v := range views {
		in[v.ID] = true
	}
	out := map[string]string{}
	for _, v := range views {
		if v.SupersedesID != "" && in[v.SupersedesID] {
			out[v.SupersedesID] = v.ID
		}
	}
	return out
}

func isNotRead(d Deps, v store.DocumentView, _ map[string]string) bool {
	return !profileReadsDoc(d.ProfileReading, v)
}

func isSuperseded(_ Deps, v store.DocumentView, old map[string]string) bool { return old[v.ID] != "" }

// docFilter is a documents filter from one rule over the register's rows.
func docFilter(keep func(d Deps, v store.DocumentView, old map[string]string) bool) func(context.Context, Deps, opActor, string) ([]string, error) {
	return func(ctx context.Context, d Deps, who opActor, project string) ([]string, error) {
		views, err := d.Store.ProjectDocumentViews(ctx, who.OrgID, project)
		if err != nil {
			return nil, err
		}
		old := supersededBy(views)
		var ids []string
		for _, v := range views {
			if keep(d, v, old) {
				ids = append(ids, v.ID)
			}
		}
		return ids, nil
	}
}

// Program rows.

func programFilter(keep func(st store.ProgramState) (map[string]bool, error)) func(context.Context, Deps, opActor, string) ([]string, error) {
	return func(ctx context.Context, d Deps, who opActor, project string) ([]string, error) {
		st, err := d.Store.LoadProgram(ctx, who.OrgID, project)
		if errors.Is(err, store.ErrNotFound) {
			return nil, nil
		}
		if err != nil {
			return nil, err
		}
		set, err := keep(st)
		if err != nil {
			return nil, err
		}
		var ids []string
		for _, a := range st.Activities {
			if set[a.ID] {
				ids = append(ids, a.ID)
			}
		}
		return ids, nil
	}
}

// programCritical is the rows the engine marks critical (total float of
// zero or less), the Program's own critical flag and the export's Critical
// column. A row whose dates are unknown is not on it, and a Program with a
// loop has no critical path at all.
func programCritical(st store.ProgramState) (map[string]bool, error) {
	if st.Computed.Cycle != nil {
		return nil, filterRefusal("The Program has a loop in its links, so it has no critical path to show.")
	}
	set := map[string]bool{}
	for _, t := range st.Computed.Activities {
		if !t.Unknown && t.Critical {
			set[t.ID] = true
		}
	}
	return set, nil
}

// Packages.

// packageNoQuote is the live packages with no quote received, whether or not
// firms are invited. It counts with attention's PackageQuotes, but it is a
// wider set than attention's procurement.invited_no_quote, which needs a
// firm invited.
func packageNoQuote(ctx context.Context, d Deps, who opActor, project string) ([]string, error) {
	in, err := loadAttention(ctx, d, who, project, attentionPackages)
	if err != nil {
		return nil, err
	}
	var ids []string
	for _, p := range in.Packages {
		if _, quoted, _ := reports.PackageQuotes(p.Firms); quoted == 0 {
			ids = append(ids, p.ID)
		}
	}
	return ids, nil
}

// inTableOrder lists the ids in set in the table's own order, dropping any
// that are not rows of this project's table.
func inTableOrder(ctx context.Context, d Deps, who opActor, project, table string, set map[string]bool) ([]string, error) {
	rows, err := tableRows(ctx, d, who, project, table)
	if err != nil {
		return nil, err
	}
	var ids []string
	for _, id := range rows {
		if set[id] {
			ids = append(ids, id)
		}
	}
	return ids, nil
}

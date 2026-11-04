package invoices

import "sitewise/internal/costs"

// GroupSummary is a display-only rollup. It never enters ledger totals or
// allocation choices; invoices remain attached to their original activity.
type GroupSummary struct {
	LineSummary
	LineIDs []string
}

func GroupSummaries(items []costs.Item, ledger []Invoice, taxBasis string) map[string]GroupSummary {
	rollups := costs.GroupRollups(items)
	out := map[string]GroupSummary{}
	ancestors := map[string][]string{}
	for _, item := range items {
		r, ok := rollups[item.ID]
		if !ok {
			continue
		}
		g := GroupSummary{LineSummary: LineSummary{ID: item.ID, Label: item.Label}, LineIDs: append([]string{item.ID}, r.LineIDs...)}
		if b := r.Values["budget"].Amount; b != nil {
			if n, ok, err := ParseCents(string(*b)); ok && err == nil {
				g.Budget = &n
			}
		}
		out[item.ID] = g
		for _, id := range g.LineIDs {
			ancestors[id] = append(ancestors[id], item.ID)
		}
	}
	for _, inv := range ledger {
		if !Counted(inv) {
			continue
		}
		seen := map[string]bool{}
		for _, a := range Shares(inv) {
			if a.Amount == nil {
				continue
			}
			for _, id := range ancestors[a.LineID] {
				g := out[id]
				g.Invoiced += *a.Amount
				if !seen[id] {
					g.Invoices++
					seen[id] = true
				}
				out[id] = g
			}
		}
	}
	for id, g := range out {
		if g.Budget != nil && taxBasis != "inc_tax" {
			r := *g.Budget - g.Invoiced
			g.Remaining = &r
			g.Percent = percent(g.Invoiced, *g.Budget)
			g.OverBudget = g.Invoiced > *g.Budget
		}
		out[id] = g
	}
	return out
}

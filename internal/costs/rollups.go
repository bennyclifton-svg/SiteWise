package costs

// Rollup is a display summary, never an additional posting line.
type Rollup struct {
	Values map[string]Total `json:"values"`
	Derived
	LineIDs []string `json:"line_ids"`
}

// GroupRollups retains the same budget ownership as Summarize. Walking each
// line's ancestors also handles nested groups without counting an intermediate
// summary as another cost. Broken/cyclic legacy ancestry terminates safely.
func GroupRollups(items []Item) map[string]Rollup {
	byID := map[string]Item{}
	out := map[string]Rollup{}
	for _, i := range items {
		byID[i.ID] = i
	}
	for _, i := range items {
		if p, ok := byID[i.ParentItemID]; ok && !p.Posting {
			out[p.ID] = Rollup{Values: emptyTotals(), LineIDs: []string{}}
		}
	}
	budgetOwners := budgetGroups(items)
	for _, i := range items {
		seen := map[string]bool{}
		for id := i.ID; id != "" && !seen[id]; {
			seen[id] = true
			ancestor, ok := byID[id]
			if !ok {
				break
			}
			if r, ok := out[id]; ok {
				if i.Posting {
					r.LineIDs = append(r.LineIDs, i.ID)
				}
				if !i.Excluded {
					// A retained budget is independent of its child contracts.
					if i.Posting {
						addMetrics(r.Values, i, budgetOwners[i.ID] != "")
					} else if _, owns := i.Values["budget"]; owns {
						addOnly(r.Values, "budget", i)
					}
				}
				out[id] = r
			}
			id = ancestor.ParentItemID
		}
	}
	for id, r := range out {
		for metric, t := range r.Values {
			if t.Lines == 0 {
				t.Amount = nil
				t.Low = nil
				t.High = nil
				t.Complete = false
				r.Values[metric] = t
			}
		}
		r.Derived = Derived{BudgetVariance: diff(r.Values["budget"].Amount, r.Values["forecast"].Amount), Remaining: diff(r.Values["forecast"].Amount, r.Values["claimed_to_date"].Amount)}
		out[id] = r
	}
	return out
}

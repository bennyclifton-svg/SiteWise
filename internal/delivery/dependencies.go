package delivery

type Dependency struct {
	PredecessorID string `json:"predecessor_id"`
	SuccessorID   string `json:"successor_id"`
	Type          string `json:"type"`
	LagDays       int    `json:"lag_days"`
}

// Cycle returns the newly closed path, including the repeated first node.
func Cycle(edges []Dependency, from, to string) []string {
	if from == to {
		return []string{from, to}
	}
	adj := map[string][]string{}
	for _, e := range edges {
		adj[e.PredecessorID] = append(adj[e.PredecessorID], e.SuccessorID)
	}
	seen := map[string]bool{}
	var visit func(string, []string) []string
	visit = func(at string, path []string) []string {
		if at == from {
			return append(path, at)
		}
		if seen[at] {
			return nil
		}
		seen[at] = true
		for _, next := range adj[at] {
			if p := visit(next, append(path, at)); p != nil {
				return p
			}
		}
		return nil
	}
	if path := visit(to, nil); path != nil {
		return append([]string{from}, path...)
	}
	return nil
}

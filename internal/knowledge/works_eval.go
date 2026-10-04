package knowledge

import "fmt"

// Truth is a three-valued predicate result: unknown never hides a record.
type Truth int

const (
	Unknown Truth = iota
	True
	False
)

func (t Truth) String() string {
	switch t {
	case True:
		return "true"
	case False:
		return "false"
	}
	return "unknown"
}

// WorkItem is what a `works` predicate reads of one in-scope work item.
// Which items are in scope is the caller's rule (D-10), not this package's.
type WorkItem struct {
	System, Action string
}

// WorksEnv is everything a works-layer predicate may read.
type WorksEnv struct {
	// Values are known determinant values by id (comma-separated for
	// multi-choice), as in Relevant.
	Values map[string]string
	// Present reports system_present: the system is in the completed building.
	Present func(system string) bool
	// Items are the in-scope work items. An empty list makes every `works`
	// condition false: no works touch anything.
	Items []WorkItem
	// Existing reports system_existing: on the site and not being replaced.
	// Nil makes it unknown.
	Existing func(system string) Truth
}

// Holds evaluates a works-layer predicate (consequence or unforeseen `when`,
// or an applies_when) in code. It never calls Jev.
func (c *Catalog) Holds(p any, env WorksEnv) Truth {
	e := predEnv{values: env.Values, present: env.Present,
		works: func(cond map[string]any) tri { return c.worksMatch(cond, env.Items) }}
	if env.Existing != nil {
		e.existing = func(sys string) tri { return toTri(env.Existing(sys)) }
	}
	return fromTri(evalPredicate(p, e))
}

// worksMatch is true when an item has one of the actions (if listed) on one
// of the systems (if listed). An item on a listed system or one of its
// descendants matches. An item on an ancestor of a listed system (a coarse
// item on a whole family) might be the listed system, so it is unknown.
func (c *Catalog) worksMatch(cond map[string]any, items []WorkItem) tri {
	actions, systems := asList(cond["action"]), asList(cond["system"])
	result := triFalse
	for _, it := range items {
		if len(actions) > 0 && !listHas(actions, it.Action) {
			continue
		}
		if len(systems) == 0 {
			return triTrue
		}
		for _, s := range systems {
			listed := fmt.Sprint(s)
			switch {
			case c.covers(it.System, listed):
				return triTrue
			case c.covers(listed, it.System):
				result = triUnknown
			}
		}
	}
	return result
}

func listHas(list []any, v string) bool {
	for _, x := range list {
		if fmt.Sprint(x) == v {
			return true
		}
	}
	return false
}

func toTri(t Truth) tri {
	switch t {
	case True:
		return triTrue
	case False:
		return triFalse
	}
	return triUnknown
}

func fromTri(t tri) Truth {
	switch t {
	case triTrue:
		return True
	case triFalse:
		return False
	}
	return Unknown
}

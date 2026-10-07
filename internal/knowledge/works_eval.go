package knowledge

import (
	"fmt"
	"strings"
)

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
	LayoutChange   string
}

// WorksEnv is everything a works-layer predicate may read.
type WorksEnv struct {
	// Values are known determinant values by id (comma-separated for
	// multi-choice), as in Relevant.
	Values map[string]string
	// WorkTypes contains the applied project and part types, supplied by code
	// (D-07). A stray determinant value cannot supply this input.
	WorkTypes []string
	// Present reports system_present: the system is in the completed building.
	// Nil makes it unknown.
	Present func(system string) bool
	// PresentState preserves unknown completed-building presence. When supplied
	// it takes precedence over the legacy boolean callback.
	PresentState func(system string) Truth
	// Items are the in-scope work items. An empty list makes every `works`
	// condition false: no works touch anything.
	Items []WorkItem
	// Existing reports the site evidence only. Live items below can establish
	// existence or override it with removal/replacement (D-10).
	Existing func(system string) Truth
	memo     *worksMemo
}

// Holds evaluates a works-layer predicate (consequence or unforeseen `when`,
// or an applies_when) in code. It never calls Jev.
func (c *Catalog) Holds(p any, env WorksEnv) Truth {
	// Override only the code-fed work type; copying every determinant for
	// every catalogue predicate creates avoidable work on profile edits.
	workType := ""
	if env.memo != nil {
		workType = env.memo.workType
	} else {
		workType = c.workTypeValue(env.WorkTypes)
	}
	e := predEnv{values: env.Values, workType: &workType, present: env.Present,
		works: func(cond map[string]any) tri { return c.matchInEnv(cond, env) }}
	if env.PresentState != nil {
		e.presentState = func(system string) tri { return toTri(env.PresentState(system)) }
	}
	e.existing = func(sys string) tri { return toTri(c.SystemExisting(sys, env)) }
	return fromTri(evalPredicate(p, e))
}

func (c *Catalog) workTypeValue(types []string) string {
	for _, value := range types {
		known := false
		for _, option := range c.profile.taxonomy.WorkTypes {
			known = known || value == option.ID
		}
		if !known {
			return "" // malformed code input must not hide a relevant record
		}
	}
	return strings.Join(types, ",")
}

// SystemExisting evaluates D-10 for already-filtered live, included items.
// An ancestor item is only possible evidence about a particular child.
// Removing a child cannot prove that its entire family has disappeared.
func (c *Catalog) SystemExisting(system string, env WorksEnv) Truth {
	if env.memo != nil {
		if state, ok := env.memo.existing[system]; ok {
			return state
		}
		state := c.systemExisting(system, env)
		env.memo.existing[system] = state
		return state
	}
	return c.systemExisting(system, env)
}

func (c *Catalog) systemExisting(system string, env WorksEnv) Truth {
	result := Unknown
	if env.Existing != nil {
		result = env.Existing(system)
	}
	ambiguous := false
	possible := false
	for _, item := range env.Items {
		descendant := c.covers(item.System, system)
		ancestor := c.covers(system, item.System)
		if !descendant && !ancestor {
			continue
		}
		switch item.Action {
		case "remove", "replace":
			if ancestor {
				return False
			}
			ambiguous = true
		case "new":
			// New work says nothing about an existing installation.
		case "alter", "upgrade", "repair", "investigate", "retain":
			if descendant {
				result = True
			} else {
				possible = true
			}
		default:
			ambiguous = true // an unresolved action could remove the system
		}
	}
	if ambiguous || (possible && result != True) {
		return Unknown
	}
	return result
}

// worksMatch is true when an item has one of the actions (if listed) on one
// of the systems (if listed). An item on a listed system or one of its
// descendants matches. An item on an ancestor of a listed system (a coarse
// item on a whole family) might be the listed system, so it is unknown.
func (c *Catalog) worksMatch(cond map[string]any, items []WorkItem) tri {
	actions, systems := asList(cond["action"]), asList(cond["system"])
	result := triFalse
	for _, it := range items {
		// An item whose action is not yet resolved might have a listed one.
		actionKnown := it.Action != ""
		if len(actions) > 0 && actionKnown && !listHas(actions, it.Action) {
			continue
		}
		layoutKnown := true
		if required, ok := cond["layout_change"]; ok {
			if it.LayoutChange == "yes" || it.LayoutChange == "no" {
				if it.LayoutChange != predicateText(required) {
					continue
				}
			} else {
				layoutKnown = false
			}
		}
		match := triUnknown
		if (actionKnown || len(actions) == 0) && layoutKnown {
			match = triTrue
		}
		if len(systems) == 0 {
			if match == triTrue {
				return triTrue
			}
			result = triUnknown
			continue
		}
		for _, s := range systems {
			listed := predicateText(s)
			switch {
			case c.covers(it.System, listed) && match == triTrue:
				return triTrue
			case c.covers(it.System, listed), c.covers(listed, it.System):
				result = triUnknown
			}
		}
	}
	return result
}

func listHas(list []any, v string) bool {
	for _, x := range list {
		if predicateText(x) == v {
			return true
		}
	}
	return false
}

// Loaded predicates contain strings. Retain the permissive evaluator behavior
// for hand-built inputs without formatting every catalogue string per item.
func predicateText(value any) string {
	if text, ok := value.(string); ok {
		return text
	}
	return fmt.Sprint(value)
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

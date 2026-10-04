package knowledge

import (
	"fmt"
	"sort"
	"strconv"
	"strings"
)

// Relevance is what a project's scope makes relevant: the rules about its
// systems that are not ruled out by known values, and the determinants those
// rules read, plus determinants tied to its systems and the always-shown core.
type Relevance struct {
	Rules        []string
	Determinants map[string]bool
}

// tri is a three-valued truth: an unknown value never hides a rule.
type tri int

const (
	triUnknown tri = iota
	triTrue
	triFalse
)

// Relevant computes relevance in code from the schema (SCHEMA.md, Rules and
// Predicates; docs/design/2026-10-03-profile-scope.md). scope holds system ids;
// values holds known determinant values by id. A rule is relevant when one of
// its systems covers or is covered by a scope system and its applies_when is
// not false.
func (c *Catalog) Relevant(scope []string, values map[string]string) Relevance {
	in := func(id string) bool {
		for _, s := range scope {
			if c.covers(s, id) || c.covers(id, s) {
				return true
			}
		}
		return false
	}
	out := Relevance{Determinants: map[string]bool{}}
	for _, id := range c.AlwaysShown() {
		out.Determinants[id] = true
	}
	ids := make([]string, 0, len(c.rules))
	for id := range c.rules {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	for _, id := range ids {
		rule := c.rules[id]
		if rule.Status == statusDeprecated {
			continue
		}
		touches := false
		for _, s := range rule.Systems {
			if in(s) {
				touches = true
				break
			}
		}
		if !touches || evalPredicate(rule.AppliesWhen, predEnv{values: values, present: in}) == triFalse {
			continue
		}
		out.Rules = append(out.Rules, id)
		for d := range predicateDeterminants(rule.AppliesWhen) {
			out.Determinants[d] = true
		}
		if rule.Derives != nil {
			for _, d := range rule.Derives.Inputs {
				out.Determinants[d] = true
			}
			out.Determinants[rule.Derives.Gives] = true
		}
	}
	for _, d := range c.profile.determinants {
		for _, s := range d.Systems {
			if in(s) {
				out.Determinants[d.ID] = true
			}
		}
	}
	return out
}

// predEnv is what a predicate can read. A nil works or existing function
// means that input is not available here (for example scope relevance before
// work items exist), so those operators are unknown, never false.
type predEnv struct {
	values   map[string]string
	present  func(string) bool
	works    func(cond map[string]any) tri
	existing func(system string) tri
}

// evalPredicate evaluates an applies_when mapping. Missing values and
// malformed conditions are unknown, so they keep a rule rather than hide it.
func evalPredicate(p any, env predEnv) tri {
	if p == nil {
		return triTrue
	}
	m, ok := asMap(p)
	if !ok {
		return triUnknown
	}
	if det, ok := m["det"]; ok {
		return evalCondition(fmt.Sprint(det), m, env.values)
	}
	result := triTrue
	for key, val := range m {
		var t tri
		switch key {
		case "all":
			t = triTrue
			for _, sub := range asList(val) {
				t = and(t, evalPredicate(sub, env))
			}
		case "any":
			t = triFalse
			for _, sub := range asList(val) {
				t = or(t, evalPredicate(sub, env))
			}
		case "not":
			t = not(evalPredicate(val, env))
		case "system_present":
			t = triFalse
			if env.present != nil && env.present(fmt.Sprint(val)) {
				t = triTrue
			}
		case "works":
			t = triUnknown
			if cond, ok := asMap(val); ok && env.works != nil {
				t = env.works(cond)
			}
		case "system_existing":
			t = triUnknown
			if env.existing != nil {
				t = env.existing(fmt.Sprint(val))
			}
		default:
			t = triUnknown
		}
		result = and(result, t)
	}
	return result
}

// evalCondition compares one determinant's known value. A multi-choice
// value is stored comma-separated; any_of and eq match any of its parts.
func evalCondition(det string, cond map[string]any, values map[string]string) tri {
	raw := strings.TrimSpace(values[det])
	if raw == "" {
		return triUnknown
	}
	parts := strings.Split(raw, ",")
	for i := range parts {
		parts[i] = strings.TrimSpace(parts[i])
	}
	has := func(want string) bool {
		for _, p := range parts {
			if p == want {
				return true
			}
		}
		return false
	}
	for op, arg := range cond {
		switch op {
		case "det":
			continue
		case "any_of":
			for _, want := range asList(arg) {
				if has(fmt.Sprint(want)) {
					return triTrue
				}
			}
			return triFalse
		case "eq":
			return boolTri(has(fmt.Sprint(arg)))
		case "is":
			want, ok := arg.(bool)
			if !ok {
				return triUnknown
			}
			switch raw {
			case "stated_true", "true":
				return boolTri(want)
			case "stated_false", "false":
				return boolTri(!want)
			}
			return triUnknown
		case "gt", "gte", "lt", "lte":
			got, err1 := strconv.ParseFloat(raw, 64)
			limit, err2 := strconv.ParseFloat(fmt.Sprint(arg), 64)
			if err1 != nil || err2 != nil {
				return triUnknown
			}
			switch op {
			case "gt":
				return boolTri(got > limit)
			case "gte":
				return boolTri(got >= limit)
			case "lt":
				return boolTri(got < limit)
			default:
				return boolTri(got <= limit)
			}
		}
	}
	return triUnknown
}

// predicateDeterminants lists the determinant ids a predicate reads.
func predicateDeterminants(p any) map[string]bool {
	out := map[string]bool{}
	var walk func(any)
	walk = func(v any) {
		if m, ok := asMap(v); ok {
			if det, ok := m["det"]; ok {
				out[fmt.Sprint(det)] = true
			}
			for _, sub := range m {
				walk(sub)
			}
			return
		}
		for _, sub := range asList(v) {
			walk(sub)
		}
	}
	walk(p)
	return out
}

func asMap(v any) (map[string]any, bool) {
	switch m := v.(type) {
	case map[string]any:
		return m, true
	case map[any]any:
		out := make(map[string]any, len(m))
		for k, val := range m {
			out[fmt.Sprint(k)] = val
		}
		return out, true
	}
	return nil, false
}

func asList(v any) []any {
	if l, ok := v.([]any); ok {
		return l
	}
	return nil
}

func boolTri(b bool) tri {
	if b {
		return triTrue
	}
	return triFalse
}

func and(a, b tri) tri {
	if a == triFalse || b == triFalse {
		return triFalse
	}
	if a == triUnknown || b == triUnknown {
		return triUnknown
	}
	return triTrue
}

func or(a, b tri) tri {
	if a == triTrue || b == triTrue {
		return triTrue
	}
	if a == triUnknown || b == triUnknown {
		return triUnknown
	}
	return triFalse
}

func not(a tri) tri {
	switch a {
	case triTrue:
		return triFalse
	case triFalse:
		return triTrue
	}
	return triUnknown
}

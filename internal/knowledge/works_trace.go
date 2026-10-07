package knowledge

import (
	"fmt"
	"sort"
)

// WorksTrace captures predicate inputs, including unknowns, so proposals can
// explain decisions without implementing a second predicate language.
type WorksTrace struct {
	Truth        Truth
	Determinants []string
	Existing     map[string]Truth
	Present      map[string]Truth
	ItemIndexes  []int
	Specificity  int
}

func (c *Catalog) TraceWorks(p any, env WorksEnv) WorksTrace {
	return c.traceWorks(p, env, false)
}

// TraceRelevantWorks avoids constructing unused explanations for records a
// false predicate excludes. Unknown predicates still receive the full trace.
func (c *Catalog) TraceRelevantWorks(p any, env WorksEnv) WorksTrace {
	return c.traceWorks(p, env, true)
}

func (c *Catalog) traceWorks(p any, env WorksEnv, relevantOnly bool) WorksTrace {
	out := WorksTrace{Truth: c.Holds(p, env)}
	if relevantOnly && out.Truth == False {
		return out
	}
	out.Existing, out.Present = map[string]Truth{}, map[string]Truth{}
	dets := map[string]bool{}
	items := map[int]bool{}
	walkPredicate(p, func(m map[string]any) {
		if value, ok := m["det"]; ok {
			dets[fmt.Sprint(value)] = true
		}
		if value, ok := m["system_existing"]; ok {
			system := fmt.Sprint(value)
			out.Existing[system] = c.SystemExisting(system, env)
			for i, item := range env.Items {
				if c.covers(item.System, system) || c.covers(system, item.System) {
					items[i] = true
				}
			}
		}
		if value, ok := m["system_present"]; ok {
			system := fmt.Sprint(value)
			state := Unknown
			if env.PresentState != nil {
				state = env.PresentState(system)
			} else if env.Present != nil {
				state = False
				if env.Present(system) {
					state = True
				}
			}
			out.Present[system] = state
		}
		if cond, ok := asMap(m["works"]); ok {
			for i, item := range env.Items {
				if c.worksMatch(cond, []WorkItem{item}) == triFalse {
					continue
				}
				items[i] = true
				specificity := 0
				if item.Action != "" && len(asList(cond["action"])) > 0 {
					specificity = 1
					for _, system := range asList(cond["system"]) {
						id := fmt.Sprint(system)
						if !c.covers(item.System, id) && !c.covers(id, item.System) {
							continue
						}
						score := 2
						if item.System == id && len(c.Children(id)) == 0 {
							score = 3
						}
						if score > specificity {
							specificity = score
						}
					}
				}
				if specificity > out.Specificity {
					out.Specificity = specificity
				}
			}
		}
	})
	for det := range dets {
		out.Determinants = append(out.Determinants, det)
	}
	sort.Strings(out.Determinants)
	for i := range items {
		out.ItemIndexes = append(out.ItemIndexes, i)
	}
	sort.Ints(out.ItemIndexes)
	return out
}

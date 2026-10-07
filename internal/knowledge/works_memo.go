package knowledge

import (
	"encoding/json"
)

type worksClauseTrace struct {
	indexes     []int
	layout      map[int]string
	specificity int
}
type worksAtom struct {
	truth tri
	known bool
	trace *worksClauseTrace
}
type worksMemo struct {
	workType string
	existing map[string]Truth
	atoms    map[string]*worksAtom
}

// MemoizeWorks shares repeated primitive predicates within one immutable part
// evaluation. Construct a fresh environment for every evaluation; never reuse
// it after changing inputs or across concurrent evaluations.
func (c *Catalog) MemoizeWorks(env WorksEnv) WorksEnv {
	env.memo = &worksMemo{workType: c.workTypeValue(env.WorkTypes), existing: map[string]Truth{}, atoms: map[string]*worksAtom{}}
	return env
}

func workAtom(cond map[string]any, env WorksEnv) *worksAtom {
	if env.memo == nil {
		return nil
	}
	raw, err := json.Marshal(cond)
	if err != nil {
		return nil
	}
	key := string(raw)
	if a, ok := env.memo.atoms[key]; ok {
		return a
	}
	a := &worksAtom{}
	env.memo.atoms[key] = a
	return a
}

func (c *Catalog) matchInEnv(cond map[string]any, env WorksEnv) tri {
	a := workAtom(cond, env)
	if a != nil && a.known {
		return a.truth
	}
	truth := c.worksMatch(cond, env.Items)
	if a != nil {
		a.truth = truth
		a.known = true
	}
	return truth
}

func (c *Catalog) traceWorkClause(cond map[string]any, env WorksEnv) worksClauseTrace {
	a := workAtom(cond, env)
	if a != nil && a.trace != nil {
		return *a.trace
	}
	out := worksClauseTrace{layout: map[int]string{}}
	for i, item := range env.Items {
		if c.worksMatch(cond, []WorkItem{item}) == triFalse {
			continue
		}
		out.indexes = append(out.indexes, i)
		if _, ok := cond["layout_change"]; ok {
			value := item.LayoutChange
			if value == "" {
				value = "unknown"
			}
			out.layout[i] = value
		}
		specificity := 0
		if item.Action != "" && len(asList(cond["action"])) > 0 {
			specificity = 1
			for _, system := range asList(cond["system"]) {
				id := predicateText(system)
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
		if specificity > out.specificity {
			out.specificity = specificity
		}
	}
	if a != nil {
		a.trace = &out
	}
	return out
}

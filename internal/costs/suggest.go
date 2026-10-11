package costs

import (
	"fmt"
	"sitewise/internal/knowledge"
	"sort"
	"strings"
)

type SuggestPackage struct {
	ID, Title, Kind, DisciplineID, WorksScope string
	Retired                                   bool
}
type ScopeWork struct {
	ID       string `json:"id"`
	SystemID string `json:"system_id"`
	Title    string `json:"title"`
	PartID   string `json:"part_id"`
}
type Suggestion struct {
	Key       string `json:"source_key"`
	ParentKey string `json:"parent_key"`
	Content
	ResolvedCategory string      `json:"resolved_category"`
	ScopeWorkItems   []ScopeWork `json:"scope_work_items"`
}
type SuggestInput struct {
	Catalog   *knowledge.Catalog
	Packages  []SuggestPackage
	Works     []ScopeWork
	Values    map[string]string
	Existing  []Item
	Dismissed map[string]bool
}

func sortedPackages(in []SuggestPackage) []SuggestPackage {
	p := append([]SuggestPackage(nil), in...)
	sort.Slice(p, func(i, j int) bool {
		if p[i].Title == p[j].Title {
			return p[i].ID < p[j].ID
		}
		return p[i].Title < p[j].Title
	})
	return p
}

// An empty ID with a title is a planning role, not a procurement assignment.
func mappedCostPackage(cat *knowledge.Catalog, system string, packages []SuggestPackage) SuggestPackage {
	trades := []string{}
	if cat != nil {
		for _, el := range cat.CostElements() {
			if cat.SystemWithin(system, el.System) {
				for _, o := range el.Overrides {
					if cat.SystemWithin(system, o.Sub) {
						trades = append(trades, o.Trade)
					}
				}
				trades = append(trades, el.Trade)
				break
			}
		}
	}

	for _, trade := range trades {
		if trade == "trade.main_works" {
			continue
		}
		matches := []SuggestPackage{}
		for _, p := range packages {
			if !p.Retired && p.Kind == "works" && p.DisciplineID == trade {
				matches = append(matches, p)
			}
		}
		if len(matches) > 0 {
			if len(matches) == 1 {
				return matches[0]
			}
			return SuggestPackage{Title: "Unassigned"}
		}
	}
	matches := []SuggestPackage{}
	for _, p := range packages {
		if !p.Retired && p.Kind == "works" && (p.WorksScope == "head_contract" || p.DisciplineID == "trade.main_works") {
			matches = append(matches, p)
		}
	}
	if len(matches) == 1 {
		return matches[0]
	}
	if len(matches) > 1 {
		return SuggestPackage{Title: "Unassigned"}
	}
	return SuggestPackage{Title: "Head contractor"}
}
func mappedPackage(cat *knowledge.Catalog, system string, packages []SuggestPackage) string {
	return mappedCostPackage(cat, system, packages).Title
}

// ResolveCostPackage returns one identity for both the category and package totals.
// Inheritance uses explicit ancestors, so a generic automatic parent cannot hide a trade.
func ResolveCostPackage(cat *knowledge.Catalog, i Item, items []Item, packages []SuggestPackage) (string, string) {
	if i.LineKind == "project_wide" {
		return "", strings.ReplaceAll(i.Category, "_", " ")
	}
	assigned := i.PackageID
	parent := i.ParentItemID
	seen := map[string]bool{i.ID: true}
	for assigned == "" && parent != "" && !seen[parent] {
		seen[parent] = true
		found := false
		for _, row := range items {
			if row.ID == parent {
				assigned = row.PackageID
				parent = row.ParentItemID
				found = true
				break
			}
		}
		if !found {
			break
		}
	}
	if assigned != "" {
		for _, p := range packages {
			if p.ID == assigned {
				label := p.Title
				if p.Retired {
					label += " (retired)"
				}
				return p.ID, label
			}
		}
		return "", "Unassigned"
	}
	if i.LineKind == "works" {
		p := mappedCostPackage(cat, i.SystemID, packages)
		return p.ID, p.Title
	}
	return "", "Unassigned"
}
func ResolveCategory(cat *knowledge.Catalog, i Item, packages []SuggestPackage) string {
	_, label := ResolveCostPackage(cat, i, nil, packages)
	return label
}
func ScopeFor(cat *knowledge.Catalog, system string, works []ScopeWork) []ScopeWork {
	out := []ScopeWork{}
	for _, w := range works {
		if cat.SystemWithin(w.SystemID, system) {
			out = append(out, w)
		}
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].SystemID != out[j].SystemID {
			return out[i].SystemID < out[j].SystemID
		}
		return out[i].ID < out[j].ID
	})
	return out
}

// Suggest derives structure only. Existing posting scope is never split or priced
// implicitly; that requires an explicit subdivision preserving the allowance.
func Suggest(in SuggestInput) []Suggestion {
	out := []Suggestion{}
	cat := in.Catalog
	if cat == nil {
		return out
	}
	packages := sortedPackages(in.Packages)
	existing := map[string]Item{}
	for _, i := range in.Existing {
		if i.SourceKey != "" {
			existing[i.SourceKey] = i
		}
	}
	add := func(s Suggestion) {
		if _, ok := existing[s.Key]; ok {
			return
		}
		if in.Dismissed[s.Key] || (s.ParentKey != "" && in.Dismissed[s.ParentKey]) {
			return
		}
		for _, i := range in.Existing {
			if s.LineKind == "fee" && i.LineKind == "fee" && i.PackageID == s.PackageID {
				return
			}
			if s.LineKind == "works" && i.LineKind == "works" && i.Posting && !strings.HasSuffix(i.SourceKey, ":remaining") && cat.SystemWithin(s.SystemID, i.SystemID) {
				return
			}
		}
		out = append(out, s)
	}
	base := func(key, label, kind, code string) Suggestion {
		return Suggestion{Key: key, Content: Content{Label: label, LineKind: kind, Code: code, Posting: true, Origin: "assumption", Meaning: "allowance"}, ScopeWorkItems: []ScopeWork{}}
	}
	fee := 0
	for _, p := range packages {
		if p.Retired || p.Kind != "services" {
			continue
		}
		fee++
		s := base("fee:"+p.ID, p.Title, "fee", fmt.Sprintf("F%02d", fee))
		s.PackageID = p.ID
		s.ResolvedCategory = p.Title
		s.Rationale = "Suggested from the " + p.Title + " package; no amount assumed."
		add(s)
	}
	for _, el := range cat.CostElements() {
		scope := ScopeFor(cat, el.System, in.Works)
		if len(scope) == 0 {
			continue
		}
		s := base("works:"+el.System, el.Label, "works", el.Code)
		s.SystemID = el.System
		s.ScopeWorkItems = scope
		s.ResolvedCategory = mappedPackage(cat, el.System, packages)
		s.Rationale = fmt.Sprintf("Draft cost structure: %d included work items. Review coverage before pricing.", len(scope))
		// A manually detailed system already has a breakdown; avoid a second total.
		covered := false
		for _, i := range in.Existing {
			if i.SourceKey == "" && i.LineKind == "works" && cat.SystemWithin(i.SystemID, el.System) {
				covered = true
			}
		}
		if covered {
			continue
		}
		split := []knowledge.CostOverride{}
		for _, o := range el.Overrides {
			for _, p := range packages {
				if !p.Retired && p.Kind == "works" && p.DisciplineID == o.Trade && len(ScopeFor(cat, o.Sub, scope)) > 0 {
					split = append(split, o)
					break
				}
			}
		}
		// Retiring a trade package must not turn an already kept breakdown back
		// into a posting total or strand new scope without a remaining-scope line.
		automaticGroup := false
		if parent, ok := existing[s.Key]; ok && !parent.Posting {
			for _, child := range in.Existing {
				if child.ParentItemID == parent.ID && child.SourceKey != "" {
					automaticGroup = true
					for _, o := range el.Overrides {
						if child.SourceKey == "works:"+o.Sub {
							found := false
							for _, old := range split {
								found = found || old.Sub == o.Sub
							}
							if !found {
								split = append(split, o)
							}
						}
					}
				}
			}
		}
		if !el.Itemised && len(split) == 0 && !automaticGroup {
			add(s)
			continue
		}
		if old, ok := existing[s.Key]; ok && old.Posting {
			continue
		}
		s.Posting = false
		children := []Suggestion{}
		addChild := func(system, label, code string, ws []ScopeWork) {
			c := base("works:"+system, label, "works", code)
			c.SystemID = system
			c.ParentKey = s.Key
			c.ScopeWorkItems = ws
			c.ResolvedCategory = mappedPackage(cat, system, packages)
			c.Rationale = fmt.Sprintf("Draft cost structure: %d included work items. Review overlap with other elements before pricing.", len(ws))
			children = append(children, c)
		}
		if el.Itemised {
			for n, sub := range cat.Children(el.System) {
				ws := ScopeFor(cat, sub.ID, scope)
				if len(ws) == 0 {
					continue
				}
				label, costed := sub.Label, true
				for _, x := range el.Items {
					if x.Sub == sub.ID {
						costed = x.Costed
						if x.Label != "" {
							label = x.Label
						}
					}
				}
				if costed {
					addChild(sub.ID, label, fmt.Sprintf("%s.%d", el.Code, n+1), ws)
				}
			}
			// A coarse item is not evidence of particular children. Keep its scope visible.
			coarse := []ScopeWork{}
			for _, w := range scope {
				if w.SystemID == el.System {
					coarse = append(coarse, w)
				}
			}
			if len(coarse) > 0 {
				addChild(el.System, el.Label+" — unspecified scope", el.Code+".0", coarse)
				children[len(children)-1].Key = s.Key + ":remaining"
			}
		} else {
			remaining := []ScopeWork{}
			for _, w := range scope {
				separate := false
				for _, o := range split {
					separate = separate || cat.SystemWithin(w.SystemID, o.Sub)
				}
				if !separate {
					remaining = append(remaining, w)
				}
			}
			if len(remaining) > 0 {
				addChild(el.System, el.Label+" — remaining scope", el.Code+".0", remaining)
				children[len(children)-1].Key = s.Key + ":remaining"
			}
			for n, o := range split {
				sub, _ := cat.System(o.Sub)
				addChild(o.Sub, sub.Label, fmt.Sprintf("%s.%d", el.Code, n+1), ScopeFor(cat, o.Sub, scope))
			}
		}
		if len(children) > 0 {
			add(s)
			for _, c := range children {
				add(c)
			}
		}
	}
	for n, st := range cat.CostStarters() {
		truth := cat.CostStarterHolds(st.AppliesWhen, in.Values)
		if truth == knowledge.False {
			continue
		}
		s := base("pw:"+st.ID, st.Label, "project_wide", fmt.Sprintf("P%02d", n+1))
		s.Category = st.Category
		s.Excluded = st.Category == "exclusions"
		s.ResolvedCategory = strings.ReplaceAll(st.Category, "_", " ")
		s.Rationale = "Draft project-wide allowance; no amount or percentage assumed."
		if truth == knowledge.Unknown {
			s.Rationale += " Applicability is unknown; confirm against the project."
		}
		add(s)
	}
	return out
}

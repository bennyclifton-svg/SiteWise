package procurement

import (
	"fmt"
	"slices"
	"sort"
	"time"

	"sitewise/internal/knowledge"
	"sitewise/internal/works"
)

type Assignment struct {
	ScopeContent
	ID        string     `json:"id"`
	PackageID string     `json:"package_id"`
	RetiredAt *time.Time `json:"retired_at,omitempty"`
}

type Gap struct {
	WorkItemID   string   `json:"work_item_id"`
	State        string   `json:"state"`
	Role         string   `json:"role,omitempty"`
	PackageIDs   []string `json:"package_ids"`
	ScopeItemIDs []string `json:"scope_item_ids"`
}

// CheckGaps counts accountable packages, not rows. Inherited roles come from
// the nearest ancestor that supplies that role; a child replaces only its own
// roles. This does not certify qualifications or contractual appointments.
func CheckGaps(items []works.Item, packages []Package, scope []Assignment, cat *knowledge.Catalog) ([]Gap, error) {
	if cat == nil {
		return nil, fmt.Errorf("action catalogue unavailable")
	}
	actions := map[string]bool{}
	for _, a := range cat.Actions() {
		actions[a.ID] = a.NeedsDesign
	}
	byID := map[string]works.Item{}
	for _, item := range items {
		if _, ok := byID[item.ID]; ok {
			return nil, fmt.Errorf("duplicate work item")
		}
		byID[item.ID] = item
	}
	pkgs := map[string]Package{}
	for _, p := range packages {
		if p.RetiredAt == nil {
			pkgs[p.ID] = p
		}
	}
	roles := map[string]map[string][]Assignment{}
	for _, a := range scope {
		if a.RetiredAt != nil || a.Inclusion != "included" || a.WorkItemID == "" || a.Role == "" {
			continue
		}
		if _, ok := pkgs[a.PackageID]; !ok {
			continue
		}
		if roles[a.WorkItemID] == nil {
			roles[a.WorkItemID] = map[string][]Assignment{}
		}
		roles[a.WorkItemID][a.Role] = append(roles[a.WorkItemID][a.Role], a)
	}
	find := func(item works.Item, role string) ([]Assignment, error) {
		seen := map[string]bool{}
		for {
			if seen[item.ID] {
				return nil, fmt.Errorf("work parent cycle at %s", item.ID)
			}
			seen[item.ID] = true
			if item.RetiredAt != nil || item.Inclusion != "included" {
				return nil, nil
			}
			if found := roles[item.ID][role]; len(found) > 0 {
				return found, nil
			}
			if item.ParentID == "" {
				return nil, nil
			}
			parent, ok := byID[item.ParentID]
			if !ok || !parent.IsGroup {
				return nil, fmt.Errorf("missing group parent for %s", item.ID)
			}
			item = parent
		}
	}
	out := []Gap{}
	for _, item := range items {
		if item.RetiredAt != nil || item.Inclusion != "included" || item.IsGroup {
			continue
		}
		if item.ReviewStatus != "accepted_for_planning" && item.ReviewStatus != "verified" {
			out = append(out, Gap{WorkItemID: item.ID, State: "not_yet_accepted", PackageIDs: []string{}, ScopeItemIDs: []string{}})
			continue
		}
		needsDesign, known := actions[item.Action]
		if !known {
			return nil, fmt.Errorf("unknown action %s", item.Action)
		}
		type check struct {
			name         string
			roles, kinds []string
			min, max     int
		}
		checks := []check{}
		switch item.Action {
		case "new", "replace", "upgrade", "alter", "repair", "remove":
			checks = append(checks, check{"install", []string{"install"}, []string{"works"}, 1, 1}, check{"supply", []string{"supply"}, []string{"works", "supply", "services"}, 0, 1})
		case "investigate":
			checks = append(checks, check{"inspect_or_test", []string{"inspect", "test"}, []string{"services", "works", "supply"}, 1, 1})
		case "retain":
			checks = append(checks, check{"maintain_operation_or_protect", []string{"maintain_operation", "protect"}, []string{"works"}, 1, 0})
		}
		if needsDesign {
			checks = append(checks, check{"design", []string{"design"}, []string{"services", "works"}, 1, 1})
		}
		for _, c := range checks {
			owners, refs := map[string]bool{}, map[string]bool{}
			for _, role := range c.roles {
				assigned, err := find(item, role)
				if err != nil {
					return nil, err
				}
				for _, a := range assigned {
					if slices.Contains(c.kinds, pkgs[a.PackageID].Kind) {
						owners[a.PackageID] = true
						refs[a.ID] = true
					}
				}
			}
			state := ""
			if len(owners) < c.min {
				state = "gap"
			} else if c.max > 0 && len(owners) > c.max {
				state = "overlap"
			}
			if state != "" {
				finding := Gap{WorkItemID: item.ID, Role: c.name, State: state, PackageIDs: []string{}, ScopeItemIDs: []string{}}
				for id := range owners {
					finding.PackageIDs = append(finding.PackageIDs, id)
				}
				for id := range refs {
					finding.ScopeItemIDs = append(finding.ScopeItemIDs, id)
				}
				sort.Strings(finding.PackageIDs)
				sort.Strings(finding.ScopeItemIDs)
				out = append(out, finding)
			}
		}
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].WorkItemID != out[j].WorkItemID {
			return out[i].WorkItemID < out[j].WorkItemID
		}
		return out[i].Role < out[j].Role
	})
	return out, nil
}

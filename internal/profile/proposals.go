package profile

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"sitewise/internal/knowledge"
	"sitewise/internal/works"
)

// ProposalInputs adapts reconciled profile rows, never raw candidates. An
// explicit part unknown shadows the whole-project value; an absent part row
// inherits it. Assumptions and planning allowances cannot suppress a risk.
func ProposalInputs(rows []Row, parts []Part, items []works.Item, cat *knowledge.Catalog) (works.ProposalInput, error) {
	if cat == nil {
		return works.ProposalInput{}, fmt.Errorf("catalogue required")
	}
	out := works.ProposalInput{Items: items, Parts: map[string]works.ProposalPart{}}
	whole := wholePart(parts)
	known := map[string]bool{}
	for _, p := range parts {
		if p.ID == "" || known[p.ID] {
			return out, fmt.Errorf("invalid proposal parts")
		}
		known[p.ID] = true
	}
	byPart := map[string]map[string]Row{}
	types := map[string]bool{}
	for _, row := range rows {
		if !known[row.PartID] {
			return out, fmt.Errorf("profile row references unknown proposal part")
		}
		if byPart[row.PartID] == nil {
			byPart[row.PartID] = map[string]Row{}
		}
		if _, ok := byPart[row.PartID][row.Key]; ok {
			return out, fmt.Errorf("duplicate profile proposal input")
		}
		byPart[row.PartID][row.Key] = row
		if row.Key == "hdr.work_type" && appliedWorkRow(row) {
			types[row.Value] = true
		}
	}
	workTypes := []string{}
	for value := range types {
		workTypes = append(workTypes, value)
	}
	sort.Strings(workTypes)
	// Existing/present evidence is indexed per effective part once, not rebuilt
	// for each of the hundreds of catalogue predicates.
	existing := map[string][]knowledge.WorkItem{}
	present := map[string][]knowledge.WorkItem{}
	completed := map[string][]knowledge.WorkItem{}
	for _, part := range parts {
		effective := map[string]Row{}
		for key, row := range byPart[whole] {
			effective[key] = row
		}
		for key, row := range byPart[part.ID] {
			effective[key] = row
		}
		p := works.ProposalPart{Values: map[string]works.ProposalValue{}, WorkTypes: append([]string{}, workTypes...), Signals: map[string]works.ProposalSignal{}}
		keys := []string{}
		for key := range effective {
			keys = append(keys, key)
		}
		sort.Strings(keys)
		for _, key := range keys {
			row := effective[key]
			if strings.HasPrefix(key, "sig.") {
				state := "unknown"
				if appliedWorkRow(row) && (row.Value == "true" || row.Value == "false") {
					state = row.Value
				}
				sources, _ := json.Marshal(row.Sources)
				p.Signals[key] = works.ProposalSignal{ID: key, State: state, Sources: sources}
			}
			if determinant, ok := strings.CutPrefix(key, "det."); ok {
				value := works.ProposalValue{Key: determinant, Origin: row.Origin}
				if appliedWorkRow(row) {
					value.Value = row.Value
				}
				p.Values[determinant] = value
			}
			if system, ok := strings.CutPrefix(key, "sys."); ok {
				action := ""
				if appliedWorkRow(row) {
					switch row.Value {
					case "present", "included":
						action = "retain"
					case "absent", "not_included":
						action = "remove"
					}
				}
				if id, ok := strings.CutSuffix(system, ".existing"); ok {
					existing[part.ID] = append(existing[part.ID], knowledge.WorkItem{System: id, Action: action})
				}
				if id, ok := strings.CutSuffix(system, ".presence"); ok {
					present[part.ID] = append(present[part.ID], knowledge.WorkItem{System: id, Action: action})
				}
			}
		}
		partID := part.ID
		p.PresentState = func(system string) knowledge.Truth {
			return cat.SystemExisting(system, knowledge.WorksEnv{Items: completed[partID], Existing: func(target string) knowledge.Truth {
				return cat.SystemExisting(target, knowledge.WorksEnv{Items: present[partID]})
			}})
		}
		out.Parts[partID] = p
	}
	for _, item := range items {
		if !known[item.PartID] {
			return out, fmt.Errorf("work item references unknown proposal part")
		}
		if item.IsGroup || item.RetiredAt != nil || item.Inclusion != "included" {
			continue
		}
		action := item.Action
		if action == "new" || action == "replace" {
			action = "retain"
		} // those works leave a system in the completed building
		completed[item.PartID] = append(completed[item.PartID], knowledge.WorkItem{System: item.SystemID, Action: action, LayoutChange: item.LayoutChange})
	}
	out.Existing = func(part, system string) knowledge.Truth {
		return cat.SystemExisting(system, knowledge.WorksEnv{Items: existing[part]})
	}
	return out, nil
}

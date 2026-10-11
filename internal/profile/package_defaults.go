package profile

import (
	"sort"
	"strings"

	"sitewise/internal/knowledge"
)

// PackageComplexityFacts consumes whole-site rows selected by the caller.
// Conflicting applied values never pick a consultant by iteration order.
func PackageComplexityFacts(rows []Row) map[string]string {
	values, conflicts := map[string]string{}, map[string]bool{}
	for _, row := range rows {
		if !appliedWorkRow(row) {
			continue
		}
		field, ok := strings.CutPrefix(row.Key, "det.")
		if !ok {
			field, ok = strings.CutPrefix(row.Key, "hdr.cond.")
		}
		if !ok {
			continue
		}
		if previous, exists := values[field]; exists && previous != row.Value {
			conflicts[field] = true
		}
		values[field] = row.Value
	}
	for field := range conflicts {
		delete(values, field)
	}
	return values
}

// PackageBaselineFacts accepts reconciled applied values, never guesses,
// assumptions or explicit unknowns. Every class adds its category: the
// header category (each reading of it), and each NCC class in det.ncc_class
// mapped through the taxonomy's single-class subclasses. Scope is every
// scope.<system> row marked in, whatever its band, as the profile shows it.
func PackageBaselineFacts(cat *knowledge.Catalog, rows []Row) (classes, workTypes, scope []string) {
	byNCC := map[string][]string{}
	if cat != nil {
		for _, c := range cat.Taxonomy().BuildingClasses {
			for _, sub := range c.Subclasses {
				byNCC[sub.NCCClass] = append(byNCC[sub.NCCClass], c.ID)
			}
		}
	}
	classSet, types, scopeSet := map[string]bool{}, map[string]bool{}, map[string]bool{}
	for _, row := range rows {
		if system, ok := strings.CutPrefix(row.Key, scopePrefix); ok {
			if row.Value == scopeIn {
				scopeSet[system] = true
			}
			continue
		}
		if !appliedWorkRow(row) {
			continue
		}
		switch row.Key {
		case "hdr.building_class":
			classSet[row.Value] = true
		case "det.ncc_class":
			for _, ncc := range strings.Split(row.Value, ",") {
				for _, id := range byNCC[strings.TrimSpace(ncc)] {
					classSet[id] = true
				}
			}
		case "hdr.work_type":
			types[row.Value] = true
		}
	}
	return sortedKeys(classSet), sortedKeys(types), sortedKeys(scopeSet)
}

func sortedKeys(set map[string]bool) []string {
	out := make([]string, 0, len(set))
	for k := range set {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

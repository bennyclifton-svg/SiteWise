package profile

import (
	"sort"
	"strings"
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
// assumptions or explicit unknowns. The caller supplies only the whole-site
// building class and the project's/parts' work types.
func PackageBaselineFacts(rows []Row) (string, []string) {
	class := ""
	conflict := false
	types := map[string]bool{}
	for _, row := range rows {
		if !appliedWorkRow(row) {
			continue
		}
		switch row.Key {
		case "hdr.building_class":
			if class != "" && class != row.Value {
				conflict = true
			}
			class = row.Value
		case "hdr.work_type":
			types[row.Value] = true
		}
	}
	if conflict {
		class = ""
	}
	workTypes := []string{}
	for value := range types {
		workTypes = append(workTypes, value)
	}
	sort.Strings(workTypes)
	return class, workTypes
}

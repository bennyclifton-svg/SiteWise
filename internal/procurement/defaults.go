package procurement

import (
	"sitewise/internal/knowledge"
	"slices"
	"sort"
	"strings"
)

type PackageSuggestion struct {
	Key              string            `json:"key"`
	Kind             string            `json:"kind"`
	Title            string            `json:"title"`
	LifecycleStatus  string            `json:"lifecycle_status"`
	ReviewStatus     string            `json:"review_status"`
	Origin           string            `json:"origin"`
	Draft            bool              `json:"draft"`
	KnowledgeVersion string            `json:"knowledge_version"`
	BuildingClass    string            `json:"building_class"`
	WorkTypes        []string          `json:"work_types"`
	MatchedFields    map[string]string `json:"matched_fields,omitempty"`
}

// SuggestedPackages adds only explicitly mapped, valid catalogue conditions.
// These remain transient proposals; reading them never creates an appointment.
func SuggestedPackages(cat *knowledge.Catalog, buildingClass string, workTypes []string, facts map[string]string, existing []Package) []PackageSuggestion {
	out := BaselinePackages(cat, buildingClass, workTypes, existing)
	if cat == nil {
		return out
	}
	taken := map[string]bool{}
	for _, p := range existing {
		if p.Kind == "services" && p.RetiredAt == nil {
			taken[strings.ToLower(strings.TrimSpace(p.Title))] = true
		}
	}
	indices := map[string]int{}
	for i, suggestion := range out {
		indices[suggestion.Title] = i
	}
	defaults := cat.PackageDefaults()
	for _, rule := range defaults.ComplexityAdditions {
		value, known := facts[rule.Field]
		if !known || !slices.Contains(rule.Values, value) || !packageFieldValue(cat, rule.Field, value) {
			continue
		}
		for _, name := range rule.Consultants {
			if taken[strings.ToLower(strings.TrimSpace(name))] {
				continue
			}
			i, found := indices[name]
			if !found {
				i = len(out)
				indices[name] = i
				out = append(out, PackageSuggestion{Key: "default:services:" + name, Kind: "services", Title: name,
					LifecycleStatus: "proposed", ReviewStatus: "proposed", Origin: "calculation",
					Draft: defaults.Status != "reviewed", KnowledgeVersion: cat.Version(), WorkTypes: []string{}})
			}
			if out[i].MatchedFields == nil {
				out[i].MatchedFields = map[string]string{}
			}
			out[i].MatchedFields[rule.Field] = value
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Title < out[j].Title })
	return out
}

func packageFieldValue(cat *knowledge.Catalog, field, value string) bool {
	if determinant, ok := cat.Determinant(field); ok {
		if determinant.Deprecated() {
			return false
		}
		if determinant.Value == "boolean" {
			return value == "true" || value == "false"
		}
		for _, option := range determinant.Options {
			if option.ID == value {
				return true
			}
		}
		return false
	}
	for _, condition := range cat.Taxonomy().Conditions {
		if condition.Key != field {
			continue
		}
		for _, option := range condition.Options {
			if option.ID == value {
				return true
			}
		}
	}
	return false
}

// BaselinePackages only consumes the supported taxonomy fields. Legacy
// complexity fields are intentionally not mapped by label or guessed value.
func BaselinePackages(cat *knowledge.Catalog, buildingClass string, workTypes []string, existing []Package) []PackageSuggestion {
	out := []PackageSuggestion{}
	if cat == nil || buildingClass == "" || len(workTypes) == 0 {
		return out
	}
	taken := map[string]bool{}
	for _, p := range existing {
		if p.Kind == "services" && p.RetiredAt == nil {
			taken[strings.ToLower(strings.TrimSpace(p.Title))] = true
		}
	}
	byName := map[string]map[string]bool{}
	defaults := cat.PackageDefaults()
	for _, b := range defaults.Baselines {
		if !slices.Contains(b.BuildingClasses, buildingClass) {
			continue
		}
		for _, w := range workTypes {
			if !slices.Contains(b.WorkTypes, w) {
				continue
			}
			for _, name := range b.Consultants {
				if taken[strings.ToLower(strings.TrimSpace(name))] {
					continue
				}
				if byName[name] == nil {
					byName[name] = map[string]bool{}
				}
				byName[name][w] = true
			}
		}
	}
	for name, types := range byName {
		matched := []string{}
		for w := range types {
			matched = append(matched, w)
		}
		sort.Strings(matched)
		out = append(out, PackageSuggestion{Key: "default:services:" + name, Kind: "services", Title: name, LifecycleStatus: "proposed", ReviewStatus: "proposed", Origin: "calculation", Draft: defaults.Status != "reviewed", KnowledgeVersion: cat.Version(), BuildingClass: buildingClass, WorkTypes: matched})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Title < out[j].Title })
	return out
}

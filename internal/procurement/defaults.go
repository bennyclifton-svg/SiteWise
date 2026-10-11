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
	Novation         bool              `json:"novation"`
}

// SuggestionInputs are the applied profile facts packages are proposed from.
// Existing holds live and retired packages: a consultant the PM removed is
// not proposed again, and neither is a dismissed proposal key.
type SuggestionInputs struct {
	Classes, WorkTypes, Scope []string
	Facts                     map[string]string
	Existing                  []Package
	Dismissed                 map[string]bool
}

// SuggestedPackages unions the work-type baselines, each known class's
// top-up, the scope additions and the explicitly mapped complexity
// conditions. These remain transient proposals; reading them never creates
// an appointment.
func SuggestedPackages(cat *knowledge.Catalog, in SuggestionInputs) []PackageSuggestion {
	out := []PackageSuggestion{}
	if cat == nil {
		return out
	}
	taken := map[string]bool{}
	for _, p := range in.Existing {
		if p.Kind == "services" {
			taken[strings.ToLower(strings.TrimSpace(p.Title))] = true
		}
	}
	defaults := cat.PackageDefaults()
	indices := map[string]int{}
	// add returns the suggestion for name, or nil when the PM already has,
	// removed or dismissed it.
	add := func(name string) *PackageSuggestion {
		key := "default:services:" + name
		if taken[strings.ToLower(strings.TrimSpace(name))] || in.Dismissed[key] {
			return nil
		}
		i, found := indices[name]
		if !found {
			i = len(out)
			indices[name] = i
			out = append(out, PackageSuggestion{Key: key, Kind: "services", Title: name,
				LifecycleStatus: "proposed", ReviewStatus: "proposed", Origin: "calculation",
				Draft: defaults.Status != "reviewed", KnowledgeVersion: cat.Version(), WorkTypes: []string{}})
		}
		return &out[i]
	}
	match := func(s *PackageSuggestion, field, value string) {
		if s.MatchedFields == nil {
			s.MatchedFields = map[string]string{}
		}
		if _, set := s.MatchedFields[field]; !set {
			s.MatchedFields[field] = value
		}
	}
	for _, b := range defaults.Baselines {
		classes := baselineClasses(b, in.Classes)
		if classes == nil {
			continue
		}
		for _, w := range in.WorkTypes {
			if !slices.Contains(b.WorkTypes, w) {
				continue
			}
			for _, name := range b.Consultants {
				if s := add(name); s != nil {
					if !slices.Contains(s.WorkTypes, w) {
						s.WorkTypes = append(s.WorkTypes, w)
						sort.Strings(s.WorkTypes)
					}
					for _, c := range classes {
						if !slices.Contains(strings.Split(s.BuildingClass, ","), c) {
							s.BuildingClass = strings.TrimPrefix(s.BuildingClass+","+c, ",")
						}
					}
				}
			}
		}
	}
	for _, rule := range defaults.ScopeAdditions {
		system, ok := scopeMatch(rule.Systems, in.Scope)
		if !ok {
			continue
		}
		for _, name := range rule.Consultants {
			if s := add(name); s != nil {
				match(s, "scope", system)
			}
		}
	}
	for _, rule := range defaults.ComplexityAdditions {
		value, known := in.Facts[rule.Field]
		if !known || !slices.Contains(rule.Values, value) || !packageFieldValue(cat, rule.Field, value) {
			continue
		}
		for _, name := range rule.Consultants {
			if s := add(name); s != nil {
				match(s, rule.Field, value)
			}
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Title < out[j].Title })
	return out
}

// baselineClasses reports which known classes a baseline applies through:
// nil when it does not apply, empty when it applies to any class.
func baselineClasses(b knowledge.PackageBaseline, known []string) []string {
	for _, c := range known {
		if slices.Contains(b.ExceptBuildingClasses, c) {
			return nil
		}
	}
	if len(b.BuildingClasses) == 0 {
		return []string{}
	}
	var hit []string
	for _, c := range known {
		if slices.Contains(b.BuildingClasses, c) {
			hit = append(hit, c)
		}
	}
	return hit
}

// scopeMatch returns the first in-scope system a rule names, itself or one
// below it, so the row can say why it was suggested.
func scopeMatch(systems, scope []string) (string, bool) {
	for _, in := range scope {
		for _, s := range systems {
			if in == s || strings.HasPrefix(in, s+".") {
				return in, true
			}
		}
	}
	return "", false
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

package procurement

import "sort"

// Combined engagements, slice 5 (docs/plans/2026-11-07-combined-engagements-slice5.md):
// rows made before groups existed get a prompt to combine. The app never
// combines on its own; the PM confirms with the ordinary Combine.

// ServicesFiling is the combined "Services Engineer" discipline, and
// ServicesDisciplines the separate services it covers.
const ServicesFiling = "consultant.services"

var ServicesDisciplines = map[string]bool{
	"consultant.mechanical": true, "consultant.electrical": true, "consultant.hydraulic": true, "consultant.fire_services": true,
}

// ServicesDisciplineIDs lists ServicesDisciplines in a fixed order, for SQL.
func ServicesDisciplineIDs() []string {
	out := make([]string, 0, len(ServicesDisciplines))
	for d := range ServicesDisciplines {
		out = append(out, d)
	}
	sort.Strings(out)
	return out
}

// Prompt keys: one of each per project, dismissed for good.
const (
	PromptServices  = "combine:services"
	PromptEngagedBy = "combine:engaged_by"
)

// CombinePrompt offers to combine existing rows. PackageIDs are the members
// ticked for the PM, in the order of the group's default name.
type CombinePrompt struct {
	Key        string   `json:"key"`
	Reason     string   `json:"reason"`
	Text       string   `json:"text"`
	PackageIDs []string `json:"package_ids"`
}

// IsCombinePrompt reports whether key names a combine prompt.
func IsCombinePrompt(key string) bool { return key == PromptServices || key == PromptEngagedBy }

// CombinePrompts lists the prompts for a project's rows. lead reports a
// discipline whose RFP names its subconsultants; awarded holds rows with a
// live award, which Combine would refuse; dismissed holds dismissed keys.
func CombinePrompts(items []Package, lead func(disciplineID string) bool, awarded, dismissed map[string]bool) []CombinePrompt {
	open := func(p Package) bool {
		return p.Kind == "services" && !p.IsGroup() && p.GroupID == "" && p.RetiredAt == nil && p.LifecycleStatus != "cancelled" && !awarded[p.ID]
	}
	var combined, services, leads, subs []Package
	for _, p := range items {
		if !open(p) {
			continue
		}
		switch {
		case p.DisciplineID == ServicesFiling:
			combined = append(combined, p)
		case ServicesDisciplines[p.DisciplineID]:
			services = append(services, p)
		}
		if p.EngagedBy == EngagedByLead {
			subs = append(subs, p)
		} else if lead != nil && p.DisciplineID != "" && lead(p.DisciplineID) {
			leads = append(leads, p)
		}
	}
	out := []CombinePrompt{}
	if !dismissed[PromptServices] && len(combined) > 0 && len(services) > 0 {
		members := append(append([]Package{}, combined...), services...)
		out = append(out, CombinePrompt{Key: PromptServices, Reason: "services", PackageIDs: ids(members),
			Text: joinTitles(combined) + " covers services that also have their own rows. Combine them into one engagement so each fee is counted once."})
	}
	if !dismissed[PromptEngagedBy] && len(leads) == 1 && len(subs) > 0 {
		members := append(append([]Package{}, leads...), subs...)
		out = append(out, CombinePrompt{Key: PromptEngagedBy, Reason: "engaged_by", PackageIDs: ids(members),
			Text: joinTitles(subs) + " " + verb(len(subs)) + " marked as engaged by the lead consultant. That setting is retired: combine with " + leads[0].Title + " into one engagement so each fee is counted once."})
	}
	return out
}

func ids(ps []Package) []string {
	out := make([]string, len(ps))
	for i, p := range ps {
		out[i] = p.ID
	}
	return out
}

func joinTitles(ps []Package) string {
	names := make([]string, len(ps))
	for i, p := range ps {
		names[i] = p.Title
	}
	switch len(names) {
	case 0:
		return ""
	case 1:
		return names[0]
	}
	s := ""
	for i, n := range names[:len(names)-1] {
		if i > 0 {
			s += ", "
		}
		s += n
	}
	return s + " and " + names[len(names)-1]
}

func verb(n int) string {
	if n == 1 {
		return "is"
	}
	return "are"
}

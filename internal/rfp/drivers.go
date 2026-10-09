// Package rfp turns a consultant RFP discipline file, the project's saved
// facts and the user's settings into a resolved RFP, then into report
// sections. It is lookup and computation only: no Jev call, no prose
// generation (AGENTS.md rule 1). Unknown facts stay unknown: they become a
// decision or a printed pricing assumption, never a silent exclusion.
package rfp

import (
	"encoding/json"
	"fmt"
	"sort"
	"strconv"
	"strings"

	"sitewise/internal/knowledge"
	"sitewise/internal/procurement"
	"sitewise/internal/reports"
)

const (
	Known    = "known"
	Unknown  = "unknown"
	Conflict = "conflict"
)

// Fact is one driver value with its state and source. Values are empty
// unless State is known; a conflict keeps every source value it saw.
type Fact struct {
	Values []string `json:"values"`
	State  string   `json:"state"`
	Source string   `json:"source"` // profile | package | derived | user | none
	Detail string   `json:"detail,omitempty"`
}

type Part struct {
	ID      string   `json:"id"`
	Label   string   `json:"label"`
	Kind    string   `json:"kind"`
	Classes []string `json:"classes"`
	GFA     string   `json:"gfa,omitempty"`
}

type Consultant struct {
	PackageID    string `json:"package_id"`
	Title        string `json:"title"`
	DisciplineID string `json:"discipline_id,omitempty"`
	Status       string `json:"status"` // appointed | to be appointed
}

// DriverSet is read once at refresh and frozen in the draft, so a setting
// change re-resolves without reloading the project.
type DriverSet struct {
	Facts       map[string]Fact   `json:"facts"`
	Parts       []Part            `json:"parts"`
	Consultants []Consultant      `json:"consultants"`
	Text        map[string]string `json:"text"`
	Documents   int               `json:"documents"`
}

func (d DriverSet) Fact(key string) Fact {
	if f, ok := d.Facts[key]; ok {
		return f
	}
	return Fact{State: Unknown, Source: "none"}
}

var known = func(source string, values ...string) Fact { return Fact{Values: values, State: Known, Source: source} }

// briefKey splits a snapshot brief ID "<part>:<key>" into its part and key.
func briefKey(id string) (string, string) {
	part, key, ok := strings.Cut(id, ":")
	if !ok {
		return "", id
	}
	return part, key
}

// Drivers reads the facts a consultant RFP switches on from the report
// snapshot. Booleans set a flag only when stated; silence stays unknown.
func Drivers(s reports.Snapshot, sk knowledge.RFPSkeleton) DriverSet {
	d := DriverSet{Facts: map[string]Fact{}, Text: map[string]string{}}
	whole := ""
	for _, p := range s.Parts {
		if p.Kind == "whole" {
			whole = p.ID
		}
	}
	byPart := map[string]map[string]reports.BriefValue{}
	for _, v := range s.Brief {
		part, key := briefKey(v.ID)
		if byPart[part] == nil {
			byPart[part] = map[string]reports.BriefValue{}
		}
		byPart[part][key] = v
	}
	value := func(key string) (string, bool) {
		if v, ok := byPart[whole][key]; ok && !v.Unknown && strings.TrimSpace(v.Text) != "" {
			return strings.TrimSpace(v.Text), true
		}
		for _, p := range s.Parts {
			if v, ok := byPart[p.ID][key]; ok && !v.Unknown && strings.TrimSpace(v.Text) != "" {
				return strings.TrimSpace(v.Text), true
			}
		}
		return "", false
	}
	choice := func(fact, key string) {
		if v, ok := value(key); ok {
			d.Facts[fact] = known("profile", v)
		}
	}
	choice("work_type", "hdr.work_type")
	choice("category", "hdr.building_class")
	choice("pathway", "hdr.cond.planning")
	choice("procurement", "hdr.cond.procurement_route")
	choice("state", "det.state")
	for fact, key := range map[string]string{"site_area": "hdr.scale.site_sqm", "storeys": "hdr.scale.storeys", "parking": "hdr.scale.car_parks"} {
		if v, ok := value(key); ok {
			if _, err := strconv.ParseFloat(v, 64); err == nil {
				d.Facts[fact] = known("profile", v)
			}
		}
	}
	for _, key := range []string{"hdr.scale.total_gfa_sqm", "hdr.scale.gfa_sqm"} {
		if v, ok := value(key); ok {
			if _, err := strconv.ParseFloat(v, 64); err == nil {
				d.Facts["gfa"] = known("profile", v)
				break
			}
		}
	}
	if s.ConstructionBudget != "" {
		d.Facts["budget"] = known("cost_plan", s.ConstructionBudget)
		d.Facts["budget"] = Fact{Values: []string{s.ConstructionBudget}, State: Known, Source: "cost_plan", Detail: s.BudgetBasis}
	}
	if s.Package.ID != "" {
		novation := "no"
		if s.Package.Novation {
			novation = "yes"
		}
		d.Facts["novation"] = known("package", novation)
	}
	d.Parts = parts(s, byPart, whole)
	d.Consultants = consultants(s)
	d.Documents = len(s.Documents)
	readFlags(d.Facts, value)
	deriveFlags(d.Facts, d.Parts)
	for _, f := range sk.Flags {
		if _, ok := d.Facts["flag."+f.Key]; !ok {
			d.Facts["flag."+f.Key] = Fact{State: Unknown, Source: "none"}
		}
	}
	var site struct {
		Project struct {
			Name string `json:"name"`
		} `json:"project"`
		Site struct {
			Address string `json:"address"`
			Lot     string `json:"lot"`
		} `json:"site"`
	}
	_ = json.Unmarshal(s.ProjectBasis, &site)
	d.Text["project"] = strings.TrimSpace(s.ProjectName)
	d.Text["address"] = strings.TrimSpace(site.Site.Address)
	d.Text["lot"] = strings.TrimSpace(site.Site.Lot)
	return d
}

// readFlags maps saved determinants to flag facts. Two sources that disagree
// become a conflict that the user resolves.
func readFlags(facts map[string]Fact, value func(string) (string, bool)) {
	boolean := func(v string) (string, bool) {
		switch strings.ToLower(v) {
		case "stated_true", "true", "yes":
			return "yes", true
		case "stated_false", "false", "no":
			return "no", true
		}
		return "", false
	}
	set := func(flag, v, detail string) {
		key := "flag." + flag
		next := Fact{Values: []string{v}, State: Known, Source: "profile", Detail: detail}
		old, ok := facts[key]
		if ok && old.State == Known && old.Values[0] != v {
			facts[key] = Fact{Values: []string{old.Values[0], v}, State: Conflict, Source: "profile", Detail: old.Detail + "; " + detail}
			return
		}
		if !ok {
			facts[key] = next
		}
	}
	for flag, key := range map[string]string{
		"bushfire": "det.bushfire_prone_land", "flood": "det.flood_hazard",
		"contamination": "det.potentially_contaminated_land", "acid_sulfate": "det.acid_sulfate_soils_mapped",
		"groundwater": "det.groundwater_above_excavation", "existing_building": "det.existing_building",
	} {
		if raw, ok := value(key); ok {
			if v, ok := boolean(raw); ok {
				set(flag, v, key)
			}
		}
	}
	if raw, ok := value("det.heritage_status"); ok {
		v := "yes"
		if raw == "none" {
			v = "no"
		}
		set("heritage", v, "det.heritage_status "+raw)
	}
	if raw, ok := value("det.heritage"); ok {
		if v, ok := boolean(raw); ok {
			set("heritage", v, "det.heritage")
		}
	}
	if raw, ok := value("hdr.cond.operational_constraints"); ok {
		switch raw {
		case "vacant":
			set("live_operations", "no", "operational constraints "+raw)
		case "partial_occupation", "live_environment", "24_7_occupied":
			set("live_operations", "yes", "operational constraints "+raw)
		}
	}
}

// deriveFlags fills facts that follow from others: an existing building from
// the work type, and mixed use from parts with different classes.
func deriveFlags(facts map[string]Fact, parts []Part) {
	if f, ok := facts["flag.existing_building"]; !ok || f.State == Unknown {
		if wt := facts["work_type"]; wt.State == Known {
			switch wt.Values[0] {
			case "refurb", "extend", "remediation":
				facts["flag.existing_building"] = Fact{Values: []string{"yes"}, State: Known, Source: "derived", Detail: "work type " + wt.Values[0]}
			case "new":
				facts["flag.existing_building"] = Fact{Values: []string{"no"}, State: Known, Source: "derived", Detail: "work type new"}
			}
		}
	}
	classes := map[string]bool{}
	missing := false
	for _, p := range parts {
		if len(p.Classes) == 0 {
			missing = true
		}
		for _, c := range p.Classes {
			classes[c] = true
		}
	}
	switch {
	case len(classes) > 1:
		facts["flag.mixed_use"] = Fact{Values: []string{"yes"}, State: Known, Source: "derived", Detail: "parts with different classes"}
	case len(classes) == 1 && !missing:
		facts["flag.mixed_use"] = Fact{Values: []string{"no"}, State: Known, Source: "derived", Detail: "one class"}
	}
}

// parts keeps each building part with its own classes. The whole-project
// part stands in only when the profile records no separate parts.
func parts(s reports.Snapshot, byPart map[string]map[string]reports.BriefValue, whole string) []Part {
	out := []Part{}
	for _, p := range s.Parts {
		if p.Kind == "whole" {
			continue
		}
		out = append(out, partFacts(p, byPart[p.ID]))
	}
	if len(out) == 0 {
		for _, p := range s.Parts {
			if p.ID == whole {
				out = append(out, partFacts(p, byPart[p.ID]))
			}
		}
	}
	return out
}

func partFacts(p reports.PartInfo, rows map[string]reports.BriefValue) Part {
	out := Part{ID: p.ID, Label: p.Label, Kind: p.Kind}
	raw := p.NCCClass
	if v, ok := rows["det.ncc_class"]; ok && !v.Unknown && v.Text != "" {
		raw = v.Text
	}
	for _, c := range strings.FieldsFunc(raw, func(r rune) bool { return r == ',' || r == ';' || r == ' ' || r == '/' }) {
		c = strings.ToLower(strings.TrimPrefix(strings.TrimPrefix(strings.TrimSpace(c), "Class"), "class"))
		if c != "" && !contains(out.Classes, c) {
			out.Classes = append(out.Classes, c)
		}
	}
	sort.Strings(out.Classes)
	for _, key := range []string{"hdr.scale.gfa_sqm", "hdr.scale.total_gfa_sqm"} {
		if v, ok := rows[key]; ok && !v.Unknown && v.Text != "" {
			out.GFA = v.Text
			break
		}
	}
	return out
}

// consultants lists the project's other live services packages with an
// honest status: a proposed or planned package is not an appointment.
func consultants(s reports.Snapshot) []Consultant {
	out := []Consultant{}
	for _, p := range s.Packages {
		if p.Kind != "services" || p.ID == s.Package.ID || p.RetiredAt != nil {
			continue
		}
		status := consultantStatus(p)
		if status == "" {
			continue
		}
		out = append(out, Consultant{PackageID: p.ID, Title: p.Title, DisciplineID: p.DisciplineID, Status: status})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Title < out[j].Title })
	return out
}

func consultantStatus(p procurement.Package) string {
	switch p.LifecycleStatus {
	case "appointed", "closed":
		return "appointed"
	case "proposed", "planned", "procuring":
		return "to be appointed"
	}
	return ""
}

func contains(list []string, v string) bool {
	for _, x := range list {
		if x == v {
			return true
		}
	}
	return false
}

func number(f Fact) (float64, bool) {
	if f.State != Known || len(f.Values) == 0 {
		return 0, false
	}
	n, err := strconv.ParseFloat(strings.ReplaceAll(f.Values[0], ",", ""), 64)
	return n, err == nil
}

func formatNumber(v string) string {
	n, err := strconv.ParseFloat(strings.ReplaceAll(v, ",", ""), 64)
	if err != nil {
		return v
	}
	if n == float64(int64(n)) {
		s := strconv.FormatInt(int64(n), 10)
		var b strings.Builder
		for i, r := range s {
			if i > 0 && (len(s)-i)%3 == 0 {
				b.WriteByte(',')
			}
			b.WriteRune(r)
		}
		return b.String()
	}
	return fmt.Sprintf("%.1f", n)
}

// Package rfp turns a consultant RFP discipline file, the project's saved
// facts and the user's settings into a resolved RFP, then into report
// sections. It is lookup and computation only: no Jev call, no prose
// generation (AGENTS.md rule 1). Unknown facts stay unknown: they become a
// decision or a printed pricing assumption, never a silent exclusion.
package rfp

import (
	"encoding/json"
	"fmt"
	"slices"
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
	// EngagedBy is client, lead_consultant or "" (not decided).
	EngagedBy string `json:"engaged_by,omitempty" yaml:"engaged_by"`
}

// Who engages a consultant (owner decision 3, 13 October 2026).
const (
	EngagedByClient = procurement.EngagedByClient
	EngagedByLead   = procurement.EngagedByLead
)

// qualityLabels mirror the quality_level condition in
// knowledge/profile/taxonomy.yaml (TestQualityLabelsMatchTaxonomy).
var qualityLabels = map[string]string{
	"project_home": "Project or volume home (builder's standard range)",
	"mid":          "Mid-range custom home",
	"high":         "High-end custom home",
	"prestige":     "Prestige home",
}

// DriverSet is read once at refresh and frozen in the draft, so a setting
// change re-resolves without reloading the project.
type DriverSet struct {
	Facts            map[string]Fact      `json:"facts"`
	Parts            []Part               `json:"parts"`
	Consultants      []Consultant         `json:"consultants"`
	Text             map[string]string    `json:"text"`
	DocumentSchedule []reports.BriefValue `json:"document_schedule"`
	Documents        int                  `json:"documents"`
	// Separate maps each trade discipline another live package holds to
	// that package's title, so a head contract leaves it out.
	Separate map[string]string `json:"separate,omitempty"`
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
	choice("regulated_design", knowledge.DetailPrefix+"regulated_design")
	// A house counts garage spaces where other buildings count car parks.
	for fact, keys := range map[string][]string{"site_area": {"hdr.scale.site_sqm"}, "storeys": {"hdr.scale.storeys"}, "parking": {"hdr.scale.car_parks", "hdr.scale.garage_spaces"},
		"bedrooms": {"hdr.scale.bedrooms"}, "bathrooms": {"hdr.scale.bathrooms"}} {
		for _, key := range keys {
			if v, ok := value(key); ok {
				if _, err := strconv.ParseFloat(v, 64); err == nil {
					d.Facts[fact] = known("profile", v)
					break
				}
			}
		}
	}
	// The unit mix prints known counts only (docs/plans/2026-11-08-unit-mix.md).
	mix := map[string]string{}
	for _, k := range knowledge.UnitMixKeys {
		if v, ok := value("hdr.scale." + k); ok {
			mix[k] = v
		}
	}
	if t := knowledge.UnitMixText(mix); t != "" {
		d.Text["unit_mix"] = t
	}
	choice("quality", "hdr.cond.quality_level")
	for _, key := range []string{"hdr.scale.total_gfa_sqm", "hdr.scale.gfa_sqm"} {
		if v, ok := value(key); ok {
			if _, err := strconv.ParseFloat(v, 64); err == nil {
				d.Facts["gfa"] = Fact{Values: []string{v}, State: Known, Source: "profile", Detail: key}
				break
			}
		}
	}
	if s.ConstructionBudget != "" {
		d.Facts["budget"] = Fact{Values: []string{s.ConstructionBudget}, State: Known, Source: "cost_plan", Detail: s.BudgetBasis}
	} else if v, ok := value(knowledge.DetailPrefix + "budget_stated"); ok {
		// A budget a document states stands in until the cost plan holds one.
		if n, ok := parseMoney(v); ok {
			d.Facts["budget"] = Fact{Values: []string{n}, State: Known, Source: "profile", Detail: statedBasis(v)}
		}
	}
	if s.Package.ID != "" {
		d.Facts["novation"] = known("package", s.Package.NovationState())
	}
	d.Parts = parts(s, byPart, whole)
	d.Consultants = consultants(s)
	d.DocumentSchedule = s.Documents
	d.Documents = len(s.Documents)
	d.Text["register"] = registerSummary(s.Documents)
	d.Text["package"] = strings.TrimSpace(s.Package.Title)
	d.Separate = separatePackages(s)
	if f, ok := systemsInScope(s); ok {
		d.Facts["system"] = f
	}
	readFlags(d.Facts, value)
	// A flag stored as a project detail is the user's word; it replaces
	// any reading of the same flag rather than disagreeing with it.
	for _, f := range DetailFlags(sk) {
		if v, ok := value(FlagHome(f.Key).Key); ok && (v == "yes" || v == "no") {
			d.Facts["flag."+f.Key] = Fact{Values: []string{v}, State: Known, Source: "profile", Detail: FlagHome(f.Key).Key}
		}
	}
	derivePackageRole(d.Facts, s.Package.DisciplineID)
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
	// What the user typed on the site record wins; otherwise the profile's
	// reading of the documents stands in for it.
	for text, key := range map[string]string{"address": knowledge.SitePrefix + "address", "lot": knowledge.SitePrefix + "lot"} {
		if d.Text[text] == "" {
			if v, ok := value(key); ok {
				d.Text[text] = v
			}
		}
	}
	for _, f := range sk.Fields {
		switch {
		case f.Profile == knowledge.SitePrefix+"address":
			d.Text[f.Key] = d.Text["address"]
		case f.Profile == knowledge.SitePrefix+"lot":
			d.Text[f.Key] = d.Text["lot"]
		case f.Profile != "":
			if v, ok := value(f.Profile); ok {
				d.Text[f.Key] = v
			}
		}
	}
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
	uses := 0
	missing := false
	for _, p := range parts {
		if len(p.Classes) == 0 {
			missing = true
		}
		for _, c := range p.Classes {
			if !classes[c] && !isClassTen(c) {
				uses++
			}
			classes[c] = true
		}
	}
	// A Class 10 outbuilding (garage, carport, pool, fence) serves the
	// building beside it, so it is not a second use: a house with a garage
	// is a house (modules.yaml; synthetic finding F11).
	switch {
	case uses > 1:
		facts["flag.mixed_use"] = Fact{Values: []string{"yes"}, State: Known, Source: "derived", Detail: "parts with different classes"}
	case len(classes) > 0 && !missing:
		facts["flag.mixed_use"] = Fact{Values: []string{"no"}, State: Known, Source: "derived", Detail: "one class, besides any Class 10"}
	}
	deriveRegulatedDesign(facts, classes, missing)
	deriveHouseCertification(facts, classes, missing)
}

func isClassTen(c string) bool { return strings.HasPrefix(c, "10") }

// packageRoles name the role a BCA-family procurement row buys: the BCA
// consultant row is advisory, the certifier row is certification. Both rows
// share one RFP file (bca-certification.yaml).
var packageRoles = map[string]map[string]string{
	"consultant.bca":       {"bca_advisory": "yes", "bca_certification": "no"},
	"consultant.certifier": {"bca_certification": "yes", "bca_advisory": "no"},
}

// derivePackageRole fills the advisory/certification choice from the row the
// RFP is for, so each RFP opens with its own role without a Profile tick
// (owner, 8 November 2026). A saved answer always wins. It runs before the
// house default so a house's BCA consultant row stays advisory.
func derivePackageRole(facts map[string]Fact, discipline string) {
	for flag, v := range packageRoles[discipline] {
		if f, ok := facts["flag."+flag]; ok && f.State != Unknown {
			continue
		}
		facts["flag."+flag] = Fact{Values: []string{v}, State: Known, Source: "derived", Detail: "the procurement row is " + discipline}
	}
}

// houseClasses mirror the house module (knowledge/rfp/modules.yaml): Class 1,
// with any Class 10 outbuilding.
var houseClasses = []string{"1a", "1b", "10a", "10b"}

// deriveHouseCertification defaults a house's certifier package to the
// principal-certifier appointment: a house cannot be built without one, and
// a separate BCA adviser is not part of a house's default scope (owner
// direction F10, docs/plans/2026-11-04-synthetic-findings-fixes.md). A saved
// answer always wins; other classes keep both choices open.
func deriveHouseCertification(facts map[string]Fact, classes map[string]bool, missing bool) {
	if missing || !(classes["1a"] || classes["1b"]) {
		return
	}
	for c := range classes {
		if !slices.Contains(houseClasses, c) {
			return
		}
	}
	for flag, v := range map[string]string{"bca_certification": "yes", "bca_advisory": "no"} {
		if f, ok := facts["flag."+flag]; ok && f.State != Unknown {
			continue
		}
		facts["flag."+flag] = Fact{Values: []string{v}, State: Known, Source: "derived", Detail: "a Class 1 house needs a principal certifier"}
	}
}

// regulatedClasses are the classes the NSW Design and Building Practitioners
// Act 2020 reaches (Class 2, and Class 3 and 9c from 3 July 2023). A project
// whose every part has a class and none of these cannot be regulated design
// (owner direction, 2 November 2026). A saved answer always wins, and a
// project with one of these classes stays for professional advice.
var regulatedClasses = []string{"2", "3", "9c"}

func deriveRegulatedDesign(facts map[string]Fact, classes map[string]bool, missing bool) {
	if f, ok := facts["regulated_design"]; ok && f.State != Unknown {
		return
	}
	if missing || len(classes) == 0 {
		return
	}
	for _, c := range regulatedClasses {
		if classes[c] {
			return
		}
	}
	facts["regulated_design"] = Fact{Values: []string{"no"}, State: Known, Source: "derived", Detail: "no Class 2, 3 or 9c part"}
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
		out = append(out, Consultant{PackageID: p.ID, Title: p.Title, DisciplineID: p.DisciplineID, Status: status, EngagedBy: p.EngagedBy})
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

// systemsInScope is the building systems the project's Profile scope holds:
// the top-level system of every included work item. With no work items
// recorded the fact stays unknown, so a trade row prints on its stated
// assumption rather than silently dropping out.
func systemsInScope(s reports.Snapshot) (Fact, bool) {
	recorded := false
	systems := []string{}
	for _, w := range s.Works {
		if w.RetiredAt != nil {
			continue
		}
		recorded = true
		if w.Inclusion != "included" || w.SystemID == "" {
			continue
		}
		root, _, _ := strings.Cut(w.SystemID, ".")
		if !slices.Contains(systems, root) {
			systems = append(systems, root)
		}
	}
	if !recorded {
		return Fact{}, false
	}
	sort.Strings(systems)
	return Fact{Values: systems, State: Known, Source: "profile", Detail: "works"}, true
}

// separatePackages names the trades let as their own contracts: every other
// live package with a trade discipline, by the first such package's title.
func separatePackages(s reports.Snapshot) map[string]string {
	out := map[string]string{}
	for _, p := range s.Packages {
		if p.ID == s.Package.ID || p.RetiredAt != nil || !strings.HasPrefix(p.DisciplineID, "trade.") {
			continue
		}
		if _, ok := out[p.DisciplineID]; !ok {
			out[p.DisciplineID] = p.Title
		}
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

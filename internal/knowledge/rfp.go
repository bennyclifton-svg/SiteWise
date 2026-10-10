package knowledge

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"sort"
	"strings"

	"gopkg.in/yaml.v3"
)

// RFP discipline files turn a golden consultant RFP into data: the stages,
// allowances and scope positions it prices, and the project facts that switch
// each row. Conditions are a closed vocabulary so the loader can reject typos,
// and an unverified statutory threshold may only raise a decision, never
// select scope (docs/plans/2026-10-09-consultant-rfp-build.md §2.2).

// RFPDrivers is the closed condition vocabulary. Numeric keys end in _over.
var RFPDrivers = map[string]bool{
	"class": true, "category": true, "work_type": true, "pathway": true,
	"procurement": true, "novation": true, "state": true, "flag": true,
	"regulated_design": true, "scale": true, "stage": true, "stage_included": true, "element": true,
	"budget_over": true, "site_area_over": true, "gfa_over": true, "parking_over": true,
}

// rawFactDrivers are the facts a statutory threshold is about. A row tied to
// an unverified statutory threshold must not test them in its when.
var rawFactDrivers = []string{"class", "category", "state", "budget_over", "site_area_over", "gfa_over", "parking_over"}

type RFPWhen []map[string][]string

type RFPProvenance struct {
	Kind string `yaml:"kind" json:"kind"` // evidence | seed | practice | client_decision
	Ref  string `yaml:"ref" json:"ref"`
	Why  string `yaml:"why" json:"why,omitempty"`
}

// RFPCell is "I" / "O" / "X" / "-" or {value, when, else}.
type RFPCell struct {
	Value string  `yaml:"value" json:"value"`
	When  RFPWhen `yaml:"when" json:"when,omitempty"`
	Else  string  `yaml:"else" json:"else,omitempty"`
}

func (c *RFPCell) UnmarshalYAML(n *yaml.Node) error {
	if n.Kind == yaml.ScalarNode {
		c.Value = n.Value
		return nil
	}
	type plain RFPCell
	var p plain
	if err := decodeKnown(n, &p); err != nil {
		return err
	}
	*c = RFPCell(p)
	return nil
}

// RFPRow is a printed line: a project-table row, a role sentence or a term.
type RFPRow struct {
	// Replace explicitly substitutes a shared project row, term or returnable at this key.
	Replace    bool            `yaml:"replace" json:"replace,omitempty"`
	Key        string          `yaml:"key" json:"key"`
	Label      string          `yaml:"label" json:"label,omitempty"`
	Text       string          `yaml:"text" json:"text"`
	When       RFPWhen         `yaml:"when" json:"when,omitempty"`
	Module     string          `yaml:"module" json:"module,omitempty"`
	Required   bool            `yaml:"required" json:"required,omitempty"`
	Provenance []RFPProvenance `yaml:"provenance" json:"provenance,omitempty"`
}

type RFPStage struct {
	Key            string   `yaml:"key" json:"key"`
	Name           string   `yaml:"name" json:"name"`
	Deliverable    string   `yaml:"deliverable" json:"deliverable"`
	Responsibility string   `yaml:"design_responsibility" json:"design_responsibility,omitempty"`
	FeeBasis       string   `yaml:"fee_basis" json:"fee_basis"`
	DeliveryStage  string   `yaml:"delivery_stage" json:"delivery_stage,omitempty"`
	Items          []string `yaml:"items" json:"items,omitempty"`
	When           RFPWhen  `yaml:"when" json:"when,omitempty"`
	Suggest        RFPWhen  `yaml:"suggest" json:"suggest,omitempty"`
	IfUnknown      string   `yaml:"if_unknown" json:"if_unknown,omitempty"`
	Assumption     string   `yaml:"assumption" json:"assumption,omitempty"`
	// Default is include or option: an option stage is priced separately so
	// the client can commission it later. IncludeWhen promotes it.
	Default     string  `yaml:"default" json:"default,omitempty"`
	IncludeWhen RFPWhen `yaml:"include_when" json:"include_when,omitempty"`
	Follows     string  `yaml:"follows" json:"follows,omitempty"`
	// ByElement stages are priced per element in the R1 grid.
	ByElement  bool            `yaml:"by_element" json:"by_element,omitempty"`
	Threshold  string          `yaml:"threshold" json:"threshold,omitempty"`
	Module     string          `yaml:"module" json:"module,omitempty"`
	ShownWhen  string          `yaml:"shown_when" json:"shown_when"`
	Note       string          `yaml:"note" json:"note,omitempty"`
	Provenance []RFPProvenance `yaml:"provenance" json:"provenance,omitempty"`
}

// RFPElement is a priced element of a stage-by-element fee grid (civil A-D).
type RFPElement struct {
	Key        string          `yaml:"key" json:"key"`
	Code       string          `yaml:"code" json:"code"`
	Name       string          `yaml:"name" json:"name"`
	Includes   string          `yaml:"includes" json:"includes"`
	When       RFPWhen         `yaml:"when" json:"when,omitempty"`
	IfUnknown  string          `yaml:"if_unknown" json:"if_unknown,omitempty"`
	Assumption string          `yaml:"assumption" json:"assumption,omitempty"`
	ShownWhen  string          `yaml:"shown_when" json:"shown_when"`
	Provenance []RFPProvenance `yaml:"provenance" json:"provenance,omitempty"`
}

type RFPAllowance struct {
	Key   string `yaml:"key" json:"key"`
	Label string `yaml:"label" json:"label"`
	// Text holds {n}, the count. Owner client means the client states the
	// quantity before issue; consultant means the consultant proposes it.
	Text           string          `yaml:"text" json:"text"`
	Unit           string          `yaml:"unit" json:"unit,omitempty"`
	Default        string          `yaml:"default" json:"default,omitempty"`
	ComplexDefault string          `yaml:"complex_default" json:"complex_default,omitempty"`
	Owner          string          `yaml:"owner" json:"owner"`
	When           RFPWhen         `yaml:"when" json:"when,omitempty"`
	Module         string          `yaml:"module" json:"module,omitempty"`
	Provenance     []RFPProvenance `yaml:"provenance" json:"provenance,omitempty"`
}

// RFPPosition is one canonical scope item and its I/O/X position. Technical
// items are selected by facts about systems and risks; only service-level
// items may follow the simple/complex scale.
type RFPPosition struct {
	Key         string          `yaml:"key" json:"key"`
	Label       string          `yaml:"label" json:"label"`
	Kind        string          `yaml:"kind" json:"kind"` // technical | service_level
	Value       *RFPCell        `yaml:"value" json:"value,omitempty"`
	Simple      *RFPCell        `yaml:"simple" json:"simple,omitempty"`
	Complex     *RFPCell        `yaml:"complex" json:"complex,omitempty"`
	Responsible string          `yaml:"responsible" json:"responsible,omitempty"`
	When        RFPWhen         `yaml:"when" json:"when,omitempty"`
	Suggest     RFPWhen         `yaml:"suggest" json:"suggest,omitempty"`
	IfUnknown   string          `yaml:"if_unknown" json:"if_unknown,omitempty"`
	Assumption  string          `yaml:"assumption" json:"assumption,omitempty"`
	Threshold   string          `yaml:"threshold" json:"threshold,omitempty"`
	Module      string          `yaml:"module" json:"module,omitempty"`
	Provenance  []RFPProvenance `yaml:"provenance" json:"provenance,omitempty"`
}

type RFPThreshold struct {
	Key      string `yaml:"key" json:"key"`
	Kind     string `yaml:"kind" json:"kind"` // statutory | client_default | evidence_estimate
	Text     string `yaml:"text" json:"text"`
	Source   string `yaml:"source" json:"source"`
	Verified *bool  `yaml:"verified" json:"verified"`
}

type RFPBoundary struct {
	When  RFPWhen `yaml:"when" json:"when,omitempty"`
	Start string  `yaml:"start" json:"start"`
	End   string  `yaml:"end" json:"end"`
}

type RFPScale struct {
	Text    string  `yaml:"text" json:"text"`
	Complex RFPWhen `yaml:"complex" json:"complex"`
}

type RFPDiscipline struct {
	ID      string `yaml:"id" json:"id"`
	Version int    `yaml:"version" json:"version"`
	Status  string `yaml:"status" json:"status"`
	// ReviewedBy and ReviewedOn record the owner's review; reviewed needs both.
	ReviewedBy    string   `yaml:"reviewed_by" json:"reviewed_by,omitempty"`
	ReviewedOn    string   `yaml:"reviewed_on" json:"reviewed_on,omitempty"`
	Title         string   `yaml:"title" json:"title"`
	Heading       string   `yaml:"heading" json:"heading"`
	Golden        string   `yaml:"golden" json:"golden"`
	DisciplineIDs []string `yaml:"discipline_ids" json:"discipline_ids"`
	// TitleAliases suggest a discipline for a package with no discipline ID.
	// A suggestion is confirmed by the user; it never selects scope itself.
	TitleAliases        []string       `yaml:"title_aliases" json:"title_aliases"`
	EvidenceLimit       string         `yaml:"evidence_limit" json:"evidence_limit,omitempty"`
	Thresholds          []RFPThreshold `yaml:"thresholds" json:"thresholds"`
	Scale               RFPScale       `yaml:"scale" json:"scale"`
	UnsupportedPathways []string       `yaml:"unsupported_pathways" json:"unsupported_pathways,omitempty"`
	ProjectRows         []RFPRow       `yaml:"project_rows" json:"project_rows"`
	Role                []RFPRow       `yaml:"role" json:"role"`
	StageIntro          string         `yaml:"stage_intro" json:"stage_intro"`
	Boundary            []RFPBoundary  `yaml:"boundary" json:"boundary"`
	Stages              []RFPStage     `yaml:"stages" json:"stages"`
	ElementIntro        string         `yaml:"element_intro" json:"element_intro,omitempty"`
	Elements            []RFPElement   `yaml:"elements" json:"elements,omitempty"`
	Allowances          []RFPAllowance `yaml:"allowances" json:"allowances"`
	Positions           []RFPPosition  `yaml:"positions" json:"positions"`
	ScopeSchedule       []RFPRow       `yaml:"scope_schedule" json:"scope_schedule,omitempty"`
	Terms               []RFPRow       `yaml:"terms" json:"terms"`
	Evaluation          []RFPWeight    `yaml:"evaluation" json:"evaluation,omitempty"`
	EvaluationNote      string         `yaml:"evaluation_note" json:"evaluation_note,omitempty"`
	AllowancesNote      string         `yaml:"allowances_note" json:"allowances_note,omitempty"`
	// Returnables holds this discipline's line for a shared schedule (r2..r9).
	Returnables map[string]string `yaml:"returnables" json:"returnables"`
	// ReturnableNotes add a line to a schedule only when their condition holds.
	ReturnableNotes []RFPRow `yaml:"returnable_notes" json:"returnable_notes,omitempty"`
	// FieldDefaults overrides a skeleton field default for this discipline.
	FieldDefaults map[string]string `yaml:"field_defaults" json:"field_defaults,omitempty"`
	Durations     string            `yaml:"durations" json:"durations"`
}

type RFPOption struct {
	Provenance []RFPProvenance `yaml:"provenance" json:"provenance,omitempty"`
	// ScopeDecision is an appointment choice, printed only by RFPs that use it.
	ScopeDecision bool   `yaml:"scope_decision" json:"scope_decision,omitempty"`
	Key           string `yaml:"key" json:"key"`
	Label         string `yaml:"label" json:"label"`
	Help          string `yaml:"help" json:"help,omitempty"`
}

type RFPField struct {
	Key      string `yaml:"key" json:"key"`
	Label    string `yaml:"label" json:"label"`
	Kind     string `yaml:"kind" json:"kind"` // text | date | money | number
	Required bool   `yaml:"required" json:"required"`
	Group    string `yaml:"group" json:"group"` // issue | terms | project | appointment
	Default  string `yaml:"default" json:"default,omitempty"`
	Help     string `yaml:"help" json:"help,omitempty"`
	// Profile is the field's one home when the project, not the RFP, owns
	// it: proj.<detail>, site.address or site.lot. The RFP reads it and
	// never stores its own value (owner decision D1).
	Profile string `yaml:"profile" json:"profile,omitempty"`
}

// GridRow is one labelled value in the project grid on RFP page one. The
// columns mirror the Profile card's (web/src/Profile.tsx COLUMNS), so a fact
// sits in the same place on both. The RFP fills each row from facts it
// already holds; a discipline may replace a row's wording (replace: true).
type GridRow struct {
	Key       string
	Label     string
	Column    string
	Essential bool // printed even when unknown
}

var ProjectGridColumns = []string{"The building", "Site and planning", "Delivery"}

var ProjectGrid = []GridRow{
	{"work_type", "Work type", "The building", true},
	{"parts", "Building", "The building", true},
	{"storeys", "Storeys", "The building", false},
	{"gfa", "Gross floor area", "The building", false},
	{"parking", "Car parks", "The building", false},
	{"address", "Site address", "Site and planning", true},
	{"lot_dp", "Lot and DP", "Site and planning", false},
	{"lga", "Local government area", "Site and planning", false},
	{"zone", "Zone", "Site and planning", false},
	{"site_area", "Site area", "Site and planning", false},
	{"pathway", "Planning pathway", "Site and planning", true},
	{"client", "Client", "Delivery", true},
	{"procurement", "Procurement route", "Delivery", true},
	{"program", "Program milestones", "Delivery", false},
	{"sustainability", "Sustainability target", "Delivery", false},
	{"regulated_design", "Regulated design applies", "Delivery", false},
}

type RFPWeight struct {
	Provenance []RFPProvenance `yaml:"provenance" json:"provenance,omitempty"`
	Key        string          `yaml:"key" json:"key"`
	Label      string          `yaml:"label" json:"label"`
	Default    int             `yaml:"default" json:"default"`
}

type RFPReturnable struct {
	Key   string `yaml:"key" json:"key"`
	Title string `yaml:"title" json:"title"`
	Text  string `yaml:"text" json:"text"`
}

type RFPModule struct {
	Key   string  `yaml:"key" json:"key"`
	Label string  `yaml:"label" json:"label"`
	When  RFPWhen `yaml:"when" json:"when"`
	// IfUnknown applies when the activation is unknown: ask raises a decision,
	// include keeps the module's rows with their stated assumptions.
	IfUnknown  string          `yaml:"if_unknown" json:"if_unknown,omitempty"`
	Provenance []RFPProvenance `yaml:"provenance" json:"provenance,omitempty"`
}

type RFPSkeleton struct {
	Version     int             `yaml:"version" json:"version"`
	Status      string          `yaml:"status" json:"status"`
	Golden      string          `yaml:"golden" json:"golden"`
	Flags       []RFPOption     `yaml:"flags" json:"flags"`
	Pathways    []RFPOption     `yaml:"pathways" json:"pathways"`
	Procurement []RFPOption     `yaml:"procurement" json:"procurement"`
	WorkTypes   []RFPOption     `yaml:"work_types" json:"work_types"`
	Fields      []RFPField      `yaml:"fields" json:"fields"`
	Evaluation  []RFPWeight     `yaml:"evaluation" json:"evaluation"`
	Thresholds  []RFPThreshold  `yaml:"thresholds" json:"thresholds"`
	ProjectRows []RFPRow        `yaml:"project_rows" json:"project_rows"`
	Terms       []RFPRow        `yaml:"terms" json:"terms"`
	Returnables []RFPReturnable `yaml:"returnables" json:"returnables"`
	Legend      string          `yaml:"legend" json:"legend"`
	Departures  string          `yaml:"departures" json:"departures"`
	Allowances  string          `yaml:"allowances" json:"allowances"`
	Evaluate    string          `yaml:"evaluation_note" json:"evaluation_note"`
	Modules     []RFPModule     `yaml:"modules" json:"modules"`
}

type rfpData struct {
	skeleton    *RFPSkeleton
	disciplines map[string]RFPDiscipline
}

// RFPSkeleton returns the shared RFP parts, or false when none are loaded.
func (c *Catalog) RFPSkeleton() (RFPSkeleton, bool) {
	if c.rfp.skeleton == nil {
		return RFPSkeleton{}, false
	}
	return *c.rfp.skeleton, true
}

func (c *Catalog) RFP(id string) (RFPDiscipline, bool) {
	d, ok := c.rfp.disciplines[id]
	return d, ok
}

// RFPDisciplines lists loaded discipline ids in order.
func (c *Catalog) RFPDisciplines() []string {
	ids := make([]string, 0, len(c.rfp.disciplines))
	for id := range c.rfp.disciplines {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	return ids
}

// RFPForDiscipline maps a package's stable discipline id to a discipline.
func (c *Catalog) RFPForDiscipline(disciplineID string) (RFPDiscipline, bool) {
	if disciplineID == "" {
		return RFPDiscipline{}, false
	}
	for _, id := range c.RFPDisciplines() {
		d := c.rfp.disciplines[id]
		if slices.Contains(d.DisciplineIDs, disciplineID) {
			return d, true
		}
	}
	return RFPDiscipline{}, false
}

// RFPSuggest proposes a discipline from a package title. It is a hint for
// the user to confirm; a free-text title never selects contract scope.
func (c *Catalog) RFPSuggest(title string) (RFPDiscipline, bool) {
	normalize := func(s string) string {
		s = strings.ToLower(strings.NewReplacer("-", " ", "/", " ", ",", " ", "(", " ", ")", " ", ".", " ").Replace(s))
		return strings.Join(strings.Fields(s), " ")
	}
	text := " " + normalize(title) + " "
	best, score, ambiguous := RFPDiscipline{}, 0, false
	for _, id := range c.RFPDisciplines() {
		d := c.rfp.disciplines[id]
		for _, alias := range d.TitleAliases {
			phrase := normalize(alias)
			if phrase == "" || !strings.Contains(text, " "+phrase+" ") {
				continue
			}
			n := len(strings.Fields(phrase))
			if n > score {
				best, score, ambiguous = d, n, false
			} else if n == score && best.ID != d.ID {
				ambiguous = true
			}
		}
	}
	return best, score > 0 && !ambiguous
}

func (c *Catalog) loadRFP(dir string) error {
	c.rfp = rfpData{disciplines: map[string]RFPDiscipline{}}
	if ok, err := exists(dir); !ok {
		return err
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		return err
	}
	read := func(name string) ([]byte, error) {
		path := filepath.Join(dir, name)
		body, err := os.ReadFile(path)
		if err != nil {
			return nil, err
		}
		return body, c.remember(path, body)
	}
	body, err := read("skeleton.yaml")
	if err != nil {
		return err
	}
	skeleton, err := parseRFPSkeleton(body)
	if err != nil {
		return fmt.Errorf("%s: %w", filepath.Join(dir, "skeleton.yaml"), err)
	}
	if ok, _ := exists(filepath.Join(dir, "modules.yaml")); ok {
		body, err := read("modules.yaml")
		if err != nil {
			return err
		}
		var file struct {
			Version int         `yaml:"version"`
			Status  string      `yaml:"status"`
			Modules []RFPModule `yaml:"modules"`
		}
		if err := strictYAML(body, &file); err != nil {
			return fmt.Errorf("%s: %w", filepath.Join(dir, "modules.yaml"), err)
		}
		skeleton.Modules = append(skeleton.Modules, file.Modules...)
		if err := checkModules(skeleton); err != nil {
			return fmt.Errorf("%s: %w", filepath.Join(dir, "modules.yaml"), err)
		}
	}
	c.rfp.skeleton = &skeleton
	for _, entry := range entries {
		name := entry.Name()
		if entry.IsDir() || !strings.HasSuffix(name, ".yaml") || name == "skeleton.yaml" || name == "modules.yaml" {
			continue
		}
		body, err := read(name)
		if err != nil {
			return err
		}
		d, err := ParseRFPDiscipline(body, skeleton)
		if err != nil {
			return fmt.Errorf("%s: %w", filepath.Join(dir, name), err)
		}
		if d.ID+".yaml" != name {
			return fmt.Errorf("%s: id %q does not match the file name", filepath.Join(dir, name), d.ID)
		}
		c.rfp.disciplines[d.ID] = d
	}
	return nil
}

func strictYAML(body []byte, dest any) error {
	dec := yaml.NewDecoder(bytes.NewReader(body))
	dec.KnownFields(true)
	return dec.Decode(dest)
}

func decodeKnown(n *yaml.Node, dest any) error {
	raw, err := yaml.Marshal(n)
	if err != nil {
		return err
	}
	return strictYAML(raw, dest)
}

func parseRFPSkeleton(body []byte) (RFPSkeleton, error) {
	var s RFPSkeleton
	if err := strictYAML(body, &s); err != nil {
		return s, err
	}
	if s.Version != 1 || !slices.Contains([]string{"draft", "reviewed"}, s.Status) {
		return s, fmt.Errorf("skeleton needs version 1 and status draft or reviewed")
	}
	for name, list := range map[string][]RFPOption{"flags": s.Flags, "pathways": s.Pathways, "procurement": s.Procurement, "work_types": s.WorkTypes} {
		if len(list) == 0 {
			return s, fmt.Errorf("skeleton %s is empty", name)
		}
		if err := uniqueKeys(name, len(list), func(i int) string { return list[i].Key }); err != nil {
			return s, err
		}
		for _, option := range list {
			if err := checkProvenance(name+" "+option.Key, option.Provenance); err != nil {
				return s, err
			}
		}
	}
	if err := uniqueKeys("fields", len(s.Fields), func(i int) string { return s.Fields[i].Key }); err != nil {
		return s, err
	}
	for _, f := range s.Fields {
		if !slices.Contains([]string{"text", "date", "money", "number"}, f.Kind) || !slices.Contains([]string{"issue", "terms", "project", "appointment"}, f.Group) || f.Label == "" {
			return s, fmt.Errorf("field %q needs a label, kind and group", f.Key)
		}
		if f.Profile != "" && (f.Group != "project" || (f.Profile != "site.address" && f.Profile != "site.lot" && !strings.HasPrefix(f.Profile, DetailPrefix))) {
			return s, fmt.Errorf("field %q: profile home %q must be proj.<detail>, site.address or site.lot on a project field", f.Key, f.Profile)
		}
	}
	total := 0
	for _, w := range s.Evaluation {
		total += w.Default
	}
	if len(s.Evaluation) == 0 || total != 100 {
		return s, fmt.Errorf("evaluation defaults total %d%%, must be 100%%", total)
	}
	if err := checkThresholds(s.Thresholds); err != nil {
		return s, err
	}
	if err := uniqueKeys("returnables", len(s.Returnables), func(i int) string { return s.Returnables[i].Key }); err != nil {
		return s, err
	}
	for _, r := range s.Returnables {
		if !regexp.MustCompile(`^r[1-9]$`).MatchString(r.Key) || r.Title == "" {
			return s, fmt.Errorf("returnable %q needs a key r1-r9 and a title", r.Key)
		}
	}
	for _, rows := range [][]RFPRow{s.ProjectRows, s.Terms} {
		if err := checkRows(rows, s); err != nil {
			return s, err
		}
	}
	return s, checkModules(s)
}

func checkModules(s RFPSkeleton) error {
	if err := uniqueKeys("modules", len(s.Modules), func(i int) string { return s.Modules[i].Key }); err != nil {
		return err
	}
	for _, m := range s.Modules {
		if m.Label == "" || !validIfUnknown(m.IfUnknown) {
			return fmt.Errorf("module %q needs a label and a valid if_unknown", m.Key)
		}
		if err := checkWhen("module "+m.Key, m.When, s); err != nil {
			return err
		}
		if err := checkProvenance("module "+m.Key, m.Provenance); err != nil {
			return err
		}
	}
	return nil
}

// ParseRFPDiscipline decodes and validates one discipline against the
// skeleton's vocabulary. Errors name the offending key.
func ParseRFPDiscipline(body []byte, s RFPSkeleton) (RFPDiscipline, error) {
	var d RFPDiscipline
	if err := strictYAML(body, &d); err != nil {
		return d, err
	}
	if !regexp.MustCompile(`^[a-z-]+$`).MatchString(d.ID) || d.Version < 1 || !slices.Contains([]string{"draft", "reviewed"}, d.Status) || d.Title == "" || d.Heading == "" {
		return d, fmt.Errorf("discipline needs an id, version, title, heading and status draft or reviewed")
	}
	if d.Status == "reviewed" && (strings.TrimSpace(d.ReviewedBy) == "" || strings.TrimSpace(d.ReviewedOn) == "") {
		return d, fmt.Errorf("discipline %s is reviewed without reviewed_by and reviewed_on; only the owner marks it reviewed", d.ID)
	}
	if len(d.DisciplineIDs) == 0 || len(d.Stages) == 0 {
		return d, fmt.Errorf("discipline %s needs discipline_ids and stages", d.ID)
	}
	if err := checkThresholds(d.Thresholds); err != nil {
		return d, err
	}
	thresholds := map[string]RFPThreshold{}
	for _, t := range append(append([]RFPThreshold(nil), s.Thresholds...), d.Thresholds...) {
		thresholds[t.Key] = t
	}
	for _, p := range d.UnsupportedPathways {
		if !optionKey(s.Pathways, p) {
			return d, fmt.Errorf("unsupported pathway %q is not in the skeleton", p)
		}
	}
	if err := checkWhen("scale", d.Scale.Complex, s); err != nil {
		return d, err
	}
	if len(d.Evaluation) > 0 {
		if err := uniqueKeys("evaluation", len(d.Evaluation), func(i int) string { return d.Evaluation[i].Key }); err != nil {
			return d, err
		}
		total := 0
		for _, w := range d.Evaluation {
			if w.Label == "" || w.Default < 0 || w.Default > 100 {
				return d, fmt.Errorf("evaluation %s needs a label and percentage", w.Key)
			}
			if err := checkProvenance("evaluation "+w.Key, w.Provenance); err != nil {
				return d, err
			}
			total += w.Default
		}
		if total != 100 {
			return d, fmt.Errorf("discipline evaluation must total 100")
		}
	}
	for _, group := range []struct {
		name        string
		own, shared []RFPRow
	}{
		{"term", d.Terms, s.Terms}, {"project", d.ProjectRows, s.ProjectRows},
	} {
		for _, row := range group.own {
			if !row.Replace {
				continue
			}
			found := false
			for _, shared := range group.shared {
				found = found || shared.Key == row.Key
			}
			if group.name == "project" {
				for _, g := range ProjectGrid {
					found = found || g.Key == row.Key
				}
			}
			if !found {
				return d, fmt.Errorf("%s replacement %q names no shared row", group.name, row.Key)
			}
		}
	}
	for _, n := range d.ReturnableNotes {
		if !regexp.MustCompile(`^r[1-9]$`).MatchString(n.Key) {
			return d, fmt.Errorf("returnable note %q must name r1-r9", n.Key)
		}
		if strings.TrimSpace(n.Text) == "" {
			return d, fmt.Errorf("returnable note %s needs text", n.Key)
		}
		if err := checkProvenance("returnable note "+n.Key, n.Provenance); err != nil {
			return d, err
		}
		if err := checkWhen("returnable note "+n.Key, n.When, s); err != nil {
			return d, err
		}
	}
	for _, rows := range [][]RFPRow{d.ProjectRows, d.Role, d.Terms, d.ScopeSchedule} {
		if err := checkRows(rows, s); err != nil {
			return d, err
		}
	}
	for _, row := range d.ScopeSchedule {
		// Unknown allocations must remain visible for the parties to resolve.
		if len(row.When) > 0 || row.Module != "" || row.Replace || len(row.Provenance) == 0 {
			return d, fmt.Errorf("scope schedule %s must be unconditional and have provenance", row.Key)
		}
	}
	stageKeys := map[string]bool{}
	for _, st := range d.Stages {
		stageKeys[st.Key] = true
	}
	elementKeys := map[string]bool{}
	for _, e := range d.Elements {
		elementKeys[e.Key] = true
	}
	if err := uniqueKeys("stages", len(d.Stages), func(i int) string { return d.Stages[i].Key }); err != nil {
		return d, err
	}
	positions := map[string]RFPPosition{}
	for _, p := range d.Positions {
		positions[p.Key] = p
	}
	for i, st := range d.Stages {
		where := "stage " + st.Key
		if st.Name == "" || st.Deliverable == "" || st.ShownWhen == "" {
			return d, fmt.Errorf("%s needs a name, deliverable and shown_when", where)
		}
		if !slices.Contains([]string{"fixed", "time_to_ceiling", "per_unit", "option"}, st.FeeBasis) {
			return d, fmt.Errorf("%s: fee_basis %q must be fixed, time_to_ceiling, per_unit or option", where, st.FeeBasis)
		}
		if !slices.Contains([]string{"", "include", "option"}, st.Default) {
			return d, fmt.Errorf("%s: default %q must be include or option", where, st.Default)
		}
		if st.Follows != "" && (!stageKeys[st.Follows] || indexOfStage(d.Stages, st.Follows) >= i) {
			return d, fmt.Errorf("%s follows %q, which must be an earlier stage", where, st.Follows)
		}
		for _, item := range st.Items {
			p, ok := positions[item]
			if !ok {
				return d, fmt.Errorf("%s lists unknown item %q", where, item)
			}
			if staticallyExcluded(p) {
				return d, fmt.Errorf("%s includes item %q, whose position is always X", where, item)
			}
		}
		if err := checkConditional(where, st.When, st.Suggest, st.IfUnknown, st.Assumption, st.Threshold, st.Module, thresholds, s); err != nil {
			return d, err
		}
		if err := checkWhen(where+" include_when", st.IncludeWhen, s); err != nil {
			return d, err
		}
		if err := checkProvenance(where, st.Provenance); err != nil {
			return d, err
		}
	}
	if err := uniqueKeys("elements", len(d.Elements), func(i int) string { return d.Elements[i].Key }); err != nil {
		return d, err
	}
	for _, e := range d.Elements {
		if e.Code == "" || e.Name == "" || e.ShownWhen == "" {
			return d, fmt.Errorf("element %s needs a code, name and shown_when", e.Key)
		}
		if err := checkConditional("element "+e.Key, e.When, nil, e.IfUnknown, e.Assumption, "", "", thresholds, s); err != nil {
			return d, err
		}
		if err := checkProvenance("element "+e.Key, e.Provenance); err != nil {
			return d, err
		}
	}
	if err := uniqueKeys("allowances", len(d.Allowances), func(i int) string { return d.Allowances[i].Key }); err != nil {
		return d, err
	}
	for _, a := range d.Allowances {
		where := "allowance " + a.Key
		if a.Label == "" || !strings.Contains(a.Text, "{n}") || !slices.Contains([]string{"client", "consultant"}, a.Owner) {
			return d, fmt.Errorf("%s needs a label, a {n} slot and owner client or consultant", where)
		}
		if err := checkWhen(where, a.When, s); err != nil {
			return d, err
		}
		if err := checkModule(where, a.Module, s); err != nil {
			return d, err
		}
		if err := checkProvenance(where, a.Provenance); err != nil {
			return d, err
		}
	}
	if err := uniqueKeys("positions", len(d.Positions), func(i int) string { return d.Positions[i].Key }); err != nil {
		return d, err
	}
	for _, p := range d.Positions {
		if err := checkPosition(p, thresholds, s); err != nil {
			return d, err
		}
	}
	for _, b := range d.Boundary {
		if !stageKeys[b.Start] || !stageKeys[b.End] || indexOfStage(d.Stages, b.Start) > indexOfStage(d.Stages, b.End) {
			return d, fmt.Errorf("boundary %s..%s needs two stages in order", b.Start, b.End)
		}
		if err := checkWhen("boundary", b.When, s); err != nil {
			return d, err
		}
	}
	if len(d.Boundary) == 0 || len(d.Boundary[len(d.Boundary)-1].When) != 0 {
		return d, fmt.Errorf("boundary needs a final suggestion with no condition")
	}
	// A discipline sets defaults for the RFP's own terms (owner decision D3);
	// a field with a Profile home belongs to the project, not the discipline.
	for key := range d.FieldDefaults {
		found := false
		for _, f := range s.Fields {
			if f.Key == key && f.Profile != "" {
				return d, fmt.Errorf("field default %q: the field lives in Profile (%s)", key, f.Profile)
			}
			found = found || f.Key == key
		}
		if !found {
			return d, fmt.Errorf("field default %q names no skeleton field", key)
		}
	}
	for key := range d.Returnables {
		if !regexp.MustCompile(`^r[1-9]$`).MatchString(key) {
			return d, fmt.Errorf("returnable %q must be r1-r9", key)
		}
	}
	// stage and element conditions may only name keys that exist.
	check := func(where string, w RFPWhen) error {
		for _, clause := range w {
			for _, v := range append(append([]string(nil), clause["stage"]...), clause["stage_included"]...) {
				if !stageKeys[v] {
					return fmt.Errorf("%s names unknown stage %q", where, v)
				}
			}
			for _, v := range clause["element"] {
				if !elementKeys[v] {
					return fmt.Errorf("%s names unknown element %q", where, v)
				}
			}
		}
		return nil
	}
	for _, p := range d.Positions {
		if err := check("position "+p.Key, p.When); err != nil {
			return d, err
		}
		for _, cell := range []*RFPCell{p.Value, p.Simple, p.Complex} {
			if cell != nil {
				if err := check("position "+p.Key, cell.When); err != nil {
					return d, err
				}
			}
		}
	}
	for _, a := range d.Allowances {
		if err := check("allowance "+a.Key, a.When); err != nil {
			return d, err
		}
	}
	for _, st := range d.Stages {
		if err := check("stage "+st.Key, st.When); err != nil {
			return d, err
		}
	}
	return d, nil
}

func checkPosition(p RFPPosition, thresholds map[string]RFPThreshold, s RFPSkeleton) error {
	where := "position " + p.Key
	if p.Label == "" {
		return fmt.Errorf("%s needs a label", where)
	}
	cells := []*RFPCell{p.Value, p.Simple, p.Complex}
	switch p.Kind {
	case "technical":
		if p.Value == nil || p.Simple != nil || p.Complex != nil {
			return fmt.Errorf("%s is technical: give one value cell, not simple/complex columns", where)
		}
		if usesScale(p.Value.When) || usesScale(p.When) {
			return fmt.Errorf("%s is technical and must not depend on scale", where)
		}
	case "service_level":
		if p.Value != nil || p.Simple == nil || p.Complex == nil {
			return fmt.Errorf("%s is a service level: give simple and complex cells", where)
		}
	default:
		return fmt.Errorf("%s: kind must be technical or service_level", where)
	}
	canExclude := false
	for _, c := range cells {
		if c == nil {
			continue
		}
		for _, v := range []string{c.Value, c.Else} {
			if v != "" && !slices.Contains([]string{"I", "O", "X", "-"}, v) {
				return fmt.Errorf("%s: position %q must be I, O, X or -", where, v)
			}
			canExclude = canExclude || v == "X"
		}
		if c.Value == "" {
			return fmt.Errorf("%s: empty position", where)
		}
		if err := checkWhen(where, c.When, s); err != nil {
			return err
		}
	}
	if canExclude && p.Responsible == "" {
		return fmt.Errorf("%s can be X and needs a responsible party", where)
	}
	if err := checkConditional(where, p.When, p.Suggest, p.IfUnknown, p.Assumption, p.Threshold, p.Module, thresholds, s); err != nil {
		return err
	}
	if len(p.Provenance) == 0 {
		return fmt.Errorf("%s needs provenance", where)
	}
	return checkProvenance(where, p.Provenance)
}

func staticallyExcluded(p RFPPosition) bool {
	all := []*RFPCell{p.Value, p.Simple, p.Complex}
	seen := false
	for _, c := range all {
		if c == nil {
			continue
		}
		seen = true
		if c.Value != "X" || (len(c.When) > 0 && c.Else != "X") {
			return false
		}
	}
	return seen
}

func usesScale(w RFPWhen) bool {
	for _, clause := range w {
		if _, ok := clause["scale"]; ok {
			return true
		}
	}
	return false
}

func checkConditional(where string, when, suggest RFPWhen, ifUnknown, assumption, threshold, module string, thresholds map[string]RFPThreshold, s RFPSkeleton) error {
	if err := checkWhen(where, when, s); err != nil {
		return err
	}
	if err := checkWhen(where+" suggest", suggest, s); err != nil {
		return err
	}
	if !validIfUnknown(ifUnknown) {
		return fmt.Errorf("%s: if_unknown %q must be ask, option or include", where, ifUnknown)
	}
	if (ifUnknown == "option" || ifUnknown == "include") && strings.TrimSpace(assumption) == "" {
		return fmt.Errorf("%s: if_unknown %s needs the assumption wording", where, ifUnknown)
	}
	if err := checkModule(where, module, s); err != nil {
		return err
	}
	if threshold == "" {
		return nil
	}
	t, ok := thresholds[threshold]
	if !ok {
		return fmt.Errorf("%s names unknown threshold %q", where, threshold)
	}
	if t.Kind == "statutory" && (t.Verified == nil || !*t.Verified) {
		if len(suggest) == 0 {
			return fmt.Errorf("%s: unverified statutory threshold %q may only raise a decision through suggest", where, threshold)
		}
		for _, clause := range when {
			for _, key := range rawFactDrivers {
				if _, ok := clause[key]; ok {
					return fmt.Errorf("%s: unverified statutory threshold %q cannot select scope from %s; use suggest", where, threshold, key)
				}
			}
		}
	}
	return nil
}

func validIfUnknown(v string) bool {
	return slices.Contains([]string{"", "ask", "option", "include"}, v)
}

func checkModule(where, module string, s RFPSkeleton) error {
	if module == "" {
		return nil
	}
	for _, m := range s.Modules {
		if m.Key == module {
			return nil
		}
	}
	return fmt.Errorf("%s names unknown module %q", where, module)
}

func checkWhen(where string, w RFPWhen, s RFPSkeleton) error {
	for _, clause := range w {
		if len(clause) == 0 {
			return fmt.Errorf("%s has an empty condition clause", where)
		}
		for key, values := range clause {
			if !RFPDrivers[key] {
				return fmt.Errorf("%s: unknown driver %q", where, key)
			}
			if len(values) == 0 {
				return fmt.Errorf("%s: driver %q lists no values", where, key)
			}
			for _, v := range values {
				var ok bool
				switch key {
				case "flag":
					ok = optionKey(s.Flags, v)
				case "pathway":
					ok = optionKey(s.Pathways, v)
				case "procurement":
					ok = optionKey(s.Procurement, v)
				case "work_type":
					ok = optionKey(s.WorkTypes, v)
				case "scale":
					ok = v == "simple" || v == "complex"
				case "novation", "regulated_design":
					ok = v == "yes" || v == "no"
				default:
					ok = v != ""
				}
				if !ok {
					return fmt.Errorf("%s: %s value %q is not in the vocabulary", where, key, v)
				}
			}
		}
	}
	return nil
}

func checkRows(rows []RFPRow, s RFPSkeleton) error {
	if err := uniqueKeys("rows", len(rows), func(i int) string { return rows[i].Key }); err != nil {
		return err
	}
	for _, r := range rows {
		if strings.TrimSpace(r.Text) == "" {
			return fmt.Errorf("row %q needs text", r.Key)
		}
		if err := checkWhen("row "+r.Key, r.When, s); err != nil {
			return err
		}
		if err := checkModule("row "+r.Key, r.Module, s); err != nil {
			return err
		}
		if err := checkProvenance("row "+r.Key, r.Provenance); err != nil {
			return err
		}
	}
	return nil
}

func checkThresholds(list []RFPThreshold) error {
	if err := uniqueKeys("thresholds", len(list), func(i int) string { return list[i].Key }); err != nil {
		return err
	}
	for _, t := range list {
		if !slices.Contains([]string{"statutory", "client_default", "evidence_estimate"}, t.Kind) || t.Verified == nil || t.Text == "" || t.Source == "" {
			return fmt.Errorf("threshold %q needs kind, text, source and verified", t.Key)
		}
	}
	return nil
}

func checkProvenance(where string, list []RFPProvenance) error {
	for _, p := range list {
		if !slices.Contains([]string{"evidence", "seed", "practice", "client_decision"}, p.Kind) || p.Ref == "" {
			return fmt.Errorf("%s: provenance needs kind evidence, seed, practice or client_decision and a ref", where)
		}
		if p.Kind == "client_decision" && strings.TrimSpace(p.Why) == "" {
			return fmt.Errorf("%s: a client_decision provenance needs a why", where)
		}
	}
	return nil
}

func uniqueKeys(what string, n int, key func(int) string) error {
	seen := map[string]bool{}
	keyRE := regexp.MustCompile(`^[a-z0-9_]+$`)
	for i := 0; i < n; i++ {
		k := key(i)
		if !keyRE.MatchString(k) || seen[k] {
			return fmt.Errorf("%s: invalid or duplicate key %q", what, k)
		}
		seen[k] = true
	}
	return nil
}

func optionKey(list []RFPOption, key string) bool {
	for _, o := range list {
		if o.Key == key {
			return true
		}
	}
	return false
}

func indexOfStage(stages []RFPStage, key string) int {
	for i, s := range stages {
		if s.Key == key {
			return i
		}
	}
	return -1
}

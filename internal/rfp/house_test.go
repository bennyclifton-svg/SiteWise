package rfp

import (
	"encoding/json"
	"strings"
	"testing"

	"sitewise/internal/procurement"
	"sitewise/internal/reports"
)

func TestParseMoney(t *testing.T) {
	for text, want := range map[string]string{
		"$1.8 million excluding GST": "1800000",
		"$1,250,000 (incl GST)":      "1250000",
		"$450k":                      "450000",
		"AUD 2.1m":                   "2100000",
	} {
		if got, ok := parseMoney(text); !ok || got != want {
			t.Errorf("%q: want %s, got %s %v", text, want, got, ok)
		}
	}
	if _, ok := parseMoney("to be confirmed"); ok {
		t.Fatal("no amount, no budget")
	}
	if b := statedBasis("$1.8 million excluding GST"); !strings.HasPrefix(b, "excl. GST") {
		t.Fatalf("GST basis from the passage: %q", b)
	}
	if b := statedBasis("$1.25m incl. GST"); !strings.HasPrefix(b, "incl. GST") {
		t.Fatalf("GST basis from the passage: %q", b)
	}
}

// houseSnapshot is Custom Home 2 as Profile holds it: a garage counted as
// garage spaces, bedrooms, and a budget a brief stated but the cost plan
// does not yet hold.
func houseSnapshot() reports.Snapshot {
	brief := func(key, text string) reports.BriefValue {
		return reports.BriefValue{ID: "w:" + key, Label: key, Text: text}
	}
	doc := func(kind, discipline string) reports.BriefValue {
		basis, _ := json.Marshal(map[string]any{"sources": []any{map[string]any{"kind": kind, "discipline": discipline}}})
		return reports.BriefValue{ID: kind + discipline, Basis: basis}
	}
	return reports.Snapshot{
		ProjectName:  "Custom Home 2",
		ProjectBasis: json.RawMessage(`{"project":{"name":"Custom Home 2"},"site":{"address":"357 Catherine Street Lilyfield NSW 2040"}}`),
		Package:      procurement.Package{ID: "a", Kind: "services", Title: "Architect", Novation: "no"},
		Parts:        []reports.PartInfo{{ID: "w", Label: "Whole project", Kind: "whole", NCCClass: "1a"}},
		Brief: []reports.BriefValue{
			brief("hdr.work_type", "new"), brief("hdr.building_class", "residential"),
			brief("hdr.scale.garage_spaces", "1"), brief("hdr.scale.bedrooms", "4"), brief("hdr.scale.bathrooms", "3"),
			brief("hdr.cond.quality_level", "mid"), brief("proj.budget_stated", "$1.8 million excluding GST"),
		},
		Packages: []procurement.Package{
			{ID: "a", Kind: "services", Title: "Architect", LifecycleStatus: "planned"},
			{ID: "s", Kind: "services", Title: "Structural", LifecycleStatus: "planned", EngagedBy: procurement.EngagedByLead},
			{ID: "g", Kind: "services", Title: "Geotechnical", LifecycleStatus: "planned"},
		},
		Documents: []reports.BriefValue{doc("design_brief", ""), doc("report", "consultant.geotechnical"), doc("drawing", "consultant.surveyor"), doc("drawing", "consultant.surveyor")},
	}
}

func TestDriversReadTheHouseBrief(t *testing.T) {
	d := Drivers(houseSnapshot(), skeleton(t))
	for key, want := range map[string]string{"parking": "1", "bedrooms": "4", "bathrooms": "3", "quality": "mid", "budget": "1800000"} {
		if f := d.Fact(key); f.State != Known || f.Values[0] != want {
			t.Errorf("%s: want %s, got %+v", key, want, f)
		}
	}
	if b := d.Fact("budget"); b.Source != "profile" || !strings.Contains(b.Detail, "not yet in the cost plan") {
		t.Fatalf("a stated budget says where it came from: %+v", b)
	}
	if got := d.Text["register"]; !strings.Contains(got, "Geotechnical report (1)") || !strings.Contains(got, "Surveyor drawing (2)") || !strings.Contains(got, "Design Brief (1)") {
		t.Fatalf("register grouped by discipline and kind: %q", got)
	}
	engaged := map[string]string{}
	for _, c := range d.Consultants {
		engaged[c.Title] = c.EngagedBy
	}
	if engaged["Structural"] != EngagedByLead || engaged["Geotechnical"] != "" {
		t.Fatalf("engagement comes from each package: %+v", engaged)
	}
}

func TestCostPlanBudgetWinsOverAStatedOne(t *testing.T) {
	s := houseSnapshot()
	s.ConstructionBudget, s.BudgetBasis = "1650000", "AUD excl. GST"
	if b := Drivers(s, skeleton(t)).Fact("budget"); b.Values[0] != "1650000" || b.Source != "cost_plan" {
		t.Fatalf("the cost plan's budget wins: %+v", b)
	}
}

func TestHouseRFPReadsTheBrief(t *testing.T) {
	sk, d := skeleton(t), discipline(t, "architecture")
	r := Resolve(d, sk, Drivers(houseSnapshot(), sk), Settings{})
	if r.Modules["house"] != "on" || r.Modules["beyond_house"] != "off" {
		t.Fatalf("a Class 1a whole project is a house: %+v", r.Modules)
	}
	grid := map[string]string{}
	for _, c := range r.Grid {
		for _, cell := range c.Cells {
			grid[cell.Key] = cell.Text
		}
	}
	if grid["bedrooms"] != "4" || grid["bathrooms"] != "3" || grid["parking"] != "1" || grid["quality"] != qualityLabels["mid"] {
		t.Fatalf("the grid prints the house: %+v", grid)
	}
	if r.Fields["pi"].Value != "$10 million" {
		t.Fatalf("a house falls back to the shared PI default: %+v", r.Fields["pi"])
	}
	role := ""
	for _, row := range r.Role {
		role += row.Text
	}
	if !strings.Contains(role, "subconsultants") || !strings.Contains(role, "Structural") || !strings.Contains(role, "still to be decided: Geotechnical") {
		t.Fatalf("role names the subconsultant and the open decision: %q", role)
	}
	if !r.HasCheck("engagement") || r.HasCheck("fact:budget") {
		t.Fatalf("an undecided package blocks; a stated budget satisfies the budget: %s", describeChecks(r))
	}
	info := ""
	for _, row := range r.Project {
		if row.Key == "information" {
			info = row.Text
		}
	}
	if !strings.Contains(info, "Geotechnical report (1)") {
		t.Fatalf("information provided follows the register: %q", info)
	}
}

// A house with a Class 10 garage is a house; a house over a shop is not, and
// a part with no class keeps the commercial rows until it is known.
func TestHouseModuleFollowsEveryPart(t *testing.T) {
	sk, d := skeleton(t), discipline(t, "architecture")
	resolve := func(parts ...Part) Resolved {
		facts := map[string]Fact{"work_type": {Values: []string{"new"}, State: Known}}
		return Resolve(d, sk, DriverSet{Facts: facts, Parts: parts, Text: map[string]string{}}, Settings{})
	}
	r := resolve(Part{ID: "h", Classes: []string{"1a"}}, Part{ID: "g", Classes: []string{"10a"}})
	if r.Modules["house"] != "on" || r.Modules["beyond_house"] != "off" {
		t.Fatalf("house and garage: %+v", r.Modules)
	}
	r = resolve(Part{ID: "h", Classes: []string{"1a"}}, Part{ID: "s", Classes: []string{"5"}})
	if r.Modules["house"] != "off" || r.Modules["beyond_house"] != "on" || r.Position("signage").Key == "" {
		t.Fatalf("a house with an office part is beyond a house: %+v", r.Modules)
	}
	r = resolve(Part{ID: "b"})
	if r.Modules["house"] != "unknown" || r.Modules["beyond_house"] != "unknown" || r.Position("signage").Key == "" || r.Position("house_interiors").Key != "" {
		t.Fatalf("unknown class keeps commercial rows and leaves house rows out: %+v", r.Modules)
	}
}

func TestQualityLabelsMatchTaxonomy(t *testing.T) {
	for _, c := range loadCatalog(t).Taxonomy().Conditions {
		if c.Key != "quality_level" {
			continue
		}
		if len(c.Options) != len(qualityLabels) {
			t.Fatalf("quality options %d, labels %d", len(c.Options), len(qualityLabels))
		}
		for _, o := range c.Options {
			if qualityLabels[o.ID] == "" {
				t.Errorf("no RFP label for quality %s", o.ID)
			}
		}
		return
	}
	t.Fatal("taxonomy has no quality_level condition")
}

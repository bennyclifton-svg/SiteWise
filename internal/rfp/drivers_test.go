package rfp

import (
	"encoding/json"
	"slices"
	"testing"

	"sitewise/internal/procurement"
	"sitewise/internal/reports"
)

func arthurSnapshot() reports.Snapshot {
	brief := func(part, key, text string, unknown bool) reports.BriefValue {
		return reports.BriefValue{ID: part + ":" + key, Label: key, Text: text, Unknown: unknown}
	}
	return reports.Snapshot{
		ProjectName:  "Arthur St",
		ProjectBasis: json.RawMessage(`{"project":{"name":"Arthur St"},"site":{"address":"145-151 Arthur St, Homebush West NSW","lot":"Lot 1 DP 1"}}`),
		Package:      procurement.Package{ID: "p1", Kind: "services", Title: "Architect", Novation: false},
		Parts: []reports.PartInfo{
			{ID: "w", Label: "Whole project", Kind: "whole"},
			{ID: "wh", Label: "Warehouse", Kind: "building", NCCClass: "7b"},
			{ID: "of", Label: "Mezzanine office", Kind: "building"},
		},
		Brief: []reports.BriefValue{
			brief("of", "det.ncc_class", "5", false),
			brief("w", "det.state", "NSW", false),
			brief("w", "hdr.work_type", "extend", false),
			brief("w", "hdr.building_class", "industrial", false),
			brief("w", "hdr.cond.planning", "da", false),
			brief("w", "hdr.cond.procurement_route", "traditional", false),
			brief("w", "hdr.scale.site_sqm", "31720", false),
			brief("w", "det.flood_hazard", "stated_false", false),
			brief("w", "det.bushfire_prone_land", "", true),
			brief("w", "det.heritage_status", "none", false),
			brief("w", "det.heritage", "stated_true", false),
			brief("w", "hdr.cond.operational_constraints", "vacant", false),
		},
		Packages: []procurement.Package{
			{ID: "p1", Kind: "services", Title: "Architect", LifecycleStatus: "planned"},
			{ID: "p2", Kind: "services", Title: "Structural", LifecycleStatus: "proposed"},
			{ID: "p3", Kind: "services", Title: "Civil", LifecycleStatus: "appointed"},
			{ID: "p4", Kind: "services", Title: "Survey", LifecycleStatus: "cancelled"},
			{ID: "p5", Kind: "works", Title: "Head contract", LifecycleStatus: "planned"},
		},
		Documents:          []reports.BriefValue{{ID: "d1"}, {ID: "d2"}},
		ConstructionBudget: "6500000",
		BudgetBasis:        "AUD excl. GST",
	}
}

func TestDriversKeepPartsAndClasses(t *testing.T) {
	sk := skeleton(t)
	d := Drivers(arthurSnapshot(), sk)
	if len(d.Parts) != 2 || d.Parts[0].Label != "Warehouse" || !slices.Equal(d.Parts[0].Classes, []string{"7b"}) || !slices.Equal(d.Parts[1].Classes, []string{"5"}) {
		t.Fatalf("parts must stay separate with their own classes: %+v", d.Parts)
	}
	if f := d.Fact("flag.mixed_use"); f.State != Known || f.Values[0] != "yes" {
		t.Fatalf("two classes make a mixed-use fact: %+v", f)
	}
}

func TestDriversReadProfileFacts(t *testing.T) {
	d := Drivers(arthurSnapshot(), skeleton(t))
	for key, want := range map[string]string{"pathway": "da", "procurement": "traditional", "work_type": "extend", "state": "NSW", "category": "industrial", "site_area": "31720", "novation": "no", "budget": "6500000"} {
		if f := d.Fact(key); f.State != Known || f.Values[0] != want {
			t.Errorf("%s: want %s, got %+v", key, want, f)
		}
	}
	if d.Fact("novation").Source != "package" || d.Fact("budget").Detail != "AUD excl. GST" {
		t.Fatal("novation comes from the package and the budget keeps its tax basis")
	}
	if d.Text["address"] != "145-151 Arthur St, Homebush West NSW" || d.Documents != 2 {
		t.Fatalf("site text and document count: %+v %d", d.Text, d.Documents)
	}
}

func TestUnknownFactStaysUnknown(t *testing.T) {
	d := Drivers(arthurSnapshot(), skeleton(t))
	if f := d.Fact("flag.bushfire"); f.State != Unknown {
		t.Fatalf("an unknown bushfire fact must stay unknown, got %+v", f)
	}
	if f := d.Fact("flag.contamination"); f.State != Unknown {
		t.Fatalf("a fact the profile never mentions is unknown, not no: %+v", f)
	}
}

func TestFalseFactIsKnownNo(t *testing.T) {
	d := Drivers(arthurSnapshot(), skeleton(t))
	if f := d.Fact("flag.flood"); f.State != Known || f.Values[0] != "no" {
		t.Fatalf("stated_false is a known no: %+v", f)
	}
	if f := d.Fact("flag.live_operations"); f.State != Known || f.Values[0] != "no" {
		t.Fatalf("a vacant site is known not to be live: %+v", f)
	}
	if f := d.Fact("flag.existing_building"); f.State != Known || f.Values[0] != "yes" || f.Source != "derived" {
		t.Fatalf("an extension works to an existing building: %+v", f)
	}
}

func TestConflictingSourcesAreConflict(t *testing.T) {
	d := Drivers(arthurSnapshot(), skeleton(t))
	if f := d.Fact("flag.heritage"); f.State != Conflict || len(f.Values) != 2 {
		t.Fatalf("heritage none vs stated_true is a conflict: %+v", f)
	}
}

func TestNoPathwayIsUnknownNotDA(t *testing.T) {
	s := arthurSnapshot()
	s.Brief = s.Brief[:0]
	d := Drivers(s, skeleton(t))
	if f := d.Fact("pathway"); f.State != Unknown {
		t.Fatalf("no pathway means unknown, never a DA default: %+v", f)
	}
	if len(d.Parts) != 2 || len(d.Parts[1].Classes) != 0 {
		t.Fatalf("a part with no class keeps an empty class list: %+v", d.Parts)
	}
}

func TestConsultantsCarryLifecycle(t *testing.T) {
	d := Drivers(arthurSnapshot(), skeleton(t))
	want := []Consultant{{PackageID: "p3", Title: "Civil", Status: "appointed"}, {PackageID: "p2", Title: "Structural", Status: "to be appointed"}}
	if !slices.Equal(d.Consultants, want) {
		t.Fatalf("consultants %+v", d.Consultants)
	}
}

func TestWholePartStandsInWhenNoParts(t *testing.T) {
	s := arthurSnapshot()
	s.Parts = []reports.PartInfo{{ID: "w", Label: "Whole project", Kind: "whole", NCCClass: "5, 7b"}}
	d := Drivers(s, skeleton(t))
	if len(d.Parts) != 1 || !slices.Equal(d.Parts[0].Classes, []string{"5", "7b"}) {
		t.Fatalf("whole part with two classes: %+v", d.Parts)
	}
}

package procurement

import (
	"sitewise/internal/knowledge"
	"testing"
)

func TestComplexityPackagesStayDraftAndRejectUnmappedValues(t *testing.T) {
	cat, err := knowledge.Load("../../knowledge")
	if err != nil {
		t.Fatal(err)
	}
	facts := map[string]string{"heritage_status": "local_item", "bal": "BAL-40", "planning": "da", "environmental_sensitivity": "protected_habitat",
		"contamination_level": "significant", "flood_exposure": "floodway"}
	got := SuggestedPackages(cat, "industrial", []string{"new"}, facts, nil)
	if len(got) != 12 {
		t.Fatalf("baseline/complexity dedup: %+v", got)
	}
	for _, p := range got {
		if !p.Draft || p.LifecycleStatus != "proposed" || p.ReviewStatus != "proposed" {
			t.Fatal(p)
		}
		if p.Title == "Environmental" {
			t.Fatal("unmapped field activated")
		}
		if p.Title == "Town Planner" && p.MatchedFields["planning"] != "da" {
			t.Fatal("lost matching reason")
		}
	}
	got = SuggestedPackages(cat, "", nil, facts, []Package{{Kind: "services", Title: " heritage "}})
	if len(got) != 3 {
		t.Fatalf("known facts need no inferred class; active package dedup: %+v", got)
	}
	if got := SuggestedPackages(cat, "", nil, map[string]string{"heritage_status": "local_heritage_item", "bal": "bal_40", "planning": "legacy_da", "contamination_level": "minor"}, nil); len(got) != 0 {
		t.Fatalf("legacy values activated: %+v", got)
	}
}

func TestBaselinePackagesStayProposedAndDeduplicate(t *testing.T) {
	cat, err := knowledge.Load("../../knowledge")
	if err != nil {
		t.Fatal(err)
	}
	got := BaselinePackages(cat, "industrial", []string{"new", "extend"}, nil)
	if len(got) != 9 {
		t.Fatalf("baseline count %d", len(got))
	}
	for _, p := range got {
		if !p.Draft || p.ReviewStatus != "proposed" || p.LifecycleStatus != "proposed" || len(p.WorkTypes) != 2 || p.KnowledgeVersion != cat.Version() {
			t.Fatal(p)
		}
	}
	got = BaselinePackages(cat, "industrial", []string{"new"}, []Package{{Kind: "services", Title: " architect "}})
	if len(got) != 8 {
		t.Fatalf("existing package not deduplicated: %d", len(got))
	}
	if len(BaselinePackages(cat, "industrial", []string{"advisory"}, nil)) != 0 || len(BaselinePackages(cat, "", []string{"new"}, nil)) != 0 {
		t.Fatal("invented baseline for missing/unsupported inputs")
	}
}

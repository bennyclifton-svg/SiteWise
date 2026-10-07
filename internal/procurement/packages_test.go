package procurement

import (
	"sitewise/internal/knowledge"
	"testing"
)

func TestPackageAndStageBoundaries(t *testing.T) {
	cat, err := knowledge.Load("../../knowledge")
	if err != nil {
		t.Fatal(err)
	}
	p := Package{Kind: "services", Title: "Design", LifecycleStatus: "proposed", Novation: true}
	if err := ValidatePackage(p); err != nil {
		t.Fatal(err)
	}
	for _, bad := range []Package{
		{Kind: "other", Title: "X", LifecycleStatus: "planned"},
		{Kind: "works", Title: "X", LifecycleStatus: "planned"},
		{Kind: "supply", WorksScope: "trade", Title: "X", LifecycleStatus: "planned"},
		{Kind: "supply", Title: "X", Novation: true, LifecycleStatus: "planned"},
		{Kind: "services", Title: " ", LifecycleStatus: "planned"},
		{Kind: "services", Title: "X", LifecycleStatus: "invented"},
	} {
		if ValidatePackage(bad) == nil {
			t.Fatalf("accepted %+v", bad)
		}
	}
	s := Stage{StageID: "concept_design", Label: "Concept", NovationPhase: "pre"}
	if err := ValidateStage(s, p, cat); err != nil {
		t.Fatal(err)
	}
	p.Novation = false
	if ValidateStage(s, p, cat) == nil {
		t.Fatal("pre phase without novation")
	}
	s.NovationPhase = "none"
	s.StageID = "unknown"
	if ValidateStage(s, p, cat) == nil {
		t.Fatal("unknown stage")
	}
}

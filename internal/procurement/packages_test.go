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
	p := Package{Kind: "services", Title: "Design", LifecycleStatus: "proposed", Novation: "option"}
	if err := ValidatePackage(p); err != nil {
		t.Fatal(err)
	}
	for _, bad := range []Package{
		{Kind: "other", Title: "X", LifecycleStatus: "planned"},
		{Kind: "works", Title: "X", LifecycleStatus: "planned"},
		{Kind: "supply", WorksScope: "trade", Title: "X", LifecycleStatus: "planned"},
		{Kind: "supply", Title: "X", Novation: "yes", LifecycleStatus: "planned"},
		{Kind: "services", Title: "X", Novation: "maybe", LifecycleStatus: "planned"},
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
	p.Novation = "no"
	if ValidateStage(s, p, cat) == nil {
		t.Fatal("pre phase without novation")
	}
	s.NovationPhase = "none"
	s.StageID = "unknown"
	if ValidateStage(s, p, cat) == nil {
		t.Fatal("unknown stage")
	}
}

// Design & Construct does not mean novated: the client elects later.
func TestDefaultNovationIsAnOptionUnderDesignAndConstruct(t *testing.T) {
	for route, want := range map[string]string{"": NovationOption, "design_construct": NovationOption, "traditional": NovationNo} {
		if got := DefaultNovation(route); got != want {
			t.Errorf("%q: %s, want %s", route, got, want)
		}
	}
	p := Package{Kind: "services", Title: "Design", LifecycleStatus: "planned"}
	if p.NovationState() != NovationNo || p.Novated() {
		t.Fatal("unset novation is not no")
	}
	p.Novation = NovationOption
	if !p.Novated() {
		t.Fatal("an option has no stages either side of novation")
	}
}

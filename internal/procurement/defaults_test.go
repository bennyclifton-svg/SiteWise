package procurement

import (
	"sitewise/internal/knowledge"
	"slices"
	"testing"
	"time"
)

func loadCatalog(t *testing.T) *knowledge.Catalog {
	t.Helper()
	cat, err := knowledge.Load("../../knowledge")
	if err != nil {
		t.Fatal(err)
	}
	return cat
}

func titles(got []PackageSuggestion) []string {
	out := []string{}
	for _, p := range got {
		out = append(out, p.Title)
	}
	return out
}

func TestComplexityPackagesStayDraftAndRejectUnmappedValues(t *testing.T) {
	cat := loadCatalog(t)
	facts := map[string]string{"heritage_status": "local_item", "bal": "BAL-40", "planning": "da", "environmental_sensitivity": "protected_habitat",
		"contamination_level": "significant", "flood_exposure": "floodway"}
	got := SuggestedPackages(cat, SuggestionInputs{Classes: []string{"industrial"}, WorkTypes: []string{"new"}, Facts: facts})
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
	for _, want := range []string{"Heritage", "Bushfire", "Ecology"} {
		if !slices.Contains(titles(got), want) {
			t.Fatalf("complexity addition %s missing: %v", want, titles(got))
		}
	}
	got = SuggestedPackages(cat, SuggestionInputs{Facts: facts, Existing: []Package{{Kind: "services", Title: " heritage "}}})
	if !slices.Equal(titles(got), []string{"Bushfire", "Ecology", "Town Planner"}) {
		t.Fatalf("known facts need no inferred class; active package dedup: %v", titles(got))
	}
	if got := SuggestedPackages(cat, SuggestionInputs{Facts: map[string]string{"heritage_status": "local_heritage_item", "bal": "bal_40", "planning": "legacy_da", "contamination_level": "minor"}}); len(got) != 0 {
		t.Fatalf("legacy values activated: %+v", got)
	}
}

// A fit-out with the building class unread still lists the consultants
// every fit-out needs, each service on its own row.
func TestWorkTypeAloneSuggestsEachService(t *testing.T) {
	got := SuggestedPackages(loadCatalog(t), SuggestionInputs{WorkTypes: []string{"refurb"}})
	if !slices.Equal(titles(got), []string{"Architect", "Certifier", "Electrical", "Hydraulic"}) {
		t.Fatalf("refurb, no class: %v", titles(got))
	}
	for _, p := range got {
		if !p.Draft || p.ReviewStatus != "proposed" || p.LifecycleStatus != "proposed" || p.Key != "default:services:"+p.Title {
			t.Fatal(p)
		}
	}
}

func TestClassesAddTopUpsWithoutDuplicates(t *testing.T) {
	cat := loadCatalog(t)
	got := titles(SuggestedPackages(cat, SuggestionInputs{Classes: []string{"commercial"}, WorkTypes: []string{"refurb"}}))
	if !slices.Equal(got, []string{"Access", "Architect", "BCA", "Certifier", "Electrical", "Fire Services", "Hydraulic", "Interior Design", "Mechanical"}) {
		t.Fatalf("commercial fit-out: %v", got)
	}
	got = titles(SuggestedPackages(cat, SuggestionInputs{Classes: []string{"commercial", "residential"}, WorkTypes: []string{"new", "extend"}}))
	if len(slices.Compact(slices.Clone(got))) != len(got) || !slices.Contains(got, "Traffic") || !slices.Contains(got, "Architect") || slices.Contains(got, "Services Engineer") {
		t.Fatalf("two classes: %v", got)
	}
	got = titles(SuggestedPackages(cat, SuggestionInputs{Classes: []string{"infrastructure"}, WorkTypes: []string{"new"}}))
	if slices.Contains(got, "Architect") || !slices.Contains(got, "Environmental") {
		t.Fatalf("infrastructure keeps its own list: %v", got)
	}
	if got := SuggestedPackages(cat, SuggestionInputs{Classes: []string{"commercial"}, WorkTypes: []string{"advisory"}}); len(got) != 0 {
		t.Fatalf("invented a list for advisory: %v", titles(got))
	}
}

func TestScopeAddsSpecialists(t *testing.T) {
	got := SuggestedPackages(loadCatalog(t), SuggestionInputs{Scope: []string{"vertical-transport.passenger-lifts", "fire-active.sprinklers", "fire-active.smoke-alarms"}})
	if !slices.Equal(titles(got), []string{"Fire Services", "Vertical Transport"}) {
		t.Fatalf("scope additions: %v", titles(got))
	}
	if got[0].MatchedFields["scope"] != "fire-active.sprinklers" || got[1].MatchedFields["scope"] != "vertical-transport.passenger-lifts" {
		t.Fatalf("scope reason lost: %+v", got)
	}
	if got := SuggestedPackages(loadCatalog(t), SuggestionInputs{}); len(got) != 0 {
		t.Fatalf("nothing known, something suggested: %v", titles(got))
	}
}

// The PM corrects the list by removing lines: a removed package or a
// dismissed suggestion is not proposed again.
func TestRemovedAndDismissedStayGone(t *testing.T) {
	retired := time.Now()
	got := titles(SuggestedPackages(loadCatalog(t), SuggestionInputs{WorkTypes: []string{"refurb"},
		Existing:  []Package{{Kind: "services", Title: " electrical ", RetiredAt: &retired}},
		Dismissed: map[string]bool{"default:services:Certifier": true}}))
	if !slices.Equal(got, []string{"Architect", "Hydraulic"}) {
		t.Fatalf("removed or dismissed came back: %v", got)
	}
}

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
	// contamination_level is not a determinant: it must not activate Environmental.
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

// F41: the environmental consultant keyed on contamination_level, which is
// neither a determinant nor a condition, so it never fired. It is keyed on
// det.potentially_contaminated_land, which the Profile stores as stated_true
// or stated_false (ST13-contaminated-land read it stated_true).
func TestPotentiallyContaminatedLandSuggestsEnvironmental(t *testing.T) {
	cat := loadCatalog(t)
	got := SuggestedPackages(cat, SuggestionInputs{Facts: map[string]string{"potentially_contaminated_land": "stated_true"}})
	if !slices.Equal(titles(got), []string{"Environmental"}) {
		t.Fatalf("stated_true: %v", titles(got))
	}
	if got[0].MatchedFields["potentially_contaminated_land"] != "true" {
		t.Fatalf("matching reason: %+v", got[0].MatchedFields)
	}
	for _, value := range []string{"stated_false", "not_stated", ""} {
		if got := SuggestedPackages(cat, SuggestionInputs{Facts: map[string]string{"potentially_contaminated_land": value}}); len(got) != 0 {
			t.Fatalf("%q suggested: %v", value, titles(got))
		}
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
	if !slices.Equal(got, []string{"Access", "Architect", "BCA", "Certifier", "Electrical", "Fire Services", "Hydraulic", "Mechanical"}) {
		t.Fatalf("commercial refurb, scope unread: %v", got)
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

// F50: ST06-pump-tank-only (a sprinkler pump and tank upgrade, no fit-out)
// was offered an Interior Design package because the commercial refurb
// baseline added it unconditionally. It now needs interiors work in scope.
func TestRefurbInteriorDesignNeedsInteriorsScope(t *testing.T) {
	cat := loadCatalog(t)
	services := []string{"fire-active.sprinklers", "fire-active.fire-water", "hydraulic.cold-water", "electrical.switchboards"}
	for _, class := range []string{"commercial", "institution", "mixed"} {
		got := SuggestedPackages(cat, SuggestionInputs{Classes: []string{class}, WorkTypes: []string{"refurb"}, Scope: services})
		if slices.Contains(titles(got), "Interior Design") {
			t.Fatalf("%s services-only upgrade: %v", class, titles(got))
		}
		if !slices.Contains(titles(got), "Fire Services") {
			t.Fatalf("%s services upgrade lost its services consultant: %v", class, titles(got))
		}
	}
	got := SuggestedPackages(cat, SuggestionInputs{Classes: []string{"commercial"}, WorkTypes: []string{"refurb"},
		Scope: append(slices.Clone(services), "interiors.ceilings")})
	i := slices.Index(titles(got), "Interior Design")
	if i < 0 {
		t.Fatalf("fit-out with interiors scope: %v", titles(got))
	}
	if got[i].MatchedFields["scope"] != "interiors.ceilings" {
		t.Fatalf("scope reason lost: %+v", got[i].MatchedFields)
	}
	if got := SuggestedPackages(cat, SuggestionInputs{Classes: []string{"residential"}, WorkTypes: []string{"refurb"}, Scope: []string{"interiors.ceilings"}}); slices.Contains(titles(got), "Interior Design") {
		t.Fatalf("scope must not widen the class list: %v", titles(got))
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

// Every building has exits and stairs, so only accessible-design scope
// brings an Access consultant to a class (residential) with no Access baseline.
func TestAccessibleScopeSuggestsAccessConsultant(t *testing.T) {
	cat := loadCatalog(t)
	in := SuggestionInputs{Classes: []string{"residential"}, WorkTypes: []string{"new"},
		Scope: []string{"access-egress.exits", "access-egress.stairs-and-ramps", "access-egress.barriers", "access-egress.egress-doors"}}
	if slices.Contains(titles(SuggestedPackages(cat, in)), "Access") {
		t.Fatal("exits and stairs alone suggested an Access consultant")
	}
	for _, system := range []string{"access-egress.accessible-access", "access-egress.accessible-facilities", "access-egress.livable-housing"} {
		got := SuggestedPackages(cat, SuggestionInputs{Classes: []string{"residential"}, WorkTypes: []string{"new"}, Scope: []string{system}})
		i := slices.Index(titles(got), "Access")
		if i < 0 || got[i].MatchedFields["scope"] != system {
			t.Fatalf("%s: %v", system, titles(got))
		}
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

// A facade is in scope on every building, so only a complex-facade reading
// brings a Facade engineer.
func TestComplexFacadeSuggestsFacadeEngineer(t *testing.T) {
	cat := loadCatalog(t)
	in := SuggestionInputs{Classes: []string{"commercial"}, WorkTypes: []string{"new"}, Scope: []string{"envelope.external-walls"}}
	if slices.Contains(titles(SuggestedPackages(cat, in)), "Facade") {
		t.Fatal("facade scope alone suggested a Facade engineer")
	}
	in.Facts = map[string]string{"facade_type": "standard"}
	if slices.Contains(titles(SuggestedPackages(cat, in)), "Facade") {
		t.Fatal("standard facade suggested a Facade engineer")
	}
	in.Facts = map[string]string{"facade_type": "complex"}
	got := SuggestedPackages(cat, in)
	i := slices.Index(titles(got), "Facade")
	if i < 0 || got[i].MatchedFields["facade_type"] != "complex" {
		t.Fatalf("complex facade: %v", titles(got))
	}
}

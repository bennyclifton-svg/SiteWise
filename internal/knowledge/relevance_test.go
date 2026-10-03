package knowledge_test

import "testing"

func relevant(t *testing.T, scope []string, values map[string]string) map[string]bool {
	t.Helper()
	return loadRepo(t).Relevant(scope, values).Determinants
}

func TestSprinklerPumpReplacementShowsOnlyFireRows(t *testing.T) {
	got := relevant(t, []string{"fire-active.sprinklers"}, map[string]string{"ncc_class": "7b"})
	for _, want := range []string{"storage_height", "ncc_class", "state", "ncc_edition"} {
		if !got[want] {
			t.Errorf("%s hidden for a sprinkler pump replacement", want)
		}
	}
	for _, hidden := range []string{"bal", "termite_prone_area", "site_class", "saline_soil", "wind_class", "gas_supply", "flood_hazard"} {
		if got[hidden] {
			t.Errorf("%s shown for a sprinkler pump replacement", hidden)
		}
	}
}

func TestNewHouseShowsSiteAndStructureRows(t *testing.T) {
	cat := loadRepo(t)
	scope := cat.ScopeDefaults("residential", "house", "new")
	if len(scope) == 0 {
		t.Fatal("house new build has no default scope")
	}
	got := cat.Relevant(scope, nil).Determinants
	for _, want := range []string{"site_class", "wind_class", "termite_prone_area", "bal", "water_supply_source", "sewer_connection"} {
		if !got[want] {
			t.Errorf("%s hidden for a new house", want)
		}
	}
	if got["storage_height"] {
		t.Error("storage height shown for a house without sprinklers")
	}
}

func TestFitOutShowsServicesNotGround(t *testing.T) {
	cat := loadRepo(t)
	fit, ok := cat.Preset("typical_fit_out")
	if !ok {
		t.Fatal("typical fit-out preset missing")
	}
	got := cat.Relevant(fit.Systems, nil).Determinants
	for _, hidden := range []string{"site_class", "termite_prone_area", "saline_soil", "acid_sulfate_soils_mapped", "flood_hazard"} {
		if got[hidden] {
			t.Errorf("%s shown for a fit-out", hidden)
		}
	}
	if !got["water_supply_source"] || !got["ncc_class"] {
		t.Error("fit-out lost its service and classification rows")
	}
}

func TestPredicateFalseDropsARuleAndUnknownKeepsIt(t *testing.T) {
	cat := loadRepo(t)
	scope := cat.ScopeDefaults("residential", "house", "new")
	unknown := cat.Relevant(scope, nil).Rules
	known := cat.Relevant(scope, map[string]string{"ncc_class": "1a", "rise_in_storeys": "2", "state": "NSW"}).Rules
	if len(known) >= len(unknown) {
		t.Fatalf("known class should drop rules limited to other classes: %d known vs %d unknown", len(known), len(unknown))
	}
}

func TestEmptyScopeShowsTheCoreRowsOnly(t *testing.T) {
	got := relevant(t, nil, nil)
	if !got["state"] || !got["ncc_class"] || !got["ncc_edition"] || len(got) != 3 {
		t.Fatalf("empty scope = %v", got)
	}
}

func TestScopeDefaultsFallBack(t *testing.T) {
	cat := loadRepo(t)
	if len(cat.ScopeDefaults("", "warehouse", "extend")) == 0 {
		t.Fatal("warehouse extension list missing")
	}
	if got := cat.ScopeDefaults("", "cold_storage", "new"); len(got) == 0 {
		t.Fatal("a class without a list must use its category's list")
	}
	if got := cat.ScopeDefaults("", "office", "extend"); len(got) == 0 {
		t.Fatal("an extension without a class list falls back")
	}
	for _, w := range []string{"refurb", "remediation", "advisory", ""} {
		if got := cat.ScopeDefaults("commercial", "office", w); got != nil {
			t.Fatalf("%q must start empty: %v", w, got)
		}
	}
}

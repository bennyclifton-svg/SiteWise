package httpapi_test

import (
	"net/http"
	"strings"
	"testing"
)

type detailsBody struct {
	Details []struct {
		Key, Label, Value, Band, Kind string
		Options                       []struct{ ID, Label string }
	} `json:"details"`
	Header []struct {
		Key, Value string
		Options    []struct{ ID, Label string }
	} `json:"header"`
}

func detailValue(p detailsBody, key string) (string, bool) {
	for _, d := range p.Details {
		if d.Key == key {
			return d.Value, true
		}
	}
	return "", false
}

// Project details and RFP flags have one home, Profile (owner decision D1).
func TestProfileHoldsProjectDetailsAndFlags(t *testing.T) {
	a := newApp(t, withProfile(t))
	m := a.member(t, newUUID(t))
	project := m.createProject(t, "Petersham")

	var p detailsBody
	m.getJSON(t, "/api/projects/"+project+"/profile", &p)
	for _, key := range []string{"proj.client", "proj.lga", "proj.zone", "proj.regulated_design", "proj.flag.basement", "proj.flag.groundwater"} {
		if _, ok := detailValue(p, key); !ok {
			t.Fatalf("%s missing from details", key)
		}
	}
	// A flag with a determinant home is edited there, not as proj.flag.*.
	for _, key := range []string{"proj.flag.flood", "proj.flag.heritage", "proj.flag.mixed_use"} {
		if _, ok := detailValue(p, key); ok {
			t.Fatalf("%s must not be a project detail", key)
		}
	}

	put(t, m, project, "proj.zone", map[string]any{"value": "R2"}, http.StatusOK, &p)
	if v, _ := detailValue(p, "proj.zone"); v != "R2" {
		t.Fatalf("zone = %q", v)
	}
	put(t, m, project, "proj.regulated_design", map[string]any{"value": "yes"}, http.StatusOK, nil)
	put(t, m, project, "proj.flag.basement", map[string]any{"value": "no"}, http.StatusOK, &p)
	if v, _ := detailValue(p, "proj.flag.basement"); v != "no" {
		t.Fatalf("basement = %q", v)
	}
	// The RFP's extra pathways are the user's word only; Jev is never offered them.
	put(t, m, project, "hdr.cond.planning", map[string]any{"value": "ref"}, http.StatusOK, nil)

	for key, value := range map[string]string{
		"proj.regulated_design": "maybe", "proj.flag.basement": "perhaps", "proj.flag.flood": "yes",
		"proj.flag.nonsense": "yes", "proj.nonsense": "x", "hdr.cond.planning": "rezoning",
	} {
		put(t, m, project, key, map[string]any{"value": value}, http.StatusUnprocessableEntity, nil)
	}
	other := a.member(t, newUUID(t))
	put(t, other, project, "proj.zone", map[string]any{"value": "B4"}, http.StatusNotFound, nil)
}

// The site address and lot are listed for the profile to show a reading
// beside what the user typed on the site record.
func TestProfileListsSiteDetailReadings(t *testing.T) {
	a := newApp(t, withProfile(t))
	m := a.member(t, newUUID(t))
	project := m.createProject(t, "Petersham")
	var p struct {
		Site []struct{ Key, Label, Kind string } `json:"site_details"`
	}
	m.getJSON(t, "/api/projects/"+project+"/profile", &p)
	got := map[string]string{}
	for _, d := range p.Site {
		got[d.Key] = d.Label
	}
	if got["site.address"] != "Site address" || got["site.lot"] != "Lot and DP" || len(got) != 2 {
		t.Fatalf("site details = %v", got)
	}
}

// Profile shows systems in build order and every input under one entry
// (knowledge/profile/layout.yaml); a flag with no home would vanish.
func TestProfilePlacesInputsInBuildOrder(t *testing.T) {
	a := newApp(t, withProfile(t))
	m := a.member(t, newUUID(t))
	project := m.createProject(t, "Petersham")

	var p struct {
		Details    []struct{ Key, Home string } `json:"details"`
		Systems    []struct{ ID string }        `json:"systems"`
		Compliance []struct {
			Rows []struct{ Key, Home string } `json:"rows"`
		} `json:"compliance"`
	}
	m.getJSON(t, "/api/projects/"+project+"/profile", &p)
	if len(p.Systems) < 2 || p.Systems[0].ID != "site" || p.Systems[1].ID != "substructure" {
		t.Fatalf("systems are not in build order: %v", p.Systems)
	}
	flags := 0
	for _, d := range p.Details {
		if strings.HasPrefix(d.Key, "proj.flag.") {
			flags++
			if d.Home == "" {
				t.Errorf("%s has no home", d.Key)
			}
		}
	}
	if flags == 0 {
		t.Fatal("no flags listed")
	}
	homes := map[string]string{}
	for _, g := range p.Compliance {
		for _, r := range g.Rows {
			homes[r.Key] = r.Home
		}
	}
	for key, want := range map[string]string{"det.climate_zone": "envelope", "det.ncc_class": "project", "det.water_supply_source": "hydraulic"} {
		if homes[key] != want {
			t.Errorf("%s home = %q, want %q", key, homes[key], want)
		}
	}
}

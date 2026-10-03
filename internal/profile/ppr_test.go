package profile_test

import (
	"sitewise/internal/profile"
	"testing"
)

func TestPPRScaleCandidatesKeepTheirMeaning(t *testing.T) {
	for _, text := range []string{"2700 mm must be provided from finished floor to ceiling for all bedrooms.", "Each bedroom must have 2 DGPOs.", "Bedrooms – 4 LED down lights"} {
		if profile.Harvest(text, repoCatalog(t)).Has("hdr.scale.bedrooms") {
			t.Errorf("bedroom fitting or dimension mistaken for count: %s", text)
		}
	}
	h := profile.Harvest("A new eight (8) storey residential development with two (2) levels of basement car park. The development will consist of 33 sole occupancy units.", repoCatalog(t))
	for key, want := range map[string]string{"hdr.scale.storeys": "8", "hdr.scale.units": "33"} {
		found := false
		for _, c := range h.Candidates[key] {
			if c.Norm == want {
				found = true
			}
		}
		if !found {
			t.Errorf("%s missing %s: %+v", key, want, h.Candidates[key])
		}
	}
	if len(h.Candidates["hdr.scale.car_parks"]) != 0 {
		t.Errorf("basement levels mistaken for spaces: %+v", h.Candidates["hdr.scale.car_parks"])
	}
	if !profile.Harvest("Date of determination: 31 December 2014", repoCatalog(t)).Has("fact.consent_date") {
		t.Error("missed consent date")
	}
}

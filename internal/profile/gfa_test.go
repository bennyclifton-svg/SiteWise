package profile_test

import (
	"testing"

	"sitewise/internal/profile"
)

func norms(cs []profile.Candidate) []string {
	var out []string
	for _, c := range cs {
		out = append(out, c.Value)
	}
	return out
}

// A floor area is existing, added by the works, or the building after the
// works (proposed). Code sorts each area by its own words, so the addition
// is never offered as the building's GFA (Walsh fee proposal, 2026-10-27).
func TestFloorAreasAreSortedExistingAddedProposed(t *testing.T) {
	cat := repoCatalog(t)
	h := profile.Harvest("Owners propose rear extension and first-floor addition over existing footprint (~95 m² addition GFA), internal reconfiguration of ground floor.", cat)
	if got := norms(h.Candidates["hdr.scale.added_gfa_sqm"]); len(got) != 1 || got[0] != "95 m²" {
		t.Fatalf("added GFA candidates %q", got)
	}
	for _, key := range []string{"hdr.scale.gfa_sqm", "hdr.scale.existing_gfa_sqm"} {
		if got := h.Candidates[key]; len(got) != 0 {
			t.Fatalf("%s offered the addition: %q", key, norms(got))
		}
	}
	h = profile.Harvest("Existing GFA 120 m2. Proposed GFA 215 m2 after the works.", cat)
	if got := norms(h.Candidates["hdr.scale.existing_gfa_sqm"]); len(got) != 1 || got[0] != "120 m2" {
		t.Fatalf("existing GFA candidates %q", got)
	}
	if got := norms(h.Candidates["hdr.scale.gfa_sqm"]); len(got) != 1 || got[0] != "215 m2" {
		t.Fatalf("proposed GFA candidates %q", got)
	}
	if got := h.Candidates["hdr.scale.added_gfa_sqm"]; len(got) != 0 {
		t.Fatalf("added GFA offered %q", norms(got))
	}
}

// Proposed GFA is existing plus added when no document states it; a stated
// proposed GFA stands; one unknown input leaves it unknown.
func TestProposedGFAIsExistingPlusAdded(t *testing.T) {
	cat := repoCatalog(t)
	in := input(jevFact("hdr.scale.existing_gfa_sqm", "120", "survey", 0.9), jevFact("hdr.scale.added_gfa_sqm", "95", "fee", 0.9))
	in.User = header("house", "extend")
	r := row(t, profile.Build(in, cat), whole, "hdr.scale.gfa_sqm")
	if r.Value != "215" || r.Derived == nil || r.Origin != profile.OriginCalculation {
		t.Fatalf("proposed GFA %+v", r)
	}
	only := input(jevFact("hdr.scale.added_gfa_sqm", "95", "fee", 0.9))
	only.User = header("house", "extend")
	for _, r := range profile.Build(only, cat) {
		if r.Key == "hdr.scale.gfa_sqm" && r.Value != "" {
			t.Fatalf("proposed GFA invented from the addition alone: %+v", r)
		}
	}
	stated := input(jevFact("hdr.scale.existing_gfa_sqm", "120", "survey", 0.9), jevFact("hdr.scale.added_gfa_sqm", "95", "fee", 0.9), jevFact("hdr.scale.gfa_sqm", "220", "da", 0.9))
	stated.User = header("house", "extend")
	if r := row(t, profile.Build(stated, cat), whole, "hdr.scale.gfa_sqm"); r.Value != "220" || r.Derived != nil {
		t.Fatalf("a stated proposed GFA must stand: %+v", r)
	}
}

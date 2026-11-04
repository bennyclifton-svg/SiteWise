package jobs_test

import (
	"testing"

	"sitewise/internal/jobs"
)

// Rooms and building elements a brief names route the passage to the systems
// they involve, so the leaf questions are asked even when the first-pass
// family label scored low (Walsh brief, 2026-10-27: interiors 0.22,
// hydraulic 0.04 for "new kitchen … ensuite"). Routing asks; Jev answers.
func TestRoomAndElementWordsRouteTheirSystems(t *testing.T) {
	cat := loadKnowledge(t)
	for _, tc := range []struct {
		text string
		want []string
	}{
		{"New kitchen and dining opening to north-facing courtyard. First-floor main bedroom with ensuite + walk-in robe.",
			[]string{"system.interiors.joinery", "system.interiors.wet-area-waterproofing", "system.hydraulic.sanitary", "system.mechanical.local-exhaust"}},
		{"Upgrade of bathroom/laundry.", []string{"system.interiors.tiling", "system.hydraulic.hot-water"}},
		{"Party wall tie-in with No. 40 — acoustic and fire separation", []string{"system.site.adjoining-structures", "system.fire-passive.compartments", "system.interiors.acoustic-separation", "sig:sig.party-wall-stated"}},
		{"Tie-ins to existing masonry not fully exposed until strip-out", []string{"system.structure.masonry", "system.site.demolition"}},
		{"Heritage facade works limited to repair/repoint", []string{"system.structure.masonry"}},
		{"Authority and certifier liaison (BASIX where applicable)", []string{"system.envelope.thermal-performance"}},
	} {
		call, ok := jobs.EvidenceCall(cat, jobs.Passage{Text: tc.text})
		if !ok {
			t.Fatalf("%q: no evidence call", tc.text)
		}
		for _, id := range tc.want {
			if _, ok := call.Questions[id]; !ok {
				t.Errorf("%q: %s not asked", tc.text, id)
			}
		}
	}
	if _, ok := jobs.EvidenceCall(cat, jobs.Passage{Text: "Fortnightly owner progress reporting."}); ok {
		t.Fatal("a passage naming no room or element was routed")
	}
}

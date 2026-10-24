package profile_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"sitewise/internal/profile"
)

// Replay the saved, private Jev readings through the real reconciler. This
// checks observed outcomes, not whether a prompt contains particular words.
func TestChenResidenceAnswerKey(t *testing.T) {
	_, file, _, _ := runtime.Caller(0)
	root := filepath.Join(filepath.Dir(file), "..", "..")
	body, err := os.ReadFile(filepath.Join(root, "data/eval/profile/private/chen1-facts.json"))
	if os.IsNotExist(err) {
		t.Skip("private Chen readings absent")
	}
	if err != nil {
		t.Fatal(err)
	}
	var fixture struct {
		Parts          []profile.Part
		Facts          []profile.Fact
		Expected       map[string]string
		ProgramPhrases []string
	}
	if err := json.Unmarshal(body, &fixture); err != nil {
		t.Fatal(err)
	}
	if len(fixture.Expected) < 15 || len(fixture.ProgramPhrases) < 4 || len(fixture.Facts) == 0 {
		t.Fatal("incomplete private answer key")
	}
	th, err := profile.LoadThresholds(filepath.Join(root, "data/profile/thresholds.json"))
	if err != nil {
		t.Fatal(err)
	}
	rows := profile.Build(profile.Input{Parts: fixture.Parts, Facts: fixture.Facts, Thresholds: th}, repoCatalog(t))
	got := map[string]string{}
	for _, r := range rows {
		if r.Band == "amber" || r.Band == "green" {
			got[r.Key] = r.Value
		}
	}
	for key, want := range fixture.Expected {
		if got[key] != want {
			t.Errorf("%s = %q, want %q", key, got[key], want)
		}
	}
	for _, key := range []string{"site.ground-conditions", "site.survey-boundaries", "site.landscape", "substructure.footings"} {
		if got["sys."+key+".presence"] == "not_included" {
			t.Errorf("false project exclusion: %s", key)
		}
	}
	for _, key := range []string{"sys.substructure.piles.presence", "sys.substructure.slab-on-ground.presence", "sys.electrical.lighting.provider", "sys.site.stormwater-detention.provider"} {
		if got[key] != "" {
			t.Errorf("unselected alternative or unallocated whole-system provider: %s=%s", key, got[key])
		}
	}
	for _, phrase := range fixture.ProgramPhrases {
		if !strings.Contains(got["proj.program"], phrase) {
			t.Errorf("programme lost %q", phrase)
		}
	}
}

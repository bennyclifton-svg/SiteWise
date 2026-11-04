package store

import "testing"

// The Program sizes an extension by the area the works add, other work by
// the proposed GFA, and keeps its stated assumption when the Profile states
// neither (Walsh review, 2026-10-27).
func TestProgramFloorAreaFromProfile(t *testing.T) {
	scale := map[string]string{"hdr.scale.added_gfa_sqm": "95", "hdr.scale.gfa_sqm": "215"}
	if got := profileFloorArea([]string{"extend"}, scale); got != "95" {
		t.Fatalf("extension area %q", got)
	}
	if got := profileFloorArea([]string{"new"}, scale); got != "215" {
		t.Fatalf("new build area %q", got)
	}
	if got := profileFloorArea([]string{"extend"}, map[string]string{"hdr.scale.gfa_sqm": "215"}); got != "215" {
		t.Fatalf("extension without an added area %q", got)
	}
	if got := profileFloorArea([]string{"extend"}, map[string]string{}); got != "" {
		t.Fatalf("invented area %q", got)
	}
}

package knowledge_test

import (
	"testing"

	"sitewise/internal/knowledge"
)

func TestUnitMixText(t *testing.T) {
	got := knowledge.UnitMixText(map[string]string{"units_1_bed": "14", "units_2_bed": "38", "units_3_bed": "12", "units_studio": ""})
	if got != "14 × 1-bed, 38 × 2-bed, 12 × 3-bed" {
		t.Fatalf("mix = %q", got)
	}
	if got := knowledge.UnitMixText(map[string]string{"units_studio": "3", "units_4_bed": "1"}); got != "3 × studio, 1 × 4+ bed" {
		t.Fatalf("mix = %q", got)
	}
	if got := knowledge.UnitMixText(map[string]string{"units_2_bed": "abc"}); got != "" {
		t.Fatalf("a non-number is not a count: %q", got)
	}
}

package knowledge

import (
	"strconv"
	"strings"
)

// UnitMixKeys are the scale field keys of the unit mix, smallest type first
// (docs/plans/2026-11-08-unit-mix.md).
var UnitMixKeys = []string{"units_studio", "units_1_bed", "units_2_bed", "units_3_bed", "units_4_bed"}

var unitMixNames = map[string]string{"units_studio": "studio", "units_1_bed": "1-bed", "units_2_bed": "2-bed",
	"units_3_bed": "3-bed", "units_4_bed": "4+ bed"}

// UnitMixText prints the known counts in type order ("14 × 1-bed, 38 × 2-bed"),
// keyed by scale field key. An unknown count is left out; nothing is
// totalled or guessed.
func UnitMixText(values map[string]string) string {
	var parts []string
	for _, k := range UnitMixKeys {
		if n, err := strconv.Atoi(strings.TrimSpace(values[k])); err == nil && n >= 0 {
			parts = append(parts, strconv.Itoa(n)+" × "+unitMixNames[k])
		}
	}
	return strings.Join(parts, ", ")
}

package httpapi

import (
	"strings"
	"testing"

	"sitewise/internal/knowledge"
)

func TestListedScaleKeepsHouseAndMixApart(t *testing.T) {
	f := func(keys ...string) []knowledge.ScaleField {
		var out []knowledge.ScaleField
		for _, k := range keys {
			out = append(out, knowledge.ScaleField{Key: k})
		}
		return out
	}
	all := f("bedrooms", "gla_sqm", "units_1_bed", "units_2_bed")
	keys := func(fs []knowledge.ScaleField) string {
		var ks []string
		for _, x := range fs {
			ks = append(ks, x.Key)
		}
		return strings.Join(ks, ",")
	}
	filled := func(evidence, user map[string]bool) func(string) (bool, bool) {
		return func(k string) (bool, bool) { return evidence[k] || user[k], user[k] }
	}
	docs := map[string]bool{"bedrooms": true, "units_2_bed": true, "gla_sqm": true}

	// A house: documents' mix counts stay off; Bedrooms is the house's own field.
	if got := keys(listedScale(f("bedrooms"), true, all, filled(docs, nil))); got != "bedrooms,gla_sqm" {
		t.Errorf("house = %s", got)
	}
	// Apartments: the mix is listed and stray Bedrooms evidence is not.
	if got := keys(listedScale(f("units_1_bed", "units_2_bed"), true, all, filled(docs, nil))); got != "units_1_bed,units_2_bed,gla_sqm" {
		t.Errorf("apartments = %s", got)
	}
	// Unknown type: a stated count shows, and then Bedrooms evidence does not.
	if got := keys(listedScale(nil, false, all, filled(docs, nil))); got != "gla_sqm,units_2_bed" {
		t.Errorf("unknown = %s", got)
	}
	// What the user typed always shows.
	user := map[string]bool{"bedrooms": true, "units_1_bed": true}
	if got := keys(listedScale(f("units_2_bed"), true, all, filled(nil, user))); got != "units_2_bed,bedrooms,units_1_bed" {
		t.Errorf("user = %s", got)
	}
	if got := keys(listedScale(f("bedrooms"), true, all, filled(nil, user))); got != "bedrooms,units_1_bed" {
		t.Errorf("house with typed count = %s", got)
	}
}

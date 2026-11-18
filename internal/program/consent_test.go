package program

import (
	"reflect"
	"strings"
	"testing"
)

// Consent gates (docs/plans/2026-11-17-consent-conditions.md, Task 7).
func TestConsentGatesPlacedAndLinked(t *testing.T) {
	cat := testCatalog(t, nil, "")
	in := outlineHouse("")
	before := mustGenerate(t, cat, in)
	in.ConsentGates = map[string]int{"before_cc": 14, "before_occupation": 3, "during_works": 9}
	p := mustGenerate(t, cat, in)
	gates := map[string]Row{}
	for _, r := range p.Rows {
		if g := ConsentGateOf(r.SourceKey); g != "" {
			gates[g] = r
		}
	}
	if len(gates) != 2 || gates["during_works"].SourceKey != "" {
		t.Fatalf("gates %v", gates)
	}
	cc := gates["before_cc"]
	if cc.Kind != KindHoldPoint || cc.Label != "Consent conditions before the construction certificate" || strings.ContainsAny(cc.Label, "0123456789") {
		t.Fatalf("gate row %+v", cc)
	}
	if SourceOf(cc.SourceKey) != SourceConsent {
		t.Fatal("source")
	}
	linked := map[string]string{}
	for _, l := range p.Links {
		if g := ConsentGateOf(l.PredKey); g != "" {
			linked[g] = l.SuccKey
		}
	}
	for g, target := range map[string]string{"before_cc": "fragment:fr.construction-certificate:", "before_occupation": "fragment:fr.occupation-certificate:"} {
		if !strings.HasPrefix(linked[g], target) {
			// The test library may not carry the certificate; then a note
			// says the gate sits at the stage start.
			noted := false
			for _, n := range p.Notes {
				noted = noted || n.ActivityID == ConsentGateKey(g)
			}
			if !noted {
				t.Fatalf("%s linked to %q with no note", g, linked[g])
			}
		}
	}
	// Without conditions nothing changes: the same rows and links.
	again := mustGenerate(t, cat, outlineHouse(""))
	if !reflect.DeepEqual(gateRowKeys(before), gateRowKeys(again)) || len(before.Links) != len(again.Links) {
		t.Fatal("no conditions changed the program")
	}
	in.ConsentGates = nil
	if !reflect.DeepEqual(gateRowKeys(mustGenerate(t, cat, in)), gateRowKeys(before)) {
		t.Fatal("an empty gate map added rows")
	}
	if got := ConsentGateOpenCheck("a1", cc.Label, 3).Message; got != "Consent conditions before the construction certificate released, but 3 consent conditions are still open." {
		t.Fatalf("check %q", got)
	}
}

func gateRowKeys(p Plan) []string {
	out := make([]string, len(p.Rows))
	for i, r := range p.Rows {
		out[i] = r.SourceKey
	}
	return out
}

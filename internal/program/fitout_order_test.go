package program

import (
	"fmt"
	"testing"
)

// t03Fitout mirrors the T03 office fit-out shape (testing/synthetic/types/
// T03-office-fitout/shape.yaml): a Class 5 tenancy fit-out on one floor of a
// 24-storey building, refurb, CDC, traditional, with the shape's 33 work
// items and no strip-out, so no services fragment has its strip-out anchor.
func t03Fitout() GenInput {
	items := []struct{ system, action string }{
		{"interiors.walls-linings", "new"}, {"interiors.internal-glazing", "new"}, {"interiors.ceilings", "alter"},
		{"interiors.internal-doors", "new"}, {"interiors.joinery", "new"}, {"interiors.floor-finishes", "new"},
		{"interiors.painting", "new"}, {"interiors.room-acoustics", "new"}, {"interiors.ffe", "new"},
		{"site.signage", "new"}, {"fire-passive.penetrations", "new"}, {"fire-active.sprinklers", "alter"},
		{"fire-active.detection", "alter"}, {"fire-active.occupant-warning", "alter"}, {"fire-active.emergency-lighting", "alter"},
		{"mechanical.air-conditioning", "alter"}, {"mechanical.air-distribution", "alter"}, {"mechanical.ventilation", "alter"},
		{"mechanical.local-exhaust", "new"}, {"mechanical.controls-bms", "alter"}, {"hydraulic.cold-water", "alter"},
		{"hydraulic.hot-water", "new"}, {"hydraulic.sanitary", "alter"}, {"electrical.switchboards", "alter"},
		{"electrical.cabling-wiring", "alter"}, {"electrical.lighting", "alter"}, {"comms-security.structured-cabling", "new"},
		{"comms-security.ict-rooms-pathways", "new"}, {"comms-security.access-control", "new"}, {"comms-security.audiovisual", "new"},
		{"access-egress.exits", "alter"}, {"access-egress.egress-doors", "new"}, {"access-egress.accessible-access", "alter"},
	}
	in := GenInput{
		WorkTypes:  []string{"refurb"},
		Conditions: map[string]string{"planning": "cdc", "procurement_route": "traditional"},
		Values:     map[string]string{"ncc_class": "5", "rise_in_storeys": "24", "state": "NSW"},
	}
	for i, it := range items {
		in.WorkItems = append(in.WorkItems, WorkItem{ID: fmt.Sprintf("w%02d", i), System: it.system, Action: it.action})
	}
	return in
}

// t03Key is the source key of a fragment activity on one of t03Fitout's
// work items.
func t03Key(frag string, item int, activity string) string {
	return fmt.Sprintf("fragment:%s:w%02d:%s", frag, item, activity)
}

// timesByID indexes a computed program by row id.
func timesByID(t *testing.T, p Plan) map[string]Times {
	t.Helper()
	out := Compute(p.Input(Calendar{}))
	if out.Cycle != nil {
		t.Fatalf("generated a loop: %v", out.Cycle)
	}
	m := map[string]Times{}
	for _, a := range out.Activities {
		if a.Unknown {
			t.Fatalf("undated row %s", a.ID)
		}
		m[a.ID] = a
	}
	return m
}

// F44 (T03 REVIEW findings 1 to 3): on a fit-out with no strip-out, services
// rough-in comes before the ceilings, fire stopping follows the rough-in it
// seals and its hold point gates the ceilings only, and the sprinkler
// pipework is pressure tested before the ceilings close.
func TestFitoutServicesRoughInBeforeCeilings(t *testing.T) {
	cat := testCatalog(t, nil, "")
	p := mustGenerate(t, cat, t03Fitout())
	times := timesByID(t, p)

	ceilings := t03Key("fr.ceilings-fitout", 2, "ceilings")
	roughIns := []string{
		t03Key("fr.electrical-fitout", 24, "rough-in"),
		t03Key("fr.mechanical-fitout", 15, "rough-in"),
		t03Key("fr.hydraulic-sanitary-fitout", 22, "rough-in"),
		t03Key("fr.hydraulic-cold-fitout", 20, "alter"),
		t03Key("fr.ventilation-fitout", 17, "exhaust"),
		t03Key("fr.sprinklers-fitout", 11, "pipework"),
	}
	for _, k := range append(roughIns, ceilings) {
		if _, ok := times[k]; !ok {
			t.Fatalf("missing row %s", k)
		}
	}
	for _, k := range roughIns {
		if times[k].EF > times[ceilings].ES {
			t.Errorf("%s finishes on %d, after the ceilings start on %d", p.Rows[rowIndexOf(p, k)].Label, times[k].EF, times[ceilings].ES)
		}
	}
	// Fit-off work in the ceiling follows it.
	for _, k := range []string{t03Key("fr.lighting-fitout", 25, "fit"), t03Key("fr.sprinklers-fitout", 11, "heads")} {
		if times[k].ES < times[ceilings].EF {
			t.Errorf("%s starts before the ceilings finish", k)
		}
	}
	// Rough-in does not wait on doors or the hot-water unit (library-order chaining).
	for _, pair := range [][2]string{
		{t03Key("fr.internal-doors-fitout", 3, "hang"), t03Key("fr.hydraulic-sanitary-fitout", 22, "rough-in")},
		{t03Key("fr.hydraulic-hot-fitout", 21, "unit"), t03Key("fr.electrical-fitout", 24, "rough-in")},
		{t03Key("fr.lighting-fitout", 25, "fit"), t03Key("fr.mechanical-fitout", 15, "rough-in")},
	} {
		if _, ok := hasLink(p, pair[0], pair[1]); ok {
			t.Errorf("rough-in chained behind %s", pair[0])
		}
	}
}

func TestFitoutFireStoppingFollowsRoughInAndGatesCeilings(t *testing.T) {
	cat := testCatalog(t, nil, "")
	p := mustGenerate(t, cat, t03Fitout())
	times := timesByID(t, p)

	seal := t03Key("fr.fire-stopping", 10, "seal")
	hold := "hold:if.fire-stopping-before-concealment"
	ceilings := t03Key("fr.ceilings-fitout", 2, "ceilings")
	for _, k := range []string{
		t03Key("fr.electrical-fitout", 24, "rough-in"),
		t03Key("fr.mechanical-fitout", 15, "rough-in"),
		t03Key("fr.hydraulic-sanitary-fitout", 22, "rough-in"),
		t03Key("fr.sprinklers-fitout", 11, "pipework"),
	} {
		if _, ok := hasLink(p, k, seal); !ok {
			t.Errorf("fire stopping does not follow %s", k)
		}
	}
	if times[seal].ES == 0 {
		t.Error("fire stopping starts on construction day 0")
	}
	if _, ok := hasLink(p, seal, hold); !ok {
		t.Error("hold point does not follow fire stopping")
	}
	if _, ok := hasLink(p, hold, ceilings); !ok {
		t.Error("hold point does not gate the ceilings")
	}
	for _, k := range []string{
		t03Key("fr.partitions-fitout", 0, "frame"),
		t03Key("fr.internal-doors-fitout", 3, "hang"),
		t03Key("fr.joinery-fitout", 4, "procure"),
		t03Key("fr.joinery-fitout", 4, "install"),
		t03Key("fr.painting-fitout", 6, "paint"),
		t03Key("fr.floors-fitout", 5, "lay"),
	} {
		if _, ok := hasLink(p, hold, k); ok {
			t.Errorf("fire-stopping hold point gates %s", k)
		}
	}
}

func TestFitoutSprinklerPressureTestBeforeCeilings(t *testing.T) {
	cat := testCatalog(t, nil, "")
	p := mustGenerate(t, cat, t03Fitout())
	timesByID(t, p)

	hold := "hold:if.sprinkler-pressure-test-before-ceilings"
	pipework := t03Key("fr.sprinklers-fitout", 11, "pipework")
	heads := t03Key("fr.sprinklers-fitout", 11, "heads")
	ceilings := t03Key("fr.ceilings-fitout", 2, "ceilings")
	for _, pair := range [][2]string{{pipework, hold}, {hold, ceilings}, {ceilings, heads}} {
		if _, ok := hasLink(p, pair[0], pair[1]); !ok {
			t.Errorf("missing link %v", pair)
		}
	}
	if _, ok := hasLink(p, heads, hold); ok {
		t.Error("pressure test waits on the heads fitted after the ceilings")
	}
	for _, n := range p.Notes {
		if n.Code == CheckLinkSkipped {
			t.Errorf("link skipped: %s", n.Message)
		}
	}
}

// F47: a new building has no investigation rows, so the skeleton's
// Investigation stage is empty. It is an empty summary of zero length, not an
// unknown leaf, and every activity after it is dated.
func TestNewBuildingEveryActivityDated(t *testing.T) {
	cat := testCatalog(t, nil, "")
	in := GenInput{
		WorkTypes:  []string{"new"},
		Conditions: map[string]string{"planning": "da", "procurement_route": "design_construct"},
		Values:     map[string]string{"ncc_class": "2", "rise_in_storeys": "4", "state": "NSW"},
		WorkItems: []WorkItem{
			{ID: "slab", System: "substructure.slab-on-ground", Action: "new", Title: "Ground slab"},
			{ID: "frame", System: "structure.concrete", Action: "new", Title: "Concrete structure"},
		},
	}
	p := mustGenerate(t, cat, in)
	if p.SkeletonID != "sk.new-dc-da" {
		t.Fatalf("skeleton %s", p.SkeletonID)
	}
	inv := "skeleton:sk.new-dc-da:investigation"
	if _, ok := p.Row(inv); !ok {
		t.Fatal("the Investigation stage was dropped")
	}
	if len(rowKeys(p, inv)) != 0 {
		t.Fatalf("investigation rows on a new building: %v", rowKeys(p, inv))
	}
	times := timesByID(t, p)
	if times[inv].EF != times[inv].ES {
		t.Fatalf("empty stage has length %d", times[inv].EF-times[inv].ES)
	}
	out := Compute(p.Input(Calendar{}))
	if out.Status.Known != out.Status.Total || !out.Finish.Complete {
		t.Fatalf("status %+v finish %+v", out.Status, out.Finish)
	}
}

func rowIndexOf(p Plan, key string) int {
	for i, r := range p.Rows {
		if r.SourceKey == key {
			return i
		}
	}
	return -1
}

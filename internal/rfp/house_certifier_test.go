package rfp

import (
	"strings"
	"testing"
)

// houseDrivers is a new Class 1a house with a Class 10a garage on the given
// pathway, with every other flag left as the drivers derive it.
func houseDrivers(t *testing.T, pathway string, flags map[string]string, parts ...Part) DriverSet {
	t.Helper()
	if len(parts) == 0 {
		parts = []Part{{ID: "h", Label: "House", Classes: []string{"1a"}}, {ID: "g", Label: "Garage", Classes: []string{"10a"}}}
	}
	facts := map[string]Fact{
		"work_type": {Values: []string{"new"}, State: Known, Source: "profile"},
		"pathway":   {Values: []string{pathway}, State: Known, Source: "profile"},
		"state":     {Values: []string{"NSW"}, State: Known, Source: "profile"},
	}
	for k, v := range flags {
		facts["flag."+k] = Fact{Values: []string{v}, State: Known, Source: "profile"}
	}
	deriveFlags(facts, parts)
	for _, f := range skeleton(t).Flags {
		if _, ok := facts["flag."+f.Key]; !ok {
			facts["flag."+f.Key] = Fact{State: Unknown, Source: "none"}
		}
	}
	return DriverSet{Facts: facts, Parts: parts, Text: map[string]string{}}
}

// F11: a Class 10 outbuilding beside one other class is not a second use.
func TestClassTenDoesNotMakeMixedUse(t *testing.T) {
	cases := []struct {
		classes [][]string
		want    string
	}{
		{[][]string{{"1a"}, {"10a"}}, "no"},
		{[][]string{{"1a", "10a"}}, "no"},
		{[][]string{{"1a"}, {"10a"}, {"10b"}}, "no"},
		{[][]string{{"5"}, {"10a"}}, "no"},
		{[][]string{{"1a"}, {"5"}}, "yes"},
		{[][]string{{"1a"}, {"5"}, {"10a"}}, "yes"},
	}
	for _, c := range cases {
		parts := []Part{}
		for i, cl := range c.classes {
			parts = append(parts, Part{ID: string(rune('a' + i)), Classes: cl})
		}
		facts := map[string]Fact{}
		deriveFlags(facts, parts)
		if f := facts["flag.mixed_use"]; f.State != Known || f.Values[0] != c.want {
			t.Errorf("%v: mixed use want %s, got %+v", c.classes, c.want, f)
		}
	}
}

// F10: a house cannot proceed without a principal certifier, so its
// certifier package defaults to certification, not BCA advice. A saved
// answer still wins, and other classes keep the advisory default.
func TestHouseCertifierDefaultsToCertification(t *testing.T) {
	sk, d := skeleton(t), discipline(t, "bca-certification")
	r := Resolve(d, sk, houseDrivers(t, "da", nil), Settings{})
	// Certification scope: the preliminary assessment, the construction
	// certificate, inspections and the occupation certificate.
	for k, line := range map[string]string{"planning": "preliminary_document_assessment", "design_finalisation": "determine_each_cc", "delivery": "determine_each_oc"} {
		if s := r.Stage(k); s.State != Include || !hasLine(s, line) {
			t.Errorf("house: %s want include with %s, got %s %v (%s)", k, line, s.State, lineKeys(s), s.Why)
		}
	}
	// No BCA advice.
	for _, k := range []string{"schematic_design", "design_development"} {
		if s := r.Stage(k); s.State != Exclude {
			t.Errorf("house: %s want exclude, got %s (%s)", k, s.State, s.Why)
		}
	}
	if hasLine(r.Stage("planning"), "review_planning_drawings") || hasLine(r.Stage("delivery"), "bca_query_changed") {
		t.Errorf("house: advisory lines in a certification RFP: %v %v", lineKeys(r.Stage("planning")), lineKeys(r.Stage("delivery")))
	}
	if f := r.Facts["flag.bca_certification"]; f.Source != "derived" {
		t.Errorf("the house default is derived, not the user's word: %+v", f)
	}
	if r.HasCheck("fact:flag.bca_certification") || r.HasCheck("fact:flag.bca_advisory") {
		t.Errorf("a house does not wait on the advisory/certification choice: %s", describeChecks(r))
	}
	for _, k := range []string{"critical_stage_inspections", "basix_occupation"} {
		if p := r.Position(k); p.Value != "I" {
			t.Errorf("house: position %s want I, got %q (%s)", k, p.Value, p.Why)
		}
	}
	if !strings.Contains(r.Position("critical_stage_inspections").Label, "waterproofing") {
		t.Errorf("the critical-stage list names the wet-area inspection: %q", r.Position("critical_stage_inspections").Label)
	}

	// The client's saved answer wins over the house default.
	r = Resolve(d, sk, houseDrivers(t, "da", map[string]string{"bca_certification": "no", "bca_advisory": "yes"}), Settings{})
	if r.Stage("design_finalisation").State != Exclude || r.Stage("schematic_design").State != Include || hasLine(r.Stage("delivery"), "determine_each_oc") {
		t.Errorf("a saved advisory-only answer wins: certificate %s, schematic advice %s", r.Stage("design_finalisation").State, r.Stage("schematic_design").State)
	}

	// An office keeps the advisory default and its scope decisions.
	office := houseDrivers(t, "da", nil, Part{ID: "o", Classes: []string{"5"}})
	if f := office.Facts["flag.bca_certification"]; f.State != Unknown {
		t.Errorf("a Class 5 building keeps certification undecided: %+v", f)
	}
	r = Resolve(d, sk, office, Settings{})
	if r.Position("critical_stage_inspections").Key != "" || r.Position("basix_occupation").Key != "" {
		t.Errorf("the house inspection and BASIX rows stay out of a Class 5 RFP")
	}
}

// F9: a CDC house needs a principal certifier. The CDC determination
// replaces the CC stage; PC services, inspections and the OC stay.
func TestCertifierSupportsCDC(t *testing.T) {
	sk, d := skeleton(t), discipline(t, "bca-certification")
	r := Resolve(d, sk, houseDrivers(t, "cdc", nil), Settings{})
	if r.HasCheck("pathway_unsupported") {
		t.Fatalf("a CDC has a certifier stage bundle: %s", describeChecks(r))
	}
	for k, want := range map[string]string{"planning": Include, "design_finalisation": Exclude, "delivery": Include} {
		if s := r.Stage(k); s.State != want {
			t.Errorf("cdc: %s want %s, got %s (%s)", k, want, s.State, s.Why)
		}
	}
	if s := r.Stage("planning"); !hasLine(s, "cdc_determine") || !hasLine(s, "preliminary_document_assessment") {
		t.Errorf("cdc: planning holds the CDC determination: %v", lineKeys(s))
	}
	if p := r.Position("critical_stage_inspections"); p.Value != "I" {
		t.Errorf("cdc: critical-stage inspections stay: %q", p.Value)
	}
	// Under a DA the CDC stage does not apply.
	r = Resolve(d, sk, houseDrivers(t, "da", nil), Settings{})
	if s := r.Stage("planning"); hasLine(s, "cdc_determine") {
		t.Errorf("da: no CDC determination: %v", lineKeys(s))
	}
}

// F12: footings near a tree's structural root zone are the structural
// engineer's design (owner ruling R9); root cutting is not pruning.
func TestStructuralRootZoneFootings(t *testing.T) {
	sk, d := skeleton(t), discipline(t, "structural")
	for _, c := range []struct {
		flags map[string]string
		want  string
	}{
		{map[string]string{"arborist_tree_impacts": "yes"}, "I"},
		{map[string]string{"arborist_offsite_trees": "yes", "arborist_tree_impacts": "no"}, "I"},
		{map[string]string{"arborist_offsite_trees": "no", "arborist_tree_impacts": "no"}, "absent"},
		// Unknown is priced (an option shows as included by stage), never
		// silently left out.
		{map[string]string{"arborist_offsite_trees": "no"}, "I"},
	} {
		r := Resolve(d, sk, houseDrivers(t, "da", c.flags), Settings{})
		p := r.Position("root_zone_footings")
		got := p.Value
		if p.Key == "" {
			got = "absent"
		}
		if got != c.want {
			t.Errorf("%v: root zone footings want %s, got %s (%s)", c.flags, c.want, got, p.Why)
		}
		if c.want != "absent" && !strings.Contains(strings.ToLower(p.Label), "structural root zone") {
			t.Errorf("label names the structural root zone: %q", p.Label)
		}
	}
}

func lineKeys(s StageResult) []string {
	keys := []string{}
	for _, l := range s.Lines {
		keys = append(keys, l.Key)
	}
	return keys
}

func hasLine(s StageResult, key string) bool {
	for _, l := range s.Lines {
		if l.Key == key {
			return true
		}
	}
	return false
}

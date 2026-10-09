package rfp

import (
	"encoding/json"
	"strings"
	"testing"

	"sitewise/internal/knowledge"
)

func skeleton(t *testing.T) knowledge.RFPSkeleton {
	t.Helper()
	sk, ok := loadCatalog(t).RFPSkeleton()
	if !ok {
		t.Fatal("skeleton missing")
	}
	return sk
}

func discipline(t *testing.T, id string) knowledge.RFPDiscipline {
	t.Helper()
	d, ok := loadCatalog(t).RFP(id)
	if !ok {
		t.Fatalf("discipline %s missing", id)
	}
	return d
}

// facts builds a driver set from "key=value" pairs; flags as flag.x=yes.
func facts(t *testing.T, pairs ...string) DriverSet {
	t.Helper()
	d := DriverSet{Facts: map[string]Fact{}, Text: map[string]string{"address": "1 Test St"}, Consultants: []Consultant{}}
	for _, p := range pairs {
		k, v, _ := strings.Cut(p, "=")
		if k == "class" {
			d.Parts = append(d.Parts, Part{ID: "p" + v, Label: "Part " + v, Kind: "building", Classes: []string{v}})
			continue
		}
		d.Facts[k] = Fact{Values: []string{v}, State: Known, Source: "profile"}
	}
	deriveFlags(d.Facts, d.Parts)
	for _, f := range skeleton(t).Flags {
		if _, ok := d.Facts["flag."+f.Key]; !ok {
			d.Facts["flag."+f.Key] = Fact{State: Unknown}
		}
	}
	return d
}

func set(pairs ...string) Settings {
	s := Settings{}
	for i := 0; i < len(pairs); i += 2 {
		s[pairs[i]] = Setting{Value: json.RawMessage(pairs[i+1])}
	}
	return s
}

func TestKleeneMatch(t *testing.T) {
	r := &resolver{sk: skeleton(t), out: Resolved{Facts: map[string]Fact{
		"pathway": {Values: []string{"da"}, State: Known}, "flag.flood": {State: Unknown}, "flag.heritage": {Values: []string{"no"}, State: Known},
	}}}
	for name, c := range map[string]struct {
		when knowledge.RFPWhen
		want tri
	}{
		"empty is true":           {nil, yes},
		"known match":             {knowledge.RFPWhen{{"pathway": {"da"}}}, yes},
		"known miss":              {knowledge.RFPWhen{{"pathway": {"ssd"}}}, no},
		"unknown alone":           {knowledge.RFPWhen{{"flag": {"flood"}}}, unk},
		"and with false is false": {knowledge.RFPWhen{{"flag": {"flood"}, "pathway": {"ssd"}}}, no},
		"and with true unknown":   {knowledge.RFPWhen{{"flag": {"flood"}, "pathway": {"da"}}}, unk},
		"or true wins":            {knowledge.RFPWhen{{"flag": {"flood"}}, {"pathway": {"da"}}}, yes},
		"or unknown over false":   {knowledge.RFPWhen{{"flag": {"heritage"}}, {"flag": {"flood"}}}, unk},
		"or all false":            {knowledge.RFPWhen{{"flag": {"heritage"}}, {"pathway": {"ssd"}}}, no},
	} {
		if got := r.eval(c.when).t; got != c.want {
			t.Errorf("%s: got %d want %d", name, got, c.want)
		}
	}
}

func TestUserSettingBeatsRule(t *testing.T) {
	d := discipline(t, "architecture")
	r := Resolve(d, skeleton(t), facts(t, "class=7b", "procurement=traditional", "pathway=da"), set("scale", `"complex"`, "position:bim_coordination", `"X"`))
	if r.Scale != "complex" || r.ScaleSource != "user" {
		t.Fatalf("scale override ignored: %s %s", r.Scale, r.ScaleSource)
	}
	if p := r.Position("bim_coordination"); p.Value != "X" || p.Source != "user" {
		t.Fatalf("%+v", p)
	}
}

func TestScaleOnlyMovesServiceLevels(t *testing.T) {
	d := discipline(t, "structural")
	base := facts(t, "class=7b", "procurement=traditional", "pathway=da", "flag.basement=no", "flag.excavation_adjoining=no")
	simple := Resolve(d, skeleton(t), base, set("scale", `"simple"`))
	complex := Resolve(d, skeleton(t), base, set("scale", `"complex"`))
	if simple.Position("shared_model").Value != "O" || complex.Position("shared_model").Value != "I" {
		t.Fatal("a service level follows the scale")
	}
	if simple.Position("retention_design").Key != "" || complex.Position("retention_design").Key != "" {
		t.Fatal("shoring follows the excavation facts, not the scale")
	}
	if simple.Position("services_fixings").Value != "O" || complex.Position("services_fixings").Value != "O" {
		t.Fatal("services fixings are an option on every project")
	}
}

func TestPerFlagOverrideLeavesOtherFlagsLive(t *testing.T) {
	d := discipline(t, "civil")
	drivers := facts(t, "class=7b", "procurement=traditional", "pathway=da", "flag.public_road=no")
	r := Resolve(d, skeleton(t), drivers, set("driver:flag.flood", `"yes"`))
	if r.Position("flood_report").Value != "I" {
		t.Fatal("the flood override applies")
	}
	if r.Fact("flag.public_road").Values[0] != "no" || r.Fact("flag.public_road").Source != "profile" {
		t.Fatal("other flags keep following the profile")
	}
}

func TestSourceChangeAfterSettingIsAConflict(t *testing.T) {
	d := discipline(t, "civil")
	old := facts(t, "class=7b", "procurement=traditional", "pathway=da", "flag.flood=no")
	s := Settings{"driver:flag.flood": {Value: json.RawMessage(`"no"`), Basis: SettingBasis(old, "driver:flag.flood")}}
	if r := Resolve(d, skeleton(t), old, s); r.HasCheck("changed:driver:flag.flood") {
		t.Fatal("unchanged source raises no conflict")
	}
	now := facts(t, "class=7b", "procurement=traditional", "pathway=da", "flag.flood=yes")
	r := Resolve(d, skeleton(t), now, s)
	if !r.HasCheck("changed:driver:flag.flood") || r.Position("flood_report").Key != "" {
		t.Fatal("the setting still applies, and the source change is shown")
	}
	for _, c := range r.Blocking() {
		if c.ID == "changed:driver:flag.flood" {
			return
		}
	}
	t.Fatal("a changed source under a scope setting blocks issue")
}

func TestStaleSettingIsKeptAsCheck(t *testing.T) {
	r := Resolve(discipline(t, "architecture"), skeleton(t), facts(t, "class=7b"), set("stage:gone", `"include"`))
	if len(r.Orphans) != 1 || r.Orphans[0] != "stage:gone" || !r.HasCheck("orphan:stage:gone") {
		t.Fatalf("a setting for a removed row must stay visible: %v", r.Orphans)
	}
}

func TestWeightsMustTotal100(t *testing.T) {
	r := Resolve(discipline(t, "traffic"), skeleton(t), facts(t, "class=7b"), set("weight:price", "60"))
	if !r.HasCheck("weights") {
		t.Fatal("weights totalling 120% must block issue")
	}
	r = Resolve(discipline(t, "traffic"), skeleton(t), facts(t, "class=7b"), set("weight:price", "60", "weight:experience", "0"))
	if r.HasCheck("weights") {
		t.Fatal("weights totalling 100% are fine")
	}
}

func TestStageSettingOverridesBoundary(t *testing.T) {
	d := discipline(t, "architecture")
	r := Resolve(d, skeleton(t), facts(t, "class=7b", "procurement=traditional", "pathway=da"), set("stage:construction_support", `"include"`, "stage:concept", `"done"`))
	if s := r.Stage("construction_support"); s.State != Include || s.Source != "user" {
		t.Fatalf("%+v", s)
	}
	if r.Stage("concept").State != Done {
		t.Fatal("a stage completed by others is recorded as done")
	}
}

func TestBoundaryUnconfirmedBlocks(t *testing.T) {
	d := discipline(t, "architecture")
	r := Resolve(d, skeleton(t), facts(t, "class=7b", "procurement=design_construct", "pathway=da", "novation=no"), nil)
	if r.Boundary.Confirmed || r.Boundary.End != "dnc_tender" || !r.HasCheck("boundary") {
		t.Fatalf("D&C suggests a boundary ending at the D&C tender, unconfirmed: %+v", r.Boundary)
	}
	r = Resolve(d, skeleton(t), facts(t, "class=7b", "procurement=traditional"), set("boundary:range", `{"start":"brief_masterplan","end":"for_construction"}`))
	if !r.Boundary.Confirmed || r.HasCheck("boundary") {
		t.Fatal("a confirmed boundary clears the check")
	}
}

func TestOptionsNeverEnterTheBaseFee(t *testing.T) {
	d := discipline(t, "architecture")
	r := Resolve(d, skeleton(t), facts(t, "class=7b", "procurement=traditional", "pathway=da"), nil)
	secs, err := Sections(r, consultantTemplate(t))
	if err != nil {
		t.Fatal(err)
	}
	var fee *[][]string
	for _, s := range secs {
		for _, b := range s.Blocks {
			if b.ID == "rfp:returnables:r1" {
				fee = &b.Table.Rows
			}
		}
	}
	if fee == nil {
		t.Fatal("R1 table missing")
	}
	base := -1
	for i, row := range *fee {
		if row[0] == "Base fee total" {
			base = i
		}
		if strings.HasPrefix(row[0], "Option: ") && base < 0 {
			t.Fatalf("option %q listed above the base fee total", row[0])
		}
	}
	if base < 0 {
		t.Fatal("no base fee total")
	}
}

func consultantTemplate(t *testing.T) knowledge.ReportTemplate {
	t.Helper()
	tpl, ok := loadCatalog(t).CurrentReportTemplate("tpl.rfp-consultant")
	if !ok {
		t.Fatal("tpl.rfp-consultant missing")
	}
	return tpl
}

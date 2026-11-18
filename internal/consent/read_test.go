package consent

import (
	"encoding/json"
	"strings"
	"testing"

	"sitewise/internal/jev"
)

var testSystems = []System{{"site", "Site"}, {"envelope", "Envelope"}}

func TestCallLeavesOutTimingWhenTheHeadingSettledIt(t *testing.T) {
	settled := Condition{Number: "46", Heading: "Landscape plan", Text: "A landscape plan is to be submitted.", Stage: "before_cc", StageHeading: "PRIOR TO CC"}
	call := Call(settled, testSystems)
	if _, ok := call.Questions["timing"]; ok {
		t.Fatal("timing asked when the heading settled it")
	}
	for _, id := range []string{"party", "system", "evidence"} {
		if _, ok := call.Questions[id]; !ok {
			t.Fatalf("%s not asked", id)
		}
	}
	general := Condition{Number: "1", Text: "Development in accordance with plans.", StageHeading: "GENERAL CONDITIONS"}
	if _, ok := Call(general, testSystems).Questions["timing"]; !ok {
		t.Fatal("timing not asked under General")
	}
	if call.Priority != jev.PriorityBackground || call.QuestionVersion != QuestionVersion {
		t.Fatal("priority or version")
	}
	// Options go in a fixed order, not the map's sorted order.
	raw, _ := json.Marshal(call.Questions["party"].Criteria)
	if !strings.HasPrefix(string(raw), `{"consultant.architect":`) || !strings.Contains(string(raw), `"applicant":`) || strings.Index(string(raw), "trade.main_works") > strings.Index(string(raw), `"applicant"`) {
		t.Fatalf("party order %s", raw[:80])
	}
	sys, _ := json.Marshal(call.Questions["system"].Criteria)
	if !strings.HasPrefix(string(sys), `{"site":`) || !strings.HasSuffix(string(sys), `"none":"Not about the building: fees, contributions, hours, signs, title, insurance."}`) {
		t.Fatalf("system order %s", sys)
	}
}

func TestCallFitsTheRequestAtThreeAndAHalfThousand(t *testing.T) {
	c := Condition{Number: "12", Heading: "Long", Text: strings.Repeat("The applicant must comply with all of the following requirements. ", 400), StageHeading: "GENERAL"}
	call := Call(c, testSystems)
	state := call.State.(map[string]any)["condition"].(map[string]any)
	if n := len([]rune(state["text"].(string))); n != stateText {
		t.Fatalf("state text %d runes", n)
	}
	if _, err := jev.CheckRequestSize(call); err != nil {
		t.Fatal(err)
	}
	if Fingerprint(call) != Fingerprint(Call(c, testSystems)) {
		t.Fatal("fingerprint not stable")
	}
	c.Heading += " more"
	if Fingerprint(call) == Fingerprint(Call(c, testSystems)) {
		t.Fatal("fingerprint did not change with the state")
	}
}

func TestApplyBandsAtTheStartingFloors(t *testing.T) {
	c := Condition{Number: "1", Text: "x"}
	call := Call(c, testSystems)
	conf := func(p float64) *float64 { return &p }
	result := jev.Result{Answers: map[string]jev.Answer{
		"timing":   {Type: jev.TypeChoice, Choice: "before_occupation", Confidence: conf(0.62)},
		"party":    {Type: jev.TypeChoice, Choice: "consultant.landscape", Confidence: conf(0.41)},
		"system":   {Type: jev.TypeChoice, Choice: "not-an-option", Confidence: conf(0.9)},
		"evidence": {Type: jev.TypeNoul, Noul: 0.5},
	}}
	r := Apply(c, call, result, false)
	if f := r.Fields["timing"]; f.Value != "before_occupation" || f.Band != "amber" {
		t.Fatalf("timing %+v", f)
	}
	if f := r.Fields["party"]; f.Value != "" || f.Band != "blank" {
		t.Fatalf("party below the floor %+v", f)
	}
	if f := r.Fields["system"]; f.Value != "" {
		t.Fatalf("an option not offered was kept %+v", f)
	}
	if f := r.Fields["evidence"]; f.Value != "" || f.Band != "blank" {
		t.Fatalf("evidence between the floors %+v", f)
	}
	result.Answers["evidence"] = jev.Answer{Type: jev.TypeNoul, Noul: 0.2}
	if f := Apply(c, call, result, false).Fields["evidence"]; f.Value != "false" || f.Band != "amber" {
		t.Fatalf("evidence no %+v", f)
	}
	for id, f := range Apply(c, call, jev.Result{}, true).Fields {
		if f.Band != "grey" || f.Value != "" {
			t.Fatalf("%s after a timeout %+v", id, f)
		}
	}
	settled := Condition{Number: "2", Text: "x", Stage: "before_cc"}
	if f := Apply(settled, Call(settled, testSystems), jev.Result{}, true).Fields["timing"]; f.Band != "rule" || f.Value != "before_cc" {
		t.Fatalf("rule timing %+v", f)
	}
	for _, f := range r.Fields {
		if f.Band == "green" {
			t.Fatal("green before the key is reviewed")
		}
	}
}

func TestPartiesAreDisciplines(t *testing.T) {
	if err := CheckParties(); err != nil {
		t.Fatal(err)
	}
	if len(PartyOptions) != 24 || len(TimingOptions) != 7 {
		t.Fatalf("%d parties, %d timings", len(PartyOptions), len(TimingOptions))
	}
}

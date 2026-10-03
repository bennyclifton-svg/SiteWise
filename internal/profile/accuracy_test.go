package profile_test

import (
	"sitewise/internal/jev"
	"sitewise/internal/profile"
	"testing"
)

func TestCountsRejectDimensionsAndIdentifiers(t *testing.T) {
	cat := repoCatalog(t)
	for _, s := range []string{"4.08 Roller Shutter Doors\nOn-grade loading dock door to be min. 5.0m wide x 6.0m high.", "Tenancy 10 250A", "Tenancy 01 1,920 m2", "The roller shutter door accommodates 2 no. cargo vans."} {
		h := profile.Harvest(s, cat)
		for _, key := range []string{"hdr.scale.dock_doors", "hdr.scale.tenancies"} {
			if h.Has(key) {
				t.Errorf("%s offered count for %q", key, s)
			}
		}
	}
	for _, tc := range []struct{ text, key, want string }{{"There are 5 dock doors.", "hdr.scale.dock_doors", "5"}, {"There are 3 tenancies.", "hdr.scale.tenancies", "3"}} {
		h := profile.Harvest(tc.text, cat)
		if len(h.Candidates[tc.key]) != 1 || h.Candidates[tc.key][0].Norm != tc.want {
			t.Errorf("missing genuine count: %+v", h)
		}
	}
}

func TestGasNoneSurvivesReadingAndReconciliation(t *testing.T) {
	cat := repoCatalog(t)
	qs, cands := profile.LabelQuestions(profile.PassageInfo{Kind: "specification", Ordinal: 50}, profile.Harvest("Natural gas: Not applicable.", cat), cat)
	r := jev.Result{Answers: map[string]jev.Answer{"det.gas_supply": {Type: jev.TypeChoice, Choice: "none", Confidence: conf(.95)}, "det.gas_supply.assertion": {Type: jev.TypeChoice, Choice: "stated", Confidence: conf(.95)}}}
	readings := profile.Readings(r, qs, cands, "Natural gas: Not applicable.")
	var facts []profile.Fact
	for _, v := range readings {
		facts = append(facts, jevFact(v.QuestionID, v.Value, "doc", .95))
	}
	got := row(t, profile.Reconcile(input(facts...), cat), whole, "det.gas_supply")
	if got.Value != "none" || got.Band != "amber" {
		t.Fatalf("%+v", got)
	}
}

func TestRequiredSprinklersRemainRequired(t *testing.T) {
	in := input(jevFact("det.sprinklered", "stated_true", "doc", .95), jevFact("det.sprinklered.assertion", "required", "doc", .95))
	r := row(t, profile.Reconcile(in, repoCatalog(t)), whole, "det.sprinklered")
	if r.Value != "stated_true" || r.Assertion != "required" || r.Band != "amber" {
		t.Fatalf("requirement became existing protection: %+v", r)
	}
}

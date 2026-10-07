package profile_test

import (
	"sitewise/internal/jev"
	"sitewise/internal/profile"
	"sitewise/internal/works"
	"testing"
)

func TestSignalsProbabilityConflictAndProvenance(t *testing.T) {
	cat := repoCatalog(t)
	const key = "sig.existing-fire-water-test-results-stated"
	questions := map[string]jev.Question{"sig:" + key: {Type: jev.TypeNoul}}
	for _, tc := range []struct {
		p    float64
		want string
	}{{.99, "true"}, {.01, "false"}, {.5, ""}} {
		readings := profile.Readings(jev.Result{Answers: map[string]jev.Answer{"sig:" + key: {Type: jev.TypeNoul, Noul: tc.p}}}, questions, nil, "Survey results stated")
		if len(readings) != 1 {
			t.Fatalf("readings: %+v", readings)
		}
		f := jevFact(key, readings[0].Value, "one", *readings[0].Confidence)
		rows := profile.Build(input(f), cat)
		inputs, err := profile.ProposalInputs(rows, input(f).Parts, nil, cat)
		if err != nil {
			t.Fatal(err)
		}
		want := tc.want
		if want == "" {
			want = "unknown"
		}
		if got := inputs.Parts[whole].Signals[key]; got.State != want || len(got.Sources) == 0 {
			t.Fatalf("p=%v signal=%+v", tc.p, got)
		}
	}
	in := input(jevFact(key, "true", "one", .99), jevFact(key, "false", "two", .99))
	rows := profile.Build(in, cat)
	inputs, err := profile.ProposalInputs(rows, in.Parts, nil, cat)
	if err != nil {
		t.Fatal(err)
	}
	got := inputs.Parts[whole].Signals[key]
	if got.State != "unknown" || len(row(t, rows, whole, key).Sources) != 2 {
		t.Fatalf("conflict %+v", got)
	}
	missing, err := profile.ProposalInputs(nil, in.Parts, nil, cat)
	if err != nil {
		t.Fatal(err)
	}
	if len(missing.Parts[whole].Signals) != 0 {
		t.Fatal("missing signal invented")
	}
}

func TestSignalNeverDismissesOrReplacesDecision(t *testing.T) {
	p := works.Proposal{InputsFingerprint: "new", Reason: works.ProposalReason{Signals: []works.ProposalSignal{{ID: "sig.test", State: "true"}}}}
	if got := works.ApplyProposalDecision(p, nil); got.State != "addressed_by_evidence" {
		t.Fatal(got.State)
	}
	for _, tc := range []struct{ decision, fingerprint, want string }{{"accepted", "old", "accepted"}, {"dismissed", "new", "dismissed"}, {"dismissed", "old", "reopened"}} {
		got := works.ApplyProposalDecision(p, &works.ProposalDecision{Decision: tc.decision, InputsFingerprint: tc.fingerprint})
		if got.State != tc.want {
			t.Fatalf("%+v -> %s", tc, got.State)
		}
	}
}

func TestSplitScopeSummaryHasOneRowAndShowsMixedInclusion(t *testing.T) {
	items := []works.Item{{ID: "a", PartID: whole, SystemID: "hydraulic.gas", Action: "repair", Inclusion: "included", UserTouched: true}, {ID: "b", PartID: whole, SystemID: "hydraulic.gas", Action: "repair", Inclusion: "excluded", UserTouched: true}}
	rows := profile.ProjectWorkScope(nil, items)
	if len(rows) != 1 || rows[0].Value != "in" || rows[0].Note == "" || rows[0].UserVersion != 0 {
		t.Fatalf("summary %+v", rows)
	}
	items[1].Inclusion = "included"
	rows = profile.ProjectWorkScope(nil, items)
	if len(rows) != 1 || rows[0].Value != "in" || rows[0].Note != "" {
		t.Fatalf("included %+v", rows)
	}
}

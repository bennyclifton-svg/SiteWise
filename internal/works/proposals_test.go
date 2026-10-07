package works

import (
	"encoding/json"
	"reflect"
	"testing"
)

func TestProposalIdentitySurvivesSplit(t *testing.T) {
	reason := ProposalReason{Triggers: []ProposalTrigger{{"old", "upgrade", "fire-active.hydrants", "warehouse"}}, Determinants: []ProposalValue{{"existing_building", "true", "stated"}}, Signals: []ProposalSignal{{ID: "test", State: "unknown"}}, OtherSideExistence: "true"}
	record := map[string]string{"id": "ic.test", "label": "Test supply"}
	before, err := ProposalFingerprint(record, reason)
	if err != nil {
		t.Fatal(err)
	}
	reason.Triggers = []ProposalTrigger{{"child-b", "upgrade", "fire-active.hydrants", "warehouse"}, {"child-a", "upgrade", "fire-active.hydrants", "warehouse"}}
	reason.Signals[0].Sources = json.RawMessage(`[{"id":"new-evidence"}]`)
	after, err := ProposalFingerprint(record, reason)
	if err != nil || before != after {
		t.Fatalf("split reopened decision: %s %s %v", before, after, err)
	}
	p := ApplyProposalDecision(Proposal{InputsFingerprint: after}, &ProposalDecision{"dismissed", before})
	if p.State != "dismissed" || p.InputsChanged {
		t.Fatal(p)
	}
	reason.Triggers[0].Action = "replace"
	after, _ = ProposalFingerprint(record, reason)
	p = ApplyProposalDecision(Proposal{InputsFingerprint: after}, &ProposalDecision{"dismissed", before})
	if p.State != "reopened" || !p.InputsChanged {
		t.Fatal(p)
	}
}

func TestProposalFingerprintTracksRelevantInputs(t *testing.T) {
	base := ProposalReason{Triggers: []ProposalTrigger{{"one", "alter", "system", "part"}}, Determinants: []ProposalValue{{"year", "2000", "stated"}}, Signals: []ProposalSignal{{ID: "survey", State: "unknown"}}, OtherSideExistence: "unknown"}
	before, _ := ProposalFingerprint("record", base)
	for _, kind := range []string{"record", "value", "origin", "signal", "existence", "part", "system"} {
		t.Run(kind, func(t *testing.T) {
			b, _ := json.Marshal(base)
			var r ProposalReason
			if err := json.Unmarshal(b, &r); err != nil {
				t.Fatal(err)
			}
			record := "record"
			switch kind {
			case "record":
				record = "changed"
			case "value":
				r.Determinants[0].Value = "2020"
			case "origin":
				r.Determinants[0].Origin = "assumed"
			case "signal":
				r.Signals[0].State = "true"
			case "existence":
				r.OtherSideExistence = "true"
			case "part":
				r.Triggers[0].Part = "other"
			case "system":
				r.Triggers[0].System = "other"
			}
			after, err := ProposalFingerprint(record, r)
			if err != nil || before == after {
				t.Fatalf("did not track %s: %v", kind, err)
			}
		})
	}
}

func TestProposalRankingKeepsCriticalBeyondCap(t *testing.T) {
	items := []Proposal{{Key: "cost", Severity: "cost", Specificity: 3}, {Key: "parent", Severity: "life-safety", Specificity: 2}, {Key: "leaf", Severity: "life-safety", Specificity: 3}, {Key: "compliance", Severity: "compliance", Specificity: 1}, {Key: "critical", Severity: "programme", Critical: true}}
	RankProposals(items)
	got := []string{}
	for _, p := range VisibleProposals(items, 1) {
		got = append(got, p.Key)
	}
	if !reflect.DeepEqual(got, []string{"leaf", "parent", "critical"}) {
		t.Fatal(got)
	}
	if items[2].Key != "compliance" || items[2].Rank != 3 {
		t.Fatal(items)
	}
}

func TestProposalKeyAndAcceptedDecision(t *testing.T) {
	key, err := ProposalKey("ic.test", "if.test", "system", "part", 0)
	if err != nil || key != "ic.test|if.test|system|part|0" {
		t.Fatalf("%s %v", key, err)
	}
	if _, err := ProposalKey("ic.test|bad", "", "", "", 0); err == nil {
		t.Fatal("accepted ambiguous key")
	}
	p := ApplyProposalDecision(Proposal{InputsFingerprint: "new"}, &ProposalDecision{"accepted", "old"})
	if p.State != "accepted" || !p.InputsChanged {
		t.Fatal(p)
	}
}

func TestProposalFingerprintIgnoresInputOrdering(t *testing.T) {
	reason := ProposalReason{Triggers: []ProposalTrigger{{"a", "alter", "one", "part"}, {"b", "new", "two", "part"}}, Determinants: []ProposalValue{{"a", "1", "stated"}, {"b", "2", "user"}}, Signals: []ProposalSignal{{ID: "a", State: "true"}, {ID: "b", State: "unknown"}}}
	record := map[string]any{"criteria": map[any]any{true: "stated", false: "absent"}}
	before, err := ProposalFingerprint(record, reason)
	if err != nil {
		t.Fatal(err)
	}
	reason.Triggers[0], reason.Triggers[1] = reason.Triggers[1], reason.Triggers[0]
	reason.Determinants[0], reason.Determinants[1] = reason.Determinants[1], reason.Determinants[0]
	reason.Signals[0], reason.Signals[1] = reason.Signals[1], reason.Signals[0]
	after, err := ProposalFingerprint(record, reason)
	if err != nil || before != after {
		t.Fatalf("unstable fingerprint %s %s %v", before, after, err)
	}
}

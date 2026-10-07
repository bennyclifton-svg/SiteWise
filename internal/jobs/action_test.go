package jobs_test

import (
	"context"
	"errors"
	"sitewise/internal/jev"
	"sitewise/internal/jobs"
	"sitewise/internal/profile"
	"sitewise/internal/store"
	"strings"
	"testing"
)

type severalAsk struct {
	calls int
	asked []map[string]jev.Question
}

func (a *severalAsk) Ask(_ context.Context, call jev.Call) (jev.Result, error) {
	a.calls++
	a.asked = append(a.asked, call.Questions)
	c := .99
	r := jev.Result{Answers: map[string]jev.Answer{}}
	for id, q := range call.Questions {
		answer := jev.Answer{Type: q.Type, Confidence: &c, Choice: "not_stated"}
		if id == "sys.hydraulic.gas.action" {
			answer.Choice = "several"
		}
		if id == "sys.hydraulic.gas.presence" {
			answer.Choice = "included"
		}
		if id == "system.hydraulic.gas" {
			answer.Noul = .99
		}
		r.Answers[id] = answer
	}
	return r, nil
}

func TestSeveralActionsUseExistingCallAndNeedMapping(t *testing.T) {
	ctx := context.Background()
	s := openStore(t)
	org, project := newID(t), newID(t)
	seedOrg(t, s, org, project)
	doc := seedDoc(t, s, org, project, store.StatusFiled)
	if err := s.ReplaceSource(ctx, org, doc, store.DocumentSource{Source: []store.SourcePage{{Text: "Repair and replace gas."}}, Units: []store.SourceUnit{{Body: "Repair and replace gas."}}}); err != nil {
		t.Fatal(err)
	}
	passages, err := s.DocumentPassages(ctx, org, doc)
	if err != nil || len(passages) != 1 {
		t.Fatalf("passages %+v %v", passages, err)
	}
	if err := s.SetPassageSystems(ctx, org, passages[0].ID, []string{"hydraulic"}); err != nil {
		t.Fatal(err)
	}
	a := &severalAsk{}
	w := &jobs.Worker{Store: s, Ask: a, Catalog: loadKnowledge(t), Profile: profile.Thresholds{Amber: map[string]float64{"action": .8}}, MinNoul: .5}
	if err := w.Perform(ctx, store.ClaimedJob{OrgID: org, DocumentID: doc, Kind: store.JobKindEvidence}); err != nil {
		t.Fatal(err)
	}
	// Presence first, then detail for the found leaf only.
	if a.calls != 2 {
		t.Fatalf("want presence then detail calls, got %d", a.calls)
	}
	if _, asked := a.asked[1]["sys.hydraulic.hot-water.action"]; asked {
		t.Fatal("detail asked for an unfound leaf")
	}
	records, err := s.SourceRecords(ctx, org, project, "", "needs_mapping", 0)
	if err != nil || len(records.Records) != 1 {
		t.Fatalf("several hidden: %+v %v", records, err)
	}
	found := false
	for _, key := range records.Records[0].Unresolved {
		found = found || key == "sys.hydraulic.gas.action"
	}
	if !found {
		t.Fatal("several action not recorded as unresolved")
	}
	for id := range a.asked[1] {
		if strings.HasPrefix(id, "sys.") && strings.HasSuffix(id, ".action") && !strings.HasPrefix(id, "sys.hydraulic.") {
			t.Fatalf("unlabelled family action %s", id)
		}
	}
	// An oversized later read fails visibly without a hidden second call or
	// overwriting the document's previously committed facts.
	all := []string{}
	for _, leaf := range w.Catalog.Leaves() {
		all = append(all, leaf.ID)
	}
	if err := s.SetPassageSystems(ctx, org, passages[0].ID, all); err != nil {
		t.Fatal(err)
	}
	err = w.Perform(ctx, store.ClaimedJob{OrgID: org, DocumentID: doc, Kind: store.JobKindEvidence})
	if !errors.Is(err, jev.ErrRequestLimit) || a.calls != 2 {
		t.Fatalf("oversized call spent or hidden: calls=%d err=%v", a.calls, err)
	}
}

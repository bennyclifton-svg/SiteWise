package jobs_test

import (
	"context"
	"encoding/json"
	"sitewise/internal/jev"
	"sitewise/internal/jobs"
	"sitewise/internal/profile"
	"sitewise/internal/store"
	"testing"
)

func TestPartsChangeBothCallFingerprints(t *testing.T) {
	cat := loadKnowledge(t)
	p := jobs.Passage{Text: "The plant room has gas.", Labels: []string{"hydraulic"}, Parts: []profile.Part{{ID: "a", Label: "Plant room", Kind: "plant_area"}}}
	label := jobs.LabelCall(cat, p)
	evidence, _ := jobs.EvidenceCall(cat, p)
	p.Parts[0].Label = "Plant room east"
	label2 := jobs.LabelCall(cat, p)
	evidence2, _ := jobs.EvidenceCall(cat, p)
	for _, pair := range [][2]jev.Call{{label, label2}, {evidence, evidence2}} {
		a, _ := json.Marshal(pair[0])
		b, _ := json.Marshal(pair[1])
		if string(a) == string(b) {
			t.Fatal("renamed part retained cache fingerprint")
		}
	}
}

type locationAsk struct{ calls int }

func (a *locationAsk) Ask(_ context.Context, c jev.Call) (jev.Result, error) {
	a.calls++
	confidence := .99
	r := jev.Result{Answers: map[string]jev.Answer{}}
	for id, q := range c.Questions {
		v := jev.Answer{Type: q.Type, Choice: "not_stated", Confidence: &confidence}
		switch id {
		case "system.hydraulic":
			v.Noul = .99
		case "source.category":
			v.Choice = "requirement"
		case "source.scope":
			v.Choice = "p1"
		case "hdr.work_type":
			v.Choice = "refurb"
		case "sys.hydraulic.gas.presence":
			v.Choice = "included"
		}
		r.Answers[id] = v
	}
	return r, nil
}

func TestLocationCarriesFromLabelToEvidenceWithoutAnotherCall(t *testing.T) {
	ctx := context.Background()
	s := openStore(t)
	org, project := newID(t), newID(t)
	seedOrg(t, s, org, project)
	if _, err := s.EnsureWholePart(ctx, org, project); err != nil {
		t.Fatal(err)
	}
	part, err := s.CreatePart(ctx, org, project, "Plant room", "plant_area", "")
	if err != nil {
		t.Fatal(err)
	}
	doc := seedDoc(t, s, org, project, store.StatusFiled)
	if err := s.ReplaceSource(ctx, org, doc, store.DocumentSource{Source: []store.SourcePage{{Text: "Natural gas in the plant room."}}, Units: []store.SourceUnit{{Body: "Natural gas in the plant room."}}}); err != nil {
		t.Fatal(err)
	}
	ask := &locationAsk{}
	w := &jobs.Worker{Store: s, Ask: ask, Catalog: loadKnowledge(t), MinNoul: .5, Profile: profile.Thresholds{Amber: map[string]float64{"location.n5": .8, "presence": .6, "header": .6}}}
	for _, kind := range []string{store.JobKindLabel, store.JobKindEvidence} {
		if err := w.Perform(ctx, store.ClaimedJob{OrgID: org, DocumentID: doc, Kind: kind}); err != nil {
			t.Fatal(err)
		}
	}
	if ask.calls != 2 {
		t.Fatalf("extra model call: %d", ask.calls)
	}
	snap, err := s.ProfileInput(ctx, org, project)
	if err != nil {
		t.Fatal(err)
	}
	located := false
	for _, f := range snap.Facts {
		if f.QuestionID == "sys.hydraulic.gas.presence" {
			located = f.PartLabel == part.Label
		}
	}
	if !located {
		t.Fatal("evidence did not inherit the applied exact label")
	}
	newLabel := "Plant room east"
	if _, err := s.UpdatePart(ctx, org, project, part.ID, &newLabel, nil, nil); err != nil {
		t.Fatal(err)
	}
	if err := w.Perform(ctx, store.ClaimedJob{OrgID: org, DocumentID: doc, Kind: store.JobKindEvidence}); err != nil {
		t.Fatal(err)
	}
	snap, err = s.ProfileInput(ctx, org, project)
	if err != nil {
		t.Fatal(err)
	}
	for _, f := range snap.Facts {
		if f.QuestionID == "sys.hydraulic.gas.presence" && f.PartLabel != "" {
			t.Fatal("stale location option was rebound after rename")
		}
	}
}

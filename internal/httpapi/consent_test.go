package httpapi_test

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"
	"time"

	"sitewise/internal/delivery"
	"sitewise/internal/httpapi"
	engine "sitewise/internal/program"
	"sitewise/internal/store"
)

// A consent condition added by hand through POST /delivery
// (docs/plans/2026-11-17-consent-conditions.md, Task 1): its number and
// words are the instrument's, an edit cannot renumber it, and another org
// gets 404.
func TestConditionAddedByHandKeepsItsNumber(t *testing.T) {
	a := newApp(t, withProfile(t))
	m := a.member(t, newUUID(t))
	project := m.createProject(t, "Conditions by hand")
	base := "/api/projects/" + project + "/delivery"
	var item delivery.Item
	m.call(t, http.MethodPost, base, []byte(`{"kind":"condition","title":"Condition 46 · Landscape plan","details":{"consent_ref":"DA/2025/0733","number":"46","heading":"Landscape plan","text":"A landscape plan is to be submitted.","stage":"before_cc"}}`), http.StatusCreated, &item)
	if item.Status != "open" || delivery.Condition(item.Content).Number != "46" || item.Number != nil {
		t.Fatalf("created %+v", item)
	}
	m.call(t, http.MethodPost, base, []byte(`{"kind":"condition","title":"No number","details":{"text":"x"}}`), http.StatusUnprocessableEntity, nil)
	m.call(t, http.MethodPost, base, []byte(`{"kind":"condition","title":"Bad status","status":"answered","details":{"number":"2"}}`), http.StatusUnprocessableEntity, nil)
	path := base + "/" + item.ID
	m.call(t, http.MethodPatch, path, []byte(`{"version":1,"details":{"number":"47","text":"Reworded","stage":"before_occupation","discipline":"consultant.landscape"}}`), http.StatusOK, &item)
	d := delivery.Condition(item.Content)
	if d.Number != "46" || d.Text != "A landscape plan is to be submitted." || d.Stage != "before_occupation" || d.Discipline != "consultant.landscape" {
		t.Fatalf("edit renumbered or reworded: %+v", d)
	}
	foreign := a.member(t, newUUID(t))
	foreign.call(t, http.MethodPost, base, []byte(`{"kind":"condition","title":"Condition 1","details":{"number":"1"}}`), http.StatusNotFound, nil)
	foreign.call(t, http.MethodPatch, path, []byte(`{"version":2,"status":"satisfied"}`), http.StatusNotFound, nil)
	other := m.createProject(t, "Another project")
	m.call(t, http.MethodPatch, "/api/projects/"+other+"/delivery/"+item.ID, []byte(`{"version":2,"status":"satisfied"}`), http.StatusNotFound, nil)
}

// Task 7: a condition offers the Program one gate hold point through the
// existing review (source consent); after Adopt the gate is linked before
// the construction certificate (the generic outline has no certificate
// activity, so before construction), and releasing it with conditions open says
// so in a check.
func TestConsentGateThroughTheProgramReview(t *testing.T) {
	a := newApp(t, withProfile(t), func(o *httpapi.Options) { o.ProgramReviewDelay = 20 * time.Millisecond })
	m := a.member(t, newUUID(t))
	project := m.createProject(t, "Consent gates")
	base := "/api/projects/" + project + "/program"
	var v programView
	m.call(t, "POST", base+"/prepare", []byte(`{"surface":"program"}`), 200, &v)
	m.call(t, http.MethodPost, "/api/projects/"+project+"/delivery", []byte(`{"kind":"condition","title":"Condition 46 · Landscape plan","details":{"number":"46","text":"A landscape plan is to be submitted.","stage":"before_cc"}}`), http.StatusCreated, nil)
	var list changeList
	deadline := time.Now().Add(5 * time.Second)
	for len(list.Changes) == 0 && time.Now().Before(deadline) {
		time.Sleep(20 * time.Millisecond)
		m.getJSON(t, base+"/changes", &list)
	}
	var gate *store.ProgramChange
	for i, c := range list.Changes {
		if c.Kind == engine.ChangeAdd && c.SourceKey == "consent:before_cc" {
			gate = &list.Changes[i]
		}
	}
	if gate == nil || gate.Source != engine.SourceConsent {
		t.Fatalf("no gate offered: %+v", list.Changes)
	}
	body, _ := json.Marshal(map[string]any{"version": list.Version, "all": "adopt"})
	var adopted programView
	m.call(t, "POST", base+"/changes/decide", body, 200, &adopted)
	m.getJSON(t, base, &adopted)
	hold := programRow(t, adopted, "Consent conditions before the construction certificate")
	if hold.Kind != engine.KindHoldPoint {
		t.Fatalf("gate row %+v", hold)
	}
	linked := false
	for _, l := range adopted.Links {
		for _, act := range adopted.Activities {
			if l.PredID == hold.ID && l.SuccID == act.ID && (strings.HasPrefix(string(act.SourceKey), "fragment:fr.construction-certificate:") || string(act.SourceKey) == "fragment:fr.construction:project:works") {
				linked = true
			}
		}
	}
	if !linked {
		for _, c := range list.Changes {
			t.Logf("change %s %s %s %s", c.Kind, c.SourceKey, c.Summary, c.Ops)
		}
		for _, l := range adopted.Links {
			if l.PredID == hold.ID || l.SuccID == hold.ID {
				t.Logf("link %s -> %s", l.PredID, l.SuccID)
			}
		}
		for _, c := range adopted.Checks {
			t.Logf("check %+v", c)
		}
		t.Logf("hold %s", hold.ID)
		t.Fatal("the gate is not linked before the construction certificate")
	}
	rel, _ := json.Marshal(map[string]any{"version": adopted.Program.Version, "id": hold.ID})
	var released programView
	m.call(t, "POST", base+"/hold-points/release", rel, 200, &released)
	var after programView
	m.getJSON(t, base, &after)
	found := false
	for _, c := range after.Checks {
		found = found || c.Code == engine.CheckConsentGateOpen && c.ActivityID == hold.ID
	}
	if !found {
		t.Fatalf("no consent_gate_open check: %+v", after.Checks)
	}
	if a.jevHits.Load() != 0 {
		t.Fatal("the Program called Jev")
	}
}

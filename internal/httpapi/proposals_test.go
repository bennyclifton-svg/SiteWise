package httpapi_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/url"
	"strings"
	"testing"

	"sitewise/internal/delivery"
	"sitewise/internal/httpapi"
	"sitewise/internal/procurement"
	"sitewise/internal/profile"
	"sitewise/internal/store"
	"sitewise/internal/works"
)

func TestProposalDecisionAPI(t *testing.T) {
	var build store.ProfileBuild
	a := newApp(t, withProfile(t), func(o *httpapi.Options) {
		o.ProposalShowCount = 1
		build = store.ProfileBuild{Catalog: o.Knowledge, KnowledgeVersion: o.Knowledge.Version(), Compute: func(s store.ProfileSnapshot) []profile.Row {
			return profile.Build(profile.Input{Parts: s.Parts, Facts: s.Facts, User: s.User, Planning: s.Planning, Thresholds: o.ProfileThresholds, Read: o.ProfileReading}, o.Knowledge)
		}}
	})
	m := a.member(t, newUUID(t))
	project := m.createProject(t, "Proposal decisions")
	var view profileBody
	m.getJSON(t, "/api/projects/"+project+"/profile", &view)
	body, _ := json.Marshal(map[string]string{"part_id": view.Parts[0].ID, "system_id": "mechanical.air-conditioning", "action": "new", "title": "Rooftop plant"})
	m.call(t, http.MethodPost, "/api/projects/"+project+"/works", body, http.StatusCreated, nil)
	e, err := works.NewEvaluator(build.Catalog)
	if err != nil {
		t.Fatal(err)
	}
	st := a.store.WithProfile(build)
	if err := st.RebuildProposals(context.Background(), m.orgID, project, e); err != nil {
		t.Fatal(err)
	}
	ps, err := st.ReadProposals(context.Background(), m.orgID, project)
	if err != nil {
		t.Fatal(err)
	}
	var p store.ProposalView
	for _, candidate := range ps {
		if candidate.RecordID == "ic.loads-investigate-supported" && candidate.InterfaceID == "if.plant-loads-structure" {
			p = candidate
			break
		}
	}
	if p.Key == "" {
		t.Fatal("missing rooftop proposal")
	}
	var listed struct {
		Items []store.ProposalView `json:"items"`
		Total int                  `json:"total"`
	}
	listURL := "/api/projects/" + project + "/proposals"
	m.getJSON(t, listURL+"?show=all", &listed)
	if len(listed.Items) != len(ps) || listed.Total != len(ps) {
		t.Fatal("incomplete full proposal list")
	}
	m.getJSON(t, listURL, &listed)
	ranked := make([]works.Proposal, len(ps))
	for i := range ps {
		ranked[i] = ps[i].Proposal
	}
	expected := works.VisibleProposals(ranked, 1)
	if len(listed.Items) != len(expected) || listed.Total != len(ps) {
		t.Fatal("visibility cap dropped critical proposals or ignored configured count")
	}
	for i := range expected {
		if listed.Items[i].Key != expected[i].Key {
			t.Fatal("proposal ordering changed")
		}
	}
	m.call(t, http.MethodGet, listURL+"?show=invalid", nil, http.StatusBadRequest, nil)
	base := "/api/projects/" + project + "/proposals/" + url.PathEscape(p.Key)
	body, _ = json.Marshal(map[string]string{"inputs_fingerprint": p.InputsFingerprint})
	foreign := a.member(t, newUUID(t))
	foreign.call(t, http.MethodGet, listURL, nil, http.StatusNotFound, nil)
	foreign.call(t, http.MethodPost, base+"/accept", body, http.StatusNotFound, nil)
	foreign.call(t, http.MethodPost, base+"/dismiss", body, http.StatusNotFound, nil)
	for _, route := range []string{"/accept", "/dismiss"} {
		m.call(t, http.MethodPost, base+route, []byte(`{"inputs_fingerprint":"invalid"}`), http.StatusUnprocessableEntity, nil)
		m.call(t, http.MethodPost, base+route, []byte(`{"inputs_fingerprint":"`+strings.Repeat("a", 64)+`"}`), http.StatusConflict, nil)
		m.call(t, http.MethodPost, base+route, []byte(`{"unexpected":true}`), http.StatusBadRequest, nil)
		m.call(t, http.MethodPost, base+route, append(append([]byte{}, body...), []byte(` {}`)...), http.StatusBadRequest, nil)
		req, _ := http.NewRequest(http.MethodPost, a.url+base+route, strings.NewReader(string(body)))
		req.Header.Set("Origin", "https://foreign.example")
		res, err := m.client.Do(req)
		if err != nil {
			t.Fatal(err)
		}
		res.Body.Close()
		if res.StatusCode != http.StatusForbidden {
			t.Fatalf("foreign origin: %d", res.StatusCode)
		}
	}
	longRationale, _ := json.Marshal(map[string]string{"inputs_fingerprint": p.InputsFingerprint, "rationale": strings.Repeat("x", 201)})
	m.call(t, http.MethodPost, base+"/dismiss", longRationale, http.StatusUnprocessableEntity, nil)
	var d store.ProposalDecisionView
	m.call(t, http.MethodPost, base+"/dismiss", body, http.StatusOK, &d)
	if d.Decision != "dismissed" || d.Version != 1 {
		t.Fatal(d)
	}
	var first, retry works.Item
	m.call(t, http.MethodPost, base+"/accept", body, http.StatusOK, &first)
	m.call(t, http.MethodPost, base+"/accept", body, http.StatusOK, &retry)
	if first.ID == "" || retry.ID != first.ID || first.Action != "investigate" || first.ReviewStatus != "accepted_for_planning" {
		t.Fatalf("accept/retry %+v %+v", first, retry)
	}
	undoURL := base + "/decision"
	foreign.call(t, http.MethodDelete, undoURL, []byte(`{"version":2}`), http.StatusNotFound, nil)
	m.call(t, http.MethodDelete, undoURL, []byte(`{"version":1}`), http.StatusConflict, nil)
	m.call(t, http.MethodDelete, undoURL, []byte(`{"version":2}`), http.StatusNoContent, nil)
	current, err := st.ReadWorks(context.Background(), m.orgID, project)
	if err != nil {
		t.Fatal(err)
	}
	for _, item := range current {
		if item.ID == first.ID {
			t.Fatal("undo left created work active")
		}
	}
	m.getJSON(t, listURL+"?show=all", &listed)
	for _, item := range listed.Items {
		if item.Key == p.Key && (item.Decision != nil || item.State != "open") {
			t.Fatal("undo retained active decision")
		}
	}
	m.call(t, http.MethodPost, base+"/accept", body, http.StatusOK, &retry)
	if retry.ID != first.ID || retry.Version != 3 {
		t.Fatalf("reaccept did not restore stable identity: %+v", retry)
	}
	m.call(t, http.MethodDelete, undoURL, []byte(`{"version":2}`), http.StatusConflict, nil)
	m.call(t, http.MethodDelete, undoURL, []byte(`{"version":4}`), http.StatusNoContent, nil)
	// An intervening dismissal must not lose the undo history required to
	// safely revive the originally created item.
	m.call(t, http.MethodPost, base+"/dismiss", body, http.StatusOK, &d)
	m.call(t, http.MethodPost, base+"/accept", body, http.StatusOK, &retry)
	if retry.ID != first.ID || retry.Version != 5 {
		t.Fatal("history lost across dismissal", retry)
	}
	if a.jevHits.Load() != 0 {
		t.Fatal("proposal decision called Jev")
	}
	// A distinct synthetic catalogue key exercises the polymorphic API path;
	// existing work-item acceptance remains attached to its original key.
	records := build.Catalog.InterfaceConsequences()
	for i := range records {
		if records[i].ID == "ic.loads-investigate-supported" {
			records[i].ID = "ic.discipline-api-test"
			records[i].Propose.Kind = "discipline"
			records[i].Propose.Label = "Structural advice"
		}
	}
	e, err = works.NewEvaluator(build.Catalog)
	if err != nil {
		t.Fatal(err)
	}
	if err := st.RebuildProposals(context.Background(), m.orgID, project, e); err != nil {
		t.Fatal(err)
	}
	ps, err = st.ReadProposals(context.Background(), m.orgID, project)
	if err != nil {
		t.Fatal(err)
	}
	var discipline store.ProposalView
	for _, candidate := range ps {
		if candidate.RecordID == "ic.discipline-api-test" && candidate.InterfaceID == "if.plant-loads-structure" {
			discipline = candidate
			break
		}
	}
	if discipline.Key == "" {
		t.Fatal("missing discipline API fixture")
	}
	disciplineURL := "/api/projects/" + project + "/proposals/" + url.PathEscape(discipline.Key) + "/accept"
	disciplineBody, _ := json.Marshal(map[string]string{"inputs_fingerprint": discipline.InputsFingerprint})
	var pkg, again procurement.Package
	m.call(t, http.MethodPost, disciplineURL, disciplineBody, http.StatusOK, &pkg)
	m.call(t, http.MethodPost, disciplineURL, disciplineBody, http.StatusOK, &again)
	if pkg.Kind != "services" || pkg.ID == "" || again.ID != pkg.ID || len(pkg.Stages) != 4 {
		t.Fatalf("package accept/retry %+v %+v", pkg, again)
	}
	disciplineUndo := strings.TrimSuffix(disciplineURL, "/accept") + "/decision"
	m.call(t, http.MethodDelete, disciplineUndo, []byte(`{"version":1}`), http.StatusNoContent, nil)
	m.call(t, http.MethodPost, disciplineURL, disciplineBody, http.StatusOK, &again)
	if again.ID != pkg.ID || again.Version != 3 {
		t.Fatal("package undo/reaccept lost identity", again)
	}
	m.call(t, http.MethodPatch, "/api/projects/"+project+"/packages/"+pkg.ID, []byte(`{"version":3,"title":"Edited service package"}`), http.StatusOK, nil)
	m.call(t, http.MethodDelete, disciplineUndo, []byte(`{"version":3}`), http.StatusConflict, nil)
	for i := range records {
		if records[i].ID == "ic.discipline-api-test" {
			records[i].ID = "ic.obligation-api-test"
			records[i].Propose.Kind = "obligation"
			records[i].Propose.Label = "Coordinate structural review"
		}
	}
	e, err = works.NewEvaluator(build.Catalog)
	if err != nil {
		t.Fatal(err)
	}
	if err := st.RebuildProposals(context.Background(), m.orgID, project, e); err != nil {
		t.Fatal(err)
	}
	ps, err = st.ReadProposals(context.Background(), m.orgID, project)
	if err != nil {
		t.Fatal(err)
	}
	var obligation store.ProposalView
	for _, candidate := range ps {
		if candidate.RecordID == "ic.obligation-api-test" && candidate.InterfaceID == "if.plant-loads-structure" {
			obligation = candidate
			break
		}
	}
	if obligation.Key == "" {
		t.Fatal("missing obligation fixture")
	}
	obligationURL := "/api/projects/" + project + "/proposals/" + url.PathEscape(obligation.Key) + "/accept"
	obligationBody, _ := json.Marshal(map[string]string{"inputs_fingerprint": obligation.InputsFingerprint, "package_id": pkg.ID})
	var scope, scopeAgain store.ScopeItem
	m.call(t, http.MethodPost, obligationURL, obligationBody, http.StatusOK, &scope)
	m.call(t, http.MethodPost, obligationURL, obligationBody, http.StatusOK, &scopeAgain)
	if scope.ID == "" || scope.ID != scopeAgain.ID || !scope.Provisional || scope.PackageID != pkg.ID {
		t.Fatalf("scope accept/retry %+v %+v", scope, scopeAgain)
	}
	obligationUndo := strings.TrimSuffix(obligationURL, "/accept") + "/decision"
	m.call(t, http.MethodDelete, obligationUndo, []byte(`{"version":1}`), http.StatusNoContent, nil)
	m.call(t, http.MethodPost, obligationURL, obligationBody, http.StatusOK, &scopeAgain)
	if scopeAgain.ID != scope.ID || scopeAgain.Version != 3 {
		t.Fatal("scope undo/reaccept", scopeAgain)
	}
	for _, kind := range []string{"approval", "hold_point"} {
		for i := range records {
			if records[i].ID == "ic.obligation-api-test" || records[i].ID == "ic.approval-api-test" {
				records[i].ID = "ic." + kind + "-api-test"
				records[i].Propose.Kind = kind
				records[i].Propose.Label = "Review before installation"
			}
		}
		e, err = works.NewEvaluator(build.Catalog)
		if err != nil {
			t.Fatal(err)
		}
		if err := st.RebuildProposals(context.Background(), m.orgID, project, e); err != nil {
			t.Fatal(err)
		}
		ps, err = st.ReadProposals(context.Background(), m.orgID, project)
		if err != nil {
			t.Fatal(err)
		}
		var proposed store.ProposalView
		for _, candidate := range ps {
			if candidate.RecordID == "ic."+kind+"-api-test" && candidate.InterfaceID == "if.plant-loads-structure" {
				proposed = candidate
				break
			}
		}
		if proposed.Key == "" {
			t.Fatal("missing delivery proposal", kind)
		}
		endpoint := "/api/projects/" + project + "/proposals/" + url.PathEscape(proposed.Key)
		payload := map[string]string{"inputs_fingerprint": proposed.InputsFingerprint, "package_id": pkg.ID, "stage_id": pkg.Stages[0].ID}
		raw, _ := json.Marshal(payload)
		foreign.call(t, http.MethodPost, endpoint+"/accept", raw, http.StatusNotFound, nil)
		payload["stage_id"] = "bad-id"
		invalid, _ := json.Marshal(payload)
		m.call(t, http.MethodPost, endpoint+"/accept", invalid, http.StatusUnprocessableEntity, nil)
		payload["stage_id"] = pkg.Stages[0].ID
		payload["package_id"] = newUUID(t)
		invalid, _ = json.Marshal(payload)
		m.call(t, http.MethodPost, endpoint+"/accept", invalid, http.StatusNotFound, nil)
		var accepted, retry delivery.Item
		m.call(t, http.MethodPost, endpoint+"/accept", raw, http.StatusOK, &accepted)
		m.call(t, http.MethodPost, endpoint+"/accept", raw, http.StatusOK, &retry)
		wantKind, wantStatus := "approval", "not_submitted"
		if kind == "hold_point" {
			wantKind, wantStatus = "milestone", "planned"
		}
		if accepted.ID == "" || retry.ID != accepted.ID || accepted.Kind != wantKind || accepted.Status != wantStatus || accepted.PackageID != pkg.ID || accepted.StageID != pkg.Stages[0].ID || accepted.TargetDate != nil || accepted.ReviewStatus != "accepted_for_planning" {
			t.Fatalf("delivery accept %+v %+v", accepted, retry)
		}
		m.call(t, http.MethodDelete, endpoint+"/decision", []byte(`{"version":1}`), http.StatusNoContent, nil)
		m.call(t, http.MethodPost, endpoint+"/accept", raw, http.StatusOK, &retry)
		if retry.ID != accepted.ID || retry.Version != 3 {
			t.Fatalf("delivery revive %+v", retry)
		}
		m.call(t, http.MethodPatch, "/api/projects/"+project+"/delivery/"+retry.ID, []byte(`{"version":3,"title":"Human delivery correction"}`), http.StatusOK, nil)
		m.call(t, http.MethodDelete, endpoint+"/decision", []byte(`{"version":3}`), http.StatusConflict, nil)
	}
	if a.jevHits.Load() != 0 {
		t.Fatal("delivery decision called Jev")
	}
}

package httpapi_test

import (
	"net/http"
	"sitewise/internal/procurement"
	"testing"
)

func TestProcurementRowsAndCreate(t *testing.T) {
	a := newApp(t, withProfile(t))
	m := a.member(t, newUUID(t))
	project := m.createProject(t, "Procurement")
	// A new residential building makes the baselines propose Architect and
	// Structural, so the suggested-row path always runs.
	put(t, m, project, "hdr.building_class", map[string]any{"value": "residential"}, http.StatusOK, nil)
	put(t, m, project, "hdr.work_type", map[string]any{"value": "new"}, http.StatusOK, nil)
	jevBefore := a.jevHits.Load()
	rows := "/api/projects/" + project + "/procurement/rows"
	create := "/api/projects/" + project + "/procurement/create"

	var p procurement.Package
	m.call(t, http.MethodPost, rows, []byte(`{"discipline_id":"consultant.structural"}`), http.StatusCreated, &p)
	if p.Kind != "services" || p.DisciplineID != "consultant.structural" || p.Title != "Structural" || p.Origin != "user" {
		t.Fatalf("%+v", p)
	}
	m.call(t, http.MethodPost, rows, []byte(`{"discipline_id":"consultant.structural"}`), http.StatusConflict, nil)
	m.call(t, http.MethodPost, rows, []byte(`{"discipline_id":"supplier.windows_glazing"}`), http.StatusUnprocessableEntity, nil)
	m.call(t, http.MethodPost, rows, []byte(`{"discipline_id":"supplier.facade"}`), http.StatusUnprocessableEntity, nil)
	m.call(t, http.MethodPost, rows, []byte(`{"discipline_id":"consultant.nope"}`), http.StatusUnprocessableEntity, nil)
	m.call(t, http.MethodPost, rows, []byte(`{}`), http.StatusBadRequest, nil)
	var trade procurement.Package
	m.call(t, http.MethodPost, rows, []byte(`{"discipline_id":"trade.electrical"}`), http.StatusCreated, &trade)
	if trade.Kind != "works" || trade.WorksScope != "trade" {
		t.Fatalf("%+v", trade)
	}

	type out struct {
		Package procurement.Package `json:"package"`
		Report  struct {
			ID        string `json:"id"`
			Kind      string `json:"kind"`
			PackageID string `json:"package_id"`
		} `json:"report"`
	}
	var first, again, rft out
	m.call(t, http.MethodPost, create, []byte(`{"package_id":"`+p.ID+`"}`), http.StatusCreated, &first)
	if first.Report.Kind != "rfp" || first.Report.PackageID != p.ID || first.Package.ID != p.ID {
		t.Fatalf("%+v", first)
	}
	m.call(t, http.MethodPost, create, []byte(`{"package_id":"`+p.ID+`"}`), http.StatusOK, &again)
	if again.Report.ID != first.Report.ID {
		t.Fatal("second create made a second report")
	}
	m.call(t, http.MethodPost, create, []byte(`{"package_id":"`+trade.ID+`"}`), http.StatusCreated, &rft)
	if rft.Report.Kind != "rft" {
		t.Fatalf("%+v", rft)
	}

	// A suggested row: accepted and created in one request, with its discipline.
	var overview struct {
		Suggestions []procurement.PackageSuggestion `json:"suggestions"`
	}
	m.getJSON(t, "/api/projects/"+project+"/packages", &overview)
	key := ""
	for _, s := range overview.Suggestions {
		if s.Title == "Architect" {
			key = s.Key
		}
	}
	if key == "" {
		t.Fatalf("no Architect suggestion: %+v", overview.Suggestions)
	}
	var sug out
	m.call(t, http.MethodPost, create, []byte(`{"proposal_key":"`+key+`"}`), http.StatusCreated, &sug)
	if sug.Package.DisciplineID != "consultant.architect" || sug.Report.Kind != "rfp" || sug.Report.PackageID != sug.Package.ID {
		t.Fatalf("%+v", sug)
	}
	m.call(t, http.MethodPost, create, []byte(`{"proposal_key":"`+key+`"}`), http.StatusNotFound, nil)

	// Renaming the Structural row brings its suggestion back; accepting it
	// would make a second consultant.structural row, so Create refuses.
	m.call(t, http.MethodPatch, "/api/projects/"+project+"/packages/"+p.ID, []byte(`{"version":1,"title":"Structural engineering"}`), http.StatusOK, nil)
	m.getJSON(t, "/api/projects/"+project+"/packages", &overview)
	structural := ""
	for _, s := range overview.Suggestions {
		if s.Title == "Structural" {
			structural = s.Key
		}
	}
	if structural == "" {
		t.Fatalf("renamed row did not bring back the Structural suggestion: %+v", overview.Suggestions)
	}
	m.call(t, http.MethodPost, create, []byte(`{"proposal_key":"`+structural+`"}`), http.StatusConflict, nil)

	// A legacy head contract (no discipline) holds Main Works.
	m.call(t, http.MethodPost, "/api/projects/"+project+"/packages", []byte(`{"kind":"works","title":"Builder","works_scope":"head_contract"}`), http.StatusCreated, nil)
	m.call(t, http.MethodPost, rows, []byte(`{"discipline_id":"trade.main_works"}`), http.StatusConflict, nil)

	// A retired row cannot be created from.
	var civil procurement.Package
	m.call(t, http.MethodPost, rows, []byte(`{"discipline_id":"consultant.civil"}`), http.StatusCreated, &civil)
	m.call(t, http.MethodPatch, "/api/projects/"+project+"/packages/"+civil.ID, []byte(`{"version":1,"retired":true}`), http.StatusOK, nil)
	m.call(t, http.MethodPost, create, []byte(`{"package_id":"`+civil.ID+`"}`), http.StatusNotFound, nil)
	m.call(t, http.MethodPost, create, []byte(`{"package_id":"`+newUUID(t)+`"}`), http.StatusNotFound, nil)

	for _, bad := range []string{`{}`, `{"package_id":"` + p.ID + `","proposal_key":"x"}`, `{"package_id":"not-a-uuid"}`} {
		m.call(t, http.MethodPost, create, []byte(bad), http.StatusBadRequest, nil)
	}
	foreign := a.member(t, newUUID(t))
	foreign.call(t, http.MethodPost, rows, []byte(`{"discipline_id":"consultant.civil"}`), http.StatusNotFound, nil)
	foreign.call(t, http.MethodPost, create, []byte(`{"package_id":"`+p.ID+`"}`), http.StatusNotFound, nil)
	foreign.call(t, http.MethodPost, create, []byte(`{"proposal_key":"`+procurement.WorksProposalKey+`"}`), http.StatusNotFound, nil)
	if a.jevHits.Load() != jevBefore {
		t.Fatal("procurement rows called Jev")
	}
}

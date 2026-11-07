package httpapi_test

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"

	"sitewise/internal/procurement"
	"sitewise/internal/store"
)

// Combined engagements, slice 2: Create RFP on a group, edit a member's
// part through the existing setting route, read the one document.
func TestGroupRFPAPI(t *testing.T) {
	a := newApp(t, withProfile(t))
	m := a.member(t, newUUID(t))
	project := m.createProject(t, "Group RFP project")
	ids := []string{}
	for _, d := range []string{"consultant.structural", "consultant.civil"} {
		var p procurement.Package
		m.call(t, http.MethodPost, "/api/projects/"+project+"/procurement/rows", []byte(`{"discipline_id":"`+d+`"}`), http.StatusCreated, &p)
		ids = append(ids, p.ID)
	}
	body, _ := json.Marshal(map[string]any{"package_ids": ids})
	var group procurement.Package
	m.call(t, http.MethodPost, "/api/projects/"+project+"/packages/combine", body, http.StatusCreated, &group)
	var created struct {
		Report store.Report `json:"report"`
	}
	m.call(t, http.MethodPost, "/api/projects/"+project+"/procurement/create", []byte(`{"package_id":"`+group.ID+`"}`), http.StatusCreated, &created)
	if created.Report.PackageID != group.ID {
		t.Fatal("the group's RFP is on the group")
	}
	base := "/api/reports/" + created.Report.ID
	var d store.ReportDraft
	m.call(t, http.MethodPost, base+"/draft", []byte(`{"use_last_completed":true}`), http.StatusOK, &d)
	var view store.ReportView
	m.getJSON(t, base, &view)
	if view.RFPGroup == nil || len(view.RFPGroup.Members) != 2 || view.RFP == nil || view.PageFit == nil || view.PageFit.Limit != 8 {
		t.Fatalf("group view: %+v", view.RFPGroup)
	}
	member := view.RFPGroup.Members[0]
	key := ""
	for _, g := range member.Controls.Groups {
		for _, c := range g.Controls {
			if strings.HasPrefix(c.Key, "position:") && c.Value == "I" && key == "" {
				key = c.Key
			}
		}
	}
	// A member's setting addressed to its own report returns the group's
	// view when the editor names it.
	raw, _ := json.Marshal(map[string]any{"value": "X", "version": member.DraftVersion, "view": created.Report.ID})
	var after store.ReportView
	m.call(t, http.MethodPut, "/api/reports/"+member.ReportID+"/settings/"+key, raw, http.StatusOK, &after)
	if after.Report.ID != created.Report.ID || after.RFPGroup == nil || after.Draft.Version <= view.Draft.Version || after.RFPGroup.Members[0].DraftVersion <= member.DraftVersion {
		t.Fatalf("setting returned %s", after.Report.ID)
	}
	// Without it, the member's own view comes back as before.
	raw, _ = json.Marshal(map[string]any{"value": "I", "version": after.RFPGroup.Members[0].DraftVersion})
	var own store.ReportView
	m.call(t, http.MethodPut, "/api/reports/"+member.ReportID+"/settings/"+key, raw, http.StatusOK, &own)
	if own.Report.ID != member.ReportID || own.RFPGroup != nil {
		t.Fatal("a member setting without a view returns the member")
	}
	// A view that is not the member's group is ignored.
	raw, _ = json.Marshal(map[string]any{"value": "X", "version": own.Draft.Version, "view": newUUID(t)})
	m.call(t, http.MethodPut, "/api/reports/"+member.ReportID+"/settings/"+key, raw, http.StatusOK, &own)
	if own.Report.ID != member.ReportID {
		t.Fatal("only the member's own group can be returned")
	}
	// A discipline's choice cannot be stored on the group. Each member write
	// recomposed the group's draft, so read its version again.
	m.getJSON(t, base, &view)
	raw, _ = json.Marshal(map[string]any{"value": "X", "version": view.Draft.Version})
	m.call(t, http.MethodPut, base+"/settings/"+key, raw, http.StatusBadRequest, nil)
	// The member alone is not issued while grouped.
	raw, _ = json.Marshal(map[string]any{"version": own.Draft.Version, "reporting_date": "2026-11-07", "accept_stale": true, "stale_reason": "fixture"})
	m.call(t, http.MethodPost, "/api/reports/"+member.ReportID+"/issue", raw, http.StatusConflict, nil)
	// The response form asks for a sum per discipline.
	res := m.do(t, http.MethodGet, base+"/response-form", nil)
	if res.StatusCode != http.StatusOK {
		t.Fatalf("response form: %d", res.StatusCode)
	}
	res.Body.Close()
	// Another org sees nothing.
	foreign := a.member(t, newUUID(t))
	foreign.call(t, http.MethodGet, base, nil, http.StatusNotFound, nil)
	foreign.call(t, http.MethodPost, base+"/draft", []byte(`{"use_last_completed":true}`), http.StatusNotFound, nil)
	foreign.call(t, http.MethodPost, "/api/projects/"+project+"/procurement/create", []byte(`{"package_id":"`+group.ID+`"}`), http.StatusNotFound, nil)
}

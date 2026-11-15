package httpapi_test

import (
	"net/http"
	"strings"
	"testing"
)

// A finish-to-start link made through an operation shows in the plain
// Program read, which is what the command key compares.
func TestProgramLinkShowsInTheChatRead(t *testing.T) {
	a := newApp(t, withProfile(t))
	m := a.member(t, newUUID(t))
	project := m.createProject(t, "Program link")
	base := "/api/projects/" + project + "/program"
	m.call(t, http.MethodPost, base+"/prepare", []byte(`{"surface":"program"}`), http.StatusOK, nil)
	var view programView
	m.getJSON(t, base, &view)
	stage := ""
	for _, act := range view.Activities {
		if act.Kind == "stage" && stage == "" {
			stage = act.ID
		}
	}
	m.ops(t, project, "program", http.StatusOK, op("program.edit", map[string]any{"op": "add", "parent_id": stage, "kind": "activity", "labels": []string{"Dig", "Pour", "Strip"}, "duration": "1 week"}))
	m.getJSON(t, base, &view)
	id := map[string]string{}
	for _, act := range view.Activities {
		id[act.Label] = act.ID
	}
	m.ops(t, project, "program", http.StatusOK, op("program.edit", map[string]any{"op": "link_add", "pred_id": id["Dig"][:8], "succ_id": id["Strip"][:8], "type": "FS"}))
	_, plain := m.text(t, http.MethodGet, "/api/projects/"+project+"/chat/read?area=program", nil)
	if !strings.Contains(plain, "Strip | 5 d") || !strings.Contains(plain, "after Dig") {
		t.Fatalf("the link is not in the read:\n%s", plain)
	}
}

package httpapi_test

import (
	"encoding/json"
	"testing"
	"time"

	"sitewise/internal/httpapi"
	"sitewise/internal/procurement"
	engine "sitewise/internal/program"
	"sitewise/internal/store"
)

type changeList struct {
	ProgramID string                `json:"program_id"`
	Version   int64                 `json:"version"`
	Changes   []store.ProgramChange `json:"changes"`
	Counts    map[string]int        `json:"counts"`
}

// The review over HTTP: a work item change kicks the debounced review,
// which raises items without touching the program; decide adopts them as
// one batch, with 422 and 409 like the other program routes. Packages add
// no program rows (owner direction 25 October 2026), so a new package
// raises nothing.
func TestProgramReviewRoutes(t *testing.T) {
	a := newApp(t, withProfile(t), func(o *httpapi.Options) { o.ProgramReviewDelay = 20 * time.Millisecond })
	m := a.member(t, newUUID(t))
	project := m.createProject(t, "Program review")
	base := "/api/projects/" + project + "/program"
	m.call(t, "POST", "/api/projects/"+project+"/packages", []byte(`{"kind":"services","title":"Architect"}`), 201, nil)
	var v programView
	m.call(t, "POST", base+"/prepare", []byte(`{"surface":"program"}`), 200, &v)

	var list changeList
	m.getJSON(t, base+"/changes", &list)
	if len(list.Changes) != 0 || list.ProgramID != v.Program.ID || list.Version != v.Program.Version || list.Counts["add"] != 0 {
		t.Fatalf("empty review %+v", list)
	}

	var eng procurement.Package
	m.call(t, "POST", "/api/projects/"+project+"/packages", []byte(`{"kind":"services","title":"Engineer"}`), 201, &eng)
	var p profileBody
	m.getJSON(t, "/api/projects/"+project+"/profile", &p)
	work, _ := json.Marshal(map[string]any{"title": "Builders power", "part_id": p.Parts[0].ID, "system_id": "electrical.temporary-power", "action": "new"})
	m.call(t, "POST", "/api/projects/"+project+"/works", work, 201, nil)
	deadline := time.Now().Add(5 * time.Second)
	for len(list.Changes) == 0 && time.Now().Before(deadline) {
		time.Sleep(20 * time.Millisecond)
		m.getJSON(t, base+"/changes", &list)
	}
	// The outline has no trade rows (owner, 12 November 2026): new work
	// changes the Construction bar's length, sized from the trades.
	if len(list.Changes) != 1 || list.Counts["duration"] != 1 {
		t.Fatalf("raised %+v", list)
	}
	for _, c := range list.Changes {
		if c.Kind != engine.ChangeDuration || c.SourceKey != "fragment:fr.construction:project:works" ||
			c.Source != engine.SourceWorkItems || c.DefaultChoice != engine.ChoiceAdopt || len(c.Ops) == 0 {
			t.Fatalf("change %+v", c)
		}
	}
	var now programView
	m.getJSON(t, base, &now)
	if now.Program.Version != v.Program.Version || len(now.Activities) != len(v.Activities) || now.ChangesPending != len(list.Changes) {
		t.Fatalf("the review wrote the program or miscounted: v%d rows %d pending %d", now.Program.Version, len(now.Activities), now.ChangesPending)
	}

	if e := m.programError(t, "POST", base+"/changes/decide", []byte(`{"version":1,"all":"keep"}`), 422); e.Field != "all" {
		t.Fatalf("all %+v", e)
	}
	if e := m.programError(t, "POST", base+"/changes/decide", []byte(`{"version":1,"decisions":[{"id":"`+list.Changes[0].ID+`","choice":"maybe"}]}`), 422); e.Field != "choice" {
		t.Fatalf("choice %+v", e)
	}
	if e := m.programError(t, "POST", base+"/changes/decide", []byte(`{"version":1,"decisions":[]}`), 422); e.Field != "decisions" {
		t.Fatalf("empty %+v", e)
	}
	if e := m.programError(t, "POST", base+"/changes/decide", []byte(`{"version":99,"all":"adopt"}`), 409); len(e.Current) == 0 {
		t.Fatalf("conflict %+v", e)
	}
	body, _ := json.Marshal(map[string]any{"version": v.Program.Version, "all": "adopt"})
	var adopted programView
	m.call(t, "POST", base+"/changes/decide", body, 200, &adopted)
	if adopted.Program.Version != v.Program.Version+1 || len(adopted.Activities) != len(v.Activities) || adopted.ChangesPending != 0 ||
		adopted.ProgramFeedback == nil || len(adopted.Toasts) != 1 {
		t.Fatalf("adopt all v%d rows %d pending %d", adopted.Program.Version, len(adopted.Activities), adopted.ChangesPending)
	}
	var undone programView
	m.call(t, "POST", base+"/undo", []byte(`{"version":`+jsonInt(adopted.Program.Version)+`}`), 200, &undone)
	if len(undone.Activities) != len(v.Activities) {
		t.Fatalf("undo left %d rows", len(undone.Activities))
	}

	// Update program applies every difference in one click, with no review
	// to answer, and one undo reverses it.
	if e := m.programError(t, "POST", base+"/update", []byte(`{}`), 422); e.Field != "version" {
		t.Fatalf("update without a version %+v", e)
	}
	if e := m.programError(t, "POST", base+"/update", []byte(`{"version":99}`), 409); len(e.Current) == 0 {
		t.Fatalf("update conflict %+v", e)
	}
	var updated programView
	m.call(t, "POST", base+"/update", []byte(`{"version":`+jsonInt(undone.Program.Version)+`}`), 200, &updated)
	if updated.Program.Version != undone.Program.Version+1 || updated.ChangesPending != 0 || updated.ProgramFeedback == nil || len(updated.Toasts) != 1 {
		t.Fatalf("update v%d pending %d", updated.Program.Version, updated.ChangesPending)
	}
	var same programView
	m.call(t, "POST", base+"/update", []byte(`{"version":`+jsonInt(updated.Program.Version)+`}`), 200, &same)
	if same.Program.Version != updated.Program.Version {
		t.Fatalf("an up-to-date program was written: v%d", same.Program.Version)
	}
}

func jsonInt(v int64) string { b, _ := json.Marshal(v); return string(b) }

package httpapi_test

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"mime/multipart"
	"net/http"
	"sync/atomic"
	"testing"
	"time"

	"sitewise/internal/httpapi"
	"sitewise/internal/jev"
	"sitewise/internal/programread"
	"sitewise/internal/xlsx"
)

// countingJev stands in for System One on the program read: it answers
// none to every question and counts calls.
type countingJev struct{ calls atomic.Int64 }

func (c *countingJev) Ask(_ context.Context, call jev.Call) (jev.Result, error) {
	c.calls.Add(1)
	if call.Priority != jev.PriorityBackground || call.QuestionVersion != programread.QuestionVersion {
		return jev.Result{}, jev.ErrRequest
	}
	conf := 0.99
	answers := map[string]jev.Answer{}
	for id := range call.Questions {
		answers[id] = jev.Answer{Type: jev.TypeChoice, Choice: "none", Confidence: &conf}
	}
	return jev.Result{Answers: answers}, nil
}

func (m *member) importProgram(t *testing.T, project, name string, file []byte) map[string]string {
	t.Helper()
	var body bytes.Buffer
	mw := multipart.NewWriter(&body)
	part, err := mw.CreateFormFile("file", name)
	if err != nil {
		t.Fatal(err)
	}
	_, _ = part.Write(file)
	_ = mw.Close()
	req, err := http.NewRequest(http.MethodPost, m.app.url+"/api/projects/"+project+"/program/import", &body)
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Origin", m.app.url)
	req.Header.Set("Content-Type", mw.FormDataContentType())
	resp, err := m.client.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusAccepted {
		t.Fatalf("import = %d: %s", resp.StatusCode, raw)
	}
	var out map[string]string
	if err := json.Unmarshal(raw, &out); err != nil {
		t.Fatal(err)
	}
	return out
}

func TestProgramImportUploadReadAcceptUndo(t *testing.T) {
	asker := &countingJev{}
	a := newApp(t, withProfile(t), func(o *httpapi.Options) { o.ProgramJev = asker })
	m := a.member(t, newUUID(t))
	project := m.createProject(t, "Program import")
	base := "/api/projects/" + project + "/program"

	// No program: nothing to import into.
	resp := m.do(t, "POST", base+"/import?name=p.xlsx", []byte("x"))
	resp.Body.Close()
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("import with no program = %d", resp.StatusCode)
	}

	var v programView
	m.call(t, "POST", base+"/prepare", []byte(`{"surface":"program"}`), http.StatusOK, &v)
	start, _ := json.Marshal(map[string]any{"program_id": v.Program.ID, "version": v.Program.Version, "ops": []map[string]any{{"op": "set_start", "start_date": "2027-03-01"}}})
	m.call(t, "POST", base+"/ops", start, http.StatusOK, &v)
	// A leaf stage takes the file's dates; its label is matched exactly.
	var target string
	kids := map[string]bool{}
	for _, act := range v.Activities {
		kids[string(act.ParentID)] = true
	}
	var label string
	for _, act := range v.Activities {
		if act.Kind == "stage" && !kids[act.ID] {
			target, label = act.ID, act.Label
			break
		}
	}
	if target == "" {
		t.Fatal("no leaf stage")
	}
	txt := func(s string) xlsx.Cell { return xlsx.Cell{Text: s} }
	var file bytes.Buffer
	if err := xlsx.WriteXLSX(&file, []xlsx.Sheet{{Name: "Program", Rows: [][]xlsx.Cell{
		{txt("Task Name"), txt("Start"), txt("Finish")},
		{txt(label), txt("15/03/2027"), txt("26/03/2027")},
		{txt("Qq zz"), txt("29/03/2027"), txt("30/03/2027")},
	}}}); err != nil {
		t.Fatal(err)
	}
	up := m.importProgram(t, project, "Smith program.xlsx", file.Bytes())
	if up["import_id"] == "" || up["document_id"] == "" {
		t.Fatalf("%+v", up)
	}

	deadline := time.Now().Add(15 * time.Second)
	for {
		m.getJSON(t, base, &v)
		if len(v.Imports) == 1 && v.Imports[0].Status == "read" {
			break
		}
		if time.Now().After(deadline) {
			t.Fatalf("not read: %+v", v.Imports)
		}
		time.Sleep(50 * time.Millisecond)
	}
	if v.Imports[0].Text != "1 reading from Smith program.xlsx" || v.ReadingsPending != 2 || len(v.Readings) != 2 || v.Imports[0].Unmatched != 1 {
		t.Fatalf("view %+v %+v", v.Imports, v.Readings)
	}
	// The exact label settled by rule: no question was worth asking.
	if n := asker.calls.Load(); n != 0 {
		t.Fatalf("Jev asked %d times", n)
	}
	before := v.Program.Version
	accept, _ := json.Marshal(map[string]any{"version": v.Program.Version, "ids": []string{v.Readings[0].ID}})
	m.call(t, "POST", base+"/accept", accept, http.StatusOK, &v)
	row := programRow(t, v, label)
	if row.DurationDays == nil || *row.DurationDays != 10 || row.DurationOrigin != "document" || row.ConstraintKind != "snet" || row.ConstraintDate != "2027-03-15" {
		t.Fatalf("accepted %+v", row)
	}
	if v.ReadingsPending != 0 || v.Program.Version != before+1 {
		t.Fatalf("pending %d version %d", v.ReadingsPending, v.Program.Version)
	}
	undo, _ := json.Marshal(map[string]any{"version": v.Program.Version})
	m.call(t, "POST", base+"/undo", undo, http.StatusOK, &v)
	if row := programRow(t, v, label); row.ConstraintKind != "asap" || row.DurationOrigin == "document" || v.ReadingsPending != 2 {
		t.Fatalf("undo %+v pending %d", row, v.ReadingsPending)
	}

	// The unmatched row is added in one click under the stage.
	add, _ := json.Marshal(map[string]any{"version": v.Program.Version, "add_rows": []map[string]string{{"import_id": up["import_id"], "row": "r2", "parent_id": target}}})
	m.call(t, "POST", base+"/accept", add, http.StatusOK, &v)
	if row := programRow(t, v, "Qq zz"); row.Origin != "document" || row.DurationDays == nil || *row.DurationDays != 2 {
		t.Fatalf("added %+v", row)
	}

	// Dismiss the rest.
	dismiss, _ := json.Marshal(map[string]any{"version": v.Program.Version, "ids": []string{v.Readings[0].ID}})
	m.call(t, "POST", base+"/readings/dismiss", dismiss, http.StatusOK, &v)
	if v.ReadingsPending != 0 {
		t.Fatalf("pending %d", v.ReadingsPending)
	}

	// Another org sees none of it.
	other := a.member(t, newUUID(t))
	if got := other.status(t, "GET", base, nil); got != http.StatusNotFound {
		t.Fatalf("other org = %d", got)
	}
}

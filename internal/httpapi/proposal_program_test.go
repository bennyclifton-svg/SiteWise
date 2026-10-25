package httpapi_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/url"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"sitewise/internal/httpapi"
	"sitewise/internal/jev"
	"sitewise/internal/programread"
	"sitewise/internal/rfp"
	"sitewise/internal/xlsx"
)

// stageWeeksJev counts program_stage_weeks calls; a well-completed R7
// returnable must make none.
type stageWeeksJev struct{ calls atomic.Int64 }

func (s *stageWeeksJev) Ask(_ context.Context, call jev.Call) (jev.Result, error) {
	if call.QuestionVersion == programread.StageWeeksQuestionVersion {
		s.calls.Add(1)
	}
	return jev.Result{}, jev.ErrRequest
}

func TestR7ReturnableReadAfterProposalShowsOnlyOnceAwarded(t *testing.T) {
	asker := &stageWeeksJev{}
	a := newApp(t, withProfile(t), greenProposals, func(o *httpapi.Options) { o.ProgramJev = asker })
	m := a.member(t, newUUID(t))
	project := m.createProject(t, "R7 project")
	var pkg struct {
		ID    string `json:"id"`
		Title string `json:"title"`
	}
	m.call(t, http.MethodPost, "/api/projects/"+project+"/procurement/rows", []byte(`{"discipline_id":"consultant.town_planner"}`), http.StatusCreated, &pkg)
	var v programView
	m.call(t, http.MethodPost, "/api/projects/"+project+"/program/prepare", []byte(`{"surface":"program"}`), http.StatusOK, &v)
	// Packages add no program rows (owner direction 25 October 2026), and a
	// stage with activities takes its length from them, so a reading lands
	// only on a stage with none. Empty every stage, then name the package's
	// stages in the R7 sheet.
	var acts []string
	for _, act := range v.Activities {
		if act.ParentID != "" {
			acts = append(acts, act.ID)
		}
	}
	if len(acts) > 0 {
		ops, _ := json.Marshal(map[string]any{"program_id": v.Program.ID, "version": v.Program.Version, "ops": []map[string]any{{"op": "delete", "ids": acts}}})
		m.call(t, http.MethodPost, "/api/projects/"+project+"/program/ops", ops, http.StatusOK, &v)
	}
	pkgs, err := a.store.ReadPackages(context.Background(), m.orgID, project)
	if err != nil {
		t.Fatal(err)
	}
	labels := map[string]string{} // package stage id → label
	for _, p := range pkgs {
		if p.ID == pkg.ID {
			for _, s := range p.Stages {
				labels[s.ID] = s.Label
			}
		}
	}
	if len(labels) == 0 {
		t.Fatal("no package stages")
	}
	// The firm's returned response form: an exact R7 sheet, every stage a
	// plain number of weeks.
	txt := func(s string) xlsx.Cell { return xlsx.Cell{Text: s} }
	r7 := xlsx.Sheet{Name: "R7 Program", Rows: [][]xlsx.Cell{{txt("R7 Program")}, nil, {txt(rfp.R7Columns[0]), txt(rfp.R7Columns[1]), txt(rfp.R7Columns[2]), txt(rfp.R7Columns[3])}}}
	for _, label := range labels {
		r7.Rows = append(r7.Rows, []xlsx.Cell{txt(label), txt("4"), txt("2"), txt("")})
	}
	fees := xlsx.Sheet{Name: "R1 Fees", Rows: [][]xlsx.Cell{{txt("Town planning fee proposal")}, {txt("Firm"), txt("Example Planning Pty Ltd")}, {txt("ABN"), txt("88 071 762 537")}}}
	var file bytes.Buffer
	if err := xlsx.WriteXLSX(&file, []xlsx.Sheet{fees, r7}); err != nil {
		t.Fatal(err)
	}
	base := "/api/projects/" + project + "/procurement/packages/" + pkg.ID
	// The fee schedule exists before a proposal arrives, as it does once
	// the RFP is drafted.
	var schedule priceTableBody
	m.getJSON(t, base+"/prices", &schedule)
	var up struct {
		SubmissionID string `json:"submission_id"`
	}
	m.call(t, http.MethodPost, base+"/submissions?name="+url.QueryEscape("Response form (Example Planning).xlsx"), file.Bytes(), http.StatusCreated, &up)
	if up.SubmissionID == "" {
		t.Fatal("no submission")
	}
	// The proposal read commits, then the programme read runs after it.
	deadline := time.Now().Add(20 * time.Second)
	for {
		var table priceTableBody
		m.getJSON(t, base+"/prices", &table)
		pending, err := a.store.PendingProposalPrograms(context.Background())
		if err != nil {
			t.Fatal(err)
		}
		busy := false
		for _, p := range pending {
			busy = busy || p.SubmissionID == up.SubmissionID
		}
		if len(table.Columns) == 1 && table.Columns[0].Submission.Status != "reading" && !busy {
			in, err := a.store.ProposalProgramForReading(context.Background(), m.orgID, up.SubmissionID)
			if err == nil && in.Status == "read" {
				break
			}
		}
		if time.Now().After(deadline) {
			t.Fatal("proposal programme not read")
		}
		time.Sleep(50 * time.Millisecond)
	}
	if n := asker.calls.Load(); n != 0 {
		t.Fatalf("a well-completed R7 returnable asked Jev %d times", n)
	}
	proposalR7 := func() []string {
		m.getJSON(t, "/api/projects/"+project+"/program", &v)
		var out []string
		for _, r := range v.Readings {
			if r.Source == "proposal_r7" {
				out = append(out, r.Text)
			}
		}
		return out
	}
	if got := proposalR7(); len(got) != 0 || v.ReadingsPending != 0 {
		t.Fatalf("readings before award: %v", got)
	}
	// Award on the PM's own priced cell, then undo.
	var table priceTableBody
	m.getJSON(t, base+"/prices", &table)
	first := ""
	for _, it := range table.Items {
		if it.Base {
			first = it.Key
			break
		}
	}
	cell := "/api/projects/" + project + "/submissions/" + up.SubmissionID + "/cells/" + url.PathEscape(first)
	m.call(t, http.MethodPut, cell, []byte(`{"state":"priced","amount":"2,000"}`), http.StatusOK, &table)
	// A quoted proposal is awarded on the PM's confirmed appointment fee.
	appointment := "/api/projects/" + project + "/submissions/" + up.SubmissionID + "/cells/" + url.PathEscape("comparison:appointment")
	m.call(t, http.MethodPut, appointment, []byte(`{"state":"priced","amount":"2000","note":"Confirmed against the proposal"}`), http.StatusOK, &table)
	award := `{"submission_id":"` + up.SubmissionID + `","lines":[{"label":"Fee","item_keys":["comparison:appointment"]}],"total":"2000.00"}`
	m.call(t, http.MethodPost, base+"/award", []byte(award), http.StatusCreated, &table)
	got := proposalR7()
	if len(got) != len(labels) || v.ReadingsPending != len(labels) {
		t.Fatalf("readings after award: %v", got)
	}
	for _, text := range got {
		if !strings.HasPrefix(text, "From the awarded proposal: ") || !strings.HasSuffix(text, " 4 weeks") {
			t.Fatalf("reading text %q", text)
		}
	}
	m.call(t, http.MethodDelete, base+"/award", nil, http.StatusOK, &table)
	if got := proposalR7(); len(got) != 0 {
		t.Fatalf("readings after undo: %v", got)
	}
}

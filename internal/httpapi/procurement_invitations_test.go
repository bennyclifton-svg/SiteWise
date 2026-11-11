package httpapi_test

import (
	"net/http"
	"net/url"
	"strings"
	"testing"
	"time"

	psample "sitewise/internal/proposals/sample"
)

type boardBody struct {
	Packages map[string][]struct {
		SubmissionID string `json:"submission_id"`
		FirmID       string `json:"firm_id"`
		FirmName     string `json:"firm_name"`
		Status       string `json:"status"`
		InvitedOn    string `json:"invited_on"`
	} `json:"packages"`
}

// The PM names a firm on a row before its quote arrives; the quote dropped
// on that firm's cell is read as that firm's. Every route is the org's own
// (docs/plans/2026-11-11-procurement-invited-firms.md).
func TestInvitedFirmOnARowThenItsQuote(t *testing.T) {
	a := newApp(t, withProfile(t), greenProposals)
	m, other := a.member(t, newUUID(t)), a.member(t, newUUID(t))
	project := m.createProject(t, "Invited firms")
	var pkg struct {
		ID string `json:"id"`
	}
	m.call(t, http.MethodPost, "/api/projects/"+project+"/procurement/rows", []byte(`{"discipline_id":"consultant.town_planner"}`), http.StatusCreated, &pkg)
	base := "/api/projects/" + project + "/procurement/packages/" + pkg.ID
	var inv struct {
		PackageID string `json:"package_id"`
		FirmID    string `json:"firm_id"`
		FirmName  string `json:"firm_name"`
	}
	m.call(t, http.MethodPost, base+"/invitations", []byte(`{"name":"Chosen Planning"}`), http.StatusCreated, &inv)
	if inv.FirmID == "" || inv.FirmName != "Chosen Planning" || inv.PackageID != pkg.ID {
		t.Fatalf("invitation %+v", inv)
	}
	m.call(t, http.MethodPost, base+"/invitations", []byte(`{"firm_id":"`+inv.FirmID+`"}`), http.StatusCreated, nil)
	var board boardBody
	m.getJSON(t, "/api/projects/"+project+"/procurement/board", &board)
	if cards := board.Packages[pkg.ID]; len(cards) != 1 || cards[0].Status != "invited" || cards[0].FirmName != "Chosen Planning" || cards[0].SubmissionID != "" || cards[0].InvitedOn == "" {
		t.Fatalf("board %+v", board.Packages[pkg.ID])
	}
	for _, body := range []string{`{}`, `{"name":"A","firm_id":"` + inv.FirmID + `"}`, `{"firm_id":"x"}`, `not json`} {
		if got := m.status(t, http.MethodPost, base+"/invitations", []byte(body)); got != http.StatusBadRequest {
			t.Errorf("invite %s = %d, want 400", body, got)
		}
	}
	if got := m.status(t, http.MethodPost, base+"/invitations", []byte(`{"name":"`+strings.Repeat("x", 201)+`"}`)); got != http.StatusUnprocessableEntity {
		t.Errorf("long name = %d, want 422", got)
	}
	sibling := m.createProject(t, "Sibling project")
	if got := m.status(t, http.MethodPost, "/api/projects/"+sibling+"/procurement/packages/"+pkg.ID+"/invitations", []byte(`{"name":"Elsewhere"}`)); got != http.StatusNotFound {
		t.Errorf("another project's row = %d, want 404", got)
	}
	for _, r := range []struct{ method, path, body string }{
		{"POST", base + "/invitations", `{"name":"Intruder"}`},
		{"DELETE", base + "/invitations/" + inv.FirmID, ""},
		{"POST", base + "/submissions?firm_id=" + inv.FirmID + "&name=x.pdf", "%PDF"},
	} {
		var body []byte
		if r.body != "" {
			body = []byte(r.body)
		}
		if got := other.status(t, r.method, r.path, body); got != http.StatusNotFound {
			t.Errorf("%s %s from another org = %d, want 404", r.method, r.path, got)
		}
	}

	// The quote dropped on the firm's cell is that firm's, whatever it prints.
	var table priceTableBody
	m.getJSON(t, base+"/prices", &table)
	var lines []psample.Line
	for _, it := range table.Items {
		if it.Base {
			lines = append(lines, psample.Line{Label: it.Label, Cents: 100000})
		}
	}
	pdf := psample.PDF(psample.Proposal{Firm: "Printed Planning Pty Ltd", ABN: "88071762537", Email: "office@printed-planning.test", Lines: lines})
	if got := m.status(t, http.MethodPost, base+"/submissions?firm_id=not-a-uuid&name=x.pdf", pdf); got != http.StatusBadRequest {
		t.Fatalf("bad firm_id = %d, want 400", got)
	}
	if got := m.status(t, http.MethodPost, base+"/submissions?firm_id="+newUUID(t)+"&name=x.pdf", pdf); got != http.StatusNotFound {
		t.Fatalf("unknown firm = %d, want 404", got)
	}
	var up struct {
		SubmissionID string `json:"submission_id"`
	}
	m.call(t, http.MethodPost, base+"/submissions?firm_id="+inv.FirmID+"&name="+url.QueryEscape("Fee proposal.pdf"), pdf, http.StatusCreated, &up)
	deadline := time.Now().Add(15 * time.Second)
	for {
		m.getJSON(t, base+"/prices", &table)
		if len(table.Columns) == 1 && table.Columns[0].Submission.Status != "reading" {
			break
		}
		if time.Now().After(deadline) {
			t.Fatalf("proposal not read: %+v", table.Columns)
		}
		time.Sleep(50 * time.Millisecond)
	}
	if col := table.Columns[0]; col.Submission.ID != up.SubmissionID || col.Firm == nil || col.Firm.Name != "Chosen Planning" {
		t.Fatalf("quote on the invited cell read as %+v", col.Firm)
	}
	m.getJSON(t, "/api/projects/"+project+"/procurement/board", &board)
	if cards := board.Packages[pkg.ID]; len(cards) != 1 || cards[0].SubmissionID != up.SubmissionID || cards[0].FirmID != inv.FirmID {
		t.Fatalf("board after the quote %+v", cards)
	}

	// Removing the invitation leaves the quote; removing it twice is 404.
	m.call(t, http.MethodDelete, base+"/invitations/"+inv.FirmID, nil, http.StatusNoContent, nil)
	if got := m.status(t, http.MethodDelete, base+"/invitations/"+inv.FirmID, nil); got != http.StatusNotFound {
		t.Fatalf("second remove = %d, want 404", got)
	}
	if got := m.status(t, http.MethodDelete, base+"/invitations/not-a-uuid", nil); got != http.StatusNotFound {
		t.Fatalf("bad firm id = %d, want 404", got)
	}
	m.getJSON(t, "/api/projects/"+project+"/procurement/board", &board)
	if cards := board.Packages[pkg.ID]; len(cards) != 1 || cards[0].SubmissionID != up.SubmissionID {
		t.Fatalf("board after remove %+v", cards)
	}
}

// A document dragged from the register onto an invited cell is that firm's.
func TestRegisterDocumentLinkedToAnInvitedCell(t *testing.T) {
	a := newApp(t, withProfile(t), greenProposals)
	m := a.member(t, newUUID(t))
	project := m.createProject(t, "Invited link")
	var pkg struct {
		ID string `json:"id"`
	}
	m.call(t, http.MethodPost, "/api/projects/"+project+"/procurement/rows", []byte(`{"discipline_id":"consultant.town_planner"}`), http.StatusCreated, &pkg)
	var other struct {
		ID string `json:"id"`
	}
	m.call(t, http.MethodPost, "/api/projects/"+project+"/procurement/rows", []byte(`{"discipline_id":"consultant.architect"}`), http.StatusCreated, &other)
	base := "/api/projects/" + project + "/procurement/packages/" + pkg.ID
	var inv struct {
		FirmID string `json:"firm_id"`
	}
	m.call(t, http.MethodPost, base+"/invitations", []byte(`{"name":"Linked Planning"}`), http.StatusCreated, &inv)
	pdf := psample.PDF(psample.Proposal{Firm: "Somebody Else Pty Ltd", Lines: []psample.Line{{Label: "Fee", Cents: 100000}}})
	// Filed first on another row, then dragged onto the invited cell.
	var up struct {
		SubmissionID string `json:"submission_id"`
		DocumentID   string `json:"document_id"`
	}
	m.call(t, http.MethodPost, "/api/projects/"+project+"/procurement/packages/"+other.ID+"/submissions?name="+url.QueryEscape("Advice.pdf"), pdf, http.StatusCreated, &up)
	if got := m.status(t, http.MethodPost, base+"/documents", []byte(`{"document_id":"`+up.DocumentID+`","firm_id":"x"}`)); got != http.StatusBadRequest {
		t.Fatalf("bad firm_id = %d, want 400", got)
	}
	deadline := time.Now().Add(15 * time.Second)
	for {
		status := m.status(t, http.MethodPost, base+"/documents", []byte(`{"document_id":"`+up.DocumentID+`","firm_id":"`+inv.FirmID+`"}`))
		if status == http.StatusOK {
			break
		}
		if status != http.StatusConflict || time.Now().After(deadline) {
			t.Fatalf("link to the invited cell = %d", status)
		}
		time.Sleep(50 * time.Millisecond)
	}
	var board boardBody
	m.getJSON(t, "/api/projects/"+project+"/procurement/board", &board)
	if cards := board.Packages[pkg.ID]; len(cards) != 1 || cards[0].SubmissionID != up.SubmissionID || cards[0].FirmName != "Linked Planning" {
		t.Fatalf("board %+v", cards)
	}
}

// A misspelt invited firm is corrected in place: same cell, no Jev.
func TestInvitedFirmNameCorrected(t *testing.T) {
	a := newApp(t, withProfile(t))
	m, other := a.member(t, newUUID(t)), a.member(t, newUUID(t))
	project := m.createProject(t, "Invited firm correction")
	var pkg struct {
		ID string `json:"id"`
	}
	m.call(t, http.MethodPost, "/api/projects/"+project+"/procurement/rows", []byte(`{"discipline_id":"consultant.town_planner"}`), http.StatusCreated, &pkg)
	base := "/api/projects/" + project + "/procurement/packages/" + pkg.ID + "/invitations"
	var typo, second struct {
		FirmID   string `json:"firm_id"`
		FirmName string `json:"firm_name"`
	}
	m.call(t, http.MethodPost, base, []byte(`{"name":"Chosn Planing"}`), http.StatusCreated, &typo)
	m.call(t, http.MethodPost, base, []byte(`{"name":"Second Planning"}`), http.StatusCreated, &second)
	var fixed struct {
		FirmID   string `json:"firm_id"`
		FirmName string `json:"firm_name"`
	}
	m.call(t, http.MethodPut, base+"/"+typo.FirmID, []byte(`{"name":"Chosen Planning"}`), http.StatusOK, &fixed)
	if fixed.FirmID != typo.FirmID || fixed.FirmName != "Chosen Planning" {
		t.Fatalf("corrected %+v", fixed)
	}
	var board boardBody
	m.getJSON(t, "/api/projects/"+project+"/procurement/board", &board)
	if cards := board.Packages[pkg.ID]; len(cards) != 2 || cards[0].FirmName != "Chosen Planning" || cards[1].FirmName != "Second Planning" {
		t.Fatalf("board %+v", cards)
	}
	for _, body := range []string{`{}`, `{"name":"A","firm_id":"` + second.FirmID + `"}`, `{"firm_id":"x"}`, `not json`} {
		if got := m.status(t, http.MethodPut, base+"/"+typo.FirmID, []byte(body)); got != http.StatusBadRequest {
			t.Errorf("correct %s = %d, want 400", body, got)
		}
	}
	if got := m.status(t, http.MethodPut, base+"/"+typo.FirmID, []byte(`{"name":"second planning"}`)); got != http.StatusConflict {
		t.Errorf("a firm already on the row = %d, want 409", got)
	}
	if got := m.status(t, http.MethodPut, base+"/not-a-uuid", []byte(`{"name":"X"}`)); got != http.StatusNotFound {
		t.Errorf("bad firm path = %d, want 404", got)
	}
	if got := other.status(t, http.MethodPut, base+"/"+typo.FirmID, []byte(`{"name":"Intruder"}`)); got != http.StatusNotFound {
		t.Errorf("another org = %d, want 404", got)
	}
}

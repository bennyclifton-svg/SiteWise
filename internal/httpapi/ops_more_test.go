package httpapi_test

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"testing"

	"sitewise/internal/chatfix"
	"sitewise/internal/llm"
)

func (m *member) plainRead(t *testing.T, project, area string) string {
	t.Helper()
	_, text := m.text(t, http.MethodGet, "/api/projects/"+project+"/chat/read?area="+area, nil)
	return text
}

// The forwarded operations run the screens' own handlers. One with Undo
// applies at once and is put back by the change log; one without, and one
// the PM vouches for, are refused here with needs_click and applied only by
// the PM's click on the prepared card.
func TestForwardedOperations(t *testing.T) {
	var next []map[string]any
	area := "procurement"
	_, m, project, _ := chatApp(t, func(int, llm.Conversation) llm.Turn { return applyCall(area, true, next...) })
	prepare := func(a string, call map[string]any) (string, string) {
		t.Helper()
		area, next = a, []map[string]any{call}
		card := cardOf(t, m.chat(t, project, "", "do it: 250 square metres gross floor area"), "prepared")
		return "/api/projects/" + project + "/chat/prepared/" + card.Card.Prepared, card.Card.Why
	}
	click := func(path string, want int) string {
		t.Helper()
		_, body := m.text(t, http.MethodPost, path+"/apply", []byte(`{}`))
		if got := m.status(t, http.MethodPost, path+"/apply", []byte(`{}`)); want == http.StatusOK && got != http.StatusConflict {
			t.Fatalf("a second click = %d, want already applied", got)
		}
		return body
	}

	// A package: no Undo yet, so it waits for a click.
	add := op("procurement.add_package", map[string]any{"discipline": "consultant.structural"})
	if r := m.ops(t, project, "procurement", http.StatusUnprocessableEntity, add); !r.NeedsClick || !strings.Contains(r.Message, "cannot be undone") {
		t.Fatalf("add_package without a click: %+v", r)
	}
	path, why := prepare("procurement", add)
	if !strings.Contains(why, "cannot be undone") || strings.Contains(m.plainRead(t, project, "procurement"), "Structural") {
		t.Fatalf("why %q; the package must not exist before the click", why)
	}
	click(path, http.StatusOK)
	board := m.plainRead(t, project, "procurement")
	if !strings.Contains(board, "Structural") {
		t.Fatalf("after the click:\n%s", board)
	}
	var pkgs struct {
		Items []struct {
			ID, Title string
		} `json:"items"`
	}
	m.getJSON(t, "/api/projects/"+project+"/packages", &pkgs)
	pkg := pkgs.Items[0].ID

	// Invited firms: applied at once, and Undo puts each back.
	invited := m.ops(t, project, "procurement", http.StatusOK, op("procurement.invite_firm", map[string]any{"package": pkg[:8], "name": "Ridgeline Structures"}))
	if invited.ChangeID == "" || !strings.Contains(m.plainRead(t, project, "procurement"), "Ridgeline Structures | invited") {
		t.Fatalf("invite: %+v\n%s", invited, m.plainRead(t, project, "procurement"))
	}
	m.undo(t, project, invited.ChangeID, http.StatusOK)
	if strings.Contains(m.plainRead(t, project, "procurement"), "Ridgeline") {
		t.Fatal("undo left the invitation")
	}
	m.ops(t, project, "procurement", http.StatusOK, op("procurement.invite_firm", map[string]any{"package": pkg, "name": "Hartwell Consulting"}))
	removed := m.ops(t, project, "procurement", http.StatusOK, op("procurement.remove_invitation", map[string]any{"package": pkg, "firm": "Hartwell Consulting"}))
	if strings.Contains(m.plainRead(t, project, "procurement"), "Hartwell") {
		t.Fatal("the invitation was not removed")
	}
	m.undo(t, project, removed.ChangeID, http.StatusOK)
	if !strings.Contains(m.plainRead(t, project, "procurement"), "Hartwell Consulting | invited") {
		t.Fatal("undo did not bring the invitation back")
	}
	if r := m.ops(t, project, "procurement", http.StatusUnprocessableEntity, op("procurement.remove_invitation", map[string]any{"package": pkg, "firm": "Nobody Pty Ltd"})); r.Field != "firm" {
		t.Fatalf("an unknown firm: %+v", r)
	}

	// Vouching: an award is prepared, and the click sends the PM to the dialog.
	path, why = prepare("procurement", op("procurement.award", map[string]any{"package": pkg, "firm": "Hartwell Consulting"}))
	if !strings.Contains(why, "yours to confirm") || !strings.Contains(click(path, http.StatusUnprocessableEntity), "award dialog") {
		t.Fatalf("award why %q", why)
	}
	// Removing a package is prepared, with what goes with it said.
	path, why = prepare("procurement", op("procurement.remove_package", map[string]any{"package": pkg}))
	if !strings.Contains(why, "draft cost lines go with it") {
		t.Fatalf("remove why %q", why)
	}
	click(path, http.StatusOK)
	if strings.Contains(m.plainRead(t, project, "procurement"), "Structural") {
		t.Fatalf("the package is still on the board:\n%s", m.plainRead(t, project, "procurement"))
	}

	// The accommodation schedule: a room is one change with Undo.
	before := m.plainRead(t, project, "accommodation")
	room := m.ops(t, project, "profile", http.StatusOK, op("profile.add_room", map[string]any{"name": "Bedroom", "level": "Level 1", "area": 14}))
	if !strings.Contains(m.plainRead(t, project, "accommodation"), "Bedroom | Level 1 | 14") {
		t.Fatalf("room: %+v\n%s", room, m.plainRead(t, project, "accommodation"))
	}
	m.undo(t, project, room.ChangeID, http.StatusOK)
	if after := m.plainRead(t, project, "accommodation"); after != before {
		t.Fatalf("undo did not restore the schedule:\n%s\n---\n%s", before, after)
	}

	// Cost Plan settings ride the change log like the other cost writes.
	m.ops(t, project, "cost_plan", http.StatusOK, op("cost_plan.add_line", map[string]any{"label": "DA fees", "section": "authority"}))
	set := m.ops(t, project, "cost_plan", http.StatusOK, op("cost_plan.settings", map[string]any{"currency": "AUD", "tax_basis": "ex_tax", "tax_rate": "0.1", "coverage": ""}))
	if p := planOf(t, m, project); p.TaxRate == nil {
		t.Fatalf("settings: %+v", set)
	}
	m.undo(t, project, set.ChangeID, http.StatusOK)
	if p := planOf(t, m, project); p.TaxRate != nil {
		t.Fatal("undo left the GST rate")
	}

	// A planning value and a document's filing: prepared, then applied by the click.
	path, _ = prepare("profile", op("profile.set_planning", map[string]any{"key": "gross_floor_area", "value": 250, "rationale": "from the brief"}))
	click(path, http.StatusOK)
	var planning struct {
		Values []struct {
			Key   string  `json:"key"`
			Value *string `json:"value"`
		} `json:"values"`
	}
	m.getJSON(t, "/api/projects/"+project+"/planning", &planning)
	found := false
	for _, v := range planning.Values {
		found = found || (v.Key == "gross_floor_area" && v.Value != nil && *v.Value == "250")
	}
	if !found {
		t.Fatalf("planning values %+v", planning.Values)
	}
}

func TestDocumentFilingCorrectionIsPrepared(t *testing.T) {
	var next map[string]any
	a, m, project, _ := chatApp(t, func(int, llm.Conversation) llm.Turn { return applyCall("documents", true, next) })
	ids, err := chatfix.SeedDocuments(context.Background(), a.store, m.orgID, project, []chatfix.Document{{Title: "Site survey", Kind: "drawing"}})
	if err != nil {
		t.Fatal(err)
	}
	next = op("documents.correct_field", map[string]any{"document": ids["Site survey"][:8], "field": "discipline", "value": "consultant.acoustic"})
	card := cardOf(t, m.chat(t, project, "", "File the survey under acoustic"), "prepared")
	m.call(t, http.MethodPost, "/api/projects/"+project+"/chat/prepared/"+card.Card.Prepared+"/apply", []byte(`{}`), http.StatusOK, nil)
	var doc struct {
		Fields []struct {
			Field, Value string
		} `json:"fields"`
	}
	m.getJSON(t, "/api/documents/"+ids["Site survey"], &doc)
	raw, _ := json.Marshal(doc)
	if !strings.Contains(string(raw), `"consultant.acoustic"`) {
		t.Fatalf("the field was not corrected: %s", raw)
	}
}

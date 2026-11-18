package httpapi_test

import (
	"net/http"
	"reflect"
	"testing"

	"sitewise/internal/delivery"
)

// The register's operations (docs/plans/2026-11-17-rfis-review-comments.md,
// Task 6): each screen action through POST /ops has one Undo; accepting an
// answer from chat waits for the PM's click.
func TestRegisterOperationsApplyWithUndo(t *testing.T) {
	a := newApp(t, withProfile(t))
	m := a.member(t, newUUID(t))
	project := m.createProject(t, "Register ops")
	base := "/api/projects/" + project + "/delivery"
	added := m.ops(t, project, "delivery", http.StatusOK, op("delivery.add", map[string]any{"kind": "rfi", "title": "Basement wall waterproofing", "question": "Which membrane?", "raised_on": "2026-11-20", "system": "substructure"}))
	if added.Undo.Kind != "change" || added.ChangeID == "" {
		t.Fatalf("add not undoable %+v", added)
	}
	var view struct {
		Items []delivery.Item `json:"items"`
	}
	m.getJSON(t, base, &view)
	if len(view.Items) != 1 || *view.Items[0].Number != 1 || delivery.RFI(view.Items[0].Content).Question != "Which membrane?" || *view.Items[0].TargetDate != "2026-11-27" {
		t.Fatalf("added %+v", view.Items)
	}
	// Three edits by RFI number in one batch, undone as one change.
	before := withoutVersions(t, m, base)
	batch := m.ops(t, project, "delivery", http.StatusOK,
		op("delivery.edit", map[string]any{"item": "RFI-001", "status": "open"}),
		op("delivery.edit", map[string]any{"item": "RFI 1", "owner_text": "Structural Co"}),
		op("delivery.edit", map[string]any{"item": "rfi-001", "system": "envelope"}))
	m.getJSON(t, base, &view)
	if view.Items[0].Status != "open" || view.Items[0].With != "Structural Co" || view.Items[0].SystemID != "envelope" {
		t.Fatalf("edited %+v", view.Items[0])
	}
	m.undo(t, project, batch.ChangeID, http.StatusOK)
	if after := withoutVersions(t, m, base); !reflect.DeepEqual(before, after) {
		t.Fatalf("undo did not put the row back:\n%v\n%v", before, after)
	}
	// The response time is an operation with Undo too.
	set := m.ops(t, project, "delivery", http.StatusOK, op("delivery.settings", map[string]any{"rfi_response_days": 10}))
	m.undo(t, project, set.ChangeID, http.StatusOK)
	var settings struct {
		ResponseDays    int  `json:"response_days"`
		ResponseTypical bool `json:"response_typical"`
	}
	m.getJSON(t, base, &settings)
	if settings.ResponseDays != 5 || !settings.ResponseTypical {
		t.Fatalf("settings after undo %+v", settings)
	}
	// Accepting from the operations call is the PM's to confirm.
	refused := m.ops(t, project, "delivery", http.StatusUnprocessableEntity, op("delivery.accept", map[string]any{"item": "RFI-001", "reading": "00000000"}))
	if !refused.NeedsClick {
		t.Fatalf("accept did not wait for a click %+v", refused)
	}
	// A raised date hidden in details is refused, so chat's figure check sees every date.
	hidden := m.ops(t, project, "delivery", http.StatusUnprocessableEntity, op("delivery.add", map[string]any{"kind": "rfi", "title": "Hidden", "details": map[string]any{"raised_on": "2026-11-20"}}))
	if hidden.Field != "details" {
		t.Fatalf("hidden date %+v", hidden)
	}
	m.ops(t, project, "delivery", http.StatusUnprocessableEntity, op("delivery.edit", map[string]any{"item": "RFI-009", "status": "closed"}))
	if a.jevHits.Load() != 0 {
		t.Fatal("a register operation called Jev")
	}
}

package httpapi_test

import (
	"context"
	"net/http"
	"reflect"
	"testing"

	"sitewise/internal/chatfix"
	"sitewise/internal/delivery"
)

// An RFI's documents are its transmittal, set as an RFP's is: the whole
// list in one edit with Undo, read back as the RFP transmittal's rows
// (owner, 20 November 2026).
func TestRFITransmittal(t *testing.T) {
	a := newApp(t, withProfile(t))
	m := a.member(t, newUUID(t))
	project := m.createProject(t, "RFI transmittal")
	other := m.createProject(t, "Another job")
	ctx := context.Background()
	ids, err := chatfix.SeedDocuments(ctx, a.store, m.orgID, project, []chatfix.Document{
		{Title: "Design Brief", Kind: "report", Discipline: "architectural", Pages: []string{"The brief."}},
		{Title: "Structural notes", Kind: "drawing", Discipline: "structural", Number: "S-001", Revision: "B", Pages: []string{"Slab 250 thick."}},
	})
	if err != nil {
		t.Fatal(err)
	}
	theirs, err := chatfix.SeedDocuments(ctx, a.store, m.orgID, other, []chatfix.Document{{Title: "Other job plan", Kind: "drawing", Pages: []string{"Plan."}}})
	if err != nil {
		t.Fatal(err)
	}
	base := "/api/projects/" + project + "/delivery"
	m.ops(t, project, "delivery", http.StatusOK, op("delivery.add", map[string]any{"kind": "rfi", "title": "Slab thickness", "raised_on": "2026-11-20"}))
	read := func() delivery.Item {
		var view struct {
			Items []delivery.Item `json:"items"`
		}
		m.getJSON(t, base, &view)
		if len(view.Items) != 1 {
			t.Fatalf("items %+v", view.Items)
		}
		return view.Items[0]
	}
	if got := read().Transmittal; len(got) != 0 {
		t.Fatalf("a new RFI has documents: %v", got)
	}
	// The screen's Update Transmittal: the selection replaces the list.
	refs := func(titles ...string) []map[string]string {
		out := []map[string]string{}
		for _, title := range titles {
			out = append(out, map[string]string{"document_id": ids[title]})
		}
		return out
	}
	m.ops(t, project, "delivery", http.StatusOK, op("delivery.edit", map[string]any{"item": "RFI-001", "source_refs": refs("Structural notes", "Design Brief")}))
	want := [][]string{{"S-001", "Structural notes", "B", "", "structural"}, {"Not stated", "Design Brief", "Not stated", "", "architectural"}}
	if got := read().Transmittal; !reflect.DeepEqual(got, want) {
		t.Fatalf("transmittal\n got %v\nwant %v", got, want)
	}
	// Chat's field: document ids, the whole list.
	m.ops(t, project, "delivery", http.StatusOK, op("delivery.edit", map[string]any{"item": "RFI-001", "documents": []string{ids["Design Brief"]}}))
	if got := read().Transmittal; len(got) != 1 || got[0][1] != "Design Brief" {
		t.Fatalf("after chat's edit %v", got)
	}
	// Another project's document is refused and nothing changes.
	m.ops(t, project, "delivery", http.StatusNotFound, op("delivery.edit", map[string]any{"item": "RFI-001", "source_refs": []map[string]string{{"document_id": theirs["Other job plan"]}}}))
	if got := read().Transmittal; len(got) != 1 {
		t.Fatalf("a foreign document changed the list: %v", got)
	}
	// An empty selection clears it, and Undo puts the list back.
	cleared := m.ops(t, project, "delivery", http.StatusOK, op("delivery.edit", map[string]any{"item": "RFI-001", "source_refs": []map[string]string{}}))
	if got := read().Transmittal; len(got) != 0 {
		t.Fatalf("cleared %v", got)
	}
	m.undo(t, project, cleared.ChangeID, http.StatusOK)
	if got := read().Transmittal; len(got) != 1 || got[0][1] != "Design Brief" {
		t.Fatalf("after undo %v", got)
	}
}

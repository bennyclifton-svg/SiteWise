package reports

import (
	"encoding/json"
	"reflect"
	"strings"
	"testing"
	"time"

	"sitewise/internal/knowledge"
)

func TestCitationsSurviveEditsAndPrintWithoutApp(t *testing.T) {
	blocks := []Block{
		{ID: "evidence", Label: "Building class", Origin: "document", Meaning: "stated", ReviewStatus: "proposed", Basis: json.RawMessage(`{"Sources":[{"document_id":"doc","filename":"Fire.pdf","document_number":"F-01","revision":"C","page":7,"location":"Warehouse A","section":"2.1"}]}`)},
		{ID: "assumption", Label: "Existing capacity", Text: "Adequate", Origin: "assumption", Basis: json.RawMessage(`{"actor":"owner"}`)},
		{ID: "calculation", Label: "Allocation", Origin: "calculation", Basis: json.RawMessage(`{"method":"gap_check"}`)},
		{ID: "user", Label: "Appointment", Origin: "user", Basis: json.RawMessage(`{"actor":"owner"}`)},
	}
	generated, err := ApplyEdits([]Section{{ID: "brief", Blocks: blocks}}, nil)
	if err != nil {
		t.Fatal(err)
	}
	edited, err := ApplyEdits(generated, []Edit{{TargetID: "evidence", Text: "Protected wording", UserID: "editor", UpdatedAt: time.Date(2026, 10, 5, 4, 0, 0, 0, time.UTC), BaseContentSHA256: generated[0].Blocks[0].ContentSHA256}})
	if err != nil {
		t.Fatal(err)
	}
	cited, refs, err := Cite(edited)
	if err != nil {
		t.Fatal(err)
	}
	if len(refs) != 5 || strings.Join(cited[0].Blocks[0].CitationIDs, ",") != "E1,U1" {
		t.Fatalf("references %+v", refs)
	}
	for _, text := range []string{"Evidence", "Fire.pdf", "F-01", "revision: C", "page: 7", "Warehouse A", "2.1"} {
		if !strings.Contains(refs[0].Basis.Text, text) {
			t.Fatalf("missing %s: %s", text, refs[0].Basis.Text)
		}
	}
	if !refs[1].Basis.ProtectedEdit || refs[1].Basis.UserID != "editor" || !refs[2].Basis.MaterialAssumption {
		t.Fatal("provenance lost", refs)
	}
	if !strings.Contains(refs[1].Basis.Text, "2026-10-05T04:00:00Z") {
		t.Fatal("edit time omitted")
	}
	if len(edited[0].Blocks[0].CitationIDs) != 0 {
		t.Fatal("mutated caller")
	}
	again, againRefs, err := Cite(cited)
	if err != nil || !reflect.DeepEqual(cited, again) || !reflect.DeepEqual(refs, againRefs) {
		t.Fatal("unstable citation refresh", err)
	}
	before, _ := ContentHash(edited[0].Blocks[2])
	after, _ := ContentHash(cited[0].Blocks[2])
	if before != after {
		t.Fatal("citations altered generated hash")
	}
}

func TestCalculationReferenceUsesSavedClauseVersionAndInputs(t *testing.T) {
	clause := knowledge.Clause{ID: "cl.saved", Version: 3, Status: "draft"}
	raw, err := json.Marshal(map[string]any{"method": "catalogue_clause", "clause": clause})
	if err != nil {
		t.Fatal(err)
	}
	_, refs, err := Cite([]Section{{Blocks: []Block{{ID: "clause", Origin: "calculation", Basis: raw}, {ID: "proposal", Origin: "calculation", Basis: json.RawMessage(`{"reason":{"record":"cq.saved","triggers":[{"work_item_id":"work","action":"replace","system":"sprinklers","part":"Warehouse A"}],"determinants":[{"key":"height","value":"12","origin":"document"}]}}`)}}}})
	if err != nil {
		t.Fatal(err)
	}
	for _, value := range []string{"cl.saved", "version: 3", "review: draft"} {
		if !strings.Contains(refs[0].Basis.Text, value) {
			t.Fatal("missing catalogue basis", value, refs[0])
		}
	}
	for _, value := range []string{"cq.saved", "replace sprinklers in Warehouse A", "height = 12 (document)"} {
		if !strings.Contains(refs[1].Basis.Text, value) {
			t.Fatal("missing calculation input", value, refs[1])
		}
	}
}

func TestCitationMissingSourceAndMalformedBasis(t *testing.T) {
	sections, err := ApplyEdits(nil, []Edit{{TargetID: "removed", Text: "Keep this", UserID: "author"}})
	if err != nil {
		t.Fatal(err)
	}
	_, refs, err := Cite(sections)
	if err != nil || len(refs) != 1 || refs[0].Label != "U" || !strings.Contains(refs[0].Basis.Text, "no longer present") {
		t.Fatal(refs, err)
	}
	for _, basis := range []string{"null", "[]", "broken"} {
		_, _, err := Cite([]Section{{Blocks: []Block{{ID: "bad", Basis: json.RawMessage(basis)}}}})
		if err == nil {
			t.Fatal("accepted malformed basis", basis)
		}
	}
}

func TestCorrectedWorkCitationUsesLatestEditor(t *testing.T) {
	_, refs, err := Cite([]Section{{Blocks: []Block{{ID: "work:saved", Origin: "user", Basis: json.RawMessage(`{"provenance":{"actor":"original","last_edited_by":"corrector","last_edited_at":"2026-10-05T12:00:00Z"}}`)}}}})
	if err != nil || len(refs) != 1 {
		t.Fatal(refs, err)
	}
	if !strings.Contains(refs[0].Basis.Text, "Author: corrector") || !strings.Contains(refs[0].Basis.Text, "Recorded: 2026-10-05T12:00:00Z") || strings.Contains(refs[0].Basis.Text, "Author: original") {
		t.Fatal(refs[0].Basis.Text)
	}
}

package reports

import (
	"encoding/json"
	"testing"
)

func TestResetProtectedEditRestoresSnapshotAndDropsOrphan(t *testing.T) {
	generated := []Section{{ID: "brief", Blocks: []Block{{ID: "value", Text: "Saved source wording", Origin: "user", Basis: json.RawMessage(`{}`)}}}}
	sections, err := ApplyEdits(generated, []Edit{{TargetID: "value", Text: "Protected", UserID: "writer"}, {TargetID: "gone", Text: "Orphaned text", UserID: "writer"}})
	if err != nil {
		t.Fatal(err)
	}
	reset, err := ResetEdit(sections, "value")
	if err != nil {
		t.Fatal(err)
	}
	b := reset[0].Blocks[0]
	if b.Text != "Saved source wording" || b.Edited || b.Conflict || b.EditUserID != "" || b.GeneratedText != nil {
		t.Fatal("reset lost generated source", b)
	}
	if sections[0].Blocks[0].Text != "Protected" {
		t.Fatal("mutated caller")
	}
	hash, _ := ContentHash(b)
	if hash != b.ContentSHA256 {
		t.Fatal("reset changed source hash")
	}
	reset, err = ResetEdit(reset, "gone")
	if err != nil || len(reset) != 1 {
		t.Fatal("orphan section retained", reset, err)
	}
	_, refs, err := Cite(reset)
	if err != nil || len(refs) != 1 || refs[0].Basis.ProtectedEdit {
		t.Fatal("protected citation retained", refs, err)
	}
	if _, err := ResetEdit(reset, "value"); err == nil {
		t.Fatal("reset unedited source")
	}
}

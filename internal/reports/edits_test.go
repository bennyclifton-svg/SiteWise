package reports

import (
	"encoding/json"
	"testing"
)

func TestProtectedEditsSurviveChangedAndRemovedSources(t *testing.T) {
	block := Block{ID: "scope:one", Text: "Generated scope", Origin: "user", Basis: json.RawMessage(`{"version":1}`)}
	hash, err := ContentHash(block)
	if err != nil {
		t.Fatal(err)
	}
	sections := []Section{{ID: "services", Essential: true, Blocks: []Block{block}}}
	edit := Edit{TargetID: block.ID, Text: "Protected wording", UserID: "actor", BaseContentSHA256: hash, Version: 1}
	out, err := ApplyEdits(sections, []Edit{edit})
	if err != nil {
		t.Fatal(err)
	}
	got := out[0].Blocks[0]
	if got.Text != edit.Text || !got.Edited || got.Conflict || got.ContentSHA256 != hash || !out[0].Essential {
		t.Fatal(got)
	}
	if sections[0].Blocks[0].Text != block.Text {
		t.Fatal("mutated saved generation")
	}
	sections[0].Blocks[0].Basis = json.RawMessage(`{"version":2}`)
	out, err = ApplyEdits(sections, []Edit{edit})
	if err != nil || !out[0].Blocks[0].Conflict || out[0].Blocks[0].Text != edit.Text {
		t.Fatalf("changed source %+v %v", out, err)
	}
	out, err = ApplyEdits(nil, []Edit{edit})
	if err != nil || len(out) != 1 || !out[0].Blocks[0].SourceMissing || out[0].Blocks[0].Text != edit.Text {
		t.Fatalf("removed source %+v %v", out, err)
	}
	if _, err := ApplyEdits(sections, []Edit{edit, edit}); err == nil {
		t.Fatal("duplicate edit")
	}
	sections[0].Blocks = append(sections[0].Blocks, block)
	if _, err := ApplyEdits(sections, nil); err == nil {
		t.Fatal("duplicate generated target")
	}
}

func TestReportDependenciesExcludeOwnWritesAndM2Costs(t *testing.T) {
	saved := SourceState{Domains: map[string]int64{"profile_inputs": 1, "works": 2, "packages": 3, "delivery": 4}, ProfileRevision: 1, ProfileFingerprint: "fingerprint", KnowledgeVersion: "knowledge", QuestionVersion: "question", ThresholdsVersion: "thresholds", AppBuild: "build", TemplateID: "tpl.rfp-capex", TemplateVersion: 1}
	current := saved
	current.Domains = map[string]int64{"profile_inputs": 1, "works": 2, "packages": 3, "delivery": 4, "reports": 99, "costs": 99}
	if got := StaleReasons("rfp", saved, current); len(got) != 0 {
		t.Fatal(got)
	}
	current.Domains["delivery"]++
	current.TemplateVersion++
	got := StaleReasons("rfp", saved, current)
	if len(got) != 2 || got[0] != "delivery" || got[1] != "template" {
		t.Fatal(got)
	}
	if got := StaleReasons("rfp", SourceState{}, current); len(got) == 0 {
		t.Fatal("missing snapshot considered fresh")
	}
}

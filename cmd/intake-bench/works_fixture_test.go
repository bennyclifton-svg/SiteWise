package main

import (
	"encoding/json"
	"testing"

	"sitewise/internal/eval"
)

func TestManualTimingFixtureRejectsStaleAndInventedInputs(t *testing.T) {
	key := eval.WorkKey{SchemaVersion: 2, Project: "0991", Documents: []string{"drawing"}, Parts: []string{"A"}, Items: []eval.ExpectedWork{{ID: "work", System: "fire-active.hydrants", Action: "upgrade", Part: "A", Document: "drawing", Page: 1}}}
	corpus := map[string]string{"drawing": "digest"}
	base := map[string]any{
		"sites":         []any{map[string]any{"id": "site"}},
		"files":         []any{map[string]any{"id": "file", "sha256": "\\xdigest"}},
		"documents":     []any{map[string]any{"id": "document", "file_id": "file"}},
		"passages":      []any{map[string]any{"document_id": "document", "body": "Synthetic fixture text"}},
		"project_parts": []any{map[string]any{"id": "part", "label": "A"}},
		"work_items":    []any{map[string]any{"id": "work", "part_id": "part", "system_id": "fire-active.hydrants", "action": "upgrade", "inclusion": "included"}},
	}
	for _, test := range []string{"valid", "key", "source", "file", "document", "passage", "action", "invented-fact", "kind"} {
		t.Run(test, func(t *testing.T) {
			body, _ := json.Marshal(base)
			f := manualWorkFixture{Kind: "manual-source-timing", Project: "0991", KeySHA256: "key", Corpus: map[string]string{"drawing": "digest"}}
			if err := json.Unmarshal(body, &f.Tables); err != nil {
				t.Fatal(err)
			}
			set := func(table, field, value string) { f.Tables[table][0][field], _ = json.Marshal(value) }
			switch test {
			case "key":
				f.KeySHA256 = "old"
			case "source":
				f.Corpus["drawing"] = "changed"
			case "file":
				set("files", "sha256", "wrong")
			case "document":
				set("documents", "file_id", "other")
			case "passage":
				set("passages", "document_id", "other")
			case "action":
				set("work_items", "action", "new")
			case "invented-fact":
				f.Tables["profile_facts"] = []map[string]json.RawMessage{{}}
			case "kind":
				f.Kind = "machine-extraction"
			}
			err := validateManualWorkFixture(f, key, "key", corpus)
			if (err == nil) != (test == "valid") {
				t.Fatalf("%v", err)
			}
		})
	}
}

package main

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

// Synthetic import plumbing test, never release evidence for Spec Home.
func syntheticProfileFixture(t *testing.T) string {
	t.Helper()
	if path := os.Getenv("SITEWISE_PROFILE_BENCH_FIXTURE"); path != "" {
		return path
	}
	tables := map[string][]map[string]any{"sites": {{"label": "Synthetic"}}}
	id := func(n int) string { return fmt.Sprintf("14000000-0000-4000-8000-%012d", n) }
	tables["project_parts"] = []map[string]any{{"id": id(1), "site_id": id(2), "created_by_project_id": id(3), "kind": "whole", "label": "Whole project"}}
	for i := 0; i < 38; i++ {
		tables["files"] = append(tables["files"], map[string]any{"id": id(100 + i), "project_id": id(3), "sha256": fmt.Sprintf("\\x%064x", i+1), "byte_size": 8, "media_type": "application/pdf"})
		tables["documents"] = append(tables["documents"], map[string]any{"id": id(200 + i), "project_id": id(3), "file_id": id(100 + i), "filename": "fixture.pdf", "status": "filed"})
	}
	for i := 0; i < 6742; i++ {
		tables["passages"] = append(tables["passages"], map[string]any{"id": id(10000 + i), "document_id": id(200 + i%38), "ordinal": i + 1, "body": "Test source"})
	}
	for i := 0; i < 1153; i++ {
		tables["profile_facts"] = append(tables["profile_facts"], map[string]any{"id": id(20000 + i), "project_id": id(3), "document_id": id(200 + i%38), "passage_id": id(10000 + i), "question_id": "det.bal", "value": "BAL-40", "decided_by": "rule", "question_version": "test"})
	}
	path := filepath.Join(t.TempDir(), "profile.json")
	writeTestJSON(t, path, tables)
	return path
}

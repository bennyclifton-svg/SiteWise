package knowledge

import (
	"os"
	"path/filepath"
	"testing"
)

func TestClauseReferencesNeedReviewedExactVersion(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "reports"), 0755); err != nil {
		t.Fatal(err)
	}
	body := "version: 1\nclauses:\n  - {id: cl.test, version: 2, text: Fixture wording, status: reviewed, section: brief, outputs: [rfp], sources: [{design: fixture}]}\n  - {id: cl.draft, version: 1, text: Draft fixture, status: draft, section: brief, outputs: [rfp], sources: [{design: fixture}]}\n"
	if err := os.WriteFile(filepath.Join(root, "reports", "clauses.yaml"), []byte(body), 0600); err != nil {
		t.Fatal(err)
	}
	c := &Catalog{root: root, loaded: map[string][]byte{}}
	if err := c.loadClauses(root); err != nil {
		t.Fatal(err)
	}
	if !c.ApprovedClause("cl.test", 2) || c.ApprovedClause("cl.test", 1) || c.ApprovedClause("cl.draft", 1) || c.ApprovedClause("missing", 1) {
		t.Fatal("unapproved clause version admitted")
	}
	if len(c.loaded) != 1 {
		t.Fatal("clause omitted from knowledge fingerprint")
	}
}

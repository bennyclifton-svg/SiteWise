package knowledge_test

import (
	"os"
	"path/filepath"
	"testing"

	"sitewise/internal/knowledge"
)

func TestLoadedKnowledgeVersion(t *testing.T) {
	root := writeFixture(t)
	before, err := knowledge.Load(root)
	if err != nil {
		t.Fatal(err)
	}
	original := before.Version()
	if len(original) != 64 {
		t.Fatalf("invalid hash %q", original)
	}
	// Unloaded prose is not an input to the running catalogue.
	if err := os.WriteFile(filepath.Join(root, "README.md"), []byte("changed"), 0600); err != nil {
		t.Fatal(err)
	}
	same, err := knowledge.Load(root)
	if err != nil || same.Version() != original {
		t.Fatalf("unloaded file changed hash: %v", err)
	}
	path := filepath.Join(root, "systems.yaml")
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, append(raw, []byte("\n# revised catalogue\n")...), 0600); err != nil {
		t.Fatal(err)
	}
	if before.Version() != original {
		t.Fatal("running catalogue changed with disk")
	}
	after, err := knowledge.Load(root)
	if err != nil || after.Version() == original {
		t.Fatalf("loaded file not versioned: %v", err)
	}
}

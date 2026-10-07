package eval

import (
	"crypto/sha256"
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

func TestWorkCorpusFailsClosed(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "drawing.pdf")
	if err := os.WriteFile(path, []byte("fixture"), 0600); err != nil {
		t.Fatal(err)
	}
	hash := fmt.Sprintf("%x", sha256.Sum256([]byte("fixture")))
	key := WorkKey{Project: "fixture", Documents: []string{"drawing"}}
	m := ProfileManifest{Projects: []CorpusProject{{ID: "fixture", Documents: key.Documents}}, Documents: []CorpusDocument{{ID: "drawing", Path: "drawing.pdf", SHA256: hash}}}
	for _, test := range []string{"valid", "changed", "missing", "escape", "absolute", "duplicate", "extra-key-document"} {
		t.Run(test, func(t *testing.T) {
			copyM := m
			copyM.Documents = append([]CorpusDocument(nil), m.Documents...)
			copyKey := key
			switch test {
			case "changed":
				copyM.Documents[0].SHA256 = "wrong"
			case "missing":
				copyM.Documents[0].Path = "missing.pdf"
			case "escape":
				copyM.Documents[0].Path = "../drawing.pdf"
			case "absolute":
				copyM.Documents[0].Path = "C:/drawing.pdf"
			case "duplicate":
				copyM.Documents = append(copyM.Documents, copyM.Documents[0])
			case "extra-key-document":
				copyKey.Documents = []string{"drawing", "extra"}
			}
			got, err := VerifyWorkCorpus(copyM, copyKey, root)
			if test == "valid" {
				if err != nil || got["drawing"] != hash {
					t.Fatalf("%v %v", got, err)
				}
			} else if err == nil {
				t.Fatal("accepted invalid corpus")
			}
		})
	}
}

func TestWorkSnapshotRejectsChangedPins(t *testing.T) {
	a := WorkSnapshot{Model: "jev", QuestionVersion: "q1", KnowledgeVersion: "k1", ThresholdsVersion: "t1", AppBuild: "build", KeySHA256: "key", Corpus: map[string]string{"doc": "hash"}}
	for _, test := range []string{"valid", "key", "corpus", "version"} {
		t.Run(test, func(t *testing.T) {
			copyA := a
			switch test {
			case "key":
				copyA.KeySHA256 = "old"
			case "corpus":
				copyA.Corpus = map[string]string{"doc": "old"}
			case "version":
				copyA.QuestionVersion = ""
			}
			err := VerifyWorkSnapshot(copyA, "key", map[string]string{"doc": "hash"})
			if (err == nil) != (test == "valid") {
				t.Fatalf("%v", err)
			}
		})
	}
}

func TestWorkVersionsRejectStaleRun(t *testing.T) {
	actual := WorkSnapshot{Model: "jev", QuestionVersion: "q", KnowledgeVersion: "k", ThresholdsVersion: "t", AppBuild: "build"}
	expected := WorkVersions{"jev", "q", "k", "t", "build"}
	if err := VerifyWorkVersions(actual, expected); err != nil {
		t.Fatal(err)
	}
	actual.QuestionVersion = "old"
	if err := VerifyWorkVersions(actual, expected); err == nil {
		t.Fatal("accepted stale question version")
	}
	expected.AppBuild = ""
	if err := VerifyWorkVersions(actual, expected); err == nil {
		t.Fatal("accepted incomplete run pins")
	}
}

func TestWorkKeyRejectsUnknownFieldsAndMultipleDocuments(t *testing.T) {
	for _, body := range []string{"project: fixture\nunknown: true\n", "project: fixture\n---\nproject: other\n"} {
		path := filepath.Join(t.TempDir(), "key.yaml")
		if err := os.WriteFile(path, []byte(body), 0600); err != nil {
			t.Fatal(err)
		}
		if _, _, err := LoadWorkKey(path); err == nil {
			t.Fatal("accepted ambiguous key")
		}
	}
}

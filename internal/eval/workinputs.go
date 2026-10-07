package eval

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

type ProfileManifest struct {
	RootEnv     string           `json:"root_env"`
	RootDefault string           `json:"root_default"`
	Note        string           `json:"note"`
	Projects    []CorpusProject  `json:"projects"`
	Documents   []CorpusDocument `json:"documents"`
}
type CorpusProject struct {
	ID        string   `json:"id"`
	Documents []string `json:"documents"`
}
type CorpusDocument struct {
	ID     string `json:"id"`
	Path   string `json:"path"`
	SHA256 string `json:"sha256"`
}

func LoadWorkKey(path string) (WorkKey, string, error) {
	var key WorkKey
	b, err := os.ReadFile(path)
	if err != nil {
		return key, "", err
	}
	d := yaml.NewDecoder(bytes.NewReader(b))
	d.KnownFields(true)
	if err = d.Decode(&key); err != nil {
		return key, "", err
	}
	var extra any
	if err = d.Decode(&extra); err != io.EOF {
		return key, "", fmt.Errorf("key must contain one YAML document")
	}
	h := sha256.Sum256(b)
	return key, hex.EncodeToString(h[:]), nil
}

func LoadProfileManifest(path string) (ProfileManifest, error) {
	var m ProfileManifest
	err := ReadStrictJSON(path, &m)
	return m, err
}

func ReadStrictJSON(path string, value any) error {
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()
	d := json.NewDecoder(f)
	d.DisallowUnknownFields()
	if err := d.Decode(value); err != nil {
		return err
	}
	var extra any
	if err := d.Decode(&extra); err != io.EOF {
		return fmt.Errorf("expected one JSON value")
	}
	return nil
}

// VerifyWorkCorpus hashes local files without reading their contents into result
// artifacts. Resolve links before opening so a manifest cannot escape its root.
func VerifyWorkCorpus(m ProfileManifest, key WorkKey, root string) (map[string]string, error) {
	if root == "" && m.RootEnv != "" {
		root = os.Getenv(m.RootEnv)
	}
	if root == "" {
		root = m.RootDefault
	}
	if root == "" {
		return nil, fmt.Errorf("corpus root required")
	}
	root, err := filepath.Abs(root)
	if err != nil {
		return nil, err
	}
	root, err = filepath.EvalSymlinks(root)
	if err != nil {
		return nil, fmt.Errorf("corpus root unavailable")
	}
	projects := map[string]CorpusProject{}
	for _, p := range m.Projects {
		if p.ID == "" || projects[p.ID].ID != "" {
			return nil, fmt.Errorf("invalid manifest projects")
		}
		projects[p.ID] = p
	}
	p, ok := projects[key.Project]
	if !ok || len(p.Documents) == 0 {
		return nil, fmt.Errorf("project absent or empty in manifest")
	}
	docs := map[string]CorpusDocument{}
	for _, d := range m.Documents {
		if d.ID == "" || docs[d.ID].ID != "" {
			return nil, fmt.Errorf("invalid manifest documents")
		}
		docs[d.ID] = d
	}
	wanted := map[string]bool{}
	for _, id := range key.Documents {
		if id == "" || wanted[id] {
			return nil, fmt.Errorf("invalid key document list")
		}
		wanted[id] = true
	}
	if len(wanted) != len(p.Documents) {
		return nil, fmt.Errorf("key and project document sets differ")
	}
	out := map[string]string{}
	for _, id := range p.Documents {
		d, exists := docs[id]
		if !exists || !wanted[id] || out[id] != "" {
			return nil, fmt.Errorf("invalid project document reference")
		}
		// Accept portable relative paths only, including when running on Unix.
		rel := strings.ReplaceAll(d.Path, "\\", "/")
		if rel == "" || strings.HasPrefix(rel, "/") || strings.Contains(rel, ":") {
			return nil, fmt.Errorf("invalid corpus path for %s", id)
		}
		for _, part := range strings.Split(rel, "/") {
			if part == ".." || part == "" {
				return nil, fmt.Errorf("invalid corpus path for %s", id)
			}
		}
		path, err := filepath.EvalSymlinks(filepath.Join(root, filepath.FromSlash(rel)))
		if err != nil {
			return nil, fmt.Errorf("corpus document %s unavailable", id)
		}
		within, err := filepath.Rel(root, path)
		if err != nil || within == ".." || strings.HasPrefix(within, ".."+string(filepath.Separator)) || filepath.IsAbs(within) {
			return nil, fmt.Errorf("corpus document %s escapes root", id)
		}
		f, err := os.Open(path)
		if err != nil {
			return nil, fmt.Errorf("corpus document %s unavailable", id)
		}
		h := sha256.New()
		_, copyErr := io.Copy(h, f)
		closeErr := f.Close()
		if copyErr != nil || closeErr != nil {
			return nil, fmt.Errorf("cannot hash corpus document %s", id)
		}
		digest := hex.EncodeToString(h.Sum(nil))
		if digest != d.SHA256 {
			return nil, fmt.Errorf("corpus document %s hash mismatch", id)
		}
		out[id] = digest
	}
	return out, nil
}

func VerifyWorkSnapshot(actual WorkSnapshot, keyHash string, corpus map[string]string) error {
	if actual.Model == "" || actual.QuestionVersion == "" || actual.KnowledgeVersion == "" || actual.ThresholdsVersion == "" || actual.AppBuild == "" {
		return fmt.Errorf("snapshot must pin model, questions, knowledge, thresholds and app build")
	}
	if keyHash == "" || actual.KeySHA256 != keyHash {
		return fmt.Errorf("snapshot key hash mismatch")
	}
	if len(corpus) == 0 || len(actual.Corpus) != len(corpus) {
		return fmt.Errorf("snapshot corpus differs")
	}
	for id, hash := range corpus {
		if actual.Corpus[id] != hash {
			return fmt.Errorf("snapshot corpus hash mismatch for %s", id)
		}
	}
	return nil
}

type WorkVersions struct {
	Model             string `json:"model"`
	QuestionVersion   string `json:"question_version"`
	KnowledgeVersion  string `json:"knowledge_version"`
	ThresholdsVersion string `json:"thresholds_version"`
	AppBuild          string `json:"app_build"`
}

func VerifyWorkVersions(actual WorkSnapshot, expected WorkVersions) error {
	if expected.Model == "" || expected.QuestionVersion == "" || expected.KnowledgeVersion == "" || expected.ThresholdsVersion == "" || expected.AppBuild == "" {
		return fmt.Errorf("expected version pins must all be nonempty")
	}
	got := WorkVersions{actual.Model, actual.QuestionVersion, actual.KnowledgeVersion, actual.ThresholdsVersion, actual.AppBuild}
	if got != expected {
		return fmt.Errorf("snapshot versions differ from expected run pins")
	}
	return nil
}

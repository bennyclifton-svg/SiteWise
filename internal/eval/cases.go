package eval

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"gopkg.in/yaml.v3"

	"sitewise/internal/config"
	"sitewise/internal/intake"
)

// Tiers separate labels transcribed from an authoritative document (a
// register or transmittal) from labels a model or a folder name produced.
// Only authoritative labels fit thresholds or gate a release.
const (
	TierAuthoritative = "authoritative"
	TierDiagnostic    = "diagnostic"
)

// Manifest is the committed evaluation contract. It pins answer keys by hash
// and names the corpus by location; it holds no private labels or text.
type Manifest struct {
	SchemaVersion        int                  `json:"schema_version"`
	QuestionVersion      string               `json:"question_version"`
	Model                string               `json:"model"`
	Privacy              string               `json:"privacy"`
	CorpusRoot           string               `json:"corpus_root"`
	KeysRoot             string               `json:"keys_root"`
	PrivateDir           string               `json:"private_dir"`
	Corpora              []Corpus             `json:"corpora"`
	AuthoritativeSources []string             `json:"authoritative_label_sources"`
	AuthoritativeFields  []string             `json:"authoritative_fields"`
	DiagnosticFields     map[string]string    `json:"diagnostic_fields"`
	KindMap              map[string]string    `json:"kind_map"`
	Flags                map[string][]string  `json:"flags"`
	Split                SplitPolicy          `json:"split"`
	Fit                  map[string]FitPolicy `json:"fit"`
	Gate                 GatePolicy           `json:"gate"`
	Baseline             string               `json:"baseline"`
}

// Corpus is one answer key and the folder its corpus_path values resolve in.
type Corpus struct {
	Name      string `json:"name"`
	Key       string `json:"key"`
	KeySHA256 string `json:"key_sha256"`
	Root      string `json:"root"`
}

// SplitPolicy assigns families to calibration by salted hash.
type SplitPolicy struct {
	Salt               string `json:"salt"`
	CalibrationPercent int    `json:"calibration_percent"`
}

// GatePolicy is what the held-out run must show.
type GatePolicy struct {
	Z                        float64        `json:"z"`
	MinCases                 int            `json:"min_cases"`
	MinHeldout               map[string]int `json:"min_heldout"`
	MaxIncorrectSupersession int            `json:"max_incorrect_supersession"`
}

// LoadManifest reads and checks the manifest.
func LoadManifest(file string) (Manifest, []byte, error) {
	body, err := os.ReadFile(file)
	if err != nil {
		return Manifest{}, nil, err
	}
	var m Manifest
	dec := json.NewDecoder(strings.NewReader(string(body)))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&m); err != nil {
		return Manifest{}, nil, fmt.Errorf("manifest: %w", err)
	}
	if err := m.check(); err != nil {
		return Manifest{}, nil, err
	}
	return m, body, nil
}

func (m Manifest) check() error {
	switch {
	case m.SchemaVersion != 1:
		return errors.New("manifest schema_version must be 1")
	case m.QuestionVersion != intake.QuestionVersion:
		return fmt.Errorf("manifest question version %q, intake asks %q", m.QuestionVersion, intake.QuestionVersion)
	case m.Model != config.PinnedJevModel:
		return fmt.Errorf("manifest model %q is not the pinned %s", m.Model, config.PinnedJevModel)
	case len(m.Corpora) == 0:
		return errors.New("manifest has no corpora")
	case m.Split.CalibrationPercent <= 0 || m.Split.CalibrationPercent >= 100:
		return errors.New("calibration_percent must leave both splits non-empty")
	case m.Gate.Z <= 0:
		return errors.New("gate z must be positive")
	}
	for _, c := range m.Corpora {
		if c.Name == "" || c.Key == "" || !isHex64(c.KeySHA256) {
			return fmt.Errorf("corpus %q needs a name, key and key_sha256", c.Name)
		}
	}
	return nil
}

// Label is one expected value for a field.
type Label struct {
	Value         string
	Authoritative bool
	Source        string
}

// Case is one file a user would drop, with the labels scored for it.
type Case struct {
	ID       string
	Corpus   string
	RelPath  string
	Path     string
	SHA256   string
	Format   string
	Family   string
	Split    string
	Labels   map[string]Label
	Flags    []string
	Adjudged string
}

// Filename is the name the file would be dropped with.
func (c Case) Filename() string { return path.Base(c.RelPath) }

// CaseSet is the verified case list and what was left out, by reason.
type CaseSet struct {
	Cases    []Case
	Excluded map[string]int
	Keys     map[string]string
}

type keyFile struct {
	SchemaVersion int        `yaml:"schema_version"`
	Corpus        string     `yaml:"corpus"`
	LabelSource   string     `yaml:"label_source"`
	Adjudication  string     `yaml:"adjudication"`
	Entries       []keyEntry `yaml:"entries"`
}

type keyEntry struct {
	ID          string             `yaml:"id"`
	CorpusPath  string             `yaml:"corpus_path"`
	Page        *int               `yaml:"page"`
	DocType     string             `yaml:"doc_type"`
	Expected    map[string]*string `yaml:"expected"`
	LabelSource string             `yaml:"label_source"`
	Flags       []string           `yaml:"flags"`
	SHA256      string             `yaml:"sha256"`
}

var knownLabelSources = map[string]bool{"award-doc": true, "owner": true, "model": true, "folder": true, "synthetic": true}

var expectedFields = map[string]string{
	"category": intake.FieldDiscipline,
	"number":   intake.FieldNumber,
	"revision": intake.FieldRevision,
	"title":    intake.FieldTitle,
	"date":     intake.FieldDate,
}

// BuildCases reads every answer key, checks its pinned hash and quality, and
// verifies each corpus file's hash. Any key or corpus problem is an error:
// a missing corpus fails the run, it is not skipped.
func BuildCases(m Manifest) (CaseSet, error) {
	set := CaseSet{Excluded: map[string]int{}, Keys: map[string]string{}}
	var problems []string
	seenIDs := map[string]bool{}
	for _, corpus := range m.Corpora {
		keyPath := filepath.Join(m.KeysRoot, corpus.Key)
		body, err := os.ReadFile(keyPath)
		if err != nil {
			problems = append(problems, fmt.Sprintf("answer key %s: %v", corpus.Key, err))
			continue
		}
		if got := hexSHA(body); got != corpus.KeySHA256 {
			problems = append(problems, fmt.Sprintf("answer key %s sha256 %s, manifest pins %s", corpus.Key, got, corpus.KeySHA256))
			continue
		}
		set.Keys[corpus.Key] = corpus.KeySHA256
		var key keyFile
		if err := yaml.Unmarshal(body, &key); err != nil {
			problems = append(problems, fmt.Sprintf("answer key %s: %v", corpus.Key, err))
			continue
		}
		cases, errs := casesFrom(m, corpus, key, seenIDs, set.Excluded)
		problems = append(problems, errs...)
		set.Cases = append(set.Cases, cases...)
	}
	if len(problems) > 0 {
		return CaseSet{}, errors.New(strings.Join(problems, "\n"))
	}
	assignSplits(m, set.Cases)
	sort.Slice(set.Cases, func(i, j int) bool {
		if set.Cases[i].Corpus != set.Cases[j].Corpus {
			return set.Cases[i].Corpus < set.Cases[j].Corpus
		}
		return set.Cases[i].RelPath < set.Cases[j].RelPath
	})
	return set, nil
}

func casesFrom(m Manifest, corpus Corpus, key keyFile, seenIDs map[string]bool, excluded map[string]int) ([]Case, []string) {
	var problems []string
	bad := func(format string, a ...any) {
		problems = append(problems, fmt.Sprintf("answer key %s: ", corpus.Key)+fmt.Sprintf(format, a...))
	}
	if key.SchemaVersion != 1 {
		bad("schema_version %d", key.SchemaVersion)
		return nil, problems
	}
	if key.Corpus != corpus.Name {
		bad("corpus %q, manifest names %q", key.Corpus, corpus.Name)
	}
	authoritative := toSet(m.AuthoritativeSources)
	authFields := toSet(m.AuthoritativeFields)
	bySHA := map[string]bool{}
	var out []Case
	for _, e := range key.Entries {
		if e.ID == "" || seenIDs[e.ID] {
			bad("duplicate or empty id %q", e.ID)
			continue
		}
		seenIDs[e.ID] = true
		if !isHex64(e.SHA256) {
			bad("%s: sha256 is not 64 hex characters", e.ID)
			continue
		}
		if !safeRel(e.CorpusPath) {
			bad("%s: corpus_path %q escapes the corpus", e.ID, e.CorpusPath)
			continue
		}
		source := e.LabelSource
		if source == "" {
			source = key.LabelSource
		}
		if !knownLabelSources[source] {
			bad("%s: label_source %q", e.ID, source)
			continue
		}
		dropped := map[string]bool{}
		for _, f := range e.Flags {
			fields, ok := m.Flags[f]
			if !ok {
				bad("%s: flag %s is not classified in the manifest", e.ID, f)
				continue
			}
			for _, field := range fields {
				dropped[field] = true
			}
		}
		for k := range e.Expected {
			if _, ok := expectedFields[k]; !ok {
				bad("%s: unknown expected field %q", e.ID, k)
			}
		}
		// Intake reads the first page only. A later page of a multi-sheet
		// file is not that file's identity.
		if e.Page != nil && *e.Page != 1 {
			excluded["later_page"]++
			continue
		}
		if bySHA[e.SHA256] {
			excluded["duplicate_file"]++
			continue
		}
		bySHA[e.SHA256] = true

		labels := map[string]Label{}
		for k, v := range e.Expected {
			field := expectedFields[k]
			if field == "" || v == nil || strings.TrimSpace(*v) == "" || dropped[field] {
				continue
			}
			labels[field] = Label{Value: strings.TrimSpace(*v), Source: source, Authoritative: authoritative[source] && authFields[field]}
		}
		if kind, ok := m.KindMap[e.DocType]; ok && kind != "" && !dropped[intake.FieldKind] {
			labels[intake.FieldKind] = Label{Value: kind, Source: "doc_type", Authoritative: false}
		}

		full := filepath.Join(m.CorpusRoot, filepath.FromSlash(corpus.Root), filepath.FromSlash(e.CorpusPath))
		got, err := fileSHA(full)
		if err != nil {
			bad("%s: corpus file %s: %v", e.ID, e.CorpusPath, err)
			continue
		}
		if got != e.SHA256 {
			bad("%s: corpus file %s sha256 %s, key pins %s", e.ID, e.CorpusPath, got, e.SHA256)
			continue
		}
		format := formatOf(e.CorpusPath)
		if format == "" {
			excluded["unsupported_format"]++
			continue
		}
		out = append(out, Case{
			ID:       e.ID,
			Corpus:   corpus.Name,
			RelPath:  e.CorpusPath,
			Path:     full,
			SHA256:   e.SHA256,
			Format:   format,
			Labels:   labels,
			Flags:    e.Flags,
			Adjudged: key.Adjudication,
		})
	}
	return out, problems
}

// assignSplits keeps a directory and a document number series in one family,
// so near-duplicate revisions and shared title-block templates cannot leak
// from calibration into held-out.
func assignSplits(m Manifest, cases []Case) {
	members := make([]Member, len(cases))
	for i, c := range cases {
		keys := []string{"dir:" + c.Corpus + "/" + path.Dir(c.RelPath)}
		if l, ok := c.Labels[intake.FieldNumber]; ok {
			if n, ok := intake.Normalize(intake.FieldNumber, l.Value); ok {
				keys = append(keys, "number:"+c.Corpus+"/"+n)
			}
		}
		members[i] = Member{ID: c.ID, Keys: keys}
	}
	families := Families(members)
	for i := range cases {
		cases[i].Family = families[cases[i].ID]
		cases[i].Split = SplitOf(cases[i].Family, m.Split.Salt, m.Split.CalibrationPercent)
	}
}

func formatOf(name string) string {
	switch strings.ToLower(path.Ext(name)) {
	case ".pdf":
		return "pdf"
	case ".docx":
		return "docx"
	case ".xlsx":
		return "xlsx"
	default:
		return ""
	}
}

var hex64 = regexp.MustCompile(`^[0-9a-f]{64}$`)

func isHex64(s string) bool { return hex64.MatchString(s) }

func safeRel(p string) bool {
	if p == "" || strings.HasPrefix(p, "/") || filepath.IsAbs(p) {
		return false
	}
	for _, part := range strings.Split(p, "/") {
		if part == ".." {
			return false
		}
	}
	return true
}

func toSet(list []string) map[string]bool {
	out := make(map[string]bool, len(list))
	for _, s := range list {
		out[s] = true
	}
	return out
}

func hexSHA(b []byte) string {
	s := sha256.Sum256(b)
	return hex.EncodeToString(s[:])
}

func fileSHA(p string) (string, error) {
	f, err := os.Open(p)
	if err != nil {
		return "", err
	}
	defer f.Close()
	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", err
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}

// SHA256Hex is the lowercase hex digest of b.
func SHA256Hex(b []byte) string { return hexSHA(b) }

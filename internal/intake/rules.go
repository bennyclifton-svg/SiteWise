package intake

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

const (
	RuleMissing   = "missing"
	RuleUnique    = "unique"
	RuleAmbiguous = "ambiguous"
)

// Result is a deterministic field decision. Settled is true only when every
// harvested value for the field compares equal. Display is one source literal.
type Result struct {
	Field      string
	Settled    bool
	Rule       string
	Display    string
	Normalized string
}

// Decide settles number, revision, title, and date when the harvested values
// are missing or all the same. It does not pick among disagreements and it
// does not invent a revision.
func Decide(candidates []Candidate) []Result {
	fields := []string{FieldNumber, FieldRevision, FieldTitle, FieldDate}
	out := make([]Result, len(fields))
	for i, field := range fields {
		out[i] = decideField(field, candidates)
	}
	return out
}

func decideField(field string, candidates []Candidate) Result {
	var group []Candidate
	seen := make(map[string]struct{})
	distinct := 0
	for _, c := range candidates {
		if c.Field != field || c.Normalized == "" {
			continue
		}
		group = append(group, c)
		if _, ok := seen[c.Normalized]; ok {
			continue
		}
		seen[c.Normalized] = struct{}{}
		distinct++
	}
	if distinct == 0 {
		return Result{Field: field, Rule: RuleMissing}
	}
	if distinct > 1 {
		return Result{Field: field, Rule: RuleAmbiguous}
	}
	chosen := prefer(group)
	return Result{
		Field:      field,
		Settled:    true,
		Rule:       RuleUnique,
		Display:    chosen.Display,
		Normalized: chosen.Normalized,
	}
}

// prefer keeps the sheet's own labeled text when it agrees with the filename.
func prefer(group []Candidate) Candidate {
	best := group[0]
	bestScore := preference(best)
	for _, c := range group[1:] {
		if score := preference(c); score > bestScore {
			best = c
			bestScore = score
		}
	}
	return best
}

func preference(c Candidate) int {
	score := 0
	if c.Provenance.Origin == OriginText && c.Provenance.Labeled {
		score += 4
	}
	if c.Provenance.Origin == OriginFilename {
		score += 2
	}
	if c.Provenance.Labeled {
		score += 1
	}
	return score
}

// Discipline is a filing view copied from the reference taxonomy.
type Discipline struct {
	ID      string   `json:"id"`
	Label   string   `json:"label"`
	Aliases []string `json:"aliases"`
}

// Kind is one document kind from the reference vocabulary.
type Kind struct {
	ID    string `json:"id"`
	Label string `json:"label"`
}

// LifecycleArea is one reference-corpus folder area.
type LifecycleArea struct {
	ID     string `json:"id"`
	Label  string `json:"label"`
	Folder string `json:"folder"`
}

// Catalog is the closed filing vocabulary. KindListComplete is false while the
// foundation design's 16 kinds are not all present in the reference data.
type Catalog struct {
	Disciplines             []Discipline
	DisciplineSourceSHA256  string
	Kinds                   []Kind
	KindDesignCount         int
	KindListComplete        bool
	KindReconciliation      string
	Lifecycle               []LifecycleArea
	LifecycleReconciliation string
}

type disciplineFile struct {
	SourceSHA256 string       `json:"source_sha256"`
	Disciplines  []Discipline `json:"disciplines"`
}

type kindFile struct {
	DesignCount    int    `json:"design_count"`
	Complete       bool   `json:"complete"`
	Reconciliation string `json:"reconciliation"`
	Kinds          []Kind `json:"kinds"`
}

type lifecycleFile struct {
	Reconciliation string          `json:"reconciliation"`
	Areas          []LifecycleArea `json:"areas"`
}

// LoadCatalog reads the intake vocabulary. It rejects a discipline list that
// is no longer the copied 57, and a kind list that claims to be complete
// without the design count.
func LoadCatalog(dir string) (Catalog, error) {
	var cat Catalog
	var disciplines disciplineFile
	if err := readJSON(filepath.Join(dir, "disciplines.json"), &disciplines); err != nil {
		return Catalog{}, err
	}
	if err := validateDisciplines(disciplines); err != nil {
		return Catalog{}, err
	}
	var kinds kindFile
	if err := readJSON(filepath.Join(dir, "kinds.json"), &kinds); err != nil {
		return Catalog{}, err
	}
	if err := validateKinds(kinds); err != nil {
		return Catalog{}, err
	}
	var life lifecycleFile
	if err := readJSON(filepath.Join(dir, "lifecycle.json"), &life); err != nil {
		return Catalog{}, err
	}
	if err := validateLifecycle(life); err != nil {
		return Catalog{}, err
	}
	cat.Disciplines = disciplines.Disciplines
	cat.DisciplineSourceSHA256 = disciplines.SourceSHA256
	cat.Kinds = kinds.Kinds
	cat.KindDesignCount = kinds.DesignCount
	cat.KindListComplete = kinds.Complete
	cat.KindReconciliation = kinds.Reconciliation
	cat.Lifecycle = life.Areas
	cat.LifecycleReconciliation = life.Reconciliation
	return cat, nil
}

func readJSON(path string, dest any) error {
	body, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("read %s: %w", filepath.Base(path), err)
	}
	if err := json.Unmarshal(body, dest); err != nil {
		return fmt.Errorf("parse %s: %w", filepath.Base(path), err)
	}
	return nil
}

func validateDisciplines(file disciplineFile) error {
	if len(file.SourceSHA256) != 64 {
		return fmt.Errorf("discipline source hash is missing")
	}
	if len(file.Disciplines) != 57 {
		return fmt.Errorf("discipline count %d", len(file.Disciplines))
	}
	seen := make(map[string]struct{}, len(file.Disciplines))
	for _, d := range file.Disciplines {
		if d.ID == "" || d.Label == "" {
			return fmt.Errorf("discipline missing id or label")
		}
		if _, ok := seen[d.ID]; ok {
			return fmt.Errorf("duplicate discipline %s", d.ID)
		}
		seen[d.ID] = struct{}{}
	}
	return nil
}

func validateKinds(file kindFile) error {
	if file.Reconciliation == "" {
		return fmt.Errorf("kind reconciliation is missing")
	}
	if file.DesignCount != 16 {
		return fmt.Errorf("kind design count %d", file.DesignCount)
	}
	if len(file.Kinds) == 0 {
		return fmt.Errorf("kind list is empty")
	}
	if file.Complete && len(file.Kinds) != file.DesignCount {
		return fmt.Errorf("kind list claims %d labels and holds %d", file.DesignCount, len(file.Kinds))
	}
	seen := make(map[string]struct{}, len(file.Kinds))
	for _, k := range file.Kinds {
		if k.ID == "" || k.Label == "" {
			return fmt.Errorf("kind missing id or label")
		}
		if _, ok := seen[k.ID]; ok {
			return fmt.Errorf("duplicate kind %s", k.ID)
		}
		seen[k.ID] = struct{}{}
	}
	return nil
}

func validateLifecycle(file lifecycleFile) error {
	if file.Reconciliation == "" {
		return fmt.Errorf("lifecycle reconciliation is missing")
	}
	if len(file.Areas) == 0 {
		return fmt.Errorf("lifecycle list is empty")
	}
	seen := make(map[string]struct{}, len(file.Areas))
	for _, area := range file.Areas {
		if area.ID == "" || area.Label == "" {
			return fmt.Errorf("lifecycle area missing id or label")
		}
		if _, ok := seen[area.ID]; ok {
			return fmt.Errorf("duplicate lifecycle area %s", area.ID)
		}
		seen[area.ID] = struct{}{}
	}
	return nil
}

package intake

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const (
	RuleMissing      = "missing"
	RuleUnique       = "unique"
	RuleAmbiguous    = "ambiguous"
	RuleCorroborated = "filename_and_label"
	RuleOwnTitle     = "explicit_sheet_title"
	RuleIssueDate    = "explicit_revision_date"
)

// Result is a deterministic field decision. Settled requires agreement or a
// unique filename number corroborated by an explicit label. Display is literal.
type Result struct {
	Field      string
	Settled    bool
	Rule       string
	Display    string
	Normalized string
}

// Decide settles matching values and corroborated sheet numbers. Other
// disagreements stay open; missing evidence never invents a revision.
func Decide(candidates []Candidate) []Result {
	fields := []string{FieldNumber, FieldRevision, FieldTitle, FieldDate}
	out := make([]Result, len(fields))
	for i, field := range fields {
		out[i] = decideField(field, candidates)
	}
	return out
}

func decideField(field string, candidates []Candidate) Result {
	// Candidate provenance is large. Count first to avoid repeatedly copying
	// and reallocating the field group on dense drawing sheets.
	count := 0
	for _, c := range candidates {
		if c.Field == field && c.Normalized != "" {
			count++
		}
	}
	group := make([]Candidate, 0, count)
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
	// A filename can name the job or document type without identifying this
	// document. Keep bare job codes and short acronyms open without text evidence.
	if distinct == 1 && (field == FieldNumber || field == FieldTitle) {
		filenameOnly := true
		for _, c := range group {
			filenameOnly = filenameOnly && c.Provenance.Origin == OriginFilename
		}
		value := group[0].Display
		bareJob := field == FieldNumber && digitsOnly(value) && (len(value) == 5 || len(value) == 6)
		acronym := field == FieldTitle && len(value) <= 3 && lettersASCII(value) && value == strings.ToUpper(value)
		if filenameOnly && (bareJob || acronym) {
			return Result{Field: field, Rule: RuleAmbiguous}
		}
	}
	// Two-digit filename dates can be YY-MM-DD or DD-MM-YY, or export dates.
	// Document evidence can corroborate the literal; the filename alone cannot.
	if field == FieldDate && distinct == 1 {
		filenameOnly, shortNumeric := true, false
		for _, c := range group {
			filenameOnly = filenameOnly && c.Provenance.Origin == OriginFilename
			parts := strings.FieldsFunc(c.Display, func(r rune) bool { return r == '-' || r == '.' || r == '/' })
			shortNumeric = len(parts) == 3 && len(parts[0]) <= 2 && len(parts[1]) <= 2 && len(parts[2]) == 2
		}
		if filenameOnly && shortNumeric {
			return Result{Field: field, Rule: RuleAmbiguous}
		}
	}
	if field == FieldRevision {
		historyOnly := true
		for _, c := range group {
			historyOnly = historyOnly && c.Provenance.IssueTable
		}
		if historyOnly {
			cover := map[string]Candidate{}
			for _, c := range candidates {
				if c.Field == FieldDate && c.Provenance.Page == 1 && !c.Provenance.IssueTable {
					cover[c.Normalized] = c
				}
			}
			matches := map[string]Candidate{}
			if len(cover) == 1 {
				for _, date := range cover {
					for _, row := range candidates {
						if row.Field != FieldDate || !row.Provenance.IssueTable || !sameCalendarDate([]Candidate{date, row}) {
							continue
						}
						for _, revision := range group {
							if revision.Normalized == row.Provenance.IssueRevision {
								matches[revision.Normalized] = revision
							}
						}
					}
				}
			}
			if len(matches) == 1 {
				for _, c := range matches {
					return Result{Field: field, Settled: true, Rule: "cover_date_and_revision_row", Display: c.Display, Normalized: c.Normalized}
				}
			}
			return Result{Field: field, Rule: RuleAmbiguous}
		}
	}
	if distinct > 1 {
		if field == FieldNumber {
			own := map[string]Candidate{}
			for _, c := range group {
				if c.Provenance.OwnNumber {
					own[c.Normalized] = c
				}
			}
			if len(own) == 1 {
				for _, c := range own {
					return Result{Field: field, Settled: true, Rule: "explicit_number_cell", Display: c.Display, Normalized: c.Normalized}
				}
			}
		}
		if field == FieldRevision {
			current := map[string]Candidate{}
			for _, c := range group {
				if c.Provenance.OwnRevision || c.Provenance.Origin == OriginFilename {
					current[c.Normalized] = c
				}
			}
			if len(current) == 1 {
				for _, c := range current {
					if c.Provenance.OwnRevision {
						return Result{Field: field, Settled: true, Rule: "explicit_revision_cell", Display: c.Display, Normalized: c.Normalized}
					}
					if c.Provenance.Origin == OriginFilename {
						dates := map[string]bool{}
						for _, candidate := range candidates {
							if candidate.Field == FieldDate && candidate.Provenance.IssueTable && candidate.Provenance.IssueRevision == c.Normalized {
								dates[candidate.Normalized] = true
							}
						}
						if len(dates) == 1 {
							return Result{Field: field, Settled: true, Rule: "filename_and_issue_row", Display: c.Display, Normalized: c.Normalized}
						}
					}
				}
			}
		}
		if field == FieldDate {
			if cover, ok := coverDateWithHistory(group); ok {
				return Result{Field: field, Settled: true, Rule: "cover_date_and_history", Display: cover.Display, Normalized: cover.Normalized}
			}
			if sameCalendarDate(group) {
				c := prefer(group)
				return Result{Field: field, Settled: true, Rule: "equivalent_calendar_date", Display: c.Display, Normalized: c.Normalized}
			}
			revision := decideField(FieldRevision, candidates)
			if revision.Settled {
				// Draft and final histories can reuse an issue letter. Preserve
				// the cover date that identified its row, not another row's date.
				if revision.Rule == "cover_date_and_revision_row" {
					for _, c := range group {
						if c.Provenance.Page == 1 && !c.Provenance.IssueTable {
							return Result{Field: field, Settled: true, Rule: "cover_date_and_revision_row", Display: c.Display, Normalized: c.Normalized}
						}
					}
				}
				paired := map[string]Candidate{}
				explicit := false
				for _, c := range group {
					if c.Provenance.IssueRevision == revision.Normalized && !c.Provenance.IssueTable {
						explicit = true
					}
				}
				for _, c := range group {
					if c.Provenance.IssueRevision == revision.Normalized && (!explicit || !c.Provenance.IssueTable) {
						paired[c.Normalized] = c
					}
				}
				if len(paired) == 1 {
					for _, c := range paired {
						return Result{Field: field, Settled: true, Rule: RuleIssueDate, Display: c.Display, Normalized: c.Normalized}
					}
				}
			}
		}
		if field == FieldTitle {
			// A literal title in document control corroborated by the cover
			// heading is stronger identity evidence than an abbreviated filename.
			controls, headings := map[string]Candidate{}, map[string]bool{}
			for _, c := range group {
				if c.Provenance.ControlTitle {
					controls[c.Normalized] = c
				}
				if c.Provenance.Heading {
					headings[c.Normalized] = true
				}
			}
			if len(controls) == 1 {
				for key, c := range controls {
					if headings[key] {
						return Result{Field: field, Settled: true, Rule: "document_control_and_heading", Display: c.Display, Normalized: c.Normalized}
					}
				}
			}
			// A complete title block identifies the first sheet of a drawing pack;
			// the pack's filename can legitimately describe the whole collection.
			ownNumber, ownRevision := false, false
			for _, c := range candidates {
				ownNumber = ownNumber || c.Provenance.OwnNumber
				ownRevision = ownRevision || c.Provenance.OwnRevision
			}
			if own, ok := explicitOwnTitle(group, ownNumber && ownRevision); ok {
				return Result{Field: field, Settled: true, Rule: RuleOwnTitle, Display: own.Display, Normalized: own.Normalized}
			}
		}
		if field == FieldNumber || field == FieldTitle {
			if own, ok := corroboratedIdentity(group); ok {
				return Result{Field: field, Settled: true, Rule: RuleCorroborated, Display: own.Display, Normalized: own.Normalized}
			}
		}
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

// Reading a report's history must not replace its corroborated cover date
// with an older issue date. An unrelated date outside that history stays open.
func coverDateWithHistory(group []Candidate) (Candidate, bool) {
	covers := map[string]Candidate{}
	history := map[string]Candidate{}
	for _, c := range group {
		if c.Provenance.IssueTable {
			history[c.Normalized] = c
		} else if c.Provenance.Page == 1 && c.Provenance.Origin == OriginText {
			covers[c.Normalized] = c
		}
	}
	if len(covers) != 1 || len(history) == 0 {
		return Candidate{}, false
	}
	var cover Candidate
	for _, c := range covers {
		cover = c
	}
	matched := false
	for _, c := range history {
		matched = matched || sameCalendarDate([]Candidate{cover, c})
	}
	if !matched {
		return Candidate{}, false
	}
	for _, c := range group {
		if c.Normalized == cover.Normalized {
			continue
		}
		if _, ok := history[c.Normalized]; !ok {
			return Candidate{}, false
		}
	}
	return cover, true
}

// Compare complete dates only. An abbreviated year or incomplete month tag
// remains literal rather than silently choosing a century or inventing a day.
func sameCalendarDate(group []Candidate) bool {
	var first time.Time
	for _, c := range group {
		var parsed time.Time
		valueText := strings.TrimSpace(c.Display)
		// Ordinal day suffixes change presentation, not the calendar day.
		// Keep the displayed source literal and still require a full year.
		parts := strings.Fields(valueText)
		if len(parts) == 3 && isDay(parts[0]) && isMonth(parts[1]) && len(parts[2]) == 4 {
			day := parts[0]
			for _, suffix := range []string{"st", "nd", "rd", "th"} {
				if strings.HasSuffix(strings.ToLower(day), suffix) {
					day = day[:len(day)-2]
					break
				}
			}
			valueText = day + " " + strings.TrimRight(parts[1], ",.") + " " + parts[2]
		}
		for _, layout := range []string{"2006-01-02", "2/1/2006", "2.1.2006", "2-1-2006", "2 Jan 2006", "2 January 2006", "January 2, 2006", "Jan 2, 2006"} {
			if value, err := time.Parse(layout, valueText); err == nil {
				parsed = value
				break
			}
		}
		if parsed.IsZero() || (!first.IsZero() && !first.Equal(parsed)) {
			return false
		}
		first = parsed
	}
	return !first.IsZero()
}

// With no filename title to contradict it, one explicit sheet-title value
// resolves incidental unlabelled text. Generic Project Title is not eligible.
func explicitOwnTitle(group []Candidate, completeBlock bool) (Candidate, bool) {
	// The physical title block precedes abbreviated regulated-design records.
	// Require its independently bound number and current revision too.
	if completeBlock {
		block := map[string]Candidate{}
		for _, c := range group {
			if c.Provenance.OwnTitle && c.Provenance.BlockTitle {
				block[c.Normalized] = c
			}
		}
		if len(block) == 1 {
			for _, c := range block {
				return c, true
			}
		}
	}
	values := map[string]Candidate{}
	inline := false
	for _, c := range group {
		if c.Provenance.OwnTitle && !c.Provenance.TitleBelow {
			inline = true
		}
	}
	for _, c := range group {
		if c.Provenance.Origin == OriginFilename && !c.Provenance.ExportName && !completeBlock {
			return Candidate{}, false
		}
		if (c.Provenance.OwnTitle || (completeBlock && c.Provenance.BlockTitle)) && (!inline || !c.Provenance.TitleBelow) {
			values[c.Normalized] = c
		}
	}
	if len(values) == 1 {
		for _, c := range values {
			return c, true
		}
	}
	return Candidate{}, false
}

// A unique labelled number agreeing with a filename candidate resolves job
// prefixes and unrelated references. Titles require one filename value.
// Conflicting labels, or filename-only evidence, still require Jev.
func corroboratedIdentity(group []Candidate) (Candidate, bool) {
	files := map[string]bool{}
	labels := map[string]Candidate{}
	for _, c := range group {
		if c.Provenance.Origin == OriginFilename {
			files[c.Normalized] = true
		}
		if c.Provenance.Origin == OriginText && c.Provenance.Labeled {
			labels[c.Normalized] = c
		}
	}
	if len(files) == 0 || len(labels) != 1 || (group[0].Field != FieldNumber && len(files) != 1) {
		return Candidate{}, false
	}
	for n, c := range labels {
		return c, files[n]
	}
	return Candidate{}, false
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

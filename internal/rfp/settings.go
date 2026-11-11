package rfp

import (
	"encoding/json"
	"errors"
	"regexp"
	"strings"
	"time"
	"unicode/utf8"

	"sitewise/internal/knowledge"
	"sitewise/internal/reports"
)

var ErrInvalidSetting = errors.New("invalid RFP setting")

// factLabels name facts in plain English for checks and controls.
var factLabels = map[string]string{
	"class": "Building class", "category": "Building category", "work_type": "Work type",
	"pathway": "Planning pathway", "procurement": "Procurement route", "novation": "Novation",
	"state": "State", "regulated_design": "Regulated design applies", "budget": "Construction budget",
	"site_area": "Site area", "gfa": "Gross floor area", "storeys": "Storeys", "parking": "Car spaces",
	"system": "Building systems in Profile scope",
}

func factLabel(sk knowledge.RFPSkeleton, key string) string {
	if flag, ok := strings.CutPrefix(key, "flag."); ok {
		return flagLabel(sk, flag)
	}
	if l, ok := factLabels[key]; ok {
		return l
	}
	return strings.ReplaceAll(key, "_", " ")
}

func flagLabel(sk knowledge.RFPSkeleton, key string) string {
	for _, f := range sk.Flags {
		if f.Key == key {
			return f.Label
		}
	}
	return key
}

func valueLabel(sk knowledge.RFPSkeleton, fact, value string) string {
	var list []knowledge.RFPOption
	switch fact {
	case "pathway":
		list = sk.Pathways
	case "procurement":
		list = sk.Procurement
	case "work_type":
		list = sk.WorkTypes
	}
	for _, o := range list {
		if o.Key == value {
			return o.Label
		}
	}
	return value
}

func slotLabel(sk knowledge.RFPSkeleton, name string) string {
	if key, ok := strings.CutPrefix(name, "field:"); ok {
		for _, f := range sk.Fields {
			if f.Key == key {
				return f.Label
			}
		}
	}
	if key, ok := strings.CutPrefix(name, "allowance:"); ok {
		return "allowance " + strings.ReplaceAll(key, "_", " ")
	}
	switch name {
	case "client":
		return "Client"
	case "address":
		return "Site address"
	case "work_phrase":
		return "Work type"
	case "parts_phrase", "parts":
		return "Building class"
	case "pathway", "planning_report", "planning_report_long":
		return "Planning pathway"
	case "procurement":
		return "Procurement"
	case "information":
		return "Information provided"
	case "quantities":
		return "Quantities"
	}
	return strings.ReplaceAll(name, "_", " ")
}

// validKey reports whether a setting key names something in the current
// discipline or skeleton. Other keys are kept as orphans, never dropped.
func validKey(d knowledge.RFPDiscipline, sk knowledge.RFPSkeleton, drivers DriverSet, key string) bool {
	prefix, name, _ := strings.Cut(key, ":")
	switch key {
	case "scale", "boundary:range", "parts:affected", "transmittal", AgreementDocs:
		return true
	}
	switch prefix {
	case "stage":
		for _, s := range d.Stages {
			if s.Key == name {
				return true
			}
		}
	case "element":
		for _, e := range d.Elements {
			if e.Key == name {
				return true
			}
		}
	case "allowance":
		for _, a := range d.Allowances {
			if a.Key == name {
				return true
			}
		}
	case "position", "responsible":
		for _, p := range d.Positions {
			if p.Key == name {
				return true
			}
		}
	case "field":
		for _, f := range sk.Fields {
			if f.Key == name {
				return f.Profile == ""
			}
		}
	case "weight":
		for _, w := range evaluationWeights(d, sk) {
			if w.Key == name {
				return true
			}
		}
	case "line":
		stage, line, _ := strings.Cut(name, ".")
		for _, s := range d.Stages {
			for _, l := range s.Lines {
				if d.ScopeByStage && s.Key == stage && l.Key == line {
					return true
				}
			}
		}
	case "term":
		return termKnown(d, sk, name)
	case "wording":
		// A service's wording is wording:position.<key>; a stage line's is
		// wording:<stage>.<line>. Both only where scope is by stage. A
		// term's is wording:terms.<key>.
		if term, ok := strings.CutPrefix(name, "terms."); ok {
			return termKnown(d, sk, term)
		}
		if service, ok := strings.CutPrefix(name, "position."); ok {
			return d.ScopeByStage && validKey(d, sk, drivers, "position:"+service)
		}
		return validKey(d, sk, drivers, "line:"+name)
	case "custom":
		if name == "terms" {
			return true
		}
		for _, s := range d.Stages {
			if d.ScopeByStage && s.Key == name {
				return true
			}
		}
	}
	return false
}

var (
	settingKeyRE = regexp.MustCompile(`^((stage|element|allowance|position|responsible|field|weight|line|wording|custom|term):[a-z0-9_.]{1,100}|scale|boundary:range|parts:affected)$`)
	customIDRE   = regexp.MustCompile(`^u[a-z0-9]{1,15}$`)
	dateRE       = regexp.MustCompile(`^\d{4}-\d{2}-\d{2}$`)
	numberRE     = regexp.MustCompile(`^[0-9]{1,12}(\.[0-9]{1,2})?$`)
)

// AgreementDocs is the setting that attaches the client agreement's
// documents, chosen from the repository as the transmittal's are.
const AgreementDocs = "agreement_docs"

// DocumentListSetting reports whether a setting holds repository documents:
// the store turns the ids a user chose into document rows before saving.
func DocumentListSetting(key string) bool {
	return key == "transmittal" || key == AgreementDocs
}

// ValidateSetting checks a key and its value for the discipline. A key that
// names nothing in the current rules is refused for writing; existing orphans
// can still be deleted.
func ValidateSetting(d knowledge.RFPDiscipline, sk knowledge.RFPSkeleton, drivers DriverSet, key string, raw json.RawMessage) error {
	if DocumentListSetting(key) {
		var docs []reports.BriefValue
		if len(raw) > 2000000 || json.Unmarshal(raw, &docs) != nil || docs == nil || len(docs) > 5000 {
			return ErrInvalidSetting
		}
		seen := map[string]bool{}
		for _, doc := range docs {
			if doc.ID == "" || seen[doc.ID] {
				return ErrInvalidSetting
			}
			seen[doc.ID] = true
		}
		return nil
	}
	if strings.HasPrefix(key, "custom:") {
		if !settingKeyRE.MatchString(key) || !validKey(d, sk, drivers, key) {
			return ErrInvalidSetting
		}
		if key == CustomTerms {
			return validCustomTerms(raw)
		}
		return validCustomLines(raw)
	}
	if !settingKeyRE.MatchString(key) || !validKey(d, sk, drivers, key) || len(raw) == 0 || len(raw) > 2000 || !json.Valid(raw) {
		return ErrInvalidSetting
	}
	prefix, name, _ := strings.Cut(key, ":")
	str := func() (string, bool) {
		var s string
		return s, json.Unmarshal(raw, &s) == nil
	}
	oneOf := func(values ...string) error {
		s, ok := str()
		if !ok || !contains(values, s) {
			return ErrInvalidSetting
		}
		return nil
	}
	switch {
	case key == "scale":
		return oneOf("simple", "complex")
	case key == "boundary:range":
		var b struct {
			Start string `json:"start"`
			End   string `json:"end"`
		}
		if json.Unmarshal(raw, &b) != nil {
			return ErrInvalidSetting
		}
		start, end := -1, -1
		for i, s := range d.Stages {
			if s.Key == b.Start {
				start = i
			}
			if s.Key == b.End {
				end = i
			}
		}
		if start < 0 || end < start {
			return ErrInvalidSetting
		}
		return nil
	case key == "parts:affected":
		var ids []string
		if json.Unmarshal(raw, &ids) != nil || len(ids) == 0 {
			return ErrInvalidSetting
		}
		for _, id := range ids {
			found := false
			for _, p := range drivers.Parts {
				found = found || p.ID == id
			}
			if !found {
				return ErrInvalidSetting
			}
		}
		return nil
	case prefix == "stage":
		return oneOf(Include, Option, Exclude, Done, Ask)
	case prefix == "element":
		return oneOf(Include, Option, Exclude, Ask)
	case prefix == "allowance":
		s, ok := str()
		if !ok || !countRE.MatchString(s) {
			return ErrInvalidSetting
		}
		return nil
	case prefix == "position" && d.ScopeByStage:
		// By stage, a service can also be deleted from this RFP.
		return oneOf("I", "O", "X", Ask, Deleted)
	case prefix == "position":
		return oneOf("I", "O", "X", Ask)
	case prefix == "line":
		return oneOf("I", "X", Deleted)
	case prefix == "term":
		return oneOf(Deleted)
	case prefix == "wording" && strings.HasPrefix(name, "terms."):
		s, ok := str()
		if !ok || !validTermText(s, termSlots(d, sk, strings.TrimPrefix(name, "terms."))) {
			return ErrInvalidSetting
		}
		return nil
	case prefix == "wording":
		s, ok := str()
		if !ok || !validLineText(s) {
			return ErrInvalidSetting
		}
		return nil
	case prefix == "responsible":
		s, ok := str()
		if !ok || strings.TrimSpace(s) == "" || utf8.RuneCountInString(s) > 120 || !printable(s) {
			return ErrInvalidSetting
		}
		return nil
	case prefix == "field":
		s, ok := str()
		if !ok || utf8.RuneCountInString(s) > 500 || !printable(s) {
			return ErrInvalidSetting
		}
		for _, f := range sk.Fields {
			if f.Key != name || s == "" {
				continue
			}
			switch f.Kind {
			case "date":
				if _, err := time.Parse("2006-01-02", s); err != nil || !dateRE.MatchString(s) {
					return ErrInvalidSetting
				}
			case "number":
				if !numberRE.MatchString(s) {
					return ErrInvalidSetting
				}
			}
		}
		return nil
	case prefix == "weight":
		var n int
		if json.Unmarshal(raw, &n) != nil || n < 0 || n > 100 {
			return ErrInvalidSetting
		}
		return nil
	}
	return ErrInvalidSetting
}

// MaxCustomLines bounds the lines a user adds to one stage.
const MaxCustomLines = 40

// validCustomLines checks a stage's own lines: each with a fresh id, wording
// of up to 300 characters that cannot be read as a {slot}, and I or X.
func validCustomLines(raw json.RawMessage) error {
	var lines []CustomLine
	if len(raw) == 0 || len(raw) > 20000 || json.Unmarshal(raw, &lines) != nil || lines == nil || len(lines) > MaxCustomLines {
		return ErrInvalidSetting
	}
	seen := map[string]bool{}
	for _, l := range lines {
		if !customIDRE.MatchString(l.ID) || seen[l.ID] || !validLineText(l.Text) || (l.Value != "I" && l.Value != "X") {
			return ErrInvalidSetting
		}
		seen[l.ID] = true
	}
	return nil
}

// CustomTerms holds the terms the user adds to an RFP, as custom:<stage>
// holds a stage's own lines.
const CustomTerms = "custom:terms"

// MaxCustomTerms bounds the terms a user adds to one RFP.
const MaxCustomTerms = 20

// TermSetting reports whether a setting edits the terms list itself: a
// term deleted, reworded or added.
func TermSetting(key string) bool {
	return strings.HasPrefix(key, "term:") || strings.HasPrefix(key, "wording:terms.") || key == CustomTerms
}

// termKnown reports whether the discipline or skeleton has a term at key.
func termKnown(d knowledge.RFPDiscipline, sk knowledge.RFPSkeleton, key string) bool {
	for _, rows := range [][]knowledge.RFPRow{sk.Terms, d.Terms} {
		for _, t := range rows {
			if t.Key == key {
				return true
			}
		}
	}
	return false
}

// termSlots are the field slots a term's standard wording holds; a
// reworded term may keep them and add no others.
func termSlots(d knowledge.RFPDiscipline, sk knowledge.RFPSkeleton, key string) map[string]bool {
	out := map[string]bool{}
	for _, rows := range [][]knowledge.RFPRow{sk.Terms, d.Terms} {
		for _, t := range rows {
			if t.Key != key {
				continue
			}
			for _, m := range slotRE.FindAllString(t.Text, -1) {
				if strings.HasPrefix(m, "{field:") {
					out[m] = true
				}
			}
		}
	}
	return out
}

// validTermText is a term's wording: one trimmed line of up to 600
// characters whose only braces are the allowed {field:x} slots.
func validTermText(text string, slots map[string]bool) bool {
	if text == "" || text != strings.TrimSpace(text) || utf8.RuneCountInString(text) > 600 || !printable(text) || strings.Contains(text, "\n") {
		return false
	}
	ok := true
	rest := slotRE.ReplaceAllStringFunc(text, func(m string) string {
		ok = ok && slots[m]
		return ""
	})
	return ok && !strings.ContainsAny(rest, "{}")
}

// validCustomTerms checks the user's own terms: each with a fresh id and
// wording without slots.
func validCustomTerms(raw json.RawMessage) error {
	var terms []CustomLine
	if len(raw) == 0 || len(raw) > 20000 || json.Unmarshal(raw, &terms) != nil || terms == nil || len(terms) > MaxCustomTerms {
		return ErrInvalidSetting
	}
	seen := map[string]bool{}
	for _, t := range terms {
		if !customIDRE.MatchString(t.ID) || seen[t.ID] || !validTermText(t.Text, nil) || t.Value != "I" {
			return ErrInvalidSetting
		}
		seen[t.ID] = true
	}
	return nil
}

// validLineText is a scope line's wording: trimmed, one line of up to 300
// characters, and never readable as a {slot}.
func validLineText(text string) bool {
	return text != "" && text == strings.TrimSpace(text) && utf8.RuneCountInString(text) <= 300 &&
		printable(text) && !strings.ContainsAny(text, "{}\n")
}

func printable(s string) bool {
	if !utf8.ValidString(s) {
		return false
	}
	for _, r := range s {
		if r < 0x20 && r != '\n' {
			return false
		}
	}
	return true
}

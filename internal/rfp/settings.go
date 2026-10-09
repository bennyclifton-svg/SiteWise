package rfp

import (
	"encoding/json"
	"errors"
	"regexp"
	"strings"
	"time"
	"unicode/utf8"

	"sitewise/internal/knowledge"
)

var ErrInvalidSetting = errors.New("invalid RFP setting")

// factLabels name facts in plain English for checks and controls.
var factLabels = map[string]string{
	"class": "Building class", "category": "Building category", "work_type": "Work type",
	"pathway": "Planning pathway", "procurement": "Procurement route", "novation": "Novation",
	"state": "State", "regulated_design": "Regulated design applies", "budget": "Construction budget",
	"site_area": "Site area", "gfa": "Gross floor area", "storeys": "Storeys", "parking": "Car spaces",
}

// textFacts may be set by the user as a value; flags are set one at a time.
var textFacts = map[string]bool{"work_type": true, "category": true, "pathway": true, "procurement": true, "state": true, "regulated_design": true, "budget": true, "site_area": true, "gfa": true, "storeys": true, "parking": true}

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
	case "scale", "boundary:range", "parts:affected":
		return true
	}
	switch prefix {
	case "driver":
		if flag, ok := strings.CutPrefix(name, "flag."); ok {
			for _, f := range sk.Flags {
				if f.Key == flag {
					return true
				}
			}
			return false
		}
		return textFacts[name]
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
				return true
			}
		}
	case "weight":
		for _, w := range sk.Evaluation {
			if w.Key == name {
				return true
			}
		}
	}
	return false
}

var (
	settingKeyRE = regexp.MustCompile(`^((driver|stage|element|allowance|position|responsible|field|weight):[a-z0-9_.]{1,100}|scale|boundary:range|parts:affected)$`)
	dateRE       = regexp.MustCompile(`^\d{4}-\d{2}-\d{2}$`)
	numberRE     = regexp.MustCompile(`^[0-9]{1,12}(\.[0-9]{1,2})?$`)
)

// ValidateSetting checks a key and its value for the discipline. A key that
// names nothing in the current rules is refused for writing; existing orphans
// can still be deleted.
func ValidateSetting(d knowledge.RFPDiscipline, sk knowledge.RFPSkeleton, drivers DriverSet, key string, raw json.RawMessage) error {
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
	case prefix == "driver" && strings.HasPrefix(name, "flag."):
		return oneOf("yes", "no", "unknown")
	case prefix == "driver":
		return validDriverValue(sk, name, raw)
	case prefix == "stage":
		return oneOf(Include, Option, Exclude, Done)
	case prefix == "element":
		return oneOf(Include, Option, Exclude)
	case prefix == "allowance":
		s, ok := str()
		if !ok || !countRE.MatchString(s) {
			return ErrInvalidSetting
		}
		return nil
	case prefix == "position":
		return oneOf("I", "O", "X")
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

func validDriverValue(sk knowledge.RFPSkeleton, name string, raw json.RawMessage) error {
	values := settingStrings(raw)
	if len(values) != 1 {
		return ErrInvalidSetting
	}
	v := values[0]
	if v == "unknown" {
		return nil
	}
	var list []knowledge.RFPOption
	switch name {
	case "pathway":
		list = sk.Pathways
	case "procurement":
		list = sk.Procurement
	case "work_type":
		list = sk.WorkTypes
	case "regulated_design":
		if v == "yes" || v == "no" {
			return nil
		}
		return ErrInvalidSetting
	case "state":
		if contains([]string{"ACT", "NSW", "NT", "QLD", "SA", "TAS", "VIC", "WA"}, v) {
			return nil
		}
		return ErrInvalidSetting
	case "category":
		if contains([]string{"residential", "commercial", "industrial", "institution", "mixed", "infrastructure"}, v) {
			return nil
		}
		return ErrInvalidSetting
	case "budget", "site_area", "gfa", "storeys", "parking":
		if numberRE.MatchString(v) {
			return nil
		}
		return ErrInvalidSetting
	}
	for _, o := range list {
		if o.Key == v {
			return nil
		}
	}
	return ErrInvalidSetting
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

// SettingBasis records the source fact a driver setting overrides, so a
// later change in that source is shown rather than hidden by the override.
func SettingBasis(drivers DriverSet, key string) json.RawMessage {
	name, ok := strings.CutPrefix(key, "driver:")
	if !ok {
		return json.RawMessage(`{}`)
	}
	f := drivers.Fact(name)
	raw, _ := json.Marshal(FactBasis{State: f.State, Values: f.Values})
	return raw
}

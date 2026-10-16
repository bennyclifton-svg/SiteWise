package rfp

import (
	"regexp"
	"strings"
)

// Schedule item kinds. A stage, element or reimbursable is in the base fee;
// an option is priced below it; a rate is an hourly rate for variations; a
// criterion is a non-price returnable the comparison report lays out.
const (
	ItemStage        = "stage"
	ItemElement      = "element"
	ItemOption       = "option"
	ItemReimbursable = "reimbursable"
	ItemRate         = "rate"
	ItemCriterion    = "criterion"
)

// ScheduleItem is one row of a package's returnable schedule. Key is stable
// across refreshes ("stage:<key>", "element:<stage>:<element>", "option:<key>",
// "reimbursables", "rate:<role>", "criterion:<returnable>"), so a proposal
// price, a cost line and an invoice can all point at the same row.
type ScheduleItem struct {
	Key     string `json:"key"`
	Kind    string `json:"kind"`
	Label   string `json:"label"`
	Detail  string `json:"detail,omitempty"`
	Base    bool   `json:"base"`
	Ordinal int    `json:"ordinal"`
}

// Price reports whether the item is priced as a fee (not a rate or criterion).
func (i ScheduleItem) Price() bool {
	return i.Kind == ItemStage || i.Kind == ItemElement || i.Kind == ItemOption || i.Kind == ItemReimbursable
}

// ReimbursablesKey is the one fee row for disbursements; R9 asks for it.
const ReimbursablesKey = "reimbursables"

// DefaultRoles are the hourly-rate roles for a discipline whose R3 note does
// not list its own. roleDetail says who each covers, so a firm's own titles
// ("Senior Access Consultant", "Engineer") can be read into them.
var DefaultRoles = []string{"Principal or director", "Senior professional", "Professional", "Graduate or technical", "Administration"}

var roleDetail = map[string]string{
	"Principal or director": "The firm's director, principal, partner or technical director.",
	"Senior professional":   "A senior engineer, senior consultant, senior surveyor, associate or senior associate.",
	"Professional":          "An engineer, consultant, surveyor, planner, scientist or designer at practising level, below senior.",
	"Graduate or technical": "A graduate, technician, drafter, CAD or BIM operator, or field technician.",
	"Administration":        "Administration, secretarial or project support staff.",
}

// criterionKeys are the non-price returnables, in the order the report
// prints them. R1 is the fee table, R3 the rates and R9 the reimbursables.
var criterionKeys = []string{"r2", "r4", "r5", "r7", "r8", "r6"}

// Schedule is the package's returnable schedule as data: every priced stage
// (or its elements when the stage is priced by element), every service
// option the RFP asks to be priced, reimbursables, the R3 rate roles and the
// non-price criteria. It reads only the resolved RFP, so a change to the
// selected stages or options changes the items with no code change.
func Schedule(r Resolved) []ScheduleItem {
	var out []ScheduleItem
	add := func(key, kind, label, detail string, base bool) {
		out = append(out, ScheduleItem{Key: key, Kind: kind, Label: strings.TrimSpace(label), Detail: strings.TrimSpace(detail), Base: base, Ordinal: len(out)})
	}
	for _, s := range r.Stages {
		if s.State == Exclude || s.State == Done {
			continue
		}
		if s.ByElement {
			priced := false
			for _, e := range r.Elements {
				if e.State == Exclude {
					continue
				}
				priced = true
				option := s.State == Option || e.State == Option
				kind := ItemElement
				if option {
					kind = ItemOption
				}
				add("element:"+s.Key+":"+e.Key, kind, s.Name+": "+e.Code+" "+e.Name, e.Text, !option)
			}
			if priced {
				continue
			}
		}
		if s.State == Option {
			add("stage:"+s.Key, ItemOption, s.Name, s.Text, false)
			continue
		}
		add("stage:"+s.Key, ItemStage, s.Name, s.Text, true)
	}
	add(ReimbursablesKey, ItemReimbursable, "Reimbursables", "Included in the fee, or each item with its rate (R9)", true)
	for _, p := range r.Positions {
		if p.Value != "O" || (r.Discipline == "hydraulic-services" && hydraulicStageService(p.Key)) {
			continue
		}
		add("option:"+p.Key, ItemOption, shortLabel(p.Label), p.Detail, false)
	}
	for _, role := range Roles(r) {
		detail := roleDetail[role]
		if detail == "" {
			detail = "Hourly rate excl. GST"
		}
		add("rate:"+slug(role), ItemRate, role, detail, false)
	}
	byKey := map[string]Row{}
	for _, ret := range r.Returnables {
		byKey[ret.Key] = ret
	}
	for _, key := range criterionKeys {
		if ret, ok := byKey[key]; ok {
			add("criterion:"+key, ItemCriterion, returnableName(ret.Label), ret.Text, false)
		}
	}
	return out
}

var rolesNote = regexp.MustCompile(`Roles:\s*([^.]+)\.`)

// Roles are the R3 hourly-rate roles printed in the RFP, or DefaultRoles.
func Roles(r Resolved) []string {
	for _, ret := range r.Returnables {
		if ret.Key != "r3" {
			continue
		}
		if m := rolesNote.FindStringSubmatch(ret.Text); m != nil {
			var roles []string
			for _, part := range strings.Split(m[1], "·") {
				if p := strings.TrimSpace(part); p != "" {
					roles = append(roles, p)
				}
			}
			if len(roles) > 0 {
				return roles
			}
		}
	}
	return DefaultRoles
}

// returnableName drops the "R2 " reference: the report names the criterion.
func returnableName(title string) string {
	if len(title) > 3 && (title[0] == 'R' || title[0] == 'r') && strings.Contains(title[:4], " ") {
		_, rest, _ := strings.Cut(title, " ")
		return rest
	}
	return title
}

func slug(s string) string {
	var b strings.Builder
	last := byte('_')
	for _, r := range strings.ToLower(s) {
		switch {
		case r >= 'a' && r <= 'z' || r >= '0' && r <= '9':
			b.WriteRune(r)
			last = 0
		default:
			if last != '_' {
				b.WriteByte('_')
				last = '_'
			}
		}
	}
	return strings.Trim(b.String(), "_")
}

// GenericSchedule is the schedule for a package whose discipline has no RFP
// knowledge: one lump sum, reimbursables, the default rate roles and the
// skeleton's non-price returnables.
func GenericSchedule(returnables []Row) []ScheduleItem {
	r := Resolved{Stages: []StageResult{{Key: "lump_sum", Name: "Lump sum fee", Text: "The whole scope as described", State: Include}}, Returnables: returnables}
	return Schedule(r)
}

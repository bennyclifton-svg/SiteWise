package rfp

import (
	"sort"
	"strings"

	"sitewise/internal/reports"
)

func proposalDateField(key string) bool { return key == "issued" || key == "closes" }

type scopeService struct {
	PositionResult
	Control string `json:"control"`
	Notes   []Row  `json:"notes,omitempty"`
}

// Shared consultant layout keeps stable block IDs for protected wording.
func (b *builder) consultantLayout() {
	project := b.section["project"]
	keptProject := project.Blocks[:0]
	for _, block := range project.Blocks {
		if block.ID == "rfp:header" {
			block.Text = "Release date: " + b.field("issued") + " · Proposals close: " + b.field("closes")
			b.section["issue_dates"] = &reports.Section{ID: "issue_dates", Blocks: []reports.Block{block}, Essential: true}
		} else {
			keptProject = append(keptProject, block)
		}
	}
	project.Blocks = keptProject
	role := b.section["role"].Blocks
	b.section["positions"].Blocks = append(role, b.section["positions"].Blocks...)
	b.section["positions"].Title = "2. Scope of services"
	b.section["terms"].Title = "5. Terms"
	b.section["stages"].Title = "3. Scope by stage and fee proposal"

	b.order = []string{"issue_dates", "project", "positions", "stages", "allowances", "terms", "returnables"}
}

func (b *builder) consultantServices() {
	rows := [][]string{}
	services := []scopeService{}
	for _, key := range []string{"investigation", "planning"} {
		if b.r.Discipline != "hydraulic-services" {
			break
		}
		s := b.r.Stage(key)
		label, detail := "Investigate existing hydraulic services", "Trace and assess retained services; record evidence and assumptions. Agree testing, access, isolation and reinstatement in Allowances; upgrades are separate."
		if key == "planning" {
			label, detail = "Prepare hydraulic authority submissions", "Prepare servicing plans and water-efficiency inputs for authority applications; respond to the agreed review rounds."
		}
		value := map[string]string{Include: "I", Option: "O", Exclude: "X", Done: "X", Ask: Ask}[s.State]
		services = append(services, scopeService{PositionResult: PositionResult{Key: key, Label: label, Detail: detail, Group: "Investigation and authority submissions", Value: value, Source: s.Source}, Control: "stage:" + key})
	}
	for pi, p := range b.r.Scope {
		if b.r.Discipline == "hydraulic-services" && hydraulicStageService(p.Key) && p.Source != "user" {
			continue
		}
		notes := []Row{}
		for _, note := range b.r.ScopeSchedule {
			if target := scopeNoteTarget(b.r.Discipline, note.Key); target == p.Key || (target == "" && pi == 0) {
				notes = append(notes, note)
			}
		}
		services = append(services, scopeService{PositionResult: p, Control: "position:" + p.Key, Notes: notes})
	}
	for _, e := range b.r.Elements {
		value := map[string]string{Include: "I", Option: "O", Exclude: "X", Ask: Ask}[e.State]
		services = append(services, scopeService{PositionResult: PositionResult{Key: e.Key, Label: e.Code + " " + e.Name, Detail: e.Text, Group: "Priced work elements", Value: value, Source: e.Source, Assumption: e.Assumption, Inactive: e.State == Exclude}, Control: "element:" + e.Key})
	}
	// Group order follows the discipline's authored service sequence.
	groupOrder := map[string]int{}
	for _, service := range services {
		if _, ok := groupOrder[service.Group]; !ok {
			groupOrder[service.Group] = len(groupOrder)
		}
	}
	sort.SliceStable(services, func(i, j int) bool { return groupOrder[services[i].Group] < groupOrder[services[j].Group] })
	lastGroup := ""
	for _, service := range services {
		// These controls share a stage selection. The issued schedule already
		// prints their deliverable and price, so do not repeat that scope.
		if strings.HasPrefix(service.Control, "stage:") || (service.Inactive && len(service.Notes) == 0) {
			continue
		}
		p := service.PositionResult
		text := p.Label
		if p.Group != "" && p.Group != lastGroup {
			text = p.Group + " — " + text
			lastGroup = p.Group
		}
		if p.Detail != "" {
			text += ". " + p.Detail
		}
		for _, note := range service.Notes {
			if note.Label != "" {
				text += ". " + note.Label + ": " + note.Text
			} else {
				text += ". " + note.Text
			}
		}
		if p.Assumption != "" {
			text += " Basis: " + p.Assumption
		}
		if p.Value == "X" && p.Party != "" {
			text = strings.TrimSuffix(text, ".") + ". By " + p.Party
		}
		if p.Value == Ask {
			text = strings.TrimSuffix(text, ".") + ". To confirm before issue"
		}
		row := []string{text, "", "", "", ""}
		for i, v := range []string{"I", "O", "X", Ask} {
			if p.Value == v {
				row[i+1] = "X"
			}
		}
		rows = append(rows, row)
	}
	b.add("positions", "rfp:positions", "", "I = included; O = optional price; X = excluded; ? = scope unknown. Resolve unknown scope before issue; propose departures in R6.",
		doc([]string{"Scope of services", "I", "O", "X", "?"}, rows), map[string]any{"positions": services, "layout": "scope_services"})
	// Interfaces now accompany their services; retained protected wording in
	// the old schedule is kept by the report refresh's orphan handling.
}

func (b *builder) consultantStages() {
	rows := [][]string{}
	pricingRows := map[string][][]string{}
	for _, s := range b.r.Stages {
		if s.State == Exclude || s.State == Done {
			continue
		}
		text := stageScopeText(s)
		state := map[string]string{Include: "Included", Option: "Option", Ask: "Unknown"}[s.State]
		fee := ""
		children := [][]string{}
		if s.ByElement {
			for _, e := range b.r.Elements {
				if e.State == Exclude {
					continue
				}
				position := "Included"
				if s.State == Option || e.State == Option {
					position = "Option"
				}
				if s.State == Ask || e.State == Ask {
					position = "Unknown"
				}
				children = append(children, []string{e.Code + " — " + position, e.Name, "", "", ""})
			}
			if len(children) > 0 {
				fee = "By element below"
			}
		}
		rows = append(rows, []string{s.Name + " — " + state, text, fee, "", ""})
		rows = append(rows, children...)
		pricingRows[s.Key] = children
	}
	stageRows := len(rows)
	rows = append(rows, []string{"Base fee total", "Included stages and elements only", "", "", ""})
	for _, p := range b.r.Positions {
		if p.Value != "O" || (b.r.Discipline == "hydraulic-services" && hydraulicStageService(p.Key)) {
			continue
		}
		rows = append(rows, []string{"Service option", shortLabel(p.Label), "", "", ""})
	}
	rows = append(rows, []string{"Options total", "Separate from the base fee", "", "", ""})
	instructions := []string{b.r.StageIntro, b.r.ElementNote}
	for _, ret := range b.r.Returnables {
		if ret.Key == "r1" {
			instructions = append(instructions, ret.Label+": "+ret.Text)
		}
	}
	instructions = append(instructions, "Price included stages in the base fee and options separately, without duplication. Fees exclude GST; state GST and total in your proposal.")
	b.add("stages", "rfp:stages", "", strings.TrimSpace(strings.Join(instructions, " ")), doc([]string{"Stage / position", "Deliverable at sign-off", "Fee $", "Weeks", "Hours"}, rows), map[string]any{"stages": b.r.Stages, "elements": b.r.Elements, "positions": b.r.Positions, "stage_rows": stageRows, "pricing_rows": pricingRows, "layout": "scope_stages"})
}

func stageScopeText(s StageResult) string {
	text := s.Text
	if s.Responsibility != "" {
		text += ". " + s.Responsibility
	}
	if s.Assumption != "" {
		text += ". " + s.Assumption
	}
	switch s.FeeBasis {
	case "time_to_ceiling":
		text += ". Fee ceiling: hours × R3 rate"
	case "per_unit":
		text += ". Fee: rate per unit × quantity"
	}
	return strings.TrimSpace(text)
}

// Allocation schedules retain their exact wording beside the service they
// qualify. These are design/test ownership details, not another fee item.
func scopeNoteTarget(discipline, key string) string {
	switch discipline {
	case "electrical-services":
		if key == "emergency" {
			return "lighting"
		}
		if key == "integrated" {
			return "handover"
		}
		return "dry_fire"
	case "mechanical-services":
		return "interfaces"
	case "fire-services":
		return "controls"
	}
	return ""
}

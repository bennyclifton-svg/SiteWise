package rfp

import (
	"sort"
	"strings"
	"unicode"
	"unicode/utf8"

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
	// Most important first (owner direction, 2 November 2026): the project,
	// the scope in words, the priced stages, the service positions, then the
	// documents supplied. The proposal dates close the RFP rather than head it.
	project := b.section["project"]
	keptProject := []reports.Block{}
	for _, block := range project.Blocks {
		switch block.ID {
		case "rfp:header":
			block.Text = "Release date: " + b.field("issued") + " · Proposals close: " + b.field("closes")
			b.section["issue_dates"] = &reports.Section{ID: "issue_dates", Title: "Proposal dates", Blocks: []reports.Block{block}, Essential: true}
		case "rfp:project:information":
			block.Label = ""
			b.section["transmittal"] = &reports.Section{ID: "transmittal", Blocks: []reports.Block{block}, Essential: true}
		default:
			keptProject = append(keptProject, block)
		}
	}
	project.Blocks = keptProject
	// The stage table and the service positions are parts of section 2, so
	// every RFP numbers 1 Project to 5 Terms whichever scope layout it uses.
	b.section["role"].Title = "2. Scope of services"
	b.section["stages"].Title = "Scope by stage and fee proposal"
	b.section["positions"].Title = "Service positions"
	b.section["transmittal"].Title = "3. Transmittal"
	b.section["allowances"].Title = "4. Allowances"
	b.section["terms"].Title = "5. Terms"
	ids := []string{"project", "role", "stages", "positions", "transmittal", "allowances", "terms", "returnables", "issue_dates"}
	if b.r.ScopeByStage {
		// One table straight under the scope wording: each stage with its
		// services beneath it (owner direction, 2 November 2026).
		b.section["role"].Blocks = append(b.section["role"].Blocks, b.section["stages"].Blocks...)
		ids = []string{"project", "role", "transmittal", "allowances", "terms", "returnables", "issue_dates"}
	}

	b.order = []string{}
	for _, id := range ids {
		if b.section[id] != nil {
			b.order = append(b.order, id)
		}
	}
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
		if strings.HasPrefix(service.Control, "stage:") || (service.Inactive && len(service.Notes) == 0) || b.opt.moved[service.Key] {
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
	if b.r.ScopeByStage {
		b.stageScopeTable()
		return
	}
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
				children = append(children, []string{e.Code + " — " + position, e.Name, ""})
			}
			if len(children) > 0 {
				fee = "By element below"
			}
		}
		rows = append(rows, []string{s.Name + " — " + state, text, fee})
		rows = append(rows, children...)
		pricingRows[s.Key] = children
	}
	stageRows := len(rows)
	rows = append(rows, []string{"Reimbursables", "Included in the fee, or each item with its rate (R9)", ""})
	rows = append(rows, []string{"Base fee total", "Included stages and elements only", ""})
	for _, p := range b.r.Positions {
		if p.Value != "O" || (b.r.Discipline == "hydraulic-services" && hydraulicStageService(p.Key)) {
			continue
		}
		rows = append(rows, []string{"Service option", shortLabel(p.Label), ""})
	}
	rows = append(rows, []string{"Options total", "Separate from the base fee", ""})
	instructions := b.stageInstructions()
	instructions = append(instructions, "Price included stages in the base fee and options separately, without duplication. Fees exclude GST; state GST and total in your proposal.")
	b.add("stages", "rfp:stages", "", strings.TrimSpace(strings.Join(instructions, " ")), doc([]string{"Stage / position", "Deliverable at sign-off", "Fee $"}, rows), map[string]any{"stages": b.r.Stages, "elements": b.r.Elements, "positions": b.r.Positions, "stage_rows": stageRows, "pricing_rows": pricingRows, "schedule": Schedule(b.r), "layout": "scope_stages"})
}

func (b *builder) stageInstructions() []string {
	instructions := []string{b.r.StageIntro, b.r.ElementNote}
	for _, ret := range b.r.Returnables {
		if ret.Key == "r1" {
			instructions = append(instructions, ret.Label+": "+ret.Text)
		}
	}
	return instructions
}

// stageScopeTable is scope by stage (owner direction, 2-3 November 2026):
// each stage, then its scope lines and services, each included or excluded,
// and one fee per included stage. A stage a rule leaves out is not printed;
// one the user leaves out prints as excluded, without its lines.
func (b *builder) stageScopeTable() {
	position := func(v string) string {
		if v == "X" {
			return "Excluded"
		}
		return "Included"
	}
	services := func(stage string) [][]string {
		rows := [][]string{}
		for _, p := range b.stageServices() {
			if p.Stage != stage || p.Inactive || b.opt.moved[p.Key] {
				continue
			}
			rows = append(rows, []string{"", serviceText(p), position(p.Value), ""})
		}
		return rows
	}
	rows := [][]string{}
	for _, s := range b.r.Stages {
		if s.State == Exclude && s.Source == "rule" {
			continue
		}
		if s.State == Exclude {
			rows = append(rows, []string{s.Name, "Not part of this appointment", "Excluded", "—"})
			continue
		}
		note := strings.TrimSpace(s.Responsibility + " " + s.Assumption)
		switch s.FeeBasis {
		case "time_to_ceiling":
			note = strings.TrimSpace(note + " Fee ceiling: hours × R3 rate.")
		case "per_unit":
			note = strings.TrimSpace(note + " Fee: rate per unit × quantity.")
		}
		rows = append(rows, []string{s.Name, note, "Included", ""})
		for _, l := range s.Lines {
			rows = append(rows, []string{"", upperFirst(l.Text), position(l.Value), ""})
		}
		// A stage once priced element by element now lists its elements
		// as lines inside the one stage fee.
		if s.ByElement {
			for _, e := range b.r.Elements {
				v := "I"
				if e.State == Exclude {
					v = "X"
				}
				rows = append(rows, []string{"", e.Code + " " + e.Name, position(v), ""})
			}
		}
		rows = append(rows, services(s.Key)...)
	}
	rows = append(rows, []string{"Across all stages", "Services through the whole appointment", "", "In stage fees"})
	rows = append(rows, services("")...)
	stageRows := len(rows)
	rows = append(rows, []string{"Reimbursables", "Included in the fee, or each item with its rate (R9)", "", ""})
	rows = append(rows, []string{"Total fee", "Included stages and reimbursables", "", ""})
	if b.opt.part {
		b.partStageTable(rows[:stageRows])
		return
	}
	instructions := append([]string{b.r.StageIntro}, "Price each included stage. A line marked Included is part of its stage fee; a line marked Excluded is not part of this appointment. Fees exclude GST; state GST and total in your proposal. Propose any departure in R6.")
	for _, ret := range b.r.Returnables {
		if ret.Key == "r1" {
			instructions = append(instructions[:1], append([]string{ret.Label + ": " + ret.Text}, instructions[1:]...)...)
		}
	}
	b.add("stages", "rfp:stages", "", strings.TrimSpace(strings.Join(instructions, " ")), doc([]string{"Stage", "Scope", "Position", "Fee $"}, rows),
		map[string]any{"stages": b.r.Stages, "elements": b.r.Elements, "services": b.stageServices(), "stage_rows": stageRows, "schedule": Schedule(b.r), "layout": "stage_scope"})
}

// partStageTable is the stage table of one discipline in a combined
// engagement: the same stages, lines and positions, with no fee column. The
// engagement's fee schedule prices every discipline once (group.go).
func (b *builder) partStageTable(rows [][]string) {
	printed := make([][]string, len(rows))
	for i, row := range rows {
		printed[i] = row[:3]
	}
	text := strings.TrimSpace(b.r.StageIntro + " A line marked Included is part of its stage; a line marked Excluded is not part of this engagement. Price each included stage in the fee schedule. Propose any departure in R6.")
	b.add("stages", "rfp:stages", "", text, doc([]string{"Stage", "Scope", "Position"}, printed),
		map[string]any{"stages": b.r.Stages, "elements": b.r.Elements, "services": b.stageServices(), "stage_rows": len(rows), "schedule": Schedule(b.r), "layout": "stage_scope"})
}

// serviceText is a service as the table prints it: its label, detail and
// any allocation note that qualifies it.
func serviceText(p scopeService) string {
	text := p.Label
	if p.Detail != "" {
		text += ". " + p.Detail
	}
	for _, n := range p.Notes {
		if n.Label != "" {
			text += ". " + n.Label + ": " + n.Text
		} else {
			text += ". " + n.Text
		}
	}
	return text
}

func upperFirst(s string) string {
	if s == "" {
		return s
	}
	r, n := utf8.DecodeRuneInString(s)
	return string(unicode.ToUpper(r)) + s[n:]
}

// stageServices are the selectable services in authored order, each with
// its home stage and control, inapplicable ones included so the screen can
// still offer them.
func (b *builder) stageServices() []scopeService {
	out := []scopeService{}
	// A note naming no service goes with the first service that prints.
	general := ""
	for _, p := range b.r.Scope {
		if !p.Inactive && !(b.r.Discipline == "hydraulic-services" && hydraulicStageService(p.Key)) {
			general = p.Key
			break
		}
	}
	for _, p := range b.r.Scope {
		// Hydraulic's stage-named services are its stages; the stage row
		// carries them.
		if b.r.Discipline == "hydraulic-services" && hydraulicStageService(p.Key) {
			continue
		}
		notes := []Row{}
		for _, note := range b.r.ScopeSchedule {
			if target := scopeNoteTarget(b.r.Discipline, note.Key); target == p.Key || (target == "" && p.Key == general) {
				notes = append(notes, note)
			}
		}
		out = append(out, scopeService{PositionResult: p, Control: "position:" + p.Key, Notes: notes})
	}
	return out
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

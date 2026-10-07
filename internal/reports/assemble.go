package reports

import (
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	"sitewise/internal/delivery"
	"sitewise/internal/knowledge"
	"sitewise/internal/procurement"
	"sitewise/internal/works"
)

var ErrReadingIncomplete = errors.New("reading is pending or failed; choose last completed state or wait")

type BriefValue struct {
	ID           string
	Label        string
	Text         string
	Origin       string
	ReviewStatus string
	Meaning      string
	Basis        json.RawMessage
	Unknown      bool
}

type ScopeRecord struct {
	procurement.ScopeContent
	ID           string          `json:"id"`
	PackageID    string          `json:"package_id"`
	Origin       string          `json:"origin"`
	ReviewStatus string          `json:"review_status"`
	Meaning      string          `json:"meaning"`
	Provenance   json.RawMessage `json:"provenance"`
	SourceRefs   json.RawMessage `json:"source_refs"`
	RetiredAt    *time.Time      `json:"retired_at,omitempty"`
	Version      int64           `json:"version"`
}

type Snapshot struct {
	ProjectID            string
	ProjectName          string
	ProjectBasis         json.RawMessage
	PartLabels           map[string]string
	Package              procurement.Package
	Brief                []BriefValue
	Scope                []ScopeRecord
	Works                []works.Item
	Proposals            []works.Proposal
	DeliveryDependencies []delivery.Dependency
	Delivery             []delivery.Item
	Gaps                 []procurement.Gap
	PendingDocuments     int
	FailedDocuments      int
	UnreadDocuments      int
	ProfileMissing       bool
	ProfileStale         []string
	Packages             []procurement.Package
	Documents            []BriefValue
	Commercial           []BriefValue
}

func AssembleRFP(s Snapshot, template knowledge.ReportTemplate, cat *knowledge.Catalog, edits []Edit, useLastCompleted bool) ([]Section, error) {
	if template.Kind != "rfp" {
		return nil, fmt.Errorf("RFP template required")
	}
	return Assemble(s, template, cat, edits, useLastCompleted)
}

// Assemble uses one saved-state pipeline for each report kind.
func Assemble(s Snapshot, template knowledge.ReportTemplate, cat *knowledge.Catalog, edits []Edit, useLastCompleted bool) ([]Section, error) {
	if _, err := json.Marshal(s); err != nil {
		return nil, fmt.Errorf("invalid report snapshot: %w", err)
	}
	if cat == nil || !validReportPackage(template.Kind, s.Package) {
		return nil, fmt.Errorf("report requires a matching live package and template")
	}
	if (s.PendingDocuments > 0 || s.FailedDocuments > 0 || (!s.ProfileMissing && len(s.ProfileStale) > 0)) && !useLastCompleted {
		return nil, ErrReadingIncomplete
	}
	sections := make([]Section, 0, len(template.Sections))
	byID := map[string]int{}
	for _, section := range template.Sections {
		if _, ok := byID[section.ID]; ok {
			return nil, fmt.Errorf("duplicate template section")
		}
		byID[section.ID] = len(sections)
		out := Section{ID: section.ID, Title: section.Title, Essential: section.Essential, Blocks: []Block{}}
		for _, ref := range section.Clauses {
			clause, ok := cat.Clause(ref.ID, ref.Version)
			if !ok {
				return nil, fmt.Errorf("clause version unavailable")
			}
			basis, _ := json.Marshal(map[string]any{"method": "catalogue_clause", "clause": clause})
			out.Blocks = append(out.Blocks, Block{ID: "clause:" + section.ID + ":" + clause.ID, Label: "Standard wording", Text: clause.Text, Origin: "calculation", ReviewStatus: "proposed", Meaning: "requirement", Basis: basis, Provisional: clause.Status != "reviewed"})
		}
		sections = append(sections, out)
	}
	for _, id := range []string{"brief", "services", "investigations", "interfaces", "dates", "fee_return", "proposal_requirements"} {
		if _, ok := byID[id]; !ok {
			return nil, fmt.Errorf("report template missing %s", id)
		}
	}
	add := func(section string, b Block) { i := byID[section]; sections[i].Blocks = append(sections[i].Blocks, b) }
	var basisError error
	calculated := func(id, label, text string, basis any) Block {
		raw, err := json.Marshal(basis)
		if err != nil && basisError == nil {
			basisError = fmt.Errorf("report basis %s: %w", id, err)
		}
		return Block{ID: id, Label: label, Text: text, Origin: "calculation", ReviewStatus: "proposed", Meaning: "stated", Basis: raw}
	}
	add("brief", Block{ID: "project", Label: "Project", Text: s.ProjectName, Origin: "user", ReviewStatus: "accepted_for_planning", Meaning: "stated", Basis: s.ProjectBasis})
	add("brief", calculated("reading_status", "Reading status", readingSummary(s, useLastCompleted), map[string]any{"method": "saved_reading_status", "pending": s.PendingDocuments, "failed": s.FailedDocuments, "unread": s.UnreadDocuments, "profile_missing": s.ProfileMissing, "profile_stale": s.ProfileStale, "use_last_completed": useLastCompleted}))
	brief := append([]BriefValue(nil), s.Brief...)
	sort.Slice(brief, func(i, j int) bool { return brief[i].ID < brief[j].ID })
	for _, v := range brief {
		text := v.Text
		if v.Unknown || text == "" {
			text = "Not established"
		}
		add("brief", Block{ID: "brief:" + v.ID, Label: v.Label, Text: text, Origin: v.Origin, ReviewStatus: v.ReviewStatus, Meaning: v.Meaning, Basis: v.Basis, Provisional: v.Unknown})
	}
	packageBasis, _ := json.Marshal(s.Package)
	if template.Kind != "pmp" {
		add("services", Block{ID: "package:" + s.Package.ID, Label: "Package", Text: s.Package.Title + " — " + s.Package.LifecycleStatus, Origin: s.Package.Origin, ReviewStatus: s.Package.ReviewStatus, Meaning: s.Package.Meaning, Basis: packageBasis})
	}
	stages := append([]procurement.Stage(nil), s.Package.Stages...)
	stageLabels := map[string]string{}
	sort.Slice(stages, func(i, j int) bool {
		if stages[i].Ordinal != stages[j].Ordinal {
			return stages[i].Ordinal < stages[j].Ordinal
		}
		return stages[i].ID < stages[j].ID
	})
	for _, stage := range stages {
		if stage.RetiredAt != nil {
			continue
		}
		stageLabels[stage.ID] = stage.Label
		basis, _ := json.Marshal(stage)
		label := stage.Label
		if stage.NovationPhase == "pre" {
			label += " — before novation"
		}
		if stage.NovationPhase == "post" {
			label += " — after novation"
		}
		add("services", Block{ID: "stage:" + stage.ID, Label: "Stage", Text: label, Origin: stage.Origin, ReviewStatus: "accepted_for_planning", Meaning: "stated", Basis: basis})
		add("fee_return", calculated("fee_stage:"+stage.ID, stage.Label, "Fee to be provided; no budget disclosed.", map[string]any{"method": "stage_fee_return", "stage": stage}))
	}
	workByID := map[string]works.Item{}
	for _, w := range s.Works {
		workByID[w.ID] = w
	}
	scope := append([]ScopeRecord(nil), s.Scope...)
	sort.Slice(scope, func(i, j int) bool { return scope[i].ID < scope[j].ID })
	for _, row := range scope {
		if (template.Kind != "pmp" && row.PackageID != s.Package.ID) || row.RetiredAt != nil {
			continue
		}
		text := row.UserText
		provisional := false
		if row.ClauseID != "" {
			clause, ok := cat.Clause(row.ClauseID, row.ClauseVersion)
			if !ok {
				return nil, fmt.Errorf("scope clause version unavailable")
			}
			text = clause.Text
			provisional = clause.Status != "reviewed"
		}
		var provenance struct {
			Proposal struct {
				Draft bool `json:"draft"`
			} `json:"proposal"`
		}
		if json.Unmarshal(row.Provenance, &provenance) == nil {
			provisional = provisional || provenance.Proposal.Draft
		}
		parts := []string{row.Inclusion, text}
		if row.StageID != "" && template.Kind != "pmp" {
			label, ok := stageLabels[row.StageID]
			if !ok {
				return nil, fmt.Errorf("scope stage unavailable")
			}
			parts = append(parts, "Stage: "+label)
		}
		if row.Role != "" {
			parts = append(parts, "Role: "+strings.ReplaceAll(row.Role, "_", " "))
		}
		if row.Deliverable != "" {
			parts = append(parts, "Deliverable: "+row.Deliverable)
		}
		if row.WorkItemID != "" {
			w, ok := workByID[row.WorkItemID]
			if !ok {
				return nil, fmt.Errorf("scope work reference unavailable")
			}
			parts = append(parts, "Work: "+w.Title+" ("+w.Action+")")
		}
		basis, _ := json.Marshal(row)
		add("services", Block{ID: "scope:" + row.ID, Label: row.ItemKind, Text: strings.Join(parts, ". "), Origin: row.Origin, ReviewStatus: row.ReviewStatus, Meaning: row.Meaning, Basis: basis, Provisional: provisional})
		for _, id := range row.InterfaceIDs {
			var edge knowledge.Interface
			for _, candidate := range cat.Interfaces {
				if candidate.ID == id {
					edge = candidate
					break
				}
			}
			if edge.ID == "" {
				return nil, fmt.Errorf("scope interface unavailable")
			}
			// Snapshot the relationship used here, not its YAML question criteria.
			basis := map[string]any{"method": "scope_interface", "scope": row, "interface": map[string]any{"id": edge.ID, "type": edge.Type, "from": edge.From, "to": edge.To, "summary": edge.Summary, "status": edge.Status}}
			add("interfaces", calculated("interface:"+row.ID+":"+id, "Coordination interface", edge.Summary, basis))
		}
	}
	worksCopy := append([]works.Item(nil), s.Works...)
	sort.Slice(worksCopy, func(i, j int) bool { return worksCopy[i].ID < worksCopy[j].ID })
	for _, w := range worksCopy {
		if w.RetiredAt != nil || w.IsGroup || (template.Kind == "rft" && !workInPackage(w, s.Works, s.Scope, s.Package.ID, reportPackages(s))) {
			continue
		}
		basis := workBasis(w, cat)
		section, label := "brief", "Work scope — "+w.Inclusion
		if w.Action == "investigate" && w.Inclusion == "included" {
			section, label = "investigations", "Investigation"
		}
		location := s.PartLabels[w.PartID]
		if location == "" {
			location = "Location not established"
		}
		add(section, Block{ID: "work:" + w.ID, Label: label + " — " + strings.ReplaceAll(w.ReviewStatus, "_", " "), Text: workDescription(w, location, cat), Origin: w.Origin, ReviewStatus: w.ReviewStatus, Meaning: w.Meaning, Basis: basis, Provisional: w.ReviewStatus == "proposed" || !workTargetClausesReviewed(w, cat)})
	}
	proposals := append([]works.Proposal(nil), s.Proposals...)
	sort.Slice(proposals, func(i, j int) bool { return proposals[i].Key < proposals[j].Key })
	for _, p := range proposals {
		if p.State == "dismissed" || p.State == "accepted" || (template.Kind == "rft" && !proposalInPackage(p, s, cat)) {
			continue
		}
		if p.RecordKind == "uc" && (template.Kind == "rft" || template.Kind == "pmp") {
			continue // The risk block below includes effects and contract treatment.
		}
		section := "interfaces"
		if p.Kind == "investigation" {
			section = "investigations"
		}
		b := calculated("proposal:"+p.Key, "Unaccepted proposal", p.Label, p)
		b.Provisional = true
		add(section, b)
	}
	dates := append([]delivery.Item(nil), s.Delivery...)
	sort.Slice(dates, func(i, j int) bool { return dates[i].ID < dates[j].ID })
	for _, d := range dates {
		if d.RetiredAt != nil || (template.Kind != "pmp" && d.PackageID != "" && d.PackageID != s.Package.ID) {
			continue
		}
		parts := []string{d.Title, "Status: " + strings.ReplaceAll(d.Status, "_", " ")}
		for _, v := range []struct {
			label string
			value *string
		}{{"Baseline", d.BaselineDate}, {"Target", d.TargetDate}, {"Forecast", d.ForecastDate}, {"Actual", d.ActualDate}, {"As of", d.AsOf}} {
			if v.value != nil {
				parts = append(parts, v.label+": "+*v.value)
			}
		}
		if d.OwnerText != "" {
			parts = append(parts, "Owner: "+d.OwnerText)
		}
		parts = append(parts, deliveryDetails(d)...)
		basis, _ := json.Marshal(d)
		section := "dates"
		if d.Kind == "risk" || d.Kind == "issue" {
			section = "brief"
		}
		add(section, Block{ID: "delivery:" + d.ID, Label: d.Kind, Text: strings.Join(parts, ". "), Origin: d.Origin, ReviewStatus: d.ReviewStatus, Meaning: d.Meaning, Basis: basis, Table: DeliveryComparison(d)})
	}
	// Dependencies are saved sequencing decisions, never inferred from dates.
	byDeliveryID := map[string]delivery.Item{}
	for _, item := range s.Delivery {
		if item.RetiredAt == nil {
			byDeliveryID[item.ID] = item
		}
	}
	edges := append([]delivery.Dependency(nil), s.DeliveryDependencies...)
	sort.Slice(edges, func(i, j int) bool {
		return edges[i].PredecessorID+edges[i].SuccessorID < edges[j].PredecessorID+edges[j].SuccessorID
	})
	for _, edge := range edges {
		before, bok := byDeliveryID[edge.PredecessorID]
		after, aok := byDeliveryID[edge.SuccessorID]
		if !bok || !aok {
			continue
		}
		if template.Kind != "pmp" && before.PackageID != s.Package.ID && after.PackageID != s.Package.ID && before.PackageID != "" && after.PackageID != "" {
			continue
		}
		add("dates", calculated("dependency:"+edge.PredecessorID+":"+edge.SuccessorID, "Sequencing", fmt.Sprintf("%s finishes before %s starts; lag %d days.", before.Title, after.Title, edge.LagDays), map[string]any{"method": "explicit_finish_to_start", "dependency": edge, "predecessor": before, "successor": after}))
	}
	gaps := append([]procurement.Gap(nil), s.Gaps...)
	sort.Slice(gaps, func(i, j int) bool {
		if gaps[i].WorkItemID != gaps[j].WorkItemID {
			return gaps[i].WorkItemID < gaps[j].WorkItemID
		}
		return gaps[i].Role < gaps[j].Role
	})
	for _, gap := range gaps {
		if template.Kind == "rft" {
			w, ok := workByID[gap.WorkItemID]
			if !ok || !workInPackage(w, s.Works, s.Scope, s.Package.ID, reportPackages(s)) {
				continue
			}
		}
		add("interfaces", calculated("gap:"+gap.WorkItemID+":"+gap.Role, "Allocation "+gap.State, strings.ReplaceAll(gap.Role, "_", " "), map[string]any{"method": "gap_check", "finding": gap}))
	}
	for _, id := range []string{"services", "investigations", "dates", "fee_return"} {
		i := byID[id]
		if len(sections[i].Blocks) == len(template.Sections[i].Clauses) {
			add(id, calculated("unknown:"+id, "Not established", "No project records have been provided for this section.", map[string]any{"method": "missing_saved_records", "section": id}))
		}
	}
	appendReportDetails(template.Kind, s, cat, add, calculated)
	if basisError != nil {
		return nil, basisError
	}
	return ApplyEdits(sections, edits)
}

func readingSummary(s Snapshot, lastCompleted bool) string {
	parts := []string{}
	if s.ProfileMissing {
		parts = append(parts, "No completed project profile is available. Update the project profile to include document findings.")
	} else if len(s.ProfileStale) > 0 {
		parts = append(parts, "The saved project profile needs updating. Update it before relying on these findings.")
	} else {
		parts = append(parts, "Document findings use the saved project profile.")
	}
	if s.PendingDocuments > 0 {
		parts = append(parts, fmt.Sprintf("%d documents are waiting to finish reading.", s.PendingDocuments))
	}
	if s.FailedDocuments > 0 {
		parts = append(parts, fmt.Sprintf("Reading stopped for %d documents; retry them from the project profile.", s.FailedDocuments))
	}
	if s.UnreadDocuments > 0 {
		parts = append(parts, fmt.Sprintf("%d documents have not been read into the profile.", s.UnreadDocuments))
	}
	if lastCompleted && !s.ProfileMissing {
		parts = append(parts, "You chose to use the last completed profile for this draft.")
	}
	return strings.Join(parts, " ")
}

func deliveryDetails(item delivery.Item) []string {
	out := []string{}
	add := func(label, value string) {
		if value != "" {
			out = append(out, label+": "+value)
		}
	}
	if item.BaselineDate != nil && item.ForecastDate != nil {
		baseline, e1 := time.Parse("2006-01-02", *item.BaselineDate)
		forecast, e2 := time.Parse("2006-01-02", *item.ForecastDate)
		if e1 == nil && e2 == nil {
			add("Forecast less baseline", fmt.Sprintf("%+d calendar days", int(forecast.Sub(baseline).Hours()/24)))
		}
	}
	switch item.Kind {
	case "risk":
		var d delivery.RiskDetails
		_ = json.Unmarshal(item.Details, &d)
		add("Likelihood", d.Likelihood)
		add("Consequence", d.Consequence)
	case "approval":
		var d delivery.ApprovalDetails
		_ = json.Unmarshal(item.Details, &d)
		add("Authority", d.Authority)
		add("Reference", d.Reference)
		if d.SubmittedOn != nil {
			add("Submitted", *d.SubmittedOn)
		}
		if d.DeterminedOn != nil {
			add("Determined", *d.DeterminedOn)
		}
	case "decision":
		var d delivery.DecisionDetails
		_ = json.Unmarshal(item.Details, &d)
		add("Options", strings.Join(d.Options, "; "))
		add("Chosen", d.Chosen)
	}
	return out
}

package reports

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"sitewise/internal/knowledge"
	"sitewise/internal/procurement"
	"sitewise/internal/works"
)

func validReportPackage(kind string, p procurement.Package) bool {
	if kind == "pmp" {
		return p.ID == ""
	}
	if p.ID == "" || p.RetiredAt != nil {
		return false
	}
	return kind == "rfp" && p.Kind == "services" || kind == "rft" && p.Kind == "works"
}

func workInPackage(w works.Item, all []works.Item, scope []ScopeRecord, pkg string, packages ...[]procurement.Package) bool {
	live := map[string]bool{}
	if len(packages) > 0 {
		for _, p := range packages[0] {
			if p.RetiredAt == nil {
				live[p.ID] = true
			}
		}
	} else {
		for _, r := range scope {
			live[r.PackageID] = true
		}
	}
	if !live[pkg] {
		return false
	}
	byID := map[string]works.Item{}
	for _, item := range all {
		byID[item.ID] = item
	}
	seen := map[string]bool{}
	resolvedRoles := map[string]bool{}
	// Match CheckGaps: each role resolves at its nearest included live ancestor.
	// Explicit non-role scope links remain associations and do not override roles.
	for {
		if seen[w.ID] || w.RetiredAt != nil || w.Inclusion != "included" {
			return false
		}
		seen[w.ID] = true
		here := map[string]bool{}
		for _, row := range scope {
			if row.WorkItemID != w.ID || row.RetiredAt != nil || row.Inclusion != "included" || !live[row.PackageID] {
				continue
			}
			if row.Role == "" {
				if row.PackageID == pkg {
					return true
				}
				continue
			}
			if resolvedRoles[row.Role] {
				continue
			}
			here[row.Role] = true
			if row.PackageID == pkg {
				return true
			}
		}
		for role := range here {
			resolvedRoles[role] = true
		}
		if w.ParentID == "" {
			return false
		}
		parent, ok := byID[w.ParentID]
		if !ok || !parent.IsGroup {
			return false
		}
		w = parent
	}
}

func reportPackages(s Snapshot) []procurement.Package {
	if len(s.Packages) > 0 {
		return s.Packages
	}
	// Small pure assembly fixtures can carry only their selected package.
	return []procurement.Package{s.Package}
}

func workDescription(w works.Item, location string, catalogues ...*knowledge.Catalog) string {
	text := w.Title + ". Action: " + w.Action + ". Location: " + location
	if w.ExistingConditionNote != "" {
		text += ". Existing condition: " + w.ExistingConditionNote
	}
	if w.Quantity != nil && w.Unit != nil {
		text += ". Quantity: " + *w.Quantity + " " + *w.Unit
	}
	if w.Action == "retain" {
		text += ". Retain: confirm protection or continued operation in the package responsibilities."
	}
	if w.LayoutChange == "yes" || w.LayoutChange == "no" {
		text += ". Room or space boundaries change: " + w.LayoutChange
	}
	if w.Target.Text != "" {
		text += ". Target: " + w.Target.Text
	}
	for _, v := range w.Target.Values {
		var value any
		decoder := json.NewDecoder(strings.NewReader(string(v.Value)))
		decoder.UseNumber()
		shown := "Not established"
		if decoder.Decode(&value) == nil {
			switch scalar := value.(type) {
			case string:
				shown = scalar
			case json.Number:
				shown = scalar.String()
			case bool:
				shown = fmt.Sprint(scalar)
			}
		}
		text += fmt.Sprintf(". %s: %s %s", v.Key, shown, v.Unit)
	}
	for _, ref := range w.Target.ClauseRefs {
		text += fmt.Sprintf(". Target clause %s v%d", ref.ID, ref.Version)
		if len(catalogues) > 0 && catalogues[0] != nil {
			if clause, ok := catalogues[0].Clause(ref.ID, ref.Version); ok {
				text += ": " + clause.Text
			}
		}
	}
	return text
}
func appendReportDetails(kind string, s Snapshot, cat *knowledge.Catalog, add func(string, Block), calculated func(string, string, string, any) Block) {
	if kind == "pmp" {
		packages := append([]procurement.Package(nil), s.Packages...)
		sort.Slice(packages, func(i, j int) bool { return packages[i].ID < packages[j].ID })
		for _, p := range packages {
			if p.RetiredAt != nil {
				continue
			}
			basis, _ := json.Marshal(p)
			add("services", Block{ID: "appointment:" + p.ID, Label: p.Kind, Text: p.Title + " — " + p.LifecycleStatus, Origin: p.Origin, ReviewStatus: p.ReviewStatus, Meaning: p.Meaning, Basis: basis})
		}
	}
	for _, v := range s.Commercial {
		text := v.Text
		if v.Unknown || text == "" {
			text = "Not available"
		}
		add("fee_return", Block{ID: "cost:" + v.ID, Label: v.Label, Text: text, Origin: v.Origin, ReviewStatus: v.ReviewStatus, Meaning: v.Meaning, Basis: v.Basis, Provisional: v.Unknown})
	}
	if len(s.Commercial) == 0 {
		add("fee_return", calculated("cost:unavailable", "Cost coverage", "Cost plan not available; no amount has been assumed.", map[string]any{"method": "cost_plan_availability"}))
	}
	if kind == "rft" || kind == "pmp" {
		docs := append([]BriefValue(nil), s.Documents...)
		sort.Slice(docs, func(i, j int) bool { return docs[i].ID < docs[j].ID })
		for _, v := range docs {
			add("proposal_requirements", Block{ID: "document:" + v.ID, Label: v.Label, Text: v.Text, Origin: "document", ReviewStatus: "proposed", Meaning: "stated", Basis: v.Basis})
		}
		if len(docs) == 0 {
			add("proposal_requirements", calculated("documents:unknown", "Document schedule", "No document revisions have been recorded.", map[string]any{"method": "document_schedule"}))
		}
	}
	if kind == "rft" || kind == "pmp" {
		conditions := map[string]knowledge.Unforeseen{}
		for _, record := range cat.UnforeseenConditions() {
			conditions[record.ID] = record
		}
		proposals := append([]works.Proposal(nil), s.Proposals...)
		sort.Slice(proposals, func(i, j int) bool { return proposals[i].Key < proposals[j].Key })
		for _, p := range proposals {
			if p.RecordKind != "uc" || p.State == "dismissed" || (kind == "rft" && !proposalInPackage(p, s, cat)) {
				continue
			}
			text := p.Label + ". Review state: " + strings.ReplaceAll(p.State, "_", " ")
			basis := map[string]any{"method": "unforeseen_condition_review", "proposal": p}
			if record, ok := conditions[p.RecordID]; ok {
				text += ". Potential effects: " + strings.Join(record.Effect, ", ")
				if record.DeRisk != nil {
					text += ". De-risk review: " + record.DeRisk.Label
				}
				if record.Contract != "" {
					text += ". Suggested contract treatment (not allocated or priced): " + record.Contract
				}
				// Freeze the exact fields shown, without serializing YAML predicates.
				basis["record"] = map[string]any{"id": record.ID, "effect": record.Effect, "de_risk": record.DeRisk, "contract": record.Contract, "status": record.Status}
			} else {
				text += ". Catalogue detail unavailable; review effects and contract treatment before relying on this entry."
			}
			label, section := "Latent condition — review required", "investigations"
			if kind == "pmp" {
				label, section = "Unforeseen-condition risk — review required", "brief"
			}
			b := calculated("latent:"+p.Key, label, text, basis)
			b.Provisional = true
			add(section, b)
		}
	}
	if kind == "rft" {
		add("fee_return", calculated("return:pricing", "Tender return", "Return prices against the listed stable cost-item IDs. Returned offers are separate from internal budgets; a quote is not a commitment.", map[string]any{"method": "pricing_return_instructions"}))
	}
}

// Project-wide reviews apply to each package. Work-specific reviews follow
// included responsibilities, including responsibilities inherited from groups.
func proposalInPackage(p works.Proposal, s Snapshot, cat *knowledge.Catalog) bool {
	if s.Package.RetiredAt != nil {
		return false
	}
	live := false
	for _, pkg := range reportPackages(s) {
		if pkg.ID == s.Package.ID && pkg.RetiredAt == nil {
			live = true
		}
	}
	if !live {
		return false
	}
	if p.InterfaceID != "" {
		for _, row := range s.Scope {
			if row.PackageID != s.Package.ID || row.RetiredAt != nil || row.Inclusion != "included" {
				continue
			}
			for _, id := range row.InterfaceIDs {
				if id == p.InterfaceID {
					return true
				}
			}
		}
	}
	if len(p.Reason.Triggers) == 0 && p.TargetSystemID == "" && p.TargetPartID == "" {
		return true
	}
	triggers := map[string]bool{}
	for _, trigger := range p.Reason.Triggers {
		triggers[trigger.WorkItemID] = true
	}
	for _, w := range s.Works {
		if w.RetiredAt != nil || w.IsGroup || w.Inclusion != "included" || !workInPackage(w, s.Works, s.Scope, s.Package.ID, reportPackages(s)) {
			continue
		}
		if triggers[w.ID] || (p.TargetSystemID != "" && (w.SystemID == p.TargetSystemID || (cat != nil && cat.SystemWithin(w.SystemID, p.TargetSystemID))) && (p.TargetPartID == "" || p.TargetPartID == w.PartID)) || (len(triggers) == 0 && p.TargetSystemID == "" && p.TargetPartID == w.PartID) {
			return true
		}
	}
	return false
}

type Change struct {
	TargetID string `json:"target_id"`
	Kind     string `json:"kind"`
	Before   string `json:"before,omitempty"`
	After    string `json:"after,omitempty"`
}

// ChangesSinceIssue compares visible wording by stable block identity, not live
// source rows. It includes removals so deleted obligations cannot disappear.
func ChangesSinceIssue(before, after []Section) []Change {
	values := func(sections []Section) map[string]string {
		out := map[string]string{}
		for _, s := range sections {
			for _, b := range s.Blocks {
				text := fmt.Sprintf("%s: %s", b.Label, b.Text)
				if b.Table != nil {
					text += "\nSaved source comparison: " + strings.Join(b.Table.Columns, " | ")
					for _, row := range b.Table.Rows {
						text += "\n" + strings.Join(row, " | ")
					}
				}
				out[b.ID] = text
			}
		}
		return out
	}
	a, b := values(before), values(after)
	out := []Change{}
	for id, old := range a {
		now, ok := b[id]
		if !ok {
			out = append(out, Change{id, "removed", old, ""})
		} else if old != now {
			out = append(out, Change{id, "changed", old, now})
		}
	}
	for id, now := range b {
		if _, ok := a[id]; !ok {
			out = append(out, Change{id, "added", "", now})
		}
	}
	sort.Slice(out, func(i, j int) bool { return strings.Compare(out[i].TargetID, out[j].TargetID) < 0 })
	return out
}

func workTargetClausesReviewed(w works.Item, cat *knowledge.Catalog) bool {
	for _, ref := range w.Target.ClauseRefs {
		if cat == nil || !cat.ApprovedClause(ref.ID, ref.Version) {
			return false
		}
	}
	return true
}
func workBasis(w works.Item, cat *knowledge.Catalog) json.RawMessage {
	raw, _ := json.Marshal(w)
	var basis map[string]json.RawMessage
	_ = json.Unmarshal(raw, &basis)
	clauses := []knowledge.Clause{}
	if cat != nil {
		for _, ref := range w.Target.ClauseRefs {
			if clause, ok := cat.Clause(ref.ID, ref.Version); ok {
				clauses = append(clauses, clause)
			}
		}
	}
	basis["target_clauses"], _ = json.Marshal(clauses)
	basis["target_clauses_reviewed"], _ = json.Marshal(workTargetClausesReviewed(w, cat))
	raw, _ = json.Marshal(basis)
	return raw
}

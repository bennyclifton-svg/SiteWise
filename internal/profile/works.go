package profile

import (
	"encoding/json"
	"sitewise/internal/knowledge"
	"sitewise/internal/works"
	"sort"
	"strings"
)

// ProposedWorks converts defaults and included evidence into coarse proposals.
// Existing-building presence alone stays on the site. A part's stated work
// type overrides the whole project's type; explicit scope choices stay final.
func ProposedWorks(project string, rows []Row, parts []Part, cat *knowledge.Catalog) []works.Item {
	whole := wholePart(parts)
	work := workContextFor(rows, whole)
	candidates := map[[2]string]Row{}
	for _, r := range rows {
		if system, ok := strings.CutPrefix(r.Key, scopePrefix); ok && r.Value == scopeIn {
			candidates[[2]string{r.PartID, system}] = r
		}
	}
	for _, r := range rows {
		if r.Value != valIncluded || !appliedWorkRow(r) || work.existingPresence(r) || work.severalActions(r) {
			continue
		}
		if system, ok := strings.CutPrefix(r.Key, "sys."); ok && strings.HasSuffix(system, ".presence") {
			candidates[[2]string{r.PartID, strings.TrimSuffix(system, ".presence")}] = r
		}
	}
	out := make([]works.Item, 0, len(candidates))
	for key, r := range candidates {
		sys, ok := cat.System(key[1])
		if !ok || sys.Status == "deprecated" {
			continue
		}
		origin := OriginCalculation
		if len(r.Sources) > 0 {
			origin = OriginDocument
		}
		support := append([]Source{}, r.Sources...)
		action := work.action(key[0], key[1])
		rationale := "Scope default or included system evidence"
		if !works.ValidAction(action) {
			action = works.DefaultAction(cat, work.workType(key[0]), "")
		} else {
			support = append(support, work.actionRows[[2]string{key[0], "sys." + key[1]}].Sources...)
			rationale = "Scope with an applied action reading"
		}
		if len(support) > 0 {
			origin = OriginDocument
		}
		sources, _ := json.Marshal(support)
		out = append(out, works.Item{ID: works.CoarseID(project, key[0], key[1]), ProjectID: project, PartID: key[0], SystemID: key[1],
			Action: action, Inclusion: "included", Title: sys.Label,
			Origin: origin, ReviewStatus: ReviewProposed, Meaning: MeaningStated, CoarseKey: key[0] + "|" + key[1],
			Provenance: works.Provenance{Sources: sources, Band: r.Band, Rationale: rationale}})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].CoarseKey < out[j].CoarseKey })
	return out
}

// ProjectWorkScope keeps the old scope.<system> response shape, using only
// live work items. Derived proposals never overwrite a person's exclusion.
func ProjectWorkScope(rows []Row, items []works.Item) []Row {
	out := make([]Row, 0, len(rows)+len(items))
	presence := map[[2]string]Row{}
	for _, r := range rows {
		if strings.HasPrefix(r.Key, scopePrefix) {
			continue
		}
		out = append(out, r)
		presence[[2]string{r.PartID, r.Key}] = r
	}
	for _, item := range items {
		if item.RetiredAt != nil || item.IsGroup {
			continue
		}
		band := item.Provenance.Band
		if item.UserTouched || item.ReviewStatus != ReviewProposed {
			band = bandUser
		}
		if band == "" {
			band = bandSuggest
		}
		value := scopeIn
		if item.Inclusion == "excluded" {
			value = scopeOut
		}
		r := Row{PartID: item.PartID, Key: scopePrefix + item.SystemID, Value: value, Band: band, Scope: knowledge.ScopeProject,
			Origin: item.Origin, ReviewStatus: item.ReviewStatus, Meaning: item.Meaning, ValueState: StateSet}
		if item.UserTouched {
			r.UserVersion = item.Version
		}
		if action, ok := presence[[2]string{item.PartID, "sys." + item.SystemID + ".action"}]; ok && appliedWorkRow(action) && action.Value != item.Action && (item.UserTouched || item.ReviewStatus != ReviewProposed) {
			r.Note = cut("Your work action is "+item.Action+"; document action is "+action.Value+". Review the action evidence.", maxNote)
		}
		if p, ok := presence[[2]string{item.PartID, "sys." + item.SystemID + ".presence"}]; ok && value == scopeOut && p.Value == valIncluded && len(p.Sources) > 0 {
			r.Note = cut("A document says it is included: "+firstExcerpt(p), maxNote)
		}
		out = append(out, r)
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].PartID != out[j].PartID {
			return out[i].PartID < out[j].PartID
		}
		return out[i].Key < out[j].Key
	})
	return out
}

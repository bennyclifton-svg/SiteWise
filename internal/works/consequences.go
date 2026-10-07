package works

import (
	"fmt"
	"sort"
	"strings"

	"sitewise/internal/knowledge"
)

// ProposalPart contains effective, provenance-bearing inputs for one part.
// Its caller resolves project/site inheritance before evaluation.
type ProposalPart struct {
	Values       map[string]ProposalValue
	WorkTypes    []string
	Present      func(string) bool
	PresentState func(string) knowledge.Truth
	Signals      map[string]ProposalSignal
}

type ProposalInput struct {
	Items    []Item
	Parts    map[string]ProposalPart
	Existing ExistingAt
}

// EvaluateProposals evaluates all loaded ic/cq/uc records in code. It does not
// apply decisions, write records, or depend on packages or reports.
func EvaluateProposals(cat *knowledge.Catalog, in ProposalInput) ([]Proposal, error) {
	return evaluateProposals(cat, in, uncachedProposalFingerprint, nil)
}

func evaluateProposals(cat *knowledge.Catalog, in ProposalInput, fingerprintRecord proposalFingerprinter, predicateKeys map[string]string) ([]Proposal, error) {
	out, err := interfaceProposals(cat, in.Items, in.Existing, in.Parts, fingerprintRecord)
	if err != nil {
		return nil, err
	}
	byPart := map[string][]Item{}
	for _, item := range in.Items {
		if item.IsGroup || item.RetiredAt != nil || item.Inclusion != "included" {
			continue
		}
		if _, ok := in.Parts[item.PartID]; !ok {
			return nil, fmt.Errorf("missing proposal part inputs")
		}
		byPart[item.PartID] = append(byPart[item.PartID], item)
	}
	partIDs := []string{}
	for part := range byPart {
		partIDs = append(partIDs, part)
	}
	sort.Strings(partIDs)
	for _, partID := range partIDs {
		items := byPart[partID]
		sort.Slice(items, func(i, j int) bool { return items[i].ID < items[j].ID })
		part := in.Parts[partID]
		env := knowledge.WorksEnv{Values: map[string]string{}, WorkTypes: part.WorkTypes, Present: part.Present, PresentState: part.PresentState}
		for key, value := range part.Values {
			env.Values[key] = value.Value
		}
		for _, item := range items {
			env.Items = append(env.Items, knowledge.WorkItem{System: item.SystemID, Action: item.Action})
		}
		if in.Existing != nil {
			env.Existing = func(system string) knowledge.Truth { return in.Existing(partID, system) }
		}
		// Identical catalogue predicates read identical inputs within this part.
		// Keep traces local: another part or evaluation has different evidence.
		traces := map[string]knowledge.WorksTrace{}
		add := func(kind, id, edge, target, severity, status string, record, predicate any, proposed []knowledge.Proposal, signals, rules []string) error {
			key, compiled := predicateKeys[id]
			trace, found := traces[key]
			if !compiled || !found {
				trace = cat.TraceRelevantWorks(predicate, env)
				if compiled {
					traces[key] = trace
				}
			}
			if trace.Truth == knowledge.False {
				return nil
			}
			reason := ProposalReason{Record: id, Interface: edge, Rules: rules, Triggers: []ProposalTrigger{}, Determinants: []ProposalValue{}, Signals: []ProposalSignal{}}
			unaccepted := false
			for _, i := range trace.ItemIndexes {
				item := items[i]
				if item.ReviewStatus != "accepted_for_planning" && item.ReviewStatus != "verified" {
					unaccepted = true
				}
				reason.Triggers = append(reason.Triggers, ProposalTrigger{item.ID, item.Action, item.SystemID, item.PartID})
			}
			for _, key := range trace.Determinants {
				value := part.Values[key]
				value.Key = key
				if key == "work_type" {
					types := append([]string{}, part.WorkTypes...)
					sort.Strings(types)
					value.Value = strings.Join(types, ",")
					value.Origin = "applied_work_types"
				}
				reason.Determinants = append(reason.Determinants, value)
			}
			for system, state := range trace.Existing {
				reason.SystemStates = append(reason.SystemStates, ProposalSystemState{"system_existing", system, state.String()})
			}
			for system, state := range trace.Present {
				reason.SystemStates = append(reason.SystemStates, ProposalSystemState{"system_present", system, state.String()})
			}
			sort.Slice(reason.SystemStates, func(i, j int) bool {
				a, b := reason.SystemStates[i], reason.SystemStates[j]
				if a.Predicate != b.Predicate {
					return a.Predicate < b.Predicate
				}
				return a.System < b.System
			})
			for _, id := range signals {
				signal, ok := part.Signals[id]
				if !ok {
					signal = ProposalSignal{ID: id, State: "unknown"}
				}
				signal.ID = id
				reason.Signals = append(reason.Signals, signal)
			}
			fingerprint, err := fingerprintRecord(id, edge, record, reason)
			if err != nil {
				return err
			}
			for index, p := range proposed {
				key, err := ProposalKey(id, edge, target, partID, index)
				if err != nil {
					return err
				}
				out = append(out, Proposal{Key: key, RecordKind: kind, RecordID: id, InterfaceID: edge, ProposalIndex: index, TargetSystemID: target, TargetPartID: partID, Kind: p.Kind, Label: p.Label, Reason: reason, Severity: severity, Specificity: trace.Specificity, Draft: status != "reviewed", UnacceptedTriggers: unaccepted, InputsFingerprint: fingerprint, KnowledgeVersion: cat.Version(), State: "open"})
			}
			return nil
		}
		for _, record := range cat.Consequences() {
			if err := add("cq", record.ID, "", "", record.Severity, record.Status, record, record.When, record.Propose, record.Signals, record.GovernedBy); err != nil {
				return nil, err
			}
		}
		for _, record := range cat.UnforeseenConditions() {
			if record.DeRisk == nil {
				continue
			}
			if err := add("uc", record.ID, record.AttachesTo["interface"], record.AttachesTo["system"], record.Severity, record.Status, record, record.When, []knowledge.Proposal{*record.DeRisk}, record.Signals, nil); err != nil {
				return nil, err
			}
		}
	}
	RankProposals(out)
	return out, nil
}

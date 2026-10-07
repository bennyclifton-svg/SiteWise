package works

import (
	"fmt"
	"sort"
	"strings"

	"sitewise/internal/knowledge"
)

// ExistingAt reports existing-site evidence for a target in the affected part.
// The caller supplies applied site/part values; missing evidence stays unknown.
type ExistingAt func(part, system string) knowledge.Truth

// InterfaceProposals evaluates the loaded interface-consequence table. It is
// pure and is not wired into rebuilds until the full engine passes step-0
// measurements. Nonphysical rows never supply a trigger or existence evidence.
func InterfaceProposals(cat *knowledge.Catalog, items []Item, existing ExistingAt) ([]Proposal, error) {
	return interfaceProposals(cat, items, existing, nil, uncachedProposalFingerprint)
}

func interfaceProposals(cat *knowledge.Catalog, items []Item, existing ExistingAt, parts map[string]ProposalPart, fingerprint proposalFingerprinter) ([]Proposal, error) {
	if cat == nil {
		return nil, fmt.Errorf("catalogue required")
	}
	live := []Item{}
	byPart := map[string][]knowledge.WorkItem{}
	for _, item := range items {
		if item.RetiredAt != nil || item.IsGroup || item.Inclusion == "excluded" {
			continue
		}
		if item.ID == "" || item.PartID == "" || item.Inclusion != "included" {
			return nil, fmt.Errorf("invalid proposal work item")
		}
		if _, ok := cat.System(item.SystemID); !ok {
			return nil, fmt.Errorf("unknown work system")
		}
		if item.Action != "" && !ValidAction(item.Action) {
			return nil, fmt.Errorf("invalid work action")
		}
		live = append(live, item)
		byPart[item.PartID] = append(byPart[item.PartID], knowledge.WorkItem{System: item.SystemID, Action: item.Action})
	}
	sort.Slice(live, func(i, j int) bool { return live[i].ID < live[j].ID })
	out := map[string]Proposal{}
	records := map[string]any{}
	// Each target reads the same immutable part inputs throughout this call.
	// Repeated interface records and triggers can share its D-10 result; a new
	// evaluation starts empty so edits and other projects never reuse it.
	existence := map[[2]string]knowledge.Truth{}
	environments := map[string]knowledge.WorksEnv{}
	for partID, items := range byPart {
		part := parts[partID]
		env := knowledge.WorksEnv{Items: items, Values: map[string]string{}, WorkTypes: part.WorkTypes, Present: part.Present, PresentState: part.PresentState}
		for key, value := range part.Values {
			env.Values[key] = value.Value
		}
		if existing != nil {
			env.Existing = func(system string) knowledge.Truth { return existing(partID, system) }
		}
		environments[partID] = env
	}
	// Conditions depend on this evaluation's part evidence, never on another
	// part or an earlier call. Multiple consequences can share the same trace.
	traces := map[[2]string]knowledge.WorksTrace{}
	for _, record := range cat.InterfaceConsequences() {
		for _, edge := range cat.Interfaces {
			if edge.Type != record.Type {
				continue
			}
			type sides struct{ touch, other []string }
			directions := []sides{}
			if record.Touches == "from" || record.Touches == "either" {
				directions = append(directions, sides{edge.From, edge.To})
			}
			if record.Touches == "to" || record.Touches == "either" {
				directions = append(directions, sides{edge.To, edge.From})
			}
			for _, direction := range directions {
				for _, item := range live {
					if !record.AnyAct && item.Action != "" && !hasString(record.Actions, item.Action) {
						continue
					}
					matched, specificity := matchInterfaceSystem(cat, item.SystemID, direction.touch, item.Action != "")
					if !matched {
						continue
					}
					trace := knowledge.WorksTrace{Truth: knowledge.True}
					if edge.AppliesWhen != nil {
						traceKey := [2]string{edge.ID, item.PartID}
						var traced bool
						trace, traced = traces[traceKey]
						if !traced {
							trace = cat.TraceRelevantWorks(edge.AppliesWhen, environments[item.PartID])
							traces[traceKey] = trace
						}
					}
					if trace.Truth == knowledge.False {
						continue
					}
					for _, target := range direction.other {
						stateKey := [2]string{item.PartID, target}
						state, known := existence[stateKey]
						if !known {
							env := knowledge.WorksEnv{Items: byPart[item.PartID]}
							if existing != nil {
								env.Existing = func(system string) knowledge.Truth { return existing(item.PartID, system) }
							}
							state = cat.SystemExisting(target, env)
							existence[stateKey] = state
						}
						if state == knowledge.False {
							continue
						}
						key, err := ProposalKey(record.ID, edge.ID, target, item.PartID, 0)
						if err != nil {
							return nil, err
						}
						p, ok := out[key]
						if !ok {
							p = Proposal{Key: key, RecordKind: "ic", RecordID: record.ID, InterfaceID: edge.ID, TargetSystemID: target, TargetPartID: item.PartID, Kind: record.Propose.Kind, Label: record.Propose.Label, State: "open", Draft: record.Status != "reviewed" || edge.Status != "reviewed", KnowledgeVersion: cat.Version(), Reason: ProposalReason{Record: record.ID, Interface: edge.ID, OtherSideExistence: state.String(), Triggers: []ProposalTrigger{}, Determinants: []ProposalValue{}, Signals: []ProposalSignal{}}}
							addInterfaceInputs(&p.Reason, trace, parts[item.PartID])
							records[key] = struct {
								Record    knowledge.InterfaceConsequence
								Actions   []string
								Interface knowledge.Interface
							}{record, record.Actions, edge}
						}
						trigger := ProposalTrigger{item.ID, item.Action, item.SystemID, item.PartID}
						if item.ReviewStatus != "accepted_for_planning" && item.ReviewStatus != "verified" {
							p.UnacceptedTriggers = true
						}
						if !hasTrigger(p.Reason.Triggers, trigger) {
							p.Reason.Triggers = append(p.Reason.Triggers, trigger)
						}
						if specificity > p.Specificity {
							p.Specificity = specificity
						}
						out[key] = p
					}
				}
			}
		}
	}
	result := make([]Proposal, 0, len(out))
	for _, p := range out {
		// A proposal can have many triggers. Hash its complete input once,
		// instead of repeatedly hashing every growing prefix of that list.
		var err error
		p.InputsFingerprint, err = fingerprint(p.RecordID, p.InterfaceID, records[p.Key], p.Reason)
		if err != nil {
			return nil, err
		}
		result = append(result, p)
	}
	RankProposals(result)
	return result, nil
}

func addInterfaceInputs(reason *ProposalReason, trace knowledge.WorksTrace, part ProposalPart) {
	for _, key := range trace.Determinants {
		value := part.Values[key]
		value.Key = key
		if key == "work_type" {
			types := append([]string{}, part.WorkTypes...)
			sort.Strings(types)
			value.Value, value.Origin = strings.Join(types, ","), "applied_work_types"
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
}

func hasString(values []string, value string) bool {
	for _, v := range values {
		if v == value {
			return true
		}
	}
	return false
}

func hasTrigger(values []ProposalTrigger, value ProposalTrigger) bool {
	for _, v := range values {
		if v == value {
			return true
		}
	}
	return false
}

func coversSystem(cat *knowledge.Catalog, system, ancestor string) bool {
	for system != "" {
		if system == ancestor {
			return true
		}
		s, ok := cat.System(system)
		if !ok {
			return false
		}
		system = s.Parent
	}
	return false
}

func matchInterfaceSystem(cat *knowledge.Catalog, system string, endpoints []string, actionKnown bool) (bool, int) {
	matched, specificity := false, 0
	for _, endpoint := range endpoints {
		if !coversSystem(cat, system, endpoint) && !coversSystem(cat, endpoint, system) {
			continue
		}
		matched = true
		value := 2
		if !actionKnown {
			value = 0
		} else if system == endpoint && len(cat.Children(system)) == 0 {
			value = 3
		}
		if value > specificity {
			specificity = value
		}
	}
	return matched, specificity
}

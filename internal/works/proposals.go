package works

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"gopkg.in/yaml.v3"
)

type ProposalTrigger struct {
	WorkItemID string `json:"work_item_id"`
	Action     string `json:"action"`
	System     string `json:"system"`
	Part       string `json:"part"`
}

type ProposalValue struct {
	Key    string `json:"key"`
	Value  string `json:"value"`
	Origin string `json:"origin"`
}

type ProposalSignal struct {
	ID      string          `json:"id"`
	State   string          `json:"state"`
	Sources json.RawMessage `json:"sources,omitempty"`
}

type ProposalReason struct {
	Record             string                `json:"record"`
	Interface          string                `json:"interface,omitempty"`
	Rules              []string              `json:"rules,omitempty"`
	Triggers           []ProposalTrigger     `json:"triggers"`
	Determinants       []ProposalValue       `json:"determinants"`
	Signals            []ProposalSignal      `json:"signals"`
	OtherSideExistence string                `json:"other_side_existence,omitempty"`
	SystemStates       []ProposalSystemState `json:"system_states,omitempty"`
}

type ProposalSystemState struct {
	Predicate string `json:"predicate"`
	System    string `json:"system"`
	State     string `json:"state"`
}

type Proposal struct {
	Key                string         `json:"key"`
	RecordKind         string         `json:"record_kind"`
	RecordID           string         `json:"record_id"`
	InterfaceID        string         `json:"interface_id,omitempty"`
	ProposalIndex      int            `json:"proposal_index"`
	TargetSystemID     string         `json:"target_system_id,omitempty"`
	TargetPartID       string         `json:"target_part_id,omitempty"`
	Kind               string         `json:"kind"`
	Label              string         `json:"label"`
	Action             string         `json:"action,omitempty"`
	Reason             ProposalReason `json:"reason"`
	Severity           string         `json:"severity"`
	Specificity        int            `json:"specificity"`
	Rank               int            `json:"rank"`
	Critical           bool           `json:"critical"`
	Draft              bool           `json:"draft"`
	UnacceptedTriggers bool           `json:"unaccepted_triggers"`
	InputsFingerprint  string         `json:"inputs_fingerprint"`
	KnowledgeVersion   string         `json:"knowledge_version"`
	State              string         `json:"state"`
	InputsChanged      bool           `json:"inputs_changed"`
}

// ProposalKey is independent of database work-item IDs (D-19). Components
// cannot contain the delimiter; accepting ambiguous keys would bind a decision
// to the wrong target.
func ProposalKey(record, edge, system, part string, index int) (string, error) {
	if record == "" || index < 0 {
		return "", fmt.Errorf("proposal record and nonnegative index required")
	}
	for _, value := range []string{record, edge, system, part} {
		if strings.Contains(value, "|") {
			return "", fmt.Errorf("invalid proposal key component")
		}
	}
	return fmt.Sprintf("%s|%s|%s|%s|%d", record, edge, system, part, index), nil
}

// ProposalFingerprint hashes only inputs read by this proposal. Callers supply
// the record content (and interface content for ic records), not a global
// catalogue version. Unrelated catalogue edits must not reopen a dismissal.
// Semantic duplicates are collapsed so splitting an item preserves decisions.
func ProposalFingerprint(record any, reason ProposalReason) (string, error) {
	recordHash, err := proposalRecordHash(record)
	if err != nil {
		return "", err
	}
	return proposalFingerprintHash(recordHash, reason)
}

func proposalRecordHash(record any) (string, error) {
	// Loaded noul criteria use boolean YAML keys. YAML's sorted map encoding
	// preserves those keys; JSON cannot encode map[any]any criteria safely.
	recordBytes, err := yaml.Marshal(record)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%x", sha256.Sum256(recordBytes)), nil
}

func proposalFingerprintHash(recordHash string, reason ProposalReason) (string, error) {
	triggers := map[[3]string]bool{}
	for _, trigger := range reason.Triggers {
		triggers[[3]string{trigger.Action, trigger.System, trigger.Part}] = true
	}
	semantic := make([][3]string, 0, len(triggers))
	for trigger := range triggers {
		semantic = append(semantic, trigger)
	}
	sort.Slice(semantic, func(i, j int) bool {
		for n := 0; n < 3; n++ {
			if semantic[i][n] != semantic[j][n] {
				return semantic[i][n] < semantic[j][n]
			}
		}
		return false
	})
	values := append([]ProposalValue{}, reason.Determinants...)
	sort.Slice(values, func(i, j int) bool {
		if values[i].Key != values[j].Key {
			return values[i].Key < values[j].Key
		}
		if values[i].Origin != values[j].Origin {
			return values[i].Origin < values[j].Origin
		}
		return values[i].Value < values[j].Value
	})
	// Evidence source IDs remain in the reason for inspection, but the proposal
	// decision responds to the signal state, not a re-extraction's new row IDs.
	signals := make([][2]string, 0, len(reason.Signals))
	for _, signal := range reason.Signals {
		signals = append(signals, [2]string{signal.ID, signal.State})
	}
	sort.Slice(signals, func(i, j int) bool {
		if signals[i][0] != signals[j][0] {
			return signals[i][0] < signals[j][0]
		}
		return signals[i][1] < signals[j][1]
	})
	states := append([]ProposalSystemState{}, reason.SystemStates...)
	sort.Slice(states, func(i, j int) bool {
		if states[i].Predicate != states[j].Predicate {
			return states[i].Predicate < states[j].Predicate
		}
		if states[i].System != states[j].System {
			return states[i].System < states[j].System
		}
		return states[i].State < states[j].State
	})
	b, err := json.Marshal(struct {
		Record       string                `json:"record"`
		Triggers     [][3]string           `json:"triggers"`
		Existing     string                `json:"existing"`
		Values       []ProposalValue       `json:"values"`
		Signals      [][2]string           `json:"signals"`
		SystemStates []ProposalSystemState `json:"system_states"`
	}{recordHash, semantic, reason.OtherSideExistence, values, signals, states})
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%x", sha256.Sum256(b)), nil
}

type ProposalDecision struct {
	Decision          string `json:"decision"`
	InputsFingerprint string `json:"inputs_fingerprint"`
}

func evidenceProposalState(reason ProposalReason) string {
	for _, signal := range reason.Signals {
		if signal.State == "true" {
			return "addressed_by_evidence"
		}
	}
	return "open"
}

// ApplyProposalDecision never mutates authoritative decisions or creates work.
// Acceptance remains visible after changes; only dismissal reopens (plan 4.7).
func ApplyProposalDecision(p Proposal, decision *ProposalDecision) Proposal {
	p.State = evidenceProposalState(p.Reason)
	p.InputsChanged = false
	if decision == nil {
		return p
	}
	p.InputsChanged = decision.InputsFingerprint != p.InputsFingerprint
	switch decision.Decision {
	case "accepted":
		p.State = "accepted"
	case "dismissed":
		p.State = "dismissed"
		if p.InputsChanged {
			p.State = "reopened"
		}
	}
	return p
}

// RankProposals applies D-28 with a key tie-breaker for reproducible output.
func RankProposals(proposals []Proposal) {
	severity := map[string]int{"life-safety": 0, "compliance": 1, "durability": 2, "cost": 3, "programme": 4}
	order := func(value string) int {
		if n, ok := severity[value]; ok {
			return n
		}
		return 5
	}
	sort.Slice(proposals, func(i, j int) bool {
		a, b := proposals[i], proposals[j]
		if order(a.Severity) != order(b.Severity) {
			return order(a.Severity) < order(b.Severity)
		}
		if a.Specificity != b.Specificity {
			return a.Specificity > b.Specificity
		}
		return a.Key < b.Key
	})
	for i := range proposals {
		proposals[i].Rank = i + 1
	}
}

// VisibleProposals assumes ranked input. Critical proposals are additional to
// the normal top list, never hidden by its cap (D-34). Complete output remains
// available to the quality gate and show=all API.
func VisibleProposals(ranked []Proposal, count int) []Proposal {
	if count < 0 {
		count = 0
	}
	out := []Proposal{}
	for i, p := range ranked {
		if i < count || p.Critical || p.Severity == "life-safety" {
			out = append(out, p)
		}
	}
	return out
}

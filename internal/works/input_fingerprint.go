package works

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"sitewise/internal/knowledge"
	"sort"
	"sync"
)

func (e *Evaluator) prepareInputFingerprint() error {
	records := make([][3]string, 0, len(e.hashes))
	for key, hash := range e.hashes {
		records = append(records, [3]string{key[0], key[1], hash})
	}
	sort.Slice(records, func(i, j int) bool {
		if records[i][0] != records[j][0] {
			return records[i][0] < records[j][0]
		}
		return records[i][1] < records[j][1]
	})
	raw, err := json.Marshal(records)
	if err != nil {
		return err
	}
	e.inputVersion = fmt.Sprintf("%s:%x", e.cat.Version(), sha256.Sum256(raw))
	pending := e.cat.TopSystems()
	for len(pending) > 0 {
		system := pending[0]
		pending = pending[1:]
		e.systems = append(e.systems, system.ID)
		pending = append(pending, e.cat.Children(system.ID)...)
	}
	sort.Strings(e.systems)
	return nil
}

// InputFingerprint materialises every predicate-readable state, including
// source citations and trigger identities. It excludes only edit-version/audit
// clocks, which the evaluator never reads. No project state is cached here.
func (e *Evaluator) InputFingerprint(in ProposalInput) (string, error) {
	_, fingerprint, err := e.MaterializeInput(in)
	return fingerprint, err
}

// MaterializeInput shares the state already resolved for the fingerprint with
// evaluation. The returned callbacks only read immutable maps, so parallel
// evaluations do not race and nested presence scans run once per system/part.
func (e *Evaluator) MaterializeInput(in ProposalInput) (ProposalInput, string, error) {
	type partInput struct {
		Values    map[string]ProposalValue
		WorkTypes []string
		Signals   map[string]ProposalSignal
		Existing  map[string]knowledge.Truth
		Present   map[string]knowledge.Truth
	}
	parts := map[string]partInput{}
	resolved := in
	resolved.materialized = true
	// Preserve fallback behavior for hand-built predicates outside the loaded
	// system set without calling arbitrary external callbacks concurrently.
	var fallback sync.Mutex
	resolved.Parts = map[string]ProposalPart{}
	for id, p := range in.Parts {
		state := partInput{Values: p.Values, WorkTypes: p.WorkTypes, Signals: p.Signals, Existing: map[string]knowledge.Truth{}, Present: map[string]knowledge.Truth{}}
		for _, system := range e.systems {
			existing, present := knowledge.Unknown, knowledge.Unknown
			if in.Existing != nil {
				existing = in.Existing(id, system)
			}
			if p.PresentState != nil {
				present = p.PresentState(system)
			} else if p.Present != nil {
				present = knowledge.False
				if p.Present(system) {
					present = knowledge.True
				}
			}
			state.Existing[system], state.Present[system] = existing, present
		}
		parts[id] = state
		stable := p
		stable.PresentState = func(system string) knowledge.Truth {
			if v, ok := state.Present[system]; ok {
				return v
			}
			if p.PresentState != nil {
				fallback.Lock()
				defer fallback.Unlock()
				return p.PresentState(system)
			}
			if p.Present != nil {
				fallback.Lock()
				defer fallback.Unlock()
				if p.Present(system) {
					return knowledge.True
				}
				return knowledge.False
			}
			return knowledge.Unknown
		}
		resolved.Parts[id] = stable
	}
	resolved.Existing = func(part, system string) knowledge.Truth {
		if p, ok := parts[part]; ok {
			if state, ok := p.Existing[system]; ok {
				return state
			}
		}
		if in.Existing != nil {
			fallback.Lock()
			defer fallback.Unlock()
			return in.Existing(part, system)
		}
		return knowledge.Unknown
	}
	items := append([]Item{}, in.Items...)
	for i := range items {
		items[i].Version = 0
		items[i].Provenance.LastEditedAt = nil
	}
	sort.Slice(items, func(i, j int) bool { return items[i].ID < items[j].ID })
	raw, err := json.Marshal(struct {
		Version string
		Items   []Item
		Parts   map[string]partInput
	}{e.inputVersion, items, parts})
	if err != nil {
		return ProposalInput{}, "", err
	}
	return resolved, fmt.Sprintf("%x", sha256.Sum256(raw)), nil
}

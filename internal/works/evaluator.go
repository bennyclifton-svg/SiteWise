package works

import (
	"fmt"

	"sitewise/internal/knowledge"
)

type proposalFingerprinter func(recordID, interfaceID string, record any, reason ProposalReason) (string, error)

func uncachedProposalFingerprint(_, _ string, record any, reason ProposalReason) (string, error) {
	return ProposalFingerprint(record, reason)
}

// Evaluator precomputes immutable record hashes and shared predicate identities
// once per loaded catalogue.
// Construct a new evaluator when reloading knowledge; never mutate its catalogue.
// Project inputs and traces are not retained between calls, so edits remain
// immediately visible and evaluations are safe to run concurrently.
type Evaluator struct {
	cat        *knowledge.Catalog
	hashes     map[[2]string]string
	predicates map[string]string
}

func (e *Evaluator) KnowledgeVersion() string { return e.cat.Version() }

func NewEvaluator(cat *knowledge.Catalog) (*Evaluator, error) {
	if cat == nil {
		return nil, fmt.Errorf("catalogue required")
	}
	e := &Evaluator{cat: cat, hashes: map[[2]string]string{}, predicates: map[string]string{}}
	addPredicate := func(id string, predicate any) error {
		hash, err := proposalRecordHash(predicate)
		if err == nil {
			e.predicates[id] = hash
		}
		return err
	}
	add := func(id, edge string, record any) error {
		hash, err := proposalRecordHash(record)
		if err != nil {
			return err
		}
		key := [2]string{id, edge}
		if _, ok := e.hashes[key]; ok {
			return fmt.Errorf("duplicate proposal record hash key")
		}
		e.hashes[key] = hash
		return nil
	}
	for _, record := range cat.InterfaceConsequences() {
		for _, edge := range cat.Interfaces {
			if edge.Type != record.Type {
				continue
			}
			if err := add(record.ID, edge.ID, struct {
				Record    knowledge.InterfaceConsequence
				Actions   []string
				Interface knowledge.Interface
			}{record, record.Actions, edge}); err != nil {
				return nil, err
			}
		}
	}
	for _, record := range cat.Consequences() {
		if err := addPredicate(record.ID, record.When); err != nil {
			return nil, err
		}
		if err := add(record.ID, "", record); err != nil {
			return nil, err
		}
	}
	for _, record := range cat.UnforeseenConditions() {
		if err := addPredicate(record.ID, record.When); err != nil {
			return nil, err
		}
		if err := add(record.ID, record.AttachesTo["interface"], record); err != nil {
			return nil, err
		}
	}
	// Unique predicates have nothing to share. Avoid retaining their potentially
	// large traces during each evaluation merely to discard them unused.
	uses := map[string]int{}
	for _, key := range e.predicates {
		uses[key]++
	}
	for id, key := range e.predicates {
		if uses[key] < 2 {
			delete(e.predicates, id)
		}
	}
	return e, nil
}

func (e *Evaluator) Evaluate(in ProposalInput) ([]Proposal, error) {
	return evaluateProposals(e.cat, in, func(id, edge string, _ any, reason ProposalReason) (string, error) {
		hash, ok := e.hashes[[2]string{id, edge}]
		if !ok {
			return "", fmt.Errorf("proposal record not in evaluator catalogue")
		}
		return proposalFingerprintHash(hash, reason)
	}, e.predicates)
}

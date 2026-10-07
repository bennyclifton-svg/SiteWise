package works

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"
)

const DefaultProposalShowCount = 10

type ProposalPolicy struct {
	ShowCount int `json:"show_count"`
}

func LoadProposalPolicy(path string) (ProposalPolicy, error) {
	var policy ProposalPolicy
	raw, err := os.ReadFile(path)
	if err != nil {
		return policy, err
	}
	dec := json.NewDecoder(bytes.NewReader(raw))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&policy); err != nil {
		return policy, err
	}
	if err := dec.Decode(new(any)); err != io.EOF {
		return policy, fmt.Errorf("expected one proposal policy object")
	}
	if policy.ShowCount < 1 || policy.ShowCount > 1000 {
		return policy, fmt.Errorf("proposal show_count must be between 1 and 1000")
	}
	return policy, nil
}

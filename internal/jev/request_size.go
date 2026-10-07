package jev

import (
	"encoding/json"
	"fmt"
)

// RequestSize records a text-free preflight. These are estimates, not provider
// token counts: there is no provider tokenizer in this binary. The measured
// English fixtures use about 4.4 JSON bytes/input token; four bytes/token and
// a 4k/2k reserve reject oversized fan-outs conservatively without truncation.
// The API remains authoritative for tokenization and validation errors.
// https://docs.typesafe.ai/models: 64k total; 32k state + longest question.
// https://docs.typesafe.ai/api: at most 255 choice options.
type RequestSize struct {
	Questions               int `json:"questions"`
	Bytes                   int `json:"bytes"`
	StateBytes              int `json:"state_bytes"`
	LongestQuestionBytes    int `json:"longest_question_bytes"`
	EstimatedTokens         int `json:"estimated_tokens"`
	EstimatedLargestContext int `json:"estimated_largest_context"`
}

func MeasureRequest(call Call) (RequestSize, error) {
	s := RequestSize{Questions: len(call.Questions)}
	state, err := json.Marshal(call.State)
	if err != nil {
		return s, fmt.Errorf("%w: state encoding", ErrRequest)
	}
	s.StateBytes = len(state)
	questions, err := json.Marshal(call.Questions)
	if err != nil {
		return s, fmt.Errorf("%w: question encoding", ErrRequest)
	}
	s.Bytes = len(state) + len(questions) + 64
	for id, q := range call.Questions {
		raw, err := json.Marshal(q)
		if err != nil {
			return s, fmt.Errorf("%w: question encoding", ErrRequest)
		}
		if n := len(raw) + len(id) + 4; n > s.LongestQuestionBytes {
			s.LongestQuestionBytes = n
		}
	}
	s.EstimatedTokens = (s.Bytes + 3) / 4
	s.EstimatedLargestContext = (s.StateBytes + s.LongestQuestionBytes + 3) / 4
	return s, nil
}

// CheckRequestSize fails the entire state before admission/HTTP. Never clip
// questions, split into hidden calls, or replace unresolved evidence with a guess.
func CheckRequestSize(call Call) (RequestSize, error) {
	s, err := MeasureRequest(call)
	if err != nil {
		return s, err
	}
	if s.EstimatedTokens > 60000 || s.EstimatedLargestContext > 30000 {
		return s, fmt.Errorf("%w: %d questions, %d request bytes, estimated %d total / %d largest-context tokens; review passage scope", ErrRequestLimit, s.Questions, s.Bytes, s.EstimatedTokens, s.EstimatedLargestContext)
	}
	return s, nil
}
